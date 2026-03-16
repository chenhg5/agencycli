# agentorg

**agentorg** is a CLI tool for managing AI agent context through a company-style organisational structure. It lets you define departments, projects, and skills once — then hire agents into projects with a fully assembled context that they can load immediately.

No more copy-pasting prompts between sessions. No more context drift between agents working on the same project.

## How it works

agentorg models your AI workflow as a company:

- **Company** — global rules and values shared by every agent
- **Departments** — capability groups (engineering, growth, product…) with their own standards and skills. Departments can be nested (`engineering/backend`)
- **Projects** — concrete products or initiatives with their own goals and tech stack
- **Hire** — assign an agent to a project. agentorg merges the full context chain (company → department → project) and writes it into an agent working directory ready to use

```
company-prompt.md
  └─ departments/engineering/prompt.md
       └─ departments/engineering/backend/prompt.md
            └─ projects/my-api/prompt.md
                 └─ projects/my-api/agents/dev/   ← hire produces this
                      ├─ CLAUDE.md                ← ready for `claude`
                      └─ .claude/skills/          ← skills auto-loaded
```

The agent working directory contains everything the agent needs. Just `cd` in and start your agent.

## Supported agents

| `--model`     | Output files                                      |
|---------------|---------------------------------------------------|
| `claude-code` | `CLAUDE.md` with `@import` layers + `.claude/skills/` |
| `codex`       | `AGENTS.md` single merged file (skills inlined)   |
| `generic-cli` | `context.md` plain text                           |

## Installation

```bash
go install github.com/agentorg/agentorg/cmd/agentorg@latest
```

Or build from source:

```bash
git clone https://github.com/agentorg/agentorg
cd agentorg
make install
```

## Quick start

```bash
# 1. Create a company workspace
agentorg create company --name "MyCompany" --desc "Building great software"
cd MyCompany

# 2. Create departments
agentorg create dept --name "engineering" --desc "Software engineering"
agentorg create dept --name "engineering/backend" --desc "Go/gRPC services" --skills "git,bash"
agentorg create dept --name "growth" --desc "Growth and marketing"
agentorg create dept --name "growth/seo" --desc "SEO and content"

# 3. Edit department prompts (add your standards and conventions)
vim departments/engineering/prompt.md
vim departments/engineering/backend/prompt.md

# 4. Create a project
agentorg create project --name "my-api" --desc "REST API service" --repo "../my-api"
vim projects/my-api/prompt.md

# 5. Hire agents
agentorg hire --project "my-api" --dept "engineering/backend" --model "claude-code" --name "dev"
agentorg hire --project "my-api" --dept "growth/seo"          --model "codex"       --name "seo"

# 6. Start working
cd projects/my-api/agents/dev
claude
```

## Workspace layout

```
MyCompany/
  .aios/
    company.yaml               # company metadata
  company-prompt.md            # company-wide context (edit this)

  departments/
    engineering/
      dept.yaml
      prompt.md                # edit: engineering standards
      backend/
        dept.yaml
        prompt.md              # edit: backend-specific rules
    growth/
      dept.yaml
      prompt.md
      seo/
        dept.yaml
        prompt.md

  projects/
    my-api/
      project.yaml
      prompt.md                # edit: project goals and tech stack
      agents/
        dev/                   # claude-code agent working directory
          CLAUDE.md            ← @imports all context layers
          .aios-context/       ← individual layer files (managed by agentorg)
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

### `agentorg create company`

Initialise a new workspace directory.

```
agentorg create company --name <name> [--desc <description>]
```

Creates the workspace directory, installs built-in skills (git, bash, github), and generates template prompt files.

### `agentorg create dept`

Create a department. Supports nested paths.

```
agentorg create dept --name <path> [--desc <description>] [--skills <skill1,skill2>]
```

Examples:
```bash
agentorg create dept --name "engineering"
agentorg create dept --name "engineering/backend" --desc "Go/gRPC" --skills "git,bash"
```

### `agentorg create project`

Create a project.

```
agentorg create project --name <name> [--desc <description>] [--repo <path>]
```

### `agentorg hire`

Assemble context and create an agent working directory.

```
agentorg hire \
  --project <project-name> \
  --dept    <dept-path> \
  --model   <claude-code|codex|generic-cli> \
  --name    <agent-name> \
  [--extra-prompt <file>] \
  [--force]
```

The context is assembled in this order:
1. `company-prompt.md`
2. `departments/<parent>/prompt.md` (each level in the chain)
3. `departments/<dept>/prompt.md`
4. `projects/<project>/prompt.md`
5. `--extra-prompt` file (if provided)

### `agentorg sync`

Regenerate agent working directories whose context has changed.

```
agentorg sync [--project <name>] [--name <agent>] [--force]
```

agentorg stores a SHA-256 hash of each prompt layer in `.aios-agent.yaml`. Running `sync` compares current file contents against stored hashes and only rewrites files that have changed.

```bash
agentorg sync                           # sync all agents in all projects
agentorg sync --project my-api         # sync all agents in one project
agentorg sync --project my-api --name dev  # sync one specific agent
```

### `agentorg list`

```bash
agentorg list depts     # list all departments
agentorg list projects  # list all projects
agentorg list agents    # list all hired agents
agentorg list skills    # list available skills
```

### `agentorg show`

```bash
agentorg show dept    engineering/backend
agentorg show project my-api
agentorg show agent   my-api dev           # summary
agentorg show agent   my-api dev --raw     # print full merged context
```

## Context inheritance

Context flows from the most general to the most specific. Later layers can override earlier ones.

```
company                  ← applies to every agent everywhere
  └─ department          ← shared by all agents in this department
       └─ sub-department ← more specific capability group
            └─ project   ← project goals, tech stack, conventions
```

Skills are collected from the entire department chain and deduplicated. A skill defined at `engineering` is available to `engineering/backend` without repeating it.

## Editing prompts

Every `prompt.md` is plain Markdown — write whatever instructions you want the agent to follow. There is no special syntax required.

After editing a prompt, run `agentorg sync` to push the changes into all affected agent directories.

```bash
vim departments/engineering/backend/prompt.md
agentorg sync
```

## Built-in skills

The following skills are installed automatically when you create a company:

| Skill    | Description                                      |
|----------|--------------------------------------------------|
| `git`    | Git commit conventions, branching, common flows  |
| `bash`   | Shell scripting best practices                   |
| `github` | GitHub CLI (`gh`) usage for PRs and issues       |

Add more skills by creating directories under `skills/`:

```
skills/
  docker/
    skill.yaml       # name + description
    prompt.md        # the skill instructions
```

Then reference the skill in a department:

```bash
agentorg create dept --name "engineering/backend" --skills "git,bash,docker"
```

## Workflow tips

**Multiple agents, same project**

Hire multiple agents from different departments for the same project:

```bash
agentorg hire --project my-api --dept engineering/backend --model claude-code --name dev
agentorg hire --project my-api --dept engineering/test    --model claude-code --name tester
agentorg hire --project my-api --dept growth/seo          --model codex       --name seo
```

Each agent gets only the context relevant to its department.

**Multiple projects**

Each project is independent. Hire the same department agent into multiple projects — they each get their own working directory and context.

**Keeping context lean**

Claude Code recommends keeping each `CLAUDE.md` file under 200 lines. agentorg uses `@import` to split context across multiple files, so each layer stays small and Claude's adherence remains high.

For Codex the combined `AGENTS.md` must stay under 32 KiB (the default limit). Use `agentorg show agent <project> <name>` to check total line counts.

## Roadmap

- [x] Company / department / project scaffolding
- [x] Context merging with department chain inheritance
- [x] Claude Code formatter (CLAUDE.md + @import + skills)
- [x] Codex formatter (AGENTS.md single file)
- [x] Generic CLI formatter (context.md)
- [x] `sync` with SHA-256 change detection
- [ ] Agent task management (TODO.md, task assignment)
- [ ] Cron-based agent scheduling
- [ ] Discord / Slack bot integration for remote control
- [ ] GitHub webhook → auto-create review tasks

## License

MIT
