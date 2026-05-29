package parammodel

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestAcquireFileLock_PerName 验证 T-0178 §9.5 的 per-filename 互斥语义:
//   - 不同文件名锁互不影响,并发持有不阻塞
//   - 同文件名锁严格串行,后到者等前者释放
func TestAcquireFileLock_PerName(t *testing.T) {
	h := &Handler{} // 仅用 fileLocks 字段,其余依赖不需要

	// 1. 同名串行:第一把锁占住,第二把必须等
	var (
		entered  atomic.Bool
		released atomic.Bool
	)
	unlock1 := h.acquireFileLock("CBQQ.xml")
	go func() {
		unlock2 := h.acquireFileLock("CBQQ.xml") // 必须阻塞,直到 unlock1 调用
		entered.Store(true)
		unlock2()
		released.Store(true)
	}()

	// 让 goroutine 跑一下,验证此时还卡住
	time.Sleep(20 * time.Millisecond)
	if entered.Load() {
		t.Error("second acquire on same name should be blocked while first holds lock")
	}

	unlock1()
	// 给 goroutine 完成时间
	for i := 0; i < 100; i++ {
		if released.Load() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !released.Load() {
		t.Error("second acquire should succeed after first release")
	}
}

// TestAcquireFileLock_DifferentNamesIndependent 验证不同名锁互不阻塞。
func TestAcquireFileLock_DifferentNamesIndependent(t *testing.T) {
	h := &Handler{}

	unlock1 := h.acquireFileLock("CBQQ.xml")
	defer unlock1()

	// 取另一个文件名的锁应立即成功
	done := make(chan struct{})
	go func() {
		unlock2 := h.acquireFileLock("BTS.xml")
		unlock2()
		close(done)
	}()

	select {
	case <-done:
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Error("lock on different name should not block")
	}
}

// TestAcquireFileLock_ConcurrentSameName 压测同名锁严格串行性:
// N 个 goroutine 抢同一把锁,每个进临界区时 counter++,出临界区前必须看到 counter==1。
func TestAcquireFileLock_ConcurrentSameName(t *testing.T) {
	h := &Handler{}
	const N = 50
	var counter atomic.Int32
	var maxSeen atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := h.acquireFileLock("CBQQ.xml")
			cur := counter.Add(1)
			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}
			// 模拟临界区里的工作
			time.Sleep(time.Millisecond)
			counter.Add(-1)
			unlock()
		}()
	}
	wg.Wait()

	if got := maxSeen.Load(); got != 1 {
		t.Errorf("max concurrent holders = %d, want 1 (lock not serializing)", got)
	}
}

// TestDeleteSourceGuard 验证守门函数对所有 4 种 loaded_from 形态的判定一致。
// 这是 Delete handler 守门"事前判定"逻辑的最小契约——handler 行为依赖于此返回值。
func TestDeleteSourceGuard(t *testing.T) {
	cases := []struct {
		loadedFrom string
		want       bool // IsDeletable
	}{
		{"param-mappings/BTS.xml", false},        // 内置 → 不可删
		{"param-mappings-custom/CBQQ.xml", true}, // 自定义 → 可删
		{"param-mappings-custom/BTS.xml", true},  // 同名覆盖 custom 仍可删 (self-healing)
		{"BTS.xml", false},                       // 历史无前缀 → 保守拒删
		{"", false},                              // 异常空 → 拒删
		{"../../etc/passwd", false},              // 路径遍历 → 拒删(防御)
	}
	for _, tc := range cases {
		if got := IsDeletable(tc.loadedFrom); got != tc.want {
			t.Errorf("IsDeletable(%q) = %v, want %v", tc.loadedFrom, got, tc.want)
		}
	}
}
