# 剩余 MML 分组参数差异处理清单
日期：2026-07-21
## 口径
- 范围：截图红框下方剩余分组，含 `SG` 到 `SR` 以及预留 NR 独占分组 `SH_NR`、`SI_NR`、`ST_NR`。
- 期望参数来源：TD-LTE V2.3 与 5G NR v1.9.4 南向数据模型对应章节。
- `新增` 表示南向模型有、当前 MML 分组未绑定；`删除` 表示当前 MML 分组有、对应南向分组没有。
- 已通过 `omcgo/migrations/seed/000004_mml_device_info_regroup_20260721.sql` 删除 206 条规范外当前绑定，并刷新受影响命令的 `target_paths` / `tree_node_refs`。
- 当前 `需删除` 已清零；剩余 `需新增` 项涉及命令归属/命令码派生，先输出清单，不直接塞入既有命令。

## 汇总

| 分组 | 南向来源 | 期望参数 | 当前参数 | 需新增 | 需删除 | 已覆盖 |
|---|---|---:|---:|---:|---:|---:|
| `SG SCTP参数管理` | LTE:SG; NR:SN | 25 | 13 | 12 | 0 | 13 |
| `SH RAN协议栈参数` | LTE:SH; NR:SK | 215 | 63 | 152 | 0 | 63 |
| `SI 邻区参数管理` | LTE:SI; NR:SL | 68 | 27 | 41 | 0 | 27 |
| `SJ 移动性参数管理` | LTE:SJ; NR:SM | 690 | 142 | 548 | 0 | 142 |
| `SK SON参数管理` | LTE:SK; NR:SG | 62 | 28 | 34 | 0 | 28 |
| `SL WAN口配置参数管理` | LTE:SL; NR:SO | 37 | 36 | 1 | 0 | 36 |
| `SM IPsec参数管理` | LTE:SM | 9 | 9 | 0 | 0 | 9 |
| `SN 时间服务器参数管理` | LTE:SN; NR:SP | 9 | 9 | 0 | 0 | 9 |
| `SO GPS信息参数管理` | LTE:SO; NR:SQ | 3 | 3 | 0 | 0 | 3 |
| `SP MR参数管理` | LTE:SP; NR:SR | 16 | 14 | 2 | 0 | 14 |
| `SQ 性能参数管理` | LTE:SQ; NR:SS | 10 | 10 | 0 | 0 | 10 |
| `SR 扩展型一体化皮基站参数` | LTE:SR | 83 | 29 | 54 | 0 | 29 |
| `SH_NR 本地分流规则` | NR:SH | 36 | 0 | 36 | 0 | 0 |
| `SI_NR 基站能力参数管理` | NR:SI | 1 | 0 | 1 | 0 | 0 |
| `ST_NR 软采规则` | NR:ST | 148 | 0 | 148 | 0 | 0 |

## 明细文件

- [remaining-groups-param-add.csv](remaining-groups-param-add.csv)：需新增参数明细，共 1029 条。
- [remaining-groups-param-delete.csv](remaining-groups-param-delete.csv)：已删除绑定明细，共 206 条；状态列标记为 `deleted_by_000005`。
- [remaining-groups-param-keep.csv](remaining-groups-param-keep.csv)：已覆盖参数明细。
- [remaining-groups-param-summary.csv](remaining-groups-param-summary.csv)：汇总表。
