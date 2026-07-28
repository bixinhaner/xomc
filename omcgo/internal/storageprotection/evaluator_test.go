package storageprotection

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRepository struct {
	policy *Policy
	events []Event
}

func (r *fakeRepository) GetEnabledPolicy(context.Context, TargetType, string, WriteScope) (*Policy, error) {
	if r.policy == nil {
		return nil, nil
	}
	copy := *r.policy
	return &copy, nil
}
func (r *fakeRepository) List(context.Context) ([]Policy, error) {
	if r.policy == nil {
		return nil, nil
	}
	return []Policy{*r.policy}, nil
}
func (r *fakeRepository) Save(context.Context, *Policy) (*Policy, error) { return r.policy, nil }
func (r *fakeRepository) UpdateState(_ context.Context, policy *Policy) error {
	copy := *policy
	r.policy = &copy
	return nil
}
func (r *fakeRepository) RecordEvent(_ context.Context, event Event) error {
	r.events = append(r.events, event)
	return nil
}
func (r *fakeRepository) ListEvents(context.Context, TargetType, string, int) ([]Event, error) {
	return append([]Event(nil), r.events...), nil
}

type fakeUsageProvider struct {
	snapshot UsageSnapshot
	err      error
}

func (p fakeUsageProvider) Snapshot(context.Context, TargetType, string) (UsageSnapshot, error) {
	return p.snapshot, p.err
}

func TestStorageProtectionStateMachineRequiresConfirmationAndRecovers(t *testing.T) {
	repo := &fakeRepository{policy: &Policy{
		ID: "policy-1", TargetType: TargetFilesystem, TargetID: "root", WriteScope: WriteScopeAll,
		Enabled: true, WarnUsedPercent: 80, BlockUsedPercent: 90, RecoverUsedPercent: 85,
		CheckIntervalSeconds: 30, UnknownBehavior: UnknownAllowWithAlarm, CurrentState: StateNormal,
		Version: 1,
	}}
	usage := &fakeUsageProvider{snapshot: UsageSnapshot{TargetType: TargetFilesystem, TargetID: "root", UsedRatio: .80, CapacityBytes: 100, UsedBytes: 80, Available: true, ObservedAt: time.Now()}}
	svc := NewService(repo, usage, nil, zap.NewNop())

	decision, err := svc.Check(context.Background(), TargetFilesystem, "root", WriteScopeAll)
	require.NoError(t, err)
	require.Equal(t, StateNormal, decision.State)
	decision, err = svc.Check(context.Background(), TargetFilesystem, "root", WriteScopeAll)
	require.NoError(t, err)
	require.Equal(t, StateWarning, decision.State)

	usage.snapshot.UsedRatio = .91
	decision, err = svc.Check(context.Background(), TargetFilesystem, "root", WriteScopeAll)
	require.NoError(t, err)
	require.Equal(t, StateWarning, decision.State)
	decision, err = svc.Check(context.Background(), TargetFilesystem, "root", WriteScopeAll)
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.Equal(t, StateBlocked, decision.State)

	usage.snapshot.UsedRatio = .84
	decision, err = svc.Check(context.Background(), TargetFilesystem, "root", WriteScopeAll)
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	decision, err = svc.Check(context.Background(), TargetFilesystem, "root", WriteScopeAll)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.Equal(t, StateNormal, decision.State)
	require.GreaterOrEqual(t, len(repo.events), 3)
}

func TestStorageProtectionUnknownBehavior(t *testing.T) {
	for _, tc := range []struct {
		behavior UnknownBehavior
		allowed  bool
	}{
		{UnknownAllowWithAlarm, true},
		{UnknownBlockNewWrites, false},
	} {
		repo := &fakeRepository{policy: &Policy{
			ID: "policy-unknown", TargetType: TargetMinIO, TargetID: "minio-data", WriteScope: WriteScopeUpload,
			Enabled: true, WarnUsedPercent: 80, BlockUsedPercent: 90, RecoverUsedPercent: 85,
			CheckIntervalSeconds: 30, UnknownBehavior: tc.behavior, CurrentState: StateNormal,
			Version: 1,
		}}
		svc := NewService(repo, fakeUsageProvider{snapshot: UsageSnapshot{Available: false, Reason: "prometheus unavailable"}}, nil, zap.NewNop())
		decision, err := svc.Check(context.Background(), TargetMinIO, "minio-data", WriteScopeUpload)
		require.NoError(t, err)
		require.Equal(t, tc.allowed, decision.Allowed)
		require.Equal(t, StateUnknown, decision.State)
	}
}

func TestValidateStorageProtectionPolicy(t *testing.T) {
	policy := &Policy{TargetType: TargetFilesystem, TargetID: "root", WriteScope: WriteScopeAll, WarnUsedPercent: 80, RecoverUsedPercent: 85, BlockUsedPercent: 90, CheckIntervalSeconds: 30, UnknownBehavior: UnknownAllowWithAlarm}
	require.NoError(t, validatePolicy(policy))
	policy.RecoverUsedPercent = 75
	require.Error(t, validatePolicy(policy))
}
