package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
)

// ── Workspace EnvVars CRUD ──────────────────────────────────────────────────

func (s *Server) handleListEnvVars(w http.ResponseWriter, r *http.Request) {
	es := store.NewEnvVarStore(s.root)
	items, err := es.List()
	if err != nil {
		s.serverError(w, err)
		return
	}
	type row struct {
		ID          string              `json:"id"`
		Key         string              `json:"key"`
		Value       string              `json:"value"`
		Scope       entity.EnvVarScope  `json:"scope"`
		Agents      []string            `json:"agents,omitempty"`
		Description string              `json:"description,omitempty"`
		CreatedAt   string              `json:"createdAt"`
		UpdatedAt   string              `json:"updatedAt"`
	}
	rows := make([]row, 0, len(items))
	for _, ev := range items {
		rows = append(rows, row{
			ID:          ev.ID,
			Key:         ev.Key,
			Value:       ev.Value,
			Scope:       ev.Scope,
			Agents:      ev.Agents,
			Description: ev.Description,
			CreatedAt:   ev.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   ev.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	_ = json.NewEncoder(w).Encode(rows)
}

func (s *Server) handleCreateEnvVar(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key         string              `json:"key"`
		Value       string              `json:"value"`
		Scope       entity.EnvVarScope  `json:"scope"`
		Agents      []string            `json:"agents"`
		Description string              `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}
	ev := entity.EnvVar{
		Key:         req.Key,
		Value:       req.Value,
		Scope:       req.Scope,
		Agents:      req.Agents,
		Description: req.Description,
	}
	es := store.NewEnvVarStore(s.root)
	created, err := es.Add(ev)
	if err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": created.ID})
}

func (s *Server) handleUpdateEnvVar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "envvar id required", http.StatusBadRequest)
		return
	}
	var req struct {
		Key         string              `json:"key"`
		Value       *string             `json:"value"`
		Scope       entity.EnvVarScope  `json:"scope"`
		Agents      []string            `json:"agents"`
		Description string              `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	es := store.NewEnvVarStore(s.root)
	existing, err := es.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if req.Key != "" {
		existing.Key = strings.TrimSpace(req.Key)
	}
	if req.Value != nil {
		existing.Value = *req.Value
	}
	if req.Scope != "" {
		existing.Scope = req.Scope
	}
	existing.Agents = req.Agents
	existing.Description = req.Description

	if _, err := es.Update(id, *existing); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteEnvVar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "envvar id required", http.StatusBadRequest)
		return
	}
	es := store.NewEnvVarStore(s.root)
	if err := es.Remove(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Per-agent Env CRUD ──────────────────────────────────────────────────────

func (s *Server) handleGetAgentEnv(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	agent := r.PathValue("agent")
	if project == "" || agent == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	meta, err := s.st.AgentMeta(project, agent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	env := meta.Env
	if env == nil {
		env = make(map[string]string)
	}
	type envEntry struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	entries := make([]envEntry, 0, len(env))
	for k, v := range env {
		entries = append(entries, envEntry{Key: k, Value: v})
	}
	_ = json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleSetAgentEnv(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	agent := r.PathValue("agent")
	if project == "" || agent == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}
	meta, err := s.st.AgentMeta(project, agent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if meta.Env == nil {
		meta.Env = make(map[string]string)
	}
	meta.Env[req.Key] = req.Value
	if err := s.st.SaveAgentMeta(project, agent, meta); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteAgentEnv(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	agent := r.PathValue("agent")
	if project == "" || agent == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		http.Error(w, "key query param is required", http.StatusBadRequest)
		return
	}
	meta, err := s.st.AgentMeta(project, agent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if meta.Env != nil {
		delete(meta.Env, key)
	}
	if err := s.st.SaveAgentMeta(project, agent, meta); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
