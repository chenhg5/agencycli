// Package runner executes agent CLI processes for a given task,
// handles sentinel detection (confirmation requests), captures session IDs,
// and writes run logs.
package runner

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agencycli/agencycli/internal/entity"
	"github.com/agencycli/agencycli/internal/store"
	"github.com/agencycli/agencycli/internal/taskstore"
)

const (
	// ConfirmSentinel is the magic prefix agents write to stdout to request
	// human confirmation. Everything after the colon is used as the summary.
	ConfirmSentinel = "AGENCYCLI_AWAIT_CONFIRM:"

	// SessionSentinel lets agents explicitly report a new session ID.
	SessionSentinel = "AGENCYCLI_SESSION_ID:"
)

// systemMetaFooter is appended to every task prompt so agents know how to
// call back into agencycli to update task state.
const systemMetaFooter = `

---
## System Metadata (do not modify)

Task ID : %s
Agent   : %s/%s

When complete successfully, run:
  agencycli task done --id %s --status success

If human confirmation needed, run:
  agencycli task confirm-request --id %s --summary "one-line explanation"
  (then exit 0)

If unable to complete, run:
  agencycli task done --id %s --status failed --error "reason"
`

// Runner executes tasks for agents using their configured CLI.
type Runner struct {
	root       string
	ts         taskstore.Store
	agentStore store.Store
}

// New creates a Runner. root is the workspace root.
func New(root string, ts taskstore.Store, as store.Store) *Runner {
	return &Runner{root: root, ts: ts, agentStore: as}
}

// RunResult holds the outcome of a single task execution.
type RunResult struct {
	Status     entity.TaskStatus
	SessionID  string
	LogPath    string
	Summary    string // set when Status == TaskStatusAwaitingConfirmation
	ErrorMsg   string // set when Status == TaskStatusDoneFailed
}

// RunTask executes a single task in the context of the given agent.
// It handles:
//   - building the full prompt (task prompt + system footer)
//   - invoking the agent CLI (with optional session resume)
//   - detecting AGENCYCLI_AWAIT_CONFIRM sentinel
//   - capturing session ID from output
//   - writing the run log
//
// It does NOT update task state in tasks.yaml — callers are responsible
// for calling ts.UpdateTask / ts.ArchiveTask based on the returned RunResult.
func (r *Runner) RunTask(project, agentName string, task *entity.Task, sessionID string) (*RunResult, error) {
	meta, err := r.agentStore.AgentMeta(project, agentName)
	if err != nil {
		return nil, fmt.Errorf("load agent meta: %w", err)
	}

	agentDir := filepath.Join(r.root, "projects", project, "agents", agentName)

	// Build full prompt.
	fullPrompt := task.Prompt + fmt.Sprintf(systemMetaFooter,
		task.ID, project, agentName, task.ID, task.ID, task.ID)

	// Write prompt to a temp file (avoids shell escaping issues).
	promptFile, err := writeTempPrompt(agentDir, fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("write prompt file: %w", err)
	}
	defer os.Remove(promptFile)

	invoker := InvokerFor(entity.NormaliseModel(meta.Model), meta.RunCommand)
	args := invoker.Args(promptFile, sessionID)

	// Prepare log file.
	logDir, err := r.ts.RunLogDir(project, agentName)
	if err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	logName := fmt.Sprintf("%s-%s.log", time.Now().UTC().Format("20060102-150405"), task.ID)
	logPath := filepath.Join(logDir, logName)
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	defer logFile.Close()

	fmt.Fprintf(logFile, "=== agencycli run: %s/%s task=%s ===\n", project, agentName, task.ID)
	fmt.Fprintf(logFile, "Command: %s\n", strings.Join(args, " "))
	fmt.Fprintf(logFile, "Started: %s\n\n", time.Now().UTC().Format(time.RFC3339))

	// Run the agent.
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = agentDir

	var outBuf bytes.Buffer
	multiOut := io.MultiWriter(&outBuf, logFile)
	cmd.Stdout = multiOut
	cmd.Stderr = multiOut

	runErr := cmd.Run()

	fmt.Fprintf(logFile, "\n=== exit code: %v  finished: %s ===\n",
		cmd.ProcessState.ExitCode(), time.Now().UTC().Format(time.RFC3339))

	output := outBuf.String()
	result := &RunResult{LogPath: logPath}

	// Parse session ID (model-specific + universal sentinel).
	if sid := invoker.ParseSessionID(output); sid != "" {
		result.SessionID = sid
	}
	if sid := parseLineSentinel(output, SessionSentinel); sid != "" {
		result.SessionID = sid
	}

	// Check for confirmation sentinel (takes priority over exit code).
	if summary := parseLineSentinel(output, ConfirmSentinel); summary != "" {
		result.Status = entity.TaskStatusAwaitingConfirmation
		result.Summary = strings.TrimSpace(summary)
		return result, nil
	}

	if runErr != nil {
		result.Status = entity.TaskStatusDoneFailed
		result.ErrorMsg = runErr.Error()
		return result, nil
	}

	result.Status = entity.TaskStatusDoneSuccess
	return result, nil
}

// ResumeTask re-invokes the agent after human confirmation.
// The original task prompt is extended with the confirmation reply.
func (r *Runner) ResumeTask(project, agentName string, task *entity.Task, confirmReply, sessionID string) (*RunResult, error) {
	original := task.Prompt
	task.Prompt = original + "\n\n---\n[Human confirmed at " +
		time.Now().UTC().Format(time.RFC3339) + "]\n" +
		confirmReply + "\n\nPlease continue from where you left off.\n"
	result, err := r.RunTask(project, agentName, task, sessionID)
	task.Prompt = original // restore
	return result, err
}

// ── helpers ────────────────────────────────────────────────────────────────────

func writeTempPrompt(dir, content string) (string, error) {
	f, err := os.CreateTemp(dir, ".prompt-*.txt")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// parseLineSentinel scans output line by line for lines starting with prefix.
// Returns everything after the prefix on the first matching line.
func parseLineSentinel(output, prefix string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if after, ok := strings.CutPrefix(line, prefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
