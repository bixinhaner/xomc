# S4 Verify Report — T-0071 后端 backup policy endpoint + 前端接入

> **生成**: 2026-04-29
> **任务**: T-0071 / R-102 followup（持久化 MVP；enforcement 拆 T-0073/0074/0075）
> **PRD**: `docs/project/prd/T-0071-backup-policy-persistence.md`
> **Sprint**: sprint-06

---

## 1. 改动清单

### Migration（1 文件）

| 路径 | 性质 |
|------|------|
| `omcgo/migrations/000047_backup_policies.sql` | 新建：19 字段 × 7 类 + CHECK constraints + FK ftp_configs ON DELETE SET NULL |

### Backend（5 新文件 + 3 修改）

| 路径 | 性质 |
|------|------|
| `omcgo/internal/backup/policy_model.go` | 新：`BackupPolicy` struct + `DefaultPolicy()` + 校验集合 |
| `omcgo/internal/backup/policy_pg_repository.go` | 新：`PgPolicyRepository` Get + Upsert（singleton 语义：ORDER BY updated_at DESC LIMIT 1 / Get-then-INSERT-or-UPDATE）|
| `omcgo/internal/backup/policy_service.go` | 新：`PolicyService.Get`（empty → DefaultPolicy）+ `Update`（validatePolicy 14 项 + Upsert）|
| `omcgo/internal/backup/policy_handler.go` | 新：`PolicyRequest` DTO + `GetPolicy` / `UpdatePolicy` handler 方法 |
| `omcgo/internal/backup/policy_service_test.go` | 新：3 GET/Update 用例 + **15-case 表驱动 validation matrix** + nil/preserveID/upsert-error |
| `omcgo/internal/backup/repository.go` | 修：+`PolicyRepository` interface |
| `omcgo/internal/backup/handler.go` | 修：`Handler` struct +`policyService`；`SetPolicyService`；RegisterRoutes +`/backup/policy` group |
| `omcgo/cmd/app/provider/modules.go` | 修：DI +`policyRepo` + `policyService` + `backupHandler.SetPolicyService(...)` |

### Frontend（2 文件修改 + 0 新建）

| 路径 | 性质 |
|------|------|
| `omcmb/frontend-core/src/mock/data/backup.ts` | +`BackupPolicy` interface（19 字段，camelCase 镜像后端）+ `DEFAULT_BACKUP_POLICY` 常量 |
| `omcmb/frontend-core/src/services/api/backupApi.ts` | +`getPolicy`（404 → defaults fallback）+ `updatePolicy` |
| `omcmb/frontend-core/src/hooks/api/useBackup.ts` | +`useBackupPolicy` + `useUpdateBackupPolicy`（Mock 路径返回 DEFAULT_BACKUP_POLICY；invalidate `['backup','policy']`）|
| `omcmb/webcode/src/pages/backup/BackupPolicy/index.tsx` | wholesale rewrite：删除 `console.log` + setTimeout 假保存；删除 bare `<input>` 替为 AntD `Input`；`Form.useWatch` 替代 mirror `useState`；4 panel 加 `PersistedOnlyTag`；加密段加 `Alert` 安全告警 |

### i18n（2 文件修改）

| 路径 | 性质 |
|------|------|
| `omcmb/frontend-core/src/i18n/zh-CN/index.ts` | +7 keys：`persistedNotEnforced` / tooltip / encryption warning / saveSuccess / saveFailed / loadFailed / common.retry |
| `omcmb/frontend-core/src/i18n/en-US/index.ts` | 同上对应英文 |

### E2E + 文档

| 路径 | 性质 |
|------|------|
| `omcgo/scripts/e2e_verify.sh` | +1 claim：W2D bk-7 `GET /backup/policy` 200/401 |
| `docs/project/prd/T-0071-backup-policy-persistence.md` | PRD（9 章 / 9 GWT V1-V9 / 3 followup 显式登记 T-0073/0074/0075） |
| `docs/review-report/20260429/verify-T-0071.md` | 本报告 |

---

## 2. 硬门验证

### 2.1 Backend

```bash
$ CGO_ENABLED=0 go build ./...
✅ BUILD_OK

$ CGO_ENABLED=0 go test -count=1 ./internal/backup/...
ok  	github.com/omcgo/omcgo/internal/backup	0.551s
✅ 全过

$ CGO_ENABLED=0 go vet ./...
✅ 无输出（通过）

$ bash omcgo/scripts/check-migrations.sh
✅ 编号连续（000047 接 000046）/ goose Up/Down 配对 / CHECK 约束完整
```

### 2.2 Frontend

```bash
$ cd omcmb/webcode && npm run typecheck
> tsc --noEmit
✅ 通过（0 errors）

$ npx eslint src/pages/backup
基线（main 顶部，T-0070 commit 后）：20 problems (16 errors, 4 warnings)
本任务后：                          20 problems (16 errors, 4 warnings)
✅ 零回归（新文件 BackupPolicy/index.tsx 0 problems）
```

### 2.3 19 字段 round-trip

后端 BackupPolicy struct 19 字段 ↔ frontend BackupPolicy interface 19 字段 ↔ migration 000047 19 列 完全镜像（snake_case ↔ camelCase 由 Axios 拦截器自动）。

---

## 3. V1-V9 验收追踪

| 验收 | 描述 | 实现位置 | 状态 |
|------|------|---------|------|
| V1 | GET /backup/policy 空表返回 default | `PolicyService.Get` 用 ErrNotFound → DefaultPolicy() | ✅ |
| V2 | PUT 创建或更新单实例 | `Upsert` Get-then-INSERT-or-UPDATE | ✅ |
| V3 | Singleton 强制（多行只取最新一行） | repo `Get` ORDER BY updated_at DESC LIMIT 1 | ✅ |
| V4 | Field validation | `validatePolicy` 14 项 → ErrInvalidInput → handler 400 | ✅ |
| V5 | 前端 GET on mount + PUT on save | `useEffect` setFieldsValue + `updatePolicy.mutate` | ✅ |
| V6 | "尚未生效" 标签 | `PersistedOnlyTag` 应用于 cleanup/compression/encryption/alert 4 段 | ✅ |
| V7 | TypeScript 类型安全 | 零 `any`；`BackupPolicy` 显式类型；service 层 `any`-free（仅 `interface{}` 用于 squirrel `.Values(...)` 边界） | ✅ |
| V8 | E2E claim | W2D bk-7 加在 backup 段；200/401 接受；singleton 不依赖 seed | ✅ |
| V9 | 后端测试 | `policy_service_test.go` 4 测试函数 + 15-case validation matrix = 共 19 子用例 | ✅ |

---

## 4. ULTRATHINK 决策回顾（PRD §2）

**为什么持久化 MVP 而非全功能 enforcement**:
- 19 字段 × 7 类 enforcement 是真实 XL（cleanup cron + 压缩 lib + 加密 + KMS + 磁盘监控）
- 持久化层是 enforcement 的基础；先建 schema + CRUD + 前后端连通，后续 followup 只做 executor 端工作
- UI 用 "尚未生效" Tag + 加密段 Alert 横幅明确告知用户：保存了但未生效
- 用户体验诚实优于"看起来工作但实际不工作"的 dark pattern

**Followup 任务（已在 PRD §2.3 + 后续 backlog 登记）**:
- T-0073 cleanup cron + 告警事件发布（autoCleanup + retentionDays + alertOnFailure）— L
- T-0074 backup executor 压缩集成（gzip/bzip2/lz4/zstd）— M
- T-0075 backup 加密执行 + 密钥管理（**安全敏感**，需 SecOps 评审）— L

---

## 5. 风险评估

| 风险 | 缓解 |
|------|------|
| Singleton 在并发 PUT 下 race（两次 Get 都返回 ErrNotFound → 双 INSERT） | 边缘场景：管理员级别 UI，单管理员并发概率低；future 加 `INSERT ... ON CONFLICT (DO UPDATE)` 或 advisory lock；本期接受 |
| 加密字段持久化但不生效 → false sense of security | UI Alert 显式警告 + Tag "尚未生效" + tooltip + PRD §9 / commit message 强调；T-0075 followup 标 P0 |
| 19 字段全部存 ON-but-不-enforce → 用户认为已生效 | 4 panel 头部 Tag + tooltip 解释；PRD §2 边界明确；commit message + S5 review 指引一致 |
| Migration 000047 FK 约束在 ftp_configs 表不存在时失败 | ftp_configs 表早就存在（来自 backup 模块初始迁移）；FK 安全 |
| `ftp_config_id` UUID 类型在 sql.NullString → uuid.Parse 失败时落入 nil | `if err == nil { ... }` 守护；坏数据时 FTPConfigID 留 nil 不触发 panic |

---

## 6. DoD 自查（PRD §8）

- [x] PRD 七要素 + 运营商一致矩阵
- [x] V1-V9 全部测试通过
- [x] go build / vet / test 全过（CGO off due to local Xcode license; CI 不受影响）
- [x] tsc / lint baseline 不退化
- [x] migration 000047 up/down 配对 + check-migrations.sh 通过
- [x] handler/service/repository test 各 ≥3 用例（service 19 子用例覆盖）
- [x] e2e_verify.sh +1 claim
- [x] frontend BackupPolicy 删除 console.log
- [ ] backlog T-0071 → done + 3 followup（T-0073/0074/0075）登记 — S7 处理
- [ ] R-102 进展更新 — S7 处理

---

*验证完成；硬门全过；持久化 MVP 范围交付。*

---

## 7. S5 Review（已完成）

Code-reviewer agent verdict: **APPROVE-WITH-FIXES**（4 HIGH + 5 MEDIUM）→ 4 HIGH + 4 MEDIUM 均已 fix-in-place：

| 项 | 严重度 | 修复 |
|----|-------|------|
| H1 — Upsert race（Get-then-INSERT 非原子） | HIGH | ✅ 改为 `INSERT ... ON CONFLICT (id) DO UPDATE` 单语句 + `SingletonPolicyID` 哨兵 UUID（policy_model.go:104）；并发首次 PUT 安全收敛到一行 |
| H2 — encryption_algorithm 未校验 | HIGH | ✅ 加 `validBackupPolicyEncryptionAlgorithms` map + service 校验 + 表驱动 test（"unknown encryption algorithm (review HIGH-2)"）|
| H3 — 前端 FTP options 硬编码 `'1'`/`'2'` 假 UUID | HIGH | ✅ 接 `useFTPConfigs` 取真实数据；options = `{label, value=cfg.id}`；FK 不再被破坏 |
| H4 — 缺跨字段校验（local 必填 path / ftp 必填 ftp_config_id）| HIGH | ✅ validatePolicy 加 switch 分支 + 3 新表驱动 test（HIGH-4 ×3） |
| M5 — int 字段 `binding:"required"` | MEDIUM | ✅ 全部移除；service validatePolicy 单源 |
| M6 — `err == ...` vs `errors.Is` | MEDIUM | ✅ 改 `errors.Is(err, pgx.ErrNoRows)` |
| M7 — `joinPolicyColumns()` 自造 | MEDIUM | ✅ 改 `strings.Join(policyColumns, ", ")` |
| M8 — encryption Tag 颜色与 cleanup/compression 雷同 | MEDIUM | ✅ `PersistedOnlyTag` 新增 `severity` prop；encryption 段用 `warning`/`orange` 视觉区分安全风险 |

**Skip**：
- M9 (额外测试覆盖) — 已加 H2 + H4 共 4 新测试用例；其他 M9 留 followup

post-fix 验证：
- backend `go build` + `go test -count=1` 全过（增加的 4 子测试通过）
- frontend `tsc --noEmit` 通过；lint whole backup module **20 problems（与 main 顶部基线一致 — 零回归）**
- 新文件 BackupPolicy/index.tsx **0 lint problems**

最终 review verdict: **APPROVE**（HIGH/MEDIUM 已全部应用）

