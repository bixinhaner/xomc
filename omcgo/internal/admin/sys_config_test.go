package admin

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSysConfigRepo 是 SysConfigRepository 的 in-memory mock，只实现本测试需要的方法。
type stubSysConfigRepo struct {
	mu sync.Mutex

	batchUpsertFn func(ctx context.Context, category string, items []BatchItem) (int, error)
}

func (s *stubSysConfigRepo) Create(_ context.Context, _ *SysConfig) error { return nil }
func (s *stubSysConfigRepo) GetByID(_ context.Context, _ uuid.UUID) (*SysConfig, error) {
	return nil, nil
}
func (s *stubSysConfigRepo) GetByKey(_ context.Context, _, _ string) (*SysConfig, error) {
	return nil, nil
}
func (s *stubSysConfigRepo) List(_ context.Context, _ string, _ bool) ([]SysConfig, error) {
	return nil, nil
}
func (s *stubSysConfigRepo) Update(_ context.Context, _ *SysConfig) error { return nil }
func (s *stubSysConfigRepo) Delete(_ context.Context, _ uuid.UUID) error  { return nil }

func (s *stubSysConfigRepo) BatchUpsert(ctx context.Context, category string, items []BatchItem) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.batchUpsertFn != nil {
		return s.batchUpsertFn(ctx, category, items)
	}
	return len(items), nil
}

func TestSysConfigService_BatchUpsert_FiresHooks(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})

	var calls atomic.Int32
	var seenCategory atomic.Value
	svc.RegisterSavedHook(func(_ context.Context, category string) {
		calls.Add(1)
		seenCategory.Store(category)
	})

	n, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "device",
		Items:    []BatchItem{{Key: "periodicSyncEnabled", Value: "true"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, int32(1), calls.Load(), "hook 应被触发 1 次")
	assert.Equal(t, "device", seenCategory.Load())
}

func TestSysConfigService_BatchUpsert_MultipleHooksFireInOrder(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})

	var order []string
	var orderMu sync.Mutex
	for _, name := range []string{"a", "b", "c"} {
		n := name
		svc.RegisterSavedHook(func(_ context.Context, _ string) {
			orderMu.Lock()
			defer orderMu.Unlock()
			order = append(order, n)
		})
	}

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "security",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, order, "hook 按注册顺序串行执行")
}

func TestSysConfigService_BatchUpsert_HookPanicIsIsolated(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})

	var hook2Called atomic.Bool
	svc.RegisterSavedHook(func(_ context.Context, _ string) {
		panic("hook 1 boom")
	})
	svc.RegisterSavedHook(func(_ context.Context, _ string) {
		hook2Called.Store(true)
	})

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "device",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.NoError(t, err, "hook panic 不应外溢到 caller")
	assert.True(t, hook2Called.Load(), "前面 hook panic 不应阻塞后续 hook")
}

func TestSysConfigService_BatchUpsert_RepoErrorSkipsHooks(t *testing.T) {
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, _ []BatchItem) (int, error) {
			return 0, errors.New("pg connection lost")
		},
	}
	svc := NewSysConfigService(repo)

	var hookCalled atomic.Bool
	svc.RegisterSavedHook(func(_ context.Context, _ string) {
		hookCalled.Store(true)
	})

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "device",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.Error(t, err)
	assert.False(t, hookCalled.Load(), "BatchUpsert 失败时 hook 不应触发")
}

func TestSysConfigService_RegisterSavedHook_NilSafe(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})
	svc.RegisterSavedHook(nil) // 不应 panic
	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "device",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.NoError(t, err)
}
