package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPMPlannedEndIsFoldedIntoPreReleaseBaseline(t *testing.T) {
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)

	require.Empty(t, nonBaselineMigrationFiles(t, "../../migrations", "000001_init_schema.sql"), "pre-release schema changes must be folded into 000001")
	require.Contains(t, string(baseline),
		"ALTER TABLE public.pm_tasks\n    ADD COLUMN IF NOT EXISTS planned_end_at")
	require.Contains(t, string(baseline),
		"ALTER TABLE public.pm_aggregation_tasks\n    ADD COLUMN IF NOT EXISTS planned_end_at")
}
