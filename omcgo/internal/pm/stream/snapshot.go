package stream

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MatchableLoader interface {
	LoadMatchable(ctx context.Context, at time.Time) ([]*TaskVersionSnapshot, error)
}

type MatchableRevision struct {
	TaskCount int64
	UpdatedAt time.Time
}

type MatchableRevisionLoader interface {
	LoadMatchableRevision(ctx context.Context) (MatchableRevision, error)
}

type TaskSnapshot struct {
	LoadedAt  time.Time
	ByDevice  map[uuid.UUID][]*TaskVersionSnapshot
	ByVersion map[uuid.UUID]*TaskVersionSnapshot
}

type SnapshotStore struct {
	loader MatchableLoader
	value  atomic.Pointer[TaskSnapshot]
	logger *zap.Logger
	now    func() time.Time

	reloadMu     sync.Mutex
	versions     []*TaskVersionSnapshot
	revision     MatchableRevision
	initialized  bool
	nextBoundary time.Time
}

func NewSnapshotStore(loader MatchableLoader, logger *zap.Logger) *SnapshotStore {
	if logger == nil {
		logger = zap.NewNop()
	}
	store := &SnapshotStore{
		loader: loader, logger: logger,
		now: func() time.Time { return time.Now().UTC() },
	}
	store.value.Store(&TaskSnapshot{
		LoadedAt:  time.Now().UTC(),
		ByDevice:  map[uuid.UUID][]*TaskVersionSnapshot{},
		ByVersion: map[uuid.UUID]*TaskVersionSnapshot{},
	})
	return store
}

func (s *SnapshotStore) Reload(ctx context.Context) error {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()
	return s.reloadLocked(ctx, nil)
}

func (s *SnapshotStore) Refresh(ctx context.Context) error {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()
	revisionLoader, ok := s.loader.(MatchableRevisionLoader)
	if !ok {
		return s.reloadLocked(ctx, nil)
	}
	revision, err := revisionLoader.LoadMatchableRevision(ctx)
	if err != nil {
		return fmt.Errorf("load PM aggregation task revision: %w", err)
	}
	now := s.now()
	if s.initialized && sameMatchableRevision(revision, s.revision) {
		if s.nextBoundary.IsZero() || now.Before(s.nextBoundary) {
			return nil
		}
		s.publishCached(now)
		return nil
	}
	return s.reloadLocked(ctx, &revision)
}

func sameMatchableRevision(left, right MatchableRevision) bool {
	return left.TaskCount == right.TaskCount && left.UpdatedAt.Equal(right.UpdatedAt)
}

func (s *SnapshotStore) reloadLocked(
	ctx context.Context,
	knownRevision *MatchableRevision,
) error {
	var revision MatchableRevision
	if revisionLoader, ok := s.loader.(MatchableRevisionLoader); ok {
		if knownRevision != nil {
			revision = *knownRevision
		} else {
			loaded, err := revisionLoader.LoadMatchableRevision(ctx)
			if err != nil {
				return fmt.Errorf("load PM aggregation task revision: %w", err)
			}
			revision = loaded
		}
	}
	now := s.now()
	versions, err := s.loader.LoadMatchable(ctx, now)
	if err != nil {
		return fmt.Errorf("load matchable PM aggregation tasks: %w", err)
	}
	s.versions = versions
	s.revision = revision
	s.initialized = true
	s.publishCached(now)
	return nil
}

func (s *SnapshotStore) publishCached(now time.Time) {
	versions, nextBoundary := snapshotVersionsAt(s.versions, now)
	next := BuildTaskSnapshot(versions)
	next.LoadedAt = now
	s.value.Store(next)
	s.nextBoundary = nextBoundary
}

const matchableVersionHistory = 45 * 24 * time.Hour

func snapshotVersionsAt(
	versions []*TaskVersionSnapshot,
	now time.Time,
) ([]*TaskVersionSnapshot, time.Time) {
	cutoff := now.Add(-matchableVersionHistory)
	filtered := make([]*TaskVersionSnapshot, 0, len(versions))
	var nextBoundary time.Time
	addBoundary := func(candidate time.Time) {
		if !candidate.After(now) {
			return
		}
		if nextBoundary.IsZero() || candidate.Before(nextBoundary) {
			nextBoundary = candidate
		}
	}
	for _, source := range versions {
		if source == nil {
			continue
		}
		if source.EffectiveTo != nil && source.EffectiveTo.Before(cutoff) {
			continue
		}
		clone := *source
		filtered = append(filtered, &clone)
		addBoundary(source.EffectiveFrom)
		if source.EffectiveTo != nil {
			addBoundary(*source.EffectiveTo)
			addBoundary(source.EffectiveTo.Add(matchableVersionHistory))
		}
	}
	return filtered, nextBoundary
}

func BuildTaskSnapshot(versions []*TaskVersionSnapshot) *TaskSnapshot {
	next := &TaskSnapshot{
		LoadedAt:  time.Now().UTC(),
		ByDevice:  make(map[uuid.UUID][]*TaskVersionSnapshot),
		ByVersion: make(map[uuid.UUID]*TaskVersionSnapshot),
	}
	allVersions := make([]*TaskVersionSnapshot, 0, len(versions)*2)
	allVersions = append(allVersions, versions...)
	latestDeviceCatalog := make(map[string]*TaskVersionSnapshot)
	deviceCatalogVersions := make(map[string][]*TaskVersionSnapshot)
	now := time.Now().UTC()
	for _, version := range versions {
		if deviceVersion, ok := devicePipelineVersion(version); ok {
			allVersions = append(allVersions, deviceVersion)
			technology := strings.ToLower(version.Technology)
			deviceCatalogVersions[technology] = append(
				deviceCatalogVersions[technology], version,
			)
			if version.EffectiveFrom.After(now) ||
				(version.EffectiveTo != nil && !now.Before(*version.EffectiveTo)) {
				continue
			}
			latest := latestDeviceCatalog[technology]
			if latest == nil || version.VersionNo > latest.VersionNo ||
				(version.VersionNo == latest.VersionNo &&
					version.EffectiveFrom.After(latest.EffectiveFrom)) {
				latestDeviceCatalog[technology] = version
			}
		}
	}
	for _, version := range allVersions {
		version.DimensionMemberCounts = buildDimensionMemberCounts(version.Members)
		next.ByVersion[version.VersionID] = version
		for deviceID := range version.Members {
			next.ByDevice[deviceID] = append(next.ByDevice[deviceID], version)
		}
	}
	for technology, source := range latestDeviceCatalog {
		if rollup, ok := deviceRollupVersion(source); ok {
			// Keep the composable Counter superset for the complete 35-day
			// matchable-version horizon. A catalog edit can change the current
			// KPI outputs, but it must never discard Counter state already
			// accumulated by an active day/week/month window.
			rollup.Counters = make(map[string]CounterRule)
			for _, catalogVersion := range deviceCatalogVersions[technology] {
				for path, rule := range catalogVersion.Counters {
					rollup.Counters[path] = rule
				}
			}
			next.ByVersion[rollup.VersionID] = rollup
		}
	}
	return next
}

func buildDimensionMemberCounts(
	members map[uuid.UUID][]TaskMember,
) map[string]int64 {
	counts := make(map[string]int64)
	for _, deviceMembers := range members {
		seen := make(map[string]struct{}, len(deviceMembers))
		for _, member := range deviceMembers {
			if _, ok := seen[member.DimensionKey]; ok {
				continue
			}
			seen[member.DimensionKey] = struct{}{}
			counts[member.DimensionKey]++
		}
	}
	return counts
}

func (s *SnapshotStore) Current() *TaskSnapshot {
	return s.value.Load()
}

func (s *SnapshotStore) RunRefresh(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Reload(ctx); err != nil {
				s.logger.Warn("reload PM aggregation task snapshot", zap.Error(err))
			}
		}
	}
}
