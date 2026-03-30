v0.2.0 发布说明

本次为大版本更新，新增内置 Web 控制台，单二进制即可运行完整管理界面。

主要更新:
- 内置 Web 控制台：agencycli start 一键启动，无需单独部署前端
- 工作台：消息和任务的统一操作中心，支持批量操作
- 完整消息/任务管理：创建、编辑、筛选、查看执行日志
- 计划管理：Heartbeat / Cron / 运行状态三 tab 视图
- Web 端雇佣 Agent、创建角色、手动唤醒 Agent
- 会话管理：查看/切换作用域、重置会话
- 用户认证：用户名密码登录 + JWT
- 国际化：English, 简体中文, 繁體中文, 日本語
- CLI 新增 run / session reset / agent set-model 命令
- SQLite 遥测持久化

安装方式:
- npm: npm install -g @agencycli/agencycli@0.2.0
- 源码: git pull && make build && ./dist/agencycli start

下载地址:
- Gitee: https://gitee.com/cg33/agentorg/releases/v0.2.0
- GitHub: https://github.com/chenhg5/agencycli/releases/tag/v0.2.0
- npm: https://www.npmjs.com/package/@agencycli/agencycli

如有问题请反馈。
