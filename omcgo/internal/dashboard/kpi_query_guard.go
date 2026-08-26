package dashboard

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"golang.org/x/sync/singleflight"
)

type KPIQueryGuardConfig struct {
	QueryTimeout  time.Duration
	MaxConcurrent int
	QueueTimeout  time.Duration
	FreshTTL      time.Duration
	StaleTTL      time.Duration
}

type KPIQueryMetadata struct {
	Stale bool
}

type KPIQueryMetrics interface {
	ObserveGuardResult(key, result string)
	ObserveGuardQuery(key, status string, duration time.Duration)
	SetInflight(float64)
}

type kpiQueryCacheEntry struct {
	value      any
	storedAt   time.Time
	generation uint64
}

type guardedKPIQueryResult struct {
	value    any
	metadata KPIQueryMetadata
}

type KPIQueryGuard struct {
	config  KPIQueryGuardConfig
	sem     chan struct{}
	group   singleflight.Group
	metrics KPIQueryMetrics
	now     func() time.Time

	mu         sync.RWMutex
	cache      map[string]kpiQueryCacheEntry
	generation uint64
	inflight   int
}

func NewKPIQueryGuard(config KPIQueryGuardConfig, metrics KPIQueryMetrics) *KPIQueryGuard {
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 4
	}
	if config.QueryTimeout <= 0 {
		config.QueryTimeout = 60 * time.Second
	}
	if config.QueueTimeout <= 0 {
		config.QueueTimeout = 100 * time.Millisecond
	}
	if config.FreshTTL <= 0 {
		config.FreshTTL = 4*time.Minute + 30*time.Second
	}
	if config.StaleTTL <= 0 {
		config.StaleTTL = 15 * time.Minute
	}
	return &KPIQueryGuard{
		config:  config,
		sem:     make(chan struct{}, config.MaxConcurrent),
		metrics: metrics,
		now:     time.Now,
		cache:   make(map[string]kpiQueryCacheEntry),
	}
}

func (g *KPIQueryGuard) Do(
	ctx context.Context,
	key string,
	loader func(context.Context) (any, error),
) (any, KPIQueryMetadata, error) {
	if value, ok := g.fresh(key); ok {
		g.observe(key, "fresh")
		return value, KPIQueryMetadata{}, nil
	}

	resultCh := g.group.DoChan(key, func() (any, error) {
		if value, ok := g.fresh(key); ok {
			g.observe(key, "fresh")
			return guardedKPIQueryResult{value: value}, nil
		}
		generation := g.currentGeneration()
		if err := g.acquire(); err != nil {
			if value, ok := g.stale(key); ok {
				g.observe(key, "stale")
				return guardedKPIQueryResult{
					value: value, metadata: KPIQueryMetadata{Stale: true},
				}, nil
			}
			g.observe(key, "rejected")
			return nil, err
		}
		defer g.release()

		queryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), g.config.QueryTimeout)
		defer cancel()
		startedAt := time.Now()
		value, err := loader(queryCtx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(queryCtx.Err(), context.DeadlineExceeded) {
				err = fmt.Errorf("dashboard KPI query exceeded %s: %w", g.config.QueryTimeout, commonerrors.ErrTimeout)
				g.observe(key, "timeout")
				g.observeQuery(key, "timeout", time.Since(startedAt))
			} else {
				g.observeQuery(key, "error", time.Since(startedAt))
			}
			if stale, ok := g.stale(key); ok {
				g.observe(key, "stale")
				return guardedKPIQueryResult{
					value: stale, metadata: KPIQueryMetadata{Stale: true},
				}, nil
			}
			return nil, err
		}
		g.observeQuery(key, "success", time.Since(startedAt))
		g.store(key, value, generation)
		g.observe(key, "miss")
		return guardedKPIQueryResult{value: value}, nil
	})

	select {
	case <-ctx.Done():
		return nil, KPIQueryMetadata{}, ctx.Err()
	case result := <-resultCh:
		if result.Shared {
			g.observe(key, "coalesced")
		}
		if result.Err != nil {
			return nil, KPIQueryMetadata{}, result.Err
		}
		guarded, ok := result.Val.(guardedKPIQueryResult)
		if !ok {
			return nil, KPIQueryMetadata{}, fmt.Errorf("dashboard KPI guard returned invalid result")
		}
		return guarded.value, guarded.metadata, nil
	}
}

func (g *KPIQueryGuard) Invalidate() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.generation++
	g.cache = make(map[string]kpiQueryCacheEntry)
}

func (g *KPIQueryGuard) fresh(key string) (any, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	entry, ok := g.cache[key]
	if !ok || g.now().Sub(entry.storedAt) > g.config.FreshTTL {
		return nil, false
	}
	return entry.value, true
}

func (g *KPIQueryGuard) stale(key string) (any, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	entry, ok := g.cache[key]
	if !ok || g.now().Sub(entry.storedAt) > g.config.StaleTTL {
		return nil, false
	}
	return entry.value, true
}

func (g *KPIQueryGuard) store(key string, value any, generation uint64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if generation != g.generation {
		return
	}
	g.cache[key] = kpiQueryCacheEntry{
		value: value, storedAt: g.now(), generation: generation,
	}
}

func (g *KPIQueryGuard) currentGeneration() uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.generation
}

func (g *KPIQueryGuard) acquire() error {
	timer := time.NewTimer(g.config.QueueTimeout)
	defer timer.Stop()
	select {
	case g.sem <- struct{}{}:
		g.mu.Lock()
		g.inflight++
		inflight := g.inflight
		g.mu.Unlock()
		if g.metrics != nil {
			g.metrics.SetInflight(float64(inflight))
		}
		return nil
	case <-timer.C:
		return fmt.Errorf("dashboard KPI query concurrency limit reached: %w", commonerrors.ErrUnavailable)
	}
}

func (g *KPIQueryGuard) release() {
	<-g.sem
	g.mu.Lock()
	g.inflight--
	inflight := g.inflight
	g.mu.Unlock()
	if g.metrics != nil {
		g.metrics.SetInflight(float64(inflight))
	}
}

func (g *KPIQueryGuard) observe(key, result string) {
	if g.metrics != nil {
		g.metrics.ObserveGuardResult(key, result)
	}
}

func (g *KPIQueryGuard) observeQuery(key, status string, duration time.Duration) {
	if g.metrics != nil {
		g.metrics.ObserveGuardQuery(key, status, duration)
	}
}
