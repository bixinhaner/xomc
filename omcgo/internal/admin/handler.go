package admin

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/omcgo/omcgo/internal/admin/loginpwd"
)

// Audit log action constants for authentication events.
//
// These are sub-actions of audit.ActionLogin (W3.G.2 / T-0063 charter
// category 1 of 5). The "login_success" / "login_failed" / "logout"
// strings are persisted to audit_logs.action so dashboards can filter
// successful vs failed authentications.
//
// The category is built from audit.ActionLogin so that any future change
// to the canonical constant ripples through to these sub-actions.
var (
	auditActionLoginSuccess = audit.ActionLogin + "_success" // "login_success"
	auditActionLoginFailed  = audit.ActionLogin + "_failed"  // "login_failed"
	auditActionLogout       = "logout"                       // sub-action of audit.ActionLogin
)

// Login rate limiting constants.
const (
	loginRateLimit    = 5               // max login attempts per IP per window
	loginRateWindow   = 1 * time.Minute // rate limit window
	limiterCleanupAge = 5 * time.Minute // remove idle limiters after this duration
)

// ipLimiterEntry holds a per-IP rate limiter and the last time it was accessed.
type ipLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Handler provides HTTP endpoints for admin operations.
type Handler struct {
	service            *AdminService
	captcha            *CaptchaService
	loginGuard         *LoginGuard
	ipGuard            *IPGuard // P0-③ Redis 滑动窗口 IP 限流 + 黑名单（替代旧 in-memory token-bucket）
	roleGroupRepo      RoleDeviceGroupRepository
	apiPermRepo        RoleApiPermissionRepository
	permService        *PermissionService
	apiEndpointService *ApiEndpointService
	loginCipher        *loginpwd.Cipher // 必填：登录类接口密码 RSA-OAEP 解密
	allowPlaintextPwd  bool             // T-0120：允许明文密码 fallback（非 secure context 部署用，默认 false）
	logRepo            LogRepository    // #122：登录成功/失败写 sys_login_logs（nil 安全，未注入则跳过）
	ginRoutes          gin.RoutesInfo   // set after router registration
	logger             *zap.Logger
	loginLimiter       sync.Map // map[string]*ipLimiterEntry
}

// NewHandler creates a new admin Handler.
func NewHandler(service *AdminService, logger *zap.Logger) *Handler {
	h := &Handler{
		service: service,
		logger:  logger.Named("admin-handler"),
	}
	// Start background goroutine to clean up stale IP rate limiters.
	go h.cleanupLoginLimiters()
	return h
}

// SetCaptchaService sets the CAPTCHA service for login protection.
func (h *Handler) SetCaptchaService(cs *CaptchaService) {
	h.captcha = cs
}

// SetLoginGuard sets the brute-force login guard.
func (h *Handler) SetLoginGuard(lg *LoginGuard) {
	h.loginGuard = lg
}

// SetIPGuard sets the Redis-backed IP rate limiter (sliding window + blacklist).
// P0-③：替代 getIPLimiter 的旧 in-memory token-bucket。
// nil 安全：未注入时 Login handler 退化到旧 token-bucket（兜底）。
func (h *Handler) SetIPGuard(ig *IPGuard) {
	h.ipGuard = ig
}

// SetRoleDeviceGroupRepo sets the role-device-group repository.
func (h *Handler) SetRoleDeviceGroupRepo(repo RoleDeviceGroupRepository) {
	h.roleGroupRepo = repo
}

// SetApiPermRepo sets the role API permission repository.
func (h *Handler) SetApiPermRepo(repo RoleApiPermissionRepository) {
	h.apiPermRepo = repo
}

// SetPermissionService sets the data permission service.
func (h *Handler) SetPermissionService(ps *PermissionService) {
	h.permService = ps
}

// SetApiEndpointService sets the API endpoint service.
func (h *Handler) SetApiEndpointService(svc *ApiEndpointService) {
	h.apiEndpointService = svc
}

// SetLogRepository 注入系统日志仓库（#122）。
// 注入后 Login 成功/失败路径异步写 sys_login_logs；nil 安全 — 未注入时跳过。
func (h *Handler) SetLogRepository(repo LogRepository) {
	h.logRepo = repo
}

// SetLoginCipher 注入登录类接口的密码加密 cipher。
// 必须在 Login / ChangePassword / ResetPassword / CreateUser 路由生效前调用。
func (h *Handler) SetLoginCipher(c *loginpwd.Cipher) {
	h.loginCipher = c
}

// SetAllowPlaintextPassword (T-0120) 控制是否接受明文密码登录 / 改密。
// 默认 false；当部署在非 secure context（http://内网IP）crypto.subtle 不可用
// 场景需启用时，启动期由 provider 据 LoginCryptoConfig.AllowPlaintext 注入。
// 明文路径 always 经 audit log 标记 reason=plaintext_login，可合规追溯。
func (h *Handler) SetAllowPlaintextPassword(allow bool) {
	h.allowPlaintextPwd = allow
}

// SetGinRoutes stores gin route info for use in SyncApiEndpoints.
func (h *Handler) SetGinRoutes(routes gin.RoutesInfo) {
	h.ginRoutes = routes
}

// GetGinRoutes returns the stored Gin routes info.
func (h *Handler) GetGinRoutes() gin.RoutesInfo {
	return h.ginRoutes
}

// getIPLimiter returns a rate.Limiter for the given IP, creating one if needed.
func (h *Handler) getIPLimiter(ip string) *rate.Limiter {
	now := time.Now()
	if val, ok := h.loginLimiter.Load(ip); ok {
		entry := val.(*ipLimiterEntry)
		entry.lastSeen = now
		return entry.limiter
	}
	// rate.NewLimiter(rate.Every(window/limit), limit) allows `limit` requests per window.
	limiter := rate.NewLimiter(rate.Every(loginRateWindow/time.Duration(loginRateLimit)), loginRateLimit)
	entry := &ipLimiterEntry{limiter: limiter, lastSeen: now}
	actual, _ := h.loginLimiter.LoadOrStore(ip, entry)
	return actual.(*ipLimiterEntry).limiter
}

// cleanupLoginLimiters periodically removes stale per-IP rate limiters.
func (h *Handler) cleanupLoginLimiters() {
	ticker := time.NewTicker(limiterCleanupAge)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-limiterCleanupAge)
		h.loginLimiter.Range(func(key, value any) bool {
			entry := value.(*ipLimiterEntry)
			if entry.lastSeen.Before(cutoff) {
				h.loginLimiter.Delete(key)
			}
			return true
		})
	}
}

// RegisterAuthRoutes registers public authentication routes (no auth required).
//
// pubKeyHandler 提供 RSA 公钥下发接口，前端登录前先拉公钥再加密密码。
// 必传 —— 没有公钥下发，前端无法构造合法登录请求。
func (h *Handler) RegisterAuthRoutes(rg *gin.RouterGroup, pubKeyHandler *loginpwd.PublicKeyHandler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.GET("/captcha", h.GetCaptcha)
		auth.GET("/public-key", pubKeyHandler.Get)
	}
}

// RegisterAuthenticatedRoutes registers routes that require authentication but not admin privileges.
// E.g. change-password is called by any logged-in user.
func (h *Handler) RegisterAuthenticatedRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/change-password", h.ChangePassword)
	}
}

// RegisterAdminRoutes registers protected admin routes (auth + admin permission required).
func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.GET("", h.ListUsers)
		users.POST("", h.CreateUser)
		users.GET("/import/template", h.DownloadImportTemplate)
		users.POST("/import", h.ImportUsers)
		users.POST("/force-logout", h.ForceLogout)
		users.POST("/assign-roles", h.BatchAssignRoles)
		users.GET("/:id", h.GetUser)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeleteUser)
		users.POST("/:id/copy", h.CopyUser)
		users.POST("/:id/roles", h.AssignRole)
		users.DELETE("/:id/roles/:roleId", h.RemoveRole)
		users.POST("/:id/reset-password", h.ResetPassword)
		users.POST("/:id/lock", h.LockUser)
		users.POST("/:id/unlock", h.UnlockUser)
	}

	roles := rg.Group("/roles")
	{
		roles.GET("", h.ListRoles)
		roles.GET("/all", h.ListAllRoles) // 返回所有角色（不分页，用于下拉框）
		roles.GET("/:id", h.GetRole)
		roles.POST("", h.CreateRole)
		roles.PUT("/:id", h.UpdateRole)
		roles.DELETE("/:id", h.DeleteRole)
		roles.POST("/:id/copy", h.CopyRole) // roles.md §7 P2 #9
		roles.GET("/:id/device-groups", h.GetRoleDeviceGroups)
		roles.PUT("/:id/device-groups", h.SetRoleDeviceGroups)
		roles.GET("/:id/menus", h.GetRoleMenus)
		roles.PUT("/:id/menus", h.SetRoleMenus)
		roles.GET("/:id/api-permissions", h.GetRoleApiPermissions)
		roles.PUT("/:id/api-permissions", h.SetRoleApiPermissions)
		roles.GET("/:id/users", h.ListRoleUsers)
	}

	// API端点管理
	apiEndpoints := rg.Group("/api-endpoints")
	{
		apiEndpoints.GET("", h.ListApiEndpoints)
		apiEndpoints.POST("", h.CreateApiEndpoint)
		apiEndpoints.PUT("/:id", h.UpdateApiEndpoint)
		apiEndpoints.DELETE("/:id", h.DeleteApiEndpoint)
		apiEndpoints.DELETE("/batch", h.BatchDeleteApiEndpoints)
		apiEndpoints.GET("/groups", h.GetApiGroups)
		apiEndpoints.POST("/sync", h.SyncApiEndpoints)
	}

	// Groups endpoint - alias for roles (for frontend compatibility)
	// Frontend UserManagement page expects /admin/groups to return role groups
	groups := rg.Group("/groups")
	{
		groups.GET("", h.ListRoles)
		groups.GET("/:id", h.GetRole)
	}

	// ----- Menu management routes -----
	menus := rg.Group("/menus")
	{
		menus.GET("", h.ListMenus)
		menus.GET("/tree", h.GetMenuTree)
		menus.GET("/user-tree", h.GetUserMenuTree)
		menus.GET("/:id", h.GetMenu)
		menus.POST("", h.CreateMenu)
		menus.PUT("/:id", h.UpdateMenu)
		menus.DELETE("", h.DeleteMenus)
	}

	rg.GET("/permissions", h.ListPermissions)
	rg.GET("/audit-logs", h.ListAuditLogs)
}

// getUserID extracts the user ID from the Gin context.
func getUserID(c *gin.Context) uuid.UUID {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}
