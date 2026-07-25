package stream

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"
)

const defaultRollupBatchValues = 500

// DirectRollupPromoter merges finalized compact Counter states into parent
// Redis windows. It deliberately does not publish rollup messages and never
// forwards KPI values or original 15-minute samples.
type DirectRollupPromoter struct {
	snapshot  *SnapshotStore
	windows   *WindowRepository
	store     *RedisWindowStore
	finalizer *Finalizer
	location  *time.Location
	logger    *zap.Logger
	batch     int
}

func NewDirectRollupPromoter(
	snapshot *SnapshotStore,
	windows *WindowRepository,
	store *RedisWindowStore,
	finalizer *Finalizer,
	location *time.Location,
	logger *zap.Logger,
) *DirectRollupPromoter {
	if location == nil {
		location = time.UTC
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DirectRollupPromoter{
		snapshot: snapshot, windows: windows, store: store, finalizer: finalizer,
		location: location, logger: logger, batch: defaultRollupBatchValues,
	}
}

func (p *DirectRollupPromoter) Promote(
	ctx context.Context,
	childKey WindowKey,
	_ CloseReason,
	childState WindowState,
) error {
	current := p.snapshot.Current()
	if current == nil {
		return fmt.Errorf("PM aggregation task snapshot missing")
	}
	version := current.ByVersion[childKey.TaskVersionID]
	if version == nil {
		return fmt.Errorf("PM aggregation task version %s missing", childKey.TaskVersionID)
	}
	targets := rollupTargets(childKey.Granularity)
	if len(targets) == 0 {
		return nil
	}
	sourceExpected := childState.SourceExpectedSlots
	sourceReceived := childState.SourceReceivedSlots
	if sourceExpected == 0 {
		sourceExpected = childState.ExpectedSlots
		sourceReceived = childState.ReceivedSlots
	}
	childComplete := childState.ReceivedSlots >= childState.ExpectedSlots &&
		sourceReceived >= sourceExpected &&
		childState.SourceIncompleteSlots == 0
	values := compactCounterValues(version, childState.Accumulators)
	if len(values) == 0 {
		return nil
	}
	for _, target := range targets {
		if !containsGranularity(version.Granularities, target) {
			continue
		}
		parentWindow, err := WindowFor(childKey.Start, target, p.location)
		if err != nil {
			return err
		}
		if parentWindow.Start.Before(version.EffectiveFrom) ||
			(version.EffectiveTo != nil && !parentWindow.Start.Before(*version.EffectiveTo)) {
			continue
		}
		chunkCount := (len(values) + p.batch - 1) / p.batch
		for chunkIndex := 0; chunkIndex < chunkCount; chunkIndex++ {
			start := chunkIndex * p.batch
			end := start + p.batch
			if end > len(values) {
				end = len(values)
			}
			incomplete := int64(0)
			if !childComplete {
				incomplete = 1
			}
			contribution := Contribution{
				Key: WindowKey{
					TaskID: childKey.TaskID, TaskVersionID: childKey.TaskVersionID,
					Granularity: target, Start: parentWindow.Start, End: parentWindow.End,
				},
				SourceFileID: fmt.Sprintf(
					"%s:%s:%d:%d",
					childKey.TaskVersionID, childKey.Granularity,
					childKey.Start.UTC().Unix(), chunkIndex,
				),
				DeviceID:              childKey.TaskVersionID.String(),
				SlotStart:             childKey.Start.UTC(),
				ExpectedSlots:         expectedChildWindows(parentWindow, target, p.location),
				SourceExpectedSlots:   expectedSlots(parentWindow, len(version.Members)),
				SourceReceivedSlots:   sourceReceived,
				SourceIncompleteSlots: incomplete,
				Rollup:                true, RollupChunkIndex: chunkIndex, RollupChunkCount: chunkCount,
				Values: values[start:end],
			}
			if err := p.accumulate(ctx, contribution); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *DirectRollupPromoter) accumulate(ctx context.Context, contribution Contribution) error {
	published, err := p.windows.IsPublished(ctx, contribution.Key)
	if err != nil {
		return err
	}
	if published {
		p.logger.Info("ignore late PM aggregation direct rollup",
			zap.String("task_version_id", contribution.Key.TaskVersionID.String()),
			zap.Time("window_start", contribution.Key.Start))
		return nil
	}
	if err := p.windows.EnsureOpen(ctx, contribution); err != nil {
		return err
	}
	result, err := p.store.Accumulate(ctx, contribution)
	if err != nil {
		return err
	}
	if err := p.windows.ObserveReceived(ctx, contribution.Key, result.ReceivedSlots); err != nil {
		return err
	}
	if result.Complete {
		// Finalize outside the current child finalizer's concurrency slot.
		// The durable open-window row and Redis AOF state let the timeout
		// scanner recover this work if the process exits before it runs.
		go func(key WindowKey) {
			finalizeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if err := p.finalizer.Finalize(finalizeCtx, key, CloseComplete); err != nil {
				p.logger.Warn("finalize promoted PM aggregation window",
					zap.String("task_version_id", key.TaskVersionID.String()),
					zap.String("granularity", string(key.Granularity)),
					zap.Time("window_start", key.Start),
					zap.Error(err))
			}
		}(contribution.Key)
	}
	return nil
}

func rollupTargets(source Granularity) []Granularity {
	switch source {
	case GranularityHourly:
		return []Granularity{GranularityDaily}
	case GranularityDaily:
		return []Granularity{GranularityWeekly, GranularityMonthly}
	default:
		return nil
	}
}

func compactCounterValues(
	version *TaskVersionSnapshot,
	accumulators []Accumulator,
) []ContributionValue {
	values := make([]ContributionValue, 0, len(accumulators))
	for _, accumulator := range accumulators {
		rule, ok := version.Counters[accumulator.Definition.MetricPath]
		if !ok || accumulator.Count <= 0 {
			continue
		}
		value := accumulator.Definition
		value.MetricType = "counter"
		value.Operation = rule.Aggregation
		value.Value = 0
		value.Sum = accumulator.Sum
		value.Count = accumulator.Count
		value.Min = accumulator.Min
		value.Max = accumulator.Max
		value.Composed = true
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		leftKey := aggregationGroupKey(values[i]) + "\x1f" + values[i].MetricPath
		rightKey := aggregationGroupKey(values[j]) + "\x1f" + values[j].MetricPath
		return leftKey < rightKey
	})
	return values
}

func containsGranularity(values []Granularity, target Granularity) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
