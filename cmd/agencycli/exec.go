package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/runner"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	var (
		project   string
		agentName string
		prompt    string
		file      string
		sessionID string
		noSession bool
	)

	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Run a prompt against an agent directly (bypasses task queue)",
		Long: `Run a raw prompt against an agent immediately, without creating a task.

Useful for quick interactive testing or one-off commands.
Output is streamed to stdout in real time and also written to a log file.
The session ID is saved automatically so the next exec resumes the same
conversation (use --no-session to start fresh).

Examples:

  # Inline prompt
  agencycli exec --project cc-connect --agent pm \
    --prompt "List all open GitHub issues and summarize them"

  # From a file
  agencycli exec --project cc-connect --agent dev-claude --file task.txt

  # From stdin (pipe)
  echo "What is 1+1?" | agencycli exec --project cc-connect --agent pm

  # From stdin (explicit)
  agencycli exec --project cc-connect --agent pm - <<'EOF'
  Check the latest PRs.
  EOF

  # Resume a specific session
  agencycli exec --project cc-connect --agent pm \
    --prompt "Prioritize those bugs" --session abc123

  # Start a fresh conversation (ignore saved session)
  agencycli exec --project cc-connect --agent pm \
    --prompt "Start over" --no-session`,
		Args: cobra.MaximumNArgs(1), // optional "-" for explicit stdin
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}

			promptText, err := resolveExecPrompt(prompt, file, args)
			if err != nil {
				return err
			}
			if strings.TrimSpace(promptText) == "" {
				return fmt.Errorf("prompt is empty — use --prompt TEXT, --file PATH, or pipe via stdin")
			}

			ts := taskstore.New(root)

			// Resolve session: --session flag > saved heartbeat session > "".
			sid := sessionID
			if !noSession && sid == "" {
				if hb, err := ts.GetHeartbeat(project, agentName); err == nil && hb.SessionID != "" {
					sid = hb.SessionID
					fmt.Fprintf(os.Stderr, "↩  resuming session %s  (--no-session to start fresh)\n\n", sid)
				}
			}

			as := store.NewFS(root)
			r := runner.New(root, ts, as)

			fmt.Fprintf(os.Stderr, "▶  exec %s/%s\n\n", project, agentName)

			result, err := r.ExecPrompt(project, agentName, promptText, sid)
			if err != nil {
				return fmt.Errorf("exec: %w", err)
			}

			fmt.Fprintf(os.Stderr, "\n── exec complete ─────────────────────────────────\n")
			fmt.Fprintf(os.Stderr, "status  : %s\n", result.Status)
			fmt.Fprintf(os.Stderr, "log     : %s\n", result.LogPath)
			if result.SessionID != "" {
				fmt.Fprintf(os.Stderr, "session : %s\n", result.SessionID)
				// Auto-save so the next exec resumes this conversation.
				if !noSession {
					if hb, err2 := ts.GetHeartbeat(project, agentName); err2 == nil {
						hb.SessionID = result.SessionID
						if err2 = ts.SaveHeartbeat(project, agentName, hb); err2 == nil {
							fmt.Fprintf(os.Stderr, "         (saved — next exec resumes automatically)\n")
						}
					}
				}
			}
			if result.ErrorMsg != "" {
				fmt.Fprintf(os.Stderr, "error   : %s\n", result.ErrorMsg)
			}
			fmt.Fprintf(os.Stderr, "──────────────────────────────────────────────────\n")

			if result.Status == entity.TaskStatusDoneFailed {
				return fmt.Errorf("agent exited with error")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "prompt text (inline)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "read prompt from file")
	cmd.Flags().StringVar(&sessionID, "session", "", "session ID to resume (overrides saved session)")
	cmd.Flags().BoolVar(&noSession, "no-session", false, "ignore saved session; start a fresh conversation")
	return cmd
}

// resolveExecPrompt returns the prompt string from the first available source:
// --prompt > --file > explicit "-" arg > piped stdin.
func resolveExecPrompt(promptFlag, fileFlag string, args []string) (string, error) {
	if promptFlag != "" {
		return promptFlag, nil
	}
	if fileFlag != "" {
		b, err := os.ReadFile(fileFlag)
		if err != nil {
			return "", fmt.Errorf("read --file: %w", err)
		}
		return string(b), nil
	}
	// Explicit "-" positional arg → read stdin.
	if len(args) == 1 && args[0] == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return string(b), nil
	}
	// Auto-detect piped stdin (stdin is not a terminal).
	if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return string(b), nil
	}
	return "", nil
}
