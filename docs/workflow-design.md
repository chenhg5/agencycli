# agencycli Workflow System Design

> Version: 0.1 Draft  
> Status: Design Phase  
> Scope: Task management, scheduling, human-in-the-loop, agent orchestration

---

## 1. Design Philosophy

### 1.1 Core Problems

The context management layer (`agencycli hire/sync`) answers: *what does an agent know?*  
The workflow layer answers: *what does an agent do, in what order, and who decides?*

Three real tensions to resolve:

| Tension | Decision |
|---------|----------|
| Autonomy vs. Control | Agents act autonomously by default; human confirmation is an explicit opt-in state |
| Push vs. Pull | Agents pull tasks from a queue; humans get pushed a unified inbox |
| Coupling vs. Flexibility | Tasks are files — no central server, no database, fully portable |

### 1.2 What We Are NOT Building

- A general-purpose project management SaaS (not Jira/Linear)
- A real-time orchestration bus (not Temporal/Airflow)
- A multi-agent chat framework (not AutoGen/CrewAI)

We are building a **local-first, file-based task queue + scheduler** that can invoke agent CLIs headlessly and route human confirmation requests to a unified inbox.

---

## 2. Concepts

### 2.1 Actor

An **Actor** is anything that can be assigned a task or can create a task.

| Actor Type | Identifier | Example |
|------------|------------|---------|
| Agent | `<project>/<agent-name>` | `cc-connect/qa-reviewer` |
| Human | `human` | The founder / operator |

### 2.2 Task

The atomic unit of work. A task has:

- A **prompt** (detailed instruction to the agent)
- A **current state** (see §3)
- An **assignee** (actor)
- Optional **triggers** (auto-create tasks in other agents on completion)
- Optional **on_confirm** behavior (what happens when human approves)

### 2.3 Cron

A recurring schedule that generates a task (or runs a prompt) at a defined interval. Crons are owned per-agent.

### 2.4 Inbox

A workspace-level aggregated view of all tasks currently assigned to `human`. The human uses `agencycli inbox` to see and act on these tasks.

### 2.5 Run Log

Every agent execution produces a timestamped log stored alongside the agent's working directory. Logs are the audit trail and debugging surface.

---

## 3. Task State Machine

### 3.1 States

| State | Meaning |
|-------|---------|
| `pending` | Created, not yet started. In the agent's queue. |
| `in_progress` | Agent is actively working on this task right now. |
| `awaiting_confirmation` | Agent has requested human input. Task is archived. Human replies via `inbox reply` and agent sees it on next wakeup. |
| `blocked` | Waiting for a dependency task to complete or an external event. |
| `done_success` | Completed successfully. Archived. |
| `done_failed` | Agent tried but could not complete it (error, gave up). Archived, can be retried. |
| `cancelled` | Will not be executed. Archived. |

### 3.2 Valid Transitions

```
                    ┌─────────────────────────────────────────────────────┐
                    │                                                     │
           retry    │                                                     ▼
pending ──────────► in_progress ──────────────────────────────► done_success
   ▲                    │  │                                        
   │                    │  └──────────────────────────────────► done_failed
   │                    │                                             │
   │         blocked ◄──┤  (dependency not met / external wait)      │ retry
   │              │     │                                             │
   │              └─────►                                             │
   │                    │                                             │
   │                    └──────────────► awaiting_confirmation (archived)│
   │                                                                   │
   │                              human replies via inbox reply        │
   │                              agent sees on next wakeup             │
   └───────────────────────────────────────────────────────────────────┘ (re-open)
```

**Transition table:**

| From | To | Trigger |
|------|----|---------|
| `pending` | `in_progress` | Agent picks up task (manual `run` or scheduler wakeup) |
| `pending` | `cancelled` | Human or PM cancels |
| `in_progress` | `done_success` | Agent reports success |
| `in_progress` | `done_failed` | Agent reports failure / max retries exceeded |
| `in_progress` | `awaiting_confirmation` | Agent emits a confirmation request (task archived) |
| `in_progress` | `blocked` | Dependency not yet done |
| `blocked` | `in_progress` | Dependency resolved |
| `awaiting_confirmation` | `done_success` | Human replies via `inbox reply` → task considered complete |
| `done_failed` | `pending` | Manual retry (`agencycli task retry <id>`) |

### 3.3 Who Drives Transitions

| Transition | Driver |
|------------|--------|
| `pending` → `in_progress` | `agencycli run` command or scheduler |
| `in_progress` → `done_*` | Agent exit code / output parser |
| `in_progress` → `awaiting_confirmation` | Agent calls `task confirm-request` (task archived) |
| `awaiting_confirmation` → `done_success` | Human replies via `inbox reply`; agent sees it on next wakeup |
| `blocked` → `in_progress` | Dependency watcher (scheduler) or manual `agencycli task unblock` |

---

## 4. Data Model

### 4.1 File Layout (per agent)

```
projects/<project>/agents/<name>/
  tasks.yaml          ← active tasks (pending, in_progress, blocked)
  tasks_archive.yaml  ← completed, cancelled, and awaiting_confirmation tasks (append-only)
  crons.yaml          ← recurring schedules
  runs/
    <YYYYMMDD-HHMMSS>-<task-id>.log   ← execution log per run
```

Workspace-level:
```
.agencycli/
  inbox.yaml          ← aggregated view of all tasks assigned to "human"
  inbox.md            ← human-readable version, auto-generated (can be opened in editor)
```

### 4.2 tasks.yaml Schema

```yaml
- id: t-20260316-001
  title: "Review PR #42: Add OAuth login"
  type: review              # feature | bug | review | triage | test | research | chore
  priority: 1               # 0=critical 1=high 2=normal 3=low
  assignee: cc-connect/qa-reviewer   # or "human"
  created_by: cc-connect/pm-agent    # or "human" or "system:cron"
  created_at: 2026-03-16T09:00:00Z
  updated_at: 2026-03-16T09:00:00Z
  started_at: ~
  finished_at: ~
  status: pending

  prompt: |
    A new PR has been opened: https://github.com/org/cc-connect/pull/42
    Title: Add OAuth login
    Author: contributor-a

    Please review this PR:
    1. Check code quality, security, and correctness
    2. Run the test suite mentally (or via CI links)
    3. Post a review comment on GitHub with your findings
    4. If you approve: mark this task done_success
    5. If changes are needed: call `task confirm-request` to archive task and notify human via inbox

  context: {}               # optional extra key-value context injected alongside prompt

  depends_on: []            # list of task IDs that must be done_success first

  on_success:               # auto-create tasks when this task reaches done_success
    - assignee: cc-connect/dev-claude
      title: "Address review comments on PR #42"
      type: bug
      prompt: |
        QA reviewer has approved PR #42 with minor comments.
        Please address the review comments and push updates.

  on_confirmation_required:  # what to show the human in the inbox
    summary: "PR #42 needs your decision: reviewer found a potential security issue"
    action_hint: "Inspect the run log and confirm or reject"

  retry_count: 0
  max_retries: 2
  last_error: ~
```

### 4.3 crons.yaml Schema

```yaml
- id: cron-qa-daily
  title: "Daily PR/Issue scan"
  schedule: "0 9 * * 1-5"   # weekdays at 9am
  enabled: true
  assignee: cc-connect/qa-reviewer
  prompt: |
    Check all open GitHub PRs and new issues opened in the last 24 hours.
    For each:
    - If it is a PR: create a review task for yourself
    - If it is a bug issue: create a triage task for cc-connect/pm-agent
    Use `agencycli task add` to create tasks.
  last_run: ~
  last_run_status: ~

- id: cron-pm-weekly-plan
  title: "Weekly planning"
  schedule: "0 8 * * 1"     # Monday 8am
  enabled: true
  assignee: cc-connect/pm-agent
  prompt: |
    Review all open GitHub issues. For each issue:
    1. Determine type: bug, feature, or wontfix
    2. Assign priority: P0/P1/P2
    3. If clear and P0/P1: assign directly to appropriate dev agent
    4. If ambiguous or P0 critical: route to human inbox for decision
    Use `agencycli task add` and `agencycli task assign` accordingly.
  last_run: ~
  last_run_status: ~
```

### 4.4 Inbox (`.agencycli/inbox.yaml`)

Auto-maintained by the runtime. Never edited directly.

```yaml
- task_id: t-20260316-007
  project: cc-connect
  agent: qa-reviewer
  title: "Security concern in PR #42 - needs your decision"
  summary: "Reviewer found a potential SQL injection vector in the new endpoint"
  action_hint: "Run: agencycli inbox show t-20260316-007 for full context"
  routed_at: 2026-03-16T11:34:00Z
  log_path: projects/cc-connect/agents/qa-reviewer/runs/20260316-113400-t-20260316-001.log
```

And `inbox.md` (auto-generated, human-readable):

```markdown
# 📬 Inbox — 1 item awaiting your confirmation

## [cc-connect / qa-reviewer] Security concern in PR #42
> Routed at: 2026-03-16 11:34

Reviewer found a potential SQL injection vector in the new endpoint.

**Run log:** projects/cc-connect/agents/qa-reviewer/runs/20260316-113400-t-20260316-001.log

**To reply:** `agencycli inbox reply t-20260316-007 --body "LGTM, proceed"`
```

---

## 5. Workflow Examples (cc-connect)

### 5.1 The QA Reviewer Loop

```
[Cron: daily 9am]
      │
      ▼
qa-reviewer wakes up
      │
      ├── gh pr list --state open → new PRs found
      │         │
      │         └── agencycli task add --assignee self --type review \
      │                 --title "Review PR #42" --prompt "..."
      │
      └── gh issue list --state open --label bug → new bugs
                │
                └── agencycli task add --assignee cc-connect/pm-agent \
                        --type triage --title "Triage: issue #88"


[Later: qa-reviewer picks up review task]
      │
      ├── Reviews PR → all good
      │         └── agencycli task done --id t-xxx --status success
      │                   → triggers: create task for dev-claude "Address minor nits"
      │
      └── Reviews PR → security concern found
                └── agencycli task confirm-request --id t-xxx \
                        --summary "Possible injection in endpoint X"
                        → task archived as awaiting_confirmation
                        → message added to human inbox

[Human checks inbox]
      │
      ├── agencycli inbox                     # list
      ├── agencycli inbox show t-xxx          # view detail + log
      └── agencycli inbox reply t-xxx \
              --body "Not an injection, it's parameterized"
                → message delivered to qa-reviewer
                → qa-reviewer sees it on next wakeup and acts accordingly
```

### 5.2 The PM → Dev Pipeline

```
[Cron: Monday 8am]
      │
      ▼
pm-agent wakes up
      │
      ├── gh issue list --state open → issues found
      │
      ├── For clear P1 bug: 
      │       agencycli task add \
      │           --assignee cc-connect/dev-claude \
      │           --type bug --priority 1 \
      │           --title "Fix login redirect on mobile" \
      │           --prompt "<detailed reproduction + fix instructions>"
      │
      └── For ambiguous feature request:
              agencycli task add \
                  --assignee human \
                  --type triage --priority 2 \
                  --title "Scope: add AI-powered search (issue #101)" \
                  --prompt "Issue #101 requests AI search. Is this in scope for Q2?"
                  → routed to human inbox


[dev-claude picks up its task]
      │
      ├── Works on the fix
      ├── Opens a PR
      └── agencycli task done --id t-xxx --status success \
              --on-success-trigger qa-reviewer:"Review PR #55 for fix to issue #77"
              → auto-creates review task for qa-reviewer


[qa-reviewer tests the fix]
      │
      ├── Automated checks pass → done_success
      │       → issue #77 gets a comment, PR merged
      │
      └── Needs manual smoke test:
              agencycli task confirm-request --id t-yyy \
                  --summary "Please smoke test the mobile login fix on your phone"
                  → human inbox
```

### 5.3 The Dev Task Lifecycle in Full

```
Issue #77 opened on GitHub
        │
        ▼ (cron or webhook script)
pm-agent: triage task
        │
        ▼ (pm decides: P1 bug, assign to dev-claude)
dev-claude: task status=pending
        │
        ▼ (scheduler wakes dev-claude)
dev-claude: status=in_progress
        │
        ▼ (dev-claude opens PR)
dev-claude: status=done_success, triggers qa-reviewer
        │
        ▼
qa-reviewer: task status=pending
        │
        ├─► done_success (auto tests pass) → PR merged, issue closed
        │
        └─► awaiting_confirmation (task archived, human notified)
                │
                ▼ (human replies via inbox)
             agent sees reply on next wakeup → continues work
```

---

## 6. CLI Commands

### 6.1 `agencycli task`

```bash
# Create a task
agencycli task add \
  --project cc-connect \
  --agent qa-reviewer \
  --title "Review PR #42" \
  --type review \
  --priority 1 \
  --prompt "Please review..."
  [--depends-on t-xxx,t-yyy]
  [--assignee human]

# List tasks for an agent
agencycli task list --project cc-connect --agent qa-reviewer
agencycli task list --project cc-connect --agent qa-reviewer --status pending

# Show task detail
agencycli task show <task-id>

# Manually run a specific task (one-shot)
agencycli run --project cc-connect --agent qa-reviewer --task <task-id>

# Run next pending task for an agent
agencycli run --project cc-connect --agent qa-reviewer

# Mark done (usually called by the agent itself inside the prompt)
agencycli task done --id <task-id> --status success
agencycli task done --id <task-id> --status failed --error "reason"

# Route to human confirmation
agencycli task confirm-request --id <task-id> --summary "..."

# Retry a failed task
agencycli task retry <task-id>

# Cancel a task
agencycli task cancel <task-id> [--reason "..."]
```

### 6.2 `agencycli cron`

```bash
# Add a cron
agencycli cron add \
  --project cc-connect \
  --agent qa-reviewer \
  --id cron-qa-daily \
  --schedule "0 9 * * 1-5" \
  --title "Daily PR scan" \
  --prompt "..."

# List crons
agencycli cron list --project cc-connect --agent qa-reviewer

# Enable/disable
agencycli cron enable  --project cc-connect --agent qa-reviewer --id cron-qa-daily
agencycli cron disable --project cc-connect --agent qa-reviewer --id cron-qa-daily

# Run a cron immediately (for testing)
agencycli cron run --project cc-connect --agent qa-reviewer --id cron-qa-daily
```

### 6.3 `agencycli inbox`

```bash
# List everything awaiting human confirmation (all projects)
agencycli inbox

# Show detail + log path for a specific item
agencycli inbox show <task-id>

# Reply: human responds, message delivered to agent's next wakeup
agencycli inbox reply <task-id> --body "yes, proceed with merge"
```

### 6.4 `agencycli run` (immediate execution)

```bash
# Run next pending task for an agent
agencycli run --project cc-connect --agent qa-reviewer

# Run a specific task
agencycli run --project cc-connect --agent qa-reviewer --task <task-id>

# Run a cron prompt immediately (without creating a task)
agencycli run --project cc-connect --agent qa-reviewer --cron cron-qa-daily

# Dry run: print what would be executed
agencycli run --project cc-connect --agent qa-reviewer --dry-run
```

### 6.5 `agencycli scheduler` (heartbeat scheduler)

```bash
# Start the scheduler (blocks, or use & / systemd)
agencycli scheduler start

# Status
agencycli scheduler status

# Stop
agencycli scheduler stop
```

---

## 7. Execution Engine

### 7.1 How `agencycli run` Works

```
1. Load agent's tasks.yaml → find next pending task (ordered by priority, then created_at)
2. Transition task: pending → in_progress (write to tasks.yaml, atomic rename)
3. Build the full prompt:
     system_context = read agent's CLAUDE.md / AGENTS.md / etc.
     task_prompt    = task.prompt + injected metadata (task ID, inbox commands hint)
4. Invoke agent CLI:
     cd <agent-working-dir>
     <agent-command> --no-interactive -p "<prompt>" > runs/<timestamp>-<id>.log 2>&1
5. Parse exit code and output:
     exit 0 + no sentinel → done_success
     exit 0 + sentinel AWAIT_CONFIRM → awaiting_confirmation, route to inbox
     exit != 0 → done_failed (or retry if retry_count < max_retries)
6. Execute on_success triggers (create tasks in other agents)
7. Update tasks.yaml, move to tasks_archive.yaml if terminal state
8. Update .agencycli/inbox.yaml and inbox.md if needed
```

### 7.2 Human Confirmation via Inbox

When an agent needs human input before continuing, it calls `task confirm-request`:

```bash
agencycli task confirm-request --id $TASK_ID \
  --summary "PR #42 ready for merge approval" \
  --action-item "Review: gh pr view 42" \
  --action-item "Reply: inbox reply <msg-id> --body 'approve'"
```

This:
1. Archives the task (status → `awaiting_confirmation`)
2. Creates an inbox message for the human

The human replies via `inbox reply`. On the agent's next wakeup, the message is injected at the top of the wakeup prompt so the agent can act on it.

### 7.3 Agent Command Map

| Model | Command |
|-------|---------|
| `claudecode` | `claude --no-interactive -p "..."` |
| `codex` | `codex -p "..."` |
| `cursor` | `cursor --agent -p "..."` *(TBD — cursor headless API)* |
| `gemini` | `gemini -p "..."` |
| `opencode` | `opencode -p "..."` |
| `generic-cli` | Configured per-agent in `.agencycli-agent.yaml` (`run_command` field) |

The `run_command` field in `.agencycli-agent.yaml` allows overriding the default:

```yaml
model: generic-cli
run_command: "my-custom-agent --input-file {prompt_file} --output-dir {run_dir}"
```

### 7.4 Prompt Injection (Task Metadata)

The runner always appends an operational footer to every prompt so agents know how to call back into the system:

```markdown
---
## System Metadata (do not remove)

Task ID: t-20260316-001
Agent:   cc-connect/qa-reviewer

When you complete successfully, run:
  agencycli task done --id t-20260316-001 --status success

If you need human confirmation, run:
  agencycli task confirm-request --id t-20260316-001 --summary "one-line explanation"
  (Then exit 0)

If you cannot complete this task, run:
  agencycli task done --id t-20260316-001 --status failed --error "reason"
```

---

## 8. Notifications & Human Inbox

### Phase 1 — File-based (MVP)

- `inbox.yaml` — machine-readable, for CLI tooling
- `inbox.md` — human-readable Markdown, auto-regenerated on every change

The human runs `agencycli inbox` or simply opens `inbox.md` in their editor/IDE.

### Phase 2 — Push Notifications (via cc-connect)

cc-connect has a driver layer capable of pushing to Feishu/Telegram. We expose a webhook endpoint in `agencycli scheduler`:

```
POST http://localhost:7370/webhook/inbox
```

cc-connect's agent driver calls this when it detects `AGENCYCLI_AWAIT_CONFIRM` in the output, and simultaneously sends a Feishu/Telegram message to the operator. The human can confirm directly from the message (if cc-connect supports reply-to-confirm flow).

---

## 9. OKR Integration

OKRs live in context, not in the task system. This is by design:

- Project OKRs go in `projects/<project>/prompt.md` (a dedicated `## OKRs` section)
- Agency-level OKRs go in `agency-prompt.md`
- The PM agent and Project Director agent are responsible for keeping these up to date
- `agencycli sync` propagates changes to all agent working directories

**Weekly PM agent cron** includes a step:

```
5. Update projects/cc-connect/prompt.md OKR section based on this week's progress
   and run `agencycli sync --project cc-connect` to propagate.
```

This keeps OKRs as living documents without inventing a new data layer.

---

## 10. Heartbeat vs Cron

These are two distinct scheduling primitives and must not be confused.

### 10.1 Heartbeat — Blocking Interval Loop

A heartbeat is an **agent-level recurring wakeup** that works like a heartbeat monitor:

```
[agent idle]
     │
     ├── wait <interval> (e.g., 30 min)
     │
     ▼
[agent wakes up]  ← only if NOT already running
     │
     ├── load pending tasks (priority order)
     ├── run task 1 ──────────────────────────── done
     ├── run task 2 ──────────────────────────── done
     └── run task N ──────────────────────────── done
                │
                └── [agent idles again]
                         │
                         └── wait <interval> from THIS moment
                                  │
                                  ▼
                             [next wakeup]
```

**Key properties:**
- **Non-overlapping**: If the agent is still running, the wakeup is skipped entirely (no duplicate execution)
- **Back-pressure aware**: The interval starts counting AFTER the previous run completes, not at a fixed wall-clock time
- **Session-preserving**: All tasks within a wakeup cycle run in the same agent session (maintaining conversation continuity)
- **Configured per-agent**: Each agent has its own `heartbeat.yaml` with `enabled`, `interval`, and state

This is analogous to OpenClaw's heartbeat mechanism.

### 10.2 Cron — Precise Calendar Schedule

A cron fires at exact calendar times (e.g., "every Monday at 9am"), regardless of what the agent is doing:

```
Monday 9:00:00 ─────────────────────────────────► fires: creates a new task
                                                   (task queues up, agent picks it up on next heartbeat)
Monday 9:01:00  [agent might be busy with other tasks — cron task just sits in queue]
```

**Key properties:**
- **Time-exact**: Fires at the wall-clock time specified (crontab syntax)
- **Non-blocking**: Cron firing just **enqueues a task**; it does not directly invoke the agent
- **Predictable**: Used for scheduled reports, periodic scans, weekly planning, etc.

### 10.3 Comparison Table

| Property | Heartbeat | Cron |
|----------|-----------|------|
| Trigger | Time elapsed since last completion | Absolute calendar time |
| Overlap prevention | Built-in (skips if running) | N/A (just enqueues) |
| Session continuity | Yes — same session per wakeup cycle | No — new task each time |
| Use case | "Check for work and do it" | "Do X every Monday at 9am" |
| Config location | `heartbeat.yaml` per agent | `crons.yaml` per agent |

---

## 11. Session Management

### 11.1 What is a Session?

An **agent session** is the persistent conversation context maintained by the underlying agent CLI between invocations. For example:

- **Claude Code**: uses `--resume <session-id>` to resume a prior conversation thread
- **Codex**: similar session continuation via `--session` flag
- **Others**: may not support sessions; they always start fresh

Sessions enable agents to "remember" prior work across heartbeat cycles, making them more effective over time (e.g., the QA reviewer remembers which PRs it already reviewed).

### 11.2 Session Lifecycle

```
First heartbeat wakeup:
  → agent invoked with no session ID
  → agent starts a new conversation
  → agencycli captures the session ID from output (or via manual set)
  → session ID saved to heartbeat.yaml

Subsequent heartbeat wakeups:
  → agent invoked with --resume <session-id>
  → agent continues the same conversation thread
  → agent "remembers" all prior context

Session reset (manual):
  → agencycli session reset --project <p> --agent <name>
  → session ID cleared from heartbeat.yaml
  → next wakeup starts a fresh conversation
```

### 11.3 Session Storage

Session state lives in `heartbeat.yaml` alongside the heartbeat config:

```yaml
# projects/<project>/agents/<name>/heartbeat.yaml
enabled: true
interval: 30m
pid: 0                      # PID of running agent process (0 = idle)
last_wakeup: 2026-03-16T09:00:00Z
last_wakeup_status: done    # running | done | failed
session_id: "abc123def456"  # current active session ID (empty = no session yet)
session_started_at: 2026-03-01T09:00:00Z
```

### 11.4 Session ID Capture (Claude Code)

When running Claude Code with `--output-format stream-json`, it emits JSON-lines output. The first line is a `system` event containing the session ID:

```json
{"type":"system","subtype":"init","session_id":"abc123def456",...}
```

The runner parses this to capture and persist the session ID automatically.

For models that do not expose a session ID, the ID can be set manually:

```bash
agencycli session set --project cc-connect --agent qa-reviewer --id abc123
agencycli session reset --project cc-connect --agent qa-reviewer
agencycli session show  --project cc-connect --agent qa-reviewer
```

### 11.5 Per-Task vs Per-Cycle Sessions

| Scope | Behaviour |
|-------|-----------|
| Per heartbeat cycle | All tasks in one wakeup share a session — agent builds cumulative context across tasks (recommended for most agents) |
| Per task | Each task gets `--resume` but is treated as a separate thread — useful for isolated, high-stakes tasks |

The default is **per heartbeat cycle** (all tasks in one wakeup share the same session). This can be overridden per-agent in `heartbeat.yaml`:

```yaml
session_scope: cycle   # cycle | task
```

---

## 12. Implementation Phases

### Phase 1 — Task Queue + Heartbeat MVP (2–3 days)

**Goal:** Agents can have tasks; tasks can be run manually; humans have an inbox.

Deliverables:
- `internal/taskstore/` — read/write `tasks.yaml`, `tasks_archive.yaml`, `heartbeat.yaml`
- `internal/runner/` — shell exec wrapper, output parser, sentinel + session ID detection
- `cmd/agencycli/task.go` — `task add/list/show/done/confirm-request/retry/cancel`
- `cmd/agencycli/run.go` — `run` command (manual trigger)
- `cmd/agencycli/inbox.go` — `inbox list/show/confirm/reject/comment`
- `cmd/agencycli/session.go` — `session set/reset/show`
- `cmd/agencycli/scheduler.go` — `scheduler start/heartbeat` (heartbeat loop, no cron yet)
- Inbox auto-generates `inbox.md`
- Heartbeat state tracked in `heartbeat.yaml` with PID-based overlap prevention

### Phase 2 — Cron Scheduling (2–3 days)

**Goal:** Precise calendar-based scheduling via cron expressions.

Deliverables:
- `internal/cronstore/` — read/write `crons.yaml`
- Extend scheduler with robfig/cron engine (cron fires → enqueues task)
- `cmd/agencycli/cron.go` — `cron add/list/enable/disable/run`
- Run log rotation

### Phase 3 — Agent Collaboration (1 week)

**Goal:** Agents create tasks for each other; full pipeline automation.

Deliverables:
- `depends_on` dependency resolution in runner
- `on_success` trigger evaluation (create tasks in other agents)
- `task assign` command (re-assign to different agent)
- cc-connect webhook integration for push notifications
- Scheduler HTTP endpoint (`localhost:7370`)

---

## 11. Directory Structure (Final)

```
<agency-root>/
  .agencycli/
    agency.yaml
    inbox.yaml          ← auto-maintained
    inbox.md            ← auto-generated, human-readable
  agency-prompt.md
  teams/
    engineering/
      team.yaml
      prompt.md
  projects/
    cc-connect/
      project.yaml
      prompt.md         ← includes ## OKRs section
      agents/
        qa-reviewer/
          .agencycli-agent.yaml
          tasks.yaml
          tasks_archive.yaml
          crons.yaml
          runs/
            20260316-090000-t-001.log
          CLAUDE.md       ← generated by formatter
          .claude/
            skills/
        pm-agent/
          ...
        dev-claude/
          ...
        dev-cursor/
          ...
```

---

## 12. Decision Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Execution engine | Shell exec (lightweight) | No external deps; agent CLIs are designed for this |
| State persistence | YAML files | Human-readable, git-friendly, no DB required |
| Human notification | `inbox.md` (Phase 1) + webhook (Phase 2) | Ship now, enhance later |
| OKR storage | Embedded in `prompt.md` context | Reuses existing sync mechanism, no new data layer |
| Task ID format | `t-<YYYYMMDD>-<seq>` | Sortable, readable, collision-resistant within an agency |
| Confirmation re-invocation | Re-run agent with original prompt + human reply appended | Simplest; agent sees full context |
| Cron engine | robfig/cron (Go library) | Standard, battle-tested, no system cron dependency |
