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
	"sync"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/chenhg5/agencycli/internal/telemetry"
)

type contextKey string

const ctxUserKey contextKey = "auth-user"

// UpdateChecker returns latest version info. Set by the caller.
type UpdateChecker func() (latestVersion, releaseNotes string, hasUpdate bool)

// DaemonStatusFunc returns daemon status as a JSON-friendly map.
type DaemonStatusFunc func() map[string]any

// Server serves JSON for one workspace root.
type Server struct {
	root         string
	apiKey       string
	version      string
	st           store.Store
	ts           taskstore.Store
	users        *UserStore
	sched        *SchedulerManager
	triggers     *triggerManager
	ccStore      *store.CCConnectStore
	okrStore     *store.OKRStore
	msStore      *store.MilestoneStore
	updateCheck       UpdateChecker
	daemonStatus      DaemonStatusFunc
	assistantMu       sync.Mutex
	assistantSessions map[string]*assistantSession
}

// NewServer builds an API server for the given workspace root.
// If apiKey is non-empty, requests must send Authorization: Bearer <apiKey>.
func NewServer(root, apiKey string) *Server {
	sched := newSchedulerManager(root)
	ts := taskstore.New(root)
	return &Server{
		root:      root,
		apiKey:    strings.TrimSpace(apiKey),
		st:        store.NewFS(root),
		ts:        ts,
		users:     newUserStore(root),
		sched:     sched,
		triggers:  newTriggerManager(root, sched.binPath, ts),
		ccStore:   store.NewCCConnectStore(root),
		okrStore:  store.NewOKRStore(root),
		msStore:   store.NewMilestoneStore(root),
	}
}

// SetVersion sets the build version string exposed via /api/v1/health.
func (s *Server) SetVersion(v string) { s.version = v }

// SetUpdateChecker sets the function used to check for updates.
func (s *Server) SetUpdateChecker(fn UpdateChecker) { s.updateCheck = fn }

// SetDaemonStatus sets the function used to get daemon status.
func (s *Server) SetDaemonStatus(fn DaemonStatusFunc) { s.daemonStatus = fn }

// Shutdown stops all managed scheduler processes.
func (s *Server) Shutdown() { s.sched.Cleanup() }

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
	mux.HandleFunc("PUT /api/v1/projects/{name}/agents/{agent}/crons/{cronId}", s.handlePutCron)
	mux.HandleFunc("DELETE /api/v1/projects/{name}/agents/{agent}/crons/{cronId}", s.handleDeleteCron)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/crons", s.handlePostCron)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/heartbeat/pause", s.handlePostHeartbeatPause)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/heartbeat/resume", s.handlePostHeartbeatResume)
	mux.HandleFunc("GET /api/v1/projects/{name}/agents/{agent}/heartbeat", s.handleGetHeartbeat)
	mux.HandleFunc("PATCH /api/v1/projects/{name}/agents/{agent}/heartbeat", s.handlePatchHeartbeat)
	mux.HandleFunc("GET /api/v1/projects/{name}/agents/{agent}/env", s.handleGetAgentEnv)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/env", s.handleSetAgentEnv)
	mux.HandleFunc("DELETE /api/v1/projects/{name}/agents/{agent}/env", s.handleDeleteAgentEnv)
	mux.HandleFunc("GET /api/v1/projects/{name}/schedule", s.handleGetProjectSchedule)
	mux.HandleFunc("GET /api/v1/projects/{name}/agents/{agent}/live-log", s.handleAgentLiveLog)
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
	mux.HandleFunc("GET /api/v1/tasks/{project}/{agent}/{taskId}/comments", s.handleGetComments)
	mux.HandleFunc("POST /api/v1/tasks/{project}/{agent}/{taskId}/comments", s.handlePostComment)
	mux.HandleFunc("DELETE /api/v1/tasks/{project}/{agent}/{taskId}/comments/{commentId}", s.handleDeleteComment)
	mux.HandleFunc("GET /api/v1/projects/{name}/tasks", s.handleProjectTasks)
	mux.HandleFunc("GET /api/v1/projects/{name}/messages", s.handleProjectMessages)
	mux.HandleFunc("GET /api/v1/projects/{name}/agents", s.handleProjectAgents)
	mux.HandleFunc("GET /api/v1/projects/{name}/agents/{agent}/context", s.handleGetAgentContext)
	mux.HandleFunc("POST /api/v1/projects/{name}/agents/{agent}/set-model", s.handleSetModel)
	mux.HandleFunc("PUT /api/v1/projects/{name}/agents/{agent}/env", s.handlePutAgentEnv)
	mux.HandleFunc("PUT /api/v1/projects/{name}/agents/{agent}/wakeup", s.handlePutAgentWakeup)
	mux.HandleFunc("PUT /api/v1/projects/{name}/agents/{agent}/sandbox", s.handlePutAgentSandbox)
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
	mux.HandleFunc("DELETE /api/v1/projects/{name}/agents/{agent}", s.handleFireAgent)
	mux.HandleFunc("POST /api/v1/run", s.handleRunAgent)
	mux.HandleFunc("POST /api/v1/session/reset", s.handleSessionReset)
	mux.HandleFunc("POST /api/v1/assistant/chat", s.handleAssistantChat)
	mux.HandleFunc("POST /api/v1/assistant/permission", s.handleAssistantPermission)
	mux.HandleFunc("GET /api/v1/docs", s.handleDocsList)
	mux.HandleFunc("GET /api/v1/docs/tree", s.handleDocsTree)
	mux.HandleFunc("GET /api/v1/docs/{id}", s.handleDocsGet)
	mux.HandleFunc("POST /api/v1/docs", s.handleDocsAdd)
	mux.HandleFunc("PUT /api/v1/docs/{id}", s.handleDocsUpdate)
	mux.HandleFunc("GET /api/v1/docs/{id}/download", s.handleDocsDownload)
	mux.HandleFunc("DELETE /api/v1/docs/{id}", s.handleDocsDelete)
	mux.HandleFunc("GET /api/v1/workbench/messages", s.handleWorkbenchMessages)
	mux.HandleFunc("GET /api/v1/workbench/tasks", s.handleWorkbenchTasks)
	mux.HandleFunc("GET /api/v1/workbench/overview", s.handleWorkbenchOverview)
	mux.HandleFunc("GET /api/v1/scheduler/status", s.handleSchedulerStatus)
	mux.HandleFunc("POST /api/v1/scheduler/start", s.handleSchedulerStart)
	mux.HandleFunc("POST /api/v1/scheduler/stop", s.handleSchedulerStop)
	mux.HandleFunc("POST /api/v1/scheduler/wakeup", s.handleSchedulerWakeup)
	mux.HandleFunc("POST /api/v1/scheduler/abort", s.handleSchedulerAbort)
	mux.HandleFunc("GET /api/v1/inbox", s.handleInbox)
	mux.HandleFunc("GET /api/v1/auth/me", s.handleAuthMe)
	mux.HandleFunc("PUT /api/v1/auth/password", s.handleChangePassword)

	mux.HandleFunc("GET /api/v1/providers", s.handleListProviders)
	mux.HandleFunc("POST /api/v1/providers", s.handleAddProvider)
	mux.HandleFunc("PUT /api/v1/providers/{id}", s.handleUpdateProvider)
	mux.HandleFunc("DELETE /api/v1/providers/{id}", s.handleDeleteProvider)

	mux.HandleFunc("GET /api/v1/envvars", s.handleListEnvVars)
	mux.HandleFunc("POST /api/v1/envvars", s.handleCreateEnvVar)
	mux.HandleFunc("PUT /api/v1/envvars/{id}", s.handleUpdateEnvVar)
	mux.HandleFunc("DELETE /api/v1/envvars/{id}", s.handleDeleteEnvVar)

	mux.HandleFunc("GET /api/v1/users", s.handleListUsers)
	mux.HandleFunc("POST /api/v1/users", s.handleCreateUser)
	mux.HandleFunc("GET /api/v1/users/{username}", s.handleGetUser)
	mux.HandleFunc("PUT /api/v1/users/{username}", s.handleUpdateUser)
	mux.HandleFunc("DELETE /api/v1/users/{username}", s.handleDeleteUser)

	// ── Goals (OKR + Milestones) ──
	mux.HandleFunc("GET /api/v1/okrs", s.handleListOKRs)
	mux.HandleFunc("POST /api/v1/okrs", s.handleCreateOKR)
	mux.HandleFunc("GET /api/v1/okrs/{id}", s.handleGetOKR)
	mux.HandleFunc("PUT /api/v1/okrs/{id}", s.handleUpdateOKR)
	mux.HandleFunc("DELETE /api/v1/okrs/{id}", s.handleDeleteOKR)
	mux.HandleFunc("POST /api/v1/okrs/{id}/key-results", s.handleAddKR)
	mux.HandleFunc("PUT /api/v1/okrs/{id}/key-results/{krId}", s.handleUpdateKR)
	mux.HandleFunc("DELETE /api/v1/okrs/{id}/key-results/{krId}", s.handleDeleteKR)
	mux.HandleFunc("POST /api/v1/okrs/{id}/reviews", s.handleAddReviewNote)
	mux.HandleFunc("GET /api/v1/projects/{name}/milestones", s.handleListMilestones)
	mux.HandleFunc("POST /api/v1/projects/{name}/milestones", s.handleCreateMilestone)
	mux.HandleFunc("GET /api/v1/projects/{name}/milestones/{msId}", s.handleGetMilestone)
	mux.HandleFunc("PUT /api/v1/projects/{name}/milestones/{msId}", s.handleUpdateMilestone)
	mux.HandleFunc("DELETE /api/v1/projects/{name}/milestones/{msId}", s.handleDeleteMilestone)

	mux.HandleFunc("GET /api/v1/ccconnect/config", s.handleCCConnectGetConfig)
	mux.HandleFunc("PUT /api/v1/ccconnect/config", s.handleCCConnectPutConfig)
	mux.HandleFunc("POST /api/v1/ccconnect/test", s.handleCCConnectTest)
	mux.HandleFunc("GET /api/v1/ccconnect/projects", s.handleCCProxyProjects)
	mux.HandleFunc("GET /api/v1/ccconnect/projects/{name}", s.handleCCProxyProjectDetail)
	mux.HandleFunc("DELETE /api/v1/ccconnect/projects/{name}", s.handleCCProxyProjectDelete)
	mux.HandleFunc("POST /api/v1/ccconnect/projects/{name}/add-platform", s.handleCCProxyAddPlatform)
	mux.HandleFunc("GET /api/v1/ccconnect/projects/{name}/sessions", s.handleCCProxyProjectSessions)
	mux.HandleFunc("POST /api/v1/ccconnect/setup/feishu/begin", s.handleCCProxySetupFeishuBegin)
	mux.HandleFunc("POST /api/v1/ccconnect/setup/feishu/poll", s.handleCCProxySetupFeishuPoll)
	mux.HandleFunc("POST /api/v1/ccconnect/setup/feishu/save", s.handleCCProxySetupFeishuSave)
	mux.HandleFunc("POST /api/v1/ccconnect/setup/weixin/begin", s.handleCCProxySetupWeixinBegin)
	mux.HandleFunc("POST /api/v1/ccconnect/setup/weixin/poll", s.handleCCProxySetupWeixinPoll)
	mux.HandleFunc("POST /api/v1/ccconnect/setup/weixin/save", s.handleCCProxySetupWeixinSave)
	mux.HandleFunc("POST /api/v1/ccconnect/reload", s.handleCCProxyReload)
	mux.HandleFunc("POST /api/v1/ccconnect/restart", s.handleCCProxyRestart)

	// ── Files ──
	mux.HandleFunc("GET /api/v1/files", s.handleListFiles)
	mux.HandleFunc("POST /api/v1/files/upload", s.handleUploadFile)
	mux.HandleFunc("POST /api/v1/files/mkdir", s.handleMkdir)
	mux.HandleFunc("POST /api/v1/files/move", s.handleMoveFile)
	mux.HandleFunc("GET /api/v1/files/content/{path...}", s.handleFileContent)
	mux.HandleFunc("DELETE /api/v1/files/{path...}", s.handleDeleteFile)

	mux.HandleFunc("GET /api/v1/check-update", s.handleCheckUpdate)
	mux.HandleFunc("GET /api/v1/daemon/status", s.handleDaemonStatus)

	publicMux := http.NewServeMux()
	publicMux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	publicMux.HandleFunc("GET /api/v1/health", s.handleHealth)
	publicMux.Handle("/", s.withTokenAuth(mux))

	return withJSONHeaders(publicMux)
}

func withJSONHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/download") && !strings.Contains(r.URL.Path, "/files/content/") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		} else if t := r.URL.Query().Get("_token"); t != "" {
			token = t
		}
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

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
		if u := s.users.GetUser(username); u != nil && u.Disabled {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "account disabled"})
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

func (s *Server) handleCheckUpdate(w http.ResponseWriter, _ *http.Request) {
	result := map[string]any{
		"currentVersion": s.version,
		"hasUpdate":      false,
	}
	if s.updateCheck != nil {
		latest, notes, has := s.updateCheck()
		result["hasUpdate"] = has
		if latest != "" {
			result["latestVersion"] = latest
		}
		if notes != "" {
			result["releaseNotes"] = notes
		}
	}
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) handleDaemonStatus(w http.ResponseWriter, _ *http.Request) {
	if s.daemonStatus != nil {
		_ = json.NewEncoder(w).Encode(s.daemonStatus())
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"supported": false, "error": "daemon status not available"})
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

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.st.ListProjects()
	if err != nil {
		s.serverError(w, err)
		return
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
	cur := s.currentUser(r)
	out := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		if cur.Role != RoleAdmin {
			if _, ok := s.users.HasProjectAccess(cur.Username, p.Name); !ok {
				continue
			}
		}
		out = append(out, map[string]any{
			"name":        p.Name,
			"description": p.Description,
			"repo":        p.Repo,
		})
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) checkProjectAccess(w http.ResponseWriter, r *http.Request, project string) bool {
	if !s.canAccessProject(r, project) {
		s.jsonError(w, http.StatusForbidden, "no access to this project")
		return false
	}
	return true
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !s.checkProjectAccess(w, r, name) {
		return
	}
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
	if !s.checkProjectAccess(w, r, name) {
		return
	}
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
	ID          string    `json:"id"`
	Project     string    `json:"project"`
	Agent       string    `json:"agent"`
	Title       string    `json:"title"`
	Type        string    `json:"type,omitempty"`
	Assignee    string    `json:"assignee,omitempty"`
	Description string    `json:"description,omitempty"`
	Prompt      string    `json:"prompt,omitempty"`
	Priority    int       `json:"priority"`
	Status      string    `json:"status"`
	StatusGroup string    `json:"statusGroup"`
	Archived    bool      `json:"archived"`
	Summary     string    `json:"summary,omitempty"`
	Labels      []string  `json:"labels"`
	ParentID    string    `json:"parentId,omitempty"`
	Position    float64   `json:"position"`
	CreatedBy   string    `json:"createdBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	DueDate     string    `json:"dueDate,omitempty"`
}

func taskToRow(t *entity.Task, project, agent string, archived bool) taskRow {
	labels := t.Labels
	if labels == nil {
		labels = []string{}
	}
	r := taskRow{
		ID: t.ID, Project: project, Agent: agent,
		Title: t.Title, Type: string(t.Type), Assignee: t.Assignee,
		Description: t.Description, Prompt: t.Prompt,
		Priority: t.Priority, Status: string(t.Status),
		StatusGroup: string(t.Status.Group()),
		Archived: archived, Summary: t.Summary,
		Labels: labels, ParentID: t.ParentID, Position: t.Position,
		CreatedBy: t.CreatedBy,
		CreatedAt: t.CreatedAt.UTC(), UpdatedAt: t.UpdatedAt.UTC(),
	}
	if t.DueDate != nil {
		r.DueDate = t.DueDate.UTC().Format("2006-01-02")
	}
	return r
}

func (s *Server) handleProjectTasks(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !s.checkProjectAccess(w, r, name) {
		return
	}
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
	qLabel := r.URL.Query().Get("label")
	qParent := r.URL.Query().Get("parent_id")
	qGroup := r.URL.Query().Get("status_group") // backlog, active, done
	qScope := r.URL.Query().Get("scope")         // "active" (default), "archived", "all"
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
		if qGroup != "" && string(t.Status.Group()) != qGroup {
			return false
		}
		if qLabel != "" {
			found := false
			for _, l := range t.Labels {
				if l == qLabel {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		if qParent != "" && t.ParentID != qParent {
			return false
		}
		return true
	}

	rows := make([]taskRow, 0)
	seenIDs := make(map[string]bool)
	for _, ag := range agents {
		if qAgent != "" && ag.Name != qAgent {
			continue
		}
		if qScope == "active" || qScope == "all" {
			tasks, err := s.ts.ListTasks(name, ag.Name)
			if err == nil {
				for _, t := range tasks {
					if !matchFilter(t) || seenIDs[t.ID] {
						continue
					}
					seenIDs[t.ID] = true
					rows = append(rows, taskToRow(t, name, ag.Name, false))
				}
			}
		}
		if qScope == "archived" || qScope == "all" {
			archived, err := s.ts.ListArchivedTasks(name, ag.Name)
			if err == nil {
				for _, t := range archived {
					if !matchFilter(t) || seenIDs[t.ID] {
						continue
					}
					seenIDs[t.ID] = true
					rows = append(rows, taskToRow(t, name, ag.Name, true))
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
	if !s.checkProjectAccess(w, r, name) {
		return
	}
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
