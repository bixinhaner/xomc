package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltInRoleAPIPermissionSeedContract(t *testing.T) {
	const migrationPath = "../../migrations/seed/000001_init_seed.sql"

	contents, err := os.ReadFile(migrationPath)
	require.NoError(t, err)
	sql := string(contents)

	assert.Contains(t, sql, "-- +goose Up")
	assert.Contains(t, sql, "10000000-0000-0000-0000-000000000001")
	assert.Contains(t, sql, "10000000-0000-0000-0000-000000000002")
	assert.Contains(t, sql, "10000000-0000-0000-0000-000000000003")
	assert.GreaterOrEqual(t, strings.Count(sql, "ON CONFLICT (role_id, endpoint_id) DO NOTHING"), 3)
	assert.Contains(t, sql, "WHERE ae.method = 'GET'")

	assert.Contains(t, sql, "-- +goose Down")
}
