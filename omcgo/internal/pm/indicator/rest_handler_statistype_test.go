package indicator

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestRESTCreateIndicator_StatisType_Enum 覆盖 CreateIndicatorRequest.StatisType 新加的
// binding:"omitempty,oneof=sum avg max min pct" 枚举校验。
//
// 测试策略：
//   - 用空 service（svc.indicatorRepo == nil）注册 RESTHandler，并启用 gin.Recovery()；
//   - 合法 statis_type 通过 bind 后进入 service.CreateIndicator → 因依赖未注入触发
//     panic/error，被 Recovery 捕获返回 500（非 400 即说明 bind 校验放行）；
//   - 非法 statis_type 在 bind 阶段被 oneof 拦下 → 400，错误信息包含 "StatisType" 字段名 + "oneof"；
//   - 关键断言用 HTTP code（400 vs 非 400）+ 响应体关键字区分。
//
// 同时验证 UpdateIndicatorRequest.StatisType 走 PUT 时也享受同样的 omitempty,oneof 守卫。
func TestRESTCreateIndicator_StatisType_Enum(t *testing.T) {
	cases := []struct {
		name       string
		statisType string
		expectBind bool // true=binding 通过；false=binding 阶段被拦
	}{
		{"valid_sum", "sum", true},
		{"valid_avg", "avg", true},
		{"valid_max", "max", true},
		{"valid_min", "min", true},
		{"valid_pct", "pct", true},
		{"valid_omitted", "", true},
		{"invalid_median", "median", false},
		{"invalid_uppercase_SUM", "SUM", false},
		{"invalid_random_xxx", "xxx", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(gin.Recovery())
			h := NewRESTHandler(&IndicatorManagementService{}, nil, zap.NewNop())
			h.RegisterRoutes(r.Group("/api/v1"))

			body := `{"en_name":"e","cn_name":"c","group_id":"g"`
			if tc.statisType != "" {
				body += `,"statis_type":"` + tc.statisType + `"`
			}
			body += `}`
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost,
				"/api/v1/indicators?deviceType=enb",
				strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if tc.expectBind {
				assert.NotEqual(t, http.StatusBadRequest, w.Code,
					"合法 statis_type=%q 不应被 binding 拦截 400; code=%d body=%s",
					tc.statisType, w.Code, w.Body.String())
				assert.NotContains(t, w.Body.String(), "StatisType",
					"合法值响应不应提到 StatisType 字段名; body=%s", w.Body.String())
			} else {
				require.Equal(t, http.StatusBadRequest, w.Code,
					"非法 statis_type=%q 应被 binding 拦截 400; body=%s",
					tc.statisType, w.Body.String())
				assert.Contains(t, w.Body.String(), "StatisType",
					"非法值错误信息应提及 StatisType; body=%s", w.Body.String())
				assert.Contains(t, w.Body.String(), "oneof",
					"非法值错误信息应提及 oneof 约束; body=%s", w.Body.String())
			}
		})
	}
}

// TestRESTUpdateIndicator_StatisType_Enum 对应 UpdateIndicatorRequest.StatisType 的
// omitempty,oneof 守卫；策略同 Create。
func TestRESTUpdateIndicator_StatisType_Enum(t *testing.T) {
	cases := []struct {
		name       string
		statisType string
		expectBind bool
	}{
		{"valid_pct", "pct", true},
		{"valid_min", "min", true},
		{"omitted_body", "", true},
		{"invalid_median", "median", false},
		{"invalid_zh", "求和", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(gin.Recovery())
			h := NewRESTHandler(&IndicatorManagementService{}, nil, zap.NewNop())
			h.RegisterRoutes(r.Group("/api/v1"))

			body := `{}`
			if tc.statisType != "" {
				body = `{"statis_type":"` + tc.statisType + `"}`
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut,
				"/api/v1/indicators/some-id?deviceType=enb",
				strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if tc.expectBind {
				assert.NotEqual(t, http.StatusBadRequest, w.Code,
					"合法 statis_type=%q 不应被 PUT binding 拦截 400; code=%d body=%s",
					tc.statisType, w.Code, w.Body.String())
			} else {
				require.Equal(t, http.StatusBadRequest, w.Code,
					"非法 statis_type=%q 应被 PUT binding 拦截 400; body=%s",
					tc.statisType, w.Body.String())
				assert.Contains(t, w.Body.String(), "StatisType")
				assert.Contains(t, w.Body.String(), "oneof")
			}
		})
	}
}
