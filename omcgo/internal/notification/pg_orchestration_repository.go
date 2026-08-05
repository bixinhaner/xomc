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

var ErrOrchestrationEventNotApplied = errors.New("notification event has not been applied to occurrence")

type PgOrchestrationRepository struct{ db storage.DB }

func NewPgOrchestrationRepository(pool *pgxpool.Pool) *PgOrchestrationRepository {
	return &PgOrchestrationRepository{db: storage.NewPoolDB(pool)}
}

func newPgOrchestrationRepository(db storage.DB) *PgOrchestrationRepository {
	return &PgOrchestrationRepository{db: db}
}

func (r *PgOrchestrationRepository) ListPendingAppliedEvents(
	ctx context.Context,
	occurrenceID uuid.UUID,
) ([]event.AlarmLifecyclePayload, error) {
	query, args, err := storage.Psql.Select("payload").From("notification_events").
		Where(sq.Eq{
			"occurrence_id": occurrenceID, "processing_state": "applied",
			"orchestration_state": []string{"pending", "failed"},
		}).OrderBy("alarm_version", "received_at", "event_id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build pending notification orchestration event list: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pending notification orchestration events: %w", err)
	}
	defer rows.Close()
	payloads := make([]event.AlarmLifecyclePayload, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan pending notification orchestration event: %w", err)
		}
		var payload event.AlarmLifecyclePayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("decode pending notification orchestration event: %w", err)
		}
		payloads = append(payloads, payload)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending notification orchestration events: %w", err)
	}
	return payloads, nil
}

// SaveDecision commits all durable effects of one event together. The event
// row is the idempotency fence, so a JetStream redelivery cannot increment a
// digest bucket twice after the first transaction commits.
func (r *PgOrchestrationRepository) SaveDecision(
	ctx context.Context,
	eventID uuid.UUID,
	decision OrchestrationDecision,
) (bool, error) {
	if eventID == uuid.Nil {
		return false, fmt.Errorf("save notification orchestration decision: event ID is required")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin notification orchestration decision: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	processingState, orchestrationState, err := lockOrchestrationEvent(ctx, tx, eventID)
	if err != nil {
		return false, err
	}
	if processingState != "applied" {
		return false, fmt.Errorf("save notification orchestration decision: %w: state=%s", ErrOrchestrationEventNotApplied, processingState)
	}
	if orchestrationState == "completed" {
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit duplicate notification orchestration decision: %w", err)
		}
		return false, nil
	}

	newAggregationSchedules := make(map[string]bool, len(decision.Aggregations))
	for index := range decision.Aggregations {
		inserted, err := persistAggregationFact(ctx, tx, decision.Aggregations[index])
		if err != nil {
			return false, err
		}
		fact := decision.Aggregations[index]
		newAggregationSchedules[aggregationScheduleKey(
			fact.RuleVersionID, fact.Channel, fact.RecipientFingerprint, fact.WindowEnd,
		)] = inserted
	}
	for index := range decision.Schedules {
		schedule := &decision.Schedules[index]
		if schedule.ScheduleKind == ScheduleKindDigestFlush {
			key := aggregationScheduleKey(schedule.RuleVersionID, schedule.Channel, schedule.RecipientFingerprint, schedule.DueAt)
			if inserted, linked := newAggregationSchedules[key]; linked && !inserted {
				continue
			}
		}
		query, args, err := buildDomainScheduleInsert(schedule)
		if err != nil {
			return false, fmt.Errorf("build orchestrated notification schedule: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return false, fmt.Errorf("insert orchestrated notification schedule: %w", err)
		}
	}
	for index := range decision.Deliveries {
		query, args, err := buildDomainDeliveryInsert(&decision.Deliveries[index])
		if err != nil {
			return false, fmt.Errorf("build orchestrated notification delivery: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return false, fmt.Errorf("insert orchestrated notification delivery: %w", err)
		}
	}
	if decision.Fence != nil {
		if err := applyOrchestrationFence(ctx, tx, *decision.Fence); err != nil {
			return false, err
		}
	}
	if err := cancelDecisionWorkBehindCurrentOccurrence(ctx, tx, eventID); err != nil {
		return false, err
	}

	explanation, err := json.Marshal(decision.Explanations)
	if err != nil {
		return false, fmt.Errorf("marshal notification match explanation: %w", err)
	}
	query, args, err := storage.Psql.Update("notification_events").
		Set("orchestration_state", "completed").
		Set("match_explanation", explanation).
		Set("orchestrated_at", sq.Expr("now()")).
		Set("last_error", nil).
		Where(sq.Eq{"event_id": eventID, "orchestration_state": []string{"pending", "failed"}}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build notification event orchestration completion: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("complete notification event orchestration: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return false, fmt.Errorf("complete notification event orchestration: event state changed concurrently")
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit notification orchestration decision: %w", err)
	}
	return true, nil
}

func applyOrchestrationFence(ctx context.Context, tx pgx.Tx, fence OrchestrationFence) error {
	query, args, err := storage.Psql.Update("notification_schedules").
		Set("state", "cancelled").Set("cancelled_at", fence.OccurredAt).Set("updated_at", fence.OccurredAt).
		Where(sq.Eq{
			"occurrence_id": fence.OccurrenceID, "state": []string{"pending", "claimed"},
			"schedule_kind": []string{ScheduleKindInitialGate, ScheduleKindRepeat, ScheduleKindQuietHoursRelease},
		}).
		Where(sq.Lt{"generation": fence.Generation}).ToSql()
	if err != nil {
		return fmt.Errorf("build orchestrated stale schedule cancellation: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("cancel orchestrated stale schedules: %w", err)
	}
	if !fence.ResumeFutureRepeats {
		return nil
	}
	selectSchedules := sq.Select(
		"DISTINCT ON (rule_version_id, channel, recipient_fingerprint, schedule_kind, sequence_no) occurrence_id",
		"rule_version_id", "channel", "recipient_fingerprint", "schedule_kind", "sequence_no",
	).Column(sq.Expr("?", fence.Generation)).Column("due_at").
		Column(sq.Expr("?", "pending")).Column(sq.Expr("?", fence.EventVersion)).
		From("notification_schedules").
		Where(sq.Eq{"occurrence_id": fence.OccurrenceID, "schedule_kind": ScheduleKindRepeat, "state": "cancelled"}).
		Where(sq.Lt{"generation": fence.Generation}).Where(sq.Gt{"due_at": fence.OccurredAt}).
		OrderBy("rule_version_id", "channel", "recipient_fingerprint", "schedule_kind", "sequence_no", "generation DESC")
	query, args, err = storage.Psql.Insert("notification_schedules").Columns(
		"occurrence_id", "rule_version_id", "channel", "recipient_fingerprint", "schedule_kind",
		"sequence_no", "generation", "due_at", "state", "created_event_version",
	).Select(selectSchedules).
		Suffix("ON CONFLICT (occurrence_id, rule_version_id, channel, recipient_fingerprint, schedule_kind, sequence_no, generation) DO NOTHING").ToSql()
	if err != nil {
		return fmt.Errorf("build orchestrated future repeat resume: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("resume orchestrated future repeats: %w", err)
	}
	return nil
}

func cancelDecisionWorkBehindCurrentOccurrence(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error {
	query, args, err := storage.Psql.Update("notification_schedules s").
		Set("state", "cancelled").Set("cancelled_at", sq.Expr("now()")).Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{
			"state":         []string{"pending", "claimed"},
			"schedule_kind": []string{ScheduleKindInitialGate, ScheduleKindRepeat, ScheduleKindQuietHoursRelease},
		}).
		Where(sq.Expr(`EXISTS (
SELECT 1 FROM notification_events e
JOIN notification_occurrences o ON o.occurrence_id = e.occurrence_id
WHERE e.event_id = ? AND s.occurrence_id = e.occurrence_id AND s.generation < o.schedule_generation
)`, eventID)).ToSql()
	if err != nil {
		return fmt.Errorf("build current occurrence schedule fence: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("apply current occurrence schedule fence: %w", err)
	}
	query, args, err = storage.Psql.Update("notification_deliveries d").
		Set("flow_state", "cancelled").Set("failure_reason", "occurrence_fenced").Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"event_id": eventID, "flow_state": "queued"}).
		Where(sq.NotEq{"dispatch_kind": DispatchKindDigest}).
		Where(sq.Expr(`EXISTS (
SELECT 1 FROM notification_occurrences o
WHERE o.occurrence_id = d.occurrence_id AND d.schedule_generation < o.schedule_generation
)`)).ToSql()
	if err != nil {
		return fmt.Errorf("build current occurrence delivery fence: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("apply current occurrence delivery fence: %w", err)
	}
	return nil
}

func lockOrchestrationEvent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) (string, string, error) {
	query, args, err := storage.Psql.Select("processing_state", "orchestration_state").
		From("notification_events").Where(sq.Eq{"event_id": eventID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return "", "", fmt.Errorf("build notification event orchestration lock: %w", err)
	}
	var processingState, orchestrationState string
	if err := tx.QueryRow(ctx, query, args...).Scan(&processingState, &orchestrationState); err != nil {
		return "", "", fmt.Errorf("lock notification event for orchestration: %w", err)
	}
	return processingState, orchestrationState, nil
}

func buildAggregationInsert(fact AggregationFact) (string, []any, error) {
	id := uuid.New()
	return storage.Psql.Insert("notification_aggregation_buckets").Columns(
		"id", "rule_version_id", "channel", "recipient_fingerprint", "scope_fingerprint",
		"severity", "window_started_at", "window_ends_at", "state", "event_count",
	).Values(
		id, fact.RuleVersionID, fact.Channel, fact.RecipientFingerprint, fact.ScopeFingerprint,
		fact.Severity, fact.WindowStart, fact.WindowEnd, "open", fact.EventCount,
	).Suffix(`ON CONFLICT (rule_version_id, channel, recipient_fingerprint, scope_fingerprint, severity, window_started_at)
DO NOTHING RETURNING id`).ToSql()
}

func persistAggregationFact(ctx context.Context, tx pgx.Tx, fact AggregationFact) (bool, error) {
	query, args, err := buildAggregationInsert(fact)
	if err != nil {
		return false, fmt.Errorf("build notification aggregation insert: %w", err)
	}
	var id uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&id); err == nil {
		return true, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("insert notification aggregation bucket: %w", err)
	}

	query, args, err = storage.Psql.Update("notification_aggregation_buckets").
		Set("event_count", sq.Expr("event_count + ?", fact.EventCount)).
		Set("window_ends_at", sq.Expr("GREATEST(window_ends_at, ?)", fact.WindowEnd)).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{
			"rule_version_id": fact.RuleVersionID, "channel": fact.Channel,
			"recipient_fingerprint": fact.RecipientFingerprint, "scope_fingerprint": fact.ScopeFingerprint,
			"severity": fact.Severity, "window_started_at": fact.WindowStart, "state": "open",
		}).ToSql()
	if err != nil {
		return false, fmt.Errorf("build notification aggregation increment: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("increment notification aggregation bucket: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return false, fmt.Errorf("increment notification aggregation bucket: open bucket is unavailable")
	}
	return false, nil
}

func aggregationScheduleKey(ruleVersionID uuid.UUID, channel string, recipientFingerprint []byte, dueAt time.Time) string {
	return ruleVersionID.String() + ":" + channel + ":" + fmt.Sprintf("%x", recipientFingerprint) + ":" + dueAt.UTC().Format(time.RFC3339Nano)
}
