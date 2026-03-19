// Package scaffold creates the initial directory layout and template files
// for agencycli workspace objects (agency, team, project).
//
// Scaffold operations are idempotent: they never overwrite an existing file,
// so running create twice is always safe.
package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"gopkg.in/yaml.v3"
)

// Scaffolder creates workspace objects using a backing store.
type Scaffolder struct {
	store store.Store
}

// New returns a Scaffolder backed by the given store.
func New(s store.Store) *Scaffolder {
	return &Scaffolder{store: s}
}

// ── Agency ────────────────────────────────────────────────────────────────────

// InitAgency writes .agencycli/agency.yaml, agency-prompt.md, and the standard
// top-level subdirectories inside root.
// root must already exist on disk.
func InitAgency(root string, a *entity.Agency) error {
	s := store.NewFS(root)

	if err := s.SaveAgency(a); err != nil {
		return fmt.Errorf("scaffold: init agency: %w", err)
	}

	if err := writeTemplateOnce(
		filepath.Join(root, "agency-prompt.md"),
		agencyPromptTmpl, a,
	); err != nil {
		return fmt.Errorf("scaffold: agency prompt: %w", err)
	}

	for _, dir := range []string{"teams", "projects", "skills"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return fmt.Errorf("scaffold: create %s dir: %w", dir, err)
		}
	}

	if err := writeTemplateOnce(
		filepath.Join(root, ".gitignore"),
		gitignoreTmpl, nil,
	); err != nil {
		return fmt.Errorf("scaffold: gitignore: %w", err)
	}

	return nil
}

// ── Team ──────────────────────────────────────────────────────────────────────

// CreateTeam writes team.yaml and an initial prompt.md for a team.
// path is a slash-separated team path, e.g. "engineering/backend".
func (sc *Scaffolder) CreateTeam(path string, t *entity.Team) error {
	if err := sc.store.SaveTeam(path, t); err != nil {
		return fmt.Errorf("scaffold: save team %q: %w", path, err)
	}
	promptPath := filepath.Join(
		sc.store.Root(), "teams",
		filepath.FromSlash(path), "prompt.md",
	)
	if err := writeTemplateOnce(promptPath, teamPromptTmpl, t); err != nil {
		return fmt.Errorf("scaffold: team prompt %q: %w", path, err)
	}
	return nil
}

// ── Project ───────────────────────────────────────────────────────────────────

// CreateProject writes project.yaml, an initial prompt.md, and an empty
// agents/ directory for a project.
func (sc *Scaffolder) CreateProject(name string, p *entity.Project) error {
	if err := sc.store.SaveProject(name, p); err != nil {
		return fmt.Errorf("scaffold: save project %q: %w", name, err)
	}
	promptPath := filepath.Join(sc.store.Root(), "projects", name, "prompt.md")
	if err := writeTemplateOnce(promptPath, projectPromptTmpl, p); err != nil {
		return fmt.Errorf("scaffold: project prompt %q: %w", name, err)
	}
	agentsDir := filepath.Join(sc.store.Root(), "projects", name, "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return fmt.Errorf("scaffold: create agents dir for %q: %w", name, err)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// writeTemplateOnce renders tmplStr with data into path.
// If path already exists it is left untouched (idempotent).
func writeTemplateOnce(path, tmplStr string, data any) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// writeYAMLOnce marshals v to YAML and writes it to path, skipping if the
// file already exists.
func writeYAMLOnce(path string, v any) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

