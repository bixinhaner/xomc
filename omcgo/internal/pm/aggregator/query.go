package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// Dimension 标识查询维度（device / device_group）。
type Dimension string

const (
	DimensionDevice      Dimension = "device"
	DimensionDeviceGroup Dimension = "device_group"
)

// QueryRequest 是 Aggregator.Query 的输入。
//
// Granularity 必填，路由表名；Dimension 默认 device。
// DeviceOUIs/DeviceSNs 是 device 维度过滤；DeviceGroupIDs 是 group 维度过滤；
// 不能同时设。
type QueryRequest struct {
	Granularity    metrics.Granularity
	Dimension      Dimension
	DeviceOUIs     []string
	DeviceSNs      []string
	DeviceGroupIDs []uuid.UUID
	MetricPaths    []string
	MetricType     *metrics.MetricType
	StartTime      time.Time
	EndTime        time.Time
	Limit          int
	Offset         int
}

// Row 是 Aggregator.Query 的输出行。device 维度填 DeviceOUI/DeviceSN/ObjectLDN；
// device_group 维度填 DeviceGroupID。其余字段两维度共用。
// JSON tag 统一 snake_case 对齐其它 REST 响应；前端 mapper 走 snake → camel。
type Row struct {
	DeviceOUI     string              `json:"device_oui,omitempty"`
	DeviceSN      string              `json:"device_sn,omitempty"`
	DeviceGroupID uuid.UUID           `json:"device_group_id,omitempty"`
	MetricPath    string              `json:"metric_path"`
	MetricType    metrics.MetricType  `json:"metric_type"`
	MetricValue   float64             `json:"metric_value"`
	StatisType    *metrics.StatisType `json:"statis_type,omitempty"`
	Granularity   metrics.Granularity `json:"granularity"`
	Time          time.Time           `json:"time"`
	StartTime     time.Time           `json:"start_time"`
	EndTime       time.Time           `json:"end_time"`
	IngestTime    time.Time           `json:"ingest_time"`
	ObjectLDN     *string             `json:"object_ldn,omitempty"`
	Extra         map[string]any      `json:"extra,omitempty"`
}

// Query 根据 (Granularity, Dimension) 路由到对应聚合表查询。
//
// 路由表：
//
//   - 15min × device       → pm_metrics
//   - 15min × device_group → 不支持（无 15min 级 group 聚合）
//   - hourly × device      → pm_metrics_hourly
//   - hourly × device_group→ pm_group_metrics_hourly
//   - daily × ...          → pm_metrics_daily / pm_group_metrics_daily
//   - weekly × ...         → pm_metrics_weekly / pm_group_metrics_weekly
//   - monthly × ...        → pm_metrics_monthly / pm_group_metrics_monthly
func (a *Aggregator) Query(ctx context.Context, q QueryRequest) ([]Row, error) {
	if q.Dimension == "" {
		q.Dimension = DimensionDevice
	}
	table, err := SelectTable(q.Granularity, q.Dimension)
	if err != nil {
		return nil, err
	}

	if q.Dimension == DimensionDeviceGroup {
		return a.queryGroupTable(ctx, table, q)
	}
	return a.queryDeviceTable(ctx, table, q)
}

func (a *Aggregator) queryDeviceTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	qb := storage.Psql.Select(
		"device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
		"statis_type", "granularity", "time", "start_time", "end_time", "ingest_time", "object_ldn", "extra",
	).From(table)
	qb = applyDeviceFilters(qb, q)
	qb = qb.OrderBy("time DESC")
	if q.Limit > 0 {
		qb = qb.Limit(uint64(q.Limit))
	}
	if q.Offset > 0 {
		qb = qb.Offset(uint64(q.Offset))
	}
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query build %s: %w", table, err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query exec %s: %w", table, err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var r Row
		var statis, ldn *string
		var metricType, granularity string
		var extraBytes []byte
		if err := rows.Scan(
			&r.DeviceOUI, &r.DeviceSN, &r.MetricPath, &metricType, &r.MetricValue,
			&statis, &granularity, &r.Time, &r.StartTime, &r.EndTime, &r.IngestTime, &ldn, &extraBytes,
		); err != nil {
			return nil, fmt.Errorf("aggregator.Query scan %s: %w", table, err)
		}
		r.MetricType = metrics.MetricType(metricType)
		r.Granularity = metrics.Granularity(granularity)
		if statis != nil {
			st := metrics.StatisType(*statis)
			r.StatisType = &st
		}
		r.ObjectLDN = ldn
		if len(extraBytes) > 0 {
			extra := make(map[string]any)
			if err := json.Unmarshal(extraBytes, &extra); err == nil {
				r.Extra = extra
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (a *Aggregator) queryGroupTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	// group 表 schema：device_group_id + metric_path/type/value/statis_type/granularity/time/start/end/ingest + extra
	qb := storage.Psql.Select(
		"device_group_id", "metric_path", "metric_type", "metric_value",
		"statis_type", "granularity", "time", "start_time", "end_time", "ingest_time", "extra",
	).From(table)
	qb = applyGroupFilters(qb, q)
	qb = qb.OrderBy("time DESC")
	if q.Limit > 0 {
		qb = qb.Limit(uint64(q.Limit))
	}
	if q.Offset > 0 {
		qb = qb.Offset(uint64(q.Offset))
	}
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query build %s: %w", table, err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query exec %s: %w", table, err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var r Row
		var statis *string
		var metricType, granularity string
		var extraBytes []byte
		if err := rows.Scan(
			&r.DeviceGroupID, &r.MetricPath, &metricType, &r.MetricValue,
			&statis, &granularity, &r.Time, &r.StartTime, &r.EndTime, &r.IngestTime, &extraBytes,
		); err != nil {
			return nil, fmt.Errorf("aggregator.Query scan %s: %w", table, err)
		}
		r.MetricType = metrics.MetricType(metricType)
		r.Granularity = metrics.Granularity(granularity)
		if statis != nil {
			st := metrics.StatisType(*statis)
			r.StatisType = &st
		}
		if len(extraBytes) > 0 {
			extra := make(map[string]any)
			if err := json.Unmarshal(extraBytes, &extra); err == nil {
				r.Extra = extra
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ── 过滤条件 ───────────────────────────────────────────────────────────────

func applyDeviceFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if len(q.DeviceOUIs) > 0 && len(q.DeviceSNs) > 0 {
		n := len(q.DeviceOUIs)
		if len(q.DeviceSNs) < n {
			n = len(q.DeviceSNs)
		}
		or := sq.Or{}
		for i := 0; i < n; i++ {
			or = append(or, sq.And{
				sq.Eq{"device_oui": q.DeviceOUIs[i]},
				sq.Eq{"device_sn": q.DeviceSNs[i]},
			})
		}
		qb = qb.Where(or)
	} else if len(q.DeviceOUIs) > 0 {
		qb = qb.Where(sq.Eq{"device_oui": q.DeviceOUIs})
	} else if len(q.DeviceSNs) > 0 {
		qb = qb.Where(sq.Eq{"device_sn": q.DeviceSNs})
	}
	return applyCommonFilters(qb, q)
}

func applyGroupFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if len(q.DeviceGroupIDs) > 0 {
		qb = qb.Where(sq.Eq{"device_group_id": q.DeviceGroupIDs})
	}
	return applyCommonFilters(qb, q)
}

func applyCommonFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if len(q.MetricPaths) > 0 {
		qb = qb.Where(sq.Eq{"metric_path": q.MetricPaths})
	}
	if q.MetricType != nil {
		qb = qb.Where(sq.Eq{"metric_type": string(*q.MetricType)})
	}
	if q.Granularity != "" {
		qb = qb.Where(sq.Eq{"granularity": string(q.Granularity)})
	}
	if !q.StartTime.IsZero() {
		qb = qb.Where(sq.GtOrEq{"time": q.StartTime})
	}
	if !q.EndTime.IsZero() {
		qb = qb.Where(sq.LtOrEq{"time": q.EndTime})
	}
	return qb
}

// ── 表名路由（导出便于单测 + 其它包查表名） ────────────────────────────────

// ErrUnsupportedQuery 表示 (Granularity, Dimension) 组合不支持。
// 唯一不支持的组合：15min × device_group（无 15min 级 group 聚合源）。
var ErrUnsupportedQuery = fmt.Errorf("aggregator: unsupported (granularity, dimension) combination")

// SelectTable 把 (granularity, dim) 路由到具体表名。
func SelectTable(g metrics.Granularity, dim Dimension) (string, error) {
	if dim == "" {
		dim = DimensionDevice
	}
	switch g {
	case metrics.Granularity15Min:
		if dim == DimensionDeviceGroup {
			return "", fmt.Errorf("%w: 15min × device_group", ErrUnsupportedQuery)
		}
		return "pm_metrics", nil
	case metrics.GranularityHourly:
		if dim == DimensionDeviceGroup {
			return "pm_group_metrics_hourly", nil
		}
		return "pm_metrics_hourly", nil
	case metrics.GranularityDaily:
		if dim == DimensionDeviceGroup {
			return "pm_group_metrics_daily", nil
		}
		return "pm_metrics_daily", nil
	case metrics.GranularityWeekly:
		if dim == DimensionDeviceGroup {
			return "pm_group_metrics_weekly", nil
		}
		return "pm_metrics_weekly", nil
	case metrics.GranularityMonthly:
		if dim == DimensionDeviceGroup {
			return "pm_group_metrics_monthly", nil
		}
		return "pm_metrics_monthly", nil
	}
	return "", fmt.Errorf("aggregator: unknown granularity %q", g)
}
