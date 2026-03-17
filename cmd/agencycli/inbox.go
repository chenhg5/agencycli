package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/runner"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
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
		newInboxForwardCmd(),
		newInboxCommentCmd(),
		// Async message commands (non-blocking, any participant can use these)
		newInboxSendCmd(),
		newInboxMessagesCmd(),
		newInboxReplyCmd(),
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
			fmt.Fprintln(w, "TASK ID\tPROJECT/AGENT\tTITLE")
			fmt.Fprintln(w, "───────\t─────────────\t─────")
			for _, item := range items {
				fmt.Fprintf(w, "%s\t%s/%s\t%s\n",
					item.TaskID,
					item.Project, item.Agent,
					item.Title,
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
		Short: "Show full details of an inbox item",
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

			hr := "────────────────────────────────────────────────────────────"
			fmt.Println(hr)
			fmt.Printf("  INBOX ITEM  %s\n", found.TaskID)
			fmt.Println(hr)
			fmt.Printf("  From    : %s / %s\n", found.Project, found.Agent)
			fmt.Printf("  Title   : %s\n", found.Title)
			if found.ForwardedTo != "" {
				fmt.Printf("  Status  : forwarded → %s\n", found.ForwardedTo)
			}
			fmt.Println()

			// Agent's summary of what it did and why it needs help
			if found.Summary != "" {
				fmt.Println("── What the agent says ──────────────────────────────────────")
				fmt.Println(found.Summary)
				fmt.Println()
			}
			if found.ActionHint != "" {
				fmt.Println("── Background / context ─────────────────────────────────────")
				fmt.Println(found.ActionHint)
				fmt.Println()
			}

			// Action items checklist
			if len(found.ActionItems) > 0 {
				fmt.Println("── Action items (what you need to do) ───────────────────────")
				for i, item := range found.ActionItems {
					fmt.Printf("  %d. %s\n", i+1, item)
				}
				fmt.Println()
			}

			// Original task prompt — the full context of what the agent was working on
			project, agentName, _ := resolveTaskOwner(root, taskID)
			if project != "" {
				if t, err2 := taskstore.New(root).GetTask(project, agentName, taskID); err2 == nil {
					fmt.Println("── Original task (full prompt) ──────────────────────────────")
					fmt.Println(t.Prompt)
					fmt.Println()
				}
			}

			// Last lines of the agent's run log
			if found.LogPath != "" {
				fmt.Printf("── Last run log  (%s)\n", found.LogPath)
				if lines, err2 := tailFile(found.LogPath, 20); err2 == nil && len(lines) > 0 {
					for _, l := range lines {
						fmt.Println("  " + l)
					}
				} else {
					fmt.Println("  (log unavailable)")
				}
				fmt.Println()
			}

			// Available actions
			fmt.Println("── Available actions ────────────────────────────────────────")
			fmt.Printf("  agencycli --dir %s inbox confirm %s --message \"your reply\"\n", root, taskID)
			fmt.Printf("  agencycli --dir %s inbox reject  %s --reason \"...\"\n", root, taskID)
			fmt.Printf("  agencycli --dir %s inbox forward %s --to <project>/<agent> --note \"...\"\n", root, taskID)
			fmt.Printf("  agencycli --dir %s inbox comment %s --message \"...\"\n", root, taskID)
			fmt.Println(hr)
			return nil
		},
	}
}

// tailFile returns the last n lines of a file.
func tailFile(path string, n int) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
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
					TaskID:  taskID,
					Project: project,
					Agent:   agentName,
					Title:   t.Title,
					Summary: result.Summary,
					LogPath: result.LogPath,
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

// ── inbox forward ─────────────────────────────────────────────────────────────

func newInboxForwardCmd() *cobra.Command {
	var (
		to   string
		note string
	)

	cmd := &cobra.Command{
		Use:   "forward <task-id>",
		Short: "Forward a task to another agent for action, then return to inbox",
		Long: `Forward an inbox item to another agent (e.g. qa-reviewer) for them to do
work on it. The forwarded task carries the full original context plus your note.
When that agent finishes, it should call confirm-request which will route the
result back to your inbox so you can make the final decision.

  agencycli inbox forward t-20260317-abc123 --to cc-connect/qa-reviewer \
    --note "Please review the diff and let me know if it looks safe to merge."`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == "" {
				return fmt.Errorf("--to is required (format: <project>/<agent>)")
			}
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			taskID := args[0]

			// Resolve originating task
			project, agentName, err := resolveTaskOwner(root, taskID)
			if err != nil {
				return err
			}
			ts := taskstore.New(root)
			t, err := ts.GetTask(project, agentName, taskID)
			if err != nil {
				return err
			}

			// Parse forwarding target
			parts := strings.SplitN(to, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return fmt.Errorf("--to must be in format <project>/<agent>, e.g. cc-connect/qa-reviewer")
			}
			targetProject, targetAgent := parts[0], parts[1]

			// Build forwarded task prompt — original context + human note + instructions
			forwardPrompt := fmt.Sprintf(`# Forwarded task: %s

**Original task from %s/%s:**

%s

---

**Human note (why this was forwarded to you):**
%s

---

**Instructions:**
Work through the above task/request and report your findings/result.
When done, call:
  agencycli task confirm-request --id <your-task-id> \
    --summary "<what you found / what action you took>" \
    --action-item "Review my findings below and confirm or adjust"

The human will see your report and make the final decision.`,
				t.Title,
				project, agentName,
				t.Prompt,
				noteOrDefault(note, "(no specific note — please review and report back)"),
			)

			now := time.Now().UTC()
			forwarded := &entity.Task{
				ID:        entity.NewTaskID(),
				Title:     "[Forwarded] " + t.Title,
				Type:      t.Type,
				Priority:  t.Priority,
				Assignee:  to,
				CreatedBy: "human (forwarded from " + project + "/" + agentName + ")",
				Status:    entity.TaskStatusPending,
				Prompt:    forwardPrompt,
				CreatedAt: now,
				UpdatedAt: now,
			}

			if err := ts.AddTask(targetProject, targetAgent, forwarded); err != nil {
				return fmt.Errorf("could not create task for %s: %w", to, err)
			}

			// Mark original inbox item as forwarded (update in place)
			items, _ := ts.ListInbox()
			for _, item := range items {
				if item.TaskID == taskID {
					item.ForwardedTo = to
					item.ForwardNote = note
					// Re-save by removing and re-adding (simpler than partial update)
					_ = ts.RemoveFromInbox(taskID)
					_ = ts.AddToInbox(item)
					break
				}
			}

			fmt.Printf("✓ Forwarded task %q to %s\n", t.Title, to)
			fmt.Printf("  New task ID : %s\n", forwarded.ID)
			fmt.Printf("  Original inbox item %s remains in awaiting_confirmation.\n", taskID)
			fmt.Printf("  When %s finishes, it will route results back to your inbox.\n", to)
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "target agent in format <project>/<agent>")
	cmd.Flags().StringVar(&note, "note", "", "your note to the target agent explaining what you need")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func noteOrDefault(note, def string) string {
	if note != "" {
		return note
	}
	return def
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

// ── inbox send ────────────────────────────────────────────────────────────────

func newInboxSendCmd() *cobra.Command {
	var (
		to      string
		subject string
		body    string
		replyTo string
		from    string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send an async message to an agent or human",
		Long: `Send a non-blocking async message to any participant.

Recipient format:
  --to human                 → agency owner's inbox
  --to cc-connect/pm         → project cc-connect, agent pm
  --to cc-connect/dev-claude → project cc-connect, agent dev-claude

The recipient will see the message on their next wakeup (agents) or in
'inbox messages' (human).  Unlike 'task confirm-request', sending a message
does not block any task.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if body == "" {
				return fmt.Errorf("--body is required")
			}
			sender := from
			if sender == "" {
				sender = "human"
			}
			msg := &entity.Message{
				ID:      entity.NewMessageID(),
				From:    sender,
				To:      to,
				Subject: subject,
				Body:    body,
				ReplyTo: replyTo,
				SentAt:  time.Now().UTC(),
			}
			ts := taskstore.New(root)
			if err := ts.SendMessage(msg); err != nil {
				return fmt.Errorf("send message: %w", err)
			}
			fmt.Printf("✓ Message sent  [%s]\n", msg.ID)
			fmt.Printf("  To      : %s\n", msg.To)
			if msg.Subject != "" {
				fmt.Printf("  Subject : %s\n", msg.Subject)
			}
			fmt.Printf("  From    : %s\n", msg.From)
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "recipient: 'human' or 'project/agent'")
	cmd.Flags().StringVar(&subject, "subject", "", "optional subject line")
	cmd.Flags().StringVar(&body, "body", "", "message body")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "ID of message being replied to")
	cmd.Flags().StringVar(&from, "from", "", "override sender (defaults to 'human'; agents set this to 'project/agent')")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

// ── inbox messages ────────────────────────────────────────────────────────────

func newInboxMessagesCmd() *cobra.Command {
	var (
		recipient string
		all       bool
		mark      bool
	)

	cmd := &cobra.Command{
		Use:   "messages",
		Short: "List async messages (from agents or other humans)",
		Long: `List async messages delivered to a mailbox.

By default shows only unread messages for 'human'.
Use --recipient to inspect an agent's mailbox.
Use --all to show all messages including already-read ones.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if recipient == "" {
				recipient = "human"
			}
			ts := taskstore.New(root)
			var msgs []*entity.Message
			if all {
				msgs, err = ts.ListMessages(recipient)
			} else {
				msgs, err = ts.ListUnreadMessages(recipient)
			}
			if err != nil {
				return err
			}
			if len(msgs) == 0 {
				if all {
					fmt.Printf("No messages for %s.\n", recipient)
				} else {
					fmt.Printf("No unread messages for %s.\n", recipient)
				}
				return nil
			}
			fmt.Printf("Messages for %s (%d):\n\n", recipient, len(msgs))
			for _, m := range msgs {
				status := "●"
				if m.ReadAt != nil {
					status = "○"
				}
				fmt.Printf("%s [%s] ID: %s\n", status, m.SentAt.Local().Format("01-02 15:04"), m.ID)
				fmt.Printf("  From    : %s\n", m.From)
				if m.Subject != "" {
					fmt.Printf("  Subject : %s\n", m.Subject)
				}
				if m.ReplyTo != "" {
					fmt.Printf("  Reply-to: %s\n", m.ReplyTo)
				}
				fmt.Printf("\n  %s\n\n", strings.ReplaceAll(m.Body, "\n", "\n  "))
				fmt.Println(strings.Repeat("─", 60))
			}
			if mark {
				if err := ts.MarkMessagesRead(recipient); err != nil {
					return err
				}
				fmt.Printf("✓ Marked %d message(s) as read.\n", len(msgs))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&recipient, "recipient", "", "mailbox to inspect: 'human' (default) or 'project/agent'")
	cmd.Flags().BoolVar(&all, "all", false, "show all messages including already-read ones")
	cmd.Flags().BoolVar(&mark, "mark-read", false, "mark displayed messages as read after listing")
	return cmd
}

// ── inbox reply ───────────────────────────────────────────────────────────────

func newInboxReplyCmd() *cobra.Command {
	var (
		body      string
		from      string
	)

	cmd := &cobra.Command{
		Use:   "reply <msg-id>",
		Short: "Reply to an async message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			msgID := args[0]
			if body == "" {
				return fmt.Errorf("--body is required")
			}
			ts := taskstore.New(root)

			// Find the original message to determine the reply recipient.
			// Search human mailbox first, then all agents.
			var original *entity.Message
			recipients := []string{"human"}
			projects, _ := ts.ListProjects()
			for _, proj := range projects {
				agents, _ := ts.ListAgents(proj)
				for _, ag := range agents {
					recipients = append(recipients, proj+"/"+ag)
				}
			}
			for _, rec := range recipients {
				all, _ := ts.ListMessages(rec)
				for _, m := range all {
					if m.ID == msgID {
						original = m
						break
					}
				}
				if original != nil {
					break
				}
			}
			if original == nil {
				return fmt.Errorf("message %s not found", msgID)
			}

			sender := from
			if sender == "" {
				sender = "human"
			}
			reply := &entity.Message{
				ID:      entity.NewMessageID(),
				From:    sender,
				To:      original.From, // reply goes back to the sender
				Subject: "Re: " + original.Subject,
				Body:    body,
				ReplyTo: msgID,
				SentAt:  time.Now().UTC(),
			}
			if err := ts.SendMessage(reply); err != nil {
				return err
			}
			fmt.Printf("✓ Reply sent  [%s]\n", reply.ID)
			fmt.Printf("  To      : %s\n", reply.To)
			fmt.Printf("  Re      : %s\n", msgID)
			return nil
		},
	}

	cmd.Flags().StringVar(&body, "body", "", "reply body")
	cmd.Flags().StringVar(&from, "from", "", "override sender (defaults to 'human')")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}
