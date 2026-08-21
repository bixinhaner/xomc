package paramsync

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingFullRunProjection struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
}

func (p *blockingFullRunProjection) Refresh(context.Context, uuid.UUID) error {
	p.calls.Add(1)
	select {
	case p.started <- struct{}{}:
	default:
	}
	<-p.release
	return nil
}

type recordingFullRunProjection struct {
	failDevice uuid.UUID
	seen       []uuid.UUID
}

func (p *recordingFullRunProjection) Refresh(_ context.Context, deviceID uuid.UUID) error {
	p.seen = append(p.seen, deviceID)
	if deviceID == p.failDevice {
		return fmt.Errorf("permanent projection failure")
	}
	return nil
}

func insertSucceededFullRunForProjectionTest(
	t *testing.T,
	pool *pgxpool.Pool,
	completedAt time.Time,
	projectionNextAttemptAt time.Time,
) (uuid.UUID, uuid.UUID) {
	t.Helper()
	req := insertParamSyncRequestForTest(t, pool, RequestStatusSucceeded)
	runID := uuid.New()
	_, err := pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status, completed_at,
 projection_status, projection_next_attempt_at)
VALUES ($1, $2, $3, $4, 'manual', 'full', 'succeeded', $5, 'pending', $6)`,
		runID, req.ID, req.DeviceID, req.DeviceSN, completedAt, projectionNextAttemptAt)
	require.NoError(t, err)
	return runID, req.DeviceID
}

func TestCompletionProjectorContinuesAfterOneRunFails(t *testing.T) {
	pool := newParamSyncTestPool(t)
	testNow := time.Now().UTC()
	projectionDueAt := testNow.Add(time.Hour)
	_, firstDevice := insertSucceededFullRunForProjectionTest(t, pool, testNow.Add(-time.Minute), projectionDueAt)
	secondRun, secondDevice := insertSucceededFullRunForProjectionTest(t, pool, testNow, projectionDueAt)
	projection := &recordingFullRunProjection{failDevice: firstDevice}
	projector := NewCompletionProjector(pool, nil, projection)
	projector.now = func() time.Time { return projectionDueAt.Add(time.Second) }

	completed, err := projector.ReconcilePending(context.Background(), 100)

	require.Error(t, err)
	assert.Equal(t, 1, completed)
	assert.Equal(t, []uuid.UUID{firstDevice, secondDevice}, projection.seen)
	var status string
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT projection_status FROM parameter_sync_runs WHERE id=$1`, secondRun).Scan(&status))
	assert.Equal(t, "completed", status)

	completed, err = projector.ReconcilePending(context.Background(), 100)
	require.NoError(t, err)
	assert.Zero(t, completed)
	assert.Equal(t, []uuid.UUID{firstDevice, secondDevice}, projection.seen, "failed projection must respect retry backoff")
}

func TestCompletionProjectorClaimsRunOnlyOnceAcrossConcurrentWorkers(t *testing.T) {
	pool := newParamSyncTestPool(t)
	runID, _ := insertSucceededFullRunForProjectionTest(t, pool, time.Now(), time.Now().Add(-time.Minute))
	projection := &blockingFullRunProjection{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	first := NewCompletionProjector(pool, nil, projection)
	second := NewCompletionProjector(pool, nil, projection)
	results := make(chan error, 2)

	go func() { results <- first.projectRun(context.Background(), runID) }()
	select {
	case <-projection.started:
	case <-time.After(2 * time.Second):
		t.Fatal("first projection did not start")
	}
	go func() { results <- second.projectRun(context.Background(), runID) }()
	time.Sleep(100 * time.Millisecond)
	close(projection.release)
	require.NoError(t, <-results)
	require.NoError(t, <-results)
	assert.Equal(t, int32(1), projection.calls.Load())
}

func TestCompletionProjectorRecoversMissingLegacyLeaseWithoutStealingLiveLease(t *testing.T) {
	pool := newParamSyncTestPool(t)
	projectionDueAt := time.Now().Add(time.Hour)
	expiredRun, expiredDevice := insertSucceededFullRunForProjectionTest(t, pool, time.Now().Add(-time.Minute), projectionDueAt)
	liveRun, _ := insertSucceededFullRunForProjectionTest(t, pool, time.Now(), projectionDueAt)
	_, err := pool.Exec(context.Background(), `UPDATE parameter_sync_runs SET
projection_status='processing', projection_lease_token=$2, projection_lease_until=$3
WHERE id=$1`, expiredRun, uuid.New(), nil)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE parameter_sync_runs SET
projection_status='processing', projection_lease_token=$2, projection_lease_until=$3
WHERE id=$1`, liveRun, uuid.New(), time.Now().Add(time.Minute))
	require.NoError(t, err)
	projection := &recordingFullRunProjection{}
	projector := NewCompletionProjector(pool, nil, projection)

	completed, err := projector.ReconcilePending(context.Background(), 100)

	require.NoError(t, err)
	assert.Equal(t, 1, completed)
	assert.Equal(t, []uuid.UUID{expiredDevice}, projection.seen)
}
