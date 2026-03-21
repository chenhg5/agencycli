# agencycli generic sandbox image
#
# Minimal Ubuntu base with common dev tools.
# Use this for generic-cli agents or as a base for custom images.
#
# Build:
#   docker build -t ghcr.io/agencycli/sandbox-generic:latest \
#                -f docker/sandbox-generic/Dockerfile .

FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    wget \
    git \
    jq \
    make \
    unzip \
    sudo \
    openssh-client \
    ripgrep \
    python3 \
    python3-pip \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Node.js (needed by many agent CLIs)
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# GitHub CLI
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

RUN useradd -m -s /bin/bash agent \
    && echo "agent ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

WORKDIR /workspace
RUN chown agent:agent /workspace
USER agent

CMD ["/bin/bash"]
