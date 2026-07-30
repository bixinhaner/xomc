package stream

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

func isDeviceHourPayload(payload RollupPayload, snapshot *TaskSnapshot) bool {
	if snapshot == nil || payload.SourceGranularity != GranularityHourly {
		return false
	}
	version := snapshot.ByVersion[payload.TaskVersionID]
	return version != nil && version.DeviceRollup
}

// matchDeviceHourRules converts one finalized device-hour Counter event into
// compact contributions for every effective rule membership. It never forwards
// device KPI values and never reads historical PM rows.
func matchDeviceHourRules(
	payload RollupPayload,
	snapshot *TaskSnapshot,
	location *time.Location,
) ([]Contribution, error) {
	if payload.SourceGranularity != GranularityHourly || payload.EntityKey == "" ||
		snapshot == nil {
		return nil, nil
	}
	deviceID, err := uuid.Parse(payload.EntityKey)
	if err != nil {
		return nil, fmt.Errorf("parse device-hour entity key: %w", err)
	}
	var out []Contribution
	for _, version := range snapshot.ByDevice[deviceID] {
		out = append(out, matchDeviceHourVersion(
			payload, version, deviceID, nil,
		)...)
	}
	return out, nil
}

func matchDeviceHourRuleWindow(
	payload RollupPayload,
	snapshot *TaskSnapshot,
	key WindowKey,
) ([]Contribution, error) {
	if payload.SourceGranularity != GranularityHourly || payload.EntityKey == "" ||
		snapshot == nil || key.Granularity != GranularityHourly {
		return nil, nil
	}
	deviceID, err := uuid.Parse(payload.EntityKey)
	if err != nil {
		return nil, fmt.Errorf("parse device-hour entity key: %w", err)
	}
	version := snapshot.ByVersion[key.TaskVersionID]
	if version == nil {
		return nil, nil
	}
	return matchDeviceHourVersion(payload, version, deviceID, &key.EntityKey), nil
}

func matchDeviceHourVersion(
	payload RollupPayload,
	version *TaskVersionSnapshot,
	deviceID uuid.UUID,
	targetDimensionKey *string,
) []Contribution {
	if version == nil || version.DevicePipeline || version.DeviceRollup ||
		!version.Enabled ||
		!containsGranularity(version.Granularities, GranularityHourly) ||
		(version.Technology != "" && len(payload.Values) > 0 &&
			!strings.EqualFold(version.Technology, payload.Values[0].Technology)) {
		return nil
	}
	if payload.WindowStart.Before(version.EffectiveFrom) ||
		(version.EffectiveTo != nil && !payload.WindowStart.Before(*version.EffectiveTo)) {
		return nil
	}
	membersByDimension := make(map[string][]TaskMember)
	for _, member := range version.Members[deviceID] {
		if targetDimensionKey != nil && member.DimensionKey != *targetDimensionKey {
			continue
		}
		membersByDimension[member.DimensionKey] = append(
			membersByDimension[member.DimensionKey], member,
		)
	}
	dimensionKeys := make([]string, 0, len(membersByDimension))
	for dimensionKey := range membersByDimension {
		dimensionKeys = append(dimensionKeys, dimensionKey)
	}
	sort.Strings(dimensionKeys)
	out := make([]Contribution, 0, len(dimensionKeys))
	for _, dimensionKey := range dimensionKeys {
		members := membersByDimension[dimensionKey]
		representative := members[0]
		values := make([]ContributionValue, 0, len(payload.Values))
		for _, source := range payload.Values {
			rule, ok := version.Counters[source.MetricPath]
			if !ok || !matchesAnyRuleMember(source.ObjectLDN, members) {
				continue
			}
			value := source
			value.Dimension = version.Dimension
			value.DimensionKey = representative.DimensionKey
			value.DimensionName = representative.DimensionName
			if objectLDN, ok := rollupObjectLDN(version.Dimension, representative); ok {
				value.ObjectLDN = objectLDN
			}
			value.DeviceOUI = ""
			value.DeviceSN = ""
			value.Operation = rule.Aggregation
			values = append(values, value)
		}
		// Preserve an empty contribution when this source chunk contains no
		// selected Counter. The source device event was already chunked; every
		// target rule must observe every chunk before counting the device hour.
		incomplete := payload.SourceIncompleteSlots
		if !payload.Complete && incomplete == 0 {
			incomplete = 1
		}
		out = append(out, Contribution{
			Key: WindowKey{
				TaskID: version.TaskID, TaskVersionID: version.VersionID,
				EntityKey: dimensionKey, Granularity: GranularityHourly,
				Start: payload.WindowStart, End: payload.WindowEnd,
			},
			SourceFileID: fmt.Sprintf(
				"%s:%s:%s",
				payload.EventID, version.VersionID, dimensionKey,
			),
			DeviceID: deviceID.String(), SlotStart: payload.WindowStart,
			ExpectedSlots:         ruleDimensionMemberCount(version, dimensionKey),
			SourceExpectedSlots:   payload.SourceExpectedSlots,
			SourceReceivedSlots:   payload.SourceReceivedSlots,
			SourceIncompleteSlots: incomplete,
			VersionEffectiveFrom:  version.EffectiveFrom,
			VersionEffectiveTo:    version.EffectiveTo,
			Rollup:                true, RollupChunkIndex: payload.ChunkIndex,
			RollupChunkCount: payload.ChunkCount, Values: values,
		})
	}
	return out
}

func rollupObjectLDN(dimension Dimension, representative TaskMember) (string, bool) {
	switch dimension {
	case DimensionNetwork, DimensionProduct, DimensionDeviceGroup, DimensionAggregateGroup:
		return "", true
	case DimensionBand:
		return representative.DimensionKey, true
	default:
		return "", false
	}
}

func matchesAnyRuleMember(objectLDN string, members []TaskMember) bool {
	for _, member := range members {
		if member.ObjectLDN == "" || member.ObjectLDN == objectLDN ||
			strings.Contains(objectLDN, "Cellid="+member.ObjectLDN) {
			return true
		}
	}
	return false
}

func ruleDimensionMemberCount(version *TaskVersionSnapshot, dimensionKey string) int64 {
	if version != nil && version.DimensionMemberCounts != nil {
		return version.DimensionMemberCounts[dimensionKey]
	}
	devices := make(map[uuid.UUID]struct{})
	for deviceID, members := range version.Members {
		for _, member := range members {
			if member.DimensionKey == dimensionKey {
				devices[deviceID] = struct{}{}
				break
			}
		}
	}
	return int64(len(devices))
}
