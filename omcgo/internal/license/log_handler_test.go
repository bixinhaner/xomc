package license

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// log_handler_test.go — T-0100-P1 GET /licenses/logs + GET /licenses/:id/logs
// 单元测试。复用 log_writer_test.go 的 memLogRepo（在 LicenseLogRepository
// 接口实现充分），通过 setupLicenseRouter 起 gin 路由测端点。

// 把 envelope { ret, msg, data } 拆出 data 节点；与 frontend http.ts 拦截器对齐。
func decodeLogsEnvelope(t *testing.T, body []byte) []byte {
	t.Helper()
	var env struct {
		Ret  int             `json:"ret"`
		Data json.RawMessage `json:"data"`
		Msg  string          `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(body, &env))
	require.Equal(t, 1, env.Ret, "ret should be 1, body=%s", string(body))
	return env.Data
}

// seedLogsForTest 往 memLogRepo 喂多条不同维度的日志，用于过滤测试。
func seedLogsForTest(t *testing.T, repo *memLogRepo, licID uuid.UUID, otherLicID uuid.UUID) {
	t.Helper()
	w := NewLogWriter(repo, zap.NewNop())
	actor := uuid.New()
	cases := []LicenseLogEntry{
		{LicenseID: &licID, LogType: LogTypeImport, ActorUserID: &actor, Result: LogResultSuccess, Details: map[string]any{"summary": "imported"}, ClientIP: "1.1.1.1"},
		{LicenseID: &licID, LogType: LogTypeActivate, ActorUserID: &actor, Result: LogResultSuccess, Details: map[string]any{"summary": "activated"}},
		{LicenseID: &licID, LogType: LogTypeRevoke, ActorUserID: &actor, Result: LogResultFailed, Details: map[string]any{"err": "no perm"}},
		{LicenseID: &licID, LogType: LogTypeEnforcementCapacity, Result: LogResultDenied, Details: map[string]any{"summary": "capacity exceeded"}},
		{LicenseID: &licID, LogType: LogTypeEnforcementExpiry, Result: LogResultDenied},
		{LicenseID: &licID, LogType: LogTypeCapacityAlert, Result: LogResultWarning},
		{LicenseID: &licID, LogType: LogTypeExpiryAlert, Result: LogResultWarning},
		{LicenseID: &licID, LogType: LogTypeAutoExpire, Result: LogResultSuccess},
		{LicenseID: &otherLicID, LogType: LogTypeImport, ActorUserID: &actor, Result: LogResultSuccess},
	}
	for _, e := range cases {
		w.Write(context.Background(), e)
	}
}

// ---------------------------------------------------------------------------
// 503 — repo 未注入
// ---------------------------------------------------------------------------

func TestListLogs_NoLogRepoConfigured(t *testing.T) {
	h, _ := newTestLicenseHandler()
	// 故意不调 SetLogRepo
	r := setupLicenseRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetLicenseLogs_NoLogRepoConfigured(t *testing.T) {
	h, _ := newTestLicenseHandler()
	r := setupLicenseRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/licenses/"+uuid.New().String()+"/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// GET /licenses/logs — 全量列表 + 分页
// ---------------------------------------------------------------------------

func TestListLogs_HappyPath(t *testing.T) {
	h, _ := newTestLicenseHandler()
	logRepo := newMemLogRepo()
	licID, otherID := uuid.New(), uuid.New()
	seedLogsForTest(t, logRepo, licID, otherID)
	h.SetLogRepo(logRepo)

	r := setupLicenseRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data := decodeLogsEnvelope(t, w.Body.Bytes())
	var resp struct {
		Items []LicenseLog `json:"items"`
		Total int64        `json:"total"`
	}
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.Equal(t, int64(9), resp.Total)
	assert.Len(t, resp.Items, 9)
}

// ---------------------------------------------------------------------------
// GET /licenses/logs?license_id=xxx — 单 license 过滤
// ---------------------------------------------------------------------------

func TestListLogs_FilterByLicenseID(t *testing.T) {
	h, _ := newTestLicenseHandler()
	logRepo := newMemLogRepo()
	licID, otherID := uuid.New(), uuid.New()
	seedLogsForTest(t, logRepo, licID, otherID)
	h.SetLogRepo(logRepo)
	r := setupLicenseRouter(h)

	q := url.Values{}
	q.Set("license_id", licID.String())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/logs?"+q.Encode(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data := decodeLogsEnvelope(t, w.Body.Bytes())
	var resp struct {
		Items []LicenseLog `json:"items"`
		Total int64        `json:"total"`
	}
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.Equal(t, int64(8), resp.Total, "应只返回 licID 的 8 条")
	for _, l := range resp.Items {
		require.NotNil(t, l.LicenseID)
		assert.Equal(t, licID, *l.LicenseID)
	}
}

// ---------------------------------------------------------------------------
// GET /licenses/logs?log_type=enforcement_capacity&log_type=enforcement_expiry — 多值
// ---------------------------------------------------------------------------

func TestListLogs_FilterByLogTypes(t *testing.T) {
	h, _ := newTestLicenseHandler()
	logRepo := newMemLogRepo()
	licID, otherID := uuid.New(), uuid.New()
	seedLogsForTest(t, logRepo, licID, otherID)
	h.SetLogRepo(logRepo)
	r := setupLicenseRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/licenses/logs?log_type=enforcement_capacity&log_type=enforcement_expiry", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data := decodeLogsEnvelope(t, w.Body.Bytes())
	var resp struct {
		Items []LicenseLog `json:"items"`
		Total int64        `json:"total"`
	}
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.Equal(t, int64(2), resp.Total, "应只返回 2 条 enforcement_*")
	for _, l := range resp.Items {
		assert.Contains(t,
			[]LogType{LogTypeEnforcementCapacity, LogTypeEnforcementExpiry},
			l.LogType,
		)
	}
}

// ---------------------------------------------------------------------------
// GET /licenses/logs?result=denied — 单 result 过滤
// ---------------------------------------------------------------------------

func TestListLogs_FilterByResult(t *testing.T) {
	h, _ := newTestLicenseHandler()
	logRepo := newMemLogRepo()
	licID, otherID := uuid.New(), uuid.New()
	seedLogsForTest(t, logRepo, licID, otherID)
	h.SetLogRepo(logRepo)
	r := setupLicenseRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/logs?result=denied", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data := decodeLogsEnvelope(t, w.Body.Bytes())
	var resp struct {
		Items []LicenseLog `json:"items"`
		Total int64        `json:"total"`
	}
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.Equal(t, int64(2), resp.Total, "应只返回 2 条 enforcement_* (result=denied)")
	for _, l := range resp.Items {
		assert.Equal(t, LogResultDenied, l.Result)
	}
}

// ---------------------------------------------------------------------------
// GET /licenses/logs?license_id=invalid → 400
// ---------------------------------------------------------------------------

func TestListLogs_InvalidLicenseID(t *testing.T) {
	h, _ := newTestLicenseHandler()
	h.SetLogRepo(newMemLogRepo())
	r := setupLicenseRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/licenses/logs?license_id=not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GET /licenses/:id/logs — 单 license 抽屉接口
// ---------------------------------------------------------------------------

func TestGetLicenseLogs_HappyPath(t *testing.T) {
	h, _ := newTestLicenseHandler()
	logRepo := newMemLogRepo()
	licID, otherID := uuid.New(), uuid.New()
	seedLogsForTest(t, logRepo, licID, otherID)
	h.SetLogRepo(logRepo)
	r := setupLicenseRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/licenses/"+licID.String()+"/logs?limit=5", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data := decodeLogsEnvelope(t, w.Body.Bytes())
	var resp struct {
		Items []LicenseLog `json:"items"`
		Total int          `json:"total"`
	}
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.LessOrEqual(t, resp.Total, 5)
	assert.LessOrEqual(t, len(resp.Items), 5)
	for _, l := range resp.Items {
		require.NotNil(t, l.LicenseID)
		assert.Equal(t, licID, *l.LicenseID)
	}
}

func TestGetLicenseLogs_InvalidUUID(t *testing.T) {
	h, _ := newTestLicenseHandler()
	h.SetLogRepo(newMemLogRepo())
	r := setupLicenseRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/not-a-uuid/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// 限流：limit 上限 100；越界用默认 10
// ---------------------------------------------------------------------------

func TestGetLicenseLogs_LimitClamping(t *testing.T) {
	h, _ := newTestLicenseHandler()
	logRepo := newMemLogRepo()
	licID, otherID := uuid.New(), uuid.New()
	// 喂 12 条同 license，验证 limit 默认 10
	for i := 0; i < 12; i++ {
		seedLogsForTest(t, logRepo, licID, otherID)
	}
	h.SetLogRepo(logRepo)
	r := setupLicenseRouter(h)

	// limit=0 / 越界 → 默认 10
	for _, q := range []string{"", "?limit=0", "?limit=-5", "?limit=999999", "?limit=abc"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet,
			"/api/v1/licenses/"+licID.String()+"/logs"+q, nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "query=%q", q)
		data := decodeLogsEnvelope(t, w.Body.Bytes())
		var resp struct {
			Items []LicenseLog `json:"items"`
			Total int          `json:"total"`
		}
		require.NoError(t, json.Unmarshal(data, &resp))
		// limit=10 max；total 反映返回数量
		assert.LessOrEqual(t, resp.Total, 100, "query=%q", q)
	}
}
