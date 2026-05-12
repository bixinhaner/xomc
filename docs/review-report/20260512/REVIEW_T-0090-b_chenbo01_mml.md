# Code Review — T-0090-b (drop mml_custom_command.product_types)

**Reviewer**: Claude (self-review)
**Date**: 2026-05-12
**Scope**: 9 files / 1 new migration + 4 backend + 4 frontend / Type=ref
**Related**: [`verify-T-0090-b.md`](./verify-T-0090-b.md)

---

## §1. Findings 严重度汇总

| Severity | Count |
|----------|-------|
| **CRITICAL (P0)** | 0 |
| **HIGH (P1)** | 0 |
| **MEDIUM (P2)** | 0 |
| **LOW (P3)** | 1 |
| **NOTE** | 4 |

无 P0/P1 issue。可以合入。

---

## §2. 逐文件审查

### 2.1 `omcgo/migrations/000087_drop_mml_custom_command_product_types.sql` (新建)

**优点**：
- Up: `DROP INDEX IF EXISTS` + `DROP COLUMN IF EXISTS`（双 IF EXISTS 幂等防护）
- Down: `ADD COLUMN IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS`（同样幂等）
- 文件头注释清晰说明：scope（仅动 mml_custom_command）、不动的同名列（mml_commands / indicator_definitions）、数据丢失不可逆
- Down 段 schema-only 恢复，注释明示"数据无法回填"
- 版本号 000087 已规避与 seed/000086 冲突，**净新增 0 conflict**（check-migrations.sh 19 个 pre-existing 全为既有冲突）

**NOTE-1**：goose StatementBegin/End 注解不需要（DROP/ALTER 是简单 DDL，不含 DO/PL-pgSQL 块）— 符合 `omcgo/CLAUDE.md §5.5.1` 规则

**LOW-1**：Down 段 ADD COLUMN BACK 使用 `NOT NULL DEFAULT '[]'::jsonb` — 这与原 schema (migrations/000014/000026) 一致；但若未来 PG 升级版本对默认值的并发处理变化，DEFAULT 表达式语义可能微调。**建议**：当前不动；release-gate 集中清理时如有 migration 整顿可一并审视。无需修改。

### 2.2 `omcgo/internal/mml/model.go` 删 1 行

**优点**：纯删字段，无副效。Struct 仍保持 12 字段，contiguous tag 不冲突。

### 2.3 `omcgo/internal/mml/service.go` 删 3 段

**NOTE-2**：删除后 `CreateCustomCommand` / `UpdateCustomCommand` / `CloneCustomCommand` 三处对称清理，无悬空逻辑。`existing.ProductTypes = cmd.ProductTypes` 这种 update 模式从此 obsolete，但不影响其它字段（categoryGroup / parameters / paramPaths 的 nil-check + 复制 pattern 都保留）

### 2.4 `omcgo/internal/mml/handler.go` 删 4 处

**NOTE-3**：CreateCustomCommandRequest / UpdateCustomCommandRequest 两 DTO 的字段定义 + 两 struct literal 的引用都对称清理。Gin 的 `ShouldBindJSON` 对未知 JSON 字段是 silent ignore（不 panic），即使旧客户端误传 `product_types` 字段，请求仍能正常完成（unknown field 被忽略）— 兼容性友好

### 2.5 `omcgo/internal/mml/pg_repository.go` 删 ~16 处（PgCustomCommandRepository 分支）

**优点**：
- `customCommandColumns` 列表对齐数据库 schema（删 `product_types` 后剩 12 列与 Scan 12 字段对齐）
- `Create` / `Update` 双方法对称清理 Marshal 块 + Columns/Values + Set 子句
- `scanCustomCommand` / `scanCustomCommandRow` 双 helper 对称清理 productTypesJSON 变量声明 + Scan 参数 + Unmarshal 块 + nil-fallback 块

**精确边界**：完全没动 `PgCommandRepository` 分支（约 lines 1-1240，处理 mml_commands 表）— grep 残留 11 处全部在 mml_commands 路径（model.go:56 / pg_repository.go:52,257-302）和 handler_test.go:239 MMLCommand fixture，符合 "T-0090-b 只动 mml_custom_command" 设计约束 ✅

### 2.6 `omcmb/frontend-core/src/types/mml.ts` 删 1 行

**NOTE-4**：MMLCustomCommand interface 删除 productTypes 后仍保留 12 字段；webcode + frontend-core typecheck 全过 = 全工程无悬空类型引用

### 2.7 `omcmb/frontend-core/src/services/api/mmlApi.ts` 删 4 处

**优点**：
- BackendMMLCustomCommand interface 删 `product_types: string[] | null` → 与后端 schema 一一对齐
- mapBackendCustomCommand 删 `productTypes: bc.product_types || []`
- createTemplate / updateTemplate payload 双方法对称删除（createTemplate 静态 payload obj / updateTemplate 条件 if 块）— pattern 对称无悬空

### 2.8 `omcmb/frontend-core/src/mock/services/mmlService.ts` 删 2 处

**优点**：updateTemplate / cloneTemplate 两个 mock 方法的返回 obj 同步清理

### 2.9 `omcmb/webcode/src/pages/mml/Console/components/AddTemplateModal.tsx` 删 1 行 + 注释更新

**优点**：
- 删除 `productTypes: []` empty default — T-0090-a 留的 schema 兼容垫片现在自然下线
- 同步更新注释：原注释说"T-0090-b 才真删 column"现改为"productTypes column 已由 T-0090-b 真删"反映新事实
- categoryGroup='' / paramPaths=[] empty default 保留（这两个字段 schema 仍 required，不在 T-0090-b scope）

---

## §3. 安全审查（FE-only + 后端 schema）

- 触及 `internal/mml/`（非 `internal/admin/` 或 `middleware/auth*`）→ **不强制 /security-review**
- 迁移破坏性变更：经 PgM 决议（subtask 文件 R-NEW-1 + 本次 live DB 6 行 100% 非空数据评估），数据丢失可接受
- 无认证 / 鉴权 / 输入校验 / SQL 注入 / 路径遍历相关改动 — 纯 column drop + 引用清理
- 后端 DTO 删字段是"减少接受面" — 比"增加接受面"安全（不引入新攻击面）

**安全 verdict**：无 security finding。

---

## §4. DoD 逐项核销（`docs/project/dod.md` 通用 + 模块特定）

### 编译与类型
- [✓] 后端 `go build ./...` 通过
- [⚠️] 后端 `go test ./...` 全绿 — **2 pre-existing FAIL 在 internal/task** 包（`TestPgRepo_Integration_ListPendingAllDevices` + `TestService_PG_RestorePendingQueues`），与 T-0090-b 完全无关；**stash 回到 main commit 74cb0a11 同样 FAIL** 已验证。mml 包测试全绿
- [N/A] 后端 `golangci-lint run` — 本会话不跑（lint 全工程级 baseline）；改动文件均 gofmt 自然格式
- [✓] 前端 `cd omcmb/webcode && npx tsc --noEmit` 通过
- [✓] 前端 4 文件 `npx eslint` 0 errors / 0 warnings

### 测试
- [N/A] 新增/修改的代码包含单元测试 — 本任务是"删字段+清扫"，无新行为，仅清理；mml 包既有测试全过；handler_test.go MMLCommand fixture 不在本任务 scope（保持）
- [N/A] 成功 + 失败两路径 — 同上，纯清理无新行为
- [N/A] 新 REST 端点 E2E — 无新端点
- [N/A] 修复 bug 回归测试 — 不是 bug fix
- [N/A] 覆盖率不退 — 行数减少，测试数不变；覆盖率绝对值变化不显著
- [✓] 禁止禁用失败测试 — 0 测试被禁用

### 迁移与数据
- [✓] 新迁移文件编号 **连续递增** — 000087 = 当前 max(85) + 2（86 已被 seed 占用，cross-dir 冲突跳过 86 选 87，符合 `omcgo/CLAUDE.md §5.5` "如果发现重复，将后合并的文件重命名为更大版本号"）
- [✓] `up/down` 配对，`down` 真正可回滚 — live DB up/down/up 三次实测验证
- [✓] 破坏性变更（删字段）有迁移注释明示数据丢失 — Down 段头部注释 "Down 仅恢复 schema 至 Up 之前的结构，历史数据无法回填"
- [N/A] TimescaleDB hypertable — 不涉及

### 代码规范
- [✓] 无遗留 TODO / FIXME / panic — grep 验证
- [✓] 错误处理符合 `CLAUDE.md §8.2` — pg_repository Marshal 错误用 `fmt.Errorf("marshal X: %w", err)` 保留（虽然删了 product_types Marshal，仍保留 parameters/paramPaths Marshal 同 pattern）
- [✓] SQL 构建使用 Squirrel — `storage.Psql.Insert/Update` 保留
- [N/A] Carrier 接口 — 无运营商分支
- [✓] 前端 BackendXxx → mapBackendXxx → Xxx — BackendMMLCustomCommand + mapBackendCustomCommand + MMLCustomCommand 三层映射完整
- [✓] 无 `any` — Go + TS 严格类型
- [N/A] zap 结构化日志 — 无新日志埋点

### 文档
- [✓] PR 说明含 Why（"productTypes UI 已删 / dead column / 真删 schema"）
- [✓] 关联 Backlog Task: T-0090-b
- [✓] 关联 PRD: subtask 文件 + backlog T-0090 Notes
- [N/A] CLAUDE.md 同步 — 无约定变更
- [N/A] Swagger 同步 — 后端 DTO 字段删减；OpenAPI 文档若有自动生成会同步；项目无独立 swagger 手维护

### 流水线闭环
- [ ] commit footer 五元组（待 S6 执行）
- [ ] backlog.md Task 状态回写（待 S7 执行）
- [N/A] 快速通道 postmortem — type=ref 走 S1+S2-S7 完整流程（S0 可省 per §C 但已等同已 done at umbrella）

### 安全
- [N/A] 全部 — 纯删字段，无认证/鉴权/输入面变化

### 可观测性
- [N/A] 全部 — 无新指标/日志/trace

### 模块特定（前端）
- [✓] API/Hook/Store/Type 归属：types/mml.ts / mmlApi.ts / mmlService.ts mock 都在 `frontend-core/`；UI 改动 AddTemplateModal.tsx 在 `webcode/`
- [✓] 改 frontend-core 评估 v2/v3 影响：MMLCustomCommand 删一个 string[] 字段，v2/v3 若复用同 mmlApi 也会自然受益（少 4 字段映射）；mock 数据 mml.ts 不变（27 行都是 mml_commands 路径，OUT OF SCOPE）

### 模块特定（数据库迁移）
- [✓] 版本号连续递增（87 = max+2 跳过 seed 占用的 86）
- [✓] up/down 配对真可回滚
- [✓] 破坏性变更有评估（Live DB 6 行 100% 非空数据丢失，subtask 文件 R-NEW-1 决议可接受）
- [✓] check-migrations.sh 净新增 0 冲突

---

## §5. 结论

**APPROVE** — 无 P0/P1 issue；1 个 LOW 不强制；4 个 NOTE 仅记录。

可以进 S6 commit。
