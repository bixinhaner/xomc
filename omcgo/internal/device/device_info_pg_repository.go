package device

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
)

// allowedSortColumnsWithInfo maps user-facing sort keys to qualified column names
// for the devices + device_info JOIN query.
var allowedSortColumnsWithInfo = map[string]string{
	"created_at":     "d.created_at",
	"updated_at":     "d.updated_at",
	"serial_number":  "d.serial_number",
	"status":         "d.status",
	"carrier":        "d.carrier",
	"technology":     "d.technology",
	"model":          "d.model_name",
	"manufacturer":   "d.manufacturer",
	"last_inform_at": "d.last_inform_at",
	"device_name":    "di.device_name",
	"rf_status":      "di.rf_status",
	"cell_status":    "di.cell_status",
	"bandwidth":      "di.bandwidth",
	"transmit_power": "di.transmit_power",
	"num_of_cells":   "di.num_of_cells",
	"gps_status":     "di.gps_status",
	"alarm_severity": "di.alarm_severity",
	"license_status": "di.license_status",
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
	query, args, err := psql.Select(deviceInfoColumns()...).
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

	query, args, err := psql.Insert("device_info").
		Columns(
			"device_id", "device_name", "address", "remark", "project_status", "height",
			"eci", "pci", "cell_id", "freq_point", "bandwidth", "transmit_power", "plmn",
			"rf_status", "cell_status", "mme_status", "sync_status", "kpi_status",
			"num_of_cells", "gps_status", "alarm_severity", "license_status",
			"mac", "hardware_version",
			"first_online_time", "last_offline_time", "run_time",
			"creator", "updater", "created_at", "updated_at",
		).
		Values(
			info.DeviceID, info.DeviceName, info.Address, info.Remark, info.ProjectStatus, info.Height,
			info.ECI, info.PCI, info.CellID, info.FreqPoint, info.Bandwidth, info.TransmitPower, info.PLMN,
			info.RFStatus, info.CellStatus, info.MMEStatus, info.SyncStatus, info.KPIStatus,
			info.NumOfCells, info.GPSStatus, info.AlarmSeverity, info.LicenseStatus,
			info.MAC, info.HardwareVersion,
			info.FirstOnlineTime, info.LastOfflineTime, info.RunTime,
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
	builder := psql.Update("device_info").Where(sq.Eq{"device_id": deviceID})

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
		return fmt.Errorf("device_info not found for device %s", deviceID)
	}
	return nil
}

func (r *PgDeviceInfoRepository) UpdateSyncFields(ctx context.Context, deviceID uuid.UUID, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	builder := psql.Update("device_info").Where(sq.Eq{"device_id": deviceID})
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

func (r *PgDeviceInfoRepository) ListDevicesWithInfo(ctx context.Context, filter DeviceFilter) (*model.ListResponse[DeviceWithInfo], error) {
	selectCols := deviceWithInfoSelectColumns()
	builder := psql.Select(selectCols...).
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id")
	countBuilder := psql.Select("COUNT(*)").
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id")

	// Apply filters from devices table
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

	// Apply filters from device_info table
	if filter.Manufacturer != nil && *filter.Manufacturer != "" {
		builder = builder.Where(sq.Eq{"d.manufacturer": *filter.Manufacturer})
		countBuilder = countBuilder.Where(sq.Eq{"d.manufacturer": *filter.Manufacturer})
	}
	if filter.ProductClass != nil && *filter.ProductClass != "" {
		builder = builder.Where(sq.Eq{"d.product_class": *filter.ProductClass})
		countBuilder = countBuilder.Where(sq.Eq{"d.product_class": *filter.ProductClass})
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
		builder = builder.Where(sq.Eq{"di.alarm_severity": *filter.AlarmSeverity})
		countBuilder = countBuilder.Where(sq.Eq{"di.alarm_severity": *filter.AlarmSeverity})
	}
	if filter.LicenseStatus != nil && *filter.LicenseStatus != "" {
		builder = builder.Where(sq.Eq{"di.license_status": *filter.LicenseStatus})
		countBuilder = countBuilder.Where(sq.Eq{"di.license_status": *filter.LicenseStatus})
	}

	// Multi-field fuzzy search (G07)
	if filter.Search != nil && *filter.Search != "" {
		keyword := "%" + *filter.Search + "%"
		cond := sq.Or{
			sq.ILike{"d.serial_number": keyword},
			sq.ILike{"d.site_name": keyword},
			sq.ILike{"d.manufacturer": keyword},
			sq.ILike{"d.model_name": keyword},
			sq.ILike{"di.device_name": keyword},
			sq.ILike{"di.address": keyword},
		}
		builder = builder.Where(cond)
		countBuilder = countBuilder.Where(cond)
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
	builder = builder.
		OrderBy(sortCol + " " + sortDir).
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

// deviceInfoColumns returns column names for the device_info table.
func deviceInfoColumns() []string {
	return []string{
		"device_id",
		"device_name", "address", "remark", "project_status", "height",
		"eci", "pci", "cell_id", "freq_point", "bandwidth", "transmit_power", "plmn",
		"rf_status", "cell_status", "mme_status", "sync_status", "kpi_status",
		"num_of_cells", "gps_status", "alarm_severity", "license_status",
		"mac", "hardware_version",
		"first_online_time", "last_offline_time", "run_time",
		"creator", "updater", "created_at", "updated_at",
	}
}

// deviceWithInfoSelectColumns returns qualified column names for the JOIN query.
func deviceWithInfoSelectColumns() []string {
	return []string{
		// devices columns (aliased with d.)
		"d.id", "d.serial_number", "d.oui", "d.product_class", "d.manufacturer", "d.model_name",
		"d.carrier", "d.technology", "d.data_model_id", "d.status", "d.firmware_version",
		"host(d.ip_address) as ip_address", "d.connection_request_url",
		"d.nat_detected", "d.udp_connection_request_address",
		"d.last_inform_at", "d.last_inform_events",
		"d.inform_interval", "d.site_name", "d.site_id", "d.latitude", "d.longitude",
		"d.extension_data", "d.created_at", "d.updated_at",
		// device_info columns
		"di.device_name", "di.address", "di.remark", "di.project_status", "di.height",
		"di.eci", "di.pci", "di.cell_id", "di.freq_point", "di.bandwidth", "di.transmit_power", "di.plmn",
		"di.rf_status", "di.cell_status", "di.mme_status", "di.sync_status", "di.kpi_status",
		"di.num_of_cells", "di.gps_status", "di.alarm_severity", "di.license_status",
		"di.mac", "di.hardware_version",
		"di.first_online_time", "di.last_offline_time", "di.run_time",
	}
}

func scanDeviceInfoFromRow(row pgx.Row) (*DeviceInfo, error) {
	var info DeviceInfo
	err := row.Scan(
		&info.DeviceID,
		&info.DeviceName, &info.Address, &info.Remark, &info.ProjectStatus, &info.Height,
		&info.ECI, &info.PCI, &info.CellID, &info.FreqPoint, &info.Bandwidth, &info.TransmitPower, &info.PLMN,
		&info.RFStatus, &info.CellStatus, &info.MMEStatus, &info.SyncStatus, &info.KPIStatus,
		&info.NumOfCells, &info.GPSStatus, &info.AlarmSeverity, &info.LicenseStatus,
		&info.MAC, &info.HardwareVersion,
		&info.FirstOnlineTime, &info.LastOfflineTime, &info.RunTime,
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
		diLastOffline   *time.Time
		diRunTime       *int64
	)

	err := rows.Scan(
		// devices fields
		&d.ID, &d.SerialNumber, &d.OUI, &d.ProductClass, &d.Manufacturer, &d.ModelName,
		&d.Carrier, &d.Technology, &d.DataModelID, &d.Status, &d.FirmwareVersion,
		&ipAddr, &d.ConnectionRequestURL,
		&d.NatDetected, &udpAddr,
		&d.LastInformAt, &eventsData,
		&d.InformInterval, &d.SiteName, &d.SiteID, &d.Latitude, &d.Longitude,
		&extData, &d.CreatedAt, &d.UpdatedAt,
		// device_info fields (all nullable from LEFT JOIN)
		&diDeviceName, &diAddress, &diRemark, &diProjectStatus, &diHeight,
		&diECI, &diPCI, &diCellID, &diFreqPoint, &diBandwidth, &diTransmitPower, &diPLMN,
		&diRFStatus, &diCellStatus, &diMMEStatus, &diSyncStatus, &diKPIStatus,
		&diNumOfCells, &diGPSStatus, &diAlarmSeverity, &diLicenseStatus,
		&diMAC, &diHWVersion,
		&diFirstOnline, &diLastOffline, &diRunTime,
	)
	if err != nil {
		return nil, fmt.Errorf("scan device with info: %w", err)
	}

	// Assign nullable device fields
	if ipAddr != nil {
		d.IPAddress = *ipAddr
	}
	if udpAddr != nil {
		d.UDPConnectionRequestAddress = *udpAddr
	}
	if len(extData) > 0 {
		// ignore unmarshal error for list view
		_ = json.Unmarshal(extData, &d.ExtensionData)
	}
	if len(eventsData) > 0 {
		_ = json.Unmarshal(eventsData, &d.LastInformEvents)
	}

	// Assign device_info fields
	d.DeviceName = diDeviceName
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
	d.AlarmSeverity = diAlarmSeverity
	d.LicenseStatus = diLicenseStatus
	d.MAC = diMAC
	d.HardwareVersion = diHWVersion
	d.FirstOnlineTime = diFirstOnline
	d.LastOfflineTime = diLastOffline
	d.RunTime = diRunTime

	return &d, nil
}
