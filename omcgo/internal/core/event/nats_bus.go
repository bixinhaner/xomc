// Package event 已在 types.go 中声明包注释。
package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/reliability"
)

const maxDeliveries = 5

const (
	defaultPendingMsgLimit   = 65536
	defaultPendingBytesLimit = 64 * 1024 * 1024
	gpvPendingMsgLimit       = 200000
	gpvPendingBytesLimit     = 256 * 1024 * 1024
	defaultPullFetchWait     = 500 * time.Millisecond
	defaultRefreshCooldown   = 2 * time.Second

	// 以下三个常量专为 provision-gpv pull consumer 调优。
	gpvPullBatchSize      = 64               // 每次 Fetch 最多拉取 64 条
	gpvPullConcurrent     = 2                // 两个 fetch/dispatch 槽覆盖 20k 设备周期 GPV 峰值
	gpvPullAckWait        = 30 * time.Second // handler 处理超时，超时后 NATS 重投
	gpvQueueMaxAckPending = 2000             // RPC 持久化双 worker 的服务端在途上限

	// queueSubscribeAckWait 是 QueueSubscribe（push consumer，pm-workers 等）的
	// 显式 ack_wait。此前不设置，NATS 用服务端默认 30s + 无限 MaxDeliver，高并发下
	// DB 慢查询导致 handler 处理耗时超过 30s 时，NATS 会在 Go 侧 decideAck 还没来得及
	// Ack/Term 之前就重投同一条消息，造成并发重复处理（现象：worker 日志里 Term 时的
	// delivery 计数远超 maxDeliveries=5 常量，说明重投已经先发生了很多次）。2 分钟对齐
	// paramSyncResultPullAckWait，给 DB 争用场景留够处理时间。
	queueSubscribeAckWait = 2 * time.Minute

	pullNoCapacityWait            = 10 * time.Millisecond
	paramSyncResultPullBatchSize  = 64
	paramSyncResultPullConcurrent = 64
	paramSyncResultPullAckWait    = 2 * time.Minute
	paramSyncResultMaxAckPending  = 512
	defaultPullConcurrent         = 1
	defaultPullMaxAckPending      = 2048 // 背压上限：未 Ack 消息超过此值时 Fetch 阻塞
	// 非 PM push consumer 保持 NATS 既有默认容量；PM subject 由 worker 按实际并发单独收紧。
	defaultQueueMaxAckPending = 1000
)

type PullTuning struct {
	BatchSize     int
	Concurrency   int
	AckWait       time.Duration
	MaxDeliver    int
	MaxAckPending int
}

// QueueTuning 控制 push durable consumer 的服务端在途上限。
type QueueTuning struct {
	AckWait       time.Duration
	MaxDeliver    int
	MaxAckPending int
}

type EventKeyFunc func(Event) (string, error)

// KeyedQueueConfig configures a fixed durable whose messages are dispatched
// through a bounded hash-sharded worker set. StartSequence is a one-time
// migration input and is ignored when Durable already exists.
type KeyedQueueConfig struct {
	Durable       string
	StartSequence uint64
	Concurrency   int
	QueueDepth    int
	AckWait       time.Duration
	MaxDeliver    int
	MaxAckPending int
}

type durableDeliveryPolicy uint8

const (
	durableDeliveryBindExisting durableDeliveryPolicy = iota
	durableDeliveryAll
	durableDeliveryNew
	durableDeliveryFromSequence
)

type durableDelivery struct {
	policy        durableDeliveryPolicy
	startSequence uint64
}

// durableDeliveryPlan keeps consumer creation lossless:
//   - an existing durable is always bound in place, preserving its delivery state;
//   - an explicit migration starts exactly at the captured predecessor AckFloor+1;
//   - a fresh consumer without a handoff sequence starts at the current tail,
//     matching the legacy ephemeral subscriber without replaying retained history.
//
// A lossless upgrade must explicitly pass the predecessor AckFloor+1. Treating
// zero as DeliverAll would replay the entire retained COMMAND stream whenever an
// older app.prod.yaml is upgraded without the new gpv_response section.
func durableDeliveryPlan(existing *nats.ConsumerInfo, startSequence uint64) durableDelivery {
	if existing != nil {
		return durableDelivery{policy: durableDeliveryBindExisting}
	}
	if startSequence > 0 {
		return durableDelivery{
			policy:        durableDeliveryFromSequence,
			startSequence: startSequence,
		}
	}
	return durableDelivery{policy: durableDeliveryNew}
}

func keyedMaxAckPending(concurrency, queueDepth, requested int) int {
	if concurrency < 1 {
		concurrency = 1
	}
	if queueDepth < 0 {
		queueDepth = 0
	}
	capacity := concurrency * (queueDepth + 1)
	if requested <= 0 || requested > capacity {
		return capacity
	}
	return requested
}

type keyedJob func()

// keyedDispatcher is a bounded per-key executor. FIFO within one key keeps all
// events for the same device ordered, while different keys can run concurrently
// up to the configured global limit. Submit blocks when total in-flight local
// work reaches capacity, propagating backpressure to JetStream instead of
// acknowledging work before it runs.
type keyedDispatcher struct {
	ctx         context.Context
	cancel      context.CancelFunc
	queues      map[string][]keyedJob
	active      map[string]bool
	pendingSlot chan struct{}
	workerSlot  chan struct{}
	subject     string
	durable     string
	metrics     *EventBusMetrics
	mu          sync.Mutex
	wg          sync.WaitGroup
	closeOnce   sync.Once
}

func newKeyedDispatcher(concurrency, queueDepth int) *keyedDispatcher {
	return newObservedKeyedDispatcher(concurrency, queueDepth, "", "", nil)
}

func newObservedKeyedDispatcher(
	concurrency, queueDepth int,
	subject, durable string,
	metrics *EventBusMetrics,
) *keyedDispatcher {
	if concurrency < 1 {
		concurrency = 1
	}
	if queueDepth < 1 {
		queueDepth = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	d := &keyedDispatcher{
		ctx:         ctx,
		cancel:      cancel,
		queues:      make(map[string][]keyedJob),
		active:      make(map[string]bool),
		pendingSlot: make(chan struct{}, keyedMaxAckPending(concurrency, queueDepth, 0)),
		workerSlot:  make(chan struct{}, concurrency),
		subject:     subject,
		durable:     durable,
		metrics:     metrics,
	}
	d.metrics.initConsumerObservation(subject, durable)
	d.observeDepth()
	return d
}

func (d *keyedDispatcher) Submit(ctx context.Context, key string, job keyedJob) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case d.pendingSlot <- struct{}{}:
		d.observeDepth()
	case <-ctx.Done():
		return fmt.Errorf("submit keyed work: %w", ctx.Err())
	case <-d.ctx.Done():
		return fmt.Errorf("submit keyed work: dispatcher closed")
	}

	d.mu.Lock()
	if err := d.ctx.Err(); err != nil {
		d.mu.Unlock()
		<-d.pendingSlot
		d.observeDepth()
		return fmt.Errorf("submit keyed work: dispatcher closed")
	}
	d.queues[key] = append(d.queues[key], job)
	if !d.active[key] {
		d.active[key] = true
		d.wg.Add(1)
		go d.runKey(key)
	}
	d.mu.Unlock()
	return nil
}

func (d *keyedDispatcher) Close() {
	d.closeOnce.Do(d.cancel)
	d.wg.Wait()
}

func (d *keyedDispatcher) runKey(key string) {
	defer d.wg.Done()
	select {
	case d.workerSlot <- struct{}{}:
	case <-d.ctx.Done():
		d.discardKey(key)
		return
	}
	defer func() { <-d.workerSlot }()

	for {
		if err := d.ctx.Err(); err != nil {
			d.discardKey(key)
			return
		}
		job, ok := d.nextKeyJob(key)
		if !ok {
			return
		}
		if job != nil {
			job()
		}
		<-d.pendingSlot
		d.observeDepth()
	}
}

func (d *keyedDispatcher) nextKeyJob(key string) (keyedJob, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	queue := d.queues[key]
	if len(queue) == 0 {
		delete(d.queues, key)
		delete(d.active, key)
		return nil, false
	}
	job := queue[0]
	copy(queue, queue[1:])
	queue[len(queue)-1] = nil
	queue = queue[:len(queue)-1]
	if len(queue) == 0 {
		delete(d.queues, key)
	} else {
		d.queues[key] = queue
	}
	return job, true
}

func (d *keyedDispatcher) discardKey(key string) {
	d.mu.Lock()
	queue := d.queues[key]
	delete(d.queues, key)
	delete(d.active, key)
	d.mu.Unlock()
	for range queue {
		<-d.pendingSlot
		d.observeDepth()
	}
}

func (d *keyedDispatcher) observeDepth() {
	if d.metrics == nil || d.subject == "" || d.durable == "" {
		return
	}
	d.metrics.observeLocalQueueDepth(d.subject, d.durable, len(d.pendingSlot))
}

func keyedShardIndex(key string, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(key))
	return int(hash.Sum32() % uint32(shardCount))
}

// keepAckPendingAlive renews the JetStream acknowledgement deadline while a
// message is waiting in a keyed queue or executing a slow handler. Without this,
// queued messages can be redelivered after AckWait and overtake earlier work
// for the same device.
func (b *NATSEventBus) keepAckPendingAlive(
	parent context.Context,
	msg *nats.Msg,
	durable string,
	ackWait time.Duration,
) func() {
	if parent == nil {
		parent = context.Background()
	}
	if ackWait <= 0 {
		ackWait = gpvPullAckWait
	}
	interval := ackWait / 3
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := msg.InProgress(); err != nil && ctx.Err() == nil {
					b.observeAckFailure(msg.Subject, durable, "in_progress", err)
					b.logger.Warn("renew keyed message ack deadline failed",
						zap.String("subject", msg.Subject),
						zap.String("durable", durable),
						zap.Error(err))
				}
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			<-done
		})
	}
}

// QueueStats is a point-in-time JetStream durable consumer health sample.
// Pending and AckPending are kept separate so callers can distinguish queued
// work from messages already delivered to a worker but not yet acknowledged.
type QueueStats struct {
	Pending             uint64
	AckPending          int
	Redelivered         int
	OldestPendingAge    time.Duration
	LastSequence        uint64
	AckSequence         uint64
	AckGap              uint64
	DeliverySequence    uint64
	AckConsumerSequence uint64
	SampledAt           time.Time
}

// queueStatsReader keeps the QueueStats NATS management calls small and
// directly testable while the bus continues to use nats.JetStreamContext for
// all publishing and subscription operations.
type queueStatsReader interface {
	StreamNameBySubject(ctx context.Context, subject string) (string, error)
	ConsumerInfo(ctx context.Context, stream, durable string) (*nats.ConsumerInfo, error)
	StreamInfo(ctx context.Context, stream string) (*nats.StreamInfo, error)
	NextMessage(ctx context.Context, stream string, startSequence uint64, subject string) (*nats.RawStreamMsg, error)
}

// JetStreamPublisher is the minimal producer surface NATSEventBus needs.
// nats.JetStreamContext implements it; tests can inject a narrow fake without
// implementing the entire JetStream management interface.
type JetStreamPublisher interface {
	Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error)
}

type JetStreamRefreshFunc func(context.Context) (*nats.Conn, JetStreamPublisher, error)

type jetStreamQueueStatsReader struct {
	js nats.JetStreamContext
}

func (r jetStreamQueueStatsReader) StreamNameBySubject(ctx context.Context, subject string) (string, error) {
	return r.js.StreamNameBySubject(subject, nats.Context(ctx))
}

func (r jetStreamQueueStatsReader) ConsumerInfo(ctx context.Context, stream, durable string) (*nats.ConsumerInfo, error) {
	return r.js.ConsumerInfo(stream, durable, nats.Context(ctx))
}

func (r jetStreamQueueStatsReader) StreamInfo(ctx context.Context, stream string) (*nats.StreamInfo, error) {
	return r.js.StreamInfo(stream, nats.Context(ctx))
}

func (r jetStreamQueueStatsReader) NextMessage(ctx context.Context, stream string, startSequence uint64, subject string) (*nats.RawStreamMsg, error) {
	return r.js.GetMsg(stream, startSequence, nats.DirectGetNext(subject), nats.Context(ctx))
}

// NATSEventBus 是基于 NATS JetStream 的生产级事件总线实现。
// 每个事件通过 JetStream 持久化存储，支持 At-Least-Once 交付语义。
// QueueSubscribe 使用 Durable Consumer，各实例彺负载均衡，适用于多实例水平扩展。
// 事件处理失败后指数退退重新投递（最多 5 次），超出后终止该消息防止无限重试。
type NATSEventBus struct {
	conn               *nats.Conn
	js                 nats.JetStreamContext
	publisher          JetStreamPublisher
	subs               []*nats.Subscription // push / queue subscriptions
	pullSubscriptions  []*pullSubscription  // pull subscriptions；Close() 负责 drain + unsubscribe
	keyedSubscriptions []*keyedQueueSubscription
	mu                 sync.Mutex
	refreshMu          sync.Mutex
	publisherVersion   uint64
	refreshCooldown    time.Duration
	lastRefreshAttempt time.Time
	pullTuning         map[string]PullTuning
	queueTuning        map[string]QueueTuning
	logger             *zap.Logger
	metrics            *EventBusMetrics // issue #20：投递结果指标；nil 时（单进程/单测）静默 no-op。
	refreshJetStream   JetStreamRefreshFunc
	queueStatsReader   queueStatsReader // nil 时直接通过 js 读取；测试可替换为受控 reader。
	ctx                context.Context
	cancel             context.CancelFunc
}

// QueueStats returns a consistent read-only health sample for one durable
// consumer. It does not create, update, or otherwise change JetStream state.
func (b *NATSEventBus) QueueStats(ctx context.Context, subject, durable string) (QueueStats, error) {
	if err := ctx.Err(); err != nil {
		return QueueStats{}, fmt.Errorf("queue stats context: %w", err)
	}

	reader := b.queueStatsReader
	if reader == nil {
		reader = jetStreamQueueStatsReader{js: b.js}
	}

	stream, err := reader.StreamNameBySubject(ctx, subject)
	if err != nil {
		return QueueStats{}, fmt.Errorf("resolve queue stream for subject %q: %w", subject, err)
	}
	consumer, err := reader.ConsumerInfo(ctx, stream, durable)
	if err != nil {
		return QueueStats{}, fmt.Errorf("load queue consumer info for stream %q durable %q: %w", stream, durable, err)
	}
	if consumer == nil {
		return QueueStats{}, fmt.Errorf("load queue consumer info for stream %q durable %q: empty response", stream, durable)
	}
	streamInfo, err := reader.StreamInfo(ctx, stream)
	if err != nil {
		return QueueStats{}, fmt.Errorf("load queue stream info for stream %q: %w", stream, err)
	}
	if streamInfo == nil {
		return QueueStats{}, fmt.Errorf("load queue stream info for stream %q: empty response", stream)
	}

	var oldest *nats.RawStreamMsg
	if startSequence := oldestRelevantPendingStart(consumer, streamInfo.State); startSequence > 0 {
		oldest, err = reader.NextMessage(ctx, stream, startSequence, subject)
		if err != nil && !errors.Is(err, nats.ErrMsgNotFound) && !errors.Is(err, nats.ErrNoResponders) {
			return QueueStats{}, fmt.Errorf("load oldest pending message for stream %q subject %q from sequence %d: %w", stream, subject, startSequence, err)
		}
	}

	// The sample completes only after the optional subject-filtered raw-message
	// lookup, so SampledAt always represents the successful collection end.
	sampledAt := time.Now()
	stats := QueueStats{
		Pending:             consumer.NumPending,
		AckPending:          consumer.NumAckPending,
		Redelivered:         consumer.NumRedelivered,
		LastSequence:        streamInfo.State.LastSeq,
		AckSequence:         consumer.AckFloor.Stream,
		AckGap:              ackGap(consumer.Delivered.Consumer, consumer.AckFloor.Consumer),
		DeliverySequence:    consumer.Delivered.Consumer,
		AckConsumerSequence: consumer.AckFloor.Consumer,
		SampledAt:           sampledAt,
	}
	if oldest != nil && !oldest.Time.IsZero() {
		stats.OldestPendingAge = sampledAt.Sub(oldest.Time)
		if stats.OldestPendingAge < 0 {
			stats.OldestPendingAge = 0
		}
	}

	return stats, nil
}

func ackGap(delivered, acked uint64) uint64 {
	if delivered <= acked {
		return 0
	}
	return delivered - acked
}

// oldestRelevantPendingStart returns a safe lower bound for a single
// subject-filtered lookup. For queued work, the consumer has not delivered
// anything beyond Delivered.Stream. For ack-only work, JetStream exposes only
// the contiguous AckFloor; later acknowledgement gaps are not enumerable via
// ConsumerInfo, so AckFloor+1 is the oldest conservative candidate.
func oldestRelevantPendingStart(consumer *nats.ConsumerInfo, state nats.StreamState) uint64 {
	if consumer == nil || state.FirstSeq == 0 || state.LastSeq == 0 || (consumer.NumPending == 0 && consumer.NumAckPending <= 0) {
		return 0
	}
	start := state.FirstSeq
	if consumer.NumAckPending > 0 {
		if consumer.AckFloor.Stream == ^uint64(0) {
			return 0
		}
		start = maxQueueSequence(start, consumer.AckFloor.Stream+1)
	} else {
		if consumer.Delivered.Stream == ^uint64(0) {
			return 0
		}
		start = maxQueueSequence(start, consumer.Delivered.Stream+1)
	}
	if start > state.LastSeq {
		return 0
	}
	return start
}

func maxQueueSequence(left, right uint64) uint64 {
	if left > right {
		return left
	}
	return right
}

// PendingCount returns queued plus delivered-but-unacked messages for one
// durable consumer. It is kept as a one-release compatibility wrapper around
// QueueStats for capacity projections that have not yet adopted the full
// health sample.
func (b *NATSEventBus) PendingCount(subject, durable string) (uint64, error) {
	stats, err := b.QueueStats(context.Background(), subject, durable)
	if err != nil {
		return 0, err
	}
	if stats.AckPending <= 0 {
		return stats.Pending, nil
	}
	return stats.Pending + uint64(stats.AckPending), nil
}

func (b *NATSEventBus) runQueueStatsMetricSampler(ctx context.Context, subject, durable string, interval time.Duration) {
	if b == nil || b.metrics == nil || !observableConsumerQueue(subject, durable) {
		return
	}
	if interval <= 0 {
		interval = QueueHealthSampleInterval
	}
	sample := func() {
		sampleCtx, cancel := context.WithTimeout(ctx, defaultQueueHealthSampleTimeout)
		defer cancel()
		stats, err := b.QueueStats(sampleCtx, subject, durable)
		if err != nil {
			b.metrics.observeConsumerQueueSampleFailure(subject, durable)
			b.logger.Warn("sample eventbus consumer queue health failed",
				zap.String("subject", subject),
				zap.String("durable", durable),
				zap.Error(err))
			return
		}
		b.metrics.observeQueueStats(subject, durable, stats)
	}
	sample()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sample()
		}
	}
}

// NewNATSEventBus creates an EventBus backed by NATS JetStream.
func NewNATSEventBus(conn *nats.Conn, js nats.JetStreamContext, logger *zap.Logger) *NATSEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &NATSEventBus{
		conn:            conn,
		js:              js,
		publisher:       js,
		refreshCooldown: defaultRefreshCooldown,
		pullTuning:      make(map[string]PullTuning),
		queueTuning:     make(map[string]QueueTuning),
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (b *NATSEventBus) SetQueueTuning(subject string, tuning QueueTuning) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.queueTuning == nil {
		b.queueTuning = make(map[string]QueueTuning)
	}
	b.queueTuning[subject] = tuning
}

// SetMetrics attaches delivery-outcome metrics (issue #20). Call before any
// Subscribe so wrapHandler observes them; nil-safe — when unset, the bus keeps
// the legacy log-only behaviour.
func (b *NATSEventBus) SetMetrics(m *EventBusMetrics) {
	b.metrics = m
}

func (b *NATSEventBus) SetJetStreamRefreshFunc(fn JetStreamRefreshFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refreshJetStream = fn
}

// NewQueueHealthSampler returns the single owner for PM queue-health metric
// observation. QueueStats itself remains a read-only collection method so
// disk-projection callers cannot accidentally create duplicate samples.
func (b *NATSEventBus) NewQueueHealthSampler(interval time.Duration) *QueueHealthSampler {
	return NewQueueHealthSampler(b, b.metrics, interval, b.logger)
}

func (b *NATSEventBus) SetPullTuning(subject string, tuning PullTuning) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pullTuning == nil {
		b.pullTuning = make(map[string]PullTuning)
	}
	b.pullTuning[subject] = tuning
}

func (b *NATSEventBus) Publish(ctx context.Context, subject string, evt Event) error {
	if ctx == nil {
		ctx = context.Background()
	}
	evt.Subject = subject
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	opts := publishOptions(ctx, evt)

	publisherVersion, err := b.publishRaw(subject, data, opts...)
	if err != nil {
		errorClass := classifyAckError(err)
		b.metrics.incPublish(subject, publishOutcomeFailed, errorClass)
		if shouldRefreshJetStreamAfterPublish(err) && ctx.Err() == nil {
			if refreshErr := b.refreshPublisher(ctx, publisherVersion); refreshErr != nil {
				b.metrics.incPublish(subject, publishOutcomeRecoveryFailed, classifyAckError(refreshErr))
				return fmt.Errorf("refresh NATS producer after publish failure: %w", refreshErr)
			}
			if _, retryErr := b.publishRaw(subject, data, opts...); retryErr != nil {
				b.metrics.incPublish(subject, publishOutcomeRecoveryFailed, classifyAckError(retryErr))
				return fmt.Errorf("publish to NATS after producer refresh: %w", retryErr)
			}
			b.metrics.incPublish(subject, publishOutcomeRecovered, errorClass)
			return nil
		}
		return fmt.Errorf("publish to NATS: %w", err)
	}
	b.metrics.incPublish(subject, publishOutcomeSuccess, "")
	return nil
}

func publishOptions(ctx context.Context, evt Event) []nats.PubOpt {
	opts := []nats.PubOpt{nats.Context(ctx)}
	if evt.ID != "" {
		opts = append(opts, nats.MsgId(evt.ID))
	}
	return opts
}

func (b *NATSEventBus) publishRaw(subject string, data []byte, opts ...nats.PubOpt) (uint64, error) {
	b.mu.Lock()
	publisher := b.publisher
	publisherVersion := b.publisherVersion
	b.mu.Unlock()
	_, err := publisher.Publish(subject, data, opts...)
	return publisherVersion, err
}

func (b *NATSEventBus) refreshPublisher(ctx context.Context, failedPublisherVersion uint64) error {
	b.refreshMu.Lock()
	defer b.refreshMu.Unlock()

	b.mu.Lock()
	if b.publisherVersion != failedPublisherVersion {
		b.mu.Unlock()
		return nil
	}
	refreshFn := b.refreshJetStream
	cooldown := b.refreshCooldown
	lastAttempt := b.lastRefreshAttempt
	if cooldown <= 0 {
		cooldown = defaultRefreshCooldown
	}
	now := time.Now()
	if !lastAttempt.IsZero() && now.Sub(lastAttempt) < cooldown {
		b.mu.Unlock()
		return fmt.Errorf("NATS producer refresh cooling down: %w", nats.ErrConnectionReconnecting)
	}
	b.lastRefreshAttempt = now
	b.mu.Unlock()

	if refreshFn == nil {
		return errors.New("NATS producer refresh is not configured")
	}

	conn, publisher, err := refreshFn(ctx)
	if err != nil {
		return err
	}
	if publisher == nil {
		return errors.New("NATS producer refresh returned nil publisher")
	}

	b.mu.Lock()
	if b.publisherVersion != failedPublisherVersion {
		b.mu.Unlock()
		return nil
	}
	b.conn = conn
	b.publisher = publisher
	b.publisherVersion++
	if js, ok := publisher.(nats.JetStreamContext); ok {
		b.js = js
		b.queueStatsReader = nil
	}
	b.mu.Unlock()
	return nil
}

func (b *NATSEventBus) Subscribe(subject string, handler EventHandler) (Subscription, error) {
	b.metrics.initConsumerObservation(subject, "")
	sub, err := b.js.Subscribe(subject, b.wrapHandler(handler, maxDeliveries),
		nats.DeliverAll(),
		nats.AckExplicit(),
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe to %s: %w", subject, err)
	}

	b.mu.Lock()
	b.subs = append(b.subs, sub)
	b.mu.Unlock()

	return &natsSubscription{sub: sub}, nil
}

func (b *NATSEventBus) QueueSubscribe(subject string, queue string, handler EventHandler) (Subscription, error) {
	tuning := b.ensureQueueConsumerTuning(subject, queue)
	stream, err := b.ensureBoundQueueConsumer(subject, queue, tuning, durableDelivery{
		policy: durableDeliveryAll,
	})
	if err != nil {
		return nil, err
	}
	b.metrics.initConsumerObservation(subject, queue)

	sub, err := b.js.QueueSubscribe(subject, queue, b.wrapHandlerWithDurable(handler, tuning.MaxDeliver, queue),
		nats.Bind(stream, queue),
		nats.AckExplicit(),
		nats.AckWait(tuning.AckWait),
		nats.MaxDeliver(tuning.MaxDeliver),
		nats.MaxAckPending(tuning.MaxAckPending),
	)
	if err != nil {
		return nil, fmt.Errorf("queue subscribe to %s (queue=%s): %w", subject, queue, err)
	}

	msgLimit, bytesLimit := pendingLimitsForSubject(subject)
	if err := sub.SetPendingLimits(msgLimit, bytesLimit); err != nil {
		_ = sub.Unsubscribe()
		return nil, fmt.Errorf("set pending limits for %s (queue=%s): %w", subject, queue, err)
	}
	b.logger.Info("queue subscription pending limits applied",
		zap.String("subject", subject),
		zap.String("queue", queue),
		zap.Int("pending_msgs", msgLimit),
		zap.Int("pending_bytes", bytesLimit),
	)

	b.mu.Lock()
	b.subs = append(b.subs, sub)
	b.mu.Unlock()

	return &natsSubscription{sub: sub}, nil
}

// ensureBoundQueueConsumer creates the durable explicitly before subscribing.
//
// nats.go marks a consumer created implicitly by QueueSubscribe for deletion
// when Subscription.Unsubscribe/Drain runs. During an application rolling
// restart, deleting many durable consumers serializes JetStream metadata work
// and can stall unrelated publish acknowledgements. Creating first and binding
// makes shutdown detach only the client subscription while preserving both the
// durable position and NATS availability for concurrent ACS publishers.
func (b *NATSEventBus) ensureBoundQueueConsumer(
	subject, durable string,
	tuning QueueTuning,
	delivery durableDelivery,
) (string, error) {
	stream, err := b.js.StreamNameBySubject(subject)
	if err != nil {
		return "", fmt.Errorf("resolve queue stream for %s: %w", subject, err)
	}
	if _, err = b.js.ConsumerInfo(stream, durable); err == nil {
		return stream, nil
	} else if !errors.Is(err, nats.ErrConsumerNotFound) {
		return "", fmt.Errorf("load queue consumer %s/%s: %w", stream, durable, err)
	}

	config := &nats.ConsumerConfig{
		Durable:        durable,
		DeliverSubject: nats.NewInbox(),
		DeliverGroup:   durable,
		AckPolicy:      nats.AckExplicitPolicy,
		AckWait:        tuning.AckWait,
		MaxDeliver:     tuning.MaxDeliver,
		MaxAckPending:  tuning.MaxAckPending,
		ReplayPolicy:   nats.ReplayInstantPolicy,
		FilterSubject:  subject,
	}
	switch delivery.policy {
	case durableDeliveryNew:
		config.DeliverPolicy = nats.DeliverNewPolicy
	case durableDeliveryFromSequence:
		config.DeliverPolicy = nats.DeliverByStartSequencePolicy
		config.OptStartSeq = delivery.startSequence
	default:
		config.DeliverPolicy = nats.DeliverAllPolicy
	}
	_, err = b.js.AddConsumer(stream, config)
	if err == nil {
		return stream, nil
	}

	// Multiple service instances may race to create the shared queue durable.
	// Treat a concurrently visible compatible consumer as success; the bind
	// below remains the final compatibility check.
	if _, lookupErr := b.js.ConsumerInfo(stream, durable); lookupErr == nil {
		return stream, nil
	}
	return "", fmt.Errorf("create queue consumer %s/%s: %w", stream, durable, err)
}

// KeyedQueueSubscribe creates or binds a fixed push durable and dispatches its
// messages by key. The single NATS callback preserves JetStream delivery order
// while bounded per-key queues provide cross-key parallelism. A message is
// settled only by the worker after handler completion; filling local capacity
// blocks the callback and lets MaxAckPending apply server-side backpressure.
func (b *NATSEventBus) KeyedQueueSubscribe(
	subject string,
	config KeyedQueueConfig,
	keyFunc EventKeyFunc,
	handler EventHandler,
) (Subscription, error) {
	if config.Durable == "" {
		return nil, fmt.Errorf("keyed queue subscribe to %s: durable is required", subject)
	}
	if keyFunc == nil {
		return nil, fmt.Errorf("keyed queue subscribe to %s: key function is required", subject)
	}
	if config.Concurrency < 1 {
		config.Concurrency = 1
	}
	if config.QueueDepth < 1 {
		config.QueueDepth = 1
	}
	if config.AckWait <= 0 {
		config.AckWait = gpvPullAckWait
	}
	if config.MaxDeliver <= 0 {
		config.MaxDeliver = maxDeliveries
	}
	config.MaxAckPending = keyedMaxAckPending(
		config.Concurrency,
		config.QueueDepth,
		config.MaxAckPending,
	)

	stream, err := b.js.StreamNameBySubject(subject)
	if err != nil {
		return nil, fmt.Errorf("resolve keyed queue stream for %s: %w", subject, err)
	}
	info, err := b.js.ConsumerInfo(stream, config.Durable)
	if err != nil {
		if !errors.Is(err, nats.ErrConsumerNotFound) {
			return nil, fmt.Errorf(
				"load keyed queue consumer %s/%s: %w",
				stream,
				config.Durable,
				err,
			)
		}
		info = nil
	}
	if info != nil {
		if info.Config.DeliverSubject == "" {
			return nil, fmt.Errorf(
				"keyed queue consumer %s/%s is pull-based; refusing unsafe replacement",
				stream,
				config.Durable,
			)
		}
		if info.Config.FilterSubject != "" && info.Config.FilterSubject != subject {
			return nil, fmt.Errorf(
				"keyed queue consumer %s/%s filters %s, expected %s",
				stream,
				config.Durable,
				info.Config.FilterSubject,
				subject,
			)
		}
	}

	tuning := b.ensureQueueConsumerTuningWithDesired(subject, config.Durable, QueueTuning{
		AckWait:       config.AckWait,
		MaxDeliver:    config.MaxDeliver,
		MaxAckPending: config.MaxAckPending,
	})
	plan := durableDeliveryPlan(info, config.StartSequence)
	stream, err = b.ensureBoundQueueConsumer(subject, config.Durable, tuning, plan)
	if err != nil {
		return nil, err
	}
	options := []nats.SubOpt{
		nats.Bind(stream, config.Durable),
		nats.AckExplicit(),
		nats.ManualAck(),
		nats.AckWait(tuning.AckWait),
		nats.MaxDeliver(tuning.MaxDeliver),
		nats.MaxAckPending(tuning.MaxAckPending),
	}
	switch plan.policy {
	case durableDeliveryNew:
		options = append(options, nats.DeliverNew())
	case durableDeliveryFromSequence:
		options = append(options, nats.StartSequence(plan.startSequence))
	case durableDeliveryBindExisting:
		// No delivery-policy option: bind without resetting consumer state.
	}

	dispatcher := newObservedKeyedDispatcher(
		config.Concurrency,
		config.QueueDepth,
		subject,
		config.Durable,
		b.metrics,
	)
	sub, err := b.js.QueueSubscribe(subject, config.Durable, func(msg *nats.Msg) {
		stopProgress := b.keepAckPendingAlive(b.ctx, msg, config.Durable, tuning.AckWait)
		evt, decodeErr := decodeEventBytes(msg.Data)
		if decodeErr != nil {
			stopProgress()
			b.dropMalformedMsg(msg, config.Durable, decodeErr)
			return
		}
		key, keyErr := keyFunc(evt)
		if keyErr != nil {
			stopProgress()
			b.settleDecodedMsg(evt, msg, config.Durable, keyErr, tuning.MaxDeliver)
			return
		}
		if submitErr := dispatcher.Submit(b.ctx, key, func() {
			b.processKeyedMsg(b.ctx, evt, msg, config.Durable, tuning, handler)
			stopProgress()
		}); submitErr != nil {
			stopProgress()
			b.metrics.inc(evt.Subject, deliveryOutcomeNak)
			if err := msg.NakWithDelay(time.Second); err != nil {
				b.observeAckFailure(evt.Subject, config.Durable, "nak", err)
				b.logger.Error("nak keyed message after submit failure failed",
					zap.String("subject", evt.Subject),
					zap.String("durable", config.Durable),
					zap.Error(err),
					zap.NamedError("submit_error", submitErr))
			}
		}
	}, options...)
	if err != nil {
		dispatcher.Close()
		return nil, fmt.Errorf(
			"keyed queue subscribe to %s (durable=%s): %w",
			subject,
			config.Durable,
			err,
		)
	}
	msgLimit, bytesLimit := pendingLimitsForSubject(subject)
	if err := sub.SetPendingLimits(msgLimit, bytesLimit); err != nil {
		_ = sub.Unsubscribe()
		dispatcher.Close()
		return nil, fmt.Errorf(
			"set keyed pending limits for %s (durable=%s): %w",
			subject,
			config.Durable,
			err,
		)
	}

	keyedSub := &keyedQueueSubscription{sub: sub, dispatcher: dispatcher}
	b.mu.Lock()
	b.keyedSubscriptions = append(b.keyedSubscriptions, keyedSub)
	b.mu.Unlock()
	b.logger.Info("keyed queue subscription started",
		zap.String("subject", subject),
		zap.String("durable", config.Durable),
		zap.Uint64("start_sequence", config.StartSequence),
		zap.Int("concurrency", config.Concurrency),
		zap.Int("queue_depth", config.QueueDepth),
		zap.Duration("ack_wait", tuning.AckWait),
		zap.Int("max_deliver", tuning.MaxDeliver),
		zap.Int("max_ack_pending", tuning.MaxAckPending))
	go b.runQueueStatsMetricSampler(b.ctx, subject, config.Durable, QueueHealthSampleInterval)
	return keyedSub, nil
}

func (b *NATSEventBus) PullSubscribe(subject string, queue string, handler EventHandler) (Subscription, error) {
	durable := pullDurableName(queue)
	stream, err := b.js.StreamNameBySubject(subject)
	if err != nil {
		return nil, fmt.Errorf("resolve pull consumer stream for %s: %w", subject, err)
	}
	info, err := b.js.ConsumerInfo(stream, durable)
	if err != nil {
		if !errors.Is(err, nats.ErrConsumerNotFound) {
			return nil, fmt.Errorf("load pull consumer %s/%s: %w", stream, durable, err)
		}
		info = nil
	}
	tuning := b.pullTuningForSubject(subject)
	tuning = b.ensurePullTuningForDurable(subject, durable, tuning)
	if err := b.ensureBoundPullConsumer(stream, subject, durable, tuning, durableDeliveryPlan(info, 0)); err != nil {
		return nil, err
	}
	b.metrics.initConsumerObservation(subject, durable)
	options := []nats.SubOpt{
		nats.Bind(stream, durable),
		nats.AckExplicit(),
		nats.AckWait(tuning.AckWait),
		nats.MaxAckPending(tuning.MaxAckPending),
		nats.MaxDeliver(tuning.MaxDeliver),
	}
	if durableDeliveryPlan(info, 0).policy == durableDeliveryNew {
		options = append(options, nats.DeliverNew())
	}
	sub, err := b.js.PullSubscribe(subject, durable, options...)
	if err != nil {
		return nil, fmt.Errorf("pull subscribe to %s (durable=%s): %w", subject, durable, err)
	}

	ctx, cancel := context.WithCancel(b.ctx)
	ps := &pullSubscription{
		sub:    sub,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go b.runPullSubscription(ctx, ps, subject, durable, handler)

	// pull subscription 独立追踪（不加入 b.subs），Close() 通过 pullSubscription.Unsubscribe()
	// 统一 drain goroutine + unsubscribe，避免与 b.subs 循环双重调用。
	b.mu.Lock()
	b.pullSubscriptions = append(b.pullSubscriptions, ps)
	b.mu.Unlock()

	b.logger.Info("pull subscription started",
		zap.String("subject", subject),
		zap.String("durable", durable),
		zap.Int("batch_size", tuning.BatchSize),
		zap.Int("concurrency", tuning.Concurrency),
		zap.Duration("ack_wait", tuning.AckWait),
		zap.Int("max_ack_pending", tuning.MaxAckPending))

	// 自动清理同 queue 名对应的旧 push consumer（迁移到 pull 后的遗留，
	// consumer 名即 queue 本身，不含 "-pull" 后缀）。幂等：不存在时静默跳过。
	if queue != durable {
		b.cleanupLegacyPushConsumer(subject, queue)
	}

	return ps, nil
}

// KeyedPullSubscribe preserves fetch order while dispatching different keys in
// parallel. Messages are decoded and submitted sequentially in JetStream
// delivery order; each key then processes its own events FIFO.
func (b *NATSEventBus) KeyedPullSubscribe(
	subject, queue string,
	queueDepth int,
	keyFunc EventKeyFunc,
	handler EventHandler,
) (Subscription, error) {
	if keyFunc == nil {
		return nil, fmt.Errorf("keyed pull subscribe to %s: key function is required", subject)
	}
	if queueDepth < 1 {
		queueDepth = 1
	}
	durable := pullDurableName(queue)
	stream, err := b.js.StreamNameBySubject(subject)
	if err != nil {
		return nil, fmt.Errorf("resolve keyed pull stream for %s: %w", subject, err)
	}
	info, err := b.js.ConsumerInfo(stream, durable)
	if err != nil {
		if !errors.Is(err, nats.ErrConsumerNotFound) {
			return nil, fmt.Errorf("load keyed pull consumer %s/%s: %w", stream, durable, err)
		}
		info = nil
	}
	if info != nil && info.Config.DeliverSubject != "" {
		return nil, fmt.Errorf(
			"keyed pull consumer %s/%s is push-based; refusing unsafe replacement",
			stream,
			durable,
		)
	}

	tuning := b.pullTuningForSubject(subject)
	tuning.MaxAckPending = keyedMaxAckPending(
		tuning.Concurrency,
		queueDepth,
		tuning.MaxAckPending,
	)
	tuning = b.ensurePullTuningForDurable(subject, durable, tuning)
	if err := b.ensureBoundPullConsumer(stream, subject, durable, tuning, durableDeliveryPlan(info, 0)); err != nil {
		return nil, err
	}
	options := []nats.SubOpt{
		nats.Bind(stream, durable),
		nats.AckExplicit(),
		nats.AckWait(tuning.AckWait),
		nats.MaxAckPending(tuning.MaxAckPending),
		nats.MaxDeliver(tuning.MaxDeliver),
	}
	if durableDeliveryPlan(info, 0).policy == durableDeliveryNew {
		options = append(options, nats.DeliverNew())
	}
	sub, err := b.js.PullSubscribe(subject, durable, options...)
	if err != nil {
		return nil, fmt.Errorf(
			"keyed pull subscribe to %s (durable=%s): %w",
			subject,
			durable,
			err,
		)
	}

	ctx, cancel := context.WithCancel(b.ctx)
	ps := &pullSubscription{sub: sub, cancel: cancel, done: make(chan struct{})}
	dispatcher := newObservedKeyedDispatcher(
		tuning.Concurrency,
		queueDepth,
		subject,
		durable,
		b.metrics,
	)
	go b.runKeyedPullSubscription(
		ctx,
		ps,
		subject,
		durable,
		tuning,
		dispatcher,
		keyFunc,
		handler,
	)

	b.mu.Lock()
	b.pullSubscriptions = append(b.pullSubscriptions, ps)
	b.mu.Unlock()
	b.logger.Info("keyed pull subscription started",
		zap.String("subject", subject),
		zap.String("durable", durable),
		zap.Int("batch_size", tuning.BatchSize),
		zap.Int("concurrency", tuning.Concurrency),
		zap.Int("queue_depth", queueDepth),
		zap.Duration("ack_wait", tuning.AckWait),
		zap.Int("max_ack_pending", tuning.MaxAckPending))
	if queue != durable {
		b.cleanupLegacyPushConsumer(subject, queue)
	}
	go b.runQueueStatsMetricSampler(ctx, subject, durable, QueueHealthSampleInterval)
	return ps, nil
}

func (b *NATSEventBus) ensureBoundPullConsumer(
	stream, subject, durable string,
	tuning PullTuning,
	delivery durableDelivery,
) error {
	if _, err := b.js.ConsumerInfo(stream, durable); err == nil {
		return nil
	} else if !errors.Is(err, nats.ErrConsumerNotFound) {
		return fmt.Errorf("load pull consumer %s/%s: %w", stream, durable, err)
	}

	config := &nats.ConsumerConfig{
		Durable:       durable,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       tuning.AckWait,
		MaxDeliver:    tuning.MaxDeliver,
		MaxAckPending: tuning.MaxAckPending,
		ReplayPolicy:  nats.ReplayInstantPolicy,
		FilterSubject: subject,
	}
	switch delivery.policy {
	case durableDeliveryFromSequence:
		config.DeliverPolicy = nats.DeliverByStartSequencePolicy
		config.OptStartSeq = delivery.startSequence
	default:
		config.DeliverPolicy = nats.DeliverNewPolicy
	}
	if _, err := b.js.AddConsumer(stream, config); err == nil {
		return nil
	} else if _, lookupErr := b.js.ConsumerInfo(stream, durable); lookupErr == nil {
		return nil
	} else {
		return fmt.Errorf("create pull consumer %s/%s: %w", stream, durable, err)
	}
}

func (b *NATSEventBus) ensurePullTuningForDurable(subject, durable string, desired PullTuning) PullTuning {
	stream, err := b.js.StreamNameBySubject(subject)
	if err != nil {
		b.logger.Warn("resolve pull consumer stream failed",
			zap.String("subject", subject),
			zap.String("durable", durable),
			zap.Error(err))
		return desired
	}
	info, err := b.js.ConsumerInfo(stream, durable)
	if err != nil {
		if !errors.Is(err, nats.ErrConsumerNotFound) {
			b.logger.Warn("load existing pull consumer failed",
				zap.String("subject", subject),
				zap.String("stream", stream),
				zap.String("durable", durable),
				zap.Error(err))
		}
		return desired
	}
	updatedConfig, needsUpdate := updatedPullConsumerConfig(info, desired)
	if !needsUpdate {
		return desired
	}
	if _, err := b.js.UpdateConsumer(stream, &updatedConfig); err != nil {
		compatible := reconcilePullTuningWithExisting(desired, info)
		b.logger.Warn("update existing pull consumer tuning failed; using existing values",
			zap.String("subject", subject),
			zap.String("stream", stream),
			zap.String("durable", durable),
			zap.Duration("desired_ack_wait", desired.AckWait),
			zap.Duration("existing_ack_wait", compatible.AckWait),
			zap.Int("desired_max_ack_pending", desired.MaxAckPending),
			zap.Int("existing_max_ack_pending", compatible.MaxAckPending),
			zap.Error(err))
		return compatible
	}
	b.logger.Info("updated existing pull consumer tuning",
		zap.String("subject", subject),
		zap.String("stream", stream),
		zap.String("durable", durable),
		zap.Duration("ack_wait", desired.AckWait),
		zap.Int("max_ack_pending", desired.MaxAckPending))
	return desired
}

// ensureQueueConsumerTuning 保证 QueueSubscribe（push consumer）显式使用
// queueSubscribeAckWait + maxDeliveries，不再依赖 NATS 服务端默认值。若同名 durable
// consumer 已存在且配置不同，尝试 UpdateConsumer 就地更新；更新失败（如 NATS 版本
// 不支持修改某字段）则退回沿用已有配置，避免后续 QueueSubscribe 因期望值与已存在
// consumer 不一致而报错——与 ensurePullTuningForDurable 的处理方式保持一致。
func (b *NATSEventBus) ensureQueueConsumerTuning(subject, durable string) QueueTuning {
	return b.ensureQueueConsumerTuningWithDesired(subject, durable, b.queueTuningForSubject(subject))
}

func (b *NATSEventBus) ensureQueueConsumerTuningWithDesired(
	subject, durable string,
	desired QueueTuning,
) QueueTuning {
	stream, err := b.js.StreamNameBySubject(subject)
	if err != nil {
		b.logger.Warn("resolve queue consumer stream failed",
			zap.String("subject", subject),
			zap.String("durable", durable),
			zap.Error(err))
		return desired
	}
	info, err := b.js.ConsumerInfo(stream, durable)
	if err != nil {
		if !errors.Is(err, nats.ErrConsumerNotFound) {
			b.logger.Warn("load existing queue consumer failed",
				zap.String("subject", subject),
				zap.String("stream", stream),
				zap.String("durable", durable),
				zap.Error(err))
		}
		return desired
	}

	cfg, changed := updatedQueueConsumerConfig(info, desired)
	if !changed {
		return desired
	}

	if _, err := b.js.UpdateConsumer(stream, &cfg); err != nil {
		b.logger.Warn("update existing queue consumer tuning failed; using existing values",
			zap.String("subject", subject),
			zap.String("stream", stream),
			zap.String("durable", durable),
			zap.Duration("desired_ack_wait", desired.AckWait),
			zap.Duration("existing_ack_wait", info.Config.AckWait),
			zap.Int("desired_max_deliver", desired.MaxDeliver),
			zap.Int("existing_max_deliver", info.Config.MaxDeliver),
			zap.Int("desired_max_ack_pending", desired.MaxAckPending),
			zap.Int("existing_max_ack_pending", info.Config.MaxAckPending),
			zap.Error(err))
		if info.Config.AckWait > 0 {
			desired.AckWait = info.Config.AckWait
		}
		if info.Config.MaxDeliver != 0 {
			desired.MaxDeliver = info.Config.MaxDeliver
		}
		if info.Config.MaxAckPending > 0 {
			desired.MaxAckPending = info.Config.MaxAckPending
		}
		return desired
	}
	b.logger.Info("updated existing queue consumer tuning",
		zap.String("subject", subject),
		zap.String("stream", stream),
		zap.String("durable", durable),
		zap.Duration("ack_wait", desired.AckWait),
		zap.Int("max_deliver", desired.MaxDeliver),
		zap.Int("max_ack_pending", desired.MaxAckPending))
	return desired
}

func (b *NATSEventBus) queueTuningForSubject(subject string) QueueTuning {
	tuning := QueueTuning{
		AckWait:       queueSubscribeAckWait,
		MaxDeliver:    maxDeliveries,
		MaxAckPending: defaultQueueMaxAckPending,
	}
	if subject == SubjectCommandGetParamsResponse {
		tuning.AckWait = gpvPullAckWait
		tuning.MaxAckPending = gpvQueueMaxAckPending
	}
	b.mu.Lock()
	configured, ok := b.queueTuning[subject]
	b.mu.Unlock()
	if !ok {
		return tuning
	}
	if configured.AckWait > 0 {
		tuning.AckWait = configured.AckWait
	}
	if configured.MaxDeliver > 0 {
		tuning.MaxDeliver = configured.MaxDeliver
	}
	if configured.MaxAckPending > 0 {
		tuning.MaxAckPending = configured.MaxAckPending
	}
	return tuning
}

func updatedQueueConsumerConfig(info *nats.ConsumerInfo, desired QueueTuning) (nats.ConsumerConfig, bool) {
	if info == nil {
		return nats.ConsumerConfig{}, false
	}
	cfg := info.Config
	if cfg.Name == "" && cfg.Durable == "" {
		cfg.Durable = info.Name
	}
	changed := false
	if desired.AckWait > 0 && cfg.AckWait != desired.AckWait {
		cfg.AckWait = desired.AckWait
		changed = true
	}
	if desired.MaxDeliver > 0 && cfg.MaxDeliver != desired.MaxDeliver {
		cfg.MaxDeliver = desired.MaxDeliver
		changed = true
	}
	if desired.MaxAckPending > 0 && cfg.MaxAckPending != desired.MaxAckPending {
		cfg.MaxAckPending = desired.MaxAckPending
		changed = true
	}
	return cfg, changed
}

// cleanupLegacyPushConsumer 删除因迁移到 pull consumer 而遗留的同名旧 push consumer。
// legacyName 就是 queue（无 "-pull" 后缀）。
// 如果旧 consumer 不存在则静默跳过，不影响主流程。
func (b *NATSEventBus) cleanupLegacyPushConsumer(subject, legacyName string) {
	stream, err := b.js.StreamNameBySubject(subject)
	if err != nil {
		b.logger.Warn("cleanupLegacyPushConsumer: cannot resolve stream for subject",
			zap.String("subject", subject), zap.Error(err))
		return
	}
	if err := b.js.DeleteConsumer(stream, legacyName); err != nil {
		if !errors.Is(err, nats.ErrConsumerNotFound) {
			b.logger.Warn("cleanupLegacyPushConsumer: delete failed",
				zap.String("stream", stream),
				zap.String("consumer", legacyName),
				zap.Error(err))
		}
		return
	}
	b.logger.Info("cleanupLegacyPushConsumer: deleted legacy push consumer",
		zap.String("stream", stream),
		zap.String("consumer", legacyName))
}

func pullDurableName(queue string) string {
	if queue == "" {
		return "pull"
	}
	if strings.HasSuffix(queue, "-pull") {
		return queue
	}
	return queue + "-pull"
}

func defaultPullTuningForSubject(subject string) PullTuning {
	if subject == SubjectCommandGetParamsResponse {
		return PullTuning{
			BatchSize: gpvPullBatchSize, Concurrency: gpvPullConcurrent,
			AckWait: gpvPullAckWait, MaxDeliver: maxDeliveries,
			MaxAckPending: defaultPullMaxAckPending,
		}
	}
	if subject == SubjectParamSyncTaskResult {
		return PullTuning{
			BatchSize: paramSyncResultPullBatchSize, Concurrency: paramSyncResultPullConcurrent,
			AckWait: paramSyncResultPullAckWait, MaxDeliver: maxDeliveries,
			MaxAckPending: paramSyncResultMaxAckPending,
		}
	}
	return PullTuning{
		BatchSize: 32, Concurrency: defaultPullConcurrent,
		AckWait: gpvPullAckWait, MaxDeliver: maxDeliveries,
		MaxAckPending: defaultPullMaxAckPending,
	}
}

func (b *NATSEventBus) pullTuningForSubject(subject string) PullTuning {
	tuning := defaultPullTuningForSubject(subject)
	b.mu.Lock()
	configured, ok := b.pullTuning[subject]
	b.mu.Unlock()
	if !ok {
		return tuning
	}
	if configured.BatchSize > 0 {
		tuning.BatchSize = configured.BatchSize
	}
	if configured.Concurrency > 0 {
		tuning.Concurrency = configured.Concurrency
	}
	if configured.AckWait > 0 {
		tuning.AckWait = configured.AckWait
	}
	if configured.MaxDeliver > 0 {
		tuning.MaxDeliver = configured.MaxDeliver
	}
	if configured.MaxAckPending > 0 {
		tuning.MaxAckPending = configured.MaxAckPending
	}
	return tuning
}

func reconcilePullTuningWithExisting(desired PullTuning, info *nats.ConsumerInfo) PullTuning {
	if info == nil {
		return desired
	}
	out := desired
	if info.Config.AckWait > 0 {
		out.AckWait = info.Config.AckWait
	}
	if info.Config.MaxDeliver != 0 {
		out.MaxDeliver = info.Config.MaxDeliver
	}
	if info.Config.MaxAckPending > 0 {
		out.MaxAckPending = info.Config.MaxAckPending
	}
	return out
}

func updatedPullConsumerConfig(info *nats.ConsumerInfo, desired PullTuning) (nats.ConsumerConfig, bool) {
	if info == nil {
		return nats.ConsumerConfig{}, false
	}
	cfg := info.Config
	if cfg.Name == "" && cfg.Durable == "" {
		cfg.Durable = info.Name
	}
	changed := false
	if desired.AckWait > 0 && cfg.AckWait != desired.AckWait {
		cfg.AckWait = desired.AckWait
		changed = true
	}
	if desired.MaxDeliver > 0 && cfg.MaxDeliver != desired.MaxDeliver {
		cfg.MaxDeliver = desired.MaxDeliver
		changed = true
	}
	if desired.MaxAckPending > 0 && cfg.MaxAckPending != desired.MaxAckPending {
		cfg.MaxAckPending = desired.MaxAckPending
		changed = true
	}
	return cfg, changed
}

func pullFetchBatchForAvailableSlots(batchSize, concurrency, inUse int) int {
	if batchSize <= 0 {
		return 0
	}
	if concurrency <= 0 {
		return 0
	}
	available := concurrency - inUse
	if available <= 0 {
		return 0
	}
	if available < batchSize {
		return available
	}
	return batchSize
}

func (b *NATSEventBus) runPullSubscription(ctx context.Context, ps *pullSubscription, subject, durable string, handler EventHandler) {
	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
		close(ps.done)
	}()

	tuning := b.pullTuningForSubject(subject)
	batchSize := tuning.BatchSize
	concurrency := tuning.Concurrency
	sem := make(chan struct{}, concurrency)
	for {
		if ctx.Err() != nil {
			return
		}

		fetchBatch := pullFetchBatchForAvailableSlots(batchSize, concurrency, len(sem))
		if fetchBatch <= 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(pullNoCapacityWait):
				continue
			}
		}

		fetchCtx, fetchCancel := context.WithTimeout(ctx, defaultPullFetchWait)
		msgs, err := ps.sub.Fetch(fetchBatch, nats.Context(fetchCtx))
		fetchCancel()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, nats.ErrBadSubscription) {
				return
			}
			if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			b.logger.Warn("pull fetch failed",
				zap.String("subject", subject),
				zap.String("durable", durable),
				zap.Error(err))
			time.Sleep(100 * time.Millisecond)
			continue
		}

		for _, msg := range msgs {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			wg.Add(1)
			go func(msg *nats.Msg) {
				defer wg.Done()
				defer func() { <-sem }()
				b.processMsgWithDurable(handler, msg, tuning.MaxDeliver, durable)
			}(msg)
		}
	}
}

func (b *NATSEventBus) runKeyedPullSubscription(
	ctx context.Context,
	ps *pullSubscription,
	subject, durable string,
	tuning PullTuning,
	dispatcher *keyedDispatcher,
	keyFunc EventKeyFunc,
	handler EventHandler,
) {
	defer close(ps.done)
	defer dispatcher.Close()

	for {
		if ctx.Err() != nil {
			return
		}
		fetchCtx, fetchCancel := context.WithTimeout(ctx, defaultPullFetchWait)
		msgs, err := ps.sub.Fetch(tuning.BatchSize, nats.Context(fetchCtx))
		fetchCancel()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) ||
				errors.Is(err, nats.ErrBadSubscription) {
				return
			}
			if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			b.logger.Warn("keyed pull fetch failed",
				zap.String("subject", subject),
				zap.String("durable", durable),
				zap.Error(err))
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Fetch can return several messages at once. Start deadline renewal for
		// the whole batch before one hot key can block sequential submission.
		stops := make([]func(), len(msgs))
		for index, msg := range msgs {
			stops[index] = b.keepAckPendingAlive(ctx, msg, durable, tuning.AckWait)
		}
		for index, msg := range msgs {
			stopProgress := stops[index]
			evt, decodeErr := decodeEventBytes(msg.Data)
			if decodeErr != nil {
				stopProgress()
				b.dropMalformedMsg(msg, durable, decodeErr)
				continue
			}
			key, keyErr := keyFunc(evt)
			if keyErr != nil {
				stopProgress()
				b.settleDecodedMsg(evt, msg, durable, keyErr, tuning.MaxDeliver)
				continue
			}
			currentMsg := msg
			currentEvent := evt
			currentStop := stopProgress
			if submitErr := dispatcher.Submit(ctx, key, func() {
				b.processKeyedMsg(ctx, currentEvent, currentMsg, durable, QueueTuning{
					AckWait:    tuning.AckWait,
					MaxDeliver: tuning.MaxDeliver,
				}, handler)
				currentStop()
			}); submitErr != nil {
				currentStop()
				b.metrics.inc(currentEvent.Subject, deliveryOutcomeNak)
				if err := currentMsg.NakWithDelay(time.Second); err != nil {
					b.observeAckFailure(currentEvent.Subject, durable, "nak", err)
					b.logger.Error("nak keyed pull message after submit failure failed",
						zap.String("subject", currentEvent.Subject),
						zap.String("durable", durable),
						zap.Error(err),
						zap.NamedError("submit_error", submitErr))
				}
			}
		}
	}
}

func pendingLimitsForSubject(subject string) (msgLimit int, bytesLimit int) {
	if subject == SubjectCommandGetParamsResponse {
		return gpvPendingMsgLimit, gpvPendingBytesLimit
	}
	return defaultPendingMsgLimit, defaultPendingBytesLimit
}

func (b *NATSEventBus) Close() error {
	// Cancel the shared context so in-flight event handlers are notified.
	b.cancel()

	// Drain pull subscriptions: cancel ctx（已上方完成）→ 等待 goroutine 退出 → unsubscribe。
	// 使用 pullSubscription.Unsubscribe() 保证通过 sync.Once 单路径执行，消除双重 unsubscribe。
	b.mu.Lock()
	pullSubs := b.pullSubscriptions
	b.mu.Unlock()
	for _, ps := range pullSubs {
		if err := ps.Unsubscribe(); err != nil {
			b.logger.Warn("pull unsubscribe error", zap.Error(err))
		}
	}

	b.mu.Lock()
	keyedSubs := append([]*keyedQueueSubscription(nil), b.keyedSubscriptions...)
	b.mu.Unlock()
	for _, sub := range keyedSubs {
		if err := sub.Unsubscribe(); err != nil {
			b.logger.Warn("keyed queue unsubscribe error", zap.Error(err))
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sub := range b.subs {
		if err := sub.Unsubscribe(); err != nil {
			b.logger.Warn("unsubscribe error", zap.Error(err))
		}
	}
	b.subs = nil
	b.pullSubscriptions = nil
	b.keyedSubscriptions = nil
	return nil
}

func (b *NATSEventBus) wrapHandler(handler EventHandler, maxDelivery int) nats.MsgHandler {
	return b.wrapHandlerWithDurable(handler, maxDelivery, "")
}

func (b *NATSEventBus) wrapHandlerWithDurable(handler EventHandler, maxDelivery int, durable string) nats.MsgHandler {
	return func(msg *nats.Msg) {
		b.processMsgWithDurable(handler, msg, maxDelivery, durable)
	}
}

func (b *NATSEventBus) processMsg(handler EventHandler, msg *nats.Msg, maxDelivery int) {
	b.processMsgWithDurable(handler, msg, maxDelivery, "")
}

func (b *NATSEventBus) processMsgWithDurable(handler EventHandler, msg *nats.Msg, maxDelivery int, durable string) {
	evt, parseErr := decodeEventBytes(msg.Data)
	if parseErr != nil {
		b.dropMalformedMsg(msg, durable, parseErr)
		return
	}

	startedAt := time.Now()
	handlerErr := handler(b.ctx, evt)
	b.metrics.observeHandlerDuration(evt.Subject, durable, time.Since(startedAt))
	b.settleDecodedMsg(evt, msg, durable, handlerErr, maxDelivery)
}

func (b *NATSEventBus) dropMalformedMsg(msg *nats.Msg, durable string, parseErr error) {
	b.logger.Error("unmarshal event", zap.Error(parseErr))
	// Permanent parse error — terminate to avoid infinite retry.
	// subject 用 msg.Subject（已解出 evt 之前），保证 dropped 指标有 subject 维度。
	b.metrics.inc(msg.Subject, deliveryOutcomeDropped)
	if err := msg.Term(); err != nil {
		b.observeAckFailure(msg.Subject, durable, "term", err)
		b.logger.Error("terminate malformed event failed",
			zap.String("subject", msg.Subject),
			zap.String("durable", durable),
			zap.Error(err),
			zap.NamedError("parse_error", parseErr))
	}
}

func (b *NATSEventBus) settleDecodedMsg(
	evt Event,
	msg *nats.Msg,
	durable string,
	handlerErr error,
	maxDelivery int,
) {
	b.settleDecodedMsgAtDelivery(evt, msg, durable, handlerErr, messageDeliveryCount(msg), maxDelivery)
}

func messageDeliveryCount(msg *nats.Msg) uint64 {
	deliveries := uint64(1)
	if meta, mErr := msg.Metadata(); mErr == nil && meta != nil {
		deliveries = meta.NumDelivered
	}
	return deliveries
}

func (b *NATSEventBus) settleDecodedMsgAtDelivery(
	evt Event,
	msg *nats.Msg,
	durable string,
	handlerErr error,
	deliveries uint64,
	maxDelivery int,
) {
	decision := decideAck(handlerErr, deliveries, uint64(maxDelivery))
	switch decision.action {
	case ackActionAck:
		b.metrics.inc(evt.Subject, deliveryOutcomeAck)
		if err := msg.Ack(); err != nil {
			b.observeAckFailure(evt.Subject, durable, "ack", err)
			b.logger.Error("ack event failed",
				zap.String("subject", evt.Subject),
				zap.String("durable", durable),
				zap.Uint64("delivery", deliveries),
				zap.Error(err))
		}
	case ackActionTerm:
		// 达到 maxDeliveries 后 Term 终止：消息被永久丢弃。此前只有 ERROR 日志，
		// 无指标 → max-retries 终止"静默"。补 terminated 指标使其可告警。
		b.metrics.inc(evt.Subject, deliveryOutcomeTerminated)
		b.logger.Error("handle event (terminating)",
			zap.String("subject", evt.Subject),
			zap.Uint64("delivery", deliveries),
			zap.Error(handlerErr))
		if err := msg.Term(); err != nil {
			b.observeAckFailure(evt.Subject, durable, "term", err)
			b.logger.Error("terminate event failed",
				zap.String("subject", evt.Subject),
				zap.String("durable", durable),
				zap.Uint64("delivery", deliveries),
				zap.Error(err),
				zap.NamedError("handler_error", handlerErr))
		}
	case ackActionNak:
		b.metrics.inc(evt.Subject, deliveryOutcomeNak)
		fields := []zap.Field{
			zap.String("subject", evt.Subject),
			zap.Uint64("delivery", deliveries),
			zap.Duration("backoff", decision.backoff),
			zap.Error(handlerErr),
		}
		if errors.Is(handlerErr, reliability.ErrDeferred) {
			// 注册竞态等延迟条件是预期控制流；每次都打 Error 会附带堆栈，
			// 20k 设备清洁启动时形成日志与磁盘 I/O 放大。交付结果由 NAK
			// 指标观测，详细单条记录仅在 Debug 级别保留。
			b.logger.Debug("handle event (deferred)", fields...)
		} else {
			b.logger.Error("handle event (retrying)", fields...)
		}
		if err := msg.NakWithDelay(decision.backoff); err != nil {
			b.observeAckFailure(evt.Subject, durable, "nak", err)
			b.logger.Error("nak event failed",
				zap.String("subject", evt.Subject),
				zap.String("durable", durable),
				zap.Uint64("delivery", deliveries),
				zap.Duration("backoff", decision.backoff),
				zap.Error(err),
				zap.NamedError("handler_error", handlerErr))
		}
	}
}

func (b *NATSEventBus) observeAckFailure(subject, durable, action string, err error) {
	b.metrics.incAckFailure(subject, durable, action, classifyAckError(err))
}

func classifyAckError(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline"
	case errors.Is(err, nats.ErrConnectionClosed), errors.Is(err, nats.ErrConnectionDraining), errors.Is(err, io.EOF):
		return "connection_closed"
	case errors.Is(err, nats.ErrConnectionReconnecting):
		return "reconnecting"
	default:
		return "other"
	}
}

func shouldRefreshJetStreamAfterPublish(err error) bool {
	return errors.Is(err, nats.ErrConnectionClosed) ||
		errors.Is(err, nats.ErrConnectionDraining) ||
		errors.Is(err, nats.ErrConnectionReconnecting) ||
		errors.Is(err, io.EOF)
}

// processKeyedMsg keeps a failed head message inside its device lane until it
// either succeeds or reaches the configured terminal attempt. NAKing the head
// immediately would release the lane and allow a later response for the same
// device to persist before the redelivery.
func (b *NATSEventBus) processKeyedMsg(
	ctx context.Context,
	evt Event,
	msg *nats.Msg,
	durable string,
	tuning QueueTuning,
	handler EventHandler,
) {
	if tuning.MaxDeliver <= 0 {
		tuning.MaxDeliver = maxDeliveries
	}
	if tuning.AckWait <= 0 {
		tuning.AckWait = gpvPullAckWait
	}
	serverDelivery := messageDeliveryCount(msg)
	for localAttempt := uint64(0); ; localAttempt++ {
		startedAt := time.Now()
		handlerErr := handler(ctx, evt)
		b.metrics.observeHandlerDuration(evt.Subject, durable, time.Since(startedAt))
		effectiveDelivery := serverDelivery + localAttempt
		if handlerErr == nil || errors.Is(handlerErr, reliability.ErrPermanent) ||
			effectiveDelivery >= uint64(tuning.MaxDeliver) {
			b.settleDecodedMsgAtDelivery(
				evt,
				msg,
				durable,
				handlerErr,
				effectiveDelivery,
				tuning.MaxDeliver,
			)
			return
		}

		backoff := keyedRetryBackoff(effectiveDelivery, tuning.AckWait)
		b.logger.Warn("keyed event head failed; retrying in lane",
			zap.String("subject", evt.Subject),
			zap.Uint64("attempt", effectiveDelivery),
			zap.Duration("backoff", backoff),
			zap.Error(handlerErr))
		if err := msg.InProgress(); err != nil {
			b.observeAckFailure(evt.Subject, durable, "in_progress", err)
			b.logger.Warn("mark keyed message in-progress failed",
				zap.String("subject", evt.Subject),
				zap.String("durable", durable),
				zap.Uint64("attempt", effectiveDelivery),
				zap.Error(err),
				zap.NamedError("handler_error", handlerErr))
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			b.metrics.inc(evt.Subject, deliveryOutcomeNak)
			if err := msg.NakWithDelay(time.Second); err != nil {
				b.observeAckFailure(evt.Subject, durable, "nak", err)
				b.logger.Error("nak keyed message on shutdown failed",
					zap.String("subject", evt.Subject),
					zap.String("durable", durable),
					zap.Error(err))
			}
			return
		case <-timer.C:
		}
	}
}

func keyedRetryBackoff(attempt uint64, ackWait time.Duration) time.Duration {
	if attempt == 0 {
		attempt = 1
	}
	shift := attempt - 1
	if shift > 30 {
		shift = 30
	}
	backoff := time.Duration(1<<shift) * time.Second
	maxBackoff := ackWait / 3
	if maxBackoff < 10*time.Millisecond {
		maxBackoff = 10 * time.Millisecond
	}
	if backoff > maxBackoff {
		return maxBackoff
	}
	return backoff
}

// ackAction 表示对一条 NATS 消息的处置动作。
type ackAction int

const (
	ackActionAck ackAction = iota
	ackActionNak
	ackActionTerm
)

// ackDecision 描述 wrapHandler 在执行业务 handler 后对 nats.Msg 的处置。
// 把决策逻辑独立成纯函数便于单元测试，避免依赖真实 NATS server。
type ackDecision struct {
	action  ackAction
	backoff time.Duration
}

// decideAck 决定对已投递 deliveries 次的消息采取何种动作：
//   - handler 无错 → Ack
//   - handler 错误包装了 reliability.ErrPermanent（明确不可恢复的业务失败）
//     → 无视 deliveries，立即 Term，避免无意义的多次重投
//   - handler 出错且未达 maxDelivery → Nak with exponential backoff
//   - handler 出错且达到 maxDelivery → Term（避免无限重试）
//
// 指数退避序列：1s, 2s, 4s, 8s ...（基于 deliveries 已投递次数）。
// 该函数无副作用，可独立测试。
func decideAck(handlerErr error, deliveries, maxDelivery uint64) ackDecision {
	if handlerErr == nil {
		return ackDecision{action: ackActionAck}
	}
	if errors.Is(handlerErr, reliability.ErrPermanent) {
		return ackDecision{action: ackActionTerm}
	}
	if deliveries >= maxDelivery {
		return ackDecision{action: ackActionTerm}
	}
	return ackDecision{
		action:  ackActionNak,
		backoff: retryBackoff(deliveries),
	}
}

func retryBackoff(deliveries uint64) time.Duration {
	if deliveries == 0 {
		deliveries = 1
	}
	// shift 上限保护，防止极端 deliveries 触发位移溢出。
	shift := deliveries - 1
	if shift > 30 {
		shift = 30
	}
	return time.Duration(1<<shift) * time.Second
}

// MaxDeliveriesForRetryHorizon returns the smallest MaxDeliver value whose
// exponential NAK delays guarantee another delivery at or after horizon.
// Keeping this calculation beside retryBackoff prevents business grace windows
// from silently outgrowing the queue retry budget when either side is tuned.
func MaxDeliveriesForRetryHorizon(horizon time.Duration) int {
	if horizon <= 0 {
		return 1
	}
	deliveries := 1
	remaining := horizon
	for {
		delay := retryBackoff(uint64(deliveries))
		deliveries++
		if delay >= remaining {
			return deliveries
		}
		remaining -= delay
	}
}

// decodeEventBytes 把 JetStream 投递的 raw bytes 解析成 Event。
// 单独提取便于测试 unmarshal 路径与错误处理。
func decodeEventBytes(data []byte) (Event, error) {
	var evt Event
	if err := json.Unmarshal(data, &evt); err != nil {
		return Event{}, fmt.Errorf("unmarshal event: %w", err)
	}
	return evt, nil
}

type natsSubscription struct {
	sub *nats.Subscription
}

func (s *natsSubscription) Unsubscribe() error {
	return s.sub.Unsubscribe()
}

type keyedQueueSubscription struct {
	sub        *nats.Subscription
	dispatcher *keyedDispatcher
	once       sync.Once
	err        error
}

func (s *keyedQueueSubscription) Unsubscribe() error {
	s.once.Do(func() {
		s.err = s.sub.Unsubscribe()
		s.dispatcher.Close()
	})
	return s.err
}

type pullSubscription struct {
	sub    *nats.Subscription
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func (s *pullSubscription) Unsubscribe() error {
	var err error
	s.once.Do(func() {
		s.cancel()
		<-s.done
		err = s.sub.Unsubscribe()
	})
	return err
}
