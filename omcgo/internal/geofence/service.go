package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type DeviceIdentity struct {
	ID           uuid.UUID
	SerialNumber string
	Carrier      string
}

type Service struct {
	repository                        Repository
	settingsRepository                SettingsRepository
	candidateReader                   GeofenceCandidateReader
	controlActionReader               GeofenceControlActionReader
	thirdPartyLocationStore           ThirdPartyLocationDeviceStore
	thirdPartyLocationBatchRepository ThirdPartyLocationBatchRepository
	now                               func() time.Time
}

func NewService(
	repository Repository,
	settingsRepository SettingsRepository,
) *Service {
	return &Service{
		repository:         repository,
		settingsRepository: settingsRepository,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) SetThirdPartyLocationStore(
	store ThirdPartyLocationDeviceStore,
) {
	s.thirdPartyLocationStore = store
}

func (s *Service) SetThirdPartyLocationBatchRepository(
	repository ThirdPartyLocationBatchRepository,
) {
	s.thirdPartyLocationBatchRepository = repository
}

func (s *Service) SetGeofenceCandidateReader(reader GeofenceCandidateReader) {
	s.candidateReader = reader
}

func (s *Service) SetGeofenceControlActionReader(reader GeofenceControlActionReader) {
	s.controlActionReader = reader
}

func (s *Service) GetSettings(ctx context.Context) (Settings, error) {
	if s.settingsRepository == nil {
		return Settings{}, fmt.Errorf("geofence settings repository is not configured")
	}
	settings, err := s.settingsRepository.GetSettings(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("get geofence settings: %w", err)
	}
	return settings, nil
}

func (s *Service) GetSettingsForVisibleGroups(
	ctx context.Context,
	visibleGroups []uuid.UUID,
) (Settings, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	if visibleGroups == nil {
		return settings, nil
	}

	visible := make([]CarrierSetting, 0, len(settings.Carriers))
	for _, setting := range settings.Carriers {
		allowed, err := s.repository.IsCarrierVisible(
			ctx,
			setting.Carrier,
			visibleGroups,
		)
		if err != nil {
			return Settings{}, fmt.Errorf(
				"authorize geofence carrier %s: %w",
				setting.Carrier,
				err,
			)
		}
		if allowed {
			visible = append(visible, setting)
		}
	}
	settings.Carriers = visible
	return settings, nil
}

func (s *Service) GetAvailability(ctx context.Context) (Availability, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return Availability{}, fmt.Errorf("get geofence availability: %w", err)
	}
	return Availability{Enabled: settings.SystemMode != RuntimeModeOff}, nil
}

func (s *Service) AuthorizeDefinitionAccess(
	ctx context.Context,
	geofenceID uuid.UUID,
	visibleGroups []uuid.UUID,
) error {
	if visibleGroups == nil {
		return nil
	}
	definition, err := s.repository.GetDefinition(ctx, geofenceID)
	if err != nil {
		return fmt.Errorf("get geofence for authorization: %w", err)
	}
	if definition == nil {
		return commonerrors.ErrNotFound
	}
	visible, err := s.repository.IsCarrierVisible(
		ctx,
		definition.Carrier,
		visibleGroups,
	)
	if err != nil {
		return fmt.Errorf("authorize geofence carrier: %w", err)
	}
	if !visible {
		return commonerrors.ErrForbidden
	}
	return nil
}

func (s *Service) AuthorizeCarrierAccess(
	ctx context.Context,
	carrier string,
	visibleGroups []uuid.UUID,
) error {
	if visibleGroups == nil {
		return nil
	}
	visible, err := s.repository.IsCarrierVisible(
		ctx,
		strings.ToLower(strings.TrimSpace(carrier)),
		visibleGroups,
	)
	if err != nil {
		return fmt.Errorf("authorize geofence carrier: %w", err)
	}
	if !visible {
		return commonerrors.ErrForbidden
	}
	return nil
}

func (s *Service) AuthorizeBindingAccess(
	ctx context.Context,
	bindingID uuid.UUID,
	visibleGroups []uuid.UUID,
) error {
	if visibleGroups == nil {
		return nil
	}
	binding, err := s.repository.GetBinding(ctx, bindingID)
	if err != nil {
		return fmt.Errorf("get geofence binding for authorization: %w", err)
	}
	if binding == nil {
		return commonerrors.ErrNotFound
	}
	return s.AuthorizeDefinitionAccess(
		ctx,
		binding.GeofenceID,
		visibleGroups,
	)
}

func (s *Service) PreviewSettings(
	ctx context.Context,
	settings Settings,
) (SettingsPreview, error) {
	if s.settingsRepository == nil {
		return SettingsPreview{}, fmt.Errorf("geofence settings repository is not configured")
	}
	if err := ValidateSettings(settings); err != nil {
		return SettingsPreview{}, err
	}
	preview, err := s.settingsRepository.PreviewSettings(ctx, settings)
	if err != nil {
		return SettingsPreview{}, fmt.Errorf("preview geofence settings: %w", err)
	}
	return preview, nil
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	settings Settings,
	actorID uuid.UUID,
) (Settings, error) {
	if actorID == uuid.Nil {
		return Settings{}, fmt.Errorf(
			"authenticated actor is required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if s.settingsRepository == nil {
		return Settings{}, fmt.Errorf("geofence settings repository is not configured")
	}
	if err := ValidateSettings(settings); err != nil {
		return Settings{}, err
	}
	updated, err := s.settingsRepository.UpdateSettings(
		ctx,
		settings,
		actorID,
		s.now(),
	)
	if err != nil {
		return Settings{}, fmt.Errorf("update geofence settings: %w", err)
	}
	return updated, nil
}

type CreateDefinitionRequest struct {
	Name          string
	Carrier       string
	RuleType      RuleType
	OwnerDeviceID *uuid.UUID
	Geometry      json.RawMessage
	Policy        json.RawMessage
	ActorID       uuid.UUID
	VisibleGroups []uuid.UUID
}

type CreateDefinitionResult struct {
	Definition   *Definition `json:"definition"`
	DraftVersion *Version    `json:"draft_version"`
}

type CreateVersionRequest struct {
	GeofenceID uuid.UUID
	Geometry   json.RawMessage
	Policy     json.RawMessage
	ActorID    uuid.UUID
}

func (s *Service) CreateDraftVersion(
	ctx context.Context,
	request CreateVersionRequest,
) (*Version, error) {
	if request.GeofenceID == uuid.Nil || request.ActorID == uuid.Nil {
		return nil, fmt.Errorf(
			"geofence and actor are required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	definition, err := s.repository.GetDefinition(ctx, request.GeofenceID)
	if err != nil {
		return nil, fmt.Errorf("get geofence for new draft: %w", err)
	}
	if definition == nil {
		return nil, commonerrors.ErrNotFound
	}
	if definition.Status == DefinitionStatusArchived {
		return nil, fmt.Errorf(
			"archived geofence cannot be edited: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	geometry, err := ParseAndValidateGeometry(
		definition.RuleType,
		request.Geometry,
	)
	if err != nil {
		return nil, err
	}
	policy, err := validatePolicy(request.Policy)
	if err != nil {
		return nil, err
	}
	version := &Version{
		ID:           uuid.New(),
		GeofenceID:   definition.ID,
		Status:       VersionStatusDraft,
		GeometryJSON: geometry.JSON,
		BoundingBox:  geometry.BoundingBox,
		PolicyJSON:   policy,
		CreatedBy:    request.ActorID,
		CreatedAt:    s.now(),
	}
	if err := s.repository.CreateDraftVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("create geofence draft version: %w", err)
	}
	return version, nil
}

func (s *Service) PreviewDefinitionTransition(
	ctx context.Context,
	geofenceID uuid.UUID,
	target DefinitionStatus,
) (LifecycleImpact, error) {
	if geofenceID == uuid.Nil {
		return LifecycleImpact{}, fmt.Errorf(
			"geofence is required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	impact, err := s.repository.GetLifecycleImpact(ctx, geofenceID)
	if err != nil {
		return LifecycleImpact{}, fmt.Errorf(
			"get geofence lifecycle impact: %w",
			err,
		)
	}
	if err := validateDefinitionTransition(impact.CurrentStatus, target); err != nil {
		return LifecycleImpact{}, err
	}
	impact.TargetStatus = target
	impact.PreviewFingerprint = lifecyclePreviewFingerprint(impact)
	return impact, nil
}

func (s *Service) TransitionDefinition(
	ctx context.Context,
	geofenceID uuid.UUID,
	target DefinitionStatus,
	actorID uuid.UUID,
	reason string,
	previewFingerprint string,
) error {
	if geofenceID == uuid.Nil || actorID == uuid.Nil {
		return fmt.Errorf(
			"geofence and actor are required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || strings.TrimSpace(previewFingerprint) == "" {
		return fmt.Errorf(
			"reason and preview fingerprint are required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if err := s.repository.TransitionDefinition(
		ctx,
		geofenceID,
		target,
		actorID,
		reason,
		previewFingerprint,
		s.now(),
	); err != nil {
		return fmt.Errorf("transition geofence definition: %w", err)
	}
	return nil
}

func (s *Service) SuspendBinding(
	ctx context.Context,
	bindingID uuid.UUID,
	actorID uuid.UUID,
	reason string,
) (*Binding, error) {
	return s.transitionBinding(
		ctx,
		bindingID,
		BindingStatusSuspended,
		actorID,
		reason,
	)
}

func (s *Service) ResumeBinding(
	ctx context.Context,
	bindingID uuid.UUID,
	actorID uuid.UUID,
	reason string,
) (*Binding, error) {
	return s.transitionBinding(
		ctx,
		bindingID,
		BindingStatusActive,
		actorID,
		reason,
	)
}

func (s *Service) RemoveBinding(
	ctx context.Context,
	bindingID uuid.UUID,
	actorID uuid.UUID,
	reason string,
) (*Binding, error) {
	return s.transitionBinding(
		ctx,
		bindingID,
		BindingStatusRemoved,
		actorID,
		reason,
	)
}

func (s *Service) transitionBinding(
	ctx context.Context,
	bindingID uuid.UUID,
	target BindingStatus,
	actorID uuid.UUID,
	reason string,
) (*Binding, error) {
	if bindingID == uuid.Nil || actorID == uuid.Nil {
		return nil, fmt.Errorf(
			"binding and actor are required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	reason = strings.TrimSpace(reason)
	if (target == BindingStatusActive ||
		target == BindingStatusSuspended ||
		target == BindingStatusRemoved) && reason == "" {
		return nil, fmt.Errorf(
			"reason is required for binding transition: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	binding, err := s.repository.TransitionBinding(
		ctx,
		bindingID,
		target,
		actorID,
		reason,
		s.now(),
	)
	if err != nil {
		return nil, fmt.Errorf("transition geofence binding: %w", err)
	}
	return binding, nil
}

func (s *Service) CreateDefinition(
	ctx context.Context,
	request CreateDefinitionRequest,
) (*CreateDefinitionResult, error) {
	name := strings.TrimSpace(request.Name)
	carrier := strings.TrimSpace(request.Carrier)
	if err := validateDefinitionName(name); err != nil {
		return nil, err
	}
	if carrier == "" || request.ActorID == uuid.Nil {
		return nil, fmt.Errorf("name, carrier and actor are required: %w", commonerrors.ErrInvalidInput)
	}
	if request.VisibleGroups != nil {
		visible, err := s.repository.IsCarrierVisible(
			ctx,
			carrier,
			request.VisibleGroups,
		)
		if err != nil {
			return nil, fmt.Errorf("authorize geofence carrier: %w", err)
		}
		if !visible {
			return nil, commonerrors.ErrForbidden
		}
	}
	if request.RuleType == RuleTypeBaselineRadius && request.OwnerDeviceID == nil {
		return nil, fmt.Errorf("baseline owner is required: %w", commonerrors.ErrInvalidInput)
	}
	if request.RuleType == RuleTypeBaselineRadius &&
		request.VisibleGroups != nil {
		visible, err := s.repository.IsDeviceVisible(
			ctx,
			*request.OwnerDeviceID,
			request.VisibleGroups,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"authorize baseline owner device: %w",
				err,
			)
		}
		if !visible {
			return nil, commonerrors.ErrForbidden
		}
	}
	if request.RuleType == RuleTypePolygonAllowZone && request.OwnerDeviceID != nil {
		return nil, fmt.Errorf("polygon rule cannot have a baseline owner: %w", commonerrors.ErrInvalidInput)
	}
	geometry, err := ParseAndValidateGeometry(request.RuleType, request.Geometry)
	if err != nil {
		return nil, err
	}
	policy, err := validatePolicy(request.Policy)
	if err != nil {
		return nil, err
	}

	now := s.now()
	definition := &Definition{
		ID:            uuid.New(),
		Name:          name,
		Carrier:       carrier,
		RuleType:      request.RuleType,
		OwnerDeviceID: request.OwnerDeviceID,
		Status:        DefinitionStatusDraft,
		CreatedBy:     request.ActorID,
		UpdatedBy:     request.ActorID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	version := &Version{
		ID:           uuid.New(),
		GeofenceID:   definition.ID,
		Version:      1,
		Status:       VersionStatusDraft,
		GeometryJSON: geometry.JSON,
		BoundingBox:  geometry.BoundingBox,
		PolicyJSON:   policy,
		CreatedBy:    request.ActorID,
		CreatedAt:    now,
	}
	if err := s.repository.CreateDefinitionWithDraft(ctx, definition, version); err != nil {
		return nil, fmt.Errorf("create geofence with draft: %w", err)
	}
	return &CreateDefinitionResult{Definition: definition, DraftVersion: version}, nil
}

func (s *Service) RenameDefinition(
	ctx context.Context,
	geofenceID uuid.UUID,
	name string,
	actorID uuid.UUID,
) error {
	name = strings.TrimSpace(name)
	if geofenceID == uuid.Nil || actorID == uuid.Nil {
		return fmt.Errorf("geofence name and actor are invalid: %w", commonerrors.ErrInvalidInput)
	}
	if err := validateDefinitionName(name); err != nil {
		return err
	}
	if err := s.repository.UpdateDefinitionName(ctx, geofenceID, name, actorID, s.now()); err != nil {
		return fmt.Errorf("rename geofence: %w", err)
	}
	return nil
}

func validateDefinitionName(name string) error {
	if name == "" || len([]rune(name)) > 128 {
		return invalidDefinitionNameError()
	}
	return nil
}

func validatePolicy(raw json.RawMessage) (json.RawMessage, error) {
	var policy map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &policy) != nil || policy == nil {
		return nil, fmt.Errorf("policy must be a JSON object: %w", commonerrors.ErrInvalidInput)
	}
	action, ok := policy["exit_action"].(string)
	if !ok || (action != "notify_only" && action != "manual_review" && action != "deactivate") {
		return nil, fmt.Errorf("unsupported exit_action: %w", commonerrors.ErrInvalidInput)
	}
	normalized, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("normalize geofence policy: %w", err)
	}
	return normalized, nil
}

func (s *Service) PublishDraft(
	ctx context.Context,
	geofenceID, versionID, actorID uuid.UUID,
) error {
	definition, err := s.repository.GetDefinition(ctx, geofenceID)
	if err != nil {
		return fmt.Errorf("get geofence for publish: %w", err)
	}
	version, err := s.repository.GetVersion(ctx, versionID)
	if err != nil {
		return fmt.Errorf("get geofence version for publish: %w", err)
	}
	if definition == nil || version == nil {
		return commonerrors.ErrNotFound
	}
	if version.GeofenceID != definition.ID || version.Status != VersionStatusDraft {
		return fmt.Errorf("version is not a draft of the requested geofence: %w", commonerrors.ErrInvalidInput)
	}
	if err := s.repository.PublishVersion(ctx, geofenceID, versionID, actorID, s.now()); err != nil {
		return fmt.Errorf("publish geofence draft: %w", err)
	}
	return nil
}

func (s *Service) GetDefinition(ctx context.Context, id uuid.UUID) (*Definition, error) {
	definition, err := s.repository.GetDefinition(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get geofence definition: %w", err)
	}
	if definition == nil {
		return nil, commonerrors.ErrNotFound
	}
	return definition, nil
}

func (s *Service) ListDefinitions(
	ctx context.Context,
	filter DefinitionFilter,
) ([]Definition, error) {
	definitions, err := s.repository.ListDefinitions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list geofence definitions: %w", err)
	}
	return definitions, nil
}

func (s *Service) ListMapDefinitions(
	ctx context.Context,
	filter MapDefinitionFilter,
) ([]MapDefinition, error) {
	definitions, err := s.repository.ListMapDefinitions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list geofence map definitions: %w", err)
	}
	return definitions, nil
}

func (s *Service) ListVersions(
	ctx context.Context,
	geofenceID uuid.UUID,
) ([]Version, error) {
	if _, err := s.GetDefinition(ctx, geofenceID); err != nil {
		return nil, err
	}
	versions, err := s.repository.ListVersions(ctx, geofenceID)
	if err != nil {
		return nil, fmt.Errorf("list geofence versions: %w", err)
	}
	return versions, nil
}

func (s *Service) ListBindingDetails(
	ctx context.Context,
	filter BindingDetailFilter,
) (BindingDetailPage, error) {
	normalized, err := normalizeBindingDetailFilter(filter)
	if err != nil {
		return BindingDetailPage{}, err
	}
	if _, err := s.GetDefinition(ctx, normalized.GeofenceID); err != nil {
		return BindingDetailPage{}, err
	}
	page, err := s.repository.ListBindingDetails(ctx, normalized)
	if err != nil {
		return BindingDetailPage{}, fmt.Errorf(
			"list geofence binding details: %w",
			err,
		)
	}
	return page, nil
}
