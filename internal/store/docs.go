package store

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type DocEntry struct {
	ID          string    `yaml:"id" json:"id"`
	Title       string    `yaml:"title" json:"title"`
	FilePath    string    `yaml:"file_path" json:"filePath"`
	Index       string    `yaml:"index" json:"index"`
	CreatedBy   string    `yaml:"created_by" json:"createdBy"`
	Tags        []string  `yaml:"tags,omitempty" json:"tags,omitempty"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time `yaml:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `yaml:"updated_at" json:"updatedAt"`
}

type DocsStore struct {
	root string
}

func NewDocsStore(root string) *DocsStore {
	return &DocsStore{root: root}
}

func (ds *DocsStore) filePath() string {
	return filepath.Join(ds.root, ".agencycli", "docs.yaml")
}

func newDocID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("doc-%s-%s", time.Now().UTC().Format("20060102"), string(b))
}

func (ds *DocsStore) load() ([]*DocEntry, error) {
	data, err := os.ReadFile(ds.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var docs []*DocEntry
	if err := yaml.Unmarshal(data, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (ds *DocsStore) save(docs []*DocEntry) error {
	fp := ds.filePath()
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(docs)
	if err != nil {
		return err
	}
	return os.WriteFile(fp, data, 0o644)
}

func (ds *DocsStore) Add(e *DocEntry) error {
	docs, err := ds.load()
	if err != nil {
		return err
	}
	if e.ID == "" {
		e.ID = newDocID()
	}
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now
	docs = append(docs, e)
	return ds.save(docs)
}

func (ds *DocsStore) List() ([]*DocEntry, error) {
	return ds.load()
}

func (ds *DocsStore) Get(id string) (*DocEntry, error) {
	docs, err := ds.load()
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, fmt.Errorf("document %q not found", id)
}

func (ds *DocsStore) Update(id string, fn func(e *DocEntry)) error {
	docs, err := ds.load()
	if err != nil {
		return err
	}
	for _, d := range docs {
		if d.ID == id {
			fn(d)
			d.UpdatedAt = time.Now().UTC()
			return ds.save(docs)
		}
	}
	return fmt.Errorf("document %q not found", id)
}

func (ds *DocsStore) Remove(id string) error {
	docs, err := ds.load()
	if err != nil {
		return err
	}
	out := make([]*DocEntry, 0, len(docs))
	found := false
	for _, d := range docs {
		if d.ID == id {
			found = true
			continue
		}
		out = append(out, d)
	}
	if !found {
		return fmt.Errorf("document %q not found", id)
	}
	return ds.save(out)
}

func (ds *DocsStore) Search(query string) ([]*DocEntry, error) {
	return ds.SearchOpts(query, false)
}

// SearchOpts searches documents. When withContent is true, file contents are
// also scanned for the query string.
func (ds *DocsStore) SearchOpts(query string, withContent bool) ([]*DocEntry, error) {
	docs, err := ds.load()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	seen := map[string]bool{}
	var results []*DocEntry
	addOnce := func(d *DocEntry) {
		if !seen[d.ID] {
			seen[d.ID] = true
			results = append(results, d)
		}
	}
	for _, d := range docs {
		if strings.Contains(strings.ToLower(d.Title), q) ||
			strings.Contains(strings.ToLower(d.Description), q) ||
			strings.Contains(strings.ToLower(d.Index), q) ||
			strings.Contains(strings.ToLower(d.FilePath), q) {
			addOnce(d)
			continue
		}
		for _, tag := range d.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				addOnce(d)
				break
			}
		}
		if withContent && !seen[d.ID] {
			content, err := ds.ReadContent(d.FilePath)
			if err == nil && strings.Contains(strings.ToLower(content), q) {
				addOnce(d)
			}
		}
	}
	return results, nil
}

type TreeNode struct {
	Name     string      `json:"name"`
	Children []*TreeNode `json:"children,omitempty"`
	Docs     []*DocEntry `json:"docs,omitempty"`
}

func (ds *DocsStore) Tree() (*TreeNode, error) {
	docs, err := ds.load()
	if err != nil {
		return nil, err
	}
	root := &TreeNode{Name: "/"}
	for _, d := range docs {
		parts := strings.Split(strings.Trim(d.Index, "/"), "/")
		if len(parts) == 1 && parts[0] == "" {
			root.Docs = append(root.Docs, d)
			continue
		}
		node := root
		for _, p := range parts {
			found := false
			for _, c := range node.Children {
				if c.Name == p {
					node = c
					found = true
					break
				}
			}
			if !found {
				child := &TreeNode{Name: p}
				node.Children = append(node.Children, child)
				node = child
			}
		}
		node.Docs = append(node.Docs, d)
	}
	sortTree(root)
	return root, nil
}

func sortTree(n *TreeNode) {
	sort.Slice(n.Children, func(i, j int) bool {
		return n.Children[i].Name < n.Children[j].Name
	})
	sort.Slice(n.Docs, func(i, j int) bool {
		return n.Docs[i].Title < n.Docs[j].Title
	})
	for _, c := range n.Children {
		sortTree(c)
	}
}

func (ds *DocsStore) ReadContent(filePath string) (string, error) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(ds.root, filePath)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ── Wiki layer ────────────────────────────────────────────────────────────────
// The wiki is a directory of LLM-generated markdown files that sit between raw
// sources and agents. Raw source files are never modified; the wiki is owned
// entirely by agents. Following the Karpathy LLM-wiki pattern.

// WikiDir returns the path to the wiki directory.
func (ds *DocsStore) WikiDir() string {
	return filepath.Join(ds.root, ".agencycli", "wiki")
}

// WikiIndexPath returns the path to the wiki's index.md catalog.
func (ds *DocsStore) WikiIndexPath() string {
	return filepath.Join(ds.WikiDir(), "index.md")
}

// WikiLogPath returns the path to the wiki's append-only activity log.
func (ds *DocsStore) WikiLogPath() string {
	return filepath.Join(ds.WikiDir(), "log.md")
}

// WikiPagePath returns the path for a wiki page given a slug.
func (ds *DocsStore) WikiPagePath(slug string) string {
	if !strings.HasSuffix(slug, ".md") {
		slug += ".md"
	}
	return filepath.Join(ds.WikiDir(), slug)
}

// EnsureWikiDir creates the wiki directory if it does not exist.
func (ds *DocsStore) EnsureWikiDir() error {
	return os.MkdirAll(ds.WikiDir(), 0o755)
}

type WikiPage struct {
	Slug    string    `json:"slug"`
	Path    string    `json:"path"`
	Title   string    `json:"title"`
	Summary string    `json:"summary"` // first non-heading paragraph
	ModTime time.Time `json:"modTime"`
	Size    int64     `json:"size"`
}

// ListWikiPages returns all .md files in the wiki directory (excluding index.md
// and log.md which are special files).
func (ds *DocsStore) ListWikiPages() ([]*WikiPage, error) {
	dir := ds.WikiDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var pages []*WikiPage
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := e.Name()
		if name == "index.md" || name == "log.md" {
			continue
		}
		info, _ := e.Info()
		slug := strings.TrimSuffix(name, ".md")
		p := &WikiPage{
			Slug: slug,
			Path: filepath.Join(dir, name),
		}
		if info != nil {
			p.ModTime = info.ModTime()
			p.Size = info.Size()
		}
		// Extract title and summary from content
		if data, err := os.ReadFile(p.Path); err == nil {
			p.Title, p.Summary = wikiPageMeta(string(data))
		}
		if p.Title == "" {
			p.Title = slug
		}
		pages = append(pages, p)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Slug < pages[j].Slug })
	return pages, nil
}

// wikiPageMeta extracts the first H1 heading and first non-empty, non-heading
// paragraph from a markdown page.
func wikiPageMeta(content string) (title, summary string) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if title == "" && strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			continue
		}
		if summary == "" && line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") {
			if len(line) > 200 {
				line = line[:197] + "..."
			}
			summary = line
		}
		if title != "" && summary != "" {
			break
		}
	}
	return
}

// WriteWikiPage writes content to a wiki page, creating the wiki directory if
// needed.
func (ds *DocsStore) WriteWikiPage(slug, content string) error {
	if err := ds.EnsureWikiDir(); err != nil {
		return err
	}
	return os.WriteFile(ds.WikiPagePath(slug), []byte(content), 0o644)
}

// ReadWikiPage reads a wiki page by slug.
func (ds *DocsStore) ReadWikiPage(slug string) (string, error) {
	data, err := os.ReadFile(ds.WikiPagePath(slug))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// AppendWikiLog appends an entry to log.md.
func (ds *DocsStore) AppendWikiLog(operation, title, details string) error {
	if err := ds.EnsureWikiDir(); err != nil {
		return err
	}
	entry := fmt.Sprintf("## [%s] %s | %s\n",
		time.Now().UTC().Format("2006-01-02 15:04"),
		operation,
		title,
	)
	if details != "" {
		entry += details + "\n"
	}
	entry += "\n"
	f, err := os.OpenFile(ds.WikiLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return err
}

// RebuildWikiIndex rewrites index.md from all current wiki pages + docs.
func (ds *DocsStore) RebuildWikiIndex() error {
	if err := ds.EnsureWikiDir(); err != nil {
		return err
	}
	pages, err := ds.ListWikiPages()
	if err != nil {
		return err
	}
	docs, err := ds.load()
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("# Knowledge Base Index\n\n")
	sb.WriteString(fmt.Sprintf("*Last updated: %s — %d wiki pages, %d sources*\n\n",
		time.Now().UTC().Format("2006-01-02 15:04 UTC"), len(pages), len(docs)))

	if len(pages) > 0 {
		sb.WriteString("## Wiki Pages\n\n")
		for _, p := range pages {
			summary := p.Summary
			if summary == "" {
				summary = "*(no summary)*"
			}
			sb.WriteString(fmt.Sprintf("- **[%s](%s.md)** — %s\n", p.Title, p.Slug, summary))
		}
		sb.WriteString("\n")
	}

	if len(docs) > 0 {
		sb.WriteString("## Source Documents\n\n")
		for _, d := range docs {
			desc := d.Description
			if desc == "" {
				desc = "*(no description)*"
			}
			index := d.Index
			if index == "" {
				index = "/"
			}
			sb.WriteString(fmt.Sprintf("- **%s** (`%s`) — %s\n", d.Title, index, desc))
			sb.WriteString(fmt.Sprintf("  - ID: `%s`  Path: `%s`\n", d.ID, d.FilePath))
		}
		sb.WriteString("\n")
	}

	return os.WriteFile(ds.WikiIndexPath(), []byte(sb.String()), 0o644)
}

// LintWikiResult holds the results of a wiki health check.
type LintWikiResult struct {
	OrphanWikiPages    []string // pages in wiki/ not linked anywhere
	DocsWithoutWiki    []string // doc IDs that have no wiki summary page
	EmptyWikiPages     []string // wiki pages with < 50 bytes of content
	DocsWithoutDesc    []string // doc IDs with empty description
	TotalDocs          int
	TotalWikiPages     int
}

// LintWiki performs a health check on the wiki and returns issues found.
func (ds *DocsStore) LintWiki() (*LintWikiResult, error) {
	res := &LintWikiResult{}

	docs, err := ds.load()
	if err != nil {
		return nil, err
	}
	res.TotalDocs = len(docs)

	pages, err := ds.ListWikiPages()
	if err != nil {
		return nil, err
	}
	res.TotalWikiPages = len(pages)

	// Docs without description
	for _, d := range docs {
		if strings.TrimSpace(d.Description) == "" {
			res.DocsWithoutDesc = append(res.DocsWithoutDesc, d.ID+" ("+d.Title+")")
		}
	}

	// Build set of wiki slugs
	wikiSlugs := map[string]bool{}
	for _, p := range pages {
		wikiSlugs[p.Slug] = true
	}

	// Docs without any associated wiki page (check if a slug matching title/id exists)
	for _, d := range docs {
		slug := docSlug(d.Title)
		slugID := docSlug(d.ID)
		if !wikiSlugs[slug] && !wikiSlugs[slugID] && !wikiSlugs[d.ID] {
			res.DocsWithoutWiki = append(res.DocsWithoutWiki, d.ID+" ("+d.Title+")")
		}
	}

	// Empty wiki pages
	for _, p := range pages {
		if p.Size < 50 {
			res.EmptyWikiPages = append(res.EmptyWikiPages, p.Slug)
		}
	}

	// Orphan wiki pages: check if they appear in index.md
	indexContent := ""
	if data, err := os.ReadFile(ds.WikiIndexPath()); err == nil {
		indexContent = string(data)
	}
	for _, p := range pages {
		if !strings.Contains(indexContent, p.Slug) && !strings.Contains(indexContent, p.Title) {
			res.OrphanWikiPages = append(res.OrphanWikiPages, p.Slug)
		}
	}

	return res, nil
}

// docSlug converts a title or ID into a filesystem-safe slug.
func docSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}

// DocSlug is the exported version for use in commands.
func DocSlug(s string) string { return docSlug(s) }

// WikiStub generates a starter wiki page for a document entry, ready for an
// agent to fill in.
func WikiStub(e *DocEntry) string {
	title := e.Title
	slug := docSlug(title)
	_ = slug
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	sb.WriteString(fmt.Sprintf("**Source:** `%s`  \n", e.FilePath))
	sb.WriteString(fmt.Sprintf("**Index:** `%s`  \n", e.Index))
	if len(e.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("**Tags:** %s  \n", strings.Join(e.Tags, ", ")))
	}
	sb.WriteString(fmt.Sprintf("**Added:** %s  \n\n", e.CreatedAt.UTC().Format("2006-01-02")))
	sb.WriteString("---\n\n")
	sb.WriteString("## Summary\n\n*(Agent: read the source and write a 2–5 sentence summary here.)*\n\n")
	sb.WriteString("## Key Points\n\n*(Agent: extract the most important points as bullet items.)*\n\n")
	sb.WriteString("## Related Pages\n\n*(Agent: link to related wiki pages once you have read index.md.)*\n\n")
	sb.WriteString("## Notes\n\n*(Agent: any additional observations, caveats, or open questions.)*\n")
	return sb.String()
}
