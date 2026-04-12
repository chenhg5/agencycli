package store

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"gopkg.in/yaml.v3"
)

type SecretStore struct {
	root string
}

func NewSecretStore(root string) *SecretStore {
	return &SecretStore{root: root}
}

func (ss *SecretStore) filePath() string {
	return filepath.Join(ss.root, ".agencycli", "secrets.yaml")
}

func newSecretID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return "sec-" + string(b)
}

func (ss *SecretStore) load() ([]entity.Secret, error) {
	data, err := os.ReadFile(ss.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []entity.Secret
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (ss *SecretStore) save(items []entity.Secret) error {
	data, err := yaml.Marshal(items)
	if err != nil {
		return err
	}
	dir := filepath.Dir(ss.filePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(ss.filePath(), data, 0o600)
}

func (ss *SecretStore) List() ([]entity.Secret, error) {
	items, err := ss.load()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []entity.Secret{}
	}
	return items, nil
}

func (ss *SecretStore) Get(id string) (*entity.Secret, error) {
	items, err := ss.load()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, fmt.Errorf("secret %q not found", id)
}

func (ss *SecretStore) Add(s entity.Secret) (*entity.Secret, error) {
	items, err := ss.load()
	if err != nil {
		return nil, err
	}
	if s.Key == "" {
		return nil, fmt.Errorf("secret key is required")
	}
	s.ID = newSecretID()
	now := time.Now().UTC()
	s.CreatedAt = now
	s.UpdatedAt = now
	if s.Scope == "" {
		s.Scope = entity.SecretScopeGlobal
	}
	items = append(items, s)
	if err := ss.save(items); err != nil {
		return nil, err
	}
	return &s, nil
}

func (ss *SecretStore) Update(id string, s entity.Secret) (*entity.Secret, error) {
	items, err := ss.load()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			s.ID = id
			s.CreatedAt = items[i].CreatedAt
			s.UpdatedAt = time.Now().UTC()
			items[i] = s
			if err := ss.save(items); err != nil {
				return nil, err
			}
			return &items[i], nil
		}
	}
	return nil, fmt.Errorf("secret %q not found", id)
}

func (ss *SecretStore) Remove(id string) error {
	items, err := ss.load()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == id {
			items = append(items[:i], items[i+1:]...)
			return ss.save(items)
		}
	}
	return fmt.Errorf("secret %q not found", id)
}

// ResolveEnvForAgent returns the merged env vars from workspace secrets that
// apply to a given agent. Global secrets are applied first, then agent-scoped
// secrets override matching keys.
func (ss *SecretStore) ResolveEnvForAgent(project, agent string) (map[string]string, error) {
	items, err := ss.load()
	if err != nil {
		return nil, err
	}
	agentID := project + "/" + agent
	env := make(map[string]string)

	// Pass 1: global secrets
	for _, s := range items {
		if s.Scope == entity.SecretScopeGlobal {
			env[s.Key] = s.Value
		}
	}
	// Pass 2: agent-scoped secrets (override globals for matching keys)
	for _, s := range items {
		if s.Scope == entity.SecretScopeAgents {
			for _, a := range s.Agents {
				if a == agentID {
					env[s.Key] = s.Value
					break
				}
			}
		}
	}
	return env, nil
}
