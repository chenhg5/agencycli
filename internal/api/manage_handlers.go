package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
)

// ── Create Role ──────────────────────────────────────────────────────────────

type createRoleBody struct {
	Team        string   `json:"team"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Skills      []string `json:"skills"`
	SetupDirs   []string `json:"setupDirs"`
}

func (s *Server) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var body createRoleBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	team := strings.TrimSpace(body.Team)
	name := strings.TrimSpace(body.Name)
	if team == "" || name == "" {
		s.jsonError(w, http.StatusBadRequest, "team and name are required")
		return
	}

	if _, err := s.st.Team(team); err != nil {
		s.jsonError(w, http.StatusNotFound, fmt.Sprintf("team %q not found", team))
		return
	}

	roleDir := s.st.RoleDir(team, name)
	if _, err := os.Stat(roleDir); err == nil {
		s.jsonError(w, http.StatusConflict, fmt.Sprintf("role %q already exists under team %q", name, team))
		return
	}

	role := &entity.Role{
		Name:        name,
		Description: strings.TrimSpace(body.Description),
		Skills:      body.Skills,
		Setup: entity.RoleSetup{
			Dirs: body.SetupDirs,
		},
	}
	if err := s.st.SaveRole(team, name, role); err != nil {
		s.serverError(w, err)
		return
	}

	stub := fmt.Sprintf("# Role: %s\n\n", name)
	if body.Description != "" {
		stub += strings.TrimSpace(body.Description) + "\n\n"
	}
	stub += "<!-- Describe this role's responsibilities, working style, and expectations. -->\n"
	_ = s.st.SaveRolePrompt(team, name, stub)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":   true,
		"name": name,
		"team": team,
	})
}

// ── Hire Agent ───────────────────────────────────────────────────────────────

type hireAgentBody struct {
	Name  string `json:"name"`
	Team  string `json:"team"`
	Role  string `json:"role"`
	Model string `json:"model"`
}

func (s *Server) handleHireAgent(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	if project == "" {
		s.jsonError(w, http.StatusBadRequest, "missing project name")
		return
	}

	var body hireAgentBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	agentName := strings.TrimSpace(body.Name)
	team := strings.TrimSpace(body.Team)
	model := strings.TrimSpace(body.Model)
	role := strings.TrimSpace(body.Role)

	if agentName == "" || team == "" || model == "" {
		s.jsonError(w, http.StatusBadRequest, "name, team, and model are required")
		return
	}

	args := []string{
		"--dir", s.root,
		"hire",
		"--project", project,
		"--team", team,
		"--model", model,
		"--name", agentName,
	}
	if role != "" {
		args = append(args, "--role", role)
	}

	cmd := exec.Command(s.sched.binPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("hire failed: %v\n%s", err, string(out)))
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"output": string(out),
		"agent":  agentName,
	})
}
