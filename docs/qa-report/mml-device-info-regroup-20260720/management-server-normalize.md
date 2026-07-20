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
