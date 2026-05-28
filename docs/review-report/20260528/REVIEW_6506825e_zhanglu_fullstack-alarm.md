# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-28 07:21 |
| 提交 | 6506825e |
| 作者 | zhanglu |
| 范围 | fullstack-alarm |
| 变更文件数 | 10 |
| 新增行数 | +588 |
| 删除行数 | -27 |

## 变更概要

本次变更修复了 5G 设备 TR-069 101 ALARM 的实时告警入库链路，补齐了 `Device.*` 与 `InternetGatewayDevice.*` 双前缀兼容、worker 对通用 Inform payload 的消费，以及 `device.inform.alarm` 下 ExpeditedEvent 的 direct path 解析。前端同时把活动告警页的自动刷新收敛为 React Query 原生轮询，并补上历史 `alarmHisMaxHoldTime` 为 `string` 时的 retention 兼容读取。

## 审查发现

### 🔴 CRITICAL (严重)

无。

### 🟡 WARNING (警告)

无。

### 🔵 INFO (建议)

1. 当前后端链路与前端类型检查已通过，但活动告警页的跨窗口清理消失体验仍建议在浏览器里做一次人工回归，确认实际轮询间隔与页面开关语义符合预期。

## 详细分析

### [omcgo/internal/alarm/receiver.go](omcgo/internal/alarm/receiver.go)

- worker 现在能兼容原始 `AlarmPayload` 和通用 Inform payload，并在 `device.inform.alarm` 主题下补解析 CurrentAlarm、AlarmInfo、ExpeditedEvent，direct path 语义闭环完整。
- `probable_cause` 回填逻辑保住了 `alarms_active` / `alarms_history` 的非空约束，避免 NATS 消费无限重试。

### [omcgo/internal/alarm/tr069_parser.go](omcgo/internal/alarm/tr069_parser.go)

- CurrentAlarm / ExpeditedEvent 兼容 `Device` 与 `InternetGatewayDevice` 双根路径，和实际 5G 设备上报形态一致。

### [omcgo/internal/acs/handler.go](omcgo/internal/acs/handler.go)

- ACS 事件发布侧补齐了 expedited helper 的双前缀判定，与 worker 兼容逻辑一致，没有引入新的事件契约分叉。

### [omcmb/frontend-core/src/hooks/api/useAlarms.ts](omcmb/frontend-core/src/hooks/api/useAlarms.ts)

- `useCurrentAlarms` 现支持页面级轮询配置，保持默认行为不变，兼顾复用与兼容。

### [omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx](omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx)

- 页面不再自建定时器，而是把自动刷新交给 React Query 查询层管理，并在开启或切换频率时立即 `refetch`，逻辑更稳定，后台窗口场景也更可控。

### [omcgo/internal/alarm/history_retention.go](omcgo/internal/alarm/history_retention.go)

- 对 `storage/alarmHisMaxHoldTime` 的 legacy `string` value_type 已做兼容，能覆盖旧库场景，消除了“配置保存成功但策略不生效”的兼容性缺口。

## 业务完整性检查

- Handler-Service-Repository 链路：通过。变更主要落在 ACS handler、alarm parser/receiver、前端 query/page 接线，没有破坏分层。
- 事件契约变更：通过。worker 兼容消费通用 Inform payload，属于向后兼容扩展。
- 前后端一致性：通过。前端活动告警页自动刷新开关与查询层轮询节奏对齐。
- 测试覆盖：通过。新增/调整了 ACS 与 alarm 的单测；后端相关包测试和前端 typecheck 已执行。

## 验证记录

- `cd omcgo && go test ./internal/acs/... ./internal/alarm/...` ✅
- `cd omcmb/webcode && npm run typecheck` ✅

## 审查结论

PASS
