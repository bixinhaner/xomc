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
			h.AlarmID, h.RetryCount, h.SentAt, h.CreatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history SQL: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
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
		&h.AlarmID, &h.RetryCount, &h.SentAt, &h.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if h.Recipients == nil {
		h.Recipients = []string{}
	}
	return &h, nil
}
