package dictsource

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// metricValue 读 dictionary_source_sync_total{result, trigger} 的当前累计值。
// 用 prometheus/testutil.ToFloat64 反射读取 CounterVec 的单个标签组合值。
func metricValue(t *testing.T, m *Metrics, result, trigger string) float64 {
	t.Helper()
	c, err := m.syncTotal.GetMetricWith(prometheus.Labels{
		"result":  result,
		"trigger": trigger,
	})
	require.NoError(t, err)
	return testutil.ToFloat64(c)
}

func TestMetrics_Observe_Increments(t *testing.T) {
	m := NewMetrics(prometheus.NewRegistry())
	assert.Equal(t, 0.0, metricValue(t, m, ResultOK, TriggerManual))

	m.Observe(ResultOK, TriggerManual)
	m.Observe(ResultOK, TriggerManual)
	m.Observe(ResultFailed, TriggerDaily)

	assert.Equal(t, 2.0, metricValue(t, m, ResultOK, TriggerManual))
	assert.Equal(t, 1.0, metricValue(t, m, ResultFailed, TriggerDaily))
	assert.Equal(t, 0.0, metricValue(t, m, ResultTimeout, TriggerInitial))
}

func TestMetrics_NilSafeObserve(t *testing.T) {
	var m *Metrics
	assert.NotPanics(t, func() { m.Observe(ResultOK, TriggerManual) })
}

func TestSyncEngine_RecordsMetricOnSuccess(t *testing.T) {
	reader := &fakeReader{rows: [][2]string{{"L1", "V1"}}}
	writer := &fakeWriter{countVal: 1}
	eng := newEngineWithMocks(t, reader, writer)
	metrics := NewMetrics(prometheus.NewRegistry())
	eng.SetMetrics(metrics)

	_, err := eng.SyncOne(context.Background(), SyncDict{
		ID: 1, SourceTable: "devices", LabelField: "product_class", ValueField: "product_class",
	})
	require.NoError(t, err)
	assert.Equal(t, 1.0, metricValue(t, metrics, ResultOK, TriggerManual))
}

func TestSyncEngine_RecordsFailureMetric(t *testing.T) {
	reader := &fakeReader{rowsErr: errors.New("pg down")}
	eng := newEngineWithMocks(t, reader, &fakeWriter{})
	metrics := NewMetrics(prometheus.NewRegistry())
	eng.SetMetrics(metrics)

	_, err := eng.SyncOne(context.Background(), SyncDict{
		ID: 1, SourceTable: "devices", LabelField: "product_class", ValueField: "product_class",
	})
	require.Error(t, err)
	assert.Equal(t, 1.0, metricValue(t, metrics, ResultFailed, TriggerManual))
}

func TestSyncEngine_SyncAllUsesDailyTrigger(t *testing.T) {
	reader := &fakeReader{rows: [][2]string{{"a", "b"}}}
	writer := &fakeWriter{countVal: 1}
	eng := newEngineWithMocks(t, reader, writer)
	metrics := NewMetrics(prometheus.NewRegistry())
	eng.SetMetrics(metrics)

	ok, failed := eng.SyncAll(context.Background(), func(ctx context.Context) ([]SyncDict, error) {
		return []SyncDict{
			{ID: 1, SourceTable: "devices", LabelField: "product_class", ValueField: "product_class"},
		}, nil
	})
	assert.Equal(t, 1, ok)
	assert.Equal(t, 0, failed)
	// daily trigger 由 SyncAll 内部固定传入,验证 label 与 manual 区分
	assert.Equal(t, 1.0, metricValue(t, metrics, ResultOK, TriggerDaily))
	assert.Equal(t, 0.0, metricValue(t, metrics, ResultOK, TriggerManual))
}
