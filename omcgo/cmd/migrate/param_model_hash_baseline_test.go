package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParamModelContentHashIsFoldedIntoPreReleaseBaseline(t *testing.T) {
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)

	sql := string(baseline)
	require.Empty(t, nonBaselineMigrationFiles(t, "../../migrations", "000001_init_schema.sql"), "pre-release schema changes must be folded into 000001")
	require.Contains(t, sql, "content_hash character varying(64)")
	require.Contains(t, sql, "COMMENT ON COLUMN public.param_models.content_hash IS")
	require.NotContains(t, sql, "ALTER TABLE public.param_models\n    ADD COLUMN IF NOT EXISTS content_hash")
}
