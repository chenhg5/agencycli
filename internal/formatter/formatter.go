// Package formatter translates a MergedContext into the file layout that a
// specific agent runtime expects inside its working directory.
//
// This is the ONLY package that contains agent-specific knowledge. All other
// packages are completely unaware of Claude Code, Codex, or any other agent.
package formatter

import (
	"fmt"

	"github.com/chenhg5/agencycli/internal/ctxbuild"
	"github.com/chenhg5/agencycli/internal/entity"
)

// Formatter writes context files into an agent working directory.
type Formatter interface {
	// Format writes all necessary files into outDir so that the target agent
	// can load the full context when started from that directory.
	Format(mc *ctxbuild.MergedContext, outDir string) error
}

// New returns the Formatter appropriate for the given agent model.
// The model is normalised (alias-resolved) before lookup.
func New(model entity.AgentModel) (Formatter, error) {
	model = entity.NormaliseModel(model)
	switch model {
	case entity.ModelClaudeCode:
		return &claudeCodeFormatter{}, nil
	case entity.ModelCodex, entity.ModelQoder:
		// Qoder uses the same AGENTS.md format as Codex.
		return &codexFormatter{}, nil
	case entity.ModelCursor:
		return &cursorFormatter{}, nil
	case entity.ModelGemini:
		return &geminiFormatter{}, nil
	case entity.ModelOpenCode:
		return &singleFileFormatter{filename: "OPENCODE.md"}, nil
	case entity.ModelIFlow:
		return &singleFileFormatter{filename: "IFLOW.md"}, nil
	case entity.ModelGenericCLI:
		return &genericFormatter{}, nil
	case entity.ModelHTTPAgent:
		// HTTP agents use context.md as the system prompt sent with every request.
		return &genericFormatter{}, nil
	default:
		return nil, fmt.Errorf("formatter: unsupported model %q", model)
	}
}
