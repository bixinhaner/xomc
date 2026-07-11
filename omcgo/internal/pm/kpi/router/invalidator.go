package router

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	componentlogger "github.com/omcgo/omcgo/internal/core/components/logger"
)

// InvalidationTrigger 是 KPI 路由失效的低基数来源标识。
type InvalidationTrigger string

const (
	InvalidationTriggerManual          InvalidationTrigger = "manual"
	InvalidationTriggerIndicatorReload InvalidationTrigger = "indicator_reload"
	InvalidationTriggerIndicatorWrite  InvalidationTrigger = "indicator_write"
	InvalidationTriggerFormulaWrite    InvalidationTrigger = "platform_formula_write"
	InvalidationTriggerGroupDelete     InvalidationTrigger = "indicator_group_delete"
	InvalidationTriggerProductWrite    InvalidationTrigger = "product_write"
	InvalidationTriggerProductReload   InvalidationTrigger = "product_reload"
)

// InvalidationScope 描述一次失效能够覆盖的进程范围。
type InvalidationScope string

const (
	InvalidationScopeGlobal InvalidationScope = "global"
	InvalidationScopeLocal  InvalidationScope = "local"
)

// LocalInvalidationTarget 清理当前进程持有的 KPI 路由 L1。
type LocalInvalidationTarget interface {
	InvalidateAll()
}

// VersionBumper 推进跨进程 KPI 路由缓存版本。
type VersionBumper interface {
	BumpVersion(ctx context.Context) (int64, error)
}

// InvalidatorOptions 配置 KPI 路由失效器。
type InvalidatorOptions struct {
	Logger     *zap.Logger
	Metrics    *Metrics
	RetryDelay time.Duration
	Timeout    time.Duration
}

// InvalidationResult 是管理入口和写后失效调用方共同消费的结果。
type InvalidationResult struct {
	CacheVersion     int64             `json:"cache_version"`
	Attempts         int               `json:"attempts"`
	Scope            InvalidationScope `json:"scope"`
	MultiProcessSync bool              `json:"multi_process_sync"`
}

// Invalidator 统一执行本进程 L1 清理与跨进程版本推进。
type Invalidator struct {
	localMu    sync.RWMutex
	local      LocalInvalidationTarget
	bumper     VersionBumper
	logger     *zap.Logger
	metrics    *Metrics
	retryDelay time.Duration
	timeout    time.Duration
}

// SetLocalTarget 绑定当前进程 Router。provider 在 Router 初始化完成后调用一次；
// 失效器更早创建，才能同时供启动期/运行期 indicator Loader 复用。
func (i *Invalidator) SetLocalTarget(local LocalInvalidationTarget) {
	i.localMu.Lock()
	defer i.localMu.Unlock()
	i.local = local
}

const (
	maxInvalidationAttempts    = 3
	defaultRetryDelay          = 100 * time.Millisecond
	defaultInvalidationTimeout = 5 * time.Second
)

// NewInvalidator 创建 KPI 路由失效器。
func NewInvalidator(local LocalInvalidationTarget, bumper VersionBumper, opts InvalidatorOptions) *Invalidator {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	retryDelay := opts.RetryDelay
	if retryDelay <= 0 {
		retryDelay = defaultRetryDelay
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultInvalidationTimeout
	}
	return &Invalidator{
		local:      local,
		bumper:     bumper,
		logger:     logger.Named("kpi.route-invalidator"),
		metrics:    opts.Metrics,
		retryDelay: retryDelay,
		timeout:    timeout,
	}
}

// Invalidate 先清当前进程 L1，再推进跨进程缓存版本。
func (i *Invalidator) Invalidate(ctx context.Context, trigger InvalidationTrigger) (InvalidationResult, error) {
	i.localMu.RLock()
	local := i.local
	i.localMu.RUnlock()
	if local != nil {
		local.InvalidateAll()
	}
	if i.bumper == nil {
		result := InvalidationResult{Scope: InvalidationScopeLocal, MultiProcessSync: false}
		i.metrics.invalidated(trigger, "success", result.Scope)
		i.logger.Warn("KPI route invalidated in local process only; Redis is unavailable",
			zap.String("trigger", safeInvalidationTrigger(trigger)),
			zap.String("scope", string(result.Scope)),
			zap.Bool("multi_process_sync", false))
		return result, nil
	}
	bumpCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), i.timeout)
	defer cancel()
	var lastErr error
	for attempt := 1; attempt <= maxInvalidationAttempts; attempt++ {
		version, err := i.bumper.BumpVersion(bumpCtx)
		if err == nil {
			result := InvalidationResult{
				CacheVersion:     version,
				Attempts:         attempt,
				Scope:            InvalidationScopeGlobal,
				MultiProcessSync: true,
			}
			i.metrics.invalidated(trigger, "success", result.Scope)
			i.logger.Info("KPI route invalidated",
				zap.String("trigger", safeInvalidationTrigger(trigger)),
				zap.String("scope", string(result.Scope)),
				zap.Int64("cache_version", version),
				zap.Int("attempts", attempt))
			return result, nil
		}
		lastErr = err
		if attempt == maxInvalidationAttempts {
			break
		}
		timer := time.NewTimer(i.retryDelay)
		select {
		case <-bumpCtx.Done():
			timer.Stop()
			result := InvalidationResult{Attempts: attempt, Scope: InvalidationScopeGlobal, MultiProcessSync: false}
			err := fmt.Errorf("KPI route cache version bump stopped after %d attempts: %w", attempt, bumpCtx.Err())
			i.recordFailure(ctx, trigger, result, err)
			return result, err
		case <-timer.C:
		}
	}
	result := InvalidationResult{Attempts: maxInvalidationAttempts, Scope: InvalidationScopeGlobal, MultiProcessSync: false}
	err := fmt.Errorf("KPI route cache version bump failed after %d attempts: %w", maxInvalidationAttempts, lastErr)
	i.recordFailure(ctx, trigger, result, err)
	return result, err
}

func (i *Invalidator) recordFailure(ctx context.Context, trigger InvalidationTrigger, result InvalidationResult, err error) {
	i.metrics.invalidated(trigger, "failure", result.Scope)
	fields := []zap.Field{
		zap.String("trigger", safeInvalidationTrigger(trigger)),
		zap.String("scope", string(result.Scope)),
		zap.Int("attempts", result.Attempts),
		zap.Error(err),
	}
	if requestID := componentlogger.GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	i.logger.Error("KPI route invalidation failed", fields...)
}

func safeInvalidationTrigger(trigger InvalidationTrigger) string {
	switch trigger {
	case InvalidationTriggerManual,
		InvalidationTriggerIndicatorReload,
		InvalidationTriggerIndicatorWrite,
		InvalidationTriggerFormulaWrite,
		InvalidationTriggerGroupDelete,
		InvalidationTriggerProductWrite,
		InvalidationTriggerProductReload:
		return string(trigger)
	default:
		return "other"
	}
}
