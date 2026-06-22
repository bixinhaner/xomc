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

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
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

// ── Validator hook（issue #548 切片 2 D 后端 sys_configs validator）─────────────────

// TestSysConfigService_RegisterValidator_RejectsInvalidValue：
// validator 返错时 BatchUpsert 应整批不落库，且错误链路含 ErrInvalidInput（handler 翻 400）。
func TestSysConfigService_RegisterValidator_RejectsInvalidValue(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)

	var hookCalled atomic.Bool
	svc.RegisterSavedHook(func(_ context.Context, _ string) {
		hookCalled.Store(true)
	})

	svc.RegisterValidator("storage", "minio_public_endpoint", func(v string) error {
		if v == "" {
			return nil
		}
		// 模拟"禁带 scheme"语义
		if len(v) >= 7 && v[:7] == "http://" {
			return errors.New("scheme not allowed")
		}
		return nil
	})

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items: []BatchItem{
			{Key: "minio_public_endpoint", Value: "http://bad"},
		},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput),
		"validator 错应包 ErrInvalidInput 让 handler 翻 400，实际: %v", err)
	assert.False(t, repoCalled.Load(), "validator 失败时 repo.BatchUpsert 不应被调用（整批不落库）")
	assert.False(t, hookCalled.Load(), "validator 失败时 SavedHook 不应触发")
}

// TestSysConfigService_RegisterValidator_PassesValidValue：合法值放行 + repo + hook 都跑。
func TestSysConfigService_RegisterValidator_PassesValidValue(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)
	var hookCalled atomic.Bool
	svc.RegisterSavedHook(func(_ context.Context, _ string) { hookCalled.Store(true) })

	svc.RegisterValidator("storage", "minio_public_endpoint", func(_ string) error { return nil })

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items:    []BatchItem{{Key: "minio_public_endpoint", Value: "ok-host:9000"}},
	})
	require.NoError(t, err)
	assert.True(t, repoCalled.Load(), "validator 通过后 repo.BatchUpsert 应被调用")
	assert.True(t, hookCalled.Load(), "validator 通过 + repo 成功 → SavedHook 应触发")
}

// TestSysConfigService_RegisterValidator_OnlyMatchesRegisteredKey：未注册 key 通用 KV 直通。
func TestSysConfigService_RegisterValidator_OnlyMatchesRegisteredKey(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})
	called := atomic.Bool{}
	svc.RegisterValidator("storage", "minio_public_endpoint", func(_ string) error {
		called.Store(true)
		return errors.New("should not be invoked")
	})

	// 同 category 但不同 key → 不应触发该 validator
	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items:    []BatchItem{{Key: "other_key", Value: "anything"}},
	})
	require.NoError(t, err)
	assert.False(t, called.Load())

	// 不同 category 同 key → 也不应触发
	_, err = svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "other_category",
		Items:    []BatchItem{{Key: "minio_public_endpoint", Value: "anything"}},
	})
	require.NoError(t, err)
	assert.False(t, called.Load())
}

// TestSysConfigService_RegisterValidator_NilFnDeletes：传 nil 等价删除该 key 的 validator。
func TestSysConfigService_RegisterValidator_NilFnDeletes(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})
	svc.RegisterValidator("storage", "k", func(_ string) error { return errors.New("reject") })

	// 先确认会拒绝
	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.Error(t, err)

	// 注册 nil → 删除
	svc.RegisterValidator("storage", "k", nil)
	_, err = svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.NoError(t, err)
}
