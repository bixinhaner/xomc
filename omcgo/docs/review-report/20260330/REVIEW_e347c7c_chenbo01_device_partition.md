# Code Review Report

| 项 | 值 |
|----|------|
| 日期 | 2026-03-30 |
| 基准 | e347c7c |
| 作者 | chenbo01 |
| Scope | device (partition optimization) |
| 文件数 | 16 (4 新建 + 10 修改 + 2 迁移) |
| 变更量 | +1926 / -39 |

## 审查结论: PASS_WITH_WARNINGS

---

## CRITICAL: 无

## WARNING

### W1: GetDeviceDetailComposite 增加了 DB 调用次数

**文件**: `internal/device/service.go:682-710`
**描述**: 原来 1 次 `GetByDevice` + 内存过滤，现在 3 次 `GetByGroup` + 1 次 `GetByDevice` = 4 次 DB 调用。
**影响**: 设备详情页延迟可能从 ~1ms 增加到 ~4ms（4 次索引扫描）。
**评估**: 可接受。详情页非热路径（非列表页），4ms 延迟在用户感知阈值内。后续可通过 pipeline 或并发查询优化。
**状态**: 已知，暂不修复

### W2: Cells 仍使用全量查询

**文件**: `internal/device/service.go:704-710`
**描述**: `AssembleCells` 仍需全量参数（跨多个 group），因此保留了 `GetByDevice` 调用。
**评估**: 正确决策。Cells 需要 fap_control + radio + license 等多个 group 的参数来组装，按 group 多次查询反而更慢。后续可考虑 `GetByFAPInstance` 替代。
**状态**: 已知，可后续优化

## INFO

### I1: scanParam helper 使用接口类型

**文件**: `internal/device/pg_param_repository.go:24-32`
**描述**: `scanParam(s interface{ Scan(dest ...any) error })` 同时适用于 `pgx.Row` 和 `pgx.Rows`，消除了 7 处重复的 Scan 代码。
**评估**: 良好的 DRY 实践。

### I2: paramColumns 统一列名管理

**文件**: `internal/device/pg_param_repository.go:16-20`
**描述**: 包级变量 `paramColumns` 统一管理 SELECT 列名，所有查询方法引用同一份列表。新增列只需改一处。
**评估**: 良好。

### I3: ON CONFLICT 不更新分类列

**文件**: `internal/device/pg_param_repository.go:64-68`
**描述**: `fap_instance` 和 `param_group` 由 `parameter_path` 唯一确定，ON CONFLICT 只更新 value/type/writable/last_updated_at，不更新分类列。
**评估**: 正确。减少不必要的写入。

### I4: 分类函数优先级设计

**文件**: `internal/device/param_classify.go:33-76`
**描述**: 12 个分组按优先级排列，高优先级关键字（如 MmePoolConfigParam）先匹配，避免被低优先级规则（如 FAPControl）吞噬。测试覆盖了 5 个优先级冲突场景。
**评估**: 良好。

### I5: Hash 分区迁移策略

**文件**: `migrations/000064_partition_device_parameters.up.sql`
**描述**: RENAME → CREATE partitioned → INSERT...SELECT 数据迁移 → DROP old → ANALYZE。CASE 表达式与 Go 分类逻辑一致。
**评估**: 标准做法。需注意迁移期间需要停机窗口（排他锁）。

### I6: 测试覆盖

**文件**: `internal/device/param_classify_test.go`
**描述**: 43 个测试用例覆盖全部 12 个分组 + fap_instance 提取 + 5 个优先级冲突。
**评估**: 充分。

## 检查清单

- [x] `go build ./...` 通过
- [x] `go vet ./...` 通过
- [x] `go test ./internal/device/...` 通过
- [x] 相关模块测试通过（interop, nedirect, northbound/sync）
- [x] 无 SQL 字��串拼��（全部 Squirrel 参数化）
- [x] 无运营商硬编码
- [x] 错误包装带上下文
- [x] 资源释放（defer rows.Close()）
- [x] 迁移 up/down 配对
