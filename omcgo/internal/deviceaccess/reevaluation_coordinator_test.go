package deviceaccess

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	devicepkg "github.com/omcgo/omcgo/internal/device"
	"github.com/stretchr/testify/require"
)

type reevaluationProbeRecorder struct {
	request GPSProbeRequest
}

func (r *reevaluationProbeRecorder) EnsureGPSProbe(_ context.Context, request GPSProbeRequest) error {
	r.request = request
	return nil
}

func TestReevaluationCoordinatorPersistsChangedDecision(t *testing.T) {
	asset := AssetEvidence{Source: AssetEvidenceSourceRegistration}
	evaluationContext := trustedReevaluationContext(t, asset)
	candidateID := uuid.New()
	evaluationContext.State = &AccessStateProjection{
		CandidateID: &candidateID, State: AccessStateCollecting,
		EffectiveDecision: EffectiveActionReview, ReasonCode: ReasonEvidenceMissing,
		EvidenceVersion: evaluationContext.Evidence.Version, DecisionVersion: 1,
	}
	repository := &accessGateRepositoryStub{context: evaluationContext}
	coordinator := NewReevaluationCoordinator(repository, accessGateAssetStub{
		evidence: asset,
	}, publishedPolicyStub{policy: CompiledPolicy{Rules: []CompiledRule{{
		ID: "accept-preregistered", Enabled: true, SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "asset-rule", Type: ConditionTypeTAC, Operator: ConditionOperatorEqual,
			Expected: "100", Required: false,
		}},
	}}}})
	coordinator.SetClock(func() time.Time { return time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC) })

	err := coordinator.Handle(context.Background(), ReevaluationRequest{
		Carrier:        "cmcc",
		SerialNumber:   "SN-RECHECK",
		TriggerType:    "list_changed",
		TriggerEventID: "list-change-2",
	})

	require.NoError(t, err)
	require.NotNil(t, repository.saved)
	require.Equal(t, "list_changed", repository.saved.TriggerType)
	require.Equal(t, "list-change-2", repository.saved.TriggerEventID)
	require.Equal(t, &candidateID, repository.saved.CandidateID)
	require.Equal(t, AccessStateAccepted, repository.saved.Decision.State)
	require.Equal(t, ReasonRuleMatched, repository.saved.Decision.ReasonCode)
	require.Equal(t, event.SubjectDeviceAccessAccepted, repository.saved.Outbox.EventType)
}

func TestReevaluationCoordinatorSkipsUnchangedDecision(t *testing.T) {
	asset := AssetEvidence{Source: AssetEvidenceSourceRegistration}
	evaluationContext := trustedReevaluationContext(t, asset)
	evaluationContext.State = &AccessStateProjection{
		State: AccessStateRejected, EffectiveDecision: EffectiveActionReject,
		ReasonCode: ReasonNoApplicableRule, EvidenceVersion: evaluationContext.Evidence.Version,
	}
	repository := &accessGateRepositoryStub{context: evaluationContext}
	coordinator := NewReevaluationCoordinator(repository, accessGateAssetStub{
		evidence: asset,
	}, publishedPolicyStub{})

	err := coordinator.Handle(context.Background(), ReevaluationRequest{
		Carrier:      "cmcc",
		SerialNumber: "SN-RECHECK",
		TriggerType:  "manual",
	})

	require.NoError(t, err)
	require.Zero(t, repository.saveCount)
}

func TestReevaluationCoordinatorSkipsAllWorkWhenBusinessSwitchIsDisabled(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	coordinator := NewReevaluationCoordinator(repository, accessGateAssetStub{}, publishedPolicyStub{})
	coordinator.SetRuntimeSettingsReader(&runtimeSettingsStub{enabled: false})

	err := coordinator.Handle(context.Background(), ReevaluationRequest{
		Carrier: "cmcc", SerialNumber: "SN-OFF", TriggerType: "list_changed",
	})

	require.NoError(t, err)
	require.Zero(t, repository.saveCount)
}

func TestReevaluationCoordinatorUsesObservedProductOnlyForProbeResolution(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	asset := AssetEvidence{Source: AssetEvidenceSourceRegistration}
	evaluationContext := trustedReevaluationContext(t, asset)
	failedGPS, err := json.Marshal(map[string]string{"status": string(EvidenceStatusCollectionFailed)})
	require.NoError(t, err)
	evaluationContext.Evidence.Records = append(evaluationContext.Evidence.Records, EvidenceRecord{
		Type: ConditionTypeGPS, Status: EvidenceStatusCollectionFailed,
		NormalizedValue: failedGPS, ObservedAt: now.Add(-time.Minute),
	})
	repository := &accessGateRepositoryStub{context: evaluationContext}
	coordinator := NewReevaluationCoordinator(repository, accessGateAssetStub{evidence: asset}, publishedPolicyStub{
		policy: CompiledPolicy{VersionID: uuid.NewString(), Rules: []CompiledRule{{
			ID: "gps", Enabled: true, SerialScope: SerialScope{Type: SerialScopeAll},
			Conditions: []CompiledCondition{{
				ID: "gps", Type: ConditionTypeGPS, Operator: ConditionOperatorWithinRadius,
				GeoFence: &GeoFence{Center: GeoPoint{Latitude: 39.9042, Longitude: 116.4074}, RadiusMeters: 1000},
				Required: true,
			}},
		}}},
	})
	coordinator.SetClock(func() time.Time { return now })
	probe := &reevaluationProbeRecorder{}
	coordinator.SetGPSProbePlanner(probe)

	err = coordinator.Handle(context.Background(), ReevaluationRequest{
		Carrier: "cmcc", SerialNumber: "SN-RECHECK", TriggerType: "access_probe_evidence",
		ObservedProductClass: "SmallCell-LTE", ObservedSoftwareVersion: "FW-2.5.1",
	})

	require.NoError(t, err)
	require.Equal(t, "SmallCell-LTE", probe.request.ProductClass)
	require.Equal(t, "FW-2.5.1", probe.request.SoftwareVersion)
	// Candidate-observed identity is a translator hint only; it must not turn the
	// registration asset into a trusted expected identity.
	require.Empty(t, asset.ExpectedProductClass)
}

func trustedReevaluationContext(t *testing.T, asset AssetEvidence) EvaluationContext {
	t.Helper()
	evidence, err := buildInformEvidence(EvaluationContext{}, devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-RECHECK", OUI: "48BF74",
		Authenticated: true, CarrierIdentityResolved: true,
	}, asset, time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC), false)
	require.NoError(t, err)
	require.NotNil(t, evidence)
	return EvaluationContext{Evidence: *evidence}
}
