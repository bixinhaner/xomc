package topology

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/global"
)

type recordingPgxExecutor struct {
	sql  string
	args []any
}

func (e *recordingPgxExecutor) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	e.sql = sql
	e.args = arguments
	return pgconn.NewCommandTag("UPDATE 3"), nil
}

func (e *recordingPgxExecutor) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	panic("Query is not used by this test")
}

// TestBuildBatchSortSQL covers the CASE WHEN SQL built by BatchSort. Regression
// for issue #120: bare $N placeholders in the CASE branches were inferred as
// text by PG, so assigning to the integer sort_order column always failed with
// SQLSTATE 42804 — placeholders must carry explicit ::int / ::uuid casts.
func TestBuildBatchSortSQL(t *testing.T) {
	t.Parallel()

	id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	cases := []struct {
		name     string
		items    []BatchSortItem
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name:  "single item casts value to int and id to uuid",
			items: []BatchSortItem{{ID: id1, SortOrder: 9}},
			wantSQL: "UPDATE device_groups SET sort_order = CASE " +
				"WHEN id = $1::uuid THEN $2::int" +
				" END WHERE id IN ($1::uuid)",
			wantArgs: []interface{}{id1, 9},
		},
		{
			name: "multiple items number placeholders pairwise",
			items: []BatchSortItem{
				{ID: id1, SortOrder: 1},
				{ID: id2, SortOrder: 2},
			},
			wantSQL: "UPDATE device_groups SET sort_order = CASE " +
				"WHEN id = $1::uuid THEN $2::int WHEN id = $3::uuid THEN $4::int" +
				" END WHERE id IN ($1::uuid, $3::uuid)",
			wantArgs: []interface{}{id1, 1, id2, 2},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotSQL, gotArgs := buildBatchSortSQL(tc.items)

			assert.Equal(t, tc.wantSQL, gotSQL)
			require.Len(t, gotArgs, len(tc.wantArgs))
			assert.Equal(t, tc.wantArgs, gotArgs)
		})
	}
}

func TestMoveGroupDevicesToDefaultTx_UpdatesMembershipsToDefaultGroup(t *testing.T) {
	t.Parallel()

	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ex := &recordingPgxExecutor{}

	affected, err := moveGroupDevicesToDefaultTx(context.Background(), ex, []uuid.UUID{groupID})
	require.NoError(t, err)

	assert.Equal(t, int64(3), affected)
	assert.Contains(t, ex.sql, "UPDATE device_group_members")
	assert.Contains(t, ex.sql, "SET group_id = $1")
	assert.Contains(t, ex.sql, "WHERE group_id = ANY($2)")
	assert.NotContains(t, ex.sql, "DELETE FROM device_group_members")
	require.Len(t, ex.args, 2)
	assert.Equal(t, uuid.MustParse(global.DefaultLevel2GroupID), ex.args[0])
	assert.Equal(t, []uuid.UUID{groupID}, ex.args[1])
}

func TestGetTreeWithCounts_DefaultGroupCountsLegacyUngroupedDevices(t *testing.T) {
	t.Parallel()

	assert.Contains(t, getTreeWithCountsRawSQL, "WHEN dg.id = $1::uuid")
	assert.Contains(t, getTreeWithCountsRawSQL, "NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = d.id)")
}

func TestDeviceGroupCountsCache_InvalidationRejectsInFlightStaleResult(t *testing.T) {
	t.Parallel()

	repo := &PgDeviceGroupRepository{}
	_, hit, generation := repo.cachedTreeWithCounts()
	require.False(t, hit)

	// 模拟查询已经在缓存失效前启动，但在失效后才尝试回填旧结果。
	repo.InvalidateDeviceGroupCounts()
	repo.storeTreeWithCountsCache([]DeviceGroup{{Name: "stale"}}, generation)

	_, hit, _ = repo.cachedTreeWithCounts()
	assert.False(t, hit, "失效前启动的查询结果不能在失效后重新写回缓存")
}
