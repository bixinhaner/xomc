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
	"github.com/omcgo/omcgo/internal/common/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

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

	extData, _ := json.Marshal(device.ExtensionData)
	eventsData, _ := json.Marshal(device.LastInformEvents)

	query, args, err := psql.Insert("devices").
		Columns("id", "serial_number", "oui", "product_class", "manufacturer", "model_name",
			"carrier", "technology", "status", "firmware_version", "ip_address",
			"connection_request_url", "last_inform_at", "last_inform_events",
			"inform_interval", "site_name", "site_id", "latitude", "longitude",
			"extension_data", "created_at", "updated_at").
		Values(device.ID, device.SerialNumber, device.OUI, device.ProductClass,
			device.Manufacturer, device.ModelName, device.Carrier, device.Technology,
			device.Status, device.FirmwareVersion, device.IPAddress,
			device.ConnectionRequestURL, device.LastInformAt, eventsData,
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
	query, args, err := psql.Select(deviceColumns()...).
		From("devices").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return r.scanDevice(ctx, query, args...)
}

func (r *PgDeviceRepository) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	query, args, err := psql.Select(deviceColumns()...).
		From("devices").
		Where(sq.Eq{"serial_number": sn}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return r.scanDevice(ctx, query, args...)
}

func (r *PgDeviceRepository) Update(ctx context.Context, device *model.Device) error {
	extData, _ := json.Marshal(device.ExtensionData)
	eventsData, _ := json.Marshal(device.LastInformEvents)

	query, args, err := psql.Update("devices").
		Set("oui", device.OUI).
		Set("product_class", device.ProductClass).
		Set("manufacturer", device.Manufacturer).
		Set("model_name", device.ModelName).
		Set("status", device.Status).
		Set("firmware_version", device.FirmwareVersion).
		Set("ip_address", device.IPAddress).
		Set("connection_request_url", device.ConnectionRequestURL).
		Set("last_inform_at", device.LastInformAt).
		Set("last_inform_events", eventsData).
		Set("inform_interval", device.InformInterval).
		Set("site_name", device.SiteName).
		Set("latitude", device.Latitude).
		Set("longitude", device.Longitude).
		Set("extension_data", extData).
		Where(sq.Eq{"id": device.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PgDeviceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, _ := psql.Delete("devices").Where(sq.Eq{"id": id}).ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PgDeviceRepository) List(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error) {
	builder := psql.Select(deviceColumns()...).From("devices")
	countBuilder := psql.Select("COUNT(*)").From("devices")

	if filter.Carrier != nil {
		builder = builder.Where(sq.Eq{"carrier": *filter.Carrier})
		countBuilder = countBuilder.Where(sq.Eq{"carrier": *filter.Carrier})
	}
	if filter.Technology != nil {
		builder = builder.Where(sq.Eq{"technology": *filter.Technology})
		countBuilder = countBuilder.Where(sq.Eq{"technology": *filter.Technology})
	}
	if filter.Status != nil {
		builder = builder.Where(sq.Eq{"status": *filter.Status})
		countBuilder = countBuilder.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.OUI != nil {
		builder = builder.Where(sq.Eq{"oui": *filter.OUI})
		countBuilder = countBuilder.Where(sq.Eq{"oui": *filter.OUI})
	}
	if filter.Search != nil && *filter.Search != "" {
		like := "%" + *filter.Search + "%"
		cond := sq.Or{sq.ILike{"serial_number": like}, sq.ILike{"site_name": like}}
		builder = builder.Where(cond)
		countBuilder = countBuilder.Where(cond)
	}

	// Count total
	countQuery, countArgs, _ := countBuilder.ToSql()
	var total int64
	r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)

	// Apply pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
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
	query, args, _ := psql.Update("devices").
		Set("status", status).
		Where(sq.Eq{"id": id}).
		ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PgDeviceRepository) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	eventsData, _ := json.Marshal(events)
	query, args, _ := psql.Update("devices").
		Set("last_inform_at", at).
		Set("last_inform_events", eventsData).
		Where(sq.Eq{"serial_number": sn}).
		ToSql()
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PgDeviceRepository) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	builder := psql.Select("status", "COUNT(*)").From("devices").GroupBy("status")
	if carrier != nil {
		builder = builder.Where(sq.Eq{"carrier": *carrier})
	}
	query, args, _ := builder.ToSql()

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[model.DeviceStatus]int64)
	for rows.Next() {
		var status model.DeviceStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}
	return result, nil
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
	return []string{
		"id", "serial_number", "oui", "product_class", "manufacturer", "model_name",
		"carrier", "technology", "data_model_id", "status", "firmware_version",
		"ip_address", "connection_request_url", "last_inform_at", "last_inform_events",
		"inform_interval", "site_name", "site_id", "latitude", "longitude",
		"extension_data", "created_at", "updated_at",
	}
}

func scanDeviceFromRow(row pgx.Row) (*model.Device, error) {
	var d model.Device
	var extData, eventsData []byte
	var ipAddr *string

	err := row.Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &d.ProductClass, &d.Manufacturer, &d.ModelName,
		&d.Carrier, &d.Technology, &d.DataModelID, &d.Status, &d.FirmwareVersion,
		&ipAddr, &d.ConnectionRequestURL, &d.LastInformAt, &eventsData,
		&d.InformInterval, &d.SiteName, &d.SiteID, &d.Latitude, &d.Longitude,
		&extData, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if ipAddr != nil {
		d.IPAddress = *ipAddr
	}
	if len(extData) > 0 {
		json.Unmarshal(extData, &d.ExtensionData)
	}
	if len(eventsData) > 0 {
		json.Unmarshal(eventsData, &d.LastInformEvents)
	}
	return &d, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanDeviceRow(rows pgx.Rows) (*model.Device, error) {
	var d model.Device
	var extData, eventsData []byte
	var ipAddr *string

	err := rows.Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &d.ProductClass, &d.Manufacturer, &d.ModelName,
		&d.Carrier, &d.Technology, &d.DataModelID, &d.Status, &d.FirmwareVersion,
		&ipAddr, &d.ConnectionRequestURL, &d.LastInformAt, &eventsData,
		&d.InformInterval, &d.SiteName, &d.SiteID, &d.Latitude, &d.Longitude,
		&extData, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if ipAddr != nil {
		d.IPAddress = *ipAddr
	}
	if len(extData) > 0 {
		json.Unmarshal(extData, &d.ExtensionData)
	}
	if len(eventsData) > 0 {
		json.Unmarshal(eventsData, &d.LastInformEvents)
	}
	return &d, nil
}
