package provider

import (
	"context"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/alarm"
)

type alarmSysConfigReader struct {
	inner admin.SysConfigRepository
}

func (a *alarmSysConfigReader) GetByKey(ctx context.Context, category, key string) (*alarm.HistoryRetentionConfigRow, error) {
	row, err := a.inner.GetByKey(ctx, category, key)
	if err != nil || row == nil {
		return nil, err
	}
	return &alarm.HistoryRetentionConfigRow{Value: row.Value, ValueType: row.ValueType}, nil
}

func initAlarmRetentionModule(c *Container) error {
	logger := c.Logger.Named("alarm-retention")

	sysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	reader := &alarmSysConfigReader{inner: sysConfigRepo}
	policyPool := c.TsPool
	if policyPool == nil {
		policyPool = c.PgPool
	}
	applier := alarm.NewTimescaleHistoryRetentionApplier(policyPool)
	svc := alarm.NewHistoryRetentionService(reader, applier, logger)

	if err := svc.ReloadAndApply(context.Background()); err != nil {
		logger.Warn("alarm history retention initial apply failed", zap.Error(err))
	}

	if c.SysConfigSvc != nil {
		c.SysConfigSvc.RegisterValidator(alarm.HistoryRetentionCategory, alarm.HistoryRetentionKey, func(value string) error {
			days, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("alarm history retention must be an integer number of days: %w", err)
			}
			if days < alarm.MinHistoryRetentionDays || days > alarm.MaxHistoryRetentionDays {
				return fmt.Errorf("alarm history retention days must be between %d and %d", alarm.MinHistoryRetentionDays, alarm.MaxHistoryRetentionDays)
			}
			return nil
		})
		c.SysConfigSvc.RegisterApplyHandler(alarm.HistoryRetentionCategory, "alarm_history_retention", func(ctx context.Context, _ admin.ConfigApplyWork) (map[string]any, error) {
			err := svc.ApplyForSysConfigCategory(ctx, alarm.HistoryRetentionCategory)
			if err != nil {
				return nil, err
			}
			return map[string]any{"alarm_history_retention_days": svc.CurrentDays()}, nil
		})
	} else {
		logger.Warn("SysConfigSvc not wired, alarm history retention won't apply on sys_configs save")
	}

	c.AlarmHistoryRetentionSvc = svc
	logger.Info("alarm history retention service initialized", zap.Int("days", svc.CurrentDays()))
	return nil
}
