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

// ---------------------------------------------------------
// Below methods support the Denormalization -> Redis migration
// for fast GIS map reads without PostgreSQL row-level locks.
// ---------------------------------------------------------

func alarmStatsKey(deviceID string) string {
	return "cache:device:alarm_stats:" + deviceID
}

// IncrementActiveAlarmCount atomicaly increments the active alarm count for a device.
func (s *RedisAlarmStore) IncrementActiveAlarmCount(ctx context.Context, deviceID string) error {
	return s.client.HIncrBy(ctx, alarmStatsKey(deviceID), "active_count", 1).Err()
}

// DecrementActiveAlarmCount atomicaly decrements the active alarm count.
func (s *RedisAlarmStore) DecrementActiveAlarmCount(ctx context.Context, deviceID string) error {
	// Let it drop below 0 momentarily, reconciliation job will fix it.
	return s.client.HIncrBy(ctx, alarmStatsKey(deviceID), "active_count", -1).Err()
}

// GetAlarmCountsPipeline fetches active alarm counts for multiple devices in ONE roundtrip.
func (s *RedisAlarmStore) GetAlarmCountsPipeline(ctx context.Context, deviceIDs []string) (map[string]int, error) {
	if len(deviceIDs) == 0 {
		return nil, nil
	}

	
	pipe := s.client.Pipeline()
	var cmds []*redis.StringCmd
	
	for _, id := range deviceIDs {
		cmds = append(cmds, pipe.HGet(ctx, alarmStatsKey(id), "active_count"))
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		// Pipeline returns redis.Nil if ANY of the keys does not exist. We can ignore it safely.
	}

	
	res := make(map[string]int, len(deviceIDs))
	for i, id := range deviceIDs {
		val, _ := cmds[i].Int() // if error or redis.Nil, val will be 0
		if val < 0 {
			val = 0
		}
		res[id] = val
	}

	
	return res, nil
}
