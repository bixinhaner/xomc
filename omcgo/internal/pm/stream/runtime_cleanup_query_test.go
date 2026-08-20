package stream

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRuntimeCleanupQueriesAreBoundedAndSkipLocked(t *testing.T) {
	before := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	legacyBefore := before.Add(-24 * time.Hour)

	tests := []struct {
		name  string
		build func() (string, []interface{}, error)
		want  []string
	}{
		{
			name: "normalized outbox",
			build: func() (string, []interface{}, error) {
				return buildOutboxCleanupBatch(before, legacyBefore, 500)
			},
			want: []string{
				"WITH candidates AS", "LIMIT 500", "FOR UPDATE SKIP LOCKED",
				"DELETE FROM pm_aggregation_outbox", "consumed_at IS NOT NULL",
			},
		},
		{
			name: "rollup outbox",
			build: func() (string, []interface{}, error) {
				return buildRollupOutboxCleanupBatch(before, 500)
			},
			want: []string{
				"WITH candidates AS", "LIMIT 500", "FOR UPDATE SKIP LOCKED",
				"DELETE FROM pm_aggregation_rollup_outbox", "consumed_at IS NOT NULL",
			},
		},
		{
			name: "terminal windows",
			build: func() (string, []interface{}, error) {
				return buildTerminalWindowCleanupBatch(before, 500)
			},
			want: []string{
				"WITH candidates AS", "LIMIT 500", "FOR UPDATE SKIP LOCKED",
				"DELETE FROM pm_aggregation_windows", "NOT EXISTS", "recovery_terminal_at",
				"result.task_version_id = pm_aggregation_windows.task_version_id",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query, args, err := tc.build()
			require.NoError(t, err)
			query = strings.Join(strings.Fields(query), " ")
			for _, fragment := range tc.want {
				require.Contains(t, query, fragment)
			}
			if tc.name == "terminal windows" {
				require.NotContains(t, query, "result.entity_key")
				serializedArgs := fmt.Sprint(args)
				for _, status := range []string{"published", "retired", "orphaned", "abandoned"} {
					require.Contains(t, serializedArgs, status)
				}
			}
		})
	}
}

func TestReplayCleanupSelectsOneExpiredChunk(t *testing.T) {
	query, args, err := buildReplayChunkCandidateQuery(
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.Len(t, args, 3)
	query = strings.Join(strings.Fields(query), " ")
	require.Contains(t, query, "timescaledb_information.chunks")
	require.Contains(t, query, "hypertable_name = $1")
	require.Contains(t, query, "range_end <= $3")
	require.Contains(t, query, "ORDER BY range_start")
	require.Contains(t, query, "LIMIT 1")
}
