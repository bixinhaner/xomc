# 剩余 MML 分组按南向小分组重建

日期：2026-07-21

## 口径

- 有南向小分组/归类时，按小分组生成 MML 命令。
- 无小分组时，按最外层目录分组生成 MML 命令。
- LTE 与 NR 同名小分组合并到同一个 MML 小分组命令下。
- `LST` 绑定该小分组全部参数，`MOD` 绑定其中 `RW` 参数。
- 邻区、WAN 口配置对象类小分组补充 `ADD/RMV` 命令，用于新增/删除对象实例。
- 旧的剩余分组标准命令标记为 deprecated，避免页面继续显示旧拆分。
- 树形命令接口过滤 deprecated 命令；例如 SCTP 旧的 `SCTP 关联状态`、`SCTP 协议配置` 不再显示，仅保留统一的 `SCTP参数管理`。

## 汇总

| 分组 | 小分组数 | LST/MOD 命令 | 参数数 | RW 参数数 |
|---|---:|---:|---:|---:|
| `SG SCTP参数管理` | 1 | 2 | 25 | 20 |
| `SH RAN协议栈参数` | 6 | 12 | 215 | 211 |
| `SI 邻区参数管理` | 5 | 10 | 68 | 68 |
| `SJ 移动性参数管理` | 15 | 30 | 690 | 650 |
| `SK SON参数管理` | 3 | 6 | 62 | 57 |
| `SL WAN口配置参数管理` | 3 | 6 | 37 | 32 |
| `SM IPsec参数管理` | 1 | 2 | 9 | 2 |
| `SN 时间服务器参数管理` | 1 | 2 | 9 | 8 |
| `SO GPS信息参数管理` | 1 | 2 | 3 | 0 |
| `SP MR参数管理` | 1 | 2 | 16 | 16 |
| `SQ 性能参数管理` | 2 | 4 | 10 | 10 |
| `SR 扩展型一体化皮基站参数` | 3 | 6 | 83 | 11 |
| `SH_NR 本地分流规则` | 7 | 14 | 36 | 31 |
| `SI_NR 基站能力参数管理` | 1 | 2 | 1 | 1 |
| `ST_NR 软采规则` | 9 | 18 | 148 | 147 |

## 明细文件

- [remaining-groups-subgroup-commands.csv](remaining-groups-subgroup-commands.csv)：每个南向小分组对应的 LST/MOD 命令与参数数量。
- [remaining-groups-neighbor-object-commands.csv](remaining-groups-neighbor-object-commands.csv)：邻区小分组补充的 ADD/RMV 对象命令。
- [remaining-groups-wan-object-commands.csv](remaining-groups-wan-object-commands.csv)：WAN 口配置小分组补充的 ADD/RMV 对象命令。
- [remaining-groups-param-add.csv](remaining-groups-param-add.csv)：上一轮差异中的新增候选保留归档。
- [remaining-groups-param-delete.csv](remaining-groups-param-delete.csv)：上一轮删除绑定明细保留归档。

## 落地文件

- `omcgo/migrations/seed/000004_mml_device_info_regroup_20260721.sql`
