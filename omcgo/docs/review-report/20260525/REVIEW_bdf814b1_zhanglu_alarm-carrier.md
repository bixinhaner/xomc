# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-25 09:36 |
| 提交 | bdf814b1 |
| 作者 | zhanglu |
| 范围 | alarm-carrier |
| 变更文件数 | 11 |
| 新增行数 | +42 |
| 删除行数 | -8 |

## 变更概要

本次变更修复了告警入库前的 severity 覆盖逻辑：当运营商适配器没有命中显式告警码映射时，不再默认把源告警级别改写为 warning，而是保留设备上报的原始级别。
同时补充了 alarm engine 与 sync processor 的回归测试，并同步更新三家运营商适配器及其测试契约。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

- 本次修复只影响重启后新进入或重新同步的告警；数据库中已被写成 warning 的历史活动告警不会被自动回填，需要后续通过重同步或单独数据修正处理。

## 详细分析

### omcgo/internal/alarm/engine.go

- 调整 severity 覆盖规则，仅在 Carrier 返回显式映射时覆盖；若源 severity 和映射都缺失，仍兜底为 warning，行为边界清晰。
- 未引入额外外部依赖，也未破坏现有 AlarmEngine 接口。

### omcgo/internal/core/carrier/cmcc/adapter.go

- 将未知告警码的默认返回值从 warning 改为 0，契合“无显式映射”语义。

### omcgo/internal/core/carrier/ctcc/adapter.go

- 与 CMCC 保持一致，未知码返回 0，避免适配器层错误吞掉设备原始 severity。

### omcgo/internal/core/carrier/cucc/adapter.go

- 与其他运营商适配器契约保持一致，未知码返回 0。

### omcgo/internal/alarm/engine_test.go

- 新增回归用例覆盖 70011 未命中运营商映射时保留 Major 的场景，能够直接防止问题回归。

### omcgo/internal/alarm/sync_processor_test.go

- 补充同步链路断言，验证告警同步新增记录不会再被错误覆盖为 warning。

### omcgo/internal/core/carrier/*_test.go 与 carrier_test.go

- 测试契约同步更新为“未知码 = 无显式映射”，与 engine 的新行为一致。

## 业务完整性检查

- Handler-Service-Repository 链路：本次未新增链路，原有链路不受影响。
- 路由注册：未改动路由。
- 数据库迁移：无 schema 变更，无需 migration。
- 错误码注册：未新增错误码。
- 测试覆盖：已补充后端单测，覆盖 engine 和 sync 两条关键路径。

## 业务影响范围检查

- 影响范围集中在告警入库前的 severity 计算，不影响接口签名。
- 与三家运营商适配器的未知码契约同步收敛，避免不同 carrier 出现不一致行为。
- 不涉及前端契约变更；前端仍按既有 1/2/3/4 severity 数值展示。

## 配套更新提醒

- 文档：如后续需要对外说明“未知运营商告警码不再默认 warning”，建议补充到告警模块设计或运行手册。
- 单元测试：本次已同步补充。
- 端到端测试：若后续有稳定的告警注入方式，建议补一条覆盖未知码保留源 severity 的 E2E 用例。

## 审查结论

PASS_WITH_WARNINGS

本次修改未发现阻塞提交的问题，变更范围小、验证充分，可以提交。
