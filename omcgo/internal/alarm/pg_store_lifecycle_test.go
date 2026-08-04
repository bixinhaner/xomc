package alarm

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestLifecycleTx_RaisedCommitsActiveRowAndOutbox(t *testing.T) {
	mutatedAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	tx := &fakeLifecycleTx{execResults: []fakeExecResult{{rows: 1}, {rows: 1}}}
	store := newPgAlarmStoreWithDB(&fakeLifecycleDB{tx: tx}, nil, func() time.Time { return mutatedAt })
	alarm := validLifecycleAlarm()

	payload, err := store.PersistRaised(context.Background(), alarm)
	require.NoError(t, err)
	require.True(t, tx.committed)
	require.True(t, tx.rolledBack, "deferred rollback remains safe after commit")
	require.Len(t, tx.execSQL, 2)
	require.Contains(t, tx.execSQL[0], "INSERT INTO alarms_active")
	require.Contains(t, tx.execSQL[1], "INSERT INTO alarm_event_outbox")
	require.Equal(t, int64(1), alarm.Version)
	require.Equal(t, int64(1), payload.AlarmVersion)
	require.Equal(t, mutatedAt, payload.OccurredAt)
	require.Equal(t, event.AlarmLifecycleRaised, payload.LifecycleType)
}

func TestOutboxRollback_RaisedOutboxFailureDoesNotCommitOrMutateCaller(t *testing.T) {
	outboxErr := errors.New("outbox unavailable")
	tx := &fakeLifecycleTx{execResults: []fakeExecResult{{rows: 1}, {err: outboxErr}}}
	store := newPgAlarmStoreWithDB(&fakeLifecycleDB{tx: tx}, nil, time.Now)
	alarm := validLifecycleAlarm()

	_, err := store.PersistRaised(context.Background(), alarm)
	require.ErrorIs(t, err, outboxErr)
	require.False(t, tx.committed)
	require.True(t, tx.rolledBack)
	require.Zero(t, alarm.Version)
}

func TestOutboxRollback_UpdatedOutboxFailureDoesNotMutateCaller(t *testing.T) {
	outboxErr := errors.New("outbox unavailable")
	current := validLifecycleAlarm()
	current.Version = 7
	desired := *current
	desired.Description = "updated description"
	tx := &fakeLifecycleTx{
		row:         lifecycleAlarmRow{alarm: current},
		execResults: []fakeExecResult{{rows: 1}, {err: outboxErr}},
	}
	store := newPgAlarmStoreWithDB(&fakeLifecycleDB{tx: tx}, nil, time.Now)

	_, err := store.PersistUpdated(
		context.Background(),
		&desired,
		[]event.AlarmChangeField{event.AlarmChangeDescription},
	)
	require.ErrorIs(t, err, outboxErr)
	require.False(t, tx.committed)
	require.True(t, tx.rolledBack)
	require.Equal(t, int64(7), desired.Version)
}

func TestLifecycleTx_CommitFailureDoesNotMutateCaller(t *testing.T) {
	commitErr := errors.New("commit failed")
	tx := &fakeLifecycleTx{
		execResults: []fakeExecResult{{rows: 1}, {rows: 1}},
		commitErr:   commitErr,
	}
	store := newPgAlarmStoreWithDB(&fakeLifecycleDB{tx: tx}, nil, time.Now)
	alarm := validLifecycleAlarm()

	_, err := store.PersistRaised(context.Background(), alarm)
	require.ErrorIs(t, err, commitErr)
	require.True(t, tx.committed)
	require.True(t, tx.rolledBack)
	require.Zero(t, alarm.Version)
}

func TestLifecycleTx_UpdatedAllocatesVersionFromLockedRow(t *testing.T) {
	mutatedAt := time.Date(2026, 8, 4, 12, 1, 0, 0, time.UTC)
	current := validLifecycleAlarm()
	current.Version = 4
	current.Severity = model.AlarmMajor
	desired := *current
	desired.Severity = model.AlarmCritical
	tx := &fakeLifecycleTx{
		row:         lifecycleAlarmRow{alarm: current},
		execResults: []fakeExecResult{{rows: 1}, {rows: 1}},
	}
	store := newPgAlarmStoreWithDB(&fakeLifecycleDB{tx: tx}, nil, func() time.Time { return mutatedAt })

	payload, err := store.PersistUpdated(
		context.Background(),
		&desired,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, int64(5), desired.Version)
	require.Equal(t, int64(5), payload.AlarmVersion)
	require.Equal(t, model.AlarmMajor, *payload.PreviousSeverity)
	require.Contains(t, payload.ChangeMask, event.AlarmChangeSeverity)
	require.Equal(t, mutatedAt, payload.OccurredAt)
	require.Contains(t, tx.querySQL, "FOR UPDATE OF alarms_active")
	require.Contains(t, tx.execSQL[0], "UPDATE alarms_active")
	require.Contains(t, tx.execSQL[1], "INSERT INTO alarm_event_outbox")
}

func TestLifecycleTx_RejectsStaleExpectedVersionBeforeWriting(t *testing.T) {
	current := validLifecycleAlarm()
	current.Version = 3
	desired := *current
	desired.Version = 2
	tx := &fakeLifecycleTx{row: lifecycleAlarmRow{alarm: current}}
	store := newPgAlarmStoreWithDB(&fakeLifecycleDB{tx: tx}, nil, time.Now)

	_, err := store.PersistAcknowledged(context.Background(), &desired)
	require.ErrorIs(t, err, ErrAlarmVersionConflict)
	require.Empty(t, tx.execSQL)
	require.False(t, tx.committed)
	require.True(t, tx.rolledBack)
}

func TestLifecycleTx_ClearedDeletesActiveAndWritesCompleteSnapshot(t *testing.T) {
	current := validLifecycleAlarm()
	current.Version = 2
	desired := *current
	desired.Status = model.AlarmCleared
	clearedAt := time.Now().UTC()
	desired.ClearedAt = &clearedAt
	tx := &fakeLifecycleTx{
		row:         lifecycleAlarmRow{alarm: current},
		execResults: []fakeExecResult{{rows: 1}, {rows: 1}},
	}
	store := newPgAlarmStoreWithDB(&fakeLifecycleDB{tx: tx}, nil, time.Now)

	payload, err := store.PersistCleared(context.Background(), &desired)
	require.NoError(t, err)
	require.Equal(t, int64(3), payload.AlarmVersion)
	require.Equal(t, model.AlarmCleared, payload.Snapshot.Status)
	require.NotNil(t, payload.Snapshot.ClearedAt)
	require.Contains(t, tx.execSQL[0], "DELETE FROM alarms_active")
	require.Contains(t, tx.execSQL[1], "INSERT INTO alarm_event_outbox")
}

func TestLifecycleTx_NewAutoClearWritesRaisedAndClearedInOneTransaction(t *testing.T) {
	mutatedAt := time.Date(2026, 8, 4, 12, 2, 0, 0, time.UTC)
	tx := &fakeLifecycleTx{
		execResults: []fakeExecResult{{rows: 1}, {rows: 1}, {rows: 1}, {rows: 1}},
	}
	store := newPgAlarmStoreWithDB(&fakeLifecycleDB{tx: tx}, nil, func() time.Time { return mutatedAt })
	alarm := validLifecycleAlarm()
	alarm.Status = model.AlarmCleared
	alarm.ClearedAt = &mutatedAt
	clearedBy := "system"
	alarm.ClearedBy = &clearedBy

	payloads, err := store.PersistRaisedAndCleared(context.Background(), alarm)
	require.NoError(t, err)
	require.True(t, tx.committed)
	require.Equal(t, int64(2), alarm.Version)
	require.Len(t, payloads, 2)
	require.Equal(t, event.AlarmLifecycleRaised, payloads[0].LifecycleType)
	require.Equal(t, int64(1), payloads[0].AlarmVersion)
	require.Equal(t, model.AlarmActive, payloads[0].Snapshot.Status)
	require.Nil(t, payloads[0].Snapshot.ClearedAt)
	require.Equal(t, event.AlarmLifecycleCleared, payloads[1].LifecycleType)
	require.Equal(t, int64(2), payloads[1].AlarmVersion)
	require.Equal(t, model.AlarmCleared, payloads[1].Snapshot.Status)
	require.NotNil(t, payloads[1].Snapshot.ClearedAt)
	require.Len(t, tx.execSQL, 4)
	require.Contains(t, tx.execSQL[0], "INSERT INTO alarms_active")
	require.Contains(t, tx.execSQL[1], "INSERT INTO alarm_event_outbox")
	require.Contains(t, tx.execSQL[2], "DELETE FROM alarms_active")
	require.Contains(t, tx.execSQL[3], "INSERT INTO alarm_event_outbox")
}

func TestLifecycleTx_NewAutoAcknowledgeWritesRaisedAndAcknowledgedInOneTransaction(t *testing.T) {
	mutatedAt := time.Date(2026, 8, 4, 12, 3, 0, 0, time.UTC)
	tx := &fakeLifecycleTx{
		execResults: []fakeExecResult{{rows: 1}, {rows: 1}, {rows: 1}, {rows: 1}},
	}
	store := newPgAlarmStoreWithDB(&fakeLifecycleDB{tx: tx}, nil, func() time.Time { return mutatedAt })
	alarm := validLifecycleAlarm()
	alarm.Status = model.AlarmAcknowledged
	alarm.AcknowledgedAt = &mutatedAt
	acknowledgedBy := "system:auto_filter"
	alarm.AcknowledgedBy = &acknowledgedBy

	payloads, err := store.PersistRaisedAndAcknowledged(context.Background(), alarm)
	require.NoError(t, err)
	require.True(t, tx.committed)
	require.Equal(t, int64(2), alarm.Version)
	require.Len(t, payloads, 2)
	require.Equal(t, event.AlarmLifecycleRaised, payloads[0].LifecycleType)
	require.Equal(t, int64(1), payloads[0].AlarmVersion)
	require.Equal(t, model.AlarmActive, payloads[0].Snapshot.Status)
	require.Nil(t, payloads[0].Snapshot.AcknowledgedAt)
	require.Equal(t, event.AlarmLifecycleAcknowledged, payloads[1].LifecycleType)
	require.Equal(t, int64(2), payloads[1].AlarmVersion)
	require.Equal(t, model.AlarmAcknowledged, payloads[1].Snapshot.Status)
	require.NotNil(t, payloads[1].Snapshot.AcknowledgedAt)
	require.Len(t, tx.execSQL, 4)
	require.Contains(t, tx.execSQL[0], "INSERT INTO alarms_active")
	require.Contains(t, tx.execSQL[1], "INSERT INTO alarm_event_outbox")
	require.Contains(t, tx.execSQL[2], "UPDATE alarms_active")
	require.Contains(t, tx.execSQL[3], "INSERT INTO alarm_event_outbox")
}

func validLifecycleAlarm() *model.Alarm {
	now := time.Now().UTC()
	return &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-LIFECYCLE-1",
		Carrier:         model.CarrierCMCC,
		Severity:        model.AlarmMajor,
		AlarmType:       "equipment",
		AlarmIdentifier: "CELL_UNAVAILABLE",
		Status:          model.AlarmActive,
		RaisedAt:        now,
		FirstRaisedAt:   now,
		LastUpdatedAt:   now,
		CreatedAt:       now,
		UpdatedAt:       now,
		AckCount:        1,
	}
}

type fakeLifecycleDB struct {
	tx  pgx.Tx
	err error
}

func (d *fakeLifecycleDB) Begin(context.Context) (pgx.Tx, error) { return d.tx, d.err }
func (d *fakeLifecycleDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected DB Exec")
}
func (d *fakeLifecycleDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected DB Query")
}
func (d *fakeLifecycleDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return errorLifecycleRow{err: errors.New("unexpected DB QueryRow")}
}

type fakeExecResult struct {
	rows int64
	err  error
}

type fakeLifecycleTx struct {
	row         pgx.Row
	execResults []fakeExecResult
	execSQL     []string
	querySQL    string
	committed   bool
	rolledBack  bool
	commitErr   error
}

func (tx *fakeLifecycleTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("unexpected nested tx")
}
func (tx *fakeLifecycleTx) Commit(context.Context) error {
	tx.committed = true
	return tx.commitErr
}
func (tx *fakeLifecycleTx) Rollback(context.Context) error {
	tx.rolledBack = true
	return nil
}
func (tx *fakeLifecycleTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	tx.execSQL = append(tx.execSQL, sql)
	index := len(tx.execSQL) - 1
	if index >= len(tx.execResults) {
		return pgconn.CommandTag{}, errors.New("unexpected transaction Exec")
	}
	result := tx.execResults[index]
	if result.err != nil {
		return pgconn.CommandTag{}, result.err
	}
	if result.rows == 0 {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (tx *fakeLifecycleTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected transaction Query")
}
func (tx *fakeLifecycleTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	tx.querySQL = sql
	return tx.row
}
func (tx *fakeLifecycleTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("unexpected CopyFrom")
}
func (tx *fakeLifecycleTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *fakeLifecycleTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *fakeLifecycleTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("unexpected Prepare")
}
func (tx *fakeLifecycleTx) Conn() *pgx.Conn { return nil }

type lifecycleAlarmRow struct {
	alarm *model.Alarm
}

func (r lifecycleAlarmRow) Scan(dest ...any) error {
	if r.alarm == nil {
		return pgx.ErrNoRows
	}
	values := map[string]any{
		"alarms_active.id":               r.alarm.ID,
		"alarms_active.device_id":        r.alarm.DeviceID,
		"alarms_active.device_sn":        r.alarm.DeviceSN,
		"alarms_active.carrier":          r.alarm.Carrier,
		"alarms_active.severity":         r.alarm.Severity,
		"alarms_active.alarm_type":       r.alarm.AlarmType,
		"alarms_active.alarm_identifier": r.alarm.AlarmIdentifier,
		"alarms_active.status":           r.alarm.Status,
		"alarms_active.raised_at":        r.alarm.RaisedAt,
		"alarms_active.ack_count":        r.alarm.AckCount,
		"alarms_active.first_raised_at":  r.alarm.FirstRaisedAt,
		"alarms_active.last_updated_at":  r.alarm.LastUpdatedAt,
		"alarms_active.created_at":       r.alarm.CreatedAt,
		"alarms_active.updated_at":       r.alarm.UpdatedAt,
		"alarms_active.alarm_version":    r.alarm.Version,
	}
	for index, column := range activeColumns {
		value, ok := values[column]
		if !ok {
			continue
		}
		target := reflect.ValueOf(dest[index])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return errors.New("scan destination is not a pointer")
		}
		target.Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

type errorLifecycleRow struct{ err error }

func (r errorLifecycleRow) Scan(...any) error { return r.err }
