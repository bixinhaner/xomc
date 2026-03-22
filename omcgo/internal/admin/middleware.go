package admin

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Context keys for authenticated user information.
const (
	CtxKeyUserID        = "user_id"
	CtxKeyUsername       = "username"
	CtxKeyCarrier        = "carrier"
	CtxKeyCarrierFilter  = "carrier_filter"
	CtxKeyRoles          = "roles"
)

// RequireAuth returns a Gin middleware that validates JWT access tokens
// and sets user information in the request context.
func RequireAuth(jwt *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing authorization header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid authorization header format",
			})
			return
		}

		claims, err := jwt.ValidateAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid or expired token",
			})
			return
		}

		c.Set(CtxKeyUserID, claims.UserID)
		c.Set(CtxKeyUsername, claims.Username)
		c.Set(CtxKeyCarrier, claims.Carrier)
		c.Set(CtxKeyRoles, claims.Roles)
		c.Next()
	}
}

// RequirePermission returns a Gin middleware that checks the authenticated
// user has the specified resource-action permission.
func RequirePermission(roleRepo RoleRepository, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get(CtxKeyUserID)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "authentication required",
			})
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "invalid user context",
			})
			return
		}

		allowed, err := roleRepo.CheckPermission(c.Request.Context(), userID, resource, action)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "permission check failed",
			})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// RequireCarrier returns a Gin middleware that sets a carrier filter
// based on the authenticated user's carrier binding.
// If the user has no carrier binding (nil), no filter is applied (super admin).
func RequireCarrier() gin.HandlerFunc {
	return func(c *gin.Context) {
		carrierVal, exists := c.Get(CtxKeyCarrier)
		if !exists {
			c.Next()
			return
		}

		if carrier, ok := carrierVal.(*model.CarrierCode); ok && carrier != nil {
			c.Set(CtxKeyCarrierFilter, *carrier)
		}

		c.Next()
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
func RequireResourcePermission(roleRepo RoleRepository, resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		action := httpMethodToAction(c.Request.Method)

		userIDVal, exists := c.Get(CtxKeyUserID)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "authentication required",
			})
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "invalid user context",
			})
			return
		}

		allowed, err := roleRepo.CheckPermission(c.Request.Context(), userID, resource, action)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "permission check failed",
			})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "insufficient permissions",
			})
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

		// Fire and forget — audit logging should not block the response
		go func() {
			_ = auditRepo.Create(c.Request.Context(), log)
		}()
	}
}
