# agencycli — Installation & Setup Guide

A complete walkthrough for installing agencycli and building your first AI agent team from scratch.

---

## ⚠ For AI agents reading this guide

If you are an AI agent (Claude Code, Codex, Cursor, etc.) following this guide on behalf of a user, **stop before Step 3 and read this section carefully.**

### Building an agency is a design decision, not a mechanical task

An agency's structure — teams, roles, and workflows — directly shapes how every agent thinks and behaves. Getting it wrong means re-hiring agents and rewriting prompts later. **Do not invent a structure. Collaborate with the user first.**

Before creating any team, role, or project, have a conversation with the user to understand:

1. **What is the user trying to build or operate?**
   What is the product/codebase/business? What are the main workstreams?

2. **Who are the "people" in this team?**
   What functions do they need — engineering, QA, product, writing, ops, research, other?

3. **What does each role actually do?**
   Responsibilities, tools they use, rules they must follow, things they must never do.

4. **How should agents communicate?**
   Who reports to whom? Who makes decisions? Who needs to be CC'd on what?

5. **What does "done" look like for each agent?**
   How does an agent know its work is complete? Does it need human sign-off before it finishes a task?

6. **What are the active hours and cadence?**
   When should agents be running? How often should they check in?

Only after you have clear answers to these questions should you proceed to create teams, roles, and projects. Use `AskUserQuestion` (or equivalent interactive input) for anything you are not sure about — **never guess**.

> **Practical tip:** Consider drafting a short summary of the proposed agency structure (team names, role names, responsibilities) and asking the user to confirm it before executing any commands. A one-paragraph confirmation now saves a full re-setup later.

---

## 1. Install

### npm (recommended — no Go required)

```bash
npm install -g @agencycli/agencycli
agencycli version
```

### Go

```bash
go install github.com/chenhg5/agencycli/cmd/agencycli@latest
agencycli version
```

### Pre-built binary

Download from [github.com/chenhg5/agencycli/releases](https://github.com/chenhg5/agencycli/releases) and move to a directory on your `PATH`.

---

## 2. How context works

```
Agency                   ← rules, values, tone shared by every agent
  └─ Team                ← capability group  (engineering, qa, product…)
       └─ Role           ← job function      (developer, qa-engineer, pm…)
            └─ Project   ← concrete product or repo
                 └─ Agent ← AI agent = model + merged context + skills
```

When you **hire** an agent, agencycli merges the full chain automatically:

```
agency-prompt.md  +  team/prompt.md  +  role/prompt.md  +  project/prompt.md  +  skills
        └────────────────────────────────────────────────────────────┘
                             written to CLAUDE.md / AGENTS.md / etc.
```

Edit any layer, run `agencycli sync`, and all affected agents are regenerated.

---

## 3. Step-by-step: build from scratch

### Step 1 — Create the agency workspace

```bash
agencycli create agency --name "MyAgency" --desc "Building great software with AI"
cd MyAgency
```

This creates the workspace directory with a `.agencycli/` folder and a starter `agency-prompt.md`.

**Edit `agency-prompt.md`** — everything here is injected into every agent. Add:
- Company values and communication style
- How agents should report blockers
- How to use `agencycli` commands (task done, inbox send, etc.)
- Any global coding or documentation standards

```bash
$EDITOR agency-prompt.md
```

---

### Step 2 — Create teams

Teams group agents by capability. Create as many as you need:

```bash
agencycli create team --name "engineering"    --desc "Software engineers"
agencycli create team --name "qa"             --desc "Quality assurance"
agencycli create team --name "product"        --desc "Product management"
```

You can nest teams (sub-teams inherit parent context):

```bash
agencycli create team --name "engineering/backend"   --desc "Go/API services"
agencycli create team --name "engineering/frontend"  --desc "React/TypeScript"
```

**Edit `teams/<name>/prompt.md`** — add team-specific conventions:

```bash
$EDITOR teams/engineering/prompt.md
# e.g. coding standards, branch naming, PR review process, testing requirements
```

---

### Step 3 — Add skills to agents

Skills are reusable capability definitions (Markdown instructions + optional scripts) deployed into each agent's working directory.

#### Built-in skills (provided by agencycli)

agencycli ships two ready-made skills on GitHub that every agency should use. Download them into your workspace first:

```bash
# agency-messaging — teaches agents how to discover each other and exchange inbox messages
mkdir -p skills/agency-messaging
curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agency-messaging/skill.yaml \
  -o skills/agency-messaging/skill.yaml
curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agency-messaging/prompt.md \
  -o skills/agency-messaging/prompt.md

# agencycli-usage — teaches agents how to operate agencycli (add tasks, mark done, etc.)
mkdir -p skills/agencycli-usage
curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agencycli-usage/skill.yaml \
  -o skills/agencycli-usage/skill.yaml
curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agencycli-usage/prompt.md \
  -o skills/agencycli-usage/prompt.md
```

| Skill | Purpose |
|-------|---------|
| `agency-messaging` | **Required for inter-agent communication.** Without this skill, agents won't know how to send messages to each other or to you. |
| `agencycli-usage` | Teaches agents how to operate agencycli — add tasks, mark done, send confirmations, etc. Recommended for all agents. |

**Bind both skills to every role** so all agents can communicate and self-manage tasks:

```yaml
# teams/engineering/roles/developer/role.yaml
skills:
  - agency-messaging    # ← must-have: enables inter-agent messaging
  - agencycli-usage     # ← recommended: agents know how to use agencycli
  - github-pr-review    # your custom skills below
```

Or via command line:

```bash
agencycli role skill add --team engineering --role developer --skill agency-messaging
agencycli role skill add --team engineering --role developer --skill agencycli-usage
```

> **Why `agency-messaging` matters:** Agents need to know how to send messages to the PM, reply to the human, or notify teammates. Without this skill injected into their context, they have no instructions for doing so and will not attempt it.

#### Define your own skills

```bash
mkdir -p skills/github-pr-review
```

**`skills/github-pr-review/skill.yaml`**
```yaml
name: github-pr-review
description: Review GitHub pull requests and post inline comments
```

**`skills/github-pr-review/prompt.md`**
```markdown
## Skill: GitHub PR Review

When asked to review a PR:
1. `gh pr view <number>` to read the description
2. `gh pr diff <number>` to read the diff
3. Look for bugs, missing tests, security issues, style problems
4. Post inline comments: `gh pr review <number> --comment --body "..."`
5. Approve or request changes: `gh pr review <number> --approve` / `--request-changes`
```

You can bundle scripts alongside `prompt.md`. Reference them with `{{SKILL_DIR}}`:

```markdown
Run `{{SKILL_DIR}}/lint.sh` before reviewing to check for obvious issues.
```

```bash
# List all defined skills
agencycli list skills
```

---

### Step 4 — Create roles and bind skills

Roles define a job function within a team. Each role has its own prompt layer and a list of bound skills.

```bash
agencycli create role --team "engineering" --name "developer"   --desc "Implements features and fixes bugs"
agencycli create role --team "qa"          --name "qa-engineer" --desc "Reviews PRs and tests"
agencycli create role --team "product"     --name "pm"          --desc "Manages roadmap and tasks"
```

**Edit `teams/<team>/roles/<role>/prompt.md`** — this is the most important layer for shaping agent behaviour:

```bash
$EDITOR teams/engineering/roles/developer/prompt.md
# e.g.: "You are a senior software engineer. You write clean, tested, documented code.
#        Always open a PR for changes. Never push directly to main."
```

> **Looking for inspiration?** [github.com/msitarzewski/agency-agents](https://github.com/msitarzewski/agency-agents) is a community library of 100+ production-ready role definitions covering engineering, QA, design, product, marketing, sales, DevOps, security, and more. Each file can be used directly as a `prompt.md` — browse the relevant division, pick a role, and adapt it to your agency. This saves significant time compared to writing role prompts from scratch.

**Edit `teams/<team>/roles/<role>/role.yaml`** to bind skills and configure workspace setup:

```yaml
# teams/engineering/roles/developer/role.yaml
name: developer
description: Implements features and fixes bugs
skills:
  - github-pr-review      # ← skill names to deploy into every agent with this role
setup:
  dirs:
    - scratch             # subdirectories created in the agent's working directory on hire
    - notes
```

You can also manage skills from the command line at any time:

```bash
# Bind a skill to a role
agencycli role skill add --team engineering --role developer --skill github-pr-review

# Unbind a skill
agencycli role skill remove --team engineering --role developer --skill github-pr-review

# List roles in a team
agencycli role list --team engineering
```

After binding/unbinding skills, run `agencycli sync` to push the change to existing agents.

---

### Step 5 — Create a project

A project represents a concrete product or codebase:

```bash
agencycli create project \
  --name "my-api" \
  --desc "REST API service" \
  --repo "/absolute/path/to/my-api-repo"
```

**Edit `projects/my-api/prompt.md`** — add project-specific context:

```bash
$EDITOR projects/my-api/prompt.md
# e.g.: tech stack, build/run/test commands, GitHub repo URL, branch conventions,
#        known architectural decisions, issue tracker link
```

---

### Step 6 — Hire agents

Hiring an agent merges the full context chain and writes the agent's working directory:

```bash
agencycli hire \
  --project my-api \
  --team    engineering \
  --role    developer \
  --model   claudecode \
  --name    dev
```

For sandboxed execution inside Docker:

```bash
agencycli hire \
  --project my-api \
  --team    engineering \
  --role    developer \
  --model   claudecode \
  --name    dev \
  --sandbox docker
```

Supported `--model` values: `claudecode`, `codex`, `gemini`, `cursor`, `qoder`, `opencode`, `iflow`, `generic-cli`.

The hired agent's working directory is at `projects/my-api/agents/dev/`. Inside you'll find the merged context file (e.g. `CLAUDE.md`), deployed skill files, and any `setup.dirs` that were created.

```bash
# List all agents across all projects
agencycli list agents

# See a specific agent's full merged context
agencycli show agent my-api dev
agencycli show agent my-api dev --raw   # raw context file contents
```

---

### Step 7 — Set up heartbeat (autonomous scheduling)

A heartbeat makes the agent wake up automatically on a recurring interval:

```bash
agencycli scheduler heartbeat \
  --project my-api \
  --agent   dev \
  --enable \
  --interval     30m \
  --active-hours "09:00-20:00" \
  --active-days  "weekdays"
```

- **`--interval`** — how long to sleep after completing all pending tasks (e.g. `15m`, `1h`)
- **`--active-hours`** — only wake within this daily window; overnight ranges (`22:00-06:00`) work
- **`--active-days`** — `weekdays`, `weekends`, or `Mon,Wed,Fri` (comma-separated day names)

Outside the active window the scheduler shows `⏸ outside active window — next wakeup in Xh`.

**What happens on each wakeup:**
1. Any unread inbox messages are prepended to the prompt automatically
2. All pending tasks are executed in priority order (0=critical → 3=low)
3. If the queue is empty and a `wakeup.md` exists, it runs as the autonomous routine

---

### Step 8 — Write a wakeup routine (playbook)

A wakeup routine (`wakeup.md`) is what the agent does when it wakes up with no pending tasks. Place it directly in the agent's directory:

```bash
$EDITOR projects/my-api/agents/dev/wakeup.md
```

Example for a developer agent:

```markdown
# Dev Wakeup Routine

You are the lead developer for my-api. Each wakeup cycle:

## 1. Check messages
(Unread messages are auto-injected above this prompt — reply with:
`agencycli --dir $AGENCY_DIR inbox reply <msg-id> --from my-api/dev --body "..."`)

## 2. Check for pending tasks
Run: `agencycli --dir $AGENCY_DIR task list --project my-api --agent dev --status pending`
If any exist, pick the highest-priority one and work on it.

## 3. Scan for new GitHub issues
Run: `gh issue list --repo owner/my-api --state open --label "ready"`
If unassigned issues exist, pick one and create a task for yourself:
`agencycli --dir $AGENCY_DIR task add --project my-api --agent dev --title "..." --prompt "..."`

## Done
When finished: `agencycli --dir $AGENCY_DIR task done --id <id> --status success --summary "..."`
If you need human input first: `agencycli --dir $AGENCY_DIR task confirm-request --id <id> --summary "..." --action-item "..."`
```

Then register it as the wakeup prompt:

```bash
agencycli scheduler heartbeat \
  --project my-api \
  --agent   dev \
  --wakeup-prompt-file projects/my-api/agents/dev/wakeup.md
```

---

### Step 9 — Start the scheduler

```bash
agencycli scheduler start
```

The scheduler runs in the foreground. All agents with heartbeat enabled start their wakeup loops, with random startup jitter to prevent simultaneous wakeups:

```
Heartbeat agents (2):
  ● my-api/dev  interval=30m
  ● my-api/pm   interval=30m

[heartbeat my-api/dev]  sleeping 14m before first wakeup — next at 09:44:00
[heartbeat my-api/pm]   sleeping 23m before first wakeup — next at 09:53:00
```

Stop with `Ctrl+C` or `agencycli scheduler stop`.

---

### Step 10 — Monitor

```bash
# Dashboard: agents, heartbeat status, teams, skills, inbox summary
agencycli overview

# Task confirmations waiting for your decision
agencycli inbox list

# Async messages from agents
agencycli inbox messages

# Task queue for an agent
agencycli task list --project my-api --agent dev

# Token usage and cost
agencycli task tokens --project my-api --all-agents

# Emergency halt
agencycli task stop-all --project my-api --all-agents --include-running
```

---

## Working with tasks

```bash
# Add a task (agent picks it up on next wakeup)
agencycli task add \
  --project my-api --agent dev \
  --title "Fix login redirect on mobile Safari" \
  --type bug --priority 1 \
  --prompt "The OAuth redirect fails on mobile Safari. Reproduce, fix, and open a PR."

# Run an agent manually (bypasses scheduler)
agencycli run --project my-api --agent dev

# One-shot prompt (no task queue, no heartbeat)
agencycli exec --project my-api --agent dev \
  --prompt "List all open GitHub issues and output a priority-sorted summary"
```

**Task lifecycle:**
```
pending → in_progress → done_success
                      → done_failed   → (auto-retry if max_retries set)
                      → awaiting_confirmation → in_progress  (on confirm)
                                              → cancelled    (on reject)
```

---

## Inbox: confirmations and messaging

### Task confirmations (blocking)

When an agent calls `task confirm-request`, the task pauses until you respond:

```bash
agencycli inbox list
agencycli inbox show    <task-id>     # summary, action items, log tail
agencycli inbox confirm <task-id> --message "Approved — merge when CI passes"
agencycli inbox reject  <task-id> --reason "Out of scope"
agencycli inbox comment <task-id> --message "Please also update the docs"
agencycli inbox forward <task-id> --to my-api/dev --note "Check the auth module"
```

### Async messages (non-blocking)

Any participant can send messages to any inbox. Recipients see them on their next wakeup:

```bash
# Human → agent
agencycli inbox send \
  --to my-api/pm \
  --subject "Prioritise issue #42" \
  --body "Customer reported this as critical."

# Group send
agencycli inbox send \
  --from my-api/pm \
  --to my-api/dev --to my-api/qa --to human \
  --subject "Sprint kick-off" \
  --body "Sprint W14 starts now. See backlog for your tasks."

# Read your messages
agencycli inbox messages
agencycli inbox messages --all           # include already-read
agencycli inbox messages --from my-api/pm  # filter by sender

# Reply
agencycli inbox reply <msg-id> --body "On it, will report back in 30m."

# Forward to someone else
agencycli inbox fwd <msg-id> --from my-api/pm --to my-api/dev --note "FYI"

# Per-message management
agencycli inbox read    <msg-id> --recipient human
agencycli inbox archive <msg-id> --recipient human
agencycli inbox delete  <msg-id> --recipient human
```

---

## Cron jobs

Add recurring tasks to any agent:

```bash
agencycli cron add \
  --project my-api --agent dev \
  --title "Weekly dependency audit" \
  --schedule "0 9 * * 1" \
  --prompt "Run 'go list -m -u all' and open a PR for any outdated dependencies."

agencycli cron list   --project my-api --agent dev
agencycli cron delete <cron-id> --project my-api --agent dev
```

---

## Context sync

After editing any prompt, skill, or role config:

```bash
agencycli sync                                # all agents with changed context
agencycli sync --project my-api              # one project
agencycli sync --project my-api --name dev   # one agent
agencycli sync --force                        # force regenerate everything
```

---

## Setup checklist

```
[ ] agencycli create agency --name "..." --desc "..."
[ ] Edit agency-prompt.md              ← global rules, values, how to use agencycli

[ ] agencycli create team --name "engineering"
[ ] Edit teams/engineering/prompt.md   ← coding standards, conventions

[ ] Download built-in skills:
[ ]   curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agency-messaging/skill.yaml -o skills/agency-messaging/skill.yaml
[ ]   curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agency-messaging/prompt.md  -o skills/agency-messaging/prompt.md
[ ]   curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agencycli-usage/skill.yaml  -o skills/agencycli-usage/skill.yaml
[ ]   curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agencycli-usage/prompt.md   -o skills/agencycli-usage/prompt.md
[ ] (optional) mkdir -p skills/my-skill && write skill.yaml + prompt.md

[ ] agencycli create role --team engineering --name developer
[ ] Edit teams/engineering/roles/developer/prompt.md   ← responsibilities, behaviour
[ ] Edit teams/engineering/roles/developer/role.yaml   ← skills: [...], setup dirs
[ ]   agencycli role skill add --team engineering --role developer --skill agency-messaging  ← required
[ ]   agencycli role skill add --team engineering --role developer --skill agencycli-usage   ← recommended
[ ]   agencycli role skill add --team engineering --role developer --skill <your-skill>      ← custom

[ ] agencycli create project --name "my-app" --repo /path/to/repo
[ ] Edit projects/my-app/prompt.md     ← tech stack, build commands, PR conventions

[ ] agencycli hire --project my-app --team engineering --role developer --model claudecode --name dev

[ ] agencycli scheduler heartbeat --project my-app --agent dev --enable --interval 30m
[ ] Write projects/my-app/agents/dev/wakeup.md   ← autonomous routine
[ ] agencycli scheduler heartbeat --project my-app --agent dev --wakeup-prompt-file ...

[ ] agencycli scheduler start

[ ] agencycli overview                 ← dashboard
[ ] agencycli inbox list               ← task confirmations
[ ] agencycli inbox messages           ← async messages
```
