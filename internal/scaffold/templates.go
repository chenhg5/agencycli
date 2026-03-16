package scaffold

// companyPromptTmpl is the default template for company-prompt.md.
const companyPromptTmpl = `# Company: {{.Name}}

{{if .Description}}{{.Description}}

{{end}}## Values & Principles

- (Describe your company's values and principles here)

## Global Rules

- All agents must follow these rules across every project
- (Add your global rules here)
`

// deptPromptTmpl is the default template for a department's prompt.md.
const deptPromptTmpl = `# Department: {{.Name}}

{{if .Description}}{{.Description}}

{{end}}## Responsibilities

- (Describe what this department is responsible for)

## Standards & Conventions

- (Add department-wide standards and conventions here)
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
// (Currently unused at runtime — each builtin skill has its own content.)
const skillPromptTmpl = `# Skill: {{.Name}}

{{if .Description}}{{.Description}}

{{end}}## How to use

(Describe how and when to use this skill)
`

// builtinSkills defines the skills installed when a company workspace is
// initialised. Prompt content uses only backtick-free Markdown so that the
// Go raw string literals are not prematurely terminated.
var builtinSkills = []struct {
	Name        string
	Description string
	Prompt      string
}{
	{
		Name:        "git",
		Description: "Git version control operations. Use for commits, branches, and history.",
		Prompt: `# Git Operations

Use git for all version control tasks.

## Conventions

- Always run 'git status' before committing to understand what changed
- Write clear commit messages using Conventional Commits format:
  type(scope): short description
  Types: feat, fix, docs, style, refactor, test, chore
- Keep commits small and focused on a single change
- Create a feature branch for every non-trivial change
- Never force-push to main or master

## Common workflows

Check current state:
  git status
  git diff

Stage and commit:
  git add -p
  git commit -m "feat(auth): add JWT validation"

Create and push a feature branch:
  git checkout -b feat/my-feature
  git push -u origin feat/my-feature
`,
	},
	{
		Name:        "github",
		Description: "GitHub operations via the gh CLI. Use for PRs, issues, and releases.",
		Prompt: `# GitHub Operations

Use the GitHub CLI (gh) for all GitHub interactions.

## Pull Requests

Create a PR:
  gh pr create --title "feat: add feature" --body "Description"

List open PRs:
  gh pr list

Review a PR:
  gh pr view 123
  gh pr diff 123

## Issues

List issues:
  gh issue list

View an issue:
  gh issue view 42

Create an issue:
  gh issue create --title "Bug: ..." --body "Steps to reproduce..."

## Releases

Create a release:
  gh release create v1.0.0 --title "v1.0.0" --notes "Release notes"
`,
	},
	{
		Name:        "bash",
		Description: "Shell scripting and command execution best practices.",
		Prompt: `# Shell / Bash

## Best practices

- Always check exit codes for critical commands
- Use 'set -euo pipefail' at the top of scripts
- Quote variables: "$var" not $var
- Use mktemp for temporary files and clean them up
- Prefer explicit paths in scripts over relying on PATH

## Useful patterns

Check a command exists:
  command -v jq >/dev/null 2>&1 || { echo "jq not found"; exit 1; }

Read a file line by line:
  while IFS= read -r line; do
    echo "$line"
  done < file.txt
`,
	},
}
