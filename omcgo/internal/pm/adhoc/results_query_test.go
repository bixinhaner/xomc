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

	assert.NotContains(t, q, "AND time >=")
	assert.NotContains(t, q, "AND time <=")
	assert.Contains(t, q, "WHERE task_id = $1")
	assert.Contains(t, q, "ORDER BY time DESC LIMIT $2 OFFSET $3")
	// args = [taskID, limit, offset]
	assert.Equal(t, []any{id, 100, 0}, args)
}

// 带大时间段：起止时间转为 time.Time 占位参数，子句出现且占位号顺延。
func Test_buildResultsQuery_WithTimeRange(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{
		StartTime: "2026-05-20T00:00:00Z",
		EndTime:   "2026-05-27T00:00:00Z",
	}, 5000, 10)

	assert.Contains(t, q, "AND time >= $2")
	assert.Contains(t, q, "AND time <= $3")
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

	assert.NotContains(t, q, "AND time >=")
	assert.NotContains(t, q, "AND time <=")
	assert.Equal(t, []any{id, 100, 0}, args)
}

// 与其他可选过滤项叠加时占位号连续递增。
func Test_buildResultsQuery_CombinedFilters(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{
		DeviceSN:  "SN-1",
		StartTime: "2026-05-20T00:00:00Z",
	}, 100, 0)

	assert.Contains(t, q, "AND device_sn = $2")
	assert.Contains(t, q, "AND time >= $3")
	assert.Contains(t, q, "LIMIT $4 OFFSET $5")
	assert.Len(t, args, 5)
	// 确保子句顺序：device_sn 先于 time
	assert.True(t, strings.Index(q, "device_sn") < strings.Index(q, "time >="))
}
