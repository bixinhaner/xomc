package adhoc

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-0194：编辑任务 UPDATE SQL 守门口径单测（纯函数，不依赖 DB）。
//
// 覆盖：
//   1. 自建任务全字段进 SET（task_name/device_sns/metric_paths/granularities/object_ldns/window_*）。
//   2. 内置任务只 SET metric_paths + updated_at，其余结构性列不进 SET（保持原值）。
//   3. mode/technology/dimension/is_builtin/expire_days 任何情况都不进 SET。
//   4. WHERE 限定 task_subtype='adhoc_aggregation'。

func Test_buildUpdateSQL_Adhoc_AllFields(t *testing.T) {
	id := uuid.New()
	req := UpdateRequest{
		IsBuiltin:     false,
		Name:          "edited",
		DeviceSNs:     []string{"S1", "S2"},
		MetricPaths:   []string{"K1", "K2"},
		Granularities: []string{"daily"},
		ObjectLDNs:    []string{"LDN-A"},
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, _, err := buildUpdateSQL(id, req)
	require.NoError(t, err)

	setClause := sql
	if idx := strings.Index(sql, "WHERE"); idx >= 0 {
		setClause = sql[:idx]
	}
	for _, col := range []string{"metric_paths", "task_name", "device_sns", "granularities", "object_ldns", "window_start", "window_end", "updated_at"} {
		assert.Contains(t, setClause, col, "自建任务应更新 %s", col)
	}
	// 结构性不可改字段绝不出现在 SET
	for _, col := range []string{"mode =", "technology =", "dimension =", "is_builtin =", "expire_days ="} {
		assert.NotContains(t, setClause, col, "不可改字段 %s 不应进 SET", col)
	}
	assert.Contains(t, sql, "task_subtype", "WHERE 应限定 task_subtype")
}

func Test_buildUpdateSQL_Builtin_OnlyMetricPaths(t *testing.T) {
	id := uuid.New()
	req := UpdateRequest{
		IsBuiltin:   true,
		Name:        "ignored",          // 内置不应进 SET
		DeviceSNs:   []string{"X1"},      // 内置不应进 SET
		MetricPaths: []string{"K9"},      // 唯一可改
		Granularities: []string{"hourly"}, // 内置不应进 SET
	}
	sql, _, err := buildUpdateSQL(id, req)
	require.NoError(t, err)

	assert.Contains(t, sql, "metric_paths", "内置应更新 metric_paths")
	assert.Contains(t, sql, "updated_at", "内置应更新 updated_at")
	// 结构性列不进 SET（device_sns/task_name/granularities/object_ldns/window_*）
	setClause := sql
	if idx := strings.Index(sql, "WHERE"); idx >= 0 {
		setClause = sql[:idx]
	}
	for _, col := range []string{"task_name", "device_sns", "granularities", "object_ldns", "window_start", "window_end"} {
		assert.NotContains(t, setClause, col, "内置任务不应更新 %s", col)
	}
}
