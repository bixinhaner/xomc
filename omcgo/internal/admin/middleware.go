package admin

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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

			c.Set(CtxKeyUserID, user.ID)
			c.Set(CtxKeyUsername, user.Username)
			c.Set(CtxKeyIsSuperAdmin, user.IsSuperAdmin())

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
			c.Set(CtxKeyRoles, roleNames)
			c.Next()
			return
		}

		// Fall back to Bearer JWT
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			commonerrors.AbortWithError(c, http.StatusUnauthorized,
				errors.New("missing authorization header"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			commonerrors.AbortWithError(c, http.StatusUnauthorized,
				errors.New("invalid authorization header format"))
			return
		}

		claims, err := jwt.ValidateAccessToken(parts[1])
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

		c.Set(CtxKeyUserID, claims.UserID)
		c.Set(CtxKeyUsername, claims.Username)
		c.Set(CtxKeyIsSuperAdmin, claims.IsSuperAdmin)
		c.Set(CtxKeyRoles, claims.Roles)
		c.Set(CtxKeyClaims, claims)
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
