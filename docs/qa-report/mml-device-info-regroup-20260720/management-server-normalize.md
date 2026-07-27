# 基站网管参数分组规范化记录

日期：2026-07-20

## 复核依据

输入规范：

- `/Users/wangyong/OBJECT/Codex/中国移动5G扩展型皮基站网管南向接口数据模型规范v1.9.4.xlsx`
- `/Users/wangyong/OBJECT/Codex/中国移动TD-LTE皮站_飞站基站设备网络管理南向接口数据配置模型规范V2.3.xlsx`

复核范围：

- `SC` 页，目录名称为 `基站网管参数管理`。

## 规范字段

两份规范共同包含 19 个 `Device.ManagementServer.*` 字段：

- `Device.ManagementServer.URL`
- `Device.ManagementServer.Username`
- `Device.ManagementServer.Password`
- `Device.ManagementServer.PeriodicInformEnable`
- `Device.ManagementServer.PeriodicInformTime`
- `Device.ManagementServer.PeriodicInformInterval`
- `Device.ManagementServer.ParameterKey`
- `Device.ManagementServer.ConnectionRequestURL`
- `Device.ManagementServer.ConnectionRequestUsername`
- `Device.ManagementServer.ConnectionRequestPassword`
- `Device.ManagementServer.UDPConnectionRequestAddress`
- `Device.ManagementServer.STUNEnable`
- `Device.ManagementServer.STUNServerAddress`
- `Device.ManagementServer.STUNServerPort`
- `Device.ManagementServer.STUNUsername`
- `Device.ManagementServer.STUNPassword`
- `Device.ManagementServer.STUNMaximumKeepAlivePeriod`
- `Device.ManagementServer.STUNMinimumKeepAlivePeriod`
- `Device.ManagementServer.NATDetected`

`MOD MANAGEMENT_SERVER` 只保留其中 15 个可写字段，不包含 `ParameterKey`、`ConnectionRequestURL`、`UDPConnectionRequestAddress`、`NATDetected`。

当前库已有且按产品口径需保留的扩展项：

- `Device.ManagementServer.X_COM_tr069_port`
- `Device.ManagementServer.sslStatus.endDate`
- `Device.ManagementServer.sslStatus.startDate`

当前库已有但规范未覆盖、需删除的扩展项：

- `Device.ManagementServer.tfcsManagerPrimsrc`
- `Device.ManagementServer.tfcsSyncState`

## 本次处理

合并到 seed 脚本：

```text
omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql
```

处理内容：

- 显式保留 `LST MANAGEMENT_SERVER`、`MOD MANAGEMENT_SERVER` 中 3 个产品扩展绑定。
- 删除 `LST MANAGEMENT_SERVER`、`MOD MANAGEMENT_SERVER` 中 2 个 `tfcs*` 规范外扩展绑定。
- 同步刷新 `LST MANAGEMENT_SERVER`、`MOD MANAGEMENT_SERVER` 的 `target_paths` 与 `tree_node_refs`。
- 增加独立 final sync，避免触发器刷新 `target_paths` 后 `tree_node_refs` 保留旧值。

## 执行与验证

本地库已直接执行 Up 段 SQL。

验证结果：

| 命令 | target_paths | tree_node_refs | 一致 |
|---|---:|---:|---|
| `LST MANAGEMENT_SERVER` | 22 | 22 | true |
| `MOD MANAGEMENT_SERVER` | 18 | 18 | true |

删除后泄漏检查：

- `mml_command_sub_fields` 中 2 个 `tfcs*` 规范外扩展绑定残留数为 0。
- `tree_node_refs` 中 2 个 `tfcs*` 规范外扩展路径残留数为 0。

## 告警参数管理只读化

范围：

- `SD` 页，目录名称为 `告警参数管理`。

调整目标：

- 页面只支持查询当前告警和历史告警。
- 不支持修改、添加、删除等设备侧写操作。
- 不展示实时告警、队列告警、支持告警类型和故障管理总览等辅助入口。

本次处理：

- `omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql` 增加 SD 告警参数清理段。
- `omcgo/data/mml-catalog/cmcc-tdlte-v2.3.json` 保留：
  - `LST FAULT_MGMT_CURRENT_ALARM`
  - `LST FAULT_MGMT_HISTORY_EVENT`
- `omcgo/datamodels/mml-catalog/cmcc-tdlte-v2.3.json` 的 `告警参数管理` 章节保留：
  - `LST:Device.FaultMgmt.CurrentAlarm.{i}.*`
  - `LST:Device.FaultMgmt.HistoryEvent.{i}.*`

执行与验证：

- 本地库已直接执行 Up 段 SQL。
- `mml_commands` 中告警相关命令仅保留 2 条 `FAULT_MGMT` 查询命令。
- 被移除的实时告警、队列告警、支持告警类型、故障管理总览与告警对象写操作命令残留数为 0。

备注：

- `ADD/RMV` 对象命令的执行路径保存在 `mml_commands.target_object`，不依赖 `mml_command_sub_fields`。
- MML 配置页的 PATH 列表原先只展示 sub-field，因此对象命令会显示为空；已调整为在无 sub-field 时展示只读目标对象路径。

## 小区服务参数管理规范化

范围：

- `SF` 页，目录名称为 `小区服务参数管理（总体）`。

复核结果：

- TD-LTE V2.3 `SF` 页包含 58 个 TR-181 标准字段。
- 当前 `cmcc-tdlte-v2.3` catalog 原缺少 5 个多实例字段/产品保留字段：
  - `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID`
  - `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.CellReservedForOperatorUse`
  - `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.Enable`
  - `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary`
  - `Device.Services.FAPService.{i}.CellConfig.LTE.VoLTE.PdcpInitParam.{i}.RohcEn`
- 当前 `FAP 载波基本配置` 保留 10 个 `RRCTimers` 字段；这些字段来源于既有 `SH/RRCTimers` 合并记录，本次保留。

本次处理：

- 新增 seed 迁移：
  - `omcgo/migrations/seed/000002_normalize_sf_cell_service_catalog.sql`
- `omcgo/data/mml-catalog/cmcc-tdlte-v2.3.json` 补齐：
  - `EPC PLMN 列表` 的 4 个字段及 `LST/MOD/ADD/RMV PLMN_LIST`
  - `VoLTE PDCP 初始参数` 的 1 个字段及 `LST/MOD/ADD/RMV PDCP_INIT_PARAM`
  - NR `SF` 页新增 `基站配置参数管理`，按旧小区服务参数格式在同一层级挂载 AMF 地址、Xn 链接信息、gNB 参数命令
- 修正 `FAP 载波基本配置` 的空命令码：
  - `LST ` -> `LST FAP_SERVICE`
  - `MOD ` -> `MOD FAP_SERVICE`
- 本地库按 catalog 白名单收敛：
  - `LST FAP_SERVICE` 从 701 条绑定收敛到 34 条。
  - `MOD FAP_SERVICE` 从 698 条绑定收敛到 32 条。
  - `LST/MOD PLMN_LIST` 保留 4 条绑定，其中 `Enable`、`IsPrimary` 按产品口径保留。
  - 同步刷新相关命令的 `target_paths` 与 `tree_node_refs`。
- 本地库新增 `基站配置参数管理`：
  - `LST/MOD/ADD/RMV NR_AMF_POOL_CONFIG_PARAM` 覆盖 AMF 地址。
  - `LST/MOD/ADD/RMV NR_XN_IP_ADDR_MAP_INFO` 覆盖 Xn 链接信息。
  - `LST/MOD NR_RAN_COMMON` 覆盖 gNB 参数。
  - 新增/更新标准参数：AMF 地址 8 条、Xn 链接信息 6 条、gNB 参数 4 条。

执行与验证：

- 本地库已直接执行 Up 段 SQL。
- `FAP_SERVICE`、`PLMN_LIST`、`PDCP_INIT_PARAM` 相关 `LST/MOD/ADD/RMV` 命令的 `target_paths = tree_node_refs` 均为 `true`。
- NR `基站配置参数管理` 下 10 条命令的 `target_paths = tree_node_refs` 均为 `true`。
- TD-LTE `SF` 规范字段缺失数为 0。
- catalog 多出的路径包含 10 个已知 `RRCTimers` 合并项，以及 2 个产品口径保留字段 `Enable`、`IsPrimary`。
- `FAP_SERVICE` 合计删除 1333 条白名单外旧扩展/私有/跨分组 TR path 绑定，明细见 `cell-service-deleted-trpaths.csv`。
