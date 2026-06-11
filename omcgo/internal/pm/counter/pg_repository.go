package counter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// PgCounterRepository 是 CounterRepository 的 TimescaleDB 实现。
//
// T-0164-P3 / G3 改造：pm_counters 表已合入 pm_metrics（metric_type='counter'）。
// 本实现成为 pm/metrics.Repository 的薄包装：
//   - BatchInsert / Query：PMCounter ↔ PMMetric 字段转换
//   - QueryForKPI：查 pm_metrics 原始数据后在内存里做 SUM 聚合（替代旧 SQL GROUP BY；
//     PM 单文件维度 counter 数量有限，开销可接受）
//   - QueryAggregated：原走 pm_counters_hourly 物化视图（已删），G3 阶段降级为查 15min 粒度
//     的 pm_metrics 并在内存按 cell+counter_name+bucket(端点对齐 hour) 聚合。G5 上线后会
//     直接路由到 hourly 聚合表替代本实现。
//
// 设备唯一标识（T-0164-P3 fix）：按 TR-069 标准用 (OUI, DeviceSN) 双键。
//   - BatchInsert：上游 PMCounter.OUI / DeviceSN 已由 collector 填充，wrapper 直接用
//   - Query / QueryForKPI：handler 仍按 device_id(uuid) 接收，wrapper 反查 devices 表
//     拿 (oui, serial_number) 后再查 pm_metrics；开销一次/请求可接受
//
// 全系统级切换见 docs/project/plan-T-0165-system-wide-oui-sn-migration.md。
type PgCounterRepository struct {
	pool        *pgxpool.Pool
	metricsRepo metrics.Repository
}

// NewPgCounterRepository creates a new PostgreSQL-backed counter repository.
func NewPgCounterRepository(pool *pgxpool.Pool) *PgCounterRepository {
	return &PgCounterRepository{
		pool:        pool,
		metricsRepo: metrics.NewPgRepository(pool),
	}
}

func (r *PgCounterRepository) BatchInsert(ctx context.Context, counters []model.PMCounter) error {
	if len(counters) == 0 {
		return nil
	}
	ms := make([]metrics.PMMetric, 0, len(counters))
	for _, c := range counters {
		ms = append(ms, counterToMetric(c))
	}
	return r.metricsRepo.BatchInsert(ctx, ms)
}

func (r *PgCounterRepository) Query(ctx context.Context, filter CounterFilter) (*model.ListResponse[model.PMCounter], error) {
	q, err := r.counterFilterToMetricsQuery(ctx, filter)
	if err != nil {
		return nil, err
	}
	total, err := r.metricsRepo.Count(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("count counters: %w", err)
	}
	q.Limit = filter.Limit()
	q.Offset = filter.Offset()
	ms, err := r.metricsRepo.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query counters: %w", err)
	}
	items := make([]model.PMCounter, 0, len(ms))
	for _, m := range ms {
		items = append(items, metricToCounter(m))
	}
	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// QueryAggregated 旧物化视图 pm_counters_hourly 已删（T-0164-P3 / G3）。G3 阶段降级为：
// 查 15min 粒度 pm_metrics 后内存按 hour bucket 聚合。G5 上线 hourly 聚合表后由路由层替换。
// G5 之前调用方应理解：数据量大时不要随便查（小时跨度 × cells × counters）。
func (r *PgCounterRepository) QueryAggregated(ctx context.Context, filter CounterFilter) ([]AggregatedCounter, error) {
	q, err := r.counterFilterToMetricsQuery(ctx, filter)
	if err != nil {
		return nil, err
	}
	q.Limit = 100000 // 安全上限
	ms, err := r.metricsRepo.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query metrics for aggregation: %w", err)
	}
	return aggregateInMemory(ms, filter.CounterGroup), nil
}

// QueryForKPI 查指定 device + cell 在时间窗口内的 counter 求和，返回 counter_name → SUM(value)。
// G3 阶段改为查 pm_metrics WHERE metric_type='counter' + 内存聚合（替代旧 SQL GROUP BY SUM）。
// 单文件 KPI 计算的 counter 集合通常 ≤ 几十，内存聚合开销可接受。
// T-0164-P3 fix: 按 (oui, sn) 双键过滤，wrapper 反查 devices 表拿到。
func (r *PgCounterRepository) QueryForKPI(ctx context.Context, deviceID uuid.UUID, cellID string, counterNames []string, startTime, endTime time.Time) (map[string]float64, error) {
	if len(counterNames) == 0 {
		return nil, nil
	}
	oui, sn, err := r.lookupDeviceOUISN(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	mt := metrics.MetricTypeCounter
	q := metrics.QueryRequest{
		DeviceOUIs:  []string{oui},
		DeviceSNs:   []string{sn},
		MetricPaths: counterNames,
		MetricType:  &mt,
		Granularity: metrics.Granularity15Min,
		StartTime:   startTime,
		EndTime:     endTime,
		Limit:       100000,
	}
	ms, err := r.metricsRepo.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query counters for kpi: %w", err)
	}
	result := make(map[string]float64, len(counterNames))
	for _, m := range ms {
		if cellID != "" {
			if m.ObjectLDN == nil || *m.ObjectLDN != cellID {
				continue
			}
		}
		result[m.MetricPath] += m.MetricValue
	}
	if duration := endTime.Sub(startTime); duration > 0 {
		result["period_seconds"] = duration.Seconds()
	}
	return result, nil
}

// --- 字段转换辅助 ---

// counterToMetric 把 model.PMCounter（业务键 OUI+SN）转 metrics.PMMetric。
// 上游 collector 已填 c.OUI 和 c.DeviceSN，这里直接拿。
func counterToMetric(c model.PMCounter) metrics.PMMetric {
	extra := map[string]any{}
	if c.CounterGroup != "" {
		extra["counter_group"] = c.CounterGroup
	}
	if c.Granularity > 0 {
		extra["granularity_minutes"] = c.Granularity
	}
	if c.DeviceID != uuid.Nil {
		extra["device_id"] = c.DeviceID.String()
	}
	var ldn *string
	if c.CellID != "" {
		v := c.CellID
		ldn = &v
	}
	endTime := c.Time
	startTime := endTime
	if c.Granularity > 0 {
		startTime = endTime.Add(-time.Duration(c.Granularity) * time.Minute)
	}
	m := metrics.PMMetric{
		DeviceOUI:   c.OUI,
		DeviceSN:    c.DeviceSN,
		MetricPath:  c.CounterName,
		MetricType:  metrics.MetricTypeCounter,
		MetricValue: c.CounterValue,
		Granularity: metrics.Granularity15Min,
		Time:        endTime,
		StartTime:   startTime,
		EndTime:     endTime,
		ObjectLDN:   ldn,
		Extra:       extra,
	}
	// T-0164-G6 收尾 BUG-A：counter 的 statis_type 由 collector.filterByWhitelist
	// 从 indicator 元数据填到 PMCounter.StatisType，这里透传给 pm_metrics.statis_type
	// 列，驱动 G5 aggregator CASE WHEN m.statis_type 路由 SUM/AVG/MAX。
	if c.StatisType != "" {
		st := metrics.StatisType(c.StatisType)
		m.StatisType = &st
	}
	return m
}

// metricToCounter 反向：PMMetric → PMCounter。
// device_id 从 extra 取（写入时存的），失败时留 uuid.Nil。
func metricToCounter(m metrics.PMMetric) model.PMCounter {
	c := model.PMCounter{
		Time:         m.Time,
		OUI:          m.DeviceOUI,
		DeviceSN:     m.DeviceSN,
		CounterName:  m.MetricPath,
		CounterValue: m.MetricValue,
		Granularity:  15, // G3 阶段固定 15min 粒度
	}
	if m.StatisType != nil {
		c.StatisType = string(*m.StatisType)
	}
	if m.ObjectLDN != nil {
		c.CellID = *m.ObjectLDN
	}
	if m.Extra != nil {
		if did, ok := m.Extra["device_id"].(string); ok {
			if parsed, err := uuid.Parse(did); err == nil {
				c.DeviceID = parsed
			}
		}
		if grp, ok := m.Extra["counter_group"].(string); ok {
			c.CounterGroup = grp
		}
		if gm, ok := m.Extra["granularity_minutes"].(float64); ok && gm > 0 {
			c.Granularity = int(gm)
		}
	}
	return c
}

// counterFilterToMetricsQuery 把 CounterFilter（按 device_id uuid 过滤）转换为 metrics.QueryRequest
// （按 oui+sn 双键过滤）。需要反查 devices 表拿 oui+sn。
func (r *PgCounterRepository) counterFilterToMetricsQuery(ctx context.Context, filter CounterFilter) (metrics.QueryRequest, error) {
	mt := metrics.MetricTypeCounter
	q := metrics.QueryRequest{
		MetricType:    &mt,
		Granularity:   metrics.Granularity15Min,
		StartTime:     filter.StartTime,
		EndTime:       filter.EndTime,
		VisibleGroups: filter.VisibleGroups, // #64 设备组数据权限透传
	}
	if filter.DeviceID != nil {
		oui, sn, err := r.lookupDeviceOUISN(ctx, *filter.DeviceID)
		if err != nil {
			return q, err
		}
		q.DeviceOUIs = []string{oui}
		q.DeviceSNs = []string{sn}
	}
	if filter.CounterName != nil {
		q.MetricPaths = []string{*filter.CounterName}
	}
	return q, nil
}

// lookupDeviceOUISN 反查 devices 表的 (oui, serial_number) 双键。
// 一次/请求开销，pgxpool 自带连接复用。
func (r *PgCounterRepository) lookupDeviceOUISN(ctx context.Context, deviceID uuid.UUID) (string, string, error) {
	var oui, sn string
	err := r.pool.QueryRow(ctx,
		`SELECT oui, serial_number FROM devices WHERE id = $1`, deviceID,
	).Scan(&oui, &sn)
	if err != nil {
		return "", "", fmt.Errorf("lookup device oui+sn by id %s: %w", deviceID, err)
	}
	return oui, sn, nil
}

// aggregateInMemory 在内存里按 (oui+sn, cell, counter_name) 做 sum/avg/min/max/count 聚合，
// bucket 对齐到小时端点（与旧 pm_counters_hourly 1h 窗口一致）。
// T-0164-P3 fix: 设备维度按 (oui, sn) 双键聚合；DeviceID(uuid) 从 extra 反查（写入时 collector 存的）。
func aggregateInMemory(ms []metrics.PMMetric, groupFilter *string) []AggregatedCounter {
	type key struct {
		bucket time.Time
		oui    string
		sn     string
		cellID string
		group  string
		name   string
	}
	agg := make(map[key]*AggregatedCounter)
	for _, m := range ms {
		bucket := m.Time.Truncate(time.Hour)
		var did uuid.UUID
		var cellID, group string
		if m.ObjectLDN != nil {
			cellID = *m.ObjectLDN
		}
		if m.Extra != nil {
			if g, ok := m.Extra["counter_group"].(string); ok {
				group = g
			}
			if didStr, ok := m.Extra["device_id"].(string); ok {
				if parsed, err := uuid.Parse(didStr); err == nil {
					did = parsed
				}
			}
		}
		if groupFilter != nil && *groupFilter != group {
			continue
		}
		k := key{bucket: bucket, oui: m.DeviceOUI, sn: m.DeviceSN, cellID: cellID, group: group, name: m.MetricPath}
		a, ok := agg[k]
		if !ok {
			a = &AggregatedCounter{
				Bucket:       bucket,
				DeviceID:     did,
				CellID:       cellID,
				CounterGroup: group,
				CounterName:  m.MetricPath,
				MinValue:     m.MetricValue,
				MaxValue:     m.MetricValue,
			}
			agg[k] = a
		}
		a.SumValue += m.MetricValue
		a.SampleCount++
		if m.MetricValue < a.MinValue {
			a.MinValue = m.MetricValue
		}
		if m.MetricValue > a.MaxValue {
			a.MaxValue = m.MetricValue
		}
	}
	result := make([]AggregatedCounter, 0, len(agg))
	for _, a := range agg {
		if a.SampleCount > 0 {
			a.AvgValue = a.SumValue / float64(a.SampleCount)
		}
		result = append(result, *a)
	}
	return result
}

var _ CounterRepository = (*PgCounterRepository)(nil)
