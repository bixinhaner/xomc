package adhoc

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// 不带大时间段：只有 task_id + LIMIT/OFFSET，无 time 窗口子句。
func Test_buildResultsQuery_NoTimeRange(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{}, 100, 0)

	assert.NotContains(t, q, "AND r.time >=")
	assert.NotContains(t, q, "AND r.time <=")
	assert.Contains(t, q, "WHERE r.task_id = $1")
	assert.Contains(t, q, "WITH current_versions AS")
	assert.Contains(t, q, "FROM pm_aggregation_results r")
	assert.Contains(t, q, "FROM pm_aggregation_windows active_window")
	assert.Contains(t, q, "JOIN pm_aggregation_publications published_window")
	assert.Contains(t, q, "published_window.status = 'published'")
	assert.Contains(t, q, "published_window.revision = r.revision")
	assert.NotContains(t, q, "FROM pm_adhoc_aggregation_results")
	assert.NotContains(t, q, "extra->>'active_version'")
	assert.Contains(t, q, "ORDER BY r.window_start DESC LIMIT $2 OFFSET $3")
	// args = [taskID, limit, offset]
	assert.Equal(t, []any{id, 100, 0}, args)
}

func Test_buildResultsQuery_FiltersInactiveVersionSlices(t *testing.T) {
	q, _ := buildResultsQuery(uuid.New(), resultsFilter{}, 100, 0)

	assert.Contains(t, q, "JOIN current_versions cv")
	assert.Contains(t, q, "SELECT DISTINCT ON (candidate.task_id, candidate.granularity, candidate.window_start)")
	assert.Contains(t, q, "FROM pm_aggregation_windows")
	assert.Contains(t, q, "AND p.status = 'published'")
	assert.Contains(t, q, "active_window.status IN ('open', 'finalizing', 'prepared', 'rebuilding', 'failed')")
	assert.Contains(t, q, "active_window.version_effective_from > candidate.version_effective_from")
	assert.Contains(t, q, "published_window.status = 'published'")
	assert.NotContains(t, q, "extra->>'active_version'")
	assert.NotContains(t, q, "'active_version'")
}

func Test_buildResultsQuery_FiltersByTaskDimension(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{Dimension: DimensionProduct}, 100, 0)

	assert.Contains(t, q, "WHERE r.task_id = $1 AND r.dimension = 'product'")
	assert.Contains(t, q, "ORDER BY r.window_start DESC LIMIT $2 OFFSET $3")
	assert.Equal(t, []any{id, 100, 0}, args)
}

func Test_buildResultsQuery_UnknownDimensionDoesNotInjectPredicate(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{Dimension: Dimension("bad-dimension")}, 100, 0)

	assert.NotContains(t, q, "bad-dimension")
	assert.Contains(t, q, "WHERE r.task_id = $1")
	assert.Equal(t, []any{id, 100, 0}, args)
}

func Test_buildResultsQuery_ProjectsCompatibilityFieldsFromAggregationResults(t *testing.T) {
	q, _ := buildResultsQuery(uuid.New(), resultsFilter{}, 100, 0)

	for _, want := range []string{
		"CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END AS device_sn",
		"CASE WHEN r.dimension = 'product' THEN r.dimension_key::uuid ELSE NULL::uuid END AS product_id",
		"r.aggregation_op::text AS statis_type",
		"r.window_start AS \"time\", r.window_start AS start_time, r.window_end AS end_time",
		"r.created_at AS ingest_time",
		"CASE WHEN r.dimension = 'device' THEN NULLIF(r.object_ldn, '') WHEN r.dimension = 'network' THEN 'Network' ELSE r.dimension_key END AS object_ldn",
		"'task_version_id', r.task_version_id",
		"'complete', r.period_complete",
		"'missing_slots', r.missing_slots",
		"'dimension', r.dimension",
		"'revision', r.revision",
		"'version_effective_from', r.version_effective_from",
		"'version_effective_to', r.version_effective_to",
		"'received_slots', r.received_slots",
		"'expected_slots', r.expected_slots",
		"'version_expected_slots', r.version_expected_slots",
		"'natural_expected_slots', r.natural_expected_slots",
		"'version_slice_complete', r.version_slice_complete",
		"'period_complete', r.period_complete",
		"'partial', false",
	} {
		assert.Contains(t, q, want)
	}
	assert.NotContains(t, q, "pm_adhoc_aggregation_results")
	assert.NotContains(t, q, "extra->>'active_version'")
	assert.NotContains(t, q, "'active_version'")
}

// PM-线名解析：SELECT 带出解析名两列，LEFT JOIN products / device_groups。
func Test_buildResultsQuery_NameJoins(t *testing.T) {
	id := uuid.New()
	q, _ := buildResultsQuery(id, resultsFilter{}, 100, 0)

	// SELECT 补出 product_name / device_group_name 两列
	assert.Contains(t, q, "p.product_name")
	assert.Contains(t, q, "g.name AS device_group_name")
	// 两个 LEFT JOIN：product 按 id 对 product_id；device_group 按 'DeviceGroup='||id 对 object_ldn 逗号前段
	// （设备组制式治本：object_ldn 改带 ',Tech=<制式>' 后缀，取组名需先 split_part 剥逗号前段）。
	assert.Contains(t, q, "LEFT JOIN product_dim p ON r.dimension = 'product' AND p.id::text = r.dimension_key")
	assert.Contains(t, q, "LEFT JOIN device_group_dim g ON r.dimension = 'device_group' AND ('DeviceGroup=' || g.id::text) = split_part(r.dimension_key, ',', 1)")
	// 主表起了别名 r
	assert.Contains(t, q, "FROM pm_aggregation_results r")
	assert.NotContains(t, q, "FROM pm_adhoc_aggregation_results")
}

// 带大时间段：起止时间转为 time.Time 占位参数，子句出现且占位号顺延。
func Test_buildResultsQuery_WithTimeRange(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{
		StartTime: "2026-05-20T00:00:00Z",
		EndTime:   "2026-05-27T00:00:00Z",
	}, 5000, 10)

	assert.Contains(t, q, "AND r.window_start >= $2")
	assert.Contains(t, q, "AND r.window_start < $3")
	assert.NotContains(t, q, "AND r.window_start <= $3")
	assert.Contains(t, q, "LIMIT $4 OFFSET $5")
	// args = [taskID, start(time.Time), end(time.Time), limit, offset]
	if assert.Len(t, args, 5) {
		assert.Equal(t, id, args[0])
		st, ok := args[1].(time.Time)
		assert.True(t, ok)
		assert.Equal(t, "2026-05-20T00:00:00Z", st.UTC().Format(time.RFC3339))
		et, ok := args[2].(time.Time)
		assert.True(t, ok)
		assert.Equal(t, "2026-05-27T00:00:00Z", et.UTC().Format(time.RFC3339))
		assert.Equal(t, 5000, args[3])
		assert.Equal(t, 10, args[4])
	}
}

// 非法时间值容错：忽略，不进 SQL，不报错。
func Test_buildResultsQuery_InvalidTimeIgnored(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{
		StartTime: "not-a-time",
		EndTime:   "",
	}, 100, 0)

	assert.NotContains(t, q, "AND r.window_start >=")
	assert.NotContains(t, q, "AND r.window_start <=")
	assert.Equal(t, []any{id, 100, 0}, args)
}

// T-0193：任务无白名单（ObjectLDNs 空）时 SQL 不含 object_ldn 谓词。
func Test_buildResultsQuery_NoObjectLDNWhitelist(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{}, 100, 0)

	assert.NotContains(t, q, "= ANY")
	assert.Equal(t, []any{id, 100, 0}, args)
}

// T-0193：任务有白名单时 SQL 含 object_ldn = ANY 谓词，占位号顺延，白名单切片入 args。
func Test_buildResultsQuery_WithObjectLDNWhitelist(t *testing.T) {
	id := uuid.New()
	ldns := []string{"Cellid=111172245,PLMN=46068", "Cellid=222,PLMN=46000"}
	q, args := buildResultsQuery(id, resultsFilter{ObjectLDNs: ldns}, 100, 0)

	assert.Contains(t, q, "ELSE r.dimension_key END = ANY($2)")
	assert.Contains(t, q, "ORDER BY r.window_start DESC LIMIT $3 OFFSET $4")
	if assert.Len(t, args, 4) {
		assert.Equal(t, id, args[0])
		assert.Equal(t, ldns, args[1])
		assert.Equal(t, 100, args[2])
		assert.Equal(t, 0, args[3])
	}
}

// T-0193：白名单与其他过滤项叠加时占位号连续递增，object_ldn 排在时间窗口之后。
func Test_buildResultsQuery_ObjectLDNWithOtherFilters(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{
		DeviceSN:   "SN-1",
		StartTime:  "2026-05-20T00:00:00Z",
		ObjectLDNs: []string{"Cellid=111172245,PLMN=46068"},
	}, 100, 0)

	assert.Contains(t, q, "CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END = $2")
	assert.Contains(t, q, "AND r.window_start >= $3")
	assert.Contains(t, q, "ELSE r.dimension_key END = ANY($4)")
	assert.Contains(t, q, "LIMIT $5 OFFSET $6")
	assert.Len(t, args, 6)
	assert.True(t, strings.Index(q, "r.window_start >=") < strings.LastIndex(q, "= ANY($4)"))
}

// 与其他可选过滤项叠加时占位号连续递增。
func Test_buildResultsQuery_CombinedFilters(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{
		DeviceSN:  "SN-1",
		StartTime: "2026-05-20T00:00:00Z",
	}, 100, 0)

	assert.Contains(t, q, "CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END = $2")
	assert.Contains(t, q, "AND r.window_start >= $3")
	assert.Contains(t, q, "LIMIT $4 OFFSET $5")
	assert.Len(t, args, 5)
	// 确保子句顺序：device_sn 先于 time
	assert.True(t, strings.Index(q, "THEN r.device_sn") < strings.Index(q, "AND r.window_start >="))
}

// ── #532 显示侧按任务配置指标集过滤 ──────────────────────────────────────────

// 成功路径：任务配置指标集非空时，SQL 含 metric_path = ANY 谓词，占位号顺延，配置指标切片入 args。
func Test_buildResultsQuery_FilterByTaskMetricPaths(t *testing.T) {
	id := uuid.New()
	metrics := []string{"KGSM0101", "KGSM0102", "KGSM0103"}
	q, args := buildResultsQuery(id, resultsFilter{TaskMetricPaths: metrics}, 100, 0)

	assert.Contains(t, q, "AND r.metric_path = ANY($2)")
	assert.Contains(t, q, "ORDER BY r.window_start DESC LIMIT $3 OFFSET $4")
	if assert.Len(t, args, 4) {
		assert.Equal(t, id, args[0])
		assert.Equal(t, metrics, args[1])
		assert.Equal(t, 100, args[2])
		assert.Equal(t, 0, args[3])
	}
}

// 空配置回退：任务配置指标集为空（历史/边界任务）时不过滤，SQL 不含按配置指标的 ANY 谓词。
func Test_buildResultsQuery_EmptyTaskMetricPathsNoFilter(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{TaskMetricPaths: nil}, 100, 0)

	assert.NotContains(t, q, "r.metric_path = ANY")
	assert.Equal(t, []any{id, 100, 0}, args)
}

// 与用户临时单指标叠加：两道子句各自独立、AND 取交集，占位号连续递增。
func Test_buildResultsQuery_TaskMetricPathsWithUserMetric(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{
		MetricPath:      "KGSM0102",
		TaskMetricPaths: []string{"KGSM0101", "KGSM0102", "KGSM0103"},
	}, 100, 0)

	// 用户临时单指标（=$2 标量）先出，任务配置集（ANY($3)）后出
	assert.Contains(t, q, "AND r.metric_path = $2")
	assert.Contains(t, q, "AND r.metric_path = ANY($3)")
	assert.Contains(t, q, "LIMIT $4 OFFSET $5")
	assert.Len(t, args, 5)
	assert.True(t, strings.Index(q, "r.metric_path = $2") < strings.Index(q, "r.metric_path = ANY"))
}

func Test_buildResultsQuery_UsesMetricScopeWithDashboardFiltersWithoutCount(t *testing.T) {
	id := uuid.New()
	f := resultsFilter{
		Granularity:      "hourly",
		StartTime:        "2026-07-27T18:00:00+08:00",
		EndTime:          "2026-07-27T22:00:00+08:00",
		ProductIDs:       []string{"11111111-1111-1111-1111-111111111111"},
		TaskMetricPaths:  []string{"K1", "K2"},
		Weekdays:         []int{1, 2, 3},
		Hours:            []int{8, 9},
		CalendarTimezone: "Asia/Shanghai",
	}

	dataSQL, dataArgs := buildResultsQuery(id, f, 100, 0)

	for _, want := range []string{
		"AND r.granularity = $2",
		"AND r.window_start >= $3",
		"AND r.window_start < $4",
		"AND r.dimension = 'product' AND r.dimension_key = ANY($5)",
		"AND r.metric_path = ANY($6)",
		"AND EXTRACT(dow FROM (r.window_start AT TIME ZONE $7))::int = ANY($8)",
		"AND EXTRACT(hour FROM (r.window_start AT TIME ZONE $9))::int = ANY($10)",
	} {
		assert.Contains(t, dataSQL, want)
	}
	assert.Contains(t, dataSQL, "ORDER BY r.window_start DESC LIMIT $11 OFFSET $12")
	assert.NotContains(t, dataSQL, "COUNT(*)")
	assert.Len(t, dataArgs, 12)
}

func Test_buildResultsQuery_CalendarFiltersUseConfiguredTimezone(t *testing.T) {
	id := uuid.New()
	f := resultsFilter{
		StartTime:        "2026-07-28T00:00:00+08:00",
		EndTime:          "2026-07-28T01:00:00+08:00",
		Weekdays:         []int{2},
		Hours:            []int{0},
		CalendarTimezone: "Asia/Shanghai",
	}

	q, args := buildResultsQuery(id, f, 100, 0)

	assert.Contains(t, q, "EXTRACT(dow FROM (r.window_start AT TIME ZONE $4))::int = ANY($5)")
	assert.Contains(t, q, "EXTRACT(hour FROM (r.window_start AT TIME ZONE $6))::int = ANY($7)")
	assert.NotContains(t, q, "EXTRACT(dow FROM r.start_time)")
	assert.Contains(t, args, "Asia/Shanghai")
	assert.Contains(t, args, []int{2})
	assert.Contains(t, args, []int{0})
}

// #185：展示侧扩展显示白名单时，会先按当前过滤条件发现结果里真实出现过的版本输出指标。
func Test_buildResultMetricScopeQuery_UsesSameResultFiltersWithoutTaskMetricGate(t *testing.T) {
	id := uuid.New()
	q, args, err := buildResultMetricScopeQuery(id, resultsFilter{
		Dimension:   DimensionProduct,
		MetricPath:  "C000000005",
		Granularity: "hourly",
		StartTime:   "2026-07-27T18:00:00+08:00",
		EndTime:     "2026-07-27T22:00:00+08:00",
		// 原任务指标白名单不能参与这次发现，否则新增进版本输出清单的指标仍会被挡掉。
		TaskMetricPaths: []string{"K-OLD"},
	})

	assert.NoError(t, err)
	assert.Contains(t, q, "SELECT DISTINCT r.metric_path")
	assert.Contains(t, q, "WHERE r.task_id = $1 AND r.dimension = 'product'")
	assert.Contains(t, q, "AND r.metric_path = $2")
	assert.Contains(t, q, "AND r.granularity = $3")
	assert.Contains(t, q, "AND r.window_start >= $4")
	assert.Contains(t, q, "AND r.window_start < $5")
	assert.Contains(t, q, "FROM pm_aggregation_results r")
	assert.Contains(t, q, "FROM pm_aggregation_windows active_window")
	assert.Contains(t, q, "JOIN pm_aggregation_publications published_window")
	assert.Contains(t, q, "JOIN current_versions cv")
	assert.NotContains(t, q, "pm_adhoc_aggregation_results")
	assert.NotContains(t, q, "extra->>'active_version'")
	assert.NotContains(t, q, "r.metric_path = ANY")
	assert.Contains(t, q, "ORDER BY r.metric_path")
	assert.Len(t, args, 5)
	assert.Equal(t, id, args[0])
	assert.Equal(t, "C000000005", args[1])
}

func Test_buildResultsResponsePagination_UsesLimitPlusOne(t *testing.T) {
	items := []adhocResultDTO{{ID: "1"}, {ID: "2"}, {ID: "3"}}

	got, total, truncated := finalizeResultsPage(items, 2, 0)

	assert.True(t, truncated)
	assert.Len(t, got, 2)
	assert.Equal(t, 3, total)
	assert.Equal(t, "1", got[0].ID)
	assert.Equal(t, "2", got[1].ID)
}

func Test_buildResultsResponsePagination_NotTruncated(t *testing.T) {
	items := []adhocResultDTO{{ID: "1"}, {ID: "2"}}

	got, total, truncated := finalizeResultsPage(items, 2, 0)

	assert.False(t, truncated)
	assert.Len(t, got, 2)
	assert.Equal(t, 2, total)
}

func Test_buildResultsResponsePagination_TotalIncludesOffset(t *testing.T) {
	items := []adhocResultDTO{{ID: "101"}, {ID: "102"}, {ID: "103"}}

	got, total, truncated := finalizeResultsPage(items, 2, 100)

	assert.True(t, truncated)
	assert.Len(t, got, 2)
	assert.Equal(t, 103, total)
}
