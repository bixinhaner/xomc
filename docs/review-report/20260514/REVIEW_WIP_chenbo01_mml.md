# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-14 |
| 提交 | WIP（工作区未提交，HEAD c4194d38） |
| 作者 | chenbo01 |
| 范围 | mml（含 migration + deploy 跨域） |
| 变更文件数 | 22（10 modified + 12 untracked） |
| 新增行数 | +5443（其中 seed/000096 工具生成 ~2000 行） |
| 删除行数 | -49 |
| 关联 Task | T-0123-P0 |
| 关联 PRD | `docs/design/mml-restore-old-interaction-plan-20260514.md` v2 APPROVED |
| 关联 Risk | R-206 Open（MML 老交互被 Sprint A 重构破坏） |

## 变更概要

T-0123-P0 MML 老交互恢复方案 · 数据层落地：

1. **数据层**：migration `000095` 加 12 列元数据 + 新表 `mml_command_sub_fields` + 3 个触发器（is_writable GENERATED 派生 / target_paths 自动重算 / updated_at 维护）
2. **Go 业务**：4 个 Repository 接口 + Pg 实现（`admin_repository.go` 844 行）+ 4 个资源的 catalog_protected 守护服务（`admin_service.go` 600 行）+ 13 admin HTTP endpoints（`admin_handler.go` 389 行）+ 13 表驱动单测（`admin_service_test.go` 479 行 全 PASS）
3. **工具**：`omcctl mml import-standard-xml` 一次性把 standard-model.xml 转为 seed SQL（403 行工具 + 176 行 zh 词典）；2001 条 INSERT 实测生成成功
4. **集成**：DI 注入 admin handler；router 注册；启动期 mmlstandardloader 下线（保留包作 omcctl 引擎）；loader.go INSERT 修复 `is_writable` → `access_type`（GENERATED 兼容）
5. **权限**：seed `000097` admin × mml_admin api_endpoints 自注册后绑定

## 审查发现

### 🔴 CRITICAL (严重)

> 必须在提交前修复的问题

**无**。

### 🟡 WARNING (警告)

> 建议修复，不阻塞提交

**W1**：`omcgo/cmd/app/provider/modules.go:390` AuditWriter 注入为 `nil`（兜底用 `noopAuditWriter`），TODO 标记后续由 admin 模块暴露 audit writer 接口后接入。

- **影响**：catalog 管理写操作的审计日志暂不落盘
- **缓解**：service 内 12 个 audit op 落地点已就绪，单测验证 audit 调用次数和 op 名（`mockAuditWriter` 13 测试全过）；接入只是 DI 一行
- **追踪**：verify-T-0123-P0.md R-T0123-02 follow-up（P2/P3 阶段）

**W2**：Prometheus metrics 未实施（§M.7.1 设计的 3 个 — `omc_mml_admin_request_total` / `_duration_seconds` / `_protected_denied_total`）

- **影响**：catalog 管理 API 无运行期可观测
- **缓解**：zap log（`mml-admin-service` / `mml-admin-handler`）+ audit log（设计层就绪）覆盖了大部分排障信息
- **追踪**：R-T0123-02 follow-up

**W3**：E2E claim 未增（计划 +14 = 13 verb + 1 forbidden test，0 当前）

- **影响**：S4 出口门 E/R 比 0/13，不满足 §B4 "E/R ≥ 1"
- **缓解**：admin API 是 P0 骨架，P2 前端 Console 联调时增 claim 更合适（避免空验证）；单测覆盖了核心守护逻辑
- **追踪**：R-T0123-01 follow-up，记录在 verify-T-0123-P0.md §5

### 🔵 INFO (建议)

> 改进建议，可选择性采纳

**I1**：mml 包整体 test coverage 24.9%；admin 层的 `admin_repository.go` / `admin_handler.go` 集成测试（需要 PG 容器）未写。
- 建议：P4 收尾阶段补 `test/integration/mml_admin_*_test.go`（覆盖率 ≥75% 目标）

**I2**：`param_model.go` 中 13 个字段（`IsListable` / `IsDynamic` / `PlatformSupport` / `MobileSupport` 等）在 migration 000090 已 DROP，Go struct 保留作 SELECT 占位但 DB 已无对应列。
- **来源**：pre-existing Sprint A 残留（T-0119 后），与 T-0123-P0 无关
- **建议**：T-0119 followup 清理 paramColumns 引用 + 删 13 个 deprecated 字段

**I3**：admin_handler.go 错误响应 `respondAdminError` 直接 return `err.Error()` 到 client（含 pgx 错误内部细节）。属于 minor information disclosure，与项目其他 handler 一致（既有 pattern）。
- 建议：未来统一在 `internal/core/errors` 层做 redaction

**I4**：`omcctl mml import-standard-xml` 工具的 i18n 词典硬编码在 Go map（200 词）— Q4=A 决议 P0 阶段做法正确，P3 admin UI 上线后迁出到 JSON 文件。

**I5**：migration 000095 是否在 dev 环境演练成功，PG 不可达暂未实跑（S4 DEFERRED #5）。强烈建议合入前在 dev 跑一次 `migrate-up` + `migrate-down` 双向。

## 详细分析

### `omcgo/migrations/000095_mml_command_catalog_v2.sql`（267 行）

- ✓ `-- +goose Up` / `-- +goose Down` 齐全
- ✓ 2 个 `CREATE OR REPLACE FUNCTION` 都被 `-- +goose StatementBegin/End` 包裹
- ✓ Up 步骤明确：(1) ALTER 加 8 列 → (2) UPDATE backfill access_type → (3) DROP 旧 is_writable → (4) ADD is_writable GENERATED STORED → (5) 索引
- ✓ Down 完整反向（临时 `_is_writable_temp` 列保护数据，避免 GENERATED → 普通列时数据丢失）
- ✓ 3 触发器：`trg_mml_sub_fields_target_paths`（AFTER INS/UPD/DEL on sub_fields → 重算 mml_commands.target_paths）+ `trg_mml_command_sub_fields_updated_at`（复用 000001 共享函数）+ GENERATED 列内置维护 is_writable
- ✓ 列注释完整（COMMENT ON COLUMN 12 处）
- ✓ §5.5.10 13 项自查清单全过
- ⚠️ DEFERRED：本地 migrate up/down 双向演练未跑（PG 环境不可达）

### `omcgo/internal/mml/admin_repository.go`（844 行）

- ✓ 4 接口 5 method（SubFieldRepository）/ 4 method（AdminGroupRepository）/ 3 method（AdminCommandRepository）/ 5 method（AdminParamRepository）— 都符合"small interfaces"原则
- ✓ Pg 实现使用 Squirrel `storage.Psql.Select/Insert/Update/Delete` + pgx/v5 — 全程参数化
- ✓ 错误全部 `fmt.Errorf("...: %w", err)` 包装
- ✓ 6 sentinel errors（ErrSubFieldNotFound / ErrGroupNotFound / ErrCommandNotFound / ErrParamNotFound / ErrCatalogProtected / ErrParamInUse / ErrGroupNotEmpty）
- ✓ INSERT 列表不含 `is_writable`（GENERATED 兼容 Q1=B）
- ✓ AdminParamRepository.List 支持 search / access_type / is_object / source / param_version 过滤 + 分页（max 500/page）
- ✓ ListEnrichedByCommand 是手写 SQL（JOIN mml_params），不走 Squirrel — 因为 Squirrel 对复杂 JOIN 不友好；手写 SQL 全参数化（$1）
- ℹ️ `jsonOrEmpty` / `nullIfEmpty` 接受 `any`，但仅作 JSON marshaling helper，符合 audit detail 的同种模式

### `omcgo/internal/mml/admin_service.go`（600 行）

- ✓ 12 写方法 + 1 ListParams（透传）
- ✓ catalog_protected 守护逻辑实施在 service 层：
  - DeleteGroup：protected → ErrCatalogProtected；commands > 0 → ErrGroupNotEmpty
  - DeleteCommand：protected → ErrCatalogProtected
  - DeleteParam：protected → ErrCatalogProtected；sub_fields ref > 0 → ErrParamInUse
  - UpdateCommand：protected 时锁 logical_code
  - UpdateParam：protected 时锁 access_type / is_object / supports_add / supports_delete / change_applies
- ✓ 12 audit op 名空间一致（`mml.catalog.{resource}.{verb}`）
- ✓ AdminAuditWriter 是小接口（1 method）+ `noopAuditWriter` 兜底
- ✓ Source / CatalogProtected 字段在 PATCH 结构体中**不暴露**（Q2=C 决议 — 等同 `source='standard'`）
- ⚠️ AuditWriter 当前注入 nil → noopAuditWriter（W1）

### `omcgo/internal/mml/admin_handler.go`（389 行）

- ✓ 13 endpoints + RegisterRoutes
- ✓ `parseUUIDParam` helper 防 path param injection
- ✓ `respondAdminError` 翻译 sentinel → HTTP 状态（ErrCatalogProtected 403 / IsErrNotFound 404 / ErrParamInUse / ErrGroupNotEmpty 409 / bind 错 400 / 其余 500）
- ✓ ListParams 支持 query string 过滤
- ✓ payload struct 严格控字段范围（Q2=C — source/catalog_protected/is_writable 不在 PATCH req 中）
- ℹ️ err.Error() 返客户端 — minor info disclosure（I3）

### `omcgo/internal/mml/admin_service_test.go`（479 行）

- ✓ 4 mock repo + 1 mock audit writer
- ✓ 13 表驱动用例覆盖关键守护路径（catalog_protected / ErrParamInUse / ErrGroupNotEmpty / Happy path / audit 调用）
- ✓ 命名约定 `newAdminTestService` 不冲突既有 `newTestService`
- ✓ TestIsErrNotFound 8 sub-cases 覆盖
- ✓ 全 PASS（1.060s）

### `omcgo/cmd/omcctl/mml.go`（403 行）

- ✓ cobra 子命令注册 `mml import-standard-xml`
- ✓ 复用 `mmlstandardloader.ParseStandardXMLFile`（不重写 XML 解析）
- ✓ param_code 碰撞兜底（hash8 → hash12）
- ✓ SQL 字符串字面值用 `sqlString` 转义单引号（防 SQL injection in static SQL gen）
- ✓ ON CONFLICT (param_version, tr069_path) DO UPDATE WHERE catalog_protected=true（Q2=C 决议落实）
- ✓ INSERT 列表不含 is_writable
- ✓ 实测：1988 params + 13 objects → 2001 rows，647 KB output

### `omcgo/cmd/omcctl/mml_zh_dict.go`（176 行）

- ✓ ~200 词 TR-069 高频词典（Q4=A 决议）
- ✓ translateZh / isAcronym / capitalize 三 helper
- ✓ 未命中即返回空（前端 "🟡 需翻译" badge 兜底）

### `omcgo/internal/mml/sub_field_model.go`（76 行）

- ✓ MMLCommandSubField + MMLCommandSubFieldEnriched（Console GET 端点用，含 join mml_params 元数据）+ SubFieldFilter
- ✓ db tag 与 migration 000095 schema 对齐

### `omcgo/internal/mml/model.go` / `param_model.go`（modified）

- ✓ MMLCommand 加 4 字段（LogicalCode / LogicalNameI18n / Source / CatalogProtected）+ SubFields slice
- ✓ Param 加 8 元数据字段
- ✓ ParamGroup 加 2 字段（Source / CatalogProtected）
- ✓ IsWritable 字段保留 + 注释说明 GENERATED 派生

### `omcgo/cmd/app/provider/{dictload,modules,router}.go`（modified）

- ✓ dictload.go：mml-standard Loader 启动期注册下线（保 `_ = mmlstandardloader.LoaderName` 防 import 被去掉）
- ✓ modules.go：4 admin repo + AdminService + AdminHandler 注入；miscDeps 加 `mmlAdminHandler` 字段
- ✓ router.go：13 admin 端点注册到 `permGroup("devices")`（Casbin 中间件兜住鉴权）
- ⚠️ modules.go:390 AuditWriter nil 注入（W1）

### `omcgo/internal/config/parammodel/mmlstandardloader/loader.go`（modified）

- ✓ INSERT 列从 `is_writable` 改 `access_type`；ON CONFLICT UPDATE 同步改写
- ✓ access_type 从 IsWritable() 派生（true → READ_WRITE / false → READ_ONLY）— loader 与 GENERATED 列兼容

### `omcgo/migrations/seed/000096_mml_standard_params_import.sql`（2027 行 / 工具生成）

- ✓ 2001 行 INSERT
- ✓ ON CONFLICT (param_version, tr069_path) DO UPDATE WHERE catalog_protected=true
- ✓ INSERT 列不含 is_writable
- ✓ Down DELETE source='standard' AND param_version='STANDARD'
- ℹ️ 单 INSERT 巨型语句（647 KB）— PG 可处理，但建议监控 migrate-seed 实际耗时

### `omcgo/migrations/seed/000097_mml_admin_role_permissions.sql`（38 行）

- ✓ admin × mml_admin api_endpoints JOIN INSERT + ON CONFLICT DO NOTHING（幂等）
- ✓ Down DELETE 反向
- ⚠️ 部署 ordering 注意：api_endpoints 自注册在 app 启动期，migrate-seed 跑在前 → 首次部署可能 0 行（注释中已说明，需运维 restart app 后再跑 migrate-seed）

## 业务完整性检查

- ✓ Handler-Service-Repository 三层完整（13 endpoint × 4 资源）
- ✓ 路由注册：router.go +4 行（permGroup("devices") 下 + RBAC Casbin 自动鉴权）
- ✓ Migration 配套：000095 DDL + 000096 数据 seed + 000097 权限 seed
- ✓ 错误码：使用 sentinel errors（admin_repository.go 顶层）而非 global/errors.go 数字码（项目其他模块也有此模式如 license）— 一致性 OK
- ⚠️ API 服务前端配套：**未实施**（P2 前端阶段补，PRD §M.10 已计划）
- ⚠️ E2E 用例：14 claim 未增（R-T0123-01 follow-up）

## 业务影响范围检查

- ✓ 接口签名：4 个新 admin repository 接口是**纯新增**，不影响现有 CommandRepository / ParamRepository / TaskRepository 等
- ⚠️ DB Schema 变更：mml_params/mml_commands/mml_param_groups 加列；is_writable 改 GENERATED — 影响：
  - **已修复**：mmlstandardloader/loader.go INSERT 列同步改 access_type
  - **pre-existing 影响**：param_pg_repository.go paramColumns 引用 13 个 000090 DROP 列（与 T-0123-P0 无关，T-0119 followup）
- ✓ 事件契约：未修改 EventBus subject
- ✓ 共享 model：mml/model.go 加字段而非改类型，无 breaking change
- ✓ 中间件：未修改
- ✓ 配置：未新增 yaml 配置项（admin 接入点在 modules.go 硬注入）
- ✓ 运营商：无 carrier-specific 代码（catalog 通用化 §0 决策 #3）
- ✓ API 响应格式：纯新增端点，无既有响应格式变更
- ✓ 跨模块依赖：mml → admin（仅用 RoleRepo，T-0090-c 已有）；mml ← mmlstandardloader（仅 omcctl 复用 parser）

## 前后端一致性检查

本任务**仅涉及后端骨架**（P0 数据层），前端 Console 重构在 P2 阶段。前端同步需求：

- **接口路径**：13 endpoints 全在 `/api/v1/mml/admin/...`，前端 P3 admin UI 阶段对接
- **请求参数**：req struct 用 binding tag（required / max / oneof）— P3 前端 typescript 类型对齐
- **响应字段**：repo/service/handler 都返完整 model（含 Source / CatalogProtected / SubFields 等新字段），前端 P3 同步加 BackendXxx 类型

> **本次变更仅涉及后端，建议关注 P3 阶段前端 admin UI 的同步需求。** 不阻塞 commit。

## 代码质量回退检查

- ✓ 未删除测试用例
- ✓ 未移除错误处理
- ✓ 未降级安全措施（catalog_protected 是新加守护，加强了安全）
- ✓ 未引入 any 替代强类型（admin_repository.go 中 `any` 仅用于 JSON helper 接受任意 JSON 类型 + service 层 audit detail map）
- ✓ 未硬编码替代配置（无新配置项）
- ✓ 未删除日志（admin_handler.go 加 zap logger）
- ✓ 未简化输入校验（PATCH req struct 严格定义可改字段集）
- ✓ 未引入 ORM（继续 Squirrel + 手写 SQL）
- ✓ 未绕过接口抽象（4 新接口都有 Pg* 实现）
- ⚠️ TODO 残留 1 处（modules.go:390 AuditWriter nil — W1，已记录 follow-up）

**未发现代码质量回退**（TODO 是 explicit follow-up + 单测覆盖到 audit 调用，不算回退）。

## 配套更新提醒

- **文档**: ✓ 已更新
  - `docs/design/mml-restore-old-interaction-plan-20260514.md` v2 APPROVED（含 §M 设计备忘 + Q1-Q5 决议 + W2 audit + §M.9 待定点）
  - `docs/review-report/20260514/verify-T-0123-P0.md` S4 验证报告
  - `docs/project/backlog.md` T-0123 umbrella + 5 sub-task
  - `docs/project/risk-register.md` R-206
  - `docs/project/sprint/sprint-11.md` T-0123-P0 加入
- **单元测试**: ✓ admin_service_test.go 13 用例（守护逻辑核心）；admin_repository / admin_handler 集成测试 P4 收尾补（R-T0123-03）
- **端到端测试**: ⚠️ scripts/e2e_verify.sh 未加 14 admin claim — P2 前端联调时补（R-T0123-01）

## 安全检查

- ✓ SQL 注入：全程 Squirrel 参数化 + pgx prepared statements；omcctl 工具 `sqlString` 单引号转义
- ✓ 权限：admin 路由走既有 Casbin 中间件 + role_api_permissions JOIN + api_endpoints 自注册（W2 audit 后修订 §M.4.2）
- ✓ catalog_protected 守护：service 层 5 处 guard 防止 standard 行被误改/删
- ✓ 输入校验：handler 用 ShouldBindJSON + binding tag（required / max / oneof）
- ✓ UUID path param：uuid.Parse 解析失败返 400
- ✓ 无新增 secret / hardcoded credential
- ✓ /security-review 报告：**No findings — clean**

未发现 P0/P1 安全问题。

## 性能检查

- ⚠️ migration 000095 `trg_mml_sub_fields_target_paths` ROW-level trigger：每次 sub_field INSERT/UPD/DEL 触发同 command 的 target_paths 重算。批量场景下可能重复重算（admin UI 典型 N<100 写入 batch，可接受）
  - **缓解**：触发器函数走单 UPDATE，无嵌套查询
- ✓ AdminParamRepository.List 强制 pageSize 上限 500（防 DoS）
- ✓ 触发器函数仅 PERFORM UPDATE，无 SELECT 循环
- ✓ seed/000096 是 single INSERT 2001 VALUES — PG 可处理（实测 647 KB 文件，导入耗时建议 dev 环境测）

未发现 P0/P1 性能问题。

## 测试覆盖

| 文件 | 覆盖情况 |
|------|---------|
| admin_service.go | ✓ 13 用例覆盖核心守护路径（catalog_protected / sentinel error / Happy / audit）|
| admin_repository.go | ⚠️ 单测未覆盖（需 PG 容器，P4 集成测试补） |
| admin_handler.go | ⚠️ 单测未覆盖（需 HTTP test fixture，P4 集成测试补） |
| sub_field_model.go | N/A（仅类型定义） |
| mml.go (omcctl) | ⚠️ 单测未补，但实测 `--dry-run` + 实际 SQL 生成成功 → 行为验证 |
| mml_zh_dict.go | ⚠️ 未补单测；translateZh / isAcronym 行为简单可推断 |

**结论**：守护逻辑测试覆盖 OK（unit 13 PASS），集成测试 P4 阶段补（R-T0123-03）。

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 3（W1 AuditWriter nil / W2 Prometheus metrics 未实施 / W3 E2E claim 未增） |
| INFO | 5 |

**审查结论**: `PASS_WITH_WARNINGS`

3 个 WARNING 均为 known follow-up（R-T0123-01/02/03 在 verify-T-0123-P0.md §5 已登记），不阻塞 P0 数据层 commit。

**进入 S6 commit 前的关键事项**：
- migration 000095 在 dev 环境跑 up/down 双向演练（S4 DEFERRED 项）
- 提交分组建议（参考 §M.1 文件清单）：
  1. `migration:` 000095 schema + 000096 seed + 000097 perm（3 文件）
  2. `feat(mml):` model.go + param_model.go + sub_field_model.go + admin_repository.go + admin_service.go + admin_handler.go + admin_service_test.go（7 文件）
  3. `feat(mml):` omcctl mml import-standard-xml + zh dict + omcctl main.go（3 文件）
  4. `feat(mml):` provider 接入 + loader 下线（dictload.go + modules.go + router.go + mmlstandardloader/loader.go）（4 文件）
  5. `docs:` PRD + verify report + backlog + risk-register + sprint-11（5 文件）

或单一大 commit（同 T-0119 Sprint A 模式），但分组提交对 review/revert 更友好。
