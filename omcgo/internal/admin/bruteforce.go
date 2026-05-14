package admin

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

const (
	failedKeyTTL = 30 * time.Minute

	// 默认 fallback —— 当 sys_configs 读不到或值非法时使用。
	// 实际生效阈值/时长来自 sys_configs：
	//   category='security' key='attemptTimes' → CAPTCHA 阈值 (FE 显示验证码)
	//   category='security' key='sumTimes'     → 账号锁定阈值 (FE: 失败 N 次锁账户)
	//   category='security' key='unlockMinu'   → 锁定时长 (分钟) (FE: 锁 N 分钟解锁)
	defaultCaptchaThreshold = 3
	defaultLockThreshold    = 10
	defaultLockDuration     = 30 * time.Minute

	// 配置缓存 TTL：避免每次登录失败都打 sys_configs SELECT，
	// 牺牲 30s 内的配置生效延迟换取性能。FE 改配置后最多等 30s 生效。
	configCacheTTL = 30 * time.Second
)

// 兼容旧 testcase 的常量别名（同名常量被改为带 default 前缀；保留旧符号导出以防回归）。
const (
	captchaThreshold = defaultCaptchaThreshold
	lockThreshold    = defaultLockThreshold
	lockDuration     = defaultLockDuration
)

// SysConfigQuerier 消费者驱动小接口：LoginGuard 只需要按 (category, key) 读 string value
// 的能力，不依赖 admin.SysConfigRepository 整体（避免反向依赖 + 易于 mock）。
// admin.PgSysConfigRepository.GetByKey 天然实现。
type SysConfigQuerier interface {
	GetByKey(ctx context.Context, category, key string) (*SysConfig, error)
}

// cachedConfig 缓存 sys_configs 中三个相关 key 的解码结果。
type cachedConfig struct {
	captchaThreshold int64
	lockThreshold    int64
	lockDuration     time.Duration
	expiresAt        time.Time
}

// LoginGuard tracks failed login attempts and enforces brute-force protections.
type LoginGuard struct {
	redis   redis.UniversalClient
	cfgRepo SysConfigQuerier // 可选；未注入时所有阈值/时长走 default 常量

	cacheMu sync.Mutex
	cache   atomic.Pointer[cachedConfig]
}

// NewLoginGuard creates a new LoginGuard.
// SysConfigQuerier 不在构造函数签名里 — 用 SetSysConfigQuerier 注入，
// 保持构造期无 admin.SysConfigRepository 循环依赖风险。
func NewLoginGuard(client redis.UniversalClient) *LoginGuard {
	return &LoginGuard{redis: client}
}

// SetSysConfigQuerier 注入 sys_configs 查询器。nil 安全：未注入时退化到 default 常量。
func (g *LoginGuard) SetSysConfigQuerier(q SysConfigQuerier) {
	g.cfgRepo = q
}

// RecordFailure increments the failed attempt counter for a username.
func (g *LoginGuard) RecordFailure(ctx context.Context, username string) (int64, error) {
	key := redisx.Keys.AuthFailed(username)
	count, err := g.redis.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("increment failed count: %w", err)
	}
	// Set/refresh TTL on every failure
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

// RequiresCaptcha returns true if the user has exceeded the CAPTCHA threshold.
// 阈值从 sys_configs security.attemptTimes 读，缺省 3。
func (g *LoginGuard) RequiresCaptcha(ctx context.Context, username string) bool {
	cfg := g.loadConfig(ctx)
	return g.GetFailedCount(ctx, username) >= cfg.captchaThreshold
}

// ShouldLock returns true if the failed count has reached the lock threshold.
// 阈值从 sys_configs security.sumTimes 读，缺省 10。
func (g *LoginGuard) ShouldLock(ctx context.Context, count int64) bool {
	cfg := g.loadConfig(ctx)
	return count >= cfg.lockThreshold
}

// LockDuration returns the account lock duration.
// 时长从 sys_configs security.unlockMinu 读（单位：分钟），缺省 30min。
func (g *LoginGuard) LockDuration(ctx context.Context) time.Duration {
	cfg := g.loadConfig(ctx)
	return cfg.lockDuration
}

// loadConfig 取当前生效的三项阈值/时长配置。带 30s in-memory 缓存避免高频登录失败
// 时每次都打 PG。读失败一律退化到 default 常量（fail-safe — 锁定行为永远不会因
// 配置读取失败而消失）。
func (g *LoginGuard) loadConfig(ctx context.Context) *cachedConfig {
	if cached := g.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
		return cached
	}

	// 双重检查防止重复加载：缓存过期时多个 goroutine 同时进来只一个真去 DB。
	g.cacheMu.Lock()
	defer g.cacheMu.Unlock()
	if cached := g.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
		return cached
	}

	cfg := &cachedConfig{
		captchaThreshold: defaultCaptchaThreshold,
		lockThreshold:    defaultLockThreshold,
		lockDuration:     defaultLockDuration,
		expiresAt:        time.Now().Add(configCacheTTL),
	}

	if g.cfgRepo != nil {
		if v := readIntConfig(ctx, g.cfgRepo, "security", "attemptTimes"); v > 0 {
			cfg.captchaThreshold = v
		}
		if v := readIntConfig(ctx, g.cfgRepo, "security", "sumTimes"); v > 0 {
			cfg.lockThreshold = v
		}
		if v := readIntConfig(ctx, g.cfgRepo, "security", "unlockMinu"); v > 0 {
			cfg.lockDuration = time.Duration(v) * time.Minute
		}
	}

	g.cache.Store(cfg)
	return cfg
}

// readIntConfig 从 sys_configs 读 string value 转 int；任何错误（key 不存在 /
// 解析失败 / 负数）都返 0 让 caller 走 default。
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

// InvalidateConfigCache 主动失效缓存（如批量更新 sys_configs 后立刻生效）。
// 当前未在 handler 调；保留 API 为未来 sys_configs Update Hook 用。
func (g *LoginGuard) InvalidateConfigCache() {
	g.cache.Store(nil)
}
