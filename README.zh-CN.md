# agencycli

**agencycli** 是一个用于按“机构式”组织结构管理 AI 智能体团队的 CLI 工具。你只需定义团队、角色、项目与智能体行动手册（playbook），即可一次性完成上下文装配；随后就能直接雇用智能体、分配任务，并让它们按心跳节奏自动执行。

不再在不同会话间复制粘贴提示词，不再出现上下文漂移，不再手动串联“谁做什么”。

## 核心模型

```
Agent = Model + Context + Skills
```

上下文在 `hire` 时按层级自动合并：

```
Agency                ← 全局规则、价值观、语气风格
  └─ Team             ← 能力组（engineering、qa、growth…）
       └─ Sub-team    ← 子团队（engineering/backend、engineering/frontend）
            └─ Role   ← 岗位职能（go-developer、pr-reviewer、content-writer）
                 └─ Project ← 具体产品或项目
```

智能体通过 **心跳循环** 自主工作：醒来、处理所有待办、休眠。当任务队列为空时，默认执行 **唤醒流程**（`wakeup.md`），让智能体主动发现新工作（如 GitHub issue、未合并 PR 等）。智能体之间与人类通过 **收件箱消息** 异步沟通——任何参与者都可给对方发消息，收件人在下一次醒来时读取。

---

## 工作原理

### 第 1 层：上下文管理

- 一次性定义机构：团队、角色、技能以及它们如何组合
- `hire` 雇用智能体 → agencycli 合并完整链路并写入可直接使用的工作目录
- 支持模型：`claudecode`、`codex`、`gemini`、`cursor`、`qoder`、`opencode`、`iflow`、`generic-cli`
- `sync` 将提示词/技能变更同步到所有受影响的智能体

### 第 2 层：任务自动化

- 每个智能体 **任务队列**（7 个状态，优先级 0=紧急 … 3=低）
- **心跳调度器**：非重叠唤醒循环，保留会话，支持时间窗调度（`active_hours`、`active_days`）
- **唤醒流程**：队列为空时执行 `wakeup.md` 作为合成任务，实现无需手动指派的主动工作
- **Cron 任务**：按 crontab 定期生成任务
- **人类收件箱**：智能体可通过 `task confirm-request` 将任务路由到这里，等待确认/拒绝/转发
- **异步消息**：任意参与者之间可发送非阻塞消息
- **Docker 沙箱**：隔离容器，自动挂载凭据与仓库

### 第 3 层：智能体行动手册（playbooks）

行动手册（`wakeup.md`）定义了智能体在任务队列为空时的行为。将它们放入 `agent-playbooks/`，即可随模板分发。

```yaml
# project.yaml
agents:
  - name: pm
    playbook: pm.md          # 在 project apply 时复制为 agent 目录下的 wakeup.md
    heartbeat:
      enabled: true
      interval: 30m
```

当执行 `project apply` 时，会把 `agent-playbooks/pm.md` → `agents/pm/wakeup.md`，并自动在 `heartbeat.yaml` 里设置 `wakeup_prompt: "@wakeup.md"`。

### 第 4 层：模板

可将整个机构（团队、角色、技能、行动手册、项目蓝图）打包为 `.tar.gz` 模板。分享后可一条命令应用到新机构。

---

## 支持的模型

| `--model`     | 上下文文件格式                                               |
|---------------|--------------------------------------------------------------|
| `claudecode`  | `CLAUDE.md` + `@import` 层 + `.claude/skills/`               |
| `codex`       | `AGENTS.md` 单文件合并（技能内联）                           |
| `cursor`      | `.cursorrules` + `.cursor/rules/agencycli.mdc`               |
| `gemini`      | `GEMINI.md` + `@import` 层 + `.gemini/skills/`               |
| `qoder`       | `AGENTS.md` 单文件合并                                       |
| `opencode`    | `OPENCODE.md` 单文件合并                                     |
| `iflow`       | `IFLOW.md` 单文件合并                                        |
| `generic-cli` | `context.md` 纯文本                                          |

---

## 安装

```bash
npm install -g agencycli        # npm（无需 Go）
go install github.com/chenhg5/agencycli/cmd/agencycli@latest  # Go
```

或从源码构建：

```bash
git clone https://github.com/chenhg5/agencycli
cd agencycli && make install
```

---

## 快速开始 —— 使用模板（推荐）

模板打包了团队、角色、技能、行动手册与项目蓝图：

```bash
# 1. 从共享模板创建机构
agencycli create agency --name "MyAgency" \
  --template https://example.com/tech-agency.tar.gz
cd MyAgency

# 2. 查看模板自带的项目蓝图
agencycli project blueprints

# 3. 创建项目 —— project.yaml 预填智能体、心跳与 playbook
agencycli create project --name "my-service" --blueprint default

# 4. 预览将要创建的内容
agencycli project show --project my-service

# 5. 应用：雇用所有智能体 + 配置心跳 + 安装 wakeup.md
agencycli project apply --project my-service

# 6. 启动调度器 —— 智能体按计划醒来执行
agencycli scheduler start

# 7. 监控
agencycli inbox list          # 等待你决策的确认任务
agencycli inbox messages      # 来自智能体的异步消息
agencycli task list --project my-service --agent pm
```

## 快速开始 —— 从零搭建

```bash
# 1. 创建工作区
agencycli create agency --name "MyAgency" --desc "Building great software"
cd MyAgency

# 2. 创建团队与角色
agencycli create team --name "engineering"
agencycli create role --team "engineering" --name "developer"

# 3. 编写智能体行动手册
mkdir -p agent-playbooks
# 编辑 agent-playbooks/dev.md —— 定义 dev 智能体醒来时做什么

# 4. 创建项目蓝图
mkdir -p project-blueprints
# 编辑 project-blueprints/default.yaml

# 5. 创建并应用项目
agencycli create project --name "my-api" --blueprint default
agencycli project apply  --project my-api

# 6. 启动调度器
agencycli scheduler start
```

---

## 工作区结构

```
MyAgency/
  .agencycli/
    agency.yaml              # 工作区元数据
    inbox.yaml               # 人类确认收件箱（自动管理）
    inbox.md                 # 人类可读收件箱（自动生成）
    messages.yaml            # 发送给人类的异步消息

  agency-prompt.md           # 机构级上下文

  teams/
    engineering/
      team.yaml
      prompt.md
      roles/
        developer/
          role.yaml          # skills[]、setup（要创建的目录/文件）
          prompt.md          # 角色级上下文

  skills/                    # 无内置，按需定义
    github-push-relay/
      skill.yaml
      prompt.md              # 使用 {{SKILL_DIR}} 访问脚本路径
      git-push-github.sh     # 附带文件，chmod+x 保留

  agent-playbooks/           # wakeup.md 模板，可随模板分发
    pm.md
    qa-reviewer.md

  project-blueprints/        # 项目模板
    default.yaml             # 声明智能体、心跳与 playbook

  projects/
    my-api/
      project.yaml           # 声明式：智能体 + 心跳 + cron + playbook
      prompt.md              # 项目级上下文
      agents/
        dev/
          CLAUDE.md              # 合并后的上下文（@import 所有层）
          wakeup.md              # 智能体自主流程（project apply 安装）
          tasks.yaml             # 活跃任务队列
          tasks_archive.yaml     # 已完成任务
          heartbeat.yaml         # 心跳配置（project apply 设置）
          crons.yaml             # 定时任务（project apply 设置）
          messages.yaml          # 发给该智能体的异步消息
          runs/                  # 执行日志
          .agencycli-context/    # 各层上下文文件（自动管理）
          .claude/skills/        # 部署后的技能文件
```

---

## 命令

### `create` —— 创建工作区

```bash
agencycli create agency  --name "MyAgency" [--desc "..."] [--template file.tar.gz|dir|URL]
agencycli create team    --name "engineering" [--desc "..."]
agencycli create team    --name "engineering/backend"        # 嵌套子团队
agencycli create role    --team "engineering" --name "developer" [--desc "..."]
agencycli create project --name "my-api" [--desc "..."] [--repo "/path/to/repo"]
agencycli create project --name "my-api" --blueprint default  # 使用项目蓝图
```

### `project` —— 项目生命周期

```bash
# 列出模板自带的蓝图
agencycli project blueprints

# 查看 project.yaml（智能体、心跳、playbook）
agencycli project show --project my-api

# 一键启动：雇用所有智能体 + 配置心跳/cron + 安装 playbook
agencycli project apply --project my-api
agencycli project apply --project my-api --dry-run   # 预览
agencycli project apply --project my-api --force     # 重新雇用已有智能体
```

**`project-blueprints/default.yaml`** 示例：

```yaml
name: "{{PROJECT_NAME}}"
description: "REST API service"
agents:
  - name: dev
    role: developer
    team: engineering
    model: claudecode
    sandbox: true
    heartbeat:
      enabled: true
      interval: 30m
      active_hours: "09:00-20:00"
      active_days: weekdays
    playbook: dev.md          # project apply 时安装为 wakeup.md

  - name: pm
    role: product-manager
    team: product
    model: claudecode
    heartbeat:
      enabled: true
      interval: 30m
    playbook: pm.md
```

### `hire` / `assign` / `fire` / `sync`

```bash
# 雇用智能体（hire 与 assign 等价）
agencycli hire \
  --project my-api --team engineering --role developer \
  --model claudecode --name dev \
  [--sandbox docker] [--force]

# 修改提示词或技能后同步
agencycli sync --project my-api --name dev
agencycli sync --project my-api   # 同项目全部智能体
agencycli sync                    # 整个机构

# 解雇智能体
agencycli fire --project my-api --agent dev           # 软删除 → .fired/
agencycli fire --project my-api --agent dev --force   # 硬删除
```

### `task` —— 任务队列

```bash
agencycli task add    --project P --agent A --title "T" --prompt "..." \
                      [--type feature|bug|chore] [--priority 0-3]
agencycli task list   --project P --agent A [--status pending] [--archived]
agencycli task show   <task-id>
agencycli task cancel <task-id>
agencycli task retry  <task-id>

# 停止所有运行中或待执行任务（紧急制动）
agencycli task stop-all --project P [--agent A | --all-agents] \
                        [--include-running] [--no-pending]

# 查看 token 用量与成本
agencycli task tokens --project P [--agent A | --all-agents] [--all]

# 由智能体在其提示中调用：
agencycli task done --id <id> --status success --summary "完成内容"
agencycli task done --id <id> --status failed  --error "失败原因"

# 发送给人类收件箱，请求确认（阻塞当前任务）
agencycli task confirm-request --id <id> --summary "PR 已就绪" \
  --action-item "Review the diff" \
  --action-item "Confirm merge"
```

**任务优先级：** 0=紧急，1=高，2=普通（默认），3=低。调度器会优先执行最高优先级任务。

**任务生命周期：**
```
pending → in_progress → done_success
                      → done_failed  → (若设置 max_retries 则自动重试)
                      → awaiting_confirmation → in_progress (confirm)
                                              → cancelled   (reject)
```

### `run` / `exec`

```bash
agencycli run  --project P --agent A              # 执行下一个待办任务
agencycli run  --project P --agent A --task <id>  # 执行指定任务
agencycli exec --project P --agent A --prompt "..." # 一次性执行，不进任务队列
```

### `inbox` —— 人类确认与异步消息

收件箱包含两个概念：

**任务确认** —— 智能体暂停等待你的决定：
```bash
agencycli inbox list
agencycli inbox show    <task-id>         # 显示摘要、操作项、日志尾部
agencycli inbox confirm <task-id> [--message "给智能体的备注"]
agencycli inbox reject  <task-id> --reason "..."
agencycli inbox comment <task-id> --message "..."
agencycli inbox forward <task-id> --to <project>/<agent> --note "..."
```

**异步消息** —— 任何参与者之间的非阻塞通信：
```bash
# 发送消息给智能体或人类
agencycli inbox send --to cc-connect/pm --subject "优先处理 #42" --body "..."
agencycli inbox send --to human --from cc-connect/pm --subject "Backlog 更新" --body "..."
agencycli inbox send --to cc-connect/dev-claude --from cc-connect/pm \
  --subject "新任务上下文" --body "刚创建任务的补充信息..."

# 读取消息（默认人类收件箱）
agencycli inbox messages
agencycli inbox messages --recipient cc-connect/pm   # 查看某智能体收件箱
agencycli inbox messages --all                       # 包含已读消息
agencycli inbox messages --mark-read                 # 列出后标记已读

# 回复消息
agencycli inbox reply <msg-id> --body "..."
agencycli inbox reply <msg-id> --from cc-connect/pm --body "..."
```

智能体会在每次唤醒时自动读取未读消息并注入到唤醒提示的顶部，无需在 `wakeup.md` 中显式轮询。

### `scheduler` —— 心跳调度

心跳是 **非重叠唤醒循环**：每次循环处理完所有待办任务后休眠 `interval`，再醒来。当队列为空时触发 **唤醒流程**。

```bash
# 为单个智能体配置心跳
agencycli scheduler heartbeat --project P --agent A \
  --enable --interval 30m \
  --active-hours "09:00-18:00" \  # 仅在该时间窗口唤醒（本地时间）
  --active-days  "weekdays"       # 周一到周五（或 Mon,Wed,Fri / weekends）

# 设置唤醒流程（队列为空时执行）
agencycli scheduler heartbeat --project P --agent A \
  --wakeup-prompt-file /path/to/wakeup.md

# 启动调度器（所有已启用智能体）
agencycli scheduler start
agencycli scheduler stop
agencycli scheduler status
```

支持跨夜时间段，如 `22:00-06:00`。在非活动窗口，调度器会显示 `⏸ outside active window — next wakeup in Xh`。

### `cron` —— 定时任务

```bash
agencycli cron add    --project P --agent A \
  --title "每日站会" --schedule "0 9 * * 1-5" \
  --prompt "生成站会报告..."
agencycli cron list   --project P --agent A
agencycli cron delete <cron-id>  --project P --agent A
agencycli cron enable <cron-id>  --project P --agent A
agencycli cron disable <cron-id> --project P --agent A
```

Cron 触发时会向队列新增任务，调度器在每次心跳唤醒时检查到期的 cron。

### `template` —— 共享机构模板

```bash
# 将当前机构打包为可分享模板
# 包含：agency-prompt.md、teams/、skills/、agent-playbooks/、project-blueprints/
agencycli template pack --output tech-agency.tar.gz \
  --name "tech-project" --version "1.0.0" \
  --author "Alice" --email "alice@example.com" \
  --description "标准软件工程机构模板" \
  --keywords "engineering,software"

# 查看模板信息（本地文件、目录或远程 URL）
agencycli template info tech-agency.tar.gz
agencycli template info tech-agency.tar.gz --json

# 使用模板创建机构
agencycli create agency --name "MyAgency" --template tech-agency.tar.gz
agencycli create agency --name "MyAgency" --template https://example.com/tpl.tar.gz
```

模板归档包含：`agency-prompt.md`、`teams/`、`skills/`、`agent-playbooks/`、`project-blueprints/`。  
归档根目录下的 `template.json` 保存元数据（name、version、author、email、description、keywords）。

### `role` —— 角色管理

```bash
agencycli role list  --team engineering
agencycli role skill add    --team engineering --role developer --skill github-push-relay
agencycli role skill remove --team engineering --role developer --skill github-push-relay
```

### `session` / `list` / `show` / `version`

```bash
agencycli session show  --project P --agent A
agencycli session clear --project P --agent A
agencycli list teams | projects | agents | skills
agencycli show team engineering
agencycli show project my-api
agencycli show agent my-api dev [--raw]
agencycli version
```

### 全局参数：`--dir`

所有命令都可作用于当前目录之外的工作区：

```bash
agencycli --dir /path/to/MyAgency inbox list
agencycli --dir /path/to/MyAgency task list --project my-api --agent dev
agencycli --dir /path/to/MyAgency scheduler start
```

---

## 技能（Skills）

技能是可复用的能力定义，会部署到智能体工作目录中。**无内置**——只定义你真正需要的能力。

```
skills/github-push-relay/
  skill.yaml             # 名称与描述
  prompt.md              # 使用 {{SKILL_DIR}} 获取附带文件的绝对路径
  git-push-github.sh     # 附带脚本
```

在 `prompt.md` 里用 `{{SKILL_DIR}}` 引用同目录文件：

```markdown
Use `{{SKILL_DIR}}/git-push-github.sh` to push code to GitHub.
```

技能可以绑定到团队（`team.yaml`）或角色（`role.yaml`）。修改技能后运行 `agencycli sync`。

---

## 智能体行动手册（Agent playbooks）

行动手册位于 `agent-playbooks/`，定义智能体在队列为空时的自主行为。在 `project.yaml` 中通过 `playbook:` 字段引用，并由 `project apply` 安装为 `wakeup.md`。

```
agent-playbooks/
  pm.md          ← PM 自主流程：扫描 issue、维护 backlog、请求人类确认
  qa-reviewer.md ← QA 自主流程：扫描 PR、评审、请求合并确认
```

调度器会在唤醒提示的顶部自动注入 **未读收件箱消息**，无需在 `wakeup.md` 中显式轮询。

---

## Docker 沙箱

```bash
agencycli hire --project my-api --team engineering --role developer \
  --model claudecode --name dev --sandbox docker
```

每次 `run` 或 `exec` 会启动一个全新容器，包含：

- 智能体工作目录挂载（读写）
- 项目仓库挂载（读写）
- 机构工作区根目录挂载（智能体可在容器内调用 `agencycli`）
- `agencycli` 可执行文件只读挂载到 `/usr/local/bin/agencycli`
- 凭据自动挂载（`~/.claude`、`~/.config/gh`、`~/.ssh`、`~/.codex`、`~/.gemini`、`~/.cursor`）
- 常见 API Key 以环境变量方式传递
- Claude Code：以 root 执行并附带 `IS_SANDBOX=1 --dangerously-skip-permissions`
- Codex：`CODEX_UNSAFE_ALLOW_NO_SANDBOX=1`

如需中国镜像构建：

```bash
docker build --build-arg CN_MIRROR=1 -t agencycli/sandbox-claudecode docker/sandbox-claudecode/
```

---

## Roadmap

### 上下文管理 ✓
- [x] 机构 / 团队 / 项目 / 角色骨架
- [x] 上下文合并：`agency → team chain → role → project`
- [x] 全模型格式化（claudecode、codex、cursor、gemini、qoder、opencode、iflow、generic-cli）
- [x] 基于 SHA-256 的 `sync` 变更检测
- [x] `hire` / `assign` 别名 / `fire`（软删除 + 硬删除）
- [x] 技能支持附带文件 + `{{SKILL_DIR}}` 解析
- [x] `add_dirs` —— 项目仓库对智能体可见
- [x] `--dir` 全局参数

### 任务自动化 ✓
- [x] 每智能体任务队列（7 状态机、优先级排序）
- [x] 人类收件箱：confirm / reject / comment / forward
- [x] 异步消息：`inbox send / messages / reply`（非阻塞、任意参与者）
- [x] `run` / `exec`
- [x] 心跳循环会话连续性
- [x] 心跳调度器（active-hours / active-days）
- [x] 唤醒流程（`wakeup.md`）—— 队列为空时执行
- [x] 未读消息自动注入唤醒提示
- [x] Cron 定时任务（`cron add/list/delete/enable/disable`）
- [x] Docker 沙箱
- [x] `task stop-all` —— 紧急停止运行/待办任务
- [x] `task tokens` —— 每任务/每智能体 token 与成本统计

### 智能体行动手册 ✓
- [x] `agent-playbooks/` 目录
- [x] `project.yaml` AgentSpec 中的 `playbook:` 字段
- [x] `project apply` 安装 playbook 为 `wakeup.md` 并设置 `wakeup_prompt`
- [x] playbook 随模板归档

### 项目蓝图 ✓
- [x] `project.yaml` —— 声明式智能体 + 心跳 + cron + playbook
- [x] `project show / apply / blueprints`
- [x] `create project --blueprint`

### 模板 ✓
- [x] `template pack` —— 归档机构为可分享 `.tar.gz`
- [x] `template info` —— 查看模板元数据
- [x] `create agency --template` —— 本地文件 / 目录 / HTTPS URL
- [x] `template.json` 元数据（name、version、author、email、description、keywords）
- [x] `agent-playbooks/` 与 `project-blueprints/` 包含在模板归档中

### 计划中
- [ ] `depends_on` 任务依赖解析
- [ ] 运行日志轮转
- [ ] E2B / Daytona 沙箱提供商

---

## 许可证

MIT
