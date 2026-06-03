package router

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
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
}

func (f *fakeDevice) GetBySerialNumber(_ context.Context, _ string) (*model.Device, error) {
	f.calls++
	return f.device, f.err
}

type fakeProduct struct {
	match *product.MatchResult
	err   error
	calls int
}

func (f *fakeProduct) MatchProductClass(_ context.Context, _ string) (*product.MatchResult, error) {
	f.calls++
	return f.match, f.err
}

type fakeIndicators struct {
	rows  []*indicator.PerfIndicator
	err   error
	calls int
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
	store    map[uuid.UUID]*KPIRoute
	getCalls int
	putCalls int
	getErr   error
}

func newFakeL2() *fakeL2 { return &fakeL2{store: map[uuid.UUID]*KPIRoute{}} }

func (f *fakeL2) Get(_ context.Context, productID uuid.UUID) (*KPIRoute, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	if r, ok := f.store[productID]; ok {
		return r, nil
	}
	return nil, nil
}

func (f *fakeL2) Put(_ context.Context, route *KPIRoute) error {
	f.putCalls++
	f.store[route.ProductID] = route
	return nil
}

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
		{ID: "C-001", EnName: "RRC_Conn_Att", IsCounter: "1", StatisType: sptr("sum"), ReportKey: sptr("RRC.AttConn")},
		{ID: "C-002", EnName: "RRC_Conn_Succ", IsCounter: "1", StatisType: sptr("sum"), ReportKey: sptr("RRC.SuccConn")},
		{ID: "K-001", EnName: "RRC_Succ_Rate", IsCounter: "0", StatisType: sptr("pct")},
	}
}

func sampleFormulas() []*indicator.PlatformFormula {
	return []*indicator.PlatformFormula{
		// 注意：is_counter='1' 的 counter 通常 platform formula 表里也会有一条占位行
		// （indicator 同时被多个 platform 引用 / loader 行为），这里只放 KPI 一条即可。
		{IndicatorID: "K-001", PlatformName: "BLQ-LTE-V1", Formula: "RRC_Conn_Succ / RRC_Conn_Att"},
	}
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
	require.Equal(t, "RRC.AttConn", reportKeyByID["C-001"])
	require.Equal(t, "RRC.SuccConn", reportKeyByID["C-002"])
	require.Len(t, route.KPIs, 1, "one is_counter='0' indicator with formula should land in KPIs")
	require.Equal(t, "RRC_Succ_Rate", route.KPIs[0].Name)
	require.Equal(t, "RRC_Conn_Succ / RRC_Conn_Att", route.KPIs[0].Formula)
	require.ElementsMatch(t, []string{"RRC_Conn_Succ", "RRC_Conn_Att"}, route.KPIs[0].Dependencies)
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
