package asyncjob

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

// Sweeper 定期扫 async_jobs 找心跳过期的 running 任务，重置回 pending（attempt+1）。
// 设计文档 §4.8：进程崩溃 / 卡死 / kill -9 等场景 worker 不可能再上报心跳，
// Sweeper 是续跑的"兜底"机制。
type Sweeper struct {
	repo      Repository
	interval  time.Duration
	threshold time.Duration
	logger    *zap.Logger
}

// NewSweeper 创建 Sweeper。interval/threshold 缺省走 SweeperInterval/ZombieThreshold。
func NewSweeper(repo Repository, interval, threshold time.Duration, logger *zap.Logger) *Sweeper {
	if interval <= 0 {
		interval = SweeperInterval
	}
	if threshold <= 0 {
		threshold = ZombieThreshold
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Sweeper{
		repo:      repo,
		interval:  interval,
		threshold: threshold,
		logger:    logger,
	}
}

// Run 阻塞循环，每 interval 扫一次。ctx 取消后返回。
// 推荐在 worker goroutine 内调：go sweeper.Run(ctx)
func (s *Sweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepOnce(ctx)
		}
	}
}

// sweepOnce 单轮扫描 — 找 zombies 并逐个 reset。
// 暴露便于单测。
func (s *Sweeper) sweepOnce(ctx context.Context) {
	zombies, err := s.repo.ListZombies(ctx, s.threshold)
	if err != nil {
		s.logger.Warn("sweeper list zombies failed", zap.Error(err))
		return
	}
	if len(zombies) == 0 {
		return
	}
	s.logger.Info("sweeper found zombie jobs",
		zap.Int("count", len(zombies)),
		zap.Duration("threshold", s.threshold),
	)
	for _, z := range zombies {
		err := s.repo.ResetZombie(ctx, z.ID)
		switch {
		case err == nil:
			s.logger.Info("zombie reset to pending",
				zap.String("job_id", z.ID.String()),
				zap.String("job_type", z.JobType),
				zap.Int("attempt", z.Attempt+1),
			)
		case errors.Is(err, ErrAttemptsExhausted):
			s.logger.Warn("zombie attempts exhausted, marked failed",
				zap.String("job_id", z.ID.String()),
				zap.String("job_type", z.JobType),
			)
		default:
			s.logger.Warn("zombie reset failed",
				zap.String("job_id", z.ID.String()),
				zap.Error(err),
			)
		}
	}
}
