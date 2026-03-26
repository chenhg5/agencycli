package formatter

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
)

// RemoveOutputsFromOtherModels deletes context files produced for agent runtimes
// other than keepModel. Call this before formatting with the new model so
// e.g. CLAUDE.md is not left next to AGENTS.md after switching claudecode → codex.
func RemoveOutputsFromOtherModels(agentDir string, keepModel entity.AgentModel) error {
	agentDir = filepath.Clean(agentDir)
	keepModel = entity.NormaliseModel(keepModel)

	if keepModel != entity.ModelClaudeCode {
		_ = os.Remove(filepath.Join(agentDir, "CLAUDE.md"))
		_ = os.RemoveAll(filepath.Join(agentDir, ".claude", "skills"))
	}

	if keepModel != entity.ModelCodex && keepModel != entity.ModelQoder {
		_ = os.Remove(filepath.Join(agentDir, "AGENTS.md"))
		_ = os.RemoveAll(filepath.Join(agentDir, ".agencycli-skills"))
	}

	if keepModel != entity.ModelGemini {
		_ = os.Remove(filepath.Join(agentDir, "GEMINI.md"))
		_ = os.RemoveAll(filepath.Join(agentDir, ".gemini", "skills"))
	}

	if keepModel != entity.ModelCursor {
		_ = os.Remove(filepath.Join(agentDir, ".cursorrules"))
		_ = os.Remove(filepath.Join(agentDir, ".cursor", "rules", "agencycli.mdc"))
	}

	if keepModel != entity.ModelOpenCode {
		_ = os.Remove(filepath.Join(agentDir, "OPENCODE.md"))
	}

	if keepModel != entity.ModelIFlow {
		_ = os.Remove(filepath.Join(agentDir, "IFLOW.md"))
	}

	if keepModel != entity.ModelGenericCLI && keepModel != entity.ModelHTTPAgent {
		_ = os.Remove(filepath.Join(agentDir, "context.md"))
	}

	// Layer markdown under .agencycli/context is used by claudecode and gemini.
	// Other runtimes ignore it but stale files confuse humans and http-agent fallbacks.
	if keepModel != entity.ModelClaudeCode && keepModel != entity.ModelGemini {
		ctxDir := filepath.Join(agentDir, ".agencycli", "context")
		entries, err := os.ReadDir(ctxDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				if strings.HasSuffix(name, ".md") {
					_ = os.Remove(filepath.Join(ctxDir, name))
				}
			}
		}
	}

	return nil
}
