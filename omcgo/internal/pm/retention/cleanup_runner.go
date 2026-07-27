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

// CleanupRunner applies the five pm.retention settings to ordinary metadata,
// compact Counter snapshots and the mixed-granularity result hypertable.
// Raw sparse hypertables use TimescaleDB drop-chunk policies because both
// tables share one raw_15min_days lifetime.
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

// Run applies every configured PM retention scope.
func (r *CleanupRunner) Run(ctx context.Context, _ *asyncjob.Job) (json.RawMessage, error) {
	// 每次跑前主动 reload retention.Service 缓存（worker 进程不订阅 SysConfigSaved 事件，
	// 24h tick 一次的成本可接受 — 5 个 SELECT 而已）。
	if err := r.service.Reload(ctx); err != nil {
		r.logger.Warn("retention reload failed; continue with cached values", zap.Error(err))
	}

	type tableSpec struct {
		name       string
		policyKey  PolicyKey
		resultKey  string
		timeColumn string
		whereExtra string
	}
	tables := []tableSpec{
		{name: "pm_files", resultKey: "pm_files", policyKey: KeyRaw15MinDays, timeColumn: "created_at"},
		{name: "pm_ingest_batches", resultKey: "pm_ingest_batches", policyKey: KeyRaw15MinDays, timeColumn: "started_at"},
		{name: "pm_aggregation_counter_rollups", resultKey: "counter_rollups_hourly", policyKey: KeyHourlyDays, timeColumn: "window_end", whereExtra: "granularity='hourly'"},
		{name: "pm_aggregation_counter_rollups", resultKey: "counter_rollups_daily", policyKey: KeyDailyDays, timeColumn: "window_end", whereExtra: "granularity='daily'"},
		{name: "pm_aggregation_results", resultKey: "results_hourly", policyKey: KeyHourlyDays, timeColumn: "window_end", whereExtra: "granularity='hourly'"},
		{name: "pm_aggregation_results", resultKey: "results_daily", policyKey: KeyDailyDays, timeColumn: "window_end", whereExtra: "granularity='daily'"},
		{name: "pm_aggregation_results", resultKey: "results_weekly", policyKey: KeyWeeklyDays, timeColumn: "window_end", whereExtra: "granularity='weekly'"},
		{name: "pm_aggregation_results", resultKey: "results_monthly", policyKey: KeyMonthlyDays, timeColumn: "window_end", whereExtra: "granularity='monthly'"},
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
		deleted, err := r.cleanupOne(ctx, t.name, t.timeColumn, t.whereExtra, days)
		if err != nil {
			r.logger.Warn("cleanup table failed; continue",
				zap.String("table", t.name),
				zap.String("scope", t.resultKey),
				zap.Int("days", days),
				zap.Error(err))
			result[t.resultKey] = map[string]any{"error": err.Error()}
			continue
		}
		result[t.resultKey] = map[string]any{"deleted": deleted, "days": days}
		totalDeleted += deleted
		r.logger.Info("cleanup table done",
			zap.String("table", t.name),
			zap.String("scope", t.resultKey),
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
func (r *CleanupRunner) cleanupOne(
	ctx context.Context,
	table, timeColumn, whereExtra string,
	days int,
) (int64, error) {
	const batchSize = 10000
	const maxBatches = 500

	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	if whereExtra == "" {
		whereExtra = "TRUE"
	}
	q := fmt.Sprintf(`
DELETE FROM %s
WHERE %s < $1
  AND %s
  AND ctid = ANY (ARRAY(
      SELECT ctid FROM %s WHERE %s < $1 AND %s LIMIT %d
  ))`, table, timeColumn, whereExtra, table, timeColumn, whereExtra, batchSize)

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
