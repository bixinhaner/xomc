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

func (r *PgDeliveryRepository) AuthorizeSend(
	ctx context.Context,
	deliveryID uuid.UUID,
	now time.Time,
) (AuthorizedDelivery, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return AuthorizedDelivery{}, fmt.Errorf("begin notification send authorization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	snapshot, err := lockDeliveryAuthorizationSnapshot(ctx, tx, deliveryID)
	if err != nil {
		return AuthorizedDelivery{}, err
	}
	if rejection := evaluateDeliveryAuthorization(snapshot, now); rejection != nil {
		if rejection.flowState != "" {
			if err := rejectUnauthorizedDelivery(ctx, tx, deliveryID, rejection, now); err != nil {
				return AuthorizedDelivery{}, err
			}
			if err := tx.Commit(ctx); err != nil {
				return AuthorizedDelivery{}, fmt.Errorf("commit rejected notification send authorization: %w", err)
			}
		}
		return AuthorizedDelivery{}, rejection.err
	}
	if err := tx.Commit(ctx); err != nil {
		return AuthorizedDelivery{}, fmt.Errorf("commit notification send authorization: %w", err)
	}
	return AuthorizedDelivery{DomainDelivery: snapshot.Delivery}, nil
}

func lockDeliveryAuthorizationSnapshot(
	ctx context.Context,
	tx pgx.Tx,
	deliveryID uuid.UUID,
) (deliveryAuthorizationSnapshot, error) {
	query, args, err := storage.Psql.Select(
		"d.id", "d.event_id", "d.occurrence_id", "d.rule_version_id", "d.template_version_id",
		"d.channel_config_id", "d.channel", "d.dispatch_kind", "d.sequence_no", "d.recipient_type",
		"d.address_ciphertext", "d.address_key_version", "d.recipient_fingerprint", "d.flow_state",
		"d.delivery_result", "d.available_at", "d.next_attempt_at", "d.locked_by", "d.locked_at",
		"d.lease_expires_at", "d.occurrence_version", "d.schedule_generation", "d.provider_message_id",
		"d.origin_delivery_id", "d.suppression_reason", "d.maintenance_window_id", "d.failure_reason",
		"d.accepted_at", "d.completed_at", "d.created_at", "d.updated_at",
		"o.status", "o.last_applied_version", "o.schedule_generation", "c.enabled",
	).Column("COALESCE(h.circuit_state, 'closed')").
		From("notification_deliveries d").
		Join("notification_occurrences o ON o.occurrence_id = d.occurrence_id").
		Join("notification_channel_configs c ON c.id = d.channel_config_id").
		LeftJoin("notification_channel_health h ON h.channel_config_id = c.id").
		Where(sq.Eq{"d.id": deliveryID}).Suffix("FOR UPDATE OF d").ToSql()
	if err != nil {
		return deliveryAuthorizationSnapshot{}, fmt.Errorf("build notification send authorization lock: %w", err)
	}
	var snapshot deliveryAuthorizationSnapshot
	delivery := &snapshot.Delivery
	if err := tx.QueryRow(ctx, query, args...).Scan(
		&delivery.ID, &delivery.EventID, &delivery.OccurrenceID, &delivery.RuleVersionID,
		&delivery.TemplateVersionID, &delivery.ChannelConfigID, &delivery.Channel, &delivery.DispatchKind,
		&delivery.SequenceNo, &delivery.RecipientType, &delivery.AddressCiphertext, &delivery.AddressKeyVersion,
		&delivery.RecipientFingerprint, &delivery.FlowState, &delivery.DeliveryResult, &delivery.AvailableAt,
		&delivery.NextAttemptAt, &delivery.LockedBy, &delivery.LockedAt, &delivery.LeaseExpiresAt,
		&delivery.OccurrenceVersion, &delivery.ScheduleGeneration, &delivery.ProviderMessageID,
		&delivery.OriginDeliveryID, &delivery.SuppressionReason, &delivery.MaintenanceWindowID,
		&delivery.FailureReason, &delivery.AcceptedAt, &delivery.CompletedAt, &delivery.CreatedAt,
		&delivery.UpdatedAt, &snapshot.OccurrenceStatus, &snapshot.OccurrenceVersion,
		&snapshot.OccurrenceGeneration, &snapshot.ChannelEnabled, &snapshot.CircuitState,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return snapshot, fmt.Errorf("lock notification delivery for send authorization: %w", err)
		}
		return snapshot, fmt.Errorf("scan notification send authorization: %w", err)
	}
	return snapshot, nil
}

func rejectUnauthorizedDelivery(
	ctx context.Context,
	tx pgx.Tx,
	deliveryID uuid.UUID,
	rejection *deliveryAuthorizationRejection,
	now time.Time,
) error {
	update := storage.Psql.Update("notification_deliveries").
		Set("flow_state", rejection.flowState).
		Set("locked_by", nil).Set("locked_at", nil).Set("lease_expires_at", nil).
		Set("updated_at", now).
		Where(sq.Eq{"id": deliveryID, "flow_state": "sending"})
	if rejection.flowState == "suppressed" {
		update = update.Set("suppression_reason", rejection.reason)
	} else {
		update = update.Set("failure_reason", rejection.reason)
	}
	query, args, err := update.ToSql()
	if err != nil {
		return fmt.Errorf("build rejected notification delivery update: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("reject unauthorized notification delivery: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("reject unauthorized notification delivery: delivery state changed concurrently")
	}
	return nil
}
