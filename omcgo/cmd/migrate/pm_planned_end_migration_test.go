package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPMPlannedEndUsesForwardMigrationAfterConsolidatedBaseline(t *testing.T) {
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	migration, err := os.ReadFile("../../migrations/000002_pm_task_planned_end.sql")
	require.NoError(t, err)

	require.NotContains(t, string(baseline), "planned_end_at",
		"already-applied baseline must not be rewritten for a post-baseline column")
	require.Contains(t, string(migration),
		"ALTER TABLE public.pm_tasks\n    ADD COLUMN IF NOT EXISTS planned_end_at")
	require.Contains(t, string(migration),
		"ALTER TABLE public.pm_aggregation_tasks\n    ADD COLUMN IF NOT EXISTS planned_end_at")
	require.Contains(t, string(migration), "-- +goose Down")
}
