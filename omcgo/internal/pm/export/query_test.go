package export

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

func TestBuildDeviceKeysetSQL_FirstBatch_NoCursor(t *testing.T) {
	mt := metrics.MetricTypeKPI
	req := aggregator.QueryRequest{
		Granularity: metrics.GranularityHourly,
		DeviceOUIs:  []string{"ABCDEF"},
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"K001"},
		MetricType:  &mt,
		StartTime:   time.Now().Add(-time.Hour),
		EndTime:     time.Now(),
	}
	q, args := buildDeviceKeysetSQL("pm_metrics_hourly", req, false, time.Time{}, uuid.Nil, 5000)
	// 首批无 keyset 游标谓词。
	assert.NotContains(t, q, `("time", id) >`)
	// ORDER BY time, id + LIMIT。
	assert.Contains(t, q, `ORDER BY "time" ASC, id ASC`)
	assert.Contains(t, q, "LIMIT 5000")
	assert.Contains(t, q, "FROM pm_metrics_hourly")
	// 过滤参数都进了 args。
	assert.NotEmpty(t, args)
}

func TestBuildDeviceKeysetSQL_NextBatch_HasCursor(t *testing.T) {
	req := aggregator.QueryRequest{Granularity: metrics.Granularity15Min}
	cur := time.Now()
	id := uuid.New()
	q, args := buildDeviceKeysetSQL("pm_metrics", req, true, cur, id, 5000)
	assert.Contains(t, q, `("time", id) > (`)
	// 游标值在 args 中。
	foundTime, foundID := false, false
	for _, a := range args {
		if tv, ok := a.(time.Time); ok && tv.Equal(cur) {
			foundTime = true
		}
		if iv, ok := a.(uuid.UUID); ok && iv == id {
			foundID = true
		}
	}
	assert.True(t, foundTime, "cursor time in args")
	assert.True(t, foundID, "cursor id in args")
}

func TestBuildDeviceKeysetSQL_PairedOUISN(t *testing.T) {
	req := aggregator.QueryRequest{
		Granularity: metrics.Granularity15Min,
		DeviceOUIs:  []string{"OUI1", "OUI2"},
		DeviceSNs:   []string{"SN1", "SN2"},
	}
	q, _ := buildDeviceKeysetSQL("pm_metrics", req, false, time.Time{}, uuid.Nil, 100)
	// 成对过滤：OR 连接两组 (oui AND sn)。
	assert.Contains(t, q, "device_oui")
	assert.Contains(t, q, "device_sn")
	assert.Contains(t, q, " OR ")
}

func TestBuildAdhocKeysetSQL(t *testing.T) {
	id := uuid.New()
	q, args := buildAdhocKeysetSQL(id, time.Time{}, time.Time{}, false, time.Time{}, uuid.Nil, 5000)
	assert.Contains(t, q, "FROM pm_adhoc_aggregation_results")
	assert.Contains(t, q, "task_id")
	assert.Contains(t, q, `ORDER BY "time" ASC, id ASC`)
	// squirrel sq.Eq 把 uuid 当 driver.Valuer 序列化成字符串 arg；pgx 端两种都接受。
	assert.Equal(t, id.String(), args[0])
}

func TestBuildAdhocKeysetSQL_WithTimeWindow(t *testing.T) {
	id := uuid.New()
	st := time.Now().Add(-time.Hour)
	et := time.Now()
	q, _ := buildAdhocKeysetSQL(id, st, et, false, time.Time{}, uuid.Nil, 100)
	assert.Contains(t, q, "time >=")
	assert.Contains(t, q, "time <=")
}
