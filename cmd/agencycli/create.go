package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenhg5/agencycli/internal/ctxbuild"
	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/scaffold"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create workspace objects (agency, team, project)",
	}
	cmd.AddCommand(
		newCreateAgencyCmd(),
		newCreateTeamCmd(),
		newCreateProjectCmd(),
	)
	return cmd
}

// ── create agency ─────────────────────────────────────────────────────────────

func newCreateAgencyCmd() *cobra.Command {
	var (
		name string
		desc string
	)

	cmd := &cobra.Command{
		Use:   "agency",
		Short: "Initialise a new agencycli workspace",
		Example: `  agencycli create agency --name "Acme Agency" --desc "Building the future"
  cd "Acme Agency"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			root, err := filepath.Abs(name)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(root, 0o755); err != nil {
				return fmt.Errorf("create agency dir: %w", err)
			}

			a := &entity.Agency{Name: name, Description: desc}
			if err := scaffold.InitAgency(root, a); err != nil {
				return err
			}

			fmt.Printf("✓ Agency workspace created: %s\n", root)
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  cd %q\n", name)
			fmt.Printf("  agencycli create team --name \"engineering\"\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Agency name (also used as directory name)")
	cmd.Flags().StringVar(&desc, "desc", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// ── create team ───────────────────────────────────────────────────────────────

func newCreateTeamCmd() *cobra.Command {
	var (
		name   string
		desc   string
		skills []string
	)

	cmd := &cobra.Command{
		Use:   "team",
		Short: "Create a team (supports nested paths)",
		Example: `  agencycli create team --name "engineering"
  agencycli create team --name "engineering/backend" --desc "Go/gRPC" --skills "git,bash"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			sc := scaffold.New(s)

			// Determine parent from path
			parent := ""
			if idx := strings.LastIndex(name, "/"); idx != -1 {
				parent = name[:idx]
			}

			// Every level in the chain must already exist before we can
			// create a child team. This ensures each level has its own
			// team.yaml and prompt.md so the context chain is complete.
			if parent != "" {
				chain := ctxbuild.ResolveChain(parent)
				for _, ancestor := range chain {
					if _, err := s.Team(ancestor); err != nil {
						return fmt.Errorf(
							"parent team %q does not exist\n"+
								"Create it first with:\n"+
								"  agencycli create team --name %q",
							ancestor, ancestor,
						)
					}
				}
			}

			t := &entity.Team{
				Name:        filepath.Base(name),
				Parent:      parent,
				Description: desc,
				Skills:      skills,
			}
			if err := sc.CreateTeam(name, t); err != nil {
				return err
			}

			fmt.Printf("✓ Team created: teams/%s\n", name)
			fmt.Printf("  Edit the prompt: vim teams/%s/prompt.md\n",
				filepath.FromSlash(name))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Team path, e.g. \"engineering\" or \"engineering/backend\"")
	cmd.Flags().StringVar(&desc, "desc", "", "Short description")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Comma-separated skill names, e.g. git,bash")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// ── create project ────────────────────────────────────────────────────────────

func newCreateProjectCmd() *cobra.Command {
	var (
		name string
		desc string
		repo string
	)

	cmd := &cobra.Command{
		Use:   "project",
		Short: "Create a project",
		Example: `  agencycli create project --name "my-api" --desc "REST API" --repo "../my-api"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)
			sc := scaffold.New(s)

			p := &entity.Project{Name: name, Description: desc, Repo: repo}
			if err := sc.CreateProject(name, p); err != nil {
				return err
			}

			fmt.Printf("✓ Project created: projects/%s\n", name)
			fmt.Printf("  Edit the prompt: vim projects/%s/prompt.md\n", name)
			fmt.Printf("  Hire an agent:   agencycli hire --project %q --team \"...\" --model \"claudecode\" --name \"dev\"\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&desc, "desc", "", "Short description")
	cmd.Flags().StringVar(&repo, "repo", "", "Path to the project code repository")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
