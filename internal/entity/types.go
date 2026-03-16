// Package entity defines the core domain types for agencycli.
// All types are plain data structures with no business logic.
// They are serialised to/from YAML by the store layer.
package entity

import "time"

// AgentModel identifies which AI agent runtime an employee uses.
// Names match the identifiers used by cc-connect for interoperability.
type AgentModel string

const (
	// ModelClaudeCode drives Anthropic Claude Code (claude CLI).
	// Context file: CLAUDE.md with @import layers + .claude/skills/.
	ModelClaudeCode AgentModel = "claudecode"

	// ModelCodex drives OpenAI Codex CLI.
	// Context file: AGENTS.md (single merged file).
	ModelCodex AgentModel = "codex"

	// ModelCursor drives Cursor Agent CLI.
	// Context files: .cursorrules + .cursor/rules/agencycli.mdc
	ModelCursor AgentModel = "cursor"

	// ModelGemini drives Google Gemini CLI.
	// Context file: GEMINI.md + .gemini/skills/.
	ModelGemini AgentModel = "gemini"

	// ModelQoder drives Qoder CLI (qodercli).
	// Context file: AGENTS.md (same format as Codex).
	ModelQoder AgentModel = "qoder"

	// ModelOpenCode drives OpenCode CLI.
	// Context file: OPENCODE.md.
	ModelOpenCode AgentModel = "opencode"

	// ModelIFlow drives iFlow CLI.
	// Context file: IFLOW.md.
	ModelIFlow AgentModel = "iflow"

	// ModelGenericCLI is a fallback for any other CLI agent.
	// Context file: context.md (plain merged text).
	ModelGenericCLI AgentModel = "generic-cli"
)

// KnownModels lists all supported agent models in display order.
var KnownModels = []AgentModel{
	ModelClaudeCode,
	ModelCodex,
	ModelCursor,
	ModelGemini,
	ModelQoder,
	ModelOpenCode,
	ModelIFlow,
	ModelGenericCLI,
}

// modelAliases maps legacy or alternate spellings to the canonical model name.
var modelAliases = map[AgentModel]AgentModel{
	"claude-code": ModelClaudeCode, // kebab-case alias kept for backward compat
}

// NormaliseModel returns the canonical AgentModel for m, resolving any alias.
func NormaliseModel(m AgentModel) AgentModel {
	if canonical, ok := modelAliases[m]; ok {
		return canonical
	}
	return m
}

// IsValidModel reports whether m (after alias resolution) is a known AgentModel.
func IsValidModel(m AgentModel) bool {
	m = NormaliseModel(m)
	for _, k := range KnownModels {
		if k == m {
			return true
		}
	}
	return false
}

// Agency is the top-level organisational unit (the "company").
// Stored at <root>/.agencycli/agency.yaml.
type Agency struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// Team represents a functional group inside the agency.
// Teams can be nested; the Parent field holds the slash-separated
// path of the parent team (empty = direct child of agency).
// Stored at <root>/teams/<path>/team.yaml.
type Team struct {
	Name        string   `yaml:"name"`
	Parent      string   `yaml:"parent,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Goals       []string `yaml:"goals,omitempty"`
	// Skills lists skill names this team uses.
	// Skills are inherited by all sub-teams.
	Skills []string `yaml:"skills,omitempty"`
}

// Project is a concrete product or initiative.
// Stored at <root>/projects/<name>/project.yaml.
type Project struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	// Repo is the path (relative or absolute) to the actual code repository.
	Repo string `yaml:"repo,omitempty"`
}

// Skill is a reusable capability definition.
// The executable behaviour is described in prompt.md.
// Stored at <root>/skills/<name>/skill.yaml.
type Skill struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// AgentMeta records the provenance of a hired agent working directory.
// Stored at <root>/projects/<project>/agents/<name>/.agencycli-agent.yaml.
// It is used by `agencycli sync` to detect which context layers have changed.
type AgentMeta struct {
	Name    string     `yaml:"name"`
	Project string     `yaml:"project"`
	Team    string     `yaml:"team"`
	Model   AgentModel `yaml:"model"`
	HiredAt time.Time  `yaml:"hired_at"`

	// ContextHash maps each layer source key to the SHA-256 hex digest
	// of its prompt content at hire time.
	ContextHash map[string]string `yaml:"context_hash,omitempty"`
}
