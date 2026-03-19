# Skill: Agency Messaging

You can discover every agent in this agency and exchange async messages with them or with the human owner. Messages are non-blocking: the sender continues working immediately, and the recipient reads the message on their next wakeup.

The agency workspace is at: `/root/code/TechStudio`

---

## Discover Agents

```bash
agencycli --dir /root/code/TechStudio list agents
agencycli --dir /root/code/TechStudio show agent <project> <agent>
```

### Recipient address format
- **Human owner**: `human`
- **Any agent**: `<project>/<agent>` — e.g. `cc-connect/pm`, `cc-connect/qa-reviewer`

---

## Send a Message

```bash
# Single recipient
agencycli --dir /root/code/TechStudio inbox send \
  --from <your-address> \
  --to   <recipient-address> \
  --subject "<subject>" \
  --body "<body>"

# Group send — repeat --to for multiple recipients
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm \
  --to cc-connect/dev-claude --to cc-connect/qa-reviewer --to human \
  --subject "Sprint kick-off" \
  --body "New sprint starts Monday. See backlog for tasks."
```

**Examples:**

```bash
# PM → dev-claude: extra context
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm --to cc-connect/dev-claude \
  --subject "Issue #205 extra context" \
  --body "Only reproduces with UTF-8 filenames. Reproduce: echo '测试' > test.txt"

# PM → human: async progress update
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm --to human \
  --subject "Backlog updated" \
  --body "Added 3 new issues (P2). No action needed, just FYI."

# PM → QA: heads-up
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm --to cc-connect/qa-reviewer \
  --subject "PR incoming for #205" \
  --body "dev-claude is working on it. Expect a PR within the hour."
```

---

## Reply to a Message

```bash
agencycli --dir /root/code/TechStudio inbox reply <msg-id> \
  --from <your-address> \
  --body "<reply text>"
```

---

## Forward a Message

```bash
# Forward to a single recipient
agencycli --dir /root/code/TechStudio inbox fwd <msg-id> \
  --from <your-address> \
  --to   <recipient-address>

# Forward to multiple recipients with a note
agencycli --dir /root/code/TechStudio inbox fwd <msg-id> \
  --from cc-connect/pm \
  --to cc-connect/dev-claude --to cc-connect/qa-reviewer \
  --note "Please coordinate on this."
```

The forwarded message includes the original sender, subject, and body. Subject is auto-prefixed with `Fwd:`.

---

## Read Messages

```bash
# Your unread messages (also auto-injected into your wakeup prompt)
agencycli --dir /root/code/TechStudio inbox messages --recipient <your-address>

# Filter by sender
agencycli --dir /root/code/TechStudio inbox messages --recipient <your-address> --from human

# All messages including already-read
agencycli --dir /root/code/TechStudio inbox messages --recipient <your-address> --all

# Show archived messages
agencycli --dir /root/code/TechStudio inbox messages --recipient <your-address> --archived

# Mark all as read after listing
agencycli --dir /root/code/TechStudio inbox messages --recipient <your-address> --mark-read
```

---

## Per-Message Status Management

```bash
# Mark a single message as read
agencycli --dir /root/code/TechStudio inbox read <msg-id> --recipient <your-address>

# Archive (hides from normal listing, retrievable with --archived)
agencycli --dir /root/code/TechStudio inbox archive <msg-id> --recipient <your-address>

# Permanently delete
agencycli --dir /root/code/TechStudio inbox delete <msg-id> --recipient <your-address>
agencycli --dir /root/code/TechStudio inbox rm     <msg-id> --recipient <your-address>
```

---

## When to Use Messaging vs. Confirm-Request

| Situation | Use |
|-----------|-----|
| Need human to make a decision before you continue | `task confirm-request` (blocks current task) |
| Sending info or a heads-up, no reply needed immediately | `inbox send` (non-blocking) |
| Coordinating context between agents asynchronously | `inbox send` (non-blocking) |
| Broadcast to multiple participants at once | `inbox send --to A --to B --to C` |
| Forwarding a message to someone else | `inbox fwd` |
| Replying to a message someone sent you | `inbox reply` |

---

## Common PM Messaging Patterns

```bash
# 1. Broadcast sprint kick-off to all agents
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm \
  --to cc-connect/dev-claude --to cc-connect/qa-reviewer --to cc-connect/biz-dev \
  --subject "Sprint W14 kick-off" \
  --body "Focus this week: <priorities>. See backlog for assigned tasks."

# 2. Notify dev of approved task
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm --to cc-connect/dev-claude \
  --subject "New task approved: <task-title>" \
  --body "Human confirmed. Priority: P<N>. Key context: <notes>"

# 3. Escalate stale task to human
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm --to human \
  --subject "Task stale: <task-title>" \
  --body "Task <id> has been in_progress for >2 days. May need intervention."

# 4. Forward a customer report from human to dev
agencycli --dir /root/code/TechStudio inbox fwd <msg-id> \
  --from cc-connect/pm --to cc-connect/dev-claude \
  --note "Customer reported this bug. Please investigate."
```
