# agencycli — Installation & Setup Guide

This document is a complete guide for installing `agencycli` and building your first AI agent team. Hand this file to any AI coding agent (Claude Code, Codex, Gemini CLI, Cursor) and it can follow these instructions autonomously to get everything running.

---

## 1. Install agencycli

Choose **one** of the three methods below.

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

### Option C — Download a pre-built binary

Go to [https://github.com/chenhg5/agencycli/releases](https://github.com/chenhg5/agencycli/releases), download the archive for your platform, extract it, and move the binary to a directory on your `PATH`.

```bash
# Example: Linux amd64
curl -L https://github.com/chenhg5/agencycli/releases/latest/download/agencycli-v0.1.0-linux-amd64.tar.gz \
  | tar xz
mv agencycli-v0.1.0-linux-amd64 /usr/local/bin/agencycli
chmod +x /usr/local/bin/agencycli
agencycli version
```

---

## 2. Understand the structure

agencycli organises agents in a four-level hierarchy:

```
Agency                   ← global context shared by every agent
  └─ Team                ← capability group (engineering, qa, growth…)
       └─ Role           ← job function within the team (go-developer, pr-reviewer…)
            └─ Project   ← concrete product or repo
                 └─ Agent ← individual AI agent instance (model + context + skills)
```

**Context flows top-down.** Every agent automatically inherits: agency context → team context → role context → project context.

**Skills** are reusable capabilities (tool instructions, scripts) defined once and bound to teams or roles. They are automatically deployed into each agent's working directory on `hire` or `sync`.

---

## 3. Create an agency workspace

```bash
# Create a new workspace directory and initialise it
agencycli create agency --name "MyAgency" --desc "We build great software with AI agents"

# Enter the workspace — all subsequent commands run from here
cd MyAgency
```

This creates:
```
MyAgency/
  .agencycli/agency.yaml    ← workspace metadata
  agency-prompt.md          ← edit this: add your global rules, values, and tone
```

**Edit `agency-prompt.md` now.** Add anything every agent should always know: coding standards, communication style, how to handle blockers, how to report progress.

---

## 4. Create teams

Teams group agents by capability. Use nested paths for sub-teams.

```bash
agencycli create team --name "engineering" --desc "Software engineers"
agencycli create team --name "engineering/backend" --desc "Go/API services"
agencycli create team --name "qa" --desc "Quality assurance"
agencycli create team --name "product" --desc "Product management"
agencycli create team --name "growth" --desc "Content and marketing"
```

Each team gets a `teams/<name>/prompt.md` — edit it to add team-specific standards (e.g. coding conventions for engineering, review criteria for QA).

---

## 5. Add skills

Skills are reusable capabilities you define. There are no built-in skills — only what you need.

A skill is a directory under `skills/` containing:
- `skill.yaml` — name and description
- `prompt.md` — instructions injected into the agent's context
- Any additional files (scripts, templates) — all are bundled and deployed automatically

Use `{{SKILL_DIR}}` in `prompt.md` to reference the path of a bundled script file:

```bash
mkdir -p skills/github-pr-review
```

```yaml
# skills/github-pr-review/skill.yaml
name: github-pr-review
description: Instructions for reviewing and approving GitHub pull requests
```

```markdown
<!-- skills/github-pr-review/prompt.md -->
## Skill: GitHub PR Review

When asked to review a PR:
1. Run `gh pr view <number>` to read the description
2. Run `gh pr diff <number>` to inspect all changes
3. Check for obvious bugs, missing tests, and security issues
4. Approve with `gh pr review <number> --approve` or request changes with `--request-changes`
```

Bind skills to a team (all agents in the team inherit them):

```bash
# Edit teams/qa/team.yaml and add under skills:
#   skills:
#     - github-pr-review
```

Or bind to a role (see Section 6 below).

---

## 6. Create roles

Roles define the job function of an agent within a team. They add an extra context layer and can bind skills and set up initial workspace directories.

```bash
# Create roles
agencycli create role --team "engineering" --name "go-developer"   --desc "Implements Go services"
agencycli create role --team "engineering" --name "release-engineer" --desc "Manages releases"
agencycli create role --team "qa"          --name "pr-reviewer"     --desc "Reviews PRs"
agencycli create role --team "product"     --name "product-manager" --desc "Owns roadmap and tasks"
agencycli create role --team "growth"      --name "content-writer"  --desc "Writes and publishes content"
```

Each role gets a `teams/<team>/roles/<role>/` directory. Edit these two files:

- **`role.yaml`** — skills and workspace setup:

```yaml
# teams/engineering/roles/go-developer/role.yaml
name: go-developer
description: Implements features and fixes bugs for Go projects
skills:
  - github-pr-review
setup:
  dirs:
    - scratch
    - notes
```

- **`prompt.md`** — role-specific instructions the agent will always follow.

Bind or unbind skills to a role at any time:

```bash
agencycli role skill add    --team engineering --role go-developer --skill github-pr-review
agencycli role skill remove --team engineering --role go-developer --skill github-pr-review
agencycli role list --team engineering    # see all roles and their current skills
```

---

## 7. Create projects

A project corresponds to a real code repository or work stream.

```bash
agencycli create project --name "my-api" \
  --desc "REST API service" \
  --repo "/absolute/path/to/my-api"   # local repo path; auto-added to agent's working dirs
```

Edit `projects/my-api/prompt.md` — add project-specific context: tech stack, build/test commands, PR conventions, issue tracker URL.

---

## 8. Hire agents

`hire` (and its alias `assign`) assembles the full context chain and creates an agent working directory. It also applies role workspace setup automatically.

```bash
# Hire a Go developer for my-api
agencycli hire \
  --project "my-api" \
  --team    "engineering" \
  --role    "go-developer" \
  --model   "claudecode" \
  --name    "dev"

# Hire a QA reviewer
agencycli hire \
  --project "my-api" \
  --team    "qa" \
  --role    "pr-reviewer" \
  --model   "claudecode" \
  --name    "reviewer"

# Hire inside a Docker sandbox (for isolation)
agencycli hire \
  --project "my-api" \
  --team    "engineering" \
  --role    "go-developer" \
  --model   "claudecode" \
  --name    "dev-sandbox" \
  --sandbox docker
```

Available models: `claudecode`, `codex`, `cursor`, `gemini`, `qoder`, `opencode`, `iflow`, `generic-cli`

The hired agent's working directory is at `projects/<project>/agents/<name>/` and contains everything the agent needs to start immediately (context file, skills, task queue, heartbeat config).

---

## 9. Start an agent manually

```bash
# Claude Code
cd projects/my-api/agents/dev
claude

# Codex
cd projects/my-api/agents/dev
codex

# Gemini CLI
cd projects/my-api/agents/dev
gemini
```

The agent automatically loads its context file (`CLAUDE.md`, `AGENTS.md`, etc.) and skills from the working directory.

---

## 10. Run one-off prompts (exec)

Test an agent without waiting for the task queue:

```bash
agencycli exec \
  --project my-api \
  --agent   dev \
  --prompt  "Run gh issue list and summarise all open issues"
```

Output streams to stdout. Logs are saved to `projects/my-api/agents/dev/runs/`.

---

## 11. Assign tasks

```bash
# Add a task for the developer
agencycli task add \
  --project my-api \
  --agent   dev \
  --title   "Implement user pagination" \
  --type    feature \
  --priority 1 \
  --prompt  "Add cursor-based pagination to GET /users. Page size default 20, max 100. Update OpenAPI spec."

# Add a task that requires human approval first
agencycli task add \
  --project my-api \
  --agent   pm \
  --title   "Q2 roadmap review" \
  --assignee human \
  --prompt  "Is the AI search feature in scope for Q2?"

# List tasks
agencycli task list --project my-api --agent dev
```

Run a task manually:

```bash
agencycli run --project my-api --agent dev
```

---

## 12. Set up heartbeat automation

The heartbeat loop wakes the agent every N minutes to process all pending tasks, then sleeps again. Cycles never overlap. Session continuity is preserved across cycles.

```bash
# Configure heartbeat intervals
agencycli daemon heartbeat --project my-api --agent dev      --enable --interval 30m
agencycli daemon heartbeat --project my-api --agent reviewer --enable --interval 1h

# Start the daemon (manages all enabled agents)
agencycli daemon start
```

---

## 13. Human inbox

Agents can route tasks to you for approval:

```bash
agencycli inbox                          # list all pending human decisions
agencycli inbox show    <task-id>        # view details
agencycli inbox confirm <task-id>        # approve → agent resumes automatically
agencycli inbox confirm <task-id> --message "Focus on the auth module only"
agencycli inbox reject  <task-id> --reason "out of scope"
```

The inbox is also available as `inbox.md` in the workspace root.

---

## 14. Context sync

After editing any `prompt.md`, `team.yaml`, `role.yaml`, or skill file, re-sync agents to propagate changes:

```bash
agencycli sync                               # sync all agents with changed context
agencycli sync --project my-api             # sync one project
agencycli sync --project my-api --name dev  # sync one agent
agencycli sync --force                       # force regenerate everything
```

---

## 15. View and inspect

```bash
agencycli list teams
agencycli list projects
agencycli list agents

agencycli role list --team engineering

agencycli show agent my-api dev
agencycli show agent my-api dev --raw    # print full merged context

agencycli session show --project my-api --agent dev
agencycli session clear --project my-api --agent dev   # start a fresh conversation
```

---

## 16. Fire (remove) an agent

```bash
# Soft delete — moves the agent directory to agents/.fired/<name>-<timestamp>/
agencycli fire --project my-api --agent dev

# Hard delete — permanently removes the agent directory
agencycli fire --project my-api --agent dev --force
```

---

## 17. Use --dir for remote workspaces

All commands accept `--dir` to target a workspace outside the current directory:

```bash
agencycli --dir /path/to/MyAgency list agents
agencycli --dir /path/to/MyAgency sync
agencycli --dir /path/to/MyAgency exec --project my-api --agent dev --prompt "..."
```

---

## Reference: complete command list

| Category | Command |
|----------|---------|
| Workspace | `create agency`, `create team`, `create project`, `create role` |
| Role mgmt | `role list`, `role skill add`, `role skill remove` |
| Agent lifecycle | `hire` / `assign`, `fire`, `sync`, `list`, `show` |
| Execution | `exec`, `run` |
| Tasks | `task add/list/show/done/retry/cancel/confirm-request` |
| Inbox | `inbox list/show/confirm/reject/comment` |
| Automation | `daemon start/stop/status`, `daemon heartbeat` |
| Session | `session show/set/clear` |
| Meta | `version` |

For detailed flags on any command:

```bash
agencycli <command> --help
agencycli <command> <subcommand> --help
```

---

## Typical agency setup checklist

```
[ ] agencycli create agency --name "..."
[ ] Edit agency-prompt.md (global rules)
[ ] agencycli create team --name "engineering"
[ ] Edit teams/engineering/prompt.md (coding standards)
[ ] agencycli create role --team engineering --name go-developer
[ ] Edit teams/engineering/roles/go-developer/prompt.md (role responsibilities)
[ ] Edit teams/engineering/roles/go-developer/role.yaml (skills, setup dirs)
[ ] mkdir -p skills/<skill-name> && write skill.yaml and prompt.md
[ ] agencycli role skill add --team engineering --role go-developer --skill <name>
[ ] agencycli create project --name "my-app" --repo /path/to/repo
[ ] Edit projects/my-app/prompt.md (project context, build commands)
[ ] agencycli hire --project my-app --team engineering --role go-developer --model claudecode --name dev
[ ] agencycli exec --project my-app --agent dev --prompt "Introduce yourself and list open issues"
[ ] agencycli task add --project my-app --agent dev --title "First task" --prompt "..."
[ ] agencycli daemon heartbeat --project my-app --agent dev --enable --interval 30m
[ ] agencycli daemon start
```
