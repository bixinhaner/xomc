---
name: browser-control
description: 为 Codex 任务判断并使用正确的浏览器控制路径。当用户要求 Codex 打开、跳转、检查、点击、输入、截图、冒烟测试或验证浏览器页面时使用；当需要在桌面 app 的 in-app Browser、终端 Codex 的 MCP + Playwright、可见系统 Chrome、本地/应用浏览器、web search 之间选择时使用；当需要诊断浏览器控制不可用原因，或需要为用户保留可见浏览器现场时使用。
---

# Browser Control

使用本 skill 作为本地唯一的浏览器控制入口。先判断运行环境和用户可见性要求，再选择正确的控制 surface。

## 决策顺序

1. 判断运行环境。Codex 桌面 app 才优先尝试 in-app Browser；终端 Codex 直接走 MCP + 可见系统 Chrome + Playwright。
2. 桌面 app 中，浏览器控制和前端验证优先使用 Codex 内置 Browser surface：Browser plugin skill `browser:control-in-app-browser`，按其 `browser-client` bootstrap 执行。
3. 桌面 app 中也必须确认 in-app Browser surface 实际可用；如果 `agent.browsers.list()` 返回空数组或 `iab` 不可选，记录 blocker，再切换到 MCP + 可见系统 Chrome + Playwright。
4. 终端 Codex 中，浏览器控制和前端验证优先使用 MCP + 可见系统 Chrome + Playwright，`headless: false`，验证结束后默认保留现场。开始前先读 [terminal-playwright.md](references/terminal-playwright.md)。
5. 用户明确指定其他浏览器 surface，或首选路径不可用时，按任务需要切换并说明原因。
6. 简单互联网查询使用 web search/browser 工具，不启动 UI 自动化。
7. 源码检查或 HTTP/API 检查足够完成任务，且 UI 行为不是重点时，不启动浏览器自动化。

## 浏览器事实优先

- 浏览器操作或验收时，不管源码显示什么，都必须以浏览器中的实际页面为准。
- 执行点击、输入、选择、保存前，先读取运行中 DOM，确认目标控件、按钮和保存行为存在且可操作。
- 源码、路由配置、i18n 文案和测试只能作为定位线索；不能替代当前浏览器页面确认。
- Web/UI 验证不得用 `curl` 代替浏览器检查。

## 桌面 App 内置 Browser

当 Codex 运行在桌面 app 中，且 Browser plugin 可用时，按以下规则执行：

- 如果 JS 执行工具还没有暴露，使用 tool discovery 搜索 `node_repl js`，不要设置 result limit；不要用 `js_reset` 或 `js_add_node_module_dir` 试图暴露 `js`。
- 通过当前会话可用的 `browser:control-in-app-browser` skill source path 推导插件根目录，并绝对路径导入其 `scripts/browser-client.mjs`；不要写死某台机器的插件缓存路径。
- 每个新的 Node session 只初始化一次 runtime，选择 `iab`，并在交互前完整读取 browser documentation。
- 使用返回的 `browser` 对象及其文档化 API 执行跳转、点击、输入、截图和 Playwright 操作。
- 如果 setup 成功但 discovery 或 selection 失败，先读取 browser bootstrap troubleshooting；如果需要检查可用 browser，执行一次 `await agent.browsers.list()`。
- 如果登录阻塞流程，让用户在可见/in-app 浏览器中登录，然后继续。

```js
const { setupBrowserRuntime } = await import('<browser-plugin-root>/scripts/browser-client.mjs');
await setupBrowserRuntime({ globals: globalThis });
globalThis.browser = await agent.browsers.get('iab');
nodeRepl.write(await browser.documentation());
```

## 终端 Codex + 可见 Chrome

终端、macOS、GUI/CDP、可见 Chrome、固定脚本、页面动作脚本、页面复用或最终现场保留相关任务，先读 [terminal-playwright.md](references/terminal-playwright.md)，再操作。

核心规则：

- 给 Codex 自身控制浏览器时，使用 workspace 级独立工具目录 `$WORKSPACE_ROOT/.codex-tools/playwright`，不要改业务项目依赖。
- macOS 终端中，只要需要可见 Chrome、GUI、browser profile、CDP 端口、标签页导航、进程检查或保留最终现场，默认用 `require_escalated` 执行固定脚本或窄用途动作脚本。
- 固定脚本目录为 `$WORKSPACE_ROOT/.codex-tools/browser-control`；缺失时从 `omc-codex-skills` plugin 的 `tools/browser-control/install-browser-control-tools.sh` 安装到 workspace。
- 优先流程是 `check-cdp.sh` -> 必要时 `start-chrome.sh` -> `list-tabs.js` -> `navigate-existing-tab.js`。同一次任务只保留一个目标页。
- 真实 UI 交互、表单保存、刷新后验证、网络检查和 console/page error 收集，用 Playwright 页面动作脚本执行；脚本输出 JSON 证据，只断开自动化连接，不关闭 Chrome。
- 除非用户要求关闭，验证结束后保留最终现场。

## OMC 本地约定

- OMC Docker 部署默认验证 `http://127.0.0.1:8081`，除非用户指定其他地址。
- 操作或验证 OMC 性能管理页面时，先加载并遵循 `omc-codex-skills:omc-pm-metrics` skill 的“页面结构”规则；进入页面后先扫描 URL、导航、页签、表单控件、按钮、图表/表格状态和关键请求参数，把运行时扫描结果写入本次任务证据，再执行点击、筛选、出图或导出。
- 需要登录时走真实 UI 登录流程。除非任务明确是 API-only，不绕过 UI。
- 除非明确需要并获批，不要运行 `npx playwright install`。
- 如果操作可能改变线上/本地运行栈的全局设置，或触发真实设备侧工作流，必须先停下并请求用户明确授权。

## 失败与降级

- 如果 in-app Browser plugin 缺少 `scripts/browser-client.mjs`，报告这个具体缺失文件，不要猜其他 API。
- 如果 in-app Browser 不可用，不要卡住流程；在任务允许时改用 MCP + 可见 Playwright，并说明原因。
- 如果可见 Playwright 因依赖或浏览器二进制不可用而无法启动，先检查 `$WORKSPACE_ROOT/.codex-tools/playwright`；只有安装或联网确实必要时才请求升级权限。
- 如果失败原因是权限、沙箱、GUI、CDP 或进程访问受限，并且任务需要真实浏览器证据，使用 `require_escalated` 请求更高权限执行相关命令。
- 如果任务只靠源码检查或 HTTP/API 检查即可完成，且 UI 行为不是重点，不启动浏览器自动化。

## in-app Browser 常见恢复

- 如果后续调用出现 `browser is not defined`，优先视为当前 JavaScript 控制会话里的 browser 绑定丢失；按桌面 App 内置 Browser 的 bootstrap 重新连接当前 in-app Browser。
- 如果 `tab.playwright.domSnapshot()` 报 `incrementalAriaSnapshot is not a function`，降级到 `tab.dom_cua.get_visible_dom()`，或使用一次只读 `tab.playwright.evaluate(...)` 提取当前页面结构。
- 对 localhost 配置页做保存类操作时，先通过当前可见 tab 和真实 UI 完成修改；保存后如需验证持久化，可以刷新页面，再重新进入对应页签或子视图检查表单值。
- 保存类配置操作如果没有看到成功 toast 或明确网络成功响应，不要只凭点击保存判断完成；刷新页面并读取表单值验证持久化。
- 恢复路径不应写入环境账号密码、一次性的 DOM `node_id` 或具体业务配置项名称。

## 验证清单

浏览器工作结束前确认：

- 所选控制路径符合运行环境和用户可见性要求。
- 已用截图、可见 DOM 或 UI 断言覆盖用户要求验证的行为。
- 已记录关键请求参数、HTTP 状态、关键响应结果，以及 console/page errors。
- 已记录最终 URL、关键可见文本、表单真实值、toast/接口结果或刷新后验证值。
- 在用户要求或本地约定需要时，保留了浏览器现场。
- 如果首选路径不可用，已明确报告具体 blocker。
