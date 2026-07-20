# 日志参数分组缺失项补充记录

日期：2026-07-20

## 复核依据

输入规范：

- `/Users/wangyong/OBJECT/Codex/中国移动5G扩展型皮基站网管南向接口数据模型规范v1.9.4.xlsx`
- `/Users/wangyong/OBJECT/Codex/中国移动TD-LTE皮站_飞站基站设备网络管理南向接口数据配置模型规范V2.3.xlsx`

复核范围：

- `SE` 页，目录名称为 `日志参数管理`。

## 规范差异

两份规范共同包含：

- `Device.LogMgmt.PeriodicUploadEnable`
- `Device.LogMgmt.URL`
- `Device.LogMgmt.Username`
- `Device.LogMgmt.Password`
- `Device.LogMgmt.PeriodicUploadInterval`

5G v1.9.4 额外包含：

- `Device.LogMgmt.LogLevel`

当前库未发现规范外冗余 `Device.LogMgmt.*` 绑定，但 `LST LOG_MGMT`、`MOD LOG_MGMT` 的 `tree_node_refs` 为空，需要同步刷新。

## 本次处理

合并到 seed 脚本：

```text
omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql
```

处理内容：

- 确保 `standard_params` 存在 `Device.LogMgmt.LogLevel`。
- 绑定到 `LST LOG_MGMT`、`MOD LOG_MGMT`，MML Code 为 `LOG_LEVEL`，中文名 `日志重要等级`，英文名 `LogLevel`。
- 设置 `sort_order = 6`，位于 5 个基础日志上传配置字段之后。
- 写入所有已具备 5 个基础日志参数的产品参数模型 `param_mappings`，`source = custom`，使控制台按产品模型过滤时可显示该字段，且不被 XML 参数模型重载删除。
- 同步刷新 `LST LOG_MGMT`、`MOD LOG_MGMT` 的 `target_paths` 与 `tree_node_refs`。

## 执行与验证

本地库已直接执行 Up 段 SQL。

验证结果：

| 命令 | target_paths | tree_node_refs | 一致 |
|---|---:|---:|---|
| `LST LOG_MGMT` | 6 | 6 | true |
| `MOD LOG_MGMT` | 6 | 6 | true |

`LST/MOD LOG_MGMT` 当前绑定：

| sort_order | MML Code | 标准路径 |
|---:|---|---|
| 1 | `PERIODIC_UPLOAD_ENABLE` | `Device.LogMgmt.PeriodicUploadEnable` |
| 2 | `URL` | `Device.LogMgmt.URL` |
| 3 | `USERNAME` | `Device.LogMgmt.Username` |
| 4 | `PASSWORD` | `Device.LogMgmt.Password` |
| 5 | `PERIODIC_UPLOAD_INTERVAL` | `Device.LogMgmt.PeriodicUploadInterval` |
| 6 | `LOG_LEVEL` | `Device.LogMgmt.LogLevel` |

## 产品模型过滤说明

MML 控制台会按当前设备/产品参数模型过滤命令字段，不是直接展示命令绑定全集。

本次复核的本地设备示例：

- `E8F2971A3DC921A03D3E4FD4A0C1`：`FAP/PGSM`，产品 `BSC 产品 / gsm`，参数模型 `BSC`。
- 该模型已通过 `source = custom` 补充 `Device.LogMgmt.LogLevel`，按当前设备过滤口径显示 6 项。
- 所有已具备 5 个基础 `Device.LogMgmt.*` 字段的产品参数模型，同步补充 `Device.LogMgmt.LogLevel`。
