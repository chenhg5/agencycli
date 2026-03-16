package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agencycli/agencycli/internal/ctxbuild"
	"github.com/agencycli/agencycli/internal/entity"
	"github.com/agencycli/agencycli/internal/formatter"
	"github.com/agencycli/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newHireCmd() *cobra.Command {
	var (
		project     string
		dept        string
		model       string
		agentName   string
		extraPrompt string
		force       bool
	)

	cmd := &cobra.Command{
		Use:   "hire",
		Short: "Hire an agent for a project (merges context and creates the agent working directory)",
		Long: `hire assembles the full context for a (project, department) pair and writes
it into an agent working directory under projects/<project>/agents/<name>/.

The context layers are merged in this order:
  1. Company
  2. Department chain (from top-level to the specified department)
  3. Project

The output format depends on --model:
  claudecode   →  CLAUDE.md + .aios-context/ + .claude/skills/
  codex        →  AGENTS.md (single merged file)
  generic-cli  →  context.md (plain text)`,
		Example: `  agencycli hire --project "my-api" --dept "engineering/backend" \
               --model "claudecode" --name "dev"

  # Then start working:
  cd projects/my-api/agents/dev
  claude`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" || dept == "" || model == "" || agentName == "" {
				return fmt.Errorf("--project, --dept, --model and --name are all required")
			}

			agentModel := entity.NormaliseModel(entity.AgentModel(model))
			if !entity.IsValidModel(agentModel) {
				return fmt.Errorf("unknown model %q (supported: %s)",
					model, joinModels(entity.KnownModels))
			}

			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			// Verify project exists
			if _, err := s.Project(project); err != nil {
				return err
			}

			// Check the agent directory doesn't already exist unless --force
			agentDir := s.AgentDir(project, agentName)
			if _, err := os.Stat(agentDir); err == nil && !force {
				return fmt.Errorf(
					"agent %q already exists at %s\n"+
						"Use --force to regenerate it",
					agentName, agentDir,
				)
			}

			// Build merged context
			builder := ctxbuild.NewBuilder(s)
			mc, err := builder.Build(project, dept)
			if err != nil {
				return fmt.Errorf("hire: build context: %w", err)
			}

			// Append extra prompt if provided
			if extraPrompt != "" {
				data, err := os.ReadFile(extraPrompt)
				if err != nil {
					return fmt.Errorf("hire: read extra prompt: %w", err)
				}
				mc.Layers = append(mc.Layers, ctxbuild.ContextLayer{
					Source:  "extra",
					Content: string(data),
				})
			}

			// Create and populate agent working directory
			if err := os.MkdirAll(agentDir, 0o755); err != nil {
				return fmt.Errorf("hire: create agent dir: %w", err)
			}

			f, err := formatter.New(agentModel)
			if err != nil {
				return err
			}
			if err := f.Format(mc, agentDir); err != nil {
				return fmt.Errorf("hire: format context: %w", err)
			}

			// Persist agent metadata for future sync
			meta := &entity.AgentMeta{
				Name:        agentName,
				Project:     project,
				Department:  dept,
				Model:       agentModel,
				HiredAt:     time.Now().UTC(),
				ContextHash: ctxbuild.LayerHashes(mc),
			}
			if err := s.SaveAgentMeta(project, agentName, meta); err != nil {
				return fmt.Errorf("hire: save agent meta: %w", err)
			}

			printHireSuccess(agentDir, agentModel, mc, project, agentName)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().StringVar(&dept, "dept", "", "Department path, e.g. \"engineering/backend\"")
	cmd.Flags().StringVar(&model, "model", "", fmt.Sprintf("Agent model (%s)", joinModels(entity.KnownModels)))
	cmd.Flags().StringVar(&agentName, "name", "", "Name for this agent (used as directory name)")
	cmd.Flags().StringVar(&extraPrompt, "extra-prompt", "", "Path to an additional Markdown file to append to the context")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing agent directory")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("dept")
	_ = cmd.MarkFlagRequired("model")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func printHireSuccess(agentDir string, model entity.AgentModel, mc *ctxbuild.MergedContext, project, agentName string) {
	fmt.Printf("✓ Agent workspace created: %s\n\n", agentDir)
	fmt.Printf("  Model:      %s\n", model)

	fmt.Printf("  Context layers merged:\n")
	for i, l := range mc.Layers {
		lines := strings.Count(l.Content, "\n") + 1
		fmt.Printf("    [%d] %-40s (%d lines)\n", i+1, l.Source, lines)
	}

	if len(mc.Skills) > 0 {
		fmt.Printf("  Skills installed:\n")
		for _, sk := range mc.Skills {
			fmt.Printf("    - %s\n", sk.Name)
		}
	}

	fmt.Printf("\n  To start working:\n")
	fmt.Printf("    cd %s\n", agentDir)
	switch model {
	case entity.ModelClaudeCode:
		fmt.Printf("    claude\n")
	case entity.ModelCodex:
		fmt.Printf("    codex\n")
	default:
		fmt.Printf("    <your-agent-command>\n")
	}
}

func joinModels(models []entity.AgentModel) string {
	parts := make([]string, len(models))
	for i, m := range models {
		parts[i] = string(m)
	}
	return strings.Join(parts, "|")
}
