# Code Review Report

| 项目 | 值 |
| --- | --- |
| 日期 | 2026-07-16 |
| 基准提交 | `origin/main` (`e1aa3b03c`) |
| 待合并提交 | `01236c9ed` + 当前工作区增量 |
| 作者 | wangyong |
| 范围 | device / provision / frontend device pages |
| 结论 | **PASS_WITH_WARNINGS** |

## 变更概要

修复设备名称同步策略在可靠参数同步与旧 Path B 链路中的生效闭环：补齐 OMC → LMT 的 SPV 任务发送器、让列表和详情页可处理待确认差异，并兼容 5G `gNBName` 路径及 `device_info.device_name` 为空的设备。

## 审查发现

### CRITICAL

无。

### WARNING

1. `getLMTDeviceName` 当前按固定顺序尝试 LTE `HNBName`、再尝试 NR `gNBName`，没有依据设备制式筛选。如果同一设备参数快照意外同时残留两种路径，NR 设备可能优先取到 LTE 名称。现网正常参数模型通常只落一种路径，因此本次不阻塞；建议后续把 `dev.Technology` 传入读取逻辑并优先选择制式对应路径。

### INFO

1. `scanDeviceInfoFromRow` 已兼容 nullable `device_name`，但当前没有针对该扫描函数的 NULL 回归测试；现有 device 包测试和全量构建已覆盖编译与主要调用链。
2. 名称同步仍沿用“第一个 `FAPService` 名称代表设备侧名称”的既有约定。多小区场景下应在产品层进一步明确设备级名称与小区级名称的边界。

## 关键链路检查

- 两套参数同步装配均注入 `DeviceNameTaskSender`，OMC → LMT 不再因 sender 缺失退化为 pending。
- LTE 下发继续使用 `HNBName`；NR 下发改用 `gNBName`，并新增对应单元测试。
- SPV 创建失败继续降级为 `name_sync_pending=true`，不阻塞参数同步主流程。
- 列表待确认红点支持选择 LMT、选择 OMC、忽略三种操作；成功后统一失效 `['devices']` 查询缓存。
- nullable `device_info.device_name` 使用临时指针扫描，避免 pgx v5 将 NULL 扫入 string 失败。
- 未新增 SQL 拼接、认证绕过、敏感日志或运营商硬编码。

## 验证

- `cd omcgo && go test ./internal/provision ./internal/device` — 通过。
- `cd omcgo && go build ./...` — 通过。
- `cd omcmb && npm run typecheck` — 通过。
- `git diff --check origin/main` — 通过。

## 总结

| 级别 | 数量 |
| --- | ---: |
| CRITICAL | 0 |
| WARNING | 1 |
| INFO | 2 |

**审查结论：PASS_WITH_WARNINGS，可以提交。**
