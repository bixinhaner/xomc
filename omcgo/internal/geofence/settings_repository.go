package geofence

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/storage"
)

const (
	geofenceConfigCategory = "geofence"
	geofenceModeConfigKey  = "mode"
)

type SettingsRepository interface {
	GetSettings(context.Context) (Settings, error)
	UpdateSettings(
		context.Context,
		Settings,
		uuid.UUID,
		time.Time,
	) (Settings, error)
	PreviewSettings(context.Context, Settings) (SettingsPreview, error)
}

type SettingsPreview struct {
	Current              Settings `json:"current"`
	Proposed             Settings `json:"proposed"`
	EnabledGeofences     int64    `json:"enabled_geofences"`
	ActiveBindings       int64    `json:"active_bindings"`
	NewlyObservedDevices int64    `json:"newly_observed_devices"`
}

type PgSettingsRepository struct {
	db storage.DB
}

func NewPgSettingsRepository(db storage.DB) *PgSettingsRepository {
	return &PgSettingsRepository{db: db}
}

func buildGetSystemModeQuery() (string, []any, error) {
	query, args, err := storage.Psql.
		Select("value").
		From("sys_configs").
		Where(sq.Eq{
			"category": geofenceConfigCategory,
			"key":      geofenceModeConfigKey,
		}).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build get geofence system mode: %w", err)
	}
	return query, args, nil
}

func buildListCarrierSettingsQuery() (string, []any, error) {
	query, args, err := storage.Psql.
		Select(
			"carrier",
			"mode",
			"default_baseline_radius_meters",
			"updated_by",
			"updated_at",
		).
		From("geofence_carrier_settings").
		OrderBy("carrier").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build list geofence carrier settings: %w", err)
	}
	return query, args, nil
}

func buildLockSystemModeQuery() (string, []any, error) {
	query, args, err := storage.Psql.
		Select("value").
		From("sys_configs").
		Where(sq.Eq{
			"category": geofenceConfigCategory,
			"key":      geofenceModeConfigKey,
		}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build lock geofence system mode: %w", err)
	}
	return query, args, nil
}

func buildLockCarrierSettingsQuery(carriers []string) (string, []any, error) {
	ordered := append([]string(nil), carriers...)
	sort.Strings(ordered)
	query, args, err := storage.Psql.
		Select("carrier").
		From("geofence_carrier_settings").
		Where(sq.Eq{"carrier": ordered}).
		OrderBy("carrier").
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build lock geofence carrier settings: %w", err)
	}
	return query, args, nil
}

func buildUpsertSystemModeQuery(
	mode RuntimeMode,
	updatedAt time.Time,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Insert("sys_configs").
		Columns(
			"category",
			"key",
			"value",
			"value_type",
			"description",
			"is_public",
			"created_at",
			"updated_at",
		).
		Values(
			geofenceConfigCategory,
			geofenceModeConfigKey,
			mode,
			"string",
			"电子围栏系统运行模式：off/observe/enforce",
			false,
			updatedAt,
			updatedAt,
		).
		Suffix(
			"ON CONFLICT (category,key) DO UPDATE SET " +
				"value = EXCLUDED.value, updated_at = EXCLUDED.updated_at",
		).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build upsert geofence system mode: %w", err)
	}
	return query, args, nil
}

func buildUpsertCarrierSettingQuery(
	setting CarrierSetting,
	actorID uuid.UUID,
	updatedAt time.Time,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Insert("geofence_carrier_settings").
		Columns(
			"carrier",
			"mode",
			"default_baseline_radius_meters",
			"updated_by",
			"updated_at",
		).
		Values(
			setting.Carrier,
			setting.Mode,
			setting.DefaultBaselineRadiusMeters,
			actorID,
			updatedAt,
		).
		Suffix(
			"ON CONFLICT (carrier) DO UPDATE SET " +
				"mode = EXCLUDED.mode, " +
				"default_baseline_radius_meters = EXCLUDED.default_baseline_radius_meters, " +
				"updated_by = EXCLUDED.updated_by, " +
				"updated_at = EXCLUDED.updated_at",
		).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build upsert geofence carrier setting: %w", err)
	}
	return query, args, nil
}

func buildSettingsPreviewQueries(
	current Settings,
	proposed Settings,
) ([]repositoryQuery, error) {
	carriers := make([]string, 0, len(proposed.Carriers))
	currentModes := make(map[string]RuntimeMode, len(current.Carriers))
	for _, setting := range current.Carriers {
		currentModes[setting.Carrier] = setting.Mode
	}
	newlyObservedCarriers := make([]string, 0, len(proposed.Carriers))
	for _, setting := range proposed.Carriers {
		carriers = append(carriers, setting.Carrier)
		if EffectiveRuntimeMode(
			current.SystemMode,
			currentModes[setting.Carrier],
		) == RuntimeModeOff &&
			EffectiveRuntimeMode(
				proposed.SystemMode,
				setting.Mode,
			) == RuntimeModeObserve {
			newlyObservedCarriers = append(newlyObservedCarriers, setting.Carrier)
		}
	}

	enabledSQL, enabledArgs, err := storage.Psql.
		Select("COUNT(*)").
		From("geofence_definitions d").
		Where(sq.Eq{
			"d.status":  DefinitionStatusEnabled,
			"d.carrier": carriers,
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count enabled geofences: %w", err)
	}
	bindingSQL, bindingArgs, err := storage.Psql.
		Select("COUNT(*)").
		From("device_geofence_bindings b").
		Join("geofence_definitions d ON d.id = b.geofence_id").
		Where(sq.Eq{
			"b.status":  BindingStatusActive,
			"d.status":  DefinitionStatusEnabled,
			"d.carrier": carriers,
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count active geofence bindings: %w", err)
	}
	devicesSQL, devicesArgs, err := storage.Psql.
		Select("COUNT(DISTINCT b.device_id)").
		From("device_geofence_bindings b").
		Join("geofence_definitions d ON d.id = b.geofence_id").
		Where(sq.Eq{
			"b.status":  BindingStatusActive,
			"d.status":  DefinitionStatusEnabled,
			"d.carrier": newlyObservedCarriers,
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count newly observed devices: %w", err)
	}
	return []repositoryQuery{
		{sql: enabledSQL, args: enabledArgs},
		{sql: bindingSQL, args: bindingArgs},
		{sql: devicesSQL, args: devicesArgs},
	}, nil
}

func (r *PgSettingsRepository) GetSettings(
	ctx context.Context,
) (Settings, error) {
	systemMode := RuntimeModeOff
	query, args, err := buildGetSystemModeQuery()
	if err != nil {
		return Settings{}, err
	}
	var storedMode RuntimeMode
	err = r.db.QueryRow(ctx, query, args...).Scan(&storedMode)
	switch {
	case err == nil:
		if _, ok := runtimeModeRank(storedMode); ok {
			systemMode = storedMode
		}
	case err == pgx.ErrNoRows:
	default:
		return Settings{}, fmt.Errorf("get geofence system mode: %w", err)
	}

	query, args, err = buildListCarrierSettingsQuery()
	if err != nil {
		return Settings{}, err
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return Settings{}, fmt.Errorf("list geofence carrier settings: %w", err)
	}
	defer rows.Close()

	settings := Settings{
		SystemMode: systemMode,
		Carriers:   make([]CarrierSetting, 0),
	}
	for rows.Next() {
		var setting CarrierSetting
		if err := rows.Scan(
			&setting.Carrier,
			&setting.Mode,
			&setting.DefaultBaselineRadiusMeters,
			&setting.UpdatedBy,
			&setting.UpdatedAt,
		); err != nil {
			return Settings{}, fmt.Errorf("scan geofence carrier setting: %w", err)
		}
		if _, ok := runtimeModeRank(setting.Mode); !ok {
			setting.Mode = RuntimeModeOff
		}
		setting.EffectiveMode = EffectiveRuntimeMode(systemMode, setting.Mode)
		settings.Carriers = append(settings.Carriers, setting)
	}
	if err := rows.Err(); err != nil {
		return Settings{}, fmt.Errorf("iterate geofence carrier settings: %w", err)
	}
	return settings, nil
}

func (r *PgSettingsRepository) PreviewSettings(
	ctx context.Context,
	proposed Settings,
) (SettingsPreview, error) {
	if err := ValidateSettings(proposed); err != nil {
		return SettingsPreview{}, err
	}
	proposed = normalizeSettings(proposed)
	current, err := r.GetSettings(ctx)
	if err != nil {
		return SettingsPreview{}, err
	}
	queries, err := buildSettingsPreviewQueries(current, proposed)
	if err != nil {
		return SettingsPreview{}, err
	}
	preview := SettingsPreview{Current: current, Proposed: proposed}
	destinations := []*int64{
		&preview.EnabledGeofences,
		&preview.ActiveBindings,
		&preview.NewlyObservedDevices,
	}
	for index, query := range queries {
		if err := r.db.QueryRow(ctx, query.sql, query.args...).
			Scan(destinations[index]); err != nil {
			return SettingsPreview{}, fmt.Errorf(
				"query geofence settings preview count %d: %w",
				index,
				err,
			)
		}
	}
	return preview, nil
}

func (r *PgSettingsRepository) UpdateSettings(
	ctx context.Context,
	settings Settings,
	actorID uuid.UUID,
	updatedAt time.Time,
) (Settings, error) {
	if err := ValidateSettings(settings); err != nil {
		return Settings{}, err
	}
	settings = normalizeSettings(settings)
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("begin update geofence settings: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	systemLockSQL, systemLockArgs, err := buildLockSystemModeQuery()
	if err != nil {
		return Settings{}, err
	}
	if _, err := tx.Exec(ctx, systemLockSQL, systemLockArgs...); err != nil {
		return Settings{}, fmt.Errorf("lock geofence system mode: %w", err)
	}
	carriers := make([]string, 0, len(settings.Carriers))
	for _, setting := range settings.Carriers {
		carriers = append(carriers, setting.Carrier)
	}
	carrierLockSQL, carrierLockArgs, err := buildLockCarrierSettingsQuery(carriers)
	if err != nil {
		return Settings{}, err
	}
	if _, err := tx.Exec(ctx, carrierLockSQL, carrierLockArgs...); err != nil {
		return Settings{}, fmt.Errorf("lock geofence carrier settings: %w", err)
	}
	systemSQL, systemArgs, err := buildUpsertSystemModeQuery(
		settings.SystemMode,
		updatedAt,
	)
	if err != nil {
		return Settings{}, err
	}
	if _, err := tx.Exec(ctx, systemSQL, systemArgs...); err != nil {
		return Settings{}, fmt.Errorf("update geofence system mode: %w", err)
	}
	for _, setting := range settings.Carriers {
		query, args, err := buildUpsertCarrierSettingQuery(
			setting,
			actorID,
			updatedAt,
		)
		if err != nil {
			return Settings{}, err
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return Settings{}, fmt.Errorf(
				"update geofence carrier %s: %w",
				setting.Carrier,
				err,
			)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Settings{}, fmt.Errorf("commit geofence settings: %w", err)
	}
	committed, err := r.GetSettings(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("read committed geofence settings: %w", err)
	}
	return committed, nil
}

func normalizeSettings(settings Settings) Settings {
	normalized := Settings{
		SystemMode: settings.SystemMode,
		Carriers:   make([]CarrierSetting, 0, len(settings.Carriers)),
	}
	for _, setting := range settings.Carriers {
		setting.Carrier = strings.ToLower(strings.TrimSpace(setting.Carrier))
		setting.EffectiveMode = EffectiveRuntimeMode(
			settings.SystemMode,
			setting.Mode,
		)
		normalized.Carriers = append(normalized.Carriers, setting)
	}
	sort.Slice(normalized.Carriers, func(i, j int) bool {
		return normalized.Carriers[i].Carrier < normalized.Carriers[j].Carrier
	})
	return normalized
}
