package main

import (
	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage and inspect individual agents",
	}
	cmd.AddCommand(
		newAgentLogCmd(),
	)
	return cmd
}
