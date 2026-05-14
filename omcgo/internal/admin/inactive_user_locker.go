package admin

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// InactiveUserLocker 实现 P2-⑨：长期未登录自动锁定。
//
// 配置字段（sys_configs category='security'）：
//   - autoLockUserDayEnable: true 启用此 cron；false 时 cron 跑但不动数据库
//   - autoLockUserDay:       连续 N 天未登录则锁
//
// 锁定规则：
//   UPDATE users SET status='disabled'
//   WHERE status='active'
//     AND source != 'builtIn'          -- 内置 admin 不锁，避免锁死入口
//     AND COALESCE(last_login_at, created_at) < NOW() - INTERVAL '<N> days'
//
// 用 COALESCE(last_login_at, created_at) 让从未登录的新用户也能被这条规则覆盖
// （创建后 N 天未首次登录 → 锁定），符合"长期未登录"的运营语义。
//
// Cron schedule: 每天 03:30 跑一次（避开整点 / 业务高峰）。
type InactiveUserLocker struct {
	pool   *pgxpool.Pool
	policy *SecurityPolicy
	cron   *cron.Cron
	cancel context.CancelFunc
	logger *zap.Logger
}

// NewInactiveUserLocker 构造 cron 但不启动；Start 才注册任务。
// policy 必须非 nil（否则 cron 永远 no-op 且无运维信号）；caller 应保证注入。
func NewInactiveUserLocker(pool *pgxpool.Pool, policy *SecurityPolicy, logger *zap.Logger) *InactiveUserLocker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &InactiveUserLocker{
		pool:   pool,
		policy: policy,
		logger: logger.Named("inactive-user-locker"),
	}
}

// Start 启动 cron。重复 Start 是安全的（仅记 warn 不 panic）。
func (l *InactiveUserLocker) Start() {
	if l.cron != nil {
		l.logger.Warn("inactive user locker already started")
		return
	}
	baseCtx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel

	l.cron = cron.New()
	// 每天 03:30 — 错开业务高峰 / 整点 cron 拥堵
	_, err := l.cron.AddFunc("30 3 * * *", func() {
		ctx, timeoutCancel := context.WithTimeout(baseCtx, 5*time.Minute)
		defer timeoutCancel()
		if locked, err := l.RunOnce(ctx); err != nil {
			l.logger.Error("inactive user lock failed", zap.Error(err))
		} else if locked > 0 {
			l.logger.Info("inactive users locked", zap.Int64("count", locked))
		}
	})
	if err != nil {
		l.logger.Error("register cron job failed", zap.Error(err))
		return
	}
	l.cron.Start()
	l.logger.Info("inactive user locker started", zap.String("schedule", "30 3 * * *"))
}

// Stop 优雅停止。
func (l *InactiveUserLocker) Stop() {
	if l.cancel != nil {
		l.cancel()
	}
	if l.cron != nil {
		l.cron.Stop()
		l.cron = nil
	}
}

// RunOnce 立即跑一次锁定扫描，返实际锁定的用户数。
// 单元/集成测试可直接调本方法，无需等 cron 触发。
func (l *InactiveUserLocker) RunOnce(ctx context.Context) (int64, error) {
	if l.policy == nil {
		return 0, nil
	}
	policy := l.policy.Get(ctx)
	if !policy.AutoLockUnusedEnabled {
		return 0, nil
	}
	if policy.AutoLockUnusedDays <= 0 {
		l.logger.Warn("autoLockUserDay invalid, skip",
			zap.Int64("value", policy.AutoLockUnusedDays))
		return 0, nil
	}

	cutoff := time.Now().AddDate(0, 0, -int(policy.AutoLockUnusedDays))

	// 用 Squirrel 构 UPDATE — 业务规范禁止字符串拼 SQL。
	query, args, err := storage.Psql.Update("users").
		Set("status", UserStatusDisabled).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"status": UserStatusActive}).
		Where(sq.NotEq{"source": UserSourceBuiltIn}).
		Where(sq.Expr("COALESCE(last_login_at, created_at) < ?", cutoff)).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build lock SQL: %w", err)
	}

	tag, err := l.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("exec lock: %w", err)
	}
	return tag.RowsAffected(), nil
}
