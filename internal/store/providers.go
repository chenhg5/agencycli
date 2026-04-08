package store

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"gopkg.in/yaml.v3"
)

type ProviderStore struct {
	root string
}

func NewProviderStore(root string) *ProviderStore {
	return &ProviderStore{root: root}
}

func (ps *ProviderStore) filePath() string {
	return filepath.Join(ps.root, ".agencycli", "providers.yaml")
}

func newProviderID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return "prov-" + string(b)
}

func (ps *ProviderStore) load() ([]entity.APIProvider, error) {
	data, err := os.ReadFile(ps.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []entity.APIProvider
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (ps *ProviderStore) save(items []entity.APIProvider) error {
	data, err := yaml.Marshal(items)
	if err != nil {
		return err
	}
	dir := filepath.Dir(ps.filePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(ps.filePath(), data, 0o644)
}

func (ps *ProviderStore) List() ([]entity.APIProvider, error) {
	items, err := ps.load()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []entity.APIProvider{}
	}
	return items, nil
}

func (ps *ProviderStore) Get(id string) (*entity.APIProvider, error) {
	items, err := ps.load()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, fmt.Errorf("provider %q not found", id)
}

func (ps *ProviderStore) Add(p entity.APIProvider) (*entity.APIProvider, error) {
	items, err := ps.load()
	if err != nil {
		return nil, err
	}
	p.ID = newProviderID()
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	items = append(items, p)
	if err := ps.save(items); err != nil {
		return nil, err
	}
	return &p, nil
}

func (ps *ProviderStore) Update(id string, p entity.APIProvider) (*entity.APIProvider, error) {
	items, err := ps.load()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			p.ID = id
			items[i] = p
			if err := ps.save(items); err != nil {
				return nil, err
			}
			return &items[i], nil
		}
	}
	return nil, fmt.Errorf("provider %q not found", id)
}

func (ps *ProviderStore) Remove(id string) error {
	items, err := ps.load()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == id {
			items = append(items[:i], items[i+1:]...)
			return ps.save(items)
		}
	}
	return fmt.Errorf("provider %q not found", id)
}

// ResolveEnv returns merged environment variables for a provider.
// Maps provider type + fields to the appropriate env var names.
func (ps *ProviderStore) ResolveEnv(id string) (map[string]string, error) {
	p, err := ps.Get(id)
	if err != nil {
		return nil, err
	}
	env := make(map[string]string)
	for k, v := range p.Env {
		env[k] = v
	}
	switch p.Type {
	case "anthropic":
		if p.APIKey != "" {
			env["ANTHROPIC_API_KEY"] = p.APIKey
		}
		if p.BaseURL != "" {
			env["ANTHROPIC_BASE_URL"] = p.BaseURL
		}
		if p.Model != "" {
			env["ANTHROPIC_MODEL"] = p.Model
		}
	case "openai":
		if p.APIKey != "" {
			env["OPENAI_API_KEY"] = p.APIKey
		}
		if p.BaseURL != "" {
			env["OPENAI_BASE_URL"] = p.BaseURL
		}
		if p.Model != "" {
			env["OPENAI_MODEL"] = p.Model
		}
	case "gemini":
		if p.APIKey != "" {
			env["GEMINI_API_KEY"] = p.APIKey
		}
		if p.BaseURL != "" {
			env["GOOGLE_API_BASE"] = p.BaseURL
		}
	}
	return env, nil
}
