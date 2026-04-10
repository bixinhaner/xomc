package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

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
	CtxKeyClaims         = "claims"
)

// RequireAuth returns a Gin middleware that validates JWT access tokens
// and sets user information in the request context.
// It also supports X-API-Key header for programmatic access.
func RequireAuth(jwt *JWTService) gin.HandlerFunc {
	return RequireAuthWithAPIKey(jwt, nil, nil, nil)
}

// RequireAuthWithAPIKey returns a Gin middleware that validates either a JWT Bearer token
// or an X-API-Key header, and sets user information in the request context.
func RequireAuthWithAPIKey(jwt *JWTService, apiKeySvc *APIKeyService, userRepo UserRepository, roleReader RoleReader) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try X-API-Key header first
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" && apiKeySvc != nil && userRepo != nil {
			key, err := apiKeySvc.Validate(c.Request.Context(), apiKey)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "invalid or expired API key",
				})
				return
			}

			// Load user to get carrier and roles
			user, err := userRepo.GetByID(c.Request.Context(), key.UserID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "API key owner not found",
				})
				return
			}

			c.Set(CtxKeyUserID, user.ID)
			c.Set(CtxKeyUsername, user.Username)
			c.Set(CtxKeyCarrier, user.Carrier)

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

		// Inject carrier domain into context for Casbin
		ctx := c.Request.Context()
		if carrierVal, exists := c.Get(CtxKeyCarrier); exists {
			if carrier, ok := carrierVal.(*model.CarrierCode); ok && carrier != nil {
				ctx = context.WithValue(ctx, CtxKeyCarrier, string(*carrier))
			}
		}

		allowed, err := roleRepo.CheckPermission(ctx, userID, resource, action)
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
func RequireResourcePermission(roleRepo PermissionChecker, resource string) gin.HandlerFunc {
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

		// Inject carrier domain into context for Casbin
		ctx := c.Request.Context()
		if carrierVal, exists := c.Get(CtxKeyCarrier); exists {
			if carrier, ok := carrierVal.(*model.CarrierCode); ok && carrier != nil {
				ctx = context.WithValue(ctx, CtxKeyCarrier, string(*carrier))
			}
		}

		allowed, err := roleRepo.CheckPermission(ctx, userID, resource, action)
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
