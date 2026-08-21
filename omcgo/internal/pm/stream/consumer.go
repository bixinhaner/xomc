package stream

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

const (
	aggregationConsumer        = "pm-aggregation-workers"
	hourlyRollupConsumer       = "pm-aggregation-hourly-rollup"
	dailyRollupConsumer        = "pm-aggregation-daily-rollup"
	aggregationControlConsumer = "pm-aggregation-control"
)

type Consumer struct {
	bus       event.EventBus
	snapshot  *SnapshotStore
	matcher   *Matcher
	windows   *WindowRepository
	store     *RedisWindowStore
	finalizer *Finalizer
	outbox    *OutboxRepository
	rollups   *RollupOutboxRepository
	rebuilds  *RebuildRepository
	logger    *zap.Logger
	metrics   *Metrics
}

func (c *Consumer) SetRebuildRepository(rebuilds *RebuildRepository) *Consumer {
	c.rebuilds = rebuilds
	return c
}

func (c *Consumer) SetMetrics(metrics *Metrics) *Consumer {
	c.metrics = metrics
	return c
}

func (c *Consumer) SetConsumeBarriers(
	outbox *OutboxRepository,
	rollups *RollupOutboxRepository,
) *Consumer {
	c.outbox = outbox
	c.rollups = rollups
	return c
}

func NewConsumer(
	bus event.EventBus,
	snapshot *SnapshotStore,
	matcher *Matcher,
	windows *WindowRepository,
	store *RedisWindowStore,
	finalizer *Finalizer,
	logger *zap.Logger,
) *Consumer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Consumer{
		bus: bus, snapshot: snapshot, matcher: matcher, windows: windows,
		store: store, finalizer: finalizer, logger: logger,
	}
}

func (c *Consumer) Subscribe() ([]event.Subscription, error) {
	dataSub, err := c.bus.PullSubscribe(
		event.SubjectPMAggregationNormalized,
		aggregationConsumer,
		c.handle,
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe PM aggregation data stream: %w", err)
	}
	hourlySub, err := c.bus.PullSubscribe(
		event.SubjectPMAggregationHourlyRollup,
		hourlyRollupConsumer,
		c.handleRollup,
	)
	if err != nil {
		_ = dataSub.Unsubscribe()
		return nil, fmt.Errorf("subscribe PM hourly rollup stream: %w", err)
	}
	dailySub, err := c.bus.PullSubscribe(
		event.SubjectPMAggregationDailyRollup,
		dailyRollupConsumer,
		c.handleRollup,
	)
	if err != nil {
		_ = dataSub.Unsubscribe()
		_ = hourlySub.Unsubscribe()
		return nil, fmt.Errorf("subscribe PM daily rollup stream: %w", err)
	}
	controlSub, err := c.bus.PullSubscribe(
		event.SubjectPMAggregationTaskVersionChanged,
		aggregationControlConsumer,
		func(ctx context.Context, _ event.Event) error {
			return c.snapshot.Reload(ctx)
		},
	)
	if err != nil {
		_ = dataSub.Unsubscribe()
		_ = hourlySub.Unsubscribe()
		_ = dailySub.Unsubscribe()
		return nil, fmt.Errorf("subscribe PM aggregation task changes: %w", err)
	}
	return []event.Subscription{dataSub, hourlySub, dailySub, controlSub}, nil
}

func (c *Consumer) handleRollup(ctx context.Context, envelope event.Event) error {
	var payload RollupPayload
	if err := envelope.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode PM compact rollup event: %w", err)
	}
	current := c.snapshot.Current()
	if current == nil {
		return fmt.Errorf("PM aggregation task snapshot missing")
	}
	version := current.ByVersion[payload.TaskVersionID]
	location := c.matcher.Location()
	contributions, err := rollupContributions(payload, version, location)
	if err != nil {
		return fmt.Errorf("build PM parent rollup contributions: %w", err)
	}
	if isDeviceHourPayload(payload, current) {
		ruleContributions, matchErr := matchDeviceHourRules(
			payload, current, location,
		)
		if matchErr != nil {
			return fmt.Errorf("match device-hour aggregation rules: %w", matchErr)
		}
		contributions = append(contributions, ruleContributions...)
	}
	if err := c.processContributions(ctx, contributions); err != nil {
		if c.metrics != nil {
			c.metrics.EventsFailedTotal.Inc()
		}
		return err
	}
	if c.rollups != nil {
		if err := c.rollups.MarkConsumed(ctx, payload.EventID); err != nil {
			return err
		}
	}
	if c.metrics != nil {
		c.metrics.EventsProcessedTotal.Inc()
	}
	return nil
}

func (c *Consumer) handle(ctx context.Context, envelope event.Event) error {
	if err := c.process(ctx, envelope); err != nil {
		if c.metrics != nil {
			c.metrics.EventsFailedTotal.Inc()
		}
		return err
	}
	if c.metrics != nil {
		c.metrics.EventsProcessedTotal.Inc()
	}
	return nil
}

func (c *Consumer) process(ctx context.Context, envelope event.Event) error {
	var payload event.PMAggregationNormalizedPayload
	if err := envelope.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode PM aggregation event: %w", err)
	}
	if err := ValidateEvent(payload); err != nil {
		return err
	}
	contributions, err := c.matcher.Match(payload, c.snapshot.Current())
	if err != nil {
		return fmt.Errorf("match PM aggregation event: %w", err)
	}
	if err := c.processContributions(ctx, contributions); err != nil {
		return err
	}
	if c.outbox != nil {
		if err := c.outbox.MarkConsumed(ctx, payload.EventID); err != nil {
			return err
		}
	}
	return nil
}

func (c *Consumer) processContributions(ctx context.Context, contributions []Contribution) error {
	for _, contribution := range contributions {
		status, err := c.windows.Status(ctx, contribution.Key)
		if err != nil {
			return err
		}
		if status == "prepared" || status == "published" || status == "finalizing" || status == "rebuilding" {
			if c.metrics != nil {
				c.metrics.LateEventsTotal.Inc()
			}
			if c.rebuilds == nil {
				return fmt.Errorf("late PM aggregation event requires rebuild repository")
			}
			if err := c.rebuilds.Enqueue(ctx, contribution.Key, contribution.SourceFileID); err != nil {
				return fmt.Errorf("enqueue late PM aggregation rebuild: %w", err)
			}
			c.logger.Info("queued late PM aggregation event for revision rebuild",
				zap.String("source_file_id", contribution.SourceFileID),
				zap.String("task_version_id", contribution.Key.TaskVersionID.String()),
				zap.Time("window_start", contribution.Key.Start))
			continue
		}
		if err := c.windows.EnsureOpen(ctx, contribution); err != nil {
			return err
		}
		result, err := c.store.Accumulate(ctx, contribution)
		if err != nil {
			return err
		}
		if result.Duplicate && c.metrics != nil {
			c.metrics.DuplicateEventsTotal.Inc()
		}
		if err := c.windows.ObserveReceived(ctx, contribution.Key, result.ReceivedSlots); err != nil {
			return err
		}
	}
	return nil
}

type TimeoutScanner struct {
	windows                finalizeWindowRepository
	finalizeWindow         func(context.Context, WindowKey, CloseReason, uuid.UUID) error
	publishReady           func(context.Context, time.Time, time.Duration, uint64) error
	snapshot               *SnapshotStore
	workerCount            int
	grace                  time.Duration
	graceByGranularity     map[Granularity]time.Duration
	logger                 *zap.Logger
	metrics                *Metrics
	claimLease             time.Duration
	renewInterval          time.Duration
	oldestDueInterval      time.Duration
	selector               *finalizeClaimSelector
	deviceVersionsFor      *TaskSnapshot
	deviceVersionIDs       []uuid.UUID
	legacyReplayDeviceHour func(context.Context, WindowKey) (bool, error)
	loadDeviceHour         func(
		context.Context,
		WindowKey,
	) (*DeviceHourReplayBatch, error)
	applyDeviceHour func(
		context.Context,
		WindowKey,
		*DeviceHourReplayBatch,
	) (DeviceHourReplayOutcome, error)
}

type finalizeWindowRepository interface {
	CountWatermarkBlocked(
		context.Context,
		time.Time,
		map[Granularity]time.Duration,
		time.Duration,
	) (int64, error)
	claimDue(
		context.Context,
		Granularity,
		time.Time,
		uint64,
		uuid.UUID,
		time.Time,
		claimVersionFilter,
		claimOrder,
	) ([]WindowRecord, error)
	hasClaimConflict(
		context.Context,
		Granularity,
		time.Time,
		claimVersionFilter,
		uuid.UUID,
	) (bool, error)
	RenewClaim(context.Context, WindowKey, uuid.UUID, time.Time) error
	CompleteClaim(context.Context, WindowKey, uuid.UUID) error
	ReleaseClaim(context.Context, WindowKey, uuid.UUID) error
	FailClaim(context.Context, WindowKey, uuid.UUID, error, time.Duration) error
	OldestDue(
		context.Context,
		map[Granularity]time.Duration,
		time.Duration,
	) (time.Duration, error)
}

type finalizeQueue int

const (
	finalizeHourlyDevice finalizeQueue = iota
	finalizeHourlyOther
	finalizeLongPeriod
	finalizeClaimLease             = 5 * time.Minute
	finalizeRetryBase              = 30 * time.Second
	finalizeRetryMax               = 30 * time.Minute
	finalizeOldestDueQueryInterval = 30 * time.Second
	finalizeClaimBatchSize         = 32
)

var finalizeQuotaWheel = [...]finalizeQueue{
	finalizeHourlyDevice, finalizeHourlyDevice, finalizeHourlyDevice, finalizeHourlyOther,
	finalizeHourlyDevice, finalizeHourlyDevice, finalizeHourlyDevice, finalizeLongPeriod,
	finalizeHourlyDevice, finalizeHourlyDevice, finalizeHourlyDevice, finalizeHourlyOther,
	finalizeHourlyDevice, finalizeHourlyDevice, finalizeHourlyDevice, finalizeLongPeriod,
	finalizeHourlyDevice, finalizeHourlyDevice, finalizeHourlyOther, finalizeLongPeriod,
}

type finalizeClaimSelector struct {
	queueCursor    int
	newestNext     map[finalizeQueue]bool
	longTermCursor int
}

func newFinalizeClaimSelector() *finalizeClaimSelector {
	return &finalizeClaimSelector{newestNext: make(map[finalizeQueue]bool)}
}

func (s *finalizeClaimSelector) nextQueue(
	available map[finalizeQueue]bool,
) finalizeQueue {
	for offset := range len(finalizeQuotaWheel) {
		index := (s.queueCursor + offset) % len(finalizeQuotaWheel)
		queue := finalizeQuotaWheel[index]
		if available[queue] {
			s.queueCursor = (index + 1) % len(finalizeQuotaWheel)
			return queue
		}
	}
	return finalizeLongPeriod
}

func (s *finalizeClaimSelector) nextOrder(queue finalizeQueue) claimOrder {
	order := claimOldestFirst
	if s.newestNext[queue] {
		order = claimNewestFirst
	}
	s.newestNext[queue] = !s.newestNext[queue]
	return order
}

type finalizeJob struct {
	window    WindowRecord
	token     uuid.UUID
	queue     finalizeQueue
	replay    *deviceHourReplayCall
	replayKey *deviceHourReplayKey
}

type deviceHourReplayKey struct {
	entityKey string
	startNS   int64
	endNS     int64
}

type deviceHourReplayCall struct {
	once  sync.Once
	batch *DeviceHourReplayBatch
	err   error
}

type deviceHourReplayCache struct {
	limit   int
	entries map[deviceHourReplayKey]*deviceHourReplayCall
	pending map[deviceHourReplayKey]int
	order   []deviceHourReplayKey
}

func newDeviceHourReplayCache(limit int) *deviceHourReplayCache {
	if limit <= 0 {
		limit = 1
	}
	return &deviceHourReplayCache{
		limit: limit, entries: make(map[deviceHourReplayKey]*deviceHourReplayCall),
		pending: make(map[deviceHourReplayKey]int),
		order:   make([]deviceHourReplayKey, 0, limit),
	}
}

func (c *deviceHourReplayCache) acquire(
	key deviceHourReplayKey,
) *deviceHourReplayCall {
	call := c.entries[key]
	if call == nil {
		if len(c.entries) >= c.limit {
			for candidateIndex, candidate := range c.order {
				if c.pending[candidate] != 0 {
					continue
				}
				delete(c.entries, candidate)
				delete(c.pending, candidate)
				c.order = append(c.order[:candidateIndex], c.order[candidateIndex+1:]...)
				break
			}
		}
		call = &deviceHourReplayCall{}
		c.entries[key] = call
		c.order = append(c.order, key)
	}
	c.pending[key]++
	return call
}

func (c *deviceHourReplayCache) release(key deviceHourReplayKey) {
	if c.pending[key] > 0 {
		c.pending[key]--
	}
}

func (c *deviceHourReplayCall) run(
	ctx context.Context,
	key WindowKey,
	metrics *Metrics,
	load func(
		context.Context,
		WindowKey,
	) (*DeviceHourReplayBatch, error),
) (*DeviceHourReplayBatch, error) {
	c.once.Do(func() {
		if metrics != nil {
			metrics.DeviceHourReplayReadsTotal.Inc()
		}
		c.batch, c.err = load(ctx, key)
		if metrics != nil {
			if c.batch != nil {
				metrics.DeviceHourReplayVersionsTotal.Add(float64(len(c.batch.Contributions)))
			}
			if c.err != nil {
				metrics.DeviceHourReplayErrorsTotal.Inc()
			}
		}
	})
	return c.batch, c.err
}

type finalizeClaimAttempt struct {
	window   *WindowRecord
	token    uuid.UUID
	order    claimOrder
	conflict bool
}

type finalizeBatchClaim struct {
	windows  []WindowRecord
	token    uuid.UUID
	order    claimOrder
	conflict bool
}

type finalizeResult struct {
	key       WindowKey
	queue     finalizeQueue
	replayKey *deviceHourReplayKey
	err       error
}

func NewTimeoutScanner(
	windows *WindowRepository,
	finalizer *Finalizer,
	grace time.Duration,
	logger *zap.Logger,
) *TimeoutScanner {
	if logger == nil {
		logger = zap.NewNop()
	}
	workerCount := finalizer.Concurrency()
	return &TimeoutScanner{
		windows: windows, finalizeWindow: finalizer.FinalizeClaimed,
		publishReady: finalizer.PublishHourlyReady,
		snapshot:     finalizer.snapshot, workerCount: workerCount,
		grace: grace, logger: logger,
		claimLease: finalizeClaimLease, renewInterval: finalizeClaimLease / 3,
		oldestDueInterval: finalizeOldestDueQueryInterval,
		selector:          newFinalizeClaimSelector(),
		graceByGranularity: map[Granularity]time.Duration{
			GranularityHourly:  grace,
			GranularityDaily:   15 * time.Minute,
			GranularityWeekly:  30 * time.Minute,
			GranularityMonthly: 30 * time.Minute,
		},
	}
}

func (s *TimeoutScanner) SetGranularityGrace(
	daily, weekly, monthly time.Duration,
) *TimeoutScanner {
	if daily > 0 {
		s.graceByGranularity[GranularityDaily] = daily
	}
	if weekly > 0 {
		s.graceByGranularity[GranularityWeekly] = weekly
	}
	if monthly > 0 {
		s.graceByGranularity[GranularityMonthly] = monthly
	}
	return s
}

func (s *TimeoutScanner) SetIncompleteDeviceHourReplay(
	replay func(context.Context, WindowKey) (bool, error),
) *TimeoutScanner {
	s.legacyReplayDeviceHour = replay
	s.loadDeviceHour = nil
	s.applyDeviceHour = nil
	return s
}

func (s *TimeoutScanner) SetIncompleteDeviceHourVersionReplay(
	load func(
		context.Context,
		WindowKey,
	) (*DeviceHourReplayBatch, error),
	apply func(
		context.Context,
		WindowKey,
		*DeviceHourReplayBatch,
	) (DeviceHourReplayOutcome, error),
) *TimeoutScanner {
	s.legacyReplayDeviceHour = nil
	s.loadDeviceHour = load
	s.applyDeviceHour = apply
	return s
}

func (s *TimeoutScanner) SetMetrics(metrics *Metrics) *TimeoutScanner {
	s.metrics = metrics
	return s
}

func (s *TimeoutScanner) Run(ctx context.Context) {
	workerCount := s.workerCount
	jobs := make(chan finalizeJob)
	results := make(chan finalizeResult, workerCount)
	var workers sync.WaitGroup
	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			s.finalizeWorker(ctx, jobs, results)
		}()
	}
	defer func() {
		close(jobs)
		workers.Wait()
	}()
	oldestDueDone := make(chan struct{})
	go func() {
		defer close(oldestDueDone)
		s.runOldestDueMonitor(ctx)
	}()
	defer func() { <-oldestDueDone }()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.runCycle(ctx, jobs, results, workerCount); err != nil && ctx.Err() == nil {
			s.logger.Warn("scan timed out PM aggregation windows", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *TimeoutScanner) runOnce(ctx context.Context) error {
	workerCount := s.workerCount
	jobs := make(chan finalizeJob)
	results := make(chan finalizeResult, workerCount)
	var workers sync.WaitGroup
	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			s.finalizeWorker(ctx, jobs, results)
		}()
	}
	err := s.runCycle(ctx, jobs, results, workerCount)
	close(jobs)
	workers.Wait()
	return errors.Join(err, s.refreshOldestDue(ctx))
}

func (s *TimeoutScanner) runCycle(
	ctx context.Context,
	jobs chan<- finalizeJob,
	results <-chan finalizeResult,
	workerCount int,
) error {
	now := time.Now().UTC()
	if s.publishReady != nil {
		if err := s.publishReady(ctx, now, s.graceFor(GranularityHourly), finalizeClaimBatchSize); err != nil {
			return err
		}
	}
	blocked, err := s.windows.CountWatermarkBlocked(
		ctx, now, s.graceByGranularity, s.grace,
	)
	if err != nil {
		return err
	}
	if blocked > 0 && s.metrics != nil {
		s.metrics.WatermarkBlockedTotal.Add(float64(blocked))
	}
	available := map[finalizeQueue]bool{
		finalizeHourlyDevice: true,
		finalizeHourlyOther:  true,
		finalizeLongPeriod:   true,
	}
	inflight := 0
	claimsSinceRefresh := 0
	// Claim ordering keeps task versions for one device-hour adjacent. Retain a
	// small bounded FIFO beyond the in-flight jobs so even one-worker scanners
	// reuse the decoded source across sequential version claims. The entire
	// cache is cycle-scoped and never grows with a 10k-device backlog.
	deviceHourReplays := newDeviceHourReplayCache(finalizeClaimBatchSize)
	var finalizeErrors []error
	for {
		openSlots := workerCount - inflight
		refillThreshold := min(finalizeClaimBatchSize, workerCount)
		// Claiming one window per transaction made a 20k-device hour spend most
		// of its time in serialized claim SQL (~133ms per round trip). Refill in
		// bounded waves so one SKIP LOCKED transaction leases up to eight
		// windows, while retaining a fixed worker ceiling and queue fairness.
		for openSlots >= refillThreshold && anyFinalizeQueueAvailable(available) {
			queue := s.selector.nextQueue(available)
			claimAt := time.Now().UTC()
			batchLimit := min(finalizeClaimBatchSize, openSlots)
			attempt, claimErr := s.claimBatch(ctx, claimAt, queue, uint64(batchLimit))
			if claimErr != nil {
				finalizeErrors = append(finalizeErrors, claimErr)
				available[queue] = false
				continue
			}
			if len(attempt.windows) == 0 {
				// Per-job tokens make an in-flight local claim indistinguishable
				// from a foreign claim in the aggregate conflict probe.
				reportedConflict := attempt.conflict && inflight == 0
				if reportedConflict && s.metrics != nil {
					s.metrics.FinalizeClaimConflictsTotal.Inc()
				}
				available[queue] = false
				continue
			}
			claimsSinceRefresh += len(attempt.windows)
			if claimsSinceRefresh >= len(finalizeQuotaWheel) {
				available[finalizeHourlyDevice] = true
				available[finalizeHourlyOther] = true
				available[finalizeLongPeriod] = true
				claimsSinceRefresh %= len(finalizeQuotaWheel)
			}
			if s.metrics != nil {
				s.metrics.FinalizeClaims.Add(float64(len(attempt.windows)))
			}
			for index := range attempt.windows {
				window := attempt.windows[index]
				var replay *deviceHourReplayCall
				var replayKey *deviceHourReplayKey
				if queue == finalizeHourlyDevice && s.loadDeviceHour != nil &&
					(window.ExpectedSlots <= 0 || window.ReceivedSlots < window.ExpectedSlots) {
					key := deviceHourReplayKey{
						entityKey: window.Key.EntityKey,
						startNS:   window.Key.Start.UnixNano(),
						endNS:     window.Key.End.UnixNano(),
					}
					replay = deviceHourReplays.acquire(key)
					replayKey = &key
				}
				select {
				case jobs <- finalizeJob{
					window: window, token: attempt.token, queue: queue,
					replay: replay, replayKey: replayKey,
				}:
					inflight++
					openSlots--
					if s.metrics != nil {
						s.metrics.FinalizeInflight.Inc()
					}
				case <-ctx.Done():
					clearCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					var releaseErrors []error
					for pending := index; pending < len(attempt.windows); pending++ {
						releaseErrors = append(
							releaseErrors,
							s.windows.ReleaseClaim(
								clearCtx, attempt.windows[pending].Key, attempt.token,
							),
						)
					}
					cancel()
					return errors.Join(append(finalizeErrors, ctx.Err(), errors.Join(releaseErrors...))...)
				}
			}
		}
		if inflight == 0 {
			return errors.Join(finalizeErrors...)
		}
		select {
		case result := <-results:
			inflight--
			if result.replayKey != nil {
				key := *result.replayKey
				deviceHourReplays.release(key)
			}
			if result.err != nil {
				finalizeErrors = append(finalizeErrors, result.err)
				s.logger.Warn("PM aggregation finalize job failed; continuing healthy queues",
					zap.String("queue", finalizeQueueName(result.queue)),
					zap.String("task_version_id", result.key.TaskVersionID.String()),
					zap.String("entity_key", result.key.EntityKey),
					zap.String("granularity", string(result.key.Granularity)),
					zap.Time("window_start", result.key.Start),
					zap.Error(result.err))
			}
		case <-ctx.Done():
			return errors.Join(append(finalizeErrors, ctx.Err())...)
		}
	}
}

func anyFinalizeQueueAvailable(available map[finalizeQueue]bool) bool {
	return available[finalizeHourlyDevice] ||
		available[finalizeHourlyOther] ||
		available[finalizeLongPeriod]
}

func (s *TimeoutScanner) claimNext(
	ctx context.Context,
	now time.Time,
	queue finalizeQueue,
) (finalizeClaimAttempt, error) {
	batch, err := s.claimBatch(ctx, now, queue, 1)
	if err != nil {
		return finalizeClaimAttempt{}, err
	}
	var window *WindowRecord
	if len(batch.windows) > 0 {
		window = &batch.windows[0]
	}
	return finalizeClaimAttempt{
		window: window, token: batch.token, order: batch.order, conflict: batch.conflict,
	}, nil
}

func (s *TimeoutScanner) claimBatch(
	ctx context.Context,
	now time.Time,
	queue finalizeQueue,
	limit uint64,
) (finalizeBatchClaim, error) {
	token := uuid.New()
	leaseUntil := now.Add(s.claimLeaseDuration())
	switch queue {
	case finalizeHourlyDevice, finalizeHourlyOther:
		filter := claimVersionFilter{
			versionIDs: s.hourlyDeviceVersionIDs(),
			exclude:    queue == finalizeHourlyOther,
		}
		order := s.selector.nextOrder(queue)
		dueBefore := preparationDueBefore(now, GranularityHourly, s.graceFor(GranularityHourly))
		windows, err := s.windows.claimDue(
			ctx, GranularityHourly, dueBefore, limit,
			token, leaseUntil, filter, order,
		)
		if err != nil {
			return finalizeBatchClaim{}, err
		}
		if len(windows) == 0 {
			conflict, conflictErr := s.windows.hasClaimConflict(
				ctx, GranularityHourly, dueBefore, filter, token,
			)
			return finalizeBatchClaim{order: order, conflict: conflict}, conflictErr
		}
		return finalizeBatchClaim{
			windows: windows, token: token, order: order,
		}, nil
	case finalizeLongPeriod:
		granularities := [...]Granularity{
			GranularityDaily, GranularityWeekly, GranularityMonthly,
		}
		conflict := false
		for range len(granularities) {
			granularity := granularities[s.selector.longTermCursor%len(granularities)]
			s.selector.longTermCursor++
			dueBefore := preparationDueBefore(now, granularity, s.graceFor(granularity))
			windows, err := s.windows.claimDue(
				ctx, granularity, dueBefore, limit,
				token, leaseUntil, claimVersionFilter{}, claimOldestFirst,
			)
			if err != nil {
				return finalizeBatchClaim{}, err
			}
			if len(windows) > 0 {
				return finalizeBatchClaim{
					windows: windows, token: token, order: claimOldestFirst,
				}, nil
			}
			granularityConflict, conflictErr := s.windows.hasClaimConflict(
				ctx, granularity, dueBefore, claimVersionFilter{}, token,
			)
			if conflictErr != nil {
				return finalizeBatchClaim{}, conflictErr
			}
			conflict = conflict || granularityConflict
		}
		return finalizeBatchClaim{order: claimOldestFirst, conflict: conflict}, nil
	default:
		return finalizeBatchClaim{}, fmt.Errorf("unsupported PM finalize queue %d", queue)
	}
}

func preparationDueBefore(now time.Time, granularity Granularity, grace time.Duration) time.Time {
	if granularity == GranularityHourly {
		return now
	}
	return now.Add(-grace)
}

func (s *TimeoutScanner) claimLeaseDuration() time.Duration {
	if s.claimLease > 0 {
		return s.claimLease
	}
	return finalizeClaimLease
}

func (s *TimeoutScanner) claimRenewInterval() time.Duration {
	if s.renewInterval > 0 {
		return s.renewInterval
	}
	return s.claimLeaseDuration() / 3
}

func (s *TimeoutScanner) refreshOldestDue(ctx context.Context) error {
	if s.metrics == nil {
		return nil
	}
	oldest, err := s.windows.OldestDue(ctx, s.graceByGranularity, s.grace)
	if err != nil {
		return fmt.Errorf("query oldest due PM aggregation window: %w", err)
	}
	s.metrics.FinalizeOldestDueSeconds.Set(oldest.Seconds())
	return nil
}

func (s *TimeoutScanner) runOldestDueMonitor(ctx context.Context) {
	interval := s.oldestDueInterval
	if interval <= 0 {
		interval = finalizeOldestDueQueryInterval
	}
	refresh := func() {
		if err := s.refreshOldestDue(ctx); err != nil && ctx.Err() == nil {
			s.logger.Warn("refresh PM aggregation oldest due metric", zap.Error(err))
		}
	}
	refresh()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}

func finalizeQueueName(queue finalizeQueue) string {
	switch queue {
	case finalizeHourlyDevice:
		return "hourly_device"
	case finalizeHourlyOther:
		return "hourly_other"
	case finalizeLongPeriod:
		return "long_period"
	default:
		return fmt.Sprintf("unknown_%d", queue)
	}
}

func finalizeRetryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return finalizeRetryBase
	}
	delay := finalizeRetryBase
	for range attempt - 1 {
		if delay >= finalizeRetryMax/2 {
			return finalizeRetryMax
		}
		delay *= 2
	}
	return min(delay, finalizeRetryMax)
}

func (s *TimeoutScanner) graceFor(granularity Granularity) time.Duration {
	if grace := s.graceByGranularity[granularity]; grace > 0 {
		return grace
	}
	return s.grace
}

func (s *TimeoutScanner) hourlyDeviceVersionIDs() []uuid.UUID {
	if s.snapshot == nil || s.snapshot.Current() == nil {
		return []uuid.UUID{}
	}
	current := s.snapshot.Current()
	if current == s.deviceVersionsFor {
		return s.deviceVersionIDs
	}
	var ids []uuid.UUID
	for versionID, version := range current.ByVersion {
		if version != nil && version.DevicePipeline {
			ids = append(ids, versionID)
		}
	}
	sort.Slice(ids, func(left, right int) bool {
		return ids[left].String() < ids[right].String()
	})
	s.deviceVersionsFor = current
	s.deviceVersionIDs = ids
	return s.deviceVersionIDs
}

func (s *TimeoutScanner) finalizeWorker(
	ctx context.Context,
	jobs <-chan finalizeJob,
	results chan<- finalizeResult,
) {
	for job := range jobs {
		window := job.window
		reason := CloseTimeout
		if window.ExpectedSlots > 0 && window.ReceivedSlots >= window.ExpectedSlots {
			reason = CloseComplete
		}
		finalizeCtx, cancelFinalize := context.WithCancel(ctx)
		renewDone := make(chan error, 1)
		go func() {
			renewErr := s.renewFinalizeClaim(finalizeCtx, job)
			if renewErr != nil {
				cancelFinalize()
			}
			renewDone <- renewErr
		}()
		var err error
		if reason == CloseTimeout && job.queue == finalizeHourlyDevice {
			var complete bool
			switch {
			case job.replay != nil && s.loadDeviceHour != nil && s.applyDeviceHour != nil:
				var batch *DeviceHourReplayBatch
				batch, err = job.replay.run(
					finalizeCtx, window.Key, s.metrics, s.loadDeviceHour,
				)
				if err == nil {
					var outcome DeviceHourReplayOutcome
					outcome, err = s.applyDeviceHour(finalizeCtx, window.Key, batch)
					complete = outcome.Complete
					if err != nil && s.metrics != nil {
						s.metrics.DeviceHourReplayErrorsTotal.Inc()
					}
				}
			case s.legacyReplayDeviceHour != nil:
				complete, err = s.legacyReplayDeviceHour(finalizeCtx, window.Key)
			}
			if err == nil && complete {
				reason = CloseComplete
			}
		}
		if err == nil {
			err = s.finalizeWindow(finalizeCtx, window.Key, reason, job.token)
		}
		cancelFinalize()
		err = errors.Join(err, <-renewDone)
		clearCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err != nil {
			s.logger.Warn("finalize timed out PM aggregation window",
				zap.String("task_version_id", window.Key.TaskVersionID.String()),
				zap.String("entity_key", window.Key.EntityKey),
				zap.Time("window_start", window.Key.Start),
				zap.Error(err))
			clearErr := s.windows.FailClaim(
				clearCtx,
				window.Key,
				job.token,
				err,
				finalizeRetryDelay(window.FinalizeAttempts+1),
			)
			err = errors.Join(err, clearErr)
		} else {
			err = s.windows.CompleteClaim(clearCtx, window.Key, job.token)
		}
		cancel()
		if s.metrics != nil {
			s.metrics.FinalizeInflight.Dec()
		}
		results <- finalizeResult{
			key: window.Key, queue: job.queue, replayKey: job.replayKey, err: err,
		}
	}
}

func (s *TimeoutScanner) renewFinalizeClaim(
	ctx context.Context,
	job finalizeJob,
) error {
	ticker := time.NewTicker(s.claimRenewInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			err := s.windows.RenewClaim(
				ctx, job.window.Key, job.token, now.Add(s.claimLeaseDuration()),
			)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("renew PM aggregation finalize claim: %w", err)
			}
		}
	}
}
