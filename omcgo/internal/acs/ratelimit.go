package acs

import (
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// DeviceEntry 包装 limiter 和最后访问时间，用于 TTL 清理。
// lastAccess 以 UnixNano 存于 atomic.Int64：entry 指针被所有 ACS 请求
// goroutine 与后台 cleanup 协程共享，普通字段在 LRU 内部锁之外读写会产生 data race。
type DeviceEntry struct {
	limiter    *rate.Limiter
	lastAccess atomic.Int64 // UnixNano
}

// touch 更新最后访问时间为当前时刻。
func (e *DeviceEntry) touch() {
	e.lastAccess.Store(time.Now().UnixNano())
}

// lastAccessTime 返回最后访问时间。
func (e *DeviceEntry) lastAccessTime() time.Time {
	return time.Unix(0, e.lastAccess.Load())
}

// DeviceRateLimiter 基于 token bucket 的设备级限流器。
// 使用 hashicorp/golang-lru 管理设备生命周期，容量满时 O(1) 自动淘汰最久未访问的设备。
type DeviceRateLimiter struct {
	cache  *lru.Cache[string, *DeviceEntry]
	limit  rate.Limit
	burst  int
	logger *zap.Logger
	stopCh chan struct{}
}

// NewDeviceRateLimiter 创建设备级限流器。
//   - perMinute: 每设备每分钟最大请求数
//   - burst: token bucket 突发容量
//   - maxDevices: 最大追踪设备数，容量满时自动淘汰 LRU 设备
func NewDeviceRateLimiter(perMinute, burst, maxDevices int, logger *zap.Logger) *DeviceRateLimiter {
	if perMinute <= 0 {
		perMinute = 10
	}
	if burst <= 0 {
		burst = 5
	}
	if maxDevices <= 0 {
		maxDevices = 100000
	}

	cache, _ := lru.NewWithEvict[string, *DeviceEntry](maxDevices, func(key string, value *DeviceEntry) {
		if logger != nil {
			logger.Debug("rate limiter evicted device",
				zap.String("device_sn", key),
				zap.Time("last_access", value.lastAccessTime()))
		}
	})

	return &DeviceRateLimiter{
		cache:  cache,
		limit:  rate.Limit(float64(perMinute) / 60.0),
		burst:  burst,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Allow 检查设备是否允许通过限流。
// Get 自动将设备提升到 LRU 队首；新设备经 PeekOrAdd 原子插入（LoadOrStore 语义），
// 两个 goroutine 同时遇到新设备时保证只保留一个 limiter，避免 Get-miss-then-Add
// 的覆盖导致已消费 token 被丢弃（burst 泄漏）。容量满时 O(1) 淘汰队尾。
func (rl *DeviceRateLimiter) Allow(deviceSN string) bool {
	if entry, ok := rl.cache.Get(deviceSN); ok {
		entry.touch()
		return entry.limiter.Allow()
	}

	// 新设备：PeekOrAdd 在 LRU 内部锁中完成 check-and-add，
	// 若并发竞争中已有其他 goroutine 插入，则复用已存在的 entry。
	entry := &DeviceEntry{limiter: rate.NewLimiter(rl.limit, rl.burst)}
	entry.touch()
	if previous, ok, _ := rl.cache.PeekOrAdd(deviceSN, entry); ok {
		// 竞争失败：丢弃本地新建的 entry，使用已存在的；
		// PeekOrAdd 不提升 recency，补一次 Get 维持 LRU 访问顺序。
		previous.touch()
		rl.cache.Get(deviceSN)
		return previous.limiter.Allow()
	}
	return entry.limiter.Allow()
}

// StartCleanup 启动后台协程，定期清理超过 timeout 未访问的设备条目。
// 利用 LRU Keys() 返回最旧在前的顺序，遇到未过期条目即提前终止遍历。
// 调用 Stop() 可安全终止清理协程。
func (rl *DeviceRateLimiter) StartCleanup(interval, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.cleanup(timeout)
			case <-rl.stopCh:
				return
			}
		}
	}()
}

// Stop 停止后台清理协程。可安全多次调用。
func (rl *DeviceRateLimiter) Stop() {
	select {
	case <-rl.stopCh:
	default:
		close(rl.stopCh)
	}
}

func (rl *DeviceRateLimiter) cleanup(timeout time.Duration) {
	now := time.Now()
	evicted := 0

	// Keys() 返回从最旧到最新的顺序，可以提前终止
	for _, key := range rl.cache.Keys() {
		entry, ok := rl.cache.Peek(key) // Peek 不提升 LRU 顺序
		if !ok {
			continue
		}
		if now.Sub(entry.lastAccessTime()) > timeout {
			rl.cache.Remove(key)
			evicted++
		} else {
			break // 后面的都更新，无需继续
		}
	}

	if evicted > 0 && rl.logger != nil {
		rl.logger.Info("rate limiter cleanup completed",
			zap.Int("evicted", evicted),
			zap.Int("remaining", rl.cache.Len()))
	}
}

// Reset 移除指定设备的限流条目，下次访问将重新创建。
func (rl *DeviceRateLimiter) Reset(deviceSN string) {
	rl.cache.Remove(deviceSN)
}

// DeviceCount 返回当前追踪的设备数量，用于监控。
func (rl *DeviceRateLimiter) DeviceCount() int {
	return rl.cache.Len()
}

// Contains 检查指定设备是否在限流器中（不提升 LRU 顺序）。
func (rl *DeviceRateLimiter) Contains(deviceSN string) bool {
	return rl.cache.Contains(deviceSN)
}
