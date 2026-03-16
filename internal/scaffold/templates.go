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

// skillPromptTmpl is the default template for a skill's prompt.md.
const skillPromptTmpl = `# Skill: {{.Name}}

{{if .Description}}{{.Description}}

{{end}}## How to use

(Describe how and when to use this skill)
`
