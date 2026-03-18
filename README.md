# agencycli

**agencycli** is a CLI tool for managing AI agent teams through an agency-style organisational structure. Define teams, roles, projects, and agent playbooks once — hire agents with fully assembled context, assign them tasks, and let them work autonomously on a heartbeat schedule.

No more copy-pasting prompts between sessions. No more context drift. No more manually wiring up who does what.

## Core model

```
Agent = Model + Context + Skills
```

Context is assembled from a hierarchy and **merged automatically** on `hire`:

```
Agency                ← global rules, values, tone
  └─ Team             ← capability group (engineering, qa, growth…)
       └─ Sub-team    ← nested (engineering/backend, engineering/frontend)
            └─ Role   ← job function (go-developer, pr-reviewer, content-writer)
                 └─ Project ← concrete product or initiative
```

Agents work autonomously via a **heartbeat loop** — wake up, process all pending tasks, sleep. When the task queue is empty, a **wakeup routine** (`wakeup.md`) runs as a default prompt, letting agents proactively scan for new work (GitHub issues, open PRs, etc.). Agents communicate asynchronously via **inbox messaging** — any agent or human can send a message to any other participant; recipients read it on their next wakeup.

---

## How it works

### Layer 1 — Context management

- Define your agency once: teams, roles, skills, and how they compose
- `hire` an agent → agencycli merges the full chain and writes a ready-to-use working directory
- Supported models: `claudecode`, `codex`, `gemini`, `cursor`, `qoder`, `opencode`, `iflow`, `generic-cli`
- `sync` propagates prompt/skill changes to all affected agents

### Layer 2 — Task automation

- Per-agent **task queues** with a 7-state lifecycle and priority ordering (0=critical … 3=low)
- **Heartbeat scheduler**: non-overlapping wakeup loop, session-preserving, with time-window scheduling (`active_hours`, `active_days`)
- **Wakeup routine**: when the task queue is empty, the agent runs `wakeup.md` as a synthetic task — enabling autonomous proactive work without manual task assignment
- **Cron jobs**: add recurring tasks on a crontab schedule
- **Human inbox**: agents route `awaiting_confirmation` tasks here via `task confirm-request`; you confirm/reject/forward
- **Async messaging**: any participant (agent or human) can send non-blocking messages to any other participant's inbox
- **Docker sandbox**: isolated containers with credentials and repos auto-mounted

### Layer 3 — Agent playbooks

Playbooks (`wakeup.md` files) define how an agent behaves when it wakes up with an empty task queue. Package them in `agent-playbooks/` to distribute with templates.

```yaml
# project.yaml
agents:
  - name: pm
    playbook: pm.md          # copied to agent dir as wakeup.md on project apply
    heartbeat:
      enabled: true
      interval: 30m
```

When `project apply` runs, it copies `agent-playbooks/pm.md` → `agents/pm/wakeup.md` and automatically sets `wakeup_prompt: "@wakeup.md"` in `heartbeat.yaml`.

### Layer 4 — Templates

Package an entire agency (teams, roles, skills, agent playbooks, project blueprints) as a `.tar.gz` template. Share it, apply it to a new agency with one command.

---

## Supported models

| `--model`     | Context file(s)                                                   |
|---------------|-------------------------------------------------------------------|
| `claudecode`  | `CLAUDE.md` + `@import` layers + `.claude/skills/`               |
| `codex`       | `AGENTS.md` single merged file (skills inlined)                  |
| `cursor`      | `.cursorrules` + `.cursor/rules/agencycli.mdc`                   |
| `gemini`      | `GEMINI.md` + `@import` layers + `.gemini/skills/`               |
| `qoder`       | `AGENTS.md` single merged file                                    |
| `opencode`    | `OPENCODE.md` single merged file                                  |
| `iflow`       | `IFLOW.md` single merged file                                     |
| `generic-cli` | `context.md` plain text                                           |

---

## Installation

```bash
npm install -g agencycli        # npm (no Go required)
go install github.com/chenhg5/agencycli/cmd/agencycli@latest  # Go
```

Or build from source:

```bash
git clone https://github.com/chenhg5/agencycli
cd agencycli && make install
```

---

## Quick start — from a template (recommended)

Templates bundle teams, roles, skills, agent playbooks, and project blueprints in one archive:

```bash
# 1. Create an agency from a shared template
agencycli create agency --name "MyAgency" \
  --template https://example.com/tech-agency.tar.gz
cd MyAgency

# 2. List the project blueprints the template ships with
agencycli project blueprints

# 3. Create a project — project.yaml is pre-filled with agents, heartbeats, and playbooks
agencycli create project --name "my-service" --blueprint default

# 4. Review what will be created
agencycli project show --project my-service

# 5. Apply: hire all agents + configure heartbeats + install wakeup.md playbooks
agencycli project apply --project my-service

# 6. Start the scheduler — agents now wake up on schedule and run their playbooks
agencycli scheduler start

# 7. Monitor
agencycli inbox list          # task confirmations awaiting your decision
agencycli inbox messages      # async messages from agents
agencycli task list --project my-service --agent pm
```

## Quick start — from scratch

```bash
# 1. Create the workspace
agencycli create agency --name "MyAgency" --desc "Building great software"
cd MyAgency

# 2. Create teams and roles
agencycli create team --name "engineering"
agencycli create role --team "engineering" --name "developer"

# 3. Write agent playbooks
mkdir -p agent-playbooks
# edit agent-playbooks/dev.md — defines what the dev agent does when it wakes up

# 4. Create a project blueprint
mkdir -p project-blueprints
# edit project-blueprints/default.yaml

# 5. Create and apply a project
agencycli create project --name "my-api" --blueprint default
agencycli project apply  --project my-api

# 6. Start scheduler
agencycli scheduler start
```

---

## Workspace layout

```
MyAgency/
  .agencycli/
    agency.yaml              # workspace metadata
    inbox.yaml               # human task-confirmation inbox (auto-managed)
    inbox.md                 # human-readable inbox (auto-generated)
    messages.yaml            # async messages delivered to human

  agency-prompt.md           # agency-wide context

  teams/
    engineering/
      team.yaml
      prompt.md
      roles/
        developer/
          role.yaml          # skills[], setup (dirs/files to create)
          prompt.md          # role-specific context layer

  skills/                    # no built-ins — define only what you need
    github-push-relay/
      skill.yaml
      prompt.md              # uses {{SKILL_DIR}} for script paths
      git-push-github.sh     # bundled file, chmod+x preserved

  agent-playbooks/           # wakeup.md templates, distributed with the agency template
    pm.md
    qa-reviewer.md

  project-blueprints/        # project templates packaged with the agency template
    default.yaml             # declares agents, heartbeats, and playbooks

  projects/
    my-api/
      project.yaml           # declarative: agents + heartbeats + crons + playbooks
      prompt.md              # project-specific context
      agents/
        dev/
          CLAUDE.md              # merged context (@imports all layers)
          wakeup.md              # agent's autonomous routine (installed by project apply)
          tasks.yaml             # active task queue
          tasks_archive.yaml     # completed tasks
          heartbeat.yaml         # wakeup config (set by project apply)
          crons.yaml             # scheduled tasks (set by project apply)
          messages.yaml          # async messages delivered to this agent
          runs/                  # execution logs
          .agencycli-context/    # individual layer files (managed)
          .claude/skills/        # deployed skill files
```

---

## Commands

### `create` — workspace setup

```bash
agencycli create agency  --name "MyAgency" [--desc "..."] [--template file.tar.gz|dir|URL]
agencycli create team    --name "engineering" [--desc "..."]
agencycli create team    --name "engineering/backend"        # nested sub-team
agencycli create role    --team "engineering" --name "developer" [--desc "..."]
agencycli create project --name "my-api" [--desc "..."] [--repo "/path/to/repo"]
agencycli create project --name "my-api" --blueprint default  # from a project blueprint
```

### `project` — project lifecycle

```bash
# List blueprints shipped with the template
agencycli project blueprints

# Show project.yaml (agents, heartbeats, playbooks)
agencycli project show --project my-api

# One-command bootstrap: hire all agents + configure heartbeats/crons + install playbooks
agencycli project apply --project my-api
agencycli project apply --project my-api --dry-run   # preview
agencycli project apply --project my-api --force     # re-hire existing agents
```

**`project-blueprints/default.yaml`** example:

```yaml
name: "{{PROJECT_NAME}}"
description: "REST API service"
agents:
  - name: dev
    role: developer
    team: engineering
    model: claudecode
    sandbox: true
    heartbeat:
      enabled: true
      interval: 30m
      active_hours: "09:00-20:00"
      active_days: weekdays
    playbook: dev.md          # installed as wakeup.md by project apply

  - name: pm
    role: product-manager
    team: product
    model: claudecode
    heartbeat:
      enabled: true
      interval: 30m
    playbook: pm.md
```

### `hire` / `assign` / `fire` / `sync`

```bash
# Hire an agent (hire and assign are identical)
agencycli hire \
  --project my-api --team engineering --role developer \
  --model claudecode --name dev \
  [--sandbox docker] [--force]

# Re-sync context after editing prompts or skills
agencycli sync --project my-api --name dev
agencycli sync --project my-api   # all agents in project
agencycli sync                    # entire agency

# Fire (remove) an agent
agencycli fire --project my-api --agent dev           # soft delete → .fired/
agencycli fire --project my-api --agent dev --force   # hard delete
```

### `task` — task queue

```bash
agencycli task add    --project P --agent A --title "T" --prompt "..." \
                      [--type feature|bug|chore] [--priority 0-3]
agencycli task list   --project P --agent A [--status pending] [--archived]
agencycli task show   <task-id>
agencycli task cancel <task-id>
agencycli task retry  <task-id>

# Stop all running or pending tasks (emergency halt)
agencycli task stop-all --project P [--agent A | --all-agents] \
                        [--include-running] [--no-pending]

# View token usage and cost across agent runs
agencycli task tokens --project P [--agent A | --all-agents] [--all]

# Called by the agent inside its prompt:
agencycli task done --id <id> --status success --summary "what was done"
agencycli task done --id <id> --status failed  --error "reason"

# Route to human inbox for a decision (blocks current task until human responds):
agencycli task confirm-request --id <id> --summary "PR ready" \
  --action-item "Review the diff" \
  --action-item "Confirm merge"
```

**Task priority:** 0=critical, 1=high, 2=normal (default), 3=low. The scheduler always picks the highest-priority pending task first.

**Task lifecycle:**
```
pending → in_progress → done_success
                      → done_failed  → (auto-retry if max_retries set)
                      → awaiting_confirmation → in_progress (confirm)
                                              → cancelled   (reject)
```

### `run` / `exec`

```bash
agencycli run  --project P --agent A              # execute next pending task
agencycli run  --project P --agent A --task <id>  # run a specific task
agencycli exec --project P --agent A --prompt "..." # one-shot, no task queue
```

### `inbox` — human confirmations and async messaging

The inbox has two distinct concepts:

**Task confirmations** — an agent pauses and waits for your decision:
```bash
agencycli inbox list
agencycli inbox show    <task-id>         # shows summary, action items, log tail
agencycli inbox confirm <task-id> [--message "notes for the agent"]
agencycli inbox reject  <task-id> --reason "..."
agencycli inbox comment <task-id> --message "..."
agencycli inbox forward <task-id> --to <project>/<agent> --note "..."
```

**Async messages** — non-blocking communication between any participants:
```bash
# Send a message to an agent or the human
agencycli inbox send --to cc-connect/pm --subject "Prioritise issue #42" --body "..."
agencycli inbox send --to human --from cc-connect/pm --subject "Backlog update" --body "..."
agencycli inbox send --to cc-connect/dev-claude --from cc-connect/pm \
  --subject "New task context" --body "Extra info for the task I just created..."

# Read messages (human's mailbox by default)
agencycli inbox messages
agencycli inbox messages --recipient cc-connect/pm   # inspect an agent's mailbox
agencycli inbox messages --all                       # include already-read messages
agencycli inbox messages --mark-read                 # mark as read after listing

# Reply to a message
agencycli inbox reply <msg-id> --body "..."
agencycli inbox reply <msg-id> --from cc-connect/pm --body "..."
```

Agents receive unread messages automatically at the top of their wakeup prompt — no need to poll in `wakeup.md`. Messages are marked as read after a successful wakeup run.

### `scheduler` — heartbeat scheduler

The heartbeat is a **non-overlapping wakeup loop**: after each cycle completes all pending tasks, the agent sleeps for `interval`, then wakes again. When the queue is empty, the **wakeup routine** fires instead.

```bash
# Configure heartbeat for one agent
agencycli scheduler heartbeat --project P --agent A \
  --enable --interval 30m \
  --active-hours "09:00-18:00" \  # only wake in this window (local time)
  --active-days  "weekdays"       # Mon–Fri only (or Mon,Wed,Fri / weekends)

# Set a wakeup routine (runs when queue is empty)
agencycli scheduler heartbeat --project P --agent A \
  --wakeup-prompt-file /path/to/wakeup.md

# Start scheduler (all enabled agents)
agencycli scheduler start
agencycli scheduler stop
agencycli scheduler status
```

Overnight ranges like `22:00-06:00` are supported. Outside the active window the scheduler shows `⏸ outside active window — next wakeup in Xh`.

### `cron` — scheduled tasks

```bash
agencycli cron add    --project P --agent A \
  --title "Daily standup" --schedule "0 9 * * 1-5" \
  --prompt "Generate a standup report..."
agencycli cron list   --project P --agent A
agencycli cron delete <cron-id>  --project P --agent A
agencycli cron enable <cron-id>  --project P --agent A
agencycli cron disable <cron-id> --project P --agent A
```

Crons enqueue a new task each time the schedule fires. The scheduler checks for due crons on every heartbeat wakeup.

### `template` — share agencies

```bash
# Pack the current agency as a shareable template
# Includes: agency-prompt.md, teams/, skills/, agent-playbooks/, project-blueprints/
agencycli template pack --output tech-agency.tar.gz \
  --name "tech-project" --version "1.0.0" \
  --author "Alice" --email "alice@example.com" \
  --description "Standard software engineering agency template" \
  --keywords "engineering,software"

# Inspect a template (local file, directory, or remote URL)
agencycli template info tech-agency.tar.gz
agencycli template info tech-agency.tar.gz --json

# Create an agency from a template
agencycli create agency --name "MyAgency" --template tech-agency.tar.gz
agencycli create agency --name "MyAgency" --template https://example.com/tpl.tar.gz
```

A template archive contains: `agency-prompt.md`, `teams/`, `skills/`, `agent-playbooks/`, `project-blueprints/`.  
A `template.json` in the archive root holds metadata (name, version, author, email, description, keywords).

### `role` — role management

```bash
agencycli role list  --team engineering
agencycli role skill add    --team engineering --role developer --skill github-push-relay
agencycli role skill remove --team engineering --role developer --skill github-push-relay
```

### `session` / `list` / `show` / `version`

```bash
agencycli session show  --project P --agent A
agencycli session clear --project P --agent A
agencycli list teams | projects | agents | skills
agencycli show team engineering
agencycli show project my-api
agencycli show agent my-api dev [--raw]
agencycli version
```

### Global flag: `--dir`

All commands work against a workspace outside the current directory:

```bash
agencycli --dir /path/to/MyAgency inbox list
agencycli --dir /path/to/MyAgency task list --project my-api --agent dev
agencycli --dir /path/to/MyAgency scheduler start
```

---

## Skills

Skills are reusable capability definitions deployed into agent working directories. **No built-ins** — define only what your agents actually need.

```
skills/github-push-relay/
  skill.yaml             # name + description
  prompt.md              # uses {{SKILL_DIR}} for absolute paths to bundled files
  git-push-github.sh     # bundled script, chmod+x preserved
```

Use `{{SKILL_DIR}}` in `prompt.md` to reference co-located files:

```markdown
Use `{{SKILL_DIR}}/git-push-github.sh` to push code to GitHub.
```

Bind to teams (`team.yaml`) or roles (`role.yaml`). After changing skills, run `agencycli sync`.

---

## Agent playbooks

Playbooks live in `agent-playbooks/` and define what an agent does when it wakes up with an empty task queue. They are referenced from `project.yaml` via the `playbook:` field on each agent spec and installed as `wakeup.md` by `project apply`.

```
agent-playbooks/
  pm.md          ← PM autonomous routine: scan issues, maintain backlog, confirm with human
  qa-reviewer.md ← QA autonomous routine: scan open PRs, review, request merge confirmation
```

The scheduler auto-injects any **unread inbox messages** at the top of the wakeup prompt before running the routine. No explicit polling needed in `wakeup.md`.

---

## Docker sandbox

```bash
agencycli hire --project my-api --team engineering --role developer \
  --model claudecode --name dev --sandbox docker
```

Each `run` or `exec` launches a fresh container with:

- Agent working directory mounted at its host path (read/write)
- Project repo mounted at its host path (read/write)
- Agency workspace root mounted (agents can call `agencycli` inside the container)
- `agencycli` binary mounted read-only at `/usr/local/bin/agencycli`
- Credentials auto-mounted (`~/.claude`, `~/.config/gh`, `~/.ssh`, `~/.codex`, `~/.gemini`, `~/.cursor`)
- Well-known API keys forwarded as environment variables
- Claude Code: root execution with `IS_SANDBOX=1 --dangerously-skip-permissions`
- Codex: `CODEX_UNSAFE_ALLOW_NO_SANDBOX=1`

Build with Chinese mirrors if needed:

```bash
docker build --build-arg CN_MIRROR=1 -t agencycli/sandbox-claudecode docker/sandbox-claudecode/
```

---

## Roadmap

### Context management ✓
- [x] Agency / team / project / role scaffolding
- [x] Context merging: `agency → team chain → role → project`
- [x] All model formatters (claudecode, codex, cursor, gemini, qoder, opencode, iflow, generic-cli)
- [x] `sync` with SHA-256 change detection
- [x] `hire` / `assign` alias / `fire` (soft + hard delete)
- [x] Skills with bundled files + `{{SKILL_DIR}}` resolution
- [x] `add_dirs` — project repo exposed to agent
- [x] `--dir` global flag

### Task automation ✓
- [x] Task queue per agent (7-state machine, priority ordering)
- [x] Human inbox with confirm / reject / comment / forward
- [x] Async messaging: `inbox send / messages / reply` (non-blocking, any participant)
- [x] `run` / `exec`
- [x] Session continuity across heartbeat cycles
- [x] Heartbeat scheduler with active-hours / active-days windows
- [x] Wakeup routine (`wakeup.md`) — runs when task queue is empty
- [x] Unread message injection into wakeup prompt (auto)
- [x] Cron scheduling (`cron add/list/delete/enable/disable`)
- [x] Docker sandbox
- [x] `task stop-all` — emergency halt for running/pending tasks
- [x] `task tokens` — token usage and cost per agent/task

### Agent playbooks ✓
- [x] `agent-playbooks/` directory in agency structure
- [x] `playbook:` field in `project.yaml` AgentSpec
- [x] `project apply` installs playbook as `wakeup.md` + sets `wakeup_prompt`
- [x] Playbooks included in template archives

### Project blueprints ✓
- [x] `project.yaml` — declarative agents + heartbeats + crons + playbooks
- [x] `project show / apply / blueprints`
- [x] `create project --blueprint`

### Templates ✓
- [x] `template pack` — archive agency as shareable `.tar.gz`
- [x] `template info` — inspect metadata
- [x] `create agency --template` — local file, directory, or HTTPS URL
- [x] `template.json` metadata (name, version, author, email, description, keywords)
- [x] `agent-playbooks/` and `project-blueprints/` included in template archives

### Planned
- [ ] `depends_on` task dependency resolution
- [ ] Run log rotation
- [ ] E2B / Daytona sandbox provider

---

## License

MIT
