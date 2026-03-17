# 监控页面表格字段对比分析

此文档以下 4 个监控页面的“字段清单”统一按主表的 `Select columns` / 显示隐藏列口径整理，不再保留旧的按整页表格组件逐个罗列方式。

## 四个页面主表字段：按 Select columns 分组整理

说明：

- 本节只按列表里的 `Select columns` / 显示隐藏列面板整理“主表可配置字段”。
- 不重复统计操作列、连接状态列、告警列、序号列、复选框列这类不在分组面板里的固定列。
- 某些字段虽然在分组面板中为禁用勾选状态，但既然实际显示在 `Select columns` 中，仍计入对应分组。

### 1. enodeb_monitor_vue.jsp

Select columns 在主页面中按 6 组定义：`deviceCol`、`cellCol`、`statusCol`、`networkCol`、`locationCol`、`satelliteCol`。

| 分组 | 对应字段 |
| --- | --- |
| 设备信息 | `serial_number`、`product`、`product_name`、`module_type`、`software_version`、`firmware_version`、`online_duration`、`up_time`、`first_online_time`、`LASTINFORMTIME`、`online_time`、`offline_time`、`mac_address`、`gps_version`、`group_name`、`sub_station_name`、`rom`、`remark` |
| 小区信息 | `enbId`、`host_name`、`cellId`、`CELL_IDENTITY`、`PHYCELLID`、`plmnid`、`tac`、`signment`、`specialSubframe`、`rootIndex`、`site_id`、`bandwidth`、`EARFCNDLINUSE`、`network_model`、`tx_power` |
| 状态信息 | `op_state`、`mme_status`、`rf_status`、`pm_report_status`、`halob_flag`、`synStatus`、`validity`、`lock_status`、`ue_count`、`euCountStr`、`ruCountStr`、`cpe_connect`、`wanSpeed`、`service_status` |
| 网络信息 | `mmepool_ipsec_addr`、`cell_ip`、`IPSEC_ADDR` |
| 位置信息 | `gps_longitude`、`gps_latitude`、`gps_height`、`mechanical_downtilt`、`electronic_downtilt`、`vertical_3dB_beam_width`、`horizontal_azimuth`、`install_address` |
| 卫星信息 | `gps_satellite_count` |

默认勾选口径（`columnForm`）为：设备信息默认含 `serial_number`、`product`、`module_type`、`software_version`、`mac_address`、`group_name`、`online_time`、`offline_time`；小区信息默认含 `host_name`、`CELL_IDENTITY`、`PHYCELLID`；状态信息默认含 `op_state`、`mme_status`、`rf_status`、`ue_count`、`cpe_connect`；网络信息默认含 `cell_ip`。

### 2. gsm_monitor_vue.jsp

当前 GSM 页面源码里没有找到和 enodeb / gnodeb / CPE 一样的显式分组弹层：

- 关联页 `gsm_query.jsp` 里显示隐藏列容器是注释掉的；
- 主表列选择来源是 `gsm_monitor_vue.jsp` 中的 `getAllDefaultCols()` + `tableColumns()`；
- 因此下面的“设备信息 / 小区信息 / 状态信息 / 网络信息 / 位置信息”是基于 `getAllDefaultCols()` 字段语义做的人工归类，便于与另外三个页面横向对照，不代表 GSM 页面源码里存在同名分组弹层。

| 分组 | 对应字段 |
| --- | --- |
| 设备信息 | `serial_number`、`host_name`、`product`、`product_name`、`module_type`、`software_version`、`firmware_version`、`group_name`、`up_time`、`online_duration`、`first_online_time`、`LASTINFORMTIME`、`BSCSerialNumber`、`BtsNum`、`remark` |
| 小区信息 | `IpaUnitId`、`BscSelect`、`currentLac`、`currentArfcn`、`uplinkFrequency`、`downlinkFrequency` |
| 状态信息 | `rf_status`、`op_state`、`ue_count`、`BscLinkStatus`、`synStatus` |
| 网络信息 | `cell_ip`、`mac_address`、`OmlRemoteIp`、`OmlRemoteIpBak` |
| 位置信息 | `sub_station_name`（仅 `supportTopoSite` 时追加）、`gps_longitude`、`gps_latitude`、`gps_height`、`gps_satellite_count` |

默认勾选口径如果仍按源码真实 `Select columns` 逻辑理解，则是按 `getAllDefaultCols()` 的返回顺序进入列配置，而不是按上表分组展开。

### 3. gnodeb_monitor.jsp

Select columns 在主页面中按 5 组定义：`device`、`cell`、`status`、`network`、`location`。

| 分组 | 对应字段 |
| --- | --- |
| 设备信息 | `serial_number`、`product`、`product_name`、`module_type`、`host_name`、`gNBId`、`firmware_version`、`software_version`、`up_time`、`first_online_time`、`LASTINFORMTIME`、`online_time`、`offline_time`、`group_name`、`mac_address`、`sub_station_name`（仅 `supportTopoSite` 时追加）、`remark` |
| 小区信息 | `nr_cell_id`、`PHYCELLID`、`tac`、`Band`、`EARFCNULINUSE`、`EARFCNDLINUSE`、`tx_power`、`network_model` |
| 状态信息 | `adminState`、`op_state`、`halob_flag`、`ue_count`、`synStatus`、`multiPlmnEnable`、`rf_status`、`amf_status` |
| 网络信息 | `IPSEC_ADDR`、`cell_ip` |
| 位置信息 | `gps_longitude`、`gps_latitude`、`gps_height` |

默认勾选口径（`colForm`）为：设备信息默认含 `serial_number`、`product`、`module_type`、`host_name`、`software_version`、`group_name`、`mac_address`、`online_time`、`offline_time`；小区信息默认含 `PHYCELLID`；状态信息默认含 `op_state`、`halob_flag`、`ue_count`；网络信息默认含 `cell_ip`。

### 4. cpe_monitor_vue.jsp

CPE 的 Select columns 不在主表模板里直写，而是在关联页 `cpe_query.jsp` 中按 5 组定义：`deviceCol`、`lteCol`、`nrCol`、`lanCol`、`locationCol`。

| 分组 | 对应字段 |
| --- | --- |
| 设备信息 | `SERIAL_NUMBER`、`CPE_NAME`、`MODEL_NAME`、`PRODUCT`、`SOFTWARE_VERSION`、`UPTIME`、`first_online_time`、`LASTINFORMTIME`、`MCC`、`MNC`、`group_name`、`module_name`、`module_version`、`LGW_IP`、`MARKET_NAME`、`IMEI`、`cpe_model` |
| LTE 状态 | `IMSI`、`SCANMODE`、`PCI`、`HOST_NAME`、`CELL_IDENTITY`、`DL_EARFCN`、`BANDWIDTH`、`CINR0`、`CINR1`、`CPE_SINR`、`DL_CURRENT_DATARATE`、`UL_CURRENT_DATARATE`、`TX_POWER`、`RSRP0`、`RSRP1`、`UL_MCS`、`DL_MCS`、`lte_connection_time`、`DL_BLER` |
| NR 状态 | `NR_BAND`、`NR_BANDWIDTH`、`NR_PCI`、`NR_EARFCN`、`NR_PLMN`、`NR_CELL_ID`、`NR_DL_FREQUENCY`、`NR_UL_FREQUENCY`、`NR_CINR`、`NR_SINR`、`NR_RSRQ`、`NR_RSRP` |
| LAN 状态 | `MACADDRESS`、`IPADDRESS`、`LGW_MAC` |
| 位置信息 | `longitude`、`latitude`、`height`、`distance` |

默认勾选口径（`form`）为：设备信息默认含 `SERIAL_NUMBER`、`CPE_NAME`、`MODEL_NAME`、`SOFTWARE_VERSION`、`group_name`、`LGW_IP`、`cpe_model`；LTE 状态默认含 `IMSI`、`PCI`、`HOST_NAME`、`CELL_IDENTITY`；NR、LAN、位置默认不勾选。

### 5. 横向结论

- `enodeb`：分组最完整，覆盖设备、小区、状态、网络、位置、卫星 6 类。
- `gnodeb`：结构最接近 enodeb，但没有卫星组，改为 5 组。
- `cpe`：分组完全围绕终端侧能力展开，拆成设备、LTE、NR、LAN、位置 5 组。
- `gsm`：当前源码未见显式分组式 `Select columns`，实际是单列表列清单，不应机械套用其他页面的分组结构。

## 四个页面的查询和筛选条件

说明：

- 本节只整理主监控列表顶部查询区的“关键字搜索”和“高级筛选”条件。
- `enodeb`、`gsm`、`cpe` 的查询区分别来自关联页 `enb_query.jsp`、`gsm_query.jsp`、`cpe_query.jsp`；`gnodeb` 的查询区直接写在主页面中。
- 某些筛选项的 `options` 在源码初始值为空，实际由页面初始化后动态加载；这类项在下文标记为“动态候选”。

### 1. enodeb_monitor_vue.jsp

关键字搜索：

- 输入框默认占位：`请输入`
- 聚焦后可搜字段：
	- 常规场景：`serial_number`、`host_name`、`cell_ip`、`mac_address`、`cell_identity`、`phycellid`
	- 当 `enbAdditionalColShow == 'true'` 且 `northOperatorScenario == 'S0009'` 时，额外支持 `sub_station_name`
- 页面初始 `queryParams.like_fields`：`serial_number,host_name,cell_ip`
- 实际点击搜索时，会把 `like_fields` 扩展为上面的关键字搜索字段集合。

默认显示的高级筛选项：

- 在线状态 `connection_status`
	- 候选值：`1` 连接正常、`0` 连接断开、`3` 同步中、`2` 同步失败、`4` 初始化中、`5` 原同步中、`6` 原同步完成
- 是否激活 `op_state`
	- 候选值：全部、`1` 激活、`0` 去激活
- 产品类型标识 `product_model`
	- 候选值：动态加载

通过“添加筛选”可展开的附加条件：

- 设备型号名 `model_name`，动态候选
- 软件版本 `software_version`，动态候选
- 固件版本 `firmware_version`，动态候选
- 设备组 `group_id`，动态候选
- HaloB 开关 `halob_flag`
	- 候选值：全部、`1` 开、`0` 关

主表默认查询参数：

- `search_text: ''`
- `like_fields: 'serial_number,host_name,cell_ip'`
- `connection_status: []`
- `op_state: ''`
- `product_model: []`
- `model_name: []`
- `software_version: []`
- `firmware_version: []`
- `halob_flag: ''`
- `group_id: []`
- `halodSerialNumbers: ''`

### 2. gsm_monitor_vue.jsp

关键字搜索：

- 输入框占位直接写明可搜内容：`BSC编码 / BSC名称 / IP地址 / MAC / 所属BSC编码`
- 实际搜索字段：`serial_number`、`host_name`、`cell_ip`、`mac_address`、`cell_identity`、`phycellid`
- 页面初始 `queryParams.like_fields`：`serial_number,host_name,cell_ip`
- 点击搜索后会把 `like_fields` 扩展为完整字段集合。

默认显示的高级筛选项：

- 在线状态 `connection_status`
	- 候选值：`1` 连接正常、`0` 连接断开、`3` 同步中、`2` 同步失败
- 是否激活 `op_state`
	- 候选值：全部、`1` 激活、`0` 去激活
- 产品类型标识 `product_model`
	- 候选值：`BSC`、`BTS`

通过“添加筛选”可展开的附加条件：

- 设备型号名 `model_name`，动态候选
- 软件版本 `software_version`，动态候选
- 固件版本 `firmware_version`，动态候选
- 设备组 `group_id`，动态候选
- 所属 BSC 编码 `bscSerialnumber`，动态候选

主表默认查询参数：

- `search_text: ''`
- `like_fields: 'serial_number,host_name,cell_ip'`
- `connection_status: []`
- `op_state: ''`
- `product_model: 'BSC,BTS'`
- `model_name: []`
- `software_version: []`
- `firmware_version: []`
- `halob_flag: ''`
- `group_id: []`
- `halodSerialNumbers: ''`
- `bscSerialnumber: ''`

### 3. gnodeb_monitor.jsp

关键字搜索：

- 输入框默认占位：`请输入`
- 聚焦后可搜字段提示：`小站编码 / 5G站点名称 / IP地址`
- 实际搜索字段：`serial_number`、`host_name`、`cell_ip`
- 页面初始和执行搜索时的 `like_fields` 都保持为 `serial_number,host_name,cell_ip`

默认显示的高级筛选项：

- 在线状态 `connection_status`
	- 候选值：`1` 连接正常、`0` 连接断开、`3` 同步中、`2` 同步失败
- gNB 状态 `op_state`
	- 候选值：全部、`1` 激活、`0` 去激活
- 产品类型标识 `product_model`
	- 候选值：全部、`BaiBNX`、`BaiBNQ`

通过“添加筛选”可展开的附加条件：

- 设备型号名 `model_name`，动态候选
- 软件版本 `software_version`，动态候选
- 硬件版本 `firmware_version`，动态候选
- 设备组 `group_id`，动态候选
- Multi PLMN 状态 `multiPlmnEnable`
	- 候选值：全部、`1` 启用、`0` 禁用
- HaloB 开关 `halob_flag`
	- 候选值：全部、`1` 开、`0` 关

主表默认查询参数：

- `search_text: ''`
- `like_fields: 'serial_number,host_name,cell_ip'`
- `op_state: ''`
- `connection_status: ''`
- `software_version: ''`
- `firmware_version: ''`
- `group_id: ''`
- `product_model: ''`
- `model_name: ''`
- `halob_flag: ''`
- `multiPlmnEnable: ''`

### 4. cpe_monitor_vue.jsp

关键字搜索：

- 输入框默认占位：`请输入`
- 聚焦后可搜字段提示：`CPE编码 / CPEName / HostName / IMSI / IP / MAC / ECI / PCI / LGW IP / LGW MAC`
- 实际搜索字段：`HOST_NAME`、`CPE_NAME`、`macaddress`、`IMSI`、`serial_number`、`ipaddress`、`CELL_IDENTITY`、`pci`、`lgw_ip`、`lgw_mac`
- 主表初始 `queryParams.like_fields` 为空，点击搜索后由查询工具栏写入完整搜索字段集合。

默认显示的高级筛选项：

- 在线状态 `connection_status`
	- 候选值：`1` 连接正常、`0` 连接断开、`3` 同步中、`2` 同步失败
- 产品型号 `product_model`
	- 候选值：动态候选

通过“添加筛选”可展开的附加条件：

- 产品类型 `cpeMonitorModule`，动态候选
- 软件版本 `software_version`，动态候选
- 设备组 `group_id`，动态候选

主表默认查询参数：

- `search_text: ''`
- `like_fields: ''`
- `connection_status: ''`
- `cpeMonitorModule: ''`
- `product_model: ''`
- `software_version: ''`
- `group_id: ''`
- `isCloudCore: isCloudCore`

### 5. 横向差异总结

- `enodeb` 的筛选维度最复杂，在线状态比另外三页多出初始化中、原同步中、原同步完成 3 类状态。
- `gsm` 与 `enodeb` 结构最接近，但产品类型固定为 `BSC/BTS`，并多出“所属 BSC 编码”筛选。
- `gnodeb` 的关键字搜索最收敛，只搜站点编码、站点名、IP；但高级筛选里额外有 `multiPlmnEnable`。
- `cpe` 的关键字搜索字段最多，覆盖 CPE 编码、名称、IMSI、IP、MAC、ECI、PCI、LGW IP、LGW MAC；高级筛选则相对更少。


## 四个页面表格中的产品类型

说明：本节只整理“表格中用于展示产品类型的列”和“从当前页面源码里能够直接确认的产品类型值”。如果某个页面的产品类型候选值来自后端接口动态返回，则这里只记录源码里能静态确认的部分，不臆测完整枚举。

### 1. enodeb_monitor_vue.jsp

- 产品类型相关表格列：`product`（产品类型标识）、`product_name`（产品名称）
- 主表 `product` 列通过 `productFmt(row, row.product, rowIndex)` 直接输出原始值。
- 当前页面主表模板里能直接看到被特判的产品类型值：
	- `PM-B4860`
	- `QAFA`
	- `QATA`
	- `QAFB`
	- `RTD`
	- `BM`
	- `BAIBLQ`
	- `RTS`
	- `QRTB`
	- `QRTB-CA`
	- `QRTB-DC`
	- `QRTB-SC`
- 备注：上面这一组属于 enodeb 监控链路中可以确认存在的产品类型值，其中一部分未在 `enodeb_monitor_vue.jsp` 主表模板里直接分支判断，更可能作为隐藏筛选值或在详情/设置页中生效。

### 2. gsm_monitor_vue.jsp

- 产品类型相关表格列：`product`（产品类型标识）、`product_name`（产品名称）
- 主表 `product` 列同样通过 `productFmt(row, row.product, rowIndex)` 直接输出原始值。
- 源码中可直接确认的产品类型值：
	- `BSC`
	- `BTS`
- 依据：页面查询参数里默认 `product_model: 'BSC,BTS'`，同时多个主表模板分支按 `row.product == 'BSC'` 和 `row.product == 'BTS'` 分开渲染。

### 3. gnodeb_monitor.jsp

- 产品类型相关表格列：`product`（产品类型标识）、`product_name`（产品名称）
- 源码中静态写出的产品类型筛选项：
	- `BaiBNX`
	- `BaiBNQ`
- 其中主表模板里还能直接看到 `scope.row.product == 'BaiBNQ'` 的专门分支处理。
- 备注：页面初始化后还会调用 `${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=1` 重新装载 `product_model` 选项，因此运行时最终候选值仍可能以接口返回为准。

### 4. cpe_monitor_vue.jsp

- 产品类型相关表格列：`PRODUCT`（CPE 类型）、`MARKET_NAME`（产品名称）、`MODEL_NAME`（产品型号）、`cpe_model`（设备型号）
- 主表展示给用户的“产品类型”来自 `PRODUCT` 列，但最终通过 `CPETypeFmt(PRODUCT, row, index)` 归并后只显示两类：
	- `IDU`
	- `ODU`
- `CPETypeFmt` 的归并规则：
	- 当原始 `PRODUCT` 值等于 `LTE WiFi VoIP Gateway`，或以 `IDU` 开头时，表格显示为 `IDU`
	- 其他情况显示为 `ODU`
- 从同页操作逻辑中还能看到被识别的原始产品类型前缀/示例：
	- `LTE WiFi VoIP Gateway`
	- `IDU/CN...`
	- `IDU/EG...`
	- `ODU/EG...`
	- `IDU/u4G...`
	- `ODU/u4G...`

### 5. 汇总表

| 页面 | 表格中的产品类型字段 | 源码里能直接确认的产品类型 |
| --- | --- | --- |
| `enodeb_monitor_vue.jsp` | `product` | 主表特判值：`PM-B4860`、`QAFA`、`QATA`、`QAFB`、`RTD`；隐藏筛选/监控链路补充值：`BM`、`BAIBLQ`、`RTS`、`QRTB`、`QRTB-CA`、`QRTB-DC`、`QRTB-SC` |
| `gsm_monitor_vue.jsp` | `product` | `BSC`、`BTS` |
| `gnodeb_monitor.jsp` | `product` | `BaiBNX`、`BaiBNQ` |
| `cpe_monitor_vue.jsp` | `PRODUCT` | 表格最终展示为 `IDU`、`ODU` |

## 按产品类型整理“设置”操作中的功能面板

说明：

- 这里的“设置”操作，指列表里点击设置图标后打开的侧滑设置页，不包含“更多操作”菜单。
- 本节同时检查了监控页和其关联设置页。
- 如果某个页面的设置菜单来自后端接口动态返回，只记录源码里能静态确认的面板，不臆测完整枚举。
- enodeb 页面里，设置面板差异更多由 `platformType`、`dual_carrier_type`、在线状态控制，而不是单纯由表格 `product` 字段控制；因此会同时注明“产品类型口径”和“平台类型口径”的边界。

### 1. enodeb_monitor_vue.jsp

设置入口与关联页面：

- 监控页入口：`openSettingPage(row, page)`
- 设置页入口 URL：`${ctx}/enb/setting/openSettingPage.action`
- 关联设置页：`enodeb/monitor/settingPage/setting.jsp`

源码里能静态确认的设置页固定面板：

- 总览 `info`
- 统计 `chart`
- TOPO `topo`
- 告警管理 `alarm`
- 升级 `upgrade`
- 备份与恢复 `backup`
- 日志 `log`
- License `lic`
- 站点扫描 `scan`
- 有效期 `expiry`
- 流量限制 `limit`
- 分布式 `distribute`
- 测速 `diagnostic`

源码里能静态确认的“设置组”面板：

- `Quick Setting`
- `Basic`
- `Network`
- `LTE`
- `BTS`
- `LTE-TURBO`
- `Special`

说明：上述“设置组”来自 `getSettingGroupTree.action` 动态返回，源码里通过 `loadPage()` 和 `changeSettingTab()` 能静态确认这些面板代码，但最终页面上是否都出现，仍受后端返回结果控制。

按产品类型整理：

| 产品类型 | “设置”操作中包含的功能面板 | 差异说明 |
| --- | --- | --- |
| `PM-B4860` | 总览、统计、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（Quick Setting、Basic、Network、LTE、BTS、LTE-TURBO、Special，实际以接口返回为准） | 设置页里直接按产品值特判的是 `CR-B4860`、`QRTB`、`MLN` 这一组 `isNova452` 逻辑；监控页列表里的 `PM-B4860` 更常结合 `platformType=Intel_CR_* / MLN_*` 来决定面板细节 |
| `QAFA` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（实际以接口返回为准） | 当前设置页未看到只针对 `QAFA` 的独立面板增删分支；如果底层 `platformType` 落到 436Q/BLQ/MLQ 分支，则会继续按平台类型裁剪设置组 |
| `QATA` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（实际以接口返回为准） | 同上 |
| `QAFB` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（实际以接口返回为准） | 同上 |
| `RTD` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（实际以接口返回为准） | 当前设置页未看到只针对 `RTD` 的独立面板增删分支 |
| `BM` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（特别是 Quick Setting/Basic） | `BM` 在设置页中有明确分支：不显示统计；`Quick Setting` 走专用页面 `toBmQuickSettingPage.action` |
| `BAIBLQ` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（Quick Setting、Basic、Network、LTE、LTE-TURBO、Special，实际以接口返回为准） | 这类设备会进入 `platformType.indexOf('BLQ') >= 0` 的分支；`Quick Setting/Basic` 强制可用，`LTE-TURBO` 是否出现取决于 `isLWAEnable` |
| `RTS` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（实际以接口返回为准） | 当前设置页未看到只针对 `RTS` 的独立面板增删分支 |
| `QRTB` | 总览、统计、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（实际以接口返回为准） | `QRTB` 在设置页里直接落入 `isNova452` 分支，因此静态可确认包含统计面板 |
| `QRTB-CA` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（实际以接口返回为准） | 当前设置页未看到只针对 `QRTB-CA` 的独立面板增删分支 |
| `QRTB-DC` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（实际以接口返回为准） | 同上 |
| `QRTB-SC` | 总览、告警管理、升级、备份与恢复、日志、License、站点扫描、有效期、流量限制、分布式、测速、动态设置组（实际以接口返回为准） | 同上 |

补充说明：

- `platformType` 命中 `436Q / BLQ / BLX / MLQ` 分支时，设置组面板会按平台规则裁剪：`Quick Setting`、`Basic` 强制可用，`LTE-TURBO` 受 `isLWAEnable` 控制，部分辅波场景会限制 `LTE` 或 `Special`。
- `platformType == 'BLX'` 且网络模式满足条件时，还会额外出现 `TOPO` 面板；这个差异不是由 `product` 字段直接决定的。

### 2. gsm_monitor_vue.jsp

设置入口与关联页面：

- 监控页入口：`openSettingPage(row, page)`
- 设置页入口 URL：`${ctx}/cell/cpeinfos/toGSMMonitorSettingPages.action`
- 关联设置页：`enodeb/monitor/GSM/gsm_setting.jsp`

源码里能静态确认的设置页面板：

- 总览 `info`
- 告警 `alarm`
- 设置组：`basic`
- 设置组：`btsSetting`
- 版本管理 `upgrade`
- License `license`

按产品类型整理：

| 产品类型 | “设置”操作中包含的功能面板 | 差异说明 |
| --- | --- | --- |
| `BSC` | 总览、告警、eNB 基础配置、版本管理、License | `BSC` 不显示 `BTS Setting`；源码里旧的 `BTS` 面板入口还在，但已 `v-if="false"` 隐藏 |
| `BTS` | 总览、告警、eNB 基础配置、BTS Setting、版本管理、License | `BTS Setting` 只在 `product == 'BTS'` 时显示；离线时该面板会被禁用 |

### 3. gnodeb_monitor.jsp

设置入口与关联页面：

- 监控页入口：`settingBtnClick(row, page)`
- 设置页入口 URL：`${ctx}/gnb/setting/openSettingPage.action`
- 关联设置页：`gnodeb/monitor/settingPage/setting.jsp`

源码里能静态确认的设置页固定面板：

- 总览 `info`
- 统计 `chart`
- 告警 `alarm`
- TOPO `topo`
- 升级 `upgrade`
- 备份与恢复 `backup`
- 日志 `log`
- License `license`

源码里能静态确认的设置组面板：

- 快速设置 `quickSetting_X86` 或 `quickSetting_GT`
- 网络设置 `network`
- 核心网 `coreNetwork`
- RAN `ran`
- BTS `bts`
- System `system`

按产品类型整理：

| 产品类型 | “设置”操作中包含的功能面板 | 差异说明 |
| --- | --- | --- |
| `BaiBNX` | 总览、统计、TOPO、告警、升级、备份与恢复、日志、License、快速设置 | 非 `BaiBNQ` 只静态确认到 `quickSetting_X86` 这一项设置组面板；`TOPO` 对该类产品开放 |
| `BaiBNQ` | 总览、统计、告警、升级、备份与恢复、日志、License、快速设置、网络设置、核心网、RAN、BTS、System | `BaiBNQ` 不显示 `TOPO`，但会展开完整设置组，且快速设置走 `quickSetting_GT` |

### 4. cpe_monitor_vue.jsp

设置入口与关联页面：

- 监控页入口：`openSettingPage(row)`
- 设置页入口 URL：`${ctx}/cpe/setting/openSettingPage.action`
- 关联设置页：`cpe/monitor/settingPage/setting.jsp`

源码里能静态确认的设置页固定面板：

- 总览 `info`
- 统计 `chart`
- 升级 `upgrade`
- 日志 `log`
- 测速 `speed`
- 站点扫描 `scan`

源码里能静态确认的设置组面板：

- 基本设置 `basic`
- WIFI Config `wifi`
- 网络设置 `network`
- LTE/NR `lte`
- 系统 `system`
- APN/L2 设置 `apnSet`

按产品类型整理：

| 产品类型 | “设置”操作中包含的功能面板 | 差异说明 |
| --- | --- | --- |
| `IDU` | 总览、统计、升级、日志、测速、基本设置、网络设置、LTE/NR、系统 | `IDU` 是表格归并显示值，不等于原始产品值全集；是否还有 `APN/L2` 要看原始 `OLDPRODUCT` 是否命中 `IDU/EG` 或 `IDU/u4G` 类规则 |
| `ODU` | 总览、统计、升级、日志、测速、基本设置、网络设置、LTE/NR、系统 | 与 `IDU` 相同，是否显示 `APN/L2` 要看原始 `OLDPRODUCT` 是否命中 `ODU/EG` 或 `ODU/u4G` 类规则 |
| `LTE WiFi VoIP Gateway` | 总览、升级、基本设置、网络设置 | 这一类被归并显示为 `IDU`，但设置页中显式禁止 `APN/L2`，同时不显示统计、日志、测速、LTE/NR、系统 |
| `IDU/CN...` | 总览、统计、升级、日志、测速、基本设置、网络设置、LTE/NR、系统 | `IDU/CN` 类不会显示 `APN/L2` |
| `IDU/EG...` | 总览、统计、升级、日志、测速、基本设置、网络设置、LTE/NR、系统、APN/L2 设置 | `IDU/EG` 类会打开 `APN/L2` 设置 |
| `ODU/EG...` | 总览、统计、升级、日志、测速、基本设置、网络设置、LTE/NR、系统、APN/L2 设置 | 同上 |
| `IDU/u4G...` | 总览、统计、升级、日志、测速、基本设置、网络设置、LTE/NR、系统、APN/L2 设置 | 同上 |
| `ODU/u4G...` | 总览、统计、升级、日志、测速、基本设置、网络设置、LTE/NR、系统、APN/L2 设置 | 同上 |
| `R005` 类 | 总览、升级、基本设置、WIFI Config、网络设置 | `R005` 会隐藏统计、日志、测速、LTE/NR、系统、APN/L2 |
| `Nova430X` / `Neutrino430X`（43XAP） | 总览、升级、基本设置、网络设置 | 43XAP 会隐藏统计、日志、测速、WIFI Config、LTE/NR、系统、APN/L2 |

补充说明：

- `scan` 站点扫描只在 `cpe_model == '4G'` 时出现，因此它是型号/制式条件，不是纯产品类型条件。
- `lte` 面板还受 `wanShow` 影响，命中 `EP3011` 时会被隐藏。

### 5. 总结

- enodeb：设置页面板最复杂，固定面板较多，但真正的“设置组”来自后端接口，静态代码只能确认常见面板代码；产品类型差异主要落在 `BM`、`QRTB` 以及若干 `platformType` 分支上。
- gsm：产品类型差异最清晰，`BTS` 比 `BSC` 多一个 `BTS Setting` 面板。
- gnodeb：`BaiBNQ` 和 `BaiBNX` 的主要差异是设置组规模不同，`BaiBNQ` 有完整设置组，`BaiBNX` 只有快速设置；同时 `TOPO` 只对非 `BaiBNQ` 开放。
- cpe：表格展示只有 `IDU/ODU` 两类，但设置页真正分支是原始 `OLDPRODUCT`；`R005`、`43XAP`、`IDU/EG`、`ODU/EG`、`u4G` 这些原始产品前缀决定了是否显示 `WIFI Config`、`APN/L2`、`LTE/NR` 等面板。

## 按产品类型整理单条操作

说明：这里整理的是“列表操作列里的单条操作”。单条操作是否最终可点，还会受权限码、在线状态、是否真实可用站、能力位等条件控制。本节只总结源码里能静态确认的“按产品类型产生的差异”。如果某个产品类型在当前页面里没有单独分支，就记为“沿用通用单条操作”。

### 1. enodeb_monitor_vue.jsp

通用单条操作基线：

- 设置页
- 同步
- 重启
- 日志收集
- 收集报文
- 激活/去激活
- 射频开/关
- HaloB 开启/关闭
- SAS 注册 / 注销
- 配置恢复
- SAS 强制 RF / 自动 RF 控制

按产品类型整理：

| 产品类型 | 对应单条操作 | 差异说明 |
| --- | --- | --- |
| `PM-B4860` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 与通用基线一致，但“激活/去激活”走专门的多小区弹窗处理 |
| `QAFA` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 当前页面未看到专门的单条操作差异分支，主要影响的是 MME 状态展示 |
| `QATA` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 同上 |
| `QAFB` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 同上 |
| `RTD` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 当前页面未看到只针对 `RTD` 的单条菜单差异 |
| `BM` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 如果该产品类型出现在列表中，源码可确认其“激活/去激活”会细分出 LTE/GSM Cell 子项 |
| `BAIBLQ` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 监控链路里能确认该类产品进入的是专用设置面板分支，但菜单项本身没有额外增删 |
| `RTS` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 当前页面没有看到只针对 `RTS` 的单条菜单增删分支，先按通用单条操作列出；该类型更多出现在相关设置链路中 |
| `QRTB` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 当前页面没有看到只针对 `QRTB` 的单条菜单增删分支，先按通用单条操作列出；该类型更多出现在相关设置链路中 |
| `QRTB-CA` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 当前页面没有看到只针对 `QRTB-CA` 的单条菜单增删分支，先按通用单条操作列出；该类型更多出现在相关设置链路中 |
| `QRTB-DC` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 当前页面没有看到只针对 `QRTB-DC` 的单条菜单增删分支，先按通用单条操作列出；该类型更多出现在相关设置链路中 |
| `QRTB-SC` | 设置页、同步、重启、日志收集、收集报文、激活/去激活、射频开/关、HaloB、SAS 注册/注销、配置恢复、SAS RF 控制 | 当前页面没有看到只针对 `QRTB-SC` 的单条菜单增删分支，先按通用单条操作列出；该类型更多出现在相关设置链路中 |

补充说明：enodeb 页面里真正决定单条操作细分差异的，更多是 `platformType` 和 `dual_carrier_type`，不是 `product` 字段本身。例如 436Q 辅波只保留射频类操作，4860 CA/BM 会把激活操作展开成多小区子项。

### 2. gsm_monitor_vue.jsp

通用单条操作基线：

- 设置页
- 同步
- 重启
- 日志收集
- 收集报文

按产品类型整理：

| 产品类型 | 对应单条操作 | 差异说明 |
| --- | --- | --- |
| `BSC` | 设置页、同步、重启、日志收集、收集报文 | 当前源码没有为 `BSC` 单独增删“更多操作”菜单项；差异主要体现在列表字段展示为 `--` |
| `BTS` | 设置页、同步、重启、日志收集、收集报文 | 当前源码没有为 `BTS` 单独增删“更多操作”菜单项 |

补充说明：GSM 页面也存在基于 `platformType` 的激活状态展示分支，以及 `PM-B4860` 的特殊激活处理代码，但就当前产品类型列表 `BSC/BTS` 来看，源码里没有把单条菜单按 `BSC`/`BTS` 拆成两套不同能力。

### 3. gnodeb_monitor.jsp

通用单条操作基线：

- 设置/信息入口
- 同步
- 重启
- 激活/去激活
- 射频开/关
- HaloB 开启/关闭
- 日志收集
- 收集报文

按产品类型整理：

| 产品类型 | 对应单条操作 | 差异说明 |
| --- | --- | --- |
| `BaiBNX` | 设置/信息入口、同步、重启、激活/去激活、射频开/关、HaloB、日志收集、收集报文 | 当前页面未看到只针对 `BaiBNX` 的单条菜单差异 |
| `BaiBNQ` | 设置/信息入口、同步、重启、激活/去激活、射频开/关、HaloB、日志收集、收集报文 | 当前页面未看到只针对 `BaiBNQ` 的单条菜单差异；源码里 `BaiBNQ` 的特判主要体现在 `amf_status` 展示 |

### 4. cpe_monitor_vue.jsp

通用单条操作基线：

- 设置页
- 重启
- 收集报文
- LTE-TURBO 启用 / 禁用（受能力位和开关状态控制）

按产品类型整理：

| 产品类型 | 对应单条操作 | 差异说明 |
| --- | --- | --- |
| `IDU` | 设置页、重启、收集报文、可选的 LTE-TURBO 启用/禁用 | 当前页面未看到只对 `IDU` 单独增删菜单项；是否显示 LTE-TURBO 主要看 `CAPABILITY`、`LTE_TURBO_ENABLE`、`isLWAEnable` |
| `ODU` | 设置页、重启、收集报文、可选的 LTE-TURBO 启用/禁用 | 同上 |
| `LTE WiFi VoIP Gateway` | 设置页、重启、收集报文、可选的 LTE-TURBO 启用/禁用 | 该原始产品值会被归并显示为 `IDU`，但当前菜单项没有单独新增/删除 |
| `IDU/CN...` | 设置页、重启、收集报文、可选的 LTE-TURBO 启用/禁用 | 仅在产品识别逻辑中作为 IDU 类前缀处理 |
| `IDU/EG...` | 设置页、重启、收集报文、可选的 LTE-TURBO 启用/禁用 | 同上 |
| `ODU/EG...` | 设置页、重启、收集报文、可选的 LTE-TURBO 启用/禁用 | 同上 |
| `IDU/u4G...` | 设置页、重启、收集报文、可选的 LTE-TURBO 启用/禁用 | 同上 |
| `ODU/u4G...` | 设置页、重启、收集报文、可选的 LTE-TURBO 启用/禁用 | 同上 |

### 5. 总结

- enodeb：产品类型里只有 `PM-B4860` 能明确看到单条“激活/去激活”采用特殊处理；其余类型更多影响展示，不明显改变菜单结构。
- gsm：按当前产品类型列表 `BSC/BTS`，未发现单条菜单能力差异，差异主要体现在字段展示。
- gnodeb：`BaiBNX`、`BaiBNQ` 的单条操作集合相同，`BaiBNQ` 的差异主要是状态展示，不是菜单项。
- cpe：当前单条菜单没有按 `IDU/ODU` 做显式拆分，更多是产品识别后影响设置页内部逻辑；菜单层面仍以通用操作为主。

## 四个页面的批量操作

说明：这里优先整理“当前页面顶部工具栏中已绑定的批量操作按钮”。另外，所有页面都有“已选列表”浮层，支持查看选中记录、清空选中项、删除单条选中项。

### 1. enodeb_monitor_vue.jsp

当前页面可见的批量操作：

- 移动到设备组：`movecells`，打开设备组选择弹窗后调用 `deviceGroupMoveSubmit` 批量移动
- 同步：`syncList`，对选中的 `small_cell_code` 发起批量同步
- 重启：`rebootList`，调用 `batchRebootCell.action` 批量重启
- 回收站：`recycleCells`，把选中设备批量移入回收站

批量选择辅助操作：

- 打开已选列表：`openBulkSelectTable`
- 清空已选：`clearBulkSelected`
- 删除单条已选：`delBulkSelected`

### 2. gsm_monitor_vue.jsp

当前页面可见的批量操作：

- 移动到设备组：`movecells`
- 同步：`syncList`
- 重启：`rebootList`

批量选择辅助操作：

- 打开已选列表：`openBulkSelectTable`
- 清空已选：`clearBulkSelected`
- 删除单条已选：`delBulkSelected`

备注：GSM 页面当前顶部没有“批量移入回收站”按钮。

### 3. gnodeb_monitor.jsp

当前页面可见的批量操作：

- 同步：`batchSyncList`，打开同步对话框，提交时走 `batchSyncCell.action`
- 重启：`batchRebootList`，调用 `batchRebootCell.action`，并带 `isGnb=1`
- 回收站：`recycleCells`，把选中 gNodeB 批量移入回收站

批量选择辅助操作：

- 打开已选列表：`openBulkSelectTable`
- 清空已选：`clearBulkSelected`
- 删除单条已选：`delBulkSelected`

备注：gNodeB 页面当前顶部没有“移动到设备组”按钮。

### 4. cpe_monitor_vue.jsp

当前页面可见的批量操作：

- 重启：`rebootCpeList`
- 修改密码：`openModifyPwd`，提交时由 `savePassword` 创建批量改密任务
- LTE-TURBO 启用：`ltmkai('1')`
- LTE-TURBO 禁用：`ltmkai('0')`
- 回收站：`recycleCells`，把选中 CPE 批量移入回收站

批量选择辅助操作：

- 打开已选列表：`openBulkSelectTable`
- 清空已选：`clearBulkSelected`
- 删除单条已选：`delBulkSelected`

代码中存在但当前页面顶部未直接绑定按钮的批量方法：

- 批量参数配置：`paramsConfigBatch`
- 批量恢复出厂：`restoreFactoryBatch` / `confirmRestoreFactory`

### 5. 汇总表

| 页面 | 当前可见批量操作 | 是否有设备组移动 | 是否有回收站 | 备注 |
| --- | --- | --- | --- | --- |
| `enodeb_monitor_vue.jsp` | 移动到设备组、同步、重启、回收站 | 是 | 是 | 批量同步和重启都基于 `small_cell_code` |
| `gsm_monitor_vue.jsp` | 移动到设备组、同步、重启 | 是 | 否 | 页面没有批量回收站入口 |
| `gnodeb_monitor.jsp` | 同步、重启、回收站 | 否 | 是 | 批量操作走 gNodeB 专用权限码和 `isGnb=1` |
| `cpe_monitor_vue.jsp` | 重启、修改密码、LTE-TURBO 启用、LTE-TURBO 禁用、回收站 | 否 | 是 | 另有未直接暴露的批量参数配置/恢复出厂方法 |

## 四个页面单条操作与批量操作对照

说明：这里的“单条操作”指主表操作列中的设置图标、更多操作图标及其菜单项；“批量操作”指页面顶部基于多选记录触发的操作。

### 1. enodeb_monitor_vue.jsp

单条操作：

- 设置页：`openSettingPage`
- 同步
- 重启
- 激活/去激活
- 射频开/关
- HaloB 开启/关闭
- 日志收集
- 收集报文
- SAS 注册 / 注销
- 配置恢复
- SAS 强制 RF / 自动 RF 控制

批量操作：

- 移动到设备组
- 同步
- 重启
- 回收站

差异：

- 只有单条有：设置、激活/去激活、射频开关、HaloB、日志收集、收集报文、SAS 注册/注销、配置恢复、SAS RF 控制
- 只有批量有：移动到设备组、回收站
- 单条和批量都有：同步、重启

### 2. gsm_monitor_vue.jsp

单条操作：

- 设置页：`openSettingPage`
- 同步
- 重启
- 激活/去激活
- HaloB 开启/关闭
- 日志收集
- 收集报文
- SAS 注册 / 注销

批量操作：

- 移动到设备组
- 同步
- 重启


### 3. gnodeb_monitor.jsp

单条操作：

- 设置/信息入口：`settingBtnClick(scope.row,'info')`
- 同步
- 重启
- 激活/去激活
- 射频开/关
- HaloB 开启/关闭
- 日志收集
- 收集报文

批量操作：

- 同步
- 重启
- 回收站


备注：当前 gNodeB 主表操作列同时有单独的设置/信息图标和“更多操作”图标。

### 4. cpe_monitor_vue.jsp

单条操作：

- 设置页：`openSettingPage`
- 重启
- 收集报文
- LTE-TURBO 启用 / 禁用

批量操作：

- 重启
- 修改密码
- LTE-TURBO 启用
- LTE-TURBO 禁用
- 回收站


备注：代码里还存在批量参数配置、批量恢复出厂方法，但当前页面顶部未直接挂出入口。

### 5. 总览矩阵

| 页面 | 单条设置入口 | 单条更多操作 | 批量操作数量 | 单条/批量共有能力 | 只在单条出现 | 只在批量出现 |
| --- | --- | --- | ---: | --- | --- | --- |
| `enodeb_monitor_vue.jsp` | 有 | 有 | 4 | 同步、重启 | 设置、激活/去激活、射频、HaloB、日志、收集报文、SAS、配置恢复 | 移动到设备组、回收站 |
| `gsm_monitor_vue.jsp` | 有 | 有 | 3 | 同步、重启 | 设置、激活/去激活、HaloB、日志、收集报文、SAS | 移动到设备组 |
| `gnodeb_monitor.jsp` | 有 | 有 | 3 | 同步、重启 | 设置/信息、激活/去激活、射频、HaloB、日志、收集报文 | 回收站 |
| `cpe_monitor_vue.jsp` | 有 | 有 | 5 | 重启、LTE-TURBO 启用/禁用 | 设置、收集报文 | 修改密码、回收站 |


