package collector

import (
	"context"
	"testing"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// countingBus 是只统计 QueueSubscribe 调用次数的最小 EventBus 假实现。
type countingBus struct {
	queueSubs int
	lastQueue string
}

func (b *countingBus) Publish(context.Context, string, event.Event) error { return nil }
func (b *countingBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return noopSub{}, nil
}
func (b *countingBus) QueueSubscribe(_ string, queue string, _ event.EventHandler) (event.Subscription, error) {
	b.queueSubs++
	b.lastQueue = queue
	return noopSub{}, nil
}
func (b *countingBus) PullSubscribe(_ string, queue string, _ event.EventHandler) (event.Subscription, error) {
	b.queueSubs++
	b.lastQueue = queue
	return noopSub{}, nil
}
func (b *countingBus) Close() error { return nil }

type noopSub struct{}

func (noopSub) Unsubscribe() error { return nil }

func newTestCollector() *PMCollector {
	return NewPMCollector(nil, "", nil, nil, nil, nil, zap.NewNop())
}

// 默认（未设并发）退化为单订阅，沿用旧行为。
func TestPMCollector_Subscribe_DefaultSingleSubscription(t *testing.T) {
	bus := &countingBus{}
	require.NoError(t, newTestCollector().Subscribe(bus))
	assert.Equal(t, 1, bus.queueSubs)
	assert.Equal(t, "pm-workers", bus.lastQueue, "所有订阅共享同一 durable queue 组")
}

// SetConcurrency(N) → N 个 QueueSubscribe，全部落在同一 pm-workers 组（JetStream 负载均衡）。
func TestPMCollector_Subscribe_NConcurrency(t *testing.T) {
	bus := &countingBus{}
	c := newTestCollector()
	c.SetConcurrency(4)
	require.NoError(t, c.Subscribe(bus))
	assert.Equal(t, 4, bus.queueSubs)
	assert.Equal(t, "pm-workers", bus.lastQueue)
}

// 并发数 <=0 兜底为 1，避免零订阅静默吞消息。
func TestPMCollector_Subscribe_NonPositiveFallsBackToOne(t *testing.T) {
	for _, n := range []int{0, -3} {
		bus := &countingBus{}
		c := newTestCollector()
		c.SetConcurrency(n)
		require.NoError(t, c.Subscribe(bus))
		assert.Equal(t, 1, bus.queueSubs, "n=%d 应兜底为单订阅", n)
	}
}
