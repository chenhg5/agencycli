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

	useAll := archivedMode == "all" || archivedMode == "yes"
	var msgs []*msgWithMailbox
	var err error

	if useAll {
		raw, e := s.ts.ListAllMessages("human")
		err = e
		for _, m := range raw {
			if m != nil {
				msgs = append(msgs, &msgWithMailbox{m, "human"})
			}
		}
	} else {
		raw, e := s.ts.ListMessages("human")
		err = e
		for _, m := range raw {
			if m != nil {
				msgs = append(msgs, &msgWithMailbox{m, "human"})
			}
		}
	}
	if err != nil {
		s.serverError(w, err)
		return
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
			Mailbox:    "human",
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

	isWakeup := func(t *entity.Task) bool {
		return strings.HasPrefix(t.Title, "[wakeup]") || t.Type == "wakeup"
	}

	rows := make([]taskRow, 0)
	for _, proj := range projects {
		if projectFilter != "" && proj != projectFilter {
			continue
		}
		agents, err := s.ts.ListAgents(proj)
		if err != nil {
			continue
		}
		for _, ag := range agents {
			active, _ := s.ts.ListTasks(proj, ag)
			archived, _ := s.ts.ListArchivedTasks(proj, ag)
			all := append(active, archived...)
			for _, t := range all {
				if t == nil || isWakeup(t) {
					continue
				}
				if statusFilter != "" && string(t.Status) != statusFilter {
					continue
				}
				isArchived := t.Status.IsTerminal() && !containsTask(active, t.ID)
				rows = append(rows, taskRow{
					ID:        t.ID,
					Project:   proj,
					Agent:     ag,
					Title:     t.Title,
					Type:      string(t.Type),
					Assignee:  t.Assignee,
					Prompt:    t.Prompt,
					Priority:  int(t.Priority),
					Status:    string(t.Status),
					Archived:  isArchived,
					CreatedAt: t.CreatedAt,
					UpdatedAt: t.UpdatedAt,
				})
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
