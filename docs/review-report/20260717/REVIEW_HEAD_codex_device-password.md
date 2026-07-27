# Review: LMT 密码任务流程

## 结论

PASS

## 范围

- 后端 ACS/RPC：新增并识别 `X_BAICELLS_COM_PasswordReset` 请求与响应。
- 前端密码管理页：按站型切换修改/重置能力，NR 修改带用户名，修改任务直接创建 `SetParameterValues`。
- 设备详情页：新增密码任务状态标签，终态后清理本地任务缓存。
- 前端共享层：新增创建设备任务 API、补充 i18n 和 mock 参数。

## 检查项

- Go 后端：RPC 方法注册、SOAP 模板、响应匹配、任务请求兼容性、单元测试覆盖。
- React/TypeScript：表单校验、任务提交路径、Hook 使用、状态持久化、错误提示。
- 通用：无 SQL 拼接、无运营商硬编码、无敏感密码持久化、无阻塞级安全问题。

## 发现

未发现 CRITICAL / WARNING 问题。

## 验证

- `cd omcgo && go build ./...`：通过
- `cd omcgo && go test ./internal/acs/... ./pkg/soap/... ./internal/task/...`：通过
- `cd omcgo && go test ./...`：通过
- `cd omcmb && npm run typecheck`：通过
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web`：通过
- `curl -I http://localhost:8081/`：返回 200

## 风险

- `X_BAICELLS_COM_PasswordReset` 是厂商私有 RPC，依赖设备侧支持该方法；不支持的设备会按任务失败路径展示。
- 密码修改绕过通用参数模型校验，按需求交由设备侧校验；这只用于密码管理入口的直接任务创建。
