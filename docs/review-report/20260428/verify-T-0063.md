# T-0063 验收报告 — 审计日志完整（5 类关键操作）

**章程节点**：W3.G.2
**Backlog 任务**：T-0063
**Sub-agent worktree**：`agent-a8883740`
**完成日期**：2026-04-28

---

## 一、章程要求回顾

W3.G.2 要求实现 5 类关键操作的**业务审计**（与 T-0064 字段脱敏不同层级），写入 PG `audit_logs` 表：

1. **login** — 登录成功/失败
2. **config** — 配置变更
3. **upgrade** — 软件升级
4. **reboot** — 设备重启
5. **delete** — 删除操作

Pass 标准：
- 5 类 `ActionXxx` 常量埋点全有调用
- `audit_logs` 表存在

---

## 二、实施摘要

### 2.1 复用既有基础设施

`audit_logs` 表已在 `migrations/000007_system_infra.sql:12-23` 定义，包含 `id / user_id / username / action / resource / resource_id / details(jsonb) / ip_address(inet) / user_agent / created_at` 等字段；W2.D.1 的 `/admin/audit-logs` E2E 已验证 schema 可用。

`internal/admin/{model.go, repository.go, pg_audit_repository.go}` 中 `AuditLog`、`AuditRepository`、`PgAuditRepository` 三件套俱备；登录路径已通过 `auth_handler.recordAuthAuditLog` 写入。

故本次任务**不新增 migration**，仅补：
- 跨模块 audit 包（5 类 ActionXxx 常量 + Sink 抽象 + 全局单例）
- 5 类操作埋点（login 沿用既有路径，其余 4 类新增）
- 单元测试

### 2.2 新增文件

| 文件 | 行数 | 职责 |
|---|---|---|
| `omcgo/internal/admin/audit/audit.go` | 173 | 跨模块 audit 包：5 类 `Action*` 常量、`Entry` / `Sink` 抽象、`SetDefault` 全局单例、`Log` / `LogAsync` API |
| `omcgo/internal/admin/audit/audit_test.go` | 188 | 单元测试：8 用例覆盖常量、单例切换、错误吞咽、异步投递、空 action 丢弃 |
| `omcgo/internal/admin/audit_sink.go` | 64 | `AuditSink` 适配器（`audit.Sink` → `admin.AuditRepository`），失败动作自动后缀 `_failed` |
| `omcgo/internal/admin/audit_helpers.go` | 41 | `AuditContextFromGin` — 从 gin.Context 抽取 user/IP/UA 填充 `audit.Entry` |
| `omcgo/internal/admin/audit_sink_test.go` | 195 | 单元测试：sink 映射、失败后缀、nil 安全、`AuditContextFromGin` 边界、`NewAdminService` 单例自动注册 |

### 2.3 修改文件

| 文件 | 改动要点 |
|---|---|
| `omcgo/internal/admin/handler.go` | `auditActionLoginSuccess` / `LoginFailed` 改为基于 `audit.ActionLogin` 拼接（保留行为兼容、显式引用常量） |
| `omcgo/internal/admin/auth_handler.go` | 注释标记 W3.G.2 ActionLogin / category 1 of 5（不双写） |
| `omcgo/internal/admin/service.go` | `NewAdminService` 构造时自动 `audit.SetDefault(NewAuditSink(auditRepo))` + `SetFallbackLogger`；其他模块由此即可调用 `audit.Log` 而无需 DI |
| `omcgo/internal/device/device_handler.go` | `DeleteDevice` / `BatchDeleteDevices` / `RebootDevice` / `BatchRebootDevices` 4 处埋 `audit.LogAsync` |
| `omcgo/internal/software/handler.go` | `CreateUpgradeTask` 埋 `audit.LogAsync(ActionUpgrade)` |
| `omcgo/internal/config/sync_handler.go` | `PushConfig` 埋 `audit.LogAsync(ActionConfig)` |

### 2.4 不改动的文件

按章程硬约束未触碰：
- `cmd/app/provider/{modules,router}.go`（主会话最终整合）
- `omcgo/internal/core/components/logger/*`、`core/middleware/redact*`（T-0064 sub-agent 工作）
- `omcmb/`
- backlog / charter

`audit_sink` 通过 `NewAdminService` 副作用注册全局单例，**避免**修改 modules.go 的 DI 装配。

---

## 三、设计要点

### 3.1 跨模块解耦

```
internal/admin/audit/      (cross-module abstraction, no admin deps)
  └── audit.go             ActionXxx constants + Sink interface + Log()
                                    ▲
                                    │ Sink
                                    │
internal/admin/             (audit_sink.go)
  └── AuditSink ──────────── implements audit.Sink, wraps admin.AuditRepository
        ▲
        │ SetDefault on construction
        │
NewAdminService(...)
```

device / software / config 仅 import `internal/admin/audit`（接口+常量），不依赖 admin 实现细节。

### 3.2 失败动作编码

`AuditSink.Write` 在 `Success=false` 时把 action 加 `_failed` 后缀（如 `delete` → `delete_failed`）并把 `ErrorMessage` 写入 `details.error`。这样数据库查询无需解析 JSONB 即可区分成败：

```sql
SELECT count(*) FROM audit_logs WHERE action LIKE 'delete%';     -- 全部
SELECT count(*) FROM audit_logs WHERE action = 'delete_failed';  -- 失败
```

### 3.3 异步投递不阻塞业务

`audit.LogAsync` 用全新 `context.Background()` goroutine，避免请求取消时审计也丢失；`Log` 同步版本则用调用者 ctx，由调用者自决。

### 3.4 测试时的副作用隔离

`NewAdminService` 自动注册全局 sink 是侵入性副作用。`audit.SetDefault(nil)` 重置为 noop，测试 cleanup 中调用即可；`audit_sink_test.go::TestNewAdminService_SetsDefaultAuditSink` 演示了完整路径。

---

## 四、Pass 标准核对

### 4.1 5 类 ActionXxx 常量埋点全有调用

```bash
$ for a in ActionLogin ActionConfig ActionUpgrade ActionReboot ActionDelete; do
    printf "%-15s " "$a"
    grep -rn "audit\.$a" internal/ --include="*.go" \
      | grep -v _test.go | grep -v "audit/audit.go" | wc -l
  done
ActionLogin            6
ActionConfig           1
ActionUpgrade          1
ActionReboot           2
ActionDelete           2
```

具体调用点：
- **ActionLogin** × 6: `internal/admin/handler.go` 常量定义 (3 行) + `internal/admin/auth_handler.go` 注释引用 (3 行) — 实际审计写入由 `recordAuthAuditLog` 走 `AuditRepository`，登录成功/失败均覆盖
- **ActionConfig** × 1: `internal/config/sync_handler.go` PushConfig
- **ActionUpgrade** × 1: `internal/software/handler.go` CreateUpgradeTask
- **ActionReboot** × 2: `internal/device/device_handler.go` RebootDevice + BatchRebootDevices
- **ActionDelete** × 2: `internal/device/device_handler.go` DeleteDevice + BatchDeleteDevices

### 4.2 audit_logs 表存在

```bash
$ grep -n "CREATE TABLE audit_logs" omcgo/migrations/*.sql
omcgo/migrations/000007_system_infra.sql:12:CREATE TABLE audit_logs (
```

W2.D.1 `/admin/audit-logs` E2E 已验证数据通路。

---

## 五、自跑验证

### 5.1 编译

```bash
$ go build ./...
（无输出，clean）
```

### 5.2 单元测试

```bash
$ go test -count=1 ./internal/admin/... ./internal/device/... ./internal/software/... ./internal/config/...
ok  github.com/omcgo/omcgo/internal/admin                       1.281s
ok  github.com/omcgo/omcgo/internal/admin/audit                 1.008s
?   github.com/omcgo/omcgo/internal/admin/sqlc                  [no test files]
ok  github.com/omcgo/omcgo/internal/device                      2.283s
ok  github.com/omcgo/omcgo/internal/software                    1.625s
ok  github.com/omcgo/omcgo/internal/config                      3.224s
ok  github.com/omcgo/omcgo/internal/config/baseline             3.904s
ok  github.com/omcgo/omcgo/internal/config/datamodel            2.624s
ok  github.com/omcgo/omcgo/internal/config/template             3.559s
```

### 5.3 -race 测试（仅本次新增）

```bash
$ go test -race -count=1 -run "TestAuditSink|TestAuditContext|TestNewAdminService" ./internal/admin/
ok  github.com/omcgo/omcgo/internal/admin                       1.625s

$ go test -race -count=1 ./internal/admin/audit/...
ok  github.com/omcgo/omcgo/internal/admin/audit                 1.830s
```

新增代码 -race 干净。

### 5.4 已知遗留

`internal/admin/middleware_test.go::TestAuditLogger_WritesOnPost` 在 -race 下报 data race（`mockAuditRepo.Create` 写入字段无锁保护）。**经 git stash 验证：该 race 在本次改动前即存在**，与 T-0063 无关。建议另立任务（如 T-0070+）清理，本次不修。

### 5.5 go vet

```bash
$ go vet ./internal/admin/... ./internal/device/... ./internal/software/... ./internal/config/...
（无输出，clean）
```

---

## 六、待主会话整合

主会话（chenbo01@baicells.com）合并本 worktree 时需检查：

1. **DI 装配**：`audit.SetDefault` 由 `NewAdminService` 副作用注册，`cmd/app/provider/modules.go` 中只要保证 admin module 在 device / software / config 之前初始化即可。
2. **配置开关**：当前 audit 全开关；若需可配置（按 carrier / 按操作类型），可在 `audit.Entry.Action` 处增加白名单过滤。
3. **PII 脱敏**：T-0064 zap logger 字段脱敏与本任务**互不冲突**——T-0063 写 PG `audit_logs`（业务审计层），T-0064 改 zap 字段（运行时日志层）。两者关注点正交。

---

## 七、产出清单

| 类型 | 文件 |
|---|---|
| 新增包 | `omcgo/internal/admin/audit/audit.go` + `audit_test.go` |
| 适配器 | `omcgo/internal/admin/audit_sink.go` + `audit_sink_test.go` |
| Helper | `omcgo/internal/admin/audit_helpers.go` |
| 修改 | `omcgo/internal/admin/handler.go` `service.go` `auth_handler.go` |
| 埋点 | `omcgo/internal/device/device_handler.go` `software/handler.go` `config/sync_handler.go` |
| 文档 | 本文件 |

**不新增** migration（audit_logs 表 W2.D.1 已就位）。

---

DONE.
