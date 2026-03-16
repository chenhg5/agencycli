package main

import (
	"fmt"
	"strings"

	"github.com/agentorg/agentorg/internal/store"
	"github.com/agentorg/agentorg/internal/workspace"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspace objects",
	}
	cmd.AddCommand(
		newListDeptsCmd(),
		newListProjectsCmd(),
		newListAgentsCmd(),
		newListSkillsCmd(),
	)
	return cmd
}

func newListDeptsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "depts",
		Aliases: []string{"dept"},
		Short:   "List all departments",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := workspace.FindRootFromCWD()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			entries, err := s.ListDepartments()
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("No departments found. Run: agentorg create dept --name <name>")
				return nil
			}
			fmt.Println("Departments:")
			for _, e := range entries {
				desc := ""
				if e.Department.Description != "" {
					desc = "  — " + e.Department.Description
				}
				skills := ""
				if len(e.Department.Skills) > 0 {
					skills = fmt.Sprintf(" [skills: %s]", strings.Join(e.Department.Skills, ", "))
				}
				fmt.Printf("  %-36s%s%s\n", e.Path, desc, skills)
			}
			return nil
		},
	}
}

func newListProjectsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project"},
		Short:   "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := workspace.FindRootFromCWD()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			projects, err := s.ListProjects()
			if err != nil {
				return err
			}
			if len(projects) == 0 {
				fmt.Println("No projects found. Run: agentorg create project --name <name>")
				return nil
			}
			fmt.Println("Projects:")
			for _, p := range projects {
				repo := ""
				if p.Repo != "" {
					repo = "  repo: " + p.Repo
				}
				fmt.Printf("  %-24s  %s%s\n", p.Name, p.Description, repo)
			}
			return nil
		},
	}
}

func newListAgentsCmd() *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:     "agents",
		Aliases: []string{"agent"},
		Short:   "List hired agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := workspace.FindRootFromCWD()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			// If no project given, list for all projects
			projectNames := []string{}
			if project != "" {
				projectNames = []string{project}
			} else {
				projects, err := s.ListProjects()
				if err != nil {
					return err
				}
				for _, p := range projects {
					projectNames = append(projectNames, p.Name)
				}
			}

			found := 0
			for _, pName := range projectNames {
				agents, err := s.ListAgents(pName)
				if err != nil {
					continue
				}
				for _, a := range agents {
					fmt.Printf("  %-16s  %-24s  model:%-12s  dept:%s\n",
						pName, a.Name, a.Meta.Model, a.Meta.Department)
					found++
				}
			}
			if found == 0 {
				fmt.Println("No agents found. Run: agentorg hire --help")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Limit to a specific project")
	return cmd
}

func newListSkillsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "skills",
		Aliases: []string{"skill"},
		Short:   "List available skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := workspace.FindRootFromCWD()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			skills, err := s.ListSkills()
			if err != nil {
				return err
			}
			if len(skills) == 0 {
				fmt.Println("No skills found.")
				return nil
			}
			fmt.Println("Skills:")
			for _, sk := range skills {
				fmt.Printf("  %-20s  %s\n", sk.Name, sk.Description)
			}
			return nil
		},
	}
}
