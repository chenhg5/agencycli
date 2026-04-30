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

	// ModelHTTPAgent sends the prompt to an OpenAI-compatible HTTP endpoint
	// (e.g. Ollama, LM Studio, LocalAI, or any /v1/chat/completions service).
	// Context file: context.md (used as the system message).
	ModelHTTPAgent AgentModel = "http-agent"

	// ModelHuman represents a human team member with their own inbox.
	// Human agents have no context files or autonomous wakeup; they receive
	// messages and tasks routed via inbox and task queues.
	ModelHuman AgentModel = "human"
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
	ModelHTTPAgent,
	ModelHuman,
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

// APIProvider defines a reusable API provider configuration.
// Stored in <root>/.agencycli/providers.yaml.
type APIProvider struct {
	ID      string            `yaml:"id"      json:"id"`
	Name    string            `yaml:"name"    json:"name"`
	Type    string            `yaml:"type"    json:"type"`     // anthropic, openai, gemini, custom
	BaseURL string            `yaml:"base_url,omitempty" json:"baseUrl,omitempty"`
	APIKey  string            `yaml:"api_key,omitempty"  json:"-"`
	Model   string            `yaml:"model,omitempty"    json:"model,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"      json:"env,omitempty"`
}

// Agency is the top-level organisational unit (the "company").
// Stored at <root>/.agencycli/agency.yaml.
type Agency struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`

	// Lang controls the language used for auto-generated scheduler messages
	// (inbox notifications, wakeup triggers, etc.).
	// Supported values: "en" (default), "zh".
	Lang string `yaml:"lang,omitempty"`

	Vision  string `yaml:"vision,omitempty"`
	Mission string `yaml:"mission,omitempty"`
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
// The skill is defined in a single SKILL.md file whose YAML frontmatter
// carries the metadata and whose Markdown body is injected into agents.
// Stored at <root>/skills/<name>/SKILL.md.
// Legacy layout (skill.yaml + prompt.md) is still recognised as a fallback.
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
// Stored at <root>/projects/<project>/agents/<name>/.agencycli/agent.yaml.
// It is used by `agencycli sync` to detect which context layers have changed.
type AgentMeta struct {
	Name    string     `yaml:"name"`
	Project string     `yaml:"project"`
	Team    string     `yaml:"team"`
	Model   AgentModel `yaml:"model"`
	HiredAt time.Time  `yaml:"hired_at"`

	// SyncedAt is updated each time `agencycli sync` rewrites this agent's
	// context. It is nil for agents that have never been synced after hire.
	SyncedAt *time.Time `yaml:"synced_at,omitempty"`

	// Playbook is the path, relative to the blueprint directory,
	// of the playbook that was copied into .agencycli/context/wakeup.md at hire time.
	// When set, `agencycli sync` tracks the playbook's content hash and
	// re-copies the file automatically if it changes.
	// Example: "playbooks/pm.md"
	Playbook string `yaml:"playbook,omitempty"`

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

	// HTTPAgent configures an HTTP LLM backend for this agent.
	// Required (and only used) when Model == "http-agent".
	HTTPAgent *HTTPAgentConfig `yaml:"http_agent,omitempty"`

	// Provider references an API provider by ID from the workspace-level
	// providers.yaml. When set, the provider's env vars are injected at runtime,
	// overriding the per-agent Env for matching keys.
	Provider string `yaml:"provider,omitempty"`

	// Env holds per-agent environment variable overrides that are set on the
	// agent subprocess (e.g. ANTHROPIC_MODEL, ANTHROPIC_BASE_URL, OPENAI_API_KEY).
	// These are merged with the host environment at runtime, with agent values
	// taking precedence.
	Env map[string]string `yaml:"env,omitempty"`
}

// HTTPAgentConfig configures a custom HTTP LLM backend that speaks the
// OpenAI Chat Completions API (POST /v1/chat/completions).
// Compatible services: Ollama, LM Studio, LocalAI, OpenAI, and any
// OpenAI-API-compatible proxy.
//
// The agent's merged context (context.md written at hire time) is sent as
// the "system" role message. The task prompt becomes the "user" message.
type HTTPAgentConfig struct {
	// URL is the chat completions endpoint.
	// e.g. "http://localhost:11434/v1/chat/completions"  (Ollama)
	//      "https://api.openai.com/v1/chat/completions"  (OpenAI)
	URL string `yaml:"url"`

	// Model is the model identifier passed in the request body.
	// e.g. "llama3.2", "gpt-4o", "mistral", "deepseek-r1:8b"
	Model string `yaml:"model,omitempty"`

	// APIKey is used as the Bearer token in the Authorization header.
	// Leave empty for unauthenticated local services (e.g. Ollama default).
	// Can also be set via the AGENCYCLI_HTTP_API_KEY environment variable.
	APIKey string `yaml:"api_key,omitempty"`

	// Timeout is the per-request timeout as a Go duration string.
	// Defaults to "10m". Increase for large models or slow hardware.
	Timeout string `yaml:"timeout,omitempty"`

	// Stream enables server-sent events streaming.
	// When true, tokens are written to the log file as they arrive.
	Stream bool `yaml:"stream,omitempty"`

	// ExtraHeaders are additional HTTP headers sent with every request.
	// Useful for proxies or services with custom authentication schemes.
	ExtraHeaders map[string]string `yaml:"extra_headers,omitempty"`
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

// StatusGroup maps a TaskStatus into one of three board columns.
type StatusGroup string

const (
	StatusGroupBacklog StatusGroup = "backlog"
	StatusGroupActive  StatusGroup = "active"
	StatusGroupDone    StatusGroup = "done"
)

// Group returns the kanban column group for a status.
func (s TaskStatus) Group() StatusGroup {
	switch s {
	case TaskStatusPending:
		return StatusGroupBacklog
	case TaskStatusInProgress, TaskStatusAwaitingConfirmation, TaskStatusBlocked:
		return StatusGroupActive
	default:
		return StatusGroupDone
	}
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
// It describes which agents exist, their roles, how they wake up, and their
// playbooks.  Running `agencycli project apply` reads this file and brings the
// live state into sync (hire agents, configure heartbeats/crons, install
// playbooks).  It can also be kept in project-blueprints/<name>.yaml inside a
// template so users can bootstrap a project in one step.
type ProjectConfig struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Agents      []AgentSpec `yaml:"agents"`
}

// AgentSpec is one agent definition inside ProjectConfig.
type AgentSpec struct {
	Name    string `yaml:"name"`
	Role    string `yaml:"role,omitempty"`   // references teams/<team>/roles/<role>
	Team    string `yaml:"team,omitempty"`   // team the role belongs to
	Model   string `yaml:"model"`            // e.g. claudecode, codex, gemini

	// Sandbox configures isolated container execution for this agent.
	// When nil the agent runs directly on the host (default behaviour).
	// Example YAML:
	//   sandbox:
	//     provider: docker
	//     docker:
	//       image: ghcr.io/myuser/my-sandbox:latest
	//       memory_mb: 8192
	Sandbox *SandboxConfig `yaml:"sandbox,omitempty"`

	// Repos lists additional repository paths to mount/expose to the agent.
	Repos []string `yaml:"repos,omitempty"`

	// Playbook is the path, relative to the blueprint directory,
	// of the agent's wakeup routine (e.g. "playbooks/pm.md").
	// project apply copies it into .agencycli/context/wakeup.md.
	Playbook string `yaml:"playbook,omitempty"`

	// Heartbeat defines the autonomous wakeup schedule.
	// If omitted the agent is purely reactive.
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
// Stored in <agent-dir>/.agencycli/tasks.yaml (active) and .agencycli/tasks_archive.yaml (terminal).
type Task struct {
	ID        string     `yaml:"id"`
	Title     string     `yaml:"title"`
	Type      TaskType   `yaml:"type,omitempty"`
	Priority  int        `yaml:"priority"` // 0=critical 1=high 2=normal 3=low
	Assignee  string     `yaml:"assignee"` // "<project>/<agent>" or "human"
	CreatedBy string     `yaml:"created_by,omitempty"`
	Status    TaskStatus `yaml:"status"`

	Description string            `yaml:"description,omitempty"` // human-readable detail; Prompt is for agents
	Prompt      string            `yaml:"prompt"`
	Context     map[string]string `yaml:"context,omitempty"`

	// Summary is what the agent reports on completion (used by workflow routing).
	Summary string `yaml:"summary,omitempty"`

	Labels      []string `yaml:"labels,omitempty"`
	ParentID    string   `yaml:"parent_id,omitempty"`    // sub-task parent
	Position    float64  `yaml:"position,omitempty"`     // manual sort order for kanban / list
	MilestoneID string   `yaml:"milestone_id,omitempty"` // linked project milestone

	CreatedAt  time.Time  `yaml:"created_at"`
	UpdatedAt  time.Time  `yaml:"updated_at"`
	StartedAt  *time.Time `yaml:"started_at,omitempty"`
	FinishedAt *time.Time `yaml:"finished_at,omitempty"`
	DueDate    *time.Time `yaml:"due_date,omitempty"`

	DependsOn []string `yaml:"depends_on,omitempty"`

	OnSuccess         []OnSuccessTrigger   `yaml:"on_success,omitempty"`
	ConfirmationReq   *ConfirmationRequest `yaml:"confirmation_request,omitempty"`
	ConfirmationReply string               `yaml:"confirmation_reply,omitempty"`

	RetryCount int    `yaml:"retry_count,omitempty"`
	MaxRetries int    `yaml:"max_retries,omitempty"`
	LastError  string `yaml:"last_error,omitempty"`

	// RunLogPath is set by the runner after execution.
	RunLogPath string `yaml:"run_log_path,omitempty"`

	Vars map[string]string `yaml:"vars,omitempty"`
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

// TaskComment is a comment on a task, posted by a human or agent.
type TaskComment struct {
	ID        string    `yaml:"id"         json:"id"`
	TaskID    string    `yaml:"task_id"    json:"taskId"`
	Author    string    `yaml:"author"     json:"author"` // "human" or "project/agent"
	Body      string    `yaml:"body"       json:"body"`
	CreatedAt time.Time `yaml:"created_at" json:"createdAt"`
}

// NewCommentID generates a unique comment ID.
func NewCommentID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("c-%s-%s", time.Now().UTC().Format("20060102"), string(b))
}

// Message is an asynchronous, non-blocking communication between any two
// participants — human or agent.  Unlike InboxItem (which blocks a task
// waiting for confirmation), a Message is fire-and-forget from the sender's
// perspective.  The recipient reads it on their next wakeup.
//
// Recipient/sender format:
//   "human"               → the agency owner's global inbox
//   "project/agent"       → e.g. "cc-connect/pm"
//
// Storage:
//   human:  <agency>/.agencycli/messages.yaml
//   agent:  <agency>/projects/<project>/agents/<agent>/messages.yaml
type Message struct {
	ID         string     `yaml:"id"`
	From       string     `yaml:"from"`               // "human" or "project/agent"
	To         string     `yaml:"to"`                 // "human" or "project/agent"
	Subject    string     `yaml:"subject,omitempty"`
	Body       string     `yaml:"body"`
	ReplyTo    string     `yaml:"reply_to,omitempty"` // ID of message being replied to
	SentAt     time.Time  `yaml:"sent_at"`
	ReadAt     *time.Time `yaml:"read_at,omitempty"`     // nil = unread
	ArchivedAt *time.Time `yaml:"archived_at,omitempty"` // nil = not archived
}

// NewMessageID returns a unique message ID.
func NewMessageID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(0)
	}
	// Use time-based prefix for ordering.
	return fmt.Sprintf("msg-%s-%s", time.Now().UTC().Format("20060102"), randomAlpha(6))
}

func randomAlpha(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	src := time.Now().UnixNano()
	for i := range b {
		src = src*6364136223846793005 + 1442695040888963407
		b[i] = chars[uint64(src)>>33%uint64(len(chars))]
	}
	return string(b)
}

// InboxItem is an entry in the confirmation inbox.
// Stored at <workspace>/.agencycli/inbox.yaml.
type InboxItem struct {
	TaskID      string   `yaml:"task_id"`
	Project     string   `yaml:"project"`
	Agent       string   `yaml:"agent"`
	// To is the intended recipient of this confirmation request.
	// "human" (default when empty) or "project/agent" (e.g. "cc-connect/pm").
	To          string   `yaml:"to,omitempty"`
	Title       string   `yaml:"title"`
	Summary     string   `yaml:"summary"`
	ActionHint  string   `yaml:"action_hint,omitempty"`
	ActionItems []string `yaml:"action_items,omitempty"` // checklist for the recipient
	ForwardedTo string   `yaml:"forwarded_to,omitempty"` // set when recipient forwards to another agent
	ForwardNote string   `yaml:"forward_note,omitempty"`
	LogPath     string   `yaml:"log_path,omitempty"`
}

// Recipient returns the effective recipient of the inbox item.
// Defaults to "human" when To is empty (backward compatible).
func (i *InboxItem) Recipient() string {
	if i.To == "" {
		return "human"
	}
	return i.To
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

// TriggerType identifies an event that can trigger an immediate agent wakeup.
type TriggerType string

const (
	TriggerOnMessage TriggerType = "message" // incoming message to this agent
	TriggerOnTask    TriggerType = "task"    // task assigned to this agent
)

// HeartbeatConfig holds the per-agent heartbeat configuration and runtime state.
// Stored at <agent-dir>/heartbeat.yaml.
type HeartbeatConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval string `yaml:"interval"` // Go duration string, e.g. "30m", "1h"

	// Paused temporarily halts the heartbeat loop for this agent without
	// removing the configuration. The scheduler continues running but skips
	// wakeup cycles for this agent until Resume is called.
	Paused bool `yaml:"paused,omitempty"`

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

	// WakeupPrompt is executed as a synthetic task when the agent's pending
	// queue is empty on wakeup.  This gives agents like PM and QA a default
	// autonomous routine (scan issues, review PRs, etc.) without requiring an
	// explicit task to be queued first.
	// Can be inline text or a path prefixed with "@" (relative to agent dir).
	// Example: "@.agencycli/context/wakeup.md" reads the prompt from <agent-dir>/.agencycli/context/wakeup.md.
	WakeupPrompt string `yaml:"wakeup_prompt,omitempty"`

	// WakeupCondition is an optional shell command evaluated before each wakeup.
	// The scheduler runs it with `sh -c` from the agent's working directory.
	// Exit 0  → condition met, proceed with wakeup.
	// Non-zero → condition not met, skip this cycle, sleep the full interval,
	//            then re-evaluate on the next tick.
	//
	// The env vars AGENCY_DIR, PROJECT, and AGENT_NAME are injected so that
	// agencycli commands can be used inside the condition script.
	//
	// Examples:
	//   # only wake PM when there are open GitHub issues labelled agent-ready
	//   wakeup_condition: "gh issue list --state open --label agent-ready --json id --jq 'length > 0'"
	//
	//   # only wake QA when there are open PRs
	//   wakeup_condition: "gh pr list --state open | grep -q ."
	//
	//   # only wake when there are unread inbox messages
	//   wakeup_condition: "agencycli --dir $AGENCY_DIR inbox messages --unread-only | grep -q ."
	WakeupCondition string `yaml:"wakeup_condition,omitempty"`

	// WakeupPreset is a built-in condition evaluated before each wakeup,
	// checked before WakeupCondition. Supported values:
	//   ""                      — no preset, always wake (default)
	//   "require_tasks"         — skip if no pending tasks
	//   "require_messages"      — skip if no unread messages
	//   "require_any"           — skip if neither pending tasks nor unread messages
	WakeupPreset string `yaml:"wakeup_preset,omitempty"`

	// MaxTasksPerCycle limits how many tasks are processed in a single wakeup
	// cycle before the cycle ends. 0 (default) means unlimited.
	// Use this to prevent a single wakeup from monopolising the agent on a large queue.
	MaxTasksPerCycle int `yaml:"max_tasks_per_cycle,omitempty"`

	// MaxCycleDuration limits how long a wakeup cycle runs before it stops.
	// The value is a Go duration string (e.g. "15m", "1h"). 0 (default) means unlimited.
	// The elapsed time is checked between tasks; a running task will complete even
	// if it pushes the total over the limit.
	MaxCycleDuration string `yaml:"max_cycle_duration,omitempty"`

	// Triggers lists event types that trigger an immediate wakeup.
	// Supported values: "message" (incoming message), "task" (task assigned).
	// When an event fires, the agent is woken up asynchronously if not already
	// running. Works independently of heartbeat Enabled — an agent can be
	// trigger-only with no periodic heartbeat.
	Triggers []TriggerType `yaml:"triggers,omitempty" json:"triggers,omitempty"`

	// Runtime state (mutated by scheduler / runner).
	PID               int        `yaml:"pid,omitempty"`
	LastWakeup        *time.Time `yaml:"last_wakeup,omitempty"`
	LastWakeupStatus  string     `yaml:"last_wakeup_status,omitempty"` // running | done | failed
	SessionID         string     `yaml:"session_id,omitempty"`
	SessionStartedAt  *time.Time `yaml:"session_started_at,omitempty"`

	// LastConditionStatus records the outcome of the most recent WakeupCondition
	// evaluation: "met", "not_met", or "" (never evaluated).
	LastConditionStatus string     `yaml:"last_condition_status,omitempty"`
	LastConditionAt     *time.Time `yaml:"last_condition_at,omitempty"`

	// Jitter adds a random delay in [0, Jitter) before each wakeup to avoid
	// thundering-herd patterns. Go duration string, e.g. "5m", "10m".
	// On the very first cycle after scheduler start, jitter is always applied
	// within [0, min(Jitter, interval)). If empty or "0", defaults to the full
	// interval on the first cycle only (backward-compatible behaviour).
	Jitter string `yaml:"jitter,omitempty"`

	// Scheduler runtime stats (persisted across scheduler restarts).
	WakeupCount      int        `yaml:"wakeup_count,omitempty"`
	WakeupCountToday int        `yaml:"wakeup_count_today,omitempty"`
	WakeupDate       string     `yaml:"wakeup_date,omitempty"`
	LastCycleDuration string    `yaml:"last_cycle_duration,omitempty"`
	NextWakeupAt     *time.Time `yaml:"next_wakeup_at,omitempty"`
	SchedulerStartedAt *time.Time `yaml:"scheduler_started_at,omitempty"`
}

// HasTrigger reports whether the heartbeat config includes the given trigger type.
func (h *HeartbeatConfig) HasTrigger(t TriggerType) bool {
	for _, tr := range h.Triggers {
		if tr == t {
			return true
		}
	}
	return false
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

	// Jitter adds a random delay in [0, Jitter) before the cron actually fires.
	// Go duration string, e.g. "5m", "10m". Prevents thundering-herd when
	// multiple crons share the same schedule.
	Jitter string `yaml:"jitter,omitempty"`

	// SessionScope controls whether each cron run starts a fresh session
	// or continues in the same persistent session.
	//   "new"        — each execution starts a new session (default)
	//   "persistent" — reuse the same session across executions
	SessionScope string `yaml:"session_scope,omitempty"`

	// SessionID is the current session ID when SessionScope is "persistent".
	SessionID        string     `yaml:"session_id,omitempty"`
	SessionStartedAt *time.Time `yaml:"session_started_at,omitempty"`

	LastRun       *time.Time `yaml:"last_run,omitempty"`
	LastRunStatus string     `yaml:"last_run_status,omitempty"`
	RunCount      int        `yaml:"run_count,omitempty"`
}

// ─────────────────────────────────────────────
// OKR (Objectives and Key Results)
// ─────────────────────────────────────────────

// OKRStatus represents the health status of an OKR.
type OKRStatus string

const (
	OKRStatusOnTrack    OKRStatus = "on_track"
	OKRStatusInProgress OKRStatus = "in_progress"
	OKRStatusAtRisk     OKRStatus = "at_risk"
	OKRStatusOffTrack   OKRStatus = "off_track"
	OKRStatusAchieved   OKRStatus = "achieved"
)

// MetricType defines how a Key Result is measured.
type MetricType string

const (
	MetricTypePercentage MetricType = "percentage"
	MetricTypeNumber     MetricType = "number"
	MetricTypeBoolean    MetricType = "boolean"
	MetricTypeCurrency   MetricType = "currency"
)

// KeyResult is a measurable outcome that drives an Objective forward.
type KeyResult struct {
	ID          string     `yaml:"id"           json:"id"`
	Description string     `yaml:"description"  json:"description"`
	MetricType  MetricType `yaml:"metric_type"  json:"metricType"`
	TargetValue float64    `yaml:"target_value" json:"targetValue"`
	CurrentValue float64   `yaml:"current_value" json:"currentValue"`
	Unit        string     `yaml:"unit,omitempty" json:"unit,omitempty"`
	Weight      float64    `yaml:"weight,omitempty" json:"weight,omitempty"` // 0 = equal weight
	LinkedMilestones []string `yaml:"linked_milestones,omitempty" json:"linkedMilestones,omitempty"`
}

// Progress returns 0-100 completion percentage.
func (kr *KeyResult) Progress() float64 {
	if kr.MetricType == MetricTypeBoolean {
		if kr.CurrentValue > 0 {
			return 100
		}
		return 0
	}
	if kr.TargetValue == 0 {
		return 0
	}
	p := kr.CurrentValue / kr.TargetValue * 100
	if p > 100 {
		p = 100
	}
	if p < 0 {
		p = 0
	}
	return p
}

// ReviewNote records a periodic check-in on an OKR.
type ReviewNote struct {
	Date   string `yaml:"date"   json:"date"`
	Note   string `yaml:"note"   json:"note"`
	Author string `yaml:"author" json:"author"`
}

// OKR is an Objective with its associated Key Results.
// Stored in <root>/.agencycli/okrs.yaml.
// OKRScope identifies the level at which an OKR is defined.
type OKRScope string

const (
	OKRScopeAgency  OKRScope = "agency"
	OKRScopeProject OKRScope = "project"
	OKRScopeAgent   OKRScope = "agent"
)

type OKR struct {
	ID          string       `yaml:"id"          json:"id"`
	Scope       OKRScope     `yaml:"scope,omitempty"    json:"scope,omitempty"`
	ScopeRef    string       `yaml:"scope_ref,omitempty" json:"scopeRef,omitempty"`
	ParentID    string       `yaml:"parent_id,omitempty" json:"parentId,omitempty"`
	Objective   string       `yaml:"objective"   json:"objective"`
	Description string       `yaml:"description,omitempty" json:"description,omitempty"`
	Owner       string       `yaml:"owner"       json:"owner"`
	Quarter     string       `yaml:"quarter"     json:"quarter"`
	Status      OKRStatus    `yaml:"status"      json:"status"`
	KeyResults  []KeyResult  `yaml:"key_results" json:"keyResults"`
	ReviewNotes []ReviewNote `yaml:"review_notes,omitempty" json:"reviewNotes,omitempty"`

	CreatedAt time.Time `yaml:"created_at" json:"createdAt"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updatedAt"`
}

// Progress returns the weighted average progress across all Key Results.
func (o *OKR) Progress() float64 {
	if len(o.KeyResults) == 0 {
		return 0
	}
	var totalWeight, weightedSum float64
	for i := range o.KeyResults {
		w := o.KeyResults[i].Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
		weightedSum += w * o.KeyResults[i].Progress()
	}
	if totalWeight == 0 {
		return 0
	}
	return weightedSum / totalWeight
}

// OKRFile is the top-level structure for okrs.yaml.
type OKRFile struct {
	CurrentQuarter string `yaml:"current_quarter" json:"currentQuarter"`
	OKRs           []OKR  `yaml:"okrs"            json:"okrs"`
}

func NewOKRID() string {
	now := time.Now().UTC()
	q := (now.Month()-1)/3 + 1
	return fmt.Sprintf("okr-%dq%d-%s", now.Year(), q, randomString(6))
}

func NewKRID() string {
	return "kr-" + randomString(6)
}

// helper shared by all ID generators
func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// ─────────────────────────────────────────────
// Milestone
// ─────────────────────────────────────────────

// MilestoneStatus represents the lifecycle state of a milestone.
type MilestoneStatus string

const (
	MilestoneStatusPlanned    MilestoneStatus = "planned"
	MilestoneStatusInProgress MilestoneStatus = "in_progress"
	MilestoneStatusCompleted  MilestoneStatus = "completed"
	MilestoneStatusCancelled  MilestoneStatus = "cancelled"
)

// Milestone is a project-level deliverable goal.
// Stored in <root>/projects/<project>/.agencycli/milestones.yaml.
type Milestone struct {
	ID          string          `yaml:"id"          json:"id"`
	Title       string          `yaml:"title"       json:"title"`
	Description string          `yaml:"description,omitempty" json:"description,omitempty"`
	Status      MilestoneStatus `yaml:"status"      json:"status"`
	DueDate     *time.Time      `yaml:"due_date,omitempty" json:"dueDate,omitempty"`
	Owner       string          `yaml:"owner,omitempty" json:"owner,omitempty"`
	Progress    int             `yaml:"progress"    json:"progress"` // 0-100, manual or auto-calc

	Criteria    []string `yaml:"criteria,omitempty"    json:"criteria,omitempty"`
	LinkedKR    []string `yaml:"linked_kr,omitempty"   json:"linkedKR,omitempty"`
	TaskLabels  []string `yaml:"task_labels,omitempty" json:"taskLabels,omitempty"`

	CreatedAt time.Time `yaml:"created_at" json:"createdAt"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updatedAt"`
}

// MilestoneFile is the top-level structure for milestones.yaml.
type MilestoneFile struct {
	Milestones []Milestone `yaml:"milestones" json:"milestones"`
}

func NewMilestoneID() string {
	return "ms-" + randomString(8)
}

// ─────────────────────────────────────────────
// Template
// ─────────────────────────────────────────────

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

// ─────────────────────────────────────────────
// Workspace Secrets
// ─────────────────────────────────────────────

// EnvVarScope defines whether an environment variable applies globally or to specific agents.
type EnvVarScope string

const (
	EnvVarScopeGlobal EnvVarScope = "global" // applied to all agents
	EnvVarScopeAgents EnvVarScope = "agents" // applied to listed agents only
)

// EnvVar is a workspace-level environment variable entry stored in envvars.yaml.
// Variables are injected into agent processes at runtime and forwarded into Docker
// containers automatically. Resolution priority (lowest → highest):
//
//	workspace global → agent-scoped → API provider env → per-agent env
type EnvVar struct {
	ID          string      `yaml:"id"          json:"id"`
	Key         string      `yaml:"key"         json:"key"`
	Value       string      `yaml:"value"       json:"-"`
	Scope       EnvVarScope `yaml:"scope"       json:"scope"`
	Agents      []string    `yaml:"agents,omitempty" json:"agents,omitempty"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time   `yaml:"created_at"  json:"createdAt"`
	UpdatedAt   time.Time   `yaml:"updated_at"  json:"updatedAt"`
}
