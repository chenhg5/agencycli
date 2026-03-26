package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/ctxbuild"
	"github.com/chenhg5/agencycli/internal/entity"
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
.agencycli/agent.yaml and regenerates any agent whose context is stale.

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
				diff, err := syncAgent(root, s, t.project, t.name, force)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  ✗ %s/%s: %v\n", t.project, t.name, err)
					continue
				}
				if len(diff.changed)+len(diff.added)+len(diff.removed) > 0 {
					fmt.Printf("  ✓ synced  %s/%s\n", t.project, t.name)
					if len(diff.changed) > 0 {
						fmt.Printf("      changed : %s\n", strings.Join(diff.changed, ", "))
					}
					if len(diff.added) > 0 {
						fmt.Printf("      added   : %s\n", strings.Join(diff.added, ", "))
					}
					if len(diff.removed) > 0 {
						fmt.Printf("      removed : %s\n", strings.Join(diff.removed, ", "))
					}
					synced++
				} else {
					fmt.Printf("  · skipped %s/%s  (up to date)\n", t.project, t.name)
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

// syncDiff holds the categorised layer changes detected during a sync.
type syncDiff struct {
	changed []string // layers whose content hash changed
	added   []string // layers/skills that did not exist before
	removed []string // layers/skills that no longer exist
}

// syncAgent rebuilds the context for one agent and writes it only if the
// content has changed (or force is true).
// Returns a syncDiff describing what changed (all-empty == up to date).
func syncAgent(root string, s store.Store, project, agentName string, force bool) (syncDiff, error) {
	meta, err := s.AgentMeta(project, agentName)
	if err != nil {
		return syncDiff{}, err
	}

	// Human agents have no context files to sync.
	if meta.Model == entity.ModelHuman {
		return syncDiff{}, nil
	}

	builder := ctxbuild.NewBuilder(s)
	mc, err := builder.Build(project, meta.Team, meta.Role)
	if err != nil {
		return syncDiff{}, fmt.Errorf("build context: %w", err)
	}

	newHashes := ctxbuild.LayerHashes(mc)

	// If meta.Playbook is empty (agent hired before this feature), fall back to
	// reading the project.yaml to find the playbook name, and persist it into meta.
	if meta.Playbook == "" {
		if cfg, cerr := s.ProjectConfig(project); cerr == nil && cfg != nil {
			for _, spec := range cfg.Agents {
				if spec.Name == agentName && spec.Playbook != "" {
					meta.Playbook = spec.Playbook
					break
				}
			}
		}
	}

	// Include playbook hash so changes to playbooks are detected.
	// meta.Playbook carries the relative path from the blueprint directory
	// (e.g. "playbooks/pm.md").
	var playbookData []byte
	if meta.Playbook != "" {
		playbookPath := filepath.Join(root, "project-blueprints", project, meta.Playbook)
		playbookData, _ = os.ReadFile(playbookPath)
		if len(playbookData) > 0 {
			newHashes["playbook:"+meta.Playbook] = ctxbuild.ContentHash(string(playbookData))
		}
	}

	diff := diffHashes(meta.ContextHash, newHashes)

	if !force && len(diff.changed)+len(diff.added)+len(diff.removed) == 0 {
		return syncDiff{}, nil
	}

	agentDir := s.AgentDir(project, agentName)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return syncDiff{}, err
	}

	// Copy playbook BEFORE running the formatter so CLAUDE.md/@import picks it up.
	if meta.Playbook != "" && len(playbookData) > 0 {
		ctxDir := filepath.Join(agentDir, ".agencycli", "context")
		if err := os.MkdirAll(ctxDir, 0o755); err != nil {
			return syncDiff{}, err
		}
		if err := os.WriteFile(filepath.Join(ctxDir, "wakeup.md"), playbookData, 0o644); err != nil {
			return syncDiff{}, fmt.Errorf("write wakeup.md: %w", err)
		}
	}

	// Re-generate context files (formatter will detect wakeup.md and @import it).
	f, err := formatter.New(meta.Model)
	if err != nil {
		return syncDiff{}, err
	}
	if err := f.Format(mc, agentDir); err != nil {
		return syncDiff{}, err
	}

	now := time.Now().UTC()
	meta.ContextHash = newHashes
	meta.SyncedAt = &now
	// HiredAt is intentionally left unchanged — sync ≠ rehire.
	if err := s.SaveAgentMeta(project, agentName, meta); err != nil {
		return syncDiff{}, err
	}

	return diff, nil
}

// diffHashes compares old and new hash maps and categorises each key.
func diffHashes(old, new map[string]string) syncDiff {
	var d syncDiff
	for k, newVal := range new {
		if oldVal, exists := old[k]; !exists {
			d.added = append(d.added, k)
		} else if oldVal != newVal {
			d.changed = append(d.changed, k)
		}
	}
	for k := range old {
		if _, exists := new[k]; !exists {
			d.removed = append(d.removed, k)
		}
	}
	sort.Strings(d.changed)
	sort.Strings(d.added)
	sort.Strings(d.removed)
	return d
}

