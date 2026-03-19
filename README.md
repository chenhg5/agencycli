<img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/banner.svg" alt="agencycli" width="900" />

<br/>

[![npm](https://img.shields.io/npm/v/%40agencycli%2Fagentctl?color=cb3837&logo=npm&label=npm&style=flat-square)](https://www.npmjs.com/package/@agencycli/agentctl)
[![Go](https://img.shields.io/github/go-mod/go-version/chenhg5/agencycli?logo=go&logoColor=white&style=flat-square)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](https://opensource.org/licenses/MIT)
[![Works with](https://img.shields.io/badge/works%20with-Claude%20%C2%B7%20Codex%20%C2%B7%20Gemini%20%C2%B7%20Cursor-8a2be2?style=flat-square)](#works-with-any-ai-coding-agent)

<br/>

**Spin up a self-managing AI agent team in minutes.**  
One CLI. No server. Agents that plan, execute, and talk to each other — while you sleep.

<br/>

[**中文文档**](README.zh-CN.md) &nbsp;·&nbsp; [Quick Start](#quick-start) &nbsp;·&nbsp; [Install](#install) &nbsp;·&nbsp; [Commands](docs/commands.md) &nbsp;·&nbsp; [Workspace Layout](docs/workspace-layout.md)

## What is this?

**agencycli** is a lightweight CLI for building and operating teams of AI agents. You define the org chart once — teams, roles, projects, skills — and agents assemble their own context, pick up tasks, and run autonomously on a heartbeat schedule.

The killer feature: **agents can hire, message, and coordinate with each other.** Your PM agent can create a task for the dev agent, the dev agent can ask a human for confirmation before merging, and the QA agent wakes up every 30 minutes to scan for open PRs — all without you lifting a finger.

---

## Works with any AI coding agent

agencycli is a runtime layer, not an SDK. Agents are whatever CLI tool you already use:

| Agent runtime | `--model` |
|---|---|
| [Claude Code](https://docs.anthropic.com/claude-code) | `claudecode` |
| [OpenAI Codex](https://github.com/openai/codex) | `codex` |
| [Gemini CLI](https://github.com/google-gemini/gemini-cli) | `gemini` |
| [Cursor](https://www.cursor.com/) | `cursor` |
| [Qoder](https://qoder.ai) | `qoder` |
| [OpenCode](https://opencode.ai) | `opencode` |
| [iFlow](https://iflow.ai) | `iflow` |
| Any CLI tool | `generic-cli` |

Mix models freely — your PM can run on Claude, your dev agents on Codex, your writer on Gemini. Each gets its context in the exact format its runtime expects.

---

## Six design pillars

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

Time windows, active days, cron schedules — all configurable. Startup jitter prevents thundering herd when the scheduler restarts.

### 3 — Inbox: agents talk to each other

Every participant (agent or human) has an inbox. Communication is non-blocking and asynchronous:

```
Human  →  PM agent:     "prioritise issue #42"
PM     →  Dev agent:    "new task ready, extra context: ..."
Dev    →  Human:        confirm-request — "PR #205 ready for merge"  ← blocks until you decide
QA     →  PM + Human:   "weekly review summary"  (group send)
PM     →  Dev + QA:     inbox fwd <msg-id> --note "FYI"
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

### 5 — Docker sandbox: safe by default

Agents can be run inside isolated Docker containers. No accidental host damage, no credential leaks to untrusted code, no runaway processes. Each task gets a fresh container; when it exits, nothing persists outside the mounted workspace.

```bash
agencycli hire --project my-api --team engineering --role developer \
  --model claudecode --name dev --sandbox docker
```

What gets mounted automatically:
- Agent working directory and project repo (read/write)
- `agencycli` binary (so agents can call `task done`, `inbox send`, etc.)
- Credentials (`~/.claude`, `~/.config/gh`, `~/.ssh`, `~/.codex`, …) read-only
- Well-known API keys forwarded as environment variables

The agent gets full access to its workspace and nothing else.

### 6 — Skills: reusable, bundled capabilities

Skills are Markdown + optional scripts deployed into agent working directories. No built-ins — define only what your agents actually need, attach them to teams or roles, and they propagate automatically on `sync`.

```
skills/github-push-relay/
  skill.yaml
  prompt.md              # {{SKILL_DIR}}/push.sh resolves to the actual path
  push.sh                # bundled script, chmod+x preserved
```

---

## Install

### Install & Configure via AI Agent (Recommended)

The easiest way — send this to Claude Code or any AI coding agent, and it will handle the entire installation and configuration for you:

```
Follow https://raw.githubusercontent.com/chenhg5/agencycli/refs/heads/main/INSTALL.md to install and configure agencycli.
```

### Manual install

```bash
npm install -g @agencycli/agentctl      # npm, no Go required

go install github.com/chenhg5/agencycli/cmd/agencycli@latest  # Go

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

## At a glance

```
agencycli
├── overview                                # dashboard: agents, teams, skills, inbox
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

Those are frameworks — you write Python to wire agents together. **agencycli is infrastructure** — you write Markdown and YAML. Agents are whatever CLI tool you already use. No SDK, no lock-in, no server to run.

| | agencycli | Framework-based |
|--|-----------|-----------------|
| Agent runtime | Your existing CLI tool | Framework's agent loop |
| Config format | Markdown + YAML | Python code |
| Multi-model | Any CLI, mix freely | Usually one SDK |
| Context management | Layered, auto-merged | Manual prompt assembly |
| Server required | No | Often yes |

---

## License

MIT
