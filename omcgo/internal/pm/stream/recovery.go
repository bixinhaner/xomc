package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

const (
	aggregationRawStreamName    = "PM_AGG_15M"
	aggregationHourlyStreamName = "PM_AGG_HOURLY"
	aggregationDailyStreamName  = "PM_AGG_DAILY"
	recoveryPageSize            = uint64(1000)
	recoveryTerminalBatchLimit  = 512
	recoveryRuntimeCleanupLimit = uint64(256)
	recoveryRuntimeCleanupTime  = 10 * time.Second
)

type recoverySource string

type recoveryTerminal struct {
	status string
	reason string
}

type RecoveryVersionStateLoader interface {
	LoadRecoveryVersionStates(
		ctx context.Context,
		versionIDs []uuid.UUID,
	) (map[uuid.UUID]RecoveryVersionState, error)
}

const (
	recoveryRaw15m      recoverySource = "raw_15m"
	recoveryDeviceHours recoverySource = "device_hours"
	recoveryHourly      recoverySource = "hourly"
	recoveryDaily       recoverySource = "daily"
)

type Recovery struct {
	js       nats.JetStreamContext
	windows  *WindowRepository
	store    *RedisWindowStore
	snapshot *SnapshotStore
	matcher  *Matcher
	rollups  *RollupOutboxRepository
	outbox   *OutboxRepository
	versions RecoveryVersionStateLoader
	logger   *zap.Logger
}

func NewRecovery(
	js nats.JetStreamContext,
	windows *WindowRepository,
	store *RedisWindowStore,
	snapshot *SnapshotStore,
	matcher *Matcher,
	logger *zap.Logger,
) *Recovery {
	if logger == nil {
		logger = zap.NewNop()
	}
	recovery := &Recovery{
		js: js, windows: windows, store: store, snapshot: snapshot,
		matcher: matcher, rollups: NewRollupOutboxRepository(windows.pool),
		outbox: NewOutboxRepository(windows.pool), logger: logger,
	}
	if snapshot != nil {
		recovery.versions, _ = snapshot.loader.(RecoveryVersionStateLoader)
	}
	return recovery
}

func (r *Recovery) RestoreActiveWindows(ctx context.Context) error {
	if r.snapshot == nil {
		return errors.New("PM aggregation recovery task snapshot is nil")
	}
	if r.versions == nil {
		return errors.New("PM aggregation recovery version-state loader is nil")
	}
	if err := r.snapshot.Refresh(ctx); err != nil {
		return fmt.Errorf("refresh PM aggregation tasks before recovery: %w", err)
	}
	current := r.snapshot.Current()
	var restoreErrors []error
	var cursor *WindowKey
	missing := make(map[recoverySource][]WindowKey)
	terminalRemaining := recoveryTerminalBatchLimit
	// 大数据升级恢复时活跃窗口以十万计，本扫描 = 每页一条查询、可能持续数十分钟。
	// 每 progressEveryPages 页打一次进度；收尾摘要仅在实际耗时超过阈值时输出
	// （周期性 Run 的稳态扫描只有一两页，保持静默避免刷屏）。
	const progressEveryPages = 25
	const restoreSummaryLogThreshold = 5 * time.Second
	restoreStart := time.Now()
	pages, scanned := 0, 0
	for {
		active, err := r.windows.ListActiveAfter(ctx, cursor, recoveryPageSize)
		if err != nil {
			return errors.Join(append(restoreErrors, err)...)
		}
		pages++
		scanned += len(active)
		versionIDs := make([]uuid.UUID, 0, len(active))
		for _, record := range active {
			version := current.ByVersion[record.Key.TaskVersionID]
			if version != nil && (version.DevicePipeline || version.DeviceRollup) {
				continue
			}
			versionIDs = append(versionIDs, record.Key.TaskVersionID)
		}
		versionStates, err := r.versions.LoadRecoveryVersionStates(ctx, versionIDs)
		if err != nil {
			return errors.Join(append(restoreErrors, err)...)
		}
		if pages%progressEveryPages == 0 {
			r.logger.Info("PM aggregation active-window restore in progress",
				zap.Int("pages", pages),
				zap.Int("scanned_windows", scanned),
				zap.Duration("elapsed", time.Since(restoreStart)))
		}
		terminals := make(map[recoveryTerminal][]WindowRecord)
		for _, record := range active {
			terminal, invalid := recoveryTerminalFor(record.Key, current, versionStates)
			if invalid {
				if terminalRemaining > 0 {
					terminals[terminal] = append(terminals[terminal], record)
					terminalRemaining--
				}
				continue
			}
			exists, existsErr := r.store.Exists(ctx, record.Key)
			if existsErr != nil {
				restoreErrors = append(restoreErrors, existsErr)
				r.logger.Error("check PM aggregation window before recovery",
					zap.String("task_version_id", record.Key.TaskVersionID.String()),
					zap.String("entity_key", record.Key.EntityKey),
					zap.Time("window_start", record.Key.Start),
					zap.Error(existsErr))
				continue
			}
			if exists {
				continue
			}
			source, sourceErr := recoverySourceForSnapshot(record.Key, current)
			if sourceErr != nil {
				restoreErrors = append(restoreErrors, sourceErr)
				_ = r.windows.MarkFailed(ctx, record.Key, sourceErr)
				continue
			}
			missing[source] = append(missing[source], record.Key)
		}
		for terminal, records := range terminals {
			updated, terminalErr := r.windows.MarkRecoveryTerminalBatch(
				ctx, records, terminal.status, terminal.reason,
			)
			if terminalErr != nil {
				restoreErrors = append(restoreErrors, terminalErr)
				continue
			}
			if updated > 0 {
				r.logger.Warn("PM aggregation recovery isolated invalid task windows",
					zap.String("status", terminal.status),
					zap.String("reason", terminal.reason),
					zap.Int64("windows", updated))
			}
		}
		if len(active) < int(recoveryPageSize) {
			break
		}
		last := active[len(active)-1].Key
		cursor = &last
	}
	if cleanupErr := r.cleanupRecoveryRuntime(ctx); cleanupErr != nil {
		restoreErrors = append(restoreErrors, cleanupErr)
	}
	if elapsed := time.Since(restoreStart); elapsed > restoreSummaryLogThreshold {
		missingTotal := 0
		for _, keys := range missing {
			missingTotal += len(keys)
		}
		r.logger.Info("PM aggregation active-window restore scan completed",
			zap.Int("pages", pages),
			zap.Int("scanned_windows", scanned),
			zap.Int("missing_windows", missingTotal),
			zap.Duration("duration", elapsed))
	}
	for source, keys := range missing {
		matched, replayErr := r.replayWindowBatch(ctx, source, keys)
		if replayErr != nil {
			restoreErrors = append(restoreErrors, replayErr)
		}
		for _, key := range keys {
			if _, ok := matched[windowRecoveryID(key)]; ok {
				continue
			}
			missingErr := fmt.Errorf(
				"no retained %s PM events matched active %s window %s",
				source, key.Granularity, key.EntityKey,
			)
			if replayErr != nil {
				missingErr = fmt.Errorf("%w: %v", missingErr, replayErr)
			}
			_ = r.windows.MarkFailed(ctx, key, missingErr)
			r.logger.Error("restore PM aggregation window",
				zap.String("task_version_id", key.TaskVersionID.String()),
				zap.String("entity_key", key.EntityKey),
				zap.Time("window_start", key.Start),
				zap.Error(missingErr))
		}
	}
	return errors.Join(restoreErrors...)
}

func recoveryTerminalFor(
	key WindowKey,
	snapshot *TaskSnapshot,
	states map[uuid.UUID]RecoveryVersionState,
) (recoveryTerminal, bool) {
	if snapshot == nil {
		return recoveryTerminal{status: "orphaned", reason: "task snapshot missing"}, true
	}
	version := snapshot.ByVersion[key.TaskVersionID]
	if version != nil && (version.DevicePipeline || version.DeviceRollup) {
		return recoveryTerminalForState(key, RecoveryVersionState{
			TaskID: version.TaskID, TaskEnabled: version.TaskEnabled,
			TaskDeletedAt: version.TaskDeletedAt, VersionID: version.VersionID,
			VersionEnabled: version.Enabled, EffectiveFrom: version.EffectiveFrom,
			EffectiveTo: version.EffectiveTo, PlannedEndAt: version.PlannedEndAt,
		})
	}
	state, exists := states[key.TaskVersionID]
	if !exists {
		return recoveryTerminal{status: "orphaned", reason: "task version missing from primary database"}, true
	}
	return recoveryTerminalForState(key, state)
}

func recoveryTerminalForState(
	key WindowKey,
	state RecoveryVersionState,
) (recoveryTerminal, bool) {
	if state.TaskID != key.TaskID {
		return recoveryTerminal{status: "orphaned", reason: "task identity does not match task version"}, true
	}
	if state.TaskDeletedAt != nil {
		return recoveryTerminal{status: "retired", reason: "task was deleted"}, true
	}
	if !state.TaskEnabled {
		return recoveryTerminal{status: "retired", reason: "task is disabled"}, true
	}
	if !state.VersionEnabled {
		return recoveryTerminal{status: "retired", reason: "task version is disabled"}, true
	}
	if state.PlannedEndAt != nil && !key.Start.Before(*state.PlannedEndAt) {
		return recoveryTerminal{status: "retired", reason: "task planned end was reached"}, true
	}
	if !state.EffectiveFrom.IsZero() && !key.End.After(state.EffectiveFrom) {
		return recoveryTerminal{status: "retired", reason: "window is before task version effective range"}, true
	}
	if state.EffectiveTo != nil && !key.Start.Before(*state.EffectiveTo) {
		return recoveryTerminal{status: "retired", reason: "window is after task version effective range"}, true
	}
	return recoveryTerminal{}, false
}

func (r *Recovery) cleanupRecoveryRuntime(ctx context.Context) error {
	pending, err := r.windows.ListRecoveryRuntimeCleanupPending(
		ctx, recoveryRuntimeCleanupLimit,
	)
	if err != nil || len(pending) == 0 {
		return err
	}
	cleanupCtx, cancel := context.WithTimeout(ctx, recoveryRuntimeCleanupTime)
	defer cancel()
	started := time.Now()
	cleaned := make([]WindowKey, 0, len(pending))
	versionIDs := make([]uuid.UUID, 0, len(pending))
	var cleanupErr error
	for _, record := range pending {
		if err := r.store.DeleteState(cleanupCtx, record.Key); err != nil {
			cleanupErr = err
			break
		}
		cleaned = append(cleaned, record.Key)
		versionIDs = append(versionIDs, record.Key.TaskVersionID)
	}
	if cleanupErr == nil {
		cleanupErr = r.store.DeleteVersionDefinitions(cleanupCtx, versionIDs)
	}
	if cleanupErr == nil && len(cleaned) > 0 {
		if _, err := r.windows.MarkRecoveryRuntimeCleaned(ctx, cleaned); err != nil {
			cleanupErr = err
		}
	}
	r.logger.Info("PM aggregation recovery runtime cleanup completed",
		zap.Int("selected_windows", len(pending)),
		zap.Int("cleaned_windows", len(cleaned)),
		zap.Duration("duration", time.Since(started)))
	if cleanupErr != nil {
		return fmt.Errorf("clean PM aggregation recovery runtime: %w", cleanupErr)
	}
	return nil
}

func (r *Recovery) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.RestoreActiveWindows(ctx); err != nil && ctx.Err() == nil {
				r.logger.Error("periodic PM aggregation recovery", zap.Error(err))
			}
		}
	}
}

func (r *Recovery) ReplayWindow(ctx context.Context, key WindowKey) error {
	if r.js == nil {
		return errors.New("PM aggregation recovery JetStream is nil")
	}
	source, err := r.recoverySourceFor(key)
	if err != nil {
		return err
	}
	matched, err := r.replayWindowBatch(ctx, source, []WindowKey{key})
	if err != nil {
		return err
	}
	if _, ok := matched[windowRecoveryID(key)]; !ok {
		return fmt.Errorf("no retained %s PM events matched active window", source)
	}
	return nil
}

// ReplayDurableDeviceHour idempotently replays retained normalized PM events
// into an incomplete synthetic device hour immediately before publication.
// Existing source_file_id values are ignored by Redis, so this can only fill
// missing slots; it cannot double-count the slots that were already accepted.
func (r *Recovery) ReplayDurableDeviceHour(
	ctx context.Context,
	key WindowKey,
) (bool, error) {
	if r == nil || r.outbox == nil || r.store == nil ||
		r.snapshot == nil || r.matcher == nil {
		return false, fmt.Errorf("durable PM device-hour replay is not configured")
	}
	snapshot := r.snapshot.Current()
	var version *TaskVersionSnapshot
	if snapshot != nil {
		version = snapshot.ByVersion[key.TaskVersionID]
	}
	if version == nil || !version.DevicePipeline || key.Granularity != GranularityHourly {
		return false, nil
	}

	_, complete, matched, err := replayDurableDeviceHourSources(
		ctx,
		key,
		snapshot,
		r.matcher,
		r.outbox.VisitPayloadsForPeriod,
		r.store.Accumulate,
	)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, nil
	}
	return complete, nil
}

type durablePayloadVisitor func(
	context.Context,
	time.Time,
	time.Time,
	func(event.PMAggregationNormalizedPayload) error,
) error

type contributionAccumulatorFunc func(
	context.Context,
	Contribution,
) (AccumulateResult, error)

func replayDurableDeviceHourSources(
	ctx context.Context,
	key WindowKey,
	snapshot *TaskSnapshot,
	matcher *Matcher,
	visit durablePayloadVisitor,
	accumulate contributionAccumulatorFunc,
) (received int64, complete bool, matched bool, err error) {
	if snapshot == nil || matcher == nil || visit == nil || accumulate == nil {
		return 0, false, false, fmt.Errorf("durable PM device-hour replay dependencies are incomplete")
	}
	err = visit(ctx, key.Start, key.End, func(payload event.PMAggregationNormalizedPayload) error {
		if payload.DeviceID.String() != key.EntityKey {
			return nil
		}
		contributions, matchErr := matcher.MatchGranularity(
			payload, snapshot, GranularityHourly,
		)
		if matchErr != nil {
			return fmt.Errorf("match durable PM device-hour source: %w", matchErr)
		}
		for _, contribution := range contributions {
			if !sameWindowKey(contribution.Key, key) {
				continue
			}
			result, accumulateErr := accumulate(ctx, contribution)
			if accumulateErr != nil {
				return fmt.Errorf("accumulate durable PM device-hour source: %w", accumulateErr)
			}
			matched = true
			if result.ReceivedSlots > received {
				received = result.ReceivedSlots
			}
			complete = complete || result.Complete
		}
		return nil
	})
	if err != nil {
		return 0, false, matched, fmt.Errorf("replay durable PM device-hour sources: %w", err)
	}
	return received, complete, matched, nil
}

func (r *Recovery) recoverySourceFor(key WindowKey) (recoverySource, error) {
	if r.snapshot == nil {
		return "", errors.New("PM aggregation task version is not recoverable: task snapshot missing")
	}
	current := r.snapshot.Current()
	version := current.ByVersion[key.TaskVersionID]
	if version == nil {
		return "", errors.New("PM aggregation task version is not recoverable: task version missing from snapshot")
	}
	if terminal, invalid := recoveryTerminalForState(key, RecoveryVersionState{
		TaskID: version.TaskID, TaskEnabled: version.TaskEnabled,
		TaskDeletedAt: version.TaskDeletedAt, VersionID: version.VersionID,
		VersionEnabled: version.Enabled, EffectiveFrom: version.EffectiveFrom,
		EffectiveTo: version.EffectiveTo, PlannedEndAt: version.PlannedEndAt,
	}); invalid {
		return "", fmt.Errorf(
			"PM aggregation task version is not recoverable: %s", terminal.reason,
		)
	}
	return recoverySourceForSnapshot(key, current)
}

func recoverySourceForSnapshot(
	key WindowKey,
	current *TaskSnapshot,
) (recoverySource, error) {
	version := current.ByVersion[key.TaskVersionID]
	switch key.Granularity {
	case GranularityHourly:
		if version != nil && version.DevicePipeline {
			return recoveryRaw15m, nil
		}
		return recoveryDeviceHours, nil
	case GranularityDaily:
		return recoveryHourly, nil
	case GranularityWeekly, GranularityMonthly:
		return recoveryDaily, nil
	default:
		return "", fmt.Errorf("unsupported PM aggregation recovery granularity %q", key.Granularity)
	}
}

func (r *Recovery) replayWindowBatch(
	ctx context.Context,
	source recoverySource,
	keys []WindowKey,
) (map[string]struct{}, error) {
	matched := make(map[string]struct{}, len(keys))
	if len(keys) == 0 {
		return matched, nil
	}
	if r.js == nil {
		return matched, errors.New("PM aggregation recovery JetStream is nil")
	}
	wanted := make(map[string]struct{}, len(keys))
	start := keys[0].Start
	for _, key := range keys {
		wanted[windowRecoveryID(key)] = struct{}{}
		if key.Start.Before(start) {
			start = key.Start
		}
	}
	streamName, subject := aggregationRawStreamName, event.SubjectPMAggregationNormalized
	if source == recoveryDeviceHours || source == recoveryHourly {
		streamName, subject = aggregationHourlyStreamName, event.SubjectPMAggregationHourlyRollup
	} else if source == recoveryDaily {
		streamName, subject = aggregationDailyStreamName, event.SubjectPMAggregationDailyRollup
	}
	subscription, err := r.js.PullSubscribe(
		subject, "", nats.BindStream(streamName), nats.StartTime(start),
		nats.AckNone(),
	)
	if err != nil {
		return matched, fmt.Errorf("create PM aggregation batch replay consumer: %w", err)
	}
	defer func() { _ = subscription.Unsubscribe() }()

	for {
		messages, fetchErr := subscription.Fetch(256, nats.MaxWait(time.Second))
		if fetchErr != nil && !errors.Is(fetchErr, nats.ErrTimeout) {
			return matched, fmt.Errorf("fetch PM aggregation batch replay messages: %w", fetchErr)
		}
		for _, message := range messages {
			var envelope event.Event
			if err := json.Unmarshal(message.Data, &envelope); err != nil {
				return matched, fmt.Errorf("decode replay PM aggregation envelope: %w", err)
			}
			contributions, decodeErr := r.recoveryContributions(source, envelope)
			if decodeErr != nil {
				return matched, decodeErr
			}
			for _, contribution := range contributions {
				id := windowRecoveryID(contribution.Key)
				if _, ok := wanted[id]; !ok {
					continue
				}
				if _, err := r.store.Accumulate(ctx, contribution); err != nil {
					return matched, err
				}
				matched[id] = struct{}{}
			}
		}
		if errors.Is(fetchErr, nats.ErrTimeout) || len(messages) == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return matched, ctx.Err()
		default:
		}
	}
	return matched, nil
}

func (r *Recovery) recoveryContributions(
	source recoverySource,
	envelope event.Event,
) ([]Contribution, error) {
	current := r.snapshot.Current()
	if current == nil {
		return nil, fmt.Errorf("PM aggregation task snapshot missing")
	}
	if source == recoveryRaw15m {
		var payload event.PMAggregationNormalizedPayload
		if err := envelope.DecodePayload(&payload); err != nil {
			return nil, fmt.Errorf("decode replay PM aggregation payload: %w", err)
		}
		return r.matcher.MatchGranularity(payload, current, GranularityHourly)
	}
	var payload RollupPayload
	if err := envelope.DecodePayload(&payload); err != nil {
		return nil, fmt.Errorf("decode compact rollup replay payload: %w", err)
	}
	if source == recoveryDeviceHours {
		if !isDeviceHourPayload(payload, current) {
			return nil, nil
		}
		return matchDeviceHourRules(payload, current, r.matcher.Location())
	}
	return rollupContributions(
		payload, current.ByVersion[payload.TaskVersionID], r.matcher.Location(),
	)
}

func windowRecoveryID(key WindowKey) string {
	return fmt.Sprintf(
		"%s\x1f%s\x1f%s\x1f%d",
		key.TaskVersionID, key.EntityKey, key.Granularity, key.Start.UTC().UnixNano(),
	)
}

func sameWindowKey(left, right WindowKey) bool {
	return left.TaskVersionID == right.TaskVersionID &&
		left.EntityKey == right.EntityKey &&
		left.Granularity == right.Granularity &&
		left.Start.Equal(right.Start)
}
