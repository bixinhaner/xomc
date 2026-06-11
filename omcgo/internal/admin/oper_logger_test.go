package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockOperLogWriter 是 OperLogWriter 的测试替身：把每次 CreateOperLog 的请求
// 推到 channel，让 fire-and-forget goroutine 的写入可被同步断言（不靠 sleep）。
type mockOperLogWriter struct {
	calls chan CreateOperLogRequest
	err   error
}

func newMockOperLogWriter() *mockOperLogWriter {
	return &mockOperLogWriter{calls: make(chan CreateOperLogRequest, 4)}
}

func (m *mockOperLogWriter) CreateOperLog(_ context.Context, req CreateOperLogRequest) error {
	m.calls <- req
	return m.err
}

// 等待一次写入到达；超时即失败（避免 goroutine 永不触发时测试挂死）。
func (m *mockOperLogWriter) waitOne(t *testing.T) CreateOperLogRequest {
	t.Helper()
	select {
	case req := <-m.calls:
		return req
	case <-time.After(2 * time.Second):
		t.Fatal("expected CreateOperLog to be called, timed out")
		return CreateOperLogRequest{}
	}
}

// 断言在窗口内没有写入发生（用于验证 GET / 匿名请求被跳过）。
func (m *mockOperLogWriter) assertNoCall(t *testing.T) {
	t.Helper()
	select {
	case req := <-m.calls:
		t.Fatalf("expected no CreateOperLog call, got: %+v", req)
	case <-time.After(200 * time.Millisecond):
	}
}

func setupOperLoggerRouter(writer OperLogWriter, withUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if withUser {
		r.Use(func(c *gin.Context) {
			c.Set(CtxKeyUserID, uuid.New())
			c.Set(CtxKeyUsername, "operator")
			c.Next()
		})
	}
	r.Use(OperLogger(writer, zap.NewNop()))
	handler := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
	r.POST("/api/v1/devices/:id", handler)
	r.PUT("/api/v1/alarms/:id", handler)
	r.DELETE("/api/v1/users/:id", handler)
	r.GET("/api/v1/devices", handler)
	r.POST("/api/v1/forbidden", func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "denied"})
	})
	return r
}

func TestOperLogger_RecordsWriteRequest(t *testing.T) {
	writer := newMockOperLogWriter()
	r := setupOperLoggerRouter(writer, true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/abc", nil)
	req.Header.Set("User-Agent", "test-agent/1.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	got := writer.waitOne(t)
	assert.Equal(t, http.MethodPost, got.Action)
	assert.Equal(t, "operator", got.Username)
	assert.Equal(t, "devices", got.Module, "module derived from /api/v1/<seg>")
	assert.Equal(t, "/api/v1/devices/:id", got.Target)
	assert.True(t, got.Status, "2xx → status=true")
	assert.Equal(t, "test-agent/1.0", got.UserAgent)
	assert.Empty(t, got.ErrorMsg)
	assert.NotNil(t, got.UserID)
}

func TestOperLogger_RecordsPutAndDeleteModules(t *testing.T) {
	writer := newMockOperLogWriter()
	r := setupOperLoggerRouter(writer, true)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPut, "/api/v1/alarms/x", nil))
	put := writer.waitOne(t)
	assert.Equal(t, http.MethodPut, put.Action)
	assert.Equal(t, "alarms", put.Module)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodDelete, "/api/v1/users/x", nil))
	del := writer.waitOne(t)
	assert.Equal(t, http.MethodDelete, del.Action)
	assert.Equal(t, "users", del.Module)
}

func TestOperLogger_SkipsGetRequest(t *testing.T) {
	writer := newMockOperLogWriter()
	r := setupOperLoggerRouter(writer, true)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	writer.assertNoCall(t)
}

func TestOperLogger_SkipsAnonymousRequest(t *testing.T) {
	writer := newMockOperLogWriter()
	// 不注入 user → username 为空，写操作也应跳过（无操作人主体）。
	r := setupOperLoggerRouter(writer, false)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	writer.assertNoCall(t)
}

func TestOperLogger_RecordsFailedRequestWithErrorMsg(t *testing.T) {
	writer := newMockOperLogWriter()
	r := setupOperLoggerRouter(writer, true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/forbidden", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)

	got := writer.waitOne(t)
	assert.False(t, got.Status, "4xx → status=false")
	assert.Equal(t, http.StatusText(http.StatusForbidden), got.ErrorMsg)
}

func TestOperLogger_NilRepoIsNoop(t *testing.T) {
	// logRepo 未注入（nil）时中间件整体降级为 no-op，不得 panic。
	r := setupOperLoggerRouter(nil, true)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/abc", nil)
	w := httptest.NewRecorder()
	assert.NotPanics(t, func() { r.ServeHTTP(w, req) })
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperModuleFromPath(t *testing.T) {
	cases := map[string]string{
		"/api/v1/devices/:id":      "devices",
		"/api/v1/alarms":           "alarms",
		"/api/v1/admin/logs/login": "admin",
		"/api/v1/":                 "system",
		"":                         "system",
		// 无 /api/v1/ 前缀时回退到首段（OperLogger 仅挂 v1 组，实际不会命中此路径）。
		"/health": "health",
	}
	for in, want := range cases {
		assert.Equalf(t, want, operModuleFromPath(in), "path=%q", in)
	}
}
