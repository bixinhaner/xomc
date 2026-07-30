package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStandardParamsUpdatedFieldsIsFoldedIntoPreReleaseBaseline(t *testing.T) {
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	sql := string(baseline)

	require.Contains(t, sql, "CREATE TABLE public.standard_params")
	require.Contains(t, sql, "updated_fields text[] DEFAULT '{}'::text[] NOT NULL")
	require.Contains(t, sql, "COMMENT ON COLUMN public.standard_params.updated_fields")
}
