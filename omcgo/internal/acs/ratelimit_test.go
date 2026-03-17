package acs

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestRateLimiter(perMinute, burst, maxDevices int) *DeviceRateLimiter {
	return NewDeviceRateLimiter(perMinute, burst, maxDevices, zap.NewNop())
}

func TestDeviceRateLimiter_Allow(t *testing.T) {
	// 60 per minute = 1/s, burst 5 => first 5 calls should succeed immediately
	rl := newTestRateLimiter(60, 5, 100)

	for i := 0; i < 5; i++ {
		assert.True(t, rl.Allow("device-001"), "request %d should be allowed", i+1)
	}
}

func TestDeviceRateLimiter_Deny(t *testing.T) {
	// burst=2 means only 2 tokens available at start
	rl := newTestRateLimiter(60, 2, 100)

	assert.True(t, rl.Allow("device-002"))
	assert.True(t, rl.Allow("device-002"))
	// Third call should exceed burst
	assert.False(t, rl.Allow("device-002"))
}

func TestDeviceRateLimiter_DifferentDevices(t *testing.T) {
	rl := newTestRateLimiter(60, 2, 100)

	// Exhaust device-A
	assert.True(t, rl.Allow("device-A"))
	assert.True(t, rl.Allow("device-A"))
	assert.False(t, rl.Allow("device-A"))

	// device-B should still have its own independent limit
	assert.True(t, rl.Allow("device-B"))
	assert.True(t, rl.Allow("device-B"))
}

func TestDeviceRateLimiter_Reset(t *testing.T) {
	rl := newTestRateLimiter(60, 2, 100)

	// Exhaust the limiter
	assert.True(t, rl.Allow("device-003"))
	assert.True(t, rl.Allow("device-003"))
	assert.False(t, rl.Allow("device-003"))

	// Reset clears the limiter; a new one is created on next Allow
	rl.Reset("device-003")

	assert.True(t, rl.Allow("device-003"))
	assert.True(t, rl.Allow("device-003"))
}

func TestNewDeviceRateLimiter_Defaults(t *testing.T) {
	// perMinute <= 0 defaults to 10, burst <= 0 defaults to 5, maxDevices <= 0 defaults to 100000
	rl := NewDeviceRateLimiter(0, 0, 0, nil)

	// With default burst=5, first 5 should pass
	for i := 0; i < 5; i++ {
		assert.True(t, rl.Allow("device-default"), "request %d should be allowed", i+1)
	}
	// 6th should be denied (burst exhausted)
	assert.False(t, rl.Allow("device-default"))

	// Also verify negative values are handled
	rl2 := NewDeviceRateLimiter(-1, -1, -1, nil)
	for i := 0; i < 5; i++ {
		assert.True(t, rl2.Allow("device-neg"), "request %d should be allowed", i+1)
	}
	assert.False(t, rl2.Allow("device-neg"))
}

func TestDeviceRateLimiter_DeviceCount(t *testing.T) {
	rl := newTestRateLimiter(60, 2, 100)

	assert.Equal(t, 0, rl.DeviceCount())

	rl.Allow("dev-001")
	assert.Equal(t, 1, rl.DeviceCount())

	rl.Allow("dev-002")
	assert.Equal(t, 2, rl.DeviceCount())

	// 同一设备不增加计数
	rl.Allow("dev-001")
	assert.Equal(t, 2, rl.DeviceCount())

	rl.Reset("dev-001")
	assert.Equal(t, 1, rl.DeviceCount())
}

func TestDeviceRateLimiter_MaxDevices_Eviction(t *testing.T) {
	rl := newTestRateLimiter(60, 2, 3)

	// 填满 3 个设备，按顺序访问确保 LRU 顺序
	rl.Allow("dev-001")
	rl.Allow("dev-002")
	rl.Allow("dev-003")
	assert.Equal(t, 3, rl.DeviceCount())

	// 新设备加入，LRU 淘汰最久未访问的 dev-001
	rl.Allow("dev-004")
	assert.Equal(t, 3, rl.DeviceCount())

	// dev-001 应已被淘汰（使用 Contains 验证，不提升 LRU 顺序）
	assert.False(t, rl.Contains("dev-001"), "dev-001 should have been evicted")
	assert.True(t, rl.Contains("dev-004"), "dev-004 should exist")
}

func TestDeviceRateLimiter_Eviction_RespectsAccessOrder(t *testing.T) {
	rl := newTestRateLimiter(60, 2, 3)

	// 添加 3 个设备
	rl.Allow("dev-001")
	rl.Allow("dev-002")
	rl.Allow("dev-003")

	// 重新访问 dev-001，将其提升到队首
	rl.Allow("dev-001")

	// 新设备加入，应淘汰 LRU 队尾的 dev-002（而非最先添加的 dev-001）
	rl.Allow("dev-004")
	assert.Equal(t, 3, rl.DeviceCount())

	assert.True(t, rl.Contains("dev-001"), "dev-001 was recently accessed, should be kept")
	assert.False(t, rl.Contains("dev-002"), "dev-002 is LRU, should have been evicted")
	assert.True(t, rl.Contains("dev-003"), "dev-003 should exist")
	assert.True(t, rl.Contains("dev-004"), "dev-004 should exist")
}

func TestDeviceRateLimiter_Cleanup(t *testing.T) {
	rl := newTestRateLimiter(60, 2, 100)

	rl.Allow("dev-001")
	rl.Allow("dev-002")
	assert.Equal(t, 2, rl.DeviceCount())

	// timeout=0 表示立即过期
	rl.cleanup(0)
	assert.Equal(t, 0, rl.DeviceCount())
}

func TestDeviceRateLimiter_Cleanup_KeepsActive(t *testing.T) {
	rl := newTestRateLimiter(60, 2, 100)

	rl.Allow("dev-old")
	time.Sleep(50 * time.Millisecond)
	rl.Allow("dev-new")

	// 清理超过 30ms 未访问的 → dev-old 被清理，dev-new 保留
	rl.cleanup(30 * time.Millisecond)
	assert.Equal(t, 1, rl.DeviceCount())
	assert.True(t, rl.Contains("dev-new"), "dev-new should be kept")
	assert.False(t, rl.Contains("dev-old"), "dev-old should have been cleaned up")
}

func TestDeviceRateLimiter_Cleanup_EarlyBreak(t *testing.T) {
	rl := newTestRateLimiter(60, 2, 100)

	// 按时间顺序添加设备
	rl.Allow("dev-old-1")
	rl.Allow("dev-old-2")
	time.Sleep(50 * time.Millisecond)
	rl.Allow("dev-new-1")
	rl.Allow("dev-new-2")

	// 只有 dev-old-* 过期，cleanup 应提前终止
	rl.cleanup(30 * time.Millisecond)
	assert.Equal(t, 2, rl.DeviceCount())
	assert.True(t, rl.Contains("dev-new-1"))
	assert.True(t, rl.Contains("dev-new-2"))
}

func TestDeviceRateLimiter_StartCleanup_And_Stop(t *testing.T) {
	rl := newTestRateLimiter(60, 2, 100)

	rl.Allow("dev-001")
	assert.Equal(t, 1, rl.DeviceCount())

	// 启动快速清理：10ms 间隔，1ms 超时
	rl.StartCleanup(10*time.Millisecond, time.Millisecond)

	// 等待清理生效
	require.Eventually(t, func() bool {
		return rl.DeviceCount() == 0
	}, 200*time.Millisecond, 5*time.Millisecond)

	// Stop 应能安全调用且不 panic
	rl.Stop()
	rl.Stop()
}

func TestDeviceRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := newTestRateLimiter(6000, 100, 1000)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sn := fmt.Sprintf("dev-%03d", id%26)
			for j := 0; j < 100; j++ {
				rl.Allow(sn)
			}
		}(i)
	}
	wg.Wait()

	// 不 panic，且 DeviceCount 合理
	count := rl.DeviceCount()
	assert.True(t, count > 0 && count <= 26, "device count should be between 1 and 26, got %d", count)
}
