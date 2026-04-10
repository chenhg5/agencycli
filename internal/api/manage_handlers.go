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

	// Auto-link user account when hiring a human
	if entity.AgentModel(model) == entity.ModelHuman {
		if u := s.users.GetUser(agentName); u != nil {
			agentID := project + "/" + agentName
			hasLink := false
			for _, la := range u.LinkedAgents {
				if la == agentID {
					hasLink = true
					break
				}
			}
			if !hasLink {
				newLinked := append(u.LinkedAgents, agentID)
				hasProj := false
				for _, pa := range u.Projects {
					if pa.Project == project {
						hasProj = true
						break
					}
				}
				var newProjects []projectAccess
				if hasProj {
					newProjects = u.Projects
				} else {
					newProjects = append(u.Projects, projectAccess{Project: project, Role: ProjectRoleOperator})
				}
				_ = s.users.UpdateUser(agentName, nil, nil, nil, nil, nil, nil, nil, newProjects, newLinked, nil)
			}
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"output": string(out),
		"agent":  agentName,
	})
}

// ── Run Agent ────────────────────────────────────────────────────────────────

type runAgentBody struct {
	Project string `json:"project"`
	Agent   string `json:"agent"`
	TaskID  string `json:"taskId"`
	Prompt  string `json:"prompt"`
	Title   string `json:"title"`
}

func (s *Server) handleRunAgent(w http.ResponseWriter, r *http.Request) {
	var body runAgentBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)
	if project == "" || agent == "" {
		s.jsonError(w, http.StatusBadRequest, "project and agent are required")
		return
	}

	var allOutput strings.Builder

	// If prompt is provided, create a task first
	if prompt := strings.TrimSpace(body.Prompt); prompt != "" {
		title := strings.TrimSpace(body.Title)
		if title == "" {
			runes := []rune(prompt)
			if len(runes) > 40 {
				title = string(runes[:40]) + "…"
			} else {
				title = prompt
			}
		}
		addArgs := []string{
			"--dir", s.root, "task", "add",
			"--project", project, "--agent", agent,
			"--title", title, "--prompt", prompt,
			"--priority", "0",
		}
		addCmd := exec.Command(s.sched.binPath, addArgs...)
		addOut, err := addCmd.CombinedOutput()
		allOutput.WriteString(string(addOut))
		if err != nil {
			s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("task creation failed: %v\n%s", err, string(addOut)))
			return
		}
		allOutput.WriteString("\n")
	}

	args := []string{"--dir", s.root, "run", "--project", project, "--agent", agent}
	if body.TaskID != "" {
		args = append(args, "--task", strings.TrimSpace(body.TaskID))
	}

	cmd := exec.Command(s.sched.binPath, args...)
	out, err := cmd.CombinedOutput()
	allOutput.WriteString(string(out))
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("run failed: %v\n%s", err, allOutput.String()))
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "output": allOutput.String()})
}

// ── Set Model ────────────────────────────────────────────────────────────────

type setModelBody struct {
	Model       string `json:"model"`
	HttpURL     string `json:"httpUrl,omitempty"`
	HttpModel   string `json:"httpModel,omitempty"`
	HttpAPIKey  string `json:"httpApiKey,omitempty"`
	HttpTimeout string `json:"httpTimeout,omitempty"`
	HttpStream  *bool  `json:"httpStream,omitempty"`
}

func (s *Server) handleSetModel(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	agent := r.PathValue("agent")
	if project == "" || agent == "" {
		s.jsonError(w, http.StatusBadRequest, "missing project or agent")
		return
	}

	var body setModelBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	model := strings.TrimSpace(body.Model)
	if model == "" {
		s.jsonError(w, http.StatusBadRequest, "model is required")
		return
	}

	args := []string{
		"--dir", s.root,
		"agent", "set-model",
		"--project", project,
		"--name", agent,
		"--model", model,
	}

	if model == "http-agent" {
		if u := strings.TrimSpace(body.HttpURL); u != "" {
			args = append(args, "--http-url", u)
		}
		if m := strings.TrimSpace(body.HttpModel); m != "" {
			args = append(args, "--http-model", m)
		}
		if k := strings.TrimSpace(body.HttpAPIKey); k != "" {
			args = append(args, "--http-api-key", k)
		}
		if t := strings.TrimSpace(body.HttpTimeout); t != "" {
			args = append(args, "--http-timeout", t)
		}
		if body.HttpStream != nil {
			if *body.HttpStream {
				args = append(args, "--http-stream")
			} else {
				args = append(args, "--http-stream=false")
			}
		}
	}

	cmd := exec.Command(s.sched.binPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("set-model failed: %v\n%s", err, string(out)))
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"output": string(out),
	})
}

// ── Session Reset ────────────────────────────────────────────────────────────

type sessionResetBody struct {
	Project string `json:"project"`
	Agent   string `json:"agent"`
}

func (s *Server) handleSessionReset(w http.ResponseWriter, r *http.Request) {
	var body sessionResetBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)
	if project == "" || agent == "" {
		s.jsonError(w, http.StatusBadRequest, "project and agent are required")
		return
	}

	hb, err := s.ts.GetHeartbeat(project, agent)
	if err != nil {
		s.serverError(w, err)
		return
	}
	oldID := hb.SessionID
	hb.SessionID = ""
	hb.SessionStartedAt = nil
	if err := s.ts.SaveHeartbeat(project, agent, hb); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "oldSessionId": oldID})
}

// ── Agent Environment Variables ──────────────────────────────────────────────

type agentEnvBody struct {
	Env      map[string]string `json:"env"`
	Provider *string           `json:"provider,omitempty"`
}

func (s *Server) handlePutAgentEnv(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	agent := r.PathValue("agent")

	var body agentEnvBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	meta, err := s.st.AgentMeta(project, agent)
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "agent not found")
			return
		}
		s.serverError(w, err)
		return
	}

	// Remove empty-value entries
	cleaned := make(map[string]string)
	for k, v := range body.Env {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k != "" && v != "" {
			cleaned[k] = v
		}
	}
	if len(cleaned) == 0 {
		meta.Env = nil
	} else {
		meta.Env = cleaned
	}
	if body.Provider != nil {
		meta.Provider = strings.TrimSpace(*body.Provider)
	}

	if err := s.st.SaveAgentMeta(project, agent, meta); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "env": meta.Env, "provider": meta.Provider})
}
