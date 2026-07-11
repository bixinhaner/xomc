# Task 7 report — 脚本执行预检与任务快照

## Implemented

- 新增 `ScriptExecutionRequest` 和 `Service.CreateScriptExecution`。
- 执行接口只接受任务名称、调度和重试策略；命令、设备和计划始终从已导入脚本的 `plan_items` 深复制。
- 执行创建前调用可注入的 `ScriptImportValidationRunner` 做动态预检；错误返回逐行结果并阻止创建，警告在 `confirm_warnings=false` 时返回冲突错误。
- 新增 `POST /api/v1/mml/scripts/:id/executions`，严格 JSON 解码并返回 422/409 及逐行问题。
- 任务保存 `script_content_sha256`、`script_validation_version`，周期子任务克隆时继续保留两个字段。
- Scheduler 在 scheduled/periodic fanout 前重新预检；阻断错误将实例置为 failed、清理触发时间并持久化逐行问题。
- 应用 DI 将 TXT 导入 validator 同时注入执行预检。

## Tests

- `go test ./internal/mml -run 'TestCreateScriptExecution|TestScheduler_.*Preflight|TestSequencer_.*Continue' -count=1 -v`
- `go test ./internal/mml -count=1`
- `go build ./cmd/app`
- `git diff --check`

All commands passed. The pre-existing unrelated carrier OUI test was not modified.

## Commit

`5ef20c71 feat(mml): 从导入脚本创建执行快照`
