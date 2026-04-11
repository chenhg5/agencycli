// Package taskstore manages per-agent task queues, heartbeat configuration,
// and the workspace-level human inbox.
//
// All business logic in higher-level packages (runner, cmd) depends only on
// the Store interface — never on a concrete implementation. This means the
// backing store can be swapped between filesystem YAML (default), SQLite, or
// any other database without changing callers.
package taskstore

import "github.com/chenhg5/agencycli/internal/entity"

// Store is the single access point for all workflow persistence.
// Implementations must be safe for concurrent use from multiple goroutines
// (the scheduler runs one heartbeat goroutine per agent).
type Store interface {
	// ── Tasks ────────────────────────────────────────────────────────────────

	// AddTask enqueues a new task for the given agent.
	AddTask(project, agent string, t *entity.Task) error

	// GetTask retrieves a task by ID, searching active and archived tasks.
	GetTask(project, agent, id string) (*entity.Task, error)

	// UpdateTask persists a modified in-flight task.
	// The task must already exist in the active (non-archived) set.
	UpdateTask(project, agent string, t *entity.Task) error

	// ListTasks returns active (non-terminal) tasks for the agent.
	// Pass one or more statuses to filter; empty filter returns all active tasks.
	ListTasks(project, agent string, filter ...entity.TaskStatus) ([]*entity.Task, error)

	// ArchiveTask moves a terminal-state task out of the active set.
	// For file backends this means tasks_archive.yaml; for DB backends a
	// status column update plus optional separate table.
	ArchiveTask(project, agent string, t *entity.Task) error

	// ListArchivedTasks returns completed/cancelled tasks.
	ListArchivedTasks(project, agent string) ([]*entity.Task, error)

	// OverwriteArchive replaces the entire archived task list for an agent.
	// Used by task retry to remove a retried task from the archive.
	OverwriteArchive(project, agent string, tasks []*entity.Task) error

	// DeleteTask removes a task by ID from the active queue or the archive.
	DeleteTask(project, agent, taskID string) error

	// ClearTasks removes all tasks (active and archived) for the given agent.
	ClearTasks(project, agent string) error

	// ClearInbox removes all items from the human inbox and rewrites the summary.
	ClearInbox() error

	// ── Heartbeat ────────────────────────────────────────────────────────────

	// GetHeartbeat returns the heartbeat config for the agent.
	// Returns a zero-value HeartbeatConfig (Enabled=false) if not configured.
	GetHeartbeat(project, agent string) (*entity.HeartbeatConfig, error)

	// SaveHeartbeat persists heartbeat config and runtime state atomically.
	SaveHeartbeat(project, agent string, h *entity.HeartbeatConfig) error

	// PauseHeartbeat sets Paused=true on the agent's heartbeat config.
	PauseHeartbeat(project, agent string) error

	// ResumeHeartbeat sets Paused=false on the agent's heartbeat config.
	ResumeHeartbeat(project, agent string) error

	// ── Crons ────────────────────────────────────────────────────────────────

	// ListCrons returns all cron definitions for the agent.
	ListCrons(project, agent string) ([]*entity.Cron, error)

	// SaveCrons replaces the entire cron list for the agent atomically.
	SaveCrons(project, agent string, crons []*entity.Cron) error

	// PauseCron sets Enabled=false for a specific cron by ID.
	PauseCron(project, agent, cronID string) error

	// ResumeCron sets Enabled=true for a specific cron by ID.
	ResumeCron(project, agent, cronID string) error

	// DeleteCron removes a cron entirely from the agent's crons list.
	DeleteCron(project, agent, cronID string) error

	// ── Inbox (task confirmations) ───────────────────────────────────────────

	// AddToInbox routes a task to the human confirmation inbox.
	// Implementations should also refresh the human-readable inbox summary.
	AddToInbox(item *entity.InboxItem) error

	// ListInbox returns all items currently awaiting human action.
	ListInbox() ([]*entity.InboxItem, error)

	// RemoveFromInbox removes an item by task ID and refreshes the inbox summary.
	RemoveFromInbox(taskID string) error

	// ── Messages (async agent ↔ human / agent ↔ agent) ───────────────────────

	// SendMessage delivers a message to the recipient's mailbox.
	// recipient is "human" or "project/agent".
	SendMessage(msg *entity.Message) error

	// ListMessages returns all messages for a recipient (newest first).
	// recipient is "human" or "project/agent".
	ListMessages(recipient string) ([]*entity.Message, error)

	// ListUnreadMessages returns only unread messages for a recipient.
	ListUnreadMessages(recipient string) ([]*entity.Message, error)

	// MarkMessagesRead marks all unread messages for a recipient as read.
	MarkMessagesRead(recipient string) error

	// ListAllMessages returns every message for a recipient including archived ones.
	ListAllMessages(recipient string) ([]*entity.Message, error)

	// MarkMessageRead marks a single message as read by ID.
	// Returns an error if the message is not found.
	MarkMessageRead(recipient, msgID string) error

	// ArchiveMessage marks a single message as archived by ID.
	// Archived messages are hidden from normal listing unless --all is used.
	// Returns an error if the message is not found.
	ArchiveMessage(recipient, msgID string) error

	// DeleteMessage permanently removes a single message by ID.
	// Returns an error if the message is not found.
	DeleteMessage(recipient, msgID string) error

	// ClearMessages removes all messages for a recipient (human or project/agent).
	ClearMessages(recipient string) error

	// ── Discovery ────────────────────────────────────────────────────────────

	// ListProjects returns the names of all projects in the workspace.
	ListProjects() ([]string, error)

	// ListAgents returns the names of all agents under a project.
	ListAgents(project string) ([]string, error)

	// FindTaskByID searches all projects and agents for a task with the given ID.
	// It checks both active and archived tasks.
	// Returns the owning project name, agent name, and task on success,
	// or ("", "", nil, error) when not found.
	FindTaskByID(id string) (project, agent string, task *entity.Task, err error)

	// ── Task Comments ────────────────────────────────────────────────────────

	// AddComment persists a comment for a task.
	AddComment(project, agent string, c *entity.TaskComment) error

	// ListComments returns all comments for a task, oldest first.
	ListComments(project, agent, taskID string) ([]*entity.TaskComment, error)

	// DeleteComment removes a comment by ID.
	DeleteComment(project, agent, commentID string) error

	// ── Run logs ─────────────────────────────────────────────────────────────

	// RunLogDir returns (and creates) the directory where execution logs for
	// the agent should be written. For non-filesystem backends this may return
	// a temporary directory.
	RunLogDir(project, agent string) (string, error)

	// ── Project config ────────────────────────────────────────────────────────

	// GetProjectConfig reads projects/<project>/project.yaml.
	// Returns nil, nil when the file does not yet exist.
	GetProjectConfig(project string) (*entity.ProjectConfig, error)

	// SaveProjectConfig writes (or overwrites) projects/<project>/project.yaml.
	SaveProjectConfig(project string, cfg *entity.ProjectConfig) error

	// GetProjectBlueprint searches for a named blueprint in project-blueprints/.
	// Returns nil, nil when not found.
	GetProjectBlueprint(name string) (*entity.ProjectConfig, error)

	// ListProjectBlueprints returns names of all available blueprints.
	ListProjectBlueprints() ([]string, error)

}
