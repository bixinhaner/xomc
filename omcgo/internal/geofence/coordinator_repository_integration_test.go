package geofence

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgCoordinatorRepositoryAtomicEvaluationAndReplayProtectionIntegration(
	t *testing.T,
) {
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	now := time.Now().UTC().Truncate(time.Microsecond)
	actorID := uuid.New()
	deviceID := uuid.New()
	geofenceID := uuid.New()
	versionID := uuid.New()
	bindingID := uuid.New()
	carrier := "t" + uuid.NewString()[:8]

	insertCoordinatorFixture(
		t,
		ctx,
		pool,
		actorID,
		deviceID,
		geofenceID,
		versionID,
		bindingID,
		carrier,
		now,
	)
	t.Cleanup(func() {
		deleteCoordinatorFixture(
			context.Background(),
			pool,
			deviceID,
			geofenceID,
			versionID,
			bindingID,
			carrier,
		)
	})

	payload := event.DeviceLocationObservedPayload{
		DeviceID:           deviceID,
		SerialNumber:       "COORDINATOR-INTEGRATION",
		Carrier:            carrier,
		ObservationVersion: 1,
		Latitude:           30.5,
		Longitude:          120.5,
		ObservedAt:         now,
		ReceivedAt:         now,
		SourcePath:         "integration",
	}
	repository := NewPgCoordinatorRepository(storage.NewPoolDB(pool))

	first, err := repository.EvaluateLocation(ctx, payload, now)
	require.NoError(t, err)
	assert.True(t, first.Processed)
	assert.False(t, first.NoOp)
	assert.Equal(t, 1, first.EvaluationCount)
	assert.Equal(t, EffectiveStateOutside, first.EffectiveState)
	assert.Equal(t, int64(2), first.EffectiveStateVersion)

	duplicate, err := repository.EvaluateLocation(
		ctx,
		payload,
		now.Add(time.Second),
	)
	require.NoError(t, err)
	assert.True(t, duplicate.NoOp)
	assert.Equal(t, "duplicate_observation", duplicate.Reason)

	var evaluationCount int
	query, args, err := storage.Psql.
		Select("COUNT(*)").
		From("geofence_evaluations").
		Where(sq.Eq{"binding_id": bindingID}).
		ToSql()
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, query, args...).Scan(&evaluationCount))
	assert.Equal(t, 1, evaluationCount)

	var confirmedState ConfirmedState
	var bindingObservationVersion int64
	query, args, err = storage.Psql.
		Select("confirmed_state", "last_observation_version").
		From("device_geofence_states").
		Where(sq.Eq{"binding_id": bindingID}).
		ToSql()
	require.NoError(t, err)
	require.NoError(
		t,
		pool.QueryRow(ctx, query, args...).Scan(
			&confirmedState,
			&bindingObservationVersion,
		),
	)
	assert.Equal(t, ConfirmedStateOutside, confirmedState)
	assert.Equal(t, int64(1), bindingObservationVersion)

	var effectiveState EffectiveState
	var effectiveVersion, effectiveObservationVersion int64
	query, args, err = storage.Psql.
		Select(
			"effective_state",
			"state_version",
			"last_observation_version",
		).
		From("device_geofence_effective_states").
		Where(sq.Eq{"device_id": deviceID}).
		ToSql()
	require.NoError(t, err)
	require.NoError(
		t,
		pool.QueryRow(ctx, query, args...).Scan(
			&effectiveState,
			&effectiveVersion,
			&effectiveObservationVersion,
		),
	)
	assert.Equal(t, EffectiveStateOutside, effectiveState)
	assert.Equal(t, int64(2), effectiveVersion)
	assert.Equal(t, int64(1), effectiveObservationVersion)

	var outboxCount int
	query, args, err = storage.Psql.
		Select("COUNT(*)").
		From("event_outbox").
		Where(sq.Like{"dedupe_key": "%:" + bindingID.String() + ":%"}).
		ToSql()
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, query, args...).Scan(&outboxCount))
	assert.Equal(t, 2, outboxCount, "evaluation and rule-edge events")

	query, args, err = storage.Psql.
		Select("COUNT(*)").
		From("event_outbox").
		Where(sq.Like{
			"dedupe_key": event.SubjectGeofenceDeviceExited + ":" +
				deviceID.String() + ":%",
		}).
		ToSql()
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, query, args...).Scan(&outboxCount))
	assert.Equal(t, 1, outboxCount, "one device edge across duplicate delivery")
}

func TestPgCoordinatorRepositoryPersistsDeterministicFailureWithoutChangingConfirmedStateIntegration(
	t *testing.T,
) {
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	now := time.Now().UTC().Truncate(time.Microsecond)
	actorID := uuid.New()
	deviceID := uuid.New()
	geofenceID := uuid.New()
	versionID := uuid.New()
	bindingID := uuid.New()
	carrier := "t" + uuid.NewString()[:8]
	insertCoordinatorFixture(
		t,
		ctx,
		pool,
		actorID,
		deviceID,
		geofenceID,
		versionID,
		bindingID,
		carrier,
		now,
	)
	t.Cleanup(func() {
		deleteCoordinatorFixture(
			context.Background(),
			pool,
			deviceID,
			geofenceID,
			versionID,
			bindingID,
			carrier,
		)
	})

	query, args, err := storage.Psql.
		Update("geofence_versions").
		Set("geometry_json", json.RawMessage(`{"type":"Point"}`)).
		Where(sq.Eq{"id": versionID}).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	result, err := NewPgCoordinatorRepository(
		storage.NewPoolDB(pool),
	).EvaluateLocation(
		ctx,
		event.DeviceLocationObservedPayload{
			DeviceID:           deviceID,
			SerialNumber:       "COORDINATOR-FAILURE",
			Carrier:            carrier,
			ObservationVersion: 1,
			Latitude:           30.5,
			Longitude:          120.5,
			ObservedAt:         now,
			ReceivedAt:         now,
			SourcePath:         "integration",
		},
		now,
	)
	require.NoError(t, err)
	assert.True(t, result.Processed)
	assert.True(t, result.Failed)
	assert.Equal(t, "invalid_geometry", result.Reason)
	assert.Equal(t, EffectiveStateUnknown, result.EffectiveState)
	assert.Equal(t, int64(1), result.EffectiveStateVersion)

	var status evaluationStatus
	var failureStage, errorCode, sourcePath string
	query, args, err = storage.Psql.
		Select("status", "failure_stage", "error_code", "source_path").
		From("geofence_evaluations").
		Where(sq.Eq{"binding_id": bindingID}).
		ToSql()
	require.NoError(t, err)
	require.NoError(
		t,
		pool.QueryRow(ctx, query, args...).Scan(
			&status,
			&failureStage,
			&errorCode,
			&sourcePath,
		),
	)
	assert.Equal(t, evaluationStatusFailed, status)
	assert.Equal(t, "geometry_evaluation", failureStage)
	assert.Equal(t, "invalid_geometry", errorCode)
	assert.Equal(t, "integration", sourcePath)

	var confirmedState ConfirmedState
	var lastObservationVersion *int64
	query, args, err = storage.Psql.
		Select("confirmed_state", "last_observation_version").
		From("device_geofence_states").
		Where(sq.Eq{"binding_id": bindingID}).
		ToSql()
	require.NoError(t, err)
	require.NoError(
		t,
		pool.QueryRow(ctx, query, args...).Scan(
			&confirmedState,
			&lastObservationVersion,
		),
	)
	assert.Equal(t, ConfirmedStateUnknown, confirmedState)
	assert.Nil(t, lastObservationVersion, "failed evaluation must not advance binding state")

	var health evaluationHealth
	var effectiveObservationVersion int64
	query, args, err = storage.Psql.
		Select("evaluation_health", "last_observation_version").
		From("device_geofence_effective_states").
		Where(sq.Eq{"device_id": deviceID}).
		ToSql()
	require.NoError(t, err)
	require.NoError(
		t,
		pool.QueryRow(ctx, query, args...).Scan(
			&health,
			&effectiveObservationVersion,
		),
	)
	assert.Equal(t, evaluationHealthFailed, health)
	assert.Equal(t, int64(1), effectiveObservationVersion)
}

func TestPgCoordinatorRepositoryRequiresSystemAndCarrierObserveIntegration(
	t *testing.T,
) {
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	tests := []struct {
		name        string
		systemMode  RuntimeMode
		carrierMode RuntimeMode
		wantCount   int
	}{
		{
			name:       "system off carrier observe",
			systemMode: RuntimeModeOff, carrierMode: RuntimeModeObserve,
		},
		{
			name:       "system observe carrier off",
			systemMode: RuntimeModeObserve, carrierMode: RuntimeModeOff,
		},
		{
			name:       "both observe",
			systemMode: RuntimeModeObserve, carrierMode: RuntimeModeObserve,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().UTC().Truncate(time.Microsecond)
			actorID := uuid.New()
			deviceID := uuid.New()
			geofenceID := uuid.New()
			versionID := uuid.New()
			bindingID := uuid.New()
			carrier := "m" + uuid.NewString()[:8]
			insertCoordinatorFixture(
				t,
				ctx,
				pool,
				actorID,
				deviceID,
				geofenceID,
				versionID,
				bindingID,
				carrier,
				now,
			)
			t.Cleanup(func() {
				deleteCoordinatorFixture(
					context.Background(),
					pool,
					deviceID,
					geofenceID,
					versionID,
					bindingID,
					carrier,
				)
			})
			setCoordinatorModes(
				t,
				ctx,
				pool,
				carrier,
				tt.systemMode,
				tt.carrierMode,
			)

			result, err := NewPgCoordinatorRepository(
				storage.NewPoolDB(pool),
			).EvaluateLocation(
				ctx,
				event.DeviceLocationObservedPayload{
					DeviceID:           deviceID,
					SerialNumber:       "COORDINATOR-MODE-INTEGRATION",
					Carrier:            carrier,
					ObservationVersion: 1,
					Latitude:           30.5,
					Longitude:          120.5,
					ObservedAt:         now,
					ReceivedAt:         now,
					SourcePath:         "integration",
				},
				now,
			)

			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, result.EvaluationCount)
			if tt.wantCount == 0 {
				assert.Equal(t, "no_active_bindings", result.Reason)
				assert.Equal(t, EffectiveStateUnmanaged, result.EffectiveState)
			} else {
				assert.Equal(t, EffectiveStateOutside, result.EffectiveState)
			}
		})
	}
}

func insertCoordinatorFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	actorID uuid.UUID,
	deviceID uuid.UUID,
	geofenceID uuid.UUID,
	versionID uuid.UUID,
	bindingID uuid.UUID,
	carrier string,
	now time.Time,
) {
	t.Helper()
	setSystemMode(t, ctx, pool, RuntimeModeObserve)
	query, args, err := storage.Psql.
		Insert("geofence_carrier_settings").
		Columns("carrier", "mode", "updated_by", "updated_at").
		Values(carrier, "observe", actorID, now).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Insert("geofence_definitions").
		Columns(
			"id",
			"name",
			"carrier",
			"rule_type",
			"status",
			"created_by",
			"updated_by",
			"created_at",
			"updated_at",
		).
		Values(
			geofenceID,
			"coordinator-"+geofenceID.String(),
			carrier,
			RuleTypePolygonAllowZone,
			DefinitionStatusEnabled,
			actorID,
			actorID,
			now,
			now,
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Insert("geofence_versions").
		Columns(
			"id",
			"geofence_id",
			"version",
			"status",
			"geometry_json",
			"bbox_min_longitude",
			"bbox_min_latitude",
			"bbox_max_longitude",
			"bbox_max_latitude",
			"policy_json",
			"created_by",
			"published_by",
			"created_at",
			"published_at",
		).
		Values(
			versionID,
			geofenceID,
			1,
			VersionStatusPublished,
			json.RawMessage(
				`{"type":"Polygon","coordinates":[[[120,30],[120.1,30],`+
					`[120.1,30.1],[120,30.1],[120,30]]]}`,
			),
			120.0,
			30.0,
			120.1,
			30.1,
			json.RawMessage(
				`{"exit_action":"notify_only","exit_consecutive_samples":1,`+
					`"minimum_state_duration_seconds":0}`,
			),
			actorID,
			actorID,
			now,
			now,
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Update("geofence_definitions").
		Set("current_version_id", versionID).
		Where(sq.Eq{"id": geofenceID}).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Insert("device_geofence_bindings").
		Columns(
			"id",
			"device_id",
			"geofence_id",
			"rule_type",
			"status",
			"bind_source",
			"bound_by",
			"bound_at",
		).
		Values(
			bindingID,
			deviceID,
			geofenceID,
			RuleTypePolygonAllowZone,
			BindingStatusActive,
			"manual",
			actorID,
			now,
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Insert("device_geofence_states").
		Columns(
			"binding_id",
			"device_id",
			"confirmed_state",
			"candidate_count",
			"state_version",
		).
		Values(
			bindingID,
			deviceID,
			ConfirmedStateUnknown,
			0,
			1,
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Insert("device_geofence_effective_states").
		Columns(
			"device_id",
			"effective_state",
			"required_action_level",
			"state_version",
			"evaluation_health",
		).
		Values(
			deviceID,
			EffectiveStateUnknown,
			ActionLevelNone,
			1,
			evaluationHealthHealthy,
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)
}

func deleteCoordinatorFixture(
	ctx context.Context,
	pool *pgxpool.Pool,
	deviceID uuid.UUID,
	geofenceID uuid.UUID,
	versionID uuid.UUID,
	bindingID uuid.UUID,
	carrier string,
) {
	deleteBy := func(table string, where sq.Sqlizer) {
		query, args, err := storage.Psql.Delete(table).Where(where).ToSql()
		if err == nil {
			_, _ = pool.Exec(ctx, query, args...)
		}
	}
	deleteBy(
		"event_outbox",
		sq.Or{
			sq.Like{"dedupe_key": "%:" + bindingID.String() + ":%"},
			sq.Like{"dedupe_key": "%:" + deviceID.String() + ":%"},
		},
	)
	deleteBy("device_geofence_effective_states", sq.Eq{"device_id": deviceID})
	deleteBy("device_geofence_states", sq.Eq{"binding_id": bindingID})
	deleteBy("geofence_evaluations", sq.Eq{"binding_id": bindingID})
	deleteBy("device_geofence_bindings", sq.Eq{"id": bindingID})
	query, args, err := storage.Psql.
		Update("geofence_definitions").
		Set("current_version_id", nil).
		Where(sq.Eq{"id": geofenceID}).
		ToSql()
	if err == nil {
		_, _ = pool.Exec(ctx, query, args...)
	}
	deleteBy("geofence_versions", sq.Eq{"id": versionID})
	deleteBy("geofence_definitions", sq.Eq{"id": geofenceID})
	deleteBy("geofence_carrier_settings", sq.Eq{"carrier": carrier})
	setSystemModeValue(ctx, pool, RuntimeModeOff)
}

func setCoordinatorModes(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	carrier string,
	systemMode RuntimeMode,
	carrierMode RuntimeMode,
) {
	t.Helper()
	setSystemMode(t, ctx, pool, systemMode)
	query, args, err := storage.Psql.
		Update("geofence_carrier_settings").
		Set("mode", carrierMode).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"carrier": carrier}).
		ToSql()
	require.NoError(t, err)
	tag, err := pool.Exec(ctx, query, args...)
	require.NoError(t, err)
	require.Equal(t, int64(1), tag.RowsAffected())
}

func setSystemMode(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	mode RuntimeMode,
) {
	t.Helper()
	require.NoError(t, setSystemModeValue(ctx, pool, mode))
}

func setSystemModeValue(
	ctx context.Context,
	pool *pgxpool.Pool,
	mode RuntimeMode,
) error {
	query, args, err := storage.Psql.
		Insert("sys_configs").
		Columns(
			"category",
			"key",
			"value",
			"value_type",
			"description",
			"is_public",
		).
		Values(
			geofenceConfigCategory,
			geofenceModeConfigKey,
			mode,
			"string",
			"coordinator integration",
			false,
		).
		Suffix(
			"ON CONFLICT (category,key) DO UPDATE SET value = EXCLUDED.value, " +
				"updated_at = NOW()",
		).
		ToSql()
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, query, args...)
	return err
}
