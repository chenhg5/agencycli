package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
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

// ── Roles ─────────────────────────────────────────────────────────────────────

func (s *fsStore) RoleDir(teamPath, roleName string) string {
	return filepath.Join(s.teamDir(teamPath), "roles", roleName)
}

func (s *fsStore) Role(teamPath, roleName string) (*entity.Role, error) {
	path := filepath.Join(s.RoleDir(teamPath, roleName), "role.yaml")
	var r entity.Role
	if err := readYAML(path, &r); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("store: role %q/%q not found", teamPath, roleName)
		}
		return nil, fmt.Errorf("store: read role %q/%q: %w", teamPath, roleName, err)
	}
	return &r, nil
}

func (s *fsStore) SaveRole(teamPath, roleName string, r *entity.Role) error {
	path := filepath.Join(s.RoleDir(teamPath, roleName), "role.yaml")
	if err := writeYAML(path, r); err != nil {
		return fmt.Errorf("store: save role %q/%q: %w", teamPath, roleName, err)
	}
	return nil
}

func (s *fsStore) RolePrompt(teamPath, roleName string) (string, error) {
	content, err := readText(filepath.Join(s.RoleDir(teamPath, roleName), "prompt.md"))
	if err != nil {
		return "", fmt.Errorf("store: read role prompt %q/%q: %w", teamPath, roleName, err)
	}
	return content, nil
}

func (s *fsStore) SaveRolePrompt(teamPath, roleName string, content string) error {
	if err := writeText(filepath.Join(s.RoleDir(teamPath, roleName), "prompt.md"), content); err != nil {
		return fmt.Errorf("store: save role prompt %q/%q: %w", teamPath, roleName, err)
	}
	return nil
}

func (s *fsStore) ListRoles(teamPath string) ([]*RoleEntry, error) {
	base := filepath.Join(s.teamDir(teamPath), "roles")
	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: list roles for team %q: %w", teamPath, err)
	}
	var roles []*RoleEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		r, err := s.Role(teamPath, e.Name())
		if err != nil {
			continue
		}
		roles = append(roles, &RoleEntry{TeamPath: teamPath, Name: e.Name(), Role: r})
	}
	return roles, nil
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

func (s *fsStore) ProjectConfig(name string) (*entity.ProjectConfig, error) {
	path := filepath.Join(s.projectDir(name), "project.yaml")
	var cfg entity.ProjectConfig
	if err := readYAML(path, &cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("store: read project config %q: %w", name, err)
	}
	return &cfg, nil
}

// ── Skills ────────────────────────────────────────────────────────────────────

func (s *fsStore) skillDir(name string) string {
	return s.abs("skills", name)
}

func (s *fsStore) SkillDir(name string) string {
	return s.skillDir(name)
}

func (s *fsStore) Skill(name string) (*entity.Skill, error) {
	skillMD := filepath.Join(s.skillDir(name), "SKILL.md")
	sk, _, err := parseSkillMD(skillMD)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("store: skill %q not found", name)
		}
		return nil, fmt.Errorf("store: read skill %q: %w", name, err)
	}
	if sk.Name == "" {
		sk.Name = name
	}
	return sk, nil
}

func (s *fsStore) SkillPrompt(name string) (string, error) {
	skillMD := filepath.Join(s.skillDir(name), "SKILL.md")
	_, body, err := parseSkillMD(skillMD)
	if err != nil {
		return "", fmt.Errorf("store: read skill prompt %q: %w", name, err)
	}
	return body, nil
}

// parseSkillMD reads a SKILL.md file and splits it into the YAML frontmatter
// (parsed into entity.Skill) and the Markdown body that follows.
// Frontmatter is optional — if the file does not start with "---" the entire
// content is treated as the body and an empty Skill is returned.
func parseSkillMD(path string) (*entity.Skill, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	content := string(raw)
	var sk entity.Skill

	if !strings.HasPrefix(content, "---") {
		return &sk, content, nil
	}
	// Find the closing "---" delimiter.
	rest := content[3:] // skip opening ---
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		// Malformed frontmatter — treat entire file as body.
		return &sk, content, nil
	}
	frontmatter := rest[:idx]
	body := strings.TrimPrefix(rest[idx+4:], "\n") // skip closing ---\n
	if err := yaml.Unmarshal([]byte(frontmatter), &sk); err != nil {
		return nil, "", fmt.Errorf("parse frontmatter: %w", err)
	}
	return &sk, body, nil
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
	path := filepath.Join(s.AgentDir(project, name), ".agencycli", "agent.yaml")
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
	path := filepath.Join(s.AgentDir(project, name), ".agencycli", "agent.yaml")
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
		// Skip hidden/system directories (e.g. .fired)
		if strings.HasPrefix(e.Name(), ".") {
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

func (s *fsStore) FiredAgentDir(project, firedDirName string) string {
	return s.abs("projects", project, "agents", ".fired", firedDirName)
}

func (s *fsStore) ListFiredAgents(project string) ([]*FiredAgentEntry, error) {
	base := s.abs("projects", project, "agents", ".fired")
	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: list fired agents for %q: %w", project, err)
	}

	var fired []*FiredAgentEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		firedDirName := e.Name()
		firedDir := filepath.Join(base, firedDirName)

		// Derive original name: strip the trailing "-YYYYMMDD-HHMMSS" suffix if present.
		originalName := firedDirName
		if len(firedDirName) > 16 && firedDirName[len(firedDirName)-16] == '-' {
			originalName = firedDirName[:len(firedDirName)-16]
		}

		// Read meta from the archived directory.
		metaPath := filepath.Join(firedDir, ".agencycli", "agent.yaml")
		var meta entity.AgentMeta
		if err := readYAML(metaPath, &meta); err != nil {
			continue // skip entries without valid meta
		}
		fired = append(fired, &FiredAgentEntry{
			Project:      project,
			FiredDirName: firedDirName,
			OriginalName: originalName,
			Meta:         &meta,
		})
	}
	return fired, nil
}
