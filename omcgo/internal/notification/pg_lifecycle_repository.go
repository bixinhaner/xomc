package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgLifecycleRepository struct{ db storage.DB }

func NewPgLifecycleRepository(pool *pgxpool.Pool) *PgLifecycleRepository {
	return &PgLifecycleRepository{db: storage.NewPoolDB(pool)}
}

func newPgLifecycleRepository(db storage.DB) *PgLifecycleRepository {
	return &PgLifecycleRepository{db: db}
}

func (r *PgLifecycleRepository) ApplyLifecycle(ctx context.Context, payload event.AlarmLifecyclePayload) (LifecycleApplyResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return LifecycleApplyResult{}, fmt.Errorf("begin notification lifecycle apply: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return LifecycleApplyResult{}, fmt.Errorf("marshal notification lifecycle payload: %w", err)
	}
	insertQuery, insertArgs, err := storage.Psql.Insert("notification_events").
		Columns("event_id", "event_type", "occurrence_id", "alarm_version", "schema_version", "occurred_at", "payload", "processing_state").
		Values(payload.EventID, payload.LifecycleType, payload.OccurrenceID, payload.AlarmVersion, payload.SchemaVersion, payload.OccurredAt, payloadJSON, "pending").
		Suffix("ON CONFLICT (event_id) DO NOTHING").ToSql()
	if err != nil {
		return LifecycleApplyResult{}, fmt.Errorf("build notification lifecycle inbox insert: %w", err)
	}
	tag, err := tx.Exec(ctx, insertQuery, insertArgs...)
	if err != nil {
		return LifecycleApplyResult{}, fmt.Errorf("insert notification lifecycle inbox event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			return LifecycleApplyResult{}, fmt.Errorf("commit duplicate notification lifecycle event: %w", err)
		}
		return LifecycleApplyResult{State: LifecycleDuplicate}, nil
	}

	occurrence, err := lockDomainOccurrence(ctx, tx, payload.OccurrenceID)
	if errors.Is(err, pgx.ErrNoRows) {
		if payload.AlarmVersion == 1 {
			if err := insertDomainOccurrence(ctx, tx, payload, 1, false); err != nil {
				return LifecycleApplyResult{}, err
			}
			if err := markLifecycleEvent(ctx, tx, payload.EventID, "applied"); err != nil {
				return LifecycleApplyResult{}, err
			}
			occurrence = occurrenceFromPayload(payload, nextScheduleGeneration(1, payload.LifecycleType), false)
		} else {
			if err := insertDomainOccurrence(ctx, tx, payload, 0, true); err != nil {
				return LifecycleApplyResult{}, err
			}
			if err := markLifecycleEvent(ctx, tx, payload.EventID, "waiting"); err != nil {
				return LifecycleApplyResult{}, err
			}
			if err := tx.Commit(ctx); err != nil {
				return LifecycleApplyResult{}, fmt.Errorf("commit waiting notification lifecycle event: %w", err)
			}
			return LifecycleApplyResult{State: LifecycleWaiting}, nil
		}
	} else if err != nil {
		return LifecycleApplyResult{}, fmt.Errorf("lock notification occurrence: %w", err)
	} else {
		switch {
		case payload.AlarmVersion <= occurrence.LastAppliedVersion:
			if err := markLifecycleEvent(ctx, tx, payload.EventID, "ignored"); err != nil {
				return LifecycleApplyResult{}, err
			}
			if err := tx.Commit(ctx); err != nil {
				return LifecycleApplyResult{}, fmt.Errorf("commit ignored notification lifecycle event: %w", err)
			}
			return LifecycleApplyResult{State: LifecycleIgnored}, nil
		case payload.AlarmVersion > occurrence.LastAppliedVersion+1:
			if err := markLifecycleEvent(ctx, tx, payload.EventID, "waiting"); err != nil {
				return LifecycleApplyResult{}, err
			}
			if err := markOccurrenceGap(ctx, tx, payload.OccurrenceID, payload.OccurredAt); err != nil {
				return LifecycleApplyResult{}, err
			}
			if err := tx.Commit(ctx); err != nil {
				return LifecycleApplyResult{}, fmt.Errorf("commit waiting notification lifecycle event: %w", err)
			}
			return LifecycleApplyResult{State: LifecycleWaiting}, nil
		default:
			if err := updateDomainOccurrence(ctx, tx, payload, occurrence.ScheduleGeneration); err != nil {
				return LifecycleApplyResult{}, err
			}
			if err := markLifecycleEvent(ctx, tx, payload.EventID, "applied"); err != nil {
				return LifecycleApplyResult{}, err
			}
			occurrence = occurrenceFromPayload(payload, nextScheduleGeneration(occurrence.ScheduleGeneration, payload.LifecycleType), false)
		}
	}

	appliedCount := 1
	for {
		nextPayload, found, err := lockNextWaitingLifecycle(ctx, tx, payload.OccurrenceID, occurrence.LastAppliedVersion+1)
		if err != nil {
			return LifecycleApplyResult{}, err
		}
		if !found {
			break
		}
		if err := updateDomainOccurrence(ctx, tx, nextPayload, occurrence.ScheduleGeneration); err != nil {
			return LifecycleApplyResult{}, err
		}
		if err := markLifecycleEvent(ctx, tx, nextPayload.EventID, "applied"); err != nil {
			return LifecycleApplyResult{}, err
		}
		occurrence = occurrenceFromPayload(nextPayload, nextScheduleGeneration(occurrence.ScheduleGeneration, nextPayload.LifecycleType), false)
		appliedCount++
	}
	if err := ignoreStaleWaitingLifecycle(ctx, tx, payload.OccurrenceID, occurrence.LastAppliedVersion); err != nil {
		return LifecycleApplyResult{}, err
	}
	if err := refreshOccurrenceGap(ctx, tx, payload.OccurrenceID, occurrence.LastAppliedVersion); err != nil {
		return LifecycleApplyResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LifecycleApplyResult{}, fmt.Errorf("commit notification lifecycle apply: %w", err)
	}
	return LifecycleApplyResult{State: LifecycleApplied, AppliedCount: appliedCount}, nil
}

func lockDomainOccurrence(ctx context.Context, tx pgx.Tx, occurrenceID uuid.UUID) (DomainOccurrence, error) {
	query, args, err := storage.Psql.Select(
		"id", "occurrence_id", "last_applied_version", "schedule_generation", "status", "severity",
		"device_id", "device_sn", "carrier", "technology", "alarm_identifier", "current_snapshot",
		"raised_at", "acknowledged_at", "cleared_at", "last_event_id", "version_gap", "gap_first_seen_at",
		"created_at", "updated_at",
	).From("notification_occurrences").Where(sq.Eq{"occurrence_id": occurrenceID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return DomainOccurrence{}, fmt.Errorf("build notification occurrence lock: %w", err)
	}
	var occurrence DomainOccurrence
	err = tx.QueryRow(ctx, query, args...).Scan(
		&occurrence.ID, &occurrence.OccurrenceID, &occurrence.LastAppliedVersion, &occurrence.ScheduleGeneration,
		&occurrence.Status, &occurrence.Severity, &occurrence.DeviceID, &occurrence.DeviceSN,
		&occurrence.Carrier, &occurrence.Technology, &occurrence.AlarmIdentifier, &occurrence.CurrentSnapshot,
		&occurrence.RaisedAt, &occurrence.AcknowledgedAt, &occurrence.ClearedAt, &occurrence.LastEventID,
		&occurrence.VersionGap, &occurrence.GapFirstSeenAt, &occurrence.CreatedAt, &occurrence.UpdatedAt,
	)
	return occurrence, err
}

func insertDomainOccurrence(ctx context.Context, tx pgx.Tx, payload event.AlarmLifecyclePayload, version int64, gap bool) error {
	snapshot := []byte(`{}`)
	if version > 0 {
		var err error
		snapshot, err = json.Marshal(payload.Snapshot)
		if err != nil {
			return fmt.Errorf("marshal notification occurrence snapshot: %w", err)
		}
	}
	technology := ""
	if payload.Snapshot.Technology != nil {
		technology = *payload.Snapshot.Technology
	}
	generation := int64(1)
	status := any(payload.Snapshot.Status)
	if version > 0 {
		generation = nextScheduleGeneration(1, payload.LifecycleType)
	} else {
		status = "active"
	}
	query, args, err := storage.Psql.Insert("notification_occurrences").Columns(
		"occurrence_id", "last_applied_version", "schedule_generation", "status", "severity", "device_id",
		"device_sn", "carrier", "technology", "alarm_identifier", "current_snapshot", "raised_at",
		"acknowledged_at", "cleared_at", "last_event_id", "version_gap", "gap_first_seen_at",
	).Values(
		payload.OccurrenceID, version, generation, status, payload.Snapshot.Severity,
		payload.Snapshot.DeviceID, payload.Snapshot.DeviceSN, payload.Snapshot.Carrier, technology,
		payload.Snapshot.AlarmIdentifier, snapshot, payload.Snapshot.RaisedAt, payload.Snapshot.AcknowledgedAt,
		payload.Snapshot.ClearedAt, payload.EventID, gap, nullableGapTime(gap, payload.OccurredAt),
	).ToSql()
	if err != nil {
		return fmt.Errorf("build notification occurrence insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert notification occurrence: %w", err)
	}
	return nil
}

func updateDomainOccurrence(ctx context.Context, tx pgx.Tx, payload event.AlarmLifecyclePayload, currentGeneration int64) error {
	snapshot, err := json.Marshal(payload.Snapshot)
	if err != nil {
		return fmt.Errorf("marshal notification occurrence snapshot: %w", err)
	}
	technology := ""
	if payload.Snapshot.Technology != nil {
		technology = *payload.Snapshot.Technology
	}
	generation := nextScheduleGeneration(currentGeneration, payload.LifecycleType)
	query, args, err := storage.Psql.Update("notification_occurrences").
		Set("last_applied_version", payload.AlarmVersion).
		Set("schedule_generation", generation).
		Set("status", payload.Snapshot.Status).
		Set("severity", payload.Snapshot.Severity).
		Set("device_sn", payload.Snapshot.DeviceSN).
		Set("carrier", payload.Snapshot.Carrier).
		Set("technology", technology).
		Set("alarm_identifier", payload.Snapshot.AlarmIdentifier).
		Set("current_snapshot", snapshot).
		Set("acknowledged_at", payload.Snapshot.AcknowledgedAt).
		Set("cleared_at", payload.Snapshot.ClearedAt).
		Set("last_event_id", payload.EventID).
		Set("updated_at", payload.OccurredAt).
		Where(sq.Eq{"occurrence_id": payload.OccurrenceID}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification occurrence update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update notification occurrence: %w", err)
	}
	if generation != currentGeneration {
		cancelQuery, cancelArgs, buildErr := storage.Psql.Update("notification_schedules").
			Set("state", "cancelled").Set("cancelled_at", payload.OccurredAt).Set("updated_at", payload.OccurredAt).
			Where(sq.Eq{
				"occurrence_id": payload.OccurrenceID, "state": []string{"pending", "claimed"},
				"schedule_kind": []string{ScheduleKindInitialGate, ScheduleKindRepeat, ScheduleKindQuietHoursRelease},
			}).
			Where(sq.Lt{"generation": generation}).ToSql()
		if buildErr != nil {
			return fmt.Errorf("build stale notification schedule cancellation: %w", buildErr)
		}
		if _, err := tx.Exec(ctx, cancelQuery, cancelArgs...); err != nil {
			return fmt.Errorf("cancel stale notification schedules: %w", err)
		}
		if payload.LifecycleType == event.AlarmLifecycleUnacknowledged {
			if err := resumeFutureRepeatSchedules(ctx, tx, payload, generation); err != nil {
				return err
			}
		}
	}
	return nil
}

func resumeFutureRepeatSchedules(
	ctx context.Context,
	tx pgx.Tx,
	payload event.AlarmLifecyclePayload,
	newGeneration int64,
) error {
	// An unacknowledge resumes only repeats that were already planned and are
	// still in the future. initial_gate is deliberately excluded, and DISTINCT
	// ON selects the latest prior generation without expanding the rule policy.
	selectSchedules := sq.Select(
		"DISTINCT ON (rule_version_id, channel, recipient_fingerprint, schedule_kind, sequence_no) event_id",
		"occurrence_id", "rule_version_id", "template_version_id", "channel_config_id", "channel",
		"dispatch_kind", "recipient_type", "address_ciphertext", "address_key_version",
		"recipient_fingerprint", "schedule_kind", "sequence_no",
	).Column(sq.Expr("?", newGeneration)).Column("due_at").
		Column(sq.Expr("?", "pending")).Column(sq.Expr("?", payload.AlarmVersion)).
		From("notification_schedules").
		Where(sq.Eq{
			"occurrence_id": payload.OccurrenceID,
			"schedule_kind": ScheduleKindRepeat,
			"state":         "cancelled",
		}).
		Where(sq.Lt{"generation": newGeneration}).
		Where(sq.Gt{"due_at": payload.OccurredAt}).
		OrderBy("rule_version_id", "channel", "recipient_fingerprint", "schedule_kind", "sequence_no", "generation DESC")
	query, args, err := storage.Psql.Insert("notification_schedules").Columns(
		"event_id", "occurrence_id", "rule_version_id", "template_version_id", "channel_config_id",
		"channel", "dispatch_kind", "recipient_type", "address_ciphertext", "address_key_version",
		"recipient_fingerprint", "schedule_kind",
		"sequence_no", "generation", "due_at", "state", "created_event_version",
	).Select(selectSchedules).
		Suffix("ON CONFLICT (occurrence_id, rule_version_id, channel, recipient_fingerprint, schedule_kind, sequence_no, generation) DO NOTHING").ToSql()
	if err != nil {
		return fmt.Errorf("build future notification repeat resume: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("resume future notification repeats: %w", err)
	}
	return nil
}

func nextScheduleGeneration(current int64, lifecycleType event.AlarmLifecycleType) int64 {
	switch lifecycleType {
	case event.AlarmLifecycleAcknowledged, event.AlarmLifecycleUnacknowledged, event.AlarmLifecycleCleared:
		return current + 1
	default:
		return current
	}
}

func occurrenceFromPayload(payload event.AlarmLifecyclePayload, generation int64, gap bool) DomainOccurrence {
	return DomainOccurrence{
		OccurrenceID: payload.OccurrenceID, LastAppliedVersion: payload.AlarmVersion,
		ScheduleGeneration: generation, VersionGap: gap,
	}
}

func markLifecycleEvent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, state string) error {
	update := storage.Psql.Update("notification_events").Set("processing_state", state).Set("last_error", nil)
	if state == "applied" || state == "ignored" {
		update = update.Set("processed_at", time.Now().UTC())
	} else {
		update = update.Set("processed_at", nil)
	}
	query, args, err := update.Where(sq.Eq{"event_id": eventID}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification event state update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update notification event state: %w", err)
	}
	return nil
}

func markOccurrenceGap(ctx context.Context, tx pgx.Tx, occurrenceID uuid.UUID, observedAt time.Time) error {
	query, args, err := storage.Psql.Update("notification_occurrences").Set("version_gap", true).
		Set("gap_first_seen_at", sq.Expr("COALESCE(gap_first_seen_at, ?)", observedAt)).
		Set("updated_at", observedAt).Where(sq.Eq{"occurrence_id": occurrenceID}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification occurrence gap update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark notification occurrence version gap: %w", err)
	}
	return nil
}

func lockNextWaitingLifecycle(ctx context.Context, tx pgx.Tx, occurrenceID uuid.UUID, version int64) (event.AlarmLifecyclePayload, bool, error) {
	query, args, err := storage.Psql.Select("payload").From("notification_events").
		Where(sq.Eq{"occurrence_id": occurrenceID, "alarm_version": version, "processing_state": "waiting"}).
		OrderBy("received_at", "event_id").Limit(1).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return event.AlarmLifecyclePayload{}, false, fmt.Errorf("build next waiting notification lifecycle lookup: %w", err)
	}
	var raw []byte
	if err := tx.QueryRow(ctx, query, args...).Scan(&raw); errors.Is(err, pgx.ErrNoRows) {
		return event.AlarmLifecyclePayload{}, false, nil
	} else if err != nil {
		return event.AlarmLifecyclePayload{}, false, fmt.Errorf("lock next waiting notification lifecycle: %w", err)
	}
	var payload event.AlarmLifecyclePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, false, fmt.Errorf("decode waiting notification lifecycle payload: %w", err)
	}
	return payload, true, nil
}

func ignoreStaleWaitingLifecycle(ctx context.Context, tx pgx.Tx, occurrenceID uuid.UUID, version int64) error {
	query, args, err := storage.Psql.Update("notification_events").Set("processing_state", "ignored").
		Set("processed_at", time.Now().UTC()).
		Where(sq.Eq{"occurrence_id": occurrenceID, "processing_state": "waiting"}).
		Where(sq.LtOrEq{"alarm_version": version}).ToSql()
	if err != nil {
		return fmt.Errorf("build stale waiting notification lifecycle update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("ignore stale waiting notification lifecycle events: %w", err)
	}
	return nil
}

func refreshOccurrenceGap(ctx context.Context, tx pgx.Tx, occurrenceID uuid.UUID, version int64) error {
	query, args, err := storage.Psql.Select("MIN(alarm_version)").From("notification_events").
		Where(sq.Eq{"occurrence_id": occurrenceID, "processing_state": "waiting"}).
		Where(sq.Gt{"alarm_version": version}).ToSql()
	if err != nil {
		return fmt.Errorf("build waiting notification lifecycle gap lookup: %w", err)
	}
	var next *int64
	if err := tx.QueryRow(ctx, query, args...).Scan(&next); err != nil {
		return fmt.Errorf("read waiting notification lifecycle gap: %w", err)
	}
	gap := next != nil && *next > version+1
	updateQuery, updateArgs, err := storage.Psql.Update("notification_occurrences").Set("version_gap", gap).
		Set("gap_first_seen_at", nullableGapTime(gap, time.Now().UTC())).Where(sq.Eq{"occurrence_id": occurrenceID}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification occurrence gap refresh: %w", err)
	}
	if _, err := tx.Exec(ctx, updateQuery, updateArgs...); err != nil {
		return fmt.Errorf("refresh notification occurrence version gap: %w", err)
	}
	return nil
}

func nullableGapTime(gap bool, value time.Time) any {
	if !gap {
		return nil
	}
	return value
}
