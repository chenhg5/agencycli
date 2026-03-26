package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/ctxbuild"
	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/formatter"
	"github.com/chenhg5/agencycli/internal/sandbox"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newHireCmd() *cobra.Command {
	cmd := buildHireCmd("hire")
	// assign is a natural-language alias for hire
	assignCmd := buildHireCmd("assign")
	assignCmd.Short = "Assign an agent to a project (alias for hire)"
	assignCmd.Hidden = false

	// Register assign as a sibling at the root level via root.go init,
	// but return the primary hire command here. assign is added in root.go.
	_ = assignCmd
	return cmd
}

func newAssignCmd() *cobra.Command {
	cmd := buildHireCmd("assign")
	cmd.Short = "Assign an agent to a project (alias for hire)"
	return cmd
}

func buildHireCmd(use string) *cobra.Command {
	var (
		project     string
		team        string
		role        string
		model       string
		agentName   string
		extraPrompt string
		force       bool

		// Sandbox flags
		sandboxProvider    string
		sandboxImage       string
		sandboxNetwork     string
		sandboxMemoryMB    int
		sandboxCPUs        float64
		sandboxNoAutoCreds bool

		// HTTP agent flags (used when --model http-agent)
		httpURL     string
		httpModel   string
		httpAPIKey  string
		httpTimeout string
		httpStream  bool
		httpHeaders []string
	)

	cmd := &cobra.Command{
		Use:   use,
		Short: "Hire an agent for a project (merges context and creates the agent working directory)",
		Long: `hire (or assign) assembles the full context for a (project, team[, role]) tuple and
writes it into an agent working directory under projects/<project>/agents/<name>/.

Context layers are merged in this order:
  1. Agency
  2. Team chain (from top-level to the specified team)
  3. Role (optional — provides extra prompt, skills, and workspace setup)
  4. Project

The output format depends on --model:
  claudecode   →  CLAUDE.md + .agencycli-context/ + .claude/skills/
  codex        →  AGENTS.md (single merged file)
  cursor       →  .cursorrules + .cursor/rules/agencycli.mdc
  gemini       →  GEMINI.md + .agencycli-context/ + .gemini/skills/
  generic-cli  →  context.md (plain text)

Claude Code (claudecode): the claude subprocess inherits your shell environment on the host.
With Docker sandbox, credentials and these provider overrides are forwarded from the host when set:
  ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN), ANTHROPIC_BASE_URL, ANTHROPIC_MODEL

To switch an existing agent to another runtime (e.g. claudecode → codex), use
  agencycli agent set-model (see agencycli agent set-model --help).`,
		Example: `  # Hire with a role (recommended)
  agencycli hire --project "my-site" --team "growth" --role "content-writer" \
               --model "claudecode" --name "writer"

  # Hire without a role (role is optional)
  agencycli hire --project "my-api" --team "engineering/backend" \
               --model "claudecode" --name "dev"

  # Hire with Docker sandbox isolation
  agencycli hire --project "my-api" --team "engineering" --role "backend-dev" \
               --model "claudecode" --name "dev" --sandbox docker

  # assign is identical to hire
  agencycli assign --project "my-api" --team "engineering" --role "backend-dev" \
                --model "cursor" --name "cursor-dev"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" || team == "" || model == "" || agentName == "" {
				return fmt.Errorf("--project, --team, --model and --name are all required")
			}

			agentModel := entity.NormaliseModel(entity.AgentModel(model))
			if !entity.IsValidModel(agentModel) {
				return fmt.Errorf("unknown model %q (supported: %s)",
					model, joinModels(entity.KnownModels))
			}

			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			if _, err := s.Project(project); err != nil {
				return err
			}

			agentDir := s.AgentDir(project, agentName)
			if _, err := os.Stat(agentDir); err == nil && !force {
				return fmt.Errorf(
					"agent %q already exists at %s\n"+
						"Use --force to regenerate it",
					agentName, agentDir,
				)
			}

			// Human agents need no context files or sandbox — just create the dir and save meta.
			if agentModel == entity.ModelHuman {
				if err := os.MkdirAll(agentDir, 0o755); err != nil {
					return fmt.Errorf("%s: create agent dir: %w", use, err)
				}
				meta := &entity.AgentMeta{
					Name:    agentName,
					Project: project,
					Team:    team,
					Role:    role,
					Model:   agentModel,
					HiredAt: time.Now().UTC(),
				}
				if err := s.SaveAgentMeta(project, agentName, meta); err != nil {
					return fmt.Errorf("%s: save agent meta: %w", use, err)
				}
				fmt.Printf("✓ Human agent hired: %s/%s\n", project, agentName)
				return nil
			}

			builder := ctxbuild.NewBuilder(s)
			mc, err := builder.Build(project, team, role)
			if err != nil {
				return fmt.Errorf("%s: build context: %w", use, err)
			}

			if extraPrompt != "" {
				data, err := os.ReadFile(extraPrompt)
				if err != nil {
					return fmt.Errorf("%s: read extra prompt: %w", use, err)
				}
				mc.Layers = append(mc.Layers, ctxbuild.ContextLayer{
					Source:  "extra",
					Content: string(data),
				})
			}

			if err := os.MkdirAll(agentDir, 0o755); err != nil {
				return fmt.Errorf("%s: create agent dir: %w", use, err)
			}

			f, err := formatter.New(agentModel)
			if err != nil {
				return err
			}
			if err := f.Format(mc, agentDir); err != nil {
				return fmt.Errorf("%s: format context: %w", use, err)
			}

			// Build sandbox config if requested.
			var sandboxCfg *entity.SandboxConfig
			if sandboxProvider != "" {
				provider := entity.SandboxProvider(sandboxProvider)
				switch provider {
				case entity.SandboxDocker:
					// Verify docker is reachable now so we fail fast at hire time.
					if err := sandbox.CheckDocker(); err != nil {
						return err
					}
					dockerCfg := &entity.DockerSandboxConfig{
						Image:             sandboxImage,
						NetworkMode:       sandboxNetwork,
						MemoryMB:          sandboxMemoryMB,
						CPUs:              sandboxCPUs,
						NoAutoCredentials: sandboxNoAutoCreds,
					}
					sandboxCfg = &entity.SandboxConfig{
						Provider: entity.SandboxDocker,
						Docker:   dockerCfg,
					}
					if sandboxImage == "" {
						sandboxImage = sandbox.ImageForModel(agentModel)
					}
				default:
					return fmt.Errorf("unknown sandbox provider %q (supported: docker)", sandboxProvider)
				}
			}

			// Automatically include the project's code repository as an
			// additional working directory so the agent can read/write it
			// without leaving its context directory (e.g. --add-dir in claude).
			var addDirs []string
			if projMeta, err2 := s.Project(project); err2 == nil && projMeta.Repo != "" {
				repoAbs := projMeta.Repo
				if !filepath.IsAbs(repoAbs) {
					repoAbs = filepath.Join(root, repoAbs)
				}
				addDirs = []string{repoAbs}
			}

			// Run role workspace setup (create dirs/files) if a role was specified.
			if role != "" {
				roleMeta, err2 := s.Role(team, role)
				if err2 == nil {
					if err3 := applyRoleSetup(roleMeta.Setup, agentDir); err3 != nil {
						return fmt.Errorf("%s: role setup: %w", use, err3)
					}
				}
			}

			// Build HTTP agent config when --model http-agent is used.
			var httpAgentCfg *entity.HTTPAgentConfig
			if agentModel == entity.ModelHTTPAgent {
				if httpURL == "" {
					return fmt.Errorf("--http-url is required when --model http-agent")
				}
				httpAgentCfg = &entity.HTTPAgentConfig{
					URL:     httpURL,
					Model:   httpModel,
					APIKey:  httpAPIKey,
					Timeout: httpTimeout,
					Stream:  httpStream,
				}
				if len(httpHeaders) > 0 {
					httpAgentCfg.ExtraHeaders = make(map[string]string, len(httpHeaders))
					for _, h := range httpHeaders {
						k, v, ok := strings.Cut(h, ":")
						if !ok {
							return fmt.Errorf("--http-header %q: expected \"Key: Value\" format", h)
						}
						httpAgentCfg.ExtraHeaders[strings.TrimSpace(k)] = strings.TrimSpace(v)
					}
				}
			}

			meta := &entity.AgentMeta{
				Name:        agentName,
				Project:     project,
				Team:        team,
				Role:        role,
				Model:       agentModel,
				HiredAt:     time.Now().UTC(),
				ContextHash: ctxbuild.LayerHashes(mc),
				Sandbox:     sandboxCfg,
				AddDirs:     addDirs,
				HTTPAgent:   httpAgentCfg,
			}
			if err := s.SaveAgentMeta(project, agentName, meta); err != nil {
				return fmt.Errorf("%s: save agent meta: %w", use, err)
			}

			printHireSuccess(agentDir, agentModel, mc, project, agentName, sandboxCfg)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().StringVar(&team, "team", "", "Team path, e.g. \"engineering/backend\"")
	cmd.Flags().StringVar(&role, "role", "", "Role name within the team (optional, e.g. \"content-writer\")")
	cmd.Flags().StringVar(&model, "model", "", fmt.Sprintf("Agent model (%s)", joinModels(entity.KnownModels)))
	cmd.Flags().StringVar(&agentName, "name", "", "Name for this agent (used as directory name)")
	cmd.Flags().StringVar(&extraPrompt, "extra-prompt", "", "Path to an additional Markdown file to append to the context")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing agent directory")

	cmd.Flags().StringVar(&sandboxProvider, "sandbox", "", "Sandbox provider: docker (default: none, runs on host)")
	cmd.Flags().StringVar(&sandboxImage, "sandbox-image", "", "Docker image override (default: ghcr.io/agencycli/sandbox-<model>:latest)")
	cmd.Flags().StringVar(&sandboxNetwork, "sandbox-network", "bridge", "Docker network mode: bridge|none|host")
	cmd.Flags().IntVar(&sandboxMemoryMB, "sandbox-memory", 0, "Container memory limit in MiB (0 = no limit)")
	cmd.Flags().Float64Var(&sandboxCPUs, "sandbox-cpus", 0, "Container CPU quota (0 = no limit)")
	cmd.Flags().BoolVar(&sandboxNoAutoCreds, "sandbox-no-auto-creds", false, "Disable automatic credential mount defaults")

	// HTTP agent flags
	cmd.Flags().StringVar(&httpURL, "http-url", "", "HTTP endpoint URL for http-agent (e.g. http://localhost:11434/v1/chat/completions)")
	cmd.Flags().StringVar(&httpModel, "http-model", "", "Model identifier to pass in the request body (e.g. llama3.2, gpt-4o)")
	cmd.Flags().StringVar(&httpAPIKey, "http-api-key", "", "Bearer token for the HTTP endpoint (or set AGENCYCLI_HTTP_API_KEY env var)")
	cmd.Flags().StringVar(&httpTimeout, "http-timeout", "10m", "Per-request timeout for http-agent (e.g. 5m, 30m)")
	cmd.Flags().BoolVar(&httpStream, "http-stream", true, "Enable SSE streaming for http-agent responses")
	cmd.Flags().StringArrayVar(&httpHeaders, "http-header", nil, "Extra HTTP headers: \"Key: Value\" (repeatable)")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("team")
	_ = cmd.MarkFlagRequired("model")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func printHireSuccess(agentDir string, model entity.AgentModel, mc *ctxbuild.MergedContext, project, agentName string, sbx *entity.SandboxConfig) {
	fmt.Printf("✓ Agent workspace created: %s\n\n", agentDir)
	fmt.Printf("  Model:      %s\n", model)

	if sbx != nil && sbx.Provider != entity.SandboxNone {
		img := ""
		if sbx.Docker != nil && sbx.Docker.Image != "" {
			img = "  image=" + sbx.Docker.Image
		}
		fmt.Printf("  Sandbox:    %s%s\n", sbx.Provider, img)
	} else {
		fmt.Printf("  Sandbox:    none (runs on host)\n")
	}

	fmt.Printf("  Context layers merged:\n")
	for i, l := range mc.Layers {
		lines := strings.Count(l.Content, "\n") + 1
		fmt.Printf("    [%d] %-40s (%d lines)\n", i+1, l.Source, lines)
	}

	if len(mc.Skills) > 0 {
		fmt.Printf("  Skills installed:\n")
		for _, sk := range mc.Skills {
			fmt.Printf("    - %s\n", sk.Name)
		}
	}

	fmt.Printf("\n  To start working:\n")
	fmt.Printf("    cd %s\n", agentDir)

	if sbx != nil && sbx.Provider == entity.SandboxDocker {
		fmt.Printf("    agencycli run --project %s --agent %s\n", project, agentName)
		fmt.Printf("    # (agencycli run executes inside a Docker container)\n")
		return
	}

	switch model {
	case entity.ModelClaudeCode:
		fmt.Printf("    claude\n")
	case entity.ModelCodex:
		fmt.Printf("    codex\n")
	case entity.ModelCursor:
		fmt.Printf("    agent\n")
	case entity.ModelGemini:
		fmt.Printf("    gemini\n")
	case entity.ModelHTTPAgent:
		fmt.Printf("    agencycli run --project %s --agent %s\n", project, agentName)
		fmt.Printf("    # (prompts are sent to your configured HTTP endpoint)\n")
	case entity.ModelHuman:
		fmt.Printf("  Human identity — inbox at: projects/%s/agents/%s/.agencycli/\n", project, agentName)
	default:
		fmt.Printf("    <your-agent-command>\n")
	}
}

// applyRoleSetup creates the directories and files specified in a role's setup
// definition inside the agent's .agencycli/ directory.
func applyRoleSetup(setup entity.RoleSetup, agentDir string) error {
	// Role setup directories live inside .agencycli/ to keep system files consolidated.
	agencycliDir := filepath.Join(agentDir, ".agencycli")
	for _, dir := range setup.Dirs {
		full := filepath.Join(agencycliDir, dir)
		if err := os.MkdirAll(full, 0o755); err != nil {
			return fmt.Errorf("create dir %q: %w", dir, err)
		}
	}
	for _, f := range setup.Files {
		full := filepath.Join(agencycliDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("create parent for %q: %w", f.Path, err)
		}
		// Only create if it doesn't exist yet (don't overwrite user content).
		if _, err := os.Stat(full); os.IsNotExist(err) {
			if err := os.WriteFile(full, []byte(f.Content), 0o644); err != nil {
				return fmt.Errorf("create file %q: %w", f.Path, err)
			}
		}
	}
	return nil
}

func joinModels(models []entity.AgentModel) string {
	parts := make([]string, len(models))
	for i, m := range models {
		parts[i] = string(m)
	}
	return strings.Join(parts, "|")
}
