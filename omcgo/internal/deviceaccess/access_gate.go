package deviceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	devicepkg "github.com/omcgo/omcgo/internal/device"
)

// PolicyProvider supplies the immutable policy version used for one decision.
// The provider may be backed by PostgreSQL during coordination or by a compiled
// snapshot once the ACS fast path is wired.
type PolicyProvider interface {
	Load(ctx context.Context, carrier string) (CompiledPolicy, error)
}

type AccessGate struct {
	repository Repository
	assets     AssetEvidenceResolver
	policies   PolicyProvider
	evaluator  Evaluator
	now        func() time.Time
	gpsProbes  GPSProbePlanner
	settings   RuntimeSettingsReader
}

func NewAccessGate(
	repository Repository,
	assets AssetEvidenceResolver,
	policies PolicyProvider,
) *AccessGate {
	return &AccessGate{
		repository: repository,
		assets:     assets,
		policies:   policies,
		now:        time.Now,
	}
}

func (g *AccessGate) SetClock(now func() time.Time) {
	if now != nil {
		g.now = now
	}
}

func (g *AccessGate) SetGPSProbePlanner(planner GPSProbePlanner) {
	g.gpsProbes = planner
}

func (g *AccessGate) SetRuntimeSettingsReader(reader RuntimeSettingsReader) {
	g.settings = reader
}

// Admit implements the device package's Inform access boundary. ACS has
// already evaluated protocol authentication before publishing Inform events;
// this method controls formal registration and maintains authorization state
// for subsequent Inform sessions.
func (g *AccessGate) Admit(ctx context.Context, observation devicepkg.AccessObservation) (devicepkg.AccessDecision, error) {
	return g.admit(ctx, observation)
}

func (g *AccessGate) admit(ctx context.Context, observation devicepkg.AccessObservation) (devicepkg.AccessDecision, error) {
	carrierCode := strings.TrimSpace(string(observation.Carrier))
	serialNumber := strings.TrimSpace(observation.SerialNumber)
	if carrierCode == "" {
		return devicepkg.AccessDecision{}, ErrCarrierRequired
	}
	if serialNumber == "" {
		return devicepkg.AccessDecision{}, ErrSerialNumberRequired
	}
	if g == nil || g.repository == nil || g.assets == nil || g.policies == nil {
		return devicepkg.AccessDecision{}, fmt.Errorf("create device access decision: %w", ErrAccessGateDependencyMissing)
	}
	enabled, err := runtimeAccessEnabled(ctx, g.settings, carrierCode)
	if err != nil {
		return devicepkg.AccessDecision{}, fmt.Errorf("load device access business switch: %w", err)
	}
	if !enabled {
		return devicepkg.AccessDecision{State: devicepkg.AccessDecisionBypassed, ReasonCode: "access_control_disabled"}, nil
	}

	evaluatedAt := g.now().UTC()
	triggerType := strings.TrimSpace(observation.TriggerType)
	if triggerType == "" {
		triggerType = TriggerInformPeriodic
	}
	remoteIP, _ := netip.ParseAddr(strings.TrimSpace(observation.RemoteIP))
	remoteIP = remoteIP.WithZone("").Unmap()
	technology, rfControlPaths := devicepkg.CandidateRFIdentitySnapshot(observation)
	asset, err := g.assets.Resolve(ctx, carrierCode, serialNumber)
	if err != nil {
		return devicepkg.AccessDecision{}, fmt.Errorf("resolve device access asset: %w", err)
	}
	candidate, err := g.repository.UpsertCandidateObservation(ctx, Observation{
		Carrier:         carrierCode,
		SerialNumber:    serialNumber,
		DeviceID:        asset.DeviceID,
		OUI:             observation.OUI,
		ProductClass:    observation.ProductClass,
		SoftwareVersion: observation.SoftwareVersion,
		Technology:      technology,
		RFControlPaths:  rfControlPaths,
		RemoteIP:        remoteIP,
		ObservedAt:      evaluatedAt,
		ExpiresAt:       evaluatedAt.Add(candidateObservationTTL),
	})
	if err != nil {
		return devicepkg.AccessDecision{}, fmt.Errorf("observe device access candidate: %w", err)
	}
	context, err := g.repository.LoadEvaluationContext(ctx, carrierCode, serialNumber)
	if err != nil {
		return devicepkg.AccessDecision{}, fmt.Errorf("load device access context: %w", err)
	}
	policy, err := g.policies.Load(ctx, carrierCode)
	if err != nil {
		return devicepkg.AccessDecision{}, fmt.Errorf("load device access policy: %w", err)
	}
	existingState := AccessStateReviewRequired
	if context.State != nil {
		existingState = context.State.State
	}
	baseInput := EvaluationInput{
		Carrier:                carrierCode,
		SerialNumber:           serialNumber,
		OUI:                    observation.OUI,
		ProductClass:           observation.ProductClass,
		AuthenticationRequired: accessAuthenticationRequired(observation.AuthMethod),
		Authenticated:          observation.Authenticated,
		AssetRetired:           asset.AssetRetired,
		ExistingState:          existingState,
		EvaluatedAt:            evaluatedAt,
		CollectionDeadline:     collectionDeadline(context.State),
		Evidence:               evidenceSetFromContext(context, asset, evaluatedAt),
		Policy:                 policy,
	}
	initialDecision := g.evaluator.Evaluate(baseInput)
	forceObservedIPRefresh := context.State != nil &&
		(decisionHasCheckResult(initialDecision, ConditionTypeObservedIP, CheckStale) ||
			(context.State.State == AccessStateRevalidating &&
				decisionHasCheckResult(initialDecision, ConditionTypeObservedIP, CheckFailed)))
	previousEvidence := context.Evidence
	evidence, err := buildInformEvidence(
		context,
		observation,
		asset,
		evaluatedAt,
		forceObservedIPRefresh,
	)
	if err != nil {
		return devicepkg.AccessDecision{}, fmt.Errorf("build inform access evidence: %w", err)
	}
	if evidence != nil {
		context.Evidence = *evidence
	}
	baseInput.Evidence = evidenceSetFromContext(context, asset, evaluatedAt)
	decision := g.evaluator.Evaluate(baseInput)
	if context.State != nil && context.State.State == AccessStateRevalidating &&
		failedCheckEvidenceRefreshed(decision, previousEvidence, context.Evidence) {
		baseInput.ConfirmedMismatch = true
		decision = g.evaluator.Evaluate(baseInput)
	}
	probeNeeds, shouldProbe := accessProbeNeeds(context, decision, evaluatedAt)
	unchanged := decisionUnchanged(context, decision, policy.VersionID) &&
		projectionOwnerUnchanged(context.State, asset.DeviceID)
	identitySnapshot := buildIdentitySnapshot(
		observation, asset, candidate, decision, "", evaluatedAt,
	)
	var savedDecisionID string
	if !unchanged {
		policyVersionID, err := policyVersionUUID(policy.VersionID)
		if err != nil {
			return devicepkg.AccessDecision{}, err
		}
		payload, err := json.Marshal(struct {
			SerialNumber      string      `json:"serial_number"`
			Carrier           string      `json:"carrier"`
			OUI               string      `json:"oui,omitempty"`
			ProductClass      string      `json:"product_class,omitempty"`
			SoftwareVersion   string      `json:"software_version,omitempty"`
			State             AccessState `json:"state"`
			ReasonCode        ReasonCode  `json:"reason_code"`
			PolicyVersion     string      `json:"policy_version"`
			DecisionVersion   int64       `json:"decision_version"`
			EvidenceVersion   int64       `json:"evidence_version"`
			NormalTasksFrozen bool        `json:"normal_tasks_frozen"`
			PublishedAt       time.Time   `json:"published_at"`
			TriggerType       string      `json:"trigger_type"`
		}{
			SerialNumber:      serialNumber,
			Carrier:           carrierCode,
			OUI:               observation.OUI,
			ProductClass:      observation.ProductClass,
			SoftwareVersion:   observation.SoftwareVersion,
			State:             decision.State,
			ReasonCode:        decision.ReasonCode,
			PolicyVersion:     policy.VersionID,
			DecisionVersion:   decisionVersion(context) + 1,
			EvidenceVersion:   context.Evidence.Version,
			NormalTasksFrozen: decision.FreezeNormal,
			PublishedAt:       evaluatedAt,
			TriggerType:       triggerType,
		})
		if err != nil {
			return devicepkg.AccessDecision{}, fmt.Errorf("encode device access decision event: %w", err)
		}
		var candidateID *uuid.UUID
		if asset.DeviceID == nil && candidate.ID != uuid.Nil {
			candidateID = &candidate.ID
		}
		saved, err := g.repository.SaveDecision(ctx, DecisionChange{
			Carrier:                 carrierCode,
			SerialNumber:            serialNumber,
			DeviceID:                asset.DeviceID,
			CandidateID:             candidateID,
			TriggerType:             triggerType,
			TriggerEventID:          observation.EventID,
			ExpectedDecisionVersion: decisionVersion(context),
			PolicyVersionID:         policyVersionID,
			EvidenceVersion:         context.Evidence.Version,
			OccurredAt:              evaluatedAt,
			DecisionExpiresAt:       nextDecisionDeadline(context.State, decision, policy, evaluatedAt),
			Decision:                decision,
			Evidence:                evidence,
			IdentitySnapshot:        &identitySnapshot,
			Outbox: OutboxEvent{
				EventType: decisionEventType(decision.State),
				EventKey:  fmt.Sprintf("device-access:%s:%s:%s", carrierCode, serialNumber, observation.EventID),
				Payload:   payload,
			},
		})
		if err != nil {
			return devicepkg.AccessDecision{}, fmt.Errorf("save device access decision: %w", err)
		}
		savedDecisionID = saved.ID.String()
	}
	if unchanged {
		identitySnapshot.DecisionID = savedDecisionID
		if err := g.repository.SaveIdentitySnapshot(ctx, identitySnapshot); err != nil {
			return devicepkg.AccessDecision{}, fmt.Errorf("save inform identity snapshot: %w", err)
		}
	}
	if unchanged && !shouldProbe {
		return devicepkg.AccessDecision{
			State:      string(decision.State),
			ReasonCode: string(decision.ReasonCode),
		}, nil
	}
	// The access projection and its evidence must become durable before a task
	// is enqueued. Besides preserving the reason for a failed path resolution,
	// this lets the dequeue guard evaluate the probe against the new state.
	if shouldProbe && g.gpsProbes != nil {
		sourceID := ""
		if candidate.ID != uuid.Nil {
			sourceID = candidate.ID.String()
		}
		if err := g.gpsProbes.EnsureGPSProbe(ctx, GPSProbeRequest{
			Carrier:         carrierCode,
			SerialNumber:    serialNumber,
			ProductClass:    observation.ProductClass,
			SoftwareVersion: observation.SoftwareVersion,
			EvidenceVersion: context.Evidence.Version + 1,
			SourceID:        sourceID,
			NeedGPS:         probeNeeds.gps,
			NeedTAC:         probeNeeds.tac,
			NeedECGI:        probeNeeds.ecgi,
		}); err != nil {
			return devicepkg.AccessDecision{}, fmt.Errorf("ensure access evidence probe: %w", err)
		}
	}
	return devicepkg.AccessDecision{
		State:      string(decision.State),
		ReasonCode: string(decision.ReasonCode),
	}, nil
}

func buildIdentitySnapshot(
	observation devicepkg.AccessObservation,
	asset AssetEvidence,
	candidate Candidate,
	decision Decision,
	decisionID string,
	observedAt time.Time,
) IdentitySnapshot {
	identityResult := identityCheckResult(observation, asset)
	status := IdentityStatusResolved
	reason := ReasonCode("")
	switch identityResult {
	case CheckFailed:
		status = IdentityStatusConflict
		reason = ReasonIdentityMismatch
	case CheckMissing:
		status = IdentityStatusUnresolved
		reason = identityMissingReason(observation)
	}
	if decision.ReasonCode == ReasonOwnershipMismatch {
		status = IdentityStatusConflict
		reason = ReasonOwnershipMismatch
	}
	remoteIP := net.ParseIP(strings.TrimSpace(observation.RemoteIP))
	snapshot := IdentitySnapshot{
		ID:                 uuid.NewString(),
		RequestID:          strings.TrimSpace(observation.EventID),
		DecisionID:         decisionID,
		Carrier:            strings.TrimSpace(string(observation.Carrier)),
		SerialNumber:       strings.TrimSpace(observation.SerialNumber),
		DeviceCode:         strings.TrimSpace(observation.DeviceCode),
		CloudKey:           strings.TrimSpace(observation.CloudKey),
		OUI:                strings.ToUpper(strings.TrimSpace(observation.OUI)),
		ProductClass:       strings.TrimSpace(observation.ProductClass),
		RawRemoteIP:        remoteIP,
		ObservedRemoteIP:   remoteIP,
		InformEvent:        strings.TrimSpace(observation.InformEvent),
		InformTime:         observedAt,
		IdentitySource:     "inform_device_id",
		IdentityStatus:     status,
		IdentityReasonCode: reason,
		CreatedAt:          observedAt,
	}
	if snapshot.DeviceCode == "" && asset.SiteID != nil {
		snapshot.DeviceCode = strings.TrimSpace(*asset.SiteID)
	}
	if asset.DeviceID != nil {
		snapshot.DeviceID = asset.DeviceID.String()
	}
	if candidate.ID != uuid.Nil {
		snapshot.CandidateID = candidate.ID.String()
	}
	return snapshot
}

func projectionOwnerUnchanged(state *AccessStateProjection, deviceID *uuid.UUID) bool {
	if state == nil || deviceID == nil {
		return true
	}
	return state.DeviceID != nil && *state.DeviceID == *deviceID
}

type accessCheckEvidence struct {
	CheckResult          CheckResult         `json:"check_result"`
	ObservedOUI          string              `json:"observed_oui,omitempty"`
	ObservedProductClass string              `json:"observed_product_class,omitempty"`
	ExpectedOUI          string              `json:"expected_oui,omitempty"`
	ExpectedProductClass string              `json:"expected_product_class,omitempty"`
	AssetSource          AssetEvidenceSource `json:"asset_source,omitempty"`
	Authenticated        bool                `json:"authenticated,omitempty"`
	CredentialStatus     CredentialStatus    `json:"credential_status,omitempty"`
	AuthMethod           string              `json:"auth_method,omitempty"`
	CredentialID         string              `json:"credential_id,omitempty"`
	DeviceCode           string              `json:"device_code,omitempty"`
	CloudKey             string              `json:"cloud_key,omitempty"`
}

func buildInformEvidence(
	current EvaluationContext,
	observation devicepkg.AccessObservation,
	asset AssetEvidence,
	observedAt time.Time,
	forceObservedIPRefresh bool,
) (*EvidenceBatch, error) {
	identityResult := identityCheckResult(observation, asset)
	ownershipResult := CheckMissing
	if asset.Source != AssetEvidenceSourceUnknown {
		ownershipResult = CheckPassed
	}
	credentialStatus := CredentialStatusUnknown
	securityResult := CheckFailed
	if observation.Authenticated {
		credentialStatus = CredentialStatusActive
		securityResult = CheckPassed
	} else if !accessAuthenticationRequired(observation.AuthMethod) {
		credentialStatus = CredentialStatusNotConfigured
		securityResult = CheckSkipped
	}

	replacements := make([]EvidenceRecord, 0, 4)
	for _, input := range []struct {
		typeName ConditionType
		status   EvidenceStatus
		source   string
		value    any
	}{
		{
			typeName: ConditionTypeIdentity,
			status:   evidenceStatusForCheck(identityResult),
			source:   "inform_device_id",
			value: accessCheckEvidence{
				CheckResult:          identityResult,
				ObservedOUI:          strings.TrimSpace(observation.OUI),
				ObservedProductClass: strings.TrimSpace(observation.ProductClass),
				ExpectedOUI:          asset.ExpectedOUI,
				ExpectedProductClass: asset.ExpectedProductClass,
				DeviceCode:           strings.TrimSpace(observation.DeviceCode),
				CloudKey:             strings.TrimSpace(observation.CloudKey),
			},
		},
		{
			typeName: ConditionTypeAsset,
			status:   evidenceStatusForCheck(ownershipResult),
			source:   "asset_inventory",
			value:    accessCheckEvidence{CheckResult: ownershipResult, AssetSource: asset.Source},
		},
		{
			typeName: ConditionTypeSecurity,
			status:   evidenceStatusForCheck(securityResult),
			source:   "acs_http_auth",
			value: accessCheckEvidence{
				CheckResult:      securityResult,
				Authenticated:    observation.Authenticated,
				CredentialStatus: credentialStatus,
				AuthMethod:       strings.TrimSpace(observation.AuthMethod),
				CredentialID:     strings.TrimSpace(observation.CredentialID),
			},
		},
	} {
		raw, err := json.Marshal(input.value)
		if err != nil {
			return nil, fmt.Errorf("encode %s evidence: %w", input.typeName, err)
		}
		replacements = append(replacements, EvidenceRecord{
			Type: input.typeName, Status: input.status, NormalizedValue: raw,
			ValueHash: evidenceHash(raw), Source: input.source, ObservedAt: observedAt,
		})
	}
	remoteIP, err := netip.ParseAddr(strings.TrimSpace(observation.RemoteIP))
	remoteStatus := EvidenceStatusAvailable
	remoteValue := ""
	if err != nil {
		remoteStatus = EvidenceStatusMissing
	} else {
		remoteValue = remoteIP.WithZone("").Unmap().String()
	}
	remoteRaw, err := json.Marshal(remoteValue)
	if err != nil {
		return nil, fmt.Errorf("encode observed IP evidence: %w", err)
	}
	replacements = append(replacements, EvidenceRecord{
		Type: ConditionTypeObservedIP, Status: remoteStatus, NormalizedValue: remoteRaw,
		ValueHash: evidenceHash(remoteRaw), Source: "transport_peer", ObservedAt: observedAt,
	})

	changed := informEvidenceChanged(current.Evidence.Records, replacements)
	if !changed && !forceObservedIPRefresh {
		return nil, nil
	}
	return &EvidenceBatch{
		Carrier:      strings.TrimSpace(string(observation.Carrier)),
		SerialNumber: strings.TrimSpace(observation.SerialNumber),
		Version:      current.Evidence.Version + 1,
		Records:      mergeEvidenceRecords(current.Evidence.Records, replacements),
	}, nil
}

func decisionHasCheckResult(decision Decision, conditionType ConditionType, result CheckResult) bool {
	for _, check := range decision.Checks {
		if check.CheckType == conditionType && check.Result == result {
			return true
		}
	}
	return false
}

func failedCheckEvidenceRefreshed(decision Decision, previous, current EvidenceBatch) bool {
	previousByType := make(map[ConditionType]EvidenceRecord, len(previous.Records))
	currentByType := make(map[ConditionType]EvidenceRecord, len(current.Records))
	for _, record := range previous.Records {
		previousByType[record.Type] = record
	}
	for _, record := range current.Records {
		currentByType[record.Type] = record
	}
	for _, check := range decision.Checks {
		if check.Result != CheckFailed {
			continue
		}
		before, hadBefore := previousByType[check.CheckType]
		after, hasAfter := currentByType[check.CheckType]
		if hasAfter && (!hadBefore || after.ValueHash != before.ValueHash || after.ObservedAt.After(before.ObservedAt)) {
			return true
		}
	}
	return false
}

func identityCheckResult(observation devicepkg.AccessObservation, asset AssetEvidence) CheckResult {
	if observation.DeviceCodeRequired && strings.TrimSpace(observation.DeviceCode) == "" {
		return CheckMissing
	}
	if observation.CloudKeyRequired && strings.TrimSpace(observation.CloudKey) == "" {
		return CheckMissing
	}
	if observation.DeviceCodeRequired && asset.SiteID != nil &&
		strings.TrimSpace(observation.DeviceCode) != "" &&
		strings.TrimSpace(observation.DeviceCode) != strings.TrimSpace(*asset.SiteID) {
		return CheckFailed
	}
	observedOUI := strings.ToUpper(strings.TrimSpace(observation.OUI))
	observedProductClass := strings.TrimSpace(observation.ProductClass)
	expectedOUI := strings.ToUpper(strings.TrimSpace(asset.ExpectedOUI))
	expectedProductClass := strings.TrimSpace(asset.ExpectedProductClass)
	if expectedOUI != "" || expectedProductClass != "" {
		if (expectedOUI != "" && observedOUI == "") ||
			(expectedProductClass != "" && observedProductClass == "") {
			return CheckMissing
		}
		if (expectedOUI != "" && observedOUI != expectedOUI) ||
			(expectedProductClass != "" && observedProductClass != expectedProductClass) {
			return CheckFailed
		}
		return CheckPassed
	}
	if observation.CarrierIdentityResolved && observedOUI != "" {
		return CheckPassed
	}
	return CheckMissing
}

func identityMissingReason(observation devicepkg.AccessObservation) ReasonCode {
	if observation.DeviceCodeRequired && strings.TrimSpace(observation.DeviceCode) == "" {
		return ReasonDeviceCodeMissing
	}
	if observation.CloudKeyRequired && strings.TrimSpace(observation.CloudKey) == "" {
		return ReasonCloudKeyMissing
	}
	return ReasonIdentityUnverified
}

func evidenceStatusForCheck(result CheckResult) EvidenceStatus {
	if result == CheckMissing {
		return EvidenceStatusMissing
	}
	return EvidenceStatusAvailable
}

func informEvidenceChanged(current, proposed []EvidenceRecord) bool {
	byType := make(map[ConditionType]EvidenceRecord, len(current))
	for _, record := range current {
		byType[record.Type] = record
	}
	for _, record := range proposed {
		previous, exists := byType[record.Type]
		if !exists || previous.Status != record.Status || previous.ValueHash != record.ValueHash || previous.Source != record.Source {
			return true
		}
	}
	return false
}

type accessProbeRequirement struct {
	tac  bool
	ecgi bool
	gps  bool
}

const accessProbeRetryBackoff = time.Minute

func accessProbeNeeds(context EvaluationContext, decision Decision, now time.Time) (accessProbeRequirement, bool) {
	isCollecting := decision.State == AccessStateCollecting &&
		(decision.ReasonCode == ReasonEvidenceMissing ||
			decision.ReasonCode == ReasonEvidenceStale ||
			decision.ReasonCode == ReasonEvidenceCollectionFailed ||
			decision.ReasonCode == ReasonEvidenceSystemError)
	isConfirmingMismatch := decision.State == AccessStateRevalidating &&
		decision.ReasonCode == ReasonRuleMismatchPendingConfirmation
	if !isCollecting && !isConfirmingMismatch {
		return accessProbeRequirement{}, false
	}
	collectionRetryDue := (decision.ReasonCode == ReasonEvidenceCollectionFailed || decision.ReasonCode == ReasonEvidenceSystemError) &&
		failedEvidenceRetryDue(context.Evidence.Records, now)
	if !collectionRetryDue && decision.ReasonCode != ReasonEvidenceMissing && context.State != nil &&
		context.State.State == decision.State &&
		context.State.EvidenceVersion == context.Evidence.Version {
		return accessProbeRequirement{}, false
	}
	var requirement accessProbeRequirement
	for _, check := range decision.Checks {
		if check.Result != CheckMissing && check.Result != CheckStale &&
			!(isConfirmingMismatch && check.Result == CheckFailed) &&
			!(check.Result == CheckError && (check.ReasonCode == ReasonEvidenceCollectionFailed || check.ReasonCode == ReasonEvidenceSystemError)) {
			continue
		}
		switch check.CheckType {
		case ConditionTypeTAC:
			requirement.tac = true
		case ConditionTypeECGI:
			requirement.ecgi = true
		case ConditionTypeGPS:
			requirement.gps = true
		}
	}
	return requirement, requirement.tac || requirement.ecgi || requirement.gps
}

func failedEvidenceRetryDue(records []EvidenceRecord, now time.Time) bool {
	var latest time.Time
	for _, record := range records {
		if (record.Status == EvidenceStatusCollectionFailed || record.Status == EvidenceStatusSystemError) && record.ObservedAt.After(latest) {
			latest = record.ObservedAt
		}
	}
	return !latest.IsZero() && !now.Before(latest.Add(accessProbeRetryBackoff))
}

const candidateObservationTTL = 24 * time.Hour

const defaultEvidenceCollectionTimeout = 15 * time.Minute

func collectionDeadline(state *AccessStateProjection) *time.Time {
	if state == nil || state.DecisionExpiresAt == nil {
		return nil
	}
	value := state.DecisionExpiresAt.UTC()
	return &value
}

func nextDecisionDeadline(state *AccessStateProjection, decision Decision, policy CompiledPolicy, now time.Time) *time.Time {
	if decision.State != AccessStateCollecting {
		return nil
	}
	if state != nil && state.State == AccessStateCollecting && state.DecisionExpiresAt != nil {
		value := state.DecisionExpiresAt.UTC()
		return &value
	}
	timeout := policy.CollectionTimeout
	if timeout <= 0 {
		timeout = defaultEvidenceCollectionTimeout
	}
	value := now.UTC().Add(timeout)
	return &value
}

var ErrAccessGateDependencyMissing = fmt.Errorf("device access gate dependency missing")

func decisionVersion(context EvaluationContext) int64 {
	if context.State == nil {
		return 0
	}
	return context.State.DecisionVersion
}

func decisionUnchanged(context EvaluationContext, decision Decision, policyVersion string) bool {
	if context.State == nil || context.State.State != decision.State ||
		context.State.ReasonCode != decision.ReasonCode ||
		context.State.EffectiveDecision != decision.EffectiveAction ||
		context.State.EvidenceVersion != context.Evidence.Version {
		return false
	}
	if context.State.PolicyVersionID == nil {
		return strings.TrimSpace(policyVersion) == ""
	}
	id, err := uuid.Parse(policyVersion)
	return err == nil && *context.State.PolicyVersionID == id
}

func policyVersionUUID(version string) (*uuid.UUID, error) {
	if strings.TrimSpace(version) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(version)
	if err != nil {
		return nil, fmt.Errorf("parse device access policy version %q: %w", version, err)
	}
	return &id, nil
}

func decisionEventType(state AccessState) string {
	switch state {
	case AccessStateAccepted:
		return event.SubjectDeviceAccessAccepted
	case AccessStateRejected:
		return event.SubjectDeviceAccessRejected
	case AccessStateRevoked:
		return event.SubjectDeviceAccessRevoked
	case AccessStateRevalidating:
		return event.SubjectDeviceAccessRevalidating
	case AccessStateCollecting:
		return event.SubjectDeviceAccessCollectRequested
	default:
		return event.SubjectDeviceAccessReviewRequired
	}
}

func evidenceSetFromContext(context EvaluationContext, asset AssetEvidence, evaluatedAt time.Time) EvidenceSet {
	identity := CheckMissing
	ownership := CheckMissing
	if asset.Source != AssetEvidenceSourceUnknown {
		ownership = CheckPassed
	}
	values := make(map[ConditionType]EvidenceValue, len(context.Evidence.Records))
	for _, record := range context.Evidence.Records {
		status := record.Status
		if status == "" {
			status = EvidenceStatusAvailable
		}
		value := EvidenceValue{
			Status:     status,
			Source:     record.Source,
			ObservedAt: record.ObservedAt,
		}
		if record.ExpiresAt != nil && !evaluatedAt.Before(*record.ExpiresAt) {
			value.Status = EvidenceStatusMissing
		}
		if text, ok := normalizedText(record.NormalizedValue); ok {
			value.Text = text
		} else if texts, ok := normalizedTexts(record.NormalizedValue); ok {
			value.Texts = texts
		} else if point, ok := normalizedPoint(record.NormalizedValue); ok {
			value.Point = &point
		}
		switch record.Type {
		case ConditionTypeTAC, ConditionTypeECGI, ConditionTypeObservedIP, ConditionTypeGPS:
			values[record.Type] = value
		case ConditionTypeIdentity:
			identity = checkResultFromEvidence(record, identity)
		case ConditionTypeAsset:
			ownership = checkResultFromEvidence(record, ownership)
		}
	}
	return EvidenceSet{Identity: identity, Ownership: ownership, Values: values}
}

func checkResultFromEvidence(record EvidenceRecord, fallback CheckResult) CheckResult {
	if record.Status == EvidenceStatusMissing {
		return CheckMissing
	}
	var payload accessCheckEvidence
	if err := json.Unmarshal(record.NormalizedValue, &payload); err == nil && payload.CheckResult != "" {
		return payload.CheckResult
	}
	if record.Status == EvidenceStatusAvailable {
		return CheckPassed
	}
	return fallback
}

func authenticationFromEvidence(context EvaluationContext) (authenticated, required bool, status CredentialStatus) {
	for _, record := range context.Evidence.Records {
		if record.Type != ConditionTypeSecurity {
			continue
		}
		var payload accessCheckEvidence
		if err := json.Unmarshal(record.NormalizedValue, &payload); err != nil {
			return false, true, CredentialStatusUnknown
		}
		status = payload.CredentialStatus
		if status == "" {
			status = CredentialStatusUnknown
		}
		return payload.Authenticated, status != CredentialStatusNotConfigured, status
	}
	return false, true, CredentialStatusUnknown
}

func accessAuthenticationRequired(method string) bool {
	method = strings.ToLower(strings.TrimSpace(method))
	// Only the explicit compatibility marker emitted by NoopAuthenticator may
	// skip the hard gate. Missing/unknown methods remain fail-closed so replayed
	// or malformed events cannot silently downgrade authentication.
	return method != "none"
}

func normalizedText(raw json.RawMessage) (string, bool) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, true
	}
	return "", false
}

func normalizedTexts(raw json.RawMessage) ([]string, bool) {
	var texts []string
	if err := json.Unmarshal(raw, &texts); err == nil && len(texts) > 0 {
		return sortedUnique(texts), true
	}
	var payload radioEvidencePayload
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload.Values) == 0 || len(payload.Instances) == 0 {
		return nil, false
	}
	return sortedUnique(payload.Values), true
}

func normalizedPoint(raw json.RawMessage) (GeoPoint, bool) {
	var payload struct {
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Latitude == nil || payload.Longitude == nil {
		return GeoPoint{}, false
	}
	if *payload.Latitude < -90 || *payload.Latitude > 90 || *payload.Longitude < -180 || *payload.Longitude > 180 {
		return GeoPoint{}, false
	}
	return GeoPoint{Latitude: *payload.Latitude, Longitude: *payload.Longitude}, true
}

var _ devicepkg.AccessGate = (*AccessGate)(nil)
