# agencycli

**agencycli** is a CLI tool for managing AI agent context and workflow through an agency-style organisational structure. It lets you define teams, roles, projects, and skills once — then hire agents into projects with fully assembled context, assign them tasks, and automate their work cycles.

No more copy-pasting prompts between sessions. No more context drift between agents working on the same project.

## How it works

agencycli models your AI workflow as an agency with two layers:

**Layer 1 — Context Management** (fully implemented)

- **Agency** — global rules and values shared by every agent
- **Teams** — capability groups (engineering, growth, product…) with their own standards and skills. Teams can be nested (`engineering/backend`)
- **Roles** — job functions within a team (go-developer, pr-reviewer, content-writer…). Each role has its own prompt, bound skills, and workspace setup
- **Projects** — concrete products or initiatives with their own goals and tech stack
- **Hire / Assign** — assign an agent to a project under a role. agencycli merges the full context chain (`agency → team → role → project`) and writes it into an agent working directory ready to use

**Layer 2 — Workflow Automation** (Phase 1 implemented)

- **Tasks** — per-agent task queues with a 7-state lifecycle; agents can call back into agencycli to update task state
- **Inbox** — human confirmation inbox; agents route `awaiting_confirmation` tasks here
- **Heartbeat** — blocking wakeup loop: agent wakes, processes all pending tasks in one session, sleeps until next cycle. No overlap, session-preserving
- **Daemon** — runs heartbeat loops for all enabled agents in the background
- **Exec** — one-shot prompt execution for quick tests without waiting for the task queue

```
agency-prompt.md
  └─ teams/engineering/prompt.md
       └─ teams/engineering/roles/go-developer/
       │    ├─ role.yaml          ← skills + setup
       │    └─ prompt.md          ← role-specific context
       └─ projects/my-api/prompt.md
            └─ projects/my-api/agents/dev/   ← hire produces this
                 ├─ CLAUDE.md                ← ready for `claude`
                 ├─ tasks.yaml               ← task queue
                 ├─ heartbeat.yaml           ← wakeup config
                 └─ .claude/skills/          ← skills auto-loaded
                      └─ github-push-relay/
                           ├─ prompt.md
                           └─ git-push-github.sh  ← bundled script
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
go install github.com/chenhg5/agencycli/cmd/agencycli@latest
```

Or build from source:

```bash
git clone https://github.com/chenhg5/agencycli
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

# 4. Create roles within teams
agencycli create role --team "engineering" --name "go-developer" --desc "Implements Go services"
agencycli create role --team "qa"          --name "pr-reviewer"  --desc "Reviews and approves PRs"
# Edit teams/engineering/roles/go-developer/prompt.md

# 5. Bind skills to roles
agencycli role skill add --team "engineering" --role "go-developer" --skill "github-push-relay"

# 6. Add custom skills (no built-ins — define what you actually need)
mkdir -p skills/github-push-relay
# write skills/github-push-relay/skill.yaml and skills/github-push-relay/prompt.md
# add any bundled scripts alongside prompt.md (e.g. git-push-github.sh)

# 7. Create a project
agencycli create project --name "my-api" --desc "REST API service" --repo "../my-api"
vim projects/my-api/prompt.md

# 8. Hire agents with a role (hire and assign are identical)
agencycli hire \
  --project "my-api" --team "engineering" --role "go-developer" \
  --model "claudecode" --name "dev"
agencycli hire \
  --project "my-api" --team "qa" --role "pr-reviewer" \
  --model "claudecode" --name "reviewer"

# 9. Start working manually
cd projects/my-api/agents/dev
claude

# 10. Or run a quick one-off prompt to test the agent
agencycli exec --project my-api --agent dev --prompt "List all open GitHub issues"

# 11. Or assign tasks and automate
agencycli task add \
  --project my-api --agent dev \
  --title "Implement rate limiting" --type feature --priority 1 \
  --prompt "Add token-bucket rate limiting to the /api/v1 endpoints..."

agencycli run --project my-api --agent dev          # run once manually
agencycli daemon heartbeat --project my-api --agent dev --enable --interval 30m
agencycli daemon start                               # start heartbeat loop
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
      roles/
        go-developer/
          role.yaml          # description, skills[], setup (dirs/files)
          prompt.md          # role-specific context layer
        release-engineer/
          role.yaml
          prompt.md
      backend/               # nested sub-team
        team.yaml
        prompt.md
    qa/
      team.yaml
      prompt.md
      roles/
        pr-reviewer/
          role.yaml
          prompt.md

  projects/
    my-api/
      project.yaml
      prompt.md
      agents/
        dev/                   # claudecode agent working directory
          .agencycli-agent.yaml  # model, team, role, add_dirs, sandbox…
          CLAUDE.md            # @imports all context layers
          tasks.yaml           # task queue
          tasks_archive.yaml   # completed tasks
          heartbeat.yaml       # wakeup config + session ID
          runs/                # execution logs
            20260316-090000-t-xxx.log
          .agencycli-context/  # individual layer files (managed)
            agency.md
            team-engineering.md
            role-engineering-go-developer.md
            project-my-api.md
          .claude/skills/
            github-push-relay/
              prompt.md        # skill instructions ({{SKILL_DIR}} resolved)
              git-push-github.sh
          scratch/             # role setup dirs (auto-created on hire)
        reviewer/
          CLAUDE.md
          tasks.yaml
          heartbeat.yaml

  skills/                    # custom skills (none pre-installed)
    github-push-relay/
      skill.yaml             # name + description
      prompt.md              # uses {{SKILL_DIR}} for script paths
      git-push-github.sh     # bundled script, chmod+x preserved
    docker-deploy/
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

#### `agencycli create role`

Create a role under a team. Roles add a context layer between team and project and can bind skills and define workspace setup.

```
agencycli create role --team <team-path> --name <role-name> [--desc <description>]
```

After creation, edit `teams/<team>/roles/<role>/prompt.md` and `role.yaml` as needed.

#### `agencycli role`

Manage role definitions.

```bash
# List roles under a team (shows bound skills)
agencycli role list --team engineering

# Bind skills to a role
agencycli role skill add --team engineering --role go-developer --skill github-push-relay
agencycli role skill add --team growth --role content-writer --skill "article-publisher,seo-checker"

# Unbind skills from a role
agencycli role skill remove --team engineering --role go-developer --skill github-push-relay
```

After modifying role skills, run `agencycli sync` to propagate to already-hired agents.

#### `agencycli hire` / `agencycli assign`

Assemble context and create an agent working directory. `hire` and `assign` are identical.

```
agencycli hire \
  --project <project-name> \
  --team    <team-path> \
  --model   <claudecode|codex|cursor|gemini|qoder|opencode|iflow|generic-cli> \
  --name    <agent-name> \
  [--role   <role-name>] \
  [--sandbox docker] \
  [--force]
```

Context is assembled in order: `agency → team chain → role → project`.

When `--role` is specified:
- The role's `prompt.md` is injected as an additional context layer
- Role skills are merged with team skills
- Role workspace setup is applied (directories and files created automatically)

When the project defines a `repo` path, it is automatically added to the agent's `add_dirs`, giving the agent access to the repository alongside its working directory.

#### `agencycli fire`

Remove an agent from a project.

```bash
# Soft delete (default) — moves agent dir to agents/.fired/<name>-<timestamp>/
agencycli fire --project my-api --agent dev

# Hard delete — permanently removes the agent directory
agencycli fire --project my-api --agent dev --force
```

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

### Direct execution

#### `agencycli exec`

Run a one-shot prompt for an agent without going through the task queue. Useful for quick tests and ad-hoc commands.

```bash
agencycli exec \
  --project my-api \
  --agent   dev \
  --prompt  "Run gh issue list and summarise open issues"
```

Output streams to stdout. A run log is saved to the agent's `runs/` directory. Session continuity works the same as `run` (session ID is captured and reused on the next `exec` or `run`).

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
agencycli session clear --project my-api --agent dev   # reset to fresh session
```

### Docker sandbox

Agents can run in isolated Docker containers, with credentials and workspace directories automatically mounted.

```bash
# Hire an agent with sandbox enabled
agencycli hire \
  --project my-api --team engineering --role go-developer \
  --model claudecode --name dev \
  --sandbox docker

# Run a task in the sandbox
agencycli run   --project my-api --agent dev
agencycli exec  --project my-api --agent dev --prompt "..."
```

What the sandbox provides:

- Agent working directory mounted at the same absolute path as on the host (read/write)
- Project repository (`repo`) mounted at its host path (read/write)
- Agency workspace root mounted at its host path (read/write) — agents can call `agencycli` inside the container
- Host `agencycli` binary mounted read-only into `/usr/local/bin/agencycli`
- Credentials auto-mounted (`~/.claude`, `~/.claude.json`, `~/.config/gh`, `~/.ssh`, `~/.codex`, `~/.gemini`, `~/.cursor`) based on model
- Well-known API keys forwarded as environment variables
- Claude Code: runs as root with `IS_SANDBOX=1` + `--dangerously-skip-permissions`
- Codex: runs with `CODEX_UNSAFE_ALLOW_NO_SANDBOX=1`

Pre-built Dockerfiles are in `docker/`. Build with Chinese mirrors if needed:

```bash
docker build --build-arg CN_MIRROR=1 -t agencycli/sandbox-claudecode docker/sandbox-claudecode/
```

### Global flag: `--dir`

Run any command against a workspace outside the current directory:

```bash
agencycli --dir /path/to/MyAgency list agents
agencycli --dir /path/to/MyAgency sync
agencycli --dir /path/to/MyAgency role list --team engineering
agencycli --dir /path/to/MyAgency daemon heartbeat --project cc-connect --agent dev --enable --interval 1h
```

## Skills

Skills are reusable capability definitions that get injected into an agent's context. There are **no pre-installed skills** — you define exactly what your agents need.

### Skill structure

A skill directory can contain:
- `skill.yaml` — name and description (required)
- `prompt.md` — instructions injected into the agent's context (required)
- Any additional files (scripts, templates, configs) — all files are bundled and deployed alongside `prompt.md`

Use `{{SKILL_DIR}}` in `prompt.md` to reference bundled files by their absolute deployed path:

```markdown
<!-- skills/github-push-relay/prompt.md -->
Use the relay script at `{{SKILL_DIR}}/git-push-github.sh` to push to GitHub.
```

agencycli resolves `{{SKILL_DIR}}` to the actual deployed path when syncing.

### Binding skills to agents

Skills can be bound at the team or role level:

**Team skills** (`teams/qa/team.yaml`):
```yaml
skills:
  - github-pr-review
  - docker-deploy
```

**Role skills** (`teams/engineering/roles/go-developer/role.yaml`):
```yaml
name: go-developer
skills:
  - github-push-relay
```

Or manage role skills via CLI:
```bash
agencycli role skill add --team engineering --role go-developer --skill github-push-relay
agencycli role skill remove --team engineering --role go-developer --skill github-push-relay
```

All skills across the full team chain are merged (with deduplication) and deployed to each agent on `hire` or `sync`.

## Roles

Roles define the job function of an agent within a team. They sit between teams and projects in the context hierarchy.

### Role structure

```
teams/engineering/roles/go-developer/
  role.yaml    # description, skills, setup
  prompt.md    # role-specific instructions
```

`role.yaml` example:

```yaml
name: go-developer
description: Implements features and fixes bugs for Go projects
skills:
  - github-push-relay
setup:
  dirs:
    - scratch
    - notes
  files:
    - path: scratch/.gitkeep
      content: ""
```

### Creating and managing roles

```bash
# Create a role
agencycli create role --team engineering --name go-developer --desc "Go service developer"

# List roles in a team
agencycli role list --team engineering

# Add / remove skills
agencycli role skill add    --team engineering --role go-developer --skill docker-build
agencycli role skill remove --team engineering --role go-developer --skill docker-build
```

After changing role skills, run `agencycli sync` to update all hired agents with that role.

### Hiring with a role

```bash
agencycli hire \
  --project my-api \
  --team    engineering \
  --role    go-developer \
  --model   claudecode \
  --name    dev
```

The assembled context chain is: `agency → engineering → go-developer role → my-api project`

## Context inheritance

```
agency                ← every agent everywhere
  └─ team             ← shared by all agents in this team
       └─ sub-team    ← nested capability group (inherits from parent)
            └─ role   ← job function within the team (optional)
                 └─ project ← project goals, tech stack, conventions
```

Skills are collected across the full team chain and role (deduplication applied). A skill bound at `engineering` is inherited by `engineering/backend`.

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

## Roadmap

### Context management
- [x] Agency / team / project scaffolding
- [x] Role system (`teams/<team>/roles/<role>/`) with skills + workspace setup
- [x] Context merging with team chain inheritance (`agency → team → role → project`)
- [x] Claude Code formatter (`CLAUDE.md` + `@import` + skills)
- [x] Codex / Qoder formatter (`AGENTS.md` single file)
- [x] Cursor formatter (`.cursorrules` + `.cursor/rules/`)
- [x] Gemini CLI formatter (`GEMINI.md` + `@import` + skills)
- [x] OpenCode / iFlow formatter (single file)
- [x] Generic CLI formatter (`context.md`)
- [x] `sync` with SHA-256 change detection
- [x] `assign` alias for `hire`
- [x] `fire` — soft/hard delete of agents
- [x] `--dir` flag for remote workspace access
- [x] Skills with bundled files + `{{SKILL_DIR}}` placeholder resolution
- [x] `add_dirs` — pass project repo to agents as an additional working directory

### Workflow (Phase 1 — complete)
- [x] Task queue per agent (`tasks.yaml`, 7-state machine)
- [x] Human inbox (`inbox.yaml` + `inbox.md`)
- [x] `task add/list/show/done/confirm-request/retry/cancel`
- [x] `run` — manual one-shot task execution
- [x] `exec` — direct prompt execution for quick tests
- [x] `inbox confirm/reject/comment`
- [x] `session set/clear/show` — conversation continuity
- [x] Heartbeat daemon — non-overlapping wakeup loop, session-preserving
- [x] Agent callback protocol (`AGENCYCLI_AWAIT_CONFIRM:` sentinel)
- [x] `on_success` triggers — auto-create tasks in other agents on completion

### Workflow (Phase 2 — planned)
- [ ] Cron scheduling (`crons.yaml` + `agencycli cron add/list/enable`)
- [ ] `depends_on` — task dependency resolution
- [ ] Run log rotation

### Sandbox
- [x] Docker sandbox (`--sandbox docker`) — isolated container per agent run
- [x] Pre-built Dockerfiles: `docker/sandbox-claudecode/`, `docker/sandbox-codex/`
- [x] Credential mounting (`~/.claude`, `~/.config/gh`, `~/.ssh`, `~/.codex`, `~/.gemini`, `~/.cursor`)
- [x] Root execution bypass for Claude Code (`IS_SANDBOX=1`) and Codex (`CODEX_UNSAFE_ALLOW_NO_SANDBOX=1`)
- [x] Chinese mirror support for faster Docker builds (`--build-arg CN_MIRROR=1`)
- [ ] E2B / Daytona provider support

See [`docs/sandbox-design.md`](docs/sandbox-design.md) for the full sandbox research and design.

## License

MIT
