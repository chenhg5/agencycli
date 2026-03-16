package main

import (
	"fmt"
	"os"

	"github.com/agencycli/agencycli/internal/workspace"
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

It lets you create a company with departments and projects, then hire
AI agents into projects with the right accumulated context so they can
start working immediately.

Typical workflow:

  agencycli create company --name "MyCompany"
  cd MyCompany

  agencycli create dept --name "engineering"
  agencycli create dept --name "engineering/backend" --desc "Go/gRPC services"

  agencycli create project --name "my-api" --repo "../my-api"

  agencycli hire --project "my-api" --dept "engineering/backend" \
                --model "claudecode" --name "dev"

  cd projects/my-api/agents/dev
  claude

You can run any command from outside the workspace by passing --dir:

  agencycli --dir /path/to/MyCompany list depts`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&globalDir, "dir", "",
		"workspace directory (default: auto-discover from current directory)",
	)

	rootCmd.AddCommand(
		newCreateCmd(),
		newHireCmd(),
		newSyncCmd(),
		newListCmd(),
		newShowCmd(),
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
