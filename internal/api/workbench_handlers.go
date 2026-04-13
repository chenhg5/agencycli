package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
)

type msgWithMailbox struct {
	msg     *entity.Message
	mailbox string
}

func (s *Server) handleWorkbenchMessages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	archivedMode := strings.TrimSpace(strings.ToLower(q.Get("archived")))
	if archivedMode == "" {
		archivedMode = "no"
	}
	readFilter := strings.TrimSpace(strings.ToLower(q.Get("read")))
	if readFilter == "" {
		readFilter = "all"
	}
	fromQ := strings.TrimSpace(q.Get("from"))
	direction := strings.TrimSpace(strings.ToLower(q.Get("direction")))
	if direction == "" {
		direction = "all"
	}

	cur := s.currentUser(r)
	isAdmin := cur.Role == RoleAdmin
	useAll := archivedMode == "all" || archivedMode == "yes"
	seen := map[string]bool{}
	var msgs []*msgWithMailbox

	if isAdmin {
		// Admin: show all messages to/from human
		if direction == "inbox" || direction == "all" {
			var raw []*entity.Message
			var err error
			if useAll {
				raw, err = s.ts.ListAllMessages("human")
			} else {
				raw, err = s.ts.ListMessages("human")
			}
			if err != nil {
				s.serverError(w, err)
				return
			}
			for _, m := range raw {
				if m != nil && !seen[m.ID] {
					seen[m.ID] = true
					msgs = append(msgs, &msgWithMailbox{m, "human"})
				}
			}
		}
		if direction == "sent" || direction == "all" {
			projects, err := s.ts.ListProjects()
			if err != nil {
				s.serverError(w, err)
				return
			}
			for _, proj := range projects {
				agents, err := s.ts.ListAgents(proj)
				if err != nil {
					continue
				}
				for _, ag := range agents {
					mailbox := proj + "/" + ag
					var raw []*entity.Message
					if useAll {
						raw, _ = s.ts.ListAllMessages(mailbox)
					} else {
						raw, _ = s.ts.ListMessages(mailbox)
					}
					for _, m := range raw {
						if m != nil && m.From == "human" && !seen[m.ID] {
							seen[m.ID] = true
							msgs = append(msgs, &msgWithMailbox{m, mailbox})
						}
					}
				}
			}
		}
	} else {
		// Member: show messages for their linked agents only
		linkedAgents := cur.LinkedAgents
		for _, la := range linkedAgents {
			if direction == "inbox" || direction == "all" {
				var raw []*entity.Message
				if useAll {
					raw, _ = s.ts.ListAllMessages(la)
				} else {
					raw, _ = s.ts.ListMessages(la)
				}
				for _, m := range raw {
					if m != nil && !seen[m.ID] {
						seen[m.ID] = true
						msgs = append(msgs, &msgWithMailbox{m, la})
					}
				}
			}
		}
	}

	rows := make([]msgRow, 0, len(msgs))
	for _, mw := range msgs {
		m := mw.msg
		if !messagePassesFilters(m, archivedMode, readFilter, fromQ, "") {
			continue
		}
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
			Mailbox:    mw.mailbox,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].SentAt.After(rows[j].SentAt) })
	_ = json.NewEncoder(w).Encode(rows)
}

func (s *Server) handleWorkbenchTasks(w http.ResponseWriter, r *http.Request) {
	projects, err := s.ts.ListProjects()
	if err != nil {
		s.serverError(w, err)
		return
	}
	q := r.URL.Query()
	statusFilter := strings.TrimSpace(q.Get("status"))
	projectFilter := strings.TrimSpace(q.Get("project"))

	cur := s.currentUser(r)

	isWakeup := func(t *entity.Task) bool {
		return strings.HasPrefix(t.Title, "[wakeup]") || t.Type == "wakeup"
	}

	linkedSet := map[string]bool{}
	for _, la := range cur.LinkedAgents {
		linkedSet[la] = true
	}

	rows := make([]taskRow, 0)
	for _, proj := range projects {
		if projectFilter != "" && proj != projectFilter {
			continue
		}
		if cur.Role != RoleAdmin {
			if _, ok := s.users.HasProjectAccess(cur.Username, proj); !ok {
				continue
			}
		}
		agents, err := s.ts.ListAgents(proj)
		if err != nil {
			continue
		}
		for _, ag := range agents {
			active, _ := s.ts.ListTasks(proj, ag)
			archived, _ := s.ts.ListArchivedTasks(proj, ag)
			all := append(active, archived...)
			seenTask := map[string]bool{}
			for _, t := range all {
				if t == nil || isWakeup(t) {
					continue
				}
				if seenTask[t.ID] {
					continue
				}
				seenTask[t.ID] = true
				if cur.Role == RoleAdmin {
					if ag != "human" && t.Assignee != "human" {
						continue
					}
				} else {
					agentID := proj + "/" + ag
					if !linkedSet[agentID] && t.Assignee != cur.Username {
						continue
					}
				}
				if statusFilter != "" && string(t.Status) != statusFilter {
					continue
				}
				isArchived := t.Status.IsTerminal() && !containsTask(active, t.ID)
				rows = append(rows, taskToRow(t, proj, ag, isArchived))
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt.After(rows[j].UpdatedAt) })
	_ = json.NewEncoder(w).Encode(rows)
}

func containsTask(tasks []*entity.Task, id string) bool {
	for _, t := range tasks {
		if t != nil && t.ID == id {
			return true
		}
	}
	return false
}

// ── Project overview for workbench ──────────────────────────────────────────

type projectOverview struct {
	Project          string `json:"project"`
	AgentCount       int    `json:"agentCount"`
	HeartbeatEnabled int    `json:"heartbeatEnabled"`
	RunningAgents    int    `json:"runningAgents"`
	SchedulerRunning bool   `json:"schedulerRunning"`
	PendingTasks     int    `json:"pendingTasks"`
	RunningTasks     int    `json:"runningTasks"`
	CompletedTasks   int    `json:"completedTasks"`
	TotalTasks       int    `json:"totalTasks"`
	UnreadMessages   int    `json:"unreadMessages"`
	TotalMessages    int    `json:"totalMessages"`
}

func (s *Server) handleWorkbenchOverview(w http.ResponseWriter, r *http.Request) {
	projects, err := s.ts.ListProjects()
	if err != nil {
		s.serverError(w, err)
		return
	}

	cur := s.currentUser(r)

	schedStatuses := s.sched.Status()
	schedRunning := map[string]bool{}
	for _, ss := range schedStatuses {
		if ss.Running {
			schedRunning[ss.Key] = true
		}
	}

	rows := make([]projectOverview, 0, len(projects))
	for _, proj := range projects {
		if cur.Role != RoleAdmin {
			if _, ok := s.users.HasProjectAccess(cur.Username, proj); !ok {
				continue
			}
		}
		agentNames, err := s.ts.ListAgents(proj)
		if err != nil {
			continue
		}

		ov := projectOverview{Project: proj, AgentCount: len(agentNames)}

		for _, ag := range agentNames {
			hb, err := s.ts.GetHeartbeat(proj, ag)
			if err == nil {
				if hb.Enabled {
					ov.HeartbeatEnabled++
				}
				if hb.LastWakeupStatus == "running" && hb.PID > 0 {
					ov.RunningAgents++
				}
			}
		}

		if schedRunning["all"] || schedRunning[proj] {
			ov.SchedulerRunning = true
		} else {
			for _, ag := range agentNames {
				if schedRunning[proj+"/"+ag] {
					ov.SchedulerRunning = true
					break
				}
			}
		}

		isWakeup := func(t *entity.Task) bool {
			return strings.HasPrefix(t.Title, "[wakeup]") || t.Type == "wakeup"
		}
		for _, ag := range agentNames {
			tasks, _ := s.ts.ListTasks(proj, ag)
			for _, t := range tasks {
				if t == nil || isWakeup(t) {
					continue
				}
				ov.TotalTasks++
				switch {
				case t.Status == entity.TaskStatusPending:
					ov.PendingTasks++
				case t.Status == entity.TaskStatusInProgress:
					ov.RunningTasks++
				case t.Status.IsTerminal():
					ov.CompletedTasks++
				}
			}
		}

		for _, ag := range agentNames {
			mailbox := proj + "/" + ag
			msgs, _ := s.ts.ListMessages(mailbox)
			for _, m := range msgs {
				if m == nil {
					continue
				}
				ov.TotalMessages++
				if m.ReadAt == nil {
					ov.UnreadMessages++
				}
			}
		}

		rows = append(rows, ov)
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Project < rows[j].Project })
	_ = json.NewEncoder(w).Encode(rows)
}
