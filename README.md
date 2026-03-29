<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/banner.svg" alt="agencycli" width="900"/>
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/@agencycli/agencycli">
    <img src="https://img.shields.io/npm/v/%40agencycli%2Fagencycli?color=cb3837&logo=npm&label=npm&style=flat-square" alt="npm"/>
  </a>
  <a href="https://github.com/chenhg5/agencycli/releases">
    <img src="https://img.shields.io/github/v/release/chenhg5/agencycli?style=flat-square&logo=github" alt="Release"/>
  </a>
  <a href="https://go.dev/">
    <img src="https://img.shields.io/github/go-mod/go-version/chenhg5/agencycli?logo=go&logoColor=white&style=flat-square" alt="Go"/>
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square" alt="License: MIT"/>
  </a>
  <a href="https://goreportcard.com/report/github.com/chenhg5/agencycli">
    <img src="https://goreportcard.com/badge/github.com/chenhg5/agencycli?style=flat-square" alt="Go Report Card"/>
  </a>
</p>

<p align="center">
  <a href="#works-with-any-ai-coding-agent">
    <img src="https://img.shields.io/badge/works%20with-Claude%20%C2%B7%20Codex%20%C2%B7%20Gemini%20%C2%B7%20Cursor-8a2be2?style=flat-square" alt="Works with"/>
  </a>
</p>

<p align="center">
  <strong>Spin up a self-managing AI agent team in minutes.</strong><br/>
  One CLI + built-in web console. Agents that plan, execute, and talk to each other — while you sleep.
</p>

<p align="center">
  <a href="./README.zh-CN.md">中文文档</a> &nbsp;|&nbsp;
  <a href="#quick-start">Quick Start</a> &nbsp;|&nbsp;
  <a href="#install">Install</a> &nbsp;|&nbsp;
  <a href="docs/commands.md">Commands</a> &nbsp;|&nbsp;
  <a href="docs/workspace-layout.md">Workspace Layout</a>
</p>

## What is this?

**agencycli** is a lightweight CLI for building and operating teams of AI agents. You define the org chart once — teams, roles, projects, skills — and agents assemble their own context, pick up tasks, and run autonomously on a heartbeat schedule.

The killer feature: **agents can hire, message, and coordinate with each other.** Your PM agent can create a task for the dev agent, the dev agent can ask a human for confirmation before merging, and the QA agent wakes up every 30 minutes to scan for open PRs — all without you lifting a finger.

https://github.com/user-attachments/assets/dbeedf20-f967-4a4f-bb7b-49dc254fcd0d

## Six design pillars

### 1 — Context grid: role × project

Context composes from two axes — role (horizontal) and project (vertical). Every agent gets `agency → team → role → project` context merged automatically at `hire` time. Change a role prompt once; every agent with that role gets it on the next `sync`.

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-1-context.svg" alt="Context grid" width="900"/>
</p>

### 2 — Autonomous heartbeat + wakeup routine

Agents wake up on a schedule, drain their task queue, then — when the queue is empty — execute a **wakeup routine** (`wakeup.md`) to proactively find new work. Time windows, active days, cron schedules — all configurable. Startup jitter prevents thundering herd when the scheduler restarts.

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-2-heartbeat.svg" alt="Heartbeat scheduler" width="900"/>
</p>

### 3 — Inbox: agents talk to each other

Every participant (agent or human) has an inbox. Messages are non-blocking and async — unread messages are auto-injected at the top of every wakeup prompt. `confirm-request` creates a blocking gate: the task pauses until you decide.

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-3-inbox.svg" alt="Inbox messaging" width="900"/>
</p>

### 4 — Templates: package and reuse entire teams

Bundle your whole agency setup — teams, roles, skills, agent playbooks, project blueprints — into a single `.tar.gz`. Share it. Apply it to a new project in one command.

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-4-templates.svg" alt="Templates" width="900"/>
</p>

### 5 — Docker sandbox: safe by default

Agents run inside isolated Docker containers. No accidental host damage, no credential leaks, no runaway processes. The workspace and `agencycli` binary are mounted read-write; credentials are mounted read-only.

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-5-docker.svg" alt="Docker sandbox" width="900"/>
</p>

### 6 — Skills: reusable, bundled capabilities

Skills are a `SKILL.md` (YAML frontmatter + Markdown prompt) plus optional scripts, deployed into every agent that has the skill bound. Define once, attach to a role, propagate automatically on `sync`.

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-6-skills.svg" alt="Skills" width="900"/>
</p>

## Install

### Install & Configure via AI Agent (Recommended)

The easiest way — send this to Claude Code or any AI coding agent, and it will handle the entire installation and configuration for you:

```
Follow https://raw.githubusercontent.com/chenhg5/agencycli/refs/heads/main/INSTALL.md to install and configure agencycli.
```

### Manual install

```bash
npm install -g @agencycli/agencycli      # npm, no Go required

go install github.com/chenhg5/agencycli/cmd/agencycli@latest  # Go

# From source (includes web console)
git clone https://github.com/chenhg5/agencycli && cd agencycli && make install
```

### From source (development)

```bash
git clone https://github.com/chenhg5/agencycli
cd agencycli
make build          # builds web frontend + Go binary → dist/agencycli
make install        # builds and installs to $GOPATH/bin
```

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

# 4. Open the web console — manage everything from the browser
agencycli start                   # http://127.0.0.1:27892
agencycli start --addr 0.0.0.0:8080 --open   # custom port + auto-open browser

# 5. Check in (CLI)
agencycli inbox list              # task confirmations waiting for your decision
agencycli inbox messages          # async messages from agents
agencycli task list --project my-service --agent pm
```

## Web console

The web console is embedded in the binary — no separate process or Node.js required. One command to launch:

```bash
agencycli start                          # default: 127.0.0.1:27892
agencycli start --addr 0.0.0.0:8080     # custom address
agencycli start --api-key my-secret     # with auth
```

**Capabilities:** workbench (messages + tasks), team/role management, project members, schedule (heartbeat/cron/runtime), run history & token costs, session management, agent run, prompt editing, skills — all with i18n (English / 中文 / 繁體中文 / 日本語) and dark mode.

> For local development with hot-reload: `agencycli api serve` + `cd web && pnpm dev`.

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

## At a glance

```
agencycli
├── start                                  # launch web console (API + frontend)
├── overview                               # CLI dashboard
├── create agency / team / role / project  # scaffold your org
├── hire / fire / sync                     # manage agents
├── task add / list / done / confirm-request # task queue (7-state lifecycle)
├── run / exec                             # manual agent execution
├── inbox send / messages / reply / fwd    # async messaging
├── scheduler start / stop / status        # heartbeat scheduler
├── session show / set / reset             # agent session management
├── cron add / list / delete               # scheduled tasks
├── template pack / info                   # share your setup
├── api serve                              # JSON API only (for dev)
└── --dir <path>                           # work on any agency from anywhere
```

→ **[Full command reference](docs/commands.md)**  
→ **[Workspace layout](docs/workspace-layout.md)**  
→ **[Docker sandbox](docs/sandbox-design.md)**  
→ **[HTTP / OpenAI-compatible agent](docs/http-agent.md)**

## Why not LangGraph / CrewAI / AutoGen?

Those are frameworks — you write Python to wire agents together. **agencycli is infrastructure** — you write Markdown and YAML. Agents are whatever CLI tool you already use. No SDK, no lock-in, no server to run.

| | agencycli | Framework-based |
|--|-----------|-----------------|
| Agent runtime | Your existing CLI tool | Framework's agent loop |
| Config format | Markdown + YAML | Python code |
| Multi-model | Any CLI, mix freely | Usually one SDK |
| Context management | Layered, auto-merged | Manual prompt assembly |
| Web UI | Built-in (single binary) | Separate deployment |
| Server required | No | Often yes |

## License

MIT
