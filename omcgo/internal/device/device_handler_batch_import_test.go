package device

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeBatchImportResp 把统一信封中的 BatchImportResponse 解出来。
// response.DecodeData 把 data 节点 marshal 后再 unmarshal 进 v，
// 这里用 map → JSON 来回的方式拿到 typed struct，避免 BatchImportResponse 上
// 手动加 mapstructure tag 的额外耦合。
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

func TestHandler_BatchImportDevices_AllSuccess(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	body := BatchImportRequest{
		Devices: []CreateDeviceRequest{
			{
				SerialNumber: "SN-IMP-001",
				OUI:          "001E91",
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechLTE,
				ProductClass: "X100W",
				DeviceName:   "Site-A",
			},
			{
				SerialNumber: "SN-IMP-002",
				OUI:          "001E91",
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechLTE,
				DeviceName:   "Site-B",
			},
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/batch-import",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchImportResp(t, w)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 2, resp.Succeeded)
	assert.Equal(t, 0, resp.Failed)
	assert.Empty(t, resp.Errors)
}

func TestHandler_BatchImportDevices_PartialFailure(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	// 预置一个 SN 制造冲突
	seedDevice(deviceRepo, uuid.New(), "SN-DUP-X", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	body := BatchImportRequest{
		Devices: []CreateDeviceRequest{
			{
				SerialNumber: "SN-NEW-1",
				OUI:          "001E91",
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechLTE,
			},
			{
				// 与 seed 冲突
				SerialNumber: "SN-DUP-X",
				OUI:          "001E91",
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechLTE,
			},
			{
				SerialNumber: "SN-NEW-2",
				OUI:          "001E91",
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechLTE,
			},
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/batch-import",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchImportResp(t, w)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.Succeeded)
	assert.Equal(t, 1, resp.Failed)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, 2, resp.Errors[0].Row)
	assert.Equal(t, "SN-DUP-X", resp.Errors[0].SN)
	assert.Equal(t, "SN 已存在", resp.Errors[0].Reason)
}

func TestHandler_BatchImportDevices_AllFailure(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	seedDevice(deviceRepo, uuid.New(), "SN-X-1", model.CarrierCMCC, model.TechLTE, model.DeviceActive)
	seedDevice(deviceRepo, uuid.New(), "SN-X-2", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	body := BatchImportRequest{
		Devices: []CreateDeviceRequest{
			{SerialNumber: "SN-X-1", OUI: "001E91", Carrier: model.CarrierCMCC, Technology: model.TechLTE},
			{SerialNumber: "SN-X-2", OUI: "001E91", Carrier: model.CarrierCMCC, Technology: model.TechLTE},
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/batch-import",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchImportResp(t, w)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 0, resp.Succeeded)
	assert.Equal(t, 2, resp.Failed)
	require.Len(t, resp.Errors, 2)
}

func TestHandler_BatchImportDevices_EmptyRejected(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	// binding:"required,min=1" → 应在 ShouldBindJSON 阶段 400
	body := BatchImportRequest{Devices: nil}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/batch-import",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BatchImportDevices_BadEnumRejected(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	// dive 会走每个元素的 binding，oneof=cmcc ctcc cucc 拦掉非法 carrier
	body := map[string]any{
		"devices": []map[string]any{
			{
				"serial_number": "SN-BAD-1",
				"oui":           "001E91",
				"carrier":       "verizon", // 非法
				"technology":    "lte",
			},
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/batch-import",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
