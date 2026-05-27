# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-27 09:37 |
| 提交 | edd70a7f |
| 作者 | zhanglu |
| 范围 | fullstack-device |
| 变更文件数 | 3 |
| 新增行数 | +244 |
| 删除行数 | -33 |

## 变更概要

本次变更聚焦设备详情页可读性和真实设备数据呈现。前端将小区信息改为摘要表格、按用户要求删减基站/状态/其他信息字段，并新增对详情复合接口的拉取；后端补充了对 FAPService 实例号的参数路径探测，修复 num_of_cells 滞后时只展示 1 条小区的问题。

## 审查发现

### 🔴 CRITICAL (严重)

> 必须在提交前修复的问题

无

### 🟡 WARNING (警告)

> 建议修复，不阻塞提交

无

### 🔵 INFO (建议)

> 改进建议，可选择性采纳

1. omcmb/webcode/src/pages/device/DeviceDetail/index.tsx 中 cellId 列当前使用 cellId -> eci 的回退链路；如果后端后续补齐真实 per-cell cellId 字段，建议直接切到后端真值，避免“Cell ID”标签长期承载 ECI 语义。

## 详细分析

### omcgo/internal/device/detail_assembler.go

- [L162] 在 AssembleCells 中引入 detectMaxFAPServiceIndex 兜底，当设备表 num_of_cells 落后于 device_parameters 实际上报时，能按参数路径扩展为真实 FAPService 实例数。逻辑局部、边界明确，未引入额外外部依赖。
- [L244] detectMaxFAPServiceIndex 仅扫描与 CellConfig/FAPControl 相关的小区路径，避免把无关 FAPService 参数误计入小区数，约束合理。

### omcgo/internal/device/detail_assembler_test.go

- [L192] 新增回归测试覆盖 num_of_cells=1 但参数里存在 FAPService.6 的场景，能有效锁住“详情页只显示 1 条小区”的真实缺陷。

### omcmb/webcode/src/pages/device/DeviceDetail/index.tsx

- [L458] buildCellRecords 优先消费详情接口返回的 cells，并保留旧来源回退，兼容真实接口与历史页面数据结构。
- [L514] getCellSummaryColumns 将小区区域收敛为摘要列，满足当前交互目标，避免继续渲染制式不匹配的折叠字段。
- [L650] 新增 detail-composite-v2 查询，并在 [L710] 的头部刷新中同步失效该查询，保证设备详情和小区表格刷新语义一致。
- [L929] 小区列渲染增加设备级 opState/rfStatus 回退；本次还补齐了 rfStatus 的 useMemo 依赖，避免回退值变化后列渲染使用旧状态。

## 业务完整性检查

业务链路完整，无遗漏。前端新增详情复合查询有对应后端接口消费；后端小区组装逻辑补充了回归测试；本次未新增 handler、路由、迁移或错误码，因此不存在链路缺口。

## 业务影响范围检查

变更范围可控，未发现跨模块影响。后端仅调整 device detail assembler 的组装上界，不涉及 schema、事件契约或共享 model 破坏；前端变更集中在设备详情页内部展示与同页查询刷新。

## 前后端一致性检查

本次变更同时涉及前后端，接口路径和消费方式保持一致。前端调用 /devices/{id}/detail 获取 cells，后端补强的是该接口底层的 cells 组装数量；未发现请求路径、响应字段命名或枚举值不一致问题。

## 代码质量回退检查

未发现代码质量回退。

## 配套更新提醒

- **文档**: 无需更新。此次变更是已有设备详情能力的展示和组装修正，未改变公共 API 契约。
- **单元测试**: 已有测试覆盖，新增了 AssembleCells 的多实例探测回归测试。
- **端到端测试**: 建议后续补一条设备详情页或 detail 接口的多小区场景 E2E，用于锁住真实设备 6 小区展示回归；本次提交不阻塞。

## 安全检查

未发现安全问题。

## 性能检查

未发现性能问题。detectMaxFAPServiceIndex 为单次线性扫描，且仍在已有 paramMap 构建前的同一批参数内处理，成本可接受。

## 测试覆盖

- `cd omcgo && go test ./internal/device/...` 通过
- `cd omcmb/webcode && npm run typecheck` 通过
- 本次未执行浏览器自动化或 E2E 脚本

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 1 |

**审查结论**: `PASS`
