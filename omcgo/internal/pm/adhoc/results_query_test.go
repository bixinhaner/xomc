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
	// 两个 LEFT JOIN：product 按 id 对 product_id；device_group 按 'DeviceGroup='||id 对 object_ldn
	assert.Contains(t, q, "LEFT JOIN products p ON p.id = r.product_id")
	assert.Contains(t, q, "LEFT JOIN device_groups g ON ('DeviceGroup=' || g.id::text) = r.object_ldn")
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
	assert.Contains(t, q, "AND r.time <= $3")
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
	assert.Contains(t, q, "AND r.time <= $6")
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
