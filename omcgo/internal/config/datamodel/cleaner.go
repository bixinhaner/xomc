package datamodel

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// DataModelCleaner handles periodic cleanup of expired parameter templates.
// Auto-discovered templates expire after a shorter idle period than manual templates.
type DataModelCleaner struct {
	repo             DataModelRepository
	cache            *DataModelCache
	autoMaxIdleDays  int
	manualMaxIdleDays int
	logger           *zap.Logger
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
