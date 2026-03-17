// Package entity defines the core domain types for agencycli.
// All types are plain data structures with no business logic.
// They are serialised to/from YAML by the store layer.
package entity

import (
	"fmt"
	"math/rand"
	"time"
)

// AgentModel identifies which AI agent runtime an employee uses.
// Names match the identifiers used by cc-connect for interoperability.
type AgentModel string

const (
	// ModelClaudeCode drives Anthropic Claude Code (claude CLI).
	// Context file: CLAUDE.md with @import layers + .claude/skills/.
	ModelClaudeCode AgentModel = "claudecode"

	// ModelCodex drives OpenAI Codex CLI.
	// Context file: AGENTS.md (single merged file).
	ModelCodex AgentModel = "codex"

	// ModelCursor drives Cursor Agent CLI.
	// Context files: .cursorrules + .cursor/rules/agencycli.mdc
	ModelCursor AgentModel = "cursor"

	// ModelGemini drives Google Gemini CLI.
	// Context file: GEMINI.md + .gemini/skills/.
	ModelGemini AgentModel = "gemini"

	// ModelQoder drives Qoder CLI (qodercli).
	// Context file: AGENTS.md (same format as Codex).
	ModelQoder AgentModel = "qoder"

	// ModelOpenCode drives OpenCode CLI.
	// Context file: OPENCODE.md.
	ModelOpenCode AgentModel = "opencode"

	// ModelIFlow drives iFlow CLI.
	// Context file: IFLOW.md.
	ModelIFlow AgentModel = "iflow"

	// ModelGenericCLI is a fallback for any other CLI agent.
	// Context file: context.md (plain merged text).
	ModelGenericCLI AgentModel = "generic-cli"
)

// KnownModels lists all supported agent models in display order.
var KnownModels = []AgentModel{
	ModelClaudeCode,
	ModelCodex,
	ModelCursor,
	ModelGemini,
	ModelQoder,
	ModelOpenCode,
	ModelIFlow,
	ModelGenericCLI,
}

// modelAliases maps legacy or alternate spellings to the canonical model name.
var modelAliases = map[AgentModel]AgentModel{
	"claude-code": ModelClaudeCode, // kebab-case alias kept for backward compat
}

// NormaliseModel returns the canonical AgentModel for m, resolving any alias.
func NormaliseModel(m AgentModel) AgentModel {
	if canonical, ok := modelAliases[m]; ok {
		return canonical
	}
	return m
}

// IsValidModel reports whether m (after alias resolution) is a known AgentModel.
func IsValidModel(m AgentModel) bool {
	m = NormaliseModel(m)
	for _, k := range KnownModels {
		if k == m {
			return true
		}
	}
	return false
}

// Agency is the top-level organisational unit (the "company").
// Stored at <root>/.agencycli/agency.yaml.
type Agency struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// Team represents a functional group inside the agency.
// Teams can be nested; the Parent field holds the slash-separated
// path of the parent team (empty = direct child of agency).
// Stored at <root>/teams/<path>/team.yaml.
type Team struct {
	Name        string   `yaml:"name"`
	Parent      string   `yaml:"parent,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Goals       []string `yaml:"goals,omitempty"`
	// Skills lists skill names this team uses.
	// Skills are inherited by all sub-teams.
	Skills []string `yaml:"skills,omitempty"`
}

// Project is a concrete product or initiative.
// Stored at <root>/projects/<name>/project.yaml.
type Project struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	// Repo is the path (relative or absolute) to the actual code repository.
	Repo string `yaml:"repo,omitempty"`
}

// Skill is a reusable capability definition.
// The executable behaviour is described in prompt.md.
// Stored at <root>/skills/<name>/skill.yaml.
type Skill struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// Role is a reusable job definition that lives under a team.
// It provides an extra prompt layer, bound skills, and workspace setup
// instructions that are applied when an agent is hired into this role.
// Stored at <root>/teams/<team>/roles/<name>/role.yaml.
type Role struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description,omitempty"`
	// Skills lists skill names bound to this role (merged with team skills).
	Skills      []string  `yaml:"skills,omitempty"`
	// Setup describes the workspace layout to create inside the agent directory
	// when an agent is hired into this role.
	Setup       RoleSetup `yaml:"setup,omitempty"`
}

// RoleSetup describes the workspace scaffolding applied at hire time.
type RoleSetup struct {
	// Dirs lists subdirectories to create inside the agent working directory.
	// e.g. ["images", "reference", "generates"]
	Dirs []string `yaml:"dirs,omitempty"`
	// Files lists files to create (with optional content) inside the agent dir.
	Files []RoleSetupFile `yaml:"files,omitempty"`
}

// RoleSetupFile is a file to create during workspace setup.
type RoleSetupFile struct {
	// Path is relative to the agent working directory.
	Path    string `yaml:"path"`
	// Content is written verbatim. Empty creates an empty file.
	Content string `yaml:"content,omitempty"`
}

// AgentMeta records the provenance of a hired agent working directory.
// Stored at <root>/projects/<project>/agents/<name>/.agencycli-agent.yaml.
// It is used by `agencycli sync` to detect which context layers have changed.
type AgentMeta struct {
	Name    string     `yaml:"name"`
	Project string     `yaml:"project"`
	Team    string     `yaml:"team"`
	Model   AgentModel `yaml:"model"`
	HiredAt time.Time  `yaml:"hired_at"`

	// ContextHash maps each layer source key to the SHA-256 hex digest
	// of its prompt content at hire time.
	ContextHash map[string]string `yaml:"context_hash,omitempty"`

	// Role is the optional role name this agent was hired into (e.g. "content-writer").
	// Empty when the agent was hired without a specific role.
	Role string `yaml:"role,omitempty"`

	// AddDirs lists additional directories the agent should have access to
	// beyond its own working directory (e.g. the project's code repository).
	// For claudecode these become --add-dir flags on the claude CLI.
	AddDirs []string `yaml:"add_dirs,omitempty"`

	// RunCommand overrides the default agent CLI invocation command.
	// Use {prompt_file} and {session_id} as placeholders.
	// Example: "my-agent --input {prompt_file}"
	RunCommand string `yaml:"run_command,omitempty"`

	// Sandbox configures isolated container execution for this agent.
	// When nil the agent runs directly on the host (default behaviour).
	Sandbox *SandboxConfig `yaml:"sandbox,omitempty"`
}

// ─────────────────────────────────────────────
// Sandbox
// ─────────────────────────────────────────────

// SandboxProvider identifies the sandbox backend.
type SandboxProvider string

const (
	// SandboxNone runs the agent directly on the host (default).
	SandboxNone SandboxProvider = ""
	// SandboxDocker runs the agent inside a plain Docker container.
	// Works on any OS with Docker installed; no Docker Desktop required.
	SandboxDocker SandboxProvider = "docker"
)

// SandboxConfig describes how to isolate an agent execution.
// Resolved at hire/run time with agency → team → agent override priority.
type SandboxConfig struct {
	Provider SandboxProvider `yaml:"provider"`

	// Docker holds Docker-specific options. Used when Provider == "docker".
	Docker *DockerSandboxConfig `yaml:"docker,omitempty"`
}

// DockerSandboxConfig holds options for Docker-based sandbox execution.
type DockerSandboxConfig struct {
	// Image is the container image to use.
	// Defaults are chosen by model when empty:
	//   claudecode → ghcr.io/agencycli/sandbox-claudecode:latest
	//   codex      → ghcr.io/agencycli/sandbox-codex:latest
	//   gemini     → ghcr.io/agencycli/sandbox-gemini:latest
	//   (others)   → ghcr.io/agencycli/sandbox-generic:latest
	Image string `yaml:"image,omitempty"`

	// NetworkMode controls Docker networking.
	// "bridge" (default) — internet access, agent can reach GitHub/APIs.
	// "none"             — fully offline, safest option.
	// "host"             — shares host network (debug only, not recommended).
	NetworkMode string `yaml:"network_mode,omitempty"`

	// CredentialMounts mounts host credential paths into the container
	// as read-only volumes. Format: "~/.claude:/root/.claude" or
	// "~/.claude:/root/.claude:ro". The tilde is expanded at runtime.
	// Defaults are set automatically per-model when empty.
	CredentialMounts []string `yaml:"credential_mounts,omitempty"`

	// ExtraVolumes mounts additional host paths. Same format as CredentialMounts.
	ExtraVolumes []string `yaml:"extra_volumes,omitempty"`

	// ExtraEnv passes additional environment variables as "KEY=VALUE" pairs.
	ExtraEnv []string `yaml:"extra_env,omitempty"`

	// MemoryMB limits container memory (0 = no limit).
	MemoryMB int `yaml:"memory_mb,omitempty"`

	// CPUs limits CPU quota, e.g. 2.0 (0 = no limit).
	CPUs float64 `yaml:"cpus,omitempty"`

	// NoAutoCredentials disables the automatic per-model credential mount
	// defaults. Set to true when you manage credential mounts manually.
	NoAutoCredentials bool `yaml:"no_auto_credentials,omitempty"`
}

// ─────────────────────────────────────────────
// Task system
// ─────────────────────────────────────────────

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskStatusPending              TaskStatus = "pending"
	TaskStatusInProgress           TaskStatus = "in_progress"
	TaskStatusAwaitingConfirmation TaskStatus = "awaiting_confirmation"
	TaskStatusBlocked              TaskStatus = "blocked"
	TaskStatusDoneSuccess          TaskStatus = "done_success"
	TaskStatusDoneFailed           TaskStatus = "done_failed"
	TaskStatusCancelled            TaskStatus = "cancelled"
)

// IsTerminal reports whether s is a terminal (archived) state.
func (s TaskStatus) IsTerminal() bool {
	return s == TaskStatusDoneSuccess || s == TaskStatusDoneFailed || s == TaskStatusCancelled
}

// TaskType categorises the kind of work a task represents.
type TaskType string

const (
	TaskTypeFeature  TaskType = "feature"
	TaskTypeBug      TaskType = "bug"
	TaskTypeReview   TaskType = "review"
	TaskTypeTriage   TaskType = "triage"
	TaskTypeTest     TaskType = "test"
	TaskTypeResearch TaskType = "research"
	TaskTypeChore    TaskType = "chore"
)

// OnSuccessTrigger describes a task to auto-create when this task completes.
type OnSuccessTrigger struct {
	// Assignee is "<project>/<agent-name>" or "human".
	Assignee string `yaml:"assignee"`
	Title    string `yaml:"title"`
	Type     string `yaml:"type,omitempty"`
	Priority int    `yaml:"priority,omitempty"`
	Prompt   string `yaml:"prompt"`
}

// ─────────────────────────────────────────────
// Project configuration  (project.yaml)
// ─────────────────────────────────────────────

// ProjectConfig is the declarative definition of a project.
// Stored at <agency>/projects/<project>/project.yaml.
// It describes which agents exist, their roles, how they wake up, and which
// workflows are relevant to the project.  Running `agencycli project apply`
// reads this file and brings the live state into sync (hire agents, configure
// heartbeats/crons).  It can also be kept in project-blueprints/<name>.yaml
// inside a template so users can bootstrap a project in one step.
type ProjectConfig struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Agents      []AgentSpec `yaml:"agents"`

	// Workflows lists the workflow names this project uses.
	// These are for documentation / `project apply` to show next steps.
	// Start a workflow instance with: agencycli workflow run <name> --project <project>
	Workflows []string `yaml:"workflows,omitempty"`
}

// AgentSpec is one agent definition inside ProjectConfig.
type AgentSpec struct {
	Name    string `yaml:"name"`
	Role    string `yaml:"role,omitempty"`   // references teams/<team>/roles/<role>
	Team    string `yaml:"team,omitempty"`   // team the role belongs to
	Model   string `yaml:"model"`            // e.g. claudecode, codex, gemini
	Sandbox bool   `yaml:"sandbox,omitempty"`

	// Repos lists additional repository paths to mount/expose to the agent.
	Repos []string `yaml:"repos,omitempty"`

	// Heartbeat defines the autonomous wakeup schedule.
	// If omitted the agent is purely reactive (only wakes when `run` is called
	// manually or triggered by the workflow routing engine).
	Heartbeat *HeartbeatConfig `yaml:"heartbeat,omitempty"`

	// Crons adds scheduled tasks to the agent's queue on a crontab schedule.
	Crons []AgentCronSpec `yaml:"crons,omitempty"`
}

// AgentCronSpec is an inline cron definition inside an AgentSpec.
// It is converted into an entity.Cron when `project apply` is run.
type AgentCronSpec struct {
	ID       string `yaml:"id"`
	Schedule string `yaml:"schedule"` // standard crontab, e.g. "0 9 * * 1-5"
	Title    string `yaml:"title"`
	Prompt   string `yaml:"prompt"`
}

// ConfirmationRequest holds information surfaced to the human inbox.
type ConfirmationRequest struct {
	Summary     string   `yaml:"summary"`
	ActionHint  string   `yaml:"action_hint,omitempty"`
	ActionItems []string `yaml:"action_items,omitempty"` // numbered checklist for the human
}

// Task is the atomic unit of work assigned to an agent or human.
// Stored in <agent-dir>/tasks.yaml (active) and tasks_archive.yaml (terminal).
type Task struct {
	ID       string     `yaml:"id"`
	Title    string     `yaml:"title"`
	Type     TaskType   `yaml:"type,omitempty"`
	Priority int        `yaml:"priority"` // 0=critical 1=high 2=normal 3=low
	Assignee string     `yaml:"assignee"` // "<project>/<agent>" or "human"
	CreatedBy string    `yaml:"created_by,omitempty"`
	Status   TaskStatus `yaml:"status"`

	Prompt  string            `yaml:"prompt"`
	Context map[string]string `yaml:"context,omitempty"`

	// Summary is what the agent reports on completion (used by workflow routing).
	Summary string `yaml:"summary,omitempty"`

	CreatedAt  time.Time  `yaml:"created_at"`
	UpdatedAt  time.Time  `yaml:"updated_at"`
	StartedAt  *time.Time `yaml:"started_at,omitempty"`
	FinishedAt *time.Time `yaml:"finished_at,omitempty"`

	DependsOn []string `yaml:"depends_on,omitempty"`

	OnSuccess         []OnSuccessTrigger   `yaml:"on_success,omitempty"`
	ConfirmationReq   *ConfirmationRequest `yaml:"confirmation_request,omitempty"`
	ConfirmationReply string               `yaml:"confirmation_reply,omitempty"`

	RetryCount int    `yaml:"retry_count,omitempty"`
	MaxRetries int    `yaml:"max_retries,omitempty"`
	LastError  string `yaml:"last_error,omitempty"`

	// RunLogPath is set by the runner after execution.
	RunLogPath string `yaml:"run_log_path,omitempty"`

	// Workflow fields — set when this task was created by a workflow route.
	WorkflowID string            `yaml:"workflow_id,omitempty"`  // instance ID
	TemplateID string            `yaml:"template_id,omitempty"`  // task template id
	Vars       map[string]string `yaml:"vars,omitempty"`         // template vars at creation time
	RoutedAt   *time.Time        `yaml:"routed_at,omitempty"`    // set when workflow routing ran
}

// NewTaskID generates a sortable unique task ID.
func NewTaskID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("t-%s-%s", time.Now().UTC().Format("20060102"), string(b))
}

// InboxItem is an entry in the human inbox.
// Stored at <workspace>/.agencycli/inbox.yaml.
type InboxItem struct {
	TaskID      string    `yaml:"task_id"`
	Project     string    `yaml:"project"`
	Agent       string    `yaml:"agent"`
	Title       string    `yaml:"title"`
	Summary     string    `yaml:"summary"`
	ActionHint  string    `yaml:"action_hint,omitempty"`
	ActionItems []string  `yaml:"action_items,omitempty"` // checklist for the human
	ForwardedTo string    `yaml:"forwarded_to,omitempty"` // set when human forwards to an agent
	ForwardNote string    `yaml:"forward_note,omitempty"`
	RoutedAt    time.Time `yaml:"routed_at"`
	LogPath     string    `yaml:"log_path,omitempty"`
}

// ─────────────────────────────────────────────
// Heartbeat & session
// ─────────────────────────────────────────────

// SessionScope controls how session IDs are shared across task runs.
type SessionScope string

const (
	SessionScopeCycle SessionScope = "cycle" // all tasks in one wakeup share a session
	SessionScopeTask  SessionScope = "task"  // each task gets a fresh session resume
)

// HeartbeatConfig holds the per-agent heartbeat configuration and runtime state.
// Stored at <agent-dir>/heartbeat.yaml.
type HeartbeatConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval string `yaml:"interval"` // Go duration string, e.g. "30m", "1h"

	// ActiveHours restricts wakeups to a specific time window each day.
	// Format: "HH:MM-HH:MM" in local time, e.g. "09:00-18:00".
	// If empty, wakeups are allowed at any time.
	// Overnight ranges like "22:00-06:00" are supported.
	ActiveHours string `yaml:"active_hours,omitempty"`

	// ActiveDays restricts wakeups to specific days of the week.
	// Comma-separated list of day names (Mon,Tue,Wed,Thu,Fri,Sat,Sun) or
	// short aliases (weekdays, weekends).  Empty means every day.
	ActiveDays string `yaml:"active_days,omitempty"`

	// SessionScope determines session sharing strategy within a wakeup cycle.
	SessionScope SessionScope `yaml:"session_scope,omitempty"`

	// Runtime state (mutated by daemon / runner).
	PID               int        `yaml:"pid,omitempty"`
	LastWakeup        *time.Time `yaml:"last_wakeup,omitempty"`
	LastWakeupStatus  string     `yaml:"last_wakeup_status,omitempty"` // running | done | failed
	SessionID         string     `yaml:"session_id,omitempty"`
	SessionStartedAt  *time.Time `yaml:"session_started_at,omitempty"`
}

// ─────────────────────────────────────────────
// Cron
// ─────────────────────────────────────────────

// Cron defines a calendar-scheduled recurring task for an agent.
// Stored at <agent-dir>/crons.yaml.
// When a cron fires it enqueues a new Task (does not directly invoke the agent).
type Cron struct {
	ID       string `yaml:"id"`
	Title    string `yaml:"title"`
	Schedule string `yaml:"schedule"` // crontab expression, e.g. "0 9 * * 1-5"
	Enabled  bool   `yaml:"enabled"`
	Prompt   string `yaml:"prompt"`

	LastRun       *time.Time `yaml:"last_run,omitempty"`
	LastRunStatus string     `yaml:"last_run_status,omitempty"`
}

// ─────────────────────────────────────────────
// Template
// ─────────────────────────────────────────────

// ─────────────────────────────────────────────
// Workflow
// ─────────────────────────────────────────────

// WorkflowManifest is the top-level definition of an async workflow.
// Stored as YAML at <agency>/workflows/<name>.yaml or
// <agency>/projects/<project>/workflows/<name>.yaml.
type WorkflowManifest struct {
	Name        string             `yaml:"name"`
	Version     string             `yaml:"version,omitempty"`
	Description string             `yaml:"description,omitempty"`
	Templates   []WFTaskTemplate   `yaml:"templates"`
	Routes      []WFRoute          `yaml:"routes"`
	Entry       *WFEntry           `yaml:"entry"`
	Triggers    []WFTrigger        `yaml:"triggers,omitempty"`
}

// WFTaskTemplate is a reusable task definition with variable slots {{var}}.
type WFTaskTemplate struct {
	ID       string `yaml:"id"`
	Title    string `yaml:"title"`
	Agent    string `yaml:"agent"`              // agent name in the project
	Type     string `yaml:"type,omitempty"`
	Priority int    `yaml:"priority,omitempty"` // default 0 (high)
	Prompt   string `yaml:"prompt"`
}

// WFRoute is triggered when a task with a matching template completes.
type WFRoute struct {
	On     WFCondition  `yaml:"on"`
	Create *WFCreate    `yaml:"create,omitempty"` // enqueue a new task
	Inbox  *WFInbox     `yaml:"inbox,omitempty"`  // route to human inbox
}

// WFCondition matches a completed task by template ID and terminal status.
type WFCondition struct {
	Template   string `yaml:"template"`
	Status     string `yaml:"status"`               // success | failed | any
	MaxTrigger int    `yaml:"max_trigger,omitempty"` // circuit-breaker; 0 = unlimited
}

// WFCreate describes a task to enqueue when a route fires.
type WFCreate struct {
	Template string            `yaml:"template"`
	Vars     map[string]string `yaml:"vars,omitempty"`
}

// WFInbox routes completion to the human inbox instead of creating a task.
type WFInbox struct {
	Title       string   `yaml:"title"`
	Summary     string   `yaml:"summary,omitempty"`
	ActionItems []string `yaml:"action_items,omitempty"`
}

// WFEntry defines the first task created when a workflow is started.
type WFEntry struct {
	Template string            `yaml:"template"`
	Vars     map[string]string `yaml:"vars,omitempty"`
}

// WFTrigger declares how the workflow can be started automatically.
type WFTrigger struct {
	Type     string            `yaml:"type"`               // manual | cron
	Schedule string            `yaml:"schedule,omitempty"` // crontab (type=cron)
	Inputs   map[string]string `yaml:"inputs,omitempty"`
}

// WorkflowInstance is the runtime state of one workflow execution.
// Stored at <agency>/projects/<project>/workflow-runs/<id>.yaml.
type WorkflowInstance struct {
	ID       string `yaml:"id"`
	Workflow string `yaml:"workflow"` // manifest name
	Project  string `yaml:"project"`
	Status   string `yaml:"status"` // running | done | failed

	Inputs map[string]string `yaml:"inputs,omitempty"`

	// StepOutputs maps template_id → agent's task.Summary (for {{steps.X.summary}})
	StepOutputs map[string]string `yaml:"step_outputs,omitempty"`

	// RouteTriggers tracks how many times each route key has fired (circuit-breaker).
	// Key: "<template_id>:<status>", e.g. "implement:failed"
	RouteTriggers map[string]int `yaml:"route_triggers,omitempty"`

	StartedAt  time.Time  `yaml:"started_at"`
	FinishedAt *time.Time `yaml:"finished_at,omitempty"`
}

// NewWorkflowInstanceID generates a sortable workflow instance ID.
func NewWorkflowInstanceID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("wf-%s-%s", time.Now().UTC().Format("20060102"), string(b))
}

// TemplateManifest is the metadata file (template.json) bundled at the root
// of every agencycli agency template archive.  Format is intentionally
// similar to npm package.json for familiarity.
type TemplateManifest struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author,omitempty"`
	Email       string   `json:"email,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Homepage    string   `json:"homepage,omitempty"` // repo / docs URL
	License     string   `json:"license,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"` // RFC3339
}
