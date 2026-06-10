package admin

// #122 回归测试：登录成功/失败路径必须写 sys_login_logs（经 LogRepository.CreateLoginLog），
// 且写失败仅 Warn、绝不阻塞登录流。复用 handler_test.go 的 mock 仓库与 helper。

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// captureLoginLogRepo 实现 LogRepository，捕获 CreateLoginLog 调用供断言。
type captureLoginLogRepo struct {
	created   chan CreateLoginLogRequest
	createErr error
}

var _ LogRepository = (*captureLoginLogRepo)(nil)

func (m *captureLoginLogRepo) CreateLoginLog(_ context.Context, req CreateLoginLogRequest) error {
	if m.created != nil {
		m.created <- req
	}
	return m.createErr
}

func (m *captureLoginLogRepo) ListLoginLogs(_ context.Context, _ LoginLogFilter) (*model.ListResponse[LoginLog], error) {
	return &model.ListResponse[LoginLog]{Items: []LoginLog{}}, nil
}

func (m *captureLoginLogRepo) CreateOperLog(_ context.Context, _ CreateOperLogRequest) error {
	return nil
}

func (m *captureLoginLogRepo) ListOperLogs(_ context.Context, _ OperLogFilter) (*model.ListResponse[OperLog], error) {
	return &model.ListResponse[OperLog]{Items: []OperLog{}}, nil
}

func (m *captureLoginLogRepo) CreateTaskLog(_ context.Context, _ CreateTaskLogRequest) error {
	return nil
}

func (m *captureLoginLogRepo) ListTaskLogs(_ context.Context, _ TaskLogFilter) (*model.ListResponse[TaskLog], error) {
	return &model.ListResponse[TaskLog]{Items: []TaskLog{}}, nil
}

// newLoginLogTestRouter 构造走明文密码路径（T-0120 allowPlaintext=true）的登录路由，
// 省去 RSA keystore 样板，让测试聚焦登录日志写入。
func newLoginLogTestRouter(t *testing.T, logRepo LogRepository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

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

	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	require.NoError(t, err)
	svc := NewAdminService(userRepo, roleRepo, &handlerMockMenuRepo{}, &handlerMockAuditRepo{}, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())
	h.SetAllowPlaintextPassword(true)
	h.SetLogRepository(logRepo)

	r := gin.New()
	r.POST("/api/v1/auth/login", h.Login)
	return r
}

func TestHandler_Login_WritesLoginLog(t *testing.T) {
	const testUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	tests := []struct {
		name        string
		password    string
		wantHTTP    int
		wantStatus  bool
		wantMessage string
	}{
		{
			name:        "登录成功写 status=true",
			password:    "correct",
			wantHTTP:    http.StatusOK,
			wantStatus:  true,
			wantMessage: "plaintext_login",
		},
		{
			name:        "密码错误写 status=false + reason",
			password:    "wrong",
			wantHTTP:    http.StatusUnauthorized,
			wantStatus:  false,
			wantMessage: "wrong_password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logRepo := &captureLoginLogRepo{created: make(chan CreateLoginLogRequest, 1)}
			r := newLoginLogTestRouter(t, logRepo)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
				handlerJSON(LoginRequest{Username: "admin", Password: tt.password}))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", testUA)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantHTTP, w.Code)

			// 登录日志为 fire-and-forget goroutine，等待异步写入
			select {
			case got := <-logRepo.created:
				assert.Equal(t, "admin", got.Username)
				assert.Equal(t, tt.wantStatus, got.Status)
				assert.Equal(t, tt.wantMessage, got.Message)
				assert.NotEmpty(t, got.IPAddress)
				assert.Equal(t, "Chrome", got.Browser)
				assert.Equal(t, "macOS", got.OS)
			case <-time.After(2 * time.Second):
				t.Fatal("login log was not created within timeout")
			}
		})
	}
}

func TestHandler_Login_LoginLogWriteFailure_DoesNotBlockLogin(t *testing.T) {
	logRepo := &captureLoginLogRepo{
		created:   make(chan CreateLoginLogRequest, 1),
		createErr: errors.New("db down"),
	}
	r := newLoginLogTestRouter(t, logRepo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{Username: "admin", Password: "correct"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// 写日志失败仅 Warn，不影响登录结果
	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case got := <-logRepo.created:
		assert.True(t, got.Status)
	case <-time.After(2 * time.Second):
		t.Fatal("login log write was not attempted within timeout")
	}
}

func TestHandler_Login_NoLogRepo_NoPanic(t *testing.T) {
	// logRepo 未注入（nil）→ recordLoginLog no-op，登录流不受影响
	r := newLoginLogTestRouter(t, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{Username: "admin", Password: "correct"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestParseUserAgent(t *testing.T) {
	tests := []struct {
		name        string
		ua          string
		wantBrowser string
		wantOS      string
	}{
		{
			name:        "Chrome on macOS",
			ua:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantBrowser: "Chrome",
			wantOS:      "macOS",
		},
		{
			name:        "Edge on Windows（Edg/ 优先于 Chrome）",
			ua:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			wantBrowser: "Edge",
			wantOS:      "Windows",
		},
		{
			name:        "Firefox on Linux",
			ua:          "Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
			wantBrowser: "Firefox",
			wantOS:      "Linux",
		},
		{
			name:        "Safari on iPhone",
			ua:          "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantBrowser: "Safari",
			wantOS:      "iOS",
		},
		{
			name:        "Chrome on Android（Android 优先于 Linux）",
			ua:          "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			wantBrowser: "Chrome",
			wantOS:      "Android",
		},
		{
			name:        "curl 等脚本类回退保留原始 UA",
			ua:          "curl/8.4.0",
			wantBrowser: "curl/8.4.0",
			wantOS:      "unknown",
		},
		{
			name:        "空 UA",
			ua:          "",
			wantBrowser: "unknown",
			wantOS:      "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			browser, osName := parseUserAgent(tt.ua)
			assert.Equal(t, tt.wantBrowser, browser)
			assert.Equal(t, tt.wantOS, osName)
		})
	}
}
