package product

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Match 端点回归测试（#125-product）：
//
// registry.MatchProductClass 在 productClass 未命中任何 active pattern 时返回
// (nil, ErrOrphan)。ErrOrphan 不是错误而是「正常的未命中查询结果」，handler
// 必须把它映射为 200 + matched=false，而不是 AbortWithError(500)。
//
// 本测试用真实 Registry（NewRegistry）+ 同包 fakeRepo（registry_test.go）驱动 handler，
// 覆盖命中 / 未命中两条路径以及缺参 400 路径。
func newMatchTestHandler(t *testing.T) (*Handler, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))
	h := &Handler{registry: r, logger: zap.NewNop()}
	return h, repo
}

// doMatch 用 gin 测试上下文跑一次 Match，并解析统一信封。
func doMatch(t *testing.T, h *Handler, query string) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	target := "/api/v1/products/match"
	if query != "" {
		target += "?" + query
	}
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	h.Match(c)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	return w.Code, envelope
}

func TestMatch_Miss_ReturnsMatchedFalse(t *testing.T) {
	h, _ := newMatchTestHandler(t) // 空 registry — 任何 class 都未命中 → ErrOrphan
	code, env := doMatch(t, h, "productClass="+url.QueryEscape("SMK-NOMATCH-xyz"))

	assert.Equal(t, http.StatusOK, code, "未命中（ErrOrphan）必须 200，不能 500")
	assert.EqualValues(t, 1, env["ret"], "信封 ret=1（成功）")
	data, ok := env["data"].(map[string]any)
	require.True(t, ok, "data 应为对象，实际 %T", env["data"])
	assert.Equal(t, false, data["matched"], "未命中返回 matched=false")
	assert.Equal(t, "SMK-NOMATCH-xyz", data["product_class"])
}

func TestMatch_Hit_ReturnsMatchedTrue(t *testing.T) {
	h, repo := newMatchTestHandler(t)
	pid := repo.addProduct("QRTB", "Baicells", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^QRTB", 1)
	require.NoError(t, h.registry.Refresh(context.Background()))

	code, env := doMatch(t, h, "productClass=QRTB-9000")

	assert.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 1, env["ret"])
	data, ok := env["data"].(map[string]any)
	require.True(t, ok, "data 应为对象，实际 %T", env["data"])
	assert.Equal(t, true, data["matched"], "命中返回 matched=true")
	assert.Equal(t, "^QRTB", data["matched_pattern"])
	assert.Equal(t, "QRTB-9000", data["product_class"])
	prod, ok := data["product"].(map[string]any)
	require.True(t, ok, "命中应带 product 对象，实际 %T", data["product"])
	assert.Equal(t, pid.String(), prod["id"])
}

func TestMatch_MissingProductClass_Returns400(t *testing.T) {
	h, _ := newMatchTestHandler(t)
	code, env := doMatch(t, h, "")

	assert.Equal(t, http.StatusBadRequest, code, "缺 productClass 参数应 400")
	assert.EqualValues(t, 0, env["ret"], "信封 ret=0（失败）")
}
