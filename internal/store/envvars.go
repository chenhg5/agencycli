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

type EnvVarStore struct {
	root string
}

func NewEnvVarStore(root string) *EnvVarStore {
	return &EnvVarStore{root: root}
}

func (es *EnvVarStore) filePath() string {
	return filepath.Join(es.root, ".agencycli", "envvars.yaml")
}

func newEnvVarID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return "ev-" + string(b)
}

func (es *EnvVarStore) load() ([]entity.EnvVar, error) {
	data, err := os.ReadFile(es.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []entity.EnvVar
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (es *EnvVarStore) save(items []entity.EnvVar) error {
	data, err := yaml.Marshal(items)
	if err != nil {
		return err
	}
	dir := filepath.Dir(es.filePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(es.filePath(), data, 0o600)
}

func (es *EnvVarStore) List() ([]entity.EnvVar, error) {
	items, err := es.load()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []entity.EnvVar{}
	}
	return items, nil
}

func (es *EnvVarStore) Get(id string) (*entity.EnvVar, error) {
	items, err := es.load()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, fmt.Errorf("envvar %q not found", id)
}

func (es *EnvVarStore) Add(ev entity.EnvVar) (*entity.EnvVar, error) {
	items, err := es.load()
	if err != nil {
		return nil, err
	}
	if ev.Key == "" {
		return nil, fmt.Errorf("envvar key is required")
	}
	ev.ID = newEnvVarID()
	now := time.Now().UTC()
	ev.CreatedAt = now
	ev.UpdatedAt = now
	if ev.Scope == "" {
		ev.Scope = entity.EnvVarScopeGlobal
	}
	items = append(items, ev)
	if err := es.save(items); err != nil {
		return nil, err
	}
	return &ev, nil
}

func (es *EnvVarStore) Update(id string, ev entity.EnvVar) (*entity.EnvVar, error) {
	items, err := es.load()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			ev.ID = id
			ev.CreatedAt = items[i].CreatedAt
			ev.UpdatedAt = time.Now().UTC()
			items[i] = ev
			if err := es.save(items); err != nil {
				return nil, err
			}
			return &items[i], nil
		}
	}
	return nil, fmt.Errorf("envvar %q not found", id)
}

func (es *EnvVarStore) Remove(id string) error {
	items, err := es.load()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == id {
			items = append(items[:i], items[i+1:]...)
			return es.save(items)
		}
	}
	return fmt.Errorf("envvar %q not found", id)
}

// ResolveEnvForAgent returns the merged env vars from workspace-level variables
// that apply to a given agent. Global variables are applied first, then
// agent-scoped variables override matching keys.
func (es *EnvVarStore) ResolveEnvForAgent(project, agent string) (map[string]string, error) {
	items, err := es.load()
	if err != nil {
		return nil, err
	}
	agentID := project + "/" + agent
	env := make(map[string]string)

	for _, v := range items {
		if v.Scope == entity.EnvVarScopeGlobal {
			env[v.Key] = v.Value
		}
	}
	for _, v := range items {
		if v.Scope == entity.EnvVarScopeAgents {
			for _, a := range v.Agents {
				if a == agentID {
					env[v.Key] = v.Value
					break
				}
			}
		}
	}
	return env, nil
}
