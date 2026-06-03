package main

import (
	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage and inspect individual agents",
		Long: `agent groups all commands that operate on a single agent.

Lifecycle:
  agencycli agent hire   — hire (provision) a new agent
  agencycli agent fire   — remove an agent from a project
  agencycli agent sync   — re-sync context after prompt/skill changes
  agencycli agent run    — run the next pending task
  agencycli agent exec   — run a one-off prompt (bypasses task queue)

Inspection:
  agencycli agent log    — stream or tail the run log
  agencycli agent set-model — switch agent to a different model runtime`,
	}
	cmd.AddCommand(
		newAgentLogCmd(),
		newAgentSetModelCmd(),
		newAgentSetEnvCmd(),
		newAgentUnsetEnvCmd(),
		newAgentListEnvCmd(),
		// Lifecycle aliases — identical to top-level verbs but discoverable
		// via "agencycli agent --help" (noun-verb tree search).
		newAgentHireCmd(),
		newAgentFireCmd(),
		newAgentSyncCmd(),
		newAgentRunCmd(),
		newAgentExecCmd(),
	)
	return cmd
}

// newAgentHireCmd returns a copy of the hire command registered under
// "agencycli agent hire". The top-level "agencycli hire" is kept for
// backward compatibility.
func newAgentHireCmd() *cobra.Command {
	cmd := buildHireCmd("hire")
	cmd.Short = "Hire (provision) an agent for a project"
	return cmd
}

func newAgentFireCmd() *cobra.Command {
	cmd := newFireCmd()
	cmd.Short = "Remove an agent from a project"
	return cmd
}

func newAgentSyncCmd() *cobra.Command {
	cmd := newSyncCmd()
	cmd.Short = "Re-sync agent context after prompt/skill changes"
	return cmd
}

func newAgentRunCmd() *cobra.Command {
	cmd := newRunCmd()
	cmd.Short = "Run the next pending task for an agent"
	return cmd
}

func newAgentExecCmd() *cobra.Command {
	cmd := newExecCmd()
	cmd.Short = "Run a one-off prompt against an agent (bypasses task queue)"
	return cmd
}
