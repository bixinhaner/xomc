package admin

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/omcgo/omcgo/internal/core/components/logger"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
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
	service       *AdminService
	captcha       *CaptchaService
	loginGuard    *LoginGuard
	roleGroupRepo RoleDeviceGroupRepository
	permService   *PermissionService
	logger        *zap.Logger
	loginLimiter  sync.Map // map[string]*ipLimiterEntry
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

// SetPermissionService sets the data permission service.
func (h *Handler) SetPermissionService(ps *PermissionService) {
	h.permService = ps
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
	}

	// Groups endpoint - alias for roles (for frontend compatibility)
	// Frontend UserManagement page expects /admin/groups to return role groups
	groups := rg.Group("/groups")
	{
		groups.GET("", h.ListRoles)
		groups.GET("/:id", h.GetRole)
	}

	rg.GET("/permissions", h.ListPermissions)
	rg.GET("/audit-logs", h.ListAuditLogs)
}

func (h *Handler) Login(c *gin.Context) {
	clientIP := c.ClientIP()

	// Per-IP login rate limiting: reject if too many attempts.
	if limiter := h.getIPLimiter(clientIP); !limiter.Allow() {
		logger.L(c.Request.Context()).Warn("login rate limited",
			zap.String("ip", clientIP),
		)
		commonerrors.AbortWithError(c, http.StatusTooManyRequests,
			errors.New("too many login attempts, please try again later"))
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	ctx := c.Request.Context()
	log := logger.L(ctx)
	userAgent := c.Request.UserAgent()

	// Brute-force protection: check if CAPTCHA is required for this username.
	if h.loginGuard != nil && h.captcha != nil {
		if h.loginGuard.RequiresCaptcha(ctx, req.Username) {
			if req.CaptchaID == "" || req.CaptchaAnswer == "" {
				c.AbortWithStatusJSON(http.StatusPreconditionRequired, gin.H{
					"code":    7010,
					"message": "captcha required due to multiple failed attempts",
				})
				return
			}
			if !h.captcha.Verify(ctx, req.CaptchaID, req.CaptchaAnswer) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"code":    7011,
					"message": "invalid captcha answer",
				})
				return
			}
		}
	}

	tokenPair, err := h.service.Login(ctx, req.Username, req.Password)
	if err != nil {
		// Classify the failure reason for audit logging
		reason := classifyLoginFailure(err)

		log.Warn("login failed",
			zap.String("username", req.Username),
			zap.String("ip", clientIP),
			zap.String("reason", reason),
		)

		h.recordAuthAuditLog(auditActionLoginFailed, req.Username, nil, clientIP, userAgent, reason)

		// Record brute-force failure and potentially lock account
		if h.loginGuard != nil {
			count, _ := h.loginGuard.RecordFailure(ctx, req.Username)
			if h.loginGuard.ShouldLock(count) {
				h.service.LockUserByUsername(ctx, req.Username, h.loginGuard.LockDuration())
				log.Warn("account auto-locked due to brute force",
					zap.String("username", req.Username),
					zap.Int64("failed_count", count),
				)
			}
		}

		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	// On successful login, reset brute-force counter
	if h.loginGuard != nil {
		h.loginGuard.Reset(ctx, req.Username)
	}

	log.Info("login success",
		zap.String("username", req.Username),
		zap.String("ip", clientIP),
	)

	h.recordAuthAuditLog(auditActionLoginSuccess, req.Username, nil, clientIP, userAgent, "")

	c.JSON(http.StatusOK, tokenPair)
}

// GetCaptcha handles GET /api/v1/auth/captcha — generates a new CAPTCHA challenge.
func (h *Handler) GetCaptcha(c *gin.Context) {
	if h.captcha == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("captcha service not configured"))
		return
	}

	challenge, err := h.captcha.Generate(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, challenge)
}

// classifyLoginFailure maps internal login errors to human-readable failure reasons.
func classifyLoginFailure(err error) string {
	switch {
	case errors.Is(err, errLoginUserNotFound):
		return "user_not_found"
	case errors.Is(err, errLoginAccountDisabled):
		return "account_disabled"
	case errors.Is(err, errLoginWrongPassword):
		return "wrong_password"
	default:
		return "unknown"
	}
}

// recordAuthAuditLog writes an authentication audit log entry asynchronously.
// It uses a background context to avoid cancellation when the HTTP request completes.
func (h *Handler) recordAuthAuditLog(action, username string, userID *uuid.UUID, ip, userAgent, reason string) {
	details := map[string]interface{}{}
	if reason != "" {
		details["reason"] = reason
	}

	auditLog := &AuditLog{
		UserID:    userID,
		Username:  username,
		Action:    action,
		Resource:  "auth",
		Details:   details,
		IPAddress: ip,
		UserAgent: userAgent,
	}

	// Fire and forget — audit logging must not block the login flow
	go func() {
		if err := h.service.auditRepo.Create(context.Background(), auditLog); err != nil {
			h.logger.Error("failed to write auth audit log",
				zap.String("action", action),
				zap.String("username", username),
				zap.Error(err),
			)
		}
	}()
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tokenPair, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, tokenPair)
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), id)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), id, req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteUser(c.Request.Context(), id); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) ListUsers(c *gin.Context) {
	var filter UserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListUsers(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) AssignRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.AssignRole(c.Request.Context(), userID, req.RoleID); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role assigned"})
}

func (h *Handler) RemoveRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.RemoveRole(c.Request.Context(), userID, roleID); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role removed"})
}

func (h *Handler) ListRoles(c *gin.Context) {
	var filter RoleFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListRolesPaginated(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListAllRoles 返回所有角色（不分页，用于下拉框）
func (h *Handler) ListAllRoles(c *gin.Context) {
	roles, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, roles)
}

// Me handles GET /api/v1/auth/me — returns current authenticated user info.
func (h *Handler) Me(c *gin.Context) {
	userIDVal, exists := c.Get(CtxKeyUserID)
	if !exists {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, commonerrors.ErrInvalidInput)
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), id, req.NewPassword); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset"})
}

func (h *Handler) LockUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.LockUser(c.Request.Context(), id); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user locked"})
}

func (h *Handler) UnlockUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.UnlockUser(c.Request.Context(), id); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user unlocked"})
}

func (h *Handler) GetRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	role, err := h.service.GetRole(c.Request.Context(), id)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *Handler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	role, err := h.service.CreateRole(c.Request.Context(), req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusCreated, role)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	role, err := h.service.UpdateRole(c.Request.Context(), id, req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *Handler) DeleteRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteRole(c.Request.Context(), id); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) ListPermissions(c *gin.Context) {
	perms, err := h.service.ListAllPermissions(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, perms)
}

// GetRoleDeviceGroups handles GET /roles/:id/device-groups.
func (h *Handler) GetRoleDeviceGroups(c *gin.Context) {
	if h.roleGroupRepo == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("role device group service not configured"))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	groupIDs, err := h.roleGroupRepo.GetGroupIDs(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"group_ids": groupIDs})
}

// SetRoleDeviceGroups handles PUT /roles/:id/device-groups.
func (h *Handler) SetRoleDeviceGroups(c *gin.Context) {
	if h.roleGroupRepo == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("role device group service not configured"))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req struct {
		GroupIDs []uuid.UUID `json:"group_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.roleGroupRepo.SetGroupIDs(c.Request.Context(), id, req.GroupIDs); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	// Invalidate permission cache for this role.
	if h.permService != nil {
		h.permService.InvalidateRoleCache(c.Request.Context(), id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "device groups updated"})
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
	var filter AuditLogFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.auditRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ==================== Menu Handlers ====================

// CreateMenu handles POST /menus.
func (h *Handler) CreateMenu(c *gin.Context) {
	var req CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operatorID := getUserID(c)

	menu, err := h.service.CreateMenu(c.Request.Context(), req, operatorID)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusCreated, menu)
}

// GetMenu handles GET /menus/:id.
func (h *Handler) GetMenu(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	menu, err := h.service.GetMenu(c.Request.Context(), id)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, menu)
}

// ListMenus handles GET /menus.
func (h *Handler) ListMenus(c *gin.Context) {
	var filter MenuFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListMenus(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateMenu handles PUT /menus/:id.
func (h *Handler) UpdateMenu(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operatorID := getUserID(c)

	if err := h.service.UpdateMenu(c.Request.Context(), id, &req, operatorID); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteMenus handles DELETE /menus.
func (h *Handler) DeleteMenus(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	ids := make([]uuid.UUID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		ids[i] = id
	}

	if err := h.service.DeleteMenus(c.Request.Context(), ids); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetMenuTree handles GET /menus/tree.
func (h *Handler) GetMenuTree(c *gin.Context) {
	statusStr := c.Query("status")
	var status *MenuStatus
	if statusStr != "" {
		s := MenuStatus(statusStr)
		status = &s
	}

	menus, err := h.service.GetMenuTree(c.Request.Context(), status)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": menus})
}

// GetUserMenuTree handles GET /menus/user.
func (h *Handler) GetUserMenuTree(c *gin.Context) {
	userID := getUserID(c)

	menus, err := h.service.GetUserMenuTree(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": menus})
}

// SetRoleMenus handles PUT /roles/:id/menus.
func (h *Handler) SetRoleMenus(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req SetRoleMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operatorID := getUserID(c)

	if err := h.service.SetRoleMenus(c.Request.Context(), roleID, req.MenuIDs, operatorID); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetRoleMenus handles GET /roles/:id/menus.
func (h *Handler) GetRoleMenus(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	menus, err := h.service.GetRoleMenus(c.Request.Context(), roleID)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": menus})
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
