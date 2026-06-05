package export

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
)

// deviceSelectCols 是 device / adhoc 行级表流式取数的固定列序（与 Scan 一一对应）。
var deviceSelectCols = []string{
	"id", "device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
	"statis_type", "granularity", "time", "start_time", "end_time", "object_ldn",
}

// buildDeviceKeysetSQL 构造 device 维度表的 (time, id) keyset 流式查询。
//
// 过滤语义与仪表盘聚合查询的 device 维度一致（成对 OUI/SN、metric_paths、metric_type、
// granularity、时窗、制式）；ORDER BY time, id 配 keyset 游标保证不漏不重。
// started=false 时取首批（无游标谓词）；之后用 (time, id) > (curTime, curID) 推进。
func buildDeviceKeysetSQL(table string, req aggregator.QueryRequest, objectLDNs []string, started bool, curTime time.Time, curID uuid.UUID, limit int) (string, []any) {
	b := storage.Psql.Select(deviceSelectCols...).From(table)
	b = applyDeviceExportFilters(b, req, objectLDNs)
	if started {
		// keyset：(time, id) 严格大于游标。time 列名带引号避免与保留字冲突。
		b = b.Where(sq.Expr(`("time", id) > (?, ?)`, curTime, curID))
	}
	b = b.OrderBy(`"time" ASC`, "id ASC").Limit(uint64(limit))
	q, args, _ := b.ToSql()
	return q, args
}

// buildAdhocKeysetSQL 构造 pm_adhoc_aggregation_results 的 (time, id) keyset 流式查询。
func buildAdhocKeysetSQL(taskID uuid.UUID, startTime, endTime time.Time, started bool, curTime time.Time, curID uuid.UUID, limit int) (string, []any) {
	b := storage.Psql.Select(deviceSelectCols...).
		From("pm_adhoc_aggregation_results").
		Where(sq.Eq{"task_id": taskID})
	if !startTime.IsZero() {
		b = b.Where(sq.GtOrEq{"time": startTime})
	}
	if !endTime.IsZero() {
		b = b.Where(sq.LtOrEq{"time": endTime})
	}
	if started {
		b = b.Where(sq.Expr(`("time", id) > (?, ?)`, curTime, curID))
	}
	b = b.OrderBy(`"time" ASC`, "id ASC").Limit(uint64(limit))
	q, args, _ := b.ToSql()
	return q, args
}

// applyDeviceExportFilters 复刻 aggregator 的 device 维度过滤（成对 OUI/SN + 公共过滤），
// 外加 A1 小区/PLMN 下钻白名单 objectLDNs（QueryRequest 无此字段，单独传入）。
// 与 aggregator.applyDeviceFilters 同语义，独立实现以避免改动其签名（scope 要求只读复用）。
func applyDeviceExportFilters(b sq.SelectBuilder, req aggregator.QueryRequest, objectLDNs []string) sq.SelectBuilder {
	if len(req.DeviceOUIs) > 0 && len(req.DeviceSNs) > 0 {
		n := len(req.DeviceOUIs)
		if len(req.DeviceSNs) < n {
			n = len(req.DeviceSNs)
		}
		or := sq.Or{}
		for i := 0; i < n; i++ {
			or = append(or, sq.And{
				sq.Eq{"device_oui": req.DeviceOUIs[i]},
				sq.Eq{"device_sn": req.DeviceSNs[i]},
			})
		}
		b = b.Where(or)
	} else if len(req.DeviceOUIs) > 0 {
		b = b.Where(sq.Eq{"device_oui": req.DeviceOUIs})
	} else if len(req.DeviceSNs) > 0 {
		b = b.Where(sq.Eq{"device_sn": req.DeviceSNs})
	}
	if len(objectLDNs) > 0 {
		// A1 小区/PLMN 下钻：只取白名单命中的 object_ldn 行（sq.Eq 切片 → IN(...)，等价 = ANY）。
		b = b.Where(sq.Eq{"object_ldn": objectLDNs})
	}
	if len(req.MetricPaths) > 0 {
		b = b.Where(sq.Eq{"metric_path": req.MetricPaths})
	}
	if req.MetricType != nil {
		b = b.Where(sq.Eq{"metric_type": string(*req.MetricType)})
	}
	if req.Granularity != "" {
		b = b.Where(sq.Eq{"granularity": string(req.Granularity)})
	}
	if !req.StartTime.IsZero() {
		b = b.Where(sq.GtOrEq{"time": req.StartTime})
	}
	if !req.EndTime.IsZero() {
		b = b.Where(sq.LtOrEq{"time": req.EndTime})
	}
	if len(req.Technologies) > 0 {
		b = b.Where(
			"(device_oui, device_sn) IN (SELECT oui, serial_number FROM devices WHERE technology = ANY(?))",
			req.Technologies,
		)
	}
	return b
}
