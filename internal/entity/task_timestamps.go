package entity

import (
	"fmt"
	"strings"
	"time"
)

// ApplyStatusTimestamps sets StartedAt / FinishedAt based on status transitions.
// StartedAt is set once when entering in_progress; FinishedAt when entering a terminal state.
func ApplyStatusTimestamps(t *Task, prevStatus TaskStatus, now time.Time) {
	if t == nil {
		return
	}
	if t.Status == TaskStatusInProgress && t.StartedAt == nil {
		t.StartedAt = &now
	}
	if t.Status.IsTerminal() && t.FinishedAt == nil {
		t.FinishedAt = &now
	}
	// Re-open to pending (retry): clear execution timestamps.
	if t.Status == TaskStatusPending && prevStatus.IsTerminal() {
		t.StartedAt = nil
		t.FinishedAt = nil
	}
}

// NormalizeEstimateDuration validates and normalizes a Go duration string (e.g. "30m", "2h").
// Empty string clears the estimate.
func NormalizeEstimateDuration(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return "", fmt.Errorf("invalid estimate duration %q: use Go duration syntax (e.g. 30m, 2h)", raw)
	}
	if d <= 0 {
		return "", fmt.Errorf("estimate duration must be positive")
	}
	return d.String(), nil
}

// TaskElapsed returns wall-clock elapsed time for a task, or zero if not started.
func TaskElapsed(t *Task, now time.Time) time.Duration {
	if t == nil || t.StartedAt == nil {
		return 0
	}
	end := now
	if t.FinishedAt != nil {
		end = *t.FinishedAt
	}
	if end.Before(*t.StartedAt) {
		return 0
	}
	return end.Sub(*t.StartedAt)
}
