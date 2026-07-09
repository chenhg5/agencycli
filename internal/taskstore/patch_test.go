package taskstore

import (
	"testing"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
)

func TestApplyTaskPatchEstimateAndTimestamps(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	tk := &entity.Task{
		ID:     "t-test",
		Status: entity.TaskStatusPending,
		Title:  "hello",
		Prompt: "do it",
	}

	est := "45m"
	_, err := ApplyTaskPatch(tk, TaskPatch{
		Status:           ptrStatus(entity.TaskStatusInProgress),
		EstimateDuration: &est,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if tk.StartedAt == nil {
		t.Fatal("expected started_at")
	}
	if tk.EstimateDuration != "45m0s" {
		t.Fatalf("estimate=%q", tk.EstimateDuration)
	}
}

func ptrStatus(s entity.TaskStatus) *entity.TaskStatus { return &s }
