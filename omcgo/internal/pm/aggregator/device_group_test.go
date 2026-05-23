package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// SQL 文本断言：device_group 维度聚合 SQL 必须 JOIN devices + device_group_members，
// SELECT 列 + ON CONFLICT 含 device_group_id。

func Test_buildDeviceGroupSQL_HourlyGroupTableHasIDAndJoins(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	sql, args := buildDeviceGroupSQL("pm_metrics_hourly", "pm_group_metrics_hourly", w)

	// 双 JOIN
	assert.Contains(t, sql, "JOIN devices d")
	assert.Contains(t, sql, "JOIN device_group_members dgm")
	assert.Contains(t, sql, "ON d.oui = m.device_oui AND d.serial_number = m.device_sn")
	assert.Contains(t, sql, "ON dgm.device_id = d.id")

	// GROUP BY 维度键
	assert.Contains(t, sql, "GROUP BY dgm.group_id, m.metric_path, m.statis_type")

	// hourly group 表含 id 列（hypertable）
	assert.Contains(t, sql, "INSERT INTO pm_group_metrics_hourly (id, device_group_id")
	assert.Contains(t, sql, "gen_random_uuid()")

	// ON CONFLICT 含 device_group_id（不含 device_oui/sn）
	assert.Contains(t, sql, "ON CONFLICT (device_group_id, metric_path, granularity, end_time, time)")

	// args 顺序与 buildCountersSQL 一致：granularity / bktStart / bktEnd / whereStart / whereEnd
	assert.Equal(t, []any{"hourly", w.Start, w.End, w.Start, w.End}, args)
}

func Test_buildDeviceGroupSQL_DailyGroupTableNoID(t *testing.T) {
	w := WindowSpec{
		Granularity: metrics.GranularityDaily,
		Start:       time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
	}
	sql, _ := buildDeviceGroupSQL("pm_metrics_daily", "pm_group_metrics_daily", w)

	// daily group 表无 id 列
	assert.Contains(t, sql, "INSERT INTO pm_group_metrics_daily (device_group_id")
	assert.NotContains(t, sql, "gen_random_uuid()")
	// daily 冲突目标 PK 4 列（无 time）
	assert.Contains(t, sql, "ON CONFLICT (device_group_id, metric_path, granularity, end_time)")
}

// AggregateDeviceGroup 通过 stub DB 验证 SQL + args 透传到 Exec。
// （真实 DB 验证留 Task 5 集成测试。）

func Test_AggregateDeviceGroup_PassesThroughToExec(t *testing.T) {
	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 2")}
	a := New(db, nil, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	n, err := a.AggregateDeviceGroup(context.Background(), "pm_metrics_hourly", "pm_group_metrics_hourly", w)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Contains(t, db.execSQL, "INSERT INTO pm_group_metrics_hourly")
	assert.Contains(t, db.execSQL, "JOIN device_group_members dgm")
	assert.Equal(t, []any{"hourly", w.Start, w.End, w.Start, w.End}, db.execArgs)
}
