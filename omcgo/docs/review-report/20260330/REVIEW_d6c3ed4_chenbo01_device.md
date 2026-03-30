# Code Review: feat(device): G28-G36 device_info 扩展与快查列计算

**Date**: 2026-03-30
**Author**: chenbo01
**Reviewer**: Claude (automated)
**Scope**: device
**Verdict**: PASS_WITH_WARNINGS

---

## 变更概要

实现 0023 设计文档 §7 的 G28-G36 差距项：
- G28: device_info 新增 4 列（num_of_cells, gps_status, alarm_severity, license_status）
- G29: Inform 通用快查列同步机制
- G30-G32, G36: mme_status/license_status/cell_status/sync_status 计算逻辑
- G33: ACS 查询频率分级策略定义
- G34-G35: 设备详情页 DTO 组装（MME/License/Antenna/多小区）

**16 文件变更**：8 新建 + 8 修改，+1794 / -10 行

---

## 审查发现

### CRITICAL

无

### WARNING（已修复）

| # | 问题 | 修复 |
|---|------|------|
| W3 | cell_status 枚举不一致：CalcCellStatus 返回 "inactive" 但 GetEnums 无此选项 | 已更新枚举列表，对齐计算函数返回值 |
| W4 | mme_status 枚举不一致：CalcMMEStatus 返回 "connected"/"partial" 但 GetEnums 用 "normal"/"error" | 已更新枚举列表为 connected/partial/disconnected |
| W5 | universalInformMapping 中 "gps_status_raw" 是假列名，依赖 delete() 清除 | 已移除该映射项，GPS 状态完全由 CalcGPSStatus 计算 |

### WARNING（遗留，低风险）

| # | 问题 | 说明 |
|---|------|------|
| W1 | ListDevicesWithInfo 的 count 查询 ToSql/Scan 错误被忽略 | 已有模式（pg_repository.go 同模块其他方法亦如此），后续统一修复 |
| W2 | UpdateManualFields/UpdateSyncFields 未显式设置 updated_at | device_info 表已有 trigger 自动更新 updated_at（复用 update_updated_at_column 触发器） |

### INFO

| # | 说明 |
|---|------|
| I1 | filterByPrefix 使用 containsAny 而非 strings.Contains，可读性可改善 |
| I2 | AssembleLicenseDetail 用 strings.Contains 匹配短子串，精度可提高 |
| I3 | GetDeviceDetailComposite 全量加载参数后内存过滤，当前规模可接受 |
| I4 | UpdateManualFields 的 not-found 错误未使用 commonerrors.ErrNotFound |
| I5 | AssembleMMEPool 输出未按 Index 排序，可能影响前端展示稳定性 |

---

## 检查项

- [x] Go 命名规范（PascalCase/camelCase）
- [x] 错误包装（fmt.Errorf + %w）
- [x] SQL 安全（Squirrel 参数化，无字符串拼接）
- [x] 无运营商硬编码（通过 Carrier 接口适配）
- [x] 接口优先设计（DeviceInfoRepository）
- [x] 测试覆盖（info_calc_test: 7 函数 35+ 用例，detail_assembler_test: 9 函数）
- [x] 编译通过（go build ./...）
- [x] 测试通过（go test ./internal/device/...）
- [x] go vet 无警告
