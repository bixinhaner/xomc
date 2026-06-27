package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// userContextWithOperator 把 gin.Context 中的认证信息复制到 context.Context，
// 供 service 层取 operator_id 写审计字段（created_by / updated_by）。
func userContextWithOperator(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	if v, ok := c.Get(CtxKeyUserID); ok {
		ctx = context.WithValue(ctx, CtxKeyUserID, v)
	}
	return ctx
}

// usernameRegex 用户名格式校验：只允许字母、数字、下划线、减号（issue #686）。
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (h *Handler) CreateUser(c *gin.Context) {
	var httpReq CreateUserHTTPRequest
	if err := c.ShouldBindJSON(&httpReq); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// issue #686：用户名格式校验（只允许字母、数字、下划线、减号）。
	if !usernameRegex.MatchString(httpReq.Username) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			errors.New("username can only contain letters, numbers, underscores and hyphens"))
		return
	}

	// issue #649：UseDefaultPassword=true 时跳过密码字段解析与必填校验。
	// service 层会从 sys_configs.security.defaultPasswd 取值并跳过强度校验（设计方案 A，
	// 避免强度规则调高后存量默认密码触发 UX 死结）。
	var plainPwd string
	if !httpReq.UseDefaultPassword {
		// T-0120 双路径密码解析
		var err error
		if httpReq.EncryptedPassword != "" && httpReq.KeyID != "" {
			if h.loginCipher == nil {
				commonerrors.AbortWithError(c, http.StatusInternalServerError,
					errors.New("login password cipher not configured"))
				return
			}
			plainPwd, err = h.loginCipher.Decrypt(c.Request.Context(), httpReq.KeyID, httpReq.EncryptedPassword)
			if err != nil {
				commonerrors.AbortWithError(c, http.StatusBadRequest,
					fmt.Errorf("decrypt password: %w", err))
				return
			}
		} else if httpReq.Password != "" {
			if !h.allowPlaintextPwd {
				commonerrors.AbortWithError(c, http.StatusBadRequest,
					errors.New("plaintext password is disabled; please use encrypted_password+key_id or deploy TLS"))
				return
			}
			plainPwd = httpReq.Password
		} else {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				errors.New("missing password: provide encrypted_password+key_id, password, or set use_default_password=true"))
			return
		}
		if len(plainPwd) < 6 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				errors.New("password must be at least 6 characters"))
			return
		}
	}

	req := CreateUserRequest{
		Username:           httpReq.Username,
		Password:           plainPwd,
		UseDefaultPassword: httpReq.UseDefaultPassword,
		DisplayName:        httpReq.DisplayName,
		Email:              httpReq.Email,
		Phone:              httpReq.Phone,
		Description:        httpReq.Description,
		ExpireAt:           httpReq.ExpireAt,
		RoleIDs:            httpReq.RoleIDs,
	}
	user, err := h.service.CreateUser(userContextWithOperator(c), req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, user)
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

	response.OK(c, user)
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

	user, err := h.service.UpdateUser(userContextWithOperator(c), id, req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OK(c, user)
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

	response.OK(c, nil)
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

	response.OK(c, result)
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

	response.OKWithMsg(c, nil, "role assigned")
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

	response.OKWithMsg(c, nil, "role removed")
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

	ctx := c.Request.Context()

	// issue #649：UseDefaultPassword=true 时跳过密码字段解析与必填校验。
	// service 层会从 sys_configs.security.defaultPasswd 取值并跳过强度校验。
	var plainPwd string
	if !req.UseDefaultPassword {
		// T-0120 双路径（reuse outer err declared at func top）
		if req.EncryptedNewPassword != "" && req.KeyID != "" {
			if h.loginCipher == nil {
				commonerrors.AbortWithError(c, http.StatusInternalServerError,
					errors.New("login password cipher not configured"))
				return
			}
			plainPwd, err = h.loginCipher.Decrypt(ctx, req.KeyID, req.EncryptedNewPassword)
			if err != nil {
				commonerrors.AbortWithError(c, http.StatusBadRequest,
					fmt.Errorf("decrypt new password: %w", err))
				return
			}
		} else if req.NewPassword != "" {
			if !h.allowPlaintextPwd {
				commonerrors.AbortWithError(c, http.StatusBadRequest,
					errors.New("plaintext password is disabled; please use encrypted_new_password+key_id or deploy TLS"))
				return
			}
			plainPwd = req.NewPassword
		} else {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				errors.New("missing password: provide encrypted_new_password+key_id, new_password, or set use_default_password=true"))
			return
		}
		if len(plainPwd) < 6 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				errors.New("new password must be at least 6 characters"))
			return
		}
	}

	if err := h.service.ResetPassword(ctx, id, ResetPasswordRequest{
		NewPassword:        plainPwd,
		UseDefaultPassword: req.UseDefaultPassword,
	}); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OKWithMsg(c, nil, "password reset")
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

	response.OKWithMsg(c, nil, "user locked")
}

// ForceLogoutRequest 是批量强制下线接口的请求载荷。
type ForceLogoutRequest struct {
	UserIDs []uuid.UUID `json:"user_ids" binding:"required,min=1"`
}

// BatchAssignRolesRequest 用于批量替换一组用户的角色集（PRD §5.5）。
type BatchAssignRolesRequest struct {
	UserIDs []uuid.UUID `json:"user_ids" binding:"required,min=1"`
	RoleIDs []uuid.UUID `json:"role_ids" binding:"required"`
}

// BatchAssignRoles 把请求体里的 RoleIDs 整体替换为每个 UserIDs 当前的角色集。
func (h *Handler) BatchAssignRoles(c *gin.Context) {
	var req BatchAssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.service.BatchAssignRoles(c.Request.Context(), req.UserIDs, req.RoleIDs); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}
	response.OK(c, gin.H{"updated": len(req.UserIDs)})
}

// CopyUser 复制源用户为新账号；返回新用户对象 + 临时密码（仅这次响应可见）。
func (h *Handler) CopyUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	user, tempPwd, err := h.service.CopyUser(c.Request.Context(), id)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, gin.H{
		"user":          user,
		"temp_password": tempPwd,
	})
}

// ForceLogout 强制指定用户下线（批量）。所有受影响 token 在下一次中间件校验时失败。
func (h *Handler) ForceLogout(c *gin.Context) {
	var req ForceLogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.ForceLogout(c.Request.Context(), req.UserIDs); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OK(c, gin.H{"revoked": len(req.UserIDs)})
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

	response.OKWithMsg(c, nil, "user unlocked")
}
