package device

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// ungroupedDevicesWhere 是"未分组设备"节点的过滤:设备未绑定任何分组(无成员关系行)。
// 2026-06-03 用户决策「未分组 = 未绑定任何分组」——选中默认 L2 组(DefaultLevel2GroupID)时,
// 不再按该组成员过滤,而是取 NOT EXISTS device_group_members 的设备。
const ungroupedDevicesWhere = "NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = d.id)"

// allowedSortColumnsWithInfo maps user-facing sort keys to qualified column names
// for the devices + device_info JOIN query.
// T-0162: status 列已 DROP，加 lifecycle_state + is_online。
var allowedSortColumnsWithInfo = map[string]string{
	"created_at":      "d.created_at",
	"updated_at":      "d.updated_at",
	"serial_number":   "d.serial_number",
	"lifecycle_state": "d.lifecycle_state",
	"is_online":       "d.is_online",
	"carrier":         "d.carrier",
	"technology":      "d.technology",
	"model":           "d.model_name",
	"manufacturer":    "d.manufacturer",
	"last_inform_at":  "d.last_inform_at",
	"device_name":     "di.device_name",
	"rf_status":       "di.rf_status",
	"cell_status":     "di.cell_status",
	"bandwidth":       "di.bandwidth",
	"transmit_power":  "di.transmit_power",
	"num_of_cells":    "di.num_of_cells",
	"gps_status":      "di.gps_status",
	// #361: 告警级别排序随显示口径切到聚合派生值 aa.top_sev（1=critical..4=warning，
	// NULL=无告警）。OrderBy 处对该列追加 NULLS LAST，使无告警设备恒排末尾。
	"alarm_severity":  "aa.top_sev",
	"license_status":  "di.license_status",
}

// PgDeviceInfoRepository implements DeviceInfoRepository using PostgreSQL.
type PgDeviceInfoRepository struct {
	pool *pgxpool.Pool
}

// NewPgDeviceInfoRepository creates a new PostgreSQL device info repository.
func NewPgDeviceInfoRepository(pool *pgxpool.Pool) *PgDeviceInfoRepository {
	return &PgDeviceInfoRepository{pool: pool}
}

func (r *PgDeviceInfoRepository) GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*DeviceInfo, error) {
	query, args, err := storage.Psql.Select(deviceInfoColumns()...).
		From("device_info").
		Where(sq.Eq{"device_id": deviceID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	info, err := scanDeviceInfoFromRow(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get device info: %w", err)
	}
	return info, nil
}

func (r *PgDeviceInfoRepository) Create(ctx context.Context, info *DeviceInfo) error {
	now := time.Now()
	info.CreatedAt = now
	info.UpdatedAt = now

	query, args, err := storage.Psql.Insert("device_info").
		Columns(
			"device_id", "device_name", "address", "remark", "project_status", "height",
			"eci", "pci", "cell_id", "freq_point", "bandwidth", "transmit_power", "plmn",
			"rf_status", "cell_status", "op_state", "mme_status", "sync_status", "kpi_status",
			"num_of_cells", "gps_status", "alarm_severity", "license_status",
			"mac", "hardware_version",
			// T-0173: 显式带上 last_online_time + cumulative_online_duration,与读路径对齐。
			"first_online_time", "last_online_time", "last_offline_time",
			"run_time", "cumulative_online_duration",
			"creator", "updater", "created_at", "updated_at",
		).
		Values(
			info.DeviceID, info.DeviceName, info.Address, info.Remark, info.ProjectStatus, info.Height,
			info.ECI, info.PCI, info.CellID, info.FreqPoint, info.Bandwidth, info.TransmitPower, info.PLMN,
			info.RFStatus, info.CellStatus, info.OpState, info.MMEStatus, info.SyncStatus, info.KPIStatus,
			info.NumOfCells, info.GPSStatus, info.AlarmSeverity, info.LicenseStatus,
			info.MAC, info.HardwareVersion,
			info.FirstOnlineTime, info.LastOnlineTime, info.LastOfflineTime,
			info.RunTime, info.CumulativeOnlineDuration,
			info.Creator, info.Updater, info.CreatedAt, info.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert device_info: %w", err)
	}
	return nil
}

func (r *PgDeviceInfoRepository) UpdateManualFields(ctx context.Context, deviceID uuid.UUID, req UpdateDeviceInfoRequest, updater string) error {
	builder := storage.Psql.Update("device_info").Where(sq.Eq{"device_id": deviceID})

	if req.DeviceName != nil {
		builder = builder.Set("device_name", *req.DeviceName)
	}
	if req.Address != nil {
		builder = builder.Set("address", *req.Address)
	}
	if req.Remark != nil {
		builder = builder.Set("remark", *req.Remark)
	}
	if req.ProjectStatus != nil {
		builder = builder.Set("project_status", *req.ProjectStatus)
	}
	if req.Height != nil {
		builder = builder.Set("height", *req.Height)
	}
	builder = builder.Set("updater", updater)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device_info manual fields: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 设备不存在或尚无 device_info 行（首次 sync 之前）——返回 sentinel
		// ErrNotFound,由 handler 经 HTTPStatusFromError 映射 404（issue #145 A 项）。
		return fmt.Errorf("device_info not found for device %s: %w", deviceID, commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgDeviceInfoRepository) UpdateSyncFields(ctx context.Context, deviceID uuid.UUID, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	builder := storage.Psql.Update("device_info").Where(sq.Eq{"device_id": deviceID})
	for col, val := range fields {
		builder = builder.Set(col, val)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build sync update query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device_info sync fields: %w", err)
	}
	return nil
}

// GetTopologyAttributes returns lac / tac for a device. NULL or empty cells
// are omitted from the map so callers can use `_, ok := m[col]` to detect
// "this column had no value before".
func (r *PgDeviceInfoRepository) GetTopologyAttributes(ctx context.Context, deviceID uuid.UUID) (map[string]string, error) {
	const sqlText = `SELECT lac, tac FROM device_info WHERE device_id = $1`
	var lac, tac *string
	if err := r.pool.QueryRow(ctx, sqlText, deviceID).Scan(&lac, &tac); err != nil {
		// 设备无 device_info 行（新设备首次 sync 之前）—— 不算错误，返回空 map
		if errors.Is(err, pgx.ErrNoRows) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("query topology attributes: %w", err)
	}
	out := make(map[string]string, 2)
	if lac != nil && *lac != "" {
		out["lac"] = *lac
	}
	if tac != nil && *tac != "" {
		out["tac"] = *tac
	}
	return out, nil
}

// softwareVersionDeviceIDSubquery 构造 software_version 过滤用的 device_id 子查询
// （供 device list 主查询与 count 查询共用，并便于单测验证 DISTINCT 优化，#11）。
//
// 用 SELECT DISTINCT device_id：device_parameters 对同一 device_id 可能存在多行
// （历史保留 / 多 instance 路径），不去重则 IN (...) 半连接需扫并去重更多行，百万设备
// 规模下浪费 I/O；DISTINCT 让 PG 先 HashAggregate 收口为唯一 device_id 集合。
func softwareVersionDeviceIDSubquery(versions []string) sq.SelectBuilder {
	return sq.Select("DISTINCT device_id").From("device_parameters").
		Where(sq.Eq{"parameter_path": "Device.DeviceInfo.SoftwareVersion"}).
		Where(sq.Eq{"parameter_value": versions})
}

func (r *PgDeviceInfoRepository) ListDevicesWithInfo(ctx context.Context, filter DeviceFilter) (*model.ListResponse[DeviceWithInfo], error) {
	selectCols := deviceWithInfoSelectColumns()
	builder := storage.Psql.Select(selectCols...).
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id").
		LeftJoin("device_group_members dgm ON dgm.device_id = d.id").
		LeftJoin("device_groups dg ON dg.id = dgm.group_id").
		LeftJoin(alarmsActiveAggJoin). // #361: 告警级别/告警数实时聚合
		Where(sq.Eq{"d.deleted_at": nil})
	countBuilder := storage.Psql.Select("COUNT(*)").
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id").
		LeftJoin("device_group_members dgm ON dgm.device_id = d.id").
		Where(sq.Eq{"d.deleted_at": nil})

	// Data permission filter: VisibleGroups semantics:
	//   nil          → superadmin (v1.0：source='builtIn'，由 PermissionService 上游决定)，no filtering (see all devices)
	//   []uuid.UUID{} → no permissions, return empty result
	//   [id1, id2]   → filter to devices in these groups
	// Note: dgm (device_group_members) is already joined via LeftJoin at line 167
	// 注：filter.Carrier 是**设备**的 carrier (devices.carrier，物理属性)，不是用户的 carrier；users.carrier 在 v1.0 已删除。
	if filter.VisibleGroups != nil && len(filter.VisibleGroups) == 0 {
		// User has no group permissions — short-circuit to empty result.
		builder = builder.Where("FALSE")
		countBuilder = countBuilder.Where("FALSE")
	} else if filter.GroupID != nil || len(filter.VisibleGroups) > 0 {
		// dgm is already joined, just add WHERE conditions
		if filter.GroupID != nil {
			if filter.GroupID.String() == global.DefaultLevel2GroupID {
				builder = builder.Where(ungroupedDevicesWhere)
				countBuilder = countBuilder.Where(ungroupedDevicesWhere)
			} else {
				builder = builder.Where(sq.Eq{"dgm.group_id": *filter.GroupID})
				countBuilder = countBuilder.Where(sq.Eq{"dgm.group_id": *filter.GroupID})
			}
		}
		if len(filter.VisibleGroups) > 0 {
			builder = builder.Where(sq.Eq{"dgm.group_id": filter.VisibleGroups})
			countBuilder = countBuilder.Where(sq.Eq{"dgm.group_id": filter.VisibleGroups})
		}
	}

	// Apply filters from devices table
	if filter.Carrier != nil {
		builder = builder.Where(sq.Eq{"d.carrier": *filter.Carrier})
		countBuilder = countBuilder.Where(sq.Eq{"d.carrier": *filter.Carrier})
	}
	if filter.Technology != nil {
		builder = builder.Where(sq.Eq{"d.technology": *filter.Technology})
		countBuilder = countBuilder.Where(sq.Eq{"d.technology": *filter.Technology})
	}
	// T-0162: filter.Status 老字段过渡兼容——翻译为 lifecycle_state + is_online
	if filter.Status != nil {
		lifecycle, isOnline := DeriveLifecycleFromStatus(*filter.Status)
		builder = builder.Where(sq.Eq{"d.lifecycle_state": lifecycle})
		countBuilder = countBuilder.Where(sq.Eq{"d.lifecycle_state": lifecycle})
		if *filter.Status == model.DeviceActive || *filter.Status == model.DeviceOffline {
			builder = builder.Where(sq.Eq{"d.is_online": isOnline})
			countBuilder = countBuilder.Where(sq.Eq{"d.is_online": isOnline})
		}
	}
	// T-0162: 新筛选维度
	if len(filter.LifecycleState) > 0 {
		builder = builder.Where(sq.Eq{"d.lifecycle_state": filter.LifecycleState})
		countBuilder = countBuilder.Where(sq.Eq{"d.lifecycle_state": filter.LifecycleState})
	}
	if filter.IsOnline != nil {
		builder = builder.Where(sq.Eq{"d.is_online": *filter.IsOnline})
		countBuilder = countBuilder.Where(sq.Eq{"d.is_online": *filter.IsOnline})
	}
	// T-0162: 4 个 device list multi-select 筛选字段（model_name /
	// firmware_version / software_version / product_class）支持 CSV 多值。
	// SplitCSV 单值场景返单元素切片，sq.Eq{slice} 自动展开为 IN (...)，行为与
	// 原 sq.Eq{string} 完全等价；多值场景才走真正的 IN 多值过滤。
	if filter.ModelName != nil && *filter.ModelName != "" {
		vs := SplitCSV(*filter.ModelName)
		builder = builder.Where(sq.Eq{"d.model_name": vs})
		countBuilder = countBuilder.Where(sq.Eq{"d.model_name": vs})
	}
	if filter.FirmwareVersion != nil && *filter.FirmwareVersion != "" {
		vs := SplitCSV(*filter.FirmwareVersion)
		builder = builder.Where(sq.Eq{"d.firmware_version": vs})
		countBuilder = countBuilder.Where(sq.Eq{"d.firmware_version": vs})
	}
	// T-0162: software_version 走 device_parameters TR-069 标准路径，不在
	// device_info 表（与 seed/000137 device_parameters 灌入 distinct 一致）。
	if filter.SoftwareVersion != nil && *filter.SoftwareVersion != "" {
		sub := softwareVersionDeviceIDSubquery(SplitCSV(*filter.SoftwareVersion))
		builder = builder.Where(sq.Expr("d.id IN (?)", sub))
		countBuilder = countBuilder.Where(sq.Expr("d.id IN (?)", sub))
	}
	if filter.OUI != nil {
		builder = builder.Where(sq.Eq{"d.oui": *filter.OUI})
		countBuilder = countBuilder.Where(sq.Eq{"d.oui": *filter.OUI})
	}
	if filter.SN != nil && *filter.SN != "" {
		builder = builder.Where(sq.Eq{"d.serial_number": *filter.SN})
		countBuilder = countBuilder.Where(sq.Eq{"d.serial_number": *filter.SN})
	}
	// 批量输入：SN 列表精确过滤（serial_number IN (...)）。主列表 inline 路径之前漏了此条件，
	// 导致 ?sn_list= 在 /devices 列表静默失效（同 product_id 问题）。
	if len(filter.SNList) > 0 {
		builder = builder.Where(sq.Eq{"d.serial_number": filter.SNList})
		countBuilder = countBuilder.Where(sq.Eq{"d.serial_number": filter.SNList})
	}

	// Apply filters from device_info table
	if filter.Manufacturer != nil && *filter.Manufacturer != "" {
		builder = builder.Where(sq.Eq{"d.manufacturer": *filter.Manufacturer})
		countBuilder = countBuilder.Where(sq.Eq{"d.manufacturer": *filter.Manufacturer})
	}
	if filter.ProductClass != nil && *filter.ProductClass != "" {
		vs := SplitCSV(*filter.ProductClass)
		builder = builder.Where(sq.Eq{"d.product_class": vs})
		countBuilder = countBuilder.Where(sq.Eq{"d.product_class": vs})
	}
	// 产品装配件 UUID 过滤（下拉来自 /products）。主列表 builder/countBuilder 之前漏了此条件
	// （只在 applyDeviceFilters 子查询里有），导致 ?product_id= 在 /devices 列表静默失效。
	if filter.ProductID != nil {
		builder = builder.Where(sq.Eq{"d.product_id": *filter.ProductID})
		countBuilder = countBuilder.Where(sq.Eq{"d.product_id": *filter.ProductID})
	}
	if filter.RFStatus != nil && *filter.RFStatus != "" {
		builder = builder.Where(sq.Eq{"di.rf_status": *filter.RFStatus})
		countBuilder = countBuilder.Where(sq.Eq{"di.rf_status": *filter.RFStatus})
	}
	if filter.CellStatus != nil && *filter.CellStatus != "" {
		builder = builder.Where(sq.Eq{"di.cell_status": *filter.CellStatus})
		countBuilder = countBuilder.Where(sq.Eq{"di.cell_status": *filter.CellStatus})
	}
	if filter.ProjectStatus != nil && *filter.ProjectStatus != "" {
		builder = builder.Where(sq.Eq{"di.project_status": *filter.ProjectStatus})
		countBuilder = countBuilder.Where(sq.Eq{"di.project_status": *filter.ProjectStatus})
	}
	if filter.GPSStatus != nil && *filter.GPSStatus != "" {
		builder = builder.Where(sq.Eq{"di.gps_status": *filter.GPSStatus})
		countBuilder = countBuilder.Where(sq.Eq{"di.gps_status": *filter.GPSStatus})
	}
	if filter.AlarmSeverity != nil && *filter.AlarmSeverity != "" {
		// #361: 告警级别筛选随显示口径切到 alarms_active 实时聚合（不再用无人维护的
		// di.alarm_severity 冗余列）。用相关子查询匹配「该设备未 cleared 活动告警的
		// 最严重级别(MIN severity) = 请求级别」，与列表展示的告警级别一致。
		if cond := alarmSeverityFilterCond(*filter.AlarmSeverity); cond != nil {
			builder = builder.Where(cond)
			countBuilder = countBuilder.Where(cond)
		}
	}
	if filter.LicenseStatus != nil && *filter.LicenseStatus != "" {
		builder = builder.Where(sq.Eq{"di.license_status": *filter.LicenseStatus})
		countBuilder = countBuilder.Where(sq.Eq{"di.license_status": *filter.LicenseStatus})
	}
	if filter.OpState != nil {
		if cond := opStateFilterCond(*filter.OpState); cond != nil {
			builder = builder.Where(cond)
			countBuilder = countBuilder.Where(cond)
		}
	}

	// Multi-field fuzzy search (G07) — 升级为多关键字（英文逗号分隔，最多 50）。
	// 任一关键字命中任一字段即匹配（设备级 OR）。单值场景与老行为完全等价。
	// caller 端 UX：前端搜索框 placeholder "SN/名称/IP/MAC/PCI"，目前后端实际
	// 匹配 SN/site_name/manufacturer/model_name/device_name/address 6 字段，
	// IP/MAC/PCI 字段扩展是另一个独立 task（见 search.go 文件头注释）。
	if filter.Search != nil {
		if cond := BuildSearchOR(*filter.Search, []string{
			"d.serial_number",
			"d.site_name",
			"d.manufacturer",
			"d.model_name",
			"di.device_name",
			"di.address",
		}); cond != nil {
			builder = builder.Where(cond)
			countBuilder = countBuilder.Where(cond)
		}
	}

	// Count total
	countQuery, countArgs, _ := countBuilder.ToSql()
	var total int64
	r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)

	// Sorting with table-qualified column names
	sortCol := "d.created_at"
	if filter.SortBy != "" {
		if col, ok := allowedSortColumnsWithInfo[filter.SortBy]; ok {
			sortCol = col
		}
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
	}
	orderClause := sortCol + " " + sortDir
	// #361: 告警级别按聚合派生值排序时，无告警(NULL)设备恒排末尾（与「无」语义一致）。
	if sortCol == "aa.top_sev" {
		orderClause += " NULLS LAST"
	}
	builder = builder.
		OrderBy(orderClause).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list with info query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list devices with info: %w", err)
	}
	defer rows.Close()

	var items []DeviceWithInfo
	for rows.Next() {
		d, err := scanDeviceWithInfoRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *d)
	}

	if items == nil {
		items = []DeviceWithInfo{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// GetByIDWithInfo 取单个设备 + device_info + 分组（与 ListDevicesWithInfo 同 JOIN）。
// 设备不存在或已软删返回 (nil, nil)。
func (r *PgDeviceInfoRepository) GetByIDWithInfo(ctx context.Context, deviceID uuid.UUID) (*DeviceWithInfo, error) {
	query, args, err := storage.Psql.
		Select(deviceWithInfoSelectColumns()...).
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id").
		LeftJoin("device_group_members dgm ON dgm.device_id = d.id").
		LeftJoin("device_groups dg ON dg.id = dgm.group_id").
		LeftJoin(alarmsActiveAggJoin). // #361: 告警级别/告警数实时聚合
		Where(sq.Eq{"d.id": deviceID}).
		Where(sq.Eq{"d.deleted_at": nil}).
		Limit(1). // 设备可能属多组，JOIN 可能出多行；详情只取一行
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get device with info query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query device with info: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil // 不存在或已软删
	}
	d, err := scanDeviceWithInfoRow(rows)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// ComputeListStats 在 ListDevicesWithInfo 同样筛选条件下跑 group-by 聚合，
// 返回 lifecycle/online/alarm 三维统计。T-0162 D5：取代前端用当前页 items
// filter() 自行估算的不准做法。
//
// 注意：本方法用与 ListDevicesWithInfo 相同的 LEFT JOIN，但只 GROUP BY 设备
// 维度（DISTINCT device_id），避免 dgm 一对多导致同设备被计数多次。
func (r *PgDeviceInfoRepository) ComputeListStats(ctx context.Context, filter DeviceFilter) (*DeviceListStats, error) {
	// 用子查询去重再聚合：先按筛选条件取所有命中设备的 (id, lifecycle, is_online)，
	// 再 GROUP BY。避免 dgm/dg LEFT JOIN 引起的设备重复计数。
	// #361: alarmed 标记——该设备是否存在未 cleared 活动告警（相关子查询 EXISTS）。
	// 放进 DISTINCT 子查询的 SELECT 列里，外层用 COUNT FILTER 数有告警的设备，
	// 与列表『告警级别』非『无』的口径一致（同一 alarms_active status<>'cleared' 源）。
	const alarmedFlagExpr = `EXISTS (
		SELECT 1 FROM alarms_active aa
		WHERE aa.device_id = d.id AND aa.status <> 'cleared'
	) AS alarmed`
	subBuilder := storage.Psql.Select("DISTINCT d.id", "d.lifecycle_state", "d.is_online", alarmedFlagExpr).
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id").
		LeftJoin("device_group_members dgm ON dgm.device_id = d.id").
		LeftJoin("device_groups dg ON dg.id = dgm.group_id").
		Where(sq.Eq{"d.deleted_at": nil})

	// 复用 ListDevicesWithInfo 的过滤逻辑——直接调 buildListFilters 抽出来更好，
	// 本期最小集：内联 carrier/technology/lifecycle/is_online/model_name 等核心
	// 筛选条件（与 ListDevicesWithInfo 保持一致）。
	subBuilder = applyDeviceFilters(subBuilder, filter)

	// 包到外层 GROUP BY
	subQ, subArgs, err := subBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build stats subquery: %w", err)
	}

	// #361: 外层 GROUP BY 增加 alarmed 真实统计——COUNT FILTER 数有活动告警的设备。
	// alarmed 已是子查询每设备唯一一行的布尔，外层直接 COUNT(*) FILTER 即得
	// COUNT(DISTINCT device_id with active alarm)，与 list 列『告警级别』非『无』一致。
	groupQ := "SELECT lifecycle_state, is_online, COUNT(*), " +
		"COUNT(*) FILTER (WHERE alarmed) FROM (" + subQ +
		") s GROUP BY lifecycle_state, is_online"

	rows, err := r.pool.Query(ctx, groupQ, subArgs...)
	if err != nil {
		return nil, fmt.Errorf("query device list stats: %w", err)
	}
	defer rows.Close()

	stats := &DeviceListStats{
		ByLifecycle: make(map[model.DeviceLifecycle]int64),
	}
	for rows.Next() {
		var lifecycle model.DeviceLifecycle
		var isOnline bool
		var count int64
		var alarmedCount int64
		if err := rows.Scan(&lifecycle, &isOnline, &count, &alarmedCount); err != nil {
			return nil, fmt.Errorf("scan device list stats: %w", err)
		}
		stats.Total += count
		stats.ByLifecycle[lifecycle] += count
		stats.Alarmed += alarmedCount
		if isOnline {
			stats.OnlineCount += count
		} else {
			stats.OfflineCount += count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device list stats: %w", err)
	}
	return stats, nil
}

// applyDeviceFilters 把 DeviceFilter 的 WHERE 子句应用到 squirrel SelectBuilder
// 上。抽出来供 ListDevicesWithInfo + ComputeListStats 共用，确保两个查询的
// 筛选条件 1:1 对齐（否则 stats 与 list 数量不一致就是 P3 重蹈 Q2 覆辙）。
func applyDeviceFilters(b sq.SelectBuilder, filter DeviceFilter) sq.SelectBuilder {
	if filter.Carrier != nil {
		b = b.Where(sq.Eq{"d.carrier": *filter.Carrier})
	}
	if filter.Technology != nil {
		b = b.Where(sq.Eq{"d.technology": *filter.Technology})
	}
	if filter.Status != nil {
		lifecycle, isOnline := DeriveLifecycleFromStatus(*filter.Status)
		b = b.Where(sq.Eq{"d.lifecycle_state": lifecycle})
		if *filter.Status == model.DeviceActive || *filter.Status == model.DeviceOffline {
			b = b.Where(sq.Eq{"d.is_online": isOnline})
		}
	}
	if len(filter.LifecycleState) > 0 {
		b = b.Where(sq.Eq{"d.lifecycle_state": filter.LifecycleState})
	}
	if filter.IsOnline != nil {
		b = b.Where(sq.Eq{"d.is_online": *filter.IsOnline})
	}
	if filter.ModelName != nil && *filter.ModelName != "" {
		b = b.Where(sq.Eq{"d.model_name": SplitCSV(*filter.ModelName)})
	}
	if filter.FirmwareVersion != nil && *filter.FirmwareVersion != "" {
		b = b.Where(sq.Eq{"d.firmware_version": SplitCSV(*filter.FirmwareVersion)})
	}
	if filter.SoftwareVersion != nil && *filter.SoftwareVersion != "" {
		sub := softwareVersionDeviceIDSubquery(SplitCSV(*filter.SoftwareVersion))
		b = b.Where(sq.Expr("d.id IN (?)", sub))
	}
	if filter.OUI != nil {
		b = b.Where(sq.Eq{"d.oui": *filter.OUI})
	}
	if filter.SN != nil && *filter.SN != "" {
		b = b.Where(sq.Eq{"d.serial_number": *filter.SN})
	}
	if len(filter.SNList) > 0 {
		// 批量输入：按 SN 列表精确过滤（serial_number IN (...)）。
		b = b.Where(sq.Eq{"d.serial_number": filter.SNList})
	}
	if filter.Manufacturer != nil && *filter.Manufacturer != "" {
		b = b.Where(sq.Eq{"d.manufacturer": *filter.Manufacturer})
	}
	if filter.ProductID != nil {
		// 产品装配件 UUID 过滤（下拉来自 /products）。之前漏在本函数实现，
		// 导致 ?product_id= 在 /devices 列表静默失效（List 方法有、此 live 路径无）。
		b = b.Where(sq.Eq{"d.product_id": *filter.ProductID})
	}
	if filter.ProductClass != nil && *filter.ProductClass != "" {
		b = b.Where(sq.Eq{"d.product_class": SplitCSV(*filter.ProductClass)})
	}
	if filter.GroupID != nil {
		if filter.GroupID.String() == global.DefaultLevel2GroupID {
			b = b.Where(ungroupedDevicesWhere)
		} else {
			b = b.Where(sq.Eq{"dgm.group_id": *filter.GroupID})
		}
	}
	if filter.VisibleGroups != nil && len(filter.VisibleGroups) == 0 {
		b = b.Where("FALSE")
	} else if len(filter.VisibleGroups) > 0 {
		b = b.Where(sq.Eq{"dgm.group_id": filter.VisibleGroups})
	}
	if filter.Search != nil {
		if cond := BuildSearchOR(*filter.Search, []string{
			"d.serial_number", "d.site_name", "d.manufacturer",
			"d.model_name", "di.device_name", "di.address",
		}); cond != nil {
			b = b.Where(cond)
		}
	}
	if filter.OpState != nil {
		if cond := opStateFilterCond(*filter.OpState); cond != nil {
			b = b.Where(cond)
		}
	}
	return b
}

// opStateFilterCond 构造 op_state 列("1"=激活/"0"=未激活)的 WHERE 条件。
// 口径同 InfoSyncer.CalcOpState、与 cell_status 严格同源派生：任一 cell active → "1" 否则 "0"。
// migrations/000003 DEFAULT '0' 后 NULL 理论不出现，但 LEFT JOIN 设备无 device_info 行
// 时 di.op_state 列会是 NULL，filter "0" 路径并入 NULL 作为“未激活”。
//
// 抽出作 helper 是为了让 ListDevicesWithInfo 与 applyDeviceFilters（供 ComputeListStats 用）
// 两处过滤口径 1:1 对齐，避免 list/stats 数量不一致的 P3 回归。
func opStateFilterCond(value string) sq.Sqlizer {
	switch value {
	case "1":
		return sq.Eq{"di.op_state": "1"}
	case "0":
		return sq.Or{sq.Eq{"di.op_state": "0"}, sq.Eq{"di.op_state": nil}}
	default:
		return nil
	}
}

// deviceInfoColumns returns column names for the device_info table.
func deviceInfoColumns() []string {
	return []string{
		"device_id",
		"device_name", "address", "remark", "project_status", "height",
		"eci", "pci", "cell_id", "freq_point", "bandwidth", "transmit_power", "plmn",
		"rf_status", "cell_status", "op_state", "mme_status", "sync_status", "kpi_status",
		"num_of_cells", "gps_status", "alarm_severity", "license_status",
		"mac", "hardware_version",
		// T-0173: 补齐 last_online_time（pre-existing 漏读) + 新增 cumulative_online_duration
		"first_online_time", "last_online_time", "last_offline_time",
		"run_time", "cumulative_online_duration",
		// Phase 2/3 (设计文档 §4.2)：扩展列
		"tac", "band", "ul_earfcn",
		"subframe_assignment", "special_subframe", "root_index",
		"gps_satellites", "gps_height", "lock_status",
		"enb_id", "network_model",
		"creator", "updater", "created_at", "updated_at",
	}
}

// deviceWithInfoSelectColumns returns qualified column names for the JOIN query.
// T-0162: d.status → d.lifecycle_state + d.is_online；离线时长 CASE 改用 is_online。
func deviceWithInfoSelectColumns() []string {
	// "离线" 在 T-0162 解耦后定义为 d.is_online=FALSE（仅 commissioned 状态下才有
	// "离线时长"概念；registered/provisioning 等还没入网的状态不计算"离线时长"）。
	offlineCond := `d.lifecycle_state = 'commissioned' AND d.is_online = FALSE`
	return []string{
		// devices columns (aliased with d.)
		"d.id", "d.serial_number", "d.oui", "d.product_class", "d.manufacturer", "d.model_name",
		"d.carrier", "d.technology",
		"d.lifecycle_state", "d.is_online", // T-0162: 替代 d.status
		"d.firmware_version",
		"host(d.ip_address) as ip_address", "d.connection_request_url",
		"d.nat_detected", "d.udp_connection_request_address",
		"d.last_inform_at", "d.last_inform_events",
		"d.last_boot_at", "d.boot_count",
		"d.inform_interval", "d.site_name", "d.site_id", "d.latitude", "d.longitude",
		"d.extension_data", "d.created_at", "d.updated_at",
		"d.last_offline_reason", // T-0173: 离线原因诊断（migration 000184)
		// device_groups columns
		"dg.name as group_name",
		// device_info columns
		"di.device_name", "di.address", "di.remark", "di.project_status", "di.height",
		"di.eci", "di.pci", "di.cell_id", "di.freq_point", "di.bandwidth", "di.transmit_power", "di.plmn",
		"di.rf_status", "di.cell_status", "di.op_state", "di.mme_status", "di.sync_status", "di.kpi_status",
		"di.num_of_cells", "di.gps_status",
		// #361: 告警级别不再读 di.alarm_severity（该冗余列仅 Radisys 自报路径写、
		// 与 OMC 告警引擎无关、从无人维护）。改实时 JOIN alarms_active 子查询 aa，
		// 取每设备未 cleared 活动告警最严重级别(MIN(severity)，1=critical..4=warning)
		// 映成文本；无活动告警 → NULL → 前端归 'none'。
		`CASE aa.top_sev
			WHEN 1 THEN 'critical'
			WHEN 2 THEN 'major'
			WHEN 3 THEN 'minor'
			WHEN 4 THEN 'warning'
			ELSE NULL
		END AS alarm_severity`,
		"di.license_status",
		"di.mac", "di.hardware_version",
		"di.first_online_time", "di.last_online_time", "di.last_offline_time", "di.run_time",
		"di.cumulative_online_duration", // T-0173: OMC 视角累计在线时长
		// Phase 2/3 (设计文档 §4.2 Layer E)：device_info 扩展列
		"di.tac", "di.band", "di.ul_earfcn",
		"di.subframe_assignment", "di.special_subframe", "di.root_index",
		"di.gps_satellites", "di.gps_height", "di.lock_status",
		"di.enb_id", "di.network_model",
		// 在线时长派生（设计文档 §13）：
		//   - is_online → 当前已在线多久（NOW - last_online_time）
		//   - 离线后 → 上次在线区间长度（last_offline_time - last_online_time）
		//   - last_online_time IS NULL → NULL（设备从未上线）
		`CASE
			WHEN di.last_online_time IS NULL THEN NULL
			WHEN d.is_online THEN EXTRACT(EPOCH FROM (NOW() - di.last_online_time))::bigint
			WHEN di.last_offline_time IS NOT NULL AND di.last_offline_time > di.last_online_time
				THEN EXTRACT(EPOCH FROM (di.last_offline_time - di.last_online_time))::bigint
			ELSE NULL
		END AS online_duration`,
		// 离线时长计算（SQL层面）：T-0162 用 lifecycle+is_online 判定
		`CASE
			WHEN ` + offlineCond + ` AND di.last_offline_time IS NOT NULL
			THEN EXTRACT(EPOCH FROM (NOW() - di.last_offline_time))::bigint
			ELSE NULL
		END AS offline_seconds`,
		`CASE
			WHEN ` + offlineCond + ` AND di.last_offline_time IS NOT NULL
			THEN FLOOR(EXTRACT(EPOCH FROM (NOW() - di.last_offline_time)) / 86400)::bigint
			ELSE NULL
		END AS offline_days`,
		`CASE
			WHEN ` + offlineCond + ` AND di.last_offline_time IS NOT NULL
			THEN FLOOR((EXTRACT(EPOCH FROM (NOW() - di.last_offline_time)) % 86400) / 3600)::bigint
			ELSE NULL
		END AS offline_hours`,
		`CASE
			WHEN ` + offlineCond + ` AND di.last_offline_time IS NOT NULL
			THEN FLOOR((EXTRACT(EPOCH FROM (NOW() - di.last_offline_time)) % 3600) / 60)::bigint
			ELSE NULL
		END AS offline_minutes`,
		// #361: 该设备未 cleared 活动告警数（来自 alarms_active 聚合子查询 aa）。
		// 无活动告警时 LEFT JOIN 命中空 → NULL → 前端归 0。
		"aa.active_alarm_count",
	}
}

// alarmsActiveAggJoin 是设备列表/单设备查询用的 alarms_active 聚合子查询 JOIN 子句。
// 按 device_id 聚合每设备未 cleared(status<>'cleared') 活动告警的最严重级别
// （MIN(severity)，smallint 1=critical..4=warning，最严重=数值最小）与告警数。
// #361：把告警级别列从无人维护的 di.alarm_severity 冗余列切到实时聚合派生值。
const alarmsActiveAggJoin = `(
	SELECT device_id,
	       MIN(severity)  AS top_sev,
	       COUNT(*)       AS active_alarm_count
	FROM alarms_active
	WHERE status <> 'cleared'
	GROUP BY device_id
) aa ON aa.device_id = d.id`

// alarmSeverityTextToNum 把前端 AlarmSeverity 文本映成 alarms_active.severity
// (smallint 1=critical..4=warning)。未知文本返回 0（调用方据此跳过过滤）。
func alarmSeverityTextToNum(text string) int {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "critical":
		return 1
	case "major":
		return 2
	case "minor":
		return 3
	case "warning":
		return 4
	default:
		return 0
	}
}

// alarmSeverityFilterCond 构造「该设备未 cleared 活动告警最严重级别 = 请求级别」
// 的相关子查询条件（#361，与列表展示口径一致）。未知级别返回 nil（不过滤）。
func alarmSeverityFilterCond(text string) sq.Sqlizer {
	num := alarmSeverityTextToNum(text)
	if num == 0 {
		return nil
	}
	return sq.Expr(
		`(SELECT MIN(aaf.severity) FROM alarms_active aaf
		   WHERE aaf.device_id = d.id AND aaf.status <> 'cleared') = ?`,
		num,
	)
}

func scanDeviceInfoFromRow(row pgx.Row) (*DeviceInfo, error) {
	var info DeviceInfo
	err := row.Scan(
		&info.DeviceID,
		&info.DeviceName, &info.Address, &info.Remark, &info.ProjectStatus, &info.Height,
		&info.ECI, &info.PCI, &info.CellID, &info.FreqPoint, &info.Bandwidth, &info.TransmitPower, &info.PLMN,
		&info.RFStatus, &info.CellStatus, &info.OpState, &info.MMEStatus, &info.SyncStatus, &info.KPIStatus,
		&info.NumOfCells, &info.GPSStatus, &info.AlarmSeverity, &info.LicenseStatus,
		&info.MAC, &info.HardwareVersion,
		// T-0173: 顺序与 deviceInfoColumns() 对齐;补齐 last_online_time + cumulative_online_duration
		&info.FirstOnlineTime, &info.LastOnlineTime, &info.LastOfflineTime,
		&info.RunTime, &info.CumulativeOnlineDuration,
		// Phase 2/3 (设计文档 §4.2)：扩展列与 deviceInfoColumns 顺序一致
		&info.TAC, &info.Band, &info.ULEarfcn,
		&info.SubframeAssignment, &info.SpecialSubframe, &info.RootIndex,
		&info.GPSSatellites, &info.GPSHeight, &info.LockStatus,
		&info.EnbID, &info.NetworkModel,
		&info.Creator, &info.Updater, &info.CreatedAt, &info.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func scanDeviceWithInfoRow(rows pgx.Rows) (*DeviceWithInfo, error) {
	var d DeviceWithInfo
	var extData, eventsData []byte
	var ipAddr, udpAddr *string
	// nullable string columns from devices table
	var productClass, manufacturer, modelName *string
	var firmwareVersion, connReqURL, siteName, siteID *string

	// device_info nullable fields
	var (
		diDeviceName    *string
		diAddress       *string
		diRemark        *string
		diProjectStatus *string
		diHeight        *float64
		diECI           *string
		diPCI           *string
		diCellID        *string
		diFreqPoint     *string
		diBandwidth     *float64
		diTransmitPower *float64
		diPLMN          *string
		diRFStatus      *string
		diCellStatus    *string
		diOpState       *string
		diMMEStatus     *string
		diSyncStatus    *string
		diKPIStatus     *string
		diNumOfCells    *int
		diGPSStatus     *string
		diAlarmSeverity *string
		diLicenseStatus *string
		diMAC           *string
		diHWVersion     *string
		diFirstOnline   *time.Time
		diLastOnline    *time.Time
		diLastOffline   *time.Time
		diRunTime       *int64
		diCumOnline     *int64 // T-0173: cumulative_online_duration
		// Phase 2/3 (设计文档 §4.2)：扩展列接收变量
		diTAC                *string
		diBand               *string
		diULEarfcn           *string
		diSubframeAssignment *string
		diSpecialSubframe    *string
		diRootIndex          *string
		diGPSSatellites      *int
		diGPSHeight          *float64
		diLockStatus         *string
		diEnbID              *string
		diNetworkModel       *string
		// 在线时长派生（SQL计算，设计文档 §13）
		onlineDuration *int64
		// 离线时长（SQL计算）
		offlineSeconds *int64
		offlineDays    *int64
		offlineHours   *int64
		offlineMinutes *int64
		// #361: 活动告警数（alarms_active 聚合，LEFT JOIN 未命中→NULL）
		activeAlarmCount *int
	)

	err := rows.Scan(
		// devices fields
		&d.ID, &d.SerialNumber, &d.OUI, &productClass, &manufacturer, &modelName,
		&d.Carrier, &d.Technology,
		&d.LifecycleState, &d.IsOnline, // T-0162: 替代 &d.Status
		&firmwareVersion,
		&ipAddr, &connReqURL,
		&d.NatDetected, &udpAddr,
		&d.LastInformAt, &eventsData,
		&d.LastBootAt, &d.BootCount,
		&d.InformInterval, &siteName, &siteID, &d.Latitude, &d.Longitude,
		&extData, &d.CreatedAt, &d.UpdatedAt,
		&d.LastOfflineReason, // T-0173: 离线原因（migration 000184)
		// device_groups field (nullable from LEFT JOIN)
		&d.GroupName,
		// device_info fields (all nullable from LEFT JOIN)
		&diDeviceName, &diAddress, &diRemark, &diProjectStatus, &diHeight,
		&diECI, &diPCI, &diCellID, &diFreqPoint, &diBandwidth, &diTransmitPower, &diPLMN,
		&diRFStatus, &diCellStatus, &diOpState, &diMMEStatus, &diSyncStatus, &diKPIStatus,
		&diNumOfCells, &diGPSStatus, &diAlarmSeverity, &diLicenseStatus,
		&diMAC, &diHWVersion,
		&diFirstOnline, &diLastOnline, &diLastOffline, &diRunTime,
		&diCumOnline, // T-0173: cumulative_online_duration（与 select 列顺序一致)
		// Phase 2/3 扩展列
		&diTAC, &diBand, &diULEarfcn,
		&diSubframeAssignment, &diSpecialSubframe, &diRootIndex,
		&diGPSSatellites, &diGPSHeight, &diLockStatus,
		&diEnbID, &diNetworkModel,
		// 在线时长派生
		&onlineDuration,
		// 离线时长（SQL计算）
		&offlineSeconds, &offlineDays, &offlineHours, &offlineMinutes,
		// #361: 活动告警数（select 列末尾 aa.active_alarm_count）
		&activeAlarmCount,
	)
	if err != nil {
		return nil, fmt.Errorf("scan device with info: %w", err)
	}

	// Assign nullable devices fields
	if productClass != nil {
		d.ProductClass = *productClass
	}
	if manufacturer != nil {
		d.Manufacturer = *manufacturer
	}
	if modelName != nil {
		d.ModelName = *modelName
	}
	if firmwareVersion != nil {
		d.FirmwareVersion = *firmwareVersion
	}
	if ipAddr != nil {
		d.IPAddress = *ipAddr
	}
	if connReqURL != nil {
		d.ConnectionRequestURL = *connReqURL
	}
	if udpAddr != nil {
		d.UDPConnectionRequestAddress = *udpAddr
	}
	if siteName != nil {
		d.DeviceName = *siteName
	}
	if siteID != nil {
		d.SiteID = *siteID
	}
	if len(extData) > 0 {
		// ignore unmarshal error for list view
		_ = json.Unmarshal(extData, &d.ExtensionData)
	}
	if len(eventsData) > 0 {
		_ = json.Unmarshal(eventsData, &d.LastInformEvents)
	}

	// Assign device_info fields
	d.InfoDeviceName = diDeviceName
	d.InfoAddress = diAddress
	d.Remark = diRemark
	d.ProjectStatus = diProjectStatus
	d.Height = diHeight
	d.ECI = diECI
	d.PCI = diPCI
	d.CellID = diCellID
	d.FreqPoint = diFreqPoint
	d.Bandwidth = diBandwidth
	d.TransmitPower = diTransmitPower
	d.PLMN = diPLMN
	d.RFStatus = diRFStatus
	d.CellStatus = diCellStatus
	d.MMEStatus = diMMEStatus
	d.SyncStatus = diSyncStatus
	d.KPIStatus = diKPIStatus
	d.NumOfCells = diNumOfCells
	d.GPSStatus = diGPSStatus
	d.AlarmSeverity = diAlarmSeverity // #361: 现来自 alarms_active 聚合 CASE，非 di.alarm_severity 冗余列
	d.ActiveAlarmCount = activeAlarmCount
	d.LicenseStatus = diLicenseStatus
	d.MAC = diMAC
	d.HardwareVersion = diHWVersion
	d.FirstOnlineTime = diFirstOnline
	d.LastOnlineTime = diLastOnline
	d.LastOfflineTime = diLastOffline
	d.RunTime = diRunTime
	d.CumulativeOnlineDuration = diCumOnline // T-0173
	// Phase 2/3 扩展列
	d.TAC = diTAC
	d.Band = diBand
	d.ULEarfcn = diULEarfcn
	d.SubframeAssignment = diSubframeAssignment
	d.SpecialSubframe = diSpecialSubframe
	d.RootIndex = diRootIndex
	d.GPSSatellites = diGPSSatellites
	d.GPSHeight = diGPSHeight
	d.LockStatus = diLockStatus
	d.EnbID = diEnbID
	d.NetworkModel = diNetworkModel
	d.OnlineDuration = onlineDuration
	// 离线时长
	d.OfflineSeconds = offlineSeconds
	d.OfflineDays = offlineDays
	d.OfflineHours = offlineHours
	d.OfflineMinutes = offlineMinutes
	// T-0162: 派生老 Status 字段给读侧兼容（DeriveStatusFromLifecycle 用
	// commissioned+online=Active / commissioned+offline=Offline / 等映射）
	d.Status = DeriveStatusFromLifecycle(d.LifecycleState, d.IsOnline)
	// op_state 从 di.op_state 读取 —— InfoSyncer.CalcOpState 在参数同步时写入,
	// 语义同 cell_status：任一 cell active → "1";全部 inactive / 无 cell 数据 → "0"。
	// di 未 JOIN 上(设备还没 device_info 行)时忄底 "" —— 前端 helper 会展示 '-'。
	if diOpState != nil {
		d.OpState = *diOpState
	}

	return &d, nil
}
