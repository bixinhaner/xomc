# 03 — eNodeB (eNodeB 监控)

> **覆盖状态：⚠️ 部分覆盖**

---

## 原系统二级菜单

| # | 二级菜单 | 原系统文件数 | 新系统对应 | 状态 |
|---|---------|------------|----------|------|
| 1 | Monitor (设备监控) | 18 | Device → DeviceList/OnlineMonitoring | ⚠️ 部分 |
| 2 | Monitor Settings (监控设置) | 23 | — | ❌ 缺失 |
| 3 | Monitor SettingPage (监控页设置) | 10 | — | ❌ 缺失 |
| 4 | Topology (拓扑) | 6 | Topology → GISMap/Canvas | ⚠️ 部分 |
| 5 | CollectLog Direct (直接日志) | 3 | File → LogRetrieval | ⚠️ |
| 6 | CollectLog Period (周期日志) | 5 | File → LogRetrieval | ⚠️ 缺周期 |
| 7 | HALoB SelfConfig | 8 | — | ❌ 缺失 |
| 8 | GSM Monitor | 15 | — | ❌ 缺失 |

---

## 3.1 Monitor — 设备监控

### 原系统 `enodeb_monitor.jsp`

**筛选字段：**
- `search_text` — 模糊搜索
- `connection_status` — 连接状态 (1=在线, 0=离线, 3=同步中, 2=同步失败, 4=初始化, 5=远程同步, 6=同步完成)
- `op_state` — 操作状态 (1=激活, 0=停用)
- `product_model` — 产品型号
- `model_name` — 设备型号
- `software_version` — 软件版本
- `firmware_version` — 固件版本
- `group_id` — 设备分组
- `halob_flag` — HaloB 开关

**表格列字段对比：**

| 原系统列 | 说明 | 新系统 | 状态 |
|---------|------|--------|------|
| `serial_number` | 序列号 | sn | ✅ |
| `host_name` | 主机名 | name | ✅ |
| `cell_ip` | IP地址 | ipAddress | ✅ |
| `mac_address` | MAC地址 | — | ❌ |
| `cell_identity` | ECI | — | ❌ |
| `phycellid` | PCI | — | ❌ |
| `connection_status` | 连接状态(7种) | connStatus(2种) | ⚠️ 简化 |
| `op_state` | 操作状态 | — | ❌ |
| `product` | 产品类型 | productType | ✅ |
| `module_type` | 设备型号 | deviceModel | ✅ |
| `software_version` | 软件版本 | — | ❌ |
| `group_name` | 设备分组 | — | ❌ |
| `ue_count` | 用户数 | — | ❌ |
| `cpe_connect` | CPE连接数 | — | ❌ |
| `mme_status` | MME状态 | — | ❌ |
| `rf_status` | RF开关 | — | ❌ |

**操作功能缺失：**
- ❌ 列自定义（显示/隐藏列）
- ❌ 消息采集（查看/下载/清除）
- ❌ 锁定/解锁自动刷新

### 新系统 OnlineMonitoring 新增字段
- CPU(%), Memory(%), Temperature(℃), Online Uptime — 原系统无

---

## 3.2 Export Config — 导出配置

### 原系统 `export_config.jsp`

**可选导出列 50+ 字段，按分组：**

| 分组 | 字段 |
|------|------|
| 设备信息 | serial_number, product, product_name, module_type, software_version, firmware_version, online_duration, up_time, first_online_time, LASTINFORMTIME, online_time, offline_time, mac_address, gps_version, group_name, sub_station_name, rom, remark |
| 小区信息 | enbId, host_name, cellId, CELL_IDENTITY(ECI), PHYCELLID(PCI), plmnid, tac, signment, specialSubframe, rootIndex, site_id, bandwidth, EARFCNDLINUSE, network_model, tx_power |
| 状态信息 | op_state, mme_status, rf_status, pm_report_status, halob_flag, synStatus, validity, lock_status, ue_count, euCountStr, ruCountStr, cpe_connect, wanSpeed, service_status |
| 网络配置 | mmepool_ipsec_addr, IPSEC_ADDR, cell_ip |
| 位置信息 | gps_longitude, gps_latitude, gps_height, mechanical_downtilt, electronic_downtilt, vertical_3dB_beam_width, horizontal_azimuth, install_address |
| 卫星信息 | gps_satellite_count |

**导出格式：** CSV / XLSX，可勾选「包含许可证信息」

### 新系统 Device → ImportExport

**导出筛选：** Vendor, Product Type, Connection Status, Date Range
**导出格式：** Excel(.xlsx) / CSV(.csv)

**缺失：** ❌ 50+ 可选导出列自定义, ❌ 按分组选列, ❌ 许可证信息选项

---

## 3.3 Topology — 拓扑

| 原系统页面 | 功能 | 新系统 | 状态 |
|-----------|------|--------|------|
| `topo.jsp` | 小区拓扑 | Topology → Canvas | ⚠️ |
| `eNBTopo_new.jsp` | 新拓扑(设备滑动面板+交互地图+节点标记) | Topology → GISMap | ⚠️ |
| `eNBTopo_tab.jsp` | 多标签拓扑 | — | ❌ |
| `eNBTopo_dashboard.jsp` | 拓扑仪表板 | — | ❌ |
| `ue_trace.jsp` | UE追踪(轨迹可视化) | — | ❌ |

---

## 3.4 CollectLog — 日志采集

| 原系统页面 | 功能 | 新系统 | 状态 |
|-----------|------|--------|------|
| `direct/log_manage.jsp` | 直接日志采集 | File → LogRetrieval | ⚠️ |
| `period/log_manage_elfcell.jsp` | 周期采集(上报周期设置/任务状态/文件列表) | — | ❌ 缺周期采集 |

**周期采集表格列：** serial_number, file_num, report_period, task_status, update_time
**日志文件列：** file_name, upload_time
**操作：** 周期上报(启/停), 批量下载, 批量删除

---

## 3.5 HALoB SelfConfig (8 文件，全部 ❌ 缺失)

halob_self_config, halob_add_config, halob_plug_and_play, halob_selfConfig_control, halob_selfConfig_task_progress 等

---

## 3.6 GSM Monitor (15 文件，全部 ❌ 缺失)

gsm_monitor_vue, gsm_addOrImportDevice, gsm_settingBasic, gsm_settingBts, gsm_settingBtsDetail, gsm_settingBtsSetting, gsm_settingOverview, gsm_settingUpgrade, gsm_syncParams, gsm_settingLicense 等

---

## 缺失操作流程汇总

| 操作流程 | 状态 |
|---------|------|
| 监控字段/列自定义 | ❌ |
| 消息采集/查看/下载 | ❌ |
| 导出列自定义选择(50+列) | ❌ |
| UE 追踪 | ❌ |
| 周期日志采集 | ❌ |
| HALoB 自配置全流程 | ❌ |
| GSM 设备管理全流程 | ❌ |
| 监控参数设置(23+10 文件) | ❌ |
