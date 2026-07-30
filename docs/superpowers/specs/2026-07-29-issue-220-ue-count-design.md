# Issue #220 UE Count 同步修复设计

## 问题

设备已经接入 UE，但 OMC 的 `device_info.ue_count` 一直为 0。当前系统没有在
Periodic Inform 会话中主动查询 UE Count；同时 `CalcUECount` 只读取第一小区的
`Device.DeviceInfo.UE_Count`，没有汇总
`Device.DeviceInfo.2.UE_Count`、`Device.DeviceInfo.3.UE_Count` 等物理小区。

## 方案

1. 新增 `UECountPolicy`，仅在设备的 Periodic Inform 中触发。
2. 通过 ACS 已有 `PathTranslationService` 解析该设备当前产品/软件版本的
   MappingSet，只选择 active、supported 的 UE Count 参数。
3. 以标准路径创建普通系统 GPV 任务；下发时继续由已有翻译器转换为厂商私有路径，
   回包时沿用已有响应回译、`device_parameters` 入库和 `device_info` 刷新链路。
4. 创建任务前查询同设备、同方法、同描述的 pending/sent 任务；已有未完成查询时
   不重复入队，避免设备离线或处理缓慢时堆积。
   多 ACS 实例先通过 Redis `SET NX` 短租约原子准入，避免“先查再建”的并发竞态；
   整个探测链路使用 500ms 超时，查重 SQL 使用 active task 部分索引。
5. `CalcUECount` 按物理小区汇总标准路径：根路径代表小区 1，数字实例路径代表对应
   小区；同一小区去重，根路径优先于 `.1`。只有不存在任何有效标准值时才回退厂商
   扩展路径。

## 边界

- 映射不可解析、产品不匹配或没有支持的 UE Count 路径时，跳过本次查询，不阻塞
  InformResponse。
- 无效、空或负数 UE Count 不参与聚合。
- 标准路径存在合法的 0 时视为有效数据，不使用厂商扩展值覆盖。
- 不启用 durable partial readback；当前 completion projector 只投影 full scope，
  普通 GPV 才能立即复用现有 `device_info` 刷新链路。

## 验证

- 单元测试覆盖多小区汇总、重复小区去重、标准 0 与厂商回退。
- 单元测试覆盖 MappingSet 路径筛选、Periodic 触发、任务参数及未完成任务去重。
- ACS 与 device 包测试、后端全量 build/test。
