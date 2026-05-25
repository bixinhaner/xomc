# 设备详情「基础信息」字段补齐设计文档

| 元数据 | 内容 |
|---|---|
| 版本 | v1.1 |
| 日期 | 2026-05-25 |
| 状态 | Draft（待评审） |
| 范围 | F06 设备详情页 / device + device_info / InfoSyncer / Carrier adapter |
| 关联 | T-XXX（待登记 backlog） |
| 测试设备 | `1202000240194DP0026` BAICELLS / FAP/mBS31001/SC / BaiBLQ_5.0.16.1_1229 |
| 前置分析 | 见对话上文「设备详情页字段空缺分析报告」 |

### 修订历史

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-05-25 | 初稿：基础信息字段补齐方案（IP / MAC / Cell ID 等 LTE 单 cell 字段 + composite assembler 修复 + ModelName 回填）|
| v1.1 | 2026-05-25 | 新增 §12「小区信息整组不显示」（networkType 枚举错位）+ §13「时间字段语义辨析与 mapper 错位」（5 个时间字段全空白）|

---

## 1. 问题陈述

设备详情页 `/device/detail/:sn?tab=basic` 多个字段显示为空（IP 地址 / MAC / Cell ID / TAC / Tx Power / EARFCN / 频段 / Band / EnbId / UE 数 / 锁状态 / GPS 高度 / MME Pool IP / License Code 等）。

**关键观察**（用户提供）：同一设备 `?tab=parameters` 参数树页**可见** `Device.IP.Interface.1.IPv4Address.1.IPAddress = 172.24.224.38` —— 说明 `device_parameters` 表里**已有数据**。

**根因定位**：`device_parameters` 表是真值源（共 **3237 个 path / 3071 个有值**，覆盖 13 个 `param_group`，含 mme_pool、license、antenna、radio、network、device_info 等）。基础信息页空白的本质是「**投影层（list 接口扁平 SQL / InfoSyncer / detail composite assembler）从 `device_parameters` 提取错路径或漏路径**」，而非「上游 ingestion 缺失」。

---

## 2. 数据流（现状）

```
CPE FileType=11 上传完整参数树
        │
        ▼
ACS upload handler → IntersectService → device_parameters 表(3237 path,3071 有值)
        │                                       │
        │                                       ├─ ① list 接口扁平 SQL ──── 不消费 ─→ devices 列(model_name / ip_address 空)
        │                                       │
        │                                       ├─ ② InfoSyncer.SyncFromParameters ──→ device_info 列
        │                                       │       依赖 Carrier.GetInfoParamMapping(LTE/NR) 字典
        │                                       │       关键 path 错位 → mac / transmit_power 空白
        │                                       │
        │                                       └─ ③ DeviceService.GetDeviceDetailComposite
        │                                               AssembleMMEPool / License / Antenna / Cells
        │                                               几处 path 后缀 / 子串匹配错位 → mme_pool 空 ip 等
        ▼
Inform 周期上报 → DeviceService.UpdateFromInform → 部分字段同步到 devices 表
```

---

## 3. 字段映射表（真值清单）

> 测试设备实测：所有"已有"列对应的 path 在 `device_parameters` 中 `parameter_value IS NOT NULL AND <> ''`。

### 3.1 设备主表 `devices` 缺失字段

| UI 字段 | 后端列 | TR-181 / TR-098 真实 path | 实测值 | 当前缺失原因 | 修复点 |
|---|---|---|---|---|---|
| IP 地址 | `devices.ip_address` | `Device.IP.Interface.1.IPv4Address.1.IPAddress` | `172.24.224.38` | UpdateFromInform 仅写 `UDPConnectionRequestAddress`（多数 CPE 不上报）；未从 device_parameters 回填 | **InfoSyncer 新增 ip 同步**；同时把 list 查询 SELECT 加 device_info.ip_address（或直接从 device_parameters 即时查） |
| 设备型号 | `devices.model_name` | （Inform/参数树均**无** ModelName） | — | TR-069 `DeviceId` 不含 ModelName；参数树也没有 | **从 ProductRegistry 装配件回填**：products 表的 `model_name` 列；不靠 CPE 上报 |

### 3.2 `device_info` 缺失字段（投影自 device_parameters）

| UI 字段 | `device_info` 列 | TR-181 / TR-098 真实 path | 实测值 | 当前 CMCC LTE mapping | 修复点 |
|---|---|---|---|---|---|
| MAC 地址 | `mac` | `Device.Ethernet.Interface.MACAddress` | `48:BF:74:0B:BC:31` | `Device.DeviceInfo.X_CMCC_MACAddress`（不存在） | **改 cmcc/adapter.go LTE mapping 增加 Ethernet path** |
| Cell ID | `cell_id` | `Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity` | `654321` | （LTE mapping 中映射为 `eci`,无 cell_id） | **新增映射：CellIdentity → cell_id**（同 path 同时映射到 eci 与 cell_id），或前端只用 eci |
| Tx Power | `transmit_power` | `Device.Services.FAPService.1.Capabilities.MaxTxPower` 或 `…RAN.RF.X_COM_MaxTxPowerExpanded` | `30` / `30` | `RAN.RF.ReferenceSignalPower`（CPE 不上报） | **改 LTE mapping 用 MaxTxPower** |
| 频段 (Band) | （需新增列）`band` | `Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.FreqBandIndicator` | `48` | 列与映射均无 | **device_info 加 band 列 + 映射** |
| TAC | （需新增列）`tac` | `Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC` | `2` | 列与映射均无 | **device_info 加 tac 列 + 映射** |
| DL EARFCN | `freq_point`（已用） | `EARFCNDL` | `55340` | ✅ 已映射 | — |
| UL EARFCN | （需新增列）`ul_earfcn` | `EARFCNUL` | `55341` | 列与映射均无 | **device_info 加 ul_earfcn 列 + 映射** |
| 卫星数 | （需新增列）`gps_satellites` | `Device.FAP.GPS.NumberOfSatellites` 或 `Device.DeviceInfo.X_COM_GPS_Satellite_count` | `18` / `21` | 列与映射均无 | **device_info 加 gps_satellites + 映射** |
| GPS 高度 | `height`（已存在，用于安装高度） | `Device.FAP.GPS.altidute` (注意拼写) / `Device.FAP.GPS.Height` | `170` | 当前 height 走「手填安装高度」语义 | **GPS 高度建议独立列 `gps_height`,避免与手填 height 混淆** |
| 纬度 | `latitude` | `Device.FAP.GPS.LockedLatitude` (units = 度 × 1e6) | `25924100` → `25.924100°` | 当前列在 devices 表,非指针 float64,DB null → JSON 0 | **改 DTO 为 \*float64;新增映射并按 1e6 缩放** |
| 经度 | `longitude` | `Device.FAP.GPS.LockedLongitude` | `115366000` → `115.366000°` | 同上 | 同上 |
| eNodeB ID | （需新增列）`enb_id` | （从 ECI 推导：`ECI >> 8`,LTE 标准）或 `Device.Services.FAPService.1.X_…EnbId` | `654321 >> 8` | 列与映射均无 | **新增列 + 计算逻辑（SyncFromParameters 派生）** |
| Lock 状态 | （需新增列）`lock_status` | `Device.Services.FAPService.1.FAPControl.LTE.AdminState` | `false` | 列与映射均无 | **新增列 + 映射** |
| AdminState | `admin_state` | 同上 | `false` | 已可走快速设置接口；详情页未取 | 前端在 basic Tab 显示同一字段 |
| HaloB | （需新增列）`halob_flag` | 无标准 path,需 vendor `X_COM_*` 或 board capability | — | — | **暂搁置：缺真值源** |
| GPS 版本 | （需新增列）`gps_version` | 暂未找到（待 CPE 升级补） | — | — | **暂搁置：CPE 不上报** |
| ROM 大小 | （需新增列）`rom` | 暂未找到 | — | — | **暂搁置：CPE 不上报** |
| IPSec 地址 | （需新增列）`ipsec_addr` | `device_parameters` 里 group=ipsec 有 113 个 path,需逐条挑 | 待定 | — | **二期：扫描 ipsec 子树后定义** |
| UE 数量 | （需新增列）`ue_count` | （PM/KPI 类指标,非配置参数）需 KPI Engine 算 | — | — | **二期：等 KPI 接入** |

### 3.3 Composite 接口（`/devices/:id/detail`）assembler 修复

| 区块 | 字段 | 真实 path（实测） | 现有 mapper 行为 | 修复点 |
|---|---|---|---|---|
| `mme_pool[].ip` | (索引 1..16) | `…MmePoolConfigParam.{N}.MMEIp1` | 只匹配 `MME1Address` / `MME1IP` 后缀（实际后缀是 `MMEIp1`） | **detail_assembler.go AssembleMMEPool 增加 `MMEIp1` 分支** |
| `mme_pool[].status` | | `…MmePoolConfigParam.{N}.MME1Status` | ✅ 已匹配（实测 `0` → "inactive"） | — |
| `mme_pool[].plmn_id` | | `…MmePoolConfigParam.{N}.PLMNID` | ✅ 已匹配 | — |
| `license.code` | | （CPE 实际**不上报** LicenseCode） | mapper 找 `LicenseCode` 子串匹配；DB 无对应行 → 永远空 | **改用 `Author` 字段或前端直接不显示 code**；如必须 code,提请固件增加上报 |
| `license.capacities[].state` | | （CPE 不上报 State,只上报 Value） | mapper 找 `State` 后缀匹配 → 永远空 | **mapper 改读 `Value` 字段并 rename 输出字段为 `state` 或新增 `value` 字段** |
| `cells[].rf_tx_status` | | `Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus`（注意位于 FAPControl 而非 RAN.RF） | mapper 找 `RAN.RF.RFTxStatus` → 不存在 | **AssembleCells 改用 FAPControl 路径** |
| `cells[].op_state` | | `Device.Services.FAPService.1.FAPControl.LTE.CellOpState` | mapper 找 `FAPControl.LTE.OpState` → 不存在 | **AssembleCells 改用 CellOpState** |
| `antenna.*` | | `Device.DeviceInfo.AntennaInfo.{Azimuth/Beamwidth/Downtilt/Gain/Height/HeightType}` | ✅ 已匹配（实测均为 0/2/AGL,**这是真实值**,非 bug） | — |

### 3.4 前后端字段名错位（不改后端只改前端）

| 前端期望（camelCase） | 后端实际（snake_case） | 处置 |
|---|---|---|
| `txPower` | `transmit_power` | **mapBackendDevice 修复映射** |
| `macAddress` | `mac` | 同上 |
| `firstOnlineTime` / `onlineTime` / `offlineTime` | `first_online_time` / `last_online_time` / `last_offline_time` | 同上 |
| `gpsHeight` | `gps_height`（新增列）| 同上 |
| `dlEarfcn` / `ulEarfcn` | `freq_point` / `ul_earfcn` | 同上 |
| `tac` / `band` / `enbId` / `lockStatus` / `ueCount` | 见 3.2 新增列 | 新列上线后映射 |

### 3.5 Mapper bug：DB 有值但 API 返回 null

| 字段 | 现象 | 根因（待复核） |
|---|---|---|
| `info.last_online_time` | DB `device_info.last_online_time = 2026-05-24 18:38:15` 实测有值;API 响应 `null` | `device_info_repo.GetByDeviceID` SELECT 列名或 scan 顺序错位（**P1 必查**） |

---

## 4. 设计方案

### 4.1 总体策略

**保持现有职责划分**，只填洞，不重构数据流：
- `device_parameters` 仍是真值源
- `device_info` 仍是「投影/快照层」，承载快速展示与 list 接口扁平化
- list 接口 + composite 接口都从 `device_info` + `device_parameters` 读

### 4.2 改造分层

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Layer A 数据库迁移                                                       │
│   migrations/000NNN_device_info_extend_basic_fields.sql                 │
│     ALTER TABLE device_info ADD COLUMN tac / band / ul_earfcn /         │
│       gps_satellites / gps_height / enb_id / lock_status / ue_count     │
│     ALTER TABLE devices ALTER COLUMN latitude TYPE numeric(10,7)        │
│       (从 double precision 改为带 null 语义的 numeric;DTO 改 *float64)   │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
┌─────────────────────────────────────────────────────────────────────────┐
│ Layer B Go 模型 + Carrier mapping                                        │
│   internal/core/model/device_info.go: 加 *string/*int 字段              │
│   internal/core/carrier/cmcc/adapter.go: GetInfoParamMapping 扩展        │
│     - Device.Ethernet.Interface.MACAddress  → mac                       │
│     - Device.Services.FAPService.1.Capabilities.MaxTxPower → transmit_power │
│     - Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC → tac         │
│     - Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.FreqBandIndicator → band │
│     - Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNUL → ul_earfcn │
│     - Device.FAP.GPS.NumberOfSatellites → gps_satellites                │
│     - Device.FAP.GPS.altidute → gps_height (注意拼写)                    │
│     - Device.FAP.GPS.LockedLatitude → latitude (按 1e6 缩放)            │
│     - Device.FAP.GPS.LockedLongitude → longitude (按 1e6 缩放)          │
│     - Device.IP.Interface.1.IPv4Address.1.IPAddress → ip_address (新写 devices 表) │
│     - Device.Services.FAPService.1.FAPControl.LTE.AdminState → lock_status │
│   ctcc / cucc adapter 同步增加（运营商差异通过 path 不同体现，非 if/else） │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
┌─────────────────────────────────────────────────────────────────────────┐
│ Layer C InfoSyncer 派生字段                                              │
│   internal/device/device_info_sync.go:                                  │
│     - SyncFromParameters 新增「派生计算」hook                            │
│         · enb_id = parseInt(eci) >> 8                                   │
│         · latitude/longitude = raw / 1e6                                │
│     - 把 ip_address 同步到 devices.ip_address（独立路径,不混 device_info）│
└─────────────────────────────────────────────────────────────────────────┘
                                  │
┌─────────────────────────────────────────────────────────────────────────┐
│ Layer D Composite assembler 修复                                         │
│   internal/device/detail_assembler.go:                                  │
│     - AssembleMMEPool: case suffix=="MMEIp1" → ip                       │
│     - AssembleCells.RFTxStatus 改 FAPControl.LTE.RFTxStatus             │
│     - AssembleCells.OpState 改 FAPControl.LTE.CellOpState               │
│     - AssembleLicenseDetail.capacity 改读 Value 字段（rename JSON 输出   │
│         字段为 `value` 而非 `state`,前端 i18n 同步）                     │
│     - 移除找不到的 LicenseCode 检查,改用 Author 或前端隐藏               │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
┌─────────────────────────────────────────────────────────────────────────┐
│ Layer E List 接口扁平 SQL                                                │
│   internal/device/device_repository.go (ListWithInfo / 等价方法):       │
│     SELECT 中加入新列                                                    │
│   internal/device/device_info_dto.go DeviceWithInfo struct 加新字段     │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
┌─────────────────────────────────────────────────────────────────────────┐
│ Layer F 前端                                                             │
│   omcmb/frontend-core/src/types/device.ts                               │
│     BackendDevice / Device 加新字段                                     │
│   omcmb/frontend-core/src/services/api/deviceApi.ts                     │
│     mapBackendDevice 加 snake_case → camelCase 转换                     │
│   omcmb/webcode/src/pages/device/DeviceDetail/index.tsx 或 BasicTab     │
│     绑定新字段显示                                                       │
│   （可选 Phase 2: DeviceDetail 引入 useDeviceDetailComposite,使         │
│    mme_pool / license / antenna / cells 也能展示）                       │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
┌─────────────────────────────────────────────────────────────────────────┐
│ Layer G P1 单独修复（高优先,与本方案解耦）                                │
│   Bug: API 返回 info.last_online_time = null 但 DB 实际有值              │
│   定位: device_info_repo.GetByDeviceID 的 SELECT/Scan/JSON tag          │
│   独立 commit                                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

### 4.3 ModelName 特殊处理

`Device.DeviceInfo.ModelName` 在 TR-069 `DeviceId` 块和参数树中**都不上报**（Baicells 实测）。两条路径二选一：

- **方案 X（推荐）**：通过 ProductRegistry 路由 `product_class` → `products` 表的 `model_name` 列回填到 `devices.model_name`。前端展示时优先用 `devices.model_name`，缺省时降级为 `product_class`。
- **方案 Y**：通过 GPV 主动拉 `Device.DeviceInfo.ModelName`，命中则写库。该字段大多数 CPE 厂商不支持，**期望失败**。

→ 采用方案 X。在 `device_service.go RegisterFromInform / UpdateFromInform` 末尾，若 `model_name == ""` 且 `ProductRegistry.MatchProductClass` 命中，则把 `product.ModelName` 写回 `devices.model_name`。

---

## 5. 实施计划（5 个阶段，可独立 commit）

| 阶段 | 范围 | 预估代码量 | 前置依赖 | 风险 |
|---|---|---|---|---|
| **Phase 1** | Layer G `last_online_time` mapper bug 修复 | <50 行 | 无 | 极低 |
| **Phase 2** | Layer A + B + C (LTE)：迁移 + cmcc adapter + InfoSyncer 派生 | ~250 行 + 1 迁移 | 数据库迁移评审 | 低 — 不动现有列,只新增 |
| **Phase 3** | Layer D：composite assembler 修复 MME / cells / license | ~80 行 + 单测 | 无 | 低 — 影响 `/devices/:id/detail` 接口,但目前前端没调（修了即生效） |
| **Phase 4** | Layer E + F：list 接口字段扩展 + 前端映射 | ~150 行后端 + ~150 行前端 | Phase 2 落库 | 中 — 前端类型变更跨三个皮肤包 |
| **Phase 5** | ModelName 回填（方案 X） | ~80 行 | Phase 2 | 低 |

### 时间预算

| 阶段 | 估时（人日） |
|---|---|
| Phase 1 | 0.5 |
| Phase 2 | 1.5 |
| Phase 3 | 1.0 |
| Phase 4 | 1.5 |
| Phase 5 | 0.5 |
| **合计** | **5 人日** |

---

## 6. 风险与缓解

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| 新增 device_info 列后旧设备的列全为 NULL,需要 backfill | 高 | 中 | InfoSyncer 在下次 Inform 时自动回填;另在迁移文件末尾发布 `event.SubjectDeviceParamsBackfill` 让 worker 触发全量回填 |
| Carrier adapter 修改触发 ctcc / cucc 测试用例失败 | 中 | 中 | 每个 adapter 独立改动;先 LTE 后 NR;先 cmcc 后其他 |
| Latitude/Longitude 缩放（1e6）方向错误 | 低 | 高 | 单测 table-driven 覆盖：input `25924100` → output `25.924100°` |
| `Device.FAP.GPS.altidute` 拼写错误是 CPE 上报字面值,后续固件可能修正 | 中 | 低 | InfoSyncer 同时识别 `altidute` 和 `Altitude` 两种拼写,以前者命中,后者覆盖 |
| 前端类型变更跨三个皮肤包破坏编译 | 中 | 中 | 全部新字段标 `?: optional`;`frontend-core` 改完后跑 `webcode / webcode-v2 / webcode-v3` 三个 typecheck |
| `/devices/:id/detail` composite 接口当前没人调,修了没人验 | 中 | 低 | Phase 4 同时为 DeviceDetail 前端引入 `useDeviceDetailComposite` 调用,使 MME Pool / License Tab 真正消费数据 |

---

## 7. 测试方案

### 7.1 单元测试

| 模块 | 用例 |
|---|---|
| `internal/core/carrier/cmcc/adapter_test.go` | LTE/NR mapping 表完整性,关键 path 命中 |
| `internal/device/device_info_sync_test.go` | SyncFromParameters：用 fixture device_parameters 16 条断言 device_info 18 列被正确设置 |
| `internal/device/detail_assembler_test.go` | MMEIp1 / RFTxStatus / CellOpState / License Value 4 个新分支 |
| `internal/device/device_service_test.go` | RegisterFromInform 后 model_name 从 ProductRegistry 回填 |

### 7.2 集成测试 / E2E

- `omcgo/scripts/cpe_simulator.py` 模拟 Baicells CPE 上传 fixture XML（含本设计涉及的所有 path）
- 调 `GET /api/v1/devices?sn=...` 断言新字段值
- 调 `GET /api/v1/devices/:id/detail` 断言 mme_pool / cells / license 真实值

### 7.3 前端 E2E（Playwright）

`omcmb/webcode/tests/e2e/device-detail-basic.spec.ts`（新增）：
- 登录 → 访问 `/device/detail/{seed_sn}?tab=basic`
- 断言 IP / MAC / Cell ID / Tx Power / TAC / 频段 各字段**有值且非默认 0/—**

---

## 8. 不在本方案的项

| 项 | 原因 | 后续 |
|---|---|---|
| `halob_flag / gps_version / rom / ipsec_addr / ue_count` 字段补齐 | 无标准 path / 需 KPI 引擎 / 需扫 ipsec 子树 | 独立 backlog 单 |
| 用 ChannelEventBus 触发 backfill 历史设备 | Phase 2 缓解策略已足够 | 如生产数据量大再做 |
| 前端 DeviceDetail 全面接入 composite 接口（MME Pool Tab / 天线 Tab / License Tab） | 工作量大,且现状未阻塞用户操作 | Phase 4 末尾选做或下个迭代 |
| `mac` 来源选 `Device.Ethernet.Interface.{i}.MACAddress` 多实例处理 | 实测设备只有 Interface（无索引）路径,够用 | 多 MAC 设备出现后再扩 |

---

## 9. 验收标准

- [ ] 测试设备 `1202000240194DP0026` 详情页 basic Tab 显示：
  - IP 地址 `172.24.224.38`（非空）
  - MAC `48:BF:74:0B:BC:31`
  - Cell ID `654321`、PCI `2`、PLMN `46068`
  - Tx Power `30`、带宽 `75`、DL EARFCN `55340`、UL EARFCN `55341`、Band `48`、TAC `2`
  - GPS 卫星数 `18` 或 `21`、GPS 高度 `170`
  - 纬度 `25.924100°`、经度 `115.366000°`
  - 设备型号 显示 `mBS31001` 或 ProductRegistry 装配件名（非空）
  - 锁状态 `false` → 解锁
- [ ] `/devices/:id/detail` 接口 `mme_pool` 数组 index=1 项 ip 为 `172.23.224.88` 而非空
- [ ] `last_online_time` 接口字段与 DB 列值一致
- [ ] 所有 `*_test.go` 单测通过；`scripts/e2e_verify.sh` 不回归
- [ ] 前端 webcode / webcode-v2 / webcode-v3 三个皮肤 `npm run typecheck` 通过

---

## 10. 附录：device_parameters 子树统计（测试设备实测）

| param_group | 行数 | 含值率 |
|---|---|---|
| other | 1332 | 92% |
| radio | 935 | 95% |
| device_info | 343 | 95% |
| fap_control | 215 | 97% |
| ipsec | 113 | 80% |
| license | 90 | 90% |
| network | 72 | 90% |
| mme_pool | 58 | 100% |
| alarm | 40 | 95% |
| antenna | 16 | 100% |
| sync | 14 | 95% |
| management | 8 | 100% |
| software | 1 | 100% |
| **合计** | **3237** | **95%** |

> 数据来源：`SELECT param_group, COUNT(*) FROM device_parameters WHERE device_id='4c15e314-5daf-4b5c-a5a6-e626f3796f58' GROUP BY param_group`

---

## 11. 评审检查表

- [ ] 数据库迁移版本号与本地最大版本号 +1 一致（参 `omcgo/CLAUDE.md §5.5`）
- [ ] Carrier adapter mapping 表更新含 LTE + NR 两个分支
- [ ] InfoSyncer 改动不破坏现有 cmcc / ctcc / cucc 单测
- [ ] 前端字段新增使用 optional `?:` 防止破坏其他皮肤
- [ ] Phase 1 last_online_time 修复在合入主分支前优先做
- [ ] 测试设备验收清单全过
- [ ] PR 描述含本文档路径

---

## 12. 小区信息组整体不显示（v1.1 新增）

### 12.1 现象

`/device/detail/1202000240194DP0026?tab=basic` 页面**完全看不到「小区信息」分组**（含 PCI / TAC / Band / ECI / EARFCN / Tx Power / 帧配比 等 9-13 个字段）。

### 12.2 根因

**`networkType` 枚举值在前后端约定错位**（一处 schema 设计漏洞），导致整组渲染条件永远不命中。

前端 `webcode/src/pages/device/DeviceDetail/index.tsx:252` 的 `getCellFields`：
```ts
if (networkType === 'eNB' || networkType === 'gNB') { /* 推入 LTE+NR 共享字段 */ }
if (networkType === 'eNB') { /* 推入 LTE 专属 enbId/eci/cellId/...  */ }
```

但前端 `deviceApi.ts:180` 的 mapper：
```ts
networkType: bd.technology,    // 后端 technology 是 "lte" / "nr"，不是 "eNB"/"gNB"
```

**`networkType === 'eNB'` 永不命中** → `fields` 数组为空 → 整个分组 `<Descriptions>` 不渲染。

同样的问题也影响：
- 状态信息组的 LTE 专属字段（mmeStatus / pmReportStatus / cpeCount / lockStatus / wanSpeed / serviceStatus / validity）
- 基站信息组的部分字段
- 其他信息组的部分字段

### 12.3 修复方案 —— 二选一

#### 方案 A（推荐）：前端 mapper 做枚举转换

修改 `omcmb/frontend-core/src/services/api/deviceApi.ts` 的 `mapBackendDevice`：
```ts
function toRadioMode(technology: string): string {
  switch (technology) {
    case 'lte': return 'eNB';
    case 'nr': return 'gNB';
    case 'gsm': return 'GSM';
    default: return technology;
  }
}
// ...
networkType: toRadioMode(bd.technology),
```

**优点**：
- 后端 schema 不变，无 DB 迁移
- 前端单点修复（一处函数）
- `Device.technology`（业务层）与 `Device.networkType`（UI 层）解耦
- 反向写时（update / create）保持现有 `bd.technology = data.networkType` 的语义（前端表单值若是 `eNB`/`gNB` 也需要在 toBackend 处反向转）

**缺点**：
- 业务语义重复（technology 与 networkType 两个字段都存在 Device 类型上）

#### 方案 B：后端新增 `network_type` 列直接返回 `eNB / gNB / GSM`

修改 `devices` 表，新增计算列或在 list 接口 SQL 中：
```sql
CASE technology
  WHEN 'lte' THEN 'eNB'
  WHEN 'nr' THEN 'gNB'
  WHEN 'gsm' THEN 'GSM'
END AS network_type
```

**优点**：所有前端皮肤一次性受益
**缺点**：业务字段冗余；多皮肤前端类型需同步加列；不推荐

→ **采用方案 A**。Phase 4 前端改动中合入。

### 12.4 小区信息字段补齐清单（LTE / eNB）

> 测试设备 `1202000240194DP0026` 实测，下表所有 path 在 `device_parameters` 中 `parameter_value` 非空。

| UI 字段 | 前端 key | 真实 path | 实测值 | 投影目标列（device_info） | 现状 |
|---|---|---|---|---|---|
| PCI | `pci` | `…CellConfig.LTE.RAN.RF.PhyCellID` | `2` | `pci` ✅（已映射）| 字段名错位 → 渲染缺失 |
| TAC | `tac` | `…CellConfig.LTE.EPC.TAC` | `2` | `tac`（**新增列**）| 列与映射均无 |
| Band | `band` | `…CellConfig.LTE.RAN.RF.FreqBandIndicator` | `48` | `band`（**新增列**）| 同上 |
| DL EARFCN | `dlEarfcn` | `…CellConfig.LTE.RAN.RF.EARFCNDL` | `55340` | `freq_point` ✅ | 前端 camelCase 错位 |
| UL EARFCN | `ulEarfcn` | `…CellConfig.LTE.RAN.RF.EARFCNUL` | `55341` | `ul_earfcn`（**新增列**）| 列与映射均无 |
| 网络模式 | `networkModel` | （TDD/FDD 由 `…X_COM.FrameStructureMode` 或固定推导）| 暂未确认 | `network_model`（**新增列**）| 待定 |
| Tx Power | `txPower` | `…Capabilities.MaxTxPower` 或 `…RAN.RF.X_COM_MaxTxPowerExpanded` | `30` / `30` | `transmit_power`（已存在）| Carrier mapping path 错（原映射用了 `ReferenceSignalPower`，CPE 不上报）|
| eNodeB ID | `enbId` | （从 ECI 推导：`(ECI >> 8) & 0xFFFFF`，LTE 标准 28-bit ECI = 20-bit eNB-ID + 8-bit CellID）| `654321` >> 8 = `2556` | `enb_id`（**新增列**，InfoSyncer 派生）| 列与映射均无 |
| Cell ID | `cellId` | `…CellConfig.LTE.RAN.Common.CellIdentity` 末 8 位，或单独从 vendor path | `654321` & `0xFF` = `81` | `cell_id` ✅（已存在）| LTE mapping 缺该映射（NR 有）|
| ECI | `eci` | `…CellConfig.LTE.RAN.Common.CellIdentity` 完整值 | `654321` | `eci` ✅ | 字段名错位（前端 camelCase 命中正常）|
| PLMN | `plmnId` | `…CellConfig.LTE.EPC.PLMNList.1.PLMNID` | `46068` | `plmn` ✅ | 前端 key `plmnId`，需对齐 |
| 帧配比 | `subframeAssignment` | `…RAN.PHY.TDDFrame.SubFrameAssignment` | `2` | `subframe_assignment`（**新增列**）| 列与映射均无 |
| 特殊子帧 | `specialSubframe` | `…RAN.PHY.TDDFrame.SpecialSubframePatterns` | `5` | `special_subframe`（**新增列**）| 同上 |
| 根索引 | `rootIndex` | `…RAN.PHY.PRACH.ZeroCorrelationZoneConfig` | `10` | `root_index`（**新增列**）| 同上 |
| Site ID | `siteId` | `devices.site_id`（运维手填）| 空 | `devices.site_id`（已存在）| Type B：手填项，CPE 不上报 |
| 带宽 | `bandwidth` | `…CellConfig.LTE.RAN.RF.DLBandwidth` | `75` | `bandwidth` ✅ | 已映射 |

### 12.5 与 §3.2 / §4.2 的关系

- 涉及的 `device_info` 新增列（`tac` / `band` / `ul_earfcn` / `subframe_assignment` / `special_subframe` / `root_index` / `network_model` / `enb_id`）**合并到 Phase 2 的单次 migration**
- Carrier adapter `GetInfoParamMapping` 同步扩充 → 与 §4.2 Layer B 合并
- 前端 mapper 的字段名对齐 + networkType 枚举转换 → 与 §4.2 Layer F 合并
- **PCI / DL EARFCN / TAC / Band / Tx Power 等字段的 DB 列实际已有值或可投影**，纯属"前端读不到"的对齐问题 —— **修了立即生效**

### 12.6 优先级建议

12.3 方案 A 的修复**单独抽出做 P0 hotfix**：
- 改动量极小（5 行）
- 直接让"小区信息"+"状态信息（LTE 专属）"两组重新可见
- 即便 device_info 新列未上线，已映射字段（PCI / EARFCN / ECI / PLMN / 带宽）立即显示
- Phase 0（hotfix）：networkType 枚举转换 ≈ 0.25 人日

---

## 13. 「其他信息组」时间字段全空白（v1.1 新增）

### 13.1 现象

前端"其他信息组"展示 6 个时间/时长字段，**全部显示为 `-`**（fmtTime/fmtDuration 容错占位符）：

| 前端 label | 前端 key | 设计语义 |
|---|---|---|
| 接入时间 | `onlineTime` | 最近一次上线时间 |
| 断开时间 | `offlineTime` | 最近一次离线时间 |
| 累计时长 | `onlineDuration` | 本次在线已持续多久（单位：秒）|
| 运行时间 | `upTime` | 设备本次开机后运行时长（单位：秒）|
| 首次接入 | `firstOnlineTime` | 设备生命周期内首次上线时间 |
| 最近报告 | `lastInformTime` | 最近一次 TR-069 Inform 触达时间 |

### 13.2 后端真实存储与语义辨析

| 后端字段 | 表 | 业务语义 | 写入路径 | 测试设备实测值 |
|---|---|---|---|---|
| `devices.last_inform_at` | `devices` | 最近一次 Inform 报文到达 ACS 的时间 | `acs.handleInform` 每次 Inform 触发；后台 OfflineDetector 不修改它 | `2026-05-25 13:48:16` ✅ |
| `device_info.first_online_time` | `device_info` | 设备生命周期内**首次** offline→online 转换时间 | `InfoSyncer.RecordOnline()` 内判断 `info.FirstOnlineTime == nil` 时设置 | `2026-05-23 16:01:53` ✅ |
| `device_info.last_online_time` | `device_info` | **最近一次** offline→online 转换时间 | `InfoSyncer.RecordOnline()` 每次都更新 | `2026-05-24 18:38:15` ✅ |
| `device_info.last_offline_time` | `device_info` | **最近一次** online→offline 转换时间 | `InfoSyncer.RecordOffline()` 在 `OfflineDetector` 判定超时时触发 | `2026-05-24 18:37:41` ✅ |
| `device_info.run_time` | `device_info` | 设备本次开机后的运行时长（秒）| `InfoSyncer.SyncFromParameters` 从 `Device.DeviceInfo.UpTime`（标准）或 `X_COM_STATION_RUN_Time`（vendor 兜底）写入 | `947429` 秒 ≈ 10.96 天 ✅ |
| —（无后端列）| — | 本次在线累计时长 | **无写入路径，需要派生计算** | — |

### 13.3 三类问题判定

#### Type ①：前后端字段名错位 → 5/6 字段全空

前端 `deviceApi.ts mapBackendDevice` 读取的字段名**全部错位**：

| 前端 mapper 读 | 后端实际返回 | 现状 |
|---|---|---|
| `bd.online_at` | `last_online_time` | ❌ undefined |
| `bd.offline_at` | `last_offline_time` | ❌ undefined |
| `bd.first_online_at` | `first_online_time` | ❌ undefined |
| `bd.up_time`（字符串）| `run_time`（int64 秒）| ❌ undefined（类型也不一致）|
| `bd.online_duration` | （后端无此列）| ❌ undefined |
| `bd.last_inform_at` | `last_inform_at` | ✅ **唯一对齐** |

**根因猜想**：mapper 是按某个 mock 版本或早期 API 草案写的，从未与最终后端 `DeviceWithInfo` DTO 字段名对齐。

#### Type ②：缺失派生字段 → `onlineDuration` 字段后端从未存在

"累计时长"（onlineDuration）业务语义是"本次在线持续多久"。两种实现路径：

- **派生方案 A（推荐）**：后端 list 接口 SQL 中即时计算
  ```sql
  EXTRACT(EPOCH FROM (
    CASE WHEN is_online THEN NOW() ELSE last_offline_time END
    - last_online_time
  ))::bigint AS online_duration
  ```
  在线时计算到现在，离线后计算到 last_offline_time 截止。
  - **优点**：零存储，单调真实，刷新即变。
  - **缺点**：null 边界要稳处理（`last_online_time IS NULL` → return null）。

- **派生方案 B（次选）**：前端基于 onlineTime/offlineTime/isOnline 自行计算
  ```ts
  onlineDuration: device.isOnline
    ? Math.floor((Date.now() - new Date(device.onlineTime).getTime()) / 1000)
    : Math.floor((new Date(device.offlineTime).getTime() - new Date(device.onlineTime).getTime()) / 1000)
  ```
  - **优点**：完全前端逻辑，零后端工作量
  - **缺点**：客户端时钟漂移会影响显示（多用户看到不同值）

→ 采用方案 A。

#### Type ③：upTime 类型转换 → `run_time` 是秒整数

前端 `fmtDuration` 已支持秒数入参，但 mapper 写的是：
```ts
upTime: bd.up_time || '',   // 期望字符串
```
后端返回 `run_time: 947429`（int），mapper 用 `||` 短路时 `0` 会被当 falsy（但 947429 不是 0，所以这里其实能透传，但类型语义是污染的）。正确做法：
```ts
upTime: typeof bd.run_time === 'number' ? bd.run_time : 0,
```
或直接 `bd.run_time ?? 0`。

### 13.4 各字段业务用途确认（是否有真实用途）

| 字段 | 真实用途 | 必要性 |
|---|---|---|
| 首次接入 (firstOnlineTime) | 设备入网时间审计；生命周期分析 | **必要**，运维强需求 |
| 接入时间 (onlineTime / last_online_time) | 故障恢复时间点定位；与告警关联 | **必要** |
| 断开时间 (offlineTime / last_offline_time) | 故障开始时间点；MTTR 计算基线 | **必要** |
| 累计时长 (onlineDuration) | "设备已在线 X 时长"快速感知；与 SLA 报表挂钩 | **必要**，但派生即可，不必存列 |
| 运行时间 (upTime / run_time) | 区分"在线但近期重启过"——`run_time` 比 `last_online_time` 短意味着 CPE 重启过但 ACS 没收到 BOOT 事件；用于异常重启检测 | **必要** |
| 最近报告 (lastInformTime) | 心跳健康度感知；超时告警判定基线 | **必要**，已工作 |

**结论：6 个字段全部有真实用途，无可裁剪项**。

### 13.5 修复方案

#### Layer B 后端 list 接口扩展（device_repository.go 的 ListWithInfo SQL）

```sql
SELECT
  d.*,
  di.first_online_time,
  di.last_online_time,
  di.last_offline_time,
  di.run_time,
  -- 新增派生列：本次在线/最近在线区间秒数（type ② 修复）
  CASE
    WHEN di.last_online_time IS NULL THEN NULL
    WHEN d.is_online THEN EXTRACT(EPOCH FROM (NOW() - di.last_online_time))::bigint
    WHEN di.last_offline_time IS NOT NULL AND di.last_offline_time > di.last_online_time
      THEN EXTRACT(EPOCH FROM (di.last_offline_time - di.last_online_time))::bigint
    ELSE NULL
  END AS online_duration,
  ...
FROM devices d
LEFT JOIN device_info di ON di.device_id = d.id
```

`DeviceWithInfo` struct 加 `OnlineDuration *int64 \`json:"online_duration"\``。

#### Layer F 前端 mapper 修正（deviceApi.ts mapBackendDevice）

```ts
// 修复前
onlineTime: bd.online_at || '',
offlineTime: bd.offline_at || '',
onlineDuration: bd.online_duration ?? 0,
upTime: bd.up_time || '',
firstOnlineTime: bd.first_online_at || '',
lastInformTime: bd.last_inform_at || '',

// 修复后
onlineTime: bd.last_online_time ?? '',
offlineTime: bd.last_offline_time ?? '',
onlineDuration: bd.online_duration ?? null,           // 后端派生字段
upTime: bd.run_time ?? null,                          // int64 秒数
firstOnlineTime: bd.first_online_time ?? '',
lastInformTime: bd.last_inform_at ?? '',
```

`Device` 类型同步：
```ts
onlineDuration?: number | null;    // 之前是 number，强制 0 导致 fmtDuration 显示 0m
upTime?: number | null;            // 之前是 string，类型错
```

#### BasicTab fmtTime / fmtDuration 已经容错（null/空字符串都返回 '-'），无需改动

### 13.6 落地范围

- **后端**：单点改动 `device_repository.go` ListWithInfo + `DeviceWithInfo` struct（+ unit test）
- **前端**：单点改动 `deviceApi.ts mapBackendDevice` + `types/device.ts` 类型（+ Vitest 单测）
- **DB 迁移**：**不需要**（全部派生 / 已存在列）
- **代码量**：~50 行后端 + ~30 行前端

### 13.7 优先级建议

与 §12.6 networkType 枚举转换合并为 **Phase 0 Hotfix**：
- networkType 枚举转换（5 行 frontend）
- 时间字段 mapper 字段名对齐（6 行 frontend）
- 后端 list SQL 加 `online_duration` 派生列（10 行 backend）
- 三项合计 ≈ 0.5 人日，**用户感知层 80% 字段空白问题消除**

---

## 14. 修订后的实施计划总览

合并 §5 原 5 阶段 + §12/§13 新增内容：

| 阶段 | 范围 | 预估 |
|---|---|---|
| **Phase 0 (Hotfix)** | networkType 枚举转换 + 时间字段 mapper 字段名对齐 + online_duration 后端派生 | 0.5 人日 |
| **Phase 1** | `last_online_time` mapper bug 修复（Layer G）— 复核 §13 修复后是否仍有问题 | 0.25 人日 |
| **Phase 2** | DB 迁移：device_info 新增列 `tac / band / ul_earfcn / subframe_assignment / special_subframe / root_index / network_model / enb_id / gps_satellites / gps_height / lock_status`；devices.latitude/longitude 改 numeric / 指针 | 1.0 人日 |
| **Phase 3** | Carrier adapter (cmcc + ctcc + cucc) `GetInfoParamMapping` LTE/NR 双分支扩展；InfoSyncer 派生字段（enb_id / lat-lon 缩放）；IP 写 devices 表 | 1.5 人日 |
| **Phase 4** | Composite assembler 修复（MMEIp1 / RFTxStatus / CellOpState / License Value） | 1.0 人日 |
| **Phase 5** | List 接口 SQL 加新列 + 前端类型 + mapper + BasicTab 字段绑定 | 1.5 人日 |
| **Phase 6** | ModelName 从 ProductRegistry 装配件回填 | 0.5 人日 |
| **合计** | | **6.25 人日** |

> Phase 0 必须先合入（hotfix），其余阶段顺序可调整。

---

## 15. 验收清单补充（v1.1）

在 §9 基础上追加：

- [ ] 测试设备 `1202000240194DP0026` 详情页 basic Tab **小区信息分组可见**，含 PCI=`2` / TAC=`2` / Band=`48` / DL EARFCN=`55340` / UL EARFCN=`55341` / Tx Power=`30` / ECI=`654321` / PLMN=`46068` / 帧配比=`2` / 特殊子帧=`5` / 根索引=`10`
- [ ] 状态信息分组的 LTE 专属字段（mmeStatus / pmReportStatus / cpeCount / lockStatus / wanSpeed / serviceStatus / validity）按字段实际值或 '-' 显示，**不再整组消失**
- [ ] 其他信息分组 6 个时间/时长字段全部显示真实值或合理派生：
  - 首次接入 `2026-05-23 16:01:53`
  - 接入时间 `2026-05-24 18:38:15`
  - 断开时间 `2026-05-24 18:37:41`
  - 运行时间 `10d 23h 11m`（来自 `run_time = 947429s`）
  - 累计时长 与 `last_online_time` ↔ `now()` 差值一致（在线时滚动增长）
  - 最近报告 `2026-05-25 13:48:16`
- [ ] 前端 type `Device.networkType` 取值为 `eNB | gNB | GSM` 之一（不再是 `lte / nr / gsm`）
- [ ] 前端 `Device.upTime` 类型从 `string` 改为 `number | null`，单测覆盖 fmtDuration 接收 null 不崩
