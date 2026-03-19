---
name: agencycli-usage
description: Operate the agencycli tool — create agencies, teams, projects, and hire AI agents into working directories.
---

# Skill: agencycli Usage

`agencycli` is the CLI tool that builds and manages this agency. Use it to manage teams, projects, agents, tasks, and inter-agent communication.

## Workspace

The current agency workspace is at: `$AGENCY_DIR`

All commands auto-discover the workspace when run from inside it, or use `--dir`:
```bash
agencycli --dir $AGENCY_DIR <command>
```

---

## Discover what exists

```bash
agencycli --dir $AGENCY_DIR list teams      # all teams
agencycli --dir $AGENCY_DIR list projects   # all projects
agencycli --dir $AGENCY_DIR list agents     # all agents across all projects
agencycli --dir $AGENCY_DIR list skills     # available skills

agencycli --dir $AGENCY_DIR show team engineering
agencycli --dir $AGENCY_DIR show project cc-connect
agencycli --dir $AGENCY_DIR show agent cc-connect pm
agencycli --dir $AGENCY_DIR show agent cc-connect pm --raw  # full merged context
```

---

## Tasks

### Add a task for an agent

```bash
agencycli --dir $AGENCY_DIR task add \
  --project <project> --agent <agent> \
  --title "Task title" \
  --prompt "Detailed instructions..." \
  --priority <0-3>   # 0=critical 1=high 2=normal(default) 3=low
```

### View the task queue (sorted by priority)

```bash
agencycli --dir $AGENCY_DIR task list --project <project> --agent <agent>
agencycli --dir $AGENCY_DIR task list --project <project> --agent <agent> --status pending
```

### Control tasks

```bash
agencycli --dir $AGENCY_DIR task cancel <task-id>
agencycli --dir $AGENCY_DIR task retry  <task-id>

# Emergency halt — cancel all pending (and optionally running) tasks
agencycli --dir $AGENCY_DIR task stop-all \
  --project <project> --all-agents
agencycli --dir $AGENCY_DIR task stop-all \
  --project <project> --agent <agent> --include-running
```

### View token usage and cost

```bash
# One agent
agencycli --dir $AGENCY_DIR task tokens \
  --project <project> --agent <agent>

# All agents in a project
agencycli --dir $AGENCY_DIR task tokens \
  --project <project> --all-agents

# Specific task
agencycli --dir $AGENCY_DIR task tokens \
  --project <project> --agent <agent> --task <task-id>
```

### Task done / confirm-request (called by agents inside their prompt)

```bash
# Report completion
agencycli --dir $AGENCY_DIR task done \
  --id $TASK_ID --status success --summary "Brief description of what was done"

# Report failure
agencycli --dir $AGENCY_DIR task done \
  --id $TASK_ID --status failed --error "reason"

# Pause and request human confirmation (blocks until human responds)
agencycli --dir $AGENCY_DIR task confirm-request \
  --id $TASK_ID \
  --summary "PR #42 ready, awaiting merge approval" \
  --action-item "Review: gh pr view 42 --repo org/repo" \
  --action-item "Confirm merge: reply 'merge'" \
  --action-item "Hold: reply 'hold <reason>'"
```

After `confirm-request`, the task is paused. The human uses `inbox confirm/reject` to resume it. The human's reply text is available as `$CONFIRMATION_REPLY` in the re-run.

---

## Inbox — task confirmations (blocking)

When an agent calls `task confirm-request`, the task appears here for the human to resolve.

```bash
agencycli --dir $AGENCY_DIR inbox list
agencycli --dir $AGENCY_DIR inbox show    <task-id>
agencycli --dir $AGENCY_DIR inbox confirm <task-id> --message "yes, proceed"
agencycli --dir $AGENCY_DIR inbox reject  <task-id> --reason "out of scope"
agencycli --dir $AGENCY_DIR inbox comment <task-id> --message "check the edge case first"
agencycli --dir $AGENCY_DIR inbox forward <task-id> \
  --to <project>/<agent> --note "please double-check the auth flow"
```

---

## Inbox — async messages (non-blocking)

Any participant (human or agent) can send non-blocking messages to any other participant. The recipient reads them on their next wakeup — the scheduler automatically injects unread messages at the top of the wakeup prompt.

### Participant address format
- Human: `human`
- Agent: `<project>/<agent>` — e.g. `cc-connect/pm`, `cc-connect/dev-claude`

### Send a message

```bash
# Human → agent
agencycli --dir $AGENCY_DIR inbox send \
  --to <project>/<agent> \
  --subject "Subject" \
  --body "Body"

# Agent → human
agencycli --dir $AGENCY_DIR inbox send \
  --from <project>/<agent> --to human \
  --subject "Subject" --body "Body"

# Agent → agent
agencycli --dir $AGENCY_DIR inbox send \
  --from <project>/pm --to <project>/dev \
  --subject "Extra context for task <id>" \
  --body "Details..."

# Group send (repeat --to for multiple recipients)
agencycli --dir $AGENCY_DIR inbox send \
  --from <project>/pm \
  --to <project>/dev --to <project>/qa --to human \
  --subject "Sprint kick-off" --body "..."
```

### Read messages

```bash
agencycli --dir $AGENCY_DIR inbox messages                          # human's unread
agencycli --dir $AGENCY_DIR inbox messages --recipient <project>/pm # agent's mailbox
agencycli --dir $AGENCY_DIR inbox messages --from <project>/qa      # filter by sender
agencycli --dir $AGENCY_DIR inbox messages --all                    # include read
agencycli --dir $AGENCY_DIR inbox messages --archived               # archived only
agencycli --dir $AGENCY_DIR inbox messages --mark-read              # mark all read
```

### Reply / Forward

```bash
agencycli --dir $AGENCY_DIR inbox reply <msg-id> --from <address> --body "..."
agencycli --dir $AGENCY_DIR inbox fwd   <msg-id> --to <address> --note "..."
```

### Per-message status

```bash
agencycli --dir $AGENCY_DIR inbox read    <msg-id>
agencycli --dir $AGENCY_DIR inbox archive <msg-id>
agencycli --dir $AGENCY_DIR inbox delete  <msg-id>
agencycli --dir $AGENCY_DIR inbox rm      <msg-id>   # alias for delete
```

---

## Heartbeat scheduler

```bash
agencycli --dir $AGENCY_DIR scheduler start
agencycli --dir $AGENCY_DIR scheduler stop
agencycli --dir $AGENCY_DIR scheduler status

# Configure an agent's heartbeat
agencycli --dir $AGENCY_DIR scheduler heartbeat \
  --project <project> --agent <agent> \
  --enable --interval 30m \
  --active-hours "09:00-20:00" \
  --active-days weekdays

# Set or update the wakeup routine (runs when task queue is empty)
agencycli --dir $AGENCY_DIR scheduler heartbeat \
  --project <project> --agent <agent> \
  --wakeup-prompt-file projects/<project>/agents/<agent>/wakeup.md
```

When the scheduler wakes an agent:
1. If **pending tasks** exist → runs the highest-priority task
2. If queue is **empty** and a wakeup routine is set → runs `wakeup.md` as a synthetic task
3. Any **unread messages** for the agent are auto-prepended to the prompt

---

## Cron jobs

```bash
agencycli --dir $AGENCY_DIR cron add \
  --project <project> --agent <agent> \
  --title "Weekly backlog review" \
  --schedule "0 9 * * 1" \
  --prompt "Review the backlog and reprioritise for the week..."

agencycli --dir $AGENCY_DIR cron list    --project <project> --agent <agent>
agencycli --dir $AGENCY_DIR cron disable <cron-id> --project <project> --agent <agent>
```

---

## Context sync

```bash
agencycli --dir $AGENCY_DIR sync                                # all agents with changed context
agencycli --dir $AGENCY_DIR sync --project <project>           # one project
agencycli --dir $AGENCY_DIR sync --project <project> --name <agent>  # one agent
agencycli --dir $AGENCY_DIR sync --force                        # force regenerate everything
```

---

## Context Inheritance

Every hired agent automatically receives context in this order:
1. **agency-prompt.md** — global rules, values
2. **teams/\<team\>/prompt.md** — team-specific standards
3. **teams/\<team\>/roles/\<role\>/prompt.md** — role responsibilities
4. **projects/\<project\>/prompt.md** — project background and tech stack
5. **skills** — all skills listed in the team's `team.yaml` and role's `role.yaml`
