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
		if version.DevicePipeline || !version.Enabled ||
			!containsGranularity(version.Granularities, GranularityHourly) ||
			(version.Technology != "" && len(payload.Values) > 0 &&
				!strings.EqualFold(version.Technology, payload.Values[0].Technology)) {
			continue
		}
		if payload.WindowStart.Before(version.EffectiveFrom) ||
			(version.EffectiveTo != nil && !payload.WindowStart.Before(*version.EffectiveTo)) {
			continue
		}
		membersByDimension := make(map[string][]TaskMember)
		for _, member := range version.Members[deviceID] {
			membersByDimension[member.DimensionKey] = append(
				membersByDimension[member.DimensionKey], member,
			)
		}
		dimensionKeys := make([]string, 0, len(membersByDimension))
		for dimensionKey := range membersByDimension {
			dimensionKeys = append(dimensionKeys, dimensionKey)
		}
		sort.Strings(dimensionKeys)
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
				value.DeviceOUI = ""
				value.DeviceSN = ""
				value.Operation = rule.Aggregation
				values = append(values, value)
			}
			// Preserve an empty contribution when this source chunk contains no
			// selected Counter. The source device event was already chunked;
			// every target rule must observe every chunk index before counting
			// the device-hour child as complete.
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
				Rollup:                true, RollupChunkIndex: payload.ChunkIndex,
				RollupChunkCount: payload.ChunkCount, Values: values,
			})
		}
	}
	return out, nil
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
