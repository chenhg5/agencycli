# Changelog

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
