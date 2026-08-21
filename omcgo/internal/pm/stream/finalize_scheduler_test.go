package stream

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func TestFinalizeSchedulerReplaysOnlyIncompleteDeviceHours(t *testing.T) {
	deviceVersion := uuid.New()
	windows := finalizeTestWindows(deviceVersion, GranularityHourly, 2)
	windows[0].ExpectedSlots, windows[0].ReceivedSlots = 4, 4
	windows[1].ExpectedSlots, windows[1].ReceivedSlots = 4, 3
	repo := newMemoryFinalizeRepository(windows...)
	replayed := make(map[string]int)
	reasons := make(map[string]CloseReason)
	scanner := newTestTimeoutScanner(repo, deviceVersion, 1, func(
		_ context.Context, key WindowKey, reason CloseReason, _ uuid.UUID,
	) error {
		reasons[key.EntityKey] = reason
		return nil
	}).SetIncompleteDeviceHourReplay(func(
		_ context.Context, key WindowKey,
	) (bool, error) {
		replayed[key.EntityKey]++
		return true, nil
	})

	if err := scanner.runOnce(context.Background()); err != nil {
		t.Fatalf("run device-hour replay scheduler: %v", err)
	}
	if replayed[windows[0].Key.EntityKey] != 0 {
		t.Fatalf("complete 4/4 window was replayed")
	}
	if replayed[windows[1].Key.EntityKey] != 1 {
		t.Fatalf("incomplete 3/4 window replay calls = %d, want 1", replayed[windows[1].Key.EntityKey])
	}
	if reasons[windows[0].Key.EntityKey] != CloseComplete ||
		reasons[windows[1].Key.EntityKey] != CloseComplete {
		t.Fatalf("final reasons = %v, replayed 3/4 and original 4/4 must both be complete", reasons)
	}
}

func TestFinalizeSchedulerSharesOneDeviceHourReplayAcrossTaskVersions(t *testing.T) {
	firstVersion, secondVersion := uuid.New(), uuid.New()
	deviceID := uuid.MustParse("30f3d50f-e1c4-4c3a-89b1-ac9da9a51350")
	start := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	windows := []WindowRecord{
		{Key: WindowKey{
			TaskID: uuid.New(), TaskVersionID: firstVersion, EntityKey: deviceID.String(),
			Granularity: GranularityHourly, Start: start, End: start.Add(time.Hour),
		}, ExpectedSlots: 4, ReceivedSlots: 3},
		{Key: WindowKey{
			TaskID: uuid.New(), TaskVersionID: secondVersion, EntityKey: deviceID.String(),
			Granularity: GranularityHourly, Start: start, End: start.Add(time.Hour),
		}, ExpectedSlots: 4, ReceivedSlots: 2},
	}
	repo := newMemoryFinalizeRepository(windows...)
	reasons := make(map[uuid.UUID]CloseReason)
	var reasonsMu sync.Mutex
	scanner := newTestTimeoutScanner(repo, firstVersion, 1, func(
		_ context.Context, key WindowKey, reason CloseReason, _ uuid.UUID,
	) error {
		reasonsMu.Lock()
		defer reasonsMu.Unlock()
		reasons[key.TaskVersionID] = reason
		return nil
	})
	scanner.snapshot.Current().ByVersion[secondVersion] = &TaskVersionSnapshot{
		VersionID: secondVersion, Dimension: DimensionDevice, DevicePipeline: true,
	}
	var reads atomic.Int64
	applied := make(map[uuid.UUID]int)
	scanner.SetIncompleteDeviceHourVersionReplay(func(
		context.Context, WindowKey,
	) (*DeviceHourReplayBatch, error) {
		reads.Add(1)
		return &DeviceHourReplayBatch{Contributions: map[uuid.UUID][]Contribution{
			firstVersion:  {{Key: windows[0].Key}},
			secondVersion: {{Key: windows[1].Key}},
		}}, nil
	}, func(
		_ context.Context, key WindowKey, _ *DeviceHourReplayBatch,
	) (DeviceHourReplayOutcome, error) {
		applied[key.TaskVersionID]++
		return DeviceHourReplayOutcome{
			ReceivedSlots: 4, Complete: true, Matched: true,
		}, nil
	})

	if err := scanner.runOnce(context.Background()); err != nil {
		t.Fatalf("run shared device-hour replay scheduler: %v", err)
	}
	if got := reads.Load(); got != 1 {
		t.Fatalf("durable device-hour reads = %d, want one shared read", got)
	}
	for _, versionID := range []uuid.UUID{firstVersion, secondVersion} {
		if applied[versionID] != 1 {
			t.Fatalf("version %s apply calls = %d, want one claimed apply", versionID, applied[versionID])
		}
		if reasons[versionID] != CloseComplete {
			t.Fatalf("version %s close reason = %s, want complete", versionID, reasons[versionID])
		}
	}
}

func TestDeviceHourReplayCacheEvictsCompletedEntriesAtFixedBound(t *testing.T) {
	cache := newDeviceHourReplayCache(4)
	start := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	for index := range 40 {
		key := deviceHourReplayKey{
			entityKey: fmt.Sprintf("device-%02d", index),
			startNS:   start.UnixNano(), endNS: start.Add(time.Hour).UnixNano(),
		}
		cache.acquire(key)
		cache.release(key)
		if len(cache.entries) > 4 {
			t.Fatalf("completed replay cache entries = %d, want <= 4", len(cache.entries))
		}
	}
}

func TestFinalizeSchedulerReusesReplayAcrossClaimBatches(t *testing.T) {
	const devices = 40
	start := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	var windows []WindowRecord
	versionIDs := make([][2]uuid.UUID, 0, devices)
	for range devices {
		deviceID := uuid.NewString()
		versions := [2]uuid.UUID{uuid.New(), uuid.New()}
		versionIDs = append(versionIDs, versions)
		for _, versionID := range versions {
			windows = append(windows, WindowRecord{Key: WindowKey{
				TaskID: uuid.New(), TaskVersionID: versionID, EntityKey: deviceID,
				Granularity: GranularityHourly, Start: start, End: start.Add(time.Hour),
			}, ExpectedSlots: 4, ReceivedSlots: 3})
		}
	}
	repo := newMemoryFinalizeRepository(windows...)
	scanner := newTestTimeoutScanner(repo, versionIDs[0][0], 1, func(
		context.Context, WindowKey, CloseReason, uuid.UUID,
	) error {
		return nil
	})
	for _, versions := range versionIDs {
		for _, versionID := range versions {
			scanner.snapshot.Current().ByVersion[versionID] = &TaskVersionSnapshot{
				VersionID: versionID, Dimension: DimensionDevice, DevicePipeline: true,
			}
		}
	}
	reads := make(map[string]int)
	scanner.SetIncompleteDeviceHourVersionReplay(func(
		_ context.Context, key WindowKey,
	) (*DeviceHourReplayBatch, error) {
		reads[key.EntityKey]++
		return &DeviceHourReplayBatch{Contributions: map[uuid.UUID][]Contribution{
			key.TaskVersionID: {{Key: key}},
		}}, nil
	}, func(
		context.Context, WindowKey, *DeviceHourReplayBatch,
	) (DeviceHourReplayOutcome, error) {
		return DeviceHourReplayOutcome{ReceivedSlots: 4, Complete: true, Matched: true}, nil
	})

	if err := scanner.runOnce(context.Background()); err != nil {
		t.Fatalf("run cross-batch replay scheduler: %v", err)
	}
	if len(reads) != devices {
		t.Fatalf("device-hour replay reads tracked devices = %d, want %d", len(reads), devices)
	}
	for deviceID, count := range reads {
		if count != 1 {
			t.Fatalf("device %s replay reads = %d, want 1 across claim batches", deviceID, count)
		}
	}
}

func TestFinalizeSchedulerDoesNotAccumulateUnclaimedPreparedOrPublishedVersion(t *testing.T) {
	claimedVersion, unclaimedVersion := uuid.New(), uuid.New()
	deviceID := uuid.New()
	start := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	claimed := WindowRecord{Key: WindowKey{
		TaskID: uuid.New(), TaskVersionID: claimedVersion, EntityKey: deviceID.String(),
		Granularity: GranularityHourly, Start: start, End: start.Add(time.Hour),
	}, ExpectedSlots: 4, ReceivedSlots: 3}
	unclaimedKey := claimed.Key
	unclaimedKey.TaskID, unclaimedKey.TaskVersionID = uuid.New(), unclaimedVersion
	for _, status := range []string{"prepared", "published"} {
		t.Run(status, func(t *testing.T) {
			unclaimed := WindowRecord{
				Key: unclaimedKey, Status: status, ExpectedSlots: 4, ReceivedSlots: 4,
			}
			repo := newMemoryFinalizeRepository(claimed, unclaimed)
			scanner := newTestTimeoutScanner(repo, claimedVersion, 1, func(
				context.Context, WindowKey, CloseReason, uuid.UUID,
			) error {
				return nil
			})
			scanner.snapshot.Current().ByVersion[unclaimedVersion] = &TaskVersionSnapshot{
				VersionID: unclaimedVersion, Dimension: DimensionDevice, DevicePipeline: true,
			}
			applied := make(map[uuid.UUID]int)
			scanner.SetIncompleteDeviceHourVersionReplay(func(
				context.Context, WindowKey,
			) (*DeviceHourReplayBatch, error) {
				return &DeviceHourReplayBatch{Contributions: map[uuid.UUID][]Contribution{
					claimedVersion:   {{Key: claimed.Key}},
					unclaimedVersion: {{Key: unclaimedKey}},
				}}, nil
			}, func(
				_ context.Context, key WindowKey, _ *DeviceHourReplayBatch,
			) (DeviceHourReplayOutcome, error) {
				applied[key.TaskVersionID]++
				return DeviceHourReplayOutcome{
					ReceivedSlots: 4, Complete: true, Matched: true,
				}, nil
			})

			if err := scanner.runOnce(context.Background()); err != nil {
				t.Fatalf("run claimed-only device-hour replay: %v", err)
			}
			if applied[claimedVersion] != 1 {
				t.Fatalf("claimed version apply calls = %d, want 1", applied[claimedVersion])
			}
			if applied[unclaimedVersion] != 0 {
				t.Fatalf("%s unclaimed version apply calls = %d, want 0", status, applied[unclaimedVersion])
			}
		})
	}
}

func TestFinalizeSchedulerKeepsSharedReplayOutcomesIndependentPerVersion(t *testing.T) {
	firstVersion, secondVersion := uuid.New(), uuid.New()
	deviceID := uuid.New()
	start := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	windows := []WindowRecord{
		{Key: WindowKey{TaskID: uuid.New(), TaskVersionID: firstVersion,
			EntityKey: deviceID.String(), Granularity: GranularityHourly,
			Start: start, End: start.Add(time.Hour)}, ExpectedSlots: 4, ReceivedSlots: 3},
		{Key: WindowKey{TaskID: uuid.New(), TaskVersionID: secondVersion,
			EntityKey: deviceID.String(), Granularity: GranularityHourly,
			Start: start, End: start.Add(time.Hour)}, ExpectedSlots: 4, ReceivedSlots: 3},
	}
	repo := newMemoryFinalizeRepository(windows...)
	finalized := make(map[uuid.UUID]CloseReason)
	var finalizedMu sync.Mutex
	scanner := newTestTimeoutScanner(repo, firstVersion, 2, func(
		_ context.Context, key WindowKey, reason CloseReason, _ uuid.UUID,
	) error {
		finalizedMu.Lock()
		defer finalizedMu.Unlock()
		finalized[key.TaskVersionID] = reason
		return nil
	})
	scanner.snapshot.Current().ByVersion[secondVersion] = &TaskVersionSnapshot{
		VersionID: secondVersion, Dimension: DimensionDevice, DevicePipeline: true,
	}
	versionErr := errors.New("second version accumulator failed")
	scanner.SetIncompleteDeviceHourVersionReplay(func(
		context.Context, WindowKey,
	) (*DeviceHourReplayBatch, error) {
		return &DeviceHourReplayBatch{Contributions: map[uuid.UUID][]Contribution{
			firstVersion:  {{Key: windows[0].Key}},
			secondVersion: {{Key: windows[1].Key}},
		}}, nil
	}, func(
		_ context.Context, key WindowKey, _ *DeviceHourReplayBatch,
	) (DeviceHourReplayOutcome, error) {
		if key.TaskVersionID == secondVersion {
			return DeviceHourReplayOutcome{Matched: true}, versionErr
		}
		return DeviceHourReplayOutcome{
			ReceivedSlots: 4, Complete: true, Matched: true,
		}, nil
	})

	err := scanner.runOnce(context.Background())
	if !errors.Is(err, versionErr) {
		t.Fatalf("shared replay scheduler error = %v, want second-version failure", err)
	}
	if finalized[firstVersion] != CloseComplete {
		t.Fatalf("healthy version close reason = %s, want complete", finalized[firstVersion])
	}
	if _, ok := finalized[secondVersion]; ok {
		t.Fatalf("failed version was finalized: %v", finalized)
	}
	if state := repo.window(windows[0].Key); !state.done {
		t.Fatal("healthy version claim was not completed")
	}
	if state := repo.window(windows[1].Key); state.record.Status != "failed" {
		t.Fatalf("failed version claim status = %s, want failed", state.record.Status)
	}
}

func TestFinalizeSchedulerKeepsTimeoutWhenDurableReplayCannotFillGap(t *testing.T) {
	deviceVersion := uuid.New()
	windows := finalizeTestWindows(deviceVersion, GranularityHourly, 1)
	windows[0].ExpectedSlots, windows[0].ReceivedSlots = 4, 3
	repo := newMemoryFinalizeRepository(windows...)
	var reason CloseReason
	scanner := newTestTimeoutScanner(repo, deviceVersion, 1, func(
		_ context.Context, _ WindowKey, got CloseReason, _ uuid.UUID,
	) error {
		reason = got
		return nil
	}).SetIncompleteDeviceHourReplay(func(
		context.Context, WindowKey,
	) (bool, error) {
		return false, nil
	})

	if err := scanner.runOnce(context.Background()); err != nil {
		t.Fatalf("run incomplete replay scheduler: %v", err)
	}
	if reason != CloseTimeout {
		t.Fatalf("unfilled device hour reason = %s, want timeout", reason)
	}
}

func TestFinalizeSchedulerDoesNotPublishWhenDurableReplayFails(t *testing.T) {
	deviceVersion := uuid.New()
	windows := finalizeTestWindows(deviceVersion, GranularityHourly, 1)
	windows[0].ExpectedSlots, windows[0].ReceivedSlots = 4, 3
	repo := newMemoryFinalizeRepository(windows...)
	var finalized atomic.Int64
	scanner := newTestTimeoutScanner(repo, deviceVersion, 1, func(
		context.Context, WindowKey, CloseReason, uuid.UUID,
	) error {
		finalized.Add(1)
		return nil
	}).SetIncompleteDeviceHourReplay(func(
		context.Context, WindowKey,
	) (bool, error) {
		return false, errors.New("durable replay unavailable")
	})

	err := scanner.runOnce(context.Background())
	if err == nil || !strings.Contains(err.Error(), "durable replay unavailable") {
		t.Fatalf("run device-hour replay scheduler error = %v", err)
	}
	if finalized.Load() != 0 {
		t.Fatalf("finalizer calls = %d, want 0 after replay failure", finalized.Load())
	}
}

func TestHourlyPreparationStartsAtWindowEndWhileLongPeriodsKeepGrace(t *testing.T) {
	now := time.Date(2026, 8, 1, 15, 0, 30, 0, time.UTC)

	if got := preparationDueBefore(now, GranularityHourly, 12*time.Minute); !got.Equal(now) {
		t.Fatalf("hourly prepare due before = %v, want window end cutoff %v", got, now)
	}
	if got := preparationDueBefore(now, GranularityDaily, 15*time.Minute); !got.Equal(now.Add(-15 * time.Minute)) {
		t.Fatalf("daily prepare due before = %v, want grace cutoff", got)
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

func TestFinalizeSchedulerClaimsBacklogInBoundedBatches(t *testing.T) {
	deviceVersion := uuid.MustParse("11000000-0000-4000-8000-000000000011")
	repo := newMemoryFinalizeRepository(
		finalizeTestWindows(deviceVersion, GranularityHourly, 64)...,
	)
	scanner := newTestTimeoutScanner(repo, deviceVersion, 32, func(
		context.Context, WindowKey, CloseReason, uuid.UUID,
	) error {
		return nil
	})

	if err := scanner.runOnce(context.Background()); err != nil {
		t.Fatalf("run batched finalize scheduler: %v", err)
	}
	if got := repo.claimCallCount(); got > 10 {
		t.Fatalf("claim SQL calls = %d, want <= 10 for 64 windows with 32 workers", got)
	}
}

func TestFinalizeSchedulerPoisonWindowDoesNotStopHealthyBacklogAcrossQueues(t *testing.T) {
	deviceVersion := uuid.MustParse("18000000-0000-4000-8000-000000000018")
	otherVersion := uuid.MustParse("19000000-0000-4000-8000-000000000019")
	deviceWindows := finalizeTestWindows(deviceVersion, GranularityHourly, 2)
	deviceWindows[0].Key.EntityKey = "poison"
	otherWindows := finalizeTestWindows(otherVersion, GranularityHourly, 1)
	longWindows := finalizeTestWindows(otherVersion, GranularityDaily, 1)
	repo := newMemoryFinalizeRepository(append(
		append(deviceWindows, otherWindows...),
		longWindows...,
	)...)
	finalized := make(map[string]int)
	scanner := newTestTimeoutScanner(repo, deviceVersion, 1, func(
		_ context.Context, key WindowKey, _ CloseReason, _ uuid.UUID,
	) error {
		if key.EntityKey == "poison" {
			return errors.New("permanent formula error")
		}
		finalized[finalizeTestKey(key)]++
		return nil
	})

	err := scanner.runOnce(context.Background())
	if err == nil || !strings.Contains(err.Error(), "permanent formula error") {
		t.Fatalf("scheduler error = %v, want the isolated poison failure", err)
	}
	if len(finalized) != 3 {
		t.Fatalf("healthy finalized windows = %d, want device/other/long-period backlog all drained", len(finalized))
	}
	poison := repo.window(deviceWindows[0].Key)
	if poison.attempts != 1 {
		t.Fatalf("poison attempts = %d, want one persisted failure", poison.attempts)
	}
	if !poison.nextAttemptAt.After(repo.currentTime()) {
		t.Fatalf("poison next attempt = %v, want persistent retry deferral", poison.nextAttemptAt)
	}
}

func TestFinalizeRetryDelayIsBoundedExponential(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 30 * time.Second},
		{attempt: 2, want: time.Minute},
		{attempt: 3, want: 2 * time.Minute},
		{attempt: 8, want: 30 * time.Minute},
		{attempt: 100, want: 30 * time.Minute},
	}
	for _, test := range tests {
		if got := finalizeRetryDelay(test.attempt); got != test.want {
			t.Fatalf("finalizeRetryDelay(%d) = %s, want %s", test.attempt, got, test.want)
		}
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
	if got := testutil.ToFloat64(metrics.FinalizeOldestDueSeconds); got <= 0 {
		t.Fatalf("foreign active lease oldest due seconds = %v, want an alertable positive age", got)
	}
}

func TestFinalizeOldestDueGaugeGrowsWithRepositoryTimeWhileLeaseIsActive(t *testing.T) {
	deviceVersion := uuid.MustParse("58000000-0000-4000-8000-000000000058")
	repo := newMemoryFinalizeRepository(
		finalizeTestWindows(deviceVersion, GranularityHourly, 1)...,
	)
	base := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	repo.setCurrentTime(base)
	repo.windows[0].record.Key.End = base.Add(-time.Hour)
	repo.windows[0].leaseOwner = uuid.New()
	repo.windows[0].leaseUntil = base.Add(time.Hour)
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	scanner := newTestTimeoutScanner(
		repo, deviceVersion, 1,
		func(context.Context, WindowKey, CloseReason, uuid.UUID) error {
			t.Fatal("actively leased window must not be finalized by this scanner")
			return nil
		},
	).SetMetrics(metrics)

	if err := scanner.refreshOldestDue(context.Background()); err != nil {
		t.Fatalf("refresh oldest due at base time: %v", err)
	}
	first := testutil.ToFloat64(metrics.FinalizeOldestDueSeconds)
	repo.setCurrentTime(base.Add(10 * time.Minute))
	if err := scanner.refreshOldestDue(context.Background()); err != nil {
		t.Fatalf("refresh oldest due after clock advance: %v", err)
	}
	second := testutil.ToFloat64(metrics.FinalizeOldestDueSeconds)
	if delta := second - first; delta < 599.999 || delta > 600.001 {
		t.Fatalf("oldest due grew by %v seconds, want %v", second-first, (10 * time.Minute).Seconds())
	}
}

func TestFinalizeSchedulerSameScannerReclaimUsesNewTokenAndRejectsStaleCompletion(t *testing.T) {
	deviceVersion := uuid.MustParse("60000000-0000-4000-8000-000000000006")
	repo := newMemoryFinalizeRepository(
		finalizeTestWindows(deviceVersion, GranularityHourly, 1)...,
	)
	scanner := newTestTimeoutScanner(
		repo, deviceVersion, 1,
		func(context.Context, WindowKey, CloseReason, uuid.UUID) error { return nil },
	)
	scanner.claimLease = 5 * time.Millisecond

	first, err := scanner.claimNext(context.Background(), time.Now(), finalizeHourlyDevice)
	if err != nil || first.window == nil {
		t.Fatalf("first claim = %+v, %v", first, err)
	}
	time.Sleep(8 * time.Millisecond)
	second, err := scanner.claimNext(context.Background(), time.Now(), finalizeHourlyDevice)
	if err != nil || second.window == nil {
		t.Fatalf("second claim = %+v, %v", second, err)
	}
	if first.token == second.token {
		t.Fatalf("reclaimed window reused token %s", first.token)
	}
	if err := repo.CompleteClaim(context.Background(), first.window.Key, first.token); !errors.Is(err, ErrFinalizeClaimLost) {
		t.Fatalf("stale completion error = %v, want ErrFinalizeClaimLost", err)
	}
	if err := repo.ReleaseClaim(context.Background(), first.window.Key, first.token); !errors.Is(err, ErrFinalizeClaimLost) {
		t.Fatalf("stale release error = %v, want ErrFinalizeClaimLost", err)
	}
	if err := repo.CompleteClaim(context.Background(), second.window.Key, second.token); err != nil {
		t.Fatalf("current token completion: %v", err)
	}
}

func TestFinalizeSchedulerRenewsShortLeaseDuringSlowFinalize(t *testing.T) {
	deviceVersion := uuid.MustParse("70000000-0000-4000-8000-000000000007")
	repo := newMemoryFinalizeRepository(
		finalizeTestWindows(deviceVersion, GranularityHourly, 1)...,
	)
	started := make(chan struct{})
	scanner := newTestTimeoutScanner(repo, deviceVersion, 1, func(
		context.Context, WindowKey, CloseReason, uuid.UUID,
	) error {
		close(started)
		time.Sleep(40 * time.Millisecond)
		return nil
	})
	scanner.claimLease = 12 * time.Millisecond
	scanner.renewInterval = 3 * time.Millisecond
	runDone := make(chan error, 1)
	go func() { runDone <- scanner.runOnce(context.Background()) }()
	<-started
	time.Sleep(20 * time.Millisecond)

	reclaimed, err := repo.claimDue(
		context.Background(), GranularityHourly, time.Now(), 1,
		uuid.New(), time.Now().Add(time.Minute),
		claimVersionFilter{versionIDs: []uuid.UUID{deviceVersion}}, claimOldestFirst,
	)
	if err != nil {
		t.Fatalf("try reclaim during slow finalize: %v", err)
	}
	if len(reclaimed) != 0 {
		t.Fatalf("slow finalize lease expired despite renewal: reclaimed=%v", reclaimed)
	}
	if err := <-runDone; err != nil {
		t.Fatalf("slow finalize scheduler: %v", err)
	}
}

func TestHourlyDeviceClaimSkipsDeviceRollupEvenWhenItSortsFirst(t *testing.T) {
	rollupVersion := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	pipelineVersion := uuid.MustParse("ffffffff-ffff-4fff-8fff-ffffffffffff")
	end := time.Now().UTC().Add(-time.Hour)
	rollup := WindowRecord{Key: WindowKey{
		TaskID: uuid.New(), TaskVersionID: rollupVersion, EntityKey: uuid.NewString(),
		Granularity: GranularityHourly, Start: end.Add(-time.Hour), End: end,
	}}
	pipeline := rollup
	pipeline.Key.TaskID = uuid.New()
	pipeline.Key.TaskVersionID = pipelineVersion
	pipeline.Key.EntityKey = uuid.NewString()
	repo := newMemoryFinalizeRepository(rollup, pipeline)
	scanner := newTestTimeoutScanner(repo, pipelineVersion, 1, func(
		context.Context, WindowKey, CloseReason, uuid.UUID,
	) error {
		return nil
	})
	scanner.snapshot.Current().ByVersion[rollupVersion] = &TaskVersionSnapshot{
		VersionID: rollupVersion, Dimension: DimensionDevice, DeviceRollup: true,
	}

	claim, err := scanner.claimNext(context.Background(), time.Now(), finalizeHourlyDevice)
	if err != nil {
		t.Fatalf("claim device pipeline hour: %v", err)
	}
	if claim.window == nil || claim.window.Key.TaskVersionID != pipelineVersion {
		t.Fatalf("hourly device claim = %+v, want DevicePipeline version %s", claim.window, pipelineVersion)
	}
}

type memoryFinalizeWindow struct {
	record        WindowRecord
	leaseOwner    uuid.UUID
	leaseUntil    time.Time
	done          bool
	attempts      int
	nextAttemptAt time.Time
}

type memoryFinalizeRepository struct {
	mu            sync.Mutex
	windows       []memoryFinalizeWindow
	now           time.Time
	claimDueCalls int
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
	r.claimDueCalls++
	now := r.currentTimeLocked()
	result := make([]WindowRecord, 0, limit)
	for uint64(len(result)) < limit {
		selected := -1
		for index := range r.windows {
			window := &r.windows[index]
			if window.done || window.record.Key.Granularity != granularity ||
				(window.record.Status != "" && window.record.Status != "open" &&
					window.record.Status != "failed") ||
				window.record.Key.End.After(dueBefore) ||
				(!window.leaseUntil.IsZero() && window.leaseUntil.After(now)) ||
				window.nextAttemptAt.After(now) ||
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
		r.windows[selected].record.FinalizeAttempts = r.windows[selected].attempts
		result = append(result, r.windows[selected].record)
	}
	return result, nil
}

func (r *memoryFinalizeRepository) claimCallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.claimDueCalls
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

func (r *memoryFinalizeRepository) FailClaim(
	_ context.Context,
	key WindowKey,
	leaseOwner uuid.UUID,
	cause error,
	retryAfter time.Duration,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.windows {
		window := &r.windows[index]
		if sameFinalizeTestWindow(window.record.Key, key) {
			if window.leaseOwner != leaseOwner {
				return ErrFinalizeClaimLost
			}
			window.record.Status = "failed"
			window.attempts++
			window.nextAttemptAt = r.currentTimeLocked().Add(retryAfter)
			window.leaseOwner = uuid.Nil
			window.leaseUntil = time.Time{}
			return nil
		}
	}
	return fmt.Errorf("fail unknown finalize test window: %v", cause)
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
	now := r.currentTimeLocked()
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

func (r *memoryFinalizeRepository) OldestDue(
	_ context.Context,
	graceByGranularity map[Granularity]time.Duration,
	fallbackGrace time.Duration,
) (time.Duration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.currentTimeLocked()
	var oldest time.Duration
	for index := range r.windows {
		window := &r.windows[index]
		if window.done ||
			(window.record.Status != "" && window.record.Status != "open" &&
				window.record.Status != "failed") {
			continue
		}
		grace := graceByGranularity[window.record.Key.Granularity]
		if grace <= 0 {
			grace = fallbackGrace
		}
		age := now.Sub(window.record.Key.End.Add(grace))
		if age > oldest {
			oldest = age
		}
	}
	return oldest, nil
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

func (r *memoryFinalizeRepository) RenewClaim(
	_ context.Context,
	key WindowKey,
	leaseOwner uuid.UUID,
	leaseUntil time.Time,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.currentTimeLocked()
	for index := range r.windows {
		window := &r.windows[index]
		if sameFinalizeTestWindow(window.record.Key, key) {
			if window.leaseOwner != leaseOwner || !window.leaseUntil.After(now) {
				return ErrFinalizeClaimLost
			}
			window.leaseUntil = leaseUntil
			return nil
		}
	}
	return fmt.Errorf("renew unknown finalize test window")
}

func (r *memoryFinalizeRepository) setCurrentTime(now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.now = now
}

func (r *memoryFinalizeRepository) currentTime() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentTimeLocked()
}

func (r *memoryFinalizeRepository) currentTimeLocked() time.Time {
	if !r.now.IsZero() {
		return r.now
	}
	return time.Now()
}

func (r *memoryFinalizeRepository) window(key WindowKey) memoryFinalizeWindow {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.windows {
		if sameFinalizeTestWindow(r.windows[index].record.Key, key) {
			return r.windows[index]
		}
	}
	return memoryFinalizeWindow{}
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
			deviceVersion: {
				VersionID: deviceVersion, Dimension: DimensionDevice, DevicePipeline: true,
			},
		},
	})
	return &TimeoutScanner{
		windows: repo, finalizeWindow: finalize, snapshot: snapshot,
		workerCount: workerCount, grace: time.Nanosecond,
		graceByGranularity: map[Granularity]time.Duration{
			GranularityHourly: time.Nanosecond, GranularityDaily: time.Nanosecond,
			GranularityWeekly: time.Nanosecond, GranularityMonthly: time.Nanosecond,
		},
		logger: zap.NewNop(), claimLease: finalizeClaimLease,
		renewInterval: finalizeClaimLease / 3, selector: newFinalizeClaimSelector(),
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
