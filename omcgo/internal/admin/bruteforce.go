package admin

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

const (
	failedKeyTTL = 30 * time.Minute
)

// 兼容旧测试用例的常量别名（与 SecurityPolicy default 字段保持一致）。
const (
	captchaThreshold = defaultCaptchaThreshold
	lockThreshold    = defaultLockThreshold
	lockDuration     = defaultLockDuration
)

// SysConfigQuerier 消费者驱动小接口：按 (category, key) 读 sys_configs 行。
// admin.PgSysConfigRepository.GetByKey 天然实现。
//
// 同时被 SecurityPolicy (security_policy.go) 复用 — 所有"安全设置"相关特性
// 共享同一个查询接口与同一份 30s 缓存。
type SysConfigQuerier interface {
	GetByKey(ctx context.Context, category, key string) (*SysConfig, error)
}

// LoginGuard tracks failed login attempts and enforces brute-force protections.
//
// 阈值与锁定时长来自 SecurityPolicy（30s 缓存），FE 字段映射：
//   - sumTimes     ↔ LockThreshold
//   - unlockMinu   ↔ LockDuration
//   - attemptTimes ↔ CaptchaThreshold
//   - verifyEnable ↔ VerifyEnable（图形验证码总开关）
//
// 未注入 policy 时所有阈值退化到包级 default 常量（fail-safe）。
type LoginGuard struct {
	redis  redis.UniversalClient
	policy *SecurityPolicy // 可选；未注入时返 default
}

// NewLoginGuard creates a new LoginGuard.
// Policy 通过 SetPolicy / SetSysConfigQuerier 注入。
func NewLoginGuard(client redis.UniversalClient) *LoginGuard {
	return &LoginGuard{redis: client}
}

// SetPolicy 注入共享 SecurityPolicy 实例（推荐：多个特性共享一份缓存）。
func (g *LoginGuard) SetPolicy(p *SecurityPolicy) {
	g.policy = p
}

// SetSysConfigQuerier 旧 API（向后兼容）：自动用 querier 构造 SecurityPolicy。
// 推荐改用 SetPolicy 复用全局 policy。
func (g *LoginGuard) SetSysConfigQuerier(q SysConfigQuerier) {
	g.policy = NewSecurityPolicy(q)
}

// RecordFailure increments the failed attempt counter for a username.
func (g *LoginGuard) RecordFailure(ctx context.Context, username string) (int64, error) {
	key := redisx.Keys.AuthFailed(username)
	count, err := g.redis.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("increment failed count: %w", err)
	}
	g.redis.Expire(ctx, key, failedKeyTTL)
	return count, nil
}

// GetFailedCount returns the current failed attempt count for a username.
func (g *LoginGuard) GetFailedCount(ctx context.Context, username string) int64 {
	count, err := g.redis.Get(ctx, redisx.Keys.AuthFailed(username)).Int64()
	if err != nil {
		return 0
	}
	return count
}

// Reset clears the failed attempt counter on successful login.
func (g *LoginGuard) Reset(ctx context.Context, username string) {
	g.redis.Del(ctx, redisx.Keys.AuthFailed(username))
}

// RequiresCaptcha returns true if captcha is enabled by the operator AND the
// user has exceeded the CAPTCHA threshold. verifyEnable=false 时永远返 false，
// 哪怕失败次数已达阈值 — 让 operator 完全掌控开关。
func (g *LoginGuard) RequiresCaptcha(ctx context.Context, username string) bool {
	p := g.snapshot(ctx)
	if !p.VerifyEnable {
		return false
	}
	return g.GetFailedCount(ctx, username) >= p.CaptchaThreshold
}

// ShouldLock returns true if the failed count has reached the lock threshold.
func (g *LoginGuard) ShouldLock(ctx context.Context, count int64) bool {
	p := g.snapshot(ctx)
	return count >= p.LockThreshold
}

// LockDuration returns the account lock duration (sys_configs security.unlockMinu).
func (g *LoginGuard) LockDuration(ctx context.Context) time.Duration {
	p := g.snapshot(ctx)
	return p.LockDuration
}

// InvalidateConfigCache 让下一次读取重新加载（FE 保存 sys_configs 后可调）。
func (g *LoginGuard) InvalidateConfigCache() {
	if g.policy != nil {
		g.policy.InvalidateCache()
	}
}

func (g *LoginGuard) snapshot(ctx context.Context) *securityPolicyValues {
	if g.policy == nil {
		return defaultPolicy()
	}
	return g.policy.Get(ctx)
}

// readIntConfig 从 sys_configs 读 string value 转 int；错误（key 不存在 /
// 解析失败 / 负数 / 零）都返 0 让 caller 走 default。
// 复用：SecurityPolicy.loadFromSysConfig 也用此辅助。
func readIntConfig(ctx context.Context, q SysConfigQuerier, category, key string) int64 {
	cfg, err := q.GetByKey(ctx, category, key)
	if err != nil || cfg == nil || cfg.Value == "" {
		return 0
	}
	n, err := strconv.ParseInt(cfg.Value, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
