# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-05 06:50 |
| 提交 | cbde93b4 |
| 作者 | zhanglu |
| 范围 | fullstack-alarm |
| 变更文件数 | 18 |
| 新增行数 | +339 |
| 删除行数 | -48 |

## 变更概要

本次变更围绕告警域做了一组连续修复和收尾：后端补齐告警自动确认/自动清除备注在活跃态与历史态之间的传递和落库，前端修正当前/历史告警更新时间显示，并收敛告警规则与告警库页面的若干交互细节。另有一组 alarm-definition 过滤与事件类型兼容性调整，用于支持按 loaded_from 下钻和数值化 event type 编辑。

## 审查发现

### 🔴 CRITICAL (严重)

> 必须在提交前修复的问题

无

### 🟡 WARNING (警告)

> 建议修复，不阻塞提交

无

### 🔵 INFO (建议)

> 改进建议，可选择性采纳

1. `loaded_from` 过滤链路已覆盖前后端契约与 definition 包编译，但目前没有针对 handler/repository 的行为级测试；后续如该筛选继续扩展，建议补一组 repository 条件拼接测试和 handler 查询参数映射测试。

## 详细分析

### `omcgo/internal/alarm/engine.go`

- 自动确认时补齐 `AckNote` 传递，避免已有活跃告警在去重更新路径上丢失确认备注。
- `Clear` 逻辑抽出 `clearActiveAlarm` 复用，自动清除路径不再重复走 ID 二次加载。
- 自动清除归档时优先复用来路上的 `ClearNote` / `ClearedBy`，修复历史记录备注与清除人漂移问题。

### `omcgo/internal/alarm/filter_engine.go`

- 自动确认和自动清除动作现在都在规则命中时直接写回备注字段，业务意图清晰。
- 默认备注文案具备 rule name fallback，未见硬编码运营商或越层依赖。

### `omcgo/internal/alarm/pg_store.go`

- `UpdateActive` 现已持久化 `ack_note`，修复此前内存态有值但数据库无值的状态漂移。
- 历史归档 `updated_at` 改为优先使用业务时间字段，能避免零时间串入前端。

### `omcgo/internal/alarm/engine_test.go`

- 新增自动确认备注、自动清除备注与更新时间保留测试，覆盖了本次修复的关键回归面。

### `omcgo/internal/alarm/pg_store_test.go`

- 新增 `historyUpdatedAt` 单测，覆盖 `last_updated_at -> updated_at -> zero` 三条分支。

### `omcgo/internal/alarm/definition/*.go`

- `loaded_from` 过滤已从 handler 查询参数、repository filter struct 到 SQL 条件拼接全链路打通。
- `__empty__` 哨兵值用于表示手工新增定义，前后端语义一致。

### `omcmb/frontend-core/src/services/api/alarmApi.ts`

- `resolveUpdatedAt` 对 `0001-01-01...` 做业务时间过滤，修复当前/历史告警更新时间展示异常。
- 回退顺序 `last_updated_at -> updated_at -> raised_at` 与当前告警业务语义一致。

### `omcmb/frontend-core/src/services/api/alarmDefinitionApi.ts`

- 后端 `event_type` 与 `severity.code` 兼容数值/字符串双形态，避免已有数据源格式不统一时前端解析失败。
- `loaded_from` 过滤参数与后端 `__empty__` 约定一致。

### `omcmb/webcode/src/pages/product/alarm-library/AlarmDefinitionDrawer.tsx`

- 事件类型编辑由自由文本切到枚举选择，能约束输入并与后端数值事件类型保持一致。
- severity 默认值改为依赖实际下拉源，避免固定值和字典配置漂移。

### `omcmb/webcode/src/pages/product/alarm-library/index.tsx`

- drill-down URL 现在同时携带 `neType` 和 `loadedFrom`，列表与汇总卡片之间的过滤上下文保持一致。
- `loaded_from` 为空时展示“手工新增”，比“未回填”更符合当前业务语义。

### `omcmb/webcode/src/pages/alarm/AlarmDetail/index.tsx`

- 移除设备名称展示列，符合本轮界面精简诉求。

### `omcmb/webcode/src/pages/alarm/AlarmRules/AlarmRuleDrawer.tsx`

- 设备选择表移除设备名称列，创建/详情两条共享路径同步生效。

## 业务完整性检查

业务链路完整，无遗漏。

## 业务影响范围检查

变更范围可控，主要影响 alarm 与 alarm-definition 相关页面和存储映射；未发现跨模块接口签名破坏或额外 schema 依赖。

## 前后端一致性检查

本次变更同时涉及前后端，`loaded_from` 过滤、`event_type` 数值兼容、更新时间回退策略和自动确认/清除备注的展示契约已对齐，未发现路径、参数或响应字段不一致。

## 代码质量回退检查

未发现代码质量回退。

## 配套更新提醒

- **文档**: 无需更新。本次主要是 bugfix 和已有页面交互收敛，未新增公开接口。
- **单元测试**: 告警引擎与归档时间已补测试；建议后续为 `alarm/definition` 的 `loaded_from` 过滤补行为级测试。
- **端到端测试**: 本次未新增 REST 端点；如后续继续扩展 alarm-definition 下钻行为，可考虑补浏览器级回归用例。

## 安全检查

未发现安全问题。

## 性能检查

未发现性能问题。

## 测试覆盖

已执行 `go test ./internal/alarm/...` 与 `cd omcmb/webcode && npm run typecheck`，均通过。当前修复点在告警引擎、归档时间处理和前端类型映射上均有直接验证。

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 1 |

**审查结论**: `PASS`
