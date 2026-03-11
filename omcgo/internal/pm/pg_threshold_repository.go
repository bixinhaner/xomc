package pm

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var thresholdColumns = []string{
	"id", "kpi_name", "carrier", "technology",
	"warning_threshold", "minor_threshold", "major_threshold", "critical_threshold",
	"comparison", "enabled", "description", "created_at", "updated_at",
}

var _ ThresholdRepository = (*PgThresholdRepository)(nil)

// PgThresholdRepository implements ThresholdRepository using PostgreSQL.
// Thresholds are configuration data, so this uses pgPool (regular Postgres),
// not tsPool (TimescaleDB).
type PgThresholdRepository struct {
	pool *pgxpool.Pool
}

// NewPgThresholdRepository creates a new PostgreSQL-backed threshold repository.
func NewPgThresholdRepository(pool *pgxpool.Pool) *PgThresholdRepository {
	return &PgThresholdRepository{pool: pool}
}

func (r *PgThresholdRepository) Create(ctx context.Context, threshold *KPIThreshold) error {
	if threshold.ID == uuid.Nil {
		threshold.ID = uuid.New()
	}
	now := time.Now()
	threshold.CreatedAt = now
	threshold.UpdatedAt = now

	query, args, err := psql.Insert("kpi_thresholds").
		Columns("id", "kpi_name", "carrier", "technology",
			"warning_threshold", "minor_threshold", "major_threshold", "critical_threshold",
			"comparison", "enabled", "description", "created_at", "updated_at").
		Values(threshold.ID, threshold.KPIName, threshold.Carrier, threshold.Technology,
			threshold.WarningThreshold, threshold.MinorThreshold, threshold.MajorThreshold, threshold.CriticalThreshold,
			threshold.Comparison, threshold.Enabled, threshold.Description, threshold.CreatedAt, threshold.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert kpi_thresholds SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert kpi_thresholds: %w", err)
	}
	return nil
}

func (r *PgThresholdRepository) GetByID(ctx context.Context, id uuid.UUID) (*KPIThreshold, error) {
	query, args, err := psql.Select(thresholdColumns...).
		From("kpi_thresholds").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get kpi_thresholds SQL: %w", err)
	}

	t, err := scanThreshold(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get kpi_thresholds: %w", err)
	}
	return t, nil
}

func (r *PgThresholdRepository) Update(ctx context.Context, threshold *KPIThreshold) error {
	threshold.UpdatedAt = time.Now()

	query, args, err := psql.Update("kpi_thresholds").
		Set("kpi_name", threshold.KPIName).
		Set("carrier", threshold.Carrier).
		Set("technology", threshold.Technology).
		Set("warning_threshold", threshold.WarningThreshold).
		Set("minor_threshold", threshold.MinorThreshold).
		Set("major_threshold", threshold.MajorThreshold).
		Set("critical_threshold", threshold.CriticalThreshold).
		Set("comparison", threshold.Comparison).
		Set("enabled", threshold.Enabled).
		Set("description", threshold.Description).
		Set("updated_at", threshold.UpdatedAt).
		Where(sq.Eq{"id": threshold.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update kpi_thresholds SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update kpi_thresholds: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgThresholdRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("kpi_thresholds").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete kpi_thresholds SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete kpi_thresholds: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgThresholdRepository) List(ctx context.Context, filter KPIThresholdFilter) (*model.ListResponse[KPIThreshold], error) {
	builder := psql.Select(thresholdColumns...).From("kpi_thresholds")
	countBuilder := psql.Select("COUNT(*)").From("kpi_thresholds")

	builder = applyThresholdFilters(builder, filter)
	countBuilder = applyThresholdFilters(countBuilder, filter)

	// Count total
	countSQL, countArgs, _ := countBuilder.ToSql()
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count kpi_thresholds: %w", err)
	}

	// Apply sorting and pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	builder = builder.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list kpi_thresholds SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list kpi_thresholds: %w", err)
	}
	defer rows.Close()

	var items []KPIThreshold
	for rows.Next() {
		t, err := scanThresholdRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *t)
	}

	if items == nil {
		items = []KPIThreshold{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func applyThresholdFilters(qb sq.SelectBuilder, f KPIThresholdFilter) sq.SelectBuilder {
	if f.KPIName != nil && *f.KPIName != "" {
		qb = qb.Where(sq.Eq{"kpi_name": *f.KPIName})
	}
	if f.Carrier != nil && *f.Carrier != "" {
		qb = qb.Where(sq.Eq{"carrier": *f.Carrier})
	}
	if f.Technology != nil && *f.Technology != "" {
		qb = qb.Where(sq.Eq{"technology": *f.Technology})
	}
	if f.Enabled != nil {
		qb = qb.Where(sq.Eq{"enabled": *f.Enabled})
	}
	return qb
}

func scanThreshold(row pgx.Row) (*KPIThreshold, error) {
	var t KPIThreshold
	err := row.Scan(
		&t.ID, &t.KPIName, &t.Carrier, &t.Technology,
		&t.WarningThreshold, &t.MinorThreshold, &t.MajorThreshold, &t.CriticalThreshold,
		&t.Comparison, &t.Enabled, &t.Description, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func scanThresholdRow(rows pgx.Rows) (*KPIThreshold, error) {
	var t KPIThreshold
	err := rows.Scan(
		&t.ID, &t.KPIName, &t.Carrier, &t.Technology,
		&t.WarningThreshold, &t.MinorThreshold, &t.MajorThreshold, &t.CriticalThreshold,
		&t.Comparison, &t.Enabled, &t.Description, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan kpi_thresholds row: %w", err)
	}
	return &t, nil
}
