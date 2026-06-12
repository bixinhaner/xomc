package software

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFairSlotPoolAcquireRelease(t *testing.T) {
	p := newFairSlotPool(2)
	taskA := uuid.New()
	ctx := context.Background()

	require.True(t, p.Acquire(ctx, taskA))
	require.True(t, p.Acquire(ctx, taskA))
	assert.Equal(t, 2, p.InUse())

	// 满载 + ctx 已取消 → 放弃排队。
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	assert.False(t, p.Acquire(cancelled, taskA))

	p.Release()
	require.True(t, p.Acquire(ctx, taskA))
	assert.Equal(t, 2, p.InUse())
}

// 轮转公平性：A 任务先排进 2 个等待者，B 任务后排 1 个；释放槽位时应 A、B 交替
// 放行（A1 → B1 → A2），而不是 FIFO 的 A1 → A2 → B1。
func TestFairSlotPoolRoundRobinAcrossTasks(t *testing.T) {
	p := newFairSlotPool(1)
	taskA, taskB := uuid.New(), uuid.New()
	require.True(t, p.Acquire(context.Background(), taskA)) // 占住唯一槽位

	type tagged struct{ tag string }
	got := make(chan tagged, 3)
	spawn := func(tag string, tid uuid.UUID) {
		go func() {
			if p.Acquire(context.Background(), tid) {
				got <- tagged{tag}
			}
		}()
		time.Sleep(20 * time.Millisecond) // 固定入队顺序：A1, A2, B1
	}
	spawn("A1", taskA)
	spawn("A2", taskA)
	spawn("B1", taskB)

	grant := func() string {
		p.Release()
		select {
		case g := <-got:
			return g.tag
		case <-time.After(2 * time.Second):
			t.Fatal("no waiter granted after release")
			return ""
		}
	}
	assert.Equal(t, "A1", grant(), "首个释放应给最早排队的 A1")
	assert.Equal(t, "B1", grant(), "轮转应跳到 B 任务，而不是 FIFO 给 A2")
	assert.Equal(t, "A2", grant(), "再轮回 A 任务")
}

func TestFairSlotPoolSetLimitGrantsWaiters(t *testing.T) {
	p := newFairSlotPool(1)
	taskA := uuid.New()
	require.True(t, p.Acquire(context.Background(), taskA))

	got := make(chan bool, 1)
	go func() { got <- p.Acquire(context.Background(), taskA) }()

	select {
	case <-got:
		t.Fatal("acquire should block at limit 1")
	case <-time.After(50 * time.Millisecond):
	}
	p.SetLimit(2)
	select {
	case ok := <-got:
		assert.True(t, ok)
	case <-time.After(2 * time.Second):
		t.Fatal("SetLimit did not grant the waiter")
	}

	// SetLimit(0) 视为未配置，不改变现值。
	p.SetLimit(0)
	assert.Equal(t, 2, p.InUse())
}

func TestFairSlotPoolCtxCancelDoesNotLeak(t *testing.T) {
	p := newFairSlotPool(1)
	taskA, taskB := uuid.New(), uuid.New()
	require.True(t, p.Acquire(context.Background(), taskA))

	// A 任务的等待者排队后取消。
	ctx, cancel := context.WithCancel(context.Background())
	gotA := make(chan bool, 1)
	go func() { gotA <- p.Acquire(ctx, taskA) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case ok := <-gotA:
		assert.False(t, ok, "cancelled waiter must not get a slot")
	case <-time.After(2 * time.Second):
		t.Fatal("ctx cancel did not unblock the waiter")
	}

	// B 任务接着排队；释放槽位时应跳过已取消的 A 等待者直接放行 B。
	gotB := make(chan bool, 1)
	go func() { gotB <- p.Acquire(context.Background(), taskB) }()
	time.Sleep(50 * time.Millisecond)
	p.Release()
	select {
	case ok := <-gotB:
		assert.True(t, ok, "cancelled waiter must be skipped, slot goes to B")
	case <-time.After(2 * time.Second):
		t.Fatal("slot leaked to a cancelled waiter")
	}
	assert.Equal(t, 1, p.InUse())
}
