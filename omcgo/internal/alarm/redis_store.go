package alarm

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisAlarmStore provides Redis-based L1 cache for active alarm deduplication.
type RedisAlarmStore struct {
	client redis.UniversalClient
}

// NewRedisAlarmStore creates a new Redis-backed alarm store for dedup.
func NewRedisAlarmStore(client redis.UniversalClient) *RedisAlarmStore {
	return &RedisAlarmStore{client: client}
}

func alarmKey(deviceSN string) string {
	return fmt.Sprintf("alarm:active:%s", deviceSN)
}

// Exists checks if an active alarm with the given code exists for the device.
func (s *RedisAlarmStore) Exists(ctx context.Context, deviceSN, alarmCode string) (bool, error) {
	return s.client.HExists(ctx, alarmKey(deviceSN), alarmCode).Result()
}

// Set records an active alarm in Redis.
func (s *RedisAlarmStore) Set(ctx context.Context, deviceSN, alarmCode, alarmID string) error {
	return s.client.HSet(ctx, alarmKey(deviceSN), alarmCode, alarmID).Err()
}

// Get returns the alarm ID for a given device and alarm code.
func (s *RedisAlarmStore) Get(ctx context.Context, deviceSN, alarmCode string) (string, error) {
	val, err := s.client.HGet(ctx, alarmKey(deviceSN), alarmCode).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Delete removes an active alarm entry from Redis.
func (s *RedisAlarmStore) Delete(ctx context.Context, deviceSN, alarmCode string) error {
	return s.client.HDel(ctx, alarmKey(deviceSN), alarmCode).Err()
}

// GetAll returns all active alarm codes and IDs for a device.
func (s *RedisAlarmStore) GetAll(ctx context.Context, deviceSN string) (map[string]string, error) {
	return s.client.HGetAll(ctx, alarmKey(deviceSN)).Result()
}
