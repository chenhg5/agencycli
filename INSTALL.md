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

### Writing prompts is not optional

Creating the directory structure is only half the work. **The prompts are what make agents intelligent.** An agent with an empty or thin `prompt.md` has no idea who it is, what it owns, or how to behave — it will produce generic, unreliable results.

After every `agencycli create` command, you **must** write a substantive prompt for that entity before moving on. The three prompt layers are:

| Layer | File | What it shapes |
|-------|------|----------------|
| Agency | `agency-prompt.md` | Identity, values, and universal rules shared by **every** agent |
| Team | `teams/<name>/prompt.md` | Standards, tools, and norms for agents in this team |
| Role | `teams/<t>/roles/<r>/prompt.md` | Who this agent is, what it owns, how it works, what "done" means |

The role prompt is the most important — spend the most time on it. See Steps 1, 2, and 4 below for concrete templates.

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

### From source (includes web console)

```bash
git clone https://github.com/chenhg5/agencycli && cd agencycli
make build      # builds web frontend + Go binary → dist/agencycli
# or
make install    # builds and installs to $GOPATH/bin
```

> **Requires:** Go 1.22+, Node.js 18+ (for building the web frontend).

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

**Optional: set the agency language** in `.agencycli/agency.yaml` — this controls the language of all auto-generated scheduler messages (inbox notifications, wakeup triggers, etc.):

```yaml
# .agencycli/agency.yaml
name: MyAgency
lang: zh   # "zh" for Chinese, "en" for English (default)
```

**Edit `agency-prompt.md`** — this is injected into **every** agent across every team. It defines the company's identity, values, and universal rules. Fill it in fully before creating any teams.

```bash
$EDITOR agency-prompt.md
```

A well-written `agency-prompt.md` should cover:

```markdown
# <Agency Name>

## Mission
<1–2 sentences: what this company/project is building and why it exists>

## Core values
- <Value 1>: <Brief description — e.g. "Quality over speed: we ship tested, reviewed code">
- <Value 2>: <e.g. "Transparency: surface blockers immediately, never stay stuck silently">
- <Value 3>: <e.g. "Ownership: if you see a problem, you own fixing it">

## Communication style
<Tone and style expected in all output — e.g. "Be concise and direct. Prefer bullet points
over prose. Use plain language.">

## Universal rules (apply to every agent)
- Never push directly to main — always open a PR
- Never delete files or code without explicit instruction
- When blocked for more than 30 minutes, report via inbox and pause
- Always confirm with the user before any irreversible action (deploy, delete, send email)
- Keep task summaries factual — record what you did, not what you intended to do

## How to use agencycli (mandatory reading)
Every agent must know these commands:

  # Complete a task
  agencycli --dir $AGENCY_DIR task done --id <id> --status success --summary "..."

  # Request human confirmation before proceeding
  agencycli --dir $AGENCY_DIR task confirm-request --id <id> \
    --summary "..." --action-item "Approve or reject"

  # Send a message to another participant
  agencycli --dir $AGENCY_DIR inbox send \
    --from <project>/<agent> --to human \
    --subject "..." --body "..."

  # Reply to a message
  agencycli --dir $AGENCY_DIR inbox reply <msg-id> \
    --from <project>/<agent> --body "..."

## Escalation policy
<Who to message when something is blocked or uncertain — e.g. "Always send blockers
to 'human'. Copy the PM role when starting or finishing major features.">
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

**Edit `teams/<name>/prompt.md`** — every agent on this team inherits this context. It defines team-level standards, norms, and tooling that all roles within the team share.

```bash
$EDITOR teams/engineering/prompt.md
```

A well-written team prompt should cover:

```markdown
# <Team Name> Team

## Team mission
<What this team is responsible for and what success looks like for the team>

## Standards and conventions
- **Language / stack:** <e.g. Go 1.22, PostgreSQL, Docker>
- **Branch naming:** `<type>/<issue-number>-<short-desc>` (e.g. `fix/142-login-redirect`)
- **Commit style:** <e.g. Conventional Commits: feat/fix/chore/docs/test>
- **PR process:** <e.g. All PRs require 1 approval + passing CI before merge>
- **Test requirements:** <e.g. New features must include unit tests; coverage must not drop>
- **Code style:** <e.g. Run gofmt/eslint before committing; no warnings left unresolved>

## Tools available to all roles in this team
- `gh` — GitHub CLI (issues, PRs, releases)
- <any other team-wide tools — e.g. `jq`, `docker`, `kubectl`>

## Team communication norms
- <e.g. "The PM role coordinates task assignment — check in with pm before starting new work">
- <e.g. "QA must sign off on every feature PR before it is merged">
- <e.g. "Post a brief status update to human inbox at the end of each work cycle">

## What "quality work" means for this team
<Define the bar — e.g. "Code is well-tested, documented, reviewed, and doesn't break
existing functionality. Documentation is updated alongside code changes.">
```

---

### Step 3 — Add skills to agents

Skills are reusable capability definitions (Markdown instructions + optional scripts) deployed into each agent's working directory.

#### Built-in skills (provided by agencycli)

agencycli ships two ready-made skills on GitHub that every agency should use. Download them into your workspace first:

```bash
# agency-messaging — teaches agents how to discover each other and exchange inbox messages
mkdir -p skills/agency-messaging
curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agency-messaging/SKILL.md \
  -o skills/agency-messaging/SKILL.md

# agencycli-usage — teaches agents how to operate agencycli (add tasks, mark done, etc.)
mkdir -p skills/agencycli-usage
curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agencycli-usage/SKILL.md \
  -o skills/agencycli-usage/SKILL.md
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

**`skills/github-pr-review/SKILL.md`**
```markdown
---
name: github-pr-review
description: Review GitHub pull requests and post inline comments
---

## Skill: GitHub PR Review

When asked to review a PR:
1. `gh pr view <number>` to read the description
2. `gh pr diff <number>` to read the diff
3. Look for bugs, missing tests, security issues, style problems
4. Post inline comments: `gh pr review <number> --comment --body "..."`
5. Approve or request changes: `gh pr review <number> --approve` / `--request-changes`
```

You can bundle scripts alongside `SKILL.md`. Reference them with `{{SKILL_DIR}}`:

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

**Edit `teams/<team>/roles/<role>/prompt.md`** — this is the most important layer. It defines who this agent is, what it owns, how it works, where its files live, and what "done" means. **Write it fully before hiring any agent with this role.**

```bash
$EDITOR teams/engineering/roles/developer/prompt.md
```

A well-written role prompt must cover all of the following:

```markdown
# Role: <Role Title>

## Identity
You are a <seniority + job title> on the <team name> team at <agency name>.
<1–2 sentences describing the agent's personality and working style — e.g.
"You are pragmatic and delivery-focused. You write clean, tested, well-documented
code and communicate proactively.">

## Responsibilities
- <Primary responsibility — e.g. "Implement features and bug fixes as assigned by the PM">
- <e.g. "Write unit and integration tests for all code you produce">
- <e.g. "Open a PR for every change; never commit directly to main">
- <e.g. "Respond to PR review comments within your next active wakeup cycle">
- <e.g. "Keep the PM informed of progress and blockers via inbox">

## Work scope and boundaries
**You own:**
- <e.g. All files under src/, internal/, cmd/ for the assigned project>
- <e.g. Writing and updating tests in the test/ directory>

**You do NOT:**
- <e.g. Modify CI/CD configuration or deployment scripts — escalate to DevOps role>
- <e.g. Merge your own PRs — always request review first>
- <e.g. Make product decisions — ask the PM role>
- <e.g. Access production infrastructure directly>

## Your workspace
Your working directory is at: projects/<project>/agents/<your-name>/

  CLAUDE.md (or AGENTS.md)   ← your full context — auto-generated, do not edit manually
  tasks/                     ← task queue (managed by agencycli, do not edit manually)
  notes/                     ← your persistent notes and research (you own this)
  scratch/                   ← temporary files, experiments, drafts

Always read your notes/ directory at the start of each wakeup cycle for continuity.

## Your skills
The following skills are deployed in your working directory. Read each SKILL.md
at the start of your first session so you know how to use them:

- `agency-messaging`  — How to discover other agents and send/receive inbox messages
- `agencycli-usage`   — How to operate agencycli: tasks, inbox, sync, overview
- `<your-skill-name>` — <One-line description of what this skill enables>

## Communication
- **Report blockers immediately** — never stay stuck. Send to human inbox after 30 min:
  `agencycli --dir $AGENCY_DIR inbox send --from <project>/<name> --to human --subject "Blocked: ..." --body "..."`
- **Request confirmation before irreversible actions** (merges, deploys, deletes):
  `agencycli --dir $AGENCY_DIR task confirm-request --id <id> --summary "..." --action-item "..."`
- **Update the PM** when starting or finishing significant work:
  `agencycli --dir $AGENCY_DIR inbox send --from <project>/<name> --to <project>/pm ...`

## Definition of done
A task is complete when **all** of the following are true:
1. <e.g. Code is committed and a PR is opened with a clear description>
2. <e.g. All tests pass — run the test suite before marking done>
3. <e.g. Relevant documentation is updated if the public interface changed>
4. <e.g. task done is called with a clear summary of what was done>

If human review is required before marking done, use `task confirm-request` instead.
```

> **Looking for inspiration?** [github.com/msitarzewski/agency-agents](https://github.com/msitarzewski/agency-agents) is a community collection of role definitions covering engineering, QA, design, product, marketing, sales, and more. Browse it as a reference when writing your own `prompt.md`.

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
3. If the queue is empty and a `wakeup.md` exists (in `.agencycli/context/`), it runs as the autonomous routine

---

### Step 8 — Write a wakeup routine (playbook)

When the agent wakes up with an empty task queue, the scheduler sends it a wakeup prompt.

#### Recommended approach — put the SOP in the role prompt, keep wakeup.md short

The detailed routine (what to check, what commands to run) belongs in the **role prompt** (`teams/<team>/roles/<role>/prompt.md`), which is baked into `CLAUDE.md` and loaded once per session. The `wakeup.md` then only needs to be a short trigger:

```bash
$EDITOR projects/my-api/agents/dev/.agencycli/context/wakeup.md
```

**English trigger example:**
```markdown
Execute your wakeup routine. Check pending tasks, unread messages, and your scheduled activities.
Refer to your role context for the detailed steps.
```

**Chinese trigger example:**
```markdown
执行你的唤醒例程。检查待处理任务、未读消息及计划中的工作事项。
详细步骤请参阅你的角色上下文。
```

The scheduler automatically prepends any unread inbox messages before this trigger (in the language configured in `.agencycli/agency.yaml`).

> **Tip:** If you don't configure a `wakeup_prompt` at all, the scheduler will send a built-in default trigger in the agency language. This is the easiest way to get started.

#### Add the detailed SOP to the role prompt

In `teams/<team>/roles/<role>/prompt.md`, add a `## Wakeup Routine` section:

```markdown
## Wakeup Routine

Each time you are woken up with no pending tasks:

1. Check messages — unread messages are injected automatically before your prompt
2. Scan for new GitHub issues: `gh issue list --repo owner/my-api --state open --label "ready"`
3. If unassigned issues exist, pick one and create a task:
   `agencycli --dir $AGENCY_DIR task add --project my-api --agent dev --title "..." --prompt "..." --created-by my-api/pm`
4. When done: `agencycli --dir $AGENCY_DIR task done --id <id> --status success --summary "..."`
```

Then register the wakeup trigger file:

```bash
agencycli scheduler heartbeat \
  --project my-api \
  --agent   dev \
  --wakeup-prompt-file projects/my-api/agents/dev/.agencycli/context/wakeup.md
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
  --type bug --priority 1 --created-by human \
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
                      → awaiting_confirmation → done_success (via inbox reply)
```

---

## Inbox: confirmations and messaging

### Task confirmations (blocking)

When an agent calls `task confirm-request`, the task pauses until you respond:

```bash
agencycli inbox list
agencycli inbox show    <task-id>     # summary, action items, log tail
agencycli inbox reply   <task-id> --body "Approved — merge when CI passes"
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
─── Agency ────────────────────────────────────────────────────────────────────
[ ] agencycli create agency --name "..." --desc "..."
[ ] Write agency-prompt.md             ← REQUIRED: mission, values, universal rules,
                                          agencycli command reference, escalation policy

─── Teams (repeat for each team) ──────────────────────────────────────────────
[ ] agencycli create team --name "engineering"
[ ] Write teams/engineering/prompt.md  ← REQUIRED: team mission, stack/standards,
                                          tools, PR process, definition of quality

─── Skills ─────────────────────────────────────────────────────────────────────
[ ] Download built-in skills:
[ ]   mkdir -p skills/agency-messaging && \
        curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agency-messaging/SKILL.md \
          -o skills/agency-messaging/SKILL.md
[ ]   mkdir -p skills/agencycli-usage && \
        curl -sL https://raw.githubusercontent.com/chenhg5/agencycli/main/skills/agencycli-usage/SKILL.md \
          -o skills/agencycli-usage/SKILL.md
[ ] (optional) mkdir -p skills/my-skill && write skills/my-skill/SKILL.md

─── Roles (repeat for each role) ──────────────────────────────────────────────
[ ] agencycli create role --team engineering --name developer
[ ] Write teams/engineering/roles/developer/prompt.md   ← REQUIRED: identity,
        responsibilities, work scope & boundaries, workspace folder explanation,
        skills overview, communication norms, definition of done
[ ] Edit teams/engineering/roles/developer/role.yaml    ← skills: [...], setup dirs
[ ]   agencycli role skill add --team engineering --role developer --skill agency-messaging
[ ]   agencycli role skill add --team engineering --role developer --skill agencycli-usage
[ ]   agencycli role skill add --team engineering --role developer --skill <your-skill>

─── Project ────────────────────────────────────────────────────────────────────
[ ] agencycli create project --name "my-app" --repo /path/to/repo
[ ] Write projects/my-app/prompt.md    ← REQUIRED: tech stack, build/test/run commands,
                                          repo URL, branch conventions, architectural notes

─── Hire ───────────────────────────────────────────────────────────────────────
[ ] agencycli hire --project my-app --team engineering --role developer --model claudecode --name dev
[ ] agencycli show agent my-app dev --raw   ← verify merged context looks correct

─── Heartbeat ──────────────────────────────────────────────────────────────────
[ ] agencycli scheduler heartbeat --project my-app --agent dev --enable --interval 30m
[ ] Write projects/my-app/agents/dev/.agencycli/context/wakeup.md
[ ] agencycli scheduler heartbeat --project my-app --agent dev --wakeup-prompt-file ...

─── Start ──────────────────────────────────────────────────────────────────────
[ ] agencycli scheduler start
[ ] agencycli overview                 ← CLI dashboard
[ ] agencycli inbox list               ← task confirmations
[ ] agencycli inbox messages           ← async messages

─── Web console (optional) ─────────────────────────────────────────────────────
[ ] agencycli start                    ← opens http://127.0.0.1:27892
[ ] agencycli start --addr 0.0.0.0:8080 --api-key <token>  ← remote access
```
