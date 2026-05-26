package device

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// T-0176-PR-D: CreateDevice inline ProductRegistry 路由 + UpdateDevice/DeleteDevice
// cache 失效 测试。
//
// 独立 _test.go 文件聚合 PR-D 改动相关断言，避免污染 service_test.go 的既有
// table-driven 风格 — 便于审查与回滚。
// ---------------------------------------------------------------------------

// stubProductMatcher 实现 ProductClassMatcher 接口（test only）。
type stubPRDProductMatcher struct {
	calls   atomic.Int32
	matchFn func(ctx context.Context, productClass string) (*product.MatchResult, error)
}

func (s *stubPRDProductMatcher) MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error) {
	s.calls.Add(1)
	if s.matchFn != nil {
		return s.matchFn(ctx, productClass)
	}
	return nil, product.ErrOrphan
}

// stubProductBinder 实现 ProductBinder 接口（test only）。
type stubPRDProductBinder struct {
	calls        atomic.Int32
	lastDeviceID uuid.UUID
	lastProduct  uuid.UUID
	lastParamMod *uuid.UUID
	bindFn       func(ctx context.Context, deviceID, productID uuid.UUID, paramModelID *uuid.UUID) error
}

func (s *stubPRDProductBinder) BindDevice(ctx context.Context, deviceID, productID uuid.UUID, paramModelID *uuid.UUID) error {
	s.calls.Add(1)
	s.lastDeviceID = deviceID
	s.lastProduct = productID
	s.lastParamMod = paramModelID
	if s.bindFn != nil {
		return s.bindFn(ctx, deviceID, productID, paramModelID)
	}
	return nil
}

// newPRDService 构造一个含 productMatcher / productBinder / metrics 的 DeviceService。
func newPRDService(deviceRepo *mockDeviceRepo, matcher ProductClassMatcher, binder ProductBinder) (*DeviceService, *DeviceMetrics) {
	logger := zap.NewNop()
	svc := NewDeviceService(deviceRepo, &mockParamRepo{}, nil, nil, logger)
	if matcher != nil {
		svc.SetProductMatcher(matcher)
	}
	if binder != nil {
		svc.SetProductBinder(binder)
	}
	metrics := NewDeviceMetrics(prometheus.NewRegistry())
	svc.SetMetrics(metrics)
	return svc, metrics
}

// testCounterValue 读 Prometheus Counter 当前累计值（使用官方 testutil 工具）。
func testCounterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	return testutil.ToFloat64(c)
}

func Test_CreateDevice_HitProductClass_CallsBindDevice(t *testing.T) {
	productID := uuid.New()
	paramModelID := uuid.New()
	matchedProduct := &product.Product{
		ID:           productID,
		Name:         "TestProd",
		ParamModelID: &paramModelID,
	}

	matcher := &stubPRDProductMatcher{
		matchFn: func(_ context.Context, pc string) (*product.MatchResult, error) {
			assert.Equal(t, "PCLASS-A", pc)
			return &product.MatchResult{Product: matchedProduct, MatchedPattern: "PCLASS-.*"}, nil
		},
	}
	binder := &stubPRDProductBinder{}

	var created *model.Device
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) { return nil, nil },
		createFn: func(_ context.Context, d *model.Device) error {
			created = d
			return nil
		},
	}

	svc, _ := newPRDService(deviceRepo, matcher, binder)

	req := CreateDeviceRequest{
		SerialNumber: "SN-HIT",
		OUI:          "112233",
		ProductClass: "PCLASS-A",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	}
	dev, err := svc.CreateDevice(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, dev)
	require.NotNil(t, created)

	assert.Equal(t, int32(1), matcher.calls.Load(), "MatchProductClass 调一次")
	assert.Equal(t, int32(1), binder.calls.Load(), "BindDevice 应被调一次")
	assert.Equal(t, dev.ID, binder.lastDeviceID, "传入的 device ID 必须与 INSERT 后的 ID 一致")
	assert.Equal(t, productID, binder.lastProduct)
	require.NotNil(t, binder.lastParamMod)
	assert.Equal(t, paramModelID, *binder.lastParamMod)
}

func Test_CreateDevice_OrphanProductClass_IncrementsMetric(t *testing.T) {
	matcher := &stubPRDProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			return nil, product.ErrOrphan
		},
	}
	binder := &stubPRDProductBinder{}

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) { return nil, nil },
		createFn:            func(_ context.Context, _ *model.Device) error { return nil },
	}
	svc, metrics := newPRDService(deviceRepo, matcher, binder)

	before := testCounterValue(t, metrics.DeviceCreateOrphan)
	dev, err := svc.CreateDevice(context.Background(), CreateDeviceRequest{
		SerialNumber: "SN-ORPHAN",
		ProductClass: "UNKNOWN-CLASS",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	})
	require.NoError(t, err)
	require.NotNil(t, dev)

	after := testCounterValue(t, metrics.DeviceCreateOrphan)
	assert.Equal(t, before+1, after, "orphan 应使 device_create_orphan_total +1")
	assert.Equal(t, int32(0), binder.calls.Load(), "orphan 时不应调 BindDevice")
}

func Test_CreateDevice_MatchTrueError_ReturnsError(t *testing.T) {
	// 非 ErrOrphan 的真错（如 DB 故障）应导致 CreateDevice 整体失败，不写脏数据。
	matcher := &stubPRDProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			return nil, errors.New("DB connection lost")
		},
	}
	binder := &stubPRDProductBinder{}

	createCalled := atomic.Bool{}
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) { return nil, nil },
		createFn: func(_ context.Context, _ *model.Device) error {
			createCalled.Store(true)
			return nil
		},
	}
	svc, _ := newPRDService(deviceRepo, matcher, binder)

	dev, err := svc.CreateDevice(context.Background(), CreateDeviceRequest{
		SerialNumber: "SN-DBERR",
		ProductClass: "PCLASS-X",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	})
	require.Error(t, err)
	assert.Nil(t, dev)
	assert.Contains(t, err.Error(), "resolve product")
	assert.False(t, createCalled.Load(), "真错时不应进入 INSERT")
	assert.Equal(t, int32(0), binder.calls.Load())
}

func Test_CreateDevice_HitButBindFails_DeviceStillCreated(t *testing.T) {
	// BindDevice 失败应仅 warn 不回滚 device 行（PR-D 事实修正：product_id 列只服务
	// 审计/denorm，不阻塞设备创建）。
	matcher := &stubPRDProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			return &product.MatchResult{Product: &product.Product{ID: uuid.New(), Name: "P"}}, nil
		},
	}
	binder := &stubPRDProductBinder{
		bindFn: func(_ context.Context, _, _ uuid.UUID, _ *uuid.UUID) error {
			return errors.New("simulated bind failure")
		},
	}
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) { return nil, nil },
		createFn:            func(_ context.Context, _ *model.Device) error { return nil },
	}
	svc, _ := newPRDService(deviceRepo, matcher, binder)

	dev, err := svc.CreateDevice(context.Background(), CreateDeviceRequest{
		SerialNumber: "SN-BIND-FAIL",
		ProductClass: "PCLASS-A",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	})
	require.NoError(t, err, "bind 失败不应影响 CreateDevice 整体成功")
	require.NotNil(t, dev)
}

func Test_CreateDevice_NoProductClass_SkipsMatch(t *testing.T) {
	matcher := &stubPRDProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			t.Fatal("空 productClass 时不应调 Match")
			return nil, nil
		},
	}
	binder := &stubPRDProductBinder{}
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) { return nil, nil },
		createFn:            func(_ context.Context, _ *model.Device) error { return nil },
	}
	svc, _ := newPRDService(deviceRepo, matcher, binder)

	_, err := svc.CreateDevice(context.Background(), CreateDeviceRequest{
		SerialNumber: "SN-NO-PC",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(0), matcher.calls.Load())
	assert.Equal(t, int32(0), binder.calls.Load())
}

func Test_UpdateDevice_DeletesCache(t *testing.T) {
	deviceID := uuid.New()
	sn := "SN-UPD-CACHE"
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn, DeviceName: "Old"}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}

	svc, _ := newPRDService(deviceRepo, nil, nil)
	// 装一个真 cache，预先 Set 一条假数据，验证 Update 后被清掉
	mr := miniredis.RunT(t)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()
	cache := NewDeviceCache(rdb, zap.NewNop())
	cache.Set(context.Background(), &model.Device{ID: deviceID, SerialNumber: sn, DeviceName: "Stale"})
	svc.SetDeviceCache(cache)

	// 先验证 cache 命中
	_, hit := cache.Get(context.Background(), sn)
	require.True(t, hit, "前置：cache 应已写入")

	newName := "New"
	_, err := svc.UpdateDevice(context.Background(), deviceID, UpdateDeviceRequest{DeviceName: &newName})
	require.NoError(t, err)

	_, hit = cache.Get(context.Background(), sn)
	assert.False(t, hit, "UpdateDevice 后 cache 应被清掉")
}

func Test_DeleteDevice_DeletesCache(t *testing.T) {
	deviceID := uuid.New()
	sn := "SN-DEL-CACHE"
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}

	svc, _ := newPRDService(deviceRepo, nil, nil)
	mr := miniredis.RunT(t)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()
	cache := NewDeviceCache(rdb, zap.NewNop())
	cache.Set(context.Background(), &model.Device{ID: deviceID, SerialNumber: sn})
	svc.SetDeviceCache(cache)

	_, hit := cache.Get(context.Background(), sn)
	require.True(t, hit, "前置：cache 应已写入")

	require.NoError(t, svc.DeleteDevice(context.Background(), deviceID))

	_, hit = cache.Get(context.Background(), sn)
	assert.False(t, hit, "DeleteDevice 后 cache 应被清掉")
}
