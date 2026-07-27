package stream

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RollupPayload is the durable compact Counter state emitted by a finalized
// hourly or daily window. KPI values and original 15-minute samples are never
// forwarded to parent windows.
type RollupPayload struct {
	SchemaVersion         int                 `json:"schema_version"`
	EventID               uuid.UUID           `json:"event_id"`
	TaskID                uuid.UUID           `json:"task_id"`
	TaskVersionID         uuid.UUID           `json:"task_version_id"`
	SourceGranularity     Granularity         `json:"source_granularity"`
	WindowStart           time.Time           `json:"window_start"`
	WindowEnd             time.Time           `json:"window_end"`
	SourceExpectedSlots   int64               `json:"source_expected_slots"`
	SourceReceivedSlots   int64               `json:"source_received_slots"`
	SourceIncompleteSlots int64               `json:"source_incomplete_slots"`
	Complete              bool                `json:"complete"`
	ChunkIndex            int                 `json:"chunk_index"`
	ChunkCount            int                 `json:"chunk_count"`
	Values                []ContributionValue `json:"values"`
}

func (p RollupPayload) Validate() error {
	if p.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: unsupported rollup schema version %d", ErrInvalidEvent, p.SchemaVersion)
	}
	if p.EventID == uuid.Nil || p.TaskID == uuid.Nil || p.TaskVersionID == uuid.Nil {
		return fmt.Errorf("%w: rollup UUID is empty", ErrInvalidEvent)
	}
	if p.SourceGranularity != GranularityHourly && p.SourceGranularity != GranularityDaily {
		return fmt.Errorf("%w: unsupported rollup source %q", ErrInvalidEvent, p.SourceGranularity)
	}
	if p.WindowStart.IsZero() || !p.WindowEnd.After(p.WindowStart) ||
		p.ChunkIndex < 0 || p.ChunkCount <= 0 || p.ChunkIndex >= p.ChunkCount ||
		len(p.Values) == 0 {
		return fmt.Errorf("%w: invalid compact rollup window or chunk", ErrInvalidEvent)
	}
	for _, value := range p.Values {
		if value.MetricPath == "" || value.Count <= 0 || !value.Composed {
			return fmt.Errorf("%w: invalid compact Counter state", ErrInvalidEvent)
		}
	}
	return nil
}

func buildRollupPayloads(
	key WindowKey,
	reason CloseReason,
	state WindowState,
	version *TaskVersionSnapshot,
	batch int,
) ([]RollupPayload, error) {
	if key.Granularity != GranularityHourly && key.Granularity != GranularityDaily {
		return nil, nil
	}
	if version == nil {
		return nil, fmt.Errorf("PM aggregation task version snapshot missing")
	}
	if batch <= 0 {
		batch = defaultRollupBatchValues
	}
	values := compactCounterValues(version, state.Accumulators)
	if len(values) == 0 {
		return nil, nil
	}
	sourceExpected := state.SourceExpectedSlots
	sourceReceived := state.SourceReceivedSlots
	if sourceExpected == 0 {
		sourceExpected = state.ExpectedSlots
		sourceReceived = state.ReceivedSlots
	}
	complete := reason == CloseComplete && state.ReceivedSlots >= state.ExpectedSlots &&
		sourceReceived >= sourceExpected && state.SourceIncompleteSlots == 0
	chunkCount := (len(values) + batch - 1) / batch
	out := make([]RollupPayload, 0, chunkCount)
	for chunkIndex := 0; chunkIndex < chunkCount; chunkIndex++ {
		start := chunkIndex * batch
		end := start + batch
		if end > len(values) {
			end = len(values)
		}
		name := fmt.Sprintf(
			"%s:%s:%d:%d",
			key.TaskVersionID, key.Granularity, key.Start.UTC().Unix(), chunkIndex,
		)
		out = append(out, RollupPayload{
			SchemaVersion:         SchemaVersion,
			EventID:               uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)),
			TaskID:                key.TaskID,
			TaskVersionID:         key.TaskVersionID,
			SourceGranularity:     key.Granularity,
			WindowStart:           key.Start,
			WindowEnd:             key.End,
			SourceExpectedSlots:   sourceExpected,
			SourceReceivedSlots:   sourceReceived,
			SourceIncompleteSlots: state.SourceIncompleteSlots,
			Complete:              complete,
			ChunkIndex:            chunkIndex,
			ChunkCount:            chunkCount,
			Values:                values[start:end],
		})
	}
	return out, nil
}

func rollupContributions(
	payload RollupPayload,
	version *TaskVersionSnapshot,
	location *time.Location,
) ([]Contribution, error) {
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	if version == nil || !version.Enabled {
		return nil, nil
	}
	targets := rollupTargets(payload.SourceGranularity)
	out := make([]Contribution, 0, len(targets))
	for _, target := range targets {
		if !containsGranularity(version.Granularities, target) {
			continue
		}
		parent, err := WindowFor(payload.WindowStart, target, location)
		if err != nil {
			return nil, err
		}
		if parent.Start.Before(version.EffectiveFrom) ||
			(version.EffectiveTo != nil && !parent.Start.Before(*version.EffectiveTo)) {
			continue
		}
		incomplete := payload.SourceIncompleteSlots
		if !payload.Complete && incomplete == 0 {
			incomplete++
		}
		out = append(out, Contribution{
			Key: WindowKey{
				TaskID: payload.TaskID, TaskVersionID: payload.TaskVersionID,
				Granularity: target, Start: parent.Start, End: parent.End,
			},
			SourceFileID:          payload.EventID.String(),
			DeviceID:              payload.TaskVersionID.String(),
			SlotStart:             payload.WindowStart.UTC(),
			ExpectedSlots:         expectedChildWindows(parent, target, location),
			SourceExpectedSlots:   expectedSlots(parent, len(version.Members)),
			SourceReceivedSlots:   payload.SourceReceivedSlots,
			SourceIncompleteSlots: incomplete,
			Rollup:                true,
			RollupChunkIndex:      payload.ChunkIndex,
			RollupChunkCount:      payload.ChunkCount,
			Values:                payload.Values,
		})
	}
	return out, nil
}
