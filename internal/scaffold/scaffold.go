// Package scaffold creates the initial directory layout and template files
// for agencycli workspace objects (company, department, project).
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

	"github.com/agencycli/agencycli/internal/entity"
	"github.com/agencycli/agencycli/internal/store"
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

// ── Company ───────────────────────────────────────────────────────────────────

// InitCompany writes .aios/company.yaml, company-prompt.md, and the standard
// top-level subdirectories inside root. It also installs the built-in skills.
// root must already exist on disk.
func InitCompany(root string, c *entity.Company) error {
	s := store.NewFS(root)

	if err := s.SaveCompany(c); err != nil {
		return fmt.Errorf("scaffold: init company: %w", err)
	}

	if err := writeTemplateOnce(
		filepath.Join(root, "company-prompt.md"),
		companyPromptTmpl, c,
	); err != nil {
		return fmt.Errorf("scaffold: company prompt: %w", err)
	}

	for _, dir := range []string{"departments", "projects", "skills"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return fmt.Errorf("scaffold: create %s dir: %w", dir, err)
		}
	}

	if err := installBuiltinSkills(root); err != nil {
		return fmt.Errorf("scaffold: install skills: %w", err)
	}

	return nil
}

// ── Department ────────────────────────────────────────────────────────────────

// CreateDept writes dept.yaml and an initial prompt.md for a department.
// path is a slash-separated dept path, e.g. "engineering/backend".
func (sc *Scaffolder) CreateDept(path string, d *entity.Department) error {
	if err := sc.store.SaveDepartment(path, d); err != nil {
		return fmt.Errorf("scaffold: save dept %q: %w", path, err)
	}
	promptPath := filepath.Join(
		sc.store.Root(), "departments",
		filepath.FromSlash(path), "prompt.md",
	)
	if err := writeTemplateOnce(promptPath, deptPromptTmpl, d); err != nil {
		return fmt.Errorf("scaffold: dept prompt %q: %w", path, err)
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

// installBuiltinSkills writes the built-in skill definitions into
// <root>/skills/. Existing files are never overwritten.
func installBuiltinSkills(root string) error {
	for _, sk := range builtinSkills {
		dir := filepath.Join(root, "skills", sk.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		def := &entity.Skill{Name: sk.Name, Description: sk.Description}
		if err := writeYAMLOnce(filepath.Join(dir, "skill.yaml"), def); err != nil {
			return err
		}
		promptPath := filepath.Join(dir, "prompt.md")
		if _, err := os.Stat(promptPath); os.IsNotExist(err) {
			if err := os.WriteFile(promptPath, []byte(sk.Prompt), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
