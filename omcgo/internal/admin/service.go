package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
)

// AdminService provides user management, authentication, and RBAC functionality.
type AdminService struct {
	userRepo  UserRepository
	roleRepo  RoleRepository
	auditRepo AuditRepository
	jwt       *JWTService
	logger    *zap.Logger
}

// NewAdminService creates a new AdminService.
func NewAdminService(
	userRepo UserRepository,
	roleRepo RoleRepository,
	auditRepo AuditRepository,
	jwtService *JWTService,
	logger *zap.Logger,
) *AdminService {
	return &AdminService{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		auditRepo: auditRepo,
		jwt:       jwtService,
		logger:    logger.Named("admin"),
	}
}

// Login authenticates a user and returns a JWT token pair.
func (s *AdminService) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, commonerrors.ErrUnauthorized
	}

	if user.Status != UserStatusActive {
		return nil, commonerrors.NewBusinessError(7001, "account is disabled", commonerrors.ErrForbidden)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, commonerrors.ErrUnauthorized
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
		UserID:   user.ID,
		Username: user.Username,
		Carrier:  user.Carrier,
		Roles:    roleNames,
	})
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("update last login", zap.Error(err))
	}

	return tokenPair, nil
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

	return s.jwt.GenerateTokenPair(&Claims{
		UserID:   user.ID,
		Username: user.Username,
		Carrier:  user.Carrier,
		Roles:    roleNames,
	})
}

// CreateUser creates a new user account.
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
			s.logger.Warn("assign role during user creation",
				zap.String("user_id", user.ID.String()),
				zap.String("role_id", roleID.String()),
				zap.Error(err),
			)
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
		return nil, err
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
		return nil, err
	}

	roles, _ := s.roleRepo.GetUserRoles(ctx, user.ID)
	user.Roles = roles
	return user, nil
}

// DeleteUser deletes a user account.
func (s *AdminService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.Delete(ctx, id)
}

// ListUsers returns a filtered, paginated list of users.
func (s *AdminService) ListUsers(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error) {
	return s.userRepo.List(ctx, filter)
}

// GetUser returns a user by ID with roles loaded.
func (s *AdminService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
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
		return err
	}
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return err
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

// ListRoles returns all roles.
func (s *AdminService) ListRoles(ctx context.Context) ([]Role, error) {
	return s.roleRepo.List(ctx)
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
		return err
	}
	user.Status = UserStatusDisabled
	return s.userRepo.Update(ctx, user)
}

// UnlockUser re-enables a user account by setting its status to active.
func (s *AdminService) UnlockUser(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	user.Status = UserStatusActive
	return s.userRepo.Update(ctx, user)
}

// GetRole returns a role by ID with permissions loaded.
func (s *AdminService) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
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
		return nil, err
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
