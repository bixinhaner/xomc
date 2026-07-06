package agentaction

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

func RequireDelegation(jwtService *admin.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if jwtService == nil {
			commonerrors.AbortWithError(c, http.StatusServiceUnavailable, commonerrors.ErrUnavailable)
			return
		}
		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			commonerrors.AbortWithError(c, http.StatusUnauthorized, errors.New("missing agent delegation token"))
			return
		}
		claims, err := jwtService.ValidateAgentDelegationToken(parts[1])
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusUnauthorized, errors.New("invalid or expired agent delegation token"))
			return
		}
		attachClaims(c, claims)
		c.Next()
	}
}

func attachClaims(c *gin.Context, claims *admin.Claims) {
	c.Set(admin.CtxKeyUserID, claims.UserID)
	c.Set(admin.CtxKeyUsername, claims.Username)
	c.Set(admin.CtxKeyIsSuperAdmin, claims.IsSuperAdmin)
	c.Set(admin.CtxKeyRoles, claims.Roles)
	c.Set(admin.CtxKeyClaims, claims)

	ctx := context.WithValue(c.Request.Context(), admin.CtxKeyUserID, claims.UserID)
	ctx = context.WithValue(ctx, admin.CtxKeyUsername, claims.Username)
	ctx = context.WithValue(ctx, admin.CtxKeyIsSuperAdmin, claims.IsSuperAdmin)
	ctx = context.WithValue(ctx, admin.CtxKeyRoles, claims.Roles)
	ctx = context.WithValue(ctx, admin.CtxKeyClaims, claims)
	c.Request = c.Request.WithContext(ctx)
}
