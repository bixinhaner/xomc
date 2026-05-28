package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 5 个 handler 测试，覆盖鉴权 / Create / Get / Update / Forbidden 路径。

func newRouterWithUser(svc *Service, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID != uuid.Nil {
			c.Set("user_id", userID)
		}
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group(""))
	return r
}

func Test_Handler_CreateRequiresUserID(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	r := newRouterWithUser(svc, uuid.Nil) // 没设 user_id

	body, _ := json.Marshal(map[string]any{"name": "x", "technology": "lte"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/dashboards", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func Test_Handler_CreateAndGet(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	r := newRouterWithUser(svc, owner)

	body, _ := json.Marshal(map[string]any{"name": "test", "technology": "lte"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/dashboards", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	dashID := data["id"].(string)

	// Get 应返 dashboard + panels
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/pm/dashboards/"+dashID, nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var getResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &getResp))
	getData := getResp["data"].(map[string]any)
	assert.NotNil(t, getData["dashboard"])
	assert.NotNil(t, getData["panels"])
}

func Test_Handler_GetForbidden(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	other := uuid.New()

	// owner 创建
	rOwner := newRouterWithUser(svc, owner)
	body, _ := json.Marshal(map[string]any{"name": "test", "technology": "lte"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/dashboards", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rOwner.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	dashID := resp["data"].(map[string]any)["id"].(string)

	// other 访问应 403
	rOther := newRouterWithUser(svc, other)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/pm/dashboards/"+dashID, nil)
	rOther.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func Test_Handler_UpdateOnlyOwner(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	other := uuid.New()

	// owner 创建
	d, _ := svc.Create(nil, owner, CreateDashboardRequest{Name: "old", Technology: TechLTE})
	// share to other
	svc.Share(nil, owner, ShareRequest{DashboardID: d.ID, UserIDs: []uuid.UUID{other}})

	// other 改名应 403
	rOther := newRouterWithUser(svc, other)
	body, _ := json.Marshal(map[string]any{"name": "new"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/pm/dashboards/"+d.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rOther.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// createDashboard 是 panel 测试的公共前置：owner 建一个 dashboard，返回其 ID。
func createDashboard(t *testing.T, r *gin.Engine) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"name": "panel-host", "technology": "lte"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/dashboards", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp["data"].(map[string]any)["id"].(string)
}

// adhoc_result 类型的 panel 不挂指标路径 / 粒度（数据来自已存的聚合任务），
// 前端按设计提交空数组，后端不应再按 min=1 拒绝。
func Test_Handler_CreatePanel_AdhocResultAllowsEmptyMetrics(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	r := newRouterWithUser(svc, owner)
	dashID := createDashboard(t, r)

	body, _ := json.Marshal(map[string]any{
		"panel_type":    "adhoc_result",
		"title":         "自定义聚合结果",
		"dimension":     "device",
		"metric_paths":  []string{},
		"granularities": []string{},
		"adhoc_task_id": uuid.New().String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/dashboards/"+dashID+"/panels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code, "adhoc_result panel 空指标应被接受")
}

// 非 adhoc_result 类型仍必须带指标路径 / 粒度，空数组应被拒（回归守护）。
func Test_Handler_CreatePanel_NonAdhocRequiresMetrics(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	r := newRouterWithUser(svc, owner)
	dashID := createDashboard(t, r)

	body, _ := json.Marshal(map[string]any{
		"panel_type":    "line_chart",
		"title":         "趋势图",
		"dimension":     "device",
		"metric_paths":  []string{},
		"granularities": []string{},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/dashboards/"+dashID+"/panels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code, "非 adhoc 类型空指标仍应拒绝")
}

func Test_Handler_GetPreferencesReturnsEmpty(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	user := uuid.New()
	r := newRouterWithUser(svc, user)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pm/user-preferences/dashboard", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, user.String(), data["user_id"])
}
