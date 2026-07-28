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
	assert.Contains(t, q, "ORDER BY r.time DESC LIMIT $2 OFFSET $3")
	// args = [taskID, limit, offset]
	assert.Equal(t, []any{id, 100, 0}, args)
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
	assert.Contains(t, q, "LEFT JOIN product_dim p ON p.id = r.product_id")
	assert.Contains(t, q, "LEFT JOIN device_group_dim g ON ('DeviceGroup=' || g.id::text) = split_part(r.object_ldn, ',', 1)")
	// 主表起了别名 r
	assert.Contains(t, q, "FROM pm_adhoc_aggregation_results r")
}

// 带大时间段：起止时间转为 time.Time 占位参数，子句出现且占位号顺延。
func Test_buildResultsQuery_WithTimeRange(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{
		StartTime: "2026-05-20T00:00:00Z",
		EndTime:   "2026-05-27T00:00:00Z",
	}, 5000, 10)

	assert.Contains(t, q, "AND r.time >= $2")
	assert.Contains(t, q, "AND r.time < $3")
	assert.NotContains(t, q, "AND r.time <= $3")
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

	assert.NotContains(t, q, "AND r.time >=")
	assert.NotContains(t, q, "AND r.time <=")
	assert.Equal(t, []any{id, 100, 0}, args)
}

// T-0193：任务无白名单（ObjectLDNs 空）时 SQL 不含 object_ldn 谓词。
func Test_buildResultsQuery_NoObjectLDNWhitelist(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{}, 100, 0)

	assert.NotContains(t, q, "r.object_ldn = ANY")
	assert.Equal(t, []any{id, 100, 0}, args)
}

// T-0193：任务有白名单时 SQL 含 object_ldn = ANY 谓词，占位号顺延，白名单切片入 args。
func Test_buildResultsQuery_WithObjectLDNWhitelist(t *testing.T) {
	id := uuid.New()
	ldns := []string{"Cellid=111172245,PLMN=46068", "Cellid=222,PLMN=46000"}
	q, args := buildResultsQuery(id, resultsFilter{ObjectLDNs: ldns}, 100, 0)

	assert.Contains(t, q, "AND r.object_ldn = ANY($2)")
	assert.Contains(t, q, "ORDER BY r.time DESC LIMIT $3 OFFSET $4")
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

	assert.Contains(t, q, "AND r.device_sn = $2")
	assert.Contains(t, q, "AND r.time >= $3")
	assert.Contains(t, q, "AND r.object_ldn = ANY($4)")
	assert.Contains(t, q, "LIMIT $5 OFFSET $6")
	assert.Len(t, args, 6)
	assert.True(t, strings.Index(q, "r.time >=") < strings.Index(q, "r.object_ldn = ANY"))
}

// 与其他可选过滤项叠加时占位号连续递增。
func Test_buildResultsQuery_CombinedFilters(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{
		DeviceSN:  "SN-1",
		StartTime: "2026-05-20T00:00:00Z",
	}, 100, 0)

	assert.Contains(t, q, "AND r.device_sn = $2")
	assert.Contains(t, q, "AND r.time >= $3")
	assert.Contains(t, q, "LIMIT $4 OFFSET $5")
	assert.Len(t, args, 5)
	// 确保子句顺序：device_sn 先于 time
	assert.True(t, strings.Index(q, "AND r.device_sn") < strings.Index(q, "AND r.time >="))
}

// ── buildResultsCountQuery（T-0194 截断诚实提示）───────────────────────────────

// COUNT 查询：SELECT COUNT(*)，无 ORDER BY / LIMIT / OFFSET / LEFT JOIN，只保留同 WHERE。
func Test_buildResultsCountQuery_BareTaskID(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsCountQuery(id, resultsFilter{})

	assert.Contains(t, q, "SELECT COUNT(*)")
	assert.Contains(t, q, "WHERE r.task_id = $1")
	assert.NotContains(t, q, "ORDER BY")
	assert.NotContains(t, q, "LIMIT")
	assert.NotContains(t, q, "OFFSET")
	assert.NotContains(t, q, "LEFT JOIN")
	assert.Equal(t, []any{id}, args)
}

// COUNT 与 buildResultsQuery 用同一套 WHERE：同样的过滤项产出同样的谓词与占位顺序（去分页）。
func Test_buildResultsCountQuery_SameWhereAsData(t *testing.T) {
	id := uuid.New()
	f := resultsFilter{
		DeviceSN:    "SN-1",
		MetricPath:  "K1001",
		Granularity: "hourly",
		StartTime:   "2026-05-20T00:00:00Z",
		EndTime:     "2026-05-21T00:00:00Z",
		ObjectLDNs:  []string{"Cellid=1,PLMN=46000"},
	}
	q, args := buildResultsCountQuery(id, f)

	assert.Contains(t, q, "AND r.device_sn = $2")
	assert.Contains(t, q, "AND r.metric_path = $3")
	assert.Contains(t, q, "AND r.granularity = $4")
	assert.Contains(t, q, "AND r.time >= $5")
	assert.Contains(t, q, "AND r.time < $6")
	assert.NotContains(t, q, "AND r.time <= $6")
	assert.Contains(t, q, "AND r.object_ldn = ANY($7)")
	// args = [taskID, SN, metricPath, granularity, start, end, ldns]，无 limit/offset 尾巴
	assert.Len(t, args, 7)
	assert.Equal(t, id, args[0])
	assert.Equal(t, "SN-1", args[1])
}

// 非法时间值与数据查询一致地被忽略（容错，不进 WHERE）。
func Test_buildResultsCountQuery_InvalidTimeIgnored(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsCountQuery(id, resultsFilter{StartTime: "not-a-time"})
	assert.NotContains(t, q, "AND r.time >=")
	assert.Equal(t, []any{id}, args)
}

// ── #532 显示侧按任务配置指标集过滤 ──────────────────────────────────────────

// 成功路径：任务配置指标集非空时，SQL 含 metric_path = ANY 谓词，占位号顺延，配置指标切片入 args。
func Test_buildResultsQuery_FilterByTaskMetricPaths(t *testing.T) {
	id := uuid.New()
	metrics := []string{"KGSM0101", "KGSM0102", "KGSM0103"}
	q, args := buildResultsQuery(id, resultsFilter{TaskMetricPaths: metrics}, 100, 0)

	assert.Contains(t, q, "AND r.metric_path = ANY($2)")
	assert.Contains(t, q, "ORDER BY r.time DESC LIMIT $3 OFFSET $4")
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

// count 与数据查询同口径：任务配置指标集子句也出现在 COUNT 查询里（否则 count 与数据对不上）。
func Test_buildResultsCountQuery_FilterByTaskMetricPaths(t *testing.T) {
	id := uuid.New()
	metrics := []string{"KGSM0101", "KGSM0102"}
	q, args := buildResultsCountQuery(id, resultsFilter{TaskMetricPaths: metrics})

	assert.Contains(t, q, "SELECT COUNT(*)")
	assert.Contains(t, q, "AND r.metric_path = ANY($2)")
	assert.NotContains(t, q, "ORDER BY")
	assert.NotContains(t, q, "LIMIT")
	if assert.Len(t, args, 2) {
		assert.Equal(t, id, args[0])
		assert.Equal(t, metrics, args[1])
	}
}

func Test_buildResultsQueryAndCountQuery_UseSameMetricScopeWithDashboardFilters(t *testing.T) {
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
	countSQL, countArgs := buildResultsCountQuery(id, f)

	for _, want := range []string{
		"AND r.granularity = $2",
		"AND r.time >= $3",
		"AND r.time < $4",
		"AND r.product_id = ANY($5)",
		"AND r.metric_path = ANY($6)",
		"AND EXTRACT(dow FROM (r.start_time AT TIME ZONE $7))::int = ANY($8)",
		"AND EXTRACT(hour FROM (r.start_time AT TIME ZONE $9))::int = ANY($10)",
	} {
		assert.Contains(t, dataSQL, want)
		assert.Contains(t, countSQL, want)
	}
	assert.Contains(t, dataSQL, "ORDER BY r.time DESC LIMIT $11 OFFSET $12")
	assert.NotContains(t, countSQL, "ORDER BY")
	assert.Equal(t, dataArgs[:len(dataArgs)-2], countArgs)
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

	assert.Contains(t, q, "EXTRACT(dow FROM (r.start_time AT TIME ZONE $4))::int = ANY($5)")
	assert.Contains(t, q, "EXTRACT(hour FROM (r.start_time AT TIME ZONE $6))::int = ANY($7)")
	assert.NotContains(t, q, "EXTRACT(dow FROM r.start_time)")
	assert.Contains(t, args, "Asia/Shanghai")
	assert.Contains(t, args, []int{2})
	assert.Contains(t, args, []int{0})
}

// count 空配置回退：与数据查询一致，配置集为空时不过滤。
func Test_buildResultsCountQuery_EmptyTaskMetricPathsNoFilter(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsCountQuery(id, resultsFilter{TaskMetricPaths: nil})

	assert.NotContains(t, q, "r.metric_path = ANY")
	assert.Equal(t, []any{id}, args)
}

// #185：展示侧扩展显示白名单时，会先按当前过滤条件发现结果里真实出现过的版本输出指标。
func Test_buildResultMetricScopeQuery_UsesSameResultFiltersWithoutTaskMetricGate(t *testing.T) {
	id := uuid.New()
	q, args, err := buildResultMetricScopeQuery(id, resultsFilter{
		MetricPath:  "C000000005",
		Granularity: "hourly",
		StartTime:   "2026-07-27T18:00:00+08:00",
		EndTime:     "2026-07-27T22:00:00+08:00",
		// 原任务指标白名单不能参与这次发现，否则新增进版本输出清单的指标仍会被挡掉。
		TaskMetricPaths: []string{"K-OLD"},
	})

	assert.NoError(t, err)
	assert.Contains(t, q, "SELECT DISTINCT r.metric_path")
	assert.Contains(t, q, "AND r.metric_path = $2")
	assert.Contains(t, q, "AND r.granularity = $3")
	assert.Contains(t, q, "AND r.time >= $4")
	assert.Contains(t, q, "AND r.time < $5")
	assert.NotContains(t, q, "r.metric_path = ANY")
	assert.Contains(t, q, "ORDER BY r.metric_path")
	assert.Len(t, args, 5)
	assert.Equal(t, id.String(), args[0])
	assert.Equal(t, "C000000005", args[1])
}
