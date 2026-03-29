<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/banner.svg" alt="agencycli" width="900"/>
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/@agencycli/agencycli">
    <img src="https://img.shields.io/npm/v/%40agencycli%2Fagencycli?color=cb3837&logo=npm&label=npm&style=flat-square" alt="npm"/>
  </a>
  <a href="https://github.com/chenhg5/agencycli/releases">
    <img src="https://img.shields.io/github/v/release/chenhg5/agencycli?style=flat-square&logo=github" alt="Release"/>
  </a>
  <a href="https://go.dev/">
    <img src="https://img.shields.io/github/go-mod/go-version/chenhg5/agencycli?logo=go&logoColor=white&style=flat-square" alt="Go"/>
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square" alt="License: MIT"/>
  </a>
  <a href="https://goreportcard.com/report/github.com/chenhg5/agencycli">
    <img src="https://goreportcard.com/badge/github.com/chenhg5/agencycli?style=flat-square" alt="Go Report Card"/>
  </a>
</p>

<p align="center">
  <a href="#支持任意-ai-编程-agent">
    <img src="https://img.shields.io/badge/%E6%94%AF%E6%8C%81-Claude%20%C2%B7%20Codex%20%C2%B7%20Gemini%20%C2%B7%20Cursor-8a2be2?style=flat-square" alt="支持"/>
  </a>
</p>

<p align="center">
  <strong>几分钟内搭建一支自运转的 AI 智能体团队。</strong><br/>
  一个 CLI + 内置 Web 控制台。智能体自主规划、执行、相互通信——你睡着的时候它们也在工作。
</p>

<p align="center">
  <a href="./README.md">English</a> &nbsp;|&nbsp;
  <a href="#快速开始">快速开始</a> &nbsp;|&nbsp;
  <a href="#安装">安装</a> &nbsp;|&nbsp;
  <a href="docs/commands.md">命令参考</a> &nbsp;|&nbsp;
  <a href="docs/workspace-layout.md">工作区结构</a>
</p>

## 这是什么？

**agencycli** 是一个轻量级 CLI 工具，用于构建和运营 AI 智能体团队。你只需定义一次组织架构——团队、角色、项目、技能——智能体就会自动装配上下文、领取任务，并按心跳节奏自主运转。

核心亮点：**智能体可以雇用、互发消息、彼此协调。** PM 智能体可以给 Dev 智能体创建任务，Dev 智能体在合并 PR 前可以向人类请求确认，QA 智能体每 30 分钟自动醒来扫描 PR——全程无需人工干预。


https://github.com/user-attachments/assets/6c2f6864-b440-46e3-86f6-2ebcd06ee6a0


## 六大设计支柱

### 1 — 上下文网格：角色 × 项目

上下文不是扁平的，而是从两个维度组合而来。每个智能体在 `hire` 时自动合并完整上下文链：`机构上下文 → 团队上下文 → 角色上下文 → 项目上下文`。修改一个角色的 prompt，所有该角色的智能体在下次 `sync` 后都会更新。

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-1-context.svg" alt="Context grid" width="900"/>
</p>

### 2 — 自主心跳 + 唤醒流程

智能体不只是被动等待任务。配置心跳后，它们会按计划自动醒来、清空任务队列；当队列为空时，执行**唤醒流程**（`wakeup.md`），主动发现新工作。时间窗口、工作日限制、Cron 定时——全部可配置。调度器重启时自动抖动，避免所有智能体同时唤醒。

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-2-heartbeat.svg" alt="Heartbeat scheduler" width="900"/>
</p>

### 3 — Inbox：智能体互相通信

每个参与者（智能体或人类）都有一个收件箱，通信异步非阻塞。未读消息在每次唤醒时自动注入提示词顶部。`confirm-request` 创建阻塞门控：任务暂停，直到你确认。

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-3-inbox.svg" alt="Inbox messaging" width="900"/>
</p>

### 4 — 模板：打包复用整支团队

把整个机构配置——团队、角色、技能、行动手册、项目蓝图——打包成一个 `.tar.gz`，分享给任何人，一条命令应用到新项目。

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-4-templates.svg" alt="Templates" width="900"/>
</p>

### 5 — Docker 沙箱：默认安全隔离

智能体在隔离的 Docker 容器中运行。不会意外破坏宿主机，不会泄露凭据，不会产生失控进程。工作区和 `agencycli` 二进制读写挂载，凭据目录只读挂载。

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-5-docker.svg" alt="Docker sandbox" width="900"/>
</p>

### 6 — Skills：可复用的打包能力

技能是 `SKILL.md`（YAML frontmatter + Markdown prompt）加可选脚本，部署到每个绑定了该技能的智能体工作目录。定义一次，绑定到角色，`sync` 时自动分发。

<p align="center">
  <img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/pillar-6-skills.svg" alt="Skills" width="900"/>
</p>

## 安装

### 通过 AI Agent 安装并配置（推荐）

最简单的方式——把下面这句话发给 Claude Code 或任意 AI 编程 Agent，它会帮你完成全部安装和配置：

```
参考 https://raw.githubusercontent.com/chenhg5/agencycli/refs/heads/main/INSTALL.md 帮我安装 agencycli 然后组建一个 agent 公司。
```

### 手动安装

```bash
npm install -g @agencycli/agencycli      # npm，无需安装 Go

go install github.com/chenhg5/agencycli/cmd/agencycli@latest  # Go

# 从源码构建（含 Web 控制台）
git clone https://github.com/chenhg5/agencycli && cd agencycli && make install
```

### 从源码构建（开发）

```bash
git clone https://github.com/chenhg5/agencycli
cd agencycli
make build          # 构建前端 + Go 二进制 → dist/agencycli
make install        # 构建并安装到 $GOPATH/bin
```

## 快速开始

```bash
# 1. 创建工作区（自动生成 .gitignore + agency-prompt.md）
agencycli create agency --name "MyAgency"
cd MyAgency

# 2. 应用项目蓝图 — 雇用所有智能体 + 配置心跳 + 安装 playbook
agencycli create project --name "my-service" --blueprint default
agencycli project apply  --project my-service

# 3. 启动调度器 — 智能体开始自主运转
agencycli scheduler start

# 4. 打开 Web 控制台 — 在浏览器中管理一切
agencycli start                   # http://127.0.0.1:27892
agencycli start --addr 0.0.0.0:8080 --open   # 自定义端口 + 自动打开浏览器

# 5. 命令行查看状态
agencycli inbox list              # 等待你决策的确认任务
agencycli inbox messages          # 来自智能体的异步消息
agencycli task list --project my-service --agent pm
```

## Web 控制台

Web 控制台内嵌在二进制文件中——无需额外进程或 Node.js。一条命令启动：

```bash
agencycli start                          # 默认：127.0.0.1:27892
agencycli start --addr 0.0.0.0:8080     # 自定义地址
agencycli start --api-key my-secret     # 带认证
```

**功能：** 工作台（消息 + 任务）、团队/角色管理、项目成员、调度计划（心跳/Cron/运行状态）、运行记录与 Token 消耗、会话管理、Agent 运行、Prompt 编辑、技能管理——支持多语言（English / 中文 / 繁體中文 / 日本語）和深色模式。

> 本地开发热更新：`agencycli api serve` + `cd web && pnpm dev`。

## 支持大多数 AI Agent

agencycli 是运行时基础设施，而非 SDK。智能体就是你已经在用的 CLI 工具：

| Agent 运行时 | `--model` |
|---|---|
| [Claude Code](https://docs.anthropic.com/claude-code) | `claudecode` |
| [OpenAI Codex](https://github.com/openai/codex) | `codex` |
| [Gemini CLI](https://github.com/google-gemini/gemini-cli) | `gemini` |
| [Cursor](https://www.cursor.com/) | `cursor` |
| [Qoder](https://qoder.ai) | `qoder` |
| [OpenCode](https://opencode.ai) | `opencode` |
| [iFlow](https://iflow.ai) | `iflow` |
| 任意 CLI 工具 | `generic-cli` |

多模型自由混用——PM 用 Claude，Dev 用 Codex，Writer 用 Gemini。每个智能体都会收到其运行时所需格式的上下文。

## 命令总览

```
agencycli
├── start                                  # 启动 Web 控制台（API + 前端）
├── overview                               # CLI 仪表盘
├── create agency / team / role / project  # 搭建组织架构
├── hire / fire / sync                     # 管理智能体
├── task add / list / done / confirm-request # 任务队列（7 状态流转）
├── run / exec                             # 手动运行 Agent
├── inbox send / messages / reply / fwd    # 异步消息通信
├── scheduler start / stop / status        # 心跳调度器
├── session show / set / reset             # Agent 会话管理
├── cron add / list / delete               # 定时任务
├── template pack / info                   # 打包分享配置
├── api serve                              # 仅 JSON API（开发用）
└── --dir <path>                           # 从任意位置操作指定工作区
```

→ **[完整命令参考](docs/commands.md)**  
→ **[工作区结构说明](docs/workspace-layout.md)**  
→ **[Docker 沙箱](docs/sandbox-design.md)**

## 为什么不用 LangGraph / CrewAI / AutoGen？

那些是框架——你用 Python 代码来串联智能体。**agencycli 是基础设施**——你用 Markdown 和 YAML 来描述。智能体就是你已经在用的 CLI 工具。无 SDK，无绑定，无需运行服务器。

| | agencycli | 框架方案 |
|--|-----------|---------|
| 智能体运行时 | 你现有的 CLI 工具 | 框架的 agent loop |
| 配置方式 | Markdown + YAML | Python 代码 |
| 多模型支持 | 任意 CLI，混用自由 | 通常绑定一个 SDK |
| 上下文管理 | 分层自动合并 | 手动拼接 prompt |
| Web UI | 内置（单一二进制） | 需要单独部署 |
| 是否需要服务器 | 否 | 通常需要 |

## 许可证

MIT
