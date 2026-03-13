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

// GracefulShutdown orchestrates shutdown of all registered components
// in priority order.
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
