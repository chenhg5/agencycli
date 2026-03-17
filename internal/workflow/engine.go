// Package workflow implements the async routing engine for agencycli workflows.
//
// When a task that belongs to a workflow completes (success or failure), Route()
// is called.  It loads the workflow manifest, finds all matching routes, and
// either enqueues a new task for another agent or routes to the human inbox.
//
// Variable interpolation supports:
//
//	{{task.summary}}        — completing task's summary
//	{{task.error}}          — completing task's error message (on failure)
//	{{task.vars.KEY}}       — vars that were set when the task was created
//	{{steps.TMPL.summary}}  — summary of any previously completed template step
//	{{inputs.KEY}}          — workflow instance input parameters
package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
)

// Route is the entry-point called after a task reaches a terminal state.
// It is a no-op for tasks that are not part of a workflow.
// Idempotent: if task.RoutedAt is already set the call is a no-op.
func Route(root, project string, task *entity.Task,
	ts taskstore.Store, s store.Store) error {

	if task.WorkflowID == "" || task.TemplateID == "" {
		return nil // not a workflow task
	}
	if task.RoutedAt != nil {
		return nil // already routed
	}

	wf, inst, err := loadContext(ts, project, task)
	if err != nil {
		return err
	}
	if wf == nil || inst == nil {
		return nil
	}

	// Record the completed step's output in the instance.
	if inst.StepOutputs == nil {
		inst.StepOutputs = map[string]string{}
	}
	if task.Summary != "" {
		inst.StepOutputs[task.TemplateID] = task.Summary
	}

	// Build the variable context for interpolation.
	vars := buildVars(task, inst)

	// Determine terminal status string for route matching.
	statusStr := "failed"
	if task.Status == entity.TaskStatusDoneSuccess {
		statusStr = "success"
	}

	fired := 0
	for _, route := range wf.Routes {
		if !matchesRoute(route.On, task.TemplateID, statusStr) {
			continue
		}
		// Circuit-breaker: check max_trigger.
		routeKey := fmt.Sprintf("%s:%s", route.On.Template, route.On.Status)
		if route.On.MaxTrigger > 0 {
			if inst.RouteTriggers == nil {
				inst.RouteTriggers = map[string]int{}
			}
			if inst.RouteTriggers[routeKey] >= route.On.MaxTrigger {
				fmt.Printf("[workflow %s] route %s hit max_trigger=%d, skipping\n",
					inst.ID, routeKey, route.On.MaxTrigger)
				continue
			}
			inst.RouteTriggers[routeKey]++
		}

		if route.Create != nil {
			if err := fireCreateRoute(wf, route.Create, vars, inst, project, task, ts); err != nil {
				fmt.Printf("[workflow %s] create route error: %v\n", inst.ID, err)
			} else {
				fired++
			}
		}
		if route.Inbox != nil {
			if err := fireInboxRoute(route.Inbox, vars, inst, project, task, ts); err != nil {
				fmt.Printf("[workflow %s] inbox route error: %v\n", inst.ID, err)
			} else {
				fired++
			}
		}
	}

	// Mark task as routed.
	now := time.Now().UTC()
	task.RoutedAt = &now

	// Persist instance state.
	_ = ts.SaveWorkflowInstance(project, inst)

	if fired == 0 && len(matchingRoutes(wf, task.TemplateID, statusStr)) == 0 {
		// No routes matched at all — mark workflow done.
		inst.Status = "done"
		t := time.Now().UTC()
		inst.FinishedAt = &t
		_ = ts.SaveWorkflowInstance(project, inst)
	}

	return nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

func loadContext(ts taskstore.Store, project string, task *entity.Task) (
	*entity.WorkflowManifest, *entity.WorkflowInstance, error) {

	inst, err := ts.GetWorkflowInstance(project, task.WorkflowID)
	if err != nil {
		return nil, nil, fmt.Errorf("load workflow instance %s: %w", task.WorkflowID, err)
	}

	wf, err := ts.GetWorkflow(project, inst.Workflow)
	if err != nil {
		return nil, nil, fmt.Errorf("load workflow %s: %w", inst.Workflow, err)
	}
	if wf == nil {
		return nil, nil, fmt.Errorf("workflow %q not found", inst.Workflow)
	}
	return wf, inst, nil
}

func matchesRoute(cond entity.WFCondition, templateID, status string) bool {
	if cond.Template != templateID {
		return false
	}
	if cond.Status == "any" {
		return true
	}
	return cond.Status == status
}

func matchingRoutes(wf *entity.WorkflowManifest, templateID, status string) []entity.WFRoute {
	var out []entity.WFRoute
	for _, r := range wf.Routes {
		if matchesRoute(r.On, templateID, status) {
			out = append(out, r)
		}
	}
	return out
}

func buildVars(task *entity.Task, inst *entity.WorkflowInstance) map[string]string {
	vars := map[string]string{}
	// Instance inputs.
	for k, v := range inst.Inputs {
		vars["inputs."+k] = v
	}
	// All prior step outputs.
	for tmplID, summary := range inst.StepOutputs {
		vars["steps."+tmplID+".summary"] = summary
	}
	// Triggering task values.
	vars["task.summary"] = task.Summary
	vars["task.error"] = task.LastError
	for k, v := range task.Vars {
		vars["task.vars."+k] = v
	}
	return vars
}

// interpolate replaces {{key}} placeholders in text.
func interpolate(text string, vars map[string]string) string {
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{{"+k+"}}", v)
	}
	return text
}

func interpolateMap(m map[string]string, vars map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = interpolate(v, vars)
	}
	return out
}

func findTemplate(wf *entity.WorkflowManifest, id string) *entity.WFTaskTemplate {
	for i := range wf.Templates {
		if wf.Templates[i].ID == id {
			return &wf.Templates[i]
		}
	}
	return nil
}

// fireCreateRoute enqueues a new task for the target agent.
func fireCreateRoute(wf *entity.WorkflowManifest, cr *entity.WFCreate,
	vars map[string]string, inst *entity.WorkflowInstance,
	project string, triggerTask *entity.Task, ts taskstore.Store) error {

	tmpl := findTemplate(wf, cr.Template)
	if tmpl == nil {
		return fmt.Errorf("template %q not found in workflow %s", cr.Template, wf.Name)
	}

	// Merge route vars on top of trigger vars.
	merged := make(map[string]string, len(vars))
	for k, v := range vars {
		merged[k] = v
	}
	for k, v := range cr.Vars {
		merged[k] = interpolate(v, vars)
	}

	priority := tmpl.Priority
	if priority == 0 {
		priority = 1 // default high
	}

	task := &entity.Task{
		ID:         entity.NewTaskID(),
		Title:      interpolate(tmpl.Title, merged),
		Type:       entity.TaskType(tmpl.Type),
		Priority:   priority,
		Assignee:   project + "/" + tmpl.Agent,
		CreatedBy:  "workflow:" + inst.ID,
		Status:     entity.TaskStatusPending,
		Prompt:     interpolate(tmpl.Prompt, merged),
		WorkflowID: inst.ID,
		TemplateID: tmpl.ID,
		Vars:       interpolateMap(cr.Vars, vars),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	// Preserve key vars that might be needed by downstream routes.
	if task.Vars == nil {
		task.Vars = map[string]string{}
	}
	for k, v := range triggerTask.Vars {
		if _, exists := task.Vars[k]; !exists {
			task.Vars[k] = v
		}
	}

	if err := ts.AddTask(project, tmpl.Agent, task); err != nil {
		return fmt.Errorf("add task for %s: %w", tmpl.Agent, err)
	}

	fmt.Printf("[workflow %s] → enqueued [%s] \"%s\" for agent %s\n",
		inst.ID, tmpl.ID, task.Title, tmpl.Agent)
	return nil
}

// fireInboxRoute adds an item to the human inbox.
func fireInboxRoute(inbox *entity.WFInbox, vars map[string]string,
	inst *entity.WorkflowInstance, project string,
	task *entity.Task, ts taskstore.Store) error {

	actionItems := make([]string, len(inbox.ActionItems))
	for i, ai := range inbox.ActionItems {
		actionItems[i] = interpolate(ai, vars)
	}

	item := &entity.InboxItem{
		TaskID:      task.ID,
		Project:     project,
		Agent:       task.Assignee,
		Title:       interpolate(inbox.Title, vars),
		Summary:     interpolate(inbox.Summary, vars),
		ActionItems: actionItems,
		RoutedAt:    time.Now().UTC(),
		LogPath:     task.RunLogPath,
	}
	if err := ts.AddToInbox(item); err != nil {
		return err
	}
	fmt.Printf("[workflow %s] → routed to human inbox: %s\n", inst.ID, item.Title)
	return nil
}
