# Agency Console（Web）

React + TypeScript + Vite + Tailwind CSS v4 + **pnpm**。轻量控制台壳子，后续对接 agencycli 工作区数据。

**信息架构**：**团队**（编制 / hire）与 **项目**（任务、消息、项目内成员）分离；进入 `/projects/:id` 后侧栏展开 **任务 / 消息 / 成员**。主题：**中性灰 + sky 强调色**（浅色 `neutral`，深色 `zinc`），侧栏加宽与间距已优化。

## 开发

```bash
pnpm install
pnpm dev
# 默认 http://localhost:27891 （避免与 3000/5173/8080 等常见端口冲突）
```

## 国际化（i18n）

- 使用 **i18next** + **react-i18next** + **i18next-browser-languagedetector**。
- 文案放在 `src/locales/<语言>/common.json`（当前：`en`、`zh-CN`）。
- 语言会写入 `localStorage`（`i18nextLng`），并同步 `<html lang>`。
- 新增文案：在两个 JSON 中加同名 key，在组件里 `useTranslation()` → `t('…')`。

## 主题（浅色 / 深色 / 跟随系统）

- `ThemeProvider`（`src/theme/ThemeProvider.tsx`）管理 `light` | `dark` | `system`。
- 持久化键：`localStorage['agency-console-theme']`。
- Tailwind 使用 **`dark:`** 变体；根节点 **`class="dark"`** 表示深色。
- `index.html` 内联脚本在首屏前应用主题，减轻闪烁（FOUC）。
- **顶部栏** 太阳 / 月亮 / 显示器图标：**循环** 浅色 → 深色 → 跟随系统。
- **设置页** 可精确选择语言与外观。

## 构建

```bash
pnpm run build          # 输出到 dist/
pnpm preview            # 本地预览

# 或使用根目录 Makefile（自动 npm install + vite build）
cd .. && make web
```

## 嵌入到 Go 二进制

`web/embed.go` 使用 `//go:embed all:dist` 将 `dist/` 目录编译进 Go 二进制。
`agencycli start` 命令在同一端口同时服务 API 和 SPA 前端。

```bash
cd .. && make build     # 构建前端 + Go 二进制（dist/agencycli）
./dist/agencycli start  # http://127.0.0.1:27892
```
