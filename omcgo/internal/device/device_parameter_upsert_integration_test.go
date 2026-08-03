package device

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func newDeviceParameterTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		dsn = "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no PostgreSQL available: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("no PostgreSQL available: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestBulkUpsertCommitsUncontendedDeviceBeforeWaitingForContendedDevice(t *testing.T) {
	ctx := context.Background()
	pool := newDeviceParameterTestPool(t)
	firstID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	blockedID := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	for _, id := range []uuid.UUID{firstID, blockedID} {
		_, err := pool.Exec(ctx, `
			INSERT INTO devices (
				id, serial_number, oui, carrier, technology, lifecycle_state, is_online
			) VALUES ($1, $2, 'AABBCC', 'cmcc', 'LTE', 'commissioned', true)
		`, id, "TEST-PARAM-WRITE-"+id.String())
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM devices WHERE id=ANY($1::uuid[])`, []uuid.UUID{firstID, blockedID})
	})

	blocker, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = blocker.Rollback(context.Background()) }()
	require.NoError(t, AcquireParameterWriteLocks(ctx, blocker, blockedID))

	rows := []deviceParameterUpsertRow{
		{deviceID: firstID, parameter: model.DeviceParameter{ParameterPath: "Device.Info.Value", ParameterValue: "first"}},
		{deviceID: blockedID, parameter: model.DeviceParameter{ParameterPath: "Device.Info.Value", ParameterValue: "blocked"}},
	}
	resultCh := make(chan error, 1)
	go func() {
		writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, writeErr := bulkUpsertDeviceParameters(writeCtx, pool, rows)
		resultCh <- writeErr
	}()

	require.Eventually(t, func() bool {
		var value string
		return pool.QueryRow(ctx, `SELECT parameter_value FROM device_parameters WHERE device_id=$1 AND parameter_path=$2`, firstID, "Device.Info.Value").Scan(&value) == nil && value == "first"
	}, time.Second, 20*time.Millisecond,
		"an unrelated device must commit while the contended device waits")

	require.NoError(t, blocker.Rollback(ctx))
	require.NoError(t, <-resultCh)
	var blockedValue string
	require.NoError(t, pool.QueryRow(ctx, `SELECT parameter_value FROM device_parameters WHERE device_id=$1 AND parameter_path=$2`, blockedID, "Device.Info.Value").Scan(&blockedValue))
	require.Equal(t, "blocked", blockedValue)
}
