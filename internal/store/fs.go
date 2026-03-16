package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentorg/agentorg/internal/entity"
	"gopkg.in/yaml.v3"
)

// fsStore is the filesystem-backed implementation of Store.
type fsStore struct {
	root string
}

// NewFS creates a Store that reads and writes files under root.
func NewFS(root string) Store {
	return &fsStore{root: root}
}

func (s *fsStore) Root() string { return s.root }

// ── helpers ──────────────────────────────────────────────────────────────────

func (s *fsStore) abs(parts ...string) string {
	return filepath.Join(append([]string{s.root}, parts...)...)
}

func readYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

func writeYAML(path string, in any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(in)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeText(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// ── Company ───────────────────────────────────────────────────────────────────

func (s *fsStore) Company() (*entity.Company, error) {
	path := s.abs(".aios", "company.yaml")
	var c entity.Company
	if err := readYAML(path, &c); err != nil {
		return nil, fmt.Errorf("store: read company: %w", err)
	}
	return &c, nil
}

func (s *fsStore) SaveCompany(c *entity.Company) error {
	path := s.abs(".aios", "company.yaml")
	if err := writeYAML(path, c); err != nil {
		return fmt.Errorf("store: save company: %w", err)
	}
	return nil
}

func (s *fsStore) CompanyPrompt() (string, error) {
	content, err := readText(s.abs("company-prompt.md"))
	if err != nil {
		return "", fmt.Errorf("store: read company prompt: %w", err)
	}
	return content, nil
}

func (s *fsStore) SaveCompanyPrompt(content string) error {
	if err := writeText(s.abs("company-prompt.md"), content); err != nil {
		return fmt.Errorf("store: save company prompt: %w", err)
	}
	return nil
}

// ── Departments ───────────────────────────────────────────────────────────────

func (s *fsStore) deptDir(path string) string {
	// path is slash-separated, e.g. "engineering/backend"
	return s.abs(append([]string{"departments"}, strings.Split(path, "/")...)...)
}

func (s *fsStore) Department(path string) (*entity.Department, error) {
	yamlPath := filepath.Join(s.deptDir(path), "dept.yaml")
	var d entity.Department
	if err := readYAML(yamlPath, &d); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("store: department %q not found", path)
		}
		return nil, fmt.Errorf("store: read department %q: %w", path, err)
	}
	return &d, nil
}

func (s *fsStore) SaveDepartment(path string, d *entity.Department) error {
	yamlPath := filepath.Join(s.deptDir(path), "dept.yaml")
	if err := writeYAML(yamlPath, d); err != nil {
		return fmt.Errorf("store: save department %q: %w", path, err)
	}
	return nil
}

func (s *fsStore) DeptPrompt(path string) (string, error) {
	content, err := readText(filepath.Join(s.deptDir(path), "prompt.md"))
	if err != nil {
		return "", fmt.Errorf("store: read dept prompt %q: %w", path, err)
	}
	return content, nil
}

func (s *fsStore) SaveDeptPrompt(path string, content string) error {
	if err := writeText(filepath.Join(s.deptDir(path), "prompt.md"), content); err != nil {
		return fmt.Errorf("store: save dept prompt %q: %w", path, err)
	}
	return nil
}

// ListDepartments walks departments/ recursively and returns every directory
// that contains a dept.yaml.
func (s *fsStore) ListDepartments() ([]*DeptEntry, error) {
	base := s.abs("departments")
	var entries []*DeptEntry

	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() != "dept.yaml" {
			return nil
		}
		dir := filepath.Dir(path)
		// Convert absolute dir to relative dept path (slash-separated)
		rel, err := filepath.Rel(base, dir)
		if err != nil {
			return err
		}
		deptPath := filepath.ToSlash(rel)

		var dept entity.Department
		if err := readYAML(path, &dept); err != nil {
			return fmt.Errorf("store: read dept %q: %w", deptPath, err)
		}
		entries = append(entries, &DeptEntry{Path: deptPath, Department: &dept})
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: list departments: %w", err)
	}
	return entries, nil
}

// ── Projects ──────────────────────────────────────────────────────────────────

func (s *fsStore) projectDir(name string) string {
	return s.abs("projects", name)
}

func (s *fsStore) Project(name string) (*entity.Project, error) {
	path := filepath.Join(s.projectDir(name), "project.yaml")
	var p entity.Project
	if err := readYAML(path, &p); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("store: project %q not found", name)
		}
		return nil, fmt.Errorf("store: read project %q: %w", name, err)
	}
	return &p, nil
}

func (s *fsStore) SaveProject(name string, p *entity.Project) error {
	path := filepath.Join(s.projectDir(name), "project.yaml")
	if err := writeYAML(path, p); err != nil {
		return fmt.Errorf("store: save project %q: %w", name, err)
	}
	return nil
}

func (s *fsStore) ProjectPrompt(name string) (string, error) {
	content, err := readText(filepath.Join(s.projectDir(name), "prompt.md"))
	if err != nil {
		return "", fmt.Errorf("store: read project prompt %q: %w", name, err)
	}
	return content, nil
}

func (s *fsStore) SaveProjectPrompt(name string, content string) error {
	if err := writeText(filepath.Join(s.projectDir(name), "prompt.md"), content); err != nil {
		return fmt.Errorf("store: save project prompt %q: %w", name, err)
	}
	return nil
}

func (s *fsStore) ListProjects() ([]*entity.Project, error) {
	base := s.abs("projects")
	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: list projects: %w", err)
	}

	var projects []*entity.Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := s.Project(e.Name())
		if err != nil {
			continue // skip directories without project.yaml
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// ── Skills ────────────────────────────────────────────────────────────────────

func (s *fsStore) skillDir(name string) string {
	return s.abs("skills", name)
}

func (s *fsStore) Skill(name string) (*entity.Skill, error) {
	path := filepath.Join(s.skillDir(name), "skill.yaml")
	var sk entity.Skill
	if err := readYAML(path, &sk); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("store: skill %q not found", name)
		}
		return nil, fmt.Errorf("store: read skill %q: %w", name, err)
	}
	return &sk, nil
}

func (s *fsStore) SkillPrompt(name string) (string, error) {
	content, err := readText(filepath.Join(s.skillDir(name), "prompt.md"))
	if err != nil {
		return "", fmt.Errorf("store: read skill prompt %q: %w", name, err)
	}
	return content, nil
}

func (s *fsStore) ListSkills() ([]*entity.Skill, error) {
	base := s.abs("skills")
	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: list skills: %w", err)
	}

	var skills []*entity.Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sk, err := s.Skill(e.Name())
		if err != nil {
			continue
		}
		skills = append(skills, sk)
	}
	return skills, nil
}

// ── Agents ────────────────────────────────────────────────────────────────────

func (s *fsStore) AgentDir(project, name string) string {
	return s.abs("projects", project, "agents", name)
}

func (s *fsStore) AgentMeta(project, name string) (*entity.AgentMeta, error) {
	path := filepath.Join(s.AgentDir(project, name), ".aios-agent.yaml")
	var m entity.AgentMeta
	if err := readYAML(path, &m); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("store: agent %q/%q not found", project, name)
		}
		return nil, fmt.Errorf("store: read agent meta %q/%q: %w", project, name, err)
	}
	return &m, nil
}

func (s *fsStore) SaveAgentMeta(project, name string, meta *entity.AgentMeta) error {
	path := filepath.Join(s.AgentDir(project, name), ".aios-agent.yaml")
	if err := writeYAML(path, meta); err != nil {
		return fmt.Errorf("store: save agent meta %q/%q: %w", project, name, err)
	}
	return nil
}

func (s *fsStore) ListAgents(project string) ([]*AgentEntry, error) {
	base := s.abs("projects", project, "agents")
	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: list agents for %q: %w", project, err)
	}

	var agents []*AgentEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := s.AgentMeta(project, e.Name())
		if err != nil {
			continue
		}
		agents = append(agents, &AgentEntry{Project: project, Name: e.Name(), Meta: meta})
	}
	return agents, nil
}
