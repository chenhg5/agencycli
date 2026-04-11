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

func newMilestoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "milestone",
		Aliases: []string{"ms"},
		Short:   "Manage project milestones",
	}
	cmd.AddCommand(
		newMilestoneListCmd(),
		newMilestoneCreateCmd(),
		newMilestoneShowCmd(),
		newMilestoneUpdateCmd(),
		newMilestoneDeleteCmd(),
	)
	return cmd
}

func newMilestoneListCmd() *cobra.Command {
	var (
		project string
		status  string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List milestones for a project",
		Example: `  agencycli milestone list --project my-service
  agencycli milestone list --project my-service --status in_progress`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			s := store.NewMilestoneStore(root)
			milestones, err := s.List(project)
			if err != nil {
				return err
			}

			if status != "" {
				var filtered []entity.Milestone
				for _, ms := range milestones {
					if string(ms.Status) == status {
						filtered = append(filtered, ms)
					}
				}
				milestones = filtered
			}

			if len(milestones) == 0 {
				fmt.Println("No milestones found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "STATUS\tPROGRESS\tID\tDUE\tTITLE")
			fmt.Fprintln(w, "──────\t────────\t──\t───\t─────")
			for _, ms := range milestones {
				icon := statusIcon(string(ms.Status))
				due := "-"
				if ms.DueDate != nil {
					due = ms.DueDate.Format("2006-01-02")
				}
				fmt.Fprintf(w, "%s %s\t%d%%\t%s\t%s\t%s\n",
					icon, ms.Status, ms.Progress, ms.ID, due, ms.Title)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name (required)")
	cmd.Flags().StringVar(&status, "status", "", "filter by status (planned|in_progress|completed|cancelled)")
	return cmd
}

func newMilestoneCreateCmd() *cobra.Command {
	var (
		project     string
		title       string
		description string
		owner       string
		dueDate     string
		criteria    []string
		labels      []string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new milestone",
		Example: `  agencycli milestone create --project my-service --title "Beta Release" \
    --due-date 2026-05-01 --owner human \
    --criteria "All P0 bugs fixed" --criteria "Docs updated"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || title == "" {
				return fmt.Errorf("--project and --title are required")
			}
			s := store.NewMilestoneStore(root)
			ms := entity.Milestone{
				Title:       title,
				Description: description,
				Owner:       owner,
				Criteria:    criteria,
				TaskLabels:  labels,
			}
			if dueDate != "" {
				dd, err := time.Parse("2006-01-02", dueDate)
				if err != nil {
					return fmt.Errorf("invalid --due-date format, use YYYY-MM-DD")
				}
				ms.DueDate = &dd
			}
			created, err := s.Create(project, ms)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Milestone %s created: %s\n", created.ID, created.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name (required)")
	cmd.Flags().StringVar(&title, "title", "", "milestone title (required)")
	cmd.Flags().StringVar(&description, "description", "", "description")
	cmd.Flags().StringVar(&owner, "owner", "", "owner (human or project/agent)")
	cmd.Flags().StringVar(&dueDate, "due-date", "", "due date (YYYY-MM-DD)")
	cmd.Flags().StringArrayVar(&criteria, "criteria", nil, "acceptance criteria (repeatable)")
	cmd.Flags().StringArrayVar(&labels, "label", nil, "task labels (repeatable)")
	return cmd
}

func newMilestoneShowCmd() *cobra.Command {
	var project string
	cmd := &cobra.Command{
		Use:   "show <milestone-id>",
		Short: "Show milestone details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			s := store.NewMilestoneStore(root)
			ms, err := s.Get(project, args[0])
			if err != nil {
				return err
			}

			fmt.Printf("ID          : %s\n", ms.ID)
			fmt.Printf("Title       : %s\n", ms.Title)
			fmt.Printf("Status      : %s %s\n", statusIcon(string(ms.Status)), ms.Status)
			fmt.Printf("Progress    : %d%%\n", ms.Progress)
			if ms.Owner != "" {
				fmt.Printf("Owner       : %s\n", ms.Owner)
			}
			if ms.DueDate != nil {
				fmt.Printf("Due Date    : %s\n", ms.DueDate.Format("2006-01-02"))
			}
			if ms.Description != "" {
				fmt.Printf("Description : %s\n", ms.Description)
			}
			fmt.Printf("Created     : %s\n", ms.CreatedAt.Format("2006-01-02 15:04"))

			if len(ms.Criteria) > 0 {
				fmt.Printf("\nAcceptance Criteria (%d):\n", len(ms.Criteria))
				for i, c := range ms.Criteria {
					fmt.Printf("  %d. %s\n", i+1, c)
				}
			}
			if len(ms.TaskLabels) > 0 {
				fmt.Printf("\nTask Labels: %s\n", fmt.Sprintf("%v", ms.TaskLabels))
			}
			if len(ms.LinkedKR) > 0 {
				fmt.Printf("Linked KRs : %s\n", fmt.Sprintf("%v", ms.LinkedKR))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name (required)")
	return cmd
}

func newMilestoneUpdateCmd() *cobra.Command {
	var (
		project  string
		title    string
		status   string
		progress int
		owner    string
		dueDate  string
	)
	cmd := &cobra.Command{
		Use:   "update <milestone-id>",
		Short: "Update a milestone",
		Args:  cobra.ExactArgs(1),
		Example: `  agencycli milestone update ms-abcd1234 --project my-service --status in_progress
  agencycli milestone update ms-abcd1234 --project my-service --progress 75`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			s := store.NewMilestoneStore(root)
			err = s.Update(project, args[0], func(ms *entity.Milestone) {
				if cmd.Flags().Changed("title") {
					ms.Title = title
				}
				if cmd.Flags().Changed("status") {
					ms.Status = entity.MilestoneStatus(status)
				}
				if cmd.Flags().Changed("progress") {
					ms.Progress = progress
				}
				if cmd.Flags().Changed("owner") {
					ms.Owner = owner
				}
				if cmd.Flags().Changed("due-date") {
					if dd, err := time.Parse("2006-01-02", dueDate); err == nil {
						ms.DueDate = &dd
					}
				}
			})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Milestone %s updated\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name (required)")
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&status, "status", "", "new status (planned|in_progress|completed|cancelled)")
	cmd.Flags().IntVar(&progress, "progress", 0, "progress percentage (0-100)")
	cmd.Flags().StringVar(&owner, "owner", "", "new owner")
	cmd.Flags().StringVar(&dueDate, "due-date", "", "new due date (YYYY-MM-DD)")
	return cmd
}

func newMilestoneDeleteCmd() *cobra.Command {
	var project string
	cmd := &cobra.Command{
		Use:   "delete <milestone-id>",
		Short: "Delete a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			s := store.NewMilestoneStore(root)
			if err := s.Delete(project, args[0]); err != nil {
				return err
			}
			fmt.Printf("✓ Milestone %s deleted\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name (required)")
	return cmd
}
