# Verify Report — T-0090-b (MML 后端 drop product_types 列)

**Task**: T-0090-b — drop `mml_custom_command.product_types` column + GIN index + 扫净 Go/FE 引用
**Type**: ref / F06/mml+migration+frontend / P2 / M (2-3d)
**Sprint**: sprint-11 (pull-forward 进 sprint-10 buffer)
**Deps**: — (T-0090-a 已 done 是协调前置 — FE UI 已 ready 接 empty default)
**Owner**: Claude
**Date**: 2026-05-12

---

## §1. 改动面

| 文件 | 操作 | 改动量 |
|------|------|-------|
| `omcgo/migrations/000087_drop_mml_custom_command_product_types.sql` | **新建** | +18 行（Up: DROP INDEX + DROP COLUMN；Down: ADD COLUMN BACK + CREATE INDEX BACK，仅 schema 恢复）|
| `omcgo/internal/mml/model.go` | 删 | -1 行（MMLCustomCommand.ProductTypes 字段）|
| `omcgo/internal/mml/service.go` | 删 | -8 行（3 段：Create nil-init / Update 复制 / Clone 复制）|
| `omcgo/internal/mml/handler.go` | 删 | -4 行（CreateRequest + UpdateRequest DTOs + 2 struct literal 引用）|
| `omcgo/internal/mml/pg_repository.go` | 删 | -16 行（customCommandColumns 列表 / Create Marshal+Columns+Values / Update Marshal+Set / scanCustomCommand+scanCustomCommandRow 各 2 段 Unmarshal）|
| `omcmb/frontend-core/src/types/mml.ts` | 删 | -1 行（MMLCustomCommand.productTypes 类型字段）|
| `omcmb/frontend-core/src/services/api/mmlApi.ts` | 删 | -4 行（BackendMMLCustomCommand + mapBackendCustomCommand + createTemplate payload + updateTemplate payload）|
| `omcmb/frontend-core/src/mock/services/mmlService.ts` | 删 | -2 行（updateTemplate + cloneTemplate mock）|
| `omcmb/webcode/src/pages/mml/Console/components/AddTemplateModal.tsx` | 删 | -1 行（T-0090-a 留的 `productTypes: []` empty default 同步删除 + 注释更新）|

**初始版本号冲突**：原 000086 与 `seed/000086_seed_menu_icon_config_and_backfill.sql` 同号 → 重命名为 **000087** 解决（check-migrations.sh 净新增 0 冲突 ✅）

---

## §2. 精确边界守护

**只动 mml_custom_command 表 + 相关 Go/FE 引用**，**不动**：

| 同名 product_types 列 | 表/位置 | 状态 |
|---------------------|---------|------|
| `mml_commands.product_types` | 内建命令树 | **保持** — 不在 T-0090-b scope |
| `indicator_definitions.product_types` | KPI 指标表 (000035) | **保持** — T-0098 域，独立工作 |
| `MMLCommand.ProductTypes` (Go model.go L56) | 内建命令树 struct | **保持** |
| `MMLCommand.productTypes` (TS types/mml.ts L56) | 内建命令树类型 | **保持** |
| `BackendCommand.product_types` (TS mmlApi.ts L29, L209) | 内建命令树映射 | **保持** |
| `mockMMLCommands[].productTypes` (TS mock/data/mml.ts × 27 行) | 内建命令树 mock | **保持** |
| `handler_test.go:239` MMLCommand fixture | mml_commands 测试 | **保持** |
| `CommandTree/index.tsx` productTypes 列 | 内建命令树前端页 | **保持** |
| `seed/000004` + `seed/000006` 对 mml_commands.product_types 引用 | 内建命令种子 | **保持** |

scope grep 全工程确认上述清单完整且零误删。

---

## §3. 出口门检查

| 检查项 | 结果 | 备注 |
|--------|------|------|
| `go build ./...` | ✅ PASS | 无输出 |
| `go test -race -count=1 ./internal/mml/...` | ✅ PASS | mml 包全过 1.075s |
| `go test -race -count=1 ./...` 全工程 | ⚠️ 2 pre-existing FAIL | `TestPgRepo_Integration_ListPendingAllDevices` + `TestService_PG_RestorePendingQueues`（internal/task 包，scan NULL into *string for col `source_id`）；**stash 回到 main commit 74cb0a11 同样 FAIL** — 与 T-0090-b 完全无关 |
| webcode `npx tsc --noEmit` | ✅ PASS | 无输出 |
| FE 4 touched 文件 `npx eslint`（从 omcmb root） | ✅ 0 errors / 0 warnings | mml.ts / mmlApi.ts / mmlService.ts / AddTemplateModal.tsx |
| 全工程 `npm run lint` | ⚠️ 225 pre-existing | 与 T-0090-a verify 同一基线，与本任务无关 |
| `scripts/check-migrations.sh` | ⚠️ 19 pre-existing conflicts | 本任务 **0 净新增**（初始 000086 与 seed/000086 冲突已重命名为 000087 解决）|
| `migrate up` (`OMCGO_DB_DSN=... go run ./cmd/migrate up --paths migrations`) | ✅ PASS | goose v85 → v87；27.84ms |
| `migrate down`（验回滚）| ✅ PASS | goose v87 → v85；5.97ms；列 + 索引重建确认 |
| `migrate up`（最终态恢复）| ✅ PASS | goose v85 → v87；7.75ms |
| 无新增 TODO/FIXME/panic | ✅ | grep 验 |
| 无新增 `any` / `interface{}` | ✅ | 严格 TS / Go 类型 |
| 无新增 `if carrier ==` 硬编码 | N/A | 无运营商分支 |
| 累计型 deps | N/A | Deps `—` |

**新端点 E/R**：N/A（无新端点）
**新 metric/log**：N/A（无新埋点）

---

## §4. PgM 破坏性变更影响评估

| 项 | 数据 |
|----|------|
| Live dev DB 当前 mml_custom_command 行数 | 6 |
| 其中 product_types 非空（!= `[]`）行数 | **6**（100%）|
| 数据丢失影响 | 6 行的 product_types JSON 数组数据永久丢失 |
| 业务影响评估 | **可接受**：T-0098 ProductRegistry 已接管产品路由；product_types 标签已无消费者；UI 由 T-0090-a 删除；Down 段仅恢复 schema 不恢复数据（迁移注释明示）|
| Down 数据回填策略 | 不实现（per subtask R-NEW-1 决议）；Down 后 product_types 列重建为 NOT NULL DEFAULT `'[]'::jsonb`，所有行 `[]` |

Live DB up/down/up 三次演练验证：
1. Up（v85→v87）：`information_schema.columns` 不再含 product_types ✅；`pg_indexes` 不再含 `idx_mml_custom_command_product_types_gin` ✅
2. Down（v87→v85）：列 + 索引重建 ✅；6 行 product_types 全部为 `[]` 默认值（数据未恢复 — 与 Down 段注释一致）
3. Re-up（v85→v87）：列 + 索引再次下线 ✅；DB 处于 commit 后预期态

---

## §5. 子任务 ② 验收逐项核销（GWT-b）

> Given DB 有 mml_custom_command 数据 / When migrate-up / Then product_types 列删除；mml backend build/test 全过；前端无 product_types 引用

- ✅ DB 有 6 行 mml_custom_command 数据（live dev DB 实测）
- ✅ migrate-up 成功（goose v85 → v87，27.84ms）
- ✅ product_types 列已删除（information_schema 验证）
- ✅ mml backend `go build` 全过 + `go test -race ./internal/mml/...` 全绿
- ✅ 前端 `MMLCustomCommand` 类型已删除 productTypes 字段；`mmlApi.ts` BackendCustomCommand + mapping + payload 共 4 处清理；`AddTemplateModal` T-0090-a 留的 `productTypes: []` empty default 同步删除；webcode typecheck 全过

---

## §6. 用户回归确认（与 T-0090-a 同 pattern）

后端 + DB 已完成实测验证；前端 UI 行为无变化（T-0090-a 已删 UI 入口）。

**建议回归路径**：
1. MML Console → 新增公有/私有命令 Modal → 填写 commandName + commandCode + operationType → 保存
2. 验证后端 API `POST /api/v1/mml/templates` 返回 200，无 product_types 字段相关错误
3. 验证 DB 行 `SELECT * FROM mml_custom_command WHERE command_name='<新建名>'` 不含 product_types 列

---

## §7. S4 出口门

- [✓] 所有命令绿（go build/test + typecheck + lint + check-migrations + migrate up/down/up）
- [N/A] E/R ≥ 1（无新端点）
- [✓] 迁移双向演练通过（up/down/up 三次 live DB 验证）
- [N/A] metric/log 名全部能找到（无新埋点）
- [N/A] 累计型依赖阈值（无累计 deps）

**S4 PASS**。
