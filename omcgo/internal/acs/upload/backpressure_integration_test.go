//go:build integration

package upload

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/require"
)

type integrationQueueStatsSource struct {
	stats event.QueueStats
	calls int
}

func (s *integrationQueueStatsSource) LatestStats() (event.QueueStats, bool) {
	s.calls++
	return s.stats, true
}

func TestIntegrationQueueBackpressureConsumesSamplerSnapshot(t *testing.T) {
	source := &integrationQueueStatsSource{stats: event.QueueStats{
		Pending:          bpDefaultQueuePendingHigh,
		OldestPendingAge: bpDefaultQueueOldestLow,
		SampledAt:        time.Now(),
	}}
	watchdog := NewWatchdog(nil, nil, NewBackpressureMetrics(nil), nil)
	watchdog.ioPressure = nil
	watchdog.loadFn = nil
	watchdog.SetQueueStatsSource(source)

	watchdog.sample(context.Background())

	require.True(t, watchdog.active.Load())
	require.Equal(t, 1, source.calls, "one watchdog pass must consume one cached snapshot")
}

func TestIntegrationQueueBackpressureAllowsYoungSynchronizedBurst(t *testing.T) {
	source := &integrationQueueStatsSource{stats: event.QueueStats{
		Pending:          bpDefaultQueuePendingHigh,
		OldestPendingAge: bpDefaultQueueOldestLow - time.Second,
		SampledAt:        time.Now(),
	}}
	watchdog := NewWatchdog(nil, nil, NewBackpressureMetrics(nil), nil)
	watchdog.ioPressure = nil
	watchdog.loadFn = nil
	watchdog.SetQueueStatsSource(source)

	watchdog.sample(context.Background())

	require.False(t, watchdog.active.Load())
	require.Equal(t, 1, source.calls, "young burst must be decided from one cached snapshot")
}

func TestIntegrationDatabasePendingProjectionIncludesAcceptedWork(t *testing.T) {
	tsdbDSN := os.Getenv("OMCGO_DB_DSN")
	if tsdbDSN == "" {
		t.Skip("OMCGO_DB_DSN is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tsdb, err := pgxpool.New(ctx, tsdbDSN)
	require.NoError(t, err)
	defer tsdb.Close()

	fileID := uuid.New()
	_, err = tsdb.Exec(ctx, `
		INSERT INTO pm_files
		    (id,device_id,device_sn,carrier,technology,file_name,file_size,minio_path,parsed)
		VALUES ($1,$2,$3,'cmcc','lte',$4,200,$5,true)`,
		fileID, uuid.New(), "projection-"+fileID.String(), fileID.String()+".xml",
		"projection/"+fileID.String())
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = tsdb.Exec(context.Background(), `DELETE FROM pm_files WHERE id=$1`, fileID)
	})

	projected, err := NewDatabasePendingProjection(
		tsdb,
		func(context.Context) (uint64, error) { return 2, nil },
		0.5,
	)(ctx)
	require.NoError(t, err)
	require.Greater(t, projected, float64(0))
}
