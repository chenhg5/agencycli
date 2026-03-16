package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/runner"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
)

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the heartbeat scheduler daemon",
		Long: `The daemon runs a heartbeat loop for each agent that has heartbeat enabled.

Heartbeat vs Cron:
  - Heartbeat: blocking loop; fires N minutes AFTER the previous run completes.
    Only one instance runs at a time per agent (no overlap).
    All tasks in one wakeup share the same agent session.
  - Cron (Phase 2): fires at exact calendar times; enqueues a task for the next
    heartbeat cycle to pick up.

Start the daemon in the foreground:
  agencycli daemon start

It will scan all projects/agents, and for each agent with heartbeat.enabled=true,
start a goroutine that repeatedly: waits → wakes → runs all pending tasks → waits.`,
	}
	cmd.AddCommand(
		newDaemonStartCmd(),
		newDaemonHeartbeatCmd(),
	)
	return cmd
}

// ── daemon start ──────────────────────────────────────────────────────────────

func newDaemonStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the heartbeat daemon (blocks until SIGINT/SIGTERM)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}

			ts := taskstore.New(root)
			s := store.NewFS(root)

			type agentKey struct{ project, agent string }
			var enabled []agentKey

			projects, err := ts.ListProjects()
			if err != nil {
				return err
			}
			for _, p := range projects {
				agents, err := ts.ListAgents(p)
				if err != nil {
					continue
				}
				for _, a := range agents {
					hb, err := ts.GetHeartbeat(p, a)
					if err != nil || !hb.Enabled {
						continue
					}
					enabled = append(enabled, agentKey{p, a})
				}
			}

			if len(enabled) == 0 {
				fmt.Println("No agents have heartbeat enabled.")
				fmt.Println("Enable with: agencycli daemon heartbeat --project P --agent A --enable --interval 30m")
				return nil
			}

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			fmt.Printf("Daemon started — watching %d agent(s)\n", len(enabled))
			for _, k := range enabled {
				hb, _ := ts.GetHeartbeat(k.project, k.agent)
				fmt.Printf("  ● %s/%s  interval=%s\n", k.project, k.agent, hb.Interval)
			}
			fmt.Println("Press Ctrl+C to stop.")

			var wg sync.WaitGroup
			for _, k := range enabled {
				k := k
				wg.Add(1)
				go func() {
					defer wg.Done()
					runHeartbeatLoop(ctx, root, k.project, k.agent, ts, s)
				}()
			}

			wg.Wait()
			fmt.Println("\nDaemon stopped.")
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
		}

		if waitDur > 0 {
			log("sleeping %s before next wakeup", waitDur.Round(time.Second))
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

		// Check overlap: if PID is set and process is still running, skip.
		if isAlreadyRunning(hb) {
			log("skipping wakeup — agent process still running (pid=%d)", hb.PID)
			time.Sleep(30 * time.Second)
			continue
		}

		// Mark as running.
		now := time.Now().UTC()
		hb.LastWakeup = &now
		hb.LastWakeupStatus = "running"
		hb.PID = os.Getpid()
		_ = ts.SaveHeartbeat(project, agentName, hb)

		log("waking up — running pending tasks")

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

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		task, err := nextPendingTask(ts, project, agentName)
		if err != nil {
			return err
		}
		if task == nil {
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
				TaskID:   task.ID,
				Project:  project,
				Agent:    agentName,
				Title:    task.Title,
				Summary:  result.Summary,
				RoutedAt: time.Now().UTC(),
				LogPath:  task.RunLogPath,
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

// ── daemon heartbeat (configure) ─────────────────────────────────────────────

func newDaemonHeartbeatCmd() *cobra.Command {
	var (
		project      string
		agentName    string
		enable       bool
		disable      bool
		interval     string
		sessionScope string
	)

	cmd := &cobra.Command{
		Use:   "heartbeat",
		Short: "Configure heartbeat for an agent",
		Example: `  # Enable heartbeat with 30-minute interval
  agencycli daemon heartbeat --project cc-connect --agent qa-reviewer \
    --enable --interval 30m

  # Disable
  agencycli daemon heartbeat --project cc-connect --agent qa-reviewer --disable

  # Show current config
  agencycli daemon heartbeat --project cc-connect --agent qa-reviewer`,
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
	return cmd
}
