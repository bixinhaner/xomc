# MML 控制台命令清单全表 · CMCC TD-LTE v2.3

> **派生文档**：本文档由 `omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md` 按调整方案文档 §6.7 Catalog Loader 的拆分规则（R-1/R-2/R-3）枚举所得，作为人类可读的"实施视角"清单，与离线工具产出的 `omcgo/datamodels/mml-catalog/cmcc-tdlte-v23.json` 在数据维度上**一一对应**。
>
> 修改 spec MD 之后，本文档应由离线工具同步重生成；以 JSON 文件的 `sourceDocSha256` 校验为准（详见调整方案 §5.3）。
>
> **版本**：cmcc-tdlte-v2.3 · **生成口径**：R-1 object 一级 / R-2 per-op 二级 / R-3 operation_type 固化

---

## 0. 速览

| 维度 | 数量 |
|------|------|
| 一级分组 (object groups) | **72** （spec 73 H4 块 - 1 个 SQ 伪命令 `.*`） |
| 二级命令 (per-op) | **约 200**（每个分组按权限矩阵展开 1-4 行） |
| TR-181 唯一路径 | **625**（去重后） |
| 章节 (chapter) | SA … SR 共 18（仅排序元数据，**不渲染为树节点**） |

**权限标记**：📖 R = 只读 · 📝 RW = 可读可写 · 📖📝 = 混合
**操作生成规则（R-3）**：
- **LST** 恒生成（GetParameterValues 全集，含 RO + RW）
- **MOD** 仅当路径集存在 ≥ 1 条 📝 → 生成（SetParameterValues，target_paths = RW 子集）
- **ADD / RMV** 仅当路径模板含 `{i}` **且** 路径集存在 ≥ 1 条 📝 → 同时生成（AddObject / DeleteObject，target_paths = 父级对象路径）
- 全 RO 的多实例对象（如 CurrentAlarm.{i}.*）→ 只 LST
- 全 RO 的非多实例对象（如 SwUpgrade.*）→ 只 LST
- **`instance_arity`** = 路径模板中 `{i}/{iα}/{iβ}/...` 占位符的层数

---

## 1. 主索引（72 group · 按 SA-SR locality 排序，**树上不显示章节名**）

| # | Chinese alias | group_code (TR-181) | chapter | perm | arity | LST | MOD | ADD | RMV |
|---|---------------|----------------------|---------|------|-------|-----|-----|-----|-----|
| G-01 | 设备信息 | `Device.DeviceInfo.*` | SA | 📖📝 (R=15/RW=2) | 0 | 17 | 2 | — | — |
| G-02 | 设备版本升级 | `Device.DeviceInfo.SwUpgrade.*` | SA | 📖 (R=3) | 0 | 3 | — | — | — |
| G-03 | 软件控制 | `Device.SoftwareCtrl.*` | SB | 📖📝 (R=2/RW=3) | 0 | 5 | 3 | — | — |
| G-04 | 网管参数 | `Device.ManagementServer.*` | SC | 📖📝 (R=4/RW=15) | 0 | 19 | 15 | — | — |
| G-05 | 故障管理 | `Device.FaultMgmt.*` | SD | 📖 (R=6) | 0 | 6 | — | — | — |
| G-06 | 当前告警实例 | `Device.FaultMgmt.CurrentAlarm.{i}.*` | SD | 📖 (R=11) | 1 | 11 | — | — | — |
| G-07 | 实时告警实例 | `Device.FaultMgmt.ExpeditedEvent.{i}.*` | SD | 📖 (R=11) | 1 | 11 | — | — | — |
| G-08 | 历史告警实例 | `Device.FaultMgmt.HistoryEvent.{i}.*` | SD | 📖 (R=11) | 1 | 11 | — | — | — |
| G-09 | 队列告警实例 | `Device.FaultMgmt.QueuedEvent.{i}.*` | SD | 📖 (R=11) | 1 | 11 | — | — | — |
| G-10 | 支持告警实例 | `Device.FaultMgmt.SupportedAlarm.{i}.*` | SD | 📖📝 (R=4/RW=1) | 1 | 5 | 1 | ✓ | ✓ |
| G-11 | 日志管理 | `Device.LogMgmt.*` | SE | 📝 (RW=5) | 0 | 5 | 5 | — | — |
| G-12 | FAPControl LTE | `Device.Services.FAPControl.LTE.*` | SF | 📖📝 (R=2/RW=1) | 0 | 3 | 1 | — | — |
| G-13 | 安全/接入网关 | `Device.Services.FAPControl.LTE.Gateway.*` | SF | 📝 (RW=10) | 0 | 10 | 10 | — | — |
| G-14 | MME池配置 | `Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.*` | SF | 📖📝 (R=3/RW=2) | 1 | 5 | 2 | ✓ | ✓ |
| G-15 | S1U | `Device.Services.FAPControl.LTE.S1U.{i}.*` | SF | 📖 (R=2) | 1 | 2 | — | — | — |
| G-16 | X2 IP映射 | `Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*` | SF | 📝 (RW=5) | 1 | 5 | 5 | ✓ | ✓ |
| G-17 | FAPService 载波 | `Device.Services.FAPService.{i}.*` (SF) | SF | 📖📝 (R=2/RW=22) | 1 (i=1~3 固定) | 24 | 22 | ✓⚠ | ✓⚠ |
| G-18 | FAPService Capabilities | `Device.Services.FAPService.{i}.Capabilities.*` | SF | 📝 (RW=1) | 1 | 1 | 1 | ✓⚠ | ✓⚠ |
| G-19 | CellConfig Capabilities | `Device.Services.FAPService.{i}.CellConfig.Capabilities.*` | SF | 📖📝 (R=2/RW=1) | 1 | 3 | 1 | ✓⚠ | ✓⚠ |
| G-20 | EPC | `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.*` | SF | 📝 (RW=2) | 1 | 2 | 2 | ✓⚠ | ✓⚠ |
| G-21 | PLMNList | `Device.Services.FAPService.{iα}.CellConfig.LTE.EPC.PLMNList.{iβ}.*` | SF | 📝 (RW=2) | 2 | 2 | 2 | ✓ | ✓ |
| G-22 | VoLTE PDCP初始 | `Device.Services.FAPService.{iα}.CellConfig.LTE.VoLTE.PdcpInitParam.{iβ}.*` | SF | 📝 (RW=1) | 2 | 1 | 1 | ✓ | ✓ |
| G-23 | SCTP | `Device.Services.FAPControl.Transport.SCTP.*` | SG | 📝 (RW=9) | 0 | 9 | 9 | — | — |
| G-24 | SCTP Assoc | `Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.*` | SG | 📖 (R=4) | 1 | 4 | — | — | — |
| G-25 | RRC Timers | `Device.Services.FAPService.{i}.*` (SH; RRCTimers) | SH | 📝 (RW=10) | 1 (i=1~3 固定) | 10 | 10 | ✓⚠ | ✓⚠ |
| G-26 | MAC | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.*` | SH | 📝 (RW=16) | 1 | 16 | 16 | ✓⚠ | ✓⚠ |
| G-27 | DRX 初始 | `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{iβ}.*` | SH | 📝 (RW=6) | 2 | 6 | 6 | ✓ | ✓ |
| G-28 | PHY | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.*` | SH | 📖📝 (R=2/RW=33) | 1 | 35 | 33 | ✓⚠ | ✓⚠ |
| G-29 | PHY MBSFN | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.*` | SH | 📝 (RW=1) | 1 | 1 | 1 | ✓⚠ | ✓⚠ |
| G-30 | MBSFN SFConfigList | `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{iβ}.*` | SH | 📝 (RW=5) | 2 | 5 | 5 | ✓ | ✓ |
| G-31 | GSM 邻区 | `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{iβ}.*` | SI | 📝 (RW=7) | 2 | 7 | 7 | ✓ | ✓ |
| G-32 | NR 邻区 | `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{iβ}.*` | SI | 📝 (RW=12) | 2 | 12 | 12 | ✓ | ✓ |
| G-33 | UMTS 邻区 | `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{iβ}.*` | SI | 📝 (RW=10) | 2 | 10 | 10 | ✓ | ✓ |
| G-34 | LTE 邻区 | `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.LTECell.{iβ}.*` | SI | 📝 (RW=10) | 2 | 10 | 10 | ✓ | ✓ |
| G-35 | ConnMode EUTRA | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.*` | SJ | 📝 (RW=1) | 1 | 1 | 1 | ✓⚠ | ✓⚠ |
| G-36 | A1 测量控制 | `...A1MeasureCtrl.{iβ}.*` | SJ | 📖📝 (R=1/RW=10) | 2 | 11 | 10 | ✓ | ✓ |
| G-37 | A2 测量控制 | `...A2MeasureCtrl.{iβ}.*` | SJ | 📖📝 (R=1/RW=10) | 2 | 11 | 10 | ✓ | ✓ |
| G-38 | A3 测量控制 | `...A3MeasureCtrl.{iβ}.*` | SJ | 📖📝 (R=1/RW=10) | 2 | 11 | 10 | ✓ | ✓ |
| G-39 | A4 测量控制 | `...A4MeasureCtrl.{iβ}.*` | SJ | 📖📝 (R=1/RW=10) | 2 | 11 | 10 | ✓ | ✓ |
| G-40 | A5 测量控制 | `...A5MeasureCtrl.{iβ}.*` | SJ | 📖📝 (R=1/RW=12) | 2 | 13 | 12 | ✓ | ✓ |
| G-41 | 周期测量控制 | `...PeriodMeasCtrl.{iβ}.*` | SJ | 📝 (RW=4) | 2 | 4 | 4 | ✓ | ✓ |
| G-42 | ConnMode IRAT | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.*` | SJ | 📝 (RW=4) | 1 | 4 | 4 | ✓⚠ | ✓⚠ |
| G-43 | B1 测量控制 | `...IRAT.B1MeasureCtrl.{iβ}.*` | SJ | 📖📝 (R=1/RW=10) | 2 | 11 | 10 | ✓ | ✓ |
| G-44 | B2 测量控制 | `...IRAT.B2MeasureCtrl.{iβ}.*` | SJ | 📖📝 (R=1/RW=12) | 2 | 13 | 12 | ✓ | ✓ |
| G-45 | IdleMode | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.*` | SJ | 📝 (RW=28) | 1 | 28 | 28 | ✓⚠ | ✓⚠ |
| G-46 | IdleMode IRAT | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.*` | SJ | 📝 (RW=2) | 1 | 2 | 2 | ✓⚠ | ✓⚠ |
| G-47 | GERAN 频组 | `...IdleMode.IRAT.GERAN.GERANFreqGroup.{iβ}.*` | SJ | 📝 (RW=6) | 2 | 6 | 6 | ✓ | ✓ |
| G-48 | UTRA FDD 频点 | `...IdleMode.IRAT.UTRA.UTRANFDDFreq.{iβ}.*` | SJ | 📝 (RW=6) | 2 | 6 | 6 | ✓ | ✓ |
| G-49 | 异频载波 | `...IdleMode.InterFreq.Carrier.{iβ}.*` | SJ | 📝 (RW=13) | 2 | 13 | 13 | ✓ | ✓ |
| G-50 | SON 配置 | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.*` | SK | 📖📝 (R=1/RW=24) | 1 | 25 | 24 | ✓⚠ | ✓⚠ |
| G-51 | 自配置启动 | `Device.Services.FAPService.{i}.FAPControl.SelfConfig.*` | SK | 📖 (R=3) | 1 | 3 | — | — | — |
| G-52 | 以太网接口 | `Device.Ethernet.Interface.{i}.*` | SL | 📖📝 (R=5/RW=4) | 1 | 9 | 4 | ✓ | ✓ |
| G-53 | IPv4 地址 | `Device.Ethernet.Interface.{iα}.IPv4Address.{iβ}.*` | SL | 📝 (RW=5) | 2 | 5 | 5 | ✓ | ✓ |
| G-54 | IPv6 地址 | `Device.Ethernet.Interface.{iα}.IPv6Address.{iβ}.*` | SL | 📝 (RW=5) | 2 | 5 | 5 | ✓ | ✓ |
| G-55 | VLAN 接口 | `Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.*` | SL | 📝 (RW=3) | 2 | 3 | 3 | ✓ | ✓ |
| G-56 | VLAN IPv4 | `Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv4Address.{iγ}.*` | SL | 📝 (RW=5) | 3 | 5 | 5 | ✓ | ✓ |
| G-57 | VLAN IPv6 | `Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv6Address.{iγ}.*` | SL | 📝 (RW=5) | 3 | 5 | 5 | ✓ | ✓ |
| G-58 | IP 路由 | `Device.Ethernet.IpRoute.{i}.*` | SL | 📝 (RW=5) | 1 | 5 | 5 | ✓ | ✓ |
| G-59 | IPsec | `Device.IPsec.*` | SM | 📖📝 (R=7/RW=2) | 0 | 9 | 2 | — | — |
| G-60 | 时间服务器 | `Device.Time.*` | SN | 📖📝 (R=1/RW=7) | 0 | 8 | 7 | — | — |
| G-61 | GPS | `Device.FAP.GPS.*` | SO | 📖 (R=3) | 0 | 3 | — | — | — |
| G-62 | MR 配置 | `Device.FAP.MRMgmt.Config.{i}.*` | SP | 📝 (RW=14) | 1 | 14 | 14 | ✓ | ✓ |
| G-63 | PM 配置 | `Device.FAP.PerfMgmt.Config.{i}.*` | SQ | 📝 (RW=10) | 1 | 10 | 10 | ✓ | ✓ |
| G-64 | MU 主机单元 | `Device.DeviceInfo.MU.{i}.*` | SR | 📖📝 (R=21/RW=3) | 1 | 24 | 3 | ✓⚠ | ✓⚠ |
| G-65 | Slot 板卡 | `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.*` | SR | 📖📝 (R=15/RW=1) | 2 | 16 | 1 | ✓⚠ | ✓⚠ |
| G-66 | EU 扩展单元 | `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.*` | SR | 📖📝 (R=11/RW=2) | 3 | 13 | 2 | ✓⚠ | ✓⚠ |
| G-67 | RU 远端单元 | `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.*` | SR | 📖📝 (R=12/RW=4) | 4 | 16 | 4 | ✓⚠ | ✓⚠ |
| G-68 | RFChannel 射频通道 | `...RFChannel.{iε}.*` | SR | 📖📝 (R=1/RW=1) | 5 | 2 | 1 | ✓⚠ | ✓⚠ |
| G-69 | RU 升级 | `...RU.{iδ}.SwUpgrade.*` | SR | 📖 (R=3) | 4 | 3 | — | — | — |
| G-70 | EU 升级 | `...EU.{iγ}.SwUpgrade.*` | SR | 📖 (R=3) | 3 | 3 | — | — | — |
| G-71 | Slot 升级 | `...Slot.{iβ}.SwUpgrade.*` | SR | 📖 (R=3) | 2 | 3 | — | — | — |
| G-72 | MU 升级 | `Device.DeviceInfo.MU.{i}.SwUpgrade.*` | SR | 📖 (R=3) | 1 | 3 | — | — | — |

> **`✓⚠`** 标记：根据 R-2 启发式自动生成 ADD/RMV（含 `{i}` + 至少 1 条 RW），但该 object 在 TR-069 业务语义上**很可能不是用户可增删的容器**（如固定枚举的 FAPService 载波 i=1~3、硬件描述的 MU/Slot/EU/RU 等）。这部分由 §12 D31 决策处理：① P2 先全自动生成 ② 后续通过 admin overlay 隐藏不合理项。
>
> 跨章节"伪命令"过滤：SQ - FAP.PerfMgmt 段下有 `#### 命令: .*`（spec 第 1166 行），其内容是规范引用的注释段，**不视为命令**，本清单已剔除。
>
> 实例参数（如 LTECell `.{i}.{i}`）汇总于 spec 附录 A，本清单仅列示主命令；运行时实例号由前端 InstanceArityInput 填写、后端 `applyInstanceSelectors` 折叠。

---

## 2. 路径明细（按 group 顺序）

> 每个 group 显示：`group_code` · chapter · 权限统计 · 实例层数 · 生成的命令列表 · TR-181 leaf 参数清单（含权限标记与类型）。
> 路径前缀已省略（与 `group_code` 一致），仅列**叶子参数名**。
> 命名约定：`📖` = RO（GetParameterValues only）/ `📝` = RW（GetParameterValues + SetParameterValues）。

---

### G-01. 设备信息 · `Device.DeviceInfo.*`
- chapter: SA · perm: 📖📝 (R=15/RW=2) · arity=0
- 命令: **LST 设备信息** (17 paths) / **MOD 设备信息** (2 paths)
- 路径:
  📝 UserLabel · string · 📝 DnPrefix · string · 📖 ManufacturerOUI · string(6) · 📖 Manufacturer · string(64) · 📖 ModelName · string(64) · 📖 SerialNumber · string(64) · 📖 HardwareVersion · string(64) · 📖 SoftwareVersion · string(64) · 📖 HardwarePlatform · string(64) · 📖 AdditionalHardwareVersion · string(64) · 📖 AdditionalSoftwareVersion · string(64) · 📖 ProvisioningCode · string(64) · 📖 ProductClass · string(64) · 📖 UpTime · unsignedInt · 📖 3GPPSpecVersion · string · 📖 FirstUseDate · dateTime · 📖 DataModelSpecVersion · string

### G-02. 设备版本升级 · `Device.DeviceInfo.SwUpgrade.*`
- chapter: SA · perm: 📖 (R=3) · arity=0
- 命令: **LST 设备版本升级** (3 paths)
- 路径: 📖 Stage · unsignedInt[1:5] · 📖 FailureCause · string(512) · 📖 Status · unsignedInt[1:3]

### G-03. 软件控制 · `Device.SoftwareCtrl.*`
- chapter: SB · perm: 📖📝 (R=2/RW=3) · arity=0
- 命令: **LST 软件控制** (5) / **MOD 软件控制** (3)
- 路径: 📝 AutoActivateEnable · boolean · 📝 ActivateTime · dateTime · 📝 ActivateEnable · boolean · 📖 SystemCurrentVersion · string(64) · 📖 SystemBackupVersion · string(64)

### G-04. 网管参数 · `Device.ManagementServer.*`
- chapter: SC · perm: 📖📝 (R=4/RW=15) · arity=0
- 命令: **LST 网管参数** (19) / **MOD 网管参数** (15)
- 路径: 📝 URL · string(256) · 📝 Username · string(256) · 📝 Password · string(256) · 📝 PeriodicInformEnable · boolean · 📝 PeriodicInformTime · dateTime · 📝 PeriodicInformInterval · unsignedInt[1:] · 📖 ParameterKey · string(32) · 📖 ConnectionRequestURL · string(256) · 📝 ConnectionRequestUsername · string(256) · 📝 ConnectionRequestPassword · string(256) · 📖 UDPConnectionRequestAddress · string(256) · 📝 STUNEnable · boolean · 📝 STUNServerAddress · string(256) · 📝 STUNServerPort · unsignedInt[0:65535] · 📝 STUNUsername · string(256) · 📝 STUNPassword · string(256) · 📝 STUNMaximumKeepAlivePeriod · int[-1:65535] · 📝 STUNMinimumKeepAlivePeriod · unsignedInt[0:65535] · 📖 NATDetected · boolean

### G-05. 故障管理 · `Device.FaultMgmt.*`
- chapter: SD · perm: 📖 (R=6) · arity=0
- 命令: **LST 故障管理** (6)
- 路径: 📖 SupportedAlarmNumberOfEntries · unsignedInt · 📖 MaxCurrentAlarmEntries · unsignedInt · 📖 CurrentAlarmNumberOfEntries · unsignedInt · 📖 HistoryEventNumberOfEntries · unsignedInt · 📖 ExpeditedEventNumberOfEntries · unsignedInt · 📖 QueuedEventNumberOfEntries · unsignedInt

### G-06. 当前告警实例 · `Device.FaultMgmt.CurrentAlarm.{i}.*`
- chapter: SD · perm: 📖 (R=11) · arity=1（事件表，read-only）
- 命令: **LST 当前告警实例** (11) — 无 MOD/ADD/RMV（设备自维护）
- 路径: 📖 AlarmIdentifier · string(20) · 📖 AlarmRaisedTime · dateTime · 📖 AlarmChangedTime · dateTime · 📖 FaultLocation · string(512) · 📖 ManagedObjectInstance · string(512) · 📖 EventType · string(64) · 📖 ProbableCause · string(64) · 📖 SpecificProblem · string(128) · 📖 PerceivedSeverity · string · 📖 AdditionalText · string(256) · 📖 AdditionalInformation · string(256)

### G-07. 实时告警实例 · `Device.FaultMgmt.ExpeditedEvent.{i}.*`
- chapter: SD · perm: 📖 (R=11) · arity=1
- 命令: **LST 实时告警实例** (11)
- 路径: 📖 EventTime · dateTime · 📖 AlarmIdentifier · string(20) · 📖 NotificationType · string · 📖 FaultLocation · string(512) · 📖 ManagedObjectInstance · string(512) · 📖 EventType · string(64) · 📖 ProbableCause · string(64) · 📖 SpecificProblem · string(128) · 📖 PerceivedSeverity · string · 📖 AdditionalText · string(256) · 📖 AdditionalInformation · string(256)

### G-08. 历史告警实例 · `Device.FaultMgmt.HistoryEvent.{i}.*`
- chapter: SD · perm: 📖 (R=11) · arity=1
- 命令: **LST 历史告警实例** (11)
- 路径: 📖 EventTime · dateTime · 📖 AlarmIdentifier · string(20) · 📖 NotificationType · string · 📖 FaultLocation · string(512) · 📖 ManagedObjectInstance · string(512) · 📖 EventType · string(64) · 📖 ProbableCause · string(64) · 📖 SpecificProblem · string(128) · 📖 PerceivedSeverity · string · 📖 AdditionalText · string(256) · 📖 AdditionalInformation · string(256)

### G-09. 队列告警实例 · `Device.FaultMgmt.QueuedEvent.{i}.*`
- chapter: SD · perm: 📖 (R=11) · arity=1
- 命令: **LST 队列告警实例** (11)
- 路径: 📖 EventTime · dateTime · 📖 AlarmIdentifier · string(20) · 📖 NotificationType · string · 📖 FaultLocation · string(512) · 📖 ManagedObjectInstance · string(512) · 📖 EventType · string(64) · 📖 ProbableCause · string(64) · 📖 SpecificProblem · string(128) · 📖 PerceivedSeverity · string · 📖 AdditionalText · string(256) · 📖 AdditionalInformation · string(256)

### G-10. 支持告警实例 · `Device.FaultMgmt.SupportedAlarm.{i}.*`
- chapter: SD · perm: 📖📝 (R=4/RW=1) · arity=1
- 命令: **LST 支持告警实例** (5) / **MOD 支持告警实例** (1) / **ADD 支持告警实例** / **RMV 支持告警实例**
- 路径: 📖 EventType · string(64) · 📖 ProbableCause · string(64) · 📖 SpecificProblem · string(128) · 📖 PerceivedSeverity · string · 📝 ReportingMechanism · string

### G-11. 日志管理 · `Device.LogMgmt.*`
- chapter: SE · perm: 📝 (RW=5) · arity=0
- 命令: **LST 日志管理** (5) / **MOD 日志管理** (5)
- 路径: 📝 PeriodicUploadEnable · boolean · 📝 URL · string(256) · 📝 Username · string(256) · 📝 Password · string(256) · 📝 PeriodicUploadInterval · unsignedInt[1:]

### G-12. FAPControl LTE · `Device.Services.FAPControl.LTE.*`
- chapter: SF · perm: 📖📝 (R=2/RW=1) · arity=0
- 命令: **LST FAPControl LTE** (3) / **MOD FAPControl LTE** (1)
- 路径: 📝 AdminState · boolean · 📖 OpState · boolean · 📖 RFTxStatus · boolean

### G-13. 安全/接入网关 · `Device.Services.FAPControl.LTE.Gateway.*`
- chapter: SF · perm: 📝 (RW=10) · arity=0
- 命令: **LST Gateway** (10) / **MOD Gateway** (10)
- 路径: 📝 SecGWServer1 · string(64) · 📝 SecGWServer2 · string(64) · 📝 SecGWServer3 · string(64) · 📝 AGServerEnable · boolean · 📝 AGServerIp1 · string(64) · 📝 AGServerIp2 · string(64) · 📝 AGServerIp3 · string(64) · 📝 AGPort1 · string(64) · 📝 AGPort2 · string(64) · 📝 AGPort3 · string(64)

### G-14. MME 池配置 · `Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.*`
- chapter: SF · perm: 📖📝 (R=3/RW=2) · arity=1
- 命令: **LST MmePool** (5) / **MOD MmePool** (2) / **ADD MmePool** / **RMV MmePool**
- 路径: 📖 PLMNIDList · string(64) · 📖 MMEGroupID · unsignedInt[0:65535] · 📖 MMECode · unsignedInt[0:255] · 📝 MMEIp1 · string(64) · 📝 MMEIp2 · string(64)

### G-15. S1U · `Device.Services.FAPControl.LTE.S1U.{i}.*`
- chapter: SF · perm: 📖 (R=2) · arity=1
- 命令: **LST S1U** (2)
- 路径: 📖 LocIpAddrList · string(64) · 📖 FarIpSubnetworkList · string(64)

### G-16. X2 IP映射 · `Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*`
- chapter: SF · perm: 📝 (RW=5) · arity=1
- 命令: **LST X2IpAddrMap** (5) / **MOD X2IpAddrMap** (5) / **ADD X2IpAddrMap** / **RMV X2IpAddrMap**
- 路径: 📝 PLMNIDList · string(64) · 📝 EnbType · unsignedInt · 📝 EnbId · unsignedInt · 📝 WanIpAddress · string(64) · 📝 SubnetMask · string(64)

### G-17. FAPService 载波 · `Device.Services.FAPService.{i}.*` (SF/RAN/RF/CellConfig)
- chapter: SF · perm: 📖📝 (R=2/RW=22) · arity=1（i=1~3 固定枚举）
- 命令: **LST FAPService** (24) / **MOD FAPService** (22) / ADD/RMV 自动生成（⚠ 载波数量固定 1~3，运行时可能不应执行 ADD/RMV，见 D31）
- 路径: 📝 CellBarred · boolean · 📝 AdminState · boolean · 📖 OpState · boolean · 📖 SupportRRCNumbers · int[-1:] · 📝 MultiBandInfoListSIB1 · unsignedInt[1:64] · 📝 MultiBandInfoListSIB5 · unsignedInt[1:64] · 📝 RouteIndexList · string(512) · 📝 RuList · string(512) · 📝 UserLabel · string · 📝 EARFCNDL · string(128) · 📝 PhyCellID · string(512) · 📝 DLBandwidth · string(32) · 📝 ULBandwidth · string(32) · 📝 PSCHPowerOffset · string(512) · 📝 SSCHPowerOffset · string(512) · 📝 PBCHPowerOffset · string(512) · 📝 EARFCNUL · string(128) · 📝 FreqBandIndicator · int[1:40] · 📝 ReferenceSignalPower · string(512) · 📝 CellIdentity · unsignedInt[0:268435455] · 📝 EnbType · unsignedInt[0:1] · 📝 SPSSwitchQCI1Ul · string(64) · 📝 CASwitchUl · string(64) · 📝 CASwitchDl · string(64)

### G-18. FAPService Capabilities · `Device.Services.FAPService.{i}.Capabilities.*`
- chapter: SF · perm: 📝 (RW=1) · arity=1
- 命令: **LST Capabilities** (1) / **MOD Capabilities** (1) / ADD/RMV(⚠)
- 路径: 📝 NNSFSupported · boolean

### G-19. CellConfig Capabilities · `Device.Services.FAPService.{i}.CellConfig.Capabilities.*`
- chapter: SF · perm: 📖📝 (R=2/RW=1) · arity=1
- 命令: **LST CellCap** (3) / **MOD CellCap** (1) / ADD/RMV(⚠)
- 路径: 📝 UeInactiveTimer · unsignedInt[0:65535] · 📖 SupportActiveRRCNumbers · unsignedInt[0:65535] · 📖 MaxTxPower · unsignedInt

### G-20. EPC · `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.*`
- chapter: SF · perm: 📝 (RW=2) · arity=1
- 命令: **LST EPC** (2) / **MOD EPC** (2) / ADD/RMV(⚠)
- 路径: 📝 EAID · unsignedInt[0:16777216] · 📝 TAC · unsignedInt[0:65535]

### G-21. PLMNList · `Device.Services.FAPService.{iα}.CellConfig.LTE.EPC.PLMNList.{iβ}.*`
- chapter: SF · perm: 📝 (RW=2) · arity=2
- 命令: **LST PLMNList** (2) / **MOD PLMNList** (2) / **ADD PLMNList** / **RMV PLMNList**
- 路径: 📝 PLMNID · string(6) · 📝 CellReservedForOperatorUse · boolean

### G-22. VoLTE PDCP 初始 · `Device.Services.FAPService.{iα}.CellConfig.LTE.VoLTE.PdcpInitParam.{iβ}.*`
- chapter: SF · perm: 📝 (RW=1) · arity=2
- 命令: **LST PdcpInit** (1) / **MOD PdcpInit** (1) / **ADD PdcpInit** / **RMV PdcpInit**
- 路径: 📝 RohcEn · string(64)

### G-23. SCTP · `Device.Services.FAPControl.Transport.SCTP.*`
- chapter: SG · perm: 📝 (RW=9) · arity=0
- 命令: **LST SCTP** (9) / **MOD SCTP** (9)
- 路径: 📝 Enable · Boolean · 📝 RTOInitial · unsignedInt · 📝 RTOMin · unsignedInt · 📝 RTOMax · unsignedInt · 📝 MaxInitRetransmits · unsignedInt · 📝 HBInterval · unsignedInt[1:] · 📝 MaxPathRetransmits · unsignedInt · 📝 MaxAssociationRetransmits · unsignedInt · 📝 ValCookieLife · unsignedInt

### G-24. SCTP Assoc · `Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.*`
- chapter: SG · perm: 📖 (R=4) · arity=1
- 命令: **LST SCTP Assoc** (4)
- 路径: 📖 SCTPAssocLocalAddr · string(64) · 📖 LocalPort · unsignedInt · 📖 PrimaryPeerAddress · string(64) · 📖 RemotePort · unsignedInt

### G-25. RRC Timers · `Device.Services.FAPService.{i}.*` (SH/RRCTimers)
- chapter: SH · perm: 📝 (RW=10) · arity=1（i=1~3 固定）
- 命令: **LST RRC Timers** (10) / **MOD RRC Timers** (10) / ADD/RMV(⚠)
- 路径: 📝 T300 · string(64) · 📝 T301 · string(64) · 📝 T302 · unsignedInt[1:16] · 📝 T304EUTRA · string(64) · 📝 T304IRAT · string(64) · 📝 T310 · unsignedInt[...] · 📝 T311 · string(64) · 📝 T320 · unsignedInt[5,10,20,30,60,120,180] · 📝 N310 · unsignedInt[1:4,6,8,10,20] · 📝 N311 · unsignedInt[1:6,8,10]

### G-26. MAC · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.*`
- chapter: SH · perm: 📝 (RW=16) · arity=1
- 命令: **LST MAC** (16) / **MOD MAC** (16) / ADD/RMV(⚠)
- 路径: 📝 RACH.NumberOfRaPreambles · string(64) · 📝 RACH.SizeOfRaGroupA · string(64) · 📝 RACH.MessageSizeGroupA · string(32) · 📝 RACH.MessagePowerOffsetGroupB · string(32) · 📝 RACH.PowerRampingStep · string(16) · 📝 RACH.PreambleInitialReceivedTargetPower · string(128) · 📝 RACH.PreambleTransMax · string(64) · 📝 RACH.ResponseWindowSize · string(32) · 📝 RACH.ContentionResolutionTimer · string(32) · 📝 RACH.MaxHARQMsg3Tx · string(32) · 📝 DRX.DRXEnabled · Boolean · 📝 ULSCH.MaxHARQTx · unsignedInt[...] · 📝 ULSCH.PeriodicBSRTimer · unsignedInt[...] · 📝 ULSCH.RetxBSRTimer · unsignedInt[320,640,1280,2560,5120,10240] · 📝 ULSCH.TTIBundling · Boolean · 📝 ULSCH.MaxUePerUlSf · unsignedInt[0:65535]

### G-27. DRX 初始 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{iβ}.*`
- chapter: SH · perm: 📝 (RW=6) · arity=2
- 命令: **LST DRX 初始** (6) / **MOD DRX 初始** (6) / **ADD** / **RMV**
- 路径: 📝 DRXShortCycleTimer · string(64) · 📝 ONDurationTimer · string(64) · 📝 DRXInactivityTimer · string(64) · 📝 DRXRetransmissionTimer · string(32) · 📝 LongDRXCycle · string(128) · 📝 ShortDRXCycle · string(64)

### G-28. PHY · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.*`
- chapter: SH · perm: 📖📝 (R=2/RW=33) · arity=1
- 命令: **LST PHY** (35) / **MOD PHY** (33) / ADD/RMV(⚠)
- 路径: 📝 PRACH.ConfigurationIndex · string(256) · 📝 PRACH.FreqOffset · string(256) · 📝 PRACH.HighSpeedFlag · boolean · 📝 PRACH.RootSequenceIndex · string(512) · 📝 PRACH.ZeroCorrelationZoneConfig · string(64) · 📝 SRS.SRSEnabled · Boolean · 📝 SRS.SRSBandwidthConfig · string(32) · 📝 SRS.SRSMaxUpPTS · Boolean · 📝 SRS.AckNackSRSSimultaneousTransmission · Boolean · 📝 PUCCH.DeltaPUCCHShift · string · 📝 PUCCH.NRBCQI · string · 📝 PUCCH.NCSAN · unsignedInt[0:7] · 📝 PUCCH.N1PUCCHAN · string(512) · 📝 PUCCH.CQIPUCCHResourceIndex · string(512) · 📝 PUCCH.K · unsignedInt[1:4] · 📝 PaParam.PUSCHPowerCtrlSwitch · unsignedInt[0:1] · 📝 PaParam.PUCCHPowerCtrlSwitch · unsignedInt[0:1] · 📝 PUSCH.Enable64QAM · Boolean · 📝 PUSCH.HoppingMode · string · 📝 PUSCH.HoppingOffset · string(256) · 📝 PUSCH.NSB · unsignedInt[1:4] · 📝 PRS.NumPRSResourceBlocks · unsignedInt · 📝 PRS.PRSConfigurationIndex · unsignedInt[0:4095] · 📝 PRS.NumConsecutivePRSSubfames · unsignedInt[1,2,4,6] · 📝 TDDFrame.SpecialSubframePatterns · unsignedInt[0:8] · 📝 TDDFrame.SubFrameAssignment · unsignedInt[0:6] · 📝 PDSCH.Pb · string(32) · 📝 PDSCH.Pa · string(64) · 📝 ULPowerControl.P0NominalPUSCHPersistent · Int[-126:24] · 📝 ULPowerControl.P0NominalPUSCH · string(512) · 📝 ULPowerControl.Alpha · string(64) · 📝 ULPowerControl.P0NominalPUCCH · string(512) · 📝 ULPowerControl.DeltaMCSEnabled · unsignedInt[0:1] · 📖 Antenna.NumOfTxAntenna · unsignedInt[1:7] · 📖 Antenna.NumOfRxAntenna · unsignedInt[1:7]

### G-29. PHY MBSFN · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.*`
- chapter: SH · perm: 📝 (RW=1) · arity=1
- 命令: **LST PHY MBSFN** (1) / **MOD PHY MBSFN** (1) / ADD/RMV(⚠)
- 路径: 📝 NeighCellConfig · unsignedInt[0:3]

### G-30. MBSFN SFConfigList · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{iβ}.*`
- chapter: SH · perm: 📝 (RW=5) · arity=2
- 命令: **LST SFConfig** (5) / **MOD SFConfig** (5) / **ADD** / **RMV**
- 路径: 📝 RadioframeAllocationOffset · unsignedInt[0:6] · 📝 RadioFrameAllocationPeriod · unsignedInt[0:8] · 📝 RadioFrameAllocationSize · unsignedInt[1,4] · 📝 SubFrameAllocations · string(64) · 📝 SyncStratumID · unsignedInt[1:8]

### G-31. GSM 邻区 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{iβ}.*`
- chapter: SI · perm: 📝 (RW=7) · arity=2
- 命令: **LST GSM 邻区** (7) / **MOD GSM 邻区** (7) / **ADD** / **RMV**
- 路径: 📝 PLMNID · string(6) · 📝 LAC · unsignedInt[0:65535] · 📝 BSIC · unsignedInt[0:255] · 📝 CI · unsignedInt[0:65535] · 📝 BandIndicator · string · 📝 BCCHARFCN · unsignedInt[0:1023] · 📝 RAC · unsignedInt[0:255]

### G-32. NR 邻区 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{iβ}.*`
- chapter: SI · perm: 📝 (RW=12) · arity=2
- 命令: **LST NR 邻区** (12) / **MOD NR 邻区** (12) / **ADD** / **RMV**
- 路径: 📝 PLMNID · string(6) · 📝 CID · unsignedLong[1:68719476735] · 📝 GnbIdLen · unsignedInt[22:32] · 📝 SsbFrequency · unsignedInt[0:3279165] · 📝 SsbPeriodicity · unsignedInt[5,10,20,40,80,160] · 📝 SsbOffset · unsignedInt[0..159] · 📝 Ssb_Duration · unsignedInt[1,2,3,4,5] · 📝 PhyCellID · unsignedInt[0:1007] · 📝 TAC · unsignedInt[0:16777215] · 📝 Qoffset · int[...] · 📝 NRband · unsignedInt[1:1024] · 📝 NeighType · unsignedInt[0,1,2]

### G-33. UMTS 邻区 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{iβ}.*`
- chapter: SI · perm: 📝 (RW=10) · arity=2
- 命令: **LST UMTS 邻区** (10) / **MOD UMTS 邻区** (10) / **ADD** / **RMV**
- 路径: 📝 PLMNID · string(6) · 📝 RNCID · unsignedInt[0:65535] · 📝 CID · unsignedInt[0:65535] · 📝 LAC · unsignedInt[0:65535] · 📝 RAC · unsignedInt[0:255] · 📝 URA · unsignedInt[0:65535] · 📝 UARFCNUL · unsignedInt[0:16383] · 📝 UARFCNDL · unsignedInt[0:16383] · 📝 PCPICHScramblingCode · unsignedInt[0:511] · 📝 PCPICHTxPower · int[-100:500]

### G-34. LTE 邻区 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.LTECell.{iβ}.*`
- chapter: SI · perm: 📝 (RW=10) · arity=2
- 命令: **LST LTE 邻区** (10) / **MOD LTE 邻区** (10) / **ADD** / **RMV**
- 路径: 📝 PLMNID · string(6) · 📝 CID · unsignedInt[1:268435455] · 📝 EUTRACarrierARFCN · unsignedInt[0:65535] · 📝 PhyCellID · unsignedInt[0:503] · 📝 QOffset · int[...] · 📝 CIO · int[...] · 📝 RSTxPower · int[-60:50] · 📝 Blacklisted · boolean · 📝 TAC · unsignedInt[0:65535] · 📝 EnbType · unsignedInt

### G-35. ConnMode EUTRA · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.*`
- chapter: SJ · perm: 📝 (RW=1) · arity=1
- 命令: **LST EUTRA** (1) / **MOD EUTRA** (1) / ADD/RMV(⚠)
- 路径: 📝 MeasureCtrl.Smeasure · unsignedInt[0:97]

### G-36. A1 测量控制 · `...A1MeasureCtrl.{iβ}.*`
- chapter: SJ · perm: 📖📝 (R=1/RW=10) · arity=2
- 命令: **LST A1** (11) / **MOD A1** (10) / **ADD A1** / **RMV A1**
- 路径: 📝 Enable · boolean · 📝 A1ThresholdRSRP · unsignedInt[0:97] · 📝 A1ThresholdRSRQ · unsignedInt[0:34] · 📝 Hysteresis · unsignedInt[0:30] · 📝 MaxReportCells · unsignedInt[1:8] · 📖 MeasurePurpose · unsignedInt[1:100] · 📝 ReportAmount · unsignedInt[0:2,4,8,16,32,64] · 📝 ReportInterval · unsignedInt[120,240,480,...] · 📝 ReportQuantity · string · 📝 TimeToTrigger · unsignedInt[0,40,64,80,...] · 📝 TriggerQuantity · string

### G-37. A2 测量控制 · `...A2MeasureCtrl.{iβ}.*`
- chapter: SJ · perm: 📖📝 (R=1/RW=10) · arity=2
- 命令: **LST A2** (11) / **MOD A2** (10) / **ADD A2** / **RMV A2**
- 路径: 📝 Enable · boolean · 📝 A2ThresholdRSRP · unsignedInt[0:97] · 📝 A2ThresholdRSRQ · unsignedInt[0:34] · 📝 Hysteresis · unsignedInt[0:30] · 📝 MaxReportCells · unsignedInt[1:8] · 📖 MeasurePurpose · unsignedInt[1:100] · 📝 ReportAmount · unsignedInt[...] · 📝 ReportInterval · unsignedInt[...] · 📝 ReportQuantity · string · 📝 TimeToTrigger · unsignedInt[...] · 📝 TriggerQuantity · string

### G-38. A3 测量控制 · `...A3MeasureCtrl.{iβ}.*`
- chapter: SJ · perm: 📖📝 (R=1/RW=10) · arity=2
- 命令: **LST A3** (11) / **MOD A3** (10) / **ADD A3** / **RMV A3**
- 路径: 📝 Enable · boolean · 📝 A3Offset · int[-30:30] · 📝 ReportOnLeave · boolean · 📝 Hysteresis · unsignedInt[0:30] · 📝 MaxReportCells · unsignedInt[1:8] · 📖 MeasurePurpose · unsignedInt[1:100] · 📝 ReportAmount · unsignedInt[...] · 📝 ReportInterval · unsignedInt[...] · 📝 ReportQuantity · string · 📝 TimeToTrigger · unsignedInt[...] · 📝 TriggerQuantity · string

### G-39. A4 测量控制 · `...A4MeasureCtrl.{iβ}.*`
- chapter: SJ · perm: 📖📝 (R=1/RW=10) · arity=2
- 命令: **LST A4** (11) / **MOD A4** (10) / **ADD A4** / **RMV A4**
- 路径: 📝 Enable · boolean · 📝 A4ThresholdRSRP · unsignedInt[0:97] · 📝 A4ThresholdRSRQ · unsignedInt[0:34] · 📝 Hysteresis · unsignedInt[0:34] · 📝 MaxReportCells · unsignedInt[1:8] · 📖 MeasurePurpose · unsignedInt[1:100] · 📝 ReportAmount · unsignedInt[...] · 📝 ReportInterval · unsignedInt[...] · 📝 ReportQuantity · string · 📝 TimeToTrigger · unsignedInt[...] · 📝 TriggerQuantity · string

### G-40. A5 测量控制 · `...A5MeasureCtrl.{iβ}.*`
- chapter: SJ · perm: 📖📝 (R=1/RW=12) · arity=2
- 命令: **LST A5** (13) / **MOD A5** (12) / **ADD A5** / **RMV A5**
- 路径: 📝 Enable · boolean · 📝 A5Threshold1RSRP · unsignedInt[0:97] · 📝 A5Threshold1RSRQ · unsignedInt[0:34] · 📝 A5Threshold2RSRP · unsignedInt[0:97] · 📝 A5Threshold2RSRQ · unsignedInt[0:34] · 📝 Hysteresis · unsignedInt[0:30] · 📝 MaxReportCells · unsignedInt[1:8] · 📖 MeasurePurpose · unsignedInt[1:100] · 📝 ReportAmount · unsignedInt[...] · 📝 ReportInterval · unsignedInt[...] · 📝 ReportQuantity · string · 📝 TimeToTrigger · unsignedInt[...] · 📝 TriggerQuantity · string

### G-41. 周期测量控制 · `...PeriodMeasCtrl.{iβ}.*`
- chapter: SJ · perm: 📝 (RW=4) · arity=2
- 命令: **LST PeriodMeas** (4) / **MOD PeriodMeas** (4) / **ADD** / **RMV**
- 路径: 📝 MeasurePurpose · unsignedInt[1:7] · 📝 MaxReportCells · unsignedInt[1:65535] · 📝 ReportInterval · unsignedInt[...] · 📝 ReportAmount · unsignedInt[...]

### G-42. ConnMode IRAT · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.*`
- chapter: SJ · perm: 📝 (RW=4) · arity=1
- 命令: **LST IRAT** (4) / **MOD IRAT** (4) / ADD/RMV(⚠)
- 路径: 📝 QoffsetGERAN · string(128) · 📝 MeasQuantityUTRAFDD · string · 📝 MeasQuantityGERAN · string · 📝 QoffsetUTRA · string(128)

### G-43. B1 测量控制 · `...IRAT.B1MeasureCtrl.{iβ}.*`
- chapter: SJ · perm: 📖📝 (R=1/RW=10) · arity=2
- 命令: **LST B1** (11) / **MOD B1** (10) / **ADD B1** / **RMV B1**
- 路径: 📝 Enable · boolean · 📝 B1ThresholdCDMA2000 · int[-5:91] · 📝 B1ThresholdGERAN · unsignedInt[0:63] · 📝 B1ThresholdUTRAEcN0 · unsignedInt[0:49] · 📝 B1ThresholdUTRARSCP · int[-5:91] · 📝 Hysteresis · unsignedInt[0:30] · 📝 MaxReportCells · unsignedInt[1:8] · 📖 MeasurePurpose · unsignedInt[1:100] · 📝 ReportAmount · unsignedInt[...] · 📝 ReportInterval · unsignedInt[...] · 📝 TimeToTrigger · unsignedInt[...]

### G-44. B2 测量控制 · `...IRAT.B2MeasureCtrl.{iβ}.*`
- chapter: SJ · perm: 📖📝 (R=1/RW=12) · arity=2
- 命令: **LST B2** (13) / **MOD B2** (12) / **ADD B2** / **RMV B2**
- 路径: 📝 Enable · boolean · 📝 B2Threshold1EutraRSRP · unsignedInt[0:97] · 📝 B2Threshold1EutraRSRQ · unsignedInt[0:34] · 📝 B2Threshold2CDMA2000 · unsignedInt[0:63] · 📝 B2Threshold2GERAN · unsignedInt[0:63] · 📝 B2Threshold2UTRAEcN0 · unsignedInt[0:49] · 📝 B2Threshold2UTRARSCP · int[-5:91] · 📝 Hysteresis · unsignedInt[0:30] · 📝 MaxReportCells · unsignedInt[1:8] · 📖 MeasurePurpose · unsignedInt[1:100] · 📝 ReportAmount · unsignedInt[...] · 📝 ReportInterval · unsignedInt[...] · 📝 TimeToTrigger · unsignedInt[...]

### G-45. IdleMode · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.*`
- chapter: SJ · perm: 📝 (RW=28) · arity=1
- 命令: **LST IdleMode** (28) / **MOD IdleMode** (28) / ADD/RMV(⚠)
- 路径: 📝 Common.Qhyst · string(64) · 📝 Common.IntraFreqReselection · unsignedInt · 📝 Common.QHystSFMedium · int[-6,-4,-2,0] · 📝 Common.QHystSFHigh · int[-6,-4,-2,0] · 📝 Common.TEvaluation · unsignedInt[...] · 📝 Common.THystNormal · unsignedInt[...] · 📝 Common.NCellChangeMedium · unsignedInt[1:16] · 📝 Common.NCellChangeHigh · unsignedInt[1:16] · 📝 IntraFreq.QRxLevMinSIB1 · int[-70:-22] · 📝 IntraFreq.QRxLevMinSIB3 · string(256) · 📝 IntraFreq.QRxLevMinOffset · unsignedInt[1:8] · 📝 IntraFreq.SIntraSearch · string(128) · 📝 IntraFreq.TReselectionEUTRA · string(32) · 📝 IntraFreq.SNonIntraSearch · string(128) · 📝 IntraFreq.SNonIntraSearchPR9 · unsignedInt[0:31] · 📝 IntraFreq.SNonIntraSearchQR9 · unsignedInt[0:31] · 📝 IntraFreq.CellReselectionPriority · unsignedInt[0:7] · 📝 IntraFreq.PMax · int[-30:33] · 📝 IntraFreq.ThreshServingLow · unsignedInt[0:31] · 📝 IntraFreq.ThreshServingLowQR9 · unsignedInt[0:31] · 📝 IntraFreq.TReselectionEUTRASFMedium · unsignedInt[25,50,75,100] · 📝 IntraFreq.TReselectionEUTRASFHigh · unsignedInt[25,50,75,100] · 📝 IntraFreq.SIntraSearchPR9 · unsignedInt[0:31] · 📝 IntraFreq.SIntraSearchQR9 · unsignedInt[0:31] · 📝 IntraFreq.QQualMinR9Reselection · int[-34:-3] · 📝 IntraFreq.QQualMinR9Selection · int[-34:-3] · 📝 IntraFreq.QQualMinOffsetR9 · int[1:8] · 📝 IntraFreq.AllowedMeasBandwidth · unsignedInt(6,15,25,50,75,100)

### G-46. IdleMode IRAT · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.*`
- chapter: SJ · perm: 📝 (RW=2) · arity=1
- 命令: **LST IdleMode IRAT** (2) / **MOD IdleMode IRAT** (2) / ADD/RMV(⚠)
- 路径: 📝 UTRA.TReselectionUTRA · string(32) · 📝 GERAN.TReselectionGERAN · string(32)

### G-47. GERAN 频组 · `...IdleMode.IRAT.GERAN.GERANFreqGroup.{iβ}.*`
- chapter: SJ · perm: 📝 (RW=6) · arity=2
- 命令: **LST GERAN 频组** (6) / **MOD GERAN 频组** (6) / **ADD** / **RMV**
- 路径: 📝 BCCHARFCN · unsignedInt[0:1023] · 📝 CellReselectionPriority · unsignedInt[0:7] · 📝 QRxLevMin · unsignedInt[0:45] · 📝 ThreshXHigh · unsignedInt[0:31] · 📝 ThreshXLow · unsignedInt[0:31] · 📝 PMaxGERAN · unsignedInt[0:39]

### G-48. UTRA FDD 频点 · `...IdleMode.IRAT.UTRA.UTRANFDDFreq.{iβ}.*`
- chapter: SJ · perm: 📝 (RW=6) · arity=2
- 命令: **LST UTRA FDD 频点** (6) / **MOD UTRA FDD 频点** (6) / **ADD** / **RMV**
- 路径: 📝 UTRACarrierARFCN · unsignedInt[0:16383] · 📝 CellReselectionPriority · unsignedInt[0:7] · 📝 ThreshXHigh · unsignedInt[0:31] · 📝 ThreshXLow · unsignedInt[0:31] · 📝 QRxLevMin · int[-60:-13] · 📝 PMaxUTRA · int[-50:33]

### G-49. 异频载波 · `...IdleMode.InterFreq.Carrier.{iβ}.*`
- chapter: SJ · perm: 📝 (RW=13) · arity=2
- 命令: **LST 异频载波** (13) / **MOD 异频载波** (13) / **ADD** / **RMV**
- 路径: 📝 EUTRACarrierARFCN · unsignedInt[0:65535] · 📝 QRxLevMinSIB5 · string(256) · 📝 QOffsetFreq · string(128) · 📝 TReselectionEUTRA · string(32) · 📝 QQualMinR9Reselection · int[-34:-3] · 📝 CellReselectionPriority · unsignedInt[0:7] · 📝 ThreshXHigh · unsignedInt[0:31] · 📝 ThreshXLow · unsignedInt[0:31] · 📝 PMax · int[-30:33] · 📝 TReselectionEUTRASFMedium · unsignedInt[25,50,75,100] · 📝 TReselectionEUTRASFHigh · unsignedInt[25,50,75,100] · 📝 ThreshXHighQR9 · unsignedInt[0:31] · 📝 ThreshXLowQR9 · unsignedInt[0:31]

### G-50. SON 配置 · `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.*`
- chapter: SK · perm: 📖📝 (R=1/RW=24) · arity=1
- 命令: **LST SON 配置** (25) / **MOD SON 配置** (24) / ADD/RMV(⚠)
- 路径: 📝 SONSysMode · string · 📝 SONWorkMode · string · 📝 PCIOptEnable · boolean · 📝 PCIReconfigWaitTime · unsignedInt · 📝 CandidateARFCNList · string(64) · 📝 CandidatePCIList · string(64) · 📝 ANREnable · boolean · 📝 ANRInterFeqEnable · boolean · 📝 ANRGERANEnable · boolean · 📝 ANRUTRANEnable · boolean · 📝 ARFCNEnable · boolean · 📝 MaxLTENeighbourCellNum · unsignedInt · 📝 MaxUTRANNeighbourCellNum · unsignedInt · 📝 MaxGRANNeighbourCellNum · unsignedInt · 📝 ReSynCellEnable · boolean · 📝 PowerEnable · boolean · 📝 LTESnifferFreqBandList · string(64) · 📝 LTESnifferChannelList · string(64) · 📝 GERANSnifferEnable · boolean · 📝 GERANSnifferChannelList · string(256) · 📝 UTRANSnifferEnable · boolean · 📝 UTRANSnifferChannelList · string(256) · 📝 MROEnable · boolean · 📝 SHEnable · boolean · 📖 SyncMode · string(64)

### G-51. 自配置启动 · `Device.Services.FAPService.{i}.FAPControl.SelfConfig.*`
- chapter: SK · perm: 📖 (R=3) · arity=1
- 命令: **LST 自配置启动** (3)
- 路径: 📖 Startup.Stage · unsignedInt[1:3] · 📖 Startup.Status · unsignedInt[1:2] · 📖 Startup.FailureCause · string(512)

### G-52. 以太网接口 · `Device.Ethernet.Interface.{i}.*`
- chapter: SL · perm: 📖📝 (R=5/RW=4) · arity=1
- 命令: **LST 以太网接口** (9) / **MOD 以太网接口** (4) / **ADD** / **RMV**
- 路径: 📝 Enable · boolean · 📝 UserLabel · string · 📖 Name · string · 📖 Status · string · 📖 MACAddress · string(17) · 📝 MaxBitRate · string · 📖 SignTransMedia · string · 📝 DuplexMode · string · 📖 PortLocation · string

### G-53. IPv4 地址 · `Device.Ethernet.Interface.{iα}.IPv4Address.{iβ}.*`
- chapter: SL · perm: 📝 (RW=5) · arity=2
- 命令: **LST IPv4 地址** (5) / **MOD IPv4 地址** (5) / **ADD** / **RMV**
- 路径: 📝 IPAddress · string(15) · 📝 DefaultGateway · string(15) · 📝 SubnetMask · string · 📝 AddressingType · string · 📝 PortType · string

### G-54. IPv6 地址 · `Device.Ethernet.Interface.{iα}.IPv6Address.{iβ}.*`
- chapter: SL · perm: 📝 (RW=5) · arity=2
- 命令: **LST IPv6 地址** (5) / **MOD IPv6 地址** (5) / **ADD** / **RMV**
- 路径: 📝 IPAddress · string(64) · 📝 PrefixLength · unsignedInt[1:128] · 📝 Origin · string · 📝 PortType · string · 📝 DefaultGateway · string(64)

### G-55. VLAN 接口 · `Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.*`
- chapter: SL · perm: 📝 (RW=3) · arity=2
- 命令: **LST VLAN 接口** (3) / **MOD VLAN 接口** (3) / **ADD** / **RMV**
- 路径: 📝 Name · string · 📝 Id · interger · 📝 Enable · boolean

### G-56. VLAN IPv4 · `Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv4Address.{iγ}.*`
- chapter: SL · perm: 📝 (RW=5) · arity=3
- 命令: **LST VLAN IPv4** (5) / **MOD VLAN IPv4** (5) / **ADD** / **RMV**
- 路径: 📝 IPAddress · string(15) · 📝 SubnetMask · string · 📝 AddressingType · string · 📝 DefaultGateway · string(15) · 📝 PortType · string

### G-57. VLAN IPv6 · `Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv6Address.{iγ}.*`
- chapter: SL · perm: 📝 (RW=5) · arity=3
- 命令: **LST VLAN IPv6** (5) / **MOD VLAN IPv6** (5) / **ADD** / **RMV**
- 路径: 📝 IPAddress · string(64) · 📝 PrefixLength · unsignedInt[1:128] · 📝 Origin · string · 📝 DefaultGateway · string(64) · 📝 PortType · string

### G-58. IP 路由 · `Device.Ethernet.IpRoute.{i}.*`
- chapter: SL · perm: 📝 (RW=5) · arity=1
- 命令: **LST IP 路由** (5) / **MOD IP 路由** (5) / **ADD** / **RMV**
- 路径: 📝 IpVer · unsignedInt[1:2] · 📝 DstIpNetwork · string(64) · 📝 PrefixLength · unsignedInt[0:128] · 📝 GatewayIpAddress · string(64) · 📝 InterfaceName · string

### G-59. IPsec · `Device.IPsec.*`
- chapter: SM · perm: 📖📝 (R=7/RW=2) · arity=0
- 命令: **LST IPsec** (9) / **MOD IPsec** (2)
- 路径: 📝 Enable · boolean · 📝 MyKeyMode · string · 📖 Status · string · 📖 AHSupported · boolean · 📖 IKEv2SupportedEncryptionAlgorithms · string · 📖 ESPSupportedEncryptionAlgorithms · string · 📖 IKEv2SupportedPseudoRandomFunctions · string · 📖 SupportedIntegrityAlgorithms · string · 📖 SupportedDiffieHellmanGroupTransforms · string

### G-60. 时间服务器 · `Device.Time.*`
- chapter: SN · perm: 📖📝 (R=1/RW=7) · arity=0
- 命令: **LST 时间服务器** (8) / **MOD 时间服务器** (7)
- 路径: 📝 Enable · boolean · 📝 NTPServer1 · string(64) · 📝 NTPServer2 · string(64) · 📝 NTPServer3 · string(64) · 📝 NTPServer4 · string(64) · 📝 NTPServer5 · string(64) · 📖 CurrentLocalTime · dateTime · 📝 LocalTimeZone · string(64)

### G-61. GPS · `Device.FAP.GPS.*`
- chapter: SO · perm: 📖 (R=3) · arity=0
- 命令: **LST GPS** (3)
- 路径: 📖 LockedLatitude · int[-90000000:90000000] · 📖 LockedLongitude · int[-180000000:180000000] · 📖 NumberOfSatellites · unsignedInt

### G-62. MR 配置 · `Device.FAP.MRMgmt.Config.{i}.*`
- chapter: SP · perm: 📝 (RW=14) · arity=1
- 命令: **LST MR 配置** (14) / **MOD MR 配置** (14) / **ADD** / **RMV**
- 路径: 📝 MrEnable · boolean · 📝 MrUrl · string(256) · 📝 MrUsername · string(256) · 📝 MrPassword · string(256) · 📝 MeasureType · string · 📝 OmcName · string · 📝 SamplePeriod · unsignedInt · 📝 UploadPeriod · unsignedInt · 📝 SampleBeginTime · dateTime · 📝 SampleEndTime · dateTime · 📝 PrbNum · string · 📝 SubFrameNum · string · 📝 MRECGIList · string · 📝 MeasureItems · string

### G-63. PM 配置 · `Device.FAP.PerfMgmt.Config.{i}.*`
- chapter: SQ · perm: 📝 (RW=10) · arity=1
- 命令: **LST PM 配置** (10) / **MOD PM 配置** (10) / **ADD** / **RMV**
- 路径: 📝 Enable · boolean · 📝 Alias · string(64) · 📝 URL · string(256) · 📝 Username · string(256) · 📝 Password · string(256) · 📝 PeriodicUploadInterval · unsignedInt[1:65535] · 📝 PeriodicUploadTime · dateTime · 📝 ReplenishEnable · boolean · 📝 ReplenishStartTime · dateTime · 📝 ReplenishEndTime · dateTime

### G-64. MU 主机单元 · `Device.DeviceInfo.MU.{i}.*`
- chapter: SR · perm: 📖📝 (R=21/RW=3) · arity=1
- 命令: **LST MU 主机单元** (24) / **MOD MU 主机单元** (3) / ADD/RMV(⚠ 硬件描述，运行时通常不可增删)
- 路径: 📝 UserLabel · string · 📝 DnPrefix · string · 📖 ManufacturerOUI · string(6) · 📖 Manufacturer · string(64) · 📖 ModelName · string(64) · 📖 VendorUnitFamilyType · string · 📖 VendorUnitTypeNumber · string · 📖 SerialNumber · string(64) · 📖 HardwareVersion · string(64) · 📖 SoftwareVersion · string(64) · 📖 HardwarePlatform · string(64) · 📖 AdditionalHardwareVersion · string(64) · 📖 AdditionalSoftwareVersion · string(64) · 📖 ProvisioningCode · string(64) · 📖 ProductClass · string(64) · 📖 Status · unsignedInt[1:3] · 📝 Reboot · boolean · 📖 UpTime · unsignedInt · 📖 FirstUseDate · dateTime · 📖 ClockSource · unsignedInt[1:11] · 📖 DateOfLastService · dateTime · 📖 DateOfManufacture · dateTime · 📖 ManufacturerData · string · 📖 SlotsInformation · string

### G-65. Slot 板卡 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.*`
- chapter: SR · perm: 📖📝 (R=15/RW=1) · arity=2
- 命令: **LST Slot** (16) / **MOD Slot** (1) / ADD/RMV(⚠)
- 路径: 📖 PackPosition · string(64) · 📖 SlotsOccupied · string(64) · 📖 ManufacturerOUI · string(6) · 📖 Manufacturer · string(64) · 📖 ModelName · string(64) · 📖 SerialNumber · string(64) · 📖 HardwareVersion · string(64) · 📖 SoftwareVersion · string(64) · 📖 ProvisioningCode · string(64) · 📖 VendorUnitFamilyType · string(64) · 📖 UpTime · unsignedInt · 📖 DataModelSpecVersion · string · 📖 3GPPSpecVersion · string · 📖 FirstUseDate · dateTime · 📖 Status · unsignedInt[1:3] · 📝 Reboot · boolean

### G-66. EU 扩展单元 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.*`
- chapter: SR · perm: 📖📝 (R=11/RW=2) · arity=3
- 命令: **LST EU** (13) / **MOD EU** (2) / ADD/RMV(⚠)
- 路径: 📝 UserLabel · string · 📖 RouteIndex · string(64) · 📖 ManufacturerOUI · string(6) · 📖 Manufacturer · string(64) · 📖 ModelName · string(64) · 📖 SerialNumber · string(64) · 📖 HardwareVersion · string(64) · 📖 SoftwareVersion · string(64) · 📖 ProvisioningCode · string(64) · 📖 Status · unsignedInt[1:3] · 📖 DLCRCSum · string(64) · 📖 ULCRCSum · string(64) · 📝 Reboot · boolean

### G-67. RU 远端单元 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.*`
- chapter: SR · perm: 📖📝 (R=12/RW=4) · arity=4
- 命令: **LST RU** (16) / **MOD RU** (4) / ADD/RMV(⚠)
- 路径: 📝 UserLabel · string · 📖 VendorUnitFamilyType · string · 📖 VendorUnitTypeNumber · string · 📖 RouteIndex · string(64) · 📖 Status · unsignedInt[1:3] · 📖 ManufacturerOUI · string(6) · 📖 Manufacturer · string(64) · 📖 ModelName · string(64) · 📖 SerialNumber · string(64) · 📖 HardwareVersion · string(64) · 📖 SoftwareVersion · string(64) · 📖 ProvisioningCode · string(64) · 📝 Reboot · boolean · 📝 FrequencyBand · string · 📝 RFTxStatus · boolean · 📖 DateOfManufacture · dateTime

### G-68. RFChannel 射频通道 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.RFChannel.{iε}.*`
- chapter: SR · perm: 📖📝 (R=1/RW=1) · arity=5
- 命令: **LST RFChannel** (2) / **MOD RFChannel** (1) / ADD/RMV(⚠)
- 路径: 📝 TxGain · string · 📖 NoisePwdBm · string

### G-69. RU 升级 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.SwUpgrade.*`
- chapter: SR · perm: 📖 (R=3) · arity=4
- 命令: **LST RU 升级** (3)
- 路径: 📖 Stage · unsignedInt[1:5] · 📖 Status · unsignedInt[1:3] · 📖 FailureCause · string(512)

### G-70. EU 升级 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.SwUpgrade.*`
- chapter: SR · perm: 📖 (R=3) · arity=3
- 命令: **LST EU 升级** (3)
- 路径: 📖 Stage · unsignedInt[1:5] · 📖 Status · unsignedInt[1:3] · 📖 FailureCause · string(512)

### G-71. Slot 升级 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.SwUpgrade.*`
- chapter: SR · perm: 📖 (R=3) · arity=2
- 命令: **LST Slot 升级** (3)
- 路径: 📖 Stage · unsignedInt[1:5] · 📖 FailureCause · string(512) · 📖 Status · unsignedInt[1:3]

### G-72. MU 升级 · `Device.DeviceInfo.MU.{i}.SwUpgrade.*`
- chapter: SR · perm: 📖 (R=3) · arity=1
- 命令: **LST MU 升级** (3)
- 路径: 📖 Stage · unsignedInt[1:5] · 📖 FailureCause · string(512) · 📖 Status · unsignedInt[1:3]

---

## 3. 统计校验

| 章节 | spec 中的 `####` 命令数 | 本清单 group 数 | 备注 |
|------|----------------------|---------------|------|
| SA | 2 | 2 | DeviceInfo / SwUpgrade |
| SB | 1 | 1 | |
| SC | 1 | 1 | |
| SD | 6 | 6 | FaultMgmt + 5 子对象 |
| SE | 1 | 1 | |
| SF | 11 | 11 | FAPControl 系列 + FAPService 子集 |
| SG | 2 | 2 | SCTP / Assoc |
| SH | 6 | 6 | RRC / MAC / DRX / PHY / MBSFN / SFConfigList |
| SI | 4 | 4 | GSM / NR / UMTS / LTE 邻区 |
| SJ | 15 | 15 | ConnMode + IdleMode 系列 |
| SK | 2 | 2 | SON / SelfConfig |
| SL | 7 | 7 | Ethernet 接口 / IPv4 / IPv6 / VLAN 4 子项 / IpRoute |
| SM | 1 | 1 | |
| SN | 1 | 1 | |
| SO | 1 | 1 | |
| SP | 1 | 1 | |
| SQ | 2（含 1 个伪命令 `.*`） | 1 | 伪命令已剔除 |
| SR | 9 | 9 | MU/Slot/EU/RU/RFChannel + 4 SwUpgrade |
| **合计** | **73** | **72** | spec metadata 73；剔除 SQ.* 伪命令 |

**生成命令估算（per-op 展开）**：
- 仅 LST：~28 group × 1 = 28
- LST + MOD：~26 group × 2 = 52
- LST + MOD + ADD + RMV：~18 group × 4 = 72
- 估计总 mml_commands 行数 ≈ **152 行**

**路径总数（重复计算的实例占位符已折叠）**：与 spec metadata `参数总数: 625（去重后唯一模板参数）` 一致。

---

## 4. 与方案 / 规范文档的对照

| 文件 | 作用 |
|------|------|
| `omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md` | **权威源**（人类编辑） |
| `omcgo/datamodels/mml-catalog/cmcc-tdlte-v23.json` | **运行时数据源**（机器读取，本清单的 1:1 等价物，含完整字段） |
| `docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md` | **实施方案** |
| `docs/design/mml-console-cmcc-tdlte-v23-catalog-listing.md` (本文件) | **人类审阅视图** — 用于评审命令树拆分与 ADD/RMV 启发式是否合理 |

> 修改 spec 后：1) 离线工具重新生成 JSON 文件；2) 本清单同步重生成；3) JSON 由 Catalog Loader 启动期幂等加载。三者保持一致由 `sourceDocSha256` 在 CI 兜底。
