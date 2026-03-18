package main

import (
	"fmt"
	"os"

	"github.com/chenhg5/agencycli/internal/workspace"
	"github.com/spf13/cobra"
)

// globalDir holds the value of the global --dir flag.
// When non-empty it is used as the starting point for workspace discovery
// instead of the current working directory.
var globalDir string

var rootCmd = &cobra.Command{
	Use:   "agencycli",
	Short: "AI agent organisation and context management",
	Long: `agencycli manages the organisational context for AI agents.

It lets you create an agency with teams and projects, then hire (or assign)
AI agents into projects with the right accumulated context so they can
start working immediately.

Typical workflow:

  agencycli create agency --name "MyAgency"
  cd MyAgency

  agencycli create team --name "engineering"
  agencycli create team --name "engineering/backend" --desc "Go/gRPC services"

  agencycli create project --name "my-api" --repo "../my-api"

  agencycli hire   --project "my-api" --team "engineering/backend" \
                   --model "claudecode" --name "dev"
  # or equivalently:
  agencycli assign --project "my-api" --team "engineering/backend" \
                   --model "cursor" --name "cursor-dev"

  cd projects/my-api/agents/dev
  claude

You can run any command from outside the workspace by passing --dir:

  agencycli --dir /path/to/MyAgency list teams`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&globalDir, "dir", "",
		"workspace directory (default: auto-discover from current directory)",
	)

	rootCmd.AddCommand(
		newVersionCmd(),
		newCreateCmd(),
		newRoleCmd(),
		newHireCmd(),
		newAssignCmd(),
		newFireCmd(),
		newSyncCmd(),
		newListCmd(),
		newShowCmd(),
		newTaskCmd(),
		newRunCmd(),
		newExecCmd(),
		newInboxCmd(),
		newSessionCmd(),
		newSchedulerCmd(),
		newCronCmd(),
		newTemplateCmd(),
		newProjectCmd(),
		newSandboxCmd(),
	)
}

// resolveRoot returns the absolute path of the agencycli workspace root.
// If --dir is set it searches from that path; otherwise it searches from CWD.
func resolveRoot() (string, error) {
	if globalDir != "" {
		return workspace.FindRoot(globalDir)
	}
	return workspace.FindRootFromCWD()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
