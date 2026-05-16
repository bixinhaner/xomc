package trace

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/omcgo/omcgo/internal/core/event"
)

// Sweeper 在 worker 进程周期巡检过期任务并 stopped；
// 同时订阅 trace.task.purged 异步执行报文物理清理（DELETE + MinIO 对象删除）。
//
// 设计文档：§7.3 worker 职责。
type Sweeper struct {
	repo     Repository
	bus      event.EventBus
	bulk     BulkObjectDeleter // 可空 — 没有 MinIO 时只删 PG
	metrics  *Metrics          // M3-01：可 nil
	logger   *zap.Logger
	interval time.Duration

	shutdownOnce sync.Once
	stopCh       chan struct{}
	doneCh       chan struct{}

	stopped uint64 // 累计超时停止数
	purged  uint64 // 累计 purge 完成数
}

// BulkObjectDeleter 抽象 MinIO bucket 对象删除（避免 trace 包硬依赖 minio-go）。
// M2-06 大报文外置启用时由 cmd/worker 注入；nil 时跳过 MinIO 清理。
type BulkObjectDeleter interface {
	// DeleteByPrefix 删除 bucket 下指定前缀的所有对象（用于 task purge）。
	DeleteByPrefix(ctx context.Context, prefix string) error
}

// SweeperConfig 配置。
type SweeperConfig struct {
	Interval time.Duration // 默认 60s
}

// DefaultSweeperConfig 默认配置。
func DefaultSweeperConfig() SweeperConfig {
	return SweeperConfig{Interval: 60 * time.Second}
}

// NewSweeper 构造函数。
func NewSweeper(repo Repository, bus event.EventBus, bulk BulkObjectDeleter, cfg SweeperConfig, logger *zap.Logger) *Sweeper {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 60 * time.Second
	}
	return &Sweeper{
		repo:     repo,
		bus:      bus,
		bulk:     bulk,
		logger:   logger.Named("trace-sweeper"),
		interval: cfg.Interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start 启动巡检 goroutine + 订阅 purge 事件。
// 返回 unsubscribe cleanup，调用方在进程关闭时执行。
func (s *Sweeper) Start(ctx context.Context) (func(), error) {
	var unsub func()
	if s.bus != nil {
		sub, err := s.bus.Subscribe(event.SubjectTraceTaskPurged, s.handlePurgeEvent)
		if err != nil {
			return nil, err
		}
		unsub = func() { _ = sub.Unsubscribe() }
		s.logger.Info("trace sweeper subscribed to trace.task.purged")
	}
	go s.loop(ctx)
	cleanup := func() {
		if unsub != nil {
			unsub()
		}
		s.Stop()
	}
	return cleanup, nil
}

// SetMetrics 注入 Prometheus 指标。
func (s *Sweeper) SetMetrics(m *Metrics) { s.metrics = m }

// Stop 停止巡检。
func (s *Sweeper) Stop() {
	s.shutdownOnce.Do(func() {
		close(s.stopCh)
		<-s.doneCh
	})
}

// StoppedCount 累计已 stopped。
func (s *Sweeper) StoppedCount() uint64 { return atomic.LoadUint64(&s.stopped) }

// PurgedCount 累计已 purged。
func (s *Sweeper) PurgedCount() uint64 { return atomic.LoadUint64(&s.purged) }

func (s *Sweeper) loop(ctx context.Context) {
	defer close(s.doneCh)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.sweepExpired(ctx)
			s.sampleActiveTasks(ctx)
			s.sampleStorageBytes(ctx)
		}
	}
}

// sampleActiveTasks 把当前 running 任务数采样到 Prometheus Gauge。
func (s *Sweeper) sampleActiveTasks(ctx context.Context) {
	if s.metrics == nil {
		return
	}
	sCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	snapshot, err := s.repo.ListRunningSNs(sCtx)
	if err != nil {
		return
	}
	s.metrics.ActiveTasks.Set(float64(len(snapshot)))
}

// sampleStorageBytes 把 trace_messages 的 inline / minio 字节总量采样到 Gauge。
// 走 SUM(payload_size_bytes) FILTER 一次查询，分区表上是常量扫描时间。
func (s *Sweeper) sampleStorageBytes(ctx context.Context) {
	if s.metrics == nil {
		return
	}
	sCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	inlineBytes, minioBytes, err := s.repo.StorageStats(sCtx)
	if err != nil {
		s.logger.Warn("trace sweeper: storage stats failed", zap.Error(err))
		return
	}
	s.metrics.StorageBytesTotal.WithLabelValues(StorageInline).Set(float64(inlineBytes))
	s.metrics.StorageBytesTotal.WithLabelValues(StorageMinIO).Set(float64(minioBytes))
}

// sweepExpired 扫一批过期任务转 stopped + 发事件。
func (s *Sweeper) sweepExpired(ctx context.Context) {
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tasks, err := s.repo.ListExpired(scanCtx, 100)
	if err != nil {
		s.logger.Warn("trace sweeper: list expired failed", zap.Error(err))
		return
	}
	for _, t := range tasks {
		t := t
		if err := s.repo.UpdateTaskStatus(scanCtx, t.ID, TaskStatusStopped); err != nil {
			s.logger.Warn("trace sweeper: update stopped failed",
				zap.String("task_id", t.ID.String()), zap.Error(err))
			continue
		}
		atomic.AddUint64(&s.stopped, 1)
		s.logger.Info("trace sweeper: task expired → stopped",
			zap.String("task_id", t.ID.String()),
			zap.String("device_sn", t.DeviceSN),
			zap.Time("expires_at", t.ExpiresAt))
		// L-10：自动停止也走 audit，运营侧追溯时能看到事件。actor=system 区别于手动操作。
		audit.Log(scanCtx, audit.Entry{
			Username:     "system",
			Action:       "trace_stop",
			ResourceType: "trace_task",
			ResourceID:   t.ID.String(),
			Details: map[string]interface{}{
				"device_sn":  t.DeviceSN,
				"reason":     "sweeper_expired",
				"expires_at": t.ExpiresAt,
			},
			UserAgent: "omcgo-worker/trace-sweeper",
			Success:   true,
		})
		if s.bus != nil {
			stoppedTask := t
			stoppedTask.Status = TaskStatusStopped
			now := time.Now()
			stoppedTask.StoppedAt = &now
			evt, evtErr := event.NewEvent(event.SubjectTraceTaskStopped, ToTaskEvent(&stoppedTask, "timeout"))
			if evtErr == nil {
				if pubErr := s.bus.Publish(scanCtx, event.SubjectTraceTaskStopped, evt); pubErr != nil {
					s.logger.Warn("trace sweeper: publish stopped event failed",
						zap.String("task_id", t.ID.String()), zap.Error(pubErr))
				}
			}
		}
	}
}

// handlePurgeEvent 异步执行 PG DELETE + MinIO 对象清理。
func (s *Sweeper) handlePurgeEvent(_ context.Context, evt event.Event) error {
	var payload TaskEvent
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("trace sweeper: decode purge event failed", zap.Error(err))
		return nil
	}
	if payload.TaskID == uuid.Nil {
		return nil
	}
	pCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1) MinIO 删 trace-bulk bucket 下 {task_id}/ 前缀对象（M2-06 启用时才有内容）
	if s.bulk != nil {
		prefix := payload.TaskID.String() + "/"
		if err := s.bulk.DeleteByPrefix(pCtx, prefix); err != nil {
			s.logger.Warn("trace sweeper: minio purge failed",
				zap.String("task_id", payload.TaskID.String()), zap.Error(err))
			// 继续 PG 删除 — 部分清理优于全失败
		}
	}
	// 2) PG DELETE trace_messages
	if err := s.repo.PurgeTaskMessages(pCtx, payload.TaskID); err != nil {
		s.logger.Warn("trace sweeper: pg purge failed",
			zap.String("task_id", payload.TaskID.String()), zap.Error(err))
		return nil
	}
	atomic.AddUint64(&s.purged, 1)
	s.logger.Info("trace sweeper: task purged",
		zap.String("task_id", payload.TaskID.String()),
		zap.String("device_sn", payload.DeviceSN))
	return nil
}
