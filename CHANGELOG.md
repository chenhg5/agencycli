# Changelog

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
