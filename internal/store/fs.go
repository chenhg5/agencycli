package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agencycli/agencycli/internal/entity"
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

// ── Agency ────────────────────────────────────────────────────────────────────

func (s *fsStore) Agency() (*entity.Agency, error) {
	path := s.abs(".agencycli", "agency.yaml")
	var a entity.Agency
	if err := readYAML(path, &a); err != nil {
		return nil, fmt.Errorf("store: read agency: %w", err)
	}
	return &a, nil
}

func (s *fsStore) SaveAgency(a *entity.Agency) error {
	path := s.abs(".agencycli", "agency.yaml")
	if err := writeYAML(path, a); err != nil {
		return fmt.Errorf("store: save agency: %w", err)
	}
	return nil
}

func (s *fsStore) AgencyPrompt() (string, error) {
	content, err := readText(s.abs("agency-prompt.md"))
	if err != nil {
		return "", fmt.Errorf("store: read agency prompt: %w", err)
	}
	return content, nil
}

func (s *fsStore) SaveAgencyPrompt(content string) error {
	if err := writeText(s.abs("agency-prompt.md"), content); err != nil {
		return fmt.Errorf("store: save agency prompt: %w", err)
	}
	return nil
}

// ── Teams ─────────────────────────────────────────────────────────────────────

func (s *fsStore) teamDir(path string) string {
	// path is slash-separated, e.g. "engineering/backend"
	return s.abs(append([]string{"teams"}, strings.Split(path, "/")...)...)
}

func (s *fsStore) Team(path string) (*entity.Team, error) {
	yamlPath := filepath.Join(s.teamDir(path), "team.yaml")
	var t entity.Team
	if err := readYAML(yamlPath, &t); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("store: team %q not found", path)
		}
		return nil, fmt.Errorf("store: read team %q: %w", path, err)
	}
	return &t, nil
}

func (s *fsStore) SaveTeam(path string, t *entity.Team) error {
	yamlPath := filepath.Join(s.teamDir(path), "team.yaml")
	if err := writeYAML(yamlPath, t); err != nil {
		return fmt.Errorf("store: save team %q: %w", path, err)
	}
	return nil
}

func (s *fsStore) TeamPrompt(path string) (string, error) {
	content, err := readText(filepath.Join(s.teamDir(path), "prompt.md"))
	if err != nil {
		return "", fmt.Errorf("store: read team prompt %q: %w", path, err)
	}
	return content, nil
}

func (s *fsStore) SaveTeamPrompt(path string, content string) error {
	if err := writeText(filepath.Join(s.teamDir(path), "prompt.md"), content); err != nil {
		return fmt.Errorf("store: save team prompt %q: %w", path, err)
	}
	return nil
}

// ListTeams walks teams/ recursively and returns every directory
// that contains a team.yaml.
func (s *fsStore) ListTeams() ([]*TeamEntry, error) {
	base := s.abs("teams")
	var entries []*TeamEntry

	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() != "team.yaml" {
			return nil
		}
		dir := filepath.Dir(path)
		// Convert absolute dir to relative team path (slash-separated)
		rel, err := filepath.Rel(base, dir)
		if err != nil {
			return err
		}
		teamPath := filepath.ToSlash(rel)

		var team entity.Team
		if err := readYAML(path, &team); err != nil {
			return fmt.Errorf("store: read team %q: %w", teamPath, err)
		}
		entries = append(entries, &TeamEntry{Path: teamPath, Team: &team})
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: list teams: %w", err)
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
	path := filepath.Join(s.AgentDir(project, name), ".agencycli-agent.yaml")
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
	path := filepath.Join(s.AgentDir(project, name), ".agencycli-agent.yaml")
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
