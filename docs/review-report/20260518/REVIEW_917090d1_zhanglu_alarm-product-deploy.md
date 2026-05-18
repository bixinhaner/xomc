# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-18 09:34 |
| 提交 | 917090d1 |
| 作者 | zhanglu |
| 范围 | alarm-product-deploy |
| 变更文件数 | 10 |
| 新增行数 | +382 |
| 删除行数 | -108 |

## 变更概要

本次变更修复了 BM 产品未知告警开关在 worker 侧不稳定生效的问题，覆盖普通告警和 expedited 告警两条路径。修复同时补上了 worker fallback wiring、product registry 跨进程刷新、Redis 二级缓存版本失效，以及对应的回归测试。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

1. 建议后续补一条覆盖“产品中心切换 enable_unknown_alarm 后 worker 无重启即时生效”的集成或 E2E 用例，避免同类跨进程缓存回归仅靠手工验证发现。

## 详细分析

### `omcgo/cmd/worker/main.go`

修复了 fallback 注入链路，统一对普通告警和 expedited 告警 receiver 注入 alarm definition registry 与 product resolver；同时在 worker 可用 Redis 时改用 RedisCache，消除跨进程配置变更无法感知的问题。未发现新的启动顺序或空指针风险。

### `omcgo/cmd/worker/main_test.go`

新增 wiring 回归测试，确保两个 receiver 都拿到 fallback 依赖，能覆盖此前局部变量遮蔽导致 expedited 路径未启用 fallback 的缺陷。

### `omcgo/internal/alarm/receiver.go`

将 unknown alarm fallback 提炼为复用函数，普通告警路径保留既有降级行为，同时把 drop/keep 的日志与 metrics 保持在同一入口，逻辑更一致。

### `omcgo/internal/alarm/expedited_receiver.go`

expedited NewAlarm 路径现在与普通告警共享同一套 unknown fallback 判定，能正确处理 `enable_unknown_alarm` 开关。异常分支采用 warn 后按旧路径继续，不会放大单点失败影响。

### `omcgo/internal/alarm/expedited_receiver_test.go`

新增 expedited unknown fallback 回归测试，验证未知告警在允许场景下会落为 `is_unknown=true` 且 severity 为 31004，测试覆盖到这次缺陷的核心行为。

### `omcgo/internal/alarm/pg_store.go`

补全 `alarms_active.status` 的表前缀，避免查询中 `status` 字段歧义。该修改风险低，且与当前告警清理/重报验证结果一致。

### `omcgo/internal/product/registry.go`

增加基于 cache version 的 `ensureFresh` 机制，在 `MatchProductClass` 与 `GetProductByID` 前进行被动刷新，解决 app 更新产品配置后 worker 继续使用旧快照的问题。版本变更时只被动 refresh 不再次 bump，避免实例间版本抖动。

### `omcgo/internal/product/cache.go`

Redis product 缓存从裸 Product JSON 升级为带 version 的包装结构，确保版本变化后旧缓存条目自动 miss，不再把过期的 `EnableUnknownAlarm` 返回给 registry。

### `omcgo/internal/product/cache_test.go`

新增 RedisCache 版本 bump 后缓存失效测试，验证二级缓存不会继续返回旧 Product。

### `omcgo/internal/product/registry_test.go`

增强 spy cache 的版本语义，并新增 version change 触发 refresh 的测试，覆盖此次跨进程刷新根因。

## 业务完整性检查

- Handler-Service-Repository 链路：本次未新增 REST handler，主要修复 worker 与 registry 行为，链路完整。
- 路由注册：无新增路由。
- 迁移文件配套：无 schema 变更，不需要 migration。
- 错误码注册：无新增业务错误码。
- 测试覆盖：已新增 worker wiring、expedited fallback、registry version refresh、Redis cache version miss 四类回归测试。

## 业务影响范围检查

- 接口签名变更：未对外暴露新的公共 API，仅在内部 receiver/registry 行为上增强。
- 事件契约变更：未修改 EventBus 消息结构，现有发布订阅方不受影响。
- 配置项变更：无新增配置项；仅在 worker 有 Redis 时启用既有 Redis 缓存能力。
- 跨模块引用：alarm 与 product 的依赖保持在 adapter/registry 抽象层，没有引入循环依赖。

## 前后端一致性检查

本次变更仅涉及后端 worker 与告警/产品内部逻辑，不涉及前端接口契约变更。

## 代码质量回退检查

未发现删除测试、绕过错误处理、降级安全措施、引入 `any/interface{}`、硬编码替代配置等质量回退问题。

## 配套更新提醒

- 文档更新：本次为内部 bugfix，可不强制更新接口文档。
- 单元测试更新：已完成。
- 端到端测试更新：建议后续补一条自动化用例，覆盖产品中心切换未知告警开关后的跨进程生效路径。

## 审查结论

PASS_WITH_WARNINGS

本次变更未发现阻塞提交的问题，当前实现和运行时复验结果一致，可提交。