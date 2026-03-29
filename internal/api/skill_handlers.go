package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type skillRow struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (s *Server) handleListSkills(w http.ResponseWriter, _ *http.Request) {
	skills, err := s.st.ListSkills()
	if err != nil {
		s.serverError(w, err)
		return
	}
	out := make([]skillRow, 0, len(skills))
	for _, sk := range skills {
		if sk == nil {
			continue
		}
		out = append(out, skillRow{Name: sk.Name, Description: sk.Description})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleGetSkillDetail(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sk, err := s.st.Skill(name)
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "skill not found")
			return
		}
		s.serverError(w, err)
		return
	}
	prompt, err := s.st.SkillPrompt(name)
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":        sk.Name,
		"description": sk.Description,
		"prompt":      prompt,
		"dir":         s.st.SkillDir(name),
	})
}

func (s *Server) handlePutSkillPrompt(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sk, err := s.st.Skill(name)
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "skill not found")
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

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", sk.Name))
	if sk.Description != "" {
		sb.WriteString(fmt.Sprintf("description: %q\n", sk.Description))
	}
	sb.WriteString("---\n")
	if body.Content != "" {
		sb.WriteString("\n")
		sb.WriteString(body.Content)
		if !strings.HasSuffix(body.Content, "\n") {
			sb.WriteString("\n")
		}
	}

	skillMD := filepath.Join(s.st.SkillDir(name), "SKILL.md")
	if err := os.WriteFile(skillMD, []byte(sb.String()), 0o644); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ── Role / Team skill binding ─────────────────────────────────────────────────

type skillBindBody struct {
	Team   string `json:"team"`
	Role   string `json:"role"`
	Skill  string `json:"skill"`
	Action string `json:"action"` // "add" or "remove"
}

func (s *Server) handlePostRoleSkillBind(w http.ResponseWriter, r *http.Request) {
	var body skillBindBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	teamPath := strings.TrimSpace(body.Team)
	roleName := strings.TrimSpace(body.Role)
	skillName := strings.TrimSpace(body.Skill)
	action := strings.TrimSpace(body.Action)

	if teamPath == "" || roleName == "" || skillName == "" {
		s.jsonError(w, http.StatusBadRequest, "team, role, and skill are required")
		return
	}
	if action != "add" && action != "remove" {
		s.jsonError(w, http.StatusBadRequest, "action must be add or remove")
		return
	}

	role, err := s.st.Role(teamPath, roleName)
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "role not found")
			return
		}
		s.serverError(w, err)
		return
	}

	if action == "add" {
		if _, err := s.st.Skill(skillName); err != nil {
			s.jsonError(w, http.StatusBadRequest, "skill not found: "+skillName)
			return
		}
		for _, sk := range role.Skills {
			if sk == skillName {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skills": role.Skills})
				return
			}
		}
		role.Skills = append(role.Skills, skillName)
	} else {
		filtered := make([]string, 0, len(role.Skills))
		for _, sk := range role.Skills {
			if sk != skillName {
				filtered = append(filtered, sk)
			}
		}
		role.Skills = filtered
	}

	if err := s.st.SaveRole(teamPath, roleName, role); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skills": role.Skills})
}

func (s *Server) handlePostTeamSkillBind(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Team   string `json:"team"`
		Skill  string `json:"skill"`
		Action string `json:"action"`
	}
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	teamPath := strings.TrimSpace(body.Team)
	skillName := strings.TrimSpace(body.Skill)
	action := strings.TrimSpace(body.Action)

	if teamPath == "" || skillName == "" {
		s.jsonError(w, http.StatusBadRequest, "team and skill are required")
		return
	}
	if action != "add" && action != "remove" {
		s.jsonError(w, http.StatusBadRequest, "action must be add or remove")
		return
	}

	team, err := s.st.Team(teamPath)
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "team not found")
			return
		}
		s.serverError(w, err)
		return
	}

	if action == "add" {
		if _, err := s.st.Skill(skillName); err != nil {
			s.jsonError(w, http.StatusBadRequest, "skill not found: "+skillName)
			return
		}
		for _, sk := range team.Skills {
			if sk == skillName {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skills": team.Skills})
				return
			}
		}
		team.Skills = append(team.Skills, skillName)
	} else {
		filtered := make([]string, 0, len(team.Skills))
		for _, sk := range team.Skills {
			if sk != skillName {
				filtered = append(filtered, sk)
			}
		}
		team.Skills = filtered
	}

	if err := s.st.SaveTeam(teamPath, team); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skills": team.Skills})
}
