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

**Workflows** define how tasks flow between agents asynchronously. When one agent finishes and calls `task done --summary "..."`, the routing engine creates the next task for the next agent — no orchestrator is blocked waiting.

**Project blueprints** declare which agents a project needs, their heartbeat schedule, and which workflows are active. Running `project apply` hires every agent and wires up all schedules in one command.

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
BLUEPRINT  AGENTS  WORKFLOWS
─────────  ──────  ─────────
default    2       feature-dev
```

### Step 3 — Create a project from a blueprint

```bash
agencycli create project --name "my-service" --blueprint default \
  --desc "My REST API service"
```

This writes `projects/my-service/project.yaml` pre-filled with agent definitions, heartbeat schedules, and workflow references.

### Step 4 — Review the project configuration

```bash
agencycli project show --project my-service
```

You will see each agent, its model, role, sandbox setting, and heartbeat schedule. Edit `projects/my-service/project.yaml` if you want to adjust anything before applying.

### Step 5 — Apply: hire agents and configure schedules

```bash
agencycli project apply --project my-service
```

This single command:
- Hires every agent declared in `project.yaml`
- Writes `heartbeat.yaml` for agents with a heartbeat schedule
- Writes `crons.yaml` for agents with cron jobs
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

Agents now wake up automatically on their heartbeat schedule, process pending tasks, and sleep again. Session continuity is preserved across cycles.

### Step 8 — Run a workflow

```bash
agencycli workflow run feature-dev --project my-service \
  --input feature="User login with OAuth" \
  --input background="Auth sprint, due end of Q2"
```

The entry task is enqueued for the first agent. When it completes (calling `task done --summary "..."`), the routing engine automatically enqueues the next task for the next agent.

### Step 9 — Monitor progress

```bash
agencycli workflow instances --project my-service
agencycli workflow status <instance-id> --project my-service
agencycli inbox list                  # human confirmations arrive here
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

### Step 5 — Write a workflow definition

```bash
mkdir -p workflows
```

```yaml
# workflows/feature-dev.yaml
name: feature-dev
version: "1.0"
description: "Dev implements, QA reviews, human approves"

templates:
  - id: implement
    title: "Implement: {{inputs.feature}}"
    agent: dev
    prompt: |
      Feature request: {{inputs.feature}}
      Background: {{inputs.background}}

      Implement the feature and open a PR, then call:
        agencycli --dir $AGENCY_DIR task done --id $TASK_ID \
          --status success --summary "PR #<number> opened: <description>"

  - id: review
    title: "Review: {{inputs.feature}}"
    agent: qa
    prompt: |
      Review the implementation of: {{inputs.feature}}
      Dev summary: {{task.summary}}

      Review the PR, then call:
        agencycli --dir $AGENCY_DIR task done --id $TASK_ID \
          --status success --summary "QA PASS: <what was verified>"

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
        - "Review the PR linked in the dev summary"
        - "Confirm it is ready to merge"

  - on:
      template: review
      status: failed
      max_trigger: 3
    create:
      template: implement
      vars:
        background: "Previous attempt failed QA: {{task.error}} — fix and retry"
```

**Variable placeholders:**

| Placeholder | Value |
|-------------|-------|
| `{{inputs.KEY}}` | `--input key=value` from `workflow run` |
| `{{task.summary}}` | agent's `--summary` from `task done` |
| `{{task.error}}` | error from a failed task |
| `{{steps.TEMPLATE_ID.summary}}` | summary from an earlier completed step |

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
    heartbeat:
      enabled: true
      interval: 30m
      active_hours: "09:00-20:00"  # only wake in this window (local time)
      active_days: weekdays         # Mon–Fri only

  - name: qa
    role: qa-engineer
    team: engineering
    model: claudecode
    sandbox: true
    heartbeat:
      enabled: true
      interval: 1h
      active_hours: "09:00-20:00"
      active_days: weekdays

workflows:
  - feature-dev
```

### Step 7 — Create and apply a project

```bash
# Create the project (writes project.yaml from blueprint)
agencycli create project --name "my-api" --blueprint default \
  --desc "REST API service" --repo "/absolute/path/to/my-api"

# Review the generated config
agencycli project show --project my-api

# Apply: hire agents + configure heartbeats
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

### Step 9 — Run a workflow

```bash
agencycli workflow run feature-dev --project my-api \
  --input feature="Rate limiting" \
  --input background="Security hardening sprint"
```

---

## Working with tasks directly (without workflows)

```bash
# Add a task manually
agencycli task add \
  --project my-api --agent dev \
  --title "Fix login redirect" --type bug --priority 1 \
  --prompt "The redirect is broken on mobile Safari. Fix and open a PR."

# Run manually
agencycli run --project my-api --agent dev

# Or run a quick one-off prompt
agencycli exec --project my-api --agent dev \
  --prompt "Run gh issue list and summarise all open issues"
```

---

## Cron jobs

Add recurring tasks to any agent:

```bash
agencycli cron add \
  --project my-api --agent dev \
  --title "Weekly dependency audit" \
  --schedule "0 9 * * 1" \
  --prompt "Run `go list -m -u all` and open a PR updating any outdated dependencies."

agencycli cron list --project my-api --agent dev
```

The cron definition is also declarable in `project.yaml` under `agents[*].crons`, so it is applied automatically by `project apply`.

---

## Human inbox

Agents can route tasks to you for decisions or approvals:

```bash
agencycli inbox list
agencycli inbox show    <task-id>   # full prompt, action items, log tail
agencycli inbox confirm <task-id> --message "Approved — merge when CI passes"
agencycli inbox reject  <task-id> --reason "Out of scope for this sprint"
agencycli inbox comment <task-id> --message "Check the auth module specifically"
agencycli inbox forward <task-id> --to my-api/dev --note "Please re-check the edge case"
```

The inbox is also rendered to `inbox.md` at the workspace root.

---

## Heartbeat time windows

Restrict when an agent can wake up:

```bash
agencycli daemon heartbeat \
  --project my-api --agent dev \
  --enable --interval 30m \
  --active-hours "09:00-18:00" \   # only between 9am and 6pm
  --active-days  "weekdays"         # Mon–Fri only
```

Supported `--active-days` values: `weekdays`, `weekends`, or `Mon,Tue,Wed,Thu,Fri,Sat,Sun` (comma-separated).

Overnight windows like `22:00-06:00` work correctly.

---

## Templates — share your agency

Pack your agency (teams, roles, skills, workflows, project-blueprints) as a shareable archive:

```bash
agencycli template pack --output tech-agency.tar.gz \
  --name "tech-project" --version "1.0.0" \
  --author "Alice" --email "alice@example.com" \
  --description "Standard software engineering agency template" \
  --keywords "engineering,software,go"
```

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

## --dir flag

Run any command against a workspace outside your current directory:

```bash
agencycli --dir /path/to/MyAgency list agents
agencycli --dir /path/to/MyAgency workflow run feature-dev --project my-api --input ...
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
| Tasks | `task add/list/show/done/retry/cancel/confirm-request` |
| Inbox | `inbox list/show/confirm/reject/comment/forward` |
| Workflow | `workflow list/show/run/instances/status` |
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

[ ] mkdir -p workflows && write workflows/feature-dev.yaml
[ ] mkdir -p project-blueprints && write project-blueprints/default.yaml

[ ] agencycli create project --name "my-app" --blueprint default --repo /path/to/repo
[ ] Edit projects/my-app/prompt.md  (tech stack, build commands, PR conventions)
[ ] agencycli project apply --project my-app

[ ] agencycli daemon start

[ ] agencycli workflow run feature-dev --project my-app --input feature="First feature"
[ ] agencycli workflow instances --project my-app
[ ] agencycli inbox list
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
[ ] agencycli workflow run <name> --project my-app --input ...
```
