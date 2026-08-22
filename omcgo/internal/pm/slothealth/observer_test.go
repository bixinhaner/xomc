package slothealth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLatestEligibleSlotEnd_UsesTwelveMinuteGrace(t *testing.T) {
	beforeBoundary := time.Date(2026, 8, 3, 15, 11, 59, 0, time.UTC)
	atBoundary := time.Date(2026, 8, 3, 15, 12, 0, 0, time.UTC)

	assert.Equal(t, time.Date(2026, 8, 3, 14, 45, 0, 0, time.UTC),
		LatestEligibleSlotEnd(beforeBoundary, 12*time.Minute))
	assert.Equal(t, time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC),
		LatestEligibleSlotEnd(atBoundary, 12*time.Minute))
}

func TestBuildSnapshots_ClassifiesCoverageAndKeepsVersion(t *testing.T) {
	end := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)
	expected := []ExpectedGroup{
		{Technology: "lte", Carrier: "cmcc", Devices: 20000, SnapshotVersion: "version-lte"},
		{Technology: "nr", Carrier: "cmcc", Devices: 100, SnapshotVersion: "version-nr"},
		{Technology: "gsm", Carrier: "cucc", Devices: 50, SnapshotVersion: "version-gsm"},
	}
	received := []ReceivedGroup{
		{Technology: "lte", Carrier: "cmcc", Devices: 19600},
		{Technology: "nr", Carrier: "cmcc", Devices: 94},
	}

	got := BuildSnapshots(end, end.Add(12*time.Minute), time.Time{}, expected, received)

	require.Len(t, got, 3)
	assert.Equal(t, StatusComplete, got[0].Status)
	assert.InDelta(t, 0.98, got[0].CoverageRatio, 1e-9)
	assert.Equal(t, "version-lte", got[0].ExpectedSnapshotVersion)
	assert.Equal(t, StatusPartial, got[1].Status)
	assert.InDelta(t, 0.94, got[1].CoverageRatio, 1e-9)
	assert.Equal(t, StatusMissing, got[2].Status)
	assert.Zero(t, got[2].ReceivedDevices)
}

func TestBuildSnapshots_MarksSlotTouchedByStartupAsIgnored(t *testing.T) {
	end := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)
	startup := end.Add(-5 * time.Minute)
	expected := []ExpectedGroup{{Technology: "lte", Carrier: "cmcc", Devices: 20000}}

	got := BuildSnapshots(end, end.Add(12*time.Minute), startup, expected, nil)

	require.Len(t, got, 1)
	assert.Equal(t, StatusBootstrapIgnored, got[0].Status)
	assert.Zero(t, got[0].CoverageRatio)
}

func TestBuildSnapshots_ClassifiesFullyReceivedStartupSlotAsComplete(t *testing.T) {
	end := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)
	startup := end.Add(-5 * time.Minute)
	expected := []ExpectedGroup{{Technology: "lte", Carrier: "cmcc", Devices: 20000}}
	received := []ReceivedGroup{{Technology: "lte", Carrier: "cmcc", Devices: 20000}}

	got := BuildSnapshots(end, end.Add(12*time.Minute), startup, expected, received)

	require.Len(t, got, 1)
	assert.Equal(t, StatusComplete, got[0].Status)
	assert.Equal(t, 1.0, got[0].CoverageRatio)
}

func TestBuildSnapshots_SuppressesPreObserverSlotWithoutHistoricalMeasurementWindow(t *testing.T) {
	end := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)
	expected := []ExpectedGroup{{Technology: "lte", Carrier: "cmcc", Devices: 20000}}

	got := BuildSnapshots(end, end.Add(12*time.Minute), end.Add(5*time.Minute), expected, nil)

	require.Len(t, got, 1)
	assert.Equal(t, StatusBootstrapIgnored, got[0].Status)
}

func TestBuildSnapshots_FirstFullPostStartupSlotIsAlertEligible(t *testing.T) {
	startup := time.Date(2026, 8, 3, 15, 5, 0, 0, time.UTC)
	end := time.Date(2026, 8, 3, 15, 30, 0, 0, time.UTC)
	expected := []ExpectedGroup{{Technology: "lte", Carrier: "cmcc", Devices: 20000}}

	got := BuildSnapshots(end, end.Add(12*time.Minute), startup, expected, nil)

	require.Len(t, got, 1)
	assert.Equal(t, StatusMissing, got[0].Status)
}

func TestBuildSnapshots_DoesNotCreateUnboundedUnexpectedGroups(t *testing.T) {
	end := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)
	received := []ReceivedGroup{{Technology: "vendor-x", Carrier: "unknown", Devices: 10}}

	assert.Empty(t, BuildSnapshots(end, end.Add(12*time.Minute), time.Time{}, nil, received))
}

func TestObserver_QueriesOnlyLatestEligibleSlotAndPublishesPersistedSummary(t *testing.T) {
	now := time.Date(2026, 8, 3, 15, 12, 0, 0, time.UTC)
	store := &recordingStore{
		expected: []ExpectedGroup{{Technology: "lte", Carrier: "cmcc", Devices: 20000}},
		received: []ReceivedGroup{{Technology: "lte", Carrier: "cmcc", Devices: 19600}},
	}
	publisher := &recordingPublisher{}
	observer := NewObserver(store, publisher, now.Add(-time.Hour), 12*time.Minute, time.Minute, zap.NewNop())
	observer.now = func() time.Time { return now }

	err := observer.Observe(context.Background())

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 3, 14, 45, 0, 0, time.UTC), store.expectedAt)
	assert.Equal(t, time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC), store.receivedAt)
	require.Len(t, store.persisted, 1)
	require.Len(t, publisher.snapshots, 1)
	assert.Equal(t, store.persisted, publisher.snapshots)
}

func TestObserver_PublishesAuthoritativeStoredSnapshotAfterRestartConflict(t *testing.T) {
	now := time.Date(2026, 8, 3, 16, 42, 0, 0, time.UTC)
	complete := Snapshot{
		SlotStart: now.Add(-42 * time.Minute), SlotEnd: now.Add(-27 * time.Minute),
		Technology: "lte", Carrier: "cmcc", ExpectedDevices: 20000,
		ReceivedDevices: 20000, CoverageRatio: 1, Status: StatusComplete,
	}
	store := &recordingStore{
		expected:     []ExpectedGroup{{Technology: "lte", Carrier: "cmcc", Devices: 20000}},
		storedResult: []Snapshot{complete},
	}
	publisher := &recordingPublisher{}
	observer := NewObserver(store, publisher, now, 12*time.Minute, time.Minute, zap.NewNop())
	observer.now = func() time.Time { return now }

	err := observer.Observe(context.Background())

	require.NoError(t, err)
	require.Len(t, store.persisted, 1)
	assert.Equal(t, StatusBootstrapIgnored, store.persisted[0].Status)
	assert.Equal(t, []Snapshot{complete}, publisher.snapshots)
}

func TestObserver_DoesNotPublishWhenExpectedSnapshotFails(t *testing.T) {
	store := &recordingStore{expectedErr: errors.New("metadata unavailable")}
	publisher := &recordingPublisher{}
	observer := NewObserver(store, publisher, time.Time{}, 12*time.Minute, time.Minute, zap.NewNop())
	observer.now = func() time.Time {
		return time.Date(2026, 8, 3, 15, 12, 0, 0, time.UTC)
	}

	err := observer.Observe(context.Background())

	require.Error(t, err)
	assert.Empty(t, store.persisted)
	assert.Empty(t, publisher.snapshots)
}

func TestObserver_SkipsWhenAnotherObserverHoldsLock(t *testing.T) {
	store := &lockingRecordingStore{locked: false}
	publisher := &recordingPublisher{}
	observer := NewObserver(store, publisher, time.Time{}, 12*time.Minute, time.Minute, zap.NewNop())
	observer.now = func() time.Time {
		return time.Date(2026, 8, 3, 15, 12, 0, 0, time.UTC)
	}

	err := observer.Observe(context.Background())

	require.NoError(t, err)
	assert.True(t, store.lockChecked)
	assert.Zero(t, store.expectedAt)
	assert.Empty(t, store.persisted)
	assert.Empty(t, publisher.snapshots)
}

type recordingStore struct {
	expected     []ExpectedGroup
	received     []ReceivedGroup
	expectedErr  error
	expectedAt   time.Time
	receivedAt   time.Time
	persisted    []Snapshot
	storedResult []Snapshot
}

func (s *recordingStore) ExpectedGroups(_ context.Context, slotStart time.Time) ([]ExpectedGroup, error) {
	s.expectedAt = slotStart
	return s.expected, s.expectedErr
}

func (s *recordingStore) ReceivedGroups(_ context.Context, slotEnd time.Time) ([]ReceivedGroup, error) {
	s.receivedAt = slotEnd
	return s.received, nil
}

func (s *recordingStore) UpsertSnapshots(_ context.Context, snapshots []Snapshot) ([]Snapshot, error) {
	s.persisted = append([]Snapshot(nil), snapshots...)
	if s.storedResult != nil {
		return append([]Snapshot(nil), s.storedResult...), nil
	}
	return append([]Snapshot(nil), snapshots...), nil
}

type lockingRecordingStore struct {
	recordingStore
	locked      bool
	lockChecked bool
}

func (s *lockingRecordingStore) TryObserveLock(context.Context) (func() error, bool, error) {
	s.lockChecked = true
	return func() error { return nil }, s.locked, nil
}

type recordingPublisher struct {
	snapshots []Snapshot
}

func (p *recordingPublisher) PublishSlotHealth(snapshots []Snapshot) {
	p.snapshots = append([]Snapshot(nil), snapshots...)
}
