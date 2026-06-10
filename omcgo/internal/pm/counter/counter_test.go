package counter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMetricsRepo 让 BatchInsert 返回预置错误，验证 wrapper 错误透传（issue #14）。
type fakeMetricsRepo struct {
	batchErr error
	gotCount int
}

func (f *fakeMetricsRepo) Insert(ctx context.Context, m metrics.PMMetric) error { return nil }
func (f *fakeMetricsRepo) BatchInsert(ctx context.Context, ms []metrics.PMMetric) error {
	f.gotCount = len(ms)
	return f.batchErr
}
func (f *fakeMetricsRepo) Query(ctx context.Context, q metrics.QueryRequest) ([]metrics.PMMetric, error) {
	return nil, nil
}
func (f *fakeMetricsRepo) Count(ctx context.Context, q metrics.QueryRequest) (int64, error) {
	return 0, nil
}

// CounterFilter 与 ListRequest 集成
func Test_CounterFilter_WithListRequest(t *testing.T) {
	filter := CounterFilter{
		ListRequest: model.ListRequest{
			Page:     2,
			PageSize: 10,
			SortBy:   "time",
			SortDir:  "asc",
		},
	}

	assert.Equal(t, 10, filter.Offset(), "page 2, size 10 should offset 10")
	assert.Equal(t, 10, filter.Limit())
}

func Test_CounterFilter_DefaultListRequest(t *testing.T) {
	filter := CounterFilter{
		ListRequest: model.DefaultListRequest(),
	}

	assert.Equal(t, 0, filter.Offset(), "page 1 should offset 0")
	assert.Equal(t, 20, filter.Limit())
}

// AggregatedCounter struct 字段稳定性
func Test_AggregatedCounter_Struct(t *testing.T) {
	now := time.Now()
	deviceID := uuid.New()

	ac := AggregatedCounter{
		Bucket:       now,
		DeviceID:     deviceID,
		CellID:       "cell-1",
		CounterGroup: "LTE.CellMeasReport",
		CounterName:  "PRB.UlAvailProcMeas",
		SumValue:     100.0,
		AvgValue:     25.0,
		MinValue:     10.0,
		MaxValue:     50.0,
		SampleCount:  4,
	}

	assert.Equal(t, now, ac.Bucket)
	assert.Equal(t, deviceID, ac.DeviceID)
	assert.Equal(t, "cell-1", ac.CellID)
	assert.Equal(t, 100.0, ac.SumValue)
	assert.Equal(t, 25.0, ac.AvgValue)
	assert.Equal(t, 10.0, ac.MinValue)
	assert.Equal(t, 50.0, ac.MaxValue)
	assert.Equal(t, int64(4), ac.SampleCount)
}

// G3 字段双向转换：PMCounter ↔ PMMetric 保证无字段丢失
// T-0164-P3 fix: 业务键改用 TR-069 标准 (OUI, DeviceSN) 双键
func Test_counterToMetric_BasicFields(t *testing.T) {
	deviceID := uuid.New()
	now := time.Now()
	c := model.PMCounter{
		Time:         now,
		DeviceID:     deviceID,
		OUI:          "48BF74",
		DeviceSN:     "1202000240194DP0026",
		CellID:       "cell-1",
		CounterGroup: "LTE.CellMeasReport",
		CounterName:  "PRB.UlAvailProcMeas",
		CounterValue: 42.5,
		Granularity:  15,
	}
	m := counterToMetric(c)

	assert.Equal(t, "48BF74", m.DeviceOUI)
	assert.Equal(t, "1202000240194DP0026", m.DeviceSN)
	assert.Equal(t, "PRB.UlAvailProcMeas", m.MetricPath)
	assert.Equal(t, "counter", string(m.MetricType))
	assert.Equal(t, 42.5, m.MetricValue)
	assert.Equal(t, "15min", string(m.Granularity))
	assert.Equal(t, now, m.EndTime)
	assert.Equal(t, now.Add(-15*time.Minute), m.StartTime)
	if assert.NotNil(t, m.ObjectLDN) {
		assert.Equal(t, "cell-1", *m.ObjectLDN)
	}
	assert.Equal(t, "LTE.CellMeasReport", m.Extra["counter_group"])
	assert.Equal(t, deviceID.String(), m.Extra["device_id"])
}

func Test_counterRoundTrip_PreservesCoreFields(t *testing.T) {
	deviceID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	original := model.PMCounter{
		Time:         now,
		DeviceID:     deviceID,
		OUI:          "48BF74",
		DeviceSN:     "1202000240194DP0026",
		CellID:       "cell-7",
		CounterGroup: "LTE.CellMeasReport",
		CounterName:  "PRB.UlAvailProcMeas",
		CounterValue: 7.5,
		Granularity:  15,
	}
	m := counterToMetric(original)
	got := metricToCounter(m)

	assert.Equal(t, deviceID, got.DeviceID, "DeviceID 应从 extra 反查")
	assert.Equal(t, "48BF74", got.OUI)
	assert.Equal(t, "1202000240194DP0026", got.DeviceSN)
	assert.Equal(t, "cell-7", got.CellID)
	assert.Equal(t, "LTE.CellMeasReport", got.CounterGroup)
	assert.Equal(t, "PRB.UlAvailProcMeas", got.CounterName)
	assert.Equal(t, 7.5, got.CounterValue)
	assert.Equal(t, 15, got.Granularity)
}

// issue #14: ErrLateArrival 必须穿透 PgCounterRepository.BatchInsert 包装层不被改写，
// 这样上层 collector 的 errors.Is(err, metrics.ErrLateArrival) 才能识别并降级跳过。
func Test_BatchInsert_PropagatesLateArrivalSentinel(t *testing.T) {
	fake := &fakeMetricsRepo{batchErr: metrics.ErrLateArrival}
	repo := &PgCounterRepository{metricsRepo: fake}

	err := repo.BatchInsert(context.Background(), []model.PMCounter{
		{OUI: "48BF74", DeviceSN: "SN1", CounterName: "c1", CounterValue: 1, Granularity: 15, Time: time.Now()},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, metrics.ErrLateArrival), "迟到数据 sentinel 必须穿透 wrapper")
	assert.Equal(t, 1, fake.gotCount, "counter 应转换为 1 条 metric 下传")
}

// 空切片不调用下游、不报错（保持既有快速返回行为）。
func Test_BatchInsert_EmptyNoop(t *testing.T) {
	fake := &fakeMetricsRepo{batchErr: errors.New("should not be called")}
	repo := &PgCounterRepository{metricsRepo: fake}
	require.NoError(t, repo.BatchInsert(context.Background(), nil))
	assert.Equal(t, 0, fake.gotCount)
}
