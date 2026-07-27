# NR 小区服务参数二级分组重建

日期：2026-07-21

## 结论

- 5G 南向目录 `SJ / Services.FAPService.{i}.CellConfig.{i}.NR` 对应 `小区服务参数管理（总体）`。
- 当前页面原有 `SF` 命令主要来自 TD-LTE 小区服务；本次按 5G `SJ` 表“归类”列补齐 NR 二级分组。
- NR 命令挂到统一分组 `chapter:SF / 小区服务参数管理（总体）` 下，命令码使用 `SF_NR_SJ_SUB_xx`，避免覆盖 LTE 既有命令。

## 二级分组

| 命令码后缀 | 南向归类 | 参数数 | RW 参数数 |
|---|---|---:|---:|
| `SF_NR_SJ_SUB_01` | `CN参数管理` | 1 | 1 |
| `SF_NR_SJ_SUB_02` | `小区闭塞` | 1 | 1 |
| `SF_NR_SJ_SUB_03` | `小区激活/去激活` | 1 | 1 |
| `SF_NR_SJ_SUB_04` | `小区状态参数` | 1 | 0 |
| `SF_NR_SJ_SUB_05` | `广播参数管理` | 2 | 1 |
| `SF_NR_SJ_SUB_06` | `小区参数管理` | 15 | 11 |
| `SF_NR_SJ_SUB_07` | `小区标识参数` | 20 | 19 |
| `SF_NR_SJ_SUB_08` | `VoNR参数` | 4 | 3 |
| `SF_NR_SJ_SUB_09` | `接入UE数量管理` | 3 | 1 |
| `SF_NR_SJ_SUB_10` | `小区底噪测量` | 2 | 1 |
| `SF_NR_SJ_SUB_11` | `小区与RU映射` | 2 | 2 |
| `SF_NR_SJ_SUB_12` | `小区最大发射功率` | 1 | 1 |
| `SF_NR_SJ_SUB_13` | `节能管理参数` | 32 | 31 |
| `SF_NR_SJ_SUB_14` | `安全参数管理` | 8 | 8 |
| `SF_NR_SJ_SUB_15` | `DRX参数管理` | 6 | 6 |
| `SF_NR_SJ_SUB_16` | `寻呼参数管理` | 4 | 4 |

## 落地文件

- `omcgo/migrations/seed/000004_mml_device_info_regroup_20260721.sql`
