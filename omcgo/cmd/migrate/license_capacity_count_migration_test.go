package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLicenseCapacityOnlineIndexesCoverProductID(t *testing.T) {
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)

	sql := string(baseline)
	for _, indexName := range []string{
		"idx_devices_is_online",
		"devices_cmcc_is_online_idx",
		"devices_ctcc_is_online_idx",
		"devices_cucc_is_online_idx",
		"devices_other_is_online_idx",
		"idx_devices_other_is_online",
	} {
		pattern := regexp.MustCompile(`(?is)CREATE\s+INDEX\s+` + regexp.QuoteMeta(indexName) +
			`\s+ON\s+(?:ONLY\s+)?public\.devices(?:_[a-z]+)?\s+USING\s+btree\s*\(\s*is_online\s*\)\s+INCLUDE\s*\(\s*product_id\s*\)\s+WHERE\s+\(\(is_online\s*=\s*true\)\s+AND\s+\(deleted_at\s+IS\s+NULL\)\)`)
		require.Regexp(t, pattern, sql,
			"%s must support index-only online device capacity counts and product_id grouping", indexName)
	}

	for _, attach := range []string{
		"ALTER INDEX public.idx_devices_is_online ATTACH PARTITION public.devices_cmcc_is_online_idx;",
		"ALTER INDEX public.idx_devices_is_online ATTACH PARTITION public.devices_ctcc_is_online_idx;",
		"ALTER INDEX public.idx_devices_is_online ATTACH PARTITION public.devices_cucc_is_online_idx;",
		"ALTER INDEX public.idx_devices_is_online ATTACH PARTITION public.devices_other_is_online_idx;",
	} {
		require.Contains(t, sql, attach)
	}
}
