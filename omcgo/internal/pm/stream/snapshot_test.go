package stream

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBuildTaskSnapshotSynthesizesDevicePipelineFromNetworkCatalog(t *testing.T) {
	deviceID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	networkVersion := &TaskVersionSnapshot{
		TaskID:    builtinNetworkRuleIDs["lte"],
		VersionID: uuid.MustParse("30000000-0000-4000-8000-000000000001"),
		VersionNo: 1, Name: "已重命名的系统规则", Enabled: true,
		Technology: "lte", Dimension: DimensionNetwork,
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily,
			GranularityWeekly, GranularityMonthly,
		},
		EffectiveFrom: time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC),
		Metrics: map[string]MetricRule{
			"K1": {MetricID: "K1", MetricPath: "K1", Aggregation: AggregationSum},
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{
				DeviceID: deviceID, DeviceSN: "SN-1",
				DimensionKey: "network", DimensionName: "Network",
			}},
		},
	}

	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{networkVersion})

	versions := snapshot.ByDevice[deviceID]
	require.Len(t, versions, 2)
	var deviceVersion *TaskVersionSnapshot
	for _, version := range versions {
		if version.DevicePipeline {
			deviceVersion = version
		}
	}
	require.NotNil(t, deviceVersion)
	require.Equal(t, DimensionDevice, deviceVersion.Dimension)
	require.NotEqual(t, networkVersion.TaskID, deviceVersion.TaskID)
	require.NotEqual(t, networkVersion.VersionID, deviceVersion.VersionID)
	require.Equal(t, deviceID.String(), deviceVersion.Members[deviceID][0].DimensionKey)
	require.Equal(t, networkVersion.Counters, deviceVersion.Counters)
	require.Same(t, deviceVersion, snapshot.ByVersion[deviceVersion.VersionID])
}

func TestBuildTaskSnapshotDoesNotSynthesizeDevicePipelineForCustomNetworkRule(t *testing.T) {
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Name: "我的全网分析",
		Enabled: true, Technology: "lte", Dimension: DimensionNetwork,
		Members: map[uuid.UUID][]TaskMember{uuid.New(): {{DeviceID: uuid.New()}}},
	}

	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{version})

	require.Len(t, snapshot.ByVersion, 1)
}

func TestDevicePipelineUsesImmutableHourlyVersionAndStableRollupLineage(t *testing.T) {
	first := &TaskVersionSnapshot{
		TaskID: builtinNetworkRuleIDs["lte"], VersionID: uuid.New(),
		Technology: "lte", Dimension: DimensionNetwork,
	}
	second := *first
	second.VersionID = uuid.New()

	firstPipeline, ok := devicePipelineVersion(first)
	require.True(t, ok)
	secondPipeline, ok := devicePipelineVersion(&second)
	require.True(t, ok)

	require.NotEqual(t, firstPipeline.VersionID, secondPipeline.VersionID)
	require.Equal(t, firstPipeline.RollupVersionID, secondPipeline.RollupVersionID)
	require.NotEqual(t, uuid.Nil, firstPipeline.RollupVersionID)
}

func TestBuildTaskSnapshotKeepsHourlyDefinitionsImmutable(t *testing.T) {
	deviceID := uuid.New()
	first := &TaskVersionSnapshot{
		TaskID: builtinNetworkRuleIDs["lte"], VersionID: uuid.New(), VersionNo: 1,
		Enabled: true, Technology: "lte", Dimension: DimensionNetwork,
		EffectiveFrom: time.Now().UTC().Add(-2 * time.Hour),
		Counters: map[string]CounterRule{
			"OLD": {MetricPath: "OLD", Aggregation: AggregationSum},
		},
		Members: map[uuid.UUID][]TaskMember{
			deviceID: {{DeviceID: deviceID, DimensionKey: "network"}},
		},
	}
	end := time.Now().UTC().Add(-time.Hour)
	first.EffectiveTo = &end
	second := &TaskVersionSnapshot{
		TaskID: first.TaskID, VersionID: uuid.New(), VersionNo: 2,
		Enabled: true, Technology: "lte", Dimension: DimensionNetwork,
		EffectiveFrom: end,
		Counters: map[string]CounterRule{
			"NEW": {MetricPath: "NEW", Aggregation: AggregationSum},
		},
		Members: first.Members,
	}

	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{first, second})
	firstPipeline, ok := devicePipelineVersion(first)
	require.True(t, ok)
	secondPipeline, ok := devicePipelineVersion(second)
	require.True(t, ok)

	require.Equal(t, first.Counters, snapshot.ByVersion[firstPipeline.VersionID].Counters)
	require.Equal(t, second.Counters, snapshot.ByVersion[secondPipeline.VersionID].Counters)
	rollup := snapshot.ByVersion[secondPipeline.RollupVersionID]
	require.NotNil(t, rollup)
	require.True(t, rollup.DeviceRollup)
	require.Contains(t, rollup.Counters, "OLD")
	require.Contains(t, rollup.Counters, "NEW")
	require.Equal(t, second.Metrics, rollup.Metrics)
}
