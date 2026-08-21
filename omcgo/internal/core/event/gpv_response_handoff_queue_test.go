package event

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/storage"
)

type recordingGPVHandoffRepo struct {
	enqueued []GPVHandoffEnqueue
	err      error
}

func (r *recordingGPVHandoffRepo) Enqueue(_ context.Context, req GPVHandoffEnqueue) error {
	r.enqueued = append(r.enqueued, req)
	return r.err
}

func (r *recordingGPVHandoffRepo) ClaimDue(context.Context, GPVHandoffClaimOptions) ([]GPVHandoffEntry, error) {
	return nil, nil
}

func (r *recordingGPVHandoffRepo) MarkDelivered(context.Context, uuid.UUID, uuid.UUID, time.Time) (bool, error) {
	return true, nil
}

func (r *recordingGPVHandoffRepo) MarkFailed(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (bool, error) {
	return true, nil
}

func (r *recordingGPVHandoffRepo) MarkDead(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (bool, error) {
	return true, nil
}

func TestGPVHandoffEnqueueHandlerPersistsEvent(t *testing.T) {
	repo := &recordingGPVHandoffRepo{}
	handler := GPVHandoffEnqueueHandler(repo, "device-rpc-gpv", func(evt Event) (string, error) {
		var payload struct {
			DeviceSN string `json:"device_sn"`
		}
		require.NoError(t, evt.DecodePayload(&payload))
		return payload.DeviceSN, nil
	})

	evt, err := NewEvent(SubjectCommandGetParamsResponse, map[string]any{"device_sn": "SN-1"})
	require.NoError(t, err)
	require.NoError(t, handler(context.Background(), evt))

	require.Len(t, repo.enqueued, 1)
	require.Equal(t, "device-rpc-gpv", repo.enqueued[0].Durable)
	require.Equal(t, "SN-1", repo.enqueued[0].DeviceSN)
	require.Equal(t, evt.ID, repo.enqueued[0].Event.ID)
}

func TestGPVHandoffEnqueueHandlerReturnsPersistenceError(t *testing.T) {
	persistErr := errors.New("postgres unavailable")
	repo := &recordingGPVHandoffRepo{err: persistErr}
	handler := GPVHandoffEnqueueHandler(repo, "device-rpc-gpv", func(Event) (string, error) {
		return "SN-1", nil
	})

	evt, err := NewEvent(SubjectCommandGetParamsResponse, json.RawMessage(`{"device_sn":"SN-1"}`))
	require.NoError(t, err)
	require.ErrorIs(t, handler(context.Background(), evt), persistErr)
}

func TestGPVHandoffEnqueueHandlerRejectsMissingDeviceSN(t *testing.T) {
	repo := &recordingGPVHandoffRepo{}
	handler := GPVHandoffEnqueueHandler(repo, "device-rpc-gpv", func(Event) (string, error) {
		return "", nil
	})

	evt, err := NewEvent(SubjectCommandGetParamsResponse, json.RawMessage(`{"device_sn":""}`))
	require.NoError(t, err)
	require.ErrorContains(t, handler(context.Background(), evt), "device SN is required")
	require.Empty(t, repo.enqueued)
}

func TestRunGPVHandoffWorkersProcessesOtherDevicesWhileSlowHeadIsBlocked(t *testing.T) {
	repo := newMemoryGPVHandoffRepo()
	const durable = "device-rpc-gpv"
	for _, deviceSN := range []string{"SN-SLOW-HEAD", "SN-FAST-1", "SN-FAST-2"} {
		evt, err := NewEvent(SubjectCommandGetParamsResponse, map[string]any{"device_sn": deviceSN})
		require.NoError(t, err)
		require.NoError(t, repo.Enqueue(context.Background(), GPVHandoffEnqueue{
			Durable:  durable,
			DeviceSN: deviceSN,
			Event:    evt,
		}))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	slowStarted := make(chan struct{})
	releaseSlow := make(chan struct{})
	fastDone := make(chan string, 2)
	var slowOnce sync.Once
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(releaseSlow) })
	}
	defer release()

	require.NoError(t, RunGPVHandoffWorkers(ctx, repo, GPVHandoffWorkerConfig{
		Durable:     durable,
		Concurrency: 1,
		BatchSize:   3,
		Lease:       time.Second,
		MaxAttempts: 3,
		PollIdle:    time.Millisecond,
	}, func(ctx context.Context, evt Event) error {
		deviceSN, err := testHandoffDeviceKey(evt)
		if err != nil {
			return err
		}
		if deviceSN == "SN-SLOW-HEAD" {
			slowOnce.Do(func() { close(slowStarted) })
			select {
			case <-releaseSlow:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		}
		fastDone <- deviceSN
		return nil
	}, zap.NewNop()))

	require.Eventually(t, func() bool {
		select {
		case <-slowStarted:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)

	completedFast := map[string]bool{}
	for len(completedFast) < 2 {
		select {
		case deviceSN := <-fastDone:
			completedFast[deviceSN] = true
		case <-time.After(time.Second):
			t.Fatal("fast device handoff entries were blocked by the slow device")
		}
	}
	require.True(t, completedFast["SN-FAST-1"])
	require.True(t, completedFast["SN-FAST-2"])
	require.Equal(t, 2, repo.CountStatus(GPVHandoffStatusDelivered))

	release()
	require.Eventually(t, func() bool {
		return repo.CountStatus(GPVHandoffStatusDelivered) == 3
	}, time.Second, time.Millisecond)
}

func TestGPVHandoffRepositoryReclaimsExpiredProcessingAtAttemptLimit(t *testing.T) {
	repo := newMemoryGPVHandoffRepo()
	const durable = "device-rpc-gpv"
	evt, err := NewEvent(SubjectCommandGetParamsResponse, map[string]any{"device_sn": "SN-RESTART"})
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, repo.Enqueue(context.Background(), GPVHandoffEnqueue{
		Durable:  durable,
		DeviceSN: "SN-RESTART",
		Event:    evt,
	}))

	first, err := repo.ClaimDue(context.Background(), GPVHandoffClaimOptions{
		Durable:     durable,
		Now:         now.Add(time.Millisecond),
		Lease:       time.Second,
		Limit:       1,
		MaxAttempts: 1,
	})
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.Equal(t, 1, first[0].AttemptCount)

	reclaimed, err := repo.ClaimDue(context.Background(), GPVHandoffClaimOptions{
		Durable:     durable,
		Now:         now.Add(2*time.Second + time.Millisecond),
		Lease:       time.Second,
		Limit:       1,
		MaxAttempts: 1,
	})
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "最后一次尝试时崩溃留下的 processing 行，租约过期后必须可恢复认领")
	require.Equal(t, first[0].EventID, reclaimed[0].EventID)
	require.Equal(t, 2, reclaimed[0].AttemptCount)

	ok, err := repo.MarkDead(context.Background(), reclaimed[0].ID, reclaimed[0].LeaseToken, "still failing", now.Add(2*time.Second))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, repo.CountStatus(GPVHandoffStatusDead))
}

func TestPGGPVHandoffRepositoryClaimPreservesPerDeviceOrderingIntegration(t *testing.T) {
	pool := newGPVHandoffTestPool(t)
	repo := NewPGGPVHandoffRepository(storage.NewPoolDB(pool))
	ctx := context.Background()
	prefix := "test-gpv-handoff-" + uuid.NewString()
	durable := "device-rpc-gpv-" + uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM command_gpv_response_handoffs WHERE event_id LIKE $1",
			prefix+"%")
	})

	a1 := prefix + "-a1"
	a2 := prefix + "-a2"
	b1 := prefix + "-b1"
	enqueueGPVHandoffForTest(t, ctx, repo, durable, a1, "SN-A", now.Add(-3*time.Second))
	enqueueGPVHandoffForTest(t, ctx, repo, durable, a2, "SN-A", now.Add(-2*time.Second))
	enqueueGPVHandoffForTest(t, ctx, repo, durable, b1, "SN-B", now.Add(-1*time.Second))

	first := claimGPVHandoffForTest(t, ctx, repo, durable, now, 1)
	require.Len(t, first, 1)
	require.Equal(t, a1, first[0].EventID)

	second := claimGPVHandoffForTest(t, ctx, repo, durable, now, 10)
	require.Len(t, second, 1)
	require.Equal(t, b1, second[0].EventID, "SN-A 的慢头处理中时，SN-B 仍应可推进")

	ok, err := repo.MarkFailed(ctx, first[0].ID, first[0].LeaseToken, "still slow", now.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.MarkDelivered(ctx, second[0].ID, second[0].LeaseToken, now)
	require.NoError(t, err)
	require.True(t, ok)

	blocked := claimGPVHandoffForTest(t, ctx, repo, durable, now.Add(30*time.Second), 10)
	require.Empty(t, blocked, "SN-A 早到失败消息未到重试时间前，不应跳过它去处理 SN-A 后续")

	retryHead := claimGPVHandoffForTest(t, ctx, repo, durable, now.Add(2*time.Minute), 10)
	require.Len(t, retryHead, 1)
	require.Equal(t, a1, retryHead[0].EventID, "重试时仍应先处理同设备早到消息")
	ok, err = repo.MarkDelivered(ctx, retryHead[0].ID, retryHead[0].LeaseToken, now.Add(2*time.Minute))
	require.NoError(t, err)
	require.True(t, ok)

	nextSameDevice := claimGPVHandoffForTest(t, ctx, repo, durable, now.Add(3*time.Minute), 10)
	require.Len(t, nextSameDevice, 1)
	require.Equal(t, a2, nextSameDevice[0].EventID)
}

func TestPGGPVHandoffRepositoryClaimReclaimsExpiredProcessingAtAttemptLimitIntegration(t *testing.T) {
	pool := newGPVHandoffTestPool(t)
	repo := NewPGGPVHandoffRepository(storage.NewPoolDB(pool))
	ctx := context.Background()
	prefix := "test-reclaim-" + uuid.NewString()
	durable := "device-rpc-gpv-" + uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM command_gpv_response_handoffs WHERE event_id LIKE $1",
			prefix+"%")
	})

	eventID := prefix + "-event"
	enqueueGPVHandoffForTest(t, ctx, repo, durable, eventID, "SN-RESTART", now)
	first := claimGPVHandoffForTest(t, ctx, repo, durable, now, 1, 1)
	require.Len(t, first, 1)
	require.Equal(t, 1, first[0].AttemptCount)

	reclaimed := claimGPVHandoffForTest(t, ctx, repo, durable, now.Add(2*time.Minute), 1, 1)
	require.Len(t, reclaimed, 1, "最后一次尝试时崩溃留下的 processing 行，租约过期后必须可恢复认领")
	require.Equal(t, eventID, reclaimed[0].EventID)
	require.Equal(t, 2, reclaimed[0].AttemptCount)
	ok, err := repo.MarkDead(ctx, reclaimed[0].ID, reclaimed[0].LeaseToken, "still failing", now.Add(2*time.Minute))
	require.NoError(t, err)
	require.True(t, ok)
}

func newGPVHandoffTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("no PostgreSQL available: %v", err)
	}
	t.Cleanup(pool.Close)
	var exists bool
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT to_regclass('public.command_gpv_response_handoffs') IS NOT NULL").Scan(&exists))
	if !exists {
		t.Skip("command_gpv_response_handoffs migration is not applied")
	}
	return pool
}

func enqueueGPVHandoffForTest(
	t *testing.T,
	ctx context.Context,
	repo *PGGPVHandoffRepository,
	durable string,
	eventID string,
	deviceSN string,
	createdAt time.Time,
) {
	t.Helper()
	evt := Event{
		ID:        eventID,
		Subject:   SubjectCommandGetParamsResponse,
		Payload:   json.RawMessage(`{"device_sn":"` + deviceSN + `"}`),
		Metadata:  map[string]string{MetadataDeviceSN: deviceSN},
		Timestamp: createdAt,
	}
	require.NoError(t, repo.Enqueue(ctx, GPVHandoffEnqueue{
		Durable:  durable,
		DeviceSN: deviceSN,
		Event:    evt,
	}))
	_, err := repo.db.Exec(ctx,
		`UPDATE command_gpv_response_handoffs
		    SET created_at = $1, updated_at = $1, next_attempt_at = $1
		  WHERE durable = $2 AND event_id = $3`,
		createdAt, durable, eventID)
	require.NoError(t, err)
}

func claimGPVHandoffForTest(
	t *testing.T,
	ctx context.Context,
	repo *PGGPVHandoffRepository,
	durable string,
	now time.Time,
	limit int,
	maxAttempts ...int,
) []GPVHandoffEntry {
	t.Helper()
	attempts := 3
	if len(maxAttempts) > 0 {
		attempts = maxAttempts[0]
	}
	entries, err := repo.ClaimDue(ctx, GPVHandoffClaimOptions{
		Durable:     durable,
		Now:         now,
		Lease:       time.Minute,
		Limit:       limit,
		MaxAttempts: attempts,
	})
	require.NoError(t, err)
	return entries
}

type memoryGPVHandoffRepo struct {
	mu   sync.Mutex
	rows []memoryGPVHandoffRow
}

type memoryGPVHandoffRow struct {
	entry         GPVHandoffEntry
	status        string
	nextAttemptAt time.Time
}

func newMemoryGPVHandoffRepo() *memoryGPVHandoffRepo {
	return &memoryGPVHandoffRepo{}
}

func (r *memoryGPVHandoffRepo) Enqueue(_ context.Context, req GPVHandoffEnqueue) error {
	if err := validateGPVHandoffEnqueue(req); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.entry.Durable == req.Durable && row.entry.EventID == req.Event.ID {
			return nil
		}
	}
	createdAt := time.Now().UTC().Add(time.Duration(len(r.rows)) * time.Nanosecond)
	r.rows = append(r.rows, memoryGPVHandoffRow{
		entry: GPVHandoffEntry{
			ID:        uuid.New(),
			Durable:   req.Durable,
			EventID:   req.Event.ID,
			DeviceSN:  req.DeviceSN,
			Subject:   req.Event.Subject,
			Payload:   append(json.RawMessage(nil), req.Event.Payload...),
			Metadata:  req.Event.Metadata,
			Timestamp: req.Event.Timestamp,
			CreatedAt: createdAt,
		},
		status:        GPVHandoffStatusPending,
		nextAttemptAt: createdAt,
	})
	return nil
}

func (r *memoryGPVHandoffRepo) ClaimDue(_ context.Context, options GPVHandoffClaimOptions) ([]GPVHandoffEntry, error) {
	if err := validateGPVHandoffClaimOptions(options); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	leaseToken := uuid.New()
	leaseUntil := options.Now.Add(options.Lease)
	claimed := make([]GPVHandoffEntry, 0, options.Limit)
	for i := range r.rows {
		if len(claimed) >= options.Limit {
			break
		}
		row := &r.rows[i]
		if row.entry.Durable != options.Durable {
			continue
		}
		if row.status != GPVHandoffStatusProcessing && row.entry.AttemptCount >= options.MaxAttempts {
			continue
		}
		if !row.claimable(options.Now) || r.hasEarlierOpenLocked(i, row.entry.Durable, row.entry.DeviceSN) {
			continue
		}
		row.status = GPVHandoffStatusProcessing
		row.entry.AttemptCount++
		row.entry.LeaseToken = leaseToken
		row.entry.LeaseUntil = leaseUntil
		claimed = append(claimed, row.entry)
	}
	return claimed, nil
}

func (r *memoryGPVHandoffRepo) MarkDelivered(_ context.Context, id uuid.UUID, leaseToken uuid.UUID, _ time.Time) (bool, error) {
	return r.mark(id, leaseToken, GPVHandoffStatusDelivered, time.Time{})
}

func (r *memoryGPVHandoffRepo) MarkFailed(_ context.Context, id uuid.UUID, leaseToken uuid.UUID, _ string, nextAttemptAt time.Time) (bool, error) {
	return r.mark(id, leaseToken, GPVHandoffStatusFailed, nextAttemptAt)
}

func (r *memoryGPVHandoffRepo) MarkDead(_ context.Context, id uuid.UUID, leaseToken uuid.UUID, _ string, _ time.Time) (bool, error) {
	return r.mark(id, leaseToken, GPVHandoffStatusDead, time.Time{})
}

func (r *memoryGPVHandoffRepo) CountStatus(status string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, row := range r.rows {
		if row.status == status {
			count++
		}
	}
	return count
}

func (r *memoryGPVHandoffRepo) hasEarlierOpenLocked(current int, durable, deviceSN string) bool {
	for i := 0; i < current; i++ {
		row := r.rows[i]
		if row.entry.Durable == durable && row.entry.DeviceSN == deviceSN && isOpenGPVHandoffStatus(row.status) {
			return true
		}
	}
	return false
}

func (r *memoryGPVHandoffRepo) mark(id uuid.UUID, leaseToken uuid.UUID, status string, nextAttemptAt time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.rows {
		row := &r.rows[i]
		if row.entry.ID != id || row.entry.LeaseToken != leaseToken || row.status != GPVHandoffStatusProcessing {
			continue
		}
		row.status = status
		row.entry.LeaseToken = uuid.Nil
		row.entry.LeaseUntil = time.Time{}
		row.nextAttemptAt = nextAttemptAt
		return true, nil
	}
	return false, nil
}

func (r memoryGPVHandoffRow) claimable(now time.Time) bool {
	switch r.status {
	case GPVHandoffStatusPending, GPVHandoffStatusFailed:
		return !r.nextAttemptAt.After(now)
	case GPVHandoffStatusProcessing:
		return !r.entry.LeaseUntil.After(now)
	default:
		return false
	}
}

func isOpenGPVHandoffStatus(status string) bool {
	return status == GPVHandoffStatusPending ||
		status == GPVHandoffStatusFailed ||
		status == GPVHandoffStatusProcessing
}

func testHandoffDeviceKey(evt Event) (string, error) {
	var payload struct {
		DeviceSN string `json:"device_sn"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return "", err
	}
	return payload.DeviceSN, nil
}
