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
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newAgentSetModelCmd() *cobra.Command {
	var (
		project     string
		agentName   string
		model       string
		httpURL     string
		httpModel   string
		httpAPIKey  string
		httpTimeout string
		httpStream  bool
		httpHeaders []string
	)

	cmd := &cobra.Command{
		Use:   "set-model",
		Short: "Switch an agent to a different runtime (e.g. claudecode → codex)",
		Long: `set-model updates the agent's model in .agencycli/agent.yaml, removes files
from the previous runtime (CLAUDE.md vs AGENTS.md, etc.), and regenerates
context for the new runtime. Team, role, hire time, playbook link, sandbox,
and add_dirs are preserved unless incompatible fields are cleared (http_agent
is removed when leaving http-agent).

Examples:
  agencycli agent set-model --project my-api --name dev --model codex
  agencycli agent set-model --project my-api --name bot --model http-agent \
    --http-url http://localhost:11434/v1/chat/completions --http-model llama3.2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" || agentName == "" || model == "" {
				return fmt.Errorf("--project, --name, and --model are required")
			}

			newModel := entity.NormaliseModel(entity.AgentModel(model))
			if !entity.IsValidModel(newModel) {
				return fmt.Errorf("unknown model %q (supported: %s)",
					model, joinModels(entity.KnownModels))
			}
			if newModel == entity.ModelHuman {
				return fmt.Errorf("use hire --force to replace a normal agent with a human agent")
			}

			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			meta, err := s.AgentMeta(project, agentName)
			if err != nil {
				return err
			}
			if meta.Model == entity.ModelHuman {
				return fmt.Errorf("human agents have no runtime model to change")
			}

			oldModel := entity.NormaliseModel(meta.Model)
			if oldModel == newModel {
				return fmt.Errorf("agent %s/%s already uses model %q", project, agentName, newModel)
			}

			if newModel == entity.ModelHTTPAgent && httpURL == "" {
				return fmt.Errorf("--http-url is required when switching to http-agent")
			}

			builder := ctxbuild.NewBuilder(s)
			mc, err := builder.Build(project, meta.Team, meta.Role)
			if err != nil {
				return fmt.Errorf("build context: %w", err)
			}

			if meta.Playbook == "" {
				if cfg, cerr := s.ProjectConfig(project); cerr == nil && cfg != nil {
					for _, spec := range cfg.Agents {
						if spec.Name == agentName && spec.Playbook != "" {
							meta.Playbook = spec.Playbook
							break
						}
					}
				}
			}

			var playbookData []byte
			if meta.Playbook != "" {
				playbookPath := filepath.Join(root, "project-blueprints", project, meta.Playbook)
				playbookData, _ = os.ReadFile(playbookPath)
			}

			newHashes := ctxbuild.LayerHashes(mc)
			if meta.Playbook != "" && len(playbookData) > 0 {
				newHashes["playbook:"+meta.Playbook] = ctxbuild.ContentHash(string(playbookData))
			}

			agentDir := s.AgentDir(project, agentName)
			if err := os.MkdirAll(agentDir, 0o755); err != nil {
				return err
			}

			// Drop previous runtime's files first (may remove stale .agencycli/context/*.md).
			if err := formatter.RemoveOutputsFromOtherModels(agentDir, newModel); err != nil {
				return err
			}

			if meta.Playbook != "" && len(playbookData) > 0 {
				ctxDir := filepath.Join(agentDir, ".agencycli", "context")
				if err := os.MkdirAll(ctxDir, 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(ctxDir, "wakeup.md"), playbookData, 0o644); err != nil {
					return fmt.Errorf("write wakeup.md: %w", err)
				}
			}

			f, err := formatter.New(newModel)
			if err != nil {
				return err
			}
			if err := f.Format(mc, agentDir); err != nil {
				return fmt.Errorf("format context: %w", err)
			}

			meta.Model = newModel
			meta.RunCommand = "" // custom invoker is model-specific; use defaults for the new runtime
			meta.ContextHash = newHashes
			now := time.Now().UTC()
			meta.SyncedAt = &now

			if newModel == entity.ModelHTTPAgent {
				cfg := &entity.HTTPAgentConfig{
					URL:     httpURL,
					Model:   httpModel,
					APIKey:  httpAPIKey,
					Timeout: httpTimeout,
					Stream:  httpStream,
				}
				if len(httpHeaders) > 0 {
					cfg.ExtraHeaders = make(map[string]string, len(httpHeaders))
					for _, h := range httpHeaders {
						k, v, ok := strings.Cut(h, ":")
						if !ok {
							return fmt.Errorf("--http-header %q: expected \"Key: Value\" format", h)
						}
						cfg.ExtraHeaders[strings.TrimSpace(k)] = strings.TrimSpace(v)
					}
				}
				meta.HTTPAgent = cfg
			} else {
				meta.HTTPAgent = nil
			}

			if err := s.SaveAgentMeta(project, agentName, meta); err != nil {
				return fmt.Errorf("save agent meta: %w", err)
			}

			if meta.Sandbox != nil && meta.Sandbox.Provider == entity.SandboxDocker &&
				meta.Sandbox.Docker != nil && meta.Sandbox.Docker.Image != "" {
				_, _ = fmt.Fprintf(os.Stderr,
					"Note: sandbox.docker.image is still %q — if runs fail, clear it in agent.yaml so the default image for %q is used, or set an image for the new model.\n",
					meta.Sandbox.Docker.Image, newModel)
			}

			fmt.Printf("✓ %s/%s: model %q → %q (context regenerated)\n", project, agentName, oldModel, newModel)
			fmt.Printf("  Directory: %s\n", agentDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().StringVar(&agentName, "name", "", "Agent name (directory name under projects/<project>/agents/)")
	cmd.Flags().StringVar(&model, "model", "", fmt.Sprintf("New agent model (%s)", joinModels(entity.KnownModels)))
	cmd.Flags().StringVar(&httpURL, "http-url", "", "HTTP chat-completions URL (required when --model http-agent)")
	cmd.Flags().StringVar(&httpModel, "http-model", "", "Model id in JSON body for http-agent")
	cmd.Flags().StringVar(&httpAPIKey, "http-api-key", "", "Bearer token for http-agent (or AGENCYCLI_HTTP_API_KEY)")
	cmd.Flags().StringVar(&httpTimeout, "http-timeout", "10m", "Per-request timeout for http-agent")
	cmd.Flags().BoolVar(&httpStream, "http-stream", true, "Enable SSE streaming for http-agent")
	cmd.Flags().StringArrayVar(&httpHeaders, "http-header", nil, `Extra HTTP headers "Key: Value" (repeatable, http-agent only)`)

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("model")

	return cmd
}
