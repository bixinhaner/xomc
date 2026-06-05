# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-05 10:20 |
| 提交 | 6b88bcc5 |
| 作者 | zhanglu |
| 范围 | alarm |
| 变更文件数 | 12 |
| 新增行数 | +312 |
| 删除行数 | -16 |

## 变更概要

本次变更修复了告警定义级别覆盖在 generic inform、expedited event、CurrentAlarm sync 三条入库路径上的不一致问题，并补上 app/worker 侧的 registry 注入顺序缺口。

同时修复了同一活动告警重复上报时的次数累加与业务更新时间语义，确保活动告警的次数、首次上报时间和最后业务更新时间能持续反映设备侧状态变化。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

- 本次改动已补齐 engine/receiver/expedited/sync 的回归测试，建议后续若再调整告警时间语义，继续优先围绕 `last_updated_at` 和前端 `updTime` 契约做增量回归。

## 详细分析

### `omcgo/internal/alarm/receiver.go`

- 已为 generic inform 告警路径补充定义级别覆盖，并放宽 registry 注入条件，使已知告警的级别覆盖不再依赖 unknown fallback 的 productResolver。

### `omcgo/internal/alarm/expedited_receiver.go`

- 已为 ChangedAlarm 路径补充定义级别覆盖，避免实时事件更新时继续沿用设备原始级别。

### `omcgo/internal/alarm/sync_processor.go`

- 已在 CurrentAlarm 同步转换为 model 后应用定义级别覆盖，保证全量同步与实时上报口径一致。

### `omcgo/internal/alarm/engine.go`

- 重复告警 dedup 分支已统一做次数递增，并把 `first_raised_at` / `last_updated_at` 维持为业务时间语义，符合当前前端映射契约。

### `omcgo/cmd/app/provider/alarm.go`

- 已把 `AlarmSyncProcessor` 挂入容器并支持 app 侧后置注入，避免模块初始化顺序导致 sync 路径缺少 registry。

### `omcgo/cmd/app/provider/alarmdef.go`

- 告警定义模块初始化后会回填 sync processor registry，补上 app 进程中的 late binding 缺口。

### `omcgo/cmd/worker/main.go`

- worker 在 registry refresh 成功后会为 sync processor 注入 alarm definition registry，保持 worker 进程与 app 进程行为一致。

### `omcgo/internal/alarm/*_test.go`

- 新增回归测试覆盖三条级别覆盖路径，以及重复告警次数和更新时间语义，测试范围与本次 bugfix 对齐。

## 业务完整性检查

- Handler-Service-Repository 链路：通过，未引入空壳链路。
- 路由注册：未新增路由，无遗漏。
- 迁移文件配套：未涉及 schema 变更，无需 migration。
- 错误码注册：未新增业务错误码，无遗漏。
- API 服务 / Hook / Mock：纯后端内部修复，无前端 API 契约变更。
- E2E 与种子数据：未新增接口，无强制补充项。

## 业务影响范围检查

- 接口签名变更：仅内部 builder 风格方法补充 registry 注入，不影响既有外部接口调用。
- 共享 model 变更：未修改 `model.Alarm` 结构，仅修正字段写入语义。
- API 响应格式变更：无字段增删改，前端继续复用现有映射。

## 前后端一致性检查

- 当前前端 `alarmCount -> ack_count` 与 `updTime -> last_updated_at || updated_at || raised_at` 的既有映射保持不变，本次后端修复的是字段值语义而非契约格式。

## 代码质量回退检查

- 未发现删除测试、删除错误处理、绕过抽象或安全回退。

## 配套更新提醒

- 文档：本次为内部 hotfix，无强制文档更新项。
- 单元测试：已同步补充。
- 端到端测试：未新增接口，现阶段可不追加。

## 审查结论

PASS