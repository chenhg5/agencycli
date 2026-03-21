package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
)

func newClearCmd() *cobra.Command {
	var (
		project   string
		taskOnly  bool
		inboxOnly bool
		yes       bool
	)

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear tasks and/or inbox items (with confirmation)",
		Long: `clear removes task queues and inbox items from the workspace.

By default it clears both tasks (active + archived) and inbox for all agents.
Use --project to limit to one project, --tasks/--inbox for partial clearing.`,
		Example: `  agencycli clear                        # clear everything (with prompt)
  agencycli clear --project my-api       # clear tasks+inbox for one project
  agencycli clear --tasks                # clear only task queues
  agencycli clear --inbox                # clear only inbox
  agencycli clear --yes                  # skip confirmation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ts := taskstore.New(root)

			clearTasks := !inboxOnly
			clearInbox := !taskOnly

			// Collect targets
			type agentTarget struct{ project, name string }
			var targets []agentTarget

			if project != "" {
				agents, err := ts.ListAgents(project)
				if err != nil {
					return err
				}
				for _, a := range agents {
					targets = append(targets, agentTarget{project, a})
				}
			} else {
				projects, err := ts.ListProjects()
				if err != nil {
					return err
				}
				for _, p := range projects {
					agents, err := ts.ListAgents(p)
					if err != nil {
						return err
					}
					for _, a := range agents {
						targets = append(targets, agentTarget{p, a})
					}
				}
			}

			// Preview counts
			taskCount := 0
			if clearTasks {
				for _, t := range targets {
					active, _ := ts.ListTasks(t.project, t.name)
					archived, _ := ts.ListArchivedTasks(t.project, t.name)
					taskCount += len(active) + len(archived)
				}
			}
			inboxCount := 0
			if clearInbox && (project == "" || !taskOnly) {
				items, _ := ts.ListInbox()
				inboxCount = len(items)
			}
			// human messages are workspace-level; count them when not scoped to a project
			var humanMsgCount int
			if clearInbox && project == "" {
				msgs, _ := ts.ListUnreadMessages("human")
				humanMsgCount = len(msgs)
			}

			if taskCount == 0 && inboxCount == 0 && humanMsgCount == 0 {
				fmt.Println("Nothing to clear.")
				return nil
			}

			// Print summary
			fmt.Println("The following will be permanently deleted:")
			if clearTasks && taskCount > 0 {
				fmt.Printf("  • %d task(s) across %d agent(s)\n", taskCount, len(targets))
			}
			if clearInbox && inboxCount > 0 {
				fmt.Printf("  • %d inbox item(s)\n", inboxCount)
			}
			if clearInbox && humanMsgCount > 0 {
				fmt.Printf("  • %d human message(s)\n", humanMsgCount)
			}
			fmt.Println()

			// Confirm
			if !yes {
				fmt.Print("Type \"yes\" to confirm: ")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(scanner.Text())
				if answer != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			// Execute
			cleared := 0
			if clearTasks {
				for _, t := range targets {
					if err := ts.ClearTasks(t.project, t.name); err != nil {
						fmt.Fprintf(os.Stderr, "  ✗ clear tasks %s/%s: %v\n", t.project, t.name, err)
					} else {
						cleared++
					}
				}
				fmt.Printf("  ✓ cleared task queues for %d agent(s)\n", cleared)
			}
			if clearInbox {
				if err := ts.ClearInbox(); err != nil {
					fmt.Fprintf(os.Stderr, "  ✗ clear inbox: %v\n", err)
				} else {
					fmt.Println("  ✓ inbox cleared")
				}
				if project == "" {
					if err := ts.ClearMessages("human"); err != nil {
						fmt.Fprintf(os.Stderr, "  ✗ clear human messages: %v\n", err)
					} else {
						fmt.Println("  ✓ human messages cleared")
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Limit to a specific project")
	cmd.Flags().BoolVar(&taskOnly, "tasks", false, "Clear task queues only (skip inbox)")
	cmd.Flags().BoolVar(&inboxOnly, "inbox", false, "Clear inbox only (skip tasks)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}
