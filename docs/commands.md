# Command Reference

All commands accept a `--dir <path>` global flag to operate on a workspace outside the current directory.

```bash
agencycli --dir /path/to/MyAgency inbox list
```

---

## `create` — workspace setup

```bash
agencycli create agency  --name "MyAgency" [--desc "..."] [--template file.tar.gz|dir|URL]
agencycli create team    --name "engineering" [--desc "..."]
agencycli create team    --name "engineering/backend"        # nested sub-team
agencycli create role    --team "engineering" --name "developer" [--desc "..."]
agencycli create project --name "my-api" [--desc "..."] [--repo "/path/to/repo"]
agencycli create project --name "my-api" --blueprint default  # from a project blueprint
```

---

## `project` — project lifecycle

```bash
# List blueprints shipped with the template
agencycli project blueprints

# Show project.yaml (agents, heartbeats, playbooks)
agencycli project show --project my-api

# One-command bootstrap: hire all agents + configure heartbeats/crons + install playbooks
agencycli project apply --project my-api
agencycli project apply --project my-api --dry-run   # preview
agencycli project apply --project my-api --force     # re-hire existing agents
```

**`project-blueprints/default.yaml`** example:

```yaml
name: "{{PROJECT_NAME}}"
description: "REST API service"
agents:
  - name: dev
    role: developer
    team: engineering
    model: claudecode
    sandbox: true
    heartbeat:
      enabled: true
      interval: 30m
      active_hours: "09:00-20:00"
      active_days: weekdays
    playbook: dev.md          # installed as wakeup.md by project apply

  - name: pm
    role: product-manager
    team: product
    model: claudecode
    heartbeat:
      enabled: true
      interval: 30m
    playbook: pm.md
```

---

## `hire` / `assign` / `fire` / `sync`

```bash
# Hire an agent (hire and assign are identical)
agencycli hire \
  --project my-api --team engineering --role developer \
  --model claudecode --name dev \
  [--sandbox docker] [--force]

# Re-sync context after editing prompts or skills
agencycli sync --project my-api --name dev
agencycli sync --project my-api   # all agents in project
agencycli sync                    # entire agency

# Fire (remove) an agent
agencycli fire --project my-api --agent dev           # soft delete → .fired/
agencycli fire --project my-api --agent dev --force   # hard delete

# HTTP / OpenAI-compatible backend (Ollama, LM Studio, custom API)
agencycli hire --project my-api --team engineering --role developer \
  --model http-agent --name local-llm \
  --http-url "http://localhost:11434/v1/chat/completions" \
  --http-model "llama3.2"
# See docs/http-agent.md for the full wire format and agent.yaml reference.
```

---

## `agent` — per-agent utilities

```bash
# Change runtime (e.g. Claude Code → Codex): regenerates context, removes the old format’s files
agencycli agent set-model --project my-api --name dev --model codex

# Switch to http-agent (requires --http-url; same flags as hire)
agencycli agent set-model --project my-api --name bot --model http-agent \
  --http-url "http://localhost:11434/v1/chat/completions" --http-model "llama3.2"
```

`set-model` keeps hire metadata (team, role, `hired_at`, playbook, sandbox, `add_dirs`) but clears `run_command` (so the new model’s default CLI is used) and drops `http_agent` when leaving `http-agent`. If you use a **fixed** Docker `sandbox.docker.image`, verify it matches the new model or clear the image so the default for that model is used.

---

## `task` — task queue

```bash
agencycli task add    --project P --agent A --title "T" --prompt "..." \
                      [--type feature|bug|chore] [--priority 0-3]
agencycli task list   --project P --agent A [--status pending] [--archived]
agencycli task show   <task-id>
agencycli task cancel <task-id>
agencycli task retry  <task-id>

# Stop all running or pending tasks (emergency halt)
agencycli task stop-all --project P [--agent A | --all-agents] \
                        [--include-running] [--no-pending]

# View token usage and cost across agent runs
agencycli task tokens --project P [--agent A | --all-agents] [--all]

# Called by the agent inside its prompt:
agencycli task done --id <id> --status success --summary "what was done"
agencycli task done --id <id> --status failed  --error "reason"

# Route to human inbox for a decision (blocks current task until human responds):
agencycli task confirm-request --id <id> --summary "PR ready" \
  --action-item "Review the diff" \
  --action-item "Confirm merge"
```

**Task priority:** 0=critical, 1=high, 2=normal (default), 3=low. The scheduler always picks the highest-priority pending task first.

**Task lifecycle:**
```
pending → in_progress → done_success
                      → done_failed  → (auto-retry if max_retries set)
                      → awaiting_confirmation → done_success (via inbox reply)
```

---

## `run` / `exec`

```bash
agencycli run  --project P --agent A              # execute next pending task
agencycli run  --project P --agent A --task <id>  # run a specific task
agencycli exec --project P --agent A --prompt "..." # one-shot, no task queue
```

---

## `inbox` — confirmations and async messaging

The inbox has two distinct concepts.

### Task confirmations — agent pauses and waits for your decision

```bash
agencycli inbox list
agencycli inbox list --to cc-connect/pm          # filter by recipient
agencycli inbox show    <task-id>                # summary, action items, log tail
agencycli inbox reply   <task-id> --body "yes, proceed"
agencycli inbox forward <task-id> --to <project>/<agent> --note "..."
```

### Async messages — non-blocking communication between any participants

```bash
# Send (single or group)
agencycli inbox send \
  --from cc-connect/pm \
  --to   cc-connect/dev-claude \
  --subject "New task context" \
  --body "Extra info for the task I just created..."

# Group send — repeat --to
agencycli inbox send \
  --from cc-connect/pm \
  --to cc-connect/dev-claude --to cc-connect/qa-reviewer --to human \
  --subject "Sprint kick-off" \
  --body "New sprint starts Monday."

# Read
agencycli inbox messages                                         # human's mailbox (unread)
agencycli inbox messages --recipient cc-connect/pm              # agent's mailbox
agencycli inbox messages --from human                           # filter by sender
agencycli inbox messages --all                                  # include read messages
agencycli inbox messages --archived                             # show archived
agencycli inbox messages --mark-read                            # mark as read after listing

# Reply
agencycli inbox reply <msg-id> --from <your-address> --body "..."

# Forward
agencycli inbox fwd <msg-id> --from <your-address> --to <recipient>
agencycli inbox fwd <msg-id> --from cc-connect/pm \
  --to cc-connect/dev-claude --to cc-connect/qa-reviewer \
  --note "Please coordinate."

# Per-message status management
agencycli inbox read    <msg-id> --recipient <your-address>   # mark as read
agencycli inbox archive <msg-id> --recipient <your-address>   # archive (hidden from normal list)
agencycli inbox delete  <msg-id> --recipient <your-address>   # permanent delete
agencycli inbox rm      <msg-id> --recipient <your-address>   # alias for delete
```

Agents receive unread messages automatically at the top of their wakeup prompt. No polling needed in `wakeup.md`.

---

## `scheduler` — heartbeat scheduler

The heartbeat is a **non-overlapping wakeup loop**: after each cycle completes all pending tasks, the agent sleeps for `interval`, then wakes again. When the queue is empty, the **wakeup routine** fires instead.

```bash
# Configure heartbeat for one agent
agencycli scheduler heartbeat --project P --agent A \
  --enable --interval 30m \
  --active-hours "09:00-18:00" \  # only wake in this window (local time)
  --active-days  "weekdays"       # Mon–Fri only (or Mon,Wed,Fri / weekends)

# Set a wakeup routine (runs when queue is empty)
agencycli scheduler heartbeat --project P --agent A \
  --wakeup-prompt-file /path/to/wakeup.md

# Start scheduler (all enabled agents)
agencycli scheduler start         # alias: sched, s
agencycli scheduler stop
agencycli scheduler status
```

Overnight ranges like `22:00-06:00` are supported. Startup jitter is applied automatically so agents don't all wake up simultaneously after a restart.

---

## `cron` — scheduled tasks

```bash
agencycli cron add    --project P --agent A \
  --title "Daily standup" --schedule "0 9 * * 1-5" \
  --prompt "Generate a standup report..."
agencycli cron list   --project P --agent A
agencycli cron delete <cron-id>  --project P --agent A
agencycli cron enable <cron-id>  --project P --agent A
agencycli cron disable <cron-id> --project P --agent A
```

Crons enqueue a new task each time the schedule fires. The scheduler checks for due crons on every heartbeat wakeup.

---

## `template` — share agencies

```bash
# Pack the current agency as a shareable template
# Includes: agency-prompt.md, teams/, skills/, agent-playbooks/, project-blueprints/
agencycli template pack --output tech-agency.tar.gz \
  --name "tech-project" --version "1.0.0" \
  --author "Alice" --email "alice@example.com" \
  --description "Standard software engineering agency template" \
  --keywords "engineering,software"

# Inspect a template (local file, directory, or remote URL)
agencycli template info tech-agency.tar.gz
agencycli template info tech-agency.tar.gz --json

# Create an agency from a template
agencycli create agency --name "MyAgency" --template tech-agency.tar.gz
agencycli create agency --name "MyAgency" --template https://example.com/tpl.tar.gz
```

---

## `role` — role management

```bash
agencycli role list  --team engineering
agencycli role skill add    --team engineering --role developer --skill github-push-relay
agencycli role skill remove --team engineering --role developer --skill github-push-relay
```

---

## `session` / `list` / `show` / `version`

```bash
agencycli session show  --project P --agent A
agencycli session clear --project P --agent A
agencycli list teams | projects | agents | skills
agencycli show team engineering
agencycli show project my-api
agencycli show agent my-api dev [--raw]
agencycli version
```
