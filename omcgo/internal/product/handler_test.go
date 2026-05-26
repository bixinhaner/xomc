package product

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// stubCacheInvalidator 实现 DeviceCacheInvalidator（test only）。
type stubCacheInvalidator struct {
	calls  atomic.Int32
	lastSN string
}

func (s *stubCacheInvalidator) Delete(_ context.Context, sn string) {
	s.calls.Add(1)
	s.lastSN = sn
}

// T-0176-PR-D：验证 BindOrphan / RematchOrphan 写 product_id 后清 SN cache
// 的统一入口 invalidateDeviceCacheBySN 的行为契约：
//   - deviceCache nil → no-op
//   - sn 空 → no-op（避免误删根 key）
//   - 都满足 → 调 Delete 一次
//
// BindOrphan 端点本身依赖 *PgRepository，full-stack 集成在 e2e/integration
// 测试覆盖；本单元测试聚焦 cache invalidation 路径。
func Test_InvalidateDeviceCacheBySN_NilCache_NoOp(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, zap.NewNop())
	// deviceCache 未设 — Delete 不应 panic 不应失败
	h.invalidateDeviceCacheBySN(context.Background(), "SN-A")
}

func Test_InvalidateDeviceCacheBySN_EmptySN_NoOp(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, zap.NewNop())
	cache := &stubCacheInvalidator{}
	h.SetDeviceCacheInvalidator(cache)

	h.invalidateDeviceCacheBySN(context.Background(), "")
	assert.Equal(t, int32(0), cache.calls.Load(), "空 SN 时不应触发 Delete（防误删根 key）")
}

func Test_InvalidateDeviceCacheBySN_CallsDelete(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, zap.NewNop())
	cache := &stubCacheInvalidator{}
	h.SetDeviceCacheInvalidator(cache)

	h.invalidateDeviceCacheBySN(context.Background(), "SN-XYZ")
	assert.Equal(t, int32(1), cache.calls.Load(), "Delete 应被调一次")
	assert.Equal(t, "SN-XYZ", cache.lastSN)
}

func Test_SetDeviceCacheInvalidator_NilSafe(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, zap.NewNop())
	// 注入 nil 应等价于禁用，不应 panic
	h.SetDeviceCacheInvalidator(nil)
	h.invalidateDeviceCacheBySN(context.Background(), "SN-NIL")
}
