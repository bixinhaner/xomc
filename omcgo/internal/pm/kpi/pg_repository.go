package kpi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// PgKPIRepository 是 KPIRepository 的 TimescaleDB 实现。
//
// T-0164-P3 / G3 改造：kpi_values 表已合入 pm_metrics（metric_type='kpi'）。
// BatchInsert / Query 改走 pm/metrics.Repository（KPIValue ↔ PMMetric 字段转换）。
// ListDefinitions / SyncDefinitions 仍走 kpi_definitions 表（独立元数据表，与 G3 无关）。
//
// DeviceSN 过渡：旧 KPIValue.DeviceID 是 UUID，新 PMMetric.DeviceSN 是 text。本 wrapper
// 内 DeviceSN = DeviceID.String()（即 UUID 的字符串形式），与 counter 包做法一致。
type PgKPIRepository struct {
	pool        *pgxpool.Pool
	metricsRepo metrics.Repository
}

// NewPgKPIRepository creates a new PostgreSQL-backed KPI repository.
func NewPgKPIRepository(pool *pgxpool.Pool) *PgKPIRepository {
	return &PgKPIRepository{
		pool:        pool,
		metricsRepo: metrics.NewPgRepository(pool),
	}
}

func (r *PgKPIRepository) BatchInsert(ctx context.Context, values []model.KPIValue) error {
	if len(values) == 0 {
		return nil
	}
	ms := make([]metrics.PMMetric, 0, len(values))
	for _, v := range values {
		ms = append(ms, kpiValueToMetric(v))
	}
	return r.metricsRepo.BatchInsert(ctx, ms)
}

func (r *PgKPIRepository) Query(ctx context.Context, filter KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	q := kpiFilterToMetricsQuery(filter)
	total, err := r.metricsRepo.Count(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("count kpi: %w", err)
	}
	q.Limit = filter.Limit()
	q.Offset = filter.Offset()
	ms, err := r.metricsRepo.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query kpi: %w", err)
	}
	items := make([]model.KPIValue, 0, len(ms))
	for _, m := range ms {
		v := metricToKPIValue(m)
		// 按 carrier / technology 二级过滤（pm_metrics 没有结构化列，存在 extra JSONB 里）
		if filter.Carrier != nil && v.Carrier != *filter.Carrier {
			continue
		}
		if filter.Technology != nil && v.Technology != *filter.Technology {
			continue
		}
		items = append(items, v)
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgKPIRepository) ListDefinitions(ctx context.Context, carrier *model.CarrierCode, tech *model.Technology) ([]model.KPIDefinition, error) {
	qb := storage.Psql.Select("name", "display_name", "formula", "unit", "category", "carrier", "technology", "counters").
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
		var ignore string
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

// --- 字段转换辅助 ---

func kpiValueToMetric(v model.KPIValue) metrics.PMMetric {
	extra := map[string]any{}
	if v.Carrier != "" {
		extra["carrier"] = string(v.Carrier)
	}
	if v.Technology != "" {
		extra["technology"] = string(v.Technology)
	}
	var ldn *string
	if v.CellID != "" {
		s := v.CellID
		ldn = &s
	}
	return metrics.PMMetric{
		DeviceSN:    v.DeviceID.String(),
		MetricPath:  v.KPIName,
		MetricType:  metrics.MetricTypeKPI,
		MetricValue: v.KPIValue,
		Granularity: metrics.Granularity15Min,
		Time:        v.Time,
		StartTime:   v.Time,
		EndTime:     v.Time,
		ObjectLDN:   ldn,
		Extra:       extra,
	}
}

func metricToKPIValue(m metrics.PMMetric) model.KPIValue {
	v := model.KPIValue{
		Time:     m.Time,
		KPIName:  m.MetricPath,
		KPIValue: m.MetricValue,
	}
	if m.ObjectLDN != nil {
		v.CellID = *m.ObjectLDN
	}
	if did, err := uuid.Parse(m.DeviceSN); err == nil {
		v.DeviceID = did
	}
	if m.Extra != nil {
		if c, ok := m.Extra["carrier"].(string); ok {
			v.Carrier = model.CarrierCode(c)
		}
		if t, ok := m.Extra["technology"].(string); ok {
			v.Technology = model.Technology(t)
		}
	}
	return v
}

func kpiFilterToMetricsQuery(filter KPIFilter) metrics.QueryRequest {
	mt := metrics.MetricTypeKPI
	q := metrics.QueryRequest{
		MetricType:  &mt,
		Granularity: metrics.Granularity15Min,
		StartTime:   filter.StartTime,
		EndTime:     filter.EndTime,
	}
	if filter.DeviceID != nil {
		q.DeviceSNs = []string{filter.DeviceID.String()}
	}
	if filter.KPIName != nil {
		q.MetricPaths = []string{*filter.KPIName}
	}
	return q
}

var _ KPIRepository = (*PgKPIRepository)(nil)
