package logretention

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

// JobTypeLogRetentionCleanup 是日志保留清理任务的 asyncjob 类型主键。
const JobTypeLogRetentionCleanup = "log_retention_cleanup"

const (
	// deleteBatchSize 单批 DELETE 行数上限，防长事务/锁表。
	deleteBatchSize = 5000
	// maxBatchesPerTable 单表单次 Run 的最大批数兜底（极端积压留到下次 Run）。
	maxBatchesPerTable = 200
)

// execPool 是清理所需的最小执行接口（*pgxpool.Pool 满足；单测注入 fake）。
type execPool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// CleanupRunner 是数据库日志统一按时间保留清理任务（asyncjob.JobRunner）。每日 cron 触发，
// 对 Tables 中每张表批量删除早于同一个 cutoff 的行。单表失败只 warn、继续其它表。
type CleanupRunner struct {
	pool   execPool
	policy *RetentionPolicy
	logger *zap.Logger
}

// NewCleanupRunner 构造日志保留清理 runner。
func NewCleanupRunner(pool execPool, policy *RetentionPolicy, logger *zap.Logger) *CleanupRunner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CleanupRunner{
		pool:   pool,
		policy: policy,
		logger: logger.Named("log.retention.cleanup"),
	}
}

// JobType 实现 asyncjob.JobRunner。
func (r *CleanupRunner) JobType() string { return JobTypeLogRetentionCleanup }

// Run 按统一数据库日志保留天数清理所有受管日志表的过期行。
func (r *CleanupRunner) Run(ctx context.Context, _ *asyncjob.Job) (json.RawMessage, error) {
	if !r.policy.Enabled(ctx) {
		r.logger.Info("log retention disabled by config; skip cleanup")
		return json.Marshal(map[string]any{"skipped": true, "reason": "disabled"})
	}

	deleted := make(map[string]int, len(Tables))
	retentionDaysByTable := make(map[string]int, len(Tables))
	days := r.policy.DatabaseDays(ctx)
	if days < minDays {
		r.logger.Warn("database retention days too small; skip cleanup", zap.Int("days", days))
		return json.Marshal(map[string]any{"skipped": true, "reason": "invalid_days"})
	}
	now := time.Now()
	for _, t := range Tables {
		tableDays := days
		if t.RetentionDays > 0 {
			tableDays = t.RetentionDays
		}
		retentionDaysByTable[t.Name] = tableDays
		cutoff := now.Add(-time.Duration(tableDays) * 24 * time.Hour)
		deleted[t.Name] = r.cleanupTable(ctx, t, cutoff)
	}

	r.logger.Info("log retention cleanup done",
		zap.Int("database_days", days),
		zap.Any("table_retention_days", retentionDaysByTable),
		zap.Any("deleted", deleted))
	return json.Marshal(map[string]any{"deleted": deleted, "retention_days": retentionDaysByTable})
}

// cleanupTable 分批删除单表过期行。用 ctid 子查询 + LIMIT 自限批量（避免一次性删大量行长事务）。
// 表名/时间列来自固定白名单（Tables），非用户输入，拼入 SQL 安全。
func (r *CleanupRunner) cleanupTable(ctx context.Context, t LogTable, cutoff time.Time) int {
	if r.pool == nil {
		return 0
	}
	// #nosec G201 -- t.Table/t.TimeCol 来自编译期白名单 Tables，非外部输入。
	sql := fmt.Sprintf(
		"DELETE FROM %s WHERE ctid IN (SELECT ctid FROM %s WHERE %s < $1 LIMIT $2)",
		t.Table, t.Table, t.TimeCol,
	)

	total := 0
	for batch := 0; batch < maxBatchesPerTable; batch++ {
		select {
		case <-ctx.Done():
			return total
		default:
		}
		tag, err := r.pool.Exec(ctx, sql, cutoff, deleteBatchSize)
		if err != nil {
			r.logger.Warn("delete expired log rows failed; stop table",
				zap.String("table", t.Table), zap.Error(err))
			return total
		}
		n := int(tag.RowsAffected())
		total += n
		if n < deleteBatchSize {
			return total // 本批不足一批 → 已删完
		}
	}
	r.logger.Warn("log cleanup hit max batches; remainder next run", zap.String("table", t.Table))
	return total
}
