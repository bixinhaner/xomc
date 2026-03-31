# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-31 |
| 基准 | 2dc3a84 (main) |
| 作者 | chenbo01 |
| Scope | device / admin / topology / software |
| 变更统计 | 52 files, +4712 / -2248 |
| 审查结论 | **PASS_WITH_WARNINGS** |

## 变更概述

实现设计文档 `0023-device-management-system-design.md` 中的全部功能模块：

1. **设备分组树重构** — 两级 L1/L2 分组，含默认分组、批量操作、设备迁移
2. **数据权限 (RBAC)** — 角色→设备组关联，PermissionService 带 Redis 缓存
3. **设备预注册** — SN 预注册、Inform 自动分配分组
4. **列配置** — 每用户 JSONB 列布局
5. **CSV 导出** — 流式导出，UTF-8 BOM
6. **设备操作扩展** — 参数同步、射频开关
7. **升级任务生命周期** — 挂起/恢复/终止/回滚，新增 suspended/terminated 状态
8. **软删除** — devices 表 deleted_at 字段，所有读查询过滤已删除记录
9. **异常重启检测** — Inform 事件分析，发布 device.reboot.abnormal 事件

## CRITICAL（已修复）

### C1. RF 参数路径硬编码 → 已添加 TODO 注释
`internal/device/service.go` `SetRFSwitch()` 中 `Device.Services.FAPService.1.FAPControl.LTE.AdminState` 仅适用于 LTE。已添加 `TODO(carrier)` 注释标记，待 Carrier 适配器层完善后重构。

### C2. VisibleGroups 空切片语义缺失 → 已修复
`device_info_pg_repository.go` 中 `nil` 与 `[]uuid.UUID{}` 语义已明确区分：
- `nil` = 超管，不过滤
- `[]uuid.UUID{}` = 无权限，`WHERE FALSE` 返回空集
- `[id1, id2, ...]` = 按组过滤

## WARNING

### W1. ErrorMessage 字段语义复用
`SuspendUpgrade` 将前驱状态存入 `error_message` 列，`ResumeUpgrade` 从中读取。逻辑正确但隐式约定缺少注释。建议后续迭代增加 `previous_status` 专用字段。

### W2. DeleteGroup 重复调用 ListChildIDs
`topology/service.go` `DeleteGroup()` 中两次调用 `ListChildIDs`，第二次可复用第一次结果。性能影响较小但应优化。

### W3. InvalidateRoleCache 依赖 TTL 自然过期
角色权限变更后最长 5 分钟延迟生效。生产环境需实现主动失效（Redis SCAN 或反向索引）。

### W4. detectAbnormalReboot 仅在 Bootstrap 触发
异常重启检测应同时覆盖 `handlePeriodic`。当前仅在 `handleBootstrap` 中调用，覆盖不完整。

### W5. 导出硬性限制 10,000 行
大型网络导出不完整无提示。建议改为分页流式读取。

### W6. Redis key 命名 `perm:` 前缀
建议对齐 CLAUDE.md 规范改为 `user:visible_groups:{userID}`。

### W7. getOperator 函数重复定义
`registration_handler.go` 和 `topology/handler.go` 各有一份，建议提取公共位置。

## INFO

### I1. regRepo.UpdateStatus 错误被静默忽略
设备上线后预注册状态更新失败静默丢弃，建议添加 Warn 日志。

### I4. deleted_at 迁移 → 已添加
新增 `000069_add_device_deleted_at.up/down.sql`。

### I5. 新功能缺少单元测试
RegistrationService、PermissionService、SuspendUpgrade 等新逻辑未附带测试。建议后续补充。

## 构建验证

- `go build ./...` ✅
- `go vet ./...` ✅
- `go test ./...` ✅（除 provision 4 个预已存在的失败）
