package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"gopkg.in/yaml.v3"
)

// MilestoneStore manages project-level milestones stored in
// projects/<project>/.agencycli/milestones.yaml.
type MilestoneStore struct {
	root string
}

func NewMilestoneStore(root string) *MilestoneStore {
	return &MilestoneStore{root: root}
}

func (s *MilestoneStore) filePath(project string) string {
	return filepath.Join(s.root, "projects", project, ".agencycli", "milestones.yaml")
}

func (s *MilestoneStore) load(project string) (*entity.MilestoneFile, error) {
	data, err := os.ReadFile(s.filePath(project))
	if err != nil {
		if os.IsNotExist(err) {
			return &entity.MilestoneFile{}, nil
		}
		return nil, err
	}
	var f entity.MilestoneFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *MilestoneStore) save(project string, f *entity.MilestoneFile) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.filePath(project))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.filePath(project), data, 0o644)
}

func (s *MilestoneStore) List(project string) ([]entity.Milestone, error) {
	f, err := s.load(project)
	if err != nil {
		return nil, err
	}
	return f.Milestones, nil
}

func (s *MilestoneStore) Get(project, id string) (*entity.Milestone, error) {
	f, err := s.load(project)
	if err != nil {
		return nil, err
	}
	for i := range f.Milestones {
		if f.Milestones[i].ID == id {
			return &f.Milestones[i], nil
		}
	}
	return nil, fmt.Errorf("milestone %q not found in project %q", id, project)
}

func (s *MilestoneStore) Create(project string, ms entity.Milestone) (*entity.Milestone, error) {
	f, err := s.load(project)
	if err != nil {
		return nil, err
	}
	if ms.ID == "" {
		ms.ID = entity.NewMilestoneID()
	}
	now := time.Now().UTC()
	ms.CreatedAt = now
	ms.UpdatedAt = now
	if ms.Status == "" {
		ms.Status = entity.MilestoneStatusPlanned
	}
	f.Milestones = append(f.Milestones, ms)
	if err := s.save(project, f); err != nil {
		return nil, err
	}
	return &ms, nil
}

func (s *MilestoneStore) Update(project, id string, fn func(*entity.Milestone)) error {
	f, err := s.load(project)
	if err != nil {
		return err
	}
	for i := range f.Milestones {
		if f.Milestones[i].ID == id {
			fn(&f.Milestones[i])
			f.Milestones[i].UpdatedAt = time.Now().UTC()
			return s.save(project, f)
		}
	}
	return fmt.Errorf("milestone %q not found in project %q", id, project)
}

func (s *MilestoneStore) Delete(project, id string) error {
	f, err := s.load(project)
	if err != nil {
		return err
	}
	for i := range f.Milestones {
		if f.Milestones[i].ID == id {
			f.Milestones = append(f.Milestones[:i], f.Milestones[i+1:]...)
			return s.save(project, f)
		}
	}
	return fmt.Errorf("milestone %q not found in project %q", id, project)
}
