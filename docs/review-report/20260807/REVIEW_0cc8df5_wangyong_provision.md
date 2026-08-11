# 即插即用策略与参数配置改动审查报告

- 日期：2026-08-07
- 基线：`0cc8df5dc`
- 分支：`feat/plug-and-play-policy-parameters`
- 范围：Provisioning 后端、即插即用前端、参数配置与产品匹配
- 结论：`PASS_WITH_WARNINGS`

## 审查范围

- 后端策略产品名称匹配、任务筛选/统计、公共参数物化、gNB ID/PCI 分配与策略启用冲突检查。
- 前端策略列表、任务筛选、新增/编辑策略、公共参数、批量规划、参数导入预检及中英文文案。
- 相关单元测试、API 映射测试与设计对比文档。
- `deployments/docker/docker-compose.yml` 按用户要求未纳入暂存和本次审查提交。

## 发现

### WARNING

1. `omcgo/internal/provision/pg_repository.go:345`：任务响应将 `current_step_name IS NULL` 归入 `self_config`，但 `module=self_config` 查询显式要求该字段非空。初始化阶段且尚未写入步骤名的策略任务可能不会出现在“参数自配置”筛选结果中。该问题不影响默认“所有任务”视图，建议后续统一分类表达式与筛选谓词。

### INFO

1. 全量 `go test ./...` 仅在未改动的 `internal/task` 包失败：测试 Redis 端口拒绝连接，`TestService_PG_GetTask_TerminalTombstoneReturnsDurableDetails` 断言失败；本次涉及的 Go 包定向测试均通过。
2. `npm run typecheck` 被未改动的 `SystemLicense/History.tsx` 与 `SystemLicense/index.tsx` 两处 `Uint8Array<ArrayBufferLike>` / `BlobPart` 既有类型错误阻断。
3. 浏览器控制台仅出现 Ant Design `Descriptions.labelStyle/contentStyle` 的弃用警告，无 `pageerror`。
4. 变更文件 ESLint 检查为 0 error、4 warning；警告集中在既有 effect 内状态初始化模式和 `handleSubmit` 手工 memoization 依赖精度，不阻断构建或当前交互。

## 安全与质量检查

- 未发现新增运营商硬编码、裸 `panic`、字符串拼接 SQL、令牌存储或 XSS 注入点。
- 新增 SQL 使用 Squirrel 参数化表达式或固定 SQL + 参数绑定。
- 策略启用冲突检查使用事务与 PostgreSQL advisory transaction lock，避免并发启用同产品策略。
- 公共参数范围、保留区间、gNB ID 位宽和冲突分配均有输入校验与单元测试。
- 未发现 CRITICAL 问题。

## 验证结果

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./internal/provision ./internal/device ./internal/product ./cmd/app/provider`：通过。
- `cd omcmb && npm run test --workspace webcode -- src/pages/device/PlugAndPlay ../frontend-core/src/services/api/__tests__/provisionApi.test.ts`：23 个测试文件、91 个用例通过。
- `cd omcmb/webcode && npm run build`：通过。
- 变更前端文件 ESLint：通过（0 error、4 warning）。
- Docker Compose 本地 Web 栈重新构建并启动，web/app/acs 均为 Up，`http://localhost:8081/` 返回 HTTP 200。
- 真实浏览器验证策略列表和新增策略页：关键 Provisioning/Products API 均返回 200，页面未出现“未找到”“加载失败”或运行时错误。

## 结论

本次改动无 CRITICAL 问题，可提交。上述 WARNING 建议作为后续兼容性修正，不阻断当前交付。
