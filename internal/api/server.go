// Package api exposes a JSON HTTP API over an agencycli workspace for the web UI and integrations.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/chenhg5/agencycli/internal/telemetry"
)

type contextKey string

const ctxUserKey contextKey = "auth-user"

// Server serves JSON for one workspace root.
type Server struct {
	root    string
	apiKey  string
	version string
	st      store.Store
	ts      taskstore.Store
	users   *UserStore
	sched   *SchedulerManager
}

// NewServer builds an API server for the given workspace root.
// If apiKey is non-empty, requests must send Authorization: Bearer <apiKey>.
func NewServer(root, apiKey string) *Server {
	return &Server{
		root:   root,
		apiKey: strings.TrimSpace(apiKey),
		st:     store.NewFS(root),
		ts:     taskstore.New(root),
		users:  newUserStore(root),
		sched:  newSchedulerManager(root),
	}
}

// SetVersion sets the build version string exposed via /api/v1/health.
func (s *Server) SetVersion(v string) { s.version = v }

// Handler returns the root HTTP handler (includes optional auth).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/agency", s.handleAgency)
	mux.HandleFunc("GET /api/v1/stats", s.handleStats)
	mux.HandleFunc("GET /api/v1/teams", s.handleTeams)
	mux.HandleFunc("GET /api/v1/teams/{teamPath...}", s.handleTeamDetail)
	mux.HandleFunc("GET /api/v1/projects", s.handleProjects)
	mux.HandleFunc("POST /api/v1/projects/{name}/tasks", s.handlePostProjectTask)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/crons/{cronId}/pause", s.handlePostCronPause)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/crons/{cronId}/resume", s.handlePostCronResume)
	mux.HandleFunc("DELETE /api/v1/projects/{name}/agents/{agent}/crons/{cronId}", s.handleDeleteCron)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/crons", s.handlePostCron)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/heartbeat/pause", s.handlePostHeartbeatPause)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/heartbeat/resume", s.handlePostHeartbeatResume)
	mux.HandleFunc("PATCH /api/v1/projects/{name}/agents/{agent}/heartbeat", s.handlePatchHeartbeat)
	mux.HandleFunc("GET /api/v1/projects/{name}/schedule", s.handleGetProjectSchedule)
	mux.HandleFunc("POST /api/v1/messages/delete", s.handlePostDeleteMessage)
	mux.HandleFunc("POST /api/v1/messages/mark-read", s.handlePostMarkMessageRead)
	mux.HandleFunc("POST /api/v1/messages/archive", s.handlePostArchiveMessage)
	mux.HandleFunc("POST /api/v1/messages/mark-all-read", s.handlePostMarkAllMessagesRead)
	mux.HandleFunc("POST /api/v1/messages", s.handlePostMessage)
	mux.HandleFunc("POST /api/v1/projects/{name}/messages/mark-all-read", s.handlePostProjectMarkAllMessagesRead)
	mux.HandleFunc("GET /api/v1/telemetry/summary", s.handleTelemetrySummary)
	mux.HandleFunc("GET /api/v1/telemetry/runs", s.handleTelemetryRuns)
	mux.HandleFunc("GET /api/v1/telemetry/log", s.handleTelemetryLog)
	mux.HandleFunc("POST /api/v1/tasks/cancel", s.handlePostCancelTask)
	mux.HandleFunc("POST /api/v1/tasks/archive", s.handlePostArchiveTask)
	mux.HandleFunc("PUT /api/v1/tasks/update", s.handlePutUpdateTask)
	mux.HandleFunc("POST /api/v1/tasks/delete", s.handlePostDeleteTask)
	mux.HandleFunc("GET /api/v1/projects/{name}/tasks", s.handleProjectTasks)
	mux.HandleFunc("GET /api/v1/projects/{name}/messages", s.handleProjectMessages)
	mux.HandleFunc("GET /api/v1/projects/{name}/agents", s.handleProjectAgents)
	mux.HandleFunc("GET /api/v1/projects/{name}/agents/{agent}/context", s.handleGetAgentContext)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/set-model", s.handleSetModel)
	mux.HandleFunc("PUT /api/v1/projects/{name}/agents/{agent}/wakeup", s.handlePutAgentWakeup)
	mux.HandleFunc("POST /api/v1/projects/{name}/sync", s.handlePostProjectSync)
	mux.HandleFunc("GET /api/v1/projects/{name}/prompt", s.handleGetProjectPrompt)
	mux.HandleFunc("PUT /api/v1/projects/{name}/prompt", s.handlePutProjectPrompt)
	mux.HandleFunc("GET /api/v1/projects/{name}", s.handleProject)
	mux.HandleFunc("GET /api/v1/prompts/agency", s.handleGetAgencyPrompt)
	mux.HandleFunc("PUT /api/v1/prompts/agency", s.handlePutAgencyPrompt)
	mux.HandleFunc("GET /api/v1/prompts/teams/{teamPath...}", s.handleGetTeamPrompt)
	mux.HandleFunc("PUT /api/v1/prompts/teams/{teamPath...}", s.handlePutTeamPrompt)
	mux.HandleFunc("GET /api/v1/prompts/roles", s.handleGetRolePrompt)
	mux.HandleFunc("PUT /api/v1/prompts/roles", s.handlePutRolePrompt)
	mux.HandleFunc("GET /api/v1/skills", s.handleListSkills)
	mux.HandleFunc("GET /api/v1/skills/{name}", s.handleGetSkillDetail)
	mux.HandleFunc("PUT /api/v1/skills/{name}", s.handlePutSkillPrompt)
	mux.HandleFunc("POST /api/v1/roles/skills", s.handlePostRoleSkillBind)
	mux.HandleFunc("POST /api/v1/teams/skills", s.handlePostTeamSkillBind)
	mux.HandleFunc("POST /api/v1/roles/create", s.handleCreateRole)
	mux.HandleFunc("POST /api/v1/projects/{name}/hire", s.handleHireAgent)
	mux.HandleFunc("POST /api/v1/run", s.handleRunAgent)
	mux.HandleFunc("POST /api/v1/session/reset", s.handleSessionReset)
	mux.HandleFunc("GET /api/v1/workbench/messages", s.handleWorkbenchMessages)
	mux.HandleFunc("GET /api/v1/workbench/tasks", s.handleWorkbenchTasks)
	mux.HandleFunc("GET /api/v1/scheduler/status", s.handleSchedulerStatus)
	mux.HandleFunc("POST /api/v1/scheduler/start", s.handleSchedulerStart)
	mux.HandleFunc("POST /api/v1/scheduler/stop", s.handleSchedulerStop)
	mux.HandleFunc("POST /api/v1/scheduler/wakeup", s.handleSchedulerWakeup)
	mux.HandleFunc("GET /api/v1/inbox", s.handleInbox)
	mux.HandleFunc("GET /api/v1/auth/me", s.handleAuthMe)
	mux.HandleFunc("PUT /api/v1/auth/password", s.handleChangePassword)

	publicMux := http.NewServeMux()
	publicMux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	publicMux.HandleFunc("GET /api/v1/health", s.handleHealth)
	publicMux.Handle("/", s.withTokenAuth(mux))

	return withJSONHeaders(publicMux)
}

func withJSONHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")

		// Legacy: static API key
		if s.apiKey != "" && token == s.apiKey {
			ctx := context.WithValue(r.Context(), ctxUserKey, "apikey")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		username, ok := s.users.ValidateToken(token)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired token"})
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	v := s.version
	if v == "" {
		v = "dev"
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "version": v})
}

func (s *Server) handleAgency(w http.ResponseWriter, _ *http.Request) {
	a, err := s.st.Agency()
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "agency not found")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":        a.Name,
		"description": a.Description,
		"lang":        a.Lang,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	projects, err := s.st.ListProjects()
	if err != nil {
		s.serverError(w, err)
		return
	}
	var pending, inProgress int
	for _, p := range projects {
		agents, err := s.st.ListAgents(p.Name)
		if err != nil {
			continue
		}
		for _, ag := range agents {
			tasks, err := s.ts.ListTasks(p.Name, ag.Name)
			if err != nil {
				continue
			}
			for _, t := range tasks {
				switch t.Status {
				case entity.TaskStatusPending:
					pending++
				case entity.TaskStatusInProgress:
					inProgress++
				}
			}
		}
	}

	runsToday := 0
	db, err := telemetry.OpenReadOnly(s.root)
	if err == nil {
		defer db.Close()
		now := time.Now()
		loc := time.Local
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).UTC()
		end := now.UTC()
		rows, err := telemetry.ReadRuns(db, &start, &end, "")
		if err == nil {
			runsToday = len(rows)
		}
	} else if !errors.Is(err, telemetry.ErrNoDatabase) {
		log.Printf("api stats: telemetry: %v", err)
	}

	_ = json.NewEncoder(w).Encode(map[string]int{
		"pendingTasks":    pending,
		"inProgressTasks": inProgress,
		"runsToday":       runsToday,
	})
}

func (s *Server) handleTeams(w http.ResponseWriter, _ *http.Request) {
	entries, err := s.st.ListTeams()
	if err != nil {
		s.serverError(w, err)
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		if e.Team == nil {
			continue
		}
		out = append(out, map[string]any{
			"path":        e.Path,
			"name":        e.Team.Name,
			"description": e.Team.Description,
			"parent":      e.Team.Parent,
			"skills":      e.Team.Skills,
			"goals":       e.Team.Goals,
		})
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleTeamDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.PathValue("teamPath"), "/")
	if path == "" {
		s.jsonError(w, http.StatusBadRequest, "missing team path")
		return
	}
	t, err := s.st.Team(path)
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "team not found")
			return
		}
		s.serverError(w, err)
		return
	}
	roles, err := s.st.ListRoles(path)
	if err != nil {
		s.serverError(w, err)
		return
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].Name < roles[j].Name })
	roleOut := make([]map[string]any, 0, len(roles))
	for _, re := range roles {
		if re.Role == nil {
			continue
		}
		sk := re.Role.Skills
		if sk == nil {
			sk = []string{}
		}
		roleOut = append(roleOut, map[string]any{
			"name":        re.Name,
			"description": re.Role.Description,
			"skills":      sk,
		})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"path":        path,
		"name":        t.Name,
		"description": t.Description,
		"parent":      t.Parent,
		"skills":      t.Skills,
		"goals":       t.Goals,
		"roles":       roleOut,
	})
}

func (s *Server) handleProjects(w http.ResponseWriter, _ *http.Request) {
	projects, err := s.st.ListProjects()
	if err != nil {
		s.serverError(w, err)
		return
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
	out := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		out = append(out, map[string]any{
			"name":        p.Name,
			"description": p.Description,
			"repo":        p.Repo,
		})
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	p, err := s.st.Project(name)
	if err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "project not found")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":        p.Name,
		"description": p.Description,
		"repo":        p.Repo,
	})
}

func (s *Server) handleProjectAgents(w http.ResponseWriter, r *http.Request) {
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
	sort.Slice(agents, func(i, j int) bool { return agents[i].Name < agents[j].Name })
	out := make([]map[string]any, 0, len(agents))
	for _, a := range agents {
		if a.Meta == nil {
			continue
		}
		out = append(out, map[string]any{
			"name":    a.Name,
			"model":   string(a.Meta.Model),
			"team":    a.Meta.Team,
			"project": a.Meta.Project,
			"hiredAt": a.Meta.HiredAt.UTC().Format(time.RFC3339Nano),
		})
	}
	_ = json.NewEncoder(w).Encode(out)
}

type taskRow struct {
	ID        string    `json:"id"`
	Project   string    `json:"project"`
	Agent     string    `json:"agent"`
	Title     string    `json:"title"`
	Type      string    `json:"type,omitempty"`
	Assignee  string    `json:"assignee,omitempty"`
	Prompt    string    `json:"prompt,omitempty"`
	Priority  int       `json:"priority"`
	Status    string    `json:"status"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s *Server) handleProjectTasks(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.st.Project(name); err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "project not found")
			return
		}
		s.serverError(w, err)
		return
	}

	qStatus := r.URL.Query().Get("status")
	qAgent := r.URL.Query().Get("agent")
	qPriority := r.URL.Query().Get("priority")
	qScope := r.URL.Query().Get("scope") // "active" (default), "archived", "all"
	if qScope == "" {
		qScope = "all"
	}

	agents, err := s.st.ListAgents(name)
	if err != nil {
		s.serverError(w, err)
		return
	}

	isWakeupTask := func(t *entity.Task) bool {
		return strings.HasPrefix(t.Title, "[wakeup]") || t.Type == "wakeup"
	}

	matchFilter := func(t *entity.Task) bool {
		if isWakeupTask(t) {
			return false
		}
		if qStatus != "" && string(t.Status) != qStatus {
			return false
		}
		if qPriority != "" && fmt.Sprintf("%d", t.Priority) != qPriority {
			return false
		}
		return true
	}

	rows := make([]taskRow, 0)
	for _, ag := range agents {
		if qAgent != "" && ag.Name != qAgent {
			continue
		}
		if qScope == "active" || qScope == "all" {
			tasks, err := s.ts.ListTasks(name, ag.Name)
			if err == nil {
				for _, t := range tasks {
					if !matchFilter(t) {
						continue
					}
					rows = append(rows, taskRow{
						ID: t.ID, Project: name, Agent: ag.Name,
						Title: t.Title, Type: string(t.Type), Assignee: t.Assignee, Prompt: t.Prompt,
						Priority: t.Priority,
						Status: string(t.Status), Archived: false,
						CreatedAt: t.CreatedAt.UTC(), UpdatedAt: t.UpdatedAt.UTC(),
					})
				}
			}
		}
		if qScope == "archived" || qScope == "all" {
			archived, err := s.ts.ListArchivedTasks(name, ag.Name)
			if err == nil {
				for _, t := range archived {
					if !matchFilter(t) {
						continue
					}
					rows = append(rows, taskRow{
						ID: t.ID, Project: name, Agent: ag.Name,
						Title: t.Title, Type: string(t.Type), Assignee: t.Assignee, Prompt: t.Prompt,
						Priority: t.Priority,
						Status: string(t.Status), Archived: true,
						CreatedAt: t.CreatedAt.UTC(), UpdatedAt: t.UpdatedAt.UTC(),
					})
				}
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].UpdatedAt.Equal(rows[j].UpdatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].UpdatedAt.After(rows[j].UpdatedAt)
	})
	_ = json.NewEncoder(w).Encode(rows)
}

type msgRow struct {
	ID         string     `json:"id"`
	From       string     `json:"from"`
	To         string     `json:"to"`
	Subject    string     `json:"subject,omitempty"`
	Body       string     `json:"body"`
	SentAt     time.Time  `json:"sentAt"`
	ReadAt     *time.Time `json:"readAt,omitempty"`
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
	Mailbox    string     `json:"mailbox"`
}

func (s *Server) handleProjectMessages(w http.ResponseWriter, r *http.Request) {
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
	q := r.URL.Query()
	archivedMode := strings.TrimSpace(strings.ToLower(q.Get("archived")))
	if archivedMode == "" && (q.Get("includeArchived") == "1" || strings.EqualFold(q.Get("includeArchived"), "true")) {
		archivedMode = "all"
	}
	if archivedMode == "" {
		archivedMode = "no"
	}
	if archivedMode != "no" && archivedMode != "yes" && archivedMode != "all" {
		s.jsonError(w, http.StatusBadRequest, "archived must be no, yes, or all")
		return
	}
	readFilter := strings.TrimSpace(strings.ToLower(q.Get("read")))
	if readFilter == "" {
		readFilter = "all"
	}
	if readFilter != "all" && readFilter != "read" && readFilter != "unread" {
		s.jsonError(w, http.StatusBadRequest, "read must be all, read, or unread")
		return
	}
	fromQ := strings.TrimSpace(q.Get("from"))
	toQ := strings.TrimSpace(q.Get("to"))
	mailboxFilter := strings.TrimSpace(q.Get("mailbox"))
	if mailboxFilter != "" {
		if err := s.validateIdentity(mailboxFilter, "mailbox"); err != nil {
			s.jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		parts := strings.SplitN(mailboxFilter, "/", 2)
		if len(parts) != 2 || parts[0] != name || !s.agentExistsInProject(name, parts[1]) {
			s.jsonError(w, http.StatusBadRequest, "mailbox must be project/agent for an agent in this project")
			return
		}
	}

	rows := make([]msgRow, 0)
	seen := map[string]struct{}{}
	useAll := archivedMode == "all" || archivedMode == "yes"

	add := func(recipient string) {
		var msgs []*entity.Message
		var err error
		if useAll {
			msgs, err = s.ts.ListAllMessages(recipient)
		} else {
			msgs, err = s.ts.ListMessages(recipient)
		}
		if err != nil {
			return
		}
		for _, m := range msgs {
			if m == nil {
				continue
			}
			if !messagePassesFilters(m, archivedMode, readFilter, fromQ, toQ) {
				continue
			}
			if _, ok := seen[m.ID]; ok {
				continue
			}
			seen[m.ID] = struct{}{}
			sent := m.SentAt.UTC()
			var read *time.Time
			if m.ReadAt != nil {
				t := m.ReadAt.UTC()
				read = &t
			}
			var arch *time.Time
			if m.ArchivedAt != nil {
				t := m.ArchivedAt.UTC()
				arch = &t
			}
			rows = append(rows, msgRow{
				ID:         m.ID,
				From:       m.From,
				To:         m.To,
				Subject:    m.Subject,
				Body:       m.Body,
				SentAt:     sent,
				ReadAt:     read,
				ArchivedAt: arch,
				Mailbox:    recipient,
			})
		}
	}

	if mailboxFilter != "" {
		add(mailboxFilter)
	} else {
		for _, ag := range agents {
			add(name + "/" + ag.Name)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].SentAt.After(rows[j].SentAt)
	})
	_ = json.NewEncoder(w).Encode(rows)
}

func messagePassesFilters(m *entity.Message, archivedMode, readFilter, fromQ, toQ string) bool {
	switch archivedMode {
	case "no":
		if m.ArchivedAt != nil {
			return false
		}
	case "yes":
		if m.ArchivedAt == nil {
			return false
		}
	}
	switch readFilter {
	case "read":
		if m.ReadAt == nil {
			return false
		}
	case "unread":
		if m.ReadAt != nil {
			return false
		}
	}
	if fromQ != "" && !strings.Contains(strings.ToLower(m.From), strings.ToLower(fromQ)) {
		return false
	}
	if toQ != "" && !strings.Contains(strings.ToLower(m.To), strings.ToLower(toQ)) {
		return false
	}
	return true
}

func (s *Server) handleInbox(w http.ResponseWriter, _ *http.Request) {
	items, err := s.ts.ListInbox()
	if err != nil {
		s.serverError(w, err)
		return
	}
	if items == nil {
		items = []*entity.InboxItem{}
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		out = append(out, map[string]any{
			"taskId":      it.TaskID,
			"project":     it.Project,
			"agent":       it.Agent,
			"title":       it.Title,
			"summary":     it.Summary,
			"recipient":   it.Recipient(),
			"actionHint":  it.ActionHint,
			"actionItems": it.ActionItems,
		})
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) jsonError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	log.Printf("api: %v", err)
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	return strings.Contains(err.Error(), "not found")
}

// ResolveAPIKey returns the API key from flag or AGENCYCLI_WEB_API_KEY.
func ResolveAPIKey(flag string) string {
	if strings.TrimSpace(flag) != "" {
		return strings.TrimSpace(flag)
	}
	return strings.TrimSpace(os.Getenv("AGENCYCLI_WEB_API_KEY"))
}
