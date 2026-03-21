# agencycli sandbox image for Claude Code (claudecode)
#
# Build (default — uses international mirrors):
#   docker build -t ghcr.io/agencycli/sandbox-claudecode:latest \
#                -f docker/sandbox-claudecode/Dockerfile .
#
# Build with Chinese mirrors (much faster inside China):
#   docker build --build-arg CN_MIRROR=1 \
#                -t ghcr.io/agencycli/sandbox-claudecode:latest \
#                -f docker/sandbox-claudecode/Dockerfile .
#
# What's inside:
#   - Claude Code CLI  (claude)
#   - GitHub CLI       (gh)
#   - Go 1.25          (go, gofmt, go test, go build …)
#   - Git, curl, jq, ripgrep, make, unzip
#   - Node.js 22 LTS (required by claude)
#   - Pre-configured ~/.claude/settings.json to bypass permission prompts
#
# Run manually for debugging:
#   docker run --rm -it \
#     -v $(pwd):/workspace \
#     -v ~/.claude:/root/.claude:ro \
#     -v ~/.config/gh:/root/.config/gh:ro \
#     -e ANTHROPIC_API_KEY=sk-ant-... \
#     ghcr.io/agencycli/sandbox-claudecode:latest \
#     claude --dangerously-skip-permissions -p "hello"

FROM ubuntu:24.04

ARG CN_MIRROR=0
ARG GO_VERSION=1.25.7

ENV DEBIAN_FRONTEND=noninteractive \
    NODE_VERSION=22 \
    # IS_SANDBOX=1 lets --dangerously-skip-permissions work as root.
    IS_SANDBOX=1

# ── APT source mirrors ─────────────────────────────────────────────────────────
# Ubuntu 24.04 uses DEB822 format in /etc/apt/sources.list.d/ubuntu.sources.
# We replace the URIs field. Falls back to the old sources.list format.
RUN if [ "$CN_MIRROR" = "1" ]; then \
      if [ -f /etc/apt/sources.list.d/ubuntu.sources ]; then \
        sed -i 's|http://archive.ubuntu.com/ubuntu|http://mirrors.aliyun.com/ubuntu|g; \
                s|http://security.ubuntu.com/ubuntu|http://mirrors.aliyun.com/ubuntu|g' \
            /etc/apt/sources.list.d/ubuntu.sources; \
      else \
        sed -i 's|http://archive.ubuntu.com/ubuntu|http://mirrors.aliyun.com/ubuntu|g; \
                s|http://security.ubuntu.com/ubuntu|http://mirrors.aliyun.com/ubuntu|g' \
            /etc/apt/sources.list; \
      fi; \
    fi

# ── System packages ────────────────────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    wget \
    git \
    jq \
    make \
    unzip \
    openssh-client \
    ripgrep \
    python3 \
    python3-pip \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# ── Node.js (required by Claude Code CLI) ─────────────────────────────────────
RUN if [ "$CN_MIRROR" = "1" ]; then \
      curl -fsSL https://mirrors.aliyun.com/nodejs-release/v${NODE_VERSION}/node-v${NODE_VERSION}.21.0-linux-x64.tar.gz 2>/dev/null \
        | tar -C /usr/local --strip-components=1 -xzf - 2>/dev/null || \
      curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
        && apt-get install -y nodejs && rm -rf /var/lib/apt/lists/*; \
    else \
      curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
        && apt-get install -y nodejs && rm -rf /var/lib/apt/lists/*; \
    fi

# ── npm registry ───────────────────────────────────────────────────────────────
RUN if [ "$CN_MIRROR" = "1" ]; then \
      npm config set registry https://registry.npmmirror.com; \
    fi

# ── Go ────────────────────────────────────────────────────────────────────────
RUN if [ "$CN_MIRROR" = "1" ]; then \
      curl -fsSL "https://mirrors.aliyun.com/golang/go${GO_VERSION}.linux-amd64.tar.gz" \
        | tar -C /usr/local -xzf -; \
    else \
      curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
        | tar -C /usr/local -xzf -; \
    fi
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}" \
    GOPATH="/root/go" \
    GOMODCACHE="/root/go/pkg/mod"
# Use Chinese Go module proxy when CN_MIRROR=1
RUN if [ "$CN_MIRROR" = "1" ]; then \
      go env -w GOPROXY=https://goproxy.cn,direct GONOSUMDB='*'; \
    fi

# ── Claude Code CLI ────────────────────────────────────────────────────────────
RUN npm install -g @anthropic-ai/claude-code

# ── GitHub CLI ─────────────────────────────────────────────────────────────────
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
        | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) \
         signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] \
         https://cli.github.com/packages stable main" \
        | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh \
    && rm -rf /var/lib/apt/lists/*

# ── Claude Code sandbox settings ──────────────────────────────────────────────
# Pre-configure to bypass permission prompts automatically.
# Belt-and-suspenders with --dangerously-skip-permissions + IS_SANDBOX=1.
RUN mkdir -p /root/.claude && printf \
    '{"skipDangerousModePermissionPrompt":true,"permissions":{"defaultMode":"bypassPermissions"}}\n' \
    > /root/.claude/settings.json

# ── agencycli shim ────────────────────────────────────────────────────────────
# Real binary is bind-mounted at runtime: -v /path/to/agencycli:/usr/local/bin/agencycli:ro
RUN printf '#!/bin/sh\necho "agencycli: binary not mounted" >&2\nexit 1\n' \
    > /usr/local/bin/agencycli && chmod +x /usr/local/bin/agencycli

WORKDIR /workspace

CMD ["/bin/bash"]
