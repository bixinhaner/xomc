package alarm

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
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

	s.mu.RLock()
	if s.initialized && s.currentDays == days {
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

func (s *HistoryRetentionService) OnSysConfigSaved(ctx context.Context, category string) {
	if category != HistoryRetentionCategory {
		return
	}
	if err := s.ReloadAndApply(ctx); err != nil {
		s.logger.Warn("reload alarm history retention failed", zap.Error(err))
	}
}

func (s *HistoryRetentionService) resolveDays(ctx context.Context) (int, error) {
	row, err := s.reader.GetByKey(ctx, HistoryRetentionCategory, HistoryRetentionKey)
	if err != nil {
		return DefaultHistoryRetentionDays, fmt.Errorf("read sys_configs (%s,%s): %w", HistoryRetentionCategory, HistoryRetentionKey, err)
	}
	if row == nil {
		return DefaultHistoryRetentionDays, nil
	}
	if row.ValueType != "" && row.ValueType != "int" {
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

type TimescaleHistoryRetentionApplier struct {
	pool *pgxpool.Pool
}

func NewTimescaleHistoryRetentionApplier(pool *pgxpool.Pool) *TimescaleHistoryRetentionApplier {
	return &TimescaleHistoryRetentionApplier{pool: pool}
}

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

	if _, err := a.pool.Exec(ctx, "SELECT remove_retention_policy('alarms_history', if_exists => TRUE)"); err != nil {
		return fmt.Errorf("remove alarms_history retention policy: %w", err)
	}
	interval := fmt.Sprintf("%d days", days)
	if _, err := a.pool.Exec(ctx, "SELECT add_retention_policy('alarms_history', $1::interval, if_not_exists => TRUE)", interval); err != nil {
		return fmt.Errorf("add alarms_history retention policy: %w", err)
	}
	return nil
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
