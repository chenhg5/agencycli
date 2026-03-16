// Package ctxbuild builds a MergedContext by walking the department chain
// and collecting prompt layers and skills. It is the core business logic
// of agentorg and has no dependency on any agent-specific format.
package ctxbuild

// MergedContext is the agent-agnostic representation of a fully assembled
// context for one (project, department) pair. The formatter layer consumes
// this and translates it into whatever format a specific agent requires.
type MergedContext struct {
	// Layers holds the ordered prompt sections, from most general to most
	// specific. The order is: company → dept chain … → project.
	Layers []ContextLayer

	// Skills holds deduplicated skills collected from the department chain.
	Skills []SkillDef
}

// ContextLayer is one prompt section with a human-readable source label.
type ContextLayer struct {
	// Source identifies where this content came from, e.g.:
	//   "company"
	//   "department:engineering"
	//   "department:engineering/backend"
	//   "project:cc-connect"
	Source string

	// Content is the raw Markdown text of the prompt.md for this layer.
	Content string
}

// SkillDef is a resolved skill with its prompt content ready to embed.
type SkillDef struct {
	Name        string
	Description string
	// Prompt is the content of the skill's prompt.md.
	Prompt string
}
