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
	docs, err := ds.load()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var results []*DocEntry
	for _, d := range docs {
		if strings.Contains(strings.ToLower(d.Title), q) ||
			strings.Contains(strings.ToLower(d.Description), q) ||
			strings.Contains(strings.ToLower(d.Index), q) ||
			strings.Contains(strings.ToLower(d.FilePath), q) {
			results = append(results, d)
		}
		if len(results) > 0 {
			continue
		}
		for _, tag := range d.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				results = append(results, d)
				break
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
