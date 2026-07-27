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
	q, args := buildDeviceKeysetSQL("pm_metrics_hourly", req, nil, false, time.Time{}, uuid.Nil, 5000)
	// 首批无 keyset 游标谓词。
	assert.NotContains(t, q, `("time", id) >`)
	// ORDER BY time, id + LIMIT。
	assert.Contains(t, q, `DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
	assert.Contains(t, q, `ingest_time DESC`)
	assert.Contains(t, q, `ORDER BY "time" ASC, id ASC`)
	assert.Contains(t, q, "LIMIT 5000")
	assert.Contains(t, q, "FROM pm_hourly_bucket_versions")
	assert.Contains(t, q, "d.metric_id=ANY(s.metric_ids)")
	assert.NotContains(t, q, "unnest(s.metric_ids)")
	// 过滤参数都进了 args。
	assert.NotEmpty(t, args)
}

func TestBuildDeviceKeysetSQL_TimeWindowUsesExclusiveEnd(t *testing.T) {
	req := aggregator.QueryRequest{
		Granularity: metrics.Granularity15Min,
		StartTime:   time.Date(2026, 7, 16, 12, 15, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60)),
		EndTime:     time.Date(2026, 7, 16, 15, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60)),
	}

	q, _ := buildDeviceKeysetSQL("pm_metrics", req, nil, false, time.Time{}, uuid.Nil, 5000)

	assert.Contains(t, q, "time >=")
	assert.Contains(t, q, "time < ")
	assert.NotContains(t, q, "time <=")
}

func TestBuildDeviceKeysetSQL_NextBatch_HasCursor(t *testing.T) {
	req := aggregator.QueryRequest{Granularity: metrics.Granularity15Min}
	cur := time.Now()
	id := uuid.New()
	q, args := buildDeviceKeysetSQL("pm_metrics", req, nil, true, cur, id, 5000)
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
	q, _ := buildDeviceKeysetSQL("pm_metrics", req, nil, false, time.Time{}, uuid.Nil, 100)
	// 成对过滤：OR 连接两组 (oui AND sn)。
	assert.Contains(t, q, "device_oui")
	assert.Contains(t, q, "device_sn")
	assert.Contains(t, q, " OR ")
}

// A1：传入 object_ldn 白名单时，SQL 带 object_ldn IN(...) 过滤，白名单值进 args。
func TestBuildDeviceKeysetSQL_ObjectLDNFilter(t *testing.T) {
	req := aggregator.QueryRequest{Granularity: metrics.Granularity15Min}
	ldns := []string{"Cellid=1,PLMN=00101", "Cellid=1,PLMN=46068"}
	q, args := buildDeviceKeysetSQL("pm_metrics", req, ldns, false, time.Time{}, uuid.Nil, 100)
	assert.Contains(t, q, "object_ldn IN (")
	foundA, foundB := false, false
	for _, a := range args {
		if a == ldns[0] {
			foundA = true
		}
		if a == ldns[1] {
			foundB = true
		}
	}
	assert.True(t, foundA && foundB, "两个白名单值都进 args")
}

// A1：空白名单时不加 object_ldn 过滤（向后兼容，导全部小区/PLMN）。
func TestBuildDeviceKeysetSQL_NoObjectLDNFilter(t *testing.T) {
	req := aggregator.QueryRequest{Granularity: metrics.Granularity15Min}
	q, _ := buildDeviceKeysetSQL("pm_metrics", req, nil, false, time.Time{}, uuid.Nil, 100)
	assert.NotContains(t, q, "object_ldn IN")
}

func TestBuildDeviceOffsetSQL_UsesStoredResultTableWithoutID(t *testing.T) {
	req := aggregator.QueryRequest{
		Granularity: metrics.GranularityDaily,
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"KGSM0101", "CGSM0010001"},
	}
	q, args := buildDeviceOffsetSQL("pm_metrics_daily", req, nil, 5000, 5000)

	assert.Contains(t, q, "FROM pm_metrics_daily")
	assert.Contains(t, q, `DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
	assert.Contains(t, q, `ingest_time DESC`)
	assert.NotContains(t, q, " id,")
	assert.Contains(t, q, "metric_path = ")
	assert.Contains(t, q, "metric_type = ")
	assert.Contains(t, q, `ORDER BY "time" ASC, device_oui ASC, device_sn ASC, object_ldn ASC, metric_path ASC, metric_type ASC`)
	assert.Contains(t, q, "LIMIT 5000")
	assert.Contains(t, q, "OFFSET 5000")
	assert.Contains(t, args, "SN1")
	assert.Contains(t, args, "KGSM0101")
	assert.Contains(t, args, "CGSM0010001")
	assert.Contains(t, args, "kpi")
	assert.Contains(t, args, "counter")
}

func TestBuildDeviceKeysetSQL_BindsMetricPathToInferredMetricTypeWhenTypeAbsent(t *testing.T) {
	req := aggregator.QueryRequest{
		Granularity: metrics.GranularityHourly,
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{" KGSM0101 ", "CGSM0010001"},
	}
	q, args := buildDeviceKeysetSQL("pm_metrics_hourly", req, nil, false, time.Time{}, uuid.Nil, 5000)

	assert.Contains(t, q, "((metric_path = ")
	assert.Contains(t, q, "AND metric_type = ")
	assert.Contains(t, q, ") OR (metric_path = ")
	assert.Contains(t, args, "KGSM0101")
	assert.Contains(t, args, "kpi")
	assert.Contains(t, args, "CGSM0010001")
	assert.Contains(t, args, "counter")
}

func TestBuildAdhocKeysetSQL(t *testing.T) {
	id := uuid.New()
	q, args := buildAdhocKeysetSQL(id, nil, time.Time{}, time.Time{}, false, time.Time{}, uuid.Nil, 5000)
	assert.Contains(t, q, "FROM pm_adhoc_aggregation_results r")
	assert.Contains(t, q, "task_id")
	assert.Contains(t, q, `ORDER BY "r"."time" ASC, r.id ASC`)
	// squirrel sq.Eq 把 uuid 当 driver.Valuer 序列化成字符串 arg；pgx 端两种都接受。
	assert.Equal(t, id.String(), args[0])
}

// adhoc 取数镜像网页关联：LEFT JOIN product_dim / device_group_dim（跨库分离后用本库影子表），
// 选出产品名 / 设备组名。
func TestBuildAdhocKeysetSQL_JoinsNames(t *testing.T) {
	q, _ := buildAdhocKeysetSQL(uuid.New(), nil, time.Time{}, time.Time{}, false, time.Time{}, uuid.Nil, 5000)
	assert.Contains(t, q, "LEFT JOIN product_dim")
	assert.Contains(t, q, "device_group_dim")
	assert.Contains(t, q, "product_name")
	assert.Contains(t, q, "device_group_name")
	assert.Contains(t, q, "pm_adhoc_aggregation_results")
	// B1 修复：object_ldn 改带 ',Tech=<制式>' 后缀后，取组名 JOIN 必须剥逗号前段再等值，
	// 否则等值 JOIN 永不命中 → 导出 CSV 设备组名丢失（回归）。与网页 buildResultsQuery 同口径。
	assert.Contains(t, q, "split_part(r.object_ldn, ',', 1)")
	// 旧的等值 JOIN（右侧裸 r.object_ldn）已被 split_part 取代，不应再出现。
	assert.NotContains(t, q, "= r.object_ldn")
}

func TestBuildAdhocKeysetSQL_WithTimeWindow(t *testing.T) {
	id := uuid.New()
	st := time.Now().Add(-time.Hour)
	et := time.Now()
	q, _ := buildAdhocKeysetSQL(id, nil, st, et, false, time.Time{}, uuid.Nil, 100)
	assert.Contains(t, q, "r.time >=")
	assert.Contains(t, q, "r.time <=")
}

// #38：adhoc 底层可全存该制式全部已启用指标，但导出表头和数据都只能包含任务配置指标集。
func TestBuildAdhocExportSQL_FiltersTaskMetricPaths(t *testing.T) {
	id := uuid.New()
	metricPaths := []string{"KGSM0101", "KGSM0102"}

	dataSQL, dataArgs := buildAdhocKeysetSQL(
		id, metricPaths, time.Time{}, time.Time{}, false, time.Time{}, uuid.Nil, 5000,
	)
	headerSQL, headerArgs := buildAdhocDistinctMetricsSQL(id, metricPaths, time.Time{}, time.Time{})

	assert.Contains(t, dataSQL, "r.metric_path IN (")
	assert.Contains(t, headerSQL, "metric_path IN (")
	for _, metricPath := range metricPaths {
		assert.Contains(t, dataArgs, metricPath)
		assert.Contains(t, headerArgs, metricPath)
	}
}

// 横表列发现：dashboard 源 DISTINCT(metric_path, metric_type)，按 metric_paths + 时窗收口。
func TestBuildDistinctMetricsSQL(t *testing.T) {
	st := time.Now().Add(-2 * time.Hour)
	et := time.Now()
	q, args := buildDistinctMetricsSQL("pm_metrics_hourly", []string{"K001", "C002"}, st, et)
	assert.Contains(t, q, "DISTINCT metric_path, metric_type")
	assert.Contains(t, q, "FROM pm_hourly_bucket_versions")
	assert.Contains(t, q, "d.metric_id=ANY(s.metric_ids)")
	assert.Contains(t, q, "metric_path IN (")
	assert.Contains(t, q, "time >=")
	assert.Contains(t, q, "time < ")
	assert.NotContains(t, q, "time <=")
	assert.NotEmpty(t, args)
}

// 空 metric_paths + 空时窗：仅 DISTINCT，无过滤谓词（边界）。
func TestBuildDistinctMetricsSQL_NoFilters(t *testing.T) {
	q, args := buildDistinctMetricsSQL("pm_metrics", nil, time.Time{}, time.Time{})
	assert.Contains(t, q, "DISTINCT metric_path, metric_type")
	assert.NotContains(t, q, "metric_path IN")
	assert.NotContains(t, q, "time >=")
	assert.Empty(t, args)
}

// adhoc 源列发现：按 task_id（+ 可选时窗）DISTINCT。
func TestBuildAdhocDistinctMetricsSQL(t *testing.T) {
	id := uuid.New()
	q, args := buildAdhocDistinctMetricsSQL(id, nil, time.Time{}, time.Time{})
	assert.Contains(t, q, "DISTINCT metric_path, metric_type")
	assert.Contains(t, q, "FROM pm_adhoc_aggregation_results")
	assert.Contains(t, q, "task_id")
	assert.Equal(t, id.String(), args[0])
}
