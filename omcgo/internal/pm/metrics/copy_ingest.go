package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/model"
)

// FileMarker 是 pm_files 幂等标记的最小字段集（copy-direct 入库路径用）。
type FileMarker struct {
	ID           uuid.UUID
	DeviceID     uuid.UUID
	DeviceSN     string
	Carrier      string
	Technology   string
	FileName     string
	FileSize     int64
	CollectTime  time.Time
	MinioPath    string
	CounterCount int
}

// MetricFromCounter 把 model.PMCounter 转 PMMetric。
//
// 从 counter 包上移到 metrics 包，使写路径（counter.BatchInsert 与 CopyIngest）共用同一份
// 转换，避免漂移。statis_type 由上游 collector.filterByWhitelist 从 indicator 元数据填到
// PMCounter.StatisType，这里透传给 pm_metrics.statis_type 驱动 G5 聚合 CASE WHEN（BUG-A）。
func MetricFromCounter(c model.PMCounter) PMMetric {
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
	m := PMMetric{
		DeviceOUI:   c.OUI,
		DeviceSN:    c.DeviceSN,
		MetricPath:  c.CounterName,
		MetricType:  MetricTypeCounter,
		MetricValue: c.CounterValue,
		Granularity: Granularity15Min,
		Time:        endTime,
		StartTime:   startTime,
		EndTime:     endTime,
		ObjectLDN:   ldn,
		Extra:       extra,
	}
	if c.StatisType != "" {
		st := StatisType(c.StatisType)
		m.StatisType = &st
	}
	return m
}

// MetricFromKPIValue 把 model.KPIValue 转 PMMetric（从 kpi 包上移，写两路共用）。
func MetricFromKPIValue(v model.KPIValue) PMMetric {
	extra := map[string]any{}
	if v.Carrier != "" {
		extra["carrier"] = string(v.Carrier)
	}
	if v.Technology != "" {
		extra["technology"] = string(v.Technology)
	}
	if v.DeviceID != uuid.Nil {
		extra["device_id"] = v.DeviceID.String()
	}
	var ldn *string
	if v.CellID != "" {
		s := v.CellID
		ldn = &s
	}
	return PMMetric{
		DeviceOUI:   v.OUI,
		DeviceSN:    v.DeviceSN,
		MetricPath:  v.IndicatorID,
		MetricType:  MetricTypeKPI,
		MetricValue: v.KPIValue,
		Granularity: Granularity15Min,
		Time:        v.Time,
		StartTime:   v.Time,
		EndTime:     v.Time,
		ObjectLDN:   ldn,
		Extra:       extra,
	}
}

// CopyIngest 原子写入一个 PM 文件的全部 metric 行 + pm_files 幂等标记（激进写路径）。
//
// 同一事务内：先插 pm_files（ON CONFLICT(device_sn,file_name) DO NOTHING）：
//   - 标记命中冲突（NATS 重投 / 并发已入库）→ 回滚跳过，返回 ingested=false（上层 ack 掉）；
//   - 否则 plain COPY 全部 metric 行进 pm_metrics（无 temp 表、无 ON CONFLICT、不依赖自然键
//     唯一索引），提交，返回 true。
//
// 这样把幂等从"每行自然键 UPSERT"（9.2M 次索引探测）下沉到"每文件一次 pm_files 唯一约束"
// （600 次），消除写 CPU 大头。三重安全：
//   - 重试安全：标记与 COPY 同事务，handler 失败 → 全回滚 → 标记未落 → 重投重做（不丢）。
//   - 并发安全：并发重投两事务都插标记，唯一约束让一个提交、另一个冲突回滚 → 不重复。
//   - 崩溃安全：提交后崩溃 → 标记与数据皆已落 → 重投因标记冲突跳过 → 不重复、不丢。
//
// 前提：已删 uq_pm_metrics_natural（migration），否则 plain COPY 撞重复行会整批失败。
func (r *PgRepository) CopyIngest(ctx context.Context, marker FileMarker, counters []model.PMCounter, kpis []model.KPIValue) (ingested bool, err error) {
	ms := make([]PMMetric, 0, len(counters)+len(kpis))
	for _, c := range counters {
		ms = append(ms, MetricFromCounter(c))
	}
	for _, k := range kpis {
		ms = append(ms, MetricFromKPIValue(k))
	}
	// 文件内按自然键去重（last-wins），复刻 UPSERT 路径的 ON CONFLICT DO UPDATE 语义：
	// plain COPY 不走 ON CONFLICT，若同一文件出现重复自然键（多个上报名经白名单改写命中同一
	// IndicatorID、或厂商把同 measType 重复上报），撞 uq_pm_metrics_natural 唯一索引会整批 COPY
	// 失败。在内存里先折叠成一行（取最后值）等价于 UPSERT 的"后写覆盖"，既避免 COPY 失败又保留
	// 该索引对跨文件误重写的兜底（保留 = 不静默 double-count）。
	ms = dedupeByNaturalKey(ms)
	rows, err := buildRows(ms)
	if err != nil {
		return false, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin copy-ingest tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if BulkAsyncCommit {
		if _, e := tx.Exec(ctx, "SET LOCAL synchronous_commit = off"); e != nil {
			return false, fmt.Errorf("set local synchronous_commit: %w", e)
		}
	}

	id := marker.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	ct, err := tx.Exec(ctx,
		`INSERT INTO pm_files (id, device_id, device_sn, carrier, technology, file_name, file_size,
		                       collect_time, minio_path, parsed, counter_count, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true,$10,NOW())
		 ON CONFLICT (device_sn, file_name) DO NOTHING`,
		id, marker.DeviceID, marker.DeviceSN, marker.Carrier, marker.Technology, marker.FileName,
		marker.FileSize, marker.CollectTime, marker.MinioPath, marker.CounterCount)
	if err != nil {
		return false, fmt.Errorf("insert pm_files marker: %w", err)
	}
	if ct.RowsAffected() == 0 {
		// 标记冲突：该文件已入库 → 跳过（回滚，不写 metrics）。
		return false, nil
	}

	if len(rows) > 0 {
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"pm_metrics"}, pmMetricsColumns, pgx.CopyFromRows(rows)); err != nil {
			return false, classifyInsertError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, classifyInsertError(err)
	}
	return true, nil
}

// dedupeByNaturalKey 把同自然键的多条 PMMetric 折叠成一条（保留切片中最后出现的那条），
// 自然键 = uq_pm_metrics_natural 的列集 (device_oui, device_sn, metric_path, granularity,
// end_time, time, object_ldn)。等价于 ON CONFLICT DO UPDATE 的"后写覆盖"，让 plain COPY 不会
// 因文件内重复自然键撞唯一索引而整批失败。保持首次出现的相对顺序（仅替换值），便于排查。
func dedupeByNaturalKey(ms []PMMetric) []PMMetric {
	idx := make(map[string]int, len(ms))
	out := make([]PMMetric, 0, len(ms))
	for _, m := range ms {
		ldn := ""
		if m.ObjectLDN != nil {
			ldn = *m.ObjectLDN
		}
		// time 缺省取 end_time（与 metricRowValues 落值规则一致），保证 key 与最终入库行对齐。
		t := m.Time
		if t.IsZero() {
			t = m.EndTime
		}
		key := strings.Join([]string{
			m.DeviceOUI, m.DeviceSN, m.MetricPath, string(m.Granularity),
			m.EndTime.Format(time.RFC3339Nano), t.Format(time.RFC3339Nano), ldn,
		}, "\x00")
		if i, ok := idx[key]; ok {
			out[i] = m // last-wins
			continue
		}
		idx[key] = len(out)
		out = append(out, m)
	}
	return out
}
