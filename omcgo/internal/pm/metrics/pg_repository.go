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

// pmMetricsColumns 是 pm_metrics 的写入列序（COPY 与 INSERT 共用，顺序必须与
// buildRows / buildBatchInsertSQL 一致）。
var pmMetricsColumns = []string{
	"id", "device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
	"statis_type", "granularity", "time", "start_time", "end_time",
	"ingest_time", "object_ldn", "extra",
}

// batchInsertThreshold 是切换到 COPY 暂存表路径的批量阈值。
//
// 小批量（单文件 KPI 计算、补单条）走原 VALUES 多行 INSERT —— 一次 round-trip、
// 直接带 ON CONFLICT、无建表开销，对几行到几十行最快。大批量（PM 文件解析后的
// 几百~几千条 counter）走 COPY 暂存表 + INSERT...SELECT...ON CONFLICT —— COPY 二进制
// 协议批量灌库远快于把成千上万个占位符塞进单条 VALUES（旧实现随行数线性膨胀 SQL 文本
// 与参数数组，本质 O(n) 文本拼接 + 单条巨型语句解析）。
//
// 阈值取 50：低于此用 VALUES 省去建表两条额外语句；高于此 COPY 的吞吐优势盖过建表开销。
const batchInsertThreshold = 50

// BatchInsert 批量插入。两条路径共享同一套自然键 ON CONFLICT DO UPDATE 补传幂等语义：
//
//		相同 (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn)
//		再次写入时更新 metric_value（取新值）+ ingest_time（取 NOW()）。
//
//	  - 小批量（< batchInsertThreshold）：VALUES 多行 INSERT，直接带 ON CONFLICT 子句。
//	  - 大批量（>= batchInsertThreshold）：COPY 进会话级 TEMP 暂存表，再
//	    INSERT ... SELECT ... ON CONFLICT 一次性 upsert（issue #14）。
//	    COPY 不支持 ON CONFLICT，故用 TEMP 表两段式既拿 COPY 吞吐又不丢补传幂等。
//
// object_ldn 列 NOT NULL DEFAULT ”（migration 000171），nil 在此统一落 ”
// 避免 UNIQUE 索引 NULL ≠ NULL 破坏幂等语义；同时让同文件多 cell 同 counter_name
// 不再因为缺 ldn 维度而撞 ON CONFLICT 二次命中（BUG-6）。
//
// TimescaleDB 压缩 chunk 不允许 ON CONFLICT：补传 end_time 落在 > compression 阈值的旧
// chunk 时 PG 抛 SQLSTATE 0A000。两条路径都把该错误识别为 ErrLateArrival，记 metric +
// 返回 sentinel（issue #14 降级），由 collector log WARN + 跳过，实时 PM 不被历史补传阻塞。
func (r *PgRepository) BatchInsert(ctx context.Context, ms []PMMetric) error {
	if len(ms) == 0 {
		return nil
	}
	if len(ms) >= batchInsertThreshold {
		return r.batchInsertCopy(ctx, ms)
	}
	return r.batchInsertValues(ctx, ms)
}

// batchInsertValues 走 VALUES 多行 INSERT（小批量路径）。
func (r *PgRepository) batchInsertValues(ctx context.Context, ms []PMMetric) error {
	sql, args, err := buildBatchInsertSQL(ms)
	if err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return classifyInsertError(err)
	}
	return nil
}

// batchInsertCopy 走 COPY 暂存表 + INSERT...SELECT...ON CONFLICT（大批量路径，issue #14）。
//
// 步骤（同一连接 / 事务内，保证 TEMP 表可见且 ON COMMIT DROP 自动清理）：
//  1. CREATE TEMP TABLE ... LIKE pm_metrics（仅列定义，不含约束 / 索引，COPY 不被 ON CONFLICT 限制）
//  2. CopyFrom 二进制批量灌入暂存表
//  3. INSERT INTO pm_metrics SELECT * FROM 暂存表 ON CONFLICT (...) DO UPDATE（拿幂等）
//  4. COMMIT（ON COMMIT DROP 清掉暂存表）
func (r *PgRepository) batchInsertCopy(ctx context.Context, ms []PMMetric) error {
	rows, err := buildRows(ms)
	if err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin pm_metrics copy tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// LIKE 仅复制列定义（不含 DEFAULTS / 约束 / 索引）：暂存表是纯缓冲区，
	// id / ingest_time / object_ldn 等已在 buildRows 里落好值，无需表级 DEFAULT。
	// ON COMMIT DROP 保证事务结束自动回收，不污染连接后续复用。
	if _, err := tx.Exec(ctx,
		`CREATE TEMP TABLE pm_metrics_copy_buf (LIKE pm_metrics) ON COMMIT DROP`,
	); err != nil {
		return fmt.Errorf("create pm_metrics temp buffer: %w", err)
	}

	if _, err := tx.CopyFrom(ctx,
		pgx.Identifier{"pm_metrics_copy_buf"},
		pmMetricsColumns,
		pgx.CopyFromRows(rows),
	); err != nil {
		return fmt.Errorf("copy pm_metrics buffer: %w", err)
	}

	upsertSQL := `INSERT INTO pm_metrics (` + joinCols(pmMetricsColumns) + `) ` +
		`SELECT ` + joinCols(pmMetricsColumns) + ` FROM pm_metrics_copy_buf ` +
		pmMetricsUpsertSuffix
	if _, err := tx.Exec(ctx, upsertSQL); err != nil {
		return classifyInsertError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return classifyInsertError(err)
	}
	return nil
}

// joinCols 把列名用逗号拼成 SQL 列清单（列名是包内常量，非用户输入，无注入风险）。
func joinCols(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out
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

// pmMetricsUpsertSuffix 自然键 ON CONFLICT 子句。
// 列顺序与 migration 000171 中 uq_pm_metrics_natural 一致（不一致 PG 会按列集合匹配，但保持顺序便于人读）。
const pmMetricsUpsertSuffix = "ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn) " +
	"DO UPDATE SET metric_value = EXCLUDED.metric_value, ingest_time = NOW()"

// metricRowValues 把单条 PMMetric 归一化为与 pmMetricsColumns 等长、等序的列值数组。
// 落值规则（id 缺省生成 / ingest 缺省 NOW / time 缺省取 end_time / object_ldn nil → ” /
// extra map → JSONB bytes）在 VALUES INSERT 与 COPY 两条路径间共享，保证两路写出的行
// 完全一致。
func metricRowValues(m PMMetric) ([]any, error) {
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
			return nil, fmt.Errorf("marshal pm_metrics extra: %w", err)
		}
		extra = b
	}
	return []any{
		id, m.DeviceOUI, m.DeviceSN, m.MetricPath, string(m.MetricType), m.MetricValue,
		statis, string(m.Granularity), t, m.StartTime, m.EndTime,
		ingest, ldn, extra,
	}, nil
}

// buildRows 把 PMMetric 切片转为 COPY 用的二维行数组（列序 = pmMetricsColumns）。
func buildRows(ms []PMMetric) ([][]any, error) {
	rows := make([][]any, 0, len(ms))
	for _, m := range ms {
		vals, err := metricRowValues(m)
		if err != nil {
			return nil, err
		}
		rows = append(rows, vals)
	}
	return rows, nil
}

// buildBatchInsertSQL 构造 pm_metrics 批量 INSERT SQL（含 ON CONFLICT 子句）。
// 抽出供单测使用，运行期由 batchInsertValues 调用（小批量路径）。
func buildBatchInsertSQL(ms []PMMetric) (string, []any, error) {
	ib := storage.Psql.Insert("pm_metrics").Columns(pmMetricsColumns...)
	for _, m := range ms {
		vals, err := metricRowValues(m)
		if err != nil {
			return "", nil, err
		}
		ib = ib.Values(vals...)
	}
	ib = ib.Suffix(pmMetricsUpsertSuffix)
	sql, args, err := ib.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build pm_metrics insert: %w", err)
	}
	return sql, args, nil
}

// buildQuerySQL 构造 pm_metrics 查询 SQL。LIMIT 经 clampLimit 强制收口，
// 保证任何输入（包括 Limit<=0 或超大值）都生成有上界的查询，防 OOM。
// 抽出供单测验证 LIMIT 边界，运行期由 Query 调用。
func buildQuerySQL(q QueryRequest) (string, []any, error) {
	qb := storage.Psql.Select(
		"id", "device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
		"statis_type", "granularity", "time", "start_time", "end_time",
		"ingest_time", "object_ldn", "extra",
	).From("pm_metrics")

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
	// #64 设备组数据权限：pm_metrics 以 device_sn 为设备键，按可见分组 fail-closed 收口。
	// nil（超管）不过滤；[] 直接 WHERE FALSE；[g...] 经 device_sn → devices → 组成员子查询限定。
	qb = authz.ApplyDeviceSNVisibilityFilter(qb, "device_sn", q.VisibleGroups)
	return qb
}

var _ Repository = (*PgRepository)(nil)
