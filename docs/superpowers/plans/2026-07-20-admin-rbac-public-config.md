# 管理接口 RBAC 与公共配置安全修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完整关闭 `/admin` 仅认证未鉴权，以及通用系统配置可公开/回显敏感值两个 P0。

**Architecture:** 复用现有 `role_api_permissions + Casbin` 恢复基于 Gin 标准路径和 HTTP method 的 fail-closed 鉴权；用集中配置策略定义 public/secret 分类，HTTP 层统一输出脱敏 DTO 并限制通用写入；前端把默认密码改成 write-only 字段，留空时不提交。

**Tech Stack:** Go、Gin、Casbin、pgx、React、TypeScript、Ant Design、Vitest、Playwright。

## Global Constraints

- 不新增第二套 RBAC 模型，必须复用现有 endpoint permission。
- 权限检查失败、无权限、checker 未配置均不得 fail-open。
- Public 与 Secret 分类必须是代码白名单，不能依赖客户端输入。
- 通用 HTTP 响应不得出现默认密码或 Agent Token 原文。
- Agent 专用 API 的 Token 保存能力必须保持兼容。
- 不修改数据库 schema，不迁移 Secret Manager。
- 严格按 RED → GREEN → REFACTOR 执行，每个生产代码改动前先看到对应测试正确失败。
- 不提交、不推送，除非用户另行明确授权。

---

### Task 1: 恢复端点级 RBAC

**Files:**
- Modify: `omcgo/internal/admin/middleware_test.go`
- Modify: `omcgo/internal/admin/middleware.go`
- Modify: `omcgo/cmd/app/provider/router.go`

**Interfaces:**
- Consumes: `PermissionChecker.CheckPermission(ctx, userID, path, method)`
- Produces: `RequireAPIPermission(roleRepo PermissionChecker) gin.HandlerFunc`

- [ ] **Step 1: 写 RBAC 失败测试**

在 `middleware_test.go` 增加表驱动测试，至少覆盖普通角色 allow/deny、checker error、super-admin bypass、无认证上下文和标准路径传参：

```go
func TestRequireAPIPermission_EnforcesEndpointPolicy(t *testing.T) {
    userID := uuid.New()
    var gotPath, gotMethod string
    repo := &mockRoleRepo{checkPermissionFn: func(_ context.Context, gotUser uuid.UUID, path, method string) (bool, error) {
        require.Equal(t, userID, gotUser)
        gotPath, gotMethod = path, method
        return false, nil
    }}

    r := gin.New()
    r.Use(func(c *gin.Context) {
        c.Set(CtxKeyUserID, userID)
        c.Set(CtxKeyIsSuperAdmin, false)
        c.Next()
    })
    r.Use(RequireAPIPermission(repo))
    r.PUT("/api/v1/admin/sysConfig/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

    req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/sysConfig/123", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusForbidden, w.Code)
    assert.Equal(t, "/api/v1/admin/sysConfig/:id", gotPath)
    assert.Equal(t, http.MethodPut, gotMethod)
}
```

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/admin -run 'TestRequireAPIPermission' -count=1
```

Expected: deny/error 测试得到 200，证明当前中间件确实只认证。

- [ ] **Step 3: 实现 fail-closed 鉴权并删除旧 fail-open 中间件**

`RequireAPIPermission` 必须：

```go
func RequireAPIPermission(roleRepo PermissionChecker) gin.HandlerFunc {
    return func(c *gin.Context) {
        userIDVal, exists := c.Get(CtxKeyUserID)
        if !exists {
            commonerrors.AbortWithError(c, http.StatusUnauthorized, errors.New("authentication required"))
            return
        }
        userID, ok := userIDVal.(uuid.UUID)
        if !ok {
            commonerrors.AbortWithError(c, http.StatusInternalServerError, errors.New("invalid user context"))
            return
        }
        if isSuper, _ := c.Get(CtxKeyIsSuperAdmin); isSuper == true {
            c.Next()
            return
        }
        if roleRepo == nil {
            commonerrors.AbortWithError(c, http.StatusInternalServerError, errors.New("permission checker not configured"))
            return
        }
        path := c.FullPath()
        if path == "" {
            commonerrors.AbortWithError(c, http.StatusInternalServerError, errors.New("route path unavailable"))
            return
        }
        allowed, err := roleRepo.CheckPermission(c.Request.Context(), userID, path, c.Request.Method)
        if err != nil {
            logger.L(c.Request.Context()).Error("API permission check failed",
                zap.String("user_id", userID.String()),
                zap.String("path", path),
                zap.String("method", c.Request.Method),
                zap.Error(err))
            commonerrors.AbortWithError(c, http.StatusInternalServerError, errors.New("permission check failed"))
            return
        }
        if !allowed {
            commonerrors.AbortWithError(c, http.StatusForbidden, errors.New("insufficient permissions"))
            return
        }
        c.Next()
    }
}
```

删除未使用的 `RequireApiPermission`、`ApiPermissionChecker`、`RoleApiEndpoint`，并移除不再需要的 `strings` import。同步修正 router 中“当前只要求认证”等错误注释。

- [ ] **Step 4: 运行 RBAC 测试确认 GREEN**

```bash
cd omcgo
go test ./internal/admin -run 'TestRequireAPIPermission' -count=1
```

Expected: 全部 PASS。

---

### Task 2: 建立 Public 与 Secret 策略分类

**Files:**
- Create: `omcgo/internal/admin/sys_config_policy.go`
- Create: `omcgo/internal/admin/sys_config_policy_test.go`
- Modify: `omcgo/internal/admin/sys_config.go`

**Interfaces:**
- Produces: `isPublicSysConfig(category,key string) bool`
- Produces: `isSecretSysConfig(category,key string) bool`
- Produces: `toSysConfigResponse(SysConfig) SysConfigResponse`

- [ ] **Step 1: 写分类与脱敏失败测试**

测试 8 个 public key、错误 category、rogue `IsPublic=true`、默认密码和 Agent Token：

```go
func TestToSysConfigResponse_RedactsSecret(t *testing.T) {
    got := toSysConfigResponse(SysConfig{
        Category: "security", Key: "defaultPasswd", Value: "OMC@123456",
    })
    assert.Empty(t, got.Value)
    assert.True(t, got.IsSecret)
    assert.True(t, got.IsConfigured)
}
```

- [ ] **Step 2: 运行测试确认 RED**

```bash
cd omcgo
go test ./internal/admin -run 'Test(IsPublicSysConfig|IsSecretSysConfig|ToSysConfigResponse)' -count=1
```

Expected: 新函数未定义。

- [ ] **Step 3: 实现策略和安全 DTO**

`sys_config_policy.go` 中定义精确白名单和 DTO：

```go
var publicSysConfigs = map[validatorKey]struct{}{
    {Category: "system", Key: "system_name"}: {},
    {Category: "system", Key: "system_version"}: {},
    {Category: "system", Key: "show_menu_icon"}: {},
    {Category: "basic", Key: "mrOMCName"}: {},
    {Category: "security", Key: "isBrowserAutoRecordPass"}: {},
    {Category: "ui_custom", Key: "ui_login_background"}: {},
    {Category: "ui_custom", Key: "ui_menu_logo_up"}: {},
    {Category: "ui_custom", Key: "ui_menu_logo_down"}: {},
}

var secretSysConfigs = map[validatorKey]struct{}{
    {Category: "security", Key: "defaultPasswd"}: {},
    {Category: "agent", Key: "agent_studio_service_token"}: {},
}
```

转换函数对 secret 设置 `Value=""`、`IsSecret=true`、`IsConfigured=raw.Value!=""`；非 secret 保留原值。

- [ ] **Step 4: 让 ListPublic 只信任白名单**

将：

```go
if item.IsPublic || isLoginPagePublicSecurityConfig(item.Category, item.Key)
```

改为：

```go
if isPublicSysConfig(item.Category, item.Key)
```

- [ ] **Step 5: 运行分类与 ListPublic 测试确认 GREEN**

```bash
cd omcgo
go test ./internal/admin -run 'Test(IsPublicSysConfig|IsSecretSysConfig|ToSysConfigResponse|SysConfigService_ListPublic)' -count=1
```

Expected: 全部 PASS，rogue `IsPublic=true` 被过滤。

---

### Task 3: 封闭 public 写入并统一 HTTP 脱敏

**Files:**
- Modify: `omcgo/internal/admin/sys_config.go`
- Modify: `omcgo/internal/admin/sys_config_handler.go`
- Modify: `omcgo/internal/admin/sys_config_policy_test.go`
- Modify: `omcgo/internal/admin/sys_config_test.go`

**Interfaces:**
- Produces: `validateGenericBatchWrite(BatchUpdateSysConfigRequest) error`
- HTTP responses: `SysConfigResponse` / `[]SysConfigResponse`

- [ ] **Step 1: 写输入守卫失败测试**

覆盖：显式 `is_public`、direct CRUD secret、batch Agent Token、空默认密码均拒绝；合法非空默认密码放行。

```go
func TestValidateGenericBatchWrite_RejectsAgentToken(t *testing.T) {
    err := validateGenericBatchWrite(BatchUpdateSysConfigRequest{
        Category: "agent",
        Items: []BatchItem{{Key: "agent_studio_service_token", Value: "secret"}},
    })
    assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}
```

- [ ] **Step 2: 运行测试确认 RED**

```bash
cd omcgo
go test ./internal/admin -run 'TestValidateGeneric|TestSysConfig.*IsPublic|TestSysConfig.*Secret' -count=1
```

Expected: 守卫不存在或现有请求被错误放行。

- [ ] **Step 3: 实现 public 派生和写入守卫**

- Create：若 `req.IsPublic != nil` 返回 `ErrInvalidInput`；`cfg.IsPublic=isPublicSysConfig(...)`。
- Update：若 `req.IsPublic != nil` 返回 `ErrInvalidInput`；按已有 category/key 重新派生 public。
- Direct Create/Update 对 secret 返回 400。
- Generic Batch 对 Agent Token 和空默认密码返回 400。
- Repository BatchUpsert 冲突更新改为 `is_public = EXCLUDED.is_public`。

- [ ] **Step 4: 所有 handler 响应改为安全 DTO**

List/Get/Create/Update 调用 `toSysConfigResponse` 或 slice 转换；ListPublic 同样使用响应 DTO，保证未来错误分类也不会直接序列化 domain model。

BatchUpdate 在调用 service 前执行 generic write guard，内部 Agent service 直接调用 `SysConfigService.BatchUpsert` 不受影响。

- [ ] **Step 5: 运行后端定向测试确认 GREEN**

```bash
cd omcgo
go test ./internal/admin ./internal/agentconfig -count=1
```

Expected: 全部 PASS，Agent 专用 Token 保存测试保持通过。

---

### Task 4: 前端接入 Secret 响应契约

**Files:**
- Modify: `omcmb/frontend-core/src/types/system.ts`
- Modify: `omcmb/frontend-core/src/services/api/adminApi.ts`
- Create or Modify: `omcmb/frontend-core/src/services/api/adminApi.test.ts`

**Interfaces:**
- Produces: `SysConfigItem.isSecret?: boolean`
- Produces: `SysConfigItem.isConfigured?: boolean`

- [ ] **Step 1: 写映射失败测试**

导出或通过 API mock 验证 snake_case 映射：

```ts
expect(mapped).toMatchObject({
  key: 'defaultPasswd',
  value: '',
  isSecret: true,
  isConfigured: true,
});
```

- [ ] **Step 2: 运行测试确认 RED**

```bash
cd omcmb
npx vitest run frontend-core/src/services/api/adminApi.test.ts
```

Expected: 新字段缺失。

- [ ] **Step 3: 实现类型与映射**

后端类型增加：

```ts
is_secret?: boolean;
is_configured?: boolean;
```

`mapBackendSysConfig` 映射为 camelCase。

- [ ] **Step 4: 运行映射测试确认 GREEN**

```bash
cd omcmb
npx vitest run frontend-core/src/services/api/adminApi.test.ts
```

---

### Task 5: 默认密码表单改为 write-only

**Files:**
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/sysConfigSerialize.test.ts`
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/sysConfigSerialize.ts`
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/index.tsx`
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/SecuritySettings.tsx`
- Modify: matching zh-CN/en-US i18n files

**Interfaces:**
- `buildBatchItems(formValues, existing)` skips blank existing secret
- `SecuritySettings` consumes `defaultPasswordConfigured: boolean`

- [ ] **Step 1: 写 serializer 失败测试**

```ts
it('已配置 secret 留空时不覆盖原值', () => {
  const existing = [{
    key: 'defaultPasswd', value: '', valueType: 'string',
    isSecret: true, isConfigured: true,
  }] as SysConfigItem[];
  expect(buildBatchItems({ defaultPasswd: '', pwdMinLength: 10 }, existing))
    .toEqual([{ key: 'pwdMinLength', value: '10', value_type: 'int' }]);
});

it('输入新 secret 时生成更新项', () => {
  const existing = [{ key: 'defaultPasswd', value: '', valueType: 'string', isSecret: true }] as SysConfigItem[];
  expect(buildBatchItems({ defaultPasswd: 'New@123456' }, existing)[0].value)
    .toBe('New@123456');
});
```

- [ ] **Step 2: 运行测试确认 RED**

```bash
cd omcmb
npx vitest run webcode/src/pages/system/SystemConfig/sysConfigSerialize.test.ts
```

Expected: 空 secret 仍生成 item。

- [ ] **Step 3: 实现 serializer 与表单行为**

`buildBatchItems`：

```ts
const existingItem = existingMap.get(key);
if (existingItem?.isSecret && String(raw ?? '').trim() === '') continue;
```

`index.tsx` 灌表单时跳过 `item.isSecret`；计算 `defaultPasswordConfigured` 传给 SecuritySettings。

`SecuritySettings.tsx`：

- 删除 initialValues 中的硬编码默认密码；
- 使用不带可见性切换的密码输入；
- placeholder 根据 configured 状态显示“已配置，留空不修改”或“未配置，请输入新密码”；
- 不显示固定掩码或原始值。

- [ ] **Step 4: 运行 serializer 与相关组件测试确认 GREEN**

```bash
cd omcmb
npx vitest run webcode/src/pages/system/SystemConfig/sysConfigSerialize.test.ts
npm run typecheck
```

---

### Task 6: 全量验证与报告更新

**Files:**
- Modify: `docs/review-report/20260720/REVIEW_cccbb2b3_system-config-code-review.md`
- Modify: `docs/review-report/20260720/TEST_cccbb2b3_system-config-functional.md`

**Interfaces:**
- Produces: 两个 P0 的关闭证据和数据库明文存储剩余风险

- [ ] **Step 1: 格式化与定向验证**

```bash
gofmt -w omcgo/internal/admin/middleware.go omcgo/internal/admin/middleware_test.go \
  omcgo/internal/admin/sys_config.go omcgo/internal/admin/sys_config_handler.go \
  omcgo/internal/admin/sys_config_policy.go omcgo/internal/admin/sys_config_policy_test.go

cd omcgo
go test ./internal/admin ./internal/agentconfig ./cmd/app/provider -count=1
```

- [ ] **Step 2: 后端全量验证**

```bash
cd omcgo
go build ./...
go test ./...
```

- [ ] **Step 3: 前端验证**

```bash
cd omcmb
npm run typecheck
npx eslint webcode/src/pages/system/SystemConfig \
  frontend-core/src/services/api/adminApi.ts \
  frontend-core/src/types/system.ts --quiet
npx vitest run webcode/src/pages/system/SystemConfig/sysConfigSerialize.test.ts
```

- [ ] **Step 4: 浏览器安全行为回归**

```bash
cd omcmb/webcode
npx playwright test e2e/security-policy.spec.ts --workers=1
```

- [ ] **Step 5: 更新报告**

报告必须逐项记录 M-01～M-14 的实现文件、测试证据、P0 状态，以及仍未解决的数据库明文/Secret Manager 风险。不得把未验证项写成通过。

- [ ] **Step 6: 最终差异检查**

```bash
git diff --check
git status --short
```

Expected: 无 whitespace error；只包含本次安全修复与既有未跟踪审查文档。
