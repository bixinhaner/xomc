package counter

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

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
func Test_counterToMetric_BasicFields(t *testing.T) {
	deviceID := uuid.New()
	now := time.Now()
	c := model.PMCounter{
		Time:         now,
		DeviceID:     deviceID,
		CellID:       "cell-1",
		CounterGroup: "LTE.CellMeasReport",
		CounterName:  "PRB.UlAvailProcMeas",
		CounterValue: 42.5,
		Granularity:  15,
	}
	m := counterToMetric(c)

	assert.Equal(t, deviceID.String(), m.DeviceSN)
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
}

func Test_counterRoundTrip_PreservesCoreFields(t *testing.T) {
	deviceID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	original := model.PMCounter{
		Time:         now,
		DeviceID:     deviceID,
		CellID:       "cell-7",
		CounterGroup: "LTE.CellMeasReport",
		CounterName:  "PRB.UlAvailProcMeas",
		CounterValue: 7.5,
		Granularity:  15,
	}
	m := counterToMetric(original)
	got := metricToCounter(m)

	assert.Equal(t, deviceID, got.DeviceID)
	assert.Equal(t, "cell-7", got.CellID)
	assert.Equal(t, "LTE.CellMeasReport", got.CounterGroup)
	assert.Equal(t, "PRB.UlAvailProcMeas", got.CounterName)
	assert.Equal(t, 7.5, got.CounterValue)
	assert.Equal(t, 15, got.Granularity)
}
