AgencyCli v0.4.1 发布说明

主要更新:

🔐 环境变量管理
- 全新工作区环境变量（envvars）：支持全局或指定 Agent 作用域
- 运行时自动注入，优先级链：全局 → Agent 级 → Provider → 独立 Agent 环境变量
- CLI 命令：agencycli envvar add/list/remove + agent set-env/unset-env/list-env
- Web 设置页：按项目分组的 Agent 多选器，环境变量增删改查
- Agent 详情页：环境变量面板，敏感值遮盖 + 👁 切换查看，内联编辑

⚡ 事件触发调度
- 新增触发式调度：收到消息或分配任务时自动唤醒 Agent
- CLI 和 Web 心跳编辑器均可配置触发器
- 去重执行 + 可配置冷却时间

📊 工作台增强
- 项目调度概览卡片：Agent 数、运行中数、调度状态、任务/消息数
- 支持单项目开关调度 + "全部启动"按钮
- 任务 Tab 显示待处理数量
- 项目卡片显示运行中 Agent 数

📚 知识库改进
- 文档全屏模式：隐藏导航居中显示，ESC 退出
- 修复代码块复制 [object Object] 问题
- 文档按创建时间排序
- 日期显示遵循 i18n 格式

🎯 多层级 OKR
- OKR 层级体系：全局、项目、团队、Agent 四级，支持父子关联
- Scope Tab 切换 + 项目级筛选
- Agent 级 OKR 支持下拉选择 Agent
- KR 目标值显示优化：0/10000 (元/月)

Bug 修复:
- 修复任务完成/取消后显示重复条目
- 修复心跳编辑弹框超出页面
- 修复非活跃日调度器显示"待激活"无下次时间
- 运行详情显示失败原因，状态列完整 i18n
- 工作台概览配色统一（蓝色常规，绿色仅未处理项）
- 修复 wakeup condition 命令注入安全漏洞

下载地址:
- Gitee: https://gitee.com/cg33/agentorg/releases/v0.4.1
- GitHub: https://github.com/chenhg5/agencycli/releases/tag/v0.4.1

npm 安装:
  npm install -g @agencycli/agencycli@0.4.1

如有问题请反馈。
