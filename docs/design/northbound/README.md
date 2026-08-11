# 北向 / OSS 设计文档索引

本目录收拢 F08 北向相关的当前设计稿和设计附件。阅读顺序建议从本文件开始，再进入综合设计和场景/Profile 目录。

## 文档分层

| 层级 | 文件 | 定位 |
|---|---|---|
| 当前权威设计 | `page-config-redesign-20260731.md` | 页面可配置化主线：能力开关、字段/指标选择、多目标传输、Socket/SNMP/API 配置和数据准确性约束 |
| 数据支持矩阵 | `data-support-matrix-20260731.md` | 逐项比对 Station/OMC Inventory 字段、旧 PM KPI/counter 与当前 xomc 数据源的支持状态 |
| SNMP 测试手册 | `Net-SNMP测试步骤文档.md` | 使用 Net-SNMP 验证 SNMP v2c/v3 MIB 查询、Trap、Inform、鉴权失败和页面配置生效 |
| 历史设计稿 | `overall-redesign-20260730.md` | XML/Profile 管理方案历史参考，不作为当前运行配置主线 |
| 场景 XML 附件 | `scenario-xml/` | `local-s0001.xml` 到 `local-s0017.xml`，仅作为旧场景模板和映射覆盖率参考 |
| Profile 附件 | `scenario-xml/profiles/` | `baicells-legacy-v1.xml`，仅作为旧 profile 方案和字段映射参考 |
| 产品/历史背景 | `../../project/prd/F08-oss-protocol.md` | 早期 F08 SNMP Trap PRD，保留为 SNMP 子能力和立项背景参考，不覆盖当前综合设计 |
| 外部资料来源 | `/Users/renpengfei/Desktop/doc/othernorth/` | 旧北向总述、`northboundApi` 清单和 `enbMonitorExport.md`，仅作为输入资料，不作为 xomc 仓库内权威文档 |

## 怎么区分

1. 看“现在北向整体怎么设计”时，读 `page-config-redesign-20260731.md`。
2. 看“为什么不继续用 XML 管理”时，读 `page-config-redesign-20260731.md` 的 `1`、`3`、`5` 章。
3. 看“设备字段、Inventory 字段、PM KPI/counter 当前到底有没有”时，读 `data-support-matrix-20260731.md`。
4. 看“SNMP v2c/v3 怎么用 Net-SNMP 验证”时，读 `Net-SNMP测试步骤文档.md`。
5. 看“`local-s0001` 到 `local-s0017` 怎么用”时，读 `scenario-xml/README.md`；它们只作为内置模板和映射参考。
6. 看“旧 XML/Profile 方案当时怎么想”时，读 `overall-redesign-20260730.md`。
7. 看“为什么早期要做 SNMP Trap”时，读 `../../project/prd/F08-oss-protocol.md`。
8. 旧桌面资料和 `othernorth` 资料只作为设计输入，不直接进入运行配置，也不代表 xomc 最终 URL、鉴权方式或响应结构。

## 维护约定

- 北向新设计文档优先放在本目录。
- 运行时样例或导入 fixture 放在 `scenario-xml/`。
- 字段列、profile、adapter 契约放在 `scenario-xml/profiles/`。
- 如果后续把 profile 落到运行目录，目标位置是 `omcgo/data/northbound/profiles/`，本目录保留设计稿和迁移说明。
- `docs/README.md` 只保留北向入口，不再逐个列举所有北向附件。
