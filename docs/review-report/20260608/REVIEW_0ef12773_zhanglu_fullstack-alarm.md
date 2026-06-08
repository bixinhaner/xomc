# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-08 03:26 |
| 提交 | 0ef12773 |
| 作者 | zhanglu |
| 范围 | fullstack-alarm |
| 变更文件数 | 11 |
| 新增行数 | +104 |
| 删除行数 | -14 |

## 变更概要

本次变更修复了告警列表按等级多选筛选失效的问题：前端不再只发送首个 severity，后端也新增了 CSV 多值解析并在查询层使用 IN 过滤。

同时，告警相关页面的时间范围筛选升级为可精确到时分秒，并补充了后端 handler 单测与前端 API 契约测试，配合前端 typecheck 完成验证。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

- 建议后续在真实接口链路上补一条 E2E 回归：覆盖 `severity=1,3` 这类 CSV 多值过滤，当前已覆盖前端 query 序列化和后端 handler/unit，但尚未覆盖真实 DB 查询链路。

## 详细分析

### `omcgo/internal/alarm/handler.go`

- 已正确将单值 severity 解析扩展为 `parseSeverities`，并在 [handler.go](omcgo/internal/alarm/handler.go#L92) 与 [handler.go](omcgo/internal/alarm/handler.go#L170) 两个入口分别处理活动/历史告警查询。
- `parseSeverities` 在 [handler.go](omcgo/internal/alarm/handler.go#L427) 做了 trim、去重和非法值跳过，没有引入 panic 或错误输入放大路径。

### `omcgo/internal/alarm/pg_store.go`

- 查询层在 [pg_store.go](omcgo/internal/alarm/pg_store.go#L370) 和 [pg_store.go](omcgo/internal/alarm/pg_store.go#L433) 使用 `squirrel.Eq` 承接切片，能够安全生成 `IN (...)` 参数化条件，没有退化成字符串拼接 SQL。

### `omcgo/internal/alarm/store.go`

- `AlarmFilter` 在 [store.go](omcgo/internal/alarm/store.go#L14) 新增 `Severities` 字段，属于局部扩展，没有破坏既有单值 `Severity` 兼容路径。

### `omcgo/internal/alarm/handler_test.go`

- 新增的 [handler_test.go](omcgo/internal/alarm/handler_test.go#L93) 和 [handler_test.go](omcgo/internal/alarm/handler_test.go#L480) 覆盖了单值/多值 severity 解析行为，能直接防住“多选退化成首项”这一回归。

### `omcmb/frontend-core/src/services/api/alarmApi.ts`

- [alarmApi.ts](omcmb/frontend-core/src/services/api/alarmApi.ts#L383) 已将多选 severity 序列化为 CSV 数字串，和后端 query 约定保持一致；没有引入 `any` 或绕开既有映射表的类型回退。

### `omcmb/frontend-core/src/services/api/__tests__/alarmApi.test.ts`

- 新增测试在 [alarmApi.test.ts](omcmb/frontend-core/src/services/api/__tests__/alarmApi.test.ts#L17) 校验 `['critical', 'minor'] -> '1,3'`，对这次前端契约修复是有效的最小保护。

### `omcmb/webcode/src/components/FilterBar/index.tsx`

- [FilterBar/index.tsx](omcmb/webcode/src/components/FilterBar/index.tsx#L21) 以 `showTime` 可选字段的方式扩展 `date-range`，改动保持向后兼容；未影响默认日期范围用法。

### `omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx`

- [CurrentAlarms/index.tsx](omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx#L186) 为告警时间范围启用 `showTime: true`，与用户报告的问题直接对应。

### `omcmb/webcode/src/pages/alarm/HistoricalAlarms/index.tsx`

- [HistoricalAlarms/index.tsx](omcmb/webcode/src/pages/alarm/HistoricalAlarms/index.tsx#L151) 同步启用时分秒选择，保持当前/历史告警筛选体验一致。

### `omcmb/webcode/src/pages/alarm/CustomAlarmStats/index.tsx`

- [CustomAlarmStats/index.tsx](omcmb/webcode/src/pages/alarm/CustomAlarmStats/index.tsx#L291) 跟随 FilterBar 能力升级，避免统计页与主列表页时间筛选表现分裂。

### `omcmb/webcode/src/pages/alarm/AlarmSync/index.tsx`

- [AlarmSync/index.tsx](omcmb/webcode/src/pages/alarm/AlarmSync/index.tsx#L114) 同步启用 `showTime`，改动小且没有引入额外业务逻辑。

## 业务完整性检查

- Handler-Service-Repository 链路：通过。查询链路改动覆盖 handler → filter struct → repository SQL 过滤，没有停在单层转发。
- 路由注册：无新增路由，不涉及遗漏注册。
- 迁移文件配套：无 schema 变更，不需要 migration。
- API 服务配套：通过。前端 `alarmApi` 已与后端 severity CSV 约定同步。
- Hook / Mock 配套：本次未新增 hook；前端 API 新增了契约测试。
- E2E 测试用例：建议补充。当前没有真实接口级 CSV 多 severity 回归。

## 业务影响范围检查

- 接口签名变更：`AlarmFilter` 新增 `Severities` 为向后兼容扩展，已有调用方不受破坏。
- API 响应格式变更：无。
- 共享组件变更：`FilterBar` 的 `showTime` 为可选字段，旧页面不受影响。
- 跨模块依赖：未引入新的不合理依赖。

## 前后端一致性检查

- 接口路径一致：无改动。
- 请求参数一致：通过。前端 `severity=1,3` 与后端 `parseSeverities` 约定一致。
- 枚举值一致：通过。`critical/major/minor/warning` 仍通过既有映射表转成 1/2/3/4。

## 代码质量回退检查

- 删除测试用例：无。
- 删除错误处理：无。
- 引入 any/interface{}：无。
- 硬编码替代配置：无。
- TODO/HACK 残留：无新增。

## 配套更新提醒

- 建议同步更新端到端测试：为告警列表补一条多 severity CSV 过滤回归。
- 文档更新：本次属于现有筛选行为修复，当前无需额外文档变更。

## 审查结论

**PASS_WITH_WARNINGS**

代码层面未发现阻塞提交的问题，可以提交。唯一建议是后续补一条真实接口级 E2E 回归，防止多选 severity 过滤在 DB 查询链路再次退化。