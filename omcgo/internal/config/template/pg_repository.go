package template

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var templateColumns = []string{
	"id", "name", "carrier", "technology", "product_class",
	"template_type", "parameters", "priority", "version", "active",
	"description", "created_at", "updated_at",
}

var allowedSortColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"name":       true,
	"carrier":    true,
	"technology": true,
	"priority":   true,
}

// PgConfigTemplateRepository implements ConfigTemplateRepository using PostgreSQL.
type PgConfigTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewPgConfigTemplateRepository creates a new PgConfigTemplateRepository.
func NewPgConfigTemplateRepository(pool *pgxpool.Pool) *PgConfigTemplateRepository {
	return &PgConfigTemplateRepository{pool: pool}
}

func (r *PgConfigTemplateRepository) Create(ctx context.Context, t *ConfigTemplate) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	query, args, err := psql.Insert("config_templates").
		Columns(templateColumns...).
		Values(
			t.ID, t.Name, t.Carrier, t.Technology, nullableString(t.ProductClass),
			t.TemplateType, t.Parameters, t.Priority, t.Version, t.Active,
			nullableString(t.Description), t.CreatedAt, t.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert template SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert template: %w", err)
	}
	return nil
}

func (r *PgConfigTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*ConfigTemplate, error) {
	query, args, err := psql.Select(templateColumns...).
		From("config_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select template SQL: %w", err)
	}

	t, err := scanTemplate(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *PgConfigTemplateRepository) Update(ctx context.Context, t *ConfigTemplate) error {
	t.UpdatedAt = time.Now()

	query, args, err := psql.Update("config_templates").
		Set("name", t.Name).
		Set("carrier", t.Carrier).
		Set("technology", t.Technology).
		Set("product_class", nullableString(t.ProductClass)).
		Set("template_type", t.TemplateType).
		Set("parameters", t.Parameters).
		Set("priority", t.Priority).
		Set("version", t.Version).
		Set("active", t.Active).
		Set("description", nullableString(t.Description)).
		Set("updated_at", t.UpdatedAt).
		Where(sq.Eq{"id": t.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update template SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update template %s: %w", t.ID, commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgConfigTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("config_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete template SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete template %s: %w", id, commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgConfigTemplateRepository) List(ctx context.Context, filter ConfigTemplateFilter) (*model.ListResponse[ConfigTemplate], error) {
	pred := sq.And{}
	if filter.Carrier != "" {
		pred = append(pred, sq.Eq{"carrier": filter.Carrier})
	}
	if filter.Technology != "" {
		pred = append(pred, sq.Eq{"technology": filter.Technology})
	}
	if filter.ProductClass != "" {
		pred = append(pred, sq.Eq{"product_class": filter.ProductClass})
	}
	if filter.TemplateType != "" {
		pred = append(pred, sq.Eq{"template_type": filter.TemplateType})
	}
	if filter.Active != nil {
		pred = append(pred, sq.Eq{"active": *filter.Active})
	}

	// Count.
	countBuilder := psql.Select("COUNT(*)").From("config_templates")
	if len(pred) > 0 {
		countBuilder = countBuilder.Where(pred)
	}
	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count template SQL: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count templates: %w", err)
	}

	limit := filter.ListRequest.Limit()
	offset := filter.ListRequest.Offset()
	page := filter.ListRequest.Page
	if page < 1 {
		page = 1
	}

	queryBuilder := psql.Select(templateColumns...).
		From("config_templates").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	if len(pred) > 0 {
		queryBuilder = queryBuilder.Where(pred)
	}

	sortCol := "created_at"
	if filter.SortBy != "" && allowedSortColumns[filter.SortBy] {
		sortCol = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
	}
	queryBuilder = queryBuilder.OrderBy(sortCol + " " + sortDir)

	querySQL, queryArgs, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list template SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, querySQL, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	items, err := scanTemplates(rows)
	if err != nil {
		return nil, err
	}

	return model.NewListResponse(items, total, page, limit), nil
}

func (r *PgConfigTemplateRepository) FindByCarrierTech(ctx context.Context, carrier model.CarrierCode, tech model.Technology, templateType TemplateType) ([]ConfigTemplate, error) {
	query, args, err := psql.Select(templateColumns...).
		From("config_templates").
		Where(sq.And{
			sq.Eq{"carrier": carrier},
			sq.Eq{"technology": tech},
			sq.Eq{"template_type": templateType},
			sq.Eq{"active": true},
		}).
		OrderBy("priority DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find templates SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find templates by carrier/tech: %w", err)
	}
	defer rows.Close()

	return scanTemplates(rows)
}

// FindBestMatch finds the highest-priority active template matching the given criteria.
// Priority order: carrier + tech + product_class > carrier + tech (no product_class).
func (r *PgConfigTemplateRepository) FindBestMatch(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, templateType TemplateType) (*ConfigTemplate, error) {
	// Try specific match first (with product_class).
	if productClass != "" {
		query, args, err := psql.Select(templateColumns...).
			From("config_templates").
			Where(sq.And{
				sq.Eq{"carrier": carrier},
				sq.Eq{"technology": tech},
				sq.Eq{"product_class": productClass},
				sq.Eq{"template_type": templateType},
				sq.Eq{"active": true},
			}).
			OrderBy("priority DESC").
			Limit(1).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("build find best match SQL: %w", err)
		}

		t, err := scanTemplate(r.pool.QueryRow(ctx, query, args...))
		if err == nil {
			return t, nil
		}
		if err != commonerrors.ErrNotFound {
			return nil, err
		}
	}

	// Fallback: carrier + tech, no product_class constraint.
	query, args, err := psql.Select(templateColumns...).
		From("config_templates").
		Where(sq.And{
			sq.Eq{"carrier": carrier},
			sq.Eq{"technology": tech},
			sq.Eq{"template_type": templateType},
			sq.Eq{"active": true},
		}).
		Where("product_class IS NULL").
		OrderBy("priority DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find default match SQL: %w", err)
	}

	t, err := scanTemplate(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return t, nil
}

// --- Helper functions ---

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func scanTemplate(row pgx.Row) (*ConfigTemplate, error) {
	var t ConfigTemplate
	var (
		productClass sql.NullString
		description  sql.NullString
		parameters   []byte
	)

	err := row.Scan(
		&t.ID, &t.Name, &t.Carrier, &t.Technology, &productClass,
		&t.TemplateType, &parameters, &t.Priority, &t.Version, &t.Active,
		&description, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan template row: %w", err)
	}

	t.Parameters = json.RawMessage(parameters)
	if productClass.Valid {
		t.ProductClass = productClass.String
	}
	if description.Valid {
		t.Description = description.String
	}

	return &t, nil
}

func scanTemplates(rows pgx.Rows) ([]ConfigTemplate, error) {
	var items []ConfigTemplate
	for rows.Next() {
		var t ConfigTemplate
		var (
			productClass sql.NullString
			description  sql.NullString
			parameters   []byte
		)

		err := rows.Scan(
			&t.ID, &t.Name, &t.Carrier, &t.Technology, &productClass,
			&t.TemplateType, &parameters, &t.Priority, &t.Version, &t.Active,
			&description, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan template row: %w", err)
		}

		t.Parameters = json.RawMessage(parameters)
		if productClass.Valid {
			t.ProductClass = productClass.String
		}
		if description.Valid {
			t.Description = description.String
		}

		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate template rows: %w", err)
	}
	return items, nil
}

// Compile-time interface compliance check.
var _ ConfigTemplateRepository = (*PgConfigTemplateRepository)(nil)
