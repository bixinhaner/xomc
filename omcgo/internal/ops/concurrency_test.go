package ops

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrencyLimiter_AcquireRelease(t *testing.T) {
	l := NewConcurrencyLimiter(2, 100)

	require.NoError(t, l.AcquireTask(context.Background()))
	require.NoError(t, l.AcquireTask(context.Background()))
	assert.Equal(t, 2, l.CurrentTaskCount())

	// 第三次 acquire 应阻塞；用短 timeout ctx 验
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := l.AcquireTask(ctx)
	require.Error(t, err, "达 max=2 时 AcquireTask 应阻塞直至 ctx 超时")

	l.ReleaseTask()
	require.NoError(t, l.AcquireTask(context.Background()), "释放后能再 acquire")
	l.ReleaseTask()
	l.ReleaseTask()
}

func TestConcurrencyLimiter_DefaultsWhenZero(t *testing.T) {
	l := NewConcurrencyLimiter(0, 0)
	// 默认 maxTasks=20，acquire 20 次应不阻塞
	for i := 0; i < DefaultMaxConcurrentTasks; i++ {
		require.NoError(t, l.AcquireTask(context.Background()))
	}
	assert.Equal(t, DefaultMaxConcurrentTasks, l.CurrentTaskCount())
}

func TestConcurrencyLimiter_WaitRPC_RatesLimit(t *testing.T) {
	// rpcPerSec=10 → burst=10 → 10 次立即放行，第 11 次须等 ~100ms
	l := NewConcurrencyLimiter(0, 10)
	for i := 0; i < 10; i++ {
		require.NoError(t, l.WaitRPC(context.Background()))
	}
	// 第 11 次：用短 timeout 验是否被节流（rate.Limiter Wait 在 burst 用尽后须等令牌补充）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := l.WaitRPC(ctx)
	require.Error(t, err, "burst 用尽后第 11 次 WaitRPC 30ms 内应被节流")
}

func TestConcurrencyLimiter_ConcurrentAcquireRelease(t *testing.T) {
	l := NewConcurrencyLimiter(3, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, l.AcquireTask(context.Background()))
			time.Sleep(10 * time.Millisecond)
			l.ReleaseTask()
		}()
	}
	wg.Wait()
	assert.Equal(t, 0, l.CurrentTaskCount(), "并发 acquire/release 后槽位归零")
}
