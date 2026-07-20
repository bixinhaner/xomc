package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	adminRoleID    = "10000000-0000-0000-0000-000000000001"
	operatorRoleID = "10000000-0000-0000-0000-000000000002"
	viewerRoleID   = "10000000-0000-0000-0000-000000000003"
)

func rbacRepairSeedPath() string {
	return filepath.Join("..", "..", "migrations", "seed", "000002_repair_builtin_role_api_permissions.sql")
}

func readRBACRepairSeed(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(rbacRepairSeedPath())
	require.NoError(t, err)
	return string(raw)
}

func gooseUpSQL(t *testing.T, raw string) string {
	t.Helper()
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"
	up := strings.Index(raw, upMarker)
	down := strings.Index(raw, downMarker)
	require.GreaterOrEqual(t, up, 0)
	require.Greater(t, down, up)
	return raw[up+len(upMarker) : down]
}

func TestRBACRepairSeedContract(t *testing.T) {
	raw := readRBACRepairSeed(t)
	up := gooseUpSQL(t, raw)

	assert.Contains(t, up, adminRoleID)
	assert.Contains(t, up, operatorRoleID)
	assert.Contains(t, up, viewerRoleID)
	assert.Equal(t, 3, strings.Count(strings.ToUpper(up), "ON CONFLICT (ROLE_ID, ENDPOINT_ID) DO NOTHING"))
	assert.Contains(t, up, "SELECT '"+adminRoleID+"'::uuid, ae.id")
	assert.Contains(t, up, "SELECT '"+operatorRoleID+"'::uuid, ae.id")
	assert.Contains(t, up, "SELECT '"+viewerRoleID+"'::uuid, ae.id")
	assert.Contains(t, up, "WHERE ae.method = 'GET'")

	down := raw[strings.Index(raw, "-- +goose Down"):]
	assert.Contains(t, down, "SELECT 1;")
	assert.NotContains(t, strings.ToUpper(down), "DELETE FROM")
}

func TestRBACRepairSeedDatabaseBehavior(t *testing.T) {
	pool := SetupTestDB(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	roles := []struct {
		id, name, description string
	}{
		{adminRoleID, "admin", "System administrator with full access"},
		{operatorRoleID, "operator", "Operator with read/write access to operational resources"},
		{viewerRoleID, "viewer", "Read-only viewer"},
	}
	for _, role := range roles {
		_, err = tx.Exec(ctx, `
			INSERT INTO roles (id, name, description, is_system)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (id) DO NOTHING`,
			role.id, role.name, role.description)
		require.NoError(t, err)
	}

	fixtures := []struct {
		id, path, method string
	}{
		{"7f200000-0000-0000-0000-000000000001", "/test/rbac-seed/get", "GET"},
		{"7f200000-0000-0000-0000-000000000002", "/test/rbac-seed/post", "POST"},
	}
	for _, fixture := range fixtures {
		_, err = tx.Exec(ctx, `
			INSERT INTO api_endpoints (id, path, method, name, description, api_group)
			VALUES ($1, $2, $3, $4, '', 'test')
			ON CONFLICT (id) DO UPDATE SET method = EXCLUDED.method`,
			fixture.id, fixture.path, fixture.method, fixture.method+" test fixture")
		require.NoError(t, err)
	}

	up := gooseUpSQL(t, readRBACRepairSeed(t))
	_, err = tx.Exec(ctx, up)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, up)
	require.NoError(t, err)

	assertions := []struct {
		roleID, endpointID string
		want               int
	}{
		{adminRoleID, fixtures[0].id, 1},
		{adminRoleID, fixtures[1].id, 1},
		{operatorRoleID, fixtures[0].id, 1},
		{operatorRoleID, fixtures[1].id, 1},
		{viewerRoleID, fixtures[0].id, 1},
		{viewerRoleID, fixtures[1].id, 0},
	}
	for _, assertion := range assertions {
		var got int
		err = tx.QueryRow(ctx, `
			SELECT count(*) FROM role_api_permissions
			WHERE role_id = $1 AND endpoint_id = $2`,
			assertion.roleID, assertion.endpointID).Scan(&got)
		require.NoError(t, err)
		assert.Equal(t, assertion.want, got)
	}
}
