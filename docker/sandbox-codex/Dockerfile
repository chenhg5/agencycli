# agencycli sandbox image for OpenAI Codex CLI
#
# Build:
#   docker build -t ghcr.io/agencycli/sandbox-codex:latest \
#                -f docker/sandbox-codex/Dockerfile .
#
# Build with Chinese mirrors:
#   docker build --build-arg CN_MIRROR=1 \
#                -t ghcr.io/agencycli/sandbox-codex:latest \
#                -f docker/sandbox-codex/Dockerfile .

FROM ubuntu:24.04

ARG CN_MIRROR=0

ENV DEBIAN_FRONTEND=noninteractive \
    NODE_VERSION=22 \
    # Required: Codex CLI mandates its own sandbox, but inside Docker we are
    # already sandboxed. This flag tells Codex to skip its inner sandbox check.
    CODEX_UNSAFE_ALLOW_NO_SANDBOX=1

# ── APT source mirrors ─────────────────────────────────────────────────────────
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
    ca-certificates curl git jq make unzip openssh-client ripgrep fzf \
    python3 python3-pip \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# ── Node.js ────────────────────────────────────────────────────────────────────
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
    && apt-get install -y nodejs && rm -rf /var/lib/apt/lists/*

# ── npm registry ───────────────────────────────────────────────────────────────
RUN if [ "$CN_MIRROR" = "1" ]; then \
      npm config set registry https://registry.npmmirror.com; \
    fi

# ── Codex CLI ─────────────────────────────────────────────────────────────────
RUN npm install -g @openai/codex

# ── GitHub CLI ─────────────────────────────────────────────────────────────────
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
        | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) \
         signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] \
         https://cli.github.com/packages stable main" \
        | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y gh \
    && rm -rf /var/lib/apt/lists/*

# ── agencycli shim ────────────────────────────────────────────────────────────
RUN printf '#!/bin/sh\necho "agencycli: binary not mounted" >&2\nexit 1\n' \
    > /usr/local/bin/agencycli && chmod +x /usr/local/bin/agencycli

WORKDIR /workspace

CMD ["/bin/bash"]
