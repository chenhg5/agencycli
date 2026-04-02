package runner

import (
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
)

// ModelInvoker knows how to invoke a specific agent model CLI.
type ModelInvoker interface {
	// Args returns the command + arguments to invoke the agent.
	// promptFile is a path to a temp file containing the full prompt text.
	// sessionID is the previous session ID (empty = start fresh).
	Args(promptFile, sessionID string) []string

	// UseStdinPrompt reports whether the runner should open promptFile and
	// pipe its contents to the process via stdin instead of referencing the
	// file path in the argument list.
	// When true, the promptFile path is NOT included in Args().
	UseStdinPrompt() bool

	// ParseSessionID attempts to extract a new session ID from the agent's
	// combined stdout output. Returns "" if not found or not supported.
	ParseSessionID(output string) string
}

// InvokerFor returns the ModelInvoker for the given model.
// If the model has a custom runCommand (from AgentMeta), it takes precedence.
// addDirs lists additional directories to expose to the agent (model-specific flags).
func InvokerFor(model entity.AgentModel, runCommand string, addDirs []string) ModelInvoker {
	if runCommand != "" {
		return &customInvoker{tmpl: runCommand}
	}
	switch entity.NormaliseModel(model) {
	case entity.ModelClaudeCode:
		return &claudeInvoker{addDirs: addDirs}
	case entity.ModelCodex:
		return &codexInvoker{addDirs: addDirs}
	case entity.ModelGemini:
		return &geminiInvoker{addDirs: addDirs}
	case entity.ModelOpenCode:
		return &openCodeInvoker{}
	case entity.ModelCursor:
		return &cursorInvoker{}
	default:
		return &genericInvoker{}
	}
}

// ── Claude Code ───────────────────────────────────────────────────────────────

type claudeInvoker struct {
	addDirs []string
}

func (c *claudeInvoker) Args(promptFile, sessionID string) []string {
	// Use -p/--print for non-interactive mode; prompt arrives on stdin
	// (UseStdinPrompt returns true so the runner pipes promptFile).
	//
	// --dangerously-skip-permissions is required when running as root inside a
	// Docker sandbox. IS_SANDBOX=1 must also be set (handled by sandbox layer).
	args := []string{
		"claude",
		"-p",
		"--verbose",
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
	}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	}
	for _, dir := range c.addDirs {
		args = append(args, "--add-dir", dir)
	}
	return args
}

func (c *claudeInvoker) UseStdinPrompt() bool { return true }

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

type codexInvoker struct {
	addDirs []string
}

func (c *codexInvoker) Args(promptFile, sessionID string) []string {
	// `codex exec -` reads the prompt from stdin.
	// --skip-git-repo-check allows running outside a git repo (agent workspace
	// dirs are not git repos themselves; the project repo is mounted separately).
	// When sessionID is provided, use `codex exec resume` to continue a session.
	var args []string
	if sessionID != "" {
		args = []string{"codex", "exec", "resume", sessionID}
	} else {
		args = []string{"codex", "exec", "--skip-git-repo-check"}
	}
	for _, dir := range c.addDirs {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "-")
	return args
}

func (c *codexInvoker) UseStdinPrompt() bool { return true }

func (c *codexInvoker) ParseSessionID(_ string) string { return "" }

// ── Gemini ────────────────────────────────────────────────────────────────────

type geminiInvoker struct {
	addDirs []string
}

func (g *geminiInvoker) Args(promptFile, sessionID string) []string {
	// -p requires a string value to activate non-interactive/headless mode.
	// Passing "" means headless mode is active; the real prompt arrives via
	// stdin. Gemini appends the -p value to stdin input, so "" = stdin only.
	// --output-format stream-json gives structured output for session parsing.
	args := []string{"gemini", "-y", "--output-format", "stream-json", "-p", ""}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	}
	for _, dir := range g.addDirs {
		args = append(args, "--include-directories", dir)
	}
	return args
}

func (g *geminiInvoker) UseStdinPrompt() bool { return true }

func (g *geminiInvoker) ParseSessionID(_ string) string { return "" }

// ── OpenCode ──────────────────────────────────────────────────────────────────

type openCodeInvoker struct{}

func (o *openCodeInvoker) Args(promptFile, _ string) []string {
	// opencode run reads prompt from stdin when no positional arg is given.
	return []string{"opencode", "run"}
}

func (o *openCodeInvoker) UseStdinPrompt() bool { return true }

func (o *openCodeInvoker) ParseSessionID(_ string) string { return "" }

// ── Cursor ────────────────────────────────────────────────────────────────────
// Cursor CLI is the `agent` binary installed via `curl https://cursor.com/install | bash`.
// Auth: credentials cached in ~/.cursor/ after `agent login`.

type cursorInvoker struct{}

func (c *cursorInvoker) Args(promptFile, sessionID string) []string {
	args := []string{
		"agent",
		"--print",
		"--output-format", "stream-json",
		"--force",
		"--trust",
	}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	}
	return args
}

func (c *cursorInvoker) UseStdinPrompt() bool { return true }

func (c *cursorInvoker) ParseSessionID(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"session_id"`) {
			const key = `"session_id":"`
			idx := strings.Index(line, key)
			if idx < 0 {
				continue
			}
			rest := line[idx+len(key):]
			if end := strings.Index(rest, `"`); end > 0 {
				return rest[:end]
			}
		}
	}
	return ""
}

// ── Generic ───────────────────────────────────────────────────────────────────

// genericInvoker falls back to `cat` so the prompt text is printed to stdout.
// Users should set run_command in .agencycli-agent.yaml for custom agents.
type genericInvoker struct{}

func (g *genericInvoker) Args(promptFile, _ string) []string {
	return []string{"cat", promptFile}
}

func (g *genericInvoker) UseStdinPrompt() bool { return false }

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

func (c *customInvoker) UseStdinPrompt() bool { return false }

func (c *customInvoker) ParseSessionID(_ string) string { return "" }
