# RBAC 提交阻断项修复设计

> 设计日期：2026-07-20
> 当前分支：`codex/fix-system-config-security-p0`
> 关联设计：`2026-07-20-admin-rbac-public-config-design.md`
> 范围：修复启用全局 API RBAC 后发现的 3 个提交阻断问题

## 1. 背景与目标

当前分支恢复了 `/api/v1/admin/**` 的端点级 fail-closed 鉴权，但提交前审查发现：

1. `api_endpoints` 有 669 条，内置角色仅有 43 条授权，普通管理员和操作员会大面积收到 403；
2. 修改角色 API 权限、分配角色或移除角色后，当前实例不会可靠地立即加载最新策略；
3. 请求鉴权与策略刷新并发访问普通 `casbin.Enforcer`，存在数据竞争。

本次修复的目标是：

- 恢复项目历史文档约定的内置角色权限基线；
- 所有影响 Casbin policy 的写操作在返回成功前刷新当前实例；
- 多实例继续通过已有 watcher 广播刷新；
- Casbin 鉴权与策略刷新可以安全并发；
- 对上述行为建立自动化回归测试。

## 2. 已确认的兼容性策略

历史 PRD 与旧 seed 的约定为：

| 内置角色 | API 权限 |
|---|---|
| `admin` | 全部已登记 API |
| `operator` | 全部已登记 API |
| `viewer` | 全部 GET API |

`builtIn` 来源用户仍按已有逻辑旁路 API permission；普通本地用户和 LDAP 用户即使角色名为 `admin`，仍严格执行角色权限。

本次不重新设计 operator/viewer 的最小权限矩阵。重新收敛权限需要按页面和业务动作逐项确认，不应混入提交阻断修复。

## 3. 权限数据修复

新增独立 seed migration：

```text
omcgo/migrations/seed/000002_repair_builtin_role_api_permissions.sql
```

Up 行为：

1. 为 admin 角色插入全部 `api_endpoints`；
2. 为 operator 角色插入全部 `api_endpoints`；
3. 为 viewer 角色插入全部 `method = 'GET'` 的 `api_endpoints`；
4. 所有插入使用 `ON CONFLICT (role_id, endpoint_id) DO NOTHING`，保留已存在授权并保证重复执行安全。

角色使用已有固定 UUID，避免按可修改的角色名称关联。

Down 采用无破坏的空操作，不删除权限。原因是 migration 无法区分“本次补入的授权”和“管理员在 migration 前手工配置的同一授权”；自动删除会破坏真实生产权限数据。该 migration 属于内置数据基线修复，不承担恢复错误安全状态的职责。

本次不修改已生成的 consolidated `000001_init_seed.sql`。新环境会顺序执行 `000001` 和 `000002`，已有环境只执行新增的 `000002`，两种路径得到一致结果。下次重新合并 baseline 时再把最终状态折叠进新的 consolidated seed。

## 4. 策略刷新一致性

### 4.1 统一刷新接口

在角色仓储和 Casbin 实现之间引入最小接口：

```go
type PolicyRefresher interface {
    ReloadPolicy() error
    NotifyPolicyChange() error
}
```

`PgRoleRepository` 依赖该接口，而不是只能接收具体的 `*CasbinAuthorizer`，以便对成功、失败和调用顺序进行单元测试。

统一刷新流程：

```text
数据库事务成功提交
  -> 同步 ReloadPolicy，刷新当前实例
  -> NotifyPolicyChange，通知其他实例
  -> 返回成功
```

覆盖以下三个写入口：

- `SetRoleApiEndpoints`
- `AssignRole`
- `RemoveRole`

### 4.2 错误语义

数据库提交发生在策略刷新之前，因此刷新失败时无法回滚已提交的数据。

处理规则：

- `ReloadPolicy` 失败：返回带上下文的错误，不发送广播；
- `NotifyPolicyChange` 失败：返回带上下文的错误；
- 不再使用 `_ = ...` 静默忽略刷新错误；
- 错误信息明确说明权限数据已提交但运行态刷新失败，便于调用方和日志判断真实状态；
- 不把角色权限明细、用户敏感信息写入错误文本。

在单实例且 watcher bus 未配置时，现有 `NotifyPolicyChange` 空操作仍视为成功，因为当前实例已经同步刷新。多实例部署必须配置现有 watcher；未配置时其他实例仍只能等待周期刷新，这属于部署约束，不在本次引入新的消息系统。

### 4.3 为什么提交后刷新

不能在事务提交前加载 Casbin：Casbin 使用独立查询，看不到未提交数据。先广播再提交也可能让其他实例加载旧 policy。因此只能先提交，再刷新当前实例并广播。

接口返回刷新错误时，客户端重试应保持幂等：

- `SetRoleApiEndpoints` 是目标集合覆盖；
- `AssignRole` 使用现有冲突保护；
- `RemoveRole` 重复删除保持既有语义。

## 5. Casbin 并发安全

将：

```go
*casbin.Enforcer
casbin.NewEnforcer(...)
```

替换为：

```go
*casbin.SyncedEnforcer
casbin.NewSyncedEnforcer(...)
```

保留当前 model、adapter、watcher 和周期刷新机制。`SyncedEnforcer` 为 `Enforce` 提供读锁，为 `LoadPolicy` 提供写锁，并在加载完成后原子替换 policy model，避免请求鉴权观察到刷新中的中间状态。

不在业务层再增加第二把锁，避免锁顺序和重复同步问题。

## 6. 测试策略

严格按 RED → GREEN → REFACTOR 实施。

### 6.1 Seed 权限契约

新增测试验证 migration 的可观察契约：

- admin 覆盖全部 API；
- operator 覆盖全部 API；
- viewer 只覆盖全部 GET API；
- migration 可重复执行；
- 已存在授权不会导致冲突。

优先复用项目已有 PostgreSQL/migration 测试基础设施；如果仓库没有可复用设施，则建立最小 SQL 契约测试，并明确其只能防止 seed 规则回退，不能替代真实数据库 migration 验证。

### 6.2 写入后策略刷新

使用可记录调用的 `PolicyRefresher` 测试：

- 三个写入口提交成功后均依次调用 reload、notify；
- reload 失败时返回错误且不 notify；
- notify 失败时返回错误；
- 数据库操作失败时不 reload、不 notify；
- transaction commit 失败时不 reload、不 notify；
- watcher 未配置的单实例路径可以成功。

### 6.3 并发安全

增加 Casbin 真实实例测试，并用 race detector 并发执行：

- 多个 goroutine 持续 `CheckPermission/Enforce`；
- 另一个 goroutine 重复 `ReloadPolicy/LoadPolicy`；
- 测试不得出现 data race、panic 或部分 policy 状态。

关键验证命令包括：

```bash
cd omcgo
go test ./internal/admin -run 'Test.*(Policy|RoleApi|Casbin)' -count=1
go test -race ./internal/admin -run 'Test.*Casbin.*Concurrent' -count=1
go vet ./...
go build ./...
go test ./...
```

若全量 `-race` 仍触发已知的 AuditLogger 历史竞争，应把“本次新增 Casbin 定向 race 通过”和“既有全量 race 阻断”分别记录，不能混写为全部通过。

## 7. 修改范围

预计修改：

- `omcgo/migrations/seed/000002_repair_builtin_role_api_permissions.sql`
- `omcgo/internal/admin/casbin.go`
- `omcgo/internal/admin/casbin_test.go` 或对应新测试文件
- `omcgo/internal/admin/pg_role_repository.go`
- `omcgo/internal/admin/pg_role_repository_test.go`
- 必要的 migration 契约测试文件
- `docs/review-report/20260720/FIX_71cdbad6_admin-rbac-public-config.md`

不修改：

- 前端角色权限产品矩阵；
- Casbin model 语义；
- builtIn 用户旁路规则；
- 外部 Secret Manager；
- 与本次 3 个阻断项无关的历史 lint、Vitest 和 AuditLogger race 问题；
- 用户工作区中现有的其他未提交修改。

## 8. 完成标准

只有同时满足以下条件，才能把 3 个提交阻断项标记为已解决：

1. 新增 seed migration 能恢复三种内置角色的历史权限基线；
2. 三个 policy 写入口都在成功返回前完成当前实例刷新和广播；
3. Casbin 使用 `SyncedEnforcer`；
4. 新增定向测试和 Casbin 定向 race 测试通过；
5. 后端 `go vet ./...`、`go build ./...`、`go test ./...` 通过；
6. 审查报告准确列出每项修改、测试证据及仍未解决的非本次问题；
7. 未暂存、未提交、未推送，等待用户进行提交前评审。
