//go:build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestIntegrationTSDBSchemaReconcileIsIdempotentAndPreservesVersionOneRows(t *testing.T) {
	dsn := os.Getenv("OMCGO_TSDB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TSDB_DSN not set; skipping TSDB schema reconciliation integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	schema := "slot_reconcile_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	ident := pgx.Identifier{schema}.Sanitize()
	_, err = pool.Exec(ctx, "CREATE SCHEMA "+ident)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DROP SCHEMA "+ident+" CASCADE") })

	_, err = pool.Exec(ctx, `CREATE TABLE `+ident+`.pm_files (
		id uuid PRIMARY KEY,
		device_id uuid NOT NULL,
		technology varchar(16) NOT NULL,
		carrier varchar(16) NOT NULL,
		parsed boolean NOT NULL DEFAULT false
	)`)
	require.NoError(t, err)
	fileID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO `+ident+`.pm_files
		(id,device_id,technology,carrier,parsed) VALUES ($1,$2,'lte','blq',true)`, fileID, uuid.New())
	require.NoError(t, err)

	sqlPath := filepath.Join("..", "..", "..", "deployments", "release", "bundle", "deploy", "tsdb-schema-reconcile.sql")
	reconcileSQL, err := os.ReadFile(sqlPath)
	require.NoError(t, err)
	scopedSQL := strings.ReplaceAll(string(reconcileSQL), "public.", ident+".")

	_, err = pool.Exec(ctx, scopedSQL)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, scopedSQL)
	require.NoError(t, err, "reconciliation must be idempotent")

	var measurementStart, measurementEnd *time.Time
	err = pool.QueryRow(ctx, `SELECT measurement_start,measurement_end FROM `+ident+`.pm_files WHERE id=$1`, fileID).
		Scan(&measurementStart, &measurementEnd)
	require.NoError(t, err)
	require.Nil(t, measurementStart)
	require.Nil(t, measurementEnd)

	var tableExists, fileIndexExists, latestIndexExists, statusConstraintExists bool
	err = pool.QueryRow(ctx, `SELECT
		to_regclass($1) IS NOT NULL,
		to_regclass($2) IS NOT NULL,
		to_regclass($3) IS NOT NULL,
		EXISTS (
			SELECT 1 FROM pg_constraint c
			JOIN pg_namespace n ON n.oid=c.connamespace
			WHERE n.nspname=$4 AND c.conname='chk_pm_slot_health_status'
		)`,
		schema+".pm_slot_health",
		schema+".idx_pm_files_measurement_slot",
		schema+".idx_pm_slot_health_latest",
		schema,
	).Scan(&tableExists, &fileIndexExists, &latestIndexExists, &statusConstraintExists)
	require.NoError(t, err)
	require.True(t, tableExists)
	require.True(t, fileIndexExists)
	require.True(t, latestIndexExists)
	require.True(t, statusConstraintExists)
}
