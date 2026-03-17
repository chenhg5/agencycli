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
// (the daemon runs one heartbeat goroutine per agent).
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

	// ── Heartbeat ────────────────────────────────────────────────────────────

	// GetHeartbeat returns the heartbeat config for the agent.
	// Returns a zero-value HeartbeatConfig (Enabled=false) if not configured.
	GetHeartbeat(project, agent string) (*entity.HeartbeatConfig, error)

	// SaveHeartbeat persists heartbeat config and runtime state atomically.
	SaveHeartbeat(project, agent string, h *entity.HeartbeatConfig) error

	// ── Crons ────────────────────────────────────────────────────────────────

	// ListCrons returns all cron definitions for the agent.
	ListCrons(project, agent string) ([]*entity.Cron, error)

	// SaveCrons replaces the entire cron list for the agent atomically.
	SaveCrons(project, agent string, crons []*entity.Cron) error

	// ── Inbox ────────────────────────────────────────────────────────────────

	// AddToInbox routes a task to the human confirmation inbox.
	// Implementations should also refresh the human-readable inbox summary.
	AddToInbox(item *entity.InboxItem) error

	// ListInbox returns all items currently awaiting human action.
	ListInbox() ([]*entity.InboxItem, error)

	// RemoveFromInbox removes an item by task ID and refreshes the inbox summary.
	RemoveFromInbox(taskID string) error

	// ── Discovery ────────────────────────────────────────────────────────────

	// ListProjects returns the names of all projects in the workspace.
	ListProjects() ([]string, error)

	// ListAgents returns the names of all agents under a project.
	ListAgents(project string) ([]string, error)

	// ── Run logs ─────────────────────────────────────────────────────────────

	// RunLogDir returns (and creates) the directory where execution logs for
	// the agent should be written. For non-filesystem backends this may return
	// a temporary directory.
	RunLogDir(project, agent string) (string, error)

	// ── Workflows ─────────────────────────────────────────────────────────────

	// GetWorkflow searches for a workflow manifest by name.
	// Search order: projects/<project>/workflows/<name>.yaml, then workflows/<name>.yaml.
	// Returns nil, nil if not found.
	GetWorkflow(project, name string) (*entity.WorkflowManifest, error)

	// ListWorkflows returns all workflow manifests visible to the project.
	ListWorkflows(project string) ([]*entity.WorkflowManifest, error)

	// SaveWorkflowInstance persists a workflow instance state.
	SaveWorkflowInstance(project string, inst *entity.WorkflowInstance) error

	// GetWorkflowInstance retrieves a workflow instance by ID.
	GetWorkflowInstance(project, id string) (*entity.WorkflowInstance, error)

	// ListWorkflowInstances returns all instances for a project, newest first.
	ListWorkflowInstances(project string) ([]*entity.WorkflowInstance, error)
}
