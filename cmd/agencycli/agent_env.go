package main

import (
	"fmt"
	"strings"

	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newAgentSetEnvCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)
	cmd := &cobra.Command{
		Use:   "set-env KEY=VALUE",
		Short: "Set an environment variable on a specific agent",
		Long: `Per-agent env vars have the highest priority and override workspace
secrets and API provider settings.`,
		Example: `  agencycli agent set-env ANTHROPIC_MODEL=claude-sonnet-4-20250514 -p myproj -a dev-claude
  agencycli agent set-env MY_TOKEN=secret123 --project myproj --agent pm`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			key, value, ok := strings.Cut(args[0], "=")
			if !ok || strings.TrimSpace(key) == "" {
				return fmt.Errorf("expected KEY=VALUE format")
			}
			key = strings.TrimSpace(key)

			st := store.NewFS(root)
			meta, err := st.AgentMeta(project, agentName)
			if err != nil {
				return fmt.Errorf("agent %s/%s: %w", project, agentName, err)
			}
			if meta.Env == nil {
				meta.Env = make(map[string]string)
			}
			meta.Env[key] = value
			if err := st.SaveAgentMeta(project, agentName, meta); err != nil {
				return err
			}
			fmt.Printf("Set %s on %s/%s\n", key, project, agentName)
			return nil
		},
	}
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name (required)")
	cmd.Flags().StringVarP(&agentName, "agent", "a", "", "Agent name (required)")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("agent")
	return cmd
}

func newAgentUnsetEnvCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)
	cmd := &cobra.Command{
		Use:   "unset-env KEY",
		Short: "Remove an environment variable from a specific agent",
		Example: `  agencycli agent unset-env ANTHROPIC_MODEL -p myproj -a dev-claude`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			key := strings.TrimSpace(args[0])
			if key == "" {
				return fmt.Errorf("key is required")
			}

			st := store.NewFS(root)
			meta, err := st.AgentMeta(project, agentName)
			if err != nil {
				return fmt.Errorf("agent %s/%s: %w", project, agentName, err)
			}
			if meta.Env == nil || meta.Env[key] == "" {
				return fmt.Errorf("env %q not set on %s/%s", key, project, agentName)
			}
			delete(meta.Env, key)
			if err := st.SaveAgentMeta(project, agentName, meta); err != nil {
				return err
			}
			fmt.Printf("Unset %s from %s/%s\n", key, project, agentName)
			return nil
		},
	}
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name (required)")
	cmd.Flags().StringVarP(&agentName, "agent", "a", "", "Agent name (required)")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("agent")
	return cmd
}

func newAgentListEnvCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)
	cmd := &cobra.Command{
		Use:   "list-env",
		Short: "List environment variables for a specific agent",
		Example: `  agencycli agent list-env -p myproj -a dev-claude`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			st := store.NewFS(root)
			meta, err := st.AgentMeta(project, agentName)
			if err != nil {
				return fmt.Errorf("agent %s/%s: %w", project, agentName, err)
			}
			if len(meta.Env) == 0 {
				fmt.Printf("No per-agent env vars set on %s/%s.\n", project, agentName)
				return nil
			}
			fmt.Printf("Per-agent env for %s/%s:\n", project, agentName)
			for k, v := range meta.Env {
				masked := maskValue(v)
				fmt.Printf("  %s=%s\n", k, masked)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name (required)")
	cmd.Flags().StringVarP(&agentName, "agent", "a", "", "Agent name (required)")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("agent")
	return cmd
}

func maskValue(v string) string {
	if len(v) <= 8 {
		return "****"
	}
	return v[:4] + "****" + v[len(v)-4:]
}
