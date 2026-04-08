package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
)

// providerBody is the JSON request struct for creating/updating providers.
// Separate from entity.APIProvider because APIKey uses json:"-" on the entity
// to prevent leaking in responses, but we need to accept it in requests.
type providerBody struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	BaseURL string            `json:"baseUrl"`
	APIKey  string            `json:"apiKey"`
	Model   string            `json:"model"`
	Env     map[string]string `json:"env,omitempty"`
}

func (b providerBody) toEntity() entity.APIProvider {
	return entity.APIProvider{
		Name:    strings.TrimSpace(b.Name),
		Type:    strings.TrimSpace(b.Type),
		BaseURL: strings.TrimSpace(b.BaseURL),
		APIKey:  strings.TrimSpace(b.APIKey),
		Model:   strings.TrimSpace(b.Model),
		Env:     b.Env,
	}
}

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
	var body providerBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	prov := body.toEntity()
	if prov.Name == "" {
		s.jsonError(w, http.StatusBadRequest, "name is required")
		return
	}
	p, err := s.providerStore().Add(prov)
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(providerToJSON(*p))
}

func (s *Server) handleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body providerBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	ps := s.providerStore()
	existing, err := ps.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	prov := body.toEntity()
	// If no new key provided, keep the existing one.
	if prov.APIKey == "" {
		prov.APIKey = existing.APIKey
	}
	p, err := ps.Update(id, prov)
	if err != nil {
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
