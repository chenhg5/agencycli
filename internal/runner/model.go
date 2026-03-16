package runner

import (
	"strings"

	"github.com/agencycli/agencycli/internal/entity"
)

// ModelInvoker knows how to invoke a specific agent model CLI.
type ModelInvoker interface {
	// Args returns the command + arguments to invoke the agent.
	// promptFile is a path to a temp file containing the full prompt text.
	// sessionID is the previous session ID (empty = start fresh).
	Args(promptFile, sessionID string) []string

	// ParseSessionID attempts to extract a new session ID from the agent's
	// combined stdout output. Returns "" if not found or not supported.
	ParseSessionID(output string) string
}

// InvokerFor returns the ModelInvoker for the given model.
// If the model has a custom runCommand (from AgentMeta), it takes precedence.
func InvokerFor(model entity.AgentModel, runCommand string) ModelInvoker {
	if runCommand != "" {
		return &customInvoker{tmpl: runCommand}
	}
	switch entity.NormaliseModel(model) {
	case entity.ModelClaudeCode:
		return &claudeInvoker{}
	case entity.ModelCodex:
		return &codexInvoker{}
	case entity.ModelGemini:
		return &geminiInvoker{}
	case entity.ModelOpenCode:
		return &openCodeInvoker{}
	default:
		return &genericInvoker{}
	}
}

// ── Claude Code ───────────────────────────────────────────────────────────────

type claudeInvoker struct{}

func (c *claudeInvoker) Args(promptFile, sessionID string) []string {
	// claude --no-interactive --output-format stream-json -p "$(cat file)"
	// We read from file via shell so the prompt isn't exposed in ps output.
	args := []string{"claude", "--no-interactive", "--output-format", "stream-json"}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	}
	args = append(args, "--print-file", promptFile)
	return args
}

func (c *claudeInvoker) ParseSessionID(output string) string {
	// Claude emits stream-json lines. First system line contains session_id.
	// Example: {"type":"system","subtype":"init","session_id":"abc123",...}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"session_id"`) && strings.Contains(line, `"type":"system"`) {
			// Simple extraction without full JSON parse to avoid import.
			const key = `"session_id":"`
			idx := strings.Index(line, key)
			if idx < 0 {
				continue
			}
			rest := line[idx+len(key):]
			end := strings.Index(rest, `"`)
			if end > 0 {
				return rest[:end]
			}
		}
	}
	return ""
}

// ── Codex ─────────────────────────────────────────────────────────────────────

type codexInvoker struct{}

func (c *codexInvoker) Args(promptFile, sessionID string) []string {
	// codex -q --full-auto -p "<prompt>"
	// Codex does not support --print-file; we inject via stdin or -p flag.
	return []string{"codex", "-q", "--full-auto", "--input-file", promptFile}
}

func (c *codexInvoker) ParseSessionID(_ string) string { return "" }

// ── Gemini ────────────────────────────────────────────────────────────────────

type geminiInvoker struct{}

func (g *geminiInvoker) Args(promptFile, _ string) []string {
	return []string{"gemini", "--yolo", "--prompt-file", promptFile}
}

func (g *geminiInvoker) ParseSessionID(_ string) string { return "" }

// ── OpenCode ──────────────────────────────────────────────────────────────────

type openCodeInvoker struct{}

func (o *openCodeInvoker) Args(promptFile, _ string) []string {
	return []string{"opencode", "run", "--file", promptFile}
}

func (o *openCodeInvoker) ParseSessionID(_ string) string { return "" }

// ── Generic ───────────────────────────────────────────────────────────────────

// genericInvoker falls back to a minimal echo invocation.
// Users should set run_command in .agencycli-agent.yaml for custom agents.
type genericInvoker struct{}

func (g *genericInvoker) Args(promptFile, _ string) []string {
	return []string{"sh", "-c", "cat " + promptFile}
}

func (g *genericInvoker) ParseSessionID(_ string) string { return "" }

// ── Custom template invoker ───────────────────────────────────────────────────

// customInvoker uses a shell template from the agent's run_command field.
// Supported placeholders: {prompt_file}, {session_id}
type customInvoker struct {
	tmpl string
}

func (c *customInvoker) Args(promptFile, sessionID string) []string {
	cmd := strings.ReplaceAll(c.tmpl, "{prompt_file}", promptFile)
	cmd = strings.ReplaceAll(cmd, "{session_id}", sessionID)
	return []string{"sh", "-c", cmd}
}

func (c *customInvoker) ParseSessionID(_ string) string { return "" }
