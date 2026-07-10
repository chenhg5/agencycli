package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newProviderCmd is a stub placeholder. The provider sub-tree was referenced
// from root.go but never landed in this revision; this stub keeps the binary
// compilable until the full implementation is restored.
func newProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Manage external AI providers (stub — not yet implemented)",
		Long: `Manage external AI providers.

This command is a stub. The provider sub-tree is not yet implemented in this
build of agencycli. See the project roadmap for status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("provider command not yet implemented")
		},
	}
	return cmd
}
