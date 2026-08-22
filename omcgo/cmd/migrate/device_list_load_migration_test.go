package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeviceListGroupMembershipIndexCoversListProjection(t *testing.T) {
	baseline, err := os.ReadFile("../../migrations/000001_init_schema.sql")
	require.NoError(t, err)

	sql := string(baseline)
	coveringIndex := regexp.MustCompile(`(?is)CREATE\s+INDEX\s+idx_dgm_device\s+ON\s+public\.device_group_members\s+USING\s+btree\s*\(\s*device_id\s*\)\s+INCLUDE\s*\(\s*group_id\s*,\s*source_type\s*\)`)
	require.Regexp(t, coveringIndex, sql,
		"device list joins device_group_members by device_id and projects group_id/source_type; the index must cover those columns")
}
