package notification

import (
	"context"
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
		Columns("id", "occurrence_id", "rule_version_id", "channel", "recipient_fingerprint", "schedule_kind", "sequence_no", "generation", "due_at", "state", "created_event_version").
		Values(schedule.ID, schedule.OccurrenceID, schedule.RuleVersionID, schedule.Channel, schedule.RecipientFingerprint, schedule.ScheduleKind, schedule.SequenceNo, schedule.Generation, schedule.DueAt, schedule.State, schedule.CreatedEventVersion).
		Suffix("ON CONFLICT (occurrence_id, rule_version_id, channel, recipient_fingerprint, schedule_kind, sequence_no, generation) DO NOTHING").
		ToSql()
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
		).
		Values(
			delivery.ID, delivery.EventID, delivery.OccurrenceID, delivery.RuleVersionID, delivery.TemplateVersionID,
			delivery.ChannelConfigID, delivery.Channel, delivery.DispatchKind, delivery.SequenceNo, delivery.RecipientType,
			delivery.AddressCiphertext, delivery.AddressKeyVersion, delivery.RecipientFingerprint, delivery.FlowState,
			delivery.DeliveryResult, delivery.AvailableAt, delivery.NextAttemptAt, delivery.OccurrenceVersion,
			delivery.ScheduleGeneration, delivery.OriginDeliveryID,
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
