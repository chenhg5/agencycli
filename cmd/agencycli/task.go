package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/chenhg5/agencycli/internal/telemetry"
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
		newTaskFindCmd(),
		newTaskDoneCmd(),
		newTaskConfirmRequestCmd(),
		newTaskRetryCmd(),
		newTaskCancelCmd(),
		newTaskStopAllCmd(),
		newTaskTokensCmd(),
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
				item := &entity.InboxItem{
					TaskID:  t.ID,
					Project: project,
					Agent:   agentName,
					Title:   t.Title,
					Summary: promptText,
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
		format    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks for an agent",
		Example: `  agencycli task list --project cc-connect --agent qa-reviewer
  agencycli task list --project cc-connect --agent qa-reviewer --status pending --format table
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

			if len(tasks) == 0 && format == "table" {
				fmt.Println("No tasks found.")
				return nil
			}

			// Sort: in_progress first, then by priority asc, then created_at asc.
			sort.Slice(tasks, func(i, j int) bool {
				ti, tj := tasks[i], tasks[j]
				// in_progress always first
				iRun := ti.Status == entity.TaskStatusInProgress
				jRun := tj.Status == entity.TaskStatusInProgress
				if iRun != jRun {
					return iRun
				}
				if ti.Priority != tj.Priority {
					return ti.Priority < tj.Priority
				}
				return ti.CreatedAt.Before(tj.CreatedAt)
			})

			if format == "json" || format == "" {
				if tasks == nil {
					tasks = []*entity.Task{}
				}
				return printJSON(tasks)
			}

			// --format table
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "STATUS\tID\tPRI\tCREATED\tTITLE")
			fmt.Fprintln(w, "──────\t──\t───\t───────\t─────")
			for _, t := range tasks {
				fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\t%s\n",
					taskstore.StatusIcon(t.Status), t.Status,
					t.ID,
					taskstore.PriorityLabel(t.Priority),
					t.CreatedAt.Local().Format("01-02 15:04"),
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
	cmd.Flags().StringVar(&format, "format", "json", "output format: json or table")
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

// ── task find ─────────────────────────────────────────────────────────────────

func newTaskFindCmd() *cobra.Command {
	var taskID string

	cmd := &cobra.Command{
		Use:   "find",
		Short: "Find a task by ID (searches all projects and agents)",
		Long: `Find a task anywhere in the workspace by its ID.

Searches active and archived tasks across every project and agent.
Useful when you have a task ID but don't know which agent owns it.

Example:
  agencycli task find --id task_abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if taskID == "" {
				return fmt.Errorf("--id is required")
			}
			root, err := resolveRoot()
			if err != nil {
				return err
			}

			ts := taskstore.New(root)
			proj, ag, t, err := ts.FindTaskByID(taskID)
			if err != nil {
				return err
			}

			fmt.Printf("Project  : %s\n", proj)
			fmt.Printf("Agent    : %s\n", ag)
			fmt.Printf("ID       : %s\n", t.ID)
			fmt.Printf("Title    : %s\n", t.Title)
			fmt.Printf("Status   : %s %s\n", taskstore.StatusIcon(t.Status), t.Status)
			fmt.Printf("Type     : %s\n", t.Type)
			fmt.Printf("Priority : %s (%d)\n", taskstore.PriorityLabel(t.Priority), t.Priority)
			if t.Assignee != "" {
				fmt.Printf("Assignee : %s\n", t.Assignee)
			}
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

	cmd.Flags().StringVar(&taskID, "id", "", "task ID to find (required)")
	_ = cmd.MarkFlagRequired("id")
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
		to          string
	)

	cmd := &cobra.Command{
		Use:   "confirm-request",
		Short: "Route a task to an inbox for confirmation",
		Long: `Intended to be called BY the agent when it needs input from a human or another agent:

  agencycli task confirm-request --id <task-id> --summary "PR #42 is ready for your review" \
    --action-item "Open https://github.com/org/repo/pull/42" \
    --action-item "Review the diff and approve or request changes" \
    --action-item "Reply with: approved / needs-changes: <reason>"

Use --to to route to a specific agent instead of the default human inbox:

  agencycli task confirm-request --id <task-id> --to cc-connect/pm --summary "PR ready"`,
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

			recipient := to
			if recipient == "" {
				recipient = "human"
			}
			item := &entity.InboxItem{
				TaskID:      taskID,
				Project:     project,
				Agent:       agentName,
				To:          recipient,
				Title:       t.Title,
				Summary:     summary,
				ActionHint:  actionHint,
				ActionItems: actionItems,
				LogPath:     t.RunLogPath,
			}
			if err := ts.AddToInbox(item); err != nil {
				return err
			}

			fmt.Printf("✓ Task %s routed to %s inbox\n", taskID, recipient)
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
	cmd.Flags().StringVar(&summary, "summary", "", "one-line summary for the recipient")
	cmd.Flags().StringVar(&actionHint, "action-hint", "", "additional context / background")
	cmd.Flags().StringVar(&to, "to", "", "recipient: 'human' (default) or 'project/agent' (e.g. cc-connect/pm)")
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
				TaskID:  newTask.ID,
				Project: targetProject,
				Agent:   targetAgent,
				Title:   newTask.Title,
				Summary: newTask.Prompt,
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

// ── task stop-all ─────────────────────────────────────────────────────────────

func newTaskStopAllCmd() *cobra.Command {
	var (
		project    string
		agentName  string
		allAgents  bool
		noPending  bool
	)

	cmd := &cobra.Command{
		Use:   "stop-all",
		Short: "Cancel all pending (and optionally in-progress) tasks",
		Long: `Cancels every pending task in the queue. Tasks that are currently in-progress
(agent is running) are also cancelled in the store so no workflow routing fires
when they finish — but the running Docker container is not forcibly killed.

Use --no-pending to skip pending tasks and only cancel in-progress ones.`,
		Example: `  # Cancel all pending tasks for one agent
  agencycli task stop-all --project cc-connect --agent dev-claude

  # Cancel all pending tasks across the whole project
  agencycli task stop-all --project cc-connect --all-agents

  # Cancel including in-progress (marks them failed in store)
  agencycli task stop-all --project cc-connect --all-agents --include-running`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			includeRunning, _ := cmd.Flags().GetBool("include-running")

			ts := taskstore.New(root)
			s := store.NewFS(root)

			// Collect agents to process.
			var agents []string
			if allAgents {
				agents, err = ts.ListAgents(project)
				if err != nil {
					return err
				}
			} else {
				if agentName == "" {
					return fmt.Errorf("--agent or --all-agents is required")
				}
				agents = []string{agentName}
			}
			_ = s

			total := 0
			for _, ag := range agents {
				tasks, err := ts.ListTasks(project, ag)
				if err != nil {
					continue
				}
				for _, t := range tasks {
					switch t.Status {
					case entity.TaskStatusPending, entity.TaskStatusBlocked:
						if noPending {
							continue
						}
					case entity.TaskStatusInProgress:
						if !includeRunning {
							continue
						}
					default:
						continue
					}

					now := time.Now().UTC()
					t.Status = entity.TaskStatusCancelled
					t.FinishedAt = &now
					t.UpdatedAt = now
					t.LastError = "cancelled by stop-all"
					if err := ts.ArchiveTask(project, ag, t); err != nil {
						fmt.Fprintf(os.Stderr, "  warn: cancel %s/%s: %v\n", ag, t.ID, err)
						continue
					}
					fmt.Printf("  ✗ cancelled  %-22s  %s/%s\n", t.ID, ag, t.Title)
					total++
				}
			}

			if total == 0 {
				fmt.Println("No cancellable tasks found.")
			} else {
				fmt.Printf("\n✓ Cancelled %d task(s)\n", total)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	cmd.Flags().BoolVar(&allAgents, "all-agents", false, "apply to all agents in the project")
	cmd.Flags().BoolVar(&noPending, "no-pending", false, "skip pending tasks (only cancel in-progress)")
	cmd.Flags().Bool("include-running", false, "also cancel tasks currently in-progress")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// ── task tokens ───────────────────────────────────────────────────────────────

// tokenUsage holds token counts from a single run log.
type tokenUsage struct {
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	TotalCostUSD      float64
	HasCost           bool // true when total_cost_usd came from the log
}

func newTaskTokensCmd() *cobra.Command {
	var (
		project   string
		agentName string
		taskID    string
		allTasks  bool
	)

	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Show token usage and estimated cost from run logs",
		Long: `Parses Claude stream-json run logs and aggregates input/output token counts.
Cost is estimated using Anthropic's Claude pricing (configurable via env):
  ANTHROPIC_INPUT_PRICE_PER_M  (default: 3.0  USD per 1M input tokens)
  ANTHROPIC_OUTPUT_PRICE_PER_M (default: 15.0 USD per 1M output tokens)`,
		Example: `  # Tokens for a specific task
  agencycli task tokens --project cc-connect --agent pm --task t-20260317-18omal

  # Aggregate all tasks for an agent
  agencycli task tokens --project cc-connect --agent pm --all

  # All agents in project
  agencycli task tokens --project cc-connect --all-agents`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}

			allAgentsFlag, _ := cmd.Flags().GetBool("all-agents")

			ts := taskstore.New(root)

			inputPrice := getEnvFloat("ANTHROPIC_INPUT_PRICE_PER_M", 3.0)
			outputPrice := getEnvFloat("ANTHROPIC_OUTPUT_PRICE_PER_M", 15.0)

			var agentList []string
			if allAgentsFlag {
				agentList, err = ts.ListAgents(project)
				if err != nil {
					return err
				}
			} else {
				if agentName == "" {
					return fmt.Errorf("--agent or --all-agents is required")
				}
				agentList = []string{agentName}
			}

			type agentSummary struct {
				name  string
				usage tokenUsage
				tasks int
			}
			var summaries []agentSummary
			grandTotal := tokenUsage{}

			for _, ag := range agentList {
				logDir, err := ts.RunLogDir(project, ag)
				if err != nil {
					continue
				}

				entries, err := os.ReadDir(logDir)
				if err != nil {
					continue
				}

				agUsage := tokenUsage{}
				taskCount := 0

				for _, e := range entries {
					if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
						continue
					}
					// Filter by task ID if specified.
					if taskID != "" && !strings.Contains(e.Name(), taskID) {
						continue
					}

					u, err := parseLogTokens(logDir + "/" + e.Name())
					if err != nil {
						continue
					}
					if u.InputTokens == 0 && u.OutputTokens == 0 && !u.HasCost {
						continue
					}
					agUsage.InputTokens += u.InputTokens
					agUsage.OutputTokens += u.OutputTokens
					agUsage.CacheReadTokens += u.CacheReadTokens
					agUsage.TotalCostUSD += u.TotalCostUSD
					agUsage.HasCost = agUsage.HasCost || u.HasCost
					taskCount++

					if taskID != "" || allTasks {
						costStr := fmt.Sprintf("$%.4f", u.TotalCostUSD)
						if !u.HasCost {
							costStr = fmt.Sprintf("~$%.4f", calcCost(u.InputTokens, u.OutputTokens, inputPrice, outputPrice))
						}
						fmt.Printf("  %-44s  in=%7s  out=%6s  cache=%6s  %s\n",
							e.Name(),
							formatTokens(u.InputTokens),
							formatTokens(u.OutputTokens),
							formatTokens(u.CacheReadTokens),
							costStr,
						)
					}
				}

			summaries = append(summaries, agentSummary{ag, agUsage, taskCount})
			grandTotal.InputTokens += agUsage.InputTokens
			grandTotal.OutputTokens += agUsage.OutputTokens
			grandTotal.CacheReadTokens += agUsage.CacheReadTokens
			grandTotal.TotalCostUSD += agUsage.TotalCostUSD
			grandTotal.HasCost = grandTotal.HasCost || agUsage.HasCost
		}

			fmt.Println()
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "AGENT\tTASKS\tINPUT\tOUTPUT\tCACHE HIT\tCOST")
			fmt.Fprintln(w, "─────\t─────\t─────\t──────\t─────────\t────")
			for _, s := range summaries {
				var costStr string
				if s.usage.HasCost {
					costStr = fmt.Sprintf("$%.4f", s.usage.TotalCostUSD)
				} else {
					costStr = fmt.Sprintf("~$%.4f", calcCost(s.usage.InputTokens, s.usage.OutputTokens, inputPrice, outputPrice))
				}
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\n",
					s.name, s.tasks,
					formatTokens(s.usage.InputTokens),
					formatTokens(s.usage.OutputTokens),
					formatTokens(s.usage.CacheReadTokens),
					costStr,
				)
			}
			if len(summaries) > 1 {
				var totalCostStr string
				if grandTotal.HasCost {
					totalCostStr = fmt.Sprintf("$%.4f", grandTotal.TotalCostUSD)
				} else {
					totalCostStr = fmt.Sprintf("~$%.4f", calcCost(grandTotal.InputTokens, grandTotal.OutputTokens, inputPrice, outputPrice))
				}
				fmt.Fprintln(w, "─────\t─────\t─────\t──────\t─────────\t────")
				fmt.Fprintf(w, "TOTAL\t-\t%s\t%s\t%s\t%s\n",
					formatTokens(grandTotal.InputTokens),
					formatTokens(grandTotal.OutputTokens),
					formatTokens(grandTotal.CacheReadTokens),
					totalCostStr,
				)
			}
			w.Flush()
			if !grandTotal.HasCost {
				fmt.Printf("\nEstimated pricing: $%.2f/M input, $%.2f/M output\n(override with ANTHROPIC_INPUT_PRICE_PER_M / ANTHROPIC_OUTPUT_PRICE_PER_M)\n",
					inputPrice, outputPrice)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	cmd.Flags().StringVar(&taskID, "task", "", "filter by task ID")
	cmd.Flags().BoolVar(&allTasks, "all", false, "show per-run breakdown")
	cmd.Flags().Bool("all-agents", false, "aggregate across all agents in the project")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// ── token helpers ─────────────────────────────────────────────────────────────

func parseLogTokens(path string) (tokenUsage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return tokenUsage{}, err
	}
	u := telemetry.ParseStreamJSONUsage(data)
	return tokenUsage{
		InputTokens:     u.InputTokens,
		OutputTokens:    u.OutputTokens,
		CacheReadTokens: u.CacheReadTokens,
		TotalCostUSD:    u.TotalCostUSD,
		HasCost:         u.SawResult,
	}, nil
}

func calcCost(in, out int64, inPricePerM, outPricePerM float64) float64 {
	return float64(in)/1e6*inPricePerM + float64(out)/1e6*outPricePerM
}

func formatTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.2fM", float64(n)/1e6)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	}
	return fmt.Sprintf("%d", n)
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			return f
		}
	}
	return def
}

// Ensure math is used (needed for potential future rounding).
var _ = math.Round
