# Review: 日志参数管理 MML 字段补齐

日期：2026-07-20

## 结论

PASS_WITH_WARNINGS

## 范围

- `omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql`
- `docs/qa-report/mml-device-info-regroup-20260720/README.md`
- `docs/qa-report/mml-device-info-regroup-20260720/process-log.md`
- `docs/qa-report/mml-device-info-regroup-20260720/log-management.md`

## 检查结果

### CRITICAL

无。

### WARNING

- `go test ./...` 首次运行时 `internal/paramsync` 的 `TestCompletionProjectorContinuesAfterOneRunFails` 失败；该测试与本次 seed/文档变更无直接依赖。单独重跑该测试通过，判断为既有偶发测试风险。

### INFO

- `Device.LogMgmt.LogLevel` 使用 `source = custom` 写入 `param_mappings`，可避免 app 启动时 XML 参数模型重载删除 `source = builtin` 补充项。
- seed 同步维护 `target_paths` 与 `tree_node_refs`，符合本轮 MML 页面显示排查得到的真实读取口径。
- QA 文档记录了命令绑定全集与控制台按产品模型过滤的差异，便于后续复核。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./...`：首次失败于 `internal/paramsync` 偶发测试，第二次完整重跑通过。
- `cd omcgo && go test ./internal/paramsync -run TestCompletionProjectorContinuesAfterOneRunFails -count=1 -v`：通过。
- 本地数据库执行 seed Up 段：通过。
- 当前设备 `E8F2971A3DC921A03D3E4FD4A0C1` 过滤口径复核：`LST LOG_MGMT` 可见 6 项，包含 `Device.LogMgmt.LogLevel`。

## 风险

- 该 seed 为 forward-only 数据修正；Down 段不恢复被规范化的数据，沿用既有脚本策略。
- `custom` 映射是产品模型补丁，后续若 XML 模型正式补充 `LogLevel`，可以清理重复语义来源，但当前唯一约束按 `private_path` 保证不会重复展示。
