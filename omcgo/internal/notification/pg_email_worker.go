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

	"github.com/omcgo/omcgo/internal/core/storage"
)

func (r *PgDeliveryRepository) ClaimEmailDeliveries(
	ctx context.Context,
	request EmailDeliveryClaimRequest,
) ([]DomainDelivery, error) {
	if request.WorkerID == "" || request.Now.IsZero() || request.LeaseDuration <= 0 || request.Limit <= 0 {
		return nil, fmt.Errorf("claim notification email deliveries: valid worker, time, lease and limit are required")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin notification email delivery claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := finishExpiredUnknownEmailAttempts(ctx, tx, request.Now, request.Limit); err != nil {
		return nil, err
	}
	query, args, err := buildEmailDeliveryClaim(request)
	if err != nil {
		return nil, fmt.Errorf("build notification email delivery claim: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("claim due notification email deliveries: %w", err)
	}
	deliveries := make([]DomainDelivery, 0, request.Limit)
	for rows.Next() {
		var delivery DomainDelivery
		if err := scanWorkerDelivery(rows, &delivery); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan claimed notification email delivery: %w", err)
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate claimed notification email deliveries: %w", err)
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit notification email delivery claim: %w", err)
	}
	return deliveries, nil
}

func buildEmailDeliveryClaim(request EmailDeliveryClaimRequest) (string, []any, error) {
	due := sq.Select("d.id").From("notification_deliveries d").
		Where(sq.Eq{"d.channel": TemplateChannelEmail}).
		Where(sq.Or{
			sq.And{
				sq.Eq{"d.flow_state": []string{"queued", "retry_wait"}},
				sq.LtOrEq{"d.next_attempt_at": request.Now},
			},
			sq.And{
				sq.Eq{"d.flow_state": "sending"}, sq.LtOrEq{"d.lease_expires_at": request.Now},
				sq.Expr(`NOT EXISTS (SELECT 1 FROM notification_delivery_attempts a
                    WHERE a.delivery_id = d.id AND a.finished_at IS NULL)`),
			},
		}).OrderBy("d.next_attempt_at", "d.id").Limit(uint64(request.Limit)).Suffix("FOR UPDATE SKIP LOCKED")
	return storage.Psql.Update("notification_deliveries").
		Set("flow_state", "sending").Set("locked_by", request.WorkerID).
		Set("locked_at", request.Now).Set("lease_expires_at", request.Now.Add(request.LeaseDuration)).
		Set("updated_at", request.Now).
		Where(sq.Expr("id IN (?)", due)).
		Suffix(`RETURNING id, event_id, occurrence_id, rule_version_id, template_version_id,
            channel_config_id, channel, dispatch_kind, sequence_no, recipient_type,
            address_ciphertext, address_key_version, recipient_fingerprint, flow_state,
            delivery_result, available_at, next_attempt_at, locked_by, locked_at,
            lease_expires_at, occurrence_version, schedule_generation, provider_message_id,
            origin_delivery_id, suppression_reason, maintenance_window_id, failure_reason,
            accepted_at, completed_at, created_at, updated_at`).ToSql()
}

func finishExpiredUnknownEmailAttempts(ctx context.Context, tx pgx.Tx, now time.Time, limit int) error {
	query, args, err := storage.Psql.Select("d.id", "a.id").
		From("notification_deliveries d").
		Join("notification_delivery_attempts a ON a.delivery_id = d.id AND a.finished_at IS NULL").
		Where(sq.Eq{"d.channel": TemplateChannelEmail, "d.flow_state": "sending"}).
		Where(sq.LtOrEq{"d.lease_expires_at": now}).
		OrderBy("d.lease_expires_at", "d.id").Limit(uint64(limit)).
		Suffix("FOR UPDATE OF d, a SKIP LOCKED").ToSql()
	if err != nil {
		return fmt.Errorf("build expired notification email attempt lock: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("lock expired notification email attempts: %w", err)
	}
	var deliveryIDs, attemptIDs []uuid.UUID
	for rows.Next() {
		var deliveryID, attemptID uuid.UUID
		if err := rows.Scan(&deliveryID, &attemptID); err != nil {
			rows.Close()
			return fmt.Errorf("scan expired notification email attempt: %w", err)
		}
		deliveryIDs, attemptIDs = append(deliveryIDs, deliveryID), append(attemptIDs, attemptID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate expired notification email attempts: %w", err)
	}
	rows.Close()
	if len(deliveryIDs) == 0 {
		return nil
	}

	query, args, err = storage.Psql.Update("notification_delivery_attempts").
		Set("finished_at", now).Set("result", "unknown").Set("error_category", EmailErrorUnknown).
		Set("status_summary", "worker lease expired after external attempt started").
		Where(sq.Eq{"id": attemptIDs, "finished_at": nil}).ToSql()
	if err != nil {
		return fmt.Errorf("build expired notification email attempt completion: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("complete expired notification email attempts: %w", err)
	}
	query, args, err = storage.Psql.Update("notification_deliveries").
		Set("flow_state", "completed").Set("delivery_result", "unknown").
		Set("failure_reason", EmailErrorUnknown).Set("completed_at", now).
		Set("locked_by", nil).Set("locked_at", nil).Set("lease_expires_at", nil).Set("updated_at", now).
		Where(sq.Eq{"id": deliveryIDs, "flow_state": "sending"}).ToSql()
	if err != nil {
		return fmt.Errorf("build expired notification email delivery completion: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("complete expired notification email deliveries: %w", err)
	}
	return nil
}

func (r *PgDeliveryRepository) StartEmailAttempt(
	ctx context.Context,
	deliveryID uuid.UUID,
	workerID string,
	now time.Time,
) (DomainDeliveryAttempt, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return DomainDeliveryAttempt{}, fmt.Errorf("begin notification email attempt: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := storage.Psql.Select("id").From("notification_deliveries").Where(sq.Eq{
		"id": deliveryID, "channel": TemplateChannelEmail, "flow_state": "sending", "locked_by": workerID,
	}).Where(sq.Gt{"lease_expires_at": now}).Where(sq.Expr(`NOT EXISTS
        (SELECT 1 FROM notification_delivery_attempts a WHERE a.delivery_id = notification_deliveries.id AND a.finished_at IS NULL)`)).
		Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return DomainDeliveryAttempt{}, fmt.Errorf("build notification email delivery attempt lock: %w", err)
	}
	var lockedID uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&lockedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DomainDeliveryAttempt{}, fmt.Errorf("lock notification email delivery attempt: %w", ErrDeliveryLeaseInvalid)
		}
		return DomainDeliveryAttempt{}, fmt.Errorf("lock notification email delivery attempt: %w", err)
	}
	query, args, err = storage.Psql.Select("COALESCE(MAX(attempt_no), 0) + 1").
		From("notification_delivery_attempts").Where(sq.Eq{"delivery_id": deliveryID}).ToSql()
	if err != nil {
		return DomainDeliveryAttempt{}, fmt.Errorf("build notification email attempt sequence: %w", err)
	}
	var attemptNo int
	if err := tx.QueryRow(ctx, query, args...).Scan(&attemptNo); err != nil {
		return DomainDeliveryAttempt{}, fmt.Errorf("read notification email attempt sequence: %w", err)
	}
	attempt := DomainDeliveryAttempt{
		ID: uuid.New(), DeliveryID: deliveryID, AttemptNo: attemptNo, StartedAt: now, Result: "started",
	}
	query, args, err = buildDomainAttemptInsert(&attempt)
	if err != nil {
		return DomainDeliveryAttempt{}, fmt.Errorf("build notification email attempt insert: %w", err)
	}
	if tag, err := tx.Exec(ctx, query, args...); err != nil {
		return DomainDeliveryAttempt{}, fmt.Errorf("insert notification email attempt: %w", err)
	} else if tag.RowsAffected() != 1 {
		return DomainDeliveryAttempt{}, fmt.Errorf("insert notification email attempt: sequence changed concurrently")
	}
	if err := tx.Commit(ctx); err != nil {
		return DomainDeliveryAttempt{}, fmt.Errorf("commit notification email attempt: %w", err)
	}
	return attempt, nil
}

func (r *PgDeliveryRepository) LoadEmailDeliveryContent(
	ctx context.Context,
	delivery AuthorizedDelivery,
) (EmailDeliveryContent, error) {
	query, args, err := storage.Psql.Select(
		"t.id", "t.template_id", "t.version_no", "t.channel", "t.language", "t.subject",
		"t.text_body", "t.html_body", "t.variables", "t.created_by", "t.change_reason",
		"t.created_at", "t.published_at", "e.payload",
	).From("notification_template_versions t").
		Join("notification_events e ON e.event_id = ?", delivery.EventID).
		Where(sq.Eq{"t.id": delivery.TemplateVersionID}).ToSql()
	if err != nil {
		return EmailDeliveryContent{}, fmt.Errorf("build notification email delivery content lookup: %w", err)
	}
	var content EmailDeliveryContent
	var payload []byte
	template := &content.Template
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&template.ID, &template.TemplateID, &template.VersionNo, &template.Channel, &template.Language,
		&template.Subject, &template.TextBody, &template.HTMLBody, &template.Variables, &template.CreatedBy,
		&template.ChangeReason, &template.CreatedAt, &template.PublishedAt, &payload,
	); err != nil {
		return EmailDeliveryContent{}, fmt.Errorf("get notification email delivery content: %w", err)
	}
	if err := json.Unmarshal(payload, &content.Payload); err != nil {
		return EmailDeliveryContent{}, fmt.Errorf("decode notification email event payload: %w", err)
	}
	if delivery.DispatchKind == DispatchKindDigest {
		query, args, err = storage.Psql.Select(
			"COALESCE(SUM(event_count), 0)", "MIN(window_started_at)", "MAX(window_ends_at)",
		).From("notification_aggregation_buckets").Where(sq.Eq{"delivery_id": delivery.ID}).ToSql()
		if err != nil {
			return EmailDeliveryContent{}, fmt.Errorf("build notification email digest content lookup: %w", err)
		}
		if err := r.db.QueryRow(ctx, query, args...).Scan(
			&content.DigestEventCount, &content.DigestWindowStartedAt, &content.DigestWindowEndsAt,
		); err != nil {
			return EmailDeliveryContent{}, fmt.Errorf("get notification email digest content: %w", err)
		}
		if content.DigestEventCount <= 0 || content.DigestWindowStartedAt == nil || content.DigestWindowEndsAt == nil {
			return EmailDeliveryContent{}, fmt.Errorf("get notification email digest content: linked aggregation bucket is required")
		}
	}
	return content, nil
}

func (r *PgDeliveryRepository) FinishEmailAttempt(ctx context.Context, completion EmailAttemptCompletion) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin notification email attempt completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := storage.Psql.Select("channel_config_id").From("notification_deliveries").Where(sq.Eq{
		"id": completion.DeliveryID, "channel": TemplateChannelEmail,
		"flow_state": "sending", "locked_by": completion.WorkerID,
	}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return fmt.Errorf("build notification email attempt completion lock: %w", err)
	}
	var channelConfigID uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&channelConfigID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lock notification email attempt completion: %w", ErrDeliveryLeaseInvalid)
		}
		return fmt.Errorf("lock notification email attempt completion: %w", err)
	}
	query, args, err = storage.Psql.Update("notification_delivery_attempts").
		Set("finished_at", completion.FinishedAt).Set("result", completion.AttemptResult).
		Set("error_category", completion.ErrorCategory).Set("status_summary", completion.StatusSummary).
		Set("next_retry_at", completion.NextRetryAt).
		Where(sq.Eq{"id": completion.AttemptID, "delivery_id": completion.DeliveryID, "finished_at": nil}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification email attempt completion: %w", err)
	}
	if tag, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("complete notification email attempt: %w", err)
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("complete notification email attempt: attempt state changed concurrently")
	}
	update := storage.Psql.Update("notification_deliveries").
		Set("flow_state", completion.FlowState).Set("delivery_result", completion.DeliveryResult).
		Set("next_attempt_at", coalesceTime(completion.NextRetryAt, completion.FinishedAt)).
		Set("failure_reason", completion.FailureReason).
		Set("locked_by", nil).Set("locked_at", nil).Set("lease_expires_at", nil).
		Set("updated_at", completion.FinishedAt).
		Where(sq.Eq{"id": completion.DeliveryID, "flow_state": "sending", "locked_by": completion.WorkerID})
	if completion.DeliveryResult == "accepted" {
		update = update.Set("accepted_at", completion.FinishedAt).Set("completed_at", completion.FinishedAt)
	} else if completion.FlowState == "completed" || completion.FlowState == "dead_letter" {
		update = update.Set("completed_at", completion.FinishedAt)
	}
	query, args, err = update.ToSql()
	if err != nil {
		return fmt.Errorf("build notification email delivery completion: %w", err)
	}
	if tag, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("complete notification email delivery: %w", err)
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("complete notification email delivery: delivery state changed concurrently")
	}
	if err := updateEmailChannelHealth(ctx, tx, channelConfigID, completion); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit notification email attempt completion: %w", err)
	}
	return nil
}

func updateEmailChannelHealth(
	ctx context.Context,
	tx pgx.Tx,
	channelConfigID uuid.UUID,
	completion EmailAttemptCompletion,
) error {
	if completion.AttemptResult == "accepted" {
		query, args, err := storage.Psql.Insert("notification_channel_health").
			Columns("channel_config_id", "circuit_state", "consecutive_successes", "consecutive_failures", "last_success_at", "updated_at").
			Values(channelConfigID, "closed", 1, 0, completion.FinishedAt, completion.FinishedAt).
			Suffix(`ON CONFLICT (channel_config_id) DO UPDATE SET
                circuit_state='closed', consecutive_successes=notification_channel_health.consecutive_successes+1,
                consecutive_failures=0, last_success_at=EXCLUDED.last_success_at,
                last_error_category=NULL, last_error_summary=NULL, circuit_opened_at=NULL,
                next_probe_at=NULL, updated_at=EXCLUDED.updated_at`).ToSql()
		if err != nil {
			return fmt.Errorf("build notification email channel success health: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("record notification email channel success: %w", err)
		}
		return nil
	}
	if completion.ErrorCategory == nil || !emailFailureAffectsCircuit(*completion.ErrorCategory) {
		return nil
	}
	immediateOpen := *completion.ErrorCategory == EmailErrorConfiguration ||
		*completion.ErrorCategory == EmailErrorAuthentication || *completion.ErrorCategory == EmailErrorTLS
	state := "closed"
	if immediateOpen {
		state = "open"
	}
	nextProbe := completion.FinishedAt.Add(5 * time.Minute)
	query, args, err := storage.Psql.Insert("notification_channel_health").
		Columns("channel_config_id", "circuit_state", "consecutive_successes", "consecutive_failures",
			"last_failure_at", "last_error_category", "last_error_summary", "circuit_opened_at", "next_probe_at", "updated_at").
		Values(channelConfigID, state, 0, 1, completion.FinishedAt, completion.ErrorCategory,
			completion.StatusSummary, nullableTime(immediateOpen, completion.FinishedAt), nullableTime(immediateOpen, nextProbe), completion.FinishedAt).
		Suffix(`ON CONFLICT (channel_config_id) DO UPDATE SET
            consecutive_successes=0,
            consecutive_failures=notification_channel_health.consecutive_failures+1,
            circuit_state=CASE WHEN ? OR notification_channel_health.consecutive_failures+1 >= 3 THEN 'open' ELSE notification_channel_health.circuit_state END,
            last_failure_at=EXCLUDED.last_failure_at, last_error_category=EXCLUDED.last_error_category,
            last_error_summary=EXCLUDED.last_error_summary,
            circuit_opened_at=CASE WHEN ? OR notification_channel_health.consecutive_failures+1 >= 3 THEN COALESCE(notification_channel_health.circuit_opened_at, EXCLUDED.last_failure_at) ELSE notification_channel_health.circuit_opened_at END,
            next_probe_at=CASE WHEN ? OR notification_channel_health.consecutive_failures+1 >= 3 THEN ? ELSE notification_channel_health.next_probe_at END,
            updated_at=EXCLUDED.updated_at`, immediateOpen, immediateOpen, immediateOpen, nextProbe).ToSql()
	if err != nil {
		return fmt.Errorf("build notification email channel failure health: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("record notification email channel failure: %w", err)
	}
	return nil
}

func emailFailureAffectsCircuit(category string) bool {
	return category == EmailErrorConfiguration || category == EmailErrorAuthentication ||
		category == EmailErrorTLS || category == EmailErrorConnection || category == EmailErrorTemporary
}

func nullableTime(include bool, value time.Time) any {
	if !include {
		return nil
	}
	return value
}

func coalesceTime(value *time.Time, fallback time.Time) time.Time {
	if value == nil {
		return fallback
	}
	return *value
}

func scanWorkerDelivery(row interface{ Scan(...any) error }, delivery *DomainDelivery) error {
	return row.Scan(
		&delivery.ID, &delivery.EventID, &delivery.OccurrenceID, &delivery.RuleVersionID,
		&delivery.TemplateVersionID, &delivery.ChannelConfigID, &delivery.Channel, &delivery.DispatchKind,
		&delivery.SequenceNo, &delivery.RecipientType, &delivery.AddressCiphertext, &delivery.AddressKeyVersion,
		&delivery.RecipientFingerprint, &delivery.FlowState, &delivery.DeliveryResult, &delivery.AvailableAt,
		&delivery.NextAttemptAt, &delivery.LockedBy, &delivery.LockedAt, &delivery.LeaseExpiresAt,
		&delivery.OccurrenceVersion, &delivery.ScheduleGeneration, &delivery.ProviderMessageID,
		&delivery.OriginDeliveryID, &delivery.SuppressionReason, &delivery.MaintenanceWindowID,
		&delivery.FailureReason, &delivery.AcceptedAt, &delivery.CompletedAt, &delivery.CreatedAt,
		&delivery.UpdatedAt,
	)
}
