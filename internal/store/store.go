// Package store defines the storage interface and filesystem implementation
// for agencycli workspace data.
//
// The Store interface abstracts all reads and writes so that higher-level
// packages (ctxbuild, formatter, scaffold) never touch the filesystem directly.
// This also makes unit testing straightforward with a stub implementation.
package store

import "github.com/chenhg5/agencycli/internal/entity"

// TeamEntry is a team together with its slash-separated path.
type TeamEntry struct {
	// Path is the slash-separated team path, e.g. "engineering/backend".
	Path string
	Team *entity.Team
}

// RoleEntry is a role together with its team path and name.
type RoleEntry struct {
	TeamPath string
	Name     string
	Role     *entity.Role
}

// AgentEntry is an agent's metadata together with its location.
type AgentEntry struct {
	Project string
	Name    string
	Meta    *entity.AgentMeta
}

// FiredAgentEntry is a soft-deleted agent together with its archived location.
type FiredAgentEntry struct {
	Project      string
	FiredDirName string // directory name under .fired/, e.g. "dev-claude-20260316-163152"
	OriginalName string // agent name before firing
	Meta         *entity.AgentMeta
}

// Store is the single access point for all workspace data.
// All path arguments are relative to the workspace root.
type Store interface {
	// Root returns the absolute path of the workspace root.
	Root() string

	// ── Agency ────────────────────────────────────────────────────────────

	Agency() (*entity.Agency, error)
	SaveAgency(a *entity.Agency) error
	AgencyPrompt() (string, error)
	SaveAgencyPrompt(content string) error

	// ── Teams ─────────────────────────────────────────────────────────────
	// path is a slash-separated team path, e.g. "engineering/backend".

	Team(path string) (*entity.Team, error)
	SaveTeam(path string, t *entity.Team) error
	TeamPrompt(path string) (string, error)
	SaveTeamPrompt(path string, content string) error
	// ListTeams returns all teams in no guaranteed order.
	ListTeams() ([]*TeamEntry, error)

	// ── Projects ──────────────────────────────────────────────────────────

	Project(name string) (*entity.Project, error)
	SaveProject(name string, p *entity.Project) error
	ProjectPrompt(name string) (string, error)
	SaveProjectPrompt(name string, content string) error
	ListProjects() ([]*entity.Project, error)

	// ProjectConfig reads the declarative project.yaml (agents, playbooks, etc.).
	// Returns nil, nil when the file does not exist.
	ProjectConfig(name string) (*entity.ProjectConfig, error)

	// ── Roles ─────────────────────────────────────────────────────────────
	// teamPath is a slash-separated team path, e.g. "engineering".
	// roleName is the role directory name, e.g. "backend-dev".

	Role(teamPath, roleName string) (*entity.Role, error)
	SaveRole(teamPath, roleName string, r *entity.Role) error
	RolePrompt(teamPath, roleName string) (string, error)
	SaveRolePrompt(teamPath, roleName string, content string) error
	RoleDir(teamPath, roleName string) string
	ListRoles(teamPath string) ([]*RoleEntry, error)

	// ── Skills ────────────────────────────────────────────────────────────

	Skill(name string) (*entity.Skill, error)
	SkillPrompt(name string) (string, error)
	ListSkills() ([]*entity.Skill, error)
	// SkillDir returns the absolute path of a skill's source directory.
	// Used by ctxbuild to scan for bundled files (scripts, etc.).
	SkillDir(name string) string

	// ── Agents ────────────────────────────────────────────────────────────

	AgentMeta(project, name string) (*entity.AgentMeta, error)
	SaveAgentMeta(project, name string, meta *entity.AgentMeta) error
	ListAgents(project string) ([]*AgentEntry, error)

	// AgentDir returns the absolute path of an agent's working directory.
	AgentDir(project, name string) string

	// FiredAgentDir returns the absolute path of the soft-delete archive directory
	// for a given fired-directory name (e.g. "dev-<timestamp>").
	FiredAgentDir(project, firedDirName string) string

	// ListFiredAgents returns all soft-deleted agents for a project.
	ListFiredAgents(project string) ([]*FiredAgentEntry, error)
}
