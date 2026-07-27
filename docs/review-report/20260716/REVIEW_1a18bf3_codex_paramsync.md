# 参数同步恢复与 result worker 改造提交审查

- 审查时间：2026-07-16
- 审查人：Codex
- 分支：`fix/param-sync-recovery-worker`
- 结论：PASS_WITH_WARNINGS

## 审查范围

- `omcgo/internal/paramsync/*`
- `omcgo/internal/core/event/nats_bus.go`
- `omcgo/internal/core/appconfig/*`
- `omcgo/cmd/app/provider/paramsync.go`
- `omcgo/cmd/app/etc/config.*.yaml`
- `omcmb/frontend-core/src/services/api/deviceParameterApi.ts`
- `omcmb/frontend-core/src/types/deviceParameter.ts`
- `omcmb/webcode/src/pages/device/DeviceDetail/ParameterTreeTab/index.tsx`
- `docs/qa-report/20260716-param-sync-*.md`

## 重点检查

- DB recovery 是否复用 `PGResultProcessor.Process()`，避免绕过 staging、失败收敛、finalize 和 terminal outbox 语义。
- result consumer shard/queue 设计是否保持 ACK-after-commit。
- NATS pull consumer 并发、batch、AckWait、MaxAckPending 是否有背压边界。
- 既有 durable consumer 配置升级是否会导致 app 启动失败或 pending result 丢失。
- 前端 `stalled_finalizing` 字段映射是否与后端响应契约一致。

## Findings

### CRITICAL

无。

### WARNING

- `go test ./...` 第一次运行时 `internal/paramsync` 的 completion projector 两个用例出现一次时序失败；随后单包、失败用例单独重跑，以及第二次全量 `go test ./...` 均通过。当前判断为既有并发测试偶发，不阻断本次提交。

### INFO

- 既有 `param-sync-results-pull` durable consumer 如果已在 NATS 中创建，会优先通过 `UpdateConsumer()` 覆盖 AckWait/MaxAckPending；这是为了让新配置生效，同时避免删除 durable consumer 造成 pending result 跳过。仅当更新失败时才沿用服务端旧值并打 warning。
- ACS 直接发布 result 与 `TaskTerminalBridge` 终态转换仍可能产生同 task 重复事件；当前由 `(run_id, task_id)` 幂等写入吸收，属于可接受的 at-least-once 行为。

## 验证

- `go test -count=1 ./internal/paramsync ./internal/core/event ./cmd/app/provider ./internal/core/appconfig`：通过
- `go build ./...`：通过
- `npm run typecheck` (`omcmb`)：通过
- `git diff --check --cached`：通过
- `go test ./...`：第一次有一次 `internal/paramsync` 偶发失败；重跑 `go test -count=1 ./internal/paramsync`、失败用例单独重跑、第二次 `go test ./...` 均通过
- `npm run build` (`omcmb/webcode`)：通过
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web`：通过
- `curl -I --max-time 10 http://localhost:8081/`：`HTTP/1.1 200 OK`
- 手动创建旧版 NATS durable consumer `PARAM_SYNC / param-sync-results-pull`（AckWait `30s`、MaxAckPending `2048`）后启动新 app：app 日志出现 `updated existing pull consumer tuning`，consumer 原地更新为 AckWait `2m0s`、MaxAckPending `512`，启动成功

## 结论

当前 staged diff 未发现阻断合入问题。升级兼容风险已通过既有 durable consumer 原地更新逻辑补齐，并有纯函数测试覆盖。
