package collector

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/omcgo/omcgo/internal/pm/resultnorm"
)

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
		"C-PCT-REPORT": {IndicatorID: "C-PCT", ReportKey: "C-PCT-REPORT", Unit: "%", StatisType: "pct"},
		"C-NUM-REPORT": {IndicatorID: "C-NUM", ReportKey: "C-NUM-REPORT", Unit: "number", StatisType: "sum"},
		"C-MISSING":    {IndicatorID: "C-MISSING", ReportKey: "C.MISSING", Unit: "number", StatisType: "sum"},
	}

	err := c.ingestViaCopy(ctx, trace.SpanFromContext(ctx), time.Now(), time.Now(), 123, uuid.New(), payload, content, allow)

	require.NoError(t, err)
	require.Len(t, copyIngestor.counters, 3)
	assert.InDelta(t, 19.45, copyIngestor.counters[0].CounterValue, 1e-9)
	assert.Equal(t, float64(4), copyIngestor.counters[1].CounterValue)
	assert.Equal(t, "C-MISSING", copyIngestor.counters[2].CounterName)
	assert.True(t, math.IsNaN(copyIngestor.counters[2].CounterValue), "缺值补齐应发生在真实值规范化之后")
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

type recordingCopyIngestor struct {
	counters []model.PMCounter
	kpis     []model.KPIValue
}

func (r *recordingCopyIngestor) CopyIngest(_ context.Context, _ metrics.FileMarker, counters []model.PMCounter, kpis []model.KPIValue) (bool, error) {
	r.counters = append([]model.PMCounter(nil), counters...)
	r.kpis = append([]model.KPIValue(nil), kpis...)
	return false, nil
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
