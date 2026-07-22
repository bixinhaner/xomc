package alarm

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

const (
	HistoryRetentionCategory    = "storage"
	HistoryRetentionKey         = "alarmHisMaxHoldTime"
	DefaultHistoryRetentionDays = 365
	MinHistoryRetentionDays     = 1
	MaxHistoryRetentionDays     = 365
)

type HistoryRetentionConfigRow struct {
	Value     string
	ValueType string
}

type HistoryRetentionConfigReader interface {
	GetByKey(ctx context.Context, category, key string) (*HistoryRetentionConfigRow, error)
}

type HistoryRetentionPolicyApplier interface {
	Apply(ctx context.Context, days int) error
}

type HistoryRetentionService struct {
	reader  HistoryRetentionConfigReader
	applier HistoryRetentionPolicyApplier
	logger  *zap.Logger

	mu          sync.RWMutex
	currentDays int
	initialized bool
}

func NewHistoryRetentionService(reader HistoryRetentionConfigReader, applier HistoryRetentionPolicyApplier, logger *zap.Logger) *HistoryRetentionService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HistoryRetentionService{
		reader:  reader,
		applier: applier,
		logger:  logger,
	}
}

func (s *HistoryRetentionService) CurrentDays() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.initialized {
		return DefaultHistoryRetentionDays
	}
	return s.currentDays
}

func (s *HistoryRetentionService) ReloadAndApply(ctx context.Context) error {
	days, resolveErr := s.resolveDays(ctx)
	if resolveErr != nil {
		s.logger.Warn("invalid alarm history retention config, fallback to default",
			zap.String("category", HistoryRetentionCategory),
			zap.String("key", HistoryRetentionKey),
			zap.Int("fallback_days", DefaultHistoryRetentionDays),
			zap.Error(resolveErr),
		)
	}

	return s.applyDays(ctx, days, false)
}

func (s *HistoryRetentionService) OnSysConfigSaved(ctx context.Context, category string) {
	if err := s.ApplyForSysConfigCategory(ctx, category); err != nil {
		s.logger.Warn("reload alarm history retention failed", zap.Error(err))
	}
}

// ApplyForSysConfigCategory 将 storage 分类的保存同步到告警历史保留策略。
// 它返回应用错误，供配置应用编排器记录和重试；保留 OnSysConfigSaved 以兼容旧的缓存 hook。
func (s *HistoryRetentionService) ApplyForSysConfigCategory(ctx context.Context, category string) error {
	if category != HistoryRetentionCategory {
		return nil
	}
	days, err := s.resolveDays(ctx)
	if err != nil {
		return fmt.Errorf("resolve alarm history retention: %w", err)
	}
	return s.applyDays(ctx, days, true)
}

func (s *HistoryRetentionService) applyDays(ctx context.Context, days int, force bool) error {
	s.mu.RLock()
	if !force && s.initialized && s.currentDays == days {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	if err := s.applier.Apply(ctx, days); err != nil {
		return fmt.Errorf("apply alarms_history retention policy: %w", err)
	}

	s.mu.Lock()
	s.currentDays = days
	s.initialized = true
	s.mu.Unlock()

	s.logger.Info("alarm history retention applied", zap.Int("days", days))
	return nil
}

func (s *HistoryRetentionService) resolveDays(ctx context.Context) (int, error) {
	row, err := s.reader.GetByKey(ctx, HistoryRetentionCategory, HistoryRetentionKey)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			return DefaultHistoryRetentionDays, nil
		}
		return DefaultHistoryRetentionDays, fmt.Errorf("read sys_configs (%s,%s): %w", HistoryRetentionCategory, HistoryRetentionKey, err)
	}
	if row == nil {
		return DefaultHistoryRetentionDays, nil
	}
	// 兼容旧库里 alarmHisMaxHoldTime 被存成 string 的历史数据；值仍按整数天数解析。
	if row.ValueType != "" && row.ValueType != "int" && row.ValueType != "string" {
		return DefaultHistoryRetentionDays, fmt.Errorf("sys_configs (%s,%s) value_type must be int, got %q", HistoryRetentionCategory, HistoryRetentionKey, row.ValueType)
	}
	days, err := strconv.Atoi(row.Value)
	if err != nil {
		return DefaultHistoryRetentionDays, fmt.Errorf("parse sys_configs (%s,%s): %w", HistoryRetentionCategory, HistoryRetentionKey, err)
	}
	if err := validateHistoryRetentionDays(days); err != nil {
		return DefaultHistoryRetentionDays, fmt.Errorf("validate sys_configs (%s,%s): %w", HistoryRetentionCategory, HistoryRetentionKey, err)
	}
	return days, nil
}

type historyRetentionDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type historyRetentionPolicyQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type TimescaleHistoryRetentionApplier struct {
	pool historyRetentionDB
}

func NewTimescaleHistoryRetentionApplier(pool *pgxpool.Pool) *TimescaleHistoryRetentionApplier {
	if pool == nil {
		return &TimescaleHistoryRetentionApplier{}
	}
	return &TimescaleHistoryRetentionApplier{pool: pool}
}

const addAlarmHistoryRetentionPolicySQL = `
SELECT add_retention_policy(
    'alarms_history',
    drop_after => $1::interval,
    schedule_interval => INTERVAL '1 day',
    initial_start => TIMESTAMPTZ '2000-01-01 01:08:00+08',
    timezone => 'Asia/Shanghai'
)`

const alarmHistoryRetentionPolicyDropAfterSQL = `
SELECT COALESCE(config ->> 'drop_after', '')
FROM timescaledb_information.jobs
WHERE proc_name = 'policy_retention'
  AND hypertable_schema = 'public'
  AND hypertable_name = 'alarms_history'`

func (a *TimescaleHistoryRetentionApplier) Apply(ctx context.Context, days int) error {
	if err := validateHistoryRetentionDays(days); err != nil {
		return err
	}
	if a.pool == nil {
		return fmt.Errorf("timescale pool is nil")
	}

	var installed int
	err := a.pool.QueryRow(ctx, "SELECT 1 FROM pg_extension WHERE extname = 'timescaledb'").Scan(&installed)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check timescaledb extension: %w", err)
	}

	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin alarms_history retention policy transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	_, _, err = alarmHistoryRetentionPolicyDropAfter(ctx, tx)
	if err != nil {
		return fmt.Errorf("read existing alarms_history retention policy: %w", err)
	}

	if _, err := tx.Exec(ctx, "SELECT remove_retention_policy('alarms_history', if_exists => TRUE)"); err != nil {
		return fmt.Errorf("remove alarms_history retention policy: %w", err)
	}
	interval := fmt.Sprintf("%d days", days)
	if _, err := tx.Exec(ctx, addAlarmHistoryRetentionPolicySQL, interval); err != nil {
		return fmt.Errorf("add alarms_history retention policy: %w", err)
	}
	actualDropAfter, policyExists, err := alarmHistoryRetentionPolicyDropAfter(ctx, tx)
	if err != nil {
		return fmt.Errorf("post-check alarms_history retention policy: %w", err)
	}
	if !policyExists || actualDropAfter != interval {
		return fmt.Errorf("post-check alarms_history retention policy: expected drop_after %q, got %q", interval, actualDropAfter)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit alarms_history retention policy transaction: %w", err)
	}
	committed = true
	return nil
}

func alarmHistoryRetentionPolicyDropAfter(ctx context.Context, q historyRetentionPolicyQuerier) (string, bool, error) {
	var dropAfter string
	err := q.QueryRow(ctx, alarmHistoryRetentionPolicyDropAfterSQL).Scan(&dropAfter)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return dropAfter, true, nil
}

func validateHistoryRetentionDays(days int) error {
	if days < MinHistoryRetentionDays {
		return fmt.Errorf("retention days must be at least %d", MinHistoryRetentionDays)
	}
	if days > MaxHistoryRetentionDays {
		return fmt.Errorf("retention days must be at most %d", MaxHistoryRetentionDays)
	}
	return nil
}
