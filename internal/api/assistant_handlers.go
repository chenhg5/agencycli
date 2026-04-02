package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.jsonError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	skill := s.loadAssistantSkill()
	prompt := buildAssistantPrompt(skill, s.root, body.History, msg)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	cliPath, cliArgs := s.resolveAssistantCLI()
	if cliPath == "" {
		s.jsonError(w, http.StatusInternalServerError, "no supported AI CLI found (tried: claude, codex, gemini)")
		return
	}

	cmd := exec.CommandContext(ctx, cliPath, cliArgs...)
	cmd.Dir = s.root
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("pipe: %v", err))
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("start: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}

	_ = cmd.Wait()
	fmt.Fprintf(w, "data: {\"type\":\"done\"}\n\n")
	flusher.Flush()
}

func (s *Server) resolveAssistantCLI() (string, []string) {
	if path, err := exec.LookPath("claude"); err == nil {
		return path, []string{"-p", "-", "--output-format", "stream-json"}
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

Always use --dir flag pointing to the agency workspace root when running commands.
Run 'agencycli --help' for full command reference.
`

func buildAssistantPrompt(skill, root string, history []assistantChatMsg, message string) string {
	var sb strings.Builder

	sb.WriteString(skill)
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(fmt.Sprintf("## Environment\n\nAgency workspace: `%s`\nAlways use `--dir %s` when running agencycli commands.\n\n", root, root))
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("You are an assistant integrated into the agencycli web console. ")
	sb.WriteString("The user will ask you to perform tasks related to managing their AI agent agency. ")
	sb.WriteString("You should execute agencycli CLI commands to fulfill their requests. ")
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
