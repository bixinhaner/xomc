# F08 北向数据支持矩阵

> 版本：2026-07-31
> 状态：当前页面可配置化设计的数据比对附件
> 对应主设计：`page-config-redesign-20260731.md`
> 结论口径：旧 XML/旧外部文档只作为字段和指标来源；北向页面最终只能选择 xomc 当前可确认的数据项。

## 1. 比对依据

本矩阵把旧北向数据项逐项映射到当前 xomc 数据源。判断分三档：

| 状态 | 含义 | 页面处理 |
|---|---|---|
| 支持 | 当前表、参数投影、PM 指标库或聚合接口已有稳定来源 | 默认可选，可进入内置模板 |
| 部分 | 当前能通过参数、表达式、系统配置或别名推导，但不是稳定列，或语义需要二次确认 | 可在页面展示为“需确认/需配置”，不默认启用 |
| 不支持 | 当前没有业务模型、数据源或已被范围排除 | 不允许选择，只在兼容报告中展示 |

输入来源：

| 来源 | 用途 |
|---|---|
| `/Users/renpengfei/Desktop/doc/config/Northbound-File-Interface-Of-Baicells-Configurations-S0001-dev.xml` 到 `S0017` | 旧 CM/PM/MR 场景、PM `Map`、对象和周期 |
| `/Users/renpengfei/Desktop/doc/othernorth/enbMonitorExport.md` | Station Inventory、OMC Inventory 字段契约 |
| `omcgo/migrations/000001_init_schema.sql` | `devices`、`device_info`、`sys_configs`、PM 指标表结构 |
| `omcgo/docs/param-model-delivery/xml/kpi-indicators/` | 当前内置 KPI/counter 指标库 |
| `omcgo/internal/device/device_info_sync.go` | 设备参数投影到 `device_info` 的真实来源 |
| `omcgo/internal/pm/indicator/loader.go`、`omcgo/internal/tsdbsync/runner.go` | 指标库导入和同步到 `pm_metric_dictionary` 的真实流程 |

## 2. 对象级结论

| 对象 | 旧资料来源 | 当前支持度 | 结论 |
|---|---|---|---|
| CM `CP/EP/CC/CE` | 17 个场景 XML | 部分 | 可用 `devices`、`device_info`、`device_parameters` 做页面字段字典；旧 SQL 表达式不能直接执行，需要按字段映射 |
| CM `COMS` | S0007/S0008 CSV | 部分 | 当前没有专门 COMS 对象模型，只能作为设备/系统配置组合模板 |
| PM `PC` LTE | 旧场景 PM Map | 部分 | 1347 个唯一旧 Map 中 827 个能命中当前指标库，520 个缺失 |
| PM `PC` NR/GNB | 旧场景 PM Map | 支持 | 119 个唯一旧 Map 全部命中当前 GNB 指标库 |
| PM `PC` GSM | 旧场景 PM Map | 部分 | 14 个唯一旧 Map 中 12 个命中当前 GSM 指标库，`TCH_Erlang` 和 `bts_id/cel_id` 不应作为 PM metric |
| PM `PE` | 旧场景 PM Map | 部分 | 可沿用 PC 的 PM 查询能力，但 PE 对象口径需要确认当前是否有对应对象维度 |
| MR `MRO/MRE/MRS` | 旧场景 MR 文件 | 部分 | 当前有 `mr_files` 元数据和对象存储记录，是否能按旧 XML 原样输出取决于 MR 原始文件保存情况 |
| Inventory `STATION` | `enbMonitorExport.md` | 部分 | 多数基础设备字段可支持；规划、联系人、线路、维护团队等人工资产字段当前没有模型 |
| Inventory `OMC` | `enbMonitorExport.md` | 部分 | 设备聚合指标可支持；OMC 名称/IP/硬件型号/HA 状态需要系统配置或部署信息补充 |

## 3. Station Inventory 字段逐项比对

旧 Station CSV 第 1 列 `Operator` 是文件级常量，建议作为 profile 配置项，不进入设备字段表。下表覆盖 `enbMonitorExport.md` 中 MySQL 81 项和 Mongo 扩展 8 项。

| 序号 | 输出列 | 旧别名 | 状态 | xomc 来源/处理 |
|---:|---|---|---|---|
| 1 | Serial Number | `serial_number` | 支持 | `devices.serial_number` |
| 2 | Cell Status | `op_state` | 支持 | `device_info.op_state` |
| 3 | Cell Status GNB | `gnb_op_state` | 部分 | 旧字段 `noShow=true`；NR 可用 `device_info.admin_state/op_state` 映射 |
| 4 | Online Status | `connection_status` | 支持 | `devices.is_online` |
| 5 | Alarms | `alarm_count` | 支持 | `device_info.active_alarm_count`，来源为活动告警聚合 |
| 6 | Cell Name | `host_name` | 支持 | `device_info.device_name` 或 `devices.site_name` |
| 7 | Shop ID | `site_id` | 支持 | `devices.site_id` |
| 8 | IP Address | `cell_ip` | 支持 | `devices.ip_address` |
| 9 | MME Interface Binding(Non-Ipsec) | `ipsec_addr` | 部分 | 可取 `device_info.ipsec_addr`，但旧列名写 Non-Ipsec，语义需确认 |
| 10 | MAC Address | `mac_address` | 支持 | `device_info.mac` |
| 11 | ECI | `cell_identity` | 支持 | `device_info.eci` |
| 12 | PCI | `pycellid` | 支持 | `device_info.pci` |
| 13 | Earfcn | `earfcndlinuse` | 支持 | `device_info.freq_point`，按 DL EARFCN 输出 |
| 14 | UL Frequency | `uplinkFrequency` | 部分 | `device_info.ul_earfcn` 是 UL EARFCN；若要 MHz 频率需要转换公式 |
| 15 | DL Frequency | `downlinkFrequency` | 部分 | `device_info.freq_point` 是 DL EARFCN；若要 MHz 频率需要转换公式 |
| 16 | MME Status | `mme_status` | 支持 | `device_info.mme_status` |
| 17 | KPI Report Status | `pm_report_status` | 支持 | `device_info.kpi_status` |
| 18 | Sync Status | `sync_status` | 支持 | `device_info.sync_status` |
| 19 | UE Count | `ue_count` | 支持 | `device_info.ue_count` |
| 20 | Last Period Time | `lastinformtime` | 支持 | `devices.last_inform_at` |
| 21 | Product Type | `product_type` | 支持 | `devices.product_class`，必要时关联 `products` |
| 22 | Hardware Version | `firmware_version` | 支持 | 优先 `device_info.hardware_version`；旧别名若确认为固件则改取 `devices.firmware_version` |
| 23 | Software Version | `software_version` | 支持 | `devices.firmware_version`，对应 TR-181 SoftwareVersion |
| 24 | Kernel Version | `kernal_version` | 部分 | 当前无稳定列，可从 `device_parameters` 指定路径补取 |
| 25 | Device Group | `group_id` | 支持 | `device_groups` / `device_group_members` |
| 26 | RF Status | `rf_status` | 支持 | `device_info.rf_status` |
| 27 | RF Status GNB | `gnb_rf_enable` | 部分 | 旧字段 `noShow=true`；NR 可用 `device_info.rf_status` 或参数别名 |
| 28 | Active Ratio(30 days) | `available_rate` | 不支持 | 当前无 30 天可用率聚合模型 |
| 29 | Satellites | `gps_satellite_count` | 支持 | `device_info.gps_satellites` |
| 30 | Longitude | `gps_longitude` | 支持 | `devices.longitude`，也可参考 `device_location_observations` |
| 31 | Latitude | `gps_latitude` | 支持 | `devices.latitude`，也可参考 `device_location_observations` |
| 32 | Height | `gps_height` | 支持 | `device_info.gps_height` |
| 33 | Duplex Mode | `network_model` | 支持 | `device_info.network_model` |
| 34 | HaloB Enable | `halob_flag` | 部分 | 当前无稳定列，可按产品私有参数补取 |
| 35 | GPS Version | `gps_version` | 部分 | 详情页按 `Device.FAP.GPS.SoftVersion` 临时投影，未入 `device_info` |
| 36 | IPsec Address | `mmepool_ipsec_addr` | 支持 | `device_info.ipsec_addr` |
| 37 | PLMN | `plmn` | 支持 | `device_info.plmn` |
| 38 | First Online Time | `first_online_time` | 支持 | `device_info.first_online_time` |
| 39 | AMF Status | `amf_status` | 支持 | NR AMF 状态归一到 `device_info.mme_status`，详情别名为 `amf_status` |
| 40 | CPE Count | `cpe_connect` | 不支持 | 当前无基站挂载 CPE 数模型 |
| 41 | TAC | `tac` | 支持 | `device_info.tac` |
| 42 | Model Name | `model_name` | 支持 | `devices.model_name` |
| 43 | Circui Ref. | `circuit_ref` | 不支持 | 当前无线路资产字段 |
| 44 | Circuit J O | `circuit_jo` | 不支持 | 当前无线路资产字段 |
| 45 | Status | `service_status` | 部分 | 可由 `devices.lifecycle_state` / `device_info.project_status` 映射，旧枚举需确认 |
| 46 | ROM | `rom` | 不支持 | 当前无 ROM 字段 |
| 47 | Cell Active State | `cell_admin_state` | 部分 | NR 有 `device_info.admin_state`，LTE 有 `lock_status/op_state`，旧枚举需映射 |
| 48 | Femto Vendor | `femto_vendor` | 部分 | 可用 `devices.manufacturer` / `oui` 近似，当前无独立 Femto Vendor |
| 49 | Manufacturer | `manufacturer` | 支持 | `devices.manufacturer` |
| 50 | State/Region | `province` | 部分 | 当前无规范化行政区字段，可从地址或人工扩展字段补充 |
| 51 | Town/City | `city` | 部分 | 当前无规范化行政区字段 |
| 52 | County | `district` | 部分 | 当前无规范化行政区字段 |
| 53 | Township | `township` | 不支持 | 需要人工资产字段 |
| 54 | Grid | `sub_grid` | 不支持 | 需要人工资产字段 |
| 55 | Branch Office | `sub_branches` | 不支持 | 需要人工资产字段 |
| 56 | Site Code | `sub_station_code` | 部分 | 可用 `devices.site_id`，但旧站点编码语义需确认 |
| 57 | Site Name | `sub_station_name` | 支持 | `devices.site_name` / `device_info.device_name` |
| 58 | Distribute System | `sub_distribute_system` | 不支持 | 需要人工资产字段 |
| 59 | ECI(manual input) | `sub_cell_id` | 部分 | 当前有真实 `device_info.eci`，无人工覆盖 ECI 字段 |
| 60 | eNodeB Name | `sub_cell_name` | 部分 | 可用 `device_info.device_name/lmt_device_name`，旧规划名需确认 |
| 61 | TAList | `sub_talist` | 不支持 | 当前仅有单值 `device_info.tac` |
| 62 | Override Scene Properties | `over_scen_attr` | 不支持 | 当前无场景覆盖属性模型 |
| 63 | Device Power | `maxtxpower` | 部分 | 当前 `device_info.transmit_power` 是实际参考信号功率；若要 MaxTxPower 需参数补取 |
| 64 | Device Access Type | `device_access_mode` | 部分 | 当前无稳定列，可按参数或人工字段补充 |
| 65 | RX Port Number | `tx_port_number` | 部分 | 当前无稳定列，可按天线/RF 参数补取 |
| 66 | OMC IP | `omc_ip` | 部分 | 需从 `sys_configs` 或部署配置显式配置 |
| 67 | Network Element Grade | `netelement_grade` | 不支持 | 需要人工资产字段 |
| 68 | Installation Detailed Address | `install_address` | 支持 | `device_info.address` |
| 69 | Network Access Time | `access_net_date` | 部分 | 可用 `first_online_time` 近似，但入网时间语义需单独字段 |
| 70 | Maintenance Team | `agent_maintain` | 不支持 | 需要人工资产字段 |
| 71 | Owner | `contact_person` | 不支持 | 需要人工资产字段 |
| 72 | Contact | `contact_number` | 不支持 | 需要人工资产字段 |
| 73 | Device Status | `device_status` | 支持 | `devices.lifecycle_state` + `devices.is_online` |
| 74 | Uplink Broadband Account | `uplink_broadband_account` | 不支持 | 需要人工资产字段 |
| 75 | First Period Time | `first_online_time` | 支持 | `device_info.first_online_time` |
| 76 | Product Name | `product_name` | 支持 | `products.name` 或 `devices.product_class` |
| 77 | System Uptime | `up_time` | 支持 | `device_info.run_time` |
| 78 | Accumulated Online Time(s) | `online_duration` | 支持 | `device_info.cumulative_online_duration` + 在线中动态时长 |
| 79 | Bandwidth | `dl_bandwidth` | 支持 | `device_info.bandwidth` |
| 80 | Height(m) | `gps_height` | 支持 | `device_info.gps_height` |
| 81 | txPower | `MAXTXPOWER` | 部分 | 同第 63 项，当前稳定列是 `transmit_power`，不是 MaxTxPower |
| 82 | EU Count | `eu_count` | 不支持 | 当前无 EU 数模型 |
| 83 | RU Count | `ru_count` | 不支持 | 当前无 RU 数模型 |
| 84 | eNB ID | `enbId` | 支持 | `device_info.enb_id` |
| 85 | Cell ID | `cellId` | 支持 | `device_info.cell_id` |
| 86 | WAN Link Speed Negotiated | `wanSpeed` | 部分 | 当前详情有 WAN 状态概念，链路速率需参数补取 |
| 87 | Subframe Assignment | `SubFrameAssignment` | 支持 | `device_info.subframe_assignment` |
| 88 | Special Subframe Patterns | `SpecialSubframePatterns` | 支持 | `device_info.special_subframe` |
| 89 | Root Sequence Index | `RootSequenceIndex` | 部分 | 当前有 `device_info.root_index`，但投影路径需确认是否应从 `RootSequenceIndex` 而不是 `ZeroCorrelationZoneConfig` 取值 |

首期建议：

1. Station 模板默认只启用“支持”字段。
2. “部分”字段在页面显示来源说明，用户确认后才能启用。
3. “不支持”字段进入兼容报告，不进入字段选择器。
4. 规划/联系人/线路类字段如果业务确认必须输出，应新增 `northbound_asset_fields` 或复用后续资产管理模型，不能塞进 `devices.extension_data` 后无约束输出。

## 4. OMC Inventory 字段逐项比对

| 序号 | 输出列 | 状态 | xomc 来源/处理 |
|---:|---|---|---|
| 1 | OMC Name | 部分 | `sys_configs` 新增显式配置，默认可取部署名称 |
| 2 | OMC IP | 部分 | `sys_configs` 新增显式配置，不能靠运行时猜网卡 |
| 3 | eNB online | 支持 | `devices.is_online` 按基站类/制式聚合 |
| 4 | eNB active | 支持 | `device_info.op_state` 或 `devices.lifecycle_state` 聚合 |
| 5 | MME status | 支持 | `device_info.mme_status` 正常数/总数 |
| 6 | UE Count | 支持 | `SUM(device_info.ue_count)` |
| 7 | Active/Standby state | 部分 | 当前 `northbound_servers` 是 OSS 目标主备，不是 OMC HA；无 HA 时输出 `SINGLE` |
| 8 | Version | 支持 | `buildinfo.ReleaseVersion` / 构建信息 |
| 9 | Hardware Model | 部分 | 需要 `sys_configs` 或部署探针显式提供 |

## 5. PM 指标逐项比对

### 5.1 当前系统 PM 指标来源

当前系统的指标来源链路是：

1. `omcgo/docs/param-model-delivery/xml/kpi-indicators/` 提供 ENB/GNB/GSM 指标库。
2. `indicator.Loader` 扫描 XML 并写入 `perf_indicators_{enb,gsm,gnb}`。
3. `tsdbsync` 把三张指标表同步成 `pm_metric_dictionary`。
4. 页面候选指标通过 `/api/v1/pm/kpi/definitions?include_counters=true` 或 PM 指标管理接口读取。
5. 北向导出按 `metric_path` 查询 PM 聚合数据，不能临时按旧 CSV 列名重算。

当前内置 XML 指标库按 loader 同制式同 `id` 去重后的结果：

| 制式 | 指标总数 | 说明 |
|---|---:|---|
| ENB/LTE | 1408 | 1333 个 counter、75 个 KPI，旧 LTE PC 主要从这里匹配 |
| GNB/NR | 282 | 旧 GNB PC 119 项全部可匹配 |
| GSM | 73 | 旧 GSM PC 14 项中 12 项可匹配 |
| 合计 | 1763 | 包含 1595 个 counter、168 个 KPI |

### 5.2 旧 PM Map 覆盖率

严格匹配规则：旧 `Map` 的源名或目标名，按原值和 `_` 转 `.` 规范化后，命中当前指标 XML 的 `id/reportKey/enName/formula` 即认为支持。

| 旧对象 | 唯一 Map 数 | 支持 | 不支持 | 结论 |
|---|---:|---:|---:|---|
| `PM/PC` LTE | 1347 | 827 | 520 | 默认模板只能启用 827 项；其余需补指标库或标记不支持 |
| `PM/PC` GNB | 119 | 119 | 0 | 可作为 GNB PC 内置模板 |
| `PM/PC` GSM | 14 | 12 | 2 | `TCH_Erlang` 和 `bts_id/cel_id` 不作为 metric 默认输出 |
LTE PC 缺口按族统计：

| 族 | 旧 Map 数 | 支持 | 缺失 | 说明 |
|---|---:|---:|---:|---|
| `PHY` | 165 | 17 | 148 | 大量旧物理层私有/派生列未进入当前指标库 |
| `RRC` | 171 | 57 | 114 | CL0/CL1/CL2、带单位展示名等缺失较多 |
| `MAC` | 101 | 28 | 73 | 旧展示别名和当前 reportKey 不完全一致 |
| `RRU` | 88 | 44 | 44 | 部分设备侧上报名未纳入当前指标库 |
| `CONTEXT` | 53 | 14 | 39 | 旧 cause/CL 分解项缺失 |
| `HO` | 113 | 80 | 33 | 部分派生成功率/失败原因项缺失 |
| `ERAB` | 262 | 230 | 32 | 大部分支持，少量旧失败原因或单位列缺失 |
| `DRB` | 143 | 139 | 4 | 基本支持 |
| `PDCP` | 84 | 82 | 2 | 基本支持 |
| `MR` | 51 | 51 | 0 | 支持 |
| `KPI` | 9 | 9 | 0 | LTE 旧 Map 中的 KPI 项可匹配 |

典型不支持 LTE PC 项：

| 旧字段 | 旧源名 | 原因 |
|---|---|---|
| `% of CQI 0-6(%)` | `% of CQI 0-6(%)` | 旧派生展示列，不是当前 metric |
| `CONTEXT.AttRelEnbCL0` | `CONTEXT_AttRelEnbCL0` | 当前指标库无 CL0 分解项 |
| `Data Volume DL(MByte)` | `Data Volume DL(MByte)` | 旧单位换算展示列，应改用当前流量 metric + renderer 单位转换 |
| `HO InterEnbOutSucc Rate S1(%)` | `HO InterEnbOutSucc Rate S1(%)` | 旧派生 KPI，当前没有同名指标 |
| `DRB.PdcpSduBitrateDlMax(Kbps)` | `DRB_PdcpSduBitrateDlMax(Kbps)` | 旧带单位别名未命中当前 reportKey |

### 5.3 旧 `KPI_*` 逐项映射

旧场景中出现的 26 个 `KPI_*` 名称全部可在当前指标库命中。页面内置模板应保存当前 `metric_path`，旧名只作为导出列名或兼容别名。

| 旧源名 | 当前 `metric_path` | 当前 `report_key` | 制式 | 类型 | `statis_type` | 状态 |
|---|---|---|---|---|---|---|
| `KPI_RrcSuccConnRate` | `KGNB0101` | `KPI.RrcSuccConnRate` | NR | KPI | `pct` | 支持 |
| `KPI_FlowSuccConnRate` | `KGNB0102` | `KPI.FlowSuccConnRate` | NR | KPI | `pct` | 支持 |
| `KPI_NGSIG_SuccConnRate` | `KGNB0103` | `KPI.NGSIG.SuccConnRate` | NR | KPI | `pct` | 支持 |
| `KPI_WirelessSuccConnRate` | `KGNB0104` | `KPI.WirelessSuccConnRate` | NR | KPI | `pct` | 支持 |
| `KPI_WirelessDropRate_CellLevel` | `KGNB0301` | `KPI.WirelessDropRate.CellLevel` | NR | KPI | `pct` | 支持 |
| `KPI_FlowDropRate_CellLevel` | `KGNB0302` | `KPI.FlowDropRate.CellLevel` | NR | KPI | `pct` | 支持 |
| `KPI_RrcConnReestabRate` | `KGNB0303` | `KPI.RrcConnReestabRate` | NR | KPI | `pct` | 支持 |
| `KPI_RlcNbrPktLossRateDl` | `KGNB0401` | `KPI.RlcNbrPktLossRateDl` | NR | KPI | `pct` | 支持 |
| `KPI_RLCPktDelayDL` | `KGNB0417` | `KPI.RLCPktDelayDL` | NR | KPI | `sum` | 支持 |
| `KPI_RlcUpOctUl` | `KGNB0501` | `KPI.RlcUpOctUl` | NR | KPI | `sum` | 支持 |
| `KPI_RlcUpOctDl` | `KGNB0502` | `KPI.RlcUpOctDl` | NR | KPI | `sum` | 支持 |
| `KPI_PdcpUpOctUl` | `KGNB0510` | `KPI.PdcpUpOctUl` | NR | KPI | `sum` | 支持 |
| `KPI_PdcpUpOctDl` | `KGNB0511` | `KPI.PdcpUpOctDl` | NR | KPI | `sum` | 支持 |
| `KPI_AvgDataRateUl` | `KGNB0516` | `KPI.AvgDataRateUl` | NR | KPI | `avg` | 支持 |
| `KPI_AvgDataRateDl` | `KGNB0517` | `KPI.AvgDataRateDl` | NR | KPI | `avg` | 支持 |
| `KPI_HandoverSuccessRate` | `KGSM0101` | `KPI.HandoverSuccessRate` | GSM | KPI | `pct` | 支持 |
| `KPI_CallSetupSuccRate` | `KGSM0102` | `KPI.CallSetupSuccRate` | GSM | KPI | `pct` | 支持 |
| `KPI_CallDropRate` | `KGSM0103` | `KPI.CallDropRate` | GSM | KPI | `pct` | 支持 |
| `KPI_SDCCHDropRate` | `KGSM0104` | `KPI.SDCCHDropRate` | GSM | KPI | `pct` | 支持 |
| `KPI_SDCCHBlockingRate` | `KGSM0105` | `KPI.SDCCHBlockingRate` | GSM | KPI | `pct` | 支持 |
| `KPI_SDCCHEstSuccessRate` | `KGSM0106` | `KPI.SDCCHEstSuccessRate` | GSM | KPI | `pct` | 支持 |
| `KPI_TCHAssignSuccessRate` | `KGSM0107` | `KPI.TCHAssignSuccessRate` | GSM | KPI | `pct` | 支持 |
| `KPI_TCHUtilizationRate` | `KGSM0108` | `KPI.TCHUtilizationRate` | GSM | KPI | `pct` | 支持 |
| `KPI_TCHAvaliableRate` | `KGSM0109` | `KPI.TCHAvaliableRate` | GSM | KPI | `pct` | 支持 |
| `KPI_Call_SameTimeMaxUe` | `KGSM0110` | `KPI.Call.SameTimeMaxUe` | GSM | KPI | `max` | 支持 |
| `KPI_Paging_Responded` | `KGSM0113` | `KPI.Paging.Responded` | GSM | KPI | `sum` | 支持 |

### 5.4 PM 页面规则

1. 页面保存 PM 文件配置时必须保存当前 `metric_path`，不能保存旧 `KPI_*` 或旧 Mongo 字段名作为查询 key。
2. 旧列名可以作为 `export_name`，用于兼容 CSV 表头。
3. `counter` 和 `KPI` 可混选，但查询必须按当前 PM 聚合模块同源读取。
4. `pct` 类型 KPI 使用 PM 模块已有公式/聚合结果，北向导出不临时重算。
5. 缺失的旧 counter 如果业务必须保留，应走“指标库导入/自定义指标”流程，让它进入 `perf_indicators_*` 和 `pm_metric_dictionary` 后再允许选择。

## 6. 实现侧校验清单

| 校验点 | 要求 |
|---|---|
| 字段字典初始化 | 每个旧字段都生成 `support_status` 和 `support_reason` |
| 内置模板生成 | 只默认启用 `supported` 项；`partial` 项默认关闭 |
| PM 指标选择 | 后端保存前校验 `metric_path` 存在于当前 PM 字典 |
| PM 覆盖率报告 | 导入旧场景时输出 supported/partial/unsupported 计数和明细 |
| Inventory 导出 | `Root Sequence Index`、`MAXTXPOWER`、`Non-Ipsec` 等语义风险项必须有单测或人工确认 |
