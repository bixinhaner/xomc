package stream

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap"
)

func TestFinalizeSchedulerUsesConfiguredFinalizerConcurrency(t *testing.T) {
	finalizer := NewFinalizer(nil, nil, nil).SetConcurrency(7)
	if got := finalizer.Concurrency(); got != 7 {
		t.Fatalf("finalizer concurrency = %d, want 7 fixed scheduler workers", got)
	}
}

func TestFinalizeSchedulerGivesNewHourShareWhileOldHourIsBacklogged(t *testing.T) {
	selector := newFinalizeClaimSelector()
	oldest := time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC)
	newest := oldest.Add(time.Hour)
	backlog := make([]time.Time, 0, 5002)
	for range 5000 {
		backlog = append(backlog, oldest)
	}
	backlog = append(backlog, newest, newest)

	claimed := make(map[time.Time]int)
	for range 4 {
		order := selector.nextOrder(finalizeHourlyDevice)
		index := 0
		if order == claimNewestFirst {
			index = len(backlog) - 1
		}
		claimed[backlog[index]]++
		backlog = append(backlog[:index], backlog[index+1:]...)
	}

	if claimed[oldest] == 0 || claimed[newest] == 0 {
		t.Fatalf("claimed hours = %v, both old and newly due hours need scheduling share", claimed)
	}
}

func TestFinalizeSchedulerAlternatingEndsEventuallyClaimsMiddleWindow(t *testing.T) {
	selector := newFinalizeClaimSelector()
	start := time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC)
	type hourBacklog struct {
		end       time.Time
		remaining int
	}
	backlog := []hourBacklog{
		{end: start, remaining: 5000},
		{end: start.Add(time.Hour), remaining: 5000},
		{end: start.Add(2 * time.Hour), remaining: 5000},
	}
	middle := start.Add(time.Hour)
	middleClaimRound := 0
	for round := 1; round <= 10001; round++ {
		index := 0
		if selector.nextOrder(finalizeHourlyDevice) == claimNewestFirst {
			index = len(backlog) - 1
		}
		if backlog[index].end == middle {
			middleClaimRound = round
			break
		}
		backlog[index].remaining--
		if backlog[index].remaining == 0 {
			backlog = append(backlog[:index], backlog[index+1:]...)
		}
	}

	if middleClaimRound == 0 || middleClaimRound > 10001 {
		t.Fatalf("middle window first claim round = %d, want a finite bound <= 10001", middleClaimRound)
	}
}

func TestFinalizeSchedulerHourlyWorkCannotBeStarvedByLongPeriodRecompute(t *testing.T) {
	selector := newFinalizeClaimSelector()
	counts := map[finalizeQueue]int{}
	available := map[finalizeQueue]bool{
		finalizeHourlyDevice: true,
		finalizeHourlyOther:  true,
		finalizeLongPeriod:   true,
	}

	for range 20 {
		counts[selector.nextQueue(available)]++
	}

	if counts[finalizeHourlyDevice] != 14 ||
		counts[finalizeHourlyOther] != 3 ||
		counts[finalizeLongPeriod] != 3 {
		t.Fatalf("20-slot quota = %v, want hourly device/other/long-period 14/3/3", counts)
	}
}

func TestFinalizeSchedulerBorrowsUnusedQuota(t *testing.T) {
	selector := newFinalizeClaimSelector()
	available := map[finalizeQueue]bool{finalizeLongPeriod: true}

	for index := range 20 {
		if got := selector.nextQueue(available); got != finalizeLongPeriod {
			t.Fatalf("slot %d = %v, unused hourly quota must be borrowed by long-period work", index, got)
		}
	}
}

func TestFinalizeSchedulerRunCycleKeepsFixedWorkerBoundAndRefills(t *testing.T) {
	deviceVersion := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	repo := newMemoryFinalizeRepository(
		finalizeTestWindows(deviceVersion, GranularityHourly, 24)...,
	)
	var active atomic.Int32
	var maximum atomic.Int32
	var finalized atomic.Int32
	scanner := newTestTimeoutScanner(repo, deviceVersion, 3, func(
		context.Context, WindowKey, CloseReason, uuid.UUID,
	) error {
		current := active.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(time.Millisecond)
		finalized.Add(1)
		active.Add(-1)
		return nil
	})

	if err := scanner.runOnce(context.Background()); err != nil {
		t.Fatalf("run finalize scheduler: %v", err)
	}
	if got := finalized.Load(); got != 24 {
		t.Fatalf("finalized windows = %d, want 24", got)
	}
	if got := maximum.Load(); got != 3 {
		t.Fatalf("maximum active finalizers = %d, want fixed worker bound 3 with continuous refill", got)
	}
}

func TestFinalizeSchedulerTwoScannersNeverFinalizeSameClaim(t *testing.T) {
	deviceVersion := uuid.MustParse("20000000-0000-4000-8000-000000000002")
	repo := newMemoryFinalizeRepository(
		finalizeTestWindows(deviceVersion, GranularityHourly, 40)...,
	)
	var finalizedMu sync.Mutex
	finalized := make(map[string]int)
	finalize := func(
		_ context.Context, key WindowKey, _ CloseReason, _ uuid.UUID,
	) error {
		finalizedMu.Lock()
		finalized[finalizeTestKey(key)]++
		finalizedMu.Unlock()
		time.Sleep(time.Millisecond)
		return nil
	}
	first := newTestTimeoutScanner(repo, deviceVersion, 2, finalize)
	second := newTestTimeoutScanner(repo, deviceVersion, 2, finalize)

	var workers sync.WaitGroup
	var firstErr, secondErr error
	workers.Add(2)
	go func() {
		defer workers.Done()
		firstErr = first.runOnce(context.Background())
	}()
	go func() {
		defer workers.Done()
		secondErr = second.runOnce(context.Background())
	}()
	workers.Wait()

	if firstErr != nil || secondErr != nil {
		t.Fatalf("scanner errors = %v, %v", firstErr, secondErr)
	}
	if len(finalized) != 40 {
		t.Fatalf("distinct finalized windows = %d, want 40", len(finalized))
	}
	for key, count := range finalized {
		if count != 1 {
			t.Fatalf("window %s finalized %d times, want exactly once", key, count)
		}
	}
}

func TestFinalizeSchedulerRunCycleAppliesQuotaToActualClaims(t *testing.T) {
	deviceVersion := uuid.MustParse("30000000-0000-4000-8000-000000000003")
	otherVersion := uuid.MustParse("40000000-0000-4000-8000-000000000004")
	windows := finalizeTestWindows(deviceVersion, GranularityHourly, 14)
	windows = append(windows, finalizeTestWindows(otherVersion, GranularityHourly, 3)...)
	windows = append(windows, finalizeTestWindows(otherVersion, GranularityDaily, 3)...)
	repo := newMemoryFinalizeRepository(windows...)
	var claimed []finalizeQueue
	scanner := newTestTimeoutScanner(repo, deviceVersion, 1, func(
		_ context.Context, key WindowKey, _ CloseReason, _ uuid.UUID,
	) error {
		switch {
		case key.Granularity != GranularityHourly:
			claimed = append(claimed, finalizeLongPeriod)
		case key.TaskVersionID == deviceVersion:
			claimed = append(claimed, finalizeHourlyDevice)
		default:
			claimed = append(claimed, finalizeHourlyOther)
		}
		return nil
	})

	if err := scanner.runOnce(context.Background()); err != nil {
		t.Fatalf("run finalize scheduler: %v", err)
	}
	if len(claimed) != len(finalizeQuotaWheel) {
		t.Fatalf("claimed count = %d, want %d", len(claimed), len(finalizeQuotaWheel))
	}
	for index, want := range finalizeQuotaWheel {
		if claimed[index] != want {
			t.Fatalf("claim %d queue = %v, want quota queue %v; sequence=%v", index, claimed[index], want, claimed)
		}
	}
}

func TestFinalizeSchedulerMetricsDistinguishLeaseConflictFromEmptyQueue(t *testing.T) {
	deviceVersion := uuid.MustParse("50000000-0000-4000-8000-000000000005")
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	empty := newTestTimeoutScanner(
		newMemoryFinalizeRepository(), deviceVersion, 2,
		func(context.Context, WindowKey, CloseReason, uuid.UUID) error { return nil },
	).SetMetrics(metrics)
	if err := empty.runOnce(context.Background()); err != nil {
		t.Fatalf("run empty scheduler: %v", err)
	}
	if got := testutil.ToFloat64(metrics.FinalizeClaimConflictsTotal); got != 0 {
		t.Fatalf("empty queues recorded %v claim conflicts, want 0", got)
	}

	leased := newTestTimeoutScanner(
		newMemoryFinalizeRepository(
			finalizeTestWindows(deviceVersion, GranularityHourly, 1)...,
		),
		deviceVersion,
		2,
		func(context.Context, WindowKey, CloseReason, uuid.UUID) error {
			time.Sleep(5 * time.Millisecond)
			return nil
		},
	).SetMetrics(metrics)
	if err := leased.runOnce(context.Background()); err != nil {
		t.Fatalf("run contended scheduler: %v", err)
	}
	if got := testutil.ToFloat64(metrics.FinalizeClaimConflictsTotal); got != 0 {
		t.Fatalf("scheduler's own active lease recorded %v claim conflicts, want 0", got)
	}
	if got := testutil.ToFloat64(metrics.FinalizeOldestDueSeconds); got != 0 {
		t.Fatalf("oldest due seconds after draining backlog = %v, want 0", got)
	}

	foreignRepo := newMemoryFinalizeRepository(
		finalizeTestWindows(deviceVersion, GranularityHourly, 1)...,
	)
	foreignOwner := uuid.New()
	_, err := foreignRepo.claimDue(
		context.Background(), GranularityHourly, time.Now(), 1,
		foreignOwner, time.Now().Add(time.Minute),
		claimVersionFilter{versionIDs: []uuid.UUID{deviceVersion}}, claimOldestFirst,
	)
	if err != nil {
		t.Fatalf("seed foreign lease: %v", err)
	}
	foreign := newTestTimeoutScanner(
		foreignRepo, deviceVersion, 1,
		func(context.Context, WindowKey, CloseReason, uuid.UUID) error {
			t.Fatal("foreign-leased window must not be finalized")
			return nil
		},
	).SetMetrics(metrics)
	if err := foreign.runOnce(context.Background()); err != nil {
		t.Fatalf("run foreign-contended scheduler: %v", err)
	}
	if got := testutil.ToFloat64(metrics.FinalizeClaimConflictsTotal); got != 1 {
		t.Fatalf("foreign active lease recorded %v claim conflicts, want 1", got)
	}
}

type memoryFinalizeWindow struct {
	record     WindowRecord
	leaseOwner uuid.UUID
	leaseUntil time.Time
	done       bool
}

type memoryFinalizeRepository struct {
	mu      sync.Mutex
	windows []memoryFinalizeWindow
}

func newMemoryFinalizeRepository(records ...WindowRecord) *memoryFinalizeRepository {
	windows := make([]memoryFinalizeWindow, len(records))
	for index, record := range records {
		windows[index].record = record
	}
	return &memoryFinalizeRepository{windows: windows}
}

func (r *memoryFinalizeRepository) CountWatermarkBlocked(
	context.Context,
	time.Time,
	map[Granularity]time.Duration,
	time.Duration,
) (int64, error) {
	return 0, nil
}

func (r *memoryFinalizeRepository) claimDue(
	_ context.Context,
	granularity Granularity,
	dueBefore time.Time,
	limit uint64,
	leaseOwner uuid.UUID,
	leaseUntil time.Time,
	filter claimVersionFilter,
	order claimOrder,
) ([]WindowRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	result := make([]WindowRecord, 0, limit)
	for uint64(len(result)) < limit {
		selected := -1
		for index := range r.windows {
			window := &r.windows[index]
			if window.done || window.record.Key.Granularity != granularity ||
				window.record.Key.End.After(dueBefore) ||
				(!window.leaseUntil.IsZero() && window.leaseUntil.After(now)) ||
				!matchesClaimVersionFilter(window.record.Key.TaskVersionID, filter) {
				continue
			}
			if selected < 0 ||
				(order == claimNewestFirst &&
					window.record.Key.End.After(r.windows[selected].record.Key.End)) ||
				(order == claimOldestFirst &&
					window.record.Key.End.Before(r.windows[selected].record.Key.End)) {
				selected = index
			}
		}
		if selected < 0 {
			break
		}
		r.windows[selected].leaseOwner = leaseOwner
		r.windows[selected].leaseUntil = leaseUntil
		result = append(result, r.windows[selected].record)
	}
	return result, nil
}

func (r *memoryFinalizeRepository) CompleteClaim(
	_ context.Context,
	key WindowKey,
	leaseOwner uuid.UUID,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.windows {
		window := &r.windows[index]
		if sameFinalizeTestWindow(window.record.Key, key) {
			if window.leaseOwner != leaseOwner {
				return ErrFinalizeClaimLost
			}
			window.done = true
			window.leaseOwner = uuid.Nil
			window.leaseUntil = time.Time{}
			return nil
		}
	}
	return fmt.Errorf("complete unknown finalize test window")
}

func (r *memoryFinalizeRepository) hasClaimConflict(
	_ context.Context,
	granularity Granularity,
	dueBefore time.Time,
	filter claimVersionFilter,
	leaseOwner uuid.UUID,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for index := range r.windows {
		window := &r.windows[index]
		if !window.done &&
			window.record.Key.Granularity == granularity &&
			!window.record.Key.End.After(dueBefore) &&
			window.leaseUntil.After(now) &&
			window.leaseOwner != leaseOwner &&
			matchesClaimVersionFilter(window.record.Key.TaskVersionID, filter) {
			return true, nil
		}
	}
	return false, nil
}

func (r *memoryFinalizeRepository) ReleaseClaim(
	_ context.Context,
	key WindowKey,
	leaseOwner uuid.UUID,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.windows {
		window := &r.windows[index]
		if sameFinalizeTestWindow(window.record.Key, key) {
			if window.leaseOwner != leaseOwner {
				return ErrFinalizeClaimLost
			}
			window.leaseOwner = uuid.Nil
			window.leaseUntil = time.Time{}
			return nil
		}
	}
	return fmt.Errorf("release unknown finalize test window")
}

func newTestTimeoutScanner(
	repo finalizeWindowRepository,
	deviceVersion uuid.UUID,
	workerCount int,
	finalize func(context.Context, WindowKey, CloseReason, uuid.UUID) error,
) *TimeoutScanner {
	snapshot := NewSnapshotStore(nil, zap.NewNop())
	snapshot.value.Store(&TaskSnapshot{
		LoadedAt: time.Now(),
		ByVersion: map[uuid.UUID]*TaskVersionSnapshot{
			deviceVersion: {VersionID: deviceVersion, Dimension: DimensionDevice},
		},
	})
	return &TimeoutScanner{
		windows: repo, finalizeWindow: finalize, snapshot: snapshot,
		workerCount: workerCount, grace: time.Nanosecond,
		graceByGranularity: map[Granularity]time.Duration{
			GranularityHourly: time.Nanosecond, GranularityDaily: time.Nanosecond,
			GranularityWeekly: time.Nanosecond, GranularityMonthly: time.Nanosecond,
		},
		logger: zap.NewNop(), leaseOwner: uuid.New(), selector: newFinalizeClaimSelector(),
	}
}

func finalizeTestWindows(
	versionID uuid.UUID,
	granularity Granularity,
	count int,
) []WindowRecord {
	end := time.Now().Add(-time.Hour)
	result := make([]WindowRecord, 0, count)
	for index := range count {
		start := end.Add(-time.Duration(index+1) * time.Hour)
		result = append(result, WindowRecord{Key: WindowKey{
			TaskID: uuid.New(), TaskVersionID: versionID,
			EntityKey:   fmt.Sprintf("%s-%d", granularity, index),
			Granularity: granularity, Start: start, End: end,
		}})
	}
	return result
}

func matchesClaimVersionFilter(versionID uuid.UUID, filter claimVersionFilter) bool {
	if filter.versionIDs == nil {
		return true
	}
	found := false
	for _, candidate := range filter.versionIDs {
		found = found || candidate == versionID
	}
	if filter.exclude {
		return !found
	}
	return found
}

func sameFinalizeTestWindow(left, right WindowKey) bool {
	return left.TaskVersionID == right.TaskVersionID &&
		left.EntityKey == right.EntityKey &&
		left.Granularity == right.Granularity &&
		left.Start.Equal(right.Start)
}

func finalizeTestKey(key WindowKey) string {
	return fmt.Sprintf(
		"%s/%s/%s/%d",
		key.TaskVersionID, key.EntityKey, key.Granularity, key.Start.UnixNano(),
	)
}
