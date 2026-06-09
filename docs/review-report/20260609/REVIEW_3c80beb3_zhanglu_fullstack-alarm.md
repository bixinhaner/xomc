# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-09 08:57 +0000 |
| 提交 | 3c80beb3 |
| 作者 | zhanglu |
| 范围 | fullstack-alarm |
| 变更文件数 | 5 |
| 新增行数 | +72 |
| 删除行数 | -8 |

## 变更概要

本次变更修正当前告警“告警时间”和“更新时间”的业务语义错位问题。后端在重复告警和同步更新时保留首次 raised_at，仅刷新 last_updated_at；前端列表与详情优先使用 first_raised_at 展示首次告警时间，并据此计算持续时长。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

- 已补充后端回归测试，覆盖重复告警和同步更新两条路径；后续若再调整 active alarm 时间语义，应同步检查前端 eventTime 与 duration 的映射。

## 详细分析

### omcgo/internal/alarm/engine.go

- `applyIncomingAlarmState` 改为仅在目标 `RaisedAt` 为空时补值，避免重复告警把首次上报时间覆盖成最新一次设备时间。
- 逻辑与 `FirstRaisedAt` / `LastUpdatedAt` 现有语义一致，未引入新的接口签名或持久化字段变更。

### omcgo/internal/alarm/sync_processor.go

- 同步更新现有告警时不再覆盖 `RaisedAt`，并显式刷新 `LastUpdatedAt`，与当前告警业务语义一致。
- 该改动保持在同步更新局部路径内，未扩大到清除或新增分支，风险可控。

### omcgo/internal/alarm/engine_test.go

- 为重复告警场景增加断言，确认 `RaisedAt` 保持首次时间不变。

### omcgo/internal/alarm/sync_processor_test.go

- 新增 `TestProcessSync_PreservesFirstRaisedAtOnUpdate`，覆盖同步更新路径下的首次时间保留与更新时间刷新。

### omcmb/frontend-core/src/services/api/alarmApi.ts

- 新增 `resolveEventTime`，优先取 `first_raised_at` 作为 `eventTime`，并让 `duration` 基于同一业务时间计算，前后端展示语义一致。
- 仍保留 `first_raised_at` 缺失时回退 `raised_at`，兼容历史或未补全字段的响应。

## 业务完整性检查

- Handler-Service-Repository 链路：本次未新增链路，现有链路未被破坏。
- API 响应契约：未新增字段，仅修正前端对既有 `first_raised_at` / `last_updated_at` 的使用方式。
- 测试覆盖：已补充后端单测，覆盖本次修复的两条核心后端路径。
- 文档/E2E：本次属于既有字段语义修复，无新增接口；E2E 可后续按需要补充 UI 侧回归验证。

## 验证记录

- `cd /home/zhanglu/goomc/omcgo && go test ./internal/alarm -run 'TestProcessDuplicateAlarm|TestProcessSync_PreservesFirstRaisedAtOnUpdate'`
- `cd /home/zhanglu/goomc/omcgo && go test ./internal/alarm`
- `cd /home/zhanglu/goomc/omcmb/webcode && npm run typecheck`

## 审查结论

PASS