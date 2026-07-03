package device

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Fakes for IDOR (device-group authorization) tests
// ---------------------------------------------------------------------------

// fakeGroupReader maps deviceID → its group memberships.
type fakeGroupReader struct {
	groups map[uuid.UUID][]uuid.UUID
}

func (f *fakeGroupReader) GetDeviceGroupIDs(_ context.Context, deviceID uuid.UUID) ([]uuid.UUID, error) {
	return f.groups[deviceID], nil
}

// fakePermService implements VisibleGroupsResolver.
//   - nil visibleByUser entry → superadmin path is decided by isSuper arg
//   - non-nil → the L2 groups visible to that user
type fakePermService struct {
	visibleByUser map[uuid.UUID][]uuid.UUID
}

func (f *fakePermService) GetUserVisibleGroupIDs(_ context.Context, userID uuid.UUID, isSuperAdmin bool) ([]uuid.UUID, error) {
	if isSuperAdmin {
		return nil, nil // superadmin sees all
	}
	return f.visibleByUser[userID], nil
}

// authzRouter wires the device Handler with permService + group reader and an
// auth-context middleware that injects user_id / is_super_admin like the real
// JWT middleware does.
func authzRouter(t *testing.T, userID uuid.UUID, isSuper bool, perm VisibleGroupsResolver, groupReader DeviceGroupReader) (*gin.Engine, *fakeDeviceRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	deviceRepo := newFakeDeviceRepo()
	paramRepo := newFakeParamRepo()
	svc := NewDeviceService(deviceRepo, paramRepo, nil, nil, zap.NewNop())
	svc.SetDeviceGroupReader(groupReader)

	h := NewHandler(svc)
	h.SetPermissionService(perm)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, userID)
		c.Set(admin.CtxKeyIsSuperAdmin, isSuper)
		c.Next()
	})
	h.RegisterRoutes(r.Group("/api/v1"))
	return r, deviceRepo
}

// ---------------------------------------------------------------------------
// Service-layer unit tests: AuthorizeDeviceGroupAccess
// ---------------------------------------------------------------------------

func TestAuthorizeDeviceGroupAccess(t *testing.T) {
	deviceID := uuid.New()
	groupA := uuid.New()
	groupB := uuid.New()
	groupC := uuid.New()

	reader := &fakeGroupReader{groups: map[uuid.UUID][]uuid.UUID{
		deviceID: {groupA, groupB},
	}}
	svc := NewDeviceService(newFakeDeviceRepo(), newFakeParamRepo(), nil, nil, zap.NewNop())
	svc.SetDeviceGroupReader(reader)

	t.Run("superadmin (nil visible groups) is allowed", func(t *testing.T) {
		err := svc.AuthorizeDeviceGroupAccess(context.Background(), deviceID, nil)
		assert.NoError(t, err)
	})

	t.Run("no permissions (empty slice) is forbidden", func(t *testing.T) {
		err := svc.AuthorizeDeviceGroupAccess(context.Background(), deviceID, []uuid.UUID{})
		assert.ErrorIs(t, err, commonerrors.ErrForbidden)
	})

	t.Run("visible group intersects device groups → allowed", func(t *testing.T) {
		err := svc.AuthorizeDeviceGroupAccess(context.Background(), deviceID, []uuid.UUID{groupB, groupC})
		assert.NoError(t, err)
	})

	t.Run("no intersection → forbidden", func(t *testing.T) {
		err := svc.AuthorizeDeviceGroupAccess(context.Background(), deviceID, []uuid.UUID{groupC})
		assert.ErrorIs(t, err, commonerrors.ErrForbidden)
	})

	t.Run("ungrouped device → forbidden for non-super", func(t *testing.T) {
		ungrouped := uuid.New() // not in reader.groups → empty memberships
		err := svc.AuthorizeDeviceGroupAccess(context.Background(), ungrouped, []uuid.UUID{groupA})
		assert.ErrorIs(t, err, commonerrors.ErrForbidden)
	})

	t.Run("ungrouped device → allowed when pseudo-group is visible", func(t *testing.T) {
		ungrouped := uuid.New() // not in reader.groups → empty memberships
		unassigned := uuid.MustParse(global.DefaultLevel2GroupID)
		err := svc.AuthorizeDeviceGroupAccess(context.Background(), ungrouped, []uuid.UUID{unassigned})
		assert.NoError(t, err)
	})

	t.Run("nil group reader degrades to allow", func(t *testing.T) {
		noReader := NewDeviceService(newFakeDeviceRepo(), newFakeParamRepo(), nil, nil, zap.NewNop())
		err := noReader.AuthorizeDeviceGroupAccess(context.Background(), deviceID, []uuid.UUID{groupC})
		assert.NoError(t, err, "无 reader 时不应拦截（dev/test 退化）")
	})
}

// ---------------------------------------------------------------------------
// Handler-layer tests: GET /devices/:id IDOR guard (success + 403 failure)
// ---------------------------------------------------------------------------

func TestGetDevice_IDOR_NonSuperWithoutAccess_Forbidden(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()
	deviceGroup := uuid.New() // group the device belongs to
	otherGroup := uuid.New()  // the only group the user can see

	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{
		userID: {otherGroup}, // user CANNOT see deviceGroup
	}}
	reader := &fakeGroupReader{groups: map[uuid.UUID][]uuid.UUID{
		deviceID: {deviceGroup},
	}}

	router, deviceRepo := authzRouter(t, userID, false, perm, reader)
	seedDevice(deviceRepo, deviceID, "SN-IDOR-403", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "越权读取他组设备应被拒 403")
}

func TestGetDevice_IDOR_NonSuperWithAccess_OK(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()
	sharedGroup := uuid.New()

	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{
		userID: {sharedGroup}, // user can see the device's group
	}}
	reader := &fakeGroupReader{groups: map[uuid.UUID][]uuid.UUID{
		deviceID: {sharedGroup},
	}}

	router, deviceRepo := authzRouter(t, userID, false, perm, reader)
	seedDevice(deviceRepo, deviceID, "SN-IDOR-OK", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "本组设备应正常返回 200")
}

func TestGetDevice_IDOR_Superadmin_OK(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()
	deviceGroup := uuid.New()

	// superadmin: GetUserVisibleGroupIDs returns nil regardless of mapping.
	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{}}
	reader := &fakeGroupReader{groups: map[uuid.UUID][]uuid.UUID{
		deviceID: {deviceGroup},
	}}

	router, deviceRepo := authzRouter(t, userID, true, perm, reader)
	seedDevice(deviceRepo, deviceID, "SN-IDOR-SUPER", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "超管应能读取任意设备")
}

func TestGetDeviceParameters_IDOR_Forbidden(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()
	deviceGroup := uuid.New()
	otherGroup := uuid.New()

	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{
		userID: {otherGroup},
	}}
	reader := &fakeGroupReader{groups: map[uuid.UUID][]uuid.UUID{
		deviceID: {deviceGroup},
	}}

	router, deviceRepo := authzRouter(t, userID, false, perm, reader)
	seedDevice(deviceRepo, deviceID, "SN-IDOR-PARAM-403", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+deviceID.String()+"/parameters", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "越权读取他组设备参数应被拒 403")
}

// TestGetDevice_NoPermService_DegradesToAllow guards the dev/test path where
// no data-permission service is wired: the by-ID read must not 403/500.
func TestGetDevice_NoPermService_DegradesToAllow(t *testing.T) {
	h, deviceRepo, _ := newTestHandler() // no permService, no group reader
	router := setupRouter(h)

	deviceID := uuid.New()
	seedDevice(deviceRepo, deviceID, "SN-NOPERM", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "未注入数据权限时应退化为放行")
}

// ---------------------------------------------------------------------------
// #64 GIS 地图读链路数据权限：ListGeo / GetGeoStats / SearchDevices 必须把
// 调用者可见分组下沉到仓库层（三态 fail-closed）。
// ---------------------------------------------------------------------------

func TestListGeo_NonSuper_RestrictsToVisibleGroups(t *testing.T) {
	userID := uuid.New()
	g1 := uuid.New()
	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{userID: {g1}}}

	router, deviceRepo := authzRouter(t, userID, false, perm, &fakeGroupReader{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/geo", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []uuid.UUID{g1}, deviceRepo.geoListVisible, "ListGeo 应把可见分组下沉到仓库层")
}

func TestListGeo_Superadmin_NoRestriction(t *testing.T) {
	userID := uuid.New()
	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{}}

	router, deviceRepo := authzRouter(t, userID, true, perm, &fakeGroupReader{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/geo", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, deviceRepo.geoListVisible, "超管 ListGeo 不应限制（nil 可见分组）")
}

func TestGetGeoStats_NonSuper_RestrictsToVisibleGroups(t *testing.T) {
	userID := uuid.New()
	g1 := uuid.New()
	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{userID: {g1}}}

	router, deviceRepo := authzRouter(t, userID, false, perm, &fakeGroupReader{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/geo/stats", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []uuid.UUID{g1}, deviceRepo.geoStatsVisible, "GetGeoStats 应把可见分组下沉到仓库层")
}

func TestSearchDevices_NonSuper_RestrictsToVisibleGroups(t *testing.T) {
	userID := uuid.New()
	g1 := uuid.New()
	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{userID: {g1}}}

	router, deviceRepo := authzRouter(t, userID, false, perm, &fakeGroupReader{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/search?keyword=abc", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []uuid.UUID{g1}, deviceRepo.geoSearchVisible, "SearchDevices 应把可见分组下沉到仓库层")
}

func TestSearchDevices_NoPermissions_FailClosedEmptySlice(t *testing.T) {
	userID := uuid.New()
	// 非超管且无任何可见分组 → 解析得 []（非 nil），仓库层 fail-closed。
	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{userID: {}}}

	router, deviceRepo := authzRouter(t, userID, false, perm, &fakeGroupReader{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/search?keyword=abc", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, deviceRepo.geoSearchVisible, "无权限应传 []（非 nil）触发 fail-closed")
	assert.Empty(t, deviceRepo.geoSearchVisible)
}
