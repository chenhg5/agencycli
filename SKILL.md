---
name: agencycli
description: "Manage AI agent teams with agencycli — a CLI tool for organising AI agents (Claude Code, Codex, Gemini, Cursor, etc.) into hierarchical teams (Agency → Team → Role → Project → Agent). Key capabilities: create agencies and teams, hire agents with merged context layers, assign and run tasks with priority queues, manage autonomous playbooks (wakeup.md), send async inbox messages between human and agents, configure heartbeat schedules and cron jobs, run agents inside Docker sandboxes, forward/confirm tasks via inbox, manage templates, and more. Use this skill whenever you need to: create or manage an agencycli workspace, hire/fire/sync agents, add/run/cancel tasks, check inbox confirmations, send messages, configure heartbeats or crons, start the scheduler, or work with agency templates."
---

# agencycli

`agencycli` is a CLI tool for organising AI agents into hierarchical teams (Agency → Team → Role → Project → Agent). Each agent = Model + Context + Skills. The binary is at `/usr/local/bin/agencycli` or in `$PATH`.

## Installation

```bash
# via npm (recommended)
npm install -g @agencycli/agentctl

# via install script
curl -fsSL https://raw.githubusercontent.com/chenhg5/agencycli/main/scripts/install.sh | sh

# verify
agencycli version
```

## Core concepts

| Concept | What it is |
|---------|-----------|
| **Agency** | The root workspace (`agency-prompt.md`, `.agencycli/agency.yaml`) |
| **Team** | A group with a shared prompt (`teams/<team>/prompt.md`) |
| **Role** | A position within a team — has its own prompt + bound skills + workspace setup |
| **Project** | A work unit with its own prompt, linked to a code repo |
| **Agent** | A hired instance: model + merged context from all layers |
| **Skill** | Reusable instructions copied into agent workspace on hire |
| **Task** | A unit of work with status, priority (0=critical…3=low), prompt, and optional heartbeat/cron |
| **Playbook** | `wakeup.md` — the autonomous routine an agent runs when its task queue is empty |
| **Message** | Async non-blocking communication between any two participants (human or agent) |

Context is merged in this order (later layers override): **Agency → Team → Role → Project → Agent**

---

## Quick start — blank agency

```bash
# 1. Create agency
agencycli create agency --name "MyAgency" --desc "My first agency"
cd MyAgency

# 2. Create team + role
agencycli create team --name "engineering" --desc "Software engineers"
agencycli create role --team "engineering" --name "developer"

# 3. Create project
agencycli create project --name "my-service" --repo "/path/to/repo"

# 4. Hire an agent
agencycli hire --project my-service --team engineering --role developer \
  --model claudecode --name dev --sandbox docker

# 5. Add and run a task
agencycli task add --project my-service --agent dev \
  --title "Implement feature X" --prompt "..." --created-by human
agencycli run --project my-service --agent dev
```

## Quick start — from a template (recommended)

```bash
# 1. Create agency from template
agencycli create agency --name "MyAgency" \
  --template https://example.com/tech-agency.tar.gz
cd MyAgency

# 2. List available project blueprints
agencycli project blueprints

# 3. Create a project from a blueprint
agencycli create project --name "my-service" --blueprint default

# 4. Review and apply (hires agents, configures heartbeats, installs wakeup.md playbooks)
agencycli project show  --project my-service
agencycli project apply --project my-service

# 5. Start the scheduler — agents wake up on schedule and run their playbooks when idle
agencycli scheduler start

# 6. Monitor
agencycli inbox list          # task confirmations awaiting your decision
agencycli inbox messages      # async messages from agents
agencycli task list --project my-service --agent pm
```

---

## Global flag

All commands support `--dir <workspace>` to point to the agency root when not running from inside it:

```bash
agencycli --dir /root/code/MyAgency task list --project my-service --agent dev
```

---

## Command reference

### Workspace setup

```bash
agencycli create agency  --name "Name" [--desc "..."] [--template file.tar.gz|dir|URL]
agencycli create team    --name "engineering" [--desc "..."]
agencycli create role    --team "engineering" --name "developer" [--desc "..."]
agencycli create project --name "my-service"  --repo "/path/to/repo" [--desc "..."]
agencycli create project --name "my-service"  --blueprint default   # from blueprint
```

### Hiring agents

```bash
agencycli hire --project <proj> --team <team> [--role <role>] \
               --model <model> --name <name> [--sandbox docker]
# Supported models: claudecode  codex  gemini  cursor  generic-cli

agencycli sync --project <proj> --agent <name>   # re-sync after editing prompts/skills
agencycli sync --project <proj>                  # sync all agents in project
agencycli fire --project <proj> --agent <name>          # soft delete → .fired/
agencycli fire --project <proj> --agent <name> --force  # hard delete
```

### Project lifecycle

```bash
agencycli project blueprints
agencycli project show  --project P
agencycli project apply --project P             # hire agents + configure heartbeats + install playbooks
agencycli project apply --project P --dry-run
agencycli project apply --project P --force
```

**project-blueprints/default.yaml** example:
```yaml
name: "{{PROJECT_NAME}}"
description: "..."
agents:
  - name: dev
    role: developer
    team: engineering
    model: claudecode
    sandbox: true
    playbook: dev.md        # installed as wakeup.md by project apply
    heartbeat:
      enabled: true
      interval: 30m
      active_hours: "09:00-20:00"
      active_days: weekdays

  - name: pm
    role: product-manager
    team: product
    model: claudecode
    playbook: pm.md
    heartbeat:
      enabled: true
      interval: 30m
```

### Tasks

```bash
agencycli task add    --project P --agent A --title "T" --prompt "..." \
                      --created-by human|<project>/<agent> \
                      [--type feature|bug|chore] [--priority 0-3]
agencycli task list   --project P --agent A [--status pending] [--archived]
agencycli task show   <task-id>
agencycli task cancel <task-id>
agencycli task retry  <task-id>

# Emergency halt — cancel pending (and optionally running) tasks
agencycli task stop-all --project P --all-agents
agencycli task stop-all --project P --agent A --include-running

# View token usage and cost
agencycli task tokens --project P --agent A
agencycli task tokens --project P --all-agents
agencycli task tokens --project P --agent A --task <task-id>

# Called by agent inside its prompt:
agencycli task done --id <task-id> --status success --summary "brief description"
agencycli task done --id <task-id> --status failed  --error "reason"

# Pause and wait for human confirmation (blocks until human responds):
agencycli task confirm-request --id <task-id> \
  --summary "PR #42 ready for review" \
  --action-item "Open the PR and check the diff" \
  --action-item "Reply: merge / hold <reason>"
```

Task priority: 0=critical, 1=high, 2=normal (default), 3=low. The scheduler always picks the highest-priority pending task first. After `confirm-request`, the human's reply is available as `$CONFIRMATION_REPLY`.

### Running agents

```bash
agencycli run  --project P --agent A              # pick next pending task
agencycli run  --project P --agent A --task <id>  # run specific task
agencycli exec --project P --agent A --prompt "..." # one-off prompt (no task queue)
```

### Inbox — task confirmations (blocking)

Agents call `task confirm-request` to pause a task and wait for the human's decision.

```bash
agencycli inbox list
agencycli inbox show    <task-id>
agencycli inbox confirm <task-id> --message "approved, go ahead"
agencycli inbox reject  <task-id> --reason "needs rework"
agencycli inbox comment <task-id> --message "check the auth module first"
agencycli inbox forward <task-id> --to <project>/<agent> --note "please re-check"
```

### Inbox — async messages (non-blocking)

Any participant (human or agent) can send messages to any other. Recipients read them on their next wakeup — the scheduler auto-injects unread messages at the top of the wakeup prompt.

**Address format:** `human` or `project/agent` (e.g. `cc-connect/pm`, `cc-connect/dev-claude`)

```bash
# Send (single recipient)
agencycli inbox send --to cc-connect/pm --subject "Prioritise #55" --body "..."
agencycli inbox send --from cc-connect/pm --to human --subject "Update" --body "..."

# Group send (repeat --to)
agencycli inbox send \
  --to cc-connect/pm --to cc-connect/dev-claude --to human \
  --subject "All-hands" --body "..."

# Read (human's mailbox by default)
agencycli inbox messages                                              # unread only
agencycli inbox messages --recipient cc-connect/pm                   # agent's mailbox
agencycli inbox messages --from cc-connect/pm                        # filter by sender
agencycli inbox messages --all                                        # include already-read
agencycli inbox messages --archived                                   # show archived messages
agencycli inbox messages --mark-read                                  # mark all as read after listing

# Reply
agencycli inbox reply <msg-id> --from cc-connect/pm --body "Acknowledged."

# Forward a message to one or more recipients
agencycli inbox fwd <msg-id> --to cc-connect/dev-claude
agencycli inbox fwd <msg-id> --to cc-connect/pm --to human --note "FYI"

# Per-message status management
agencycli inbox read    <msg-id>                    # mark single message as read
agencycli inbox archive <msg-id>                    # archive (hidden from normal listing)
agencycli inbox delete  <msg-id>                    # permanently delete
agencycli inbox rm      <msg-id>                    # alias for delete
# --recipient flag available on all above to specify mailbox (default: human)
```

### Knowledge base (docs)

A bookmark index for documents. Files stay where they are; only metadata is tracked in `.agencycli/docs.yaml`. Virtual directories are created automatically.

```bash
# Add a document
agencycli docs add --path ./reports/design.md --title "System Design" \
  --index "cc-connect/architecture" --created-by human --tag design

# List / search
agencycli docs list [--index prefix] [--tag tag] [--created-by human] [--json]
agencycli docs search "design"
agencycli docs tree                     # virtual directory tree

# View details
agencycli docs show <doc-id> [--content]

# Update metadata / move
agencycli docs update <doc-id> --title "New Title" --tag newtag
agencycli docs move   <doc-id> --index "new/category"

# Remove from index (file is NOT deleted)
agencycli docs remove <doc-id>
```

The Web UI provides a visual knowledge base viewer at the "Knowledge Base" page with directory tree navigation and Markdown rendering.

### Daemon (heartbeat + wakeup routines)

```bash
# Configure heartbeat
agencycli scheduler heartbeat --project P --agent A \
  --enable --interval 30m \
  --active-hours "09:00-18:00" \
  --active-days  "weekdays"

# Set wakeup routine (runs as synthetic task when queue is empty)
agencycli scheduler heartbeat --project P --agent A \
  --wakeup-prompt-file /path/to/wakeup.md

# Start scheduler (aliases: sched, s)
agencycli scheduler start
agencycli scheduler stop
agencycli scheduler status

# Cron jobs
agencycli cron add     --project P --agent A \
  --title "Daily standup" --schedule "0 9 * * 1-5" --prompt "..."
agencycli cron list    --project P --agent A
agencycli cron delete  <cron-id>  --project P --agent A
agencycli cron enable  <cron-id>  --project P --agent A
agencycli cron disable <cron-id>  --project P --agent A
```

Each heartbeat cycle: if pending tasks exist → run highest-priority task; if queue is empty and `wakeup.md` is set → run wakeup routine. Unread messages are always prepended automatically.

### Agent playbooks

A playbook (`wakeup.md`) defines what an agent does when its task queue is empty. Store in `agent-playbooks/` and reference from `project.yaml` via `playbook:`.

`project apply` copies `agent-playbooks/<playbook>` → `agents/<name>/.agencycli/context/wakeup.md` and sets `wakeup_prompt: "@.agencycli/context/wakeup.md"` in `heartbeat.yaml`.

Typical wakeup.md patterns:
- Check injected unread messages (auto-prepended — no `inbox messages` call needed)
- Reply with: `agencycli inbox reply <msg-id> --from project/agent --body "..."`
- Send async update: `agencycli inbox send --from project/agent --to human --subject "..." --body "..."`
- Pause for human decision: `agencycli task confirm-request --id $TASK_ID --summary "..." --action-item "..."`
- Complete: `agencycli task done --id $TASK_ID --status success --summary "..."`

### Skills

```bash
agencycli role skill add    --team <t> --role <r> --skill <s>
agencycli role skill remove --team <t> --role <r> --skill <s>
agencycli role list         --team <t>
agencycli list skills
```

### Templates

A template bundles: `agency-prompt.md`, `teams/`, `skills/`, `agent-playbooks/`, `project-blueprints/`.

```bash
agencycli template pack --output my-agency.tar.gz \
  --name "tech-project" --version "1.0.0" \
  --author "Alice" --email "alice@example.com" \
  --description "Standard software engineering agency" \
  --keywords "engineering,software"

agencycli template info my-agency.tar.gz
agencycli template info my-agency.tar.gz --json

agencycli create agency --name "My Agency" --template my-agency.tar.gz
agencycli create agency --name "My Agency" --template https://example.com/tpl.tar.gz
```

### Sessions & misc

```bash
agencycli session show  --project P --agent A
agencycli session clear --project P --agent A
agencycli list teams | projects | agents | skills
agencycli show team engineering
agencycli show project my-api
agencycli show agent my-api dev [--raw]
agencycli version
```

---

## Agent context file locations

| Model | Context file | Skills dir |
|-------|-------------|------------|
| claudecode | `CLAUDE.md` | `.claude/skills/` |
| codex | `AGENTS.md` | (inlined) |
| gemini | `GEMINI.md` | `.gemini/skills/` |
| cursor | `.cursorrules` | `.cursor/rules/` |
| generic-cli | `context.md` | — |

---

## Agency directory structure

```
<AgencyName>/
  .agencycli/
    agency.yaml            ← workspace marker
    inbox.yaml             ← human task-confirmation inbox
    inbox.md               ← human-readable inbox summary
    messages.yaml          ← async messages for the human
  agency-prompt.md
  teams/
    <team>/
      team.yaml
      prompt.md
      roles/<role>/
        role.yaml
        prompt.md
  skills/
    <skill>/
      skill.yaml
      prompt.md
      [other files, e.g. scripts]
  agent-playbooks/         ← wakeup.md templates, distributed with agency template
    pm.md
    qa-reviewer.md
  project-blueprints/
    default.yaml
  projects/
    <project>/
      project.yaml         ← agents + heartbeats + crons + playbooks (declarative)
      prompt.md
      agents/
        <agent>/
          CLAUDE.md           ← merged context (claudecode)
          .claude/skills/     ← deployed skill files
          .agencycli/
            context/
              agency.md       ← agency-level prompt
              role-<team>-<role>.md
              project-<project>.md
              wakeup.md       ← autonomous routine (installed by project apply)
          heartbeat.yaml      ← set by project apply or scheduler heartbeat
          crons.yaml          ← set by project apply or cron add
          tasks.yaml          ← active tasks
          tasks_archive.yaml  ← completed tasks
          messages.yaml       ← async messages for this agent
          runs/               ← execution logs
```

---

## Context compression

Long-running agents accumulate context that may degrade quality. Each CLI has built-in auto-compression:

| Agent | Mechanism | Default | Configuration |
|-------|-----------|---------|---------------|
| **Claude Code** | auto-compact | ~90% of context window | Env vars: `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE` (threshold %, recommend 70-80), `CLAUDE_CODE_AUTO_COMPACT_WINDOW` (effective token window size) |
| **Codex** | auto-compact | ~90% of context window | Config key `model_auto_compact_token_limit` in codex config |
| **Gemini** | auto-compress | 70% of context window | `chatCompression.contextPercentageThreshold` (0-1) in `.gemini/settings.json` |

Set per-agent env vars via the Web UI (Agent → API Provider) or directly in `agent.yaml`:

```yaml
env:
  CLAUDE_AUTOCOMPACT_PCT_OVERRIDE: "75"
```

---

## Tips for agents

1. Always use `--dir <workspace>` if you are not inside the agency directory.
2. Get your own task ID from `$TASK_ID` (set by the scheduler when running a task).
3. When you need human approval **and must wait**: use `task confirm-request` — do NOT call `task done`. The task resumes when the human confirms; their reply is in `$CONFIRMATION_REPLY`.
4. When you want to notify someone **without blocking**: use `inbox send --from <your-address> --to <recipient> --body "..."`.
5. Unread messages are auto-injected at the top of your wakeup prompt — no need to call `inbox messages` yourself.
6. Use `inbox reply <msg-id> --from <your-address>` to reply to a message.
7. Use `agencycli list agents` to discover all agents and their `project/agent` addresses.
8. Use `agencycli exec --project P --agent A --prompt "..."` for quick one-off tests without adding a task.
