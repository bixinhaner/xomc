package trace

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// CaptureConsumer worker 进程消费 trace.message.captured（JetStream WorkQueue），
// 累积到 FlushSize 或 FlushDelay 后批量写 trace_messages。
//
// 单 worker 实例内串行 flush 保证批量批次原子；多 worker 实例由 JetStream WorkQueuePolicy
// 自动 fan-out 负载均衡，互不重复。
//
// M2-06：注入 BulkStore 后，payload > MaxInlinePayloadBytes 时上传 MinIO trace-bulk，
// PG 仅存 payload_object_key + payload_size_bytes。
type CaptureConsumer struct {
	repo    Repository
	bulk    BulkPutter // 可 nil（M2-06 未启用时不外置）；接口便于单测注入 mock
	metrics *Metrics   // M3-01：可 nil
	logger  *zap.Logger

	flushSize  int
	flushDelay time.Duration
	queueName  string

	// pendingMu 保护 pending 切片不与 flush 并发
	pendingMu sync.Mutex
	pending   []*Message

	flushTimer *time.Timer
	dropped    uint64
	flushed    uint64
}

// CaptureConsumerConfig 配置。
type CaptureConsumerConfig struct {
	FlushSize  int           // 默认 64
	FlushDelay time.Duration // 默认 500ms
	QueueName  string        // 默认 "trace-capture"
}

// DefaultCaptureConsumerConfig 默认配置。
func DefaultCaptureConsumerConfig() CaptureConsumerConfig {
	return CaptureConsumerConfig{
		FlushSize:  64,
		FlushDelay: 500 * time.Millisecond,
		QueueName:  "trace-capture",
	}
}

// NewCaptureConsumer 构造函数。
func NewCaptureConsumer(repo Repository, cfg CaptureConsumerConfig, logger *zap.Logger) *CaptureConsumer {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.FlushSize <= 0 {
		cfg.FlushSize = 64
	}
	if cfg.FlushDelay <= 0 {
		cfg.FlushDelay = 500 * time.Millisecond
	}
	if cfg.QueueName == "" {
		cfg.QueueName = "trace-capture"
	}
	return &CaptureConsumer{
		repo:       repo,
		logger:     logger.Named("trace-capture-consumer"),
		flushSize:  cfg.FlushSize,
		flushDelay: cfg.FlushDelay,
		queueName:  cfg.QueueName,
	}
}

// BulkPutter 抽象大报文外置上传，便于单测 mock。
// 真实实现是 *BulkStore（写 MinIO）；测试用 in-memory mock。
type BulkPutter interface {
	Enabled() bool
	Put(ctx context.Context, taskID, msgID uuid.UUID, payload string) (string, error)
}

// SetBulkStore 注入 MinIO 大报文外置 store（M2-06）。
// 未注入时所有报文 inline；注入后 payload 超阈值自动外置。
func (c *CaptureConsumer) SetBulkStore(bulk BulkPutter) {
	c.bulk = bulk
}

// SetMetrics 注入 Prometheus 指标（M3-01）。
func (c *CaptureConsumer) SetMetrics(m *Metrics) { c.metrics = m }

// Subscribe 注册 QueueSubscribe 到 trace.message.captured。
// 多 worker 实例同一 queue 自动负载均衡。
func (c *CaptureConsumer) Subscribe(bus event.EventBus) (event.Subscription, error) {
	return bus.QueueSubscribe(event.SubjectTraceMessageCaptured, c.queueName, c.handle)
}

// handle 单条事件投递；累积到批量阈值或超时后 flush。
func (c *CaptureConsumer) handle(_ context.Context, evt event.Event) error {
	var msg Message
	if err := evt.DecodePayload(&msg); err != nil {
		atomic.AddUint64(&c.dropped, 1)
		if c.metrics != nil {
			c.metrics.MessagesDroppedTotal.WithLabelValues(DropReasonDecodeFailed).Inc()
		}
		c.logger.Warn("trace: decode capture event failed", zap.Error(err))
		return nil // 不 Nack — payload 损坏的事件无重试价值
	}
	if msg.CapturedAt.IsZero() {
		msg.CapturedAt = time.Now()
	}

	c.pendingMu.Lock()
	c.pending = append(c.pending, &msg)
	shouldFlushNow := len(c.pending) >= c.flushSize
	if !shouldFlushNow && c.flushTimer == nil {
		// 第一次入队时启动定时器
		c.flushTimer = time.AfterFunc(c.flushDelay, c.timedFlush)
	}
	c.pendingMu.Unlock()

	if shouldFlushNow {
		c.flush()
	}
	return nil
}

// timedFlush 由 AfterFunc 在 flushDelay 后调用；线程安全。
func (c *CaptureConsumer) timedFlush() {
	c.flush()
}

// flush 单次批量写库；线程安全（取出 pending 后释放锁再写库）。
func (c *CaptureConsumer) flush() {
	c.pendingMu.Lock()
	if c.flushTimer != nil {
		c.flushTimer.Stop()
		c.flushTimer = nil
	}
	if len(c.pending) == 0 {
		c.pendingMu.Unlock()
		return
	}
	batch := c.pending
	c.pending = nil
	c.pendingMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// M2-06：超阈值的 payload 上传 MinIO 后清空 inline + 写 object_key
	c.externalize(ctx, batch)

	if err := c.repo.InsertMessages(ctx, batch); err != nil {
		atomic.AddUint64(&c.dropped, uint64(len(batch)))
		if c.metrics != nil {
			c.metrics.MessagesDroppedTotal.WithLabelValues(DropReasonInsertFailed).Add(float64(len(batch)))
		}
		c.logger.Error("trace: batch insert failed",
			zap.Int("size", len(batch)), zap.Error(err))
		return
	}
	atomic.AddUint64(&c.flushed, uint64(len(batch)))
	if c.metrics != nil {
		// 按方向累加 captured 计数
		for _, m := range batch {
			c.metrics.MessagesCapturedTotal.WithLabelValues(string(m.Direction)).Inc()
		}
	}
	c.tallyCounts(ctx, batch)
}

// externalize 把超阈值的报文转存到 MinIO；失败仅 WARN 并保持 inline（确保不丢报文）。
func (c *CaptureConsumer) externalize(ctx context.Context, batch []*Message) {
	if c.bulk == nil || !c.bulk.Enabled() {
		return
	}
	for _, m := range batch {
		if m.PayloadObjectKey != "" || m.PayloadInline == "" {
			continue
		}
		if m.PayloadSizeBytes <= MaxInlinePayloadBytes && len(m.PayloadInline) <= MaxInlinePayloadBytes {
			continue
		}
		// 确保 message_id 在 InsertMessages 之前固化（外置对象键需要 id）
		if m.ID == uuid.Nil {
			m.ID = uuid.New()
		}
		key, err := c.bulk.Put(ctx, m.TaskID, m.ID, m.PayloadInline)
		if err != nil {
			c.logger.Warn("trace: bulk upload failed; keeping inline",
				zap.String("task_id", m.TaskID.String()),
				zap.String("msg_id", m.ID.String()),
				zap.Int("size", m.PayloadSizeBytes), zap.Error(err))
			continue
		}
		m.PayloadObjectKey = key
		m.PayloadInline = "" // 清空，PG 只存元数据
	}
}

// tallyCounts 按 task_id 聚合累加 message_count。
func (c *CaptureConsumer) tallyCounts(ctx context.Context, batch []*Message) {
	counts := map[string]int{}
	taskIDLookup := map[string]*Message{}
	for _, m := range batch {
		key := m.TaskID.String()
		counts[key]++
		taskIDLookup[key] = m
	}
	for key, delta := range counts {
		m := taskIDLookup[key]
		if err := c.repo.IncrementMessageCount(ctx, m.TaskID, delta); err != nil {
			c.logger.Warn("trace: increment message_count failed",
				zap.String("task_id", key), zap.Error(err))
		}
	}
}

// FlushedCount 已落库累计。
func (c *CaptureConsumer) FlushedCount() uint64 { return atomic.LoadUint64(&c.flushed) }

// DroppedCount 因 decode 或 DB 失败丢弃的累计。
func (c *CaptureConsumer) DroppedCount() uint64 { return atomic.LoadUint64(&c.dropped) }
