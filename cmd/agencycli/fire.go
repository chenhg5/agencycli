package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newFireCmd() *cobra.Command {
	var (
		project   string
		agentName string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "fire",
		Short: "Fire (remove) an agent from a project",
		Long: `fire removes an agent from a project.

By default this is a SOFT DELETE: the agent's working directory is moved to

  projects/<project>/agents/.fired/<name>-<timestamp>/

so that it can be inspected or restored later.

Pass --force to permanently delete the directory (irreversible).`,
		Example: `  # Soft-delete — recoverable
  agencycli fire --project my-api --agent dev

  # Hard delete — permanent, cannot be undone
  agencycli fire --project my-api --agent dev --force

  # From outside the workspace
  agencycli --dir /path/to/Agency fire --project my-api --agent dev`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			// Confirm the agent actually exists before doing anything.
			if _, err := s.AgentMeta(project, agentName); err != nil {
				return fmt.Errorf("fire: %w", err)
			}

			agentDir := s.AgentDir(project, agentName)

			if force {
				if err := os.RemoveAll(agentDir); err != nil {
					return fmt.Errorf("fire: remove agent dir: %w", err)
				}
				fmt.Printf("✓ Agent %q/%q permanently deleted.\n", project, agentName)
				return nil
			}

			// Soft delete: move to .fired/<name>-<timestamp>/
			timestamp := time.Now().UTC().Format("20060102-150405")
			firedDirName := agentName + "-" + timestamp
			firedDir := s.FiredAgentDir(project, firedDirName)

			if err := os.MkdirAll(filepath.Dir(firedDir), 0o755); err != nil {
				return fmt.Errorf("fire: create .fired directory: %w", err)
			}
			if err := os.Rename(agentDir, firedDir); err != nil {
				return fmt.Errorf("fire: archive agent directory: %w", err)
			}

			fmt.Printf("✓ Agent %q/%q soft-deleted.\n", project, agentName)
			fmt.Printf("  Archived at: %s\n", firedDir)
			fmt.Printf("\n  To restore manually: mv %s %s\n", firedDir, agentDir)
			fmt.Printf("  To permanently delete:\n")
			fmt.Printf("    rm -rf %s\n", firedDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "Name of the agent to fire")
	cmd.Flags().BoolVar(&force, "force", false, "Permanently delete the agent directory (irreversible)")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("agent")

	return cmd
}
