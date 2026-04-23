package provision

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

// SyncPlanStore persists two-phase sync plan state per device in Redis.
// Previously the plan state was smuggled through the command queue as a
// fake command with method "__sync_plan__" and Priority=-1; that hack
// required the queue to support Peek/Clear semantics. Now the plan is
// a plain Redis STRING with TTL.
type SyncPlanStore struct {
	client redis.UniversalClient
	ttl    time.Duration
}

const defaultSyncPlanTTL = time.Hour

// NewSyncPlanStore creates a new SyncPlanStore.
func NewSyncPlanStore(client redis.UniversalClient) *SyncPlanStore {
	return &SyncPlanStore{client: client, ttl: defaultSyncPlanTTL}
}

// WithTTL overrides the default plan TTL.
func (s *SyncPlanStore) WithTTL(ttl time.Duration) *SyncPlanStore {
	s.ttl = ttl
	return s
}

func (s *SyncPlanStore) key(deviceSN string) string {
	return redisx.Keys.ProvisionSyncPlan(deviceSN)
}

// Save stores the serialized plan state for a device.
func (s *SyncPlanStore) Save(ctx context.Context, deviceSN string, data []byte) error {
	if err := s.client.Set(ctx, s.key(deviceSN), data, s.ttl).Err(); err != nil {
		return fmt.Errorf("save sync plan: %w", err)
	}
	return nil
}

// Load returns the serialized plan state, or nil if missing.
func (s *SyncPlanStore) Load(ctx context.Context, deviceSN string) []byte {
	data, err := s.client.Get(ctx, s.key(deviceSN)).Bytes()
	if err != nil {
		return nil
	}
	return data
}

// Clear removes the plan state for a device.
func (s *SyncPlanStore) Clear(ctx context.Context, deviceSN string) error {
	if err := s.client.Del(ctx, s.key(deviceSN)).Err(); err != nil {
		return fmt.Errorf("clear sync plan: %w", err)
	}
	return nil
}
