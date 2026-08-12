package geofence

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgRepositoryDefinitionLifecyclePreservesBindingsUntilArchiveIntegration(
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

	fixture := newLifecycleFixture()
	insertLifecycleFixture(t, ctx, pool, fixture)
	t.Cleanup(func() {
		deleteLifecycleFixture(context.Background(), pool, fixture)
	})

	repository := NewPgRepository(pool)
	service := NewService(repository, nil)

	disablePreview, err := service.PreviewDefinitionTransition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusDisabled,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(2), disablePreview.BindingCount)
	assert.Equal(t, int64(2), disablePreview.DeviceCount)

	err = service.TransitionDefinition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusDisabled,
		fixture.actorID,
		"maintenance window",
		disablePreview.PreviewFingerprint,
	)
	require.NoError(t, err)
	assertDefinitionStatus(t, ctx, pool, fixture.geofenceID, DefinitionStatusDisabled)
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.activeBindingID,
		BindingStatusActive,
		nil,
		"",
	)
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.suspendedBindingID,
		BindingStatusSuspended,
		nil,
		"",
	)

	archivePreview, err := service.PreviewDefinitionTransition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusArchived,
	)
	require.NoError(t, err)
	err = service.TransitionDefinition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusArchived,
		fixture.actorID,
		"retired coverage",
		archivePreview.PreviewFingerprint,
	)
	require.NoError(t, err)
	assertDefinitionStatus(t, ctx, pool, fixture.geofenceID, DefinitionStatusArchived)
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.activeBindingID,
		BindingStatusRemoved,
		&fixture.actorID,
		"retired coverage",
	)
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.suspendedBindingID,
		BindingStatusRemoved,
		&fixture.actorID,
		"retired coverage",
	)
}

func TestPgRepositoryDefinitionLifecycleRejectsStalePreviewIntegration(
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

	fixture := newLifecycleFixture()
	insertLifecycleFixture(t, ctx, pool, fixture)
	t.Cleanup(func() {
		deleteLifecycleFixture(context.Background(), pool, fixture)
	})

	repository := NewPgRepository(pool)
	service := NewService(repository, nil)
	preview, err := service.PreviewDefinitionTransition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusDisabled,
	)
	require.NoError(t, err)

	newBindingID := uuid.New()
	newDeviceID := uuid.New()
	insertLifecycleBinding(
		t,
		ctx,
		pool,
		newBindingID,
		newDeviceID,
		fixture.geofenceID,
		BindingStatusActive,
		fixture.actorID,
		fixture.now,
	)
	fixture.extraBindingIDs = append(fixture.extraBindingIDs, newBindingID)

	err = service.TransitionDefinition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusDisabled,
		fixture.actorID,
		"maintenance window",
		preview.PreviewFingerprint,
	)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrStaleLifecyclePreview))
	assertDefinitionStatus(t, ctx, pool, fixture.geofenceID, DefinitionStatusEnabled)
}

func TestPgRepositoryDefinitionLifecycleRejectsChangedCandidateSetIntegration(
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

	fixture := newLifecycleFixture()
	insertLifecycleFixture(t, ctx, pool, fixture)
	t.Cleanup(func() {
		deleteLifecycleFixture(context.Background(), pool, fixture)
	})

	_, err = pool.Exec(
		ctx,
		"UPDATE geofence_versions SET policy_json = $1 WHERE id = $2",
		json.RawMessage(`{"exit_action":"deactivate"}`),
		fixture.versionID,
	)
	require.NoError(t, err)
	_, err = pool.Exec(
		ctx,
		"UPDATE device_geofence_bindings SET status = 'active' WHERE id = $1",
		fixture.suspendedBindingID,
	)
	require.NoError(t, err)
	_, err = pool.Exec(
		ctx,
		`INSERT INTO device_geofence_states
            (binding_id, device_id, confirmed_state, last_observation_version)
          VALUES ($1, $2, 'inside', 7), ($3, $4, 'outside', 7)`,
		fixture.activeBindingID,
		fixture.activeDeviceID,
		fixture.suspendedBindingID,
		fixture.suspendedDeviceID,
	)
	require.NoError(t, err)

	service := NewService(NewPgRepository(pool), nil)
	preview, err := service.PreviewDefinitionTransition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusDisabled,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), preview.DeactivationDeviceCount)

	_, err = pool.Exec(
		ctx,
		`UPDATE device_geofence_states
          SET confirmed_state = CASE binding_id
            WHEN $1 THEN 'outside'
            WHEN $2 THEN 'inside'
          END
          WHERE binding_id IN ($1, $2)`,
		fixture.activeBindingID,
		fixture.suspendedBindingID,
	)
	require.NoError(t, err)

	err = service.TransitionDefinition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusDisabled,
		fixture.actorID,
		"candidate set changed",
		preview.PreviewFingerprint,
	)

	require.ErrorIs(t, err, ErrStaleLifecyclePreview)
	assertDefinitionStatus(t, ctx, pool, fixture.geofenceID, DefinitionStatusEnabled)
}

func TestPgRepositoryArchiveRejectsActiveManualBindJobIntegration(t *testing.T) {
	service, pool, fixture := lifecycleFixtureWithPendingBatchJob(t)
	ctx := context.Background()

	preview, err := service.PreviewDefinitionTransition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusArchived,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), preview.ActiveBatchJobCount)

	err = service.TransitionDefinition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusArchived,
		fixture.actorID,
		"retired",
		preview.PreviewFingerprint,
	)

	require.ErrorIs(t, err, ErrActiveBatchJobs)
	assertDefinitionStatus(t, ctx, pool, fixture.geofenceID, DefinitionStatusDisabled)
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.activeBindingID,
		BindingStatusActive,
		nil,
		"",
	)
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.suspendedBindingID,
		BindingStatusSuspended,
		nil,
		"",
	)
}

func TestPgRepositoryArchiveRejectsRunningManualBindJobIntegration(t *testing.T) {
	service, pool, fixture := lifecycleFixtureWithActiveManualBindJob(t, "running")
	ctx := context.Background()

	preview, err := service.PreviewDefinitionTransition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusArchived,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), preview.ActiveBatchJobCount)

	err = service.TransitionDefinition(
		ctx,
		fixture.geofenceID,
		DefinitionStatusArchived,
		fixture.actorID,
		"retired",
		preview.PreviewFingerprint,
	)

	require.ErrorIs(t, err, ErrActiveBatchJobs)
	assertDefinitionStatus(t, ctx, pool, fixture.geofenceID, DefinitionStatusDisabled)
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.activeBindingID,
		BindingStatusActive,
		nil,
		"",
	)
}

func TestPgRepositoryBindingLifecycleGuardsResumeAndSoftRemovesIntegration(
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

	fixture := newLifecycleFixture()
	insertLifecycleFixture(t, ctx, pool, fixture)
	t.Cleanup(func() {
		deleteLifecycleFixture(context.Background(), pool, fixture)
	})

	service := NewService(NewPgRepository(pool), nil)
	suspended, err := service.SuspendBinding(
		ctx,
		fixture.activeBindingID,
		fixture.actorID,
		"planned maintenance",
	)
	require.NoError(t, err)
	assert.Equal(t, BindingStatusSuspended, suspended.Status)
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.activeBindingID,
		BindingStatusSuspended,
		nil,
		"",
	)

	updateDefinitionStatus(
		t,
		ctx,
		pool,
		fixture.geofenceID,
		DefinitionStatusDisabled,
	)
	_, err = service.ResumeBinding(
		ctx,
		fixture.activeBindingID,
		fixture.actorID,
		"resume disabled fence check",
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.activeBindingID,
		BindingStatusSuspended,
		nil,
		"",
	)

	updateDefinitionStatus(
		t,
		ctx,
		pool,
		fixture.geofenceID,
		DefinitionStatusEnabled,
	)
	resumed, err := service.ResumeBinding(
		ctx,
		fixture.activeBindingID,
		fixture.actorID,
		"planned maintenance completed",
	)
	require.NoError(t, err)
	assert.Equal(t, BindingStatusActive, resumed.Status)

	removed, err := service.RemoveBinding(
		ctx,
		fixture.activeBindingID,
		fixture.actorID,
		"device reassigned",
	)
	require.NoError(t, err)
	assert.Equal(t, BindingStatusRemoved, removed.Status)
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		fixture.activeBindingID,
		BindingStatusRemoved,
		&fixture.actorID,
		"device reassigned",
	)

	_, err = service.ResumeBinding(
		ctx,
		fixture.activeBindingID,
		fixture.actorID,
		"resume removed binding check",
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestPgRepositoryBindingResumeMapsActiveRuleConflictIntegration(
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

	fixture := newLifecycleFixture()
	insertLifecycleFixture(t, ctx, pool, fixture)
	t.Cleanup(func() {
		deleteLifecycleFixture(context.Background(), pool, fixture)
	})

	conflictingBindingID := uuid.New()
	insertLifecycleBinding(
		t,
		ctx,
		pool,
		conflictingBindingID,
		fixture.activeDeviceID,
		fixture.geofenceID,
		BindingStatusSuspended,
		fixture.actorID,
		fixture.now,
	)
	fixture.extraBindingIDs = append(
		fixture.extraBindingIDs,
		conflictingBindingID,
	)

	_, err = NewService(
		NewPgRepository(pool),
		nil).
		ResumeBinding(
			ctx,
			conflictingBindingID,
			fixture.actorID,
			"resume conflict check",
		)

	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists))
	assertBindingLifecycle(
		t,
		ctx,
		pool,
		conflictingBindingID,
		BindingStatusSuspended,
		nil,
		"",
	)
}

type lifecycleFixture struct {
	actorID            uuid.UUID
	geofenceID         uuid.UUID
	versionID          uuid.UUID
	activeBindingID    uuid.UUID
	suspendedBindingID uuid.UUID
	activeDeviceID     uuid.UUID
	suspendedDeviceID  uuid.UUID
	extraBindingIDs    []uuid.UUID
	batchJobIDs        []uuid.UUID
	batchItemIDs       []uuid.UUID
	now                time.Time
}

func newLifecycleFixture() *lifecycleFixture {
	return &lifecycleFixture{
		actorID:            uuid.New(),
		geofenceID:         uuid.New(),
		versionID:          uuid.New(),
		activeBindingID:    uuid.New(),
		suspendedBindingID: uuid.New(),
		activeDeviceID:     uuid.New(),
		suspendedDeviceID:  uuid.New(),
		now:                time.Now().UTC().Truncate(time.Microsecond),
	}
}

func insertLifecycleFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture *lifecycleFixture,
) {
	t.Helper()
	deviceInsert := storage.Psql.
		Insert("devices").
		Columns("id", "serial_number", "oui", "carrier", "technology").
		Values(
			fixture.activeDeviceID,
			"LIFECYCLE-"+fixture.activeDeviceID.String(),
			"001122",
			"cmcc",
			"4G",
		).
		Values(
			fixture.suspendedDeviceID,
			"LIFECYCLE-"+fixture.suspendedDeviceID.String(),
			"001122",
			"cmcc",
			"4G",
		)
	query, args, err := deviceInsert.ToSql()
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
			fixture.geofenceID,
			"lifecycle-"+fixture.geofenceID.String(),
			"cmcc",
			RuleTypePolygonAllowZone,
			DefinitionStatusEnabled,
			fixture.actorID,
			fixture.actorID,
			fixture.now,
			fixture.now,
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
			fixture.versionID,
			fixture.geofenceID,
			1,
			VersionStatusPublished,
			json.RawMessage(
				`{"type":"Polygon","coordinates":[[[120,30],[121,30],`+
					`[121,31],[120,30]]]}`,
			),
			120,
			30,
			121,
			31,
			json.RawMessage(`{"exit_action":"notify_only"}`),
			fixture.actorID,
			fixture.actorID,
			fixture.now,
			fixture.now,
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Update("geofence_definitions").
		Set("current_version_id", fixture.versionID).
		Where(sq.Eq{"id": fixture.geofenceID}).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	insertLifecycleBinding(
		t,
		ctx,
		pool,
		fixture.activeBindingID,
		fixture.activeDeviceID,
		fixture.geofenceID,
		BindingStatusActive,
		fixture.actorID,
		fixture.now,
	)
	insertLifecycleBinding(
		t,
		ctx,
		pool,
		fixture.suspendedBindingID,
		fixture.suspendedDeviceID,
		fixture.geofenceID,
		BindingStatusSuspended,
		fixture.actorID,
		fixture.now,
	)
}

func lifecycleFixtureWithPendingBatchJob(
	t *testing.T,
) (*Service, *pgxpool.Pool, *lifecycleFixture) {
	return lifecycleFixtureWithActiveManualBindJob(t, "pending")
}

func lifecycleFixtureWithActiveManualBindJob(
	t *testing.T,
	activeJobStatus string,
) (*Service, *pgxpool.Pool, *lifecycleFixture) {
	t.Helper()
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	fixture := newLifecycleFixture()
	insertLifecycleFixture(t, ctx, pool, fixture)
	updateDefinitionStatus(
		t,
		ctx,
		pool,
		fixture.geofenceID,
		DefinitionStatusDisabled,
	)
	t.Cleanup(func() {
		deleteLifecycleFixture(context.Background(), pool, fixture)
	})

	insertLifecycleBatchJob(
		t,
		ctx,
		pool,
		fixture,
		ManualBindJobType,
		activeJobStatus,
		fixture.geofenceID,
	)
	insertLifecycleBatchJob(
		t,
		ctx,
		pool,
		fixture,
		ManualBindJobType,
		"pending",
		uuid.New(),
	)
	insertLifecycleBatchJob(
		t,
		ctx,
		pool,
		fixture,
		"unrelated_job",
		"pending",
		fixture.geofenceID,
	)
	insertLifecycleBatchJob(
		t,
		ctx,
		pool,
		fixture,
		ManualBindJobType,
		"succeeded",
		fixture.geofenceID,
	)

	return NewService(NewPgRepository(pool), nil), pool, fixture
}

func insertLifecycleBatchJob(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture *lifecycleFixture,
	jobType string,
	status string,
	payloadGeofenceID uuid.UUID,
) {
	t.Helper()
	jobID := uuid.New()
	payload, err := json.Marshal(ManualBindJobPayload{
		SchemaVersion:      ManualBindPayloadVersion,
		GeofenceID:         payloadGeofenceID,
		GeofenceVersionID:  fixture.versionID,
		RequestedBy:        fixture.actorID,
		PreviewFingerprint: jobID.String(),
		Reason:             "lifecycle integration test",
	})
	require.NoError(t, err)
	query, args, err := storage.Psql.
		Insert("async_jobs").
		Columns("id", "job_type", "status", "scheduled_at", "payload", "max_attempts").
		Values(jobID, jobType, status, fixture.now, json.RawMessage(payload), 3).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)
	fixture.batchJobIDs = append(fixture.batchJobIDs, jobID)

	itemID := uuid.New()
	query, args, err = storage.Psql.
		Insert("geofence_batch_items").
		Columns(
			"id",
			"job_id",
			"geofence_id",
			"input_key",
			"input_kind",
			"input_value",
			"status",
		).
		Values(
			itemID,
			jobID,
			fixture.geofenceID,
			jobID.String(),
			BindingInputDeviceSN,
			"lifecycle-test-device",
			BatchItemPending,
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)
	fixture.batchItemIDs = append(fixture.batchItemIDs, itemID)
}

func insertLifecycleBinding(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	bindingID uuid.UUID,
	deviceID uuid.UUID,
	geofenceID uuid.UUID,
	status BindingStatus,
	actorID uuid.UUID,
	boundAt time.Time,
) {
	t.Helper()
	query, args, err := storage.Psql.
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
			status,
			"manual",
			actorID,
			boundAt,
		).
		ToSql()
	require.NoError(t, err)
	_, err = pool.Exec(ctx, query, args...)
	require.NoError(t, err)
}

func assertDefinitionStatus(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	geofenceID uuid.UUID,
	expected DefinitionStatus,
) {
	t.Helper()
	query, args, err := storage.Psql.
		Select("status").
		From("geofence_definitions").
		Where(sq.Eq{"id": geofenceID}).
		ToSql()
	require.NoError(t, err)
	var actual DefinitionStatus
	require.NoError(t, pool.QueryRow(ctx, query, args...).Scan(&actual))
	assert.Equal(t, expected, actual)
}

func updateDefinitionStatus(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	geofenceID uuid.UUID,
	status DefinitionStatus,
) {
	t.Helper()
	query, args, err := storage.Psql.
		Update("geofence_definitions").
		Set("status", status).
		Where(sq.Eq{"id": geofenceID}).
		ToSql()
	require.NoError(t, err)
	tag, err := pool.Exec(ctx, query, args...)
	require.NoError(t, err)
	require.Equal(t, int64(1), tag.RowsAffected())
}

func assertBindingLifecycle(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	bindingID uuid.UUID,
	expectedStatus BindingStatus,
	expectedRemovedBy *uuid.UUID,
	expectedReason string,
) {
	t.Helper()
	query, args, err := storage.Psql.
		Select("status", "removed_by", "removed_at", "remove_reason").
		From("device_geofence_bindings").
		Where(sq.Eq{"id": bindingID}).
		ToSql()
	require.NoError(t, err)
	var (
		actualStatus    BindingStatus
		actualRemovedBy *uuid.UUID
		actualRemovedAt *time.Time
		actualReason    *string
	)
	require.NoError(
		t,
		pool.QueryRow(ctx, query, args...).Scan(
			&actualStatus,
			&actualRemovedBy,
			&actualRemovedAt,
			&actualReason,
		),
	)
	assert.Equal(t, expectedStatus, actualStatus)
	assert.Equal(t, expectedRemovedBy, actualRemovedBy)
	if expectedRemovedBy == nil {
		assert.Nil(t, actualRemovedAt)
		assert.Nil(t, actualReason)
		return
	}
	assert.NotNil(t, actualRemovedAt)
	require.NotNil(t, actualReason)
	assert.Equal(t, expectedReason, *actualReason)
}

func deleteLifecycleFixture(
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture *lifecycleFixture,
) {
	deleteBy := func(table string, where sq.Sqlizer) {
		query, args, err := storage.Psql.Delete(table).Where(where).ToSql()
		if err == nil {
			_, _ = pool.Exec(ctx, query, args...)
		}
	}
	deleteBy("geofence_batch_items", sq.Eq{"id": fixture.batchItemIDs})
	deleteBy("async_jobs", sq.Eq{"id": fixture.batchJobIDs})
	deleteBy("device_geofence_states", sq.Eq{
		"binding_id": append(
			[]uuid.UUID{fixture.activeBindingID, fixture.suspendedBindingID},
			fixture.extraBindingIDs...,
		),
	})
	bindingIDs := []uuid.UUID{
		fixture.activeBindingID,
		fixture.suspendedBindingID,
	}
	bindingIDs = append(bindingIDs, fixture.extraBindingIDs...)
	deleteBy("device_geofence_bindings", sq.Eq{"id": bindingIDs})
	query, args, err := storage.Psql.
		Update("geofence_definitions").
		Set("current_version_id", nil).
		Where(sq.Eq{"id": fixture.geofenceID}).
		ToSql()
	if err == nil {
		_, _ = pool.Exec(ctx, query, args...)
	}
	deleteBy("geofence_versions", sq.Eq{"id": fixture.versionID})
	deleteBy("geofence_definitions", sq.Eq{"id": fixture.geofenceID})
	deleteBy("devices", sq.Eq{"id": []uuid.UUID{
		fixture.activeDeviceID,
		fixture.suspendedDeviceID,
	}})
}
