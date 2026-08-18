package deviceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ReevaluationHandler interface {
	Handle(ctx context.Context, request ReevaluationRequest) error
}

type ReevaluationCoordinator struct {
	repository Repository
	assets     AssetEvidenceResolver
	policies   PolicyProvider
	evaluator  Evaluator
	now        func() time.Time
	gpsProbes  GPSProbePlanner
	settings   RuntimeSettingsReader
}

func (c *ReevaluationCoordinator) SetGPSProbePlanner(planner GPSProbePlanner) {
	c.gpsProbes = planner
}

func (c *ReevaluationCoordinator) SetRuntimeSettingsReader(reader RuntimeSettingsReader) {
	c.settings = reader
}

func NewReevaluationCoordinator(repository Repository, assets AssetEvidenceResolver, policies PolicyProvider) *ReevaluationCoordinator {
	return &ReevaluationCoordinator{
		repository: repository,
		assets:     assets,
		policies:   policies,
		now:        time.Now,
	}
}

func (c *ReevaluationCoordinator) SetClock(now func() time.Time) {
	if now != nil {
		c.now = now
	}
}

func (c *ReevaluationCoordinator) Handle(ctx context.Context, request ReevaluationRequest) error {
	carrier := strings.TrimSpace(request.Carrier)
	serialNumber := strings.TrimSpace(request.SerialNumber)
	if carrier == "" {
		return ErrCarrierRequired
	}
	if serialNumber == "" {
		return ErrSerialNumberRequired
	}
	if c == nil || c.repository == nil || c.assets == nil || c.policies == nil {
		return ErrAccessGateDependencyMissing
	}
	enabled, err := runtimeAccessEnabled(ctx, c.settings, carrier)
	if err != nil {
		return fmt.Errorf("load device access business switch for reevaluation: %w", err)
	}
	if !enabled {
		return nil
	}
	asset, err := c.assets.Resolve(ctx, carrier, serialNumber)
	if err != nil {
		return fmt.Errorf("resolve reevaluation asset: %w", err)
	}
	current, err := c.repository.LoadEvaluationContext(ctx, carrier, serialNumber)
	if err != nil {
		return fmt.Errorf("load reevaluation context: %w", err)
	}
	evaluatedAt := c.now().UTC()
	policy, err := c.policies.Load(ctx, carrier)
	if err != nil {
		return fmt.Errorf("load published policy for reevaluation: %w", err)
	}
	existingState := AccessStateReviewRequired
	if current.State != nil {
		existingState = current.State.State
	}
	authenticated, authenticationRequired, _ := authenticationFromEvidence(current)
	input := EvaluationInput{
		Carrier:                carrier,
		SerialNumber:           serialNumber,
		AuthenticationRequired: authenticationRequired,
		Authenticated:          authenticated,
		AssetRetired:           asset.AssetRetired,
		ExistingState:          existingState,
		ConfirmedMismatch: request.TriggerType == "access_probe_evidence" &&
			current.State != nil && current.State.State == AccessStateRevalidating &&
			current.Evidence.Version > current.State.EvidenceVersion,
		EvaluatedAt: evaluatedAt,
		Evidence:    evidenceSetFromContext(current, asset, evaluatedAt),
		Policy:      policy,
	}
	decision := c.evaluator.Evaluate(input)
	probeNeeds, shouldProbe := accessProbeNeeds(current, decision, evaluatedAt)
	var probeRequest GPSProbeRequest
	if shouldProbe && c.gpsProbes != nil {
		productClass := strings.TrimSpace(asset.ExpectedProductClass)
		softwareVersion := strings.TrimSpace(asset.ExpectedSoftwareVersion)
		if productClass == "" {
			productClass = strings.TrimSpace(request.ObservedProductClass)
			softwareVersion = strings.TrimSpace(request.ObservedSoftwareVersion)
		}
		sourceID := ""
		if current.State != nil && current.State.CandidateID != nil {
			sourceID = current.State.CandidateID.String()
		}
		probeRequest = GPSProbeRequest{
			Carrier: carrier, SerialNumber: serialNumber,
			ProductClass:    productClass,
			SoftwareVersion: softwareVersion,
			EvidenceVersion: current.Evidence.Version + 1,
			SourceID:        sourceID,
			NeedGPS:         probeNeeds.gps, NeedTAC: probeNeeds.tac, NeedECGI: probeNeeds.ecgi,
		}
	}
	if decisionUnchanged(current, decision, policy.VersionID) {
		if shouldProbe && c.gpsProbes != nil {
			if err := c.gpsProbes.EnsureGPSProbe(ctx, probeRequest); err != nil {
				return fmt.Errorf("enqueue unchanged reevaluation evidence probe: %w", err)
			}
		}
		return nil
	}
	policyVersionID, err := policyVersionUUID(policy.VersionID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"carrier":        carrier,
		"serial_number":  serialNumber,
		"state":          decision.State,
		"reason_code":    decision.ReasonCode,
		"trigger_type":   request.TriggerType,
		"policy_version": policy.VersionID,
	})
	if err != nil {
		return fmt.Errorf("encode reevaluation event: %w", err)
	}
	triggerID := strings.TrimSpace(request.TriggerEventID)
	if triggerID == "" {
		triggerID = fmt.Sprintf(
			"reevaluation:%s:%s:%s:%s:%d",
			request.TriggerType,
			carrier,
			serialNumber,
			policy.VersionID,
			current.Evidence.Version,
		)
	}
	var candidateID *uuid.UUID
	if current.State != nil {
		candidateID = current.State.CandidateID
	}
	if _, err := c.repository.SaveDecision(ctx, DecisionChange{
		Carrier:                 carrier,
		SerialNumber:            serialNumber,
		DeviceID:                asset.DeviceID,
		CandidateID:             candidateID,
		TriggerType:             request.TriggerType,
		TriggerEventID:          triggerID,
		ExpectedDecisionVersion: decisionVersion(current),
		PolicyVersionID:         policyVersionID,
		EvidenceVersion:         current.Evidence.Version,
		OccurredAt:              evaluatedAt,
		Decision:                decision,
		Outbox: OutboxEvent{
			EventType: decisionEventType(decision.State),
			EventKey:  triggerID,
			Payload:   payload,
		},
	}); err != nil {
		return fmt.Errorf("save reevaluation decision: %w", err)
	}
	if shouldProbe && c.gpsProbes != nil {
		if err := c.gpsProbes.EnsureGPSProbe(ctx, probeRequest); err != nil {
			return fmt.Errorf("enqueue reevaluation evidence probe: %w", err)
		}
	}
	return nil
}

var _ ReevaluationHandler = (*ReevaluationCoordinator)(nil)
