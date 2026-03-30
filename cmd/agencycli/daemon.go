package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/daemon"
	"github.com/spf13/cobra"
)

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage agencycli as a system service",
		Long: `Install, start, stop, and manage agencycli as a background system service.

On Linux (systemd) and macOS (launchd), the daemon command sets up
agencycli start as a system/user service that auto-restarts on failure.`,
	}
	cmd.AddCommand(
		newDaemonInstallCmd(),
		newDaemonUninstallCmd(),
		newDaemonStartCmd(),
		newDaemonStopCmd(),
		newDaemonRestartCmd(),
		newDaemonStatusCmd(),
		newDaemonLogsCmd(),
	)
	return cmd
}

func newDaemonInstallCmd() *cobra.Command {
	var (
		logFile    string
		logMaxSize int
		workDir    string
		addr       string
		force      bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and start as system service",
		Example: `  agencycli daemon install
  agencycli daemon install --work-dir /path/to/workspace
  agencycli daemon install --addr 0.0.0.0:8080 --force`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := daemon.Config{
				WorkDir: workDir,
				LogFile: logFile,
				Addr:    addr,
			}
			if logMaxSize > 0 {
				cfg.LogMaxSize = int64(logMaxSize) * 1024 * 1024
			}

			if err := daemon.Resolve(&cfg); err != nil {
				return err
			}

			mgr, err := daemon.NewManager()
			if err != nil {
				return err
			}

			st, _ := mgr.Status()
			if st != nil && st.Installed && !force {
				return fmt.Errorf("service already installed. Use --force to reinstall")
			}

			if err := mgr.Install(cfg); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}

			if err := daemon.SaveMeta(&daemon.Meta{
				LogFile:     cfg.LogFile,
				LogMaxSize:  cfg.LogMaxSize,
				WorkDir:     cfg.WorkDir,
				BinaryPath:  cfg.BinaryPath,
				Addr:        cfg.Addr,
				InstalledAt: daemon.NowISO(),
			}); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save metadata: %v\n", err)
			}

			fmt.Println("agencycli daemon installed and started.")
			fmt.Println()
			fmt.Printf("  Platform:  %s\n", mgr.Platform())
			fmt.Printf("  Binary:    %s\n", cfg.BinaryPath)
			fmt.Printf("  WorkDir:   %s\n", cfg.WorkDir)
			fmt.Printf("  Addr:      %s\n", cfg.Addr)
			fmt.Printf("  Log:       %s\n", cfg.LogFile)
			fmt.Printf("  LogMax:    %d MB\n", cfg.LogMaxSize/1024/1024)
			fmt.Println()
			fmt.Println("Commands:")
			fmt.Println("  agencycli daemon status    - Check status")
			fmt.Println("  agencycli daemon logs -f   - Follow logs")
			fmt.Println("  agencycli daemon restart   - Restart")
			fmt.Println("  agencycli daemon stop      - Stop")
			fmt.Println("  agencycli daemon uninstall - Remove")
			return nil
		},
	}

	cmd.Flags().StringVar(&workDir, "work-dir", "", "workspace directory (default: current dir)")
	cmd.Flags().StringVar(&logFile, "log-file", "", "log file path (default: ~/.agencycli/logs/agencycli.log)")
	cmd.Flags().IntVar(&logMaxSize, "log-max-size", 10, "max log file size in MB")
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:27892", "listen address for the web console")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing installation")
	return cmd
}

func newDaemonUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove system service",
		RunE: func(_ *cobra.Command, _ []string) error {
			mgr, err := daemon.NewManager()
			if err != nil {
				return err
			}

			st, _ := mgr.Status()
			if st != nil && !st.Installed {
				fmt.Println("Service is not installed.")
				return nil
			}

			if err := mgr.Uninstall(); err != nil {
				return fmt.Errorf("uninstall failed: %w", err)
			}

			daemon.RemoveMeta()
			fmt.Println("agencycli daemon uninstalled.")
			return nil
		},
	}
}

func newDaemonStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the service",
		RunE: func(_ *cobra.Command, _ []string) error {
			mgr, err := daemon.NewManager()
			if err != nil {
				return err
			}
			requireInstalled(mgr)
			if err := mgr.Start(); err != nil {
				return fmt.Errorf("start failed: %w", err)
			}
			fmt.Println("agencycli daemon started.")
			return nil
		},
	}
}

func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the service",
		RunE: func(_ *cobra.Command, _ []string) error {
			mgr, err := daemon.NewManager()
			if err != nil {
				return err
			}
			requireInstalled(mgr)
			if err := mgr.Stop(); err != nil {
				return fmt.Errorf("stop failed: %w", err)
			}
			fmt.Println("agencycli daemon stopped.")
			return nil
		},
	}
}

func newDaemonRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the service",
		RunE: func(_ *cobra.Command, _ []string) error {
			mgr, err := daemon.NewManager()
			if err != nil {
				return err
			}
			requireInstalled(mgr)
			if err := mgr.Restart(); err != nil {
				return fmt.Errorf("restart failed: %w", err)
			}
			fmt.Println("agencycli daemon restarted.")
			return nil
		},
	}
}

func newDaemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show service status",
		RunE: func(_ *cobra.Command, _ []string) error {
			mgr, err := daemon.NewManager()
			if err != nil {
				return err
			}
			st, err := mgr.Status()
			if err != nil {
				return err
			}

			fmt.Println("agencycli daemon status")
			fmt.Println()

			if !st.Installed {
				fmt.Println("  Status:    Not installed")
				fmt.Printf("  Platform:  %s\n", st.Platform)
				fmt.Println()
				fmt.Println("  Run: agencycli daemon install")
				return nil
			}

			statusStr := "Stopped"
			if st.Running {
				statusStr = "Running"
			}
			fmt.Printf("  Status:    %s\n", statusStr)
			fmt.Printf("  Platform:  %s\n", st.Platform)
			if st.PID > 0 {
				fmt.Printf("  PID:       %d\n", st.PID)
			}

			if meta, err := daemon.LoadMeta(); err == nil {
				fmt.Printf("  Log:       %s\n", meta.LogFile)
				fmt.Printf("  WorkDir:   %s\n", meta.WorkDir)
				if meta.Addr != "" {
					fmt.Printf("  Addr:      %s\n", meta.Addr)
				}
				if t, err := time.Parse(time.RFC3339, meta.InstalledAt); err == nil {
					fmt.Printf("  Installed: %s\n", t.Format("2006-01-02 15:04:05"))
				}
			}
			return nil
		},
	}
}

func newDaemonLogsCmd() *cobra.Command {
	var (
		follow  bool
		lines   int
		logFile string
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View log output",
		RunE: func(_ *cobra.Command, _ []string) error {
			if logFile == "" {
				if meta, err := daemon.LoadMeta(); err == nil {
					logFile = meta.LogFile
				} else {
					logFile = daemon.DefaultLogFile()
				}
			}

			if _, err := os.Stat(logFile); err != nil {
				return fmt.Errorf("log file not found: %s", logFile)
			}

			printLastLines(logFile, lines)
			if follow {
				followFile(logFile)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output (like tail -f)")
	cmd.Flags().IntVarP(&lines, "lines", "n", 100, "number of lines to show")
	cmd.Flags().StringVar(&logFile, "log-file", "", "custom log file path")
	return cmd
}

func requireInstalled(mgr daemon.Manager) {
	st, _ := mgr.Status()
	if st == nil || !st.Installed {
		fmt.Fprintln(os.Stderr, "Service is not installed. Run first:")
		fmt.Fprintln(os.Stderr, "  agencycli daemon install --work-dir /path/to/workspace")
		os.Exit(1)
	}
}

func printLastLines(path string, n int) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading log: %v\n", err)
		return
	}

	allLines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	start := 0
	if len(allLines) > n {
		start = len(allLines) - n
	}
	for _, line := range allLines[start:] {
		fmt.Println(line)
	}
}

func followFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	_, _ = f.Seek(0, io.SeekEnd)
	reader := bufio.NewReader(f)

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			fmt.Print(line)
		}
		if err == io.EOF {
			time.Sleep(300 * time.Millisecond)
			reader.Reset(f)
			continue
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
	}
}

// daemonStatusJSON returns JSON-friendly status for the API.
func daemonStatusJSON() map[string]any {
	result := map[string]any{
		"supported": true,
	}

	mgr, err := daemon.NewManager()
	if err != nil {
		result["supported"] = false
		result["error"] = err.Error()
		return result
	}

	st, err := mgr.Status()
	if err != nil {
		result["error"] = err.Error()
		return result
	}

	result["installed"] = st.Installed
	result["running"] = st.Running
	result["platform"] = st.Platform
	if st.PID > 0 {
		result["pid"] = st.PID
	}

	if meta, err := daemon.LoadMeta(); err == nil {
		result["logFile"] = meta.LogFile
		result["workDir"] = meta.WorkDir
		result["addr"] = meta.Addr
		result["installedAt"] = meta.InstalledAt
	}
	return result
}

