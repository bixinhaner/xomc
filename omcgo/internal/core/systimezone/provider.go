// Package systimezone 提供"读取当前系统时区"的统一入口（issue #456，子单 A）。
//
// 系统时区唯一源存于 sys_configs（category='basic', key='timezoneCode'），
// 默认种子值 'UTC'。app 响应层（子单 B）与 worker 聚合（子单 C）共用本 Provider，
// 避免两进程各自解析、口径漂移。
//
// 设计要点：
//   - 带 TTL 缓存：避免每次取时区都打 DB；
//   - 失效刷新：配置页保存后可调用 Invalidate() 强制下次重读；
//   - 解析失败回落 UTC：空值 / 非法 IANA 名一律安全回落 time.UTC，不 panic、不阻断业务。
package systimezone

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SysConfig 中系统时区的固定坐标。
const (
	// Category 系统时区所在的配置分类。
	Category = "basic"
	// Key 系统时区在 sys_configs 中的键名。
	Key = "timezoneCode"
	// DefaultTimezone 默认时区（与 goose 种子保持一致）。
	DefaultTimezone = "UTC"
	// defaultTTL 缓存有效期：在无显式失效刷新时，最长容忍这么久的口径滞后。
	defaultTTL = 5 * time.Minute
)

// Fetcher 从配置源读取指定 (category,key) 的原始字符串值。
// 返回 (value, true) 表示读到；(_, false) 表示无此配置项或读取失败。
// 解耦设计：Provider 不直接依赖 admin 包，由调用方注入读取闭包，
// 既可包 PgSysConfigRepository（app/worker 生产），也便于单测注入桩。
type Fetcher func(ctx context.Context, category, key string) (string, bool)

// Provider 统一的系统时区读取入口（带缓存 + 失效刷新）。
type Provider struct {
	fetch  Fetcher
	ttl    time.Duration
	logger *zap.Logger

	mu       sync.Mutex
	cached   *time.Location
	cachedAt time.Time
}

// Option 可选配置项。
type Option func(*Provider)

// WithTTL 自定义缓存有效期（默认 5 分钟）。
func WithTTL(ttl time.Duration) Option {
	return func(p *Provider) {
		if ttl > 0 {
			p.ttl = ttl
		}
	}
}

// New 构造 Provider。fetch 必填；logger 为 nil 时用 Nop。
func New(fetch Fetcher, logger *zap.Logger, opts ...Option) *Provider {
	if logger == nil {
		logger = zap.NewNop()
	}
	p := &Provider{
		fetch:  fetch,
		ttl:    defaultTTL,
		logger: logger,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Location 返回当前系统时区（解析好的 *time.Location）。
// 命中未过期缓存时直接返回；否则从配置源重读并解析。
// 任何失败（读不到 / 空值 / 非法 IANA 名）一律回落 time.UTC，永不返回 nil。
func (p *Provider) Location(ctx context.Context) *time.Location {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cached != nil && time.Since(p.cachedAt) < p.ttl {
		return p.cached
	}

	loc := p.resolve(ctx)
	p.cached = loc
	p.cachedAt = time.Now()
	return loc
}

// Invalidate 主动失效缓存，使下次 Location 重新从配置源读取。
// 供配置页保存系统时区后调用（接通刷新放在后续子单 / 调用方）。
func (p *Provider) Invalidate() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cached = nil
	p.cachedAt = time.Time{}
}

// resolve 从配置源读取并解析时区，失败回落 UTC。不加锁（调用方持锁）。
func (p *Provider) resolve(ctx context.Context) *time.Location {
	raw, ok := p.fetch(ctx, Category, Key)
	tz := strings.TrimSpace(raw)
	if !ok || tz == "" {
		// 读不到或空值：用默认 UTC（与种子一致）。
		return resolveName(DefaultTimezone, p.logger)
	}
	return resolveName(tz, p.logger)
}

// resolveName 把时区名解析为 *time.Location；空 / 非法一律回落 time.UTC。
func resolveName(tz string, logger *zap.Logger) *time.Location {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		logger.Warn("load system timezone failed; fall back to UTC",
			zap.String("timezone", tz),
			zap.Error(err),
		)
		return time.UTC
	}
	return loc
}
