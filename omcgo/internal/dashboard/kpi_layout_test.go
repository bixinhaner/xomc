package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
)

// fakeLayoutRepo 是 KPILayoutRepository 的内存假实现，按制式隔离存储，供 service 测试。
type fakeLayoutRepo struct {
	store     map[string]*KPILayout
	upsertErr error
	getErr    error
}

func newFakeLayoutRepo() *fakeLayoutRepo {
	return &fakeLayoutRepo{store: map[string]*KPILayout{}}
}

func (f *fakeLayoutRepo) GetByTech(_ context.Context, tech string) (*KPILayout, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	l, ok := f.store[tech]
	if !ok {
		return nil, nil // 无行
	}
	return l, nil
}

func (f *fakeLayoutRepo) Upsert(_ context.Context, tech string, layout json.RawMessage, updatedBy uuid.UUID) (*KPILayout, error) {
	if f.upsertErr != nil {
		return nil, f.upsertErr
	}
	uid := updatedBy
	l := &KPILayout{Tech: tech, Layout: layout, UpdatedBy: &uid, UpdatedAt: time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)}
	f.store[tech] = l
	return l, nil
}

func newTestService(repo KPILayoutRepository) *Service {
	return &Service{layoutRepo: repo, logger: zap.NewNop()}
}

// --- 读：无配置时回退内置默认 ---

func TestGetKPILayout_FallbackToDefaultWhenNoConfig(t *testing.T) {
	svc := newTestService(newFakeLayoutRepo()) // 空 store → 无行

	for _, tech := range []string{techLTE, techNR, techGSM} {
		got, err := svc.GetKPILayout(context.Background(), tech)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, tech, got.Tech)

		// 回退布局应与内置默认逐图等价（panels 非空）。
		var parsed struct {
			Panels []map[string]any `json:"panels"`
		}
		require.NoError(t, json.Unmarshal(got.Layout, &parsed))
		assert.NotEmpty(t, parsed.Panels, "default layout for %s must have panels", tech)
	}
}

func TestGetKPILayout_FallbackWhenRepoNil(t *testing.T) {
	svc := newTestService(nil) // 无仓库 → 回退默认
	got, err := svc.GetKPILayout(context.Background(), techLTE)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, techLTE, got.Tech)
}

// --- 读：有配置时返回所存布局 ---

func TestGetKPILayout_ReturnsStoredLayout(t *testing.T) {
	repo := newFakeLayoutRepo()
	stored := json.RawMessage(`{"panels":[{"title":"custom","metrics":["X"],"x":0,"y":0,"w":12,"h":8}]}`)
	repo.store[techLTE] = &KPILayout{Tech: techLTE, Layout: stored}
	svc := newTestService(repo)

	got, err := svc.GetKPILayout(context.Background(), techLTE)
	require.NoError(t, err)
	assert.JSONEq(t, string(stored), string(got.Layout))
}

func TestGetKPILayout_InvalidTech(t *testing.T) {
	svc := newTestService(newFakeLayoutRepo())
	_, err := svc.GetKPILayout(context.Background(), "wifi")
	assert.ErrorIs(t, err, ErrInvalidTech)
}

func TestGetKPILayout_RepoError(t *testing.T) {
	repo := newFakeLayoutRepo()
	repo.getErr = errors.New("db down")
	svc := newTestService(repo)
	_, err := svc.GetKPILayout(context.Background(), techLTE)
	assert.Error(t, err)
}

// --- 存：成功并持久化 ---

func TestSaveKPILayout_PersistsAndReadsBack(t *testing.T) {
	repo := newFakeLayoutRepo()
	svc := newTestService(repo)
	uid := uuid.New()
	body := json.RawMessage(`{"panels":[{"title":"t","metrics":["LTE_PDCP_VOLUME_DL"],"x":0,"y":0,"w":6,"h":8}]}`)

	saved, err := svc.SaveKPILayout(context.Background(), techLTE, body, uid)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, techLTE, saved.Tech)
	require.NotNil(t, saved.UpdatedBy)
	assert.Equal(t, uid, *saved.UpdatedBy)

	// 读回一致。
	got, err := svc.GetKPILayout(context.Background(), techLTE)
	require.NoError(t, err)
	assert.JSONEq(t, string(body), string(got.Layout))
}

func TestSaveKPILayout_InvalidTech(t *testing.T) {
	svc := newTestService(newFakeLayoutRepo())
	_, err := svc.SaveKPILayout(context.Background(), "5g", json.RawMessage(`{"panels":[]}`), uuid.New())
	assert.ErrorIs(t, err, ErrInvalidTech)
}

func TestSaveKPILayout_RejectsInvalidJSON(t *testing.T) {
	svc := newTestService(newFakeLayoutRepo())
	_, err := svc.SaveKPILayout(context.Background(), techLTE, json.RawMessage(`{not json`), uuid.New())
	assert.Error(t, err)
}

func TestSaveKPILayout_RejectsEmptyLayout(t *testing.T) {
	svc := newTestService(newFakeLayoutRepo())
	_, err := svc.SaveKPILayout(context.Background(), techLTE, json.RawMessage(``), uuid.New())
	assert.Error(t, err)
}

func TestSaveKPILayout_NoRepoConfigured(t *testing.T) {
	svc := newTestService(nil)
	_, err := svc.SaveKPILayout(context.Background(), techLTE, json.RawMessage(`{"panels":[]}`), uuid.New())
	assert.Error(t, err)
}

// --- 按制式隔离：存 lte 不影响 nr ---

func TestSaveKPILayout_TechIsolation(t *testing.T) {
	repo := newFakeLayoutRepo()
	svc := newTestService(repo)
	uid := uuid.New()

	lteBody := json.RawMessage(`{"panels":[{"title":"lte-only","metrics":["A"],"x":0,"y":0,"w":12,"h":8}]}`)
	_, err := svc.SaveKPILayout(context.Background(), techLTE, lteBody, uid)
	require.NoError(t, err)

	// nr 未存 → 仍回退默认（不被 lte 写入污染）。
	nr, err := svc.GetKPILayout(context.Background(), techNR)
	require.NoError(t, err)
	assert.NotContains(t, string(nr.Layout), "lte-only")
	assert.Contains(t, string(nr.Layout), "dashboard.panel.traffic") // 默认 NR 含 traffic

	// lte 读回 = 所存。
	lte, err := svc.GetKPILayout(context.Background(), techLTE)
	require.NoError(t, err)
	assert.JSONEq(t, string(lteBody), string(lte.Layout))
}

// ---------------------------------------------------------------------------
// Handler 层：非管理员存盘被拒（失败路径）
// ---------------------------------------------------------------------------

// fakePermChecker 是 admin.PermissionChecker 的假实现，按 allowed 控制是否放行。
type fakePermChecker struct{ allowed bool }

func (f fakePermChecker) CheckPermission(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
	return f.allowed, nil
}
func (f fakePermChecker) GetPermissions(_ context.Context, _ uuid.UUID) ([]admin.Permission, error) {
	return nil, nil
}
func (f fakePermChecker) ListAllPermissions(_ context.Context) ([]admin.Permission, error) {
	return nil, nil
}

func newSaveLayoutRequest(t *testing.T) *http.Request {
	t.Helper()
	body := `{"tech":"lte","layout":{"panels":[]}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/kpi-layout", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// 非管理员（非 super_admin 且无 Casbin 权限点）存盘 → 403。
func TestSaveKPILayoutHandler_NonAdminRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(newTestService(newFakeLayoutRepo()), fakePermChecker{allowed: false})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = newSaveLayoutRequest(t)
	c.Set(admin.CtxKeyUserID, uuid.New())
	c.Set(admin.CtxKeyIsSuperAdmin, false)

	h.SaveKPILayout(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// super_admin 存盘 → 放行（200）。
func TestSaveKPILayoutHandler_SuperAdminAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(newTestService(newFakeLayoutRepo()), fakePermChecker{allowed: false})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = newSaveLayoutRequest(t)
	c.Set(admin.CtxKeyUserID, uuid.New())
	c.Set(admin.CtxKeyIsSuperAdmin, true) // 超管旁路

	h.SaveKPILayout(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 普通用户但被 Casbin 授予该端点权限点 → 放行（200）。
func TestSaveKPILayoutHandler_GrantedViaCasbin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(newTestService(newFakeLayoutRepo()), fakePermChecker{allowed: true})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = newSaveLayoutRequest(t)
	c.Set(admin.CtxKeyUserID, uuid.New())
	c.Set(admin.CtxKeyIsSuperAdmin, false)

	h.SaveKPILayout(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 无 permChecker（nil）且非 super_admin → 保守拒绝（403）。
func TestSaveKPILayoutHandler_NilCheckerRejectsNonSuper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(newTestService(newFakeLayoutRepo()), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = newSaveLayoutRequest(t)
	c.Set(admin.CtxKeyUserID, uuid.New())
	c.Set(admin.CtxKeyIsSuperAdmin, false)

	h.SaveKPILayout(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
