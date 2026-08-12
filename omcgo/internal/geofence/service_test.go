package geofence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepository struct {
	Repository
	createdDefinition       *Definition
	createdVersion          *Version
	definition              *Definition
	version                 *Version
	publishCalls            int
	createdDraft            *Version
	lifecycleImpact         LifecycleImpact
	transitionCalls         int
	transitionDefinitionErr error
	transitionBindingTarget BindingStatus
	transitionBindingCalls  int
	transitionBindingErr    error
	mapDefinitions          []MapDefinition
	mapDefinitionFilter     MapDefinitionFilter
	mapDefinitionErr        error
	versions                []Version
	listVersionsGeofenceID  uuid.UUID
	listVersionsErr         error
	bindingDetailPage       BindingDetailPage
	bindingDetailFilter     BindingDetailFilter
	bindingDetailErr        error
	denyCarrierVisibility   bool
	deniedCarriers          map[string]bool
	carrierVisibilityCalls  int
	carrierVisibilityGroups []uuid.UUID
	denyDeviceVisibility    bool
	deviceVisibilityCalls   int
	updatedName             string
	updateDefinitionNameErr error
}

type candidateReaderStub struct {
	devices []CandidateDevice
}

func (s candidateReaderStub) ListGeofenceCandidates(
	context.Context,
	string,
	[]uuid.UUID,
) ([]CandidateDevice, error) {
	return s.devices, nil
}

func (f *fakeRepository) CreateDefinitionWithDraft(
	_ context.Context,
	definition *Definition,
	version *Version,
) error {
	f.createdDefinition = definition
	f.createdVersion = version
	return nil
}

func (f *fakeRepository) GetDefinition(context.Context, uuid.UUID) (*Definition, error) {
	return f.definition, nil
}

func (f *fakeRepository) GetVersion(context.Context, uuid.UUID) (*Version, error) {
	return f.version, nil
}

func (f *fakeRepository) UpdateDefinitionName(
	_ context.Context,
	_ uuid.UUID,
	name string,
	_ uuid.UUID,
	_ time.Time,
) error {
	f.updatedName = name
	return f.updateDefinitionNameErr
}

func (f *fakeRepository) IsCarrierVisible(
	_ context.Context,
	carrier string,
	visibleGroups []uuid.UUID,
) (bool, error) {
	f.carrierVisibilityCalls++
	f.carrierVisibilityGroups = visibleGroups
	return !f.denyCarrierVisibility && !f.deniedCarriers[carrier], nil
}

func (f *fakeRepository) IsDeviceVisible(
	context.Context,
	uuid.UUID,
	[]uuid.UUID,
) (bool, error) {
	f.deviceVisibilityCalls++
	return !f.denyDeviceVisibility, nil
}

func (f *fakeRepository) PublishVersion(
	context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time,
) error {
	f.publishCalls++
	return nil
}

func (f *fakeRepository) CreateDraftVersion(
	_ context.Context,
	version *Version,
) error {
	version.Version = 2
	f.createdDraft = version
	return nil
}

func (f *fakeRepository) GetLifecycleImpact(
	context.Context,
	uuid.UUID,
) (LifecycleImpact, error) {
	return f.lifecycleImpact, nil
}

func (f *fakeRepository) TransitionDefinition(
	context.Context,
	uuid.UUID,
	DefinitionStatus,
	uuid.UUID,
	string,
	string,
	time.Time,
) error {
	f.transitionCalls++
	return f.transitionDefinitionErr
}

func (f *fakeRepository) TransitionBinding(
	_ context.Context,
	bindingID uuid.UUID,
	target BindingStatus,
	actorID uuid.UUID,
	reason string,
	_ time.Time,
) (*Binding, error) {
	f.transitionBindingCalls++
	f.transitionBindingTarget = target
	if f.transitionBindingErr != nil {
		return nil, f.transitionBindingErr
	}
	return &Binding{
		ID: bindingID, Status: target, RemovedBy: func() *uuid.UUID {
			if target == BindingStatusRemoved {
				return &actorID
			}
			return nil
		}(), RemoveReason: reason,
	}, nil
}

type fakeSettingsRepository struct {
	settings     Settings
	preview      SettingsPreview
	getCalls     int
	previewCalls int
	updateCalls  int
	updatedBy    uuid.UUID
}

func (f *fakeSettingsRepository) GetSettings(context.Context) (Settings, error) {
	f.getCalls++
	return f.settings, nil
}

func (f *fakeSettingsRepository) PreviewSettings(
	context.Context,
	Settings,
) (SettingsPreview, error) {
	f.previewCalls++
	return f.preview, nil
}

func (f *fakeSettingsRepository) UpdateSettings(
	_ context.Context,
	settings Settings,
	actorID uuid.UUID,
	_ time.Time,
) (Settings, error) {
	f.updateCalls++
	f.updatedBy = actorID
	return settings, nil
}

func TestService_CreateDefinitionCreatesDraftWithNormalizedGeometry(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, nil)
	actorID := uuid.New()

	result, err := service.CreateDefinition(context.Background(), CreateDefinitionRequest{
		Name:     " 杭州测试围栏 ",
		Carrier:  "cmcc",
		RuleType: RuleTypePolygonAllowZone,
		Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[[121.1,31.1],[121.2,31.1],[121.2,31.2]]]}`),
		Policy:   json.RawMessage(`{"exit_action":"notify_only"}`),
		ActorID:  actorID,
	})

	require.NoError(t, err)
	assert.Equal(t, "杭州测试围栏", result.Definition.Name)
	assert.Equal(t, DefinitionStatusDraft, result.Definition.Status)
	assert.Equal(t, int64(1), result.DraftVersion.Version)
	assert.Equal(t, VersionStatusDraft, result.DraftVersion.Status)
	assert.Equal(t, actorID, result.Definition.CreatedBy)
	assert.NotEmpty(t, repo.createdVersion.GeometryJSON)
	assert.Nil(t, result.Definition.CurrentVersionID)
}

func TestService_CreateDefinitionRejectsManualReviewPolicy(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, nil)

	_, err := service.CreateDefinition(
		context.Background(),
		CreateDefinitionRequest{
			Name:     "legacy manual review",
			Carrier:  "cmcc",
			RuleType: RuleTypePolygonAllowZone,
			Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[[121.1,31.1],[121.2,31.1],[121.2,31.2]]]}`),
			Policy:   json.RawMessage(`{"exit_action":"manual_review"}`),
			ActorID:  uuid.New(),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.ErrorContains(t, err, "unsupported exit_action")
	assert.Nil(t, repository.createdDefinition)
	assert.Nil(t, repository.createdVersion)
}

func TestService_DefinitionNameBoundaryIsSharedByCreateAndRename(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, nil)
	actorID := uuid.New()
	validRequest := CreateDefinitionRequest{
		Carrier: "cmcc", RuleType: RuleTypePolygonAllowZone,
		Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[[121.1,31.1],[121.2,31.1],[121.2,31.2]]]}`),
		Policy:   json.RawMessage(`{"exit_action":"notify_only"}`), ActorID: actorID,
	}

	validRequest.Name = strings.Repeat("围", 128)
	_, err := service.CreateDefinition(context.Background(), validRequest)
	require.NoError(t, err)

	validRequest.Name = strings.Repeat("围", 129)
	_, err = service.CreateDefinition(context.Background(), validRequest)
	require.Error(t, err)
	var businessErr *commonerrors.BusinessError
	require.True(t, errors.As(err, &businessErr))
	assert.Equal(t, ErrCodeDefinitionNameInvalid, businessErr.Code)

	err = service.RenameDefinition(
		context.Background(), uuid.New(), strings.Repeat("围", 129), actorID,
	)
	require.Error(t, err)
	require.True(t, errors.As(err, &businessErr))
	assert.Equal(t, ErrCodeDefinitionNameInvalid, businessErr.Code)
}

func TestService_PreviewGeofenceCandidatesClassifiesDevices(t *testing.T) {
	geofenceID := uuid.New()
	versionID := uuid.New()
	insideLat, insideLng := 31.1, 121.1
	outsideLat, outsideLng := 32.0, 122.0
	repo := &fakeRepository{
		definition: &Definition{
			ID: geofenceID, Carrier: "cmcc", RuleType: RuleTypePolygonAllowZone,
			CurrentVersionID: &versionID,
		},
		version: &Version{
			ID: versionID, GeofenceID: geofenceID, Status: VersionStatusPublished,
			GeometryJSON: json.RawMessage(`{"type":"Polygon","coordinates":[[[121,31],[121.2,31],[121.2,31.2],[121,31.2],[121,31]]]}`),
		},
	}
	service := NewService(repo, nil)
	service.SetGeofenceCandidateReader(candidateReaderStub{devices: []CandidateDevice{
		{ID: uuid.New(), SerialNumber: "inside", Latitude: &insideLat, Longitude: &insideLng},
		{ID: uuid.New(), SerialNumber: "outside", Latitude: &outsideLat, Longitude: &outsideLng},
		{ID: uuid.New(), SerialNumber: "unknown"},
	}})

	preview, err := service.PreviewGeofenceCandidates(context.Background(), geofenceID, nil)

	require.NoError(t, err)
	require.Len(t, preview.Inside, 1)
	require.Equal(t, "inside", preview.Inside[0].SerialNumber)
	require.Len(t, preview.Outside, 1)
	require.Equal(t, "outside", preview.Outside[0].SerialNumber)
	require.Len(t, preview.NoLocation, 1)
}

func TestService_CreateDefinitionRejectsCarrierOutsideVisibleGroups(
	t *testing.T,
) {
	repository := &fakeRepository{denyCarrierVisibility: true}
	service := NewService(repository, nil)
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	_, err := service.CreateDefinition(
		context.Background(),
		CreateDefinitionRequest{
			Name:          "restricted",
			Carrier:       "cmcc",
			RuleType:      RuleTypePolygonAllowZone,
			Geometry:      json.RawMessage(`{"type":"Polygon","coordinates":[[[120,30],[121,30],[121,31],[120,30]]]}`),
			Policy:        json.RawMessage(`{"exit_action":"notify_only"}`),
			ActorID:       uuid.New(),
			VisibleGroups: []uuid.UUID{groupID},
		},
	)

	require.ErrorIs(t, err, commonerrors.ErrForbidden)
	require.Nil(t, repository.createdDefinition)
	require.Equal(t, 1, repository.carrierVisibilityCalls)
	require.Equal(t, []uuid.UUID{groupID}, repository.carrierVisibilityGroups)
}

func TestService_CreateDefinitionRejectsInvisibleBaselineOwner(
	t *testing.T,
) {
	ownerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repository := &fakeRepository{denyDeviceVisibility: true}
	service := NewService(repository, nil)

	_, err := service.CreateDefinition(
		context.Background(),
		CreateDefinitionRequest{
			Name:          "restricted baseline",
			Carrier:       "cmcc",
			RuleType:      RuleTypeBaselineRadius,
			OwnerDeviceID: &ownerID,
			Geometry:      json.RawMessage(`{"type":"PointRadius","center":[120,30],"radius_meters":100}`),
			Policy:        json.RawMessage(`{"exit_action":"notify_only"}`),
			ActorID:       uuid.New(),
			VisibleGroups: []uuid.UUID{uuid.New()},
		},
	)

	require.ErrorIs(t, err, commonerrors.ErrForbidden)
	require.Nil(t, repository.createdDefinition)
	require.Equal(t, 1, repository.deviceVisibilityCalls)
}

func TestService_PublishDraftAllowsDeactivatePolicy(t *testing.T) {
	geofenceID := uuid.New()
	versionID := uuid.New()
	repo := &fakeRepository{
		definition: &Definition{ID: geofenceID},
		version: &Version{
			ID:         versionID,
			GeofenceID: geofenceID,
			Status:     VersionStatusDraft,
			PolicyJSON: json.RawMessage(`{"exit_action":"deactivate"}`),
		},
	}
	service := NewService(repo, nil)

	err := service.PublishDraft(context.Background(), geofenceID, versionID, uuid.New())

	require.NoError(t, err)
	assert.Equal(t, 1, repo.publishCalls)
}

func TestService_GetSettingsDelegatesToSettingsRepository(t *testing.T) {
	settingsRepository := &fakeSettingsRepository{settings: Settings{
		SystemMode: RuntimeModeOff,
		Carriers: []CarrierSetting{{
			Carrier: "cmcc", Mode: RuntimeModeOff,
			DefaultBaselineRadiusMeters: 100,
		}},
	}}
	service := NewService(&fakeRepository{}, settingsRepository)

	settings, err := service.GetSettings(context.Background())

	require.NoError(t, err)
	assert.Equal(t, RuntimeModeOff, settings.SystemMode)
	assert.Equal(t, 1, settingsRepository.getCalls)
}

func TestService_PreviewSettingsAcceptsEnforce(t *testing.T) {
	settingsRepository := &fakeSettingsRepository{}
	service := NewService(&fakeRepository{}, settingsRepository)

	_, err := service.PreviewSettings(context.Background(), Settings{
		SystemMode: RuntimeModeEnforce,
		Carriers: []CarrierSetting{{
			Carrier: "cmcc", Mode: RuntimeModeObserve,
			DefaultBaselineRadiusMeters: 100,
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, settingsRepository.previewCalls)
}

func TestService_UpdateSettingsRequiresActorAndAcceptsEnforce(
	t *testing.T,
) {
	settingsRepository := &fakeSettingsRepository{}
	service := NewService(&fakeRepository{}, settingsRepository)
	valid := Settings{
		SystemMode: RuntimeModeObserve,
		Carriers: []CarrierSetting{{
			Carrier: "cmcc", Mode: RuntimeModeObserve,
			DefaultBaselineRadiusMeters: 100,
		}},
	}

	_, err := service.UpdateSettings(context.Background(), valid, uuid.Nil)
	require.Error(t, err)
	assert.Zero(t, settingsRepository.updateCalls)

	valid.SystemMode = RuntimeModeEnforce
	_, err = service.UpdateSettings(context.Background(), valid, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 1, settingsRepository.updateCalls)
}

func TestService_UpdateSettingsDelegatesValidatedSnapshot(t *testing.T) {
	settingsRepository := &fakeSettingsRepository{}
	service := NewService(&fakeRepository{}, settingsRepository)
	actorID := uuid.New()

	updated, err := service.UpdateSettings(context.Background(), Settings{
		SystemMode: RuntimeModeObserve,
		Carriers: []CarrierSetting{{
			Carrier: "cmcc", Mode: RuntimeModeObserve,
			DefaultBaselineRadiusMeters: 100,
		}},
	}, actorID)

	require.NoError(t, err)
	assert.Equal(t, RuntimeModeObserve, updated.SystemMode)
	assert.Equal(t, 1, settingsRepository.updateCalls)
	assert.Equal(t, actorID, settingsRepository.updatedBy)
}

func TestService_CreateDraftVersionValidatesAndPreservesPublishedVersion(
	t *testing.T,
) {
	geofenceID := uuid.New()
	currentVersionID := uuid.New()
	actorID := uuid.New()
	repository := &fakeRepository{definition: &Definition{
		ID: geofenceID, RuleType: RuleTypePolygonAllowZone,
		Status: DefinitionStatusEnabled, CurrentVersionID: &currentVersionID,
	}}
	service := NewService(repository, nil)

	version, err := service.CreateDraftVersion(
		context.Background(),
		CreateVersionRequest{
			GeofenceID: geofenceID,
			Geometry: json.RawMessage(
				`{"type":"Polygon","coordinates":[[[120,30],[121,30],` +
					`[121,31],[120,30]]]}`,
			),
			Policy:  json.RawMessage(`{"exit_action":"notify_only"}`),
			ActorID: actorID,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, repository.createdDraft)
	assert.Equal(t, int64(2), version.Version)
	assert.Equal(t, VersionStatusDraft, version.Status)
	assert.Equal(t, actorID, version.CreatedBy)
	assert.Equal(t, currentVersionID, *repository.definition.CurrentVersionID)
	assert.Equal(t, DefinitionStatusEnabled, repository.definition.Status)
}

func TestService_CreateDraftVersionRejectsInvalidGeometryBeforeRepository(
	t *testing.T,
) {
	geofenceID := uuid.New()
	repository := &fakeRepository{definition: &Definition{
		ID: geofenceID, RuleType: RuleTypePolygonAllowZone,
		Status: DefinitionStatusEnabled,
	}}
	service := NewService(repository, nil)

	_, err := service.CreateDraftVersion(
		context.Background(),
		CreateVersionRequest{
			GeofenceID: geofenceID,
			Geometry:   json.RawMessage(`{"type":"Point"}`),
			Policy:     json.RawMessage(`{"exit_action":"notify_only"}`),
			ActorID:    uuid.New(),
		},
	)

	require.Error(t, err)
	assert.Nil(t, repository.createdDraft)
}

func TestService_CreateDraftVersionRejectsManualReviewPolicy(
	t *testing.T,
) {
	geofenceID := uuid.New()
	repository := &fakeRepository{definition: &Definition{
		ID: geofenceID, RuleType: RuleTypePolygonAllowZone,
		Status: DefinitionStatusEnabled,
	}}
	service := NewService(repository, nil)

	_, err := service.CreateDraftVersion(
		context.Background(),
		CreateVersionRequest{
			GeofenceID: geofenceID,
			Geometry: json.RawMessage(
				`{"type":"Polygon","coordinates":[[[120,30],[121,30],` +
					`[121,31],[120,30]]]}`,
			),
			Policy:  json.RawMessage(`{"exit_action":"manual_review"}`),
			ActorID: uuid.New(),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.ErrorContains(t, err, "unsupported exit_action")
	assert.Nil(t, repository.createdDraft)
}

func TestService_PreviewDefinitionTransitionReturnsImpactFingerprint(t *testing.T) {
	geofenceID := uuid.New()
	versionID := uuid.New()
	repository := &fakeRepository{lifecycleImpact: LifecycleImpact{
		GeofenceID: geofenceID, CurrentVersionID: &versionID,
		CurrentStatus: DefinitionStatusEnabled,
		BindingCount:  3, DeviceCount: 3,
	}}
	service := NewService(repository, nil)

	impact, err := service.PreviewDefinitionTransition(
		context.Background(),
		geofenceID,
		DefinitionStatusDisabled,
	)

	require.NoError(t, err)
	assert.Equal(t, DefinitionStatusDisabled, impact.TargetStatus)
	assert.NotEmpty(t, impact.PreviewFingerprint)
	assert.Equal(t, int64(3), impact.DeviceCount)
}

func TestService_EnablePreviewExcludesDeactivationImpact(t *testing.T) {
	geofenceID := uuid.New()
	repository := &fakeRepository{lifecycleImpact: LifecycleImpact{
		GeofenceID:              geofenceID,
		CurrentStatus:           DefinitionStatusDisabled,
		DeactivationDeviceCount: 2,
		deactivationSignature:   "candidate-a,candidate-b",
	}}
	service := NewService(repository, nil)

	impact, err := service.PreviewDefinitionTransition(
		context.Background(),
		geofenceID,
		DefinitionStatusEnabled,
	)

	require.NoError(t, err)
	assert.Zero(t, impact.DeactivationDeviceCount)
	assert.Empty(t, impact.deactivationSignature)
}

func TestService_TransitionDefinitionRequiresReasonAndValidPreview(t *testing.T) {
	geofenceID := uuid.New()
	repository := &fakeRepository{}
	service := NewService(repository, nil)

	err := service.TransitionDefinition(
		context.Background(),
		geofenceID,
		DefinitionStatusDisabled,
		uuid.New(),
		"",
		"fingerprint",
	)

	require.Error(t, err)
	assert.Zero(t, repository.transitionCalls)
}

func TestService_BindingLifecycleRequiresActorAndDestructiveReason(t *testing.T) {
	bindingID := uuid.New()
	repository := &fakeRepository{}
	service := NewService(repository, nil)

	_, err := service.SuspendBinding(
		context.Background(),
		bindingID,
		uuid.Nil,
		"maintenance",
	)
	require.Error(t, err)
	_, err = service.SuspendBinding(
		context.Background(),
		bindingID,
		uuid.New(),
		" ",
	)
	require.Error(t, err)
	_, err = service.RemoveBinding(
		context.Background(),
		bindingID,
		uuid.New(),
		"",
	)
	require.Error(t, err)
	_, err = service.ResumeBinding(
		context.Background(),
		bindingID,
		uuid.Nil,
		"resume after maintenance",
	)
	require.Error(t, err)
	_, err = service.ResumeBinding(
		context.Background(),
		bindingID,
		uuid.New(),
		" ",
	)
	require.Error(t, err)
	assert.Zero(t, repository.transitionBindingCalls)
}

func TestService_BindingLifecycleDelegatesExplicitTargets(t *testing.T) {
	bindingID := uuid.New()
	actorID := uuid.New()
	repository := &fakeRepository{}
	service := NewService(repository, nil)

	suspended, err := service.SuspendBinding(
		context.Background(),
		bindingID,
		actorID,
		"maintenance",
	)
	require.NoError(t, err)
	assert.Equal(t, BindingStatusSuspended, suspended.Status)
	assert.Equal(t, BindingStatusSuspended, repository.transitionBindingTarget)

	resumed, err := service.ResumeBinding(
		context.Background(),
		bindingID,
		actorID,
		"maintenance completed",
	)
	require.NoError(t, err)
	assert.Equal(t, BindingStatusActive, resumed.Status)
	assert.Equal(t, BindingStatusActive, repository.transitionBindingTarget)

	removed, err := service.RemoveBinding(
		context.Background(),
		bindingID,
		actorID,
		"retired",
	)
	require.NoError(t, err)
	assert.Equal(t, BindingStatusRemoved, removed.Status)
	assert.Equal(t, BindingStatusRemoved, repository.transitionBindingTarget)
	assert.Equal(t, 3, repository.transitionBindingCalls)
}
