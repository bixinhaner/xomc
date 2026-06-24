package admin

import (
	"context"
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
	createFn               func(ctx context.Context, role *Role) error
	getByIDFn              func(ctx context.Context, id uuid.UUID) (*Role, error)
	getByNameFn            func(ctx context.Context, name string) (*Role, error)
	updateFn               func(ctx context.Context, role *Role) error
	deleteFn               func(ctx context.Context, id uuid.UUID) error
	listFn                 func(ctx context.Context) ([]Role, error)
	listWithPaginationFn   func(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error)
	assignRoleFn           func(ctx context.Context, userID, roleID uuid.UUID) error
	removeRoleFn           func(ctx context.Context, userID, roleID uuid.UUID) error
	getUserRolesFn         func(ctx context.Context, userID uuid.UUID) ([]Role, error)
	listUserIDsByRoleFn    func(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	getPermissionsFn       func(ctx context.Context, roleID uuid.UUID) ([]Permission, error)
	checkPermissionFn      func(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	listAllPermissionsFn   func(ctx context.Context) ([]Permission, error)
	addPermissionsFn       func(ctx context.Context, roleID uuid.UUID, perms []Permission) error
	removeAllPermissionsFn func(ctx context.Context, roleID uuid.UUID) error
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
	getAllActiveFn func(ctx context.Context) ([]Menu, error)
	setRoleMenusFn func(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error
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
func (m *mockMenuRepo) GetTree(_ context.Context, _ *MenuStatus) ([]Menu, error) {
	return nil, nil
}
func (m *mockMenuRepo) GetByRole(_ context.Context, _ uuid.UUID) ([]Menu, error) {
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
func (m *mockMenuRepo) GetRoleMenuIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
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
