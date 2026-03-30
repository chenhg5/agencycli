# Changelog

## [v0.2.2] - 2026-03-30

### Fixed
- npm install: Gitee fallback download URL pointed to wrong repository name

## [v0.2.1] - 2026-03-30

### Added
- Workbench: sent messages view with direction filter (inbox / sent / all)
- Task completion summary field with notification to task creator
- Agent model switching (including http-agent) from the web UI
- Copy-to-clipboard resume command in schedule runtime session column
- Refresh buttons on all table/filter pages
- Multi-page tab bar in header for quick page switching

### Fixed
- Unread message badge not updating after processing messages
- Scheduler next wakeup time showing stale values outside active window
- Message dialog recipients only showing one project's agents
- `forms.save` i18n key not applied in locale files
- Runs page table cell alignment
- Task type labels missing i18n support

### Changed
- Rename "Agency Console" to "AgencyCli" across all locales
- Workbench tasks panel defaults to showing pending tasks
- Simplified tab titles to show only the last breadcrumb segment
- Refresh buttons styled consistently with filter buttons

## [v0.2.0] - 2026-03-29

### Added

**Web console (built-in)**
- Single-binary web console served by `agencycli start` — no separate frontend deployment needed
- Frontend built with React + TypeScript + Tailwind CSS, embedded via `//go:embed`
- Workbench page: unified operator hub for messages and tasks with batch operations
- Full message management: send (multi-recipient), reply, filter (read/unread/archived/from), batch archive/delete
- Full task management: create, edit (status/priority/type), view detail with execution logs, batch cancel/archive/delete
- Schedule management: tabbed Heartbeat / Cron / Runtime views with CRUD operations
- Run management: filterable table with Markdown-rendered conversation logs
- Agent hiring and role creation from the web UI
- Project settings page for editing project prompts
- Skills page for viewing team and agent skills
- Manual agent wakeup and `agencycli run` from the Workbench
- Session management: view session ID/scope, switch scope (cycle/task), reset session
- Scheduler start/stop control from the web UI
- Authentication: username/password login with JWT tokens, user settings page
- i18n: English, 简体中文, 繁體中文, 日本語
- Plane-inspired professional UI: responsive sidebar, card layouts, sticky table columns, global footer

**CLI enhancements**
- `agencycli start` — unified command serving API + embedded web console on a single port
- `agencycli run` — manually execute an agent with optional prompt or next pending task
- `agencycli session reset` — clear agent session
- `--project` and `--agent` filters for `scheduler start`
- SQLite telemetry: persistent agent run data with `runs summary` and `runs agents` commands
- `agent set-model` — change agent model after hiring

**API**
- `POST /api/v1/run` — trigger agent execution with optional prompt
- `POST /api/v1/session/reset` — reset agent session
- `POST /api/v1/roles/create` — create new roles within teams
- `POST /api/v1/projects/{name}/hire` — hire new agents into projects
- `GET /api/v1/version` — dynamic version endpoint

**Build & release**
- Makefile: `web`, `web-install`, `web-dev` targets; `build` now embeds frontend automatically
- Cross-platform release archives embed the web console

### Changed

- Scheduler startup banner refactored with lipgloss for cleaner terminal rendering
- Scheduler table columns aligned with proper width handling
- `inbox send` now requires `--from` flag and validates identities

## [v0.1.1] - 2026-03-21

### Added

- `scheduler heartbeat pause <project>/<agent>` — temporarily halt heartbeat without removing config; scheduler stays alive
- `scheduler heartbeat resume <project>/<agent>` — resume a paused heartbeat
- `scheduler cron list <project>/<agent>` — list all crons with enabled status
- `scheduler cron pause <project>/<agent> <cron-id>` — disable a cron
- `scheduler cron resume <project>/<agent> <cron-id>` — re-enable a paused cron
- `scheduler cron delete <project>/<agent> <cron-id>` — remove a cron entirely
- `--model human` support for multiple human identities in inbox routing

### Fixed

- Scheduler `active_hours` timing: `waitDur` is now correctly capped to the remaining window so displayed "next at" times are accurate and the scheduler never schedules a wakeup outside the active window
- Scheduler now shows accurate "next at" time when the projected wake falls outside the active window (shows window closing time instead)
- Scheduler: moved `LastWakeup` assignment to after all checks so window-skip does not corrupt elapsed-time calculation for the next cycle
- Scheduler: fixed jitter being negated when multiple agents have wake times that all fall before the window opens on restart
- Sandbox: agent `AddDirs` are now correctly mounted into Docker containers (previously only the project-level `repo:` was checked, which was always empty when repos are defined per-agent in `AgentSpec.repos`)

### Changed

- `scheduler heartbeat configure` renamed from `scheduler heartbeat` (subcommands added); old usage still works via flags
- Scheduler startup log now shows which agents have `active_hours` windows configured

## [v0.1.0] - 2026-03-19

First public release of agencycli.

### Added

**Context management**
- Agency / team / sub-team / role / project scaffolding with `create` commands
- Layered context merging: `agency → team chain → role → project`, auto-assembled at `hire` time
- Support for 8 agent runtimes: `claudecode`, `codex`, `gemini`, `cursor`, `qoder`, `opencode`, `iflow`, `generic-cli`
- Skills system: reusable capability definitions with bundled files and `{{SKILL_DIR}}` resolution
- `sync` command with SHA-256 change detection — only re-generates changed layers
- `hire` / `assign` / `fire` (soft + hard delete) agent lifecycle commands
- `--dir` global flag to operate on any workspace from anywhere

**Task automation**
- Per-agent task queues with 6-state lifecycle (`pending → in_progress → done_success / done_failed / awaiting_confirmation / cancelled`)
- Priority ordering: 0=critical, 1=high, 2=normal, 3=low
- `task add / list / show / cancel / retry / stop-all / tokens`
- `task confirm-request` — agent escalates to human inbox (non-blocking, task archived)
- `run` (queue-based) and `exec` (one-shot) execution modes

**Heartbeat scheduler**
- Non-overlapping wakeup loop per agent: drain queue → sleep → repeat
- `active_hours` and `active_days` scheduling windows
- Startup jitter: prevents thundering herd when scheduler restarts
- Renamed from `daemon` to `scheduler` (aliases: `sched`, `s`)

**Wakeup routines**
- `wakeup.md` per agent: runs as synthetic task when queue is empty
- Enables fully autonomous proactive agents (scan issues, review PRs, etc.)
- Unread inbox messages auto-injected at top of wakeup prompt

**Cron scheduling**
- `cron add / list / delete / enable / disable` with standard crontab syntax
- Crons enqueue tasks; picked up on next heartbeat wakeup

**Inbox: task confirmations**
- Human confirmation inbox: `inbox list / show / confirm / reject / comment / forward`
- `--to` flag on `task confirm-request` to route to another agent instead of human

**Inbox: async messaging**
- Non-blocking message delivery between any participants (human or agent)
- `inbox send` with group send support (`--to` flag repeatable)
- `inbox messages` with `--from`, `--all`, `--archived`, `--mark-read` filters
- `inbox reply` — threaded replies by message ID
- `inbox fwd` — forward messages to one or more recipients with optional `--note`
- Per-message status: `inbox read / archive / delete` (alias: `rm`)

**Project blueprints**
- Declarative `project.yaml` defining agents, heartbeats, crons, and playbooks
- `project apply` — one command to hire all agents + configure schedules + install playbooks
- `project show / blueprints` — inspect project configuration

**Agent playbooks**
- `agent-playbooks/` directory for wakeup routine templates
- `playbook:` field in project blueprint installs as `wakeup.md` on `project apply`
- Playbooks included in template archives

**Templates**
- `template pack` — bundle agency as shareable `.tar.gz` (teams, roles, skills, playbooks, blueprints)
- `template info` — inspect metadata (local file, directory, or HTTPS URL)
- `create agency --template` — bootstrap from local file, directory, or remote URL
- `template.json` metadata: name, version, author, email, description, keywords

**Docker sandbox**
- Isolated container execution per task
- Auto-mounts: agent dir, project repo, agency workspace, credentials, `agencycli` binary
- API keys forwarded as environment variables
- Supports `claudecode` and `codex` sandbox images

**Dashboard**
- `agencycli overview` (aliases: `status`, `stat`) — ANSI TUI showing agents, heartbeat status, teams, skills, inbox summary
- Correct East Asian wide-character column width handling
