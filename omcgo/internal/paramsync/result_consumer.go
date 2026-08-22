package paramsync

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/event"
)

const (
	defaultResultConsumerShardCount = 4
	defaultResultConsumerQueueDepth = 32
	maxResultConsumerShardCount     = 4
	maxResultConsumerQueueDepth     = 64
)

type ResultConsumer struct {
	bus        event.EventBus
	processor  ResultProcessor
	sub        event.Subscription
	subject    string
	queue      string
	shards     []chan resultWork
	shardCount int
	queueDepth int
	metrics    *Metrics
	wg         sync.WaitGroup
	mu         sync.RWMutex
}

type resultWork struct {
	ctx     context.Context
	payload event.ParamSyncTaskResultPayload
	done    chan error
}

func NewResultConsumer(bus event.EventBus, processor ResultProcessor) *ResultConsumer {
	return &ResultConsumer{
		bus: bus, processor: processor,
		subject:    event.SubjectParamSyncTaskResult,
		queue:      "param-sync-results",
		shardCount: defaultResultConsumerShardCount,
		queueDepth: defaultResultConsumerQueueDepth,
	}
}

func (c *ResultConsumer) WithSubscription(subject, queue string) *ResultConsumer {
	if subject != "" {
		c.subject = subject
	}
	if queue != "" {
		c.queue = queue
	}
	return c
}

func (c *ResultConsumer) WithWorkerConfig(shardCount, queueDepth int) *ResultConsumer {
	c.shardCount, c.queueDepth = normalizeResultConsumerWorkerConfig(shardCount, queueDepth)
	return c
}

func (c *ResultConsumer) WithMetrics(metrics *Metrics) *ResultConsumer {
	c.metrics = metrics
	return c
}

func (c *ResultConsumer) Start() error {
	if c.bus == nil || c.processor == nil {
		return fmt.Errorf("parameter sync result consumer requires event bus and processor")
	}
	c.startWorkers(c.shardCount, c.queueDepth)
	sub, err := c.bus.PullSubscribe(c.subject, c.queue, c.Handle)
	if err != nil {
		c.stopWorkers()
		return fmt.Errorf("subscribe parameter sync task results: %w", err)
	}
	c.sub = sub
	return nil
}

func (c *ResultConsumer) Handle(ctx context.Context, evt event.Event) error {
	var payload event.ParamSyncTaskResultPayload
	if err := evt.DecodePayload(&payload); err != nil {
		// Malformed payload is permanent: acknowledge it after surfacing a typed
		// error is not possible with the generic EventBus. Keep it observable by
		// returning nil only after the bus-level poison-message limit/DLQ policy.
		return fmt.Errorf("decode parameter sync task result: %w", err)
	}
	if err := c.dispatch(ctx, payload); err != nil {
		return err
	}
	return nil
}

func (c *ResultConsumer) dispatch(ctx context.Context, payload event.ParamSyncTaskResultPayload) error {
	c.mu.RLock()
	shards := c.shards
	if len(shards) == 0 {
		c.mu.RUnlock()
		return c.process(ctx, payload)
	}
	work := resultWork{ctx: ctx, payload: payload, done: make(chan error, 1)}
	shardIndex := resultShardIndex(payload, len(shards))
	shard := shards[shardIndex]
	shardLabel := strconv.Itoa(shardIndex)
	select {
	case shard <- work:
		if c.metrics != nil {
			c.metrics.ResultShardQueueDepth.WithLabelValues(shardLabel).Set(float64(len(shard)))
		}
	case <-ctx.Done():
		c.mu.RUnlock()
		return ctx.Err()
	default:
		if c.metrics != nil {
			c.metrics.ResultShardQueueDepth.WithLabelValues(shardLabel).Set(float64(len(shard)))
			c.metrics.ResultShardQueueFull.WithLabelValues(shardLabel).Inc()
		}
		c.mu.RUnlock()
		return fmt.Errorf("parameter sync result shard queue is full")
	}
	c.mu.RUnlock()
	select {
	case err := <-work.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *ResultConsumer) process(ctx context.Context, payload event.ParamSyncTaskResultPayload) error {
	start := time.Now()
	defer func() {
		if c.metrics != nil {
			c.metrics.ResultProcessDuration.Observe(time.Since(start).Seconds())
		}
	}()
	if _, err := c.processor.Process(ctx, payload); err != nil {
		// Returning an error causes NATSEventBus to NAK/redeliver. A nil return is
		// therefore strictly after the database transaction committed.
		return err
	}
	return nil
}

func (c *ResultConsumer) Stop() error {
	var err error
	if c.sub == nil {
		c.stopWorkers()
		return nil
	}
	err = c.sub.Unsubscribe()
	c.stopWorkers()
	return err
}

func (c *ResultConsumer) startWorkers(shardCount, queueDepth int) {
	shardCount, queueDepth = normalizeResultConsumerWorkerConfig(shardCount, queueDepth)
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.shards) > 0 {
		return
	}
	c.shards = make([]chan resultWork, shardCount)
	for i := range c.shards {
		queue := make(chan resultWork, queueDepth)
		c.shards[i] = queue
		c.wg.Add(1)
		go c.runShard(i, queue)
	}
}

func normalizeResultConsumerWorkerConfig(shardCount, queueDepth int) (int, int) {
	if shardCount <= 0 {
		shardCount = defaultResultConsumerShardCount
	}
	if shardCount > maxResultConsumerShardCount {
		shardCount = maxResultConsumerShardCount
	}
	if queueDepth <= 0 {
		queueDepth = defaultResultConsumerQueueDepth
	}
	if queueDepth > maxResultConsumerQueueDepth {
		queueDepth = maxResultConsumerQueueDepth
	}
	return shardCount, queueDepth
}

func (c *ResultConsumer) runShard(index int, queue <-chan resultWork) {
	defer c.wg.Done()
	shardLabel := strconv.Itoa(index)
	for work := range queue {
		if c.metrics != nil {
			c.metrics.ResultShardQueueDepth.WithLabelValues(shardLabel).Set(float64(len(queue)))
			c.metrics.ResultShardInflight.WithLabelValues(shardLabel).Inc()
		}
		work.done <- c.process(work.ctx, work.payload)
		if c.metrics != nil {
			c.metrics.ResultShardInflight.WithLabelValues(shardLabel).Dec()
			c.metrics.ResultShardQueueDepth.WithLabelValues(shardLabel).Set(float64(len(queue)))
		}
	}
}

func (c *ResultConsumer) stopWorkers() {
	c.mu.Lock()
	shards := c.shards
	if len(shards) > 0 {
		c.shards = nil
		for _, shard := range shards {
			close(shard)
		}
	}
	c.mu.Unlock()
	if len(shards) > 0 {
		c.wg.Wait()
	}
}

func resultShardIndex(payload event.ParamSyncTaskResultPayload, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	key := payload.DeviceSN
	if payload.DeviceID != uuid.Nil {
		key = payload.DeviceID.String()
	}
	if key == "" {
		key = payload.RunID.String()
	}
	if key == "" {
		key = payload.TaskID
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(shardCount))
}
