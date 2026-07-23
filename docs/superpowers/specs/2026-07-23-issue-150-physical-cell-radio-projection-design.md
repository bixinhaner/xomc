# Issue 150 物理小区无线字段投影设计

## 背景

设备列表当前把 `device_parameters` 中命中的所有小区路径按值聚合：

- 不使用 `NumOfCells` / 产品 `SC|DC|CA` 载波模式限制物理小区范围；
- 同一小区的主路径和兜底路径会同时进入结果；
- 不同小区值相同时会按值去重，丢失小区位置。

因此 `FAP/MLN/DC` 设备可能把逻辑 `FAPService.3` 显示成第三个 DL EARFCN，
同时把两个相同的 UL EARFCN 压缩成一个。RF 状态已经在 Issue 157 中按物理载波
投影，本设计把相同边界用于 DL/UL EARFCN。

## 目标

- LTE/NR 的 DL/UL EARFCN 按物理小区索引投影。
- 有可靠物理小区数时只接受 `1..N`。
- 同一小区的候选路径按明确优先级选择一个值。
- 不同小区即使值相同也保留重复值，维持 CSV 的位置语义。
- 保持 `device_info.freq_point` / `device_info.ul_earfcn` 和现有 API CSV 契约。
- 不改变 GSM 频点、IPSec 等非 LTE/NR 小区字段的既有聚合。

## 方案

新增独立的无线频点投影函数，在通用多实例聚合之后覆盖 LTE/NR 的
`freq_point` 和 `ul_earfcn`。

物理小区数复用 Issue 157 的规则，并将函数从 RF 专用命名泛化：

1. 合法 `NumOfCells` 为权威值。
2. `/SC` 和 `/DC` 分别提供 1、2 的缺失兜底，并与上报值做一致性校验。
3. `/CA` 必须有合法 `NumOfCells`。
4. 数量未知时按实际观测到的小区索引排序投影，保持旧设备兼容性。

LTE 每个索引的路径优先级：

- DL：`RAN.RF.EARFCNDL`，然后 `RAN.Common.EARFCNDL`。
- UL：`RAN.RF.EARFCNUL`。

NR 每个索引的路径：

- DL：`CellConfig.{i}.NR.RAN.RF.NRARFCNDL`。
- UL：`CellConfig.{i}.NR.RAN.RF.NRARFCNUL`。

当物理小区数已知且某字段缺少任一小区值时，该字段投影为空并记录诊断原因，
不能压缩已有值造成位置错配，也不能未经设备契约确认就用 DL 伪造 UL。

## 数据流

`device_parameters`
→ 通用 carrier/universal mapping
→ 既有非频点多实例聚合
→ 物理小区频点投影
→ `device_info.freq_point/ul_earfcn`
→ 现有列表 API 和前端 CSV 展示。

## 测试

- DC + `NumOfCells=2` 忽略 `FAPService.3`。
- 两个物理小区 UL 相同仍输出两项。
- 同一小区同时存在 RF/Common DL 时只取 RF。
- 物理小区数已知但字段不完整时清空投影。
- NR 使用 `CellConfig.{i}` 作为物理索引。
- 数量未知时按观测索引兼容投影。
- 既有 RF、GSM、IPSec 聚合测试继续通过。

## 发布与验收

后端重新构建并部署 `app` 服务。部署后触发问题设备参数同步，使已有
`device_info` 快照重新投影；验证设备列表 DL/UL 数量、RF `[0/2]`、详情页小区
信息三者一致。
