package mr

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ---- column lists ----

var indicatorColumns = []string{
	"id", "indicator_name", "indicator_code", "description",
	"unit", "category", "value_range_min", "value_range_max", "created_at",
}

var mappingColumns = []string{
	"id", "device_sn", "device_name", "cell_id", "cell_name",
	"enabled", "sampling_interval", "last_collect_time",
	"total_records", "created_at", "updated_at",
}

// ======================================================================
// PgIndicatorRepository
// ======================================================================

var _ IndicatorRepository = (*PgIndicatorRepository)(nil)

// PgIndicatorRepository is a PostgreSQL implementation of IndicatorRepository.
type PgIndicatorRepository struct {
	pool *pgxpool.Pool
}

// NewPgIndicatorRepository creates a new PgIndicatorRepository.
func NewPgIndicatorRepository(pool *pgxpool.Pool) *PgIndicatorRepository {
	return &PgIndicatorRepository{pool: pool}
}

func (r *PgIndicatorRepository) List(ctx context.Context, filter IndicatorFilter) (*model.ListResponse[MRIndicator], error) {
	base := psql.Select(indicatorColumns...).From("mr_indicators")
	countBase := psql.Select("COUNT(*)").From("mr_indicators")

	if filter.Category != nil {
		base = base.Where(squirrel.Eq{"category": *filter.Category})
		countBase = countBase.Where(squirrel.Eq{"category": *filter.Category})
	}
	if filter.Keyword != nil && *filter.Keyword != "" {
		like := "%" + *filter.Keyword + "%"
		cond := squirrel.Or{
			squirrel.Like{"indicator_name": like},
			squirrel.Like{"indicator_code": like},
		}
		base = base.Where(cond)
		countBase = countBase.Where(cond)
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count mr_indicators SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mr_indicators: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list mr_indicators SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mr_indicators: %w", err)
	}
	defer rows.Close()

	var items []MRIndicator
	for rows.Next() {
		ind, err := scanIndicatorRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mr_indicator row: %w", err)
		}
		items = append(items, *ind)
	}

	if items == nil {
		items = []MRIndicator{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgIndicatorRepository) ListAll(ctx context.Context) ([]MRIndicator, error) {
	query, args, err := psql.Select(indicatorColumns...).
		From("mr_indicators").
		OrderBy("indicator_code ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list all mr_indicators SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all mr_indicators: %w", err)
	}
	defer rows.Close()

	var items []MRIndicator
	for rows.Next() {
		ind, err := scanIndicatorRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mr_indicator row: %w", err)
		}
		items = append(items, *ind)
	}

	if items == nil {
		items = []MRIndicator{}
	}
	return items, nil
}

func (r *PgIndicatorRepository) GetByCode(ctx context.Context, code string) (*MRIndicator, error) {
	query, args, err := psql.Select(indicatorColumns...).
		From("mr_indicators").
		Where(squirrel.Eq{"indicator_code": code}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mr_indicator SQL: %w", err)
	}

	ind, err := scanIndicator(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mr_indicator: %w", err)
	}
	return ind, nil
}

// ---- indicator scanning helpers ----

func scanIndicator(row pgx.Row) (*MRIndicator, error) {
	var ind MRIndicator
	err := row.Scan(
		&ind.ID, &ind.IndicatorName, &ind.IndicatorCode, &ind.Description,
		&ind.Unit, &ind.Category, &ind.ValueRangeMin, &ind.ValueRangeMax, &ind.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ind, nil
}

func scanIndicatorRow(rows pgx.Rows) (*MRIndicator, error) {
	var ind MRIndicator
	err := rows.Scan(
		&ind.ID, &ind.IndicatorName, &ind.IndicatorCode, &ind.Description,
		&ind.Unit, &ind.Category, &ind.ValueRangeMin, &ind.ValueRangeMax, &ind.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ind, nil
}

// ======================================================================
// PgMappingRepository
// ======================================================================

var _ MappingRepository = (*PgMappingRepository)(nil)

// PgMappingRepository is a PostgreSQL implementation of MappingRepository.
type PgMappingRepository struct {
	pool *pgxpool.Pool
}

// NewPgMappingRepository creates a new PgMappingRepository.
func NewPgMappingRepository(pool *pgxpool.Pool) *PgMappingRepository {
	return &PgMappingRepository{pool: pool}
}

func (r *PgMappingRepository) List(ctx context.Context, filter MappingFilter) (*model.ListResponse[MRDeviceMapping], error) {
	base := psql.Select(mappingColumns...).From("mr_device_mappings")
	countBase := psql.Select("COUNT(*)").From("mr_device_mappings")

	if filter.DeviceSN != nil && *filter.DeviceSN != "" {
		base = base.Where(squirrel.Eq{"device_sn": *filter.DeviceSN})
		countBase = countBase.Where(squirrel.Eq{"device_sn": *filter.DeviceSN})
	}
	if filter.Enabled != nil {
		base = base.Where(squirrel.Eq{"enabled": *filter.Enabled})
		countBase = countBase.Where(squirrel.Eq{"enabled": *filter.Enabled})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count mr_device_mappings SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mr_device_mappings: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list mr_device_mappings SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mr_device_mappings: %w", err)
	}
	defer rows.Close()

	var items []MRDeviceMapping
	for rows.Next() {
		m, err := scanMappingRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mr_device_mapping row: %w", err)
		}
		items = append(items, *m)
	}

	if items == nil {
		items = []MRDeviceMapping{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgMappingRepository) Update(ctx context.Context, mapping *MRDeviceMapping) error {
	query, args, err := psql.Update("mr_device_mappings").
		Set("device_sn", mapping.DeviceSN).
		Set("device_name", mapping.DeviceName).
		Set("cell_id", mapping.CellID).
		Set("cell_name", mapping.CellName).
		Set("enabled", mapping.Enabled).
		Set("sampling_interval", mapping.SamplingInterval).
		Set("last_collect_time", mapping.LastCollectTime).
		Set("total_records", mapping.TotalRecords).
		Where(squirrel.Eq{"id": mapping.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update mr_device_mapping SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mr_device_mapping: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgMappingRepository) ToggleEnabled(ctx context.Context, id uuid.UUID, enabled bool) (*MRDeviceMapping, error) {
	query, args, err := psql.Update("mr_device_mappings").
		Set("enabled", enabled).
		Where(squirrel.Eq{"id": id}).
		Suffix("RETURNING " + joinMappingColumns(mappingColumns)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build toggle mr_device_mapping SQL: %w", err)
	}

	m, err := scanMapping(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("toggle mr_device_mapping: %w", err)
	}
	return m, nil
}

// ---- mapping scanning helpers ----

func scanMapping(row pgx.Row) (*MRDeviceMapping, error) {
	var m MRDeviceMapping
	err := row.Scan(
		&m.ID, &m.DeviceSN, &m.DeviceName, &m.CellID, &m.CellName,
		&m.Enabled, &m.SamplingInterval, &m.LastCollectTime,
		&m.TotalRecords, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func scanMappingRow(rows pgx.Rows) (*MRDeviceMapping, error) {
	var m MRDeviceMapping
	err := rows.Scan(
		&m.ID, &m.DeviceSN, &m.DeviceName, &m.CellID, &m.CellName,
		&m.Enabled, &m.SamplingInterval, &m.LastCollectTime,
		&m.TotalRecords, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func joinMappingColumns(cols []string) string {
	result := ""
	for i, c := range cols {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}
