package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/chenhg5/agencycli/internal/telemetry"
	"github.com/spf13/cobra"
)

func newTaskStatsCmd() *cobra.Command {
	var (
		since    string
		until    string
		allTime  bool
		project  string
		agent    string
		assignee string
		label    string
		groupBy  string
		detail   bool
		format   string
	)

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Aggregate task completion and duration stats",
		Long: `Summarize tasks finished in a time window: counts, actual elapsed time,
and estimated duration. Useful for reviewing individual or team throughput.

Default window when --since/--until omitted: start of today (local) → now.
Tasks are attributed by executor queue (--by agent, default), assignee, or label (--by label:value).

Examples:
  agencycli task stats --since today
  agencycli task stats --since today --project cc-connect --agent dev-cursor
  agencycli task stats --since today --assignee cc-connect/dev-cursor
  agencycli task stats --since 7d --by assignee
  agencycli task stats --since today --by label:value
  agencycli task stats --since today --by label:category
  agencycli task stats --since today --label value:owner
  agencycli task stats --since today --detail
  agencycli task stats --since 2026-07-01 --until 2026-07-09 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskStats(since, until, allTime, project, agent, assignee, label, groupBy, detail, format)
		},
	}

	cmd.Flags().StringVar(&since, "since", "", "window start: today, yesterday, 7d, 24h, 2006-01-02, RFC3339")
	cmd.Flags().StringVar(&until, "until", "", "window end: now, today, 2006-01-02, RFC3339")
	cmd.Flags().BoolVar(&allTime, "all-time", false, "entire history (ignore --since/--until)")
	cmd.Flags().StringVar(&project, "project", "", "filter by project")
	cmd.Flags().StringVar(&agent, "agent", "", "filter by agent queue (requires --project or auto with --assignee)")
	cmd.Flags().StringVar(&assignee, "assignee", "", "filter by assignee (project/agent or human)")
	cmd.Flags().StringVar(&label, "label", "", "filter tasks with exact label (e.g. value:owner, category:health)")
	cmd.Flags().StringVar(&groupBy, "by", "agent", "group by: agent | assignee | label | label:value | label:category")
	cmd.Flags().BoolVar(&detail, "detail", false, "list each finished task in the window")
	cmd.Flags().StringVar(&format, "format", "", "output format: json")
	return cmd
}

func runTaskStats(since, until string, allTime bool, project, agent, assignee, label, groupBy string, detail bool, format string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	now := time.Now()
	from, to, err := telemetry.ParseWindow(since, until, allTime, now, time.Local)
	if err != nil {
		return err
	}

	spec, err := taskstore.ParseGroupBy(groupBy)
	if err != nil {
		return err
	}

	ts := taskstore.New(root)
	records, err := ts.ListAllTaskRecords(project)
	if err != nil {
		return err
	}

	filter := taskstore.StatsFilter{
		Project:  strings.TrimSpace(project),
		Agent:    strings.TrimSpace(agent),
		Assignee: strings.TrimSpace(assignee),
		Label:    strings.TrimSpace(label),
		Window:   taskstore.StatsWindow{From: from, To: to},
	}

	rows := taskstore.AggregateTaskStats(records, filter, spec)

	if resolveFormat(format) == "json" {
		rowOut := make([]taskstore.TaskStatsRowOut, len(rows))
		for i, r := range rows {
			rowOut[i] = r.Out()
		}
		payload := map[string]any{
			"window":  windowLabel(from, to, allTime),
			"groupBy": groupByLabel(spec),
			"rows":    rowOut,
		}
		if filter.Label != "" {
			payload["label"] = filter.Label
		}
		if detail {
			finished := taskstore.FinishedTasksInWindow(records, filter)
			tasks := make([]taskstore.FinishedTaskOut, len(finished))
			for i, rec := range finished {
				tasks[i] = taskstore.FinishedTaskOutFrom(rec, now)
			}
			payload["tasks"] = tasks
		}
		return printJSON(payload)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Task stats\t%s\n", windowLabel(from, to, allTime))
	if filter.Project != "" || filter.Agent != "" || filter.Assignee != "" || filter.Label != "" {
		var parts []string
		if filter.Project != "" {
			parts = append(parts, "project="+filter.Project)
		}
		if filter.Agent != "" {
			parts = append(parts, "agent="+filter.Agent)
		}
		if filter.Assignee != "" {
			parts = append(parts, "assignee="+filter.Assignee)
		}
		if filter.Label != "" {
			parts = append(parts, "label="+filter.Label)
		}
		fmt.Fprintf(w, "Filter\t%s\n", strings.Join(parts, " "))
	}
	fmt.Fprintf(w, "Group by\t%s\n", groupByLabel(spec))
	fmt.Fprintln(w, "")

	if len(rows) == 0 {
		fmt.Fprintln(w, "(no matching tasks)")
		_ = w.Flush()
		return nil
	}

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"SUBJECT", "SUCCESS", "FAILED", "CANCEL", "ACTUAL", "ESTIMATE", "COVER", "EFFICIENCY")
	for _, r := range rows {
		eff := "—"
		if r.CompareCount > 0 {
			pct := r.EfficiencyRatio() * 100
			eff = fmt.Sprintf("%.0f%%", pct)
		}
		cover := fmt.Sprintf("%d/%d", r.EstimateCount, r.DoneSuccess)
		if r.DoneSuccess == 0 {
			cover = "—"
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\t%s\t%s\t%s\n",
			r.Key,
			r.DoneSuccess,
			r.DoneFailed,
			r.Cancelled,
			taskstore.FormatDurationHuman(r.ElapsedSum),
			taskstore.FormatDurationHuman(r.EstimateSum),
			cover,
			eff,
		)
	}
	_ = w.Flush()

	// Totals when multiple rows or single row summary
	if len(rows) == 1 {
		printTaskStatsSummary(rows[0])
	} else {
		var total taskstore.TaskStatsRow
		total.Key = "TOTAL"
		for _, r := range rows {
			total.DoneSuccess += r.DoneSuccess
			total.DoneFailed += r.DoneFailed
			total.Cancelled += r.Cancelled
			total.ElapsedSum += r.ElapsedSum
			total.EstimateSum += r.EstimateSum
			total.ElapsedCount += r.ElapsedCount
			total.EstimateCount += r.EstimateCount
			total.CompareCount += r.CompareCount
			total.CompareElapsedSum += r.CompareElapsedSum
			total.CompareEstimateSum += r.CompareEstimateSum
		}
		fmt.Println()
		printTaskStatsSummary(total)
	}

	if detail {
		finished := taskstore.FinishedTasksInWindow(records, filter)
		if len(finished) == 0 {
			fmt.Println("\n(no finished tasks in window)")
			return nil
		}
		fmt.Println("\n── Finished tasks ──")
		dw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(dw, "%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "STATUS", "FINISHED", "ACTUAL", "ESTIMATE", "TITLE")
		for _, rec := range finished {
			t := rec.Task
			fin := ""
			if t.FinishedAt != nil {
				fin = t.FinishedAt.In(time.Local).Format("2006-01-02 15:04")
			}
			est, _ := entity.ParseEstimateDuration(t.EstimateDuration)
			fmt.Fprintf(dw, "%s\t%s\t%s\t%s\t%s\t%s\n",
				t.ID,
				t.Status,
				fin,
				taskstore.FormatDurationHuman(entity.TaskElapsed(t, now)),
				taskstore.FormatDurationHuman(est),
				truncateStr(t.Title, 48),
			)
		}
		_ = dw.Flush()
	}

	return nil
}

func printTaskStatsSummary(r taskstore.TaskStatsRow) {
	fmt.Printf("Summary · %s\n", r.Key)
	fmt.Printf("  Completed (success): %d\n", r.DoneSuccess)
	if r.DoneFailed > 0 {
		fmt.Printf("  Failed:              %d\n", r.DoneFailed)
	}
	if r.Cancelled > 0 {
		fmt.Printf("  Cancelled:           %d\n", r.Cancelled)
	}
	fmt.Printf("  Actual time:         %s", taskstore.FormatDurationHuman(r.ElapsedSum))
	if r.ElapsedCount > 0 && r.DoneSuccess > 0 {
		avg := r.ElapsedSum / time.Duration(r.ElapsedCount)
		fmt.Printf("  (avg %s/task)", taskstore.FormatDurationHuman(avg))
	}
	fmt.Println()
	fmt.Printf("  Estimated time:      %s", taskstore.FormatDurationHuman(r.EstimateSum))
	if r.EstimateCount > 0 {
		avg := r.EstimateSum / time.Duration(r.EstimateCount)
		fmt.Printf("  (%d tasks with estimate, avg %s)", r.EstimateCount, taskstore.FormatDurationHuman(avg))
	}
	fmt.Println()
	if r.CompareCount > 0 {
		pct := r.EfficiencyRatio() * 100
		fmt.Printf("  Actual/estimate:     %.0f%%", pct)
		if pct < 100 {
			fmt.Printf("  (faster than estimate)")
		} else if pct > 100 {
			fmt.Printf("  (slower than estimate)")
		}
		fmt.Println()
	}
}

func truncateStr(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func groupByLabel(spec taskstore.GroupBySpec) string {
	switch spec.Mode {
	case "label":
		if spec.LabelPrefix != "" {
			return "label:" + strings.TrimSuffix(spec.LabelPrefix, ":")
		}
		return "label"
	default:
		return spec.Mode
	}
}
