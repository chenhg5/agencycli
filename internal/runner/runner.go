// Package runner executes agent CLI processes for a given task,
// handles sentinel detection (confirmation requests), captures session IDs,
// and writes run logs.
//
// When the agent's SandboxConfig specifies provider=docker, execution is
// delegated to the sandbox package which wraps the agent command in
// `docker run`. Otherwise the agent CLI is invoked directly on the host.
package runner

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/sandbox"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/chenhg5/agencycli/internal/telemetry"
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

// ExecPrompt runs a raw prompt against an agent directly, bypassing the task
// queue entirely. It is intended for quick interactive testing:
//
//	agencycli exec --project p --agent a --prompt "hello"
//
// Unlike RunTask, it does NOT append the system meta footer, does NOT create
// or update any task record, and streams output directly to stdout in real
// time (as well as writing a log file).
//
// sessionID may be "" to start a fresh conversation, or a previous session ID
// to resume. The returned RunResult contains the detected session ID (if any)
// and the log path.
func (r *Runner) ExecPrompt(project, agentName, prompt, sessionID string) (*RunResult, error) {
	meta, err := r.agentStore.AgentMeta(project, agentName)
	if err != nil {
		return nil, fmt.Errorf("load agent meta: %w", err)
	}

	agentDir := filepath.Join(r.root, "projects", project, "agents", agentName)

	// HTTP agent: bypass CLI subprocess.
	if entity.NormaliseModel(meta.Model) == entity.ModelHTTPAgent {
		return r.execPromptHTTP(agentDir, meta, prompt)
	}

	// Write prompt to a temp file.
	promptFile, err := writeTempPrompt(agentDir, prompt)
	if err != nil {
		return nil, fmt.Errorf("write prompt file: %w", err)
	}
	defer os.Remove(promptFile)

	model := entity.NormaliseModel(meta.Model)
	agentEnv := resolveProviderEnv(r.root, meta)
	effectiveEnv := mergeEnv(os.Environ(), agentEnv)
	apiModel, apiBaseURL := resolveAPIModelFromEnv(model, effectiveEnv)
	invoker := InvokerFor(model, meta.RunCommand, meta.AddDirs)
	innerArgs := invoker.Args(promptFile, sessionID)

	var (
		executable string
		args       []string
		execDir    string
	)

	if meta.Sandbox != nil && meta.Sandbox.Provider == entity.SandboxDocker {
		if err := sandbox.CheckDocker(); err != nil {
			return nil, err
		}
		dockerCfg := cloneDockerCfg(meta.Sandbox.Docker)
		for _, addDir := range meta.AddDirs {
			absDir, err := filepath.Abs(addDir)
			if err != nil {
				continue
			}
			if _, err := os.Stat(absDir); err != nil {
				continue
			}
			dockerCfg.ExtraVolumes = append(dockerCfg.ExtraVolumes, absDir+":"+absDir)
		}
		if wsMount := r.root + ":" + r.root; r.root != "" {
			dockerCfg.ExtraVolumes = append(dockerCfg.ExtraVolumes, wsMount)
		}
		if binMount := resolveAgencycliBinaryMount(); binMount != "" {
			dockerCfg.ExtraVolumes = append(dockerCfg.ExtraVolumes, binMount)
		}
		containerPromptFile := agentDir + "/" + filepath.Base(promptFile)
		remappedInner := remapPromptFile(innerArgs, promptFile, containerPromptFile)
		var err error
		executable, args, err = sandbox.RunArgs(agentDir, model, dockerCfg, remappedInner)
		if err != nil {
			return nil, fmt.Errorf("sandbox: build docker args: %w", err)
		}
	} else {
		executable = innerArgs[0]
		args = innerArgs[1:]
		execDir = agentDir
	}

	// Prepare log file.
	logDir, err := r.ts.RunLogDir(project, agentName)
	if err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	logName := fmt.Sprintf("%s-exec.log", time.Now().UTC().Format("20060102-150405"))
	logPath := filepath.Join(logDir, logName)
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	defer logFile.Close()

	sandboxLabel := "host"
	if meta.Sandbox != nil && meta.Sandbox.Provider != entity.SandboxNone {
		sandboxLabel = string(meta.Sandbox.Provider)
	}
	fmt.Fprintf(logFile, "=== agencycli exec: %s/%s sandbox=%s ===\n", project, agentName, sandboxLabel)
	fmt.Fprintf(logFile, "Command: %s %s\n", executable, strings.Join(args, " "))
	fmt.Fprintf(logFile, "Started: %s\n\n", time.Now().UTC().Format(time.RFC3339))

	// Stream output to stdout AND the log file simultaneously.
	cmd := exec.Command(executable, args...)
	if execDir != "" {
		cmd.Dir = execDir
	}
	cmd.Env = effectiveEnv

	// When the invoker reads the prompt from stdin, open the prompt file and
	// pipe it through. For Docker this works because `-i` is always present in
	// the run args (see sandbox.BuildArgs), so stdin is forwarded into the
	// container transparently.
	if invoker.UseStdinPrompt() {
		pf, err := os.Open(promptFile)
		if err != nil {
			return nil, fmt.Errorf("open prompt file for stdin: %w", err)
		}
		defer pf.Close()
		cmd.Stdin = pf
	}

	var outBuf bytes.Buffer
	multiOut := io.MultiWriter(&outBuf, logFile, os.Stdout)
	cmd.Stdout = multiOut
	cmd.Stderr = multiOut

	runStarted := time.Now()
	runErr := cmd.Run()
	runFinished := time.Now()

	fmt.Fprintf(logFile, "\n=== exit code: %v  finished: %s ===\n",
		cmd.ProcessState.ExitCode(), time.Now().UTC().Format(time.RFC3339))

	output := outBuf.String()
	result := &RunResult{LogPath: logPath}

	if sid := invoker.ParseSessionID(output); sid != "" {
		result.SessionID = sid
	} else if sid := parseLineSentinel(output, SessionSentinel); sid != "" {
		result.SessionID = sid
	}

	ec := exitCodeOrZero(cmd)
	if runErr != nil {
		if sessionID != "" && isThinkingSignatureError(output) {
			fmt.Fprintf(logFile, "\n=== thinking block signature invalid — clearing heartbeat session + retrying fresh ===\n")
			r.recordAgentRun(telemetry.KindExec, project, agentName, "", "", string(model), sandboxLabel,
				apiModel, apiBaseURL,
				runStarted, runFinished, entity.TaskStatusDoneFailed, &ec, result.SessionID,
				"thinking block signature invalid, retrying fresh",
				logPath, telemetry.FormatExecCommand(executable, args), prompt, outBuf.Bytes())
			r.clearHeartbeatSession(project, agentName)
			return r.ExecPrompt(project, agentName, prompt, "")
		}
		result.Status = entity.TaskStatusDoneFailed
		result.ErrorMsg = runErr.Error()
	} else {
		result.Status = entity.TaskStatusDoneSuccess
	}
	r.recordAgentRun(telemetry.KindExec, project, agentName, "", "", string(model), sandboxLabel,
		apiModel, apiBaseURL,
		runStarted, runFinished, result.Status, &ec, result.SessionID, result.ErrorMsg,
		logPath, telemetry.FormatExecCommand(executable, args), prompt, outBuf.Bytes())
	return result, nil
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

	// HTTP agent: bypass CLI subprocess, send prompt to HTTP endpoint directly.
	if entity.NormaliseModel(meta.Model) == entity.ModelHTTPAgent {
		return r.runTaskHTTP(project, agentName, agentDir, meta, task)
	}

	// Build full prompt.
	fullPrompt := task.Prompt + fmt.Sprintf(systemMetaFooter,
		task.ID, project, agentName, task.ID, task.ID, task.ID)

	// Write prompt to a temp file (avoids shell escaping issues).
	promptFile, err := writeTempPrompt(agentDir, fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("write prompt file: %w", err)
	}
	defer os.Remove(promptFile)

	model := entity.NormaliseModel(meta.Model)
	agentEnv := resolveProviderEnv(r.root, meta)
	effectiveEnv := mergeEnv(os.Environ(), agentEnv)
	apiModel, apiBaseURL := resolveAPIModelFromEnv(model, effectiveEnv)
	invoker := InvokerFor(model, meta.RunCommand, meta.AddDirs)

	// Build the inner agent CLI arguments.
	innerArgs := invoker.Args(promptFile, sessionID)

	// Determine the actual executable and final argument list.
	// When a Docker sandbox is configured the inner args become the command
	// run inside the container; otherwise they run directly on the host.
	var (
		executable string
		args       []string
		execDir    string // working directory for the host process
	)

	if meta.Sandbox != nil && meta.Sandbox.Provider == entity.SandboxDocker {
		// Validate docker is available once before the first run.
		if err := sandbox.CheckDocker(); err != nil {
			return nil, err
		}

		// Clone the docker config so we can inject extra mounts without
		// mutating the original AgentMeta.
		dockerCfg := cloneDockerCfg(meta.Sandbox.Docker)

		// Auto-mount the project's code repository at the same absolute path
		// inside the container. This lets the agent read/write/commit code at
		// the exact path it expects (e.g. /root/code/cc-connect), matching
		// what is written in CLAUDE.md / the project prompt.
		for _, addDir := range meta.AddDirs {
			absDir, err := filepath.Abs(addDir)
			if err != nil {
				continue
			}
			if _, err := os.Stat(absDir); err != nil {
				continue
			}
			dockerCfg.ExtraVolumes = append(dockerCfg.ExtraVolumes, absDir+":"+absDir)
		}

		// Auto-mount the workspace root at the same path so agents can use
		// `agencycli task add --agent other-agent` to assign tasks to peers.
		// This enables PM agents to coordinate dev/qa agents without human
		// intervention.
		if wsMount := r.root + ":" + r.root; r.root != "" {
			dockerCfg.ExtraVolumes = append(dockerCfg.ExtraVolumes, wsMount)
		}

		// Auto-mount the agencycli binary itself (read-only) so agents can
		// invoke `agencycli` inside the container.
		if binMount := resolveAgencycliBinaryMount(); binMount != "" {
			dockerCfg.ExtraVolumes = append(dockerCfg.ExtraVolumes, binMount)
		}

		// The prompt file path inside the container.
		// innerArgs reference the host promptFile path — remap it to the real agent path.
		containerPromptFile := agentDir + "/" + filepath.Base(promptFile)
		remappedInner := remapPromptFile(innerArgs, promptFile, containerPromptFile)

		var err error
		executable, args, err = sandbox.RunArgs(agentDir, model, dockerCfg, remappedInner)
		if err != nil {
			return nil, fmt.Errorf("sandbox: build docker args: %w", err)
		}
		// docker run executes from wherever; cwd doesn't matter for container.
		execDir = ""
	} else {
		// Direct host execution.
		executable = innerArgs[0]
		args = innerArgs[1:]
		execDir = agentDir
	}

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

	sandboxLabel := "host"
	if meta.Sandbox != nil && meta.Sandbox.Provider != entity.SandboxNone {
		sandboxLabel = string(meta.Sandbox.Provider)
	}
	fmt.Fprintf(logFile, "=== agencycli run: %s/%s task=%s sandbox=%s ===\n",
		project, agentName, task.ID, sandboxLabel)
	fmt.Fprintf(logFile, "Command: %s %s\n", executable, strings.Join(args, " "))
	fmt.Fprintf(logFile, "Started: %s\n\n", time.Now().UTC().Format(time.RFC3339))

	// Run the agent.
	// Note: when UseStdinPrompt is true, we wrap in `bash -c "cmd < file"` because
	// docker's `-i` flag does not work correctly with stdin from a regular file
	// via Go's cmd.Stdin = file (docker tries to seek on stdin, which fails for regular files).
	// Using bash stdin redirect works correctly.
	var cmd *exec.Cmd
	if invoker.UseStdinPrompt() {
		shellCmd := fmt.Sprintf("%s %s < %s", executable, strings.Join(args, " "), shellEscape(promptFile))
		cmd = exec.Command("bash", "-c", shellCmd)
	} else {
		cmd = exec.Command(executable, args...)
	}
	if execDir != "" {
		cmd.Dir = execDir
	}
	cmd.Env = effectiveEnv

	var outBuf bytes.Buffer
	multiOut := io.MultiWriter(&outBuf, logFile)
	cmd.Stdout = multiOut
	cmd.Stderr = multiOut

	runStarted := time.Now()
	runErr := cmd.Run()
	runFinished := time.Now()

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

	ec := exitCodeOrZero(cmd)
	cmdSummary := telemetry.FormatExecCommand(executable, args)

	// Check for confirmation sentinel (takes priority over exit code).
	if summary := parseLineSentinel(output, ConfirmSentinel); summary != "" {
		result.Status = entity.TaskStatusAwaitingConfirmation
		result.Summary = strings.TrimSpace(summary)
		r.recordAgentRun(telemetry.KindTask, project, agentName, task.ID, task.Title, string(model), sandboxLabel,
			apiModel, apiBaseURL,
			runStarted, runFinished, result.Status, &ec, result.SessionID, result.ErrorMsg,
			logPath, cmdSummary, fullPrompt, outBuf.Bytes())
		return result, nil
	}

	if runErr != nil {
		if sessionID != "" && isThinkingSignatureError(output) {
			fmt.Fprintf(logFile, "\n=== thinking block signature invalid — clearing heartbeat session + retrying fresh ===\n")
			r.recordAgentRun(telemetry.KindTask, project, agentName, task.ID, task.Title, string(model), sandboxLabel,
				apiModel, apiBaseURL,
				runStarted, runFinished, entity.TaskStatusDoneFailed, &ec, result.SessionID,
				"thinking block signature invalid, retrying fresh",
				logPath, cmdSummary, fullPrompt, outBuf.Bytes())
			r.clearHeartbeatSession(project, agentName)
			return r.RunTask(project, agentName, task, "")
		}
		result.Status = entity.TaskStatusDoneFailed
		result.ErrorMsg = runErr.Error()
		r.recordAgentRun(telemetry.KindTask, project, agentName, task.ID, task.Title, string(model), sandboxLabel,
			apiModel, apiBaseURL,
			runStarted, runFinished, result.Status, &ec, result.SessionID, result.ErrorMsg,
			logPath, cmdSummary, fullPrompt, outBuf.Bytes())
		return result, runErr
	}

	result.Status = entity.TaskStatusDoneSuccess
	r.recordAgentRun(telemetry.KindTask, project, agentName, task.ID, task.Title, string(model), sandboxLabel,
		apiModel, apiBaseURL,
		runStarted, runFinished, result.Status, &ec, result.SessionID, result.ErrorMsg,
		logPath, cmdSummary, fullPrompt, outBuf.Bytes())
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

func isThinkingSignatureError(output string) bool {
	return strings.Contains(output, "Invalid signature in thinking block")
}

// clearHeartbeatSession zeroes the stored session ID in the heartbeat config
// so subsequent heartbeat triggers don't reuse a stale/invalid session.
func (r *Runner) clearHeartbeatSession(project, agent string) {
	hb, err := r.ts.GetHeartbeat(project, agent)
	if err != nil || hb == nil {
		return
	}
	hb.SessionID = ""
	hb.SessionStartedAt = nil
	_ = r.ts.SaveHeartbeat(project, agent, hb)
}

func exitCodeOrZero(cmd *exec.Cmd) int {
	if cmd == nil || cmd.ProcessState == nil {
		return 0
	}
	return cmd.ProcessState.ExitCode()
}

func httpCommandSummary(url string) string {
	return telemetry.TruncateCommand("HTTP POST "+strings.TrimSpace(url), 4000)
}

// recordAgentRun persists a row to .agencycli/agencycli.db; failures are ignored so runs are never blocked.
func (r *Runner) recordAgentRun(
	kind string,
	project, agent string,
	taskID, taskTitle string,
	modelNorm, sandbox string,
	apiModel, apiBaseURL string,
	started, finished time.Time,
	status entity.TaskStatus,
	exitCode *int,
	sessionID, errMsg string,
	absLogPath, cmdSummary, prompt string,
	stdout []byte,
) {
	rec := telemetry.Record{
		Kind:           kind,
		StartedAt:      started,
		FinishedAt:     finished,
		Project:        project,
		Agent:          agent,
		TaskID:         taskID,
		TaskTitle:      taskTitle,
		Model:          modelNorm,
		APIModel:       apiModel,
		APIBaseURL:     apiBaseURL,
		Sandbox:        sandbox,
		Status:         string(status),
		SessionID:      sessionID,
		ErrorMsg:       errMsg,
		LogPathRel:     telemetry.RelLogPath(r.root, absLogPath),
		CommandSummary: telemetry.TruncateCommand(cmdSummary, 4000),
	}
	if exitCode != nil {
		rec.ExitCode = sql.NullInt64{Int64: int64(*exitCode), Valid: true}
	}
	rec.PromptBytes, rec.PromptSHA256 = telemetry.PromptFingerprint(prompt)
	telemetry.ApplyStreamUsage(&rec, telemetry.ParseStreamJSONUsage(stdout))
	_ = telemetry.Insert(r.root, rec)
}

// resolveAPIModelFromEnv extracts the actual API model name and base URL
// from the effective environment for the given agent model type.
func resolveAPIModelFromEnv(modelType entity.AgentModel, env []string) (apiModel, apiBaseURL string) {
	lookup := func(keys ...string) string {
		for i := len(env) - 1; i >= 0; i-- {
			k, v, _ := strings.Cut(env[i], "=")
			for _, want := range keys {
				if k == want && v != "" {
					return v
				}
			}
		}
		return ""
	}
	switch modelType {
	case entity.ModelClaudeCode:
		apiModel = lookup("ANTHROPIC_MODEL", "CLAUDE_MODEL")
		apiBaseURL = lookup("ANTHROPIC_BASE_URL", "ANTHROPIC_API_BASE")
	case entity.ModelCodex:
		apiModel = lookup("OPENAI_MODEL", "CODEX_MODEL")
		apiBaseURL = lookup("OPENAI_BASE_URL", "OPENAI_API_BASE")
	case entity.ModelGemini:
		apiModel = lookup("GEMINI_MODEL", "GOOGLE_MODEL")
		apiBaseURL = lookup("GOOGLE_API_BASE")
	case entity.ModelCursor:
		apiModel = lookup("CURSOR_MODEL")
	case entity.ModelOpenCode:
		apiModel = lookup("OPENAI_MODEL")
		apiBaseURL = lookup("OPENAI_BASE_URL", "OPENAI_API_BASE")
	}
	return
}

// ── helpers ────────────────────────────────────────────────────────────────────

func writeTempPrompt(agentDir, content string) (string, error) {
	// Store temp prompt files in .agencycli/ to keep agent root clean.
	dir := filepath.Join(agentDir, ".agencycli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
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

// shellEscape returns a single-quoted string safe for use in a bash command.
func shellEscape(s string) string {
	// Replace ' with '\'' and wrap in single quotes.
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
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

// resolveAgencycliBinaryMount returns a read-only Docker volume mount for the
// agencycli binary running on the host, so that agent containers can invoke
// `agencycli task add`, `agencycli inbox`, etc. to coordinate with peers.
// Returns "" if the binary path cannot be determined.
func resolveAgencycliBinaryMount() string {
	binPath, err := os.Executable()
	if err != nil {
		return ""
	}
	// Resolve symlinks so we get the real binary path.
	binPath, err = filepath.EvalSymlinks(binPath)
	if err != nil {
		return ""
	}
	if _, err := os.Stat(binPath); err != nil {
		return ""
	}
	return binPath + ":" + sandbox.AgencycliMount + ":ro"
}

// cloneDockerCfg returns a shallow copy of cfg (or a fresh struct if nil)
// so callers can mutate ExtraVolumes/ExtraEnv without affecting the original.
func cloneDockerCfg(cfg *entity.DockerSandboxConfig) *entity.DockerSandboxConfig {
	if cfg == nil {
		return &entity.DockerSandboxConfig{}
	}
	cp := *cfg
	cp.ExtraVolumes = append([]string(nil), cfg.ExtraVolumes...)
	cp.ExtraEnv = append([]string(nil), cfg.ExtraEnv...)
	cp.CredentialMounts = append([]string(nil), cfg.CredentialMounts...)
	return &cp
}

// ── HTTP agent task/exec methods ───────────────────────────────────────────────

// runTaskHTTP runs a task by posting the full prompt to the agent's HTTP
// endpoint. The agent's context.md is sent as the system message; the task
// prompt + system meta footer become the user message.
func (r *Runner) runTaskHTTP(project, agentName, agentDir string, meta *entity.AgentMeta, task *entity.Task) (*RunResult, error) {
	if meta.HTTPAgent == nil {
		return nil, fmt.Errorf("http-agent: no http_agent config in .agencycli-agent.yaml (re-hire with --http-url)")
	}

	userPrompt := task.Prompt + fmt.Sprintf(systemMetaFooter,
		task.ID, project, agentName, task.ID, task.ID, task.ID)

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

	fmt.Fprintf(logFile, "=== agencycli run: %s/%s task=%s model=http-agent url=%s ===\n",
		project, agentName, task.ID, meta.HTTPAgent.URL)
	fmt.Fprintf(logFile, "Started: %s\n\n", time.Now().UTC().Format(time.RFC3339))

	systemPrompt := readAgentContextFile(agentDir)
	runStarted := time.Now()
	output, httpErr := httpExec(meta.HTTPAgent, systemPrompt, userPrompt, logFile, false)
	runFinished := time.Now()

	fmt.Fprintf(logFile, "\n=== finished: %s ===\n", time.Now().UTC().Format(time.RFC3339))

	result := &RunResult{LogPath: logPath}
	httpSummary := httpCommandSummary(meta.HTTPAgent.URL)
	modelNorm := string(entity.ModelHTTPAgent)
	sandboxLabel := "host"
	if meta.Sandbox != nil && meta.Sandbox.Provider != entity.SandboxNone {
		sandboxLabel = string(meta.Sandbox.Provider)
	}

	if httpErr != nil {
		result.Status = entity.TaskStatusDoneFailed
		result.ErrorMsg = httpErr.Error()
		r.recordAgentRun(telemetry.KindTask, project, agentName, task.ID, task.Title, modelNorm, sandboxLabel,
			"", "",
			runStarted, runFinished, result.Status, nil, result.SessionID, result.ErrorMsg,
			logPath, httpSummary, userPrompt, []byte(output))
		return result, nil
	}

	// Check for sentinels in the response text.
	if summary := parseLineSentinel(output, ConfirmSentinel); summary != "" {
		result.Status = entity.TaskStatusAwaitingConfirmation
		result.Summary = strings.TrimSpace(summary)
		r.recordAgentRun(telemetry.KindTask, project, agentName, task.ID, task.Title, modelNorm, sandboxLabel,
			"", "",
			runStarted, runFinished, result.Status, nil, result.SessionID, result.ErrorMsg,
			logPath, httpSummary, userPrompt, []byte(output))
		return result, nil
	}
	if sid := parseLineSentinel(output, SessionSentinel); sid != "" {
		result.SessionID = sid
	}

	result.Status = entity.TaskStatusDoneSuccess
	r.recordAgentRun(telemetry.KindTask, project, agentName, task.ID, task.Title, modelNorm, sandboxLabel,
		"", "",
		runStarted, runFinished, result.Status, nil, result.SessionID, result.ErrorMsg,
		logPath, httpSummary, userPrompt, []byte(output))
	return result, nil
}

// execPromptHTTP handles ExecPrompt for http-agent: sends the raw prompt to
// the HTTP endpoint and streams the response to stdout + log file.
func (r *Runner) execPromptHTTP(agentDir string, meta *entity.AgentMeta, prompt string) (*RunResult, error) {
	if meta.HTTPAgent == nil {
		return nil, fmt.Errorf("http-agent: no http_agent config in .agencycli-agent.yaml (re-hire with --http-url)")
	}

	logDir, err := r.ts.RunLogDir(meta.Project, meta.Name)
	if err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	logName := fmt.Sprintf("%s-exec.log", time.Now().UTC().Format("20060102-150405"))
	logPath := filepath.Join(logDir, logName)
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	defer logFile.Close()

	fmt.Fprintf(logFile, "=== agencycli exec: %s/%s model=http-agent url=%s ===\n",
		meta.Project, meta.Name, meta.HTTPAgent.URL)
	fmt.Fprintf(logFile, "Started: %s\n\n", time.Now().UTC().Format(time.RFC3339))

	systemPrompt := readAgentContextFile(agentDir)
	runStarted := time.Now()
	output, httpErr := httpExec(meta.HTTPAgent, systemPrompt, prompt, logFile, true)
	runFinished := time.Now()

	fmt.Fprintf(logFile, "\n=== finished: %s ===\n", time.Now().UTC().Format(time.RFC3339))

	result := &RunResult{LogPath: logPath}
	httpSummary := httpCommandSummary(meta.HTTPAgent.URL)
	modelNorm := string(entity.ModelHTTPAgent)
	sandboxLabel := "host"
	if meta.Sandbox != nil && meta.Sandbox.Provider != entity.SandboxNone {
		sandboxLabel = string(meta.Sandbox.Provider)
	}
	if httpErr != nil {
		result.Status = entity.TaskStatusDoneFailed
		result.ErrorMsg = httpErr.Error()
		r.recordAgentRun(telemetry.KindExec, meta.Project, meta.Name, "", "", modelNorm, sandboxLabel,
			"", "",
			runStarted, runFinished, result.Status, nil, result.SessionID, result.ErrorMsg,
			logPath, httpSummary, prompt, []byte(output))
		return result, nil
	}
	if sid := parseLineSentinel(output, SessionSentinel); sid != "" {
		result.SessionID = sid
	}
	result.Status = entity.TaskStatusDoneSuccess
	r.recordAgentRun(telemetry.KindExec, meta.Project, meta.Name, "", "", modelNorm, sandboxLabel,
		"", "",
		runStarted, runFinished, result.Status, nil, result.SessionID, result.ErrorMsg,
		logPath, httpSummary, prompt, []byte(output))
	return result, nil
}

// remapPromptFile replaces occurrences of hostPath with containerPath in args.
// This is needed when the prompt file is written to the host working directory
// but the container sees it at the /workspace mount point.
func remapPromptFile(args []string, hostPath, containerPath string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = strings.ReplaceAll(a, hostPath, containerPath)
	}
	return out
}

// resolveProviderEnv loads the API provider's env vars (if any) and merges
// them with the agent's per-agent env. Provider env is applied first, then
// agent env overrides, so agent-level settings always win.
func resolveProviderEnv(root string, meta *entity.AgentMeta) map[string]string {
	merged := make(map[string]string)
	if meta.Provider != "" {
		ps := store.NewProviderStore(root)
		if provEnv, err := ps.ResolveEnv(meta.Provider); err == nil {
			for k, v := range provEnv {
				merged[k] = v
			}
		}
	}
	for k, v := range meta.Env {
		merged[k] = v
	}
	return merged
}

// mergeEnv returns a copy of base with the entries in override applied.
// Keys in override replace matching keys in base (case-sensitive match on
// the part before '='); new keys are appended.
func mergeEnv(base []string, override map[string]string) []string {
	if len(override) == 0 {
		return base
	}
	overKeys := make(map[string]string, len(override))
	for k, v := range override {
		overKeys[k] = v
	}
	out := make([]string, 0, len(base)+len(override))
	seen := make(map[string]bool, len(override))
	for _, entry := range base {
		k, _, _ := strings.Cut(entry, "=")
		if v, ok := overKeys[k]; ok {
			out = append(out, k+"="+v)
			seen[k] = true
		} else {
			out = append(out, entry)
		}
	}
	for k, v := range overKeys {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}
