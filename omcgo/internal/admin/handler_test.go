package admin

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin/loginpwd"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
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
func (m *handlerMockUserRepo) GetUsernamesByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}

type handlerMockRoleRepo struct {
	listFn         func(ctx context.Context) ([]Role, error)
	getUserRolesFn func(ctx context.Context, userID uuid.UUID) ([]Role, error)
}

func (m *handlerMockRoleRepo) Create(_ context.Context, _ *Role) error { return nil }
func (m *handlerMockRoleRepo) GetByID(_ context.Context, _ uuid.UUID) (*Role, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *handlerMockRoleRepo) GetByName(_ context.Context, _ string) (*Role, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *handlerMockRoleRepo) Update(_ context.Context, _ *Role) error     { return nil }
func (m *handlerMockRoleRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *handlerMockRoleRepo) List(ctx context.Context) ([]Role, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []Role{}, nil
}
func (m *handlerMockRoleRepo) AssignRole(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *handlerMockRoleRepo) RemoveRole(_ context.Context, _, _ uuid.UUID) error { return nil }
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
func (m *handlerMockRoleRepo) GetUserRolesBatch(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]Role, error) {
	return nil, nil
}
func (m *handlerMockRoleRepo) ListWithPagination(_ context.Context, _ RoleFilter) (*model.ListResponse[Role], error) {
	return model.NewListResponse([]Role{}, 0, 1, 20), nil
}
func (m *handlerMockRoleRepo) GetDefaultRoleID(_ context.Context, _ uuid.UUID) (*uuid.UUID, error) {
	return nil, nil
}
func (m *handlerMockRoleRepo) SetDefaultRole(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (m *handlerMockRoleRepo) ListRoleUsers(_ context.Context, _ uuid.UUID, _, _ int) ([]RoleUserItem, int64, error) {
	return []RoleUserItem{}, 0, nil
}
func (m *handlerMockRoleRepo) ListUserIDsByRole(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

type handlerMockAuditRepo struct{}

func (m *handlerMockAuditRepo) Create(_ context.Context, _ *AuditLog) error { return nil }
func (m *handlerMockAuditRepo) List(_ context.Context, _ AuditLogFilter) (*model.ListResponse[AuditLog], error) {
	return &model.ListResponse[AuditLog]{Items: []AuditLog{}}, nil
}

type handlerMockMenuRepo struct{}

func (m *handlerMockMenuRepo) Create(_ context.Context, _ *Menu, _ uuid.UUID) error  { return nil }
func (m *handlerMockMenuRepo) GetByID(_ context.Context, _ uuid.UUID) (*Menu, error) { return nil, nil }
func (m *handlerMockMenuRepo) GetByPermissionKey(_ context.Context, _ string) (*Menu, error) {
	return nil, nil
}
func (m *handlerMockMenuRepo) List(_ context.Context, _ MenuFilter) (*model.ListResponse[Menu], error) {
	return &model.ListResponse[Menu]{Items: []Menu{}}, nil
}
func (m *handlerMockMenuRepo) Update(_ context.Context, _ uuid.UUID, _ *UpdateMenuRequest, _ uuid.UUID) error {
	return nil
}
func (m *handlerMockMenuRepo) Delete(_ context.Context, _ []uuid.UUID) error { return nil }
func (m *handlerMockMenuRepo) GetTree(_ context.Context, _ *MenuStatus) ([]Menu, error) {
	return nil, nil
}
func (m *handlerMockMenuRepo) GetByRole(_ context.Context, _ uuid.UUID) ([]Menu, error) {
	return nil, nil
}
func (m *handlerMockMenuRepo) GetByUser(_ context.Context, _ uuid.UUID) ([]Menu, error) {
	return nil, nil
}
func (m *handlerMockMenuRepo) GetAllActive(_ context.Context) ([]Menu, error) {
	return nil, nil
}
func (m *handlerMockMenuRepo) SetRoleMenus(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (m *handlerMockMenuRepo) GetRoleMenuIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func handlerHashPassword(pw string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	return string(h)
}

// handlerTestEnv 聚合一次性的测试上下文：除路由 Engine 外还携带登录密码加密所需的
// keystore / cipher / publicKey，便于测试构造合法的加密 LoginRequest。
type handlerTestEnv struct {
	Engine    *gin.Engine
	KeyID     string
	PublicKey *rsa.PublicKey
	Now       time.Time
}

// encryptPassword 构造前端等价物：把 {password, ts, nonce} JSON 用 RSA-OAEP-SHA256
// 加密 → base64。每次调用 nonce 都会变化，可避免单测内 ReplayGuard 误判。
func (e *handlerTestEnv) encryptPassword(t *testing.T, plain string) string {
	t.Helper()
	nonceBytes := make([]byte, 16)
	_, err := rand.Read(nonceBytes)
	require.NoError(t, err)

	body, err := json.Marshal(loginpwd.Payload{
		Password: plain,
		TS:       e.Now.Unix(),
		Nonce:    base64.RawURLEncoding.EncodeToString(nonceBytes),
	})
	require.NoError(t, err)

	ct, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, e.PublicKey, body, nil)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(ct)
}

// handlerNewTestEnv 与 handlerNewTestRouter 等价但额外暴露加密所需信息。
//
// 测试需要构造合法 LoginRequest / CreateUserHTTPRequest / ResetPasswordRequest /
// ChangePasswordHTTPRequest 时调用本函数；其它测试可继续调 handlerNewTestRouter。
func handlerNewTestEnv(
	t *testing.T,
	userRepo *handlerMockUserRepo,
	roleRepo *handlerMockRoleRepo,
) *handlerTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	require.NoError(t, err)
	svc := NewAdminService(userRepo, roleRepo, &handlerMockMenuRepo{}, &handlerMockAuditRepo{}, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())

	now := time.Now()
	ks, err := loginpwd.NewKeystore(filepath.Join(t.TempDir(), "key.pem"))
	require.NoError(t, err)
	guard := loginpwd.NewMemReplayGuard(func() time.Time { return now })
	cipher := loginpwd.NewCipher(ks, guard)
	h.SetLoginCipher(cipher)
	pubHandler := loginpwd.NewPublicKeyHandler(cipher)

	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterAuthRoutes(api, pubHandler)
	h.RegisterAdminRoutes(api)

	return &handlerTestEnv{
		Engine:    r,
		KeyID:     ks.ActiveKeyID(),
		PublicKey: parsePublicKeyForTest(t, ks.ActivePublicKeyPEM()),
		Now:       now,
	}
}

func handlerNewTestRouter(
	userRepo *handlerMockUserRepo,
	roleRepo *handlerMockRoleRepo,
) *gin.Engine {
	// nil-tolerant fallback: tests in this file pre-date *testing.T threading;
	// 走 handlerNewTestEnv 即可，但 t 拿不到。这里用一个最简等价路径。
	gin.SetMode(gin.TestMode)
	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	if err != nil {
		panic(err)
	}
	svc := NewAdminService(userRepo, roleRepo, &handlerMockMenuRepo{}, &handlerMockAuditRepo{}, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())

	keyDir, err := filepathTempDir()
	if err != nil {
		panic(err)
	}
	ks, err := loginpwd.NewKeystore(filepath.Join(keyDir, "key.pem"))
	if err != nil {
		panic(err)
	}
	cipher := loginpwd.NewCipher(ks, loginpwd.NewMemReplayGuard(nil))
	h.SetLoginCipher(cipher)
	pubHandler := loginpwd.NewPublicKeyHandler(cipher)

	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterAuthRoutes(api, pubHandler)
	h.RegisterAdminRoutes(api)
	return r
}

// filepathTempDir 在 testing.T 不可达时分配一个进程级临时目录，进程退出由 OS 回收。
func filepathTempDir() (string, error) {
	return os.MkdirTemp("", "admin-handler-test-keys-*")
}

// parsePublicKeyForTest 把 PEM 字符串解码为 *rsa.PublicKey。
func parsePublicKeyForTest(t *testing.T, pemStr string) *rsa.PublicKey {
	t.Helper()
	block, _ := pem.Decode([]byte(pemStr))
	require.NotNil(t, block)
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	rsaPub, ok := pub.(*rsa.PublicKey)
	require.True(t, ok)
	return rsaPub
}

// encryptPayloadForTest 用前端等价流程构造一份 RSA-OAEP-SHA256 + base64 密文。
func encryptPayloadForTest(t *testing.T, pub *rsa.PublicKey, plain string, now time.Time) string {
	t.Helper()
	nonceBytes := make([]byte, 16)
	_, err := rand.Read(nonceBytes)
	require.NoError(t, err)
	body, err := json.Marshal(loginpwd.Payload{
		Password: plain,
		TS:       now.Unix(),
		Nonce:    base64.RawURLEncoding.EncodeToString(nonceBytes),
	})
	require.NoError(t, err)
	ct, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, body, nil)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(ct)
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
	env := handlerNewTestEnv(t, userRepo, roleRepo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{
			Username:          "admin",
			EncryptedPassword: env.encryptPassword(t, "correct"),
			KeyID:             env.KeyID,
		}))
	req.Header.Set("Content-Type", "application/json")
	env.Engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp TokenPair
	response.DecodeData(t, w.Body, &resp)
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
	env := handlerNewTestEnv(t, userRepo, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{
			Username:          "admin",
			EncryptedPassword: env.encryptPassword(t, "wrong"),
			KeyID:             env.KeyID,
		}))
	req.Header.Set("Content-Type", "application/json")
	env.Engine.ServeHTTP(w, req)

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
	roleID := uuid.New()
	userRepo := &handlerMockUserRepo{
		listFn: func(_ context.Context, filter UserFilter) (*model.ListResponse[User], error) {
			require.NotNil(t, filter.Status)
			assert.Equal(t, UserStatusDisabled, *filter.Status)
			require.NotNil(t, filter.RoleID)
			assert.Equal(t, roleID.String(), *filter.RoleID)
			return model.NewListResponse([]User{
				{ID: uuid.New(), Username: "user1", Status: UserStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				{ID: uuid.New(), Username: "user2", Status: UserStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			}, 2, 1, 20), nil
		},
	}
	r := handlerNewTestRouter(userRepo, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users?page=1&page_size=20&status=disabled&role_id=%s", roleID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[User]
	response.DecodeData(t, w.Body, &resp)
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
	env := handlerNewTestEnv(t, userRepo, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
		handlerJSON(CreateUserHTTPRequest{
			Username:          "newuser",
			EncryptedPassword: env.encryptPassword(t, "password123"),
			KeyID:             env.KeyID,
			DisplayName:       "New User",
			Email:             "new@example.com",
		}))
	req.Header.Set("Content-Type", "application/json")
	env.Engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp User
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "newuser", resp.Username)
}

func TestHandler_UserEmailRequired(t *testing.T) {
	env := handlerNewTestEnv(t, &handlerMockUserRepo{}, &handlerMockRoleRepo{})

	t.Run("create rejects missing email", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
			handlerJSON(CreateUserHTTPRequest{
				Username:          "newuser",
				EncryptedPassword: env.encryptPassword(t, "password123"),
				KeyID:             env.KeyID,
			}))
		req.Header.Set("Content-Type", "application/json")
		env.Engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update rejects missing email", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+uuid.NewString(),
			bytes.NewBufferString(`{"display_name":"new name"}`))
		req.Header.Set("Content-Type", "application/json")
		env.Engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UserPhoneValidation(t *testing.T) {
	env := handlerNewTestEnv(t, &handlerMockUserRepo{}, &handlerMockRoleRepo{})

	t.Run("update rejects invalid phone", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+uuid.NewString(),
			bytes.NewBufferString(`{"email":"user@example.com","phone":"123#456"}`))
		req.Header.Set("Content-Type", "application/json")
		env.Engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestHandler_CreateUser_InvalidUsername_Rejects 验证 issue #686 修复：
// 用户名格式校验（只允许字母、数字、下划线、减号）。
func TestHandler_CreateUser_InvalidUsername_Rejects(t *testing.T) {
	userRepo := &handlerMockUserRepo{}
	env := handlerNewTestEnv(t, userRepo, &handlerMockRoleRepo{})

	testCases := []struct {
		name     string
		username string
	}{
		{"中文用户名", "测试用户"},
		{"含空格", "user name"},
		{"含特殊字符", "user@name"},
		{"含点号", "user.name"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
				handlerJSON(CreateUserHTTPRequest{
					Username:          tc.username,
					EncryptedPassword: env.encryptPassword(t, "password123"),
					KeyID:             env.KeyID,
					Email:             "new@example.com",
				}))
			req.Header.Set("Content-Type", "application/json")
			env.Engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, "用户名 %q 应该被拒绝", tc.username)
		})
	}
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
	response.DecodeData(t, w.Body, &resp)
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
		// service.DeleteUser 在删除前需先 GetByID 校验 source（仅非内置用户允许删除）。
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Source: UserSourceAdmin}, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	r := handlerNewTestRouter(userRepo, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, deleteCalled)
}

func TestClassifyLoginFailure(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "user not found",
			err:      fmt.Errorf("%w: %w", errLoginUserNotFound, commonerrors.ErrUnauthorized),
			expected: "user_not_found",
		},
		{
			name:     "account disabled",
			err:      fmt.Errorf("%w: %w", errLoginAccountDisabled, commonerrors.ErrForbidden),
			expected: "account_disabled",
		},
		{
			name:     "wrong password",
			err:      fmt.Errorf("%w: %w", errLoginWrongPassword, commonerrors.ErrUnauthorized),
			expected: "wrong_password",
		},
		{
			name:     "unknown error",
			err:      fmt.Errorf("some other error"),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyLoginFailure(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLoginErrorToFriendlyError 验证内部 error → 用户面文案的翻译表，并防回归
// "wrong password: unauthorized" 泄露 Go 错误链字符串。
//
// 文案策略：字面拆分（密码错误 vs 用户不存在），UX 优先；用户名枚举防护
// 由 LoginGuard 失败计数 + IPGuard 限流 + CAPTCHA 兜底。
func TestLoginErrorToFriendlyError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    int
		wantMsg     string
		shouldMatch bool // BusinessError 应能被 errors.As 匹配
	}{
		{
			name:        "wrong password → 密码错误",
			err:         fmt.Errorf("%w: %w", errLoginWrongPassword, commonerrors.ErrUnauthorized),
			wantCode:    7005,
			wantMsg:     "密码错误",
			shouldMatch: true,
		},
		{
			name:        "user not found → 用户不存在",
			err:         fmt.Errorf("%w: %w", errLoginUserNotFound, commonerrors.ErrUnauthorized),
			wantCode:    7001,
			wantMsg:     "用户不存在",
			shouldMatch: true,
		},
		{
			name:        "account disabled",
			err:         fmt.Errorf("%w: %w", errLoginAccountDisabled, commonerrors.ErrForbidden),
			wantCode:    7002,
			wantMsg:     "账号已被禁用，请联系管理员",
			shouldMatch: true,
		},
		{
			name:        "纯 ErrUnauthorized fallback",
			err:         commonerrors.ErrUnauthorized,
			wantCode:    7000,
			wantMsg:     "登录凭据无效，请重试",
			shouldMatch: true,
		},
		{
			name:        "未分类错误 fallback 友好文案",
			err:         fmt.Errorf("some unexpected error"),
			wantCode:    7099,
			wantMsg:     "登录失败，请稍后重试",
			shouldMatch: true,
		},		{
			name:        "#690：已有 BusinessError 直接透传",
			err:         commonerrors.NewBusinessError(7012, "该账号已锁定，请 5 分钟后重试", commonerrors.ErrForbidden),
			wantCode:    7012,
			wantMsg:     "该账号已锁定，请 5 分钟后重试",
			shouldMatch: true,
		},	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			friendly := loginErrorToFriendlyError(tt.err)
			var bErr *commonerrors.BusinessError
			ok := errors.As(friendly, &bErr)
			assert.Equal(t, tt.shouldMatch, ok)
			if ok {
				assert.Equal(t, tt.wantCode, bErr.Code)
				assert.Equal(t, tt.wantMsg, bErr.Message)
				// 防回归：友好 msg 中不应包含 Go 错误链字符串
				assert.NotContains(t, bErr.Message, "wrong password")
				assert.NotContains(t, bErr.Message, "unauthorized")
				assert.NotContains(t, bErr.Message, ":")
			}
		})
	}
}

func TestHandler_Login_AuditLog_OnFailure(t *testing.T) {
	userRepo := &handlerMockUserRepo{
		getByUsernameFn: func(_ context.Context, _ string) (*User, error) {
			return &User{
				ID:           uuid.New(),
				PasswordHash: handlerHashPassword("correct"),
				Status:       UserStatusActive,
			}, nil
		},
	}

	auditCreated := make(chan *AuditLog, 1)
	auditRepo := &handlerMockAuditRepoWithCapture{
		createFn: func(_ context.Context, log *AuditLog) error {
			auditCreated <- log
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	require.NoError(t, err)
	svc := NewAdminService(userRepo, &handlerMockRoleRepo{}, &handlerMockMenuRepo{}, auditRepo, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())

	now := time.Now()
	ks, err := loginpwd.NewKeystore(filepath.Join(t.TempDir(), "key.pem"))
	require.NoError(t, err)
	cipher := loginpwd.NewCipher(ks, loginpwd.NewMemReplayGuard(func() time.Time { return now }))
	h.SetLoginCipher(cipher)
	pubKey := parsePublicKeyForTest(t, ks.ActivePublicKeyPEM())
	encrypted := encryptPayloadForTest(t, pubKey, "wrong", now)

	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterAuthRoutes(api, loginpwd.NewPublicKeyHandler(cipher))

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{
			Username:          "admin",
			EncryptedPassword: encrypted,
			KeyID:             ks.ActiveKeyID(),
		}))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "TestAgent/1.0")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Wait for the async audit log goroutine
	select {
	case log := <-auditCreated:
		assert.Equal(t, "login_failed", log.Action)
		assert.Equal(t, "admin", log.Username)
		assert.Equal(t, "auth", log.Resource)
		assert.Equal(t, "TestAgent/1.0", log.UserAgent)
		assert.Equal(t, "wrong_password", log.Details["reason"])
	case <-time.After(2 * time.Second):
		t.Fatal("audit log was not created within timeout")
	}
}

func TestHandler_Login_AuditLog_OnSuccess(t *testing.T) {
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

	auditCreated := make(chan *AuditLog, 1)
	auditRepo := &handlerMockAuditRepoWithCapture{
		createFn: func(_ context.Context, log *AuditLog) error {
			auditCreated <- log
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	require.NoError(t, err)
	svc := NewAdminService(userRepo, roleRepo, &handlerMockMenuRepo{}, auditRepo, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())

	now := time.Now()
	ks, err := loginpwd.NewKeystore(filepath.Join(t.TempDir(), "key.pem"))
	require.NoError(t, err)
	cipher := loginpwd.NewCipher(ks, loginpwd.NewMemReplayGuard(func() time.Time { return now }))
	h.SetLoginCipher(cipher)
	pubKey := parsePublicKeyForTest(t, ks.ActivePublicKeyPEM())
	encrypted := encryptPayloadForTest(t, pubKey, "correct", now)

	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterAuthRoutes(api, loginpwd.NewPublicKeyHandler(cipher))

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{
			Username:          "admin",
			EncryptedPassword: encrypted,
			KeyID:             ks.ActiveKeyID(),
		}))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case log := <-auditCreated:
		assert.Equal(t, "login_success", log.Action)
		assert.Equal(t, "admin", log.Username)
		assert.Equal(t, "auth", log.Resource)
	case <-time.After(2 * time.Second):
		t.Fatal("audit log was not created within timeout")
	}
}

// handlerMockAuditRepoWithCapture captures audit log Create calls for test assertions.
type handlerMockAuditRepoWithCapture struct {
	createFn func(ctx context.Context, log *AuditLog) error
}

func (m *handlerMockAuditRepoWithCapture) Create(ctx context.Context, log *AuditLog) error {
	if m.createFn != nil {
		return m.createFn(ctx, log)
	}
	return nil
}

func (m *handlerMockAuditRepoWithCapture) List(_ context.Context, _ AuditLogFilter) (*model.ListResponse[AuditLog], error) {
	return &model.ListResponse[AuditLog]{Items: []AuditLog{}}, nil
}

func TestHandler_ListRoles(t *testing.T) {
	roleRepo := &handlerMockRoleRepo{}
	r := handlerNewTestRouter(&handlerMockUserRepo{}, roleRepo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles?page=1&page_size=20", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[Role]
	response.DecodeData(t, w.Body, &resp)
}

// ===================== issue #649 「默认密码」接通 — handler 层 =====================
//
// service 层行为矩阵已在 service_test.go 充分覆盖；本处仅验 handler 分支语义：
//   - UseDefaultPassword=true → 跳过密码字段必填校验、不解密、透传 service
//   - UseDefaultPassword=false / 缺省 → 走现有 T-0120 双路径
//
// 默认密码兜底来源：handlerNewTestEnv 未注入 SecurityPolicy，service.policySnapshot()
// 退化到 defaultPolicy() 的 fail-safe 常量 "OMC@123456"，足以覆盖 happy path。

func TestHandler_CreateUser_UseDefaultPassword_Succeeds(t *testing.T) {
	var created *User
	userRepo := &handlerMockUserRepo{
		createFn: func(_ context.Context, user *User) error {
			user.ID = uuid.New()
			user.CreatedAt = time.Now()
			user.UpdatedAt = time.Now()
			created = user
			return nil
		},
	}
	env := handlerNewTestEnv(t, userRepo, &handlerMockRoleRepo{})

	// 注意：use_default_password=true 时不带 encrypted_password / password / key_id。
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
		handlerJSON(CreateUserHTTPRequest{
			Username:           "newuser",
			UseDefaultPassword: true,
			DisplayName:        "New User",
			Email:              "new@example.com",
		}))
	req.Header.Set("Content-Type", "application/json")
	env.Engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, created)
	assert.True(t, created.MustChangePassword, "硬规则：默认密码创建即强制首次改密")
}

func TestHandler_CreateUser_MissingPassword_NoUseDefault_Rejects(t *testing.T) {
	env := handlerNewTestEnv(t, &handlerMockUserRepo{}, &handlerMockRoleRepo{})

	// 三个密码字段全空 + 未启用 use_default_password → handler 400 "missing password"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
		handlerJSON(CreateUserHTTPRequest{Username: "newuser", Email: "new@example.com"}))
	req.Header.Set("Content-Type", "application/json")
	env.Engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing password")
}

func TestHandler_ResetPassword_UseDefaultPassword_Succeeds(t *testing.T) {
	target := uuid.New()
	updateCalled := false
	userRepo := &handlerMockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "alice", Source: UserSourceAdmin}, nil
		},
		updatePasswordFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			updateCalled = true
			return nil
		},
	}
	env := handlerNewTestEnv(t, userRepo, &handlerMockRoleRepo{})

	// use_default_password=true 时不带 encrypted_new_password / new_password / key_id。
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+target.String()+"/reset-password",
		handlerJSON(ResetPasswordRequest{UseDefaultPassword: true}))
	req.Header.Set("Content-Type", "application/json")
	env.Engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, updateCalled, "service 应消费默认密码并调 UpdatePassword")
}

func TestHandler_ResetPassword_MissingPassword_NoUseDefault_Rejects(t *testing.T) {
	target := uuid.New()
	userRepo := &handlerMockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*User, error) {
			return &User{ID: target, Username: "alice", Source: UserSourceAdmin}, nil
		},
	}
	env := handlerNewTestEnv(t, userRepo, &handlerMockRoleRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+target.String()+"/reset-password",
		handlerJSON(ResetPasswordRequest{}))
	req.Header.Set("Content-Type", "application/json")
	env.Engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing password")
}
