package taskstore

import (
	"testing"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
)

func TestAggregateTaskStats(t *testing.T) {
	now := time.Date(2026, 7, 9, 18, 0, 0, 0, time.UTC)
	start := now.Add(-2 * time.Hour)
	finish := now.Add(-30 * time.Minute)
	from := now.Add(-24 * time.Hour)

	records := []TaskRecord{{
		Project: "p", Agent: "dev",
		Task: &entity.Task{
			ID: "t1", Status: entity.TaskStatusDoneSuccess,
			Assignee: "p/dev", EstimateDuration: "1h0m0s",
			StartedAt: &start, FinishedAt: &finish,
		},
	}, {
		Project: "p", Agent: "dev",
		Task: &entity.Task{
			ID: "t2", Status: entity.TaskStatusDoneFailed,
			Assignee: "p/dev", FinishedAt: &finish,
		},
	}}

	rows := AggregateTaskStats(records, StatsFilter{
		Window: StatsWindow{From: &from, To: &now},
	}, GroupBySpec{Mode: "agent"})
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].DoneSuccess != 1 || rows[0].DoneFailed != 1 {
		t.Fatalf("counts=%+v", rows[0])
	}
	if rows[0].ElapsedCount != 1 || rows[0].EstimateCount != 1 {
		t.Fatalf("elapsed/estimate counts=%+v", rows[0])
	}
}

func TestAggregateTaskStatsByLabel(t *testing.T) {
	now := time.Date(2026, 7, 9, 18, 0, 0, 0, time.UTC)
	finish := now.Add(-30 * time.Minute)
	from := now.Add(-24 * time.Hour)

	records := []TaskRecord{
		{
			Project: "p", Agent: "dev",
			Task: &entity.Task{
				ID: "t1", Status: entity.TaskStatusDoneSuccess,
				Labels: []string{"value:owner", "category:health"},
				FinishedAt: &finish,
			},
		},
		{
			Project: "p", Agent: "dev",
			Task: &entity.Task{
				ID: "t2", Status: entity.TaskStatusDoneSuccess,
				Labels: []string{"value:fulltime-company", "category:social"},
				FinishedAt: &finish,
			},
		},
		{
			Project: "p", Agent: "dev",
			Task: &entity.Task{
				ID: "t3", Status: entity.TaskStatusDoneSuccess,
				FinishedAt: &finish,
			},
		},
	}

	valueRows := AggregateTaskStats(records, StatsFilter{
		Window: StatsWindow{From: &from, To: &now},
	}, GroupBySpec{Mode: "label", LabelPrefix: "value:"})
	if len(valueRows) != 2 {
		t.Fatalf("value rows=%d want 2", len(valueRows))
	}
	keys := map[string]int{}
	for _, r := range valueRows {
		keys[r.Key] = r.DoneSuccess
	}
	if keys["value:owner"] != 1 || keys["value:fulltime-company"] != 1 {
		t.Fatalf("value keys=%v", keys)
	}

	filtered := AggregateTaskStats(records, StatsFilter{
		Label:  "value:owner",
		Window: StatsWindow{From: &from, To: &now},
	}, GroupBySpec{Mode: "agent"})
	if len(filtered) != 1 || filtered[0].DoneSuccess != 1 {
		t.Fatalf("label filter=%+v", filtered)
	}
}

func TestTaskStatsRowOutHuman(t *testing.T) {
	row := TaskStatsRow{
		Key:         "value:owner",
		DoneSuccess: 2,
		ElapsedSum:  14*time.Minute + 32*time.Second,
		ElapsedCount: 2,
		EstimateSum: time.Hour,
		EstimateCount: 1,
		CompareCount: 1,
		CompareElapsedSum: 30 * time.Minute,
		CompareEstimateSum: time.Hour,
	}
	out := row.Out()
	if out.ElapsedHuman != "14m32s" {
		t.Fatalf("elapsedHuman=%q", out.ElapsedHuman)
	}
	if out.AvgElapsedHuman != "7m16s" {
		t.Fatalf("avgElapsedHuman=%q", out.AvgElapsedHuman)
	}
	if out.EfficiencyPct != "50%" {
		t.Fatalf("efficiencyPct=%q", out.EfficiencyPct)
	}
}

func TestParseGroupBy(t *testing.T) {
	spec, err := ParseGroupBy("label:value")
	if err != nil || spec.Mode != "label" || spec.LabelPrefix != "value:" {
		t.Fatalf("spec=%+v err=%v", spec, err)
	}
	spec, err = ParseGroupBy("label:category")
	if err != nil || spec.LabelPrefix != "category:" {
		t.Fatalf("category spec=%+v", spec)
	}
}
