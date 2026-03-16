# agencycli

**agencycli** is a CLI tool for managing AI agent context through an agency-style organisational structure. It lets you define teams, projects, and skills once — then hire (or assign) agents into projects with a fully assembled context that they can load immediately.

No more copy-pasting prompts between sessions. No more context drift between agents working on the same project.

## How it works

agencycli models your AI workflow as an agency:

- **Agency** — global rules and values shared by every agent
- **Teams** — capability groups (engineering, growth, product…) with their own standards and skills. Teams can be nested (`engineering/backend`)
- **Projects** — concrete products or initiatives with their own goals and tech stack
- **Hire / Assign** — assign an agent to a project. agencycli merges the full context chain (agency → team → project) and writes it into an agent working directory ready to use

```
agency-prompt.md
  └─ teams/engineering/prompt.md
       └─ teams/engineering/backend/prompt.md
            └─ projects/my-api/prompt.md
                 └─ projects/my-api/agents/dev/   ← hire produces this
                      ├─ CLAUDE.md                ← ready for `claude`
                      └─ .claude/skills/          ← skills auto-loaded
```

The agent working directory contains everything the agent needs. Just `cd` in and start your agent.

## Supported agents

| `--model`     | Output files                                                      |
|---------------|-------------------------------------------------------------------|
| `claudecode`  | `CLAUDE.md` with `@import` layers + `.claude/skills/`            |
| `codex`       | `AGENTS.md` single merged file (skills inlined)                  |
| `cursor`      | `.cursorrules` + `.cursor/rules/agencycli.mdc`                   |
| `gemini`      | `GEMINI.md` with `@import` layers + `.gemini/skills/`            |
| `qoder`       | `AGENTS.md` single merged file                                   |
| `opencode`    | `OPENCODE.md` single merged file                                 |
| `iflow`       | `IFLOW.md` single merged file                                    |
| `generic-cli` | `context.md` plain text                                          |

## Installation

```bash
go install github.com/agencycli/agencycli/cmd/agencycli@latest
```

Or build from source:

```bash
git clone https://github.com/agencycli/agencycli
cd agencycli
make install
```

## Quick start

```bash
# 1. Create an agency workspace
agencycli create agency --name "MyAgency" --desc "Building great software"
cd MyAgency

# 2. Create teams
agencycli create team --name "engineering" --desc "Software engineering"
agencycli create team --name "engineering/backend" --desc "Go/gRPC services" --skills "git,bash"
agencycli create team --name "growth" --desc "Growth and marketing"
agencycli create team --name "growth/seo" --desc "SEO and content"

# 3. Edit team prompts (add your standards and conventions)
vim teams/engineering/prompt.md
vim teams/engineering/backend/prompt.md

# 4. Create a project
agencycli create project --name "my-api" --desc "REST API service" --repo "../my-api"
vim projects/my-api/prompt.md

# 5. Hire agents (hire and assign are identical)
agencycli hire   --project "my-api" --team "engineering/backend" --model "claudecode" --name "dev"
agencycli assign --project "my-api" --team "growth/seo"          --model "codex"      --name "seo"

# 6. Start working
cd projects/my-api/agents/dev
claude
```

## Workspace layout

```
MyAgency/
  .agencycli/
    agency.yaml                # agency metadata
  agency-prompt.md             # agency-wide context (edit this)

  teams/
    engineering/
      team.yaml
      prompt.md                # edit: engineering standards
      backend/
        team.yaml
        prompt.md              # edit: backend-specific rules
    growth/
      team.yaml
      prompt.md
      seo/
        team.yaml
        prompt.md

  projects/
    my-api/
      project.yaml
      prompt.md                # edit: project goals and tech stack
      agents/
        dev/                   # claudecode agent working directory
          CLAUDE.md            ← @imports all context layers
          .agencycli-context/  ← individual layer files (managed by agencycli)
          .claude/
            skills/
              git/SKILL.md
              bash/SKILL.md
        seo/                   # codex agent working directory
          AGENTS.md            ← single merged file

  skills/
    git/
      skill.yaml
      prompt.md                # edit or extend
    bash/
      skill.yaml
      prompt.md
    github/
      skill.yaml
      prompt.md
```

## Commands

### `agencycli create agency`

Initialise a new workspace directory.

```
agencycli create agency --name <name> [--desc <description>]
```

Creates the workspace directory, installs built-in skills (git, bash, github), and generates template prompt files.

### `agencycli create team`

Create a team. Supports nested paths.

```
agencycli create team --name <path> [--desc <description>] [--skills <skill1,skill2>]
```

Examples:
```bash
agencycli create team --name "engineering"
agencycli create team --name "engineering/backend" --desc "Go/gRPC" --skills "git,bash"
```

Parent teams must exist before creating a child team — agencycli enforces this so every level in the context chain has its own `team.yaml` and `prompt.md`.

### `agencycli create project`

Create a project.

```
agencycli create project --name <name> [--desc <description>] [--repo <path>]
```

### `agencycli hire` / `agencycli assign`

Assemble context and create an agent working directory. `hire` and `assign` are identical — use whichever feels natural.

```
agencycli hire \
  --project <project-name> \
  --team    <team-path> \
  --model   <claudecode|codex|cursor|gemini|qoder|opencode|iflow|generic-cli> \
  --name    <agent-name> \
  [--extra-prompt <file>] \
  [--force]
```

The context is assembled in this order:
1. `agency-prompt.md`
2. `teams/<parent>/prompt.md` (each level in the chain)
3. `teams/<team>/prompt.md`
4. `projects/<project>/prompt.md`
5. `--extra-prompt` file (if provided)

### `agencycli sync`

Regenerate agent working directories whose context has changed.

```
agencycli sync [--project <name>] [--name <agent>] [--force]
```

agencycli stores a SHA-256 hash of each prompt layer in `.agencycli-agent.yaml`. Running `sync` compares current file contents against stored hashes and only rewrites files that have changed.

```bash
agencycli sync                               # sync all agents in all projects
agencycli sync --project my-api             # sync all agents in one project
agencycli sync --project my-api --name dev  # sync one specific agent
agencycli sync --force                       # force-regenerate everything
```

### `agencycli list`

```bash
agencycli list teams    # list all teams
agencycli list projects # list all projects
agencycli list agents   # list all hired agents
agencycli list skills   # list available skills
```

### `agencycli show`

```bash
agencycli show team    engineering/backend
agencycli show project my-api
agencycli show agent   my-api dev           # summary
agencycli show agent   my-api dev --raw     # print full merged context
```

### Global flag: `--dir`

Run any command against a workspace that is not your current directory:

```bash
agencycli --dir /path/to/MyAgency list agents
agencycli --dir /path/to/MyAgency sync
```

## Context inheritance

Context flows from the most general to the most specific. Later layers can override earlier ones.

```
agency                ← applies to every agent everywhere
  └─ team             ← shared by all agents in this team
       └─ sub-team    ← more specific capability group
            └─ project ← project goals, tech stack, conventions
```

Skills are collected from the entire team chain and deduplicated. A skill defined at `engineering` is available to `engineering/backend` without repeating it.

## Editing prompts

Every `prompt.md` is plain Markdown — write whatever instructions you want the agent to follow. There is no special syntax required.

After editing a prompt, run `agencycli sync` to push the changes into all affected agent directories.

```bash
vim teams/engineering/backend/prompt.md
agencycli sync
```

## Built-in skills

The following skills are installed automatically when you create an agency:

| Skill    | Description                                      |
|----------|--------------------------------------------------|
| `git`    | Git commit conventions, branching, common flows  |
| `bash`   | Shell scripting best practices                   |
| `github` | GitHub CLI (`gh`) usage for PRs and issues       |

Add custom skills by creating directories under `skills/`:

```
skills/
  docker/
    skill.yaml       # name + description
    prompt.md        # the skill instructions
```

Then reference the skill in a team:

```bash
agencycli create team --name "engineering/backend" --skills "git,bash,docker"
```

Or add it to an existing team's `team.yaml` and run `agencycli sync`.

## Workflow tips

**Multiple agents, same project**

Hire multiple agents from different teams for the same project:

```bash
agencycli hire --project my-api --team engineering/backend --model claudecode --name dev
agencycli hire --project my-api --team qa                  --model claudecode --name reviewer
agencycli hire --project my-api --team growth              --model codex      --name writer
```

Each agent gets only the context relevant to its team.

**Multiple projects**

Each project is independent. Hire the same team's agent into multiple projects — they each get their own working directory and context.

**Keeping context lean**

Claude Code recommends keeping each `CLAUDE.md` file under 200 lines. agencycli uses `@import` to split context across multiple files (stored in `.agencycli-context/`), so each layer stays small and Claude's adherence remains high.

For Codex the combined `AGENTS.md` must stay under 32 KiB (the default limit). Use `agencycli show agent <project> <name>` to check total line counts.

## Roadmap

- [x] Agency / team / project scaffolding
- [x] Context merging with team chain inheritance
- [x] Claude Code formatter (CLAUDE.md + @import + skills)
- [x] Codex / Qoder formatter (AGENTS.md single file)
- [x] Cursor formatter (.cursorrules + .cursor/rules/)
- [x] Gemini CLI formatter (GEMINI.md + @import + skills)
- [x] OpenCode / iFlow formatter (single file)
- [x] Generic CLI formatter (context.md)
- [x] `sync` with SHA-256 change detection
- [x] `assign` alias for `hire`
- [x] `--dir` flag for remote workspace access
- [ ] Agent task management (TODO.md, task assignment)
- [ ] Cron-based agent scheduling
- [ ] GitHub webhook → auto-create review tasks

## License

MIT
