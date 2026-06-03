package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspace objects",
	}
	cmd.AddCommand(
		newListTeamsCmd(),
		newListProjectsCmd(),
		newListAgentsCmd(),
		newListSkillsCmd(),
	)
	return cmd
}

func newListTeamsCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "teams",
		Aliases: []string{"team"},
		Short:   "List all teams",
		Example: `  agencycli list teams
  agencycli list teams --format table`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			entries, err := s.ListTeams()
			if err != nil {
				return err
			}

			if resolveFormat(format) == "json" {
				if entries == nil {
					entries = []*store.TeamEntry{}
				}
				return printJSON(entries)
			}

			if len(entries) == 0 {
				fmt.Println("No teams found. Run: agencycli create team --name <name>")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PATH\tDESCRIPTION\tSKILLS")
			for _, e := range entries {
				fmt.Fprintf(w, "%s\t%s\t%s\n", e.Path, e.Team.Description, strings.Join(e.Team.Skills, ", "))
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "output format: json or table (default: json)")
	return cmd
}

func newListProjectsCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project"},
		Short:   "List all projects",
		Example: `  agencycli list projects
  agencycli list projects --format table`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			projects, err := s.ListProjects()
			if err != nil {
				return err
			}

			if resolveFormat(format) == "json" {
				if projects == nil {
					projects = []*entity.Project{}
				}
				return printJSON(projects)
			}

			if len(projects) == 0 {
				fmt.Println("No projects found. Run: agencycli create project --name <name>")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION\tREPO")
			for _, p := range projects {
				fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Description, p.Repo)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "output format: json or table (default: json)")
	return cmd
}

func newListAgentsCmd() *cobra.Command {
	var project string
	var format string

	cmd := &cobra.Command{
		Use:     "agents",
		Aliases: []string{"agent"},
		Short:   "List hired agents",
		Example: `  agencycli list agents
  agencycli list agents --project cc-connect
  agencycli list agents --format table`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			projectNames := []string{}
			if project != "" {
				projectNames = []string{project}
			} else {
				allProjects, err := s.ListProjects()
				if err != nil {
					return err
				}
				for _, p := range allProjects {
					projectNames = append(projectNames, p.Name)
				}
			}

			type agentRow struct {
				Project string `json:"project"`
				Name    string `json:"name"`
				Model   string `json:"model"`
				Team    string `json:"team"`
				Dir     string `json:"dir"`
			}
			var rows []agentRow
			for _, pName := range projectNames {
				agents, err := s.ListAgents(pName)
				if err != nil {
					continue
				}
				for _, a := range agents {
					rows = append(rows, agentRow{
						Project: pName,
						Name:    a.Name,
						Model:   string(a.Meta.Model),
						Team:    a.Meta.Team,
						Dir:     s.AgentDir(pName, a.Name),
					})
				}
			}

			if resolveFormat(format) == "json" {
				if rows == nil {
					rows = []agentRow{}
				}
				return printJSON(rows)
			}

			if len(rows) == 0 {
				fmt.Println("No agents found. Run: agencycli hire --help")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROJECT\tNAME\tMODEL\tTEAM")
			for _, r := range rows {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Project, r.Name, r.Model, r.Team)
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "limit to a specific project")
	cmd.Flags().StringVar(&format, "format", "", "output format: json or table (default: json)")
	return cmd
}

func newListSkillsCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "skills",
		Aliases: []string{"skill"},
		Short:   "List available skills",
		Example: `  agencycli list skills
  agencycli list skills --format table`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			skills, err := s.ListSkills()
			if err != nil {
				return err
			}

			if resolveFormat(format) == "json" {
				if skills == nil {
					skills = []*entity.Skill{}
				}
				return printJSON(skills)
			}

			if len(skills) == 0 {
				fmt.Println("No skills found.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION")
			for _, sk := range skills {
				fmt.Fprintf(w, "%s\t%s\n", sk.Name, sk.Description)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "output format: json or table (default: json)")
	return cmd
}
