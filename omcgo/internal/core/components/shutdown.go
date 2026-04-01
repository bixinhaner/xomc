package components

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"
)

// ShutdownHook defines a function to execute during graceful shutdown.
type ShutdownHook struct {
	Name     string
	Priority int // Lower value = execute first
	Fn       func(ctx context.Context) error
}

// GracefulShutdown 统一管理服务关机顺序。
// 各组件在初始化时调用 Register 注册关机回调，指定优先级。
// 收到 SIGINT/SIGTERM 时按优先级从小到大依次执行（即：先关 HTTP，后关 DB）。
// 内置超时为 30 秒，超时后强制终止。
type GracefulShutdown struct {
	timeout time.Duration
	hooks   []ShutdownHook
	logger  *zap.Logger
}

// NewGracefulShutdown creates a new GracefulShutdown orchestrator.
func NewGracefulShutdown(timeout time.Duration, logger *zap.Logger) *GracefulShutdown {
	return &GracefulShutdown{
		timeout: timeout,
		logger:  logger,
	}
}

// Register adds a shutdown hook with the given priority.
// Priority order: 1=HTTP servers, 2=NATS, 3=Redis, 4=DB, 5=MinIO
func (gs *GracefulShutdown) Register(name string, priority int, fn func(ctx context.Context) error) {
	gs.hooks = append(gs.hooks, ShutdownHook{
		Name:     name,
		Priority: priority,
		Fn:       fn,
	})
}

// Shutdown executes all hooks in priority order (lowest first).
func (gs *GracefulShutdown) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, gs.timeout)
	defer cancel()

	sort.Slice(gs.hooks, func(i, j int) bool {
		return gs.hooks[i].Priority < gs.hooks[j].Priority
	})

	var errs []error
	for _, hook := range gs.hooks {
		gs.logger.Info("shutting down component",
			zap.String("name", hook.Name),
			zap.Int("priority", hook.Priority))

		if err := hook.Fn(ctx); err != nil {
			gs.logger.Error("shutdown error",
				zap.String("name", hook.Name),
				zap.Error(err))
			errs = append(errs, fmt.Errorf("%s: %w", hook.Name, err))
		} else {
			gs.logger.Info("component shut down successfully",
				zap.String("name", hook.Name))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}
