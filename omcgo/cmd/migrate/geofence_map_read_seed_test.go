package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestGeofenceMapReadSeedPostgreSQL16(t *testing.T) {
	dsn := os.Getenv("OMCGO_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TEST_DB_DSN not set")
	}
	adminDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, adminDB.Close())
	})
	require.NoError(t, adminDB.Ping())

	databaseName := "geofence_map_seed_" +
		strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = adminDB.Exec(fmt.Sprintf(`CREATE DATABASE %q`, databaseName))
	require.NoError(t, err)

	databaseDSN := geofenceBatchMigrationDatabaseDSN(
		t,
		dsn,
		databaseName,
	)
	var db *sql.DB
	t.Cleanup(func() {
		if db != nil {
			require.NoError(t, db.Close())
		}
		_, cleanupErr := adminDB.Exec(
			fmt.Sprintf(
				`DROP DATABASE IF EXISTS %q WITH (FORCE)`,
				databaseName,
			),
		)
		require.NoError(t, cleanupErr)
	})

	db, err = sql.Open("pgx", databaseDSN)
	require.NoError(t, err)
	require.NoError(t, db.Ping())

	var serverVersion string
	require.NoError(
		t,
		db.QueryRow(`SHOW server_version_num`).Scan(&serverVersion),
	)
	require.True(
		t,
		strings.HasPrefix(serverVersion, "16"),
		"requires PostgreSQL 16, got %s",
		serverVersion,
	)

	require.NoError(
		t,
		goose.Up(db, geofenceBatchMigrationDirectory(t)),
	)
	goose.SetTableName("goose_seed_db_version")
	t.Cleanup(func() {
		goose.SetTableName("goose_db_version")
	})
	seedDirectory, err := filepath.Abs("../../migrations/seed")
	require.NoError(t, err)
	require.NoError(t, goose.Up(db, seedDirectory))

	expected := []struct {
		method string
		path   string
	}{
		{method: "GET", path: "/api/v1/geofences"},
		{method: "POST", path: "/api/v1/geofences"},
		{method: "GET", path: "/api/v1/geofences/map"},
		{method: "GET", path: "/api/v1/geofences/:id"},
		{method: "GET", path: "/api/v1/geofences/:id/versions"},
		{method: "POST", path: "/api/v1/geofences/:id/publish"},
		{method: "GET", path: "/api/v1/geofences/:id/bindings"},
	}
	const superAdministratorRoleID = "10000000-0000-0000-0000-000000000001"
	for _, endpoint := range expected {
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			var endpointCount int
			var grantCount int
			require.NoError(
				t,
				db.QueryRow(`
SELECT
    COUNT(DISTINCT endpoint.id),
    COUNT(permission.endpoint_id)
FROM public.api_endpoints endpoint
LEFT JOIN public.role_api_permissions permission
    ON permission.endpoint_id = endpoint.id
    AND permission.role_id = $1
WHERE endpoint.method = $2
  AND endpoint.path = $3`,
					superAdministratorRoleID,
					endpoint.method,
					endpoint.path,
				).Scan(&endpointCount, &grantCount),
			)
			require.Equal(t, 1, endpointCount)
			require.Equal(t, 1, grantCount)
		})
	}
}
