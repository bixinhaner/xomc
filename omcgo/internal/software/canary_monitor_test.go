package software

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeCanaryRepo is a hand-rolled mock satisfying canaryRepoView for monitor tests.
type fakeCanaryRepo struct {
	activeIDs   []uuid.UUID
	canary      map[uuid.UUID]*CanaryFields
	tasks       map[uuid.UUID]*UpgradeTask
	updatedWith map[uuid.UUID]*CanaryFields
}

func newFakeCanaryRepo() *fakeCanaryRepo {
	return &fakeCanaryRepo{
		canary:      map[uuid.UUID]*CanaryFields{},
		tasks:       map[uuid.UUID]*UpgradeTask{},
		updatedWith: map[uuid.UUID]*CanaryFields{},
	}
}

func (r *fakeCanaryRepo) ListActiveCanaryTaskIDs(_ context.Context) ([]uuid.UUID, error) {
	return r.activeIDs, nil
}
func (r *fakeCanaryRepo) GetCanaryFields(_ context.Context, id uuid.UUID) (*CanaryFields, error) {
	return r.canary[id], nil
}
func (r *fakeCanaryRepo) UpdateCanaryFields(_ context.Context, id uuid.UUID, fields *CanaryFields) error {
	cp := *fields
	r.updatedWith[id] = &cp
	r.canary[id] = &cp
	return nil
}
func (r *fakeCanaryRepo) GetByID(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
	return r.tasks[id], nil
}

func fixedNow(t time.Time) func() {
	prev := nowFunc
	nowFunc = func() time.Time { return t }
	return func() { nowFunc = prev }
}

func TestCanaryMonitor_NoActiveTasks(t *testing.T) {
	repo := newFakeCanaryRepo()
	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	require.NoError(t, m.CheckCanaryTasks(context.Background()))
}

func TestCanaryMonitor_ThresholdExceeded_Pauses(t *testing.T) {
	defer fixedNow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()

	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:       id,
		Strategy:     StrategyCanary,
		Stages:       DefaultCanaryStages,
		CurrentStage: 1,
		StageStatus:  StageStatusRunning,
		TotalCount:   100,
	}
	// stage 1 has 1 device (1%); 1 fail / 0 success → 100% > 5% threshold
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 1, SuccessCount: 0, TotalCount: 100}

	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	require.NoError(t, m.CheckCanaryTasks(context.Background()))

	updated := repo.updatedWith[id]
	require.NotNil(t, updated)
	assert.Equal(t, StageStatusPaused, updated.StageStatus)
	require.Len(t, updated.StageHistory, 1)
	assert.Equal(t, "paused", updated.StageHistory[0].Action)
	assert.Contains(t, updated.StageHistory[0].Reason, "threshold")
}

func TestCanaryMonitor_BelowThreshold_NoChange(t *testing.T) {
	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:       id,
		Strategy:     StrategyCanary,
		Stages:       DefaultCanaryStages,
		CurrentStage: 1,
		StageStatus:  StageStatusRunning,
		TotalCount:   100,
	}
	// 0 fails out of 1 success → 0% < 5%
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 0, SuccessCount: 1, TotalCount: 100}

	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	require.NoError(t, m.CheckCanaryTasks(context.Background()))
	assert.Nil(t, repo.updatedWith[id], "below threshold should not update without auto_advance")
}

func TestCanaryMonitor_AutoAdvance_PromotesStage(t *testing.T) {
	defer fixedNow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()

	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:       id,
		Strategy:     StrategyCanary,
		Stages:       DefaultCanaryStages,
		CurrentStage: 1,
		StageStatus:  StageStatusRunning,
		AutoAdvance:  true,
		TotalCount:   100,
	}
	// stage 1 expects 1 device; got 1 success / 0 fail → advance to stage 2
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 0, SuccessCount: 1, TotalCount: 100}

	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	require.NoError(t, m.CheckCanaryTasks(context.Background()))

	updated := repo.updatedWith[id]
	require.NotNil(t, updated)
	assert.Equal(t, 2, updated.CurrentStage, "should advance from stage 1 to 2")
}

func TestCanaryMonitor_AutoAdvance_AllStagesCompleted(t *testing.T) {
	defer fixedNow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()

	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:       id,
		Strategy:     StrategyCanary,
		Stages:       DefaultCanaryStages,
		CurrentStage: 4, // last stage (100%)
		StageStatus:  StageStatusRunning,
		AutoAdvance:  true,
		TotalCount:   100,
	}
	// all 100 devices completed
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 0, SuccessCount: 100, TotalCount: 100}

	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	require.NoError(t, m.CheckCanaryTasks(context.Background()))

	updated := repo.updatedWith[id]
	require.NotNil(t, updated)
	assert.Equal(t, StageStatusCompleted, updated.StageStatus)
}

func TestCanaryMonitor_PausedTaskNotAutoAdvanced(t *testing.T) {
	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:       id,
		Strategy:     StrategyCanary,
		Stages:       DefaultCanaryStages,
		CurrentStage: 1,
		StageStatus:  StageStatusPaused, // already paused
		AutoAdvance:  true,
		TotalCount:   100,
	}
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 0, SuccessCount: 1, TotalCount: 100}

	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	require.NoError(t, m.CheckCanaryTasks(context.Background()))
	assert.Nil(t, repo.updatedWith[id], "paused task must not auto-advance")
}

func TestCanaryMonitor_NonCanaryTaskSkipped(t *testing.T) {
	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:   id,
		Strategy: StrategyFull, // not canary
	}
	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	require.NoError(t, m.CheckCanaryTasks(context.Background()))
	assert.Nil(t, repo.updatedWith[id])
}

func TestCanaryMonitor_StartStop(t *testing.T) {
	repo := newFakeCanaryRepo()
	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	require.NoError(t, m.Start(context.Background()))
	m.Stop()
}

// ---------------------------------------------------------------------------
// T-0021 auto-rollback opt-in tests
// ---------------------------------------------------------------------------

// fakeRollbackTrigger captures TriggerCanaryFailureRollback invocations.
type fakeRollbackTrigger struct {
	calls    int
	lastID   uuid.UUID
	lastMsg  string
	failWith error
}

func (f *fakeRollbackTrigger) TriggerCanaryFailureRollback(_ context.Context, taskID uuid.UUID, reason string) error {
	f.calls++
	f.lastID = taskID
	f.lastMsg = reason
	return f.failWith
}

// TestCanaryMonitor_ThresholdExceeded_AutoRollback_OptIn — when
// RollbackOnFailure=true and trigger is wired, threshold breach also fires
// the rollback trigger (in addition to pause).
func TestCanaryMonitor_ThresholdExceeded_AutoRollback_OptIn(t *testing.T) {
	defer fixedNow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()

	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:            id,
		Strategy:          StrategyCanary,
		Stages:            DefaultCanaryStages,
		CurrentStage:      1,
		StageStatus:       StageStatusRunning,
		RollbackOnFailure: true,
		TotalCount:        100,
	}
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 1, SuccessCount: 0, TotalCount: 100}

	trigger := &fakeRollbackTrigger{}
	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	m.SetRollbackTrigger(trigger)
	require.NoError(t, m.CheckCanaryTasks(context.Background()))

	updated := repo.updatedWith[id]
	require.NotNil(t, updated)
	assert.Equal(t, StageStatusPaused, updated.StageStatus, "still pauses (rollback runs in addition to pause, not in place of)")
	assert.Equal(t, 1, trigger.calls, "auto-rollback trigger should fire exactly once")
	assert.Equal(t, id, trigger.lastID)
	assert.Contains(t, trigger.lastMsg, "threshold")
}

// TestCanaryMonitor_ThresholdExceeded_AutoRollback_OptOut — RollbackOnFailure=false
// (default) means trigger never fires; canary stays paused for manual review
// (preserves T-0018 D4 invariant).
func TestCanaryMonitor_ThresholdExceeded_AutoRollback_OptOut(t *testing.T) {
	defer fixedNow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()

	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:            id,
		Strategy:          StrategyCanary,
		Stages:            DefaultCanaryStages,
		CurrentStage:      1,
		StageStatus:       StageStatusRunning,
		RollbackOnFailure: false, // default; explicit for test clarity
		TotalCount:        100,
	}
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 1, SuccessCount: 0, TotalCount: 100}

	trigger := &fakeRollbackTrigger{}
	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	m.SetRollbackTrigger(trigger)
	require.NoError(t, m.CheckCanaryTasks(context.Background()))

	assert.Equal(t, StageStatusPaused, repo.updatedWith[id].StageStatus)
	assert.Zero(t, trigger.calls, "opt-out: trigger must not fire")
}

// TestCanaryMonitor_ThresholdExceeded_AutoRollback_NoTrigger — even if
// RollbackOnFailure=true, a nil trigger silently skips (DI not yet wired).
func TestCanaryMonitor_ThresholdExceeded_AutoRollback_NoTrigger(t *testing.T) {
	defer fixedNow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()

	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:            id,
		Strategy:          StrategyCanary,
		Stages:            DefaultCanaryStages,
		CurrentStage:      1,
		StageStatus:       StageStatusRunning,
		RollbackOnFailure: true,
		TotalCount:        100,
	}
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 1, SuccessCount: 0, TotalCount: 100}

	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	// no SetRollbackTrigger
	require.NoError(t, m.CheckCanaryTasks(context.Background()))
	assert.Equal(t, StageStatusPaused, repo.updatedWith[id].StageStatus, "pause still happens; rollback gracefully skipped")
}

// TestCanaryMonitor_ThresholdExceeded_AutoRollback_TriggerError — pause is
// already persisted; trigger error must not propagate (operators recover manually).
func TestCanaryMonitor_ThresholdExceeded_AutoRollback_TriggerError(t *testing.T) {
	defer fixedNow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()

	id := uuid.New()
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:            id,
		Strategy:          StrategyCanary,
		Stages:            DefaultCanaryStages,
		CurrentStage:      1,
		StageStatus:       StageStatusRunning,
		RollbackOnFailure: true,
		TotalCount:        100,
	}
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 1, SuccessCount: 0, TotalCount: 100}

	trigger := &fakeRollbackTrigger{failWith: assertExpectedTriggerError()}
	m := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	m.SetRollbackTrigger(trigger)

	// Whole CheckCanaryTasks loop must not surface the trigger error: pause is
	// already saved and the error is logged + swallowed by design.
	require.NoError(t, m.CheckCanaryTasks(context.Background()))
	assert.Equal(t, 1, trigger.calls)
	assert.Equal(t, StageStatusPaused, repo.updatedWith[id].StageStatus)
}

// assertExpectedTriggerError returns an error to simulate a downstream failure.
func assertExpectedTriggerError() error {
	return errStub("simulated trigger failure")
}

type errStub string

func (e errStub) Error() string { return string(e) }
