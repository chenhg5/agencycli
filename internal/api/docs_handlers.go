package api

import (
	"encoding/json"
	"net/http"
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
	_ = json.NewEncoder(w).Encode(entry)
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
