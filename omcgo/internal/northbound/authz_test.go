package northbound

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// --- fakes ---

// fakeVisibleGroups 模拟用户可见设备组解析。
type fakeVisibleGroups struct {
	groups map[uuid.UUID][]uuid.UUID // userID → 可见组（nil 表示超管全可见）
}

func (f fakeVisibleGroups) GetUserVisibleGroupIDs(_ context.Context, userID uuid.UUID, isSuperAdmin bool) ([]uuid.UUID, error) {
	if isSuperAdmin {
		return nil, nil // 超管：nil 表示全可见（与真实 PermissionService 语义一致）
	}
	// 非超管：缺省返回非 nil 空切片（无可见组），与真实 PermissionService 区分
	// "超管(nil)" 与 "无权限([])"。
	if g, ok := f.groups[userID]; ok {
		return g, nil
	}
	return []uuid.UUID{}, nil
}

// fakeDeviceAuthz 按 deviceID→所属组 判定设备是否在可见组内（复刻 device 域语义）。
type fakeDeviceAuthz struct {
	deviceGroups map[uuid.UUID][]uuid.UUID // deviceID → 所属组
}

func (f fakeDeviceAuthz) AuthorizeDeviceGroupAccess(_ context.Context, deviceID uuid.UUID, visibleGroups []uuid.UUID) error {
	if visibleGroups == nil { // 超管
		return nil
	}
	if len(visibleGroups) == 0 {
		return commonerrors.ErrForbidden
	}
	visible := map[uuid.UUID]struct{}{}
	for _, g := range visibleGroups {
		visible[g] = struct{}{}
	}
	for _, g := range f.deviceGroups[deviceID] {
		if _, ok := visible[g]; ok {
			return nil
		}
	}
	return commonerrors.ErrForbidden
}

// fakeSNResolver 把固定 SN 映射成 deviceID。
type fakeSNResolver struct {
	byID map[string]uuid.UUID
}

func (f fakeSNResolver) ResolveDeviceID(_ context.Context, sn string) (uuid.UUID, bool, error) {
	id, ok := f.byID[sn]
	return id, ok, nil
}

// ginCtxWith 构造一个携带认证上下文的 gin.Context + recorder。
func ginCtxWith(userID uuid.UUID, isSuper bool) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/northbound/x", nil)
	c.Set(admin.CtxKeyUserID, userID)
	c.Set(admin.CtxKeyIsSuperAdmin, isSuper)
	return c, w
}

// --- 跨租户 IDOR 回归（核心验收）---

// CTCC 用户访问 CMCC 设备 → 403（跨租户拒绝）。这里用「不同设备组」模拟运营商隔离边界。
func TestScoper_AuthorizeDevice_CrossTenantForbidden(t *testing.T) {
	ctccUser := uuid.New()
	ctccGroup := uuid.New()
	cmccGroup := uuid.New()
	cmccDevice := uuid.New()

	scoper := NewScoper(
		fakeVisibleGroups{groups: map[uuid.UUID][]uuid.UUID{ctccUser: {ctccGroup}}},
		fakeDeviceAuthz{deviceGroups: map[uuid.UUID][]uuid.UUID{cmccDevice: {cmccGroup}}},
		nil,
	)

	c, w := ginCtxWith(ctccUser, false)
	allowed := scoper.AuthorizeDevice(c, cmccDevice)

	assert.False(t, allowed, "跨租户访问应被拒")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// 同租户设备 → 放行。
func TestScoper_AuthorizeDevice_SameTenantAllowed(t *testing.T) {
	user := uuid.New()
	group := uuid.New()
	dev := uuid.New()

	scoper := NewScoper(
		fakeVisibleGroups{groups: map[uuid.UUID][]uuid.UUID{user: {group}}},
		fakeDeviceAuthz{deviceGroups: map[uuid.UUID][]uuid.UUID{dev: {group}}},
		nil,
	)

	c, w := ginCtxWith(user, false)
	allowed := scoper.AuthorizeDevice(c, dev)

	assert.True(t, allowed)
	assert.Equal(t, http.StatusOK, w.Code) // 未写错误响应
}

// 超管访问任意设备 → 放行。
func TestScoper_AuthorizeDevice_SuperAdminUnrestricted(t *testing.T) {
	superAdmin := uuid.New()
	dev := uuid.New()

	scoper := NewScoper(
		fakeVisibleGroups{},
		fakeDeviceAuthz{deviceGroups: map[uuid.UUID][]uuid.UUID{dev: {uuid.New()}}},
		nil,
	)

	c, _ := ginCtxWith(superAdmin, true)
	assert.True(t, scoper.AuthorizeDevice(c, dev))
}

// 无任何可见组的非超管 → 拒绝（403）。
func TestScoper_AuthorizeDevice_NoVisibleGroupsForbidden(t *testing.T) {
	user := uuid.New()
	dev := uuid.New()

	scoper := NewScoper(
		fakeVisibleGroups{groups: map[uuid.UUID][]uuid.UUID{}}, // user 无可见组
		fakeDeviceAuthz{deviceGroups: map[uuid.UUID][]uuid.UUID{dev: {uuid.New()}}},
		nil,
	)

	c, w := ginCtxWith(user, false)
	assert.False(t, scoper.AuthorizeDevice(c, dev))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// AuthorizeDeviceBySN：跨租户 SN → 403。
func TestScoper_AuthorizeDeviceBySN_CrossTenantForbidden(t *testing.T) {
	ctccUser := uuid.New()
	ctccGroup := uuid.New()
	cmccGroup := uuid.New()
	cmccDevice := uuid.New()

	scoper := NewScoper(
		fakeVisibleGroups{groups: map[uuid.UUID][]uuid.UUID{ctccUser: {ctccGroup}}},
		fakeDeviceAuthz{deviceGroups: map[uuid.UUID][]uuid.UUID{cmccDevice: {cmccGroup}}},
		fakeSNResolver{byID: map[string]uuid.UUID{"SN-CMCC": cmccDevice}},
	)

	c, w := ginCtxWith(ctccUser, false)
	assert.False(t, scoper.AuthorizeDeviceBySN(c, "SN-CMCC"))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// AuthorizeDeviceBySN：未知 SN → 404。
func TestScoper_AuthorizeDeviceBySN_UnknownReturns404(t *testing.T) {
	user := uuid.New()
	scoper := NewScoper(
		fakeVisibleGroups{groups: map[uuid.UUID][]uuid.UUID{user: {uuid.New()}}},
		fakeDeviceAuthz{deviceGroups: map[uuid.UUID][]uuid.UUID{}},
		fakeSNResolver{byID: map[string]uuid.UUID{}},
	)

	c, w := ginCtxWith(user, false)
	assert.False(t, scoper.AuthorizeDeviceBySN(c, "SN-DOES-NOT-EXIST"))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// RequireDeviceScope：非超管无设备维度 → 403；超管放行；有维度放行。
func TestScoper_RequireDeviceScope(t *testing.T) {
	user := uuid.New()
	scoper := NewScoper(
		fakeVisibleGroups{groups: map[uuid.UUID][]uuid.UUID{user: {uuid.New()}}},
		fakeDeviceAuthz{},
		nil,
	)

	// 非超管 + 无设备维度 → 403。
	c1, w1 := ginCtxWith(user, false)
	assert.False(t, scoper.RequireDeviceScope(c1, false))
	assert.Equal(t, http.StatusForbidden, w1.Code)

	// 非超管 + 有设备维度 → 放行。
	c2, _ := ginCtxWith(user, false)
	assert.True(t, scoper.RequireDeviceScope(c2, true))

	// 超管 + 无设备维度 → 放行（可全量）。
	c3, _ := ginCtxWith(uuid.New(), true)
	assert.True(t, scoper.RequireDeviceScope(c3, false))
}

func TestScoper_NorthboundAPIUserBypassesSystemUserScope(t *testing.T) {
	deviceID := uuid.New()
	scoper := NewScoper(
		fakeVisibleGroups{groups: map[uuid.UUID][]uuid.UUID{}},
		fakeDeviceAuthz{},
		nil,
	)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("northbound_api_user", "oss-a")

	assert.True(t, scoper.RequireDeviceScope(c, false))
	assert.True(t, scoper.AuthorizeDevice(c, deviceID))
}

// nil scoper 依赖（dev/test）→ 退化放行。
func TestScoper_NilDeps_FailOpen(t *testing.T) {
	scoper := NewScoper(nil, nil, nil)
	c, _ := ginCtxWith(uuid.New(), false)
	assert.True(t, scoper.AuthorizeDevice(c, uuid.New()))
	assert.True(t, scoper.RequireDeviceScope(c, false))
}

// --- 过滤白名单 ---

func TestFilterWhitelist_RejectsUnknown(t *testing.T) {
	require.Empty(t, unknownFilterParams(map[string][]string{"data_type": {"device"}}, allowedSyncParams))
	bad := unknownFilterParams(map[string][]string{"data_type": {"device"}, "evil": {"1"}}, allowedSyncParams)
	require.Len(t, bad, 1)
	assert.Equal(t, "evil", bad[0])

	require.Empty(t, unknownJSONFields(map[string]any{"device_id": "x", "start_time": "t"}, allowedPMExportParams))
	badJSON := unknownJSONFields(map[string]any{"device_id": "x", "drop_table": "1"}, allowedPMExportParams)
	require.Len(t, badJSON, 1)
	assert.Equal(t, "drop_table", badJSON[0])
}
