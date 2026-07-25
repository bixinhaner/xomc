package stream

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTaskContentHashIgnoresCollectionOrder(t *testing.T) {
	firstDevice := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	secondDevice := uuid.MustParse("10000000-0000-4000-8000-000000000002")
	left := SaveTaskRequest{
		Name:          "内置-全网-LTE",
		Enabled:       true,
		Visibility:    "public",
		Creator:       "system",
		Technology:    "lte",
		Dimension:     DimensionNetwork,
		Granularities: []Granularity{GranularityDaily, GranularityHourly},
		ObjectLDNs:    []string{"Cell=2", "Cell=1"},
		Metrics: []MetricRule{
			{MetricID: "K2", MetricPath: "K2", MetricType: "kpi", Aggregation: AggregationAvg},
			{MetricID: "K1", MetricPath: "K1", MetricType: "counter", Aggregation: AggregationSum},
		},
		Members: []TaskMember{
			{DeviceID: secondDevice, DeviceSN: "SN-2", DimensionKey: "network", ObjectLDN: "Cell=2"},
			{DeviceID: firstDevice, DeviceSN: "SN-1", DimensionKey: "network", ObjectLDN: "Cell=1"},
		},
	}
	right := left
	right.Granularities = []Granularity{GranularityHourly, GranularityDaily}
	right.ObjectLDNs = []string{"Cell=1", "Cell=2"}
	right.Metrics = []MetricRule{left.Metrics[1], left.Metrics[0]}
	right.Members = []TaskMember{left.Members[1], left.Members[0]}

	require.Equal(t, taskContentHash(left), taskContentHash(right))
}

func TestTaskContentHashChangesWithExecutableContent(t *testing.T) {
	deviceID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	base := SaveTaskRequest{
		Name: "内置-全网-LTE", Enabled: true, Visibility: "public", Creator: "system",
		Technology: "lte", Dimension: DimensionNetwork,
		Granularities: []Granularity{GranularityHourly},
		Metrics: []MetricRule{
			{MetricID: "K1", MetricPath: "K1", MetricType: "counter", Aggregation: AggregationSum},
		},
		Members: []TaskMember{
			{DeviceID: deviceID, DeviceSN: "SN-1", DimensionKey: "network"},
		},
	}

	tests := map[string]func(*SaveTaskRequest){
		"enabled":       func(req *SaveTaskRequest) { req.Enabled = false },
		"technology":    func(req *SaveTaskRequest) { req.Technology = "nr" },
		"dimension":     func(req *SaveTaskRequest) { req.Dimension = DimensionProduct },
		"granularity":   func(req *SaveTaskRequest) { req.Granularities = []Granularity{GranularityDaily} },
		"metric rule":   func(req *SaveTaskRequest) { req.Metrics[0].Aggregation = AggregationAvg },
		"member device": func(req *SaveTaskRequest) { req.Members[0].DeviceID = uuid.New() },
		"dimension key": func(req *SaveTaskRequest) { req.Members[0].DimensionKey = "changed" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			changed := cloneSaveTaskRequest(base)
			mutate(&changed)
			require.NotEqual(t, taskContentHash(base), taskContentHash(changed))
		})
	}
}

func TestTaskContentHashExcludesDisplayAndSchedulingMetadata(t *testing.T) {
	base := SaveTaskRequest{
		Name: "name-a", Enabled: true, Visibility: "private", Creator: "alice",
		Technology: "lte", Dimension: DimensionNetwork,
		Granularities: []Granularity{GranularityHourly},
		Metrics: []MetricRule{
			{MetricID: "K1", MetricPath: "K1", MetricType: "counter", Aggregation: AggregationSum},
		},
		Members: []TaskMember{
			{DeviceID: uuid.MustParse("10000000-0000-4000-8000-000000000001"), DeviceSN: "SN-1", DimensionKey: "network"},
		},
	}
	changed := cloneSaveTaskRequest(base)
	changed.Name = "name-b"
	changed.Visibility = "public"
	changed.Creator = "bob"

	require.Equal(t, taskContentHash(base), taskContentHash(changed))
}

func cloneSaveTaskRequest(req SaveTaskRequest) SaveTaskRequest {
	cloned := req
	cloned.Granularities = append([]Granularity(nil), req.Granularities...)
	cloned.ObjectLDNs = append([]string(nil), req.ObjectLDNs...)
	cloned.Metrics = append([]MetricRule(nil), req.Metrics...)
	cloned.Members = append([]TaskMember(nil), req.Members...)
	return cloned
}
