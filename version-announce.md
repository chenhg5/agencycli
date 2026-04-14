AgencyCli v0.4.1 发布说明

自 v0.3.0 以来的完整更新：

🎯 目标管理（OKR & 里程碑）
- OKR 体系：Objective + Key Results，支持数值/百分比/布尔/金额四种度量类型
- 多层级 OKR：全局、项目、团队、Agent 四级，支持父子关联
- 里程碑管理：项目级里程碑，含完成标准、任务标签、截止日期
- Web 页面：OKR 仪表盘 + 里程碑面板，Scope Tab 切换、项目级筛选、内联编辑
- CLI 命令：agencycli okr / agencycli milestone 完整子命令
- Agent 上下文注入：自动将活跃目标摘要注入 Agent prompt

👥 多用户与权限
- 多用户支持：人员管理页面，支持创建/编辑/删除人员账号
- RBAC 权限模型设计
- 人员详情页：支持编辑 email/头像/电话/简介等字段
- Fire/移除成员功能

💬 IM 平台对接（cc-connect）
- 集成 cc-connect：通过 API 代理连接飞书/微信等 IM 平台
- Web 端配置面板：Settings 页面一站式配置 cc-connect
- Agent IM 连接面板：在 Agent 详情页扫码绑定 IM 账号

🔐 环境变量管理
- 全新工作区环境变量（envvars）：支持全局或指定 Agent 作用域
- 运行时自动注入，优先级链：全局 → Agent 级 → Provider → 独立 Agent 环境变量
- CLI 命令：agencycli envvar add/list/remove + agent set-env/unset-env/list-env
- Web 设置页：按项目分组的 Agent 多选器，环境变量增删改查
- Agent 详情页：环境变量面板，敏感值遮盖 + 切换查看，内联编辑

⚡ 调度与运维
- 事件触发调度：收到消息或分配任务时自动唤醒 Agent，去重执行 + 可配置冷却
- 心跳会话管理：SessionID 跟踪、Context Usage 统计
- 统一 API Provider 管理：Web 端配置 API 密钥和 Base URL
- 运行记录追踪实际 API Model 和 Base URL
- Ctrl+C 优雅停止调度器

📋 任务与工作台
- 看板视图：任务页面支持列表/看板切换，拖拽改状态
- 批量操作：任务批量取消/归档/删除
- 工作台看板：消息/任务统一看板
- 项目调度概览卡片：Agent 数、运行中数、调度状态、任务/消息数
- 支持单项目开关调度 + "全部启动"按钮
- 任务 Tab 显示待处理数量

📚 知识库改进
- 文档全屏模式：隐藏导航居中显示，ESC 退出
- 修复代码块复制 [object Object] 问题
- 文档按创建时间排序
- 日期显示遵循 i18n 格式

🤖 AI 助手
- 交互权限控制：工具调用的 allow/deny/allow-all 控制

🎨 UI 与体验
- 页面按钮风格统一（outline 设计语言）
- 日期显示遵循 i18n locale 习惯
- 深色模式全面适配
- 分页、面包屑、toast 通知
- 消息详情弹窗内直接回复

Bug 修复:
- 修复 Claude thinking signature 校验错误自动重试
- 修复 Codex Docker 沙箱 seccomp 权限
- 修复知识库三级目录导航
- 修复调度器 ActiveDays 配置不生效
- 修复任务完成/取消后显示重复条目
- 修复心跳编辑弹框超出页面
- 修复非活跃日调度器显示"待激活"无下次时间
- 运行详情显示失败原因，状态列完整 i18n
- 工作台概览配色统一
- 修复 wakeup condition 命令注入安全漏洞

下载地址:
- Gitee: https://gitee.com/cg33/agentorg/releases/v0.4.1
- GitHub: https://github.com/chenhg5/agencycli/releases/tag/v0.4.1

npm 安装:
  npm install -g @agencycli/agencycli@0.4.1

如有问题请反馈。
