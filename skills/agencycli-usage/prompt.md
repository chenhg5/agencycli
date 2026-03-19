# Skill: agencycli Usage

`agencycli` is the CLI tool that builds and manages this agency. Use it to manage teams, projects, agents, tasks, and inter-agent communication.

## Workspace

The current agency workspace is at: `/root/code/TechStudio`

All commands auto-discover the workspace when run from inside it, or use `--dir`:
```bash
agencycli --dir /root/code/TechStudio <command>
```

---

## Discover what exists

```bash
agencycli --dir /root/code/TechStudio list teams      # all teams
agencycli --dir /root/code/TechStudio list projects   # all projects
agencycli --dir /root/code/TechStudio list agents     # all agents across all projects
agencycli --dir /root/code/TechStudio list skills     # available skills

agencycli --dir /root/code/TechStudio show team engineering
agencycli --dir /root/code/TechStudio show project cc-connect
agencycli --dir /root/code/TechStudio show agent cc-connect pm
agencycli --dir /root/code/TechStudio show agent cc-connect pm --raw  # full merged context
```

---

## Tasks

### Add a task for an agent

```bash
agencycli --dir /root/code/TechStudio task add \
  --project <project> --agent <agent> \
  --title "Task title" \
  --prompt "Detailed instructions..." \
  --priority <0-3>   # 0=critical 1=high 2=normal(default) 3=low
```

### View the task queue (sorted by priority)

```bash
agencycli --dir /root/code/TechStudio task list --project cc-connect --agent dev-claude
agencycli --dir /root/code/TechStudio task list --project cc-connect --agent pm --status pending
```

### Control tasks

```bash
agencycli --dir /root/code/TechStudio task cancel <task-id>
agencycli --dir /root/code/TechStudio task retry  <task-id>

# Emergency halt — cancel all pending (and optionally running) tasks
agencycli --dir /root/code/TechStudio task stop-all \
  --project cc-connect --all-agents
agencycli --dir /root/code/TechStudio task stop-all \
  --project cc-connect --agent dev-claude --include-running
```

### View token usage and cost

```bash
# One agent
agencycli --dir /root/code/TechStudio task tokens \
  --project cc-connect --agent dev-claude

# All agents in a project
agencycli --dir /root/code/TechStudio task tokens \
  --project cc-connect --all-agents

# Specific task
agencycli --dir /root/code/TechStudio task tokens \
  --project cc-connect --agent dev-claude --task <task-id>
```

### Task done / confirm-request (called by agents inside their prompt)

```bash
# Report completion
agencycli --dir /root/code/TechStudio task done \
  --id $TASK_ID --status success --summary "Brief description of what was done"

# Report failure
agencycli --dir /root/code/TechStudio task done \
  --id $TASK_ID --status failed --error "reason"

# Pause and request human confirmation (blocks until human responds)
agencycli --dir /root/code/TechStudio task confirm-request \
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
agencycli --dir /root/code/TechStudio inbox list
agencycli --dir /root/code/TechStudio inbox show    <task-id>
agencycli --dir /root/code/TechStudio inbox confirm <task-id> --message "yes, proceed"
agencycli --dir /root/code/TechStudio inbox reject  <task-id> --reason "out of scope"
agencycli --dir /root/code/TechStudio inbox comment <task-id> --message "check the edge case first"
agencycli --dir /root/code/TechStudio inbox forward <task-id> \
  --to cc-connect/qa-reviewer --note "please double-check the auth flow"
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
agencycli --dir /root/code/TechStudio inbox send \
  --to cc-connect/pm \
  --subject "Prioritise issue #55" \
  --body "This is blocking a customer. Move to P0."

# Agent → human
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm --to human \
  --subject "Backlog updated" --body "Added 2 new issues. FYI."

# Agent → agent
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm --to cc-connect/dev-claude \
  --subject "Extra context for task <id>" \
  --body "Only reproduces with UTF-8 filenames."

# Group send (repeat --to for multiple recipients)
agencycli --dir /root/code/TechStudio inbox send \
  --from cc-connect/pm \
  --to cc-connect/dev-claude --to cc-connect/qa-reviewer --to human \
  --subject "Sprint kick-off" --body "New sprint starts Monday..."
```

### Read messages

```bash
# Human's unread messages (default)
agencycli --dir /root/code/TechStudio inbox messages

# An agent's mailbox
agencycli --dir /root/code/TechStudio inbox messages --recipient cc-connect/pm

# Filter by sender
agencycli --dir /root/code/TechStudio inbox messages --from cc-connect/qa-reviewer
agencycli --dir /root/code/TechStudio inbox messages --recipient cc-connect/pm --from human

# All messages including already-read
agencycli --dir /root/code/TechStudio inbox messages --all

# Show archived messages
agencycli --dir /root/code/TechStudio inbox messages --archived

# Mark all as read after listing
agencycli --dir /root/code/TechStudio inbox messages --mark-read
```

### Reply to a message

```bash
agencycli --dir /root/code/TechStudio inbox reply <msg-id> \
  --from cc-connect/pm \
  --body "Acknowledged, will handle on next wakeup."
```

### Forward a message

```bash
# Forward to a single recipient
agencycli --dir /root/code/TechStudio inbox fwd <msg-id> \
  --to cc-connect/dev-claude

# Forward to multiple recipients with a note
agencycli --dir /root/code/TechStudio inbox fwd <msg-id> \
  --to cc-connect/pm --to cc-connect/qa-reviewer \
  --note "Please review and coordinate."

# Agent forwarding (specify --from)
agencycli --dir /root/code/TechStudio inbox fwd <msg-id> \
  --from cc-connect/pm --to human --note "Escalating to you."
```

### Per-message status management

```bash
# Mark a single message as read
agencycli --dir /root/code/TechStudio inbox read <msg-id>

# Archive (hides from normal listing, still retrievable with --archived)
agencycli --dir /root/code/TechStudio inbox archive <msg-id>

# Permanently delete
agencycli --dir /root/code/TechStudio inbox delete <msg-id>
agencycli --dir /root/code/TechStudio inbox rm     <msg-id>   # alias

# All status commands accept --recipient to target an agent's mailbox
agencycli --dir /root/code/TechStudio inbox read <msg-id> --recipient cc-connect/pm
```

---

## Heartbeat scheduler

```bash
# Start scheduler (aliases: sched, s)
agencycli --dir /root/code/TechStudio scheduler start
agencycli --dir /root/code/TechStudio scheduler stop
agencycli --dir /root/code/TechStudio scheduler status

# Configure an agent's heartbeat
agencycli --dir /root/code/TechStudio scheduler heartbeat \
  --project cc-connect --agent pm \
  --enable --interval 30m \
  --active-hours "09:00-20:00" \
  --active-days weekdays

# Set or update the wakeup routine (runs when task queue is empty)
agencycli --dir /root/code/TechStudio scheduler heartbeat \
  --project cc-connect --agent pm \
  --wakeup-prompt-file /root/code/TechStudio/projects/cc-connect/agents/pm/wakeup.md
```

When the scheduler wakes an agent:
1. If **pending tasks** exist → runs the highest-priority task
2. If queue is **empty** and a wakeup routine is set → runs `wakeup.md` as a synthetic task
3. Any **unread messages** for the agent are auto-prepended to the prompt

---

## Cron jobs

```bash
agencycli --dir /root/code/TechStudio cron add \
  --project cc-connect --agent pm \
  --title "Weekly backlog review" \
  --schedule "0 9 * * 1" \
  --prompt "Review the backlog and reprioritise for the week..."

agencycli --dir /root/code/TechStudio cron list --project cc-connect --agent pm
agencycli --dir /root/code/TechStudio cron disable <cron-id> --project cc-connect --agent pm
```

---

## Workspace File Structure

```
TechStudio/
├── .agencycli/
│   ├── agency.yaml          ← agency metadata
│   ├── inbox.yaml           ← human task-confirmation inbox
│   ├── inbox.md             ← human-readable inbox summary
│   └── messages.yaml        ← async messages for human
├── agency-prompt.md         ← global rules for ALL agents
├── teams/
│   └── engineering/
│       ├── team.yaml        ← name, description, skills list
│       ├── prompt.md        ← team context
│       └── roles/
│           └── developer/
│               ├── role.yaml   ← skills[], setup dirs
│               └── prompt.md   ← role context
├── skills/
│   └── github-pr-review/
│       ├── skill.yaml
│       └── prompt.md
├── agent-playbooks/         ← wakeup.md templates distributed with the agency
│   ├── pm.md
│   └── qa-reviewer.md
├── project-blueprints/      ← project templates
│   └── cc-connect.yaml
└── projects/
    └── cc-connect/
        ├── project.yaml
        ├── prompt.md
        └── agents/
            └── pm/
                ├── CLAUDE.md        ← merged context (@imports all layers)
                ├── wakeup.md        ← autonomous routine (installed by project apply)
                ├── tasks.yaml       ← active task queue
                ├── heartbeat.yaml   ← heartbeat config
                ├── messages.yaml    ← async messages for this agent
                ├── runs/            ← execution logs
                └── .claude/skills/ ← deployed skill files
```

---

## Context Inheritance

Every hired agent automatically receives context in this order:
1. **agency-prompt.md** — global rules, values
2. **teams/\<team\>/prompt.md** — team-specific standards
3. **teams/\<team\>/roles/\<role\>/prompt.md** — role responsibilities
4. **projects/\<project\>/prompt.md** — project background and tech stack
5. **skills** — all skills listed in the team's `team.yaml` and role's `role.yaml`

Edit any of these files and run `agencycli sync` to propagate changes.

---

## Manage context

```bash
# Create a new team/role/project
agencycli --dir /root/code/TechStudio create team --name "devops" --desc "..."
agencycli --dir /root/code/TechStudio create role --team devops --name sre --desc "..."
agencycli --dir /root/code/TechStudio create project --name "new-service" --desc "..."

# Hire an agent
agencycli --dir /root/code/TechStudio hire \
  --project new-service --team engineering --role developer \
  --model claudecode --name dev

# Re-sync after editing prompts or skills
agencycli --dir /root/code/TechStudio sync --project cc-connect
agencycli --dir /root/code/TechStudio sync --force  # regenerate everything

# Add/remove skills from a role
agencycli --dir /root/code/TechStudio role skill add \
  --team product --role product-manager --skill agency-messaging
```

---

## Project apply (blueprint → running agents)

```bash
# Apply a project blueprint: hire agents + configure heartbeats + install playbooks
agencycli --dir /root/code/TechStudio project apply --project cc-connect
agencycli --dir /root/code/TechStudio project apply --project cc-connect --dry-run
agencycli --dir /root/code/TechStudio project apply --project cc-connect --force
```
