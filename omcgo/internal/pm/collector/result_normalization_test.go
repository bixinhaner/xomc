package collector

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	pmroot "github.com/omcgo/omcgo/internal/pm"
	pmkpi "github.com/omcgo/omcgo/internal/pm/kpi"
	pmrouter "github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/omcgo/omcgo/internal/pm/resultnorm"
)

func TestIngestViaCopy_DiscardsChangedSourceIdentityWithoutRetry(t *testing.T) {
	ctx := context.Background()
	registry := prometheus.NewRegistry()
	discards := &recordingRawDiscarder{}
	c := &PMCollector{
		bucket: "pm-files", rawDiscarder: discards,
		copyIngestor: &recordingCopyIngestor{
			err: fmt.Errorf("conflicting PM marker: %w", metrics.ErrSourceContentChanged),
		},
		eventBus: noopEventBus{}, logger: zap.NewNop(), metrics: pmroot.NewPMMetrics(registry),
	}
	payload := &FileReceivedPayload{
		Bucket: "incoming-pm", MinIOPath: "illegal/A20260802.0230.xml",
		DeviceSN: "SN-1", Carrier: "cmcc", Technology: "lte",
	}

	err := c.ingestViaCopy(
		ctx, trace.SpanFromContext(ctx), time.Now(), time.Now(), 1, uuid.New(),
		payload, &PMFileContent{CollectTime: time.Now()}, nil, false, make([]byte, 32),
	)

	require.NoError(t, err)
	require.Equal(t, []discardCall{{
		bucket: "incoming-pm", object: "illegal/A20260802.0230.xml",
	}}, discards.calls)
	require.Equal(t, float64(1), testutil.ToFloat64(
		c.metrics.FilesDiscardedTotal.WithLabelValues("source_content_changed"),
	))
}

func TestIngestViaCopy_IncrementsDiscoveryMetricsOnlyForNewMarker(t *testing.T) {
	ctx := context.Background()
	registry := prometheus.NewRegistry()
	copyIngestor := &recordingCopyIngestor{ingested: false}
	c := &PMCollector{
		copyIngestor: copyIngestor, eventBus: noopEventBus{},
		logger: zap.NewNop(), metrics: pmroot.NewPMMetrics(registry),
	}
	content := &PMFileContent{
		CollectTime: time.Now(), whitelistMissValues: 2, knownDisabledValues: 3,
	}
	payload := &FileReceivedPayload{
		MinIOPath: "pm/A.xml", DeviceSN: "SN-1", Carrier: "cmcc", Technology: "lte",
	}
	run := func() {
		require.NoError(t, c.ingestViaCopy(
			ctx, trace.SpanFromContext(ctx), time.Now(), time.Now(), 1, uuid.New(),
			payload, content, nil, false, make([]byte, 32),
		))
	}

	run()
	require.Zero(t, testutil.ToFloat64(
		c.metrics.WhitelistMissValuesTotal.WithLabelValues("cmcc", "lte"),
	))
	require.Zero(t, testutil.ToFloat64(
		c.metrics.KnownDisabledValuesTotal.WithLabelValues("cmcc", "lte"),
	))

	copyIngestor.ingested = true
	run()
	require.Equal(t, float64(2), testutil.ToFloat64(
		c.metrics.WhitelistMissValuesTotal.WithLabelValues("cmcc", "lte"),
	))
	require.Equal(t, float64(3), testutil.ToFloat64(
		c.metrics.KnownDisabledValuesTotal.WithLabelValues("cmcc", "lte"),
	))
}

func TestIngestViaCopy_NormalizesCounterValuesBeforeCopyIngest(t *testing.T) {
	ctx := context.Background()
	end := time.Date(2026, 7, 6, 14, 15, 0, 0, time.UTC)
	copyIngestor := &recordingCopyIngestor{}
	c := &PMCollector{
		copyIngestor: copyIngestor,
		eventBus:     noopEventBus{},
		logger:       zap.NewNop(),
		numberProcessLookup: func(context.Context) (string, error) {
			return resultnorm.NumberProcessIntHalfUp, nil
		},
	}

	content := &PMFileContent{
		CollectTime: end,
		Counters: []model.PMCounter{
			{
				Time:         end,
				DeviceID:     uuid.New(),
				OUI:          "48BF74",
				DeviceSN:     "SN-1",
				CellID:       "Cellid=1",
				CounterGroup: "C",
				CounterName:  "C-PCT",
				CounterValue: 19.45245145567464,
				Granularity:  15,
				Unit:         "%",
				StatisType:   "pct",
			},
			{
				Time:         end,
				DeviceID:     uuid.New(),
				OUI:          "48BF74",
				DeviceSN:     "SN-1",
				CellID:       "Cellid=1",
				CounterGroup: "C",
				CounterName:  "C-NUM",
				CounterValue: 3.6,
				Granularity:  15,
				Unit:         "number",
				StatisType:   "sum",
			},
		},
	}
	payload := &FileReceivedPayload{
		MinIOPath:  "pm/A20260706.xml",
		DeviceID:   uuid.NewString(),
		DeviceOUI:  "48BF74",
		DeviceSN:   "SN-1",
		Carrier:    "cmcc",
		Technology: "lte",
	}

	allow := map[string]CounterMeta{
		"C.PCT":     {IndicatorID: "C-PCT", ReportKey: "C.PCT", Unit: "%", StatisType: "pct"},
		"C.NUM":     {IndicatorID: "C-NUM", ReportKey: "C.NUM", Unit: "number", StatisType: "sum"},
		"C.MISSING": {IndicatorID: "C-MISSING", ReportKey: "C.MISSING", Unit: "number", StatisType: "sum"},
	}

	err := c.ingestViaCopy(
		ctx, trace.SpanFromContext(ctx), time.Now(), time.Now(), 123, uuid.New(),
		payload, content, allow, false, make([]byte, 32),
	)

	require.NoError(t, err)
	require.Len(t, copyIngestor.counters, 3)
	assert.InDelta(t, 19.45, copyIngestor.counters[0].CounterValue, 1e-9)
	assert.Equal(t, float64(4), copyIngestor.counters[1].CounterValue)
	assert.Equal(t, "C-MISSING", copyIngestor.counters[2].CounterName)
	assert.True(t, math.IsNaN(copyIngestor.counters[2].CounterValue), "缺值补齐应发生在真实值规范化之后")
}

func TestIngestViaCopy_PreservesXMLMeasurementWindowSeparatelyFromUploadTime(t *testing.T) {
	ctx := context.Background()
	windowEnd := time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC)
	windowStart := windowEnd.Add(-15 * time.Minute)
	uploadedAt := windowEnd.Add(2 * time.Minute)
	copyIngestor := &recordingCopyIngestor{ingested: true}
	registry := prometheus.NewRegistry()
	c := &PMCollector{
		copyIngestor: copyIngestor,
		eventBus:     noopEventBus{},
		logger:       zap.NewNop(),
		metrics:      pmroot.NewPMMetrics(registry),
	}
	content := &PMFileContent{
		CollectTime:   windowEnd,
		FileBeginTime: windowStart,
		FileEndTime:   windowEnd,
	}
	payload := &FileReceivedPayload{
		MinIOPath: "pm/A20260803.1045.xml.gz", DeviceSN: "SN-1",
		Carrier: "cmcc", Technology: "lte",
	}

	err := c.ingestViaCopy(
		ctx, trace.SpanFromContext(ctx), uploadedAt, uploadedAt, 123, uuid.New(),
		payload, content, nil, true, make([]byte, 32),
	)

	require.NoError(t, err)
	assert.Equal(t, uploadedAt, copyIngestor.marker.CollectTime)
	assert.Equal(t, windowStart, copyIngestor.marker.MeasurementStart)
	assert.Equal(t, windowEnd, copyIngestor.marker.MeasurementEnd)
	assert.Equal(t, float64(uploadedAt.Unix()), testutil.ToFloat64(
		c.metrics.IngestLastSuccessTimestamp.WithLabelValues("cmcc", "lte"),
	))
}

func TestPMMeasurementWindow_FallsBackToCollectTimeAndGranularity(t *testing.T) {
	end := time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC)

	start, gotEnd := pmMeasurementWindow(&PMFileContent{CollectTime: end, Granularity: 15})

	assert.Equal(t, end.Add(-15*time.Minute), start)
	assert.Equal(t, end, gotEnd)
}

func TestIngestViaCopy_Filters15MinRowsByEnabledIndicatorsAfterKPICalculation(t *testing.T) {
	ctx := context.Background()
	end := time.Date(2026, 7, 6, 14, 15, 0, 0, time.UTC)
	copyIngestor := &recordingCopyIngestor{}
	engine := pmkpi.NewKPIEngine(nil, nil, staticKPIRouter{route: &pmrouter.KPIRoute{
		KPIs: []pmrouter.KPIDef{{
			IndicatorID:  "K0001",
			Name:         "K.Enabled",
			Formula:      "C0001/C0002*100",
			Dependencies: []string{"C0001", "C0002"},
			StatisType:   "pct",
			Unit:         "%",
		}},
	}}, zap.NewNop())
	c := &PMCollector{
		kpiEngine:    engine,
		copyIngestor: copyIngestor,
		eventBus:     noopEventBus{},
		logger:       zap.NewNop(),
		enabledIndicators: fakeEnabledIndicators{set: map[string]struct{}{
			"C0001": {},
			"K0001": {},
		}},
		numberProcessLookup: func(context.Context) (string, error) {
			return resultnorm.NumberProcessNone, nil
		},
	}

	content := &PMFileContent{
		CollectTime: end,
		Counters: []model.PMCounter{
			{
				Time:         end,
				DeviceID:     uuid.New(),
				OUI:          "48BF74",
				DeviceSN:     "SN-1",
				CellID:       "Cellid=1",
				CounterGroup: "C",
				CounterName:  "C0001",
				CounterValue: 10,
				Granularity:  15,
				Unit:         "number",
				StatisType:   "sum",
			},
			{
				Time:         end,
				DeviceID:     uuid.New(),
				OUI:          "48BF74",
				DeviceSN:     "SN-1",
				CellID:       "Cellid=1",
				CounterGroup: "C",
				CounterName:  "C0002",
				CounterValue: 20,
				Granularity:  15,
				Unit:         "number",
				StatisType:   "sum",
			},
		},
	}
	payload := &FileReceivedPayload{
		MinIOPath:  "pm/A20260706.xml",
		DeviceID:   uuid.NewString(),
		DeviceOUI:  "48BF74",
		DeviceSN:   "SN-1",
		Carrier:    "cmcc",
		Technology: "lte",
	}
	allow := map[string]CounterMeta{
		"C.KEEP": {IndicatorID: "C0001", ReportKey: "C.KEEP", Unit: "number", StatisType: "sum"},
		"C.DEP":  {IndicatorID: "C0002", ReportKey: "C.DEP", Unit: "number", StatisType: "sum"},
	}

	err := c.ingestViaCopy(
		ctx, trace.SpanFromContext(ctx), time.Now(), time.Now(), 123, uuid.New(),
		payload, content, allow, false, make([]byte, 32),
	)

	require.NoError(t, err)
	require.Len(t, copyIngestor.counters, 1, "未启用依赖 counter 可参与 KPI 计算，但自身不落 15min counter 行")
	assert.Equal(t, "C0001", copyIngestor.counters[0].CounterName)
	require.Len(t, copyIngestor.kpis, 1)
	assert.Equal(t, "K0001", copyIngestor.kpis[0].IndicatorID)
	assert.InDelta(t, 50, copyIngestor.kpis[0].KPIValue, 1e-9)
}

func TestIngestViaCopy_EmptyEnabledSetWritesNoRows(t *testing.T) {
	ctx := context.Background()
	copyIngestor := &recordingCopyIngestor{}
	c := &PMCollector{
		copyIngestor:      copyIngestor,
		eventBus:          noopEventBus{},
		logger:            zap.NewNop(),
		enabledIndicators: fakeEnabledIndicators{set: map[string]struct{}{}},
	}
	content := &PMFileContent{
		CollectTime: time.Date(2026, 7, 6, 14, 15, 0, 0, time.UTC),
		Counters: []model.PMCounter{{
			CounterName:  "C0001",
			CounterValue: 1,
			Unit:         "number",
			StatisType:   "sum",
		}},
	}
	payload := &FileReceivedPayload{MinIOPath: "pm/A20260706.xml", DeviceSN: "SN-1", Carrier: "cmcc", Technology: "lte"}
	allow := map[string]CounterMeta{
		"C0001": {IndicatorID: "C0001", ReportKey: "C0001", Unit: "number", StatisType: "sum"},
	}

	err := c.ingestViaCopy(
		ctx, trace.SpanFromContext(ctx), time.Now(), time.Now(), 123, uuid.New(),
		payload, content, allow, false, make([]byte, 32),
	)

	require.NoError(t, err)
	assert.True(t, copyIngestor.called, "空启用集代表全部禁用，应完成文件 marker 写入而不是跳过 CopyIngest")
	assert.Empty(t, copyIngestor.counters)
	assert.Empty(t, copyIngestor.kpis)
}

func TestIngestViaCopy_EnabledIndicatorLookupFailureFailsFile(t *testing.T) {
	ctx := context.Background()
	copyIngestor := &recordingCopyIngestor{}
	c := &PMCollector{
		copyIngestor:      copyIngestor,
		eventBus:          noopEventBus{},
		logger:            zap.NewNop(),
		enabledIndicators: fakeEnabledIndicators{err: assert.AnError},
	}
	content := &PMFileContent{
		CollectTime: time.Date(2026, 7, 6, 14, 15, 0, 0, time.UTC),
		Counters: []model.PMCounter{{
			CounterName: "C0001",
			Unit:        "number",
			StatisType:  "sum",
		}},
	}
	payload := &FileReceivedPayload{MinIOPath: "pm/A20260706.xml", DeviceSN: "SN-1", Carrier: "cmcc", Technology: "lte"}

	err := c.ingestViaCopy(
		ctx, trace.SpanFromContext(ctx), time.Now(), time.Now(), 123, uuid.New(),
		payload, content, nil, false, make([]byte, 32),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "lookup enabled PM indicators")
	assert.False(t, copyIngestor.called)
	assert.Empty(t, copyIngestor.counters)
}

func TestNormalizeResults_NormalizesKPIValuesAndFailsMissingMetadata(t *testing.T) {
	c := &PMCollector{
		numberProcessLookup: func(context.Context) (string, error) {
			return resultnorm.NumberProcessIntHalfUp, nil
		},
	}
	kpis := []model.KPIValue{
		{IndicatorID: "K-PCT", KPIValue: 150.456, Unit: "%", StatisType: "pct"},
		{IndicatorID: "K-AVG", KPIValue: 12.345, Unit: "Mbps", StatisType: "avg"},
	}

	err := c.normalizeResults(context.Background(), nil, kpis)

	require.NoError(t, err)
	assert.Equal(t, float64(100), kpis[0].KPIValue)
	assert.InDelta(t, 12.35, kpis[1].KPIValue, 1e-9)

	err = c.normalizeResults(context.Background(), []model.PMCounter{
		{CounterName: "C-MISSING", CounterValue: 1.23, Unit: "number"},
	}, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, resultnorm.ErrMissingMetadata)
	assert.Contains(t, err.Error(), "C-MISSING")
}

func TestNormalizeResults_PreservesUnknownCounterWithoutMetadata(t *testing.T) {
	c := &PMCollector{}
	counters := []model.PMCounter{{CounterName: "Vendor.New.Counter", CounterValue: 12.345}}

	err := c.normalizeResults(context.Background(), counters, nil)

	require.NoError(t, err)
	assert.Equal(t, 12.345, counters[0].CounterValue)
}

type recordingCopyIngestor struct {
	called   bool
	ingested bool
	err      error
	marker   metrics.FileMarker
	counters []model.PMCounter
	kpis     []model.KPIValue
}

func (r *recordingCopyIngestor) CopyIngest(_ context.Context, marker metrics.FileMarker, counters []model.PMCounter, kpis []model.KPIValue) (bool, error) {
	r.called = true
	r.marker = marker
	r.counters = append([]model.PMCounter(nil), counters...)
	r.kpis = append([]model.KPIValue(nil), kpis...)
	return r.ingested, r.err
}

type noopEventBus struct{}

func (noopEventBus) Publish(context.Context, string, event.Event) error { return nil }
func (noopEventBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return noopSubscription{}, nil
}
func (noopEventBus) QueueSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return noopSubscription{}, nil
}
func (noopEventBus) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return noopSubscription{}, nil
}
func (noopEventBus) Close() error { return nil }

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() error { return nil }

type fakeEnabledIndicators struct {
	set map[string]struct{}
	err error
}

func (f fakeEnabledIndicators) LookupEnabledIndicators(context.Context, string) (map[string]struct{}, error) {
	return f.set, f.err
}

type staticKPIRouter struct {
	route *pmrouter.KPIRoute
	err   error
}

func (s staticKPIRouter) LookupByDevice(context.Context, string) (*pmrouter.KPIRoute, error) {
	return s.route, s.err
}
