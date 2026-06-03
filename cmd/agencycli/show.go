package main

import (
	"fmt"
	"strings"

	"github.com/chenhg5/agencycli/internal/ctxbuild"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show details of a workspace object",
	}
	cmd.AddCommand(
		newShowTeamCmd(),
		newShowProjectCmd(),
		newShowAgentCmd(),
	)
	return cmd
}

// ── show team ─────────────────────────────────────────────────────────────────

func newShowTeamCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "team <path>",
		Short: "Show team details and its prompt",
		Example: `  agencycli show team engineering/backend
  agencycli show team engineering --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamPath := args[0]
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			t, err := s.Team(teamPath)
			if err != nil {
				return err
			}
			prompt, _ := s.TeamPrompt(teamPath)

			if resolveFormat(format) == "json" {
				type teamOut struct {
					Path        string   `json:"path"`
					Description string   `json:"description"`
					Parent      string   `json:"parent,omitempty"`
					Goals       []string `json:"goals,omitempty"`
					Skills      []string `json:"skills,omitempty"`
					Prompt      string   `json:"prompt,omitempty"`
				}
				return printJSON(teamOut{
					Path:        teamPath,
					Description: t.Description,
					Parent:      t.Parent,
					Goals:       t.Goals,
					Skills:      t.Skills,
					Prompt:      prompt,
				})
			}

			fmt.Printf("Team: %s\n", teamPath)
			if t.Description != "" {
				fmt.Printf("  Description: %s\n", t.Description)
			}
			if t.Parent != "" {
				fmt.Printf("  Parent:      %s\n", t.Parent)
			}
			if len(t.Goals) > 0 {
				fmt.Printf("  Goals:\n")
				for _, g := range t.Goals {
					fmt.Printf("    - %s\n", g)
				}
			}
			if len(t.Skills) > 0 {
				fmt.Printf("  Skills:      %s\n", strings.Join(t.Skills, ", "))
			}
			if prompt != "" {
				fmt.Printf("\n--- prompt.md ---\n%s\n", prompt)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "output format: json or table (default: json)")
	return cmd
}

// ── show project ──────────────────────────────────────────────────────────────

func newShowProjectCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "project <name>",
		Short: "Show project details and its prompt",
		Example: `  agencycli show project my-api
  agencycli show project my-api --format json`,
		Args: cobra.ExactArgs(1),
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
			agents, _ := s.ListAgents(name)

			if resolveFormat(format) == "json" {
				type agentSummary struct {
					Name  string `json:"name"`
					Model string `json:"model"`
					Team  string `json:"team"`
				}
				type projectOut struct {
					Name        string         `json:"name"`
					Description string         `json:"description,omitempty"`
					Repo        string         `json:"repo,omitempty"`
					Agents      []agentSummary `json:"agents"`
					Prompt      string         `json:"prompt,omitempty"`
				}
				out := projectOut{
					Name:        name,
					Description: p.Description,
					Repo:        p.Repo,
					Agents:      []agentSummary{},
					Prompt:      prompt,
				}
				for _, a := range agents {
					out.Agents = append(out.Agents, agentSummary{
						Name:  a.Name,
						Model: string(a.Meta.Model),
						Team:  a.Meta.Team,
					})
				}
				return printJSON(out)
			}

			fmt.Printf("Project: %s\n", name)
			if p.Description != "" {
				fmt.Printf("  Description: %s\n", p.Description)
			}
			if p.Repo != "" {
				fmt.Printf("  Repo:        %s\n", p.Repo)
			}
			if len(agents) > 0 {
				fmt.Printf("  Agents:\n")
				for _, a := range agents {
					fmt.Printf("    - %-16s  model:%-12s  team:%s\n",
						a.Name, a.Meta.Model, a.Meta.Team)
				}
			}
			if prompt != "" {
				fmt.Printf("\n--- prompt.md ---\n%s\n", prompt)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "output format: json or table (default: json)")
	return cmd
}

// ── show agent ────────────────────────────────────────────────────────────────

func newShowAgentCmd() *cobra.Command {
	var raw bool
	var format string

	cmd := &cobra.Command{
		Use:   "agent <project> <name>",
		Short: "Show merged context for a hired agent",
		Example: `  agencycli show agent my-api dev
  agencycli show agent my-api dev --format json
  agencycli show agent my-api dev --raw`,
		Args: cobra.ExactArgs(2),
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

			builder := ctxbuild.NewBuilder(s)
			mc, err := builder.Build(project, meta.Team, meta.Role)
			if err != nil {
				return err
			}

			if resolveFormat(format) == "json" && !raw {
				type layerSummary struct {
					Source string `json:"source"`
					Lines  int    `json:"lines"`
				}
				type agentOut struct {
					Project  string         `json:"project"`
					Name     string         `json:"name"`
					Model    string         `json:"model"`
					Team     string         `json:"team"`
					Role     string         `json:"role,omitempty"`
					HiredAt  string         `json:"hired_at"`
					Dir      string         `json:"dir"`
					Layers   []layerSummary `json:"context_layers"`
					Skills   []string       `json:"skills"`
				}
				out := agentOut{
					Project:  project,
					Name:     agentName,
					Model:    string(meta.Model),
					Team:     meta.Team,
					Role:     meta.Role,
					HiredAt:  meta.HiredAt.Format("2006-01-02T15:04:05Z"),
					Dir:      s.AgentDir(project, agentName),
					Layers:   []layerSummary{},
					Skills:   []string{},
				}
				for _, l := range mc.Layers {
					out.Layers = append(out.Layers, layerSummary{
						Source: l.Source,
						Lines:  strings.Count(l.Content, "\n") + 1,
					})
				}
				for _, sk := range mc.Skills {
					out.Skills = append(out.Skills, sk.Name)
				}
				return printJSON(out)
			}

			fmt.Printf("Agent:     %s/%s\n", project, agentName)
			fmt.Printf("Model:     %s\n", meta.Model)
			fmt.Printf("Team:      %s\n", meta.Team)
			fmt.Printf("Hired at:  %s\n", meta.HiredAt.Format("2006-01-02 15:04:05 UTC"))
			fmt.Printf("Agent dir: %s\n", s.AgentDir(project, agentName))

			if raw {
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

	cmd.Flags().BoolVar(&raw, "raw", false, "print the full merged context content")
	cmd.Flags().StringVar(&format, "format", "", "output format: json or table (default: json)")
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
