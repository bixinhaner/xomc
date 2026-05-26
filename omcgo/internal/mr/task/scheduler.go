package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/internal/mr"
)

// CellMappingSource — DEPRECATED 2026-05-26：cell 维度已下线，scheduler 直接
// 按 task.TargetDeviceSNs 一对一写 progress。保留接口空壳只为 SetMappings 兼容签名，
// 调用 SetMappings 现已是 no-op；下个清理 PR 一并删 SetMappings 与本接口。
type CellMappingSource interface {
	List(ctx context.Context, filter mr.MappingFilter) (*model.ListResponse[mr.MRDeviceMapping], error)
}

// Scheduler 在 worker 进程内跑三件事：
//  1. 扫描 task_status='waitting' 且 start_time<=now 的任务 → 调 dispatcher.Open
//  2. 扫描 task_status∈{'on','termination'} 且 end_time/手动停止已触发的任务 → 调 dispatcher.Close
//  3. 巡检所有活跃 cell 的 Redis 心跳，超阈值未命中标 abnormal
//
// 多 worker 部署用 Redis SETNX 互斥（lockKey TTL = interval × 3，避免一个 tick 跨过下一个）。
// 单进程部署也安全：锁会自动续期失败时由另一进程接管。
type Scheduler struct {
	repo       Repository
	dispatcher *Dispatcher
	redisCli   redis.UniversalClient
	mappings   CellMappingSource // 可 nil；nil 时 tickOpen 不生成任何 progress（任务空跑）
	logger     *zap.Logger
	interval   time.Duration
	missThresh int
	metrics    *Metrics // 可 nil

	cron     *cron.Cron
	entryIDs []cron.EntryID
	mu       sync.Mutex
	started  bool
}

// SetMetrics 注入 Prometheus 指标（可选）。
func (s *Scheduler) SetMetrics(m *Metrics) { s.metrics = m }

// SetMappings 注入 cell 枚举源。tickOpen 时按 enabled=true 拉一遍写入 progress。
func (s *Scheduler) SetMappings(src CellMappingSource) { s.mappings = src }

// SchedulerConfig 控制 Scheduler 调度行为。
type SchedulerConfig struct {
	IntervalSeconds        int
	HeartbeatMissThreshold int
	BatchSize              int
}

// Defaults 给出 SchedulerConfig 的合理默认值（PRD §3 AC-2 / §7 度量目标）。
func (c SchedulerConfig) Defaults() SchedulerConfig {
	out := c
	if out.IntervalSeconds <= 0 {
		out.IntervalSeconds = 30
	}
	if out.HeartbeatMissThreshold <= 0 {
		out.HeartbeatMissThreshold = 2
	}
	if out.BatchSize <= 0 {
		out.BatchSize = 100
	}
	return out
}

// NewScheduler 创建调度器。redisCli 不能为 nil（用于分布式锁与心跳查询）。
func NewScheduler(
	repo Repository,
	dispatcher *Dispatcher,
	redisCli redis.UniversalClient,
	cfg SchedulerConfig,
	logger *zap.Logger,
) *Scheduler {
	if logger == nil {
		logger = zap.NewNop()
	}
	c := cfg.Defaults()
	return &Scheduler{
		repo:       repo,
		dispatcher: dispatcher,
		redisCli:   redisCli,
		logger:     logger.Named("mr-scheduler"),
		interval:   time.Duration(c.IntervalSeconds) * time.Second,
		missThresh: c.HeartbeatMissThreshold,
	}
}

// Start 注册三个 cron entry：开任务 / 关任务 / 心跳巡检。幂等。
//
// 三个独立 entry 而不是合一个 tick：让指标 / 日志 / 失败影响域分开，
// 一个分支挂掉不影响其它（与 license/topology/admin 现有 cron 风格一致）。
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	s.cron = cron.New(cron.WithLocation(time.UTC))

	spec := fmt.Sprintf("@every %s", s.interval)
	openID, err := s.cron.AddFunc(spec, func() { s.runWithLock(ctx, "open", s.tickOpen) })
	if err != nil {
		return fmt.Errorf("register mr-open cron: %w", err)
	}
	closeID, err := s.cron.AddFunc(spec, func() { s.runWithLock(ctx, "close", s.tickClose) })
	if err != nil {
		return fmt.Errorf("register mr-close cron: %w", err)
	}
	heartbeatID, err := s.cron.AddFunc(spec, func() { s.runWithLock(ctx, "heartbeat", s.tickHeartbeat) })
	if err != nil {
		return fmt.Errorf("register mr-heartbeat cron: %w", err)
	}
	s.entryIDs = []cron.EntryID{openID, closeID, heartbeatID}
	s.cron.Start()
	s.started = true
	s.logger.Info("mr scheduler started",
		zap.Duration("interval", s.interval),
		zap.Int("miss_threshold", s.missThresh),
	)
	return nil
}

// Stop 停止 cron。安全幂等。
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started || s.cron == nil {
		return
	}
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.cron = nil
	s.started = false
	s.logger.Info("mr scheduler stopped")
}

// ---------- 分布式锁 ----------

// runWithLock 包装一次 tick：先抢 Redis SETNX 锁，抢到才执行 body。
// 没抢到说明另一 worker 正在跑同一分支，本次安静跳过（不算错误）。
//
// 每个 tick 起一个 root span "mr.task.scheduler.tick"，让 trace 视图能看到
// "调度→下发→设备应答→进度回写"完整链路（dispatcher.Open/Close 的子 span
// 通过 ctx 自动挂在本 span 下）。
func (s *Scheduler) runWithLock(ctx context.Context, branch string, body func(context.Context)) {
	ctx, span := tracing.StartSpan(ctx, tracing.MRTaskTracerName, "mr.task.scheduler.tick",
		attribute.String("mr.branch", branch),
	)
	defer span.End()

	if s.redisCli == nil {
		body(ctx) // 单进程退化路径（如单测）
		return
	}
	lockKey := "mr:scheduler:lock:" + branch
	lockTTL := s.interval * 3
	if lockTTL < 30*time.Second {
		lockTTL = 30 * time.Second
	}
	ok, err := s.redisCli.SetNX(ctx, lockKey, time.Now().Format(time.RFC3339), lockTTL).Result()
	if err != nil {
		s.logger.Warn("mr scheduler lock acquire failed",
			zap.String("branch", branch), zap.Error(err))
		// Redis 临时故障：保守起见跳过本次（避免重复下发）；下一 tick 自然重试
		return
	}
	if !ok {
		s.logger.Debug("mr scheduler lock held by another worker", zap.String("branch", branch))
		return
	}
	defer func() {
		// 释放锁：用 DEL 即可（短窗口，无需 lua 校验持有者）。
		// 锁 TTL 兜底防进程崩溃留死锁。
		if _, err := s.redisCli.Del(context.Background(), lockKey).Result(); err != nil {
			s.logger.Debug("mr scheduler lock release failed", zap.Error(err))
		}
	}()
	body(ctx)
}

// ---------- 三个 tick ----------

// tickOpen 处理 waitting → on 转移。
//
// 同一任务的所有 cell 在同一 tick 内派发，避免"半个 cell open 完，再下一个 tick
// 来扫描时把任务状态推到 on 但还有 cell 在排队"的窗口期问题。
func (s *Scheduler) tickOpen(ctx context.Context) {
	tasks, err := s.repo.ListDueWaitingTasks(ctx, 100)
	if err != nil {
		s.logger.Error("list due waiting tasks failed", zap.Error(err))
		return
	}
	for _, t := range tasks {
		s.openOneTask(ctx, t)
	}
}

func (s *Scheduler) openOneTask(ctx context.Context, t Task) {
	// 1) 把任务推到 on，避免下一 tick 重复触发；如果中途宕机，新 worker 接管
	// 时会看到 on 状态 + pending progress，是健康的可恢复中间态。
	if err := s.repo.UpdateTaskStatus(ctx, t.TaskID, StatusOn, nil); err != nil {
		s.logger.Error("update task status to on failed",
			zap.String("task_id", t.TaskID.String()), zap.Error(err))
		return
	}

	// 2) 简化模型（2026-05-26）：device 直接作为 SPV 单位，不再走 cell 维度。
	// 一对一把 target_device_sns 写入 progress（small_cell_code 字段复用为 device_sn 占位 —
	// 表名不变是为了避免 schema 大迁移；语义上现在是 device-level progress）。
	if len(t.TargetDeviceSNs) > 0 {
		targets := make([]CellTarget, 0, len(t.TargetDeviceSNs))
		for _, sn := range t.TargetDeviceSNs {
			targets = append(targets, CellTarget{
				SmallCellCode: sn, // 复用列存设备 SN
				SerialNumber:  sn,
			})
		}
		if err := s.repo.InsertProgressRows(ctx, t.TaskID, targets); err != nil {
			s.logger.Error("insert progress rows failed",
				zap.String("task_id", t.TaskID.String()), zap.Error(err))
			// 不 return — 让现有 progress 继续派发
		}
	}

	// 3) 分页拉 progress 列表（pending）→ 派发 open SPV
	const batch = 200
	for page := 1; ; page++ {
		resp, err := s.repo.ListProgress(ctx, ProgressListFilter{
			TaskID: t.TaskID, Page: page, PageSize: batch,
		})
		if err != nil {
			s.logger.Error("list progress for open failed",
				zap.String("task_id", t.TaskID.String()), zap.Error(err))
			return
		}
		for _, p := range resp.Items {
			if p.ProgressStatus != ProgressPending {
				continue // 跨 tick 恢复：跳过已处理的 cell
			}
			if err := s.dispatcher.Open(ctx, &t, p); err != nil {
				s.logger.Warn("dispatch open failed",
					zap.String("task_id", t.TaskID.String()),
					zap.String("cell", p.SmallCellCode),
					zap.Error(err))
			}
		}
		if int64(page*batch) >= resp.Total {
			return
		}
	}
}

// tickClose 处理两类需要关闭的任务：
//  1. on + end_time<=now（自动关闭）
//  2. termination（手动 stop 触发，需要把已开启 cell 关掉）
func (s *Scheduler) tickClose(ctx context.Context) {
	// 自动关闭
	dueOn, err := s.repo.ListDueOnTasks(ctx, 100)
	if err != nil {
		s.logger.Error("list due on tasks failed", zap.Error(err))
	} else {
		for _, t := range dueOn {
			s.closeOneTask(ctx, t)
		}
	}
	// 手动停止
	terminating, err := s.repo.ListTasks(ctx, TaskListFilter{
		Status: statusPtr(StatusTermination), PageSize: 100,
	})
	if err != nil {
		s.logger.Error("list terminating tasks failed", zap.Error(err))
		return
	}
	for _, t := range terminating.Items {
		s.closeOneTask(ctx, t)
	}
}

func (s *Scheduler) closeOneTask(ctx context.Context, t Task) {
	const batch = 200
	allClosed := true
	for page := 1; ; page++ {
		resp, err := s.repo.ListProgress(ctx, ProgressListFilter{
			TaskID: t.TaskID, Page: page, PageSize: batch,
		})
		if err != nil {
			s.logger.Error("list progress for close failed",
				zap.String("task_id", t.TaskID.String()), zap.Error(err))
			return
		}
		for _, p := range resp.Items {
			// 仅 openSuccess 的 cell 需要关闭；其余记录已是终止态。
			// Close() 内部会再判断一次（双重保险）。
			if p.ProgressStatus != ProgressOpenSuccess {
				continue
			}
			if err := s.dispatcher.Close(ctx, &t, p); err != nil {
				s.logger.Warn("dispatch close failed",
					zap.String("task_id", t.TaskID.String()),
					zap.String("cell", p.SmallCellCode),
					zap.Error(err))
				allClosed = false
			}
		}
		if int64(page*batch) >= resp.Total {
			break
		}
	}
	// 把任务整体推到 off（即使部分关闭失败也推 off：失败 cell 已记 closeFailure，
	// 用户能看到；不推 off 会导致下一 tick 反复尝试）。
	if allClosed || t.TaskStatus == StatusOn || t.TaskStatus == StatusTermination {
		if err := s.repo.UpdateTaskStatus(ctx, t.TaskID, StatusOff, nil); err != nil {
			s.logger.Warn("update task status to off failed",
				zap.String("task_id", t.TaskID.String()), zap.Error(err))
		}
	}
}

// tickHeartbeat 巡检活跃任务下的 openSuccess cell：
//   - Redis MRFileReport_{cellCode} 存在 → 正常
//   - 不存在 → missed_heartbeat+1，超过阈值 → 标 abnormal
//
// 性能注意：当前实现按 cell 一一 GET Redis；10 万 cell 规模下需要换成
// MGET 批量或本地缓存。此为 Phase 2 MVP，留 TODO 待 Phase 4 压测后优化。
func (s *Scheduler) tickHeartbeat(ctx context.Context) {
	if s.redisCli == nil {
		return
	}
	onTasks, err := s.repo.ListTasks(ctx, TaskListFilter{
		Status: statusPtr(StatusOn), PageSize: 100,
	})
	if err != nil {
		s.logger.Error("list on tasks for heartbeat failed", zap.Error(err))
		return
	}
	// 把 active 任务数同步到 Gauge —— 本 tick 是 scheduler 唯一一个跑得最频繁
	// 又确切知道 on 任务数的地方，借势更新 active_count 指标。
	s.metrics.SetActiveCount(float64(onTasks.Total))
	for _, t := range onTasks.Items {
		s.healthCheckTask(ctx, t)
	}
}

func (s *Scheduler) healthCheckTask(ctx context.Context, t Task) {
	const batch = 200
	openSuccess := ProgressOpenSuccess
	for page := 1; ; page++ {
		resp, err := s.repo.ListProgress(ctx, ProgressListFilter{
			TaskID: t.TaskID, Status: &openSuccess, Page: page, PageSize: batch,
		})
		if err != nil {
			s.logger.Error("list progress for heartbeat failed", zap.Error(err))
			return
		}
		for _, p := range resp.Items {
			key := HeartbeatRedisKey(p.SmallCellCode)
			n, err := s.redisCli.Exists(ctx, key).Result()
			if err != nil {
				s.logger.Debug("redis EXISTS failed", zap.Error(err))
				continue
			}
			if n > 0 {
				// 心跳健康 — TouchHeartbeat 已由 transfer/bridge 在文件落 MinIO 时调用，
				// 这里不再重复更新；只在"缺失"时累加 missed。
				continue
			}
			// 心跳缺失：累计 missed，超阈值标 abnormal
			abnormal := s.markMissedHeartbeat(ctx, p)
			s.metrics.IncHeartbeatMissed()
			if abnormal {
				s.metrics.IncHeartbeatAbnormal()
			}
		}
		if int64(page*batch) >= resp.Total {
			return
		}
	}
}

// markMissedHeartbeat 调 Repository.IncrementMissedHeartbeat 在 PG 端原子 +1，
// 超过阈值时把 health_status 切到 'abnormal'。返回 (becameAbnormal, err) 供
// scheduler 上报指标。
func (s *Scheduler) markMissedHeartbeat(ctx context.Context, p Progress) (becameAbnormal bool) {
	matched, newMissed, abnormal, err := s.repo.IncrementMissedHeartbeat(ctx, p.SmallCellCode, s.missThresh)
	if err != nil {
		s.logger.Warn("increment missed heartbeat failed",
			zap.String("cell", p.SmallCellCode),
			zap.Error(err),
		)
		return false
	}
	if !matched {
		return false
	}
	if abnormal {
		s.logger.Warn("MR cell switched to abnormal (heartbeat missed >= threshold)",
			zap.String("task_id", p.TaskID.String()),
			zap.String("cell", p.SmallCellCode),
			zap.Int("missed_count", newMissed),
			zap.Int("threshold", s.missThresh),
		)
	} else {
		s.logger.Debug("MR cell missed heartbeat (below threshold)",
			zap.String("cell", p.SmallCellCode),
			zap.Int("missed_count", newMissed),
		)
	}
	return abnormal
}

// ---------- 辅助 ----------

func statusPtr(s TaskStatus) *TaskStatus { return &s }

// HeartbeatRedisKey 返回 MR 心跳 key。transfer/bridge.go 和 scheduler 共用此函数
// 避免命名漂移。
func HeartbeatRedisKey(cellCode string) string {
	return "MRFileReport_" + cellCode
}
