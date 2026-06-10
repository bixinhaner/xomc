package nedirect

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/ratelimit"
)

// fakeAuthenticator 用固定结果模拟认证。
type fakeAuthenticator struct {
	principal *Principal
	err       error
}

func (f fakeAuthenticator) Authenticate(_ context.Context, _ *http.Request) (*Principal, error) {
	return f.principal, f.err
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// 未注入认证器 → 一律 401，绝不静默放行。
func TestMiddleware_NilAuthenticator_Rejects(t *testing.T) {
	mw := NewMiddleware(nil, nil, nil, zap.NewNop())
	h := mw.Wrap("status", nil, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/nedirect/status", nil)
	w := httptest.NewRecorder()
	h(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 认证失败 → 401。
func TestMiddleware_AuthFails_Returns401(t *testing.T) {
	mw := NewMiddleware(fakeAuthenticator{err: errors.New("bad token")}, nil, nil, zap.NewNop())
	h := mw.Wrap("status", nil, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/nedirect/status", nil)
	w := httptest.NewRecorder()
	h(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 认证成功 → 业务 handler 执行，且 principal 注入 ctx。
func TestMiddleware_AuthSucceeds_InjectsPrincipal(t *testing.T) {
	p := &Principal{UserID: uuid.New(), Username: "admin", IsSuperAdmin: true}
	mw := NewMiddleware(fakeAuthenticator{principal: p}, nil, nil, zap.NewNop())

	var got *Principal
	h := mw.Wrap("status", nil, func(w http.ResponseWriter, r *http.Request) {
		got, _ = PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/nedirect/status", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, got)
	assert.Equal(t, "admin", got.Username)
}

// 端点级限流：超过阈值 → 429。
func TestMiddleware_EndpointRateLimited(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	p := &Principal{UserID: uuid.New(), Username: "admin"}
	endpointLimiter := ratelimit.NewFixedWindowLimiter(rdb, "ne:endpoint", 1, time.Minute)
	mw := NewMiddleware(fakeAuthenticator{principal: p}, endpointLimiter, nil, zap.NewNop())
	h := mw.Wrap("status", nil, okHandler)

	// 第一次放行。
	req1 := httptest.NewRequest(http.MethodGet, "/nedirect/status", nil)
	w1 := httptest.NewRecorder()
	h(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 第二次超限 → 429。
	req2 := httptest.NewRequest(http.MethodGet, "/nedirect/status", nil)
	w2 := httptest.NewRecorder()
	h(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

// FuncAuthenticator：缺凭据 → error；Bearer 走 token validator；X-API-Key 走 key validator。
func TestFuncAuthenticator(t *testing.T) {
	tokenP := &Principal{UserID: uuid.New(), Username: "via-jwt"}
	keyP := &Principal{UserID: uuid.New(), Username: "via-apikey"}
	auth := FuncAuthenticator{
		ValidateToken: func(_ context.Context, tok string) (*Principal, error) {
			if tok == "good" {
				return tokenP, nil
			}
			return nil, errors.New("bad jwt")
		},
		ValidateAPIKey: func(_ context.Context, key string) (*Principal, error) {
			if key == "good-key" {
				return keyP, nil
			}
			return nil, errors.New("bad key")
		},
	}

	// 无凭据 → error。
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	_, err := auth.Authenticate(context.Background(), req)
	assert.Error(t, err)

	// 合法 Bearer。
	reqJWT := httptest.NewRequest(http.MethodGet, "/x", nil)
	reqJWT.Header.Set("Authorization", "Bearer good")
	gotJWT, err := auth.Authenticate(context.Background(), reqJWT)
	require.NoError(t, err)
	assert.Equal(t, "via-jwt", gotJWT.Username)

	// X-API-Key 优先。
	reqKey := httptest.NewRequest(http.MethodGet, "/x", nil)
	reqKey.Header.Set("X-API-Key", "good-key")
	gotKey, err := auth.Authenticate(context.Background(), reqKey)
	require.NoError(t, err)
	assert.Equal(t, "via-apikey", gotKey.Username)

	// 非法 Bearer 格式 → error。
	reqBad := httptest.NewRequest(http.MethodGet, "/x", nil)
	reqBad.Header.Set("Authorization", "Token xyz")
	_, err = auth.Authenticate(context.Background(), reqBad)
	assert.Error(t, err)
}
