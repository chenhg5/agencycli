---
name: task-management
description: Create, assign, monitor, and close tasks across agents. Covers full task lifecycle — add, priority, dependencies, confirm-request, retry, cancel, bulk ops, and cross-agent search.
---

# Skill: Task Management

Use this skill when your role involves planning, delegating, and tracking work across agents. The agency workspace is at `$AGENCY_DIR`.

---

## Priority levels

| Value | Label    | When to use |
|-------|----------|-------------|
| 0     | critical | Blocking production / blocking other agents |
| 1     | high     | Should be picked up in the current cycle |
| 2     | normal   | Default — regular backlog work |
| 3     | low      | Nice-to-have, do when free |

---

## Create a task for an agent

```bash
# Minimal
agencycli --dir $AGENCY_DIR task add \
  --project <project> --agent <agent> \
  --title "Short title" \
  --prompt "Detailed instructions for the agent..."

# With priority and type
agencycli --dir $AGENCY_DIR task add \
  --project <project> --agent <agent> \
  --title "Fix auth bug" \
  --prompt "Reproduce with: curl -X POST /login with empty password. Root cause is in auth/validator.go." \
  --priority 1 \
  --type bug

# With dependencies (task will not start until listed IDs are done)
agencycli --dir $AGENCY_DIR task add \
  --project <project> --agent <agent> \
  --title "Deploy to staging" \
  --prompt "Run: make deploy-staging" \
  --depends-on <task-id-1> --depends-on <task-id-2>
```

### Task types

`feature` · `bug` · `chore` · `review` · `deploy` · `research` · `wakeup` · (custom string)

---

## Inspect task queues

```bash
# All active tasks for one agent (sorted by priority)
agencycli --dir $AGENCY_DIR task list --project <project> --agent <agent>

# Filter by status
agencycli --dir $AGENCY_DIR task list --project <project> --agent <agent> --status pending
agencycli --dir $AGENCY_DIR task list --project <project> --agent <agent> --status in_progress

# Include archived (completed / cancelled) tasks
agencycli --dir $AGENCY_DIR task list --project <project> --agent <agent> --archived

# Show full detail of a specific task (project + agent known)
agencycli --dir $AGENCY_DIR task show <task-id> --project <project> --agent <agent>

# Find a task anywhere by ID (no project/agent needed)
agencycli --dir $AGENCY_DIR task find --id <task-id>
```

---

## Monitor progress across agents

To get an overview of all agents and their queue depths:

```bash
agencycli --dir $AGENCY_DIR overview
```

To iterate agents and check queues programmatically:

```bash
# List all agents in a project
agencycli --dir $AGENCY_DIR list agents --project <project>

# Then per agent
agencycli --dir $AGENCY_DIR task list --project <project> --agent <agent> --status pending
agencycli --dir $AGENCY_DIR task list --project <project> --agent <agent> --status in_progress
```

---

## Mark tasks done (called by the executing agent, or manually)

```bash
# Success
agencycli --dir $AGENCY_DIR task done \
  --id <task-id> --status success --summary "What was accomplished"

# Failure
agencycli --dir $AGENCY_DIR task done \
  --id <task-id> --status failed --error "reason"
```

---

## Request human confirmation (non-blocking)

When an agent needs a human decision before continuing, it calls:

```bash
agencycli --dir $AGENCY_DIR task confirm-request \
  --id $TASK_ID \
  --summary "One-line description of what needs approval" \
  --action-item "Option A: reply 'approve'" \
  --action-item "Option B: reply 'reject <reason>'"
```

The task is archived. The human responds via:

```bash
agencycli --dir $AGENCY_DIR inbox list
agencycli --dir $AGENCY_DIR inbox messages    # view messages
agencycli --dir $AGENCY_DIR inbox reply <msg-id> --body "approve"
agencycli --dir $AGENCY_DIR inbox reject  <task-id> --reason "out of scope"
```

The human's reply is sent as a message. The agent sees it on the next wakeup and continues via session memory.

---

## Escalate or delegate a task

If a task is better suited for a different agent, cancel it and re-create for the target agent:

```bash
# 1. Cancel the original
agencycli --dir $AGENCY_DIR task cancel <task-id> --project <project> --agent <from-agent>

# 2. Create for the new agent with transferred context
agencycli --dir $AGENCY_DIR task add \
  --project <project> --agent <to-agent> \
  --title "<same title>" \
  --prompt "<original prompt + re-delegation note>"
```

To notify the new agent with async context before the task runs:

```bash
agencycli --dir $AGENCY_DIR inbox send \
  --from <project>/pm \
  --to   <project>/<to-agent> \
  --subject "Incoming task: <title>" \
  --body "Extra context: ..."
```

---

## Retry a failed task

```bash
agencycli --dir $AGENCY_DIR task retry <task-id> \
  --project <project> --agent <agent>
```

Optionally update the prompt before retrying:

```bash
agencycli --dir $AGENCY_DIR task retry <task-id> \
  --project <project> --agent <agent> \
  --prompt "Updated instructions based on failure: ..."
```

---

## Cancel tasks

```bash
# Cancel a specific task
agencycli --dir $AGENCY_DIR task cancel <task-id> \
  --project <project> --agent <agent>

# Emergency halt — cancel all pending tasks for all agents in a project
agencycli --dir $AGENCY_DIR task stop-all \
  --project <project> --all-agents

# Cancel pending + in-progress for a single agent
agencycli --dir $AGENCY_DIR task stop-all \
  --project <project> --agent <agent> --include-running
```

---

## Recurring tasks with cron

Schedule a task to be automatically enqueued on a fixed schedule:

```bash
agencycli --dir $AGENCY_DIR cron add \
  --project <project> --agent <agent> \
  --id weekly-review \
  --title "Weekly backlog review" \
  --schedule "0 9 * * 1" \
  --prompt "Review all open issues. Prioritise for the week. Update task queue accordingly."

agencycli --dir $AGENCY_DIR cron list    --project <project> --agent <agent>
agencycli --dir $AGENCY_DIR cron disable weekly-review --project <project> --agent <agent>
agencycli --dir $AGENCY_DIR cron enable  weekly-review --project <project> --agent <agent>
```

Cron syntax: `minute hour day month weekday` (standard 5-field)

---

## Token usage and cost

```bash
# One agent
agencycli --dir $AGENCY_DIR task tokens --project <project> --agent <agent>

# All agents in a project
agencycli --dir $AGENCY_DIR task tokens --project <project> --all-agents

# Specific task
agencycli --dir $AGENCY_DIR task tokens \
  --project <project> --agent <agent> --task <task-id>
```

---

## Task assignment decision guide

| Situation | Action |
|-----------|--------|
| Clear scope, right agent | `task add` directly |
| Scope unclear, need human input first | `inbox send --to human` to clarify, then `task add` |
| Depends on another task being done first | `task add --depends-on <id>` |
| Task failed once | `task retry` with updated prompt |
| Wrong agent received the task | Cancel + re-create for correct agent |
| Repeated recurring work | `cron add` with a schedule |
| Human must approve before agent proceeds | Agent calls `task confirm-request` |
| Need to know which agent owns a task ID | `task find --id <id>` |
