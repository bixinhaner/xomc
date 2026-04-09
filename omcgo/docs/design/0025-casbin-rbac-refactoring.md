# 系统管理模块 Casbin RBAC 改造方案

> 版本: v1.0
> 日期: 2026-04-09
> 状态: 设计中
> 关联文档: `docs/rbac_complete_design.md`, `docs/rbac_implementation_plan.md`, `files/Back-end/系统管理模块-开发设计方案.md`

---

## 一、现状分析

### 1.1 当前权限架构

```
请求 → RequireAuthWithAPIKey → RequireResourcePermission
                                      │
                                      ▼
                        CheckPermission(userID, resource, action)
                                      │
                                      ▼
                        SQL: SELECT COUNT(*) FROM sys_permissions p
                             JOIN user_roles ur ON ur.role_id = p.role_id
                             WHERE ur.user_id = $1 AND p.resource = $2 AND p.action = $3
                                      │
                                      ▼
                        每次请求都查 PostgreSQL（无缓存）
```

### 1.2 当前实现的六个问题

| # | 问题 | 影响 |
|---|------|------|
| P1 | **每次请求查 DB** | `CheckPermission` 对每个受保护 API 执行 SQL COUNT 查询，高并发下 DB 压力大 |
| P2 | **无角色继承** | viewer/operator/admin 各自独立列举权限，无法继承 |
| P3 | **无通配符** | 不支持 `devices:*` 表示设备资源的全部操作 |
| P4 | **无拒绝规则** | 只有 allow 规则，无法显式禁止某角色访问特定资源 |
| P5 | **无策略缓存** | 中间件层零缓存，数据权限有 Redis 缓存但功能权限没有 |
| P6 | **扩展性差** | 新增权限维度（如 ABAC 属性判断）需大量改代码 |

### 1.3 不需要改的部分

以下部分与 Casbin 无关，保持不变：

| 组件 | 说明 |
|------|------|
| JWT 认证 | `RequireAuthWithAPIKey`、JWT Claims 结构、token 签发 |
| API Key 认证 | `APIKeyService`、X-API-Key 头处理 |
| 数据权限 | `PermissionService.GetUserVisibleGroupIDs`（设备分组可见范围） |
| 运营商过滤 | `RequireCarrier` 中间件（carrier=nil 为超管） |
| 审计日志 | `AuditLogger` 中间件 |
| 角色/权限 CRUD | `AdminService` 的 CreateRole/UpdateRole/DeleteRole |
| 路由注册 | `permGroup(resource)` 模式 |

---

## 二、Casbin 技术方案

### 2.1 为什么选 Casbin

| 维度 | 手写 SQL | Casbin |
|------|---------|--------|
| 策略评估 | 每次查 DB | 内存评估（微秒级） |
| 角色继承 | 不支持 | 原生支持 |
| 通配符 | 不支持 | `*` 匹配 |
| 拒绝规则 | 不支持 | priority + deny |
| 多模型 | 仅 RBAC | RBAC/ABAC/ACL 可切换 |
| 策略变更 | 无通知机制 | Watcher 模式（Redis/NATS） |
| 跨实例同步 | 无 | 内置支持 |

### 2.2 依赖引入

```
github.com/casbin/casbin/v2                    # 核心引擎
github.com/casbin/casbin/v2/model              # 模型定义
github.com/casbin/ent-adapter                  # PostgreSQL 适配器（基于 ent ORM）
```

> **注意**：不使用 `xorm-adapter` 或 `gorm-adapter`，因为项目不使用 ORM。采用自定义 Adapter 复用现有 `sys_permissions` + `user_roles` 表。

### 2.3 Casbin 模型定义

创建文件 `configs/casbin_model.conf`：

```ini
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && (r.obj == p.obj || keyMatch(r.obj, p.obj)) && (r.act == p.act || p.act == "*")
```

**模型说明：**

| 元素 | 含义 | 示例 |
|------|------|------|
| `sub` | 主体（角色名） | `role:admin` |
| `dom` | 域（运营商代码或 `system`） | `cmcc`, `ctcc`, `system` |
| `obj` | 对象（资源） | `devices`, `datamodels` |
| `act` | 动作 | `read`, `write`, `delete`, `*` |
| `g` | 角色继承关系（含域） | `g, role:operator, role:viewer, cmcc` |

**与现有数据的映射：**

```
现有 sys_permissions 表: (role_id, resource, action)
  ↓ 通过角色名转换
Casbin 策略:        (role:{name}, {carrier|system}, {resource}, {action})

现有 user_roles 表:  (user_id, role_id)
  ↓ 通过用户ID和角色名转换
Casbin 角色分配:     g, {user_id}, role:{name}, {carrier|system}
```

### 2.4 自定义 PostgreSQL Adapter

不复用 ent/xorm/gorm 适配器，而是实现 `persist.Adapter` 接口直接读取现有表：

```go
// internal/admin/casbin_adapter.go

type pgAdapter struct {
    pool *pgxpool.Pool
}

func newPgAdapter(pool *pgxpool.Pool) *pgAdapter { ... }

// LoadPolicy 从现有表加载全部策略到 Casbin
func (a *pgAdapter) LoadPolicy(model model.Model) error {
    // 1. 加载角色继承: g = (_, _, _)
    //    SELECT r.name AS role_name, u.id::text AS user_id,
    //           COALESCE(u.carrier, 'system') AS domain
    //    FROM user_roles ur
    //    JOIN roles r ON r.id = ur.role_id
    //    JOIN users u ON u.id = ur.user_id

    // 2. 加载权限策略: p = (sub, dom, obj, act)
    //    SELECT r.name AS role_name,
    //           COALESCE(u.carrier, 'system') AS domain,
    //           p.resource, p.action
    //    FROM sys_permissions p
    //    JOIN roles r ON r.id = p.role_id
    //    JOIN user_roles ur ON ur.role_id = r.id
    //    JOIN users u ON u.id = ur.user_id
    //    GROUP BY r.name, u.carrier, p.resource, p.action
}

// SavePolicy 由 Watcher 触发增量更新，非必须实现
func (a *pgAdapter) SavePolicy(model model.Model) error { return nil }

// AddPolicy / RemovePolicy — 增量策略变更
func (a *pgAdapter) AddPolicy(sec string, ptype string, rule []string) error { ... }
func (a *pgAdapter) RemovePolicy(sec string, ptype string, rule []string) error { ... }
```

### 2.5 策略变更通知（Watcher）

利用现有 Redis 基础设施实现跨实例策略同步：

```go
// internal/admin/casbin_watcher.go

type redisWatcher struct {
    client redis.UniversalClient
    pubSub *redis.PubSub
    callback func(string)
}

const casbinPolicyChannel = "casbin:policy:reload"

func newRedisWatcher(client redis.UniversalClient) *redisWatcher { ... }

// UpdateCallback 设置回调函数
func (w *redisWatcher) SetUpdateCallback(callback func(string)) { ... }

// StartListener 监听策略变更通知
func (w *redisWatcher) StartListener() {
    // SUBSCRIBE casbin:policy:reload
    // 收到消息后调用 callback → 触发 enforcer.LoadPolicy()
}
```

**触发时机**：角色权限变更时（`AddPermissions`、`RemoveAllPermissions`、`AssignRole`、`RemoveRole`）发布消息。

---

## 三、代码改造

### 3.1 新增文件

| 文件 | 职责 |
|------|------|
| `configs/casbin_model.conf` | Casbin 模型定义 |
| `internal/admin/casbin_adapter.go` | PostgreSQL 自定义适配器 |
| `internal/admin/casbin_watcher.go` | Redis Watcher 跨实例同步 |
| `internal/admin/casbin_enforcer.go` | Enforcer 封装，提供 `Enforce` 方法 |

### 3.2 Enforcer 封装

```go
// internal/admin/casbin_enforcer.go

package admin

import (
    "github.com/casbin/casbin/v2"
    "github.com/casbin/casbin/v2/model"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
)

// CasbinAuthorizer 基于 Casbin 的权限校验器，替代 PermissionChecker 接口的 SQL 实现。
type CasbinAuthorizer struct {
    enforcer *casbin.Enforcer
    adapter  *pgAdapter
    watcher  *redisWatcher
}

// NewCasbinAuthorizer 创建并初始化 Casbin 权限校验器。
func NewCasbinAuthorizer(pool *pgxpool.Pool, redisClient redis.UniversalClient, modelPath string) (*CasbinAuthorizer, error) {
    // 1. 加载模型
    m, err := model.NewModelFromFile(modelPath)

    // 2. 创建适配器
    adapter := newPgAdapter(pool)

    // 3. 创建 Enforcer
    enforcer, err := casbin.NewEnforcer(m, adapter)

    // 4. 创建 Watcher
    watcher := newRedisWatcher(redisClient)
    enforcer.SetWatcher(watcher)

    // 5. 加载策略
    err = enforcer.LoadPolicy()

    return &CasbinAuthorizer{enforcer: enforcer, adapter: adapter, watcher: watcher}, nil
}

// CheckPermission 检查用户是否有权限（实现 PermissionChecker 接口）。
func (a *CasbinAuthorizer) CheckPermission(userID string, domain string, resource string, action string) bool {
    ok, _ := a.enforcer.Enforce(userID, domain, resource, action)
    return ok
}

// ReloadPolicy 手动触发策略重载。
func (a *CasbinAuthorizer) ReloadPolicy() error {
    return a.enforcer.LoadPolicy()
}

// NotifyPolicyChange 通知所有实例重新加载策略。
func (a *CasbinAuthorizer) NotifyPolicyChange() error {
    return a.watcher.Notify()
}

// Stop 关闭 watcher。
func (a *CasbinAuthorizer) Stop() {
    a.watcher.Close()
}
```

### 3.3 Repository 层变更

**`pg_role_repository.go` — `CheckPermission` 改为调用 Casbin：**

```go
// 改造前：SQL 查询
func (r *PgRoleRepository) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
    var count int
    err := r.pool.QueryRow(ctx,
        `SELECT COUNT(*) FROM sys_permissions p JOIN user_roles ur ON ur.role_id = p.role_id
         WHERE ur.user_id = $1 AND p.resource = $2 AND p.action = $3`,
        userID, resource, action,
    ).Scan(&count)
    return count > 0, err
}

// 改造后：Casbin 内存评估
func (r *PgRoleRepository) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
    // 从 context 获取 domain（carrier 或 "system"）
    domain := getDomainFromContext(ctx)
    return r.authorizer.CheckPermission(userID.String(), domain, resource, action), nil
}
```

### 3.4 Middleware 层变更

**`middleware.go` — `RequirePermission` 和 `RequireResourcePermission` 传递 domain：**

```go
// RequireResourcePermission — 改造前后对比
// 改造前：
func RequireResourcePermission(roleRepo admin.RoleRepository, resource string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetString(string(admin.CtxKeyUserID))
        action := methodToAction(c.Request.Method)
        ok, err := roleRepo.CheckPermission(c.Request.Context(), uuid.MustParse(userID), resource, action)
        // ...
    }
}

// 改造后：domain 从 context 自动获取（由 RequireCarrier 中间件已设置）
// 唯一变化是 CheckPermission 内部从 SQL 变为 Casbin，接口签名不变
// 中间件代码零改动
```

**关键点**：由于 `CheckPermission` 的**接口签名不变**，中间件代码**无需修改**。

### 3.5 策略变更触发

在 `pg_role_repository.go` 的写操作中增加通知：

```go
// AddPermissions — 增加权限后通知
func (r *PgRoleRepository) AddPermissions(ctx context.Context, roleID uuid.UUID, perms []admin.Permission) error {
    // ... 现有 INSERT 逻辑不变 ...

    // 通知所有实例重新加载策略
    return r.authorizer.NotifyPolicyChange()
}

// RemoveAllPermissions — 删除权限后通知
func (r *PgRoleRepository) RemoveAllPermissions(ctx context.Context, roleID uuid.UUID) error {
    // ... 现有 DELETE 逻辑不变 ...
    return r.authorizer.NotifyPolicyChange()
}

// AssignRole — 分配角色后通知
func (r *PgRoleRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
    // ... 现有 INSERT 逻辑不变 ...
    return r.authorizer.NotifyPolicyChange()
}

// RemoveRole — 移除角色后通知
func (r *PgRoleRepository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
    // ... 现有 DELETE 逻辑不变 ...
    return r.authorizer.NotifyPolicyChange()
}
```

### 3.6 DI 注册

**`cmd/app/provider/admin.go` — 初始化 CasbinAuthorizer：**

```go
func initAdminModule(c *Container) error {
    // ... 现有代码 ...

    // 新增：初始化 Casbin 权限引擎
    authorizer, err := admin.NewCasbinAuthorizer(c.PgPool, c.Redis, "configs/casbin_model.conf")
    if err != nil {
        return fmt.Errorf("init casbin authorizer: %w", err)
    }
    c.GS.Register("casbin-watcher", 1, func(ctx context.Context) error { authorizer.Stop(); return nil })

    // 将 authorizer 注入到 RoleRepository
    roleRepo := admin.NewPgRoleRepository(c.PgPool)
    roleRepo.SetAuthorizer(authorizer)  // 新增方法

    // ...
}
```

---

## 四、角色继承设计

### 4.1 继承链

```
role:admin (cmcc)
    └── role:operator (cmcc)
          └── role:viewer (cmcc)

role:admin (ctcc)
    └── role:operator (ctcc)
          └── role:viewer (ctcc)
```

### 4.2 数据库变更

新增 `role_inheritance` 表：

```sql
-- 0000xx_add_role_inheritance.up.sql

CREATE TABLE role_inheritance (
    parent_role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    child_role_id  UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    domain         VARCHAR(16) NOT NULL DEFAULT 'system',
    PRIMARY KEY (parent_role_id, child_role_id, domain)
);

-- 种子数据：admin 继承 operator，operator 继承 viewer
INSERT INTO role_inheritance (parent_role_id, child_role_id, domain)
SELECT p.id, c.id, 'system'
FROM roles p, roles c
WHERE p.name = 'admin' AND c.name = 'operator';

INSERT INTO role_inheritance (parent_role_id, child_role_id, domain)
SELECT p.id, c.id, 'system'
FROM roles p, roles c
WHERE p.name = 'operator' AND c.name = 'viewer';
```

### 4.3 Casbin Adapter 加载继承

```go
func (a *pgAdapter) LoadPolicy(model model.Model) error {
    // ... 加载 p 策略（权限）...

    // 加载 g 策略（角色继承）
    rows, _ := a.pool.Query(ctx, `
        SELECT rp.name AS parent, rc.name AS child, ri.domain
        FROM role_inheritance ri
        JOIN roles rp ON rp.id = ri.parent_role_id
        JOIN roles rc ON rc.id = ri.child_role_id
    `)
    for rows.Next() {
        rows.Scan(&parent, &child, &domain)
        model.AddPolicy("g", "g", []string{"role:" + child, "role:" + parent, domain})
    }

    return nil
}
```

### 4.4 权限简化

角色继承建立后，种子权限可大幅简化：

| 角色 | 需要显式声明的权限 | 继承获得的权限 |
|------|-------------------|---------------|
| `viewer` | 8 resources × read | — |
| `operator` | 8 resources × write | viewer 的全部 read |
| `admin` | 10 resources × delete + admin | operator 的全部 read + write |

---

## 五、与多角色切换的整合

第九章需求中的"角色切换"与 Casbin 的整合：

### 5.1 切换后的权限评估

```go
// 用户切换角色后，JWT Claims 中的 current_role_id 变更
// CheckPermission 改为基于 current_role_id 而非 user_id

func (r *PgRoleRepository) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
    // 从 JWT Claims 获取当前角色（而非遍历用户全部角色）
    currentRole := getContextCurrentRole(ctx)
    domain := getDomainFromContext(ctx)

    // 直接以角色名作为 sub（而非用户ID）
    ok, _ := r.authorizer.enforcer.Enforce("role:"+currentRole, domain, resource, action)
    return ok, nil
}
```

### 5.2 角色切换后无需重载策略

Casbin 策略是全局加载的，角色切换只改变 `sub` 参数，不需要重新加载策略。只需前端刷新菜单即可。

---

## 六、目录结构

```
internal/admin/
├── model.go                     # 数据模型（不变）
├── repository.go                # 接口定义（不变）
├── pg_user_repository.go        # 用户仓储（不变）
├── pg_role_repository.go        # 角色仓储（改动：CheckPermission 用 Casbin）
├── pg_menu_repository.go        # 菜单仓储（不变）
├── pg_audit_repository.go       # 审计仓储（不变）
├── casbin_enforcer.go           # [新增] Casbin 权限引擎封装
├── casbin_adapter.go            # [新增] PostgreSQL 自定义适配器
├── casbin_watcher.go            # [新增] Redis Watcher 策略同步
├── service.go                   # 业务逻辑（微调：注入 authorizer）
├── permission_service.go        # 数据权限（不变）
├── handler.go                   # HTTP 处理器（不变）
├── middleware.go                 # 中间件（不变）
├── jwt.go                       # JWT（不变）
├── bruteforce.go                # 防暴力破解（不变）
├── captcha.go                   # 验证码（不变）
├── apikey_*.go                  # API Key（不变）

configs/
├── casbin_model.conf            # [新增] Casbin RBAC 模型定义

migrations/
├── 0000xx_add_role_inheritance.up.sql   # [新增] 角色继承表
```

---

## 七、实施计划

### Phase 1: Casbin 集成基础（2 天）

| 任务 | 说明 |
|------|------|
| 引入依赖 | `go get github.com/casbin/casbin/v2` |
| 编写 `casbin_model.conf` | RBAC with domains 模型 |
| 实现 `pgAdapter` | 从现有 `sys_permissions` + `user_roles` 加载策略 |
| 实现 `CasbinAuthorizer` | 封装 Enforcer，提供 `CheckPermission` |
| DI 注册 | `provider/admin.go` 初始化并注入 |
| 验证 | `CheckPermission` 接口签名不变，跑通全量测试 |

### Phase 2: Watcher + 策略热更新（1 天）

| 任务 | 说明 |
|------|------|
| 实现 `redisWatcher` | Redis Pub/Sub 通知策略变更 |
| 写操作触发通知 | `AddPermissions`/`AssignRole` 等操作后 `NotifyPolicyChange` |
| 验证 | 多实例策略同步测试 |

### Phase 3: 角色继承（1 天）

| 任务 | 说明 |
|------|------|
| 数据库迁移 | `role_inheritance` 表 + 种子数据 |
| Adapter 加载继承 | `LoadPolicy` 增加继承策略加载 |
| 简化种子权限 | viewer 只需声明 read，operator/admin 通过继承获得 |
| 前端角色管理 | 增加"继承自"下拉框 |

### Phase 4: 多角色切换整合（2 天）

| 任务 | 说明 |
|------|------|
| JWT Claims 扩展 | 增加 `current_role_id` |
| `CheckPermission` 改为角色维度 | `sub` 从 userID 改为 `role:{name}` |
| `switch-role` API | 切换角色重签 JWT |
| 前端角色切换 UI | 个人中心下拉 + 菜单重新加载 |

### Phase 5: 测试与验证（1 天）

| 任务 | 说明 |
|------|------|
| 单元测试 | `casbin_enforcer_test.go`：策略加载、权限检查、继承 |
| 集成测试 | E2E 验证全量 API 权限 |
| 性能对比 | Casbin 内存评估 vs 原 SQL 查询延迟对比 |
| 压测 | 并发 5000 设备在线场景下权限检查吞吐量 |

---

## 八、风险与缓解

| 风险 | 缓解措施 |
|------|---------|
| Casbin 内存占用 | 10 万用户 × 平均 2 角色 × 10 权限 = 200 万条策略，约 200MB 内存，可接受 |
| 策略加载延迟 | 启动时一次性加载（约 1-2 秒），运行时增量更新 |
| Casbin 依赖锁定 | Casbin 是 CNCF 沙箱项目，社区活跃，v2 API 稳定 |
| Adapter 与现有表不兼容 | 自定义 Adapter 直接映射现有表结构，无需迁移数据 |
| Watcher 通知丢失 | 启动时全量加载 + 定时兜底刷新（5 分钟间隔） |

---

## 九、验收标准

| 标准 | 验证方式 |
|------|---------|
| 所有 API 权限检查行为不变 | E2E 全量 452 用例通过 |
| `CheckPermission` 不再查询 DB | 日志/指标确认无 SQL 执行 |
| 角色继承生效 | operator 自动拥有 viewer 的全部 read 权限 |
| 策略变更实时生效 | 修改角色权限后，立即对新请求生效（< 1 秒） |
| 多实例同步 | 两个 app 实例，A 实例改权限，B 实例立即生效 |
| 单次权限检查延迟 < 1ms | Prometheus 指标 P99 < 1ms |
