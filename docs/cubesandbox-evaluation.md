# CubeSandbox 沙箱方案评估

> 评估 CubeSandbox（腾讯云开源 MicroVM 沙箱）替代 Docker 作为 AgencyCLI agent 执行环境的可行性。
>
> 基于 CubeSandbox 源码研究，2026-04-20

## 一、CubeSandbox 是什么

CubeSandbox 是基于 RustVMM + KVM 的高性能安全沙箱服务，为 AI Agent 提供硬件级隔离的代码执行环境。核心特点：

- **MicroVM 架构**：每个沙箱是一个轻量虚拟机，拥有独立内核、文件系统、网络栈
- **快照克隆启动**：模板预热后通过内存快照恢复，冷启动 < 60ms
- **极低内存开销**：CoW 技术复用内存，每实例额外开销 < 5MB
- **E2B SDK 兼容**：替换一个 URL 环境变量即可从 E2B Cloud 迁移
- **eBPF 网络隔离**：CubeVS 在内核空间强制执行每沙箱的网络策略
- **已在腾讯云生产环境验证**

### 架构组件

| 组件 | 职责 |
|------|------|
| CubeAPI | Rust Axum REST 网关，E2B 兼容 |
| CubeMaster | 集群编排调度器 |
| CubeProxy | 反向代理，路由请求到沙箱实例 |
| Cubelet | 单节点沙箱生命周期管理 |
| CubeVS | eBPF 虚拟交换机，内核级网络隔离 |
| CubeHypervisor | KVM MicroVM 管理（基于 Cloud Hypervisor） |
| CubeShim | containerd Shim v2 API 集成 |

## 二、与 Docker 的正面对比

### 性能指标

| 指标 | Docker | CubeSandbox | 对比 |
|------|--------|-------------|------|
| **隔离级别** | 低（共享内核 Namespace） | 极高（独立内核 + eBPF） | CubeSandbox 远胜 |
| **冷启动** | ~200ms | **< 60ms**（快照恢复） | CubeSandbox 3x 快 |
| **内存开销** | 低（共享内核）但每容器 ~50-100MB | **< 5MB**（CoW 极致复用） | CubeSandbox 远胜 |
| **单机部署密度** | 高（数十-数百） | **极高（数千）** | CubeSandbox 远胜 |
| **50 并发创建** | 未测 | avg 67ms, P95 90ms, P99 137ms | 生产级 |

### 安全性

| 维度 | Docker | CubeSandbox |
|------|--------|-------------|
| 内核隔离 | 共享宿主内核（Namespace + cgroups） | 独立 Guest OS 内核 |
| 逃逸风险 | 存在已知容器逃逸漏洞（CVE 历史） | 硬件级隔离，逃逸难度极高 |
| 网络隔离 | iptables 规则（用户空间） | eBPF 内核空间强制执行 |
| 网络策略 | 手动配置 | 每沙箱 allow/deny CIDR，内置私网屏蔽 |
| Root 权限 | 容器内 root = 潜在风险 | Guest VM 内 root ≠ 宿主机权限 |

### 功能覆盖

| 能力 | Docker | CubeSandbox |
|------|--------|-------------|
| 执行代码 | `docker exec` | E2B SDK `sandbox.run_code()` |
| Shell 命令 | `docker exec sh -c` | E2B SDK `sandbox.commands.run()` |
| 文件读写 | Volume mount / `docker cp` | E2B SDK `sandbox.files.read/write()` |
| 宿主机目录挂载 | `-v` bind mount | `metadata.host-mount` (virtiofs) |
| 网络策略 | 手动 iptables | `allow_internet_access`, `allow_out`, `deny_out` |
| 暂停/恢复 | `docker pause/unpause` | `sandbox.pause()` / `sandbox.connect()`（含内存快照） |
| 浏览器自动化 | 需自行安装 | 内置 Chromium + CDP 支持 |
| 模板系统 | Dockerfile → Image | Image → Boot → Snapshot → Template |
| 集群支持 | Docker Swarm / K8s | CubeMaster 原生多节点 |

## 三、对 AgencyCLI 的适用性分析

### 优势

1. **安全性质的提升**：Agent 执行 LLM 生成的代码时，Docker Namespace 隔离不够安全（共享内核）。CubeSandbox 的独立内核消除了容器逃逸风险，这对运行不可信代码至关重要。

2. **启动速度更快**：当前 Docker sandbox 需要 ~200ms 启动。CubeSandbox < 60ms，对频繁创建/销毁沙箱的场景（如 code interpreter）优势明显。

3. **资源效率极高**：当前 Docker 每个 agent 容器 ~50-100MB 内存。CubeSandbox < 5MB 额外开销意味着单机可以运行数十倍更多的 agent 沙箱。

4. **E2B SDK 兼容**：已有成熟的 Python SDK，OpenClaw 集成示例完备，AgencyCLI 可以直接复用。

5. **网络安全开箱即用**：eBPF 内核级网络策略，per-sandbox allow/deny CIDR，自动屏蔽私网访问——比手动配 Docker iptables 规则简单且安全。

6. **暂停/恢复含内存快照**：Docker `pause` 只是冻结进程，CubeSandbox 的 `pause` 保存完整内存状态后释放资源，`connect` 恢复。适合长时间闲置的 agent 节省资源。

### 劣势

1. **环境要求高**：需要 KVM 支持的 x86_64 Linux（裸金属 / 支持嵌套虚拟化的 VM / WSL2）。Mac 原生不支持，云上普通 VM 实例通常也不支持 KVM。Docker 几乎在所有平台都能运行。

2. **部署复杂度高**：需要部署 CubeAPI + CubeMaster + Cubelet + CubeProxy + CubeVS 一套组件，比 `docker run` 重得多。虽有一键部署脚本，但维护成本远高于 Docker。

3. **模板制作成本**：Docker 只需 `Dockerfile → docker build`。CubeSandbox 需要 Image → MicroVM 冷启动 → 等待环境就绪 → Snapshot → 注册为 Template，流程更长。

4. **生态成熟度**：Docker 有极其丰富的镜像生态（Docker Hub）。CubeSandbox 的模板生态刚起步，需要自行从 OCI 镜像转换。

5. **调试体验**：Docker `exec -it` 交互式 shell 非常方便。CubeSandbox 通过 SDK API 执行命令，缺少交互式终端的直接体验。

6. **WSL2 下的体验**：虽然支持 WSL2，但需要嵌套虚拟化，性能和稳定性不如裸金属。AgencyCLI 用户中 WSL2 比例不低。

## 四、集成方案设计

如果要集成，推荐**双轨模式**——Docker 作为默认沙箱，CubeSandbox 作为高安全选项：

```
agencycli hire alice --sandbox docker       # 默认，兼容所有平台
agencycli hire alice --sandbox cube          # 高安全模式，需要 KVM
agencycli hire alice --sandbox none          # 无沙箱，本地进程
```

### 后端集成点

```go
// internal/sandbox/sandbox.go
type Sandbox interface {
    Create(config SandboxConfig) (string, error)
    Exec(id string, cmd []string) (ExecResult, error)
    ReadFile(id string, path string) ([]byte, error)
    WriteFile(id string, path string, data []byte) error
    Pause(id string) error
    Resume(id string) error
    Destroy(id string) error
}

type DockerSandbox struct { ... }      // 现有实现
type CubeSandbox struct { ... }        // 新增：通过 E2B SDK REST API 交互
```

### 配置层面

```yaml
# .agencycli/config.yaml
sandbox:
  provider: docker | cube | none
  cube:
    api_url: http://localhost:3000
    template_id: tpl-xxx
    allow_internet: true
    network:
      deny_out: ["192.168.0.0/16"]
```

## 五、结论与建议

### 推荐策略：**保持 Docker 为默认，CubeSandbox 作为可选高安全后端**

| 场景 | 推荐方案 | 理由 |
|------|---------|------|
| 个人开发者，Mac/Windows | Docker | 平台兼容性好，零额外部署 |
| 团队服务器，信任代码 | Docker | 够用，维护简单 |
| 运行 LLM 生成的不可信代码 | **CubeSandbox** | 硬件隔离消除逃逸风险 |
| 高并发 Agent 场景（RL 训练） | **CubeSandbox** | 极低内存开销，单机数千实例 |
| 生产环境多租户 | **CubeSandbox** | 内核级网络隔离 + 安全策略 |

### 不建议现阶段切换为 CubeSandbox-only

原因：
1. AgencyCLI 定位轻量 CLI，用户环境多样，KVM 要求会排除大量用户
2. Docker 的隔离级别对大多数 agent 任务已足够
3. CubeSandbox 的部署运维成本与 AgencyCLI "零依赖安装" 理念冲突

### 短期行动项

1. **抽象 Sandbox 接口**：将现有 Docker sandbox 代码抽象为接口，为后续接入 CubeSandbox 铺路
2. **跟踪 CubeSandbox 发展**：关注其 snapshot rollback（毫秒级回滚）功能，对 agent 试错/回退场景价值极大
3. **文档记录**：在 sandbox 配置文档中说明 CubeSandbox 选项，面向有裸金属服务器的高级用户
