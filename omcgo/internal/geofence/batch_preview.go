package geofence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

const (
	ReasonDeviceUnavailable      = "device_unavailable"
	ReasonGeofenceNotEnabled     = "geofence_not_enabled"
	ReasonCarrierMismatch        = "carrier_mismatch"
	ReasonBaselineOwnerMismatch  = "baseline_owner_mismatch"
	ReasonAlreadyBound           = "already_bound"
	ReasonBindingSuspended       = "binding_suspended"
	ReasonActiveRuleConflict     = "active_rule_conflict"
	ReasonBindingChanged         = "binding_changed"
	ReasonReassigned             = "reassigned"
	ReasonGeofenceVersionChanged = "geofence_version_changed"
)

func (s *Service) PreviewManualBindings(
	ctx context.Context,
	req ManualBindingPreviewRequest,
) (ManualBindingPreview, error) {
	if req.GeofenceID == uuid.Nil {
		return ManualBindingPreview{}, fmt.Errorf(
			"geofence is required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	inputs, err := normalizeBindingInputs(req.Inputs)
	if err != nil {
		return ManualBindingPreview{}, err
	}
	snapshot, err := s.repository.LoadManualBindSnapshot(
		ctx,
		req.GeofenceID,
		inputs,
		req.VisibleGroups,
	)
	if err != nil {
		return ManualBindingPreview{}, fmt.Errorf("load manual binding preview: %w", err)
	}
	return buildManualBindingPreview(snapshot)
}

func (s *Service) CreateManualBindJob(
	ctx context.Context,
	req CreateManualBindJobRequest,
) (BatchJobAccepted, error) {
	if req.GeofenceID == uuid.Nil || req.ActorID == uuid.Nil {
		return BatchJobAccepted{}, fmt.Errorf(
			"geofence and actor are required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	req.PreviewFingerprint = strings.TrimSpace(req.PreviewFingerprint)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.PreviewFingerprint == "" || req.Reason == "" {
		return BatchJobAccepted{}, fmt.Errorf(
			"preview fingerprint and reason are required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	inputs, err := normalizeBindingInputs(req.Inputs)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	scheduledAt := req.ScheduledAt
	if scheduledAt.IsZero() {
		scheduledAt = s.now()
	}
	accepted, err := s.repository.CreateManualBindJob(
		ctx,
		CreateManualBindJobParams{
			GeofenceID:         req.GeofenceID,
			ActorID:            req.ActorID,
			VisibleGroups:      cloneVisibleGroups(req.VisibleGroups),
			Inputs:             inputs,
			PreviewFingerprint: req.PreviewFingerprint,
			Reason:             req.Reason,
			ScheduledAt:        scheduledAt,
		},
	)
	if err != nil {
		return BatchJobAccepted{}, fmt.Errorf("create manual bind job: %w", err)
	}
	return accepted, nil
}

func (s *Service) GetManualBindJob(
	ctx context.Context,
	jobID uuid.UUID,
	actorID uuid.UUID,
	superAdministrator bool,
) (*BatchJob, error) {
	if jobID == uuid.Nil || actorID == uuid.Nil {
		return nil, fmt.Errorf(
			"job and actor are required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	job, err := s.repository.GetManualBindJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get manual bind job: %w", err)
	}
	if job == nil {
		return nil, commonerrors.ErrNotFound
	}
	if !superAdministrator && job.RequestedBy != actorID {
		return nil, commonerrors.ErrForbidden
	}
	return job, nil
}

func (s *Service) ListManualBindItems(
	ctx context.Context,
	filter BatchItemFilter,
	actorID uuid.UUID,
	superAdministrator bool,
) (BatchItemPage, error) {
	if filter.JobID == uuid.Nil || actorID == uuid.Nil {
		return BatchItemPage{}, fmt.Errorf(
			"job and actor are required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if filter.PageSize > MaxBatchItemPageSize {
		return BatchItemPage{}, fmt.Errorf(
			"batch item page size must not exceed %d: %w",
			MaxBatchItemPageSize,
			commonerrors.ErrInvalidInput,
		)
	}
	if filter.Status != "" && !validBatchItemStatus(filter.Status) {
		return BatchItemPage{}, fmt.Errorf(
			"invalid batch item status: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = DefaultBatchItemPageSize
	}
	filter.VisibleGroups = cloneVisibleGroups(filter.VisibleGroups)
	if _, err := s.GetManualBindJob(
		ctx,
		filter.JobID,
		actorID,
		superAdministrator,
	); err != nil {
		return BatchItemPage{}, err
	}
	page, err := s.repository.ListManualBindItems(ctx, filter)
	if err != nil {
		return BatchItemPage{}, fmt.Errorf("list manual bind items: %w", err)
	}
	return page, nil
}

func cloneVisibleGroups(visibleGroups []uuid.UUID) []uuid.UUID {
	if visibleGroups == nil {
		return nil
	}
	cloned := make([]uuid.UUID, len(visibleGroups))
	copy(cloned, visibleGroups)
	return cloned
}

func validBatchItemStatus(status BatchItemStatus) bool {
	switch status {
	case BatchItemPending, BatchItemSucceeded, BatchItemSkipped, BatchItemFailed:
		return true
	default:
		return false
	}
}

func buildManualBindingPreview(snapshot ManualBindSnapshot) (ManualBindingPreview, error) {
	preview := ManualBindingPreview{
		GeofenceID: snapshot.Definition.ID,
		RuleType:   snapshot.Definition.RuleType,
		InputCount: len(snapshot.Inputs),
		Items:      make([]BindingPreviewItem, 0, len(snapshot.Inputs)),
	}
	if snapshot.Definition.CurrentVersionID != nil {
		preview.GeofenceVersionID = *snapshot.Definition.CurrentVersionID
	}

	for _, fact := range snapshot.Inputs {
		item := bindingPreviewItem(fact)
		item.Decision, item.ReasonCode = classifyBindingCandidate(
			snapshot.Definition,
			fact,
		)
		switch item.Decision {
		case BindingDecisionEligible:
			preview.EligibleCount++
		case BindingDecisionMove:
			preview.MoveCount++
		default:
			preview.SkippedCount++
		}
		preview.Items = append(preview.Items, item)
	}

	fingerprint, err := manualBindingPreviewFingerprint(snapshot, preview.Items)
	if err != nil {
		return ManualBindingPreview{}, fmt.Errorf("fingerprint manual binding preview: %w", err)
	}
	preview.PreviewFingerprint = fingerprint
	return preview, nil
}

func bindingPreviewItem(fact BindingCandidateFact) BindingPreviewItem {
	item := BindingPreviewItem{
		InputKey: fact.Input.Key,
		Input:    fact.Input.Value,
	}
	if fact.Device == nil {
		if fact.Input.Kind == BindingInputDeviceID && fact.Input.DeviceID != nil {
			idCopy := *fact.Input.DeviceID
			item.DeviceID = &idCopy
		} else if fact.Input.Kind == BindingInputDeviceSN {
			item.DeviceSN = fact.Input.Value
		}
		return item
	}
	idCopy := fact.Device.ID
	item.DeviceID = &idCopy
	item.DeviceSN = fact.Device.SerialNumber
	if fact.ActiveRuleBindingID != nil && fact.ActiveRuleGeofenceID != nil {
		bindingID := *fact.ActiveRuleBindingID
		geofenceID := *fact.ActiveRuleGeofenceID
		item.SourceBindingID = &bindingID
		item.SourceGeofenceID = &geofenceID
	}
	return item
}

func classifyBindingCandidate(
	definition Definition,
	fact BindingCandidateFact,
) (BindingDecision, string) {
	if fact.Device == nil {
		return BindingDecisionSkipped, ReasonDeviceUnavailable
	}
	if definition.Status != DefinitionStatusEnabled || definition.CurrentVersionID == nil {
		return BindingDecisionSkipped, ReasonGeofenceNotEnabled
	}
	if fact.Device.Carrier != definition.Carrier {
		return BindingDecisionSkipped, ReasonCarrierMismatch
	}
	if definition.RuleType == RuleTypeBaselineRadius &&
		(definition.OwnerDeviceID == nil || *definition.OwnerDeviceID != fact.Device.ID) {
		return BindingDecisionSkipped, ReasonBaselineOwnerMismatch
	}
	if fact.SameGeofenceStatus != nil {
		switch *fact.SameGeofenceStatus {
		case BindingStatusActive:
			return BindingDecisionSkipped, ReasonAlreadyBound
		case BindingStatusSuspended:
			return BindingDecisionSkipped, ReasonBindingSuspended
		}
	}
	if fact.ActiveRuleGeofenceID != nil && *fact.ActiveRuleGeofenceID != definition.ID {
		return BindingDecisionMove, ReasonReassigned
	}
	return BindingDecisionEligible, ""
}

type fingerprintDocument struct {
	GeofenceID        uuid.UUID
	GeofenceVersionID uuid.UUID
	GeofenceUpdatedAt time.Time
	GeofenceStatus    DefinitionStatus
	RuleType          RuleType
	Carrier           string
	OwnerDeviceID     *uuid.UUID
	VisibilityDigest  string
	Items             []fingerprintItem
}

type fingerprintItem struct {
	InputKey              string
	DeviceID              *uuid.UUID
	DeviceSN              string
	DeviceCarrier         string
	SameGeofenceBindingID *uuid.UUID
	SameGeofenceStatus    *BindingStatus
	ActiveRuleBindingID   *uuid.UUID
	ActiveRuleGeofenceID  *uuid.UUID
	Decision              BindingDecision
	ReasonCode            string
}

func manualBindingPreviewFingerprint(
	snapshot ManualBindSnapshot,
	items []BindingPreviewItem,
) (string, error) {
	previewItems := make(map[string]BindingPreviewItem, len(items))
	for _, item := range items {
		previewItems[item.InputKey] = item
	}
	canonicalItems := make([]fingerprintItem, 0, len(snapshot.Inputs))
	for _, fact := range snapshot.Inputs {
		item := previewItems[fact.Input.Key]
		deviceCarrier := ""
		if fact.Device != nil {
			deviceCarrier = fact.Device.Carrier
		}
		canonicalItems = append(canonicalItems, fingerprintItem{
			InputKey:              item.InputKey,
			DeviceID:              item.DeviceID,
			DeviceSN:              item.DeviceSN,
			DeviceCarrier:         deviceCarrier,
			SameGeofenceBindingID: fact.SameGeofenceBindingID,
			SameGeofenceStatus:    fact.SameGeofenceStatus,
			ActiveRuleBindingID:   fact.ActiveRuleBindingID,
			ActiveRuleGeofenceID:  fact.ActiveRuleGeofenceID,
			Decision:              item.Decision,
			ReasonCode:            item.ReasonCode,
		})
	}
	sort.Slice(canonicalItems, func(i, j int) bool {
		return canonicalItems[i].InputKey < canonicalItems[j].InputKey
	})

	versionID := uuid.Nil
	if snapshot.Definition.CurrentVersionID != nil {
		versionID = *snapshot.Definition.CurrentVersionID
	}
	document := fingerprintDocument{
		GeofenceID:        snapshot.Definition.ID,
		GeofenceVersionID: versionID,
		GeofenceUpdatedAt: snapshot.Definition.UpdatedAt,
		GeofenceStatus:    snapshot.Definition.Status,
		RuleType:          snapshot.Definition.RuleType,
		Carrier:           snapshot.Definition.Carrier,
		OwnerDeviceID:     snapshot.Definition.OwnerDeviceID,
		VisibilityDigest:  snapshot.VisibilityDigest,
		Items:             canonicalItems,
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("marshal manual binding fingerprint: %w", err)
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
