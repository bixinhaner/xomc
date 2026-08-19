package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreReleaseMigrationStreamsOnlyKeepBaselines(t *testing.T) {
	streams := map[string]string{
		"../../migrations":      "000001_init_schema.sql",
		"../../migrations/seed": "000001_init_seed.sql",
		"../../migrations/tsdb": "000001_tsdb_schema.sql",
	}
	for stream, baseline := range streams {
		require.Emptyf(t, nonBaselineMigrationFiles(t, stream, baseline), "%s must not contain extra migrations before release versioning is enabled", stream)
	}
}

func TestMainBaselineProductsConstraintAllowsEmptyIndicatorDeviceType(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)
	sql := string(data)

	require.Contains(t, sql,
		"CONSTRAINT chk_products_indicator_device_type CHECK (indicator_device_type = '' OR indicator_device_type IN ('enb', 'gsm', 'gnb'))")
	require.NotContains(t, sql, "ARRAY[(''::text,")
}

func nonBaselineMigrationFiles(t *testing.T, stream, baseline string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(stream, "[0-9]*.sql"))
	require.NoError(t, err)
	var extra []string
	for _, file := range files {
		if !strings.EqualFold(filepath.Base(file), baseline) {
			extra = append(extra, file)
		}
	}
	return extra
}
