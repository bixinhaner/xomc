package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		valid    bool
	}{
		{name: "minimum length", username: "abc", valid: true},
		{name: "letters digits underscore hyphen", username: "admin_01-test", valid: true},
		{name: "too short", username: "ab", valid: false},
		{name: "too long", username: "abcdefghijklmnopqrstuvwxyz1234567", valid: false},
		{name: "contains whitespace", username: "admin user", valid: false},
		{name: "contains password-like symbols", username: "admin(.!@#$%^&*?)", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUsername(tt.username)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestHandler_Login_InvalidUsernameRejectedBeforeSensitiveSideEffects(t *testing.T) {
	const invalidUsername = "admin\n(.!@#$%^&*?)bai1234"
	userLookupCalled := false
	userRepo := &handlerMockUserRepo{
		getByUsernameFn: func(_ context.Context, _ string) (*User, error) {
			userLookupCalled = true
			return nil, commonerrors.ErrNotFound
		},
	}
	auditCreated := make(chan *AuditLog, 1)
	auditRepo := &handlerMockAuditRepoWithCapture{createFn: func(_ context.Context, log *AuditLog) error {
		auditCreated <- log
		return nil
	}}
	loginLogCreated := make(chan CreateLoginLogRequest, 1)
	logRepo := &captureLoginLogRepo{created: loginLogCreated}

	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	require.NoError(t, err)
	svc := NewAdminService(userRepo, &handlerMockRoleRepo{}, &handlerMockMenuRepo{}, auditRepo, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())
	h.SetAllowPlaintextPassword(true)
	h.SetLogRepository(logRepo)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	loginGuard := NewLoginGuard(client)
	h.SetLoginGuard(loginGuard)

	r := gin.New()
	r.POST("/api/v1/auth/login", h.Login)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{Username: invalidUsername, Password: "unused-password"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid username format")
	assert.NotContains(t, w.Body.String(), invalidUsername, "错误响应不得回显非法登录标识")
	assert.False(t, userLookupCalled, "非法 username 不得进入数据库查询")
	assert.Zero(t, loginGuard.GetFailedCount(context.Background(), invalidUsername),
		"非法 username 不得创建账号级失败计数 key")
	select {
	case got := <-auditCreated:
		t.Fatalf("非法 username 不得写入审计，实际记录: %+v", got)
	case got := <-loginLogCreated:
		t.Fatalf("非法 username 不得写入登录日志，实际记录: %+v", got)
	case <-time.After(100 * time.Millisecond):
	}
}
