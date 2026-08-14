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
	"business_type",
	"business_id",
	"dedup_key",
	"retry_count",
	"attempted_at",
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

func (r *PgHistoryRepository) GetByDedupKey(ctx context.Context, dedupKey string) (*NotificationHistory, error) {
	query, args, err := storage.Psql.Select(historyColumns...).
		From("notification_history").
		Where(sq.Eq{"dedup_key": dedupKey}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get history by dedup key SQL: %w", err)
	}
	h, err := scanHistory(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get history by dedup key: %w", err)
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
			h.AlarmID, h.BusinessType, h.BusinessID, h.DedupKey,
			h.RetryCount, h.AttemptedAt, h.SentAt, h.CreatedAt,
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

// InsertIfAbsent 原子创建带幂等键的待发送记录。返回 created=false 时，
// existing 是此前已创建的记录，调用方据其状态决定跳过或重试。
func (r *PgHistoryRepository) InsertIfAbsent(ctx context.Context, h *NotificationHistory) (*NotificationHistory, bool, error) {
	if h.DedupKey == nil || strings.TrimSpace(*h.DedupKey) == "" {
		if err := r.Insert(ctx, h); err != nil {
			return nil, false, err
		}
		return h, true, nil
	}
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now()
	}
	query, args, err := storage.Psql.Insert("notification_history").
		Columns(historyColumns...).
		Values(
			h.ID, h.TemplateID, h.Channel, h.Recipients,
			h.Subject, h.Body, h.Status, h.ErrorMessage,
			h.AlarmID, h.BusinessType, h.BusinessID, h.DedupKey,
			h.RetryCount, h.AttemptedAt, h.SentAt, h.CreatedAt,
		).
		Suffix("ON CONFLICT (dedup_key) WHERE dedup_key IS NOT NULL DO NOTHING").
		ToSql()
	if err != nil {
		return nil, false, fmt.Errorf("build insert history if absent SQL: %w", err)
	}
	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("insert history if absent: %w", err)
	}
	if cmd.RowsAffected() == 1 {
		return h, true, nil
	}
	existing, err := r.GetByDedupKey(ctx, *h.DedupKey)
	if err != nil {
		return nil, false, fmt.Errorf("load existing history after dedup conflict: %w", err)
	}
	return existing, false, nil
}

// ClaimAttempt 原子抢占一次实际 SMTP 尝试，并清理上次失败信息。
// 状态判断和更新在同一条 SQL 中完成，避免并发重试双重发信。
func (r *PgHistoryRepository) ClaimAttempt(ctx context.Context, id uuid.UUID, attemptedAt, staleBefore time.Time) (bool, error) {
	query, args, err := storage.Psql.Update("notification_history").
		Set("status", HistoryStatusPending).
		Set("attempted_at", attemptedAt).
		Set("error_message", nil).
		Set("retry_count", sq.Expr("retry_count + 1")).
		Where(sq.Eq{"id": id}).
		Where(sq.Or{
			sq.Eq{"status": HistoryStatusFailed},
			sq.And{
				sq.Eq{"status": HistoryStatusPending},
				sq.Or{
					sq.Eq{"attempted_at": nil},
					sq.LtOrEq{"attempted_at": staleBefore},
				},
			},
		}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build claim history attempt SQL: %w", err)
	}
	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("claim history attempt: %w", err)
	}
	return cmd.RowsAffected() == 1, nil
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
		&h.AlarmID, &h.BusinessType, &h.BusinessID, &h.DedupKey,
		&h.RetryCount, &h.AttemptedAt, &h.SentAt, &h.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if h.Recipients == nil {
		h.Recipients = []string{}
	}
	return &h, nil
}
