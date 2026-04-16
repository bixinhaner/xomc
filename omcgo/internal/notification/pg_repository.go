package notification

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// Sort whitelist to prevent SQL injection in ORDER BY.
var notifAllowedSortColumns = map[string]bool{
	"created_at": true,
	"priority":   true,
	"type":       true,
}

var notifColumns = []string{
	"id", "user_id", "type", "priority",
	"title", "content", "link", "sender",
	"is_read", "read_at", "created_at",
}

// Compile-time interface check.
var _ Repository = (*PgRepository)(nil)

// PgRepository is a PostgreSQL implementation of the notification Repository.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository creates a new PgRepository.
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// List returns a paginated list of notifications matching the filter.
func (r *PgRepository) List(ctx context.Context, filter NotificationFilter) (*model.ListResponse[Notification], error) {
	base := storage.Psql.Select(notifColumns...).From("notifications")
	countBase := storage.Psql.Select("COUNT(*)").From("notifications")

	// user_id is required
	base = base.Where(sq.Eq{"user_id": filter.UserID})
	countBase = countBase.Where(sq.Eq{"user_id": filter.UserID})

	if filter.Type != nil {
		base = base.Where(sq.Eq{"type": *filter.Type})
		countBase = countBase.Where(sq.Eq{"type": *filter.Type})
	}
	if filter.IsRead != nil {
		base = base.Where(sq.Eq{"is_read": *filter.IsRead})
		countBase = countBase.Where(sq.Eq{"is_read": *filter.IsRead})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count notifications SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count notifications: %w", err)
	}

	// Pagination and ordering
	sortBy := "created_at"
	if filter.SortBy != "" && notifAllowedSortColumns[filter.SortBy] {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list notifications SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var items []Notification
	for rows.Next() {
		notif, err := scanNotification(rows)
		if err != nil {
			return nil, fmt.Errorf("scan notification row: %w", err)
		}
		items = append(items, *notif)
	}

	if items == nil {
		items = []Notification{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// GetByID returns a single notification by ID.
func (r *PgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	query, args, err := storage.Psql.Select(notifColumns...).
		From("notifications").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get notification SQL: %w", err)
	}

	notif, err := scanNotification(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get notification: %w", err)
	}
	return notif, nil
}

// Create inserts a new notification and updates the struct with DB-generated fields.
func (r *PgRepository) Create(ctx context.Context, notif *Notification) error {
	query, args, err := storage.Psql.Insert("notifications").
		Columns("user_id", "type", "priority", "title", "content", "link", "sender").
		Values(notif.UserID, notif.Type, notif.Priority, notif.Title, notif.Content, notif.Link, notif.Sender).
		Suffix("RETURNING " + joinColumns(notifColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert notification SQL: %w", err)
	}

	created, err := scanNotification(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	*notif = *created
	return nil
}

// MarkRead marks a single notification as read. It also validates user ownership.
func (r *PgRepository) MarkRead(ctx context.Context, id uuid.UUID, userID string) error {
	now := time.Now()
	query, args, err := storage.Psql.Update("notifications").
		Set("is_read", true).
		Set("read_at", now).
		Where(sq.Eq{"id": id, "user_id": userID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark read SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// MarkAllRead marks all unread notifications for a user as read.
func (r *PgRepository) MarkAllRead(ctx context.Context, userID string) error {
	now := time.Now()
	query, args, err := storage.Psql.Update("notifications").
		Set("is_read", true).
		Set("read_at", now).
		Where(sq.Eq{"user_id": userID, "is_read": false}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark all read SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

// GetUnreadCount returns the number of unread notifications for a user.
func (r *PgRepository) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	query, args, err := storage.Psql.Select("COUNT(*)").
		From("notifications").
		Where(sq.Eq{"user_id": userID, "is_read": false}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build unread count SQL: %w", err)
	}

	var count int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

// Delete removes a notification. It also validates user ownership.
func (r *PgRepository) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	query, args, err := storage.Psql.Delete("notifications").
		Where(sq.Eq{"id": id, "user_id": userID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete notification SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// ---- scan helpers ----

func scanNotification(row interface{ Scan(dest ...any) error }) (*Notification, error) {
	var n Notification
	err := row.Scan(
		&n.ID, &n.UserID, &n.Type, &n.Priority,
		&n.Title, &n.Content, &n.Link, &n.Sender,
		&n.IsRead, &n.ReadAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func joinColumns(cols []string) string {
	result := ""
	for i, c := range cols {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}
