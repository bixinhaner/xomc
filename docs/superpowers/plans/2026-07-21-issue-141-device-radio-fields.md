# Issue #141 设备列表无线字段展示修复实施计划

> **供代理执行者：** 必须使用 `superpowers:test-driven-development` 按任务逐项实施；每完成一项先运行该项测试，再进入下一项。禁止在未取得测试环境原始参数证据前猜测新增参数路径。

**目标：** 让设备列表中的 PCI、TAC、Band、DL/UL EARFCN、Tx Power 正确展示；产品明确不支持的字段显示 `-`，产品支持但设备未上报的字段保持空白。

**架构：** 参数同步和字段投影继续由后端 `device_info` 快照承担；前端只负责依据产品能力区分“不适用”和“缺失”。参数路径按产品模型补齐，不把最大能力值误当实时发射功率。

**技术栈：** Go、PostgreSQL/pgx、React、TypeScript、Vitest。

---

## 一、问题拆分与原因结论

### 1. 不支持字段和未上报字段都显示为空白

**结论：已确认。**

- `omcmb/frontend-core/src/services/api/deviceApi.ts` 将后端 `null/undefined` 统一映射成空字符串。
- `omcmb/webcode/src/pages/device/DeviceList/index.tsx` 中 PCI、TAC、Band、DL EARFCN、UL EARFCN、Tx Power 均直接绑定字段，没有“不支持”展示规则。
- 因而当前页面无法表达两种不同语义：
  - 产品不支持：应显示 `-`；
  - 产品支持但设备未上报：应保留空白，提示数据链路仍需排查。

**根因：** 设备字段模型只有“值”，没有“该产品是否适用此字段”的能力信息；表格层也没有统一的适用性渲染器。

### 2. 部分支持字段没有进入设备列表快照

**结论：部分已确认。**

- 当前主线的 `omcgo/internal/device/device_info_sync.go` 已能投影 LTE/NR 的 PCI、TAC、Band、EARFCN，相关现有单元测试通过。因此不能再笼统归因于“后端完全没有映射”。
- `omcmb/webcode/src/pages/device/DeviceList/deviceListParamSync.ts` 的 Tx Power 同步路径只包含 LTE `ReferenceSignalPower` 和 `Capabilities.MaxTxPower`：
  - `MaxTxPower` 是能力上限，不是当前发射功率，不能作为设备列表 Tx Power；
  - 后端已经识别的 NR `PowerModify`、GSM `GsmBtsRFPower`/`BtsRfPower` 没有被列表同步任务请求，支持这些参数的产品会长期为空。
- `omcmb/frontend-core/src/services/api/deviceApi.ts` 原来用 `transmit_power >= 0` 判断有效值，导致 LTE 合法的负 dBm `ReferenceSignalPower`（例如 `-21`）在 API 映射阶段被清空；`-1` 仍按项目既有约定作为未知占位值。
- BaiBNQ 产品模型的 Band 路径是 `NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.{i}.FreqBandIndicatorNR`，原前后端却请求和投影不存在的 `NR.RAN.RF.FreqBandIndicator`，所以 5G Band 无法进入列表快照。

**根因：** 前端同步路径集合与后端投影路径集合不对称，混入了语义错误的能力参数，并且前端对功率正负号作了错误的数据有效性判断。

### 3. 测试服务器中 4G/5G 的 PCI、TAC、Band、EARFCN 全部为空

**结论：已确认测试服务器当前首先断在参数采集层，不能把所有空值归因于列表渲染。**

2026-07-21 对 `http://172.17.9.239:8081` 做了只读核验：

- 设备列表共 11 台设备，当前显示的 PCI、TAC、Band、DL EARFCN、UL EARFCN、Tx Power 六列全部为空。
- 抽查 BTS `12020003332278B0010`、BSC `6D8A45543B979167F18A530054E4`、MLN `120200055922C8B0068`、BaiBNQ `120200087125BJB0002`、BaiBLQ `120200024719AAB0039`，参数树均显示“从未同步”。
- 在上述参数树中分别检索 `BtsRfPower`、`FreqBandIndicator`、`PhyCellID`、`FreqBandIndicatorNR`，均没有结果；BaiBLQ 详情页的小区 PCI、频点、Band 也为空。
- 设备列表“自动同步”配置中“启用周期性参数同步（默认关闭）”未勾选，周期同步没有为这些设备补采参数。

由此可还原当前测试服务器的数据链路：

```text
设备支持无线参数
  └─ 周期同步关闭，且设备从未执行参数同步
       └─ device_parameters 没有目标原始值
            └─ InfoSyncer 无值可投影到 device_info
                 └─ 详情和设备列表均为空
```

这解释了“所有支持字段同时为空”的环境现象，但不替代代码修复：用户手动执行列表参数同步后，仍会受到 Tx Power 请求路径不全、BaiBNQ Band 路径错误和负功率被清空等代码问题影响。本次修复补齐这些代码断点；部署后验收必须先对在线设备执行一次参数同步，再核对原始参数、快照和列表三层结果。

本次没有在测试服务器上点击“参数同步”或保存自动同步配置，避免在只读分析阶段创建任务或改变服务器运行配置。当前页面未暴露构建 commit，因此部署版本需要由发布记录另行确认。

### 4. 产品适用性边界

- LTE/NR：PCI、TAC、Band、上下行频点属于可支持字段；截图中的“5G TAC 无值”应作为缺失数据排查，不能直接显示 `-` 掩盖。
- BSC/BTS：PCI、TAC 等 LTE/NR 专属字段不适用，应显示 `-`。
- Band、频点、发射功率是否适用于具体 GSM/BSC/BTS 产品，必须以产品 XML 参数模型为准，不使用产品名称字符串猜测。

## 二、修复方案

### 方案原则

1. 后端值为空不等于“不支持”。
2. 产品能力判断集中在一个纯函数中，列表、列导出共用，避免页面与导出结果不一致。
3. 参数采集路径与后端投影路径必须成对维护并由测试约束。
4. 不修改空值为 `0`，不把 `MaxTxPower` 当作当前 Tx Power。

## 三、实施任务

### 任务 1：建立测试环境数据证据

**只读核验，不修改代码。**

- [x] 记录测试环境地址；页面未暴露构建 Git commit，保留为发布侧核对项。
- [x] 对 LTE、NR、BSC、BTS 分别抽样核对参数树、详情和列表。
- [x] 确认本轮测试环境的首要断点是“同步未执行”，目标原始参数尚未进入 OMC。
- [x] 核对自动同步配置为关闭；未在只读分析阶段修改配置或创建同步任务。

验收：每一个截图框选字段都有数据链路证据，能够回答“设备是否上报、OMC 是否保存、列表为何为空”。

### 任务 2：补齐无线字段同步路径并锁定投影行为

**文件：**

- 修改：`omcmb/webcode/src/pages/device/DeviceList/deviceListParamSync.ts`
- 修改：`omcmb/webcode/src/pages/device/DeviceList/deviceBatchTask.test.ts`
- 修改：`omcmb/frontend-core/src/services/api/deviceApi.ts`
- 修改：`omcmb/frontend-core/src/services/api/__tests__/deviceApi.test.ts`
- 视任务 1 证据修改：`omcgo/internal/device/device_info_sync.go`
- 视任务 1 证据修改：`omcgo/internal/device/device_info_sync_test.go`

- [ ] 先写失败测试：同步路径应包含已确认的 LTE/NR/GSM 当前功率路径，且不使用 `Capabilities.MaxTxPower` 作为 Tx Power。
- [ ] 若任务 1 发现新路径，先为具体产品、具体字段补后端失败测试，再添加最小映射。
- [ ] 验证多实例字段仍按既有聚合规则输出，不覆盖其他小区实例。

前端定向测试：

```bash
cd omcmb && npm test --workspace webcode -- --run src/pages/device/DeviceList/deviceListParamSync.test.ts
```

后端定向测试：

```bash
cd omcgo && go test ./internal/device -run 'TestInfoSyncer|TestUniversalInformMapping|TestAggregateInstanceFields' -count=1
```

### 任务 3：增加字段适用性模型和统一展示

**文件：**

- 新增：`omcmb/webcode/src/pages/device/DeviceList/deviceRadioFieldSupport.ts`
- 新增：`omcmb/webcode/src/pages/device/DeviceList/deviceRadioFieldSupport.test.ts`
- 修改：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`
- 修改列表导出所调用的同一格式化入口（以实施时实际函数位置为准）。

建议接口：

```ts
type RadioField = 'pci' | 'tac' | 'band' | 'dlEarfcn' | 'ulEarfcn' | 'txPower';

export function isRadioFieldSupported(
  device: Pick<Device, 'technology' | 'productClass' | 'productName'>,
  field: RadioField,
): boolean;

export function formatRadioField(
  device: Pick<Device, 'technology' | 'productClass' | 'productName'>,
  field: RadioField,
  value: unknown,
): string;
```

- [ ] 先写失败测试：BSC/BTS 的 PCI、TAC 返回 `-`。
- [ ] 先写失败测试：LTE/NR 支持字段为空时返回空字符串，而不是 `-`。
- [ ] 先写失败测试：非空值原样展示；数值 `0` 不得被当成空值。
- [ ] 表格 render 与导出格式化共用该函数。
- [ ] 能力矩阵只根据后端已有规范化技术类型和已确认产品模型建立，未知产品默认“数据缺失”，不贸然判定不支持。

### 任务 4：完整验证

- [ ] 运行前端类型检查：`cd omcmb && npm run typecheck`
- [ ] 运行设备列表相关测试：`cd omcmb && npm test --workspace webcode -- --run src/pages/device/DeviceList`
- [ ] 若修改后端，运行：`cd omcgo && go build ./... && go test ./internal/device -count=1`
- [ ] 部署测试环境后重新执行参数同步。
- [ ] 按 LTE、NR、BSC、BTS 验收：支持且上报显示值；支持但未上报为空白；不支持显示 `-`；导出结果与页面一致。

## 四、不接受的表面修复

- 对所有空值统一显示 `-`：会掩盖“应上报但缺失”的真实问题。
- 仅在截图所示列上写 JSX 三元表达式：导出和其他入口仍会不一致。
- 继续增加模糊参数路径但不核对原始数据：容易把不同语义字段映射到一起。
- 用 `MaxTxPower` 填充 Tx Power：能力上限不等于当前发射功率。
