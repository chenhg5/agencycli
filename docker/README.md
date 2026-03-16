# agencycli Sandbox Images

Pre-built Docker images for running AI agents in isolated containers.

## Available images

| Image | Agent | Base |
|-------|-------|------|
| `ghcr.io/agencycli/sandbox-claudecode:latest` | Claude Code (`claude`) | ubuntu:24.04 + Node 22 |
| `ghcr.io/agencycli/sandbox-codex:latest` | OpenAI Codex (`codex`) | ubuntu:24.04 + Node 22 |
| `ghcr.io/agencycli/sandbox-gemini:latest` | Gemini CLI (`gemini`) | ubuntu:24.04 + Node 22 |
| `ghcr.io/agencycli/sandbox-generic:latest` | Any / custom | ubuntu:24.04 + Node 22 |

All images include: `git`, `gh` (GitHub CLI), `curl`, `jq`, `ripgrep`, `make`, `openssh-client`.

**Not included by default** (install inside the container or in a custom image):
Go, Python, Rust, database clients, etc. Agents can install these during a task via `apt-get` or the appropriate package manager.

## Build locally

```bash
# Default (international mirrors)
docker build -t ghcr.io/agencycli/sandbox-claudecode:latest \
             -f docker/sandbox-claudecode/Dockerfile .

# Faster inside China (Aliyun apt + npmmirror)
docker build --build-arg CN_MIRROR=1 \
             -t ghcr.io/agencycli/sandbox-claudecode:latest \
             -f docker/sandbox-claudecode/Dockerfile .
```

Same `--build-arg CN_MIRROR=1` works for all sandbox images.

## Root + permission bypass

All images run as **root** (so credential mounts at `/root/.claude`, `/root/.ssh`, etc. work without path remapping).

Each agent CLI requires specific flags/env vars to run non-interactively as root:

| Agent | Mechanism |
|-------|-----------|
| Claude Code | `IS_SANDBOX=1` (baked into image ENV) + `--dangerously-skip-permissions` (added by runner) + pre-configured `~/.claude/settings.json` |
| Codex CLI | `CODEX_UNSAFE_ALLOW_NO_SANDBOX=1` (baked into image ENV) |
| Gemini CLI | No extra flags needed; use `GEMINI_API_KEY` or mount `~/.gemini/` |

These are set automatically — you don't need to configure anything.

## How agencycli uses these images

When an agent is hired with `--sandbox docker`, every `agencycli run` invocation becomes:

```bash
docker run --rm -i \
  --memory=4096m \
  --network=bridge \
  -v /path/to/agent-dir:/workspace \          ← agent working dir
  -v /path/to/workspace-root:/path/to/workspace-root \ ← full workspace (inter-agent tasks)
  -v /path/to/repo:/path/to/repo \            ← project repo (same path as host)
  -v ~/.claude:/root/.claude:ro \             ← credentials, read-only
  -v ~/.config/gh:/root/.config/gh:ro \
  -v ~/.ssh:/root/.ssh:ro \
  -v /usr/local/bin/agencycli:/usr/local/bin/agencycli:ro \  ← agencycli binary
  -w /workspace \
  -e IS_SANDBOX=1 \                           ← claudecode: allow root run
  -e ANTHROPIC_API_KEY \                      ← inherited from host (value hidden)
  ghcr.io/agencycli/sandbox-claudecode:latest \
  claude --dangerously-skip-permissions \
         --output-format stream-json \
         --resume <session-id> \
         -p --print-file /workspace/.prompt-xxx.txt
```

The agent working directory (`CLAUDE.md`, tasks, skills, etc.) is bind-mounted at `/workspace`. The container is ephemeral (`--rm`) — only the mounted workspace persists between runs.

## Custom images

Extend any base image to add project-specific tools:

```dockerfile
FROM ghcr.io/agencycli/sandbox-claudecode:latest

# Add your project's toolchain
RUN apt-get update && apt-get install -y golang-go python3-pip \
    && rm -rf /var/lib/apt/lists/*
RUN npm install -g yarn
```

Then use it:

```bash
agencycli hire --project my-api --team engineering --model claudecode \
               --name dev --sandbox docker \
               --sandbox-image "myorg/my-claude-sandbox:v1"
```

## Credential setup

Credentials are mounted read-only from your host:

| Agent | Required path | How to set up |
|-------|--------------|---------------|
| Claude Code | `~/.claude.json` + `~/.claude/` | Run `claude` interactively once on host |
| Codex | `~/.codex/auth.json` | Run `codex login` on host, or set `OPENAI_API_KEY` env var |
| Gemini | `~/.gemini/` (`oauth_creds.json`) | Run `gemini` on host, or set `GEMINI_API_KEY` env var |
| GitHub CLI | `~/.config/gh/` | Run `gh auth login` on host |
| SSH (git push) | `~/.ssh/` | Standard SSH key setup |

API keys are forwarded from your host environment using `-e KEY` (value never appears in command line).

## Security notes

- Containers run as **root** — required for credential mounts at `/root/.*`
- Credential mounts are read-only (`:ro`) — containers cannot modify host credentials
- Network defaults to `bridge` — use `--sandbox-network none` for fully offline operation
- Each run is ephemeral: the container is destroyed after the agent exits (`--rm`)
- The agencycli binary is mounted read-only — agents can use it but not modify it
