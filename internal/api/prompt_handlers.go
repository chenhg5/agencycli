package api

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	}
	if meta.HTTPAgent != nil {
		resp["httpAgent"] = meta.HTTPAgent
	}
	if len(meta.Env) > 0 {
		resp["env"] = meta.Env
	}
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
