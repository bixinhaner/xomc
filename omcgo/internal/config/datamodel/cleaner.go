package datamodel

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// DataModelCleaner handles periodic cleanup of expired parameter templates.
// Auto-discovered templates expire after a shorter idle period than manual templates.
type DataModelCleaner struct {
	repo              DataModelRepository
	cache             *DataModelCache
	autoMaxIdleDays   int
	manualMaxIdleDays int
	logger            *zap.Logger
	scheduler         *cron.Cron
}

// NewDataModelCleaner creates a new DataModelCleaner with the given expiry configuration.
func NewDataModelCleaner(repo DataModelRepository, cache *DataModelCache, autoMaxIdleDays, manualMaxIdleDays int, logger *zap.Logger) *DataModelCleaner {
	return &DataModelCleaner{
		repo:              repo,
		cache:             cache,
		autoMaxIdleDays:   autoMaxIdleDays,
		manualMaxIdleDays: manualMaxIdleDays,
		logger:            logger,
	}
}

// Start begins periodic cleanup using the given cron expression (e.g. "0 3 * * *" for daily at 3 AM).
func (c *DataModelCleaner) Start(cronExpr string) error {
	c.scheduler = cron.New()
	_, err := c.scheduler.AddFunc(cronExpr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*60*1e9) // 5 minutes
		defer cancel()
		if err := c.CleanExpired(ctx); err != nil {
			c.logger.Error("data model cleanup failed", zap.Error(err))
		}
	})
	if err != nil {
		return fmt.Errorf("schedule datamodel cleanup with cron %q: %w", cronExpr, err)
	}
	c.scheduler.Start()
	c.logger.Info("data model cleaner started",
		zap.String("cron", cronExpr),
		zap.Int("auto_max_idle_days", c.autoMaxIdleDays),
		zap.Int("manual_max_idle_days", c.manualMaxIdleDays),
	)
	return nil
}

// Stop gracefully stops the cron scheduler.
func (c *DataModelCleaner) Stop() {
	if c.scheduler != nil {
		ctx := c.scheduler.Stop()
		<-ctx.Done()
	}
}

// CleanExpired deletes data models that have not been accessed within their
// configured idle period. After deletion, the cache version is incremented
// to force all ACS instances to refresh their local caches.
func (c *DataModelCleaner) CleanExpired(ctx context.Context) error {
	deleted, err := c.repo.DeleteExpired(ctx, c.autoMaxIdleDays, c.manualMaxIdleDays)
	if err != nil {
		return fmt.Errorf("clean expired data models: %w", err)
	}

	if deleted > 0 {
		c.logger.Info("expired data models cleaned",
			zap.Int64("deleted_count", deleted),
			zap.Int("auto_max_idle_days", c.autoMaxIdleDays),
			zap.Int("manual_max_idle_days", c.manualMaxIdleDays),
		)

		if c.cache != nil {
			if _, err := c.cache.IncrCacheVersion(ctx); err != nil {
				c.logger.Warn("failed to increment cache version after cleanup", zap.Error(err))
			}
		}
	}

	return nil
}
