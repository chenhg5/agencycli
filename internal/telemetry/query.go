package telemetry

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

// ErrNoDatabase means .agencycli/agencycli.db is missing or empty.
var ErrNoDatabase = errors.New("telemetry database not found (no agent runs recorded yet)")

// RunRow is one row from agent_runs for aggregation.
type RunRow struct {
	Project, Agent string
	Kind, Status   string

	InputTokens, OutputTokens, CacheReadTokens sql.NullInt64
	TotalCostUSD                               sql.NullFloat64
	HasCost                                    bool

	StartedAt, FinishedAt time.Time

	TaskID, TaskTitle   sql.NullString
	Model               string
	CommandSummary      string
	LogPath             string
	SessionID           sql.NullString
	ErrorMsg            sql.NullString
}

// Summary aggregates for a time window.
type Summary struct {
	Runs int64

	TaskRuns int64
	ExecRuns int64

	InputTokens, OutputTokens, CacheReadTokens int64
	CostUSD                                    float64
	RunsWithCost                               int64

	Success, Failed, Awaiting, Other int64

	WallDuration time.Duration
}

// AgentSummary is per-(project, agent) aggregates.
type AgentSummary struct {
	Project, Agent string

	Runs int64
	Task int64
	Exec int64

	InputTokens, OutputTokens, CacheReadTokens int64
	CostUSD                                    float64
	RunsWithCost                               int64

	Success, Failed, Awaiting, Other int64
	WallDuration                     time.Duration
}

// OpenReadOnly opens the telemetry DB read-only. Returns ErrNoDatabase if the file is absent or zero-length.
func OpenReadOnly(root string) (*sql.DB, error) {
	p := dbPath(root)
	st, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoDatabase
		}
		return nil, err
	}
	if st.Size() == 0 {
		return nil, ErrNoDatabase
	}
	uri := "file:" + filepath.ToSlash(p) + "?mode=ro&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// ReadRuns returns all runs in the window (filter on started_at). Nil from/to means unbounded on that side.
func ReadRuns(db *sql.DB, from, to *time.Time, project string) ([]RunRow, error) {
	q := `SELECT project, agent, kind, status,
		input_tokens, output_tokens, cache_read_tokens, total_cost_usd, has_cost,
		started_at, finished_at,
		task_id, task_title, model, command_summary, log_path, session_id, error_msg
	FROM agent_runs WHERE 1=1`
	var args []any
	if from != nil {
		q += ` AND started_at >= ?`
		args = append(args, from.UTC().Format(time.RFC3339Nano))
	}
	if to != nil {
		q += ` AND started_at <= ?`
		args = append(args, to.UTC().Format(time.RFC3339Nano))
	}
	if project != "" {
		q += ` AND project = ?`
		args = append(args, project)
	}
	q += ` ORDER BY started_at ASC`

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RunRow
	for rows.Next() {
		var r RunRow
		var inTok, outTok, cacheTok sql.NullInt64
		var cost sql.NullFloat64
		var hasCost int
		var startedStr, finishedStr string
		if err := rows.Scan(
			&r.Project, &r.Agent, &r.Kind, &r.Status,
			&inTok, &outTok, &cacheTok, &cost, &hasCost,
			&startedStr, &finishedStr,
			&r.TaskID, &r.TaskTitle, &r.Model, &r.CommandSummary, &r.LogPath, &r.SessionID, &r.ErrorMsg,
		); err != nil {
			return nil, err
		}
		r.InputTokens = inTok
		r.OutputTokens = outTok
		r.CacheReadTokens = cacheTok
		r.TotalCostUSD = cost
		r.HasCost = hasCost != 0
		st, err := parseDBTime(startedStr)
		if err != nil {
			continue
		}
		r.StartedAt = st
		ft, err := parseDBTime(finishedStr)
		if err != nil {
			continue
		}
		r.FinishedAt = ft
		out = append(out, r)
	}
	return out, rows.Err()
}

func parseDBTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

// Summarize builds workspace-level aggregates.
func Summarize(rows []RunRow) Summary {
	var s Summary
	for _, r := range rows {
		s.Runs++
		switch r.Kind {
		case KindTask:
			s.TaskRuns++
		case KindExec:
			s.ExecRuns++
		}
		if r.InputTokens.Valid {
			s.InputTokens += r.InputTokens.Int64
		}
		if r.OutputTokens.Valid {
			s.OutputTokens += r.OutputTokens.Int64
		}
		if r.CacheReadTokens.Valid {
			s.CacheReadTokens += r.CacheReadTokens.Int64
		}
		if r.HasCost && r.TotalCostUSD.Valid {
			s.CostUSD += r.TotalCostUSD.Float64
			s.RunsWithCost++
		}
		switch r.Status {
		case "done_success":
			s.Success++
		case "done_failed":
			s.Failed++
		case "awaiting_confirmation":
			s.Awaiting++
		default:
			s.Other++
		}
		d := r.FinishedAt.Sub(r.StartedAt)
		if d > 0 {
			s.WallDuration += d
		}
	}
	return s
}

type pairKey struct{ p, a string }

// SummarizeByAgent groups rows by project and agent.
func SummarizeByAgent(rows []RunRow) []AgentSummary {
	m := make(map[pairKey]*AgentSummary)
	for _, r := range rows {
		k := pairKey{r.Project, r.Agent}
		ag := m[k]
		if ag == nil {
			ag = &AgentSummary{Project: r.Project, Agent: r.Agent}
			m[k] = ag
		}
		ag.Runs++
		switch r.Kind {
		case KindTask:
			ag.Task++
		case KindExec:
			ag.Exec++
		}
		if r.InputTokens.Valid {
			ag.InputTokens += r.InputTokens.Int64
		}
		if r.OutputTokens.Valid {
			ag.OutputTokens += r.OutputTokens.Int64
		}
		if r.CacheReadTokens.Valid {
			ag.CacheReadTokens += r.CacheReadTokens.Int64
		}
		if r.HasCost && r.TotalCostUSD.Valid {
			ag.CostUSD += r.TotalCostUSD.Float64
			ag.RunsWithCost++
		}
		switch r.Status {
		case "done_success":
			ag.Success++
		case "done_failed":
			ag.Failed++
		case "awaiting_confirmation":
			ag.Awaiting++
		default:
			ag.Other++
		}
		d := r.FinishedAt.Sub(r.StartedAt)
		if d > 0 {
			ag.WallDuration += d
		}
	}
	out := make([]AgentSummary, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Project != out[j].Project {
			return out[i].Project < out[j].Project
		}
		return out[i].Agent < out[j].Agent
	})
	return out
}
