package main

import (
	"fmt"
	"os"
	"time"

	"github.com/chenhg5/agencycli/internal/ctxbuild"
	"github.com/chenhg5/agencycli/internal/formatter"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var (
		project   string
		agentName string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Regenerate agent working directories whose context has changed",
		Long: `sync compares the current prompt file contents against the hashes stored in
.agencycli-agent.yaml and regenerates any agent whose context is stale.

With no flags it syncs all agents in all projects.
Use --project to limit to one project, --project + --name for a single agent.`,
		Example: `  agencycli sync                          # sync everything
  agencycli sync --project my-api         # sync all agents in my-api
  agencycli sync --project my-api --name dev  # sync one specific agent`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			// Collect the list of (project, agentName) pairs to sync
			type target struct{ project, name string }
			var targets []target

			if project != "" && agentName != "" {
				targets = append(targets, target{project, agentName})
			} else if project != "" {
				agents, err := s.ListAgents(project)
				if err != nil {
					return err
				}
				for _, a := range agents {
					targets = append(targets, target{project, a.Name})
				}
			} else {
				projects, err := s.ListProjects()
				if err != nil {
					return err
				}
				for _, p := range projects {
					agents, err := s.ListAgents(p.Name)
					if err != nil {
						return err
					}
					for _, a := range agents {
						targets = append(targets, target{p.Name, a.Name})
					}
				}
			}

			if len(targets) == 0 {
				fmt.Println("No agents found.")
				return nil
			}

			synced, skipped := 0, 0
			for _, t := range targets {
				changed, err := syncAgent(s, t.project, t.name, force)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  ✗ %s/%s: %v\n", t.project, t.name, err)
					continue
				}
				if changed {
					fmt.Printf("  ✓ synced  %s/%s\n", t.project, t.name)
					synced++
				} else {
					fmt.Printf("  - skipped %s/%s (up to date)\n", t.project, t.name)
					skipped++
				}
			}
			fmt.Printf("\n%d synced, %d already up to date\n", synced, skipped)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Limit sync to this project")
	cmd.Flags().StringVar(&agentName, "name", "", "Limit sync to this agent (requires --project)")
	cmd.Flags().BoolVar(&force, "force", false, "Regenerate even if context hasn't changed")
	return cmd
}

// syncAgent rebuilds the context for one agent and writes it only if the
// content has changed (or force is true). Returns true if a write occurred.
func syncAgent(s store.Store, project, agentName string, force bool) (bool, error) {
	meta, err := s.AgentMeta(project, agentName)
	if err != nil {
		return false, err
	}

	builder := ctxbuild.NewBuilder(s)
	mc, err := builder.Build(project, meta.Team)
	if err != nil {
		return false, fmt.Errorf("build context: %w", err)
	}

	newHashes := ctxbuild.LayerHashes(mc)

	if !force && hashesEqual(meta.ContextHash, newHashes) {
		return false, nil
	}

	agentDir := s.AgentDir(project, agentName)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return false, err
	}

	f, err := formatter.New(meta.Model)
	if err != nil {
		return false, err
	}
	if err := f.Format(mc, agentDir); err != nil {
		return false, err
	}

	meta.ContextHash = newHashes
	meta.HiredAt = time.Now().UTC() // update timestamp on sync
	if err := s.SaveAgentMeta(project, agentName, meta); err != nil {
		return false, err
	}

	return true, nil
}

// hashesEqual reports whether two hash maps have identical contents.
func hashesEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
