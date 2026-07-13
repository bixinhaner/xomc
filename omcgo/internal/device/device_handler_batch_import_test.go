package device

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 批量导入语义（产品决策）：按 SN 更新【已注册】设备的 名称/备注，并把命中的 SN
// 划入当前所选分组；SN 不存在的行失败（不新建设备）。下列用例围绕这一语义。

func sp(s string) *string { return &s }

// decodeBatchImportResp 把统一信封中的 BatchImportResponse 解出来。
func decodeBatchImportResp(t *testing.T, w *httptest.ResponseRecorder) BatchImportResponse {
	t.Helper()
	var raw map[string]any
	response.DecodeData(t, w.Body, &raw)
	b, err := json.Marshal(raw)
	require.NoError(t, err)
	var resp BatchImportResponse
	require.NoError(t, json.Unmarshal(b, &resp))
	return resp
}

func postBatchImport(t *testing.T, router http.Handler, body any) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/batch-import",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

// 全部 SN 已注册 → 全部成功（按 SN 更新名称）。
func TestHandler_BatchImportDevices_UpdateExistingBySN(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	seedDevice(deviceRepo, uuid.New(), "SN-IMP-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)
	seedDevice(deviceRepo, uuid.New(), "SN-IMP-002", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	body := BatchImportRequest{
		Devices: []BatchImportDeviceRow{
			{SerialNumber: "SN-IMP-001", DeviceName: sp("北京中关村站")},
			{SerialNumber: "SN-IMP-002", DeviceName: sp("北京CBD站")},
		},
	}

	w := postBatchImport(t, router, body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchImportResp(t, w)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 2, resp.Succeeded)
	assert.Equal(t, 0, resp.Failed)
	assert.Empty(t, resp.Errors)
	// 名称已写回
	assert.Equal(t, "北京中关村站", deviceRepo.bySN["SN-IMP-001"].DeviceName)
}

// 部分 SN 不存在 → 该行失败（设备不存在），其余成功。
func TestHandler_BatchImportDevices_MissingSNFails(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	seedDevice(deviceRepo, uuid.New(), "SN-EXIST", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	body := BatchImportRequest{
		Devices: []BatchImportDeviceRow{
			{SerialNumber: "SN-EXIST", DeviceName: sp("已注册站")},
			{SerialNumber: "SN-MISSING", DeviceName: sp("未注册站")},
		},
	}

	w := postBatchImport(t, router, body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchImportResp(t, w)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 1, resp.Succeeded)
	assert.Equal(t, 1, resp.Failed)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, 2, resp.Errors[0].Row)
	assert.Equal(t, "SN-MISSING", resp.Errors[0].SN)
	assert.Equal(t, "device_not_found", resp.Errors[0].ErrorCode,
		"未注册 SN 应返回 error_code=device_not_found")
	assert.True(t, strings.Contains(resp.Errors[0].Reason, "device not found"),
		"未注册 SN reason 应为英文，实际：%s", resp.Errors[0].Reason)
}

// 全部 SN 未注册 → 全部失败。
func TestHandler_BatchImportDevices_AllNotFound(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	body := BatchImportRequest{
		Devices: []BatchImportDeviceRow{
			{SerialNumber: "SN-X-1", DeviceName: sp("a")},
			{SerialNumber: "SN-X-2", DeviceName: sp("b")},
		},
	}

	w := postBatchImport(t, router, body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchImportResp(t, w)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 0, resp.Succeeded)
	assert.Equal(t, 2, resp.Failed)
	require.Len(t, resp.Errors, 2)
}

// 空 devices → binding:"required,min=1" 在 ShouldBindJSON 阶段 400。
func TestHandler_BatchImportDevices_EmptyRejected(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	w := postBatchImport(t, router, BatchImportRequest{Devices: nil})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 缺 serial_number → dive 元素 binding:"required" 拦截 → 400。
func TestHandler_BatchImportDevices_MissingSerialRejected(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	body := map[string]any{
		"devices": []map[string]any{
			{"device_name": "无 SN 的行", "remark": "x"},
		},
	}

	w := postBatchImport(t, router, body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
