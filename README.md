# agencycli

**agencycli** is a CLI tool for managing AI agent teams through an agency-style organisational structure. Define teams, roles, projects, and workflows once — hire agents with fully assembled context, assign them tasks, and let them work autonomously on a heartbeat schedule.

No more copy-pasting prompts between sessions. No more context drift. No more manually wiring up who does what next.

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

Agents work autonomously via a **heartbeat loop** — wake up, process all pending tasks, sleep — with conversation continuity preserved across cycles. Complex multi-agent flows are coordinated by an **async workflow engine**: each agent does its part and calls `task done --summary "..."`, the engine routes the result to the next agent automatically.

---

## How it works

### Layer 1 — Context management

- Define your agency once: teams, roles, skills, and how they compose
- `hire` an agent → agencycli merges the full chain and writes a ready-to-use working directory
- Supported models: `claudecode`, `codex`, `gemini`, `cursor`, `qoder`, `opencode`, `iflow`, `generic-cli`
- `sync` propagates prompt/skill changes to all affected agents

### Layer 2 — Task automation

- Per-agent **task queues** with a 7-state lifecycle
- **Heartbeat daemon**: non-overlapping wakeup loop, session-preserving, with time-window scheduling (`active_hours`, `active_days`)
- **Cron jobs**: add recurring tasks on a crontab schedule
- **Human inbox**: agents route `awaiting_confirmation` tasks here; you confirm/reject/forward
- **Docker sandbox**: isolated containers with credentials and repos auto-mounted

### Layer 3 — Async workflow orchestration

Workflows are defined as **task templates + routing rules**, not sequential scripts. Each agent works at its own pace:

1. Agent A receives a task, completes it, calls `task done --summary "..."`
2. The routing engine reads the workflow definition, finds the matching route, and enqueues a task for Agent B — passing `{{task.summary}}` and any variables
3. Agent B picks it up on its next heartbeat wakeup
4. Final results route to the human inbox for approval

No orchestrator is blocked waiting. Agents are fully decoupled.

### Layer 4 — Templates

Package an entire agency (teams, roles, skills, workflows, project blueprints) as a `.tar.gz` template. Share it, apply it to a new agency with one command.

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

Templates bundle teams, roles, skills, workflows, and project blueprints in one archive:

```bash
# 1. Create an agency from a shared template
agencycli create agency --name "MyAgency" \
  --template https://example.com/tech-agency.tar.gz
cd MyAgency

# 2. List the project blueprints the template ships with
agencycli project blueprints

# 3. Create a project — project.yaml is pre-filled with agents, heartbeats, workflows
agencycli create project --name "my-service" --blueprint default

# 4. Review what will be created
agencycli project show --project my-service

# 5. Apply: hire all agents + configure heartbeats + crons in one command
agencycli project apply --project my-service

# 6. Start the daemon — agents now wake up on schedule
agencycli daemon start

# 7. Kick off a workflow
agencycli workflow run feature-dev --project my-service \
  --input feature="User login" \
  --input background="Auth sprint Q2"

# 8. Monitor
agencycli workflow instances --project my-service
agencycli workflow status <instance-id> --project my-service
agencycli inbox list       # human confirmations arrive here
```

## Quick start — from scratch

```bash
# 1. Create the workspace
agencycli create agency --name "MyAgency" --desc "Building great software"
cd MyAgency

# 2. Create teams and roles
agencycli create team --name "engineering"
agencycli create role --team "engineering" --name "developer"
agencycli create role --team "engineering" --name "qa-engineer"

# 3. Write a workflow definition
mkdir -p workflows
# edit workflows/feature-dev.yaml (see Workflow section below)

# 4. Create a project blueprint
mkdir -p project-blueprints
# edit project-blueprints/default.yaml

# 5. Create and apply a project
agencycli create project --name "my-api" --blueprint default
agencycli project apply  --project my-api

# 6. Start daemon
agencycli daemon start
```

---

## Workspace layout

```
MyAgency/
  .agencycli/
    agency.yaml              # workspace metadata
    inbox.yaml               # human inbox (auto-managed)
    inbox.md                 # human-readable inbox (auto-generated)
  agency-prompt.md           # agency-wide context

  teams/
    engineering/
      team.yaml
      prompt.md
      roles/
        developer/
          role.yaml          # skills[], setup (dirs/files to create)
          prompt.md          # role-specific context layer
        qa-engineer/
          role.yaml
          prompt.md

  skills/                    # no built-ins — define only what you need
    github-push-relay/
      skill.yaml
      prompt.md              # uses {{SKILL_DIR}} for script paths
      git-push-github.sh     # bundled file, chmod+x preserved

  workflows/                 # async workflow definitions (shared)
    feature-dev.yaml

  project-blueprints/        # project templates packaged with the agency template
    default.yaml             # declares agents, heartbeats, and active workflows

  projects/
    my-api/
      project.yaml           # declarative: agents + heartbeats + crons + workflows
      prompt.md              # project-specific context
      workflows/             # project-specific workflow overrides
      workflow-runs/         # runtime workflow instance state
        wf-20260317-abc123.yaml
      agents/
        dev/
          .agencycli-agent.yaml  # model, team, role, sandbox, add_dirs
          CLAUDE.md              # merged context (@imports all layers)
          tasks.yaml             # active task queue
          tasks_archive.yaml     # completed tasks
          heartbeat.yaml         # wakeup config (set by project apply)
          crons.yaml             # scheduled tasks (set by project apply)
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

# Show project.yaml (agents, heartbeats, workflows)
agencycli project show --project my-api

# One-command bootstrap: hire all agents + configure heartbeats/crons
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
  - name: qa
    role: qa-engineer
    team: engineering
    model: claudecode
    sandbox: true
    heartbeat:
      enabled: true
      interval: 1h
workflows:
  - feature-dev
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

### `workflow` — async orchestration

```bash
agencycli workflow list [--project P]
agencycli workflow show <name> [--project P]
agencycli workflow run  <name> --project P [--input key=value ...]
agencycli workflow instances --project P [--status running|done|failed]
agencycli workflow status <instance-id> --project P
```

**`workflows/feature-dev.yaml`** example:

```yaml
name: feature-dev
version: "1.0"
description: "Dev implements a feature, QA reviews, human approves"

templates:
  - id: implement
    title: "Implement: {{inputs.feature}}"
    agent: dev
    prompt: |
      Implement the feature: {{inputs.feature}}
      Background: {{inputs.background}}
      When done: agencycli task done --id $TASK_ID --status success \
        --summary "Brief description of what was done"

  - id: review
    title: "Review: {{inputs.feature}}"
    agent: qa
    prompt: |
      Review the implementation of: {{inputs.feature}}
      Dev summary: {{task.summary}}
      When done: agencycli task done --id $TASK_ID --status success \
        --summary "QA PASS/FAIL: ..."

entry:
  template: implement

routes:
  - on:
      template: implement
      status: success
    create:
      template: review

  - on:
      template: review
      status: success
    inbox:
      title: "Approve: {{inputs.feature}}"
      summary: "Dev: {{steps.implement.summary}}\nQA: {{task.summary}}"
      action_items:
        - "Review the changes"
        - "Confirm ready to ship"

  - on:
      template: review
      status: failed
      max_trigger: 3         # circuit-breaker
    create:
      template: implement
      vars:
        background: "QA failed: {{task.error}} — fix and retry"
```

**Variable interpolation** in prompts and `vars`:

| Placeholder | Value |
|-------------|-------|
| `{{inputs.KEY}}` | `--input key=value` passed to `workflow run` |
| `{{task.summary}}` | what the previous agent reported via `--summary` |
| `{{task.error}}` | error from a failed task |
| `{{steps.TEMPLATE_ID.summary}}` | output of any earlier completed step |
| `{{task.vars.KEY}}` | vars inherited from the triggering task |

### `task` — task queue

```bash
agencycli task add    --project P --agent A --title "T" --prompt "..." \
                      [--type feature|bug|chore] [--priority 0-3]
agencycli task list   --project P --agent A [--status pending] [--archived]
agencycli task show   <task-id>
agencycli task cancel <task-id>
agencycli task retry  <task-id>

# Called by the agent inside its prompt:
agencycli task done --id <id> --status success --summary "what was done"
agencycli task done --id <id> --status failed  --error "reason"

# Route to human inbox (called by agent):
agencycli task confirm-request --id <id> --summary "PR ready" \
  --action-item "Review the diff" \
  --action-item "Approve or request changes"
```

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

### `inbox` — human confirmations

```bash
agencycli inbox list
agencycli inbox show    <task-id>         # shows summary, action items, log tail
agencycli inbox confirm <task-id> [--message "notes for the agent"]
agencycli inbox reject  <task-id> --reason "..."
agencycli inbox comment <task-id> --message "..."
agencycli inbox forward <task-id> --to <project>/<agent> --note "..."
```

### `daemon` — heartbeat scheduler

The heartbeat is a **non-overlapping wakeup loop**: after each cycle completes all pending tasks, the agent sleeps for `interval`, then wakes again. Session continuity is preserved across cycles.

```bash
# Configure heartbeat for one agent
agencycli daemon heartbeat --project P --agent A \
  --enable --interval 30m \
  --active-hours "09:00-18:00" \  # only wake in this window (local time)
  --active-days  "weekdays"       # Mon–Fri only (or Mon,Wed,Fri / weekends)

# Start daemon (all enabled agents)
agencycli daemon start
agencycli daemon stop
agencycli daemon status
```

Overnight ranges like `22:00-06:00` are supported. Outside the active window the daemon shows `⏸ outside active window — next wakeup in Xh`.

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

Crons enqueue a new task each time the schedule fires. The daemon checks for due crons on every heartbeat wakeup.

### `template` — share agencies

```bash
# Pack the current agency as a shareable template
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

A template archive contains: `agency-prompt.md`, `teams/`, `skills/`, `workflows/`, `project-blueprints/`.  
A `template.json` in the archive root holds metadata (similar to `npm package.json`).

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
agencycli --dir /path/to/MyAgency workflow run feature-dev --project my-api --input ...
agencycli --dir /path/to/MyAgency inbox list
agencycli --dir /path/to/MyAgency daemon start
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
- [x] Task queue per agent (7-state machine)
- [x] Human inbox with confirm / reject / comment / forward
- [x] `run` / `exec`
- [x] Session continuity across heartbeat cycles
- [x] Heartbeat daemon with active-hours / active-days windows
- [x] Cron scheduling (`cron add/list/delete/enable/disable`)
- [x] Docker sandbox
- [x] `on_success` triggers

### Workflow orchestration ✓
- [x] Async workflow engine: task templates + routing rules
- [x] Variable interpolation (`{{inputs.*}}`, `{{task.summary}}`, `{{steps.*}}`)
- [x] Circuit-breaker (`max_trigger` per route)
- [x] `workflow run / list / show / instances / status`
- [x] `task done --summary` — agent output passed to next step
- [x] Routing on `success`, `failed`, or `any`
- [x] Inbox routing from workflows

### Project blueprints ✓
- [x] `project.yaml` — declarative agents + heartbeats + crons + workflows
- [x] `project show / apply / blueprints`
- [x] `create project --blueprint`

### Templates ✓
- [x] `template pack` — archive agency as shareable `.tar.gz`
- [x] `template info` — inspect metadata
- [x] `create agency --template` — local file, directory, or HTTPS URL
- [x] `template.json` metadata (name, version, author, email, description, keywords)
- [x] `project-blueprints/` included in template archives

### Planned
- [ ] `depends_on` task dependency resolution
- [ ] Run log rotation
- [ ] Workflow cron triggers (auto-start workflows on schedule)
- [ ] E2B / Daytona sandbox provider

---

## License

MIT
