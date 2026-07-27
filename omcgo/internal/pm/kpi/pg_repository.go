package kpi

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

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
// 设备唯一标识（T-0164-P3 fix）：按 TR-069 标准用 (OUI, DeviceSN) 双键。
//   - BatchInsert：上游 KPIEngine 已填 v.OUI / v.DeviceSN
//   - Query：handler 仍按 device_id(uuid) 接收，wrapper 反查 devices 表拿 (oui, sn)
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

// ReplaceForRecompute 原子替换 (oui, sn, cellID, 15min 窗口 end_time) 的 KPI 行：单事务内
// advisory 锁串行化 → DELETE 旧行 → INSERT 新行 → 提交。替代旧自然键 ON CONFLICT DO UPDATE 的
// 原子"后写覆盖"（migration 000042 删唯一索引后）。
//   - 原子：DELETE 与 INSERT 同事务，INSERT 失败整体回滚，旧 KPI 行不丢（修复非原子崩溃窗口）。
//   - 并发安全：pg_advisory_xact_lock 按 (oui,sn,cellID,end_time) 串行化同范围并发重算，杜绝两次
//     重算交错（各自 DELETE 后各自 INSERT）产生重复行。锁随事务结束自动释放。
//
// scope DELETE 限定单设备单窗口，走 idx_pm_metrics_device_time 定位，删唯一索引后依然高效。
// object_ldn 列 NOT NULL DEFAULT ”，cellID="" 对应 object_ldn=”（与写入侧 nil→” 一致）。
func (r *PgKPIRepository) ReplaceForRecompute(ctx context.Context, oui, deviceSN, cellID string, endTime time.Time, values []model.KPIValue) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin kpi recompute tx (sn=%s cell=%s): %w", deviceSN, cellID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 同范围并发重算串行化：advisory xact 锁键 = (oui|sn|cell|窗口) 的 64 位哈希。
	// 必须在 Go 侧哈希后传 int64 给 pg_advisory_xact_lock —— 不能把含 \x00 分隔符的锁键当
	// text 参数交给 PG 的 hashtextextended，否则 PG 拒收 NUL 字节：
	// invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)（device 级 cell="" 时必现）。
	h := fnv.New64a()
	_, _ = h.Write([]byte(oui + "\x00" + deviceSN + "\x00" + cellID + "\x00" + endTime.UTC().Format(time.RFC3339Nano)))
	lockKey := int64(h.Sum64())
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lockKey); err != nil {
		return fmt.Errorf("acquire recompute lock (sn=%s cell=%s): %w", deviceSN, cellID, err)
	}

	if err := metrics.DeleteKPIAnchorsTx(ctx, tx, oui, deviceSN, cellID, endTime); err != nil {
		return fmt.Errorf("delete kpi for recompute (sn=%s cell=%s): %w", deviceSN, cellID, err)
	}

	if len(values) > 0 {
		ms := make([]metrics.PMMetric, 0, len(values))
		for _, v := range values {
			ms = append(ms, metrics.MetricFromKPIValue(v))
		}
		if err := metrics.InsertRowsTx(ctx, tx, ms); err != nil {
			return fmt.Errorf("insert recomputed kpi (sn=%s cell=%s): %w", deviceSN, cellID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit kpi recompute (sn=%s cell=%s): %w", deviceSN, cellID, err)
	}
	return nil
}

func (r *PgKPIRepository) Query(ctx context.Context, filter KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	q, err := r.kpiFilterToMetricsQuery(ctx, filter)
	if err != nil {
		return nil, err
	}
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

// kpiValueToMetric: KPIValue → PMMetric。
//
// 转换实现已上移到 metrics 包（metrics.MetricFromKPIValue），让 kpi.BatchInsert（UPSERT 路径）
// 与 metrics.CopyIngest（copy-direct 路径）共用同一份字段映射，杜绝两路写出的行漂移。
func kpiValueToMetric(v model.KPIValue) metrics.PMMetric {
	return metrics.MetricFromKPIValue(v)
}

func metricToKPIValue(m metrics.PMMetric) model.KPIValue {
	// metric_path 现在存 K 编号；显示名由查询层按编号回填，本层无法还原。
	v := model.KPIValue{
		Time:        m.Time,
		OUI:         m.DeviceOUI,
		DeviceSN:    m.DeviceSN,
		IndicatorID: m.MetricPath,
		KPIName:     m.MetricPath,
		KPIValue:    m.MetricValue,
	}
	if m.ObjectLDN != nil {
		v.CellID = *m.ObjectLDN
	}
	if m.Extra != nil {
		if didStr, ok := m.Extra["device_id"].(string); ok {
			if parsed, err := uuid.Parse(didStr); err == nil {
				v.DeviceID = parsed
			}
		}
		if c, ok := m.Extra["carrier"].(string); ok {
			v.Carrier = model.CarrierCode(c)
		}
		if t, ok := m.Extra["technology"].(string); ok {
			v.Technology = model.Technology(t)
		}
	}
	return v
}

// kpiFilterToMetricsQuery: 反查 devices 表拿 oui+sn。
func (r *PgKPIRepository) kpiFilterToMetricsQuery(ctx context.Context, filter KPIFilter) (metrics.QueryRequest, error) {
	mt := metrics.MetricTypeKPI
	q := metrics.QueryRequest{
		MetricType: &mt,
		// Don't hardcode granularity - let the caller specify or query all granularities
		// This fixes the issue where Dashboard KPI data (hourly granularity) was not being returned
		// because the query was filtering for 15min granularity only
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
	if filter.KPIName != nil {
		q.MetricPaths = []string{*filter.KPIName}
	}
	return q, nil
}

// lookupDeviceOUISN 反查设备的 (oui, serial_number) 双键。
// r.pool 是 TsPool；devices 反查改读本库影子表 device_dim（跨库分离）。
func (r *PgKPIRepository) lookupDeviceOUISN(ctx context.Context, deviceID uuid.UUID) (string, string, error) {
	var oui, sn string
	err := r.pool.QueryRow(ctx,
		`SELECT oui, serial_number FROM device_dim WHERE id = $1`, deviceID,
	).Scan(&oui, &sn)
	if err != nil {
		return "", "", fmt.Errorf("lookup device oui+sn by id %s: %w", deviceID, err)
	}
	return oui, sn, nil
}

var _ KPIRepository = (*PgKPIRepository)(nil)
