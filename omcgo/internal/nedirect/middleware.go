package nedirect

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/ratelimit"
)

var (
	errMissingCredentials = errors.New("missing credentials")
	errInvalidAuthHeader  = errors.New("invalid authorization header")
)

// Principal 是经认证的调用方身份（管理员 / 内部 API Key 持有者）。
// NE Direct 是面向运维管理员的内部接口（默认 ne_direct.enabled=false），
// 所有端点都要求一个已认证的 Principal 才能访问。
type Principal struct {
	UserID       uuid.UUID
	Username     string
	IsSuperAdmin bool
}

// Authenticator 校验请求凭据并返回 Principal。
//
// 定义为窄接口（而非直接依赖 internal/admin）是为了：① 避免 nedirect → admin 的
// 包依赖耦合；② 便于单测注入 fake。生产实现见 cmd/app/provider 中对 admin.JWTService /
// admin.APIKeyService 的适配（jwtAuthenticator）。
type Authenticator interface {
	// Authenticate 解析 HTTP 请求中的 Bearer JWT 或 X-API-Key，校验通过返回 Principal；
	// 失败返回 error（中间件统一映射为 401）。
	Authenticate(ctx context.Context, r *http.Request) (*Principal, error)
}

// TokenValidator 校验一个 Bearer JWT 字符串，返回 Principal。由 admin.JWTService 适配。
type TokenValidator func(ctx context.Context, token string) (*Principal, error)

// APIKeyValidator 校验一个 X-API-Key，返回 Principal。由 admin.APIKeyService + UserRepo 适配。
type APIKeyValidator func(ctx context.Context, apiKey string) (*Principal, error)

// FuncAuthenticator 用两个校验函数组合成 Authenticator：优先 X-API-Key（程序化访问），
// 再退化到 Bearer JWT。两者皆缺/皆失败 → error（→ 401）。
//
// 这样 nedirect 无需 import internal/admin，provider 侧把 admin.JWTService.ValidateAccessToken /
// admin.APIKeyService.Validate 适配成本类型即可（见 cmd/app/provider）。
type FuncAuthenticator struct {
	ValidateToken  TokenValidator
	ValidateAPIKey APIKeyValidator
}

// Authenticate 实现 Authenticator。
func (a FuncAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*Principal, error) {
	// 1. X-API-Key 优先（与管理面 RequireAuthWithAPIKey 顺序一致）。
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" && a.ValidateAPIKey != nil {
		return a.ValidateAPIKey(ctx, apiKey)
	}
	// 2. Bearer JWT。
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errMissingCredentials
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return nil, errInvalidAuthHeader
	}
	if a.ValidateToken == nil {
		return nil, errMissingCredentials
	}
	return a.ValidateToken(ctx, parts[1])
}

// principalCtxKey 是 Principal 在 request context 中的键类型（避免字符串键碰撞）。
type principalCtxKey struct{}

// PrincipalFromContext 从 context 取出已认证的 Principal。第二个返回值表示是否存在。
func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalCtxKey{}).(*Principal)
	return p, ok
}

// Middleware 为 NE Direct 的 net/http 处理器统一加上：
//
//	认证（必选）→ 端点级限流 → per-device 限流 → 审计日志 → 业务处理
//
// 与管理面 Gin 中间件分离：NE Direct 跑在独立 stdlib http.Server 上，无法复用
// admin.RequireAuth（Gin）。本中间件用同一套 JWT / API-Key 校验（经 Authenticator 适配），
// 不另造鉴权体系。
type Middleware struct {
	auth           Authenticator
	endpointLimiter *ratelimit.FixedWindowLimiter
	deviceLimiter   *ratelimit.FixedWindowLimiter
	logger          *zap.Logger
}

// NewMiddleware 构造 NE Direct 安全中间件。
//   - auth            : 认证器（nil 时所有请求 401，安全默认 —— 绝不静默放行）；
//   - endpointLimiter : 端点级限流器（per-endpoint，key=端点名）；nil 时不限流；
//   - deviceLimiter   : 设备级限流器（per-device，key=设备 SN）；nil 时不限流。
func NewMiddleware(
	auth Authenticator,
	endpointLimiter *ratelimit.FixedWindowLimiter,
	deviceLimiter *ratelimit.FixedWindowLimiter,
	logger *zap.Logger,
) *Middleware {
	return &Middleware{
		auth:            auth,
		endpointLimiter: endpointLimiter,
		deviceLimiter:   deviceLimiter,
		logger:          logger.Named("nedirect-mw"),
	}
}

// Wrap 把一个业务 handler 包装为「认证 + 端点限流 + 审计」的 handler。
// deviceSNFn 从请求中提取设备 SN 用于 per-device 限流与审计（可能返回空串，
// 例如 list/sessions 这类不针对单设备的端点 —— 此时跳过 device 限流）。
func (m *Middleware) Wrap(endpoint string, deviceSNFn func(*http.Request) string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sourceIP := clientIP(r)

		// 1. 认证（强制）。auth 为 nil 视为配置缺失 —— 拒绝，绝不静默放行。
		if m.auth == nil {
			m.logger.Error("ne-direct authenticator not configured, rejecting request",
				zap.String("endpoint", endpoint), zap.String("source_ip", sourceIP))
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		principal, err := m.auth.Authenticate(ctx, r)
		if err != nil || principal == nil {
			m.logger.Warn("ne-direct authentication failed",
				zap.String("endpoint", endpoint),
				zap.String("source_ip", sourceIP),
				zap.Error(err))
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		// 2. 端点级限流。
		if m.endpointLimiter != nil {
			allowed, _, rlErr := m.endpointLimiter.Allow(ctx, endpoint)
			if rlErr != nil {
				m.logger.Warn("ne-direct endpoint ratelimit redis error (fail-open)",
					zap.String("endpoint", endpoint), zap.Error(rlErr))
			}
			if !allowed {
				m.logger.Warn("ne-direct endpoint rate limited",
					zap.String("endpoint", endpoint),
					zap.String("source_ip", sourceIP),
					zap.String("principal", principal.Username))
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
				return
			}
		}

		// 3. 设备级限流（仅当能提取到设备 SN 时）。
		deviceSN := ""
		if deviceSNFn != nil {
			deviceSN = deviceSNFn(r)
		}
		if deviceSN != "" && m.deviceLimiter != nil {
			allowed, _, rlErr := m.deviceLimiter.Allow(ctx, deviceSN)
			if rlErr != nil {
				m.logger.Warn("ne-direct device ratelimit redis error (fail-open)",
					zap.String("device_sn", deviceSN), zap.Error(rlErr))
			}
			if !allowed {
				m.logger.Warn("ne-direct device rate limited",
					zap.String("endpoint", endpoint),
					zap.String("device_sn", deviceSN),
					zap.String("source_ip", sourceIP),
					zap.String("principal", principal.Username))
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests for device"})
				return
			}
		}

		// 4. 审计日志（source_ip + device_sn + principal）。商用网管的合规要求：
		// 所有网元直连访问可追溯到具体操作人 + 源 IP + 目标设备。
		m.logger.Info("ne-direct access",
			zap.String("endpoint", endpoint),
			zap.String("method", r.Method),
			zap.String("source_ip", sourceIP),
			zap.String("device_sn", deviceSN),
			zap.String("principal", principal.Username),
			zap.String("principal_id", principal.UserID.String()),
			zap.Bool("is_super_admin", principal.IsSuperAdmin),
			zap.Time("at", time.Now()))

		// 把 principal 放进 ctx，业务层做 IDOR 归属校验时取用。
		r = r.WithContext(context.WithValue(ctx, principalCtxKey{}, principal))
		next(w, r)
	}
}

// clientIP 从请求中提取调用方源 IP，优先 X-Forwarded-For / X-Real-IP（反代场景），
// 退化到 RemoteAddr。仅用于审计日志，不作为安全判定依据。
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// XFF 可能是逗号分隔列表，取第一个（最初客户端）。
		if idx := strings.IndexByte(xff, ','); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	// RemoteAddr 形如 "ip:port"，去掉端口。
	addr := r.RemoteAddr
	if idx := strings.LastIndexByte(addr, ':'); idx >= 0 {
		return addr[:idx]
	}
	return addr
}
