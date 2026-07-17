# REVIEW_95e38c9_codex_param-sync

## 结论

PASS_WITH_WARNINGS

本次变更将模型/底本上传终态接入 durable parameter sync，请求元数据扩展合入 `000001_init_schema.sql`，并将手动离线参数同步默认策略调整为 `reject`。审查未发现阻断提交的 CRITICAL 问题。

## 审查范围

- `omcgo/internal/provision/*`
- `omcgo/internal/paramsync/*`
- `omcgo/internal/device/*`
- `omcgo/cmd/app/provider/*`
- `omcgo/cmd/app/etc/config.*.yaml`
- `omcgo/migrations/000001_init_schema.sql`
- `omcmb/webcode/src/pages/device/DeviceList/*`
- `docs/qa-report/20260717-param-sync-simulated-device-*.md`

## 发现

### WARNING-1: `model_upload_intents` 仍是 schema 预埋

`000001_init_schema.sql` 已包含 `model_upload_intents` 及 admission/recovery 表，但当前业务代码没有写入 `model_upload_intents`；模型上传终态通过 `parameter_sync_requests.model_upload_intent_id` 记录 discovery log ID 并提交 durable sync。

影响：不影响当前模型上传后触发参数同步的主链路，但如果后续要依赖 intent 表做调度/审计，需要补对应 repository/service 写入和状态流转。

## 验证

- `cd omcgo && go build ./...` — 通过
- `cd omcgo && go test -count=1 ./internal/paramsync ./internal/provision ./internal/device ./internal/core/appconfig ./cmd/app/provider` — 通过
- `cd omcmb && npm run typecheck` — 通过
- `cd omcmb && npm run test --workspace webcode -- src/pages/device/DeviceList/deviceBatchTask.test.ts` — 通过

## 风险与影响

- 生产配置 `routing_mode: closed` 会阻断旧 device_online/bootstrap 自动同步入口，模型上传终态、周期同步、手动同步等 durable 入口继续工作。
- `manual_offline_mode` 默认变为 `reject`，离线设备手动参数同步会直接返回离线错误；显式配置 `queue` 才进入排队行为。
- 前端批量参数同步将已存在运行中同步识别为跳过成功，避免乐观转圈状态滞留。
