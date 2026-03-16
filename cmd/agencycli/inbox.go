package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/agencycli/agencycli/internal/entity"
	"github.com/agencycli/agencycli/internal/runner"
	"github.com/agencycli/agencycli/internal/store"
	"github.com/agencycli/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
)

func newInboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Manage tasks awaiting your confirmation",
		Long: `inbox shows all tasks that have been routed to you for confirmation.

Agents route tasks here when they need human input before proceeding.
Use 'inbox confirm' or 'inbox reject' to resolve them.`,
	}
	cmd.AddCommand(
		newInboxListCmd(),
		newInboxShowCmd(),
		newInboxConfirmCmd(),
		newInboxRejectCmd(),
		newInboxCommentCmd(),
	)
	return cmd
}

// ── inbox list ────────────────────────────────────────────────────────────────

func newInboxListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all items awaiting confirmation",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ts := taskstore.New(root)
			items, err := ts.ListInbox()
			if err != nil {
				return err
			}

			if len(items) == 0 {
				fmt.Println("Inbox is empty.")
				return nil
			}

			fmt.Printf("Inbox — %d item(s) awaiting confirmation\n\n", len(items))
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TASK ID\tPROJECT/AGENT\tTITLE\tROUTED AT")
			fmt.Fprintln(w, "───────\t─────────────\t─────\t─────────")
			for _, item := range items {
				fmt.Fprintf(w, "%s\t%s/%s\t%s\t%s\n",
					item.TaskID,
					item.Project, item.Agent,
					item.Title,
					item.RoutedAt.Format("2006-01-02 15:04"),
				)
			}
			w.Flush()
			fmt.Printf("\nRun 'agencycli inbox show <task-id>' for details.\n")
			return nil
		},
	}
}

// ── inbox show ────────────────────────────────────────────────────────────────

func newInboxShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show details of an inbox item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			taskID := args[0]
			ts := taskstore.New(root)
			items, err := ts.ListInbox()
			if err != nil {
				return err
			}
			var found *entity.InboxItem
			for _, item := range items {
				if item.TaskID == taskID {
					found = item
					break
				}
			}
			if found == nil {
				return fmt.Errorf("inbox item %q not found", taskID)
			}

			fmt.Printf("Task ID  : %s\n", found.TaskID)
			fmt.Printf("Agent    : %s / %s\n", found.Project, found.Agent)
			fmt.Printf("Title    : %s\n", found.Title)
			fmt.Printf("Routed   : %s\n", found.RoutedAt.Format(time.RFC3339))
			if found.Summary != "" {
				fmt.Printf("\nSummary:\n%s\n", found.Summary)
			}
			if found.ActionHint != "" {
				fmt.Printf("\nHint: %s\n", found.ActionHint)
			}
			if found.LogPath != "" {
				fmt.Printf("\nLog: %s\n", found.LogPath)
			}
			fmt.Printf("\nagencycli inbox confirm %s\n", taskID)
			fmt.Printf("agencycli inbox reject  %s --reason \"...\"\n", taskID)
			fmt.Printf("agencycli inbox comment %s --message \"...\"\n", taskID)
			return nil
		},
	}
}

// ── inbox confirm ─────────────────────────────────────────────────────────────

func newInboxConfirmCmd() *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:   "confirm <task-id>",
		Short: "Confirm and continue a task",
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
			if t.Status != entity.TaskStatusAwaitingConfirmation {
				return fmt.Errorf("task %s is in state %s, not awaiting_confirmation", taskID, t.Status)
			}

			hb, err := ts.GetHeartbeat(project, agentName)
			if err != nil {
				return err
			}

			reply := message
			if reply == "" {
				reply = "Confirmed. Please proceed."
			}
			t.ConfirmationReply = reply

			// Re-run the agent with the confirmation context.
			s := store.NewFS(root)
			r := runner.New(root, ts, s)
			fmt.Printf("▶ Resuming task %s with your confirmation...\n", taskID)

			now := time.Now().UTC()
			t.Status = entity.TaskStatusInProgress
			t.UpdatedAt = now
			if err := ts.UpdateTask(project, agentName, t); err != nil {
				return err
			}

			result, err := r.ResumeTask(project, agentName, t, reply, hb.SessionID)
			if err != nil {
				return fmt.Errorf("resume execution error: %w", err)
			}

			// Update session ID if changed.
			if result.SessionID != "" && result.SessionID != hb.SessionID {
				hb.SessionID = result.SessionID
				_ = ts.SaveHeartbeat(project, agentName, hb)
			}

			t.RunLogPath = result.LogPath
			finished := time.Now().UTC()
			t.FinishedAt = &finished
			t.Status = result.Status

			switch result.Status {
			case entity.TaskStatusDoneSuccess:
				fmt.Printf("✓ Task %s completed after confirmation\n", taskID)
				_ = ts.ArchiveTask(project, agentName, t)
				_ = ts.RemoveFromInbox(taskID)
				if len(t.OnSuccess) > 0 {
					_ = fireOnSuccessTriggers(root, project, agentName, t)
				}

			case entity.TaskStatusDoneFailed:
				t.LastError = result.ErrorMsg
				fmt.Printf("✗ Task %s failed after confirmation: %s\n", taskID, result.ErrorMsg)
				_ = ts.ArchiveTask(project, agentName, t)
				_ = ts.RemoveFromInbox(taskID)

			case entity.TaskStatusAwaitingConfirmation:
				// Agent needs another round.
				t.ConfirmationReq = &entity.ConfirmationRequest{Summary: result.Summary}
				t.UpdatedAt = time.Now().UTC()
				_ = ts.UpdateTask(project, agentName, t)
				// Update inbox item summary.
				_ = ts.RemoveFromInbox(taskID)
				item := &entity.InboxItem{
					TaskID:   taskID,
					Project:  project,
					Agent:    agentName,
					Title:    t.Title,
					Summary:  result.Summary,
					RoutedAt: time.Now().UTC(),
					LogPath:  result.LogPath,
				}
				_ = ts.AddToInbox(item)
				fmt.Printf("? Task %s needs another confirmation round\n", taskID)
				fmt.Printf("  %s\n", result.Summary)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&message, "message", "", "confirmation message to the agent")
	return cmd
}

// ── inbox reject ──────────────────────────────────────────────────────────────

func newInboxRejectCmd() *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "reject <task-id>",
		Short: "Reject and cancel a task",
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

			now := time.Now().UTC()
			t.Status = entity.TaskStatusCancelled
			t.FinishedAt = &now
			t.UpdatedAt = now
			if reason != "" {
				t.LastError = "rejected: " + reason
			}

			if err := ts.ArchiveTask(project, agentName, t); err != nil {
				return err
			}
			_ = ts.RemoveFromInbox(taskID)

			fmt.Printf("✓ Task %s rejected and cancelled\n", taskID)
			return nil
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "reason for rejection")
	return cmd
}

// ── inbox comment ─────────────────────────────────────────────────────────────

func newInboxCommentCmd() *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:   "comment <task-id>",
		Short: "Add a comment to an inbox item (task stays awaiting)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if message == "" {
				return fmt.Errorf("--message is required")
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

			// Append comment to ConfirmationReply (the agent will see it on next resume).
			if t.ConfirmationReply == "" {
				t.ConfirmationReply = message
			} else {
				t.ConfirmationReply += "\n" + message
			}
			t.UpdatedAt = time.Now().UTC()

			if err := ts.UpdateTask(project, agentName, t); err != nil {
				return err
			}

			fmt.Printf("✓ Comment added to task %s\n", taskID)
			fmt.Printf("  (Task remains in awaiting_confirmation — use 'inbox confirm' to resume)\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&message, "message", "", "comment text")
	return cmd
}
