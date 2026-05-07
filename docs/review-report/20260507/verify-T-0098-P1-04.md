# S4 Verify Report — T-0098-P1-04

| 字段 | 值 |
|------|-----|
| Task | T-0098-P1-04（迁移 `000059_alarm_dictionary.sql` — alarm_severity_levels + 4 行种子 + alarm_definitions + alarms_active.is_unknown） |
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/methodology/AI承诺对峙清单.md` W3 / `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 1 |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §3.2.1 / §3.2.2 / §3.3 |
| 模式 | wave-batched（Skip S0/S1，per dev-pipeline §C.1） |
| Deps | — |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/migrations/000059_alarm_dictionary.sql` | 新增 | ~85 |

## 出口门核销

| 门 | 命令 | 结果 |
|----|------|------|
| 编号连续性 | `bash omcgo/scripts/check-migrations.sh` | ✅ 59 文件连续 |
| go build | `go build ./...` | ✅ 无输出 |
| migrate up | up 15.71ms → version=59 | ✅ |
| 种子数据可查 | `SELECT code,name,display_order FROM alarm_severity_levels` | ✅ 4 行（31001 Critical / 31002 Major / 31003 Minor / 31004 Warning） |
| 跨域 FK | alarm_definitions.severity_id REFERENCES alarm_severity_levels(id) ON DELETE RESTRICT | ✅ |
| ALTER ADD COLUMN | alarms_active.is_unknown BOOLEAN NOT NULL DEFAULT FALSE | ✅ data_type=boolean is_nullable=NO column_default=false |
| migrate down | down 6.82ms；2 表 NULL / is_unknown 列 count=0 | ✅ |
| migrate up（再次） | 14.73ms 成功 | ✅ |
| 新端点 E/R | — | N/A |
| metric/log grep | — | N/A |

## schema 摘要

| 对象 | 关键列 | 索引 / FK |
|------|-------|----------|
| `alarm_severity_levels` | id PK / code UNIQUE / name UNIQUE / display_order | 4 行种子（31001/31002/31003/31004） |
| `alarm_definitions` | identifier UNIQUE / ne_type / severity_id FK RESTRICT / cn/en × {name,probable_cause,suggestion} / event_type / is_show | (ne_type) / (severity_id, is_show) 两个查询索引 |
| `alarms_active.is_unknown` | BOOLEAN NOT NULL DEFAULT FALSE + 部分索引 (is_unknown) WHERE is_unknown | fallback 路径标记，治理闭环过滤依据 |

## 设计契约要点

1. **D2=B 决策落地分两步**：本迁移**只新建**新表 + 新增 is_unknown 列；旧 `alarm_libraries` / `alarm_library_i18n` 留待 P5-06 DROP。原因：P2-10 fallback 接收路径未上线时若过早删旧表，receiver 立即崩溃。XML 真相源稳定，重载即恢复。
2. **alarm_severity_levels 4 行种子用 INSERT ... ON CONFLICT DO NOTHING**：CLAUDE.md §5.5.7 幂等要求；运营商规范固定码 31001-31004 跨系统稳定，迁移即写、Loader 不重写。
3. **severity_id ON DELETE RESTRICT**：与设计 §3.2.2 对齐；不允许误删严重级致告警定义悬挂；treats severity_levels as immutable reference data.
4. **identifier UNIQUE 跨 ne_type**：设计 §3.2.2 + §3.4 加载流程"校验 identifier 跨文件全局唯一"；运行时 `Registry.GetByIdentifier(id)` 一次命中。
5. **alarms_active 非分区表**：迁移 000006 验证（无 PARTITION BY）；ADD COLUMN IF NOT EXISTS 安全。
6. **is_unknown 部分索引 WHERE is_unknown**：默认 false 占绝大多数，部分索引体积小，治理查询走该索引高效命中"未识别告警"子集。

## 不在本任务交付范围（后续接力）

- `alarm_definitions` 业务数据写入 → P1-06 Loader 从 7 个 ne_type XML 加载（ENB.xml 217 / GNB.xml 108 / OMC.xml 28 / EPC.xml 52 / EGW.xml 16 / CPE.xml 2 / UPS.xml 19 = 442 行）
- 接收路径 fallback 分支 → P2-10
- handler / Registry / unknown-stats 治理端点 → Phase 2/3
- 旧 `alarm_libraries` DROP → Phase 5 P5-06（D2=B 决议）

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | 告警 i18n 在新设计中按列折叠（cn_/en_ 前缀）；旧 `alarm_library_i18n` 保留至 P5-06 | 兼容期老接口仍可读旧表 |
| L2 | event_type INT 字段未加 CHECK，因事件类型扩展性强（设计示例 30003）| Loader 端校验 |
| L3 | UPS / CPE 仅 2-19 行规模，单表存储更经济（设计 §3.2.2 决策已论证） | — |

## 结论

**S4 出口门通过**。S5 wave-batched 模式以本 verify-md 为审查证据，可直接进入 S6 commit。
