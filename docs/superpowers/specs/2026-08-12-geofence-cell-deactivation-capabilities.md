# 电子围栏小区去激活产品能力矩阵

> Issue #304 修订，2026-08-12。本文是电子围栏 `enforce` 的产品启用门禁，不能用 RF/IPSec
> 回读替代小区运行终态。路径实例必须来自当前设备快照和 ParamModel，禁止按产品名猜路径。

## 统一能力契约

每个允许 `enforce` 的产品必须同时解析到：

1. 可写 RF 控制，或经模型和真机证明同时承担 RF/小区管理语义的组合控制；
2. 可写小区管理控制；
3. 只读 `OpState` 或等价运行状态；
4. 可写 IPSec 控制；
5. 控制前实际值、目标值和恢复值的布尔语义。

缺任一角色、Access 冲突、实例不完整或映射歧义时 fail closed，不创建南向控制任务。
`OpState` 即使在个别 XML 中被误标为可写，也只允许作为只读后置条件。

产品可使用两种控制形态：

- `Admin + RF`：两个独立控制，越界按 Admin → RF → IPSec，下发后 GPV，再轮询 OpState；
- `AdminRF`：一个经 ParamModel 和真机证明的组合控制，例如私有 `AdminCellState`。它同时承担
  管理去激活和 RF 关闭，但仍必须独立读取 OpState。

恢复顺序统一为 IPSec → RF → Admin/AdminRF，最后轮询原始运行终态。只恢复本围栏实际修改且
GPV 已验证的控制。

目标小区实例按以下顺序解析：

1. 当前快照存在合法 `FAPControl.LTE.InUse` 时，只控制 `InUse=true` 的实例；
2. 否则使用合法 `NumOfCells` 对应的连续 `1..N`；
3. 两者缺失、非法或与当前运行快照冲突时，按业务连续性要求回退到产品装配 `radioModes` 与
   ParamModel `NumOfCells.enumValues` 共同声明的设备最大能力。例如 MLN/0041 的最大能力是三小区，
   当前 SC 模式不改变该产品能力上限；BAIBLQ/SC 与共享 BLQ 模型的 QRTB（436Q）则分别收敛为
   最大 1 和 2；249 属于 MLQ，不与 BAIBLQ 混称。
   最大集合中的每个实例仍须具备可写控制和只读终态快照证据；缺项时明确失败，不能静默缩成较小
   集合，也不允许合成无法回读或无法恢复的路径。

回区恢复使用“原动作实际修改且验证成功的实例 ∩ 当前有效实例”；当前有效实例仍无法确认时，
最多恢复原动作拥有的实例，不能扩展到后来新出现的实例。`OpState` 只判断运行终态，不用于判断
小区是否配置有效。

## 产品矩阵

| 产品 / ParamModel | RF / 私有映射 | 管理控制 | 只读终态 | IPSec | 联动证据与启用结论 |
|---|---|---|---|---|---|
| BLQ / BLQ | `RFTxStatus` → `X_COM_RadioEnable`，1/0 | 运行模型补齐 `Device.DeviceInfo.FAP_adminstate` ↔ `FAPService.1.FAPControl.LTE.AdminState` 双向映射，按独立 Admin + RF 控制 | `FAPControl.LTE.OpState` RO | 全局及 tunnel enable | 251 已实证 RF=false、OpState=true，不联动；代码门禁已补齐，须以 104 的 0005 完整 SPV/GPV/OpState/恢复验收签署 |
| BLX、QRTB、mBS31001 / BLQ | `RFTxStatus` → `X_COM_RadioEnable` | 私有 `FAP_adminstate` 为设备级无实例路径，尚无多小区覆盖语义证据 | `FAPControl.LTE.OpState` RO | 全局及 tunnel enable | 不把 BLQ/SC 的具体实例映射外推到多小区产品；管理覆盖不完整时 fail closed |
| MLQ / MLQ | `RFTxStatus` → `X_COM_RadioEnable` | 交付模型缺失 | `FAPControl.LTE.OpState` RO | 全局、tunnel、MultiIpsec | 未取得 Admin 真机证据，fail closed |
| BLN / BLN | `RFTxStatus` → `AdminCellState` | 模型另有 `AdminState`；因 452 证据属于 MLN，BLN 不继承 `AdminRF` 联动语义，按独立 Admin + RF 控制 | `FAPControl.LTE.OpState` RO | 全局及 tunnel | 静态能力完整；必须以 OpState 轮询判断终态，不假设 RF 自动联动 |
| MLN / MLN（452 产品） | `RFTxStatus` → `AdminCellState` | 251 现场已验证 `AdminCellState` 作为 `AdminRF` 组合控制，避免与 `AdminState` 重复下发 | `FAPControl.LTE.OpState` RO | 全局及 tunnel | SN `120200055922C8B0068` 已在 251 验证越界后去激活；新门禁仍要求 GPV 控制值和 OpState=0 才标记 Verified |
| BM LTE / BM | `RFTxStatus` → `X_COM_RadioEnable` | `FAPControl.LTE.AdminState` RW | 私有 `AdminCellState` → 标准 OpState RO | 全局及 tunnel | 静态能力完整，待真机确认值语义和终态延迟 |
| BM GSM / BM | `GsmBTSCellDT.{i}.RfState` RO | 未确认；XML 将 OpState 标为 RW，不能据此下发 | `GsmBTSCellDT.{i}.OpState` 候选 | 未形成完整证据 | 禁止写 OpState，fail closed |
| BTS GSM / BTS | 模型混入 LTE RF 路径，无可信 GSM RF 控制 | 无可信管理控制 | 无完整等价终态契约 | 无完整能力 | fail closed |
| BSC/PGSM / BSC | 无设备级 RF | 无 | 无 | 无 | 不支持设备级电子围栏控制 |
| BNQ / BaiBNQ | `NR.RAN.rftxEnable` RW | `CellEnable.AdminState` / Common AdminState RW | `NR.RAN.OpState` 候选，但 XML Access 冲突 | tunnel enable | 修正/验证 OpState 只读语义及标准路径污染前 fail closed |
| CICT、Datang / ENB_DEFAULT_181 | RU `RFTxStatus` RW，FAP RF 多为 RO | LTE `AdminState` RW | LTE `OpState` RO | 模型缺完整控制 | IPSec 能力未确认，fail closed |
| 第三方、Huawei、Comba / ENB_DEFAULT_098 | 同 181 | LTE `AdminState` RW | LTE `OpState` RO | 模型缺完整控制 | IPSec 能力未确认，fail closed |

## 现场证据

### 251 BLQ，SN 120200024719AAB0039

- 2026-08-12 16:40 左右，CWMP 回读 `X_COM_RadioEnable=false`；
- 同一回读 `FAPControl.LTE.OpState=true`；
- 基站本机显示“射频状态：关、已激活”；
- OMC 列表仍显示“激活、射频开”；
- 17:41 参数树仍保存标准 `RFTxStatus=true`，私有 RF 最新值未形成有效摘要投影。

结论：BLQ RF 不会自动触发真实去激活；同时存在 GPV 别名/参数投影滞后问题。

### 452 产品（251 测试服务器），SN 120200055922C8B0068

- 2026-08-12 17:48:01，真实 SPV 对 FAPService 1/2/3 下发私有
  `CellConfig.LTE.RAN.RF.AdminCellState=0`；该历史请求包含三个模型实例，不再作为“三个实例均为
  当前有效小区”的依据；新实现必须先按 `InUse/NumOfCells` 解析目标集合；
- 设备返回 `SetParameterValuesResponse/Status=1`；无论 SPV 协议响应如何，仍需后续 GPV 和
  OpState 终态，不能单独作为新控制动作的 Verified；
- 17:49:54 设备列表显示“未激活、射频关”。

结论：251 是测试服务器，452 是产品；仓库现有设备清单将该 SN 归类为 MLN，因此这份现场证据只为
MLN/452 产品模型启用 `AdminCellState` 的 `AdminRF` 组合语义，不外推到 BLN、BM 或仅有同名路径的
其他产品。452 的旧链路已证明越界去激活生效；按 #304 扩大后的门禁，后续正式签署还要同时保存
GPV、三个小区 OpState=0、IPSec 以及控制动作终态。

## 状态机与超时

```text
outside
  → capability resolve + effective cells (InUse → NumOfCells → product/model maximum capability)
  → snapshot controls + OpState
  → SPV(Admin/AdminRF → RF → IPSec)
  → GPV controls
  → poll all target OpState inactive
  → Verified

inside
  → select changed-and-verified owned controls ∩ current effective cells
  → SPV(IPSec → RF → Admin/AdminRF)
  → GPV controls
  → poll original OpState (active if originally active)
  → Verified
```

初始终态窗口为 2 分钟，每 5 秒产生一个新的、动作关联的 GPV，最多 24 次。进度持久化到
`geofence_control_actions`，worker 重启后继续。控制值不一致立即 `partial_failed`；控制值一致但
OpState 未到终态保持 `verifying`；截止时间后 `partial_failed`，错误中列出路径、期望值和实际值。
