package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/telemetry"
)

func (s *Server) handleTelemetrySummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	since := strings.TrimSpace(q.Get("since"))
	until := strings.TrimSpace(q.Get("until"))
	allTime := q.Get("allTime") == "1" || strings.EqualFold(q.Get("allTime"), "true")
	project := strings.TrimSpace(q.Get("project"))

	from, to, err := telemetry.ParseWindow(since, until, allTime, time.Now(), time.Local)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := telemetry.OpenReadOnly(s.root)
	if err != nil {
		if err == telemetry.ErrNoDatabase {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"window":    windowJSON(from, to, allTime),
				"available": false,
				"summary":   nil,
				"byAgent":   []any{},
			})
			return
		}
		s.serverError(w, err)
		return
	}
	defer db.Close()

	rows, err := telemetry.ReadRuns(db, from, to, project)
	if err != nil {
		s.serverError(w, err)
		return
	}
	sum := telemetry.Summarize(rows)
	byAgent := telemetry.SummarizeByAgent(rows)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"window":    windowJSON(from, to, allTime),
		"available": true,
		"summary":   summaryJSON(sum),
		"byAgent":   agentSummariesJSON(byAgent),
	})
}

func windowJSON(from, to *time.Time, allTime bool) map[string]any {
	out := map[string]any{"allTime": allTime}
	if from != nil {
		out["from"] = from.UTC().Format(time.RFC3339Nano)
	}
	if to != nil {
		out["to"] = to.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func summaryJSON(sum telemetry.Summary) map[string]any {
	return map[string]any{
		"runs":            sum.Runs,
		"taskRuns":        sum.TaskRuns,
		"execRuns":        sum.ExecRuns,
		"inputTokens":     sum.InputTokens,
		"outputTokens":    sum.OutputTokens,
		"cacheReadTokens": sum.CacheReadTokens,
		"costUSD":         sum.CostUSD,
		"runsWithCost":    sum.RunsWithCost,
		"success":         sum.Success,
		"failed":          sum.Failed,
		"awaiting":        sum.Awaiting,
		"other":           sum.Other,
		"wallDurationMs":  sum.WallDuration.Milliseconds(),
	}
}

func agentSummariesJSON(in []telemetry.AgentSummary) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, a := range in {
		out = append(out, map[string]any{
			"project":         a.Project,
			"agent":           a.Agent,
			"runs":            a.Runs,
			"task":            a.Task,
			"exec":            a.Exec,
			"inputTokens":     a.InputTokens,
			"outputTokens":    a.OutputTokens,
			"cacheReadTokens": a.CacheReadTokens,
			"costUSD":         a.CostUSD,
			"runsWithCost":    a.RunsWithCost,
			"success":         a.Success,
			"failed":          a.Failed,
			"awaiting":        a.Awaiting,
			"other":           a.Other,
			"wallDurationMs":  a.WallDuration.Milliseconds(),
		})
	}
	return out
}

func (s *Server) handleTelemetryRuns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	since := strings.TrimSpace(q.Get("since"))
	until := strings.TrimSpace(q.Get("until"))
	allTime := q.Get("allTime") == "1" || strings.EqualFold(q.Get("allTime"), "true")
	project := strings.TrimSpace(q.Get("project"))
	limit := 200
	if v := strings.TrimSpace(q.Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 2000 {
			limit = n
		}
	}

	from, to, err := telemetry.ParseWindow(since, until, allTime, time.Now(), time.Local)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := telemetry.OpenReadOnly(s.root)
	if err != nil {
		if err == telemetry.ErrNoDatabase {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"window":    windowJSON(from, to, allTime),
				"available": false,
				"runs":      []any{},
			})
			return
		}
		s.serverError(w, err)
		return
	}
	defer db.Close()

	rows, err := telemetry.ReadRuns(db, from, to, project)
	if err != nil {
		s.serverError(w, err)
		return
	}
	// Newest first
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}

	runOut := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m := map[string]any{
			"project":    row.Project,
			"agent":      row.Agent,
			"kind":       row.Kind,
			"status":     row.Status,
			"startedAt":  row.StartedAt.UTC().Format(time.RFC3339Nano),
			"finishedAt": row.FinishedAt.UTC().Format(time.RFC3339Nano),
			"model":      row.Model,
			"command":    row.CommandSummary,
			"logPath":    row.LogPath,
		}
		if row.TaskID.Valid && row.TaskID.String != "" {
			m["taskId"] = row.TaskID.String
		}
		if row.TaskTitle.Valid && row.TaskTitle.String != "" {
			m["taskTitle"] = row.TaskTitle.String
		}
		if row.SessionID.Valid && row.SessionID.String != "" {
			m["sessionId"] = row.SessionID.String
		}
		if row.ErrorMsg.Valid && row.ErrorMsg.String != "" {
			m["errorMsg"] = row.ErrorMsg.String
		}
		if row.InputTokens.Valid {
			m["inputTokens"] = row.InputTokens.Int64
		}
		if row.OutputTokens.Valid {
			m["outputTokens"] = row.OutputTokens.Int64
		}
		if row.CacheReadTokens.Valid {
			m["cacheReadTokens"] = row.CacheReadTokens.Int64
		}
		if row.HasCost && row.TotalCostUSD.Valid {
			m["costUSD"] = row.TotalCostUSD.Float64
		}
		if row.APIModel != "" {
			m["apiModel"] = row.APIModel
		}
		if row.APIBaseURL != "" {
			m["apiBaseUrl"] = row.APIBaseURL
		}
		runOut = append(runOut, m)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"window":    windowJSON(from, to, allTime),
		"available": true,
		"runs":      runOut,
	})
}

func (s *Server) handleTelemetryLog(w http.ResponseWriter, r *http.Request) {
	logPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if logPath == "" {
		s.jsonError(w, http.StatusBadRequest, "path is required")
		return
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.jsonError(w, http.StatusNotFound, "log file not found")
			return
		}
		s.serverError(w, err)
		return
	}
	const maxBytes = 512 * 1024
	content := string(data)
	truncated := false
	if len(data) > maxBytes {
		content = string(data[:maxBytes])
		truncated = true
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"path":      logPath,
		"content":   content,
		"truncated": truncated,
	})
}
