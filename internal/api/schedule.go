package api

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
)

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
		if meta, err := s.st.AgentMeta(name, ag.Name); err == nil && meta != nil {
			entry["model"] = string(meta.Model)
			entry["agentDir"] = s.st.AgentDir(name, ag.Name)
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
		"maxTasksPerCycle":      h.MaxTasksPerCycle,
		"maxCycleDuration":      h.MaxCycleDuration,
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
		"id":       c.ID,
		"title":    c.Title,
		"schedule": c.Schedule,
		"enabled":  c.Enabled,
		"prompt":   c.Prompt,
		"runCount": c.RunCount,
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
	Paused           *bool   `json:"paused,omitempty"`
	ActiveHours      *string `json:"activeHours,omitempty"`
	ActiveDays       *string `json:"activeDays,omitempty"`
	SessionScope     *string `json:"sessionScope,omitempty"`
	WakeupPrompt     *string `json:"wakeupPrompt,omitempty"`
	WakeupCondition  *string `json:"wakeupCondition,omitempty"`
	MaxTasksPerCycle *int    `json:"maxTasksPerCycle,omitempty"`
	MaxCycleDuration *string `json:"maxCycleDuration,omitempty"`
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
	if body.WakeupPrompt != nil {
		hb.WakeupPrompt = *body.WakeupPrompt
	}
	if body.WakeupCondition != nil {
		hb.WakeupCondition = *body.WakeupCondition
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
	Title    string `json:"title"`
	Schedule string `json:"schedule"`
	Prompt   string `json:"prompt"`
	Enabled  *bool  `json:"enabled"`
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
	c := &entity.Cron{
		ID:       id,
		Title:    title,
		Schedule: schedule,
		Enabled:  enabled,
		Prompt:   prompt,
	}
	crons = append(crons, c)
	if err := s.ts.SaveCrons(name, agent, crons); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(cronToJSON(c))
}

func randomAlpha(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
