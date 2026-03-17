package main

import (
	"fmt"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/runner"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var (
		project   string
		agentName string
		taskID    string
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute a task for an agent (manual trigger)",
		Long: `Run executes the next pending task (or a specific task) for an agent.

The agent's configured CLI is invoked inside the agent's working directory.
Task state is updated automatically based on the exit code and output.

This is a one-shot manual trigger. For recurring automated runs, use
'agencycli daemon start' to activate the heartbeat scheduler.`,
		Example: `  # Run the next pending task
  agencycli run --project cc-connect --agent qa-reviewer

  # Run a specific task
  agencycli run --project cc-connect --agent qa-reviewer --task t-20260316-abc123

  # Dry run: print what would be executed
  agencycli run --project cc-connect --agent qa-reviewer --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}

			ts := taskstore.New(root)
			s := store.NewFS(root)

			var task *entity.Task
			if taskID != "" {
				task, err = ts.GetTask(project, agentName, taskID)
				if err != nil {
					return err
				}
			} else {
				// Pick next pending task ordered by priority then created_at.
				task, err = nextPendingTask(ts, project, agentName)
				if err != nil {
					return err
				}
				if task == nil {
					fmt.Println("No pending tasks.")
					return nil
				}
			}

			if dryRun {
				fmt.Printf("Dry run — would execute:\n")
				fmt.Printf("  Project : %s\n", project)
				fmt.Printf("  Agent   : %s\n", agentName)
				fmt.Printf("  Task    : %s  %s\n", task.ID, task.Title)
				fmt.Printf("  Status  : %s\n", task.Status)
				return nil
			}

			hb, err := ts.GetHeartbeat(project, agentName)
			if err != nil {
				return err
			}

			fmt.Printf("▶ Running task %s  [%s]\n", task.ID, task.Title)

			// Transition to in_progress.
			now := time.Now().UTC()
			task.Status = entity.TaskStatusInProgress
			task.StartedAt = &now
			task.UpdatedAt = now
			if err := ts.UpdateTask(project, agentName, task); err != nil {
				return err
			}

			r := runner.New(root, ts, s)
			result, err := r.RunTask(project, agentName, task, hb.SessionID)
			if err != nil {
				// Execution error (not the same as task failure).
				task.Status = entity.TaskStatusDoneFailed
				task.LastError = err.Error()
				finished := time.Now().UTC()
				task.FinishedAt = &finished
				_ = ts.ArchiveTask(project, agentName, task)
				return fmt.Errorf("execution error: %w", err)
			}

			// Persist new session ID if captured.
			if result.SessionID != "" && result.SessionID != hb.SessionID {
				hb.SessionID = result.SessionID
				now := time.Now().UTC()
				hb.SessionStartedAt = &now
				_ = ts.SaveHeartbeat(project, agentName, hb)
			}

			task.RunLogPath = result.LogPath
			finished := time.Now().UTC()
			task.FinishedAt = &finished

			// If the agent used `task confirm-request` internally it already
			// set the task to awaiting_confirmation in the store; don't
			// overwrite that with the runner's default done_success.
			if result.Status == entity.TaskStatusDoneSuccess {
				if fresh, ferr := ts.GetTask(project, agentName, task.ID); ferr == nil &&
					fresh.Status == entity.TaskStatusAwaitingConfirmation {
					// Agent routed to inbox via the CLI; honour that status and stop here.
					task.Status = entity.TaskStatusAwaitingConfirmation
					task.UpdatedAt = time.Now().UTC()
					_ = ts.UpdateTask(project, agentName, task)
					fmt.Printf("⏳ Task %s awaiting human confirmation (routed to inbox)\n", task.ID)
					return nil
				}
			}
			task.Status = result.Status

			switch result.Status {
			case entity.TaskStatusDoneSuccess:
				fmt.Printf("✓ Task %s completed successfully\n", task.ID)
				if task.RunLogPath != "" {
					fmt.Printf("  Log: %s\n", task.RunLogPath)
				}
				if err := ts.ArchiveTask(project, agentName, task); err != nil {
					return err
				}
				// Fire triggers.
				if len(task.OnSuccess) > 0 {
					if err := fireOnSuccessTriggers(root, project, agentName, task); err != nil {
						fmt.Printf("  warning: trigger errors: %v\n", err)
					}
				}

			case entity.TaskStatusDoneFailed:
				task.LastError = result.ErrorMsg
				fmt.Printf("✗ Task %s failed: %s\n", task.ID, result.ErrorMsg)
				if task.RunLogPath != "" {
					fmt.Printf("  Log: %s\n", task.RunLogPath)
				}
				if task.RetryCount < task.MaxRetries {
					task.RetryCount++
					task.Status = entity.TaskStatusPending
					task.StartedAt = nil
					task.FinishedAt = nil
					if err := ts.UpdateTask(project, agentName, task); err != nil {
						return err
					}
					fmt.Printf("  Auto-retrying (%d/%d)...\n", task.RetryCount, task.MaxRetries)
				} else {
					if err := ts.ArchiveTask(project, agentName, task); err != nil {
						return err
					}
				}

			case entity.TaskStatusAwaitingConfirmation:
				task.ConfirmationReq = &entity.ConfirmationRequest{Summary: result.Summary}
				task.UpdatedAt = time.Now().UTC()
				if err := ts.UpdateTask(project, agentName, task); err != nil {
					return err
				}
				item := &entity.InboxItem{
					TaskID:  task.ID,
					Project: project,
					Agent:   agentName,
					Title:   task.Title,
					Summary: result.Summary,
					LogPath: task.RunLogPath,
				}
				if err := ts.AddToInbox(item); err != nil {
					return err
				}
				fmt.Printf("? Task %s is awaiting your confirmation\n", task.ID)
				fmt.Printf("  Summary: %s\n", result.Summary)
				fmt.Printf("  Run: agencycli inbox confirm %s\n", task.ID)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	cmd.Flags().StringVar(&taskID, "task", "", "specific task ID to run (default: next pending)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be executed without running")
	return cmd
}

// nextPendingTask returns the highest-priority pending task (lowest priority
// number wins; ties broken by created_at).
func nextPendingTask(ts taskstore.Store, project, agentName string) (*entity.Task, error) {
	tasks, err := ts.ListTasks(project, agentName, entity.TaskStatusPending)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	best := tasks[0]
	for _, t := range tasks[1:] {
		if t.Priority < best.Priority ||
			(t.Priority == best.Priority && t.CreatedAt.Before(best.CreatedAt)) {
			best = t
		}
	}
	return best, nil
}
