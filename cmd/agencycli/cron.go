package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

func randomID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func newCronCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Manage scheduled cron jobs for agents",
		Long: `Cron jobs fire at calendar-scheduled times and enqueue a Task for the
agent's heartbeat loop to pick up and execute.

The schedule uses standard 5-field crontab syntax:
  ┌──────── minute  (0-59)
  │ ┌────── hour    (0-23)
  │ │ ┌──── day of month (1-31)
  │ │ │ ┌── month   (1-12 or JAN-DEC)
  │ │ │ │ ┌ day of week (0-6, Sun=0, or SUN-SAT)
  │ │ │ │ │
  * * * * *

Examples:
  "0 9 * * 1-5"   — 09:00 on weekdays
  "*/30 * * * *"  — every 30 minutes
  "0 0 * * *"     — midnight every day`,
	}
	cmd.AddCommand(
		newCronAddCmd(),
		newCronListCmd(),
		newCronDeleteCmd(),
		newCronEnableCmd(),
		newCronDisableCmd(),
	)
	return cmd
}

// ── cron add ──────────────────────────────────────────────────────────────────

func newCronAddCmd() *cobra.Command {
	var (
		project   string
		agentName string
		title     string
		schedule  string
		prompt    string
		disabled  bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a cron job for an agent",
		Example: `  # Run a daily standup summary at 09:00 on weekdays
  agencycli cron add --project cc-connect --agent pm \
    --title "Daily standup" \
    --schedule "0 9 * * 1-5" \
    --prompt "Summarise open tasks and post a standup report."

  # Every 30 minutes health check
  agencycli cron add --project cc-connect --agent qa-reviewer \
    --title "Periodic health check" \
    --schedule "*/30 * * * *" \
    --prompt "Run a quick sanity check on the project."`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}
			if title == "" || schedule == "" || prompt == "" {
				return fmt.Errorf("--title, --schedule, and --prompt are required")
			}
			if err := validateCronSchedule(schedule); err != nil {
				return err
			}

			ts := taskstore.New(root)
			crons, err := ts.ListCrons(project, agentName)
			if err != nil {
				return err
			}

			id := fmt.Sprintf("c-%s-%s", time.Now().UTC().Format("20060102"), randomID(6))
			c := &entity.Cron{
				ID:       id,
				Title:    title,
				Schedule: schedule,
				Enabled:  !disabled,
				Prompt:   prompt,
			}
			crons = append(crons, c)
			if err := ts.SaveCrons(project, agentName, crons); err != nil {
				return err
			}

			status := "enabled"
			if !c.Enabled {
				status = "disabled"
			}
			fmt.Printf("✓ Cron %s created  [%s / %s]\n", id, project, agentName)
			fmt.Printf("  Title    : %s\n", title)
			fmt.Printf("  Schedule : %s\n", schedule)
			fmt.Printf("  Status   : %s\n", status)
			if next := nextCronTime(schedule); !next.IsZero() {
				fmt.Printf("  Next run : %s\n", next.Local().Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	cmd.Flags().StringVar(&title, "title", "", "short description of what this job does")
	cmd.Flags().StringVar(&schedule, "schedule", "", "crontab expression (5 fields)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "prompt text that will be enqueued as a task")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "create the cron in disabled state")
	return cmd
}

// ── cron list ─────────────────────────────────────────────────────────────────

func newCronListCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cron jobs for an agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}
			ts := taskstore.New(root)
			crons, err := ts.ListCrons(project, agentName)
			if err != nil {
				return err
			}
			if len(crons) == 0 {
				fmt.Println("No cron jobs configured.")
				return nil
			}

			fmt.Printf("%-22s  %-6s  %-16s  %-20s  %s\n", "ID", "STATUS", "SCHEDULE", "NEXT RUN", "TITLE")
			fmt.Println(strings.Repeat("─", 90))
			for _, c := range crons {
				status := "enabled"
				if !c.Enabled {
					status = "disabled"
				}
				nextStr := "-"
				if c.Enabled {
					if next := nextCronTime(c.Schedule); !next.IsZero() {
						nextStr = next.Local().Format("01-02 15:04")
					}
				}
				fmt.Printf("%-22s  %-6s  %-16s  %-20s  %s\n",
					c.ID, status, c.Schedule, nextStr, c.Title)
				if c.LastRun != nil {
					fmt.Printf("  last run: %s (%s)\n",
						c.LastRun.Local().Format("2006-01-02 15:04"), c.LastRunStatus)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	return cmd
}

// ── cron delete ───────────────────────────────────────────────────────────────

func newCronDeleteCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)

	cmd := &cobra.Command{
		Use:   "delete <cron-id>",
		Short: "Delete a cron job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}
			id := args[0]
			ts := taskstore.New(root)
			crons, err := ts.ListCrons(project, agentName)
			if err != nil {
				return err
			}
			filtered := crons[:0]
			found := false
			for _, c := range crons {
				if c.ID == id {
					found = true
					continue
				}
				filtered = append(filtered, c)
			}
			if !found {
				return fmt.Errorf("cron %q not found", id)
			}
			if err := ts.SaveCrons(project, agentName, filtered); err != nil {
				return err
			}
			fmt.Printf("✓ Cron %s deleted\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	return cmd
}

// ── cron enable / disable ─────────────────────────────────────────────────────

func newCronEnableCmd() *cobra.Command  { return newCronToggleCmd(true) }
func newCronDisableCmd() *cobra.Command { return newCronToggleCmd(false) }

func newCronToggleCmd(enable bool) *cobra.Command {
	var (
		project   string
		agentName string
	)

	action := "enable"
	if !enable {
		action = "disable"
	}

	cmd := &cobra.Command{
		Use:   action + " <cron-id>",
		Short: strings.Title(action) + " a cron job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}
			id := args[0]
			ts := taskstore.New(root)
			crons, err := ts.ListCrons(project, agentName)
			if err != nil {
				return err
			}
			found := false
			for _, c := range crons {
				if c.ID == id {
					c.Enabled = enable
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("cron %q not found", id)
			}
			if err := ts.SaveCrons(project, agentName, crons); err != nil {
				return err
			}
			status := "enabled"
			if !enable {
				status = "disabled"
			}
			fmt.Printf("✓ Cron %s %s\n", id, status)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	return cmd
}

// ── helpers ───────────────────────────────────────────────────────────────────

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// validateCronSchedule parses and validates a 5-field cron expression.
func validateCronSchedule(expr string) error {
	if _, err := cronParser.Parse(expr); err != nil {
		return fmt.Errorf("invalid cron schedule %q: %w", expr, err)
	}
	return nil
}

// nextCronTime returns the next scheduled time for a cron expression.
func nextCronTime(expr string) time.Time {
	sched, err := cronParser.Parse(expr)
	if err != nil {
		return time.Time{}
	}
	return sched.Next(time.Now())
}
