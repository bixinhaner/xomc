# MML 设备信息分组清理归档

日期：2026-07-20

## 背景

`设备信息参数管理` 分组下的 `LST DEVICE_INFO`、`MOD DEVICE_INFO` 命令混入了大量非设备基础信息字段，例如 `WAN_CONFIG*`、`ROUTE_CONFIG*`、`AntennaInfo.*`、`EU/RU`、`GPS`、`1588/PTP`、`SAS/CBRS` 等。

本次处理目标是清理设备信息分组：

- 只保留真正属于设备基础信息/设备软件升级状态的字段。
- 对分析中应归属其他分组的字段，不迁移、不复制，直接从设备信息分组删除，避免和其他已有分组重复。
- 对标准规范未覆盖或对象占位字段，也从设备信息分组删除。

## 输入依据

工作区输入文件：

- `/Users/wangyong/OBJECT/Codex/MML配置参数分组分析报告-设备信息重分组-20260720.md`
- `/Users/wangyong/OBJECT/Codex/MML配置参数分组-设备信息重分组-20260720.xlsx`
- `/Users/wangyong/OBJECT/Codex/中国移动5G扩展型皮基站网管南向接口数据模型规范v1.9.4.xlsx`
- `/Users/wangyong/OBJECT/Codex/中国移动TD-LTE皮站_飞站基站设备网络管理南向接口数据配置模型规范V2.3.xlsx`

项目内执行脚本：

- `omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql`

## 处理结果

数据库已直接执行到本地 `goomc-local-postgres-1 / omcgo`。

最终剩余：

| 命令 | 剩余绑定 |
|---|---:|
| `LST DEVICE_INFO` | 24 |
| `MOD DEVICE_INFO` | 5 |
| `LST DEVICE_INFO_SW_UPGRADE` | 3 |

删除结果：

| 命令 | 删除绑定 |
|---|---:|
| `LST DEVICE_INFO` | 419 |
| `MOD DEVICE_INFO` | 397 |

合计删除 816 条命令-path 绑定，涉及 419 个去重标准路径。

## 归档文件

- [process-log.md](process-log.md)：处理过程、执行 SQL、验证 SQL、问题修正记录。
- [delete-and-keep-detail.md](delete-and-keep-detail.md)：删除与保留的完整 Markdown 明细。
- [delete-detail.csv](delete-detail.csv)：删除绑定明细，便于表格筛选。
- [keep-detail.csv](keep-detail.csv)：最终保留绑定明细。
- [software-version-patch-info.md](software-version-patch-info.md)：软件版本参数分组缺失项补充记录。
- [management-server-normalize.md](management-server-normalize.md)：基站网管参数分组规范化记录。
- [log-management.md](log-management.md)：日志参数分组缺失项补充记录。
- [cell-service-normalize-summary.md](cell-service-normalize-summary.md)：小区服务参数管理规范化汇总，包含原始数量、新增字段、删除字段与最终状态。
- [cell-service-deleted-trpaths.csv](cell-service-deleted-trpaths.csv)：小区服务参数管理删除 TR path 明细。

## 关键注意

清理时必须同时刷新 `mml_commands.target_paths` 和 `mml_commands.tree_node_refs`。

第一次执行只清理了 `mml_command_sub_fields` 和 `target_paths`，页面仍显示大量 `WAN_CONFIG*`。排查发现前端/接口优先读 `tree_node_refs`；随后已将 `tree_node_refs` 同步为当前剩余 sub-fields 聚合结果，并重启 `goomc-local-app-1`。

最终验证：

- `target_paths = tree_node_refs`
- `tree_node_refs` 中 `WAN_CONFIG` 泄漏数为 0
- 三个设备信息命令剩余绑定总数为 32

## 后续补充：软件版本参数分组

按两份南向规范复核 `SB / 软件版本参数管理` 后，补齐 5G v1.9.4 中存在、当前库缺失的 `Device.SoftwareCtrl.PatchInfo`。

本地数据库已执行 `omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql`：

- `LST SOFTWARE_CTRL` 新增 `PATCH_INFO -> Device.SoftwareCtrl.PatchInfo`，排序为 6。
- 删除 `AccCard1PpsDelay`、`N48N78SharedRfEnable` 两个规范外扩展绑定。
- `LST SOFTWARE_CTRL` 当前 6 条绑定，含 5 项基础版本控制和 1 项 `PatchInfo`。
- `MOD SOFTWARE_CTRL` 当前 3 条绑定。
- `LST/MOD SOFTWARE_CTRL` 的 `target_paths` 与 `tree_node_refs` 已同步一致。

## 后续补充：基站网管参数分组

按两份南向规范复核 `SC / 基站网管参数管理` 后，确认共同包含 19 个 `Device.ManagementServer.*` 标准字段。

本地数据库已执行 `omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql`：

- 保留 `X_COM_tr069_port`、`sslStatus.endDate`、`sslStatus.startDate` 3 个产品扩展绑定。
- 删除 `tfcsManagerPrimsrc`、`tfcsSyncState` 2 个规范外扩展绑定。
- `LST MANAGEMENT_SERVER` 当前 22 条绑定。
- `MOD MANAGEMENT_SERVER` 当前 18 条绑定。
- `LST/MOD MANAGEMENT_SERVER` 的 `target_paths` 与 `tree_node_refs` 已同步一致。

## 后续补充：日志参数分组

按两份南向规范复核 `SE / 日志参数管理` 后，确认 TD-LTE V2.3 包含 5 个 `Device.LogMgmt.*` 字段，5G v1.9.4 额外包含 `Device.LogMgmt.LogLevel`。

本地数据库已执行 `omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql`：

- `LST LOG_MGMT`、`MOD LOG_MGMT` 新增 `LOG_LEVEL -> Device.LogMgmt.LogLevel`，排序为 6。
- `LST LOG_MGMT` 当前 6 条绑定。
- `MOD LOG_MGMT` 当前 6 条绑定。
- `LST/MOD LOG_MGMT` 的 `target_paths` 与 `tree_node_refs` 已同步一致。

## 后续补充：小区服务参数分组

按 TD-LTE V2.3 规范复核 `SF / 小区服务参数管理（总体）` 后，确认 `SF` 页包含 58 个 TR-181 标准字段。

本次处理落地到：

- `omcgo/data/mml-catalog/cmcc-tdlte-v2.3.json`
- `omcgo/migrations/seed/000002_normalize_sf_cell_service_catalog.sql`
- `omcgo/migrations/seed/000003_add_nr_sf_base_station_config_catalog.sql`
- `docs/qa-report/mml-device-info-regroup-20260720/cell-service-normalize-summary.md`

处理摘要：

- 补齐 `EPC PLMN 列表` 4 个字段：`PLMNID`、`CellReservedForOperatorUse`、`Enable`、`IsPrimary`。
- 补齐 `VoLTE PDCP 初始参数` 1 个字段：`RohcEn`。
- 补齐 `PLMN_LIST`、`PDCP_INIT_PARAM` 的 `LST/MOD/ADD/RMV` catalog 配置。
- 补齐 NR `SF` 页 `基站配置参数管理`，采用与小区服务参数一致的单层命令结构，新增 AMF 地址、Xn 链接信息、gNB 参数相关命令。
- 修正 `FAP 载波基本配置` 空命令码：`LST ` / `MOD ` 改为 `LST FAP_SERVICE` / `MOD FAP_SERVICE`。
- 本地库 `LST FAP_SERVICE` 从 701 条绑定收敛到 34 条，删除 667 条。
- 本地库 `MOD FAP_SERVICE` 从 698 条绑定收敛到 32 条，删除 666 条。
- 本地库 `LST/MOD PLMN_LIST` 保留 4 条绑定，其中 `Enable`、`IsPrimary` 按产品口径保留。
- 本地库 `基站配置参数管理` 直接挂 10 条命令：AMF 地址、Xn 链接信息各含查询/修改/添加/删除，gNB 参数含查询/修改。
- 本地库小区服务主命令合计删除 1333 条白名单外 TR path 绑定，明细见 `cell-service-deleted-trpaths.csv`。
- 小区服务相关 `LST/MOD/ADD/RMV` 命令的 `target_paths` 与 `tree_node_refs` 已同步一致。

备注：

- `FAP_SERVICE` 当前保留 10 个 `RRCTimers` 字段；这些字段来源于既有 `SH/RRCTimers` 合并记录，本次未删除。
- `PLMN_LIST` 当前保留 `Enable`、`IsPrimary`；这两个字段按产品口径保留。
