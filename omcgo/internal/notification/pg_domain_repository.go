package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgInboxRepository struct{ db storage.DB }

func NewPgInboxRepository(pool *pgxpool.Pool) *PgInboxRepository {
	return &PgInboxRepository{db: storage.NewPoolDB(pool)}
}

func newPgInboxRepository(db storage.DB) *PgInboxRepository { return &PgInboxRepository{db: db} }

func (r *PgInboxRepository) InsertEvent(ctx context.Context, event *DomainEvent) (bool, error) {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.ProcessingState == "" {
		event.ProcessingState = "pending"
	}
	query, args, err := buildDomainEventInsert(event)
	if err != nil {
		return false, fmt.Errorf("build notification event insert: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("insert notification event: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func buildDomainEventInsert(event *DomainEvent) (string, []any, error) {
	return storage.Psql.Insert("notification_events").
		Columns("id", "event_id", "event_type", "occurrence_id", "alarm_version", "schema_version", "occurred_at", "payload", "processing_state").
		Values(event.ID, event.EventID, event.EventType, event.OccurrenceID, event.AlarmVersion, event.SchemaVersion, event.OccurredAt, event.Payload, event.ProcessingState).
		Suffix("ON CONFLICT (event_id) DO NOTHING").
		ToSql()
}

func (r *PgInboxRepository) GetOccurrence(ctx context.Context, occurrenceID uuid.UUID) (*DomainOccurrence, error) {
	query, args, err := storage.Psql.Select(
		"id", "occurrence_id", "last_applied_version", "schedule_generation", "status", "severity",
		"device_id", "device_sn", "carrier", "technology", "alarm_identifier", "current_snapshot",
		"raised_at", "acknowledged_at", "cleared_at", "last_event_id", "version_gap", "gap_first_seen_at",
		"created_at", "updated_at",
	).From("notification_occurrences").Where(sq.Eq{"occurrence_id": occurrenceID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification occurrence lookup: %w", err)
	}
	var occurrence DomainOccurrence
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&occurrence.ID, &occurrence.OccurrenceID, &occurrence.LastAppliedVersion, &occurrence.ScheduleGeneration,
		&occurrence.Status, &occurrence.Severity, &occurrence.DeviceID, &occurrence.DeviceSN,
		&occurrence.Carrier, &occurrence.Technology, &occurrence.AlarmIdentifier, &occurrence.CurrentSnapshot,
		&occurrence.RaisedAt, &occurrence.AcknowledgedAt, &occurrence.ClearedAt, &occurrence.LastEventID,
		&occurrence.VersionGap, &occurrence.GapFirstSeenAt, &occurrence.CreatedAt, &occurrence.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get notification occurrence: %w", err)
	}
	return &occurrence, nil
}

type PgRuleVersionRepository struct{ db storage.DB }

func NewPgRuleVersionRepository(pool *pgxpool.Pool) *PgRuleVersionRepository {
	return &PgRuleVersionRepository{db: storage.NewPoolDB(pool)}
}

func newPgRuleVersionRepository(db storage.DB) *PgRuleVersionRepository {
	return &PgRuleVersionRepository{db: db}
}

func (r *PgRuleVersionRepository) GetRuleVersion(ctx context.Context, id uuid.UUID) (*DomainRuleVersion, error) {
	query, args, err := storage.Psql.Select(
		"id", "rule_id", "version_no", "match_conditions", "policy", "created_by", "change_reason", "created_at", "published_at",
	).From("notification_rule_versions").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification rule version lookup: %w", err)
	}
	var version DomainRuleVersion
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&version.ID, &version.RuleID, &version.VersionNo, &version.MatchConditions, &version.Policy,
		&version.CreatedBy, &version.ChangeReason, &version.CreatedAt, &version.PublishedAt,
	); err != nil {
		return nil, fmt.Errorf("get notification rule version: %w", err)
	}
	return &version, nil
}

type PgTemplateVersionRepository struct{ db storage.DB }

func NewPgTemplateVersionRepository(pool *pgxpool.Pool) *PgTemplateVersionRepository {
	return &PgTemplateVersionRepository{db: storage.NewPoolDB(pool)}
}

func newPgTemplateVersionRepository(db storage.DB) *PgTemplateVersionRepository {
	return &PgTemplateVersionRepository{db: db}
}

func (r *PgTemplateVersionRepository) GetTemplateVersion(ctx context.Context, id uuid.UUID) (*DomainTemplateVersion, error) {
	query, args, err := storage.Psql.Select(
		"id", "template_id", "version_no", "channel", "language", "subject", "text_body", "html_body",
		"variables", "created_by", "change_reason", "created_at", "published_at",
	).From("notification_template_versions").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build notification template version lookup: %w", err)
	}
	var version DomainTemplateVersion
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&version.ID, &version.TemplateID, &version.VersionNo, &version.Channel, &version.Language,
		&version.Subject, &version.TextBody, &version.HTMLBody, &version.Variables, &version.CreatedBy,
		&version.ChangeReason, &version.CreatedAt, &version.PublishedAt,
	); err != nil {
		return nil, fmt.Errorf("get notification template version: %w", err)
	}
	return &version, nil
}

type PgScheduleRepository struct{ db storage.DB }

var ErrScheduleLeaseLost = errors.New("notification schedule lease is no longer owned by worker")

func NewPgScheduleRepository(pool *pgxpool.Pool) *PgScheduleRepository {
	return &PgScheduleRepository{db: storage.NewPoolDB(pool)}
}

func newPgScheduleRepository(db storage.DB) *PgScheduleRepository {
	return &PgScheduleRepository{db: db}
}

func (r *PgScheduleRepository) InsertSchedule(ctx context.Context, schedule *DomainSchedule) (bool, error) {
	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	if schedule.State == "" {
		schedule.State = "pending"
	}
	query, args, err := buildDomainScheduleInsert(schedule)
	if err != nil {
		return false, fmt.Errorf("build notification schedule insert: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("insert notification schedule: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func buildDomainScheduleInsert(schedule *DomainSchedule) (string, []any, error) {
	return storage.Psql.Insert("notification_schedules").
		Columns(
			"id", "event_id", "occurrence_id", "rule_version_id", "template_version_id", "channel_config_id",
			"channel", "dispatch_kind", "recipient_type", "address_ciphertext", "address_key_version",
			"recipient_fingerprint", "schedule_kind", "sequence_no", "generation", "due_at", "state", "created_event_version",
		).
		Values(
			schedule.ID, schedule.EventID, schedule.OccurrenceID, schedule.RuleVersionID,
			schedule.TemplateVersionID, schedule.ChannelConfigID, schedule.Channel, schedule.DispatchKind,
			schedule.RecipientType, schedule.AddressCiphertext, schedule.AddressKeyVersion,
			schedule.RecipientFingerprint, schedule.ScheduleKind, schedule.SequenceNo, schedule.Generation,
			schedule.DueAt, schedule.State, schedule.CreatedEventVersion,
		).
		Suffix("ON CONFLICT (occurrence_id, rule_version_id, channel, recipient_fingerprint, schedule_kind, sequence_no, generation) DO NOTHING").
		ToSql()
}

func (r *PgScheduleRepository) ClaimDue(ctx context.Context, request ScheduleClaimRequest) ([]DomainSchedule, error) {
	if request.WorkerID == "" {
		return nil, fmt.Errorf("claim notification schedules: worker ID is required")
	}
	if request.Now.IsZero() {
		return nil, fmt.Errorf("claim notification schedules: current time is required")
	}
	if request.LeaseDuration <= 0 {
		return nil, fmt.Errorf("claim notification schedules: lease duration must be positive")
	}
	if request.Limit <= 0 {
		return nil, fmt.Errorf("claim notification schedules: limit must be positive")
	}
	query, args, err := buildScheduleClaim(request)
	if err != nil {
		return nil, fmt.Errorf("build notification schedule claim: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("claim due notification schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]DomainSchedule, 0, request.Limit)
	for rows.Next() {
		var schedule DomainSchedule
		if err := scanDomainSchedule(rows, &schedule); err != nil {
			return nil, fmt.Errorf("scan claimed notification schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed notification schedules: %w", err)
	}
	return schedules, nil
}

func buildScheduleClaim(request ScheduleClaimRequest) (string, []any, error) {
	// Keep the nested builder on question placeholders. The outer Psql builder
	// performs the single final dollar-placeholder pass across both statements.
	due := sq.Select("id").From("notification_schedules").
		Where(sq.Or{
			sq.And{sq.Eq{"state": "pending"}, sq.LtOrEq{"due_at": request.Now}},
			sq.And{sq.Eq{"state": "claimed"}, sq.LtOrEq{"lease_expires_at": request.Now}},
		})
	if len(request.Kinds) > 0 {
		due = due.Where(sq.Eq{"schedule_kind": request.Kinds})
	}
	due = due.OrderBy("due_at", "id").Limit(uint64(request.Limit)).Suffix("FOR UPDATE SKIP LOCKED")
	return storage.Psql.Update("notification_schedules").
		Set("state", "claimed").
		Set("locked_by", request.WorkerID).
		Set("locked_at", request.Now).
		Set("lease_expires_at", request.Now.Add(request.LeaseDuration)).
		Set("updated_at", request.Now).
		Where(sq.Expr("id IN (?)", due)).
		Suffix(`RETURNING id, event_id, occurrence_id, rule_version_id, template_version_id,
            channel_config_id, channel, dispatch_kind, recipient_type, address_ciphertext,
            address_key_version, recipient_fingerprint,
            schedule_kind, sequence_no, generation, due_at, state, locked_by, locked_at,
            lease_expires_at, cancelled_at, completed_at, created_event_version, delivery_id,
            created_at, updated_at`).ToSql()
}

type scheduleScanner interface{ Scan(...any) error }

func scanDomainSchedule(row scheduleScanner, schedule *DomainSchedule) error {
	return row.Scan(
		&schedule.ID, &schedule.EventID, &schedule.OccurrenceID, &schedule.RuleVersionID,
		&schedule.TemplateVersionID, &schedule.ChannelConfigID, &schedule.Channel,
		&schedule.DispatchKind, &schedule.RecipientType, &schedule.AddressCiphertext,
		&schedule.AddressKeyVersion, &schedule.RecipientFingerprint, &schedule.ScheduleKind, &schedule.SequenceNo,
		&schedule.Generation, &schedule.DueAt, &schedule.State, &schedule.LockedBy,
		&schedule.LockedAt, &schedule.LeaseExpiresAt, &schedule.CancelledAt,
		&schedule.CompletedAt, &schedule.CreatedEventVersion, &schedule.DeliveryID,
		&schedule.CreatedAt, &schedule.UpdatedAt,
	)
}

func (r *PgScheduleRepository) MarkCompleted(ctx context.Context, id uuid.UUID, workerID string, now time.Time) error {
	return r.transitionClaimed(ctx, id, workerID, "completed", now, now, nil)
}

func (r *PgScheduleRepository) MarkCancelled(ctx context.Context, id uuid.UUID, workerID string, now time.Time) error {
	return r.transitionClaimed(ctx, id, workerID, "cancelled", now, nil, now)
}

func (r *PgScheduleRepository) Release(ctx context.Context, id uuid.UUID, workerID string, dueAt, now time.Time) error {
	query, args, err := storage.Psql.Update("notification_schedules").
		Set("state", "pending").Set("due_at", dueAt).
		Set("locked_by", nil).Set("locked_at", nil).Set("lease_expires_at", nil).
		Set("updated_at", now).
		Where(sq.Eq{"id": id, "state": "claimed", "locked_by": workerID}).
		Where(sq.Gt{"lease_expires_at": now}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification schedule release: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("release notification schedule: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("release notification schedule: %w", ErrScheduleLeaseLost)
	}
	return nil
}

func (r *PgScheduleRepository) transitionClaimed(
	ctx context.Context,
	id uuid.UUID,
	workerID string,
	state string,
	now time.Time,
	completedAt any,
	cancelledAt any,
) error {
	query, args, err := storage.Psql.Update("notification_schedules").
		Set("state", state).Set("completed_at", completedAt).Set("cancelled_at", cancelledAt).
		Set("locked_by", nil).Set("locked_at", nil).Set("lease_expires_at", nil).
		Set("updated_at", now).
		Where(sq.Eq{"id": id, "state": "claimed", "locked_by": workerID}).
		Where(sq.Gt{"lease_expires_at": now}).ToSql()
	if err != nil {
		return fmt.Errorf("build notification schedule %s transition: %w", state, err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark notification schedule %s: %w", state, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("mark notification schedule %s: %w", state, ErrScheduleLeaseLost)
	}
	return nil
}

type PgDeliveryRepository struct{ db storage.DB }

func NewPgDeliveryRepository(pool *pgxpool.Pool) *PgDeliveryRepository {
	return &PgDeliveryRepository{db: storage.NewPoolDB(pool)}
}

func newPgDeliveryRepository(db storage.DB) *PgDeliveryRepository {
	return &PgDeliveryRepository{db: db}
}

func (r *PgDeliveryRepository) InsertDelivery(ctx context.Context, delivery *DomainDelivery) (bool, error) {
	if delivery.ID == uuid.Nil {
		delivery.ID = uuid.New()
	}
	if delivery.FlowState == "" {
		delivery.FlowState = "queued"
	}
	if delivery.DeliveryResult == "" {
		delivery.DeliveryResult = "none"
	}
	now := time.Now().UTC()
	if delivery.AvailableAt.IsZero() {
		delivery.AvailableAt = now
	}
	if delivery.NextAttemptAt.IsZero() {
		delivery.NextAttemptAt = delivery.AvailableAt
	}
	query, args, err := buildDomainDeliveryInsert(delivery)
	if err != nil {
		return false, fmt.Errorf("build notification delivery insert: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("insert notification delivery: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func buildDomainDeliveryInsert(delivery *DomainDelivery) (string, []any, error) {
	return storage.Psql.Insert("notification_deliveries").
		Columns(
			"id", "event_id", "occurrence_id", "rule_version_id", "template_version_id", "channel_config_id",
			"channel", "dispatch_kind", "sequence_no", "recipient_type", "address_ciphertext",
			"address_key_version", "recipient_fingerprint", "flow_state", "delivery_result", "available_at",
			"next_attempt_at", "occurrence_version", "schedule_generation", "origin_delivery_id",
			"suppression_reason", "maintenance_window_id", "failure_reason",
		).
		Values(
			delivery.ID, delivery.EventID, delivery.OccurrenceID, delivery.RuleVersionID, delivery.TemplateVersionID,
			delivery.ChannelConfigID, delivery.Channel, delivery.DispatchKind, delivery.SequenceNo, delivery.RecipientType,
			delivery.AddressCiphertext, delivery.AddressKeyVersion, delivery.RecipientFingerprint, delivery.FlowState,
			delivery.DeliveryResult, delivery.AvailableAt, delivery.NextAttemptAt, delivery.OccurrenceVersion,
			delivery.ScheduleGeneration, delivery.OriginDeliveryID, delivery.SuppressionReason,
			delivery.MaintenanceWindowID, delivery.FailureReason,
		).
		Suffix("ON CONFLICT (event_id, dispatch_kind, sequence_no, channel, recipient_fingerprint) DO NOTHING").
		ToSql()
}

func (r *PgDeliveryRepository) InsertAttempt(ctx context.Context, attempt *DomainDeliveryAttempt) (bool, error) {
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	query, args, err := buildDomainAttemptInsert(attempt)
	if err != nil {
		return false, fmt.Errorf("build notification delivery attempt insert: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("insert notification delivery attempt: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func buildDomainAttemptInsert(attempt *DomainDeliveryAttempt) (string, []any, error) {
	return storage.Psql.Insert("notification_delivery_attempts").
		Columns("id", "delivery_id", "attempt_no", "started_at", "finished_at", "result", "error_category", "status_summary", "provider_request_id", "next_retry_at").
		Values(attempt.ID, attempt.DeliveryID, attempt.AttemptNo, attempt.StartedAt, attempt.FinishedAt, attempt.Result, attempt.ErrorCategory, attempt.StatusSummary, attempt.ProviderRequestID, attempt.NextRetryAt).
		Suffix("ON CONFLICT (delivery_id, attempt_no) DO NOTHING").
		ToSql()
}
