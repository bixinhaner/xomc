package kpi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/model"
)

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// PgKPIRepository implements KPIRepository using TimescaleDB.
type PgKPIRepository struct {
	pool *pgxpool.Pool
}

// NewPgKPIRepository creates a new PostgreSQL-backed KPI repository.
func NewPgKPIRepository(pool *pgxpool.Pool) *PgKPIRepository {
	return &PgKPIRepository{pool: pool}
}

func (r *PgKPIRepository) BatchInsert(ctx context.Context, values []model.KPIValue) error {
	if len(values) == 0 {
		return nil
	}
	columns := []string{"time", "device_id", "cell_id", "kpi_name", "kpi_value", "carrier", "technology"}
	rows := make([][]interface{}, len(values))
	for i, v := range values {
		rows[i] = []interface{}{v.Time, v.DeviceID, v.CellID, v.KPIName, v.KPIValue, v.Carrier, v.Technology}
	}
	_, err := r.pool.CopyFrom(ctx, pgx.Identifier{"kpi_values"}, columns, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("copy kpi_values: %w", err)
	}
	return nil
}

func (r *PgKPIRepository) Query(ctx context.Context, filter KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	qb := psql.Select("time", "device_id", "cell_id", "kpi_name", "kpi_value", "carrier", "technology").From("kpi_values")
	countQb := psql.Select("COUNT(*)").From("kpi_values")

	if filter.DeviceID != nil {
		qb = qb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
		countQb = countQb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
	}
	if filter.CellID != nil {
		qb = qb.Where(squirrel.Eq{"cell_id": *filter.CellID})
		countQb = countQb.Where(squirrel.Eq{"cell_id": *filter.CellID})
	}
	if filter.KPIName != nil {
		qb = qb.Where(squirrel.Eq{"kpi_name": *filter.KPIName})
		countQb = countQb.Where(squirrel.Eq{"kpi_name": *filter.KPIName})
	}
	if filter.Carrier != nil {
		qb = qb.Where(squirrel.Eq{"carrier": *filter.Carrier})
		countQb = countQb.Where(squirrel.Eq{"carrier": *filter.Carrier})
	}
	if filter.Technology != nil {
		qb = qb.Where(squirrel.Eq{"technology": *filter.Technology})
		countQb = countQb.Where(squirrel.Eq{"technology": *filter.Technology})
	}
	if !filter.StartTime.IsZero() {
		qb = qb.Where(squirrel.GtOrEq{"time": filter.StartTime})
		countQb = countQb.Where(squirrel.GtOrEq{"time": filter.StartTime})
	}
	if !filter.EndTime.IsZero() {
		qb = qb.Where(squirrel.LtOrEq{"time": filter.EndTime})
		countQb = countQb.Where(squirrel.LtOrEq{"time": filter.EndTime})
	}

	countSQL, countArgs, err := countQb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count query: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count kpi_values: %w", err)
	}

	qb = qb.OrderBy("time DESC").Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))
	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build kpi query: %w", err)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query kpi_values: %w", err)
	}
	defer rows.Close()

	var items []model.KPIValue
	for rows.Next() {
		var v model.KPIValue
		if err := rows.Scan(&v.Time, &v.DeviceID, &v.CellID, &v.KPIName, &v.KPIValue, &v.Carrier, &v.Technology); err != nil {
			return nil, fmt.Errorf("scan kpi_value: %w", err)
		}
		items = append(items, v)
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgKPIRepository) ListDefinitions(ctx context.Context, carrier *model.CarrierCode, tech *model.Technology) ([]model.KPIDefinition, error) {
	qb := psql.Select("name", "display_name", "formula", "unit", "category", "carrier", "technology", "counters").
		From("kpi_definitions")
	if carrier != nil {
		qb = qb.Where(squirrel.Or{squirrel.Eq{"carrier": *carrier}, squirrel.Eq{"carrier": nil}})
	}
	if tech != nil {
		qb = qb.Where(squirrel.Or{squirrel.Eq{"technology": *tech}, squirrel.Eq{"technology": nil}})
	}
	qb = qb.OrderBy("category", "name")

	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build definitions query: %w", err)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query kpi_definitions: %w", err)
	}
	defer rows.Close()

	var items []model.KPIDefinition
	for rows.Next() {
		var d model.KPIDefinition
		var countersJSON []byte
		var ignore string // category column, not in model
		var carrierPtr, techPtr *string
		if err := rows.Scan(&d.Name, &d.DisplayName, &d.Formula, &d.Unit, &ignore, &carrierPtr, &techPtr, &countersJSON); err != nil {
			return nil, fmt.Errorf("scan kpi_definition: %w", err)
		}
		if carrierPtr != nil {
			d.Carrier = model.CarrierCode(*carrierPtr)
		}
		if techPtr != nil {
			d.Technology = model.Technology(*techPtr)
		}
		if err := json.Unmarshal(countersJSON, &d.Counters); err != nil {
			d.Counters = nil
		}
		items = append(items, d)
	}
	return items, nil
}

func (r *PgKPIRepository) SyncDefinitions(ctx context.Context, defs []model.KPIDefinition) error {
	for _, d := range defs {
		countersJSON, _ := json.Marshal(d.Counters)
		_, err := r.pool.Exec(ctx,
			`INSERT INTO kpi_definitions (name, display_name, formula, unit, category, carrier, technology, counters)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (name) DO UPDATE SET
			   display_name = EXCLUDED.display_name,
			   formula = EXCLUDED.formula,
			   unit = EXCLUDED.unit,
			   category = EXCLUDED.category,
			   carrier = EXCLUDED.carrier,
			   technology = EXCLUDED.technology,
			   counters = EXCLUDED.counters`,
			d.Name, d.DisplayName, d.Formula, d.Unit, "", nilIfEmpty(string(d.Carrier)), nilIfEmpty(string(d.Technology)), countersJSON,
		)
		if err != nil {
			return fmt.Errorf("sync kpi definition %s: %w", d.Name, err)
		}
	}
	return nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var _ KPIRepository = (*PgKPIRepository)(nil)
