package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agentorg",
	Short: "AI agent organisation and context management",
	Long: `agentorg manages the organisational context for AI agents.

It lets you create a company with departments and projects, then hire
AI agents into projects with the right accumulated context so they can
start working immediately.

Typical workflow:

  agentorg create company --name "MyCompany"
  cd MyCompany

  agentorg create dept --name "engineering"
  agentorg create dept --name "engineering/backend" --desc "Go/gRPC services"

  agentorg create project --name "my-api" --repo "../my-api"

  agentorg hire --project "my-api" --dept "engineering/backend" \
                --model "claude-code" --name "dev"

  cd projects/my-api/agents/dev
  claude`,
}

func init() {
	rootCmd.AddCommand(
		newCreateCmd(),
		newHireCmd(),
		newSyncCmd(),
		newListCmd(),
		newShowCmd(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
