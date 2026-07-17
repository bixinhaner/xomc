// Package event 已在 types.go 中声明包注释。
package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const maxDeliveries = 5

const (
	defaultPendingMsgLimit   = 65536
	defaultPendingBytesLimit = 64 * 1024 * 1024
	gpvPendingMsgLimit       = 200000
	gpvPendingBytesLimit     = 256 * 1024 * 1024
	defaultPullFetchWait     = 500 * time.Millisecond

	// 以下三个常量专为 provision-gpv pull consumer 调优。
	gpvPullBatchSize = 64               // 每次 Fetch 最多拉取 64 条
	gpvPullAckWait   = 30 * time.Second // handler 处理超时，超时后 NATS 重投

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
)

type PullTuning struct {
	BatchSize     int
	Concurrency   int
	AckWait       time.Duration
	MaxAckPending int
}

// NATSEventBus 是基于 NATS JetStream 的生产级事件总线实现。
// 每个事件通过 JetStream 持久化存储，支持 At-Least-Once 交付语义。
// QueueSubscribe 使用 Durable Consumer，各实例彺负载均衡，适用于多实例水平扩展。
// 事件处理失败后指数退退重新投递（最多 5 次），超出后终止该消息防止无限重试。
type NATSEventBus struct {
	conn              *nats.Conn
	js                nats.JetStreamContext
	subs              []*nats.Subscription // push / queue subscriptions
	pullSubscriptions []*pullSubscription  // pull subscriptions；Close() 负责 drain + unsubscribe
	mu                sync.Mutex
	pullTuning        map[string]PullTuning
	logger            *zap.Logger
	metrics           *EventBusMetrics // issue #20：投递结果指标；nil 时（单进程/单测）静默 no-op。
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewNATSEventBus creates an EventBus backed by NATS JetStream.
func NewNATSEventBus(conn *nats.Conn, js nats.JetStreamContext, logger *zap.Logger) *NATSEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &NATSEventBus{
		conn:       conn,
		js:         js,
		pullTuning: make(map[string]PullTuning),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// SetMetrics attaches delivery-outcome metrics (issue #20). Call before any
// Subscribe so wrapHandler observes them; nil-safe — when unset, the bus keeps
// the legacy log-only behaviour.
func (b *NATSEventBus) SetMetrics(m *EventBusMetrics) {
	b.metrics = m
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
	evt.Subject = subject
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	_, err = b.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("publish to NATS: %w", err)
	}
	return nil
}

func (b *NATSEventBus) Subscribe(subject string, handler EventHandler) (Subscription, error) {
	sub, err := b.js.Subscribe(subject, b.wrapHandler(handler),
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
	ackWait, maxDeliver := b.ensureQueueConsumerTuning(subject, queue)

	sub, err := b.js.QueueSubscribe(subject, queue, b.wrapHandler(handler),
		nats.Durable(queue),
		nats.AckExplicit(),
		nats.AckWait(ackWait),
		nats.MaxDeliver(maxDeliver),
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

func (b *NATSEventBus) PullSubscribe(subject string, queue string, handler EventHandler) (Subscription, error) {
	durable := pullDurableName(queue)
	tuning := b.pullTuningForSubject(subject)
	tuning = b.ensurePullTuningForDurable(subject, durable, tuning)
	sub, err := b.js.PullSubscribe(subject, durable,
		nats.AckExplicit(),
		nats.DeliverNew(), // 首次创建时从当前 stream 尾部开始，不重播历史消息（review R1）
		nats.AckWait(tuning.AckWait),
		nats.MaxAckPending(tuning.MaxAckPending),
		nats.MaxDeliver(maxDeliveries),
	)
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
	b.cleanupLegacyPushConsumer(subject, queue)

	return ps, nil
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
func (b *NATSEventBus) ensureQueueConsumerTuning(subject, durable string) (ackWait time.Duration, maxDeliver int) {
	ackWait, maxDeliver = queueSubscribeAckWait, maxDeliveries

	stream, err := b.js.StreamNameBySubject(subject)
	if err != nil {
		b.logger.Warn("resolve queue consumer stream failed",
			zap.String("subject", subject),
			zap.String("durable", durable),
			zap.Error(err))
		return
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
		return
	}

	cfg := info.Config
	changed := false
	if cfg.AckWait != ackWait {
		cfg.AckWait = ackWait
		changed = true
	}
	if cfg.MaxDeliver != maxDeliver {
		cfg.MaxDeliver = maxDeliver
		changed = true
	}
	if !changed {
		return
	}

	if _, err := b.js.UpdateConsumer(stream, &cfg); err != nil {
		b.logger.Warn("update existing queue consumer tuning failed; using existing values",
			zap.String("subject", subject),
			zap.String("stream", stream),
			zap.String("durable", durable),
			zap.Duration("desired_ack_wait", ackWait),
			zap.Duration("existing_ack_wait", info.Config.AckWait),
			zap.Int("desired_max_deliver", maxDeliver),
			zap.Int("existing_max_deliver", info.Config.MaxDeliver),
			zap.Error(err))
		if info.Config.AckWait > 0 {
			ackWait = info.Config.AckWait
		}
		if info.Config.MaxDeliver != 0 {
			maxDeliver = info.Config.MaxDeliver
		}
		return
	}
	b.logger.Info("updated existing queue consumer tuning",
		zap.String("subject", subject),
		zap.String("stream", stream),
		zap.String("durable", durable),
		zap.Duration("ack_wait", ackWait),
		zap.Int("max_deliver", maxDeliver))
	return
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
		return PullTuning{BatchSize: gpvPullBatchSize, Concurrency: defaultPullConcurrent, AckWait: gpvPullAckWait, MaxAckPending: defaultPullMaxAckPending}
	}
	if subject == SubjectParamSyncTaskResult {
		return PullTuning{BatchSize: paramSyncResultPullBatchSize, Concurrency: paramSyncResultPullConcurrent, AckWait: paramSyncResultPullAckWait, MaxAckPending: paramSyncResultMaxAckPending}
	}
	return PullTuning{BatchSize: 32, Concurrency: defaultPullConcurrent, AckWait: gpvPullAckWait, MaxAckPending: defaultPullMaxAckPending}
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
				b.processMsg(handler, msg)
			}(msg)
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
	defer b.mu.Unlock()

	for _, sub := range b.subs {
		if err := sub.Unsubscribe(); err != nil {
			b.logger.Warn("unsubscribe error", zap.Error(err))
		}
	}
	b.subs = nil
	b.pullSubscriptions = nil
	return nil
}

func (b *NATSEventBus) wrapHandler(handler EventHandler) nats.MsgHandler {
	return func(msg *nats.Msg) {
		b.processMsg(handler, msg)
	}
}

func (b *NATSEventBus) processMsg(handler EventHandler, msg *nats.Msg) {
	evt, parseErr := decodeEventBytes(msg.Data)
	if parseErr != nil {
		b.logger.Error("unmarshal event", zap.Error(parseErr))
		// Permanent parse error — terminate to avoid infinite retry.
		// subject 用 msg.Subject（已解出 evt 之前），保证 dropped 指标有 subject 维度。
		b.metrics.inc(msg.Subject, deliveryOutcomeDropped)
		_ = msg.Term()
		return
	}

	handlerErr := handler(b.ctx, evt)
	deliveries := uint64(1)
	if meta, mErr := msg.Metadata(); mErr == nil && meta != nil {
		deliveries = meta.NumDelivered
	}

	decision := decideAck(handlerErr, deliveries, maxDeliveries)
	switch decision.action {
	case ackActionAck:
		b.metrics.inc(evt.Subject, deliveryOutcomeAck)
		_ = msg.Ack()
	case ackActionTerm:
		// 达到 maxDeliveries 后 Term 终止：消息被永久丢弃。此前只有 ERROR 日志，
		// 无指标 → max-retries 终止"静默"。补 terminated 指标使其可告警。
		b.metrics.inc(evt.Subject, deliveryOutcomeTerminated)
		b.logger.Error("handle event (terminating)",
			zap.String("subject", evt.Subject),
			zap.Uint64("delivery", deliveries),
			zap.Error(handlerErr))
		_ = msg.Term()
	case ackActionNak:
		b.metrics.inc(evt.Subject, deliveryOutcomeNak)
		b.logger.Error("handle event (retrying)",
			zap.String("subject", evt.Subject),
			zap.Uint64("delivery", deliveries),
			zap.Duration("backoff", decision.backoff),
			zap.Error(handlerErr))
		_ = msg.NakWithDelay(decision.backoff)
	}
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
//   - handler 出错且未达 maxDelivery → Nak with exponential backoff
//   - handler 出错且达到 maxDelivery → Term（避免无限重试）
//
// 指数退避序列：1s, 2s, 4s, 8s ...（基于 deliveries 已投递次数）。
// 该函数无副作用，可独立测试。
func decideAck(handlerErr error, deliveries, maxDelivery uint64) ackDecision {
	if handlerErr == nil {
		return ackDecision{action: ackActionAck}
	}
	if deliveries >= maxDelivery {
		return ackDecision{action: ackActionTerm}
	}
	if deliveries == 0 {
		deliveries = 1
	}
	// shift 上限保护，防止极端 deliveries 触发位移溢出。
	shift := deliveries - 1
	if shift > 30 {
		shift = 30
	}
	return ackDecision{
		action:  ackActionNak,
		backoff: time.Duration(1<<shift) * time.Second,
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
