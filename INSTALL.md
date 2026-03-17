# agencycli — Installation & Setup Guide

This document is a complete guide for installing `agencycli` and building your first AI agent team. Hand this file to any AI coding agent (Claude Code, Codex, Gemini CLI, Cursor) and it can follow these instructions autonomously to get everything running.

---

## 1. Install agencycli

Choose **one** method:

### Option A — npm (recommended, no Go required)

```bash
npm install -g agencycli
agencycli version
```

### Option B — Go install (requires Go 1.22+)

```bash
go install github.com/chenhg5/agencycli/cmd/agencycli@latest
agencycli version
```

### Option C — Pre-built binary

Download from [https://github.com/chenhg5/agencycli/releases](https://github.com/chenhg5/agencycli/releases) and move to a directory on your `PATH`.

---

## 2. Understand the structure

```
Agency                   ← global context shared by every agent
  └─ Team                ← capability group (engineering, qa, growth…)
       └─ Role           ← job function (developer, qa-engineer, content-writer…)
            └─ Project   ← concrete product or code repository
                 └─ Agent ← individual AI agent (model + merged context + skills)
```

**Context flows top-down.** Every agent automatically inherits: agency → team → role → project context.

**Skills** are reusable capability definitions (instructions + scripts). They are bound to teams or roles and deployed into each agent's working directory automatically on `hire` or `sync`.

**Agent playbooks** (`wakeup.md` files) define what an agent does when its task queue is empty — for example, scanning GitHub issues, reviewing open PRs, or sending status updates. Store them in `agent-playbooks/` and reference them from `project.yaml` via the `playbook:` field. `project apply` installs them automatically.

**Task queues** drive all agent work. Tasks have priorities (0=critical … 3=low); the daemon always picks the highest-priority pending task first. The wakeup routine fires as a low-priority synthetic task when the queue is empty.

**Async inbox messaging** lets any participant (agent or human) send non-blocking messages to any other. Recipients see their unread messages automatically at the top of their wakeup prompt.

**Project blueprints** declare which agents a project needs, their heartbeat schedule, and which playbooks to install. Running `project apply` hires every agent and wires up all schedules in one command.

---

## Path A — Start from a template (recommended)

If someone gives you an agency template (`.tar.gz` or URL), this is the fastest path.

### Step 1 — Create agency from template

```bash
agencycli create agency --name "MyAgency" \
  --template https://example.com/tech-agency.tar.gz

cd MyAgency
```

### Step 2 — List available project blueprints

```bash
agencycli project blueprints
```

Output example:
```
BLUEPRINT     AGENTS  PLAYBOOKS
────────────  ──────  ─────────
cc-connect    3       pm.md, qa-reviewer.md
```

### Step 3 — Create a project from a blueprint

```bash
agencycli create project --name "my-service" --blueprint default \
  --desc "My REST API service"
```

This writes `projects/my-service/project.yaml` pre-filled with agent definitions, heartbeat schedules, and playbook references.

### Step 4 — Review the project configuration

```bash
agencycli project show --project my-service
```

You will see each agent, its model, role, sandbox setting, heartbeat schedule, and playbook. Edit `projects/my-service/project.yaml` if you want to adjust anything before applying.

### Step 5 — Apply: hire agents, configure schedules, install playbooks

```bash
agencycli project apply --project my-service
```

This single command:
- Hires every agent declared in `project.yaml`
- Writes `heartbeat.yaml` for agents with a heartbeat schedule
- Writes `crons.yaml` for agents with cron jobs
- Copies `agent-playbooks/<playbook>` → `agents/<name>/wakeup.md` for agents with a `playbook:` field
- Sets `wakeup_prompt: "@wakeup.md"` in each agent's `heartbeat.yaml`
- Merges the full context chain (agency → team → role → project) into each agent's working directory

Use `--dry-run` to preview without making changes:
```bash
agencycli project apply --project my-service --dry-run
```

### Step 6 — Edit context prompts

```bash
vim agency-prompt.md                                         # global rules
vim teams/engineering/prompt.md                              # team standards
vim teams/engineering/roles/developer/prompt.md              # role responsibilities
vim projects/my-service/prompt.md                            # project specifics
```

After editing, re-sync all agents:

```bash
agencycli sync --project my-service
```

### Step 7 — Start the daemon

```bash
agencycli daemon start
```

Agents now wake up automatically on their heartbeat schedule:
- If there are **pending tasks**, the daemon picks the highest-priority one and runs it.
- If the queue is **empty** and a `wakeup.md` is configured, the agent runs its playbook autonomously (scanning issues, reviewing PRs, etc.).
- Any **unread inbox messages** are automatically prepended to the wakeup prompt so the agent sees them immediately.

### Step 8 — Monitor

```bash
# Task confirmations awaiting your decision
agencycli inbox list

# Async messages from agents
agencycli inbox messages

# Task queue for a specific agent (sorted by priority)
agencycli task list --project my-service --agent pm

# Token usage across all agents
agencycli task tokens --project my-service --all-agents

# Emergency halt: cancel all pending/running tasks
agencycli task stop-all --project my-service --all-agents
```

---

## Path B — Build from scratch

### Step 1 — Create the workspace

```bash
agencycli create agency --name "MyAgency" --desc "Building great software with AI"
cd MyAgency
```

Edit `agency-prompt.md` — add anything every agent should always know: coding standards, communication style, how to handle blockers, how to report progress.

### Step 2 — Create teams

```bash
agencycli create team --name "engineering"          --desc "Software engineers"
agencycli create team --name "engineering/backend"  --desc "Go/API services"
agencycli create team --name "qa"                   --desc "Quality assurance"
agencycli create team --name "product"              --desc "Product management"
agencycli create team --name "growth"               --desc "Content and marketing"
```

Edit `teams/<name>/prompt.md` to add team-specific conventions.

### Step 3 — Add skills

Skills have no built-in entries — define only what you need:

```bash
mkdir -p skills/github-pr-review
```

```yaml
# skills/github-pr-review/skill.yaml
name: github-pr-review
description: Instructions for reviewing GitHub pull requests
```

```markdown
<!-- skills/github-pr-review/prompt.md -->
## Skill: GitHub PR Review

When asked to review a PR:
1. Run `gh pr view <number>` and `gh pr diff <number>`
2. Check for bugs, missing tests, security issues
3. Approve with `gh pr review <number> --approve` or request changes
```

You can also bundle scripts alongside `prompt.md`. Use `{{SKILL_DIR}}` to reference them:

```markdown
Use `{{SKILL_DIR}}/my-script.sh` to run the deployment.
```

### Step 4 — Create roles

```bash
agencycli create role --team "engineering" --name "developer"    --desc "Implements features"
agencycli create role --team "engineering" --name "qa-engineer"  --desc "Reviews and tests"
agencycli create role --team "product"     --name "pm"           --desc "Owns roadmap and tasks"
agencycli create role --team "growth"      --name "content-writer" --desc "Writes and publishes"
```

Edit each role:

```yaml
# teams/engineering/roles/developer/role.yaml
name: developer
description: Implements features and fixes bugs
skills:
  - github-pr-review
setup:
  dirs:
    - scratch
    - notes
```

```markdown
<!-- teams/engineering/roles/developer/prompt.md -->
You are a senior software engineer. You write clean, tested, documented code...
```

Bind or unbind skills at any time:

```bash
agencycli role skill add    --team engineering --role developer --skill github-pr-review
agencycli role skill remove --team engineering --role developer --skill github-pr-review
agencycli role list --team engineering
```

### Step 5 — Write agent playbooks

Playbooks define what an agent does when it wakes up with no pending tasks.

```bash
mkdir -p agent-playbooks
```

```markdown
<!-- agent-playbooks/dev.md -->
# Dev Autonomous Routine

You are the lead developer. Each wakeup cycle:

## Step 1: Check unread messages
(Injected automatically above this prompt if any — reply with `inbox reply <msg-id> --from project/dev --body "..."`)

## Step 2: Scan for work
Check if any tasks are pending:
  agencycli --dir $AGENCY_DIR task list --project my-api --agent dev

## Step 3: Self-assign from backlog (if PM has left tasks)
Pick the highest-priority pending task and start work.

## Done
  agencycli --dir $AGENCY_DIR task done --id $TASK_ID --status success --summary "..."
```

Key patterns:
- Unread messages are **auto-injected** at the top of the prompt by the daemon — no need to call `inbox messages` in the playbook
- To reply to a message: `agencycli inbox reply <msg-id> --from project/agent --body "..."`
- To send a non-blocking message to another agent: `agencycli inbox send --from project/dev --to project/pm --subject "..." --body "..."`
- To pause and wait for human confirmation: `agencycli task confirm-request --id $TASK_ID --summary "..." --action-item "..."`

### Step 6 — Create a project blueprint

```bash
mkdir -p project-blueprints
```

```yaml
# project-blueprints/default.yaml
name: "{{PROJECT_NAME}}"
description: "Service managed by engineering team"

agents:
  - name: dev
    role: developer
    team: engineering
    model: claudecode
    sandbox: true
    playbook: dev.md          # installed as wakeup.md by project apply
    heartbeat:
      enabled: true
      interval: 30m
      active_hours: "09:00-20:00"
      active_days: weekdays

  - name: pm
    role: pm
    team: product
    model: claudecode
    playbook: pm.md
    heartbeat:
      enabled: true
      interval: 30m
      active_hours: "09:00-20:00"
      active_days: weekdays
```

### Step 7 — Create and apply a project

```bash
# Create the project (writes project.yaml from blueprint)
agencycli create project --name "my-api" --blueprint default \
  --desc "REST API service" --repo "/absolute/path/to/my-api"

# Review the generated config
agencycli project show --project my-api

# Apply: hire agents + configure heartbeats + install playbooks
agencycli project apply --project my-api
```

Edit `projects/my-api/prompt.md` — add project-specific context: tech stack, build/test commands, PR conventions, issue tracker URL.

Re-sync after editing:

```bash
agencycli sync --project my-api
```

### Step 8 — Start the daemon

```bash
agencycli daemon start
```

---

## Working with tasks directly

```bash
# Add a task manually (agents pick this up on next wakeup)
agencycli task add \
  --project my-api --agent dev \
  --title "Fix login redirect" --type bug --priority 1 \
  --prompt "The redirect is broken on mobile Safari. Fix and open a PR."

# Run manually (bypasses daemon scheduling)
agencycli run --project my-api --agent dev

# Run a quick one-off prompt
agencycli exec --project my-api --agent dev \
  --prompt "Run gh issue list and summarise all open issues"

# View token usage
agencycli task tokens --project my-api --agent dev

# Emergency halt
agencycli task stop-all --project my-api --all-agents --include-running
```

---

## Cron jobs

Add recurring tasks to any agent:

```bash
agencycli cron add \
  --project my-api --agent dev \
  --title "Weekly dependency audit" \
  --schedule "0 9 * * 1" \
  --prompt "Run 'go list -m -u all' and open a PR updating any outdated dependencies."

agencycli cron list --project my-api --agent dev
```

The cron definition is also declarable in `project.yaml` under `agents[*].crons`, so it is applied automatically by `project apply`.

---

## Human inbox

### Task confirmations (blocking)

Agents call `task confirm-request` when they need your decision before proceeding. The task pauses until you respond.

```bash
agencycli inbox list
agencycli inbox show    <task-id>   # full prompt, action items, log tail
agencycli inbox confirm <task-id> --message "Approved — merge when CI passes"
agencycli inbox reject  <task-id> --reason "Out of scope for this sprint"
agencycli inbox comment <task-id> --message "Check the auth module specifically"
agencycli inbox forward <task-id> --to my-api/dev --note "Please re-check the edge case"
```

### Async messages (non-blocking)

Any agent or human can send a message to any inbox. The recipient reads it on their next wakeup.

```bash
# Human → agent
agencycli inbox send --to my-api/pm \
  --subject "Prioritise issue #42" \
  --body "Customer reported this as critical. Please move to P0."

# Read your messages (human inbox by default)
agencycli inbox messages

# Inspect an agent's mailbox
agencycli inbox messages --recipient my-api/pm --all

# Reply
agencycli inbox reply <msg-id> --body "Noted, I'll update the backlog on next wakeup."
```

The inbox task-confirmation list is also rendered to `.agencycli/inbox.md` at the workspace root for easy reading.

---

## Heartbeat time windows

Restrict when an agent can wake up:

```bash
agencycli daemon heartbeat \
  --project my-api --agent dev \
  --enable --interval 30m \
  --active-hours "09:00-18:00" \
  --active-days  "weekdays"
```

Supported `--active-days` values: `weekdays`, `weekends`, or `Mon,Tue,Wed,Thu,Fri,Sat,Sun` (comma-separated).

Overnight windows like `22:00-06:00` work correctly.

---

## Templates — share your agency

Pack your agency (teams, roles, skills, agent playbooks, project blueprints) as a shareable archive:

```bash
agencycli template pack --output tech-agency.tar.gz \
  --name "tech-project" --version "1.0.0" \
  --author "Alice" --email "alice@example.com" \
  --description "Standard software engineering agency template" \
  --keywords "engineering,software,go"
```

The archive includes: `agency-prompt.md`, `teams/`, `skills/`, `agent-playbooks/`, `project-blueprints/`.

Inspect a template before using it:

```bash
agencycli template info tech-agency.tar.gz
agencycli template info tech-agency.tar.gz --json
```

Create an agency from a template:

```bash
agencycli create agency --name "MyAgency" --template tech-agency.tar.gz
agencycli create agency --name "MyAgency" --template https://example.com/tpl.tar.gz
agencycli create agency --name "MyAgency" --template ./my-local-template-dir
```

---

## Context sync

After editing any prompt, skill, or role, regenerate affected agent working directories:

```bash
agencycli sync                               # all agents with changed context
agencycli sync --project my-api             # one project
agencycli sync --project my-api --name dev  # one agent
agencycli sync --force                       # force regenerate everything
```

---

## `--dir` flag

Run any command against a workspace outside your current directory:

```bash
agencycli --dir /path/to/MyAgency list agents
agencycli --dir /path/to/MyAgency task list --project my-api --agent dev
agencycli --dir /path/to/MyAgency inbox list
agencycli --dir /path/to/MyAgency daemon start
```

---

## Command reference

| Category | Commands |
|----------|----------|
| Workspace | `create agency/team/role/project` |
| Project lifecycle | `project show/apply/blueprints` |
| Role management | `role list`, `role skill add/remove` |
| Agent lifecycle | `hire`/`assign`, `fire`, `sync`, `list`, `show` |
| Execution | `exec`, `run` |
| Tasks | `task add/list/show/done/retry/cancel/confirm-request/stop-all/tokens` |
| Inbox (confirmations) | `inbox list/show/confirm/reject/comment/forward` |
| Inbox (messaging) | `inbox send/messages/reply` |
| Scheduling | `daemon start/stop/status/heartbeat`, `cron add/list/delete/enable/disable` |
| Templates | `template pack/info` |
| Session | `session show/set/clear` |
| Meta | `version` |

```bash
agencycli <command> --help
agencycli <command> <subcommand> --help
```

---

## Setup checklist — from scratch

```
[ ] agencycli create agency --name "..." --desc "..."
[ ] Edit agency-prompt.md  (global rules, values, communication style)

[ ] agencycli create team --name "engineering"
[ ] Edit teams/engineering/prompt.md  (coding standards, conventions)

[ ] agencycli create role --team engineering --name developer
[ ] Edit teams/engineering/roles/developer/prompt.md  (responsibilities)
[ ] Edit teams/engineering/roles/developer/role.yaml  (skills, setup dirs)

[ ] mkdir -p skills/<name> && write skill.yaml + prompt.md
[ ] agencycli role skill add --team engineering --role developer --skill <name>

[ ] mkdir -p agent-playbooks && write agent-playbooks/dev.md
[ ] mkdir -p project-blueprints && write project-blueprints/default.yaml
      (reference playbook: dev.md in the blueprint)

[ ] agencycli create project --name "my-app" --blueprint default --repo /path/to/repo
[ ] Edit projects/my-app/prompt.md  (tech stack, build commands, PR conventions)
[ ] agencycli project apply --project my-app

[ ] agencycli daemon start

[ ] agencycli inbox list          # check for task confirmations
[ ] agencycli inbox messages      # check for async messages from agents
[ ] agencycli task list --project my-app --agent dev
```

## Setup checklist — from a template

```
[ ] agencycli create agency --name "..." --template <url-or-file>
[ ] Edit agency-prompt.md  (personalise global rules)
[ ] agencycli project blueprints  (see what's available)
[ ] agencycli create project --name "my-app" --blueprint default
[ ] Edit projects/my-app/prompt.md  (project-specific context)
[ ] agencycli project apply --project my-app
[ ] agencycli daemon start
[ ] agencycli inbox list
[ ] agencycli inbox messages
```
