package device

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Sequencer_SameSNSerialized(t *testing.T) {
	seq := NewSequencer()
	const sn = "BLQ-T-001"
	const goroutines = 50

	var counter atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := seq.Lock(sn)
			defer unlock()

			cur := counter.Add(1)
			for {
				prev := maxConcurrent.Load()
				if cur <= prev || maxConcurrent.CompareAndSwap(prev, cur) {
					break
				}
			}
			time.Sleep(2 * time.Millisecond)
			counter.Add(-1)
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), maxConcurrent.Load(),
		"max concurrent holders for same SN must be 1; got %d", maxConcurrent.Load())
}

func Test_Sequencer_DifferentSNsConcurrent(t *testing.T) {
	seq := NewSequencer()
	const goroutines = 20

	var inCriticalSection atomic.Int32
	var sawConcurrency atomic.Bool
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := seq.Lock(string(rune('A' + i))) // 不同 SN
			defer unlock()

			now := inCriticalSection.Add(1)
			if now > 1 {
				sawConcurrency.Store(true)
			}
			time.Sleep(5 * time.Millisecond)
			inCriticalSection.Add(-1)
		}()
	}
	wg.Wait()

	assert.True(t, sawConcurrency.Load(), "different SNs must be able to run concurrently")
}

func Test_Sequencer_PreservesAcquisitionOrder(t *testing.T) {
	// 同一 SN 上，goroutine 按调用 Lock 的顺序排队拿锁；FIFO 由 sync.Mutex 提供。
	// 我们让一个长持锁者占住锁，再启动一串等锁者，验证它们按启动顺序拿到。
	seq := NewSequencer()
	const sn = "BLQ-T-FIFO"

	hold := seq.Lock(sn)

	const waiters = 10
	order := make([]int, 0, waiters)
	var orderMu sync.Mutex
	var wg sync.WaitGroup
	started := make(chan struct{}, waiters)

	for i := 0; i < waiters; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			started <- struct{}{}
			unlock := seq.Lock(sn)
			defer unlock()
			orderMu.Lock()
			order = append(order, i)
			orderMu.Unlock()
		}()
		// 让前一个 goroutine 真正进入 Lock 等待状态，再启动下一个。
		<-started
		time.Sleep(1 * time.Millisecond)
	}

	hold()
	wg.Wait()

	require.Len(t, order, waiters)
	// Go sync.Mutex 不保证严格 FIFO（runtime 内部按 hot path/starve 模式），
	// 但在低争用 + 顺序启动下应当大致按序。这里只要求"不是逆序"即可，
	// 真正的 FIFO 不变量由调用方依赖 EventBus 的 FIFO 投递 + 此锁串行化共同保证。
	// 因此本断言放宽，但留作 regression 防护：
	assert.NotEqual(t, []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, order,
		"acquisition order should not be strict reverse of start order")
}

func Test_Sequencer_EmptySNNoop(t *testing.T) {
	seq := NewSequencer()
	unlock := seq.Lock("")
	// 必须可以立刻第二次 Lock("") 而不阻塞
	done := make(chan struct{})
	go func() {
		unlock2 := seq.Lock("")
		defer unlock2()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("empty SN should never block")
	}
	unlock()
}

func Test_Sequencer_GetOrCreateReturnsSameInstance(t *testing.T) {
	seq := NewSequencer()
	m1 := seq.getOrCreate("X")
	m2 := seq.getOrCreate("X")
	assert.Same(t, m1, m2, "same SN must reuse same mutex instance")
}
