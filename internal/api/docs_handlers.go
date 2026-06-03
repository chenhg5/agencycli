package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenhg5/agencycli/internal/store"
)

func (s *Server) handleDocsTree(w http.ResponseWriter, r *http.Request) {
	ds := store.NewDocsStore(s.root)
	tree, err := ds.Tree()
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(tree)
}

func (s *Server) handleDocsList(w http.ResponseWriter, r *http.Request) {
	ds := store.NewDocsStore(s.root)
	docs, err := ds.List()
	if err != nil {
		s.serverError(w, err)
		return
	}

	index := r.URL.Query().Get("index")
	tag := r.URL.Query().Get("tag")
	q := r.URL.Query().Get("q")

	if q != "" {
		results, err := ds.Search(q)
		if err != nil {
			s.serverError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(results)
		return
	}

	if index != "" || tag != "" {
		var filtered []*store.DocEntry
		for _, d := range docs {
			if index != "" && !strings.HasPrefix(d.Index, index) {
				continue
			}
			if tag != "" {
				has := false
				for _, t := range d.Tags {
					if strings.EqualFold(t, tag) {
						has = true
						break
					}
				}
				if !has {
					continue
				}
			}
			filtered = append(filtered, d)
		}
		_ = json.NewEncoder(w).Encode(filtered)
		return
	}

	_ = json.NewEncoder(w).Encode(docs)
}

func (s *Server) handleDocsGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ds := store.NewDocsStore(s.root)
	doc, err := ds.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}

	withContent := r.URL.Query().Get("content") == "true"
	type resp struct {
		*store.DocEntry
		Content string `json:"content,omitempty"`
	}
	out := resp{DocEntry: doc}
	if withContent {
		content, err := ds.ReadContent(doc.FilePath)
		if err != nil {
			out.Content = "Error reading file: " + err.Error()
		} else {
			out.Content = content
		}
	}
	_ = json.NewEncoder(w).Encode(out)
}

type docsAddBody struct {
	FilePath    string   `json:"filePath"`
	Title       string   `json:"title"`
	Index       string   `json:"index"`
	CreatedBy   string   `json:"createdBy"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

func (s *Server) handleDocsAdd(w http.ResponseWriter, r *http.Request) {
	var body docsAddBody
	if err := s.readJSON(w, r, &body); err != nil {
		return
	}
	if body.FilePath == "" {
		s.jsonError(w, http.StatusBadRequest, "filePath is required")
		return
	}
	if body.CreatedBy == "" {
		s.jsonError(w, http.StatusBadRequest, "createdBy is required")
		return
	}
	if _, err := os.Stat(body.FilePath); err != nil {
		s.jsonError(w, http.StatusBadRequest, "file not found: "+body.FilePath)
		return
	}

	ds := store.NewDocsStore(s.root)
	entry := &store.DocEntry{
		Title:       body.Title,
		FilePath:    body.FilePath,
		Index:       strings.Trim(body.Index, "/"),
		CreatedBy:   body.CreatedBy,
		Tags:        body.Tags,
		Description: body.Description,
	}
	if entry.Title == "" {
		parts := strings.Split(entry.FilePath, "/")
		entry.Title = parts[len(parts)-1]
	}
	if err := ds.Add(entry); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(docsEntryResponse(r, entry))
}

func docsWebPath(docID string) string {
	return "/docs/" + url.PathEscape(docID)
}

func docsEntryResponse(r *http.Request, entry *store.DocEntry) map[string]any {
	webPath := docsWebPath(entry.ID)
	out := map[string]any{
		"id":          entry.ID,
		"title":       entry.Title,
		"filePath":    entry.FilePath,
		"index":       entry.Index,
		"createdBy":   entry.CreatedBy,
		"tags":        entry.Tags,
		"description": entry.Description,
		"createdAt":   entry.CreatedAt,
		"updatedAt":   entry.UpdatedAt,
		"webPath":     webPath,
		"webUrl":      requestBaseURL(r) + webPath,
	}
	return out
}

func requestBaseURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host
}

type docsUpdateBody struct {
	Title       *string  `json:"title,omitempty"`
	Index       *string  `json:"index,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Description *string  `json:"description,omitempty"`
}

func (s *Server) handleDocsUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body docsUpdateBody
	if err := s.readJSON(w, r, &body); err != nil {
		return
	}
	ds := store.NewDocsStore(s.root)
	if err := ds.Update(id, func(e *store.DocEntry) {
		if body.Title != nil {
			e.Title = *body.Title
		}
		if body.Index != nil {
			e.Index = strings.Trim(*body.Index, "/")
		}
		if body.Tags != nil {
			e.Tags = body.Tags
		}
		if body.Description != nil {
			e.Description = *body.Description
		}
	}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	doc, _ := ds.Get(id)
	_ = json.NewEncoder(w).Encode(doc)
}

func (s *Server) handleDocsDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ds := store.NewDocsStore(s.root)
	doc, err := ds.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	data, err := os.ReadFile(doc.FilePath)
	if err != nil {
		s.serverError(w, err)
		return
	}
	filename := filepath.Base(doc.FilePath)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Write(data)
}

func (s *Server) handleDocsDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ds := store.NewDocsStore(s.root)
	if err := ds.Remove(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ── refs handlers ─────────────────────────────────────────────────────────────

func (s *Server) handleDocsGetRefs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ds := store.NewDocsStore(s.root)
	refs, err := ds.GetRefs(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	backrefs, err := ds.GetBackrefs(id)
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"refs":     refs,
		"backrefs": backrefs,
	})
}

type docsAddRefBody struct {
	RefID string `json:"refId"`
}

func (s *Server) handleDocsAddRef(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body docsAddRefBody
	if err := s.readJSON(w, r, &body); err != nil {
		return
	}
	if body.RefID == "" {
		s.jsonError(w, http.StatusBadRequest, "refId is required")
		return
	}
	ds := store.NewDocsStore(s.root)
	if err := ds.AddRef(id, body.RefID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleDocsRemoveRef(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	refID := r.PathValue("refId")
	ds := store.NewDocsStore(s.root)
	if err := ds.RemoveRef(id, refID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ── query / lint handlers ─────────────────────────────────────────────────────

func (s *Server) handleDocsQuery(w http.ResponseWriter, r *http.Request) {
	question := r.URL.Query().Get("q")
	if question == "" {
		s.jsonError(w, http.StatusBadRequest, "q parameter is required")
		return
	}
	withContent := r.URL.Query().Get("content") == "true"
	ds := store.NewDocsStore(s.root)
	results, err := ds.QueryDocs(question, withContent, 10)
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(results)
}

func (s *Server) handleDocsLint(w http.ResponseWriter, r *http.Request) {
	ds := store.NewDocsStore(s.root)
	result, err := ds.Lint()
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}
