package deviceaccess

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type policyStoreStub struct {
	versions  map[string]PolicyVersion
	created   PolicyVersion
	published string
	deleted   string
}

type policyHistoryStoreStub struct {
	*policyStoreStub
	previous PolicyVersion
}

func (s *policyHistoryStoreStub) PreviousVersion(context.Context, string) (PolicyVersion, error) {
	return s.previous, nil
}

func (s *policyStoreStub) UpdateDraft(_ context.Context, version PolicyVersion) (PolicyVersion, error) {
	s.versions[version.ID] = version
	return version, nil
}

func (s *policyStoreStub) CreateDraft(_ context.Context, version PolicyVersion) (PolicyVersion, error) {
	if s.versions == nil {
		s.versions = map[string]PolicyVersion{}
	}
	version.ID = "draft-1"
	version.Version = 1
	s.created = version
	s.versions[version.ID] = version
	return version, nil
}

func (s *policyStoreStub) GetVersion(_ context.Context, versionID string) (PolicyVersion, error) {
	return s.versions[versionID], nil
}

func (s *policyStoreStub) Publish(_ context.Context, versionID, subjectID string) (PolicyVersion, error) {
	version := s.versions[versionID]
	version.Status = PolicyVersionPublished
	version.PublishedBy = subjectID
	s.published = versionID
	s.versions[versionID] = version
	return version, nil
}

func (s *policyStoreStub) DeleteDraft(_ context.Context, versionID string) error {
	s.deleted = versionID
	delete(s.versions, versionID)
	return nil
}

type reevaluationQueueStub struct {
	requests []ReevaluationRequest
}

func (q *reevaluationQueueStub) Enqueue(_ context.Context, request ReevaluationRequest) error {
	q.requests = append(q.requests, request)
	return nil
}

type publishedPolicyStub struct {
	policy CompiledPolicy
}

func (s publishedPolicyStub) Load(_ context.Context, _ string) (CompiledPolicy, error) {
	return s.policy, nil
}

func policyServiceRule() CompiledRule {
	return CompiledRule{
		ID:      "rule-1",
		Enabled: true,
		SerialScope: SerialScope{
			Type: SerialScopeAll,
		},
		Conditions: []CompiledCondition{{
			ID:       "tac-1",
			Type:     ConditionTypeTAC,
			Operator: ConditionOperatorEqual,
			Expected: "100",
			Required: true,
		}},
	}
}

func TestNormalizeCompiledPolicyAcceptsHostPrefixForObservedIP(t *testing.T) {
	policy := normalizeCompiledPolicy(CompiledPolicy{Rules: []CompiledRule{{
		ID: "ip-rule", Enabled: true, SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{
			{ID: "ipv4", Type: ConditionTypeObservedIP, Operator: ConditionOperatorEqual, Expected: "192.0.2.10/32"},
			{ID: "ipv6", Type: ConditionTypeObservedIP, Operator: ConditionOperatorIn, ExpectedAny: []string{"2001:db8::1/128"}},
			{ID: "network", Type: ConditionTypeObservedIP, Operator: ConditionOperatorEqual, Expected: "192.0.2.0/24"},
		},
	}}})

	require.Equal(t, "192.0.2.10", policy.Rules[0].Conditions[0].Expected)
	require.Equal(t, []string{"2001:db8::1"}, policy.Rules[0].Conditions[1].ExpectedAny)
	require.Equal(t, "192.0.2.0/24", policy.Rules[0].Conditions[2].Expected)
	require.NoError(t, validatePolicyCondition("ip-rule", policy.Rules[0].Conditions[0]))
	require.NoError(t, validatePolicyCondition("ip-rule", policy.Rules[0].Conditions[1]))
	require.ErrorContains(t, validatePolicyCondition("ip-rule", policy.Rules[0].Conditions[2]), "invalid IP address")
}

func TestPolicyServiceCreatesDraftWithStableContentHash(t *testing.T) {
	store := &policyStoreStub{}
	service := NewPolicyService(store, nil)

	version, err := service.CreateDraft(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: "operator-1",
	}, "production", CompiledPolicy{Rules: []CompiledRule{policyServiceRule()}})

	require.NoError(t, err)
	require.Equal(t, PolicyVersionDraft, version.Status)
	require.NotEmpty(t, version.ContentHash)
	require.Equal(t, version.ContentHash, store.created.ContentHash)
	require.Equal(t, PolicyDefaultActionReject, version.Policy.DefaultAction)
	require.Equal(t, "rule-1", version.Policy.Rules[0].Name)
	require.Equal(t, 100, version.Policy.Rules[0].Priority)
	require.NotEqual(t, "rule-1", version.Policy.Rules[0].ID)
	require.NotEqual(t, "tac-1", version.Policy.Rules[0].Conditions[0].ID)
	require.NotEqual(t, version.Policy.Rules[0].ID, store.created.Policy.Rules[0].Conditions[0].ID)
	_, ruleIDErr := uuid.Parse(version.Policy.Rules[0].ID)
	require.NoError(t, ruleIDErr)
	_, conditionIDErr := uuid.Parse(version.Policy.Rules[0].Conditions[0].ID)
	require.NoError(t, conditionIDErr)
}

func TestPolicyServiceGetVersionEnforcesCarrierScope(t *testing.T) {
	store := &policyStoreStub{versions: map[string]PolicyVersion{
		"v1": {ID: "v1", Carrier: "cmcc", Policy: CompiledPolicy{DefaultAction: PolicyDefaultActionReject}},
	}}
	service := NewPolicyService(store, nil)

	version, err := service.GetVersion(context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, "v1")
	require.NoError(t, err)
	require.Equal(t, PolicyDefaultActionReject, version.Policy.DefaultAction)

	_, err = service.GetVersion(context.Background(), PolicyActor{Carrier: "ctcc", SubjectID: "u2"}, "v1")
	require.ErrorIs(t, err, ErrPolicyCarrierScope)
}

func TestPolicyServiceUpdatePreservesOwnedIDsAndRegeneratesForeignIDs(t *testing.T) {
	existingRuleID := uuid.NewString()
	existingConditionID := uuid.NewString()
	foreignRuleID := uuid.NewString()
	foreignConditionID := uuid.NewString()
	store := &policyStoreStub{versions: map[string]PolicyVersion{
		"draft": {
			ID: "draft", Carrier: "cmcc", Status: PolicyVersionDraft,
			Policy: CompiledPolicy{DefaultAction: PolicyDefaultActionReject, FailureMode: FailureModeFailClosed, CollectionTimeout: 15 * time.Minute, Rules: []CompiledRule{{
				ID: existingRuleID, Name: "existing", Enabled: true, Priority: 100, SerialScope: SerialScope{Type: SerialScopeAll},
				Conditions: []CompiledCondition{{ID: existingConditionID, Type: ConditionTypeTAC, Operator: ConditionOperatorEqual, Expected: "100", Required: true}},
			}}},
		},
	}}
	service := NewPolicyService(store, nil)
	incoming := store.versions["draft"].Policy
	incoming.Rules = append([]CompiledRule(nil), incoming.Rules...)
	incoming.Rules[0].Conditions = append([]CompiledCondition(nil), incoming.Rules[0].Conditions...)
	incoming.Rules[0].Conditions = append(incoming.Rules[0].Conditions, CompiledCondition{
		ID: foreignConditionID, Type: ConditionTypeECGI, Operator: ConditionOperatorEqual, Expected: "00101", Required: true,
	})
	incoming.Rules = append(incoming.Rules, CompiledRule{
		ID: foreignRuleID, Name: "new", Enabled: true, Priority: 200, SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{ID: uuid.NewString(), Type: ConditionTypeTAC, Operator: ConditionOperatorEqual, Expected: "200", Required: true}},
	})

	updated, err := service.UpdateDraft(context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, "draft", incoming)

	require.NoError(t, err)
	require.Equal(t, existingRuleID, updated.Policy.Rules[0].ID)
	require.Equal(t, existingConditionID, updated.Policy.Rules[0].Conditions[0].ID)
	require.NotEqual(t, foreignConditionID, updated.Policy.Rules[0].Conditions[1].ID)
	require.NotEqual(t, foreignRuleID, updated.Policy.Rules[1].ID)
}

func TestPolicyServicePublishEnforcesCarrierScopeAndImmutability(t *testing.T) {
	store := &policyStoreStub{}
	service := NewPolicyService(store, nil)
	version, err := service.CreateDraft(context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, "p", CompiledPolicy{Rules: []CompiledRule{policyServiceRule()}})
	require.NoError(t, err)

	_, err = service.Publish(context.Background(), PolicyActor{Carrier: "ctcc", SubjectID: "u2"}, version.ID)
	require.ErrorIs(t, err, ErrPolicyCarrierScope)

	published, err := service.Publish(context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, version.ID)
	require.NoError(t, err)
	require.Equal(t, PolicyVersionPublished, published.Status)

	_, err = service.Publish(context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, version.ID)
	require.ErrorIs(t, err, ErrPolicyVersionImmutable)
}

func TestPolicyServicePublishRejectsLegacyManualReviewDraft(t *testing.T) {
	store := &policyStoreStub{versions: map[string]PolicyVersion{}}
	store.versions["legacy-review"] = PolicyVersion{
		ID:      "legacy-review",
		Carrier: "cmcc",
		Status:  PolicyVersionDraft,
		Policy: CompiledPolicy{
			DefaultAction: PolicyDefaultActionReview,
			FailureMode:   FailureModeFailClosed,
		},
	}
	service := NewPolicyService(store, nil)

	_, err := service.Publish(context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, "legacy-review")
	require.ErrorIs(t, err, ErrInvalidAccessInput)
	require.Equal(t, PolicyVersionDraft, store.versions["legacy-review"].Status)
}

func TestPolicyServiceDeletesOnlyDraftInActorCarrier(t *testing.T) {
	store := &policyStoreStub{versions: map[string]PolicyVersion{
		"draft":     {ID: "draft", Carrier: "cmcc", Status: PolicyVersionDraft},
		"published": {ID: "published", Carrier: "cmcc", Status: PolicyVersionPublished},
		"other":     {ID: "other", Carrier: "ctcc", Status: PolicyVersionDraft},
	}}
	service := NewPolicyService(store, nil)
	actor := PolicyActor{Carrier: "cmcc", SubjectID: "operator-1"}

	require.NoError(t, service.DeleteDraft(context.Background(), actor, "draft"))
	require.Equal(t, "draft", store.deleted)
	require.ErrorIs(t, service.DeleteDraft(context.Background(), actor, "published"), ErrPolicyVersionImmutable)
	require.ErrorIs(t, service.DeleteDraft(context.Background(), actor, "other"), ErrPolicyCarrierScope)
}

func TestPolicyServiceDifferenceIgnoresGeneratedIDsAndReportsSemanticChanges(t *testing.T) {
	baseRuleID, targetRuleID := uuid.NewString(), uuid.NewString()
	baseConditionID, targetConditionID := uuid.NewString(), uuid.NewString()
	base := PolicyVersion{ID: "v1", Version: 1, Carrier: "cmcc", Policy: CompiledPolicy{
		DefaultAction: PolicyDefaultActionReject, FailureMode: FailureModeFailClosed, CollectionTimeout: 15 * time.Minute,
		Rules: []CompiledRule{{ID: baseRuleID, Name: "network", Enabled: true, Priority: 100, SerialScope: SerialScope{Type: SerialScopeAll}, Conditions: []CompiledCondition{{
			ID: baseConditionID, Type: ConditionTypeTAC, Operator: ConditionOperatorIn, ExpectedAny: []string{"101", "100"}, Required: true,
		}}}},
	}}
	target := base
	target.ID, target.Version = "v2", 2
	target.Policy.Rules = append([]CompiledRule(nil), base.Policy.Rules...)
	target.Policy.Rules[0].ID = targetRuleID
	target.Policy.Rules[0].Conditions = append([]CompiledCondition(nil), base.Policy.Rules[0].Conditions...)
	target.Policy.Rules[0].Conditions[0].ID = targetConditionID
	target.Policy.Rules[0].Conditions[0].ExpectedAny = []string{"100", "101"}

	difference := comparePolicyVersions(base, target)
	require.True(t, difference.NoChanges)

	target.Policy.Rules[0].Priority = 10
	difference = comparePolicyVersions(base, target)
	require.True(t, difference.NoChanges, "absolute priority changes do not alter a one-rule policy")

	target.Policy.Rules[0].Conditions[0].ExpectedAny = []string{"100", "102"}
	difference = comparePolicyVersions(base, target)
	require.False(t, difference.NoChanges)
	require.Equal(t, 1, difference.RulesChanged)
	require.Equal(t, 1, difference.ConditionsAdded)
	require.Equal(t, 1, difference.ConditionsRemoved)
}

func TestPolicyServiceDifferenceDefaultsToPreviousVersion(t *testing.T) {
	base := PolicyVersion{ID: "v1", Version: 1, Carrier: "cmcc", Policy: CompiledPolicy{DefaultAction: PolicyDefaultActionReject}}
	target := PolicyVersion{ID: "v2", Version: 2, Carrier: "cmcc", Policy: CompiledPolicy{DefaultAction: PolicyDefaultActionReview}}
	store := &policyHistoryStoreStub{policyStoreStub: &policyStoreStub{versions: map[string]PolicyVersion{"v2": target}}, previous: base}

	difference, err := NewPolicyService(store, nil).Difference(
		context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, "v2", "",
	)

	require.NoError(t, err)
	require.True(t, difference.DefaultActionChanged)
	require.Equal(t, "v1", difference.BaseVersionID)
}

func TestPolicyServiceRollbackPublishesNewVersionWithoutMutatingHistory(t *testing.T) {
	ruleID, conditionID := uuid.NewString(), uuid.NewString()
	source := PolicyVersion{ID: "retired-v1", Carrier: "cmcc", Name: "production", Status: PolicyVersionRetired, Policy: CompiledPolicy{
		DefaultAction: PolicyDefaultActionReject, FailureMode: FailureModeFailClosed, CollectionTimeout: 15 * time.Minute,
		Rules: []CompiledRule{{ID: ruleID, Name: "network", Enabled: true, Priority: 100, SerialScope: SerialScope{Type: SerialScopeAll}, Conditions: []CompiledCondition{{
			ID: conditionID, Type: ConditionTypeTAC, Operator: ConditionOperatorEqual, Expected: "100", Required: true,
		}}}},
	}}
	store := &policyStoreStub{versions: map[string]PolicyVersion{source.ID: source}}

	published, err := NewPolicyService(store, nil).Rollback(
		context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, source.ID,
	)

	require.NoError(t, err)
	require.Equal(t, PolicyVersionPublished, published.Status)
	require.Equal(t, "draft-1", published.ID)
	require.NotEqual(t, ruleID, published.Policy.Rules[0].ID)
	require.Equal(t, source, store.versions[source.ID])
}

func TestPolicyServiceRollbackRejectsNonRetiredSource(t *testing.T) {
	store := &policyStoreStub{versions: map[string]PolicyVersion{
		"current": {ID: "current", Carrier: "cmcc", Status: PolicyVersionPublished},
	}}
	_, err := NewPolicyService(store, nil).Rollback(
		context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, "current",
	)
	require.ErrorIs(t, err, ErrPolicyRollbackSource)
}

func TestPolicyServiceQueuesSingleDeviceReevaluation(t *testing.T) {
	queue := &reevaluationQueueStub{}
	service := NewPolicyService(&policyStoreStub{}, queue)
	err := service.ReevaluateOne(context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, "SN-1")
	require.NoError(t, err)
	require.Len(t, queue.requests, 1)
	require.Equal(t, "SN-1", queue.requests[0].SerialNumber)
}

func TestPolicyServiceReevaluationEnforcesTargetIdentityScope(t *testing.T) {
	queue := &reevaluationQueueStub{}
	checker := &identityVisibilityStub{}
	service := NewPolicyService(&policyStoreStub{}, queue)
	service.SetIdentityVisibilityChecker(checker)
	groupID := uuid.New()

	err := service.ReevaluateOne(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: "u1", VisibleGroups: []uuid.UUID{groupID},
	}, "SN-OTHER-GROUP")

	require.Error(t, err)
	require.Empty(t, queue.requests)
	require.Equal(t, "SN-OTHER-GROUP", checker.serialNumber)
}

func TestPolicyServiceRejectsPolicyDataThatCannotBePersistedOrEvaluated(t *testing.T) {
	validPolicy := func() CompiledPolicy {
		return CompiledPolicy{Rules: []CompiledRule{policyServiceRule()}}
	}
	tests := []struct {
		name    string
		mutate  func(*CompiledPolicy)
		wantErr string
	}{
		{
			name: "unsupported default action",
			mutate: func(policy *CompiledPolicy) {
				policy.DefaultAction = "accept"
			},
			wantErr: "policy default action must be",
		},
		{
			name: "manual review default has no operator workflow",
			mutate: func(policy *CompiledPolicy) {
				policy.DefaultAction = PolicyDefaultActionReview
			},
			wantErr: "policy default action must be",
		},
		{
			name: "manual review failure hold has no operator workflow",
			mutate: func(policy *CompiledPolicy) {
				policy.FailureMode = FailureModeReviewHold
			},
			wantErr: "policy failure mode must be",
		},
		{
			name: "embedded list entry",
			mutate: func(policy *CompiledPolicy) {
				policy.ListEntries = []CompiledListEntry{{Type: ListEntryTypeAllow}}
			},
			wantErr: "access-list API",
		},
		{
			name: "duplicate rule id",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules = append(policy.Rules, policyServiceRule())
			},
			wantErr: "rule id \"rule-1\" is duplicated",
		},
		{
			name: "duplicate explicit priority",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].Priority = 10
				second := policyServiceRule()
				second.ID = "rule-2"
				second.Priority = 10
				policy.Rules = append(policy.Rules, second)
			},
			wantErr: "priority 10 is duplicated",
		},
		{
			name: "negative priority",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].Priority = -1
			},
			wantErr: "invalid priority -1",
		},
		{
			name: "empty serial list",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].SerialScope = SerialScope{Type: SerialScopeList}
			},
			wantErr: "serial list is empty",
		},
		{
			name: "reversed serial range",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].SerialScope = SerialScope{Type: SerialScopeRange, Start: "SN-9", End: "SN-1"}
			},
			wantErr: "range start exceeds end",
		},
		{
			name: "identity pseudo condition",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].Conditions[0].Type = ConditionTypeIdentity
			},
			wantErr: "unsupported type \"identity\"",
		},
		{
			name: "TAC with CIDR operator",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].Conditions[0].Operator = ConditionOperatorCIDR
			},
			wantErr: "does not support operator \"cidr\"",
		},
		{
			name: "invalid source network",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].Conditions[0] = CompiledCondition{
					ID: "source-network", Type: ConditionTypeObservedIP,
					Operator: ConditionOperatorCIDR, Expected: "10.0.0.0/99", Required: true,
				}
			},
			wantErr: "invalid CIDR",
		},
		{
			name: "reversed IP range",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].Conditions[0] = CompiledCondition{
					ID: "source-range", Type: ConditionTypeObservedIP,
					Operator: ConditionOperatorIPRange, IPRange: &IPRange{Start: "10.0.0.20", End: "10.0.0.1"}, Required: true,
				}
			},
			wantErr: "invalid IP range",
		},
		{
			name: "invalid geofence",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].Conditions[0] = CompiledCondition{
					ID: "gps", Type: ConditionTypeGPS, Operator: ConditionOperatorWithinRadius,
					GeoFence: &GeoFence{Center: GeoPoint{Latitude: 91}, RadiusMeters: 100}, Required: true,
				}
			},
			wantErr: "invalid geofence center",
		},
		{
			name: "reversed GPS bounds",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].Conditions[0] = CompiledCondition{
					ID: "gps-bounds", Type: ConditionTypeGPS, Operator: ConditionOperatorWithinBounds,
					GeoBounds: &GeoBounds{MinLatitude: 32, MaxLatitude: 30, MinLongitude: 120, MaxLongitude: 122}, Required: true,
				}
			},
			wantErr: "invalid GPS bounds",
		},
		{
			name: "negative evidence TTL",
			mutate: func(policy *CompiledPolicy) {
				policy.Rules[0].Conditions[0].EvidenceTTL = -time.Second
			},
			wantErr: "negative evidence TTL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := validPolicy()
			tt.mutate(&policy)
			service := NewPolicyService(&policyStoreStub{}, nil)
			_, err := service.CreateDraft(
				context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "u1"}, "policy", policy,
			)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
