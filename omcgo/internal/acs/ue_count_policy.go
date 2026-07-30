package acs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
)

const ueCountGPVDescription = "UECountPolicy:GPV"

const (
	defaultUECountProbeTimeout  = 500 * time.Millisecond
	defaultUECountProbeLeaseTTL = 30 * time.Second
)

// UECountPathResolver resolves concrete, product-supported standard paths for
// the current device. PathTranslationService satisfies this interface.
type UECountPathResolver interface {
	ResolveUECountPaths(ctx context.Context, deviceSN string) ([]string, error)
}

// UECountTaskService is the narrow task surface required by UECountPolicy.
type UECountTaskService interface {
	CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error)
	LatestOpenTaskByDeviceAndMethod(
		ctx context.Context,
		deviceSN, method, description string,
	) (*task.Task, error)
}

// UECountProbeGate atomically admits at most one concurrent probe per device.
type UECountProbeGate interface {
	Acquire(ctx context.Context, deviceSN string) (bool, error)
}

type redisUECountProbeGate struct {
	client redis.Cmdable
	ttl    time.Duration
}

func NewRedisUECountProbeGate(client redis.Cmdable, ttl time.Duration) UECountProbeGate {
	if client == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = defaultUECountProbeLeaseTTL
	}
	return &redisUECountProbeGate{client: client, ttl: ttl}
}

func (g *redisUECountProbeGate) Acquire(ctx context.Context, deviceSN string) (bool, error) {
	if g == nil || g.client == nil {
		return false, fmt.Errorf("UE count probe gate is disabled")
	}
	acquired, err := g.client.SetNX(
		ctx,
		redisx.Keys.ACSUECountProbe(deviceSN),
		"1",
		g.ttl,
	).Result()
	if err != nil {
		return false, fmt.Errorf("acquire UE count probe lease: %w", err)
	}
	return acquired, nil
}

// UECountPolicy schedules one direct GPV during a Periodic Inform session.
// Outstanding pending/sent probes are coalesced per device.
type UECountPolicy struct {
	resolver UECountPathResolver
	tasks    UECountTaskService
	gate     UECountProbeGate
	timeout  time.Duration
	logger   *zap.Logger
}

func NewUECountPolicy(
	resolver UECountPathResolver,
	tasks UECountTaskService,
	gate UECountProbeGate,
	logger *zap.Logger,
) *UECountPolicy {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &UECountPolicy{
		resolver: resolver,
		tasks:    tasks,
		gate:     gate,
		timeout:  defaultUECountProbeTimeout,
		logger:   logger.Named("ue-count-policy"),
	}
}

func (p *UECountPolicy) Enabled() bool {
	return p != nil && p.resolver != nil && p.tasks != nil && p.gate != nil
}

func (p *UECountPolicy) ShouldTrigger(eventCodes []string) bool {
	if !p.Enabled() {
		return false
	}
	for _, code := range eventCodes {
		if code == tr069.EventPeriodic {
			return true
		}
	}
	return false
}

// Enqueue creates a normal (non sync-gpv) system task so the existing GPV
// response subscriber persists device_parameters and refreshes device_info.
func (p *UECountPolicy) Enqueue(ctx context.Context, deviceSN string) error {
	if !p.Enabled() {
		return nil
	}
	timeout := p.timeout
	if timeout <= 0 {
		timeout = defaultUECountProbeTimeout
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	acquired, err := p.gate.Acquire(probeCtx, deviceSN)
	if err != nil {
		return fmt.Errorf("admit UE count query: %w", err)
	}
	if !acquired {
		p.logger.Debug("skip concurrent UE count query",
			zap.String("device_sn", deviceSN))
		return nil
	}

	open, err := p.tasks.LatestOpenTaskByDeviceAndMethod(
		probeCtx,
		deviceSN,
		"GetParameterValues",
		ueCountGPVDescription,
	)
	if err != nil {
		return fmt.Errorf("find outstanding UE count query: %w", err)
	}
	if open != nil {
		p.logger.Debug("skip duplicate UE count query",
			zap.String("device_sn", deviceSN),
			zap.String("task_id", open.ID))
		return nil
	}

	paths, err := p.resolver.ResolveUECountPaths(probeCtx, deviceSN)
	if err != nil {
		return fmt.Errorf("resolve UE count paths: %w", err)
	}
	if len(paths) == 0 {
		p.logger.Debug("skip UE count query without supported paths",
			zap.String("device_sn", deviceSN))
		return nil
	}

	params, err := json.Marshal(GPVParams{Names: paths})
	if err != nil {
		return fmt.Errorf("marshal UE count query: %w", err)
	}
	_, err = p.tasks.CreateTask(probeCtx, &task.CreateTaskRequest{
		DeviceSN:    deviceSN,
		Method:      "GetParameterValues",
		Params:      params,
		Priority:    10,
		Source:      task.TaskSourceSystem,
		Description: ueCountGPVDescription,
	})
	if err != nil {
		return fmt.Errorf("create UE count query: %w", err)
	}

	p.logger.Debug("enqueued UE count query",
		zap.String("device_sn", deviceSN),
		zap.Strings("paths", paths))
	return nil
}
