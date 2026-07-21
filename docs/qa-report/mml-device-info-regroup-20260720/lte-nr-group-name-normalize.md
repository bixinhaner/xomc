# LTE/NR MML 分组命名规范化

日期：2026-07-21

## 依据

- `/Users/wangyong/OBJECT/Codex/中国移动5G扩展型皮基站网管南向接口数据模型规范v1.9.4.xlsx`
- `/Users/wangyong/OBJECT/Codex/中国移动TD-LTE皮站_飞站基站设备网络管理南向接口数据配置模型规范V2.3.xlsx`

处理口径：

- LTE 与 NR 都有的业务域，MML 使用统一分组名称。
- 仅 LTE 或仅 NR 有的业务域，MML 使用对应南向目录分组名称。
- 本次只规范化分组/章节展示名，不调整命令字段绑定；字段绑定仍以 `mml_command_sub_fields` 为真值源。
- 截图红框内 `SA` 到 `SF_NR` 已完成，本次落地只处理红框下方剩余分组。

## 统一分组名称

| 统一名称 | LTE 目录 | NR 目录 |
|---|---|---|
| `设备信息参数管理` | `SA` | `SA` |
| `软件版本参数管理` | `SB` | `SB` |
| `基站网管参数管理` | `SC` | `SC` |
| `告警参数管理` | `SD` | `SD` |
| `日志参数管理` | `SE` | `SE` |
| `小区服务参数管理（总体）` | `SF` | `SJ` |
| `SCTP参数管理` | `SG` | `SN` |
| `RAN协议栈参数` | `SH` | `SK` |
| `邻区参数管理` | `SI` | `SL` |
| `移动性参数管理` | `SJ` | `SM` |
| `SON参数管理` | `SK` | `SG` |
| `WAN口配置参数管理` | `SL` | `SO` |
| `时间服务器参数管理` | `SN` | `SP` |
| `GPS信息参数管理` | `SO` | `SQ` |
| `MR参数管理` | `SP` | `SR` |
| `性能参数管理` | `SQ` | `SS` |

## 独占分组名称

| 制式 | 南向目录 | 分组名称 |
|---|---|---|
| NR | `SF` | `基站配置参数管理` |
| NR | `SH` | `本地分流规则` |
| NR | `SI` | `基站能力参数管理` |
| NR | `ST` | `软采规则` |
| LTE | `SM` | `IPsec参数管理` |
| LTE | `SR` | `扩展型一体化皮基站参数` |

## 落地文件

- `omcgo/internal/mml/flat_group_tree.go`
- `omcgo/migrations/seed/000004_mml_device_info_regroup_20260721.sql`
