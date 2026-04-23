package alarm

import (
	"context"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

const alarmKeyTTL = 24 * time.Hour

// RedisAlarmStore provides Redis-based L1 cache for active alarm deduplication.
type RedisAlarmStore struct {
	client redis.UniversalClient
}

// NewRedisAlarmStore creates a new Redis-backed alarm store for dedup.
func NewRedisAlarmStore(client redis.UniversalClient) *RedisAlarmStore {
	return &RedisAlarmStore{client: client}
}

func alarmKey(deviceSN string) string {
	return redisx.Keys.AlarmActive(deviceSN)
}

// Exists checks if an active alarm with the given identifier exists for the device.
func (s *RedisAlarmStore) Exists(ctx context.Context, deviceSN, alarmIdentifier string) (bool, error) {
	return s.client.HExists(ctx, alarmKey(deviceSN), alarmIdentifier).Result()
}

// Set records an active alarm in Redis and refreshes the key TTL.
func (s *RedisAlarmStore) Set(ctx context.Context, deviceSN, alarmIdentifier, alarmID string) error {
	key := alarmKey(deviceSN)
	if err := s.client.HSet(ctx, key, alarmIdentifier, alarmID).Err(); err != nil {
		return err
	}
	return s.client.Expire(ctx, key, alarmKeyTTL).Err()
}

// Get returns the alarm ID for a given device and alarm identifier.
func (s *RedisAlarmStore) Get(ctx context.Context, deviceSN, alarmIdentifier string) (string, error) {
	val, err := s.client.HGet(ctx, alarmKey(deviceSN), alarmIdentifier).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Delete removes an active alarm entry from Redis.
func (s *RedisAlarmStore) Delete(ctx context.Context, deviceSN, alarmIdentifier string) error {
	return s.client.HDel(ctx, alarmKey(deviceSN), alarmIdentifier).Err()
}

// GetAll returns all active alarm codes and IDs for a device.
func (s *RedisAlarmStore) GetAll(ctx context.Context, deviceSN string) (map[string]string, error) {
	return s.client.HGetAll(ctx, alarmKey(deviceSN)).Result()
}
