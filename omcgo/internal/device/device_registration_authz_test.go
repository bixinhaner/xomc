package device

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// #64 预注册（device-registrations）列表设备组数据权限测试。
//
// 验证：handler 把调用者可见分组下沉到 RegistrationFilter.VisibleGroups（三态），
// 仓库层据此对 group_id 取交（group_id IS NULL 的待上线条目对非超管天然不命中被排除，
// 仅超管 nil 可见全部）。

// fakeRegRepo 记录最近一次 List 收到的 filter，便于断言可见分组下沉。
type fakeRegRepo struct {
	gotFilter RegistrationFilter
}

func (r *fakeRegRepo) Create(_ context.Context, _ *DeviceRegistration) error { return nil }
func (r *fakeRegRepo) BatchCreate(_ context.Context, _ []*DeviceRegistration) (int, error) {
	return 0, nil
}
func (r *fakeRegRepo) GetBySerialNumber(_ context.Context, _ string) (*DeviceRegistration, error) {
	return nil, nil
}
func (r *fakeRegRepo) List(_ context.Context, filter RegistrationFilter) (*model.ListResponse[DeviceRegistration], error) {
	r.gotFilter = filter
	return model.NewListResponse([]DeviceRegistration{}, 0, 1, 20), nil
}
func (r *fakeRegRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *fakeRegRepo) Delete(_ context.Context, _ uuid.UUID) error                 { return nil }

func regAuthRouter(userID uuid.UUID, isSuper bool, perm VisibleGroupsResolver) (*gin.Engine, *fakeRegRepo) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRegRepo{}
	svc := NewRegistrationService(repo, zap.NewNop())
	h := NewRegistrationHandler(svc)
	h.SetPermissionService(perm)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, userID)
		c.Set(admin.CtxKeyIsSuperAdmin, isSuper)
		c.Next()
	})
	h.RegisterRoutes(r.Group("/api/v1"))
	return r, repo
}

func TestListRegistrations_NonSuper_RestrictsToVisibleGroups(t *testing.T) {
	userID := uuid.New()
	g1 := uuid.New()
	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{userID: {g1}}}

	router, repo := regAuthRouter(userID, false, perm)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-registrations", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []uuid.UUID{g1}, repo.gotFilter.VisibleGroups,
		"非超管预注册列表应限定到可见分组（未分组/他组条目被排除）")
}

func TestListRegistrations_Superadmin_SeesAll(t *testing.T) {
	userID := uuid.New()
	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{}}

	router, repo := regAuthRouter(userID, true, perm)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-registrations", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, repo.gotFilter.VisibleGroups, "超管不限制（nil），含 group_id IS NULL 的待上线条目全可见")
}

func TestListRegistrations_NoPermissions_FailClosedEmptySlice(t *testing.T) {
	userID := uuid.New()
	perm := &fakePermService{visibleByUser: map[uuid.UUID][]uuid.UUID{userID: {}}}

	router, repo := regAuthRouter(userID, false, perm)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-registrations", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, repo.gotFilter.VisibleGroups, "无权限应传 []（非 nil）触发 fail-closed")
	assert.Empty(t, repo.gotFilter.VisibleGroups)
}

func TestListRegistrations_NoPermService_DegradesToNoFilter(t *testing.T) {
	userID := uuid.New()
	router, repo := regAuthRouter(userID, false, nil) // permService 未注入

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-registrations", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, repo.gotFilter.VisibleGroups, "dev/test 未注入数据权限时退化为不过滤")
}
