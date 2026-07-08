---
name: browser-control
description: 为 Codex 任务判断并使用正确的浏览器控制路径。当用户要求 Codex 打开、跳转、检查、点击、输入、截图、冒烟测试或验证浏览器页面时使用；当需要在桌面 app 的 in-app Browser、终端 Codex 的 MCP + Playwright、可见系统 Chrome、本地/应用浏览器、web search 之间选择时使用；当需要诊断浏览器控制不可用原因，或需要为用户保留可见浏览器现场时使用。
---

# Browser Control

使用本 skill 作为本地唯一的浏览器控制入口。先判断运行环境和用户可见性要求，再选择正确的控制 surface。

## 决策顺序

1. 先判断当前运行环境：Codex 桌面 app 才优先尝试 in-app Browser；终端 Codex 不尝试 in-app Browser，直接走 MCP + 可见系统 Chrome + Playwright。如果当前会话暴露了 `environment_context`、`cwd`、`shell`、sandbox/approval 说明，且没有明确的桌面 app Browser surface，不要再读取或 bootstrap `browser:control-in-app-browser`。
2. 如果在 Codex 桌面 app 中，需要浏览器控制或前端验证时，优先使用 Codex 内置 Browser surface：Browser plugin skill `browser:control-in-app-browser`，严格按其 `browser-client` bootstrap 执行。
3. 桌面 app 中也必须确认 in-app Browser surface 实际可用：如果 `agent.browsers.list()` 返回空数组或 `iab` 不可选，明确记录该 blocker，再切换到 MCP + 可见系统 Chrome + Playwright。
4. 如果在终端中使用 Codex，需要浏览器控制或前端验证时，优先使用 MCP + 可见系统 Chrome + Playwright，`headless: false`，验证结束后默认保留现场。
5. 如果用户明确要求使用另一种浏览器 surface，或首选路径不可用，再按任务需要切换到对应控制方式，并明确说明原因。
6. 如果只是简单互联网查询，使用 web search/browser 工具，不要启动 UI 自动化。
7. 如果源码检查或 HTTP/API 检查足够完成任务，且 UI 行为不是重点，不要启动浏览器自动化。

## 浏览器事实优先

- 浏览器操作或验收时，不管源码显示什么，都必须以浏览器中的实际页面为准。
- 执行点击、输入、选择、保存前，先读取运行中 DOM，确认目标控件、按钮和保存行为存在且可操作。
- 源码、路由配置、i18n 文案和测试只能作为定位线索；不能替代对当前浏览器页面的确认。

## 桌面 App 内置 Browser

当 Codex 运行在桌面 app 中，且 Browser plugin 可用时，浏览器控制和前端验证优先使用 in-app Browser，按以下方式执行：

- 如果 JS 执行工具还没有暴露，使用 tool discovery 搜索 `node_repl js`，不要设置 result limit；不要用 `js_reset` 或 `js_add_node_module_dir` 试图暴露 `js`。
- 通过绝对路径导入插件的 `browser-client.mjs`。不要使用内置或猜测出来的 `browser-client` 包。
- 每个新的 Node session 只初始化一次 runtime，选择 `iab`，并在交互前完整读取 browser documentation。
- 使用返回的 `browser` 对象及其文档化 API 执行跳转、点击、输入、截图和 Playwright 操作。
- 如果 setup 成功但 discovery 或 selection 失败，先读取 browser bootstrap troubleshooting，再考虑重置或切换工具。
- 如果 troubleshooting 要求检查可用 browser，执行一次 `await agent.browsers.list()`。如果返回空数组，表示当前 session 没有暴露任何 browser surface，应明确报告这一点。

优先从当前会话可用的 `browser:control-in-app-browser` skill source path 推导插件根目录，并导入其 `scripts/browser-client.mjs`。不要把某个人机器上的插件缓存路径写死到团队 skill 中。

```js
const { setupBrowserRuntime } = await import('<browser-plugin-root>/scripts/browser-client.mjs');
await setupBrowserRuntime({ globals: globalThis });
globalThis.browser = await agent.browsers.get('iab');
nodeRepl.write(await browser.documentation());
```

规则：

- 只有 Node REPL JS 工具能控制 in-app Browser surface。
- 不要用外部 MCP 浏览器工具或另起 Playwright 来替代 in-app Browser。
- 除非用户询问实现细节，不要向用户提 Node REPL 等内部机制。
- 如果登录阻塞流程，请让用户在可见/in-app 浏览器中登录，然后继续。

## 终端 Codex MCP + Playwright

当 Codex 运行在终端中，或没有桌面 app 可控的内置 Browser surface 时，优先使用 MCP + 可见系统 Chrome + Playwright 控制浏览器：

- 给 Codex 自身控制浏览器时，优先使用 workspace 级独立工具目录 `$WORKSPACE_ROOT/.codex-tools/playwright`，不要改业务项目依赖。`$WORKSPACE_ROOT` 指当前团队 workspace 根目录，通常是包含 `xomc/` 和 `.codex-tools/` 的目录；如果实际目录不同，按当前机器的 workspace root 推导。
- 只有在验证项目自身 Playwright 测试或复现项目依赖问题时，才使用项目内 Playwright。
- 在 macOS 终端 Codex 中，只要任务需要打开可见系统 Chrome、连接 GUI、访问 browser profile、保留最终现场，默认直接使用提权两段式；不要先用 MCP 内 `chromium.launch()` 试错。
- 在 macOS 终端 Codex 中，凡是访问 Chrome CDP 端口 `127.0.0.1:9222`、读取/导航标签页、启动可见 Chrome、检查或清理托管 Chrome 进程，默认直接用 `require_escalated` 执行固定脚本或窄命令，不要先在普通沙箱里试一次再处理 `EPERM`。
- 在 macOS 终端 Codex 中，凡是需要对可见 Chrome 执行真实 UI 操作（点击、输入、选择、保存、刷新验证），不要先用 MCP `node_repl` 里的 `chromium.connectOverCDP('http://127.0.0.1:9222')` 试连；该路径经常被沙箱拦截为 `connect EPERM 127.0.0.1:9222`。直接使用 `require_escalated` 运行 workspace 级浏览器控制脚本，或按本节的“页面动作脚本”模板创建窄用途脚本。
- 只有任务不需要 GUI/profile 权限，或只是做无头/轻量 Playwright 检查时，才优先使用 `mcp__node_repl__js` 直接运行 Playwright 代码。
- 如果仍遇到权限/沙箱阻止，继续按审批机制请求更高权限，不要静默退回低保真验证。
- 需要自动化断言、网络检查、console/page error 收集时，用 Playwright API 执行。
- 如果 MCP/Playwright 缺依赖或需要联网安装，先报告原因并按权限规则请求批准。

已验证的拦截模式与默认策略：

- MCP Node REPL 里直接 `chromium.launch({ executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' })` 可能失败，典型日志是 Chrome Crashpad 访问 `~/Library/Application Support/Google/Chrome/Crashpad/settings.dat` 返回 `Operation not permitted`，随后浏览器进程被关闭或中止。
- MCP Node REPL 里直接 `chromium.connectOverCDP('http://127.0.0.1:9222')` 可能失败，典型错误是 `connect EPERM 127.0.0.1:9222`。
- 普通沙箱下访问本机 CDP 端口也可能失败：`curl http://127.0.0.1:9222/json/list` 返回 code 7；同一命令提权后可用。
- 普通沙箱下进程检查可能失败：`ps` 返回 `operation not permitted`；需要确认或清理残留 Chrome 时提权执行 `ps`/`kill`。
- 如果提权 `chromium.connectOverCDP` 报 `Protocol error (Browser.setDownloadBehavior): Browser context management is not supported`，通常表示当前 9222 上的 Chrome 残留状态不适合 Playwright context 接管；先用 DevTools HTTP 接口 `/json/list` 判断页面，必要时清理该残留 Chrome 后从干净状态重启。
- 这些不是 Playwright 版本问题；在已知需要可见 Chrome/GUI/profile 的任务中，直接使用两段式提权方案，不要为了确认错误而先失败一次。
- 不要改业务项目依赖，也不要退回只看 DOM/source 的低保真验证。
- 重复执行 `open -na "Google Chrome" ... URL` 会导致多标签/多窗口；这不是业务页面问题，是控制流程没有先复用已有 CDP 页面。


Codex 独立工具的 MCP 内直接写法，仅在已知 MCP 进程具备可见 Chrome、GUI 和 profile 权限时使用：

```js
const workspaceRoot = '<absolute workspace root containing .codex-tools>';
const { chromium } = await import(`${workspaceRoot}/.codex-tools/playwright/node_modules/playwright/index.mjs`);

const browser = await chromium.launch({
  headless: false,
  executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
});
const page = await browser.newPage();
await page.goto('http://127.0.0.1:8081', { waitUntil: 'domcontentloaded' });
```

macOS 终端 Codex 的可见浏览器默认使用固定脚本，不要提权任意 `node` 或重复手写 `open` 命令。

固定脚本目录：`$WORKSPACE_ROOT/.codex-tools/browser-control`

这些固定脚本随 `omc-codex-skills` plugin 一起发布在 `<omc-codex-skills-plugin-root>/tools/browser-control`。如果 `$WORKSPACE_ROOT/.codex-tools/browser-control` 不存在，或缺少下面任一固定脚本，先从 plugin 安装到 workspace，再继续浏览器控制流程：

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

4. 需要跳转时复用现有业务标签页；如果 CDP 已监听但 `list-tabs.js` 返回 `pageCount: 0`，不要停止 Chrome，直接创建/打开一个新 page target 到目标 URL：

```bash
$WORKSPACE_ROOT/.codex-tools/browser-control/navigate-existing-tab.js http://127.0.0.1:8081/system/config
```

`navigate-existing-tab.js` 使用 DevTools page websocket 操作已有 page target，避免 Playwright `connectOverCDP` 的 context 管理问题，也避免新增标签页。

如果当前没有任何 page target，`navigate-existing-tab.js` 应通过 DevTools `/json/new?<url>` 新建标签页；这属于恢复空目标状态，不需要运行 `stop-managed-chrome.sh`。

5. 只有明确需要清理残留调试 Chrome 时，按次审批运行：

```bash
$WORKSPACE_ROOT/.codex-tools/browser-control/stop-managed-chrome.sh
```

该脚本只会停止匹配 `--remote-debugging-port=9222` 且使用 `$WORKSPACE_ROOT/.codex-tools/chrome-profile` 的托管 Chrome；不要给 stop 脚本持久审批。

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

页面复用纪律：

- 走固定脚本流程时，以 `list-tabs.js` 的输出和 `navigate-existing-tab.js` 的选择逻辑为准复用页面。
- 同一次任务只保留一个目标页；已有目标页时，用 `navigate-existing-tab.js` 跳转，不要再次用 `open -na ... URL` 打开地址。
- 如果 `list-tabs.js` 显示 `pageCount: 0`，使用 `navigate-existing-tab.js <url>` 创建新标签页并导航；不要因此停止托管 Chrome。
- 固定脚本流程不需要、也不应该再通过 Playwright `context.pages()` 查找页面；避免回到 `connectOverCDP` 的 context 接管路径。
- 保留最终现场时，不要运行 `stop-managed-chrome.sh`，让 Chrome 继续留在桌面。

页面动作脚本：

- `check-cdp.sh`、`list-tabs.js`、`navigate-existing-tab.js` 只负责 CDP 状态、页面发现和导航；当任务需要真实 UI 交互、表单保存或刷新后验证时，使用 workspace 级脚本执行 Playwright 动作，不要回退到 MCP `node_repl` 直连 CDP。
- 如果 `$WORKSPACE_ROOT/.codex-tools/browser-control` 已提供通用动作脚本，优先复用；如果没有，可以在该目录创建一次性窄用途脚本，任务完成后删除。脚本应只覆盖当前任务需要的动作，不要做成任意代码执行入口。
- 页面动作脚本必须统一处理：
  - 使用 `$WORKSPACE_ROOT/.codex-tools/playwright/node_modules/playwright`，不改业务项目依赖。
  - `chromium.connectOverCDP('http://127.0.0.1:9222')` 连接已启动的托管 Chrome。
  - 从现有 targets 中选择目标业务页，例如 URL 包含 `127.0.0.1:8081` 的页面；找不到时先用 `navigate-existing-tab.js <url>` 创建/导航，不要脚本里重复打开 Chrome。
  - `bringToFront()`，等待 `domcontentloaded` 或目标控件可见，再执行点击、输入、选择、保存。
  - 输出 JSON 证据，包括当前 URL、关键可见文本、表单真实值、toast/接口结果或刷新后验证值。
  - 脚本结束只断开自动化连接，不主动关闭 Chrome；验证结束默认保留最终现场。
- 配置保存类操作的推荐验证顺序：点击保存后优先等待成功 toast 或保存接口成功响应；如果反馈不稳定，刷新页面，重新进入目标页签或子视图，读取表单真实值验证持久化，并在最终答复中报告字段和值。

仅当明确使用 Playwright CDP 时：

- 先从 `context.pages()` 找已有目标页面。
- 已有目标页时，用 `page.bringToFront()` 和 `page.goto()` 导航。
- 只有 `context.pages()` 为空时才 `context.newPage()`。
- 保留最终现场时，不要调用 `browser.close()`；如果 Node 进程因 CDP 连接不退出，用 `process.exit(0)` 结束脚本连接。
- 临时 CDP/Playwright 自动化脚本必须保证自身退出，例如在 `finally` 中调用 `process.exit(0)`，避免脚本输出完成后仍挂住连接；退出脚本不等于关闭 Chrome。

OMC 项目依赖写法，仅在需要跟随项目 lockfile 时使用：

```js
const { chromium } = require('./omcmb/node_modules/playwright');
```

本地约定：

- Codex 浏览器控制默认使用 `$WORKSPACE_ROOT/.codex-tools/playwright/node_modules/playwright`。
- 使用 OMC hoisted 依赖时，从 `$WORKSPACE_ROOT/xomc` 或当前 OMC 仓库根目录运行脚本。
- 如果任务是验证 OMC 项目自身测试，才使用 `./omcmb/node_modules/playwright` 或 `./omcmb/node_modules/@playwright/test`。
- OMC Docker 部署默认验证 `http://127.0.0.1:8081`，除非用户指定其他地址。
- 除非明确需要并获批，不要运行 `npx playwright install`。
- 除非用户要求关闭，验证结束后不要主动关闭 Chrome，保留最终现场。
- 需要登录时走真实 UI 登录流程。除非任务明确是 API-only，不绕过 UI。
- 如果操作可能改变线上/本地运行栈的全局设置，或触发真实设备侧工作流，必须先停下并请求用户明确授权。

## 失败与降级

- 如果 in-app Browser plugin 缺少 `scripts/browser-client.mjs`，报告这个具体缺失文件，不要猜其他 API。
- 如果 in-app Browser 不可用，不要卡住流程；在任务允许时改用 MCP + 可见 Playwright，并说明原因。
- 如果可见 Playwright 因依赖或浏览器二进制不可用而无法启动，先检查 `$WORKSPACE_ROOT/.codex-tools/playwright`；只有安装或联网确实必要时才请求升级权限。
- 如果失败原因是权限/沙箱/GUI/CDP/进程访问受限，并且任务需要真实浏览器证据，使用 `require_escalated` 请求更高权限执行相关命令。
- 如果任务只靠源码检查或 HTTP/API 检查即可完成，且 UI 行为不是重点，就不要启动浏览器自动化。

## in-app Browser 常见恢复

- 如果后续调用出现 `browser is not defined`，优先视为当前 JavaScript 控制会话里的 browser 绑定丢失；按桌面 App 内置 Browser 的 bootstrap 重新连接当前 in-app Browser，不要把它误判为页面权限、登录权限或系统沙箱权限问题。
- 如果 `tab.playwright.domSnapshot()` 报 `incrementalAriaSnapshot is not a function`，优先降级到 `tab.dom_cua.get_visible_dom()` 读取可见控件，或使用一次只读 `tab.playwright.evaluate(...)` 提取当前页面的表单标签、输入值、按钮文本等结构化信息。
- 对 localhost 配置页做保存类操作时，先通过当前可见 tab 和真实 UI 完成修改；保存后如需验证持久化，可以刷新页面，再重新进入对应页签或子视图检查表单值，因为刷新后应用可能回到默认子页。
- 对保存类配置操作，如果没有看到成功 toast 或明确网络成功响应，不要只凭点击保存判断完成；刷新页面并重新进入目标页签或子视图，读取表单值验证持久化。
- 这些恢复路径属于浏览器控制经验，不应写入环境账号密码、一次性的 DOM `node_id`，或具体业务配置项名称。

## 验证清单

浏览器工作结束前确认：

- 所选控制路径符合运行环境和用户可见性要求。
- 截图或 UI 断言覆盖了用户要求验证的行为。
- 已记录关键证据：URL、可见 UI 断言、网络/API 状态、console/page errors。
- 在用户要求或本地约定需要时，保留了浏览器现场。
- 如果首选路径不可用，已明确报告具体 blocker。
