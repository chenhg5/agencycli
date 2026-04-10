# 分布式部署架构方案

> 状态：Draft  
> 日期：2026-04-06

## 1. 背景

当前 agencycli 为单机架构——调度器、API、Runner、存储全部在一台机器上。随着 agent 数量增长和资源异构化需求（GPU、大内存、特定 CLI），需要支持跨机器分布式部署。

## 2. 现有架构

```
┌─ 单机 ─────────────────────────────┐
│  agencycli start                    │
│  ┌─────────┐  ┌──────────┐         │
│  │ API/Web  │  │ Scheduler│         │
│  └────┬─────┘  └────┬─────┘        │
│       │              │              │
│  ┌────▼──────────────▼────┐        │
│  │     Runner (local)      │        │
│  │  docker run / exec CLI  │        │
│  └────────────────────────┘        │
│  ┌────────────────────────┐        │
│  │  YAML + SQLite (local) │        │
│  └────────────────────────┘        │
└────────────────────────────────────┘
```

## 3. 核心挑战

| 挑战 | 说明 |
|------|------|
| 异构执行环境 | 不同 agent 需要不同硬件/CLI（GPU、Claude Code、Codex） |
| 上下文分发 | Agent 运行需要 context layers（agency → team → role → project），跨机器需同步 |
| 状态共享 | Task queue、message inbox、run records 需全局可见 |
| 日志收集 | 远端运行的日志需实时流回控制面 |
| 密钥管理 | API keys 不能明文传输到 worker |

## 4. 推荐架构：Control Plane + Worker Nodes

```
┌─ Control Plane ────────────────────┐     ┌─ Worker Node A (GPU) ───┐
│  ┌─────────┐  ┌──────────┐         │     │  agencycli worker       │
│  │ API/Web  │  │ Scheduler│         │     │   labels: gpu=4,        │
│  └────┬─────┘  └────┬─────┘        │     │           cli=claude    │
│       │              │              │◄───►│  pull context → run     │
│  ┌────▼──────────────▼────┐        │ WS  │  stream logs → control  │
│  │    PostgreSQL / SQLite  │        │     └────────────────────────┘
│  └────────────────────────┘        │
│  ┌────────────────────────┐        │     ┌─ Worker Node B (CPU) ───┐
│  │   Object Store (S3/    │        │     │  agencycli worker       │
│  │   local / MinIO)       │        │◄───►│   labels: cli=codex     │
│  └────────────────────────┘        │ WS  │  pull context → run     │
└────────────────────────────────────┘     └────────────────────────┘
```

## 5. 分层设计

### 5.1 P0: 存储抽象

将存储逻辑从业务代码解耦，通过 interface 支持多后端。

```go
type Store interface {
    // Projects
    ListProjects() ([]Project, error)
    GetProject(name string) (*Project, error)
    
    // Tasks
    CreateTask(t *Task) error
    ListTasks(filter TaskFilter) ([]Task, error)
    UpdateTask(id string, update TaskUpdate) error
    
    // Messages
    SendMessage(m *Message) error
    ListMessages(filter MsgFilter) ([]Message, error)
    
    // Runs
    RecordRun(r *RunRecord) error
    ListRuns(filter RunFilter) ([]RunRecord, error)
    
    // Agents
    ListAgents(project string) ([]Agent, error)
    GetAgent(project, name string) (*Agent, error)
}
```

配置方式：

```yaml
# agency.yaml
storage:
  driver: local        # local | postgres
  # postgres:
  #   dsn: postgres://user:pass@host:5432/agencycli
```

**影响最小化**：`local` driver 继续使用 YAML + SQLite，现有单机部署零改动。

### 5.2 P1: 上下文管理

Context Bundle = 分层上下文的打包产物，按 content hash 寻址。

```
context-bundle = hash(agency.md + team.md + role.md + agent.md + docs/*)
```

流程：
1. Control Plane 构建 context bundle → 上传 Object Store → 记录 hash
2. Worker 拉取 bundle（本地缓存，hash 未变则跳过）
3. Agent 运行时挂载 bundle 到工作目录

增量同步：只传 diff（基于文件 hash tree），大幅减少网络开销。

### 5.3 P2: Worker 协议

Worker 是一个轻量守护进程 `agencycli worker`：

```
agencycli worker --control-plane wss://control.example.com \
                 --token <worker-token> \
                 --labels "gpu=4,cli=claude,cli=codex,region=cn"
```

通信协议（WebSocket / gRPC）：

| 消息类型 | 方向 | 说明 |
|----------|------|------|
| `register` | Worker → CP | 注册 + 上报 labels/capacity |
| `heartbeat` | Worker → CP | 定期心跳 + 资源使用率 |
| `assign_task` | CP → Worker | 分配运行任务 |
| `log_stream` | Worker → CP | 实时日志流 |
| `run_result` | Worker → CP | 运行结果 + token 统计 |
| `cancel_task` | CP → Worker | 取消运行 |

密钥管理：
- API keys 不直接传给 worker
- Worker 启动时拉取加密的 provider credentials
- 或由 Control Plane 通过 secure channel 注入环境变量

### 5.4 P3: 调度策略

Agent 配置中增加 `worker_selector`：

```yaml
# agent.yaml
model: claude-code
worker_selector:
  cli: claude
  region: cn
  prefer: gpu    # soft preference
```

调度逻辑：
1. 匹配 `required` labels → 筛选候选 workers
2. 按 `prefer` labels → 排序
3. 按负载（running tasks / capacity）→ 选最优
4. 如无远端 worker 匹配 → fallback 到 Control Plane 本地执行

### 5.5 P4: 高可用（可选）

- Control Plane：多副本 + 共享 PostgreSQL
- Scheduler：leader election（基于 PostgreSQL advisory lock）
- Worker：无状态，随时扩缩

## 6. 渐进实施路线

| 阶段 | 内容 | 周期 | 前置 |
|------|------|------|------|
| P0 | Store interface + local driver | 1 周 | 无 |
| P0.5 | PostgreSQL driver | 1 周 | P0 |
| P1 | Context bundling + Object Store | 1 周 | P0 |
| P2 | Worker 协议 + worker CLI | 2 周 | P0, P1 |
| P3 | 调度策略 + selector 配置 | 1 周 | P2 |
| P4 | HA + 多副本 Control Plane | 2 周 | P0.5, P2 |

## 7. 对现有部署的影响

- **P0 做完后**，现有单机部署完全不受影响（`driver: local` 默认值）
- **P1-P3** 为增量能力，不启用 worker 模式则不影响
- **迁移路径**：`local` → `postgres` 提供一键迁移脚本

## 8. 与人员管理的关系

分布式架构下，`users.json` 需迁移到共享存储（PostgreSQL），确保多节点一致性。P0 的 Store interface 自然覆盖这一点。

## 9. 待决策

- [ ] Worker 与 Control Plane 之间使用 WebSocket 还是 gRPC？
- [ ] Object Store 选型：S3 兼容（MinIO）还是简单文件同步？
- [ ] 密钥管理：是否引入 Vault 或简单 AES 加密？
- [ ] 是否需要支持 agent 在 worker 间迁移（session 续连）？
