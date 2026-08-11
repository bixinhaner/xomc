package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/omcgo/omcgo/internal/admin/audit"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// operatorIDFromContext 从 gin/middleware 注入的 ctx 取出当前操作者用户 ID。
// 用于审计字段 created_by / updated_by；未注入或类型不匹配时返回 nil。
func operatorIDFromContext(ctx context.Context) *uuid.UUID {
	v := ctx.Value(CtxKeyUserID)
	if v == nil {
		return nil
	}
	if id, ok := v.(uuid.UUID); ok {
		return &id
	}
	return nil
}

// generateRandomPassword 返回长度为 n 的 url-safe base64 字符串作为临时密码。
// 实际熵源是 crypto/rand，前端必须提示管理员立刻让用户重置。
func generateRandomPassword(n int) (string, error) {
	if n < 8 {
		n = 8
	}
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:n], nil
}

// Login failure sentinel errors used for audit log classification.
// These are internal to the service and handler; callers receive generic ErrUnauthorized/ErrForbidden.
var (
	errLoginUserNotFound    = fmt.Errorf("user not found")
	errLoginAccountDisabled = fmt.Errorf("account disabled")
	errLoginWrongPassword   = fmt.Errorf("wrong password")
)

// User source protection errors. Wrap commonerrors.ErrAlreadyExists so HTTPStatusFromError → 409 Conflict.
// 决策依据：docs/prd/system/users.md §1.1 / §4.1 / §11.2。
var (
	ErrBuiltInUserProtected = fmt.Errorf("built-in user cannot be deleted or disabled: %w", commonerrors.ErrAlreadyExists)
	ErrLDAPPasswordExternal = fmt.Errorf("LDAP user password is managed externally: %w", commonerrors.ErrAlreadyExists)
)

// builtInRoleIDs 固定内置角色 UUID 白名单（admin / operator / viewer）。
// 删除保护的兜底口径：即便某环境 roles.is_system 标记被错误改写（issue #136 的
// operator/viewer 历史误标 false），这三个内置角色仍不可删除。与 seed/000036
// 把 is_system 收紧为 true 形成双重保护（白名单 + is_system 双重判定）。
var (
	builtInAdminRoleID    = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	builtInOperatorRoleID = uuid.MustParse("10000000-0000-0000-0000-000000000002")
	builtInViewerRoleID   = uuid.MustParse("10000000-0000-0000-0000-000000000003")

	builtInRoleIDs = map[uuid.UUID]struct{}{
		builtInAdminRoleID:    {},
		builtInOperatorRoleID: {},
		builtInViewerRoleID:   {},
	}
)

// isBuiltInRole 判定给定角色 ID 是否为固定内置角色。
func isBuiltInRole(id uuid.UUID) bool {
	_, ok := builtInRoleIDs[id]
	return ok
}

// PermissionInvalidator 抽象 PermissionService 的失效操作，便于注入与测试。
// 详见 PRD docs/prd/system/users.md §10 DoD「数据权限缓存一致性」。
type PermissionInvalidator interface {
	InvalidateUserCache(ctx context.Context, userID uuid.UUID) error
}

// roleCopyBindingRepository is the narrow association interface required by
// CopyRole. Keeping it separate from RoleRepository avoids coupling ordinary
// role CRUD tests and callers to every device-group/API binding operation.
type roleCopyBindingRepository interface {
	GetDeviceGroupData(ctx context.Context, roleID uuid.UUID) (*RoleDeviceGroupData, error)
	SetDeviceGroupData(ctx context.Context, roleID uuid.UUID, data RoleDeviceGroupData) error
	GetRoleApiEndpointIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	SetRoleApiEndpoints(ctx context.Context, roleID uuid.UUID, endpointIDs []uuid.UUID) error
}

const copyRoleCleanupTimeout = 5 * time.Second

// AdminService provides user management, authentication, and RBAC functionality.
// LicenseFeatureGate 判定菜单的 feature_code 是否被当前系统 license 授权。
// 实现方（license.SystemLicenseService）按 code 派生路径后查授权树。菜单可见性
// fail-open：无 license / 查询异常时返回全集（不隐藏，避免锁死 UI）；设备级
// fail-closed 走 license.Enforcer，与此正交。
type LicenseFeatureGate interface {
	// FilterAuthorized 返回 codes 中被授权的子集（仅在 HasActiveLicense=true 时调用）。
	FilterAuthorized(ctx context.Context, codes []string) ([]string, error)
	// HasActiveLicense 报告当前是否已配置系统 license。
	// 无 license（GetCurrent 404）→ (false, nil)；瞬时错误 → (false, err)。
	HasActiveLicense(ctx context.Context) (bool, error)
}

type AdminService struct {
	userRepo         UserRepository
	roleRepo         RoleRepository
	roleCopyBindings roleCopyBindingRepository
	menuRepo         MenuRepository
	auditRepo        AuditRepository
	jwt              *JWTService
	revoker          *TokenRevoker
	permInvalidator  PermissionInvalidator
	metrics          *AdminMetrics
	policy           *SecurityPolicy // 可选；nil 时单点登录 / 密码策略等走 default
	featureGate      LicenseFeatureGate // 可选；nil 时菜单不做 license feature 过滤
	logger           *zap.Logger
}

// NewAdminService creates a new AdminService.
//
// As a side effect, this registers the package-level audit sink (see
// internal/admin/audit) so cross-module callers can use audit.Log without
// directly depending on AdminService. The first AdminService constructed
// wins; subsequent calls overwrite the sink (matters mainly for tests).
func NewAdminService(
	userRepo UserRepository,
	roleRepo RoleRepository,
	menuRepo MenuRepository,
	auditRepo AuditRepository,
	jwtService *JWTService,
	logger *zap.Logger,
) *AdminService {
	if auditRepo != nil {
		audit.SetDefault(NewAuditSink(auditRepo))
	}
	if logger != nil {
		audit.SetFallbackLogger(logger.Named("audit"))
	}
	svc := &AdminService{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		menuRepo:  menuRepo,
		auditRepo: auditRepo,
		jwt:       jwtService,
		logger:    logger.Named("admin"),
	}
	if bindingRepo, ok := roleRepo.(roleCopyBindingRepository); ok {
		svc.roleCopyBindings = bindingRepo
	}
	return svc
}

// SetTokenRevoker 注入 token 撤销器，启用强制下线能力。
// 不调用时 ForceLogout 退化为 no-op（仅记录日志），以便单元测试无需 Redis 也能跑。
func (s *AdminService) SetTokenRevoker(r *TokenRevoker) {
	s.revoker = r
}

// SetSecurityPolicy 注入共享安全策略；用于 Login / ChangePassword 等流程读取
// sys_configs (category='security') 的运行时配置（如单点登录开关、密码强度）。
func (s *AdminService) SetSecurityPolicy(p *SecurityPolicy) {
	s.policy = p
}

// SetLicenseFeatureGate 注入 license 特性闸门，启用菜单按 license feature 过滤。
// 不调用（nil）时 GetUserMenuTreeByRole 不做 feature 过滤，菜单全可见。
func (s *AdminService) SetLicenseFeatureGate(g LicenseFeatureGate) {
	s.featureGate = g
}

// policySnapshot 返当前生效的 SecurityPolicy 快照；未注入时返 default。
// 让 CreateUser / Login / ChangePassword 等多处一致地读取策略。
func (s *AdminService) policySnapshot(ctx context.Context) *securityPolicyValues {
	if s.policy == nil {
		return defaultPolicy()
	}
	return s.policy.Get(ctx)
}

// SetPermissionInvalidator 注入权限缓存失效器；用户角色 / 运营商 / 删除等写操作后会调用。
// 详见 PRD docs/prd/system/users.md §10 DoD。
func (s *AdminService) SetPermissionInvalidator(p PermissionInvalidator) {
	s.permInvalidator = p
}

// SetMetrics 注入 admin 模块指标；未注入时不影响业务。
func (s *AdminService) SetMetrics(m *AdminMetrics) {
	s.metrics = m
}

// invalidatePermCache 失败仅记 ERROR 日志 + 递增计数器，不阻塞业务返回。
// 计数器：omc_perm_cache_invalidate_failed_total（PRD §10 DoD）。
func (s *AdminService) invalidatePermCache(ctx context.Context, userID uuid.UUID) {
	if s.permInvalidator == nil {
		return
	}
	if err := s.permInvalidator.InvalidateUserCache(ctx, userID); err != nil {
		s.logger.Error("invalidate permission cache failed",
			zap.String("user_id", userID.String()), zap.Error(err))
		if s.metrics != nil {
			s.metrics.PermCacheInvalidateFailed.Inc()
		}
	}
}

// InvalidatePermCacheByRole 遍历该角色的所有用户，逐一调 invalidatePermCache。
// 用于 PRD docs/prd/system/roles.md §10 DoD：角色侧写操作（菜单/分组/API/删除）后
// 同步失效该角色下所有用户的可见域缓存。public 以便 handler 在直接调 repo 写库后调用。
func (s *AdminService) InvalidatePermCacheByRole(ctx context.Context, roleID uuid.UUID) {
	if s.permInvalidator == nil {
		return
	}
	userIDs, err := s.roleRepo.ListUserIDsByRole(ctx, roleID)
	if err != nil {
		s.logger.Error("list users by role for cache invalidate failed",
			zap.String("role_id", roleID.String()), zap.Error(err))
		if s.metrics != nil {
			s.metrics.PermCacheInvalidateFailed.Inc()
		}
		return
	}
	for _, uid := range userIDs {
		s.invalidatePermCache(ctx, uid)
	}
}

// ForceLogout 把指定用户的 access/refresh token 全部标记为失效（写入 Redis 撤销时间戳）。
// 已签发但 iat < 撤销时间戳的 token 会在下一次中间件校验时被拒绝。
// 内置用户（source=builtIn）不允许下线，避免锁死系统登录入口。
func (s *AdminService) ForceLogout(ctx context.Context, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}
	for _, id := range userIDs {
		user, err := s.userRepo.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("get user %s: %w", id, err)
		}
		if user.Source == UserSourceBuiltIn {
			return ErrBuiltInUserProtected
		}
	}
	if s.revoker == nil {
		s.logger.Warn("ForceLogout called but TokenRevoker not configured", zap.Int("count", len(userIDs)))
		return nil
	}
	return s.revoker.RevokeBatch(ctx, userIDs)
}

// Login authenticates a user and returns a JWT token pair.
func (s *AdminService) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoginUserNotFound, commonerrors.ErrUnauthorized)
	}

	// Check if account is temporarily locked due to brute-force
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		// issue #220：账号锁定文案与"IP 限流"文案刻意拆开，让用户能区分
		// 是"自己这个账号被锁"还是"来源网络被整体限流"。
		// issue #690：显示剩余锁定分钟数，帮助用户知道等多久。
		remainMinutes := int(time.Until(*user.LockedUntil).Minutes()) + 1 // 向上取整
		return nil, commonerrors.NewBusinessError(7012, fmt.Sprintf("该账号已锁定，请 %d 分钟后重试", remainMinutes), commonerrors.ErrForbidden)
	}

	// PRD §7 P1：账号过期校验。expire_at <= now → 拒绝登录。
	if user.ExpireAt != nil && !time.Now().Before(*user.ExpireAt) {
		return nil, commonerrors.NewBusinessError(7013, "account expired", commonerrors.ErrForbidden)
	}

	if user.Status != UserStatusActive {
		return nil, fmt.Errorf("%w: %w", errLoginAccountDisabled, commonerrors.ErrForbidden)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("%w: %w", errLoginWrongPassword, commonerrors.ErrUnauthorized)
	}

	roles, err := s.roleRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}

	// ⑩ 单点登录（sys_configs security.isOnlyOneUserLoginEnable）：
	//   FE 字段语义为"允许多端并发登录"，true=允许多端 / false=单点登录。
	//   policy.AllowConcurrent=false 时，新登录前先把该用户所有现存 token 标
	//   记为撤销（写 Redis revokedAt=now_us，见 TokenRevoker.Revoke），再签发
	//   带 issued_at_us 的新 token。IsRevoked 用严格小于比较：旧 token
	//   issued_at_us < revokedAt → 被踢；新 token issued_at_us >= revokedAt →
	//   不会被自己踢下线。
	//   Revoker / Policy 任一未注入则跳过（fail-safe — 不阻塞登录）。
	if s.policy != nil && s.revoker != nil {
		if !s.policy.Get(ctx).AllowConcurrent {
			if err := s.revoker.Revoke(ctx, user.ID); err != nil {
				s.logger.Warn("single-session revoke failed (non-fatal)",
					zap.String("username", user.Username),
					zap.Error(err),
				)
				// 不返错 — 单点登录失败比"用户登不进来"风险低，且失败原因通常
				// 是 redis 短暂不可用，下次登录会自动重试。
			}
		}
	}

	tokenPair, err := s.jwt.GenerateTokenPair(&Claims{
		UserID:        user.ID,
		Username:      user.Username,
		IsSuperAdmin:  user.IsSuperAdmin(),
		Roles:         roleNames,
		CurrentRoleID: s.getDefaultRoleID(ctx, user.ID, roles),
	})
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("update last login", zap.Error(err))
	}

	// P1 ①+④ + P2-⑪ 把策略派生字段附在 Login 响应里让 FE 弹改密页 / 提示。
	annotatePasswordPolicyState(ctx, user, s.policySnapshot(ctx), tokenPair)
	annotateLoginNotice(s.policySnapshot(ctx), tokenPair)

	return tokenPair, nil
}

// annotateLoginNotice 附"登录提示"文案（P2-⑪）。
// FE 显示 Modal/Notification；msg 为空 → 静默；enabledFlag=false → 静默。
func annotateLoginNotice(policy *securityPolicyValues, tp *TokenPair) {
	if policy == nil || tp == nil || !policy.LoginNotifyEnabled {
		return
	}
	if policy.LoginNotifyMsg == "" {
		return
	}
	tp.LoginNotifyMsg = policy.LoginNotifyMsg
}

// annotatePasswordPolicyState 把"是否必须改密 / 离过期还剩几天"两个派生字段
// 填入 TokenPair 响应。规则：
//
//	① must_change_password=true              → 用户首次登录 / 管理员重置
//	④ password_expires_at <= now             → 已过期 → 一并要求改密
//	④ password_expires_at - now <= prompt    → 不强制但附 days 让 FE 弹提示
//
// 任何字段未启用（expires=false）或 policy 未注入时，函数无副作用。
func annotatePasswordPolicyState(_ context.Context, user *User, policy *securityPolicyValues, tp *TokenPair) {
	if user == nil || tp == nil {
		return
	}
	// ① 显式标记
	if user.MustChangePassword {
		tp.MustChangePassword = true
	}
	if policy == nil || !policy.PasswordExpiresEnabled || policy.PasswordValidDays <= 0 {
		return
	}
	// ④ 密码过期判定：未设过修改时间戳 → 视为初始密码，立刻强制改密
	if user.PasswordChangedAt == nil {
		tp.MustChangePassword = true
		return
	}
	validDuration := time.Duration(policy.PasswordValidDays) * 24 * time.Hour
	expiresAt := user.PasswordChangedAt.Add(validDuration)
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		tp.MustChangePassword = true
		return
	}
	// 剩余天数（向下取整避免"还剩 0.4 天"显示为 0）
	days := int(remaining / (24 * time.Hour))
	if days < 0 {
		days = 0
	}
	if policy.PasswordPromptDays > 0 && int64(days) <= policy.PasswordPromptDays {
		tp.PasswordExpiresInDays = &days
	}
}

// getDefaultRoleID returns the user's default role ID, or the first role if none is set.
func (s *AdminService) getDefaultRoleID(ctx context.Context, userID uuid.UUID, roles []Role) *uuid.UUID {
	if defaultID, err := s.roleRepo.GetDefaultRoleID(ctx, userID); err == nil && defaultID != nil {
		return defaultID
	}
	if len(roles) > 0 {
		return &roles[0].ID
	}
	return nil
}

// RefreshToken generates a new token pair from a valid refresh token.
func (s *AdminService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, commonerrors.ErrUnauthorized
	}

	// 检查用户级撤销（单点登录踢出）—— refresh token 也需要被撤销机制覆盖，
	// 否则旧设备可以通过 refresh 绕过单点登录限制（issue #698 根因）
	if s.revoker != nil {
		revoked, err := s.revoker.IsRevoked(ctx, claims.UserID, claims.IssuedAt, claims.IssuedAtMicros)
		if err != nil {
			s.logger.Warn("check token revocation failed", zap.Error(err))
			// fail-open: 不阻塞 refresh，但记录错误
		} else if revoked {
			return nil, commonerrors.ErrUnauthorized
		}
	}

	// Verify user still exists and is active
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, commonerrors.ErrUnauthorized
	}
	if user.Status != UserStatusActive {
		return nil, commonerrors.NewBusinessError(7001, "account is disabled", commonerrors.ErrForbidden)
	}

	// Re-fetch roles in case they changed
	roles, err := s.roleRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}

	defaultRoleID := s.getDefaultRoleID(ctx, user.ID, roles)

	return s.jwt.GenerateTokenPair(&Claims{
		UserID:        user.ID,
		Username:      user.Username,
		IsSuperAdmin:  user.IsSuperAdmin(),
		Roles:         roleNames,
		CurrentRoleID: defaultRoleID,
	})
}

// CreateUser creates a new user account with role assignments.
// If role assignment fails, the created user is rolled back to maintain consistency.
//
// issue #649 默认密码接通：
//   - req.UseDefaultPassword=true → 从 sys_configs.security.defaultPasswd 取值，跳过强度校验；
//   - false → 走现有 ValidatePassword（依赖 sys_configs security.pwdMinLength / pwdMaxLength /
//     passwordContent；policy nil 时 fail-open）。
//
// 落库 MustChangePassword 强制为 true（硬规则：管理员注入凭据 → 用户下次登录一律强制改密），
// 不再受 sys_configs.security.modifyPWD 配置开关影响（该开关已废弃）。
//
// 审计：成功/失败均写一条 audit.ActionUserCreate（details.used_default_password 标识路径）。
func (s *AdminService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	policy := s.policySnapshot(ctx)
	plainPwd, err := s.resolveAdminInjectedPassword(req.UseDefaultPassword, req.Password, policy, "密码")
	if err != nil {
		s.auditUserCreateFailure(ctx, req, err)
		return nil, err
	}

	hash, err := hashSecret([]byte(plainPwd))
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// 通过本接口创建的用户固定为管理员添加。
	// LDAP 同步任务走独立内部路径写 UserSourceLDAP；seed 文件写 UserSourceBuiltIn。
	now := time.Now()
	user := &User{
		Username:     req.Username,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Phone:        req.Phone,
		Description:  req.Description,
		ExpireAt:     req.ExpireAt,
		Status:       UserStatusActive,
		Source:       UserSourceAdmin,
		// issue #649 硬规则：管理员创建即强制首次改密（无论手填还是默认密码）。
		MustChangePassword: true,
		// P1-④ 让密码有效期从创建时刻起计；NULL 会被 Login 视为初始密码立刻过期。
		PasswordChangedAt: &now,
	}
	if op := operatorIDFromContext(ctx); op != nil {
		user.CreatedBy = op
		user.UpdatedBy = op
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.auditUserCreateFailure(ctx, req, err)
		return nil, fmt.Errorf("create user: %w", err)
	}

	for _, roleID := range req.RoleIDs {
		if err := s.roleRepo.AssignRole(ctx, user.ID, roleID); err != nil {
			// Roll back user creation to maintain consistency
			if delErr := s.userRepo.Delete(ctx, user.ID); delErr != nil {
				s.logger.Error("failed to rollback user after role assignment failure",
					zap.String("user_id", user.ID.String()),
					zap.Error(delErr),
				)
			}
			s.auditUserCreateFailure(ctx, req, err)
			return nil, fmt.Errorf("assign role %s: %w", roleID, err)
		}
	}

	roles, _ := s.roleRepo.GetUserRoles(ctx, user.ID)
	user.Roles = roles
	s.auditUserCreateSuccess(ctx, req, user)
	return user, nil
}

// resolveAdminInjectedPassword 解析「管理员注入密码」场景下实际要写库的明文密码。
//
// 严格按 useDefault 标志位分支（设计 §3.2 拍板，避免「字段为空即走默认密码」推断歧义）：
//   - useDefault=true  → 取 policy.DefaultPassword；空则返 ErrInvalidInput「系统默认密码未设置」；
//     **跳过 ValidatePassword**（defaultPasswd 在写入侧已校验，是受信任的运维设定值；
//     强度规则调高不影响存量默认密码的可用性，避免 UX 死结，设计 §3.2 方案 A）。
//   - useDefault=false + plain 非空 → 跑 ValidatePassword 强度校验。
//   - useDefault=false + plain 空 → 业务错误「<fieldLabel>不能为空」。
//
// fieldLabel 用于错误消息（CreateUser 传"密码"，ResetPassword 传"新密码"）。
func (s *AdminService) resolveAdminInjectedPassword(useDefault bool, plain string, policy *securityPolicyValues, fieldLabel string) (string, error) {
	if useDefault {
		if policy == nil || policy.DefaultPassword == "" {
			return "", fmt.Errorf("%w: 系统默认密码未设置", commonerrors.ErrInvalidInput)
		}
		return policy.DefaultPassword, nil
	}
	if plain == "" {
		return "", fmt.Errorf("%w: %s不能为空", commonerrors.ErrInvalidInput, fieldLabel)
	}
	if err := ValidatePassword(plain, PasswordPolicySnapshotFromSecurityPolicy(policy)); err != nil {
		return "", fmt.Errorf("%w: %w", err, commonerrors.ErrInvalidInput)
	}
	return plain, nil
}

// auditUserCreateSuccess 写一条用户创建成功的审计日志。
// details.used_default_password 让审计端可区分两条路径，便于合规追溯。
func (s *AdminService) auditUserCreateSuccess(ctx context.Context, req CreateUserRequest, user *User) {
	entry := auditEntryFromContext(ctx)
	entry.Action = audit.ActionUserCreate
	entry.ResourceType = audit.ResourceUser
	entry.ResourceID = user.ID.String()
	entry.Success = true
	entry.Details = map[string]interface{}{
		"used_default_password": req.UseDefaultPassword,
		"target_user_id":        user.ID.String(),
		"target_username":       req.Username,
	}
	audit.Log(ctx, entry)
}

// auditUserCreateFailure 写一条用户创建失败的审计日志（密码强度/默认密码未设置/落库失败等）。
func (s *AdminService) auditUserCreateFailure(ctx context.Context, req CreateUserRequest, err error) {
	entry := auditEntryFromContext(ctx)
	entry.Action = audit.ActionUserCreate
	entry.ResourceType = audit.ResourceUser
	entry.Success = false
	entry.ErrorMessage = err.Error()
	entry.Details = map[string]interface{}{
		"used_default_password": req.UseDefaultPassword,
		"target_username":       req.Username,
	}
	audit.Log(ctx, entry)
}

// UpdateUser updates an existing user.
// 当 req.RoleIDs != nil 时，对用户当前角色与目标 RoleIDs 做差量同步（Assign/Remove）。
func (s *AdminService) UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.Description != nil {
		user.Description = *req.Description
	}
	if req.ExpireAt != nil {
		user.ExpireAt = req.ExpireAt
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if op := operatorIDFromContext(ctx); op != nil {
		user.UpdatedBy = op
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	if req.RoleIDs != nil {
		if err := s.syncUserRoles(ctx, user.ID, *req.RoleIDs); err != nil {
			return nil, fmt.Errorf("sync user roles: %w", err)
		}
	}

	// 角色变更 → 失效该用户的可见分组缓存（PRD §10 DoD）。
	// v1.0：carrier 字段已删，仅判定 RoleIDs 变化。
	if req.RoleIDs != nil {
		s.invalidatePermCache(ctx, user.ID)
	}

	roles, _ := s.roleRepo.GetUserRoles(ctx, user.ID)
	user.Roles = roles
	return user, nil
}

// BatchAssignRoles 整体替换一组用户的角色集合（语义与 UpdateUser.RoleIDs 差量同步一致）。
// 失败语义：单个用户失败立即返回，已成功用户的变更不回滚（与 PRD §4.2 批量"汇总成功/跳过"约定）。
// 每个用户成功后调用 InvalidateUserCache（PRD §10 DoD）。
func (s *AdminService) BatchAssignRoles(ctx context.Context, userIDs, roleIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}
	for _, uid := range userIDs {
		if _, err := s.userRepo.GetByID(ctx, uid); err != nil {
			return fmt.Errorf("get user %s: %w", uid, err)
		}
		if err := s.syncUserRoles(ctx, uid, roleIDs); err != nil {
			return fmt.Errorf("sync roles for %s: %w", uid, err)
		}
		s.invalidatePermCache(ctx, uid)
	}
	return nil
}

// CopyUser 复制一个用户：用户名 = "<src.username>_copy_<n>"（n 自动递增直到不冲突）。
// 复制项：display_name / email / phone / 当前 roles（v1.0：carrier 字段已删除）。
// 不复制项：password（强制随机生成 12 位密码并随响应返回，前端必须立刻提示管理员重置）/ source（新副本固定 admin）。
func (s *AdminService) CopyUser(ctx context.Context, sourceID uuid.UUID) (*User, string, error) {
	src, err := s.userRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, "", fmt.Errorf("get source user: %w", err)
	}

	newUsername, err := s.allocateCopyUsername(ctx, src.Username)
	if err != nil {
		return nil, "", err
	}

	tempPwd, err := generateRandomPassword(12)
	if err != nil {
		return nil, "", fmt.Errorf("generate temp password: %w", err)
	}
	hash, err := hashSecret([]byte(tempPwd))
	if err != nil {
		return nil, "", fmt.Errorf("hash temp password: %w", err)
	}

	srcRoles, err := s.roleRepo.GetUserRoles(ctx, src.ID)
	if err != nil {
		return nil, "", fmt.Errorf("get source roles: %w", err)
	}

	copyUser := &User{
		Username:     newUsername,
		PasswordHash: string(hash),
		DisplayName:  src.DisplayName,
		Email:        src.Email,
		Phone:        src.Phone,
		Status:       UserStatusActive,
		Source:       UserSourceAdmin,
	}
	if err := s.userRepo.Create(ctx, copyUser); err != nil {
		return nil, "", fmt.Errorf("create copy user: %w", err)
	}

	for _, r := range srcRoles {
		if err := s.roleRepo.AssignRole(ctx, copyUser.ID, r.ID); err != nil {
			// 失败不回滚已创建的副本（管理员可手动调整角色），但记日志便于排查
			s.logger.Warn("assign role to copy user failed",
				zap.String("user_id", copyUser.ID.String()),
				zap.String("role_id", r.ID.String()),
				zap.Error(err))
		}
	}

	loaded, _ := s.roleRepo.GetUserRoles(ctx, copyUser.ID)
	copyUser.Roles = loaded
	return copyUser, tempPwd, nil
}

// allocateCopyUsername 寻找一个可用的副本用户名：base_copy / base_copy_2 / base_copy_3 ...
// 上限 99 次以避免极端情况下无限循环。
func (s *AdminService) allocateCopyUsername(ctx context.Context, base string) (string, error) {
	for i := 1; i <= 99; i++ {
		candidate := base + "_copy"
		if i > 1 {
			candidate = fmt.Sprintf("%s_copy_%d", base, i)
		}
		if _, err := s.userRepo.GetByUsername(ctx, candidate); err != nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no available copy username for %s after 99 attempts", base)
}

// syncUserRoles 对比当前角色与目标角色，差量调用 AssignRole/RemoveRole。
// 顺序：先 Assign（避免最后一个角色被先删导致中间态无角色）再 Remove。
func (s *AdminService) syncUserRoles(ctx context.Context, userID uuid.UUID, target []uuid.UUID) error {
	current, err := s.roleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return fmt.Errorf("get current roles: %w", err)
	}

	currentSet := make(map[uuid.UUID]struct{}, len(current))
	for _, r := range current {
		currentSet[r.ID] = struct{}{}
	}
	targetSet := make(map[uuid.UUID]struct{}, len(target))
	for _, id := range target {
		targetSet[id] = struct{}{}
	}

	for id := range targetSet {
		if _, exists := currentSet[id]; exists {
			continue
		}
		if err := s.roleRepo.AssignRole(ctx, userID, id); err != nil {
			return fmt.Errorf("assign role %s: %w", id, err)
		}
	}
	for id := range currentSet {
		if _, keep := targetSet[id]; keep {
			continue
		}
		if err := s.roleRepo.RemoveRole(ctx, userID, id); err != nil {
			return fmt.Errorf("remove role %s: %w", id, err)
		}
	}
	return nil
}

// DeleteUser deletes a user account.
// 内置用户（source=builtIn）受保护，返回 ErrBuiltInUserProtected → HTTP 409。
// 删除成功后失效该用户权限缓存，防止旧 key 在 TTL 内被陈旧请求命中（PRD §10 DoD）。
func (s *AdminService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user for delete: %w", err)
	}
	if user.Source == UserSourceBuiltIn {
		return ErrBuiltInUserProtected
	}
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidatePermCache(ctx, id)
	return nil
}

// ListUsers returns a filtered, paginated list of users with roles loaded.
//
// 同时反查 created_by / updated_by 对应的 username（一次性 N→K，K 为去重后的
// operator 数）注入 CreatorUsername / UpdaterUsername，省去前端二次拉全量
// /admin/users 仅为展示创建人/更新人列。
func (s *AdminService) ListUsers(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error) {
	result, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// 批量加载所有用户的角色 + creator/updater username。
	if len(result.Items) > 0 {
		userIds := make([]uuid.UUID, len(result.Items))
		for i, user := range result.Items {
			userIds[i] = user.ID
		}

		rolesMap, err := s.roleRepo.GetUserRolesBatch(ctx, userIds)
		if err != nil {
			s.logger.Warn("failed to load roles for users", zap.Error(err))
			// 即使角色加载失败，仍然返回用户列表
		} else {
			for i := range result.Items {
				result.Items[i].Roles = rolesMap[result.Items[i].ID]
			}
		}

		s.enrichOperatorUsernames(ctx, result.Items)
	}

	return result, nil
}

// GetUser returns a user by ID with roles loaded.
//
// 同 ListUsers 一并填充 CreatorUsername / UpdaterUsername，详情面板与列表口径一致。
func (s *AdminService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	roles, err := s.roleRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	user.Roles = roles

	one := []User{*user}
	s.enrichOperatorUsernames(ctx, one)
	user.CreatorUsername = one[0].CreatorUsername
	user.UpdaterUsername = one[0].UpdaterUsername
	return user, nil
}

// enrichOperatorUsernames 收集 users 中出现过的 created_by / updated_by ID 去重，
// 一次 SQL 拉回 username 映射，再回填到每条 User 的派生字段上。
// repo 失败时只 warn 不 abort——丢失派生字段不应阻塞列表返回。
func (s *AdminService) enrichOperatorUsernames(ctx context.Context, users []User) {
	if len(users) == 0 {
		return
	}
	idSet := make(map[uuid.UUID]struct{}, len(users)*2)
	for _, u := range users {
		if u.CreatedBy != nil {
			idSet[*u.CreatedBy] = struct{}{}
		}
		if u.UpdatedBy != nil {
			idSet[*u.UpdatedBy] = struct{}{}
		}
	}
	nameMap := map[uuid.UUID]string{}
	if len(idSet) > 0 {
		ids := make([]uuid.UUID, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}
		var err error
		nameMap, err = s.userRepo.GetUsernamesByIDs(ctx, ids)
		if err != nil {
			s.logger.Warn("enrich operator usernames", zap.Error(err))
			nameMap = map[uuid.UUID]string{}
		}
	}
	for i := range users {
		if users[i].CreatedBy != nil {
			users[i].CreatorUsername = nameMap[*users[i].CreatedBy]
		} else if users[i].Source == UserSourceAdmin {
			users[i].CreatorUsername = "admin"
		}
		if users[i].UpdatedBy != nil {
			users[i].UpdaterUsername = nameMap[*users[i].UpdatedBy]
		} else if users[i].Source == UserSourceAdmin {
			users[i].UpdaterUsername = "admin"
		}
	}
}

// enrichRoleOperatorUsernames 与用户列表同口径，给角色列表/详情派生创建人、更新人 username。
// 历史自建角色若 created_by / updated_by 为空，按管理员创建兜底显示 admin；内置角色留空，前端按 builtIn 渲染"内置"。
func (s *AdminService) enrichRoleOperatorUsernames(ctx context.Context, roles []Role) {
	if len(roles) == 0 {
		return
	}
	idSet := make(map[uuid.UUID]struct{}, len(roles)*2)
	for _, role := range roles {
		if role.CreatedBy != nil {
			idSet[*role.CreatedBy] = struct{}{}
		}
		if role.UpdatedBy != nil {
			idSet[*role.UpdatedBy] = struct{}{}
		}
	}
	nameMap := map[uuid.UUID]string{}
	if len(idSet) > 0 {
		ids := make([]uuid.UUID, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}
		var err error
		nameMap, err = s.userRepo.GetUsernamesByIDs(ctx, ids)
		if err != nil {
			s.logger.Warn("enrich role operator usernames", zap.Error(err))
			nameMap = map[uuid.UUID]string{}
		}
	}
	for i := range roles {
		if roles[i].CreatedBy != nil {
			roles[i].CreatorUsername = nameMap[*roles[i].CreatedBy]
		}
		if roles[i].CreatorUsername == "" && !roles[i].IsSystem {
			roles[i].CreatorUsername = "admin"
		}
		if roles[i].UpdatedBy != nil {
			roles[i].UpdaterUsername = nameMap[*roles[i].UpdatedBy]
		}
		if roles[i].UpdaterUsername == "" && !roles[i].IsSystem {
			roles[i].UpdaterUsername = "admin"
		}
	}
}

// AssignRole assigns a role to a user.
// 成功后失效该用户权限缓存，使新角色对应的设备分组立即生效（PRD §10 DoD）。
func (s *AdminService) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return fmt.Errorf("get user for role assignment: %w", err)
	}
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return fmt.Errorf("get role for assignment: %w", err)
	}
	if err := s.roleRepo.AssignRole(ctx, userID, roleID); err != nil {
		return err
	}
	s.invalidatePermCache(ctx, userID)
	return nil
}

// RemoveRole removes a role from a user.
// 成功后失效该用户权限缓存，避免继续看到已解绑角色的设备数据（PRD §10 DoD）。
func (s *AdminService) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if err := s.roleRepo.RemoveRole(ctx, userID, roleID); err != nil {
		return err
	}
	s.invalidatePermCache(ctx, userID)
	return nil
}

// CheckPermission checks if a user has a specific permission.
func (s *AdminService) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	return s.roleRepo.CheckPermission(ctx, userID, resource, action)
}

// ListRoles returns all roles (legacy, for backward compatibility).
func (s *AdminService) ListRoles(ctx context.Context) ([]Role, error) {
	roles, err := s.roleRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	s.enrichRoleOperatorUsernames(ctx, roles)
	return roles, nil
}

// ListRolesPaginated returns roles with pagination support.
func (s *AdminService) ListRolesPaginated(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error) {
	result, err := s.roleRepo.ListWithPagination(ctx, filter)
	if err != nil {
		return nil, err
	}
	s.enrichRoleOperatorUsernames(ctx, result.Items)
	return result, nil
}

// ResetPassword resets a user's password using the admin-initiated path.
//
// 拦截顺序（短路）：
//  1. LDAP 用户（source=LDAP）密码归属外部域 → ErrLDAPPasswordExternal（HTTP 409）。
//  2. 内置用户（source=builtIn，issue #649 新增）→ ErrBuiltInUserProtected（HTTP 409）；
//     与 LockUser/DeleteUser/ForceLogout 拦截策略对齐，内置 admin 忘密走 omcctl CLI / DB 手工。
//  3. 按 req.UseDefaultPassword 解析密码来源（resolveAdminInjectedPassword）。
//
// 写库成功后：
//   - 硬规则置 must_change_password=true（issue #649：管理员注入凭据 → 用户下次登录强制改密；
//     UpdatePassword 已自动清零该字段，此处 setter 回 true）；
//   - 清登录失败计数与锁定状态，避免管理员重置后用户仍处锁定；
//   - **revoker.Revoke(id)**（issue #649 新增，OWASP A07）：旧 access/refresh token 立即失效，
//     与 LockUser 既有行为对齐；revoker 未注入（测试场景）时 warn 不阻断业务。
//
// 审计：成功/失败均写一条 audit.ActionPasswordReset。
func (s *AdminService) ResetPassword(ctx context.Context, id uuid.UUID, req ResetPasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		s.auditPasswordResetFailure(ctx, id, "", req.UseDefaultPassword, err)
		return fmt.Errorf("get user for reset password: %w", err)
	}
	if user.Source == UserSourceLDAP {
		s.auditPasswordResetFailure(ctx, id, user.Username, req.UseDefaultPassword, ErrLDAPPasswordExternal)
		return ErrLDAPPasswordExternal
	}
	if user.Source == UserSourceBuiltIn {
		s.auditPasswordResetFailure(ctx, id, user.Username, req.UseDefaultPassword, ErrBuiltInUserProtected)
		return ErrBuiltInUserProtected
	}
	policy := s.policySnapshot(ctx)
	plainPwd, err := s.resolveAdminInjectedPassword(req.UseDefaultPassword, req.NewPassword, policy, "新密码")
	if err != nil {
		s.auditPasswordResetFailure(ctx, id, user.Username, req.UseDefaultPassword, err)
		return err
	}
	hash, err := hashSecret([]byte(plainPwd))
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, id, string(hash)); err != nil {
		s.auditPasswordResetFailure(ctx, id, user.Username, req.UseDefaultPassword, err)
		return fmt.Errorf("update password: %w", err)
	}
	// issue #649 硬规则：管理员重置后下次登录必须改密。UpdatePassword 已清零，此处再 setter 回 true。
	// SetMustChangePassword / ResetLoginSecurity 不在 UserRepository 接口上，
	// 通过类型断言直接调具体实现，失败仅警告（密码已成功重置）。
	if repo, ok := s.userRepo.(*PgUserRepository); ok {
		if err := repo.SetMustChangePassword(ctx, id, true); err != nil {
			s.logger.Warn("set must_change_password after reset", zap.Error(err))
		}
		if err := repo.ResetLoginSecurity(ctx, id); err != nil {
			s.logger.Warn("reset login security after password reset", zap.Error(err))
		}
	}
	// issue #649 OWASP A07：旧 access/refresh token 立即失效，对齐 LockUser 既有行为。
	// revoker 未注入（测试场景）时退化为 no-op，业务返回 nil。
	if s.revoker != nil {
		if err := s.revoker.Revoke(ctx, id); err != nil {
			s.logger.Error("revoke tokens after password reset failed",
				zap.String("user_id", id.String()), zap.Error(err))
		}
	}
	s.auditPasswordResetSuccess(ctx, id, user.Username, req.UseDefaultPassword)
	return nil
}

// auditPasswordResetSuccess 写一条管理员重置密码成功的审计日志。
func (s *AdminService) auditPasswordResetSuccess(ctx context.Context, targetID uuid.UUID, username string, usedDefault bool) {
	entry := auditEntryFromContext(ctx)
	entry.Action = audit.ActionPasswordReset
	entry.ResourceType = audit.ResourceUser
	entry.ResourceID = targetID.String()
	entry.Success = true
	entry.Details = map[string]interface{}{
		"used_default_password": usedDefault,
		"target_user_id":        targetID.String(),
		"target_username":       username,
	}
	audit.Log(ctx, entry)
}

// auditPasswordResetFailure 写一条管理员重置密码失败的审计日志
// （LDAP 拒绝 / 内置用户拒绝 / 默认密码未设置 / 强度不通过 / 落库失败等）。
func (s *AdminService) auditPasswordResetFailure(ctx context.Context, targetID uuid.UUID, username string, usedDefault bool, err error) {
	entry := auditEntryFromContext(ctx)
	entry.Action = audit.ActionPasswordReset
	entry.ResourceType = audit.ResourceUser
	entry.ResourceID = targetID.String()
	entry.Success = false
	entry.ErrorMessage = err.Error()
	entry.Details = map[string]interface{}{
		"used_default_password": usedDefault,
		"target_username":       username,
	}
	audit.Log(ctx, entry)
}

// LockUser disables a user account by setting its status to disabled.
// 内置用户（source=builtIn）禁止禁用，避免锁死系统登录入口 → HTTP 409。
// PRD §1.2 验收口径：禁用用户 → 该用户已签发的 token 强制失效（force-logout 联动）。
func (s *AdminService) LockUser(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user for lock: %w", err)
	}
	if user.Source == UserSourceBuiltIn {
		return ErrBuiltInUserProtected
	}
	user.Status = UserStatusDisabled
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}
	// PRD §1.2 验收口径：禁用 → 已签发 token 立即失效。
	// revoker 未注入（测试场景）时退化为 no-op，状态变更仍然生效。
	if s.revoker != nil {
		if err := s.revoker.Revoke(ctx, id); err != nil {
			s.logger.Error("revoke tokens after lock failed",
				zap.String("user_id", id.String()), zap.Error(err))
		}
	}
	return nil
}

// LockUserByUsername temporarily locks a user account due to brute-force protection.
func (s *AdminService) LockUserByUsername(ctx context.Context, username string, duration time.Duration) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return
	}
	lockedUntil := time.Now().Add(duration)
	if repo, ok := s.userRepo.(*PgUserRepository); ok {
		if err := repo.UpdateLoginSecurity(ctx, user.ID, int(lockThreshold), &lockedUntil); err != nil {
			s.logger.Error("lock user by username", zap.Error(err), zap.String("username", username))
		}
	}
}

// UnlockUser re-enables a user account by setting its status to active.
func (s *AdminService) UnlockUser(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user for unlock: %w", err)
	}
	user.Status = UserStatusActive
	return s.userRepo.Update(ctx, user)
}

// GetRole returns a role by ID. B3-Phase2-B 起 permissions 表已 DROP，
// role.Permissions 永远为空切片；权限以 role_menus 承载。
func (s *AdminService) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	roles := []Role{*role}
	s.enrichRoleOperatorUsernames(ctx, roles)
	role.CreatorUsername = roles[0].CreatorUsername
	role.UpdaterUsername = roles[0].UpdaterUsername
	return role, nil
}

// CreateRole creates a new role with optional permissions.
// v0.6（roles.md §7 P1 #5）：自动写入 created_by / updated_by = ctx 中的操作者 ID。
func (s *AdminService) CreateRole(ctx context.Context, req CreateRoleRequest) (*Role, error) {
	role := &Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
	}
	if op := operatorIDFromContext(ctx); op != nil {
		role.CreatedBy = op
		role.UpdatedBy = op
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	// B3-Phase2-B：permissions 表已 DROP；角色权限改由 role_menus（菜单可见性）+
	// req.Permissions 仅为前端兼容字段，已忽略；菜单权限改由 role_menus 承载。
	_ = req.Permissions

	return s.GetRole(ctx, role.ID)
}

// UpdateRole updates an existing role's fields and replaces its permissions.
// v0.6（roles.md §7 P1 #5）：自动写入 updated_by = ctx 中的操作者 ID。
func (s *AdminService) UpdateRole(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get role for update: %w", err)
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Code != nil {
		role.Code = *req.Code
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if op := operatorIDFromContext(ctx); op != nil {
		role.UpdatedBy = op
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}

	// B3-Phase2-B：permissions 表已 DROP（同 CreateRole）。req.Permissions 仅为前端
	// 兼容字段，后端忽略。前端 P3 落地后该字段在 DTO 中删除。
	_ = req.Permissions

	return s.GetRole(ctx, id)
}

// DeleteRole deletes a role by ID.
// PRD roles.md §10 DoD：删除前先取该角色用户列表（之后 user_roles 由 ON DELETE CASCADE
// 自动清空），删库成功后再逐一失效这些用户的权限缓存。
func (s *AdminService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	// 固定内置 UUID 白名单兜底：admin/operator/viewer 三个内置角色一律不可删除，
	// 即便 is_system 标记在某环境被错误改写也拦得住（issue #136 双重保护）。
	if isBuiltInRole(id) {
		return commonerrors.ErrForbidden
	}
	userIDs, err := s.roleRepo.ListUserIDsByRole(ctx, id)
	if err != nil {
		s.logger.Warn("list users by role before delete failed",
			zap.String("role_id", id.String()), zap.Error(err))
	}
	// 引用检查：仍有用户引用该角色时拦截删除，返回 in-use(7008) 业务错误
	// （经 HTTPStatusFromError → 409 Conflict）。放在内置角色保护之后、repo.Delete 之前。
	if len(userIDs) > 0 {
		return commonerrors.NewBusinessError(
			commonerrors.ErrCodeRoleInUse, "role still in use", commonerrors.ErrAlreadyExists)
	}
	if err := s.roleRepo.Delete(ctx, id); err != nil {
		return err
	}
	for _, uid := range userIDs {
		s.invalidatePermCache(ctx, uid)
	}
	return nil
}

// CopyRole 一键复制一个角色（roles.md §7 P2 #9）。
// 复制项：name + "_copy_N"（N 自动递增直到不冲突）/ description /
// role_menus / role_device_groups（含 network_types）/ role_api_permissions。
// 不复制项：is_system（副本固定 false，绝不会复制出新内置角色）/ created_by / updated_by（重新写为操作者）。
//
// 失败语义：源不存在 → ErrNotFound；副本名冲突超 99 次 → 报错；任一绑定
// 复制失败时删除新建副本，避免留下权限不完整的半成品角色。
func (s *AdminService) CopyRole(ctx context.Context, sourceID uuid.UUID) (*Role, error) {
	src, err := s.roleRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("get source role: %w", err)
	}

	newName, err := s.allocateCopyRoleName(ctx, src.Name)
	if err != nil {
		return nil, err
	}

	op := operatorIDFromContext(ctx)
	copied := &Role{
		Name:        newName,
		Description: src.Description,
		IsSystem:    false,
		CreatedBy:   op,
		UpdatedBy:   op,
	}
	if err := s.roleRepo.Create(ctx, copied); err != nil {
		return nil, fmt.Errorf("create copy role: %w", err)
	}

	// B3-Phase2-B：permissions 表已 DROP；复制路径走 role_menus +
	// role_api_permissions。src.Permissions 字段保留向后兼容，不消费。
	_ = src.Permissions

	if s.roleCopyBindings == nil {
		return nil, s.copyRoleFailure(ctx, copied.ID,
			fmt.Errorf("copy role bindings: %w", commonerrors.ErrInternal))
	}

	groupData, err := s.roleCopyBindings.GetDeviceGroupData(ctx, sourceID)
	if err != nil {
		return nil, s.copyRoleFailure(ctx, copied.ID,
			fmt.Errorf("copy role device groups: %w", err))
	}
	if groupData == nil {
		groupData = &RoleDeviceGroupData{}
	}
	if err := s.roleCopyBindings.SetDeviceGroupData(ctx, copied.ID, *groupData); err != nil {
		return nil, s.copyRoleFailure(ctx, copied.ID,
			fmt.Errorf("copy role device groups: %w", err))
	}

	menuIDs, err := s.menuRepo.GetRoleMenuIDs(ctx, sourceID)
	if err != nil {
		return nil, s.copyRoleFailure(ctx, copied.ID,
			fmt.Errorf("copy role menus: %w", err))
	}
	opID := uuid.Nil
	if op != nil {
		opID = *op
	}
	if err := s.menuRepo.SetRoleMenus(ctx, copied.ID, menuIDs, opID); err != nil {
		return nil, s.copyRoleFailure(ctx, copied.ID,
			fmt.Errorf("copy role menus: %w", err))
	}

	// API 权限最后写入。SetRoleApiEndpoints 会刷新 Casbin；把它放在最后可
	// 避免菜单/设备组的后续失败在内存中留下已删除副本的策略。
	endpointIDs, err := s.roleCopyBindings.GetRoleApiEndpointIDs(ctx, sourceID)
	if err != nil {
		return nil, s.copyRoleFailure(ctx, copied.ID,
			fmt.Errorf("copy role API permissions: %w", err))
	}
	if err := s.roleCopyBindings.SetRoleApiEndpoints(ctx, copied.ID, endpointIDs); err != nil {
		return nil, s.copyRoleFailure(ctx, copied.ID,
			fmt.Errorf("copy role API permissions: %w", err))
	}

	result, err := s.GetRole(ctx, copied.ID)
	if err != nil {
		return nil, s.copyRoleFailure(ctx, copied.ID,
			fmt.Errorf("get copied role: %w", err))
	}
	return result, nil
}

func (s *AdminService) copyRoleFailure(ctx context.Context, copiedID uuid.UUID, cause error) error {
	// HTTP 请求取消/超时正是最需要补偿的场景；保留 context values，但让清理
	// 脱离原请求的取消信号，并用短超时防止后台无限悬挂。
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), copyRoleCleanupTimeout)
	defer cancel()
	if err := s.roleRepo.Delete(cleanupCtx, copiedID); err != nil {
		return fmt.Errorf("%w; cleanup copied role: %w", cause, err)
	}
	return cause
}

// allocateCopyRoleName 寻找一个可用的副本角色名：base_copy / base_copy_2 / base_copy_3 ...
// 上限 99 次以避免极端情况下无限循环（与 allocateCopyUsername 对齐）。
func (s *AdminService) allocateCopyRoleName(ctx context.Context, base string) (string, error) {
	for i := 1; i <= 99; i++ {
		candidate := base + "_copy"
		if i > 1 {
			candidate = fmt.Sprintf("%s_copy_%d", base, i)
		}
		if _, err := s.roleRepo.GetByName(ctx, candidate); err != nil {
			// GetByName 返回 ErrNotFound 即可用
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no available copy role name for %s after 99 attempts", base)
}

// ListAllPermissions 已 deprecated（B3-Phase2-B 起 permissions 表 DROP）。
// 永远返回空切片；前端 P3 完成后该路由 + 方法整体移除。
func (s *AdminService) ListAllPermissions(ctx context.Context) ([]Permission, error) {
	return s.roleRepo.ListAllPermissions(ctx)
}

// ==================== Menu Methods ====================

// CreateMenu creates a new menu item.
func (s *AdminService) CreateMenu(ctx context.Context, req CreateMenuRequest, operatorID uuid.UUID) (*Menu, error) {
	// Check if permission key already exists
	existing, _ := s.menuRepo.GetByPermissionKey(ctx, req.PermissionKey)
	if existing != nil {
		return nil, commonerrors.NewBusinessError(7003, "permission key already exists", nil)
	}

	// Check parent menu exists if provided
	if req.ParentID != nil {
		parent, err := s.menuRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, commonerrors.NewBusinessError(7004, "parent menu not found", nil)
		}
		if parent.Type != MenuTypeDirectory && parent.Type != MenuTypeMenu {
			return nil, commonerrors.NewBusinessError(7005, "parent menu must be directory or menu type", nil)
		}
	}

	menu := &Menu{
		Name:          req.Name,
		NameI18n:      req.NameI18n,
		Type:          req.Type,
		PermissionKey: req.PermissionKey,
		ParentID:      req.ParentID,
		SortOrder:     req.SortOrder,
		RoutePath:     req.RoutePath,
		ComponentPath: req.ComponentPath,
		Icon:          req.Icon,
		ShowStatus:    defaultMenuShowStatus(req.ShowStatus),
		Status:        MenuStatusNormal,
	}

	if err := s.menuRepo.Create(ctx, menu, operatorID); err != nil {
		return nil, fmt.Errorf("create menu: %w", err)
	}

	return s.menuRepo.GetByID(ctx, menu.ID)
}

// GetMenu returns a menu by ID.
func (s *AdminService) GetMenu(ctx context.Context, id uuid.UUID) (*Menu, error) {
	return s.menuRepo.GetByID(ctx, id)
}

// ListMenus returns a filtered, paginated list of menus.
func (s *AdminService) ListMenus(ctx context.Context, filter MenuFilter) (*model.ListResponse[Menu], error) {
	return s.menuRepo.List(ctx, filter)
}

// UpdateMenu updates an existing menu.
func (s *AdminService) UpdateMenu(ctx context.Context, id uuid.UUID, req *UpdateMenuRequest, operatorID uuid.UUID) error {
	_, err := s.menuRepo.GetByID(ctx, id)
	if err != nil {
		return commonerrors.ErrNotFound
	}

	return s.menuRepo.Update(ctx, id, req, operatorID)
}

// DeleteMenus deletes menu items by IDs.
func (s *AdminService) DeleteMenus(ctx context.Context, ids []uuid.UUID) error {
	return s.menuRepo.Delete(ctx, ids)
}

// GetMenuTree returns the full menu tree.
func (s *AdminService) GetMenuTree(ctx context.Context, status *MenuStatus) ([]Menu, error) {
	return s.menuRepo.GetTree(ctx, status)
}

// GetUserMenuTree returns the menu tree for a specific user.
//
// 超管旁路（参 docs/prd/system/menu-dynamic-loading.md §4.2.2）：
// 若 user.source='builtIn' → 返回所有 status='normal' 的菜单（不依赖 role_menus）。
// 这与 internal/admin/middleware.go:138/191 的 builtIn 用户直接放行行为对齐。
// 仅 source='builtIn' 视为超管，role.code='admin' 的非 builtIn 用户仍受 role_menus 限制。
func (s *AdminService) GetUserMenuTree(ctx context.Context, userID uuid.UUID) ([]Menu, error) {
	menus, err := s.menuRepoGetMenusForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	menus = s.applyLicenseMenuGate(ctx, menus)
	tree := pruneEmptyDirectories(buildMenuTree(menus))
	return filterTreeForDisplay(tree), nil
}

// menuRepoGetMenusForUser 根据 user.source 选择菜单加载策略：
// - source='builtIn'  → GetAllActive（超管旁路）
// - 其他              → GetByUser（按 user_roles → role_menus 过滤）
func (s *AdminService) menuRepoGetMenusForUser(ctx context.Context, userID uuid.UUID) ([]Menu, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user for menu tree: %w", err)
	}
	if user.Source == UserSourceBuiltIn {
		return s.menuRepo.GetAllActive(ctx)
	}
	return s.menuRepo.GetByUser(ctx, userID)
}

// SetRoleMenus sets the menu permissions for a role.
//
// 任何被授予的后代节点（menu / button）都会自动补全其祖先目录链；
// 否则 assembleMenuTree 会因父节点缺失把整棵子树丢弃（孤儿节点既不是 root，
// 也不挂在任何 root 下，导致前端 NavMenu 看不到该子树）。
//
// 例：仅授 [设备列表→查询] 时，若不补祖先，前端看不到"设备管理 / 设备列表"。
// 修复后会自动加入 [设备管理, 设备列表]，整条目录链可见。
func (s *AdminService) SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error {
	// Verify role exists
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return commonerrors.ErrNotFound
	}

	expanded, err := s.expandMenuAncestors(ctx, menuIDs)
	if err != nil {
		return fmt.Errorf("expand menu ancestors: %w", err)
	}

	if err := s.menuRepo.SetRoleMenus(ctx, roleID, expanded, operatorID); err != nil {
		return err
	}
	// PRD roles.md §10 DoD：菜单变更后失效该角色下所有用户的可见域缓存。
	s.InvalidatePermCacheByRole(ctx, roleID)
	return nil
}

// expandMenuAncestors 把输入的 menuIDs 集合扩展为"自身 + 全部祖先目录"。
// 全菜单数量小（< 200），加载一次即可在内存里做闭包。
//
// 边界：
//   - 空输入直接返回空切片，不查 DB
//   - 输入中包含数据库不存在的 ID 时静默跳过（不阻塞保存，由 repo 端 FK 约束兜底）
//   - 防自环 / 重复出现：用 map 去重 + 已访问标记
func (s *AdminService) expandMenuAncestors(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return ids, nil
	}
	all, err := s.menuRepo.GetAllActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active menus: %w", err)
	}
	byID := make(map[uuid.UUID]Menu, len(all))
	for _, m := range all {
		byID[m.ID] = m
	}

	seen := make(map[uuid.UUID]struct{}, len(ids))
	queue := append([]uuid.UUID(nil), ids...)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if _, ok := seen[cur]; ok {
			continue
		}
		seen[cur] = struct{}{}
		m, exists := byID[cur]
		if !exists {
			continue
		}
		if m.ParentID != nil {
			queue = append(queue, *m.ParentID)
		}
	}

	out := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out, nil
}

// GetRoleMenus returns the menu permissions for a role.
func (s *AdminService) GetRoleMenus(ctx context.Context, roleID uuid.UUID) ([]Menu, error) {
	return s.menuRepo.GetByRole(ctx, roleID)
}

// GetRoleMenuIDs returns the menu IDs for a role.
func (s *AdminService) GetRoleMenuIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	return s.menuRepo.GetRoleMenuIDs(ctx, roleID)
}

// CheckMenuPermission checks if a user has a specific menu permission.
func (s *AdminService) CheckMenuPermission(ctx context.Context, userID uuid.UUID, permissionKey string) (bool, error) {
	menus, err := s.menuRepo.GetByUser(ctx, userID)
	if err != nil {
		return false, err
	}

	return checkPermissionInMenus(menus, permissionKey), nil
}

// buildMenuTree builds a tree structure from a flat list of menus.
//
// 复用 pg_menu_repository.assembleMenuTree（修复了早期 2-层装配 bug，参 buildTree 注释）。
func buildMenuTree(menus []Menu) []Menu {
	return assembleMenuTree(menus)
}

// filterTreeForDisplay filters the tree to only show directories and menus (no buttons).
func filterTreeForDisplay(nodes []Menu) []Menu {
	var result []Menu
	for _, node := range nodes {
		if node.Type == MenuTypeButton {
			continue
		}

		filtered := Menu{
			ID:            node.ID,
			Name:          node.Name,
			Type:          node.Type,
			PermissionKey: node.PermissionKey,
			SortOrder:     node.SortOrder,
			RoutePath:     node.RoutePath,
			Icon:          node.Icon,
			ShowStatus:    node.ShowStatus,
			Status:        node.Status,
		}

		if len(node.Children) > 0 {
			filtered.Children = filterTreeForDisplay(node.Children)
		}

		result = append(result, filtered)
	}
	return result
}

// defaultMenuShowStatus 当请求中未给定 show_status 时回退默认值 "show"，
// 防止空串触发 DDL CHECK (show_status IN ('show','hide')) 失败。
func defaultMenuShowStatus(s MenuShowStatus) MenuShowStatus {
	if s == MenuShow || s == MenuHide {
		return s
	}
	return MenuShow
}

// checkPermissionInMenus recursively checks if a permission key exists in the menu tree.
func checkPermissionInMenus(menus []Menu, permissionKey string) bool {
	for _, menu := range menus {
		// Exact match
		if menu.PermissionKey == permissionKey {
			return true
		}

		// Prefix match (e.g., "device:list" matches "device:list:query")
		if len(permissionKey) > len(menu.PermissionKey) &&
			permissionKey[len(menu.PermissionKey)] == ':' &&
			permissionKey[:len(menu.PermissionKey)] == menu.PermissionKey {
			return true
		}

		// Check children recursively
		if len(menu.Children) > 0 && checkPermissionInMenus(menu.Children, permissionKey) {
			return true
		}
	}

	return false
}

// ==================== Role Switching ====================

// SwitchRole switches the user's active role and returns a new token pair.
func (s *AdminService) SwitchRole(ctx context.Context, userID uuid.UUID, targetRoleID uuid.UUID) (*TokenPair, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Verify the target role is assigned to this user
	roles, err := s.roleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	found := false
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
		if r.ID == targetRoleID {
			found = true
		}
	}
	if !found {
		return nil, commonerrors.NewBusinessError(7003, "target role not assigned to user", commonerrors.ErrForbidden)
	}

	// Update default role
	if err := s.roleRepo.SetDefaultRole(ctx, userID, targetRoleID); err != nil {
		s.logger.Warn("set default role", zap.Error(err))
	}

	return s.jwt.GenerateTokenPair(&Claims{
		UserID:        user.ID,
		Username:      user.Username,
		IsSuperAdmin:  user.IsSuperAdmin(),
		Roles:         roleNames,
		CurrentRoleID: &targetRoleID,
	})
}

// GetUserMenuTreeByRole returns the menu tree based on the user's current active role.
//
// 超管旁路同 GetUserMenuTree：source='builtIn' 用户返回全量 active 菜单，忽略 roleID 入参。
// 非 builtIn 用户按 roleID 查询 role_menus 关联，按树形结构返回（含 button 节点供前端按
// 钮级权限判断）。参 docs/prd/system/menu-dynamic-loading.md §4.2.2。
func (s *AdminService) GetUserMenuTreeByRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) ([]Menu, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user for menu tree by role: %w", err)
	}
	var menus []Menu
	if user.Source == UserSourceBuiltIn {
		menus, err = s.menuRepo.GetAllActive(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		menus, err = s.menuRepo.GetByRole(ctx, roleID)
		if err != nil {
			return nil, err
		}
		if !isBuiltInRole(roleID) {
			allMenus, err := s.menuRepo.GetAllActive(ctx)
			if err != nil {
				return nil, fmt.Errorf("load active menus for custom role: %w", err)
			}
			menus = includeButtonsUnderGrantedMenus(menus, allMenus)
		}
	}
	menus = s.applyLicenseMenuGate(ctx, menus)
	return pruneEmptyDirectories(buildMenuTree(menus)), nil
}

// applyLicenseMenuGate 统一菜单 license 闸门（GetUserMenuTree 与 GetUserMenuTreeByRole 共用）。
// 无 license → 只剩 /license；有 license → 按 feature_code 过滤 + 确保 /license 始终可见；
// 瞬时错误 → fail-open。
func (s *AdminService) applyLicenseMenuGate(ctx context.Context, menus []Menu) []Menu {
	if s.featureGate == nil {
		return menus
	}
	active, err := s.featureGate.HasActiveLicense(ctx)
	switch {
	case err != nil:
		// 瞬时错误（DB 抖动等）→ fail-open，不隐藏菜单；前端 licenseOnlyMode 只在 404 触发。
		s.logger.Warn("license active check failed, skipping menu filter", zap.Error(err))
	case !active:
		// 无 license → 只保留 /license 菜单。/license 可能不在 GetAllActive/GetByUser 结果里
		// （show_status=hide 或 role_menus 未分配），故从 GetTree（含 hide）全集里取，
		// 确保用户能进上传入口。
		all, err := s.menuRepo.GetTree(ctx, nil)
		if err != nil {
			s.logger.Warn("load menu tree for license-only filter failed, skip filter", zap.Error(err))
			return menus
		}
		return filterLicenseOnlyMenus(flattenMenuTree(all))
	default:
		gated := s.applyLicenseFeatureGate(ctx, menus)
		// 有 license 时 /license 也必须可见：用户需要查看/替换/管理 license。
		// 走 GetByRole 的非 builtIn 用户（含系统 admin 角色）若 role_menus 未分配 /license，
		// 菜单会缺失，此处从全集补回，与无 license 分支行为一致。
		return s.ensureLicenseMenuVisible(ctx, gated)
	}
	return menus
}

// ensureLicenseMenuVisible 确保 /license 菜单出现在结果集中。
// 若 menus 已含 /license 直接返回；否则从 GetTree 全集取 status=normal && show_status=show
// 的 /license 节点追加。菜单表小（<200 行），仅在缺失时才查全集。
func (s *AdminService) ensureLicenseMenuVisible(ctx context.Context, menus []Menu) []Menu {
	for _, m := range menus {
		if isLicenseMenu(m) {
			return menus
		}
	}
	all, err := s.menuRepo.GetTree(ctx, nil)
	if err != nil {
		s.logger.Warn("load menu tree for license menu ensure failed, skip", zap.Error(err))
		return menus
	}
	for _, m := range flattenMenuTree(all) {
		if isLicenseMenu(m) && m.Status == MenuStatusNormal && m.ShowStatus == MenuShow {
			menus = append(menus, m)
		}
	}
	return menus
}

// applyLicenseFeatureGate 按 license feature_code 过滤菜单（fail-open）。
// 保留 feature_code 为空（不受控）或任一 code 已授权（OR）的菜单；移除孤儿子节点；
// 无 license / 查询异常时返回原集合（不隐藏，避免锁死 UI）。
func (s *AdminService) applyLicenseFeatureGate(ctx context.Context, menus []Menu) []Menu {
	codeSet := make(map[string]struct{})
	for _, m := range menus {
		for _, c := range m.FeatureCodes {
			if c = strings.TrimSpace(c); c != "" {
				codeSet[c] = struct{}{}
			}
		}
	}
	if len(codeSet) == 0 {
		return menus
	}
	codes := make([]string, 0, len(codeSet))
	for c := range codeSet {
		codes = append(codes, c)
	}
	authorized, err := s.featureGate.FilterAuthorized(ctx, codes)
	if err != nil {
		s.logger.Warn("license feature gate unavailable, skipping menu filter", zap.Error(err))
		return menus
	}
	authSet := make(map[string]struct{}, len(authorized))
	for _, c := range authorized {
		authSet[strings.TrimSpace(c)] = struct{}{}
	}

	kept := make([]Menu, 0, len(menus))
	keptIDs := make(map[uuid.UUID]struct{}, len(menus))
	for _, m := range menus {
		if menuAuthorizedByFeature(m, authSet) {
			kept = append(kept, m)
			keptIDs[m.ID] = struct{}{}
		}
	}
	// 清理孤儿：父节点被过滤的子节点（按钮等）一并移除。层级最多 3 层，两遍收敛。
	for iter := 0; iter < 2; iter++ {
		pruned := make([]Menu, 0, len(kept))
		for _, m := range kept {
			if m.ParentID == nil {
				pruned = append(pruned, m)
				continue
			}
			if _, ok := keptIDs[*m.ParentID]; ok {
				pruned = append(pruned, m)
			}
		}
		kept = pruned
		keptIDs = make(map[uuid.UUID]struct{}, len(kept))
		for _, m := range kept {
			keptIDs[m.ID] = struct{}{}
		}
	}
	return kept
}

func menuAuthorizedByFeature(m Menu, authorized map[string]struct{}) bool {
	if len(m.FeatureCodes) == 0 {
		return true
	}
	for _, c := range m.FeatureCodes {
		if _, ok := authorized[strings.TrimSpace(c)]; ok {
			return true
		}
	}
	return false
}

// pruneEmptyDirectories 移除子树已被过滤光的一级目录（directory），避免空分组残留。
func pruneEmptyDirectories(nodes []Menu) []Menu {
	result := make([]Menu, 0, len(nodes))
	for _, n := range nodes {
		if len(n.Children) > 0 {
			n.Children = pruneEmptyDirectories(n.Children)
		}
		if n.Type == MenuTypeDirectory && len(n.Children) == 0 {
			continue
		}
		result = append(result, n)
	}
	return result
}

// filterLicenseOnlyMenus 无 license 时只保留 /license 菜单（其他全不可见），
// 确保用户能进入上传 License 的入口，其余业务菜单在后端即被裁掉。
func filterLicenseOnlyMenus(menus []Menu) []Menu {
	kept := make([]Menu, 0, len(menus))
	for _, m := range menus {
		if isLicenseMenu(m) {
			kept = append(kept, m)
		}
	}
	return kept
}

func isLicenseMenu(m Menu) bool {
	rp := strings.TrimSpace(m.RoutePath)
	if rp == "" {
		return false
	}
	return rp == "/license" || strings.HasPrefix(rp, "/license/")
}

// flattenMenuTree 把树形菜单展平成列表（保留每个节点，含其 Children 字段）。
func flattenMenuTree(nodes []Menu) []Menu {
	var out []Menu
	for _, n := range nodes {
		out = append(out, n)
		if len(n.Children) > 0 {
			out = append(out, flattenMenuTree(n.Children)...)
		}
	}
	return out
}

// includeButtonsUnderGrantedMenus derives effective operation permissions for
// custom roles. The role editor exposes menu permissions only, so granting a
// page menu also grants its active button children without changing role_menus.
func includeButtonsUnderGrantedMenus(granted, all []Menu) []Menu {
	grantedMenuIDs := make(map[uuid.UUID]struct{}, len(granted))
	seen := make(map[uuid.UUID]struct{}, len(granted))
	result := append([]Menu(nil), granted...)
	for _, menu := range granted {
		seen[menu.ID] = struct{}{}
		if menu.Type == MenuTypeMenu {
			grantedMenuIDs[menu.ID] = struct{}{}
		}
	}

	for _, menu := range all {
		if menu.Type != MenuTypeButton || menu.ParentID == nil {
			continue
		}
		if _, ok := grantedMenuIDs[*menu.ParentID]; !ok {
			continue
		}
		if _, ok := seen[menu.ID]; ok {
			continue
		}
		result = append(result, menu)
		seen[menu.ID] = struct{}{}
	}
	return result
}

// ==================== Password Change ====================

// ChangePasswordRequest 是 service.ChangePassword 的明文入参。
// HTTP 端点不直接绑定本结构（见 ChangePasswordHTTPRequest），handler 解密后构造。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePasswordHTTPRequest 是 POST /api/v1/auth/change-password 的请求体。
// 旧/新密码均需 RSA-OAEP 加密传输。两个密文使用同一 key_id（一次公钥）。
//
// T-0120：去掉 `required` tag — handler 内做二选一校验：
//   - 加密路径（secure context）：EncryptedOldPassword + EncryptedNewPassword + KeyID 同时非空
//   - 明文路径（仅 LoginCrypto.AllowPlaintext=true 时启用）：OldPassword + NewPassword 非空
//
// 两路径都不满足 → handler 返 400 missing password。
type ChangePasswordHTTPRequest struct {
	EncryptedOldPassword string `json:"encrypted_old_password"`
	EncryptedNewPassword string `json:"encrypted_new_password"`
	KeyID                string `json:"key_id"`
	OldPassword          string `json:"old_password"` // T-0120 plaintext fallback
	NewPassword          string `json:"new_password"` // T-0120 plaintext fallback
}

// ChangePassword verifies the old password and updates to the new one.
func (s *AdminService) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) (retErr error) {
	username := ""
	defer func() {
		s.auditPasswordChange(ctx, userID, username, retErr)
	}()

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	username = user.Username

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return commonerrors.NewBusinessError(7020, "old password is incorrect", commonerrors.ErrUnauthorized)
	}

	// P1-② 密码强度校验（与 CreateUser / ResetPassword 同套规则）
	if err := ValidatePassword(req.NewPassword,
		PasswordPolicySnapshotFromSecurityPolicy(s.policySnapshot(ctx))); err != nil {
		return fmt.Errorf("%w: %w", err, commonerrors.ErrInvalidInput)
	}

	newHash, err := hashSecret([]byte(req.NewPassword))
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	// 先建立撤销屏障，避免密码已经更新后因 Redis 故障返回失败，导致用户继续
	// 使用已经失效的旧密码重试。更新成功后再刷新一次时间戳，覆盖并发登录窗口。
	if s.revoker != nil {
		if err := s.revoker.Revoke(ctx, userID); err != nil {
			return fmt.Errorf("revoke tokens before password change: %w", err)
		}
	}
	if err := s.userRepo.UpdatePassword(ctx, userID, string(newHash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if s.revoker != nil {
		if err := s.revoker.Revoke(ctx, userID); err != nil {
			s.logger.Error("refresh token revocation after password change failed",
				zap.String("user_id", userID.String()), zap.Error(err))
		}
	}
	return nil
}

func (s *AdminService) auditPasswordChange(ctx context.Context, targetID uuid.UUID, username string, err error) {
	entry := auditEntryFromContext(ctx)
	entry.Action = audit.ActionPasswordChange
	entry.ResourceType = audit.ResourceUser
	entry.ResourceID = targetID.String()
	entry.Success = err == nil
	entry.Details = map[string]interface{}{
		"success":        err == nil,
		"target_user_id": targetID.String(),
	}
	if username != "" {
		entry.Details["target_username"] = username
	}
	if err != nil {
		entry.ErrorMessage = err.Error()
	}
	audit.Log(ctx, entry)
}

// ==================== Role User List ====================

// RoleUserItem represents a user summary in a role's user list.
type RoleUserItem struct {
	ID          uuid.UUID  `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Email       string     `json:"email,omitempty"`
	Status      UserStatus `json:"status"`
}

// GetRoleUsers returns a paginated list of users assigned to a role.
func (s *AdminService) GetRoleUsers(ctx context.Context, roleID uuid.UUID, filter model.ListRequest) (*model.ListResponse[RoleUserItem], error) {
	users, total, err := s.roleRepo.ListRoleUsers(ctx, roleID, filter.Limit(), filter.Offset())
	if err != nil {
		return nil, fmt.Errorf("list role users: %w", err)
	}
	return model.NewListResponse(users, total, filter.Page, filter.PageSize), nil
}
