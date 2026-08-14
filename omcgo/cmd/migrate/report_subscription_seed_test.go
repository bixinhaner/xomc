package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReportSubscriptionEndpointSeedContract(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)
	seed := string(contents)

	for _, endpoint := range []string{
		"GET /api/v1/pm/query-templates/:id/report-subscription",
		"PUT /api/v1/pm/query-templates/:id/report-subscription",
		"DELETE /api/v1/pm/query-templates/:id/report-subscription",
		"GET /api/v1/pm/query-templates/:id/report-subscription/runs",
	} {
		require.Contains(t, seed, endpoint)
	}

	endpointIndex := strings.Index(seed, "KPI query report subscription endpoints")
	roleGrantIndex := strings.Index(seed, "-- admin：沿用内置角色兼容基线")
	require.NotEqual(t, -1, endpointIndex)
	require.NotEqual(t, -1, roleGrantIndex)
	require.Less(t, endpointIndex, roleGrantIndex, "endpoints must exist before built-in role grants execute")
}
