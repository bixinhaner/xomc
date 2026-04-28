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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// templateColumns must include every column read/written by the repository.
// All five SQL operations (List/Get/GetByName/Create/Update) share this list to
// prevent the kind of column-mismatch bug seen in W1.5.
var templateColumns = []string{
	"id",
	"name",
	"channel",
	"language",
	"subject",
	"body",
	"variables",
	"enabled",
	"created_at",
	"updated_at",
}

var templateAllowedSortColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"name":       true,
	"channel":    true,
}

// Compile-time interface check.
var _ TemplateRepository = (*PgTemplateRepository)(nil)

// PgTemplateRepository is a PostgreSQL-backed TemplateRepository.
type PgTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewPgTemplateRepository constructs a PgTemplateRepository.
func NewPgTemplateRepository(pool *pgxpool.Pool) *PgTemplateRepository {
	return &PgTemplateRepository{pool: pool}
}

// List returns a paginated list of templates matching the filter.
func (r *PgTemplateRepository) List(ctx context.Context, filter NotificationTemplateFilter) (*model.ListResponse[NotificationTemplate], error) {
	base := storage.Psql.Select(templateColumns...).From("notification_templates")
	countBase := storage.Psql.Select("COUNT(*)").From("notification_templates")

	if filter.Channel != nil && *filter.Channel != "" {
		base = base.Where(sq.Eq{"channel": *filter.Channel})
		countBase = countBase.Where(sq.Eq{"channel": *filter.Channel})
	}
	if filter.Language != nil && *filter.Language != "" {
		base = base.Where(sq.Eq{"language": *filter.Language})
		countBase = countBase.Where(sq.Eq{"language": *filter.Language})
	}
	if filter.Enabled != nil {
		base = base.Where(sq.Eq{"enabled": *filter.Enabled})
		countBase = countBase.Where(sq.Eq{"enabled": *filter.Enabled})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count templates SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count templates: %w", err)
	}

	sortBy := "created_at"
	if filter.SortBy != "" && templateAllowedSortColumns[filter.SortBy] {
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
		return nil, fmt.Errorf("build list templates SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	items := make([]NotificationTemplate, 0)
	for rows.Next() {
		tpl, err := scanTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("scan template row: %w", err)
		}
		items = append(items, *tpl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate template rows: %w", err)
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// GetByID returns a template by its UUID.
func (r *PgTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*NotificationTemplate, error) {
	query, args, err := storage.Psql.Select(templateColumns...).
		From("notification_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get template SQL: %w", err)
	}

	tpl, err := scanTemplate(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get template: %w", err)
	}
	return tpl, nil
}

// GetByName returns a template by its unique name.
func (r *PgTemplateRepository) GetByName(ctx context.Context, name string) (*NotificationTemplate, error) {
	query, args, err := storage.Psql.Select(templateColumns...).
		From("notification_templates").
		Where(sq.Eq{"name": name}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get template by name SQL: %w", err)
	}

	tpl, err := scanTemplate(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get template by name: %w", err)
	}
	return tpl, nil
}

// Create inserts a new template, returning ErrAlreadyExists if the name conflicts.
func (r *PgTemplateRepository) Create(ctx context.Context, tpl *NotificationTemplate) error {
	if tpl.ID == uuid.Nil {
		tpl.ID = uuid.New()
	}
	now := time.Now()
	if tpl.CreatedAt.IsZero() {
		tpl.CreatedAt = now
	}
	tpl.UpdatedAt = now
	if tpl.Variables == nil {
		tpl.Variables = []string{}
	}

	query, args, err := storage.Psql.Insert("notification_templates").
		Columns(templateColumns...).
		Values(
			tpl.ID, tpl.Name, tpl.Channel, tpl.Language,
			tpl.Subject, tpl.Body, tpl.Variables, tpl.Enabled,
			tpl.CreatedAt, tpl.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert template SQL: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return commonerrors.ErrAlreadyExists
		}
		return fmt.Errorf("insert template: %w", err)
	}
	return nil
}

// Update patches an existing template (full replacement of all fields).
func (r *PgTemplateRepository) Update(ctx context.Context, tpl *NotificationTemplate) error {
	tpl.UpdatedAt = time.Now()
	if tpl.Variables == nil {
		tpl.Variables = []string{}
	}

	query, args, err := storage.Psql.Update("notification_templates").
		Set("name", tpl.Name).
		Set("channel", tpl.Channel).
		Set("language", tpl.Language).
		Set("subject", tpl.Subject).
		Set("body", tpl.Body).
		Set("variables", tpl.Variables).
		Set("enabled", tpl.Enabled).
		Set("updated_at", tpl.UpdatedAt).
		Where(sq.Eq{"id": tpl.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update template SQL: %w", err)
	}

	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return commonerrors.ErrAlreadyExists
		}
		return fmt.Errorf("update template: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// Delete removes a template; returns ErrNotFound when the row does not exist.
func (r *PgTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("notification_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete template SQL: %w", err)
	}

	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// scanTemplate reads a single row into a NotificationTemplate. Column order MUST
// match templateColumns above.
func scanTemplate(row interface{ Scan(dest ...any) error }) (*NotificationTemplate, error) {
	var tpl NotificationTemplate
	err := row.Scan(
		&tpl.ID, &tpl.Name, &tpl.Channel, &tpl.Language,
		&tpl.Subject, &tpl.Body, &tpl.Variables, &tpl.Enabled,
		&tpl.CreatedAt, &tpl.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if tpl.Variables == nil {
		tpl.Variables = []string{}
	}
	return &tpl, nil
}
