package api

import (
	"fmt"
	"strings"
)

// buildGoalSummary generates a compact Markdown summary of active OKRs and
// project milestones for injection into an Agent's context. Returns empty
// string when there is nothing relevant.
func (s *Server) buildGoalSummary(project string) string {
	var sb strings.Builder

	if f, err := s.okrStore.Load(); err == nil && len(f.OKRs) > 0 {
		sb.WriteString("## Current OKRs\n\n")
		for _, o := range f.OKRs {
			if o.Status == "achieved" {
				continue
			}
			sb.WriteString(fmt.Sprintf("### O: %s  [%s] %.0f%%\n", o.Objective, o.Status, o.Progress()))
			for _, kr := range o.KeyResults {
				check := "  "
				if kr.Progress() >= 100 {
					check = "✓ "
				}
				sb.WriteString(fmt.Sprintf("- %sKR: %s (%.0f/%.0f %s) %.0f%%\n",
					check, kr.Description, kr.CurrentValue, kr.TargetValue, kr.Unit, kr.Progress()))
			}
			sb.WriteString("\n")
		}
	}

	if milestones, err := s.msStore.List(project); err == nil && len(milestones) > 0 {
		sb.WriteString("## Project Milestones\n\n")
		for _, ms := range milestones {
			if ms.Status == "completed" || ms.Status == "cancelled" {
				continue
			}
			due := ""
			if ms.DueDate != nil {
				due = fmt.Sprintf(" (due %s)", ms.DueDate.Format("2006-01-02"))
			}
			sb.WriteString(fmt.Sprintf("- **%s** [%s] %d%%%s\n", ms.Title, ms.Status, ms.Progress, due))
			for _, c := range ms.Criteria {
				sb.WriteString(fmt.Sprintf("  - [ ] %s\n", c))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
