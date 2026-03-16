package main

import (
	"fmt"
	"time"

	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
)

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage agent session IDs",
		Long: `Agent sessions allow the agent CLI to resume a prior conversation thread,
giving the agent memory of past work across heartbeat cycles.

For Claude Code, the session ID is captured automatically from the agent's
stream-json output. For other models, it can be set manually.`,
	}
	cmd.AddCommand(
		newSessionShowCmd(),
		newSessionSetCmd(),
		newSessionResetCmd(),
	)
	return cmd
}

func newSessionShowCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the current session ID for an agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}
			ts := taskstore.New(root)
			hb, err := ts.GetHeartbeat(project, agentName)
			if err != nil {
				return err
			}
			if hb.SessionID == "" {
				fmt.Printf("No session ID set for %s/%s\n", project, agentName)
				return nil
			}
			fmt.Printf("Session ID : %s\n", hb.SessionID)
			if hb.SessionStartedAt != nil {
				fmt.Printf("Started    : %s\n", hb.SessionStartedAt.Format(time.RFC3339))
			}
			fmt.Printf("Scope      : %s\n", hb.SessionScope)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	return cmd
}

func newSessionSetCmd() *cobra.Command {
	var (
		project   string
		agentName string
		sessionID string
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Manually set the session ID for an agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" || sessionID == "" {
				return fmt.Errorf("--project, --agent, and --id are required")
			}
			ts := taskstore.New(root)
			hb, err := ts.GetHeartbeat(project, agentName)
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			hb.SessionID = sessionID
			hb.SessionStartedAt = &now
			if err := ts.SaveHeartbeat(project, agentName, hb); err != nil {
				return err
			}
			fmt.Printf("✓ Session ID set for %s/%s: %s\n", project, agentName, sessionID)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	cmd.Flags().StringVar(&sessionID, "id", "", "session ID to set")
	return cmd
}

func newSessionResetCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Clear the session ID (next run starts a fresh conversation)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}
			ts := taskstore.New(root)
			hb, err := ts.GetHeartbeat(project, agentName)
			if err != nil {
				return err
			}
			old := hb.SessionID
			hb.SessionID = ""
			hb.SessionStartedAt = nil
			if err := ts.SaveHeartbeat(project, agentName, hb); err != nil {
				return err
			}
			if old != "" {
				fmt.Printf("✓ Session ID cleared for %s/%s (was: %s)\n", project, agentName, old)
			} else {
				fmt.Printf("✓ No session ID was set for %s/%s\n", project, agentName)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	return cmd
}
