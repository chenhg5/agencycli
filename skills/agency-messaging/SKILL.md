---
name: agency-messaging
description: Discover all agents in the current agency and send/receive async messages between agents and the human owner via the inbox system.
---

# Skill: Agency Messaging

You can discover every agent in this agency and exchange async messages with them or with the human owner. Messages are non-blocking: the sender continues working immediately, and the recipient reads the message on their next wakeup.

The agency workspace is at: `$AGENCY_DIR`

---

## Discover Agents

```bash
agencycli --dir $AGENCY_DIR list agents
agencycli --dir $AGENCY_DIR show agent <project> <agent>
```

### Recipient address format
- **Human owner**: `human`
- **Any agent**: `<project>/<agent>` — e.g. `cc-connect/pm`, `cc-connect/qa-reviewer`

---

## Send a Message

```bash
# Single recipient
agencycli --dir $AGENCY_DIR inbox send \
  --from <your-address> \
  --to   <recipient-address> \
  --subject "<subject>" \
  --body "<body>"

# Group send — repeat --to for multiple recipients
agencycli --dir $AGENCY_DIR inbox send \
  --from cc-connect/pm \
  --to cc-connect/dev-claude --to cc-connect/qa-reviewer --to human \
  --subject "Sprint kick-off" \
  --body "New sprint starts Monday. See backlog for tasks."
```

**Examples:**

```bash
# PM → dev-claude: extra context
agencycli --dir $AGENCY_DIR inbox send \
  --from cc-connect/pm --to cc-connect/dev-claude \
  --subject "Issue #205 extra context" \
  --body "Only reproduces with UTF-8 filenames. Reproduce: echo '测试' > test.txt"

# PM → human: async progress update
agencycli --dir $AGENCY_DIR inbox send \
  --from cc-connect/pm --to human \
  --subject "Backlog updated" \
  --body "Added 3 new issues (P2). No action needed, just FYI."

# PM → QA: heads-up
agencycli --dir $AGENCY_DIR inbox send \
  --from cc-connect/pm --to cc-connect/qa-reviewer \
  --subject "PR incoming for #205" \
  --body "dev-claude is working on it. Expect a PR within the hour."
```

---

## Reply to a Message

```bash
agencycli --dir $AGENCY_DIR inbox reply <msg-id> \
  --from <your-address> \
  --body "<reply text>"
```

---

## Forward a Message

```bash
# Forward to a single recipient
agencycli --dir $AGENCY_DIR inbox fwd <msg-id> \
  --from <your-address> \
  --to   <recipient-address>

# Forward to multiple recipients with a note
agencycli --dir $AGENCY_DIR inbox fwd <msg-id> \
  --from cc-connect/pm \
  --to cc-connect/dev-claude --to cc-connect/qa-reviewer \
  --note "Please coordinate on this."
```

The forwarded message includes the original sender, subject, and body. Subject is auto-prefixed with `Fwd:`.

---

## Read Messages

```bash
# Your unread messages (also auto-injected into your wakeup prompt)
agencycli --dir $AGENCY_DIR inbox messages --recipient <your-address>

# Filter by sender
agencycli --dir $AGENCY_DIR inbox messages --recipient <your-address> --from human

# All messages including already-read
agencycli --dir $AGENCY_DIR inbox messages --recipient <your-address> --all

# Show archived messages
agencycli --dir $AGENCY_DIR inbox messages --recipient <your-address> --archived

# Mark all as read after listing
agencycli --dir $AGENCY_DIR inbox messages --recipient <your-address> --mark-read
```

---

## Per-Message Status Management

```bash
# Mark a single message as read
agencycli --dir $AGENCY_DIR inbox read <msg-id> --recipient <your-address>

# Archive (hides from normal listing, retrievable with --archived)
agencycli --dir $AGENCY_DIR inbox archive <msg-id> --recipient <your-address>

# Permanently delete
agencycli --dir $AGENCY_DIR inbox delete <msg-id> --recipient <your-address>
agencycli --dir $AGENCY_DIR inbox rm     <msg-id> --recipient <your-address>
```

---

## When to Use Messaging vs. Confirm-Request

| Situation | Use |
|-----------|-----|
| Need human to make a decision before you continue | `task confirm-request` (non-blocking, archived) |
| Sending info or a heads-up, no reply needed immediately | `inbox send` (non-blocking) |
| Coordinating context between agents asynchronously | `inbox send` (non-blocking) |
| Broadcast to multiple participants at once | `inbox send --to A --to B --to C` |
| Forwarding a message to someone else | `inbox fwd` |
| Replying to a message someone sent you | `inbox reply` |

---

## Common PM Messaging Patterns

```bash
# 1. Broadcast sprint kick-off to all agents
agencycli --dir $AGENCY_DIR inbox send \
  --from cc-connect/pm \
  --to cc-connect/dev-claude --to cc-connect/qa-reviewer --to cc-connect/biz-dev \
  --subject "Sprint W14 kick-off" \
  --body "Focus this week: <priorities>. See backlog for assigned tasks."

# 2. Notify dev of approved task
agencycli --dir $AGENCY_DIR inbox send \
  --from cc-connect/pm --to cc-connect/dev-claude \
  --subject "New task approved: <task-title>" \
  --body "Human confirmed. Priority: P<N>. Key context: <notes>"

# 3. Escalate stale task to human
agencycli --dir $AGENCY_DIR inbox send \
  --from cc-connect/pm --to human \
  --subject "Task stale: <task-title>" \
  --body "Task <id> has been in_progress for >2 days. May need intervention."

# 4. Forward a customer report from human to dev
agencycli --dir $AGENCY_DIR inbox fwd <msg-id> \
  --from cc-connect/pm --to cc-connect/dev-claude \
  --note "Customer reported this bug. Please investigate."
```
