package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Injected at build time via -ldflags.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("agencycli %s\n", version)
			fmt.Printf("  commit : %s\n", commit)
			fmt.Printf("  built  : %s\n", buildDate)
			fmt.Printf("  go     : %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}
