package stream

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MatchableLoader interface {
	LoadMatchable(ctx context.Context, at time.Time) ([]*TaskVersionSnapshot, error)
}

type TaskSnapshot struct {
	LoadedAt time.Time
	ByDevice map[uuid.UUID][]*TaskVersionSnapshot
}

type SnapshotStore struct {
	loader MatchableLoader
	value  atomic.Pointer[TaskSnapshot]
	logger *zap.Logger
}

func NewSnapshotStore(loader MatchableLoader, logger *zap.Logger) *SnapshotStore {
	if logger == nil {
		logger = zap.NewNop()
	}
	store := &SnapshotStore{loader: loader, logger: logger}
	store.value.Store(&TaskSnapshot{LoadedAt: time.Now().UTC(), ByDevice: map[uuid.UUID][]*TaskVersionSnapshot{}})
	return store
}

func (s *SnapshotStore) Reload(ctx context.Context) error {
	now := time.Now().UTC()
	versions, err := s.loader.LoadMatchable(ctx, now)
	if err != nil {
		return err
	}
	next := BuildTaskSnapshot(versions)
	next.LoadedAt = now
	s.value.Store(next)
	return nil
}

func BuildTaskSnapshot(versions []*TaskVersionSnapshot) *TaskSnapshot {
	next := &TaskSnapshot{
		LoadedAt: time.Now().UTC(),
		ByDevice: make(map[uuid.UUID][]*TaskVersionSnapshot),
	}
	for _, version := range versions {
		for deviceID := range version.Members {
			next.ByDevice[deviceID] = append(next.ByDevice[deviceID], version)
		}
	}
	return next
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
