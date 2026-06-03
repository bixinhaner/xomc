package adhoc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/core/middleware"
)

// Test_taskToDTO_HTTPLocale 端到端验证：真实 Locale 中间件挂在 gin 上后，
// 一个内置任务经 taskToDTO 出来的 Name 是否随请求 Accept-Language 头本地化。
// 复现「英文态内置任务名仍中文」的疑似 bug —— 用服务端 httptest（curl 不受
// 浏览器 forbidden-header 限制，可真正发出 Accept-Language），排除部署/nginx/
// 浏览器丢头等干扰，只测中间件 → ctx → taskToDTO → localizeTaskName 这条链。
func Test_taskToDTO_HTTPLocale(t *testing.T) {
	gin.SetMode(gin.TestMode)

	builtin := &Task{
		Name:       "内置-设备组-LTE",
		IsBuiltin:  true,
		Dimension:  DimensionDeviceGroup,
		Technology: "lte",
	}

	r := gin.New()
	r.Use(middleware.Locale())
	r.GET("/t", func(c *gin.Context) {
		c.JSON(http.StatusOK, taskToDTO(c.Request.Context(), builtin))
	})

	cases := []struct {
		acceptLang string
		wantName   string
	}{
		{"en-US,en;q=0.9", "Built-in-Device Group-LTE"},
		{"zh-CN,zh;q=0.9", "内置-设备组-LTE"},
		{"", "内置-设备组-LTE"},
	}

	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		if c.acceptLang != "" {
			req.Header.Set("Accept-Language", c.acceptLang)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Accept-Language=%q: status=%d", c.acceptLang, w.Code)
		}
		var dto taskResponseDTO
		if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
			t.Fatalf("Accept-Language=%q: unmarshal: %v", c.acceptLang, err)
		}
		if dto.Name != c.wantName {
			t.Errorf("Accept-Language=%q: Name=%q, want %q", c.acceptLang, dto.Name, c.wantName)
		}
	}
}
