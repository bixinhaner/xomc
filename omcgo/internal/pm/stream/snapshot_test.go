package stream

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type revisionMatchableLoader struct {
	mu            sync.Mutex
	revision      MatchableRevision
	versions      []*TaskVersionSnapshot
	loadCount     int
	revisionCount int
	loadErr       error
	revisionErr   error
}

func (l *revisionMatchableLoader) LoadMatchable(
	context.Context,
	time.Time,
) ([]*TaskVersionSnapshot, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loadCount++
	return l.versions, l.loadErr
}

func (l *revisionMatchableLoader) LoadMatchableRevision(
	context.Context,
) (MatchableRevision, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.revisionCount++
	return l.revision, l.revisionErr
}

func (l *revisionMatchableLoader) setRevision(revision MatchableRevision) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.revision = revision
}

func (l *revisionMatchableLoader) setVersions(versions []*TaskVersionSnapshot) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.versions = versions
}

func (l *revisionMatchableLoader) setErrors(loadErr, revisionErr error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loadErr = loadErr
	l.revisionErr = revisionErr
}

func (l *revisionMatchableLoader) counts() (int, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.loadCount, l.revisionCount
}

func snapshotTestVersion() *TaskVersionSnapshot {
	return &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: time.Now().UTC().Add(-time.Hour),
		Metrics:       map[string]MetricRule{},
		Counters:      map[string]CounterRule{},
		Members:       map[uuid.UUID][]TaskMember{},
	}
}

func TestSnapshotRefreshSkipsFullLoadWhenRevisionIsUnchanged(t *testing.T) {
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)},
		versions: []*TaskVersionSnapshot{snapshotTestVersion()},
	}
	store := NewSnapshotStore(loader, nil)

	require.NoError(t, store.Refresh(context.Background()))
	first := store.Current()
	require.NoError(t, store.Refresh(context.Background()))

	loadCount, revisionCount := loader.counts()
	require.Equal(t, 1, loadCount)
	require.Equal(t, 2, revisionCount)
	require.Same(t, first, store.Current(), "unchanged refresh must reuse the published snapshot")
}

func TestSnapshotRefreshReloadsAfterRevisionChanges(t *testing.T) {
	firstVersion := snapshotTestVersion()
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)},
		versions: []*TaskVersionSnapshot{firstVersion},
	}
	store := NewSnapshotStore(loader, nil)
	require.NoError(t, store.Refresh(context.Background()))

	secondVersion := snapshotTestVersion()
	loader.setRevision(MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(2, 0)})
	loader.setVersions([]*TaskVersionSnapshot{secondVersion})
	require.NoError(t, store.Refresh(context.Background()))

	loadCount, _ := loader.counts()
	require.Equal(t, 2, loadCount)
	require.NotContains(t, store.Current().ByVersion, firstVersion.VersionID)
	require.Contains(t, store.Current().ByVersion, secondVersion.VersionID)
}

func TestSnapshotRefreshReloadsWhenFingerprintChangesAtSameMaximumTimestamp(t *testing.T) {
	updatedAt := time.Unix(2, 0)
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{
			TaskCount: 1, UpdatedAt: updatedAt, Fingerprint: "before",
		},
		versions: []*TaskVersionSnapshot{snapshotTestVersion()},
	}
	store := NewSnapshotStore(loader, nil)
	require.NoError(t, store.Refresh(context.Background()))

	loader.setRevision(MatchableRevision{
		TaskCount: 1, UpdatedAt: updatedAt, Fingerprint: "after",
	})
	require.NoError(t, store.Refresh(context.Background()))

	loadCount, _ := loader.counts()
	require.Equal(t, 2, loadCount)
}

func TestSnapshotRefreshRevisionFailurePreservesLastGoodSnapshot(t *testing.T) {
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)},
		versions: []*TaskVersionSnapshot{snapshotTestVersion()},
	}
	store := NewSnapshotStore(loader, nil)
	require.NoError(t, store.Refresh(context.Background()))
	lastGood := store.Current()
	loader.setErrors(nil, errors.New("revision unavailable"))

	err := store.Refresh(context.Background())

	require.ErrorContains(t, err, "revision unavailable")
	require.Same(t, lastGood, store.Current())
}

func TestSnapshotRefreshLoadFailureRetriesChangedRevision(t *testing.T) {
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)},
		versions: []*TaskVersionSnapshot{snapshotTestVersion()},
	}
	store := NewSnapshotStore(loader, nil)
	require.NoError(t, store.Refresh(context.Background()))
	lastGood := store.Current()
	loader.setRevision(MatchableRevision{TaskCount: 2, UpdatedAt: time.Unix(2, 0)})
	loader.setErrors(errors.New("catalog unavailable"), nil)

	err := store.Refresh(context.Background())
	require.ErrorContains(t, err, "catalog unavailable")
	require.Same(t, lastGood, store.Current())

	loader.setErrors(nil, nil)
	require.NoError(t, store.Refresh(context.Background()))
	loadCount, _ := loader.counts()
	require.Equal(t, 3, loadCount, "failed changed revision must be retried")
}

type legacyMatchableLoader struct {
	mu        sync.Mutex
	loadCount int
	versions  []*TaskVersionSnapshot
}

func (l *legacyMatchableLoader) LoadMatchable(
	context.Context,
	time.Time,
) ([]*TaskVersionSnapshot, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loadCount++
	return l.versions, nil
}

func TestSnapshotRefreshLegacyLoaderKeepsFullRefreshBehavior(t *testing.T) {
	loader := &legacyMatchableLoader{versions: []*TaskVersionSnapshot{snapshotTestVersion()}}
	store := NewSnapshotStore(loader, nil)

	require.NoError(t, store.Refresh(context.Background()))
	first := store.Current()
	require.NoError(t, store.Refresh(context.Background()))

	loader.mu.Lock()
	loadCount := loader.loadCount
	loader.mu.Unlock()
	require.Equal(t, 2, loadCount)
	require.NotSame(t, first, store.Current())
}

func TestSnapshotRefreshConcurrentFirstLoadRunsOnce(t *testing.T) {
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)},
		versions: []*TaskVersionSnapshot{snapshotTestVersion()},
	}
	store := NewSnapshotStore(loader, nil)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.Refresh(context.Background())
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	loadCount, revisionCount := loader.counts()
	require.Equal(t, 1, loadCount)
	require.Equal(t, 16, revisionCount)
}

func TestSnapshotRefreshRebuildsCachedVersionsOnlyAtTimeBoundary(t *testing.T) {
	boundary := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	version := snapshotTestVersion()
	version.EffectiveFrom = boundary
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)},
		versions: []*TaskVersionSnapshot{version},
	}
	store := NewSnapshotStore(loader, nil)
	now := boundary.Add(-time.Minute)
	store.now = func() time.Time { return now }
	require.NoError(t, store.Refresh(context.Background()))
	beforeBoundary := store.Current()

	now = boundary.Add(-time.Second)
	require.NoError(t, store.Refresh(context.Background()))
	require.Same(t, beforeBoundary, store.Current())

	now = boundary
	require.NoError(t, store.Refresh(context.Background()))
	require.NotSame(t, beforeBoundary, store.Current())
	loadCount, _ := loader.counts()
	require.Equal(t, 1, loadCount, "time boundary rebuild must use cached source versions")
}

func TestSnapshotRefreshDoesNotMutateCachedSourceVersions(t *testing.T) {
	version := snapshotTestVersion()
	deviceID := uuid.New()
	version.Members[deviceID] = []TaskMember{{
		DeviceID: deviceID, DimensionKey: "network",
	}}
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)},
		versions: []*TaskVersionSnapshot{version},
	}
	store := NewSnapshotStore(loader, nil)

	require.NoError(t, store.Refresh(context.Background()))

	require.Nil(t, version.DimensionMemberCounts)
	require.Equal(t, int64(1), store.Current().ByVersion[version.VersionID].DimensionMemberCounts["network"])
}

func TestSnapshotRunRefreshPollsRevisionWithoutRepeatingFullLoad(t *testing.T) {
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)},
		versions: []*TaskVersionSnapshot{snapshotTestVersion()},
	}
	store := NewSnapshotStore(loader, nil)
	require.NoError(t, store.Refresh(context.Background()))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		store.RunRefresh(ctx, time.Millisecond)
	}()
	require.Eventually(t, func() bool {
		_, revisionCount := loader.counts()
		return revisionCount >= 3
	}, time.Second, time.Millisecond)
	cancel()
	<-done

	loadCount, _ := loader.counts()
	require.Equal(t, 1, loadCount)
}

func TestBuildTaskSnapshotSynthesizesDevicePipelineFromNetworkCatalog(t *testing.T) {
	deviceID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	networkVersion := &TaskVersionSnapshot{
		TaskID:    builtinNetworkRuleIDs["lte"],
		VersionID: uuid.MustParse("30000000-0000-4000-8000-000000000001"),
		VersionNo: 1, Name: "已重命名的系统规则", Enabled: true,
		Technology: "lte", Dimension: DimensionNetwork,
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily,
			GranularityWeekly, GranularityMonthly,
		},
		EffectiveFrom: time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC),
		Metrics: map[string]MetricRule{
			"K1": {MetricID: "K1", MetricPath: "K1", Aggregation: AggregationSum},
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{
				DeviceID: deviceID, DeviceSN: "SN-1",
				DimensionKey: "network", DimensionName: "Network",
			}},
		},
	}

	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{networkVersion})

	versions := snapshot.ByDevice[deviceID]
	require.Len(t, versions, 2)
	var deviceVersion *TaskVersionSnapshot
	for _, version := range versions {
		if version.DevicePipeline {
			deviceVersion = version
		}
	}
	require.NotNil(t, deviceVersion)
	require.Equal(t, DimensionDevice, deviceVersion.Dimension)
	require.NotEqual(t, networkVersion.TaskID, deviceVersion.TaskID)
	require.NotEqual(t, networkVersion.VersionID, deviceVersion.VersionID)
	require.Equal(t, deviceID.String(), deviceVersion.Members[deviceID][0].DimensionKey)
	require.Equal(t, networkVersion.Counters, deviceVersion.Counters)
	require.Same(t, deviceVersion, snapshot.ByVersion[deviceVersion.VersionID])
}

func TestBuildTaskSnapshotDoesNotSynthesizeDevicePipelineForCustomNetworkRule(t *testing.T) {
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Name: "我的全网分析",
		Enabled: true, Technology: "lte", Dimension: DimensionNetwork,
		Members: map[uuid.UUID][]TaskMember{uuid.New(): {{DeviceID: uuid.New()}}},
	}

	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{version})

	require.Len(t, snapshot.ByVersion, 1)
}

func TestDevicePipelineUsesImmutableHourlyVersionAndStableRollupLineage(t *testing.T) {
	first := &TaskVersionSnapshot{
		TaskID: builtinNetworkRuleIDs["lte"], VersionID: uuid.New(),
		Technology: "lte", Dimension: DimensionNetwork,
	}
	second := *first
	second.VersionID = uuid.New()

	firstPipeline, ok := devicePipelineVersion(first)
	require.True(t, ok)
	secondPipeline, ok := devicePipelineVersion(&second)
	require.True(t, ok)

	require.NotEqual(t, firstPipeline.VersionID, secondPipeline.VersionID)
	require.Equal(t, firstPipeline.RollupVersionID, secondPipeline.RollupVersionID)
	require.NotEqual(t, uuid.Nil, firstPipeline.RollupVersionID)
}

func TestBuildTaskSnapshotKeepsHourlyDefinitionsImmutable(t *testing.T) {
	deviceID := uuid.New()
	first := &TaskVersionSnapshot{
		TaskID: builtinNetworkRuleIDs["lte"], VersionID: uuid.New(), VersionNo: 1,
		Enabled: true, Technology: "lte", Dimension: DimensionNetwork,
		EffectiveFrom:        time.Now().UTC().Add(-2 * time.Hour),
		LineageEffectiveFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Counters: map[string]CounterRule{
			"OLD": {MetricPath: "OLD", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DimensionKey: "network"}},
		},
	}
	end := time.Now().UTC().Add(-time.Hour)
	first.EffectiveTo = &end
	second := &TaskVersionSnapshot{
		TaskID: first.TaskID, VersionID: uuid.New(), VersionNo: 2,
		Enabled: true, Technology: "lte", Dimension: DimensionNetwork,
		EffectiveFrom: end, LineageEffectiveFrom: first.LineageEffectiveFrom,
		Counters: map[string]CounterRule{
			"NEW": {MetricPath: "NEW", Aggregation: AggregationSum},
		},
		Members: first.Members,
	}

	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{first, second})
	firstPipeline, ok := devicePipelineVersion(first)
	require.True(t, ok)
	secondPipeline, ok := devicePipelineVersion(second)
	require.True(t, ok)

	require.Equal(t, first.Counters, snapshot.ByVersion[firstPipeline.VersionID].Counters)
	require.Equal(t, second.Counters, snapshot.ByVersion[secondPipeline.VersionID].Counters)
	rollup := snapshot.ByVersion[secondPipeline.RollupVersionID]
	require.NotNil(t, rollup)
	require.True(t, rollup.DeviceRollup)
	require.Contains(t, rollup.Counters, "OLD")
	require.Contains(t, rollup.Counters, "NEW")
	require.Equal(t, second.Metrics, rollup.Metrics)
	require.Equal(t, first.LineageEffectiveFrom, rollup.EffectiveFrom)
}

func TestDeviceRollupEpochDoesNotDriftWithMatchableHistory(t *testing.T) {
	source := &TaskVersionSnapshot{
		TaskID: builtinNetworkRuleIDs["lte"], VersionID: uuid.New(),
		Enabled: true, Technology: "lte", Dimension: DimensionNetwork,
		EffectiveFrom:        time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		LineageEffectiveFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	rollup, ok := deviceRollupVersion(source)

	require.True(t, ok)
	require.Equal(t, source.LineageEffectiveFrom, rollup.EffectiveFrom)
}
