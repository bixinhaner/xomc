# Terminal Playwright

终端 Codex 需要控制可见 Chrome、macOS GUI、CDP、标签页复用、真实 UI 交互或保留浏览器现场时，使用本 reference。

## 默认策略

- 给 Codex 自身控制浏览器时，使用 workspace 级独立工具目录 `$WORKSPACE_ROOT/.codex-tools/playwright`，不要改业务项目依赖。
- `$WORKSPACE_ROOT` 指当前团队 workspace 根目录，通常是包含 `xomc/` 和 `.codex-tools/` 的目录；如果实际目录不同，按当前机器推导。
- 只有验证项目自身 Playwright 测试或复现项目依赖问题时，才使用项目内 Playwright。
- macOS 终端中，只要任务需要打开可见系统 Chrome、连接 GUI、访问 browser profile、访问 `127.0.0.1:9222`、读取或导航标签页、检查或清理托管 Chrome 进程、保留最终现场，默认直接使用 `require_escalated` 执行固定脚本或窄命令。
- 只有任务不需要 GUI/profile 权限，或只是做无头/轻量 Playwright 检查时，才优先使用 `mcp__node_repl__js` 直接运行 Playwright 代码。
- 如果仍遇到权限或沙箱阻止，继续按审批机制请求更高权限，不要静默退回低保真验证。

## 已知 macOS 拦截模式

- MCP Node REPL 里直接 `chromium.launch({ executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' })` 可能因 Chrome Crashpad 访问 `~/Library/Application Support/Google/Chrome/Crashpad/settings.dat` 被拒而失败。
- MCP Node REPL 里直接 `chromium.connectOverCDP('http://127.0.0.1:9222')` 可能失败，典型错误是 `connect EPERM 127.0.0.1:9222`。
- 普通沙箱下访问本机 CDP 端口可能失败；同一检查提权后可用。
- 普通沙箱下进程检查可能失败；需要确认或清理残留 Chrome 时提权执行 `ps`/`kill`。
- 提权 `chromium.connectOverCDP` 如果报 `Protocol error (Browser.setDownloadBehavior): Browser context management is not supported`，通常表示 9222 上的 Chrome 残留状态不适合 Playwright context 接管；先用 DevTools HTTP 接口 `/json/list` 判断页面，必要时清理残留 Chrome 后从干净状态重启。
- 这些通常不是 Playwright 版本问题。已知需要可见 Chrome、GUI 或 profile 的任务中，直接使用两段式提权方案。

## 固定脚本

固定脚本目录：`$WORKSPACE_ROOT/.codex-tools/browser-control`

这些脚本随 `omc-codex-skills` plugin 发布在 `<omc-codex-skills-plugin-root>/tools/browser-control`。如果 workspace 目录缺少任一固定脚本，先安装：

```bash
<omc-codex-skills-plugin-root>/tools/browser-control/install-browser-control-tools.sh "$WORKSPACE_ROOT"
```

安装脚本只复制 `check-cdp.sh`、`start-chrome.sh`、`list-tabs.js`、`navigate-existing-tab.js`、`stop-managed-chrome.sh` 并设置可执行权限；不安装 Playwright，不联网，也不修改业务项目依赖。`start-chrome.sh` 和 `stop-managed-chrome.sh` 会从安装后的 `$WORKSPACE_ROOT/.codex-tools/browser-control` 自动推导默认 Chrome profile：`$WORKSPACE_ROOT/.codex-tools/chrome-profile`。

优先流程：

1. 检查 Chrome CDP 是否已监听：

```bash
$WORKSPACE_ROOT/.codex-tools/browser-control/check-cdp.sh
```

2. 只有没有监听进程时，才启动一次可见 Chrome：

```bash
$WORKSPACE_ROOT/.codex-tools/browser-control/start-chrome.sh http://127.0.0.1:8081
```

`start-chrome.sh` 内部会再次检查 9222；如果已经有监听，它会直接退出，不会新开窗口或标签页。

3. 读取业务标签页数量：

```bash
$WORKSPACE_ROOT/.codex-tools/browser-control/list-tabs.js
```

4. 需要跳转时复用现有业务标签页；如果 CDP 已监听但 `list-tabs.js` 返回 `pageCount: 0`，不要停止 Chrome，直接创建或打开一个新 page target 到目标 URL：

```bash
$WORKSPACE_ROOT/.codex-tools/browser-control/navigate-existing-tab.js http://127.0.0.1:8081/system/config
```

`navigate-existing-tab.js` 使用 DevTools page websocket 操作已有 page target，避免 Playwright `connectOverCDP` 的 context 管理问题，也避免新增标签页。

5. 只有明确需要清理残留调试 Chrome 时，按次审批运行：

```bash
$WORKSPACE_ROOT/.codex-tools/browser-control/stop-managed-chrome.sh
```

该脚本只会停止匹配 `--remote-debugging-port=9222` 且使用 `$WORKSPACE_ROOT/.codex-tools/chrome-profile` 的托管 Chrome；不要给 stop 脚本持久审批。

## 审批边界

适合持久审批的窄前缀：

- `$WORKSPACE_ROOT/.codex-tools/browser-control/start-chrome.sh`
- `$WORKSPACE_ROOT/.codex-tools/browser-control/list-tabs.js`
- `$WORKSPACE_ROOT/.codex-tools/browser-control/navigate-existing-tab.js`

实际调用 `prefix_rule` 时使用当前机器上展开后的绝对路径，不要把 `$WORKSPACE_ROOT` 变量字面量传给审批规则。

不建议持久审批：

- 任意 `node`
- 任意 `open`
- `stop-managed-chrome.sh` 或 `kill`

如果已有监听但 `/json/list` 不可访问、固定脚本失败或 Chrome 残留状态不干净，提权检查该 Chrome 进程，必要时按次审批清理残留后再从干净状态启动一次。`pageCount: 0` 不是清理条件，优先创建新标签页。

## 页面复用纪律

- 走固定脚本流程时，以 `list-tabs.js` 的输出和 `navigate-existing-tab.js` 的选择逻辑为准复用页面。
- 同一次任务只保留一个目标页；已有目标页时，用 `navigate-existing-tab.js` 跳转，不要再次用 `open -na ... URL` 打开地址。
- 如果 `list-tabs.js` 显示 `pageCount: 0`，使用 `navigate-existing-tab.js <url>` 创建新标签页并导航；不要因此停止托管 Chrome。
- 固定脚本流程不需要、也不应该再通过 Playwright `context.pages()` 查找页面；避免回到 `connectOverCDP` 的 context 接管路径。
- 保留最终现场时，不要运行 `stop-managed-chrome.sh`，让 Chrome 继续留在桌面。

## 页面动作脚本

- `check-cdp.sh`、`list-tabs.js`、`navigate-existing-tab.js` 只负责 CDP 状态、页面发现和导航；真实 UI 交互、表单保存或刷新后验证使用 workspace 级 Playwright 动作脚本。
- 如果 `$WORKSPACE_ROOT/.codex-tools/browser-control` 已提供通用动作脚本，优先复用；如果没有，可以在该目录创建一次性窄用途脚本，任务完成后删除。
- 脚本应只覆盖当前任务需要的动作，不要做成任意代码执行入口。
- 页面动作脚本必须使用 `$WORKSPACE_ROOT/.codex-tools/playwright/node_modules/playwright`，不改业务项目依赖。
- 页面动作脚本连接已启动的托管 Chrome：`chromium.connectOverCDP('http://127.0.0.1:9222')`。
- 从现有 targets 中选择目标业务页，例如 URL 包含 `127.0.0.1:8081` 的页面；找不到时先用 `navigate-existing-tab.js <url>` 创建或导航，不要脚本里重复打开 Chrome。
- 执行 `bringToFront()`，等待 `domcontentloaded` 或目标控件可见，再点击、输入、选择、保存。
- 输出 JSON 证据，包括当前 URL、关键可见文本、表单真实值、toast/接口结果、关键请求参数、HTTP 状态、console/page errors 或刷新后验证值。
- 脚本结束只断开自动化连接，不主动关闭 Chrome；验证结束默认保留最终现场。
- 临时 CDP/Playwright 自动化脚本必须保证自身退出，例如在 `finally` 中调用 `process.exit(0)`，避免脚本输出完成后仍挂住连接。

配置保存类操作的推荐验证顺序：点击保存后优先等待成功 toast 或保存接口成功响应；如果反馈不稳定，刷新页面，重新进入目标页签或子视图，读取表单真实值验证持久化，并在最终答复中报告字段和值。

## 仅当明确使用 Playwright CDP

- 先从 `context.pages()` 找已有目标页面。
- 已有目标页时，用 `page.bringToFront()` 和 `page.goto()` 导航。
- 只有 `context.pages()` 为空时才 `context.newPage()`。
- 保留最终现场时，不要调用 `browser.close()`；如果 Node 进程因 CDP 连接不退出，用 `process.exit(0)` 结束脚本连接。

## OMC 项目依赖

Codex 浏览器控制默认使用 `$WORKSPACE_ROOT/.codex-tools/playwright/node_modules/playwright`。

仅在需要跟随 OMC 项目 lockfile 时，从 `$WORKSPACE_ROOT/xomc` 或当前 OMC 仓库根目录运行脚本，并使用：

```js
const { chromium } = require('./omcmb/node_modules/playwright');
```

如果任务是验证 OMC 项目自身测试，才使用 `./omcmb/node_modules/playwright` 或 `./omcmb/node_modules/@playwright/test`。
