package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/chenhg5/agencycli/internal/errs"
	"github.com/chenhg5/agencycli/internal/workspace"
	"github.com/spf13/cobra"
)

// globalDir holds the value of the global --dir flag.
// When non-empty it is used as the starting point for workspace discovery
// instead of the current working directory.
var globalDir string

var rootCmd = &cobra.Command{
	SilenceErrors: true, // error is printed by main()
	SilenceUsage:  true, // no usage on error
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
  agencycli create project --name "my-api" --repo "../my-api"

  agencycli agent hire --project "my-api" --team "engineering" \
                       --model "claudecode" --name "dev"
  agencycli task add   --project "my-api" --agent "dev" \
                       --title "Implement feature X" --prompt "..." \
                       --created-by human
  agencycli agent run  --project "my-api" --agent "dev"

Agent-friendly tips:
  - All list/show commands output JSON by default (--format table for humans)
  - agencycli schema [command]  — discover any command's flags as JSON
  - Exit codes: 0=ok  1=error  2=bad-args  3=not-found  5=conflict
  - run --dry-run  outputs JSON preview instead of executing

You can run any command from outside the workspace by passing --dir:

  agencycli --dir /path/to/MyAgency list agents`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&globalDir, "dir", "",
		"workspace directory (default: auto-discover from current directory)",
	)

	rootCmd.AddCommand(
		newVersionCmd(),
		newCreateCmd(),
		newTeamCmd(),
		newRoleCmd(),
		newHireCmd(),
		newAssignCmd(),
		newFireCmd(),
		newSyncCmd(),
		newListCmd(),
		newShowCmd(),
		newTaskCmd(),
		newAgentCmd(),
		newRunCmd(),
		newExecCmd(),
		newInboxCmd(),
		newSessionCmd(),
		newSchedulerCmd(),
		newCronCmd(),
		newTemplateCmd(),
		newProjectCmd(),
		newSandboxCmd(),
		newOverviewCmd(),
		newRunsCmd(),
		newClearCmd(),
		newAPICmd(),
		newStartCmd(),
		newDaemonCmd(),
		newCheckUpdateCmd(),
		newUpdateCmd(),
		newDocsCmd(),
		newOKRCmd(),
		newMilestoneCmd(),
		newEnvVarCmd(),
		newProviderCmd(),
		newSchemaCmd(),
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
	checkUpdateAsync()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeFor(err))
	}
}

// exitCodeFor maps sentinel error types to meaningful exit codes:
//
//	0   success
//	1   general error (default)
//	2   usage / bad arguments
//	3   resource not found
//	5   conflict / already exists
func exitCodeFor(err error) int {
	var notFound *errs.NotFoundError
	if errors.As(err, &notFound) {
		return 3
	}
	var conflict *errs.ConflictError
	if errors.As(err, &conflict) {
		return 5
	}
	var usage *errs.UsageError
	if errors.As(err, &usage) {
		return 2
	}
	return 1
}
