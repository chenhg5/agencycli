# agencycli Sandbox Design

> Version: 0.1 Draft  
> Status: Research → Design Phase

---

## 1. The Problem

Running AI agents (`claude`, `codex`, `gemini` …) directly on your host machine has three serious issues:

| Issue | Consequence |
|-------|-------------|
| **Broad filesystem access** | Agent can read SSH keys, cloud credentials, `.env` files outside the project |
| **Unrestricted network** | Agent can exfiltrate data, hit unexpected endpoints, rack up API bills |
| **Root/sudo availability** | On many dev machines the agent can install packages, modify system files |
| **Permission prompts** | Agents like Claude Code require `--dangerously-skip-permissions` to run headlessly, which is unsafe on a shared or production host |

For agencycli's **scheduler-driven headless execution** (heartbeat loop), the agent must run fully autonomously — you can't manually answer permission prompts. That makes sandboxing a requirement, not an optional nicety.

---

## 2. Landscape Research

### 2.1 Docker Sandboxes (Official Docker feature, Nov 2025)

[Docker Sandboxes](https://docs.docker.com/ai/sandboxes/) runs agents in lightweight **microVMs** with a private Docker daemon inside each VM. This provides:

- Hardware-level isolation (Firecracker / similar)
- Automatic workspace mounting
- Network egress control
- Claude Code, Gemini CLI, Codex supported out of the box

```bash
docker sandbox run claude ~/my-project
docker sandbox run claude ~/my-project -- "Fix the login bug"
```

Claude Code launches with `--dangerously-skip-permissions` inside the sandbox by default, which is safe because the VM is isolated.

**Status**: Experimental, Docker Desktop required.  
**Best for**: One-shot interactive sessions; not yet ideal for headless scheduler loops.

---

### 2.2 sandbox-agent (Rivet, open-source, ~1k ⭐)

[rivet-dev/sandbox-agent](https://github.com/rivet-dev/sandbox-agent) — a lightweight **Rust HTTP server** that runs *inside* a sandbox and exposes a unified API for controlling coding agents. It normalises the fragmented protocols of different agents (JSONL, JSON-RPC, SSE) into a single event schema.

```
Your App ──(HTTP/SDK)──► Sandbox Agent Server ──► Claude/Codex/OpenCode process
                         (inside sandbox)
```

Supports:
- **Claude Code**, Codex, OpenCode, Amp, Cursor, Pi
- **Providers**: Local, Docker, E2B, Daytona, Vercel Sandboxes
- TypeScript SDK; no Go SDK yet
- Session persistence, permission request handling

**Key events**: `item.started`, `permission.requested`, `question.requested`, `item.done`

**Status**: Active, production-ready on their own platform.  
**Best for**: TypeScript backends; Go integration would require HTTP calls or a thin wrapper.

---

### 2.3 E2B (open-source, self-hostable)

[e2b-dev/E2B](https://github.com/e2b-dev/e2b) — Firecracker **microVM** sandboxes. Extremely fast (~150ms boot), hardware-level isolation (each VM has its own kernel).

- Apache 2.0, self-hostable on GCP (AWS in progress)
- Python + JS/TS SDKs; **no Go SDK**
- Used in production by OpenCode cloud runner
- 200M+ sandboxes started, 88% Fortune 100 adoption

**Best for**: Cloud deployments, multi-tenant security requirements, untrusted code.  
**Limitation**: Requires cloud infrastructure or KVM-capable host to self-host.

---

### 2.4 Daytona (open-source)

Docker container-based workspaces. Persistent, stateful (unlike E2B's ephemeral VMs). Faster startup (~90ms), but shares host kernel.

**Best for**: Persistent dev environments where the agent needs to keep state between sessions.

---

### 2.5 Devcontainers (VS Code spec)

Standard `.devcontainer/devcontainer.json` + Dockerfile. The entire dev environment runs inside Docker. Agents (Cursor, Claude Code) run in the container alongside the IDE.

```
Container:
  ├── workspace files (mounted)
  ├── claude / codex installed
  └── no access to host SSH keys, credentials
```

Network and filesystem restrictions set via Docker `--cap-drop` and volume mount controls.

**Best for**: Developer laptop usage where security is important but you still want IDE integration.

---

### 2.6 claude-code-docker (community)

[yury-egorenkov/claude-code-docker](https://github.com/yury-egorenkov/claude-code-docker) — opinionated Docker setup:

- iptables egress rules: only GitHub, npm, Anthropic APIs, and essential services allowed
- Persistent state: `~/.claude` mounted from host (credentials, conversation history)
- Pre-installed: Node 20, Go, Zsh, Git, GitHub CLI

**Best for**: Quickly sandboxing Claude Code with sensible defaults on a single machine.

---

## 3. Comparison Matrix

| Solution | Isolation | Persistent state | Headless-friendly | Go-native | Self-host | Complexity |
|----------|-----------|-----------------|-------------------|-----------|-----------|------------|
| **Docker (plain)** | Medium (container) | Yes | ✓ | ✓ | ✓ | Low |
| **Docker Sandboxes** | High (microVM) | Partial | ✓ | Docker CLI | Needs Docker Desktop | Low |
| **sandbox-agent** | Pluggable | Via provider | ✓ | HTTP only | ✓ | Medium |
| **E2B** | Very high (KVM) | No (ephemeral) | ✓ | No SDK | Infra required | High |
| **Daytona** | Medium (container) | Yes | ✓ | No SDK | ✓ | Medium |
| **Devcontainer** | Medium (container) | Yes | Partial | Docker CLI | ✓ | Low |

---

## 4. agencycli Sandbox Design

### 4.1 Design Goals

1. **Default: local** — no forced infrastructure; agents run directly for simple/trusted setups
2. **Docker: first sandbox tier** — easy, works everywhere, no cloud needed
3. **Pluggable providers** — Docker is the first; E2B/Daytona can follow
4. **Per-agency, per-team, per-agent override** — sandbox config inherits down the hierarchy
5. **Transparent to agents** — agents don't know they're in a sandbox; their working directory is mounted at the same path

### 4.2 SandboxConfig Data Model

```go
// SandboxProvider identifies the sandbox backend.
type SandboxProvider string

const (
    SandboxNone       SandboxProvider = ""        // run locally (default)
    SandboxDocker     SandboxProvider = "docker"
    SandboxE2B        SandboxProvider = "e2b"
    SandboxDaytona    SandboxProvider = "daytona"
)

// SandboxConfig describes how to sandbox an agent execution.
// Resolved at hire/run time with agency → team → agent override priority.
type SandboxConfig struct {
    Provider SandboxProvider `yaml:"provider,omitempty"`

    // Docker options
    Docker *DockerSandboxConfig `yaml:"docker,omitempty"`
}

type DockerSandboxConfig struct {
    // Image to use. Defaults to a sensible per-model image.
    // e.g. "ghcr.io/agencycli/sandbox-claudecode:latest"
    Image string `yaml:"image,omitempty"`

    // NetworkMode: "bridge" (default, internet access),
    //              "none" (fully offline),
    //              "host" (unsafe, for debugging only)
    NetworkMode string `yaml:"network_mode,omitempty"`

    // ExtraVolumes mounts additional host paths into the container.
    // Format: "host/path:container/path[:ro]"
    ExtraVolumes []string `yaml:"extra_volumes,omitempty"`

    // ExtraEnv passes additional environment variables into the container.
    ExtraEnv []string `yaml:"extra_env,omitempty"`

    // MemoryMB limits container memory (0 = no limit).
    MemoryMB int `yaml:"memory_mb,omitempty"`

    // CPUs limits CPU quota (0.0 = no limit).
    CPUs float64 `yaml:"cpus,omitempty"`

    // CredentialsMount mounts agent credential files from the host.
    // e.g. ~/.claude, ~/.config/gh — so agents don't need to re-auth inside sandbox.
    CredentialsMount []string `yaml:"credentials_mount,omitempty"`
}
```

### 4.3 Configuration Hierarchy

Sandbox config lives in three places (lower = higher priority):

```yaml
# agency-level default: .agencycli/agency.yaml
sandbox:
  provider: docker
  docker:
    network_mode: bridge
    memory_mb: 4096

# team-level override: teams/engineering/team.yaml
sandbox:
  docker:
    memory_mb: 8192   # engineering needs more memory

# agent-level override: .agencycli-agent.yaml
sandbox:
  docker:
    image: "ghcr.io/agencycli/sandbox-claudecode:latest"
    credentials_mount:
      - "~/.claude:/root/.claude"
      - "~/.config/gh:/root/.config/gh"
```

At hire/run time, agencycli merges: agency defaults ← team override ← agent override.

### 4.4 Docker Sandbox: How It Works

When `sandbox.provider = docker`, `agencycli run` replaces the bare `exec.Command` invocation with:

```
docker run --rm \
  -v <agent-working-dir>:/workspace \
  -v ~/.claude:/root/.claude:ro \       ← credentials
  -w /workspace \
  -e ANTHROPIC_API_KEY=... \
  --memory=4g --cpus=2 \
  ghcr.io/agencycli/sandbox-claudecode:latest \
  claude --no-interactive --output-format stream-json \
         --resume <session-id> \
         --print-file /workspace/.prompt-xxx.txt
```

Key properties:
- **Agent working directory** (`CLAUDE.md`, skills, tasks, etc.) is bind-mounted into the container at `/workspace`
- **Credentials** are mounted read-only so the agent can authenticate without host exposure
- **The container is ephemeral** (`--rm`); state persists only in the mounted workspace
- **Session IDs** are captured from stdout and stored in `heartbeat.yaml` — next heartbeat resumes the same conversation

### 4.5 Pre-built Sandbox Images

We will provide minimal base images per model:

| Image | Base | Pre-installed |
|-------|------|---------------|
| `agencycli/sandbox-claudecode` | `debian:12-slim` | `claude` CLI, `gh`, `git`, `curl`, common dev tools |
| `agencycli/sandbox-codex` | `node:22-slim` | `codex` CLI, `git`, `gh` |
| `agencycli/sandbox-gemini` | `debian:12-slim` | `gemini` CLI, `git` |
| `agencycli/sandbox-generic` | `debian:12-slim` | `git`, `curl`, `jq`, build essentials |

Users can extend these or provide their own image with a custom `Dockerfile`.

### 4.6 Credential Mounting

Different agents store credentials in different locations:

| Agent | Credentials path |
|-------|-----------------|
| Claude Code | `~/.claude/` |
| GitHub CLI | `~/.config/gh/` |
| Codex | `~/.codex/` |
| Gemini | `~/.config/gemini/` |
| SSH (for git push) | `~/.ssh/` |

The `credentials_mount` list in `DockerSandboxConfig` handles these. A safe default:

```yaml
credentials_mount:
  - "~/.claude:/root/.claude:ro"
  - "~/.config/gh:/root/.config/gh:ro"
  - "~/.ssh:/root/.ssh:ro"
```

These are mounted **read-only** to prevent the container from modifying host credentials.

---

## 5. CLI Changes

### 5.1 `agencycli hire` (sandbox option)

```bash
agencycli hire \
  --project cc-connect \
  --team engineering \
  --model claudecode \
  --name dev-claude \
  --sandbox docker \
  --sandbox-image "ghcr.io/agencycli/sandbox-claudecode:latest" \
  --sandbox-memory 4096
```

Sandbox config is written to `.agencycli-agent.yaml`:

```yaml
sandbox:
  provider: docker
  docker:
    image: ghcr.io/agencycli/sandbox-claudecode:latest
    memory_mb: 4096
    credentials_mount:
      - "~/.claude:/root/.claude:ro"
      - "~/.config/gh:/root/.config/gh:ro"
```

### 5.2 `agencycli sandbox` subcommand

```bash
# Set default sandbox for the whole agency
agencycli sandbox set --provider docker --memory 4096

# Test that docker sandbox works for an agent
agencycli sandbox test --project cc-connect --agent dev-claude

# Show resolved sandbox config for an agent
agencycli sandbox show --project cc-connect --agent dev-claude
```

---

## 6. Implementation Plan

### Phase 1 — Docker sandbox (local, next milestone)

1. Add `SandboxConfig` / `DockerSandboxConfig` to `entity/types.go`
2. Update `AgentMeta` to include `Sandbox *entity.SandboxConfig`
3. Update `hire` command to accept `--sandbox` flags and write config
4. Add `internal/sandbox/docker.go` — `Run(agentDir, model, args []string, cfg *entity.DockerSandboxConfig) error`
5. Update `internal/runner/runner.go` — if `meta.Sandbox.Provider == docker`, use `sandbox.DockerRun` instead of bare `exec.Command`
6. Publish `agencycli/sandbox-claudecode` and `agencycli/sandbox-codex` base images

### Phase 2 — sandbox subcommand + devcontainer support

1. `agencycli sandbox` subcommand (`set`, `test`, `show`)
2. `agencycli sandbox devcontainer` — generate `.devcontainer/` for an agent working directory
3. Network egress policy (`--network allow-list github.com,api.anthropic.com`)

### Phase 3 — E2B / Daytona support

1. Abstract `sandbox.Runner` interface (local / docker / e2b / daytona)
2. E2B provider: Go HTTP client calling E2B REST API
3. Daytona provider: similar

---

## 7. Immediate Recommendation

For the next release, add **Docker sandbox as opt-in**:

```bash
# Hire with sandbox
agencycli hire --project cc-connect --team qa --model claudecode \
  --name qa-reviewer --sandbox docker

# Then the scheduler/run command automatically uses docker run
agencycli run --project cc-connect --agent qa-reviewer
```

This covers the core security concern (host isolation) with minimal complexity, uses technology everyone already has (Docker), and is transparent to agents — they don't know they're containerised.

The `--dangerously-skip-permissions` flag for Claude Code is safe inside a Docker container because the blast radius is limited to the mounted workspace.
