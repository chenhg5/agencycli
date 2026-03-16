// Package formatter translates a MergedContext into the file layout that a
// specific agent runtime expects inside its working directory.
//
// This is the ONLY package that contains agent-specific knowledge. All other
// packages are completely unaware of Claude Code, Codex, or any other agent.
package formatter

import (
	"fmt"

	"github.com/agentorg/agentorg/internal/ctxbuild"
	"github.com/agentorg/agentorg/internal/entity"
)

// Formatter writes context files into an agent working directory.
type Formatter interface {
	// Format writes all necessary files into outDir so that the target agent
	// can load the full context when started from that directory.
	Format(mc *ctxbuild.MergedContext, outDir string) error
}

// New returns the Formatter appropriate for the given agent model.
func New(model entity.AgentModel) (Formatter, error) {
	switch model {
	case entity.ModelClaudeCode:
		return &claudeCodeFormatter{}, nil
	case entity.ModelCodex:
		return &codexFormatter{}, nil
	case entity.ModelGenericCLI:
		return &genericFormatter{}, nil
	default:
		return nil, fmt.Errorf("formatter: unsupported model %q", model)
	}
}
