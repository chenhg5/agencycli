package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/chenhg5/agencycli/internal/ctxbuild"
	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/formatter"
	"github.com/chenhg5/agencycli/internal/sandbox"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage project configuration and lifecycle",
		Long: `project commands manage the declarative project.yaml that defines which agents
belong to a project, how they wake up, and which workflows are active.

Typical flow after creating an agency from a template:

  1. Create a project from a blueprint:
       agencycli create project --name my-service --blueprint default

  2. Review / edit the generated project.yaml:
       agencycli project show --project my-service

  3. Apply the configuration (hire agents + configure heartbeats/crons):
       agencycli project apply --project my-service

  4. Start a workflow:
       agencycli workflow run feature-dev --project my-service \
         --input feature="User login"

  5. Start the scheduler to process tasks automatically:
       agencycli scheduler start`,
	}
	cmd.AddCommand(
		newProjectShowCmd(),
		newProjectApplyCmd(),
		newProjectBlueprintsCmd(),
	)
	return cmd
}

// ── project show ──────────────────────────────────────────────────────────────

func newProjectShowCmd() *cobra.Command {
	var (
		project string
		asYAML  bool
	)
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show project.yaml configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			ts := taskstore.New(root)
			cfg, err := ts.GetProjectConfig(project)
			if err != nil {
				return err
			}
			if cfg == nil {
				fmt.Printf("No project.yaml found for project %q.\n", project)
				fmt.Printf("Create one with: agencycli create project --name %s --blueprint <name>\n", project)
				return nil
			}
			if asYAML {
				data, _ := yaml.Marshal(cfg)
				fmt.Print(string(data))
				return nil
			}
			printProjectConfig(cfg, project, root)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().BoolVar(&asYAML, "yaml", false, "print raw YAML")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// ── project apply ─────────────────────────────────────────────────────────────

func newProjectApplyCmd() *cobra.Command {
	var (
		project string
		dryRun  bool
		force   bool
	)
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply project.yaml: hire agents, configure heartbeats and crons",
		Long: `apply reads projects/<project>/project.yaml and brings the live state into sync:
  • Hires any agent that does not yet have a working directory (or --force re-hires all)
  • Writes heartbeat.yaml for agents that define a heartbeat schedule
  • Writes crons.yaml for agents that define cron jobs

After apply, start the scheduler to let agents work autonomously:
  agencycli scheduler start`,
		Example: `  agencycli project apply --project my-service
  agencycli project apply --project my-service --dry-run
  agencycli project apply --project my-service --force   # re-hire all agents`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}

			ts := taskstore.New(root)
			s := store.NewFS(root)

			cfg, err := ts.GetProjectConfig(project)
			if err != nil {
				return err
			}
			if cfg == nil {
				return fmt.Errorf("no project.yaml found for project %q — run `agencycli create project` first", project)
			}

			if dryRun {
				fmt.Printf("[dry-run] project apply --project %s\n\n", project)
			}

			for _, spec := range cfg.Agents {
				if err := applyAgentSpec(root, project, spec, s, ts, force, dryRun); err != nil {
					fmt.Fprintf(os.Stderr, "  ✗ agent %s: %v\n", spec.Name, err)
				}
			}

			fmt.Println()
			fmt.Printf("✓ project apply done for %q\n", project)
			if !dryRun {
				fmt.Printf("\nStart the scheduler to let agents run autonomously:\n")
				fmt.Printf("  agencycli scheduler start\n")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be done without making changes")
	cmd.Flags().BoolVar(&force, "force", false, "re-hire agents even if they already exist")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// ── project blueprints ────────────────────────────────────────────────────────

func newProjectBlueprintsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blueprints",
		Short: "List available project blueprints in this agency",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ts := taskstore.New(root)
			names, err := ts.ListProjectBlueprints()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Printf("No project blueprints found.\n")
				fmt.Printf("Add blueprints at: %s/project-blueprints/<name>.yaml\n", root)
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "BLUEPRINT\tAGENTS")
			fmt.Fprintln(w, "─────────\t──────")
			for _, name := range names {
				bp, _ := ts.GetProjectBlueprint(name)
				if bp == nil {
					continue
				}
				fmt.Fprintf(w, "%s\t%d\n", name, len(bp.Agents))
			}
			w.Flush()
			return nil
		},
	}
	return cmd
}

// ── core logic ────────────────────────────────────────────────────────────────

func applyAgentSpec(root, project string, spec entity.AgentSpec,
	s store.Store, ts taskstore.Store, force, dryRun bool) error {

	agentDir := filepath.Join(root, "projects", project, "agents", spec.Name)

	alreadyExists := false
	if _, err := os.Stat(filepath.Join(agentDir, "agent.yaml")); err == nil {
		alreadyExists = true
	}

	// ── Hire ──────────────────────────────────────────────────────────────────

	if !alreadyExists || force {
		action := "hire"
		if alreadyExists {
			action = "re-hire"
		}
		if dryRun {
			fmt.Printf("  [dry-run] would %s agent %s (model=%s role=%s team=%s)\n",
				action, spec.Name, spec.Model, spec.Role, spec.Team)
		} else {
			if err := hireAgentFromSpec(root, project, spec, s, force); err != nil {
				return fmt.Errorf("hire: %w", err)
			}
			fmt.Printf("  ✓ %s agent %s (model=%s)\n", action, spec.Name, spec.Model)
		}
	} else {
		fmt.Printf("  · agent %s already exists, skipping hire (use --force to re-hire)\n", spec.Name)
	}

	// ── Playbook → wakeup.md ──────────────────────────────────────────────────

	if spec.Playbook != "" {
		playbookSrc := filepath.Join(root, "agent-playbooks", spec.Playbook)
		wakeupDst := filepath.Join(agentDir, ".agencycli-context", "wakeup.md")
		if dryRun {
			fmt.Printf("    [dry-run] would install playbook %s → .agencycli-context/wakeup.md\n", spec.Playbook)
		} else {
			data, err := os.ReadFile(playbookSrc)
			if err != nil {
				fmt.Printf("    ⚠ playbook %s not found: %v (skipping)\n", spec.Playbook, err)
			} else {
				if err := os.MkdirAll(filepath.Dir(wakeupDst), 0o755); err != nil {
					return fmt.Errorf("create .agencycli-context: %w", err)
				}
				if err := os.WriteFile(wakeupDst, data, 0644); err != nil {
					return fmt.Errorf("write wakeup.md: %w", err)
				}
				fmt.Printf("    ✓ playbook installed: %s → .agencycli-context/wakeup.md\n", spec.Playbook)
			}
		}
	}

	// ── Heartbeat ─────────────────────────────────────────────────────────────

	if spec.Heartbeat != nil {
		hb := spec.Heartbeat
		// If a playbook is configured, auto-set wakeup prompt.
		if spec.Playbook != "" && hb.WakeupPrompt == "" {
			hb.WakeupPrompt = "@.agencycli-context/wakeup.md"
		}
		if dryRun {
			interval := "not set"
			if hb.Interval != "" {
				interval = hb.Interval
			}
			fmt.Printf("    [dry-run] would configure heartbeat: enabled=%v interval=%s active_hours=%s active_days=%s\n",
				hb.Enabled, interval, hb.ActiveHours, hb.ActiveDays)
		} else {
			if err := ts.SaveHeartbeat(project, spec.Name, hb); err != nil {
				return fmt.Errorf("save heartbeat: %w", err)
			}
			status := "disabled"
			if hb.Enabled {
				status = fmt.Sprintf("every %s", hb.Interval)
				if hb.ActiveHours != "" {
					status += fmt.Sprintf(" (%s)", hb.ActiveHours)
				}
				if hb.ActiveDays != "" {
					status += fmt.Sprintf(" [%s]", hb.ActiveDays)
				}
			}
			if hb.WakeupPrompt != "" {
				status += " +wakeup"
			}
			fmt.Printf("    ✓ heartbeat: %s\n", status)
		}
	}

	// ── Crons ─────────────────────────────────────────────────────────────────

	if len(spec.Crons) > 0 {
		var crons []*entity.Cron
		for _, cs := range spec.Crons {
			crons = append(crons, &entity.Cron{
				ID:       cs.ID,
				Schedule: cs.Schedule,
				Title:    cs.Title,
				Prompt:   cs.Prompt,
				Enabled:  true,
			})
		}
		if dryRun {
			fmt.Printf("    [dry-run] would configure %d cron(s)\n", len(crons))
		} else {
			if err := ts.SaveCrons(project, spec.Name, crons); err != nil {
				return fmt.Errorf("save crons: %w", err)
			}
			fmt.Printf("    ✓ %d cron(s) configured\n", len(crons))
		}
	}

	return nil
}

// hireAgentFromSpec performs the same work as the `hire` command but driven
// from an AgentSpec struct (used by `project apply`).
func hireAgentFromSpec(root, project string, spec entity.AgentSpec,
	s store.Store, force bool) error {

	agentModel := entity.AgentModel(spec.Model)

	builder := ctxbuild.NewBuilder(s)
	mc, err := builder.Build(project, spec.Team, spec.Role)
	if err != nil {
		return fmt.Errorf("build context: %w", err)
	}

	agentDir := filepath.Join(root, "projects", project, "agents", spec.Name)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	f, err := formatter.New(agentModel)
	if err != nil {
		return fmt.Errorf("formatter: %w", err)
	}
	if err := f.Format(mc, agentDir); err != nil {
		return fmt.Errorf("write context: %w", err)
	}

	// Resolve repo add-dirs from project metadata.
	var addDirs []string
	if projMeta, err2 := s.Project(project); err2 == nil && projMeta != nil && projMeta.Repo != "" {
		repoAbs := projMeta.Repo
		if !filepath.IsAbs(repoAbs) {
			repoAbs = filepath.Join(root, repoAbs)
		}
		addDirs = []string{repoAbs}
	}
	// Add agent-spec repos on top.
	for _, r := range spec.Repos {
		if !filepath.IsAbs(r) {
			r = filepath.Join(root, r)
		}
		addDirs = append(addDirs, r)
	}

	// Role workspace setup.
	if spec.Role != "" {
		roleMeta, err2 := s.Role(spec.Team, spec.Role)
		if err2 == nil {
			if err3 := applyRoleSetup(roleMeta.Setup, agentDir); err3 != nil {
				return fmt.Errorf("role setup: %w", err3)
			}
		}
	}

	// Build sandbox config if requested.
	var sandboxCfg *entity.SandboxConfig
	if spec.Sandbox {
		if err := sandbox.CheckDocker(); err != nil {
			return fmt.Errorf("sandbox requires Docker: %w", err)
		}
		img := sandbox.ImageForModel(agentModel)
		sandboxCfg = &entity.SandboxConfig{
			Provider: entity.SandboxDocker,
			Docker: &entity.DockerSandboxConfig{
				Image:       img,
				NetworkMode: "bridge",
			},
		}
	}

	meta := &entity.AgentMeta{
		Name:        spec.Name,
		Project:     project,
		Team:        spec.Team,
		Role:        spec.Role,
		Model:       agentModel,
		HiredAt:     time.Now().UTC(),
		ContextHash: ctxbuild.LayerHashes(mc),
		Sandbox:     sandboxCfg,
		AddDirs:     addDirs,
		Playbook:    spec.Playbook,
	}
	return s.SaveAgentMeta(project, spec.Name, meta)
}

// ── display helpers ───────────────────────────────────────────────────────────

func printProjectConfig(cfg *entity.ProjectConfig, project, root string) {
	fmt.Printf("Project: %s\n", cfg.Name)
	if cfg.Description != "" {
		fmt.Printf("  %s\n", cfg.Description)
	}
	fmt.Println()

	if len(cfg.Agents) > 0 {
		fmt.Printf("Agents (%d):\n", len(cfg.Agents))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  NAME\tMODEL\tROLE\tSANDBOX\tHEARTBEAT")
		fmt.Fprintln(w, "  ────\t─────\t────\t───────\t─────────")
		for _, a := range cfg.Agents {
			sbx := "no"
			if a.Sandbox {
				sbx = "docker"
			}
			hb := "(manual only)"
			if a.Heartbeat != nil && a.Heartbeat.Enabled {
				hb = a.Heartbeat.Interval
				if a.Heartbeat.ActiveHours != "" {
					hb += " " + a.Heartbeat.ActiveHours
				}
				if a.Heartbeat.ActiveDays != "" {
					hb += " [" + a.Heartbeat.ActiveDays + "]"
				}
			}
			role := a.Role
			if role == "" {
				role = "-"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n", a.Name, a.Model, role, sbx, hb)
		}
	w.Flush()
	fmt.Println()
}

fmt.Printf("Apply with:\n")
	fmt.Printf("  agencycli project apply --project %s\n", project)
}
