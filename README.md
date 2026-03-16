# agencycli

**agencycli** is a CLI tool for managing AI agent context and workflow through an agency-style organisational structure. It lets you define teams, projects, and skills once — then hire agents into projects with fully assembled context, assign them tasks, and automate their work cycles.

No more copy-pasting prompts between sessions. No more context drift between agents working on the same project.

## How it works

agencycli models your AI workflow as an agency with two layers:

**Layer 1 — Context Management** (fully implemented)

- **Agency** — global rules and values shared by every agent
- **Teams** — capability groups (engineering, growth, product…) with their own standards and skills. Teams can be nested (`engineering/backend`)
- **Projects** — concrete products or initiatives with their own goals and tech stack
- **Hire / Assign** — assign an agent to a project. agencycli merges the full context chain (agency → team → project) and writes it into an agent working directory ready to use

**Layer 2 — Workflow Automation** (Phase 1 implemented)

- **Tasks** — per-agent task queues with a 7-state lifecycle; agents can call back into agencycli to update task state
- **Inbox** — human confirmation inbox; agents route `awaiting_confirmation` tasks here
- **Heartbeat** — blocking wakeup loop: agent wakes, processes all pending tasks in one session, sleeps until next cycle. No overlap, session-preserving
- **Daemon** — runs heartbeat loops for all enabled agents in the background

```
agency-prompt.md
  └─ teams/engineering/prompt.md
       └─ teams/engineering/backend/prompt.md
            └─ projects/my-api/prompt.md
                 └─ projects/my-api/agents/dev/   ← hire produces this
                      ├─ CLAUDE.md                ← ready for `claude`
                      ├─ tasks.yaml               ← task queue
                      ├─ heartbeat.yaml           ← wakeup config
                      └─ .claude/skills/          ← skills auto-loaded
```

## Supported agents

| `--model`     | Output files                                                      |
|---------------|-------------------------------------------------------------------|
| `claudecode`  | `CLAUDE.md` with `@import` layers + `.claude/skills/`            |
| `codex`       | `AGENTS.md` single merged file (skills inlined)                  |
| `cursor`      | `.cursorrules` + `.cursor/rules/agencycli.mdc`                   |
| `gemini`      | `GEMINI.md` with `@import` layers + `.gemini/skills/`            |
| `qoder`       | `AGENTS.md` single merged file                                   |
| `opencode`    | `OPENCODE.md` single merged file                                 |
| `iflow`       | `IFLOW.md` single merged file                                    |
| `generic-cli` | `context.md` plain text                                          |

## Installation

```bash
go install github.com/agencycli/agencycli/cmd/agencycli@latest
```

Or build from source:

```bash
git clone https://github.com/agencycli/agencycli
cd agencycli
make install
```

## Quick start

```bash
# 1. Create an agency workspace
agencycli create agency --name "MyAgency" --desc "Building great software"
cd MyAgency

# 2. Create teams
agencycli create team --name "engineering" --desc "Software engineering"
agencycli create team --name "engineering/backend" --desc "Go/gRPC services"
agencycli create team --name "qa" --desc "Quality assurance"

# 3. Edit team prompts (add your standards and conventions)
vim teams/engineering/prompt.md
vim teams/engineering/backend/prompt.md

# 4. Add custom skills (no built-ins — define what you actually need)
mkdir -p skills/docker
# write skills/docker/skill.yaml and skills/docker/prompt.md

# 5. Create a project
agencycli create project --name "my-api" --desc "REST API service" --repo "../my-api"
vim projects/my-api/prompt.md

# 6. Hire agents (hire and assign are identical)
agencycli hire   --project "my-api" --team "engineering/backend" --model "claudecode" --name "dev"
agencycli assign --project "my-api" --team "qa"                  --model "claudecode" --name "reviewer"

# 7. Start working manually
cd projects/my-api/agents/dev
claude

# 8. Or assign tasks and automate
agencycli task add \
  --project my-api --agent dev \
  --title "Implement rate limiting" --type feature --priority 1 \
  --prompt "Add token-bucket rate limiting to the /api/v1 endpoints..."

agencycli run --project my-api --agent dev     # run once manually
agencycli daemon heartbeat --project my-api --agent dev --enable --interval 30m
agencycli daemon start                         # start heartbeat loop
```

## Workspace layout

```
MyAgency/
  .agencycli/
    agency.yaml              # agency metadata
    inbox.yaml               # human confirmation queue (auto-managed)
    inbox.md                 # human-readable inbox (auto-generated)
  agency-prompt.md           # agency-wide context

  teams/
    engineering/
      team.yaml
      prompt.md
      backend/
        team.yaml
        prompt.md
    qa/
      team.yaml
      prompt.md

  projects/
    my-api/
      project.yaml
      prompt.md
      agents/
        dev/                   # claudecode agent working directory
          .agencycli-agent.yaml
          CLAUDE.md            # @imports all context layers
          tasks.yaml           # task queue
          tasks_archive.yaml   # completed tasks
          heartbeat.yaml       # wakeup config + session ID
          crons.yaml           # scheduled tasks (Phase 2)
          runs/                # execution logs
            20260316-090000-t-xxx.log
          .agencycli-context/  # individual layer files (managed)
          .claude/skills/
        reviewer/
          CLAUDE.md
          tasks.yaml
          heartbeat.yaml

  skills/                    # custom skills (none pre-installed)
    docker/
      skill.yaml
      prompt.md
```

## Commands

### Context management

#### `agencycli create agency`

Initialise a new workspace directory.

```
agencycli create agency --name <name> [--desc <description>]
```

Creates the workspace directory and generates template prompt files. No built-in skills are pre-installed — define only the skills your agents actually need.

#### `agencycli create team`

Create a team. Supports nested paths.

```
agencycli create team --name <path> [--desc <description>] [--skills <skill1,skill2>]
```

Parent teams must exist first. agencycli enforces the full chain so every level has its own `prompt.md`.

#### `agencycli create project`

```
agencycli create project --name <name> [--desc <description>] [--repo <path>]
```

#### `agencycli hire` / `agencycli assign`

Assemble context and create an agent working directory. `hire` and `assign` are identical.

```
agencycli hire \
  --project <project-name> \
  --team    <team-path> \
  --model   <claudecode|codex|cursor|gemini|qoder|opencode|iflow|generic-cli> \
  --name    <agent-name> \
  [--extra-prompt <file>] \
  [--force]
```

Context is assembled in order: `agency → team chain → project → extra-prompt`.

#### `agencycli sync`

Regenerate agent working directories whose context has changed.

```bash
agencycli sync                               # sync all agents
agencycli sync --project my-api             # sync one project
agencycli sync --project my-api --name dev  # sync one agent
agencycli sync --force                       # force-regenerate everything
```

#### `agencycli list` / `agencycli show`

```bash
agencycli list teams
agencycli list projects
agencycli list agents
agencycli list skills

agencycli show team    engineering/backend
agencycli show project my-api
agencycli show agent   my-api dev
agencycli show agent   my-api dev --raw   # print full merged context
```

### Task management

#### `agencycli task`

```bash
# Add a task
agencycli task add \
  --project my-api --agent dev \
  --title "Fix login redirect" --type bug --priority 1 \
  --prompt "The redirect is broken on mobile. Fix and open a PR."

# Add a task for human decision
agencycli task add \
  --project my-api --agent pm \
  --title "Scope AI search" --assignee human \
  --prompt "Is AI search in scope for Q2?"

# List / inspect
agencycli task list --project my-api --agent dev
agencycli task list --project my-api --agent dev --status pending
agencycli task show <task-id> --project my-api --agent dev

# Mark done (called by the agent itself inside its prompt)
agencycli task done --id <task-id> --status success
agencycli task done --id <task-id> --status failed --error "reason"

# Route to human inbox (called by the agent)
agencycli task confirm-request --id <task-id> --summary "Need your decision on..."

# Retry / cancel
agencycli task retry  <task-id>
agencycli task cancel <task-id> [--reason "..."]
```

**Task states:**

```
pending → in_progress → done_success
                      → done_failed  (auto-retry up to max_retries)
                      → awaiting_confirmation → in_progress (after human confirm)
                                              → cancelled   (after human reject)
                      → blocked
```

#### `agencycli run`

Manually execute the next pending task (or a specific task) for an agent:

```bash
agencycli run --project my-api --agent dev
agencycli run --project my-api --agent dev --task <task-id>
agencycli run --project my-api --agent dev --dry-run
```

### Human inbox

Tasks routed to `--assignee human` or via `task confirm-request` appear in the inbox:

```bash
agencycli inbox                          # list all items
agencycli inbox show    <task-id>        # view detail + log path
agencycli inbox confirm <task-id>        # approve → agent resumes
agencycli inbox confirm <task-id> --message "Check line 42 specifically"
agencycli inbox reject  <task-id> --reason "false positive"
agencycli inbox comment <task-id> --message "..."   # add note, task stays pending
```

The inbox is also rendered as `inbox.md` at the workspace root — open it in any editor.

### Heartbeat scheduler

The heartbeat is a **blocking, non-overlapping wakeup loop**: after each cycle completes, the agent sleeps for `interval`, then wakes again. All tasks in one cycle share the same agent session (conversation continuity).

```bash
# Configure heartbeat for an agent
agencycli daemon heartbeat \
  --project my-api --agent dev \
  --enable --interval 30m

# Start the daemon (watches all enabled agents)
agencycli daemon start

# Show heartbeat status
agencycli daemon heartbeat --project my-api --agent dev
```

Heartbeat vs Cron:

| | Heartbeat | Cron (Phase 2) |
|-|-----------|----------------|
| Trigger | N minutes after last completion | Exact calendar time |
| Overlap | Never (PID-checked) | N/A (just enqueues a task) |
| Session | All tasks in one cycle share a session | New task each time |
| Use case | "Check for work and do it" | "Run at 9am every Monday" |

### Session management

Agents maintain conversation history across heartbeat cycles via session IDs. For Claude Code the session ID is captured automatically from `--output-format stream-json` output. For other models it can be set manually.

```bash
agencycli session show  --project my-api --agent dev
agencycli session set   --project my-api --agent dev --id abc123
agencycli session reset --project my-api --agent dev   # start fresh next cycle
```

### Global flag: `--dir`

Run any command against a workspace outside the current directory:

```bash
agencycli --dir /path/to/MyAgency list agents
agencycli --dir /path/to/MyAgency sync
agencycli --dir /path/to/MyAgency daemon heartbeat --project cc-connect --agent dev --enable --interval 1h
```

## Skills

Skills are reusable capability definitions that get injected into an agent's context. There are **no pre-installed skills** — you define exactly what your agents need.

Create a skill:

```
skills/
  github-pr-review/
    skill.yaml       # name + description
    prompt.md        # instructions (e.g. how to review PRs, when to approve/reject)
  docker-deploy/
    skill.yaml
    prompt.md
```

Reference skills in a team:

```bash
agencycli create team --name "qa" --skills "github-pr-review"
```

Or add directly in `teams/qa/team.yaml`:

```yaml
skills:
  - github-pr-review
  - docker-deploy
```

Then run `agencycli sync` to propagate to all agents in that team.

## Workflow: agent callbacks

Agents signal state changes back to agencycli by running CLI commands from inside their prompt. agencycli injects these instructions automatically as a footer in every task prompt:

```bash
# Signal completion
agencycli task done --id <task-id> --status success

# Route to human inbox
agencycli task confirm-request --id <task-id> --summary "PR #42 has a security concern"

# Signal failure
agencycli task done --id <task-id> --status failed --error "reason"
```

This works because the agent's working directory is always on `PATH` and the workspace is auto-discovered from the working directory.

## Context inheritance

```
agency                ← every agent everywhere
  └─ team             ← shared by all agents in this team
       └─ sub-team    ← nested capability group
            └─ project ← project goals, tech stack, conventions
```

Skills are collected across the full team chain (deduplication applied). A skill at `engineering` is inherited by `engineering/backend`.

## Roadmap

### Context management
- [x] Agency / team / project scaffolding
- [x] Context merging with team chain inheritance
- [x] Claude Code formatter (`CLAUDE.md` + `@import` + skills)
- [x] Codex / Qoder formatter (`AGENTS.md` single file)
- [x] Cursor formatter (`.cursorrules` + `.cursor/rules/`)
- [x] Gemini CLI formatter (`GEMINI.md` + `@import` + skills)
- [x] OpenCode / iFlow formatter (single file)
- [x] Generic CLI formatter (`context.md`)
- [x] `sync` with SHA-256 change detection
- [x] `assign` alias for `hire`
- [x] `--dir` flag for remote workspace access

### Workflow (Phase 1 — complete)
- [x] Task queue per agent (`tasks.yaml`, 7-state machine)
- [x] Human inbox (`inbox.yaml` + `inbox.md`)
- [x] `task add/list/show/done/confirm-request/retry/cancel`
- [x] `run` — manual one-shot task execution
- [x] `inbox confirm/reject/comment`
- [x] `session set/reset/show` — conversation continuity
- [x] Heartbeat daemon — non-overlapping wakeup loop, session-preserving
- [x] Agent callback protocol (`AGENCYCLI_AWAIT_CONFIRM:` sentinel)
- [x] `on_success` triggers — auto-create tasks in other agents on completion

### Workflow (Phase 2 — planned)
- [ ] Cron scheduling (`crons.yaml` + `agencycli cron add/list/enable`)
- [ ] `depends_on` — task dependency resolution
- [ ] Run log rotation

### Sandbox (planned)
- [ ] Docker sandbox (`--sandbox docker`) — isolated container per agent run
- [ ] Pre-built images: `agencycli/sandbox-claudecode`, `agencycli/sandbox-codex`
- [ ] Credential mounting (`~/.claude`, `~/.config/gh`, `~/.ssh`)
- [ ] `agencycli sandbox test/show/set`
- [ ] Devcontainer generation (`agencycli sandbox devcontainer`)
- [ ] E2B / Daytona provider support

See [`docs/sandbox-design.md`](docs/sandbox-design.md) for the full sandbox research and design.

## License

MIT
