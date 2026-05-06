package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
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

// PermissionInvalidator 抽象 PermissionService 的失效操作，便于注入与测试。
// 详见 PRD docs/prd/system/users.md §10 DoD「数据权限缓存一致性」。
type PermissionInvalidator interface {
	InvalidateUserCache(ctx context.Context, userID uuid.UUID) error
}

// AdminService provides user management, authentication, and RBAC functionality.
type AdminService struct {
	userRepo        UserRepository
	roleRepo        RoleRepository
	menuRepo        MenuRepository
	auditRepo       AuditRepository
	jwt             *JWTService
	revoker         *TokenRevoker
	permInvalidator PermissionInvalidator
	metrics         *AdminMetrics
	logger          *zap.Logger
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
	return &AdminService{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		menuRepo:  menuRepo,
		auditRepo: auditRepo,
		jwt:       jwtService,
		logger:    logger.Named("admin"),
	}
}

// SetTokenRevoker 注入 token 撤销器，启用强制下线能力。
// 不调用时 ForceLogout 退化为 no-op（仅记录日志），以便单元测试无需 Redis 也能跑。
func (s *AdminService) SetTokenRevoker(r *TokenRevoker) {
	s.revoker = r
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
		return nil, commonerrors.NewBusinessError(7012, "account temporarily locked due to too many failed attempts", commonerrors.ErrForbidden)
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

	tokenPair, err := s.jwt.GenerateTokenPair(&Claims{
		UserID:        user.ID,
		Username:      user.Username,
		Carrier:       user.Carrier,
		Roles:         roleNames,
		CurrentRoleID: s.getDefaultRoleID(ctx, user.ID, roles),
	})
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("update last login", zap.Error(err))
	}

	return tokenPair, nil
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
		Carrier:       user.Carrier,
		Roles:         roleNames,
		CurrentRoleID: defaultRoleID,
	})
}

// CreateUser creates a new user account with role assignments.
// If role assignment fails, the created user is rolled back to maintain consistency.
func (s *AdminService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// 通过本接口创建的用户固定为管理员添加。
	// LDAP 同步任务走独立内部路径写 UserSourceLDAP；seed 文件写 UserSourceBuiltIn。
	user := &User{
		Username:     req.Username,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Phone:        req.Phone,
		Description:  req.Description,
		ExpireAt:     req.ExpireAt,
		Carrier:      req.Carrier,
		Status:       UserStatusActive,
		Source:       UserSourceAdmin,
	}
	if op := operatorIDFromContext(ctx); op != nil {
		user.CreatedBy = op
		user.UpdatedBy = op
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
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
			return nil, fmt.Errorf("assign role %s: %w", roleID, err)
		}
	}

	roles, _ := s.roleRepo.GetUserRoles(ctx, user.ID)
	user.Roles = roles
	return user, nil
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
	if req.Carrier != nil {
		user.Carrier = req.Carrier
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

	// 角色或运营商变更 → 失效该用户的可见分组缓存（PRD §10 DoD）。
	if req.RoleIDs != nil || req.Carrier != nil {
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
// 复制项：display_name / email / phone / carrier / 当前 roles。
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
	hash, err := bcrypt.GenerateFromPassword([]byte(tempPwd), bcrypt.DefaultCost)
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
		Carrier:      src.Carrier,
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
func (s *AdminService) ListUsers(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error) {
	result, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// 批量加载所有用户的角色
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
			// 填充角色数据
			for i := range result.Items {
				result.Items[i].Roles = rolesMap[result.Items[i].ID]
			}
		}
	}

	return result, nil
}

// GetUser returns a user by ID with roles loaded.
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
	return user, nil
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
	return s.roleRepo.List(ctx)
}

// ListRolesPaginated returns roles with pagination support.
func (s *AdminService) ListRolesPaginated(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error) {
	return s.roleRepo.ListWithPagination(ctx, filter)
}

// ResetPassword resets a user's password to the provided new password.
// LDAP 用户（source=LDAP）密码归属外部域，拒绝重置 → HTTP 409。
// 重置成功后同时清除登录失败计数与锁定状态，避免管理员重置后仍处于锁定。
func (s *AdminService) ResetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user for reset password: %w", err)
	}
	if user.Source == UserSourceLDAP {
		return ErrLDAPPasswordExternal
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, id, string(hash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	// 与 LockUserByUsername 保持一致：ResetLoginSecurity 不在 UserRepository 接口上，
	// 通过类型断言直接调具体实现，失败仅警告（密码已成功重置）。
	if repo, ok := s.userRepo.(*PgUserRepository); ok {
		if err := repo.ResetLoginSecurity(ctx, id); err != nil {
			s.logger.Warn("reset login security after password reset", zap.Error(err))
		}
	}
	return nil
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

// GetRole returns a role by ID with permissions loaded.
func (s *AdminService) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	perms, err := s.roleRepo.GetPermissions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}
	role.Permissions = perms
	return role, nil
}

// CreateRole creates a new role with optional permissions.
func (s *AdminService) CreateRole(ctx context.Context, req CreateRoleRequest) (*Role, error) {
	role := &Role{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	if len(req.Permissions) > 0 {
		perms := make([]Permission, len(req.Permissions))
		for i, p := range req.Permissions {
			perms[i] = Permission{
				RoleID:   role.ID,
				Resource: p.Resource,
				Action:   p.Action,
			}
		}
		if err := s.roleRepo.AddPermissions(ctx, role.ID, perms); err != nil {
			return nil, fmt.Errorf("add permissions: %w", err)
		}
	}

	return s.GetRole(ctx, role.ID)
}

// UpdateRole updates an existing role's fields and replaces its permissions.
func (s *AdminService) UpdateRole(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get role for update: %w", err)
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}

	if req.Permissions != nil {
		if err := s.roleRepo.RemoveAllPermissions(ctx, id); err != nil {
			return nil, fmt.Errorf("remove old permissions: %w", err)
		}
		if len(req.Permissions) > 0 {
			perms := make([]Permission, len(req.Permissions))
			for i, p := range req.Permissions {
				perms[i] = Permission{
					RoleID:   id,
					Resource: p.Resource,
					Action:   p.Action,
				}
			}
			if err := s.roleRepo.AddPermissions(ctx, id, perms); err != nil {
				return nil, fmt.Errorf("add new permissions: %w", err)
			}
		}
	}

	return s.GetRole(ctx, id)
}

// DeleteRole deletes a role by ID.
func (s *AdminService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return s.roleRepo.Delete(ctx, id)
}

// ListAllPermissions returns all permissions across all roles.
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
		Type:          req.Type,
		PermissionKey: req.PermissionKey,
		ParentID:      req.ParentID,
		SortOrder:     req.SortOrder,
		RoutePath:     req.RoutePath,
		ComponentPath: req.ComponentPath,
		Icon:          req.Icon,
		ShowStatus:    req.ShowStatus,
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
func (s *AdminService) GetUserMenuTree(ctx context.Context, userID uuid.UUID) ([]Menu, error) {
	menus, err := s.menuRepo.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build tree and filter out button type items for display
	tree := buildMenuTree(menus)
	return filterTreeForDisplay(tree), nil
}

// SetRoleMenus sets the menu permissions for a role.
func (s *AdminService) SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error {
	// Verify role exists
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return commonerrors.ErrNotFound
	}

	return s.menuRepo.SetRoleMenus(ctx, roleID, menuIDs, operatorID)
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
func buildMenuTree(menus []Menu) []Menu {
	menuMap := make(map[uuid.UUID]*Menu)
	var roots []Menu

	for i := range menus {
		menuMap[menus[i].ID] = &menus[i]
		menus[i].Children = nil
	}

	for _, m := range menus {
		if m.ParentID == nil {
			roots = append(roots, m)
		} else if parent, ok := menuMap[*m.ParentID]; ok {
			parent.Children = append(parent.Children, m)
		}
	}

	return roots
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
		Carrier:       user.Carrier,
		Roles:         roleNames,
		CurrentRoleID: &targetRoleID,
	})
}

// GetUserMenuTreeByRole returns the menu tree based on the user's current active role.
func (s *AdminService) GetUserMenuTreeByRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) ([]Menu, error) {
	return s.menuRepo.GetByRole(ctx, roleID)
}

// ==================== Password Change ====================

// ChangePasswordRequest is the input for changing a user's own password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword verifies the old password and updates to the new one.
func (s *AdminService) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return commonerrors.NewBusinessError(7020, "old password is incorrect", commonerrors.ErrUnauthorized)
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(newHash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
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
