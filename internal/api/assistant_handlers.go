package api

import (
	"context"
	"encoding/json"
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

	skill := s.loadAssistantSkill()
	prompt := buildAssistantPrompt(skill, s.root, body.History, msg)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	output, err := runAssistantCLI(ctx, s.root, prompt)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("assistant error: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"response": strings.TrimSpace(output),
	})
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
			return string(data)
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

func runAssistantCLI(ctx context.Context, workDir, prompt string) (string, error) {
	// Try claude CLI first (most common for agencycli users).
	if path, err := exec.LookPath("claude"); err == nil {
		cmd := exec.CommandContext(ctx, path, "--print", "-p", prompt)
		cmd.Dir = workDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err == nil {
			return string(out), nil
		}
		if len(out) > 0 {
			return string(out), nil
		}
		return "", fmt.Errorf("claude: %w\n%s", err, string(out))
	}

	// Try codex CLI.
	if path, err := exec.LookPath("codex"); err == nil {
		cmd := exec.CommandContext(ctx, path, "exec", "-q", prompt)
		cmd.Dir = workDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err == nil {
			return string(out), nil
		}
		if len(out) > 0 {
			return string(out), nil
		}
		return "", fmt.Errorf("codex: %w", err)
	}

	// Try gemini CLI.
	if path, err := exec.LookPath("gemini"); err == nil {
		cmd := exec.CommandContext(ctx, path, "--print", "-p", prompt)
		cmd.Dir = workDir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err == nil {
			return string(out), nil
		}
		if len(out) > 0 {
			return string(out), nil
		}
		return "", fmt.Errorf("gemini: %w", err)
	}

	return "", fmt.Errorf("no supported AI CLI found (tried: claude, codex, gemini). Install one to use the assistant")
}
