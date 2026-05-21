# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-21 10:25 |
| 提交 | b8d76fd5 |
| 作者 | zhanglu |
| 范围 | fullstack-alarm-device-migration |
| 变更文件数 | 14 |
| 新增行数 | +500 |
| 删除行数 | -127 |

## 变更概要

本次变更集中收口了告警规则页面、告警过滤引擎、设备在线状态兼容和动态菜单隐藏四类问题。后端重点修复了告警规则多维条件匹配和 auto-ack 状态落库，前端同步修复了告警规则列表/抽屉的数据源、回填、批量操作和旧后端 schema 兼容。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

1. omcmb/frontend-core/src/services/api/alarmApi.ts:293
   `ruleToBackendPayload()` 这次通过 `hasConditions` 避免了 partial update 误改 `filter_type`，方向正确；但当调用方显式传入 `conditions: []` 或把某一维条件清空为“空数组”时，当前实现仍只在 `length > 0` 时发送 `alarm_identifiers/device_ids/device_group_ids/alarm_sources`。这意味着“清空已有条件”未必能真正同步到后端，旧条件可能残留。该问题不阻塞本次提交，但如果后续产品允许把规则从“指定设备/指定告警”改成“无该维条件”，需要补成显式发送空数组或后端支持 clear 语义。

### 🔵 INFO (建议)

1. omcmb/webcode/src/pages/alarm/AlarmRules/AlarmRuleDrawer.tsx:209
   已选设备回填通过 `useDevicesByIds()` 逐 ID 拉详情，能修正“查看时看似未勾选”的问题；但它会按 ID 数量线性放大请求数。当前规则通常规模较小，可以接受；若后续支持大批量设备过滤，建议补一个批量按 ID 查询接口。

2. 前端本次改动主要覆盖告警规则抽屉与列表交互，但未见对应自动化测试补充。考虑到这批改动包含批量启停、查看态回填、文案映射和状态切换，后续建议补至少一条页面级交互测试，避免 UI 回归只能靠手工点测发现。

## 详细分析

### omcgo/internal/alarm/engine.go

- 60, 122, 139, 154 行附近：新增 `applyIncomingAlarmState()`，把过滤引擎可能预先写入的 `Status/AcknowledgedAt/AcknowledgedBy` 在去重更新路径和新告警路径上保留下来，修复了 auto-ack 规则执行后又被主流程强制改回 active 的问题。
- 新告警仅在 `alarm.Status == ""` 时回填 `active`，兼容过滤动作已经决定状态的场景，行为与本次测试目标一致。

### omcgo/internal/alarm/filter_engine.go

- 112 行附近：`match()` 从“按 `filter_type` 单维匹配”调整为“规则中已填写的所有维度必须同时命中”，与前端规则配置界面的实际语义一致，能修复“指定设备 + 指定告警却只命中其中一维”的误触发问题。
- 该逻辑已有 `filter_engine_test.go:139` 单测覆盖多维同时命中，`engine_filter_integration_test.go` 也补了 auto-ack 持久化验证，回归面基本覆盖到了本次后端修复点。

### omcgo/internal/device/device_service.go

- 600 行附近：收到 Inform 后显式把 `device.IsOnline = true`，修复了设备被离线探测器标记离线后，即使继续正常 Inform 也无法恢复在线显示的问题。
- `service_test.go` 同步补了从离线恢复在线的断言，修改闭环完整。

### omcmb/frontend-core/src/services/api/alarmApi.ts

- 293 行附近：`hasConditions` 防止编辑规则时未改条件却仍重算并覆盖 `filter_type`，这正对了本次“启用状态切换/编辑后条件丢失”的一类根因。
- 上述 warning 仍然成立：目前“防误改”已修，但“显式清空条件”语义尚未完整表达。

### omcmb/frontend-core/src/services/api/deviceApi.ts

- 154, 171 行附近：新增 `deriveLegacyLifecycle()`，并在 `lifecycle_state/is_online` 缺失时回退到老 `status` 字段，能兼容 Docker 运行库未完全升级到 T-0162 双字段的场景，和用户现场遇到的“设备在线状态显示异常”相符。

### omcmb/frontend-core/src/hooks/api/useDevices.ts

- 59 行新增 `useDevicesByIds()`，为规则查看态补拉未出现在当前分页内的已选设备提供了最小侵入的 Hook，符合现有 React Query 使用方式。

### omcmb/webcode/src/pages/alarm/AlarmRules/AlarmRuleDrawer.tsx

- 87, 209, 326-328 行附近：去掉无实际意义的 `default` 动作选项，告警库改为真实 `useAlarmDefinitionList()` 数据源，且在查看/编辑时通过条件解析与补拉设备详情恢复已选状态，基本对齐了用户反馈的多个 UI 问题。
- 设备类型推断与告警 eventType/severity 映射仍属于前端展示层转换，未引入类型退化或不安全写法。

### omcmb/webcode/src/pages/alarm/AlarmRules/index.tsx

- 247, 273 行附近：新增批量启用/禁用操作，并补齐文案与显式“查看”入口，能直接解决当前页面无法批量操作和查看入口过深的问题。
- 本次 diff 未见 `any`、XSS 风险或 token 处理回退。

### omcgo/migrations/seed/000146_disable_custom_alarm_menu.sql

- 该 seed 迁移通过 `permission_key = 'alarm:custom-stats'` 及其按钮子节点统一置 `disabled + hide`，与动态菜单由 DB 驱动的现状一致，也提供了可逆的 Down 迁移。
- SQL 为简单 UPDATE，无字符串拼接、无破坏性 DDL，风险较低。

## 业务完整性检查

业务链路完整，无遗漏。后端规则匹配与状态落库均补了对应测试；前端规则列表/抽屉与 API 适配层同步修改，没有出现单侧变更导致的明显空壳链路。

## 业务影响范围检查

变更范围可控，主要影响 alarm/device 两个模块及其前端消费层。未发现接口签名破坏、共享 model 不兼容、事件契约变更或新的跨模块循环依赖。

## 前后端一致性检查

前后端一致性总体正常。本次后端把多维条件解释为“已填写维度全部命中”，前端规则编辑器也已按 `conditions[]` 多维回填；设备接口则通过 `deviceApi` 做了新旧 schema 双兼容，能缓解运行环境未完全迁移时的展示问题。

## 代码质量回退检查

未发现代码质量回退。没有删除测试、绕过错误处理、降级安全措施、引入 `any` 或用硬编码替代现有配置的情况。

## 配套更新提醒

- **文档**: 无需强制更新；若后续把“告警规则条件可清空”作为正式能力开放，建议补充接口契约说明。
- **单元测试**: 后端已有测试覆盖；前端建议后续为告警规则页面补 1 条交互测试。
- **端到端测试**: 当前提交以 bugfix 为主，建议后续在真实运行环境补做一次“指定设备 + 指定告警 + auto-ack”端到端复验。

## 安全检查

未发现安全问题。

## 性能检查

未发现显著性能问题。唯一需要关注的是按 ID 补拉设备详情会产生线性请求数，当前规模下可接受。

## 测试覆盖

已知验证包括：`go test ./internal/alarm/...`、`go test ./internal/device/...`、`cd omcmb/webcode && npm run typecheck` 通过；本次后端新增逻辑均有针对性测试覆盖。前端页面层仍以手工验证为主。

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 1 |
| INFO | 2 |

**审查结论**: `PASS_WITH_WARNINGS`
