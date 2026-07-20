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
