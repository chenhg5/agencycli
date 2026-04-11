package api

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// assistantSession tracks an in-flight interactive claude process so the
// permission endpoint can write allow/deny responses to its stdin.
type assistantSession struct {
	stdin     io.WriteCloser
	stdinMu   sync.Mutex
	autoAllow atomic.Bool
	done      chan struct{}
}

func (sess *assistantSession) writePermission(requestID, behavior string, input map[string]any, message string) error {
	var permResp map[string]any
	if behavior == "allow" {
		updated := input
		if updated == nil {
			updated = make(map[string]any)
		}
		permResp = map[string]any{"behavior": "allow", "updatedInput": updated}
	} else {
		if message == "" {
			message = "User denied this action."
		}
		permResp = map[string]any{"behavior": "deny", "message": message}
	}
	data, err := json.Marshal(map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": requestID,
			"response":   permResp,
		},
	})
	if err != nil {
		return err
	}
	sess.stdinMu.Lock()
	defer sess.stdinMu.Unlock()
	_, err = sess.stdin.Write(append(data, '\n'))
	return err
}

func generateAssistantSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "ast_" + hex.EncodeToString(b)
}

type assistantChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type assistantChatBody struct {
	Message string             `json:"message"`
	History []assistantChatMsg `json:"history"`
}

func (s *Server) handleAssistantChat(w http.ResponseWriter, r *http.Request) {
	var body assistantChatBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	msg := strings.TrimSpace(body.Message)
	if msg == "" {
		s.jsonError(w, http.StatusBadRequest, "message is required")
		return
	}

	skill := s.loadAssistantSkill()
	prompt := s.buildAssistantPromptWithGoals(skill, s.root, body.History, msg)

	wantStream := strings.Contains(r.Header.Get("Accept"), "text/event-stream")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	if wantStream {
		s.assistantStream(w, ctx, prompt)
	} else {
		s.assistantJSON(w, ctx, prompt)
	}
}

func (s *Server) assistantJSON(w http.ResponseWriter, ctx context.Context, prompt string) {
	cliPath, cliArgs := s.resolveAssistantCLI()
	if cliPath == "" {
		s.jsonError(w, http.StatusInternalServerError, "no supported AI CLI found (tried: claude, codex, gemini)")
		return
	}
	// For JSON mode, use --print for plain text output.
	args := make([]string, 0, len(cliArgs))
	for _, a := range cliArgs {
		if a == "--output-format" || a == "stream-json" {
			continue
		}
		args = append(args, a)
	}
	if cliPath != "" {
		base := filepath.Base(cliPath)
		if base == "claude" || base == "gemini" {
			hasP := false
			for _, a := range args {
				if a == "--print" {
					hasP = true
					break
				}
			}
			if !hasP {
				args = append([]string{"--print"}, args...)
			}
		}
	}
	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Dir = s.root
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("assistant error: %v", err))
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"response": strings.TrimSpace(string(out)),
	})
}

func (s *Server) assistantStream(w http.ResponseWriter, ctx context.Context, prompt string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.jsonError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	cliPath, cliArgs := s.resolveAssistantCLI()
	if cliPath == "" {
		s.jsonError(w, http.StatusInternalServerError, "no supported AI CLI found (tried: claude, codex, gemini)")
		return
	}

	if filepath.Base(cliPath) == "claude" {
		s.assistantStreamClaude(w, ctx, prompt, cliPath, flusher)
		return
	}

	// Fallback: one-shot streaming for other CLIs (codex, gemini).
	cmd := exec.CommandContext(ctx, cliPath, cliArgs...)
	cmd.Dir = s.root
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("pipe: %v", err))
		return
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("start: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	lineCount := 0
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		lineCount++
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}

	exitErr := cmd.Wait()

	if lineCount == 0 {
		errMsg := strings.TrimSpace(stderrBuf.String())
		if errMsg == "" && exitErr != nil {
			errMsg = exitErr.Error()
		}
		if errMsg == "" {
			errMsg = "CLI produced no output"
		}
		errJSON := fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Error: %s\n\nCommand: %s %s"}]}}`,
			strings.ReplaceAll(errMsg, `"`, `\"`),
			filepath.Base(cliPath),
			strings.Join(cliArgs, " "))
		fmt.Fprintf(w, "data: %s\n\n", errJSON)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: {\"type\":\"done\"}\n\n")
	flusher.Flush()
}

// assistantStreamClaude runs an interactive claude session with bidirectional
// stream-json I/O and the permission-prompt-tool protocol. Permission requests
// are forwarded as SSE events; handleAssistantPermission writes responses back.
func (s *Server) assistantStreamClaude(w http.ResponseWriter, ctx context.Context, prompt, cliPath string, flusher http.Flusher) {
	args := []string{
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--permission-prompt-tool", "stdio",
		"--verbose",
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Dir = s.root
	cmd.Env = os.Environ()

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("stdin pipe: %v", err))
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("stdout pipe: %v", err))
		return
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("start: %v", err))
		return
	}

	sessID := generateAssistantSessionID()
	sess := &assistantSession{stdin: stdinPipe, done: make(chan struct{})}

	s.assistantMu.Lock()
	if s.assistantSessions == nil {
		s.assistantSessions = make(map[string]*assistantSession)
	}
	s.assistantSessions[sessID] = sess
	s.assistantMu.Unlock()

	defer func() {
		s.assistantMu.Lock()
		delete(s.assistantSessions, sessID)
		s.assistantMu.Unlock()
		close(sess.done)
		stdinPipe.Close()
	}()

	// Send user prompt via stream-json stdin.
	promptMsg, _ := json.Marshal(map[string]any{
		"type":    "user",
		"message": map[string]any{"role": "user", "content": prompt},
	})
	sess.stdinMu.Lock()
	_, _ = stdinPipe.Write(append(promptMsg, '\n'))
	sess.stdinMu.Unlock()

	// SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// First event: session ID so the frontend can send permission responses.
	fmt.Fprintf(w, "data: {\"type\":\"session\",\"session_id\":\"%s\"}\n\n", sessID)
	flusher.Flush()

	lineCount := 0
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		lineCount++

		// Intercept control_request / control_cancel_request for permission protocol.
		if strings.Contains(line, `"control_request"`) || strings.Contains(line, `"control_cancel_request"`) {
			var raw map[string]any
			if err := json.Unmarshal([]byte(line), &raw); err == nil {
				rawType, _ := raw["type"].(string)

				if rawType == "control_request" {
					request, _ := raw["request"].(map[string]any)
					requestID, _ := request["request_id"].(string)
					toolName, _ := request["tool_name"].(string)
					input, _ := request["input"].(map[string]any)

					if sess.autoAllow.Load() {
						_ = sess.writePermission(requestID, "allow", input, "")
						continue
					}

					permEvt, _ := json.Marshal(map[string]any{
						"type":       "permission_request",
						"session_id": sessID,
						"request_id": requestID,
						"tool_name":  toolName,
						"input":      input,
					})
					fmt.Fprintf(w, "data: %s\n\n", permEvt)
					flusher.Flush()
					continue
				}

				if rawType == "control_cancel_request" {
					cancelEvt, _ := json.Marshal(map[string]any{
						"type":       "permission_cancel",
						"request_id": raw["request_id"],
					})
					fmt.Fprintf(w, "data: %s\n\n", cancelEvt)
					flusher.Flush()
					continue
				}
			}
		}

		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}

	exitErr := cmd.Wait()

	if lineCount == 0 {
		errMsg := strings.TrimSpace(stderrBuf.String())
		if errMsg == "" && exitErr != nil {
			errMsg = exitErr.Error()
		}
		if errMsg == "" {
			errMsg = "CLI produced no output"
		}
		errJSON := fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Error: %s\n\nCommand: claude %s"}]}}`,
			strings.ReplaceAll(errMsg, `"`, `\"`),
			strings.Join(args, " "))
		fmt.Fprintf(w, "data: %s\n\n", errJSON)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: {\"type\":\"done\"}\n\n")
	flusher.Flush()
}

// handleAssistantPermission receives allow/deny/allowAll from the frontend
// and writes a control_response to the claude process stdin.
func (s *Server) handleAssistantPermission(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string         `json:"session_id"`
		RequestID string         `json:"request_id"`
		Behavior  string         `json:"behavior"`
		Input     map[string]any `json:"input,omitempty"`
	}
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.SessionID == "" || body.RequestID == "" || body.Behavior == "" {
		s.jsonError(w, http.StatusBadRequest, "session_id, request_id, and behavior are required")
		return
	}

	s.assistantMu.Lock()
	sess := s.assistantSessions[body.SessionID]
	s.assistantMu.Unlock()

	if sess == nil {
		s.jsonError(w, http.StatusNotFound, "assistant session not found or already ended")
		return
	}

	behavior := body.Behavior
	if behavior == "allowAll" {
		sess.autoAllow.Store(true)
		behavior = "allow"
	}

	if err := sess.writePermission(body.RequestID, behavior, body.Input, ""); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("write permission: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resolveAssistantCLI() (string, []string) {
	if path, err := exec.LookPath("claude"); err == nil {
		return path, []string{
			"-p", "-",
			"--output-format", "stream-json", "--verbose",
			"--allowedTools", "Bash(command:*)", "Read", "Write", "Edit",
			"Glob", "Grep", "WebSearch", "WebFetch",
		}
	}
	if path, err := exec.LookPath("codex"); err == nil {
		return path, []string{"exec", "-q", "-"}
	}
	if path, err := exec.LookPath("gemini"); err == nil {
		return path, []string{"-p", "-", "--output-format", "stream-json"}
	}
	return "", nil
}

func (s *Server) loadAssistantSkill() string {
	candidates := []string{
		filepath.Join(s.root, "SKILL.md"),
		filepath.Join(s.root, "skills", "agencycli", "SKILL.md"),
		filepath.Join(os.Getenv("HOME"), ".claude", "skills", "agencycli", "SKILL.md"),
		filepath.Join(os.Getenv("HOME"), ".cursor", "skills-cursor", "agencycli", "SKILL.md"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil && len(data) > 0 {
			return stripFrontmatter(string(data))
		}
	}
	return defaultAssistantSkill
}

const defaultAssistantSkill = `# agencycli Assistant

You are an AI assistant for the agencycli platform. You can help users manage their AI agent teams by running agencycli CLI commands.

Key commands:
- agencycli create agency/team/project/role - Create resources
- agencycli hire - Hire an agent into a project
- agencycli task add - Add a task for an agent
- agencycli inbox messages - View messages
- agencycli scheduler start - Start the scheduler
- agencycli sync - Sync agent context
- agencycli okr list/create/show/update/delete - Manage OKRs
- agencycli okr kr add/update - Manage Key Results
- agencycli milestone list/create/show/update/delete - Manage milestones

Always use --dir flag pointing to the agency workspace root when running commands.
Run 'agencycli --help' for full command reference.
`

func (s *Server) buildAssistantPromptWithGoals(skill, root string, history []assistantChatMsg, message string) string {
	goalSummary := s.buildGoalSummary("")
	return buildAssistantPrompt(skill, root, goalSummary, history, message)
}

func buildAssistantPrompt(skill, root, goalSummary string, history []assistantChatMsg, message string) string {
	var sb strings.Builder

	sb.WriteString(skill)
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(fmt.Sprintf("## Environment\n\nAgency workspace: `%s`\nAlways use `--dir %s` when running agencycli commands.\n\n", root, root))

	if goalSummary != "" {
		sb.WriteString("## Current Goals & OKRs\n\n")
		sb.WriteString(goalSummary)
		sb.WriteString("\n---\n\n")
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("You are an assistant integrated into the agencycli web console. ")
	sb.WriteString("The user will ask you to perform tasks related to managing their AI agent agency. ")
	sb.WriteString("You should execute agencycli CLI commands to fulfill their requests. ")
	sb.WriteString("You are aware of the agency's OKRs and milestones — reference them when relevant. ")
	sb.WriteString("When the user asks about goals, progress, or what to focus on, consult the Current Goals section. ")
	sb.WriteString("Always explain what you're doing and show the results. ")
	sb.WriteString("Respond concisely in the same language as the user's message.\n\n")

	if len(history) > 0 {
		sb.WriteString("## Conversation History\n\n")
		for _, h := range history {
			if h.Role == "user" {
				sb.WriteString(fmt.Sprintf("**User**: %s\n\n", h.Content))
			} else {
				sb.WriteString(fmt.Sprintf("**Assistant**: %s\n\n", h.Content))
			}
		}
		sb.WriteString("---\n\n")
	}

	sb.WriteString(fmt.Sprintf("## Current Request\n\n%s\n", message))

	return sb.String()
}

func stripFrontmatter(s string) string {
	if !strings.HasPrefix(s, "---") {
		return s
	}
	rest := s[3:]
	idx := strings.Index(rest, "---")
	if idx < 0 {
		return s
	}
	return strings.TrimLeft(rest[idx+3:], "\r\n")
}
