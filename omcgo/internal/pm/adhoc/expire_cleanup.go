package adhoc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

// DefaultExpireCleanupCron 是过期任务定义清理的默认 cron（每天 03:00）。
// 与 T-0178/T-0180 备份清理同 cadence，错峰业务高峰。
const DefaultExpireCleanupCron = "0 3 * * *"

// execer 是 ExpireCleanup 所需的最小 pgxpool 子集（仅 Exec），便于单测 stub。
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ExpireCleanup 清理过期的 adhoc 任务【定义行】。
//
// 两层口径分离（设计 §2.3）：
//   - 本清理只删 pm_tasks 表里的任务定义行（mode='oneshot' 且 is_builtin=false 且
//     created_at < now() - expire_days 天），且必须 task_subtype='adhoc_aggregation'。
//   - **绝不触碰结果表 pm_adhoc_aggregation_results**——结果数据是 TimescaleDB hypertable，
//     已有 add_retention_policy 自动 drop_chunks，与任务定义过期完全分离。
//
// 不删的：
//   - continuous 任务（持续型不消费 expire_days，长期存活）
//   - 内置任务（is_builtin=true，由 seed 维护，永不按过期清）
//   - 未到期的 oneshot 任务（created_at + expire_days 仍在未来）
type ExpireCleanup struct {
	db     execer
	logger *zap.Logger
}

// NewExpireCleanup 构造清理器。
func NewExpireCleanup(db execer, logger *zap.Logger) *ExpireCleanup {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ExpireCleanup{db: db, logger: logger.Named("pm.adhoc.expire_cleanup")}
}

// deleteExpiredSQL 删除过期的 oneshot 非内置 adhoc 任务定义行。
//
// 条件全部 AND：
//   - task_subtype = 'adhoc_aggregation'（只动 adhoc 行，不碰老 extraction/report 行）
//   - mode = 'oneshot'（continuous 排除）
//   - is_builtin = false（内置任务排除）
//   - created_at < now() - (expire_days || ' days')::interval（按每行自己的 expire_days 算过期）
const deleteExpiredSQL = `
DELETE FROM pm_tasks
WHERE task_subtype = 'adhoc_aggregation'
  AND mode = 'oneshot'
  AND is_builtin = false
  AND created_at < NOW() - (expire_days::text || ' days')::interval`

// Run 执行一次清理，返回删除的任务定义行数。
func (c *ExpireCleanup) Run(ctx context.Context) (int64, error) {
	tag, err := c.db.Exec(ctx, deleteExpiredSQL)
	if err != nil {
		return 0, fmt.Errorf("adhoc.ExpireCleanup: delete expired tasks: %w", err)
	}
	deleted := tag.RowsAffected()
	if deleted > 0 {
		c.logger.Info("adhoc expired task definitions cleaned",
			zap.Int64("deleted", deleted))
	}
	return deleted, nil
}
