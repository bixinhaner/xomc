package export

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

const (
	maxExportDeviceSNs   = 50
	maxExportMetricPaths = 50
)

// DashboardParams 是 source_type=dashboard 时 pm_kpi_export_tasks.params(jsonb) 的字段集。
//
// 字段对齐仪表盘聚合查询入参（同 /pm/metrics/aggregated），导出时去掉 limit 全量取数。
// 字段名沿用 REST query 的 snake_case，便于前端 T4 建任务时直接透传当前筛选。
type DashboardParams struct {
	Granularity    string   `json:"granularity"`
	Dimension      string   `json:"dimension"`
	DeviceOUIs     []string `json:"device_ouis"`
	DeviceSNs      []string `json:"device_sns"`
	DeviceGroupIDs []string `json:"device_group_ids"`
	ProductIDs     []string `json:"product_ids"`
	MetricPaths    []string `json:"metric_paths"`
	MetricType     string   `json:"metric_type"`
	Technologies   []string `json:"technologies"`
	StartTime      string   `json:"start_time"`
	EndTime        string   `json:"end_time"`
	Weekdays       []int    `json:"weekdays"`
	Hours          []int    `json:"hours"`
	// ObjectLDNs 是小区/PLMN 下钻白名单（A1）。空 = 不过滤，导该设备全部小区/PLMN；
	// 非空 = 只导命中行，与仪表盘下钻定格口径一致。device 维度专属（聚合维度无意义）。
	ObjectLDNs []string `json:"object_ldns"`
}

// AdhocParams 是 adhoc 结果类导出 params(jsonb) 的字段集。
type AdhocParams struct {
	TaskID     string   `json:"task_id"`
	StartTime  string   `json:"start_time"` // 可选二次时窗筛选
	EndTime    string   `json:"end_time"`
	ProductIDs []string `json:"product_ids"`
	ObjectLDNs []string `json:"object_ldns"`
	Weekdays   []int    `json:"weekdays"`
	Hours      []int    `json:"hours"`
}

type adhocExportFilter struct {
	TaskID           uuid.UUID
	StartTime        time.Time
	EndTime          time.Time
	ProductIDs       []uuid.UUID
	ObjectLDNs       []string
	Weekdays         []int
	Hours            []int
	CalendarTimezone string
}

func validateDashboardExportLimits(raw []byte) error {
	var p DashboardParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("parse dashboard export params: %w", err)
		}
	}
	if len(p.DeviceSNs) > maxExportDeviceSNs {
		return fmt.Errorf("device_sns exceeds maximum of %d", maxExportDeviceSNs)
	}
	if len(p.MetricPaths) > maxExportMetricPaths {
		return fmt.Errorf("metric_paths exceeds maximum of %d", maxExportMetricPaths)
	}
	return nil
}

// parseDashboardParams 把 params(jsonb) 解析成 aggregator.QueryRequest（去 limit/offset）
// 与小区/PLMN 下钻白名单 objectLDNs（A1，QueryRequest 无此字段，单独返回供 export 自身过滤）。
//
// 解析失败 / granularity 缺失返错（让任务走 failed 而非产出空文件）。
func parseDashboardParams(raw []byte) (aggregator.QueryRequest, []string, error) {
	var p DashboardParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return aggregator.QueryRequest{}, nil, fmt.Errorf("parse dashboard export params: %w", err)
		}
	}
	if p.Granularity == "" {
		return aggregator.QueryRequest{}, nil, fmt.Errorf("dashboard export params: granularity is required")
	}

	req := aggregator.QueryRequest{
		Granularity:  metrics.Granularity(p.Granularity),
		Dimension:    aggregator.Dimension(p.Dimension),
		DeviceOUIs:   p.DeviceOUIs,
		DeviceSNs:    p.DeviceSNs,
		MetricPaths:  p.MetricPaths,
		Technologies: p.Technologies,
		ObjectLDNs:   p.ObjectLDNs,
		Weekdays:     p.Weekdays,
		Hours:        p.Hours,
	}
	if p.MetricType != "" {
		mt := metrics.MetricType(p.MetricType)
		req.MetricType = &mt
	}
	for _, s := range p.DeviceGroupIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return aggregator.QueryRequest{}, nil, fmt.Errorf("dashboard export params: invalid device_group_id %q: %w", s, err)
		}
		req.DeviceGroupIDs = append(req.DeviceGroupIDs, id)
	}
	for _, s := range p.ProductIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return aggregator.QueryRequest{}, nil, fmt.Errorf("dashboard export params: invalid product_id %q: %w", s, err)
		}
		req.ProductIDs = append(req.ProductIDs, id)
	}
	if p.StartTime != "" {
		t, err := time.Parse(time.RFC3339, p.StartTime)
		if err != nil {
			return aggregator.QueryRequest{}, nil, fmt.Errorf("dashboard export params: invalid start_time %q: %w", p.StartTime, err)
		}
		req.StartTime = t
	}
	if p.EndTime != "" {
		t, err := time.Parse(time.RFC3339, p.EndTime)
		if err != nil {
			return aggregator.QueryRequest{}, nil, fmt.Errorf("dashboard export params: invalid end_time %q: %w", p.EndTime, err)
		}
		req.EndTime = t
	}
	return req, p.ObjectLDNs, nil
}

// parseAdhocParams 把 params(jsonb) 解析成 adhoc 取数条件。
func parseAdhocParams(raw []byte) (adhocExportFilter, error) {
	var p AdhocParams
	if len(raw) > 0 {
		if uerr := json.Unmarshal(raw, &p); uerr != nil {
			return adhocExportFilter{}, fmt.Errorf("parse adhoc export params: %w", uerr)
		}
	}
	if p.TaskID == "" {
		return adhocExportFilter{}, fmt.Errorf("adhoc export params: task_id is required")
	}
	taskID, err := uuid.Parse(p.TaskID)
	if err != nil {
		return adhocExportFilter{}, fmt.Errorf("adhoc export params: invalid task_id %q: %w", p.TaskID, err)
	}
	filter := adhocExportFilter{
		TaskID:     taskID,
		ObjectLDNs: p.ObjectLDNs,
		Weekdays:   p.Weekdays,
		Hours:      p.Hours,
	}
	if p.StartTime != "" {
		filter.StartTime, err = time.Parse(time.RFC3339, p.StartTime)
		if err != nil {
			return adhocExportFilter{}, fmt.Errorf("adhoc export params: invalid start_time %q: %w", p.StartTime, err)
		}
	}
	if p.EndTime != "" {
		filter.EndTime, err = time.Parse(time.RFC3339, p.EndTime)
		if err != nil {
			return adhocExportFilter{}, fmt.Errorf("adhoc export params: invalid end_time %q: %w", p.EndTime, err)
		}
	}
	for _, s := range p.ProductIDs {
		id, perr := uuid.Parse(s)
		if perr != nil {
			return adhocExportFilter{}, fmt.Errorf("adhoc export params: invalid product_id %q: %w", s, perr)
		}
		filter.ProductIDs = append(filter.ProductIDs, id)
	}
	return filter, nil
}
