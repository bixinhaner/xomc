package geofence

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/require"
)

func TestPgRepositoryCreateManualBindJobIsAtomicAndIdempotentIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	req := fixture.validCreateJobParams(t)

	first, err := repo.CreateManualBindJob(context.Background(), req)
	require.NoError(t, err)
	second, err := repo.CreateManualBindJob(context.Background(), req)
	require.NoError(t, err)

	require.Equal(t, first.JobID, second.JobID)
	require.Equal(t, 1, fixture.countJobsByFingerprint(t, req.PreviewFingerprint))
	require.Equal(t, len(req.Inputs), fixture.countItems(t, first.JobID))
	fixture.requireIdempotencyPayload(t, first.JobID)
}

func TestPgRepositoryCreateManualBindJobConcurrentConfirmationsReuseJobIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	req := fixture.validCreateJobParams(t)

	const confirmations = 4
	results := make(chan BatchJobAccepted, confirmations)
	errs := make(chan error, confirmations)
	var wait sync.WaitGroup
	for range confirmations {
		wait.Add(1)
		go func() {
			defer wait.Done()
			accepted, err := repo.CreateManualBindJob(context.Background(), req)
			results <- accepted
			errs <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	var jobID uuid.UUID
	for result := range results {
		if jobID == uuid.Nil {
			jobID = result.JobID
		}
		require.Equal(t, jobID, result.JobID)
	}
	require.Equal(t, 1, fixture.countJobsByFingerprint(t, req.PreviewFingerprint))
	require.Equal(t, len(req.Inputs), fixture.countItems(t, jobID))
}

func TestPgRepositoryCreateManualBindJobReplaysOriginalConfirmationAfterSuccessIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	req := fixture.validCreateJobParams(t)
	first, err := repo.CreateManualBindJob(context.Background(), req)
	require.NoError(t, err)

	registry := asyncjob.NewRegistry(
		asyncjob.NewPgRepository(fixture.pool),
		"manual-bind-success-replay",
		nil,
	)
	registry.Register(NewManualBindRunner(repo, nil))
	ran, err := registry.RunNext(context.Background(), ManualBindJobType)
	require.NoError(t, err)
	require.True(t, ran)
	job, err := repo.GetManualBindJob(context.Background(), first.JobID)
	require.NoError(t, err)
	require.NotNil(t, job)
	require.Equal(t, asyncjob.StatusSucceeded, job.Status)

	replayed, err := repo.CreateManualBindJob(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, first.JobID, replayed.JobID)
	require.Equal(t, 1, fixture.countJobsByFingerprint(t, req.PreviewFingerprint))
	require.Equal(t, len(req.Inputs), fixture.countItems(t, first.JobID))
}

func TestPgRepositoryCreateManualBindJobReevaluatesAfterNonReusableTerminalStatusIntegration(
	t *testing.T,
) {
	for _, status := range []asyncjob.Status{
		asyncjob.StatusFailed,
		asyncjob.StatusCanceled,
		asyncjob.StatusZombie,
	} {
		t.Run(string(status), func(t *testing.T) {
			repo, fixture := newBatchRepositoryFixture(t)
			req := fixture.validCreateJobParams(t)
			first, err := repo.CreateManualBindJob(context.Background(), req)
			require.NoError(t, err)
			_, err = fixture.pool.Exec(
				context.Background(),
				`UPDATE async_jobs
				    SET status = $2, finished_at = $3
				  WHERE id = $1`,
				first.JobID,
				status,
				fixture.now,
			)
			require.NoError(t, err)

			second, err := repo.CreateManualBindJob(context.Background(), req)

			require.NoError(t, err)
			require.NotEqual(t, first.JobID, second.JobID)
			require.Equal(
				t,
				2,
				fixture.countJobsByFingerprint(t, req.PreviewFingerprint),
			)
			require.Equal(t, len(req.Inputs), fixture.countItems(t, second.JobID))
		})
	}
}

func TestPgRepositoryCreateManualBindJobKeepsStaleProtectionAfterFailedJobIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	req := fixture.validCreateJobParams(t)
	first, err := repo.CreateManualBindJob(context.Background(), req)
	require.NoError(t, err)
	_, err = fixture.pool.Exec(
		context.Background(),
		`UPDATE async_jobs
		    SET status = 'failed', finished_at = $2
		  WHERE id = $1`,
		first.JobID,
		fixture.now,
	)
	require.NoError(t, err)
	fixture.insertConflictingActiveBinding(t, fixture.deviceIDs[0])

	_, err = repo.CreateManualBindJob(context.Background(), req)

	require.ErrorIs(t, err, ErrStaleBindingPreview)
	require.Equal(t, 1, fixture.countJobsByFingerprint(t, req.PreviewFingerprint))
}

func TestPgRepositoryCreateManualBindJobRejectsStalePreviewIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	req := fixture.validCreateJobParams(t)
	fixture.insertConflictingActiveBinding(t, fixture.deviceIDs[0])

	_, err := repo.CreateManualBindJob(context.Background(), req)

	require.ErrorIs(t, err, ErrStaleBindingPreview)
	require.Zero(t, fixture.countJobsByFingerprint(t, req.PreviewFingerprint))
}

func TestPgRepositoryCreateManualBindJobPersistsEligibleAndSkippedItemsIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	missingSN := "MISSING-" + uuid.NewString()
	req := fixture.validCreateJobParams(t)
	req.Inputs = append(req.Inputs, BindingInput{
		Key: "sn:" + missingSN, Kind: BindingInputDeviceSN, Value: missingSN,
	})
	snapshot, err := repo.LoadManualBindSnapshot(
		context.Background(),
		req.GeofenceID,
		req.Inputs,
		req.VisibleGroups,
	)
	require.NoError(t, err)
	preview, err := buildManualBindingPreview(snapshot)
	require.NoError(t, err)
	req.PreviewFingerprint = preview.PreviewFingerprint

	accepted, err := repo.CreateManualBindJob(context.Background(), req)
	require.NoError(t, err)

	var pending, skipped int
	err = fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FILTER (WHERE status = 'pending'),
		        COUNT(*) FILTER (WHERE status = 'skipped')
		   FROM geofence_batch_items
		  WHERE job_id = $1`,
		accepted.JobID,
	).Scan(&pending, &skipped)
	require.NoError(t, err)
	require.Equal(t, len(fixture.deviceIDs), pending)
	require.Equal(t, 1, skipped)
}

func TestPgRepositoryManualBindJobProgressAndVisibilityRedactionIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	req := fixture.validCreateJobParams(t)
	accepted, err := repo.CreateManualBindJob(context.Background(), req)
	require.NoError(t, err)

	job, err := repo.GetManualBindJob(context.Background(), accepted.JobID)
	require.NoError(t, err)
	require.Equal(t, int64(len(req.Inputs)), job.Progress.Total)
	require.Equal(t, int64(len(req.Inputs)), job.Progress.Pending)

	visible, err := repo.ListManualBindItems(
		context.Background(),
		BatchItemFilter{
			JobID: accepted.JobID, Page: 2, PageSize: 1, VisibleGroups: nil,
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(len(req.Inputs)), visible.Total)
	require.Len(t, visible.Items, 1)
	require.NotNil(t, visible.Items[0].DeviceID)
	require.NotEmpty(t, visible.Items[0].InputValue)
	require.NotEmpty(t, visible.Items[0].DeviceSNSnapshot)

	hidden, err := repo.ListManualBindItems(
		context.Background(),
		BatchItemFilter{
			JobID: accepted.JobID, Page: 1, PageSize: 1,
			VisibleGroups: []uuid.UUID{},
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(len(req.Inputs)), hidden.Total)
	require.Len(t, hidden.Items, 1)
	require.Nil(t, hidden.Items[0].DeviceID)
	require.Empty(t, hidden.Items[0].InputKey)
	require.Empty(t, hidden.Items[0].InputValue)
	require.Empty(t, hidden.Items[0].DeviceSNSnapshot)
	require.Empty(t, hidden.Items[0].ErrorMessage)
}

func TestPgRepositoryCreateManualBindJobRollsBackJobWhenItemInsertFailsIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	req := fixture.validCreateJobParams(t)
	req.Inputs[0].Value = strings.Repeat("x", 256)

	_, err := repo.CreateManualBindJob(context.Background(), req)

	require.Error(t, err)
	require.Zero(t, fixture.countJobsByFingerprint(t, req.PreviewFingerprint))
	var itemCount int
	err = fixture.pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM geofence_batch_items WHERE geofence_id = $1",
		fixture.geofenceID,
	).Scan(&itemCount)
	require.NoError(t, err)
	require.Zero(t, itemCount)
}

func TestPgRepositoryManualBindJobAndCreateBindingLinearizeOnSharedLocksIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	req := fixture.validCreateJobParams(t)
	definitionLocked := make(chan struct{})
	releaseBatch := make(chan struct{})
	repo.hooks = &pgRepositoryHooks{
		afterManualBindDefinitionLock: func(context.Context) error {
			close(definitionLocked)
			<-releaseBatch
			return nil
		},
	}

	type jobResult struct {
		accepted BatchJobAccepted
		err      error
	}
	jobResults := make(chan jobResult, 1)
	go func() {
		accepted, err := repo.CreateManualBindJob(context.Background(), req)
		jobResults <- jobResult{accepted: accepted, err: err}
	}()
	<-definitionLocked

	binding := &Binding{
		ID:         uuid.New(),
		DeviceID:   fixture.deviceIDs[0],
		GeofenceID: fixture.geofenceID,
		RuleType:   RuleTypePolygonAllowZone,
		Status:     BindingStatusActive,
		BindSource: "manual",
		BoundBy:    fixture.actorID,
		BoundAt:    fixture.now,
	}
	bindingResults := make(chan error, 1)
	go func() {
		bindingResults <- repo.CreateBinding(context.Background(), binding)
	}()
	close(releaseBatch)

	job := <-jobResults
	require.NoError(t, job.err)
	require.NoError(t, <-bindingResults)
	require.NotEqual(t, uuid.Nil, job.accepted.JobID)

	currentSnapshot, err := repo.LoadManualBindSnapshot(
		context.Background(),
		req.GeofenceID,
		req.Inputs,
		req.VisibleGroups,
	)
	require.NoError(t, err)
	currentPreview, err := buildManualBindingPreview(currentSnapshot)
	require.NoError(t, err)
	require.NotEqual(t, req.PreviewFingerprint, currentPreview.PreviewFingerprint)
}

func TestPgRepositoryManualBindJobRejectsPreviewWhenResumeLinearizesFirstIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	bindingID := uuid.New()
	fixture.insertBinding(
		t,
		bindingID,
		fixture.deviceIDs[0],
		BindingStatusSuspended,
	)
	req := fixture.validCreateJobParams(t)

	definitionLocked := make(chan struct{})
	releaseResume := make(chan struct{})
	repo.hooks = &pgRepositoryHooks{
		afterBindingDefinitionLock: func(context.Context) error {
			close(definitionLocked)
			<-releaseResume
			return nil
		},
	}
	resumeResults := make(chan error, 1)
	go func() {
		_, err := repo.TransitionBinding(
			context.Background(),
			bindingID,
			BindingStatusActive,
			fixture.actorID,
			"",
			fixture.now,
		)
		resumeResults <- err
	}()
	<-definitionLocked

	jobResults := make(chan error, 1)
	go func() {
		_, err := repo.CreateManualBindJob(context.Background(), req)
		jobResults <- err
	}()
	close(releaseResume)

	require.NoError(t, <-resumeResults)
	require.ErrorIs(t, <-jobResults, ErrStaleBindingPreview)
	require.Zero(t, fixture.countJobsByFingerprint(t, req.PreviewFingerprint))
}

type batchRepositoryFixture struct {
	pool             *pgxpool.Pool
	repo             *PgRepository
	actorID          uuid.UUID
	geofenceID       uuid.UUID
	versionID        uuid.UUID
	deviceIDs        []uuid.UUID
	now              time.Time
	otherGeofenceIDs []uuid.UUID
	otherBindingIDs  []uuid.UUID
}

func newBatchRepositoryFixture(
	t *testing.T,
) (*PgRepository, *batchRepositoryFixture) {
	t.Helper()
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	fixture := &batchRepositoryFixture{
		pool:       pool,
		repo:       NewPgRepository(pool),
		actorID:    uuid.New(),
		geofenceID: uuid.New(),
		versionID:  uuid.New(),
		deviceIDs:  []uuid.UUID{uuid.New(), uuid.New()},
		now:        time.Now().UTC().Truncate(time.Microsecond),
	}
	fixture.insert(t)
	t.Cleanup(func() {
		fixture.cleanup(context.Background())
	})
	return fixture.repo, fixture
}

func (f *batchRepositoryFixture) insert(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	deviceInsert := storage.Psql.
		Insert("devices").
		Columns("id", "serial_number", "oui", "carrier", "technology")
	for index, deviceID := range f.deviceIDs {
		deviceInsert = deviceInsert.Values(
			deviceID,
			"BATCH-"+deviceID.String(),
			"001122",
			"cmcc",
			"4G",
		)
		_ = index
	}
	query, args, err := deviceInsert.ToSql()
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Insert("geofence_definitions").
		Columns(
			"id", "name", "carrier", "rule_type", "status",
			"created_by", "updated_by", "created_at", "updated_at",
		).
		Values(
			f.geofenceID,
			"batch-"+f.geofenceID.String(),
			"cmcc",
			RuleTypePolygonAllowZone,
			DefinitionStatusEnabled,
			f.actorID,
			f.actorID,
			f.now,
			f.now,
		).
		ToSql()
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Insert("geofence_versions").
		Columns(
			"id", "geofence_id", "version", "status", "geometry_json",
			"bbox_min_longitude", "bbox_min_latitude",
			"bbox_max_longitude", "bbox_max_latitude", "policy_json",
			"created_by", "published_by", "created_at", "published_at",
		).
		Values(
			f.versionID,
			f.geofenceID,
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
			f.actorID,
			f.actorID,
			f.now,
			f.now,
		).
		ToSql()
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Update("geofence_definitions").
		Set("current_version_id", f.versionID).
		Where(sq.Eq{"id": f.geofenceID}).
		ToSql()
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, query, args...)
	require.NoError(t, err)
}

func (f *batchRepositoryFixture) validCreateJobParams(
	t *testing.T,
) CreateManualBindJobParams {
	t.Helper()
	inputs := make([]BindingInput, 0, len(f.deviceIDs))
	for _, deviceID := range f.deviceIDs {
		idCopy := deviceID
		inputs = append(inputs, BindingInput{
			Key:      "id:" + deviceID.String(),
			Kind:     BindingInputDeviceID,
			Value:    deviceID.String(),
			DeviceID: &idCopy,
		})
	}
	snapshot, err := f.repo.LoadManualBindSnapshot(
		context.Background(),
		f.geofenceID,
		inputs,
		nil,
	)
	require.NoError(t, err)
	preview, err := buildManualBindingPreview(snapshot)
	require.NoError(t, err)
	return CreateManualBindJobParams{
		GeofenceID:         f.geofenceID,
		ActorID:            f.actorID,
		VisibleGroups:      nil,
		Inputs:             inputs,
		PreviewFingerprint: preview.PreviewFingerprint,
		Reason:             "integration confirmation",
		ScheduledAt:        f.now,
	}
}

func (f *batchRepositoryFixture) insertConflictingActiveBinding(
	t *testing.T,
	deviceID uuid.UUID,
) {
	t.Helper()
	otherGeofenceID := uuid.New()
	bindingID := uuid.New()
	f.otherGeofenceIDs = append(f.otherGeofenceIDs, otherGeofenceID)
	f.otherBindingIDs = append(f.otherBindingIDs, bindingID)
	query, args, err := storage.Psql.
		Insert("geofence_definitions").
		Columns(
			"id", "name", "carrier", "rule_type", "status",
			"created_by", "updated_by", "created_at", "updated_at",
		).
		Values(
			otherGeofenceID,
			"batch-conflict-"+otherGeofenceID.String(),
			"cmcc",
			RuleTypePolygonAllowZone,
			DefinitionStatusEnabled,
			f.actorID,
			f.actorID,
			f.now,
			f.now,
		).
		ToSql()
	require.NoError(t, err)
	_, err = f.pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)

	query, args, err = storage.Psql.
		Insert("device_geofence_bindings").
		Columns(
			"id", "device_id", "geofence_id", "rule_type", "status",
			"bind_source", "bound_by", "bound_at",
		).
		Values(
			bindingID,
			deviceID,
			otherGeofenceID,
			RuleTypePolygonAllowZone,
			BindingStatusActive,
			"manual",
			f.actorID,
			f.now,
		).
		ToSql()
	require.NoError(t, err)
	_, err = f.pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)
}

func (f *batchRepositoryFixture) insertBinding(
	t *testing.T,
	bindingID uuid.UUID,
	deviceID uuid.UUID,
	status BindingStatus,
) {
	t.Helper()
	query, args, err := storage.Psql.
		Insert("device_geofence_bindings").
		Columns(
			"id", "device_id", "geofence_id", "rule_type", "status",
			"bind_source", "bound_by", "bound_at",
		).
		Values(
			bindingID,
			deviceID,
			f.geofenceID,
			RuleTypePolygonAllowZone,
			status,
			"manual",
			f.actorID,
			f.now,
		).
		ToSql()
	require.NoError(t, err)
	_, err = f.pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)
}

func (f *batchRepositoryFixture) countJobsByFingerprint(
	t *testing.T,
	fingerprint string,
) int {
	t.Helper()
	var count int
	err := f.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*)
		   FROM async_jobs
		  WHERE job_type = $1
		    AND payload->>'geofence_id' = $2
		    AND payload->>'requested_by' = $3
		    AND payload->>'preview_fingerprint' = $4`,
		ManualBindJobType,
		f.geofenceID.String(),
		f.actorID.String(),
		fingerprint,
	).Scan(&count)
	require.NoError(t, err)
	return count
}

func (f *batchRepositoryFixture) countItems(t *testing.T, jobID uuid.UUID) int {
	t.Helper()
	var count int
	err := f.pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM geofence_batch_items WHERE job_id = $1",
		jobID,
	).Scan(&count)
	require.NoError(t, err)
	return count
}

func (f *batchRepositoryFixture) requireIdempotencyPayload(
	t *testing.T,
	jobID uuid.UUID,
) {
	t.Helper()
	var geofenceID, requestedBy, fingerprint string
	err := f.pool.QueryRow(
		context.Background(),
		`SELECT payload->>'geofence_id',
		        payload->>'requested_by',
		        payload->>'preview_fingerprint'
		   FROM async_jobs
		  WHERE id = $1`,
		jobID,
	).Scan(&geofenceID, &requestedBy, &fingerprint)
	require.NoError(t, err)
	require.Equal(t, f.geofenceID.String(), geofenceID)
	require.Equal(t, f.actorID.String(), requestedBy)
	require.NotEmpty(t, fingerprint)
}

func (f *batchRepositoryFixture) cleanup(ctx context.Context) {
	_, _ = f.pool.Exec(
		ctx,
		"DELETE FROM device_geofence_states WHERE device_id = ANY($1::uuid[])",
		f.deviceIDs,
	)
	_, _ = f.pool.Exec(
		ctx,
		"DELETE FROM device_geofence_effective_states "+
			"WHERE device_id = ANY($1::uuid[])",
		f.deviceIDs,
	)
	_, _ = f.pool.Exec(
		ctx,
		"DELETE FROM geofence_batch_items WHERE geofence_id = $1",
		f.geofenceID,
	)
	_, _ = f.pool.Exec(
		ctx,
		`DELETE FROM async_jobs
		  WHERE job_type = $1
		    AND payload->>'geofence_id' = $2`,
		ManualBindJobType,
		f.geofenceID.String(),
	)
	_, _ = f.pool.Exec(
		ctx,
		"DELETE FROM device_geofence_bindings WHERE device_id = ANY($1::uuid[])",
		f.deviceIDs,
	)
	allGeofenceIDs := append(
		[]uuid.UUID{f.geofenceID},
		f.otherGeofenceIDs...,
	)
	_, _ = f.pool.Exec(
		ctx,
		"UPDATE geofence_definitions SET current_version_id = NULL "+
			"WHERE id = ANY($1::uuid[])",
		allGeofenceIDs,
	)
	_, _ = f.pool.Exec(
		ctx,
		"DELETE FROM geofence_versions WHERE geofence_id = ANY($1::uuid[])",
		allGeofenceIDs,
	)
	_, _ = f.pool.Exec(
		ctx,
		"DELETE FROM geofence_definitions WHERE id = ANY($1::uuid[])",
		allGeofenceIDs,
	)
	_, _ = f.pool.Exec(
		ctx,
		"DELETE FROM devices WHERE id = ANY($1::uuid[])",
		f.deviceIDs,
	)
}
