package deviceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	devicepkg "github.com/omcgo/omcgo/internal/device"
	"github.com/stretchr/testify/require"
)

func TestNormalizedPointAcceptsValidZeroCoordinateAndRequiresBothFields(t *testing.T) {
	point, ok := normalizedPoint(json.RawMessage(`{"latitude":0,"longitude":0}`))
	require.True(t, ok)
	require.Equal(t, GeoPoint{}, point)

	_, ok = normalizedPoint(json.RawMessage(`{"latitude":0}`))
	require.False(t, ok)

	_, ok = normalizedPoint(json.RawMessage(`{"latitude":91,"longitude":0}`))
	require.False(t, ok)
}

type accessGateProbeStub struct{ err error }

func (s accessGateProbeStub) EnsureGPSProbe(context.Context, GPSProbeRequest) error {
	return s.err
}

type accessGateRepositoryStub struct {
	observation Observation
	context     EvaluationContext
	saved       *DecisionChange
	snapshots   []IdentitySnapshot
	saveCount   int
}

func (s *accessGateRepositoryStub) LoadEvaluationContext(context.Context, string, string) (EvaluationContext, error) {
	return s.context, nil
}

func (s *accessGateRepositoryStub) SaveDecision(_ context.Context, change DecisionChange) (SavedDecision, error) {
	s.saved = &change
	s.saveCount++
	decisionID := uuid.New()
	if change.IdentitySnapshot != nil {
		snapshot := *change.IdentitySnapshot
		snapshot.DecisionID = decisionID.String()
		s.snapshots = append(s.snapshots, snapshot)
	}
	return SavedDecision{ID: decisionID}, nil
}

func (s *accessGateRepositoryStub) SaveIdentitySnapshot(_ context.Context, snapshot IdentitySnapshot) error {
	s.snapshots = append(s.snapshots, snapshot)
	return nil
}

func (s *accessGateRepositoryStub) UpsertCandidateObservation(_ context.Context, observation Observation) (Candidate, error) {
	s.observation = observation
	return Candidate{}, nil
}

func (s *accessGateRepositoryStub) AppendEvidence(context.Context, EvidenceBatch) (int64, error) {
	return 0, nil
}

type accessGateAssetStub struct {
	evidence AssetEvidence
}

func (s accessGateAssetStub) Resolve(context.Context, string, string) (AssetEvidence, error) {
	return s.evidence, nil
}

type accessGatePolicyStub struct {
	policy CompiledPolicy
}

func (s accessGatePolicyStub) Load(context.Context, string) (CompiledPolicy, error) {
	return s.policy, nil
}

func TestAccessGateUnknownAssetRequiresReview(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{
		evidence: AssetEvidence{Source: AssetEvidenceSourceUnknown},
	}, accessGatePolicyStub{})
	gate.SetClock(func() time.Time { return time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC) })

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier:                 model.CarrierCMCC,
		SerialNumber:            "SN-UNKNOWN",
		OUI:                     "48BF74",
		ProductClass:            "BaiStation",
		Authenticated:           true,
		CarrierIdentityResolved: true,
		EventID:                 "evt-unknown",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionReviewRequired, decision.State)
	require.Equal(t, string(ReasonOwnershipUnverified), decision.ReasonCode)
	require.Equal(t, "SN-UNKNOWN", repository.observation.SerialNumber)
	require.NotNil(t, repository.saved)
	require.Equal(t, "evt-unknown", repository.saved.TriggerEventID)
	require.Equal(t, event.SubjectDeviceAccessReviewRequired, repository.saved.Outbox.EventType)
}

func TestAccessGatePersistsInformIdentitySnapshotWithStableTrigger(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: AssetEvidence{
		Source: AssetEvidenceSourceDevice, ExpectedOUI: "48BF74", ExpectedProductClass: "BaiStation",
	}}, accessGatePolicyStub{policy: CompiledPolicy{ListEntries: []CompiledListEntry{{
		ID: "allow-1", Type: ListEntryTypeAllow, IdentityType: IdentityTypeSerialNumber,
		IdentityValue: "SN-SNAPSHOT", Status: ListEntryStatusActive,
	}}}})
	observedAt := time.Date(2026, 8, 19, 11, 30, 0, 0, time.UTC)
	gate.SetClock(func() time.Time { return observedAt })

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-SNAPSHOT", OUI: "48BF74", ProductClass: "BaiStation",
		RemoteIP: "192.0.2.10", Authenticated: true, AuthMethod: "digest", CredentialID: "auth-user",
		DeviceCode: "SITE-001", CloudKey: "CLOUD1",
		CarrierIdentityResolved: true, TriggerType: TriggerInformBoot, InformEvent: "1 BOOT", EventID: "request-1",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionAccepted, decision.State)
	require.Len(t, repository.snapshots, 1)
	snapshot := repository.snapshots[0]
	require.Equal(t, "request-1", snapshot.RequestID)
	require.Equal(t, "CLOUD1", snapshot.CloudKey)
	require.Equal(t, "SITE-001", snapshot.DeviceCode)
	require.Equal(t, "1 BOOT", snapshot.InformEvent)
	require.Equal(t, IdentityStatusResolved, snapshot.IdentityStatus)
	require.Equal(t, observedAt, snapshot.InformTime)
	require.Equal(t, TriggerInformBoot, repository.saved.TriggerType)
}

func TestAccessGateBypassesWithoutCreatingCandidateWhenBusinessSwitchIsDisabled(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{}, accessGatePolicyStub{})
	gate.SetRuntimeSettingsReader(&runtimeSettingsStub{enabled: false})

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-SWITCH-OFF", EventID: "evt-off",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionBypassed, decision.State)
	require.Equal(t, "access_control_disabled", decision.ReasonCode)
	require.Empty(t, repository.observation.SerialNumber)
	require.Zero(t, repository.saveCount)
}

func TestAccessGateRejectsUnauthenticatedInform(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: AssetEvidence{
		Source: AssetEvidenceSourceDevice, ExpectedOUI: "48BF74",
	}}, accessGatePolicyStub{})

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-NO-AUTH", OUI: "48BF74",
		AuthMethod: "basic", EventID: "evt-no-auth",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionRejected, decision.State)
	require.Equal(t, string(ReasonAuthenticationFailed), decision.ReasonCode)
}

func TestAccessGateDoesNotRejectWhenProtocolAuthenticationIsNotConfigured(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: AssetEvidence{
		Source: AssetEvidenceSourceDevice, ExpectedOUI: "48BF74",
	}}, accessGatePolicyStub{policy: CompiledPolicy{ListEntries: []CompiledListEntry{{
		ID: "allow-no-auth", Type: ListEntryTypeAllow, IdentityType: IdentityTypeSerialNumber,
		IdentityValue: "SN-NO-AUTH-CONFIG", Status: ListEntryStatusActive,
	}}}})

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-NO-AUTH-CONFIG", OUI: "48BF74",
		AuthMethod: "none", EventID: "evt-no-auth-config",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionAccepted, decision.State)
	require.Equal(t, string(ReasonAllowlistMatched), decision.ReasonCode)
	require.NotNil(t, repository.saved)

	_, required, status := authenticationFromEvidence(EvaluationContext{Evidence: *repository.saved.Evidence})
	require.False(t, required)
	require.Equal(t, CredentialStatusNotConfigured, status)
}

func TestAccessAuthenticationRequiredFailsClosedForMissingMethod(t *testing.T) {
	require.True(t, accessAuthenticationRequired(""))
	require.True(t, accessAuthenticationRequired("basic"))
	require.True(t, accessAuthenticationRequired("Digest"))
	require.False(t, accessAuthenticationRequired(" none "))
}

func TestAccessGateRejectsAssetIdentityMismatch(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: AssetEvidence{
		Source: AssetEvidenceSourceDevice, ExpectedOUI: "48BF74", ExpectedProductClass: "FAP-A",
	}}, accessGatePolicyStub{})

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-ID-MISMATCH", OUI: "001122",
		ProductClass: "FAP-A", Authenticated: true, EventID: "evt-id-mismatch",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionRejected, decision.State)
	require.Equal(t, string(ReasonIdentityMismatch), decision.ReasonCode)
}

func TestRequiredCarrierIdentityEvidenceCannotBeSilentlyInferredFromOUI(t *testing.T) {
	assetCode := "SITE-001"
	asset := AssetEvidence{ExpectedOUI: "48BF74", ExpectedProductClass: "BaiStation", SiteID: &assetCode}

	require.Equal(t, CheckMissing, identityCheckResult(devicepkg.AccessObservation{
		OUI: "48BF74", ProductClass: "BaiStation", CarrierIdentityResolved: true,
		DeviceCodeRequired: true,
	}, asset))
	require.Equal(t, ReasonDeviceCodeMissing, identityMissingReason(devicepkg.AccessObservation{DeviceCodeRequired: true}))
	require.Equal(t, ReasonCloudKeyMissing, identityMissingReason(devicepkg.AccessObservation{CloudKeyRequired: true}))
	require.Equal(t, CheckFailed, identityCheckResult(devicepkg.AccessObservation{
		OUI: "48BF74", ProductClass: "BaiStation", DeviceCode: "OTHER-SITE", DeviceCodeRequired: true,
	}, asset))
}

func TestAccessGateDoesNotTreatMissingProductClassAsConfirmedMismatch(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: AssetEvidence{
		Source: AssetEvidenceSourceDevice, ExpectedOUI: "48BF74", ExpectedProductClass: "FAP-A",
	}}, accessGatePolicyStub{})

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-ID-INCOMPLETE", OUI: "48BF74",
		Authenticated: true, EventID: "evt-id-incomplete",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionReviewRequired, decision.State)
	require.Equal(t, string(ReasonIdentityUnverified), decision.ReasonCode)
}

func TestAccessGateRevokesDecommissionedAsset(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: AssetEvidence{
		Source: AssetEvidenceSourceDevice, ExpectedOUI: "48BF74", AssetRetired: true,
	}}, accessGatePolicyStub{})

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-RETIRED", OUI: "48BF74",
		Authenticated: true, EventID: "evt-retired",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionRevoked, decision.State)
	require.Equal(t, string(ReasonAssetRetired), decision.ReasonCode)
}

func TestAccessGateConfirmsObservedIPMismatchWithSecondInform(t *testing.T) {
	now := time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC)
	asset := AssetEvidence{Source: AssetEvidenceSourceDevice, ExpectedOUI: "48BF74"}
	policy := CompiledPolicy{Rules: []CompiledRule{{
		ID: "source-network", Enabled: true, SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "source-ip", Type: ConditionTypeObservedIP, Operator: ConditionOperatorCIDR,
			Expected: "192.0.2.0/24", Required: true,
		}},
	}}}
	acceptedObservation := devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-IP-CONFIRM", OUI: "48BF74",
		RemoteIP: "192.0.2.10", Authenticated: true,
	}
	acceptedEvidence, err := buildInformEvidence(
		EvaluationContext{}, acceptedObservation, asset, now.Add(-time.Minute), false,
	)
	require.NoError(t, err)
	repository := &accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{
			State: AccessStateAccepted, EffectiveDecision: EffectiveActionAccept,
			ReasonCode: ReasonRuleMatched, EvidenceVersion: acceptedEvidence.Version,
		},
		Evidence: *acceptedEvidence,
	}}
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: asset}, accessGatePolicyStub{policy: policy})
	gate.SetClock(func() time.Time { return now })

	first, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-IP-CONFIRM", OUI: "48BF74",
		RemoteIP: "198.51.100.20", Authenticated: true, EventID: "evt-ip-first",
	})
	require.NoError(t, err)
	require.Equal(t, string(AccessStateRevalidating), first.State)
	require.NotNil(t, repository.saved.Evidence)

	firstChange := *repository.saved
	repository.context = EvaluationContext{
		State: &AccessStateProjection{
			State: firstChange.Decision.State, EffectiveDecision: firstChange.Decision.EffectiveAction,
			ReasonCode: firstChange.Decision.ReasonCode, EvidenceVersion: firstChange.EvidenceVersion,
			DecisionVersion: 1,
		},
		Evidence: *firstChange.Evidence,
	}
	gate.SetClock(func() time.Time { return now.Add(time.Minute) })
	second, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-IP-CONFIRM", OUI: "48BF74",
		RemoteIP: "198.51.100.20", Authenticated: true, EventID: "evt-ip-second",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionRejected, second.State)
	require.Equal(t, string(ReasonRuleMismatchConfirmed), second.ReasonCode)

	secondChange := *repository.saved
	repository.context = EvaluationContext{
		State: &AccessStateProjection{
			State: secondChange.Decision.State, EffectiveDecision: secondChange.Decision.EffectiveAction,
			ReasonCode: secondChange.Decision.ReasonCode, EvidenceVersion: secondChange.EvidenceVersion,
			DecisionVersion: 2,
		},
		Evidence: *secondChange.Evidence,
	}
	repository.saved = nil
	gate.SetClock(func() time.Time { return now.Add(2 * time.Minute) })
	third, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-IP-CONFIRM", OUI: "48BF74",
		RemoteIP: "198.51.100.20", Authenticated: true, EventID: "evt-ip-stable-reject",
	})
	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionRejected, third.State)
	require.NotNil(t, repository.saved)
	require.Nil(t, repository.saved.Evidence, "stable rejected IP evidence must not refresh on every Inform")

	thirdChange := *repository.saved
	repository.context.State = &AccessStateProjection{
		State: thirdChange.Decision.State, EffectiveDecision: thirdChange.Decision.EffectiveAction,
		ReasonCode: thirdChange.Decision.ReasonCode, EvidenceVersion: thirdChange.EvidenceVersion,
		DecisionVersion: 3,
	}
	repository.saved = nil
	gate.SetClock(func() time.Time { return now.Add(3 * time.Minute) })
	fourth, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-IP-CONFIRM", OUI: "48BF74",
		RemoteIP: "198.51.100.20", Authenticated: true, EventID: "evt-ip-stable-reject-2",
	})
	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionRejected, fourth.State)
	require.Nil(t, repository.saved, "stable rejected IP decision must not be saved on every Inform")
}

func TestAccessGateRefreshesStaleObservedIPFromCurrentInform(t *testing.T) {
	now := time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC)
	asset := AssetEvidence{Source: AssetEvidenceSourceDevice, ExpectedOUI: "48BF74"}
	policy := CompiledPolicy{Rules: []CompiledRule{{
		ID: "source-network", Enabled: true, SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "source-ip", Type: ConditionTypeObservedIP, Operator: ConditionOperatorEqual,
			Expected: "172.19.3.54", Required: true, EvidenceTTL: 15 * time.Minute,
		}},
	}}}
	staleEvidence, err := buildInformEvidence(
		EvaluationContext{}, devicepkg.AccessObservation{
			Carrier: model.CarrierCMCC, SerialNumber: "SN-IP-STALE", OUI: "48BF74",
			RemoteIP: "172.19.3.54", Authenticated: true,
		}, asset, now.Add(-time.Hour), false,
	)
	require.NoError(t, err)
	repository := &accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{
			State: AccessStateCollecting, EffectiveDecision: EffectiveActionReview,
			ReasonCode: ReasonEvidenceStale, EvidenceVersion: staleEvidence.Version,
		},
		Evidence: *staleEvidence,
	}}
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: asset}, accessGatePolicyStub{policy: policy})
	gate.SetClock(func() time.Time { return now })

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-IP-STALE", OUI: "48BF74",
		RemoteIP: "172.19.3.54", Authenticated: true, EventID: "evt-ip-refresh",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionAccepted, decision.State)
	require.NotNil(t, repository.saved.Evidence)
	refreshed := evidenceRecordByType(t, *repository.saved.Evidence, ConditionTypeObservedIP)
	require.Equal(t, now, refreshed.ObservedAt)
}

func TestAccessProbeCollectionFailureUsesBoundedRetryBackoff(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	context := EvaluationContext{
		State: &AccessStateProjection{State: AccessStateCollecting, EvidenceVersion: 2},
		Evidence: EvidenceBatch{Version: 2, Records: []EvidenceRecord{{
			Type: ConditionTypeTAC, Status: EvidenceStatusCollectionFailed, ObservedAt: now,
		}},
		},
	}
	decision := Decision{
		State: AccessStateCollecting, ReasonCode: ReasonEvidenceCollectionFailed,
		Checks: []DecisionCheck{{
			CheckType: ConditionTypeTAC, Result: CheckError, ReasonCode: ReasonEvidenceCollectionFailed,
		}},
	}

	_, shouldRetry := accessProbeNeeds(context, decision, now.Add(accessProbeRetryBackoff-time.Second))
	require.False(t, shouldRetry)
	requirement, shouldRetry := accessProbeNeeds(context, decision, now.Add(accessProbeRetryBackoff))
	require.True(t, shouldRetry)
	require.True(t, requirement.tac)
}

func TestAccessProbeMissingEvidenceRequeuesDurableIntent(t *testing.T) {
	context := EvaluationContext{
		State:    &AccessStateProjection{State: AccessStateCollecting, EvidenceVersion: 2},
		Evidence: EvidenceBatch{Version: 2},
	}
	decision := Decision{
		State: AccessStateCollecting, ReasonCode: ReasonEvidenceMissing,
		Checks: []DecisionCheck{{CheckType: ConditionTypeGPS, Result: CheckMissing}},
	}

	requirement, shouldProbe := accessProbeNeeds(context, decision, time.Now())

	require.True(t, shouldProbe)
	require.True(t, requirement.gps)
}

func TestAccessGatePreRegisteredAssetStillRequiresApplicablePolicy(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{
		evidence: AssetEvidence{
			Source:         AssetEvidenceSourceRegistration,
			RegistrationID: ptrUUID(uuid.New()),
		},
	}, accessGatePolicyStub{policy: preregisteredPolicy()})

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier:                 model.CarrierCMCC,
		SerialNumber:            "SN-REGISTERED",
		OUI:                     "48BF74",
		Authenticated:           true,
		CarrierIdentityResolved: true,
		EventID:                 "evt-registered",
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionAccepted, decision.State)
	require.Equal(t, string(ReasonRuleMatched), decision.ReasonCode)
	require.Equal(t, event.SubjectDeviceAccessAccepted, repository.saved.Outbox.EventType)
}

func TestAccessGateRequiresAllDependencies(t *testing.T) {
	gate := NewAccessGate(nil, nil, nil)

	_, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier:      model.CarrierCMCC,
		SerialNumber: "SN-MISSING-DEPS",
	})

	require.ErrorIs(t, err, ErrAccessGateDependencyMissing)
}

func TestAccessGatePersistsCollectingDecisionBeforeProbePlanningFailure(t *testing.T) {
	repository := &accessGateRepositoryStub{}
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: AssetEvidence{
		Source: AssetEvidenceSourceDevice, ExpectedOUI: "48BF74",
	}}, accessGatePolicyStub{policy: CompiledPolicy{Rules: []CompiledRule{{
		ID: "tac-rule", Enabled: true, SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "required-tac", Type: ConditionTypeTAC,
			Operator: ConditionOperatorEqual, Expected: "100", Required: true,
		}},
	}}}})
	gate.SetGPSProbePlanner(accessGateProbeStub{err: errors.New("path unavailable")})

	_, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-PROBE-FAIL", OUI: "48BF74",
		Authenticated: true, EventID: "evt-probe-fail",
	})

	require.ErrorContains(t, err, "ensure access evidence probe")
	require.NotNil(t, repository.saved)
	require.Equal(t, AccessStateCollecting, repository.saved.Decision.State)
	require.Equal(t, ReasonEvidenceMissing, repository.saved.Decision.ReasonCode)
	require.NotNil(t, repository.saved.Evidence)
}

func TestAccessGateDoesNotAppendUnchangedDecision(t *testing.T) {
	policyVersion := uuid.New()
	asset := AssetEvidence{Source: AssetEvidenceSourceRegistration}
	observation := devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-UNCHANGED", OUI: "48BF74",
		Authenticated: true, CarrierIdentityResolved: true, EventID: "evt-unchanged",
	}
	evidence, err := buildInformEvidence(EvaluationContext{}, observation, asset, time.Now().UTC(), false)
	require.NoError(t, err)
	require.NotNil(t, evidence)
	repository := &accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{
			State:             AccessStateAccepted,
			EffectiveDecision: EffectiveActionAccept,
			ReasonCode:        ReasonRuleMatched,
			PolicyVersionID:   &policyVersion,
			EvidenceVersion:   evidence.Version,
		},
		Evidence: *evidence,
	}}
	gate := NewAccessGate(repository, accessGateAssetStub{
		evidence: asset,
	}, accessGatePolicyStub{policy: func() CompiledPolicy {
		policy := preregisteredPolicy()
		policy.VersionID = policyVersion.String()
		return policy
	}()})

	decision, err := gate.Admit(context.Background(), observation)

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionAccepted, decision.State)
	require.Zero(t, repository.saveCount)
}

func TestAccessGateLinksFormalDeviceWhenAcceptedProjectionIsStillCandidateOwned(t *testing.T) {
	policyVersion := uuid.New()
	deviceID := uuid.New()
	asset := AssetEvidence{Source: AssetEvidenceSourceDevice, DeviceID: &deviceID}
	observation := devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-OWNER-SYNC", OUI: "48BF74",
		Authenticated: true, CarrierIdentityResolved: true, EventID: "evt-owner-sync",
	}
	evidence, err := buildInformEvidence(EvaluationContext{}, observation, asset, time.Now().UTC(), false)
	require.NoError(t, err)
	require.NotNil(t, evidence)
	repository := &accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{
			State: AccessStateAccepted, EffectiveDecision: EffectiveActionAccept,
			ReasonCode: ReasonRuleMatched, PolicyVersionID: &policyVersion,
			EvidenceVersion: evidence.Version,
		},
		Evidence: *evidence,
	}}
	policy := preregisteredPolicy()
	policy.VersionID = policyVersion.String()
	gate := NewAccessGate(repository, accessGateAssetStub{evidence: asset}, accessGatePolicyStub{policy: policy})

	decision, err := gate.Admit(context.Background(), observation)

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionAccepted, decision.State)
	require.Equal(t, 1, repository.saveCount)
	require.Equal(t, &deviceID, repository.saved.DeviceID)
}

func ptrUUID(value uuid.UUID) *uuid.UUID {
	return &value
}

func preregisteredPolicy() CompiledPolicy {
	return CompiledPolicy{Rules: []CompiledRule{{
		ID:          "preregistered-assets",
		Enabled:     true,
		SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID:       "optional-tac",
			Type:     ConditionTypeTAC,
			Operator: ConditionOperatorEqual,
			Expected: "100",
			Required: false,
		}},
	}}}
}
