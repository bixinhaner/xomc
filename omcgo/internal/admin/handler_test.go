package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// handlerMock* types — prefixed to avoid conflicts with service_test.go mocks
// ---------------------------------------------------------------------------

type handlerMockUserRepo struct {
	createFn          func(ctx context.Context, user *User) error
	getByIDFn         func(ctx context.Context, id uuid.UUID) (*User, error)
	getByUsernameFn   func(ctx context.Context, username string) (*User, error)
	updateFn          func(ctx context.Context, user *User) error
	updatePasswordFn  func(ctx context.Context, id uuid.UUID, passwordHash string) error
	deleteFn          func(ctx context.Context, id uuid.UUID) error
	listFn            func(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error)
	updateLastLoginFn func(ctx context.Context, id uuid.UUID) error
}

func (m *handlerMockUserRepo) Create(ctx context.Context, user *User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	user.ID = uuid.New()
	return nil
}
func (m *handlerMockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *handlerMockUserRepo) GetByUsername(ctx context.Context, username string) (*User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *handlerMockUserRepo) Update(ctx context.Context, user *User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}
func (m *handlerMockUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, id, hash)
	}
	return nil
}
func (m *handlerMockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *handlerMockUserRepo) List(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &model.ListResponse[User]{Items: []User{}}, nil
}
func (m *handlerMockUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	if m.updateLastLoginFn != nil {
		return m.updateLastLoginFn(ctx, id)
	}
	return nil
}

type handlerMockRoleRepo struct {
	listFn         func(ctx context.Context) ([]Role, error)
	getUserRolesFn func(ctx context.Context, userID uuid.UUID) ([]Role, error)
}

func (m *handlerMockRoleRepo) Create(_ context.Context, _ *Role) error             { return nil }
func (m *handlerMockRoleRepo) GetByID(_ context.Context, _ uuid.UUID) (*Role, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *handlerMockRoleRepo) GetByName(_ context.Context, _ string) (*Role, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *handlerMockRoleRepo) Update(_ context.Context, _ *Role) error             { return nil }
func (m *handlerMockRoleRepo) Delete(_ context.Context, _ uuid.UUID) error         { return nil }
func (m *handlerMockRoleRepo) List(ctx context.Context) ([]Role, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []Role{}, nil
}
func (m *handlerMockRoleRepo) AssignRole(_ context.Context, _, _ uuid.UUID) error  { return nil }
func (m *handlerMockRoleRepo) RemoveRole(_ context.Context, _, _ uuid.UUID) error  { return nil }
func (m *handlerMockRoleRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	if m.getUserRolesFn != nil {
		return m.getUserRolesFn(ctx, userID)
	}
	return []Role{}, nil
}
func (m *handlerMockRoleRepo) GetPermissions(_ context.Context, _ uuid.UUID) ([]Permission, error) {
	return []Permission{}, nil
}
func (m *handlerMockRoleRepo) CheckPermission(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
	return false, nil
}
func (m *handlerMockRoleRepo) ListAllPermissions(_ context.Context) ([]Permission, error) {
	return []Permission{}, nil
}
func (m *handlerMockRoleRepo) AddPermissions(_ context.Context, _ uuid.UUID, _ []Permission) error {
	return nil
}
func (m *handlerMockRoleRepo) RemoveAllPermissions(_ context.Context, _ uuid.UUID) error {
	return nil
}

type handlerMockAuditRepo struct{}

func (m *handlerMockAuditRepo) Create(_ context.Context, _ *AuditLog) error { return nil }
func (m *handlerMockAuditRepo) List(_ context.Context, _ AuditLogFilter) (*model.ListResponse[AuditLog], error) {
	return &model.ListResponse[AuditLog]{Items: []AuditLog{}}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func handlerHashPassword(pw string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	return string(h)
}

func handlerNewTestRouter(
	userRepo *handlerMockUserRepo,
	roleRepo *handlerMockRoleRepo,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	jwt := NewJWTService("test-secret-key-minimum-32-chars!!")
	svc := NewAdminService(userRepo, roleRepo, &handlerMockAuditRepo{}, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())
	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterAuthRoutes(api)
	h.RegisterAdminRoutes(api)
	return r
}

func handlerJSON(v interface{}) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_Login_Success(t *testing.T) {
	userRepo := &handlerMockUserRepo{
		getByUsernameFn: func(_ context.Context, username string) (*User, error) {
			return &User{
				ID:           uuid.New(),
				Username:     username,
				PasswordHash: handlerHashPassword("correct"),
				Status:       UserStatusActive,
			}, nil
		},
	}
	roleRepo := &handlerMockRoleRepo{
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) {
			return []Role{{Name: "admin"}}, nil
		},
	}
	r := handlerNewTestRouter(userRepo, roleRepo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{Username: "admin", Password: "correct"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp TokenPair
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	userRepo := &handlerMockUserRepo{
		getByUsernameFn: func(_ context.Context, _ string) (*User, error) {
			return &User{
				ID:           uuid.New(),
				PasswordHash: handlerHashPassword("correct"),
				Status:       UserStatusActive,
			}, nil
		},
	}
	r := handlerNewTestRouter(userRepo, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{Username: "admin", Password: "wrong"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Login_BadRequest(t *testing.T) {
	r := handlerNewTestRouter(&handlerMockUserRepo{}, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(map[string]string{}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListUsers(t *testing.T) {
	userRepo := &handlerMockUserRepo{
		listFn: func(_ context.Context, _ UserFilter) (*model.ListResponse[User], error) {
			return model.NewListResponse([]User{
				{ID: uuid.New(), Username: "user1", Status: UserStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				{ID: uuid.New(), Username: "user2", Status: UserStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			}, 2, 1, 20), nil
		},
	}
	r := handlerNewTestRouter(userRepo, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&page_size=20", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[User]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.Total)
}

func TestHandler_CreateUser_Success(t *testing.T) {
	userRepo := &handlerMockUserRepo{
		createFn: func(_ context.Context, user *User) error {
			user.ID = uuid.New()
			user.CreatedAt = time.Now()
			user.UpdatedAt = time.Now()
			return nil
		},
	}
	r := handlerNewTestRouter(userRepo, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
		handlerJSON(CreateUserRequest{Username: "newuser", Password: "password123", DisplayName: "New User"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "newuser", resp.Username)
}

func TestHandler_GetUser_Success(t *testing.T) {
	userID := uuid.New()
	userRepo := &handlerMockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Username: "testuser", Status: UserStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}
	roleRepo := &handlerMockRoleRepo{
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) {
			return []Role{{Name: "operator"}}, nil
		},
	}
	r := handlerNewTestRouter(userRepo, roleRepo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "testuser", resp.Username)
}

func TestHandler_GetUser_NotFound(t *testing.T) {
	r := handlerNewTestRouter(&handlerMockUserRepo{}, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_DeleteUser_Success(t *testing.T) {
	deleteCalled := false
	userRepo := &handlerMockUserRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	r := handlerNewTestRouter(userRepo, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, deleteCalled)
}

func TestHandler_ListRoles(t *testing.T) {
	roleRepo := &handlerMockRoleRepo{
		listFn: func(_ context.Context) ([]Role, error) {
			return []Role{
				{ID: uuid.New(), Name: "admin", CreatedAt: time.Now(), UpdatedAt: time.Now()},
				{ID: uuid.New(), Name: "operator", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			}, nil
		},
	}
	r := handlerNewTestRouter(&handlerMockUserRepo{}, roleRepo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []Role
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}
