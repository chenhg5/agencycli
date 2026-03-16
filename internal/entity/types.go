// Package entity defines the core domain types for agentorg.
// All types are plain data structures with no business logic.
// They are serialised to/from YAML by the store layer.
package entity

import "time"

// AgentModel identifies which AI agent runtime an employee uses.
type AgentModel string

const (
	ModelClaudeCode AgentModel = "claude-code"
	ModelCodex      AgentModel = "codex"
	ModelGenericCLI AgentModel = "generic-cli"
)

// KnownModels lists all supported agent models.
var KnownModels = []AgentModel{
	ModelClaudeCode,
	ModelCodex,
	ModelGenericCLI,
}

// IsValidModel reports whether m is a known AgentModel.
func IsValidModel(m AgentModel) bool {
	for _, k := range KnownModels {
		if k == m {
			return true
		}
	}
	return false
}

// Company is the top-level organisational unit.
// Stored at <root>/.aios/company.yaml.
type Company struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// Department represents a functional group inside the company.
// Departments can be nested; the Parent field holds the slash-separated
// path of the parent department (empty = direct child of company).
// Stored at <root>/departments/<path>/dept.yaml.
type Department struct {
	Name        string   `yaml:"name"`
	Parent      string   `yaml:"parent,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Goals       []string `yaml:"goals,omitempty"`
	// Skills lists skill names this department uses.
	// Skills are inherited by all sub-departments.
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
// Stored at <root>/projects/<project>/agents/<name>/.aios-agent.yaml.
// It is used by `agentorg sync` to detect which context layers have changed.
type AgentMeta struct {
	Name       string     `yaml:"name"`
	Project    string     `yaml:"project"`
	Department string     `yaml:"department"`
	Model      AgentModel `yaml:"model"`
	HiredAt    time.Time  `yaml:"hired_at"`

	// ContextHash maps each layer source key to the SHA-256 hex digest
	// of its prompt content at hire time.
	ContextHash map[string]string `yaml:"context_hash,omitempty"`
}
