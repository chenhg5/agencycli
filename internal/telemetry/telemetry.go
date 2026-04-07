// Package telemetry persists agent invocation records to a workspace-local SQLite DB.
package telemetry

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

func sqlInt64Ptr(n sql.NullInt64) interface{} {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func sqlFloat64Ptr(n sql.NullFloat64) interface{} {
	if !n.Valid {
		return nil
	}
	return n.Float64
}

// ApplyStreamUsage fills token/cost SQL fields when stream-json contained a "result" line.
func ApplyStreamUsage(rec *Record, u StreamUsage) {
	if !u.SawResult {
		return
	}
	rec.InputTokens = sql.NullInt64{Int64: u.InputTokens, Valid: true}
	rec.OutputTokens = sql.NullInt64{Int64: u.OutputTokens, Valid: true}
	rec.CacheReadTokens = sql.NullInt64{Int64: u.CacheReadTokens, Valid: true}
	rec.TotalCostUSD = sql.NullFloat64{Float64: u.TotalCostUSD, Valid: true}
	rec.HasCost = true
}

const (
	KindExec = "exec"
	KindTask = "task"
)

// Record is one agent run (exec or task queue).
type Record struct {
	Kind       string
	StartedAt  time.Time
	FinishedAt time.Time

	Project string
	Agent   string

	TaskID    string
	TaskTitle string

	Model   string
	Sandbox string

	Status   string
	ExitCode sql.NullInt64

	SessionID string
	ErrorMsg  string

	LogPathRel string // relative to workspace root

	CommandSummary string

	APIModel   string
	APIBaseURL string

	PromptBytes int64
	PromptSHA256  string

	InputTokens     sql.NullInt64
	OutputTokens    sql.NullInt64
	CacheReadTokens sql.NullInt64
	TotalCostUSD    sql.NullFloat64
	HasCost         bool
}

func dbPath(root string) string {
	return filepath.Join(root, ".agencycli", "agencycli.db")
}

// RelLogPath returns logPath as a path relative to root, or the original if Rel fails.
func RelLogPath(root, logPath string) string {
	if root == "" || logPath == "" {
		return logPath
	}
	rel, err := filepath.Rel(root, logPath)
	if err != nil {
		return logPath
	}
	return rel
}

func openDB(root string) (*sql.DB, error) {
	p := dbPath(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return nil, err
	}
	uri := "file:" + filepath.ToSlash(p) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS agent_runs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at TEXT NOT NULL,
	started_at TEXT NOT NULL,
	finished_at TEXT NOT NULL,
	kind TEXT NOT NULL,
	project TEXT NOT NULL,
	agent TEXT NOT NULL,
	task_id TEXT,
	task_title TEXT,
	model TEXT NOT NULL,
	sandbox TEXT NOT NULL,
	status TEXT NOT NULL,
	exit_code INTEGER,
	session_id TEXT,
	error_msg TEXT,
	log_path TEXT NOT NULL,
	command_summary TEXT NOT NULL,
	prompt_bytes INTEGER NOT NULL,
	prompt_sha256 TEXT NOT NULL,
	input_tokens INTEGER,
	output_tokens INTEGER,
	cache_read_tokens INTEGER,
	total_cost_usd REAL,
	has_cost INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_runs_created ON agent_runs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_runs_project_agent ON agent_runs(project, agent)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_runs_task_id ON agent_runs(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_runs_started ON agent_runs(started_at)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	// Schema evolution: add columns that may not exist in older databases.
	for _, alter := range []string{
		`ALTER TABLE agent_runs ADD COLUMN api_model TEXT DEFAULT ''`,
		`ALTER TABLE agent_runs ADD COLUMN api_base_url TEXT DEFAULT ''`,
	} {
		_, _ = db.Exec(alter)
	}
	return nil
}

// Insert writes one row. Errors are returned to the caller; runner ignores them.
func Insert(root string, rec Record) error {
	db, err := openDB(root)
	if err != nil {
		return err
	}
	defer db.Close()

	hc := 0
	if rec.HasCost {
		hc = 1
	}

	_, err = db.Exec(`
INSERT INTO agent_runs (
	created_at, started_at, finished_at, kind, project, agent,
	task_id, task_title, model, sandbox, status, exit_code,
	session_id, error_msg, log_path, command_summary,
	prompt_bytes, prompt_sha256,
	input_tokens, output_tokens, cache_read_tokens, total_cost_usd, has_cost,
	api_model, api_base_url
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano),
		rec.StartedAt.UTC().Format(time.RFC3339Nano),
		rec.FinishedAt.UTC().Format(time.RFC3339Nano),
		rec.Kind, rec.Project, rec.Agent,
		nullIfEmpty(rec.TaskID), nullIfEmpty(rec.TaskTitle),
		rec.Model, rec.Sandbox, rec.Status, sqlInt64Ptr(rec.ExitCode),
		nullIfEmpty(rec.SessionID), nullIfEmpty(rec.ErrorMsg),
		rec.LogPathRel, rec.CommandSummary,
		rec.PromptBytes, rec.PromptSHA256,
		sqlInt64Ptr(rec.InputTokens), sqlInt64Ptr(rec.OutputTokens), sqlInt64Ptr(rec.CacheReadTokens), sqlFloat64Ptr(rec.TotalCostUSD), hc,
		rec.APIModel, rec.APIBaseURL,
	)
	return err
}

func nullIfEmpty(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// PromptFingerprint returns byte length and hex SHA256 of the prompt (UTF-8).
func PromptFingerprint(prompt string) (bytes int64, sha256hex string) {
	b := []byte(prompt)
	h := sha256.Sum256(b)
	return int64(len(b)), hex.EncodeToString(h[:])
}

// TruncateCommand limits command summary size for SQLite rows.
func TruncateCommand(s string, maxRunes int) string {
	if maxRunes <= 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= maxRunes-3 {
			b.WriteString("...")
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}

// FormatExecCommand builds "executable arg1 arg2 ...".
func FormatExecCommand(executable string, args []string) string {
	if len(args) == 0 {
		return executable
	}
	return fmt.Sprintf("%s %s", executable, strings.Join(args, " "))
}
