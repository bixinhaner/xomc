package admin

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Audit log action constants for authentication events.
const (
	auditActionLoginSuccess = "login_success"
	auditActionLoginFailed  = "login_failed"
	auditActionLogout       = "logout"
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
	roleGroupRepo      RoleDeviceGroupRepository
	apiPermRepo        RoleApiPermissionRepository
	permService        *PermissionService
	apiEndpointService *ApiEndpointService
	ginRoutes          gin.RoutesInfo // set after router registration
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

// SetGinRoutes stores gin route info for use in SyncApiEndpoints.
func (h *Handler) SetGinRoutes(routes gin.RoutesInfo) {
	h.ginRoutes = routes
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
func (h *Handler) RegisterAuthRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.GET("/captcha", h.GetCaptcha)
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
		users.GET("/:id", h.GetUser)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeleteUser)
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

	// ----- Role menu assignment -----
	roles.GET("/:id/menus", h.GetRoleMenus)
	roles.PUT("/:id/menus", h.SetRoleMenus)

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
