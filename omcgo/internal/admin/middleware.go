package admin

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components/logger"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// Context keys for authenticated user information.
//
// v1.0：移除 CtxKeyCarrier / CtxKeyCarrierFilter（users.carrier 已删，不再传播）。
// 新增 CtxKeyIsSuperAdmin：来自 JWT claims.IsSuperAdmin（user.IsSuperAdmin() 派生）。
const (
	CtxKeyUserID       = "user_id"
	CtxKeyUsername     = "username"
	CtxKeyIsSuperAdmin = "is_super_admin"
	CtxKeyRoles        = "roles"
	CtxKeyClaims       = "claims"
)

func attachAuthContext(c *gin.Context, userID uuid.UUID, username string, isSuperAdmin bool, roles []string, claims *Claims) {
	c.Set(CtxKeyUserID, userID)
	c.Set(CtxKeyUsername, username)
	c.Set(CtxKeyIsSuperAdmin, isSuperAdmin)
	c.Set(CtxKeyRoles, roles)
	if claims != nil {
		c.Set(CtxKeyClaims, claims)
	}

	ctx := context.WithValue(c.Request.Context(), CtxKeyUserID, userID)
	ctx = context.WithValue(ctx, CtxKeyUsername, username)
	ctx = context.WithValue(ctx, CtxKeyIsSuperAdmin, isSuperAdmin)
	ctx = context.WithValue(ctx, CtxKeyRoles, roles)
	if claims != nil {
		ctx = context.WithValue(ctx, CtxKeyClaims, claims)
	}
	c.Request = c.Request.WithContext(ctx)
}

// RequireAuth returns a Gin middleware that validates JWT access tokens
// and sets user information in the request context.
// It also supports X-API-Key header for programmatic access.
func RequireAuth(jwt *JWTService) gin.HandlerFunc {
	return RequireAuthWithAPIKey(jwt, nil, nil, nil, nil)
}

// RequireAuthWithAPIKey returns a Gin middleware that validates either a JWT Bearer token
// or an X-API-Key header, and sets user information in the request context.
// 如果 revoker 非 nil，会在 JWT 验证后查询用户级撤销记录（强制下线场景），
// token.iat 早于撤销时间戳则拒绝（401）。
func RequireAuthWithAPIKey(jwt *JWTService, apiKeySvc *APIKeyService, userRepo UserRepository, roleReader RoleReader, revoker *TokenRevoker) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try X-API-Key header first
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" && apiKeySvc != nil && userRepo != nil {
			key, err := apiKeySvc.Validate(c.Request.Context(), apiKey)
			if err != nil {
				commonerrors.AbortWithError(c, http.StatusUnauthorized,
					errors.New("invalid or expired API key"))
				return
			}

			// Load user to derive IsSuperAdmin and roles (v1.0：carrier 已删除)
			user, err := userRepo.GetByID(c.Request.Context(), key.UserID)
			if err != nil {
				commonerrors.AbortWithError(c, http.StatusUnauthorized,
					errors.New("API key owner not found"))
				return
			}

			// Load roles for API key user when roleReader is available
			var roleNames []string
			if roleReader != nil {
				if roles, err := roleReader.List(c.Request.Context()); err == nil {
					// API key inherits all roles of the key owner
					roleNames = make([]string, 0, len(roles))
					for _, r := range roles {
						roleNames = append(roleNames, r.Name)
					}
				}
			}
			attachAuthContext(c, user.ID, user.Username, user.IsSuperAdmin(), roleNames, nil)
			c.Next()
			return
		}

		// Fall back to Bearer JWT.
		//
		// L-3 修复：浏览器 EventSource 不支持自定义 header，SSE 端点必须用
		// ?token= query 鉴权。如果 Authorization header 缺失，再 fallback 到
		// ?token= query。注意 token 进 URL 会被 access log / referer 记录，
		// 仅推荐给 SSE / 浏览器下载等无法发 header 的场景。
		var tokenStr string
		authHeader := c.GetHeader("Authorization")
		switch {
		case authHeader != "":
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				commonerrors.AbortWithError(c, http.StatusUnauthorized,
					errors.New("invalid authorization header format"))
				return
			}
			tokenStr = parts[1]
		default:
			tokenStr = c.Query("token")
		}
		if tokenStr == "" {
			commonerrors.AbortWithError(c, http.StatusUnauthorized,
				errors.New("missing authorization header"))
			return
		}

		claims, err := jwt.ValidateAccessToken(tokenStr)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusUnauthorized,
				errors.New("invalid or expired token"))
			return
		}

		// 强制下线检查：token 签发时间早于用户最近一次 ForceLogout 时拒绝。
		if revoker != nil {
			revoked, err := revoker.IsRevoked(c.Request.Context(), claims.UserID, claims.IssuedAt)
			if err == nil && revoked {
				commonerrors.AbortWithError(c, http.StatusUnauthorized,
					errors.New("session was forcibly terminated, please re-login"))
				return
			}
		}

		attachAuthContext(c, claims.UserID, claims.Username, claims.IsSuperAdmin, claims.Roles, claims)
		c.Next()
	}
}

// RequirePermission returns a Gin middleware that checks the authenticated
// user has the specified resource-action permission.
func RequirePermission(roleRepo PermissionChecker, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get(CtxKeyUserID)
		if !exists {
			commonerrors.AbortWithError(c, http.StatusUnauthorized,
				errors.New("authentication required"))
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				errors.New("invalid user context"))
			return
		}

		// v1.0：超管旁路 — builtIn 用户直接放行（不再走 Casbin domain）。
		if isSuper, _ := c.Get(CtxKeyIsSuperAdmin); isSuper == true {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		allowed, err := roleRepo.CheckPermission(ctx, userID, resource, action)
		if err != nil {
			// 把真实 err 写日志，前端只暴露通用 500 文案。常见根因：
			//   - "casbin authorizer not configured"（启动期 Casbin 初始化失败）
			//   - "casbin enforce: ..."（策略加载/匹配出错）
			logger.L(ctx).Error("permission check failed",
				zap.String("user_id", userID.String()),
				zap.String("resource", resource),
				zap.String("action", action),
				zap.Error(err),
			)
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				errors.New("permission check failed"))
			return
		}

		if !allowed {
			commonerrors.AbortWithError(c, http.StatusForbidden,
				errors.New("insufficient permissions"))
			return
		}

		c.Next()
	}
}

// RequireAPIPermission returns a Gin middleware that checks the authenticated
// user has permission for the current request's URL path + HTTP method.
//
// 端点级（path+method）鉴权 — B3-Phase2 切换段使用，对齐 GVA 风格。
// 当前 (Phase1) 已挂上的 LoadPolicy 双源加载使端点级策略可被 Casbin Enforce 命中，
// 但本中间件暂未挂任何路由组——Phase2 会替换 router.go 中的 RequirePermission /
// RequireResourcePermission 调用为本中间件。
//
// 与 RequirePermission 的差异：
//   - RequirePermission(roleRepo, resource, action) → 硬编码业务字符串（粗粒度）
//   - RequireAPIPermission(roleRepo)               → 自动读 c.Request.URL.Path /
//     c.Request.Method（端点级）
//
// 超管旁路：claims.IsSuperAdmin（user.source='builtIn' 派生）一致。
// 详见 docs/prd/system/menu-dynamic-loading.md §4.2.4 (B3-Phase1/Phase2)。
func RequireAPIPermission(roleRepo PermissionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get(CtxKeyUserID)
		if !exists {
			commonerrors.AbortWithError(c, http.StatusUnauthorized,
				errors.New("authentication required"))
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				errors.New("invalid user context"))
			return
		}

		// builtIn 超管旁路（与 RequirePermission/RequireResourcePermission 行为一致）
		if isSuper, _ := c.Get(CtxKeyIsSuperAdmin); isSuper == true {
			c.Next()
			return
		}

		// 端点级鉴权：obj=path, act=method
		path := c.Request.URL.Path
		method := c.Request.Method
		ctx := c.Request.Context()
		allowed, err := roleRepo.CheckPermission(ctx, userID, path, method)
		if err != nil {
			logger.L(ctx).Error("api permission check failed",
				zap.String("user_id", userID.String()),
				zap.String("path", path),
				zap.String("method", method),
				zap.Error(err),
			)
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				errors.New("permission check failed"))
			return
		}
		if !allowed {
			commonerrors.AbortWithError(c, http.StatusForbidden,
				errors.New("insufficient permissions"))
			return
		}

		c.Next()
	}
}

// RequireCarrier — v1.0 已删除：users.carrier 已从 schema 移除，无 carrier 过滤需求。
// 如旧代码仍 import 该函数，需要按 PRD §11.11 改造。

// RequireSuperAdmin 严格仅放行 super_admin（builtIn 用户）。
//
// 与 RequirePermission 的区别：RequirePermission 让 super_admin 旁路，但其他角色
// 也可能因显式 RBAC 授权而通过；RequireSuperAdmin 则**只**放 super_admin，
// 不接受任何其他角色，即使该角色被赋予了对应资源的权限。
//
// 用于 T-0098 P3-05 收口：产品装配件 / 参数模型 / KPI 库 / 告警库治理 API
// 仅 super_admin 可写改读。运维 admin / operator / viewer 一律 403。
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSuper, _ := c.Get(CtxKeyIsSuperAdmin); isSuper == true {
			c.Next()
			return
		}
		commonerrors.AbortWithError(c, http.StatusForbidden,
			errors.New("super_admin role required"))
	}
}

// RequireResourcePermission returns a Gin middleware that dynamically determines
// the permission action based on the HTTP method and checks the user has the
// corresponding resource permission.
//
// Method → action mapping:
//   - GET, HEAD, OPTIONS → "read"
//   - POST, PUT, PATCH   → "write"
//   - DELETE              → "delete"
func RequireResourcePermission(roleRepo PermissionChecker, resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		action := httpMethodToAction(c.Request.Method)

		userIDVal, exists := c.Get(CtxKeyUserID)
		if !exists {
			commonerrors.AbortWithError(c, http.StatusUnauthorized,
				errors.New("authentication required"))
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				errors.New("invalid user context"))
			return
		}

		// v1.0：超管旁路 — builtIn 用户直接放行（不再走 Casbin domain）。
		if isSuper, _ := c.Get(CtxKeyIsSuperAdmin); isSuper == true {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		allowed, err := roleRepo.CheckPermission(ctx, userID, resource, action)
		if err != nil {
			logger.L(ctx).Error("resource permission check failed",
				zap.String("user_id", userID.String()),
				zap.String("resource", resource),
				zap.String("action", action),
				zap.Error(err),
			)
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				errors.New("permission check failed"))
			return
		}

		if !allowed {
			commonerrors.AbortWithError(c, http.StatusForbidden,
				errors.New("insufficient permissions"))
			return
		}

		c.Next()
	}
}

// httpMethodToAction maps an HTTP method to a permission action string.
func httpMethodToAction(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return "read"
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return "write"
	case http.MethodDelete:
		return "delete"
	default:
		return "read"
	}
}

// ApiPermissionChecker is implemented by CasbinAuthorizer and provides
// role-based API path permission checking.
type ApiPermissionChecker interface {
	// GetRoleApiEndpoints returns allowed (path, method) pairs for the given role IDs.
	// An empty result means no explicit API permissions are configured.
	GetRoleApiEndpoints(ctx context.Context, roleNames []string) ([]RoleApiEndpoint, error)
}

// RoleApiEndpoint represents a single (path, method) permission entry for a role.
type RoleApiEndpoint struct {
	Path   string
	Method string
}

// RequireApiPermission returns a Gin middleware that checks whether the current user's
// roles have explicit permission to access the current request path + method.
//
// Bypass rules:
//   - If apiChecker is nil, the middleware is a no-op (allows everything).
//   - If the user has no roles, the check is skipped (other middleware handles auth).
//   - If the role list is empty in the database (no API permissions configured), the
//     middleware passes through to avoid a hard break during initial setup.
//
// Access rules:
//   - Roles named "admin" or "super_admin" are always granted access.
//   - Otherwise, at least one of the user's roles must have an explicit allow entry.
func RequireApiPermission(apiChecker ApiPermissionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiChecker == nil {
			c.Next()
			return
		}

		// Extract role names from JWT context
		rolesVal, exists := c.Get(CtxKeyRoles)
		if !exists {
			c.Next()
			return
		}
		roleNames, ok := rolesVal.([]string)
		if !ok || len(roleNames) == 0 {
			c.Next()
			return
		}

		// Admin roles bypass API permission check
		for _, name := range roleNames {
			if name == "admin" || name == "super_admin" {
				c.Next()
				return
			}
		}

		// Query allowed endpoints for the current roles
		allowed, err := apiChecker.GetRoleApiEndpoints(c.Request.Context(), roleNames)
		if err != nil {
			// On error, allow through to avoid service disruption
			c.Next()
			return
		}

		// If no API permissions are configured, allow through (initial setup mode)
		if len(allowed) == 0 {
			c.Next()
			return
		}

		reqPath := c.FullPath()
		reqMethod := c.Request.Method

		for _, ep := range allowed {
			if ep.Path == reqPath && strings.EqualFold(ep.Method, reqMethod) {
				c.Next()
				return
			}
		}

		commonerrors.AbortWithError(c, http.StatusForbidden,
			errors.New("API access not permitted"))
	}
}

// AuditLogger returns a Gin middleware that records write operations
// (POST, PUT, PATCH, DELETE) to the audit log.
func AuditLogger(auditRepo AuditRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only audit write operations
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			return
		}

		// Only audit successful operations
		if c.Writer.Status() >= 400 {
			return
		}

		username, _ := c.Get(CtxKeyUsername)
		usernameStr, _ := username.(string)
		if usernameStr == "" {
			return
		}

		var userID *uuid.UUID
		if uid, exists := c.Get(CtxKeyUserID); exists {
			if id, ok := uid.(uuid.UUID); ok {
				userID = &id
			}
		}

		log := &AuditLog{
			UserID:    userID,
			Username:  usernameStr,
			Action:    method,
			Resource:  c.FullPath(),
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		}

		// Fire and forget — audit logging should not block the response.
		// Use context.Background() because the request context will be
		// canceled after the handler returns.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = auditRepo.Create(ctx, log)
		}()
	}
}

// OperLogger 返回一个 Gin 中间件，把管理面的写操作（POST/PUT/PATCH/DELETE）
// 记录到 sys_oper_logs（#122）。
//
// 与 AuditLogger（合规审计 audit_logs）并行：本表支撑 /admin/logs/operation
// 视图，字段更贴近 GVA 风格（method/path/操作人/IP/UA/状态码/耗时）。两者刻意
// 并存而非合并——audit_logs 是不可变合规链，sys_oper_logs 是可分页查询的运维视图。
//
// 写入策略与 recordLoginLog 对齐：fire-and-forget goroutine + context.Background，
// 写失败仅 Warn 不阻塞请求；logRepo 为 nil 时整体降级为 no-op。
//
// 记录范围：仅写操作。GET/HEAD/OPTIONS 读端点与高频健康检查不记录，避免
// sys_oper_logs 被读流量淹没。失败请求（状态码 >=400）也记录一条 status=false，
// 便于排查越权 / 校验失败的操作尝试。
func OperLogger(logRepo OperLogWriter, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		if logRepo == nil {
			return
		}

		// 只记录写操作；读端点（GET/HEAD/OPTIONS）与健康检查不入库。
		method := c.Request.Method
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			return
		}

		// 未认证（无 username）请求一般已被前置 auth 中间件 Abort，
		// 此处再兜底跳过——无操作人主体的写操作不计入操作日志。
		usernameVal, _ := c.Get(CtxKeyUsername)
		username, _ := usernameVal.(string)
		if username == "" {
			return
		}

		var userID *uuid.UUID
		if uid, exists := c.Get(CtxKeyUserID); exists {
			if id, ok := uid.(uuid.UUID); ok {
				userID = &id
			}
		}

		status := c.Writer.Status()
		req := CreateOperLogRequest{
			UserID:    userID,
			Username:  username,
			Action:    method,
			Module:    operModuleFromPath(c.FullPath()),
			Target:    c.FullPath(),
			Detail:    method + " " + c.Request.URL.Path,
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Status:    status < 400,
			CostMs:    int(time.Since(start).Milliseconds()),
		}
		if status >= 400 {
			req.ErrorMsg = http.StatusText(status)
		}

		// Fire and forget — 操作日志写入不阻塞响应；用 background context
		// 避免 HTTP 请求结束后 ctx 被取消。
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := logRepo.CreateOperLog(ctx, req); err != nil {
				logger.Warn("failed to write operation log",
					zap.String("username", username),
					zap.String("method", method),
					zap.String("path", req.Target),
					zap.Error(err))
			}
		}()
	}
}

// OperLogWriter 是 OperLogger 中间件依赖的最小写入接口（LogRepository 的子集）。
// 独立定义而非直接用 LogRepository，是为了让中间件只暴露它真正需要的能力，
// 也便于单测注入轻量 mock。
type OperLogWriter interface {
	CreateOperLog(ctx context.Context, req CreateOperLogRequest) error
}

// operModuleFromPath 从受保护路由的 FullPath（形如 /api/v1/devices/:id）粗提
// 出业务模块名（devices/alarms/...），供 sys_oper_logs.module 列分组筛选。
// 取 /api/v1/ 之后的第一段；无法识别时回退 "system"。
func operModuleFromPath(fullPath string) string {
	const prefix = "/api/v1/"
	rest := fullPath
	if idx := strings.Index(fullPath, prefix); idx >= 0 {
		rest = fullPath[idx+len(prefix):]
	}
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		return "system"
	}
	if slash := strings.IndexByte(rest, '/'); slash >= 0 {
		rest = rest[:slash]
	}
	if rest == "" {
		return "system"
	}
	return rest
}
