# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-19 07:59 |
| 提交 | eae61065 |
| 作者 | zhanglu |
| 范围 | fullstack-multi |
| 变更文件数 | 9 |
| 新增行数 | +173 |
| 删除行数 | -18 |

## 变更概要

本次变更主要修复两类问题：一是告警同步链路在新增告警时补齐 DeviceID、Carrier、Technology，避免同步落库后出现设备维度字段缺失；二是校正 TR-069 GetParameterValues / GetParameterNames 模板的子元素命名，并把设备列表批量操作确认文案改为国际化模板插值。

后端同时补充了针对告警字段回填和 SOAP 模板输出的聚焦测试，前端改动规模较小，主要是文案键和调用方式对齐。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

- `omcgo/internal/acs/rpc/dispatcher_test.go` 所在包的聚焦验证未完全转绿：`go test ./internal/acs/rpc` 仍被该文件中 Download 相关既有断言阻塞。现有模板在 `omcgo/pkg/soap/templates.go` 保持 `<cwmp:CommandKey>`、`<cwmp:Username>` 等带前缀子元素，这与最近提交 `24161091` 的方向一致；失败断言位于该测试文件既有区段而非本次 diff，属于仓库现存测试债，但会影响同包回归信号的完整性。

### 🔵 INFO (建议)

- `omcgo/internal/alarm/sync_processor_test.go` 已覆盖设备仓库回填和本地活动告警兜底两条主路径；如果后续继续演进同步链路，建议再补一条 `deviceReader` 返回错误时的 no-op 路径测试，锁定当前“尽力回填、失败不阻断同步”的语义。

## 详细分析

### `omcgo/internal/alarm/sync_processor.go`

- 在 `processSync` 中把“先查本地活动告警、再构造 remoteAlarms”的顺序前移，允许新告警复用现存告警上的 `DeviceID`、`Carrier`、`Technology`。
- 新增 `resolveDeviceFields`，当本地活动告警不足以提供完整字段时，再通过 `deviceReader.GetBySerialNumber` 做一次补查；读取失败时直接回退到已有值，不阻断同步主流程。

### `omcgo/cmd/app/provider/alarm.go`

- App 进程初始化 AlarmSyncProcessor 时显式注入 `device.NewPgDeviceRepository(c.PgPool)`，让同步处理器具备按 SN 回填设备字段的能力。

### `omcgo/cmd/worker/main.go`

- Worker 进程对 AlarmSyncProcessor 做了与 App 一致的 `WithDeviceReader` 注入，避免多进程路径行为漂移。

### `omcgo/internal/alarm/sync_processor_test.go`

- 新增两个聚焦测试，分别验证“本地无活动告警时走设备仓库回填”和“设备仓库缺失时沿用本地活动告警字段”的行为，测试覆盖与实现改动匹配。

### `omcgo/pkg/soap/templates.go`

- 将 GetParameterValues / GetParameterNames 的 body 子元素改为不带 `cwmp:` 前缀的形式，符合仓库已有 TR-069 结论：RPC 方法元素保留 `cwmp:`，其子元素按未限定名输出。

### `omcgo/internal/acs/rpc/dispatcher_test.go`

- 为 GPV/GPN 新增正反向断言，明确要求 `ParameterNames`、`ParameterPath`、`NextLevel` 不带 `cwmp:` 前缀，能有效防回退。
- 同文件内 Download 相关断言仍是现存失败源，见上方 WARNING。

### `omcmb/webcode/src/pages/device/DeviceList/index.tsx`

- 批量操作确认框从字符串拼接切换到 i18n 模板插值，避免不同语言下数量、动作顺序不可控。

### `omcmb/frontend-core/src/i18n/en-US/index.ts`

- 补充 `device.batch.actionConfirm` 英文文案键，和页面调用保持一致。

### `omcmb/frontend-core/src/i18n/zh-CN/index.ts`

- 补充 `device.batch.actionConfirm` 中文文案键，和页面调用保持一致。

## 业务完整性检查

- Handler-Service-Repository 链路：通过。未新增空壳 handler；仅在现有 DI 装配点补充 `deviceReader` 注入。
- 路由注册：不涉及。
- 迁移文件配套：不涉及 schema 变更。
- 错误码注册：不涉及新增业务错误码。
- API 服务 / Hook / Mock：不涉及新增 API。
- 测试配套：后端已补充针对性单测；前端变更为纯文案/调用对齐，类型检查通过。

## 验证记录

- `go test ./cmd/app/provider ./cmd/worker ./internal/alarm ./internal/acs/rpc ./pkg/soap`
  - `cmd/app/provider`：无测试文件，编译通过
  - `cmd/worker`：通过
  - `internal/alarm`：通过
  - `pkg/soap`：通过
  - `internal/acs/rpc`：失败，原因见 WARNING（Download 既有断言）
- `cd omcmb/webcode && npm run typecheck`：通过

## 审查结论

PASS_WITH_WARNINGS

当前 diff 未发现需要阻塞提交的 CRITICAL 问题；告警同步字段回填、SOAP 模板修正和前端文案对齐都具备直接价值，且核心改动已有聚焦测试覆盖。残留风险是 `internal/acs/rpc` 包存在与本次 diff 无关的既有失败断言，导致该包无法给出完全绿色的回归信号，建议后续单独清理。