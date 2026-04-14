package main

import (
	"fmt"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newEnvVarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "envvar",
		Short: "Manage workspace-level environment variables",
		Long: `Workspace environment variables are injected into agent processes at runtime.
Variables can apply globally (all agents) or to specific agents only.

Resolution priority (lowest → highest):
  1. Workspace global variables
  2. Workspace agent-scoped variables
  3. API provider env
  4. Per-agent env (agent set-env)`,
		Aliases: []string{"ev"},
	}
	cmd.AddCommand(
		newEnvVarAddCmd(),
		newEnvVarListCmd(),
		newEnvVarRemoveCmd(),
	)
	return cmd
}

func newEnvVarAddCmd() *cobra.Command {
	var (
		scope       string
		agents      string
		description string
	)
	cmd := &cobra.Command{
		Use:   "add KEY=VALUE",
		Short: "Add a workspace environment variable",
		Example: `  # Add a global variable (applied to all agents)
  agencycli envvar add GITHUB_TOKEN=ghp_xxxx

  # Add a variable for specific agents only
  agencycli envvar add MY_API_KEY=sk-xxx --scope agents --agents "myproj/dev-claude,myproj/pm"

  # Add with description
  agencycli envvar add NPM_TOKEN=npm_xxx --description "npm publish token"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			key, value, ok := strings.Cut(args[0], "=")
			if !ok || key == "" {
				return fmt.Errorf("expected KEY=VALUE format")
			}

			ev := entity.EnvVar{
				Key:         strings.TrimSpace(key),
				Value:       value,
				Scope:       entity.EnvVarScope(scope),
				Description: description,
			}
			if scope == "agents" && agents != "" {
				for _, a := range strings.Split(agents, ",") {
					a = strings.TrimSpace(a)
					if a != "" {
						ev.Agents = append(ev.Agents, a)
					}
				}
			}
			es := store.NewEnvVarStore(root)
			created, err := es.Add(ev)
			if err != nil {
				return err
			}
			fmt.Printf("Variable added: %s (id: %s, scope: %s)\n", ev.Key, created.ID, ev.Scope)
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "global", "Scope: global (all agents) or agents (specific agents)")
	cmd.Flags().StringVar(&agents, "agents", "", "Comma-separated list of project/agent IDs (when --scope=agents)")
	cmd.Flags().StringVar(&description, "description", "", "Optional description")
	return cmd
}

func newEnvVarListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all workspace environment variables",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			es := store.NewEnvVarStore(root)
			items, err := es.List()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Println("No environment variables configured.")
				return nil
			}
			fmt.Printf("%-14s %-24s %-8s %-30s %s\n", "ID", "KEY", "SCOPE", "AGENTS", "DESCRIPTION")
			for _, v := range items {
				agentStr := "-"
				if len(v.Agents) > 0 {
					agentStr = strings.Join(v.Agents, ",")
				}
				desc := v.Description
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				fmt.Printf("%-14s %-24s %-8s %-30s %s\n", v.ID, v.Key, v.Scope, agentStr, desc)
			}
			return nil
		},
	}
}

func newEnvVarRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <id-or-key>",
		Short:   "Remove a workspace environment variable by ID or key name",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			es := store.NewEnvVarStore(root)
			target := args[0]

			if err := es.Remove(target); err == nil {
				fmt.Printf("Variable %s removed.\n", target)
				return nil
			}
			items, err := es.List()
			if err != nil {
				return err
			}
			for _, v := range items {
				if v.Key == target {
					if err := es.Remove(v.ID); err != nil {
						return err
					}
					fmt.Printf("Variable %s (id: %s) removed.\n", v.Key, v.ID)
					return nil
				}
			}
			return fmt.Errorf("variable %q not found", target)
		},
	}
}
