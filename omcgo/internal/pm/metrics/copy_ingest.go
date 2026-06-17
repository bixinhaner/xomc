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
//
// 运维须知（migration 000042 删自然键唯一索引后，pm_files 标记是入库幂等的唯一锚点）：
//   - pm_metrics 的行不可脱离其 pm_files 标记被单独删除——标记在而数据缺会让重投/重放（NATS 重投、
//     从 MinIO 重新 publish pm.file.received）因标记冲突被永久跳过，形成不可自愈的空洞。
//   - 需要强制重灌某文件时，先删其 pm_files 标记行（device_sn+file_name）再重新 publish 事件。
//   - 源文件仍留在 MinIO（入库后不删），故标记被清后总可重放重建。
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
		// #479 改动二：time 统一为桶起点（= start_time），不再写桶结束时刻。
		// time 列只作时序库分区/排序/去重的时间轴，语义恒等于 start_time。
		Time:      startTime,
		StartTime: startTime,
		EndTime:   endTime,
		ObjectLDN: ldn,
		Extra:     extra,
	}
	if c.StatisType != "" {
		st := StatisType(c.StatisType)
		m.StatisType = &st
	}
	return m
}

// granularity15MinDuration 是 Granularity15Min（"15min" 桶）对应的窗口时长。
// KPIValue 由 KPI 引擎在 15min 窗口收尾时构造（engine 全程 collectTime.Add(-15*time.Minute)
// 假设 15min 粒度），core/model.KPIValue 只携带窗口止点 Time、无独立 Start/End，故窗口起点在
// 本入库层按粒度推导补齐，与 MetricFromCounter 的 start = end - granularity 口径一致。
const granularity15MinDuration = 15 * time.Minute

// MetricFromKPIValue 把 model.KPIValue 转 PMMetric（从 kpi 包上移，写两路共用）。
//
// KPIValue 仅有窗口止点 Time（= 引擎 collectTime = granPeriod.endTime），无独立窗口起点。
// 历史上 start/end 都写成 v.Time，致 pm_metrics.start_time == end_time，前端悬浮框「开始/结束」
// 显示同一时刻（#199 / #208 打点起止相同子项）。这里把 StartTime 推导为 EndTime - 15min，给出
// 完整 15min 区间，与 MetricFromCounter 的 start = end - granularity 口径对齐。
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
	endTime := v.Time
	startTime := endTime.Add(-granularity15MinDuration)
	return PMMetric{
		DeviceOUI:   v.OUI,
		DeviceSN:    v.DeviceSN,
		MetricPath:  v.IndicatorID,
		MetricType:  MetricTypeKPI,
		MetricValue: v.KPIValue,
		Granularity: Granularity15Min,
		// #479 改动二：time 统一为桶起点（= start_time），不再写桶结束时刻。
		Time:      startTime,
		StartTime: startTime,
		EndTime:   endTime,
		ObjectLDN: ldn,
		Extra:     extra,
	}
}

// CopyIngest 原子写入一个 PM 文件的全部 metric 行 + pm_files 幂等标记（copy-direct 写路径）。
//
// 同一事务内：先插 pm_files（ON CONFLICT(device_sn,file_name) DO NOTHING）：
//   - 标记命中冲突（NATS 重投 / 并发已入库）→ 回滚跳过，返回 ingested=false（上层 ack 掉）；
//   - 否则 plain COPY 全部 metric 行进 pm_metrics（无 temp 表、无 ON CONFLICT），提交，返回 true。
//
// 把幂等从"每行自然键 UPSERT"下沉到"每文件一次 pm_files 唯一约束"。三重安全：
//   - 重试安全：标记与 COPY 同事务，handler 失败 → 全回滚 → 标记未落 → 重投重做（不丢）。
//   - 并发安全：并发重投两事务都插标记，唯一约束让一个提交、另一个冲突回滚 → 不重复。
//   - 崩溃安全：标记与数据同事务，提交即同时持久；提交后崩溃 → 二者皆已落 → 重投因标记冲突跳过
//     → 不重复不丢；提交前崩溃 → 二者皆未落 → 重投重做。
//
// 关键：本事务必须 durable 提交（synchronous_commit=on），不可走 BulkAsyncCommit 异步提交。
// 因为标记是幂等的唯一锚点、且 handler 返回 nil 后 NATS 立即 ack（at-least-once 消费）：若异步提交
// 在 WAL 落盘前返回成功并 ack，宿主崩溃会丢掉这次提交（标记+数据全无），而消息已 ack 不再重投，
// 文件将静默永久缺失（无自愈，对比旧路径靠 per-row UPSERT 重投自愈）。durable 提交保证 ack 时数据
// 必已落盘，崩溃只会落在"提交前"从而重投重做。（删 uq_pm_metrics_natural 后 plain COPY 不再受
// 唯一索引约束，文件内重复自然键已由 dedupeByNaturalKey 折叠。）
func (r *PgRepository) CopyIngest(ctx context.Context, marker FileMarker, counters []model.PMCounter, kpis []model.KPIValue) (ingested bool, err error) {
	ms := make([]PMMetric, 0, len(counters)+len(kpis))
	for _, c := range counters {
		ms = append(ms, MetricFromCounter(c))
	}
	for _, k := range kpis {
		ms = append(ms, MetricFromKPIValue(k))
	}
	// 文件内按自然键去重（last-wins），复刻旧 UPSERT 的 ON CONFLICT DO UPDATE"后写覆盖"语义：
	// 删 uq_pm_metrics_natural 后 plain COPY 不再因重复键失败，但若同一文件出现重复自然键（多个
	// 上报名经白名单改写命中同一 IndicatorID、或厂商把同 measType 重复上报），不折叠会写成两行 →
	// 读侧 SUM 重复计数。先在内存折叠成一行（取最后值）避免文件内 double-count。
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

	// 刻意不设 synchronous_commit=off：本事务承载 pm_files 幂等锚点，必须 durable 提交后才能让上层
	// ack 该 NATS 消息，否则崩溃丢提交 + 已 ack 不重投 = 文件静默永久缺失（见函数 doc）。

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
// 自然键 = 旧 uq_pm_metrics_natural 的列集 (device_oui, device_sn, metric_path, granularity,
// end_time, time, object_ldn)。等价于 ON CONFLICT DO UPDATE 的"后写覆盖"：删该唯一索引后 plain
// COPY 不会因重复键失败，但文件内重复自然键若不折叠会写成多行被读侧 SUM 重复计数，故先折叠。
// 保持首次出现的相对顺序（仅替换值），便于排查。
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
