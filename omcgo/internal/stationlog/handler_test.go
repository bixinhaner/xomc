package stationlog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/device"
)

// newTestHandler 起一个完整的 gin 引擎挂载 stationlog 路由，并预置一条 detected
// 占位记录用于端点契约验证。
func newTestHandler(t *testing.T) (*gin.Engine, *Service, *memRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	runRepo := newMemRunRepo()
	faultRepo := newMemFaultRepo()
	svc := NewService(runRepo, faultRepo, stubDeviceLookup{}, nil, appconfig.BucketConfig{}, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())

	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterRoutes(api)
	return r, svc, faultRepo
}

func TestAbnormalReboot_ListAndGet(t *testing.T) {
	r, svc, faultRepo := newTestHandler(t)
	_ = faultRepo

	// 写两条记录：一条 detected、一条 file_received
	id1 := uuid.New()
	id2 := uuid.New()
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), device.AbnormalRebootSnapshot{
		DeviceID:       id1,
		DeviceSN:       "SN-A",
		HaltMainReason: "halt_reboot",
		DeviceType:     "eNB",
		DetectedAt:     time.Now(),
	}))
	// file_received 模拟（直接走旧路径插入）
	require.NoError(t, faultRepo.Create(context.Background(), &LogFile{
		DeviceID:     &id2,
		DeviceSN:     "SN-B",
		FileName:     "abnormalLog_SN-B.tar.gz",
		ObjectPath:   "fault/2026/abnormalLog_SN-B.tar.gz",
		Bucket:       "logs",
		FileSize:     2048,
		DeviceType:   "gNB",
		IsGNB:        true,
		RecordStatus: FaultRecordStatusFileReceived,
		FaultReason:  "halt_reboot",
		CollectedAt:  time.Now(),
	}))

	// 列表
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/device-abnormal-reboots", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []LogFile `json:"items"`
			Total int64     `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.Data.Total)
	assert.Len(t, resp.Data.Items, 2)

	// record_status 过滤
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/device-abnormal-reboots?record_status=detected", nil)
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
	if assert.Len(t, resp.Data.Items, 1) {
		assert.Equal(t, FaultRecordStatusDetected, resp.Data.Items[0].RecordStatus)
		assert.Equal(t, "SN-A", resp.Data.Items[0].DeviceSN)
	}

	// 详情 - 选用 detected 那条
	detectedID := resp.Data.Items[0].ID
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/v1/device-abnormal-reboots/"+detectedID.String(), nil)
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code)

	// 详情 - 不存在
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/api/v1/device-abnormal-reboots/"+uuid.New().String(), nil)
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusNotFound, w4.Code)

	// 详情 - bad UUID
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest("GET", "/api/v1/device-abnormal-reboots/not-a-uuid", nil)
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusBadRequest, w5.Code)
}

func TestAbnormalReboot_Delete(t *testing.T) {
	r, svc, faultRepo := newTestHandler(t)
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), device.AbnormalRebootSnapshot{
		DeviceID:       uuid.New(),
		DeviceSN:       "SN-DEL",
		HaltMainReason: "halt_reboot",
		DetectedAt:     time.Now(),
	}))
	id := faultRepo.rows[0].ID

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/device-abnormal-reboots/"+id.String(), nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, faultRepo.rows[0].IsDeleted)
}

func TestAbnormalReboot_DownloadDetected_Returns409(t *testing.T) {
	// detected 状态的记录还没有文件，下载应返回 409 而非 500。
	r, svc, faultRepo := newTestHandler(t)
	require.NoError(t, svc.RecordAbnormalReboot(context.Background(), device.AbnormalRebootSnapshot{
		DeviceID:       uuid.New(),
		DeviceSN:       "SN-NOFILE",
		HaltMainReason: "halt_reboot",
		DetectedAt:     time.Now(),
	}))
	id := faultRepo.rows[0].ID

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/device-abnormal-reboots/"+id.String()+"/download", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestDelete_NonexistentReturns404 锁定 #125 修复：对格式合法但不存在的 UUID，
// DELETE /station-logs/:id 与 DELETE /device-abnormal-reboots/:id 都必须返回 404
// （此前 service 返回裸 fmt.Errorf，handler 一律按 500 抛出）。
func TestDelete_NonexistentReturns404(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"station-logs", "/api/v1/station-logs/"},
		{"station-logs-fault", "/api/v1/station-logs/"}, // log_type=fault 走 faultRepo
		{"device-abnormal-reboots", "/api/v1/device-abnormal-reboots/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _, _ := newTestHandler(t)
			url := tc.path + uuid.New().String()
			if tc.name == "station-logs-fault" {
				url += "?log_type=fault"
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", url, nil)
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusNotFound, w.Code,
				"不存在 UUID 删除应返回 404，实际 %d body=%s", w.Code, w.Body.String())
		})
	}
}

// TestDelete_BadUUIDReturns400 负路径对照：格式非法的 ID 仍走参数校验返回 400，
// 与 404（合法 UUID 但记录不存在）区分开。
func TestDelete_BadUUIDReturns400(t *testing.T) {
	for _, path := range []string{
		"/api/v1/station-logs/not-a-uuid",
		"/api/v1/device-abnormal-reboots/not-a-uuid",
	} {
		r, _, _ := newTestHandler(t)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", path, nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, "path=%s", path)
	}
}

// TestDownload_NonexistentReturns404 station-logs 下载对不存在记录返回 404
// （DownloadURL 现返回 wrap ErrNotFound 的哨兵，handler 经 HTTPStatusFromError 映射）。
func TestDownload_NonexistentReturns404(t *testing.T) {
	r, _, _ := newTestHandler(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/station-logs/"+uuid.New().String()+"/download", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
}
