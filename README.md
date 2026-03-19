# agencycli

> **Spin up a self-managing AI agent team in minutes.**  
> One CLI. No server. Agents that plan, execute, and talk to each other — while you sleep.

```
npm install -g @agencycli/agentctl
```

---

## What is this?

**agencycli** is a lightweight CLI for building and operating teams of AI agents. You define the org chart once — teams, roles, projects, skills — and agents assemble their own context, pick up tasks, and run autonomously on a heartbeat schedule.

The killer feature: **agents can hire, message, and coordinate with each other.** Your PM agent can create a task for the dev agent, the dev agent can ask a human for confirmation before merging, and the QA agent wakes up every 30 minutes to scan for open PRs — all without you lifting a finger.

---

## Four design pillars

### 1 — Context grid: role × project

Context is not flat. It composes from two axes:

```
             ← horizontal: what you are (Role) →
              engineer    pm    qa-reviewer    writer
              ─────────────────────────────────────
vertical:  api-service │  ●        ●              
project    mobile-app  │  ●                   ●   
axis       data-pipeline│  ●        ●              
```

Every agent gets: `agency context → team context → role context → project context` — merged automatically at `hire` time. Change a role prompt once; every agent with that role gets it on the next `sync`.

### 2 — Autonomous heartbeat + wakeup routine

Agents don't just sit and wait for tasks. Configure a heartbeat and they wake up on a schedule, drain their task queue, then — when the queue is empty — execute a **wakeup routine** (`wakeup.md`) to proactively find new work:

```
[heartbeat cc-connect/pm]  sleeping 28m → next at 09:30:00
[heartbeat cc-connect/qa]  waking up — checking crons, running pending tasks
[heartbeat cc-connect/qa]  ▶ wakeup routine  (scanning open PRs…)
[heartbeat cc-connect/qa]  wakeup cycle done — sleeping 24m → next at 09:27:00
```

Time windows, active days, cron schedules — all configurable. Agents outside their window show `⏸ outside active window — next wakeup in Xh`.

### 3 — Inbox: agents talk to each other

Every participant (agent or human) has an inbox. Communication is non-blocking and asynchronous:

```
Human  →  PM agent:     "prioritise issue #42"
PM     →  Dev agent:    "new task ready, extra context: ..."
Dev    →  Human:        confirm-request — "PR #205 ready for merge"  ← blocks until you decide
QA     →  PM + Human:   "weekly review summary (group send)"
PM     →  Dev + QA:     inbox fwd <msg-id> --to dev --to qa --note "FYI"
```

Unread messages are auto-injected at the top of every wakeup prompt — no polling code needed in your playbooks.

### 4 — Templates: package and reuse entire teams

Bundle your whole agency setup — teams, roles, skills, agent playbooks, project blueprints — into a single `.tar.gz`. Share it. Apply it to a new project in one command:

```bash
agencycli create agency --name "AcmeCorp" \
  --template https://yourcdn.com/tech-agency.tar.gz

agencycli project apply --project my-new-service
agencycli scheduler start
# Done. Agents are running.
```

---

## Install

```bash
# npm (no Go required)
npm install -g @agencycli/agentctl

# Go
go install github.com/chenhg5/agencycli/cmd/agencycli@latest

# From source
git clone https://github.com/chenhg5/agencycli && cd agencycli && make install
```

---

## Quick start

```bash
# 1. Create a workspace (generates .gitignore + agency-prompt.md)
agencycli create agency --name "MyAgency"
cd MyAgency

# 2. Apply a project blueprint — hires all agents + configures heartbeats + installs playbooks
agencycli create project --name "my-service" --blueprint default
agencycli project apply  --project my-service

# 3. Start the scheduler — agents wake up and run autonomously
agencycli scheduler start

# 4. Check in
agencycli inbox list              # task confirmations waiting for your decision
agencycli inbox messages          # async messages from agents
agencycli task list --project my-service --agent pm
```

---

## Supported AI models

| `--model`     | Context file(s) |
|---------------|-----------------|
| `claudecode`  | `CLAUDE.md` + `@import` layers + `.claude/skills/` |
| `codex`       | `AGENTS.md` (skills inlined) |
| `cursor`      | `.cursorrules` + `.cursor/rules/agencycli.mdc` |
| `gemini`      | `GEMINI.md` + `@import` layers + `.gemini/skills/` |
| `qoder` / `opencode` / `iflow` | single merged file |
| `generic-cli` | `context.md` plain text |

---

## At a glance

```
agencycli
├── create agency / team / role / project   # scaffold your org
├── hire / fire / sync                      # manage agents
├── task add / list / done / confirm-request# task queue (7-state lifecycle)
├── inbox send / messages / reply / fwd     # async messaging
├── scheduler start / stop / status         # heartbeat scheduler
├── cron add / list / delete                # scheduled tasks
├── template pack / info                    # share your setup
└── --dir <path>                            # work on any agency from anywhere
```

→ **[Full command reference](docs/commands.md)**  
→ **[Workspace layout](docs/workspace-layout.md)**  
→ **[Docker sandbox](docs/sandbox-design.md)**

---

## Why not LangGraph / CrewAI / AutoGen?

Those are frameworks — you write Python to wire agents together. **agencycli is infrastructure** — you write Markdown and YAML. Agents are whatever CLI tool you already use (Claude Code, Codex, Gemini CLI…). No SDK, no lock-in, no server to run.

| | agencycli | Framework-based |
|--|-----------|-----------------|
| Agent runtime | Your existing CLI tool | Framework's agent loop |
| Config format | Markdown + YAML | Python code |
| Multi-model | Any CLI, mix freely | Usually one SDK |
| Context management | Layered, auto-merged | Manual prompt assembly |
| Server required | No | Often yes |

---

## Roadmap

- [x] Context grid (agency → team → role → project)
- [x] Heartbeat scheduler with active-hours / active-days windows
- [x] Wakeup routines (`wakeup.md`) — proactive autonomous work
- [x] Async inbox messaging between any participants
- [x] Task confirm-request — agents escalate to humans
- [x] Cron scheduling
- [x] Docker sandbox (credentials auto-mounted)
- [x] Template pack/apply
- [x] Project blueprints
- [ ] `depends_on` task dependency resolution
- [ ] E2B / Daytona sandbox provider
- [ ] Run log rotation

---

## License

MIT
