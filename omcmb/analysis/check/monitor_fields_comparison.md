# 监控页面列表字段对比

> 基于三个原始监控页面 JSP 文件逐字段提取，精确记录每个字段的 field name、显示标签、默认可见性和锁定状态。
>
> 源文件：
> - **LTE (eNodeB)**: `enodeb/monitor/enodeb_monitor_vue.jsp`
> - **GSM**: `enodeb/monitor/GSM/gsm_monitor_vue.jsp`
> - **5G NR (gNodeB)**: `gnodeb/monitor/gnodeb_monitor.jsp`

---

## 1. 固定列（始终显示，不可配置）

三个页面都有以下固定列，始终 fixed 在左侧：

| # | 功能 | LTE eNodeB | GSM | 5G NR gNodeB |
|---|------|:----------:|:---:|:------------:|
| 1 | 复选框 (checkbox) | ✅ (需权限) | ✅ (需权限) | ✅ (需权限) |
| 2 | 操作 (operation) | ✅ (70px) | ✅ (70px) | ✅ (80px) |
| 3 | 连接状态图标 | ✅ (40px) | ✅ (40px) | ✅ (60px) |
| 4 | 告警数 | ✅ (115px) | ✅ (110px) | ✅ (110px) |
| 5 | 序列号 (serial_number) | ✅ (180px) fixed | ✅ (180px) fixed | ✅ (min-240px) fixed |
| 6 | 主机名 (host_name) | ✅ (150px) fixed | ✅ (150px) fixed | ✅ (180px) fixed |
| 7 | 序号 (row_index) | ✅ (50px) | ✅ (60px) | - |

---

## 2. 各页面完整字段清单

### 2.1 LTE eNodeB — 共 60 个可配置字段，分 6 组

#### 设备信息组 (deviceCol) — 18 个

| # | field | 中文标签 | 默认勾选 | 锁定(disabled) | 宽度 |
|---|-------|---------|:--------:|:--------------:|------|
| 1 | serial_number | 小站编码 | ✅ | ✅ | 180 |
| 2 | product | 产品类型标识 | ✅ | ✅ | 110 |
| 3 | product_name | 产品名称 | - | - | 120 |
| 4 | module_type | 设备型号名 | ✅ | ✅ | 120 |
| 5 | software_version | 软件版本 | ✅ | ✅ | 140 |
| 6 | firmware_version | 固件版本 | - | - | 135 |
| 7 | online_duration | 累计时长 | - | - | 120 |
| 8 | up_time | 运行时间 | - | - | 120 |
| 9 | first_online_time | 第一次连接时间 | - | - | 140 |
| 10 | LASTINFORMTIME | 上次连接时间 | - | - | 140 |
| 11 | online_time | 接入时间 | ✅ | - | 140 |
| 12 | offline_time | 断开时间 | ✅ | - | 140 |
| 13 | mac_address | MAC地址 | ✅ | ✅ | 130 |
| 14 | gps_version | GPS版本 | - | - | 120 |
| 15 | group_name | 设备组 | ✅ | ✅ | 130 |
| 16 | sub_station_name | 站点名称 | - | - | 120 |
| 17 | rom | Rom | - | - | 120 |
| 18 | remark | 备注 | - | - | 140 |

#### 小区信息组 (cellCol) — 15 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | enbId | eNodeB ID | - | - | 100 |
| 2 | host_name | 主机名 | ✅ | ✅ | 150 |
| 3 | cellId | 小区ID | - | - | 100 |
| 4 | CELL_IDENTITY | ECI | ✅ | ✅ | 100 |
| 5 | PHYCELLID | PCI | ✅ | ✅ | 70 |
| 6 | plmnid | PLMN | - | - | 70 |
| 7 | tac | TAC | - | - | 80 |
| 8 | signment | 子帧配比 | - | - | 80 |
| 9 | specialSubframe | 特殊子帧配比 | - | - | 80 |
| 10 | rootIndex | 根序列索引 | - | - | 80 |
| 11 | site_id | 站点ID | - | - | 80 |
| 12 | bandwidth | 带宽 | - | - | 80 |
| 13 | EARFCNDLINUSE | 频点 | - | - | 130 |
| 14 | network_model | 基站指示 | - | - | 120 |
| 15 | tx_power | CPE发射功率 | - | - | 80 |

#### 状态信息组 (statusCol) — 14 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | op_state | 是否激活 | ✅ | ✅ | 120 |
| 2 | mme_status | MME状态 | ✅ | ✅ | 140 |
| 3 | rf_status | 射频开关状态 | ✅ | ✅ | 150 |
| 4 | pm_report_status | KPI上报状态 | - | - | 135 |
| 5 | halob_flag | HaloX | - | - | 100 |
| 6 | synStatus | 同步状态 | - | - | 160 |
| 7 | validity | 有效期 | - | - | 125 |
| 8 | lock_status | 锁定状态 | - | - | 90 |
| 9 | ue_count | UE数 | ✅ | ✅ | 80 |
| 10 | euCountStr | EU数 | - | - | 80 |
| 11 | ruCountStr | RU数 | - | - | 80 |
| 12 | cpe_connect | CPE连接数 | ✅ | ✅ | 100 |
| 13 | wanSpeed | WAN状态 | - | - | 180 |
| 14 | service_status | 状态 | - | - | 120 |

#### 网络信息组 (networkCol) — 3 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | cell_ip | IP地址 | ✅ | ✅ | 120 |
| 2 | IPSEC_ADDR | IPSEC地址 | - | - | 170 |
| 3 | mmepool_ipsec_addr | MME Pool IPSEC地址 | - | - | 120 |

#### 位置信息组 (locationCol) — 8 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | gps_longitude | GPS经度 | - | - | 100 |
| 2 | gps_latitude | GPS纬度 | - | - | 90 |
| 3 | gps_height | GPS高度 | - | - | 70 |
| 4 | mechanical_downtilt | 机械下倾角 | - | - | 135 |
| 5 | electronic_downtilt | 电子下倾角 | - | - | 135 |
| 6 | vertical_3dB_beam_width | 垂直波束宽度 | - | - | 160 |
| 7 | horizontal_azimuth | 水平方位角 | - | - | 135 |
| 8 | install_address | 安装详细地址 | - | - | 200 |

#### 卫星信息组 (satelliteCol) — 1 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | gps_satellite_count | GPS卫星数 | - | - | 85 |

**LTE 默认勾选汇总 (17个)**：serial_number, product, module_type, software_version, mac_address, group_name, online_time, offline_time, host_name, CELL_IDENTITY, PHYCELLID, op_state, mme_status, rf_status, ue_count, cpe_connect, cell_ip

**LTE 锁定字段 (15个)**：serial_number, product, module_type, software_version, mac_address, group_name, host_name, CELL_IDENTITY, PHYCELLID, op_state, mme_status, rf_status, ue_count, cpe_connect, cell_ip

---

### 2.2 GSM — 共 33 个可配置字段，扁平列表（无分组）

| # | field | 中文标签 | 在列选择器(18项) | 宽度 |
|---|-------|---------|:----------------:|------|
| 1 | rf_status | 射频开关状态 | - | 150 |
| 2 | op_state | 是否激活 | ✅ | 120 |
| 3 | product | 产品类型标识 | ✅ | 110 |
| 4 | module_type | 设备型号名 | ✅ | 120 |
| 5 | product_name | 产品名称 | ✅ | 120 |
| 6 | cell_ip | IP地址 | ✅ | 120 |
| 7 | sub_station_name | 站址名称 | ✅ | 120 |
| 8 | mac_address | MAC | ✅ | 130 |
| 9 | ue_count | UE数 | ✅ | 80 |
| 10 | IpaUnitId | Ipa Unit Id | - | 120 |
| 11 | OmlRemoteIp | Oml Remote Ip | - | 120 |
| 12 | OmlRemoteIpBak | Oml Remote Ip Bak | - | 150 |
| 13 | BscSelect | BSC Select | - | 100 |
| 14 | BscLinkStatus | BSC连接状态 | - | 120 |
| 15 | BSCSerialNumber | 所属BSC编码 | - | 160 |
| 16 | BtsNum | BTS数 | - | 90 |
| 17 | synStatus | ���步状态 | - | 160 |
| 18 | gps_satellite_count | GPS卫星数 | - | 85 |
| 19 | gps_longitude | GPS经度 | - | 100 |
| 20 | gps_latitude | GPS纬度 | - | 90 |
| 21 | gps_height | GPS高度 | - | 70 |
| 22 | currentLac | LAC | - | 70 |
| 23 | currentArfcn | 频点 | - | 70 |
| 24 | uplinkFrequency | 上行频率 | - | 110 |
| 25 | downlinkFrequency | 下行频率 | - | 110 |
| 26 | up_time | 运行时间 | ✅ | 120 |
| 27 | online_duration | 累计时长 | ✅ | 120 |
| 28 | first_online_time | 第一次连接时间 | ✅ | 140 |
| 29 | LASTINFORMTIME | 上次连接时间 | ✅ | 220 |
| 30 | software_version | 软件版本 | ✅ | 140 |
| 31 | firmware_version | 固件版本 | ✅ | 135 |
| 32 | group_name | 设备组 | ✅ | 130 |
| 33 | remark | 备注 | ✅ | 185 |

> **注意**：GSM 页面的 `showColums` 过滤逻辑被注释掉，实际运行时**所有 33 个字段默认全部显示**。无锁定机制。列选择器仅包含 18 个基础字段。

---

### 2.3 5G NR gNodeB — 共 42 个可配置字段，分 5 组 + 同步对话框扩展

#### 设备信息组 (device) — 17 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | serial_number | 小站编码 | ✅ | ✅ | min-240 |
| 2 | product | 产品类型标识 | ✅ | ✅ | 120 |
| 3 | product_name | 产品名称 | - | - | 120 |
| 4 | module_type | 设备型号名 | ✅ | ✅ | 120 |
| 5 | host_name | 5G站点名称 | ✅ | ✅ | 180 |
| 6 | gNBId | gNodeB ID | - | - | 80 |
| 7 | firmware_version | 硬件版本 | - | - | 150 |
| 8 | software_version | 软件版本 | ✅ | ✅ | 150 |
| 9 | up_time | 运行时间 | - | - | 120 |
| 10 | first_online_time | 第一次连接时间 | - | - | 150 |
| 11 | LASTINFORMTIME | 上次连接时间 | - | - | 150 |
| 12 | online_time | 接入时间 | ✅ | - | 140 |
| 13 | offline_time | 断开时间 | ✅ | - | 140 |
| 14 | group_name | 设备组 | ✅ | ✅ | min-120 |
| 15 | mac_address | MAC | ✅ | ✅ | 130 |
| 16 | sub_station_name | 站址名称 | 条件 | - | 120 |
| 17 | remark | 备注 | - | - | 185 |

#### 小区信息组 (cell) — 8 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | nr_cell_id | NR小区ID | - | - | 100 |
| 2 | PHYCELLID | PCI | ✅ | ✅ | 120 |
| 3 | tac | TAC | - | - | 80 |
| 4 | Band | 频段 | - | - | 140 |
| 5 | EARFCNULINUSE | NR频点上限 | - | - | 120 |
| 6 | EARFCNDLINUSE | NR频点下限 | - | - | 120 |
| 7 | tx_power | Tx Power | - | - | 80 |
| 8 | network_model | 基站指示 | - | - | 120 |

#### 状态信息组 (status) — 8 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | adminState | Admin状态 | - | - | 120 |
| 2 | op_state | gNB状态 | ✅ | ✅ | 120 |
| 3 | halob_flag | HaloB开关 | ✅ | ✅ | 120 |
| 4 | ue_count | UE数 | ✅ | ✅ | 80 |
| 5 | synStatus | 同步状态 | - | - | 120 |
| 6 | multiPlmnEnable | Multi PLMN状态 | - | - | 150 |
| 7 | rf_status | 射频开关状态 | - | - | 150 |
| 8 | amf_status | AMF Status | - | - | 140 |

#### 网络信息组 (network) — 2 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | cell_ip | IP地址 | ✅ | ✅ | 120 |
| 2 | IPSEC_ADDR | IPSEC地址 | - | - | 250 |

#### 位置信息组 (location) — 3 个

| # | field | 中文标签 | 默认勾选 | 锁定 | 宽度 |
|---|-------|---------|:--------:|:----:|------|
| 1 | gps_longitude | GPS经度 | - | - | 100 |
| 2 | gps_latitude | GPS纬度 | - | - | 90 |
| 3 | gps_height | GPS高度 | - | - | 70 |

#### 同步对话框扩展 (othersCol) — 7 个（仅在同步对话框中，不在主列表）

| # | field | 中文标签 |
|---|-------|---------|
| 1 | rollback_version | 回滚版本 |
| 2 | sas_param | SAS参数 |
| 3 | eu_ru | EU/RU数 |
| 4 | halob_license | HaloB License |
| 5 | energy_saving | 节能 |
| 6 | gnb_topo_cellmgr | gNB TOPO |
| 7 | ssl_cert_validity | SSL证书有效期 |

**5G NR 默认勾选汇总 (14个)**：serial_number, product, module_type, host_name, software_version, group_name, mac_address, online_time, offline_time, PHYCELLID, op_state, halob_flag, ue_count, cell_ip

**5G NR 锁定字段 (12个)**：serial_number, product, module_type, host_name, software_version, group_name, mac_address, PHYCELLID, op_state, halob_flag, ue_count, cell_ip


## 3. 统一设备列表 — 三制式字段适用性对照表

> 每个字段标注在哪些制式下有数据，不适用的制式显示为空（`-`）。

#### 公共字段 (common)

| key | 标题 | eNB | gNB | GSM | 默认显示 | 状态渲染 |
|-----|------|:---:|:---:|:---:|:--------:|:--------:|
| sn | SN | ✅ | ✅ | ✅ | ✅ | - |
| connStatus | 连接状态 | ✅ | ✅ | ✅ | ✅ | StatusIndicator |
| alarmLevel | 告警级别 | ✅ | ✅ | ✅ | ✅ | Tag(颜色) |
| hostName | 名称 | ✅(主机名) | ✅(站点名) | ✅(BSC名) | ✅ | - |
| networkType | 基站制式 | ✅ | ✅ | ✅ | ✅ | Tag(颜色) |
| productType | 产品类型 | ✅ | ✅ | ✅ | ✅ | - |
| deviceModel | 设备型号 | ✅ | ✅ | ✅ | ✅ | - |
| softwareVersion | 软件版本 | ✅ | ✅ | ✅ | ✅ | - |
| macAddress | MAC地址 | ✅ | ✅ | ✅ | ✅ | - |
| groupName | 设备组 | ✅ | ✅ | ✅ | ✅ | - |
| ipAddress | IP地址 | ✅ | ✅ | ✅ | ✅ | - |
| onlineTime | 接入时间 | ✅ | ✅ | - | ✅ | - |
| offlineTime | 断开时间 | ✅ | ✅ | - | ✅ | - |
| opState | 激活状态 | ✅ | ✅ | ✅(仅BTS) | ✅ | Tag(active/inactive) |
| ueCount | UE数 | ✅ | ✅ | ✅ | ✅ | - |
| rfStatus | 射频状态 | ✅ | ✅ | ✅(仅BTS) | ✅ | Tag(on/off) |
| syncStatus | 同步状态 | ✅ | ✅ | ✅(仅BTS) | ❌ | Tag(GPS同步/1588同步/未同步) |
| productName | 产品名称 | ✅ | ✅ | ✅ | ❌ | - |
| firmwareVersion | 固件版本 | ✅ | ✅(硬件版本) | ✅ | ❌ | - |
| onlineDuration | 累计时长 | ✅ | - | ✅ | ❌ | - |
| upTime | 运行时间 | ✅ | ✅ | ✅ | ❌ | - |
| firstOnlineTime | 首次连接 | ✅ | ✅ | ✅ | ❌ | - |
| lastInformTime | 上次连接 | ✅ | ✅ | ✅ | ❌ | - |
| lastOnlineTime | 最后在线 | ✅ | ✅ | ✅ | ❌ | - |
| siteName | 站址名称 | ✅ | ✅ | ✅ | ❌ | - |
| remark | 备注 | ✅ | ✅ | ✅ | ❌ | - |
| longitude | GPS经度 | ✅ | ✅ | ✅ | ❌ | - |
| latitude | GPS纬度 | ✅ | ✅ | ✅ | ❌ | - |
| gpsHeight | GPS高度 | ✅ | ✅ | ✅ | ❌ | - |
| gpsSatelliteCount | GPS卫星数 | ✅ | - | ✅ | ❌ | - |
| installAddress | 安装地址 | ✅ | - | - | ❌ | - |

#### eNB+gNB 共享字段

| key | 标题 | eNB | gNB | GSM | 状态渲染 |
|-----|------|:---:|:---:|:---:|:--------:|
| pci | PCI | ✅ | ✅ | - | - |
| tac | TAC | ✅ | ✅ | - | - |
| band | Band | ✅ | ✅ | - | - |
| dlEarfcn | DL频点 | ✅(EARFCN) | ✅(NR ARFCN) | - | - |
| ulEarfcn | UL频点 | ✅(EARFCN) | ✅(NR ARFCN) | - | - |
| networkModel | 基站指示 | ✅ | ✅ | - | - |
| txPower | Tx Power | ✅(CPE发射功率) | ✅ | - | - |
| halobFlag | HaloB | ✅(HaloX) | ✅(HaloB) | - | Tag(启用/禁用) |
| adminState | Admin State | ✅ | ✅ | - | Tag(激活/取消激活) |
| ipsecAddr | IPSec地址 | ✅ | ✅ | - | - |

#### eNB 独有字段

| key | 标题 | 状态渲染 |
|-----|------|:--------:|
| enbId | eNodeB ID | - |
| cellId | 小区ID | - |
| eci | ECI | - |
| plmnId | PLMN | - |
| subframeAssignment | 子帧配比 | - |
| specialSubframe | 特殊子帧配比 | - |
| rootIndex | 根序列索引 | - |
| siteId | Site ID | - |
| bandwidth | 带宽 | - |
| mmeStatus | MME状态 | Tag(已连接/未连接) |
| pmReportStatus | KPI上报状态 | - |
| cpeCount | CPE连接数 | - |
| lockStatus | 锁定状态 | Tag(锁定/未锁定) |
| wanSpeed | WAN状态 | - |
| serviceStatus | 状态 | - |
| validity | 有效期 | - |
| gpsVersion | GPS版本 | - |
| rom | ROM | - |
| mmepoolIpsecAddr | MME Pool IPSec | - |
| mechanicalDowntilt | 机械下倾角 | - |
| electronicDowntilt | 电子下倾角 | - |
| verticalBeamWidth | 垂直波束宽度 | - |
| horizontalAzimuth | 水平方位角 | - |

#### gNB 独有字段

| key | 标题 | 状态渲染 |
|-----|------|:--------:|
| gnbId | gNB ID | - |
| nrCellId | NR Cell ID | - |
| amfStatus | AMF状态 | Tag(已连接/未连接) |
| multiPlmnEnable | Multi PLMN | Tag(启用/禁用) |
| euCount | EU数 | - |
| ruCount | RU数 | - |
| rollbackVersion | 回退版本 | - |
| sasParam | SAS参数 | - |
| euRu | EU/RU数 | - |
| halobLicense | HaloB License | - |
| energySaving | Energy Saving | - |
| gnbTopoCellmgr | gNB TOPO | - |
| sslCertValidity | SSL证书有效期 | - |

#### GSM 独有字段

| key | 标题 | 状态渲染 |
|-----|------|:--------:|
| lac | LAC | - |
| arfcn | 频点 | - |
| uplinkFrequency | 上行频率 | - |
| downlinkFrequency | 下行频率 | - |
| bscLinkStatus | BSC连接状态 | Tag(连接/断开) |
| bscSelect | BSC Select | - |
| bscSerialNumber | 所属BSC编码 | - |
| btsNum | BTS数 | - |
| ipaUnitId | IPA Unit ID | - |
| omlRemoteIp | OML Remote IP | - |
| omlRemoteIpBak | OML Remote IP Bak | - |

## 4. 状态字段枚举值参考

> 基于原始 JSP 页面的格式化函数和条件渲染逻辑提取。

#### connStatus — 连接状态

| 前端值 | 后端值 | 中文 | 显示样式 | 适用制式 |
|--------|:------:|------|---------|:--------:|
| online | 1 | 在线 / 连接正常 | 绿色状态指示 | 全部 |
| offline | 0 | 离线 / 连接断开 | 灰色状态指示 | 全部 |
| - | 3 | 同步中 | - | 全部 |
| - | 2 | 同步失败 | - | 全部 |
| - | 4 | 初始化中 | - | 仅eNB |
| - | 5 | 远同步中 | - | 仅eNB |
| - | 6 | 远同步完成 | - | 仅eNB |

#### opState — 激活状态

| 前端值 | 后端值 | 中文 | Tag颜色 | 适用制式 |
|--------|:------:|------|---------|:--------:|
| active | 1 | 激活 | success(绿) | 全部 |
| inactive | 0 | 未激活 / 取消激活 | error(红) | 全部 |

> GSM 中 BSC 类型设备显示 `--`（不适用），仅 BTS 显示激活状态。
> 多小区设备(Intel_CR_CA/TC, MLN_CA, BM, QA_436Q_CA)支持逗号分隔多值如 `"1,0,1"`。

#### rfStatus — 射频开关状态

| 前端值 | 后端值 | 中文 | Tag颜色 | 适用制式 |
|--------|:------:|------|---------|:--------:|
| on | on / 1 | 射频开 | success(绿) | 全部 |
| off | off / 0 | 射频关 | error(红) | 全部 |
| - | -- | 不支持 | - | GSM(BSC) |

> 多小区设备支持逗号分隔如 `"on,off,on"`。

#### mmeStatus — MME状态（仅eNB）

| 前端值 | 后端值 | 中文 | Tag颜色 |
|--------|:------:|------|---------|
| connected | 1 / Connected | 已连接 | success(绿) |
| disconnected | 0 / Disconnected | 未连接 | error(红) |

#### amfStatus — AMF状态（仅gNB）

| 前端值 | 后端值 | 中文 | Tag颜色 |
|--------|:------:|------|---------|
| connected | 1 | 已连接 | success(绿) |
| disconnected | 0 | 未连接 | error(红) |

> 后端返回 JSON 数组字符串，解析后判断全部连接/全部断开/部分连接。

#### syncStatus — 同步状态

| 前端值 | 后端值 | 中文 | Tag颜色 | 适用制式 |
|--------|--------|------|---------|:--------:|
| synchronized | 1 | 已同步 | success(绿) | eNB+gNB |
| not synchronized | 0 | 未同步 | error(红) | eNB+gNB |
| GPS synchronized | GPS synchronized | GPS同步 | success(绿) | GSM(BTS) |
| 1588 synchronized | 1588 synchronized | 1588同步 | success(绿) | GSM(BTS) |
| REM synchronized | REM synchronized | REM同步 | success(绿) | GSM(BTS) |

#### halobFlag — HaloB/HaloX 开关

| 前端值 | 后端值 | 中文 | Tag颜色 | 适用制式 |
|--------|:------:|------|---------|:--------:|
| true | 1 | 已启用 | success(绿) | eNB+gNB |
| false | 0 | 已禁用 | default(灰) | eNB+gNB |

#### adminState — Admin 状态

| 前端值 | 后端值 | 中文 | Tag颜色 | 适用制式 |
|--------|:------:|------|---------|:--------:|
| active | 1 | 激活 | success(绿) | eNB+gNB |
| inactive | 0 | 取消激活 | error(红) | eNB+gNB |

#### multiPlmnEnable — Multi PLMN 状态（仅gNB）

| 前端值 | 后端值 | 中文 | Tag颜色 |
|--------|:------:|------|---------|
| enabled | 1 | 已启用 | success(绿) |
| disabled | 0 | 已禁用 | default(灰) |

#### lockStatus — 锁定状态（仅eNB）

| 前端值 | 后端值 | 中文 | Tag颜色 |
|--------|:------:|------|---------|
| locked | 1 | 锁定 | warning(橙) |
| unlocked | 0 | 未锁定 | success(绿) |

#### bscLinkStatus — BSC连接状态（仅GSM）

| 前端值 | 后端值 | 中文 | Tag颜色 |
|--------|:------:|------|---------|
| connected | 0 | 连接 | success(绿) |
| disconnected | 1 | 断开 | error(红) |

> 注意：BSC连接状态的后端值与其他状态相反（0=连接，1=断开）。

#### alarmLevel — 告警级别

| 前端值 | 后端值 | 中文 | Tag颜色 | 说明 |
|--------|:------:|------|---------|------|
| critical | 31001 | 紧急 | red | 最高级 |
| major | 31002 | 重要 | orange | |
| minor | 31003 | 次要 | gold | |
| warning | 31004 | 警告 | blue | |
| none | - | 无告警 | default(灰) | |

---

## 5. 统一设备列表实现方案

> 前端路径：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`

### 5.1 设计思路

将三个原始监控页面（LTE eNodeB、GSM、5G NR gNodeB）合并为一个统一的设备列表页面。通过 **基站制式（networkType）** 列区分 eNB / gNB / GSM，所有字段按归属制式分为五组。某制式不支持的字段在该制式设备行中显示为空。

### 5.2 字段分组方案

每列通过 `group` 属性标记归属：

| 分组 | group 值 | 说明 | 列数 | 默认显示 |
|------|----------|------|:----:|:--------:|
| 公共字段 | `common` | 三制式共有 + 状态字段 | 31 | 16 |
| eNB+gNB 共享 | `eNB+gNB` | LTE 和 5G NR 共有（GSM 显示空） | 10 | 0 |
| eNB 字段 | `eNB` | LTE 独有 | 23 | 0 |
| gNB 字段 | `gNB` | 5G NR 独有 | 13 | 0 |
| GSM 字段 | `GSM` | GSM 独有 | 11 | 0 |
| **合计** | | | **88** | **16** |

> 各分组的字段明细详见第 3 节「统一设备列表 — 三制式字段适用性对照表」。

### 5.3 列设置交互设计

| 功能 | 实现方式 |
|------|---------|
| 触发方式 | 工具栏设置图标 → 右侧 Drawer (380px) |
| 分组展示 | Ant Design Collapse 折叠面板，每组一个面板 |
| 搜索 | 顶部搜索输入框，实时过滤列名 |
| 分组全选 | 每组标题左侧 Checkbox（全选/半选/取消） |
| 计数显示 | 标题栏全局 `已选/总数`，每组右侧 Badge `n/m` |
| 拖拽排序 | @dnd-kit 实现组内列排序，拖拽手柄在左侧 |
| 持久化 | localStorage 分别存��列可见性和列顺序 |
| 重置 | 右上角重置按钮恢复默认可见性和排序 |
| 默认展开 | 公共字段组默认展开，其他组默认折叠 |

### 5.4 工具栏功能

从左到右依次为：

| # | 功能 | 图标 | 说明 |
|---|------|------|------|
| 1 | 锁定/解锁刷新 | 🔒/🔓 | 锁定时禁用刷新按钮，防止数据意外刷新 |
| 2 | 列设置 | ⚙️ | 打开 Drawer 管理列显示/排序 |
| 3 | 行高调整 | ≡ | 紧凑/默认/宽松三种密度 |
| 4 | 导出 | ⬇️ | Excel/CSV 导出 |
| 5 | 刷新 | 🔄 | 手动刷新数据（锁定时禁用） |

### 5.5 搜索项与筛选项

> 基于三制式 JSP 原始搜索/筛选功能逐项分析合并。

#### 5.5.1 三制式搜索项对比

| 维度 | eNB | gNB | GSM |
|------|-----|-----|-----|
| 搜索框 | ✅ 宽340px | ✅ 宽260px | ✅ 宽365px |
| like_fields | serial_number, host_name, cell_ip, mac_address, cell_identity, phycellid [, sub_station_name] | serial_number, host_name, cell_ip | serial_number, host_name, cell_ip, mac_address, cell_identity, phycellid |
| placeholder | SN / 主机名 / IP / MAC / ECI / PCI [/ 站址名称] | SN / 5G站点名 / IP | BSC编码 / BSC名称 / IP / MAC / 所属BSC编码 |
| 搜索字段数 | 6~7 | 3 | 6 |

#### 5.5.2 三制式筛选项对比

| # | 筛选项 | 参数名 | 类型 | eNB | gNB | GSM | 默认显示 |
|---|--------|--------|------|:---:|:---:|:---:|:--------:|
| 1 | 连接状态 | connection_status | 多选 | ✅ 7选项 | ✅ 4选项 | ✅ 4选项 | ✅ |
| 2 | 激活状态 | op_state | 单选 | ✅ | ✅ | ✅ | ✅ |
| 3 | 产品类型 | product_model | 多选/单选 | ✅ 多选(动态) | ✅ 单选(BaiBNX/BaiBNQ) | ✅ 多选(BSC/BTS) | ✅ |
| 4 | 设备型号名 | model_name | 多选 | ✅ (动态) | ✅ (动态) | ✅ (动态) | ❌ |
| 5 | 软件版本 | software_version | 多选 | ✅ (动态) | ✅ (动态) | ✅ (动态) | ❌ |
| 6 | 固件版本 | firmware_version | 多选 | ✅ (动态) | ✅ (动态) | ✅ (动态) | ❌ |
| 7 | 设备组 | group_id | 多选 | ✅ (动态) | ✅ (动态) | ✅ (动态) | ❌ |
| 8 | HaloB开关 | halob_flag | 单选 | ✅ | ✅ | ❌ | ❌ |
| 9 | MultiPLMN | multiPlmnEnable | 单选 | ❌ | ✅ | ❌ | ❌ |
| 10 | 所属BSC编码 | bscSerialnumber | 多选 | ❌ | ❌ | ✅ (动态) | ❌ |

**连接状态选项差异**：

| # | 选项 | 值 | eNB | gNB | GSM |
|---|------|:--:|:---:|:---:|:---:|
| 1 | 连接正常 | 1 | ✅ | ✅ | ✅ |
| 2 | 连接断开 | 0 | ✅ | ✅ | ✅ |
| 3 | 同步中 | 3 | ✅ | ✅ | ✅ |
| 4 | 同步失败 | 2 | ✅ | ✅ | ✅ |
| 5 | 初始化中 | 4 | ✅ | ❌ | ❌ |
| 6 | 远同步中 | 5 | ✅ | ❌ | ❌ |
| 7 | 远同步完成 | 6 | ✅ | ❌ | ❌ |

**动态加载 API**：

| 筛选项 | API | 参数差异 |
|--------|-----|---------|
| 产品类型 | `/cell/cpeinfos/getEnbMonitorProductList.action` | eNB 专用 |
| 设备型号名 | `/cell/cpeinfos/getModelNameList.action` | gNB: `isGnb=1`, GSM: `isGnb=0&isGSM=1` |
| 软件版本 | `/cell/cpeinfos/getCellVersionList.action` | GSM: `isGSM=1` |
| 固件版本 | `/cell/cpeinfos/getFirmwareVersionList.action` | gNB: `isGnb=1`, GSM: `isGnb=0&isGSM=1` |
| 设备组 | `/cell/cpeinfos/getDeviceGroupListByCell.action` | 三制式统一 |
| 所属BSC编码 | `/cell/cpeinfos/getBSCSnForBTSList.action` | GSM: `isGSM=1` |

#### 5.5.3 统一设备列表搜索与筛选实现

##### 搜索项

| 字段名 | 类型 | 说明 |
|--------|------|------|
| searchText | 输入框 (span=2) | 模糊搜索，覆盖 SN / 名称 / IP / MAC / ECI / PCI |

##### 筛选项

| # | 字段名 | 标签 | 类型 | 选项来源 | 适用范围 | 默��显示 |
|---|--------|------|------|---------|:--------:|:--------:|
| 1 | connStatus | 连接状态 | 多选 | 连接正常/断开/同步中/同步失败 | 三制式公共 | ✅ 第1行 |
| 2 | opState | 激活状态 | 单选 | 激活/未激活 | 三制式公共 | ✅ 第1行 |
| 3 | networkType | 基站制式 | 单选 | eNB/gNB/GSM | 统一列表特有 | ✅ 第1行 |
| 4 | productModel | 产品类型 | 多选 | eNB(动态)+gNB(BaiBNX/BaiBNQ)+GSM(BSC/BTS) | 三制式公共 | ✅ 第1行 |
| 5 | modelName | 设备型号 | 多选 | 动态(API) | 三制式公共 | ❌ 第2行 |
| 6 | softwareVersion | 软件版本 | 多选 | 动态(API) | 三制式公共 | ❌ 第2行 |
| 7 | firmwareVersion | 固件版本 | 多选 | 动态(API) | 三制式公共 | ❌ 第2行 |
| 8 | groupId | 设备组 | 多选 | 动态(API) | 三制式公共 | ❌ 第2行 |
| 9 | halobFlag | HaloB | 单选 | 是/否 | eNB+gNB | ❌ 第2行 |
| 10 | multiPlmnEnable | MultiPLMN | 单选 | 启用/禁用 | 仅gNB | ❌ 第2行 |
| 11 | bscSerialnumber | 所属BSC编码 | 多选 | 动态(API) | 仅GSM | ❌ 第2行 |

> FilterBar 配置 `collapsedRows={1}`，第1行显示搜索框+4个默认筛选项；展开后显示第2行的7个高级筛选项。

### 5.6 涉及的代码文件

| 文件 | 改动内容 |
|------|---------|
| `types/device.ts` | Device 接口扩展 ~70 个监控字段 |
| `services/api/deviceApi.ts` | BackendDevice 接口 + mapBackendDevice 映射 |
| `pages/device/DeviceList/index.tsx` | 84 列定义（含 group 属性）、筛选、格式化 |
| `mock/data/devices.ts` | 200 条模拟数据（区分 eNB/gNB/GSM） |
| `i18n/zh-CN/index.ts` | ~60 个 device.* 翻译键 |
| `i18n/en-US/index.ts` | 对应英文翻译 |
| `components/DataTable/index.tsx` | ColumnGroup 类型、列排序支持 |
| `components/DataTable/ColumnVisibility.tsx` | 折叠面板分组 + 搜索 + 拖拽排序 |
| `components/DataTable/Toolbar.tsx` | 锁定刷新按钮 |

---

## 6. 单元格交互行为对照表

> 对比三个原始 JSP 监控页面中每个字段的单元格交互行为（点击、弹窗、图标、格式化等），记录统一设备列表的实现状态。

### 6.1 已实现的交互

| 字段 | 原始行为 | 统一设备列表实现 |
|------|---------|---------------|
| **SN** (serial_number) | eNB/GSM: 纯文本; gNB: 纯文本 | ✅ 改进：可点击链接跳转设备详情页 |
| **连接状态** (connection_status) | 多状态图标(在线/离线/初始化/同步中/异常)，hover 显示同步时间 | ✅ StatusIndicator 组件(在线/离线) |
| **告警** (alarm) | 颜色圆形徽章(红/橙/黄/蓝)，**点击跳转告警 tab** | ✅ 彩色 Tag + 点击跳转 `?tab=alarm` |
| **主机名** (host_name) | 名称不匹配时显示⚠图标+同步弹窗 | ⚠ 仅纯文本 (TODO: 名称不匹配警告) |
| **制式** (networkType) | 原始无此列 | ✅ 新增列，彩色 Tag 区分三制式 |
| **IP 地址** (cell_ip) | **可点击**：`https://{ip}` 新窗口打开设备 Web UI | ✅ 蓝色链接，点击新窗口打开 |
| **激活状态** (op_state) | 多小区弹窗(激活/去激活列表)，License 过期⚠图标 | ✅ 多小区支持：汇总 Tag + [N/M] Popover 逐小区明细，兼容 "1,0,1" 和 "active,inactive" 格式 |
| **射频状态** (rf_status) | 多小区弹窗(开/关列表) | ✅ 多小区支持：汇总 Tag + [N/M] Popover 逐小区明细，兼容 "on,off,on" 和 "1,0,1" 格式 |
| **同步状态** (synStatus) | 多种同步状态文本，"未同步"红色 | ✅ 多状态 Tag，"未同步"加粗红色 |
| **UE 数** (ue_count) | eNB: -1/null→"--", 0→"0", >0 且非 CA 站可点击 Slide 面板(18 字段); gNB/GSM: 不可点击纯文本 | ✅ eNB 非 CA 站 >0 → Link 打开右侧 Drawer(`UeDetailDrawer`)显示 UE 列表；gNB/GSM 及 CA 站纯文本；436Q 自动切换 P/S 双载波列 |
| **KPI 上报** (pm_report_status) | off→关, normal→正常, broken→损坏(红色) | ✅ 彩色 Tag(关/正常/损坏) |
| **CPE 连接数** (cpe_connect) | -1/null→"--", 0→"0", **>0 可点击查看 CPE 详情** | ✅ 格式化 + 可点击链接跳转 `?tab=cpe` |
| **EU/RU 数** (euCount/ruCount) | "connected/total" 格式，**连接数<总数时红色** | ✅ 降级时红色文本 |
| **GPS 卫星数** (gps_satellite_count) | **有详情时可点击查看卫星信号表** | ✅ >0 时可点击链接跳转 `?tab=gps` |
| **HaloB** (halob_flag) | 1→绿色启用图标, 0→红色禁用图标 | ✅ 彩色 Tag(启用/禁用) |
| **Admin State** (adminState) | gNB: 1→Locked, 2→Unlocked, 3→ShuttingDown | ✅ Tag(Locked/Unlocked/ShuttingDown)，颜色 warning/success/error |
| **Multi PLMN** (multiPlmnEnable) | 0→禁用, 1→启用 | ✅ 彩色 Tag(启用/禁用) |
| **BSC 连接状态** (BscLinkStatus) | 0→未连接, 1→已连接 | ✅ 彩色 Tag(已连接/未连接) |
| **BSC Select** (BscSelect) | 0→"主"(Primary), 1→"备"(Backup) | ✅ 值映射中文显示 |
| **上行/下行频率** (uplink/downlinkFrequency) | 数值 + "MHz" 后缀 | ✅ 值 + MHz 后缀 |
| **锁定状态** (lock_status) | 可点击切换锁定/解锁，锁定原因弹窗 | ⚠ 仅 Tag，TODO: 交互式锁定操作 |
| **Remark 列头** | 可编辑列头名称，同步到所有页面 | ✅ 可编辑(EditOutlined + Input 确认/取消) |
| **MME 状态** (mme_status) | 多 MME 弹窗(IP/状态/PLMN 列表) | ✅ 多 MME 支持：汇总 Tag + [N/M] Popover 含 IP/PLMN 明细，兼容 JSON 数组和旧格式 "1"/"0" |
| **AMF 状态** (amf_status) | 多 AMF 弹窗(IP/状态/PLMN 列表) | ✅ 多 AMF 支持：汇总 Tag + [N/M] Popover 含 IP/PLMN 明细，兼容 JSON 数组和旧格式 |

### 6.2 待实现的复杂交互 (TODO)

以下交互需要后端 API 支持或复杂的前端组件，标记为后续迭代：

| 交互类型 | 原始行为 | 涉及字段 | 原始 API |
|---------|---------|---------|---------|
| **主���名同步** | ���备上报名称 ≠ OMC 名称时显示⚠图标，点击弹窗提示同步 | hostName | `syncCellName.action` |
| **GPS 坐标同步** | 设备上报 GPS ≠ OMC GPS 时显示⚠图标，点击弹窗提示同步 | longitude, latitude, gpsHeight | `syncGPSInfo.action` |
| ~~**多小区激活状态**~~ | ~~多小区设备显示汇总 + 可点击 `[N/M]` 弹窗逐小区显示~~ | ~~opState~~ | ✅ 已实现 `renderMultiCellStatus` |
| ~~**多小区射频状态**~~ | ~~同上，逐小区射频开/关弹窗~~ | ~~rfStatus~~ | ✅ 已实现 `renderMultiCellStatus` |
| ~~**多 MME 详情**~~ | ~~多个 MME IP/连接状态/PLMN 列表弹窗~~ | ~~mmeStatus~~ | ✅ 已实现 `renderMultiConnStatus` |
| ~~**多 AMF 详情**~~ | ~~多个 AMF IP/连接状态/PLMN 列表弹窗~~ | ~~amfStatus~~ | ✅ 已实现 `renderMultiConnStatus` |
| ~~**UE 详情面板**~~ | ~~eNB: >0 且非 CA 站点击打开右侧 Slide 面板(18字段+436Q变体); gNB/GSM: 不可点击~~ | ~~ueCount~~ | ✅ 已实现 `UeDetailDrawer` 右侧 Drawer 面板，436Q 自动切换 P/S 双载波列 |
| **CPE 详情面板** | 点击 CPE 数打开滑出面板 | cpeCount | `toUEDetailPage.action` |
| **卫星详情面板** | 点击卫星数打开面板，显示卫星号+信号强度 | gpsSatelliteCount | `getSatellitesDataList.action` |
| **锁定状态操作** | 点击切换锁定/解锁 + MAC 锁定对话框 | lockStatus | `cellModifyLockStatus.action` |
| **EARFCN→频率转换** | EARFCN 编号转为频率显示："38650(2300MHz)" | dlEarfcn | 前端查表转换 |
| **连接状态增强** | 初始化/同步中/异常等多状态图标 + hover 同步时间 | connStatus | `lastsyntime` 字段 |
| **HaloD 模式** | HaloD 关联设备弹窗（锁定/过滤/关联列表） | halobFlag | `queryHalodRelationInfo.action` |
| **有效期警告** | License 过期时在 opState 列显示红色⚠图标 | validity + opState | `delay_avaliable` 字段 |

#### 6.2.1 eNB UE 详情面板字段（仅 eNB 非 CA 站可触发）

> 原始 API: `POST /system/device/enb/uedata/getENBUeStatisticsDataList.action`，参数 `{enb_code}`
> 原始 JSP: 右侧 Slide 面板（ueslide）展示 UE 列表

| # | 字段 | 属性名 | 宽度 | 说明 | 436Q 变体 |
|---|------|--------|------|------|----------|
| 1 | UEID | ue_id | 100 | 用户设备 ID | — |
| 2 | IMSI | imsi | 150 | 国际移动订户身份码 | — |
| 3 | VMAC | vmac | 130 | 虚拟 MAC 地址 | — |
| 4 | CPE名称 | cpe_name | 150 | 客户端设备名称 | — |
| 5 | 下行吞吐速率 | downlink_rate | 140 | Mbps | — |
| 6 | 上行吞吐速率 | uplink_rate | 135 | Mbps | — |
| 7 | IP地址 | ip | 140 | 用户 IP | — |
| 8 | 端口 | port | 80 | 通信端口 | — |
| 9 | 上行SINR | ulsinr | 100 | dB | — |
| 10 | 下行CQI | dlcqi | 100 | 信道质量指示 | 436Q: P_Dlcqi (p_dlcqi) + S_Dlcqi (s_dlcqi) |
| 11 | 上行MCS | ulmcs | 100 | ���制编码方案 | — |
| 12 | 下行MCS | dlmcs | 100 | 调制编码方案 | 436Q: P_Dlmcs (p_dlmcs) + S_Dlmcs (s_dlmcs) |
| 13 | 发送功率 | txpower | 100 | dBm | — |
| 14 | 上行BLER | uplink_bler | 120 | 块错误率(%) | — |
| 15 | 下行BLER | downlink_bler | 165 | 块错误率(%) | 436Q: P_TB1/P_TB2/S_TB1/S_TB2 四列 |
| 16 | 路径损耗 | pathloss | 120 | dBm | — |
| 17 | UE_S1AP_ID | ue_s1ap_id | 120 | S1-AP 标识（可选） | — |
| 18 | MME_S1AP_ID | mme_s1ap_id | 120 | MME S1-AP 标识（可选） | — |

> **注**: 436Q 平台（QA_436Q_*）使用 Primary/Secondary Carrier 双载波列替换单列 CQI/MCS/BLER。
> gNB 和 GSM 的 UE 数不提供点击跳转，无详情面板。

### 6.3 操作列菜单对照

三个页面的行级操作菜单项（原始通过右键上下文菜单实现，统一设备列表通过 `⋯` 下拉菜单实现）：

| 操作 | eNB | gNB | GSM | 权限码 | 统一列表 |
|------|:---:|:---:|:---:|-------|---------|
| 同步 | ✅ | ✅ | ✅ | `CODE_ENB_SYNCHRONIZE` / `CODE_GNB_SYNCHRONIZE` | ✅ sync |
| TR069 报文采集 | ✅ | ✅ | ✅ (超级用户) | `CODE_ENB_TR069_MSG_EXCHANGE` / `CODE_GNB_TR069_MSG_EXCHANGE` | ✅ tr069Collect |
| 重启 | ✅ | ✅ | ✅ | `CODE_ENB_REBOOT` / `CODE_GNB_REBOOT` | ✅ reboot |
| 恢复默认配置 | ✅ | — | ✅ | — | ✅ resetConfig |
| 激活/去激活 | ✅ (多小区) | ✅ (逐小区) | ✅ (多小区 CA) | `CODE_ENB_ACTIVE` / `CODE_GNB_ACTIVE` | ✅ activate (多小区子菜单) |
| 射频开/关 | ✅ (多小区) | ✅ (逐小区) | — | `CODE_ENB_RF_ENABLE` / `CODE_GNB_RF_ENABLE` | ✅ rf (多射频子菜单) |
| HaloB 开/关 | ✅ | ✅ | — | 动态检测 | ✅ halob |
| 日志采集 | ✅ | ✅ | ✅ | `CODE_ENB_LOGS` / `CODE_GNB_LOGS` | ✅ logCollect |

### 6.4 行级操作点击反馈行为详细对照

> 从三个原始 JSP 逐操作提取的点击后反馈动作（确认弹窗、API 调用、成功/失败通知、特殊逻辑），以及统一设备列表的实现状态。

#### 6.4.1 同步 (sync)

| 维度 | eNB | gNB | GSM | 统一列表 |
|------|-----|-----|-----|---------|
| **触发函数** | `openSyncDialog()` | `openSyncDialog()` | `vm.openSyncDialog()` | `handleRowAction('sync')` |
| **点击行为** | 弹出同步参数弹窗，加载 `toSyncParamsPage.action` | 弹出同步参数弹窗（基础+高级参数） | 弹出同步参数弹窗（含 GPS/1588/REM） | ⚠ TODO: 接入同步参数弹窗 |
| **API** | `cell/param/refreshCellInfo.action` | `cell/param/refreshCellInfo.action` + `batchSyncCell.action` | 同 eNB | — |
| **参数** | `smallCellCode`, `selectedParams` | `smallCellCode`, `selectedParams`, `isGnb=1` | `cell_code` | — |
| **成功反馈** | 弹窗自动关闭 | 弹窗自动关闭 | 弹窗自动关闭 | — |
| **前置条件** | 必须在线 | 必须在线 | 必须在线 | ✅ 离线禁用 |

#### 6.4.2 报文采集 (tr069Collect)

| 维度 | eNB | gNB | GSM | 统一列表 |
|------|-----|-----|-----|---------|
| **触发函数** | `vm.showCollectMessage()` | `showCollectMessage()` | `vm.showCollectMessage()` | `handleRowAction('tr069Collect')` |
| **点击行为** | ①检查是否有其他设备在采集 → ②弹出采集时长选择弹窗（5/10分钟） | 同 eNB | 同 eNB（需超级用户） | ⚠ TODO: 接入检查+时长弹窗 |
| **检查 API** | `trace/isExistTracingDevice.action` | 同 eNB | 同 eNB | — |
| **执行 API** | `trace/start.action` | 同 eNB | 同 eNB | — |
| **参数** | `deviceCode`, `serialNumber`, `type='enb'`, `collectInterval` | `type='gnb'` 其余同 eNB | 同 eNB | — |
| **成功反馈** | "成功"提示 + 计时器启动 | "成功"提示 + 计时器启动 | "成功"提示 | ✅ "报文正在收集" |
| **失败反馈** | "SN=xxx正在收集" (已有采集) | 同 eNB | 同 eNB | — |
| **前置条件** | 必须在线；非双载波 | 必须在线；非双载波 | 必须在线 + 超级用户 | ✅ 离线禁用 |

#### 6.4.3 重启 (reboot)

| 维度 | eNB | gNB | GSM | 统一列表 |
|------|-----|-----|-----|---------|
| **触发函数** | `cellReboot()` | `goReboot()` | `cellReboot()` | `handleRowAction('reboot')` |
| **点击行为** | 弹出**危险确认弹窗**（红色样式）"确定重启设备吗？" | 弹出确认弹窗 | 弹出确认弹窗 | ✅ `modal.confirm` 危险样式 |
| **API** | `cell/cpeinfos/cellReboot.action` | 同 eNB + `isGnb=1` | 同 eNB | ⚠ TODO: 接入 API |
| **参数** | `cell_code` | `cell_code`, `isGnb=1` | `cell_code` | — |
| **成功反馈** | "命令已下发" 提示 | "命令已下发" 提示 | 提示通知 | ✅ "命令已下发" |
| **失败反馈** | 服务端错误消息 | 服务端错误消息 | 服务端错误消息 | — |
| **前置条件** | 必须在线 | 必须在线 | 必须在线 | ✅ 离线禁用 |

#### 6.4.4 恢复默认配置 (resetConfig)

| 维度 | eNB | gNB | GSM | 统一列表 |
|------|-----|-----|-----|---------|
| **触发函数** | `configReset()` | — | `configReset()` | `handleRowAction('resetConfig')` |
| **点击行为** | 弹出**危险确认弹窗**"确定恢复默认配置吗？" | 不支持 | 同 eNB | ✅ `modal.confirm` 危险样式 |
| **API** | 未明确 (configReset action) | — | 同 eNB | ⚠ TODO: 接入 API |
| **成功反馈** | 无明确成功提示 | — | 同 eNB | ✅ "命令已下发" |
| **前置条件** | 必须在线 | — | 必须在线 | ✅ 离线禁用 |

#### 6.4.5 激活/去激活 (activate)

| 维度 | eNB | gNB | GSM | 统一列表 |
|------|-----|-----|-----|---------|
| **触发函数** | `activeOpStatus()` | `goMultActive()` | `activeOpStatus()` | `handleRowAction('activate_N')` |
| **点击行为** | **单小区**: 直接切换。**多小区(CA)**: 展开为 Cell1/Cell2/Cell3 子菜单，逐小区切换。**PM-B4860**: 先弹出 slot 选择弹窗 | 展开为 Cell1/Cell2/Cell3 子菜单 | 同 eNB (支持 CA 多小区) | ✅ 单小区→确认弹窗；多小区→子菜单逐 Cell |
| **API** | `cell/cpeinfos/cellModifyActiveStatus.action` | 同 eNB | 同 eNB | ⚠ TODO: 接入 API |
| **参数** | `op_state`(0/1), `small_cell_code`, `cellNumber` | 同 eNB | `+isGsm='1'` | — |
| **成功反馈** | "下发成功" + 刷新列表 | "下发成功" + 刷新列表 | "成功" + 刷新列表 | ✅ "命令已下发" |
| **失败反馈** | 服务端错误消息 | 服务端错误消息 | 服务端错误消息 | — |
| **特殊逻辑** | 位置变更时(isSystemDeActivation)显示额外警告弹窗 | — | — | — |
| **前置条件** | 必须在线 | 必须在线 | 必须在线 | ✅ 离线禁用 |

#### 6.4.6 射频开/关 (rf)

| 维度 | eNB | gNB | GSM | 统一列表 |
|------|-----|-----|-----|---------|
| **触发函数** | `setRFStatus()` | `setRFStatus()` | — | `handleRowAction('rf_N')` |
| **点击行为** | **单射频**: 直接切换 on↔off。**多射频**: 展开为 RF1/RF2/RF3 子菜单逐 RF 切换 | 同 eNB (逐小区 RF 控制) | 不支持 | ✅ 单射频→确认弹窗；多射频→子菜单逐 RF |
| **API** | `cell/cpeinfos/cellModifyRadioStatus.action` | 同 eNB | — | ⚠ TODO: 接入 API |
| **参数** | `smallCellCode`, `radioStatus`(on/off), `cellNumber` | 同 eNB | — | — |
| **成功反馈** | "下发成功" + 刷新列表 | "下发成功" + 刷新列表 | — | ✅ "命令已下发" |
| **失败反馈** | "设置失败" | 服务端错误消息 | — | — |
| **前置条件** | 必须在线；RF 状态非空/非未知 | 必须在线；RF 状态有效 | — | ✅ 离线禁用 |

#### 6.4.7 HaloB 开/关 (halob)

| 维度 | eNB | gNB | GSM | 统一列表 |
|------|-----|-----|-----|---------|
| **触发函数** | `openCloseHalob()` | `openCloseHalob()` | — | `handleRowAction('halob')` |
| **点击行为** | 弹出**警告确认弹窗**"参数修改需要重启基站" | 同 eNB | 不支持 | ✅ `modal.confirm` 危险样式 + 重启警告 |
| **API** | `cell/cpeinfos/setCellHalobSwitch.action` | 同 eNB | — | ⚠ TODO: 接入 API |
| **参数** | `cell_code`, `halob_switch`(0=开,1=关) | 同 eNB | — | — |
| **成功反馈** | 弹窗关闭（无明确成功提示） | 弹窗关闭 | — | ✅ "命令已下发" |
| **特殊逻辑** | 集中化模式下不显示 | 同 eNB | — | — |
| **前置条件** | 必须在线；需 HaloB License | 必须在线；需 HaloB License | — | ✅ 离线禁用 |

#### 6.4.8 日志收集 (logCollect)

| 维度 | eNB | gNB | GSM | 统一列表 |
|------|-----|-----|-----|---------|
| **触发函数** | `confirmImmediateCollectLogFile()` | `goLogs()` | `gsmConfirmImmediateCollectLogFile()` | `handleRowAction('logCollect')` |
| **点击行为** | **直接下发**（无确认弹窗），立即发送采集请求 | **直接下发** | **直接下发** | ✅ 直接提示"日志正在收集" |
| **API** | `cell/collect/goImmediateCollectLogFile.action` | 同 eNB + `isGnb=1` | 同 eNB | ⚠ TODO: 接入 API |
| **参数** | `serial_number`, `device_code`, `device_type='eNB'`, `execute_type='Immediately'`, `timeZone` | `+isGnb=1` | 同 eNB | — |
| **成功反馈** | "日志正在收集" | "gNB日志收集提示"(i18n) | "日志正在收集" toast | ✅ "日志正在收集" |
| **失败反馈** | 服务端错误消息 toast | 服务端错误消息 | 服务端错误消息 toast | — |
| **前置条件** | 必须在线；非双载波 | 必须在线 | 必须在线 | ✅ 离线禁用 |

### 6.5 eNB 特有操作（统一列表暂不纳入）

以下操作仅在 eNB 原始页面存在，暂未整合到统一列表：

| 操作 | 触发函数 | 行为 | 前置条件 |
|------|---------|------|---------|
| SAS 注册 | `registerSas()` | 确认弹窗 → 发送注册请求 | SAS 开关启用 (sasSwitch=="1") |
| SAS 取消注册 | `deregisterSAS()` | 确认弹窗 → 发送注销请求 | SAS 已注册 |
| SAS 强制/自动 RF | `setForceRFStatus()` | 切换 SAS 自动控制 ↔ 强制关闭 RF | SAS 启用 (sasEnable=="on") |
| STUN 重启 | `vm.restartEnb()` | 通过 STUN 通道发送重启 (stunReboot) | stun_reboot==1 + 超级用户 |
| 有效期/流量限制 | `setEffectPeriod()` / `setLimitation()` | 打开配置滑出面板 | eNB 独有功能 |
| 设备信息 | `goCellDetailParamInfoWin()` | 打开设备参数信息滑出面板 | 任何状态可用 |
| 设备设置 | `jumpToSetting()` / `goSettingPanel()` | 打开设置表单（不同平台不同处理） | 任何状态可用 |

---

## 7. 各监控页面产品类型分析

### 7.1 LTE eNodeB 监控页面

LTE 页面使用两个字段标识产品：`product`（产品编号）和 `platformType`（平台类型）。

#### product 取值

| 产品编号 | 说明 | 特殊处理 |
|---------|------|---------|
| `PM-B4860` | NXP 芯片产品 | 独立的激活/取消激活逻辑、NXP 专用接口 |
| `QAFA` | QA 系列 A 型 | MME 状态列表独立显示逻辑 |
| `QATA` | QA 系列 T/A 型 | 同 QAFA |
| `QAFB` | QA 系列 B 型 | 同 QAFA |
| `RTD` | 辅站标识 | 双载波辅站不可选中 |
| 其他 | 通用产品 | 标准显示逻辑 |

#### platformType 取值

| 平台类型 | 芯片平台 | 小区配置 | 说明 |
|---------|---------|---------|------|
| `Intel_CR` | Intel | 基础 | 英特尔 CR 基础版 |
| `Intel_CR_CA` | Intel | 双小区(CA) | 支持载波聚合，多小区激活显示 |
| `Intel_CR_TC` | Intel | 三小区(TC) | 三小区配置 |
| `Intel_CR_SC` | Intel | 单小区(SC) | 单小区配置 |
| `Intel_CR_DC` | Intel | 双载波(DC) | 双载波辅波特殊处理 |
| `MLN` | MLN | 基础 | 美联平台基础版 |
| `MLN_CA` | MLN | 双小区(CA) | 多小区激活显示 |
| `MLN_SC` | MLN | 单小区(SC) | 单小区配置 |
| `MLN_DC` | MLN | 双载波(DC) | 双载波辅波特殊处理 |
| `BM` | BM | 多制式 | LTE+GSM 多制式，Cell ID/RF 状态/UE 数独立逻辑 |
| `QA_436Q_CA` | QA 436Q | 双小区(CA) | 高通芯片，扫描功能 |
| `QA_436Q_SC` | QA 436Q | 单小区(SC) | 高通芯片 |
| `QA_436Q_DC` | QA 436Q | 双载波(DC) | 双载波辅波特殊处理 |
| `NEU430_DC` | NEU 430 | 双载波(DC) | 扫描功能，双载波辅波处理 |
| `BAIBLQ` | BAIBLQ | - | 扫描功能，设置面板跳转 |
| `MLQ` | MLQ | - | 扫描功能，设置面板跳转 |

#### 产品类型影响的功能差异

| 功能 | 受影响的 platformType / product | 差异行为 |
|------|-------------------------------|---------|
| 激活状态显示 | Intel_CR_CA, Intel_CR_TC, MLN_CA, BM, PM-B4860 | 多小区分别显示激活状态 |
| MME 状态列 | QAFA, QATA, QAFB | 独立的 MME 列表渲染 |
| PLMN 显示 | Intel_CR 系列, MLN 系列, BM | 条件显示 PLMN 信息 |
| UE 数显示 | BM | 独立的 UE 计数逻辑 |
| Cell ID 显示 | BM | 多制式 Cell ID 显示 |
| 扫描功能 | QA_436Q_CA/SC/DC, NEU430_DC, BAIBLQ, MLQ | 支持扫描操作 |
| 双载波辅波 | Intel_CR_DC, MLN_DC, QA_436Q_DC, NEU430_DC | 辅波限制操作，仅射频可用 |
| 设置面板 | 436Q, BAIBLQ, MLQ | 独立的设置面板跳转 |
| 行选中限制 | RTD (辅站) | 双载波辅站不可选中 |

---

### 7.2 GSM 监控页面

GSM 页面同时使用 `product` 和 `platformType`，但产品类型较少。

#### product 取值

| 产品编号 | 说明 | 特殊处理 |
|---------|------|---------|
| `BSC` | 基站控制器 | 状态列显示 `--`（不可操作） |
| `BTS` | 基站收发台 | 标准 GSM 设备，完整状态显示 |
| `PM-B4860` | NXP 芯片产品 | 独立激活逻辑，NXP 专用接口 |
| `RTD` | 辅站标识 | 双载波辅站不可选中 |

#### platformType 取值

GSM 页面复用了 LTE 的 platformType 判断逻辑：

| 平台类型 | 说明 |
|---------|------|
| `Intel_CR_CA` | 多小区激活状态显示 |
| `Intel_CR_TC` | 多小区激活状态显示 |
| `MLN_CA` | 多小区激活状态显示 |
| `QA_436Q_CA` | 独立的多小区激活显示 |
| `436Q` | 设置面板跳转条件 |
| `BAIBLQ` | 设置面板跳转条件 |
| `MLQ` | 设置面板跳转条件 |

#### 筛选默认值

- 默认筛选条件：`product_model = 'BSC,BTS'`（即默认查看 BSC 和 BTS 两种设备）

---

### 7.3 5G NR gNodeB 监控页面

5G NR 页面产品类型最简单，仅两种产品。**不使用 platformType 字段**。

#### product 取值

| 产品编号 | 说明 | 特殊处理 |
|---------|------|---------|
| `BaiBNX` | 5G gNodeB X 型 | 标准 5G 设备 |
| `BaiBNQ` | 5G gNodeB Q 型 | AMF 状态列独立显示逻辑 |

#### 筛选下拉选项

```
全部 | BaiBNX | BaiBNQ
```

- 产品类型标识（`product`）和设备型号名（`module_type`）均为锁定字段（disabled: true）
- `BaiBNQ` 特有 AMF Status 列的条件渲染


### 7.4 各制式行级操作（"执行"菜单）按产品类型对比

> 基于三个 JSP 中 `optClick()` 函数逐条件分析，仅列出**当前菜单构建中实际生效**的操作项（不含残留 handler 代码）。
> 排除"设置"操作。

#### 7.4.1 eNB (LTE) 行级操作

LTE 操作受 `product`、`platformType`、`dualCarrierType`、`have_connected` 多维度影响，是三制式中最复杂的。

##### 通用操作（所有在线 eNB 设备的基线）

| 操作组 | 操作项 | 权限码 | 离线禁用 |
|--------|--------|--------|:--------:|
| Group1 同步/报文 | 同步 | `CODE_ENB_SYNCHRONIZE` | ✅ |
| | 收集报文 (TR069) | `CODE_ENB_TR069_MSG_EXCHANGE` | - |
| Group2 重启 | 重启 | `CODE_ENB_REBOOT` | ✅ |
| Group3 激活/射频/HaloB | 激活/去激活 | `CODE_ENB_ACTIVE` | ✅ |
| | 射频 开/关 | `CODE_ENB_RF_ENABLE` | ✅ |
| | HaloB 开/关 | `CODE_ENB_HALOB_ENABLE` | ✅ |
| Group4 日志 | 日志收集 | `CODE_ENB_LOGS` | ✅ |

##### 产品/平台类型差异

| 产品/平台条件 | 操作差异 |
|--------------|---------|
| **PM-B4860** | 激活操作打开**板卡/槽位多 Cell 选择对话框**（非简单切换），其余不变 |
| **Intel_CR_CA / Intel_CR_TC / MLN_CA** | 激活操作变为**子菜单**：`Cell1 激活/去激活`, `Cell2 激活/去激活`, ... |
| **BM** (LTE+GSM 双模) | 激活子菜单包含 **LTE Cell + GSM Cell**：`LTE Cell1 激活`, `GSM Cell1 激活`, ... |
| **Intel_CR_DC / MLN_DC + dualCarrier=2** (双载波子站) | Group3 缩减为仅 **激活**(id:411) + **射频**(单Cell)；**无 HaloB**；Group1 **无收集报文** |
| **QA_436Q_CA/SC/DC / NEU430_DC + dualCarrier=2** (子设备) | Group1 **无收集报文**；其余保持基线 |
| **QA_436Q_DC + SN 末尾 `-2`** | **激活操作隐藏** (`opStateShowFlag=false`) |
| **have_connected=2** (从未连接) | **所有操作组清空**，全部禁用 |

##### 附加操作（不在主菜单分组，由全局配置控制）

| 操作 | 条件 |
|------|------|
| SAS 注册 | `sasSwitch == "1"` (全局开关) |
| SAS 注销 | `sasSwitch == "1"` |
| 强制关闭 RF / SAS 自动 RF | `CODE_ENB_SAS_RF_ENABLE` + `rfForceShowFlag` |
| 配置恢复 | `CODE_ENB_RESET_CONFIG` (在 Maintenance 分类) |

---

#### 7.4.2 gNB (5G NR) 行级操作

5G NR **不按产品类型区分操作**。BaiBNX 和 BaiBNQ 的操作菜单完全一致。

| 操作组 | 操作项 | 权限码 | 离线禁用 |
|--------|--------|--------|:--------:|
| 同步/报文 | 同步 | `CODE_GNB_SYNCHRONIZE` | ✅ |
| | 收集报文 | `CODE_GNB_TR069_MSG_EXCHANGE` | - |
| 重启 | 重启 | `CODE_GNB_REBOOT` | ✅ |
| 激活 | 激活/去激活 (按 Cell 子菜单) | `CODE_GNB_ACTIVE` | ✅ |
| 射频 | 射频 开/关（单 Cell 或多 Cell 子菜单） | `CODE_GNB_RF_ENABLE` | ✅ |
| HaloB | HaloB 开/关 | 由 License AJAX 检查控制 | ✅ |
| 日志 | 日志收集 | `CODE_GNB_LOGS` | ✅ |

| 条件 | 差异 |
|------|------|
| `dualCarrierType=2` | 收集报文隐藏 |
| HaloB 集中管理模式 | HaloB 操作项隐藏 |
| `DeviceLogView='0'` | 日志收集整体隐藏 |

---

#### 7.4.3 GSM 行级操作

GSM **不按产品类型区分操作**。BSC 和 BTS 的操作菜单一致（Group3 始终为空）。

| 操作组 | 操作项 | 权限码 | 离线禁用 |
|--------|--------|--------|:--------:|
| Group1 同步/报文 | 同步 | `CODE_ENB_SYNCHRONIZE` | ✅ |
| | 收集报文 | 仅超级用户 (`is_super_user`) | - |
| Group2 重启 | 重启 | `CODE_ENB_REBOOT` | ✅ |
| Group3 | **空**（无激活/射频/HaloB） | - | - |
| Group4 日志 | 日志收集 | `CODE_ENB_LOGS` | ✅ |

| 条件 | 差异 |
|------|------|
| `have_connected=2` | 所有操作组清空 |

---

#### 7.4.4 三制式行级操作汇总矩阵

| 操作 | eNB 通用 | eNB DC子站 | eNB CA多载波 | eNB BM双模 | eNB PM-B4860 | gNB | GSM |
|------|:--------:|:---------:|:-----------:|:---------:|:-----------:|:---:|:---:|
| 同步 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 收集报文 | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅* |
| 重启 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 激活/去激活 | ✅ 单项 | ✅ 单项 | ✅ 多Cell子菜单 | ✅ LTE+GSM子菜单 | ✅ 板卡对话框 | ✅ 多Cell子菜单 | ❌ |
| 射频 开/关 | ✅ | ✅ 仅单Cell | ✅ | ✅ | ✅ | ✅ | ❌ |
| HaloB 开/关 | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ |
| 日志收集 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

> `✅*` GSM 的收集报文仅超级用户可见。
> `eNB DC子站` 指 Intel_CR_DC / MLN_DC + dualCarrier=2。
> `eNB CA多载波` 指 Intel_CR_CA / Intel_CR_TC / MLN_CA。

---

### 7.5 批量操作（工具栏）对比

> 基于三个 JSP 工具栏区域逐按钮分析，列出需要选中行的批量操作和不需选中的全局操作。

#### 7.5.1 三制式批量操作汇总

| # | 操作 | eNB (LTE) | gNB (5G NR) | GSM | 需选中行 |
|---|------|:---------:|:-----------:|:---:|:--------:|
| 1 | **导出** | ✅ 全参数导出配置页 | ✅ CSV/XLSX + License 导出 | ✅ CSV/XLSX + License 导出 | ❌ |
| 2 | **移动到设备组** | ✅ `CODE_ENB_MONITOR` | ❌ | ✅ | ✅ |
| 3 | **同步** | ✅ `CODE_ENB_SYNCHRONIZE` | ✅ `CODE_GNB_SYNCHRONIZE` | ✅ `CODE_ENB_SYNCHRONIZE` | ✅ |
| 4 | **重启** | ✅ `CODE_ENB_REBOOT` | ✅ `CODE_GNB_REBOOT` | ✅ `CODE_ENB_REBOOT` | ✅ |
| 5 | **回收站** | ✅ `CODE_ENB_DEVICE_REGISTER` | ✅ `CODE_GNB_DEVICE_REGISTER` | ❌ | ✅ |
| 6 | **添加基站** | ✅ `CODE_ENB_DEVICE_REGISTER` | ❌ | ❌ (已注释) | ❌ |

#### 7.5.2 各操作交互流程对比

##### 移动到设备组

| 维度 | eNB | GSM | gNB |
|------|-----|-----|-----|
| 支持 | ✅ | ✅ | ❌ |
| 弹窗类型 | Modal 对话框 (620px) | Modal 对话框 (620px) | - |
| 交互流程 | 分页表格列出设备组 → 单选目标组 → 确定 | 同 eNB | - |
| 验证 | 未选择时提示 "请选择设备组。" | 同 eNB | - |
| API | `POST /system/deviceGroup/moveCellToGroup.action` | 同 eNB | - |
| 参数 | `toGroupId` + `ids`(逗号分隔 small_cell_code) | 同 eNB | - |
| 成功反馈 | "成功" toast | "成功" toast | - |
| 失败反馈 | error toast 显示服务端消息 | 同 eNB | - |

##### 同步

| 维度 | eNB | gNB | GSM |
|------|-----|-----|-----|
| 弹窗类型 | Modal 对话框 | Modal 对话框 (660px) | Modal 对话框 |
| 交互流程 | 复选框选择同步参数 → 确定 | 同 eNB | 同 eNB |
| 参数分组 | 告警(活动告警) + 基础配置(~25项) + 高级配置(~20项) | 告警 + 基础配置(~17项) + 高级配置(~7项) | 告警 + 基础配置(~7项) + BSC(1项) + BTS(~15项) |
| 默认选中 | 告警 ✅ + 部分基础配置 | 告警 ✅ + 部分基础配置 | 告警 ✅ |
| 全选功能 | ✅ 分组全选 + 总全选 | ✅ 分组全选 + 总全选 | ✅ 分组全选 + 总全选 |
| API | `POST /cell/quicksettings/batchSyncCell.action` | 同 eNB + `isGnb=1` | 同 eNB |
| 参数 | `smallCellCode` + `selectedParams` | 同 eNB | 同 eNB |
| 成功反馈 | 静默关闭对话框 | 静默关闭对话框 | 静默关闭对话框 |
| 失败反馈 | 页面顶部红色滑动横幅(5秒) | 同 eNB | 同 eNB |

##### 重启

| 维度 | eNB | gNB | GSM |
|------|-----|-----|-----|
| 弹窗类型 | 确认对话框 (warning) | 确认对话框 (warning) | 确认对话框 (warning) |
| 确认文案 | "确定重启设备？" | "确定重启设备？" | "确定重启设备？" |
| API | `POST /task/reboot/batchRebootCell.action` | 同 eNB + `isGnb=1` | 同 eNB |
| 参数 | `cellCodes`(逗号分隔) | `cellCodes` + `isGnb=1` | `cellCodes`(逗号分隔) |
| 成功反馈 | "命令已经下发。" 提示(API调用前即显示) | "成功" toast | "命令已经下发。" 提示 |
| 失败反馈 | 页面顶部红色滑动横幅 | error toast | 页面顶部红色滑动横幅 |

##### 回收站

| 维度 | eNB | gNB | GSM |
|------|-----|-----|-----|
| 支持 | ✅ | ✅ | ❌ |
| 弹窗类型 | 警告确认对话框 (warning + HTML) | 同 eNB | - |
| 确认文案(第一行) | "确认将设备移入回收站？" | 同 eNB | - |
| 确认文案(第二行) | "回收站的设备，将不进行数据监控（监控数据、配置、警报、KPI等...）" | 同 eNB | - |
| API | `POST /recycle/moveDeviceToRecycle.action` | 同 eNB + `?isGnb=1` | - |
| 参数 | `smallCellCodeStr`(逗号分隔) | 同 eNB | - |
| 成功反馈 | "成功" toast | "成功" toast | - |
| 失败反馈 | error toast | error toast | - |

##### 导出（独立工具栏按钮，不在批量操作中）

| 维度 | eNB | gNB | GSM |
|------|-----|-----|-----|
| 交互方式 | 打开导出配置页面 | 下拉菜单 | 下拉菜单 |
| 格式选项 | 参数列选择后导出 | CSV / XLSX | CSV / XLSX |
| License 导出 | - | ✅ 可选 | ✅ 可选 |
| 云管理员模式 | - | - | ✅ 运营商选择 + 格式选择 |
| 需选中行 | ❌ (导出全量) | ❌ | ❌ |

##### 添加基站（独立按钮，不在批量操作中）


- 仅 eNB 页面有此按钮（gNB 无，GSM 已注释）
- 统一设备列表中不再提供添加基站入口（改为导出按钮）

#### 7.5.3 统一设备列表批量操作实现

| # | 操作 | 图标 | 交互反馈 | 显示条件 | 状态 |
|---|------|------|---------|---------|:----:|
| 1 | 移动到设备组 | `SwapOutlined` | Modal 弹窗：设备组分页表格 + 单选 → "成功" toast | 选中任意设备 | ✅ |
| 2 | 同步 (制式) | `SyncOutlined` | Modal 弹窗：仅显示当前制式的同步字段（无 scope 标签） | **仅选中相同制式** | ✅ |
| 3 | 重启 | `ReloadOutlined` | 确认对话框："确定重启设备？" → "命令已经下发。" 提示 | 选中任意设备 | ✅ |
| 4 | 回收站 | `RestOutlined` (danger) | 警告确认：两行说明文字 → "成功" toast | 选中任意设备 | ✅ |

> **批量同步制式限制**：只有当所有选中设备的 `networkType` 完全相同时（全 eNB / 全 gNB / 全 GSM），才在批量操作栏显示"同步"按钮。按钮文本显示当前制式，如"批量同步 (eNB)"。弹窗标题带制式 Tag，仅展示该制式的同步参数（参见 7.5.4.1 各制式字段表）。
>
> 页面顶部"导出"按钮为独立功能，不需要选中行，点击弹出导出配置弹窗。

#### 7.5.4 批量同步参数整合（三制式字段合并）

> 源文件：`SyncParamsModal.tsx`
>
> 原始 JSP 源码：
> - eNB: `enodeb/monitor/sync_params.jsp`
> - GSM: `enodeb/monitor/GSM/gsm_syncParams.jsp`
> - gNB: `gnodeb/monitor/gnodeb_monitor.jsp`（内嵌同步弹窗）

##### 7.5.4.1 三制式原始同步弹窗字段对比

###### eNB 同步弹窗（sync_params.jsp）

共 2 组 + 告警管理，基础配置 29 字段（3 个条件字段可能被过滤），高级配置 22 字段。

**告警管理**：`alarmSync`（默认勾选） → 传 `sync_alarm`

**基础配置 (basicCol) — 29 字段**：

| # | code | 标签 | 条件 |
|---|------|------|------|
| 1 | module_type | 设备型号名 | |
| 2 | software_version | 软件版本 | |
| 3 | firmware_version | 固件版本 | |
| 4 | MAC | 小站MAC | |
| 5 | cell_name | 主机名 | |
| 6 | ECI | ECI | |
| 7 | PCI | PCI | |
| 8 | plmn | PLMN | |
| 9 | tac | TAC | |
| 10 | bandwidth | 带宽 | |
| 11 | earfcn | 频点 | |
| 12 | duplex_mode | 基站指示（双工模式） | |
| 13 | tx_power | CPE发射功率 | |
| 14 | cell_status | 是否激活 | |
| 15 | mme_status | MME状态 | |
| 16 | rf_status | 射频开关状态 | |
| 17 | halob_flag | HaloX | ⚠️ `isSupportHalob == 'true'` |
| 18 | sync_status | 同步状态 | |
| 19 | IP | IP地址 | |
| 20 | mme_addr | MME Pool IPSEC地址 | ⚠️ `is_super_user == 'true'` |
| 21 | lease | 锁定状态 | ⚠️ `writableMap["CODE_ENB_EXPIRY_DATE"]` 存在 |
| 22 | ue_count | UE数 | |
| 23 | root_sequence_index | 根序列索引 | |
| 24 | gps_satellites | GPS卫星数 | |
| 25 | sub_frame_assignment | 子帧配比 + 特殊子帧配比 | |
| 26 | wan_speed | WAN状态 | |
| 27 | ipsec_addr | IPSEC地址 | ⚠️ `is_super_user == 'true'` |
| 28 | gps_position | GPS版本 + GPS经度 + GPS纬度 + GPS高度 | |
| 29 | electronic_downtilt | 电子下倾角 | |

**高级配置 (othersCol) — 22 字段**：

| # | code | 标签 |
|---|------|------|
| 1 | band | 频段 |
| 2 | sas_param | SAS 监测参数名 |
| 3 | cell_neighbor | SAS 邻区 |
| 4 | itfn_param | 背向接口 |
| 5 | son_pci | SON PCI |
| 6 | rollback_enable | 回退 设置开关 |
| 7 | rollback_version | 回退版本 |
| 8 | uboot_version | UBoot版本 |
| 9 | kernel_version | Kernel版本 |
| 10 | is_https | Https 状态 |
| 11 | lan_enable | LAN状态 |
| 12 | wan_ip | WAN IP地址 |
| 13 | lte_turbo_enable | LTE Turbo |
| 14 | eu_ru | EU/RU数 |
| 15 | halob_license | HaloB License |
| 16 | lock_mac_addr | 锁定Mac |
| 17 | lock_mac_status | 锁定Mac 状态 |
| 18 | ipsec_bind_interface | IPSec Bind Interface |
| 19 | lgw_basic | WCG 监测参数名 |
| 20 | lgw_mode_ue_speed_statistics | UE Speed Statistics |
| 21 | ipsec_auto_enroll | IPSec Auto Enroll |
| 22 | slot | Slot |

> eNB 总计：29 + 22 = **51 字段**（+ 告警管理）
> 提交 API：`POST /cell/quicksettings/batchSyncCell.action`（批量）/ `POST /cell/param/refreshCellInfo.action`（单个）
> 参数：`smallCellCode=<sn逗号分隔>&selectedParams=<code逗号分隔>[,sync_alarm]`

---

###### gNB 同步弹窗（gnodeb_monitor.jsp 内嵌）

共 2 组 + 告警管理，基础配置 16 字段（+ 1 条件字段），高级配置 7 字段。

**告警管理**：`alarmSync`（默认勾选） → 传 `sync_alarm`

**基础配置 (deviceCol, computed) — 16 字段 + 1 条件**：

| # | code | 标签 | 条件 |
|---|------|------|------|
| 1 | cell_name | 5G站点名称 | |
| 2 | IP | IP地址 | |
| 3 | module_type | 设备型号名 | |
| 4 | software_version | 软件版本 | |
| 5 | firmware_version | 硬件版本 | |
| 6 | halob_flag | HaloB开关 | |
| 7 | ue_count | UE数 | |
| 8 | sync_status | 同步状态 | |
| 9 | adminState | Admin状态 | |
| 10 | amf_status | AMF Status | |
| 11 | multiPlmnEnable | MultiPLMN状态 | |
| 12 | ECI | ECI (gNodeB ID + Cell Identity) | |
| 13 | cellConfig | 小区参数 | |
| 14 | MAC | MAC | |
| 15 | mme_addr | IPSEC地址 | |
| 16 | gps_position | GPS经度 + GPS纬度 | |
| 17 | sub_station_name | 站址名称 | ⚠️ `supportTopoSite == true` |

> 注意：gNB 的 `gps_position` 只含经度+纬度，不含高度（eNB 含高度），gNB 的 `firmware_version` 标签为"硬件版本"。

**高级配置 (othersCol) — 7 字段**：

| # | code | 标签 |
|---|------|------|
| 1 | rollback_version | 回退版本 |
| 2 | sas_param | SAS 监测参数名 |
| 3 | eu_ru | EU/RU数 |
| 4 | halob_license | HaloB License |
| 5 | energy_saving | Energy |
| 6 | gnb_topo_cellmgr | gNB TOPO |
| 7 | ssl_cert_validity | SSL Cert Validity |

> gNB 总计：16 + 7 = **23 字段**（+ 1 条件字段 + 告警管理）
> 提交 API：同 eNB `POST /cell/quicksettings/batchSyncCell.action`
> 默认预选：`form.device: ['cell_name','IP','module_type','software_version','halob_flag','ue_count']`

---

###### GSM 同步弹窗（gsm_syncParams.jsp）

共 3 组 + 告警管理，基础配置 7 字段，BSC 1 字段，BTS 8 字段。

**告警管理**：`alarmSync`（默认勾选） → 传 `sync_alarm`

**基础配置 (basicCol) — 7 字段**：

| # | code | 标签 |
|---|------|------|
| 1 | module_type | 设备型号名 |
| 2 | software_version | 软件版本 |
| 3 | firmware_version | 固件版本 |
| 4 | MAC | 小站MAC |
| 5 | IP | IP地址 |
| 6 | ue_count | UE数 |
| 7 | halob_license | License |

**BSC 参数 (bscCol) — 1 字段**：

| # | code | 标签 |
|---|------|------|
| 1 | BtsNum | BTS数 |

**BTS 参数 (btsCol) — 8 字段**：

| # | code | 标签 |
|---|------|------|
| 1 | cell_status | 是否激活 |
| 2 | rf_status | 射频开关状态 |
| 3 | sync_status | 同步状态 |
| 4 | gps_satellites | GPS卫星数 |
| 5 | currentLac | LAC |
| 6 | currentArfcn | 频点 + 上行频率 + 下行频率 |
| 7 | gps_position | GPS经度 + GPS纬度 + GPS高度 |
| 8 | bts_bsc_relationship | Ipa Unit Id + Oml Remote Ip + Oml Remote Ip Bak + BSC Select + BSC连接状态 + 所属BSC编码 |

> GSM 总计：7 + 1 + 8 = **16 字段**（+ 告警管理）
> 提交 API：同 eNB `POST /cell/quicksettings/batchSyncCell.action`（批量）/ `POST /cell/param/refreshCellInfo.action`（单个）
> 无条件字段，无默认预选

---

###### 三制式同步弹窗结构对比

| 维度 | eNB (LTE) | gNB (5G NR) | GSM |
|------|:---------:|:-----------:|:---:|
| 分组数 | 2（基础 + 高级） | 2（基础 + 高级） | 3（基础 + BSC + BTS） |
| 基础字段数 | 29 | 16 (+1 条件) | 7 |
| 高级/扩展字段数 | 22 | 7 | 9 (BSC:1 + BTS:8) |
| **字段总数** | **51** | **23 (+1)** | **16** |
| 条件字段 | 3 (halob_flag, ipsec_addr, mme_addr, lease) | 1 (sub_station_name) | 0 |
| 默认预选 | 动态 (7 字段，见下) | 6 字段硬编码 | 无 |
| 告警管理 | ✅ 默认勾选 | ✅ 默认勾选 | ✅ 默认勾选 |
| 提交 API | batchSyncCell.action | batchSyncCell.action | batchSyncCell.action |

###### 默认预选字段详情

**eNB 默认预选**（原始 JSP 中通过 `init()` 从监控列表可见列 `enbvm.showProps` 动态映射，经 `monitorCols` 转换为同步参数 code；统一实现取设备列表默认可见列的静态等价集）：

| # | code | 标签 | 映射来源 (monitorCols) |
|---|------|------|----------------------|
| 1 | module_type | 设备型号名 | deviceModel (直接匹配) |
| 2 | software_version | 软件版本 | softwareVersion (直接匹配) |
| 3 | MAC | MAC地址 | mac_address → MAC |
| 4 | cell_name | 主机名 | host_name → cell_name |
| 5 | IP | IP地址 | cell_ip → IP |
| 6 | cell_status | 激活状态 | op_state → cell_status |
| 7 | ue_count | UE数 | ueCount (直接匹配) |

**gNB 默认预选**（`gnodeb_monitor.jsp` 中 `form.device` 硬编码初始值）：

| # | code | 标签 |
|---|------|------|
| 1 | cell_name | 5G站点名称 |
| 2 | IP | IP地址 |
| 3 | module_type | 设备型号名 |
| 4 | software_version | 软件版本 |
| 5 | halob_flag | HaloB开关 |
| 6 | ue_count | UE数 |

**GSM 默认预选**：无（`gsm_syncParams.jsp` 中 `form.basic/bsc/bts` 均为空数组，`mounted()` 为空）

##### 7.5.4.2 统一同步弹窗（按制式分类显示）

批量同步仅在所有选中设备为**同一制式**时可用。弹窗根据制式过滤参数，只显示当前制式的同步字段。各制式的默认预选字段同上。

##### 告警管理（独立复选框，默认勾选）

所有制式均支持，勾选后额外传 `sync_alarm` 参数。

##### 基础配置参数

| # | code | 标签 | 适用范围 | eNB | gNB | GSM |
|---|------|------|:--------:|:---:|:---:|:---:|
| 1 | module_type | 设备型号名 | 三制式公共 | ✅ | ✅ | ✅ |
| 2 | software_version | 软件版本 | 三制式公共 | ✅ | ✅ | ✅ |
| 3 | firmware_version | 固件版本 | 三制式公共 | ✅ | ✅ | ✅ |
| 4 | MAC | MAC地址 | 三制式公共 | ✅ | ✅ | ✅ |
| 5 | IP | IP地址 | 三制式公共 | ✅ | ✅ | ✅ |
| 6 | ue_count | UE数 | 三制式公共 | ✅ | ✅ | ✅ |
| 7 | cell_name | 主机名/站点名 | eNB+gNB | ✅ | ✅ | |
| 8 | ECI | ECI | eNB+gNB | ✅ | ✅ | |
| 9 | halob_flag | HaloB开关 | eNB+gNB | ✅⚠️ | ✅ | |
| 10 | sync_status | 同步状态 | eNB+gNB | ✅ | ✅ | |
| 11 | mme_addr | IPSec地址 | eNB+gNB | ✅⚠️ | ✅ | |
| 12 | gps_position | GPS位置 | eNB+gNB | ✅(含高度) | ✅(经纬度) | |
| 13 | PCI | PCI | 仅eNB | ✅ | | |
| 14 | plmn | PLMN | 仅eNB | ✅ | | |
| 15 | tac | TAC | 仅eNB | ✅ | | |
| 16 | bandwidth | 带宽 | 仅eNB | ✅ | | |
| 17 | earfcn | 频点 | 仅eNB | ✅ | | |
| 18 | duplex_mode | 双工模式 | 仅eNB | ✅ | | |
| 19 | tx_power | 发射功率 | 仅eNB | ✅ | | |
| 20 | cell_status | 激活状态 | 仅eNB | ✅ | | |
| 21 | mme_status | MME状态 | 仅eNB | ✅ | | |
| 22 | rf_status | 射频开关状态 | 仅eNB | ✅ | | |
| 23 | lease | 锁定状态 | 仅eNB | ✅⚠️ | | |
| 24 | root_sequence_index | 根序列索引 | 仅eNB | ✅ | | |
| 25 | gps_satellites | GPS卫星数 | 仅eNB | ✅ | | |
| 26 | sub_frame_assignment | 子帧配比 | 仅eNB | ✅ | | |
| 27 | wan_speed | WAN状态 | 仅eNB | ✅ | | |
| 28 | ipsec_addr | IPSec地址(eNB) | 仅eNB | ✅⚠️ | | |
| 29 | electronic_downtilt | 电子下倾角 | 仅eNB | ✅ | | |
| 30 | adminState | Admin State | 仅gNB | | ✅ | |
| 31 | amf_status | AMF Status | 仅gNB | | ✅ | |
| 32 | multiPlmnEnable | MultiPLMN状态 | 仅gNB | | ✅ | |
| 33 | cellConfig | 小区参数 | 仅gNB | | ✅ | |
| 34 | halob_license | License | 仅GSM | | | ✅ |

##### 高级配置参数

| # | code | 标签 | 适用范围 | eNB | gNB | GSM |
|---|------|------|:--------:|:---:|:---:|:---:|
| 1 | rollback_version | 回退版本 | eNB+gNB | ✅ | ✅ | |
| 2 | sas_param | SAS参数 | eNB+gNB | ✅ | ✅ | |
| 3 | eu_ru | EU/RU数 | eNB+gNB | ✅ | ✅ | |
| 4 | halob_license | HaloB License | eNB+gNB | ✅ | ✅ | |
| 5 | band | 频段 | 仅eNB | ✅ | | |
| 6 | cell_neighbor | SAS邻区 | 仅eNB | ✅ | | |
| 7 | itfn_param | 背向接口 | 仅eNB | ✅ | | |
| 8 | son_pci | SON PCI | 仅eNB | ✅ | | |
| 9 | rollback_enable | 回退开关 | 仅eNB | ✅ | | |
| 10 | uboot_version | UBoot版本 | 仅eNB | ✅ | | |
| 11 | kernel_version | Kernel版本 | 仅eNB | ✅ | | |
| 12 | is_https | Https状态 | 仅eNB | ✅ | | |
| 13 | lan_enable | LAN状态 | 仅eNB | ✅ | | |
| 14 | wan_ip | WAN IP地址 | 仅eNB | ✅ | | |
| 15 | lte_turbo_enable | LTE Turbo | 仅eNB | ✅ | | |
| 16 | lock_mac_addr | 锁定MAC | 仅eNB | ✅ | | |
| 17 | lock_mac_status | 锁定MAC状态 | 仅eNB | ✅ | | |
| 18 | ipsec_bind_interface | IPSec Bind Interface | 仅eNB | ✅ | | |
| 19 | lgw_basic | WCG参数 | 仅eNB | ✅ | | |
| 20 | lgw_mode_ue_speed_statistics | UE Speed Statistics | 仅eNB | ✅ | | |
| 21 | ipsec_auto_enroll | IPSec Auto Enroll | 仅eNB | ✅ | | |
| 22 | slot | Slot | 仅eNB | ✅ | | |
| 23 | energy_saving | Energy Saving | 仅gNB | | ✅ | |
| 24 | gnb_topo_cellmgr | gNB TOPO | 仅gNB | | ✅ | |
| 25 | ssl_cert_validity | SSL Cert Validity | 仅gNB | | ✅ | |

##### BSC 参数（仅 GSM）

| # | code | 标签 | 适用范围 | eNB | gNB | GSM |
|---|------|------|:--------:|:---:|:---:|:---:|
| 1 | BtsNum | BTS数 | 仅GSM | | | ✅ |

##### BTS 参数（仅 GSM）

| # | code | 标签 | 适用范围 | eNB | gNB | GSM |
|---|------|------|:--------:|:---:|:---:|:---:|
| 1 | cell_status | 激活状态 | 仅GSM | | | ✅ |
| 2 | rf_status | 射频开关状态 | 仅GSM | | | ✅ |
| 3 | sync_status | 同步状态 | 仅GSM | | | ✅ |
| 4 | gps_satellites | GPS卫星数 | 仅GSM | | | ✅ |
| 5 | currentLac | LAC | 仅GSM | | | ✅ |
| 6 | currentArfcn | 频点+上行频率+下行频率 | 仅GSM | | | ✅ |
| 7 | gps_position | GPS经度+纬度+高度 | 仅GSM | | | ✅ |
| 8 | bts_bsc_relationship | IPA Unit ID + OML Remote IP + BSC关系 | 仅GSM | | | ✅ |

##### 参数统计

| 分组 | 三制式公共 | eNB+gNB | 仅eNB | 仅gNB | 仅GSM | 总计 |
|------|:---------:|:-------:|:-----:|:-----:|:-----:|:----:|
| 基础配置 | 6 | 6 | 17 | 4 | 1 | **34** |
| 高级配置 | 0 | 4 | 18 | 3 | 0 | **25** |
| BSC | 0 | 0 | 0 | 0 | 1 | **1** |
| BTS | 0 | 0 | 0 | 0 | 8 | **8** |
| **合计** | **6** | **10** | **35** | **7** | **10** | **68** |

> ⚠️ = 条件字段（受权限或功能开关控制，可能被过滤不显示）

#### 7.5.5 导出功能（页面顶部按钮）

> 源文件：`ExportModal.tsx`

原三制式各自有独立的导出入口（eNB: Popover 全参数配置页, gNB: Popover, GSM: Dropdown/Popover），统一为页面右上角"导出"按钮 → Modal 弹窗。

##### 三制式导出功能对比

| 维度 | eNB | gNB | GSM |
|------|-----|-----|-----|
| 入口 | Popover (920×500px) | Popover (700×400px) | Dropdown / Popover (云管理员) |
| 格式 | CSV / XLSX | CSV / XLSX | XLSX (云管理员: XLSX) |
| 字段选择 | ✅ 6组可折叠 | ✅ 5组可折叠 | ❌ 使用当前可见列 |
| License 导出 | ✅ | ✅ | ✅ (仅云管理员) |
| 运营商选择 | ✅ (仅云管理员) | ❌ | ✅ (仅云管理员) |
| 需选中行 | ❌ | ❌ | ❌ |

##### 统一导出弹窗配置

| # | 配置项 | 类型 | 说明 |
|---|--------|------|------|
| 1 | 选择运营商 | 多选下拉 | 中国移动/中国电信/中国联通，支持多选 |
| 2 | 列表字段 | 分组复选框 | 4 组折叠面板：公共字段(19) / eNB(22) / gNB(14) / GSM(10)，各组支持全选 |
| 3 | 导出License | 复选框 | 同时导出设备License信��� |
| 4 | 导出格式 | Radio | XLSX（默认）/ CSV |

##### 导出列字段分组

| 分组 | Tag 颜色 | 字段数 | 说明 |
|------|:--------:|:------:|------|
| 公共字段 | - | 19 | SN/连接状态/告警/名称/产品类型/型号/版本/MAC/设备组/IP/时间/状态/UE/���频/GPS |
| eNB 字段 | 蓝色 | 22 | ECI/PCI/PLMN/TAC/带宽/频点/功率/MME/CPE/IPSec/子帧/WAN/倾角/方位角/卫星 |
| gNB 字段 | 绿色 | 14 | NR Cell ID/PCI/TAC/Band/ARFCN/功率/AdminState/HaloB/同步/MultiPLMN/AMF/IPSec |
| GSM 字段 | 橙色 | 10 | BSC编码/BTS数/LAC/频点/频率/BSC连接/BSC Select/IPA/OML |
| **合计** | | **65** | |

##### 导出 API 端点

| 制式 | CSV | XLSX | License |
|------|-----|------|---------|
| eNB | `/cell/cpeinfos/exportCellsToCsv.action` | `/cell/cpeinfos/exportCellsToExcel.action` | `/cell/cpeinfos/exportEnodebLicenseInfos.action` |
| gNB | `/gnb/gnbMonitor/exportGnbInfoToCsv.action` | `/gnb/gnbMonitor/exportGnbInfoToExcel.action` | 同 eNB + `isGnb=1` |
| GSM | - | `/cell/cpeinfos/exportGSMInfosToExcel.action` | 同 eNB + `isGSM=1` |

