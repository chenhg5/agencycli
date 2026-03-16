package main

import (
	"fmt"
	"strings"

	"github.com/agencycli/agencycli/internal/ctxbuild"
	"github.com/agencycli/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show details of a workspace object",
	}
	cmd.AddCommand(
		newShowDeptCmd(),
		newShowProjectCmd(),
		newShowAgentCmd(),
	)
	return cmd
}

// ── show dept ─────────────────────────────────────────────────────────────────

func newShowDeptCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "dept <path>",
		Short:   "Show department details and its prompt",
		Example: `  agencycli show dept engineering/backend`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deptPath := args[0]
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			d, err := s.Department(deptPath)
			if err != nil {
				return err
			}
			prompt, _ := s.DeptPrompt(deptPath)

			fmt.Printf("Department: %s\n", deptPath)
			if d.Description != "" {
				fmt.Printf("  Description: %s\n", d.Description)
			}
			if d.Parent != "" {
				fmt.Printf("  Parent:      %s\n", d.Parent)
			}
			if len(d.Goals) > 0 {
				fmt.Printf("  Goals:\n")
				for _, g := range d.Goals {
					fmt.Printf("    - %s\n", g)
				}
			}
			if len(d.Skills) > 0 {
				fmt.Printf("  Skills:      %s\n", strings.Join(d.Skills, ", "))
			}
			if prompt != "" {
				fmt.Printf("\n--- prompt.md ---\n%s\n", prompt)
			}
			return nil
		},
	}
}

// ── show project ──────────────────────────────────────────────────────────────

func newShowProjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "project <name>",
		Short:   "Show project details and its prompt",
		Example: `  agencycli show project my-api`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			p, err := s.Project(name)
			if err != nil {
				return err
			}
			prompt, _ := s.ProjectPrompt(name)

			fmt.Printf("Project: %s\n", name)
			if p.Description != "" {
				fmt.Printf("  Description: %s\n", p.Description)
			}
			if p.Repo != "" {
				fmt.Printf("  Repo:        %s\n", p.Repo)
			}

			// List hired agents
			agents, _ := s.ListAgents(name)
			if len(agents) > 0 {
				fmt.Printf("  Agents:\n")
				for _, a := range agents {
					fmt.Printf("    - %-16s  model:%-12s  dept:%s\n",
						a.Name, a.Meta.Model, a.Meta.Department)
				}
			}

			if prompt != "" {
				fmt.Printf("\n--- prompt.md ---\n%s\n", prompt)
			}
			return nil
		},
	}
}

// ── show agent ────────────────────────────────────────────────────────────────

func newShowAgentCmd() *cobra.Command {
	var raw bool

	cmd := &cobra.Command{
		Use:     "agent <project> <name>",
		Short:   "Show merged context for a hired agent",
		Example: `  agencycli show agent my-api dev`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, agentName := args[0], args[1]
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			meta, err := s.AgentMeta(project, agentName)
			if err != nil {
				return err
			}

			fmt.Printf("Agent:      %s/%s\n", project, agentName)
			fmt.Printf("Model:      %s\n", meta.Model)
			fmt.Printf("Department: %s\n", meta.Department)
			fmt.Printf("Hired at:   %s\n", meta.HiredAt.Format("2006-01-02 15:04:05 UTC"))
			fmt.Printf("Agent dir:  %s\n", s.AgentDir(project, agentName))

			if raw {
				// Rebuild and print the full merged context
				builder := ctxbuild.NewBuilder(s)
				mc, err := builder.Build(project, meta.Department)
				if err != nil {
					return err
				}
				fmt.Printf("\n%s\n", separator("MERGED CONTEXT"))
				for _, l := range mc.Layers {
					fmt.Printf("\n## [%s]\n\n%s\n", l.Source, strings.TrimSpace(l.Content))
				}
				if len(mc.Skills) > 0 {
					fmt.Printf("\n%s\n", separator("SKILLS"))
					for _, sk := range mc.Skills {
						fmt.Printf("\n### %s\n\n%s\n", sk.Name, strings.TrimSpace(sk.Prompt))
					}
				}
			} else {
				// Just show layer summary
				builder := ctxbuild.NewBuilder(s)
				mc, err := builder.Build(project, meta.Department)
				if err != nil {
					return err
				}
				fmt.Printf("\nContext layers:\n")
				total := 0
				for i, l := range mc.Layers {
					lines := strings.Count(l.Content, "\n") + 1
					total += lines
					fmt.Printf("  [%d] %-40s %d lines\n", i+1, l.Source, lines)
				}
				fmt.Printf("  Total: %d lines\n", total)
				if len(mc.Skills) > 0 {
					fmt.Printf("\nSkills: %s\n", joinSkillNames(mc.Skills))
				}
				fmt.Printf("\nTip: use --raw to print the full merged content\n")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Print the full merged context content")
	return cmd
}

func separator(label string) string {
	line := strings.Repeat("─", 60)
	return fmt.Sprintf("%s %s %s", line[:20], label, line[:20])
}

func joinSkillNames(skills []ctxbuild.SkillDef) string {
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}
