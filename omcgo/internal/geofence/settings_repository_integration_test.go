package geofence

import (
	"context"
	"os"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgSettingsRepositoryAtomicUpdateAndPreviewIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	carrier := "s" + uuid.NewString()[:8]
	now := time.Now().UTC().Truncate(time.Microsecond)
	actorID := uuid.New()
	seedSettingsFixture(t, ctx, pool, carrier, now)
	t.Cleanup(func() {
		deleteSettingsFixture(context.Background(), pool, carrier)
	})

	repository := NewPgSettingsRepository(storage.NewPoolDB(pool))
	current, err := repository.GetSettings(ctx)
	require.NoError(t, err)
	assert.Equal(t, RuntimeModeOff, current.SystemMode)
	assert.Equal(t, RuntimeModeOff, findCarrierSetting(t, current, carrier).EffectiveMode)

	proposed := Settings{
		SystemMode: RuntimeModeObserve,
		Carriers: []CarrierSetting{{
			Carrier:                     carrier,
			Mode:                        RuntimeModeObserve,
			DefaultBaselineRadiusMeters: 150,
		}},
	}
	preview, err := repository.PreviewSettings(ctx, proposed)
	require.NoError(t, err)
	assert.Equal(t, RuntimeModeOff, preview.Current.SystemMode)
	assert.Equal(t, RuntimeModeObserve, preview.Proposed.SystemMode)
	assert.Zero(t, preview.EnabledGeofences)
	assert.Zero(t, preview.ActiveBindings)
	assert.Zero(t, preview.NewlyObservedDevices)

	updated, err := repository.UpdateSettings(
		ctx,
		proposed,
		actorID,
		now.Add(time.Minute),
	)
	require.NoError(t, err)
	assert.Equal(t, RuntimeModeObserve, updated.SystemMode)
	carrierSetting := findCarrierSetting(t, updated, carrier)
	assert.Equal(t, RuntimeModeObserve, carrierSetting.Mode)
	assert.Equal(t, RuntimeModeObserve, carrierSetting.EffectiveMode)
	assert.Equal(t, float64(150), carrierSetting.DefaultBaselineRadiusMeters)
	require.NotNil(t, carrierSetting.UpdatedBy)
	assert.Equal(t, actorID, *carrierSetting.UpdatedBy)

	var storedSystemMode RuntimeMode
	query, args, err := storage.Psql.
		Select("value").
		From("sys_configs").
		Where(sq.Eq{
			"category": geofenceConfigCategory,
			"key":      geofenceModeConfigKey,
		}).
		ToSql()
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, query, args...).Scan(&storedSystemMode))
	assert.Equal(t, RuntimeModeObserve, storedSystemMode)
}

func findCarrierSetting(
	t *testing.T,
	settings Settings,
	carrier string,
) CarrierSetting {
	t.Helper()
	for _, setting := range settings.Carriers {
		if setting.Carrier == carrier {
			return setting
		}
	}
	require.FailNow(t, "carrier setting not found", carrier)
	return CarrierSetting{}
}

func seedSettingsFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	carrier string,
	now time.Time,
) {
	t.Helper()
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
			RuntimeModeOff,
			"string",
			"integration test",
			false,
			now,
			now,
		).
		Suffix(
			"ON CONFLICT (category,key) DO UPDATE SET value = EXCLUDED.value, " +
				"updated_at = EXCLUDED.updated_at",
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Insert("geofence_carrier_settings").
		Columns(
			"carrier",
			"mode",
			"default_baseline_radius_meters",
			"updated_at",
		).
		Values(carrier, RuntimeModeOff, 100, now).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)
}

func deleteSettingsFixture(
	ctx context.Context,
	pool *pgxpool.Pool,
	carrier string,
) {
	query, args, err := storage.Psql.
		Delete("geofence_carrier_settings").
		Where(sq.Eq{"carrier": carrier}).
		ToSql()
	if err == nil {
		_, _ = pool.Exec(ctx, query, args...)
	}
	query, args, err = storage.Psql.
		Update("sys_configs").
		Set("value", RuntimeModeOff).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{
			"category": geofenceConfigCategory,
			"key":      geofenceModeConfigKey,
		}).
		ToSql()
	if err == nil {
		_, _ = pool.Exec(ctx, query, args...)
	}
}
