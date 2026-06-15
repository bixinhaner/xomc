package response

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/jsonx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newCtx 构造一个绑定到 ResponseRecorder 的 gin.Context，供 helper 写入。
func newCtx() (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, rec
}

// decode 解出响应 body 的信封字段，断言用。
func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

// --- 成功路径 ---

func TestOK(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{name: "带数据", data: map[string]any{"k": "v"}},
		{name: "nil 数据（纯写操作）", data: nil},
		{name: "切片数据", data: []string{"a", "b"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newCtx()
			OK(c, tc.data)

			assert.Equal(t, http.StatusOK, rec.Code)
			body := decode(t, rec)
			assert.EqualValues(t, 1, body["ret"])
			assert.Equal(t, MsgOK, body["msg"])
			// data 键必须存在（即使为 nil），契约要求失败时固定 null、成功时回传业务节点。
			_, hasData := body["data"]
			assert.True(t, hasData, "data 键必须始终存在")
		})
	}
}

func TestOKWithStatus(t *testing.T) {
	c, rec := newCtx()
	OKWithStatus(c, http.StatusCreated, map[string]any{"id": "u-1"})

	assert.Equal(t, http.StatusCreated, rec.Code)
	body := decode(t, rec)
	assert.EqualValues(t, 1, body["ret"])
	assert.Equal(t, MsgOK, body["msg"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "u-1", data["id"])
}

func TestOKWithMsg(t *testing.T) {
	c, rec := newCtx()
	OKWithMsg(c, nil, "创建成功")

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	assert.EqualValues(t, 1, body["ret"])
	assert.Equal(t, "创建成功", body["msg"])
	assert.Nil(t, body["data"])
}

// TestOK_NonFiniteFloatPayload 回归 issue #387：响应体含非有限浮点（NaN/Inf）时，
// 用 jsonx.Float 兜底后 Gin 仍应输出 HTTP 200 + 完整非空 body，且该字段为 null。
// 修复前：encoding/json 遇 NaN 报错、Gin 已写 200 头后中断 → body 为空。
func TestOK_NonFiniteFloatPayload(t *testing.T) {
	type row struct {
		Path  string      `json:"metric_path"`
		Value jsonx.Float `json:"metric_value"`
	}
	// 含全部三类非有限值 + 正常有限值，模拟真实聚合结果集。
	items := []row{
		{Path: "C1", Value: jsonx.Float(100)},
		{Path: "C2", Value: jsonx.Float(math.NaN())},
		{Path: "C3", Value: jsonx.Float(math.Inf(1))},
		{Path: "C4", Value: jsonx.Float(math.Inf(-1))},
		{Path: "C5", Value: jsonx.Float(3.5)},
	}

	c, rec := newCtx()
	OK(c, gin.H{"items": items})

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotZero(t, rec.Body.Len(), "body 不得为空（修复前含 NaN 即返回 0 字节）")

	body := decode(t, rec)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	got, ok := data["items"].([]any)
	require.True(t, ok)
	require.Len(t, got, 5)

	first := got[0].(map[string]any)
	assert.EqualValues(t, 100, first["metric_value"])

	for _, idx := range []int{1, 2, 3} {
		r := got[idx].(map[string]any)
		v, has := r["metric_value"]
		assert.True(t, has, "字段应存在")
		assert.Nil(t, v, "非有限值应呈现为 null")
	}

	last := got[4].(map[string]any)
	assert.EqualValues(t, 3.5, last["metric_value"])
}

// TestOK_AllFiniteFloatPayload 正常路径：全为有限值时返回完整 JSON，数值原样保留。
func TestOK_AllFiniteFloatPayload(t *testing.T) {
	type row struct {
		Path  string      `json:"metric_path"`
		Value jsonx.Float `json:"metric_value"`
	}
	items := []row{
		{Path: "C1", Value: jsonx.Float(1.25)},
		{Path: "C2", Value: jsonx.Float(0)},
	}

	c, rec := newCtx()
	OK(c, gin.H{"items": items})

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotZero(t, rec.Body.Len())

	body := decode(t, rec)
	data := body["data"].(map[string]any)
	got := data["items"].([]any)
	require.Len(t, got, 2)
	assert.EqualValues(t, 1.25, got[0].(map[string]any)["metric_value"])
	assert.EqualValues(t, 0, got[1].(map[string]any)["metric_value"])
}

// --- 失败路径 ---

func TestFail(t *testing.T) {
	c, rec := newCtx()
	Fail(c, http.StatusBadRequest, "参数非法")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.True(t, c.IsAborted(), "Fail 必须中止后续 handler")
	body := decode(t, rec)
	assert.EqualValues(t, 0, body["ret"])
	assert.Equal(t, "参数非法", body["msg"])
	assert.Nil(t, body["data"])
	_, hasBiz := body["biz_code"]
	assert.False(t, hasBiz, "Fail 不应带 biz_code")
}

func TestFailWithBizCode(t *testing.T) {
	tests := []struct {
		name       string
		bizCode    int
		wantBizKey bool
	}{
		{name: "非零业务码写入 biz_code", bizCode: 2030, wantBizKey: true},
		{name: "零业务码省略 biz_code", bizCode: 0, wantBizKey: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newCtx()
			FailWithBizCode(c, http.StatusConflict, "重复", tc.bizCode)

			assert.Equal(t, http.StatusConflict, rec.Code)
			assert.True(t, c.IsAborted())
			body := decode(t, rec)
			assert.EqualValues(t, 0, body["ret"])
			assert.Equal(t, "重复", body["msg"])
			assert.Nil(t, body["data"])

			val, has := body["biz_code"]
			assert.Equal(t, tc.wantBizKey, has)
			if tc.wantBizKey {
				assert.EqualValues(t, tc.bizCode, val)
			}
		})
	}
}

func TestFailWithData(t *testing.T) {
	c, rec := newCtx()
	detail := []map[string]any{{"field": "name", "error": "required"}}
	FailWithData(c, http.StatusUnprocessableEntity, "校验失败", detail)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.True(t, c.IsAborted())
	body := decode(t, rec)
	assert.EqualValues(t, 0, body["ret"])
	assert.Equal(t, "校验失败", body["msg"])
	// 失败信封通常 data=null，但校验失败场景允许回传结构化明细。
	data, ok := body["data"].([]any)
	require.True(t, ok)
	assert.Len(t, data, 1)
}
