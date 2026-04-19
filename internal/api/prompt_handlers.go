package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
)

type promptResponse struct {
	Content string `json:"content"`
}

type promptSaveBody struct {
	Content string `json:"content"`
}

// ── Agency prompt ─────────────────────────────────────────────────────────────

func (s *Server) handleGetAgencyPrompt(w http.ResponseWriter, _ *http.Request) {
	content, err := s.st.AgencyPrompt()
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(promptResponse{Content: content})
}

func (s *Server) handlePutAgencyPrompt(w http.ResponseWriter, r *http.Request) {
	var body promptSaveBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.st.SaveAgencyPrompt(body.Content); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ── Team prompt ───────────────────────────────────────────────────────────────

func (s *Server) handleGetTeamPrompt(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.PathValue("teamPath"), "/")
	if path == "" {
		s.jsonError(w, http.StatusBadRequest, "missing team path")
		return
	}
	content, err := s.st.TeamPrompt(path)
	if err != nil {
		if isNotFoundErr(err) {
			_ = json.NewEncoder(w).Encode(promptResponse{Content: ""})
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(promptResponse{Content: content})
}

func (s *Server) handlePutTeamPrompt(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.PathValue("teamPath"), "/")
	if path == "" {
		s.jsonError(w, http.StatusBadRequest, "missing team path")
		return
	}
	var body promptSaveBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.st.SaveTeamPrompt(path, body.Content); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ── Role prompt ───────────────────────────────────────────────────────────────

func (s *Server) handleGetRolePrompt(w http.ResponseWriter, r *http.Request) {
	teamPath := strings.TrimSpace(r.URL.Query().Get("team"))
	roleName := strings.TrimSpace(r.URL.Query().Get("role"))
	if teamPath == "" || roleName == "" {
		s.jsonError(w, http.StatusBadRequest, "team and role query params are required")
		return
	}
	content, err := s.st.RolePrompt(teamPath, roleName)
	if err != nil {
		if isNotFoundErr(err) {
			_ = json.NewEncoder(w).Encode(promptResponse{Content: ""})
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(promptResponse{Content: content})
}

func (s *Server) handlePutRolePrompt(w http.ResponseWriter, r *http.Request) {
	teamPath := strings.TrimSpace(r.URL.Query().Get("team"))
	roleName := strings.TrimSpace(r.URL.Query().Get("role"))
	if teamPath == "" || roleName == "" {
		s.jsonError(w, http.StatusBadRequest, "team and role query params are required")
		return
	}
	var body promptSaveBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.st.SaveRolePrompt(teamPath, roleName, body.Content); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ── Project prompt ────────────────────────────────────────────────────────────

func (s *Server) handleGetProjectPrompt(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	content, err := s.st.ProjectPrompt(name)
	if err != nil {
		if isNotFoundErr(err) {
			_ = json.NewEncoder(w).Encode(promptResponse{Content: ""})
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(promptResponse{Content: content})
}

func (s *Server) handlePutProjectPrompt(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.st.Project(name); err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "project not found")
			return
		}
		s.serverError(w, err)
		return
	}
	var body promptSaveBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.st.SaveProjectPrompt(name, body.Content); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ── Agent context (merged, read-only) + wakeup ───────────────────────────────

func (s *Server) handleGetAgentContext(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	agent := r.PathValue("agent")
	agentDir := s.st.AgentDir(project, agent)

	meta, err := s.st.AgentMeta(project, agent)
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "agent not found")
			return
		}
		s.serverError(w, err)
		return
	}

	contextFile := contextFileName(string(meta.Model))
	mergedPath := filepath.Join(agentDir, contextFile)
	merged, _ := os.ReadFile(mergedPath)

	wakeupPath := filepath.Join(agentDir, ".agencycli", "context", "wakeup.md")
	wakeup, _ := os.ReadFile(wakeupPath)

	var skills []string
	seen := map[string]bool{}
	if t, err := s.st.Team(meta.Team); err == nil && t != nil {
		for _, sk := range t.Skills {
			if !seen[sk] {
				skills = append(skills, sk)
				seen[sk] = true
			}
		}
	}
	if meta.Role != "" {
		if rl, err := s.st.Role(meta.Team, meta.Role); err == nil && rl != nil {
			for _, sk := range rl.Skills {
				if !seen[sk] {
					skills = append(skills, sk)
					seen[sk] = true
				}
			}
		}
	}
	if skills == nil {
		skills = []string{}
	}

	resp := map[string]any{
		"contextFile": contextFile,
		"context":     string(merged),
		"wakeup":      string(wakeup),
		"model":       string(meta.Model),
		"team":        meta.Team,
		"role":        meta.Role,
		"syncedAt":    meta.SyncedAt,
		"skills":      skills,
		"workDir":     agentDir,
	}
	if meta.HTTPAgent != nil {
		resp["httpAgent"] = meta.HTTPAgent
	}
	if len(meta.Env) > 0 {
		resp["env"] = meta.Env
	}
	if meta.Provider != "" {
		resp["provider"] = meta.Provider
	}
	if meta.Sandbox != nil {
		resp["sandbox"] = meta.Sandbox
	}

	goalSummary := s.buildGoalSummary(project)
	if goalSummary != "" {
		resp["goals"] = goalSummary
	}

	resp["setupChecks"] = buildSetupChecks(meta)

	_ = json.NewEncoder(w).Encode(resp)
}

func contextFileName(model string) string {
	s := strings.ToLower(model)
	switch {
	case strings.Contains(s, "claude"):
		return "CLAUDE.md"
	case strings.Contains(s, "codex"):
		return "AGENTS.md"
	case strings.Contains(s, "gemini"):
		return "GEMINI.md"
	case strings.Contains(s, "cursor"):
		return ".cursorrules"
	default:
		return "context.md"
	}
}

// ── Agent wakeup prompt (editable) ───────────────────────────────────────────

func (s *Server) handlePutAgentWakeup(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	agent := r.PathValue("agent")

	if _, err := s.st.AgentMeta(project, agent); err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "agent not found")
			return
		}
		s.serverError(w, err)
		return
	}

	var body promptSaveBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	agentDir := s.st.AgentDir(project, agent)
	wakeupDir := filepath.Join(agentDir, ".agencycli", "context")
	if err := os.MkdirAll(wakeupDir, 0o755); err != nil {
		s.serverError(w, err)
		return
	}
	if err := os.WriteFile(filepath.Join(wakeupDir, "wakeup.md"), []byte(body.Content), 0o644); err != nil {
		s.serverError(w, err)
		return
	}

	// Ensure heartbeat.yaml references the wakeup file.
	hb, _ := s.ts.GetHeartbeat(project, agent)
	if hb == nil {
		hb = &entity.HeartbeatConfig{}
	}
	if hb.WakeupPrompt != "@.agencycli/context/wakeup.md" {
		hb.WakeupPrompt = "@.agencycli/context/wakeup.md"
		_ = s.ts.SaveHeartbeat(project, agent, hb)
	}

	// Re-sync so CLAUDE.md/@import picks up wakeup.md.
	s.syncAgent(project, agent)

	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ── Sync agent ────────────────────────────────────────────────────────────────

type syncBody struct {
	Agent string `json:"agent"`
}

func (s *Server) handlePostProjectSync(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	if _, err := s.st.Project(project); err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "project not found")
			return
		}
		s.serverError(w, err)
		return
	}

	var body syncBody
	if r.ContentLength > 0 {
		if err := s.readJSON(w, r, &body); err != nil {
			s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}
	agentName := strings.TrimSpace(body.Agent)

	bin, err := exec.LookPath("agencycli")
	if err != nil {
		bin, err = os.Executable()
		if err != nil {
			s.jsonError(w, http.StatusInternalServerError, "cannot find agencycli binary")
			return
		}
	}

	args := []string{"sync", "--dir", s.root, "--project", project}
	if agentName != "" {
		args = append(args, "--name", agentName)
	}

	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "sync failed: "+string(out))
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"output": string(out),
	})
}

func (s *Server) handlePutAgentSandbox(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	agent := r.PathValue("agent")

	meta, err := s.st.AgentMeta(project, agent)
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "agent not found")
			return
		}
		s.serverError(w, err)
		return
	}

	var body struct {
		Provider string `json:"provider"`
		Image    string `json:"image"`
		Network  string `json:"network"`
		MemoryMB int    `json:"memoryMb"`
	}
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if body.Provider == "" || body.Provider == "none" {
		meta.Sandbox = nil
	} else {
		meta.Sandbox = &entity.SandboxConfig{
			Provider: entity.SandboxProvider(body.Provider),
		}
		if body.Provider == "docker" {
			dc := &entity.DockerSandboxConfig{}
			if body.Image != "" {
				dc.Image = body.Image
			}
			if body.Network != "" {
				dc.NetworkMode = body.Network
			}
			if body.MemoryMB > 0 {
				dc.MemoryMB = body.MemoryMB
			}
			meta.Sandbox.Docker = dc
		}
	}

	if err := s.st.SaveAgentMeta(project, agent, meta); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) syncAgent(project, agent string) {
	bin, err := exec.LookPath("agencycli")
	if err != nil {
		bin, _ = os.Executable()
	}
	if bin == "" {
		return
	}
	cmd := exec.Command(bin, "sync", "--dir", s.root, "--project", project, "--name", agent, "--force")
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("sync after wakeup save failed", "project", project, "agent", agent, "err", err, "output", string(out))
	}
}

type setupCheck struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Status string `json:"status"` // ok, warning, error
	Detail string `json:"detail,omitempty"`
}

func buildSetupChecks(meta *entity.AgentMeta) []setupCheck {
	model := entity.NormaliseModel(meta.Model)
	if model == entity.ModelHuman || model == entity.ModelHTTPAgent {
		return nil
	}

	var checks []setupCheck
	isDocker := meta.Sandbox != nil && meta.Sandbox.Provider == entity.SandboxDocker

	// 1. CLI tool check
	cliName, installCmd := cliInfoForModel(model)
	if cliName != "" {
		if isDocker {
			checks = append(checks, setupCheck{
				Key: "cli", Label: cliName + " CLI", Status: "ok",
				Detail: "Docker 沙箱内已预装",
			})
		} else if _, err := exec.LookPath(cliName); err == nil {
			checks = append(checks, setupCheck{
				Key: "cli", Label: cliName + " CLI", Status: "ok",
			})
		} else {
			checks = append(checks, setupCheck{
				Key: "cli", Label: cliName + " CLI", Status: "error",
				Detail: "未安装。请运行: " + installCmd,
			})
		}
	}

	// 2. Docker check (if using docker sandbox)
	if isDocker {
		if _, err := exec.LookPath("docker"); err == nil {
			dcmd := exec.Command("docker", "info")
			dcmd.Stdout = nil
			dcmd.Stderr = nil
			if dcmd.Run() == nil {
				checks = append(checks, setupCheck{Key: "docker", Label: "Docker", Status: "ok"})
			} else {
				checks = append(checks, setupCheck{
					Key: "docker", Label: "Docker", Status: "error",
					Detail: "Docker 已安装但守护进程未运行。请运行: sudo systemctl start docker",
				})
			}
		} else {
			checks = append(checks, setupCheck{
				Key: "docker", Label: "Docker", Status: "error",
				Detail: "未安装。请访问: https://docs.docker.com/get-docker/",
			})
		}
	}

	// 3. Auth / credential check
	authCheck := checkAuthForModel(model, isDocker)
	if authCheck != nil {
		checks = append(checks, *authCheck)
	}

	// 4. API provider check
	if meta.Provider != "" {
		checks = append(checks, setupCheck{
			Key: "provider", Label: "API 供应商", Status: "ok",
			Detail: meta.Provider,
		})
	} else {
		switch model {
		case entity.ModelClaudeCode:
			if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" {
				checks = append(checks, setupCheck{Key: "provider", Label: "API 供应商", Status: "ok", Detail: "通过环境变量配置"})
			} else {
				checks = append(checks, setupCheck{
					Key: "provider", Label: "API 供应商", Status: "warning",
					Detail: "未配置 API 供应商或 ANTHROPIC_API_KEY。请在设置页添加供应商或设置环境变量",
				})
			}
		case entity.ModelCodex, entity.ModelQoder:
			if os.Getenv("OPENAI_API_KEY") != "" {
				checks = append(checks, setupCheck{Key: "provider", Label: "API 供应商", Status: "ok", Detail: "通过环境变量配置"})
			} else {
				checks = append(checks, setupCheck{
					Key: "provider", Label: "API 供应商", Status: "warning",
					Detail: "未配置 OPENAI_API_KEY。请在设置页添加供应商或设置环境变量",
				})
			}
		}
	}

	return checks
}

func cliInfoForModel(model entity.AgentModel) (name, install string) {
	switch model {
	case entity.ModelClaudeCode:
		return "claude", "npm install -g @anthropic-ai/claude-code"
	case entity.ModelCodex:
		return "codex", "npm install -g @openai/codex"
	case entity.ModelCursor:
		return "agent", "curl -fsSL https://www.cursor.com/install-agent.sh | sh"
	case entity.ModelGemini:
		return "gemini", "npm install -g @anthropic-ai/gemini-cli"
	case entity.ModelQoder:
		return "qoder", "npm install -g @anthropic-ai/qoder"
	case entity.ModelOpenCode:
		return "opencode", "go install github.com/opencode-ai/opencode@latest"
	}
	return "", ""
}

func checkAuthForModel(model entity.AgentModel, isDocker bool) *setupCheck {
	home, _ := os.UserHomeDir()
	switch model {
	case entity.ModelCursor:
		authFile := filepath.Join(home, ".config", "cursor", "auth.json")
		if _, err := os.Stat(authFile); err == nil {
			return &setupCheck{Key: "auth", Label: "Cursor 认证", Status: "ok"}
		}
		return &setupCheck{
			Key: "auth", Label: "Cursor 认证", Status: "error",
			Detail: "未登录。请运行: agent login",
		}
	case entity.ModelClaudeCode:
		claudeJSON := filepath.Join(home, ".claude.json")
		if _, err := os.Stat(claudeJSON); err == nil {
			return &setupCheck{Key: "auth", Label: "Claude 认证", Status: "ok"}
		}
		return &setupCheck{
			Key: "auth", Label: "Claude 认证", Status: "warning",
			Detail: "~/.claude.json 不存在（如果使用 API Key 可忽略）",
		}
	}
	return nil
}
