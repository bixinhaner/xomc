package geofence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/require"
)

func TestBuildVisibleBatchDevicesQueryFailsClosed(t *testing.T) {
	sql, _, err := buildVisibleBatchDevicesQuery(
		[]uuid.UUID{uuid.New()}, nil, []uuid.UUID{},
	)
	require.NoError(t, err)
	require.Contains(t, sql, "FALSE")
}

func TestBuildVisibleBatchDevicesQueryLeavesNilVisibilityUnfiltered(t *testing.T) {
	sql, _, err := buildVisibleBatchDevicesQuery(
		[]uuid.UUID{uuid.New()}, nil, nil,
	)
	require.NoError(t, err)
	require.NotContains(t, sql, "device_group_members")
	require.NotContains(t, strings.ToUpper(sql), "FALSE")
}

func TestBuildActiveBindingFactsQueryIncludesActiveAndSuspended(t *testing.T) {
	sql, _, err := buildActiveBindingFactsQuery(
		uuid.New(),
		[]uuid.UUID{uuid.New()},
	)
	require.NoError(t, err)
	require.Contains(t, sql, "active")
	require.Contains(t, sql, "suspended")
	require.Contains(t, sql, "rule_type")
}

type batchServiceRepository struct {
	Repository
	createParams CreateManualBindJobParams
	createResult BatchJobAccepted
	createErr    error
	job          *BatchJob
	getErr       error
	itemFilter   BatchItemFilter
	itemPage     BatchItemPage
	listCalls    int
}

func (r *batchServiceRepository) CreateManualBindJob(
	_ context.Context,
	params CreateManualBindJobParams,
) (BatchJobAccepted, error) {
	r.createParams = params
	return r.createResult, r.createErr
}

func (r *batchServiceRepository) GetManualBindJob(
	context.Context,
	uuid.UUID,
) (*BatchJob, error) {
	return r.job, r.getErr
}

func (r *batchServiceRepository) ListManualBindItems(
	_ context.Context,
	filter BatchItemFilter,
) (BatchItemPage, error) {
	r.listCalls++
	r.itemFilter = filter
	return r.itemPage, nil
}

func TestService_CreateManualBindJobNormalizesInputsAndDelegates(t *testing.T) {
	now := time.Date(2026, time.July, 30, 8, 0, 0, 0, time.UTC)
	deviceID := uuid.New()
	groupID := uuid.New()
	repository := &batchServiceRepository{
		createResult: BatchJobAccepted{JobID: uuid.New()},
	}
	service := NewService(repository, nil)
	service.now = func() time.Time { return now }

	accepted, err := service.CreateManualBindJob(
		context.Background(),
		CreateManualBindJobRequest{
			GeofenceID:         uuid.New(),
			ActorID:            uuid.New(),
			VisibleGroups:      []uuid.UUID{groupID},
			Inputs:             BindingInputRequest{DeviceIDs: []uuid.UUID{deviceID, deviceID}},
			PreviewFingerprint: " sha256:preview ",
			Reason:             " maintenance ",
		},
	)

	require.NoError(t, err)
	require.Equal(t, repository.createResult, accepted)
	require.Equal(t, "sha256:preview", repository.createParams.PreviewFingerprint)
	require.Equal(t, "maintenance", repository.createParams.Reason)
	require.Equal(t, now, repository.createParams.ScheduledAt)
	require.Equal(t, []uuid.UUID{groupID}, repository.createParams.VisibleGroups)
	require.Equal(t, []BindingInput{{
		Key:      "id:" + deviceID.String(),
		Kind:     BindingInputDeviceID,
		Value:    deviceID.String(),
		DeviceID: &deviceID,
	}}, repository.createParams.Inputs)
}

func TestService_GetManualBindJobEnforcesOwnership(t *testing.T) {
	requesterID := uuid.New()
	repository := &batchServiceRepository{
		job: &BatchJob{ID: uuid.New(), RequestedBy: requesterID},
	}
	service := NewService(repository, nil)

	_, err := service.GetManualBindJob(
		context.Background(),
		repository.job.ID,
		uuid.New(),
		false,
	)
	require.ErrorIs(t, err, commonerrors.ErrForbidden)

	got, err := service.GetManualBindJob(
		context.Background(),
		repository.job.ID,
		uuid.New(),
		true,
	)
	require.NoError(t, err)
	require.Equal(t, repository.job, got)
}

func TestService_ListManualBindItemsNormalizesPageAndChecksOwner(t *testing.T) {
	requesterID := uuid.New()
	jobID := uuid.New()
	repository := &batchServiceRepository{
		job:      &BatchJob{ID: jobID, RequestedBy: requesterID},
		itemPage: BatchItemPage{Page: 1, PageSize: 20},
	}
	service := NewService(repository, nil)

	page, err := service.ListManualBindItems(
		context.Background(),
		BatchItemFilter{JobID: jobID, Page: 0, PageSize: 20},
		requesterID,
		false,
	)

	require.NoError(t, err)
	require.Equal(t, 1, repository.itemFilter.Page)
	require.Equal(t, 20, repository.itemFilter.PageSize)
	require.Equal(t, repository.itemPage, page)
}

func TestService_ListManualBindItemsRejectsOversizedPageBeforeRepository(
	t *testing.T,
) {
	repository := &batchServiceRepository{}
	service := NewService(repository, nil)

	_, err := service.ListManualBindItems(
		context.Background(),
		BatchItemFilter{
			JobID: uuid.New(), Page: 1, PageSize: MaxBatchItemPageSize + 1,
		},
		uuid.New(),
		false,
	)

	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	require.Zero(t, repository.listCalls)
}

func TestService_GetManualBindJobWrapsRepositoryError(t *testing.T) {
	repository := &batchServiceRepository{getErr: errors.New("read failed")}
	service := NewService(repository, nil)

	_, err := service.GetManualBindJob(
		context.Background(),
		uuid.New(),
		uuid.New(),
		false,
	)

	require.ErrorContains(t, err, "get manual bind job")
	require.ErrorContains(t, err, "read failed")
}

func TestBuildInsertManualBindJobQueryUsesPartialIndexSafeConflictHandling(
	t *testing.T,
) {
	payload, err := json.Marshal(ManualBindJobPayload{
		SchemaVersion:      ManualBindPayloadVersion,
		GeofenceID:         uuid.New(),
		GeofenceVersionID:  uuid.New(),
		RequestedBy:        uuid.New(),
		PreviewFingerprint: "sha256:preview",
		Reason:             "maintenance",
	})
	require.NoError(t, err)

	sql, args, err := buildInsertManualBindJobQuery(
		uuid.New(),
		time.Now().UTC(),
		payload,
	)

	require.NoError(t, err)
	require.Contains(t, sql, "ON CONFLICT DO NOTHING")
	require.NotContains(t, sql, "ON CONFLICT (")
	require.Contains(t, sql, "RETURNING id")
	require.Contains(t, string(payload), `"geofence_id"`)
	require.Contains(t, string(payload), `"requested_by"`)
	require.Contains(t, string(payload), `"preview_fingerprint"`)
	require.Contains(t, args, json.RawMessage(payload))
}

func TestBuildListManualBindItemsQueryRedactsWithoutFilteringTotal(t *testing.T) {
	sql, _, err := buildListManualBindItemsQuery(BatchItemFilter{
		JobID:         uuid.New(),
		Page:          2,
		PageSize:      20,
		VisibleGroups: []uuid.UUID{},
	})

	require.NoError(t, err)
	require.Contains(t, sql, "COUNT(*) OVER()")
	require.Contains(t, sql, "FALSE")
	require.Contains(t, sql, "CASE WHEN")
	require.NotContains(t, strings.ToUpper(sql), "WHERE FALSE")
	require.Contains(t, sql, "LIMIT")
	require.Contains(t, sql, "OFFSET")
}

func TestBuildBatchSnapshotRejectsMissingExplicitUUIDWithoutDisclosure(
	t *testing.T,
) {
	deviceID := uuid.New()
	_, err := buildBatchSnapshot(
		Definition{ID: uuid.New()},
		[]BindingInput{{
			Key:      "id:" + deviceID.String(),
			Kind:     BindingInputDeviceID,
			Value:    deviceID.String(),
			DeviceID: &deviceID,
		}},
		newManualBindingDevices(),
		manualBindingFacts{
			sameGeofenceStatus:  map[uuid.UUID]BindingStatus{},
			activeRuleConflicts: map[uuid.UUID]manualBindingConflict{},
		},
		[]uuid.UUID{},
	)

	require.ErrorIs(t, err, commonerrors.ErrForbidden)
}

func TestBuildBatchSnapshotCollapsesUUIDAndSNAliasesToOneDevice(t *testing.T) {
	deviceID := uuid.New()
	device := &DeviceIdentity{
		ID: deviceID, SerialNumber: "SN001", Carrier: "cmcc",
	}
	devices := newManualBindingDevices()
	devices.add(device, "cmcc")

	snapshot, err := buildBatchSnapshot(
		Definition{ID: uuid.New(), Carrier: "cmcc"},
		[]BindingInput{
			{
				Key: "id:" + deviceID.String(), Kind: BindingInputDeviceID,
				Value: deviceID.String(), DeviceID: &deviceID,
			},
			{Key: "sn:SN001", Kind: BindingInputDeviceSN, Value: "SN001"},
		},
		devices,
		manualBindingFacts{
			sameGeofenceStatus:  map[uuid.UUID]BindingStatus{},
			activeRuleConflicts: map[uuid.UUID]manualBindingConflict{},
		},
		nil,
	)

	require.NoError(t, err)
	require.Len(t, snapshot.Inputs, 1)
	require.Equal(t, BindingInputDeviceID, snapshot.Inputs[0].Input.Kind)
	require.Equal(t, deviceID, snapshot.Inputs[0].Device.ID)
}

func TestBuildBatchSnapshotKeepsMissingSNAsSkippedCandidate(t *testing.T) {
	snapshot, err := buildBatchSnapshot(
		Definition{ID: uuid.New(), Carrier: "cmcc"},
		[]BindingInput{{
			Key: "sn:MISSING", Kind: BindingInputDeviceSN, Value: "MISSING",
		}},
		newManualBindingDevices(),
		manualBindingFacts{
			sameGeofenceStatus:  map[uuid.UUID]BindingStatus{},
			activeRuleConflicts: map[uuid.UUID]manualBindingConflict{},
		},
		nil,
	)

	require.NoError(t, err)
	require.Len(t, snapshot.Inputs, 1)
	require.Nil(t, snapshot.Inputs[0].Device)
}

func TestBatchDatabaseErrorDoesNotExposeConstraintName(t *testing.T) {
	err := batchDatabaseError(
		"insert manual bind items",
		&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "uq_geofence_batch_items_job_device",
		},
	)

	require.ErrorIs(t, err, commonerrors.ErrInternal)
	require.NotContains(t, err.Error(), "uq_geofence_batch_items_job_device")
}

func TestBuildFindExistingManualBindJobQueryIncludesTerminalStatuses(t *testing.T) {
	sql, _, err := buildFindExistingManualBindJobQuery(
		CreateManualBindJobParams{
			GeofenceID:         uuid.New(),
			ActorID:            uuid.New(),
			PreviewFingerprint: "sha256:preview",
		},
	)

	require.NoError(t, err)
	require.NotContains(t, sql, "status IN")
	require.Contains(t, sql, "created_at DESC")
	require.Contains(t, sql, "id DESC")
}

func TestBuildFindReusableManualBindJobQueryUsesFullKeyAndReusableStatuses(
	t *testing.T,
) {
	params := CreateManualBindJobParams{
		GeofenceID:         uuid.New(),
		ActorID:            uuid.New(),
		PreviewFingerprint: "sha256:preview",
	}

	sql, args, err := buildFindReusableManualBindJobQuery(params)

	require.NoError(t, err)
	require.Contains(t, sql, "status IN")
	require.Contains(t, args, params.GeofenceID.String())
	require.Contains(t, args, params.ActorID.String())
	require.Contains(t, args, params.PreviewFingerprint)
	require.Contains(t, args, asyncjob.StatusPending)
	require.Contains(t, args, asyncjob.StatusRunning)
	require.Contains(t, args, asyncjob.StatusSucceeded)
	require.NotContains(t, args, asyncjob.StatusFailed)
	require.NotContains(t, args, asyncjob.StatusCanceled)
	require.NotContains(t, args, asyncjob.StatusZombie)
}

func TestBuildManualBindConfirmationLockQueryUsesBusinessKey(t *testing.T) {
	sql, args, err := buildManualBindConfirmationLockQuery(
		CreateManualBindJobParams{
			GeofenceID:         uuid.New(),
			ActorID:            uuid.New(),
			PreviewFingerprint: "sha256:preview",
		},
	)

	require.NoError(t, err)
	require.Contains(t, sql, "pg_advisory_xact_lock")
	require.Len(t, args, 1)
	require.Contains(t, args[0], "sha256:preview")
}
