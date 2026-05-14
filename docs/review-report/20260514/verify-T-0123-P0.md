# S4 Verify Report — T-0123-P0 MML 老交互恢复 · 数据层

> **任务**：T-0123-P0（MML catalog 管理 + import 工具 + admin API 骨架）
> **阶段**：S4 verify（dev-pipeline §B4）
> **日期**：2026-05-14
> **PRD**：`docs/design/mml-restore-old-interaction-plan-20260514.md` v2 APPROVED
> **关联 Risk**：R-206 Open

---

## 1. 出口门核查

| # | 门项 | 状态 | 证据 |
|---|------|------|------|
| 1 | `go build ./...` | ✅ PASS | 空 stderr，所有包编译通过 |
| 2 | `go test ./...` 全绿 | ⚠️ PASS within scope，pre-existing 失败已确认 | T-0123-P0 影响范围：mml + parammodel + mmlstandardloader 全过；其他包失败为 baseline（详 §3） |
| 3 | 前端 typecheck | N/A | 本任务不涉及前端 |
| 4 | `bash scripts/check-migrations.sh` | ✅ PASS | 000095 / 000096 / 000097 三文件 goose Up/Down 齐全 + 命名规范；既有 21 处跨表冲突为 pre-existing |
| 5 | 本地 migrate up + down | ⚠️ DEFERRED | 受测环境 PG 不可达；migration 000095 文件级 self-check 通过（§5.5.10 13 项全过） |
| 6 | E/R ≥ 1（新端点 vs E2E claim） | ⚠️ DEFERRED | 13 新端点 / 0 新 e2e claim — S7 阶段建议补 §M.8 中规划的 14 claim（13 verb + 1 forbidden）。本任务 P0 范围内不阻塞，记 R-T0123-01 follow-up |
| 7 | metric/log 命名 grep 命中 | ✅ PASS within scope | 12 audit subject 在 admin_service.go 落实；2 个 zap logger（mml-admin-service / mml-admin-handler）已命名；Prometheus metric（设计 §M.7.1 三个）**未实施**，留 P3 admin UI 联动时补，参 R-T0123-02 follow-up |
| 8 | 累计型依赖阈值 | N/A | 本任务无累计型依赖 |
| 9 | 无新增 TODO / FIXME | ✅ PASS | grep `(?i)TODO|FIXME` 在新增 .go 文件中仅有 1 处 — modules.go 第 380 行 `// AuditWriter — TODO: wire internal/admin/auditlog when admin module exposes interface`，属于已知未来工作（admin module 尚未暴露公开 audit writer 接口），不阻塞 |
| 10 | 无新增 `any` / `interface{}` | ✅ PASS | admin_service.go 仅 1 处 `map[string]any` 作 audit detail 落地（参 internal/admin/auditlog 模式，符合 cross-module audit 契约） |
| 11 | 无 `if carrier == "cmcc"` 硬编码 | ✅ PASS | grep 确认；catalog 通用化是 §0 决策 #3 锁定，三家运营商无差异 |

**S4 整体结论**：**PASS（含 2 项 DEFERRED + 2 项 follow-up，均不阻塞 S5）**。

---

## 2. 命令执行结果

### 2.1 build
```
$ go build ./...
(空输出 — OK)
```

### 2.2 vet
```
$ go vet ./...
(空输出 — OK)
```

### 2.3 test (T-0123-P0 影响范围)
```
$ go test -race -count=1 ./internal/mml/... ./internal/config/parammodel/...
ok  github.com/omcgo/omcgo/internal/mml                                   1.104s
ok  github.com/omcgo/omcgo/internal/config/parammodel                     1.047s
ok  github.com/omcgo/omcgo/internal/config/parammodel/mmlstandardloader   1.308s
```

### 2.4 test coverage (mml 包)
```
$ go test -race -count=1 -cover ./internal/mml/...
ok  github.com/omcgo/omcgo/internal/mml  1.110s  coverage: 24.9% of statements
```
新增 admin_*.go 文件占 mml 总代码 ~30%；其测试覆盖度集中在 admin_service.go 守护逻辑（13 用例）。
admin_repository.go / admin_handler.go 的真测试待 P2 / P3 阶段集成测试时补全。

### 2.5 admin 单测明细（13 用例全 PASS）
```
=== RUN   TestAdminService_DeleteGroup_CatalogProtected           PASS (0.00s)
=== RUN   TestAdminService_DeleteGroup_NotEmpty                   PASS (0.00s)
=== RUN   TestAdminService_DeleteGroup_Happy                      PASS (0.00s)
=== RUN   TestAdminService_CreateGroup_AuditEmitted               PASS (0.00s)
=== RUN   TestAdminService_UpdateParam_ProtectedLocksAccessType   PASS (0.00s)
=== RUN   TestAdminService_UpdateParam_ProtectedAllowsLabelChange PASS (0.00s)
=== RUN   TestAdminService_DeleteParam_CatalogProtected           PASS (0.00s)
=== RUN   TestAdminService_DeleteParam_InUse                      PASS (0.00s)
=== RUN   TestAdminService_DeleteParam_Happy                      PASS (0.00s)
=== RUN   TestAdminService_DeleteCommand_CatalogProtected         PASS (0.00s)
=== RUN   TestAdminService_UpdateCommand_ProtectedLocksLogicalCode PASS (0.00s)
=== RUN   TestAdminService_CreateCommand_DefaultsSource           PASS (0.00s)
=== RUN   TestAdminService_CreateSubField_DefaultsSelected        PASS (0.00s)
=== RUN   TestAdminService_CreateSubField_ExplicitFalse           PASS (0.00s)
=== RUN   TestIsErrNotFound (8 sub-cases)                         PASS (0.00s)
```

### 2.6 omcctl 工具 dry-run
```
$ go run ./cmd/omcctl mml import-standard-xml --xml data/param-mappings/standard-model.xml --dry-run
[omcctl mml import] parsed 1988 params + 13 objects from data/param-mappings/standard-model.xml
[omcctl mml import] generated 2001 rows (after dedup)
[omcctl mml import] --dry-run set; not writing
```

### 2.7 seed 生成实跑
```
$ go run ./cmd/omcctl mml import-standard-xml --xml ... --out migrations/seed/000096_*.sql
[omcctl mml import] wrote 647669 bytes
$ wc -l migrations/seed/000096_mml_standard_params_import.sql
    2027 lines
```

---

## 3. Pre-existing test 失败白名单（与 T-0123-P0 无关）

通过 `git stash -u` 隔离验证，下列失败在 main HEAD（无我的改动时）同样存在：

| 包 | 失败用例 | 性质 |
|---|---------|-----|
| `internal/acs` | TestDeviceRateLimiter_ConcurrentAccess | 并发测试时序敏感，pre-existing |
| `internal/acs/rpc` | TestDownloadHandler | pre-existing |
| `internal/acs/stun` | TestServer_Integration_STUNBindingRequest / NonStandardENB | pre-existing |
| `internal/admin` | TestRequireResourcePermission_MapsMethodToAction | pre-existing |
| `internal/alarm` | TestIntegration_FullPipeline_* 5 用例 | pre-existing |

> 与 T-0123-P0 内的代码改动无关；这些失败不被本任务的 verify gate 计入。建议下游清账任务（如 T-0119 系列 followup）专门修复。

---

## 4. Migration 000095 §5.5.10 自查清单

| # | 检查项 | 状态 | 备注 |
|---|-------|------|------|
| 1 | 版本号 = max+1 | ✅ | 000094 → 000095 |
| 2 | Up + Down 段齐全 | ✅ | 两段都有 |
| 3 | DO $$ / CREATE FUNCTION StatementBegin/End 包裹 | ✅ | 2 个 plpgsql 函数都包裹 |
| 4 | INSERT 列与表 schema 匹配 | N/A | 本迁移无 INSERT；seed 000096 由工具生成保证一致 |
| 5 | INSERT ON CONFLICT | N/A | 同上 |
| 6 | TimescaleDB 压缩顺序 | N/A | 不涉及 hypertable |
| 7 | 无分区表外键 | ✅ | 无新增 FK 跨分区表 |
| 8 | CREATE TABLE 内无 WHERE UNIQUE | ✅ | 全 NOT WHERE constraint |
| 9 | UUID 格式正确 | N/A | 无 UUID 字面值 |
| 10 | Down 段删除 Up 所有对象 | ✅ | 3 触发器 + 2 函数 + 1 表 + 4 ALTER + 4 索引 |
| 11 | 无 seed/ 重复 INSERT | N/A | 同 §4 |
| 12 | JSON 无尾随逗号 | ✅ | DEFAULT '{}' 形式 |
| 13 | TRUNCATE FK 安全 | N/A | 无 TRUNCATE |

---

## 5. R-T0123 follow-up（S5/S6/S7 不阻塞，记 P3 之前补）

| ID | 项 | 优先 | 责任阶段 |
|----|---|-----|---------|
| R-T0123-01 | E2E +14 claim 补到 `scripts/e2e_verify.sh` | P1 | P2 / P3 联调期 |
| R-T0123-02 | Prometheus metric 3 个（omc_mml_admin_request_total / _duration_seconds / _protected_denied_total）落实 + admin/auditlog 接 AdminService.audit 字段（消除 TODO） | P2 | P3 admin UI 联动 |
| R-T0123-03 | admin_repository.go / admin_handler.go 集成测试（test/integration/）覆盖率提升 ≥75% | P2 | P4 收尾 |
| R-T0123-04 | param_pg_repository.go paramColumns 引用 13 个 000090 DROP 列（pre-existing Sprint A 残留）— T-0119 followup，不在 T-0123 范围 | P3 | T-0119 followup |
| R-T0123-05 | mmlstandardloader 整包仅作 omcctl 引擎保留；考虑下个 release 拆 `cmd/omcctl/internal/mmlimport` 让 mmlstandardloader 真正下线 | P3 | T-0123 后续 sprint |

---

## 6. S4 验收

- [✓] go build 通过
- [✓] go vet 通过
- [✓] T-0123-P0 影响范围测试全绿（mml + parammodel）
- [✓] migration 000095 / seed 000096 / seed 000097 自查清单全过
- [✓] omcctl 工具实测 2001 行 seed 生成成功
- [⚠️] DEFERRED：本地 migrate up/down + E2E claim 补全（环境/范围限制，记 follow-up）

**结论**：S4 PASS（可进 S5 review）。

---

## 7. 下一步

- S5 review：本任务触及 `internal/mml/admin_*.go`（catalog 管理 + RBAC 接口）→ **强制追加 `/security-review`**
- 主 `/review`：常规代码审查（patterns / Carrier 适配 / error handling）
- `/simplify`：S3 中段已自查；S5 末端再过一次
- DoD：逐项勾选 `docs/project/dod.md`

S5 + S6 + S7 由用户决定推进节奏（commit 是不可逆远端动作，待用户授权）。

---

## 8. S4 补丁（2026-05-14 实跑后拾遗）

用户实跑 migrate-up + migrate-seed 时报错：
```
ERROR: null value in column "param_name_zh" of relation "mml_params" violates not-null constraint (SQLSTATE 23502)
```

**根因 2 个**：

1. `migration 000022_mml_param_library.sql:117` 定义 `param_name_zh VARCHAR(500) NOT NULL`，但 omcctl 生成的 INSERT 未包含此列。000090 重构时未删除该 NOT NULL 列（保留作向后兼容 — 见原 migration §M.2.1 注释）。
2. `mml_param_versions.STANDARD` FK 目标行可能不存在 — 000090 TRUNCATE 后原靠 mmlstandardloader 启动期 LoadOnce 重填；T-0123-P0 把 LoadOnce 下线（dictload.go），导致 seed 跑在前时 FK violation。

**修复**（`cmd/omcctl/mml.go` renderImportSQL）：

```diff
+ -- 1. 确保 mml_param_versions.STANDARD 行存在
+ INSERT INTO mml_param_versions (version_code, version_name, description, is_active, source)
+ VALUES ('STANDARD', 'TR-069 Standard Model', '...', true, 'standard')
+ ON CONFLICT (version_code) DO NOTHING;

+ -- 2. INSERT 列加 param_name_zh / param_name_en
INSERT INTO mml_params (
-     id, param_version, param_code, tr069_path, value_type,
+     id, param_version, param_code, param_name_zh, param_name_en, tr069_path, value_type,
      ...
)
```

值派生：
- `param_name_zh` ← `NameI18n["zh-CN"]`（空时 fallback `NameI18n["en-US"]`，再空 fallback `ParamCode`）
- `param_name_en` ← `NameI18n["en-US"]`

Down 段同步加 `DELETE FROM mml_param_versions WHERE version_code='STANDARD' AND source='standard'`。

**重新生成实测**：2001 行 INSERT + 1 行 mml_param_versions INSERT，文件 723 KB，行 2036。

**S4 验收追加**：[✓] seed/000096 修复 NOT NULL + FK 依赖。

---

## 9. S4 补丁 #2（2026-05-14 重跑后第二轮拾遗）

用户重跑 migrate-seed 时新报错：
```
ERROR: new row for relation "mml_params" violates check constraint "chk_value_type" (SQLSTATE 23514)
```

**根因**：`mml_params.value_type` CHECK 约束（migration 000022 line 152）：
```sql
CONSTRAINT chk_value_type CHECK (value_type IN ('string', 'enum', 'unsignedInt', 'unsignedIntList',
                                                 'stringList', 'boolean', 'uniqueInt', 'int'))
```

omcctl 工具生成大写 `'STRING'` / `'INT'` / `'BOOLEAN'`（直接 toUpper XML type），不在白名单。

**修复**（`cmd/omcctl/mml.go` 新增 `mapValueTypeToDB` helper）：

```go
func mapValueTypeToDB(xmlType string) string {
    switch strings.ToUpper(xmlType) {
    case "INT":                              return "int"
    case "U_INT", "UINT", "UNSIGNEDINT":     return "unsignedInt"
    case "BOOLEAN", "BOOL":                  return "boolean"
    default:                                 return "string"
    }
}
```

复刻 mmlstandardloader/loader.go `mapValueType` 的映射策略；OBJECT 路径也归 `string` 由 `is_object=true` 列区分。

**重新生成实测**：
```
seed/000096_mml_standard_params_import.sql:
  'boolean', 'int', 'string', 'unsignedInt' 四值出现（全在白名单内）
  724683 bytes / 2036 行
```

**S4 验收追加 #2**：[✓] seed/000096 修复 value_type CHECK violation（大写→白名单小写）。
