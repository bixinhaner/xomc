package loginpwd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ReplayWindow 控制密文 ts 字段相对当前服务器时间的允许偏差窗口。
// 在该窗口外的密文会被拒绝（即便 nonce 未被使用过），用于防止抓包后长时间重放。
const ReplayWindow = 5 * time.Minute

// ReplayKeyTTL 是 nonce 在 Redis 中保留的时长。必须 ≥ 2 × ReplayWindow，
// 否则一个 nonce 在 ts 仍合法时就被淘汰，攻击者可绕过去重。
const ReplayKeyTTL = 15 * time.Minute

// nonceKeyPrefix 是 Redis 防重放命名空间。
const nonceKeyPrefix = "auth:nonce:"

var (
	// ErrTimestampOutOfRange 表示密文中的 ts 字段超出当前时间 ±ReplayWindow。
	ErrTimestampOutOfRange = errors.New("encrypted payload timestamp out of allowed range")
	// ErrReplayDetected 表示该 nonce 已在 Redis 中存在，视为重放攻击。
	ErrReplayDetected = errors.New("encrypted payload nonce already used (replay detected)")
	// ErrEmptyNonce 表示密文中的 nonce 为空字符串。
	ErrEmptyNonce = errors.New("encrypted payload nonce is empty")
)

// ReplayGuard 校验密文的时间戳与 nonce，防止抓包重放。
//
// 接口（而非具体类型）便于在单元测试中替换为内存实现。
type ReplayGuard interface {
	// Check 校验 ts 在窗口内、且 nonce 在过去 ReplayKeyTTL 内未出现过。
	// 校验通过会立即"消费" nonce（占位写 Redis），后续相同 nonce 会被拒绝。
	Check(ctx context.Context, ts int64, nonce string) error
}

// RedisReplayGuard 是 ReplayGuard 的 Redis 实现。
type RedisReplayGuard struct {
	client redis.UniversalClient
	now    func() time.Time // injectable clock for tests
}

// NewRedisReplayGuard 构造基于 redis SETNX 的防重放守卫。
func NewRedisReplayGuard(client redis.UniversalClient) *RedisReplayGuard {
	return &RedisReplayGuard{client: client, now: time.Now}
}

// Check 实现 ReplayGuard。
func (g *RedisReplayGuard) Check(ctx context.Context, ts int64, nonce string) error {
	if nonce == "" {
		return ErrEmptyNonce
	}

	now := g.now().Unix()
	delta := now - ts
	if delta < 0 {
		delta = -delta
	}
	if delta > int64(ReplayWindow.Seconds()) {
		return fmt.Errorf("%w: delta=%ds", ErrTimestampOutOfRange, delta)
	}

	ok, err := g.client.SetNX(ctx, nonceKeyPrefix+nonce, "1", ReplayKeyTTL).Result()
	if err != nil {
		return fmt.Errorf("redis setnx for nonce: %w", err)
	}
	if !ok {
		return ErrReplayDetected
	}
	return nil
}
