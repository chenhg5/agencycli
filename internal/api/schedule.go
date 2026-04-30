package api

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/telemetry"
)

// processAlive checks whether a process with the given PID is still running.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func (s *Server) handleGetProjectSchedule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.st.Project(name); err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "project not found")
			return
		}
		s.serverError(w, err)
		return
	}
	agents, err := s.st.ListAgents(name)
	if err != nil {
		s.serverError(w, err)
		return
	}
	sortAgents := make([]map[string]any, 0, len(agents))
	for _, ag := range agents {
		if ag == nil {
			continue
		}
		hb, err := s.ts.GetHeartbeat(name, ag.Name)
		if err != nil {
			s.serverError(w, err)
			return
		}
		// Fix stale "running" status: if the recorded process is no longer alive,
		// the wakeup/scheduler must have exited without cleanup (crash/kill).
		if hb.LastWakeupStatus == "running" && hb.PID > 0 && !processAlive(hb.PID) {
			hb.LastWakeupStatus = "done"
			hb.PID = 0
			_ = s.ts.SaveHeartbeat(name, ag.Name, hb)
		}
		crons, err := s.ts.ListCrons(name, ag.Name)
		if err != nil {
			s.serverError(w, err)
			return
		}
		if crons == nil {
			crons = []*entity.Cron{}
		}
		cronOut := make([]map[string]any, 0, len(crons))
		for _, c := range crons {
			if c == nil {
				continue
			}
			cronOut = append(cronOut, cronToJSON(c))
		}
		entry := map[string]any{
			"name":      ag.Name,
			"heartbeat": heartbeatToJSON(hb),
			"crons":     cronOut,
		}
		modelStr := ""
		if meta, err := s.st.AgentMeta(name, ag.Name); err == nil && meta != nil {
			modelStr = string(meta.Model)
			entry["model"] = modelStr
			entry["agentDir"] = s.st.AgentDir(name, ag.Name)
		}
		if hb.SessionID != "" {
			if db, err := telemetry.OpenReadOnly(s.root); err == nil {
				if usage, err := telemetry.ReadSessionUsage(db, hb.SessionID); err == nil && usage != nil && usage.RunCount > 0 {
					ctxLimit := telemetry.ContextWindowLimit(modelStr)
					entry["sessionUsage"] = map[string]any{
						"lastInputTokens":   usage.LastInputTokens,
						"totalInputTokens":  usage.TotalInputTokens,
						"totalOutputTokens": usage.TotalOutputTokens,
						"totalCacheRead":    usage.TotalCacheRead,
						"totalCostUsd":      usage.TotalCostUSD,
						"runCount":          usage.RunCount,
						"contextLimit":      ctxLimit,
					}
				}
				db.Close()
			}
		}
		sortAgents = append(sortAgents, entry)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"project": name,
		"agents":  sortAgents,
	})
}

func heartbeatToJSON(h *entity.HeartbeatConfig) map[string]any {
	if h == nil {
		return map[string]any{"enabled": false}
	}
	out := map[string]any{
		"enabled":               h.Enabled,
		"interval":              h.Interval,
		"paused":                h.Paused,
		"activeHours":           h.ActiveHours,
		"activeDays":            h.ActiveDays,
		"sessionScope":          string(h.SessionScope),
		"wakeupPrompt":          h.WakeupPrompt,
		"wakeupCondition":       h.WakeupCondition,
		"wakeupPreset":          h.WakeupPreset,
		"jitter":                h.Jitter,
		"maxTasksPerCycle":      h.MaxTasksPerCycle,
		"maxCycleDuration":      h.MaxCycleDuration,
		"triggers":              h.Triggers,
		"pid":                   h.PID,
		"lastWakeupStatus":      h.LastWakeupStatus,
		"sessionId":             h.SessionID,
		"lastConditionStatus":   h.LastConditionStatus,
		"wakeupCount":           h.WakeupCount,
		"wakeupCountToday":      h.WakeupCountToday,
		"lastCycleDuration":     h.LastCycleDuration,
	}
	if h.LastWakeup != nil {
		out["lastWakeup"] = h.LastWakeup.UTC().Format(time.RFC3339Nano)
	}
	if h.SessionStartedAt != nil {
		out["sessionStartedAt"] = h.SessionStartedAt.UTC().Format(time.RFC3339Nano)
	}
	if h.LastConditionAt != nil {
		out["lastConditionAt"] = h.LastConditionAt.UTC().Format(time.RFC3339Nano)
	}
	if h.NextWakeupAt != nil {
		out["nextWakeupAt"] = h.NextWakeupAt.UTC().Format(time.RFC3339Nano)
	}
	if h.SchedulerStartedAt != nil {
		out["schedulerStartedAt"] = h.SchedulerStartedAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func cronToJSON(c *entity.Cron) map[string]any {
	out := map[string]any{
		"id":           c.ID,
		"title":        c.Title,
		"schedule":     c.Schedule,
		"enabled":      c.Enabled,
		"prompt":       c.Prompt,
		"runCount":     c.RunCount,
		"sessionScope": c.SessionScope,
		"jitter":       c.Jitter,
	}
	if c.SessionID != "" {
		out["sessionId"] = c.SessionID
	}
	if c.SessionStartedAt != nil {
		out["sessionStartedAt"] = c.SessionStartedAt.UTC().Format(time.RFC3339Nano)
	}
	if c.LastRun != nil {
		out["lastRun"] = c.LastRun.UTC().Format(time.RFC3339Nano)
	}
	out["lastRunStatus"] = c.LastRunStatus
	return out
}

func (s *Server) handlePostHeartbeatPause(w http.ResponseWriter, r *http.Request) {
	s.toggleHeartbeatPause(w, r, true)
}

func (s *Server) handlePostHeartbeatResume(w http.ResponseWriter, r *http.Request) {
	s.toggleHeartbeatPause(w, r, false)
}

func (s *Server) toggleHeartbeatPause(w http.ResponseWriter, r *http.Request, pause bool) {
	name, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	if pause {
		if err := s.ts.PauseHeartbeat(name, agent); err != nil {
			s.serverError(w, err)
			return
		}
	} else {
		if err := s.ts.ResumeHeartbeat(name, agent); err != nil {
			s.serverError(w, err)
			return
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) parseProjectAgent(w http.ResponseWriter, r *http.Request) (project, agent string, ok bool) {
	project = r.PathValue("name")
	agent = r.PathValue("agent")
	if _, err := s.st.Project(project); err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "project not found")
			return "", "", false
		}
		s.serverError(w, err)
		return "", "", false
	}
	if !s.agentExistsInProject(project, agent) {
		s.jsonError(w, http.StatusNotFound, "agent not found")
		return "", "", false
	}
	return project, agent, true
}

type patchHeartbeatBody struct {
	Enabled          *bool   `json:"enabled,omitempty"`
	Interval         *string `json:"interval,omitempty"`
	Jitter           *string `json:"jitter,omitempty"`
	Paused           *bool   `json:"paused,omitempty"`
	ActiveHours      *string `json:"activeHours,omitempty"`
	ActiveDays       *string `json:"activeDays,omitempty"`
	SessionScope     *string `json:"sessionScope,omitempty"`
	SessionID        *string `json:"sessionId,omitempty"`
	WakeupPrompt     *string `json:"wakeupPrompt,omitempty"`
	// WakeupCondition is intentionally excluded from API PATCH for security.
	// Setting this field via API would allow command injection since it's
	// executed with sh -c. Use CLI 'scheduler configure --wakeup-condition' instead.
	WakeupPreset     *string              `json:"wakeupPreset,omitempty"`
	Triggers         *[]entity.TriggerType `json:"triggers"` // null = not sent, [] = clear
	MaxTasksPerCycle *int                 `json:"maxTasksPerCycle,omitempty"`
	MaxCycleDuration *string              `json:"maxCycleDuration,omitempty"`
}

func (s *Server) handleGetHeartbeat(w http.ResponseWriter, r *http.Request) {
	name, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	hb, err := s.ts.GetHeartbeat(name, agent)
	if err != nil {
		s.serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(heartbeatToJSON(hb))
}

func (s *Server) handlePatchHeartbeat(w http.ResponseWriter, r *http.Request) {
	name, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	var body patchHeartbeatBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	hb, err := s.ts.GetHeartbeat(name, agent)
	if err != nil {
		s.serverError(w, err)
		return
	}
	if body.Enabled != nil {
		hb.Enabled = *body.Enabled
	}
	if body.Interval != nil {
		if strings.TrimSpace(*body.Interval) != "" {
			if _, err := time.ParseDuration(strings.TrimSpace(*body.Interval)); err != nil {
				s.jsonError(w, http.StatusBadRequest, "invalid interval duration")
				return
			}
		}
		hb.Interval = strings.TrimSpace(*body.Interval)
	}
	if body.Jitter != nil {
		if t := strings.TrimSpace(*body.Jitter); t != "" {
			if _, err := time.ParseDuration(t); err != nil {
				s.jsonError(w, http.StatusBadRequest, "invalid jitter duration")
				return
			}
		}
		hb.Jitter = strings.TrimSpace(*body.Jitter)
	}
	if body.Paused != nil {
		hb.Paused = *body.Paused
	}
	if body.ActiveHours != nil {
		hb.ActiveHours = strings.TrimSpace(*body.ActiveHours)
	}
	if body.ActiveDays != nil {
		hb.ActiveDays = strings.TrimSpace(*body.ActiveDays)
	}
	if body.SessionScope != nil && strings.TrimSpace(*body.SessionScope) != "" {
		hb.SessionScope = entity.SessionScope(strings.TrimSpace(*body.SessionScope))
	}
	if hb.SessionScope == "" {
		hb.SessionScope = entity.SessionScopeCycle
	}
	if body.SessionID != nil {
		newSID := strings.TrimSpace(*body.SessionID)
		if newSID != hb.SessionID {
			hb.SessionID = newSID
			if newSID == "" {
				hb.SessionStartedAt = nil
			} else {
				now := time.Now().UTC()
				hb.SessionStartedAt = &now
			}
		}
	}
	if body.WakeupPrompt != nil {
		hb.WakeupPrompt = *body.WakeupPrompt
	}
	// WakeupCondition is NOT accepted via API for security reasons.
	// It's executed with sh -c, so allowing arbitrary input would be a
	// command injection vulnerability. Use CLI 'scheduler configure' instead.
	if body.WakeupPreset != nil {
		hb.WakeupPreset = strings.TrimSpace(*body.WakeupPreset)
	}
	if body.Triggers != nil {
		valid := make([]entity.TriggerType, 0, len(*body.Triggers))
		for _, t := range *body.Triggers {
			switch t {
			case entity.TriggerOnMessage, entity.TriggerOnTask:
				valid = append(valid, t)
			}
		}
		hb.Triggers = valid
	}
	if body.MaxTasksPerCycle != nil {
		hb.MaxTasksPerCycle = *body.MaxTasksPerCycle
	}
	if body.MaxCycleDuration != nil {
		if t := strings.TrimSpace(*body.MaxCycleDuration); t != "" {
			if _, err := time.ParseDuration(t); err != nil {
				s.jsonError(w, http.StatusBadRequest, "invalid maxCycleDuration")
				return
			}
		}
		hb.MaxCycleDuration = strings.TrimSpace(*body.MaxCycleDuration)
	}
	if err := s.ts.SaveHeartbeat(name, agent, hb); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(heartbeatToJSON(hb))
	// Auto-restart scheduler so config changes take effect immediately.
	go func() {
		statuses := s.sched.Status()
		for _, st := range statuses {
			if !st.Running {
				continue
			}
			if st.Key == name || st.Key == "all" || st.Key == name+"/"+agent {
				proj := st.Project
				ag := st.Agent
				_ = s.sched.Stop(proj, ag)
				time.Sleep(500 * time.Millisecond)
				_ = s.sched.Start(proj, ag)
				break
			}
		}
	}()
}

func (s *Server) handleAgentLiveLog(w http.ResponseWriter, r *http.Request) {
	name, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	logDir, err := s.ts.RunLogDir(name, agent)
	if err != nil {
		s.serverError(w, err)
		return
	}
	entries, err := os.ReadDir(logDir)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"content": "", "path": "", "finished": true})
		return
	}
	// Find latest .log file by name (names start with timestamp).
	var latest string
	for i := len(entries) - 1; i >= 0; i-- {
		if !entries[i].IsDir() && strings.HasSuffix(entries[i].Name(), ".log") {
			latest = filepath.Join(logDir, entries[i].Name())
			break
		}
	}
	if latest == "" {
		_ = json.NewEncoder(w).Encode(map[string]any{"content": "", "path": "", "finished": true})
		return
	}
	data, err := os.ReadFile(latest)
	if err != nil {
		s.serverError(w, err)
		return
	}
	content := string(data)
	const maxBytes = 1024 * 1024
	if len(data) > maxBytes {
		content = string(data[len(data)-maxBytes:])
	}
	// Check if the log has a "=== exit code:" or "=== finished:" line, meaning execution is done.
	finished := strings.Contains(content, "=== exit code:") || strings.Contains(content, "=== finished:")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"content":  content,
		"path":     latest,
		"finished": finished,
	})
}

func (s *Server) handlePostCronPause(w http.ResponseWriter, r *http.Request) {
	name, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	id := r.PathValue("cronId")
	if err := s.ts.PauseCron(name, agent, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "cron not found")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handlePostCronResume(w http.ResponseWriter, r *http.Request) {
	name, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	id := r.PathValue("cronId")
	if err := s.ts.ResumeCron(name, agent, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "cron not found")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleDeleteCron(w http.ResponseWriter, r *http.Request) {
	name, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	id := r.PathValue("cronId")
	if err := s.ts.DeleteCron(name, agent, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "cron not found")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

type postCronBody struct {
	Title        string  `json:"title"`
	Schedule     string  `json:"schedule"`
	Prompt       string  `json:"prompt"`
	Enabled      *bool   `json:"enabled"`
	SessionScope *string `json:"sessionScope"`
	Jitter       *string `json:"jitter"`
	SessionID    *string `json:"sessionId"`
}

func (s *Server) handlePostCron(w http.ResponseWriter, r *http.Request) {
	name, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	var body postCronBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	title := strings.TrimSpace(body.Title)
	schedule := strings.TrimSpace(body.Schedule)
	prompt := strings.TrimSpace(body.Prompt)
	if title == "" || schedule == "" || prompt == "" {
		s.jsonError(w, http.StatusBadRequest, "title, schedule, and prompt are required")
		return
	}
	if err := validateCronSchedule(schedule); err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	crons, err := s.ts.ListCrons(name, agent)
	if err != nil {
		s.serverError(w, err)
		return
	}
	if crons == nil {
		crons = []*entity.Cron{}
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	id := fmt.Sprintf("c-%s-%s", time.Now().UTC().Format("20060102"), randomAlpha(6))
	scope := ""
	if body.SessionScope != nil {
		scope = *body.SessionScope
	}
	jitter := ""
	if body.Jitter != nil {
		if t := strings.TrimSpace(*body.Jitter); t != "" {
			if _, err := time.ParseDuration(t); err != nil {
				s.jsonError(w, http.StatusBadRequest, "invalid jitter duration")
				return
			}
			jitter = t
		}
	}
	c := &entity.Cron{
		ID:           id,
		Title:        title,
		Schedule:     schedule,
		Enabled:      enabled,
		Prompt:       prompt,
		SessionScope: scope,
		Jitter:       jitter,
	}
	crons = append(crons, c)
	if err := s.ts.SaveCrons(name, agent, crons); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(cronToJSON(c))
}

func (s *Server) handlePutCron(w http.ResponseWriter, r *http.Request) {
	name, agent, ok := s.parseProjectAgent(w, r)
	if !ok {
		return
	}
	cronID := r.PathValue("cronId")
	var body postCronBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	crons, err := s.ts.ListCrons(name, agent)
	if err != nil {
		s.serverError(w, err)
		return
	}
	var target *entity.Cron
	for _, c := range crons {
		if c.ID == cronID {
			target = c
			break
		}
	}
	if target == nil {
		s.jsonError(w, http.StatusNotFound, "cron not found")
		return
	}
	if t := strings.TrimSpace(body.Title); t != "" {
		target.Title = t
	}
	if p := strings.TrimSpace(body.Prompt); p != "" {
		target.Prompt = p
	}
	if sc := strings.TrimSpace(body.Schedule); sc != "" {
		if err := validateCronSchedule(sc); err != nil {
			s.jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		target.Schedule = sc
	}
	if body.Enabled != nil {
		target.Enabled = *body.Enabled
	}
	if body.SessionScope != nil {
		target.SessionScope = *body.SessionScope
		if *body.SessionScope != "persistent" {
			target.SessionID = ""
			target.SessionStartedAt = nil
		}
	}
	if body.Jitter != nil {
		if t := strings.TrimSpace(*body.Jitter); t != "" {
			if _, err := time.ParseDuration(t); err != nil {
				s.jsonError(w, http.StatusBadRequest, "invalid jitter duration")
				return
			}
			target.Jitter = t
		} else {
			target.Jitter = ""
		}
	}
	if body.SessionID != nil {
		newSID := strings.TrimSpace(*body.SessionID)
		if newSID != target.SessionID {
			target.SessionID = newSID
			if newSID == "" {
				target.SessionStartedAt = nil
			} else {
				now := time.Now().UTC()
				target.SessionStartedAt = &now
			}
		}
	}
	if err := s.ts.SaveCrons(name, agent, crons); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(cronToJSON(target))
}

func randomAlpha(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
