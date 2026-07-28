package stream

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

var builtinNetworkRuleIDs = map[string]uuid.UUID{
	"lte": uuid.MustParse("0184dddd-0001-4000-8000-000000000001"),
	"nr":  uuid.MustParse("0184dddd-0001-4000-8000-000000000002"),
	"gsm": uuid.MustParse("0184dddd-0001-4000-8000-000000000003"),
}

func devicePipelineVersion(source *TaskVersionSnapshot) (*TaskVersionSnapshot, bool) {
	if source == nil || source.Dimension != DimensionNetwork || source.Technology == "" {
		return nil, false
	}
	systemRuleID, ok := builtinNetworkRuleIDs[strings.ToLower(source.Technology)]
	if !ok || source.TaskID != systemRuleID {
		return nil, false
	}
	taskID := devicePipelineTaskID(source.Technology)
	// Raw hourly definitions are immutable and follow their catalog source
	// version. Their finalized counters enter a separate stable rollup lineage.
	versionID := uuid.NewSHA1(taskID, []byte("source:"+source.VersionID.String()))
	rollupVersionID := deviceRollupVersionID(source.Technology)
	members := make(map[uuid.UUID][]TaskMember, len(source.Members))
	for deviceID, sourceMembers := range source.Members {
		deviceMembers := make([]TaskMember, 0, len(sourceMembers))
		for _, member := range sourceMembers {
			name := member.DeviceSN
			if name == "" {
				name = deviceID.String()
			}
			deviceMembers = append(deviceMembers, TaskMember{
				DeviceID: deviceID, DeviceSN: member.DeviceSN,
				DimensionKey: deviceID.String(), DimensionName: name,
				ObjectLDN: member.ObjectLDN,
			})
		}
		members[deviceID] = deviceMembers
	}
	return &TaskVersionSnapshot{
		TaskID: taskID, VersionID: versionID, VersionNo: source.VersionNo,
		Name:    "设备基础流水线-" + strings.ToUpper(source.Technology),
		Enabled: true, Technology: source.Technology, Dimension: DimensionDevice,
		Granularities: append([]Granularity(nil), source.Granularities...),
		ObjectLDNs:    source.ObjectLDNs, EffectiveFrom: source.EffectiveFrom,
		EffectiveTo: source.EffectiveTo, Metrics: source.Metrics,
		Counters: source.Counters, Members: members, DevicePipeline: true,
		RollupVersionID: rollupVersionID,
	}, true
}

func deviceRollupVersion(source *TaskVersionSnapshot) (*TaskVersionSnapshot, bool) {
	pipeline, ok := devicePipelineVersion(source)
	if !ok {
		return nil, false
	}
	pipeline.VersionID = pipeline.RollupVersionID
	pipeline.VersionNo = 1
	pipeline.Name = "设备逐级汇聚-" + strings.ToUpper(source.Technology)
	pipeline.EffectiveFrom = time.Time{}
	pipeline.EffectiveTo = nil
	pipeline.DevicePipeline = false
	pipeline.DeviceRollup = true
	return pipeline, true
}

func devicePipelineTaskID(technology string) uuid.UUID {
	return uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte("omcgo.pm.device-pipeline."+strings.ToLower(technology)),
	)
}

func deviceRollupVersionID(technology string) uuid.UUID {
	return uuid.NewSHA1(
		devicePipelineTaskID(technology),
		[]byte("fixed-device-rollup-v1"),
	)
}
