package pageconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type PgRepository struct {
	pool   *pgxpool.Pool
	tsPool *pgxpool.Pool

	// defaultsSeeded / extDefaultsSeeded gate the idempotent default-seeding so
	// the catalog defaults are written exactly once per process lifetime (see
	// EnsureDefaults / EnsureExtendedDefaults). Seeding was previously re-run on
	// every request — 21 + ~9 sequential round-trips each — which dominated the
	// latency of every read/write on this repository.
	defaultsSeeded    onceSuccess
	extDefaultsSeeded onceSuccess
}

var _ Repository = (*PgRepository)(nil)

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// onceSuccess runs f exactly once on success. Failures are not sticky: a later
// call retries f, so a transient DB error (e.g. a request-scoped context
// cancellation during the very first seed) does not permanently disable
// default seeding for the lifetime of the process.
type onceSuccess struct {
	done atomic.Bool
	mu   sync.Mutex
}

func (o *onceSuccess) Do(f func() error) error {
	if o.done.Load() {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done.Load() {
		return nil
	}
	if err := f(); err != nil {
		return err
	}
	o.done.Store(true)
	return nil
}

func (r *PgRepository) WithTsPool(tsPool *pgxpool.Pool) *PgRepository {
	r.tsPool = tsPool
	return r
}

func (r *PgRepository) EnsureDefaults(ctx context.Context, catalog *Catalog) error {
	return r.defaultsSeeded.Do(func() error {
		return r.seedDefaults(ctx, catalog)
	})
}

func (r *PgRepository) seedDefaults(ctx context.Context, catalog *Catalog) error {
	if catalog == nil {
		catalog = NewDefaultCatalog()
	}
	for _, profile := range catalog.FileProfiles() {
		if err := r.insertDefaultFileProfile(ctx, profile); err != nil {
			return err
		}
		if err := r.backfillDefaultFileProfileGroupMetadata(ctx, profile); err != nil {
			return err
		}
	}
	for _, profile := range catalog.InventoryProfiles() {
		if err := r.insertDefaultInventoryProfile(ctx, profile); err != nil {
			return err
		}
	}
	return nil
}

func (r *PgRepository) backfillDefaultFileProfileGroupMetadata(ctx context.Context, profile FileProfile) error {
	defaultGroupsByID := make(map[string]FileGroup, len(profile.Groups))
	for _, group := range profile.Groups {
		defaultGroupsByID[group.ID] = group
	}
	if len(defaultGroupsByID) == 0 {
		return nil
	}

	var raw []byte
	if err := r.pool.QueryRow(ctx, `SELECT groups FROM northbound_file_profiles WHERE code = $1`, profile.Code).Scan(&raw); err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return fmt.Errorf("query default northbound_file_profiles groups %s: %w", profile.Code, err)
	}
	var groups []FileGroup
	if err := json.Unmarshal(raw, &groups); err != nil {
		return fmt.Errorf("unmarshal default northbound_file_profiles groups %s: %w", profile.Code, err)
	}
	changed := false
	for i := range groups {
		defaultGroup, ok := defaultGroupsByID[groups[i].ID]
		if !ok {
			continue
		}
		if strings.TrimSpace(groups[i].CSVSeparator) == "" && strings.TrimSpace(defaultGroup.CSVSeparator) != "" {
			groups[i].CSVSeparator = defaultGroup.CSVSeparator
			changed = true
		}
	}
	if !changed {
		return nil
	}
	updated, err := json.Marshal(groups)
	if err != nil {
		return fmt.Errorf("marshal default northbound_file_profiles groups %s: %w", profile.Code, err)
	}
	_, err = r.pool.Exec(ctx, `UPDATE northbound_file_profiles SET groups = $2, updated_at = now() WHERE code = $1`, profile.Code, updated)
	if err != nil {
		return fmt.Errorf("backfill default northbound_file_profiles groups %s: %w", profile.Code, err)
	}
	return nil
}

func (r *PgRepository) insertDefaultFileProfile(ctx context.Context, profile FileProfile) error {
	flags, err := json.Marshal(profile.Flags)
	if err != nil {
		return fmt.Errorf("marshal northbound file profile flags: %w", err)
	}
	groups, err := json.Marshal(profile.Groups)
	if err != nil {
		return fmt.Errorf("marshal northbound file profile groups: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
INSERT INTO northbound_file_profiles (
  code, name, vendor, scenario_name, scenario_name_en, description,
  flags, enabled, status, groups
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (code) DO NOTHING`,
		profile.Code, profile.Name, profile.Vendor, profile.ScenarioName, profile.ScenarioNameEn,
		profile.Description, flags, profile.Enabled, profile.Status, groups)
	if err != nil {
		return fmt.Errorf("insert default northbound_file_profiles %s: %w", profile.Code, err)
	}
	return nil
}

func (r *PgRepository) insertDefaultInventoryProfile(ctx context.Context, profile InventoryProfile) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO northbound_inventory_profiles (
  code, name, object_code, tech, period, start_minute, path_template,
  file_name_template, compression_enabled, compression_format, enabled, status
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (code) DO NOTHING`,
		profile.Code, profile.Name, profile.ObjectCode, profile.Tech, profile.Period, profile.StartMinute,
		profile.PathTemplate, profile.FileNameTemplate, profile.CompressionEnabled, profile.CompressionFormat,
		profile.Enabled, profile.Status)
	if err != nil {
		return fmt.Errorf("insert default northbound_inventory_profiles %s: %w", profile.Code, err)
	}
	return nil
}

func (r *PgRepository) ListFileProfiles(ctx context.Context) ([]FileProfile, error) {
	rows, err := r.pool.Query(ctx, `
SELECT code, name, vendor, scenario_name, scenario_name_en, description,
       flags, enabled, status, groups
  FROM northbound_file_profiles
 ORDER BY code ASC`)
	if err != nil {
		return nil, fmt.Errorf("query northbound_file_profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]FileProfile, 0)
	for rows.Next() {
		profile, err := scanFileProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, *profile)
	}
	return profiles, rows.Err()
}

func (r *PgRepository) CreateFileProfile(ctx context.Context, profile FileProfile) (*FileProfile, error) {
	flags, err := json.Marshal(profile.Flags)
	if err != nil {
		return nil, fmt.Errorf("marshal northbound file profile flags: %w", err)
	}
	groups, err := json.Marshal(profile.Groups)
	if err != nil {
		return nil, fmt.Errorf("marshal northbound file profile groups: %w", err)
	}

	row := r.pool.QueryRow(ctx, `
INSERT INTO northbound_file_profiles (
  code, name, vendor, scenario_name, scenario_name_en, description,
  flags, enabled, status, groups
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING code, name, vendor, scenario_name, scenario_name_en, description,
          flags, enabled, status, groups`,
		profile.Code, profile.Name, profile.Vendor, profile.ScenarioName, profile.ScenarioNameEn,
		profile.Description, flags, profile.Enabled, profile.Status, groups)
	created, err := scanFileProfile(row)
	if err != nil {
		if strings.Contains(err.Error(), "northbound_file_profiles_code_key") {
			return nil, fmt.Errorf("%w: file profile code %s already exists", commonerrors.ErrInvalidInput, profile.Code)
		}
		if strings.Contains(err.Error(), "northbound_file_profiles_code_check") {
			return nil, fmt.Errorf("%w: file profile code must match S0000 format", commonerrors.ErrInvalidInput)
		}
		return nil, err
	}
	return created, nil
}

func (r *PgRepository) UpdateFileProfile(ctx context.Context, idOrCode string, req UpdateFileProfileRequest) (*FileProfile, error) {
	current, err := r.getFileProfile(ctx, idOrCode)
	if err != nil {
		return nil, err
	}
	merged := mergeFileProfile(*current, req)
	flags, err := json.Marshal(merged.Flags)
	if err != nil {
		return nil, fmt.Errorf("marshal northbound file profile flags: %w", err)
	}
	groups, err := json.Marshal(merged.Groups)
	if err != nil {
		return nil, fmt.Errorf("marshal northbound file profile groups: %w", err)
	}

	row := r.pool.QueryRow(ctx, `
UPDATE northbound_file_profiles
   SET name=$2, vendor=$3, scenario_name=$4, scenario_name_en=$5, description=$6,
       flags=$7, enabled=$8, status=$9, groups=$10, version=version+1
 WHERE code=$1 OR id::text=$1
 RETURNING code, name, vendor, scenario_name, scenario_name_en, description,
           flags, enabled, status, groups`,
		idOrCode, merged.Name, merged.Vendor, merged.ScenarioName, merged.ScenarioNameEn, merged.Description,
		flags, merged.Enabled, merged.Status, groups)
	return scanFileProfile(row)
}

func (r *PgRepository) ListInventoryProfiles(ctx context.Context) ([]InventoryProfile, error) {
	rows, err := r.pool.Query(ctx, `
SELECT code, name, object_code, tech, period, start_minute, path_template,
       file_name_template, compression_enabled, compression_format, enabled, status, config
  FROM northbound_inventory_profiles
 ORDER BY code ASC`)
	if err != nil {
		return nil, fmt.Errorf("query northbound_inventory_profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]InventoryProfile, 0)
	for rows.Next() {
		profile, err := scanInventoryProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, *profile)
	}
	return profiles, rows.Err()
}

func (r *PgRepository) UpdateInventoryProfile(ctx context.Context, idOrCode string, req UpdateInventoryProfileRequest) (*InventoryProfile, error) {
	current, err := r.getInventoryProfile(ctx, idOrCode)
	if err != nil {
		return nil, err
	}
	merged := mergeInventoryProfile(*current, req)
	config, err := marshalInventoryProfileConfig(merged.Fields)
	if err != nil {
		return nil, fmt.Errorf("marshal northbound inventory profile config: %w", err)
	}

	row := r.pool.QueryRow(ctx, `
UPDATE northbound_inventory_profiles
   SET name=$2, object_code=$3, tech=$4, period=$5, start_minute=$6,
       path_template=$7, file_name_template=$8, compression_enabled=$9,
       compression_format=$10, enabled=$11, status=$12, config=$13, version=version+1
 WHERE code=$1 OR id::text=$1
 RETURNING code, name, object_code, tech, period, start_minute, path_template,
           file_name_template, compression_enabled, compression_format, enabled, status, config`,
		idOrCode, merged.Name, merged.ObjectCode, merged.Tech, merged.Period, merged.StartMinute,
		merged.PathTemplate, merged.FileNameTemplate, merged.CompressionEnabled, merged.CompressionFormat,
		merged.Enabled, merged.Status, config)
	return scanInventoryProfile(row)
}

func (r *PgRepository) CreateFileRun(ctx context.Context, run FileRun) (*FileRun, error) {
	summary, err := json.Marshal(run.Summary)
	if err != nil {
		return nil, fmt.Errorf("marshal northbound file run summary: %w", err)
	}
	row := r.pool.QueryRow(ctx, `
INSERT INTO northbound_file_runs (
  profile_kind, profile_code, group_id, domain, object_code, status,
  window_start, window_end, artifact_path, artifact_name, artifact_content,
  artifact_size, row_count, compression_enabled, compression_format,
  error_message, summary
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17
)
RETURNING id::text, profile_kind, profile_code, group_id, domain, object_code, status,
          window_start, window_end, artifact_path, artifact_name, artifact_content,
          artifact_size, row_count, compression_enabled, COALESCE(compression_format, ''),
          error_message, summary, created_at, updated_at`,
		run.ProfileKind, run.ProfileCode, run.GroupID, run.Domain, run.ObjectCode, run.Status,
		run.WindowStart, run.WindowEnd, run.ArtifactPath, run.ArtifactName, run.ArtifactContent,
		run.ArtifactSize, run.RowCount, run.CompressionEnabled, run.CompressionFormat,
		run.ErrorMessage, summary)
	return scanFileRun(row)
}

func (r *PgRepository) ListFileRuns(ctx context.Context, filter RunFilter) (RunListResult, error) {
	args := make([]any, 0, 4)
	where := []string{"true"}
	if filter.ProfileKind != "" {
		args = append(args, filter.ProfileKind)
		where = append(where, fmt.Sprintf("profile_kind = $%d", len(args)))
	}
	if strings.TrimSpace(filter.ProfileCode) != "" {
		args = append(args, filter.ProfileCode)
		where = append(where, fmt.Sprintf("profile_code = $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	whereClause := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`
SELECT COUNT(*)
  FROM northbound_file_runs
 WHERE %s`, whereClause), args...).Scan(&total); err != nil {
		return RunListResult{}, fmt.Errorf("count northbound_file_runs: %w", err)
	}
	limit := normalizeLimit(filter.Limit)
	offset := normalizeOffset(filter.Offset)
	args = append(args, limit, offset)
	query := fmt.Sprintf(`
SELECT id::text, profile_kind, profile_code, group_id, domain, object_code, status,
       window_start, window_end, artifact_path, artifact_name, ''::text AS artifact_content,
       artifact_size, row_count, compression_enabled, COALESCE(compression_format, ''),
       error_message, summary, created_at, updated_at
  FROM northbound_file_runs
 WHERE %s
 ORDER BY created_at DESC
 LIMIT $%d OFFSET $%d`, whereClause, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return RunListResult{}, fmt.Errorf("query northbound_file_runs: %w", err)
	}
	defer rows.Close()
	items, err := scanFileRuns(rows)
	if err != nil {
		return RunListResult{}, err
	}
	return RunListResult{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (r *PgRepository) GetFileRun(ctx context.Context, id string) (*FileRun, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id::text, profile_kind, profile_code, group_id, domain, object_code, status,
       window_start, window_end, artifact_path, artifact_name, artifact_content,
       artifact_size, row_count, compression_enabled, COALESCE(compression_format, ''),
       error_message, summary, created_at, updated_at
  FROM northbound_file_runs
 WHERE id::text = $1`, id)
	return scanFileRun(row)
}

func (r *PgRepository) UpdateFileRunLocalArchive(ctx context.Context, id string, result LocalArchiveResult, clearContent bool) (*FileRun, error) {
	patch, err := json.Marshal(map[string]any{
		"local_archive": map[string]any{
			"bucket":       result.Bucket,
			"object_key":   result.ObjectKey,
			"object_name":  result.ObjectName,
			"bytes":        result.Bytes,
			"content_type": result.ContentType,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal northbound local archive summary: %w", err)
	}
	row := r.pool.QueryRow(ctx, `
UPDATE northbound_file_runs
   SET artifact_content = CASE WHEN $3 THEN '' ELSE artifact_content END,
       summary = COALESCE(summary, '{}'::jsonb) || $2::jsonb
 WHERE id::text = $1
 RETURNING id::text, profile_kind, profile_code, group_id, domain, object_code, status,
           window_start, window_end, artifact_path, artifact_name, artifact_content,
           artifact_size, row_count, compression_enabled, COALESCE(compression_format, ''),
           error_message, summary, created_at, updated_at`, id, patch, clearContent)
	return scanFileRun(row)
}

func (r *PgRepository) LoadDeviceSnapshotRows(ctx context.Context, tech string, limit int) ([]ExportDataRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT
  COALESCE(d.serial_number, ''),
  COALESCE(d.manufacturer, ''),
  COALESCE(d.model_name, ''),
  COALESCE(d.product_class, ''),
  COALESCE(d.firmware_version, ''),
  COALESCE(d.ip_address::text, ''),
  COALESCE(d.site_name, ''),
  COALESCE(d.site_id, ''),
  COALESCE((
    SELECT string_agg(DISTINCT dg.name, ',' ORDER BY dg.name)
      FROM device_group_members dgm
      JOIN device_groups dg ON dg.id = dgm.group_id
     WHERE dgm.device_id = d.id
  ), ''),
  COALESCE((SELECT p.product_name FROM products p WHERE p.id = d.product_id), ''),
  COALESCE(d.last_inform_at::text, ''),
  COALESCE(d.is_online::text, ''),
  COALESCE(d.longitude::text, ''),
  COALESCE(d.latitude::text, ''),
  COALESCE(d.lifecycle_state, ''),
  COALESCE(d.created_at::text, ''),
  COALESCE(di.device_name, ''),
  COALESCE(di.address, ''),
  COALESCE(di.hardware_version, ''),
  COALESCE(di.mac, ''),
  COALESCE(di.plmn, ''),
  COALESCE(di.sync_status, ''),
  COALESCE(di.enb_id, ''),
  COALESCE(di.eci, ''),
  COALESCE(di.pci, ''),
  COALESCE(di.cell_id, ''),
  COALESCE(di.freq_point, ''),
  COALESCE(di.bandwidth::text, ''),
  COALESCE(di.transmit_power::text, ''),
  COALESCE(di.op_state, ''),
  COALESCE(di.ue_count::text, ''),
  COALESCE(di.rf_status, ''),
  COALESCE(di.kpi_status, ''),
  COALESCE(di.num_of_cells::text, ''),
  COALESCE(di.active_alarm_count::text, ''),
  COALESCE(di.highest_alarm_severity::text, ''),
  COALESCE(di.cumulative_online_duration::text, ''),
  COALESCE(di.mme_status, ''),
  COALESCE(di.tac, ''),
  COALESCE(di.band, ''),
  COALESCE(di.ul_earfcn, ''),
  COALESCE(di.subframe_assignment, ''),
  COALESCE(di.special_subframe, ''),
  COALESCE(di.root_index, ''),
  COALESCE(di.gps_satellites::text, ''),
  COALESCE(di.gps_height::text, ''),
  COALESCE(di.ipsec_addr, ''),
  COALESCE(di.lac, ''),
  COALESCE(di.run_time::text, ''),
  COALESCE(di.first_online_time::text, '')
FROM devices d
LEFT JOIN device_info di ON di.device_id = d.id
WHERE d.deleted_at IS NULL
  AND ($1 = '' OR UPPER(d.technology) = UPPER($1))
ORDER BY d.serial_number ASC
LIMIT $2`, normalizeDeviceTech(tech), normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("query device snapshot rows: %w", err)
	}
	defer rows.Close()

	out := make([]ExportDataRow, 0)
	for rows.Next() {
		var serialNumber, manufacturer, modelName, productClass, firmwareVersion string
		var ipAddress, siteName, siteID, deviceGroup, productName, lastInformAt, isOnline, longitude, latitude, lifecycleState, createdAt string
		var deviceName, address, hardwareVersion, mac, plmn, syncStatus, enbID, eci, pci, cellID, freqPoint string
		var bandwidth, transmitPower, opState, ueCount, rfStatus, kpiStatus, numOfCells string
		var activeAlarmCount, highestAlarmSeverity, cumulativeOnlineDuration string
		var mmeStatus, tac, band, ulEarfcn, subframeAssignment, specialSubframe, rootIndex string
		var gpsSatellites, gpsHeight, ipsecAddr, lac, runTime, firstOnlineTime string
		if err := rows.Scan(
			&serialNumber, &manufacturer, &modelName, &productClass, &firmwareVersion,
			&ipAddress, &siteName, &siteID, &deviceGroup, &productName, &lastInformAt, &isOnline, &longitude, &latitude,
			&lifecycleState, &createdAt,
			&deviceName, &address, &hardwareVersion, &mac, &plmn, &syncStatus, &enbID, &eci, &pci, &cellID, &freqPoint,
			&bandwidth, &transmitPower, &opState, &ueCount, &rfStatus, &kpiStatus, &numOfCells,
			&activeAlarmCount, &highestAlarmSeverity, &cumulativeOnlineDuration,
			&mmeStatus, &tac, &band, &ulEarfcn, &subframeAssignment, &specialSubframe, &rootIndex,
			&gpsSatellites, &gpsHeight, &ipsecAddr, &lac, &runTime, &firstOnlineTime,
		); err != nil {
			return nil, fmt.Errorf("scan device snapshot row: %w", err)
		}
		out = append(out, ExportDataRow{
			"device.serial_number":                   serialNumber,
			"device.manufacturer":                    manufacturer,
			"device.model_name":                      modelName,
			"device.product_class":                   productClass,
			"device.firmware_version":                firmwareVersion,
			"device.ip_address":                      ipAddress,
			"device.site_name":                       siteName,
			"device.site_id":                         siteID,
			"device_groups.name":                     deviceGroup,
			"product.name":                           productName,
			"device.last_inform_at":                  lastInformAt,
			"device.is_online":                       isOnline,
			"device.longitude":                       longitude,
			"device.latitude":                        latitude,
			"device.lifecycle_state":                 lifecycleState,
			"device.created_at":                      createdAt,
			"device_info.device_name":                deviceName,
			"device_info.address":                    address,
			"device_info.hardware_version":           hardwareVersion,
			"device_info.mac":                        mac,
			"device_info.plmn":                       plmn,
			"device_info.sync_status":                syncStatus,
			"device_info.enb_id":                     enbID,
			"device_info.eci":                        eci,
			"device_info.pci":                        pci,
			"device_info.cell_id":                    cellID,
			"device_info.freq_point":                 freqPoint,
			"device_info.bandwidth":                  bandwidth,
			"device_info.transmit_power":             transmitPower,
			"device_info.op_state":                   opState,
			"device_info.ue_count":                   ueCount,
			"device_info.rf_status":                  rfStatus,
			"device_info.kpi_status":                 kpiStatus,
			"device_info.num_of_cells":               numOfCells,
			"device_info.active_alarm_count":         activeAlarmCount,
			"device_info.highest_alarm_severity":     highestAlarmSeverity,
			"device_info.cumulative_online_duration": cumulativeOnlineDuration,
			"device_info.mme_status":                 mmeStatus,
			"device_info.tac":                        tac,
			"device_info.band":                       band,
			"device_info.ul_earfcn":                  ulEarfcn,
			"device_info.subframe_assignment":        subframeAssignment,
			"device_info.special_subframe":           specialSubframe,
			"device_info.root_index":                 rootIndex,
			"device_info.gps_satellites":             gpsSatellites,
			"device_info.gps_height":                 gpsHeight,
			"device_info.ipsec_addr":                 ipsecAddr,
			"device_info.lac":                        lac,
			"device_info.run_time":                   runTime,
			"device_info.first_online_time":          firstOnlineTime,
			"inventory.enb.snapshot_time":            time.Now().Format(time.RFC3339),
			"inventory.gnb.snapshot_time":            time.Now().Format(time.RFC3339),
			"inventory.gsm.snapshot_time":            time.Now().Format(time.RFC3339),
			"inventory.omc.snapshot_time":            time.Now().Format(time.RFC3339),
			"inventory.enb.serial_number":            serialNumber,
			"inventory.gnb.serial_number":            serialNumber,
			"inventory.gsm.serial_number":            serialNumber,
			"inventory.enb.cell_status":              opState,
			"inventory.gnb.cell_status":              opState,
			"inventory.gsm.cell_status":              opState,
			"inventory.enb.online_status":            isOnline,
			"inventory.gnb.online_status":            isOnline,
			"inventory.gsm.online_status":            isOnline,
			"inventory.enb.ip_address":               ipAddress,
			"inventory.gnb.ip_address":               ipAddress,
			"inventory.gsm.ip_address":               ipAddress,
			"inventory.enb.product_type":             productClass,
			"inventory.gnb.product_type":             productClass,
			"inventory.gsm.product_type":             productClass,
		})
	}
	return out, rows.Err()
}

func (r *PgRepository) LoadOMCInventoryRows(ctx context.Context) ([]ExportDataRow, error) {
	row := r.pool.QueryRow(ctx, `
SELECT
  COUNT(*) FILTER (WHERE UPPER(d.technology) = 'LTE' AND d.is_online)::text,
  COUNT(*) FILTER (WHERE UPPER(d.technology) = 'LTE' AND COALESCE(di.op_state, '') = '1')::text,
  COALESCE(string_agg(DISTINCT NULLIF(di.mme_status, ''), ',' ORDER BY NULLIF(di.mme_status, '')), ''),
  COALESCE(SUM(COALESCE(di.ue_count, 0)), 0)::text
FROM devices d
LEFT JOIN device_info di ON di.device_id = d.id
WHERE d.deleted_at IS NULL`)
	var enbOnline, enbActive, mmeStatus, ueCount string
	if err := row.Scan(&enbOnline, &enbActive, &mmeStatus, &ueCount); err != nil {
		return nil, fmt.Errorf("query omc inventory rows: %w", err)
	}
	return []ExportDataRow{{
		"inventory.omc.enb_online":    enbOnline,
		"inventory.omc.enb_active":    enbActive,
		"inventory.omc.mme_status":    mmeStatus,
		"inventory.omc.ue_count":      ueCount,
		"inventory.omc.version":       "xomc",
		"inventory.omc.snapshot_time": time.Now().Format(time.RFC3339),
	}}, nil
}

func (r *PgRepository) LoadPMMetricRows(ctx context.Context, req PMMetricQuery) ([]ExportDataRow, error) {
	if r.tsPool == nil {
		return nil, fmt.Errorf("timescale database is not configured for PM northbound export")
	}
	args := make([]any, 0, 5)
	where := []string{"true"}
	if len(req.MetricPaths) > 0 {
		args = append(args, req.MetricPaths)
		where = append(where, fmt.Sprintf("metric_path = ANY($%d::text[])", len(args)))
	}
	if tech := normalizeTech(req.Tech); tech != "" {
		args = append(args, tech)
		where = append(where, fmt.Sprintf("UPPER(COALESCE(extra->>'technology', '')) = UPPER($%d)", len(args)))
	}
	if req.WindowStart != nil {
		args = append(args, *req.WindowStart)
		where = append(where, fmt.Sprintf("end_time >= $%d", len(args)))
	}
	if req.WindowEnd != nil {
		args = append(args, *req.WindowEnd)
		where = append(where, fmt.Sprintf("end_time <= $%d", len(args)))
	}
	args = append(args, normalizeLimit(req.Limit))
	query := fmt.Sprintf(`
SELECT
  COALESCE(device_sn, ''),
  COALESCE(metric_path, ''),
  COALESCE(metric_type, ''),
  COALESCE(metric_value::text, ''),
  COALESCE(statis_type, ''),
  COALESCE(granularity, ''),
  COALESCE(end_time::text, ''),
  COALESCE(object_ldn, '')
FROM pm_metrics
WHERE %s
ORDER BY end_time DESC
LIMIT $%d`, strings.Join(where, " AND "), len(args))

	rows, err := r.tsPool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pm_metrics rows: %w", err)
	}
	defer rows.Close()

	out := make([]ExportDataRow, 0)
	for rows.Next() {
		var deviceSN, metricPath, metricType, metricValue, statisType, granularity, endTime, objectLDN string
		if err := rows.Scan(&deviceSN, &metricPath, &metricType, &metricValue, &statisType, &granularity, &endTime, &objectLDN); err != nil {
			return nil, fmt.Errorf("scan pm_metrics row: %w", err)
		}
		out = append(out, ExportDataRow{
			"pm.device_sn":    deviceSN,
			"pm.metric_path":  metricPath,
			"pm.metric_type":  metricType,
			"pm.metric_value": metricValue,
			"pm.statis_type":  statisType,
			"pm.granularity":  granularity,
			"pm.end_time":     endTime,
			"pm.object_ldn":   objectLDN,
		})
	}
	return out, rows.Err()
}

func (r *PgRepository) ListPMMetricFields(ctx context.Context, filter FieldFilter) ([]FieldDefinition, error) {
	tables := pmIndicatorTables(filter.Tech)
	out := make([]FieldDefinition, 0)
	for _, table := range tables {
		rows, err := r.pool.Query(ctx, table.query)
		if err != nil {
			return nil, fmt.Errorf("query %s PM indicators: %w", table.name, err)
		}
		for rows.Next() {
			var id, reportKey, enName, cnName, unit, statisType, isCounter, dataType, productTypes string
			if err := rows.Scan(&id, &reportKey, &enName, &cnName, &unit, &statisType, &isCounter, &dataType, &productTypes); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan %s PM indicator: %w", table.name, err)
			}
			objectCode := "PE"
			metricType := "kpi"
			if strings.TrimSpace(isCounter) == "1" {
				objectCode = "PC"
				metricType = "counter"
			}
			if filter.ObjectCode != "" && !strings.EqualFold(filter.ObjectCode, objectCode) {
				continue
			}
			alias := firstNonEmpty(reportKey, enName, id)
			out = append(out, FieldDefinition{
				Key:           strings.ToLower("PM." + objectCode + "." + table.tech + "." + id),
				Domain:        DomainPM,
				ObjectCode:    objectCode,
				Tech:          table.tech,
				OutputAlias:   alias,
				SystemField:   id,
				Source:        table.name + ".id -> pm_metrics.metric_path",
				DataType:      firstNonEmpty(dataType, "number"),
				Renderer:      "number",
				ProductClass:  productTypes,
				MetricType:    metricType,
				StatisType:    statisType,
				Unit:          unit,
				CnName:        firstNonEmpty(cnName, enName, id),
				SupportStatus: SupportSupported,
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate %s PM indicators: %w", table.name, err)
		}
		rows.Close()
	}
	return out, nil
}

func (r *PgRepository) ValidatePMMetricPaths(ctx context.Context, metricPaths []string) ([]string, error) {
	paths := normalizeMetricPaths(metricPaths)
	if len(paths) == 0 {
		return nil, nil
	}
	found := make(map[string]struct{}, len(paths))
	rows, err := r.pool.Query(ctx, `
SELECT id FROM perf_indicators_enb WHERE id = ANY($1::text[])
UNION
SELECT id FROM perf_indicators_gnb WHERE id = ANY($1::text[])
UNION
SELECT id FROM perf_indicators_gsm WHERE id = ANY($1::text[])`, paths)
	if err != nil {
		return nil, fmt.Errorf("query PM indicator ids: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan PM indicator id: %w", err)
		}
		found[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate PM indicator ids: %w", err)
	}
	rows.Close()

	if r.tsPool != nil {
		tsRows, err := r.tsPool.Query(ctx, `SELECT metric_path FROM pm_metric_dictionary WHERE metric_path = ANY($1::text[])`, paths)
		if err != nil {
			return nil, fmt.Errorf("query PM metric dictionary: %w", err)
		}
		for tsRows.Next() {
			var path string
			if err := tsRows.Scan(&path); err != nil {
				tsRows.Close()
				return nil, fmt.Errorf("scan PM metric dictionary path: %w", err)
			}
			found[path] = struct{}{}
		}
		if err := tsRows.Err(); err != nil {
			tsRows.Close()
			return nil, fmt.Errorf("iterate PM metric dictionary paths: %w", err)
		}
		tsRows.Close()
	}

	missing := make([]string, 0)
	for _, path := range paths {
		if _, ok := found[path]; !ok {
			missing = append(missing, path)
		}
	}
	return missing, nil
}

func (r *PgRepository) LoadMRRows(ctx context.Context, objectCode string, limit int) ([]ExportDataRow, error) {
	if r.tsPool == nil {
		return nil, fmt.Errorf("timescale database is not configured for MR northbound export")
	}
	rows, err := r.tsPool.Query(ctx, `
SELECT
  COALESCE(mr_type, ''),
  COALESCE(device_sn, ''),
  COALESCE(file_name, ''),
  COALESCE(file_size::text, ''),
  COALESCE(collect_time::text, ''),
  COALESCE(minio_path, ''),
  COALESCE(record_count::text, '')
FROM mr_files
WHERE ($1 = '' OR UPPER(mr_type) = UPPER($1))
ORDER BY collect_time DESC
LIMIT $2`, normalizeMRType(objectCode), normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("query mr_files rows: %w", err)
	}
	defer rows.Close()

	out := make([]ExportDataRow, 0)
	for rows.Next() {
		var mrType, deviceSN, fileName, fileSize, collectTime, minioPath, recordCount string
		if err := rows.Scan(&mrType, &deviceSN, &fileName, &fileSize, &collectTime, &minioPath, &recordCount); err != nil {
			return nil, fmt.Errorf("scan mr_files row: %w", err)
		}
		out = append(out, ExportDataRow{
			"mr.file_type":    mrType,
			"mr.device_sn":    deviceSN,
			"mr.file_name":    fileName,
			"mr.file_size":    fileSize,
			"mr.collect_time": collectTime,
			"mr.minio_path":   minioPath,
			"mr.record_count": recordCount,
		})
	}
	return out, rows.Err()
}

func (r *PgRepository) LoadLogRows(ctx context.Context, objectCode string, limit int) ([]ExportDataRow, error) {
	switch strings.ToLower(strings.TrimSpace(objectCode)) {
	case "login", "login_fix":
		return r.loadLoginLogRows(ctx, limit)
	case "operation", "operation_fix":
		return r.loadOperationLogRows(ctx, limit)
	default:
		return nil, fmt.Errorf("%w: unsupported log object %s", commonerrors.ErrInvalidInput, objectCode)
	}
}

func (r *PgRepository) loadLoginLogRows(ctx context.Context, limit int) ([]ExportDataRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT
  COALESCE(username, ''),
  COALESCE(ip_address, ''),
  COALESCE(browser, ''),
  COALESCE(os, ''),
  COALESCE(status::text, ''),
  COALESCE(message, ''),
  COALESCE(login_at::text, '')
FROM sys_login_logs
ORDER BY login_at DESC
LIMIT $1`, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("query sys_login_logs rows: %w", err)
	}
	defer rows.Close()

	out := make([]ExportDataRow, 0)
	for rows.Next() {
		var username, ipAddress, browser, osName, status, msg, loginAt string
		if err := rows.Scan(&username, &ipAddress, &browser, &osName, &status, &msg, &loginAt); err != nil {
			return nil, fmt.Errorf("scan sys_login_logs row: %w", err)
		}
		out = append(out, ExportDataRow{
			"log.username":   username,
			"log.client_ip":  ipAddress,
			"log.browser":    browser,
			"log.os":         osName,
			"log.result":     status,
			"log.message":    msg,
			"log.login_time": loginAt,
		})
	}
	return out, rows.Err()
}

func (r *PgRepository) loadOperationLogRows(ctx context.Context, limit int) ([]ExportDataRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT
  COALESCE(username, ''),
  COALESCE(action, ''),
  COALESCE(module, ''),
  COALESCE(target, ''),
  COALESCE(status::text, ''),
  COALESCE(error_msg, ''),
  COALESCE(ip_address, ''),
  COALESCE(user_agent, ''),
  COALESCE(created_at::text, '')
FROM sys_oper_logs
ORDER BY created_at DESC
LIMIT $1`, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("query sys_oper_logs rows: %w", err)
	}
	defer rows.Close()

	out := make([]ExportDataRow, 0)
	for rows.Next() {
		var username, action, module, target, status, errorMsg, ipAddress, userAgent, createdAt string
		if err := rows.Scan(&username, &action, &module, &target, &status, &errorMsg, &ipAddress, &userAgent, &createdAt); err != nil {
			return nil, fmt.Errorf("scan sys_oper_logs row: %w", err)
		}
		out = append(out, ExportDataRow{
			"log.operator":       username,
			"log.action":         action,
			"log.module":         module,
			"log.resource":       target,
			"log.result":         status,
			"log.message":        errorMsg,
			"log.client_ip":      ipAddress,
			"log.user_agent":     userAgent,
			"log.operation_time": createdAt,
		})
	}
	return out, rows.Err()
}

type pmIndicatorTable struct {
	name  string
	tech  string
	query string
}

func pmIndicatorTables(tech string) []pmIndicatorTable {
	all := []pmIndicatorTable{
		{
			name: "perf_indicators_enb",
			tech: "LTE",
			query: `
SELECT id, COALESCE(report_key, ''), COALESCE(en_name, ''), COALESCE(cn_name, ''),
       COALESCE(unit_id, ''), COALESCE(statis_type, ''), COALESCE(is_counter, '1'),
       COALESCE(data_type, ''), COALESCE(product_types, '')
  FROM perf_indicators_enb
 ORDER BY id ASC
 LIMIT 2000`,
		},
		{
			name: "perf_indicators_gnb",
			tech: "GNB",
			query: `
SELECT id, COALESCE(report_key, ''), COALESCE(en_name, ''), COALESCE(cn_name, ''),
       COALESCE(unit_id, ''), COALESCE(statis_type, ''), COALESCE(is_counter, '1'),
       COALESCE(data_type, ''), ''::text
  FROM perf_indicators_gnb
 ORDER BY id ASC
 LIMIT 2000`,
		},
		{
			name: "perf_indicators_gsm",
			tech: "GSM",
			query: `
SELECT id, COALESCE(report_key, ''), COALESCE(en_name, ''), COALESCE(cn_name, ''),
       COALESCE(unit_id, ''), COALESCE(statis_type, ''), COALESCE(is_counter, '1'),
       COALESCE(data_type, ''), COALESCE(product_types, '')
  FROM perf_indicators_gsm
 ORDER BY id ASC
 LIMIT 2000`,
		},
	}
	normalized := normalizeTech(tech)
	if normalized == "" {
		return all
	}
	out := make([]pmIndicatorTable, 0, len(all))
	for _, table := range all {
		if strings.EqualFold(table.tech, normalized) {
			out = append(out, table)
		}
	}
	return out
}

func normalizeMetricPaths(metricPaths []string) []string {
	out := make([]string, 0, len(metricPaths))
	seen := map[string]struct{}{}
	for _, path := range metricPaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func (r *PgRepository) getFileProfile(ctx context.Context, idOrCode string) (*FileProfile, error) {
	row := r.pool.QueryRow(ctx, `
SELECT code, name, vendor, scenario_name, scenario_name_en, description,
       flags, enabled, status, groups
  FROM northbound_file_profiles
 WHERE code=$1 OR id::text=$1`, idOrCode)
	return scanFileProfile(row)
}

func (r *PgRepository) getInventoryProfile(ctx context.Context, idOrCode string) (*InventoryProfile, error) {
	row := r.pool.QueryRow(ctx, `
SELECT code, name, object_code, tech, period, start_minute, path_template,
       file_name_template, compression_enabled, compression_format, enabled, status, config
  FROM northbound_inventory_profiles
 WHERE code=$1 OR id::text=$1`, idOrCode)
	return scanInventoryProfile(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanFileProfile(row scanner) (*FileProfile, error) {
	var profile FileProfile
	var flagsRaw, groupsRaw []byte
	var status string
	if err := row.Scan(
		&profile.Code, &profile.Name, &profile.Vendor, &profile.ScenarioName,
		&profile.ScenarioNameEn, &profile.Description, &flagsRaw, &profile.Enabled,
		&status, &groupsRaw,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_file_profiles row: %w", err)
	}
	if len(flagsRaw) > 0 {
		if err := json.Unmarshal(flagsRaw, &profile.Flags); err != nil {
			return nil, fmt.Errorf("unmarshal northbound file profile flags: %w", err)
		}
	}
	if len(groupsRaw) > 0 {
		if err := json.Unmarshal(groupsRaw, &profile.Groups); err != nil {
			return nil, fmt.Errorf("unmarshal northbound file profile groups: %w", err)
		}
	}
	profile.ID = profile.Code
	profile.Status = ProfileStatus(status)
	return &profile, nil
}

func scanInventoryProfile(row scanner) (*InventoryProfile, error) {
	var profile InventoryProfile
	var status string
	var period string
	var compression string
	var configRaw []byte
	if err := row.Scan(
		&profile.Code, &profile.Name, &profile.ObjectCode, &profile.Tech, &period,
		&profile.StartMinute, &profile.PathTemplate, &profile.FileNameTemplate,
		&profile.CompressionEnabled, &compression, &profile.Enabled, &status, &configRaw,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_inventory_profiles row: %w", err)
	}
	profile.ID = profile.Code
	profile.Period = Period(period)
	profile.CompressionFormat = CompressionFormat(compression)
	profile.Status = ProfileStatus(status)
	if len(configRaw) > 0 {
		if err := unmarshalInventoryProfileConfig(configRaw, &profile.Fields); err != nil {
			return nil, fmt.Errorf("unmarshal northbound inventory profile config: %w", err)
		}
	}
	return &profile, nil
}

type inventoryProfileConfig struct {
	Fields []InventoryFieldConfig `json:"fields"`
}

func marshalInventoryProfileConfig(fields []InventoryFieldConfig) ([]byte, error) {
	return json.Marshal(inventoryProfileConfig{Fields: fields})
}

func unmarshalInventoryProfileConfig(raw []byte, fields *[]InventoryFieldConfig) error {
	var config inventoryProfileConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	*fields = config.Fields
	return nil
}

func scanFileRuns(rows pgx.Rows) ([]FileRun, error) {
	runs := make([]FileRun, 0)
	for rows.Next() {
		run, err := scanFileRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *run)
	}
	return runs, rows.Err()
}

func scanFileRun(row scanner) (*FileRun, error) {
	var run FileRun
	var profileKind, domain, status string
	var compression string
	var windowStart, windowEnd sql.NullTime
	var summaryRaw []byte
	if err := row.Scan(
		&run.ID, &profileKind, &run.ProfileCode, &run.GroupID, &domain, &run.ObjectCode, &status,
		&windowStart, &windowEnd, &run.ArtifactPath, &run.ArtifactName, &run.ArtifactContent,
		&run.ArtifactSize, &run.RowCount, &run.CompressionEnabled, &compression,
		&run.ErrorMessage, &summaryRaw, &run.CreatedAt, &run.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_file_runs row: %w", err)
	}
	run.ProfileKind = ProfileKind(profileKind)
	run.Domain = Domain(domain)
	run.Status = RunStatus(status)
	run.CompressionFormat = CompressionFormat(compression)
	if windowStart.Valid {
		t := windowStart.Time
		run.WindowStart = &t
	}
	if windowEnd.Valid {
		t := windowEnd.Time
		run.WindowEnd = &t
	}
	if len(summaryRaw) > 0 {
		if err := json.Unmarshal(summaryRaw, &run.Summary); err != nil {
			return nil, fmt.Errorf("unmarshal northbound file run summary: %w", err)
		}
	}
	return &run, nil
}

func mergeFileProfile(current FileProfile, req UpdateFileProfileRequest) FileProfile {
	if req.Name != "" {
		current.Name = req.Name
	}
	if req.Vendor != "" {
		current.Vendor = req.Vendor
	}
	if req.ScenarioName != "" {
		current.ScenarioName = req.ScenarioName
	}
	if req.ScenarioNameEn != "" {
		current.ScenarioNameEn = req.ScenarioNameEn
	}
	if req.Description != "" {
		current.Description = req.Description
	}
	if req.Flags != nil {
		current.Flags = req.Flags
	}
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.Status != "" {
		current.Status = req.Status
	} else {
		current.Status = statusFromEnabled(current.Enabled)
	}
	if req.Groups != nil {
		current.Groups = req.Groups
	}
	return current
}

func mergeInventoryProfile(current InventoryProfile, req UpdateInventoryProfileRequest) InventoryProfile {
	if req.Name != "" {
		current.Name = req.Name
	}
	if req.ObjectCode != "" {
		current.ObjectCode = req.ObjectCode
	}
	if req.Tech != "" {
		current.Tech = req.Tech
	}
	if req.Period != "" {
		current.Period = req.Period
	}
	if req.StartMinute != nil && *req.StartMinute >= 0 && *req.StartMinute <= 59 {
		current.StartMinute = *req.StartMinute
	}
	if req.PathTemplate != "" {
		current.PathTemplate = req.PathTemplate
	}
	if req.FileNameTemplate != "" {
		current.FileNameTemplate = req.FileNameTemplate
	}
	if req.CompressionEnabled != nil {
		current.CompressionEnabled = *req.CompressionEnabled
	}
	if req.CompressionFormat != "" {
		current.CompressionFormat = req.CompressionFormat
	}
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.Status != "" {
		current.Status = req.Status
	} else {
		current.Status = statusFromEnabled(current.Enabled)
	}
	if req.Fields != nil {
		current.Fields = append([]InventoryFieldConfig(nil), req.Fields...)
	}
	return current
}

func statusFromEnabled(enabled bool) ProfileStatus {
	if enabled {
		return StatusNormal
	}
	return StatusTerminated
}

func normalizeTech(tech string) string {
	tech = strings.TrimSpace(tech)
	switch strings.ToUpper(tech) {
	case "ENB", "LTE":
		return "LTE"
	case "GNB", "NR":
		return "GNB"
	case "GSM":
		return "GSM"
	default:
		return strings.ToUpper(tech)
	}
}

func normalizeDeviceTech(tech string) string {
	tech = strings.TrimSpace(tech)
	switch strings.ToUpper(tech) {
	case "ENB", "LTE":
		return "LTE"
	case "GNB", "NR":
		return "NR"
	case "GSM":
		return "GSM"
	default:
		return strings.ToUpper(tech)
	}
}

func normalizeMRType(objectCode string) string {
	switch strings.ToUpper(strings.TrimSpace(objectCode)) {
	case "MRO", "MRE", "MRS":
		return strings.ToUpper(strings.TrimSpace(objectCode))
	default:
		return ""
	}
}
