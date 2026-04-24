# Changelog

## [v0.5.1] - 2026-04-20

### Added

**Knowledge base document viewer improvements**
- Table of contents (TOC) navigation with scroll-spy and smooth scrolling
- Floating transparent TOC overlay on small screens (avoids content obstruction)
- Copy document relative path button

**File manager module**
- New file browser page for managing agency promotional materials (images, videos, etc.)
- Grid/list view toggle with media thumbnails and preview
- Drag-and-drop file upload, folder creation, and file deletion
- Image zoom viewer and video/audio preview with native controls
- Drag-and-drop file/folder move (including breadcrumb drop targets)
- Copy file path button in grid, list, and preview modal

**Cron job session management**
- Session scope setting for cron jobs: "new each run" or "persistent" (matching heartbeat behavior)
- Persistent session shows session ID with copy-command button for direct CLI resume
- Edit and create forms include session scope selector

**Documentation**
- Competitive analysis document (HiClaw + Molecule AI comparison)
- CubeSandbox vs Docker evaluation document

## [v0.5.0] - 2026-04-20

### Added

**Provider management**
- `agencycli provider` CLI commands for managing API providers (add/list/remove/set-default)
- Setup guidance system: interactive first-run wizard for new workspaces

**Cron editing & execution**
- Web UI: inline cron editing (expression, prompt, enabled toggle)
- Fire pending crons during heartbeat sleep phase (no longer wait for next full wakeup cycle)

**Agent identity injection**
- Inject agent identity environment variables (`AGENCYCLI_AGENT`, `AGENCYCLI_PROJECT`, etc.) into every agent process

**Agent abort & sandbox config**
- Abort running agents from web UI and CLI (`agencycli run abort`)
- Sandbox configuration panel in agent detail page (image, mounts, env)
- Workbench sort/filter improvements and skill deep-links on agent page

**Draggable page tabs**
- Top bar tabs are now draggable for custom ordering

### Fixed
- Mount `.claude.json` as read-write in Docker sandbox (was read-only, breaking session persistence)
- Remove inline error text flash on schedule page; default cursor style on tabs
- Use explicit PATH in Docker container instead of unexpanded `$PATH` variable
- Use `exec.Command` + stdin pipe instead of `bash -c` for Docker runs (fixes quoting issues)
- Shell-escape Docker args when using stdin prompt redirect
- Include process output tail in run error messages for better diagnostics

## [v0.4.1] - 2026-04-06

### Added

**Environment variable management**
- Workspace-level environment variables (envvars): global or agent-scoped, injected at runtime
- Resolution priority chain: workspace global → agent-scoped → API provider → per-agent env
- CLI commands: `agencycli envvar add/list/remove` (alias: `ev`)
- CLI commands: `agencycli agent set-env/unset-env/list-env` for per-agent variables
- Web Settings page: envvar CRUD with agent picker (project-grouped multi-select)
- Agent detail page: env panel with inline editing, sensitive value masking, and eye toggle

**Event-driven triggers**
- New scheduling mechanism: trigger agent wakeup on message received or task assigned
- Trigger configuration via CLI (`agencycli agent set-trigger`) and web heartbeat editor
- Deduplicated trigger execution with configurable cooldown

**Workbench enhancements**
- Project schedule overview cards: agent count, running agents, scheduler status, task/message counts
- Start/stop individual project schedulers and "start all" button from workbench
- Task tab badge showing pending task count
- Running agents count displayed on project cards

**Knowledge base improvements**
- Document fullscreen mode: hide header/nav, centered content, ESC to exit
- Code block copy fix (no more `[object Object]`)
- Documents sorted by creation time
- i18n-aware date formatting

**Multi-level OKR**
- OKR hierarchy: global, project, team, and agent scopes with parent linking
- Scope tabs with project-level filtering
- Agent dropdown selector for agent-scoped OKRs
- KR target value display: `0/10000 (unit)` format

### Fixed
- Task duplication when status changed to completed/cancelled (missing archive call)
- Heartbeat edit modal overflowing viewport
- Scheduler showing "pending activation" instead of next-window time on inactive days
- Run detail page now shows failure reason; status column fully i18n'd
- Workbench overview colors unified (blue general, green for pending items only)

### Changed
- `secrets.yaml` → `envvars.yaml`; Secret type → EnvVar; API `/secrets` → `/envvars`
- CLI `secret` command → `envvar` (alias `ev`)

### Security
- Fix command injection in wakeup condition pipe chain validation

## [v0.4.0] - 2026-04-06

### Added

**Goal management (OKR & Milestones)**
- OKR system: Objectives with Key Results supporting number/percentage/boolean/currency metric types
- Milestone management: project-level milestones with completion criteria, task labels, and due dates
- OKR web dashboard with inline KR value editing, create/edit/delete modals, and description fields
- Milestone panel with create/edit modal, progress tracking, and i18n-aware date formatting
- CLI commands: `agencycli okr list/create/update/delete` with `kr add/update` and `review` subcommands
- CLI commands: `agencycli milestone list/create/show/update/delete`
- Agent context injection: active OKR and milestone summaries auto-injected into agent prompts
- Web AI assistant prompt updated with goal management awareness

**Multi-user support**
- People management page: create/edit/delete user accounts with username/password
- RBAC permission model design
- Person detail page with editable profile fields (email, avatar, phone, bio)
- Human hiring flow via web UI

**IM platform integration (cc-connect)**
- cc-connect API proxy: connect agents to Feishu/WeChat via QR code scanning
- Settings page: one-stop cc-connect configuration panel
- Agent detail page: IM connection panel for binding IM accounts per agent
- Explicit project creation wizard with auto-restart polling

**Task & workbench enhancements**
- Kanban board view: list/board toggle with drag-and-drop status changes
- Batch operations: bulk cancel/archive/delete tasks
- Workbench kanban: unified message/task kanban in workbench
- Fire/remove agent or human member from projects

**Scheduler & operations**
- Heartbeat session management with SessionID tracking
- Context usage statistics (token consumption per agent)
- Unified API provider management via web UI (key + base URL configuration)
- AI assistant interactive permissions: allow/deny/allow-all for tool calls
- Run records track actual API model and base URL used
- Graceful scheduler shutdown on Ctrl+C

### Fixed
- Claude thinking signature validation error auto-retry (backtick variant)
- Codex Docker sandbox seccomp permission (`bwrap` namespace creation)
- Knowledge base third-level directory navigation
- Scheduler `ActiveDays` configuration not being respected
- Workbench reply textarea hiding while typing
- cc-connect project name path encoding with URL-safe separators
- Dark mode select dropdown option styling across all pages
- React error #310 in workbench message detail modal

### Changed
- Page header buttons unified to outline style across OKR, milestone, people, and task pages
- Date display follows i18n locale conventions (Intl.DateTimeFormat)
- Workbench defaults to inbox tab; reply available from message detail modal
- Markdown rendering in message detail modal

## [v0.3.0] - 2026-04-03

### Added

**Knowledge base (docs)**
- `agencycli docs add` — index documents by file path with virtual directory structure
- `agencycli docs list / tree / show / update / move / remove / search` — full document management
- Web document viewer with Notion-style Markdown rendering, syntax highlighting, and YAML frontmatter stripping
- Collapsible sidebar with virtual directory tree navigation
- Document download via authenticated API endpoint
- URL deep-linking: access documents directly via `/docs/<index>/<slug>`

**AI assistant**
- Built-in AI assistant widget (floating, draggable, resizable) powered by Claude CLI
- Streaming chat with tool permission handling (`--allowedTools`)
- Pre-loaded agencycli SKILL for guided operations

**Scheduler & heartbeat**
- Daemon service management (`agencycli service install/start/stop/status/uninstall`)
- Version update checking with footer notification in web UI
- Heartbeat UX: wakeup presets (pending tasks / unread messages), live log viewer
- Wakeup auto-sync: editing wakeup prompt on web UI immediately regenerates CLAUDE.md
- Explicit wakeup trigger prompt directs agents to follow `wakeup.md` steps

**Cursor agent support**
- `--force --trust` flags for full sandbox permissions in headless mode
- Token usage parsing for Cursor's camelCase stream-json format (`inputTokens`/`outputTokens`)
- Conversation log viewer adapted for Cursor tool_call/thinking events

**Web UI polish**
- Dark mode contrast overhaul: layered backgrounds (zinc-950 content / zinc-900 chrome)
- Breadcrumb bar with brand indicator and improved typography
- Agent detail page restructured with section headers and info cards
- Context compression env vars configurable per agent (Claude Code autocompact)
- Formatted conversation log for schedule wakeup results (reuses run viewer)

### Fixed
- `task add` now requires explicit `--created-by` flag; rejects `<project>/human` format
- Wakeup prompt path corrected to `.agencycli/context/wakeup.md` across all docs
- `inbox reply` default `from` field set to original recipient (was incorrectly `human`)
- AI assistant: YAML frontmatter in SKILL no longer passed as CLI argument
- AI assistant: position validation prevents widget disappearing off-screen
- Schedule page: agent column links to member detail page
- `sync --force` now always reports "synced" instead of misleading "skipped"
- Version compare strips git-describe suffixes for accurate footer display
- Dark mode text contrast improved globally (zinc-700→600→500 cascade)

### Changed
- Schedule heartbeat status label: "等待中" → "待激活" for clarity
- Heartbeat wakeup preconditions check all non-completed task statuses (not just pending)

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
