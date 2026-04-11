AgencyCli v0.4.0 发布说明

主要更新:

🎯 目标管理（OKR & 里程碑）
- OKR 体系：Objective + Key Results，支持数值/百分比/布尔/金额四种度量类型
- 里程碑管理：项目级里程碑，含完成标准、任务标签、截止日期
- Web 页面：OKR 仪表盘 + 项目里程碑面板，支持内联编辑、创建/编辑/删除
- CLI 命令：agencycli okr / agencycli milestone 完整子命令
- Agent 上下文注入：自动将活跃目标摘要注入 Agent prompt

👥 多用户与权限
- 多用户支持：人员管理页面，支持创建/编辑/删除人员账号
- RBAC 权限模型设计
- 人员详情页：支持编辑 email/头像/电话/简介等字段

💬 IM 平台对接（cc-connect）
- 集成 cc-connect：通过 API 代理连接飞书/微信等 IM 平台
- Web 端配置面板：Settings 页面一站式配置 cc-connect
- Agent IM 连接面板：在 Agent 详情页扫码绑定 IM 账号

📋 任务与工作台增强
- 看板视图：任务页面支持列表/看板切换，拖拽改状态
- 批量操作：任务批量取消/归档/删除
- 工作台看板：Workbench 消息/任务统一看板
- Fire/移除成员功能

🔧 调度与运维
- 心跳会话管理：SessionID 跟踪、Context Usage 统计
- 统一 API Provider 管理：Web 端配置 API 密钥和 Base URL
- AI 助手权限交互：工具调用的 allow/deny/allow-all 控制
- 运行记录追踪实际 API Model 和 Base URL
- Ctrl+C 优雅停止调度器

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
- 修复工作台回复框隐藏问题
- 修复 cc-connect 项目名路径编码

下载地址:
- Gitee: https://gitee.com/cg33/agentorg/releases/v0.4.0
- GitHub: https://github.com/chenhg5/agencycli/releases/tag/v0.4.0

npm 安装:
  npm install -g @agencycli/agencycli@0.4.0

如有问题请反馈。
