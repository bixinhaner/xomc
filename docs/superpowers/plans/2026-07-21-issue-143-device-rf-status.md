# Issue #143 设备列表 RF 状态修复实施计划

> **供代理执行者：** 必须使用 `superpowers:test-driven-development` 按任务逐项实施；先用失败测试证明状态语义，再修改实现。禁止仅更换图标或文案绕过数据错误。

**目标：** 设备列表 RF 状态与设备实际状态一致；不支持或未取得 RF 状态时显示 `-`，不再错误显示 `RF Off`。

**架构：** 后端从各产品实际参数路径计算规范化 RF 状态；前端仅渲染后端状态，不再用连接状态覆盖 RF 状态。多小区状态沿用现有多值展示机制。

**技术栈：** Go、React、TypeScript、Vitest。

---

## 一、问题拆分与原因结论

### 1. 没有 RF 参数时被计算为 Off

**结论：已确认。**

`omcgo/internal/device/device_info_calc.go` 的 `CalcRFStatus` 在找不到任何已识别参数时直接返回 `"off"`。现有 `TestCalcRFStatus/empty_params` 还把该错误语义固定成了测试期望。

**根因：** 计算函数没有“未知/不支持”状态，把“没有证据”当成“已关闭”。这会让所有参数路径未覆盖的网元统一显示 RF Off。

### 2. 产品 RF 参数路径覆盖不完整

**结论：已确认。**

- 当前实现只读取固定 LTE、NR `X_COM_RadioEnable` 和少量遗留 `RFTxStatus` 路径。
- 设备列表局部同步接口按 `standardPath` 过滤产品映射，但原实现请求了部分产品的私有路径；即使设备支持，该映射也不会进入同步计划。
- BTS、BLQ 等产品把私有 `CellConfig.LTE.RAN.RF.X_COM_RadioEnable` 归一到标准路径 `FAPControl.LTE.RFTxStatus`，后端却没有读取该标准路径。
- BaiBNQ 使用标准路径 `CellConfig.{i}.NR.RAN.rftxEnable`，其私有 SAS 开关会归一到 `Device.DeviceInfo.SAS.RadioEnable`；GSM BTS 使用 `GsmBTSCellDT.{i}.RfState`。原同步集合和计算函数均未完整覆盖。

**根因：** 参数请求和后端计算各自维护了不完整的路径清单，且缺乏跨产品契约测试。

### 3. 前端用离线状态强制覆盖 RF 状态

**结论：已确认。**

- `omcmb/frontend-core/src/utils/rfStatus.ts` 在 `isOnline === false` 时无条件返回 `off`。
- `omcmb/webcode/src/pages/device/DeviceList/index.tsx` 的列表和导出也有同样的离线覆盖逻辑。
- 现有前端测试明确断言“离线设备即使原始值为 ON 也显示 OFF”。

**根因：** 混淆了“管理连接状态”和“最近一次上报的射频状态”。二者是独立维度，连接离线不能证明射频已关闭。

### 4. 测试服务器数据链路证据

**结论：已确认服务器现象与上述根因一致。**

2026-07-21 对 `http://172.17.9.239:8081` 做了故障基线核验，并在用户要求验证后执行了三台在线设备的受控列表参数同步：

- BTS `12020003332278B0010`：任务完成，已同步 16 条；参数树仍检索不到 `RFTxStatus` 和 `RadioEnable`。
- BaiBNQ `120200087125BJB0002`：任务完成，已同步 12 条、跳过 2 条无关的不支持参数；参数树仍检索不到 `rftxEnable` 和 `RadioEnable`。
- BaiBLQ `120200024719AAB0039`：任务完成，已同步 32 条；PCI、Band、频点和 Tx Power 已进入列表，证明总体同步与快照投影链路工作正常，但参数树仍检索不到 `RFTxStatus` 和 `RadioEnable`。
- 服务器的周期参数同步开关未启用；执行受控同步前，上述设备参数树均显示“从未同步”。
- Issue 截图已给出上述 BTS、5G、4G 设备 LMT RF 为 ON，而列表显示 RF Off；在 OMC 没有 RF 原始参数的前提下，现有 `CalcRFStatus` 恰好会无条件生成 `off`，随后前端又可能因连接状态再次覆盖成 `off`。

因此当前错误不是“设备明确上报了 Off”，而是旧代码没有把 RF 标准路径纳入局部同步，又在缺少采集证据时制造了 Off。部署修复后应再次执行参数同步，再按“设备私有路径 → 产品标准路径 → `device_parameters` → `device_info.rf_status` → 列表”逐层验收。验证仅创建了上述三台设备的参数同步任务，没有修改自动同步或其他服务器配置。

## 二、修复方案

RF 状态采用三态语义：

| 后端规范值 | 含义 | 页面展示 |
| --- | --- | --- |
| `on` | 参数明确表示开启 | RF On |
| `off` | 参数明确表示关闭 | RF Off |
| 空/unknown | 不支持、未上报或路径未识别 | `-` |

多小区设备继续输出逗号分隔的规范状态，由现有多小区渲染器展示。连接状态不得改变 RF 值。

## 三、实施任务

### 任务 1：重写后端 RF 状态契约测试

**文件：**

- 修改：`omcgo/internal/device/info_calc_test.go`
- 修改：`omcgo/internal/device/device_info_calc.go`

- [ ] 将空参数期望由 `off` 改为空/unknown，先确认测试失败。
- [ ] 添加 LTE 开/关、NR 带实例开/关、GSM/BTS、BaiBNQ/SAS 路径测试。
- [ ] 添加多小区混合状态和无效值测试。
- [ ] 明确定义各协议值的映射，至少覆盖布尔值及项目现有的数字枚举；未知枚举不得默认 Off。
- [ ] 实现最小的路径匹配和规范化逻辑，使测试通过。

定向测试：

```bash
cd omcgo && go test ./internal/device -run '^TestCalcRFStatus$' -count=1 -v
```

### 任务 2：使参数同步路径与计算路径一致

**文件：**

- 修改：`omcmb/webcode/src/pages/device/DeviceList/deviceListParamSync.ts`
- 修改：`omcmb/webcode/src/pages/device/DeviceList/deviceListParamSync.test.ts`

- [ ] 为每条后端支持的 RF 路径添加同步路径测试。
- [ ] 补齐 LTE、带实例 NR、SAS/BaiBNQ 和经产品模型确认的 GSM/BTS 路径。
- [ ] 验证不支持 RF 的产品不会因同步失败被写成 Off。

### 任务 3：移除前端连接状态覆盖

**文件：**

- 修改：`omcmb/frontend-core/src/utils/rfStatus.ts`
- 修改：`omcmb/frontend-core/src/utils/__tests__/rfStatus.test.ts`
- 修改：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`
- 修改：设备列表导出相关测试。

- [ ] 先反转现有错误测试：离线且最近状态为 On 时仍显示 RF On。
- [ ] 添加空值/unknown 显示 `-` 的测试。
- [ ] 列表、详情入口和导出只调用统一 RF 格式化函数，不自行根据 `isOnline` 改写状态。
- [ ] 保留多小区状态展示，不将部分小区 Off 简化成整机 Off。

定向测试：

```bash
cd omcmb && npm test --workspace webcode -- --run ../frontend-core/src/utils/__tests__/rfStatus.test.ts
```

### 任务 4：完整验证

- [ ] `cd omcgo && go build ./... && go test ./internal/device -count=1`
- [ ] `cd omcmb && npm run typecheck`
- [ ] `cd omcmb && npm test --workspace webcode -- --run src/pages/device/DeviceList ../frontend-core/src/utils/__tests__/rfStatus.test.ts`
- [ ] 在测试环境分别核对截图中的 BTS、5G、4G 设备：LMT 为 ON 时列表为 RF On。
- [ ] 核对不支持 RF 或无上报值的网元显示 `-`。
- [ ] 核对在线/离线切换不会篡改最后一次有效 RF 状态。

## 四、不接受的表面修复

- 把红色 RF Off 标签换颜色或图标：数据仍然错误。
- 只为截图中的三个 SN 写例外：无法覆盖同产品其他设备。
- 无参数时继续默认 Off：仍会制造错误告警感知。
- 以连接状态推导 RF 状态：协议语义不成立。
