package retention

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// hypertableTables maps policies that can be expressed as one TimescaleDB
// drop_after value. The mixed-granularity result hypertable is cleaned by the
// worker because each granularity has a different configured lifetime.
var hypertableTables = map[PolicyKey][]string{
	KeyRaw15MinDays: {
		"public.pm_measurement_anchors",
		"public.pm_metric_values",
	},
}

// retentionQuerier is the minimal DB interface required by PMRetentionApplier,
// extracted so unit tests can inject a stub without testcontainers or pgxmock.
// *pgxpool.Pool satisfies this interface.
type retentionQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type retentionPolicyQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const retentionPolicyDropAfterSQL = `
SELECT COALESCE(config ->> 'drop_after', '')
FROM timescaledb_information.jobs
WHERE proc_name = 'policy_retention'
  AND format('%I.%I', hypertable_schema, hypertable_name) = $1`

const retentionPolicyExpectedDropAfterSQL = `SELECT ($1::int * INTERVAL '1 day')::text`

// PMRetentionApplier 对 TimescaleDB 的 retention policy 做 remove+add 幂等更新。
// 仅操作允许自动清理的 hypertable；原始稀疏表由 worker 做水位安全清理，
// daily/weekly/monthly 由 cleanup_runner cron 处理，均不在本 applier 范畴。
type PMRetentionApplier struct {
	db         retentionQuerier
	logger     *zap.Logger
	tsdbOnce   sync.Once // TimescaleDB 扩展存在性只需检查一次
	tsdbAbsent bool      // true 表示确认 timescaledb 未安装，跳过所有 policy 调用
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

	if a.db == nil {
		return fmt.Errorf("retention database is nil")
	}
	if !a.checkTimescaleDB(ctx) {
		a.logger.Warn("timescaledb extension not found, skip retention policy update",
			zap.String("table", table))
		return nil
	}

	return a.applyPoliciesAtomic(ctx, map[string]int{table: days})
}

func (a *PMRetentionApplier) applyPoliciesAtomic(ctx context.Context, policies map[string]int) error {
	tx, err := a.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin retention policy transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	tables := make([]string, 0, len(policies))
	for table := range policies {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		days := policies[table]
		if _, _, err := retentionPolicyDropAfter(ctx, tx, table); err != nil {
			return fmt.Errorf("read existing retention policy for %s: %w", table, err)
		}
		expectedDropAfter, err := retentionPolicyExpectedDropAfter(ctx, tx, days)
		if err != nil {
			return fmt.Errorf("normalize retention policy drop_after for %s: %w", table, err)
		}
		if _, err := tx.Exec(ctx, "SELECT remove_retention_policy($1, if_exists => TRUE)", table); err != nil {
			return fmt.Errorf("remove retention policy for %s: %w", table, err)
		}
		if _, err := tx.Exec(ctx, "SELECT add_retention_policy($1, $2::int * INTERVAL '1 day')", table, days); err != nil {
			return fmt.Errorf("add retention policy for %s: %w", table, err)
		}
		actualDropAfter, policyExists, err := retentionPolicyDropAfter(ctx, tx, table)
		if err != nil {
			return fmt.Errorf("post-check retention policy for %s: %w", table, err)
		}
		if !policyExists || actualDropAfter != expectedDropAfter {
			return fmt.Errorf("post-check retention policy for %s: expected drop_after %q, got %q", table, expectedDropAfter, actualDropAfter)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit retention policy transaction: %w", err)
	}
	committed = true
	for _, table := range tables {
		a.logger.Info("PM retention policy updated", zap.String("table", table), zap.Int("days", policies[table]))
	}
	return nil
}

func retentionPolicyExpectedDropAfter(ctx context.Context, q retentionPolicyQuerier, days int) (string, error) {
	var dropAfter string
	if err := q.QueryRow(ctx, retentionPolicyExpectedDropAfterSQL, days).Scan(&dropAfter); err != nil {
		return "", err
	}
	return dropAfter, nil
}

func retentionPolicyDropAfter(ctx context.Context, q retentionPolicyQuerier, table string) (string, bool, error) {
	var dropAfter string
	err := q.QueryRow(ctx, retentionPolicyDropAfterSQL, table).Scan(&dropAfter)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return dropAfter, true, nil
}

// ApplyAll 是兼容旧 ReloadListener 的回调入口。新代码应使用 ApplyAllWithError
// 持久化并重试错误；此处保留日志行为，避免破坏既有启动期接线。
func (a *PMRetentionApplier) ApplyAll(ctx context.Context, current map[PolicyKey]int, changed []PolicyKey) {
	if err := a.ApplyAllWithError(ctx, current, changed); err != nil {
		a.logger.Warn("PMRetentionApplier.ApplyAll: failed to apply retention policy", zap.Error(err))
	}
}

// ApplyAllWithError 批量更新 changed keys 对应的 hypertable retention policy，并返回第一个错误。
// 调用方可将错误持久化为配置应用失败状态，而不是只记录日志后报告保存成功。
// 原始稀疏表按 raw_15min_days 自动 drop chunk；混合粒度结果和 Counter
// 快照由 cleanup_runner 按 granularity 分批清理。
func (a *PMRetentionApplier) ApplyAllWithError(ctx context.Context, current map[PolicyKey]int, changed []PolicyKey) error {
	policies := make(map[string]int)
	for _, k := range changed {
		tables, ok := hypertableTables[k]
		if !ok {
			continue // 普通表（daily/weekly/monthly），cron 已处理
		}
		days, ok := current[k]
		if !ok {
			return fmt.Errorf("PMRetentionApplier.ApplyAllWithError: missing days for key %s", k)
		}
		for _, table := range tables {
			policies[table] = days
		}
	}
	if len(policies) == 0 {
		return nil
	}
	if a.db == nil {
		return fmt.Errorf("retention database is nil")
	}
	if !a.checkTimescaleDB(ctx) {
		a.logger.Warn("timescaledb extension not found, skip retention policy update")
		return nil
	}
	if err := a.applyPoliciesAtomic(ctx, policies); err != nil {
		return fmt.Errorf("apply retention policies atomically: %w", err)
	}
	return nil
}
