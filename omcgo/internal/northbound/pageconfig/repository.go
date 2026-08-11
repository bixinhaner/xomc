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
		if err := r.backfillDefaultFileProfileScenarioNames(ctx, profile); err != nil {
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
		if defaultGroup.Domain == DomainLOG {
			if strings.TrimSpace(groups[i].PathTemplate) == "" || isLegacyLogPathTemplate(groups[i].PathTemplate) {
				groups[i].PathTemplate = defaultGroup.PathTemplate
				changed = true
			}
			if strings.TrimSpace(groups[i].FileNameTemplate) == "" || isLegacyLogFileNameTemplate(groups[i].FileNameTemplate) {
				groups[i].FileNameTemplate = defaultGroup.FileNameTemplate
				changed = true
			}
			if !groups[i].CompressionEnabled {
				groups[i].CompressionEnabled = true
				changed = true
			}
			if groups[i].CompressionFormat != defaultGroup.CompressionFormat {
				groups[i].CompressionFormat = defaultGroup.CompressionFormat
				changed = true
			}
			if normalizeScenarioLogObjects(groups[i].Objects) {
				changed = true
			}
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

func (r *PgRepository) backfillDefaultFileProfileScenarioNames(ctx context.Context, profile FileProfile) error {
	scenarioName := strings.TrimSpace(profile.ScenarioName)
	scenarioNameEn := strings.TrimSpace(profile.ScenarioNameEn)
	if scenarioName == "" || scenarioNameEn == "" {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
UPDATE northbound_file_profiles
   SET scenario_name = $2,
       scenario_name_en = $3,
       updated_at = now()
 WHERE code = $1
   AND (
       NULLIF(BTRIM(scenario_name), '') IS NULL
       OR (scenario_name = scenario_name_en AND scenario_name_en = $3)
   )`,
		profile.Code, scenarioName, scenarioNameEn)
	if err != nil {
		return fmt.Errorf("backfill default northbound_file_profiles scenario names %s: %w", profile.Code, err)
	}
	return nil
}

func isLegacyLogPathTemplate(template string) bool {
	normalized := strings.ToLower(strings.TrimSpace(template))
	return strings.Contains(normalized, "/logs/#datetime#/") || strings.Contains(normalized, "/#province#/#omc-r#/logs/")
}

func isLegacyLogFileNameTemplate(template string) bool {
	normalized := strings.ToLower(strings.TrimSpace(template))
	return strings.Contains(normalized, "northbound-log") || strings.Contains(normalized, "{login|operation}")
}

func normalizeScenarioLogObjects(objects []ScenarioObject) bool {
	changed := false
	for i := range objects {
		normalized := normalizeLogObjectCode(objects[i].Code)
		if normalized != objects[i].Code {
			objects[i].Code = normalized
			changed = true
		}
	}
	return changed
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
       flags, enabled, status, groups, created_at, updated_at
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
          flags, enabled, status, groups, created_at, updated_at`,
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
           flags, enabled, status, groups, created_at, updated_at`,
		idOrCode, merged.Name, merged.Vendor, merged.ScenarioName, merged.ScenarioNameEn, merged.Description,
		flags, merged.Enabled, merged.Status, groups)
	return scanFileProfile(row)
}

func (r *PgRepository) ListInventoryProfiles(ctx context.Context) ([]InventoryProfile, error) {
	rows, err := r.pool.Query(ctx, `
SELECT code, name, object_code, tech, period, start_minute, path_template,
       file_name_template, compression_enabled, compression_format, enabled, status, config,
       created_at, updated_at
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
           file_name_template, compression_enabled, compression_format, enabled, status, config,
           created_at, updated_at`,
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
	if filter.LatestPerProfile {
		if err := r.pool.QueryRow(ctx, fmt.Sprintf(`
SELECT COUNT(*)
  FROM (
    SELECT DISTINCT profile_code
      FROM northbound_file_runs
     WHERE %s
  ) latest`, whereClause), args...).Scan(&total); err != nil {
			return RunListResult{}, fmt.Errorf("count latest northbound_file_runs: %w", err)
		}
		limit := normalizeLimit(filter.Limit)
		offset := normalizeOffset(filter.Offset)
		args = append(args, limit, offset)
		query := fmt.Sprintf(`
SELECT id, profile_kind, profile_code, group_id, domain, object_code, status,
       window_start, window_end, artifact_path, artifact_name, artifact_content,
       artifact_size, row_count, compression_enabled, compression_format,
       error_message, summary, created_at, updated_at
  FROM (
    SELECT DISTINCT ON (profile_code)
           id::text AS id, profile_kind, profile_code, group_id, domain, object_code, status,
           window_start, window_end, artifact_path, artifact_name,
           ''::text AS artifact_content, artifact_size, row_count, compression_enabled,
           COALESCE(compression_format, '') AS compression_format,
           error_message, summary, created_at, updated_at
      FROM northbound_file_runs
     WHERE %s
     ORDER BY profile_code, created_at DESC
  ) latest
 ORDER BY created_at DESC
 LIMIT $%d OFFSET $%d`, whereClause, len(args)-1, len(args))
		rows, err := r.pool.Query(ctx, query, args...)
		if err != nil {
			return RunListResult{}, fmt.Errorf("query latest northbound_file_runs: %w", err)
		}
		defer rows.Close()
		items, err := scanFileRuns(rows)
		if err != nil {
			return RunListResult{}, err
		}
		return RunListResult{Items: items, Total: total, Limit: limit, Offset: offset}, nil
	}

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
  COALESCE(d.serial_number, '') AS "device.serial_number",
  COALESCE(d.manufacturer, '') AS "device.manufacturer",
  COALESCE(d.model_name, '') AS "device.model_name",
  COALESCE(d.product_class, '') AS "device.product_class",
  COALESCE(d.firmware_version, '') AS "device.firmware_version",
  COALESCE(d.ip_address::text, '') AS "device.ip_address",
  COALESCE(d.site_name, '') AS "device.site_name",
  COALESCE(d.site_id, '') AS "device.site_id",
  COALESCE((
    SELECT string_agg(DISTINCT dg.name, ',' ORDER BY dg.name)
      FROM device_group_members dgm
      JOIN device_groups dg ON dg.id = dgm.group_id
     WHERE dgm.device_id = d.id
  ), '') AS "device_groups.name",
  COALESCE((SELECT p.product_name FROM products p WHERE p.id = d.product_id), '') AS "product.name",
  COALESCE(d.last_inform_at::text, '') AS "device.last_inform_at",
  COALESCE(d.is_online::text, '') AS "device.is_online",
  COALESCE(d.longitude::text, '') AS "device.longitude",
  COALESCE(d.latitude::text, '') AS "device.latitude",
  COALESCE(d.lifecycle_state, '') AS "device.lifecycle_state",
  COALESCE(d.created_at::text, '') AS "device.created_at",
  COALESCE(di.device_id::text, '') AS "device_info.device_id",
  COALESCE(di.device_name, '') AS "device_info.device_name",
  COALESCE(di.address, '') AS "device_info.address",
  COALESCE(di.remark, '') AS "device_info.remark",
  COALESCE(di.project_status, '') AS "device_info.project_status",
  COALESCE(di.height::text, '') AS "device_info.height",
  COALESCE(di.eci, '') AS "device_info.eci",
  COALESCE(di.pci, '') AS "device_info.pci",
  COALESCE(di.cell_id, '') AS "device_info.cell_id",
  COALESCE(di.freq_point, '') AS "device_info.freq_point",
  COALESCE(di.bandwidth::text, '') AS "device_info.bandwidth",
  COALESCE(di.transmit_power::text, '') AS "device_info.transmit_power",
  COALESCE(di.plmn, '') AS "device_info.plmn",
  COALESCE(di.rf_status, '') AS "device_info.rf_status",
  COALESCE(di.cell_status, '') AS "device_info.cell_status",
  COALESCE(di.mme_status, '') AS "device_info.mme_status",
  COALESCE(di.sync_status, '') AS "device_info.sync_status",
  COALESCE(di.kpi_status, '') AS "device_info.kpi_status",
  COALESCE(di.num_of_cells::text, '') AS "device_info.num_of_cells",
  COALESCE(di.gps_status, '') AS "device_info.gps_status",
  COALESCE(di.alarm_severity, '') AS "device_info.alarm_severity",
  COALESCE(di.license_status, '') AS "device_info.license_status",
  COALESCE(di.mac, '') AS "device_info.mac",
  COALESCE(di.hardware_version, '') AS "device_info.hardware_version",
  COALESCE(di.first_online_time::text, '') AS "device_info.first_online_time",
  COALESCE(di.last_online_time::text, '') AS "device_info.last_online_time",
  COALESCE(di.last_offline_time::text, '') AS "device_info.last_offline_time",
  COALESCE(di.run_time::text, '') AS "device_info.run_time",
  COALESCE(di.creator, '') AS "device_info.creator",
  COALESCE(di.updater, '') AS "device_info.updater",
  COALESCE(di.created_at::text, '') AS "device_info.created_at",
  COALESCE(di.updated_at::text, '') AS "device_info.updated_at",
  COALESCE(di.tac, '') AS "device_info.tac",
  COALESCE(di.band, '') AS "device_info.band",
  COALESCE(di.ul_earfcn, '') AS "device_info.ul_earfcn",
  COALESCE(di.subframe_assignment, '') AS "device_info.subframe_assignment",
  COALESCE(di.special_subframe, '') AS "device_info.special_subframe",
  COALESCE(di.root_index, '') AS "device_info.root_index",
  COALESCE(di.gps_satellites::text, '') AS "device_info.gps_satellites",
  COALESCE(di.gps_height::text, '') AS "device_info.gps_height",
  COALESCE(di.lock_status, '') AS "device_info.lock_status",
  COALESCE(di.enb_id, '') AS "device_info.enb_id",
  COALESCE(di.network_model, '') AS "device_info.network_model",
  COALESCE(di.lac, '') AS "device_info.lac",
  COALESCE(di.cumulative_online_duration::text, '') AS "device_info.cumulative_online_duration",
  COALESCE(di.op_state, '') AS "device_info.op_state",
  COALESCE(di.admin_state, '') AS "device_info.admin_state",
  COALESCE(di.ipsec_addr, '') AS "device_info.ipsec_addr",
  COALESCE(di.bsc_select, '') AS "device_info.bsc_select",
  COALESCE(di.oml_remote_ip, '') AS "device_info.oml_remote_ip",
  COALESCE(di.oml_remote_ip_bak, '') AS "device_info.oml_remote_ip_bak",
  COALESCE(di.ipa_unit_id, '') AS "device_info.ipa_unit_id",
  COALESCE(di.ue_count::text, '') AS "device_info.ue_count",
  COALESCE(di.active_alarm_count::text, '') AS "device_info.active_alarm_count",
  COALESCE(di.name_sync_pending::text, '') AS "device_info.name_sync_pending",
  COALESCE(di.lmt_device_name, '') AS "device_info.lmt_device_name",
  COALESCE(di.highest_alarm_severity::text, '') AS "device_info.highest_alarm_severity",
  COALESCE(di.highest_severity_alarm_count::text, '') AS "device_info.highest_severity_alarm_count",
  COALESCE(to_jsonb(di), '{}'::jsonb)::text AS "device_info.__json"
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

	fieldDescriptions := rows.FieldDescriptions()
	out := make([]ExportDataRow, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("scan device snapshot row: %w", err)
		}
		row := make(ExportDataRow, len(values)+16)
		for i, value := range values {
			row[string(fieldDescriptions[i].Name)] = exportDataValue(value)
		}
		flattenDeviceInfoJSON(row, row["device_info.__json"])
		delete(row, "device_info.__json")
		snapshotTime := time.Now().Format(time.RFC3339)
		serialNumber := row["device.serial_number"]
		opState := row["device_info.op_state"]
		isOnline := row["device.is_online"]
		ipAddress := row["device.ip_address"]
		productClass := row["device.product_class"]
		row["inventory.enb.snapshot_time"] = snapshotTime
		row["inventory.gnb.snapshot_time"] = snapshotTime
		row["inventory.gsm.snapshot_time"] = snapshotTime
		row["inventory.omc.snapshot_time"] = snapshotTime
		row["inventory.enb.serial_number"] = serialNumber
		row["inventory.gnb.serial_number"] = serialNumber
		row["inventory.gsm.serial_number"] = serialNumber
		row["inventory.enb.cell_status"] = opState
		row["inventory.gnb.cell_status"] = opState
		row["inventory.gsm.cell_status"] = opState
		row["inventory.enb.online_status"] = isOnline
		row["inventory.gnb.online_status"] = isOnline
		row["inventory.gsm.online_status"] = isOnline
		row["inventory.enb.ip_address"] = ipAddress
		row["inventory.gnb.ip_address"] = ipAddress
		row["inventory.gsm.ip_address"] = ipAddress
		row["inventory.enb.product_type"] = productClass
		row["inventory.gnb.product_type"] = productClass
		row["inventory.gsm.product_type"] = productClass
		out = append(out, row)
	}
	return out, rows.Err()
}

func exportDataValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}

func flattenDeviceInfoJSON(row ExportDataRow, raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" || raw == "null" {
		return
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var values map[string]any
	if err := decoder.Decode(&values); err != nil {
		return
	}
	for column, value := range values {
		key := "device_info." + column
		if existing, ok := row[key]; ok && existing != "" {
			continue
		}
		row[key] = exportJSONDataValue(value)
	}
}

func exportJSONDataValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		return fmt.Sprint(v)
	default:
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
		return fmt.Sprint(v)
	}
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

func (r *PgRepository) ListDeviceInfoFields(ctx context.Context, filter FieldFilter) ([]FieldDefinition, error) {
	if !supportsOptionalDeviceInfoFields(filter.Domain, filter.ObjectCode) {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'device_info'
ORDER BY ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("query device_info field catalog: %w", err)
	}
	defer rows.Close()

	out := make([]FieldDefinition, 0)
	for rows.Next() {
		var column, dataType string
		if err := rows.Scan(&column, &dataType); err != nil {
			return nil, fmt.Errorf("scan device_info field catalog: %w", err)
		}
		if field, ok := optionalDeviceInfoFieldFromColumn(filter.Domain, filter.ObjectCode, column, dataType); ok {
			out = append(out, field)
		}
	}
	return out, rows.Err()
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

func (r *PgRepository) LoadLogRows(ctx context.Context, objectCode string, windowStart time.Time, windowEnd time.Time, limit int) ([]ExportDataRow, error) {
	switch normalizeLogObjectCode(objectCode) {
	case "login", "login_fix":
		return r.loadLoginLogRows(ctx, windowStart, windowEnd, limit)
	case "operation", "operation_fix":
		return r.loadOperationLogRows(ctx, windowStart, windowEnd, limit)
	default:
		return nil, fmt.Errorf("%w: unsupported log object %s", commonerrors.ErrInvalidInput, objectCode)
	}
}

func (r *PgRepository) loadLoginLogRows(ctx context.Context, windowStart time.Time, windowEnd time.Time, limit int) ([]ExportDataRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT
  COALESCE(id::text, ''),
  COALESCE(username, ''),
  COALESCE(ip_address, ''),
  COALESCE(browser, ''),
  COALESCE(os, ''),
  COALESCE(status::text, ''),
  COALESCE(message, ''),
  COALESCE(to_char(login_at, 'YYYY-MM-DD HH24:MI:SS'), '')
FROM sys_login_logs
WHERE login_at >= $1 AND login_at < $2
ORDER BY login_at ASC
LIMIT $3`, windowStart, windowEnd, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("query sys_login_logs rows: %w", err)
	}
	defer rows.Close()

	out := make([]ExportDataRow, 0)
	for rows.Next() {
		var id, username, ipAddress, browser, osName, status, msg, loginAt string
		if err := rows.Scan(&id, &username, &ipAddress, &browser, &osName, &status, &msg, &loginAt); err != nil {
			return nil, fmt.Errorf("scan sys_login_logs row: %w", err)
		}
		result := "fail"
		resultText := "Fail"
		failureReason := firstNonEmpty(msg, " ")
		if isLogSuccess(status) {
			result = "success"
			resultText = "Success"
			failureReason = " "
		}
		detail := firstNonEmpty(msg, "user login")
		out = append(out, ExportDataRow{
			"log.id":              id,
			"log.username":        username,
			"log.user_name":       username,
			"log.account_name":    username,
			"log.client_ip":       ipAddress,
			"log.ip_address":      ipAddress,
			"log.terminal_ip":     ipAddress,
			"log.terminal_name":   firstNonEmpty(browser, osName, "Browser"),
			"log.browser":         browser,
			"log.os":              osName,
			"log.result":          result,
			"log.result_text":     resultText,
			"log.failure_reason":  failureReason,
			"log.message":         msg,
			"log.detail":          detail,
			"log.log_name":        "LoginLogout",
			"log.login_time":      loginAt,
			"log.log_time":        loginAt,
			"log.log_start_time":  loginAt,
			"log.log_end_time":    loginAt,
			"log.time":            loginAt,
			"log.sys_source_name": "baicells omc",
		})
	}
	return out, rows.Err()
}

func (r *PgRepository) loadOperationLogRows(ctx context.Context, windowStart time.Time, windowEnd time.Time, limit int) ([]ExportDataRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT
  COALESCE(username, ''),
  COALESCE(action, ''),
  COALESCE(module, ''),
  COALESCE(target, ''),
  COALESCE(detail, ''),
  COALESCE(status::text, ''),
  COALESCE(error_msg, ''),
  COALESCE(ip_address, ''),
  COALESCE(user_agent, ''),
  COALESCE(to_char(created_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
  COALESCE(to_char(created_at + (COALESCE(cost_ms, 0) * interval '1 millisecond'), 'YYYY-MM-DD HH24:MI:SS'), '')
FROM sys_oper_logs
WHERE created_at >= $1 AND created_at < $2
ORDER BY created_at ASC
LIMIT $3`, windowStart, windowEnd, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("query sys_oper_logs rows: %w", err)
	}
	defer rows.Close()

	out := make([]ExportDataRow, 0)
	for rows.Next() {
		var username, action, module, target, detail, status, errorMsg, ipAddress, userAgent, createdAt, endedAt string
		if err := rows.Scan(&username, &action, &module, &target, &detail, &status, &errorMsg, &ipAddress, &userAgent, &createdAt, &endedAt); err != nil {
			return nil, fmt.Errorf("scan sys_oper_logs row: %w", err)
		}
		result := "fail"
		resultText := "Fail"
		failureReason := firstNonEmpty(errorMsg, " ")
		if isLogSuccess(status) {
			result = "success"
			resultText = "Success"
			failureReason = " "
		}
		logName := firstNonEmpty(action, module, "Operation")
		recordDetail := firstNonEmpty(detail, target, action, module, "operation")
		out = append(out, ExportDataRow{
			"log.operator":        username,
			"log.username":        username,
			"log.user_name":       username,
			"log.main_name":       username,
			"log.action":          action,
			"log.module":          module,
			"log.resource":        target,
			"log.target":          target,
			"log.detail":          recordDetail,
			"log.log_name":        logName,
			"log.result":          result,
			"log.result_text":     resultText,
			"log.failure_reason":  failureReason,
			"log.message":         firstNonEmpty(errorMsg, recordDetail),
			"log.client_ip":       ipAddress,
			"log.ip_address":      ipAddress,
			"log.terminal_ip":     ipAddress,
			"log.user_agent":      userAgent,
			"log.terminal_name":   firstNonEmpty(userAgent, "Browser"),
			"log.operation_time":  createdAt,
			"log.log_time":        createdAt,
			"log.op_start_time":   createdAt,
			"log.op_end_time":     firstNonEmpty(endedAt, createdAt),
			"log.sys_source_name": "baicells omc",
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
       flags, enabled, status, groups, created_at, updated_at
  FROM northbound_file_profiles
 WHERE code=$1 OR id::text=$1`, idOrCode)
	return scanFileProfile(row)
}

func (r *PgRepository) getInventoryProfile(ctx context.Context, idOrCode string) (*InventoryProfile, error) {
	row := r.pool.QueryRow(ctx, `
SELECT code, name, object_code, tech, period, start_minute, path_template,
       file_name_template, compression_enabled, compression_format, enabled, status, config,
       created_at, updated_at
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
		&status, &groupsRaw, &profile.CreatedAt, &profile.UpdatedAt,
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
		&profile.CreatedAt, &profile.UpdatedAt,
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
