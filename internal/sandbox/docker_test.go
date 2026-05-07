package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhg5/agencycli/internal/entity"
)

func TestBuildArgsBinPATHKeepsToolchainPaths(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "projects", "demo", "agents", "dev")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("create agent dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}

	args, err := BuildArgs(agentDir, entity.ModelCodex, nil, []string{"codex", "exec", "-"})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}

	pathEnv := findEnvArg(args, "PATH=")
	if pathEnv == "" {
		t.Fatalf("PATH env not found in args: %v", args)
	}
	for _, want := range []string{AgencycliBin, "/usr/local/go/bin", "/root/go/bin", "/usr/local/bin"} {
		if !strings.Contains(pathEnv, want) {
			t.Fatalf("PATH %q missing %s", pathEnv, want)
		}
	}
}

func findEnvArg(args []string, prefix string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-e" && strings.HasPrefix(args[i+1], prefix) {
			return args[i+1]
		}
	}
	return ""
}
