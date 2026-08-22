package acs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
)

const ueCountGPVDescription = "UECountPolicy:GPV"

const (
	defaultUECountProbeTimeout             = 3 * time.Second
	defaultUECountProbeLeaseTTL            = time.Hour
	defaultUECountProbeRetry               = 5 * time.Minute
	ueCountProbeSpreadSlot                 = 5 * time.Minute
	defaultUECountQueueSize                = 4096
	defaultUECountWorkerCount              = 4
	ueCountAdmissionBacklogDivisor         = 2
	ueCountProcessBacklogDivisor           = 16
	ueCountDeferredPruneInterval           = time.Minute
	defaultUECountTaskRetryIntervalSeconds = 30
	// 任务生命周期必须覆盖聚合关闭宽限(12m)、下一次 Periodic Inform(5m)
	// 和调度抖动(1m)。结果即使晚于 12m 关闭点到达，也会由现有迟到事件重算吸收。
	defaultUECountTaskExpiresInSeconds = 18 * 60
	// 重试预算与 TTL 同源，保证在线设备不会在 expires_at 之前先因固定次数耗尽而失败。
	defaultUECountTaskMaxRetries = defaultUECountTaskExpiresInSeconds / defaultUECountTaskRetryIntervalSeconds
)

var errUECountPolicyDeferred = errors.New("UE count policy deferred due to local backlog")

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
	RetryAfter(ctx context.Context, deviceSN string, delay time.Duration) error
}

type redisUECountProbeGate struct {
	client redis.Cmdable
	ttl    time.Duration
}

var acquireUECountProbeScript = redis.NewScript(`
local clock = redis.call("TIME")
local now = tonumber(clock[1])
local raw = redis.call("GET", KEYS[1])
local period = tonumber(ARGV[1])
local cold_delay = tonumber(ARGV[2])
local retention = tonumber(ARGV[3])

if not raw or raw == "1" then
	local first_due = now + cold_delay
	local admitted = 0
	if cold_delay == 0 then
		first_due = now + period
		admitted = 1
	end
	redis.call("SET", KEYS[1], tostring(first_due), "EX", retention)
	return admitted
end

local due = tonumber(raw)
if not due then
	return redis.error_reply("invalid UE count probe due timestamp")
end
if due > now then
	return 0
end

redis.call("SET", KEYS[1], tostring(now + period), "EX", retention)
return 1
`)

var retryUECountProbeScript = redis.NewScript(`
local clock = redis.call("TIME")
local now = tonumber(clock[1])
local delay = tonumber(ARGV[1])
local retention = tonumber(ARGV[2])
redis.call("SET", KEYS[1], tostring(now + delay), "EX", retention)
return 1
`)

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
	periodSeconds := max(int64(g.ttl/time.Second), int64(1))
	retentionSeconds := max(periodSeconds*2, int64(1))
	spreadSlots := max(int(g.ttl/ueCountProbeSpreadSlot), 1)
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(deviceSN))
	coldDelaySeconds := int64(hasher.Sum32()%uint32(spreadSlots)) *
		int64(ueCountProbeSpreadSlot/time.Second)
	admitted, err := acquireUECountProbeScript.Run(
		ctx, g.client,
		[]string{
			redisx.Keys.ACSUECountProbe(deviceSN),
		},
		periodSeconds,
		coldDelaySeconds,
		retentionSeconds,
	).Int()
	if err != nil {
		return false, fmt.Errorf("acquire UE count probe lease: %w", err)
	}
	return admitted == 1, nil
}

func (g *redisUECountProbeGate) RetryAfter(ctx context.Context, deviceSN string, delay time.Duration) error {
	if g == nil || g.client == nil {
		return fmt.Errorf("UE count probe gate is disabled")
	}
	delaySeconds := max(int64(delay/time.Second), int64(1))
	periodSeconds := max(int64(g.ttl/time.Second), int64(1))
	retentionSeconds := max(periodSeconds*2, delaySeconds*2)
	if err := retryUECountProbeScript.Run(
		ctx,
		g.client,
		[]string{redisx.Keys.ACSUECountProbe(deviceSN)},
		delaySeconds,
		retentionSeconds,
	).Err(); err != nil {
		return fmt.Errorf("schedule UE count probe retry: %w", err)
	}
	return nil
}

// UECountPolicy schedules one direct GPV during a Periodic Inform session.
// Outstanding pending/sent probes are coalesced per device.
type UECountPolicy struct {
	resolver UECountPathResolver
	tasks    UECountTaskService
	gate     UECountProbeGate
	timeout  time.Duration
	logger   *zap.Logger
	metrics  *ACSMetrics

	queue       chan string
	pendingMu   sync.Mutex
	pending     map[string]struct{}
	deferred    map[string]time.Time
	pruneAfter  time.Time
	now         func() time.Time
	workerCount int
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
		resolver:    resolver,
		tasks:       tasks,
		gate:        gate,
		timeout:     defaultUECountProbeTimeout,
		logger:      logger.Named("ue-count-policy"),
		queue:       make(chan string, defaultUECountQueueSize),
		pending:     make(map[string]struct{}, defaultUECountQueueSize),
		deferred:    make(map[string]time.Time, defaultUECountQueueSize),
		now:         time.Now,
		workerCount: defaultUECountWorkerCount,
	}
}

func (p *UECountPolicy) Enabled() bool {
	return p != nil && p.resolver != nil && p.tasks != nil && p.gate != nil
}

func (p *UECountPolicy) SetMetrics(metrics *ACSMetrics) {
	if p == nil {
		return
	}
	p.metrics = metrics
	if metrics != nil {
		metrics.UECountQueueCapacity.Set(float64(cap(p.queue)))
		metrics.UECountQueueDepth.Set(float64(len(p.queue)))
	}
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

// Enqueue only admits the device into a bounded, process-local work queue. It
// deliberately performs no Redis, PostgreSQL, or path-translation calls so a
// slow dependency can never extend the Inform request latency.
//
// A saturated queue defers the local device briefly instead of consuming the
// distributed probe lease. The next periodic Inform can therefore retry after
// the local backlog has drained.
func (p *UECountPolicy) Enqueue(_ context.Context, deviceSN string) error {
	if !p.Enabled() {
		return nil
	}
	now := p.clockNow()
	p.pendingMu.Lock()
	p.pruneDeferredLocked(now)
	if p.isDeferredLocked(deviceSN, now) {
		p.pendingMu.Unlock()
		p.recordEnqueue("deferred")
		p.observeQueueDepth()
		return nil
	}
	if _, exists := p.pending[deviceSN]; exists {
		p.pendingMu.Unlock()
		p.recordEnqueue("coalesced")
		return nil
	}
	if p.shouldDeferAdmission() {
		p.deferLocked(deviceSN, now)
		p.pendingMu.Unlock()
		p.recordEnqueue("deferred")
		p.observeQueueDepth()
		return nil
	}
	p.pending[deviceSN] = struct{}{}
	select {
	case p.queue <- deviceSN:
		p.pendingMu.Unlock()
		p.recordEnqueue("accepted")
		p.observeQueueDepth()
		return nil
	default:
		delete(p.pending, deviceSN)
		p.deferLocked(deviceSN, now)
		p.pendingMu.Unlock()
		p.recordEnqueue("deferred")
		p.observeQueueDepth()
		return nil
	}
}

// Run processes queued UE count probes until ctx is cancelled. Callers should
// start exactly one Run goroutine and tie its context to the ACS lifecycle.
func (p *UECountPolicy) Run(ctx context.Context) {
	if !p.Enabled() {
		return
	}
	workers := p.workerCount
	if workers <= 0 {
		workers = defaultUECountWorkerCount
	}
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			p.runWorker(ctx)
		}()
	}
	wg.Wait()
}

func (p *UECountPolicy) runWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case deviceSN := <-p.queue:
			p.observeQueueDepth()
			started := time.Now()
			err := p.process(ctx, deviceSN)
			if p.metrics != nil {
				p.metrics.UECountProcessDuration.Observe(time.Since(started).Seconds())
				result := "success"
				if err != nil {
					result = "error"
				}
				if errors.Is(err, errUECountPolicyDeferred) {
					result = "deferred"
				}
				p.metrics.UECountProcessTotal.WithLabelValues(result).Inc()
			}
			if err != nil && !errors.Is(err, errUECountPolicyDeferred) {
				p.logger.Warn("process UE count query failed",
					zap.String("device_sn", deviceSN),
					zap.Error(err))
			}
			p.pendingMu.Lock()
			delete(p.pending, deviceSN)
			p.pendingMu.Unlock()
		}
	}
}

func (p *UECountPolicy) recordEnqueue(result string) {
	if p.metrics != nil {
		p.metrics.UECountEnqueueTotal.WithLabelValues(result).Inc()
	}
}

func (p *UECountPolicy) observeQueueDepth() {
	if p.metrics != nil {
		p.metrics.UECountQueueDepth.Set(float64(len(p.queue)))
	}
}

func (p *UECountPolicy) shouldDeferAdmission() bool {
	return len(p.queue) >= queueThreshold(cap(p.queue), ueCountAdmissionBacklogDivisor)
}

func (p *UECountPolicy) shouldDeferProcess() bool {
	return len(p.queue) >= queueThreshold(cap(p.queue), ueCountProcessBacklogDivisor)
}

func queueThreshold(capacity, divisor int) int {
	if capacity <= 0 {
		return 0
	}
	if divisor <= 1 {
		return capacity
	}
	threshold := capacity / divisor
	if threshold < 1 {
		return 1
	}
	return threshold
}

func (p *UECountPolicy) deferDevice(deviceSN string, now time.Time) {
	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()
	p.deferLocked(deviceSN, now)
}

func (p *UECountPolicy) deferLocked(deviceSN string, now time.Time) {
	if p.deferred == nil {
		p.deferred = make(map[string]time.Time, defaultUECountQueueSize)
	}
	p.deferred[deviceSN] = now.Add(defaultUECountProbeRetry)
}

func (p *UECountPolicy) isDeferredLocked(deviceSN string, now time.Time) bool {
	until, ok := p.deferred[deviceSN]
	if !ok {
		return false
	}
	if now.Before(until) {
		return true
	}
	delete(p.deferred, deviceSN)
	return false
}

func (p *UECountPolicy) pruneDeferredLocked(now time.Time) {
	if p.deferred == nil {
		return
	}
	if !p.pruneAfter.IsZero() && now.Before(p.pruneAfter) {
		return
	}
	for deviceSN, until := range p.deferred {
		if !now.Before(until) {
			delete(p.deferred, deviceSN)
		}
	}
	p.pruneAfter = now.Add(ueCountDeferredPruneInterval)
}

func (p *UECountPolicy) clockNow() time.Time {
	if p.now == nil {
		return time.Now()
	}
	return p.now()
}

// process creates a normal (non sync-gpv) system task so the existing GPV
// response subscriber persists device_parameters and refreshes device_info.
func (p *UECountPolicy) process(ctx context.Context, deviceSN string) (retErr error) {
	if !p.Enabled() {
		return nil
	}
	if p.shouldDeferProcess() {
		p.deferDevice(deviceSN, p.clockNow())
		return errUECountPolicyDeferred
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
	defer func() {
		if retErr == nil {
			return
		}
		retryCtx, retryCancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			defaultUECountProbeTimeout,
		)
		defer retryCancel()
		if retryErr := p.gate.RetryAfter(retryCtx, deviceSN, defaultUECountProbeRetry); retryErr != nil {
			p.logger.Warn("schedule failed UE count query retry",
				zap.String("device_sn", deviceSN),
				zap.Error(retryErr))
		}
	}()

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
	maxRetries := defaultUECountTaskMaxRetries
	_, err = p.tasks.CreateTask(probeCtx, &task.CreateTaskRequest{
		DeviceSN:             deviceSN,
		Method:               "GetParameterValues",
		Params:               params,
		Priority:             10,
		Source:               task.TaskSourceSystem,
		Description:          ueCountGPVDescription,
		MaxRetries:           &maxRetries,
		RetryIntervalSeconds: defaultUECountTaskRetryIntervalSeconds,
		ExpiresIn:            defaultUECountTaskExpiresInSeconds,
	})
	if err != nil {
		return fmt.Errorf("create UE count query: %w", err)
	}

	p.logger.Debug("enqueued UE count query",
		zap.String("device_sn", deviceSN),
		zap.Strings("paths", paths))
	return nil
}
