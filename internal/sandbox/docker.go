// Package sandbox wraps agent CLI execution inside isolated Docker containers.
//
// Design:
//   - Only plain `docker run` is used (no Docker Desktop sandbox/microVM API).
//     This works on any OS with Docker installed: Linux, macOS, Windows.
//   - The agent working directory is bind-mounted into the container at
//     /workspace; the container process runs with that as its cwd.
//   - Credential files (e.g. ~/.claude, ~/.config/gh) are mounted read-only
//     so the agent can authenticate without copying secrets into the image.
//   - API keys are injected as environment variables.
//   - The container is removed after each run (--rm).
//   - All stdout/stderr is streamed to the caller via the provided io.Writer.
package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chenhg5/agencycli/internal/entity"
)

const (
	// WorkspaceMount is the path inside the container where the agent
	// working directory is mounted.
	WorkspaceMount = "/workspace"

	// DefaultMemoryMB is the container memory limit when none is specified.
	DefaultMemoryMB = 4096

	// Image registry prefix for agencycli-provided sandbox images.
	imagePrefix = "ghcr.io/agencycli"

	// AgencycliMount is where the agencycli binary is mounted inside the
	// container so that agents can run `agencycli task add` etc.
	AgencycliMount = "/usr/local/bin/agencycli"
)

// defaultImage returns the default Docker image for a given agent model.
// Users can override this via DockerSandboxConfig.Image.
var defaultImages = map[entity.AgentModel]string{
	entity.ModelClaudeCode: imagePrefix + "/sandbox-claudecode:latest",
	entity.ModelCodex:      imagePrefix + "/sandbox-codex:latest",
	entity.ModelGemini:     imagePrefix + "/sandbox-gemini:latest",
	entity.ModelOpenCode:   imagePrefix + "/sandbox-opencode:latest",
	entity.ModelCursor:     imagePrefix + "/sandbox-cursor:latest",
	entity.ModelQoder:      imagePrefix + "/sandbox-codex:latest", // Qoder uses same base as Codex
}

// defaultCredentialMounts returns the credential paths that should be mounted
// read-only for a given model.
// The host paths use ~ which is expanded at runtime.
var defaultCredentialMounts = map[entity.AgentModel][]string{
	entity.ModelClaudeCode: {
		// ~/.claude.json holds user state / onboarding read at every startup.
		// ~/.claude/ holds settings.json, sessions, skills, etc.
		"~/.claude.json:/root/.claude.json:ro",
		"~/.claude:/root/.claude:ro",
		"~/.config/gh:/root/.config/gh:ro",
		"~/.ssh:/root/.ssh:ro",
	},
	entity.ModelCodex: {
		"~/.codex:/root/.codex:ro",
		"~/.config/gh:/root/.config/gh:ro",
		"~/.ssh:/root/.ssh:ro",
	},
	entity.ModelGemini: {
		// Gemini CLI stores OAuth tokens in ~/.gemini/oauth_creds.json
		// and account info in ~/.gemini/google_accounts.json.
		"~/.gemini:/root/.gemini:ro",
		"~/.config/gh:/root/.config/gh:ro",
		"~/.ssh:/root/.ssh:ro",
	},
	entity.ModelOpenCode: {
		"~/.config/opencode:/root/.config/opencode:ro",
		"~/.config/gh:/root/.config/gh:ro",
		"~/.ssh:/root/.ssh:ro",
	},
	entity.ModelCursor: {
		"~/.cursor:/root/.cursor:ro",
		"~/.config/gh:/root/.config/gh:ro",
		"~/.ssh:/root/.ssh:ro",
	},
	entity.ModelQoder: {
		"~/.codex:/root/.codex:ro",
		"~/.config/gh:/root/.config/gh:ro",
		"~/.ssh:/root/.ssh:ro",
	},
}

// BuildArgs constructs the full `docker run` argument list for running an
// agent inside a container. The returned slice is ready to pass to exec.Command.
//
// agentDir is the agent's working directory on the host (will be mounted at /workspace).
// model is used to select defaults when cfg fields are empty.
// innerArgs are the agent CLI arguments to run inside the container.
func BuildArgs(agentDir string, model entity.AgentModel, cfg *entity.DockerSandboxConfig, innerArgs []string) ([]string, error) {
	args := []string{"run", "--rm", "-i"}

	// ── Image ────────────────────────────────────────────────────────────────
	image := resolveImage(model, cfg)

	// ── Memory / CPU limits ──────────────────────────────────────────────────
	memMB := DefaultMemoryMB
	if cfg != nil && cfg.MemoryMB > 0 {
		memMB = cfg.MemoryMB
	}
	args = append(args, fmt.Sprintf("--memory=%dm", memMB))

	if cfg != nil && cfg.CPUs > 0 {
		args = append(args, fmt.Sprintf("--cpus=%.2f", cfg.CPUs))
	}

	// ── Network ──────────────────────────────────────────────────────────────
	networkMode := "bridge"
	if cfg != nil && cfg.NetworkMode != "" {
		networkMode = cfg.NetworkMode
	}
	args = append(args, "--network="+networkMode)

	// ── Workspace mount ──────────────────────────────────────────────────────
	absAgentDir, err := filepath.Abs(agentDir)
	if err != nil {
		return nil, fmt.Errorf("sandbox: resolve agent dir: %w", err)
	}
	args = append(args,
		"-v", absAgentDir+":"+WorkspaceMount,
		"-w", WorkspaceMount,
	)

	// ── Credential mounts ────────────────────────────────────────────────────
	mounts := resolveCredentialMounts(model, cfg)
	for _, m := range mounts {
		expanded := expandTilde(m)
		// Only mount if the host path exists (skip silently if not set up).
		hostPath := strings.SplitN(expanded, ":", 2)[0]
		if _, err := os.Stat(hostPath); err == nil {
			args = append(args, "-v", expanded)
		}
	}

	// ── Extra volumes ────────────────────────────────────────────────────────
	if cfg != nil {
		for _, v := range cfg.ExtraVolumes {
			args = append(args, "-v", expandTilde(v))
		}
	}

	// ── Environment variables ────────────────────────────────────────────────
	// 1. Fixed sandbox env vars: always injected with explicit values because
	//    they must be set regardless of the host environment. These tell each
	//    agent CLI that it is running inside a sandbox and should not enforce
	//    extra permission prompts that require human interaction.
	for _, kv := range sandboxEnvVars(model) {
		args = append(args, "-e", kv)
	}

	// 2. API keys: use `-e KEY` (no value) so Docker inherits from the host.
	//    The token value never appears in the command line or process table.
	for _, envKey := range wellKnownEnvKeys(model) {
		if os.Getenv(envKey) != "" {
			args = append(args, "-e", envKey)
		}
	}

	// 3. ExtraEnv supports both "KEY" (inherit) and "KEY=VALUE" (explicit).
	if cfg != nil {
		for _, kv := range cfg.ExtraEnv {
			args = append(args, "-e", kv)
		}
	}

	// ── Image + inner command ────────────────────────────────────────────────
	args = append(args, image)
	args = append(args, innerArgs...)

	return args, nil
}

// RunArgs is a convenience wrapper: returns ("docker", BuildArgs(...)).
// Callers can pass this directly to exec.Command.
func RunArgs(agentDir string, model entity.AgentModel, cfg *entity.DockerSandboxConfig, innerArgs []string) (string, []string, error) {
	a, err := BuildArgs(agentDir, model, cfg, innerArgs)
	if err != nil {
		return "", nil, err
	}
	return "docker", a, nil
}

// CheckDocker verifies that the docker CLI is available and the daemon is reachable.
// Returns a user-friendly error if not.
func CheckDocker() error {
	cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker sandbox: cannot reach Docker daemon — is Docker running?\n" +
			"  Install Docker: https://docs.docker.com/get-docker/\n" +
			"  Start daemon:  sudo systemctl start docker  (Linux)\n" +
			"  Original error: " + err.Error())
	}
	return nil
}

// PullImage pulls a Docker image if it is not already present locally.
// This is a best-effort call; errors are non-fatal.
func PullImage(image string) error {
	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ImageForModel returns the default Docker image name for an agent model.
func ImageForModel(model entity.AgentModel) string {
	return resolveImage(model, nil)
}

// ── internal helpers ──────────────────────────────────────────────────────────

func resolveImage(model entity.AgentModel, cfg *entity.DockerSandboxConfig) string {
	if cfg != nil && cfg.Image != "" {
		return cfg.Image
	}
	model = entity.NormaliseModel(model)
	if img, ok := defaultImages[model]; ok {
		return img
	}
	return imagePrefix + "/sandbox-generic:latest"
}

// ResolveCredentialMounts is the exported form for use by CLI commands.
func ResolveCredentialMounts(model entity.AgentModel, cfg *entity.DockerSandboxConfig) []string {
	return resolveCredentialMounts(model, cfg)
}

// WellKnownEnvKeys is the exported form for use by CLI commands.
func WellKnownEnvKeys(model entity.AgentModel) []string {
	return wellKnownEnvKeys(model)
}

func resolveCredentialMounts(model entity.AgentModel, cfg *entity.DockerSandboxConfig) []string {
	if cfg != nil && cfg.NoAutoCredentials {
		return cfg.CredentialMounts
	}
	model = entity.NormaliseModel(model)
	defaults := defaultCredentialMounts[model]
	if cfg == nil || len(cfg.CredentialMounts) == 0 {
		return defaults
	}
	// Merge: start with defaults, append user-specified extras.
	seen := make(map[string]bool)
	var merged []string
	for _, m := range defaults {
		merged = append(merged, m)
		seen[mountKey(m)] = true
	}
	for _, m := range cfg.CredentialMounts {
		if !seen[mountKey(m)] {
			merged = append(merged, m)
		}
	}
	return merged
}

// mountKey extracts the container path (the key for deduplication).
func mountKey(mount string) string {
	parts := strings.SplitN(mount, ":", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return mount
}

// expandTilde replaces a leading ~ with the user's home directory.
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return home + path[1:]
}

// sandboxEnvVars returns environment variables that MUST be explicitly set
// (as KEY=VALUE) inside the sandbox container for each agent model to function
// correctly as root without interactive permission prompts.
//
// These are NOT inherited from the host — they are fixed values that tell the
// agent CLI it is already running in an isolated sandbox environment.
func sandboxEnvVars(model entity.AgentModel) []string {
	switch entity.NormaliseModel(model) {
	case entity.ModelClaudeCode:
		// IS_SANDBOX=1 is required to use --dangerously-skip-permissions as
		// root. Without it, Claude Code refuses to start with:
		//   "--dangerously-skip-permissions cannot be used with root/sudo"
		return []string{"IS_SANDBOX=1"}

	case entity.ModelCodex, entity.ModelQoder:
		// Codex CLI mandates its own sandboxing mechanism. Inside a Docker
		// container we are already sandboxed, so we disable the inner sandbox
		// requirement. Without this it fails with:
		//   "Sandbox was mandated, but no sandbox is available!"
		return []string{"CODEX_UNSAFE_ALLOW_NO_SANDBOX=1"}
	}
	return nil
}

// wellKnownEnvKeys returns the environment variable names an agent model needs
// to authenticate. These are forwarded from the host into the container using
// `-e KEY` (no value in the argument — Docker inherits from host env), so the
// token value never appears in the docker run command line or process table.
func wellKnownEnvKeys(model entity.AgentModel) []string {
	// Keys common to all models.
	common := []string{
		"HTTPS_PROXY", "HTTP_PROXY", "NO_PROXY", // honour proxy settings
	}
	var modelKeys []string
	switch entity.NormaliseModel(model) {
	case entity.ModelClaudeCode:
		modelKeys = []string{
			"ANTHROPIC_API_KEY",
			"ANTHROPIC_AUTH_TOKEN", // alternative auth used by some proxies
			"ANTHROPIC_BASE_URL",   // allow pointing at a proxy / local model
			"ANTHROPIC_MODEL",      // override default model
			"CLAUDE_CODE_USE_BEDROCK",
			"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION", // Bedrock auth
		}
	case entity.ModelCodex, entity.ModelQoder:
		modelKeys = []string{
			"OPENAI_API_KEY",
			"OPENAI_BASE_URL",
			"OPENAI_ORG_ID",
		}
	case entity.ModelGemini:
		modelKeys = []string{
			"GEMINI_API_KEY",
			"GOOGLE_API_KEY",
			"GOOGLE_APPLICATION_CREDENTIALS",
			"GOOGLE_CLOUD_PROJECT",
		}
	case entity.ModelOpenCode:
		// OpenCode supports multiple providers; pass all of them.
		modelKeys = []string{
			"ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL",
			"OPENAI_API_KEY", "OPENAI_BASE_URL",
			"GEMINI_API_KEY", "GOOGLE_API_KEY",
			"GROQ_API_KEY",
			"XAI_API_KEY",
		}
	case entity.ModelCursor:
		modelKeys = []string{
			"CURSOR_API_KEY",
		}
	case entity.ModelIFlow:
		modelKeys = []string{
			"IFLOW_API_KEY",
		}
	}
	return append(common, modelKeys...)
}
