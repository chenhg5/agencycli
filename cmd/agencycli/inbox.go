package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
)

// validateRecipient checks that recipient is either "human" or "project/agent".
// If a bare agent name (no "/") is used, it looks up whether it's a known agent
// and returns an error suggesting the correct project/agent format.
func validateRecipient(ts taskstore.Store, recipient string) error {
	ident := validateIdentity(ts, recipient, "recipient")
	if ident != nil {
		return ident
	}
	return nil
}

// validateIdentity checks that identity is either "human" or "project/agent".
// For "project/agent", it also verifies the project and agent exist.
// If a bare agent name (no "/") is used, it looks up whether it's a known agent
// and returns an error suggesting the correct project/agent format or using "human".
func validateIdentity(ts taskstore.Store, identity, fieldName string) error {
	if identity == "human" {
		return nil
	}
	if strings.Contains(identity, "/") {
		// Validate project/agent format exists
		parts := strings.SplitN(identity, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid %s %q: expected 'human' or 'project/agent'", fieldName, identity)
		}
		project, agent := parts[0], parts[1]
		fs, ok := ts.(*taskstore.FSStore)
		if !ok {
			return nil
		}
		agents, err := fs.ListAgents(project)
		if err != nil || !slices.Contains(agents, agent) {
			return fmt.Errorf("agent %q not found in project %q (hint: check --dir workspace or verify the project/agent exists)", agent, project)
		}
		return nil
	}
	// Bare name — check if it's a known agent in any project
	fs, ok := ts.(*taskstore.FSStore)
	if !ok {
		return nil
	}
	projects, err := fs.ListProjects()
	if err != nil {
		return err
	}
	for _, project := range projects {
		agents, err := fs.ListAgents(project)
		if err != nil {
			continue
		}
		if slices.Contains(agents, identity) {
			return fmt.Errorf("%s %q is an agent in project %q; did you mean --%s %s/%s? or use --%s human",
				fieldName, identity, project, fieldName, project, identity, fieldName)
		}
	}
	return fmt.Errorf("unknown %s %q (hint: use 'human' or 'project/agent' format, e.g. cc-connect/pm)", fieldName, identity)
}

func newInboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Manage inbox tasks and async messages",
		Long: `inbox shows all tasks and messages routed to you for review.

Agents route tasks here when they need human input.
Use 'inbox reply' to respond to messages or 'inbox forward' to route elsewhere.`,
	}
	cmd.AddCommand(
		newInboxListCmd(),
		newInboxShowCmd(),
		newInboxRejectCmd(),
		newInboxForwardCmd(),
		// Async message commands (non-blocking, any participant can use these)
		newInboxSendCmd(),
		newInboxMessagesCmd(),
		newInboxReplyCmd(),
		newInboxFwdCmd(),
		newInboxReadCmd(),
		newInboxArchiveCmd(),
		newInboxDeleteCmd(),
	)
	return cmd
}

// ── inbox list ────────────────────────────────────────────────────────────────

func newInboxListCmd() *cobra.Command {
	var (
		recipient string
		unreadOnly bool
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List messages in the inbox",
		Long: `List async messages in the mailbox.

By default shows unread messages for 'human'.
Use --recipient to filter by mailbox (e.g. --recipient cc-connect/pm).
Use --all to show all messages including read ones.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ts := taskstore.New(root)

			recip := recipient
			if recip == "" {
				recip = "human"
			}

			var msgs []*entity.Message
			if unreadOnly {
				msgs, err = ts.ListUnreadMessages(recip)
			} else {
				msgs, err = ts.ListMessages(recip)
			}
			if err != nil {
				return err
			}

			if jsonOut || !isTerminal(os.Stdout) {
				if msgs == nil {
					msgs = []*entity.Message{}
				}
				return printJSON(msgs)
			}

			if len(msgs) == 0 {
				fmt.Printf("No messages for %s.\n", recip)
				return nil
			}

			unread := 0
			for _, m := range msgs {
				if m.ReadAt == nil {
					unread++
				}
			}

			header := fmt.Sprintf("Messages for %s (%d unread, %d total)", recip, unread, len(msgs))
			fmt.Println(header)
			fmt.Println()

			for _, m := range msgs {
				symbol := " "
				if m.ReadAt == nil {
					symbol = "●"
					unread++
				}
				from := m.From
				if m.Subject != "" {
					fmt.Printf("%s [%s] From: %s → To: %s — %s\n", symbol, formatInboxTime(m.SentAt), from, m.To, m.Subject)
				} else {
					preview := m.Body
					if len(preview) > 60 {
						preview = preview[:57] + "..."
					}
					preview = strings.ReplaceAll(preview, "\n", " ")
					fmt.Printf("%s [%s] From: %s → To: %s — %s\n", symbol, formatInboxTime(m.SentAt), from, m.To, preview)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&recipient, "recipient", "", "mailbox to inspect: 'human' (default) or 'project/agent'")
	cmd.Flags().BoolVar(&unreadOnly, "unread-only", false, "show only unread messages")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

// ── inbox show ────────────────────────────────────────────────────────────────

func newInboxShowCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
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

			// Fetch the original task prompt for enriched output.
			project, agentName, _ := resolveTaskOwner(root, taskID)
			var taskPrompt string
			if project != "" {
				if t, err2 := taskstore.New(root).GetTask(project, agentName, taskID); err2 == nil {
					taskPrompt = t.Prompt
				}
			}

			if jsonOut {
				out := map[string]any{
					"inbox_item":  found,
					"task_prompt": taskPrompt,
				}
				return printJSON(out)
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

			if len(found.ActionItems) > 0 {
				fmt.Println("── Action items (what you need to do) ───────────────────────")
				for i, item := range found.ActionItems {
					fmt.Printf("  %d. %s\n", i+1, item)
				}
				fmt.Println()
			}

			if taskPrompt != "" {
				// For wakeup tasks the prompt is the routine trigger — not useful to
				// the human deciding on the confirmation.  Show a short excerpt instead.
				isWakeup := found.Title == "[wakeup] routine" ||
					strings.HasPrefix(found.Title, "[wakeup]")
				maxLines := 12
				lines := strings.Split(strings.TrimSpace(taskPrompt), "\n")
				if isWakeup && len(lines) > maxLines {
					fmt.Println("── Original task (wakeup trigger — truncated) ───────────────")
					for _, l := range lines[:maxLines] {
						fmt.Println(l)
					}
					fmt.Printf("  … (%d more lines, this is the wakeup routine prompt)\n", len(lines)-maxLines)
				} else {
					fmt.Println("── Original task (full prompt) ──────────────────────────────")
					fmt.Println(taskPrompt)
				}
				fmt.Println()
			}

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

			fmt.Println("── Available actions ────────────────────────────────────────")
			fmt.Printf("  agencycli --dir %s inbox reply   %s --body \"your reply\"\n", root, taskID)
			fmt.Printf("  agencycli --dir %s inbox forward %s --to <project>/<agent> --note \"...\"\n", root, taskID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
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
			fmt.Printf("  Original task %s has been archived.\n", taskID)
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

// ── inbox send ────────────────────────────────────────────────────────────────

func newInboxSendCmd() *cobra.Command {
	var (
		to      []string
		subject string
		body    string
		replyTo string
		from    string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send an async message to one or more recipients",
		Long: `Send a non-blocking async message to any participant.
Supports group send by repeating --to.

Recipient format:
  --to human                 → agency owner's inbox
  --to cc-connect/pm         → project cc-connect, agent pm
  --to cc-connect/dev-claude → project cc-connect, agent dev-claude

Examples:
  # Single recipient
  agencycli inbox send --to cc-connect/pm --body "..."

  # Group send (repeat --to)
  agencycli inbox send --to cc-connect/pm --to cc-connect/dev-claude --to human --body "..."`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if body == "" {
				return fmt.Errorf("--body is required")
			}
			if len(to) == 0 {
				return fmt.Errorf("--to is required")
			}
			if from == "" {
				return fmt.Errorf("--from is required")
			}
			sender := from
			ts := taskstore.New(root)

			// Validate from and to identities exist
			if err := validateIdentity(ts, sender, "from"); err != nil {
				return err
			}
			for _, recipient := range to {
				if err := validateIdentity(ts, recipient, "to"); err != nil {
					return err
				}
			}

			sentAt := time.Now().UTC()
			for _, recipient := range to {
				msg := &entity.Message{
					ID:      entity.NewMessageID(),
					From:    sender,
					To:      recipient,
					Subject: subject,
					Body:    body,
					ReplyTo: replyTo,
					SentAt:  sentAt,
				}
				if err := ts.SendMessage(msg); err != nil {
					return fmt.Errorf("send to %s: %w", recipient, err)
				}
				fmt.Printf("✓ Message sent  [%s]\n", msg.ID)
				fmt.Printf("  To      : %s\n", msg.To)
				if msg.Subject != "" {
					fmt.Printf("  Subject : %s\n", msg.Subject)
				}
			}
			fmt.Printf("  From    : %s\n", sender)
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&to, "to", nil, "recipient: 'human' or 'project/agent' (repeatable for group send)")
	cmd.Flags().StringVar(&subject, "subject", "", "optional subject line")
	cmd.Flags().StringVar(&body, "body", "", "message body")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "ID of message being replied to")
	cmd.Flags().StringVar(&from, "from", "", "sender identity: 'human' or 'project/agent'")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

// ── inbox messages ────────────────────────────────────────────────────────────

func newInboxMessagesCmd() *cobra.Command {
	var (
		recipient string
		from      string
		all       bool
		archived  bool
		mark      bool
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "messages",
		Short: "List async messages (from agents or other humans)",
		Long: `List async messages delivered to a mailbox.

By default shows only unread messages for 'human'.
Use --recipient to inspect an agent's mailbox.
Use --from to filter by sender (e.g. --from cc-connect/pm).
Use --all to show all messages including already-read ones.
Use --archived to show archived messages.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if recipient == "" {
				recipient = "human"
			}
			ts := taskstore.New(root)
			if err := validateRecipient(ts, recipient); err != nil {
				return err
			}
			var msgs []*entity.Message
			if archived {
				allMsgs, e := ts.ListAllMessages(recipient)
				if e != nil {
					return e
				}
				for _, m := range allMsgs {
					if m.ArchivedAt != nil {
						msgs = append(msgs, m)
					}
				}
			} else if all {
				msgs, err = ts.ListMessages(recipient)
			} else {
				msgs, err = ts.ListUnreadMessages(recipient)
			}
			if err != nil {
				return err
			}

			if from != "" {
				filtered := msgs[:0]
				for _, m := range msgs {
					if m.From == from {
						filtered = append(filtered, m)
					}
				}
				msgs = filtered
			}

			if jsonOut || !isTerminal(os.Stdout) {
				if msgs == nil {
					msgs = []*entity.Message{}
				}
				return printJSON(msgs)
			}

			if len(msgs) == 0 {
				if from != "" {
					fmt.Printf("No messages for %s from %s.\n", recipient, from)
				} else if all {
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
				fmt.Printf("  To      : %s\n", m.To)
				if m.Subject != "" {
					fmt.Printf("  Subject : %s\n", m.Subject)
				}
				if m.ReplyTo != "" {
					fmt.Printf("  Reply-to: %s\n", m.ReplyTo)
				}
				fmt.Printf("\n  %s\n\n", strings.ReplaceAll(m.Body, "\n", "\n  "))
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
	cmd.Flags().StringVar(&from, "from", "", "filter by sender: 'human' or 'project/agent' (e.g. cc-connect/pm)")
	cmd.Flags().BoolVar(&all, "all", false, "show all messages including already-read ones")
	cmd.Flags().BoolVar(&archived, "archived", false, "show only archived messages")
	cmd.Flags().BoolVar(&mark, "mark-read", false, "mark displayed messages as read after listing")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
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

			// Validate --from if provided (empty means "human" which is always valid)
			if from != "" {
				if err := validateIdentity(ts, from, "from"); err != nil {
					return err
				}
			}

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
				sender = original.To
			}
			reply := &entity.Message{
				ID:      entity.NewMessageID(),
				From:    sender,
				To:      original.From,
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

// ── inbox fwd ─────────────────────────────────────────────────────────────────

func newInboxFwdCmd() *cobra.Command {
	var (
		to        []string
		note      string
		from      string
		recipient string
	)

	cmd := &cobra.Command{
		Use:     "fwd <msg-id>",
		Aliases: []string{"forward-message", "forward-msg"},
		Short:   "Forward a message to one or more recipients",
		Args:    cobra.ExactArgs(1),
		Long: `Forward an async message to one or more recipients.
The forwarded message includes the original content and an optional note.
Supports group forward by repeating --to.

  agencycli inbox fwd msg-20260317-abc123 --to cc-connect/dev-claude --note "FYI"
  agencycli inbox fwd msg-20260317-abc123 --to cc-connect/pm --to cc-connect/qa-reviewer`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if len(to) == 0 {
				return fmt.Errorf("--to is required")
			}
			msgID := args[0]
			if recipient == "" {
				recipient = "human"
			}
			sender := from
			if sender == "" {
				sender = "human"
			}

			// Find the original message across all mailboxes.
			ts := taskstore.New(root)

			// Validate identities if provided
			if from != "" {
				if err := validateIdentity(ts, from, "from"); err != nil {
					return err
				}
			}
			for _, r := range to {
				if err := validateIdentity(ts, r, "to"); err != nil {
					return err
				}
			}

			var original *entity.Message
			searchBoxes := []string{recipient}
			// If recipient not specified (human), also search agent mailboxes.
			projects, _ := ts.ListProjects()
			for _, proj := range projects {
				agents, _ := ts.ListAgents(proj)
				for _, ag := range agents {
					searchBoxes = append(searchBoxes, proj+"/"+ag)
				}
			}
			for _, box := range searchBoxes {
				all, _ := ts.ListAllMessages(box)
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
				return fmt.Errorf("message %q not found", msgID)
			}

			// Build forwarded body.
			fwdBody := fmt.Sprintf("---------- Forwarded message ----------\n"+
				"From    : %s\n"+
				"Subject : %s\n\n"+
				"%s",
				original.From,
				original.Subject,
				original.Body,
			)
			if note != "" {
				fwdBody = note + "\n\n" + fwdBody
			}

			subject := original.Subject
			if subject != "" && !strings.HasPrefix(subject, "Fwd: ") {
				subject = "Fwd: " + subject
			}

			sentAt := time.Now().UTC()
			for _, r := range to {
				msg := &entity.Message{
					ID:      entity.NewMessageID(),
					From:    sender,
					To:      r,
					Subject: subject,
					Body:    fwdBody,
					ReplyTo: msgID,
					SentAt:  sentAt,
				}
				if err := ts.SendMessage(msg); err != nil {
					return fmt.Errorf("forward to %s: %w", r, err)
				}
				fmt.Printf("✓ Forwarded [%s] → %s\n", msg.ID, r)
			}
			fmt.Printf("  Original: %s\n", msgID)
			fmt.Printf("  From    : %s\n", sender)
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&to, "to", nil, "recipient(s): 'human' or 'project/agent' (repeatable)")
	cmd.Flags().StringVar(&note, "note", "", "optional note prepended to the forwarded message")
	cmd.Flags().StringVar(&from, "from", "", "override sender (defaults to 'human')")
	cmd.Flags().StringVar(&recipient, "recipient", "", "your mailbox where the original message lives (default: human)")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

// ── inbox read ────────────────────────────────────────────────────────────────

func newInboxReadCmd() *cobra.Command {
	var recipient string

	cmd := &cobra.Command{
		Use:   "read <msg-id>",
		Short: "Mark a message as read",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if recipient == "" {
				recipient = "human"
			}
			ts := taskstore.New(root)
			if err := validateRecipient(ts, recipient); err != nil {
				return err
			}
			if err := ts.MarkMessageRead(recipient, args[0]); err != nil {
				return err
			}
			fmt.Printf("✓ Message %s marked as read\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&recipient, "recipient", "", "mailbox: 'human' (default) or 'project/agent'")
	return cmd
}

// ── inbox archive ─────────────────────────────────────────────────────────────

func newInboxArchiveCmd() *cobra.Command {
	var recipient string

	cmd := &cobra.Command{
		Use:   "archive <msg-id>",
		Short: "Archive a message (hidden from normal listing)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if recipient == "" {
				recipient = "human"
			}
			ts := taskstore.New(root)
			if err := validateRecipient(ts, recipient); err != nil {
				return err
			}
			if err := ts.ArchiveMessage(recipient, args[0]); err != nil {
				return err
			}
			fmt.Printf("✓ Message %s archived\n", args[0])
			fmt.Printf("  View archived messages: agencycli inbox messages --archived\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&recipient, "recipient", "", "mailbox: 'human' (default) or 'project/agent'")
	return cmd
}

// ── inbox delete ──────────────────────────────────────────────────────────────

func newInboxDeleteCmd() *cobra.Command {
	var recipient string

	cmd := &cobra.Command{
		Use:     "delete <msg-id>",
		Aliases: []string{"rm"},
		Short:   "Permanently delete a message",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if recipient == "" {
				recipient = "human"
			}
			ts := taskstore.New(root)
			if err := validateRecipient(ts, recipient); err != nil {
				return err
			}
			if err := ts.DeleteMessage(recipient, args[0]); err != nil {
				return err
			}
			fmt.Printf("✓ Message %s deleted\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&recipient, "recipient", "", "mailbox: 'human' (default) or 'project/agent'")
	return cmd
}

// printJSON marshals v to indented JSON and writes it to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// formatInboxTime returns a local "MM-DD HH:MM" string for display.
func formatInboxTime(t time.Time) string {
	return t.Local().Format("01-02 15:04")
}
