package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// historyColumns must include every column read/written by the repository.
var historyColumns = []string{
	"id",
	"template_id",
	"channel",
	"recipients",
	"subject",
	"body",
	"status",
	"error_message",
	"alarm_id",
	"source_type",
	"source_id",
	"event_id",
	"correlation_id",
	"retry_count",
	"sent_at",
	"created_at",
}

var historyAllowedSortColumns = map[string]bool{
	"created_at": true,
	"sent_at":    true,
	"channel":    true,
	"status":     true,
}

// Compile-time interface check.
var _ HistoryRepository = (*PgHistoryRepository)(nil)

// PgHistoryRepository is a PostgreSQL-backed HistoryRepository.
type PgHistoryRepository struct {
	pool *pgxpool.Pool
}

// NewPgHistoryRepository constructs a PgHistoryRepository.
func NewPgHistoryRepository(pool *pgxpool.Pool) *PgHistoryRepository {
	return &PgHistoryRepository{pool: pool}
}

// List returns paginated notification history matching the filter.
func (r *PgHistoryRepository) List(ctx context.Context, filter NotificationHistoryFilter) (*model.ListResponse[NotificationHistory], error) {
	base := storage.Psql.Select(historyColumns...).From("notification_history")
	countBase := storage.Psql.Select("COUNT(*)").From("notification_history")

	if filter.Channel != nil && *filter.Channel != "" {
		base = base.Where(sq.Eq{"channel": *filter.Channel})
		countBase = countBase.Where(sq.Eq{"channel": *filter.Channel})
	}
	if filter.Status != nil && *filter.Status != "" {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.TemplateID != nil {
		base = base.Where(sq.Eq{"template_id": *filter.TemplateID})
		countBase = countBase.Where(sq.Eq{"template_id": *filter.TemplateID})
	}
	if filter.AlarmID != nil {
		base = base.Where(sq.Eq{"alarm_id": *filter.AlarmID})
		countBase = countBase.Where(sq.Eq{"alarm_id": *filter.AlarmID})
	}
	if filter.SourceType != nil && *filter.SourceType != "" {
		base = base.Where(sq.Eq{"source_type": *filter.SourceType})
		countBase = countBase.Where(sq.Eq{"source_type": *filter.SourceType})
	}
	if filter.SourceID != nil {
		base = base.Where(sq.Eq{"source_id": *filter.SourceID})
		countBase = countBase.Where(sq.Eq{"source_id": *filter.SourceID})
	}
	if filter.EventID != nil {
		base = base.Where(sq.Eq{"event_id": *filter.EventID})
		countBase = countBase.Where(sq.Eq{"event_id": *filter.EventID})
	}
	if filter.CorrelationID != nil && *filter.CorrelationID != "" {
		base = base.Where(sq.Eq{"correlation_id": *filter.CorrelationID})
		countBase = countBase.Where(sq.Eq{"correlation_id": *filter.CorrelationID})
	}
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count history SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count history: %w", err)
	}

	sortBy := "created_at"
	if filter.SortBy != "" && historyAllowedSortColumns[filter.SortBy] {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if strings.EqualFold(filter.SortDir, "asc") {
		sortDir = "ASC"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list history SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	items := make([]NotificationHistory, 0)
	for rows.Next() {
		h, err := scanHistory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan history row: %w", err)
		}
		items = append(items, *h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history rows: %w", err)
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// GetByID returns a single history entry.
func (r *PgHistoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*NotificationHistory, error) {
	query, args, err := storage.Psql.Select(historyColumns...).
		From("notification_history").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get history SQL: %w", err)
	}

	h, err := scanHistory(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get history: %w", err)
	}
	return h, nil
}

// Insert creates a history entry; intended for internal dispatchers.
func (r *PgHistoryRepository) Insert(ctx context.Context, h *NotificationHistory) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now()
	}
	if h.Recipients == nil {
		h.Recipients = []string{}
	}

	query, args, err := storage.Psql.Insert("notification_history").
		Columns(historyColumns...).
		Values(
			h.ID, h.TemplateID, h.Channel, h.Recipients,
			h.Subject, h.Body, h.Status, h.ErrorMessage,
			h.AlarmID, strings.TrimSpace(h.SourceType), h.SourceID, h.EventID, strings.TrimSpace(h.CorrelationID),
			h.RetryCount, h.SentAt, h.CreatedAt,
		).
		Suffix("ON CONFLICT (event_id) WHERE event_id IS NOT NULL DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history SQL: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}

// BackfillDeviceAccessDecisions closes the notification-history gap for
// decisions committed before the decision subscriber was deployed. The
// decision UUID is also the event UUID, so the backfill and live subscriber
// remain idempotent through the existing unique event index.
func (r *PgHistoryRepository) BackfillDeviceAccessDecisions(ctx context.Context) (int64, error) {
	selectRows := storage.Psql.Select(
		"decision.id", "NULL::uuid", "'system'", "ARRAY['device-access-operators']::text[]",
		`CASE decision.new_state
			WHEN 'accepted' THEN 'DEVICE_ACCESS_ACCEPTED'
			WHEN 'rejected' THEN 'DEVICE_ACCESS_REJECTED'
			WHEN 'revoked' THEN 'DEVICE_ACCESS_REVOKED'
			ELSE 'DEVICE_ACCESS_REVIEW_REQUIRED'
		END`,
		`jsonb_build_object(
			'carrier', decision.carrier,
			'serial_number', decision.serial_number,
			'device_id', decision.device_id,
			'candidate_id', decision.candidate_id,
			'state', decision.new_state,
			'reason_code', decision.reason_code,
			'policy_version_id', decision.policy_version_id,
			'decision_version', decision.decision_version
		)::text`,
		"'not_configured'", "NULL::text", "NULL::uuid", "'device_access_decision'",
		"decision.id", "decision.id", "COALESCE(decision.trigger_event_id, '')", "0", "NULL::timestamptz", "decision.occurred_at",
	).From("device_access_decisions decision").
		Where(sq.Eq{"decision.new_state": []string{"accepted", "rejected", "review_required", "revoked"}}).
		Where(`NOT EXISTS (
			SELECT 1 FROM notification_history history WHERE history.event_id = decision.id
		)`)
	query, args, err := storage.Psql.Insert("notification_history").Columns(historyColumns...).
		Select(selectRows).
		Suffix("ON CONFLICT (event_id) WHERE event_id IS NOT NULL DO NOTHING").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build device access decision notification backfill: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("backfill device access decision notification history: %w", err)
	}
	return tag.RowsAffected(), nil
}

// UpdateStatus updates the delivery status of an existing history entry.
func (r *PgHistoryRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, sentAt *time.Time) error {
	q := storage.Psql.Update("notification_history").
		Set("status", status).
		Set("error_message", errorMessage).
		Set("sent_at", sentAt).
		Where(sq.Eq{"id": id})

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build update history status SQL: %w", err)
	}

	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update history status: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// scanHistory reads a single row into a NotificationHistory. Column order MUST
// match historyColumns above.
func scanHistory(row interface{ Scan(dest ...any) error }) (*NotificationHistory, error) {
	var h NotificationHistory
	err := row.Scan(
		&h.ID, &h.TemplateID, &h.Channel, &h.Recipients,
		&h.Subject, &h.Body, &h.Status, &h.ErrorMessage,
		&h.AlarmID, &h.SourceType, &h.SourceID, &h.EventID, &h.CorrelationID,
		&h.RetryCount, &h.SentAt, &h.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if h.Recipients == nil {
		h.Recipients = []string{}
	}
	return &h, nil
}
