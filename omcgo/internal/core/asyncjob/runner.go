package asyncjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// JobRunner 业务模块实现本接口注册到 Registry。
// JobType 应与 InsertRequest.JobType 对齐。
type JobRunner interface {
	JobType() string
	Run(ctx context.Context, job *Job) (result json.RawMessage, err error)
}

// Registry 持有 JobType → JobRunner 的注册表 + Repository 引用 + 进程身份。
// RunNext(ctx, jobType) 抢一个任务跑一遍：装心跳 ticker → runner.Run → 收尾
// 状态。runner panic 被 recover 隔离不破坏外层。
type Registry struct {
	repo      Repository
	lockOwner string
	logger    *zap.Logger
	runners   map[string]JobRunner
	metrics   *Metrics // G8-Gap-4 Prometheus hook；nil 则不记
}

// NewRegistry 创建 Registry。
// lockOwner 推荐 "{hostname}-{pid}" 便于排查"哪个 worker 抢到任务"。
func NewRegistry(repo Repository, lockOwner string, logger *zap.Logger) *Registry {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Registry{
		repo:      repo,
		lockOwner: lockOwner,
		logger:    logger,
		runners:   make(map[string]JobRunner),
	}
}

// SetMetrics 注入 Prometheus 指标采集器；nil 关闭采集（G8-Gap-4）。
func (r *Registry) SetMetrics(m *Metrics) { r.metrics = m }

// Register 注册一个 JobRunner，相同 JobType 重复注册以最后一次为准（覆盖）。
func (r *Registry) Register(runner JobRunner) {
	if runner == nil {
		return
	}
	r.runners[runner.JobType()] = runner
}

// RunNext 抢一个 jobType 任务跑一遍。
// 返回值：
//   - true: 跑了一个任务（成功或失败已写库）
//   - false + nil: 当前无可用任务
//   - false + err: 抢锁 / 收尾失败（业务错误已由 MarkFailed 写库，此 err 是基础设施层）
func (r *Registry) RunNext(ctx context.Context, jobType string) (didRun bool, err error) {
	runner, ok := r.runners[jobType]
	if !ok {
		return false, fmt.Errorf("no runner registered for job_type=%s", jobType)
	}

	job, err := r.repo.LockNextPending(ctx, jobType, r.lockOwner)
	if err != nil {
		if errors.Is(err, ErrNoPendingJob) {
			return false, nil
		}
		return false, fmt.Errorf("lock next pending %s: %w", jobType, err)
	}

	r.logger.Info("async job picked up",
		zap.String("job_type", jobType),
		zap.String("job_id", job.ID.String()),
		zap.Int("attempt", job.Attempt),
	)

	startedAt := time.Now()

	// 心跳 ticker：另起 goroutine 每 30s 上报，job 结束后由本函数主控停 ticker。
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	heartbeatDone := make(chan struct{})
	go r.runHeartbeat(heartbeatCtx, job.ID, heartbeatDone)

	// Run with panic recover
	result, runErr := safeRun(ctx, runner, job, r.logger)

	cancelHeartbeat()
	<-heartbeatDone

	// G8-Gap-4: 单任务耗时记录（成功 / 失败都记，便于查 latency）
	r.metrics.ObserveDuration(jobType, time.Since(startedAt).Seconds())

	// 收尾：用 background context + 10s timeout（避免外层 ctx cancel 时收尾 SQL 写不进去；
	// 任务已结束，只为持久化状态留窗口，超时不致命）。
	finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer finalizeCancel()

	if runErr != nil {
		r.metrics.IncFailed(jobType) // G8-Gap-4
		if mErr := r.repo.MarkFailed(finalizeCtx, job.ID, runErr.Error()); mErr != nil {
			r.logger.Error("mark failed write error", zap.String("job_id", job.ID.String()), zap.Error(mErr))
		}
		r.logger.Warn("async job failed",
			zap.String("job_type", jobType),
			zap.String("job_id", job.ID.String()),
			zap.Error(runErr),
		)
		return true, nil
	}

	if mErr := r.repo.MarkSucceeded(finalizeCtx, job.ID, result); mErr != nil {
		r.logger.Error("mark succeeded write error", zap.String("job_id", job.ID.String()), zap.Error(mErr))
		return true, mErr
	}
	r.logger.Info("async job succeeded", zap.String("job_id", job.ID.String()))
	return true, nil
}

// runHeartbeat 在 ctx alive 期间每 HeartbeatInterval 跳一次心跳。
func (r *Registry) runHeartbeat(ctx context.Context, jobID uuid.UUID, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.repo.UpdateHeartbeat(context.Background(), jobID); err != nil {
				// ErrJobNotRunning 意味着任务已被 sweeper 改 zombie 重置回 pending，
				// 此时本 runner 仍在跑（双写风险）— log warn 让运维察觉。
				r.logger.Warn("heartbeat update failed",
					zap.String("job_id", jobID.String()),
					zap.Error(err))
			}
		}
	}
}

// safeRun 包 panic recover。runner.Run panic 转成 error，runner 内部 ctx.Done 信号也归一。
func safeRun(ctx context.Context, runner JobRunner, job *Job, logger *zap.Logger) (result json.RawMessage, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			logger.Error("async job runner panicked",
				zap.String("job_type", job.JobType),
				zap.String("job_id", job.ID.String()),
				zap.Any("panic", rec),
			)
			err = fmt.Errorf("runner panicked: %v", rec)
		}
	}()
	return runner.Run(ctx, job)
}
