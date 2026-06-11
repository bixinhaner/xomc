package pm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// #64 PM 读链路设备组数据权限测试。
//
// 验证：非超管（受限分组）+ 请求 device_id 落在域外 → 403；落在域内 → 放行且
// filter.VisibleGroups 透传到仓库层；超管（resolver 返 nil）→ 不限制、不预检。

// pmFakePerm 是 authz.VisibleGroupsResolver 的测试替身。
type pmFakePerm struct {
	groups []uuid.UUID
	err    error
}

func (p pmFakePerm) GetUserVisibleGroupIDs(_ context.Context, _ uuid.UUID, _ bool) ([]uuid.UUID, error) {
	return p.groups, p.err
}

// pmAuthRouter 构造一个注入了权限服务 + 主体上下文的 PM 路由。
// deviceGroups 是 requestedDeviceInScope 预检时 fakeDeviceQuery 返回的设备分组。
func pmAuthRouter(cr counter.CounterRepository, kr kpi.KPIRepository, perm pmFakePerm, isSuper bool, deviceGroups []uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	dq := &fakeDeviceQuery{
		groupsFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) { return deviceGroups, nil },
	}
	h := &Handler{
		counterRepo: cr,
		kpiRepo:     kr,
		deviceQuery: dq,
		logger:      zap.NewNop(),
	}
	h.SetPermissionService(perm)
	// 模拟鉴权中间件：把主体身份写进 ctx，供 resolver 读取。
	r.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, uuid.New())
		c.Set(admin.CtxKeyIsSuperAdmin, isSuper)
		c.Next()
	})
	h.RegisterRoutes(r.Group(""))
	return r
}

func TestListCounters_OutOfScopeDevice_403(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()
	called := false
	cr := &pmHCounterRepo{
		queryFn: func(_ context.Context, _ counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			called = true
			return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
		},
	}
	// 调用者可见 g1；请求设备属于 g2 → 域外 → 403，且不触达仓库。
	router := pmAuthRouter(cr, &pmHKPIRepo{}, pmFakePerm{groups: []uuid.UUID{g1}}, false, []uuid.UUID{g2})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters?device_id="+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called, "out-of-scope device must not reach repository")
}

func TestListCounters_InScopeDevice_PropagatesVisibleGroups(t *testing.T) {
	g1 := uuid.New()
	var got counter.CounterFilter
	cr := &pmHCounterRepo{
		queryFn: func(_ context.Context, f counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			got = f
			return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
		},
	}
	// 可见 g1；请求设备也属于 g1 → 域内 → 放行，filter.VisibleGroups=[g1]。
	router := pmAuthRouter(cr, &pmHKPIRepo{}, pmFakePerm{groups: []uuid.UUID{g1}}, false, []uuid.UUID{g1})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters?device_id="+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []uuid.UUID{g1}, got.VisibleGroups)
	require.NotNil(t, got.DeviceID)
}

func TestListCounters_NoDeviceID_RestrictedGroupsStillApplied(t *testing.T) {
	g1 := uuid.New()
	var got counter.CounterFilter
	cr := &pmHCounterRepo{
		queryFn: func(_ context.Context, f counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			got = f
			return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
		},
	}
	router := pmAuthRouter(cr, &pmHKPIRepo{}, pmFakePerm{groups: []uuid.UUID{g1}}, false, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []uuid.UUID{g1}, got.VisibleGroups, "list-wide query must carry restricted groups for fail-closed repo filter")
}

func TestListCounters_SuperAdmin_NoRestriction(t *testing.T) {
	var got counter.CounterFilter
	cr := &pmHCounterRepo{
		queryFn: func(_ context.Context, f counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			got = f
			return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
		},
	}
	// 超管：resolver 返 nil（不限制），即便请求 device_id 也不预检 403。
	router := pmAuthRouter(cr, &pmHKPIRepo{}, pmFakePerm{groups: nil}, true, []uuid.UUID{uuid.New()})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters?device_id="+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Nil(t, got.VisibleGroups, "superadmin must see all (nil visibleGroups)")
}

func TestListKPIValues_OutOfScopeDevice_403(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()
	called := false
	kr := &pmHKPIRepo{
		queryFn: func(_ context.Context, _ kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
			called = true
			return model.NewListResponse([]model.KPIValue{}, 0, 1, 20), nil
		},
	}
	router := pmAuthRouter(&pmHCounterRepo{}, kr, pmFakePerm{groups: []uuid.UUID{g1}}, false, []uuid.UUID{g2})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi?device_id="+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called, "out-of-scope device must not reach kpi repository")
}

func TestListKPIValues_InScopeDevice_PropagatesVisibleGroups(t *testing.T) {
	g1 := uuid.New()
	var got kpi.KPIFilter
	kr := &pmHKPIRepo{
		queryFn: func(_ context.Context, f kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
			got = f
			return model.NewListResponse([]model.KPIValue{}, 0, 1, 20), nil
		},
	}
	router := pmAuthRouter(&pmHCounterRepo{}, kr, pmFakePerm{groups: []uuid.UUID{g1}}, false, []uuid.UUID{g1})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi?device_id="+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []uuid.UUID{g1}, got.VisibleGroups)
}
