package api

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/taskstore"
)

// triggerManager handles event-driven agent wakeups.
// It deduplicates concurrent triggers for the same agent and runs wakeups
// asynchronously so the originating API call returns immediately.
type triggerManager struct {
	mu      sync.Mutex
	inflight map[string]time.Time // key = "project/agent" → trigger start time
	root    string
	binPath string
	ts      taskstore.Store
}

func newTriggerManager(root, binPath string, ts taskstore.Store) *triggerManager {
	return &triggerManager{
		inflight: make(map[string]time.Time),
		root:     root,
		binPath:  binPath,
		ts:       ts,
	}
}

// Fire checks whether the agent has the given trigger configured and, if so,
// launches an asynchronous wakeup. It is safe to call from any goroutine.
// reason is a human-readable label for logging (e.g. "message from pm").
func (tm *triggerManager) Fire(project, agent string, triggerType entity.TriggerType, reason string) {
	hb, err := tm.ts.GetHeartbeat(project, agent)
	if err != nil || hb == nil {
		fmt.Fprintf(os.Stderr, "[trigger] %s/%s: skip — no heartbeat (err=%v)\n", project, agent, err)
		return
	}
	if !hb.HasTrigger(triggerType) {
		fmt.Fprintf(os.Stderr, "[trigger] %s/%s: skip — trigger %q not configured (has: %v)\n", project, agent, triggerType, hb.Triggers)
		return
	}
	if hb.Paused {
		fmt.Fprintf(os.Stderr, "[trigger] %s/%s: skip — paused\n", project, agent)
		return
	}

	key := project + "/" + agent

	tm.mu.Lock()
	if _, ok := tm.inflight[key]; ok {
		tm.mu.Unlock()
		fmt.Fprintf(os.Stderr, "[trigger] %s/%s: skip — already inflight\n", project, agent)
		return
	}
	if hb.PID > 0 && hb.LastWakeupStatus == "running" {
		if proc, err := os.FindProcess(hb.PID); err == nil {
			if proc.Signal(syscall.Signal(0)) == nil {
				tm.mu.Unlock()
				fmt.Fprintf(os.Stderr, "[trigger] %s/%s: skip — agent already running (pid=%d)\n", project, agent, hb.PID)
				return
			}
		}
	}
	tm.inflight[key] = time.Now()
	tm.mu.Unlock()

	go func() {
		defer func() {
			tm.mu.Lock()
			delete(tm.inflight, key)
			tm.mu.Unlock()
		}()

		fmt.Fprintf(os.Stderr, "[trigger] %s/%s fired (%s: %s)\n", project, agent, triggerType, reason)

		args := []string{"--dir", tm.root, "scheduler", "wakeup", "--project", project, "--agent", agent}
		cmd := exec.Command(tm.binPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "[trigger] %s/%s: wakeup command failed: %v\n", project, agent, err)
		}
	}()
}
