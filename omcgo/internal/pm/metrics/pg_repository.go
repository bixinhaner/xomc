package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// Repository 是 pm_metrics 的统一持久层接口。
//
// 替代旧的 pm/counter + pm/kpi 双 repository 实现。
// 旧 pm/counter/PgCounterRepository 和 pm/kpi/PgKPIRepository 已改写为本接口的薄包装
// （PMCounter / KPIValue ↔ PMMetric 转换）。
type Repository interface {
	Insert(ctx context.Context, m PMMetric) error
	BatchInsert(ctx context.Context, ms []PMMetric) error
	Query(ctx context.Context, q QueryRequest) ([]PMMetric, error)
	Count(ctx context.Context, q QueryRequest) (int64, error)
}

// QueryRequest 是 Query / Count 的查询条件。空切片 / nil 字段忽略。
// 设备过滤用 (DeviceOUI, DeviceSN) 双键（TR-069 标准）。
type QueryRequest struct {
	DeviceOUIs  []string // 与 DeviceSNs 配对（按位置 i 对应同一设备 (ouis[i], sns[i])）
	DeviceSNs   []string
	MetricPaths []string
	MetricType  *MetricType
	Granularity Granularity // 必填 — 客户端按粒度查询（'15min' / 'hourly' / ...）
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
}

// PgRepository 是 Repository 的 TimescaleDB 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository 创建一个新的 pm_metrics 持久层。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// Insert 单条插入。
func (r *PgRepository) Insert(ctx context.Context, m PMMetric) error {
	return r.BatchInsert(ctx, []PMMetric{m})
}

// BatchInsert 批量插入。使用自然键 ON CONFLICT DO UPDATE 实现补传幂等：
//
//	相同 (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn)
//	再次写入时更新 metric_value（取新值）+ ingest_time（取 NOW()）。
//
// object_ldn 列 NOT NULL DEFAULT ''（migration 000171），nil 在此统一落 ''
// 避免 UNIQUE 索引 NULL ≠ NULL 破坏幂等语义；同时让同文件多 cell 同 counter_name
// 不再因为缺 ldn 维度而撞 ON CONFLICT 二次命中（BUG-6）。
//
// 注意 TimescaleDB 压缩 chunk 不允许 UPSERT；本逻辑假设新写入只命中 7d 内的未压缩 chunk。
// 补传 > 7d 旧数据将报错（业务约束：补传窗口受限于 retention/compression policy）。
func (r *PgRepository) BatchInsert(ctx context.Context, ms []PMMetric) error {
	if len(ms) == 0 {
		return nil
	}
	sql, args, err := buildBatchInsertSQL(ms)
	if err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("insert pm_metrics: %w", err)
	}
	return nil
}

// pmMetricsUpsertSuffix 自然键 ON CONFLICT 子句。
// 列顺序与 migration 000171 中 uq_pm_metrics_natural 一致（不一致 PG 会按列集合匹配，但保持顺序便于人读）。
const pmMetricsUpsertSuffix = "ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn) " +
	"DO UPDATE SET metric_value = EXCLUDED.metric_value, ingest_time = NOW()"

// buildBatchInsertSQL 构造 pm_metrics 批量 INSERT SQL（含 ON CONFLICT 子句）。
// 抽出供单测使用，运行期由 BatchInsert 调用。
func buildBatchInsertSQL(ms []PMMetric) (string, []any, error) {
	ib := storage.Psql.Insert("pm_metrics").Columns(
		"id", "device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
		"statis_type", "granularity", "time", "start_time", "end_time",
		"ingest_time", "object_ldn", "extra",
	)
	for _, m := range ms {
		id := m.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		ingest := m.IngestTime
		if ingest.IsZero() {
			ingest = time.Now()
		}
		t := m.Time
		if t.IsZero() {
			t = m.EndTime
		}
		var statis interface{}
		if m.StatisType != nil {
			statis = string(*m.StatisType)
		}
		// object_ldn 列 NOT NULL DEFAULT ''（migration 000171）：
		// nil/空指针统一落 ''，与 UNIQUE 索引语义一致。
		ldn := ""
		if m.ObjectLDN != nil {
			ldn = *m.ObjectLDN
		}
		var extra interface{}
		if len(m.Extra) > 0 {
			b, err := json.Marshal(m.Extra)
			if err != nil {
				return "", nil, fmt.Errorf("marshal pm_metrics extra: %w", err)
			}
			extra = b
		}
		ib = ib.Values(
			id, m.DeviceOUI, m.DeviceSN, m.MetricPath, string(m.MetricType), m.MetricValue,
			statis, string(m.Granularity), t, m.StartTime, m.EndTime,
			ingest, ldn, extra,
		)
	}
	ib = ib.Suffix(pmMetricsUpsertSuffix)
	sql, args, err := ib.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build pm_metrics insert: %w", err)
	}
	return sql, args, nil
}

// Query 按条件查询。
func (r *PgRepository) Query(ctx context.Context, q QueryRequest) ([]PMMetric, error) {
	qb := storage.Psql.Select(
		"id", "device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
		"statis_type", "granularity", "time", "start_time", "end_time",
		"ingest_time", "object_ldn", "extra",
	).From("pm_metrics")

	qb = applyFilters(qb, q)
	qb = qb.OrderBy("time DESC")
	if q.Limit > 0 {
		qb = qb.Limit(uint64(q.Limit))
	}
	if q.Offset > 0 {
		qb = qb.Offset(uint64(q.Offset))
	}
	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build pm_metrics query: %w", err)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query pm_metrics: %w", err)
	}
	defer rows.Close()

	var items []PMMetric
	for rows.Next() {
		var m PMMetric
		var statis, ldn *string
		var extraBytes []byte
		var metricType, granularity string
		if err := rows.Scan(
			&m.ID, &m.DeviceOUI, &m.DeviceSN, &m.MetricPath, &metricType, &m.MetricValue,
			&statis, &granularity, &m.Time, &m.StartTime, &m.EndTime,
			&m.IngestTime, &ldn, &extraBytes,
		); err != nil {
			return nil, fmt.Errorf("scan pm_metrics: %w", err)
		}
		m.MetricType = MetricType(metricType)
		m.Granularity = Granularity(granularity)
		if statis != nil {
			st := StatisType(*statis)
			m.StatisType = &st
		}
		m.ObjectLDN = ldn
		if len(extraBytes) > 0 {
			extra := make(map[string]any)
			if err := json.Unmarshal(extraBytes, &extra); err == nil {
				m.Extra = extra
			}
		}
		items = append(items, m)
	}
	return items, nil
}

// Count 按条件统计。
func (r *PgRepository) Count(ctx context.Context, q QueryRequest) (int64, error) {
	qb := storage.Psql.Select("COUNT(*)").From("pm_metrics")
	qb = applyFilters(qb, q)
	sql, args, err := qb.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build pm_metrics count: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, sql, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count pm_metrics: %w", err)
	}
	return total, nil
}

func applyFilters(qb squirrel.SelectBuilder, q QueryRequest) squirrel.SelectBuilder {
	// 设备双键过滤：(oui[i], sn[i]) 配对成 OR 条件
	// 如 ([A,B], [X,Y]) → WHERE (oui='A' AND sn='X') OR (oui='B' AND sn='Y')
	// 单 OUIs / 单 SNs 仅一边过滤；位长度不等时按 min(len) 截断配对
	if len(q.DeviceOUIs) > 0 && len(q.DeviceSNs) > 0 {
		n := len(q.DeviceOUIs)
		if len(q.DeviceSNs) < n {
			n = len(q.DeviceSNs)
		}
		or := squirrel.Or{}
		for i := 0; i < n; i++ {
			or = append(or, squirrel.And{
				squirrel.Eq{"device_oui": q.DeviceOUIs[i]},
				squirrel.Eq{"device_sn": q.DeviceSNs[i]},
			})
		}
		qb = qb.Where(or)
	} else if len(q.DeviceOUIs) > 0 {
		qb = qb.Where(squirrel.Eq{"device_oui": q.DeviceOUIs})
	} else if len(q.DeviceSNs) > 0 {
		qb = qb.Where(squirrel.Eq{"device_sn": q.DeviceSNs})
	}
	if len(q.MetricPaths) > 0 {
		qb = qb.Where(squirrel.Eq{"metric_path": q.MetricPaths})
	}
	if q.MetricType != nil {
		qb = qb.Where(squirrel.Eq{"metric_type": string(*q.MetricType)})
	}
	if q.Granularity != "" {
		qb = qb.Where(squirrel.Eq{"granularity": string(q.Granularity)})
	}
	if !q.StartTime.IsZero() {
		qb = qb.Where(squirrel.GtOrEq{"time": q.StartTime})
	}
	if !q.EndTime.IsZero() {
		qb = qb.Where(squirrel.LtOrEq{"time": q.EndTime})
	}
	return qb
}

var _ Repository = (*PgRepository)(nil)
