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
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// ===== 接口定义 =====

// DeviceFilter specifies criteria for listing devices.
type DeviceFilter struct {
	Carrier    *model.CarrierCode
	Technology *model.Technology
	Status     *model.DeviceStatus
	OUI        *string
	SN         *string // exact match on serial_number
	Search     *string // fuzzy search across serial_number/site_name/manufacturer/device_name/address

	// Group filters
	GroupID       *uuid.UUID  // filter by specific device group
	VisibleGroups []uuid.UUID // data permission: restrict to these groups (nil = no restriction)

	// Extended filters (device_info / devices additional fields)
	Manufacturer  *string // devices.manufacturer exact match
	ProductClass  *string // devices.product_class exact match
	RFStatus      *string // device_info.rf_status exact match
	CellStatus    *string // device_info.cell_status exact match
	ProjectStatus *string // device_info.project_status exact match
	GPSStatus     *string // device_info.gps_status exact match
	AlarmSeverity *string // device_info.alarm_severity exact match
	LicenseStatus *string // device_info.license_status exact match
	OpState       *string // "1" = active (status='active'), "0" = not active (status!='active')

	model.ListRequest
}

// RecycleBinFilter specifies criteria for listing soft-deleted devices.
type RecycleBinFilter struct {
	Search     *string // fuzzy search across serial_number/site_name
	Carrier    *model.CarrierCode
	Technology *model.Technology
	GroupID    *uuid.UUID // filter by original device group
	DeletedBy  *string    // filter by who deleted the device

	model.ListRequest
}

// GeoDeviceFilter specifies criteria for listing devices with geo data.
type GeoDeviceFilter struct {
	GroupIDs []string
	Status   []model.DeviceStatus
	Keyword  string
	Bounds   *GeoBounds
	Page     int
	PageSize int
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
}

// GeoStats represents device statistics for map display.
type GeoStats struct {
	Total       int64                        `json:"total"`
	StatusCount map[model.DeviceStatus]int64 `json:"status_count"`
	AlarmCount  int64                        `json:"alarm_count"`
	Center      *GeoCenter                   `json:"center,omitempty"` // 平均经纬度中心点
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
	GetGeoStats(ctx context.Context, groupIDs []string) (*GeoStats, error)
	// SearchDevices searches devices by keyword for map display.
	SearchDevices(ctx context.Context, keyword string, limit int) ([]GeoDevice, error)
	// ListProductClasses returns distinct product_class values from devices,
	// merged with a set of mandatory types that must always appear.
	ListProductClasses(ctx context.Context) ([]string, error)
	// FindStaleDevices finds active devices that haven't sent Inform within the threshold.
	// Used by OfflineDetector to mark devices as offline.
	FindStaleDevices(ctx context.Context, threshold time.Time, limit int) ([]*model.Device, error)
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
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error
	// RecordBoot atomically increments boot_count and sets last_boot_at for the device
	// identified by serial number. Invoked when the ACS receives a "1 BOOT" or
	// "M Reboot" Inform. Returns the updated boot_count.
	RecordBoot(ctx context.Context, sn string, at time.Time) (int, error)
	// RecycleBin operations
	// ListRecycleBin returns soft-deleted devices with filtering.
	ListRecycleBin(ctx context.Context, filter RecycleBinFilter) (*model.ListResponse[model.Device], error)
	// RestoreDevices restores soft-deleted devices (sets deleted_at to NULL).
	RestoreDevices(ctx context.Context, ids []uuid.UUID) (int64, error)
	// PermanentDelete permanently removes devices from the database.
	PermanentDelete(ctx context.Context, ids []uuid.UUID) (int64, error)
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
var allowedSortColumns = map[string]bool{
	"created_at":     true,
	"updated_at":     true,
	"serial_number":  true,
	"status":         true,
	"carrier":        true,
	"technology":     true,
	"model":          true,
	"manufacturer":   true,
	"last_inform_at": true,
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
			"carrier", "technology", "status", "firmware_version", "ip_address",
			"connection_request_url", "nat_detected", "udp_connection_request_address",
			"last_inform_at", "last_inform_events",
			"last_boot_at", "boot_count",
			"inform_interval", "site_name", "site_id", "latitude", "longitude",
			"extension_data", "created_at", "updated_at").
		Values(device.ID, device.SerialNumber, device.OUI, device.ProductClass,
			device.Manufacturer, device.ModelName, device.Carrier, device.Technology,
			device.Status, device.FirmwareVersion, ipAddr,
			device.ConnectionRequestURL, device.NatDetected, udpAddr,
			device.LastInformAt, eventsData,
			device.LastBootAt, device.BootCount,
			device.InformInterval, device.SiteName, device.SiteID,
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

	query, args, err := storage.Psql.Update("devices").
		Set("oui", device.OUI).
		Set("product_class", device.ProductClass).
		Set("manufacturer", device.Manufacturer).
		Set("model_name", device.ModelName).
		Set("status", device.Status).
		Set("firmware_version", device.FirmwareVersion).
		Set("ip_address", ipAddr).
		Set("connection_request_url", device.ConnectionRequestURL).
		Set("nat_detected", device.NatDetected).
		Set("udp_connection_request_address", udpAddr).
		Set("last_inform_at", device.LastInformAt).
		Set("last_inform_events", eventsData).
		Set("inform_interval", device.InformInterval).
		Set("site_name", device.SiteName).
		Set("latitude", device.Latitude).
		Set("longitude", device.Longitude).
		Set("extension_data", extData).
		Where(sq.Eq{"id": device.ID}).
		Where(notDeleted). // 防软删 device 被 inform 静默复活；命中时 RowsAffected=0 → 上层 ErrNotFound → 清 cache 走 auto-register
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

func (r *PgDeviceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, _ := storage.Psql.Update("devices").
		Set("deleted_at", time.Now()).
		Where(sq.Eq{"id": id}).
		Where(notDeleted).
		ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("soft delete device: %w", err)
	}
	return nil
}

// BatchDelete soft-deletes multiple devices and removes related device_group_members
// and device_info records within a single transaction.
// Returns the number of devices actually soft-deleted.
func (r *PgDeviceRepository) BatchDelete(ctx context.Context, ids []uuid.UUID, deletedBy string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Remove group memberships
	_, err = tx.Exec(ctx,
		`DELETE FROM device_group_members WHERE device_id = ANY($1)`,
		ids,
	)
	if err != nil {
		return 0, fmt.Errorf("delete device_group_members: %w", err)
	}

	// Remove device_info records
	_, err = tx.Exec(ctx,
		`DELETE FROM device_info WHERE device_id = ANY($1)`,
		ids,
	)
	if err != nil {
		return 0, fmt.Errorf("delete device_info: %w", err)
	}

	// Soft-delete devices with metadata
	now := time.Now()
	tag, err := tx.Exec(ctx,
		`UPDATE devices SET deleted_at = $1, deleted_by = $2 WHERE id = ANY($3) AND deleted_at IS NULL`,
		now, deletedBy, ids,
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
	if len(filter.VisibleGroups) > 0 {
		if filter.GroupID == nil {
			// Only add JOIN if not already added by GroupID
			builder = builder.Join("device_group_members dgm2 ON d.id = dgm2.device_id")
			countBuilder = countBuilder.Join("device_group_members dgm2 ON d.id = dgm2.device_id")
		}
		// Use separate alias if JOIN already exists
		joinAlias := "dgm"
		if filter.GroupID != nil {
			// Already joined with dgm, need subquery or additional condition
			// For simplicity, we use EXISTS subquery for visible groups check
			builder = builder.Where(sq.Eq{"d.id": sq.Select("dgm_vis.device_id").
				From("device_group_members dgm_vis").
				Where(sq.Eq{"dgm_vis.group_id": filter.VisibleGroups})})
			countBuilder = countBuilder.Where(sq.Eq{"d.id": sq.Select("dgm_vis.device_id").
				From("device_group_members dgm_vis").
				Where(sq.Eq{"dgm_vis.group_id": filter.VisibleGroups})})
		} else {
			builder = builder.Where(sq.Eq{joinAlias + ".group_id": filter.VisibleGroups})
			countBuilder = countBuilder.Where(sq.Eq{joinAlias + ".group_id": filter.VisibleGroups})
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
	if filter.Status != nil {
		builder = builder.Where(sq.Eq{"d.status": *filter.Status})
		countBuilder = countBuilder.Where(sq.Eq{"d.status": *filter.Status})
	}
	if filter.OUI != nil {
		builder = builder.Where(sq.Eq{"d.oui": *filter.OUI})
		countBuilder = countBuilder.Where(sq.Eq{"d.oui": *filter.OUI})
	}
	if filter.SN != nil && *filter.SN != "" {
		builder = builder.Where(sq.Eq{"d.serial_number": *filter.SN})
		countBuilder = countBuilder.Where(sq.Eq{"d.serial_number": *filter.SN})
	}
	if filter.Search != nil && *filter.Search != "" {
		like := "%" + *filter.Search + "%"
		cond := sq.Or{sq.ILike{"d.serial_number": like}, sq.ILike{"d.site_name": like}}
		builder = builder.Where(cond)
		countBuilder = countBuilder.Where(cond)
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

func (r *PgDeviceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	query, args, _ := storage.Psql.Update("devices").
		Set("status", status).
		Where(sq.Eq{"id": id}).
		ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device status: %w", err)
	}
	return nil
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

func (r *PgDeviceRepository) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	builder := storage.Psql.Select("d.status", "COUNT(*)").From("devices d").Where(notDeleted).GroupBy("d.status")
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
		var status model.DeviceStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan device count: %w", err)
		}
		result[status] = count
	}
	return result, nil
}

func (r *PgDeviceRepository) ListActiveByLastInform(ctx context.Context, cursorTime *time.Time, cursorID *uuid.UUID, limit int) ([]model.Device, error) {
	builder := storage.Psql.Select(deviceColumns()...).From("devices d").
		Where(sq.Eq{"d.status": model.DeviceActive}).
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
	return []string{
		"d.id", "d.serial_number", "d.oui", "d.product_class", "d.manufacturer", "d.model_name",
		"d.carrier", "d.technology", "d.status", "d.firmware_version",
		"host(d.ip_address) as ip_address", "d.connection_request_url",
		"d.nat_detected", "d.udp_connection_request_address",
		"d.last_inform_at", "d.last_inform_events",
		"d.last_boot_at", "d.boot_count",
		"d.inform_interval", "d.site_name", "d.site_id", "d.latitude", "d.longitude",
		"d.extension_data", "d.created_at", "d.updated_at", "d.deleted_at", "d.deleted_by",
	}
}

// notDeleted is the standard soft-delete filter applied to all read queries.
var notDeleted = sq.Eq{"d.deleted_at": nil}

func scanDeviceFromRow(row pgx.Row) (*model.Device, error) {
	var d model.Device
	var extData, eventsData []byte
	var ipAddr, udpAddr *string
	// nullable string columns from devices table
	var productClass, manufacturer, modelName *string
	var firmwareVersion, connReqURL, siteName, siteID *string
	var deletedBy *string

	err := row.Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &productClass, &manufacturer, &modelName,
		&d.Carrier, &d.Technology, &d.Status, &firmwareVersion,
		&ipAddr, &connReqURL,
		&d.NatDetected, &udpAddr,
		&d.LastInformAt, &eventsData,
		&d.LastBootAt, &d.BootCount,
		&d.InformInterval, &siteName, &siteID, &d.Latitude, &d.Longitude,
		&extData, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt, &deletedBy,
	)
	if err != nil {
		return nil, err
	}

	if deletedBy != nil {
		d.DeletedBy = *deletedBy
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
		d.SiteName = *siteName
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
	var firmwareVersion, connReqURL, siteName, siteID, deletedBy *string

	err := rows.Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &productClass, &manufacturer, &modelName,
		&d.Carrier, &d.Technology, &d.Status, &firmwareVersion,
		&ipAddr, &connReqURL,
		&d.NatDetected, &udpAddr,
		&d.LastInformAt, &eventsData,
		&d.LastBootAt, &d.BootCount,
		&d.InformInterval, &siteName, &siteID, &d.Latitude, &d.Longitude,
		&extData, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt, &deletedBy,
	)

	if deletedBy != nil {
		d.DeletedBy = *deletedBy
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
		d.SiteName = *siteName
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
	return &d, nil
}

// ListGeo returns devices with geographic coordinates for map display.
func (r *PgDeviceRepository) ListGeo(ctx context.Context, filter GeoDeviceFilter) ([]GeoDevice, int64, error) {
	// Build base query with device group join
	builder := storage.Psql.Select(
		"d.id", "d.serial_number", "d.serial_number as name", "d.status",
		"d.latitude", "d.longitude", "dg.id as group_id", "dg.name as group_name",
		"d.site_name as address", "0 as alarm_count", "d.model_name as type",
	).From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		Where(sq.NotEq{"d.latitude": nil}).
		Where(sq.NotEq{"d.longitude": nil})

	// Apply filters
	if len(filter.GroupIDs) > 0 {
		builder = builder.Where(sq.Eq{"dg.id": filter.GroupIDs})
	}
	if len(filter.Status) > 0 {
		builder = builder.Where(sq.Eq{"d.status": filter.Status})
	}
	if filter.Keyword != "" {
		builder = builder.Where(sq.Or{
			sq.ILike{"d.serial_number": "%" + filter.Keyword + "%"},
			sq.ILike{"d.site_name": "%" + filter.Keyword + "%"},
		})
	}

	// Get total count with a separate query
	countBuilder := storage.Psql.Select("COUNT(DISTINCT d.id)").
		From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		Where(sq.NotEq{"d.latitude": nil}).
		Where(sq.NotEq{"d.longitude": nil})

	if len(filter.GroupIDs) > 0 {
		countBuilder = countBuilder.Where(sq.Eq{"dg.id": filter.GroupIDs})
	}
	if len(filter.Status) > 0 {
		countBuilder = countBuilder.Where(sq.Eq{"d.status": filter.Status})
	}
	if filter.Keyword != "" {
		countBuilder = countBuilder.Where(sq.Or{
			sq.ILike{"d.serial_number": "%" + filter.Keyword + "%"},
			sq.ILike{"d.site_name": "%" + filter.Keyword + "%"},
		})
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
		var d GeoDevice
		var groupID *uuid.UUID
		var groupName, address, deviceType *string
		var alarmCount int

		err := rows.Scan(
			&d.ID, &d.SerialNumber, &d.Name, &d.Status,
			&d.Latitude, &d.Longitude, &groupID, &groupName,
			&address, &alarmCount, &deviceType,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan geo device: %w", err)
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

		devices = append(devices, d)
	}

	if devices == nil {
		devices = []GeoDevice{}
	}

	return devices, total, nil
}

// GetGeoStats returns device statistics for map display.
func (r *PgDeviceRepository) GetGeoStats(ctx context.Context, groupIDs []string) (*GeoStats, error) {
	// Build base condition for all queries
	baseCondition := sq.And{
		sq.NotEq{"d.latitude": nil},
		sq.NotEq{"d.longitude": nil},
	}

	// Query 1: Get status counts
	// Use COUNT(DISTINCT d.id) to avoid counting devices multiple times
	// when they belong to multiple groups due to LEFT JOIN
	statusBuilder := storage.Psql.Select("d.status", "COUNT(DISTINCT d.id) as cnt").
		From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		Where(baseCondition).
		GroupBy("d.status")

	if len(groupIDs) > 0 {
		statusBuilder = statusBuilder.Where(sq.Eq{"dg.id": groupIDs})
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
		var status model.DeviceStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan geo stats: %w", err)
		}
		stats.StatusCount[status] = count
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

	if len(groupIDs) > 0 {
		centerBuilder = centerBuilder.Where(sq.Eq{"dg.id": groupIDs})
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

	return stats, nil
}

// SearchDevices searches devices by keyword for map display.
func (r *PgDeviceRepository) SearchDevices(ctx context.Context, keyword string, limit int) ([]GeoDevice, error) {
	if keyword == "" {
		return []GeoDevice{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	builder := storage.Psql.Select(
		"d.id", "d.serial_number", "d.serial_number as name", "d.status",
		"d.latitude", "d.longitude", "dg.id as group_id", "dg.name as group_name",
		"d.site_name as address", "0 as alarm_count", "d.model_name as type",
	).From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dgm.group_id = dg.id").
		Where(sq.Or{
			sq.ILike{"d.serial_number": "%" + keyword + "%"},
			sq.ILike{"d.site_name": "%" + keyword + "%"},
		}).
		Limit(uint64(limit))

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
		var d GeoDevice
		var groupID *uuid.UUID
		var groupName, address, deviceType *string
		var alarmCount int

		err := rows.Scan(
			&d.ID, &d.SerialNumber, &d.Name, &d.Status,
			&d.Latitude, &d.Longitude, &groupID, &groupName,
			&address, &alarmCount, &deviceType,
		)
		if err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
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

		devices = append(devices, d)
	}

	if devices == nil {
		devices = []GeoDevice{}
	}

	return devices, nil
}

// ===== Recycle Bin Operations =====

// recycleBinColumns returns column names for recycle bin query (includes group_name via JOIN).
func recycleBinColumns() []string {
	return append(deviceColumns(), "dg.name as group_name")
}

// scanRecycleBinRow scans a recycle bin device row including group_name.
func scanRecycleBinRow(rows pgx.Rows) (*model.Device, error) {
	var d model.Device
	var extData, eventsData []byte
	var ipAddr, udpAddr *string
	// nullable string columns from devices table
	var productClass, manufacturer, modelName, firmwareVersion, connReqURL, siteName, siteID, deletedBy *string
	// group_name from JOIN
	var groupName *string

	err := rows.Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &productClass, &manufacturer, &modelName,
		&d.Carrier, &d.Technology, &d.Status, &firmwareVersion,
		&ipAddr, &connReqURL,
		&d.NatDetected, &udpAddr,
		&d.LastInformAt, &eventsData,
		&d.InformInterval, &siteName, &siteID, &d.Latitude, &d.Longitude,
		&extData, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt, &deletedBy,
		&groupName,
	)
	if err != nil {
		return nil, err
	}

	// Assign nullable fields
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
		d.SiteName = *siteName
	}
	if siteID != nil {
		d.SiteID = *siteID
	}
	if deletedBy != nil {
		d.DeletedBy = *deletedBy
	}
	if groupName != nil {
		d.GroupName = *groupName
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
	return &d, nil
}

// ListRecycleBin returns soft-deleted devices with filtering.
func (r *PgDeviceRepository) ListRecycleBin(ctx context.Context, filter RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	// Build base query for deleted devices with group info
	builder := storage.Psql.Select(recycleBinColumns()...).
		From("devices d").
		LeftJoin("device_group_members dgm ON d.id = dgm.device_id").
		LeftJoin("device_groups dg ON dg.id = dgm.group_id").
		Where(sq.NotEq{"d.deleted_at": nil})

	countBuilder := storage.Psql.Select("COUNT(*)").
		From("devices d").
		Where(sq.NotEq{"d.deleted_at": nil})

	// Apply filters
	if filter.Search != nil && *filter.Search != "" {
		searchPattern := "%" + strings.ToLower(*filter.Search) + "%"
		searchCond := sq.Or{
			sq.Like{"LOWER(d.serial_number)": searchPattern},
			sq.Like{"LOWER(d.site_name)": searchPattern},
		}
		builder = builder.Where(searchCond)
		countBuilder = countBuilder.Where(searchCond)
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

	// GroupID filter (LEFT JOIN already in main query, just add WHERE)
	if filter.GroupID != nil {
		builder = builder.Where(sq.Eq{"dgm.group_id": *filter.GroupID})
		countBuilder = countBuilder.
			Join("device_group_members dgm ON d.id = dgm.device_id").
			Where(sq.Eq{"dgm.group_id": *filter.GroupID})
	}

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

	var devices []model.Device
	for rows.Next() {
		d, err := scanRecycleBinRow(rows)
        if err != nil {
            return nil, fmt.Errorf("scan recycle bin device: %w", err)
        }
        devices = append(devices, *d)
    }

	if devices == nil {
		devices = []model.Device{}
	}

	return &model.ListResponse[model.Device]{
		Items:    devices,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// RestoreDevices restores soft-deleted devices by setting deleted_at to NULL.
// Returns the number of devices actually restored.
func (r *PgDeviceRepository) RestoreDevices(ctx context.Context, ids []uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE devices SET deleted_at = NULL, deleted_by = '', updated_at = NOW() WHERE id = ANY($1) AND deleted_at IS NOT NULL`,
		ids,
	)
	if err != nil {
		return 0, fmt.Errorf("restore devices: %w", err)
	}

	return tag.RowsAffected(), nil
}

// PermanentDelete permanently removes devices from the database.
// This also removes related device_group_members and device_info records.
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

	// Remove device_info records
	_, err = tx.Exec(ctx,
		`DELETE FROM device_info WHERE device_id = ANY($1)`,
		ids,
	)
	if err != nil {
		return 0, fmt.Errorf("delete device_info: %w", err)
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

// FindStaleDevices finds active devices that haven't sent Inform within the threshold.
// Used by OfflineDetector to mark devices as offline.
func (r *PgDeviceRepository) FindStaleDevices(ctx context.Context, threshold time.Time, limit int) ([]*model.Device, error) {
	builder := storage.Psql.Select(deviceColumns()...).
		From("devices d").
		Where(sq.Eq{"d.status": model.DeviceActive}).
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

// mandatoryProductClasses are always returned even if no devices exist in the database.
var mandatoryProductClasses = []string{"QAFA", "QAFB", "BM", "BNQ", "MLQ", "MLN", "BLQ"}

// ListProductClasses returns distinct product_class values from devices,
// merged with mandatory types, sorted alphabetically.
func (r *PgDeviceRepository) ListProductClasses(ctx context.Context) ([]string, error) {
	query := `SELECT product_class FROM (
		SELECT DISTINCT product_class FROM devices WHERE product_class IS NOT NULL AND product_class != ''
		UNION
		SELECT unnest($1::text[])
	) sub ORDER BY product_class`

	rows, err := r.pool.Query(ctx, query, mandatoryProductClasses)
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
