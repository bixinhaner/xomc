package response

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

// stubTZProvider 单测桩：恒返回固定时区。
type stubTZProvider struct{ loc *time.Location }

func (s stubTZProvider) Location(context.Context) *time.Location { return s.loc }

func mustLoadTokyo(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	return loc
}

// withProvider 设置时区 Provider，并在用例结束后复位（避免污染其他用例）。
func withProvider(t *testing.T, p TimezoneProvider) {
	t.Helper()
	SetTimezoneProvider(p)
	t.Cleanup(func() { SetTimezoneProvider(nil) })
}

// 固定 UTC 时刻：2026-06-16 01:00:00Z，转 Tokyo 应为 2026-06-16 10:00:00+09:00（同一 instant）。
func fixedUTC() time.Time {
	return time.Date(2026, 6, 16, 1, 0, 0, 0, time.UTC)
}

type inner struct {
	CreatedAt time.Time `json:"created_at"`
	Label     string    `json:"label"`
}

type outer struct {
	ID    string `json:"id"`
	Inner inner  `json:"inner"`
}

// 1) 嵌套结构：内层时间字段被转到系统时区，instant 不变。
func TestConvertTimezone_NestedStruct(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	body := outer{ID: "x", Inner: inner{CreatedAt: fixedUTC(), Label: "n1"}}
	out := convertResponseTimezone(nil, body)

	m, ok := out.(map[string]any)
	require.True(t, ok, "结构体应被重建为 map")
	im, ok := m["inner"].(map[string]any)
	require.True(t, ok)
	gotT, ok := im["created_at"].(time.Time)
	require.True(t, ok)
	assert.True(t, gotT.Equal(fixedUTC()), "instant 必须不变")
	_, offsetSec := gotT.Zone()
	assert.Equal(t, 9*3600, offsetSec, "应为 +09:00 偏移")
	assert.Equal(t, "n1", im["label"])
}

// 2) 数组：切片内每个元素的时间字段都被转换。
func TestConvertTimezone_Slice(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	body := []inner{
		{CreatedAt: fixedUTC(), Label: "a"},
		{CreatedAt: fixedUTC().Add(time.Hour), Label: "b"},
	}
	out := convertResponseTimezone(nil, body)
	arr, ok := out.([]any)
	require.True(t, ok)
	require.Len(t, arr, 2)
	for _, e := range arr {
		em := e.(map[string]any)
		gotT := em["created_at"].(time.Time)
		_, off := gotT.Zone()
		assert.Equal(t, 9*3600, off)
	}
}

// 3) 零值时间：绝不转出脏偏移，保持零值原样。
func TestConvertTimezone_ZeroTime(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	body := inner{CreatedAt: time.Time{}, Label: "z"}
	out := convertResponseTimezone(nil, body)
	m := out.(map[string]any)
	gotT := m["created_at"].(time.Time)
	assert.True(t, gotT.IsZero(), "零值时间必须保持零值")
	_, off := gotT.Zone()
	assert.Equal(t, 0, off, "零值不得带非零偏移")
}

// 4) 北向跳过：上下文标记 northbound 时整体不转换，保持 UTC 原样。
func TestConvertTimezone_NorthboundSkip(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/northbound/sync/full", nil)
	c.Set(ContextKeyNorthbound, true)

	body := inner{CreatedAt: fixedUTC(), Label: "nb"}
	out := convertResponseTimezone(c, body)
	// 北向：原样返回入参（同一结构体值，未被重建为 map）。
	got, ok := out.(inner)
	require.True(t, ok, "北向响应应原样返回，不重建")
	_, off := got.CreatedAt.Zone()
	assert.Equal(t, 0, off, "北向时间必须保持 UTC")
}

// model.Time 同一响应内与标准 time.Time 一起被转换。
func TestConvertTimezone_ModelTime(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	type withModelTime struct {
		Std time.Time  `json:"std"`
		Mdl model.Time `json:"mdl"`
		Zer model.Time `json:"zer"`
	}
	body := withModelTime{Std: fixedUTC(), Mdl: model.Time(fixedUTC())}
	out := convertResponseTimezone(nil, body)
	m := out.(map[string]any)

	gotStd := m["std"].(time.Time)
	_, off := gotStd.Zone()
	assert.Equal(t, 9*3600, off)

	gotMdl := m["mdl"].(model.Time)
	assert.True(t, gotMdl.Std().Equal(fixedUTC()), "model.Time instant 不变")
	_, mOff := gotMdl.Std().Zone()
	assert.Equal(t, 9*3600, mOff, "model.Time 应被转到系统时区")

	gotZer := m["zer"].(model.Time)
	assert.True(t, gotZer.IsZero(), "零值 model.Time 保持零值")
}

// 未注入 Provider 时不转换，原样返回（默认安全：UTC）。
func TestConvertTimezone_NoProvider(t *testing.T) {
	SetTimezoneProvider(nil)
	body := inner{CreatedAt: fixedUTC(), Label: "p"}
	out := convertResponseTimezone(nil, body)
	got, ok := out.(inner)
	require.True(t, ok, "无 Provider 时原样返回")
	assert.True(t, got.CreatedAt.Equal(fixedUTC()))
}

// 类型短路：不含任何时间字段的类型直接原样返回（不重建为 map）。
func TestConvertTimezone_ShortCircuitNoTimeField(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	type noTime struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	body := []noTime{{Name: "a", Count: 1}, {Name: "b", Count: 2}}
	out := convertResponseTimezone(nil, body)
	// 短路：原样返回同一切片（未被重建为 []any）。
	got, ok := out.([]noTime)
	require.True(t, ok, "无时间字段的切片应原样返回")
	assert.Len(t, got, 2)
}

// 回归（issue #457 回合2）：开启系统时区后，与时间字段同居一结构体的 omitempty 字段
// 仍须像 encoding/json 那样被省略——绝不能从「缺失」变成 null / 0 / "" / false。
// 这一条直接复刻检查方报的阻塞缺陷场景（Device 模型：last_inform_at 时间字段 + 多个 omitempty 字段）。
func TestConvertTimezone_OmitEmptyPreserved(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	type deviceLike struct {
		Name       string     `json:"name"`
		LastInform *time.Time `json:"last_inform_at,omitempty"` // 空指针：应省略
		ProductID  string     `json:"product_id,omitempty"`     // 空串：应省略
		Status     int        `json:"status,omitempty"`         // 0：应省略
		Lat        float64    `json:"latitude,omitempty"`       // 0：应省略
		Enabled    bool       `json:"enabled,omitempty"`        // false：应省略
		Tags       []string   `json:"tags,omitempty"`           // nil slice：应省略
		Extra      map[string]any `json:"extra,omitempty"`      // nil map：应省略
		CreatedAt  time.Time  `json:"created_at"`               // 时间字段：触发结构体重建路径
	}

	body := deviceLike{Name: "dev1", CreatedAt: fixedUTC()}
	out := convertResponseTimezone(nil, body)
	m, ok := out.(map[string]any)
	require.True(t, ok, "含时间字段的结构体应被重建为 map")

	// 时间字段仍被转换为带偏移。
	gotT, ok := m["created_at"].(time.Time)
	require.True(t, ok)
	_, off := gotT.Zone()
	assert.Equal(t, 9*3600, off)

	// 关键断言：所有 omitempty 空值字段必须缺席，不能出现为 null/0/""/false。
	for _, k := range []string{"last_inform_at", "product_id", "status", "latitude", "enabled", "tags", "extra"} {
		_, present := m[k]
		assert.Falsef(t, present, "omitempty 空值字段 %q 应被省略，实际出现于输出", k)
	}
	// 非 omitempty 字段照常存在。
	assert.Equal(t, "dev1", m["name"])
}

// 回归：omitempty 字段在「非空」时仍正常输出（不被误省），且其中的时间值照常转换。
func TestConvertTimezone_OmitEmptyNonEmptyKept(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	ts := fixedUTC()
	type withPtr struct {
		Name       string     `json:"name"`
		LastInform *time.Time `json:"last_inform_at,omitempty"`
		ProductID  string     `json:"product_id,omitempty"`
	}
	body := withPtr{Name: "dev2", LastInform: &ts, ProductID: "p-1"}
	out := convertResponseTimezone(nil, body)
	m := out.(map[string]any)

	require.Contains(t, m, "last_inform_at", "非空 omitempty 指针字段应保留")
	gotT, ok := m["last_inform_at"].(time.Time)
	require.True(t, ok, "指针被解引用为 time.Time 并转换")
	_, off := gotT.Zone()
	assert.Equal(t, 9*3600, off, "非空 omitempty 时间字段也要转到系统时区")
	assert.Equal(t, "p-1", m["product_id"])
}

// 端到端（issue #457 回合2）：经 renderJSON 的真实信封里，omitempty 字段在开启时区后仍被省略。
func TestRenderJSON_OmitEmptyPreservedEndToEnd(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)

	type deviceLike struct {
		Name       string     `json:"name"`
		LastInform *time.Time `json:"last_inform_at,omitempty"`
		Status     int        `json:"status,omitempty"`
		CreatedAt  time.Time  `json:"created_at"`
	}
	OK(c, deviceLike{Name: "dev1", CreatedAt: fixedUTC()})

	var env struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	_, hasLastInform := env.Data["last_inform_at"]
	_, hasStatus := env.Data["status"]
	assert.False(t, hasLastInform, "开启时区后 last_inform_at(omitempty,nil) 仍应省略")
	assert.False(t, hasStatus, "开启时区后 status(omitempty,0) 仍应省略")
	createdAt, _ := env.Data["created_at"].(string)
	assert.Contains(t, createdAt, "+09:00")
}

// 端到端：经 renderJSON（OK）的成功响应，时间字段带系统时区偏移。
func TestRenderJSON_TimezoneEndToEnd(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)

	OK(c, inner{CreatedAt: fixedUTC(), Label: "e2e"})

	var env struct {
		Data struct {
			CreatedAt string `json:"created_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Data.CreatedAt, "+09:00", "经统一出口应带 +09:00 偏移")
}

// 端到端：北向响应经 renderJSON 仍输出 UTC（Z）。
func TestRenderJSON_NorthboundEndToEnd(t *testing.T) {
	tokyo := mustLoadTokyo(t)
	withProvider(t, stubTZProvider{loc: tokyo})

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/northbound/sync/full", nil)
	c.Set(ContextKeyNorthbound, true)

	OK(c, inner{CreatedAt: fixedUTC(), Label: "nb"})

	var env struct {
		Data struct {
			CreatedAt string `json:"created_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.True(t, env.Data.CreatedAt == "2026-06-16T01:00:00Z",
		"北向应保持 UTC，实际=%s", env.Data.CreatedAt)
}
