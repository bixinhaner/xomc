package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// SQLSTATE 分类（不引入 pgerrcode 依赖，直接用裸码常量）：
//   - sqlstateFeatureNotSupported（0A000）：TimescaleDB 对压缩 chunk 的 INSERT ... ON CONFLICT
//     抛 "invalid ON CONFLICT clause ... on compressed chunk"，归 feature_not_supported 类。
const sqlstateFeatureNotSupported = "0A000"

// ErrLateArrival 标记一次因 TimescaleDB 压缩 chunk 不允许 UPSERT 而被拒绝的迟到补传写入
// （PM 文件 end_time 落在 > compression 阈值的旧 chunk 上）。
//
// 降级语义（issue #14）：BatchInsert 命中此错误时不再硬失败把整批数据丢进 retry/DLQ，
// 而是 wrap 成本 sentinel 返回，由上层（collector）log WARN + 跳过，保证实时 PM 不被
// 历史补传阻塞。专用的迟到数据表（late-arrivals staging）是更大的设计，记入遗留。
var ErrLateArrival = errors.New("pm_metrics: late-arriving data hit compressed chunk (UPSERT not supported)")

// lateArrivalTotal 迟到补传命中压缩 chunk 被降级跳过的计数（issue #14）。
// 包级注册（DefaultRegisterer）：BatchInsert 在 pm/metrics 层即可观测，无需把 *pm.PMMetrics
// 反向注入仓库层（会造成 pm → pm/metrics 的反向依赖环）。
var lateArrivalTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "omc_pm_late_arrival_total",
	Help: "Total PM metric batches skipped because late-arriving data hit a compressed TimescaleDB chunk (UPSERT unsupported).",
})

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
	// VisibleGroups 是 #64 设备组数据权限的三态可见分组（nil=超管不过滤 / []=fail-closed 空集 /
	// [g...]=仅这些组下设备）。pm_metrics 以 device_sn 为设备键，过滤经
	// authz.ApplyDeviceSNVisibilityFilter 收口（device_sn → devices.id → device_group_members）。
	VisibleGroups []uuid.UUID
	Limit         int
	Offset        int
}

// LIMIT 边界（防 OOM）：pm_metrics 是时序大表，单次查询若无上界，恶意/误操作的宽时间窗 ×
// 多设备 × 多 counter 组合可拉出百万行直接灌满进程内存。Query() 在存储层强制收口：
//
//   - QueryRequest.Limit <= 0 → 落 DefaultQueryLimit（不再隐式"全表扫"）。
//   - QueryRequest.Limit > MaxQueryLimit → 收口到 MaxQueryLimit。
//
// MaxQueryLimit 取 100000，与既有调用方（QueryAggregated / QueryForKPI / adhoc executor）
// 的"安全上限"常量一致，不改变这些合法批量路径的行为；仅对未设上界 / 设了超大值的查询兜底。
// DefaultQueryLimit 取 1000，与 model.ListRequest 的分页 PageSize 上限同档，避免裸调 Query()
// 误拉全表。分页仍由调用方经 Offset 游标推进（见 PgCounterRepository.Query 等）。
const (
	DefaultQueryLimit = 1000
	MaxQueryLimit     = 100000
)

// clampLimit 把请求 Limit 收口到 [1, MaxQueryLimit]：<=0 落默认上界，超 Max 截到 Max。
func clampLimit(limit int) int {
	if limit <= 0 {
		return DefaultQueryLimit
	}
	if limit > MaxQueryLimit {
		return MaxQueryLimit
	}
	return limit
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

// BulkAsyncCommit is retained as a configuration compatibility knob. Sparse
// file ingestion is always durably committed because its local file ledger and
// values form one idempotency transaction.
var BulkAsyncCommit = false

// BatchInsert 批量插入 —— plain INSERT，不带 ON CONFLICT（migration 000042 删 uq_pm_metrics_natural
// 后无唯一索引可冲突）。
//
// 幂等语义已外移：
//   - 正常入库走 copy-direct 写模式（CopyIngest，每文件一次 pm_files 唯一约束 + 文件内 last-wins
//     去重），不经本方法；
//   - admin KPI 重算（kpi.CalculateAndStore）在调用本方法前 scoped DELETE 旧窗口行，保证重算幂等。
//
// 故本方法只需把行高吞吐灌库：
//   - 小批量（< batchInsertThreshold）：VALUES 多行 INSERT，一次 round-trip。
//   - 大批量（>= batchInsertThreshold）：CopyFrom 二进制协议直灌 pm_metrics（不再需要 TEMP 暂存
//     表 + INSERT...SELECT —— 那是为了在 COPY 上套 ON CONFLICT，去掉幂等后可直接 COPY，更快）。
//
// object_ldn 列 NOT NULL DEFAULT ”（migration 000171），nil 在 buildRows 统一落 ”。
//
// 附带：去掉 ON CONFLICT 后，写入 TimescaleDB 压缩 chunk 不再抛 SQLSTATE 0A000（那是 ON CONFLICT
// on compressed chunk 专有约束），迟到补传写压缩 chunk 不再被降级跳过（issue #14 写侧约束解除）。
// classifyInsertError 仍保留以兜底其它潜在错误归类。
func (r *PgRepository) BatchInsert(ctx context.Context, ms []PMMetric) error {
	if len(ms) == 0 {
		return nil
	}
	measurements, err := BuildSparseMeasurementsFromMetrics(ms)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin sparse metric insert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := writeSparseMeasurements(ctx, tx, nil, nil, measurements); err != nil {
		return classifyInsertError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return classifyInsertError(err)
	}
	return nil
}

// InsertRowsTx 在调用方给定的事务内 plain INSERT 一批 PMMetric（小批量 VALUES / 大批量 CopyFrom），
// 与 BatchInsert 同写出语义但不自建事务——供需要把 DELETE+INSERT 收进单个原子事务的"替换"路径
// 复用（如 KPI 重算 ReplaceForRecompute：同 tx 内先删旧窗口行再插新行，保证原子替换）。
func InsertRowsTx(ctx context.Context, tx pgx.Tx, ms []PMMetric) error {
	if len(ms) == 0 {
		return nil
	}
	measurements, err := BuildSparseMeasurementsFromMetrics(ms)
	if err != nil {
		return err
	}
	if err := writeSparseMeasurements(ctx, tx, nil, nil, measurements); err != nil {
		return classifyInsertError(err)
	}
	return nil
}

// classifyInsertError 把 pm_metrics 写入错误归类：
//   - TimescaleDB 压缩 chunk 拒绝 ON CONFLICT（SQLSTATE 0A000）→ 记迟到数据 metric +
//     返回 ErrLateArrival（issue #14 降级，上层跳过而非进 DLQ）。
//   - 其余错误原样 wrap。
func classifyInsertError(err error) error {
	if err == nil {
		return nil
	}
	if isLateArrivalError(err) {
		lateArrivalTotal.Inc()
		return fmt.Errorf("%w: %v", ErrLateArrival, err)
	}
	return fmt.Errorf("insert pm_metrics: %w", err)
}

// isLateArrivalError 判断 err 是否为"补传命中压缩 chunk"。
// TimescaleDB 对压缩 chunk 的 INSERT ... ON CONFLICT 抛 feature_not_supported（0A000）。
func isLateArrivalError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == sqlstateFeatureNotSupported
}

// buildQuerySQL 构造 pm_metrics 查询 SQL。LIMIT 经 clampLimit 强制收口，
// 保证任何输入（包括 Limit<=0 或超大值）都生成有上界的查询，防 OOM。
// 抽出供单测验证 LIMIT 边界，运行期由 Query 调用。
func buildQuerySQL(q QueryRequest) (string, []any, error) {
	qb := newLogicalPMSelect(q,
		"id", "device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
		"statis_type", "granularity", "time", "start_time", "end_time",
		"ingest_time", "object_ldn", "extra",
	)

	qb = applyFilters(qb, q)
	qb = qb.OrderBy("time DESC")
	qb = qb.Limit(uint64(clampLimit(q.Limit)))
	if q.Offset > 0 {
		qb = qb.Offset(uint64(q.Offset))
	}
	sql, args, err := qb.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build pm_metrics query: %w", err)
	}
	return sql, args, nil
}

// Query 按条件查询。
func (r *PgRepository) Query(ctx context.Context, q QueryRequest) ([]PMMetric, error) {
	sql, args, err := buildQuerySQL(q)
	if err != nil {
		return nil, err
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
		var metricValue jsonx.Float
		if err := rows.Scan(
			&m.ID, &m.DeviceOUI, &m.DeviceSN, &m.MetricPath, &metricType, &metricValue,
			&statis, &granularity, &m.Time, &m.StartTime, &m.EndTime,
			&m.IngestTime, &ldn, &extraBytes,
		); err != nil {
			return nil, fmt.Errorf("scan pm_metrics: %w", err)
		}
		m.MetricValue = float64(metricValue)
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
	qb := newLogicalPMSelect(q, "COUNT(*)")
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

// newLogicalPMSelect avoids expanding every member of every historical metric
// set when callers request explicit paths. The dictionary is resolved first and
// only matching IDs are tested against each candidate anchor's immutable set.
// Broad unfiltered requests retain the compatibility view.
func newLogicalPMSelect(q QueryRequest, columns ...string) squirrel.SelectBuilder {
	if len(q.MetricPaths) == 0 {
		return storage.Psql.Select(columns...).From("pm_metrics")
	}
	targeted := storage.Psql.Select(
		"md5(a.anchor_id::text || ':' || d.metric_id::text)::uuid AS id",
		"COALESCE(dev.oui,'')::text AS device_oui",
		"COALESCE(dev.serial_number,f.device_sn)::text AS device_sn",
		"d.metric_path", "d.metric_type", "v.metric_value", "d.statis_type", "a.granularity",
		`a."time"`, "a.start_time", "a.end_time",
		"COALESCE(b.committed_at,f.created_at,now()) AS ingest_time",
		"a.object_ldn",
		`jsonb_strip_nulls(jsonb_build_object(
			'device_id',a.device_dim_id::text,'counter_group',a.counter_group,
			'carrier',COALESCE(dev.carrier,f.carrier),
			'technology',COALESCE(dev.technology,f.technology))) AS extra`,
	).From("pm_measurement_anchors a").
		Join("pm_metric_sets s ON s.metric_set_id=a.metric_set_id").
		Join("pm_metric_dictionary d ON d.metric_id=ANY(s.metric_ids)").
		LeftJoin(`pm_metric_values v ON v."time"=a."time" AND v.anchor_id=a.anchor_id AND v.metric_id=d.metric_id`).
		LeftJoin("pm_files f ON f.id=a.source_file_id").
		LeftJoin("pm_ingest_batches b ON b.ingest_batch_id=a.ingest_batch_id").
		LeftJoin("device_dim dev ON dev.id=a.device_dim_id").
		Where(squirrel.Eq{"d.metric_path": q.MetricPaths})
	return storage.Psql.Select(columns...).FromSelect(targeted, "pm_metrics")
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
		qb = qb.Where(squirrel.Lt{"time": q.EndTime})
	}
	// #64 设备组数据权限：pm_metrics 以 device_sn 为设备键，按可见分组 fail-closed 收口。
	// nil（超管）不过滤；[] 直接 WHERE FALSE；[g...] 经 device_sn → devices → 组成员子查询限定。
	qb = authz.ApplyDeviceSNVisibilityFilter(qb, "device_sn", q.VisibleGroups)
	return qb
}

var _ Repository = (*PgRepository)(nil)
