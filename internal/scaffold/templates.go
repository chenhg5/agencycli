package scaffold

// agencyPromptTmpl is the default template for agency-prompt.md.
const agencyPromptTmpl = `# Agency: {{.Name}}

{{if .Description}}{{.Description}}

{{end}}## Values & Principles

- (Describe your agency's values and principles here)

## Global Rules

- All agents must follow these rules across every project
- (Add your global rules here)
`

// teamPromptTmpl is the default template for a team's prompt.md.
const teamPromptTmpl = `# Team: {{.Name}}

{{if .Description}}{{.Description}}

{{end}}## Responsibilities

- (Describe what this team is responsible for)

## Standards & Conventions

- (Add team-wide standards and conventions here)
`

// projectPromptTmpl is the default template for a project's prompt.md.
const projectPromptTmpl = `# Project: {{.Name}}

{{if .Description}}{{.Description}}

{{end}}## Goal

(Describe the project goal in detail)

## Tech Stack

- (List the main technologies used)

{{if .Repo}}## Repository

Code lives at: {{.Repo}}

{{end}}## Context

(Add any project-specific context, architecture notes, or conventions here)
`

// gitignoreTmpl is the default .gitignore for a new agency workspace.
const gitignoreTmpl = `# ── agencycli runtime state ──────────────────────────────────────────────────

# Human inbox (generated summary + runtime queue)
.agencycli/inbox.md
.agencycli/inbox.yaml
.agencycli/messages.yaml

# Agent runtime state (PID, session IDs, last wakeup times)
projects/*/agents/*/.agencycli/heartbeat.yaml

# Async message queues
projects/*/agents/*/.agencycli/messages.yaml

# Task history (runtime, can be large)
projects/*/agents/*/.agencycli/tasks_archive.yaml

# Execution logs
projects/*/agents/*/.agencycli/runs/
*.log

# Fired agents
projects/*/agents/.fired/

# Wakeup prompt temp files
projects/*/agents/*/.agencycli/.prompt-*.txt

# ── OS ───────────────────────────────────────────────────────────────────────
.DS_Store
Thumbs.db

# ── Editor ───────────────────────────────────────────────────────────────────
.idea/
.vscode/
*.swp
*.swo

# ── Environment / secrets ────────────────────────────────────────────────────
.env
.env.*
!.env.example
`

// skillMDTmpl is the default template for a skill's SKILL.md.
// The YAML frontmatter at the top carries the skill metadata; the Markdown
// body below is injected into every agent that has this skill bound.
const skillMDTmpl = `---
name: {{.Name}}
{{- if .Description}}
description: {{.Description}}
{{- end}}
---

# Skill: {{.Name}}

{{if .Description}}{{.Description}}

{{end}}## How to use this skill

(Describe how and when to use this skill, and what the agent should do)
`
