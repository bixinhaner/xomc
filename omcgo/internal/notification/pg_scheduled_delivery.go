package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/omcgo/omcgo/internal/core/storage"
)

func (r *PgDeliveryRepository) CreateScheduledDelivery(
	ctx context.Context,
	schedule DomainSchedule,
	workerID string,
	now time.Time,
) (uuid.UUID, error) {
	if schedule.ID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("create scheduled notification delivery: schedule ID is required")
	}
	if workerID == "" {
		return uuid.Nil, fmt.Errorf("create scheduled notification delivery: worker ID is required")
	}
	if now.IsZero() {
		return uuid.Nil, fmt.Errorf("create scheduled notification delivery: current time is required")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin scheduled notification delivery: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	linkedID, err := lockClaimedScheduleDelivery(ctx, tx, schedule.ID, workerID, now)
	if err != nil {
		return uuid.Nil, err
	}
	if linkedID != nil {
		if err := tx.Commit(ctx); err != nil {
			return uuid.Nil, fmt.Errorf("commit existing scheduled notification delivery: %w", err)
		}
		return *linkedID, nil
	}

	delivery := deliveryFromSchedule(schedule, now)
	deliveryID, err := insertOrFindScheduledDelivery(ctx, tx, delivery)
	if err != nil {
		return uuid.Nil, err
	}
	if schedule.ScheduleKind == ScheduleKindDigestFlush {
		if err := closeDigestBuckets(ctx, tx, schedule, deliveryID, now); err != nil {
			return uuid.Nil, err
		}
	}
	query, args, err := storage.Psql.Update("notification_schedules").
		Set("delivery_id", deliveryID).Set("updated_at", now).
		Where(sq.Eq{"id": schedule.ID, "state": "claimed", "locked_by": workerID, "delivery_id": nil}).
		Where(sq.Gt{"lease_expires_at": now}).ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build notification schedule delivery link: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return uuid.Nil, fmt.Errorf("link notification schedule delivery: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return uuid.Nil, fmt.Errorf("link notification schedule delivery: %w", ErrScheduleLeaseLost)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit scheduled notification delivery: %w", err)
	}
	return deliveryID, nil
}

func closeDigestBuckets(
	ctx context.Context,
	tx pgx.Tx,
	schedule DomainSchedule,
	deliveryID uuid.UUID,
	now time.Time,
) error {
	query, args, err := storage.Psql.Update("notification_aggregation_buckets").
		Set("state", "closed").Set("delivery_id", deliveryID).Set("updated_at", now).
		Where(sq.Eq{
			"rule_version_id": schedule.RuleVersionID, "channel": schedule.Channel,
			"recipient_fingerprint": schedule.RecipientFingerprint,
			"window_ends_at":        schedule.DueAt, "state": "open",
		}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification digest bucket close: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("close notification digest buckets: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("close notification digest buckets: no open bucket matches schedule %s", schedule.ID)
	}
	return nil
}

func lockClaimedScheduleDelivery(
	ctx context.Context,
	tx pgx.Tx,
	scheduleID uuid.UUID,
	workerID string,
	now time.Time,
) (*uuid.UUID, error) {
	query, args, err := storage.Psql.Select("delivery_id").From("notification_schedules").
		Where(sq.Eq{"id": scheduleID, "state": "claimed", "locked_by": workerID}).
		Where(sq.Gt{"lease_expires_at": now}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build claimed notification schedule lock: %w", err)
	}
	var deliveryID *uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&deliveryID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("lock claimed notification schedule: %w", ErrScheduleLeaseLost)
		}
		return nil, fmt.Errorf("lock claimed notification schedule: %w", err)
	}
	return deliveryID, nil
}

func insertOrFindScheduledDelivery(
	ctx context.Context,
	tx pgx.Tx,
	delivery DomainDelivery,
) (uuid.UUID, error) {
	query, args, err := buildDomainDeliveryInsert(&delivery)
	if err != nil {
		return uuid.Nil, fmt.Errorf("build scheduled notification delivery insert: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert scheduled notification delivery: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return delivery.ID, nil
	}

	query, args, err = storage.Psql.Select("id").From("notification_deliveries").Where(sq.Eq{
		"event_id": delivery.EventID, "dispatch_kind": delivery.DispatchKind,
		"sequence_no": delivery.SequenceNo, "channel": delivery.Channel,
		"recipient_fingerprint": delivery.RecipientFingerprint,
	}).ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build existing scheduled notification delivery lookup: %w", err)
	}
	var deliveryID uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&deliveryID); err != nil {
		return uuid.Nil, fmt.Errorf("find existing scheduled notification delivery: %w", err)
	}
	return deliveryID, nil
}
