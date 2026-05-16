package trace

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

// payloadFetcher 抽象大报文回拉（只读），避免 service 硬依赖 minio-go。
// 实际由 BulkStore 实现。
type payloadFetcher interface {
	Get(ctx context.Context, key string) (string, error)
}

// Service trace 业务层。
//   - 任务 CRUD（创建会自动覆盖旧 running 任务）+ 状态变化触发 NATS 事件（M2 起）
//   - 报文 sink（仅 ACS 进程注入 EventBus 时走 JetStream Publish；否则降级本地 channel + flusher 兼容 M1 测试）
//   - 白名单查询：ACS 启动加载 + 兜底对账使用 ListRunningSNs（M2 由 NATS 实时同步主导）
type Service struct {
	repo    Repository
	bus     event.EventBus // nil 表示本地模式（M1 兼容 / 单测）
	fetcher payloadFetcher // 注入后大报文外置由此读回；nil 时 external 报文返回 object_key 让前端自行处理
	metrics *Metrics       // M3-01：可 nil
	logger  *zap.Logger

	// captureQueue 本地模式 fallback：仅当 bus == nil 时启用 channel + flusher。
	// bus != nil 时 EnqueueCapture 改走 bus.Publish(SubjectTraceMessageCaptured)。
	captureQueue chan *Message
	flushSize    int
	flushDelay   time.Duration

	dropped uint64 // 队列满（或 publish 失败）时丢弃计数（Prometheus 在 M3 接入）

	shutdownOnce sync.Once
	doneCh       chan struct{}
}

// Config service 配置。
type Config struct {
	// QueueSize 内部 channel 容量。默认 4096。
	QueueSize int
	// FlushSize 批量写库阈值。默认 64。
	FlushSize int
	// FlushDelay 去抖动窗口。默认 500ms。
	FlushDelay time.Duration
}

// DefaultConfig 默认配置。
func DefaultConfig() Config {
	return Config{QueueSize: 4096, FlushSize: 64, FlushDelay: 500 * time.Millisecond}
}

// NewService 构造函数；调用方需调 Start 启动 flusher goroutine（仅本地模式需要）。
func NewService(repo Repository, cfg Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 4096
	}
	if cfg.FlushSize <= 0 {
		cfg.FlushSize = 64
	}
	if cfg.FlushDelay <= 0 {
		cfg.FlushDelay = 500 * time.Millisecond
	}
	return &Service{
		repo:         repo,
		logger:       logger,
		captureQueue: make(chan *Message, cfg.QueueSize),
		flushSize:    cfg.FlushSize,
		flushDelay:   cfg.FlushDelay,
		doneCh:       make(chan struct{}),
	}
}

// SetPayloadFetcher 注入大报文 MinIO 读回器（M2-06）。
// 推荐传入 *BulkStore；nil 表示禁用（external 报文将仅返回 object_key 元信息）。
func (s *Service) SetPayloadFetcher(f payloadFetcher) {
	s.fetcher = f
}

// LoadMessagePayload 拉取单条报文的完整 payload。
//   - inline：直接返回 PayloadInline
//   - external：通过 BulkStore 拉 MinIO；fetcher 未注入时返回 ErrInvalidInput
//
// 注意：调用方需先 GetTask 验证 task_id 属主 / 权限 — 当前 handler 信任 task_id 同所有人。
func (s *Service) LoadMessagePayload(ctx context.Context, m *Message) (string, error) {
	if m == nil {
		return "", commonerrors.ErrNotFound
	}
	if m.PayloadInline != "" {
		return m.PayloadInline, nil
	}
	if m.PayloadObjectKey == "" {
		return "", nil // empty body
	}
	if s.fetcher == nil {
		return "", fmt.Errorf("%w: external payload requires BulkStore", commonerrors.ErrUnavailable)
	}
	return s.fetcher.Get(ctx, m.PayloadObjectKey)
}

// SetMetrics 注入 Prometheus 指标（M3-01）。
func (s *Service) SetMetrics(m *Metrics) { s.metrics = m }

// ObserveCaptureLatency 由 ACS hook 调用，记录单次拦截耗时（M3-01 反例监控）。
// 单位：秒。P99 反例阈值 0.005s（5ms）。
func (s *Service) ObserveCaptureLatency(d time.Duration) {
	if s.metrics == nil {
		return
	}
	s.metrics.CaptureLatencySeconds.Observe(d.Seconds())
}

// SetEventBus 注入 NATS EventBus，开启跨进程模式（M2）。
// 注入后：
//   - 任务生命周期变化（CreateTask / StopTask）会自动 publish trace.task.* 事件
//   - EnqueueCapture 改走 JetStream（trace.message.captured），不再写本地 channel
func (s *Service) SetEventBus(bus event.EventBus) {
	s.bus = bus
}

// Start 启动 flusher goroutine（仅本地模式生效；bus != nil 时为 no-op）。
func (s *Service) Start(ctx context.Context) {
	if s.bus != nil {
		// JetStream 模式：报文由 ACS publish → worker consume，本进程不需 flusher
		close(s.doneCh)
		return
	}
	go s.flusher(ctx)
}

// Stop 停止 flusher，flush 残留消息。
func (s *Service) Stop() {
	s.shutdownOnce.Do(func() {
		if s.bus != nil {
			// JetStream 模式无 flusher，doneCh 已由 Start 关闭
			return
		}
		close(s.captureQueue)
		<-s.doneCh
	})
}

// DroppedCount 队列满时丢弃报文数（M3 Prometheus 接入）。
func (s *Service) DroppedCount() uint64 {
	return atomic.LoadUint64(&s.dropped)
}

// CreateTask 创建新任务；同 SN 已有 running 时先 stop 旧任务再开新（与老 OMC 行为对齐）。
func (s *Service) CreateTask(ctx context.Context, req CreateTaskRequest, operatorCode, createdBy string) (*Task, error) {
	if req.DeviceSN == "" {
		return nil, fmt.Errorf("%w: device_sn required", commonerrors.ErrInvalidInput)
	}
	duration := req.DurationMinutes
	if duration <= 0 {
		duration = DefaultDurationMinutes
	}
	if duration > MaxDurationMinutes {
		return nil, fmt.Errorf("%w: duration_minutes must be <= %d", commonerrors.ErrInvalidInput, MaxDurationMinutes)
	}
	if operatorCode == "" {
		operatorCode = "default"
	}
	if createdBy == "" {
		createdBy = "system"
	}

	// 1) 同 SN 已存在 running 则先停掉
	old, err := s.repo.GetRunningTaskBySN(ctx, req.DeviceSN)
	if err != nil && !errors.Is(err, commonerrors.ErrNotFound) {
		return nil, fmt.Errorf("check existing running task: %w", err)
	}
	if old != nil {
		if err := s.repo.UpdateTaskStatus(ctx, old.ID, TaskStatusStopped); err != nil {
			return nil, fmt.Errorf("stop existing running task: %w", err)
		}
		s.logger.Info("trace: existing running task stopped before new",
			zap.String("old_task_id", old.ID.String()),
			zap.String("device_sn", req.DeviceSN))
		// 发 stopped 事件让 ACS 实例释放旧 SN（不会立刻：旧 task_id 在白名单里，
		// 后续 capture 仍命中；M2 用 task_id 而非 sn 作为白名单 key 会更精确。
		// 当前用 sn 简化：旧 task stopped 事件 + 立刻 started 事件，结果是 SN 仍在白名单 — 期望行为）
		stale, _ := s.repo.GetTask(ctx, old.ID)
		if stale != nil {
			s.publishTaskEvent(ctx, event.SubjectTraceTaskStopped, stale, "new_task_replaced")
		}
	}

	now := time.Now()
	task := &Task{
		DeviceSN:     req.DeviceSN,
		OperatorCode: operatorCode,
		Status:       TaskStatusRunning,
		StartTime:    now,
		ExpiresAt:    now.Add(time.Duration(duration) * time.Minute),
		CreatedBy:    createdBy,
	}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	s.logger.Info("trace: task created",
		zap.String("task_id", task.ID.String()),
		zap.String("device_sn", task.DeviceSN),
		zap.Duration("duration", time.Duration(duration)*time.Minute))
	s.publishTaskEvent(ctx, event.SubjectTraceTaskStarted, task, "")
	return task, nil
}

// GetTask 查任务详情。
func (s *Service) GetTask(ctx context.Context, id uuid.UUID) (*Task, error) {
	return s.repo.GetTask(ctx, id)
}

// GetActiveTaskBySN 给设备详情页用：返回该 SN 当前的 running 任务（无则 nil + ErrNotFound）。
func (s *Service) GetActiveTaskBySN(ctx context.Context, sn string) (*Task, error) {
	return s.repo.GetRunningTaskBySN(ctx, sn)
}

// ListTasks 任务列表。
func (s *Service) ListTasks(ctx context.Context, filter TaskFilter) (*model.ListResponse[Task], error) {
	return s.repo.ListTasks(ctx, filter)
}

// StopTask 停止任务；purge=true 时同时请求清理报文。
//
// M2 行为：
//   - status running → stopped 后发 trace.task.stopped 事件，ACS 实例订阅后实时移除 SN
//   - purge=true 时另发 trace.task.purged 事件，worker 订阅后异步 DELETE + 清 MinIO 对象
//     状态先转 purged（前端立即可见），实际数据清理可异步完成；M1 同步 DELETE 在 M2 仍保留
//     作为 worker 不可达 fallback，但默认走异步路径。
func (s *Service) StopTask(ctx context.Context, id uuid.UUID, purge bool) (*Task, error) {
	t, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Status == TaskStatusRunning {
		if err := s.repo.UpdateTaskStatus(ctx, id, TaskStatusStopped); err != nil {
			return nil, err
		}
		stopped, _ := s.repo.GetTask(ctx, id)
		if stopped != nil {
			s.publishTaskEvent(ctx, event.SubjectTraceTaskStopped, stopped, "manual")
		}
	}
	if purge {
		// 先转状态让前端立即看到；实际 DELETE 由 worker 异步执行（订阅 trace.task.purged）。
		if err := s.repo.UpdateTaskStatus(ctx, id, TaskStatusPurged); err != nil {
			return nil, err
		}
		// 没有 NATS 时退化为同步 DELETE，保持单进程模式工作
		if s.bus == nil {
			if err := s.repo.PurgeTaskMessages(ctx, id); err != nil {
				return nil, fmt.Errorf("purge messages: %w", err)
			}
		}
		purged, _ := s.repo.GetTask(ctx, id)
		if purged != nil {
			s.publishTaskEvent(ctx, event.SubjectTraceTaskPurged, purged, "manual")
		}
	}
	// 返回最新状态
	updated, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("trace: task stopped",
		zap.String("task_id", id.String()),
		zap.Bool("purge", purge),
		zap.String("status", string(updated.Status)))
	return updated, nil
}

// publishTaskEvent 发 trace.task.* 事件到 NATS（bus 为 nil 时静默 no-op）。
func (s *Service) publishTaskEvent(ctx context.Context, subject string, t *Task, reason string) {
	if s.bus == nil || t == nil {
		return
	}
	evt, err := event.NewEvent(subject, ToTaskEvent(t, reason))
	if err != nil {
		s.logger.Warn("trace: build task event failed",
			zap.String("subject", subject), zap.Error(err))
		return
	}
	if err := s.bus.Publish(ctx, subject, evt); err != nil {
		s.logger.Warn("trace: publish task event failed",
			zap.String("subject", subject),
			zap.String("task_id", t.ID.String()),
			zap.Error(err))
	}
}

// ListMessages 按任务查报文。
func (s *Service) ListMessages(ctx context.Context, filter MessageFilter) (*model.ListResponse[Message], error) {
	return s.repo.ListMessages(ctx, filter)
}

// GetMessage 查询单条报文（含 inline payload 或 object_key 元数据）。
func (s *Service) GetMessage(ctx context.Context, taskID, msgID uuid.UUID) (*Message, error) {
	return s.repo.GetMessage(ctx, taskID, msgID)
}

// ---------- 异步导出 (M2-08) ----------

// RequestExport 用户触发异步导出。
//   - 校验 task 存在
//   - 创建 trace_export_jobs 行（status=queued）
//   - 发 trace.export.requested 事件，worker 消费后生成 XML
//
// 没有 bus（本地模式）时不发事件，调用方需另行驱动 exporter。
func (s *Service) RequestExport(ctx context.Context, taskID uuid.UUID, requestedBy string) (*ExportJob, error) {
	if _, err := s.repo.GetTask(ctx, taskID); err != nil {
		return nil, err
	}
	if requestedBy == "" {
		requestedBy = "system"
	}
	job := &ExportJob{
		TaskID:      taskID,
		RequestedBy: requestedBy,
		Status:      ExportJobQueued,
	}
	if err := s.repo.CreateExportJob(ctx, job); err != nil {
		return nil, err
	}
	if s.bus != nil {
		evt, err := event.NewEvent(event.SubjectTraceExportRequested, ExportRequestedEvent{
			JobID:  job.ID,
			TaskID: taskID,
		})
		if err == nil {
			if pubErr := s.bus.Publish(ctx, event.SubjectTraceExportRequested, evt); pubErr != nil {
				s.logger.Warn("trace: publish export.requested failed",
					zap.String("job_id", job.ID.String()), zap.Error(pubErr))
			}
		}
	}
	s.logger.Info("trace: export job requested",
		zap.String("job_id", job.ID.String()),
		zap.String("task_id", taskID.String()))
	return job, nil
}

// GetExportJob 查任务状态。
func (s *Service) GetExportJob(ctx context.Context, id uuid.UUID) (*ExportJob, error) {
	return s.repo.GetExportJob(ctx, id)
}

// UpdateExportJob 透传给 repo（worker exporter 写状态用）。
func (s *Service) UpdateExportJob(ctx context.Context, job *ExportJob) error {
	return s.repo.UpdateExportJob(ctx, job)
}

// ListRunningSNs 暴露给 ACS 侧白名单加载使用。
func (s *Service) ListRunningSNs(ctx context.Context) (map[string]uuid.UUID, error) {
	return s.repo.ListRunningSNs(ctx)
}

// EnqueueCapture 由 ACS hook 调用，非阻塞投递；队列满 / Publish 失败时丢弃并计数。
//
// 投递路径：
//   - bus != nil（M2 生产）：JetStream Publish trace.message.captured，worker 群组消费
//   - bus == nil（M1 兼容 / 单测）：本地 channel + flusher 批量落库
//
// 调用方保证 payload 已按 MaxInlinePayloadBytes 处理；msg.CapturedAt 为空时此处填充。
func (s *Service) EnqueueCapture(msg *Message) {
	if msg == nil {
		return
	}
	if msg.CapturedAt.IsZero() {
		msg.CapturedAt = time.Now()
	}
	if s.bus != nil {
		s.publishCapture(msg)
		return
	}
	select {
	case s.captureQueue <- msg:
	default:
		atomic.AddUint64(&s.dropped, 1)
		if s.metrics != nil {
			s.metrics.MessagesDroppedTotal.WithLabelValues(DropReasonQueueFull).Inc()
		}
		s.logger.Warn("trace: capture queue full, message dropped",
			zap.String("task_id", msg.TaskID.String()),
			zap.String("device_sn", msg.DeviceSN))
	}
}

// publishCapture JetStream 发送一条报文事件。
// 非阻塞：失败计入 dropped，不阻塞 ACS hot path。
func (s *Service) publishCapture(msg *Message) {
	evt, err := event.NewEvent(event.SubjectTraceMessageCaptured, msg)
	if err != nil {
		atomic.AddUint64(&s.dropped, 1)
		s.logger.Warn("trace: build capture event failed", zap.Error(err))
		return
	}
	// 用短超时 context — JetStream Publish 是同步 ACK，应在 ms 级完成
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := s.bus.Publish(ctx, event.SubjectTraceMessageCaptured, evt); err != nil {
		atomic.AddUint64(&s.dropped, 1)
		if s.metrics != nil {
			s.metrics.MessagesDroppedTotal.WithLabelValues(DropReasonPublishFailed).Inc()
		}
		s.logger.Warn("trace: publish capture event failed",
			zap.String("task_id", msg.TaskID.String()),
			zap.String("device_sn", msg.DeviceSN),
			zap.Error(err))
	}
}

// flusher 批量写库 goroutine；ctx Done 或 queue close 时退出。
func (s *Service) flusher(ctx context.Context) {
	defer close(s.doneCh)
	batch := make([]*Message, 0, s.flushSize)
	ticker := time.NewTicker(s.flushDelay)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// 用独立的 context 避免父 ctx 取消后丢失最后一批
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.repo.InsertMessages(bgCtx, batch); err != nil {
			s.logger.Error("trace: batch insert failed",
				zap.Int("size", len(batch)), zap.Error(err))
		} else {
			s.tallyMessageCounts(bgCtx, batch)
		}
		cancel()
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case msg, ok := <-s.captureQueue:
			if !ok {
				flush()
				return
			}
			batch = append(batch, msg)
			if len(batch) >= s.flushSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// tallyMessageCounts 按 task_id 聚合后累加 trace_tasks.message_count；失败仅日志（非关键路径）。
func (s *Service) tallyMessageCounts(ctx context.Context, batch []*Message) {
	counts := map[uuid.UUID]int{}
	for _, m := range batch {
		counts[m.TaskID]++
	}
	for taskID, delta := range counts {
		if err := s.repo.IncrementMessageCount(ctx, taskID, delta); err != nil {
			s.logger.Warn("trace: increment message_count failed",
				zap.String("task_id", taskID.String()), zap.Error(err))
		}
	}
}
