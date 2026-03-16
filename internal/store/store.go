// Package store defines the storage interface and filesystem implementation
// for agencycli workspace data.
//
// The Store interface abstracts all reads and writes so that higher-level
// packages (ctxbuild, formatter, scaffold) never touch the filesystem directly.
// This also makes unit testing straightforward with a stub implementation.
package store

import "github.com/agencycli/agencycli/internal/entity"

// DeptEntry is a department together with its slash-separated path.
type DeptEntry struct {
	// Path is the slash-separated dept path, e.g. "engineering/backend".
	Path       string
	Department *entity.Department
}

// AgentEntry is an agent's metadata together with its location.
type AgentEntry struct {
	Project string
	Name    string
	Meta    *entity.AgentMeta
}

// Store is the single access point for all workspace data.
// All path arguments are relative to the workspace root.
type Store interface {
	// Root returns the absolute path of the workspace root.
	Root() string

	// ── Company ──────────────────────────────────────────────────────────

	Company() (*entity.Company, error)
	SaveCompany(c *entity.Company) error
	CompanyPrompt() (string, error)
	SaveCompanyPrompt(content string) error

	// ── Departments ───────────────────────────────────────────────────────
	// path is a slash-separated dept path, e.g. "engineering/backend".

	Department(path string) (*entity.Department, error)
	SaveDepartment(path string, d *entity.Department) error
	DeptPrompt(path string) (string, error)
	SaveDeptPrompt(path string, content string) error
	// ListDepartments returns all departments in no guaranteed order.
	ListDepartments() ([]*DeptEntry, error)

	// ── Projects ──────────────────────────────────────────────────────────

	Project(name string) (*entity.Project, error)
	SaveProject(name string, p *entity.Project) error
	ProjectPrompt(name string) (string, error)
	SaveProjectPrompt(name string, content string) error
	ListProjects() ([]*entity.Project, error)

	// ── Skills ────────────────────────────────────────────────────────────

	Skill(name string) (*entity.Skill, error)
	SkillPrompt(name string) (string, error)
	ListSkills() ([]*entity.Skill, error)

	// ── Agents ────────────────────────────────────────────────────────────

	AgentMeta(project, name string) (*entity.AgentMeta, error)
	SaveAgentMeta(project, name string, meta *entity.AgentMeta) error
	ListAgents(project string) ([]*AgentEntry, error)

	// AgentDir returns the absolute path of an agent's working directory.
	AgentDir(project, name string) string
}
