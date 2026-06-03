package api

import (
	"encoding/json"
	"fmt"
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

// ── wiki/ingest/query/lint handlers ──────────────────────────────────────────

type docsIngestBody struct {
	FilePath    string   `json:"filePath"`
	Title       string   `json:"title"`
	Index       string   `json:"index"`
	CreatedBy   string   `json:"createdBy"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	NoWiki      bool     `json:"noWiki"`
}

func (s *Server) handleDocsIngest(w http.ResponseWriter, r *http.Request) {
	var body docsIngestBody
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
	title := body.Title
	if title == "" {
		parts := strings.Split(body.FilePath, "/")
		title = parts[len(parts)-1]
	}
	entry := &store.DocEntry{
		Title:       title,
		FilePath:    body.FilePath,
		Index:       strings.Trim(body.Index, "/"),
		CreatedBy:   body.CreatedBy,
		Tags:        body.Tags,
		Description: body.Description,
	}
	if err := ds.Add(entry); err != nil {
		s.serverError(w, err)
		return
	}
	var wikiStubPath string
	if !body.NoWiki {
		slug := store.DocSlug(title)
		stubPath := ds.WikiPagePath(slug)
		if _, err := os.Stat(stubPath); err != nil {
			stub := store.WikiStub(entry)
			if err := ds.WriteWikiPage(slug, stub); err == nil {
				wikiStubPath = stubPath
			}
		} else {
			wikiStubPath = stubPath
		}
	}
	_ = ds.RebuildWikiIndex()
	_ = ds.AppendWikiLog("ingest", title, fmt.Sprintf("  source: %s\n  doc-id: %s", body.FilePath, entry.ID))

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"doc":          docsEntryResponse(r, entry),
		"wikiStubPath": wikiStubPath,
		"indexPath":    ds.WikiIndexPath(),
		"logPath":      ds.WikiLogPath(),
	})
}

func (s *Server) handleDocsQuery(w http.ResponseWriter, r *http.Request) {
	question := r.URL.Query().Get("q")
	if question == "" {
		s.jsonError(w, http.StatusBadRequest, "q parameter is required")
		return
	}
	ds := store.NewDocsStore(s.root)

	var indexContent string
	if data, err := os.ReadFile(ds.WikiIndexPath()); err == nil {
		indexContent = string(data)
	}

	pages, _ := ds.ListWikiPages()
	q := strings.ToLower(question)
	words := strings.Fields(q)
	type scoredPage struct {
		*store.WikiPage
		Score   int    `json:"score"`
		Content string `json:"content"`
	}
	var matched []scoredPage
	for _, p := range pages {
		score := 0
		haystack := strings.ToLower(p.Title + " " + p.Summary + " " + p.Slug)
		for _, w := range words {
			if len(w) > 2 && strings.Contains(haystack, w) {
				score++
			}
		}
		if score > 0 {
			content, _ := ds.ReadWikiPage(p.Slug)
			matched = append(matched, scoredPage{WikiPage: p, Score: score, Content: content})
		}
	}

	_ = ds.AppendWikiLog("query", question, "")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"question":      question,
		"indexContent":  indexContent,
		"relevantPages": matched,
	})
}

func (s *Server) handleDocsLint(w http.ResponseWriter, r *http.Request) {
	ds := store.NewDocsStore(s.root)
	result, err := ds.LintWiki()
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

// ── wiki CRUD handlers ────────────────────────────────────────────────────────

func (s *Server) handleWikiList(w http.ResponseWriter, r *http.Request) {
	ds := store.NewDocsStore(s.root)
	pages, err := ds.ListWikiPages()
	if err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(pages)
}

func (s *Server) handleWikiShow(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	ds := store.NewDocsStore(s.root)
	content, err := ds.ReadWikiPage(slug)
	if err != nil {
		s.jsonError(w, http.StatusNotFound, "wiki page not found: "+slug)
		return
	}
	pages, _ := ds.ListWikiPages()
	var meta *store.WikiPage
	for _, p := range pages {
		if p.Slug == slug {
			meta = p
			break
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"slug":    slug,
		"content": content,
		"meta":    meta,
	})
}

type wikiWriteBody struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Append  bool   `json:"append"`
}

func (s *Server) handleWikiWrite(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var body wikiWriteBody
	if err := s.readJSON(w, r, &body); err != nil {
		return
	}
	if body.Content == "" {
		s.jsonError(w, http.StatusBadRequest, "content is required")
		return
	}
	ds := store.NewDocsStore(s.root)
	content := body.Content
	if body.Append {
		existing, _ := ds.ReadWikiPage(slug)
		content = existing + "\n" + content
	}
	if body.Title != "" && !strings.HasPrefix(strings.TrimSpace(content), "# ") {
		content = "# " + body.Title + "\n\n" + content
	}
	if err := ds.WriteWikiPage(slug, content); err != nil {
		s.serverError(w, err)
		return
	}
	_ = ds.AppendWikiLog("write", slug, "  via API")
	_ = ds.RebuildWikiIndex()
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "slug": slug, "path": ds.WikiPagePath(slug)})
}

func (s *Server) handleWikiRebuildIndex(w http.ResponseWriter, r *http.Request) {
	ds := store.NewDocsStore(s.root)
	if err := ds.RebuildWikiIndex(); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "indexPath": ds.WikiIndexPath()})
}
