package device

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/license"
)

// ===== 接口定义 =====

// DeviceFilter specifies criteria for listing devices.
type DeviceFilter struct {
	Carrier      *model.CarrierCode
	Technology   *model.Technology
	Technologies []model.Technology
	// DeviceType 是设备列表 Tab 维度。UPS 只按 Inform ProductClass 的 UPS 前缀判断；
	// 不把 UPS 塞进 technology/network_type，避免污染既有无线制式过滤。
	DeviceType string

	// DEPRECATED (T-0162): 用 LifecycleState / IsOnline 替代。保留过渡期供
	// 老 query 参数自动翻译；handler 收到 `?status=` 会派生到 LifecycleState +
	// IsOnline。新代码请直接用新字段，不要写 Filter.Status。
	Status *model.DeviceStatus

	// T-0162: 拆分原 Status 为两个正交字段
	LifecycleState []model.DeviceLifecycle // 多选过滤；空切片表示不过滤
	IsOnline       *bool                   // 三态：nil=不过滤；true=仅在线；false=仅离线

	OUI    *string
	SN     *string  // exact match on serial_number
	SNList []string // exact match on serial_number IN (...)（批量输入：按 SN 列表精确过滤）
	Search *string  // fuzzy search across serial_number/site_name/manufacturer/device_name/address

	// Group filters
	GroupID             *uuid.UUID                    // filter by specific device group
	GroupIDs            []uuid.UUID                   // filter by any of these device groups (OR semantics)
	VisibleGroups       []uuid.UUID                   // legacy data permission: restrict to these groups (nil = no restriction)
	VisibleDeviceGrants []model.DeviceVisibilityGrant // grant-based device data permission

	// Extended filters (device_info / devices additional fields)
	Manufacturer  *string     // devices.manufacturer exact match
	ProductID     *uuid.UUID  // devices.product_id exact match（T-0098 产品装配件软引用；下拉来自 /products）
	ProductIDs    []uuid.UUID // devices.product_id IN (...)，供跨多个产品名称检测
	ProductClass  *string     // devices.product_class exact match; CSV means match any value
	RFStatus      *string     // device_info.rf_status exact match
	CellStatus    *string     // device_info.cell_status exact match
	ProjectStatus *string     // device_info.project_status exact match
	GPSStatus     *string     // device_info.gps_status exact match
	AlarmSeverity *string     // device_info.alarm_severity exact match
	LicenseStatus *string     // device_info.license_status exact match
	OpState       *string     // "1" = activated (first_online_time NOT NULL), "0" = not activated
	ControlSource *string     // current OMC control source; currently only "geofence"
	ControlPhases []string    // current OMC control phases (OR semantics)

	// T-0162 新增 3 个 device list 筛选维度（之前前端下拉空、后端无字段）
	ModelName *string // devices.model_name exact match (字典 device_model)
	// SoftwareVersion 走 device_parameters 表的 TR-069 路径
	// 'Device.DeviceInfo.SoftwareVersion'（不在 device_info 表，是 TR-069 标准参数走
	// 参数树）。Repository 用 EXISTS 子查询过滤，避免 LEFT JOIN 引起的行膨胀。
	SoftwareVersion *string // device_parameters.parameter_value (字典 software_version)
	FirmwareVersion *string // devices.firmware_version exact match (字典 firmware_version)

	model.ListRequest
}

// RecycleBinFilter specifies criteria for listing soft-deleted devices.
type RecycleBinFilter struct {
	Search     *string // fuzzy search across serial_number/site_name/mac
	Carrier    *model.CarrierCode
	Technology *model.Technology
	GroupID    *uuid.UUID // filter by original device group
	DeletedBy  *string    // filter by who deleted the device

	model.ListRequest
}

// GeoDeviceFilter specifies criteria for listing devices with geo data.
// GroupIDs 是真实分组 ID；IncludeUngrouped 仅兼容历史无归属设备兜底。
type GeoDeviceFilter struct {
	GroupIDs         []string
	IncludeUngrouped bool
	Status           []model.DeviceStatus
	Keyword          string
	Bounds           *GeoBounds
	Page             int
	PageSize         int
	// VisibleGroups 是 #64 设备组数据权限的三态可见分组（nil=超管不过滤 / []=fail-closed 空集 /
	// [g...]=仅这些组下设备）。GIS 地图读链路按调用者可见分组 fail-closed 收口，过滤经
	// authz.ApplyDeviceVisibilityFilter 在 d.id 上做相关子查询（避免与已有 LEFT JOIN 行翻倍）。
	VisibleGroups       []uuid.UUID
	VisibleDeviceGrants []model.DeviceVisibilityGrant
	// UECountMax 过滤接入 UE 数：nil=不过滤，指向0=只返回 UE=0 的基站。
	UECountMax *int
}

// GeoStatsFilter specifies criteria for /devices/geo/stats.
// 与 GeoDeviceFilter 拆分是为了让 stats 也能接受 Status 过滤（handler 入参对齐 ListGeo），
// 同时与 list 接口共用 splitGeoGroupIDs 归一化（service 层填好 GroupIDs/IncludeUngrouped）。
type GeoStatsFilter struct {
	GroupIDs            []string
	IncludeUngrouped    bool
	Status              []model.DeviceStatus
	VisibleGroups       []uuid.UUID
	VisibleDeviceGrants []model.DeviceVisibilityGrant
}

// GeoBounds defines a geographic bounding box.
type GeoBounds struct {
	MinLng float64
	MaxLng float64
	MinLat float64
	MaxLat float64
}

// GeoDevice represents device data for map display.
type GeoDevice struct {
	ID           uuid.UUID          `json:"id"`
	SerialNumber string             `json:"sn"`
	Name         string             `json:"name"`
	Status       model.DeviceStatus `json:"status"`
	Latitude     float64            `json:"latitude"`
	Longitude    float64            `json:"longitude"`
	GroupID      *uuid.UUID         `json:"group_id,omitempty"`
	GroupName    string             `json:"group_name,omitempty"`
	Address      string             `json:"address,omitempty"`
	AlarmCount   int                `json:"alarm_count"`
	Type         string             `json:"type,omitempty"`
	// 搜索相关字段
	IPAddress  *string `json:"ip_address,omitempty"`  // devices.ip_address
	MAC        *string `json:"mac,omitempty"`         // device_info.mac
	PCI        *string `json:"pci,omitempty"`         // device_info.pci
	DeviceName *string `json:"device_name,omitempty"` // device_info.device_name
	// GIS 地图字段
	UECount                   int  `json:"ue_count"`                         // 当前接入 UE 数
	HighestAlarmSeverity      *int `json:"highest_alarm_severity,omitempty"` // 最高告警级别 1=Critical..4=Warning; nil=无告警
	HighestSeverityAlarmCount int  `json:"highest_severity_alarm_count"`     // 最高级别的告警数量
}

// GeoStats represents device statistics for map display.
type GeoStats struct {
	Total       int64                        `json:"total"`
	StatusCount map[model.DeviceStatus]int64 `json:"status_count"`
	AlarmCount  int64                        `json:"alarm_count"`
	Center      *GeoCenter                   `json:"center,omitempty"` // 平均经纬度中心点
	UEZeroCount int64                        `json:"ue_zero_count"`    // UE数为0的基站数
}

// GeoCenter represents the geographic center point of all devices.
type GeoCenter struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

// DeviceReader provides read-only access to devices.
type DeviceReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error)
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
	// GetDeletedBySerialNumber returns the most recently soft-deleted device with
	// the given serial number, or nil if no such row exists. Used by
	// RegisterFromInform to detect recycle-bin devices and auto-restore them
	// instead of creating a duplicate active row.
	GetDeletedBySerialNumber(ctx context.Context, sn string, carrier model.CarrierCode) (*model.Device, error)
	List(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error)
	CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
	// ListActiveByLastInform returns active devices ordered by last_inform_at ASC
	// using keyset (cursor) pagination to avoid the sliding-window problem caused
	// by OFFSET-based pagination when rows are mutated during iteration.
	// Pass cursorTime=nil for the first batch. Subsequent calls should pass the
	// last_inform_at of the last device returned, along with its ID as cursorID.
	ListActiveByLastInform(ctx context.Context, cursorTime *time.Time, cursorID *uuid.UUID, limit int) ([]model.Device, error)
	// ListGeo returns devices with geographic coordinates for map display.
	ListGeo(ctx context.Context, filter GeoDeviceFilter) ([]GeoDevice, int64, error)
	// GetGeoStats returns device statistics for map display.
	// filter.VisibleDeviceGrants 为 #64 设备数据权限的三态可见 grant（nil 超管 / [] fail-closed / [grant...] 限定）。
	GetGeoStats(ctx context.Context, filter GeoStatsFilter) (*GeoStats, error)
	// SearchDevices searches devices by keyword for map display.
	// visibleGrants 为 #64 设备数据权限的三态可见 grant（nil 超管 / [] fail-closed / [grant...] 限定）。
	SearchDevices(ctx context.Context, keyword string, limit int, visibleGrants []model.DeviceVisibilityGrant) ([]GeoDevice, error)
	// ListProductClasses returns distinct product_class values from devices,
	// merged with a set of mandatory types that must always appear.
	ListProductClasses(ctx context.Context) ([]string, error)
	// FindStaleDevices finds active devices that haven't sent Inform within the threshold.
	//
	// DEPRECATED (T-0173): 原 OfflineDetector 唯一调用方已废弃。新代码请用
	// *PgDeviceRepository.FindStaleDevicesAdaptive（自适应阈值 = max(2×inform_interval, minStaleSec))。
	// 保留接口仅为兼容全仓 11 个手写 mockDeviceRepo —— 等后续统一清理。
	FindStaleDevices(ctx context.Context, threshold time.Time, limit int) ([]*model.Device, error)
	// ListStaleForParamSync 找 last_param_sync_at IS NULL 或 < threshold 的 active 设备，
	// 供 PeriodicSyncer 按 interval 入队 Path B 同步（NULLS FIRST：从未同步过的设备优先）。
	ListStaleForParamSync(ctx context.Context, threshold time.Time, limit int) ([]*model.Device, error)
	// ListSerialsByIDs 按 ID 列表查 serial_number，供 service 层在批量删除/恢复
	// 操作前后拿到受影响的 SN 列表，统一调 cache.Delete 维护 cache 一致性。
	// 返回 map[id]sn，找不到的 ID 不在 map 中。
	ListSerialsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// DeviceWriter provides write operations for devices.
type DeviceWriter interface {
	Create(ctx context.Context, device *model.Device) error
	Update(ctx context.Context, device *model.Device) error
	Delete(ctx context.Context, id uuid.UUID) error
	// BatchDelete soft-deletes multiple devices and removes their group memberships
	// and device_info records within a transaction. Returns the number of deleted devices.
	BatchDelete(ctx context.Context, ids []uuid.UUID, deletedBy string) (int64, error)
	// UpdateStatus is the LEGACY status update (DEPRECATED T-0162).
	// 内部 shim 把 status 翻译为 lifecycle_state + is_online 双列。新代码请用
	// UpdateLifecycle / UpdateOnlineStatus 之一。
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error

	// T-0162: 解耦后的两个独立 update。
	UpdateLifecycle(ctx context.Context, id uuid.UUID, lifecycle model.DeviceLifecycle) error
	UpdateOnlineStatus(ctx context.Context, id uuid.UUID, isOnline bool) error

	UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error
	// UpdateLastParamSyncAt 回写 last_param_sync_at（HandleSyncResultPathB BatchUpsert
	// 成功后调；不区分触发源统一口径，回写口径详见 PRD F09 §2.6）。
	UpdateLastParamSyncAt(ctx context.Context, id uuid.UUID, at time.Time) error
	// UpdateLastParamSyncFailed 由 sync 失败订阅者在 Path B GPV task 失败时调用,
	// 写 last_param_sync_failed_at + last_param_sync_error 并清空 last_param_sync_at。
	UpdateLastParamSyncFailed(ctx context.Context, id uuid.UUID, failedAt time.Time, errMsg string) error
	// UpdateSiteName 仅更新 devices.site_name（设备主名称），供名称同步 use_lmt 使用。
	// 比 Update() 轻量：不需要完整设备对象，不清 cache（调用方按需清）。
	UpdateSiteName(ctx context.Context, id uuid.UUID, name string) error
	// RecordBoot atomically increments boot_count and sets last_boot_at for the device
	// identified by serial number. Invoked when the ACS receives a "1 BOOT" or
	// "M Reboot" Inform. Returns the updated boot_count.
	RecordBoot(ctx context.Context, sn string, at time.Time) (int, error)
	// RecycleBin operations
	// ListRecycleBin returns soft-deleted devices with filtering, including device_info fields.
	// T-2026-07-02: 返回 DeviceWithInfo 以支持完整的设备信息（MAC、GPS、项目状态等）。
	ListRecycleBin(ctx context.Context, filter RecycleBinFilter) (*model.ListResponse[DeviceWithInfo], error)
	// RestoreDevices restores soft-deleted devices (sets deleted_at to NULL).
	// #378: 可部分成功——返回恢复数 + 因 SN 冲突被跳过的明细，避免整批回滚 500。
	RestoreDevices(ctx context.Context, ids []uuid.UUID) (*RestoreResult, error)
	// PermanentDelete permanently removes devices from the database.
	PermanentDelete(ctx context.Context, ids []uuid.UUID) (int64, error)
}

type RecycleType string

const (
	RecycleTypeManual RecycleType = "manual"
	RecycleTypeAuto   RecycleType = "auto"
)

// RecycleMetadata separates the business account shown in the recycle bin
// from the process that actually executed the soft delete.
type RecycleMetadata struct {
	DeletedBy string
	Type      RecycleType
	Executor  string
}

// DeviceRepository defines the full interface for device persistence.
// It composes smaller interfaces for backward compatibility.
//
//go:generate go run go.uber.org/mock/mockgen -destination=mock_device_repository_test.go -package=device . DeviceRepository
type DeviceRepository interface {
	DeviceReader
	DeviceWriter
}

// ===== PostgreSQL 实现 =====

// allowedSortColumns prevents SQL injection in ORDER BY clauses.
// T-0162: 移除 "status" (列已 DROP)，加 "lifecycle_state" + "is_online"。
var allowedSortColumns = map[string]bool{
	"created_at":      true,
	"updated_at":      true,
	"serial_number":   true,
	"lifecycle_state": true,
	"is_online":       true,
	"carrier":         true,
	"technology":      true,
	"model":           true,
	"manufacturer":    true,
	"last_inform_at":  true,
}

// PgDeviceRepository implements DeviceRepository using PostgreSQL.
type PgDeviceRepository struct {
	pool *pgxpool.Pool
}

// NewPgDeviceRepository creates a new PostgreSQL device repository.
func NewPgDeviceRepository(pool *pgxpool.Pool) *PgDeviceRepository {
	return &PgDeviceRepository{pool: pool}
}

func (r *PgDeviceRepository) Create(ctx context.Context, device *model.Device) error {
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now

	// T-0162: 若调用方只设了老 Status 字段（P3 渐进迁移期间常见），从 Status
	// 派生 LifecycleState + IsOnline；新调用方设了新字段则直传
	normalizeDeviceForPersist(device)

	extData, err := json.Marshal(device.ExtensionData)
	if err != nil {
		return fmt.Errorf("marshal extension_data: %w", err)
	}
	eventsData, err := json.Marshal(device.LastInformEvents)
	if err != nil {
		return fmt.Errorf("marshal last_inform_events: %w", err)
	}

	// INET column requires nil for empty values, not empty string
	var ipAddr interface{}
	if device.IPAddress != "" {
		ipAddr = device.IPAddress
	}

	var udpAddr interface{}
	if device.UDPConnectionRequestAddress != "" {
		udpAddr = device.UDPConnectionRequestAddress
	}

	query, args, err := storage.Psql.Insert("devices").
		Columns("id", "serial_number", "oui", "product_class", "manufacturer", "model_name",
			"carrier", "technology",
			"lifecycle_state", "is_online", // T-0162: 替代 status
			"firmware_version", "ip_address",
			"connection_request_url", "nat_detected", "udp_connection_request_address",
			"last_inform_at", "last_inform_events",
			"last_boot_at", "boot_count",
			"inform_interval", "site_name", "site_id", "latitude", "longitude",
			"extension_data", "created_at", "updated_at").
		Values(device.ID, device.SerialNumber, device.OUI, device.ProductClass,
			device.Manufacturer, device.ModelName, device.Carrier, device.Technology,
			device.LifecycleState, device.IsOnline, // T-0162: 替代 device.Status
			device.FirmwareVersion, ipAddr,
			device.ConnectionRequestURL, device.NatDetected, udpAddr,
			device.LastInformAt, eventsData,
			device.LastBootAt, device.BootCount,
			device.InformInterval, device.DeviceName, device.SiteID,
			device.Latitude, device.Longitude, extData, device.CreatedAt, device.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}
	return nil
}

func (r *PgDeviceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	query, args, err := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		Where(sq.Eq{"d.id": id}).
		Where(notDeleted).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return r.scanDevice(ctx, query, args...)
}

func (r *PgDeviceRepository) GetCoordinates(ctx context.Context, id uuid.UUID) (*Location, error) {
	query, args, err := buildGetCoordinatesQuery(id)
	if err != nil {
		return nil, err
	}

	var latitude, longitude *float64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&latitude, &longitude); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get device coordinates: %w", err)
	}
	if latitude == nil || longitude == nil {
		return nil, nil
	}
	return &Location{Latitude: *latitude, Longitude: *longitude}, nil
}

func buildGetCoordinatesQuery(id uuid.UUID) (string, []interface{}, error) {
	query, args, err := storage.Psql.Select("latitude", "longitude").
		From("devices d").
		Where(sq.Eq{"id": id}).
		Where(notDeleted).
		Limit(1).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build get device coordinates query: %w", err)
	}
	return query, args, nil
}

func (r *PgDeviceRepository) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	query, args, err := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		Where(sq.Eq{"d.serial_number": sn}).
		Where(notDeleted).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return r.scanDevice(ctx, query, args...)
}

// GetDeletedBySerialNumber returns the most recently soft-deleted device with
// the given serial number and carrier. Returns nil when no soft-deleted row exists.
// Called by RegisterFromInform to auto-restore recycle-bin devices instead of
// creating a duplicate active row.
func (r *PgDeviceRepository) GetDeletedBySerialNumber(ctx context.Context, sn string, carrier model.CarrierCode) (*model.Device, error) {
	query, args, err := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		Where(sq.Eq{"d.serial_number": sn}).
		Where(sq.Eq{"d.carrier": carrier}).
		Where(sq.NotEq{"d.deleted_at": nil}).
		OrderBy("d.deleted_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return r.scanDevice(ctx, query, args...)
}

func (r *PgDeviceRepository) Update(ctx context.Context, device *model.Device) error {
	extData, err := json.Marshal(device.ExtensionData)
	if err != nil {
		return fmt.Errorf("marshal extension_data: %w", err)
	}
	eventsData, err := json.Marshal(device.LastInformEvents)
	if err != nil {
		return fmt.Errorf("marshal last_inform_events: %w", err)
	}

	// INET column requires nil for empty values, not empty string
	var ipAddr interface{}
	if device.IPAddress != "" {
		ipAddr = device.IPAddress
	}

	var udpAddr interface{}
	if device.UDPConnectionRequestAddress != "" {
		udpAddr = device.UDPConnectionRequestAddress
	}

	// T-0162: 若调用方只设了老 Status 字段，从 Status 派生 LifecycleState + IsOnline
	normalizeDeviceForPersist(device)

	query, args, err := storage.Psql.Update("devices").
		Set("oui", device.OUI).
		Set("product_class", device.ProductClass).
		Set("manufacturer", device.Manufacturer).
		Set("model_name", device.ModelName).
		Set("technology", device.Technology).
		Set("lifecycle_state", device.LifecycleState). // T-0162: 替代 status
		Set("is_online", device.IsOnline).             // T-0162: 新增
		Set("firmware_version", device.FirmwareVersion).
		Set("ip_address", ipAddr).
		Set("connection_request_url", device.ConnectionRequestURL).
		Set("nat_detected", device.NatDetected).
		Set("udp_connection_request_address", udpAddr).
		Set("last_inform_at", device.LastInformAt).
		Set("last_inform_events", eventsData).
		Set("inform_interval", device.InformInterval).
		Set("site_name", device.DeviceName).
		Set("latitude", device.Latitude).
		Set("longitude", device.Longitude).
		Set("location_source_mode", device.LocationSourceMode).
		Set("extension_data", extData).
		Where(sq.Eq{"id": device.ID}).
		Where(sq.Eq{"deleted_at": nil}). // 防软删 device 被 inform 静默复活；命中时 RowsAffected=0 → 上层 ErrNotFound → 清 cache 走 auto-register
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	ct, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}
	// device row vanished between cache hit and update — surface as ErrNotFound
	// so the caller can fall back to register / cache invalidation.
	if ct.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// UpdateSiteName 仅更新 devices.site_name，比 Update() 轻量，不处理全量字段。
func (r *PgDeviceRepository) UpdateSiteName(ctx context.Context, id uuid.UUID, name string) error {
	query, args, err := storage.Psql.Update("devices").
		Set("site_name", name).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"deleted_at": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update site_name query: %w", err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update site_name: %w", err)
	}
	return nil
}

func (r *PgDeviceRepository) UpdateCoordinates(ctx context.Context, id uuid.UUID, latitude, longitude float64) error {
	query, args, err := storage.Psql.Update("devices").
		Set("latitude", latitude).
		Set("longitude", longitude).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"deleted_at": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update coordinates query: %w", err)
	}

	ct, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device coordinates: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	return nil
}

func (r *PgDeviceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, _ := storage.Psql.Update("devices").
		Set("deleted_at", time.Now()).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"deleted_at": nil}).
		ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("soft delete device: %w", err)
	}
	return nil
}

// BatchDelete soft-deletes multiple devices within a single transaction.
// It intentionally keeps device_group_members and device_info rows so recycle
// bin can still display historical group and extended info.
// Returns the number of devices actually soft-deleted.
func (r *PgDeviceRepository) BatchDelete(ctx context.Context, ids []uuid.UUID, deletedBy string) (int64, error) {
	return r.BatchDeleteWithMetadata(ctx, ids, RecycleMetadata{
		DeletedBy: deletedBy,
		Type:      RecycleTypeManual,
		Executor:  deletedBy,
	})
}

// BatchDeleteWithMetadata soft-deletes devices while preserving how the move
// was initiated and which actor/process executed it.
func (r *PgDeviceRepository) BatchDeleteWithMetadata(ctx context.Context, ids []uuid.UUID, metadata RecycleMetadata) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// T-2026-07-02: 不删除 device_info 和 device_group_members，保留历史信息供回收站显示。
	// 删除时仅软删除 devices 表，保持参照完整性和审计日志。

	// Soft-delete devices with metadata
	now := time.Now()
	tag, err := tx.Exec(ctx,
		`UPDATE devices
		 SET deleted_at = $1, deleted_by = $2, recycle_type = $3, recycle_executor = $4
		 WHERE id = ANY($5) AND deleted_at IS NULL`,
		now, metadata.DeletedBy, metadata.Type, metadata.Executor, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("soft delete devices: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit batch delete: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (r *PgDeviceRepository) List(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error) {
	builder := storage.Psql.Select(deviceColumns()...).From("devices d").Where(notDeleted)
	countBuilder := storage.Psql.Select("COUNT(*)").From("devices d").Where(notDeleted)

	// GroupID filter - requires JOIN with device_group_members
	if filter.GroupID != nil {
		builder = builder.
			Join("device_group_members dgm ON d.id = dgm.device_id").
			Where(sq.Eq{"dgm.group_id": *filter.GroupID})
		countBuilder = countBuilder.
			Join("device_group_members dgm ON d.id = dgm.device_id").
			Where(sq.Eq{"dgm.group_id": *filter.GroupID})
	}

	// VisibleGroups filter - data permission restriction
	// #64 fail-open 收口：沿用 authz.ApplyDeviceVisibilityFilter 的三态 fail-closed 语义。
	//   nil         → 超管，不过滤
	//   []（非 nil）  → 非超管且无任何可见分组，短路空集
	//   [g1, ...]   → 限定到这些分组；包含 DefaultLevel2GroupID 时自动放行未分组设备
	if filter.VisibleGroups != nil {
		builder = authz.ApplyDeviceVisibilityFilter(builder, "d.id", filter.VisibleGroups)
		countBuilder = authz.ApplyDeviceVisibilityFilter(countBuilder, "d.id", filter.VisibleGroups)
	}

	if filter.Carrier != nil {
		builder = builder.Where(sq.Eq{"d.carrier": *filter.Carrier})
		countBuilder = countBuilder.Where(sq.Eq{"d.carrier": *filter.Carrier})
	}
	if filter.Technology != nil {
		builder = builder.Where(sq.Eq{"d.technology": *filter.Technology})
		countBuilder = countBuilder.Where(sq.Eq{"d.technology": *filter.Technology})
	}
	builder = applyDeviceListDeviceTypeFilter(builder, filter.DeviceType)
	countBuilder = applyDeviceListDeviceTypeFilter(countBuilder, filter.DeviceType)
	// T-0162: filter.Status 老字段过渡兼容——翻译为 lifecycle_state + is_online
	if filter.Status != nil {
		lifecycle, isOnline := DeriveLifecycleFromStatus(*filter.Status)
		builder = builder.Where(sq.Eq{"d.lifecycle_state": lifecycle})
		countBuilder = countBuilder.Where(sq.Eq{"d.lifecycle_state": lifecycle})
		// 只有 Active/Offline 同时蕴含 is_online；其他 status 不限制 is_online
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
	if filter.ModelName != nil && *filter.ModelName != "" {
		builder = builder.Where(sq.Eq{"d.model_name": *filter.ModelName})
		countBuilder = countBuilder.Where(sq.Eq{"d.model_name": *filter.ModelName})
	}
	if filter.FirmwareVersion != nil && *filter.FirmwareVersion != "" {
		builder = builder.Where(sq.Eq{"d.firmware_version": *filter.FirmwareVersion})
		countBuilder = countBuilder.Where(sq.Eq{"d.firmware_version": *filter.FirmwareVersion})
	}
	// SoftwareVersion 在 device_info 表，本 Repository (devices-only) 不处理；
	// 由 device_info_pg_repository 的 ListDevicesWithInfo 接力。
	if filter.OUI != nil {
		builder = builder.Where(sq.Eq{"d.oui": *filter.OUI})
		countBuilder = countBuilder.Where(sq.Eq{"d.oui": *filter.OUI})
	}
	if filter.SN != nil && *filter.SN != "" {
		builder = builder.Where(sq.Eq{"d.serial_number": *filter.SN})
		countBuilder = countBuilder.Where(sq.Eq{"d.serial_number": *filter.SN})
	}
	if len(filter.SNList) > 0 {
		builder = builder.Where(sq.Eq{"d.serial_number": filter.SNList})
		countBuilder = countBuilder.Where(sq.Eq{"d.serial_number": filter.SNList})
	}
	if filter.Search != nil && *filter.Search != "" {
		like := "%" + *filter.Search + "%"
		cond := sq.Or{sq.ILike{"d.serial_number": like}, sq.ILike{"d.site_name": like}}
		builder = builder.Where(cond)
		countBuilder = countBuilder.Where(cond)
	}
	if filter.ProductID != nil {
		builder = builder.Where(sq.Eq{"d.product_id": *filter.ProductID})
		countBuilder = countBuilder.Where(sq.Eq{"d.product_id": *filter.ProductID})
	}
	if len(filter.ProductIDs) > 0 {
		builder = builder.Where(sq.Eq{"d.product_id": filter.ProductIDs})
		countBuilder = countBuilder.Where(sq.Eq{"d.product_id": filter.ProductIDs})
	}
	if filter.ProductClass != nil && *filter.ProductClass != "" {
		productClasses := SplitCSV(*filter.ProductClass)
		builder = builder.Where(sq.Eq{"d.product_class": productClasses})
		countBuilder = countBuilder.Where(sq.Eq{"d.product_class": productClasses})
	}

	// Count total
	countQuery, countArgs, _ := countBuilder.ToSql()
	var total int64
	r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)

	// Apply pagination
	sortBy := "created_at"
	if filter.SortBy != "" && allowedSortColumns[filter.SortBy] {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "desc" || filter.SortDir == "" {
		sortDir = "DESC"
	} else if filter.SortDir == "asc" {
		sortDir = "ASC"
	}
	builder = builder.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *d)
	}

	if devices == nil {
		devices = []model.Device{}
	}

	return model.NewListResponse(devices, total, filter.Page, filter.PageSize), nil
}

// UpdateStatus is the LEGACY status update method.
//
// DEPRECATED (T-0162): 老 DeviceStatus 类型混淆生命周期+在线。本方法保留过渡
// 期供 P3 调用方逐步迁到 UpdateLifecycle / UpdateOnlineStatus。当前实现是
// shim：把 status 翻译为 (lifecycle, is_online) 后更新两列。
//
// 推荐：HeartbeatMonitor / OfflineDetector 调 UpdateOnlineStatus；管理面
// maintenance / decommission 操作调 UpdateLifecycle。
func (r *PgDeviceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	lifecycle, isOnline := DeriveLifecycleFromStatus(status)
	query, args, _ := storage.Psql.Update("devices").
		Set("lifecycle_state", lifecycle).
		Set("is_online", isOnline).
		Where(sq.Eq{"id": id}).
		ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device status (shim): %w", err)
	}
	return nil
}

// UpdateLifecycle updates only the lifecycle_state column. is_online untouched.
//
// 使用场景：维护操作 (TransitionLifecycle to Maintenance) / 退役 / Provisioning
// engine 推进到 Commissioned。
func (r *PgDeviceRepository) UpdateLifecycle(ctx context.Context, id uuid.UUID, lifecycle model.DeviceLifecycle) error {
	query, args, _ := storage.Psql.Update("devices").
		Set("lifecycle_state", lifecycle).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"deleted_at": nil}).
		ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device lifecycle: %w", err)
	}
	return nil
}

// UpdateOnlineStatus updates only the is_online column. lifecycle_state untouched.
//
// 使用场景：HeartbeatMonitor 标记设备离线 (false)；OfflineDetector 同；
// ACS Inform 接收时标记在线 (true) + 更新 last_inform_at。lifecycle 完全不动
// （commissioned 设备 is_online=false 是合法状态：已入网但当前掉线）。
func (r *PgDeviceRepository) UpdateOnlineStatus(ctx context.Context, id uuid.UUID, isOnline bool) error {
	query, args, _ := storage.Psql.Update("devices").
		Set("is_online", isOnline).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"deleted_at": nil}).
		ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device is_online: %w", err)
	}
	return nil
}

// OfflineExcessByTypeCapacity 把各容量组超出容量上限的在线设备（按 created_at
// 晚接入的）置为离线，用于 license 降容清理（issue #316，在线口径：离线不占容量，
// 故腾容量无需删除设备）。typeCapacity key 为 ne_type（大小写不敏感匹配
// alarm_ne_type），value 为容量上限。未出现在 typeCapacity 中但有在线设备的类型
// 视为未授权（容量 0），其在线设备全部置离线。返回被置离线设备的 serial_number
// 列表（caller 据此清 Redis 设备缓存，避免缓存 stale 的 is_online=true 让被踢
// 设备下次 Inform 跳过容量校验又上线）。
//
// 容量分组（issue #318）：GSM 与 eNB 共用容量。两条 SQL 内的 CASE 把 ne_type
// 归一到容量组 key（GSM→ENB，与 license 包 neTypeCapacityGroups 保持一致），
// 使超容排名跨组内类型混合按 created_at，且 eNB 已配置时 GSM 不落入"未配置
// 类型"分支。修改分组规则时必须同步 license/capacity_group.go。
//
// 满足 ExcessOffliner 接口；两条静态 SQL 全参数化，无字符串拼接。
func (r *PgDeviceRepository) OfflineExcessByTypeCapacity(ctx context.Context, typeCapacity map[string]int) ([]string, error) {
	offlined := make([]string, 0)
	if len(typeCapacity) == 0 {
		return offlined, nil
	}
	configuredUpper := make([]string, 0, len(typeCapacity))
	// 静态 SQL：踢某容量组 created_at 最晚的超出部分（rn > capacity）。
	// $1=容量组 key（如 eNB），$2=capacity。
	const excessByType = `WITH ranked AS (
		SELECT d.id, ROW_NUMBER() OVER (ORDER BY d.created_at) AS rn
		FROM devices d JOIN products p ON d.product_id = p.id
		WHERE CASE WHEN UPPER(p.alarm_ne_type) = 'GSM' THEN 'ENB' ELSE UPPER(p.alarm_ne_type) END = UPPER($1)
		  AND d.is_online = true AND d.deleted_at IS NULL
	)
	UPDATE devices SET is_online = false, updated_at = now()
	WHERE id IN (SELECT id FROM ranked WHERE rn > $2)
	RETURNING serial_number`
	exemptUpper := license.CapacityExemptTypes()
	for nt, cap := range typeCapacity {
		if license.IsCapacityExempt(nt) {
			// 容量豁免类型（license/capacity_group.go）：不受容量控制，降容时
			// 既不按配额踢线，也不落入下方"未配置类型"分支。
			continue
		}
		configuredUpper = append(configuredUpper, strings.ToUpper(nt))
		if cap < 0 {
			cap = 0
		}
		rows, err := r.pool.Query(ctx, excessByType, nt, cap)
		if err != nil {
			return offlined, fmt.Errorf("offline excess devices for type %s: %w", nt, err)
		}
		for rows.Next() {
			var sn string
			if err := rows.Scan(&sn); err != nil {
				rows.Close()
				return offlined, fmt.Errorf("scan offlined serial_number: %w", err)
			}
			offlined = append(offlined, sn)
		}
		rows.Close()
	}
	// 未配置类型（有在线设备但新 license 未授权）：全部置离线。$1=text[] 已配置
	// 容量组 key（UPPER；CASE 归一后 GSM 设备归入 ENB 组）；$2=text[] 容量豁免
	// 类型（license.CapacityExemptTypes，如 IMSCORE）永不落入"未配置类型"分支。
	const unconfigured = `UPDATE devices SET is_online = false, updated_at = now()
		WHERE id IN (
			SELECT d.id FROM devices d LEFT JOIN products p ON d.product_id = p.id
			WHERE d.is_online = true AND d.deleted_at IS NULL
			  AND CASE WHEN UPPER(COALESCE(p.alarm_ne_type,'')) = 'GSM' THEN 'ENB' ELSE UPPER(COALESCE(p.alarm_ne_type,'')) END <> ALL($1::text[])
			  AND UPPER(COALESCE(p.alarm_ne_type,'')) <> ALL($2::text[])
		)
		RETURNING serial_number`
	rows, err := r.pool.Query(ctx, unconfigured, configuredUpper, exemptUpper)
	if err != nil {
		return offlined, fmt.Errorf("offline unconfigured types: %w", err)
	}
	for rows.Next() {
		var sn string
		if err := rows.Scan(&sn); err != nil {
			rows.Close()
			return offlined, fmt.Errorf("scan offlined serial_number (unconfigured): %w", err)
		}
		offlined = append(offlined, sn)
	}
	rows.Close()
	return offlined, nil
}

func (r *PgDeviceRepository) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	eventsData, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("marshal last_inform_events: %w", err)
	}
	query, args, err := storage.Psql.Update("devices").
		Set("last_inform_at", at).
		Set("last_inform_events", eventsData).
		Where(sq.Eq{"serial_number": sn}).
		ToSql()
	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update last inform: %w", err)
	}
	return nil
}

// RecordBoot atomically increments boot_count and stamps last_boot_at for the
// device identified by serial number. Used when the ACS receives a 1 BOOT /
// M Reboot Inform. The atomic UPDATE prevents concurrent Informs from racing
// on the counter. Returns the new boot_count, or 0 with no error if the
// device row does not exist (caller decides whether to auto-register).
func (r *PgDeviceRepository) RecordBoot(ctx context.Context, sn string, at time.Time) (int, error) {
	const q = `UPDATE devices
	   SET last_boot_at = $1,
	       boot_count = COALESCE(boot_count, 0) + 1
	   WHERE serial_number = $2 AND deleted_at IS NULL
	   RETURNING boot_count`
	var bootCount int
	err := r.pool.QueryRow(ctx, q, at, sn).Scan(&bootCount)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("record boot: %w", err)
	}
	return bootCount, nil
}

// CountByStatus 兼容老格式返回 map[DeviceStatus]int64。
//
// T-0162: 内部走 lifecycle_state + is_online 双维度 GROUP BY，再聚合派生回
// 老 DeviceStatus 取值给老调用方（dashboard/service.go 等）。新代码请用
// CountByLifecycle + CountOnline 拿两维独立计数。
func (r *PgDeviceRepository) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	builder := storage.Psql.Select("d.lifecycle_state", "d.is_online", "COUNT(*)").
		From("devices d").Where(notDeleted).
		GroupBy("d.lifecycle_state", "d.is_online")
	if carrier != nil {
		builder = builder.Where(sq.Eq{"d.carrier": *carrier})
	}
	query, args, _ := builder.ToSql()

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query device count by status: %w", err)
	}
	defer rows.Close()

	result := make(map[model.DeviceStatus]int64)
	for rows.Next() {
		var lifecycle model.DeviceLifecycle
		var isOnline bool
		var count int64
		if err := rows.Scan(&lifecycle, &isOnline, &count); err != nil {
			return nil, fmt.Errorf("scan device count: %w", err)
		}
		status := DeriveStatusFromLifecycle(lifecycle, isOnline)
		result[status] += count // 不同 (lifecycle, online) 可能映射到同一 status，累加
	}
	return result, nil
}

func (r *PgDeviceRepository) ListActiveByLastInform(ctx context.Context, cursorTime *time.Time, cursorID *uuid.UUID, limit int) ([]model.Device, error) {
	// T-0162: "active" 语义 = lifecycle_state='commissioned' AND is_online=TRUE
	builder := storage.Psql.Select(deviceColumns()...).From("devices d").
		Where(sq.Eq{"d.lifecycle_state": model.LifecycleCommissioned}).
		Where(sq.Eq{"d.is_online": true}).
		Where(notDeleted).
		OrderBy("d.last_inform_at ASC NULLS FIRST", "d.id ASC").
		Limit(uint64(limit))

	if cursorTime != nil && cursorID != nil {
		// Keyset condition: (last_inform_at, id) > (cursorTime, cursorID)
		// Handles NULL last_inform_at: NULLs sort first, so after we pass them
		// we only need the non-NULL condition.
		builder = builder.Where(
			"(d.last_inform_at > ? OR (d.last_inform_at = ? AND d.id > ?))",
			*cursorTime, *cursorTime, *cursorID,
		)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list active by last inform query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active by last inform: %w", err)
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *d)
	}
	return devices, nil
}

func (r *PgDeviceRepository) scanDevice(ctx context.Context, query string, args ...interface{}) (*model.Device, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	d, err := scanDeviceFromRow(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func deviceColumns() []string {
	// T-0098 P5-02：移除 data_model_id（列已 DROP）。
	// T-0124：新增 last_param_sync_at（HandleSyncResultPathB 回写；PeriodicSyncer 据此找过期设备）。
	// T-0162: status 列已 DROP，替换为 lifecycle_state + is_online 两列。
	return []string{
		"d.id", "d.serial_number", "d.oui", "d.product_class", "d.manufacturer", "d.model_name",
		"d.carrier", "d.technology",
		"d.product_id", "d.param_model_id",
		"d.lifecycle_state", "d.is_online", // T-0162: 替代 d.status
		"d.firmware_version",
		"host(d.ip_address) as ip_address", "d.connection_request_url",
		"d.nat_detected", "d.udp_connection_request_address",
		"d.last_inform_at", "d.last_inform_events",
		"d.last_boot_at", "d.boot_count",
		"d.inform_interval", "d.site_name", "d.site_id", "d.latitude", "d.longitude",
		"d.location_source_mode",
		"d.extension_data", "d.created_at", "d.updated_at", "d.deleted_at", "d.deleted_by",
		"d.recycle_type", "d.recycle_executor",
		"d.last_param_sync_at",
		"d.last_param_sync_failed_at", "d.last_param_sync_error", // migration 000142
		"d.last_offline_reason", // T-0173: 离线原因诊断列（migration 000184）
	}
}

// notDeleted is the standard soft-delete filter applied to read queries that
// alias devices as "d" (FROM devices d). UPDATE/DELETE on devices is single-table
// without an alias and must use sq.Eq{"deleted_at": nil} inline.
var notDeleted = sq.Eq{"d.deleted_at": nil}

func scanDeviceFromRow(row pgx.Row) (*model.Device, error) {
	var d model.Device
	var extData, eventsData []byte
	var ipAddr, udpAddr *string
	// nullable string columns from devices table
	var productClass, manufacturer, modelName *string
	var firmwareVersion, connReqURL, siteName, siteID *string
	var deletedBy, recycleType, recycleExecutor *string

	err := row.Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &productClass, &manufacturer, &modelName,
		&d.Carrier, &d.Technology,
		&d.ProductID, &d.ParamModelID,
		&d.LifecycleState, &d.IsOnline, // T-0162: 替代 &d.Status
		&firmwareVersion,
		&ipAddr, &connReqURL,
		&d.NatDetected, &udpAddr,
		&d.LastInformAt, &eventsData,
		&d.LastBootAt, &d.BootCount,
		&d.InformInterval, &siteName, &siteID, &d.Latitude, &d.Longitude,
		&d.LocationSourceMode,
		&extData, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt, &deletedBy,
		&recycleType, &recycleExecutor,
		&d.LastParamSyncAt,
		&d.LastParamSyncFailedAt, &d.LastParamSyncError, // migration 000142
		&d.LastOfflineReason, // T-0173: 离线原因（migration 000184)
	)
	if err != nil {
		return nil, err
	}

	if deletedBy != nil {
		d.DeletedBy = *deletedBy
	}
	if recycleType != nil {
		d.RecycleType = *recycleType
	}
	if recycleExecutor != nil {
		d.RecycleExecutor = *recycleExecutor
	}

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
		if err := json.Unmarshal(extData, &d.ExtensionData); err != nil {
			return nil, fmt.Errorf("unmarshal extension_data: %w", err)
		}
	}
	if len(eventsData) > 0 {
		if err := json.Unmarshal(eventsData, &d.LastInformEvents); err != nil {
			return nil, fmt.Errorf("unmarshal last_inform_events: %w", err)
		}
	}
	// T-0162: 派生 Status + OpState 给老调用方
	populateDeviceCompat(&d)
	return &d, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanDeviceRow(rows pgx.Rows) (*model.Device, error) {
	var d model.Device
	var extData, eventsData []byte
	var ipAddr, udpAddr *string
	// nullable string columns from devices table
	var productClass, manufacturer, modelName *string
	var firmwareVersion, connReqURL, siteName, siteID, deletedBy, recycleType, recycleExecutor *string

	err := rows.Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &productClass, &manufacturer, &modelName,
		&d.Carrier, &d.Technology,
		&d.ProductID, &d.ParamModelID,
		&d.LifecycleState, &d.IsOnline, // T-0162: 替代 &d.Status
		&firmwareVersion,
		&ipAddr, &connReqURL,
		&d.NatDetected, &udpAddr,
		&d.LastInformAt, &eventsData,
		&d.LastBootAt, &d.BootCount,
		&d.InformInterval, &siteName, &siteID, &d.Latitude, &d.Longitude,
		&d.LocationSourceMode,
		&extData, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt, &deletedBy,
		&recycleType, &recycleExecutor,
		&d.LastParamSyncAt,
		&d.LastParamSyncFailedAt, &d.LastParamSyncError, // migration 000142
		&d.LastOfflineReason, // T-0173: 离线原因（migration 000184)
	)

	if deletedBy != nil {
		d.DeletedBy = *deletedBy
	}
	if recycleType != nil {
		d.RecycleType = *recycleType
	}
	if recycleExecutor != nil {
		d.RecycleExecutor = *recycleExecutor
	}
	if err != nil {
		return nil, err
	}

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
		if err := json.Unmarshal(extData, &d.ExtensionData); err != nil {
			return nil, fmt.Errorf("unmarshal extension_data: %w", err)
		}
	}
	if len(eventsData) > 0 {
		if err := json.Unmarshal(eventsData, &d.LastInformEvents); err != nil {
			return nil, fmt.Errorf("unmarshal last_inform_events: %w", err)
		}
	}
	// T-0162: 派生 Status + OpState 给老调用方
	populateDeviceCompat(&d)
	return &d, nil
}

// scanGeoDeviceRow 扫描一行 geo 查询结果（ListGeo / SearchDevices 共用，列顺序
// 与两处 SELECT 严格一致）。
//
// #117: devices.latitude / longitude 列可空（double precision 无 NOT NULL），
// SearchDevices 不带 IS NOT NULL 过滤，命中 NULL 坐标设备时直接扫进
// float64 会报 "cannot scan NULL into *float64" 导致整个搜索 500。
// 统一扫进 *float64 再判空赋值：NULL 坐标回零值，响应字段类型保持不变。
func scanGeoDeviceRow(row scannable) (GeoDevice, error) {
	var d GeoDevice
	var lifecycle model.DeviceLifecycle
	var isOnline bool
	var latitude, longitude *float64 // #117: 坐标列可空
	var groupID *uuid.UUID
	var groupName, address, deviceType *string
	var alarmCount int
	var ipAddress, mac, pci, deviceName string // COALESCE 保证非 NULL
	var ueCount int                            // COALESCE 保证非 NULL
	var highestAlarmSeverity *int              // NULL = 无活跃告警
	var highestSeverityAlarmCount int          // 最高级别的告警数量

	err := row.Scan(
		&d.ID, &d.SerialNumber, &d.Name,
		&lifecycle, &isOnline,
		&latitude, &longitude, &groupID, &groupName,
		&address, &alarmCount, &deviceType,
		&ipAddress, &mac, &pci, &deviceName,
		&ueCount,
		&highestAlarmSeverity,
		&highestSeverityAlarmCount,
	)
	if err != nil {
		return GeoDevice{}, err
	}
	// T-0162: 派生 Status 给老消费方
	d.Status = DeriveStatusFromLifecycle(lifecycle, isOnline)

	if latitude != nil {
		d.Latitude = *latitude
	}
	if longitude != nil {
		d.Longitude = *longitude
	}
	d.GroupID = groupID
	if groupName != nil {
		d.GroupName = *groupName
	}
	if address != nil {
		d.Address = *address
	}
	if deviceType != nil {
		d.Type = *deviceType
	}
	d.AlarmCount = alarmCount

	// 搜索 4 个字段：空字符串转为 nil
	if ipAddress != "" {
		d.IPAddress = &ipAddress
	}
	if mac != "" {
		d.MAC = &mac
	}
	if pci != "" {
		d.PCI = &pci
	}
	if deviceName != "" {
		d.DeviceName = &deviceName
	}
	d.UECount = ueCount
	d.HighestAlarmSeverity = highestAlarmSeverity
	d.HighestSeverityAlarmCount = highestSeverityAlarmCount
	return d, nil
}

// applyGeoGroupFilter 给 GIS Geo 查询（list/stats/center）拼上设备组过滤。
// includeUngrouped 仅用于兼容历史无归属设备；默认组本身作为真实分组传入 realIDs。
func applyGeoGroupFilter(builder sq.SelectBuilder, realIDs []string, includeUngrouped bool) sq.SelectBuilder {
	switch {
	case len(realIDs) > 0 && includeUngrouped:
		return builder.Where(sq.Or{
			sq.Eq{"dg.id": realIDs},
			sq.Expr(ungroupedDevicesWhere),
		})
	case len(realIDs) > 0:
		return builder.Where(sq.Eq{"dg.id": realIDs})
	case includeUngrouped:
		return builder.Where(sq.Expr(ungroupedDevicesWhere))
	default:
		return builder
	}
}

// applyGeoStatusFilter 给 GIS Geo 查询拼上 status OR 条件（lifecycle_state + is_online 翻译）。
// ListGeo 与 GetGeoStats 共享，避免两处表达式拼装不一致。
func applyGeoStatusFilter(builder sq.SelectBuilder, statuses []model.DeviceStatus) sq.SelectBuilder {
	if len(statuses) == 0 {
		return builder
	}
	var orClauses sq.Or
	for _, s := range statuses {
		lifecycle, isOnline := DeriveLifecycleFromStatus(s)
		clause := sq.Eq{"d.lifecycle_state": lifecycle}
		if s == model.DeviceActive || s == model.DeviceOffline {
			orClauses = append(orClauses, sq.And{clause, sq.Eq{"d.is_online": isOnline}})
		} else {
			orClauses = append(orClauses, clause)
		}
	}
	return builder.Where(orClauses)
}

// applyGeoBoundsFilter 将地图视口范围同时应用到列表与计数查询，保证分页完整性
// 元数据描述的是当前视口内的设备集合，而不是未过滤的全局设备集合。
func applyGeoBoundsFilter(builder sq.SelectBuilder, bounds *GeoBounds) sq.SelectBuilder {
	if bounds == nil {
		return builder
	}

	return builder.
		Where(sq.GtOrEq{"d.longitude": bounds.MinLng}).
		Where(sq.LtOrEq{"d.longitude": bounds.MaxLng}).
		Where(sq.GtOrEq{"d.latitude": bounds.MinLat}).
		Where(sq.LtOrEq{"d.latitude": bounds.MaxLat})
}

// ListGeo returns devices with geographic coordinates for map display.
func (r *PgDeviceRepository) ListGeo(ctx context.Context, filter GeoDeviceFilter) ([]GeoDevice, int64, error) {
	// T-0162: SELECT 改用 lifecycle_state + is_online，scan 后派生 Status 给老
	// 调用方（前端 geo 接口仍按 onlineActive/onlineInactive/offline 三档使用）
	builder := storage.Psql.Select(
		"d.id", "d.serial_number", "d.serial_number as name",
		"d.lifecycle_state", "d.is_online",
		"d.latitude", "d.longitude", "dg.id as group_id", "dg.name as group_name",
		"d.site_name as address", "0 as alarm_count", "d.model_name as type",
		"COALESCE(host(d.ip_address), '')", "COALESCE(di.mac, '')", "COALESCE(di.pci, '')", "COALESCE(di.device_name, '')",
		"COALESCE(di.ue_count, 0)",
		"NULL::int as highest_alarm_severity",
		"0 as highest_severity_alarm_count",
	).From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		LeftJoin("device_info di ON d.id = di.device_id AND COALESCE(d.product_class, '') NOT LIKE 'UPS%'").
		Where(sq.NotEq{"d.latitude": nil}).
		Where(sq.NotEq{"d.longitude": nil})

	// Apply filters
	builder = applyGeoGroupFilter(builder, filter.GroupIDs, filter.IncludeUngrouped)
	// T-0162: filter.Status 翻译到 lifecycle + is_online（与 GetGeoStats 共享 applyGeoStatusFilter）。
	builder = applyGeoStatusFilter(builder, filter.Status)
	builder = applyGeoBoundsFilter(builder, filter.Bounds)
	if filter.Keyword != "" {
		// GIS 地图搜索字段（6 个）：SN / 名称 / IP / MAC / PCI / 设备名称
		// 注意：d.ip_address 是 INET 类型，需要用 host() 转为 TEXT 后才能 ILIKE
		searchFields := []string{
			"d.serial_number",
			"d.site_name",
			"host(d.ip_address)",
			"di.mac",
			"di.pci",
			"di.device_name",
		}
		if cond := BuildSearchOR(filter.Keyword, searchFields); cond != nil {
			builder = builder.Where(cond)
		}
	}
	// #64 设备组数据权限：按 d.id 相关子查询三态 fail-closed 收口（nil 超管不过滤 /
	// [] WHERE FALSE / [grant...] 限定到可见分组+制式下设备），避免与已有 LEFT JOIN dgm 行翻倍。
	if filter.VisibleDeviceGrants != nil {
		builder = authz.ApplyDeviceVisibilityGrantsFilter(builder, "d.id", "d.technology", filter.VisibleDeviceGrants)
	} else {
		builder = authz.ApplyDeviceVisibilityFilter(builder, "d.id", filter.VisibleGroups)
	}
	if filter.UECountMax != nil {
		builder = builder.Where(sq.LtOrEq{"COALESCE(di.ue_count, 0)": *filter.UECountMax})
	}

	// Get total count with a separate query
	// 注意：去掉 DISTINCT，因为 device 与 device_info 是 1:1 关系
	countBuilder := storage.Psql.Select("COUNT(d.id)").
		From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		LeftJoin("device_info di ON d.id = di.device_id AND COALESCE(d.product_class, '') NOT LIKE 'UPS%'").
		Where(sq.NotEq{"d.latitude": nil}).
		Where(sq.NotEq{"d.longitude": nil})

	countBuilder = applyGeoGroupFilter(countBuilder, filter.GroupIDs, filter.IncludeUngrouped)
	// T-0162: countBuilder 同样翻译 filter.Status
	countBuilder = applyGeoStatusFilter(countBuilder, filter.Status)
	countBuilder = applyGeoBoundsFilter(countBuilder, filter.Bounds)
	if filter.Keyword != "" {
		// GIS 地图搜索字段（6 个）：SN / 名称 / IP / MAC / PCI / 设备名称
		// 注意：d.ip_address 是 INET 类型，需要用 host() 转为 TEXT 后才能 ILIKE
		searchFields := []string{
			"d.serial_number",
			"d.site_name",
			"host(d.ip_address)",
			"di.mac",
			"di.pci",
			"di.device_name",
		}
		if cond := BuildSearchOR(filter.Keyword, searchFields); cond != nil {
			countBuilder = countBuilder.Where(cond)
		}
	}
	// #64 设备组数据权限：count 与 list 同口径施加可见分组/制式收口。
	if filter.VisibleDeviceGrants != nil {
		countBuilder = authz.ApplyDeviceVisibilityGrantsFilter(countBuilder, "d.id", "d.technology", filter.VisibleDeviceGrants)
	} else {
		countBuilder = authz.ApplyDeviceVisibilityFilter(countBuilder, "d.id", filter.VisibleGroups)
	}
	if filter.UECountMax != nil {
		countBuilder = countBuilder.Where(sq.LtOrEq{"COALESCE(di.ue_count, 0)": *filter.UECountMax})
	}

	countQuery, countArgs, _ := countBuilder.ToSql()
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count geo devices: %w", err)
	}

	// Apply pagination
	if filter.PageSize > 0 {
		builder = builder.Limit(uint64(filter.PageSize))
	}
	if filter.Page > 1 && filter.PageSize > 0 {
		offset := uint64((filter.Page - 1) * filter.PageSize)
		builder = builder.Offset(offset)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list geo query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list geo devices: %w", err)
	}
	defer rows.Close()

	var devices []GeoDevice
	for rows.Next() {
		d, err := scanGeoDeviceRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan geo device: %w", err)
		}
		devices = append(devices, d)
	}

	if devices == nil {
		devices = []GeoDevice{}
	}

	return devices, total, nil
}

// GetGeoStats returns device statistics for map display.
// filter.VisibleGroups 是 #64 设备组数据权限的三态可见分组（nil 超管 / [] fail-closed / [g...] 限定）。
// filter.GroupIDs/IncludeUngrouped 由 service.splitGeoGroupIDs 归一化后传入，使 stats 与 ListGeo
// 在未分组节点选中场景下口径一致。filter.Status 接受三档筛选（onlineActive/onlineInactive/
// offline）翻译后的 model.DeviceStatus 列表，让顶部统计带随状态筛选变化与点位保持一致。
func (r *PgDeviceRepository) GetGeoStats(ctx context.Context, filter GeoStatsFilter) (*GeoStats, error) {
	// Build base condition for all queries
	baseCondition := sq.And{
		sq.NotEq{"d.latitude": nil},
		sq.NotEq{"d.longitude": nil},
	}

	// T-0162: 按 lifecycle_state + is_online 双维度 GROUP BY，scan 后派生回老
	// DeviceStatus key 给前端 GeoStats handler（前端按 onlineActive/onlineInactive/
	// offline 三档解释，详见 device_handler.go GetGeoStats）。
	statusBuilder := storage.Psql.Select(
		"d.lifecycle_state", "d.is_online", "COUNT(DISTINCT d.id) as cnt",
	).
		From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		Where(baseCondition).
		GroupBy("d.lifecycle_state", "d.is_online")

	statusBuilder = applyGeoGroupFilter(statusBuilder, filter.GroupIDs, filter.IncludeUngrouped)
	statusBuilder = applyGeoStatusFilter(statusBuilder, filter.Status)
	// #64 设备组数据权限：状态统计按可见分组+制式三态收口（子查询走 d.id 避免 LEFT JOIN 行翻倍）。
	if filter.VisibleDeviceGrants != nil {
		statusBuilder = authz.ApplyDeviceVisibilityGrantsFilter(statusBuilder, "d.id", "d.technology", filter.VisibleDeviceGrants)
	} else {
		statusBuilder = authz.ApplyDeviceVisibilityFilter(statusBuilder, "d.id", filter.VisibleGroups)
	}

	query, args, _ := statusBuilder.ToSql()
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get geo stats: %w", err)
	}
	defer rows.Close()

	stats := &GeoStats{
		StatusCount: make(map[model.DeviceStatus]int64),
	}

	for rows.Next() {
		var lifecycle model.DeviceLifecycle
		var isOnline bool
		var count int64
		if err := rows.Scan(&lifecycle, &isOnline, &count); err != nil {
			return nil, fmt.Errorf("scan geo stats: %w", err)
		}
		status := DeriveStatusFromLifecycle(lifecycle, isOnline)
		stats.StatusCount[status] += count
		stats.Total += count
	}

	// Query 2: Calculate center point (average latitude and longitude)
	centerBuilder := storage.Psql.Select(
		"AVG(d.latitude) as avg_lat",
		"AVG(d.longitude) as avg_lng",
	).
		From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		Where(baseCondition)

	centerBuilder = applyGeoGroupFilter(centerBuilder, filter.GroupIDs, filter.IncludeUngrouped)
	// #490: center 只按 group 收口，不随 status 变化——避免用户切换在线/离线档时
	// 地图中心漂移（center 语义是「当前组下设备的几何中心」，是定位锚点而非筛选反馈）。
	// #64 设备组数据权限：中心点计算同口径按可见分组收口。
	if filter.VisibleDeviceGrants != nil {
		centerBuilder = authz.ApplyDeviceVisibilityGrantsFilter(centerBuilder, "d.id", "d.technology", filter.VisibleDeviceGrants)
	} else {
		centerBuilder = authz.ApplyDeviceVisibilityFilter(centerBuilder, "d.id", filter.VisibleGroups)
	}

	centerQuery, centerArgs, _ := centerBuilder.ToSql()
	var avgLat, avgLng *float64
	err = r.pool.QueryRow(ctx, centerQuery, centerArgs...).Scan(&avgLat, &avgLng)
	if err != nil {
		return nil, fmt.Errorf("get geo center: %w", err)
	}

	if avgLat != nil && avgLng != nil {
		stats.Center = &GeoCenter{
			Latitude:  *avgLat,
			Longitude: *avgLng,
		}
	}

	// Query 3: UE=0 基站数（JOIN device_info 统计 ue_count=0 的设备）
	ueZeroBuilder := storage.Psql.Select("COUNT(DISTINCT d.id)").
		From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		LeftJoin("device_info di ON d.id = di.device_id AND COALESCE(d.product_class, '') NOT LIKE 'UPS%'").
		Where(baseCondition).
		Where(sq.LtOrEq{"COALESCE(di.ue_count, 0)": 0})

	ueZeroBuilder = applyGeoGroupFilter(ueZeroBuilder, filter.GroupIDs, filter.IncludeUngrouped)
	ueZeroBuilder = applyGeoStatusFilter(ueZeroBuilder, filter.Status)
	if filter.VisibleDeviceGrants != nil {
		ueZeroBuilder = authz.ApplyDeviceVisibilityGrantsFilter(ueZeroBuilder, "d.id", "d.technology", filter.VisibleDeviceGrants)
	} else {
		ueZeroBuilder = authz.ApplyDeviceVisibilityFilter(ueZeroBuilder, "d.id", filter.VisibleGroups)
	}

	ueZeroQuery, ueZeroArgs, _ := ueZeroBuilder.ToSql()
	if err := r.pool.QueryRow(ctx, ueZeroQuery, ueZeroArgs...).Scan(&stats.UEZeroCount); err != nil {
		return nil, fmt.Errorf("get geo ue zero count: %w", err)
	}

	return stats, nil
}

// SearchDevices searches devices by keyword for map display.
// visibleGroups 是 #64 设备组数据权限的三态可见分组（nil 超管 / [] fail-closed / [g...] 限定）。
func (r *PgDeviceRepository) SearchDevices(ctx context.Context, keyword string, limit int, visibleGrants []model.DeviceVisibilityGrant) ([]GeoDevice, error) {
	if keyword == "" {
		return []GeoDevice{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	// GIS 地图搜索字段（6 个）：SN / 名称 / IP / MAC / PCI / 设备名称
	// 注意：d.ip_address 是 INET 类型，需要用 host() 转为 TEXT 后才能 ILIKE
	searchFields := []string{
		"d.serial_number",
		"d.site_name",
		"host(d.ip_address)",
		"di.mac",
		"di.pci",
		"di.device_name",
	}

	// T-0162: 同 ListGeo，用 lifecycle_state + is_online
	builder := storage.Psql.Select(
		"d.id", "d.serial_number", "d.serial_number as name",
		"d.lifecycle_state", "d.is_online",
		"d.latitude", "d.longitude", "dg.id as group_id", "dg.name as group_name",
		"d.site_name as address", "0 as alarm_count", "d.model_name as type",
		"COALESCE(host(d.ip_address), '')", "COALESCE(di.mac, '')", "COALESCE(di.pci, '')", "COALESCE(di.device_name, '')",
		"COALESCE(di.ue_count, 0)",
		"NULL::int as highest_alarm_severity",
		"0 as highest_severity_alarm_count",
	).From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		LeftJoin("device_info di ON d.id = di.device_id AND COALESCE(d.product_class, '') NOT LIKE 'UPS%'")

	// 复用 BuildSearchOR：支持逗号分隔多值搜索 + 50 个值限制
	if cond := BuildSearchOR(keyword, searchFields); cond != nil {
		builder = builder.Where(cond)
	} else {
		return []GeoDevice{}, nil
	}
	// #64 设备组数据权限：搜索结果按可见分组+制式三态收口（子查询走 d.id 避免 LEFT JOIN 行翻倍）。
	if visibleGrants != nil {
		builder = authz.ApplyDeviceVisibilityGrantsFilter(builder, "d.id", "d.technology", visibleGrants)
	} else {
		builder = authz.ApplyDeviceVisibilityFilter(builder, "d.id", nil)
	}

	builder = builder.Limit(uint64(limit))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build search devices query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search devices: %w", err)
	}
	defer rows.Close()

	var devices []GeoDevice
	for rows.Next() {
		// #117: SearchDevices 无 latitude/longitude IS NOT NULL 过滤（搜索按
		// 关键字命中，不要求设备已有坐标），扫描必须容忍 NULL 坐标。
		d, err := scanGeoDeviceRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		devices = append(devices, d)
	}

	if devices == nil {
		devices = []GeoDevice{}
	}

	return devices, nil
}

// ===== Recycle Bin Operations =====

// recycleBinSelectColumns 回收站轻量级查询字段（无需告警聚合）。
// 回收站主要展示：基本设备信息 + device_info + 分组；不需要活动告警数据。
func recycleBinSelectColumns() []string {
	return []string{
		// devices columns
		"d.id", "d.serial_number", "d.oui", "d.product_class", "d.manufacturer", "d.model_name",
		"d.carrier", "d.technology",
		"d.lifecycle_state", "d.is_online",
		"d.firmware_version",
		"host(d.ip_address) as ip_address", "d.connection_request_url",
		"d.nat_detected", "d.udp_connection_request_address",
		"d.last_inform_at", "d.last_inform_events",
		"d.last_boot_at", "d.boot_count",
		"d.inform_interval", "d.site_name",
		`CASE WHEN ` + upsProductClassPredicate + ` THEN udi.site_id ELSE d.site_id END AS site_id`,
		"d.latitude", "d.longitude",
		"d.location_source_mode",
		"dlo.latitude AS reported_latitude", "dlo.longitude AS reported_longitude",
		"dlo.gps_height AS reported_gps_height", "dlo.observed_at AS reported_observed_at",
		"dlo.version AS reported_version", "dlo.source_path AS reported_source_path",
		"d.extension_data", "d.created_at", "d.updated_at", "d.deleted_at", "d.deleted_by",
		"d.recycle_type", "d.recycle_executor",
		"d.last_param_sync_at",
		"d.last_offline_reason",
		// device_groups columns
		"dg.id as group_id",
		"dg.name as group_name",
		"COALESCE(dgm.source_type, 'auto') as source_type",
		// device info columns (no alarm_severity needed for recycle bin).
		// UPS uses device_ups_info; radio devices keep using device_info.
		`CASE WHEN ` + upsProductClassPredicate + ` THEN udi.device_name ELSE di.device_name END AS device_name`,
		`CASE WHEN ` + upsProductClassPredicate + ` THEN udi.address ELSE di.address END AS address`,
		`CASE WHEN ` + upsProductClassPredicate + ` THEN udi.remark ELSE di.remark END AS remark`,
		"di.project_status", "di.height",
		"di.eci", "di.pci", "di.cell_id", "di.freq_point", "di.bandwidth", "di.transmit_power", "di.plmn",
		"di.rf_status", "di.cell_status", "di.op_state", "di.mme_status", "di.sync_status", "di.kpi_status",
		"di.num_of_cells", "di.gps_status", "COALESCE(di.ue_count, 0)",
		"NULL::text AS alarm_severity", // Placeholder for compatibility with DeviceWithInfo
		"di.license_status",
		"di.mac", "di.hardware_version",
		`CASE WHEN ` + upsProductClassPredicate + ` THEN udi.first_online_time ELSE di.first_online_time END AS first_online_time`,
		`CASE WHEN ` + upsProductClassPredicate + ` THEN udi.last_online_time ELSE di.last_online_time END AS last_online_time`,
		`CASE WHEN ` + upsProductClassPredicate + ` THEN udi.last_offline_time ELSE di.last_offline_time END AS last_offline_time`,
		`CASE WHEN ` + upsProductClassPredicate + ` THEN udi.run_time ELSE di.run_time END AS run_time`,
		`CASE WHEN ` + upsProductClassPredicate + ` THEN udi.cumulative_online_duration ELSE di.cumulative_online_duration END AS cumulative_online_duration`,
		// Phase 2/3 扩展列
		"di.tac", "di.lac", "di.band", "di.ul_earfcn",
		"di.subframe_assignment", "di.special_subframe", "di.root_index",
		"di.gps_satellites", "di.gps_height", "di.lock_status",
		"di.admin_state", "di.ipsec_addr",
		"di.enb_id", "di.network_model",
		// GSM/BTS 专属字段
		"di.bsc_select", "di.oml_remote_ip", "di.oml_remote_ip_bak", "di.ipa_unit_id",
		// 设备名称同步
		"di.name_sync_pending", "di.lmt_device_name",
		`CASE
			WHEN di.oml_remote_ip IS NOT NULL AND di.oml_remote_ip <> '' AND d.is_online
			THEN 'connected'
			WHEN di.oml_remote_ip IS NOT NULL AND di.oml_remote_ip <> ''
			THEN 'disconnected'
			ELSE NULL
		END AS bsc_link_status`,
		// 在线时长派生
		`CASE
			WHEN di.last_online_time IS NULL THEN NULL
			WHEN d.is_online THEN EXTRACT(EPOCH FROM (NOW() - di.last_online_time))::bigint
			WHEN di.last_offline_time IS NOT NULL AND di.last_offline_time > di.last_online_time
				THEN EXTRACT(EPOCH FROM (di.last_offline_time - di.last_online_time))::bigint
			ELSE NULL
		END AS online_duration`,
		// 回收时的离线时长快照：以 deleted_at - last_inform_at 计算，
		// 避免记录进入回收站后继续随 NOW() 增长。
		`CASE
			WHEN d.deleted_at IS NOT NULL AND d.last_inform_at IS NOT NULL
			THEN GREATEST(EXTRACT(EPOCH FROM (d.deleted_at - d.last_inform_at)), 0)::bigint
			ELSE NULL
		END AS offline_seconds`,
		`CASE
			WHEN d.deleted_at IS NOT NULL AND d.last_inform_at IS NOT NULL
			THEN FLOOR(GREATEST(EXTRACT(EPOCH FROM (d.deleted_at - d.last_inform_at)), 0) / 86400)::bigint
			ELSE NULL
		END AS offline_days`,
		`CASE
			WHEN d.deleted_at IS NOT NULL AND d.last_inform_at IS NOT NULL
			THEN FLOOR((GREATEST(EXTRACT(EPOCH FROM (d.deleted_at - d.last_inform_at)), 0) % 86400) / 3600)::bigint
			ELSE NULL
		END AS offline_hours`,
		`CASE
			WHEN d.deleted_at IS NOT NULL AND d.last_inform_at IS NOT NULL
			THEN FLOOR((GREATEST(EXTRACT(EPOCH FROM (d.deleted_at - d.last_inform_at)), 0) % 3600) / 60)::bigint
			ELSE NULL
		END AS offline_minutes`,
		"FALSE AS param_sync_running",     // Placeholder for shared DeviceWithInfo scanner
		"NULL::int AS active_alarm_count", // Placeholder for compatibility
		`CASE
			WHEN COALESCE(d.product_class, '') LIKE 'UPS%' THEN 'UPS'
			ELSE 'BASE_STATION'
		END AS device_type`,
		"udi.external_ip", "udi.total_voltage", "udi.total_temperature", "udi.total_current",
		"udi.software_version", "udi.hardware_version", "udi.manufacturer", "udi.manufacturer_oui",
		"udi.run_time", "udi.bms_charging",
		"udi.ac_power", "udi.ac_voltage", "udi.dc_voltage", "udi.dc_current", "udi.board_temperature",
		"udi.sfp_state", "udi.port0_state", "udi.port1_state", "udi.port2_state", "udi.port3_state",
		"udi.average_soc", "udi.pack_counts", "udi.last_inform_at",
	}
}

func recycleBinSearchFields() []string {
	return []string{
		"d.serial_number",
		"d.site_name",
		"udi.device_name",
		"udi.site_id",
		"di.mac",
	}
}

func buildRecycleBinListBuilders(filter RecycleBinFilter) (sq.SelectBuilder, sq.SelectBuilder) {
	builder := storage.Psql.Select(recycleBinSelectColumns()...).
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id AND COALESCE(d.product_class, '') NOT LIKE 'UPS%'").
		LeftJoin("device_ups_info udi ON udi.device_id = d.id AND COALESCE(d.product_class, '') LIKE 'UPS%'").
		LeftJoin("device_location_observations dlo ON d.id = dlo.device_id").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dg.id = dgm.group_id").
		Where(sq.NotEq{"d.deleted_at": nil})

	countBuilder := storage.Psql.Select("COUNT(*)").
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id AND COALESCE(d.product_class, '') NOT LIKE 'UPS%'").
		LeftJoin("device_ups_info udi ON udi.device_id = d.id AND COALESCE(d.product_class, '') LIKE 'UPS%'").
		Where(sq.NotEq{"d.deleted_at": nil})

	// Apply filters
	if filter.Search != nil {
		if cond := BuildSearchOR(*filter.Search, recycleBinSearchFields()); cond != nil {
			builder = builder.Where(cond)
			countBuilder = countBuilder.Where(cond)
		}
	}

	if filter.Carrier != nil {
		builder = builder.Where(sq.Eq{"d.carrier": *filter.Carrier})
		countBuilder = countBuilder.Where(sq.Eq{"d.carrier": *filter.Carrier})
	}

	if filter.Technology != nil {
		builder = builder.Where(sq.Eq{"d.technology": *filter.Technology})
		countBuilder = countBuilder.Where(sq.Eq{"d.technology": *filter.Technology})
	}

	if filter.DeletedBy != nil && *filter.DeletedBy != "" {
		builder = builder.Where(sq.Like{"d.deleted_by": "%" + *filter.DeletedBy + "%"})
		countBuilder = countBuilder.Where(sq.Like{"d.deleted_by": "%" + *filter.DeletedBy + "%"})
	}

	// GroupID filter. 默认组额外兜底历史无归属设备；新数据仍写真实默认组 membership。
	if filter.GroupID != nil {
		if filter.GroupID.String() == global.DefaultLevel2GroupID {
			cond := sq.Or{
				sq.Eq{"dgm.group_id": *filter.GroupID},
				sq.Expr(ungroupedDevicesWhere),
			}
			builder = builder.Where(cond)
			countBuilder = countBuilder.
				LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
				Where(cond)
		} else {
			builder = builder.Where(sq.Eq{"dgm.group_id": *filter.GroupID})
			countBuilder = countBuilder.
				Join("device_group_members dgm ON d.id = dgm.device_id").
				Where(sq.Eq{"dgm.group_id": *filter.GroupID})
		}
	}

	return builder, countBuilder
}

// ListRecycleBin returns soft-deleted devices with filtering.
func (r *PgDeviceRepository) ListRecycleBin(ctx context.Context, filter RecycleBinFilter) (*model.ListResponse[DeviceWithInfo], error) {
	// Build base query for deleted devices with group info and device_info
	// T-2026-07-02: 新增 device_info LEFT JOIN 以支持完整的设备信息返回（修复缺失字段）。
	builder, countBuilder := buildRecycleBinListBuilders(filter)

	// Get total count
	var total int64
	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count query: %w", err)
	}
	err = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count recycle bin devices: %w", err)
	}

	// Apply sorting
	sortCol := "d.deleted_at"
	if filter.SortBy != "" {
		if allowedSortColumns[filter.SortBy] {
			sortCol = "d." + filter.SortBy
		}
	}
	sortOrder := "DESC"
	if filter.SortDir == "asc" {
		sortOrder = "ASC"
	}
	builder = builder.OrderBy(sortCol + " " + sortOrder)

	// Apply pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	builder = builder.Limit(uint64(pageSize)).Offset(uint64(offset))

	// Execute query
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list recycle bin query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list recycle bin devices: %w", err)
	}
	defer rows.Close()

	var devices []DeviceWithInfo
	for rows.Next() {
		d, err := scanDeviceWithInfoRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recycle bin device: %w", err)
		}
		devices = append(devices, *d)
	}

	if devices == nil {
		devices = []DeviceWithInfo{}
	}

	return &model.ListResponse[DeviceWithInfo]{
		Items:    devices,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// RestoreDevices restores soft-deleted devices by setting deleted_at to NULL.
//
// #378：改为「可部分成功」。原实现是单条全或无批量 UPDATE，无 SN 冲突处理——
// 当回收站某软删行与某活跃行同 serial_number+carrier 时，恢复（deleted_at→NULL）
// 触发部分唯一索引 (serial_number, carrier) WHERE deleted_at IS NULL 的 23505
// unique_violation，整条批量 UPDATE 回滚 → 一台冲突拖垮整批 → handler 一律 500。
//
// 现用 NOT EXISTS 守卫让冲突行自然不被更新（一条语句、无需逐条事务），RETURNING
// 返回真正恢复的 id；再单独查出被跳过（仍软删且存在同 SN+carrier 活跃行）的明细
// 回传上层。RowsAffected 反映实际恢复数，冲突场景不再 500。
func (r *PgDeviceRepository) RestoreDevices(ctx context.Context, ids []uuid.UUID) (*RestoreResult, error) {
	result := &RestoreResult{}
	if len(ids) == 0 {
		return result, nil
	}

	// 1) 恢复——NOT EXISTS 守卫排除会撞部分唯一索引的冲突行。RETURNING 拿恢复 id。
	rows, err := r.pool.Query(ctx,
		`UPDATE devices d
		    SET deleted_at = NULL, deleted_by = '', recycle_type = '',
		        recycle_executor = '', updated_at = NOW()
		  WHERE d.id = ANY($1)
		    AND d.deleted_at IS NOT NULL
		    AND NOT EXISTS (
		        SELECT 1 FROM devices a
		         WHERE a.serial_number = d.serial_number
		           AND a.carrier = d.carrier
		           AND a.deleted_at IS NULL
		    )
		RETURNING d.id`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("restore devices: %w", err)
	}
	restoredIDs := make(map[uuid.UUID]struct{})
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan restored device id: %w", err)
		}
		restoredIDs[id] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate restored device ids: %w", err)
	}
	result.Restored = int64(len(restoredIDs))

	// 2) 查出被跳过（仍软删且存在同 SN+carrier 活跃行）的冲突明细回传上层。
	confRows, err := r.pool.Query(ctx,
		`SELECT d.id, d.serial_number
		   FROM devices d
		  WHERE d.id = ANY($1)
		    AND d.deleted_at IS NOT NULL
		    AND EXISTS (
		        SELECT 1 FROM devices a
		         WHERE a.serial_number = d.serial_number
		           AND a.carrier = d.carrier
		           AND a.deleted_at IS NULL
		    )`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("query restore conflicts: %w", err)
	}
	defer confRows.Close()
	for confRows.Next() {
		var id uuid.UUID
		var sn string
		if err := confRows.Scan(&id, &sn); err != nil {
			return nil, fmt.Errorf("scan restore conflict: %w", err)
		}
		result.Conflicts = append(result.Conflicts, RestoreConflict{
			ID:           id.String(),
			SerialNumber: sn,
			Reason:       "serial_number_conflict",
		})
	}
	if err := confRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate restore conflicts: %w", err)
	}
	result.Skipped = int64(len(result.Conflicts))

	return result, nil
}

// PermanentDelete permanently removes devices from the database.
// This also removes related device_group_members and device-specific info records.
// Returns the number of devices actually deleted.
func (r *PgDeviceRepository) PermanentDelete(ctx context.Context, ids []uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Remove group memberships (if any exist for deleted devices)
	_, err = tx.Exec(ctx,
		`DELETE FROM device_group_members WHERE device_id = ANY($1)`,
		ids,
	)
	if err != nil {
		return 0, fmt.Errorf("delete device_group_members: %w", err)
	}

	// Remove radio device_info records and UPS-only info records.
	_, err = tx.Exec(ctx,
		`DELETE FROM device_info di
		  USING devices d
		  WHERE di.device_id = d.id
		    AND d.id = ANY($1)
		    AND COALESCE(d.product_class, '') NOT LIKE 'UPS%'`,
		ids,
	)
	if err != nil {
		return 0, fmt.Errorf("delete device_info: %w", err)
	}
	_, err = tx.Exec(ctx,
		`DELETE FROM device_ups_info WHERE device_id = ANY($1)`,
		ids,
	)
	if err != nil {
		return 0, fmt.Errorf("delete device_ups_info: %w", err)
	}

	// Permanently delete devices
	tag, err := tx.Exec(ctx,
		`DELETE FROM devices WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return 0, fmt.Errorf("permanent delete devices: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit permanent delete: %w", err)
	}

	return tag.RowsAffected(), nil
}

// ListStaleForParamSync finds active devices whose last_param_sync_at IS NULL or < threshold.
// Used by PeriodicSyncer (T-0124) to enqueue Path B sync per device.
// NULLS FIRST ensures devices that have never been synced are prioritized.
func (r *PgDeviceRepository) ListStaleForParamSync(ctx context.Context, threshold time.Time, limit int) ([]*model.Device, error) {
	// T-0162: "active" 语义 = lifecycle_state='commissioned' AND is_online=TRUE
	builder := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		Where(sq.Eq{"d.lifecycle_state": model.LifecycleCommissioned}).
		Where(sq.Eq{"d.is_online": true}).
		Where(sq.Or{
			sq.Eq{"d.last_param_sync_at": nil},
			sq.Lt{"d.last_param_sync_at": threshold},
		}).
		Where(sq.Expr("NOT (" + upsProductClassPredicate + ")")).
		Where(notDeleted).
		OrderBy("d.last_param_sync_at ASC NULLS FIRST").
		Limit(uint64(limit))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list stale param sync devices query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list stale param sync devices: %w", err)
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan stale param sync device: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// UpdateLastParamSyncAt 成功路径回写 — 由 HandleSyncResultPathB BatchUpsert 成功后调用。
// 同时清空失败列(migration 000142):成功 trump 之前的失败,前端只需看哪个时间戳非空。
func (r *PgDeviceRepository) UpdateLastParamSyncAt(ctx context.Context, id uuid.UUID, at time.Time) error {
	query, args, err := storage.Psql.Update("devices").
		Set("last_param_sync_at", at).
		Set("last_param_sync_failed_at", nil).
		Set("last_param_sync_error", nil).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"deleted_at": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update last_param_sync_at query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update last_param_sync_at: %w", err)
	}
	return nil
}

// UpdateLastParamSyncFailed 失败路径回写 — 由 sync 失败订阅者(rpc_response_subscriber
// handleSyncTaskFailed) 在 Path B GPV task 失败时调用。同时清空成功时刻:让前端 idle
// 状态明确显示"上次同步失败 X 时间前 · 错误 ...",且 PeriodicSyncer 会把该设备当作
// "需重新同步"在下个周期重排(配合 FindStaleDevices 用 last_param_sync_at IS NULL
// 或过期判定)。
func (r *PgDeviceRepository) UpdateLastParamSyncFailed(ctx context.Context, id uuid.UUID, failedAt time.Time, errMsg string) error {
	query, args, err := storage.Psql.Update("devices").
		Set("last_param_sync_failed_at", failedAt).
		Set("last_param_sync_error", errMsg).
		Set("last_param_sync_at", nil).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"deleted_at": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update last_param_sync_failed query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update last_param_sync_failed: %w", err)
	}
	return nil
}

// FindStaleDevices finds active devices that haven't sent Inform within the threshold.
//
// DEPRECATED (T-0173): 原 OfflineDetector 唯一调用方已废弃。新代码请用
// FindStaleDevicesAdaptive(minStaleSec) —— 自适应阈值,按设备 inform_interval
// 动态计算 max(2×inform_interval, minStaleSec) 秒。
//
// T-0162: "active" 语义 = lifecycle_state='commissioned' AND is_online=TRUE。
func (r *PgDeviceRepository) FindStaleDevices(ctx context.Context, threshold time.Time, limit int) ([]*model.Device, error) {
	builder := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		Where(sq.Eq{"d.lifecycle_state": model.LifecycleCommissioned}).
		Where(sq.Eq{"d.is_online": true}).
		Where(sq.Lt{"d.last_inform_at": threshold}).
		Where(notDeleted).
		OrderBy("d.last_inform_at ASC").
		Limit(uint64(limit))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find stale devices query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find stale devices: %w", err)
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan stale device: %w", err)
		}
		devices = append(devices, d)
	}

	return devices, nil
}

// ListSerialsByIDs 按 ID 批量查 serial_number。supplied IDs 中找不到行的 ID 不在结果 map 中。
// 用于 service 层批量 Delete/Restore/PermanentDelete 前后拿到 SN 列表统一清 cache。
func (r *PgDeviceRepository) ListSerialsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]string{}, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, serial_number FROM devices WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("list serials by ids: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]string, len(ids))
	for rows.Next() {
		var id uuid.UUID
		var sn string
		if err := rows.Scan(&id, &sn); err != nil {
			return nil, fmt.Errorf("scan id-sn pair: %w", err)
		}
		out[id] = sn
	}
	return out, rows.Err()
}

// FindOfflineForRecycle 查询已离线且 last_inform_at < olderThan、尚未软删除的设备 ID 列表。
// 返回 ID 列表供调用方批量软删除（BatchDelete）。
// limit <= 0 时回退到 500 防止单批过大。
func (r *PgDeviceRepository) FindOfflineForRecycle(ctx context.Context, olderThan time.Time, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id FROM devices
		 WHERE is_online = FALSE
		   AND last_inform_at IS NOT NULL
		   AND last_inform_at < $1
		   AND deleted_at IS NULL
		 ORDER BY last_inform_at ASC
		 LIMIT $2`,
		olderThan, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("find offline for recycle: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan device id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListProductClasses returns distinct product_class values, sorted alphabetically.
//
// qa-614 #379：合并 devices ∪ firmware_versions 两个来源。固件上传抽屉的"产品类型标识"
// 下拉消费本端点；若只取 devices 表，BM 等"已有固件但暂无在线设备"的产品类就选不到 →
// "BM 版本导入失败"。把已上传固件的 product_class 一并并入，保证这些类可被再次选中。
func (r *PgDeviceRepository) ListProductClasses(ctx context.Context) ([]string, error) {
	query := `SELECT product_class FROM (
			SELECT DISTINCT product_class FROM devices
			WHERE product_class IS NOT NULL AND product_class != ''
			UNION
			SELECT DISTINCT product_class FROM firmware_versions
			WHERE product_class IS NOT NULL AND product_class != ''
		) merged
		ORDER BY product_class`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list product classes: %w", err)
	}
	defer rows.Close()

	var classes []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("scan product class: %w", err)
		}
		classes = append(classes, c)
	}

	if classes == nil {
		classes = []string{}
	}
	return classes, nil
}

// ===== T-0173: 异步在线状态治理（DeviceStatusReconciler）专用方法 =====
//
// 这两个方法故意只挂在 *PgDeviceRepository 上,不进 DeviceRepository 接口 ——
// 全仓 11 个手写 mock 不需要为只被 Reconciler 单点消费的方法增加桩。Reconciler
// 自己在 status_reconciler.go 定义 2 方法的窄接口 statusReconcilerRepo,
// *PgDeviceRepository 自然满足。

// FindStaleDevicesAdaptive 找出"超过自适应阈值未上报心跳"的在线设备。
//
// 自适应阈值 = max(2 × inform_interval, minStaleSec)。inform_interval 是每设备
// 心跳间隔（来自 Device.ManagementServer.PeriodicInformInterval,默认 300s）。
// 短心跳设备（如 60s）2 倍只有 120s,不足以判定真正离线;minStaleSec 给一个全局
// 下限（推荐 600s),避免在心跳轻微抖动时误标离线。
//
// 用于 DeviceStatusReconciler.detect。返回按 last_inform_at ASC 排序,
// 优先处理最久未心跳的设备。
func (r *PgDeviceRepository) FindStaleDevicesAdaptive(ctx context.Context, minStaleSec int, limit int) ([]*model.Device, error) {
	if minStaleSec <= 0 {
		minStaleSec = 600
	}
	if limit <= 0 {
		limit = 1000
	}
	// 连接状态检查：凡 is_online=true 但超过 max(2×inform_interval, minStaleSec) 未上报的设备
	// 一律判离线，不再限定 lifecycle_state=commissioned —— 否则 maintenance 等其它状态的"僵尸
	// 在线"设备永不离线（本次 4011 假在线里 1500 台即 maintenance）。
	// last_inform_at IS NULL 的脏数据（从未上报却标在线）由迁移一次性清理；收到 Inform 必写
	// last_inform_at，故正常在线设备该列非空，离线累计可正确计算。
	builder := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		Where(sq.Eq{"d.is_online": true}).
		Where(notDeleted).
		Where("d.last_inform_at IS NOT NULL").
		Where(
			"d.last_inform_at < NOW() - (GREATEST(COALESCE(d.inform_interval, 300) * 2, ?) * INTERVAL '1 second')",
			minStaleSec,
		).
		OrderBy("d.last_inform_at ASC").
		Limit(uint64(limit))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find stale devices adaptive query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find stale devices adaptive: %w", err)
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan stale device adaptive: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// cpeProductClassPredicate 是判定设备是否属于 "CPE 类" 的 SQL 谓词。
//
// 与 topology.inferNodeType 的分类口径对齐：product_class 命中 CPE/Home/
// Residential/Indoor 任一关键字即视为 CPE，其余（含基站 eNB/gNB、网关等）一律
// 归 "基站类"。表里无显式 CPE 列，故按 product_class 模式匹配。
const cpeProductClassPredicate = `(
	d.product_class ILIKE '%cpe%' OR
	d.product_class ILIKE '%home%' OR
	d.product_class ILIKE '%residential%' OR
	d.product_class ILIKE '%indoor%'
)`

// FindStaleDevicesByClass 找出 "上次心跳距今 > 该设备类阈值" 的在线设备。
//
// issue #203：离线判定接 sys_configs 实时配置。按设备类（基站 / CPE）应用各自
// 阈值，**不叠加 2×inform_interval 安全网**——纯按配置阈值与 last_inform_at 的
// 时间差判离线，让用户配置直接生效。CPE 与基站分类见 cpeProductClassPredicate。
//
// enbThresholdSec：基站类阈值（秒）；cpeThresholdSec：CPE 类阈值（秒）。
// 返回按 last_inform_at ASC 排序，优先处理最久未心跳的设备。
func (r *PgDeviceRepository) FindStaleDevicesByClass(ctx context.Context, enbThresholdSec, cpeThresholdSec, upsThresholdSec, limit int) ([]*model.Device, error) {
	if enbThresholdSec <= 0 {
		enbThresholdSec = defaultENBOfflineSec
	}
	if cpeThresholdSec <= 0 {
		cpeThresholdSec = defaultCPEOfflineSec
	}
	if upsThresholdSec <= 0 {
		upsThresholdSec = defaultUPSOfflineSec
	}
	if limit <= 0 {
		limit = 1000
	}
	// 凡 is_online=true 但 last_inform_at 距今超过 "该类阈值" 的设备一律判离线。
	// 阈值按 product_class 分类选取（CPE 用 cpeThresholdSec，其余用 enbThresholdSec）。
	// 不限定 lifecycle_state（与 FindStaleDevicesAdaptive 一致，避免 maintenance 等
	// 状态的僵尸在线设备永不离线）。
	//
	// issue #203 回合2：占位符经 pgx 绑成 text，PG 无 `text * interval` 运算符
	// （报 operator does not exist: text * interval, SQLSTATE 42883），整轮扫描失败、
	// 从不进入判离线分支。用 make_interval(secs => (?)::int) 把秒数显式转 int 再造
	// interval，绕开 text*interval 运算。
	staleExpr := fmt.Sprintf(
		"d.last_inform_at < NOW() - make_interval(secs => (CASE WHEN %s THEN (?)::int WHEN %s THEN (?)::int ELSE (?)::int END))",
		upsProductClassPredicate,
		cpeProductClassPredicate,
	)
	builder := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		Where(sq.Eq{"d.is_online": true}).
		Where(notDeleted).
		Where("d.last_inform_at IS NOT NULL").
		Where(staleExpr, upsThresholdSec, cpeThresholdSec, enbThresholdSec).
		OrderBy("d.last_inform_at ASC").
		Limit(uint64(limit))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find stale devices by class query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find stale devices by class: %w", err)
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan stale device by class: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// FindOfflineDevicesBefore 找出当前仍离线且 last_offline_time 早于 cutoff 的设备。
//
// 用于 F04 离线超时告警清理：设备离线满 1 小时仍未恢复上线时，将当前告警转历史。
// 只返回 commissioned 且未软删设备，避免把未入网或已删除设备卷进告警生命周期处理。
func (r *PgDeviceRepository) FindOfflineDevicesBefore(ctx context.Context, cutoff time.Time, limit int) ([]*model.Device, error) {
	if limit <= 0 {
		limit = 200
	}

	query, args, err := buildFindOfflineDevicesBeforeQuery(cutoff, limit)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find offline devices before cutoff: %w", err)
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan offline device row: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func buildFindOfflineDevicesBeforeQuery(cutoff time.Time, limit int) (string, []interface{}, error) {
	lastOfflineExpr := `CASE WHEN COALESCE(d.product_class, '') LIKE 'UPS%' THEN udi.last_offline_time ELSE di.last_offline_time END`
	builder := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id AND COALESCE(d.product_class, '') NOT LIKE 'UPS%'").
		LeftJoin("device_ups_info udi ON udi.device_id = d.id AND COALESCE(d.product_class, '') LIKE 'UPS%'").
		Where(sq.Eq{"d.lifecycle_state": model.LifecycleCommissioned}).
		Where(sq.Eq{"d.is_online": false}).
		Where(notDeleted).
		Where(lastOfflineExpr + " IS NOT NULL").
		Where(sq.LtOrEq{lastOfflineExpr: cutoff}).
		OrderBy(lastOfflineExpr + " ASC").
		Limit(uint64(limit))

	query, args, err := builder.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build find offline devices query: %w", err)
	}
	return query, args, nil
}

// MarkOfflineWithAccounting 事务性把单台设备从"在线"翻转到"离线":
//  1. devices.is_online → false（仅当当前为 true,否则整个事务空转）
//  2. devices.last_offline_reason → reason
//  3. 基站写 device_info.last_offline_time；UPS 写 device_ups_info.last_offline_time
//  4. 对应信息表 cumulative_online_duration += GREATEST(0, now - last_online_time)
//
// 全部在同一 TX 内完成,失败回滚不留半成品。
//
// 返回 (transitioned, error):
//   - transitioned=true 表示这次调用真的把设备从 online→offline 翻转了；
//   - transitioned=false 表示调用前设备已 offline / 已删除,事务无副作用,调用方
//     不应再发 device.offline 事件（幂等保护)。
//
// 入参 now 由调用方传入而非 NOW(),便于测试与事件载荷时间戳对齐。
func (r *PgDeviceRepository) MarkOfflineWithAccounting(ctx context.Context, deviceID uuid.UUID, reason string, now time.Time) (bool, error) {
	if reason == "" {
		reason = "unknown"
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin mark offline tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Step 1: 翻 is_online + 记原因,只在当前在线时才动手。
	var productClass string
	err = tx.QueryRow(ctx, `
		UPDATE devices
		   SET is_online = false,
		       last_offline_reason = $2,
		       updated_at = NOW()
		 WHERE id = $1
		   AND is_online = true
		   AND deleted_at IS NULL
		RETURNING COALESCE(product_class, '')`,
		deviceID, reason).Scan(&productClass)
	if err == pgx.ErrNoRows {
		// 已是 offline 或不存在 / 已软删 —— 幂等返回。
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit no-op tx: %w", err)
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("update devices offline: %w", err)
	}

	// Step 2: 离线时间戳 + 累计在线时长。
	// last_online_time 为 NULL 时（如设备从未触发 RecordOnline）累加 0,保守。
	if isUPSProductClass(productClass) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO device_ups_info (
				device_id, last_offline_time, cumulative_online_duration, created_at, updated_at
			) VALUES ($1, $2, 0, NOW(), NOW())
			ON CONFLICT (device_id) DO UPDATE SET
				last_offline_time = EXCLUDED.last_offline_time,
				cumulative_online_duration = COALESCE(device_ups_info.cumulative_online_duration, 0)
					+ GREATEST(0, EXTRACT(EPOCH FROM ($2 - COALESCE(device_ups_info.last_online_time, $2)))::bigint),
				updated_at = NOW()`,
			deviceID, now); err != nil {
			return false, fmt.Errorf("update UPS device info offline accounting: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE device_info
			   SET last_offline_time = $2,
			       cumulative_online_duration = COALESCE(cumulative_online_duration, 0)
			                                  + GREATEST(0, EXTRACT(EPOCH FROM ($2 - COALESCE(last_online_time, $2)))::bigint)
			 WHERE device_id = $1`,
			deviceID, now); err != nil {
			return false, fmt.Errorf("update device_info offline accounting: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit mark offline tx: %w", err)
	}
	return true, nil
}
