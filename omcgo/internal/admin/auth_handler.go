package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components/logger"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

func (h *Handler) Login(c *gin.Context) {
	clientIP := c.ClientIP()
	ctxEarly := c.Request.Context()

	// ⑥ IP 限流（sys_configs security.limitMinus / limitCount / limitTimes）：
	// 优先用 Redis 实现的 IPGuard（滑动窗口 + 黑名单 TTL，跨进程一致）；
	// 未注入时退化到旧 in-memory token-bucket 兜底（单进程，仅控总请求速率）。
	if h.ipGuard != nil {
		allowed, remaining, _ := h.ipGuard.CheckAllowed(ctxEarly, clientIP)
		if !allowed {
			logger.L(ctxEarly).Warn("ip locked",
				zap.String("ip", clientIP),
				zap.Duration("retry_after", remaining),
			)
			c.Header("Retry-After", fmt.Sprintf("%.0f", remaining.Seconds()))
			commonerrors.AbortWithError(c, http.StatusTooManyRequests,
				fmt.Errorf("ip locked: retry after %s", remaining.Round(time.Second)))
			return
		}
	} else {
		// 兜底：进程内 token-bucket，节奏由硬编码 loginRateLimit/Window 控制
		if limiter := h.getIPLimiter(clientIP); !limiter.Allow() {
			logger.L(ctxEarly).Warn("login rate limited",
				zap.String("ip", clientIP),
			)
			commonerrors.AbortWithError(c, http.StatusTooManyRequests,
				errors.New("too many login attempts, please try again later"))
			return
		}
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	ctx := c.Request.Context()
	log := logger.L(ctx)
	userAgent := c.Request.UserAgent()

	// ⑤ 图形验证码（sys_configs security.verifyEnable + attemptTimes）：
	//   LoginGuard.RequiresCaptcha 内部已读 verifyEnable 总开关；只有当
	//   开关打开 AND 失败次数 >= attemptTimes 时才要求 CAPTCHA。
	//   captchaService nil 时跳过（部署不带验证码服务的容错降级）。
	if h.loginGuard != nil && h.captcha != nil && h.loginGuard.RequiresCaptcha(ctx, req.Username) {
		if req.CaptchaID == "" || req.CaptchaAnswer == "" {
			h.recordAuthAuditLog(auditActionLoginFailed, req.Username, nil, clientIP, userAgent, "captcha_required")
			c.AbortWithStatusJSON(http.StatusPreconditionRequired, gin.H{
				"ret":  0,
				"msg":  "captcha required due to multiple failed attempts",
				"data": nil,
				"biz_code": 7010,
			})
			return
		}
		if !h.captcha.Verify(ctx, req.CaptchaID, req.CaptchaAnswer) {
			h.recordAuthAuditLog(auditActionLoginFailed, req.Username, nil, clientIP, userAgent, "captcha_invalid")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"ret":  0,
				"msg":  "invalid captcha answer",
				"data": nil,
				"biz_code": 7011,
			})
			return
		}
	}

	// T-0120 双路径密码解析：
	//   - 加密路径：EncryptedPassword + KeyID 同时非空 → loginCipher.Decrypt
	//   - 明文路径（allowPlaintextPwd=true 时启用）：Password 非空 → 直接采用
	//   - 都不满足 → 400 missing password
	// 明文路径成功的请求在审计日志里 reason=plaintext_login 留痕。
	var (
		plainPwd        string
		pwdSourceReason string // 仅 plaintext 时非空，进审计
		err             error
	)
	if req.EncryptedPassword != "" && req.KeyID != "" {
		if h.loginCipher == nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				errors.New("login password cipher not configured"))
			return
		}
		plainPwd, err = h.loginCipher.Decrypt(ctx, req.KeyID, req.EncryptedPassword)
		if err != nil {
			log.Warn("login decrypt failed",
				zap.String("username", req.Username),
				zap.String("ip", clientIP),
				zap.Error(err),
			)
			// 对外仅以 401 暴露失败原因，不区分"密钥错"/"重放"/"过期"，避免给攻击者反馈。
			h.recordAuthAuditLog(auditActionLoginFailed, req.Username, nil, clientIP, userAgent, "decrypt_failed")
			commonerrors.AbortWithError(c, http.StatusUnauthorized,
				commonerrors.NewBusinessError(7003, "登录凭据无效，请重试", err))
			return
		}
	} else if req.Password != "" {
		if !h.allowPlaintextPwd {
			log.Warn("plaintext login rejected (allow_plaintext=false)",
				zap.String("username", req.Username),
				zap.String("ip", clientIP),
			)
			h.recordAuthAuditLog(auditActionLoginFailed, req.Username, nil, clientIP, userAgent, "plaintext_disabled")
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				commonerrors.NewBusinessError(7004, "明文密码登录已禁用，请使用 HTTPS 或 localhost 访问", nil))
			return
		}
		plainPwd = req.Password
		pwdSourceReason = "plaintext_login" // 进审计；service.Login 成功后再写 success 行
		log.Warn("plaintext login accepted (allow_plaintext=true; non-secure context)",
			zap.String("username", req.Username),
			zap.String("ip", clientIP),
		)
	} else {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			errors.New("missing password: provide encrypted_password+key_id or (if allow_plaintext) password"))
		return
	}

	tokenPair, err := h.service.Login(ctx, req.Username, plainPwd)
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
			// 阈值与锁定时长从 sys_configs (category='security') 读：
			//   sumTimes   → ShouldLock 阈值
			//   unlockMinu → LockDuration（分钟）
			// 读失败或未注入 SysConfigQuerier 时 LoginGuard 内部退化到 default 常量。
			if h.loginGuard.ShouldLock(ctx, count) {
				h.service.LockUserByUsername(ctx, req.Username, h.loginGuard.LockDuration(ctx))
				log.Warn("account auto-locked due to brute force",
					zap.String("username", req.Username),
					zap.Int64("failed_count", count),
				)
			}
		}
		// P0-③ IP 限流：登录失败时同步 +1 IP 计数器；达到 limitCount 后该 IP
		// 立即拉黑 limitTimes 分钟。failure 不阻塞 — 单纯计数。
		if h.ipGuard != nil {
			if err := h.ipGuard.RecordFailure(ctx, clientIP); err != nil {
				log.Warn("ip guard record failure", zap.String("ip", clientIP), zap.Error(err))
			}
		}

		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, loginErrorToFriendlyError(err))
		return
	}

	if h.loginGuard != nil {
		h.loginGuard.Reset(ctx, req.Username)
	}
	// 登录成功清零该 IP 失败计数器（黑名单 TTL 不动 — 锁了就锁了）
	if h.ipGuard != nil {
		h.ipGuard.Reset(ctx, clientIP)
	}

	log.Info("login success",
		zap.String("username", req.Username),
		zap.String("ip", clientIP),
		zap.String("pwd_source", func() string {
			if pwdSourceReason != "" {
				return pwdSourceReason
			}
			return "encrypted"
		}()),
	)

	// W3.G.2 ActionLogin / category 1 of 5: see comment in failure branch.
	// T-0120：plaintext 路径在 reason 标记 plaintext_login 供合规追溯。
	h.recordAuthAuditLog(auditActionLoginSuccess, req.Username, nil, clientIP, userAgent, pwdSourceReason)

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

// loginErrorToFriendlyError 把内部登录错误翻译为面向用户的友好消息。
//
// 安全考虑：合并"用户不存在"和"密码错误"为同一文案——避免向攻击者透露
// 用户名是否存在（防枚举）。直接返 *BusinessError 让 AbortWithError 走
// BizCode + Message 分支，不再泄露 errors.Wrap 链字符串。
func loginErrorToFriendlyError(err error) error {
	switch {
	case errors.Is(err, errLoginUserNotFound),
		errors.Is(err, errLoginWrongPassword):
		return commonerrors.NewBusinessError(7001, "用户名或密码错误", err)
	case errors.Is(err, errLoginAccountDisabled):
		return commonerrors.NewBusinessError(7002, "账号已被禁用，请联系管理员", err)
	case errors.Is(err, commonerrors.ErrUnauthorized):
		return commonerrors.NewBusinessError(7000, "用户名或密码错误", err)
	default:
		return commonerrors.NewBusinessError(7099, "登录失败，请稍后重试", err)
	}
}

// ChangePassword handles POST /api/v1/auth/change-password.
// The user must be authenticated; it verifies the old password before updating.
//
// 旧/新密码均需 RSA-OAEP 加密传输（共用一次 keyID）。
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	var httpReq ChangePasswordHTTPRequest
	if err := c.ShouldBindJSON(&httpReq); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	ctx := c.Request.Context()

	// T-0120 双路径：加密 vs 明文 fallback
	var (
		oldPlain string
		newPlain string
		err      error
	)
	if httpReq.EncryptedOldPassword != "" && httpReq.EncryptedNewPassword != "" && httpReq.KeyID != "" {
		if h.loginCipher == nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				errors.New("login password cipher not configured"))
			return
		}
		oldPlain, err = h.loginCipher.Decrypt(ctx, httpReq.KeyID, httpReq.EncryptedOldPassword)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("decrypt old password: %w", err))
			return
		}
		newPlain, err = h.loginCipher.Decrypt(ctx, httpReq.KeyID, httpReq.EncryptedNewPassword)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("decrypt new password: %w", err))
			return
		}
	} else if httpReq.OldPassword != "" && httpReq.NewPassword != "" {
		if !h.allowPlaintextPwd {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				errors.New("plaintext password change is disabled; please use encrypted_*+key_id or deploy TLS"))
			return
		}
		oldPlain = httpReq.OldPassword
		newPlain = httpReq.NewPassword
	} else {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			errors.New("missing password: provide encrypted_*+key_id or (if allow_plaintext) old_password+new_password"))
		return
	}
	if len(newPlain) < 6 {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			errors.New("new password must be at least 6 characters"))
		return
	}

	req := ChangePasswordRequest{OldPassword: oldPlain, NewPassword: newPlain}
	if err := h.service.ChangePassword(ctx, userID, req); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OKWithMsg(c, nil, "password changed successfully")
}
