package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/chenhg5/agencycli/internal/workflow"
	"github.com/spf13/cobra"
)

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage agent tasks",
	}
	cmd.AddCommand(
		newTaskAddCmd(),
		newTaskListCmd(),
		newTaskShowCmd(),
		newTaskDoneCmd(),
		newTaskConfirmRequestCmd(),
		newTaskRetryCmd(),
		newTaskCancelCmd(),
	)
	return cmd
}

// ── task add ──────────────────────────────────────────────────────────────────

func newTaskAddCmd() *cobra.Command {
	var (
		project    string
		agentName  string
		title      string
		taskType   string
		priority   int
		prompt     string
		promptFile string
		dependsOn  []string
		assignee   string
		createdBy  string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new task for an agent",
		Example: `  # PM adds a bug task for dev-claude
  agencycli task add \
    --project cc-connect --agent dev-claude \
    --title "Fix login redirect on mobile" \
    --type bug --priority 1 \
    --prompt "The login redirect is broken on mobile. Fix it and open a PR."

  # Assign to human inbox
  agencycli task add \
    --project cc-connect --agent pm-agent \
    --title "Scope AI search feature" --assignee human \
    --prompt "Issue #101 requests AI search. Is this in scope for Q2?"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" || title == "" {
				return fmt.Errorf("--project, --agent, and --title are required")
			}

			var promptText string
			if promptFile != "" {
				data, err := os.ReadFile(promptFile)
				if err != nil {
					return fmt.Errorf("read prompt file: %w", err)
				}
				promptText = string(data)
			} else {
				promptText = prompt
			}
			if promptText == "" {
				return fmt.Errorf("--prompt or --prompt-file is required")
			}

			if assignee == "" {
				assignee = project + "/" + agentName
			}

			now := time.Now().UTC()
			t := &entity.Task{
				ID:        entity.NewTaskID(),
				Title:     title,
				Type:      entity.TaskType(taskType),
				Priority:  priority,
				Assignee:  assignee,
				CreatedBy: createdBy,
				Status:    entity.TaskStatusPending,
				Prompt:    promptText,
				DependsOn: dependsOn,
				CreatedAt: now,
				UpdatedAt: now,
			}

			ts := taskstore.New(root)

			// If assignee is "human", create the task under the source agent
			// but also route it directly to the inbox.
			if assignee == "human" {
				if err := ts.AddTask(project, agentName, t); err != nil {
					return err
				}
				now := time.Now().UTC()
				item := &entity.InboxItem{
					TaskID:   t.ID,
					Project:  project,
					Agent:    agentName,
					Title:    t.Title,
					Summary:  promptText,
					RoutedAt: now,
				}
				if err := ts.AddToInbox(item); err != nil {
					return err
				}
				fmt.Printf("✓ Task %s created and routed to human inbox\n", t.ID)
				return nil
			}

			if err := ts.AddTask(project, agentName, t); err != nil {
				return err
			}
			fmt.Printf("✓ Task %s created  [%s / %s]\n", t.ID, project, agentName)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	cmd.Flags().StringVar(&title, "title", "", "task title")
	cmd.Flags().StringVar(&taskType, "type", "chore", "task type (feature|bug|review|triage|test|research|chore)")
	cmd.Flags().IntVar(&priority, "priority", 2, "priority: 0=critical 1=high 2=normal 3=low")
	cmd.Flags().StringVar(&prompt, "prompt", "", "task prompt text")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "read task prompt from file")
	cmd.Flags().StringArrayVar(&dependsOn, "depends-on", nil, "task IDs this task depends on")
	cmd.Flags().StringVar(&assignee, "assignee", "", "assignee override (default: <project>/<agent>, or 'human')")
	cmd.Flags().StringVar(&createdBy, "created-by", "human", "who created this task")
	return cmd
}

// ── task list ─────────────────────────────────────────────────────────────────

func newTaskListCmd() *cobra.Command {
	var (
		project   string
		agentName string
		status    string
		archived  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks for an agent",
		Example: `  agencycli task list --project cc-connect --agent qa-reviewer
  agencycli task list --project cc-connect --agent qa-reviewer --status pending
  agencycli task list --project cc-connect --agent qa-reviewer --archived`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}

			ts := taskstore.New(root)
			var tasks []*entity.Task

			if archived {
				tasks, err = ts.ListArchivedTasks(project, agentName)
			} else if status != "" {
				tasks, err = ts.ListTasks(project, agentName, entity.TaskStatus(status))
			} else {
				tasks, err = ts.ListTasks(project, agentName)
			}
			if err != nil {
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "STATUS\tID\tPRI\tTITLE")
			fmt.Fprintln(w, "──────\t──\t───\t─────")
			for _, t := range tasks {
				fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\n",
					taskstore.StatusIcon(t.Status), t.Status,
					t.ID,
					taskstore.PriorityLabel(t.Priority),
					t.Title,
				)
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	cmd.Flags().StringVar(&status, "status", "", "filter by status")
	cmd.Flags().BoolVar(&archived, "archived", false, "show archived (terminal) tasks")
	return cmd
}

// ── task show ─────────────────────────────────────────────────────────────────

func newTaskShowCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)

	cmd := &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show full detail of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}

			ts := taskstore.New(root)
			t, err := ts.GetTask(project, agentName, args[0])
			if err != nil {
				return err
			}

			fmt.Printf("ID       : %s\n", t.ID)
			fmt.Printf("Title    : %s\n", t.Title)
			fmt.Printf("Status   : %s %s\n", taskstore.StatusIcon(t.Status), t.Status)
			fmt.Printf("Type     : %s\n", t.Type)
			fmt.Printf("Priority : %s (%d)\n", taskstore.PriorityLabel(t.Priority), t.Priority)
			fmt.Printf("Assignee : %s\n", t.Assignee)
			fmt.Printf("CreatedBy: %s\n", t.CreatedBy)
			fmt.Printf("Created  : %s\n", t.CreatedAt.Format(time.RFC3339))
			if t.StartedAt != nil {
				fmt.Printf("Started  : %s\n", t.StartedAt.Format(time.RFC3339))
			}
			if t.FinishedAt != nil {
				fmt.Printf("Finished : %s\n", t.FinishedAt.Format(time.RFC3339))
			}
			if len(t.DependsOn) > 0 {
				fmt.Printf("Depends  : %s\n", strings.Join(t.DependsOn, ", "))
			}
			if t.RetryCount > 0 {
				fmt.Printf("Retries  : %d / %d\n", t.RetryCount, t.MaxRetries)
			}
			if t.LastError != "" {
				fmt.Printf("Error    : %s\n", t.LastError)
			}
			if t.RunLogPath != "" {
				fmt.Printf("Log      : %s\n", t.RunLogPath)
			}
			if t.ConfirmationReq != nil {
				fmt.Printf("Confirm  : %s\n", t.ConfirmationReq.Summary)
			}
			fmt.Printf("\n── Prompt ──────────────────────────────────────────\n")
			fmt.Println(t.Prompt)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	return cmd
}

// ── task done ─────────────────────────────────────────────────────────────────

func newTaskDoneCmd() *cobra.Command {
	var (
		taskID   string
		status   string
		errorMsg string
		summary  string
	)

	cmd := &cobra.Command{
		Use:   "done",
		Short: "Mark a task as done (success or failed)",
		Long: `Intended to be called BY the agent itself from inside its prompt:

  agencycli task done --id <task-id> --status success
  agencycli task done --id <task-id> --status success --summary "PR #42 opened at https://..."
  agencycli task done --id <task-id> --status failed --error "reason"

The --summary value is passed to the next step via workflow routing ({{task.summary}}).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if taskID == "" {
				return fmt.Errorf("--id is required")
			}

			var finalStatus entity.TaskStatus
			switch status {
			case "success", "done_success":
				finalStatus = entity.TaskStatusDoneSuccess
			case "failed", "done_failed":
				finalStatus = entity.TaskStatusDoneFailed
			default:
				return fmt.Errorf("--status must be 'success' or 'failed'")
			}

			project, agentName, err := resolveTaskOwner(root, taskID)
			if err != nil {
				return err
			}

			ts := taskstore.New(root)
			s := store.NewFS(root)
			t, err := ts.GetTask(project, agentName, taskID)
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			t.Status = finalStatus
			t.FinishedAt = &now
			t.UpdatedAt = now
			if summary != "" {
				t.Summary = summary
			}
			if errorMsg != "" {
				t.LastError = errorMsg
			}

			if err := ts.ArchiveTask(project, agentName, t); err != nil {
				return err
			}

			// Fire on_success triggers if applicable.
			if finalStatus == entity.TaskStatusDoneSuccess && len(t.OnSuccess) > 0 {
				if err := fireOnSuccessTriggers(root, project, agentName, t); err != nil {
					fmt.Fprintf(os.Stderr, "warning: some triggers failed: %v\n", err)
				}
			}

			// Trigger workflow routing (idempotent — no-op if not a workflow task).
			if err := workflow.Route(root, project, t, ts, s); err != nil {
				fmt.Fprintf(os.Stderr, "warning: workflow routing failed: %v\n", err)
			}

			fmt.Printf("✓ Task %s marked %s\n", taskID, finalStatus)
			if summary != "" {
				fmt.Printf("  Summary: %s\n", summary)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&taskID, "id", "", "task ID")
	cmd.Flags().StringVar(&status, "status", "", "success or failed")
	cmd.Flags().StringVar(&errorMsg, "error", "", "error message (for failed status)")
	cmd.Flags().StringVar(&summary, "summary", "", "what was accomplished (passed to next workflow step via {{task.summary}})")
	return cmd
}

// ── task confirm-request ──────────────────────────────────────────────────────

func newTaskConfirmRequestCmd() *cobra.Command {
	var (
		taskID      string
		summary     string
		actionHint  string
		actionItems []string
	)

	cmd := &cobra.Command{
		Use:   "confirm-request",
		Short: "Route a task to the human inbox for confirmation",
		Long: `Intended to be called BY the agent when it needs human input:

  agencycli task confirm-request --id <task-id> --summary "PR #42 is ready for your review" \
    --action-item "Open https://github.com/org/repo/pull/42" \
    --action-item "Review the diff and approve or request changes" \
    --action-item "Reply with: approved / needs-changes: <reason>"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if taskID == "" || summary == "" {
				return fmt.Errorf("--id and --summary are required")
			}

			project, agentName, err := resolveTaskOwner(root, taskID)
			if err != nil {
				return err
			}

			ts := taskstore.New(root)
			t, err := ts.GetTask(project, agentName, taskID)
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			t.Status = entity.TaskStatusAwaitingConfirmation
			t.UpdatedAt = now
			t.ConfirmationReq = &entity.ConfirmationRequest{
				Summary:     summary,
				ActionHint:  actionHint,
				ActionItems: actionItems,
			}
			if err := ts.UpdateTask(project, agentName, t); err != nil {
				return err
			}

			item := &entity.InboxItem{
				TaskID:      taskID,
				Project:     project,
				Agent:       agentName,
				Title:       t.Title,
				Summary:     summary,
				ActionHint:  actionHint,
				ActionItems: actionItems,
				RoutedAt:    now,
				LogPath:     t.RunLogPath,
			}
			if err := ts.AddToInbox(item); err != nil {
				return err
			}

			fmt.Printf("✓ Task %s routed to human inbox\n", taskID)
			fmt.Printf("  Summary: %s\n", summary)
			if len(actionItems) > 0 {
				fmt.Printf("  Action items (%d):\n", len(actionItems))
				for i, item := range actionItems {
					fmt.Printf("    %d. %s\n", i+1, item)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&taskID, "id", "", "task ID")
	cmd.Flags().StringVar(&summary, "summary", "", "one-line summary for the human")
	cmd.Flags().StringVar(&actionHint, "action-hint", "", "additional context / background")
	cmd.Flags().StringArrayVar(&actionItems, "action-item", nil, "a specific action for the human (repeatable)")
	return cmd
}

// ── task retry ────────────────────────────────────────────────────────────────

func newTaskRetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry <task-id>",
		Short: "Reset a failed task back to pending",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			taskID := args[0]

			project, agentName, err := resolveTaskOwner(root, taskID)
			if err != nil {
				return err
			}

			ts := taskstore.New(root)
			archived, err := ts.ListArchivedTasks(project, agentName)
			if err != nil {
				return err
			}

			var found *entity.Task
			var remaining []*entity.Task
			for _, t := range archived {
				if t.ID == taskID {
					found = t
				} else {
					remaining = append(remaining, t)
				}
			}
			if found == nil {
				return fmt.Errorf("task %q not found in archive (only failed tasks can be retried)", taskID)
			}
			if found.Status != entity.TaskStatusDoneFailed {
				return fmt.Errorf("task %s has status %s; only done_failed tasks can be retried", taskID, found.Status)
			}

			now := time.Now().UTC()
			found.Status = entity.TaskStatusPending
			found.RetryCount++
			found.LastError = ""
			found.StartedAt = nil
			found.FinishedAt = nil
			found.UpdatedAt = now

			// Re-add to active queue.
			if err := ts.AddTask(project, agentName, found); err != nil {
				return err
			}
			// Rewrite archive without the retried task.
			if err := rewriteArchive(root, project, agentName, remaining); err != nil {
				return err
			}

			fmt.Printf("✓ Task %s reset to pending (retry %d)\n", taskID, found.RetryCount)
			return nil
		},
	}
	return cmd
}

// ── task cancel ───────────────────────────────────────────────────────────────

func newTaskCancelCmd() *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "Cancel a pending or in-progress task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			taskID := args[0]

			project, agentName, err := resolveTaskOwner(root, taskID)
			if err != nil {
				return err
			}

			ts := taskstore.New(root)
			t, err := ts.GetTask(project, agentName, taskID)
			if err != nil {
				return err
			}
			if t.Status.IsTerminal() {
				return fmt.Errorf("task %s is already in terminal state %s", taskID, t.Status)
			}

			now := time.Now().UTC()
			t.Status = entity.TaskStatusCancelled
			t.FinishedAt = &now
			t.UpdatedAt = now
			if reason != "" {
				t.LastError = reason
			}

			if err := ts.ArchiveTask(project, agentName, t); err != nil {
				return err
			}
			// Remove from inbox if present.
			_ = ts.RemoveFromInbox(taskID)

			fmt.Printf("✓ Task %s cancelled\n", taskID)
			return nil
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "cancellation reason")
	return cmd
}

// ── helpers ───────────────────────────────────────────────────────────────────

// resolveTaskOwner searches all projects/agents in the workspace for the task.
// This allows callers (agents) to omit --project/--agent when using task done/
// confirm-request from within their working directory.
func resolveTaskOwner(root, taskID string) (project, agent string, err error) {
	ts := taskstore.New(root)
	projects, err := ts.ListProjects()
	if err != nil {
		return "", "", err
	}
	for _, p := range projects {
		agents, err := ts.ListAgents(p)
		if err != nil {
			continue
		}
		for _, a := range agents {
			t, _ := ts.GetTask(p, a, taskID)
			if t != nil {
				return p, a, nil
			}
		}
	}
	return "", "", fmt.Errorf("task %q not found in any project/agent", taskID)
}

// fireOnSuccessTriggers creates follow-up tasks defined in t.OnSuccess.
func fireOnSuccessTriggers(root, project, agentName string, t *entity.Task) error {
	ts := taskstore.New(root)
	var errs []string
	for _, trigger := range t.OnSuccess {
		// Parse assignee: "<project>/<agent>" or "human"
		targetProject, targetAgent, err := parseAssignee(trigger.Assignee, project)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		now := time.Now().UTC()
		newTask := &entity.Task{
			ID:        entity.NewTaskID(),
			Title:     trigger.Title,
			Type:      entity.TaskType(trigger.Type),
			Priority:  trigger.Priority,
			Assignee:  trigger.Assignee,
			CreatedBy: project + "/" + agentName,
			Status:    entity.TaskStatusPending,
			Prompt:    trigger.Prompt,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if trigger.Assignee == "human" {
			if err := ts.AddTask(targetProject, targetAgent, newTask); err != nil {
				errs = append(errs, err.Error())
				continue
			}
			item := &entity.InboxItem{
				TaskID:   newTask.ID,
				Project:  targetProject,
				Agent:    targetAgent,
				Title:    newTask.Title,
				Summary:  newTask.Prompt,
				RoutedAt: now,
			}
			_ = ts.AddToInbox(item)
		} else {
			if err := ts.AddTask(targetProject, targetAgent, newTask); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func parseAssignee(assignee, fallbackProject string) (project, agent string, err error) {
	if assignee == "human" {
		return fallbackProject, "human", nil
	}
	parts := strings.SplitN(assignee, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid assignee %q (expected <project>/<agent>)", assignee)
	}
	return parts[0], parts[1], nil
}

func rewriteArchive(root, project, agentName string, tasks []*entity.Task) error {
	return taskstore.New(root).OverwriteArchive(project, agentName, tasks)
}

// resolveStores returns a taskstore.Store and an org store.Store for the workspace.
func resolveStores(root string) (taskstore.Store, store.Store) {
	return taskstore.New(root), store.NewFS(root)
}
