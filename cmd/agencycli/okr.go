package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newOKRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "okr",
		Aliases: []string{"goals"},
		Short:   "Manage OKRs (Objectives and Key Results)",
	}
	cmd.AddCommand(
		newOKRListCmd(),
		newOKRCreateCmd(),
		newOKRShowCmd(),
		newOKRUpdateCmd(),
		newOKRDeleteCmd(),
		newOKRKRCmd(),
		newOKRReviewCmd(),
	)
	return cmd
}

func newOKRListCmd() *cobra.Command {
	var (
		quarter  string
		scope    string
		scopeRef string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List OKRs, optionally filtered by scope and quarter",
		Example: `  agencycli okr list
  agencycli okr list --quarter 2026-Q2
  agencycli okr list --scope agency
  agencycli okr list --scope project --scope-ref my-service
  agencycli okr list --scope agent --scope-ref my-service/dev`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewOKRStore(root)
			f, err := s.Load()
			if err != nil {
				return err
			}

			if f.CurrentQuarter != "" {
				fmt.Printf("Current quarter: %s\n\n", f.CurrentQuarter)
			}

			okrs, err := s.ListOKRs(entity.OKRScope(scope), scopeRef)
			if err != nil {
				return err
			}
			if quarter != "" {
				var filtered []entity.OKR
				for _, o := range okrs {
					if o.Quarter == quarter {
						filtered = append(filtered, o)
					}
				}
				okrs = filtered
			}

			if len(okrs) == 0 {
				fmt.Println("No OKRs found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "STATUS\tPROGRESS\tSCOPE\tID\tQUARTER\tOBJECTIVE")
			fmt.Fprintln(w, "──────\t────────\t─────\t──\t───────\t─────────")
			for _, o := range okrs {
				icon := statusIcon(string(o.Status))
				scopeStr := string(o.Scope)
				if o.ScopeRef != "" {
					scopeStr += ":" + o.ScopeRef
				}
				if scopeStr == "" {
					scopeStr = "agency"
				}
				fmt.Fprintf(w, "%s %s\t%.0f%%\t%s\t%s\t%s\t%s\n",
					icon, o.Status, o.Progress(), scopeStr, o.ID, o.Quarter, o.Objective)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&quarter, "quarter", "", "filter by quarter (e.g. 2026-Q2)")
	cmd.Flags().StringVar(&scope, "scope", "", "filter by scope (agency|project|agent)")
	cmd.Flags().StringVar(&scopeRef, "scope-ref", "", "scope reference (project name or project/agent)")
	return cmd
}

func newOKRCreateCmd() *cobra.Command {
	var (
		objective   string
		description string
		owner       string
		quarter     string
		scope       string
		scopeRef    string
		parentID    string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new OKR",
		Example: `  agencycli okr create --objective "Ship v2.0" --owner human --quarter 2026-Q2
  agencycli okr create --objective "Reduce latency" --scope project --scope-ref my-service
  agencycli okr create --objective "Agent autonomy" --scope agent --scope-ref my-service/dev --parent okr-xxx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if objective == "" {
				return fmt.Errorf("--objective is required")
			}
			s := store.NewOKRStore(root)
			okr := entity.OKR{
				Objective:   objective,
				Description: description,
				Owner:       owner,
				Quarter:     quarter,
				Scope:       entity.OKRScope(scope),
				ScopeRef:    scopeRef,
				ParentID:    parentID,
			}
			created, err := s.CreateOKR(okr)
			if err != nil {
				return err
			}
			fmt.Printf("✓ OKR %s created: %s (scope: %s)\n", created.ID, created.Objective, created.Scope)
			return nil
		},
	}
	cmd.Flags().StringVar(&objective, "objective", "", "objective text (required)")
	cmd.Flags().StringVar(&description, "description", "", "detailed description")
	cmd.Flags().StringVar(&owner, "owner", "human", "owner")
	cmd.Flags().StringVar(&quarter, "quarter", "", "quarter (e.g. 2026-Q2)")
	cmd.Flags().StringVar(&scope, "scope", "agency", "scope level (agency|project|agent)")
	cmd.Flags().StringVar(&scopeRef, "scope-ref", "", "scope reference (project name or project/agent)")
	cmd.Flags().StringVar(&parentID, "parent", "", "parent OKR ID for hierarchical alignment")
	return cmd
}

func newOKRShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <okr-id>",
		Short: "Show OKR details with Key Results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewOKRStore(root)
			o, err := s.GetOKR(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("ID        : %s\n", o.ID)
			fmt.Printf("Objective : %s\n", o.Objective)
			if o.Description != "" {
				fmt.Printf("Desc      : %s\n", o.Description)
			}
			fmt.Printf("Scope     : %s", o.Scope)
			if o.ScopeRef != "" {
				fmt.Printf(" (%s)", o.ScopeRef)
			}
			fmt.Println()
			if o.ParentID != "" {
				fmt.Printf("Parent    : %s\n", o.ParentID)
			}
			fmt.Printf("Status    : %s %s\n", statusIcon(string(o.Status)), o.Status)
			fmt.Printf("Progress  : %.0f%%\n", o.Progress())
			fmt.Printf("Owner     : %s\n", o.Owner)
			fmt.Printf("Quarter   : %s\n", o.Quarter)
			fmt.Printf("Created   : %s\n", o.CreatedAt.Format("2006-01-02 15:04"))

			if len(o.KeyResults) > 0 {
				fmt.Printf("\nKey Results (%d):\n", len(o.KeyResults))
				for i, kr := range o.KeyResults {
					check := "○"
					if kr.Progress() >= 100 {
						check = "●"
					}
					fmt.Printf("  %s %d. [%s] %s  (%.0f/%.0f %s = %.0f%%)\n",
						check, i+1, kr.ID, kr.Description,
						kr.CurrentValue, kr.TargetValue, kr.Unit, kr.Progress())
				}
			}

			if len(o.ReviewNotes) > 0 {
				fmt.Printf("\nReview Notes (%d):\n", len(o.ReviewNotes))
				for _, n := range o.ReviewNotes {
					fmt.Printf("  [%s] @%s: %s\n", n.Date, n.Author, n.Note)
				}
			}
			return nil
		},
	}
	return cmd
}

func newOKRUpdateCmd() *cobra.Command {
	var (
		objective   string
		description string
		owner       string
		quarter     string
		status      string
		scope       string
		scopeRef    string
		parentID    string
	)
	cmd := &cobra.Command{
		Use:   "update <okr-id>",
		Short: "Update an OKR",
		Args:  cobra.ExactArgs(1),
		Example: `  agencycli okr update okr-xxx --status at_risk
  agencycli okr update okr-xxx --scope project --scope-ref my-service
  agencycli okr update okr-xxx --parent okr-yyy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewOKRStore(root)
			err = s.UpdateOKR(args[0], func(o *entity.OKR) {
				if cmd.Flags().Changed("objective") {
					o.Objective = objective
				}
				if cmd.Flags().Changed("description") {
					o.Description = description
				}
				if cmd.Flags().Changed("owner") {
					o.Owner = owner
				}
				if cmd.Flags().Changed("quarter") {
					o.Quarter = quarter
				}
				if cmd.Flags().Changed("status") {
					o.Status = entity.OKRStatus(status)
				}
				if cmd.Flags().Changed("scope") {
					o.Scope = entity.OKRScope(scope)
				}
				if cmd.Flags().Changed("scope-ref") {
					o.ScopeRef = scopeRef
				}
				if cmd.Flags().Changed("parent") {
					o.ParentID = parentID
				}
			})
			if err != nil {
				return err
			}
			fmt.Printf("✓ OKR %s updated\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&objective, "objective", "", "new objective")
	cmd.Flags().StringVar(&description, "description", "", "new description")
	cmd.Flags().StringVar(&owner, "owner", "", "new owner")
	cmd.Flags().StringVar(&quarter, "quarter", "", "new quarter")
	cmd.Flags().StringVar(&status, "status", "", "new status (on_track|in_progress|at_risk|off_track|achieved)")
	cmd.Flags().StringVar(&scope, "scope", "", "new scope (agency|project|agent)")
	cmd.Flags().StringVar(&scopeRef, "scope-ref", "", "new scope reference")
	cmd.Flags().StringVar(&parentID, "parent", "", "parent OKR ID")
	return cmd
}

func newOKRDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <okr-id>",
		Short: "Delete an OKR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewOKRStore(root)
			if err := s.DeleteOKR(args[0]); err != nil {
				return err
			}
			fmt.Printf("✓ OKR %s deleted\n", args[0])
			return nil
		},
	}
	return cmd
}

// ── KR sub-commands ──────────────────────────────────────────────────────────

func newOKRKRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kr",
		Aliases: []string{"key-result"},
		Short:   "Manage Key Results within an OKR",
	}
	cmd.AddCommand(
		newOKRKRAddCmd(),
		newOKRKRUpdateCmd(),
	)
	return cmd
}

func newOKRKRAddCmd() *cobra.Command {
	var (
		okrID       string
		description string
		target      float64
		metric      string
		unit        string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a Key Result to an OKR",
		Example: `  agencycli okr kr add --okr okr-2026q2-abc123 \
    --description "Reduce p95 latency to <200ms" --target 200 --unit ms --metric number`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if okrID == "" || description == "" {
				return fmt.Errorf("--okr and --description are required")
			}
			s := store.NewOKRStore(root)
			kr := entity.KeyResult{
				Description: description,
				TargetValue: target,
				MetricType:  entity.MetricType(metric),
				Unit:        unit,
			}
			created, err := s.AddKR(okrID, kr)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Key Result %s added to %s\n", created.ID, okrID)
			return nil
		},
	}
	cmd.Flags().StringVar(&okrID, "okr", "", "OKR ID (required)")
	cmd.Flags().StringVar(&description, "description", "", "key result description (required)")
	cmd.Flags().Float64Var(&target, "target", 100, "target value")
	cmd.Flags().StringVar(&metric, "metric", "number", "metric type (number|percentage|boolean|currency)")
	cmd.Flags().StringVar(&unit, "unit", "", "unit label")
	return cmd
}

func newOKRKRUpdateCmd() *cobra.Command {
	var (
		okrID   string
		krID    string
		current float64
		target  float64
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a Key Result's progress",
		Example: `  agencycli okr kr update --okr okr-2026q2-abc123 --kr kr-def456 --current 150`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if okrID == "" || krID == "" {
				return fmt.Errorf("--okr and --kr are required")
			}
			s := store.NewOKRStore(root)
			err = s.UpdateKR(okrID, krID, func(kr *entity.KeyResult) {
				if cmd.Flags().Changed("current") {
					kr.CurrentValue = current
				}
				if cmd.Flags().Changed("target") {
					kr.TargetValue = target
				}
			})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Key Result %s updated\n", krID)
			return nil
		},
	}
	cmd.Flags().StringVar(&okrID, "okr", "", "OKR ID (required)")
	cmd.Flags().StringVar(&krID, "kr", "", "Key Result ID (required)")
	cmd.Flags().Float64Var(&current, "current", 0, "current value")
	cmd.Flags().Float64Var(&target, "target", 0, "target value")
	return cmd
}

// ── Review ───────────────────────────────────────────────────────────────────

func newOKRReviewCmd() *cobra.Command {
	var (
		okrID  string
		note   string
		author string
	)
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Add a review note to an OKR",
		Example: `  agencycli okr review --okr okr-2026q2-abc123 --note "Good progress on KR1" --author human`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if okrID == "" || note == "" {
				return fmt.Errorf("--okr and --note are required")
			}
			s := store.NewOKRStore(root)
			err = s.UpdateOKR(okrID, func(o *entity.OKR) {
				o.ReviewNotes = append(o.ReviewNotes, entity.ReviewNote{
					Date:   time.Now().UTC().Format("2006-01-02"),
					Note:   note,
					Author: author,
				})
			})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Review note added to %s\n", okrID)
			return nil
		},
	}
	cmd.Flags().StringVar(&okrID, "okr", "", "OKR ID (required)")
	cmd.Flags().StringVar(&note, "note", "", "review note text (required)")
	cmd.Flags().StringVar(&author, "author", "human", "author")
	return cmd
}

func statusIcon(status string) string {
	switch status {
	case "on_track":
		return "🟢"
	case "in_progress":
		return "🔵"
	case "at_risk":
		return "🟡"
	case "off_track":
		return "🔴"
	case "achieved":
		return "🔵"
	case "planned":
		return "⚪"
	case "completed":
		return "✅"
	case "cancelled":
		return "❌"
	default:
		return "○"
	}
}
