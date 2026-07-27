package export

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// batchSize 是流式取数每批行数（设计 §5.4：每批约 5000，绝不一次性把全量读进内存）。
const batchSize = 5000

// RowSource 是流式取数的统一契约：每次调 Next 返回下一批 ExportRow，
// done=true 时表示数据已取完。实现内部用 keyset / 批次游标推进，不持有全量行。
type RowSource interface {
	Next(ctx context.Context) (rows []ExportRow, done bool, err error)
}

// PgQuerier 是取数所需的最小 pgxpool 子集（device / adhoc 直查表用），便于单测 stub。
type PgQuerier = aggregator.PgQuerier

// ── dashboard device 维度：keyset 流式直查 ───────────────────────────────────
//
// 对 device 维度（pm_metrics / pm_metrics_hourly 等含 id 列的行级表），按 (time, id) keyset
// 游标推进分批查库——这是设计 §5.4 指定的"按 time,id 推进、不漏不重"的关键路径，
// 死判 T2-rowcount-match 走此路。聚合维度（group/product/band/network）无行级 id，走
// dashboardAggregateSource 的批次游标兜底。

// dashboardDeviceSource 对 device 维度做 (time, id) keyset 流式取数。
type dashboardDeviceSource struct {
	db         PgQuerier
	table      string
	req        aggregator.QueryRequest
	objectLDNs []string // A1 小区/PLMN 下钻白名单（空=不过滤）

	curTime time.Time
	curID   uuid.UUID
	started bool
	done    bool
}

func newDashboardDeviceSource(db PgQuerier, table string, req aggregator.QueryRequest, objectLDNs []string) *dashboardDeviceSource {
	return &dashboardDeviceSource{db: db, table: table, req: req, objectLDNs: objectLDNs}
}

func (s *dashboardDeviceSource) Next(ctx context.Context) ([]ExportRow, bool, error) {
	if s.done {
		return nil, true, nil
	}
	sqlStr, args := buildDeviceKeysetSQL(s.table, s.req, s.objectLDNs, s.started, s.curTime, s.curID, batchSize)
	rows, err := s.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, false, fmt.Errorf("export dashboard device query %s: %w", s.table, err)
	}
	defer rows.Close()

	out := make([]ExportRow, 0, batchSize)
	var lastTime time.Time
	var lastID uuid.UUID
	n := 0
	for rows.Next() {
		var id uuid.UUID
		var oui, sn, metricPath, metricType, gran string
		var statis, ldn *string
		var value jsonx.Float
		var tm, st, et time.Time
		if err := rows.Scan(&id, &oui, &sn, &metricPath, &metricType, &value, &statis, &gran, &tm, &st, &et, &ldn); err != nil {
			return nil, false, fmt.Errorf("export dashboard device scan %s: %w", s.table, err)
		}
		out = append(out, ExportRow{
			Device:      deviceSNLabel(oui, sn),
			CellPLMN:    derefStr(ldn),
			MetricCode:  metricPath,
			MetricType:  metricType,
			Granularity: gran,
			Time:        tm,
			StartTime:   st,
			EndTime:     et,
			Value:       float64(value),
			StatisType:  derefStr(statis),
		})
		lastTime, lastID = tm, id
		n++
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("export dashboard device rows %s: %w", s.table, err)
	}

	s.started = true
	if n < batchSize {
		s.done = true
	} else {
		s.curTime, s.curID = lastTime, lastID
	}
	return out, false, nil
}

type dashboardDeviceOffsetSource struct {
	db         PgQuerier
	table      string
	req        aggregator.QueryRequest
	objectLDNs []string
	offset     int
	done       bool
}

func newDashboardDeviceOffsetSource(db PgQuerier, table string, req aggregator.QueryRequest, objectLDNs []string) *dashboardDeviceOffsetSource {
	return &dashboardDeviceOffsetSource{db: db, table: table, req: req, objectLDNs: objectLDNs}
}

func (s *dashboardDeviceOffsetSource) Next(ctx context.Context) ([]ExportRow, bool, error) {
	if s.done {
		return nil, true, nil
	}
	sqlStr, args := buildDeviceOffsetSQL(s.table, s.req, s.objectLDNs, s.offset, batchSize)
	rows, err := s.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, false, fmt.Errorf("export dashboard device query %s: %w", s.table, err)
	}
	defer rows.Close()

	out := make([]ExportRow, 0, batchSize)
	n := 0
	for rows.Next() {
		var oui, sn, metricPath, metricType, gran string
		var statis, ldn *string
		var value jsonx.Float
		var tm, st, et time.Time
		if err := rows.Scan(&oui, &sn, &metricPath, &metricType, &value, &statis, &gran, &tm, &st, &et, &ldn); err != nil {
			return nil, false, fmt.Errorf("export dashboard device scan %s: %w", s.table, err)
		}
		out = append(out, ExportRow{
			Device:      deviceSNLabel(oui, sn),
			CellPLMN:    derefStr(ldn),
			MetricCode:  metricPath,
			MetricType:  metricType,
			Granularity: gran,
			Time:        tm,
			StartTime:   st,
			EndTime:     et,
			Value:       float64(value),
			StatisType:  derefStr(statis),
		})
		n++
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("export dashboard device rows %s: %w", s.table, err)
	}

	if n < batchSize {
		s.done = true
	} else {
		s.offset += batchSize
	}
	return out, false, nil
}

// ── dashboard 聚合维度：批次游标兜底（含 KPI 反算） ─────────────────────────
//
// group/product/band/network 维度在 aggregator.Query 内现场 GROUP BY + KPI 反算，无行级 id，
// 无法做 (time,id) keyset。改用 limit/offset 批次游标分批拉 aggregator.Query（每批 batchSize），
// 同样不把全量读进内存。聚合行通常远少于 device 行（分组后），批次游标开销可接受。

// dashboardAggregateSource 对聚合维度按 offset 批次游标流式取数。
//
// objectLDNs 是 A1 小区/PLMN 下钻白名单：device 维度的 daily/weekly/monthly 表无行级 id 走此路，
// aggregator.Query 不支持 object_ldn 过滤，故取数后按白名单内存裁剪（非空时）。批次游标按原始
// 行数推进，裁剪只影响输出行、不破坏分页。聚合维度（group/product/network）行 ObjectLDN 为空，
// 前端也不会对这些维度发白名单，故无影响。
type dashboardAggregateSource struct {
	aggr       *aggregator.Aggregator
	req        aggregator.QueryRequest
	objectLDNs []string
	offset     int
	done       bool
}

func newDashboardAggregateSource(aggr *aggregator.Aggregator, req aggregator.QueryRequest, objectLDNs []string) *dashboardAggregateSource {
	return &dashboardAggregateSource{aggr: aggr, req: req, objectLDNs: objectLDNs}
}

func (s *dashboardAggregateSource) Next(ctx context.Context) ([]ExportRow, bool, error) {
	if s.done {
		return nil, true, nil
	}
	req := s.req
	req.Limit = batchSize
	req.Offset = s.offset
	aggRows, err := s.aggr.Query(ctx, req)
	if err != nil {
		return nil, false, fmt.Errorf("export dashboard aggregate query: %w", err)
	}
	allow := ldnAllowSet(s.objectLDNs)
	out := make([]ExportRow, 0, len(aggRows))
	for i := range aggRows {
		if allow != nil && !allow[derefStr(aggRows[i].ObjectLDN)] {
			continue // A1：白名单非空时只留命中小区/PLMN 行
		}
		out = append(out, aggregatorRowToExport(aggRows[i]))
	}
	if len(aggRows) < batchSize {
		s.done = true
	} else {
		s.offset += batchSize
	}
	return out, false, nil
}

// aggregatorRowToExport 把 aggregator.Row 映射成 ExportRow。
// device 维度的设备列只用 SN（与页面一致）；聚合维度 SN 为空，退化设备组 / 产品标识。
func aggregatorRowToExport(r aggregator.Row) ExportRow {
	device := deviceSNLabel(r.DeviceOUI, r.DeviceSN)
	if device == "" && r.DeviceGroupID != uuid.Nil {
		device = "DeviceGroup=" + r.DeviceGroupID.String()
	}
	if device == "" && r.ProductID != uuid.Nil {
		device = "Product=" + r.ProductID.String()
	}
	value := float64(r.MetricValue)
	if r.Filled {
		value = math.NaN()
	}
	return ExportRow{
		Device:      device,
		CellPLMN:    derefStr(r.ObjectLDN),
		MetricCode:  r.MetricPath,
		MetricName:  r.DisplayName,
		MetricType:  string(r.MetricType),
		Granularity: string(r.Granularity),
		Time:        r.Time,
		StartTime:   r.StartTime,
		EndTime:     r.EndTime,
		Value:       value,
		StatisType:  statisStr(r.StatisType),
	}
}

type fillEmptySource struct {
	src    RowSource
	req    aggregator.QueryRequest
	rows   []ExportRow
	i      int
	loaded bool
}

func newFillEmptySource(src RowSource, req aggregator.QueryRequest) *fillEmptySource {
	return &fillEmptySource{src: src, req: req}
}

func (s *fillEmptySource) Next(ctx context.Context) ([]ExportRow, bool, error) {
	if !s.loaded {
		if err := s.load(ctx); err != nil {
			return nil, false, err
		}
	}
	if s.i >= len(s.rows) {
		return nil, true, nil
	}
	end := s.i + batchSize
	if end > len(s.rows) {
		end = len(s.rows)
	}
	out := s.rows[s.i:end]
	s.i = end
	return out, false, nil
}

func (s *fillEmptySource) load(ctx context.Context) error {
	s.loaded = true
	var rows []ExportRow
	for {
		batch, done, err := s.src.Next(ctx)
		if err != nil {
			return err
		}
		rows = append(rows, batch...)
		if done {
			break
		}
	}
	aggRows := make([]aggregator.Row, 0, len(rows))
	for _, r := range rows {
		aggRows = append(aggRows, exportRowToAggregator(r))
	}
	filled := aggregator.FillEmptyBuckets(aggRows, s.req)
	out := make([]ExportRow, 0, len(filled))
	for _, r := range filled {
		out = append(out, aggregatorRowToExport(r))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Time.Equal(out[j].Time) {
			return out[i].Time.Before(out[j].Time)
		}
		if out[i].Device != out[j].Device {
			return out[i].Device < out[j].Device
		}
		if out[i].CellPLMN != out[j].CellPLMN {
			return out[i].CellPLMN < out[j].CellPLMN
		}
		return out[i].MetricCode < out[j].MetricCode
	})
	s.rows = out
	return nil
}

func exportRowToAggregator(r ExportRow) aggregator.Row {
	var objectLDN *string
	if r.CellPLMN != "" {
		ldn := r.CellPLMN
		objectLDN = &ldn
	}
	mt := metrics.MetricType(r.MetricType)
	if mt == "" {
		if strings.HasPrefix(r.MetricCode, "K") {
			mt = metrics.MetricTypeKPI
		} else {
			mt = metrics.MetricTypeCounter
		}
	}
	return aggregator.Row{
		DeviceSN:    r.Device,
		MetricPath:  r.MetricCode,
		DisplayName: r.MetricName,
		MetricType:  mt,
		MetricValue: jsonx.Float(r.Value),
		Granularity: metrics.Granularity(r.Granularity),
		Time:        r.Time,
		StartTime:   r.StartTime,
		EndTime:     r.EndTime,
		ObjectLDN:   objectLDN,
	}
}

func normalizeStoredResultExportRequest(req aggregator.QueryRequest) aggregator.QueryRequest {
	if len(req.MetricPaths) == 0 || req.MetricType == nil {
		return req
	}
	var inferred *metrics.MetricType
	for _, raw := range req.MetricPaths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		mt := metricTypeFromPath(path)
		if inferred == nil {
			v := mt
			inferred = &v
			continue
		}
		if *inferred != mt {
			req.MetricType = nil
			return req
		}
	}
	if inferred != nil && *inferred != *req.MetricType {
		req.MetricType = nil
	}
	return req
}

// statisStr 把 *metrics.StatisType 解引用成字符串（nil → 空串）。
func statisStr(p *metrics.StatisType) string {
	if p == nil {
		return ""
	}
	return string(*p)
}

// ── adhoc：keyset 流式直查 pm_adhoc_aggregation_results ───────────────────────

// adhocSource 按 task_id 过滤、(time, id) keyset 流式取 adhoc 结果表。
// dimension 决定首列对象名的解析口径（设备组名 / 产品名 / 频段 / 全网 / 聚合组）与是否填小区列；
// deviceCount 为任务圈选设备数，仅 aggregate_group 维度用于"聚合组(N个设备)"标签。
type adhocSource struct {
	db          PgQuerier
	taskID      uuid.UUID
	metricPaths []string
	startTime   time.Time
	endTime     time.Time
	dimension   string
	deviceCount int
	locale      appcontext.Locale

	curTime time.Time
	curID   uuid.UUID
	started bool
	done    bool
}

func newAdhocSource(db PgQuerier, taskID uuid.UUID, metricPaths []string, startTime, endTime time.Time, dimension string, deviceCount int, locs ...appcontext.Locale) *adhocSource {
	loc := appcontext.LocaleZH
	if len(locs) > 0 {
		loc = locs[0]
	}
	return &adhocSource{db: db, taskID: taskID, metricPaths: metricPaths, startTime: startTime, endTime: endTime, dimension: dimension, deviceCount: deviceCount, locale: loc}
}

func (s *adhocSource) Next(ctx context.Context) ([]ExportRow, bool, error) {
	if s.done {
		return nil, true, nil
	}
	sqlStr, args := buildAdhocKeysetSQL(s.taskID, s.metricPaths, s.startTime, s.endTime, s.started, s.curTime, s.curID, batchSize)
	rows, err := s.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, false, fmt.Errorf("export adhoc query: %w", err)
	}
	defer rows.Close()

	out := make([]ExportRow, 0, batchSize)
	var lastTime time.Time
	var lastID uuid.UUID
	n := 0
	for rows.Next() {
		var id uuid.UUID
		var oui, sn, metricPath, metricType, gran string
		var statis, ldn, productID, productName, groupName *string
		var value jsonx.Float
		var tm, st, et time.Time
		// 列序必须与 adhocSelectCols 完全一致：
		// id, oui, sn, metric_path, metric_type, metric_value, statis_type, granularity,
		// time, start_time, end_time, object_ldn, product_id, product_name, device_group_name。
		if err := rows.Scan(&id, &oui, &sn, &metricPath, &metricType, &value, &statis, &gran,
			&tm, &st, &et, &ldn, &productID, &productName, &groupName); err != nil {
			return nil, false, fmt.Errorf("export adhoc scan: %w", err)
		}
		device := adhocObjectLabel(s.dimension, oui, sn, derefStr(productID), derefStr(productName),
			derefStr(ldn), derefStr(groupName), s.deviceCount, s.locale)
		cell := ""
		if s.dimension == "device" {
			cell = derefStr(ldn)
		}
		tech := ""
		if s.dimension == "device_group" {
			// device_group 维度 object_ldn 形如 'DeviceGroup=<uuid>,Tech=<制式>'，解析出制式供「制式」列。
			tech = adhocTechnology(derefStr(ldn))
		}
		out = append(out, ExportRow{
			Device:      device,
			Technology:  tech,
			CellPLMN:    cell,
			MetricCode:  metricPath,
			MetricType:  metricType,
			Granularity: gran,
			Time:        tm,
			StartTime:   st,
			EndTime:     et,
			Value:       float64(value),
			StatisType:  derefStr(statis),
		})
		lastTime, lastID = tm, id
		n++
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("export adhoc rows: %w", err)
	}

	s.started = true
	if n < batchSize {
		s.done = true
	} else {
		s.curTime, s.curID = lastTime, lastID
	}
	return out, false, nil
}

// ── 横表列发现：DISTINCT(metric_path, metric_type) ────────────────────────────
//
// 横表表头必须先于数据写出，列集 = 所选指标全集（含类型）。指标编号/类型与设备/小区无关，
// 一次轻量 DISTINCT 即得，且类型直接取数据侧 metric_type（真值源，与行一致）。名字由 nameResolver 解析。

// colKey 是列发现阶段的 (编号, 类型) 对，名待解析。
type colKey struct {
	code  string
	mtype string
}

// discoverMetricColumns 发现 dashboard 源（device / aggregate 维度）的指标列集。
// 前端明确传 metric_paths 时，导出列必须按请求全集保留：页面 fill_empty 会让“窗口内无真实行但已选择”的指标仍显示为占位列。
// 未传 metric_paths（全量导出）才回退到数据侧 DISTINCT 发现。
func discoverMetricColumns(ctx context.Context, db PgQuerier, table string, metricPaths []string, start, end time.Time) ([]colKey, error) {
	if len(metricPaths) > 0 {
		return requestedMetricColumns(metricPaths), nil
	}
	sqlStr, args := buildDistinctMetricsSQL(table, metricPaths, start, end)
	rows, err := db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("export discover columns %s: %w", table, err)
	}
	defer rows.Close()
	return scanColKeys(rows)
}

func requestedMetricColumns(metricPaths []string) []colKey {
	out := make([]colKey, 0, len(metricPaths))
	seen := make(map[string]struct{}, len(metricPaths))
	for _, raw := range metricPaths {
		code := strings.TrimSpace(raw)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, colKey{code: code, mtype: metricColumnType(code)})
	}
	return out
}

func metricColumnType(code string) string {
	return string(metricTypeFromPath(code))
}

func metricTypeFromPath(code string) metrics.MetricType {
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(code)), "C") {
		return metrics.MetricTypeCounter
	}
	return metrics.MetricTypeKPI
}

// discoverAdhocColumns 发现 adhoc 源的指标列集，按编号升序。
func discoverAdhocColumns(ctx context.Context, db PgQuerier, taskID uuid.UUID, metricPaths []string, start, end time.Time) ([]colKey, error) {
	sqlStr, args := buildAdhocDistinctMetricsSQL(taskID, metricPaths, start, end)
	rows, err := db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("export discover adhoc columns: %w", err)
	}
	defer rows.Close()
	return scanColKeys(rows)
}

// scanColKeys 扫 (metric_path, metric_type) 两列 → colKey 切片，按编号升序求稳定列序。
func scanColKeys(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]colKey, error) {
	out := make([]colKey, 0)
	for rows.Next() {
		var code, mt string
		if err := rows.Scan(&code, &mt); err != nil {
			return nil, fmt.Errorf("export scan column key: %w", err)
		}
		out = append(out, colKey{code: code, mtype: mt})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("export column rows: %w", err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].code < out[j].code })
	return out, nil
}

// ── helper ──────────────────────────────────────────────────────────────────

func deviceLabel(oui, sn string) string {
	switch {
	case oui != "" && sn != "":
		return oui + "/" + sn
	case sn != "":
		return sn
	case oui != "":
		return oui
	default:
		return ""
	}
}

// deviceSNLabel 设备维度的设备列标识：只用 SN（与页面表格「设备 SN」列一致），
// SN 缺失时回退 OUI，二者皆空返回空串。
func deviceSNLabel(oui, sn string) string {
	if sn != "" {
		return sn
	}
	return oui
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ldnAllowSet 把 object_ldn 白名单转成查表集合；空白名单返回 nil（= 不过滤）。
func ldnAllowSet(ldns []string) map[string]bool {
	if len(ldns) == 0 {
		return nil
	}
	m := make(map[string]bool, len(ldns))
	for _, l := range ldns {
		m[l] = true
	}
	return m
}
