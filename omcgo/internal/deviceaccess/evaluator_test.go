package deviceaccess

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluatorPriorityOrder(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	base := EvaluationInput{
		Carrier:                "cmcc",
		SerialNumber:           "SN-001",
		AuthenticationRequired: true,
		Authenticated:          true,
		EvaluatedAt:            now,
		Evidence: EvidenceSet{
			Identity:  CheckPassed,
			Ownership: CheckPassed,
			Values: map[ConditionType]EvidenceValue{
				ConditionTypeTAC: availableTextEvidence("100", now),
			},
		},
		Policy: CompiledPolicy{
			VersionID: "policy-v1",
			ListEntries: []CompiledListEntry{{
				ID:            "allow-sn",
				Type:          ListEntryTypeAllow,
				IdentityType:  IdentityTypeSerialNumber,
				IdentityValue: "SN-001",
				Status:        ListEntryStatusActive,
			}},
			Rules: []CompiledRule{{
				ID:      "rule-tac",
				Enabled: true,
				SerialScope: SerialScope{
					Type: SerialScopeAll,
				},
				Conditions: []CompiledCondition{{
					ID:       "tac",
					Type:     ConditionTypeTAC,
					Operator: ConditionOperatorEqual,
					Expected: "100",
					Required: true,
				}},
			}},
		},
	}

	tests := []struct {
		name       string
		mutate     func(*EvaluationInput)
		wantState  AccessState
		wantAction EffectiveAction
		wantReason ReasonCode
	}{
		{
			name: "protocol authentication failure wins over allowlist",
			mutate: func(in *EvaluationInput) {
				in.Authenticated = false
			},
			wantState:  AccessStateRejected,
			wantAction: EffectiveActionReject,
			wantReason: ReasonAuthenticationFailed,
		},
		{
			name: "disabled protocol authentication does not fabricate a failure",
			mutate: func(in *EvaluationInput) {
				in.AuthenticationRequired = false
				in.Authenticated = false
			},
			wantState:  AccessStateAccepted,
			wantAction: EffectiveActionAccept,
			wantReason: ReasonAllowlistMatched,
		},
		{
			name: "retired asset wins over allowlist",
			mutate: func(in *EvaluationInput) {
				in.AssetRetired = true
			},
			wantState:  AccessStateRevoked,
			wantAction: EffectiveActionRevoke,
			wantReason: ReasonAssetRetired,
		},
		{
			name: "denylist wins when same identity is allowlisted",
			mutate: func(in *EvaluationInput) {
				in.Policy.ListEntries = append(in.Policy.ListEntries, CompiledListEntry{
					ID:            "deny-sn",
					Type:          ListEntryTypeDeny,
					IdentityType:  IdentityTypeSerialNumber,
					IdentityValue: "SN-001",
					Status:        ListEntryStatusActive,
				})
			},
			wantState:  AccessStateRejected,
			wantAction: EffectiveActionReject,
			wantReason: ReasonDenylistMatched,
		},
		{
			name: "strong identity mismatch wins over allowlist",
			mutate: func(in *EvaluationInput) {
				in.Evidence.Identity = CheckFailed
			},
			wantState:  AccessStateRejected,
			wantAction: EffectiveActionReject,
			wantReason: ReasonIdentityMismatch,
		},
		{
			name: "asset ownership mismatch wins over allowlist",
			mutate: func(in *EvaluationInput) {
				in.Evidence.Ownership = CheckFailed
			},
			wantState:  AccessStateRejected,
			wantAction: EffectiveActionReject,
			wantReason: ReasonOwnershipMismatch,
		},
		{
			name: "allowlist skips an ordinary planning mismatch",
			mutate: func(in *EvaluationInput) {
				in.Evidence.Values[ConditionTypeTAC] = availableTextEvidence("999", now)
			},
			wantState:  AccessStateAccepted,
			wantAction: EffectiveActionAccept,
			wantReason: ReasonAllowlistMatched,
		},
		{
			name: "preregistration remains subject to ordinary planning rules",
			mutate: func(in *EvaluationInput) {
				in.Policy.ListEntries = nil
				in.Evidence.Values[ConditionTypeTAC] = availableTextEvidence("999", now)
			},
			wantState:  AccessStateRejected,
			wantAction: EffectiveActionReject,
			wantReason: ReasonRuleMismatch,
		},
		{
			name: "ordinary rule accepts after all higher priorities pass",
			mutate: func(in *EvaluationInput) {
				in.Policy.ListEntries = nil
			},
			wantState:  AccessStateAccepted,
			wantAction: EffectiveActionAccept,
			wantReason: ReasonRuleMatched,
		},
		{
			name: "device without list or applicable rule is rejected",
			mutate: func(in *EvaluationInput) {
				in.Policy.ListEntries = nil
				in.Policy.Rules = nil
			},
			wantState:  AccessStateRejected,
			wantAction: EffectiveActionReject,
			wantReason: ReasonNoApplicableRule,
		},
	}

	evaluator := Evaluator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := cloneEvaluationInput(base)
			tt.mutate(&in)

			got := evaluator.Evaluate(in)

			assert.Equal(t, tt.wantState, got.State)
			assert.Equal(t, tt.wantAction, got.EffectiveAction)
			assert.Equal(t, tt.wantReason, got.ReasonCode)
		})
	}
}

func TestEvaluatorRulesUseANDWithinRuleAndORAcrossRules(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.SerialNumber = "SN-105"
	input.Evidence.Values = map[ConditionType]EvidenceValue{
		ConditionTypeTAC:        availableTextEvidence("100", now),
		ConditionTypeECGI:       availableTextEvidence("460-00-12345", now),
		ConditionTypeObservedIP: availableTextEvidence("10.21.7.9", now),
		ConditionTypeGPS: {
			Status:     EvidenceStatusAvailable,
			Point:      &GeoPoint{Latitude: 39.9042, Longitude: 116.4074},
			Source:     "gpv",
			ObservedAt: now,
		},
	}
	input.Policy.Rules = []CompiledRule{
		{
			ID:      "rule-and-fails",
			Enabled: true,
			SerialScope: SerialScope{
				Type:   SerialScopePrefix,
				Prefix: "SN-",
			},
			Conditions: []CompiledCondition{
				{ID: "tac-equals", Type: ConditionTypeTAC, Operator: ConditionOperatorEqual, Expected: "100", Required: true},
				{ID: "ip-wrong-cidr", Type: ConditionTypeObservedIP, Operator: ConditionOperatorCIDR, Expected: "192.0.2.0/24", Required: true},
			},
		},
		{
			ID:      "rule-or-passes",
			Enabled: true,
			SerialScope: SerialScope{
				Type:  SerialScopeRange,
				Start: "SN-100",
				End:   "SN-199",
			},
			Conditions: []CompiledCondition{
				{ID: "ecgi-in", Type: ConditionTypeECGI, Operator: ConditionOperatorIn, ExpectedAny: []string{"460-00-12344", "460-00-12345"}, Required: true},
				{
					ID:       "gps-radius",
					Type:     ConditionTypeGPS,
					Operator: ConditionOperatorWithinRadius,
					GeoFence: &GeoFence{Center: GeoPoint{Latitude: 39.9042, Longitude: 116.4074}, RadiusMeters: 100},
					Required: true,
				},
			},
		},
	}

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateAccepted, got.State)
	assert.Equal(t, ReasonRuleMatched, got.ReasonCode)
	assert.Equal(t, "rule-or-passes", got.MatchedRuleID)
	require.Len(t, got.Checks, 4)
	assert.Equal(t, CheckFailed, got.Checks[1].Result)
	assert.Equal(t, CheckPassed, got.Checks[2].Result)
	assert.Equal(t, CheckPassed, got.Checks[3].Result)
}

func TestEvaluatorRequiresEveryServingCellValueInPlannedSet(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.Evidence.Values[ConditionTypeTAC] = EvidenceValue{
		Status: EvidenceStatusAvailable, Texts: []string{"100", "999"}, Source: "gpv", ObservedAt: now,
	}
	input.Policy.Rules = []CompiledRule{{
		ID: "multi-cell-tac", Enabled: true, SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "tac-in", Type: ConditionTypeTAC, Operator: ConditionOperatorIn,
			ExpectedAny: []string{"100", "101"}, Required: true,
		}},
	}}

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateRejected, got.State)
	require.Len(t, got.Checks, 1)
	assert.Equal(t, CheckFailed, got.Checks[0].Result)
	assert.Equal(t, "100,999", got.Checks[0].ObservedSummary)
}

func TestEvaluatorDoesNotTreatMissingStaleOrCollectionFailureAsPass(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		evidence   EvidenceValue
		wantState  AccessState
		wantReason ReasonCode
		wantCheck  CheckResult
	}{
		{
			name:       "missing evidence enters collection",
			evidence:   EvidenceValue{Status: EvidenceStatusMissing},
			wantState:  AccessStateCollecting,
			wantReason: ReasonEvidenceMissing,
			wantCheck:  CheckMissing,
		},
		{
			name: "stale evidence enters collection",
			evidence: EvidenceValue{
				Status:     EvidenceStatusAvailable,
				Text:       "100",
				Source:     "gpv",
				ObservedAt: now.Add(-2 * time.Hour),
			},
			wantState:  AccessStateCollecting,
			wantReason: ReasonEvidenceStale,
			wantCheck:  CheckStale,
		},
		{
			name:       "collection failure enters collection",
			evidence:   EvidenceValue{Status: EvidenceStatusCollectionFailed, Source: "gpv"},
			wantState:  AccessStateCollecting,
			wantReason: ReasonEvidenceCollectionFailed,
			wantCheck:  CheckError,
		},
		{
			name:       "system error requires review",
			evidence:   EvidenceValue{Status: EvidenceStatusSystemError, Source: "repository"},
			wantState:  AccessStateReviewRequired,
			wantReason: ReasonEvidenceSystemError,
			wantCheck:  CheckError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validInput(now)
			input.Evidence.Values[ConditionTypeTAC] = tt.evidence
			input.Policy.Rules = []CompiledRule{{
				ID:          "rule-tac",
				Enabled:     true,
				SerialScope: SerialScope{Type: SerialScopeAll},
				Conditions: []CompiledCondition{{
					ID:          "tac",
					Type:        ConditionTypeTAC,
					Operator:    ConditionOperatorEqual,
					Expected:    "100",
					Required:    true,
					EvidenceTTL: time.Hour,
				}},
			}}

			got := (Evaluator{}).Evaluate(input)

			assert.Equal(t, tt.wantState, got.State)
			assert.Equal(t, tt.wantReason, got.ReasonCode)
			require.Len(t, got.Checks, 1)
			assert.Equal(t, tt.wantCheck, got.Checks[0].Result)
		})
	}
}

func TestEvaluatorSkipsOptionalMissingCondition(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.Evidence.Values[ConditionTypeTAC] = availableTextEvidence("100", now)
	input.Policy.Rules = []CompiledRule{{
		ID:          "rule-optional",
		Enabled:     true,
		SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{
			{ID: "tac", Type: ConditionTypeTAC, Operator: ConditionOperatorEqual, Expected: "100", Required: true},
			{ID: "gps", Type: ConditionTypeGPS, Operator: ConditionOperatorWithinRadius, Required: false},
		},
	}}

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateAccepted, got.State)
	require.Len(t, got.Checks, 2)
	assert.Equal(t, CheckSkipped, got.Checks[1].Result)
}

func TestEvaluatorAllowsExplicitlyMissingGPSCoordinates(t *testing.T) {
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	delete(input.Evidence.Values, ConditionTypeGPS)
	input.Policy.Rules = []CompiledRule{{
		ID:          "gps-missing-allowed",
		Enabled:     true,
		SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID:       "gps",
			Type:     ConditionTypeGPS,
			Operator: ConditionOperatorWithinRadius,
			GeoFence: &GeoFence{
				Center: GeoPoint{Latitude: 30, Longitude: 104}, RadiusMeters: 100,
				AllowMissing: true,
			},
			Required: true,
		}},
	}}

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateAccepted, got.State)
	require.Len(t, got.Checks, 1)
	assert.Equal(t, CheckSkipped, got.Checks[0].Result)
	assert.Equal(t, "geofence;allow_missing=true", got.Checks[0].ExpectedSummary)
}

func TestEvaluatorDoesNotTreatInvalidGPSAsMissing(t *testing.T) {
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.Evidence.Values[ConditionTypeGPS] = EvidenceValue{Status: EvidenceStatusCollectionFailed, ObservedAt: now}
	input.Policy.Rules = []CompiledRule{{
		ID:          "gps-invalid",
		Enabled:     true,
		SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "gps", Type: ConditionTypeGPS, Operator: ConditionOperatorWithinRadius,
			GeoFence: &GeoFence{Center: GeoPoint{Latitude: 30, Longitude: 104}, RadiusMeters: 100, AllowMissing: true},
			Required: true,
		}},
	}}

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateCollecting, got.State)
	require.Len(t, got.Checks, 1)
	assert.Equal(t, CheckError, got.Checks[0].Result)
}

func TestEvaluatorExistingAcceptedMismatchRevalidatesBeforeRejecting(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.ExistingState = AccessStateAccepted
	input.Evidence.Values[ConditionTypeTAC] = availableTextEvidence("999", now)
	input.Policy.Rules = []CompiledRule{{
		ID:          "rule-tac",
		Enabled:     true,
		SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "tac", Type: ConditionTypeTAC, Operator: ConditionOperatorEqual, Expected: "100", Required: true,
		}},
	}}

	first := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateRevalidating, first.State)
	assert.Equal(t, EffectiveActionReview, first.EffectiveAction)
	assert.Equal(t, ReasonRuleMismatchPendingConfirmation, first.ReasonCode)
	assert.True(t, first.FreezeNormal)

	input.ConfirmedMismatch = true
	confirmed := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateRejected, confirmed.State)
	assert.Equal(t, EffectiveActionReject, confirmed.EffectiveAction)
	assert.Equal(t, ReasonRuleMismatchConfirmed, confirmed.ReasonCode)
	assert.True(t, confirmed.FreezeNormal)
}

func TestEvaluatorSupportsSerialListScope(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.Policy.Rules = []CompiledRule{{
		ID:      "serial-list-rule",
		Enabled: true,
		SerialScope: SerialScope{
			Type:   SerialScopeList,
			Values: []string{"SN-OTHER", input.SerialNumber},
		},
		Conditions: []CompiledCondition{{
			ID: "tac", Type: ConditionTypeTAC, Operator: ConditionOperatorEqual, Expected: "100", Required: true,
		}},
	}}

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateAccepted, got.State)
	assert.Equal(t, "serial-list-rule", got.MatchedRuleID)
}

func TestEvaluatorChecksCIDRMembership(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.Evidence.Values[ConditionTypeObservedIP] = availableTextEvidence("10.21.7.9", now)
	input.Policy.Rules = []CompiledRule{{
		ID:          "ip-rule",
		Enabled:     true,
		SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "ip", Type: ConditionTypeObservedIP, Operator: ConditionOperatorCIDR, Expected: "10.21.0.0/16", Required: true,
		}},
	}}

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateAccepted, got.State)
	assert.Equal(t, "ip-rule", got.MatchedRuleID)
}

func TestEvaluatorRejectsPointOutsideGPSRadius(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.Evidence.Values[ConditionTypeGPS] = EvidenceValue{
		Status:     EvidenceStatusAvailable,
		Point:      &GeoPoint{Latitude: 31.2304, Longitude: 121.4737},
		Source:     "gpv",
		ObservedAt: now,
	}
	input.Policy.Rules = []CompiledRule{{
		ID:          "beijing-rule",
		Enabled:     true,
		SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID:       "gps",
			Type:     ConditionTypeGPS,
			Operator: ConditionOperatorWithinRadius,
			GeoFence: &GeoFence{Center: GeoPoint{Latitude: 39.9042, Longitude: 116.4074}, RadiusMeters: 1000},
			Required: true,
		}},
	}}

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateRejected, got.State)
	assert.Equal(t, ReasonRuleMismatch, got.ReasonCode)
	require.Len(t, got.Checks, 1)
	assert.Equal(t, CheckFailed, got.Checks[0].Result)
}

func TestEvaluatorIgnoresInactiveAndExpiredListEntries(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.Policy.Rules = nil
	expiredAt := now.Add(-time.Minute)
	input.Policy.ListEntries = []CompiledListEntry{
		{
			ID:            "expired-deny",
			Type:          ListEntryTypeDeny,
			IdentityType:  IdentityTypeSerialNumber,
			IdentityValue: input.SerialNumber,
			Status:        ListEntryStatusActive,
			ValidUntil:    &expiredAt,
		},
		{
			ID:            "disabled-allow",
			Type:          ListEntryTypeAllow,
			IdentityType:  IdentityTypeSerialNumber,
			IdentityValue: input.SerialNumber,
			Status:        ListEntryStatusDisabled,
		},
	}

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateRejected, got.State)
	assert.Equal(t, ReasonNoApplicableRule, got.ReasonCode)
}

func TestEvaluatorUsesVersionedRejectDefaultWhenNoRuleApplies(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)
	input := validInput(now)
	input.Policy.Rules = nil
	input.Policy.DefaultAction = PolicyDefaultActionReject

	got := (Evaluator{}).Evaluate(input)

	assert.Equal(t, AccessStateRejected, got.State)
	assert.Equal(t, EffectiveActionReject, got.EffectiveAction)
	assert.Equal(t, ReasonNoApplicableRule, got.ReasonCode)
	assert.True(t, got.FreezeNormal)
}

func TestEvaluatorFreezesNormalTasksUnlessDecisionAccepts(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC)

	reviewInput := validInput(now)
	reviewInput.Policy.Rules = nil

	collectingInput := validInput(now)
	collectingInput.Evidence.Values = map[ConditionType]EvidenceValue{}
	collectingInput.Policy.Rules = []CompiledRule{{
		ID:          "needs-tac",
		Enabled:     true,
		SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "tac", Type: ConditionTypeTAC, Operator: ConditionOperatorEqual, Expected: "100", Required: true,
		}},
	}}

	rejectedInput := validInput(now)
	rejectedInput.Evidence.Identity = CheckFailed

	revokedInput := validInput(now)
	revokedInput.AssetRetired = true

	acceptedInput := validInput(now)
	acceptedInput.Policy.Rules = []CompiledRule{{
		ID:          "accepted-rule",
		Enabled:     true,
		SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "accepted-tac", Type: ConditionTypeTAC, Operator: ConditionOperatorEqual,
			Expected: "100", Required: true,
		}},
	}}

	tests := []struct {
		name       string
		input      EvaluationInput
		wantFrozen bool
	}{
		{name: "review required", input: reviewInput, wantFrozen: true},
		{name: "collecting evidence", input: collectingInput, wantFrozen: true},
		{name: "rejected", input: rejectedInput, wantFrozen: true},
		{name: "revoked", input: revokedInput, wantFrozen: true},
		{name: "accepted", input: acceptedInput, wantFrozen: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (Evaluator{}).Evaluate(tt.input)
			assert.Equal(t, tt.wantFrozen, got.FreezeNormal)
		})
	}
}

func validInput(now time.Time) EvaluationInput {
	return EvaluationInput{
		Carrier:                "cmcc",
		SerialNumber:           "SN-001",
		AuthenticationRequired: true,
		Authenticated:          true,
		EvaluatedAt:            now,
		Evidence: EvidenceSet{
			Identity:  CheckPassed,
			Ownership: CheckPassed,
			Values: map[ConditionType]EvidenceValue{
				ConditionTypeTAC: availableTextEvidence("100", now),
			},
		},
		Policy: CompiledPolicy{VersionID: "policy-v1"},
	}
}

func availableTextEvidence(value string, observedAt time.Time) EvidenceValue {
	return EvidenceValue{
		Status:     EvidenceStatusAvailable,
		Text:       value,
		Source:     "inform",
		ObservedAt: observedAt,
	}
}

func cloneEvaluationInput(in EvaluationInput) EvaluationInput {
	out := in
	out.Evidence.Values = make(map[ConditionType]EvidenceValue, len(in.Evidence.Values))
	for key, value := range in.Evidence.Values {
		out.Evidence.Values[key] = value
	}
	out.Policy.ListEntries = append([]CompiledListEntry(nil), in.Policy.ListEntries...)
	out.Policy.Rules = append([]CompiledRule(nil), in.Policy.Rules...)
	return out
}
