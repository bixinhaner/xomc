# Issue #153 Agent 手册与日志任务排障能力审查

## 结论

PASS，无阻塞发现，可以进入 GitLab CI 和合入流程。

## 审查范围

- GitLab Issue：#153「AI 分析日志收集任务失败原因时提示没有读取入口」
- UFTE 任务与设备执行明细的语义、筛选参数和失败字段说明
- Agent 手册的 Go Handler / OpenAPI 契约提取、运行时路由合并与确定性打包
- Docker 构建、Makefile、GitHub Actions 和 GitLab CI 的手册一致性门禁

## 关键判断

1. 根因不是 OMC 缺少查询 API，而是旧手册把 `/api/v1/ufte/tasks` 和 `/api/v1/ufte/devices` 描述成通用列表，模型无法可靠发现日志收集任务、设备执行明细及 `failureReason` / `failureDetail`。
2. 修复补齐 `station_log`、`RUNTIME_LOG_COLLECT`、失败状态和失败原因等业务语义，同时从实际 Handler 与 OpenAPI 自动生成全量契约，避免只修单个接口后再次漂移。
3. 运行时仍以实际 Gin 路由为发布边界；契约资产只补充说明和参数，不会发布不存在的 API，也没有改变 Web UI 原有调用流程。
4. Agent 执行策略保持只读风险标识和现有用户权限校验，本次没有增加绕过鉴权、数据库直连或写操作入口。

## 验证证据

- `make agent-handbook-check`：通过，契约和嵌入包一致。
- `go test -count=1 ./internal/agentruntime/... ./internal/ufte/...`：通过。
- `go test -count=1 ./cmd/agent-handbook/...`：通过。
- `go vet ./...`：通过。
- `go build ./...`：通过。
- `git diff --check`：通过。
- 本地 8081 → 生产 Agent Studio 真实浏览器验证：首次下载并校验 708 个接口；日志收集/文件传输失败记录查询能找到 UFTE 设备明细能力并返回真实结果；后续手册命中缓存。
- Agent 业务调用日志：相关请求均为只读 `GET 200`，没有业务写请求。

## 非阻塞环境项

- 本机未安装 `golangci-lint`，由 GitLab CI 容器执行最终静态检查。
- 本机完整 `go test ./...` 有 7 个 `internal/task` PostgreSQL 集成用例因现有数据库缺少 `parameter_sync_runs` / `parameter_sync_requests` 等迁移表失败；失败与本次改动无文件交集，相关模块、全量构建和静态检查均通过。

## 风险与回滚

- 主要风险是生成资产体积增加及源码解析规则误提取。确定性测试、陈旧资产检查和运行时实际路由过滤已覆盖该风险。
- 回滚本提交即可恢复旧手册生成方式，不涉及数据库迁移和业务数据回滚。
