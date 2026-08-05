package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

func (r *PgDeliveryRepository) ListDeliveries(ctx context.Context, filter DeliveryFilter) ([]DeliveryRecord, error) {
	builder := deliveryRecordSelect()
	if filter.OccurrenceID != nil {
		builder = builder.Where(sq.Eq{"d.occurrence_id": *filter.OccurrenceID})
	}
	if filter.Channel != "" {
		builder = builder.Where(sq.Eq{"d.channel": filter.Channel})
	}
	if filter.FlowState != "" {
		builder = builder.Where(sq.Eq{"d.flow_state": filter.FlowState})
	}
	builder = authz.ApplyDeviceVisibilityGrantsFilter(builder, "o.device_id", "o.technology", filter.Grants).
		OrderBy("d.created_at DESC", "d.id").Limit(uint64(filter.Limit)).Offset(uint64(filter.Offset))
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification delivery list: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query notification deliveries: %w", err)
	}
	defer rows.Close()
	records := make([]DeliveryRecord, 0, filter.Limit)
	for rows.Next() {
		var record DeliveryRecord
		if err := scanDeliveryRecord(rows, &record); err != nil {
			return nil, fmt.Errorf("scan notification delivery: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification deliveries: %w", err)
	}
	return records, nil
}

func (r *PgDeliveryRepository) GetDelivery(
	ctx context.Context,
	id uuid.UUID,
	grants []model.DeviceVisibilityGrant,
) (*DeliveryRecord, error) {
	builder := deliveryRecordSelect().Where(sq.Eq{"d.id": id})
	builder = authz.ApplyDeviceVisibilityGrantsFilter(builder, "o.device_id", "o.technology", grants)
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification delivery lookup: %w", err)
	}
	var record DeliveryRecord
	if err := scanDeliveryRecord(r.db.QueryRow(ctx, query, args...), &record); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeliveryNotFound
		}
		return nil, fmt.Errorf("get notification delivery: %w", err)
	}
	return &record, nil
}

func (r *PgDeliveryRepository) ListDeliveryAttempts(ctx context.Context, deliveryID uuid.UUID) ([]DomainDeliveryAttempt, error) {
	query, args, err := storage.Psql.Select(
		"id", "delivery_id", "attempt_no", "started_at", "finished_at", "result", "error_category",
		"status_summary", "provider_request_id", "next_retry_at", "created_at",
	).From("notification_delivery_attempts").Where(sq.Eq{"delivery_id": deliveryID}).
		OrderBy("attempt_no", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification delivery attempt list: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query notification delivery attempts: %w", err)
	}
	defer rows.Close()
	attempts := make([]DomainDeliveryAttempt, 0)
	for rows.Next() {
		var attempt DomainDeliveryAttempt
		if err := rows.Scan(
			&attempt.ID, &attempt.DeliveryID, &attempt.AttemptNo, &attempt.StartedAt, &attempt.FinishedAt,
			&attempt.Result, &attempt.ErrorCategory, &attempt.StatusSummary, &attempt.ProviderRequestID,
			&attempt.NextRetryAt, &attempt.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification delivery attempt: %w", err)
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification delivery attempts: %w", err)
	}
	return attempts, nil
}

func (r *PgDeliveryRepository) RetryDeadLetters(
	ctx context.Context,
	ids []uuid.UUID,
	grants []model.DeviceVisibilityGrant,
	reason string,
	actor string,
	now time.Time,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin notification dead-letter retry: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	builder := storage.Psql.Select("d.id").From("notification_deliveries d").
		Join("notification_occurrences o ON o.occurrence_id = d.occurrence_id").
		Join("notification_channel_configs c ON c.id = d.channel_config_id").
		LeftJoin("notification_channel_health h ON h.channel_config_id = c.id").
		Where(sq.Eq{
			"d.id": ids, "d.flow_state": "dead_letter", "c.enabled": true,
			"d.failure_reason": []string{"configuration_error", "authentication_error", "tls_error", "channel_config_error"},
		}).
		Where(sq.Expr("COALESCE(h.circuit_state, 'closed') = 'closed'")).
		Where(sq.Expr("h.last_verified_at IS NOT NULL AND h.last_verified_at > COALESCE(d.completed_at, d.updated_at)"))
	builder = authz.ApplyDeviceVisibilityGrantsFilter(builder, "o.device_id", "o.technology", grants).
		OrderBy("d.id").Suffix("FOR UPDATE OF d")
	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build retryable notification delivery lock: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("lock retryable notification deliveries: %w", err)
	}
	locked := make([]uuid.UUID, 0, len(ids))
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan retryable notification delivery: %w", err)
		}
		locked = append(locked, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate retryable notification deliveries: %w", err)
	}
	rows.Close()
	if len(locked) != len(ids) {
		return 0, ErrDeliveryRetryNotAllowed
	}

	query, args, err = storage.Psql.Update("notification_deliveries").
		Set("flow_state", "queued").Set("delivery_result", "none").
		Set("available_at", now).Set("next_attempt_at", now).
		Set("locked_by", nil).Set("locked_at", nil).Set("lease_expires_at", nil).
		Set("failure_reason", nil).Set("manual_retry_reason", reason).
		Set("manual_retry_by", actor).Set("manual_retry_at", now).Set("updated_at", now).
		Where(sq.Eq{"id": locked, "flow_state": "dead_letter"}).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build notification dead-letter retry: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("retry notification dead letters: %w", err)
	}
	if tag.RowsAffected() != int64(len(locked)) {
		return 0, fmt.Errorf("retry notification dead letters: delivery state changed concurrently")
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit notification dead-letter retry: %w", err)
	}
	return len(locked), nil
}

func deliveryRecordSelect() sq.SelectBuilder {
	return storage.Psql.Select(
		"d.id", "d.event_id", "d.occurrence_id", "d.rule_version_id", "d.template_version_id",
		"d.channel_config_id", "d.channel", "d.dispatch_kind", "d.sequence_no", "d.recipient_type",
		"d.address_ciphertext", "d.address_key_version", "d.recipient_fingerprint", "d.flow_state",
		"d.delivery_result", "d.available_at", "d.next_attempt_at", "d.locked_by", "d.locked_at",
		"d.lease_expires_at", "d.occurrence_version", "d.schedule_generation", "d.provider_message_id",
		"d.origin_delivery_id", "d.suppression_reason", "d.maintenance_window_id", "d.failure_reason",
		"d.manual_retry_reason", "d.manual_retry_by", "d.manual_retry_at", "d.accepted_at", "d.completed_at",
		"d.created_at", "d.updated_at", "o.device_id", "o.device_sn", "o.technology", "o.severity", "o.status",
	).From("notification_deliveries d").Join("notification_occurrences o ON o.occurrence_id = d.occurrence_id")
}

type deliveryRecordScanner interface{ Scan(...any) error }

func scanDeliveryRecord(row deliveryRecordScanner, record *DeliveryRecord) error {
	delivery := &record.DomainDelivery
	return row.Scan(
		&delivery.ID, &delivery.EventID, &delivery.OccurrenceID, &delivery.RuleVersionID,
		&delivery.TemplateVersionID, &delivery.ChannelConfigID, &delivery.Channel, &delivery.DispatchKind,
		&delivery.SequenceNo, &delivery.RecipientType, &delivery.AddressCiphertext, &delivery.AddressKeyVersion,
		&delivery.RecipientFingerprint, &delivery.FlowState, &delivery.DeliveryResult, &delivery.AvailableAt,
		&delivery.NextAttemptAt, &delivery.LockedBy, &delivery.LockedAt, &delivery.LeaseExpiresAt,
		&delivery.OccurrenceVersion, &delivery.ScheduleGeneration, &delivery.ProviderMessageID,
		&delivery.OriginDeliveryID, &delivery.SuppressionReason, &delivery.MaintenanceWindowID,
		&delivery.FailureReason, &delivery.ManualRetryReason, &delivery.ManualRetryBy, &delivery.ManualRetryAt,
		&delivery.AcceptedAt, &delivery.CompletedAt, &delivery.CreatedAt, &delivery.UpdatedAt,
		&record.DeviceID, &record.DeviceSN, &record.Technology, &record.Severity, &record.AlarmStatus,
	)
}
