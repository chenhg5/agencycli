package taskstore

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"gopkg.in/yaml.v3"
)

const (
	tasksFile   = "tasks.yaml"
	archiveFile = "tasks_archive.yaml"
	heartbeatFile = "heartbeat.yaml"
	cronsFile   = "crons.yaml"
	inboxYAML   = "inbox.yaml"
	inboxMD     = "inbox.md"
	aiosDir     = ".agencycli"
	runsDir     = "runs"
)

// FSStore is the filesystem-backed implementation of Store.
// Tasks, heartbeat config, crons, and inbox are stored as YAML files inside
// the workspace directory tree.
//
// To use a different backend (SQLite, Postgres …) implement the Store interface
// and pass the new implementation wherever taskstore.New() is currently called.
type FSStore struct {
	root string // workspace root
}

// New returns a filesystem-backed Store rooted at the given workspace root.
// The return type is the Store interface — callers must not depend on *FSStore.
func New(root string) Store {
	return &FSStore{root: root}
}

// agentDir returns <root>/projects/<project>/agents/<agent>.
func (s *FSStore) agentDir(project, agent string) string {
	return filepath.Join(s.root, "projects", project, "agents", agent)
}

// projectDir returns <root>/projects/<project>.
func (s *FSStore) projectDir(project string) string {
	return filepath.Join(s.root, "projects", project)
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

func (s *FSStore) AddTask(project, agent string, t *entity.Task) error {
	tasks, err := s.loadTasks(project, agent)
	if err != nil {
		return err
	}
	tasks = append(tasks, t)
	return s.saveTasks(project, agent, tasks)
}

func (s *FSStore) GetTask(project, agent, id string) (*entity.Task, error) {
	tasks, err := s.loadTasks(project, agent)
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.ID == id {
			return t, nil
		}
	}
	// Check archive
	archived, err := s.ListArchivedTasks(project, agent)
	if err != nil {
		return nil, err
	}
	for _, t := range archived {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, fmt.Errorf("task %q not found", id)
}

func (s *FSStore) UpdateTask(project, agent string, t *entity.Task) error {
	tasks, err := s.loadTasks(project, agent)
	if err != nil {
		return err
	}
	for i, task := range tasks {
		if task.ID == t.ID {
			t.UpdatedAt = time.Now().UTC()
			tasks[i] = t
			return s.saveTasks(project, agent, tasks)
		}
	}
	return fmt.Errorf("task %q not found in active queue", t.ID)
}

func (s *FSStore) ListTasks(project, agent string, filter ...entity.TaskStatus) ([]*entity.Task, error) {
	tasks, err := s.loadTasks(project, agent)
	if err != nil {
		return nil, err
	}
	if len(filter) == 0 {
		return tasks, nil
	}
	set := make(map[entity.TaskStatus]bool, len(filter))
	for _, f := range filter {
		set[f] = true
	}
	var out []*entity.Task
	for _, t := range tasks {
		if set[t.Status] {
			out = append(out, t)
		}
	}
	return out, nil
}

func (s *FSStore) ArchiveTask(project, agent string, t *entity.Task) error {
	// Append to archive
	archived, err := s.ListArchivedTasks(project, agent)
	if err != nil {
		return err
	}
	archived = append(archived, t)
	dir := s.agentDir(project, agent)
	if err := writeYAMLAtomic(filepath.Join(dir, archiveFile), archived); err != nil {
		return err
	}
	// Remove from active
	tasks, err := s.loadTasks(project, agent)
	if err != nil {
		return err
	}
	var remaining []*entity.Task
	for _, task := range tasks {
		if task.ID != t.ID {
			remaining = append(remaining, task)
		}
	}
	return s.saveTasks(project, agent, remaining)
}

func (s *FSStore) ListArchivedTasks(project, agent string) ([]*entity.Task, error) {
	path := filepath.Join(s.agentDir(project, agent), archiveFile)
	return loadTasksFromFile(path)
}

func (s *FSStore) loadTasks(project, agent string) ([]*entity.Task, error) {
	path := filepath.Join(s.agentDir(project, agent), tasksFile)
	return loadTasksFromFile(path)
}

func (s *FSStore) saveTasks(project, agent string, tasks []*entity.Task) error {
	dir := s.agentDir(project, agent)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeYAMLAtomic(filepath.Join(dir, tasksFile), tasks)
}

func loadTasksFromFile(path string) ([]*entity.Task, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var tasks []*entity.Task
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return tasks, nil
}

// ── Heartbeat ─────────────────────────────────────────────────────────────────

func (s *FSStore) GetHeartbeat(project, agent string) (*entity.HeartbeatConfig, error) {
	path := filepath.Join(s.agentDir(project, agent), heartbeatFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &entity.HeartbeatConfig{}, nil
	}
	if err != nil {
		return nil, err
	}
	var h entity.HeartbeatConfig
	if err := yaml.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &h, nil
}

func (s *FSStore) SaveHeartbeat(project, agent string, h *entity.HeartbeatConfig) error {
	dir := s.agentDir(project, agent)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeYAMLAtomic(filepath.Join(dir, heartbeatFile), h)
}

// ── Crons ─────────────────────────────────────────────────────────────────────

func (s *FSStore) ListCrons(project, agent string) ([]*entity.Cron, error) {
	path := filepath.Join(s.agentDir(project, agent), cronsFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var crons []*entity.Cron
	if err := yaml.Unmarshal(data, &crons); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return crons, nil
}

func (s *FSStore) SaveCrons(project, agent string, crons []*entity.Cron) error {
	dir := s.agentDir(project, agent)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeYAMLAtomic(filepath.Join(dir, cronsFile), crons)
}

// ── Inbox ──────────────────────────────────────────────────────────────────────

func (s *FSStore) inboxDir() string {
	return filepath.Join(s.root, aiosDir)
}

func (s *FSStore) AddToInbox(item *entity.InboxItem) error {
	items, err := s.ListInbox()
	if err != nil {
		return err
	}
	items = append(items, item)
	if err := s.saveInbox(items); err != nil {
		return err
	}
	return s.regenerateInboxMD(items)
}

func (s *FSStore) ListInbox() ([]*entity.InboxItem, error) {
	path := filepath.Join(s.inboxDir(), inboxYAML)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var items []*entity.InboxItem
	if err := yaml.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse inbox.yaml: %w", err)
	}
	return items, nil
}

func (s *FSStore) RemoveFromInbox(taskID string) error {
	items, err := s.ListInbox()
	if err != nil {
		return err
	}
	var remaining []*entity.InboxItem
	for _, item := range items {
		if item.TaskID != taskID {
			remaining = append(remaining, item)
		}
	}
	if err := s.saveInbox(remaining); err != nil {
		return err
	}
	return s.regenerateInboxMD(remaining)
}

func (s *FSStore) saveInbox(items []*entity.InboxItem) error {
	dir := s.inboxDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeYAMLAtomic(filepath.Join(dir, inboxYAML), items)
}

func (s *FSStore) regenerateInboxMD(items []*entity.InboxItem) error {
	var buf bytes.Buffer
	if len(items) == 0 {
		buf.WriteString("# Inbox\n\nNo items awaiting your confirmation.\n")
	} else {
		fmt.Fprintf(&buf, "# Inbox — %d item(s) awaiting your confirmation\n\n", len(items))
		for _, item := range items {
			fmt.Fprintf(&buf, "## [%s / %s] %s\n", item.Project, item.Agent, item.Title)
			fmt.Fprintf(&buf, "> Routed at: %s\n\n", item.RoutedAt.Format("2006-01-02 15:04"))
			if item.Summary != "" {
				fmt.Fprintf(&buf, "%s\n\n", item.Summary)
			}
			if item.LogPath != "" {
				fmt.Fprintf(&buf, "**Run log:** `%s`\n\n", item.LogPath)
			}
			if item.ActionHint != "" {
				fmt.Fprintf(&buf, "> %s\n\n", item.ActionHint)
			}
			fmt.Fprintf(&buf, "```\n")
			fmt.Fprintf(&buf, "agencycli inbox confirm %s\n", item.TaskID)
			fmt.Fprintf(&buf, "agencycli inbox reject  %s --reason \"...\"\n", item.TaskID)
			fmt.Fprintf(&buf, "agencycli inbox comment %s --message \"...\"\n", item.TaskID)
			fmt.Fprintf(&buf, "```\n\n")
			buf.WriteString("---\n\n")
		}
	}
	path := filepath.Join(s.inboxDir(), inboxMD)
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// RunLogDir returns (and creates) the runs/ directory for the given agent.
func (s *FSStore) RunLogDir(project, agent string) (string, error) {
	dir := filepath.Join(s.agentDir(project, agent), runsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// ListAgents returns all agent names under a project.
func (s *FSStore) ListAgents(project string) ([]string, error) {
	base := filepath.Join(s.root, "projects", project, "agents")
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// OverwriteArchive replaces the entire archive for an agent.
// Used by task retry to remove the retried entry from the archive.
func (s *FSStore) OverwriteArchive(project, agent string, tasks []*entity.Task) error {
	dir := s.agentDir(project, agent)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeYAMLAtomic(filepath.Join(dir, archiveFile), tasks)
}

// ListProjects returns all project names in the workspace.
func (s *FSStore) ListProjects() ([]string, error) {
	base := filepath.Join(s.root, "projects")
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

// writeYAMLAtomic marshals v to YAML and writes it atomically using a temp file.
func writeYAMLAtomic(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// priorityLabel converts numeric priority to a readable label.
func PriorityLabel(p int) string {
	switch p {
	case 0:
		return "critical"
	case 1:
		return "high"
	case 2:
		return "normal"
	case 3:
		return "low"
	default:
		return fmt.Sprintf("p%d", p)
	}
}

// StatusIcon returns a compact status symbol for display.
func StatusIcon(s entity.TaskStatus) string {
	switch s {
	case entity.TaskStatusPending:
		return "○"
	case entity.TaskStatusInProgress:
		return "●"
	case entity.TaskStatusAwaitingConfirmation:
		return "?"
	case entity.TaskStatusBlocked:
		return "⊘"
	case entity.TaskStatusDoneSuccess:
		return "✓"
	case entity.TaskStatusDoneFailed:
		return "✗"
	case entity.TaskStatusCancelled:
		return "–"
	default:
		return " "
	}
}

// FormatDuration formats a Go duration string for display.
func FormatDuration(d string) string {
	d = strings.TrimSpace(d)
	if d == "" {
		return "not set"
	}
	return d
}

// ── Workflows ─────────────────────────────────────────────────────────────────

func (s *FSStore) workflowRunsDir(project string) string {
	return filepath.Join(s.projectDir(project), "workflow-runs")
}

func (s *FSStore) GetWorkflow(project, name string) (*entity.WorkflowManifest, error) {
	// Search project-level first, then agency-level.
	candidates := []string{
		filepath.Join(s.projectDir(project), "workflows", name+".yaml"),
		filepath.Join(s.root, "workflows", name+".yaml"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var wf entity.WorkflowManifest
		return &wf, yaml.Unmarshal(data, &wf)
	}
	return nil, nil
}

func (s *FSStore) ListWorkflows(project string) ([]*entity.WorkflowManifest, error) {
	seen := map[string]bool{}
	var out []*entity.WorkflowManifest
	dirs := []string{
		filepath.Join(s.projectDir(project), "workflows"),
		filepath.Join(s.root, "workflows"),
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			var wf entity.WorkflowManifest
			if err := yaml.Unmarshal(data, &wf); err != nil || wf.Name == "" {
				continue
			}
			if seen[wf.Name] {
				continue // project-level already added
			}
			seen[wf.Name] = true
			out = append(out, &wf)
		}
	}
	return out, nil
}

func (s *FSStore) SaveWorkflowInstance(project string, inst *entity.WorkflowInstance) error {
	dir := s.workflowRunsDir(project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeYAMLAtomic(filepath.Join(dir, inst.ID+".yaml"), inst)
}

func (s *FSStore) GetWorkflowInstance(project, id string) (*entity.WorkflowInstance, error) {
	data, err := os.ReadFile(filepath.Join(s.workflowRunsDir(project), id+".yaml"))
	if err != nil {
		return nil, err
	}
	var inst entity.WorkflowInstance
	return &inst, yaml.Unmarshal(data, &inst)
}

func (s *FSStore) ListWorkflowInstances(project string) ([]*entity.WorkflowInstance, error) {
	dir := s.workflowRunsDir(project)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []*entity.WorkflowInstance
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var inst entity.WorkflowInstance
		if err := yaml.Unmarshal(data, &inst); err == nil {
			out = append(out, &inst)
		}
	}
	// Newest first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// Compile-time assertion: FSStore must fully implement Store.
// If a new method is added to Store, this line will fail to compile
// until FSStore also implements it — ensuring no silent drift.
var _ Store = (*FSStore)(nil)
