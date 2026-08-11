package admin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// --- Mock Repositories ---

type mockUserRepo struct {
	createFn            func(ctx context.Context, user *User) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*User, error)
	getByUsernameFn     func(ctx context.Context, username string) (*User, error)
	updateFn            func(ctx context.Context, user *User) error
	updatePasswordFn    func(ctx context.Context, id uuid.UUID, passwordHash string) error
	deleteFn            func(ctx context.Context, id uuid.UUID) error
	listFn              func(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error)
	getUsernamesByIDsFn func(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
	updateLastLoginFn   func(ctx context.Context, id uuid.UUID) error
}

func (m *mockUserRepo) Create(ctx context.Context, user *User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	user.ID = uuid.New()
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return nil, commonerrors.ErrNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, user *User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, id, passwordHash)
	}
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserRepo) List(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &model.ListResponse[User]{Items: []User{}}, nil
}

func (m *mockUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	if m.updateLastLoginFn != nil {
		return m.updateLastLoginFn(ctx, id)
	}
	return nil
}

// GetUsernamesByIDs：mockUserRepo 默认返回空 map（测试不关心 enrich 结果时 no-op）。
func (m *mockUserRepo) GetUsernamesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if m.getUsernamesByIDsFn != nil {
		return m.getUsernamesByIDsFn(ctx, ids)
	}
	return map[uuid.UUID]string{}, nil
}

type mockRoleRepo struct {
	createFn                func(ctx context.Context, role *Role) error
	getByIDFn               func(ctx context.Context, id uuid.UUID) (*Role, error)
	getByNameFn             func(ctx context.Context, name string) (*Role, error)
	updateFn                func(ctx context.Context, role *Role) error
	deleteFn                func(ctx context.Context, id uuid.UUID) error
	getDeviceGroupDataFn    func(ctx context.Context, roleID uuid.UUID) (*RoleDeviceGroupData, error)
	setDeviceGroupDataFn    func(ctx context.Context, roleID uuid.UUID, data RoleDeviceGroupData) error
	getRoleAPIEndpointIDsFn func(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	setRoleAPIEndpointsFn   func(ctx context.Context, roleID uuid.UUID, endpointIDs []uuid.UUID) error
	listFn                  func(ctx context.Context) ([]Role, error)
	listWithPaginationFn    func(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error)
	assignRoleFn            func(ctx context.Context, userID, roleID uuid.UUID) error
	removeRoleFn            func(ctx context.Context, userID, roleID uuid.UUID) error
	getUserRolesFn          func(ctx context.Context, userID uuid.UUID) ([]Role, error)
	listUserIDsByRoleFn     func(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	getPermissionsFn        func(ctx context.Context, roleID uuid.UUID) ([]Permission, error)
	checkPermissionFn       func(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	listAllPermissionsFn    func(ctx context.Context) ([]Permission, error)
	addPermissionsFn        func(ctx context.Context, roleID uuid.UUID, perms []Permission) error
	removeAllPermissionsFn  func(ctx context.Context, roleID uuid.UUID) error
}

func (m *mockRoleRepo) Create(ctx context.Context, role *Role) error {
	if m.createFn != nil {
		return m.createFn(ctx, role)
	}
	return nil
}

func (m *mockRoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}

func (m *mockRoleRepo) GetByName(ctx context.Context, name string) (*Role, error) {
	if m.getByNameFn != nil {
		return m.getByNameFn(ctx, name)
	}
	return nil, commonerrors.ErrNotFound
}

func (m *mockRoleRepo) Update(ctx context.Context, role *Role) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, role)
	}
	return nil
}

func (m *mockRoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockRoleRepo) GetDeviceGroupData(ctx context.Context, roleID uuid.UUID) (*RoleDeviceGroupData, error) {
	if m.getDeviceGroupDataFn != nil {
		return m.getDeviceGroupDataFn(ctx, roleID)
	}
	return &RoleDeviceGroupData{}, nil
}

func (m *mockRoleRepo) SetDeviceGroupData(ctx context.Context, roleID uuid.UUID, data RoleDeviceGroupData) error {
	if m.setDeviceGroupDataFn != nil {
		return m.setDeviceGroupDataFn(ctx, roleID, data)
	}
	return nil
}

func (m *mockRoleRepo) GetRoleApiEndpointIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	if m.getRoleAPIEndpointIDsFn != nil {
		return m.getRoleAPIEndpointIDsFn(ctx, roleID)
	}
	return []uuid.UUID{}, nil
}

func (m *mockRoleRepo) SetRoleApiEndpoints(ctx context.Context, roleID uuid.UUID, endpointIDs []uuid.UUID) error {
	if m.setRoleAPIEndpointsFn != nil {
		return m.setRoleAPIEndpointsFn(ctx, roleID, endpointIDs)
	}
	return nil
}

func (m *mockRoleRepo) List(ctx context.Context) ([]Role, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []Role{}, nil
}

func (m *mockRoleRepo) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if m.assignRoleFn != nil {
		return m.assignRoleFn(ctx, userID, roleID)
	}
	return nil
}

func (m *mockRoleRepo) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if m.removeRoleFn != nil {
		return m.removeRoleFn(ctx, userID, roleID)
	}
	return nil
}

func (m *mockRoleRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	if m.getUserRolesFn != nil {
		return m.getUserRolesFn(ctx, userID)
	}
	return []Role{}, nil
}

func (m *mockRoleRepo) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]Permission, error) {
	if m.getPermissionsFn != nil {
		return m.getPermissionsFn(ctx, roleID)
	}
	return []Permission{}, nil
}

func (m *mockRoleRepo) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	if m.checkPermissionFn != nil {
		return m.checkPermissionFn(ctx, userID, resource, action)
	}
	return false, nil
}

func (m *mockRoleRepo) ListAllPermissions(ctx context.Context) ([]Permission, error) {
	if m.listAllPermissionsFn != nil {
		return m.listAllPermissionsFn(ctx)
	}
	return []Permission{}, nil
}

func (m *mockRoleRepo) AddPermissions(ctx context.Context, roleID uuid.UUID, perms []Permission) error {
	if m.addPermissionsFn != nil {
		return m.addPermissionsFn(ctx, roleID, perms)
	}
	return nil
}

func (m *mockRoleRepo) RemoveAllPermissions(ctx context.Context, roleID uuid.UUID) error {
	if m.removeAllPermissionsFn != nil {
		return m.removeAllPermissionsFn(ctx, roleID)
	}
	return nil
}

func (m *mockRoleRepo) GetUserRolesBatch(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]Role, error) {
	return nil, nil
}
func (m *mockRoleRepo) ListWithPagination(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error) {
	if m.listWithPaginationFn != nil {
		return m.listWithPaginationFn(ctx, filter)
	}
	return model.NewListResponse([]Role{}, 0, 1, 20), nil
}
func (m *mockRoleRepo) GetDefaultRoleID(_ context.Context, _ uuid.UUID) (*uuid.UUID, error) {
	return nil, nil
}
func (m *mockRoleRepo) SetDefaultRole(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (m *mockRoleRepo) ListRoleUsers(_ context.Context, _ uuid.UUID, _, _ int) ([]RoleUserItem, int64, error) {
	return []RoleUserItem{}, 0, nil
}

func (m *mockRoleRepo) ListUserIDsByRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	if m.listUserIDsByRoleFn != nil {
		return m.listUserIDsByRoleFn(ctx, roleID)
	}
	return nil, nil
}

type mockAuditRepo struct {
	createFn func(ctx context.Context, log *AuditLog) error
	listFn   func(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[AuditLog], error)
}

func (m *mockAuditRepo) Create(ctx context.Context, log *AuditLog) error {
	if m.createFn != nil {
		return m.createFn(ctx, log)
	}
	return nil
}

func (m *mockAuditRepo) List(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[AuditLog], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &model.ListResponse[AuditLog]{Items: []AuditLog{}}, nil
}

type mockMenuRepo struct {
	getAllActiveFn   func(ctx context.Context) ([]Menu, error)
	getByRoleFn      func(ctx context.Context, roleID uuid.UUID) ([]Menu, error)
	getTreeFn        func(ctx context.Context, status *MenuStatus) ([]Menu, error)
	getRoleMenuIDsFn func(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	setRoleMenusFn   func(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error
}

func (m *mockMenuRepo) Create(_ context.Context, _ *Menu, _ uuid.UUID) error  { return nil }
func (m *mockMenuRepo) GetByID(_ context.Context, _ uuid.UUID) (*Menu, error) { return nil, nil }
func (m *mockMenuRepo) GetByPermissionKey(_ context.Context, _ string) (*Menu, error) {
	return nil, nil
}
func (m *mockMenuRepo) List(_ context.Context, _ MenuFilter) (*model.ListResponse[Menu], error) {
	return &model.ListResponse[Menu]{Items: []Menu{}}, nil
}
func (m *mockMenuRepo) Update(_ context.Context, _ uuid.UUID, _ *UpdateMenuRequest, _ uuid.UUID) error {
	return nil
}
func (m *mockMenuRepo) Delete(_ context.Context, _ []uuid.UUID) error { return nil }
func (m *mockMenuRepo) GetTree(ctx context.Context, status *MenuStatus) ([]Menu, error) {
	if m.getTreeFn != nil {
		return m.getTreeFn(ctx, status)
	}
	return nil, nil
}
func (m *mockMenuRepo) GetByRole(ctx context.Context, roleID uuid.UUID) ([]Menu, error) {
	if m.getByRoleFn != nil {
		return m.getByRoleFn(ctx, roleID)
	}
	return nil, nil
}
func (m *mockMenuRepo) GetByUser(_ context.Context, _ uuid.UUID) ([]Menu, error) {
	return nil, nil
}
func (m *mockMenuRepo) GetAllActive(ctx context.Context) ([]Menu, error) {
	if m.getAllActiveFn != nil {
		return m.getAllActiveFn(ctx)
	}
	return nil, nil
}
func (m *mockMenuRepo) SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error {
	if m.setRoleMenusFn != nil {
		return m.setRoleMenusFn(ctx, roleID, menuIDs, operatorID)
	}
	return nil
}
func (m *mockMenuRepo) GetRoleMenuIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	if m.getRoleMenuIDsFn != nil {
		return m.getRoleMenuIDsFn(ctx, roleID)
	}
	return nil, nil
}

// --- Helper ---

func newTestService(userRepo *mockUserRepo, roleRepo *mockRoleRepo, auditRepo *mockAuditRepo) *AdminService {
	jwt, err := NewJWTService("test-secret-minimum-32-characters!!")
	if err != nil {
		panic(err)
	}
	logger := zap.NewNop()
	return NewAdminService(userRepo, roleRepo, &mockMenuRepo{}, auditRepo, jwt, logger)
}

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return string(hash)
}

// --- Tests ---

func TestAdminService_Login_Success(t *testing.T) {
	userID := uuid.New()
	passwordHash := hashPassword("correctpassword")

	userRepo := &mockUserRepo{
		getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
			return &User{
				ID:           userID,
				Username:     username,
				PasswordHash: passwordHash,
				Status:       UserStatusActive,
			}, nil
		},
	}

	roleRepo := &mockRoleRepo{
		getUserRolesFn: func(ctx context.Context, uid uuid.UUID) ([]Role, error) {
			return []Role{{Name: "admin"}}, nil
		},
	}

	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})

	pair, err := svc.Login(context.Background(), "admin", "correctpassword")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
}

func TestAdminService_Login_WrongPassword(t *testing.T) {
	userRepo := &mockUserRepo{
		getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
			return &User{
				ID:           uuid.New(),
				Username:     username,
				PasswordHash: hashPassword("correctpassword"),
				Status:       UserStatusActive,
			}, nil
		},
	}

	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})

	_, err := svc.Login(context.Background(), "admin", "wrongpassword")
	assert.ErrorIs(t, err, commonerrors.ErrUnauthorized)
}

func TestAdminService_Login_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
			return nil, commonerrors.ErrNotFound
		},
	}

	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})

	_, err := svc.Login(context.Background(), "nonexistent", "password")
	assert.ErrorIs(t, err, commonerrors.ErrUnauthorized)
}

func TestAdminService_Login_DisabledAccount(t *testing.T) {
	userRepo := &mockUserRepo{
		getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
			return &User{
				ID:           uuid.New(),
				Username:     username,
				PasswordHash: hashPassword("password"),
				Status:       UserStatusDisabled,
			}, nil
		},
	}

	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})

	_, err := svc.Login(context.Background(), "admin", "password")
	assert.Error(t, err)
}

func TestAdminService_RefreshToken_Success(t *testing.T) {
	userID := uuid.New()
	jwt, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	original, err := jwt.GenerateTokenPair(&Claims{
		UserID:   userID,
		Username: "admin",
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)

	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID, Username: "admin", Status: UserStatusActive}, nil
		},
	}
	roleRepo := &mockRoleRepo{
		getUserRolesFn: func(ctx context.Context, uid uuid.UUID) ([]Role, error) {
			return []Role{{Name: "admin"}, {Name: "operator"}}, nil
		},
	}

	svc := NewAdminService(userRepo, roleRepo, &mockMenuRepo{}, &mockAuditRepo{}, jwt, zap.NewNop())

	pair, err := svc.RefreshToken(context.Background(), original.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEqual(t, original.AccessToken, pair.AccessToken)
}

func TestAdminService_RefreshToken_DisabledUser(t *testing.T) {
	userID := uuid.New()
	jwt, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	original, err := jwt.GenerateTokenPair(&Claims{
		UserID:   userID,
		Username: "admin",
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)

	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID, Username: "admin", Status: UserStatusDisabled}, nil
		},
	}

	svc := NewAdminService(userRepo, &mockRoleRepo{}, &mockMenuRepo{}, &mockAuditRepo{}, jwt, zap.NewNop())

	_, err = svc.RefreshToken(context.Background(), original.RefreshToken)
	assert.Error(t, err)
}

func TestAdminService_CreateUser_Success(t *testing.T) {
	roleID := uuid.New()
	operatorID := uuid.New()
	var createdUser *User

	userRepo := &mockUserRepo{
		createFn: func(ctx context.Context, user *User) error {
			user.ID = uuid.New()
			user.CreatedAt = time.Now()
			user.UpdatedAt = time.Now()
			createdUser = user
			return nil
		},
	}
	roleRepo := &mockRoleRepo{
		getUserRolesFn: func(ctx context.Context, uid uuid.UUID) ([]Role, error) {
			return []Role{{ID: roleID, Name: "operator"}}, nil
		},
	}

	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})

	ctx := context.WithValue(context.Background(), CtxKeyUserID, operatorID)
	user, err := svc.CreateUser(ctx, CreateUserRequest{
		Username:    "newuser",
		Password:    "password123",
		DisplayName: "New User",
		Email:       "new@example.com",
		RoleIDs:     []uuid.UUID{roleID},
	})

	require.NoError(t, err)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, UserStatusActive, user.Status)
	assert.NotEmpty(t, createdUser.PasswordHash)
	require.NotNil(t, createdUser.CreatedBy)
	require.NotNil(t, createdUser.UpdatedBy)
	assert.Equal(t, operatorID, *createdUser.CreatedBy)
	assert.Equal(t, operatorID, *createdUser.UpdatedBy)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(createdUser.PasswordHash), []byte("password123")))
}

func TestAdminService_ListUsers_AdminSourceWithoutOperatorShowsAdmin(t *testing.T) {
	userRepo := &mockUserRepo{
		listFn: func(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error) {
			return &model.ListResponse[User]{
				Items:      []User{{ID: uuid.New(), Username: "verify_system_user", Source: UserSourceAdmin}},
				Total:      1,
				Page:       1,
				PageSize:   20,
				TotalPages: 1,
			}, nil
		},
	}
	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})

	result, err := svc.ListUsers(context.Background(), UserFilter{})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "admin", result.Items[0].CreatorUsername)
	assert.Equal(t, "admin", result.Items[0].UpdaterUsername)
}

func TestAdminService_ListRolesPaginated_EnrichesOperatorUsernames(t *testing.T) {
	operatorID := uuid.New()
	legacyRoleID := uuid.New()
	roleRepo := &mockRoleRepo{
		listWithPaginationFn: func(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error) {
			return &model.ListResponse[Role]{
				Items: []Role{
					{ID: uuid.New(), Name: "custom", CreatedBy: &operatorID, UpdatedBy: &operatorID},
					{ID: legacyRoleID, Name: "legacy_custom"},
					{ID: uuid.New(), Name: "admin", IsSystem: true},
				},
				Total:      3,
				Page:       1,
				PageSize:   20,
				TotalPages: 1,
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		getUsernamesByIDsFn: func(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
			assert.Contains(t, ids, operatorID)
			return map[uuid.UUID]string{operatorID: "admin"}, nil
		},
	}
	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})

	result, err := svc.ListRolesPaginated(context.Background(), RoleFilter{})

	require.NoError(t, err)
	require.Len(t, result.Items, 3)
	assert.Equal(t, "admin", result.Items[0].CreatorUsername)
	assert.Equal(t, "admin", result.Items[0].UpdaterUsername)
	assert.Equal(t, "admin", result.Items[1].CreatorUsername)
	assert.Equal(t, "admin", result.Items[1].UpdaterUsername)
	assert.Empty(t, result.Items[2].CreatorUsername)
	assert.Empty(t, result.Items[2].UpdaterUsername)
}

func TestAdminService_UpdateUser_Success(t *testing.T) {
	userID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{
				ID:          userID,
				Username:    "testuser",
				DisplayName: "Old Name",
				Email:       "old@example.com",
				Status:      UserStatusActive,
			}, nil
		},
	}
	roleRepo := &mockRoleRepo{
		getUserRolesFn: func(ctx context.Context, uid uuid.UUID) ([]Role, error) {
			return []Role{}, nil
		},
	}

	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})

	newName := "New Name"
	user, err := svc.UpdateUser(context.Background(), userID, UpdateUserRequest{
		DisplayName: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", user.DisplayName)
	assert.Equal(t, "old@example.com", user.Email)
}

func TestAdminService_DeleteUser(t *testing.T) {
	deleteCalled := false
	userID := uuid.New()
	userRepo := &mockUserRepo{
		// service.DeleteUser 在删除前需先 GetByID 校验 source（非内置用户才允许）。
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Source: UserSourceAdmin}, nil
		},
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}

	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})

	err := svc.DeleteUser(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestAdminService_DeleteUser_BuiltInProtected(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Source: UserSourceBuiltIn}, nil
		},
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			t.Fatal("Delete should not be called for built-in user")
			return nil
		},
	}

	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})

	err := svc.DeleteUser(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBuiltInUserProtected)
}

func TestAdminService_AssignRole(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	assignCalled := false

	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID, Status: UserStatusActive}, nil
		},
	}
	roleRepo := &mockRoleRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Role, error) {
			return &Role{ID: roleID, Name: "operator"}, nil
		},
		assignRoleFn: func(ctx context.Context, uid, rid uuid.UUID) error {
			assignCalled = true
			assert.Equal(t, userID, uid)
			assert.Equal(t, roleID, rid)
			return nil
		},
	}

	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})

	err := svc.AssignRole(context.Background(), userID, roleID)
	require.NoError(t, err)
	assert.True(t, assignCalled)
}

func TestAdminService_CheckPermission(t *testing.T) {
	userID := uuid.New()

	roleRepo := &mockRoleRepo{
		checkPermissionFn: func(ctx context.Context, uid uuid.UUID, resource, action string) (bool, error) {
			if resource == "devices" && action == "read" {
				return true, nil
			}
			return false, nil
		},
	}

	svc := newTestService(&mockUserRepo{}, roleRepo, &mockAuditRepo{})

	ok, err := svc.CheckPermission(context.Background(), userID, "devices", "read")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = svc.CheckPermission(context.Background(), userID, "users", "admin")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAdminService_SetRoleMenus_ExpandsAncestors(t *testing.T) {
	// 菜单结构：
	//   设备管理 (directory) ← parent=nil
	//     └─ 设备列表 (menu) ← parent=设备管理
	//          └─ 查询 (button) ← parent=设备列表
	dirID := uuid.New()
	menuID := uuid.New()
	btnID := uuid.New()
	roleID := uuid.New()
	allMenus := []Menu{
		{ID: dirID, Name: "设备管理", Type: MenuTypeDirectory},
		{ID: menuID, Name: "设备列表", Type: MenuTypeMenu, ParentID: &dirID},
		{ID: btnID, Name: "查询", Type: MenuTypeButton, ParentID: &menuID},
	}

	cases := []struct {
		name   string
		input  []uuid.UUID
		wantIn []uuid.UUID // 期望保存集合至少包含这些 ID（顺序不约束）
	}{
		{
			name:   "仅授叶子按钮 → 自动补两层祖先",
			input:  []uuid.UUID{btnID},
			wantIn: []uuid.UUID{btnID, menuID, dirID},
		},
		{
			name:   "仅授中间 menu → 自动补 directory",
			input:  []uuid.UUID{menuID},
			wantIn: []uuid.UUID{menuID, dirID},
		},
		{
			name:   "授 directory → 不变",
			input:  []uuid.UUID{dirID},
			wantIn: []uuid.UUID{dirID},
		},
		{
			name:   "空输入 → 空保存",
			input:  []uuid.UUID{},
			wantIn: []uuid.UUID{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured []uuid.UUID
			menuRepo := &mockMenuRepo{
				getAllActiveFn: func(_ context.Context) ([]Menu, error) {
					return allMenus, nil
				},
				setRoleMenusFn: func(_ context.Context, _ uuid.UUID, ids []uuid.UUID, _ uuid.UUID) error {
					captured = append([]uuid.UUID(nil), ids...)
					return nil
				},
			}
			roleRepo := &mockRoleRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*Role, error) {
					return &Role{ID: roleID}, nil
				},
			}
			jwt, _ := NewJWTService("test-secret-minimum-32-characters!!")
			svc := NewAdminService(&mockUserRepo{}, roleRepo, menuRepo, &mockAuditRepo{}, jwt, zap.NewNop())

			err := svc.SetRoleMenus(context.Background(), roleID, tc.input, uuid.New())
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.wantIn, captured)
		})
	}
}

func newCopyRoleTestService(t *testing.T, roleRepo *mockRoleRepo, menuRepo *mockMenuRepo) *AdminService {
	t.Helper()
	jwt, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	return NewAdminService(
		&mockUserRepo{},
		roleRepo,
		menuRepo,
		&mockAuditRepo{},
		jwt,
		zap.NewNop(),
	)
}

func copyRoleFixtureRepository(sourceID, copiedID uuid.UUID) *mockRoleRepo {
	return &mockRoleRepo{
		createFn: func(_ context.Context, role *Role) error {
			role.ID = copiedID
			return nil
		},
		getByIDFn: func(_ context.Context, id uuid.UUID) (*Role, error) {
			switch id {
			case sourceID:
				return &Role{ID: sourceID, Name: "operator", Description: "source"}, nil
			case copiedID:
				return &Role{ID: copiedID, Name: "operator_copy", Description: "source"}, nil
			default:
				return nil, commonerrors.ErrNotFound
			}
		},
	}
}

func TestAdminService_CopyRole_CopiesExactAPIPermissions(t *testing.T) {
	sourceID := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	copiedID := uuid.MustParse("30000000-0000-0000-0000-000000000002")
	endpointIDs := []uuid.UUID{
		uuid.MustParse("40000000-0000-0000-0000-000000000001"),
		uuid.MustParse("40000000-0000-0000-0000-000000000002"),
	}
	roleRepo := copyRoleFixtureRepository(sourceID, copiedID)
	roleRepo.getRoleAPIEndpointIDsFn = func(_ context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
		require.Equal(t, sourceID, roleID)
		return endpointIDs, nil
	}
	var savedRoleID uuid.UUID
	var savedEndpointIDs []uuid.UUID
	roleRepo.setRoleAPIEndpointsFn = func(_ context.Context, roleID uuid.UUID, ids []uuid.UUID) error {
		savedRoleID = roleID
		savedEndpointIDs = append([]uuid.UUID(nil), ids...)
		return nil
	}

	svc := newCopyRoleTestService(t, roleRepo, &mockMenuRepo{})
	_, err := svc.CopyRole(context.Background(), sourceID)

	require.NoError(t, err)
	assert.Equal(t, copiedID, savedRoleID)
	assert.Equal(t, endpointIDs, savedEndpointIDs)
}

func TestAdminService_CopyRole_PersistsEmptyAPIPermissions(t *testing.T) {
	sourceID := uuid.MustParse("30000000-0000-0000-0000-000000000011")
	copiedID := uuid.MustParse("30000000-0000-0000-0000-000000000012")
	roleRepo := copyRoleFixtureRepository(sourceID, copiedID)
	setCalled := false
	roleRepo.setRoleAPIEndpointsFn = func(_ context.Context, roleID uuid.UUID, ids []uuid.UUID) error {
		setCalled = true
		assert.Equal(t, copiedID, roleID)
		assert.Empty(t, ids)
		return nil
	}

	svc := newCopyRoleTestService(t, roleRepo, &mockMenuRepo{})
	_, err := svc.CopyRole(context.Background(), sourceID)

	require.NoError(t, err)
	assert.True(t, setCalled)
}

func TestAdminService_CopyRole_APIPermissionFailureDeletesCopy(t *testing.T) {
	sourceID := uuid.MustParse("30000000-0000-0000-0000-000000000021")
	copiedID := uuid.MustParse("30000000-0000-0000-0000-000000000022")
	roleRepo := copyRoleFixtureRepository(sourceID, copiedID)
	roleRepo.getRoleAPIEndpointIDsFn = func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
		return nil, fmt.Errorf("read endpoint grants")
	}
	var deletedID uuid.UUID
	roleRepo.deleteFn = func(_ context.Context, id uuid.UUID) error {
		deletedID = id
		return nil
	}

	svc := newCopyRoleTestService(t, roleRepo, &mockMenuRepo{})
	_, err := svc.CopyRole(context.Background(), sourceID)

	require.Error(t, err)
	assert.ErrorContains(t, err, "copy role API permissions")
	assert.ErrorContains(t, err, "read endpoint grants")
	assert.Equal(t, copiedID, deletedID)
}

func TestAdminService_CopyRole_CleanupFailurePreservesBothErrors(t *testing.T) {
	sourceID := uuid.MustParse("30000000-0000-0000-0000-000000000031")
	copiedID := uuid.MustParse("30000000-0000-0000-0000-000000000032")
	roleRepo := copyRoleFixtureRepository(sourceID, copiedID)
	roleRepo.getRoleAPIEndpointIDsFn = func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
		return nil, fmt.Errorf("read endpoint grants")
	}
	roleRepo.deleteFn = func(_ context.Context, _ uuid.UUID) error {
		return fmt.Errorf("delete copied role")
	}

	svc := newCopyRoleTestService(t, roleRepo, &mockMenuRepo{})
	_, err := svc.CopyRole(context.Background(), sourceID)

	require.Error(t, err)
	assert.ErrorContains(t, err, "read endpoint grants")
	assert.ErrorContains(t, err, "cleanup copied role")
	assert.ErrorContains(t, err, "delete copied role")
}

func TestAdminService_CopyRole_GroupFailureStopsBeforeAPIPersistence(t *testing.T) {
	sourceID := uuid.MustParse("30000000-0000-0000-0000-000000000041")
	copiedID := uuid.MustParse("30000000-0000-0000-0000-000000000042")
	roleRepo := copyRoleFixtureRepository(sourceID, copiedID)
	roleRepo.getDeviceGroupDataFn = func(_ context.Context, _ uuid.UUID) (*RoleDeviceGroupData, error) {
		return nil, fmt.Errorf("read device group grants")
	}
	apiWriteCalled := false
	roleRepo.setRoleAPIEndpointsFn = func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
		apiWriteCalled = true
		return nil
	}
	var deletedID uuid.UUID
	roleRepo.deleteFn = func(_ context.Context, id uuid.UUID) error {
		deletedID = id
		return nil
	}

	svc := newCopyRoleTestService(t, roleRepo, &mockMenuRepo{})
	_, err := svc.CopyRole(context.Background(), sourceID)

	require.Error(t, err)
	assert.ErrorContains(t, err, "copy role device groups")
	assert.False(t, apiWriteCalled)
	assert.Equal(t, copiedID, deletedID)
}

func TestAdminService_CopyRole_MenuFailureStopsBeforeAPIPersistence(t *testing.T) {
	sourceID := uuid.MustParse("30000000-0000-0000-0000-000000000051")
	copiedID := uuid.MustParse("30000000-0000-0000-0000-000000000052")
	roleRepo := copyRoleFixtureRepository(sourceID, copiedID)
	apiWriteCalled := false
	roleRepo.setRoleAPIEndpointsFn = func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
		apiWriteCalled = true
		return nil
	}
	var deletedID uuid.UUID
	roleRepo.deleteFn = func(_ context.Context, id uuid.UUID) error {
		deletedID = id
		return nil
	}
	menuRepo := &mockMenuRepo{
		getRoleMenuIDsFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return nil, fmt.Errorf("read menu grants")
		},
	}

	svc := newCopyRoleTestService(t, roleRepo, menuRepo)
	_, err := svc.CopyRole(context.Background(), sourceID)

	require.Error(t, err)
	assert.ErrorContains(t, err, "copy role menus")
	assert.False(t, apiWriteCalled)
	assert.Equal(t, copiedID, deletedID)
}

func TestAdminService_CopyRole_FinalReadFailureDeletesCopy(t *testing.T) {
	sourceID := uuid.MustParse("30000000-0000-0000-0000-000000000061")
	copiedID := uuid.MustParse("30000000-0000-0000-0000-000000000062")
	roleRepo := copyRoleFixtureRepository(sourceID, copiedID)
	roleRepo.getByIDFn = func(_ context.Context, id uuid.UUID) (*Role, error) {
		if id == sourceID {
			return &Role{ID: sourceID, Name: "operator", Description: "source"}, nil
		}
		return nil, fmt.Errorf("read copied role")
	}
	var deletedID uuid.UUID
	roleRepo.deleteFn = func(_ context.Context, id uuid.UUID) error {
		deletedID = id
		return nil
	}

	svc := newCopyRoleTestService(t, roleRepo, &mockMenuRepo{})
	_, err := svc.CopyRole(context.Background(), sourceID)

	require.Error(t, err)
	assert.ErrorContains(t, err, "get copied role")
	assert.ErrorContains(t, err, "read copied role")
	assert.Equal(t, copiedID, deletedID)
}

func TestAdminService_CopyRole_CanceledRequestStillDeletesCopy(t *testing.T) {
	sourceID := uuid.MustParse("30000000-0000-0000-0000-000000000071")
	copiedID := uuid.MustParse("30000000-0000-0000-0000-000000000072")
	roleRepo := copyRoleFixtureRepository(sourceID, copiedID)
	ctx, cancel := context.WithCancel(context.Background())
	roleRepo.getRoleAPIEndpointIDsFn = func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
		cancel()
		return nil, context.Canceled
	}
	deleteCalled := false
	roleRepo.deleteFn = func(cleanupCtx context.Context, id uuid.UUID) error {
		deleteCalled = true
		assert.Equal(t, copiedID, id)
		assert.NoError(t, cleanupCtx.Err(), "补偿清理不能继承已取消的请求 context")
		_, hasDeadline := cleanupCtx.Deadline()
		assert.True(t, hasDeadline, "补偿清理必须有独立的超时上限")
		return cleanupCtx.Err()
	}

	svc := newCopyRoleTestService(t, roleRepo, &mockMenuRepo{})
	_, err := svc.CopyRole(ctx, sourceID)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.True(t, deleteCalled)
	assert.NotContains(t, err.Error(), "cleanup copied role")
}

// ===================== P0-④ 单点登录 — 登录时撤销旧 token =====================
//
// service.Login 当 policy.AllowConcurrent=false 时调用 TokenRevoker.Revoke 把
// 该用户所有旧 token 标记失效。验证：
//   1. 默认 AllowConcurrent=true → 不调用 Revoke
//   2. AllowConcurrent=false → 调用 Revoke 且 Redis 写入了撤销时间戳
//   3. Revoker / Policy 任一未注入 → 不阻塞登录（fail-safe）
//   4. Revoker 抛错 → 登录仍成功（fail-safe，下次会重试）

func newSingleSessionTestService(
	t *testing.T,
	userRepo *mockUserRepo,
	roleRepo *mockRoleRepo,
	allowConcurrent string, // "true"/"false"/""=不设
) (*AdminService, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	jwt, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	svc := NewAdminService(userRepo, roleRepo, &mockMenuRepo{}, &mockAuditRepo{}, jwt, zap.NewNop())
	svc.SetTokenRevoker(NewTokenRevoker(client, 24*time.Hour))

	if allowConcurrent != "" {
		policy := NewSecurityPolicy(&policyMockQuerier{entries: map[string]string{
			"security.isOnlyOneUserLoginEnable": allowConcurrent,
		}})
		svc.SetSecurityPolicy(policy)
	}
	return svc, mr
}

func TestAdminService_Login_SingleSession_DefaultAllowConcurrent_NoRevoke(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{
		getByUsernameFn: func(_ context.Context, u string) (*User, error) {
			return &User{ID: userID, Username: u, PasswordHash: hashPassword("p"), Status: UserStatusActive}, nil
		},
	}
	roleRepo := &mockRoleRepo{getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) {
		return []Role{{Name: "admin"}}, nil
	}}

	svc, mr := newSingleSessionTestService(t, userRepo, roleRepo, "true") // 允许多端
	_, err := svc.Login(context.Background(), "admin", "p")
	require.NoError(t, err)

	// 默认允许多端 → 不应写撤销 key
	assert.False(t, mr.Exists("auth:revoked_at:user:"+userID.String()),
		"AllowConcurrent=true 时不应触发 revoke")
}

func TestAdminService_Login_SingleSession_RevokesOldTokens(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{
		getByUsernameFn: func(_ context.Context, u string) (*User, error) {
			return &User{ID: userID, Username: u, PasswordHash: hashPassword("p"), Status: UserStatusActive}, nil
		},
	}
	roleRepo := &mockRoleRepo{getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) {
		return []Role{{Name: "admin"}}, nil
	}}

	svc, mr := newSingleSessionTestService(t, userRepo, roleRepo, "false") // 单点登录
	tp, err := svc.Login(context.Background(), "admin", "p")
	require.NoError(t, err)
	assert.NotEmpty(t, tp.AccessToken, "登录仍成功")

	// 单点 → 应写撤销 key
	assert.True(t, mr.Exists("auth:revoked_at:user:"+userID.String()),
		"AllowConcurrent=false 时必须 revoke 旧 token")
}

func TestAdminService_Login_SingleSession_NoPolicy_NoRevoke(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{
		getByUsernameFn: func(_ context.Context, u string) (*User, error) {
			return &User{ID: userID, Username: u, PasswordHash: hashPassword("p"), Status: UserStatusActive}, nil
		},
	}
	roleRepo := &mockRoleRepo{getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) {
		return []Role{{Name: "admin"}}, nil
	}}

	// 不注入 policy（""）
	svc, mr := newSingleSessionTestService(t, userRepo, roleRepo, "")
	_, err := svc.Login(context.Background(), "admin", "p")
	require.NoError(t, err, "policy 未注入应 fail-safe 走默认（不 revoke）")
	assert.False(t, mr.Exists("auth:revoked_at:user:"+userID.String()))
}

func TestAdminService_Login_SingleSession_RevokerError_DoesNotFailLogin(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{
		getByUsernameFn: func(_ context.Context, u string) (*User, error) {
			return &User{ID: userID, Username: u, PasswordHash: hashPassword("p"), Status: UserStatusActive}, nil
		},
	}
	roleRepo := &mockRoleRepo{getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) {
		return []Role{{Name: "admin"}}, nil
	}}

	svc, mr := newSingleSessionTestService(t, userRepo, roleRepo, "false")
	mr.Close() // 模拟 redis 故障

	tp, err := svc.Login(context.Background(), "admin", "p")
	require.NoError(t, err, "revoker 抛错时登录应仍成功（fail-safe — 单点登录失败比锁死用户风险低）")
	assert.NotEmpty(t, tp.AccessToken)
}

// ===================== issue #649 「默认密码」接通 — service 层 =====================
//
// 覆盖 CreateUser / ResetPassword 的全部行为矩阵 + 内置用户拦截 + LDAP 拦截 +
// 默认密码消费跳过强度校验 + revoker.Revoke 联动 + audit 写入。
//
// 设计权威：~/Documents/notes/tmp/默认密码配置接通-修改方案.md §1 + §3.2

// newServiceWithPolicyAndRevoker 注入 policy + revoker（用 miniredis）的测试 helper。
// auditRepo 也回传给 caller 用于断言审计日志写入。
func newServiceWithPolicyAndRevoker(
	t *testing.T,
	userRepo *mockUserRepo,
	roleRepo *mockRoleRepo,
	policyEntries map[string]string,
) (*AdminService, *mockAuditRepo, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	jwt, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	auditRepo := &mockAuditRepo{}
	svc := NewAdminService(userRepo, roleRepo, &mockMenuRepo{}, auditRepo, jwt, zap.NewNop())
	if policyEntries != nil {
		svc.SetSecurityPolicy(NewSecurityPolicy(&policyMockQuerier{entries: policyEntries}))
	}
	svc.SetTokenRevoker(NewTokenRevoker(client, time.Hour))
	return svc, auditRepo, mr
}

// auditCollector 把 mockAuditRepo.createFn 装成"收集所有写入"的轻量记录器。
func auditCollector() (*[]*AuditLog, func(ctx context.Context, log *AuditLog) error) {
	var logs []*AuditLog
	return &logs, func(_ context.Context, log *AuditLog) error {
		logs = append(logs, log)
		return nil
	}
}

// --- CreateUser ---

func TestAdminService_CreateUser_UseDefaultPassword_Succeeds(t *testing.T) {
	var created *User
	userRepo := &mockUserRepo{createFn: func(_ context.Context, u *User) error {
		u.ID = uuid.New()
		u.CreatedAt = time.Now()
		u.UpdatedAt = time.Now()
		created = u
		return nil
	}}
	logs, fn := auditCollector()
	svc, auditRepo, _ := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.defaultPasswd":   "OMC@123456",
		"security.passwordContent": "true",
		"security.pwdMinLength":    "8",
		"security.pwdMaxLength":    "32",
	})
	auditRepo.createFn = fn

	user, err := svc.CreateUser(context.Background(), CreateUserRequest{
		Username:           "newuser",
		UseDefaultPassword: true,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte("OMC@123456")),
		"应落库 sys_configs 配的默认密码 hash")
	assert.True(t, created.MustChangePassword, "硬规则：管理员创建即强制首次改密")
	require.NotNil(t, user)
	require.Len(t, *logs, 1)
	assert.Equal(t, "user_create", (*logs)[0].Action)
	assert.Equal(t, true, (*logs)[0].Details["used_default_password"])
}

func TestAdminService_CreateUser_UseDefaultPassword_PolicyEmpty_Rejects(t *testing.T) {
	userRepo := &mockUserRepo{createFn: func(_ context.Context, _ *User) error {
		t.Fatal("DefaultPassword 为空时 service 不应创建用户")
		return nil
	}}
	logs, fn := auditCollector()
	svc, auditRepo, _ := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.defaultPasswd": "", // 显式清空 → 重置走 fallback
	})
	auditRepo.createFn = fn

	_, err := svc.CreateUser(context.Background(), CreateUserRequest{
		Username:           "newuser",
		UseDefaultPassword: true,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	require.Len(t, *logs, 1)
	assert.Equal(t, "user_create_failed", (*logs)[0].Action, "失败路径 sink 自动追加 _failed 后缀")
}

func TestAdminService_CreateUser_ManualPassword_StrengthEnforced(t *testing.T) {
	userRepo := &mockUserRepo{createFn: func(_ context.Context, _ *User) error {
		t.Fatal("强度不通过的密码不应落库")
		return nil
	}}
	svc, _, _ := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.passwordContent": "true",
		"security.pwdMinLength":    "8",
	})
	_, err := svc.CreateUser(context.Background(), CreateUserRequest{
		Username: "newuser",
		Password: "abc", // 太短
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooShort)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestAdminService_CreateUser_ManualPassword_HardRuleMustChange(t *testing.T) {
	var created *User
	userRepo := &mockUserRepo{createFn: func(_ context.Context, u *User) error {
		u.ID = uuid.New()
		created = u
		return nil
	}}
	svc, _, _ := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.passwordContent": "true",
		"security.pwdMinLength":    "8",
	})
	_, err := svc.CreateUser(context.Background(), CreateUserRequest{
		Username: "newuser",
		Password: "Abc@12345678", // 满足强度
	})
	require.NoError(t, err)
	assert.True(t, created.MustChangePassword,
		"硬规则：手填强密码也必须强制首次改密")
}

func TestAdminService_CreateUser_EmptyPassword_NotUsingDefault_Rejects(t *testing.T) {
	svc, _, _ := newServiceWithPolicyAndRevoker(t, &mockUserRepo{}, &mockRoleRepo{}, nil)
	_, err := svc.CreateUser(context.Background(), CreateUserRequest{
		Username:           "newuser",
		Password:           "",
		UseDefaultPassword: false,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestAdminService_CreateUser_UseDefaultPassword_SkipsStrengthValidation(t *testing.T) {
	// 设计 §3.2 方案 A：消费默认密码时跳过 ValidatePassword（受信任运维设定值；
	// 避免管理员调高强度规则后存量默认密码触发 UX 死结）。
	var created *User
	userRepo := &mockUserRepo{createFn: func(_ context.Context, u *User) error {
		u.ID = uuid.New()
		created = u
		return nil
	}}
	svc, _, _ := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.defaultPasswd":   "OMC@1", // 5 位，故意违反 pwdMinLength=8
		"security.passwordContent": "true",
		"security.pwdMinLength":    "8",
	})
	_, err := svc.CreateUser(context.Background(), CreateUserRequest{
		Username:           "newuser",
		UseDefaultPassword: true,
	})
	require.NoError(t, err, "消费默认密码应跳过强度校验")
	require.NotNil(t, created)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte("OMC@1")))
}

// --- ChangePassword ---

func TestAdminService_ChangePassword_RevokesExistingTokens(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			assert.Equal(t, userID, id)
			return &User{
				ID:           userID,
				Username:     "admin",
				PasswordHash: hashPassword("Old@123456"),
			}, nil
		},
		updatePasswordFn: func(_ context.Context, id uuid.UUID, passwordHash string) error {
			assert.Equal(t, userID, id)
			assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("New@123456")))
			return nil
		},
	}
	svc, _, mr := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.passwordContent": "true",
		"security.pwdMinLength":    "8",
	})

	err := svc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
		OldPassword: "Old@123456",
		NewPassword: "New@123456",
	})
	require.NoError(t, err)
	assert.True(t, mr.Exists("auth:revoked_at:user:"+userID.String()),
		"自助改密成功后必须撤销该用户已有 token")
}

func TestAdminService_ChangePassword_RevokeFailureDoesNotChangePassword(t *testing.T) {
	userID := uuid.New()
	updateCalled := false
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: userID, Username: "admin", PasswordHash: hashPassword("Old@123456")}, nil
		},
		updatePasswordFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			updateCalled = true
			return nil
		},
	}
	svc, _, mr := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, nil)
	mr.Close()

	err := svc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
		OldPassword: "Old@123456",
		NewPassword: "New@123456",
	})
	require.Error(t, err)
	assert.False(t, updateCalled, "撤销基础设施不可用时不得先修改密码再返回失败")
}

func TestAdminService_ChangePassword_AuditsSuccessWithoutPasswords(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: userID, Username: "admin", PasswordHash: hashPassword("Old@123456")}, nil
		},
	}
	logs, fn := auditCollector()
	svc, auditRepo, _ := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, nil)
	auditRepo.createFn = fn
	ctx := context.WithValue(context.Background(), CtxKeyUserID, userID)
	ctx = context.WithValue(ctx, CtxKeyUsername, "admin")
	ctx = context.WithValue(ctx, auditRequestMetadataContextKey{}, auditRequestMetadata{
		IPAddress: "10.10.30.155",
		UserAgent: "change-password-test",
	})

	err := svc.ChangePassword(ctx, userID, ChangePasswordRequest{
		OldPassword: "Old@123456",
		NewPassword: "New@123456",
	})
	require.NoError(t, err)
	require.Len(t, *logs, 1)
	log := (*logs)[0]
	assert.Equal(t, "password_change", log.Action)
	assert.Equal(t, "admin", log.Username)
	assert.Equal(t, "10.10.30.155", log.IPAddress)
	assert.Equal(t, "change-password-test", log.UserAgent)
	assert.Equal(t, userID.String(), log.ResourceID)
	assert.Equal(t, true, log.Details["success"])
	assert.Equal(t, userID.String(), log.Details["target_user_id"])
	serialized := fmt.Sprintf("%+v", log)
	assert.NotContains(t, serialized, "Old@123456")
	assert.NotContains(t, serialized, "New@123456")
}

func TestAdminService_ChangePassword_AuditsFailureWithoutPasswords(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: userID, Username: "admin", PasswordHash: hashPassword("Correct@123456")}, nil
		},
	}
	logs, fn := auditCollector()
	svc, auditRepo, _ := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, nil)
	auditRepo.createFn = fn

	err := svc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
		OldPassword: "Wrong@123456",
		NewPassword: "New@123456",
	})
	require.Error(t, err)
	require.Len(t, *logs, 1)
	log := (*logs)[0]
	assert.Equal(t, "password_change_failed", log.Action)
	assert.Equal(t, false, log.Details["success"])
	assert.Equal(t, userID.String(), log.Details["target_user_id"])
	serialized := fmt.Sprintf("%+v", log)
	assert.NotContains(t, serialized, "Wrong@123456")
	assert.NotContains(t, serialized, "New@123456")
	assert.NotContains(t, serialized, "Correct@123456")
}

// --- ResetPassword ---

func TestAdminService_ResetPassword_LDAP_Rejects(t *testing.T) {
	target := uuid.New()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "ldap_alice", Source: UserSourceLDAP}, nil
		},
		updatePasswordFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			t.Fatal("LDAP 拦截后不应调用 UpdatePassword")
			return nil
		},
	}
	svc, _, mr := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, nil)

	err := svc.ResetPassword(context.Background(), target, ResetPasswordRequest{NewPassword: "Abc@12345678"})
	assert.ErrorIs(t, err, ErrLDAPPasswordExternal)
	assert.False(t, mr.Exists("auth:revoked_at:user:"+target.String()),
		"LDAP 拦截后 revoker.Revoke 不应被调用")
}

func TestAdminService_ResetPassword_BuiltInUser_Rejects(t *testing.T) {
	// issue #649：新增内置用户拦截，与 LockUser/DeleteUser/ForceLogout 对齐。
	target := uuid.New()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "admin", Source: UserSourceBuiltIn}, nil
		},
		updatePasswordFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			t.Fatal("内置用户拦截后不应调用 UpdatePassword")
			return nil
		},
	}
	svc, _, mr := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, nil)

	err := svc.ResetPassword(context.Background(), target, ResetPasswordRequest{NewPassword: "Abc@12345678"})
	assert.ErrorIs(t, err, ErrBuiltInUserProtected)
	assert.False(t, mr.Exists("auth:revoked_at:user:"+target.String()),
		"内置用户拦截后 revoker.Revoke 不应被调用")
}

func TestAdminService_ResetPassword_UseDefaultPassword_Succeeds(t *testing.T) {
	target := uuid.New()
	var updatedHash string
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "alice", Source: UserSourceAdmin}, nil
		},
		updatePasswordFn: func(_ context.Context, _ uuid.UUID, h string) error {
			updatedHash = h
			return nil
		},
	}
	logs, fn := auditCollector()
	svc, auditRepo, mr := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.defaultPasswd": "OMC@123456",
	})
	auditRepo.createFn = fn

	err := svc.ResetPassword(context.Background(), target, ResetPasswordRequest{UseDefaultPassword: true})
	require.NoError(t, err)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(updatedHash), []byte("OMC@123456")))
	assert.True(t, mr.Exists("auth:revoked_at:user:"+target.String()),
		"OWASP A07：重置成功后 revoker.Revoke 应被调用，旧 token 立即失效")
	require.Len(t, *logs, 1)
	assert.Equal(t, "password_reset", (*logs)[0].Action)
	assert.Equal(t, true, (*logs)[0].Details["used_default_password"])
}

func TestAdminService_ResetPassword_UseDefaultPassword_PolicyEmpty_Rejects(t *testing.T) {
	target := uuid.New()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "alice", Source: UserSourceAdmin}, nil
		},
		updatePasswordFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			t.Fatal("默认密码为空时不应调用 UpdatePassword")
			return nil
		},
	}
	svc, _, mr := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.defaultPasswd": "", // 显式清空
	})

	err := svc.ResetPassword(context.Background(), target, ResetPasswordRequest{UseDefaultPassword: true})
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, mr.Exists("auth:revoked_at:user:"+target.String()),
		"默认密码为空被拒后 revoker.Revoke 不应被调用")
}

func TestAdminService_ResetPassword_ManualPassword_StrengthEnforcedAndRevoked(t *testing.T) {
	target := uuid.New()
	called := false
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "alice", Source: UserSourceAdmin}, nil
		},
		updatePasswordFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			called = true
			return nil
		},
	}
	svc, _, mr := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.passwordContent": "true",
		"security.pwdMinLength":    "8",
	})

	err := svc.ResetPassword(context.Background(), target, ResetPasswordRequest{NewPassword: "Abc@12345678"})
	require.NoError(t, err)
	assert.True(t, called)
	assert.True(t, mr.Exists("auth:revoked_at:user:"+target.String()),
		"手填强密码重置成功后 revoker.Revoke 也应被调用")
}

func TestAdminService_ResetPassword_ManualPassword_WeakRejected(t *testing.T) {
	target := uuid.New()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "alice", Source: UserSourceAdmin}, nil
		},
		updatePasswordFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			t.Fatal("弱密码不应落库")
			return nil
		},
	}
	svc, _, mr := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, map[string]string{
		"security.passwordContent": "true",
		"security.pwdMinLength":    "8",
	})

	err := svc.ResetPassword(context.Background(), target, ResetPasswordRequest{NewPassword: "abc"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooShort)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, mr.Exists("auth:revoked_at:user:"+target.String()))
}

func TestAdminService_ResetPassword_ManualPassword_Empty_Rejects(t *testing.T) {
	target := uuid.New()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "alice", Source: UserSourceAdmin}, nil
		},
	}
	svc, _, _ := newServiceWithPolicyAndRevoker(t, userRepo, &mockRoleRepo{}, nil)
	err := svc.ResetPassword(context.Background(), target, ResetPasswordRequest{
		NewPassword:        "",
		UseDefaultPassword: false,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestAdminService_ResetPassword_RevokerNotInjected_StillSucceeds(t *testing.T) {
	// revoker nil → warn 不阻断业务（与 LockUser 既有行为对齐，便于单元测试无需 Redis 也能跑）。
	target := uuid.New()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "alice", Source: UserSourceAdmin}, nil
		},
	}
	// 不走 newServiceWithPolicyAndRevoker —— 直接 newTestService（不注入 revoker）。
	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})
	svc.SetSecurityPolicy(NewSecurityPolicy(&policyMockQuerier{entries: map[string]string{
		"security.defaultPasswd": "OMC@123456",
	}}))

	err := svc.ResetPassword(context.Background(), target, ResetPasswordRequest{UseDefaultPassword: true})
	require.NoError(t, err, "revoker 未注入时仍应成功（fail-safe）")
}

func TestAdminService_GetUserMenuTreeByRole_CustomRoleIncludesButtonsUnderGrantedMenu(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	directoryID := uuid.New()
	menuID := uuid.New()
	executeButtonID := uuid.New()

	assignedMenus := []Menu{
		{ID: directoryID, Type: MenuTypeDirectory, PermissionKey: "mml"},
		{ID: menuID, Type: MenuTypeMenu, PermissionKey: "mml:console", ParentID: &directoryID},
	}
	allMenus := append(append([]Menu{}, assignedMenus...), Menu{
		ID:            executeButtonID,
		Type:          MenuTypeButton,
		PermissionKey: "mml:console:execute",
		ParentID:      &menuID,
		Status:        MenuStatusNormal,
		ShowStatus:    MenuShow,
	})

	menuRepo := &mockMenuRepo{
		getByRoleFn: func(_ context.Context, gotRoleID uuid.UUID) ([]Menu, error) {
			require.Equal(t, roleID, gotRoleID)
			return assignedMenus, nil
		},
		getAllActiveFn: func(context.Context) ([]Menu, error) {
			return allMenus, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, gotUserID uuid.UUID) (*User, error) {
			require.Equal(t, userID, gotUserID)
			return &User{ID: userID, Source: UserSourceAdmin}, nil
		},
	}
	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})
	svc.menuRepo = menuRepo

	tree, err := svc.GetUserMenuTreeByRole(context.Background(), userID, roleID)
	require.NoError(t, err)
	require.Len(t, tree, 1)
	require.Len(t, tree[0].Children, 1)
	require.Len(t, tree[0].Children[0].Children, 1)
	assert.Equal(t, executeButtonID, tree[0].Children[0].Children[0].ID)
}

func TestAdminService_GetUserMenuTreeByRole_BuiltInRoleKeepsExplicitButtons(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.MustParse(builtinViewerRoleID)
	directoryID := uuid.New()
	menuID := uuid.New()
	exportButtonID := uuid.New()

	menuRepo := &mockMenuRepo{
		getByRoleFn: func(_ context.Context, gotRoleID uuid.UUID) ([]Menu, error) {
			require.Equal(t, roleID, gotRoleID)
			return []Menu{
				{ID: directoryID, Type: MenuTypeDirectory, PermissionKey: "mml"},
				{ID: menuID, Type: MenuTypeMenu, PermissionKey: "mml:console", ParentID: &directoryID},
				{ID: exportButtonID, Type: MenuTypeButton, PermissionKey: "mml:console:export", ParentID: &menuID},
			}, nil
		},
		getAllActiveFn: func(context.Context) ([]Menu, error) {
			t.Fatal("built-in roles must not derive additional button permissions")
			return nil, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(context.Context, uuid.UUID) (*User, error) {
			return &User{ID: userID, Source: UserSourceAdmin}, nil
		},
	}
	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})
	svc.menuRepo = menuRepo

	tree, err := svc.GetUserMenuTreeByRole(context.Background(), userID, roleID)
	require.NoError(t, err)
	require.Len(t, tree, 1)
	require.Len(t, tree[0].Children, 1)
	require.Len(t, tree[0].Children[0].Children, 1)
	assert.Equal(t, exportButtonID, tree[0].Children[0].Children[0].ID)
}

// --- sys_config validator ---

func TestRegisterSecurityValidators_DefaultPasswd(t *testing.T) {
	// 设计 §3.1：写入侧强校验，与 service 消费侧"跳过强度校验"形成互补。
	policy := NewSecurityPolicy(&policyMockQuerier{entries: map[string]string{
		"security.passwordContent": "true",
		"security.pwdMinLength":    "8",
	}})
	svc := NewSysConfigService(&stubSysConfigRepo{})
	RegisterSecurityValidators(svc, policy)

	t.Run("空字符串放行（允许清空）", func(t *testing.T) {
		_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
			Category: "security",
			Items:    []BatchItem{{Key: "defaultPasswd", Value: ""}},
		})
		assert.NoError(t, err)
	})

	t.Run("非空但不满足强度被拒", func(t *testing.T) {
		_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
			Category: "security",
			Items:    []BatchItem{{Key: "defaultPasswd", Value: "abc"}},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	})

	t.Run("非空且满足强度放行", func(t *testing.T) {
		_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
			Category: "security",
			Items:    []BatchItem{{Key: "defaultPasswd", Value: "Abc@12345678"}},
		})
		assert.NoError(t, err)
	})
}

// stubFeatureGate implements LicenseFeatureGate for menu filter tests.
type stubFeatureGate struct {
	authorized map[string]bool
	err        error // FilterAuthorized error
	active     bool
	activeErr  error // HasActiveLicense error
}

func (s stubFeatureGate) FilterAuthorized(_ context.Context, codes []string) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	var out []string
	for _, c := range codes {
		if s.authorized[c] {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s stubFeatureGate) HasActiveLicense(_ context.Context) (bool, error) {
	if s.activeErr != nil {
		return false, s.activeErr
	}
	return s.active, nil
}

func TestApplyLicenseFeatureGate(t *testing.T) {
	svc := NewAdminService(nil, nil, nil, nil, nil, zap.NewNop())
	svc.featureGate = stubFeatureGate{authorized: map[string]bool{"CODE_KEEP": true}}

	deviceDir := uuid.New()
	keepMenu := uuid.New()
	dropMenu := uuid.New()

	menus := []Menu{
		{ID: deviceDir, Type: MenuTypeDirectory, PermissionKey: "device"},
		{ID: keepMenu, Type: MenuTypeMenu, PermissionKey: "device:list", ParentID: &deviceDir, FeatureCodes: []string{"CODE_KEEP"}},
		{ID: dropMenu, Type: MenuTypeMenu, PermissionKey: "alarm:current", ParentID: &deviceDir, FeatureCodes: []string{"CODE_DROP"}},
		{ID: uuid.New(), Type: MenuTypeButton, PermissionKey: "alarm:current:query", ParentID: &dropMenu},
	}

	got := svc.applyLicenseFeatureGate(context.Background(), menus)
	permKeys := make(map[string]bool)
	for _, m := range got {
		permKeys[m.PermissionKey] = true
	}
	assert.True(t, permKeys["device"], "directory (no feature_code) kept")
	assert.True(t, permKeys["device:list"], "authorized menu kept (CODE_KEEP)")
	assert.False(t, permKeys["alarm:current"], "unauthorized menu dropped (CODE_DROP)")
	assert.False(t, permKeys["alarm:current:query"], "orphan button under dropped menu removed")
}

func TestApplyLicenseFeatureGate_FailOpen(t *testing.T) {
	svc := NewAdminService(nil, nil, nil, nil, nil, zap.NewNop())
	svc.featureGate = stubFeatureGate{err: fmt.Errorf("db down")}
	menus := []Menu{{ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "x", FeatureCodes: []string{"CODE_X"}}}
	got := svc.applyLicenseFeatureGate(context.Background(), menus)
	assert.Len(t, got, 1, "gate error → fail-open, menu not hidden")
}

func TestFilterLicenseOnlyMenus(t *testing.T) {
	menus := []Menu{
		{ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "system_license", RoutePath: "/license"},
		{ID: uuid.New(), Type: MenuTypeDirectory, PermissionKey: "device", RoutePath: ""},
		{ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "device:list", RoutePath: "/device/list"},
		{ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "system:user", RoutePath: "/system/users"},
	}
	got := filterLicenseOnlyMenus(menus)
	assert.Len(t, got, 1, "no license → only /license menu kept")
	assert.Equal(t, "/license", got[0].RoutePath)
}

// TestEnsureLicenseMenuVisible_PresentWhenAlreadyInSet 验证 menus 已含 /license 时直接返回。
func TestEnsureLicenseMenuVisible_PresentWhenAlreadyInSet(t *testing.T) {
	svc := NewAdminService(nil, nil, nil, nil, nil, zap.NewNop())
	svc.menuRepo = &mockMenuRepo{}
	lic := Menu{ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "system_license", RoutePath: "/license"}
	menus := []Menu{
		{ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "device:list", RoutePath: "/device/list"},
		lic,
	}
	got := svc.ensureLicenseMenuVisible(context.Background(), menus)
	assert.Len(t, got, len(menus), "no extra append when /license already present")
}

// TestEnsureLicenseMenuVisible_AppendedWhenMissing 验证 /license 缺失时从 GetTree 全集补回
// （模拟非 builtIn 用户 role_menus 未分配 /license 的场景）。
func TestEnsureLicenseMenuVisible_AppendedWhenMissing(t *testing.T) {
	licenseMenu := Menu{
		ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "system_license", RoutePath: "/license",
		Status: MenuStatusNormal, ShowStatus: MenuShow,
	}
	repo := &mockMenuRepo{
		getTreeFn: func(_ context.Context, _ *MenuStatus) ([]Menu, error) {
			return []Menu{
				licenseMenu,
				{ID: uuid.New(), Type: MenuTypeDirectory, PermissionKey: "device"},
				// hidden /license 副本不应被补回（show_status=hide）
				{ID: uuid.New(), Type: MenuTypeMenu, RoutePath: "/license/hidden", ShowStatus: MenuHide, Status: MenuStatusNormal},
			}, nil
		},
	}
	svc := NewAdminService(nil, nil, nil, nil, nil, zap.NewNop())
	svc.menuRepo = repo

	menus := []Menu{
		{ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "device:list", RoutePath: "/device/list"},
	}
	got := svc.ensureLicenseMenuVisible(context.Background(), menus)

	permKeys := make(map[string]bool)
	for _, m := range got {
		permKeys[m.PermissionKey] = true
	}
	assert.True(t, permKeys["system_license"], "/license must be appended when missing")
	assert.True(t, permKeys["device:list"], "existing menus preserved")
}

// TestApplyLicenseMenuGate_WithLicenseKeepsLicenseMenu 验证有 license 时（applyLicenseFeatureGate 后）
// /license 仍可见，覆盖 bug：source=admin 用户走 GetByRole，role_menus 未分配 /license 导致菜单丢失。
func TestApplyLicenseMenuGate_WithLicenseKeepsLicenseMenu(t *testing.T) {
	licenseMenu := Menu{
		ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "system_license", RoutePath: "/license",
		Status: MenuStatusNormal, ShowStatus: MenuShow,
	}
	repo := &mockMenuRepo{
		getTreeFn: func(_ context.Context, _ *MenuStatus) ([]Menu, error) {
			return []Menu{licenseMenu}, nil
		},
	}
	svc := NewAdminService(nil, nil, nil, nil, nil, zap.NewNop())
	svc.menuRepo = repo
	// 有 active license；授权 CODE_KEEP。
	svc.featureGate = stubFeatureGate{active: true, authorized: map[string]bool{"CODE_KEEP": true}}

	// 模拟 GetByRole 结果：不含 /license（role_menus 未分配）。
	menus := []Menu{
		{ID: uuid.New(), Type: MenuTypeMenu, PermissionKey: "device:list", FeatureCodes: []string{"CODE_KEEP"}},
	}
	got := svc.applyLicenseMenuGate(context.Background(), menus)

	hasLicense := false
	for _, m := range got {
		if m.PermissionKey == "system_license" {
			hasLicense = true
		}
	}
	assert.True(t, hasLicense, "/license must remain visible when license active even if role_menus lacks it")
}
