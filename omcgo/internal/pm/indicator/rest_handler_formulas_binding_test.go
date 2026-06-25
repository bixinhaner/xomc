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

// TestRESTCreateIndicator_Formulas_Binding 覆盖 CreateIndicatorRequest.Formulas（issue #640 C 方案）
// 新加的 binding:"omitempty,dive" + FormulaInput 元素 platform_name/formula 各自 required 校验。
//
// 测试策略与 statis_type 测试同源：
//   - 用空 service（svc.indicatorRepo == nil）注册 RESTHandler + Recovery；
//   - bind 通过的请求会进入 service → 因依赖空触发 panic/error → 被 Recovery 捕获返回 500；
//   - bind 阶段拦下的返回 400 + 错误信息含字段名（PlatformName / Formula）+ 约束（required）。
//
// 覆盖：未传 formulas（omitempty）/ 多条合法 / 空字符串 platform_name / 空字符串 formula / 数组非数组类型。
func TestRESTCreateIndicator_Formulas_Binding(t *testing.T) {
	cases := []struct {
		name         string
		formulasJSON string // 完整 json 片段如 `,"formulas":[...]` 或空串
		expectBind   bool   // true=binding 通过；false=binding 阶段被拦
		expectField  string // 非法时错误信息应包含的字段名（PlatformName / Formula）
	}{
		{
			name:         "omitted_compatible_with_old_clients",
			formulasJSON: "",
			expectBind:   true,
		},
		{
			name:         "single_valid_formula",
			formulasJSON: `,"formulas":[{"platform_name":"ALL","formula":"A+B"}]`,
			expectBind:   true,
		},
		{
			name:         "multiple_valid_formulas",
			formulasJSON: `,"formulas":[{"platform_name":"ALL","formula":"A+B"},{"platform_name":"BSC6900","formula":"C/D"}]`,
			expectBind:   true,
		},
		{
			name:         "empty_array_omitempty_allows",
			formulasJSON: `,"formulas":[]`,
			expectBind:   true,
		},
		{
			name:         "empty_platform_name_rejected",
			formulasJSON: `,"formulas":[{"platform_name":"","formula":"A+B"}]`,
			expectBind:   false,
			expectField:  "PlatformName",
		},
		{
			name:         "missing_formula_rejected",
			formulasJSON: `,"formulas":[{"platform_name":"ALL","formula":""}]`,
			expectBind:   false,
			expectField:  "Formula",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(gin.Recovery())
			h := NewRESTHandler(&IndicatorManagementService{}, nil, zap.NewNop())
			h.RegisterRoutes(r.Group("/api/v1"))

			body := `{"en_name":"e","cn_name":"c","group_id":"g"` + tc.formulasJSON + `}`
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost,
				"/api/v1/indicators?deviceType=enb",
				strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if tc.expectBind {
				assert.NotEqual(t, http.StatusBadRequest, w.Code,
					"合法 formulas 不应被 binding 拦 400; case=%s code=%d body=%s",
					tc.name, w.Code, w.Body.String())
			} else {
				require.Equal(t, http.StatusBadRequest, w.Code,
					"非法 formulas 应被 binding 拦 400; case=%s body=%s",
					tc.name, w.Body.String())
				assert.Contains(t, w.Body.String(), tc.expectField,
					"非法值错误信息应提及字段 %q; body=%s", tc.expectField, w.Body.String())
				assert.Contains(t, w.Body.String(), "required",
					"非法值错误信息应提及 required 约束; body=%s", w.Body.String())
			}
		})
	}
}
