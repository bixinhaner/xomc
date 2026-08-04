package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

func (s *PgAlarmStore) PersistRaised(
	ctx context.Context,
	alarm *model.Alarm,
) (event.AlarmLifecyclePayload, error) {
	if alarm == nil {
		return event.AlarmLifecyclePayload{}, fmt.Errorf("persist raised alarm: alarm is required")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return event.AlarmLifecyclePayload{}, fmt.Errorf("begin raised alarm lifecycle tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	persisted := *alarm
	mutatedAt := s.lifecycleNow()
	persisted.Version = 1
	if persisted.Status == "" {
		persisted.Status = model.AlarmActive
	}
	if persisted.CreatedAt.IsZero() {
		persisted.CreatedAt = mutatedAt
	}
	persisted.UpdatedAt = mutatedAt
	normalizeAlarmOccurrenceFields(&persisted, mutatedAt)

	payload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleRaised,
		persisted,
		nil,
		persisted.Version,
		mutatedAt,
		nil,
	)
	if err != nil {
		return event.AlarmLifecyclePayload{}, fmt.Errorf("build raised alarm lifecycle payload: %w", err)
	}
	if err := insertActiveLifecycle(ctx, tx, &persisted); err != nil {
		return event.AlarmLifecyclePayload{}, err
	}
	if err := insertAlarmOutbox(ctx, tx, payload); err != nil {
		return event.AlarmLifecyclePayload{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return event.AlarmLifecyclePayload{}, fmt.Errorf("commit raised alarm lifecycle tx: %w", err)
	}
	*alarm = persisted
	return payload, nil
}

func (s *PgAlarmStore) PersistUpdated(
	ctx context.Context,
	alarm *model.Alarm,
	changeMask []event.AlarmChangeField,
) (event.AlarmLifecyclePayload, error) {
	return s.persistExistingLifecycle(ctx, event.AlarmLifecycleUpdated, alarm, changeMask, false)
}

func (s *PgAlarmStore) PersistAcknowledged(
	ctx context.Context,
	alarm *model.Alarm,
) (event.AlarmLifecyclePayload, error) {
	return s.persistExistingLifecycle(ctx, event.AlarmLifecycleAcknowledged, alarm, nil, false)
}

func (s *PgAlarmStore) PersistUnacknowledged(
	ctx context.Context,
	alarm *model.Alarm,
) (event.AlarmLifecyclePayload, error) {
	return s.persistExistingLifecycle(ctx, event.AlarmLifecycleUnacknowledged, alarm, nil, false)
}

func (s *PgAlarmStore) PersistCleared(
	ctx context.Context,
	alarm *model.Alarm,
) (event.AlarmLifecyclePayload, error) {
	return s.persistExistingLifecycle(ctx, event.AlarmLifecycleCleared, alarm, nil, true)
}

// PersistRaisedAndAcknowledged records a newly observed occurrence that an
// automatic rule acknowledges immediately. Both ordered facts and the final
// acknowledged active row commit together.
func (s *PgAlarmStore) PersistRaisedAndAcknowledged(
	ctx context.Context,
	alarm *model.Alarm,
) ([]event.AlarmLifecyclePayload, error) {
	if alarm == nil {
		return nil, fmt.Errorf("persist raised and acknowledged alarm: alarm is required")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin raised and acknowledged alarm lifecycle tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	mutatedAt := s.lifecycleNow()
	raised := *alarm
	raised.Version = 1
	raised.Status = model.AlarmActive
	raised.AcknowledgedAt = nil
	raised.AcknowledgedBy = nil
	raised.AckNote = nil
	if raised.CreatedAt.IsZero() {
		raised.CreatedAt = mutatedAt
	}
	raised.UpdatedAt = mutatedAt
	normalizeAlarmOccurrenceFields(&raised, mutatedAt)

	raisedPayload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleRaised,
		raised,
		nil,
		raised.Version,
		mutatedAt,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build auto-acknowledged raised lifecycle payload: %w", err)
	}

	acknowledged := raised
	acknowledged.Version = 2
	acknowledged.Status = model.AlarmAcknowledged
	acknowledged.AcknowledgedAt = alarm.AcknowledgedAt
	if acknowledged.AcknowledgedAt == nil {
		acknowledged.AcknowledgedAt = &mutatedAt
	}
	acknowledged.AcknowledgedBy = alarm.AcknowledgedBy
	acknowledged.AckNote = alarm.AckNote
	acknowledgedPayload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleAcknowledged,
		acknowledged,
		&raised,
		acknowledged.Version,
		mutatedAt,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build auto-acknowledged lifecycle payload: %w", err)
	}

	if err := insertActiveLifecycle(ctx, tx, &raised); err != nil {
		return nil, err
	}
	if err := insertAlarmOutbox(ctx, tx, raisedPayload); err != nil {
		return nil, err
	}
	if err := updateActiveLifecycle(ctx, tx, &acknowledged, raised.Version); err != nil {
		return nil, err
	}
	if err := insertAlarmOutbox(ctx, tx, acknowledgedPayload); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit raised and acknowledged alarm lifecycle tx: %w", err)
	}
	*alarm = acknowledged
	return []event.AlarmLifecyclePayload{raisedPayload, acknowledgedPayload}, nil
}

// PersistRaisedAndCleared records a newly observed alarm that an auto-clear
// rule suppresses immediately. The occurrence must still have an ordered
// raised v1 -> cleared v2 lifecycle even though no active row remains.
func (s *PgAlarmStore) PersistRaisedAndCleared(
	ctx context.Context,
	alarm *model.Alarm,
) ([]event.AlarmLifecyclePayload, error) {
	if alarm == nil {
		return nil, fmt.Errorf("persist raised and cleared alarm: alarm is required")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin raised and cleared alarm lifecycle tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	mutatedAt := s.lifecycleNow()
	raised := *alarm
	raised.Version = 1
	raised.Status = model.AlarmActive
	raised.ClearedAt = nil
	raised.ClearedBy = nil
	raised.ClearNote = nil
	if raised.CreatedAt.IsZero() {
		raised.CreatedAt = mutatedAt
	}
	raised.UpdatedAt = mutatedAt
	normalizeAlarmOccurrenceFields(&raised, mutatedAt)

	raisedPayload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleRaised,
		raised,
		nil,
		raised.Version,
		mutatedAt,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build auto-cleared raised lifecycle payload: %w", err)
	}

	cleared := raised
	cleared.Version = 2
	cleared.Status = model.AlarmCleared
	cleared.ClearedAt = alarm.ClearedAt
	if cleared.ClearedAt == nil {
		cleared.ClearedAt = &mutatedAt
	}
	cleared.ClearedBy = alarm.ClearedBy
	cleared.ClearNote = alarm.ClearNote
	clearedPayload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleCleared,
		cleared,
		&raised,
		cleared.Version,
		mutatedAt,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build auto-cleared cleared lifecycle payload: %w", err)
	}

	if err := insertActiveLifecycle(ctx, tx, &raised); err != nil {
		return nil, err
	}
	if err := insertAlarmOutbox(ctx, tx, raisedPayload); err != nil {
		return nil, err
	}
	if err := deleteActiveLifecycle(ctx, tx, raised.ID, raised.Version); err != nil {
		return nil, err
	}
	if err := insertAlarmOutbox(ctx, tx, clearedPayload); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit raised and cleared alarm lifecycle tx: %w", err)
	}
	*alarm = cleared
	return []event.AlarmLifecyclePayload{raisedPayload, clearedPayload}, nil
}

func (s *PgAlarmStore) persistExistingLifecycle(
	ctx context.Context,
	lifecycleType event.AlarmLifecycleType,
	desired *model.Alarm,
	changeMask []event.AlarmChangeField,
	deleteActive bool,
) (event.AlarmLifecyclePayload, error) {
	if desired == nil {
		return event.AlarmLifecyclePayload{}, fmt.Errorf("persist %s alarm: alarm is required", lifecycleType)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return event.AlarmLifecyclePayload{}, fmt.Errorf("begin %s alarm lifecycle tx: %w", lifecycleType, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := loadActiveForUpdate(ctx, tx, desired.ID)
	if err != nil {
		return event.AlarmLifecyclePayload{}, fmt.Errorf("lock %s alarm occurrence: %w", lifecycleType, err)
	}
	if current.Version != desired.Version {
		return event.AlarmLifecyclePayload{}, fmt.Errorf(
			"%w: occurrence %s expected version %d, current version %d",
			ErrAlarmVersionConflict,
			desired.ID,
			desired.Version,
			current.Version,
		)
	}

	persisted := *desired
	mutatedAt := s.lifecycleNow()
	persisted.Version = current.Version + 1
	persisted.UpdatedAt = mutatedAt
	if lifecycleType == event.AlarmLifecycleUpdated {
		// The locked database snapshot is authoritative. Some callers (notably
		// alarm sync) receive an object whose mutable fields were already merged
		// before entering AlarmEngine, so an engine-side diff alone can miss a
		// severity transition and its required previous_severity.
		changeMask = alarmChangeMask(current, &persisted)
	}
	payload, err := event.NewAlarmLifecyclePayload(
		lifecycleType,
		persisted,
		current,
		persisted.Version,
		mutatedAt,
		changeMask,
	)
	if err != nil {
		return event.AlarmLifecyclePayload{}, fmt.Errorf("build %s alarm lifecycle payload: %w", lifecycleType, err)
	}

	if deleteActive {
		if err := deleteActiveLifecycle(ctx, tx, current.ID, current.Version); err != nil {
			return event.AlarmLifecyclePayload{}, err
		}
	} else if err := updateActiveLifecycle(ctx, tx, &persisted, current.Version); err != nil {
		return event.AlarmLifecyclePayload{}, err
	}
	if err := insertAlarmOutbox(ctx, tx, payload); err != nil {
		return event.AlarmLifecyclePayload{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return event.AlarmLifecyclePayload{}, fmt.Errorf("commit %s alarm lifecycle tx: %w", lifecycleType, err)
	}
	*desired = persisted
	return payload, nil
}

func (s *PgAlarmStore) lifecycleNow() time.Time {
	if s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func loadActiveForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Alarm, error) {
	query, args, err := activeAlarmSelect().
		Where(squirrel.Eq{"alarms_active.id": id}).
		Suffix("FOR UPDATE OF alarms_active").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build active alarm lock query: %w", err)
	}
	alarm, err := scanAlarmRow(tx.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return alarm, nil
}

func insertActiveLifecycle(ctx context.Context, tx pgx.Tx, alarm *model.Alarm) error {
	additionalJSON, err := json.Marshal(alarm.AdditionalInfo)
	if err != nil {
		return fmt.Errorf("marshal active alarm additional info: %w", err)
	}
	query, args, err := storage.Psql.Insert("alarms_active").Columns(
		"id", "device_id", "device_sn", "carrier", "severity", "alarm_type",
		"alarm_identifier", "description", "status", "raised_at", "acknowledged_at",
		"acknowledged_by", "ack_note", "additional_info", "created_at", "updated_at",
		"device_name", "technology", "alarm_source", "event_type", "network_location",
		"explicit_cause", "is_read", "ack_count", "first_raised_at", "last_updated_at",
		"probable_cause", "is_unknown", "alarm_version",
	).Values(
		alarm.ID, alarm.DeviceID, alarm.DeviceSN, alarm.Carrier, alarm.Severity, alarm.AlarmType,
		alarm.AlarmIdentifier, alarm.Description, alarm.Status, alarm.RaisedAt, alarm.AcknowledgedAt,
		alarm.AcknowledgedBy, alarm.AckNote, additionalJSON, alarm.CreatedAt, alarm.UpdatedAt,
		alarm.DeviceName, alarm.Technology, alarm.AlarmSource, alarm.EventType, alarm.NetworkLocation,
		alarm.ExplicitCause, alarm.IsRead, alarm.AckCount, alarm.FirstRaisedAt, alarm.LastUpdatedAt,
		alarmProbableCauseValue(alarm.ProbableCause), alarm.IsUnknown, alarm.Version,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build active alarm lifecycle insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert active alarm lifecycle row: %w", err)
	}
	return nil
}

func updateActiveLifecycle(ctx context.Context, tx pgx.Tx, alarm *model.Alarm, expectedVersion int64) error {
	additionalJSON, err := json.Marshal(alarm.AdditionalInfo)
	if err != nil {
		return fmt.Errorf("marshal updated alarm additional info: %w", err)
	}
	query, args, err := storage.Psql.Update("alarms_active").
		Set("severity", alarm.Severity).
		Set("alarm_type", alarm.AlarmType).
		Set("description", alarm.Description).
		Set("status", alarm.Status).
		Set("raised_at", alarm.RaisedAt).
		Set("acknowledged_at", alarm.AcknowledgedAt).
		Set("acknowledged_by", alarm.AcknowledgedBy).
		Set("ack_note", alarm.AckNote).
		Set("additional_info", additionalJSON).
		Set("updated_at", alarm.UpdatedAt).
		Set("device_name", alarm.DeviceName).
		Set("technology", alarm.Technology).
		Set("alarm_source", alarm.AlarmSource).
		Set("event_type", alarm.EventType).
		Set("network_location", alarm.NetworkLocation).
		Set("explicit_cause", alarm.ExplicitCause).
		Set("is_read", alarm.IsRead).
		Set("ack_count", alarm.AckCount).
		Set("first_raised_at", alarm.FirstRaisedAt).
		Set("last_updated_at", alarm.LastUpdatedAt).
		Set("probable_cause", alarmProbableCauseValue(alarm.ProbableCause)).
		Set("is_unknown", alarm.IsUnknown).
		Set("alarm_version", alarm.Version).
		Where(squirrel.Eq{"id": alarm.ID, "alarm_version": expectedVersion}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build active alarm lifecycle update: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update active alarm lifecycle row: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: occurrence %s changed during update", ErrAlarmVersionConflict, alarm.ID)
	}
	return nil
}

func alarmProbableCauseValue(probableCause *string) string {
	if probableCause == nil {
		return ""
	}
	return *probableCause
}

func deleteActiveLifecycle(ctx context.Context, tx pgx.Tx, id uuid.UUID, expectedVersion int64) error {
	query, args, err := storage.Psql.Delete("alarms_active").
		Where(squirrel.Eq{"id": id, "alarm_version": expectedVersion}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build active alarm lifecycle delete: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete active alarm lifecycle row: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: occurrence %v changed during clear", ErrAlarmVersionConflict, id)
	}
	return nil
}

func insertAlarmOutbox(ctx context.Context, tx pgx.Tx, payload event.AlarmLifecyclePayload) error {
	subject, err := payload.LifecycleType.Subject()
	if err != nil {
		return fmt.Errorf("resolve alarm lifecycle subject: %w", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal alarm lifecycle payload: %w", err)
	}
	query, args, err := storage.Psql.Insert("alarm_event_outbox").Columns(
		"event_id", "aggregate_type", "aggregate_id", "aggregate_version", "subject", "payload",
	).Values(
		payload.EventID,
		OutboxAggregateTypeAlarmOccurrence,
		payload.OccurrenceID,
		payload.AlarmVersion,
		subject,
		json.RawMessage(data),
	).ToSql()
	if err != nil {
		return fmt.Errorf("build alarm event outbox insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert alarm event outbox: %w", err)
	}
	return nil
}
