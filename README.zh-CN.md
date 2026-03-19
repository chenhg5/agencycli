# agencycli

> **几分钟内搭建一支自运转的 AI 智能体团队。**  
> 一个 CLI，无需服务器。智能体自主规划、执行、相互通信——你睡着的时候它们也在工作。

```
npm install -g @agencycli/agentctl
```

---

## 这是什么？

**agencycli** 是一个轻量级 CLI 工具，用于构建和运营 AI 智能体团队。你只需定义一次组织架构——团队、角色、项目、技能——智能体就会自动装配上下文、领取任务，并按心跳节奏自主运转。

核心亮点：**智能体可以雇用、互发消息、彼此协调。** PM 智能体可以给 Dev 智能体创建任务，Dev 智能体在合并 PR 前可以向人类请求确认，QA 智能体每 30 分钟自动醒来扫描 PR——全程无需人工干预。

---

## 四大设计支柱

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

时间窗口、工作日限制、Cron 定时——全部可配置。非活动时段显示 `⏸ outside active window — next wakeup in Xh`。

### 3 — Inbox：智能体互相通信

每个参与者（智能体或人类）都有一个收件箱，通信异步非阻塞：

```
Human  →  PM 智能体：   "优先处理 issue #42"
PM     →  Dev 智能体：  "新任务已就绪，补充上下文：..."
Dev    →  Human：       confirm-request — "PR #205 待合并"  ← 阻塞直到你决定
QA     →  PM + Human：  "本周评审汇总（群发）"
PM     →  Dev + QA：    inbox fwd <msg-id> --to dev --to qa --note "仅供参考"
```

未读消息在每次唤醒时自动注入提示词顶部——无需在 `wakeup.md` 中写轮询逻辑。

### 4 — 模板：打包复用整支团队

把整个机构配置——团队、角色、技能、智能体行动手册、项目蓝图——打包成一个 `.tar.gz`，分享给任何人，一条命令应用到新项目：

```bash
agencycli create agency --name "AcmeCorp" \
  --template https://yourcdn.com/tech-agency.tar.gz

agencycli project apply --project my-new-service
agencycli scheduler start
# 完成，智能体开始运转。
```

---

## 安装

```bash
# npm（无需安装 Go）
npm install -g @agencycli/agentctl

# Go
go install github.com/chenhg5/agencycli/cmd/agencycli@latest

# 从源码构建
git clone https://github.com/chenhg5/agencycli && cd agencycli && make install
```

---

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

---

## 支持的 AI 模型

| `--model`     | 上下文文件格式 |
|---------------|----------------|
| `claudecode`  | `CLAUDE.md` + `@import` 层 + `.claude/skills/` |
| `codex`       | `AGENTS.md`（技能内联） |
| `cursor`      | `.cursorrules` + `.cursor/rules/agencycli.mdc` |
| `gemini`      | `GEMINI.md` + `@import` 层 + `.gemini/skills/` |
| `qoder` / `opencode` / `iflow` | 单文件合并 |
| `generic-cli` | `context.md` 纯文本 |

---

## 命令总览

```
agencycli
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

---

## 为什么不用 LangGraph / CrewAI / AutoGen？

那些是框架——你用 Python 代码来串联智能体。**agencycli 是基础设施**——你用 Markdown 和 YAML 来描述。智能体就是你已经在用的 CLI 工具（Claude Code、Codex、Gemini CLI……）。无 SDK，无绑定，无需运行服务器。

| | agencycli | 框架方案 |
|--|-----------|---------|
| 智能体运行时 | 你现有的 CLI 工具 | 框架的 agent loop |
| 配置方式 | Markdown + YAML | Python 代码 |
| 多模型支持 | 任意 CLI，混用自由 | 通常绑定一个 SDK |
| 上下文管理 | 分层自动合并 | 手动拼接 prompt |
| 是否需要服务器 | 否 | 通常需要 |

---

## Roadmap

- [x] 上下文网格（agency → team → role → project）
- [x] 心跳调度器（active-hours / active-days 时间窗口）
- [x] 唤醒流程（`wakeup.md`）——主动自主工作
- [x] 任意参与者间的异步 Inbox 消息通信
- [x] task confirm-request——智能体向人类升级决策
- [x] Cron 定时任务
- [x] Docker 沙箱（凭据自动挂载）
- [x] 模板打包/应用
- [x] 项目蓝图
- [ ] `depends_on` 任务依赖解析
- [ ] E2B / Daytona 沙箱提供商
- [ ] 运行日志轮转

---

## 许可证

MIT
