package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/runner"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

func newSchedulerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scheduler",
		Aliases: []string{"sched", "s"},
		Short:   "Run the heartbeat scheduler and manage agent schedules",
		Long: `The scheduler drives all periodic agent activity.

Heartbeat: fires N minutes AFTER the previous run completes (interval-based).
  Only one run at a time per agent (no overlap).
  All tasks in one wakeup cycle share the same agent session.

Cron: fires at exact calendar times (crontab syntax).
  When a cron fires it enqueues a Task; the heartbeat loop picks it up.
  If no heartbeat is enabled, the scheduler executes the cron task directly.

Start the scheduler in the foreground:
  agencycli scheduler start`,
	}
	cmd.AddCommand(
		newSchedulerStartCmd(),
		newSchedulerHeartbeatCmd(),
		newSchedulerWakeupCmd(),
	)
	return cmd
}

// ── scheduler start ───────────────────────────────────────────────────────────

func newSchedulerStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the scheduler (blocks until SIGINT/SIGTERM)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}

			ts := taskstore.New(root)
			s := store.NewFS(root)

			type agentKey struct{ project, agent string }

			projects, err := ts.ListProjects()
			if err != nil {
				return err
			}

		// Collect agents with heartbeat enabled.
		var heartbeatAgents []agentKey
		// Collect agents with at least one enabled cron.
		var cronAgents []agentKey

		for _, p := range projects {
			agents, err := ts.ListAgents(p)
			if err != nil {
				continue
			}
			for _, a := range agents {
				hb, err := ts.GetHeartbeat(p, a)
				if err == nil && hb.Enabled {
					heartbeatAgents = append(heartbeatAgents, agentKey{p, a})
				}
				crons, err := ts.ListCrons(p, a)
				if err == nil {
					for _, c := range crons {
						if c.Enabled {
							cronAgents = append(cronAgents, agentKey{p, a})
							break
						}
					}
				}
			}
		}

		if len(heartbeatAgents) == 0 && len(cronAgents) == 0 {
			fmt.Println("No agents have heartbeat or cron enabled.")
			fmt.Println("  Heartbeat: agencycli scheduler heartbeat --project P --agent A --enable --interval 30m")
			fmt.Println("  Cron     : agencycli cron add --project P --agent A --schedule \"0 9 * * *\" --title T --prompt P")
			return nil
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		fmt.Printf("Scheduler started\n")
		if len(heartbeatAgents) > 0 {
			fmt.Printf("  Heartbeat agents (%d):\n", len(heartbeatAgents))
			for _, k := range heartbeatAgents {
				hb, _ := ts.GetHeartbeat(k.project, k.agent)
				fmt.Printf("    ● %s/%s  interval=%s\n", k.project, k.agent, hb.Interval)
			}
		}
		if len(cronAgents) > 0 {
			fmt.Printf("  Cron agents (%d):\n", len(cronAgents))
			for _, k := range cronAgents {
				crons, _ := ts.ListCrons(k.project, k.agent)
				for _, c := range crons {
					if c.Enabled {
						fmt.Printf("    ● %s/%s  [%s]  %s\n", k.project, k.agent, c.Schedule, c.Title)
					}
				}
			}
		}
		fmt.Println("Press Ctrl+C to stop.")

		var wg sync.WaitGroup

		// Deduplicate: if agent is in both lists, heartbeat loop handles cron too.
		heartbeatSet := map[agentKey]bool{}
		for _, k := range heartbeatAgents {
			heartbeatSet[k] = true
		}

		for _, k := range heartbeatAgents {
			k := k
			wg.Add(1)
			go func() {
				defer wg.Done()
				runHeartbeatLoop(ctx, root, k.project, k.agent, ts, s)
			}()
		}

		// Cron-only agents (no heartbeat): run cron loop that executes tasks directly.
		for _, k := range cronAgents {
			if heartbeatSet[k] {
				continue // already handled in heartbeat loop
			}
			k := k
			wg.Add(1)
			go func() {
				defer wg.Done()
				runCronOnlyLoop(ctx, root, k.project, k.agent, ts, s)
			}()
		}

		wg.Wait()
		fmt.Println("\nScheduler stopped.")
		return nil
		},
	}
}

// runHeartbeatLoop runs the blocking heartbeat loop for a single agent.
// It respects the non-overlapping constraint: the interval starts after
// each run completes, not at fixed wall-clock intervals.
func runHeartbeatLoop(ctx context.Context, root, project, agentName string,
	ts taskstore.Store, s store.Store) {

	log := func(format string, a ...any) {
		fmt.Printf("[heartbeat %s/%s] %s\n", project, agentName,
			fmt.Sprintf(format, a...))
	}

	firstCycle := true
	for {
		hb, err := ts.GetHeartbeat(project, agentName)
		if err != nil || !hb.Enabled {
			return
		}

		interval, err := time.ParseDuration(hb.Interval)
		if err != nil {
			log("invalid interval %q: %v", hb.Interval, err)
			return
		}

		// Determine how long to wait before the next wakeup.
		waitDur := interval
		if hb.LastWakeup != nil && hb.LastWakeupStatus != "running" {
			elapsed := time.Since(*hb.LastWakeup)
			if elapsed < interval {
				waitDur = interval - elapsed
			} else {
				waitDur = 0
			}
		} else if hb.LastWakeup == nil {
			waitDur = 0 // will get startup jitter below
		}

		// On the first cycle, always randomise the initial delay within [0, interval)
		// regardless of LastWakeup. This decouples agents from each other on every
		// scheduler restart, even when they share the same interval and LastWakeup time.
		if firstCycle {
			waitDur = time.Duration(rand.Float64() * float64(interval))
		}
		firstCycle = false

		if waitDur > 0 {
			nextAt := time.Now().Add(waitDur).Format("15:04:05")
			if hb.LastWakeup == nil {
				log("sleeping %s before first wakeup (startup jitter) — next at %s", waitDur.Round(time.Second), nextAt)
			} else {
				log("sleeping %s before next wakeup — next at %s", waitDur.Round(time.Second), nextAt)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(waitDur):
			}
		}

		// Re-check context after sleep.
		if ctx.Err() != nil {
			return
		}

		// Check active-hours / active-days window before waking up.
		if !isInActiveWindow(hb) {
			nextWake := nextWindowStart(hb)
			if nextWake > 0 {
				log("outside active window — sleeping %s until window opens", nextWake.Round(time.Minute))
				select {
				case <-ctx.Done():
					return
				case <-time.After(nextWake):
				}
				continue
			}
		}

		// Check overlap: if PID is set and process is still running, skip.
		if isAlreadyRunning(hb) {
			log("skipping wakeup — agent process still running (pid=%d)", hb.PID)
			time.Sleep(30 * time.Second)
			continue
		}

		// Evaluate wakeup condition (if configured).
		if hb.WakeupCondition != "" {
			met, output := checkWakeupCondition(
				hb.WakeupCondition,
				agentDir(root, project, agentName),
				root, project, agentName,
			)
			condTime := time.Now().UTC()
			hb.LastConditionAt = &condTime
			if met {
				hb.LastConditionStatus = "met"
				_ = ts.SaveHeartbeat(project, agentName, hb)
			} else {
				hb.LastConditionStatus = "not_met"
				_ = ts.SaveHeartbeat(project, agentName, hb)
				if output != "" {
					log("condition not met (%s) — skipping cycle, next check in %s",
						truncate(output, 80), interval.Round(time.Second))
				} else {
					log("condition not met — skipping cycle, next check in %s",
						interval.Round(time.Second))
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(interval):
				}
				continue
			}
		}

		// Mark as running.
		now := time.Now().UTC()
		hb.LastWakeup = &now
		hb.LastWakeupStatus = "running"
		hb.PID = os.Getpid()
		_ = ts.SaveHeartbeat(project, agentName, hb)

		log("waking up — checking crons and running pending tasks")

		// Fire any due cron jobs (enqueues tasks) before processing the queue.
		if n := fireDueCrons(ts, project, agentName); n > 0 {
			log("cron: enqueued %d task(s)", n)
		}

		if err := runAllPendingTasks(ctx, root, project, agentName, ts, s, hb); err != nil {
			log("wakeup cycle failed: %v", err)
			hb, _ = ts.GetHeartbeat(project, agentName)
			hb.LastWakeupStatus = "failed"
			hb.PID = 0
		} else {
			log("wakeup cycle done")
			hb, _ = ts.GetHeartbeat(project, agentName)
			hb.LastWakeupStatus = "done"
			hb.PID = 0
		}
		_ = ts.SaveHeartbeat(project, agentName, hb)
	}
}

// runAllPendingTasks processes all pending tasks in a single heartbeat cycle.
// Tasks within one cycle share the same agent session.
func runAllPendingTasks(ctx context.Context, root, project, agentName string,
	ts taskstore.Store, s store.Store, hb *entity.HeartbeatConfig) error {

	r := runner.New(root, ts, s)
	sessionID := hb.SessionID
	i18n := wakeupStrings(agencyLang(s))

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		task, err := nextPendingTask(ts, project, agentName)
		if err != nil {
			return err
		}
		if task == nil {
			// Queue is empty. Determine the wakeup prompt to run.
			// WakeupPrompt may be "@<file>", inline text, or empty (use built-in trigger).
			// The wakeup task is persisted to tasks.yaml so that the agent can
			// call `task confirm-request --id $TASK_ID` without hitting "not found".
			//
			// NOTE: When WakeupPrompt points to a file (e.g., "@.agencycli/context/wakeup.md"),
			// we do NOT read the file content here. The wakeup.md is already imported
			// via CLAUDE.md (@.agencycli/context/wakeup.md), so the agent has access to it.
			// We just send a short trigger to start the wakeup routine.
			var prompt string
			if hb.WakeupPrompt != "" {
				if strings.HasPrefix(hb.WakeupPrompt, "@") {
					// File reference: wakeup content is in CLAUDE.md via @import, use short trigger.
					prompt = i18n.DefaultTrigger
				} else {
					// Inline text: use as-is.
					prompt = hb.WakeupPrompt
				}
			} else {
				prompt = i18n.DefaultTrigger
			}
			if prompt != "" {
				// Prepend any unread messages to the wakeup prompt.
				recipient := project + "/" + agentName
				unread, _ := ts.ListUnreadMessages(recipient)
				if len(unread) > 0 {
					var msgSection strings.Builder
					msgSection.WriteString(i18n.InboxHeader)
					msgSection.WriteString(i18n.InboxIntro)
					for _, m := range unread {
						msgSection.WriteString(fmt.Sprintf("---\n**[%s] From: %s**",
							m.SentAt.Local().Format("01-02 15:04"), m.From))
						if m.Subject != "" {
							msgSection.WriteString(fmt.Sprintf("  Subject: %s", m.Subject))
						}
						msgSection.WriteString(fmt.Sprintf("\nID: `%s`\n\n%s\n\n", m.ID, m.Body))
					}
						msgSection.WriteString("---\n\n")
					msgSection.WriteString(i18n.InboxReplyHint)
					prompt = msgSection.String() + prompt
					fmt.Printf("[heartbeat %s/%s] ▶ wakeup routine (%d unread message(s))\n",
						project, agentName, len(unread))
				} else {
					fmt.Printf("[heartbeat %s/%s] ▶ wakeup routine\n", project, agentName)
				}

				now := time.Now().UTC()
				wakeupTask := &entity.Task{
					ID:        entity.NewTaskID(),
					Title:     "[wakeup] routine",
					Type:      "wakeup",
					Priority:  9,
					Status:    entity.TaskStatusPending,
					Prompt:    prompt,
					CreatedBy: "heartbeat:wakeup",
					CreatedAt: now,
					UpdatedAt: now,
				}
				// Persist before running so `task confirm-request --id $TASK_ID` works.
				if addErr := ts.AddTask(project, agentName, wakeupTask); addErr != nil {
					fmt.Printf("[heartbeat %s/%s] failed to persist wakeup task: %v\n", project, agentName, addErr)
				} else {
					wakeupTask.Status = entity.TaskStatusInProgress
					wakeupTask.StartedAt = &now
					wakeupTask.UpdatedAt = now
					_ = ts.UpdateTask(project, agentName, wakeupTask)
				}

				result, rErr := r.RunTask(project, agentName, wakeupTask, sessionID)
				if rErr == nil && result.SessionID != "" {
					sessionID = result.SessionID
					latestHB, _ := ts.GetHeartbeat(project, agentName)
					latestHB.SessionID = sessionID
					_ = ts.SaveHeartbeat(project, agentName, latestHB)
				}

				finished := time.Now().UTC()
				wakeupTask.FinishedAt = &finished
				wakeupTask.RunLogPath = ""
				if rErr != nil {
					wakeupTask.Status = entity.TaskStatusDoneFailed
					wakeupTask.LastError = rErr.Error()
					_ = ts.ArchiveTask(project, agentName, wakeupTask)
				} else {
					wakeupTask.Status = result.Status
					wakeupTask.RunLogPath = result.LogPath
					switch result.Status {
					case entity.TaskStatusAwaitingConfirmation:
						// Leave in tasks.yaml; confirm-request already added the inbox item.
						wakeupTask.ConfirmationReq = &entity.ConfirmationRequest{Summary: result.Summary}
						wakeupTask.UpdatedAt = time.Now().UTC()
						_ = ts.UpdateTask(project, agentName, wakeupTask)
					default:
						// done_success, done_failed, or anything unexpected → archive.
						_ = ts.ArchiveTask(project, agentName, wakeupTask)
						if len(unread) > 0 {
							_ = ts.MarkMessagesRead(recipient)
						}
					}
				}
			}
			break
		}

		fmt.Printf("[heartbeat %s/%s] ▶ task %s  %s\n", project, agentName, task.ID, task.Title)

		now := time.Now().UTC()
		task.Status = entity.TaskStatusInProgress
		task.StartedAt = &now
		task.UpdatedAt = now
		if err := ts.UpdateTask(project, agentName, task); err != nil {
			return err
		}

		result, err := r.RunTask(project, agentName, task, sessionID)
		if err != nil {
			task.Status = entity.TaskStatusDoneFailed
			task.LastError = err.Error()
			finished := time.Now().UTC()
			task.FinishedAt = &finished
			_ = ts.ArchiveTask(project, agentName, task)
			fmt.Printf("[heartbeat %s/%s] ✗ task %s failed: %v\n", project, agentName, task.ID, err)
			continue
		}

		// Update session ID for the cycle (per-cycle scope by default).
		if result.SessionID != "" {
			sessionID = result.SessionID
			latestHB, _ := ts.GetHeartbeat(project, agentName)
			latestHB.SessionID = sessionID
			if latestHB.SessionStartedAt == nil {
				t := time.Now().UTC()
				latestHB.SessionStartedAt = &t
			}
			_ = ts.SaveHeartbeat(project, agentName, latestHB)
		}

		finished := time.Now().UTC()
		task.FinishedAt = &finished
		task.Status = result.Status
		task.RunLogPath = result.LogPath

		switch result.Status {
		case entity.TaskStatusDoneSuccess:
			fmt.Printf("[heartbeat %s/%s] ✓ task %s done\n", project, agentName, task.ID)
			_ = ts.ArchiveTask(project, agentName, task)
			if len(task.OnSuccess) > 0 {
				_ = fireOnSuccessTriggers(root, project, agentName, task)
			}

		case entity.TaskStatusDoneFailed:
			task.LastError = result.ErrorMsg
			fmt.Printf("[heartbeat %s/%s] ✗ task %s failed: %s\n", project, agentName, task.ID, result.ErrorMsg)
			if task.RetryCount < task.MaxRetries {
				task.RetryCount++
				task.Status = entity.TaskStatusPending
				task.StartedAt = nil
				task.FinishedAt = nil
				_ = ts.UpdateTask(project, agentName, task)
			} else {
				_ = ts.ArchiveTask(project, agentName, task)
			}

		case entity.TaskStatusAwaitingConfirmation:
			task.ConfirmationReq = &entity.ConfirmationRequest{Summary: result.Summary}
			task.UpdatedAt = time.Now().UTC()
			_ = ts.UpdateTask(project, agentName, task)
			item := &entity.InboxItem{
				TaskID:  task.ID,
				Project: project,
				Agent:   agentName,
				Title:   task.Title,
				Summary: result.Summary,
				LogPath: task.RunLogPath,
			}
			_ = ts.AddToInbox(item)
			fmt.Printf("[heartbeat %s/%s] ? task %s awaiting confirmation\n", project, agentName, task.ID)
		}

		// Per-task session scope: reset sessionID so next task starts independently.
		if hb.SessionScope == entity.SessionScopeTask {
			sessionID = hb.SessionID
		}
	}
	return nil
}

// agentDir returns the filesystem path of an agent's workspace.
func agentDir(root, project, agentName string) string {
	return root + "/projects/" + project + "/agents/" + agentName
}

// ── i18n ─────────────────────────────────────────────────────────────────────

// wakeupI18n holds the auto-generated strings injected around the wakeup prompt.
type wakeupI18n struct {
	InboxHeader    string // section heading for unread-message block
	InboxIntro     string // sentence before the message list
	InboxReplyHint string // hint line showing how to reply
	DefaultTrigger string // used when wakeup_prompt is empty
}

// wakeupStrings returns the localised strings for the given lang code.
// Supported: "zh", anything else falls back to "en".
func wakeupStrings(lang string) wakeupI18n {
	switch lang {
	case "zh":
		return wakeupI18n{
			InboxHeader:    "## 📬 未读消息\n\n",
			InboxIntro:     "你收到了以下消息，请在本次唤醒中处理：\n\n",
			InboxReplyHint: "如需回复某条消息：\n  agencycli --dir $AGENCY_DIR inbox reply <msg-id> --body \"...\"\n\n",
			DefaultTrigger: "执行你的唤醒例程。检查待处理任务、未读消息及计划中的工作事项。如需了解具体例程，请参阅你的角色上下文。",
		}
	default: // "en"
		return wakeupI18n{
			InboxHeader:    "## 📬 Unread Messages\n\n",
			InboxIntro:     "You have the following unread messages. Please handle them in this wakeup cycle:\n\n",
			InboxReplyHint: "To reply to a message:\n  agencycli --dir $AGENCY_DIR inbox reply <msg-id> --body \"...\"\n\n",
			DefaultTrigger: "Execute your wakeup routine. Check pending tasks, unread messages, and your scheduled activities. Refer to your role context for the detailed routine.",
		}
	}
}

// agencyLang loads the agency config and returns its Lang field (default "en").
func agencyLang(s store.Store) string {
	if s == nil {
		return "en"
	}
	a, err := s.Agency()
	if err != nil || a.Lang == "" {
		return "en"
	}
	return a.Lang
}

// ── cron helpers ─────────────────────────────────────────────────────────────

var schedulerCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// fireDueCrons inspects all enabled crons for an agent, fires any that are due
// by enqueuing a new Task, and updates LastRun.  Returns the number enqueued.
func fireDueCrons(ts taskstore.Store, project, agentName string) int {
	crons, err := ts.ListCrons(project, agentName)
	if err != nil || len(crons) == 0 {
		return 0
	}
	now := time.Now()
	enqueued := 0
	changed := false
	for _, c := range crons {
		if !c.Enabled {
			continue
		}
		sched, err := schedulerCronParser.Parse(c.Schedule)
		if err != nil {
			continue
		}
		// The cron is due if the last scheduled time before now is after LastRun.
		// We look back 2 minutes to tolerate minor timing jitter.
		lookback := now.Add(-2 * time.Minute)
		lastExpected := prevCronTime(sched, now)
		if lastExpected.IsZero() || lastExpected.Before(lookback) {
			continue
		}
		if c.LastRun != nil && !c.LastRun.Before(lastExpected) {
			continue // already ran this slot
		}
		// Enqueue task.
		const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
		rb := make([]byte, 6)
		for i := range rb {
			rb[i] = chars[rand.Intn(len(chars))]
		}
		taskID := fmt.Sprintf("t-%s-%s", now.UTC().Format("20060102"), string(rb))
		task := &entity.Task{
			ID:        taskID,
			Title:     fmt.Sprintf("[cron] %s", c.Title),
			Status:    entity.TaskStatusPending,
			Type:      "cron",
			Priority:  5,
			Prompt:    c.Prompt,
			CreatedBy: "cron:" + c.ID,
			CreatedAt: now.UTC(),
			UpdatedAt: now.UTC(),
		}
		if err := ts.AddTask(project, agentName, task); err == nil {
			t := now
			c.LastRun = &t
			c.LastRunStatus = "enqueued"
			changed = true
			enqueued++
		}
	}
	if changed {
		_ = ts.SaveCrons(project, agentName, crons)
	}
	return enqueued
}

// prevCronTime returns the most recent scheduled time before or equal to `now`.
func prevCronTime(sched cron.Schedule, now time.Time) time.Time {
	// Binary search: find t such that Next(t) <= now < Next(t + epsilon).
	// We approximate by going back one full schedule cycle.
	// Simple approach: t = now - 1min, then compute Next and see.
	probe := now.Add(-2 * time.Minute)
	t := sched.Next(probe)
	if t.After(now) {
		return time.Time{}
	}
	return t
}

// runCronOnlyLoop is for agents that have crons but no heartbeat.
// It checks crons every minute, enqueues due tasks, and runs them immediately.
func runCronOnlyLoop(ctx context.Context, root, project, agentName string,
	ts taskstore.Store, s store.Store) {

	log := func(format string, a ...any) {
		fmt.Printf("[cron %s/%s] %s\n", project, agentName,
			fmt.Sprintf(format, a...))
	}

	// Align to the next minute boundary.
	now := time.Now()
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(nextMinute)):
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		n := fireDueCrons(ts, project, agentName)
		if n > 0 {
			log("fired %d cron(s) — running pending tasks", n)
			hb, _ := ts.GetHeartbeat(project, agentName)
			if err := runAllPendingTasks(ctx, root, project, agentName, ts, s, hb); err != nil {
				log("task execution error: %v", err)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// isAlreadyRunning checks whether the PID recorded in heartbeat is still alive.
func isAlreadyRunning(hb *entity.HeartbeatConfig) bool {
	if hb.PID <= 0 || hb.LastWakeupStatus != "running" {
		return false
	}
	proc, err := os.FindProcess(hb.PID)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; signal 0 checks liveness.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// ── active-window helpers ─────────────────────────────────────────────────────

// isInActiveWindow returns true if the current local time falls within the
// heartbeat's configured ActiveHours and ActiveDays restrictions.
// Both fields are optional; empty means "always allowed".
func isInActiveWindow(hb *entity.HeartbeatConfig) bool {
	now := time.Now()

	if hb.ActiveDays != "" && !isActiveDay(hb.ActiveDays, now) {
		return false
	}
	if hb.ActiveHours != "" {
		ok, _ := isActiveHour(hb.ActiveHours, now)
		return ok
	}
	return true
}

// nextWindowStart returns how long to sleep until the active window opens.
// Returns 0 if the window is currently open or cannot be determined.
func nextWindowStart(hb *entity.HeartbeatConfig) time.Duration {
	now := time.Now()

	// If active-hours is set, compute exact time until window start.
	if hb.ActiveHours != "" {
		_, next := isActiveHour(hb.ActiveHours, now)
		return next
	}
	// If only active-days: sleep until midnight then re-check.
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return time.Until(tomorrow)
}

// parseHHMM parses "HH:MM" into hour and minute.
func parseHHMM(s string) (int, int, error) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, 0, fmt.Errorf("invalid time %q (want HH:MM)", s)
	}
	return h, m, nil
}

// isActiveHour checks whether now is within the "HH:MM-HH:MM" range.
// Also returns duration until the window starts (0 if already inside).
func isActiveHour(activeHours string, now time.Time) (bool, time.Duration) {
	parts := strings.SplitN(activeHours, "-", 2)
	if len(parts) != 2 {
		return true, 0 // malformed — don't block
	}
	startH, startM, err1 := parseHHMM(strings.TrimSpace(parts[0]))
	endH, endM, err2 := parseHHMM(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return true, 0
	}

	loc := now.Location()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), startH, startM, 0, 0, loc)
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), endH, endM, 0, 0, loc)

	// Overnight range (e.g. 22:00-06:00): end wraps to next day.
	overnight := todayEnd.Before(todayStart) || todayEnd.Equal(todayStart)
	if overnight {
		todayEnd = todayEnd.Add(24 * time.Hour)
	}

	// Check whether now is inside [start, end).
	if now.Equal(todayStart) || (now.After(todayStart) && now.Before(todayEnd)) {
		return true, 0
	}

	// Compute time until window opens.
	nextOpen := todayStart
	if now.After(todayStart) {
		// Start already passed today; next open is tomorrow's start.
		nextOpen = todayStart.Add(24 * time.Hour)
	}
	return false, time.Until(nextOpen)
}

// isActiveDay checks whether now's weekday is allowed by the activeDays spec.
// Supported: comma-separated "Mon","Tue","Wed","Thu","Fri","Sat","Sun"
// or the aliases "weekdays" (Mon-Fri) and "weekends" (Sat-Sun).
func isActiveDay(activeDays string, now time.Time) bool {
	wd := now.Weekday()
	for _, token := range strings.Split(activeDays, ",") {
		t := strings.TrimSpace(strings.ToLower(token))
		switch t {
		case "weekdays":
			if wd >= time.Monday && wd <= time.Friday {
				return true
			}
		case "weekends":
			if wd == time.Saturday || wd == time.Sunday {
				return true
			}
		default:
			// Match abbreviated or full day names.
			day, err := parseDayName(t)
			if err == nil && wd == day {
				return true
			}
		}
	}
	return false
}

func parseDayName(s string) (time.Weekday, error) {
	switch strings.ToLower(s) {
	case "sun", "sunday":
		return time.Sunday, nil
	case "mon", "monday":
		return time.Monday, nil
	case "tue", "tuesday":
		return time.Tuesday, nil
	case "wed", "wednesday":
		return time.Wednesday, nil
	case "thu", "thursday":
		return time.Thursday, nil
	case "fri", "friday":
		return time.Friday, nil
	case "sat", "saturday":
		return time.Saturday, nil
	}
	return 0, fmt.Errorf("unknown day %q", s)
}

// ── wakeup condition ──────────────────────────────────────────────────────────

// checkWakeupCondition runs the condition shell command and returns whether
// the condition is met (exit 0 = met, non-zero = not met).
// output contains trimmed stdout+stderr (useful for logging on failure).
// The command runs with a 30-second timeout and inherits the host environment
// plus three extra variables: AGENCY_DIR, PROJECT, AGENT_NAME.
func checkWakeupCondition(condition, agentWorkDir, agencyDir, project, agentName string) (met bool, output string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", condition)
	cmd.Dir = agentWorkDir
	cmd.Env = append(os.Environ(),
		"AGENCY_DIR="+agencyDir,
		"PROJECT="+project,
		"AGENT_NAME="+agentName,
	)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	output = strings.TrimSpace(buf.String())
	return err == nil, output
}


// ── scheduler wakeup ──────────────────────────────────────────────────────────

func newSchedulerWakeupCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)

	cmd := &cobra.Command{
		Use:   "wakeup",
		Short: "Immediately trigger a full wakeup cycle for an agent",
		Long: `Wakeup immediately triggers an agent's full heartbeat cycle:

  1. Fire any due cron jobs (enqueue tasks)
  2. Run all pending tasks in priority order
  3. If the queue is empty and a wakeup_prompt is configured, execute it

Unlike 'agencycli run' (which runs one task), wakeup drains the entire
task queue and runs the wakeup routine — the same behaviour as the scheduler.

Active-window, interval, and wakeup_condition checks are bypassed.
If the agent is currently running (another cycle in progress), returns an error.

This command works whether or not the scheduler is running, making it
useful for testing and for agent-to-agent wakeup from inside a task.`,
		Example: `  # Immediately trigger a wakeup (for testing)
  agencycli scheduler wakeup --project my-api --agent pm

  # Agent-to-agent: wake up a peer from inside a running task
  agencycli --dir $AGENCY_DIR scheduler wakeup --project my-api --agent qa`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}

			ts := taskstore.New(root)
			s := store.NewFS(root)

			hb, err := ts.GetHeartbeat(project, agentName)
			if err != nil {
				return err
			}

			if isAlreadyRunning(hb) {
				return fmt.Errorf(
					"agent %s/%s is already running (pid %d) — wakeup skipped",
					project, agentName, hb.PID,
				)
			}

			// Mark running so the scheduler loop (if active) skips this cycle.
			now := time.Now().UTC()
			hb.LastWakeup = &now
			hb.LastWakeupStatus = "running"
			hb.PID = os.Getpid()
			_ = ts.SaveHeartbeat(project, agentName, hb)

			fmt.Printf("[wakeup %s/%s] triggered manually — running full cycle\n", project, agentName)

			if n := fireDueCrons(ts, project, agentName); n > 0 {
				fmt.Printf("[wakeup %s/%s] cron: enqueued %d task(s)\n", project, agentName, n)
			}

			cycleErr := runAllPendingTasks(context.Background(), root, project, agentName, ts, s, hb)

			hb, _ = ts.GetHeartbeat(project, agentName)
			hb.PID = 0
			if cycleErr != nil {
				hb.LastWakeupStatus = "failed"
				_ = ts.SaveHeartbeat(project, agentName, hb)
				return fmt.Errorf("[wakeup %s/%s] cycle failed: %w", project, agentName, cycleErr)
			}
			hb.LastWakeupStatus = "done"
			_ = ts.SaveHeartbeat(project, agentName, hb)

			fmt.Printf("[wakeup %s/%s] cycle complete\n", project, agentName)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("agent")
	return cmd
}

// ── scheduler heartbeat (configure) ──────────────────────────────────────────

func newSchedulerHeartbeatCmd() *cobra.Command {
	var (
		project          string
		agentName        string
		enable           bool
		disable          bool
		interval         string
		sessionScope     string
		activeHours      string
		activeDays       string
		wakeupPromptFile string
		wakeupCondition  string
	)

	cmd := &cobra.Command{
		Use:   "heartbeat",
		Short: "Configure heartbeat for an agent",
		Example: `  # Enable heartbeat with 30-minute interval
  agencycli scheduler heartbeat --project cc-connect --agent qa-reviewer \
    --enable --interval 30m

  # Only wake up between 09:00 and 18:00 on weekdays
  agencycli scheduler heartbeat --project cc-connect --agent dev \
    --enable --interval 1h --active-hours "09:00-18:00" --active-days "weekdays"

  # Night-shift agent: only wake up between 22:00 and 06:00
  agencycli scheduler heartbeat --project cc-connect --agent dev \
    --active-hours "22:00-06:00"

  # Clear active-hours restriction (run anytime)
  agencycli scheduler heartbeat --project cc-connect --agent dev \
    --active-hours ""

  # Disable
  agencycli scheduler heartbeat --project cc-connect --agent qa-reviewer --disable

		# Show current config
  agencycli scheduler heartbeat --project cc-connect --agent qa-reviewer

  # Set a wakeup routine (runs when queue is empty)
  agencycli scheduler heartbeat --project cc-connect --agent pm \
    --wakeup-prompt-file /root/code/TechStudio/projects/cc-connect/agents/pm/.agencycli-context/wakeup.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}

			ts := taskstore.New(root)
			hb, err := ts.GetHeartbeat(project, agentName)
			if err != nil {
				return err
			}

			changed := false
			if enable {
				hb.Enabled = true
				changed = true
			}
			if disable {
				hb.Enabled = false
				changed = true
			}
			if interval != "" {
				if _, err := time.ParseDuration(interval); err != nil {
					return fmt.Errorf("invalid interval %q: %w", interval, err)
				}
				hb.Interval = interval
				changed = true
			}
			if sessionScope != "" {
				hb.SessionScope = entity.SessionScope(sessionScope)
				changed = true
			}
			if hb.SessionScope == "" {
				hb.SessionScope = entity.SessionScopeCycle
			}
			if cmd.Flags().Changed("active-hours") {
				if activeHours != "" {
					// Validate format.
					parts := strings.SplitN(activeHours, "-", 2)
					if len(parts) != 2 {
						return fmt.Errorf("--active-hours must be HH:MM-HH:MM, got %q", activeHours)
					}
					if _, _, err := parseHHMM(strings.TrimSpace(parts[0])); err != nil {
						return err
					}
					if _, _, err := parseHHMM(strings.TrimSpace(parts[1])); err != nil {
						return err
					}
				}
				hb.ActiveHours = activeHours
				changed = true
			}
			if cmd.Flags().Changed("active-days") {
				// Validate tokens.
				if activeDays != "" {
					for _, tok := range strings.Split(activeDays, ",") {
						t := strings.TrimSpace(strings.ToLower(tok))
						if t == "weekdays" || t == "weekends" {
							continue
						}
						if _, err := parseDayName(t); err != nil {
							return fmt.Errorf("unknown day %q in --active-days", tok)
						}
					}
				}
				hb.ActiveDays = activeDays
				changed = true
			}
			if wakeupPromptFile != "" {
				// Verify the file exists and is readable.
				if _, err := os.ReadFile(wakeupPromptFile); err != nil {
					return fmt.Errorf("cannot read wakeup prompt file: %w", err)
				}
				hb.WakeupPrompt = "@" + wakeupPromptFile
				changed = true
			}
			if cmd.Flags().Changed("wakeup-condition") {
				hb.WakeupCondition = wakeupCondition
				changed = true
			}

			if changed {
				if err := ts.SaveHeartbeat(project, agentName, hb); err != nil {
					return err
				}
			}

			// Display current config.
			status := "disabled"
			if hb.Enabled {
				status = "enabled"
			}
			fmt.Printf("Heartbeat config — %s/%s\n", project, agentName)
			fmt.Printf("  Status  : %s\n", status)
			fmt.Printf("  Interval: %s\n", taskstore.FormatDuration(hb.Interval))
			fmt.Printf("  Session : %s\n", hb.SessionScope)
			if hb.ActiveHours != "" {
				fmt.Printf("  Active hours: %s\n", hb.ActiveHours)
			}
			if hb.ActiveDays != "" {
				fmt.Printf("  Active days : %s\n", hb.ActiveDays)
			}
			if hb.ActiveHours == "" && hb.ActiveDays == "" {
				fmt.Printf("  Active window: any time\n")
			}
			if !hb.Enabled {
				fmt.Printf("  (currently disabled — no wakeups scheduled)\n")
			} else if !isInActiveWindow(hb) {
				dur := nextWindowStart(hb)
				if dur > 0 {
					fmt.Printf("  ⏸  outside active window — next wakeup in %s\n", dur.Round(time.Minute))
				}
			}
			if hb.WakeupPrompt != "" {
				display := hb.WakeupPrompt
				if len(display) > 60 {
					display = display[:57] + "..."
				}
				fmt.Printf("  Wakeup  : %s\n", display)
			}
			if hb.WakeupCondition != "" {
				fmt.Printf("  Condition: %s\n", hb.WakeupCondition)
				if hb.LastConditionStatus != "" && hb.LastConditionAt != nil {
					symbol := "✓"
					if hb.LastConditionStatus == "not_met" {
						symbol = "✗"
					}
					fmt.Printf("  Last check: %s %s (%s)\n",
						symbol, hb.LastConditionStatus,
						hb.LastConditionAt.Local().Format("01-02 15:04:05"))
				}
			}
			if hb.LastWakeup != nil {
				fmt.Printf("  Last    : %s  (%s)\n",
					hb.LastWakeup.Format(time.RFC3339), hb.LastWakeupStatus)
			}
			if hb.SessionID != "" {
				fmt.Printf("  Session ID: %s\n", hb.SessionID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	cmd.Flags().BoolVar(&enable, "enable", false, "enable heartbeat")
	cmd.Flags().BoolVar(&disable, "disable", false, "disable heartbeat")
	cmd.Flags().StringVar(&interval, "interval", "", "heartbeat interval (e.g. 30m, 1h)")
	cmd.Flags().StringVar(&sessionScope, "session-scope", "", "session scope: cycle (default) or task")
	cmd.Flags().StringVar(&activeHours, "active-hours", "", `restrict wakeups to a time window, e.g. "09:00-18:00" or "22:00-06:00"`)
	cmd.Flags().StringVar(&activeDays, "active-days", "", `restrict wakeups to specific days, e.g. "weekdays", "Mon,Wed,Fri", "Sat,Sun"`)
	cmd.Flags().StringVar(&wakeupPromptFile, "wakeup-prompt-file", "", "path to a markdown file used as the default wakeup routine when queue is empty")
	cmd.Flags().StringVar(&wakeupCondition, "wakeup-condition", "", `shell command evaluated before each wakeup; exit 0 = proceed, non-zero = skip cycle (e.g. "gh issue list --state open | grep -q .")`)
	return cmd
}
