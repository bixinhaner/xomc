package acs

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

// DeviceSessionStore 跟踪每个设备「当前活跃会话」的指针（deviceSN → sessionID）。
//
// issue #65（Option B）：取代进程内 deviceSessions sync.Map。在多实例无亲和部署下，
// 设备 A 实例上未完成的孤儿会话，可在下一个 Inform 落到 B 实例时被读到并跨实例清理。
//
// 语义：
//   - Swap：原子地把设备指针换成新 sessionID，返回被顶替的旧 sessionID（若有且不同）。
//     调用方据此跨实例清理旧会话（删 SessionStore + 释放准入槽位 + postSessionWake）。
//   - CompareAndDelete：仅当指针仍等于本会话 ID 时删除，避免误删已被新 Inform 覆盖的指针。
type DeviceSessionStore interface {
	// Swap 把设备的当前会话指针设为 newSessionID，返回此前的 sessionID。
	// 若此前为空或与 newSessionID 相同，返回 ""（无需清理）。
	Swap(ctx context.Context, deviceSN, newSessionID string) (oldSessionID string, err error)
	// CompareAndDelete 仅当当前指针 == sessionID 时删除（CAS 删除，避免误删）。
	CompareAndDelete(ctx context.Context, deviceSN, sessionID string) error
	// Get 读取设备当前会话指针（空表示无活跃会话）。
	Get(ctx context.Context, deviceSN string) (string, error)
}

// RedisDeviceSessionStore 用 Redis STRING + TTL 实现 DeviceSessionStore。
type RedisDeviceSessionStore struct {
	rdb redis.UniversalClient
	ttl time.Duration
}

// NewRedisDeviceSessionStore 创建 Redis 后端的设备会话指针存储。
// ttl 应 ≥ 会话 TTL（5min）+ 余量，使正常会话期间指针不会过早过期。
func NewRedisDeviceSessionStore(rdb redis.UniversalClient, ttl time.Duration) *RedisDeviceSessionStore {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &RedisDeviceSessionStore{rdb: rdb, ttl: ttl}
}

// swapScript 原子地读旧值并写新值，返回旧值（无则空串）。
// KEYS[1]=device session key；ARGV[1]=newSessionID；ARGV[2]=ttl 秒。
var swapScript = redis.NewScript(`
local old = redis.call('GET', KEYS[1])
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[2])
if old then return old end
return ''
`)

// Swap 见接口注释。
func (s *RedisDeviceSessionStore) Swap(ctx context.Context, deviceSN, newSessionID string) (string, error) {
	key := redisx.Keys.ACSDeviceSession(deviceSN)
	old, err := swapScript.Run(ctx, s.rdb, []string{key},
		newSessionID, int64(s.ttl.Seconds())).Text()
	if err != nil {
		return "", fmt.Errorf("device session swap: %w", err)
	}
	if old == "" || old == newSessionID {
		return "", nil
	}
	return old, nil
}

// casDeleteScript 仅当 GET == ARGV[1] 时 DEL，返回删除条数（0/1）。
// KEYS[1]=device session key；ARGV[1]=expected sessionID。
var casDeleteScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`)

// CompareAndDelete 见接口注释。
func (s *RedisDeviceSessionStore) CompareAndDelete(ctx context.Context, deviceSN, sessionID string) error {
	key := redisx.Keys.ACSDeviceSession(deviceSN)
	if err := casDeleteScript.Run(ctx, s.rdb, []string{key}, sessionID).Err(); err != nil {
		return fmt.Errorf("device session compare-and-delete: %w", err)
	}
	return nil
}

// Get 见接口注释。
func (s *RedisDeviceSessionStore) Get(ctx context.Context, deviceSN string) (string, error) {
	v, err := s.rdb.Get(ctx, redisx.Keys.ACSDeviceSession(deviceSN)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("device session get: %w", err)
	}
	return v, nil
}
