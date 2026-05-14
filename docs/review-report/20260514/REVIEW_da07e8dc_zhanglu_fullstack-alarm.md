# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-14 09:41 |
| 提交 | da07e8dc |
| 作者 | zhanglu |
| 范围 | fullstack-alarm |
| 变更文件数 | 16 |
| 新增行数 | +430 |
| 删除行数 | -86 |

## 变更概要

本次变更修复了告警列表前后端筛选链路，补齐了 `event_type`、`ne_type`、时间范围、处理状态、已读状态等查询参数在页面、API 和后端过滤器之间的传递。后端同时补上了从设备表回填告警制式的逻辑，并新增两条 seed 迁移隐藏已经下线的旧告警库菜单入口。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

1. [omcgo/internal/alarm/handler.go](omcgo/internal/alarm/handler.go) 的历史告警 `ne_type` 过滤已补齐，但当前新增的 handler 级回归测试主要覆盖了 active 列表。后续可补一个 history 场景测试，避免同类参数再次只修一侧。

## 详细分析

### `omcgo/internal/alarm/*`

- `handler.go` 把 `start_time` / `end_time` / `ne_type` 映射到 `AlarmFilter`，active 与 history 两条查询链路都补齐了入口参数。
- `pg_store.go` 对 active/history 查询统一增加了 `devices` 左连接，并用 `COALESCE` 与别名归一化实现制式过滤兼容，能兼容历史脏数据和回填不足场景。
- `receiver.go` 在告警 payload 不带制式时，从设备仓储按 `device_id` / `device_sn` 回填 `technology`，与前端新增筛选项形成闭环。
- `engine_test.go`、`handler_test.go`、`pg_store_event_type_test.go`、`receiver_test.go` 补了对应回归覆盖，未见类型安全或错误处理退化。

### `omcmb/frontend-core` 与 `omcmb/webcode`

- `alarmApi.ts` 把处理状态、已读状态和 `ne_type` 正确映射到后端查询参数，和 handler 的字段命名保持一致。
- `FilterBar/index.tsx` 增加日期范围序列化后再写入 `sessionStorage`，修复了日期筛选在刷新或切页后无法正确回填的问题。
- `CurrentAlarms` / `HistoricalAlarms` / `AlarmDetail` 统一使用基站制式标签工具，避免同一字段在列表、详情、不同页面上展示不一致。

### `omcgo/migrations/seed/*`

- `000088` 按历史 UUID 隐藏旧告警库菜单，并在迁移内备份原始 `show_status` 以支持精确回滚。
- `000089` 用稳定业务键再次兜底隐藏，并通过同一备份表按菜单 ID 恢复原始状态，避免 Down 误把原本隐藏的记录恢复为 `show`。

## 业务完整性检查

业务链路完整，无遗漏。前端筛选字段、共享 API 参数、后端 handler 解析、store 过滤与接收侧制式回填已经形成闭环；旧告警库菜单隐藏也补了 seed 迁移入口。

## 业务影响范围检查

变更范围可控，主要影响 F04 告警查询与展示链路，以及 worker 启动时 alarm receiver 的设备信息回填能力。未发现接口签名破坏、跨模块循环依赖或 schema 破坏性变更。

## 前后端一致性检查

前后端接口契约一致：

- `ne_type`、`event_type`、`start_time`、`end_time`、`status`、`is_read` 的命名已在 API 与 handler 两侧对齐。
- 告警制式字段在页面展示层统一转换为 `eNB (LTE)` / `gNB (NR)` 等标签，和后端 `technology` / `neType` 实际值兼容。

## 代码质量回退检查

未发现代码质量回退。

## 配套更新提醒

- **文档**: 无需更新。变更属于告警筛选与历史菜单收口，未引入新的外部接口文档要求。
- **单元测试**: 已有测试覆盖 active handler、technology alias 和 receiver backfill；建议后续补 history handler 参数回归。
- **端到端测试**: 如需把本次筛选修复纳入长期回归，建议后续在告警页面或 API E2E 中增加 `ne_type` / 时间范围组合过滤断言。

## 安全检查

未发现安全问题。

## 性能检查

未发现性能问题。`devices` 左连接增加了查询复杂度，但当前过滤条件均为告警列表读取路径上的合理扩展，未见明显退化信号。

## 测试覆盖

- `cd omcgo && go test ./internal/alarm/...` 通过
- `cd omcmb/webcode && npm run typecheck` 通过

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 1 |

**审查结论**: `PASS`

- **PASS**: 无 CRITICAL 和 WARNING
- **PASS_WITH_WARNINGS**: 无 CRITICAL，有 WARNING
- **NEEDS_FIX**: 存在 CRITICAL 问题