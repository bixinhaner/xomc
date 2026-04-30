# T-0094 Verify — MML category_group DB 列缺失（误诊核实）

**关联 Task**：backlog §4 Triaged T-0094  
**类型**：bug 三层核查 / pre-existing  
**Owner**：Claude（静态分析） + 用户/PM（运行时一行 SQL 闭环）  
**报告日期**：2026-04-30  
**结论**：**静态分析倾向于"列存在"**；backlog 描述与迁移事实不符；runtime 真库验证待 docker 起后用户/PM 一行 SQL 终结

---

## 1. backlog 描述 vs migrations 真相

backlog §4 Triaged 原文（T-0094 Notes 字段）：

> 现状：Go struct `MMLCustomCommand.CategoryGroup` (`model.go:188`) + INSERT (`pg_repository.go:1253`) + FE 表单 (`AddTemplateModal:246`) 三处引用，但 `mml_custom_command` 表（migrations/000014 + 000026）**从未创建此列**

migrations 时间序逐行核查：

| 文件 | 关键操作 | category_group 列状态 |
|------|---------|----------------------|
| `migrations/000014_mml_templates_and_audit.sql` | `CREATE TABLE mml_templates` | ❌ 无（DDL 不含此列） |
| `migrations/000020_mml_category_group_and_executor.sql` Up L3 | `ALTER TABLE mml_templates ADD COLUMN IF NOT EXISTS category_group VARCHAR(50)` | ✅ **首次添加** |
| `migrations/000020` Up L4 | `CREATE INDEX IF NOT EXISTS idx_mml_templates_scope_group ON mml_templates(template_scope, category_group)` | ✅ index 间接证据列存在 |
| `migrations/000020` Down L12 | `ALTER TABLE mml_templates DROP COLUMN IF EXISTS category_group` | （回滚专用） |
| `migrations/000026_mml_table_restructure.sql` Up L28 | `ALTER TABLE mml_templates RENAME TO mml_custom_command` | ✅ 列随表迁移（PostgreSQL `RENAME TO` 仅改表名，**不删列、不动列名**） |
| `migrations/000026` Up L31-32 | `RENAME COLUMN template_name → command_name` / `template_scope → command_scope` | （未触 category_group → 列名保持） |
| `migrations/000026` Up L43-46 | 4 处 `ALTER INDEX RENAME`（覆盖 command_code / scope_creator / product_types_gin / parameters_gin） | ⚠️ **遗漏** `idx_mml_templates_scope_group` 未重命名（但功能不受影响 — 索引仍存在仅前缀过期） |

**事实**：000020 加列 + 000026 RENAME 表（不删列、不重命名 category_group）→ 现表 `mml_custom_command` **应有** `category_group VARCHAR(50)` 列，可空。

**backlog 描述"000014 + 000026 从未创建此列"是事实性错误** — 漏掉了 000020 的 ALTER ADD。

---

## 2. INSERT 写不存在列的 PG 行为（理论核查）

`omcgo/internal/mml/pg_repository.go:1282-1288`（squirrel 构建 INSERT）：

```go
Columns("command_name", "command_code", "operation_type",
        "command_scope", "category_group", ...)
Values(cmd.CommandName, cmd.CommandCode, cmd.OperationType,
       cmd.CommandScope, cmd.CategoryGroup, ...)
```

**链路**：squirrel `Insert().Columns().Values()` → 生成参数化 SQL → pgx/v5 `Exec`/`Query` 发往 PG。

**若 `category_group` 列不存在** → PostgreSQL 返 `42703 column "category_group" of relation "mml_custom_command" does not exist`（standard SQLSTATE） → pgx 包装为 `*pgconn.PgError` → service/handler 返 error → HTTP 500 / 业务 error code。

**结论**：**不会 silent skip**。若现网 INSERT 真触发此错，user-facing 表现一定是"添加 MML 自定义命令失败"且日志可见 SQLSTATE 42703。

---

## 3. 数据流三处引用确认

| 层 | 文件 | 行 | 关键引用 |
|----|------|----|---------|
| Go struct | `omcgo/internal/mml/model.go` | 188 | `CategoryGroup string \`json:"category_group,omitempty"\`` （仅 JSON tag，无 db tag — 由 squirrel `Columns()` 显式指定列名映射） |
| Repo INSERT | `omcgo/internal/mml/pg_repository.go` | 1282-1288 | squirrel Insert `category_group` 列 |
| FE 表单字段 | `omcmb/webcode/src/pages/mml/Console/components/AddTemplateModal.tsx` | 112 | `categoryGroup: values.categoryGroup ?? ''` |
| FE i18n | `omcmb/webcode/src/i18n/zh-CN.ts` | （key 存在） | `'mml.console.categoryGroup': '所属分类'` → user-visible（删字段会破坏 UX） |

---

## 4. 单元测试覆盖盲区

`omcgo/internal/mml/handler_test.go` + `service_test.go`：仅 mock repository，**未跑真实 DB INSERT**。集成测试也无此用例。**这是为什么静态分析会有疑问点未被早暴露**：若 mock-only，列存在性永远不被代码路径检查。

加一条真库集成测试是 long-term 修补建议（不在本任务 scope）。

---

## 5. Runtime 真库验证（**待用户/PM 操作**）

docker 当前未起，此 verify 报告无法独自终结。一行 SQL 闭环：

```bash
cd ~/code/baicells/goomc
docker compose -f deployments/docker/docker-compose.yml up -d postgres
docker compose -f deployments/docker/docker-compose.yml exec postgres \
  psql -U omcuser -d omcdb -c '\d mml_custom_command' | grep -i category_group
```

**预期输出**（列存在）：

```
 category_group  | character varying(50) |           |          |
```

**Alternative — 一句 SQL 直接判**：

```sql
SELECT column_name, data_type, character_maximum_length
FROM information_schema.columns
WHERE table_name = 'mml_custom_command' AND column_name = 'category_group';
```

返 1 行 = 列存在；返 0 行 = 列缺失（极低概率）。

---

## 6. 决策路径

| 真库结果 | 判决 | 行动 | 工作量 |
|---------|------|------|--------|
| 列存在 ✅（最大概率） | T-0094 = misdiagnosis | 移到 §8 Rejected，Reason="backlog 描述漏读 000020 ALTER ADD；列由 000020 创建并随 000026 RENAME 保留"；解锁 T-0090（deps T-0094 ✅） | 0（仅 backlog 编辑） |
| 列不存在 ❌（极低概率） | T-0094 = real bug | **不**新增 migration（000020 已有该 DDL） — 排查 `goose_db_version` 表确认 000020 未应用；可能成因：本地 schema 漂移 / migrate-up 未跑全 / 历史回滚未恢复 | 排查 ≤ 1h；不写代码 |

---

## 7. 关联工作

- **解锁 T-0090**（MML 控制台公/私命令新增页面 UX 整改）的 deps T-0094 ✅
- **索引名遗漏**：`idx_mml_templates_scope_group` 在 000026 索引重命名循环中漏掉（functional impact 0，cosmetic only）— 登记为 P3 housekeeping，可在 T-0090 一并清理

---

## 8. 静态分析 self-check

- [x] 读全 `migrations/000014_*.sql`（确认无 category_group 创建）
- [x] 读全 `migrations/000020_*.sql`（Up + Down 双向验证）
- [x] 读全 `migrations/000026_*.sql`（Up + Down 双向验证）
- [x] 三处代码引用 file:line 全部 spot-check（model / pg_repository / FE）
- [x] PostgreSQL `RENAME TO` 语义文档对照（不删列、保留所有约束/索引）
- [x] PostgreSQL `42703` SQLSTATE 行为对照（确认硬错而非 silent skip）
- [x] squirrel 参数化路径不绕列名（即列名错则 SQL 错）
- [x] FE 字段 user-visible（删字段会破坏 UX，需先决产品意图）
- [x] 单元测试无真库 INSERT 覆盖（盲区已识别）

---

**报告 by**：Claude /dev-pipeline ULTRATHINK Step 1  
**关联报告**：`docs/project/backlog.md` §10 changelog 2026-04-30 "静态核实 T-0094"
