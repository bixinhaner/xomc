package retention

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// hypertableTables maps PolicyKey to the hypertables it governs.
// Only hypertable keys are listed; ordinary tables (daily/weekly/monthly) are
// managed by the cleanup_runner cron and are not covered here.
//
// 注：migrations/tsdb 中共有 4 张 hypertable，此处仅列 3 张：
//   pm_metrics                  ← KeyRaw15MinDays
//   pm_metrics_hourly           ← KeyHourlyDays
//   pm_group_metrics_hourly     ← KeyHourlyDays
//   pm_adhoc_aggregation_results ← 有意跳过：retention 365d hardcode 于 migration，
//                                   无 UI 配置键，不受本 applier 管辖。
var hypertableTables = map[PolicyKey][]string{
	KeyRaw15MinDays: {"public.pm_metrics"},
	KeyHourlyDays:   {"public.pm_metrics_hourly", "public.pm_group_metrics_hourly"},
}

// retentionQuerier is the minimal DB interface required by PMRetentionApplier,
// extracted so unit tests can inject a stub without testcontainers or pgxmock.
// *pgxpool.Pool satisfies this interface.
type retentionQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PMRetentionApplier 对 TimescaleDB 的 retention policy 做 remove+add 幂等更新。
// 仅操作 hypertable（pm_metrics、pm_metrics_hourly、pm_group_metrics_hourly）；
// daily/weekly/monthly 为普通表，由 cleanup_runner cron 处理，不在本 applier 范畴。
type PMRetentionApplier struct {
	db          retentionQuerier
	logger      *zap.Logger
	tsdbOnce    sync.Once // TimescaleDB 扩展存在性只需检查一次
	tsdbAbsent  bool      // true 表示确认 timescaledb 未安装，跳过所有 policy 调用
}

// NewPMRetentionApplier 创建 PMRetentionApplier。
// pool 必须指向已启用 timescaledb 扩展的 TimescaleDB 实例（c.TsPool）。
func NewPMRetentionApplier(pool *pgxpool.Pool, logger *zap.Logger) *PMRetentionApplier {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PMRetentionApplier{db: pool, logger: logger}
}

// checkTimescaleDB 检查 timescaledb 扩展是否安装，结果通过 sync.Once 缓存。
// 扩展存在性在进程生命周期内不变，无需每次 Apply 重复查询。
// ErrNoRows → 扩展缺失（设置 tsdbAbsent=true）；其他 DB 错误 → 保守假设存在，让后续 SQL 自行报错。
func (a *PMRetentionApplier) checkTimescaleDB(ctx context.Context) bool {
	a.tsdbOnce.Do(func() {
		var installed int
		err := a.db.QueryRow(ctx, "SELECT 1 FROM pg_extension WHERE extname = 'timescaledb'").Scan(&installed)
		if err == pgx.ErrNoRows {
			a.tsdbAbsent = true
		} else if err != nil {
			// 暂时性错误：保守假设存在，让 remove/add SQL 自行报错；下次不再重试（Once 已 fired）。
			a.logger.Warn("check timescaledb extension failed, assuming present", zap.Error(err))
		}
	})
	return !a.tsdbAbsent
}

// Apply 对指定 hypertable 重设 retention policy（remove + add，幂等）。
// 若 TimescaleDB 扩展未安装，跳过并记录警告（兼容纯 PG 测试环境）。
func (a *PMRetentionApplier) Apply(ctx context.Context, table string, days int) error {
	if err := ValidateDays(days); err != nil {
		return fmt.Errorf("invalid retention days for %s: %w", table, err)
	}

	if !a.checkTimescaleDB(ctx) {
		a.logger.Warn("timescaledb extension not found, skip retention policy update",
			zap.String("table", table))
		return nil
	}

	if _, err := a.db.Exec(ctx, "SELECT remove_retention_policy($1, if_exists => TRUE)", table); err != nil {
		return fmt.Errorf("remove retention policy for %s: %w", table, err)
	}
	interval := fmt.Sprintf("%d days", days)
	// 不使用 if_not_exists => TRUE：若 remove 未生效仍存在旧 policy，add 应当报错而非静默跳过。
	if _, err := a.db.Exec(ctx, "SELECT add_retention_policy($1, $2::interval)", table, interval); err != nil {
		return fmt.Errorf("add retention policy for %s: %w", table, err)
	}
	a.logger.Info("PM retention policy updated", zap.String("table", table), zap.Int("days", days))
	return nil
}

// ApplyAll 是 ReloadListener 回调实现，批量更新 changed keys 对应的 hypertable retention policy。
// 仅 KeyRaw15MinDays 和 KeyHourlyDays 对应 hypertable；其余 key（daily/weekly/monthly）忽略，
// 由 cleanup_runner 按天数清理。
func (a *PMRetentionApplier) ApplyAll(ctx context.Context, current map[PolicyKey]int, changed []PolicyKey) {
	for _, k := range changed {
		tables, ok := hypertableTables[k]
		if !ok {
			continue // 普通表（daily/weekly/monthly），cron 已处理
		}
		days, ok := current[k]
		if !ok {
			a.logger.Warn("PMRetentionApplier.ApplyAll: missing days for key",
				zap.String("key", string(k)))
			continue
		}
		for _, table := range tables {
			if err := a.Apply(ctx, table, days); err != nil {
				a.logger.Warn("PMRetentionApplier.ApplyAll: failed to apply retention policy",
					zap.String("table", table), zap.Int("days", days), zap.Error(err))
			}
		}
	}
}
