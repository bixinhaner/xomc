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
			wantErr: "unsupported policy default action",
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
