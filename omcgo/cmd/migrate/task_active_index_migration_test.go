package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskActiveCreatedIndexMigration(t *testing.T) {
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)

	activeTaskIndex := regexp.MustCompile(`(?is)CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\s+idx_device_tasks_active_created_id\s+ON\s+public\.device_tasks\s*\(\s*created_at\s*,\s*id\s*\)\s+WHERE\s+status\s+IN\s*\(\s*'pending'\s*,\s*'sent'\s*\)`)
	require.Regexp(t, activeTaskIndex, string(baseline),
		"active task reconciliation requires a stable partial index for pending and sent tasks")
}
