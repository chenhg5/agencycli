package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"gopkg.in/yaml.v3"
)

// OKRStore manages OKRs stored in .agencycli/okrs.yaml.
// All scopes (agency / project / agent) live in the same file.
type OKRStore struct {
	root string
}

func NewOKRStore(root string) *OKRStore {
	return &OKRStore{root: root}
}

func (s *OKRStore) filePath() string {
	return filepath.Join(s.root, ".agencycli", "okrs.yaml")
}

func (s *OKRStore) load() (*entity.OKRFile, error) {
	data, err := os.ReadFile(s.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &entity.OKRFile{}, nil
		}
		return nil, err
	}
	var f entity.OKRFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *OKRStore) save(f *entity.OKRFile) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.filePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.filePath(), data, 0o644)
}

func (s *OKRStore) Load() (*entity.OKRFile, error) {
	return s.load()
}

// ListOKRs returns all OKRs, optionally filtered by scope and scopeRef.
func (s *OKRStore) ListOKRs(scope entity.OKRScope, scopeRef string) ([]entity.OKR, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	if scope == "" {
		return f.OKRs, nil
	}
	var out []entity.OKR
	for _, o := range f.OKRs {
		s := o.Scope
		if s == "" {
			s = entity.OKRScopeAgency
		}
		if s == scope && (scopeRef == "" || o.ScopeRef == scopeRef) {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *OKRStore) GetOKR(id string) (*entity.OKR, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range f.OKRs {
		if f.OKRs[i].ID == id {
			return &f.OKRs[i], nil
		}
	}
	return nil, fmt.Errorf("OKR %q not found", id)
}

func (s *OKRStore) CreateOKR(okr entity.OKR) (*entity.OKR, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	if okr.ID == "" {
		okr.ID = entity.NewOKRID()
	}
	now := time.Now().UTC()
	okr.CreatedAt = now
	okr.UpdatedAt = now
	if okr.Status == "" {
		okr.Status = entity.OKRStatusOnTrack
	}
	if okr.Scope == "" {
		okr.Scope = entity.OKRScopeAgency
	}
	f.OKRs = append(f.OKRs, okr)
	if err := s.save(f); err != nil {
		return nil, err
	}
	return &okr, nil
}

func (s *OKRStore) UpdateOKR(id string, fn func(*entity.OKR)) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	for i := range f.OKRs {
		if f.OKRs[i].ID == id {
			fn(&f.OKRs[i])
			f.OKRs[i].UpdatedAt = time.Now().UTC()
			return s.save(f)
		}
	}
	return fmt.Errorf("OKR %q not found", id)
}

func (s *OKRStore) DeleteOKR(id string) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	for i := range f.OKRs {
		if f.OKRs[i].ID == id {
			f.OKRs = append(f.OKRs[:i], f.OKRs[i+1:]...)
			return s.save(f)
		}
	}
	return fmt.Errorf("OKR %q not found", id)
}

func (s *OKRStore) SetCurrentQuarter(q string) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	f.CurrentQuarter = q
	return s.save(f)
}

func (s *OKRStore) AddKR(okrID string, kr entity.KeyResult) (*entity.KeyResult, error) {
	if kr.ID == "" {
		kr.ID = entity.NewKRID()
	}
	err := s.UpdateOKR(okrID, func(o *entity.OKR) {
		o.KeyResults = append(o.KeyResults, kr)
	})
	if err != nil {
		return nil, err
	}
	return &kr, nil
}

func (s *OKRStore) UpdateKR(okrID, krID string, fn func(*entity.KeyResult)) error {
	return s.UpdateOKR(okrID, func(o *entity.OKR) {
		for i := range o.KeyResults {
			if o.KeyResults[i].ID == krID {
				fn(&o.KeyResults[i])
				return
			}
		}
	})
}
