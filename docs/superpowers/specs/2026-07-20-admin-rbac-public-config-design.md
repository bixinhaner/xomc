# 管理接口 RBAC 与公共配置白名单修复设计

> 设计日期：2026-07-20
> 源码基线：`cccbb2b3bca9b8c4267e140856d4a2a65881d356`
> 必须完成范围：完整关闭管理接口未授权访问与通用配置敏感信息暴露两个 P0
> 暂缓范围：数据库静态加密、外部 Secret Manager

## 1. 目标

本次修复完整解决两条直接暴露链路：

1. `/api/v1/admin/**` 当前只要求登录，未执行已有的端点权限；
2. 匿名公共配置接口信任数据库 `is_public=true`，错误标记或恶意修改可使任意配置公开。

完成后：

- 受保护接口按现有 `role_api_permissions` 的 `(path, method)` 规则鉴权；
- 超级管理员继续旁路；
- 未授权请求返回 403，权限检查异常返回 500，不得 fail-open；
- 公共配置响应只包含代码明确列出的 `(category,key)`；
- 数据库中非白名单记录即使 `is_public=true`，也不会被匿名接口返回。
- 客户端不能再决定配置是否公开；
- 通用配置读取永不返回默认密码和 Agent Token 原文；
- 默认密码采用 write-only 的“留空不修改”语义；
- Agent Token 只能通过已有专用 API 修改，通用配置写接口拒绝该 key。

## 2. 当前根因

### 2.1 权限链被短路

项目已有以下基础设施：

- `role_api_permissions` 保存角色允许访问的 API endpoint；
- `api_endpoints` 保存标准化路径与 HTTP method；
- `CasbinAuthorizer.CheckPermission` 按 `(path,method)` 执行策略；
- JWT 上下文包含 user ID、roles 和 super-admin 标记。

但 `RequireAPIPermission` 目前只验证 user ID 是否存在，然后直接 `Next()`。路由注释、Casbin 注释与运行行为互相矛盾。

### 2.2 公共读取信任数据库标志

`SysConfigService.ListPublic` 当前允许：

```text
item.IsPublic == true
OR
security.isBrowserAutoRecordPass
```

因此任何能把记录改成 `is_public=true` 的调用方，都能使该记录进入匿名接口。

## 3. 方案决策

### 3.1 恢复全局端点级 RBAC

修改现有 `RequireAPIPermission`，不新增第二套权限模型。

请求流程：

```text
RequireAuth
  -> 检查 user ID 类型
  -> super-admin 直接通过
  -> roleRepo.CheckPermission(userID, c.FullPath(), request.Method)
     -> allowed: 继续
     -> denied: 403
     -> error: 记录 user/path/method/error，返回 500
```

约束：

- 使用 Gin 注册后的标准路径，例如 `/api/v1/admin/sysConfig/:id`，不能用包含实际 UUID 的 URL；
- HTTP method 原样使用大写值；
- 没有权限记录必须拒绝，不能按“初始化模式”放行；
- repository/checker 为 nil 或未配置必须返回 500，不能绕过；
- 只有 JWT 上下文中的 `isSuperAdmin=true` 可以旁路，普通名称为 `admin` 的角色仍以策略为准；
- public、login、health 等没有挂载该中间件的路由不受影响。

现有未使用的 `RequireApiPermission` 采用多处 fail-open 规则，容易被误用。本次删除该死代码及仅为它存在的接口/类型，统一使用 `RequireAPIPermission`。

### 3.2 公共配置采用唯一代码白名单

建立集中函数：

```go
func isPublicSysConfig(category, key string) bool
```

白名单与当前登录前页面的真实消费者一致：

| Category | Key | 用途 |
|---|---|---|
| `system` | `system_name` | 系统名称 |
| `system` | `system_version` | 系统版本 |
| `system` | `show_menu_icon` | 菜单图标开关 |
| `basic` | `mrOMCName` | OMC 品牌名称 |
| `security` | `isBrowserAutoRecordPass` | 登录页浏览器记密行为 |
| `ui_custom` | `ui_login_background` | 登录背景 |
| `ui_custom` | `ui_menu_logo_up` | 折叠菜单 Logo |
| `ui_custom` | `ui_menu_logo_down` | 展开菜单 Logo |

行为：

- `ListPublic` 读取 category 后只按 `isPublicSysConfig` 过滤；
- `item.IsPublic` 不再参与是否可以匿名返回的判断；
- `BatchUpsert` 首次创建和后续保存时复用同一白名单计算 `is_public`；
- 非白名单记录历史上即使被错误标记为 public，匿名端点也立即不可见；
- 不修改数据库 schema，也不要求先清洗历史数据。

### 3.3 禁止客户端控制 `is_public`

通用配置 Create/Update 的请求兼容字段暂时保留为 `*bool`，但只要客户端显式提交就返回 400。保留字段是为了给旧客户端明确错误，而不是静默忽略未知 JSON。

服务端行为：

- Create 根据 `isPublicSysConfig(category,key)` 决定 `IsPublic`；
- Update 根据记录原有的 category/key 重新计算 `IsPublic`；
- BatchUpsert 对冲突记录执行 `is_public = EXCLUDED.is_public`，修正历史错误标记；
- 所有公开属性都由服务端分类器产生，客户端没有覆盖入口。

### 3.4 通用响应使用安全 DTO

内部 `SysConfig` 继续保存真实值，供受控的运行态 consumer 使用；HTTP handler 不再直接序列化 domain model。

新增响应 DTO：

```go
type SysConfigResponse struct {
    ID           uuid.UUID `json:"id"`
    Category     string    `json:"category"`
    Key          string    `json:"key"`
    Value        string    `json:"value"`
    ValueType    string    `json:"value_type"`
    Description  string    `json:"desc"`
    IsPublic     bool      `json:"is_public"`
    IsSecret     bool      `json:"is_secret"`
    IsConfigured bool      `json:"is_configured"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

敏感键集中登记：

| Category | Key | 通用读取 | 通用写入 |
|---|---|---|---|
| `security` | `defaultPasswd` | `value=""`、返回 configured 状态 | 仅允许 batch 写入非空新值 |
| `agent` | `agent_studio_service_token` | `value=""`、返回 configured 状态 | 拒绝，必须走 Agent 专用 API |

所有 List/Get/Create/Update HTTP 响应都经过同一个转换函数；不能只修 List 而让 `GET /:id` 或写入响应继续泄露。

`is_configured` 只表示存储值非空，不返回长度、前后缀或掩码，避免侧信道泄露。

### 3.5 Secret 写入边界

新增“通用 HTTP 写入策略”，与内部 service 写入分开：

- direct Create/Update 遇到敏感键返回 400，避免绕过 validator/hook；
- generic BatchUpdate 遇到 Agent Token 返回 400；
- generic BatchUpdate 的 `security.defaultPasswd` 只接受非空值，并继续执行现有密码强度 validator；
- Agent 专用 service 仍可在进程内部调用 `SysConfigService.BatchUpsert` 保存 Token；
- 任何 handler 错误、审计日志和业务日志都不能包含 secret value。

这样不会破坏 Agent 的专用 `serviceTokenConfigured` 契约，也不会让通用 API 成为第二条 Token 写通道。

### 3.6 前端默认密码改为 write-only

前端类型增加：

```ts
isSecret?: boolean;
isConfigured?: boolean;
```

安全页行为：

- 删除 `defaultPasswd: 'OMC@123456'` 的前端初始值；
- 后端返回 `isSecret=true` 时不把空响应灌入表单；
- 输入框始终为空，不显示原值和固定掩码；
- 已配置时提示“已配置，留空保持不变”，未配置时提示“未配置，请输入新默认密码”；
- `buildBatchItems` 对“已有 secret + 表单空值”不生成 item；
- 用户输入非空新密码时才生成 `security.defaultPasswd` 的 batch item；
- 保存其他安全策略不会覆盖默认密码。

## 4. 逐项修改清单

| ID | 层次 | 修改 | 主要文件 |
|---|---|---|---|
| M-01 | Middleware | `RequireAPIPermission` 恢复 `(FullPath,Method)` Casbin 检查并 fail-closed | `omcgo/internal/admin/middleware.go` |
| M-02 | Cleanup | 删除未使用且 fail-open 的 `RequireApiPermission`、`ApiPermissionChecker`、`RoleApiEndpoint` | `omcgo/internal/admin/middleware.go` |
| M-03 | Wiring | 修正 `/admin`、system info、DLQ 路由注释，明确端点 RBAC 已启用 | `omcgo/cmd/app/provider/router.go` |
| M-04 | Public policy | 用集中白名单替代数据库 `is_public` 判断 | `omcgo/internal/admin/sys_config.go` |
| M-05 | Persistence | Create/Update/Batch 的 public 标记全部由服务端白名单派生 | `omcgo/internal/admin/sys_config.go` |
| M-06 | Input guard | 客户端显式提交 `is_public` 返回 400 | `omcgo/internal/admin/sys_config.go`、`sys_config_handler.go` |
| M-07 | Secret registry | 登记默认密码和 Agent Token 两个敏感键 | `omcgo/internal/admin/sys_config.go` |
| M-08 | Safe DTO | List/Get/Create/Update 统一返回脱敏 DTO | `omcgo/internal/admin/sys_config_handler.go` |
| M-09 | Write guard | direct CRUD 拒绝 secret；generic batch 拒绝 Agent Token、仅接受非空默认密码 | `omcgo/internal/admin/sys_config_handler.go`、`sys_config.go` |
| M-10 | FE contract | 前端映射 `isSecret/isConfigured` | `omcmb/frontend-core/src/types/system.ts`、`services/api/adminApi.ts` |
| M-11 | FE form | 默认密码输入框改为 write-only，删除硬编码初始密码 | `omcmb/webcode/src/pages/system/SystemConfig/SecuritySettings.tsx`、`index.tsx` |
| M-12 | FE serializer | 已配置 secret 留空时不生成 batch item | `omcmb/webcode/src/pages/system/SystemConfig/sysConfigSerialize.ts` |
| M-13 | Tests | 新增 RBAC、public allowlist、secret 脱敏、写入守卫、前端不覆盖测试 | 对应 Go/TS test 文件 |
| M-14 | Documentation | 更新测试报告中两个 P0 的状态和剩余风险 | `docs/review-report/20260720/` |

## 5. 兼容性与剩余风险

### 5.1 RBAC 兼容性

恢复鉴权后，没有 endpoint permission 的历史角色会收到 403。这是预期的 fail-closed 行为。

部署前检查：

- 内置 admin/operator/viewer 的 endpoint permissions 是否完整；
- 新增但未同步到 `api_endpoints` 的路由；
- 页面进入时调用的 `/admin` GET 是否已授给对应角色；
- worker/system 内部调用是否使用 super-admin 或独立内部认证。

### 5.2 完成本次修复后仍存在的风险

以下存储层强化不属于当前两个 API P0 的关闭条件：

- `sys_configs.value` 仍是数据库明文，拥有数据库读取权限的人仍能看到 secret；
- Secret 尚未迁移至外部 Secret Manager；
- 数据库备份中的历史 secret 尚未轮换或重加密。

这些风险必须单列后续安全任务，不能与本次 HTTP 授权和响应脱敏混为一谈。

## 6. 错误处理与审计

- 未认证：401；
- 认证上下文 user ID 类型错误：500；
- Casbin/数据库检查失败：记录结构化错误并返回 500；
- 权限不足或无匹配策略：403；
- 公共白名单之外的配置：不返回，不暴露记录是否存在；
- 客户端控制 `is_public`、通用接口写 Agent Token、direct CRUD 写 secret：400；
- 不在响应错误中包含数据库、Casbin 策略或配置值细节。

写请求继续由现有 Audit/Operation Log 中间件记录。测试必须确认审计内容不包含请求中的 secret value。

## 7. 测试设计

### 7.1 RBAC 中间件

先写失败测试并确认 RED：

1. 有 endpoint permission 的普通角色返回 200；
2. 无 endpoint permission 返回 403；
3. super-admin 不调用 checker 并返回 200；
4. checker 返回错误时返回 500；
5. 缺少认证上下文返回 401；
6. 无效 user ID 上下文返回 500；
7. checker 收到 Gin 标准路径和实际 HTTP method。

### 7.2 公共配置

先写失败测试并确认 RED：

1. 白名单记录即使 `IsPublic=false` 仍可公开；
2. 非白名单记录即使 `IsPublic=true` 也不得公开；
3. `security.defaultPasswd` 永不进入匿名响应；
4. `agent.agent_studio_service_token` 永不进入匿名响应；
5. category 过滤不会扩大白名单；
6. 8 个既有公开键全部被分类为 public；
7. 相同 key 位于错误 category 时不公开。

### 7.3 Secret 与前端保存

先写失败测试并确认 RED：

1. List/Get/Create/Update 响应均不返回 secret 原文；
2. configured 状态按存储值是否非空计算；
3. 显式提交 `is_public` 返回 400；
4. generic batch 写 Agent Token 返回 400；
5. generic batch 写空默认密码返回 400；
6. generic batch 写合法默认密码进入现有 validator；
7. 前端已配置 secret 留空时不生成 batch item；
8. 前端输入新默认密码时正常生成 batch item；
9. 保存其他安全字段不覆盖默认密码。

### 7.4 回归

- `go test ./internal/admin ./cmd/app/provider`;
- `go test ./...`;
- 前端 `npm run typecheck`；
- SystemConfig serializer/component 定向 Vitest；
- Security Playwright；
- 真实后端 smoke 在部署后验证 admin 页面继续可用；
- 使用非管理员测试账号验证允许的 GET 成功、未授权写入返回 403。

## 8. 验收标准

1. `RequireAPIPermission` 对非 super-admin 的每个受保护请求调用一次 endpoint permission checker；
2. checker 错误和空权限均不得放行；
3. viewer/operator/admin 的结果与 `role_api_permissions` 一致；
4. 手工把 `security.defaultPasswd.is_public` 改为 true，匿名接口仍不返回该项；
5. 8 个白名单项继续支持登录前页面；
6. 任意通用配置读取路径均不返回默认密码或 Agent Token 原文；
7. 客户端无法控制 `is_public`；
8. 保存其他安全设置不会创建、清空或覆盖默认密码；
9. Agent Token 只能通过 Agent 专用 API 修改；
10. 所有新增单元测试、后端全量测试、前端 typecheck 和定向前端测试通过；
11. 审查报告将两个 P0 标记为已关闭，并单列数据库明文存储风险。
