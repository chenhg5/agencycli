package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
)

func (s *Server) providerStore() *store.ProviderStore {
	return store.NewProviderStore(s.root)
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	items, err := s.providerStore().List()
	if err != nil {
		s.serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, p := range items {
		out = append(out, providerToJSON(p))
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleAddProvider(w http.ResponseWriter, r *http.Request) {
	var body entity.APIProvider
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		s.jsonError(w, http.StatusBadRequest, "name is required")
		return
	}
	p, err := s.providerStore().Add(body)
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(providerToJSON(*p))
}

func (s *Server) handleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body entity.APIProvider
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	p, err := s.providerStore().Update(id, body)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(providerToJSON(*p))
}

func (s *Server) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.providerStore().Remove(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func providerToJSON(p entity.APIProvider) map[string]any {
	out := map[string]any{
		"id":      p.ID,
		"name":    p.Name,
		"type":    p.Type,
		"baseUrl": p.BaseURL,
		"model":   p.Model,
		"hasKey":  p.APIKey != "",
	}
	if len(p.Env) > 0 {
		out["env"] = p.Env
	}
	return out
}
