package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/sandbox"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newSandboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sandbox",
		Short: "Inspect and manage agent sandbox configuration",
	}
	cmd.AddCommand(
		newSandboxShowCmd(),
		newSandboxTestCmd(),
	)
	return cmd
}

// ── sandbox show ──────────────────────────────────────────────────────────────

func newSandboxShowCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show sandbox config and the generated docker run command for an agent",
		Example: `  agencycli sandbox show --project cc-connect --agent dev-claude`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}

			s := store.NewFS(root)
			meta, err := s.AgentMeta(project, agentName)
			if err != nil {
				return err
			}
			agentDir := s.AgentDir(project, agentName)

			fmt.Printf("Agent   : %s/%s\n", project, agentName)
			fmt.Printf("Model   : %s\n", meta.Model)

			if meta.Sandbox == nil || meta.Sandbox.Provider == entity.SandboxNone {
				fmt.Printf("Sandbox : none (agent runs directly on host)\n\n")
				fmt.Printf("To enable:\n")
				fmt.Printf("  agencycli hire --project %s --team %s --model %s --name %s --sandbox docker --force\n",
					project, meta.Team, meta.Model, agentName)
				return nil
			}

			fmt.Printf("Sandbox : %s\n", meta.Sandbox.Provider)

			if meta.Sandbox.Provider != entity.SandboxDocker {
				return nil
			}

			dockerCfg := meta.Sandbox.Docker

			image := sandbox.ImageForModel(meta.Model)
			if dockerCfg != nil && dockerCfg.Image != "" {
				image = dockerCfg.Image
			}
			network := "bridge"
			if dockerCfg != nil && dockerCfg.NetworkMode != "" {
				network = dockerCfg.NetworkMode
			}
			memMB := sandbox.DefaultMemoryMB
			if dockerCfg != nil && dockerCfg.MemoryMB > 0 {
				memMB = dockerCfg.MemoryMB
			}

			fmt.Printf("Image   : %s\n", image)
			fmt.Printf("Network : %s\n", network)
			fmt.Printf("Memory  : %d MiB\n", memMB)

			// Repo mount (auto-detected from project.yaml)
			repoMount := sbxResolveRepoMount(s, root, project)
			if repoMount != "" {
				fmt.Printf("\nRepo mount (auto, same path as host):\n")
				fmt.Printf("  -v %s\n", repoMount)
				fmt.Printf("  Agent can read/write/commit at: %s\n", strings.SplitN(repoMount, ":", 2)[0])
			} else {
				fmt.Printf("\nRepo mount : none (project has no repo configured in project.yaml)\n")
			}

			// Credential mounts
			mounts := sandbox.ResolveCredentialMounts(meta.Model, dockerCfg)
			if len(mounts) > 0 {
				fmt.Printf("\nCredential mounts (read-only, skipped if path not found on host):\n")
				for _, m := range mounts {
					fmt.Printf("  %s\n", sbxExpandHome(m))
				}
			}

			// API key env vars
			envKeys := sandbox.WellKnownEnvKeys(meta.Model)
			var apiKeys []string
			for _, k := range envKeys {
				if !sbxIsProxyKey(k) {
					apiKeys = append(apiKeys, k)
				}
			}
			if len(apiKeys) > 0 {
				fmt.Printf("\nAPI keys forwarded from host (value hidden, use -e KEY form):\n")
				for _, k := range apiKeys {
					set := "not set on host"
					if os.Getenv(k) != "" {
						set = "✓ set"
					}
					fmt.Printf("  %-40s %s\n", k, set)
				}
			}

			// Workspace root mount (auto, enables inter-agent task assignment)
			wsMount := root + ":" + root
			fmt.Printf("\nWorkspace root mount (auto, enables inter-agent coordination):\n")
			fmt.Printf("  -v %s\n", wsMount)

			// agencycli binary mount (auto, so agents can run agencycli inside container)
			agencycliBin, _ := os.Executable()
			if real, err2 := filepath.EvalSymlinks(agencycliBin); err2 == nil {
				agencycliBin = real
			}
			fmt.Printf("\nAgencycli binary mount (auto, enables `agencycli` inside container):\n")
			fmt.Printf("  -v %s:%s:ro\n", agencycliBin, sandbox.AgencycliMount)

			// Preview the actual docker run command (include all auto mounts)
			previewCfg := sbxCloneDockerCfg(dockerCfg)
			if repoMount != "" {
				previewCfg.ExtraVolumes = append(previewCfg.ExtraVolumes, repoMount)
			}
			previewCfg.ExtraVolumes = append(previewCfg.ExtraVolumes, wsMount)
			if agencycliBin != "" {
				previewCfg.ExtraVolumes = append(previewCfg.ExtraVolumes,
					agencycliBin+":"+sandbox.AgencycliMount+":ro")
			}
			innerArgs := agentInnerArgs(meta.Model)
			dockerArgs, err := sandbox.BuildArgs(agentDir, meta.Model, previewCfg, innerArgs)
			if err != nil {
				return fmt.Errorf("build docker args preview: %w", err)
			}
			fmt.Printf("\nGenerated docker run command (example):\n")
			fmt.Printf("  docker %s\n", sbxFormatArgs(dockerArgs))
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	return cmd
}

// ── sandbox test ──────────────────────────────────────────────────────────────

func newSandboxTestCmd() *cobra.Command {
	var (
		project   string
		agentName string
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Verify the sandbox works by running 'echo ok' inside the container",
		Example: `  agencycli sandbox test --project cc-connect --agent dev-claude`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}

			s := store.NewFS(root)
			meta, err := s.AgentMeta(project, agentName)
			if err != nil {
				return err
			}

			if meta.Sandbox == nil || meta.Sandbox.Provider == entity.SandboxNone {
				fmt.Printf("Agent %s/%s has no sandbox configured.\n", project, agentName)
				return nil
			}

			if err := sandbox.CheckDocker(); err != nil {
				return err
			}

			agentDir := s.AgentDir(project, agentName)
			dockerCfg := meta.Sandbox.Docker

			image := sandbox.ImageForModel(meta.Model)
			if dockerCfg != nil && dockerCfg.Image != "" {
				image = dockerCfg.Image
			}

			fmt.Printf("Testing sandbox for %s/%s ...\n", project, agentName)
			fmt.Printf("Image: %s\n\n", image)

			// Pull image (shows progress; no-op if already cached).
			fmt.Println("Pulling image (skipped if already cached)...")
			if err := sandbox.PullImage(image); err != nil {
				fmt.Printf("warning: pull failed (image may not exist yet): %v\n", err)
			}

			// Run a quick echo test — verifies workspace mount works.
			testArgs := []string{"echo", "agencycli sandbox test: OK"}
			_, finalArgs, err := sandbox.RunArgs(agentDir, meta.Model, dockerCfg, testArgs)
			if err != nil {
				return err
			}

			testCmd := exec.Command("docker", finalArgs...)
			testCmd.Stdout = os.Stdout
			testCmd.Stderr = os.Stderr
			if err := testCmd.Run(); err != nil {
				return fmt.Errorf("sandbox test failed: %w", err)
			}

			fmt.Printf("\n✓ Sandbox test passed for %s/%s\n", project, agentName)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name")
	return cmd
}

// ── helpers ───────────────────────────────────────────────────────────────────

// agentInnerArgs returns a representative inner command for display purposes.
func agentInnerArgs(model entity.AgentModel) []string {
	switch entity.NormaliseModel(model) {
	case entity.ModelClaudeCode:
		return []string{"claude", "--no-interactive", "--output-format", "stream-json",
			"--print-file", "/workspace/.prompt-example.txt"}
	case entity.ModelCodex, entity.ModelQoder:
		return []string{"codex", "-q", "--full-auto", "--input-file", "/workspace/.prompt-example.txt"}
	case entity.ModelGemini:
		return []string{"gemini", "--yolo", "--prompt-file", "/workspace/.prompt-example.txt"}
	case entity.ModelCursor:
		// Cursor CLI binary is `agent` (installed via curl https://cursor.com/install)
		return []string{"agent", "-p", "--force", "--output-format", "stream-json",
			"--print-file", "/workspace/.prompt-example.txt"}
	case entity.ModelOpenCode:
		return []string{"opencode", "run", "--file", "/workspace/.prompt-example.txt"}
	default:
		return []string{"sh", "-c", "cat /workspace/.prompt-example.txt"}
	}
}

// sbxFormatArgs pretty-prints docker args with line continuations.
func sbxFormatArgs(args []string) string {
	var sb strings.Builder
	i := 0
	lineLen := 0
	for i < len(args) {
		a := args[i]
		// Flags that take a value argument: print together on one line.
		if (a == "-v" || a == "-e" || a == "--memory" || a == "--network" ||
			a == "--cpus" || a == "-w") && i+1 < len(args) {
			chunk := a + " " + args[i+1]
			if lineLen > 0 {
				sb.WriteString(" \\\n    ")
				lineLen = 4
			}
			sb.WriteString(chunk)
			lineLen += len(chunk)
			i += 2
		} else {
			if lineLen > 60 {
				sb.WriteString(" \\\n    ")
				lineLen = 4
			} else if lineLen > 0 {
				sb.WriteString(" ")
				lineLen++
			}
			sb.WriteString(a)
			lineLen += len(a)
			i++
		}
	}
	return sb.String()
}

func sbxExpandHome(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, "~/") {
		return home + path[1:]
	}
	return path
}

func sbxIsProxyKey(k string) bool {
	k = strings.ToUpper(k)
	return k == "HTTPS_PROXY" || k == "HTTP_PROXY" || k == "NO_PROXY"
}

// sbxResolveRepoMount looks up the project's repo field and returns a
// "host_path:host_path" volume string for same-path container mounting.
func sbxResolveRepoMount(s store.Store, root, project string) string {
	proj, err := s.Project(project)
	if err != nil || proj.Repo == "" {
		return ""
	}
	repoPath := proj.Repo
	if !filepath.IsAbs(repoPath) {
		repoPath = filepath.Join(root, repoPath)
	}
	repoPath, err = filepath.Abs(repoPath)
	if err != nil {
		return ""
	}
	if _, err := os.Stat(repoPath); err != nil {
		return ""
	}
	return repoPath + ":" + repoPath
}

// sbxCloneDockerCfg returns a shallow copy so we can mutate ExtraVolumes.
func sbxCloneDockerCfg(cfg *entity.DockerSandboxConfig) *entity.DockerSandboxConfig {
	if cfg == nil {
		return &entity.DockerSandboxConfig{}
	}
	cp := *cfg
	cp.ExtraVolumes = append([]string(nil), cfg.ExtraVolumes...)
	cp.ExtraEnv = append([]string(nil), cfg.ExtraEnv...)
	return &cp
}
