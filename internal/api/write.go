package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
)

const maxJSONBody = 1 << 20 // 1 MiB

func (s *Server) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func validTaskType(s string) bool {
	switch entity.TaskType(s) {
	case entity.TaskTypeFeature, entity.TaskTypeBug, entity.TaskTypeReview,
		entity.TaskTypeTriage, entity.TaskTypeTest, entity.TaskTypeResearch, entity.TaskTypeChore:
		return true
	default:
		return false
	}
}

func validTaskStatus(s string) bool {
	switch entity.TaskStatus(s) {
	case entity.TaskStatusPending, entity.TaskStatusInProgress, entity.TaskStatusAwaitingConfirmation,
		entity.TaskStatusBlocked, entity.TaskStatusDoneSuccess, entity.TaskStatusDoneFailed, entity.TaskStatusCancelled:
		return true
	default:
		return false
	}
}

type postTaskBody struct {
	Agent       string   `json:"agent"`
	Title       string   `json:"title"`
	Prompt      string   `json:"prompt"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Priority    int      `json:"priority"`
	Assignee    string   `json:"assignee"`
	CreatedBy   string   `json:"createdBy"`
	Labels      []string `json:"labels"`
	ParentID    string   `json:"parentId"`
	DueDate     string   `json:"dueDate"` // YYYY-MM-DD
}

func (s *Server) handlePostProjectTask(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.st.Project(name); err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "project not found")
			return
		}
		s.serverError(w, err)
		return
	}

	var body postTaskBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	agentName := strings.TrimSpace(body.Agent)
	title := strings.TrimSpace(body.Title)
	promptText := strings.TrimSpace(body.Prompt)
	if agentName == "" || title == "" || promptText == "" {
		s.jsonError(w, http.StatusBadRequest, "agent, title, and prompt are required")
		return
	}
	if !s.agentExistsInProject(name, agentName) {
		s.jsonError(w, http.StatusBadRequest, "agent not found in this project")
		return
	}

	taskType := strings.TrimSpace(body.Type)
	if taskType == "" {
		taskType = string(entity.TaskTypeChore)
	}
	if !validTaskType(taskType) {
		s.jsonError(w, http.StatusBadRequest, "invalid task type")
		return
	}

	priority := body.Priority
	if priority < 0 || priority > 3 {
		s.jsonError(w, http.StatusBadRequest, "priority must be 0–3")
		return
	}

	assignee := strings.TrimSpace(body.Assignee)
	if assignee == "" {
		assignee = name + "/" + agentName
	}

	createdBy := strings.TrimSpace(body.CreatedBy)
	if createdBy == "" {
		createdBy = "human"
	}

	now := time.Now().UTC()
	t := &entity.Task{
		ID:          entity.NewTaskID(),
		Title:       title,
		Description: strings.TrimSpace(body.Description),
		Type:        entity.TaskType(taskType),
		Priority:    priority,
		Assignee:    assignee,
		CreatedBy:   createdBy,
		Status:      entity.TaskStatusPending,
		Prompt:      promptText,
		Labels:      body.Labels,
		ParentID:    strings.TrimSpace(body.ParentID),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if body.DueDate != "" {
		if dd, err := time.Parse("2006-01-02", body.DueDate); err == nil {
			t.DueDate = &dd
		}
	}

	if assignee == "human" {
		if err := s.ts.AddTask(name, agentName, t); err != nil {
			s.serverError(w, err)
			return
		}
		item := &entity.InboxItem{
			TaskID:  t.ID,
			Project: name,
			Agent:   agentName,
			Title:   t.Title,
			Summary: promptText,
		}
		if err := s.ts.AddToInbox(item); err != nil {
			s.serverError(w, err)
			return
		}
	} else {
		if err := s.validateIdentity(assignee, "assignee"); err != nil {
			s.jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.ts.AddTask(name, agentName, t); err != nil {
			s.serverError(w, err)
			return
		}
	}

	// Fire trigger if configured for this agent.
	s.triggers.Fire(name, agentName, entity.TriggerOnTask, "task "+t.ID)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":      t.ID,
		"project": name,
		"agent":   agentName,
	})
}

type taskActionBody struct {
	Project string `json:"project"`
	Agent   string `json:"agent"`
	ID      string `json:"id"`
}

func (s *Server) handlePostCancelTask(w http.ResponseWriter, r *http.Request) {
	var body taskActionBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)
	id := strings.TrimSpace(body.ID)
	if project == "" || agent == "" || id == "" {
		s.jsonError(w, http.StatusBadRequest, "project, agent, and id are required")
		return
	}
	t, err := s.ts.GetTask(project, agent, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "task not found")
			return
		}
		s.serverError(w, err)
		return
	}
	if t.Status.IsTerminal() {
		s.jsonError(w, http.StatusBadRequest, "task is already in terminal state")
		return
	}
	t.Status = entity.TaskStatusCancelled
	t.UpdatedAt = time.Now().UTC()
	if err := s.ts.UpdateTask(project, agent, t); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handlePostArchiveTask(w http.ResponseWriter, r *http.Request) {
	var body taskActionBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)
	id := strings.TrimSpace(body.ID)
	if project == "" || agent == "" || id == "" {
		s.jsonError(w, http.StatusBadRequest, "project, agent, and id are required")
		return
	}
	t, err := s.ts.GetTask(project, agent, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "task not found")
			return
		}
		s.serverError(w, err)
		return
	}
	if err := s.ts.ArchiveTask(project, agent, t); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

type updateTaskBody struct {
	Project     string    `json:"project"`
	Agent       string    `json:"agent"`
	ID          string    `json:"id"`
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Status      *string   `json:"status,omitempty"`
	Priority    *int      `json:"priority,omitempty"`
	Type        *string   `json:"type,omitempty"`
	Summary     *string   `json:"summary,omitempty"`
	Labels      *[]string `json:"labels,omitempty"`
	ParentID    *string   `json:"parentId,omitempty"`
	DueDate     *string   `json:"dueDate,omitempty"` // YYYY-MM-DD or "" to clear
	Position    *float64  `json:"position,omitempty"`
	Assignee    *string   `json:"assignee,omitempty"`
	Prompt      *string   `json:"prompt,omitempty"`
}

func (s *Server) handlePutUpdateTask(w http.ResponseWriter, r *http.Request) {
	var body updateTaskBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)
	id := strings.TrimSpace(body.ID)
	if project == "" || agent == "" || id == "" {
		s.jsonError(w, http.StatusBadRequest, "project, agent, and id are required")
		return
	}
	hasUpdate := body.Status != nil || body.Priority != nil || body.Type != nil || body.Summary != nil ||
		body.Title != nil || body.Description != nil || body.Labels != nil || body.ParentID != nil ||
		body.DueDate != nil || body.Position != nil || body.Assignee != nil || body.Prompt != nil
	if !hasUpdate {
		s.jsonError(w, http.StatusBadRequest, "at least one field to update is required")
		return
	}

	t, err := s.ts.GetTask(project, agent, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "task not found")
			return
		}
		s.serverError(w, err)
		return
	}

	if body.Status != nil {
		st := strings.TrimSpace(*body.Status)
		if st == "" {
			s.jsonError(w, http.StatusBadRequest, "status cannot be empty")
			return
		}
		if !validTaskStatus(st) {
			s.jsonError(w, http.StatusBadRequest, "invalid task status")
			return
		}
		t.Status = entity.TaskStatus(st)
	}
	if body.Priority != nil {
		p := *body.Priority
		if p < 0 || p > 3 {
			s.jsonError(w, http.StatusBadRequest, "priority must be 0–3")
			return
		}
		t.Priority = p
	}
	if body.Type != nil {
		typ := strings.TrimSpace(*body.Type)
		if typ == "" {
			s.jsonError(w, http.StatusBadRequest, "type cannot be empty")
			return
		}
		if !validTaskType(typ) {
			s.jsonError(w, http.StatusBadRequest, "invalid task type")
			return
		}
		t.Type = entity.TaskType(typ)
	}
	if body.Summary != nil {
		t.Summary = strings.TrimSpace(*body.Summary)
	}
	if body.Title != nil {
		t.Title = strings.TrimSpace(*body.Title)
	}
	if body.Description != nil {
		t.Description = strings.TrimSpace(*body.Description)
	}
	if body.Labels != nil {
		t.Labels = *body.Labels
	}
	if body.ParentID != nil {
		t.ParentID = strings.TrimSpace(*body.ParentID)
	}
	if body.DueDate != nil {
		dd := strings.TrimSpace(*body.DueDate)
		if dd == "" {
			t.DueDate = nil
		} else if parsed, err := time.Parse("2006-01-02", dd); err == nil {
			t.DueDate = &parsed
		}
	}
	if body.Position != nil {
		t.Position = *body.Position
	}
	if body.Assignee != nil {
		t.Assignee = strings.TrimSpace(*body.Assignee)
	}
	if body.Prompt != nil {
		t.Prompt = strings.TrimSpace(*body.Prompt)
	}

	now := time.Now().UTC()
	t.UpdatedAt = now
	if t.Status.IsTerminal() && t.FinishedAt == nil {
		t.FinishedAt = &now
	}

	activeTasks, err := s.ts.ListTasks(project, agent)
	if err != nil {
		s.serverError(w, err)
		return
	}
	inActive := false
	for _, at := range activeTasks {
		if at.ID == id {
			inActive = true
			break
		}
	}
	if inActive {
		if err := s.ts.UpdateTask(project, agent, t); err != nil {
			s.serverError(w, err)
			return
		}
		if t.Status.IsTerminal() {
			if err := s.ts.ArchiveTask(project, agent, t); err != nil {
				s.serverError(w, err)
				return
			}
		}
	} else {
		archived, err := s.ts.ListArchivedTasks(project, agent)
		if err != nil {
			s.serverError(w, err)
			return
		}
		found := false
		for i, at := range archived {
			if at.ID == t.ID {
				archived[i] = t
				found = true
				break
			}
		}
		if !found {
			s.jsonError(w, http.StatusNotFound, "task not found")
			return
		}
		if err := s.ts.OverwriteArchive(project, agent, archived); err != nil {
			s.serverError(w, err)
			return
		}
	}

	if body.Status != nil && t.Status.IsTerminal() && t.CreatedBy != "" {
		s.notifyTaskDone(t, project, agent)
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handlePostDeleteTask(w http.ResponseWriter, r *http.Request) {
	var body taskActionBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	project := strings.TrimSpace(body.Project)
	agent := strings.TrimSpace(body.Agent)
	id := strings.TrimSpace(body.ID)
	if project == "" || agent == "" || id == "" {
		s.jsonError(w, http.StatusBadRequest, "project, agent, and id are required")
		return
	}
	if err := s.ts.DeleteTask(project, agent, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "task not found")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

type postMessageBody struct {
	From    string `json:"from"`
	To      any    `json:"to"` // string or []any (strings)
	Subject string `json:"subject"`
	Body    string `json:"body"`
	ReplyTo string `json:"replyTo"`
}

func normalizeToRecipients(v any) ([]string, error) {
	if v == nil {
		return nil, fmt.Errorf("to is required")
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil, fmt.Errorf("to is required")
		}
		return []string{s}, nil
	case []any:
		var out []string
		for _, e := range x {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("to must be strings")
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("to is required")
		}
		return out, nil
	default:
		return nil, fmt.Errorf("to must be a string or array of strings")
	}
}

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	var body postMessageBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	bodyText := strings.TrimSpace(body.Body)
	if bodyText == "" {
		s.jsonError(w, http.StatusBadRequest, "body is required")
		return
	}
	from := strings.TrimSpace(body.From)
	if from == "" {
		s.jsonError(w, http.StatusBadRequest, "from is required")
		return
	}
	if err := s.validateIdentity(from, "from"); err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	recipients, err := normalizeToRecipients(body.To)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	for _, rec := range recipients {
		if err := s.validateIdentity(rec, "to"); err != nil {
			s.jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	sentAt := time.Now().UTC()
	ids := make([]string, 0, len(recipients))
	replyTo := strings.TrimSpace(body.ReplyTo)
	subject := strings.TrimSpace(body.Subject)

	for _, recipient := range recipients {
		msg := &entity.Message{
			ID:      entity.NewMessageID(),
			From:    from,
			To:      recipient,
			Subject: subject,
			Body:    bodyText,
			ReplyTo: replyTo,
			SentAt:  sentAt,
		}
		if err := s.ts.SendMessage(msg); err != nil {
			s.serverError(w, fmt.Errorf("send to %s: %w", recipient, err))
			return
		}
		ids = append(ids, msg.ID)

		// Fire trigger if configured for the recipient agent.
		if parts := strings.SplitN(recipient, "/", 2); len(parts) == 2 {
			s.triggers.Fire(parts[0], parts[1], entity.TriggerOnMessage, "from "+from)
		}
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"ids": ids})
}

type markReadBody struct {
	Mailbox string `json:"mailbox"`
	ID      string `json:"id"`
}

func (s *Server) handlePostMarkMessageRead(w http.ResponseWriter, r *http.Request) {
	var body markReadBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	mailbox := strings.TrimSpace(body.Mailbox)
	id := strings.TrimSpace(body.ID)
	if mailbox == "" || id == "" {
		s.jsonError(w, http.StatusBadRequest, "mailbox and id are required")
		return
	}
	if err := s.validateIdentity(mailbox, "mailbox"); err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ts.MarkMessageRead(mailbox, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "message not found")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handlePostArchiveMessage(w http.ResponseWriter, r *http.Request) {
	var body markReadBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	mailbox := strings.TrimSpace(body.Mailbox)
	id := strings.TrimSpace(body.ID)
	if mailbox == "" || id == "" {
		s.jsonError(w, http.StatusBadRequest, "mailbox and id are required")
		return
	}
	if err := s.validateIdentity(mailbox, "mailbox"); err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ts.ArchiveMessage(mailbox, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "message not found")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handlePostDeleteMessage(w http.ResponseWriter, r *http.Request) {
	var body markReadBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	mailbox := strings.TrimSpace(body.Mailbox)
	id := strings.TrimSpace(body.ID)
	if mailbox == "" || id == "" {
		s.jsonError(w, http.StatusBadRequest, "mailbox and id are required")
		return
	}
	if err := s.validateIdentity(mailbox, "mailbox"); err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ts.DeleteMessage(mailbox, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, "message not found")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

type markAllMailboxBody struct {
	Mailbox string `json:"mailbox"`
}

func (s *Server) handlePostMarkAllMessagesRead(w http.ResponseWriter, r *http.Request) {
	var body markAllMailboxBody
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	mailbox := strings.TrimSpace(body.Mailbox)
	if mailbox == "" {
		s.jsonError(w, http.StatusBadRequest, "mailbox is required")
		return
	}
	if err := s.validateIdentity(mailbox, "mailbox"); err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ts.MarkMessagesRead(mailbox); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

type markAllProjectBody struct {
	// Mailbox is optional "project/agent" scoped to this project; empty = all agent mailboxes in project.
	Mailbox string `json:"mailbox"`
}

func (s *Server) handlePostProjectMarkAllMessagesRead(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.st.Project(name); err != nil {
		if isNotFoundErr(err) {
			s.jsonError(w, http.StatusNotFound, "project not found")
			return
		}
		s.serverError(w, err)
		return
	}

	var body markAllProjectBody
	if r.ContentLength > 0 {
		if err := s.readJSON(w, r, &body); err != nil {
			s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}
	mailbox := strings.TrimSpace(body.Mailbox)
	if mailbox != "" {
		if err := s.validateIdentity(mailbox, "mailbox"); err != nil {
			s.jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		parts := strings.SplitN(mailbox, "/", 2)
		if len(parts) != 2 || parts[0] != name {
			s.jsonError(w, http.StatusBadRequest, "mailbox must be an agent in this project")
			return
		}
		if !s.agentExistsInProject(name, parts[1]) {
			s.jsonError(w, http.StatusBadRequest, "agent not found in this project")
			return
		}
		if err := s.ts.MarkMessagesRead(mailbox); err != nil {
			s.serverError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "mailboxes": []string{mailbox}})
		return
	}

	agents, err := s.st.ListAgents(name)
	if err != nil {
		s.serverError(w, err)
		return
	}
	mailboxes := make([]string, 0, len(agents))
	for _, ag := range agents {
		mb := name + "/" + ag.Name
		mailboxes = append(mailboxes, mb)
		if err := s.ts.MarkMessagesRead(mb); err != nil {
			s.serverError(w, err)
			return
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "mailboxes": mailboxes})
}

func (s *Server) notifyTaskDone(t *entity.Task, project, agent string) {
	assignee := t.Assignee
	if assignee == "" {
		assignee = project + "/" + agent
	}
	if assignee == t.CreatedBy {
		return
	}

	statusLabel := "completed"
	if t.Status == entity.TaskStatusDoneFailed {
		statusLabel = "failed"
	} else if t.Status == entity.TaskStatusCancelled {
		statusLabel = "cancelled"
	}

	body := fmt.Sprintf("Task **%s** (`%s`) has been marked as **%s** by `%s`.",
		t.Title, t.ID, statusLabel, assignee)
	if t.Summary != "" {
		body += "\n\n**Summary:** " + t.Summary
	}

	msg := &entity.Message{
		ID:      entity.NewMessageID(),
		From:    assignee,
		To:      t.CreatedBy,
		Subject: fmt.Sprintf("[Task %s] %s", statusLabel, t.Title),
		Body:    body,
		SentAt:  time.Now().UTC(),
	}
	_ = s.ts.SendMessage(msg)
}

// ── Task Comments ─────────────────────────────────────────────────────────────

func (s *Server) handleGetComments(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	agent := r.PathValue("agent")
	taskID := r.PathValue("taskId")
	comments, err := s.ts.ListComments(project, agent, taskID)
	if err != nil {
		s.serverError(w, err)
		return
	}
	if comments == nil {
		comments = []*entity.TaskComment{}
	}
	_ = json.NewEncoder(w).Encode(comments)
}

func (s *Server) handlePostComment(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	agent := r.PathValue("agent")
	taskID := r.PathValue("taskId")
	var body struct {
		Author string `json:"author"`
		Body   string `json:"body"`
	}
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.Body = strings.TrimSpace(body.Body)
	body.Author = strings.TrimSpace(body.Author)
	if body.Body == "" {
		s.jsonError(w, http.StatusBadRequest, "body is required")
		return
	}
	if body.Author == "" {
		cur := s.currentUser(r)
		body.Author = cur.Username
	}
	c := &entity.TaskComment{
		ID:        entity.NewCommentID(),
		TaskID:    taskID,
		Author:    body.Author,
		Body:      body.Body,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.ts.AddComment(project, agent, c); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(c)
}

func (s *Server) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	agent := r.PathValue("agent")
	commentID := r.PathValue("commentId")
	if err := s.ts.DeleteComment(project, agent, commentID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleFireAgent(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("name")
	agent := r.PathValue("agent")
	if project == "" || agent == "" {
		s.jsonError(w, http.StatusBadRequest, "project and agent are required")
		return
	}
	if !s.checkProjectAccess(w, r, project) {
		return
	}
	if _, err := s.st.AgentMeta(project, agent); err != nil {
		s.jsonError(w, http.StatusNotFound, "agent not found")
		return
	}

	force := r.URL.Query().Get("force") == "true"
	agentDir := s.st.AgentDir(project, agent)

	if force {
		if err := os.RemoveAll(agentDir); err != nil {
			s.serverError(w, fmt.Errorf("fire: %w", err))
			return
		}
	} else {
		timestamp := time.Now().UTC().Format("20060102-150405")
		firedDirName := agent + "-" + timestamp
		firedDir := s.st.FiredAgentDir(project, firedDirName)
		if err := os.MkdirAll(filepath.Dir(firedDir), 0o755); err != nil {
			s.serverError(w, fmt.Errorf("fire: %w", err))
			return
		}
		if err := os.Rename(agentDir, firedDir); err != nil {
			s.serverError(w, fmt.Errorf("fire: %w", err))
			return
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
