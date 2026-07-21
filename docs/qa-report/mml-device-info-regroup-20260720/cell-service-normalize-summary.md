# 小区服务参数管理规范化汇总

日期：2026-07-21

## 范围

- 章节：`SF`
- 目录：`小区服务参数管理（总体）`
- catalog：`omcgo/data/mml-catalog/cmcc-tdlte-v2.3.json`
- 本地库：`goomc-local-postgres-1 / omcgo`

## 原始状态

处理前本地库中小区服务相关命令绑定数量如下：

| 命令 | 原绑定数 | 原 target_paths | 原 tree_node_refs | 问题 |
|---|---:|---:|---:|---|
| `LST FAP_SERVICE` | 701 | 701 | 0 | 混入大量非 TD-LTE SF catalog 白名单路径，且树引用为空 |
| `MOD FAP_SERVICE` | 698 | 698 | 0 | 混入大量非 TD-LTE SF catalog 白名单路径，且树引用为空 |
| `LST PLMN_LIST` | 4 | 4 | 0 | 字段需保留，但树引用为空 |
| `MOD PLMN_LIST` | 4 | 4 | 0 | 字段需保留，但树引用为空 |
| `LST PDCP_INIT_PARAM` | 1 | 1 | 0 | 字段正确，但树引用为空 |
| `MOD PDCP_INIT_PARAM` | 1 | 1 | 0 | 字段正确，但树引用为空 |
| `ADD/RMV PLMN_LIST` | 0 | 0 | 0 | 对象命令 path 未同步到树引用 |
| `ADD/RMV PDCP_INIT_PARAM` | 0 | 0 | 0 | 对象命令 path 未同步到树引用 |

文件侧问题：

- `FAP 载波基本配置` 命令码为空：
  - `LST `
  - `MOD `
- `EPC PLMN 列表` 为 empty group：
  - `paths: null`
  - `commands: null`
- `VoLTE PDCP 初始参数` 为 empty group：
  - `paths: null`
  - `commands: null`

## 新增内容

`EPC PLMN 列表` 补齐 catalog 中缺失的 4 个字段：

| 字段 | 权限 | 类型 |
|---|---|---|
| `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID` | `READ_WRITE` | `string(6)` |
| `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.CellReservedForOperatorUse` | `READ_WRITE` | `boolean` |
| `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.Enable` | `READ_WRITE` | `boolean` |
| `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary` | `READ_WRITE` | `boolean` |

`VoLTE PDCP 初始参数` 新增 1 个 TD-LTE V2.3 `SF` 规范字段：

| 字段 | 权限 | 类型 |
|---|---|---|
| `Device.Services.FAPService.{i}.CellConfig.LTE.VoLTE.PdcpInitParam.{i}.RohcEn` | `READ_WRITE` | `string(64)` |

文件侧补齐命令：

| 分组 | 新增命令 |
|---|---|
| `EPC PLMN 列表` | `LST PLMN_LIST`、`MOD PLMN_LIST`、`ADD PLMN_LIST`、`RMV PLMN_LIST` |
| `VoLTE PDCP 初始参数` | `LST PDCP_INIT_PARAM`、`MOD PDCP_INIT_PARAM`、`ADD PDCP_INIT_PARAM`、`RMV PDCP_INIT_PARAM` |
| `基站配置参数管理` | `LST/MOD/ADD/RMV NR_AMF_POOL_CONFIG_PARAM`、`LST/MOD/ADD/RMV NR_XN_IP_ADDR_MAP_INFO`、`LST/MOD NR_RAN_COMMON` |

NR `SF` 页新增同级顶层分组 `基站配置参数管理`，并按旧小区服务参数格式在同一层级挂载命令。新增 18 个字段：

AMF 地址：

| 字段 | 权限 | 类型 |
|---|---|---|
| `Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfName` | `READ_ONLY` | `string(150)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.PLMNID` | `READ_WRITE` | `string(6)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfRegionID` | `READ_ONLY` | `string(8)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfSetID` | `READ_ONLY` | `string(10)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfPointer` | `READ_ONLY` | `string(6)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.RelativeAmfCapacity` | `READ_ONLY` | `unsignedInt[0:255]` |
| `Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP1` | `READ_WRITE` | `string(64)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP2` | `READ_WRITE` | `string(64)` |

Xn 链接信息：

| 字段 | 权限 | 类型 |
|---|---|---|
| `Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.TAISupportList.{i}.TAC` | `READ_ONLY` | `string(24)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.TAISupportList.{i}.BroadcastPLMNs.{i}.PLMNID` | `READ_ONLY` | `string(6)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.gNBID` | `READ_WRITE` | `unsignedInt` |
| `Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.RemoteAddress` | `READ_WRITE` | `string(64)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.SubnetMask` | `READ_ONLY` | `string(64)` |
| `Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.LocalAddress` | `READ_ONLY` | `string(64)` |

gNB 参数：

| 字段 | 权限 | 类型 |
|---|---|---|
| `Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.AdminState` | `READ_WRITE` | `unsignedInt[1:3]` |
| `Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBId` | `READ_WRITE` | `unsignedInt[0:4294967295]` |
| `Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBIdLength` | `READ_WRITE` | `unsignedInt[22:32]` |
| `Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBName` | `READ_WRITE` | `string(150)` |

命令码修正：

| 原命令码 | 新命令码 |
|---|---|
| `LST ` | `LST FAP_SERVICE` |
| `MOD ` | `MOD FAP_SERVICE` |

## 删除内容

按 catalog 白名单从本地库删除旧绑定：

| 命令 | 原绑定数 | 现绑定数 | 删除绑定数 |
|---|---:|---:|---:|
| `LST FAP_SERVICE` | 701 | 34 | 667 |
| `MOD FAP_SERVICE` | 698 | 32 | 666 |
| `LST PLMN_LIST` | 4 | 4 | 0 |
| `MOD PLMN_LIST` | 4 | 4 | 0 |
| 合计 | 1407 | 74 | 1333 |

`PLMN_LIST` 保留的产品字段：

- `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.Enable`
- `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary`

`FAP_SERVICE` 删除的是白名单外旧扩展/私有/跨分组路径绑定；本次不迁移、不复制，只从小区服务主命令移除，避免继续污染 SF 命令树。

逐条删除 TR path 明细见：

- [cell-service-deleted-trpaths.csv](cell-service-deleted-trpaths.csv)

## 保留说明

`PLMN_LIST` 保留 2 个产品口径字段：

- `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.Enable`
- `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary`

`FAP_SERVICE` 中保留 10 个 `RRCTimers` 字段：

- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300`
- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301`
- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302`
- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA`
- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT`
- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310`
- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311`
- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320`
- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310`
- `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311`

这 10 项不是 TD-LTE `SF` 页字段，但属于既有设计文档记录的 `SH/RRCTimers` 合并项，本次保留。

## 最终状态

| 命令 | sub_fields | target_paths | tree_node_refs | 一致 |
|---|---:|---:|---:|---|
| `LST FAP_SERVICE` | 34 | 34 | 34 | true |
| `MOD FAP_SERVICE` | 32 | 32 | 32 | true |
| `LST PLMN_LIST` | 4 | 4 | 4 | true |
| `MOD PLMN_LIST` | 4 | 4 | 4 | true |
| `LST PDCP_INIT_PARAM` | 1 | 1 | 1 | true |
| `MOD PDCP_INIT_PARAM` | 1 | 1 | 1 | true |
| `ADD PLMN_LIST` | 0 | 1 | 1 | true |
| `RMV PLMN_LIST` | 0 | 1 | 1 | true |
| `ADD PDCP_INIT_PARAM` | 0 | 1 | 1 | true |
| `RMV PDCP_INIT_PARAM` | 0 | 1 | 1 | true |
| `LST NR_AMF_POOL_CONFIG_PARAM` | 8 | 8 | 8 | true |
| `MOD NR_AMF_POOL_CONFIG_PARAM` | 3 | 3 | 3 | true |
| `ADD NR_AMF_POOL_CONFIG_PARAM` | 0 | 1 | 1 | true |
| `RMV NR_AMF_POOL_CONFIG_PARAM` | 0 | 1 | 1 | true |
| `LST NR_XN_IP_ADDR_MAP_INFO` | 6 | 6 | 6 | true |
| `MOD NR_XN_IP_ADDR_MAP_INFO` | 2 | 2 | 2 | true |
| `ADD NR_XN_IP_ADDR_MAP_INFO` | 0 | 1 | 1 | true |
| `RMV NR_XN_IP_ADDR_MAP_INFO` | 0 | 1 | 1 | true |
| `LST NR_RAN_COMMON` | 4 | 4 | 4 | true |
| `MOD NR_RAN_COMMON` | 4 | 4 | 4 | true |

## 落地文件

- `omcgo/data/mml-catalog/cmcc-tdlte-v2.3.json`
- `omcgo/migrations/seed/000002_normalize_sf_cell_service_catalog.sql`
- `omcgo/migrations/seed/000003_add_nr_sf_base_station_config_catalog.sql`
- `docs/qa-report/mml-device-info-regroup-20260720/management-server-normalize.md`
- `docs/qa-report/mml-device-info-regroup-20260720/cell-service-deleted-trpaths.csv`
