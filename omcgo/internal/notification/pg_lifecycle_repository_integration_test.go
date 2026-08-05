package notification

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

func TestPgLifecycleRepository_Integration(t *testing.T) {
	dsn := os.Getenv("NOTIFICATION_LIFECYCLE_TEST_DSN")
	if dsn == "" {
		t.Skip("set NOTIFICATION_LIFECYCLE_TEST_DSN to a dedicated database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var databaseName string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName))
	require.Equal(t, "omcgo_notification_lifecycle", databaseName,
		"refusing to run lifecycle integration test against a non-dedicated database")

	repository := NewPgLifecycleRepository(pool)
	now := time.Date(2026, 8, 5, 12, 30, 0, 0, time.UTC)
	alarm := model.Alarm{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "LAB-LIFECYCLE", Carrier: model.CarrierCMCC,
		Severity: model.AlarmMajor, AlarmType: "communications", AlarmIdentifier: "CELL_UNAVAILABLE",
		Description: "cell unavailable", Status: model.AlarmActive, RaisedAt: now,
	}

	raised := lifecyclePayload(t, alarm, event.AlarmLifecycleRaised, model.AlarmActive, 1, now)
	result, err := repository.ApplyLifecycle(ctx, raised)
	require.NoError(t, err)
	require.Equal(t, LifecycleApplyResult{State: LifecycleApplied, AppliedCount: 1}, result)
	result, err = repository.ApplyLifecycle(ctx, raised)
	require.NoError(t, err)
	require.Equal(t, LifecycleDuplicate, result.State)

	acknowledged := lifecyclePayload(t, alarm, event.AlarmLifecycleAcknowledged, model.AlarmAcknowledged, 3, now.Add(2*time.Minute))
	result, err = repository.ApplyLifecycle(ctx, acknowledged)
	require.NoError(t, err)
	require.Equal(t, LifecycleWaiting, result.State)
	occurrence, err := NewPgInboxRepository(pool).GetOccurrence(ctx, alarm.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), occurrence.LastAppliedVersion)
	require.True(t, occurrence.VersionGap)

	updated := lifecyclePayload(t, alarm, event.AlarmLifecycleUpdated, model.AlarmActive, 2, now.Add(time.Minute))
	result, err = repository.ApplyLifecycle(ctx, updated)
	require.NoError(t, err)
	require.Equal(t, LifecycleApplied, result.State)
	require.Equal(t, 2, result.AppliedCount)
	occurrence, err = NewPgInboxRepository(pool).GetOccurrence(ctx, alarm.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), occurrence.LastAppliedVersion)
	require.Equal(t, int64(2), occurrence.ScheduleGeneration)
	require.Equal(t, "acknowledged", occurrence.Status)
	require.False(t, occurrence.VersionGap)
	var acknowledgedState string
	require.NoError(t, pool.QueryRow(ctx, `SELECT processing_state FROM notification_events WHERE event_id=$1`, acknowledged.EventID).Scan(&acknowledgedState))
	require.Equal(t, "applied", acknowledgedState)

	stale := lifecyclePayload(t, alarm, event.AlarmLifecycleUpdated, model.AlarmActive, 2, now.Add(3*time.Minute))
	result, err = repository.ApplyLifecycle(ctx, stale)
	require.NoError(t, err)
	require.Equal(t, LifecycleIgnored, result.State)

	ruleID, ruleVersionID := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_rules (id,name,created_by) VALUES ($1,$2,'integration')`,
		ruleID, "lifecycle-"+ruleID.String())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notification_rule_versions (id,rule_id,version_no,created_by,published_at) VALUES ($1,$2,1,'integration',$3)`,
		ruleVersionID, ruleID, now)
	require.NoError(t, err)
	schedule := &DomainSchedule{
		OccurrenceID: alarm.ID, RuleVersionID: ruleVersionID, Channel: "email",
		RecipientFingerprint: []byte("recipient"), ScheduleKind: ScheduleKindRepeat,
		Generation: 2, DueAt: now.Add(time.Hour), CreatedEventVersion: 3,
	}
	inserted, err := NewPgScheduleRepository(pool).InsertSchedule(ctx, schedule)
	require.NoError(t, err)
	require.True(t, inserted)
	initialSchedule := &DomainSchedule{
		OccurrenceID: alarm.ID, RuleVersionID: ruleVersionID, Channel: "email",
		RecipientFingerprint: []byte("recipient"), ScheduleKind: ScheduleKindInitialGate,
		Generation: 2, DueAt: now.Add(time.Hour), CreatedEventVersion: 3,
	}
	inserted, err = NewPgScheduleRepository(pool).InsertSchedule(ctx, initialSchedule)
	require.NoError(t, err)
	require.True(t, inserted)

	unacknowledged := lifecyclePayload(t, alarm, event.AlarmLifecycleUnacknowledged, model.AlarmActive, 4, now.Add(4*time.Minute))
	result, err = repository.ApplyLifecycle(ctx, unacknowledged)
	require.NoError(t, err)
	require.Equal(t, LifecycleApplied, result.State)
	occurrence, err = NewPgInboxRepository(pool).GetOccurrence(ctx, alarm.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), occurrence.ScheduleGeneration)
	var scheduleState string
	require.NoError(t, pool.QueryRow(ctx, `SELECT state FROM notification_schedules WHERE id=$1`, schedule.ID).Scan(&scheduleState))
	require.Equal(t, "cancelled", scheduleState)
	var pendingRepeatCount, pendingInitialCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM notification_schedules WHERE occurrence_id=$1 AND state='pending' AND schedule_kind='repeat' AND generation=3`, alarm.ID).Scan(&pendingRepeatCount))
	require.Equal(t, 1, pendingRepeatCount, "unacknowledge must resume the existing future repeat")
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM notification_schedules WHERE occurrence_id=$1 AND state='pending' AND schedule_kind='initial_gate'`, alarm.ID).Scan(&pendingInitialCount))
	require.Zero(t, pendingInitialCount, "unacknowledge must not recreate the initial notification")

	cleared := lifecyclePayload(t, alarm, event.AlarmLifecycleCleared, model.AlarmCleared, 5, now.Add(5*time.Minute))
	result, err = repository.ApplyLifecycle(ctx, cleared)
	require.NoError(t, err)
	require.Equal(t, LifecycleApplied, result.State)
	occurrence, err = NewPgInboxRepository(pool).GetOccurrence(ctx, alarm.ID)
	require.NoError(t, err)
	require.Equal(t, int64(5), occurrence.LastAppliedVersion)
	require.Equal(t, int64(4), occurrence.ScheduleGeneration)
	require.Equal(t, "cleared", occurrence.Status)

	claimable := &DomainSchedule{
		OccurrenceID: alarm.ID, RuleVersionID: ruleVersionID, Channel: "email",
		RecipientFingerprint: []byte("digest-recipient"), ScheduleKind: ScheduleKindDigestFlush,
		Generation: 4, DueAt: now, CreatedEventVersion: 5,
	}
	schedules := NewPgScheduleRepository(pool)
	inserted, err = schedules.InsertSchedule(ctx, claimable)
	require.NoError(t, err)
	require.True(t, inserted)
	claimAt := now.Add(10 * time.Minute)
	claimed, err := schedules.ClaimDue(ctx, ScheduleClaimRequest{
		WorkerID: "worker-a", Now: claimAt, LeaseDuration: 30 * time.Second, Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, claimable.ID, claimed[0].ID)
	claimed, err = NewPgScheduleRepository(pool).ClaimDue(ctx, ScheduleClaimRequest{
		WorkerID: "worker-b", Now: claimAt, LeaseDuration: 30 * time.Second, Limit: 10,
	})
	require.NoError(t, err)
	require.Empty(t, claimed, "a live lease must not be claimed by another instance")
	claimed, err = NewPgScheduleRepository(pool).ClaimDue(ctx, ScheduleClaimRequest{
		WorkerID: "worker-b", Now: claimAt.Add(31 * time.Second), LeaseDuration: 30 * time.Second, Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, claimed, 1, "an expired lease must be recoverable")
	err = schedules.MarkCompleted(ctx, claimable.ID, "worker-a", claimAt.Add(31*time.Second))
	require.ErrorIs(t, err, ErrScheduleLeaseLost)
	require.NoError(t, schedules.MarkCompleted(ctx, claimable.ID, "worker-b", claimAt.Add(31*time.Second)))
}

func lifecyclePayload(
	t *testing.T,
	base model.Alarm,
	lifecycleType event.AlarmLifecycleType,
	status model.AlarmStatus,
	version int64,
	occurredAt time.Time,
) event.AlarmLifecyclePayload {
	t.Helper()
	base.Status, base.LastUpdatedAt = status, occurredAt
	if status == model.AlarmAcknowledged {
		base.AcknowledgedAt = &occurredAt
	}
	if status == model.AlarmCleared {
		base.ClearedAt = &occurredAt
	}
	payload, err := event.NewAlarmLifecyclePayload(lifecycleType, base, nil, version, occurredAt, nil)
	require.NoError(t, err)
	return payload
}
