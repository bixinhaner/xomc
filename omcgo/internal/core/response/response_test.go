package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
