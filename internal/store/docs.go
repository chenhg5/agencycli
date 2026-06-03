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

// QueryResult is one document match returned by QueryDocs.
type QueryResult struct {
	*DocEntry
	Score   int    `json:"score"`
	Snippet string `json:"snippet,omitempty"` // first matching excerpt
}

// QueryDocs finds documents relevant to a natural-language question by scoring
// keyword overlap across title, description, tags, index path, and (optionally)
// file contents. Returns results sorted by score descending.
func (ds *DocsStore) QueryDocs(question string, withContent bool, maxResults int) ([]*QueryResult, error) {
	docs, err := ds.load()
	if err != nil {
		return nil, err
	}

	words := tokenise(question)
	var results []*QueryResult
	for _, d := range docs {
		score := 0
		haystack := strings.ToLower(d.Title + " " + d.Description + " " + d.Index)
		for _, t := range d.Tags {
			haystack += " " + strings.ToLower(t)
		}
		snippet := ""
		for _, w := range words {
			if strings.Contains(haystack, w) {
				score += 2 // metadata match is worth more
			}
		}
		if withContent || score == 0 {
			content, err := ds.ReadContent(d.FilePath)
			if err == nil {
				lower := strings.ToLower(content)
				for _, w := range words {
					if strings.Contains(lower, w) {
						score++
						if snippet == "" {
							snippet = extractSnippet(content, w, 160)
						}
					}
				}
			}
		}
		if score > 0 {
			results = append(results, &QueryResult{DocEntry: d, Score: score, Snippet: snippet})
		}
	}
	// Sort by score descending
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}
	return results, nil
}

// LintResult holds the results of a knowledge base health check.
type LintResult struct {
	DocsWithoutDesc []string `json:"docsWithoutDesc"` // IDs of docs with no description
	TotalDocs       int      `json:"totalDocs"`
}

// Lint checks the knowledge base for maintenance issues.
func (ds *DocsStore) Lint() (*LintResult, error) {
	docs, err := ds.load()
	if err != nil {
		return nil, err
	}
	res := &LintResult{TotalDocs: len(docs)}
	for _, d := range docs {
		if strings.TrimSpace(d.Description) == "" {
			res.DocsWithoutDesc = append(res.DocsWithoutDesc, d.ID+" ("+d.Title+")")
		}
	}
	return res, nil
}

// tokenise splits a query into lowercase words longer than 2 characters,
// ignoring common stop words.
func tokenise(query string) []string {
	stop := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "was": true,
		"what": true, "how": true, "why": true, "who": true, "when": true,
		"this": true, "that": true, "with": true, "from": true,
	}
	var words []string
	seen := map[string]bool{}
	for _, f := range strings.Fields(strings.ToLower(query)) {
		// strip punctuation
		f = strings.Trim(f, ".,?!;:\"'()[]")
		if len(f) > 2 && !stop[f] && !seen[f] {
			seen[f] = true
			words = append(words, f)
		}
	}
	return words
}

// extractSnippet returns a short excerpt of content around the first occurrence
// of word, up to maxLen characters.
func extractSnippet(content, word string, maxLen int) string {
	lower := strings.ToLower(content)
	idx := strings.Index(lower, word)
	if idx < 0 {
		return ""
	}
	start := idx - 60
	if start < 0 {
		start = 0
	}
	end := idx + maxLen
	if end > len(content) {
		end = len(content)
	}
	snippet := strings.TrimSpace(content[start:end])
	// trim to line boundaries where possible
	if nl := strings.Index(snippet, "\n"); nl > 0 && nl < 40 {
		snippet = snippet[nl+1:]
	}
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	if len(snippet) > maxLen {
		snippet = snippet[:maxLen-3] + "..."
	}
	return snippet
}
