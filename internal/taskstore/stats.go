package taskstore

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
)

// TaskRecord is a task with its owning project/agent queue.
type TaskRecord struct {
	Project string
	Agent   string
	Task    *entity.Task
}

// StatsWindow bounds filtering on finished_at (and optional started_at for in-progress).
type StatsWindow struct {
	From *time.Time
	To   *time.Time
}

// StatsFilter selects tasks for aggregation.
type StatsFilter struct {
	Project  string
	Agent    string
	Assignee string
	Label    string // exact label match, e.g. value:owner
	Window   StatsWindow
}

// GroupBySpec describes how to bucket stats rows.
type GroupBySpec struct {
	Mode        string // agent | assignee | label
	LabelPrefix string // when Mode=label, e.g. value: or category:
}

// ParseGroupBy parses --by: agent, assignee, label, label:value, label:category.
func ParseGroupBy(s string) (GroupBySpec, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return GroupBySpec{Mode: "agent"}, nil
	}
	if s == "agent" || s == "assignee" {
		return GroupBySpec{Mode: s}, nil
	}
	if s == "label" {
		return GroupBySpec{Mode: "label"}, nil
	}
	if strings.HasPrefix(s, "label:") {
		prefix := normalizeLabelPrefix(strings.TrimPrefix(s, "label:"))
		return GroupBySpec{Mode: "label", LabelPrefix: prefix}, nil
	}
	return GroupBySpec{}, fmt.Errorf("unknown group-by %q (use agent, assignee, label, or label:<prefix>)", s)
}

func normalizeLabelPrefix(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if !strings.Contains(p, ":") {
		p += ":"
	}
	return p
}

// TaskStatsRow is aggregated metrics for one group (agent or assignee).
type TaskStatsRow struct {
	Key string

	DoneSuccess int
	DoneFailed  int
	Cancelled   int
	OtherDone   int

	InProgressInWindow int
	PendingInWindow    int

	ElapsedSum   time.Duration
	EstimateSum  time.Duration
	ElapsedCount int
	EstimateCount int

	// Tasks with both estimate and elapsed among done_success in window.
	CompareCount int
	CompareElapsedSum  time.Duration
	CompareEstimateSum time.Duration
}

// TotalFinished returns terminal tasks finished in the window.
func (r TaskStatsRow) TotalFinished() int {
	return r.DoneSuccess + r.DoneFailed + r.Cancelled + r.OtherDone
}

// EfficiencyRatio returns actual/estimate (e.g. 0.8 = 20% faster than estimate). Zero if no data.
func (r TaskStatsRow) EfficiencyRatio() float64 {
	if r.CompareEstimateSum <= 0 {
		return 0
	}
	return float64(r.CompareElapsedSum) / float64(r.CompareEstimateSum)
}

// TaskStatsRowOut is the JSON-friendly stats row with human-readable durations.
type TaskStatsRowOut struct {
	Key string `json:"key"`

	DoneSuccess int `json:"doneSuccess"`
	DoneFailed  int `json:"doneFailed"`
	Cancelled   int `json:"cancelled"`
	OtherDone   int `json:"otherDone"`

	InProgressInWindow int `json:"inProgressInWindow,omitempty"`
	PendingInWindow    int `json:"pendingInWindow,omitempty"`

	ElapsedCount  int `json:"elapsedCount"`
	EstimateCount int `json:"estimateCount"`
	CompareCount  int `json:"compareCount"`

	ElapsedHuman         string `json:"elapsedHuman"`
	EstimateHuman        string `json:"estimateHuman"`
	AvgElapsedHuman      string `json:"avgElapsedHuman,omitempty"`
	AvgEstimateHuman     string `json:"avgEstimateHuman,omitempty"`
	CompareElapsedHuman  string `json:"compareElapsedHuman,omitempty"`
	CompareEstimateHuman string `json:"compareEstimateHuman,omitempty"`
	EfficiencyPct        string `json:"efficiencyPct,omitempty"`
}

// Out converts a stats row for JSON output.
func (r TaskStatsRow) Out() TaskStatsRowOut {
	out := TaskStatsRowOut{
		Key:                  r.Key,
		DoneSuccess:          r.DoneSuccess,
		DoneFailed:           r.DoneFailed,
		Cancelled:            r.Cancelled,
		OtherDone:            r.OtherDone,
		InProgressInWindow:   r.InProgressInWindow,
		PendingInWindow:      r.PendingInWindow,
		ElapsedCount:         r.ElapsedCount,
		EstimateCount:        r.EstimateCount,
		CompareCount:         r.CompareCount,
		ElapsedHuman:         FormatDurationCompact(r.ElapsedSum),
		EstimateHuman:        FormatDurationCompact(r.EstimateSum),
		CompareElapsedHuman:  FormatDurationCompact(r.CompareElapsedSum),
		CompareEstimateHuman: FormatDurationCompact(r.CompareEstimateSum),
	}
	if r.ElapsedCount > 0 {
		out.AvgElapsedHuman = FormatDurationCompact(r.ElapsedSum / time.Duration(r.ElapsedCount))
	}
	if r.EstimateCount > 0 {
		out.AvgEstimateHuman = FormatDurationCompact(r.EstimateSum / time.Duration(r.EstimateCount))
	}
	if r.CompareCount > 0 {
		out.EfficiencyPct = fmt.Sprintf("%.0f%%", r.EfficiencyRatio()*100)
	}
	return out
}

// FinishedTaskOut is a finished task line for JSON --detail output.
type FinishedTaskOut struct {
	ID            string   `json:"id"`
	Project       string   `json:"project"`
	Agent         string   `json:"agent"`
	Status        string   `json:"status"`
	Title         string   `json:"title"`
	Labels        []string `json:"labels,omitempty"`
	FinishedAt    string   `json:"finishedAt,omitempty"`
	ElapsedHuman  string   `json:"elapsedHuman"`
	EstimateHuman string   `json:"estimateHuman"`
}

// FinishedTaskOutFrom builds JSON detail for one finished task.
func FinishedTaskOutFrom(rec TaskRecord, now time.Time) FinishedTaskOut {
	t := rec.Task
	out := FinishedTaskOut{
		ID:           t.ID,
		Project:      rec.Project,
		Agent:        rec.Agent,
		Status:       string(t.Status),
		Title:        t.Title,
		Labels:       append([]string(nil), t.Labels...),
		ElapsedHuman: FormatDurationCompact(entity.TaskElapsed(t, now)),
	}
	if t.FinishedAt != nil {
		out.FinishedAt = t.FinishedAt.UTC().Format(time.RFC3339)
	}
	est, _ := entity.ParseEstimateDuration(t.EstimateDuration)
	out.EstimateHuman = FormatDurationCompact(est)
	return out
}

// ListAllTaskRecords returns active + archived tasks across the workspace.
func (s *FSStore) ListAllTaskRecords(projectFilter string) ([]TaskRecord, error) {
	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}
	var out []TaskRecord
	for _, proj := range projects {
		if projectFilter != "" && proj != projectFilter {
			continue
		}
		agents, err := s.ListAgents(proj)
		if err != nil {
			return nil, err
		}
		for _, ag := range agents {
			active, err := s.ListTasks(proj, ag)
			if err != nil {
				return nil, err
			}
			for _, t := range active {
				out = append(out, TaskRecord{Project: proj, Agent: ag, Task: t})
			}
			archived, err := s.ListArchivedTasks(proj, ag)
			if err != nil {
				return nil, err
			}
			for _, t := range archived {
				out = append(out, TaskRecord{Project: proj, Agent: ag, Task: t})
			}
		}
	}
	return out, nil
}

func executorKey(project, agent string) string {
	return project + "/" + agent
}

func inWindow(ts time.Time, w StatsWindow) bool {
	t := ts.UTC()
	if w.From != nil && t.Before(w.From.UTC()) {
		return false
	}
	if w.To != nil && t.After(w.To.UTC()) {
		return false
	}
	return true
}

func taskHasLabel(labels []string, exact string) bool {
	exact = strings.TrimSpace(exact)
	if exact == "" {
		return true
	}
	for _, l := range labels {
		if strings.TrimSpace(l) == exact {
			return true
		}
	}
	return false
}

func matchingLabels(labels []string, prefix string) []string {
	var out []string
	for _, l := range labels {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if prefix == "" || strings.HasPrefix(l, prefix) {
			out = append(out, l)
		}
	}
	return out
}

func matchesFilter(rec TaskRecord, f StatsFilter) bool {
	if f.Project != "" && rec.Project != f.Project {
		return false
	}
	if f.Agent != "" && rec.Agent != f.Agent {
		return false
	}
	if f.Assignee != "" && rec.Task.Assignee != f.Assignee {
		return false
	}
	if f.Label != "" && !taskHasLabel(rec.Task.Labels, f.Label) {
		return false
	}
	return true
}

func keysForRecord(rec TaskRecord, spec GroupBySpec) []string {
	switch spec.Mode {
	case "assignee":
		a := strings.TrimSpace(rec.Task.Assignee)
		if a == "" {
			a = executorKey(rec.Project, rec.Agent)
		}
		return []string{a}
	case "label":
		labels := matchingLabels(rec.Task.Labels, spec.LabelPrefix)
		if len(labels) == 0 {
			if spec.LabelPrefix != "" {
				return nil
			}
			return []string{"(no label)"}
		}
		return labels
	default:
		return []string{executorKey(rec.Project, rec.Agent)}
	}
}

func addFinishedMetrics(row *TaskStatsRow, t *entity.Task, now time.Time) {
	elapsed := entity.TaskElapsed(t, now)
	est, _ := entity.ParseEstimateDuration(t.EstimateDuration)

	switch t.Status {
	case entity.TaskStatusDoneSuccess:
		row.DoneSuccess++
		if elapsed > 0 {
			row.ElapsedSum += elapsed
			row.ElapsedCount++
		}
		if est > 0 {
			row.EstimateSum += est
			row.EstimateCount++
		}
		if elapsed > 0 && est > 0 {
			row.CompareCount++
			row.CompareElapsedSum += elapsed
			row.CompareEstimateSum += est
		}
	case entity.TaskStatusDoneFailed:
		row.DoneFailed++
	case entity.TaskStatusCancelled:
		row.Cancelled++
	default:
		if t.Status.IsTerminal() {
			row.OtherDone++
		}
	}
}

// AggregateTaskStats groups task records by agent, assignee, or label.
func AggregateTaskStats(records []TaskRecord, f StatsFilter, spec GroupBySpec) []TaskStatsRow {
	if spec.Mode == "" {
		spec.Mode = "agent"
	}

	rows := map[string]*TaskStatsRow{}

	get := func(key string) *TaskStatsRow {
		if rows[key] == nil {
			rows[key] = &TaskStatsRow{Key: key}
		}
		return rows[key]
	}

	now := time.Now().UTC()

	for _, rec := range records {
		if !matchesFilter(rec, f) {
			continue
		}
		t := rec.Task
		keys := keysForRecord(rec, spec)
		if len(keys) == 0 {
			continue
		}

		for _, key := range keys {
			row := get(key)
			if t.StartedAt != nil && inWindow(*t.StartedAt, f.Window) {
				switch t.Status {
				case entity.TaskStatusInProgress, entity.TaskStatusAwaitingConfirmation, entity.TaskStatusBlocked:
					row.InProgressInWindow++
				case entity.TaskStatusPending:
					row.PendingInWindow++
				}
			}

			if t.FinishedAt == nil || !inWindow(*t.FinishedAt, f.Window) {
				continue
			}
			addFinishedMetrics(row, t, now)
		}
	}

	out := make([]TaskStatsRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalFinished() != out[j].TotalFinished() {
			return out[i].TotalFinished() > out[j].TotalFinished()
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// FinishedTasksInWindow returns finished tasks matching filter (for --detail).
func FinishedTasksInWindow(records []TaskRecord, f StatsFilter) []TaskRecord {
	var out []TaskRecord
	for _, rec := range records {
		if !matchesFilter(rec, f) {
			continue
		}
		t := rec.Task
		if t.FinishedAt == nil || !inWindow(*t.FinishedAt, f.Window) {
			continue
		}
		if !t.Status.IsTerminal() {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Task.FinishedAt.After(*out[j].Task.FinishedAt)
	})
	return out
}

func formatDurationParts(d time.Duration, zero string) string {
	if d <= 0 {
		return zero
	}
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}

// FormatDurationHuman formats a duration for CLI table display.
func FormatDurationHuman(d time.Duration) string {
	return formatDurationParts(d, "—")
}

// FormatDurationCompact formats a duration for JSON (e.g. 14m32s, 0s when empty).
func FormatDurationCompact(d time.Duration) string {
	return formatDurationParts(d, "0s")
}
