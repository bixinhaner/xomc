package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltInRoleAPIPermissionSeedContract(t *testing.T) {
	const migrationPath = "../../migrations/seed/000002_repair_builtin_role_api_permission_drift.sql"

	contents, err := os.ReadFile(migrationPath)
	require.NoError(t, err)
	sql := string(contents)

	assert.Contains(t, sql, "-- +goose Up")
	assert.Contains(t, sql, "10000000-0000-0000-0000-000000000001")
	assert.Contains(t, sql, "10000000-0000-0000-0000-000000000002")
	assert.Contains(t, sql, "10000000-0000-0000-0000-000000000003")
	assert.GreaterOrEqual(t, strings.Count(sql, "ON CONFLICT (role_id, endpoint_id) DO NOTHING"), 3)
	assert.Contains(t, sql, "WHERE ae.method = 'GET'")

	parts := strings.Split(sql, "-- +goose Down")
	require.Len(t, parts, 2)
	assert.NotContains(t, strings.ToUpper(parts[1]), "DELETE")
	assert.Contains(t, parts[1], "SELECT 1")
}
