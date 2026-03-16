package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentorg/agentorg/internal/entity"
	"github.com/agentorg/agentorg/internal/scaffold"
	"github.com/agentorg/agentorg/internal/store"
	"github.com/agentorg/agentorg/internal/workspace"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create workspace objects (company, dept, project)",
	}
	cmd.AddCommand(
		newCreateCompanyCmd(),
		newCreateDeptCmd(),
		newCreateProjectCmd(),
	)
	return cmd
}

// ── create company ────────────────────────────────────────────────────────────

func newCreateCompanyCmd() *cobra.Command {
	var (
		name string
		desc string
	)

	cmd := &cobra.Command{
		Use:   "company",
		Short: "Initialise a new agentorg workspace",
		Example: `  agentorg create company --name "Acme Corp" --desc "Building the future"
  cd "Acme Corp"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			// Create the company root directory
			root, err := filepath.Abs(name)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(root, 0o755); err != nil {
				return fmt.Errorf("create company dir: %w", err)
			}

			c := &entity.Company{Name: name, Description: desc}
			if err := scaffold.InitCompany(root, c); err != nil {
				return err
			}

			fmt.Printf("✓ Company workspace created: %s\n", root)
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  cd %q\n", name)
			fmt.Printf("  agentorg create dept --name \"engineering\"\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Company name (also used as directory name)")
	cmd.Flags().StringVar(&desc, "desc", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// ── create dept ───────────────────────────────────────────────────────────────

func newCreateDeptCmd() *cobra.Command {
	var (
		name   string
		desc   string
		skills []string
	)

	cmd := &cobra.Command{
		Use:   "dept",
		Short: "Create a department (supports nested paths)",
		Example: `  agentorg create dept --name "engineering"
  agentorg create dept --name "engineering/backend" --desc "Go/gRPC" --skills "git,bash"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			root, err := workspace.FindRootFromCWD()
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

			d := &entity.Department{
				Name:        filepath.Base(name),
				Parent:      parent,
				Description: desc,
				Skills:      skills,
			}
			if err := sc.CreateDept(name, d); err != nil {
				return err
			}

			fmt.Printf("✓ Department created: departments/%s\n", name)
			fmt.Printf("  Edit the prompt: vim departments/%s/prompt.md\n",
				filepath.FromSlash(name))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Department path, e.g. \"engineering\" or \"engineering/backend\"")
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
		Example: `  agentorg create project --name "my-api" --desc "REST API" --repo "../my-api"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			root, err := workspace.FindRootFromCWD()
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
			fmt.Printf("  Hire an agent:   agentorg hire --project %q --dept \"...\" --model \"claude-code\" --name \"dev\"\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&desc, "desc", "", "Short description")
	cmd.Flags().StringVar(&repo, "repo", "", "Path to the project code repository")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
