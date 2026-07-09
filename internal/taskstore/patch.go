package taskstore

import (
	"fmt"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
)

// TaskPatch holds optional task field updates. Only non-nil pointers are applied.
type TaskPatch struct {
	Title            *string
	Description      *string
	Status           *entity.TaskStatus
	Priority         *int
	Type             *entity.TaskType
	Summary          *string
	Labels           *[]string
	ParentID         *string
	DueDate          *string // YYYY-MM-DD; empty string clears
	EstimateDuration *string // Go duration; empty string clears
	Position         *float64
	Assignee         *string
	Prompt           *string
}

func (p TaskPatch) empty() bool {
	return p.Title == nil && p.Description == nil && p.Status == nil && p.Priority == nil &&
		p.Type == nil && p.Summary == nil && p.Labels == nil && p.ParentID == nil &&
		p.DueDate == nil && p.EstimateDuration == nil && p.Position == nil &&
		p.Assignee == nil && p.Prompt == nil
}

// ApplyTaskPatch mutates t according to patch and returns the previous status.
func ApplyTaskPatch(t *entity.Task, patch TaskPatch, now time.Time) (entity.TaskStatus, error) {
	if t == nil {
		return "", fmt.Errorf("task is nil")
	}
	if patch.empty() {
		return t.Status, fmt.Errorf("no fields to update")
	}

	prev := t.Status

	if patch.Title != nil {
		title := strings.TrimSpace(*patch.Title)
		if title == "" {
			return prev, fmt.Errorf("title cannot be empty")
		}
		t.Title = title
	}
	if patch.Description != nil {
		t.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.Status != nil {
		if !entity.ValidTaskStatus(string(*patch.Status)) {
			return prev, fmt.Errorf("invalid task status %q", *patch.Status)
		}
		t.Status = *patch.Status
	}
	if patch.Priority != nil {
		p := *patch.Priority
		if p < 0 || p > 3 {
			return prev, fmt.Errorf("priority must be 0–3")
		}
		t.Priority = p
	}
	if patch.Type != nil {
		if !entity.ValidTaskType(string(*patch.Type)) {
			return prev, fmt.Errorf("invalid task type %q", *patch.Type)
		}
		t.Type = *patch.Type
	}
	if patch.Summary != nil {
		t.Summary = strings.TrimSpace(*patch.Summary)
	}
	if patch.Labels != nil {
		t.Labels = *patch.Labels
	}
	if patch.ParentID != nil {
		t.ParentID = strings.TrimSpace(*patch.ParentID)
	}
	if patch.DueDate != nil {
		dd := strings.TrimSpace(*patch.DueDate)
		if dd == "" {
			t.DueDate = nil
		} else if parsed, err := time.Parse("2006-01-02", dd); err != nil {
			return prev, fmt.Errorf("invalid due date %q, use YYYY-MM-DD", dd)
		} else {
			t.DueDate = &parsed
		}
	}
	if patch.EstimateDuration != nil {
		est, err := entity.NormalizeEstimateDuration(*patch.EstimateDuration)
		if err != nil {
			return prev, err
		}
		t.EstimateDuration = est
	}
	if patch.Position != nil {
		t.Position = *patch.Position
	}
	if patch.Assignee != nil {
		t.Assignee = strings.TrimSpace(*patch.Assignee)
	}
	if patch.Prompt != nil {
		prompt := strings.TrimSpace(*patch.Prompt)
		if prompt == "" {
			return prev, fmt.Errorf("prompt cannot be empty")
		}
		t.Prompt = prompt
	}

	t.UpdatedAt = now
	entity.ApplyStatusTimestamps(t, prev, now)
	return prev, nil
}

// PersistTask saves an updated task to the active queue or archive.
// Terminal tasks in the active queue are moved to the archive automatically.
func (s *FSStore) PersistTask(project, agent string, t *entity.Task) error {
	active, err := s.ListTasks(project, agent)
	if err != nil {
		return err
	}
	for _, at := range active {
		if at.ID == t.ID {
			if err := s.UpdateTask(project, agent, t); err != nil {
				return err
			}
			if t.Status.IsTerminal() {
				return s.ArchiveTask(project, agent, t)
			}
			return nil
		}
	}

	archived, err := s.ListArchivedTasks(project, agent)
	if err != nil {
		return err
	}
	found := false
	for i, at := range archived {
		if at.ID == t.ID {
			archived[i] = t
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("task %q not found", t.ID)
	}
	return s.OverwriteArchive(project, agent, archived)
}
