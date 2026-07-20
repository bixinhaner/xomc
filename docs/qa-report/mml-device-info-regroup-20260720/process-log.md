# MML 设备信息分组清理过程记录

日期：2026-07-20

## 目标口径

用户确认后的最终口径：

- 设备信息分组中应迁移到其他标准分组的字段，也不做迁移。
- 其他分组应该已经存在对应字段，避免重复创建。
- 本次只从 `设备信息参数管理` 分组删除非设备信息字段。
- 保留清单以分析工作簿中 `调整后分组名称 = 设备信息参数管理` 且 `导入动作 = 更新` 的记录为准。

## 数据源分析

分析源：

- `mml配置参数分组-20260717.xlsx`
- `MML配置参数分组-设备信息重分组-20260720.xlsx`

原始 `设备信息参数管理`：

- 848 条绑定。
- 446 个去重标准路径。
- 涉及命令：`LST DEVICE_INFO`、`MOD DEVICE_INFO`、`LST DEVICE_INFO_SW_UPGRADE`。

根据标准南向分组和 path 映射分析后：

- 保留 32 条绑定。
- 删除 816 条绑定。
- 删除项包含原分析建议迁移到 `WAN口配置参数管理`、`小区服务参数管理（总体）`、`EU/RU`、`GPS信息参数管理`、`时间服务器参数管理` 等分组的字段，以及 `跳过` 类多余字段。

## 数据库表定位

关键表：

- `mml_command_groups`：MML 分组。
- `mml_commands`：命令节点；包含 `target_paths`、`tree_node_refs`。
- `mml_command_sub_fields`：命令字段绑定真值。
- `standard_params`：标准 path 字典。

页面最终显示的 path 不只依赖 `mml_command_sub_fields`，还会使用 `mml_commands.tree_node_refs`。因此清理必须同步刷新：

- `mml_command_sub_fields`
- `mml_commands.target_paths`
- `mml_commands.tree_node_refs`

## 迁移脚本

脚本路径：

```text
omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql
```

脚本策略：

- `WITH keep(...)` 写入 32 条保留白名单。
- 删除三个设备信息命令中不在白名单内的 `mml_command_sub_fields`。
- 按当前剩余 `mml_command_sub_fields` 聚合刷新 `target_paths` 和 `tree_node_refs`。
- 不 `INSERT`，不改其他分组，避免重复。

## 实际执行

本地数据库容器：

```text
goomc-local-postgres-1
```

数据库：

```text
omcgo
```

执行时发现 `goose_db_version_seed` 当前版本为 13，因此新增的 `000002` 在既有库上不会被 goose 自动执行。本次按用户“不考虑升级前”的要求，直接执行迁移脚本 Up 段 SQL：

```bash
awk '/^-- \+goose Down/{exit} {print}' migrations/seed/000003_prune_device_info_mml_sub_fields.sql \
  | docker exec -i goomc-local-postgres-1 psql -U omcgo -d omcgo -v ON_ERROR_STOP=1
```

第一次执行结果：

```text
UPDATE 3
```

随后页面仍显示大量 `WAN_CONFIG*`，排查发现 `tree_node_refs` 仍保留旧大列表。执行补充刷新：

```sql
WITH affected AS (
  SELECT id
  FROM public.mml_commands
  WHERE command_code IN ('LST DEVICE_INFO', 'MOD DEVICE_INFO', 'LST DEVICE_INFO_SW_UPGRADE')
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
    updated_at = now()
FROM affected a
LEFT JOIN LATERAL (
  SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
  FROM public.mml_command_sub_fields csf
  JOIN public.standard_params sp ON sp.id = csf.standard_path_id
  WHERE csf.command_id = a.id
    AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = a.id;
```

补充执行结果：

```text
UPDATE 3
```

随后已将迁移脚本同步修正为相同的 `LEFT JOIN LATERAL` 刷新逻辑。

## 应用重启

为清掉后端可能的运行期缓存，重启了 app 容器：

```bash
docker restart goomc-local-app-1
```

## 验证 SQL

剩余绑定数：

```sql
SELECT command_code,
       jsonb_array_length(COALESCE(target_paths,'[]'::jsonb)) AS target_count,
       jsonb_array_length(COALESCE(tree_node_refs,'[]'::jsonb)) AS tree_count,
       target_paths = tree_node_refs AS refs_match
FROM public.mml_commands
WHERE command_code IN ('LST DEVICE_INFO', 'MOD DEVICE_INFO', 'LST DEVICE_INFO_SW_UPGRADE')
ORDER BY command_code;
```

验证结果：

| command_code | target_count | tree_count | refs_match |
|---|---:|---:|---|
| `LST DEVICE_INFO` | 24 | 24 | true |
| `LST DEVICE_INFO_SW_UPGRADE` | 3 | 3 | true |
| `MOD DEVICE_INFO` | 5 | 5 | true |

泄漏检查：

```sql
SELECT COUNT(*) AS leaked_wan_refs
FROM public.mml_commands c
CROSS JOIN LATERAL jsonb_array_elements_text(
  COALESCE(NULLIF(c.tree_node_refs, '[]'::jsonb), c.target_paths, '[]'::jsonb)
) refs(value)
WHERE c.command_code IN ('LST DEVICE_INFO', 'MOD DEVICE_INFO', 'LST DEVICE_INFO_SW_UPGRADE')
  AND refs.value LIKE '%WAN_CONFIG%';
```

验证结果：

```text
leaked_wan_refs = 0
```

## 最终状态

保留明细：

- `LST DEVICE_INFO`：24 条。
- `MOD DEVICE_INFO`：5 条。
- `LST DEVICE_INFO_SW_UPGRADE`：3 条。

删除明细：

- `LST DEVICE_INFO`：419 条。
- `MOD DEVICE_INFO`：397 条。

完整明细见：

- [delete-and-keep-detail.md](delete-and-keep-detail.md)
- [delete-detail.csv](delete-detail.csv)
- [keep-detail.csv](keep-detail.csv)

## 2026-07-20 软件版本参数分组补充

按以下规范复核 `SB / 软件版本参数管理`：

- `中国移动5G扩展型皮基站网管南向接口数据模型规范v1.9.4.xlsx`
- `中国移动TD-LTE皮站_飞站基站设备网络管理南向接口数据配置模型规范V2.3.xlsx`

发现 5G v1.9.4 中 `Device.SoftwareCtrl.PatchInfo` 未在当前库中维护。TD-LTE V2.3 不包含该项，但 5G 规范明确包含该只读字段。

合并到既有脚本：

```text
omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql
```

执行：

```bash
awk '/^-- \+goose Down/{exit} {print}' migrations/seed/000003_prune_device_info_mml_sub_fields.sql \
  | docker exec -i goomc-local-postgres-1 psql -U omcgo -d omcgo -v ON_ERROR_STOP=1
```

处理结果：

- `standard_params` 新增/更新 `Device.SoftwareCtrl.PatchInfo`。
- `LST SOFTWARE_CTRL` 新增 `PATCH_INFO` 绑定，`sort_order = 6`。
- 删除 `Device.SoftwareCtrl.AccCard1PpsDelay`、`Device.SoftwareCtrl.N48N78SharedRfEnable` 两个规范外扩展绑定。
- 刷新 `LST SOFTWARE_CTRL`、`MOD SOFTWARE_CTRL` 的 `target_paths` 与 `tree_node_refs`。

验证结果：

| command_code | target_count | tree_count | refs_match |
|---|---:|---:|---|
| `LST SOFTWARE_CTRL` | 6 | 6 | true |
| `MOD SOFTWARE_CTRL` | 3 | 3 | true |

泄漏检查：

```text
Device.SoftwareCtrl.AccCard1PpsDelay / Device.SoftwareCtrl.N48N78SharedRfEnable stale_refs = 0
```

## 2026-07-20 基站网管参数分组补充

按以下规范复核 `SC / 基站网管参数管理`：

- `中国移动5G扩展型皮基站网管南向接口数据模型规范v1.9.4.xlsx`
- `中国移动TD-LTE皮站_飞站基站设备网络管理南向接口数据配置模型规范V2.3.xlsx`

两份规范共同包含 19 个 `Device.ManagementServer.*` 字段。当前库中 `LST MANAGEMENT_SERVER`、`MOD MANAGEMENT_SERVER` 额外包含 5 个扩展字段。

按产品口径需保留 3 个扩展字段：

- `Device.ManagementServer.X_COM_tr069_port`
- `Device.ManagementServer.sslStatus.endDate`
- `Device.ManagementServer.sslStatus.startDate`

需删除 2 个规范外扩展字段：

- `Device.ManagementServer.tfcsManagerPrimsrc`
- `Device.ManagementServer.tfcsSyncState`

合并到既有脚本：

```text
omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql
```

执行：

```bash
awk '/^-- \+goose Down/{exit} {print}' migrations/seed/000003_prune_device_info_mml_sub_fields.sql \
  | docker exec -i goomc-local-postgres-1 psql -U omcgo -d omcgo -v ON_ERROR_STOP=1
```

处理结果：

- 保留 `LST MANAGEMENT_SERVER`、`MOD MANAGEMENT_SERVER` 中 3 个产品扩展绑定。
- 删除 `LST MANAGEMENT_SERVER`、`MOD MANAGEMENT_SERVER` 中 2 个 `tfcs*` 规范外扩展绑定。
- 刷新 `LST MANAGEMENT_SERVER`、`MOD MANAGEMENT_SERVER` 的 `target_paths` 与 `tree_node_refs`。
- 初次验证发现 `tree_node_refs` 仍保留旧 24/20 条；随后在脚本中增加独立 final sync 并重新执行。

验证结果：

| command_code | target_count | tree_count | refs_match |
|---|---:|---:|---|
| `LST MANAGEMENT_SERVER` | 22 | 22 | true |
| `MOD MANAGEMENT_SERVER` | 18 | 18 | true |

泄漏检查：

```text
2 个 tfcs* ManagementServer 扩展字段 sub_fields stale_refs = 0
2 个 tfcs* ManagementServer 扩展字段 tree_node_refs stale_refs = 0
```
