<div align="center">

<img src="https://raw.githubusercontent.com/chenhg5/agencycli/main/docs/banner.svg" alt="agencycli" width="900" />

<br/>

[![npm](https://img.shields.io/npm/v/%40agencycli%2Fagencycli?color=cb3837&logo=npm&label=npm&style=flat-square)](https://www.npmjs.com/package/@agencycli/agencycli)
[![Go](https://img.shields.io/github/go-mod/go-version/chenhg5/agencycli?logo=go&logoColor=white&style=flat-square)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](https://opensource.org/licenses/MIT)
[![Works with](https://img.shields.io/badge/%E6%94%AF%E6%8C%81-Claude%20%C2%B7%20Codex%20%C2%B7%20Gemini%20%C2%B7%20Cursor-8a2be2?style=flat-square)](#支持任意-ai-编程-agent)


**几分钟内搭建一支自运转的 AI 智能体团队。**  
一个 CLI，无需服务器。智能体自主规划、执行、相互通信——你睡着的时候它们也在工作。


[**English**](README.md) &nbsp;·&nbsp; [快速开始](#快速开始) &nbsp;·&nbsp; [安装](#安装) &nbsp;·&nbsp; [命令参考](docs/commands.md) &nbsp;·&nbsp; [工作区结构](docs/workspace-layout.md)

</div>

## 这是什么？

**agencycli** 是一个轻量级 CLI 工具，用于构建和运营 AI 智能体团队。你只需定义一次组织架构——团队、角色、项目、技能——智能体就会自动装配上下文、领取任务，并按心跳节奏自主运转。

核心亮点：**智能体可以雇用、互发消息、彼此协调。** PM 智能体可以给 Dev 智能体创建任务，Dev 智能体在合并 PR 前可以向人类请求确认，QA 智能体每 30 分钟自动醒来扫描 PR——全程无需人工干预。

## 六大设计支柱

### 1 — 上下文网格：角色 × 项目

上下文不是扁平的，而是从两个维度组合而来：

```
             ← 横向：你是谁（角色）→
              engineer    pm    qa-reviewer    writer
              ─────────────────────────────────────
纵向：     api-service │  ●        ●              
项目轴     mobile-app  │  ●                   ●   
           data-pipeline│  ●        ●              
```

每个智能体在 `hire` 时自动合并完整上下文链：`机构上下文 → 团队上下文 → 角色上下文 → 项目上下文`。修改一个角色的 prompt，所有该角色的智能体在下次 `sync` 后都会更新。

### 2 — 自主心跳 + 唤醒流程

智能体不只是被动等待任务。配置心跳后，它们会按计划自动醒来、清空任务队列；当队列为空时，执行**唤醒流程**（`wakeup.md`），主动发现新工作：

```
[heartbeat cc-connect/pm]  sleeping 28m → next at 09:30:00
[heartbeat cc-connect/qa]  waking up — checking crons, running pending tasks
[heartbeat cc-connect/qa]  ▶ wakeup routine  (扫描 open PR…)
[heartbeat cc-connect/qa]  wakeup cycle done — sleeping 24m → next at 09:27:00
```

时间窗口、工作日限制、Cron 定时——全部可配置。调度器重启时自动抖动，避免所有智能体同时唤醒。

### 3 — Inbox：智能体互相通信

每个参与者（智能体或人类）都有一个收件箱，通信异步非阻塞：

```
Human  →  PM 智能体：   "优先处理 issue #42"
PM     →  Dev 智能体：  "新任务已就绪，补充上下文：..."
Dev    →  Human：       confirm-request — "PR #205 待合并"  ← 阻塞直到你决定
QA     →  PM + Human：  "本周评审汇总"  （群发）
PM     →  Dev + QA：    inbox fwd <msg-id> --note "仅供参考"
```

未读消息在每次唤醒时自动注入提示词顶部——无需在 `wakeup.md` 中写轮询逻辑。

### 4 — 模板：打包复用整支团队

把整个机构配置——团队、角色、技能、行动手册、项目蓝图——打包成一个 `.tar.gz`，分享给任何人，一条命令应用到新项目：

```bash
agencycli create agency --name "AcmeCorp" \
  --template https://yourcdn.com/tech-agency.tar.gz

agencycli project apply --project my-new-service
agencycli scheduler start
# 完成，智能体开始运转。
```

### 5 — Docker 沙箱：默认安全隔离

智能体可以在隔离的 Docker 容器中运行。不会意外破坏宿主机，不会泄露凭据给不可信代码，不会产生失控进程。每个任务都在全新容器中执行，任务结束后容器销毁，只有挂载的工作区目录会保留变更。

```bash
agencycli hire --project my-api --team engineering --role developer \
  --model claudecode --name dev --sandbox docker
```

自动挂载的内容：
- 智能体工作目录和项目仓库（读写）
- `agencycli` 二进制文件（供智能体调用 `task done`、`inbox send` 等命令）
- 凭据目录（`~/.claude`、`~/.config/gh`、`~/.ssh`、`~/.codex` 等，只读）
- 常见 API Key 以环境变量方式注入

智能体对自己的工作区有完整权限，对宿主机的其余部分则完全隔离。

### 6 — Skills：可复用的打包能力

技能是 Markdown + 可选脚本，部署到智能体工作目录。无内置技能——只定义你真正需要的，绑定到团队或角色，`sync` 时自动分发。

```
skills/github-push-relay/
  skill.yaml
  prompt.md              # {{SKILL_DIR}}/push.sh 解析为实际路径
  push.sh                # 附带脚本，chmod+x 保留
```

## 安装

### 通过 AI Agent 安装并配置（推荐）

最简单的方式——把下面这句话发给 Claude Code 或任意 AI 编程 Agent，它会帮你完成全部安装和配置：

```
Follow https://raw.githubusercontent.com/chenhg5/agencycli/refs/heads/main/INSTALL.md to install and configure agencycli.
```

### 手动安装

```bash
npm install -g @agencycli/agencycli      # npm，无需安装 Go

go install github.com/chenhg5/agencycli/cmd/agencycli@latest  # Go

# 从源码构建
git clone https://github.com/chenhg5/agencycli && cd agencycli && make install
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

# 4. 查看状态
agencycli inbox list              # 等待你决策的确认任务
agencycli inbox messages          # 来自智能体的异步消息
agencycli task list --project my-service --agent pm
```

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
├── overview                                # 仪表盘：智能体、团队、技能、收件箱一览
├── create agency / team / role / project   # 搭建组织架构
├── hire / fire / sync                      # 管理智能体
├── task add / list / done / confirm-request# 任务队列（7 状态流转）
├── inbox send / messages / reply / fwd     # 异步消息通信
├── scheduler start / stop / status         # 心跳调度器
├── cron add / list / delete                # 定时任务
├── template pack / info                    # 打包分享配置
└── --dir <path>                            # 从任意位置操作指定工作区
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
| 是否需要服务器 | 否 | 通常需要 |

## 许可证

MIT
