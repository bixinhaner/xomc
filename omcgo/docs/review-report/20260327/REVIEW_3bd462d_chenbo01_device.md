# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-27 |
| 基准提交 | 3bd462d |
| 作者 | chenbo01 |
| Scope | device |
| 结论 | **PASS_WITH_WARNINGS** |

---

## 变更概述

修复 `HeartbeatMonitor.CheckHeartbeats` 在遍历 active 设备时因 OFFSET 分页"滑动窗口"导致部分设备被跳过无法检查的 bug。改用 keyset（游标）分页，按 `last_inform_at ASC` 排序，优先检查最久未上报的设备。

### 变更文件（14 个）

| 文件 | 改动类型 | 说明 |
|------|---------|------|
| `internal/device/repository.go` | 接口扩展 | `DeviceReader` 新增 `ListActiveByLastInform` 方法 |
| `internal/device/pg_repository.go` | 新增实现 | keyset 分页查询，`(last_inform_at, id)` 复合游标 |
| `internal/device/heartbeat.go` | 重写 | `CheckHeartbeats` 从 OFFSET 分页改为游标分页 |
| `internal/device/heartbeat_test.go` | 测试更新 | mock 新增方法 + 测试适配新 API |
| 10 个 `*_test.go` | mock 补充 | 各模块 mock 实现补充 `ListActiveByLastInform` 存根 |

---

## 审查发现

### WARNING

1. **NULL `last_inform_at` 游标跳过**
   - 位置: `pg_repository.go:315`, `heartbeat.go:122`
   - 描述: 若 `last_inform_at` 为 NULL，`cursorTime` 为 nil，下一轮 `cursorTime != nil && cursorID != nil` 条件不满足，游标条件被跳过
   - 风险: 理论上可能导致重复查询同一批设备
   - 缓解: (1) Active 设备必然经过 Inform 流程，`last_inform_at` 不应为 NULL；(2) 30s context timeout 兜底；(3) `NULLS FIRST` 排序确保 NULL 行在首批被处理
   - 建议: 未来可考虑在 NULL 游标场景下仅用 `id > cursorID` 作为备用游标条件

### INFO

2. **代码清理**: 移除未使用的 `statusPtr` 辅助函数
3. **可观测性提升**: 心跳检查完成日志新增 `devices_checked` 字段

---

## 检查项

- [x] 错误处理: `fmt.Errorf` 包装上下文
- [x] SQL 安全: Squirrel 参数化查询，无字符串拼接
- [x] 资源释放: `defer rows.Close()`
- [x] 接口优先: 新增方法定义在 `DeviceReader` 接口
- [x] 命名规范: PascalCase 导出方法
- [x] 测试覆盖: 心跳测试已适配新 API，覆盖成功/失败/错误路径
- [x] 无运营商硬编码
- [x] 无循环依赖
