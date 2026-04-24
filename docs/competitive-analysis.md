# 竞品分析：AgencyCLI vs HiClaw vs Molecule AI

> 基于 HiClaw v1.0.9 和 Molecule AI (molecule-core) 源码研究，提炼对 AgencyCLI 有借鉴价值的设计模式。
>
> 更新时间：2026-04-20

## 一、产品定位对比

| 维度 | AgencyCLI | HiClaw | Molecule AI |
|------|-----------|--------|-------------|
| 一句话定位 | 轻量级 AI Agent 团队管理 CLI | 企业级 K8s-native 多 Agent 协作平台 | Org-native 异构 Agent 团队治理平台 |
| 技术栈 | Go + React SPA + 文件存储 | Docker + Matrix IM + Higress + MinIO | Go/Gin + Next.js Canvas + Postgres + Redis |
| 运行时 | 本地进程 / Docker sandbox | Docker 容器（OpenClaw / CoPaw / ZeroClaw） | Python 统一镜像 + 6 种 Adapter |
| Agent 间通信 | Inbox 异步消息 | Matrix IM 协议 + @mention | A2A JSON-RPC 点对点 + WebSocket |
| 部署方式 | CLI 单机安装 | Helm / K8s Operator / Docker Compose | Docker Compose / Railway / Render |
| 组织层级 | Agency → Team → Role → Project → Agent | Manager → Team Leader → Worker | Org → Workspace（可嵌套 Team） |
| 前端 | 管理面板（列表/表单/文件浏览器） | Element Web（Matrix 聊天客户端） | 画布编辑器（React Flow 拖拽拓扑） |
| 开源协议 | Apache 2.0 | Apache 2.0 | BSL 1.1（2029 转 Apache 2.0） |

### 各自核心优势

| 产品 | 核心优势 |
|------|---------|
| **AgencyCLI** | 零依赖单二进制安装、CLI-first 工作流、轻量文件存储、多 provider 适配 |
| **HiClaw** | 网关级凭证隔离、Matrix 全程可见通信、K8s 声明式资源管理、容器级安全隔离 |
| **Molecule AI** | 分层记忆体系、技能进化循环、异构运行时统一治理、Canvas 可视化拓扑 |

---

## 二、借鉴点分析

按对 AgencyCLI 的实用价值分为三档。每个借鉴点标注灵感来源。

### 高价值（直接提升核心能力）

#### 1. 分层记忆架构 `来源: Molecule`

**竞品做法**：记忆按组织边界分 `LOCAL`（私有）、`TEAM`（团队共享）、`GLOBAL`（全局）三层。写入指定 scope，查询按层级搜索。记忆存事实（facts），技能存可重复流程（procedure），两者有明确晋升路径。

**落地方案**：

```
.agencycli/memory/
  global/         # 全局记忆（所有 agent 可读）
  teams/
    frontend/     # team 级记忆（team 内 agent 可读）
  agents/
    alice/        # agent 私有记忆
```

`agencycli memory add --scope local|team|global "key insight"`，agent 在 heartbeat 中按 scope 链搜索。文件目录即边界，无需数据库。

#### 2. 技能进化循环 `来源: Molecule`

**竞品做法**：反复出现的成功模式从 memory 晋升为 skill，晋升显式记录为平台事件。Skill 热加载到运行时，2-3 秒内生效。

```
任务执行 → 持久化洞察到记忆 → 重复成功成为信号 → 晋升为可复用 skill → 未来工作更快
```

**落地方案**：wakeup.md 模板引导 agent 将高频工作流写入 `skills/`。CLI 增加 `agencycli skill promote --from-memory <key>` 转换记忆为 skill 文件。

#### 3. 有限/无限任务区分 `来源: HiClaw`

**竞品做法**：任务分为 finite（一次性）和 infinite（持续/定期），各有独立状态机。infinite task 有严格的"触发后不重复记录"规则防止循环。

**落地方案**：task 增加 `mode: once | recurring` 字段。recurring 在 heartbeat 中检查触发条件但不重复创建子任务，与 cron 互补。

#### 4. Agent Card 能力声明 `来源: Molecule`

**竞品做法**：每个 workspace 暴露 `agent-card.json`（名称、skills、输入输出类型），Canvas 从 Card 渲染 UI，协作者注入对方 Card 到 system prompt。

**落地方案**：自动生成 `agent-card.json`：

```json
{
  "name": "alice", "role": "frontend-developer", "team": "frontend",
  "provider": "cursor", "skills": ["react", "typescript"],
  "status": "idle", "lastHeartbeat": "2026-04-20T10:00:00Z"
}
```

`agencycli agent card <name>` 查看；Web UI 渲染状态卡片；任务委托时参考 Card 选接收者。

#### 5. Team Leader 中间管理层 `来源: HiClaw`

**竞品做法**：Team Leader 是独立 agent，有自己的 heartbeat 和 state。Manager 只跟 Leader 对话，不直接指挥 worker。通信有防火墙。

**落地方案**：支持 team 内指定 leader role。leader 接收上级任务、内部拆解分发、汇总进度。上级只需 @leader，减少管理扇出。可选配置，不强制。

#### 6. 任务依赖图（DAG） `来源: HiClaw`

**竞品做法**：`plan.md` + `resolve-dag.sh` 做依赖解析，支持 validate / ready / status 三操作，实现并行调度。

**落地方案**：task 增加可选 `depends_on: [task_id, ...]`。`agencycli task list --ready` 只返回依赖已完成的任务。无依赖时行为不变，向后兼容。

#### 7. 组织图即通信策略 `来源: Molecule`

**竞品做法**：`CanCommunicate` 由层级自动推导：兄弟、父子可通信，跨级跨团队禁止。组织图就是安全策略。

**落地方案**：`agencycli inbox send` 增加可选 `--strict` 模式，按层级校验通信权限。默认宽松保持现有行为。

---

### 中等价值（增强治理和可维护性）

#### 8. 统一状态聚合视图 `来源: HiClaw`

**竞品做法**：核心状态集中在 `state.json`，只通过脚本修改，禁止手动编辑。

**落地方案**：提供只读的 `agency-state.json` 聚合快照，汇总各 agent 的 task 进度、活跃时间、heartbeat 状态。agent 在 heartbeat 中读此视图做自省。

#### 9. Bundle 导入/导出 `来源: Molecule`

**竞品做法**：导出 workspace 为 `.bundle.json`（skills、prompt、config、子 workspace 递归），导入时分配新 UUID 保留溯源。

**落地方案**：`agencycli export --bundle` 导出完整快照，`agencycli import --bundle` 恢复。与 template（蓝图）互补——bundle 是活体快照。

#### 10. 事件日志 `来源: Molecule`

**竞品做法**：`structure_events` append-only 不可变日志记录所有结构变更。`workspaces` 表是投影。

**落地方案**：每次写操作追加一行到 `.agencycli/events.jsonl`。`agencycli events [--since 1d]` 查看。低成本审计追踪。

#### 11. 声明式配置 `来源: HiClaw`

**竞品做法**：K8s 风格 YAML 定义 Worker/Team/Human，`hiclaw apply -f`，fsnotify → kine → controller reconciler。

**落地方案**：

```yaml
apiVersion: agencycli/v1
kind: Agency
spec:
  teams:
    - name: frontend
      roles:
        - name: developer
          agents:
            - name: alice
              provider: cursor
              skills: [react, typescript]
```

`agencycli apply -f agency.yaml` 差异化 reconcile。

#### 12. 乐观并发（任务锁） `来源: HiClaw`

**竞品做法**：`.processing` JSON 标记文件做任务抢占，含 agent ID、开始时间、过期时间。

**落地方案**：共享任务池场景下，task 目录放 `.lock` 文件（agent ID + TTL），agent 启动前尝试获取锁。

#### 13. 团队扩展/收缩 `来源: Molecule`

**竞品做法**：workspace 原地扩展为 team，保持外部身份不变。收缩时 handoff-to-memory 保留知识。

**落地方案**：`agencycli team expand` 扩展内部人员不改外部接口。`agencycli team collapse --keep-memory` 收缩保留知识。

#### 14. Skill references 子目录 `来源: HiClaw`

**竞品做法**：SKILL.md 引用 `references/*.md` 做深度文档分离，按需加载减少 token 消耗。

**落地方案**：context 合并支持 `references/` 子目录懒加载。

#### 15. 通信防噪规则 `来源: HiClaw`

**竞品做法**：Worker NO_REPLY 规则——"收到"不回复，只在有实质内容时 @mention，heartbeat 只报异常。

**落地方案**：默认 wakeup.md 模板内置 anti-noise 规则。

---

### 可选价值（特定场景有用）

#### 16. System Prompt 分层组装 `来源: Molecule`

**竞品做法**：按顺序组装：prompt files → parent context → 平台规则 → skill → peer Card → 委托处理。

**落地方案**：文档化 context 合并优先级（agent 私有 > team 共享 > agency 全局 > skills > 协作者描述）。

#### 17. Rules vs Skills 分离 `来源: Molecule`

**竞品做法**：rules 始终生效，skills 按需加载。

**落地方案**：context 下引入 `rules/`（始终注入）vs `skills/`（按需引用）子目录。

#### 18. 安全分级 `来源: Molecule`

**竞品做法**：T1-T4 安全等级影响容器参数，不影响通信协议。

**落地方案**：`agencycli hire --sandbox-tier <1-3>` 分级沙箱。

#### 19. 热加载 Skill 监听 `来源: Molecule`

**竞品做法**：fsnotify + 2s 防抖 + Agent Card 广播。

**落地方案**：agent wakeup 时重新读取最新 skills；长期可加 watch 目录实时合并。

#### 20. 外部 Agent 注册 `来源: Molecule`

**竞品做法**：`external: true` 不启动容器，通过公网 URL + bearer token 注册，走标准 heartbeat。

**落地方案**：`agencycli hire --external --url https://... --token xxx` 注册远程 agent。

#### 21. 文件显式同步 `来源: HiClaw`

**竞品做法**：MinIO 不自动同步，必须显式 `mc mirror` + 通知，避免竞态。

**落地方案**：`agencycli files sync <source> <target> [path]` 显式同步 + inbox 通知。

---

## 三、架构模式汇总

| 模式 | HiClaw | Molecule AI | AgencyCLI 可吸收形式 |
|------|--------|-------------|---------------------|
| 记忆体系 | — | LOCAL/TEAM/GLOBAL 分层 | 文件目录按 scope 分层 |
| 技能进化 | — | memory → skill promotion | promote 命令 + wakeup 引导 |
| 能力声明 | — | Agent Card JSON + WS 广播 | agent-card.json 自动生成 |
| 任务分类 | finite / infinite 状态机 | — | task mode: once / recurring |
| 任务依赖 | plan.md DAG 脚本 | — | depends_on 字段 |
| 中间管理 | Team Leader agent | 团队扩展保持外部身份 | leader role（可选） |
| 通信策略 | 防火墙 groupAllowFrom | CanCommunicate 层级推导 | --strict 模式 |
| 状态管理 | state.json 集中 | structure_events 投影 | 聚合视图 + events.jsonl |
| 声明式配置 | K8s CRD + controller | — | agency.yaml + apply |
| 并发控制 | .processing 标记 + TTL | — | .lock 文件 |
| 工作快照 | — | Bundle 导入/导出 | export/import --bundle |
| 防噪规则 | NO_REPLY + 只报异常 | — | 默认 wakeup 模板内置 |
| 文档分层 | SKILL.md + references | 5 层 prompt 组装 | references 懒加载 + 优先级文档 |
| 插件分类 | — | rules（始终）vs skills（按需） | rules/ vs skills/ 目录 |
| 安全分级 | — | T1-T4 Workspace Tiers | sandbox-tier 1-3 |
| 热加载 | — | fsnotify + 2s debounce | wakeup 重读 / watch |
| 外部 agent | — | external URL + bearer | hire --external |
| 文件同步 | MinIO 显式 mirror | Docker volume + files API | files sync 命令 |

---

## 四、不适合照搬的部分

### 基建过重
- **Matrix IM + Element Web**（HiClaw）—— AgencyCLI 的异步 inbox 更轻量
- **Postgres + Redis + Langfuse + Temporal**（Molecule）—— 文件存储足够
- **K8s Operator / Helm**（HiClaw）—— 单机 CLI 不需要
- **React Flow Canvas**（Molecule）—— 管理面板不需要画布编辑器

### 资源开销大
- **每 agent 一个 Docker 容器**（两者皆有）—— 500MB+/agent 太重
- **Higress AI Gateway**（HiClaw）—— 单机直连 API key 更简单
- **A2A JSON-RPC 点对点协议**（Molecule）—— inbox 异步够用

---

## 五、综合洞察

两个竞品的启示互补：

| 维度 | HiClaw 贡献 | Molecule AI 贡献 |
|------|------------|-----------------|
| 核心关注 | 运维纪律 —— 安全隔离、通信可见性 | 知识管理 —— 组织治理、记忆体系 |
| 任务管理 | 状态机 + 防循环 | 委托 + 审批 + 活动流 |
| 团队模式 | Leader 中间人（隔离） | 扩展/收缩（弹性） |
| 技能体系 | SKILL.md + references（按需深度） | 热加载 + promotion（进化） |
| 最大独特价值 | 防噪、并发锁、infinite task 分离 | 分层记忆、Agent Card、Bundle |

**AgencyCLI 的差异化定位**：不追求企业级基建的复杂度，而是从两者中提取**最适合单机/小团队场景的轻量模式**——用文件目录代替数据库，用 CLI 命令代替 K8s controller，用 inbox 代替实时协议，保持零依赖快速安装的核心优势。
