package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// IPGuard 防止某个 IP 在短时间内反复试错密码 — 对应 SecuritySettings.tsx
// "IP 限流"区块的 3 个字段：
//
//   limitMinus  → 滑动窗口分钟数（窗口 TTL）
//   limitCount  → 窗口内最大失败次数（超过即锁）
//   limitTimes  → IP 锁定时长分钟（黑名单 TTL）
//
// 与旧 in-memory token-bucket 区别：
//   1. 跨进程一致（Redis 实现，多 app 实例不会各自计数）
//   2. 触发后真正"锁住" — 黑名单 TTL 内拒绝所有该 IP 请求
//   3. 只对失败请求计数（成功登录不计）— 与 FE 文案"输错密码或用户名 N 次"对齐
//
// 阈值与时长全部从 SecurityPolicy 读取（30s 缓存），FE 改配置最多 30s 生效。
// 未注入 policy 时退化到 default。
type IPGuard struct {
	redis  redis.UniversalClient
	policy *SecurityPolicy // 可选；nil 时走 default
}

// NewIPGuard creates a new IPGuard.
func NewIPGuard(client redis.UniversalClient) *IPGuard {
	return &IPGuard{redis: client}
}

// SetPolicy 注入共享 SecurityPolicy 启用动态阈值。
func (g *IPGuard) SetPolicy(p *SecurityPolicy) {
	g.policy = p
}

const (
	ipFailKeyPrefix = "auth:ip:fail:"   // INCR 计数器，TTL=limitMinus
	ipLockKeyPrefix = "auth:ip:locked:" // 黑名单，存在即锁
)

// ErrIPLocked 表示 IP 当前在黑名单内。handler 应返 429 + retry-after。
var ErrIPLocked = errors.New("ip locked due to too many failed attempts")

// CheckAllowed 在登录请求**入口**调用。返：
//   - allowed=false → IP 已被锁，调用方应直接拒绝（429）
//   - remaining > 0 → 距离解锁还剩多少秒（用于 Retry-After header）
func (g *IPGuard) CheckAllowed(ctx context.Context, ip string) (allowed bool, remaining time.Duration, err error) {
	if g == nil || g.redis == nil {
		return true, 0, nil
	}
	key := ipLockKeyPrefix + ip
	ttl, err := g.redis.TTL(ctx, key).Result()
	if err != nil {
		// 错误降级为 allow — 防止 redis 抖动导致全站登录瘫痪。失败次数在日志可见。
		return true, 0, nil
	}
	// TTL = -2 (key 不存在) → allow / TTL = -1 (无 TTL，理论不该出现) → allow
	// TTL > 0 → locked
	if ttl > 0 {
		return false, ttl, nil
	}
	return true, 0, nil
}

// RecordFailure 在登录失败时调用：失败计数器 +1。若达到阈值，把 IP 写入黑名单
// TTL=limitTimes。所有阈值/时长来自 SecurityPolicy。
//
// 调用方应在收到 ErrInvalidCredentials / ErrUserLocked 等业务错误时调用本方法。
// （成功登录不应调用本方法，否则会污染计数器。）
func (g *IPGuard) RecordFailure(ctx context.Context, ip string) error {
	if g == nil || g.redis == nil {
		return nil
	}
	policy := g.snapshot(ctx)

	failKey := ipFailKeyPrefix + ip
	count, err := g.redis.Incr(ctx, failKey).Result()
	if err != nil {
		return fmt.Errorf("ip fail incr: %w", err)
	}
	// 每次失败都刷新窗口 TTL（让恶意 IP 必须停 limitMinus 分钟才能清零）
	g.redis.Expire(ctx, failKey, policy.IPLimitWindow)

	// 触发锁
	if count >= policy.IPLimitCount {
		lockKey := ipLockKeyPrefix + ip
		if err := g.redis.Set(ctx, lockKey, "1", policy.IPLockMinutes).Err(); err != nil {
			return fmt.Errorf("ip lock set: %w", err)
		}
		// 失败计数器在锁定后无意义，删掉以便锁定窗口结束后从 0 重新开始
		g.redis.Del(ctx, failKey)
	}
	return nil
}

// Reset 在登录成功时调用：清零该 IP 失败计数器（黑名单内的不动 — 黑名单只有 TTL 到期才解锁）。
func (g *IPGuard) Reset(ctx context.Context, ip string) {
	if g == nil || g.redis == nil {
		return
	}
	g.redis.Del(ctx, ipFailKeyPrefix+ip)
}

func (g *IPGuard) snapshot(ctx context.Context) *securityPolicyValues {
	if g.policy == nil {
		return defaultPolicy()
	}
	return g.policy.Get(ctx)
}
