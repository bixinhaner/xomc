package datamodel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// DataModelRegistry provides three-level fallback resolution for data model
// definitions with three-level caching (L1 in-memory, L2 Redis, L3 PostgreSQL).
//
// Resolution priority (highest to lowest):
//  1. product  — carrier + tech + oui + productClass
//  2. oui      — carrier + tech + oui
//  3. carrier_default — carrier + tech
type DataModelRegistry struct {
	repo         DataModelRepository
	cache        *DataModelCache
	localCache   sync.Map // L1 in-memory cache: string -> *DataModel
	cacheVersion int64
	logger       *zap.Logger
	stopCh       chan struct{}
}

// NewDataModelRegistry creates a new DataModelRegistry.
func NewDataModelRegistry(repo DataModelRepository, cache *DataModelCache, logger *zap.Logger) *DataModelRegistry {
	return &DataModelRegistry{
		repo:   repo,
		cache:  cache,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Resolve performs three-level fallback resolution to find the best-matching
// active data model for the given carrier, technology, OUI, and product class.
//
// The resolution algorithm:
//  1. Check L1 (sync.Map) in-memory cache.
//  2. Check L2 (Redis) resolve result cache; if hit, fetch full model from Redis/DB.
//  3. Query the database with three-level fallback:
//     a. product scope  (if oui and productClass are both non-empty)
//     b. oui scope      (if oui is non-empty)
//     c. carrier_default scope
//  4. Write back to L1 + L2 caches on DB hit.
//  5. Return nil, nil if no matching model is found.
func (r *DataModelRegistry) Resolve(ctx context.Context, carrier model.CarrierCode,
	tech model.Technology, oui, productClass string) (*DataModel, error) {

	key := localCacheKey(carrier, tech, oui, productClass)

	// Step 1: Check L1 in-memory cache.
	if val, ok := r.localCache.Load(key); ok {
		r.logger.Debug("data model resolved from L1 cache",
			zap.String("key", key),
		)
		return val.(*DataModel), nil
	}

	// Step 2: Check L2 Redis resolve result cache (skip if cache is nil).
	if r.cache != nil {
		modelID, err := r.cache.GetResolveResult(ctx, string(carrier), string(tech), oui, productClass)
		if err != nil {
			r.logger.Warn("failed to get resolve result from L2 cache",
				zap.String("key", key),
				zap.Error(err),
			)
			// Fall through to DB lookup; treat cache errors as misses.
		} else if modelID != uuid.Nil {
			// Resolve result found in L2; fetch the full model.
			dm, fetchErr := r.fetchModel(ctx, modelID)
			if fetchErr != nil {
				return nil, fmt.Errorf("fetch model after L2 resolve hit: %w", fetchErr)
			}
			if dm != nil {
				r.localCache.Store(key, dm)
				r.logger.Debug("data model resolved from L2 cache",
					zap.String("key", key),
					zap.String("model_id", modelID.String()),
				)
				return dm, nil
			}
			// Model was deleted or deactivated since resolve result was cached.
			// Fall through to DB lookup.
		}
	}

	// Step 3: DB three-level fallback.
	dm, err := r.resolveFromDB(ctx, carrier, tech, oui, productClass)
	if err != nil {
		return nil, fmt.Errorf("resolve data model: %w", err)
	}
	if dm == nil {
		// No matching model found at any level.
		return nil, nil
	}

	// Step 4: Write back to L1 + L2 caches.
	r.localCache.Store(key, dm)

	if r.cache != nil {
		if cacheErr := r.cache.SetResolveResult(ctx, string(carrier), string(tech), oui, productClass, dm.ID); cacheErr != nil {
			r.logger.Warn("failed to write resolve result to L2 cache",
				zap.String("key", key),
				zap.Error(cacheErr),
			)
		}

		modelKey := ModelKey(string(dm.Scope), string(carrier), string(tech), oui, productClass)
		if cacheErr := r.cache.SetModel(ctx, modelKey, dm); cacheErr != nil {
			r.logger.Warn("failed to write model to L2 cache",
				zap.String("key", modelKey),
				zap.Error(cacheErr),
			)
		}
	}

	r.logger.Info("data model resolved from database",
		zap.String("key", key),
		zap.String("model_id", dm.ID.String()),
		zap.String("scope", string(dm.Scope)),
	)

	return dm, nil
}

// resolveFromDB performs the three-level fallback query against the database.
// When a model is not found at a specific level, it continues to the next level.
// Only system errors (not ErrNotFound) cause the resolution to fail.
func (r *DataModelRegistry) resolveFromDB(ctx context.Context, carrier model.CarrierCode,
	tech model.Technology, oui, productClass string) (*DataModel, error) {

	// Level 1 (product): most specific, requires both oui and productClass.
	if oui != "" && productClass != "" {
		dm, err := r.repo.FindActive(ctx, carrier, tech, oui, productClass, model.ScopeProduct)
		if err != nil {
			// ErrNotFound means no match at this level, continue to next level
			if !errors.Is(err, commonerrors.ErrNotFound) {
				return nil, fmt.Errorf("find active product model: %w", err)
			}
			r.logger.Debug("no product-level data model found, trying oui level",
				zap.String("carrier", string(carrier)),
				zap.String("tech", string(tech)),
				zap.String("oui", oui),
				zap.String("product_class", productClass),
			)
		} else if dm != nil {
			return dm, nil
		}
	}

	// Level 2 (oui): vendor-level default, requires oui.
	if oui != "" {
		dm, err := r.repo.FindActive(ctx, carrier, tech, oui, "", model.ScopeOUI)
		if err != nil {
			// ErrNotFound means no match at this level, continue to next level
			if !errors.Is(err, commonerrors.ErrNotFound) {
				return nil, fmt.Errorf("find active oui model: %w", err)
			}
			r.logger.Debug("no oui-level data model found, trying carrier default",
				zap.String("carrier", string(carrier)),
				zap.String("tech", string(tech)),
				zap.String("oui", oui),
			)
		} else if dm != nil {
			return dm, nil
		}
	}

	// Level 3 (carrier_default): broadest fallback.
	dm, err := r.repo.FindActive(ctx, carrier, tech, "", "", model.ScopeCarrierDefault)
	if err != nil {
		// ErrNotFound at this level means no data model configured at all
		if errors.Is(err, commonerrors.ErrNotFound) {
			r.logger.Warn("no data model found for carrier/tech combination",
				zap.String("carrier", string(carrier)),
				zap.String("tech", string(tech)),
			)
			return nil, nil
		}
		return nil, fmt.Errorf("find active carrier default model: %w", err)
	}
	return dm, nil
}

// fetchModel retrieves a data model by ID, first trying L2 cache then the database.
func (r *DataModelRegistry) fetchModel(ctx context.Context, id uuid.UUID) (*DataModel, error) {
	// Try the database directly; the cache layer (SetModel) is populated on resolve.
	dm, err := r.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get model by id %s: %w", id, err)
	}
	return dm, nil
}

// ResolveForDevice is a convenience wrapper that resolves the data model for a device
// using the device's carrier, technology, OUI, and product class.
func (r *DataModelRegistry) ResolveForDevice(ctx context.Context, device *model.Device) (*DataModel, error) {
	if device == nil {
		return nil, fmt.Errorf("resolve data model for device: device is nil")
	}
	return r.Resolve(ctx, device.Carrier, device.Technology, device.OUI, device.ProductClass)
}

// InvalidateCache invalidates L1 and L2 caches for a specific data model.
// This should be called when a model is activated, deprecated, or updated.
func (r *DataModelRegistry) InvalidateCache(ctx context.Context, dm *DataModel) error {
	if dm == nil {
		return fmt.Errorf("invalidate cache: data model is nil")
	}

	// Invalidate L1: remove the exact key for this model's parameters.
	key := localCacheKey(dm.Carrier, dm.Technology, dm.OUI, dm.ProductClass)
	r.localCache.Delete(key)

	// Invalidate L2: remove model content and resolve results from Redis.
	if r.cache != nil {
		if err := r.cache.InvalidateModel(ctx, dm); err != nil {
			return fmt.Errorf("invalidate L2 cache for model %s: %w", dm.ID, err)
		}

		// Increment cache version to notify other ACS instances.
		if _, err := r.cache.IncrCacheVersion(ctx); err != nil {
			return fmt.Errorf("increment cache version: %w", err)
		}
	}

	r.logger.Info("cache invalidated for data model",
		zap.String("model_id", dm.ID.String()),
		zap.String("scope", string(dm.Scope)),
		zap.String("key", key),
	)

	return nil
}

// InvalidateAll clears the entire L1 in-memory cache and increments the L2 cache
// version to force all instances to refresh.
func (r *DataModelRegistry) InvalidateAll(ctx context.Context) error {
	// Clear all L1 entries.
	r.localCache.Range(func(key, _ any) bool {
		r.localCache.Delete(key)
		return true
	})

	// Increment cache version to notify all ACS instances.
	if r.cache != nil {
		newVersion, err := r.cache.IncrCacheVersion(ctx)
		if err != nil {
			return fmt.Errorf("invalidate all: increment cache version: %w", err)
		}
		r.cacheVersion = newVersion
	}

	r.logger.Info("all data model caches invalidated",
		zap.Int64("new_cache_version", r.cacheVersion),
	)

	return nil
}

// refreshLocalCache checks the cache version in Redis and clears L1 if it has changed.
// This is called periodically by the background cache watcher.
func (r *DataModelRegistry) refreshLocalCache(ctx context.Context) {
	if r.cache == nil {
		return
	}
	version, err := r.cache.GetCacheVersion(ctx)
	if err != nil {
		r.logger.Warn("failed to get cache version from Redis",
			zap.Error(err),
		)
		return
	}

	if version != r.cacheVersion {
		r.logger.Info("cache version changed, clearing L1 cache",
			zap.Int64("old_version", r.cacheVersion),
			zap.Int64("new_version", version),
		)

		r.localCache.Range(func(key, _ any) bool {
			r.localCache.Delete(key)
			return true
		})

		r.cacheVersion = version
	}
}

// Start begins a background goroutine that periodically checks the cache version
// in Redis (every 10 seconds) and clears the L1 in-memory cache if the version
// has changed. This ensures cross-instance cache coherence when data models are
// activated or deprecated on any instance.
func (r *DataModelRegistry) Start() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// Perform an initial sync on startup.
		r.refreshLocalCache(context.Background())

		for {
			select {
			case <-ticker.C:
				r.refreshLocalCache(context.Background())
			case <-r.stopCh:
				r.logger.Info("data model cache watcher stopped")
				return
			}
		}
	}()

	r.logger.Info("data model cache watcher started",
		zap.Duration("interval", 10*time.Second),
	)
}

// Stop signals the background cache watcher goroutine to exit.
func (r *DataModelRegistry) Stop() {
	close(r.stopCh)
}

// localCacheKey builds the L1 cache key from the resolution parameters.
// Format: {carrier}:{tech}:{oui}:{productClass}
func localCacheKey(carrier model.CarrierCode, tech model.Technology, oui, productClass string) string {
	return fmt.Sprintf("%s:%s:%s:%s", carrier, tech, oui, productClass)
}
