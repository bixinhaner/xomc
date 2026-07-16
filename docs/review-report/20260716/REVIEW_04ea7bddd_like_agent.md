# Code Review: Agent 文件与会话控制

## 结论

**PASS_WITH_WARNINGS**。未发现 CRITICAL 问题，可以进入 Merge Request 流程。

## 范围

- Issue: #91
- Commit: `04ea7bddd`
- 后端：Agent Studio 服务凭证读取、会话历史、附件上传/删除、运行中止、产物读取代理
- 前端：附件交互、生成文件预览/下载、历史恢复、停止响应及中英文提示

## 审查结果

### 安全与边界

- 新接口仍注册在 OMC `/api/v1` 已认证路由组内，并使用当前 JWT 用户生成外部用户标识。
- Agent Studio 服务 Token 只在后端运行时配置中读取，不进入管理配置响应、前端状态或日志。
- Connector ID、conversation ID、run ID、attachment ID 和 artifact ID 均使用 URL path escaping。
- 附件在 OMC 前后端同时限制：空文件拒绝、单文件最大 25 MB、单轮最多 10 个。
- 产物读取由 OMC 携带当前 OMC 用户身份转发，Agent Studio 再校验 Connector、会话和外部用户归属。
- 长连接和普通代理请求均继承请求 context；用户停止时会调用 Agent Studio cancel 接口。

### 行为与用户体验

- 单附件、中文文件名、多附件联合分析均通过真实浏览器验证。
- 生成文件可预览、下载，下载内容与服务端产物逐字节一致。
- 关闭重开 Agent 和刷新页面后，会话、附件及产物均可恢复。
- 停止响应后显示明确状态，并可继续同一会话。
- 空文件和超大文件在 OMC 前端直接给出可行动的中文提示，不产生无效上传请求。

### 线程与权限隔离

- 多轮、刷新、附件和中止恢复后，Agent Studio 数据库仍为 1 个会话绑定、1 个 Codex thread。
- 无 Service Token 请求返回 401。
- 使用错误外部用户读取历史或产物时被拒绝。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./...`：全量通过，含 integration/e2e。
- `cd omcmb && npm run typecheck`：通过。
- Agent 前端定向测试：2 个测试文件、8 个测试通过。
- 改动前端文件定向 ESLint：0 error，3 warnings。
- Docker OMC `http://localhost:8081` 到生产 Agent Studio：真实浏览器端到端通过。
- Agent Studio `8787/healthz`、`8791/healthz`：正常。

## Warning / N/A

1. 本机未安装 `golangci-lint`，因此该项未执行；`go build` 与全量 `go test` 已通过。
2. 全仓 `npm run lint` 受历史问题阻塞（4 errors、1656 warnings）；本次改动文件定向 ESLint 无 error，3 条 warning 均为现有 controller 的 effect 状态同步模式，未由本次功能引入新的 lint error。
3. `omcgo/scripts/e2e_verify.sh` 未增加依赖生产 Agent Studio 的固定用例。该链路需要有效 Connector、Service Token 与远端 Codex runtime，本轮使用本地 Docker OMC + 生产 Agent Studio 完成了覆盖更完整的真实浏览器 E2E，并验证了服务端数据隔离。
4. 附件 MIME 类型允许通用文件，由 Agent Studio 的隔离工作区负责消费；OMC 负责身份、大小和数量边界，不在业务侧维护易失真的文件类型白名单。

## DoD 判断

- 编译、全量 Go 测试、前端类型检查、真实 E2E、安全隔离：通过。
- 数据迁移、运营商适配、SQL、菜单 seed：N/A，本次未涉及。
- 对外 Swagger：N/A，新增端点属于 OMC 内嵌 Agent 适配层，不作为第三方开放 API。
- 最终结论：无阻塞合入的问题；上述工具缺失和仓库历史 lint 基线需如实保留在 MR 说明中。
