package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/omcgo/omcgo/internal/admin/audit"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Login failure sentinel errors used for audit log classification.
// These are internal to the service and handler; callers receive generic ErrUnauthorized/ErrForbidden.
var (
	errLoginUserNotFound    = fmt.Errorf("user not found")
	errLoginAccountDisabled = fmt.Errorf("account disabled")
	errLoginWrongPassword   = fmt.Errorf("wrong password")
)

// AdminService provides user management, authentication, and RBAC functionality.
type AdminService struct {
	userRepo  UserRepository
	roleRepo  RoleRepository
	menuRepo  MenuRepository
	auditRepo AuditRepository
	jwt       *JWTService
	logger    *zap.Logger
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

	user := &User{
		Username:     req.Username,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Carrier:      req.Carrier,
		Status:       UserStatusActive,
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
	if req.Carrier != nil {
		user.Carrier = req.Carrier
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	roles, _ := s.roleRepo.GetUserRoles(ctx, user.ID)
	user.Roles = roles
	return user, nil
}

// DeleteUser deletes a user account.
func (s *AdminService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.Delete(ctx, id)
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
func (s *AdminService) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return fmt.Errorf("get user for role assignment: %w", err)
	}
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return fmt.Errorf("get role for assignment: %w", err)
	}
	return s.roleRepo.AssignRole(ctx, userID, roleID)
}

// RemoveRole removes a role from a user.
func (s *AdminService) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.roleRepo.RemoveRole(ctx, userID, roleID)
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
func (s *AdminService) ResetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.userRepo.UpdatePassword(ctx, id, string(hash))
}

// LockUser disables a user account by setting its status to disabled.
func (s *AdminService) LockUser(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user for lock: %w", err)
	}
	user.Status = UserStatusDisabled
	return s.userRepo.Update(ctx, user)
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
