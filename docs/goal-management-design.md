# agencycli 目标管理系统设计

> Version: 0.1 Draft
> Status: Design Phase
> Scope: OKR、里程碑、目标对齐、进度追踪、多 Agent/人类协作中的目标驱动体系

---

## 1. 为什么需要目标管理

agencycli 当前有完善的 **任务执行层**（Task → Agent → Runner → 完成），但缺少 **目标导向层**。现有问题：

| 痛点 | 表现 |
|------|------|
| 任务缺乏方向 | Agent 每次唤醒处理队列中的任务，但不知道这些任务最终要达成什么 |
| 优先级无锚点 | Priority 0-3 是相对值，缺少「为了什么目标所以更优先」的上下文 |
| 进度不可度量 | 知道做了多少任务，不知道离目标还有多远 |
| 跨项目协同困难 | 多个项目的 Agent 各干各的，缺少 Agency 层面的统一方向 |
| 人机对齐不足 | 人类操作者和 Agent 没有共同的「北极星」指标 |

**核心原则**：目标系统必须服务于 agencycli 的 Agent 自主执行能力——目标不是给人看的报表，而是 Agent 决策的输入信号。

---

## 2. 参考框架分析

### 2.1 Plane (开源项目管理)

Plane 的层级结构：

```
Initiative (战略级) → Epic (跨周期大特性) → Module (主题分组) → Cycle (时间迭代) → Work Item (具体任务)
```

**可借鉴**：
- Initiative 到 Work Item 的逐级分解和进度上卷
- 状态更新系统（On Track / At Risk / Off Track）
- 多视图切换（List、Board、Timeline、Spreadsheet）

**不适用**：
- 过于 SaaS 化，层级太深（5 层），对 CLI-first 的 agencycli 太重
- 面向人类团队，没有 Agent 自主决策的考量

### 2.2 Linear (产品工程管理)

Linear 的方法论：

```
Strategic Initiative → Project → Cycle (sprint) → Issue
```

**可借鉴**：
- "Create Momentum, Don't Sprint" 哲学——持续节奏而非冲刺
- Triage 概念：新任务先进入分诊区，显式审查后才进入执行
- Cycle 自动滚转（未完成自动进入下一个 Cycle）
- 冷却期（Cooldown）：Cycle 之间留空处理技术债

**不适用**：
- 强交互性设计（Cmd+K、实时协作），Agent 无法直接使用

### 2.3 OKR 最佳实践

现代 OKR 方法论的共识：

- **3-5 个 Objective**，每个 2-4 个 Key Result
- **季度周期**设定，周/双周频率检查
- Key Result 必须可量化（不是「改善性能」而是「P95 延迟 < 200ms」）
- 结合 **CFR**（Conversations, Feedback, Recognition）持续沟通
- OKR 不是绩效考核工具，70% 达成率是健康的

### 2.4 CrewAI / 多 Agent 协作

多 Agent 框架的目标分解模式：

- **层级分解**：Manager Agent 拆解高层目标 → Worker Agent 执行子任务
- **上下文隔离**：子任务只拿到局部上下文，避免 context rot
- **反馈环路**：子任务结果经验证后再汇报，错误不向上传播
- **目标作为约束**：Agent 的自主决策以「是否推进目标」为判断依据

---

## 3. 设计方案

### 3.1 三层目标模型

结合 agencycli 的 Agency → Project → Agent 结构，设计三层目标模型：

```
Agency Vision (使命/愿景)
  └── Agency OKR (季度级)
        ├── Project Milestone (项目里程碑)
        │     ├── Task (具体任务)
        │     └── Task
        └── Project Milestone
              └── Task
```

每一层的职责：

| 层级 | 载体 | 周期 | 谁制定 | 谁执行 | 进度度量 |
|------|------|------|--------|--------|----------|
| **Vision** | `agency-prompt.md` | 长期 | 人类 | 注入所有 Agent 的上下文 | 定性描述 |
| **OKR** | `.agencycli/okrs.yaml` | 季度 | 人类（可 Agent 辅助） | Agency 全体 | Key Result 完成比 |
| **Milestone** | `projects/<name>/.agencycli/milestones.yaml` | 灵活 | 人类或 PM Agent | 项目成员 | 关联任务完成率 |
| **Task** | `agents/<name>/.agencycli/tasks.yaml` | 天/小时 | Agent 或人类 | 具体 Agent/人类 | 状态流转 |

### 3.2 数据结构

#### Vision（愿景）

不引入新实体。Agency 的愿景已体现在 `agency-prompt.md` 中。
扩展 `Agency` 结构增加可选的 vision 和 mission 字段：

```yaml
# .agencycli/agency.yaml
name: "TechStudio"
description: "AI-first software development studio"
lang: "zh"
vision: "让每个创意都能成为可运行的产品"
mission: "通过 AI Agent 团队，将软件开发成本降低 10 倍"
```

#### OKR

```yaml
# .agencycli/okrs.yaml
current_quarter: "2026-Q2"
okrs:
  - id: "okr-2026q2-01"
    objective: "建立可靠的多 Agent 自主运营体系"
    owner: "human"                    # 或 "project/agent"
    quarter: "2026-Q2"
    status: "on_track"               # on_track | at_risk | off_track | achieved
    key_results:
      - id: "kr-01"
        description: "Agent 任务成功率达到 90%"
        metric_type: "percentage"     # percentage | number | boolean | currency
        target_value: 90
        current_value: 72
        unit: "%"
        linked_milestones:            # 哪些里程碑推动这个 KR
          - "cc-connect/ms-stable-release"

      - id: "kr-02"
        description: "每周自动完成 50+ 任务（无需人工干预）"
        metric_type: "number"
        target_value: 50
        current_value: 33
        unit: "tasks/week"

      - id: "kr-03"
        description: "客户支持响应时间 < 5 分钟"
        metric_type: "number"
        target_value: 5
        current_value: 12
        unit: "minutes"
        linked_milestones:
          - "cc-connect/ms-im-integration"
          - "core/ms-auto-response"

    created_at: "2026-04-01T00:00:00Z"
    updated_at: "2026-04-11T00:00:00Z"
    review_notes:
      - date: "2026-04-07"
        note: "KR-01 受 thinking signature 问题影响，已修复"
        author: "human"
```

KR 的 `metric_type` 支持以下类型：

| 类型 | 示例 | 进度计算 |
|------|------|----------|
| `percentage` | 成功率 90% | `current / target * 100` |
| `number` | 完成 50 个任务 | `current / target * 100` |
| `boolean` | 是否上线 | 0% 或 100% |
| `currency` | 营收 $10K | `current / target * 100` |

Objective 的整体进度 = 其 Key Results 进度的加权平均（默认等权）。

#### Milestone（里程碑）

```yaml
# projects/cc-connect/.agencycli/milestones.yaml
milestones:
  - id: "ms-stable-release"
    title: "CC-Connect v1.0 稳定版发布"
    description: "完成所有核心功能，通过完整测试，可公开发布"
    status: "in_progress"             # planned | in_progress | completed | cancelled
    due_date: "2026-05-01"
    owner: "cc-connect/pm"            # 负责人
    progress: 65                      # 0-100，可手动设置或自动计算

    # 完成标准（Definition of Done）
    criteria:
      - "所有 P0/P1 Bug 已修复"
      - "API 文档覆盖率 100%"
      - "集成测试通过率 > 95%"
      - "性能基准符合 SLA"

    # 关联的 OKR Key Result
    linked_kr: ["okr-2026q2-01/kr-01"]

    # 关联的任务标签——里程碑通过标签关联任务
    task_labels: ["v1.0", "release-blocker"]

    created_at: "2026-04-01T00:00:00Z"
    updated_at: "2026-04-11T00:00:00Z"

  - id: "ms-im-integration"
    title: "全平台 IM 接入"
    description: "支持飞书、微信、Telegram、Discord 全渠道接入"
    status: "in_progress"
    due_date: "2026-04-20"
    owner: "cc-connect/dev-claude"
    progress: 40
    criteria:
      - "飞书扫码接入可用"
      - "微信扫码接入可用"
      - "Telegram Bot Token 接入可用"
      - "Discord Bot 接入可用"
    linked_kr: ["okr-2026q2-01/kr-03"]
    task_labels: ["im-integration"]
```

#### Task 扩展

在现有 Task 结构中添加可选的目标关联字段：

```yaml
# 现有 Task 新增字段
milestone_id: "ms-stable-release"     # 所属里程碑
okr_kr_id: "okr-2026q2-01/kr-01"     # 直接关联的 KR（可选）
```

### 3.3 进度上卷机制

进度从下往上自动汇聚：

```
Task 完成
  → Milestone 进度更新（按关联任务完成率计算）
    → KR current_value 更新（按关联 Milestone 进度计算，或直接从指标源拉取）
      → Objective 进度更新（KR 加权平均）
```

两种进度计算模式：

| 模式 | 适用场景 | 计算方式 |
|------|----------|----------|
| **任务驱动** | Milestone 进度 | `completed_tasks / total_tasks * 100` |
| **指标驱动** | KR 进度 | 从外部指标源（API、数据库、手动输入）获取 `current_value` |

Milestone 可以选择自动计算或手动覆盖进度值。

### 3.4 Agent 如何感知目标

**核心问题**：Agent 怎么知道当前的目标？怎么用目标指导决策？

#### 方案：上下文注入 + 唤醒增强

1. **上下文注入**：`agencycli sync` 时将相关目标摘要注入到 Agent 的上下文文件中

```markdown
<!-- 自动生成，嵌入 CLAUDE.md / AGENTS.md 尾部 -->
## Current Goals

### Agency OKR (2026-Q2)
- O1: 建立可靠的多 Agent 自主运营体系
  - KR1: 任务成功率 90% (当前: 72%) ⚠️
  - KR2: 每周 50+ 自动任务 (当前: 33) ⚠️
  - KR3: 响应时间 < 5min (当前: 12min) 🔴

### Project Milestones
- [65%] v1.0 稳定版发布 (Due: 2026-05-01)
- [40%] 全平台 IM 接入 (Due: 2026-04-20) ⚠️ 即将到期
```

2. **唤醒增强**：Agent 唤醒时，系统消息中包含目标上下文

```
你的当前目标优先级：
1. [里程碑] 全平台 IM 接入（截止 4/20，当前 40%）← 即将到期，优先处理
2. [KR] 任务成功率达到 90%（当前 72%）
3. [里程碑] v1.0 稳定版发布（截止 5/1，当前 65%）

请优先处理与上述目标相关的任务。
```

3. **任务创建引导**：PM Agent 或人类创建任务时，系统提示关联到里程碑

```bash
agencycli task add \
  --project cc-connect --agent dev-claude \
  --title "实现 Telegram Bot 接入" \
  --milestone ms-im-integration \
  --priority 1
```

### 3.5 时间节奏

参考 Linear 的 Cycle 和 OKR 的 CFR，建议以下节奏：

| 周期 | 活动 | 参与者 | 自动化程度 |
|------|------|--------|------------|
| **每日** | Agent 唤醒时检查目标进度 | Agent | 全自动 |
| **每周** | 周报生成（进度汇总、风险标记） | PM Agent | 半自动 |
| **双周** | 里程碑 Review（调整优先级） | 人类 + PM Agent | 人类主导 |
| **每季** | OKR 复盘和制定 | 人类 | 人类主导，Agent 辅助数据收集 |

可以利用现有的 **Cron 系统** 驱动：

```yaml
# project.yaml 中的 cron 配置
crons:
  - id: weekly-goal-review
    schedule: "0 9 * * 1"    # 每周一 9:00
    title: "Weekly Goal Review"
    prompt: |
      生成本周目标进度报告。检查：
      1. 各 Milestone 进度变化
      2. KR 指标变化
      3. 风险项（进度落后的目标）
      4. 下周建议优先事项
      将报告发送到人类收件箱。
```

---

## 4. CLI 命令设计

### 4.1 OKR 管理

```bash
# 查看当前 OKR
agencycli okr list
agencycli okr show okr-2026q2-01

# 创建/编辑 OKR
agencycli okr create --quarter 2026-Q2 --objective "建立可靠的多 Agent 自主运营体系"
agencycli okr add-kr --okr okr-2026q2-01 \
  --description "任务成功率达到 90%" \
  --metric percentage --target 90

# 更新 KR 进度
agencycli okr update-kr --kr kr-01 --value 78

# OKR 状态标记
agencycli okr status --okr okr-2026q2-01 --status at_risk

# 查看全景
agencycli okr overview
```

### 4.2 里程碑管理

```bash
# 项目里程碑
agencycli milestone list --project cc-connect
agencycli milestone create --project cc-connect \
  --title "v1.0 稳定版发布" \
  --due-date 2026-05-01 \
  --owner cc-connect/pm

# 关联任务标签
agencycli milestone add-label --id ms-stable-release --label "v1.0"

# 更新进度
agencycli milestone update --id ms-stable-release --progress 70

# 关联到 KR
agencycli milestone link-kr --id ms-stable-release --kr okr-2026q2-01/kr-01

# 查看进度
agencycli milestone show ms-stable-release
```

### 4.3 任务关联

```bash
# 创建任务时关联里程碑
agencycli task add \
  --project cc-connect --agent dev-claude \
  --title "Telegram Bot 接入" \
  --milestone ms-im-integration

# 给现有任务关联里程碑
agencycli task update --id t-20260411-abc123 --milestone ms-im-integration
```

---

## 5. Web UI 设计

### 5.1 新增页面/模块

| 页面 | 位置 | 功能 |
|------|------|------|
| **OKR 看板** | 全局导航（Agency 级） | 显示当前季度所有 OKR，进度仪表盘，KR 指标趋势图 |
| **里程碑视图** | 项目详情页新增 Tab | 时间线/甘特图展示里程碑，点击展开关联任务 |
| **目标关联** | 任务创建/编辑弹窗 | 下拉选择关联里程碑和 KR |
| **进度总览** | 工作台首页 | 目标健康度仪表盘（绿/黄/红） |

### 5.2 OKR 看板布局

```
┌──────────────────────────────────────────────────────┐
│  2026-Q2 OKR Overview                    [+ New OKR] │
├──────────────────────────────────────────────────────┤
│                                                      │
│  O1: 建立可靠的多 Agent 自主运营体系          72%    │
│  ├─ KR1: 任务成功率 90%         ████████░░  72%  ⚠️ │
│  ├─ KR2: 每周 50+ 自动任务      ██████░░░░  66%  ⚠️ │
│  └─ KR3: 响应时间 < 5min        ███░░░░░░░  42%  🔴 │
│      └─ Milestones: IM 接入(40%), 自动响应(30%)      │
│                                                      │
│  O2: 拓展到 3 个付费客户                     35%    │
│  ├─ KR1: 完成产品文档网站        █████░░░░░  50%     │
│  ├─ KR2: 3 个客户 POC            ██░░░░░░░░  33%  ⚠️ │
│  └─ KR3: NPS > 8                 ░░░░░░░░░░  0%   🔴 │
│                                                      │
└──────────────────────────────────────────────────────┘
```

### 5.3 里程碑时间线

```
Apr 2026                        May 2026
|----|----|----|----|----|----|----|----|
     ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░         ← v1.0 稳定版 (65%)
     ▓▓▓▓▓▓░░░░                     ← IM 接入 (40%) ⚠️ due 4/20
          ▓▓▓▓▓▓▓▓▓▓▓░░░░░         ← 自动响应 (30%)
                         ▓░░░░░░░░  ← 文档网站 (15%)
```

---

## 6. 文件系统布局

在现有 workspace layout 中新增以下文件：

```
MyAgency/
  .agencycli/
    agency.yaml              # 扩展 vision/mission 字段
    okrs.yaml                # Agency 级 OKR（当前 + 历史）
    okrs_archive.yaml        # 已归档的历史 OKR

  projects/
    cc-connect/
      .agencycli/
        milestones.yaml      # 项目里程碑定义
      agents/
        dev-claude/
          .agencycli/
            tasks.yaml       # Task 扩展 milestone_id 字段
            context/
              goals.md       # 自动生成的目标上下文（sync 时更新）
```

---

## 7. 实施路线

### Phase 1: 数据模型 + CLI（1-2 周）

- [ ] 扩展 Agency 结构（vision/mission）
- [ ] 定义 OKR 数据结构和 Store 层
- [ ] 定义 Milestone 数据结构和 Store 层
- [ ] 扩展 Task 结构（milestone_id, okr_kr_id）
- [ ] 实现 `agencycli okr` 子命令族
- [ ] 实现 `agencycli milestone` 子命令族
- [ ] `agencycli sync` 注入目标上下文到 Agent

### Phase 2: 进度追踪 + Agent 集成（1-2 周）

- [ ] 进度上卷计算逻辑
- [ ] 任务完成时自动更新 Milestone 进度
- [ ] Agent 唤醒消息中注入目标优先级
- [ ] PM Agent 周报模板（Cron 驱动）
- [ ] OKR/Milestone 的 REST API 端点

### Phase 3: Web UI（2-3 周）

- [ ] OKR 看板页面
- [ ] 里程碑时间线视图
- [ ] 任务创建时关联里程碑
- [ ] 进度总览仪表盘
- [ ] 目标健康度通知

### Phase 4: 智能化（持续迭代）

- [ ] KR 指标自动采集（从 telemetry 计算任务成功率等）
- [ ] 风险预警（进度偏离自动通知）
- [ ] OKR 制定辅助（Agent 基于历史数据建议 KR 目标值）
- [ ] 里程碑自动创建（PM Agent 根据 OKR 分解）

---

## 8. 关键设计决策

### Q1: 为什么不直接用 Plane/Linear 而要自建？

agencycli 的核心场景是 **Agent 自主执行** + **人机协作**。现有 PM 工具都假设执行者是人类——通知、界面、交互都是为人设计的。agencycli 需要：
- 目标以文件形式存在，Agent 可以直接读取和理解
- 进度更新可以完全自动化（任务完成 → Milestone 更新 → KR 更新）
- 目标信息可以注入到 Agent 的上下文中，影响其决策
- 不依赖外部服务，保持 local-first 原则

### Q2: 为什么选 OKR 而不是 KPI？

OKR 强调方向和挑战性（"stretch goals"），适合探索性的 AI Agent 团队。KPI 强调达标，适合稳定运营。agencycli 目前处于快速发展阶段，OKR 更合适。后续可以增加 KPI 模式（把 KR 的 target 改为 baseline + stretch）。

### Q3: 为什么里程碑在项目级而不是 Agency 级？

里程碑是 **交付物导向** 的——"什么时候发布 v1.0"、"什么时候完成 IM 接入"。这些和具体项目强绑定。跨项目的战略对齐由 OKR 层完成——一个 KR 可以关联多个项目的多个里程碑。

### Q4: Agent 能不能自己修改目标？

初版设计中 Agent 只能 **读取** 目标和 **更新进度**，不能修改目标本身（Objective、KR 描述、Milestone 定义）。目标的设定权保留给人类或被授权的 PM Agent（通过 RBAC 控制）。后续可以根据实际使用逐步放开。

### Q5: 如何处理目标冲突？

当多个目标竞争资源时：
1. OKR 优先级（Objective 的排列顺序即优先级）
2. 里程碑截止日期（到期时间近的优先）
3. 任务优先级（0=critical > 3=low）

Agent 唤醒时注入的目标上下文已经按此顺序排列，Agent 自然会优先处理排在前面的目标相关任务。

---

## 9. 与现有系统的关系

```
                    ┌─────────────┐
                    │   Vision    │  agency-prompt.md
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │     OKR     │  .agencycli/okrs.yaml
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
       ┌──────▼──────┐  ┌─▼──┐  ┌─────▼─────┐
       │  Milestone   │  │ MS │  │ Milestone  │  projects/*/.agencycli/milestones.yaml
       └──────┬──────┘  └─┬──┘  └─────┬─────┘
              │            │            │
         ┌────▼────┐  ┌───▼───┐  ┌────▼────┐
         │  Tasks  │  │ Tasks │  │  Tasks  │  agents/*/.agencycli/tasks.yaml
         └────┬────┘  └───┬───┘  └────┬────┘
              │            │            │
         ┌────▼────┐  ┌───▼───┐  ┌────▼────┐
         │  Agent  │  │ Agent │  │  Human  │  执行者
         └─────────┘  └───────┘  └─────────┘

  ─── 已有 ───────────    ─── 新增 ───────────
  Agency, Project,         OKR, Milestone,
  Team, Role, Agent,       进度追踪,
  Task, Message, Inbox     目标注入
```

本设计保持了 agencycli 「文件即数据」的核心理念——所有目标信息都是 YAML 文件，可以 git 管理，Agent 可以直接读取，不依赖任何外部服务。
