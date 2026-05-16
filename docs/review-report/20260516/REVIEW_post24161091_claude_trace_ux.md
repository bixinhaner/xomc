# 代码审查报告 — T-0137 报文跟踪 UX follow-up（设备 SN 选择 / 报文详情 / 下载行为）

| 项 | 值 |
|---|---|
| 审查时间 | 2026-05-16 |
| 审查对象 | 4 文件 / +149 / -27 |
| 基线 commit | 24161091（段 2 V-1 hygiene） |
| 关联 Backlog | T-0137（活体试用后用户反馈的 UX 修复） |
| 审查者 | Claude（自审，附 playwright 端到端验证 + curl 服务器响应头验证） |
| **审查结论** | **PASS** |

---

## 1. 变更范围

### 1.1 设备 SN 选择（用户反馈：不应让用户手输）

`omcmb/webcode/src/pages/ops/MessageTrace/index.tsx` — `<Input>` 改成 `<DeviceSnSelect>` 组件：

- 基于 `useDeviceList`（ahooks `useDebounce` 300ms）
- 不输入不预加载（10 万级规模友好）
- 输入 ≥ 1 字触发后端 `searchText` 模糊匹配（覆盖 SN/名称/IP/MAC）
- 最多回 20 条；未输入时下拉显示"输入 SN / 站点名搜索设备"提示
- 同时支持粘贴完整 SN（运维场景常见）

`omcmb/frontend-core/src/hooks/api/useDevices.ts` — `UseDeviceListOptions` 加 `enabled?: boolean` 字段（新增可选，向后兼容），用于 AutoComplete 异步搜索场景的按需 fetch 控制。

### 1.2 报文详情：XML 美化 + 暗色主题文字可见（用户反馈：白字白底看不到）

`MessageDetail` 组件：

- 旧代码用了不存在的 CSS 变量 `var(--color-surface-2)`（项目用 Ant Design Token 不是手写 CSS var），fallback 到 `#f5f5f5` 浅灰 — 暗色主题下文字仍是白色，导致白字白底
- 改用 `theme.useToken()` 拿 `colorFillTertiary` / `colorText` / `colorBorderSecondary` — 自动适配明暗主题
- 加 monospace 字体（Menlo / Consolas / Courier New）
- 新增 `prettyXML()` 工具：按 tag 边界拆行 + 计算缩进 + CDATA 占位符保护（防止 CDATA 内部被误格式化）

### 1.3 下载文件名 + 下载行为（用户反馈：文件名无意义 + 下载文件夹找不到）

**后端 `omcgo/internal/trace/handler.go`**：
- `exportFilename` 格式从 `{sn}_{jobIdShort}.xml` 改成 `trace_{sn}_{YYYYMMDD-HHmm}.xml`，含 SN + 抓包开始时间，运维归档可按时间 ls 排序
- 例：`trace_1202000240194DP0026_20260516-1442.xml`

**后端 `omcgo/internal/trace/exporter.go`**：
- 对象上传 Content-Type 从 `application/xml` 改成 `application/octet-stream`
- **关键根因**：Chrome 对可 inline 渲染的 MIME（XML/HTML/PDF/Image）会忽略 `<a download>` 属性，把 URL 当 navigation 在新 tab 打开，导致用户找不到下载文件
- 在上传源对象时就指定 octet-stream，避免预签名 URL 上覆盖 `response-content-type` query 参数（实测会破坏 AWS SigV4 签名 → MinIO 返回 403 SignatureDoesNotMatch）

**前端 `triggerDownload(url)`**：
- 旧：`async function triggerDownload(url)` 内部 `await fetch + await arrayBuffer + new Blob + a.click()` — Chrome user-gesture 模型要求 `a.click()` 在 user-click 同步调用栈内，async 跳出 task 会丢失 user gesture 上下文
- 新：同步函数，直接 `<a href={url}>.click()` — 在 user click 同步上下文里。服务器侧 `Content-Disposition: attachment` + `Content-Type: application/octet-stream` 头组合让 Chrome 强制下载

## 2. 端到端验证证据

### 2.1 服务器响应头（curl 直接拿 MinIO 响应）

```
HTTP/1.1 200 OK
content-disposition: attachment; filename="trace_1202000240194DP0026_20260516-1549.xml"
content-length: 88671
content-type: application/octet-stream
```

### 2.2 浏览器下载触发（playwright）

playwright 模拟点击导出按钮 → Chrome 触发 download 事件 → 文件保存到磁盘：
```
.playwright-mcp/trace-1202000240194DP0026-20260516-1549.xml — 88671 bytes
头部: <?xml version="1.0" encoding="UTF-8"?><TraceExport task_id="..." device_sn="1202000240194DP0026">
```

### 2.3 编译产物验证

`omcmb/webcode/dist/assets/index-B7IUgWg7.js` 含 minified triggerDownload：
```js
function je(e){const t=document.createElement("a");t.href=e,t.rel="noopener noreferrer",
document.body.appendChild(t),t.click(),document.body.removeChild(t)}
```
完全同步，无 fetch/blob 痕迹 — 新代码生效。

### 2.4 自动化检查

- `cd omcgo && go build ./...` ✅
- `cd omcgo && go test ./internal/trace/...` ✅ (0.78s)
- `cd omcmb/webcode && npx tsc --noEmit` ✅

### 2.5 设备 SN 选择（playwright UI 测试）

- 未输入时下拉显示"输入 SN / 站点名搜索设备"，0 options，不预加载 ✅
- 输入 "1202" 后 debounced 300ms 触发，返回 5 条匹配（真实基站首位 + 4 个含 1202 的 mock）✅

### 2.6 报文详情样式（playwright getComputedStyle）

```
bg: rgba(255, 255, 255, 0.08)   <- 暗色主题浅灰底（非纯白）
fg: rgb(245, 245, 247)          <- 接近白但非纯白（前景色）
border: 1px solid rgb(51, 51, 54)
font: Menlo, Consolas, monospace
line_count: 174                  <- XML 已按层级缩进美化（之前 1 行长串）
```

## 3. 检查项

| 检查项 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `go test ./...` | ✅ |
| `go vet ./...` | ✅ |
| 前端 typecheck | ✅ |
| 接口向后兼容 | ✅（`UseDeviceListOptions.enabled` 新增可选）|
| SigV4 签名 | ✅（撤销破坏签名的 `response-content-type` query 参数）|
| 暗色主题 | ✅（用 Ant Design Token 替代手写 CSS var）|

## 4. 发现

### 4.1 CRITICAL / WARNING

无。

### 4.2 INFO

- 排查 download 失败时走了 2 个错误方向：① 前端 blob+octet-stream ② 预签名 URL 覆盖 response-content-type。最终找到真根因是 ① async user-gesture 丢失 + ② 源对象 Content-Type 是 application/xml。两者叠加导致 Chrome 决定 navigation 而非 download，下载历史显示 URL 末段 UUID。
- exporter Content-Type 一旦定，旧对象不会自动更新（MinIO 已存对象保留原 metadata）。3 天 retention drop 后自动清。

## 5. 结论

**PASS** — 可合入。
