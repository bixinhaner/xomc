package adhoc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-0182：建任务校验单测。
//
// 这些 case 全部在 handler 进 repo / DB 之前的纯校验阶段命中，故用 nil pool 的 stub repo 即可：
//   - granularities len != 1 → 400（成功 = 恰好 1 个；失败 = 0 个 / 多个）
//   - technology DTO binding oneof=lte nr gsm → 非法制式值 400（建任务拒非法制式入口）
//
// 跨制式（设备制式与任务制式不一致）的"全链路拒绝"需真实 devices 表查询（rejectCrossTechnology
// 走 h.pool），属集成/真机验证范畴；本单测覆盖到 DTO 制式合法性这层入口校验。

func postCreate(t *testing.T, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	r := newTestRouter(&handlerStubRepo{})
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/adhoc/tasks", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func baseCreateBody() map[string]any {
	return map[string]any{
		"name": "t", "mode": "oneshot",
		"device_sns":   []string{"S1"},
		"metric_paths": []string{"M1"},
		"window_start": "2026-05-22T10:00:00Z",
		"window_end":   "2026-05-22T11:00:00Z",
	}
}

// 成功路径：granularities 恰好 1 个 → 201。
func Test_Handler_Create_SingleGranularity_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 失败路径：granularities 多个 → 400（设计 §2.4/§2.6 单粒度）。
func Test_Handler_Create_MultiGranularity_Rejected(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly", "daily"}
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 失败路径：granularities 为空 → 400（binding required,min=1 命中）。
func Test_Handler_Create_EmptyGranularity_Rejected(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{}
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 成功路径：technology 合法值（lte）+ 单粒度 → 201（无 pool 时跨制式校验因 device_sns 仍会触发，
// 这里用空 device_sns 以隔离到 DTO 校验层；建任务允许空 device_sns by binding? device_sns required,min=1）。
//
// 注意：technology != "" 且 device_sns 非空时 handler 会调 rejectCrossTechnology（需 pool）。
// 为隔离 DTO 制式合法性校验，这里只断言"非法制式值在 bind 阶段即被拒"，合法值的全链路放真机验证。

// 失败路径：technology 非法值（5g）→ 400（DTO binding oneof=lte nr gsm 命中，建任务拒非法制式）。
func Test_Handler_Create_InvalidTechnology_Rejected(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["technology"] = "5g" // 非 lte/nr/gsm
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 失败路径：dimension 非法值 → 400（DTO binding oneof=device aggregate_group product band 命中）。
func Test_Handler_Create_InvalidDimension_Rejected(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "site" // 非合法维度
	w := postCreate(t, b)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 成功路径：dimension=product 合法 → 201（product 已入枚举，建任务可选）。
func Test_Handler_Create_ProductDimension_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "product"
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 成功路径：dimension=network 合法 → 201（T-0184 全网维度已入 binding oneof）。
func Test_Handler_Create_NetworkDimension_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "network"
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 成功路径：dimension=device_group 合法 → 201（T-0184 设备组维度已入 binding oneof）。
func Test_Handler_Create_DeviceGroupDimension_OK(t *testing.T) {
	b := baseCreateBody()
	b["granularities"] = []string{"hourly"}
	b["dimension"] = "device_group"
	w := postCreate(t, b)
	assert.Equal(t, http.StatusCreated, w.Code)
}
