# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-03 22:25 |
| 提交 | 6fd27b41 |
| 作者 | zhanglu |
| 范围 | fullstack-multi |
| 变更文件数 | 9 |
| 新增行数 | +369 |
| 删除行数 | -30 |

## 变更概要

本次变更包含两部分：一是前端告警详情、设备详情活动告警、告警规则设备类型识别和告警日志文案的多处 UI 修正；二是后端新增离线超时告警清理器，在 worker 中周期扫描长时间离线设备并将其活动告警归档到历史告警。

前端侧目标是按用户验收意见移除不需要展示的字段并纠正表头；后端侧目标是补齐离线设备告警生命周期的自动收敛路径，避免活动告警长期悬挂。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

1. `omcgo/cmd/worker/main.go` 新增的离线告警清理 goroutine 通过 `context.Background()` 启动，未绑定到 worker 的统一生命周期上下文。当前进程退出时不会造成资源泄漏，但会失去显式的停止信号和一致的 shutdown 语义，建议后续接到已有的进程级 context。

### 🔵 INFO (建议)

1. 本次提交混合了前端告警 UI 修复和后端离线告警清理逻辑，变更主题偏宽。若后续继续演进，建议按“frontend alarm ui”和“worker offline alarm cleanup”拆分提交，降低回滚和追责成本。
2. 新增离线告警清理逻辑已经补了单测，并完成了 `go build ./cmd/worker` 验证；前端侧完成了 `npm run typecheck`。本轮验证覆盖达标。

## 详细分析

### `omcgo/cmd/worker/main.go`

- [WARNING] 新增 `offlineAlarmCleaner.Run(context.Background())`，启动方式绕过了 worker 生命周期上下文。建议改为复用进程 shutdown ctx，保证停止日志、ticker 停止和未来扩展的一致性。

### `omcgo/internal/device/device_repository.go`

- [INFO] 新增 `FindOfflineDevicesBefore` 查询使用 Squirrel 参数化构建，过滤条件和排序均清晰，符合仓库现有 repository 模式。
- [INFO] 查询只选择 commissioned、未删除、当前离线且 `last_offline_time` 到期的设备，业务边界明确。

### `omcgo/internal/alarm/offline_alarm_cleaner.go`

- [INFO] 新增离线告警清理器实现较完整，清理动作复用 `ClearBySync`，没有直接绕过告警引擎修改状态，业务闭环正确。
- [INFO] 失败按设备维度隔离，单台设备查询失败不会阻断整轮 sweep，符合后台巡检任务容错预期。

### `omcgo/internal/alarm/offline_alarm_cleaner_test.go`

- [INFO] 已覆盖正常清理、单设备查询失败后继续、单设备中途清理失败三条关键路径，新增功能具备最小必要测试。

### `omcmb/webcode/src/pages/alarm/AlarmDetail/index.tsx`

- [INFO] 详情抽屉已按验收要求移除“告警码/ID”“网元定位”“确认时间”，同时保留“告警标识”，与用户最新诉求一致。
- [INFO] 标题改为优先显示可能原因/具体故障，避免再次把告警标识当成主标题重复展示。

### `omcmb/webcode/src/pages/device/DeviceDetail/index.tsx`

- [INFO] 活动告警表头已修正为“告警标识 / 可能原因 / 具体故障”，与当前告警页的字段语义对齐。

### `omcmb/webcode/src/pages/alarm/AlarmRules/AlarmRuleDrawer.tsx`

- [INFO] `attachDeviceType` 现在兼容前端映射后的 `networkType='gNB'`，修复 NR 设备在告警规则设备列表中被误识别为 eNB 的问题。

### `omcmb/webcode/src/pages/log/AlarmLog/index.tsx`

- [INFO] 告警日志页中的“告警码”文案已切换为“告警标识”，与其他告警页面口径统一。

### `omcmb/frontend-core/src/i18n/zh-CN/index.ts`

- [INFO] `alarm.alarmId` 文案从“告警码”调整为“告警ID”，不会影响当前详情页隐藏该字段的行为，但可避免其他仍使用该 key 的页面继续误导。

## 业务完整性检查

- Handler-Service-Repository 链路：通过。后端新增的是后台清理组件与 repository 查询补充，不涉及残缺 handler。
- 路由注册：不适用。本次无新增 HTTP 路由。
- 迁移文件配套：通过。本次后端逻辑复用现有 `device_info.last_offline_time` 与告警表结构，无新增 schema 依赖。
- API 服务配套 / Hook 配套：不适用。本次前端修改为现有页面显示修正。
- 测试覆盖：通过。新增 backend 行为已有针对性单测；前端变更完成 typecheck 验证。

## 业务影响范围检查

- 接口签名变更：无。
- 数据库 Schema 变更：无。
- API 响应格式变更：无。
- 共享 model 变更：无。
- 中间件变更：无。

## 前后端一致性检查

- 前后端接口契约未发生变化，一致性风险低。
- 告警详情和设备详情的修正均为前端展示层调整，不影响后端返回字段。

## 代码质量回退检查

- 删除测试用例：无。
- 删除错误处理：无。
- 引入 any/interface{}：无。
- 硬编码替代配置：无明显回退。
- TODO/HACK 残留：无。

## 配套更新提醒

- 文档更新：本次主要为缺陷修复和后台清理逻辑补充，暂不构成必须更新产品文档的门槛。
- 单元测试更新：已完成 backend 新增逻辑测试。
- 端到端测试更新：若后续需要把“离线超时告警自动归档”纳入回归，建议补一个 worker/告警生命周期的集成验证脚本。

## 审查结论

**PASS_WITH_WARNINGS**

可提交。唯一 warning 为 worker 新 goroutine 未接入统一生命周期 context，不阻塞本次提交。