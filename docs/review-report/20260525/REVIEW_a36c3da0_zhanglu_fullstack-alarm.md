# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-25 16:12 |
| 提交 | a36c3da0 |
| 作者 | zhanglu |
| 范围 | fullstack-alarm |
| 变更文件数 | 16 |
| 新增行数 | +422 |
| 删除行数 | -61 |

## 变更概要

本次变更围绕告警详情与操作闭环做了前后端联动修复。后端补齐了历史告警详情查询和 `additional_info` 归档/读取链路，并通过 migration 为 `alarms_history` 增加 `additional_info` 字段；前端统一了当前告警、历史告警、设备详情页活动告警的详情入口与操作菜单，同时补充“附加信息 / 附加文本”展示。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

1. 历史告警的 `additional_info` 只会对本次修复后新归档的数据完整生效；修复前已写入 `alarms_history` 的旧记录不会自动补回附加信息，属于存量数据限制，不阻塞提交。
2. `alarm.historyClearConfirmMsg` 词条在前端页面已不再使用，当前历史告警页已统一切到 `alarm.deleteConfirmMsg`，后续可以做一次 i18n 清理，避免冗余键继续漂移。

## 详细分析

### `omcgo/internal/alarm/handler.go`

- [通过] `GetByID` 在活动告警查不到时回退到历史告警查询，返回路径清晰，404 语义保持不变。
- [通过] 未引入裸 panic，错误处理仍走统一 HTTP 错误返回。

### `omcgo/internal/alarm/pg_store.go`

- [通过] `Archive` 已将 `additional_info` 持久化到 `alarms_history`，与 `ListHistory` / `GetHistoryByID` 的反序列化路径形成闭环。
- [通过] 查询仍采用参数化 SQL / Squirrel 构建，没有引入字符串拼接 SQL。
- [信息] `scanHistoryAlarm` / `ListHistory` 对 `additional_info` 的反序列化采用容错处理，符合历史脏数据兼容预期。

### `omcgo/internal/alarm/store.go`

- [通过] `AlarmStore` 新增 `GetHistoryByID` 后，相关 mock 已同步补齐，没有留下接口签名漂移导致的编译裂缝。

### `omcgo/internal/alarm/handler_test.go`

- [通过] 新增历史告警回退测试，覆盖了 `GetByID` 的新增行为路径。

### `omcgo/migrations/000181_alarm_history_additional_info.sql`

- [通过] migration 仅做新增列与回滚删除列，变更边界清晰。
- [通过] 版本号已避开已有 `000175_*`，不会再触发 goose duplicate version 问题。

### `omcmb/frontend-core/src/services/api/alarmApi.ts`

- [通过] `mapBackendAlarm` 已把 `additional_info.additional_text` 映射到前端 `Alarm.additionalText`，与详情页展示需求一致。

### `omcmb/frontend-core/src/types/alarm.ts`

- [通过] 新增 `additionalText` 字段，类型扩展最小且未引入 `any`。

### `omcmb/webcode/src/pages/alarm/AlarmDetail/index.tsx`

- [通过] 详情抽屉改为优先调用 `useAlarmById` 拉取完整数据，而不是依赖列表投影对象，方向正确。
- [通过] `additional_information`、`additional_text`、`managed_object_instance` 的展示做了用户侧清洗，避免把底层保留字段直接暴露到 UI。

### `omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx`

- [通过] 当前告警操作入口收敛到三点菜单，详情/确认/清除操作更一致。

### `omcmb/webcode/src/pages/alarm/HistoricalAlarms/index.tsx`

- [通过] 历史告警页面的单条与批量操作已统一使用删除语义，并调用历史删除接口，语义与业务流程一致。
- [信息] 页面已切换到 `alarm.deleteConfirmMsg`，`alarm.historyClearConfirmMsg` 词条目前只剩资源定义。

### `omcmb/webcode/src/pages/device/DeviceDetail/index.tsx`

- [通过] 设备详情页活动告警新增详情、确认、反确认、清除操作，补齐了从设备详情进入活动告警后的操作闭环。
- [通过] `useState` 等运行时依赖已补齐，之前的 `useState is not defined` 回归已消除。

## 业务完整性检查

| 检查项 | 结论 | 说明 |
|------|------|------|
| Handler-Service-Repository 链路 | 通过 | 后端变更集中在 handler/store 持久化链路，控制路径完整 |
| 路由注册 | 通过 | 复用既有 `/alarms/:id` 路由，无新增漏注册端点 |
| 迁移文件配套 | 通过 | 新增历史字段已提供 migration |
| 错误码注册 | 通过 | 未新增业务错误码 |
| API 服务配套 | 通过 | 前端 `alarmApi` 与后端详情/历史字段同步 |
| Hook 配套 | 通过 | 详情页复用已有 `useAlarmById`，未出现孤儿 API |
| Mock 配套 | 可接受 | 本次主要修复真实链路，未新增必须补 mock 的新接口 |
| E2E/单测配套 | 通过 | 后端新增测试，前端完成 typecheck，告警模块 go test 已通过 |

## 业务影响范围检查

| 检查项 | 结论 | 说明 |
|------|------|------|
| 接口签名变更 | 通过 | `AlarmStore` 新方法已同步所有测试 mock |
| 数据库 Schema 变更 | 通过 | 仅新增 `alarms_history.additional_info`，查询侧已同步读取 |
| API 响应格式变更 | 通过 | 前端已消费新增历史详情字段 |
| 跨模块引用 | 通过 | 未引入循环依赖或跨域硬耦合 |

## 前后端一致性检查

| 检查项 | 结论 | 说明 |
|------|------|------|
| 接口路径一致 | 通过 | 前端详情仍走既有 `/alarms/:id`，后端已补历史回退 |
| 响应字段一致 | 通过 | `additional_info` / `additional_text` 已在前端 mapper 对齐 |
| 枚举值一致 | 通过 | `dealState`、`eventType` 映射未破坏原契约 |

## 配套更新提醒

- 建议在后续告警模块文档中补一句说明：旧历史告警不会自动补回本次新增的附加信息字段，避免联调时把存量数据限制误判为新 bug。

## 审查结论

**PASS**

本次变更已形成前后端闭环，且已通过针对性验证：

- `go test ./internal/alarm/...`
- `npm run typecheck`
- 8081 web 容器重建并返回 200

未发现阻塞提交的问题，可提交。