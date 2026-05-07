package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/telemetry"
)

type agentChatBody struct {
	Message   string `json:"message"`
	SessionID string `json:"sessionId"`
	NoSession bool   `json:"noSession,omitempty"`
}

type agentChatHistoryRun struct {
	StartedAt string `json:"startedAt"`
	Status    string `json:"status"`
	LogPath   string `json:"logPath"`
}

func (s *Server) handleAgentChatHistory(w http.ResponseWriter, r *http.Request) {
	project, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	if !s.checkProjectAccess(w, r, project) {
		return
	}

	sessionID := strings.TrimSpace(r.URL.Query().Get("sessionId"))
	if sessionID == "" {
		if hb, err := s.ts.GetHeartbeat(project, agent); err == nil && hb.SessionID != "" {
			sessionID = hb.SessionID
		}
	}

	resolvedSessionID := sessionID
	content := ""
	truncated := false
	runs := []agentChatHistoryRun{}
	if sessionID != "" {
		var err error
		content, runs, resolvedSessionID, truncated, err = s.readAgentSessionHistory(project, agent, sessionID)
		if err != nil {
			s.serverError(w, err)
			return
		}
	} else {
		var err error
		content, runs, resolvedSessionID, truncated, err = s.readAgentSessionHistory(project, agent, "")
		if err != nil {
			s.serverError(w, err)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"sessionId": resolvedSessionID,
		"content":   content,
		"runs":      runs,
		"truncated": truncated,
	})
}

func (s *Server) readAgentSessionHistory(project, agent, sessionID string) (string, []agentChatHistoryRun, string, bool, error) {
	db, err := telemetry.OpenReadOnly(s.root)
	if err != nil {
		if err == telemetry.ErrNoDatabase {
			return "", []agentChatHistoryRun{}, sessionID, false, nil
		}
		return "", nil, sessionID, false, err
	}
	defer db.Close()

	rows, err := telemetry.ReadRuns(db, nil, nil, project)
	if err != nil {
		return "", nil, sessionID, false, err
	}

	const maxRuns = 8
	filtered := make([]telemetry.RunRow, 0, maxRuns)
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		if row.Agent != agent || row.LogPath == "" {
			continue
		}
		if sessionID != "" && (!row.SessionID.Valid || row.SessionID.String != sessionID) {
			continue
		}
		filtered = append(filtered, row)
		if len(filtered) >= maxRuns {
			break
		}
	}
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	type historySegment struct {
		row     telemetry.RunRow
		logPath string
		data    []byte
	}

	segments := make([]historySegment, 0, len(filtered))
	truncated := false
	for _, row := range filtered {
		logPath := row.LogPath
		absLogPath := logPath
		if !filepath.IsAbs(absLogPath) {
			absLogPath = filepath.Join(s.root, absLogPath)
		}
		data, err := os.ReadFile(absLogPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", nil, sessionID, false, err
		}
		if sessionID == "" {
			if row.SessionID.Valid && row.SessionID.String != "" {
				sessionID = row.SessionID.String
			} else if sid := extractAgentChatSessionID(string(data)); sid != "" {
				sessionID = sid
			}
		}
		segments = append(segments, historySegment{
			row:     row,
			logPath: logPath,
			data:    data,
		})
	}

	const maxBytes = 768 * 1024
	total := 0
	selected := make([]historySegment, 0, len(segments))
	for i := len(segments) - 1; i >= 0; i-- {
		seg := segments[i]
		if total+len(seg.data) > maxBytes {
			remaining := maxBytes - total
			if remaining <= 0 {
				truncated = true
				break
			}
			seg.data = append([]byte("=== earlier log content truncated ===\n"), seg.data[len(seg.data)-remaining:]...)
			truncated = true
		}
		selected = append([]historySegment{seg}, selected...)
		total += len(seg.data)
		if truncated {
			break
		}
	}

	var sb strings.Builder
	outRuns := make([]agentChatHistoryRun, 0, len(selected))
	for _, seg := range selected {
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.Write(seg.data)
		outRuns = append(outRuns, agentChatHistoryRun{
			StartedAt: seg.row.StartedAt.UTC().Format(time.RFC3339Nano),
			Status:    seg.row.Status,
			LogPath:   seg.logPath,
		})
	}
	return sb.String(), outRuns, sessionID, truncated, nil
}

func (s *Server) handleAgentChat(w http.ResponseWriter, r *http.Request) {
	project, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	if !s.checkProjectAccess(w, r, project) {
		return
	}

	var body agentChatBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	msg := strings.TrimSpace(body.Message)
	if msg == "" {
		s.jsonError(w, http.StatusBadRequest, "message is required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.jsonError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// If a process is already running for this project/agent, kill it first.
	key := project + "/" + agent
	s.execMu.Lock()
	if existing, ok := s.execProcs[key]; ok {
		if existing.cmd.Process != nil {
			killProcessGroup(existing.cmd.Process.Pid)
		}
	}
	s.execProcs[key] = nil // placeholder; will be replaced after cmd.Start
	s.execMu.Unlock()

	args := []string{"--dir", s.root, "exec", "--project", project, "--agent", agent, "--prompt", msg}
	sessionID := strings.TrimSpace(body.SessionID)
	if sessionID != "" && !body.NoSession {
		args = append(args, "--session", sessionID)
	}
	if body.NoSession {
		args = append(args, "--no-session")
	}

	// Do not bind the child process to the HTTP request context. The browser
	// aborts fetches when navigating away; killing the agent at that point would
	// prevent run logs and telemetry from being recorded.
	cmd := exec.Command(s.sched.binPath, args...)
	cmd.Dir = s.root
	setProcGroup(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.serverError(w, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.serverError(w, err)
		return
	}
	if err := cmd.Start(); err != nil {
		s.serverError(w, err)
		return
	}

	// Register the running process so it can be stopped via the /chat DELETE endpoint.
	s.execMu.Lock()
	s.execProcs[key] = &execProcess{cmd: cmd, started: time.Now()}
	s.execMu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	lines := make(chan string, 64)
	var wg sync.WaitGroup
	scan := func(src io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(src)
		scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), "\r")
			if line != "" {
				lines <- line
			}
		}
	}
	wg.Add(2)
	go scan(stdout)
	go scan(stderr)
	go func() {
		wg.Wait()
		close(lines)
	}()

	detectedSessionID := sessionID
	agentModel := entity.AgentModel("")
	if meta, err := s.st.AgentMeta(project, agent); err == nil && meta != nil {
		agentModel = meta.Model
	}
	clientGone := false
	for line := range lines {
		if sid := extractAgentChatSessionID(line); sid != "" {
			detectedSessionID = sid
		}
		if clientGone {
			continue
		}
		payload := chatSSELine(line, agentModel)
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			clientGone = true
			continue
		}
		flusher.Flush()
	}

	waitErr := cmd.Wait()

	// Unregister the process now that it has finished.
	s.execMu.Lock()
	delete(s.execProcs, key)
	s.execMu.Unlock()

	if waitErr != nil && !clientGone {
		evt, _ := json.Marshal(map[string]any{
			"type":  "chat_error",
			"error": waitErr.Error(),
		})
		fmt.Fprintf(w, "data: %s\n\n", evt)
		flusher.Flush()
	}

	if clientGone {
		return
	}
	done, _ := json.Marshal(map[string]any{
		"type":       "chat_done",
		"session_id": detectedSessionID,
	})
	fmt.Fprintf(w, "data: %s\n\n", done)
	flusher.Flush()
}

// handleAgentChatStop kills a running agent exec process for a project/agent.
func (s *Server) handleAgentChatStop(w http.ResponseWriter, r *http.Request) {
	project, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	if !s.checkProjectAccess(w, r, project) {
		return
	}

	key := project + "/" + agent
	s.execMu.Lock()
	proc, ok := s.execProcs[key]
	if ok {
		delete(s.execProcs, key)
	}
	s.execMu.Unlock()

	if proc == nil || proc.cmd.Process == nil {
		// No process running, treat as success (idempotent).
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "msg": "no process running"})
		return
	}

	pid := proc.cmd.Process.Pid
	killProcessGroup(pid)

	// Give it a moment then force kill if still alive.
	time.Sleep(500 * time.Millisecond)
	if proc.cmd.Process != nil {
		_ = proc.cmd.Process.Kill()
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "pid": pid})
}

func chatSSELine(line string, model entity.AgentModel) string {
	trimmed := strings.TrimSpace(line)
	if model == entity.ModelCodex || model == entity.ModelQoder {
		return line
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "===") ||
		strings.HasPrefix(trimmed, "Command:") || strings.HasPrefix(trimmed, "Started:") {
		return line
	}
	return "=== " + line + " ==="
}

func extractAgentChatSessionID(line string) string {
	if strings.Contains(line, "\n") {
		scanner := bufio.NewScanner(strings.NewReader(line))
		for scanner.Scan() {
			if sid := extractAgentChatSessionID(scanner.Text()); sid != "" {
				return sid
			}
		}
		return ""
	}
	var raw map[string]any
	if strings.Contains(line, `"session_id"`) && json.Unmarshal([]byte(line), &raw) == nil {
		if sid, ok := raw["session_id"].(string); ok && sid != "" {
			return sid
		}
	}
	trimmed := strings.TrimSpace(line)
	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{"session id:", "session:", "session :"} {
		if after, ok := strings.CutPrefix(lower, prefix); ok {
			start := len(trimmed) - len(after)
			return strings.TrimSpace(trimmed[start:])
		}
	}
	return ""
}
