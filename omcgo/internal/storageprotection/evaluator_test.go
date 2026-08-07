package storageprotection

import (
	"context"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/components"
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
	targets  []UsageSnapshot
	err      error
}

type fakeLogAdmissionController struct{ blocked bool }

func (g *fakeLogAdmissionController) SetBlocked(blocked bool) { g.blocked = blocked }

func (p fakeUsageProvider) Snapshot(context.Context, TargetType, string) (UsageSnapshot, error) {
	return p.snapshot, p.err
}

func (p fakeUsageProvider) ListTargets(context.Context) ([]UsageSnapshot, error) {
	if len(p.targets) > 0 {
		return append([]UsageSnapshot(nil), p.targets...), p.err
	}
	return []UsageSnapshot{p.snapshot}, p.err
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

func TestStorageProtectionWarningRemainsLatchedWhileUsageStaysHigh(t *testing.T) {
	repo := &fakeRepository{policy: &Policy{
		ID: "policy-warning", TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, WriteScope: WriteScopeAll,
		Enabled: true, WarnUsedPercent: 80, BlockUsedPercent: 90, RecoverUsedPercent: 85,
		CheckIntervalSeconds: 30, UnknownBehavior: UnknownAllowWithAlarm, CurrentState: StateNormal,
		Version: 1,
	}}
	usage := &fakeUsageProvider{snapshot: UsageSnapshot{UsedRatio: .81, CapacityBytes: 100, UsedBytes: 81, Available: true, ObservedAt: time.Now()}}
	svc := NewService(repo, usage, nil, zap.NewNop())

	decision, err := svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.Equal(t, StateNormal, decision.State)
	decision, err = svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.Equal(t, StateWarning, decision.State)
	decision, err = svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.Equal(t, StateWarning, decision.State)
	require.Len(t, repo.events, 1)
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
			ID: "policy-unknown", TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, WriteScope: WriteScopeAll,
			Enabled: true, WarnUsedPercent: 80, BlockUsedPercent: 90, RecoverUsedPercent: 85,
			CheckIntervalSeconds: 30, UnknownBehavior: tc.behavior, CurrentState: StateNormal,
			Version: 1,
		}}
		svc := NewService(repo, fakeUsageProvider{snapshot: UsageSnapshot{Available: false, Reason: "prometheus unavailable"}}, nil, zap.NewNop())
		decision, err := svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeUpload)
		require.NoError(t, err)
		require.Equal(t, tc.allowed, decision.Allowed)
		require.Equal(t, StateUnknown, decision.State)
	}
}

func TestStorageProtectionClosesAndReopensLogAdmission(t *testing.T) {
	repo := &fakeRepository{policy: &Policy{
		ID: "policy-log", TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, WriteScope: WriteScopeAll,
		Enabled: true, WarnUsedPercent: 80, BlockUsedPercent: 90, RecoverUsedPercent: 85,
		CheckIntervalSeconds: 30, UnknownBehavior: UnknownAllowWithAlarm, CurrentState: StateNormal,
		Version: 1,
	}}
	usage := &fakeUsageProvider{snapshot: UsageSnapshot{
		TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, UsedRatio: .91,
		CapacityBytes: 100, UsedBytes: 91, Available: true, ObservedAt: time.Now(),
	}}
	gate := &fakeLogAdmissionController{}
	svc := NewService(repo, usage, nil, zap.NewNop())
	svc.SetLogAdmissionController(gate)

	_, err := svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.False(t, gate.blocked, "first threshold observation waits for confirmation")
	_, err = svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.True(t, gate.blocked, "blocked storage state must close the service log gate")

	usage.snapshot.UsedRatio = .84
	_, err = svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.True(t, gate.blocked, "recovery also requires two observations")
	_, err = svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.False(t, gate.blocked, "normal storage state must reopen the service log gate")
}

func TestStorageProtectionUnknownClosesLogAdmission(t *testing.T) {
	repo := &fakeRepository{policy: &Policy{
		ID: "policy-log-unknown", TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, WriteScope: WriteScopeAll,
		Enabled: true, WarnUsedPercent: 80, BlockUsedPercent: 90, RecoverUsedPercent: 85,
		CheckIntervalSeconds: 30, UnknownBehavior: UnknownAllowWithAlarm, CurrentState: StateNormal,
		Version: 1,
	}}
	gate := &fakeLogAdmissionController{}
	svc := NewService(repo, fakeUsageProvider{snapshot: UsageSnapshot{Available: false, Reason: "collector unavailable"}}, nil, zap.NewNop())
	svc.SetLogAdmissionController(gate)

	_, err := svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.True(t, gate.blocked, "unknown capacity must fail-closed for logs")
}

func TestStorageProtectionBlocksOnWorstProtectedMountpoint(t *testing.T) {
	repo := &fakeRepository{policy: &Policy{
		ID: "policy-worst", TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, WriteScope: WriteScopeAll,
		Enabled: true, WarnUsedPercent: 80, BlockUsedPercent: 90, RecoverUsedPercent: 85,
		CheckIntervalSeconds: 30, UnknownBehavior: UnknownAllowWithAlarm, CurrentState: StateNormal,
		Version: 1,
	}}
	at := time.Now()
	provider := NewCollectorUsageProviderWithResolver(
		fakeStorageCollector{metrics: []components.StorageMetric{
			hostMetric("/opt", 1000, 700, 70, at),
			hostMetric("/var/lib/docker", 1000, 920, 92, at),
		}},
		staticProtectedPathResolver{paths: []ProtectedPath{
			{ID: "logs", Path: "/opt/omc/run/logs"},
			{ID: "postgres", Path: "/var/lib/docker/volumes/omcgo_pgdata/_data"},
		}},
	)
	svc := NewService(repo, provider, nil, zap.NewNop())

	decision, err := svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.True(t, decision.Allowed, "first block observation waits for confirmation")
	decision, err = svc.Check(context.Background(), TargetFilesystem, UnifiedStorageTargetID, WriteScopeAll)
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.Equal(t, StateBlocked, decision.State)
	require.NotNil(t, repo.policy.LastObservedRatio)
	require.Equal(t, .92, *repo.policy.LastObservedRatio)
	require.Contains(t, decision.Reason, "/var/lib/docker")
}

func TestValidateStorageProtectionPolicy(t *testing.T) {
	policy := &Policy{TargetType: TargetFilesystem, TargetID: "root", WriteScope: WriteScopeAll, WarnUsedPercent: 80, RecoverUsedPercent: 85, BlockUsedPercent: 90, CheckIntervalSeconds: 30, UnknownBehavior: UnknownAllowWithAlarm}
	require.NoError(t, validatePolicy(policy))
	policy.RecoverUsedPercent = 75
	require.Error(t, validatePolicy(policy))
	policy.RecoverUsedPercent = 85
	policy.TargetType = TargetMinIO
	require.Error(t, validatePolicy(policy))
	policy.TargetType = TargetFilesystem
	policy.WriteScope = WriteScopeUpload
	require.Error(t, validatePolicy(policy))
}

func TestListTargetsAlwaysIncludesUnifiedPhysicalFilesystem(t *testing.T) {
	total, used := uint64(100), uint64(80)
	repo := &fakeRepository{}
	svc := NewService(repo, fakeUsageProvider{snapshot: UsageSnapshot{
		TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID,
		CapacityBytes: total, UsedBytes: used, UsedRatio: .8, Available: true, ObservedAt: time.Now(),
	}}, nil, zap.NewNop())

	targets, err := svc.ListTargets(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, TargetFilesystem, targets[0].TargetType)
	require.Equal(t, UnifiedStorageTargetID, targets[0].TargetID)
	require.Equal(t, total, targets[0].CapacityBytes)
	require.Equal(t, used, targets[0].UsedBytes)
	require.True(t, targets[0].Available)
}

func TestListTargetsReturnsProtectedMountpointsWithGlobalPolicyState(t *testing.T) {
	repo := &fakeRepository{policy: &Policy{
		ID: "policy-targets", TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, WriteScope: WriteScopeAll,
		Enabled: true, WarnUsedPercent: 80, BlockUsedPercent: 90, RecoverUsedPercent: 85,
		CheckIntervalSeconds: 30, UnknownBehavior: UnknownAllowWithAlarm, CurrentState: StateNormal,
		Version: 1,
	}}
	at := time.Now()
	svc := NewService(repo, fakeUsageProvider{targets: []UsageSnapshot{
		{TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, Mountpoint: "/", CapacityBytes: 1000, UsedBytes: 500, UsedRatio: .5, Available: true, ObservedAt: at, ProtectedPaths: []string{"/opt/omc/data"}},
		{TargetType: TargetFilesystem, TargetID: "mount-data", Mountpoint: "/data", CapacityBytes: 1000, UsedBytes: 910, UsedRatio: .91, Available: true, ObservedAt: at, ProtectedPaths: []string{"/data/minio"}},
	}}, nil, zap.NewNop())

	targets, err := svc.ListTargets(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 2)
	require.Equal(t, UnifiedStorageTargetID, targets[0].TargetID)
	require.Equal(t, "mount-data", targets[1].TargetID)
	require.Equal(t, "/data", targets[1].Mountpoint)
	require.Equal(t, StateBlocked, targets[1].CurrentState)
	require.Equal(t, []WriteScope{WriteScopeAll}, targets[1].WriteScopes)
	require.Equal(t, []string{"/data/minio"}, targets[1].ProtectedPaths)
}
