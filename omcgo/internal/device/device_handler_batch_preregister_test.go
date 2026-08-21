package device

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeBatchPreRegResp(t *testing.T, w *httptest.ResponseRecorder) BatchPreRegisterResponse {
	t.Helper()
	var raw map[string]any
	response.DecodeData(t, w.Body, &raw)
	b, err := json.Marshal(raw)
	require.NoError(t, err)
	var resp BatchPreRegisterResponse
	require.NoError(t, json.Unmarshal(b, &resp))
	return resp
}

func postBatchPreReg(t *testing.T, router interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, body BatchPreRegisterRequest) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/batch-preregister", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// 已存在的 SN → 更新名称/备注，计入 updated。
func TestHandler_BatchPreRegister_UpdateExisting(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)
	seedDevice(deviceRepo, uuid.New(), "SN-EXIST-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	name := "新名称"
	body := BatchPreRegisterRequest{
		Devices: []BatchPreRegisterRow{
			{SerialNumber: "SN-EXIST-001", DeviceName: &name},
		},
	}
	w := postBatchPreReg(t, router, body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchPreRegResp(t, w)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, 1, resp.Updated)
	assert.Equal(t, 0, resp.Created)
	assert.Equal(t, 0, resp.Failed)
}

// 不存在的 SN + carrier 显式指定 → 新建设备，计入 created。
func TestHandler_BatchPreRegister_CreateNew(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	name := "预登记基站"
	body := BatchPreRegisterRequest{
		Devices: []BatchPreRegisterRow{
			{
				SerialNumber: "SN-NEW-001",
				DeviceName:   &name,
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechLTE,
				OUI:          "120288",
			},
		},
	}
	w := postBatchPreReg(t, router, body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchPreRegResp(t, w)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, 1, resp.Created)
	assert.Equal(t, 0, resp.Failed)
}

// UPS 预登记可携带 ProductClass，占位设备在首次 Inform 前也能进入 UPS 识别口径。
func TestHandler_BatchPreRegister_UPSProductClass(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	name := "机房UPS-01"
	body := BatchPreRegisterRequest{
		Devices: []BatchPreRegisterRow{
			{
				SerialNumber: "UPS-SN-00001",
				DeviceName:   &name,
				ProductClass: "UPS_M3_BMU",
				Carrier:      model.CarrierCTCC,
				Technology:   model.TechLTE,
				OUI:          "ABCDEF",
			},
		},
	}
	w := postBatchPreReg(t, router, body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchPreRegResp(t, w)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, 1, resp.Created)
	assert.Equal(t, 0, resp.Failed)

	created, err := deviceRepo.GetBySerialNumber(t.Context(), "UPS-SN-00001")
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "UPS_M3_BMU", created.ProductClass)
	assert.Equal(t, name, created.DeviceName)
}

// carrier 无法推断（OUI 未知且 CSV 未填）→ error_code="carrier_required"，计入 failed。
func TestHandler_BatchPreRegister_CarrierRequired(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	body := BatchPreRegisterRequest{
		Devices: []BatchPreRegisterRow{
			// OUI 无法被 CarrierRegistry 识别，且 Carrier 为空
			{SerialNumber: "UNKNOWN-OUI-00001"},
		},
	}
	w := postBatchPreReg(t, router, body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchPreRegResp(t, w)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, 0, resp.Created)
	assert.Equal(t, 1, resp.Failed)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, "carrier_required", resp.Errors[0].ErrorCode)
}

func TestHandler_BatchPreRegister_SharedOUIRequiresExplicitCarrier(t *testing.T) {
	h, _, _ := newTestHandler()
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())
	registry.Register(ctcc.New())
	h.service.SetCarrierRegistry(registry)
	router := setupRouter(h)

	body := BatchPreRegisterRequest{
		Devices: []BatchPreRegisterRow{
			{SerialNumber: "SHARED-OUI-NO-CARRIER", OUI: "00E0FC"},
			{
				SerialNumber: "SHARED-OUI-EXPLICIT-CTCC",
				OUI:          "00E0FC",
				Carrier:      model.CarrierCTCC,
				Technology:   model.TechLTE,
			},
		},
	}
	w := postBatchPreReg(t, router, body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchPreRegResp(t, w)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 1, resp.Created)
	assert.Equal(t, 1, resp.Failed)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, "carrier_required", resp.Errors[0].ErrorCode)
}

// 混合行：已存在 + 新建 + OUI 推断失败 → 各自独立处理，互不阻断。
func TestHandler_BatchPreRegister_Mixed(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)
	seedDevice(deviceRepo, uuid.New(), "SN-MIXED-EXIST", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	name := "更新名"
	body := BatchPreRegisterRequest{
		Devices: []BatchPreRegisterRow{
			{SerialNumber: "SN-MIXED-EXIST", DeviceName: &name},                                                  // 已存在，updated
			{SerialNumber: "SN-MIXED-NEW", Carrier: model.CarrierCTCC, Technology: model.TechLTE, OUI: "ABCDEF"}, // 新建
			{SerialNumber: "UNKNOWN-OUI-999"},                                                                    // carrier 推断失败
		},
	}
	w := postBatchPreReg(t, router, body)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBatchPreRegResp(t, w)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 1, resp.Updated)
	assert.Equal(t, 1, resp.Created)
	assert.Equal(t, 1, resp.Failed)
}
