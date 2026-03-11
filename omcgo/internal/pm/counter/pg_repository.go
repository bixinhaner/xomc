package counter

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/model"
)

// PgCounterRepository implements CounterRepository using TimescaleDB.
type PgCounterRepository struct {
	pool *pgxpool.Pool
}

// NewPgCounterRepository creates a new PostgreSQL-backed counter repository.
func NewPgCounterRepository(pool *pgxpool.Pool) *PgCounterRepository {
	return &PgCounterRepository{pool: pool}
}

func (r *PgCounterRepository) BatchInsert(ctx context.Context, counters []model.PMCounter) error {
	if len(counters) == 0 {
		return nil
	}
	columns := []string{"time", "device_id", "cell_id", "counter_group", "counter_name", "counter_value", "granularity"}
	rows := make([][]interface{}, len(counters))
	for i, c := range counters {
		rows[i] = []interface{}{c.Time, c.DeviceID, c.CellID, c.CounterGroup, c.CounterName, c.CounterValue, c.Granularity}
	}
	_, err := r.pool.CopyFrom(ctx, pgx.Identifier{"pm_counters"}, columns, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("copy pm_counters: %w", err)
	}
	return nil
}

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

func applyCounterFilters(qb squirrel.SelectBuilder, filter CounterFilter) squirrel.SelectBuilder {
	if filter.DeviceID != nil {
		qb = qb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
	}
	if filter.CellID != nil {
		qb = qb.Where(squirrel.Eq{"cell_id": *filter.CellID})
	}
	if filter.CounterGroup != nil {
		qb = qb.Where(squirrel.Eq{"counter_group": *filter.CounterGroup})
	}
	if filter.CounterName != nil {
		qb = qb.Where(squirrel.Eq{"counter_name": *filter.CounterName})
	}
	if !filter.StartTime.IsZero() {
		qb = qb.Where(squirrel.GtOrEq{"time": filter.StartTime})
	}
	if !filter.EndTime.IsZero() {
		qb = qb.Where(squirrel.LtOrEq{"time": filter.EndTime})
	}
	return qb
}

func (r *PgCounterRepository) Query(ctx context.Context, filter CounterFilter) (*model.ListResponse[model.PMCounter], error) {
	// Count
	countQb := applyCounterFilters(psql.Select("COUNT(*)").From("pm_counters"), filter)
	countSQL, countArgs, err := countQb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count query: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count pm_counters: %w", err)
	}

	// Data
	qb := applyCounterFilters(
		psql.Select("time", "device_id", "cell_id", "counter_group", "counter_name", "counter_value", "granularity").From("pm_counters"),
		filter,
	)
	sortBy := "time"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
	}
	qb = qb.OrderBy(fmt.Sprintf("%s %s", sortBy, sortDir)).
		Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))

	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query pm_counters: %w", err)
	}
	defer rows.Close()

	var items []model.PMCounter
	for rows.Next() {
		var c model.PMCounter
		if err := rows.Scan(&c.Time, &c.DeviceID, &c.CellID, &c.CounterGroup, &c.CounterName, &c.CounterValue, &c.Granularity); err != nil {
			return nil, fmt.Errorf("scan pm_counter: %w", err)
		}
		items = append(items, c)
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgCounterRepository) QueryAggregated(ctx context.Context, filter CounterFilter) ([]AggregatedCounter, error) {
	qb := psql.Select("bucket", "device_id", "cell_id", "counter_group", "counter_name",
		"sum_value", "avg_value", "min_value", "max_value", "sample_count").From("pm_counters_hourly")

	if filter.DeviceID != nil {
		qb = qb.Where(squirrel.Eq{"device_id": *filter.DeviceID})
	}
	if filter.CellID != nil {
		qb = qb.Where(squirrel.Eq{"cell_id": *filter.CellID})
	}
	if filter.CounterGroup != nil {
		qb = qb.Where(squirrel.Eq{"counter_group": *filter.CounterGroup})
	}
	if filter.CounterName != nil {
		qb = qb.Where(squirrel.Eq{"counter_name": *filter.CounterName})
	}
	if !filter.StartTime.IsZero() {
		qb = qb.Where(squirrel.GtOrEq{"bucket": filter.StartTime})
	}
	if !filter.EndTime.IsZero() {
		qb = qb.Where(squirrel.LtOrEq{"bucket": filter.EndTime})
	}
	qb = qb.OrderBy("bucket DESC").Limit(1000)

	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build aggregated query: %w", err)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query pm_counters_hourly: %w", err)
	}
	defer rows.Close()

	var items []AggregatedCounter
	for rows.Next() {
		var a AggregatedCounter
		if err := rows.Scan(&a.Bucket, &a.DeviceID, &a.CellID, &a.CounterGroup, &a.CounterName,
			&a.SumValue, &a.AvgValue, &a.MinValue, &a.MaxValue, &a.SampleCount); err != nil {
			return nil, fmt.Errorf("scan aggregated counter: %w", err)
		}
		items = append(items, a)
	}
	return items, nil
}

func (r *PgCounterRepository) QueryForKPI(ctx context.Context, deviceID uuid.UUID, cellID string, counterNames []string, startTime, endTime time.Time) (map[string]float64, error) {
	if len(counterNames) == 0 {
		return nil, nil
	}
	qb := psql.Select("counter_name", "SUM(counter_value) as total").
		From("pm_counters").
		Where(squirrel.Eq{"device_id": deviceID}).
		Where(squirrel.Eq{"counter_name": counterNames}).
		Where(squirrel.GtOrEq{"time": startTime}).
		Where(squirrel.LtOrEq{"time": endTime}).
		GroupBy("counter_name")
	if cellID != "" {
		qb = qb.Where(squirrel.Eq{"cell_id": cellID})
	}
	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build kpi query: %w", err)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query counters for kpi: %w", err)
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scan kpi counter: %w", err)
		}
		result[name] = value
	}
	duration := endTime.Sub(startTime)
	if duration > 0 {
		result["period_seconds"] = duration.Seconds()
	}
	return result, nil
}

var _ CounterRepository = (*PgCounterRepository)(nil)
