package task

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// ExpiredSweeper 周期扫描 expires_at < now 且仍处于活跃态（pending/sent）的任务，
// 将其标记为 expired 并广播 task 终态事件（T-0157 C2）。
//
// 设计要点：
//   - 接口注入 (ExpiredCandidatesLister + TaskExpirer)，便于单测用 mock 不依赖 PG/Redis
//   - SweepOnce 暴露同步单轮扫描接口，供单测直接断言；Run 包装 Ticker 循环
//   - 错误不中断扫描：单条 Expire 失败仅记 warn，继续处理下一条
//   - 批量上限 batchSize 防 worker 长事务；剩余项下一轮处理
type ExpiredSweeper struct {
	lister    ExpiredCandidatesLister
	expirer   TaskExpirer
	interval  time.Duration
	batchSize int
	logger    *zap.Logger
}

// ExpiredCandidatesLister 抽象 repo.ListExpiredCandidates，便于单测。
type ExpiredCandidatesLister interface {
	ListExpiredCandidates(ctx context.Context, now time.Time, limit int) ([]*Task, error)
}

// TaskExpirer 抽象 service.ExpireTask，便于单测。
type TaskExpirer interface {
	ExpireTask(ctx context.Context, task *Task) error
}

// NewExpiredSweeper 创建 ExpiredSweeper。
//
// interval <= 0 或 batchSize <= 0 时使用安全默认值（10s / 100）。
// logger 为 nil 时使用 zap.NewNop()，但不会发生在 worker 主进程中。
func NewExpiredSweeper(
	lister ExpiredCandidatesLister,
	expirer TaskExpirer,
	interval time.Duration,
	batchSize int,
	logger *zap.Logger,
) *ExpiredSweeper {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ExpiredSweeper{
		lister:    lister,
		expirer:   expirer,
		interval:  interval,
		batchSize: batchSize,
		logger:    logger.Named("task-expired-sweeper"),
	}
}

// Run 阻塞循环执行 SweepOnce 直到 ctx.Done()。
// 首次扫描在 ctx Start 后第一个 interval tick 触发（避免与启动期其他初始化抢资源）。
func (s *ExpiredSweeper) Run(ctx context.Context) {
	s.logger.Info("expired sweeper running",
		zap.Duration("interval", s.interval),
		zap.Int("batch_size", s.batchSize))

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("expired sweeper stopped", zap.Error(ctx.Err()))
			return
		case <-ticker.C:
			if processed, err := s.SweepOnce(ctx); err != nil {
				s.logger.Warn("sweep round failed", zap.Error(err), zap.Int("processed_before_err", processed))
			}
		}
	}
}

// SweepOnce 跑一轮扫描，返回处理（标记 expired）的任务数 + 第一个致命错误。
// 单条 ExpireTask 失败不中断；仅 lister 报错才是致命错误。
func (s *ExpiredSweeper) SweepOnce(ctx context.Context) (int, error) {
	now := time.Now()
	tasks, err := s.lister.ListExpiredCandidates(ctx, now, s.batchSize)
	if err != nil {
		return 0, err
	}
	if len(tasks) == 0 {
		return 0, nil
	}

	processed := 0
	for _, t := range tasks {
		if t == nil {
			continue
		}
		if err := s.expirer.ExpireTask(ctx, t); err != nil {
			s.logger.Warn("expire task failed",
				zap.String("task_id", t.ID),
				zap.String("device_sn", t.DeviceSN),
				zap.Error(err))
			continue
		}
		processed++
	}

	if processed > 0 {
		s.logger.Info("expired tasks swept",
			zap.Int("processed", processed),
			zap.Int("candidates", len(tasks)))
	}
	return processed, nil
}
