package admin

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components/logger"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

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

	// TODO: captcha verification temporarily disabled
	// // Brute-force protection: check if CAPTCHA is required for this username.
	// if h.loginGuard != nil && h.captcha != nil {
	// 	if h.loginGuard.RequiresCaptcha(ctx, req.Username) {
	// 		if req.CaptchaID == "" || req.CaptchaAnswer == "" {
	// 			c.AbortWithStatusJSON(http.StatusPreconditionRequired, gin.H{
	// 				"code":    7010,
	// 				"message": "captcha required due to multiple failed attempts",
	// 			})
	// 			return
	// 		}
	// 		if !h.captcha.Verify(ctx, req.CaptchaID, req.CaptchaAnswer) {
	// 			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
	// 				"code":    7011,
	// 				"message": "invalid captcha answer",
	// 			})
	// 			return
	// 		}
	// 	}
	// }

	tokenPair, err := h.service.Login(ctx, req.Username, req.Password)
	if err != nil {
		reason := classifyLoginFailure(err)

		log.Warn("login failed",
			zap.String("username", req.Username),
			zap.String("ip", clientIP),
			zap.String("reason", reason),
		)

		// W3.G.2 ActionLogin / category 1 of 5: handled by recordAuthAuditLog,
		// which writes "login_failed" — a sub-action of audit.ActionLogin —
		// directly through the AuditRepository. Avoid double-emission via
		// audit.LogAsync; both paths share the same repo.
		h.recordAuthAuditLog(auditActionLoginFailed, req.Username, nil, clientIP, userAgent, reason)

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

	if h.loginGuard != nil {
		h.loginGuard.Reset(ctx, req.Username)
	}

	log.Info("login success",
		zap.String("username", req.Username),
		zap.String("ip", clientIP),
	)

	// W3.G.2 ActionLogin / category 1 of 5: see comment in failure branch.
	h.recordAuthAuditLog(auditActionLoginSuccess, req.Username, nil, clientIP, userAgent, "")

	response.OK(c, tokenPair)
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

	response.OK(c, challenge)
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

	response.OK(c, tokenPair)
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

	response.OK(c, user)
}

// SwitchRole handles POST /auth/switch-role
func (h *Handler) SwitchRole(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}

	var req struct {
		RoleID uuid.UUID `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tokenPair, err := h.service.SwitchRole(c.Request.Context(), userID, req.RoleID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, tokenPair, "角色切换成功")
}

// GetUserMenusByRole handles GET /auth/menus — returns menus for current role
func (h *Handler) GetUserMenusByRole(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}

	// Get current role from JWT claims if available, otherwise fall back to user menus
	claims, exists := c.Get("claims")
	if exists {
		if cl, ok := claims.(*Claims); ok && cl.CurrentRoleID != nil {
			menus, err := h.service.GetUserMenuTreeByRole(c.Request.Context(), userID, *cl.CurrentRoleID)
			if err != nil {
				commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
				return
			}
			response.OKWithMsg(c, menus, "查询成功")
			return
		}
	}

	// Fallback: all user menus
	menus, err := h.service.GetUserMenuTree(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, menus, "查询成功")
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

// ChangePassword handles POST /api/v1/auth/change-password.
// The user must be authenticated; it verifies the old password before updating.
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), userID, req); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OKWithMsg(c, nil, "password changed successfully")
}
