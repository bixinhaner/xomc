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

---

## 3. 公共字段 vs 独立字段

### 3.1 三页面公共字段（三个页面都有）

| # | field | LTE 标签 | GSM 标签 | 5G NR 标签 | LTE默认 | GSM默认 | 5G默认 |
|---|-------|---------|---------|-----------|:-------:|:-------:|:------:|
| 1 | serial_number | 小站编码 | BSC编码 | 小站编码 | ✅锁定 | ✅固定 | ✅锁定 |
| 2 | host_name | 主机名 | BSC名称 | 5G站点名称 | ✅锁定 | ✅固定 | ✅锁定 |
| 3 | product | 产品类型标识 | 产品类型标识 | 产品类型标识 | ✅锁定 | ✅ | ✅锁定 |
| 4 | module_type | 设备型号名 | 设备型号名 | 设备型号名 | ✅锁定 | ✅ | ✅锁定 |
| 5 | software_version | 软件版本 | 软件版本 | 软件版本 | ✅锁定 | ✅ | ✅锁定 |
| 6 | firmware_version | 固件版本 | 固件版本 | 硬件版本 | - | ✅ | - |
| 7 | mac_address | MAC地址 | MAC | MAC | ✅锁定 | ✅ | ✅锁定 |
| 8 | group_name | 设备组 | 设备组 | 设备组 | ✅锁定 | ✅ | ✅锁定 |
| 9 | product_name | 产品名称 | 产品名称 | 产品名称 | - | ✅ | - |
| 10 | sub_station_name | 站点名称 | 站址名称 | 站址名称 | - | ✅条件 | -条件 |
| 11 | remark | 备注 | 备注 | 备注 | - | ✅ | - |
| 12 | up_time | 运行时间 | 运行时间 | 运行时间 | - | ✅ | - |
| 13 | first_online_time | 第一次连接时间 | 第一次连接时间 | 第一次连接时间 | - | ✅ | - |
| 14 | LASTINFORMTIME | 上次连接时间 | 上次连接时间 | 上次连接时间 | - | ✅ | - |
| 15 | cell_ip | IP地址 | IP地址 | IP地址 | ✅锁定 | ✅ | ✅锁定 |
| 16 | op_state | 是否激活 | 是否激活 | gNB状态 | ✅锁定 | ✅ | ✅锁定 |
| 17 | rf_status | 射频开关状态 | 射频开关状态 | 射频开关状态 | ✅锁定 | ✅ | - |
| 18 | ue_count | UE数 | UE数 | UE数 | ✅锁定 | ✅ | ✅锁定 |
| 19 | synStatus | 同步状态 | 同步状态 | 同步状态 | - | ✅ | - |
| 20 | gps_longitude | GPS经度 | GPS经度 | GPS经度 | - | ✅ | - |
| 21 | gps_latitude | GPS纬度 | GPS纬度 | GPS纬度 | - | ✅ | - |
| 22 | gps_height | GPS高度 | GPS高度 | GPS高度 | - | ✅ | - |

### 3.2 LTE + 5G NR 共有（GSM 无）

| # | field | LTE 标签 | 5G NR 标签 | LTE默认 | 5G默认 |
|---|-------|---------|-----------|:-------:|:------:|
| 1 | online_time | 接入时间 | 接入时间 | ✅ | ✅ |
| 2 | offline_time | 断开时间 | 断开时间 | ✅ | ✅ |
| 3 | PHYCELLID | PCI | PCI | ✅锁定 | ✅锁定 |
| 4 | tac | TAC | TAC | - | - |
| 5 | EARFCNDLINUSE | 频点 | NR频点下限 | - | - |
| 6 | network_model | 基站指示 | 基站指示 | - | - |
| 7 | tx_power | CPE发射功率 | Tx Power | - | - |
| 8 | IPSEC_ADDR | IPSEC地址 | IPSEC地址 | - | - |
| 9 | halob_flag | HaloX | HaloB开关 | - | ✅锁定 |

### 3.3 LTE + GSM 共有（5G NR 无）

| # | field | LTE 标签 | GSM 标签 | LTE默认 | GSM默认 |
|---|-------|---------|---------|:-------:|:-------:|
| 1 | online_duration | 累计时长 | 累计时长 | - | ✅ |
| 2 | gps_satellite_count | GPS卫星数 | GPS卫星数 | - | ✅ |

### 3.4 LTE 独有字段（仅 LTE 有）

| # | field | 中文标签 | 默认 | 所属组 |
|---|-------|---------|:----:|--------|
| 1 | enbId | eNodeB ID | - | cell |
| 2 | cellId | 小区ID | - | cell |
| 3 | CELL_IDENTITY | ECI | ✅锁定 | cell |
| 4 | plmnid | PLMN | - | cell |
| 5 | signment | 子帧配比 | - | cell |
| 6 | specialSubframe | 特殊子帧配比 | - | cell |
| 7 | rootIndex | 根序列索引 | - | cell |
| 8 | site_id | 站点ID | - | cell |
| 9 | bandwidth | 带宽 | - | cell |
| 10 | mme_status | MME状态 | ✅锁定 | status |
| 11 | pm_report_status | KPI上报状态 | - | status |
| 12 | validity | 有效期 | - | status |
| 13 | lock_status | 锁定状态 | - | status |
| 14 | euCountStr | EU数 | - | status |
| 15 | ruCountStr | RU数 | - | status |
| 16 | cpe_connect | CPE连接数 | ✅锁定 | status |
| 17 | wanSpeed | WAN状态 | - | status |
| 18 | service_status | 状态 | - | status |
| 19 | mmepool_ipsec_addr | MME Pool IPSEC地址 | - | network |
| 20 | gps_version | GPS版本 | - | device |
| 21 | rom | Rom | - | device |
| 22 | mechanical_downtilt | 机械下倾角 | - | location |
| 23 | electronic_downtilt | 电子下倾角 | - | location |
| 24 | vertical_3dB_beam_width | 垂直波束宽度 | - | location |
| 25 | horizontal_azimuth | 水平方位角 | - | location |
| 26 | install_address | 安装详细地址 | - | location |

### 3.5 GSM 独有字段（仅 GSM 有）

| # | field | 中文标签 | 所属 |
|---|-------|---------|------|
| 1 | IpaUnitId | Ipa Unit Id | 网络 |
| 2 | OmlRemoteIp | Oml Remote Ip | 网络 |
| 3 | OmlRemoteIpBak | Oml Remote Ip Bak | 网络 |
| 4 | BscSelect | BSC Select | 状态 |
| 5 | BscLinkStatus | BSC连接状态 | 状态 |
| 6 | BSCSerialNumber | 所属BSC编码 | 状态 |
| 7 | BtsNum | BTS数 | 状态 |
| 8 | currentLac | LAC | 小区 |
| 9 | currentArfcn | ��点 | 小区 |
| 10 | uplinkFrequency | 上行频率 | 小区 |
| 11 | downlinkFrequency | 下行频率 | 小区 |

### 3.6 5G NR 独有字段（仅 5G NR 有）

| # | field | 中文标签 | 默认 | 所属组 |
|---|-------|---------|:----:|--------|
| 1 | gNBId | gNodeB ID | - | device |
| 2 | nr_cell_id | NR小区ID | - | cell |
| 3 | Band | 频段 | - | cell |
| 4 | EARFCNULINUSE | NR频点上限 | - | cell |
| 5 | adminState | Admin状态 | - | status |
| 6 | amf_status | AMF Status | - | status |
| 7 | multiPlmnEnable | Multi PLMN状态 | - | status |

#### 5G NR 同步对话框扩展（othersCol，不在主列表中）

| # | field | 中文标签 |
|---|-------|---------|
| 1 | rollback_version | 回滚版本 |
| 2 | sas_param | SAS参数 |
| 3 | eu_ru | EU/RU数 |
| 4 | halob_license | HaloB License |
| 5 | energy_saving | 节能 |
| 6 | gnb_topo_cellmgr | gNB TOPO |
| 7 | ssl_cert_validity | SSL证书有效期 |

---

## 4. 统计汇总

### 4.1 字段数量

| 维度 | LTE eNodeB | GSM | 5G NR gNodeB |
|------|:----------:|:---:|:------------:|
| 固定列 | 7 | 7 | 6 |
| 可配置列 | 60 | 33 | 42 (+7 othersCol) |
| 默认勾选数 | 17 | 33 (全部) | 14 |
| 锁定字段数 | 15 | 0 | 12 |
| 列分组数 | 6 | 1 (扁平) | 5 |

### 4.2 默认勾选对比

| field | LTE | GSM | 5G NR | 三页面共有默认 |
|-------|:---:|:---:|:-----:|:-------------:|
| serial_number | ✅锁定 | ✅ | ✅锁定 | ✅ |
| host_name | ✅锁定 | ✅ | ✅锁定 | ✅ |
| product | ✅锁定 | ✅ | ✅锁定 | ✅ |
| module_type | ✅锁定 | ✅ | ✅锁定 | ✅ |
| software_version | ✅锁定 | ✅ | ✅锁定 | ✅ |
| mac_address | ✅锁定 | ✅ | ✅锁定 | ✅ |
| group_name | ✅锁定 | ✅ | ✅锁定 | ✅ |
| cell_ip | ✅锁定 | ✅ | ✅锁定 | ✅ |
| op_state | ✅锁定 | ✅ | ✅锁定 | ✅ |
| ue_count | ✅锁定 | ✅ | ✅锁定 | ✅ |
| rf_status | ✅锁定 | ✅ | - | LTE+GSM |
| online_time | ✅ | - | ✅ | LTE+5G |
| offline_time | ✅ | - | ✅ | LTE+5G |
| CELL_IDENTITY | ✅锁定 | - | - | 仅LTE |
| PHYCELLID | ✅锁定 | - | ✅锁定 | LTE+5G |
| mme_status | ✅锁定 | - | - | 仅LTE |
| cpe_connect | ✅锁定 | - | - | 仅LTE |
| halob_flag | - | - | ✅锁定 | 仅5G |
| firmware_version | - | ✅ | - | 仅GSM |
| product_name | - | ✅ | - | 仅GSM |
| sub_station_name | - | ✅ | - | 仅GSM |
| up_time | - | ✅ | - | 仅GSM |
| online_duration | - | ✅ | - | 仅GSM |
| first_online_time | - | ✅ | - | 仅GSM |
| LASTINFORMTIME | - | ✅ | - | 仅GSM |
| remark | - | ✅ | - | 仅GSM |

### 4.3 三页面共有且默认显示的 10 个核心字段

```
serial_number, host_name, product, module_type, software_version,
mac_address, group_name, cell_ip, op_state, ue_count
```

这 10 个字段在所有三个监控页面中都是**默认勾选**的，应作为统一设备列表的默认显示列。

---

## 5. 统一设备列表实现方案

> 前端路径：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`

### 5.1 设计思路

将三个原始监控页面（LTE eNodeB、GSM、5G NR gNodeB）合并为一个统一的设备列表页面。通过 **基站制式（networkType）** 列区分 eNB / gNB / GSM，所有字段按归属制式分为四组。

### 5.2 字段分组方案

每列通过 `group` 属性标记归属：

| 分组 | group 值 | 说明 | 列数 | 默认显示 |
|------|----------|------|:----:|:--------:|
| 公共字段 | `common` | 三制式共有的字段 | 27 | 15 |
| eNB 字段 | `eNB` | LTE 独有 / LTE+GSM 共有 | 31 | 0 |
| gNB 字段 | `gNB` | 5G NR 独有 + othersCol | 15 | 0 |
| GSM 字段 | `GSM` | GSM 独有 | 11 | 0 |
| **合计** | | | **84** | **15** |

#### 公共字段 (common) — 27 列

默认显示（15列）：

| # | key | 标题 | 说明 |
|---|-----|------|------|
| 1 | sn | SN | 设备序列号，可点击跳转详情 |
| 2 | connStatus | 连接状态 | 在线/离线状态指示 |
| 3 | alarmLevel | 告警级别 | 彩色 Tag 显示 |
| 4 | hostName | 名称 | 原 host_name，LTE=主机名，GSM=BSC名称，5G=站点名称 |
| 5 | networkType | 基站制式 | eNB(蓝)/gNB(绿)/GSM(橙) Tag |
| 6 | productType | 产品类型 | eNB / gNB / GSM |
| 7 | deviceModel | 设备型号 | BBU3910 等 |
| 8 | softwareVersion | 软件版本 | |
| 9 | macAddress | MAC地址 | 等宽字体，可复制 |
| 10 | groupName | 设备组 | |
| 11 | ipAddress | IP地址 | 等宽字体，可复制 |
| 12 | onlineTime | 接入时间 | 格式化时间戳 |
| 13 | offlineTime | 断开时间 | 格式化时间戳 |
| 14 | opState | 操作状态 | active(绿)/inactive(红) Tag |
| 15 | ueCount | UE数 | |

默认隐藏（12列）：productName, firmwareVersion, onlineDuration, upTime, firstOnlineTime, lastInformTime, lastOnlineTime, siteName, rfStatus, longitude, latitude, gpsHeight, gpsSatelliteCount, installAddress

#### eNB 字段 — 31 列（全部默认隐藏）

小区信息：enbId, cellId, eci, pci, plmnId, tac, subframeAssignment, specialSubframe, rootIndex, siteId, bandwidth, dlEarfcn, ulEarfcn, networkModel, txPower, band
状态信息：mmeStatus, pmReportStatus, cpeCount, lockStatus, wanSpeed, serviceStatus, adminState, multiPlmnEnable
设备信息：gpsVersion, rom, remark
网络信息：ipsecAddr, mmepoolIpsecAddr
位置信息：mechanicalDowntilt, electronicDowntilt, verticalBeamWidth, horizontalAzimuth

#### gNB 字段 — 15 列（全部默认隐藏）

小区/状态：gnbId, nrCellId, amfStatus, halobFlag, syncStatus, validity, euCount, ruCount
5G NR 扩展（原 othersCol）：rollbackVersion, sasParam, euRu, halobLicense, energySaving, gnbTopoCellmgr, sslCertValidity

#### GSM 字段 — 11 列（全部默认隐藏）

小区信息：lac, arfcn, uplinkFrequency, downlinkFrequency
状态/网络：bscLinkStatus, bscSelect, bscSerialNumber, btsNum, ipaUnitId, omlRemoteIp, omlRemoteIpBak

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

### 5.5 筛选条件

| # | 字段 | 类型 | 选项 |
|---|------|------|------|
| 1 | sn | 输入框 | - |
| 2 | hostName（名称） | 输入框 | - |
| 3 | networkType（基站制式） | 下拉 | eNB / gNB / GSM |
| 4 | productType（产品类型） | 下拉 | eNB / gNB / GSM |
| 5 | connStatus（连接状态） | 下拉 | 在线 / 离线 |
| 6 | opState（操作状态） | 下拉 | 激活 / 未激活 |
| 7 | groupName（设备组） | 输入框 | - |

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

## 6. 各监控页面产品类型分析

### 6.1 LTE eNodeB 监控页面

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

### 6.2 GSM 监控页面

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

### 6.3 5G NR gNodeB 监控页面

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

---

### 6.4 筛选下拉选项对比（前端 UI 层面）

> 以下分析区分了**前端筛选下拉选项**（用户在 UI 中可选的值）和**代码条件渲染中引用的 product/platformType 值**（仅用于列渲染逻辑）。

#### LTE eNodeB 筛选下拉

- **产品类型标识 (`product_model`)**：JSP 中定义为 `product_model: []`（空数组），无硬编码选项
- **加载方式**：JSP 中未找到调用 `getEnbMonitorProductList` 的 API 请求，下拉选项完全由后端动态填充
- **结论**：LTE 页面的产品类型筛选选项由后端接口返回，前端不维护固定列表

#### GSM 筛选下拉

- **产品类型标识 (`product_model`)**：JSP 中硬编码为 `product_model: 'BSC,BTS'`（逗号分隔字符串）
- **加载方式**：无动态 API 加载，固定值作为默认筛选条件
- **结论**：GSM 页面默认筛选 BSC 和 BTS 两种产品类型，无动态扩展

#### 5G NR gNodeB 筛选下拉

- **硬编码选项**：`advancedQueryItemList` 中定义了 `BaiBNX` 和 `BaiBNQ` 两个选项
- **动态加载**：页面初始化时调用 `getEnbMonitorProductList.action?isGnb=1` 接口，返回值**覆盖**硬编码选项
- **结论**：5G NR 页面有初始硬编码值，但实际运行时以后端返回为准

#### 下拉选项汇总

| 页面 | 硬编码选项 | 动态加载 | 实际行为 |
|------|-----------|:--------:|---------|
| LTE eNodeB | 无（空数组） | 未明确调用 | 由后端完全控制 |
| GSM | `BSC, BTS` | 无 | 固定两种产品 |
| 5G NR gNodeB | `BaiBNX, BaiBNQ` | ✅ `getEnbMonitorProductList?isGnb=1` | 硬编码为初始值，后端可覆盖 |

> **注意**：代码中通过 `product` 和 `platformType` 条件判断的产品类型（如 PM-B4860、QAFA、Intel_CR_CA 等）仅影响列渲染逻辑（激活状态显示、MME 列表、Cell ID 格式等），并不等同于用户可选的筛选下拉选项。这些值来自设备上报的数据，不需要在筛选下拉中逐一列举。

---

### 6.5 三页面产品类型汇总

| 维度 | LTE eNodeB | GSM | 5G NR gNodeB |
|------|:----------:|:---:|:------------:|
| product 取值数 | 5+ (PM-B4860, QAFA, QATA, QAFB, RTD, 通用) | 4 (BSC, BTS, PM-B4860, RTD) | 2 (BaiBNX, BaiBNQ) |
| platformType 取值数 | 16 | 7（复用 LTE 逻辑） | 不使用 |
| 小区配置类型 | CA/TC/SC/DC/基础 | 复用 LTE | 无 |
| 多制式支持 | BM (LTE+GSM) | - | - |
| 双载波 | Intel_CR_DC, MLN_DC, QA_436Q_DC, NEU430_DC | 复用 LTE | - |
| 筛选下拉选项 | 后端动态（空数组） | BSC, BTS（硬编码） | BaiBNX, BaiBNQ（硬编码+动态覆盖） |
| 复杂度 | 高（16 种平台 × 5 种产品） | 中（复用 LTE 逻辑） | 低（仅 2 种产品） |
