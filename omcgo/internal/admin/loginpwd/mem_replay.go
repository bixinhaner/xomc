package loginpwd

import (
	"context"
	"sync"
	"time"
)

// MemReplayGuard 是 ReplayGuard 的内存实现，仅供单元测试使用。
//
// 不要在生产路径里使用 —— 内存状态进程级唯一，多实例间不共享，无法防御跨实例重放。
type MemReplayGuard struct {
	mu         sync.Mutex
	seen       map[string]struct{}
	now        func() time.Time
	maxSkewSec int64
}

// NewMemReplayGuard 构造一个内存 ReplayGuard。
// now 可注入以让测试控制"当前时间"，传 nil 走 time.Now。
func NewMemReplayGuard(now func() time.Time) *MemReplayGuard {
	if now == nil {
		now = time.Now
	}
	return &MemReplayGuard{
		seen:       make(map[string]struct{}),
		now:        now,
		maxSkewSec: int64(ReplayWindow.Seconds()),
	}
}

// Check 实现 ReplayGuard。
func (m *MemReplayGuard) Check(_ context.Context, ts int64, nonce string) error {
	if nonce == "" {
		return ErrEmptyNonce
	}
	delta := m.now().Unix() - ts
	if delta < 0 {
		delta = -delta
	}
	if delta > m.maxSkewSec {
		return ErrTimestampOutOfRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, used := m.seen[nonce]; used {
		return ErrReplayDetected
	}
	m.seen[nonce] = struct{}{}
	return nil
}
