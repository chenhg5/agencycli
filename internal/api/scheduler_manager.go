package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

type schedulerProcess struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	project   string
	agent     string
	startedAt time.Time
	stopped   bool
	exitErr   error
	doneCh    chan struct{}
}

type SchedulerManager struct {
	mu       sync.Mutex
	root     string
	binPath  string
	procs    map[string]*schedulerProcess // key = "all" or "project" or "project/agent"
}

func newSchedulerManager(root string) *SchedulerManager {
	bin, _ := os.Executable()
	return &SchedulerManager{
		root:    root,
		binPath: bin,
		procs:   make(map[string]*schedulerProcess),
	}
}

func schedKey(project, agent string) string {
	if project == "" {
		return "all"
	}
	if agent == "" {
		return project
	}
	return project + "/" + agent
}

func (m *SchedulerManager) Start(project, agent string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := schedKey(project, agent)
	if p, ok := m.procs[key]; ok {
		select {
		case <-p.doneCh:
			// process already exited, allow restart
		default:
			return fmt.Errorf("scheduler already running for %q", key)
		}
	}

	args := []string{"--dir", m.root, "scheduler", "start"}
	if project != "" {
		args = append(args, "--project", project)
	}
	if agent != "" {
		args = append(args, "--agent", agent)
	}

	cmd := exec.Command(m.binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setProcGroup(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}

	proc := &schedulerProcess{
		cmd:       cmd,
		project:   project,
		agent:     agent,
		startedAt: time.Now(),
		doneCh:    make(chan struct{}),
	}

	go func() {
		err := cmd.Wait()
		proc.mu.Lock()
		proc.exitErr = err
		proc.stopped = true
		proc.mu.Unlock()
		close(proc.doneCh)
	}()

	m.procs[key] = proc
	return nil
}

func (m *SchedulerManager) Stop(project, agent string) error {
	m.mu.Lock()
	key := schedKey(project, agent)
	proc, ok := m.procs[key]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("no scheduler running for %q", key)
	}

	select {
	case <-proc.doneCh:
		return fmt.Errorf("scheduler for %q already stopped", key)
	default:
	}

	if proc.cmd.Process != nil {
		killProcessGroup(proc.cmd.Process.Pid)
	}

	select {
	case <-proc.doneCh:
	case <-time.After(5 * time.Second):
		if proc.cmd.Process != nil {
			_ = proc.cmd.Process.Kill()
		}
	}

	m.mu.Lock()
	delete(m.procs, key)
	m.mu.Unlock()
	return nil
}

type schedStatus struct {
	Key       string `json:"key"`
	Running   bool   `json:"running"`
	PID       int    `json:"pid,omitempty"`
	Project   string `json:"project,omitempty"`
	Agent     string `json:"agent,omitempty"`
	StartedAt string `json:"startedAt,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (m *SchedulerManager) Status() []schedStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]schedStatus, 0, len(m.procs))
	for key, proc := range m.procs {
		s := schedStatus{
			Key:     key,
			Project: proc.project,
			Agent:   proc.agent,
		}
		select {
		case <-proc.doneCh:
			s.Running = false
			proc.mu.Lock()
			if proc.exitErr != nil {
				s.Error = proc.exitErr.Error()
			}
			proc.mu.Unlock()
		default:
			s.Running = true
			if proc.cmd.Process != nil {
				s.PID = proc.cmd.Process.Pid
			}
			s.StartedAt = proc.startedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, s)
	}
	return out
}

func (m *SchedulerManager) Cleanup() {
	m.mu.Lock()
	keys := make([]string, 0, len(m.procs))
	for k := range m.procs {
		keys = append(keys, k)
	}
	m.mu.Unlock()

	for _, k := range keys {
		parts := strings.SplitN(k, "/", 2)
		project := ""
		agent := ""
		if k != "all" {
			project = parts[0]
			if len(parts) > 1 {
				agent = parts[1]
			}
		}
		_ = m.Stop(project, agent)
	}
}

// ── HTTP handlers ──

func (s *Server) handleSchedulerStatus(w http.ResponseWriter, r *http.Request) {
	statuses := s.sched.Status()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"schedulers": statuses,
	})
}

type schedActionBody struct {
	Project string `json:"project"`
	Agent   string `json:"agent"`
}

func (s *Server) handleSchedulerStart(w http.ResponseWriter, r *http.Request) {
	var body schedActionBody
	if r.ContentLength > 0 {
		if err := s.readJSON(w, r, &body); err != nil {
			s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)

	if agent != "" && project == "" {
		s.jsonError(w, http.StatusBadRequest, "agent requires project")
		return
	}

	if err := s.sched.Start(project, agent); err != nil {
		s.jsonError(w, http.StatusConflict, err.Error())
		return
	}

	key := schedKey(project, agent)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":  true,
		"key": key,
	})
}

func (s *Server) handleSchedulerWakeup(w http.ResponseWriter, r *http.Request) {
	var body schedActionBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)
	if project == "" || agent == "" {
		s.jsonError(w, http.StatusBadRequest, "project and agent are required")
		return
	}

	args := []string{"--dir", s.sched.root, "scheduler", "wakeup", "--project", project, "--agent", agent}
	cmd := exec.Command(s.sched.binPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("wakeup failed: %v\n%s", err, string(out)))
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "output": string(out)})
}

func (s *Server) handleSchedulerStop(w http.ResponseWriter, r *http.Request) {
	var body schedActionBody
	if r.ContentLength > 0 {
		if err := s.readJSON(w, r, &body); err != nil {
			s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)

	if err := s.sched.Stop(project, agent); err != nil {
		s.jsonError(w, http.StatusNotFound, err.Error())
		return
	}

	s.clearSchedulerRuntimeFields(project, agent)

	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) clearSchedulerRuntimeFields(project, agent string) {
	if project == "" {
		return
	}
	agents := []string{agent}
	if agent == "" {
		projAgents, err := s.ts.ListAgents(project)
		if err != nil {
			return
		}
		agents = projAgents
	}
	for _, ag := range agents {
		hb, err := s.ts.GetHeartbeat(project, ag)
		if err != nil || hb == nil {
			continue
		}
		hb.NextWakeupAt = nil
		hb.SchedulerStartedAt = nil
		hb.PID = 0
		if hb.LastWakeupStatus == "running" {
			hb.LastWakeupStatus = "done"
		}
		_ = s.ts.SaveHeartbeat(project, ag, hb)
	}
}

func (s *Server) handleSchedulerAbort(w http.ResponseWriter, r *http.Request) {
	var body schedActionBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)
	if project == "" || agent == "" {
		s.jsonError(w, http.StatusBadRequest, "project and agent are required")
		return
	}

	hb, err := s.ts.GetHeartbeat(project, agent)
	if err != nil || hb == nil {
		s.jsonError(w, http.StatusNotFound, "heartbeat config not found")
		return
	}

	if hb.PID <= 0 || hb.LastWakeupStatus != "running" {
		s.jsonError(w, http.StatusConflict, "agent is not currently running")
		return
	}

	pid := hb.PID

	proc, err := os.FindProcess(pid)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("cannot find process %d: %v", pid, err))
		return
	}

	// Signal 0 checks liveness.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		hb.PID = 0
		hb.LastWakeupStatus = "aborted"
		_ = s.ts.SaveHeartbeat(project, agent, hb)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "msg": "process already dead, status updated"})
		return
	}

	// Kill the process group to ensure child processes (docker, claude) are also terminated.
	killProcessGroup(pid)

	// Give it a moment then force kill if needed.
	time.Sleep(500 * time.Millisecond)
	if err := proc.Signal(syscall.Signal(0)); err == nil {
		_ = proc.Kill()
	}

	hb.PID = 0
	hb.LastWakeupStatus = "aborted"
	_ = s.ts.SaveHeartbeat(project, agent, hb)

	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "pid": pid})
}
