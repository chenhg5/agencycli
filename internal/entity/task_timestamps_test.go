package entity

import (
	"testing"
	"time"
)

func TestApplyStatusTimestamps(t *testing.T) {
	now := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)

	t.Run("in_progress sets started", func(t *testing.T) {
		task := &Task{Status: TaskStatusInProgress}
		ApplyStatusTimestamps(task, TaskStatusPending, now)
		if task.StartedAt == nil || !task.StartedAt.Equal(now) {
			t.Fatalf("StartedAt = %v, want %v", task.StartedAt, now)
		}
		if task.FinishedAt != nil {
			t.Fatalf("FinishedAt should be nil")
		}
	})

	t.Run("terminal sets finished", func(t *testing.T) {
		started := now.Add(-time.Hour)
		task := &Task{Status: TaskStatusDoneSuccess, StartedAt: &started}
		ApplyStatusTimestamps(task, TaskStatusInProgress, now)
		if task.FinishedAt == nil || !task.FinishedAt.Equal(now) {
			t.Fatalf("FinishedAt = %v, want %v", task.FinishedAt, now)
		}
	})

	t.Run("retry from terminal clears timestamps", func(t *testing.T) {
		task := &Task{
			Status:     TaskStatusPending,
			StartedAt:  &now,
			FinishedAt: &now,
		}
		ApplyStatusTimestamps(task, TaskStatusDoneFailed, now)
		if task.StartedAt != nil || task.FinishedAt != nil {
			t.Fatalf("expected cleared timestamps, got started=%v finished=%v", task.StartedAt, task.FinishedAt)
		}
	})
}

func TestNormalizeEstimateDuration(t *testing.T) {
	got, err := NormalizeEstimateDuration("90m")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1h30m0s" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeEstimateDuration("nope"); err == nil {
		t.Fatal("expected error")
	}
}
