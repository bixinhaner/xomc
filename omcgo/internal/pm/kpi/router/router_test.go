package router

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/product"
)

// ── 测试夹具：mock 依赖 ───────────────────────────────────────────────

type fakeDevice struct {
	device *model.Device
	err    error
	calls  int
	mu     sync.Mutex
	called chan struct{}
}

func (f *fakeDevice) GetBySerialNumber(_ context.Context, _ string) (*model.Device, error) {
	f.mu.Lock()
	f.calls++
	device, err, called := f.device, f.err, f.called
	f.mu.Unlock()
	if called != nil {
		called <- struct{}{}
	}
	return device, err
}

type fakeProduct struct {
	match *product.MatchResult
	err   error
	calls int
	mu    sync.Mutex
}

func (f *fakeProduct) MatchProductClass(_ context.Context, _ string) (*product.MatchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.match, f.err
}

type fakeIndicators struct {
	rows  []*indicator.PerfIndicator
	err   error
	calls int
}

type blockingIndicators struct {
	mu      sync.Mutex
	rows    []*indicator.PerfIndicator
	calls   int
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

type versionBumpingIndicators struct {
	rows  []*indicator.PerfIndicator
	cache *RedisCache
	calls int
	bump  bool
}

func (f *versionBumpingIndicators) ListByIDs(ctx context.Context, _ indicator.DeviceType, _ []string) ([]*indicator.PerfIndicator, error) {
	f.calls++
	if f.bump {
		if _, err := f.cache.BumpVersion(ctx); err != nil {
			return nil, err
		}
	}
	return f.rows, nil
}

func (f *blockingIndicators) ListByIDs(_ context.Context, _ indicator.DeviceType, _ []string) ([]*indicator.PerfIndicator, error) {
	f.mu.Lock()
	f.calls++
	rows := append([]*indicator.PerfIndicator(nil), f.rows...)
	f.mu.Unlock()
	f.once.Do(func() {
		close(f.started)
		<-f.release
	})
	return rows, nil
}

func (f *blockingIndicators) setRows(rows []*indicator.PerfIndicator) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = rows
}

func (f *blockingIndicators) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeIndicators) ListByIDs(_ context.Context, _ indicator.DeviceType, _ []string) ([]*indicator.PerfIndicator, error) {
	f.calls++
	return f.rows, f.err
}

type fakeFormulas struct {
	rows  []*indicator.PlatformFormula
	err   error
	calls int
}

func (f *fakeFormulas) ListByPlatform(_ context.Context, _ indicator.DeviceType, _ string) ([]*indicator.PlatformFormula, error) {
	f.calls++
	return f.rows, f.err
}

type fakeL2 struct {
	store        map[uuid.UUID]*KPIRoute
	storeVersion map[uuid.UUID]int64
	getCalls     int
	putCalls     int
	getErr       error
	version      int64
	versionErr   error
}

func newFakeL2() *fakeL2 {
	return &fakeL2{
		store:        map[uuid.UUID]*KPIRoute{},
		storeVersion: map[uuid.UUID]int64{},
	}
}

func (f *fakeL2) Get(_ context.Context, productID uuid.UUID) (*KPIRoute, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	if r, ok := f.store[productID]; ok {
		if f.storeVersion[productID] != f.version {
			return nil, nil
		}
		return r, nil
	}
	return nil, nil
}

func (f *fakeL2) Put(_ context.Context, route *KPIRoute, version int64) error {
	f.putCalls++
	f.store[route.ProductID] = route
	f.storeVersion[route.ProductID] = version
	return nil
}

func (f *fakeL2) GetVersion(_ context.Context) (int64, error) { return f.version, f.versionErr }

// ── 公共夹具 helper ──────────────────────────────────────────────────

func sptr(s string) *string { return &s }

func newRouterWithFakes(t *testing.T, dev DeviceLookup, prod ProductMatcher, ind IndicatorLister, frm FormulaLister, l2 L2Cache) *Router {
	t.Helper()
	r, err := New(dev, prod, ind, frm, Options{L2Cache: l2, Logger: zap.NewNop()})
	require.NoError(t, err)
	return r
}

func newProductMatch(productID uuid.UUID, platform, deviceType string) *fakeProduct {
	return &fakeProduct{match: &product.MatchResult{
		Product: &product.Product{
			ID:                  productID,
			Name:                "test-product",
			IndicatorPlatform:   platform,
			IndicatorDeviceType: deviceType,
		},
	}}
}

func newBaseDevice(sn, productClass string) *fakeDevice {
	return &fakeDevice{device: &model.Device{SerialNumber: sn, ProductClass: productClass}}
}

func sampleIndicators() []*indicator.PerfIndicator {
	return []*indicator.PerfIndicator{
		{ID: "C000000001", EnName: "RRC_Conn_Att", IsCounter: "1", StatisType: sptr("sum"), ReportKey: sptr("RRC.AttConn")},
		{ID: "C000000002", EnName: "RRC_Conn_Succ", IsCounter: "1", StatisType: sptr("sum"), ReportKey: sptr("RRC.SuccConn")},
		// PM-P3：KPI 公式来源切到编号版 arithmetic（引用 counter 编号），不再用标准名 formula 表。
		{ID: "K-001", EnName: "RRC_Succ_Rate", IsCounter: "0", StatisType: sptr("pct"), Arithmetic: sptr("C000000002/C000000001*100")},
	}
}

func expandedIndicators() []*indicator.PerfIndicator {
	return append(sampleIndicators(),
		&indicator.PerfIndicator{ID: "C000000003", EnName: "TCH_Request", IsCounter: "1", StatisType: sptr("sum"), ReportKey: sptr("TCH.Request")},
		&indicator.PerfIndicator{ID: "K-002", EnName: "TCH_Success_Rate", IsCounter: "0", StatisType: sptr("pct"), Arithmetic: sptr("C000000002/C000000003*100")},
	)
}

func sampleFormulas() []*indicator.PlatformFormula {
	// PM-P3：formula 表仅决定「该平台支持哪些 KPI」的成员资格（uniqueIndicatorIDs → ListByIDs），
	// 公式内容已切到 arithmetic，故此处只需声明 K-001 被该平台引用即可（formula 内容不再被读取）。
	return []*indicator.PlatformFormula{
		{IndicatorID: "K-001", PlatformName: "BLQ-LTE-V1", Formula: "RRC_Conn_Succ / RRC_Conn_Att"},
	}
}

func expandedFormulas() []*indicator.PlatformFormula {
	return append(sampleFormulas(),
		&indicator.PlatformFormula{IndicatorID: "K-002", PlatformName: "BLQ-LTE-V1", Formula: "TCH_Success/TCH_Request"},
	)
}

// ── 5 个 TDD case ────────────────────────────────────────────────────

// Case 1: 全 miss 路径 — 走完 device → product → DB → 装配 KPIRoute，验证拆分语义正确。
func TestRouter_LookupByDevice_AllMissThenDBLoad(t *testing.T) {
	productID := uuid.New()
	r := newRouterWithFakes(t,
		newBaseDevice("SN-001", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		&fakeIndicators{rows: sampleIndicators()},
		&fakeFormulas{rows: sampleFormulas()},
		nil, // no L2
	)

	route, err := r.LookupByDevice(context.Background(), "SN-001")
	require.NoError(t, err)
	require.NotNil(t, route)
	require.Equal(t, productID, route.ProductID)
	require.Equal(t, indicator.DeviceTypeENB, route.IndicatorDeviceType)
	require.Equal(t, "BLQ-LTE-V1", route.IndicatorPlatform)
	require.Len(t, route.Counters, 2, "two is_counter='1' indicators should land in Counters")
	// PM-P2: assembleRoute 把 perf_indicators.report_key 填进 CounterDef.ReportKey，
	// 供解析侧白名单按 report_key 建键。
	reportKeyByID := map[string]string{}
	for _, cd := range route.Counters {
		reportKeyByID[cd.IndicatorID] = cd.ReportKey
	}
	require.Equal(t, "RRC.AttConn", reportKeyByID["C000000001"])
	require.Equal(t, "RRC.SuccConn", reportKeyByID["C000000002"])
	require.Len(t, route.KPIs, 1, "one is_counter='0' indicator with arithmetic should land in KPIs")
	require.Equal(t, "RRC_Succ_Rate", route.KPIs[0].Name)
	// PM-P3：Formula 为编号版 arithmetic（不再是标准名 formula）。
	require.Equal(t, "C000000002/C000000001*100", route.KPIs[0].Formula)
	// Dependencies 与 Formula 同源（从 arithmetic 提取），是 counter 编号。
	require.ElementsMatch(t, []string{"C000000002", "C000000001"}, route.KPIs[0].Dependencies)
}

// Case 2: L1 命中 — 第二次查询不再调用 DB / L2。
func TestRouter_LookupByDevice_L1Hit(t *testing.T) {
	productID := uuid.New()
	dev := newBaseDevice("SN-002", "FAPService.BLQ_LTE")
	prod := newProductMatch(productID, "BLQ-LTE-V1", "ENB")
	ind := &fakeIndicators{rows: sampleIndicators()}
	frm := &fakeFormulas{rows: sampleFormulas()}
	l2 := newFakeL2()
	r := newRouterWithFakes(t, dev, prod, ind, frm, l2)

	_, err := r.LookupByDevice(context.Background(), "SN-002")
	require.NoError(t, err)
	require.Equal(t, 1, ind.calls, "first lookup → 1 DB indicator call")

	_, err = r.LookupByDevice(context.Background(), "SN-002")
	require.NoError(t, err)
	require.Equal(t, 1, ind.calls, "second lookup → L1 hit, no DB indicator call")
	require.Equal(t, 1, frm.calls, "second lookup → no formula DB call either")
	require.Equal(t, 1, l2.getCalls, "L1 hit shouldn't fall through to L2 either")
}

// 指标 Loader 已有测试证明 reload 成功会调用注入的 bumper；这里从该 bumper seam
// 继续验证 app/worker 两个 Router 共享 Redis 时，worker 的旧 L1 会在下一次查询失效。
func Test_LookupByDevice_IndicatorReloadBumpInvalidatesWorkerL1(t *testing.T) {
	productID := uuid.New()
	ind := &fakeIndicators{rows: sampleIndicators()}
	frm := &fakeFormulas{rows: sampleFormulas()}
	workerCache, appCache := newRedisCachePair(t)
	workerRouter := newRouterWithFakes(t,
		newBaseDevice("SN-REMOTE-BUMP", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		ind,
		frm,
		workerCache,
	)
	appRouter := newRouterWithFakes(t,
		newBaseDevice("SN-REMOTE-BUMP", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		ind,
		frm,
		appCache,
	)

	oldRoute, err := workerRouter.LookupByDevice(context.Background(), "SN-REMOTE-BUMP")
	require.NoError(t, err)
	require.Len(t, oldRoute.Counters, 2)
	require.Len(t, oldRoute.KPIs, 1)

	ind.rows = expandedIndicators()
	frm.rows = expandedFormulas()
	indicatorReloadBump := func(ctx context.Context) error {
		_, bumpErr := appCache.BumpVersion(ctx)
		return bumpErr
	}
	require.NoError(t, indicatorReloadBump(context.Background()))

	appRoute, err := appRouter.LookupByDevice(context.Background(), "SN-REMOTE-BUMP")
	require.NoError(t, err)
	require.Len(t, appRoute.Counters, 3)
	require.Len(t, appRoute.KPIs, 2)

	newRoute, err := workerRouter.LookupByDevice(context.Background(), "SN-REMOTE-BUMP")
	require.NoError(t, err)
	require.Len(t, newRoute.Counters, 3, "remote bump must evict the stale counter whitelist")
	require.Len(t, newRoute.KPIs, 2, "remote bump must expose newly registered KPIs")
}

func Test_LookupByDevice_VersionReadFailureFailsOpenAndRecovers(t *testing.T) {
	productID := uuid.New()
	ind := &fakeIndicators{rows: sampleIndicators()}
	frm := &fakeFormulas{rows: sampleFormulas()}
	cache := newFakeL2()
	reg := prometheus.NewRegistry()
	r, err := New(
		newBaseDevice("SN-REDIS-RECOVERY", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		ind,
		frm,
		Options{L2Cache: cache, Metrics: NewMetrics(reg), Logger: zap.NewNop()},
	)
	require.NoError(t, err)

	oldRoute, err := r.LookupByDevice(context.Background(), "SN-REDIS-RECOVERY")
	require.NoError(t, err)
	require.Len(t, oldRoute.Counters, 2)

	ind.rows = append(sampleIndicators(),
		&indicator.PerfIndicator{ID: "C000000003", EnName: "TCH_Request", IsCounter: "1", StatisType: sptr("sum"), ReportKey: sptr("TCH.Request")},
	)
	cache.version = 1
	cache.versionErr = errors.New("redis unavailable")

	staleRoute, err := r.LookupByDevice(context.Background(), "SN-REDIS-RECOVERY")
	require.NoError(t, err, "Redis version failure must not interrupt PM route lookup")
	require.Len(t, staleRoute.Counters, 2, "fail-open keeps the current complete route snapshot")

	cache.versionErr = nil
	newRoute, err := r.LookupByDevice(context.Background(), "SN-REDIS-RECOVERY")
	require.NoError(t, err)
	require.Len(t, newRoute.Counters, 3, "the first lookup after Redis recovery must adopt the new version")

	wantMetrics := `
# HELP omc_kpi_router_cache_version_read_failures_total KPI Router cache version reads that failed open.
# TYPE omc_kpi_router_cache_version_read_failures_total counter
omc_kpi_router_cache_version_read_failures_total 1
# HELP omc_kpi_router_stale_evictions_total KPI Router L1 evictions caused by cache version changes.
# TYPE omc_kpi_router_stale_evictions_total counter
omc_kpi_router_stale_evictions_total 1
`
	require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(wantMetrics),
		"omc_kpi_router_cache_version_read_failures_total",
		"omc_kpi_router_stale_evictions_total",
	))
}

func Test_LookupByDevice_VersionRollbackAlsoInvalidatesL1(t *testing.T) {
	productID := uuid.New()
	ind := &fakeIndicators{rows: sampleIndicators()}
	cache := newFakeL2()
	cache.version = 2
	r := newRouterWithFakes(t,
		newBaseDevice("SN-VERSION-ROLLBACK", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		ind,
		&fakeFormulas{rows: sampleFormulas()},
		cache,
	)

	oldRoute, err := r.LookupByDevice(context.Background(), "SN-VERSION-ROLLBACK")
	require.NoError(t, err)
	require.Len(t, oldRoute.Counters, 2)

	ind.rows = append(sampleIndicators(),
		&indicator.PerfIndicator{ID: "C000000003", EnName: "TCH_Request", IsCounter: "1", StatisType: sptr("sum"), ReportKey: sptr("TCH.Request")},
	)
	cache.version = 1 // Redis restore/rebuild can move the observed version backwards.

	newRoute, err := r.LookupByDevice(context.Background(), "SN-VERSION-ROLLBACK")
	require.NoError(t, err)
	require.Len(t, newRoute.Counters, 3)
	require.Equal(t, 2, ind.calls)
}

func Test_LookupByDevice_VersionChangeDuringDBLoadRetriesBeforeCaching(t *testing.T) {
	productID := uuid.New()
	cache := newRedisHarness(t)
	_, err := cache.BumpVersion(context.Background()) // start the lookup on version 1
	require.NoError(t, err)

	ind := &blockingIndicators{
		rows:    sampleIndicators(),
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	r := newRouterWithFakes(t,
		newBaseDevice("SN-BUMP-DURING-LOAD", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		ind,
		&fakeFormulas{rows: sampleFormulas()},
		cache,
	)

	type lookupResult struct {
		route *KPIRoute
		err   error
	}
	resultCh := make(chan lookupResult, 1)
	go func() {
		route, lookupErr := r.LookupByDevice(context.Background(), "SN-BUMP-DURING-LOAD")
		resultCh <- lookupResult{route: route, err: lookupErr}
	}()

	<-ind.started
	ind.setRows(append(sampleIndicators(),
		&indicator.PerfIndicator{ID: "C000000003", EnName: "TCH_Request", IsCounter: "1", StatisType: sptr("sum"), ReportKey: sptr("TCH.Request")},
	))
	_, err = cache.BumpVersion(context.Background()) // committed metadata now belongs to version 2
	require.NoError(t, err)
	close(ind.release)

	result := <-resultCh
	require.NoError(t, result.err)
	require.Len(t, result.route.Counters, 3, "a route built under version 1 must not be published as version 2")
	require.Equal(t, 2, ind.calls, "the stale DB result must be discarded and rebuilt once")

	// The stable result must also be what subsequent L1 lookups expose.
	again, err := r.LookupByDevice(context.Background(), "SN-BUMP-DURING-LOAD")
	require.NoError(t, err)
	require.Len(t, again.Counters, 3)
}

func Test_LookupByDevice_ContinuousVersionChangesStopAfterBoundedRetries(t *testing.T) {
	productID := uuid.New()
	cache := newRedisHarness(t)
	ind := &versionBumpingIndicators{rows: sampleIndicators(), cache: cache, bump: true}
	r := newRouterWithFakes(t,
		newBaseDevice("SN-CONTINUOUS-BUMP", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		ind,
		&fakeFormulas{rows: sampleFormulas()},
		cache,
	)

	route, err := r.LookupByDevice(context.Background(), "SN-CONTINUOUS-BUMP")
	require.Nil(t, route)
	require.ErrorContains(t, err, "changed during 3 consecutive load attempts")
	require.Equal(t, maxRouteLoadAttempts, ind.calls)

	// No unstable result may have reached L1: a later stable metadata read must hit DB again.
	ind.bump = false
	route, err = r.LookupByDevice(context.Background(), "SN-CONTINUOUS-BUMP")
	require.NoError(t, err)
	require.NotNil(t, route)
	require.Equal(t, maxRouteLoadAttempts+1, ind.calls)
}

func Test_LookupByDevice_ConcurrentMissCoalescesRouteLoadAndWaiterCanCancel(t *testing.T) {
	productID := uuid.New()
	ind := &blockingIndicators{
		rows:    sampleIndicators(),
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	dev := newBaseDevice("SN-CONCURRENT", "FAPService.BLQ_LTE")
	dev.called = make(chan struct{}, 2)
	r := newRouterWithFakes(t,
		dev,
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		ind,
		&fakeFormulas{rows: sampleFormulas()},
		newFakeL2(),
	)

	type lookupResult struct {
		route *KPIRoute
		err   error
	}
	leaderCh := make(chan lookupResult, 1)
	go func() {
		route, err := r.LookupByDevice(context.Background(), "SN-CONCURRENT")
		leaderCh <- lookupResult{route: route, err: err}
	}()
	<-dev.called
	<-ind.started

	waiterCtx, cancelWaiter := context.WithCancel(context.Background())
	waiterCh := make(chan lookupResult, 1)
	go func() {
		route, err := r.LookupByDevice(waiterCtx, "SN-CONCURRENT")
		waiterCh <- lookupResult{route: route, err: err}
	}()
	<-dev.called
	cancelWaiter()

	select {
	case waiter := <-waiterCh:
		require.ErrorIs(t, waiter.err, context.Canceled)
		require.Nil(t, waiter.route)
	case <-time.After(time.Second):
		close(ind.release)
		<-leaderCh
		t.Fatal("a coalesced waiter did not respond to its own context cancellation")
	}
	require.Equal(t, 1, ind.callCount(), "concurrent misses for one product must share one route rebuild")

	close(ind.release)
	leader := <-leaderCh
	require.NoError(t, leader.err)
	require.Len(t, leader.route.Counters, 2)
}

// Case 3: L1 miss / L2 hit — DB 不应被调用。
func TestRouter_LookupByDevice_L2HitSkipsDB(t *testing.T) {
	productID := uuid.New()
	dev := newBaseDevice("SN-003", "FAPService.BLQ_LTE")
	prod := newProductMatch(productID, "BLQ-LTE-V1", "ENB")
	ind := &fakeIndicators{rows: sampleIndicators()}
	frm := &fakeFormulas{rows: sampleFormulas()}
	l2 := newFakeL2()
	l2.store[productID] = &KPIRoute{
		ProductID:           productID,
		IndicatorPlatform:   "BLQ-LTE-V1",
		IndicatorDeviceType: indicator.DeviceTypeENB,
		KPIs:                []KPIDef{{IndicatorID: "K-001", Name: "RRC_Succ_Rate"}},
	}
	r := newRouterWithFakes(t, dev, prod, ind, frm, l2)

	route, err := r.LookupByDevice(context.Background(), "SN-003")
	require.NoError(t, err)
	require.NotNil(t, route)
	require.Equal(t, "RRC_Succ_Rate", route.KPIs[0].Name)
	require.Equal(t, 0, ind.calls, "L2 hit should skip DB indicator")
	require.Equal(t, 0, frm.calls, "L2 hit should skip DB formula")
	require.Equal(t, 1, l2.getCalls)
}

// Case 4: productClass 未匹配 — 返回 ErrProductNotMatched，DB / L2 都不应被触发。
func TestRouter_LookupByDevice_OrphanProductClass(t *testing.T) {
	dev := newBaseDevice("SN-004", "Unknown.Vendor.Model")
	prod := &fakeProduct{err: product.ErrOrphan}
	ind := &fakeIndicators{rows: sampleIndicators()}
	frm := &fakeFormulas{rows: sampleFormulas()}
	r := newRouterWithFakes(t, dev, prod, ind, frm, nil)

	route, err := r.LookupByDevice(context.Background(), "SN-004")
	require.Nil(t, route)
	require.ErrorIs(t, err, ErrProductNotMatched)
	require.Equal(t, 0, ind.calls)
	require.Equal(t, 0, frm.calls)
}

// Case 5: product 命中但 platform 无 formula 行 — 返回空 KPIRoute（log warn），不抛错。
func TestRouter_LookupByDevice_PlatformNoFormula(t *testing.T) {
	productID := uuid.New()
	r := newRouterWithFakes(t,
		newBaseDevice("SN-005", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		&fakeIndicators{rows: nil}, // 无 indicator 行（platform 匹配空）
		&fakeFormulas{rows: nil},   // 无 formula 行
		nil,
	)

	route, err := r.LookupByDevice(context.Background(), "SN-005")
	require.NoError(t, err)
	require.NotNil(t, route)
	require.Equal(t, productID, route.ProductID)
	require.Empty(t, route.Counters)
	require.Empty(t, route.KPIs)
}

// PM-P3 Case: is_counter='0' 但 arithmetic 为空 → 跳过 + warn，不进 KPIs。
func TestRouter_LookupByDevice_KPIEmptyArithmeticSkipped(t *testing.T) {
	productID := uuid.New()
	r := newRouterWithFakes(t,
		newBaseDevice("SN-A01", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		&fakeIndicators{rows: []*indicator.PerfIndicator{
			{ID: "C000000001", EnName: "RRC_Conn_Att", IsCounter: "1", StatisType: sptr("sum"), ReportKey: sptr("RRC.AttConn")},
			// KPI 但 arithmetic 为空（运维漏配编号公式）→ 必须被跳过。
			{ID: "K-NIL", EnName: "Broken_KPI", IsCounter: "0", StatisType: sptr("pct"), Arithmetic: nil},
			// KPI arithmetic 为空串 → 同样跳过。
			{ID: "K-EMPTY", EnName: "Broken_KPI2", IsCounter: "0", StatisType: sptr("pct"), Arithmetic: sptr("")},
		}},
		&fakeFormulas{rows: []*indicator.PlatformFormula{
			{IndicatorID: "K-NIL", PlatformName: "BLQ-LTE-V1", Formula: "x/y"},
			{IndicatorID: "K-EMPTY", PlatformName: "BLQ-LTE-V1", Formula: "x/y"},
		}},
		nil,
	)

	route, err := r.LookupByDevice(context.Background(), "SN-A01")
	require.NoError(t, err)
	require.NotNil(t, route)
	require.Len(t, route.Counters, 1)
	require.Empty(t, route.KPIs, "arithmetic 为空的 KPI 被跳过，不产生 KPIDef")
}

// ── 衍生健壮性 case（非 plan 5 个，但顺手写一下确保边界）──────────

// product 元数据残缺（IndicatorPlatform 空）→ 返回空路由 + warn，不抛错。
func TestRouter_LookupByDevice_MissingProductMetadata(t *testing.T) {
	productID := uuid.New()
	prod := &fakeProduct{match: &product.MatchResult{
		Product: &product.Product{ID: productID, Name: "incomplete"}, // 无 IndicatorPlatform / IndicatorDeviceType
	}}
	r := newRouterWithFakes(t,
		newBaseDevice("SN-006", "FAPService.BLQ_LTE"),
		prod,
		&fakeIndicators{rows: sampleIndicators()},
		&fakeFormulas{rows: sampleFormulas()},
		nil,
	)

	route, err := r.LookupByDevice(context.Background(), "SN-006")
	require.NoError(t, err)
	require.NotNil(t, route)
	require.Empty(t, route.Counters)
	require.Empty(t, route.KPIs)
}

// DB 返回错误 — Router 应当透传。
func TestRouter_LookupByDevice_DBErrorPropagates(t *testing.T) {
	productID := uuid.New()
	wantErr := errors.New("db down")
	r := newRouterWithFakes(t,
		newBaseDevice("SN-007", "FAPService.BLQ_LTE"),
		newProductMatch(productID, "BLQ-LTE-V1", "ENB"),
		&fakeIndicators{},
		&fakeFormulas{err: wantErr},
		nil,
	)

	_, err := r.LookupByDevice(context.Background(), "SN-007")
	require.ErrorIs(t, err, wantErr)
}
