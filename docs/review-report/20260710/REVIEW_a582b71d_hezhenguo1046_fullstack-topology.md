# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-07-10 16:10 +0800 |
| 基线提交 | a582b71d |
| 作者 | hezhenguo1046 |
| 范围 | fullstack-topology |
| 变更文件数 | 27 |
| 新增行数 | +1009 |
| 删除行数 | -199 |

## 变更概要

本次修改为二级设备组自动归属规则加入显式 `source_group_id`，使规则只从指定源组迁移设备；写入阶段通过带源组条件的 SQL 再次校验，避免并发下误移动其他组的设备。

同时修复 TAC 等属性触发路径只使用旧事件值或失去当前归属的问题：匹配前从数据库读取当前 LAC/TAC 与设备归属，规则更新后对源组设备全量扫描，三套前端皮肤均可配置和回显源设备组、TAC/LAC、名称及序列号规则。

## 审查发现

### 🔴 CRITICAL（严重）

无。

### 🟡 WARNING（警告）

无。原发现的“只清空 `source_group_id`、保留 `matching_mode` 会让规则静默失效”已在 [omcgo/internal/topology/service.go](omcgo/internal/topology/service.go#L332) 修复，并由 `TestDeviceGroupService_UpdateGroup_RejectsClearingSourceForExistingRule` 覆盖。

### 🔵 INFO（建议）

1. [omcgo/internal/topology/group_match_engine_test.go](omcgo/internal/topology/group_match_engine_test.go#L61) 已验证引擎把源组 ID 传给 `ListForRuleEval` 并使用条件移动，但没有 PostgreSQL 集成测试证明 `ListForRuleEval` 不分页返回源组全部 TAC 命中设备、且 `MoveDeviceAutoMatched` 不会移动非源组设备。此次问题正是“部分 TAC 未迁移”，建议补充一个真实数据库用例，至少覆盖同 TAC 的多台源组设备、另一组同 TAC 设备、以及未分组设备。

2. [omcgo/internal/topology/group_match_engine.go](omcgo/internal/topology/group_match_engine.go#L67) 对大源组逐设备执行更新，当前能保证完整性，但一次规则变更会产生 $N$ 次 SQL 往返。当前改动不引入分页或 `LIMIT`，因此能修复本次漏迁移；当源组规模增长时，应评估在 repository 中按规则条件执行集合更新或可观测的分批处理，避免异步 goroutine 长时间占用连接。

## 详细分析

### 源设备组范围

- [omcgo/internal/topology/model.go](omcgo/internal/topology/model.go#L90) 到 [omcgo/internal/topology/pg_repository.go](omcgo/internal/topology/pg_repository.go#L146) 已完整贯通模型、请求 DTO、查询扫描、创建与更新持久化。
- [omcgo/internal/topology/service.go](omcgo/internal/topology/service.go#L114) 限制来源和目标均为 L2、禁止自引用，并在创建时要求带规则的组必须指定来源。
- [omcgo/internal/topology/pg_device_lister.go](omcgo/internal/topology/pg_device_lister.go#L118) 查询限定 `dgm.group_id = sourceGroupID`；来源为“未分组设备”时改为 `dgm.device_id IS NULL`，没有分页或 `LIMIT`。
- [omcgo/internal/topology/pg_repository.go](omcgo/internal/topology/pg_repository.go#L445) 的更新条件同时检查 `membership.device_id` 和 `membership.group_id = sourceGroupID`。因此即使扫描后设备已转移，也只会得到 `RowsAffected = 0`，不会覆盖其他组归属。这是正确的并发保护。

结论：第一项修复落在查询范围和原子更新两个必要位置，逻辑合理；不再存在“所有设备组的同条件设备都被搬走”的路径。

### TAC 全量归属

- [omcgo/internal/topology/group_match_engine.go](omcgo/internal/topology/group_match_engine.go#L67) 对来源查询得到的全部设备逐台执行匹配；没有页大小、偏移或首批截断逻辑。
- [omcgo/internal/topology/matcher.go](omcgo/internal/topology/matcher.go#L117) 的 TAC 模式使用数据库最新值 `DeviceForMatch.TAC`，并在事件/心跳路径通过 [omcgo/internal/topology/pg_device_lister.go](omcgo/internal/topology/pg_device_lister.go#L159) 重新读取真实当前归属与 TAC，避免仅依赖注册事件中缺失或过期的字段。
- [omcgo/internal/topology/matcher.go](omcgo/internal/topology/matcher.go#L109) 与 [omcgo/internal/topology/pg_repository.go](omcgo/internal/topology/pg_repository.go#L445) 共同保证设备仅从指定来源移动。

结论：针对截图中 TAC=1 的多设备场景，修改后的扫描不再受到列表页分页、旧事件载荷或全局重分组的影响；只要设备的 `device_info.tac` 是可解析整数、当前归属仍在所选源组，并满足规则，都会被处理。

### 前后端契约

- 共享 API/Hook/类型已声明 `source_group_id`、`serial_number_list` 与 `serialNumber` 模式。
- v1 在 [omcmb/webcode/src/pages/device/DeviceGrouping/GroupDialogs.tsx](omcmb/webcode/src/pages/device/DeviceGrouping/GroupDialogs.tsx#L106) 仅提供 L2 组作为来源并排除当前目标；v2/v3 也提供相同来源选择和保存约束。
- 三皮肤均在保存时传递源组、匹配模式和当前规则字段；编辑时可回显已有规则。

### 数据库迁移

[omcgo/migrations/000015_add_device_group_rule_source.sql](omcgo/migrations/000015_add_device_group_rule_source.sql) 采用可空外键、删除来源后自动置空、自引用检查和部分索引，符合“存量规则先暂停，人工重新指定来源”的迁移策略。干净库验证确认 `source_group_id`、`device_groups_source_group_id_fkey` 与 `device_groups_rule_source_not_self` 均已创建。

## 业务完整性检查

- Handler → Service → Repository：通过；无 handler 绕过服务层直接操作数据库。
- 路由：未新增端点，沿用现有设备组创建/更新接口。
- 接口签名：`GroupMembership` 的替换已同步到 mock、测试和所有调用点。
- SQL：新增持久化使用 Squirrel；必要的原子迁移 SQL 使用参数绑定，无字符串拼接输入。
- 权限：未改变既有路由与认证边界。
- 三皮肤：v1/v2/v3 均具备来源选择与规则编辑能力，`skin-parity` 通过。

## 验证记录

- `cd omcgo && go build ./... && go test ./internal/topology/...`：通过。
- `cd omcmb && npm run typecheck`：通过；包含 `skin-parity`，v2/v3 均与 v1 对齐（129 条路由、37 个可见菜单项）。
- `cd omcgo && bash scripts/check-migrations.sh --strict`：通过。输出的 seed 与 schema 历史同号、以及 `000008` 空 Down，均为已有告警，非本次新增。
- `docker compose -f deployments/docker/docker-compose.yml down -v` 后重新构建并执行全部 schema、seed、tsdb 迁移：通过；schema 已执行至 `000015_add_device_group_rule_source.sql`。

## 审查结论

**通过。** 核心的源组范围约束和 TAC 全量处理设计正确；更新接口已拒绝“清空来源但仍保留匹配模式”的无效规则状态，且已通过后端编译、拓扑测试、三皮肤类型检查与干净数据库迁移验证。
