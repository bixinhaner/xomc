# 软件版本参数分组缺失项补充记录

日期：2026-07-20

## 复核依据

输入规范：

- `/Users/wangyong/OBJECT/Codex/中国移动5G扩展型皮基站网管南向接口数据模型规范v1.9.4.xlsx`
- `/Users/wangyong/OBJECT/Codex/中国移动TD-LTE皮站_飞站基站设备网络管理南向接口数据配置模型规范V2.3.xlsx`

复核范围：

- `SB` 页，目录名称为 `软件版本参数管理`。

## 规范差异

两份规范共同包含：

- `Device.SoftwareCtrl.AutoActivateEnable`
- `Device.SoftwareCtrl.ActivateTime`
- `Device.SoftwareCtrl.ActivateEnable`
- `Device.SoftwareCtrl.SystemCurrentVersion`
- `Device.SoftwareCtrl.SystemBackupVersion`

5G v1.9.4 额外包含：

- `Device.SoftwareCtrl.PatchInfo`

当前库已有但规范未覆盖、需删除的扩展项：

- `Device.SoftwareCtrl.AccCard1PpsDelay`
- `Device.SoftwareCtrl.N48N78SharedRfEnable`

## 本次处理

合并到 seed 脚本：

```text
omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql
```

处理内容：

- 确保 `standard_params` 存在 `Device.SoftwareCtrl.PatchInfo`。
- 绑定到 `LST SOFTWARE_CTRL`，MML Code 为 `PATCH_INFO`，中文名 `补丁信息`，英文名 `PatchInfo`。
- 设置 `sort_order = 6`，位于 5 个基础软件版本控制字段之后。
- 删除 `Device.SoftwareCtrl.AccCard1PpsDelay`、`Device.SoftwareCtrl.N48N78SharedRfEnable` 两个规范外扩展绑定。
- 同步刷新 `LST SOFTWARE_CTRL`、`MOD SOFTWARE_CTRL` 的 `target_paths` 与 `tree_node_refs`。

## 执行与验证

本地库已直接执行 Up 段 SQL。

验证结果：

| 命令 | target_paths | tree_node_refs | 一致 |
|---|---:|---:|---|
| `LST SOFTWARE_CTRL` | 6 | 6 | true |
| `MOD SOFTWARE_CTRL` | 3 | 3 | true |

`LST SOFTWARE_CTRL` 当前绑定：

| sort_order | MML Code | 标准路径 |
|---:|---|---|
| 1 | `AUTO_ACTIVATE_ENABLE` | `Device.SoftwareCtrl.AutoActivateEnable` |
| 2 | `ACTIVATE_TIME` | `Device.SoftwareCtrl.ActivateTime` |
| 3 | `ACTIVATE_ENABLE` | `Device.SoftwareCtrl.ActivateEnable` |
| 4 | `SYSTEM_CURRENT_VERSION` | `Device.SoftwareCtrl.SystemCurrentVersion` |
| 5 | `SYSTEM_BACKUP_VERSION` | `Device.SoftwareCtrl.SystemBackupVersion` |
| 6 | `PATCH_INFO` | `Device.SoftwareCtrl.PatchInfo` |

`MOD SOFTWARE_CTRL` 本次不新增 `PatchInfo`，因为规范访问权限为只读。

删除后泄漏检查：

- `tree_node_refs` 中 `Device.SoftwareCtrl.AccCard1PpsDelay`、`Device.SoftwareCtrl.N48N78SharedRfEnable` 残留数为 0。
