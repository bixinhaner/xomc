package pm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/calendarfilter"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock repositories (pmH prefix to avoid collision with threshold_handler_test)
// ---------------------------------------------------------------------------

type pmHCounterRepo struct {
	queryFn           func(ctx context.Context, filter counter.CounterFilter) (*model.ListResponse[model.PMCounter], error)
	queryAggregatedFn func(ctx context.Context, filter counter.CounterFilter) ([]counter.AggregatedCounter, error)
}

func (m *pmHCounterRepo) BatchInsert(_ context.Context, _ []model.PMCounter) error { return nil }
func (m *pmHCounterRepo) Query(ctx context.Context, filter counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, filter)
	}
	return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
}
func (m *pmHCounterRepo) QueryAggregated(ctx context.Context, filter counter.CounterFilter) ([]counter.AggregatedCounter, error) {
	if m.queryAggregatedFn != nil {
		return m.queryAggregatedFn(ctx, filter)
	}
	return nil, nil
}
func (m *pmHCounterRepo) QueryForKPI(_ context.Context, _ uuid.UUID, _ string, _ []string, _, _ time.Time) (map[string]float64, error) {
	return nil, nil
}
func (m *pmHCounterRepo) QueryForKPICells(_ context.Context, _ uuid.UUID, _ []string, _ []string, _, _ time.Time) (map[string]map[string]float64, error) {
	return nil, nil
}

type pmHKPIRepo struct {
	queryFn func(ctx context.Context, filter kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error)
}

func (m *pmHKPIRepo) BatchInsert(_ context.Context, _ []model.KPIValue) error { return nil }
func (m *pmHKPIRepo) ReplaceForRecompute(_ context.Context, _, _, _ string, _ time.Time, _ []model.KPIValue) error {
	return nil
}
func (m *pmHKPIRepo) Query(ctx context.Context, filter kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, filter)
	}
	return model.NewListResponse([]model.KPIValue{}, 0, 1, 20), nil
}
func (m *pmHKPIRepo) ListDefinitions(_ context.Context, _ *model.CarrierCode, _ *model.Technology) ([]model.KPIDefinition, error) {
	return nil, nil
}
func (m *pmHKPIRepo) SyncDefinitions(_ context.Context, _ []model.KPIDefinition) error { return nil }

type pmHTaskRepo struct {
	listFn   func(ctx context.Context, filter TaskFilter) (*model.ListResponse[PerformanceTask], error)
	createFn func(ctx context.Context, task *PerformanceTask) error
}

func (m *pmHTaskRepo) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[PerformanceTask], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return model.NewListResponse([]PerformanceTask{}, 0, 1, 20), nil
}
func (m *pmHTaskRepo) Create(ctx context.Context, task *PerformanceTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	return nil
}

// pmHIndicatorRepo 是 ListKPIDefinitions 用的最小 mock：只实现 ListAll，
// 其余方法借接口嵌入留空（本测试不会触发，触发即 nil 解引用 panic 暴露误用）。
type pmHIndicatorRepo struct {
	indicator.IndicatorRepository
	listAllFn func(ctx context.Context, filter indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error)
}

func (m *pmHIndicatorRepo) ListAll(ctx context.Context, filter indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx, filter)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// pmHStubRouter 是测试夹具用的 KPIRouter 实现：永远返"产品未匹配"，让 Engine 跳过 KPI 计算。
// 测试 handler 路由层不关心 KPI 值，需要 router 不抛硬错。
type pmHStubRouter struct{}

func (pmHStubRouter) LookupByDevice(_ context.Context, _ string) (*router.KPIRoute, error) {
	return nil, router.ErrProductNotMatched
}

func pmHNewEngine() *kpi.KPIEngine {
	return kpi.NewKPIEngine(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHStubRouter{}, zap.NewNop())
}

func pmHSetupRouter(cr counter.CounterRepository, kr kpi.KPIRepository, engine *kpi.KPIEngine, tr TaskRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// indicatorRepo 传 nil → ListKPIDefinitions 走退化路径返空集合。
	h := NewHandler(cr, kr, engine, tr, nil, nil, "pm-files", nil, nil, zap.NewNop())
	h.RegisterRoutes(r.Group(""))
	return r
}

// pmHSetupRouterWithIndicator 与 pmHSetupRouter 一致，但注入指定 indicatorRepo，
// 供 ListKPIDefinitions 的非退化路径测试用。
func pmHSetupRouterWithIndicator(ir indicator.IndicatorRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{}, nil, nil, "pm-files", nil, ir, zap.NewNop())
	h.RegisterRoutes(r.Group(""))
	return r
}

func pmHSetupRouterWithAggregator(aggr *aggregator.Aggregator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{}, nil, nil, "pm-files", nil, nil, zap.NewNop())
	h.WithAggregator(aggr)
	h.RegisterRoutes(r.Group(""))
	return r
}

func pmHSetupRouterWithAggregatedBackend(backend aggregatedMetricsBackend) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{}, nil, nil, "pm-files", nil, nil, zap.NewNop())
	h.withAggregatedMetricsBackend(backend)
	h.RegisterRoutes(r.Group(""))
	return r
}

type pmHAggregatedBackend struct {
	queryRows     []aggregator.Row
	discoverLDNs  []string
	pivotRowKeys  []aggregator.PivotRowKey
	queryReqs     []aggregator.QueryRequest
	countReqs     []aggregator.QueryRequest
	discoverReqs  []aggregator.QueryRequest
	pivotKeyReqs  []aggregator.QueryRequest
	backfillCalls int
}

func (m *pmHAggregatedBackend) Query(_ context.Context, req aggregator.QueryRequest) ([]aggregator.Row, error) {
	m.queryReqs = append(m.queryReqs, req)
	return m.queryRows, nil
}

func (m *pmHAggregatedBackend) Count(_ context.Context, req aggregator.QueryRequest) (int, error) {
	m.countReqs = append(m.countReqs, req)
	return len(m.queryRows), nil
}

func (m *pmHAggregatedBackend) DiscoverObjectLDNs(_ context.Context, req aggregator.QueryRequest) ([]string, error) {
	m.discoverReqs = append(m.discoverReqs, req)
	return m.discoverLDNs, nil
}

func (m *pmHAggregatedBackend) DevicePivotRowKeys(_ context.Context, req aggregator.QueryRequest) ([]aggregator.PivotRowKey, error) {
	m.pivotKeyReqs = append(m.pivotKeyReqs, req)
	return m.pivotRowKeys, nil
}

func (m *pmHAggregatedBackend) BackfillDisplayNames(_ context.Context, rows []aggregator.Row) {
	m.backfillCalls++
	for i := range rows {
		if rows[i].DisplayName == "" {
			rows[i].DisplayName = rows[i].MetricPath
		}
	}
}

func (m *pmHAggregatedBackend) SetTimezoneProvider(calendarfilter.TimezoneProvider) {}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ListCounters_Default(t *testing.T) {
	cr := &pmHCounterRepo{
		queryFn: func(_ context.Context, f counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			assert.Equal(t, 1, f.Page)
			assert.Equal(t, 20, f.PageSize)
			return model.NewListResponse([]model.PMCounter{
				{CounterName: "rrc_conn_setup_att", CounterValue: 1000},
			}, 1, 1, 20), nil
		},
	}
	router := pmHSetupRouter(cr, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[model.PMCounter]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
}

func TestHandler_ListCounters_WithDeviceFilter(t *testing.T) {
	deviceID := uuid.New()
	cr := &pmHCounterRepo{
		queryFn: func(_ context.Context, f counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			require.NotNil(t, f.DeviceID)
			assert.Equal(t, deviceID, *f.DeviceID)
			return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
		},
	}
	router := pmHSetupRouter(cr, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters?device_id="+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListCounters_InvalidDeviceID(t *testing.T) {
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters?device_id=bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListAggregatedCounters(t *testing.T) {
	cr := &pmHCounterRepo{
		queryAggregatedFn: func(_ context.Context, _ counter.CounterFilter) ([]counter.AggregatedCounter, error) {
			return []counter.AggregatedCounter{
				{CounterName: "rrc_conn_setup_att", SumValue: 5000, AvgValue: 1000},
			}, nil
		},
	}
	router := pmHSetupRouter(cr, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters/aggregated?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []counter.AggregatedCounter `json:"items"`
	}
	response.DecodeData(t, w.Body, &body)
	require.Len(t, body.Items, 1)
	assert.Equal(t, float64(5000), body.Items[0].SumValue)
}

// TestHandler_ListAggregatedCounters_NoPagination 回归 issue#126 第4项：
// 不带 page/page_size 时不应 400（Page min 校验失败），应与 ListCounters 一致
// 预填 DefaultListRequest 后正常 200。该端点本身不消费分页字段。
func TestHandler_ListAggregatedCounters_NoPagination(t *testing.T) {
	cr := &pmHCounterRepo{
		queryAggregatedFn: func(_ context.Context, _ counter.CounterFilter) ([]counter.AggregatedCounter, error) {
			return []counter.AggregatedCounter{
				{CounterName: "rrc_conn_setup_att", SumValue: 5000, AvgValue: 1000},
			}, nil
		},
	}
	router := pmHSetupRouter(cr, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters/aggregated", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []counter.AggregatedCounter `json:"items"`
	}
	response.DecodeData(t, w.Body, &body)
	require.Len(t, body.Items, 1)
	assert.Equal(t, float64(5000), body.Items[0].SumValue)
}

func TestHandler_ListAggregatedMetrics_RejectsTooManyDeviceSNs(t *testing.T) {
	router := pmHSetupRouterWithAggregator(&aggregator.Aggregator{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(
		http.MethodGet,
		"/pm/metrics/aggregated?granularity=15min&device_sns="+pmHJoinStrings("SN", 51)+"&metric_paths=K1",
		nil,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListAggregatedMetrics_RejectsTooManyMetricPaths(t *testing.T) {
	router := pmHSetupRouterWithAggregator(&aggregator.Aggregator{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(
		http.MethodGet,
		"/pm/metrics/aggregated?granularity=15min&device_sns=SN-1&metric_paths="+pmHJoinStrings("K", 51),
		nil,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListAggregatedMetrics_FillEmptyAutoDiscoverUsesSingleObjectDiscovery(t *testing.T) {
	bucket := time.Date(2026, 8, 23, 1, 0, 0, 0, time.UTC)
	objectLDN := "Cellid=66"
	backend := &pmHAggregatedBackend{
		discoverLDNs: []string{objectLDN},
		pivotRowKeys: []aggregator.PivotRowKey{{
			DeviceOUI:   "48BF74",
			DeviceSN:    "SN-1",
			ObjectLDN:   objectLDN,
			Granularity: metrics.Granularity15Min,
			Time:        bucket,
		}},
		queryRows: []aggregator.Row{{
			DeviceOUI:   "48BF74",
			DeviceSN:    "SN-1",
			MetricPath:  "K1",
			MetricType:  metrics.MetricTypeKPI,
			MetricValue: 1,
			Granularity: metrics.Granularity15Min,
			Time:        bucket,
			StartTime:   bucket,
			EndTime:     bucket.Add(15 * time.Minute),
			ObjectLDN:   &objectLDN,
		}},
	}
	router := pmHSetupRouterWithAggregatedBackend(backend)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(
		http.MethodGet,
		"/pm/metrics/aggregated?granularity=15min&device_sn=SN-1&metric_paths=K1,K2&start_time=2026-08-23T01:00:00Z&end_time=2026-08-23T01:15:00Z&page_by=pivot_row&count_mode=n_plus_one&fill_empty=true&limit=5000",
		nil,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, backend.discoverReqs, 1)
	require.Len(t, backend.pivotKeyReqs, 1)
	require.Len(t, backend.queryReqs, 1)
	assert.Empty(t, backend.countReqs, "count_mode=n_plus_one must avoid exact Count")
	assert.Equal(t, []string{objectLDN}, backend.pivotKeyReqs[0].ObjectLDNs)
	assert.Equal(t, []string{objectLDN}, backend.queryReqs[0].ObjectLDNs)
	assert.Len(t, backend.queryReqs[0].PivotRowKeys, 1)
	assert.Equal(t, 1, backend.backfillCalls)

	var body struct {
		Items     []aggregator.Row `json:"items"`
		Total     int              `json:"total"`
		Truncated bool             `json:"truncated"`
	}
	response.DecodeData(t, w.Body, &body)
	require.Len(t, body.Items, 2)
	assert.Equal(t, 2, body.Total)
	assert.False(t, body.Truncated)
	assert.True(t, body.Items[1].Filled)
	assert.Equal(t, "K2", body.Items[1].MetricPath)
}

func TestHandler_ListAggregatedMetrics_EmptyAutoDiscoverDoesNotRepeatDiscovery(t *testing.T) {
	backend := &pmHAggregatedBackend{}
	router := pmHSetupRouterWithAggregatedBackend(backend)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(
		http.MethodGet,
		"/pm/metrics/aggregated?granularity=15min&device_sn=SN-1&metric_paths=K1,K2&start_time=2026-08-23T01:00:00Z&end_time=2026-08-23T01:15:00Z&page_by=pivot_row&count_mode=n_plus_one&fill_empty=true&limit=5000",
		nil,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, backend.discoverReqs, 1)
	assert.Empty(t, backend.pivotKeyReqs)
	require.Len(t, backend.queryReqs, 1)
	assert.Empty(t, backend.countReqs, "count_mode=n_plus_one must avoid exact Count")
	assert.Equal(t, 1, backend.backfillCalls)
}

func TestTruncateAggregatedRows_NPlusOnePlainRows(t *testing.T) {
	rows := []aggregator.Row{
		{DeviceSN: "SN-1", MetricPath: "K1"},
		{DeviceSN: "SN-1", MetricPath: "K2"},
		{DeviceSN: "SN-1", MetricPath: "K3"},
	}

	got, truncated := truncateAggregatedRows(rows, 2, false)

	require.True(t, truncated)
	require.Len(t, got, 2)
	assert.Equal(t, "K1", got[0].MetricPath)
	assert.Equal(t, "K2", got[1].MetricPath)
}

func TestTruncateAggregatedRows_NPlusOnePivotRowsKeepsFullMetricSet(t *testing.T) {
	t1 := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 7, 24, 9, 45, 0, 0, time.UTC)
	ldn := "Cellid=1"
	rows := []aggregator.Row{
		{DeviceOUI: "48BF74", DeviceSN: "SN-1", ObjectLDN: &ldn, Granularity: metrics.Granularity15Min, Time: t1, MetricPath: "K1"},
		{DeviceOUI: "48BF74", DeviceSN: "SN-1", ObjectLDN: &ldn, Granularity: metrics.Granularity15Min, Time: t1, MetricPath: "K2"},
		{DeviceOUI: "48BF74", DeviceSN: "SN-1", ObjectLDN: &ldn, Granularity: metrics.Granularity15Min, Time: t2, MetricPath: "K1"},
	}

	got, truncated := truncateAggregatedRows(rows, 1, true)

	require.True(t, truncated)
	require.Len(t, got, 2)
	assert.Equal(t, []string{"K1", "K2"}, []string{got[0].MetricPath, got[1].MetricPath})
}

func TestHandler_ListKPIValues(t *testing.T) {
	kr := &pmHKPIRepo{
		queryFn: func(_ context.Context, _ kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
			return model.NewListResponse([]model.KPIValue{
				{KPIName: "rrc_success_rate", KPIValue: 95.0},
			}, 1, 1, 20), nil
		},
	}
	router := pmHSetupRouter(&pmHCounterRepo{}, kr, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[model.KPIValue]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
}

func TestHandler_ListKPIDefinitions(t *testing.T) {
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []model.KPIDefinition `json:"items"`
		Total int                   `json:"total"`
	}
	response.DecodeData(t, w.Body, &body)
	assert.Equal(t, 0, body.Total)
}

// kpiDefRespItem 镜像 handler 的响应 wire 形态（含 id / is_counter + 历史字段）。
type kpiDefRespItem struct {
	ID             string `json:"id"`
	IsCounter      string `json:"is_counter"`
	IndicatorLevel string `json:"indicator_level"`
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Formula        string `json:"formula"`
	Unit           string `json:"unit"`
}

// 默认（不带 include_counters）：仅请求 is_counter='0' 的 KPI，响应含 id/is_counter，
// 历史字段（name=en_name / display_name=cn_name / formula / unit）保持不变 → 零回归。
func TestHandler_ListKPIDefinitions_DefaultKPIOnly(t *testing.T) {
	var captured indicator.IndicatorListFilter
	ir := &pmHIndicatorRepo{
		listAllFn: func(_ context.Context, f indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error) {
			captured = f
			return []indicator.IndicatorListItem{
				{PerfIndicator: indicator.PerfIndicator{
					ID: "K1001", EnName: "rrc_succ_rate", CnName: strPtr("RRC连接建立成功率"),
					IsCounter: "0", Arithmetic: strPtr("a/b"), UnitID: strPtr("3"),
				}},
			}, nil
		},
	}
	router := pmHSetupRouterWithIndicator(ir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions?device_type=ENB", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// 默认必须按 is_counter='0' 过滤。
	require.NotNil(t, captured.IsCounter)
	assert.Equal(t, "0", *captured.IsCounter)
	assert.Equal(t, "ENB", captured.DeviceType)

	var body struct {
		Items []kpiDefRespItem `json:"items"`
		Total int              `json:"total"`
	}
	response.DecodeData(t, w.Body, &body)
	assert.Equal(t, 1, body.Total)
	require.Len(t, body.Items, 1)
	it := body.Items[0]
	assert.Equal(t, "K1001", it.ID)
	assert.Equal(t, "0", it.IsCounter)
	assert.Equal(t, "rrc_succ_rate", it.Name)
	assert.Equal(t, "RRC连接建立成功率", it.DisplayName)
	assert.Equal(t, "a/b", it.Formula)
	assert.Equal(t, "3", it.Unit)
}

func TestHandler_ListKPIDefinitions_EnglishLocale(t *testing.T) {
	ir := &pmHIndicatorRepo{
		listAllFn: func(_ context.Context, _ indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error) {
			return []indicator.IndicatorListItem{
				{PerfIndicator: indicator.PerfIndicator{
					ID: "K1001", EnName: "RRC Setup Success Rate", CnName: strPtr("RRC连接建立成功率"),
					IsCounter: "0",
				}},
			}, nil
		},
	}
	router := pmHSetupRouterWithIndicator(ir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions?device_type=ENB", nil)
	req = req.WithContext(appcontext.WithLocale(req.Context(), appcontext.LocaleEN))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []kpiDefRespItem `json:"items"`
	}
	response.DecodeData(t, w.Body, &body)
	require.Len(t, body.Items, 1)
	assert.Equal(t, "RRC Setup Success Rate", body.Items[0].DisplayName)
}

// include_counters=true：不按 is_counter 过滤，KPI + 计数器都返回。
func TestHandler_ListKPIDefinitions_IncludeCounters(t *testing.T) {
	var captured indicator.IndicatorListFilter
	ir := &pmHIndicatorRepo{
		listAllFn: func(_ context.Context, f indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error) {
			captured = f
			return []indicator.IndicatorListItem{
				{PerfIndicator: indicator.PerfIndicator{ID: "K1001", EnName: "kpi_a", IsCounter: "0"}},
				{PerfIndicator: indicator.PerfIndicator{ID: "C2001", EnName: "counter_b", IsCounter: "1"}},
			}, nil
		},
	}
	router := pmHSetupRouterWithIndicator(ir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions?device_type=ENB&include_counters=true", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// include_counters=true → 不再按 is_counter 过滤。
	assert.Nil(t, captured.IsCounter)

	var body struct {
		Items []kpiDefRespItem `json:"items"`
		Total int              `json:"total"`
	}
	response.DecodeData(t, w.Body, &body)
	assert.Equal(t, 2, body.Total)
	require.Len(t, body.Items, 2)
	assert.Equal(t, "0", body.Items[0].IsCounter)
	assert.Equal(t, "1", body.Items[1].IsCounter)
	assert.Equal(t, "C2001", body.Items[1].ID)
}

func TestHandler_ListKPIDefinitions_OnlyEnabled(t *testing.T) {
	var captured indicator.IndicatorListFilter
	ir := &pmHIndicatorRepo{
		listAllFn: func(_ context.Context, f indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error) {
			captured = f
			return []indicator.IndicatorListItem{
				{PerfIndicator: indicator.PerfIndicator{ID: "K1001", EnName: "kpi_a", IsCounter: "0"}},
			}, nil
		},
	}
	router := pmHSetupRouterWithIndicator(ir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions?device_type=ENB&include_counters=true&only_enabled=true", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, captured.IsCounter, "include_counters=true should still include counters")
	require.NotNil(t, captured.IsEnabled)
	assert.Equal(t, "1", *captured.IsEnabled)
	require.NotNil(t, captured.OperatorCode)
	assert.Equal(t, "default", *captured.OperatorCode)
}

func TestHandler_ListKPIDefinitions_IndicatorLevel(t *testing.T) {
	level := "both"
	ir := &pmHIndicatorRepo{
		listAllFn: func(_ context.Context, f indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error) {
			require.Equal(t, "GSM", f.DeviceType)
			return []indicator.IndicatorListItem{
				{PerfIndicator: indicator.PerfIndicator{
					ID: "K3001", EnName: "gsm_call_sr", CnName: strPtr("GSM呼叫成功率"),
					IsCounter: "0", IndicatorLevel: &level,
				}},
			}, nil
		},
	}
	router := pmHSetupRouterWithIndicator(ir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions?device_type=GSM", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []kpiDefRespItem `json:"items"`
		Total int              `json:"total"`
	}
	response.DecodeData(t, w.Body, &body)
	assert.Equal(t, 1, body.Total)
	require.Len(t, body.Items, 1)
	assert.Equal(t, "both", body.Items[0].IndicatorLevel)
}

// device_type 指定时只查该制式表（不传则三表合并）。
func TestHandler_ListKPIDefinitions_DeviceTypeFilter(t *testing.T) {
	var calls []string
	ir := &pmHIndicatorRepo{
		listAllFn: func(_ context.Context, f indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error) {
			calls = append(calls, f.DeviceType)
			return nil, nil
		},
	}
	router := pmHSetupRouterWithIndicator(ir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions?device_type=GNB", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"GNB"}, calls)
}

// 不带 device_type → 三制式表都查一遍。
func TestHandler_ListKPIDefinitions_AllDeviceTypes(t *testing.T) {
	var calls []string
	ir := &pmHIndicatorRepo{
		listAllFn: func(_ context.Context, f indicator.IndicatorListFilter) ([]indicator.IndicatorListItem, error) {
			calls = append(calls, f.DeviceType)
			return nil, nil
		},
	}
	router := pmHSetupRouterWithIndicator(ir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.ElementsMatch(t, []string{"ENB", "GSM", "GNB"}, calls)
}

// 非法 device_type → 400。
func TestHandler_ListKPIDefinitions_InvalidDeviceType(t *testing.T) {
	router := pmHSetupRouterWithIndicator(&pmHIndicatorRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions?device_type=XXX", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CalculateKPI_BadRequest(t *testing.T) {
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	// Missing required fields
	body, _ := json.Marshal(map[string]string{"device_id": "not-a-uuid"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/pm/kpi/calculate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListTasks(t *testing.T) {
	tr := &pmHTaskRepo{
		listFn: func(_ context.Context, f TaskFilter) (*model.ListResponse[PerformanceTask], error) {
			assert.Nil(t, f.Status)
			assert.Nil(t, f.TaskType)
			return model.NewListResponse([]PerformanceTask{
				{TaskName: "Daily PM", TaskType: PMTaskExtraction, Status: PMTaskPending},
			}, 1, 1, 20), nil
		},
	}
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), tr)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/tasks", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[PerformanceTask]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
}

func TestHandler_CreateTask_Success(t *testing.T) {
	var created *PerformanceTask
	tr := &pmHTaskRepo{
		createFn: func(_ context.Context, task *PerformanceTask) error {
			task.ID = uuid.New()
			task.Status = PMTaskPending
			created = task
			return nil
		},
	}
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), tr)

	body, _ := json.Marshal(CreatePerformanceTaskRequest{
		TaskName:    "Nightly Report",
		TaskType:    PMTaskReport,
		Granularity: "15min",
		Creator:     "admin",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/pm/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, created)
	assert.Equal(t, "Nightly Report", created.TaskName)
}

func TestHandler_CreateTask_MissingName(t *testing.T) {
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	body, _ := json.Marshal(map[string]string{"task_type": "report"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/pm/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func pmHJoinStrings(prefix string, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = prefix + "-" + strconv.Itoa(i+1)
	}
	return strings.Join(parts, ",")
}
