package datamodel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
)

const (
	// modelTTL is the TTL for cached data model definitions.
	modelTTL = 24 * time.Hour

	// resolveTTL is the TTL for cached resolution results.
	resolveTTL = 1 * time.Hour

	// scanBatchSize is the number of keys to return per SCAN iteration.
	scanBatchSize = 100
)

// DataModelCache provides Redis L2 caching for data model resolution.
// It wraps a redis.UniversalClient and manages cached data model definitions,
// resolution results, and a global cache version counter for cross-instance
// invalidation coordination.
type DataModelCache struct {
	client redis.UniversalClient
}

// NewDataModelCache creates a new DataModelCache backed by the given Redis client.
func NewDataModelCache(client redis.UniversalClient) *DataModelCache {
	return &DataModelCache{client: client}
}

// GetModel retrieves a cached data model from Redis by its full key.
// Returns nil and a nil error on cache miss.
func (c *DataModelCache) GetModel(ctx context.Context, key string) (*DataModel, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get cached model %s: %w", key, err)
	}

	var dm DataModel
	if err := json.Unmarshal(data, &dm); err != nil {
		return nil, fmt.Errorf("unmarshal cached model %s: %w", key, err)
	}

	return &dm, nil
}

// SetModel caches a data model definition in Redis with a 24-hour TTL.
func (c *DataModelCache) SetModel(ctx context.Context, key string, dm *DataModel) error {
	data, err := json.Marshal(dm)
	if err != nil {
		return fmt.Errorf("marshal model for caching: %w", err)
	}

	if err := c.client.Set(ctx, key, data, modelTTL).Err(); err != nil {
		return fmt.Errorf("set cached model %s: %w", key, err)
	}

	return nil
}

// ModelKey builds the appropriate Redis key for a data model based on its scope.
//
// Key patterns by scope:
//   - product:         datamodel:product:{carrier}:{tech}:{oui}:{product_class}
//   - oui:             datamodel:oui:{carrier}:{tech}:{oui}
//   - carrier_default: datamodel:default:{carrier}:{tech}
func ModelKey(scope, carrier, tech, oui, productClass string) string {
	switch model.DataModelScope(scope) {
	case model.ScopeProduct:
		return redisx.Keys.DataModelProduct(carrier, tech, oui, productClass)
	case model.ScopeOUI:
		return redisx.Keys.DataModelOUI(carrier, tech, oui)
	case model.ScopeCarrierDefault:
		return redisx.Keys.DataModelDefault(carrier, tech)
	default:
		return redisx.Keys.DataModelUnknown(scope, carrier, tech, oui, productClass)
	}
}

// resolveKey builds the Redis key for a cached resolution result.
func resolveKey(carrier, tech, oui, productClass string) string {
	return redisx.Keys.DataModelResolve(carrier, tech, oui, productClass)
}

// GetResolveResult retrieves a cached data model resolution result.
// Returns uuid.Nil and a nil error on cache miss.
func (c *DataModelCache) GetResolveResult(ctx context.Context, carrier, tech, oui, productClass string) (uuid.UUID, error) {
	key := resolveKey(carrier, tech, oui, productClass)

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return uuid.Nil, nil
		}
		return uuid.Nil, fmt.Errorf("get resolve result %s: %w", key, err)
	}

	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse resolve result UUID %s: %w", val, err)
	}

	return id, nil
}

// SetResolveResult caches a data model resolution result with a 1-hour TTL.
func (c *DataModelCache) SetResolveResult(ctx context.Context, carrier, tech, oui, productClass string, modelID uuid.UUID) error {
	key := resolveKey(carrier, tech, oui, productClass)

	if err := c.client.Set(ctx, key, modelID.String(), resolveTTL).Err(); err != nil {
		return fmt.Errorf("set resolve result %s: %w", key, err)
	}

	return nil
}

// GetCacheVersion returns the current global cache version counter.
// Returns 0 if the key does not exist.
func (c *DataModelCache) GetCacheVersion(ctx context.Context) (int64, error) {
	val, err := c.client.Get(ctx, redisx.Keys.DataModelCacheVersion()).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("get cache version: %w", err)
	}

	return val, nil
}

// IncrCacheVersion atomically increments the global cache version counter
// and returns the new value. This is used to signal all ACS instances that
// cached data models may be stale.
func (c *DataModelCache) IncrCacheVersion(ctx context.Context) (int64, error) {
	val, err := c.client.Incr(ctx, redisx.Keys.DataModelCacheVersion()).Result()
	if err != nil {
		return 0, fmt.Errorf("incr cache version: %w", err)
	}

	return val, nil
}

// InvalidateModel deletes the cached model entry and the resolve entry for
// the given data model, then increments the cache version counter.
func (c *DataModelCache) InvalidateModel(ctx context.Context, dm *DataModel) error {
	modelCacheKey := ModelKey(
		string(dm.Scope),
		string(dm.Carrier),
		string(dm.Technology),
		dm.OUI,
		dm.ProductClass,
	)

	resolveCacheKey := resolveKey(
		string(dm.Carrier),
		string(dm.Technology),
		dm.OUI,
		dm.ProductClass,
	)

	pipe := c.client.Pipeline()
	pipe.Del(ctx, modelCacheKey)
	pipe.Del(ctx, resolveCacheKey)
	pipe.Incr(ctx, redisx.Keys.DataModelCacheVersion())

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("invalidate model %s: %w", dm.ID, err)
	}

	return nil
}

// InvalidateAll deletes all datamodel:* keys from Redis using SCAN to avoid
// blocking the server, then increments the cache version counter.
func (c *DataModelCache) InvalidateAll(ctx context.Context) error {
	var cursor uint64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, redisx.Keys.DataModelPattern(), scanBatchSize).Result()
		if err != nil {
			return fmt.Errorf("scan datamodel keys: %w", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("delete datamodel keys: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if _, err := c.IncrCacheVersion(ctx); err != nil {
		return fmt.Errorf("incr cache version after invalidate all: %w", err)
	}

	return nil
}
