package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
)

// ── Workspace Secrets CRUD ──────────────────────────────────────────────────

func (s *Server) handleListSecrets(w http.ResponseWriter, r *http.Request) {
	ss := store.NewSecretStore(s.root)
	items, err := ss.List()
	if err != nil {
		s.serverError(w, err)
		return
	}
	type row struct {
		ID          string              `json:"id"`
		Key         string              `json:"key"`
		Value       string              `json:"value"`
		Scope       entity.SecretScope  `json:"scope"`
		Agents      []string            `json:"agents,omitempty"`
		Description string              `json:"description,omitempty"`
		CreatedAt   string              `json:"createdAt"`
		UpdatedAt   string              `json:"updatedAt"`
	}
	rows := make([]row, 0, len(items))
	for _, sec := range items {
		rows = append(rows, row{
			ID:          sec.ID,
			Key:         sec.Key,
			Value:       sec.Value,
			Scope:       sec.Scope,
			Agents:      sec.Agents,
			Description: sec.Description,
			CreatedAt:   sec.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   sec.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	_ = json.NewEncoder(w).Encode(rows)
}

func (s *Server) handleCreateSecret(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key         string             `json:"key"`
		Value       string             `json:"value"`
		Scope       entity.SecretScope `json:"scope"`
		Agents      []string           `json:"agents"`
		Description string             `json:"description"`
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
	sec := entity.Secret{
		Key:         req.Key,
		Value:       req.Value,
		Scope:       req.Scope,
		Agents:      req.Agents,
		Description: req.Description,
	}
	ss := store.NewSecretStore(s.root)
	created, err := ss.Add(sec)
	if err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": created.ID})
}

func (s *Server) handleUpdateSecret(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "secret id required", http.StatusBadRequest)
		return
	}
	var req struct {
		Key         string             `json:"key"`
		Value       *string            `json:"value"`
		Scope       entity.SecretScope `json:"scope"`
		Agents      []string           `json:"agents"`
		Description string             `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ss := store.NewSecretStore(s.root)
	existing, err := ss.Get(id)
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

	if _, err := ss.Update(id, *existing); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "secret id required", http.StatusBadRequest)
		return
	}
	ss := store.NewSecretStore(s.root)
	if err := ss.Remove(id); err != nil {
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
	req := struct{ Key string }{Key: key}
	meta, err := s.st.AgentMeta(project, agent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if meta.Env != nil {
		delete(meta.Env, req.Key)
	}
	if err := s.st.SaveAgentMeta(project, agent, meta); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

