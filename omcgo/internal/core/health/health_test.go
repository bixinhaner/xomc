package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLivenessHandler_AlwaysOK liveness 永远 200，即使没有任何依赖检查。
// 模拟 `curl /healthz` → 期望 HTTP 200 + status:"ok"。
func TestLivenessHandler_AlwaysOK(t *testing.T) {
	handler := LivenessHandler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var body Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body.Status)
	assert.Empty(t, body.Components)
}

// TestReadinessHandler_AllHealthy 所有依赖 OK 时返 200，components 包含每项明细。
func TestReadinessHandler_AllHealthy(t *testing.T) {
	pgChecker := NewChecker("postgres", func(ctx context.Context) error { return nil })
	redisChecker := NewChecker("redis", func(ctx context.Context) error { return nil })

	handler := ReadinessHandler(2*time.Second, pgChecker, redisChecker)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body.Status)
	require.Len(t, body.Components, 2)
	for _, r := range body.Components {
		assert.Equal(t, "healthy", r.Status)
		assert.Empty(t, r.Error)
		assert.NotEmpty(t, r.Latency)
	}
}

// TestReadinessHandler_OneUnhealthy 任一依赖失败 → 整体 503，但 components 仍完整列出。
// 模拟 `curl /readyz`（依赖故障）→ 期望 HTTP 503。
func TestReadinessHandler_OneUnhealthy(t *testing.T) {
	pgChecker := NewChecker("postgres", func(ctx context.Context) error { return nil })
	redisChecker := NewChecker("redis", func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	handler := ReadinessHandler(2*time.Second, pgChecker, redisChecker)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "unhealthy", body.Status)
	require.Len(t, body.Components, 2)

	byName := map[string]Result{}
	for _, r := range body.Components {
		byName[r.Name] = r
	}
	assert.Equal(t, "healthy", byName["postgres"].Status)
	assert.Equal(t, "unhealthy", byName["redis"].Status)
	assert.Equal(t, "connection refused", byName["redis"].Error)
}

// TestReadinessHandler_NoCheckers 没有任何 Checker 时退化为 liveness：恒 200。
// 这保证了即使某个进程没有外部依赖（理论场景），/readyz 也不会因空集合误报。
func TestReadinessHandler_NoCheckers(t *testing.T) {
	handler := ReadinessHandler(0)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body.Status)
	assert.Empty(t, body.Components)
}

// TestDynamicReadinessHandler_RefreshesCheckers 验证 handler 创建后新增的检查项
// 会被后续 /readyz 请求看到。app 会提前启动 metrics 端口，模块健康检查在
// provider.Setup 期间陆续注册；这里防止提前启动导致 readyz 永久停留在旧快照。
func TestDynamicReadinessHandler_RefreshesCheckers(t *testing.T) {
	checkers := []Checker{
		NewChecker("postgres", func(ctx context.Context) error { return nil }),
	}
	handler := DynamicReadinessHandler(time.Second, func() []Checker {
		return checkers
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, req)
	require.Equal(t, http.StatusOK, first.Code)

	var firstBody Response
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &firstBody))
	require.Len(t, firstBody.Components, 1)
	assert.Equal(t, "postgres", firstBody.Components[0].Name)

	checkers = append(checkers, NewChecker("dictload", func(ctx context.Context) error {
		return errors.New("warming up")
	}))

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)
	require.Equal(t, http.StatusServiceUnavailable, second.Code)

	var secondBody Response
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &secondBody))
	require.Len(t, secondBody.Components, 2)
	assert.Equal(t, "dictload", secondBody.Components[1].Name)
	assert.Equal(t, "unhealthy", secondBody.Components[1].Status)
	assert.Equal(t, "warming up", secondBody.Components[1].Error)
}

// TestReadinessHandler_Timeout 超时后 Checker 收到取消信号；通过返回 ctx.Err() 表示故障。
func TestReadinessHandler_Timeout(t *testing.T) {
	slowChecker := NewChecker("slow", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return nil
		}
	})

	handler := ReadinessHandler(20*time.Millisecond, slowChecker)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "unhealthy", body.Status)
	require.Len(t, body.Components, 1)
	assert.Equal(t, "unhealthy", body.Components[0].Status)
	assert.Contains(t, body.Components[0].Error, "context")
}

// TestReadinessHandler_PanicSafe 单个 Checker panic 不影响其他 Checker。
func TestReadinessHandler_PanicSafe(t *testing.T) {
	panicChecker := NewChecker("panic", func(ctx context.Context) error {
		panic("boom")
	})
	okChecker := NewChecker("ok", func(ctx context.Context) error { return nil })

	handler := ReadinessHandler(time.Second, panicChecker, okChecker)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Components, 2)

	byName := map[string]Result{}
	for _, r := range body.Components {
		byName[r.Name] = r
	}
	assert.Equal(t, "unhealthy", byName["panic"].Status)
	assert.Equal(t, "healthy", byName["ok"].Status)
}
