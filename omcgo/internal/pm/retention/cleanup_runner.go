package retention

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

// CleanupRunner 是普通表（pm_metrics_{daily,weekly,monthly} + pm_group_metrics_{daily,weekly,monthly}）
// 的定时清理任务（T-0164 收尾 G2-Gap-2）。
//
// 与 hypertable 不同点：
//   - pm_metrics（15min）+ pm_metrics_hourly + pm_group_metrics_hourly = TimescaleDB hypertable
//     → 走 add_retention_policy 自动 drop_chunks（已挂在 migration 内）
//   - pm_metrics_{daily,weekly,monthly} + pm_group_metrics_{daily,weekly,monthly} = 普通表
//     → 需要本 runner 走 DELETE 分批清理
//
// 调度：每日 03:00（worker 内 cron 错峰非业务高峰）。
// JobType：pm_retention_cleanup（单 JobType 跑全部 6 张普通表，不按粒度拆）。
//
// 失败处理：单表 DELETE 失败 log warn 跳过，不阻塞其它表；不写 cron_state（cleanup 不需要补跑）。
type CleanupRunner struct {
	pool    *pgxpool.Pool
	service *Service
	logger  *zap.Logger
}

// JobTypeCleanup 是 retention 清理任务的 asyncjob 类型主键。
const JobTypeCleanup = "pm_retention_cleanup"

// NewCleanupRunner 构造清理 runner。
func NewCleanupRunner(pool *pgxpool.Pool, service *Service, logger *zap.Logger) *CleanupRunner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CleanupRunner{pool: pool, service: service, logger: logger.Named("pm.retention.cleanup")}
}

// JobType 实现 asyncjob.JobRunner。
func (r *CleanupRunner) JobType() string { return JobTypeCleanup }

// Run 跑一遍全部 6 张普通表 DELETE。
func (r *CleanupRunner) Run(ctx context.Context, _ *asyncjob.Job) (json.RawMessage, error) {
	// 每次跑前主动 reload retention.Service 缓存（worker 进程不订阅 SysConfigSaved 事件，
	// 24h tick 一次的成本可接受 — 5 个 SELECT 而已）。
	if err := r.service.Reload(ctx); err != nil {
		r.logger.Warn("retention reload failed; continue with cached values", zap.Error(err))
	}

	type tableSpec struct {
		name        string
		policyKey   PolicyKey
		dimension   string // log only
		timeColumn  string // 业务时间列，end_time 用于"业务上属于这一桶的截止时间"
	}
	tables := []tableSpec{
		{name: "pm_metrics_daily", policyKey: KeyDailyDays, dimension: "device", timeColumn: "end_time"},
		{name: "pm_metrics_weekly", policyKey: KeyWeeklyDays, dimension: "device", timeColumn: "end_time"},
		{name: "pm_metrics_monthly", policyKey: KeyMonthlyDays, dimension: "device", timeColumn: "end_time"},
		{name: "pm_group_metrics_daily", policyKey: KeyDailyDays, dimension: "device_group", timeColumn: "end_time"},
		{name: "pm_group_metrics_weekly", policyKey: KeyWeeklyDays, dimension: "device_group", timeColumn: "end_time"},
		{name: "pm_group_metrics_monthly", policyKey: KeyMonthlyDays, dimension: "device_group", timeColumn: "end_time"},
	}

	result := make(map[string]any)
	totalDeleted := int64(0)
	for _, t := range tables {
		days := r.service.Get(ctx, t.policyKey)
		if days < MinRetentionDays {
			r.logger.Warn("retention days too small; skip table",
				zap.String("table", t.name), zap.Int("days", days))
			continue
		}
		deleted, err := r.cleanupOne(ctx, t.name, t.timeColumn, days)
		if err != nil {
			r.logger.Warn("cleanup table failed; continue",
				zap.String("table", t.name),
				zap.String("dimension", t.dimension),
				zap.Int("days", days),
				zap.Error(err))
			result[t.name] = map[string]any{"error": err.Error()}
			continue
		}
		result[t.name] = map[string]any{"deleted": deleted, "days": days}
		totalDeleted += deleted
		r.logger.Info("cleanup table done",
			zap.String("table", t.name),
			zap.String("dimension", t.dimension),
			zap.Int("days", days),
			zap.Int64("deleted", deleted))
	}

	result["total_deleted"] = totalDeleted
	return json.Marshal(result)
}

// cleanupOne 删除单表过保留期数据。
//
// 分批 DELETE 防长事务锁表：每次最多删 5000 行，循环直到该批 < 5000（即清空）。
// 超过 30 批（150000 行）也停止，避免极端积压打爆 DB；剩余下次再清。
func (r *CleanupRunner) cleanupOne(ctx context.Context, table, timeColumn string, days int) (int64, error) {
	const batchSize = 5000
	const maxBatches = 30

	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	q := fmt.Sprintf(`
DELETE FROM %s
WHERE %s < $1
  AND ctid = ANY (ARRAY(
      SELECT ctid FROM %s WHERE %s < $1 LIMIT %d
  ))`, table, timeColumn, table, timeColumn, batchSize)

	var total int64
	for batch := 0; batch < maxBatches; batch++ {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
		tag, err := r.pool.Exec(ctx, q, cutoff)
		if err != nil {
			return total, fmt.Errorf("delete batch %d from %s: %w", batch, table, err)
		}
		n := tag.RowsAffected()
		total += n
		if n < batchSize {
			return total, nil
		}
	}
	r.logger.Warn("cleanup hit max batches; remaining rows will be cleaned next run",
		zap.String("table", table),
		zap.Int64("deleted_this_run", total))
	return total, nil
}
