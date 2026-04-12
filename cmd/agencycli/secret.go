package main

import (
	"fmt"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage workspace-level secrets (environment variables)",
		Long: `Workspace secrets are injected into agent processes at runtime.
Secrets can apply globally (all agents) or to specific agents only.

Resolution priority (lowest → highest):
  1. Workspace global secrets
  2. Workspace agent-scoped secrets
  3. API provider env
  4. Per-agent env (agent set-env)`,
	}
	cmd.AddCommand(
		newSecretAddCmd(),
		newSecretListCmd(),
		newSecretRemoveCmd(),
	)
	return cmd
}

func newSecretAddCmd() *cobra.Command {
	var (
		scope       string
		agents      string
		description string
	)
	cmd := &cobra.Command{
		Use:   "add KEY=VALUE",
		Short: "Add a workspace secret",
		Example: `  # Add a global secret (applied to all agents)
  agencycli secret add GITHUB_TOKEN=ghp_xxxx

  # Add a secret for specific agents only
  agencycli secret add MY_API_KEY=sk-xxx --scope agents --agents "myproj/dev-claude,myproj/pm"

  # Add with description
  agencycli secret add NPM_TOKEN=npm_xxx --description "npm publish token"`,
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

			sec := entity.Secret{
				Key:         strings.TrimSpace(key),
				Value:       value,
				Scope:       entity.SecretScope(scope),
				Description: description,
			}
			if scope == "agents" && agents != "" {
				for _, a := range strings.Split(agents, ",") {
					a = strings.TrimSpace(a)
					if a != "" {
						sec.Agents = append(sec.Agents, a)
					}
				}
			}
			ss := store.NewSecretStore(root)
			created, err := ss.Add(sec)
			if err != nil {
				return err
			}
			fmt.Printf("Secret added: %s (id: %s, scope: %s)\n", sec.Key, created.ID, sec.Scope)
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "global", "Scope: global (all agents) or agents (specific agents)")
	cmd.Flags().StringVar(&agents, "agents", "", "Comma-separated list of project/agent IDs (when --scope=agents)")
	cmd.Flags().StringVar(&description, "description", "", "Optional description")
	return cmd
}

func newSecretListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all workspace secrets (values masked)",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ss := store.NewSecretStore(root)
			items, err := ss.List()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Println("No secrets configured.")
				return nil
			}
			fmt.Printf("%-14s %-24s %-8s %-30s %s\n", "ID", "KEY", "SCOPE", "AGENTS", "DESCRIPTION")
			for _, s := range items {
				agentStr := "-"
				if len(s.Agents) > 0 {
					agentStr = strings.Join(s.Agents, ",")
				}
				desc := s.Description
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				fmt.Printf("%-14s %-24s %-8s %-30s %s\n", s.ID, s.Key, s.Scope, agentStr, desc)
			}
			return nil
		},
	}
}

func newSecretRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id-or-key>",
		Short: "Remove a workspace secret by ID or key name",
		Aliases: []string{"rm"},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ss := store.NewSecretStore(root)
			target := args[0]

			// Try by ID first
			if err := ss.Remove(target); err == nil {
				fmt.Printf("Secret %s removed.\n", target)
				return nil
			}
			// Try by key name
			items, err := ss.List()
			if err != nil {
				return err
			}
			for _, s := range items {
				if s.Key == target {
					if err := ss.Remove(s.ID); err != nil {
						return err
					}
					fmt.Printf("Secret %s (id: %s) removed.\n", s.Key, s.ID)
					return nil
				}
			}
			return fmt.Errorf("secret %q not found", target)
		},
	}
}
