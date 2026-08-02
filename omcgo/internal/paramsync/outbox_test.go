package paramsync

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/task"
)

func TestRunConcurrentOutboxBatchesRunsBoundedWorkers(t *testing.T) {
	var calls atomic.Int32
	var inFlight atomic.Int32
	var maxFlight atomic.Int32

	delivered, err := runConcurrentOutboxBatches(8, func() (int, error) {
		calls.Add(1)
		current := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			max := maxFlight.Load()
			if current <= max || maxFlight.CompareAndSwap(max, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		return 1, nil
	})

	if err != nil {
		t.Fatalf("dispatch batches: %v", err)
	}
	if delivered != 8 || calls.Load() != 8 {
		t.Fatalf("delivered=%d calls=%d, want 8/8", delivered, calls.Load())
	}
	if maxFlight.Load() <= 1 {
		t.Fatalf("max concurrent workers=%d, want >1", maxFlight.Load())
	}
}

type concurrentWakeLifecycle struct {
	releases atomic.Int32
	wakes    atomic.Int32
}

func (l *concurrentWakeLifecycle) ReleasePlannedTask(context.Context, *task.Task) (bool, error) {
	return true, nil
}

func (l *concurrentWakeLifecycle) ReleasePlannedTaskWithoutWake(context.Context, *task.Task) (bool, error) {
	l.releases.Add(1)
	return true, nil
}

func (l *concurrentWakeLifecycle) WakePlannedDevice(string) {
	l.wakes.Add(1)
}

func (l *concurrentWakeLifecycle) EvictPlannedTask(context.Context, *task.Task) error {
	return nil
}

func TestPlannedTaskReleaseBatchCoalescesConcurrentWakeByDevice(t *testing.T) {
	lifecycle := &concurrentWakeLifecycle{}
	batch := newPlannedTaskReleaseBatch(lifecycle)
	var wg sync.WaitGroup

	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := batch.Release(context.Background(), &task.Task{DeviceSN: "same-device"}); err != nil {
				t.Errorf("release: %v", err)
			}
		}()
	}
	wg.Wait()
	batch.WakeDevices()

	if lifecycle.releases.Load() != 32 {
		t.Fatalf("releases=%d, want 32", lifecycle.releases.Load())
	}
	if lifecycle.wakes.Load() != 1 {
		t.Fatalf("wakes=%d, want one coalesced wake", lifecycle.wakes.Load())
	}
}

func TestBuildClaimOutboxSQLKeepsPartialIndexPredicateLiteral(t *testing.T) {
	now := time.Date(2026, 8, 1, 2, 30, 0, 0, time.UTC)

	query, args, err := buildClaimOutboxSQL(now, 100)

	if err != nil {
		t.Fatalf("build claim SQL: %v", err)
	}
	if want := "status IN ('pending', 'failed')"; !strings.Contains(query, want) {
		t.Fatalf("query = %q, want literal partial-index predicate %q", query, want)
	}
	if strings.Contains(query, "status IN ($1,$2)") {
		t.Fatalf("query = %q, status values must not be parameters", query)
	}
	if len(args) != 1 || args[0] != now {
		t.Fatalf("args = %#v, want only next_attempt_at=%v", args, now)
	}
}

func TestBuildRequeueStaleOutboxSQLKeepsPartialIndexPredicateLiteral(t *testing.T) {
	now := time.Date(2026, 8, 1, 2, 30, 0, 0, time.UTC)
	staleBefore := now.Add(-time.Minute)

	query, args, err := buildRequeueStaleOutboxSQL(now, staleBefore)

	if err != nil {
		t.Fatalf("build requeue stale SQL: %v", err)
	}
	if want := "status = 'delivering'"; !strings.Contains(query, want) {
		t.Fatalf("query = %q, want literal partial-index predicate %q", query, want)
	}
	if strings.Contains(query, "status = $") {
		t.Fatalf("query = %q, delivering status must not be parameterized", query)
	}
	if len(args) != 2 || args[0] != now || args[1] != staleBefore {
		t.Fatalf("args = %#v, want now/staleBefore", args)
	}
}

func TestParameterSyncOutboxRuntimeIndexesCoverRecoveryAndObservation(t *testing.T) {
	if !strings.Contains(parameterSyncOutboxDeliveringIndexSQL, "CREATE INDEX CONCURRENTLY") ||
		!strings.Contains(parameterSyncOutboxDeliveringIndexSQL, "updated_at") ||
		!strings.Contains(parameterSyncOutboxDeliveringIndexSQL, "WHERE status = 'delivering'") {
		t.Fatalf("delivering index does not cover stale recovery: %s", parameterSyncOutboxDeliveringIndexSQL)
	}
	if !strings.Contains(parameterSyncOutboxStatusIndexSQL, "CREATE INDEX CONCURRENTLY") ||
		!strings.Contains(parameterSyncOutboxStatusIndexSQL, "(status)") ||
		!strings.Contains(parameterSyncOutboxStatusIndexSQL, "WHERE status IN ('delivered', 'dead')") {
		t.Fatalf("status index does not cover queue observation: %s", parameterSyncOutboxStatusIndexSQL)
	}
	if !strings.Contains(parameterSyncActiveRunConvergenceIndexSQL, "CREATE INDEX CONCURRENTLY") ||
		!strings.Contains(parameterSyncActiveRunConvergenceIndexSQL, "(started_at, id)") ||
		!strings.Contains(parameterSyncActiveRunConvergenceIndexSQL, "planning") ||
		!strings.Contains(parameterSyncActiveRunConvergenceIndexSQL, "cancelling") {
		t.Fatalf("active run index does not cover bounded convergence: %s", parameterSyncActiveRunConvergenceIndexSQL)
	}
}

func TestParameterSyncOutboxRuntimeIndexMaintenanceUsesSessionAdvisoryLock(t *testing.T) {
	if !strings.Contains(parameterSyncOutboxIndexLockSQL, "pg_advisory_lock") {
		t.Fatalf("lock SQL must acquire a session advisory lock: %s", parameterSyncOutboxIndexLockSQL)
	}
	if !strings.Contains(parameterSyncOutboxIndexUnlockSQL, "pg_advisory_unlock") {
		t.Fatalf("unlock SQL must release the session advisory lock: %s", parameterSyncOutboxIndexUnlockSQL)
	}
	if strings.Replace(parameterSyncOutboxIndexLockSQL, "pg_advisory_lock", "pg_advisory_unlock", 1) != parameterSyncOutboxIndexUnlockSQL {
		t.Fatalf("lock and unlock must use the same advisory key: lock=%q unlock=%q", parameterSyncOutboxIndexLockSQL, parameterSyncOutboxIndexUnlockSQL)
	}
}

type recordingPlannedTaskLifecycle struct {
	released         int
	releasedNoWake   int
	wokenDeviceCount map[string]int
}

func (r *recordingPlannedTaskLifecycle) ReleasePlannedTask(_ context.Context, _ *task.Task) (bool, error) {
	r.released++
	return true, nil
}

func (r *recordingPlannedTaskLifecycle) ReleasePlannedTaskWithoutWake(_ context.Context, _ *task.Task) (bool, error) {
	r.releasedNoWake++
	return true, nil
}

func (r *recordingPlannedTaskLifecycle) WakePlannedDevice(deviceSN string) {
	if r.wokenDeviceCount == nil {
		r.wokenDeviceCount = make(map[string]int)
	}
	r.wokenDeviceCount[deviceSN]++
}

func (r *recordingPlannedTaskLifecycle) EvictPlannedTask(_ context.Context, _ *task.Task) error {
	return nil
}

func TestPlannedTaskReleaseBatchCoalescesWakeByDevice(t *testing.T) {
	lifecycle := &recordingPlannedTaskLifecycle{}
	batch := newPlannedTaskReleaseBatch(lifecycle)
	ctx := context.Background()

	for i := 0; i < 13; i++ {
		if _, err := batch.Release(ctx, &task.Task{DeviceSN: "device-a"}); err != nil {
			t.Fatalf("release device-a task %d: %v", i, err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := batch.Release(ctx, &task.Task{DeviceSN: "device-b"}); err != nil {
			t.Fatalf("release device-b task %d: %v", i, err)
		}
	}
	batch.WakeDevices()

	if lifecycle.released != 0 {
		t.Fatalf("ordinary release calls = %d, want 0", lifecycle.released)
	}
	if lifecycle.releasedNoWake != 15 {
		t.Fatalf("release-without-wake calls = %d, want 15", lifecycle.releasedNoWake)
	}
	if lifecycle.wokenDeviceCount["device-a"] != 1 || lifecycle.wokenDeviceCount["device-b"] != 1 {
		t.Fatalf("wake counts = %#v, want one per device", lifecycle.wokenDeviceCount)
	}
}

type legacyPlannedTaskLifecycle struct {
	released int
}

func (r *legacyPlannedTaskLifecycle) ReleasePlannedTask(_ context.Context, _ *task.Task) (bool, error) {
	r.released++
	return true, nil
}

func (r *legacyPlannedTaskLifecycle) EvictPlannedTask(_ context.Context, _ *task.Task) error {
	return nil
}

func TestPlannedTaskReleaseBatchFallsBackToOrdinaryLifecycle(t *testing.T) {
	lifecycle := &legacyPlannedTaskLifecycle{}
	batch := newPlannedTaskReleaseBatch(lifecycle)

	for i := 0; i < 3; i++ {
		if _, err := batch.Release(context.Background(), &task.Task{DeviceSN: "device-a"}); err != nil {
			t.Fatalf("release task %d: %v", i, err)
		}
	}
	batch.WakeDevices()

	if lifecycle.released != 3 {
		t.Fatalf("ordinary release calls = %d, want 3", lifecycle.released)
	}
}
