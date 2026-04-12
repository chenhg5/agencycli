package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
)

// ── OKR endpoints ────────────────────────────────────────────────────────────

func (s *Server) handleListOKRs(w http.ResponseWriter, r *http.Request) {
	f, err := s.okrStore.Load()
	if err != nil {
		s.serverError(w, err)
		return
	}
	scope := entity.OKRScope(r.URL.Query().Get("scope"))
	scopeRef := r.URL.Query().Get("scopeRef")
	okrs, err := s.okrStore.ListOKRs(scope, scopeRef)
	if err != nil {
		s.serverError(w, err)
		return
	}
	type krJSON struct {
		entity.KeyResult
		Progress float64 `json:"progress"`
	}
	type okrJSON struct {
		ID          string              `json:"id"`
		Scope       entity.OKRScope     `json:"scope,omitempty"`
		ScopeRef    string              `json:"scopeRef,omitempty"`
		ParentID    string              `json:"parentId,omitempty"`
		Objective   string              `json:"objective"`
		Description string              `json:"description,omitempty"`
		Owner       string              `json:"owner"`
		Quarter     string              `json:"quarter"`
		Status      entity.OKRStatus    `json:"status"`
		KeyResults  []krJSON            `json:"keyResults"`
		ReviewNotes []entity.ReviewNote `json:"reviewNotes,omitempty"`
		Progress    float64             `json:"progress"`
		CreatedAt   time.Time           `json:"createdAt"`
		UpdatedAt   time.Time           `json:"updatedAt"`
	}
	out := struct {
		CurrentQuarter string    `json:"currentQuarter"`
		OKRs           []okrJSON `json:"okrs"`
	}{CurrentQuarter: f.CurrentQuarter}
	for _, o := range okrs {
		krs := make([]krJSON, 0, len(o.KeyResults))
		for _, kr := range o.KeyResults {
			krs = append(krs, krJSON{KeyResult: kr, Progress: kr.Progress()})
		}
		oScope := o.Scope
		if oScope == "" {
			oScope = entity.OKRScopeAgency
		}
		out.OKRs = append(out.OKRs, okrJSON{
			ID: o.ID, Scope: oScope, ScopeRef: o.ScopeRef, ParentID: o.ParentID,
			Objective: o.Objective, Description: o.Description,
			Owner: o.Owner, Quarter: o.Quarter,
			Status: o.Status, KeyResults: krs, ReviewNotes: o.ReviewNotes,
			Progress: o.Progress(), CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
		})
	}
	if out.OKRs == nil {
		out.OKRs = []okrJSON{}
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleGetOKR(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	o, err := s.okrStore.GetOKR(id)
	if err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	type krWithProgress struct {
		entity.KeyResult
		Progress float64 `json:"progress"`
	}
	krs := make([]krWithProgress, 0, len(o.KeyResults))
	for _, kr := range o.KeyResults {
		krs = append(krs, krWithProgress{KeyResult: kr, Progress: kr.Progress()})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":          o.ID,
		"scope":       o.Scope,
		"scopeRef":    o.ScopeRef,
		"parentId":    o.ParentID,
		"objective":   o.Objective,
		"description": o.Description,
		"owner":       o.Owner,
		"quarter":     o.Quarter,
		"status":      o.Status,
		"keyResults":  krs,
		"reviewNotes": o.ReviewNotes,
		"progress":    o.Progress(),
		"createdAt":   o.CreatedAt,
		"updatedAt":   o.UpdatedAt,
	})
}

func (s *Server) handleCreateOKR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scope       entity.OKRScope    `json:"scope"`
		ScopeRef    string             `json:"scopeRef"`
		ParentID    string             `json:"parentId"`
		Objective   string             `json:"objective"`
		Description string             `json:"description"`
		Owner       string             `json:"owner"`
		Quarter     string             `json:"quarter"`
		Status      entity.OKRStatus   `json:"status"`
		KeyResults  []entity.KeyResult `json:"keyResults"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Objective == "" {
		s.jsonError(w, http.StatusBadRequest, "objective is required")
		return
	}
	okr := entity.OKR{
		Scope:       req.Scope,
		ScopeRef:    req.ScopeRef,
		ParentID:    req.ParentID,
		Objective:   req.Objective,
		Description: req.Description,
		Owner:       req.Owner,
		Quarter:     req.Quarter,
		Status:      req.Status,
		KeyResults:  req.KeyResults,
	}
	created, err := s.okrStore.CreateOKR(okr)
	if err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (s *Server) handleUpdateOKR(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Objective   *string            `json:"objective"`
		Description *string            `json:"description"`
		Owner       *string            `json:"owner"`
		Quarter     *string            `json:"quarter"`
		Status      *entity.OKRStatus  `json:"status"`
		Scope       *entity.OKRScope   `json:"scope"`
		ScopeRef    *string            `json:"scopeRef"`
		ParentID    *string            `json:"parentId"`
		KeyResults  []entity.KeyResult `json:"keyResults"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	err := s.okrStore.UpdateOKR(id, func(o *entity.OKR) {
		if req.Objective != nil {
			o.Objective = *req.Objective
		}
		if req.Description != nil {
			o.Description = *req.Description
		}
		if req.Owner != nil {
			o.Owner = *req.Owner
		}
		if req.Quarter != nil {
			o.Quarter = *req.Quarter
		}
		if req.Status != nil {
			o.Status = *req.Status
		}
		if req.Scope != nil {
			o.Scope = *req.Scope
		}
		if req.ScopeRef != nil {
			o.ScopeRef = *req.ScopeRef
		}
		if req.ParentID != nil {
			o.ParentID = *req.ParentID
		}
		if req.KeyResults != nil {
			o.KeyResults = req.KeyResults
		}
	})
	if err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	o, _ := s.okrStore.GetOKR(id)
	_ = json.NewEncoder(w).Encode(o)
}

func (s *Server) handleDeleteOKR(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.okrStore.DeleteOKR(id); err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// ── Key Result endpoints ─────────────────────────────────────────────────────

func (s *Server) handleAddKR(w http.ResponseWriter, r *http.Request) {
	okrID := r.PathValue("id")
	var kr entity.KeyResult
	if err := json.NewDecoder(r.Body).Decode(&kr); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if kr.Description == "" {
		s.jsonError(w, http.StatusBadRequest, "description is required")
		return
	}
	created, err := s.okrStore.AddKR(okrID, kr)
	if err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (s *Server) handleUpdateKR(w http.ResponseWriter, r *http.Request) {
	okrID := r.PathValue("id")
	krID := r.PathValue("krId")
	var req struct {
		Description  *string  `json:"description"`
		CurrentValue *float64 `json:"currentValue"`
		TargetValue  *float64 `json:"targetValue"`
		Unit         *string  `json:"unit"`
		Weight       *float64 `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	err := s.okrStore.UpdateKR(okrID, krID, func(kr *entity.KeyResult) {
		if req.Description != nil {
			kr.Description = *req.Description
		}
		if req.CurrentValue != nil {
			kr.CurrentValue = *req.CurrentValue
		}
		if req.TargetValue != nil {
			kr.TargetValue = *req.TargetValue
		}
		if req.Unit != nil {
			kr.Unit = *req.Unit
		}
		if req.Weight != nil {
			kr.Weight = *req.Weight
		}
	})
	if err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) handleDeleteKR(w http.ResponseWriter, r *http.Request) {
	okrID := r.PathValue("id")
	krID := r.PathValue("krId")
	err := s.okrStore.UpdateOKR(okrID, func(o *entity.OKR) {
		for i := range o.KeyResults {
			if o.KeyResults[i].ID == krID {
				o.KeyResults = append(o.KeyResults[:i], o.KeyResults[i+1:]...)
				return
			}
		}
	})
	if err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// ── OKR review notes ─────────────────────────────────────────────────────────

func (s *Server) handleAddReviewNote(w http.ResponseWriter, r *http.Request) {
	okrID := r.PathValue("id")
	var note entity.ReviewNote
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if note.Date == "" {
		note.Date = time.Now().UTC().Format("2006-01-02")
	}
	err := s.okrStore.UpdateOKR(okrID, func(o *entity.OKR) {
		o.ReviewNotes = append(o.ReviewNotes, note)
	})
	if err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// ── Milestone endpoints ──────────────────────────────────────────────────────

func (s *Server) handleListMilestones(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	list, err := s.msStore.List(project)
	if err != nil {
		s.serverError(w, err)
		return
	}
	if list == nil {
		list = []entity.Milestone{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"milestones": list})
}

func (s *Server) handleGetMilestone(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	id := r.PathValue("msId")
	ms, err := s.msStore.Get(project, id)
	if err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	_ = json.NewEncoder(w).Encode(ms)
}

func (s *Server) handleCreateMilestone(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	var ms entity.Milestone
	if err := json.NewDecoder(r.Body).Decode(&ms); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if ms.Title == "" {
		s.jsonError(w, http.StatusBadRequest, "title is required")
		return
	}
	created, err := s.msStore.Create(project, ms)
	if err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (s *Server) handleUpdateMilestone(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	id := r.PathValue("msId")
	var req struct {
		Title       *string                  `json:"title"`
		Description *string                  `json:"description"`
		Status      *entity.MilestoneStatus  `json:"status"`
		DueDate     *time.Time               `json:"dueDate"`
		Owner       *string                  `json:"owner"`
		Progress    *int                     `json:"progress"`
		Criteria    []string                 `json:"criteria"`
		LinkedKR    []string                 `json:"linkedKR"`
		TaskLabels  []string                 `json:"taskLabels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	err := s.msStore.Update(project, id, func(ms *entity.Milestone) {
		if req.Title != nil {
			ms.Title = *req.Title
		}
		if req.Description != nil {
			ms.Description = *req.Description
		}
		if req.Status != nil {
			ms.Status = *req.Status
		}
		if req.DueDate != nil {
			ms.DueDate = req.DueDate
		}
		if req.Owner != nil {
			ms.Owner = *req.Owner
		}
		if req.Progress != nil {
			ms.Progress = *req.Progress
		}
		if req.Criteria != nil {
			ms.Criteria = req.Criteria
		}
		if req.LinkedKR != nil {
			ms.LinkedKR = req.LinkedKR
		}
		if req.TaskLabels != nil {
			ms.TaskLabels = req.TaskLabels
		}
	})
	if err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	ms, _ := s.msStore.Get(project, id)
	_ = json.NewEncoder(w).Encode(ms)
}

func (s *Server) handleDeleteMilestone(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	id := r.PathValue("msId")
	if err := s.msStore.Delete(project, id); err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
