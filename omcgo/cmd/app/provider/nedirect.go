package provider

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/ratelimit"
	"github.com/omcgo/omcgo/internal/nedirect"
)

// buildNEDirectAuthenticator 把 admin 的 JWT / API-Key 校验适配成
// nedirect.Authenticator —— 让 NE Direct（独立 stdlib server）复用与管理面
// 完全相同的认证体系（不另造一套）。
//
// JWTService 缺失时返回 nil；调用方据此判断是否能安全启用 NE Direct
// （nil authenticator → 中间件拒绝所有请求，绝不静默放行）。
func buildNEDirectAuthenticator(c *Container) nedirect.Authenticator {
	if c.JWTService == nil {
		return nil
	}

	validateToken := func(_ context.Context, token string) (*nedirect.Principal, error) {
		claims, err := c.JWTService.ValidateAccessToken(token)
		if err != nil {
			return nil, fmt.Errorf("validate ne-direct jwt: %w", err)
		}
		return &nedirect.Principal{
			UserID:       claims.UserID,
			Username:     claims.Username,
			IsSuperAdmin: claims.IsSuperAdmin,
		}, nil
	}

	var validateAPIKey nedirect.APIKeyValidator
	if c.APIKeySvc != nil && c.UserRepo != nil {
		validateAPIKey = func(ctx context.Context, apiKey string) (*nedirect.Principal, error) {
			key, err := c.APIKeySvc.Validate(ctx, apiKey)
			if err != nil {
				return nil, fmt.Errorf("validate ne-direct api key: %w", err)
			}
			user, err := c.UserRepo.GetByID(ctx, key.UserID)
			if err != nil {
				return nil, fmt.Errorf("ne-direct api key owner lookup: %w", err)
			}
			return &nedirect.Principal{
				UserID:       user.ID,
				Username:     user.Username,
				IsSuperAdmin: user.IsSuperAdmin(),
			}, nil
		}
	}

	return nedirect.FuncAuthenticator{
		ValidateToken:  validateToken,
		ValidateAPIKey: validateAPIKey,
	}
}

// buildNEDirectMiddleware 组装 NE Direct 安全中间件：认证 + per-endpoint/per-device
// Redis 固定窗口限流 + 审计日志。阈值取自 NEDirectConfig（带默认值）。
func buildNEDirectMiddleware(c *Container, logger *zap.Logger) *nedirect.Middleware {
	auth := buildNEDirectAuthenticator(c)
	endpointLimiter := ratelimit.NewFixedWindowLimiter(
		c.Redis, "ne:endpoint", int64(c.Cfg.NEDirect.EndpointRateLimit()), time.Minute)
	deviceLimiter := ratelimit.NewFixedWindowLimiter(
		c.Redis, "ne:device", int64(c.Cfg.NEDirect.DeviceRateLimit()), time.Minute)
	return nedirect.NewMiddleware(auth, endpointLimiter, deviceLimiter, logger)
}
