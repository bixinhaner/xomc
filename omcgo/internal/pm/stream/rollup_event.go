package stream

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// RollupPayload is the durable compact Counter state emitted by a finalized
// hourly or daily window. KPI values and original 15-minute samples are never
// forwarded to parent windows.
type RollupPayload struct {
	SchemaVersion         int                 `json:"sv"`
	EventID               uuid.UUID           `json:"eid"`
	TaskID                uuid.UUID           `json:"rid"`
	TaskVersionID         uuid.UUID           `json:"rv"`
	SourceGranularity     Granularity         `json:"sg"`
	EntityKey             string              `json:"e,omitempty"`
	WindowStart           time.Time           `json:"ws"`
	WindowEnd             time.Time           `json:"we"`
	SourceExpectedSlots   int64               `json:"se"`
	SourceReceivedSlots   int64               `json:"sr"`
	SourceIncompleteSlots int64               `json:"si,omitempty"`
	Complete              bool                `json:"ok"`
	ChunkIndex            int                 `json:"ci"`
	ChunkCount            int                 `json:"cc"`
	Values                []ContributionValue `json:"v"`
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
	complete := state.ReceivedSlots >= state.ExpectedSlots &&
		sourceReceived >= sourceExpected && state.SourceIncompleteSlots == 0
	grouped := map[string][]ContributionValue{key.EntityKey: values}
	entityKeys := make([]string, 0, len(grouped))
	for entityKey := range grouped {
		entityKeys = append(entityKeys, entityKey)
	}
	sort.Strings(entityKeys)
	out := make([]RollupPayload, 0)
	rollupVersionID := key.TaskVersionID
	if version.DevicePipeline && version.RollupVersionID != uuid.Nil {
		rollupVersionID = version.RollupVersionID
	}
	for _, entityKey := range entityKeys {
		entityValues := grouped[entityKey]
		chunks, err := chunkRollupValues(entityValues, batch, maxRollupEventBytes-2048)
		if err != nil {
			return nil, err
		}
		chunkCount := len(chunks)
		entityExpected, entityReceived, entityIncomplete := sourceExpected, sourceReceived, state.SourceIncompleteSlots
		payloadComplete := complete
		if version.DevicePipeline {
			entityExpected = 4
			if key.Granularity == GranularityDaily {
				entityExpected = 24
			}
			if entity, ok := state.Entities[entityKey]; ok {
				entityReceived = entity.SourceReceivedSlots
				entityIncomplete = entity.SourceIncompleteSlots
				payloadComplete = entity.ReceivedSlots >= entityExpected &&
					entity.SourceReceivedSlots >= entity.SourceExpectedSlots &&
					entity.SourceIncompleteSlots == 0
				entityExpected = entity.SourceExpectedSlots
			} else {
				entityReceived = entityExpected
				entityIncomplete = 0
				if !complete {
					entityReceived = 0
					entityIncomplete = 1
				}
			}
		}
		for chunkIndex := 0; chunkIndex < chunkCount; chunkIndex++ {
			name := fmt.Sprintf(
				"%s:%s:%d:%s:%d",
				rollupVersionID, key.Granularity, key.Start.UTC().Unix(), entityKey, chunkIndex,
			)
			out = append(out, RollupPayload{
				SchemaVersion: SchemaVersion, EventID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)),
				TaskID: key.TaskID, TaskVersionID: rollupVersionID,
				SourceGranularity: key.Granularity, EntityKey: entityKey,
				WindowStart: key.Start, WindowEnd: key.End,
				SourceExpectedSlots: entityExpected, SourceReceivedSlots: entityReceived,
				SourceIncompleteSlots: entityIncomplete, Complete: payloadComplete,
				ChunkIndex: chunkIndex, ChunkCount: chunkCount,
				Values: chunks[chunkIndex],
			})
		}
	}
	return out, nil
}

func chunkRollupValues(
	values []ContributionValue,
	maxValues int,
	maxBytes int,
) ([][]ContributionValue, error) {
	if maxValues <= 0 {
		maxValues = defaultRollupBatchValues
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("PM compact rollup byte limit is too small")
	}
	var chunks [][]ContributionValue
	current := make([]ContributionValue, 0, min(maxValues, len(values)))
	currentBytes := 2
	for _, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal PM compact Counter state: %w", err)
		}
		valueBytes := len(encoded) + 1
		if valueBytes+2 > maxBytes {
			return nil, fmt.Errorf(
				"one PM compact Counter state is %d bytes, exceeds payload budget %d",
				valueBytes, maxBytes,
			)
		}
		if len(current) > 0 &&
			(len(current) >= maxValues || currentBytes+valueBytes > maxBytes) {
			chunks = append(chunks, current)
			current = make([]ContributionValue, 0, min(maxValues, len(values)))
			currentBytes = 2
		}
		current = append(current, value)
		currentBytes += valueBytes
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks, nil
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
		incomplete := payload.SourceIncompleteSlots
		if !payload.Complete && incomplete == 0 {
			incomplete++
		}
		expected := expectedChildWindows(parent, target, location)
		if !version.DevicePipeline && !version.DeviceRollup {
			expected = expectedVersionChildWindows(
				parent, payload.SourceGranularity, version, location,
			)
			if expected == 0 {
				continue
			}
		}
		deviceID := payload.TaskVersionID.String()
		if payload.EntityKey != "" {
			deviceID = payload.EntityKey
		}
		out = append(out, Contribution{
			Key: WindowKey{
				TaskID: payload.TaskID, TaskVersionID: payload.TaskVersionID,
				EntityKey:   payload.EntityKey,
				Granularity: target, Start: parent.Start, End: parent.End,
			},
			SourceFileID:          payload.EventID.String(),
			DeviceID:              deviceID,
			SlotStart:             payload.WindowStart.UTC(),
			ExpectedSlots:         expected,
			SourceExpectedSlots:   payload.SourceExpectedSlots,
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

func expectedVersionChildWindows(
	parent Window,
	source Granularity,
	version *TaskVersionSnapshot,
	location *time.Location,
) int64 {
	if version == nil {
		return 0
	}
	var count int64
	for cursor := parent.Start; cursor.Before(parent.End); {
		var childEnd time.Time
		switch source {
		case GranularityHourly:
			childEnd = cursor.Add(time.Hour)
		case GranularityDaily:
			local := cursor.In(location)
			childEnd = local.AddDate(0, 0, 1).UTC()
		default:
			return 0
		}
		overlaps := childEnd.After(version.EffectiveFrom) &&
			(version.EffectiveTo == nil || cursor.Before(*version.EffectiveTo))
		if overlaps {
			count++
		}
		cursor = childEnd
	}
	return count
}
