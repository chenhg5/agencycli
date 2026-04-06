AgencyCli v0.3.0 发布说明

主要更新:
- 知识库（docs）：文档索引 + 虚拟目录 + Web 端 Notion 风格浏览器，支持搜索/下载/URL 直链
- AI 助手：内置浮窗式 AI 助手，预加载 agencycli SKILL，流式对话 + 工具调用
- Cursor agent：沙箱全权限、token 消耗解析、日志查看器适配 tool_call/thinking
- 守护服务：agencycli service install/start/stop 一键管理
- 心跳增强：唤醒预设条件、实时日志、wakeup prompt 编辑即时生效
- UI 全面优化：深色模式对比度、面包屑导航、成员详情页重构

Bug 修复:
- task add 强制要求 --created-by，拒绝无效格式
- inbox reply 默认 from 修正
- wakeup.md 路径统一为 .agencycli/context/wakeup.md
- 版本对比去除 git-describe 后缀

下载地址:
- Gitee: https://gitee.com/cg33/agentorg/releases/v0.3.0
- GitHub: https://github.com/chenhg5/agencycli/releases/tag/v0.3.0

npm 安装（待 token 更新后生效）:
  npm install -g @agencycli/agencycli@0.3.0

如有问题请反馈。
