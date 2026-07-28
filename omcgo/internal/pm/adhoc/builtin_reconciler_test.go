package adhoc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"github.com/stretchr/testify/require"
)

func TestBuiltinReconcilerSavesEmptyDefinitionForLaterMembershipRefresh(t *testing.T) {
	task := builtinTaskForTest()
	saveCalls := 0
	reconciler := &BuiltinReconciler{
		list: func(context.Context) ([]Task, error) {
			return []Task{task}, nil
		},
		resolveRules: func(context.Context, string, []string) ([]pmstream.MetricRule, error) {
			return builtinRulesForTest(), nil
		},
		resolveCounters: func(context.Context, string, []pmstream.MetricRule) ([]pmstream.CounterRule, error) {
			return builtinCountersForTest(), nil
		},
		resolveMembers: func(context.Context, *Task) ([]pmstream.TaskMember, error) {
			return nil, nil
		},
		save: func(context.Context, pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error) {
			saveCalls++
			return nil, nil
		},
	}

	result, err := reconciler.Reconcile(context.Background())

	require.NoError(t, err)
	require.Equal(t, BuiltinReconcileResult{Definitions: 2, Saved: 2, Empty: 2}, result)
	require.Equal(t, 2, saveCalls)
}

func TestBuiltinReconcilerSavesEditedDefinitionAndHiddenDeviceDefinition(t *testing.T) {
	task := builtinTaskForTest()
	task.MetricPaths = []string{"K-EDITED"}
	networkMembers := []pmstream.TaskMember{{
		DeviceID: uuid.MustParse("10000000-0000-4000-8000-000000000001"),
		DeviceSN: "SN-1", DimensionKey: "network", DimensionName: "Network",
	}}
	deviceMembers := []pmstream.TaskMember{{
		DeviceID: uuid.MustParse("10000000-0000-4000-8000-000000000001"),
		DeviceSN: "SN-1", DimensionKey: "10000000-0000-4000-8000-000000000001", DimensionName: "SN-1",
	}}
	now := time.Date(2026, 7, 27, 6, 4, 30, 0, time.UTC)
	var captured []pmstream.SaveTaskRequest
	reconciler := &BuiltinReconciler{
		now: func() time.Time { return now },
		list: func(context.Context) ([]Task, error) {
			return []Task{task}, nil
		},
		resolveEnabledMetricPaths: func(_ context.Context, technology string) ([]string, error) {
			require.Equal(t, "lte", technology)
			return []string{"C000000005", "K-EDITED", ""}, nil
		},
		resolveRules: func(_ context.Context, technology string, paths []string) ([]pmstream.MetricRule, error) {
			require.Equal(t, "lte", technology)
			require.Equal(t, []string{"K-EDITED", "C000000005"}, paths)
			return metricRulesForPaths(paths), nil
		},
		resolveCounters: func(context.Context, string, []pmstream.MetricRule) ([]pmstream.CounterRule, error) {
			return builtinCountersForTest(), nil
		},
		resolveMembers: func(_ context.Context, task *Task) ([]pmstream.TaskMember, error) {
			if task.Dimension == DimensionDevice {
				return deviceMembers, nil
			}
			return networkMembers, nil
		},
		save: func(_ context.Context, req pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error) {
			captured = append(captured, req)
			return &pmstream.TaskVersionSnapshot{NewVersion: true}, nil
		},
	}

	result, err := reconciler.Reconcile(context.Background())

	require.NoError(t, err)
	require.Equal(t, 2, result.Definitions)
	require.Equal(t, 2, result.Saved)
	require.Equal(t, 2, result.Changed)
	require.Len(t, captured, 2)
	require.Equal(t, task.ID, captured[0].TaskID)
	require.Equal(t, task.Name, captured[0].Name)
	require.Equal(t, "lte", captured[0].Technology)
	require.Equal(t, pmstream.DimensionNetwork, captured[0].Dimension)
	require.Equal(t, []pmstream.Granularity{
		pmstream.GranularityHourly, pmstream.GranularityDaily,
		pmstream.GranularityWeekly, pmstream.GranularityMonthly,
	}, captured[0].Granularities)
	require.Equal(t, networkMembers, captured[0].Members)
	require.Equal(t, metricRulesForPaths([]string{"K-EDITED", "C000000005"}), captured[0].Metrics)
	require.Equal(t, builtinCountersForTest(), captured[0].Counters)

	require.Equal(t, hiddenDeviceBuiltinTaskIDs["lte"], captured[1].TaskID)
	require.Equal(t, "内置-设备-LTE", captured[1].Name)
	require.Equal(t, "lte", captured[1].Technology)
	require.Equal(t, pmstream.DimensionDevice, captured[1].Dimension)
	require.Equal(t, string(VisibilityPrivate), captured[1].Visibility)
	require.True(t, captured[1].EffectiveFrom.Equal(time.Date(2026, 7, 27, 6, 0, 0, 0, time.UTC)))
	require.Equal(t, metricRulesForPaths([]string{"K-EDITED", "C000000005"}), captured[1].Metrics)
	require.Equal(t, builtinCountersForTest(), captured[1].Counters)
	require.Equal(t, deviceMembers, captured[1].Members)
}

func TestBuiltinReconcilerContinuesAfterOneDefinitionFails(t *testing.T) {
	first := builtinTaskForTest()
	second := builtinTaskForTest()
	second.ID = uuid.MustParse("0184dddd-0001-4000-8000-000000000002")
	second.Name = "内置-全网-NR"
	second.Technology = "nr"
	saveCalls := 0
	reconciler := &BuiltinReconciler{
		list: func(context.Context) ([]Task, error) {
			return []Task{first, second}, nil
		},
		resolveRules: func(_ context.Context, _ string, paths []string) ([]pmstream.MetricRule, error) {
			if saveCalls == 0 {
				saveCalls++
				return nil, errors.New("metadata unavailable")
			}
			return builtinRulesForTest(), nil
		},
		resolveCounters: func(context.Context, string, []pmstream.MetricRule) ([]pmstream.CounterRule, error) {
			return builtinCountersForTest(), nil
		},
		resolveMembers: func(context.Context, *Task) ([]pmstream.TaskMember, error) {
			return []pmstream.TaskMember{{
				DeviceID: uuid.New(), DeviceSN: "SN", DimensionKey: "network",
			}}, nil
		},
		save: func(context.Context, pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error) {
			saveCalls++
			return &pmstream.TaskVersionSnapshot{NewVersion: false}, nil
		},
	}

	result, err := reconciler.Reconcile(context.Background())

	require.ErrorContains(t, err, first.ID.String())
	require.Equal(t, 3, result.Definitions)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, 2, result.Saved)
	require.Zero(t, result.Changed)
}

func TestBuiltinReconcilerRejectsNonBuiltinOrNonContinuousDefinitions(t *testing.T) {
	custom := builtinTaskForTest()
	custom.IsBuiltin = false
	oneshot := builtinTaskForTest()
	oneshot.Mode = ModeOneshot
	reconciler := &BuiltinReconciler{
		list: func(context.Context) ([]Task, error) {
			return []Task{custom, oneshot}, nil
		},
		resolveRules: func(context.Context, string, []string) ([]pmstream.MetricRule, error) {
			t.Fatal("ineligible definitions must not be resolved")
			return nil, nil
		},
		resolveCounters: func(context.Context, string, []pmstream.MetricRule) ([]pmstream.CounterRule, error) {
			t.Fatal("ineligible definitions must not be resolved")
			return nil, nil
		},
		resolveMembers: func(context.Context, *Task) ([]pmstream.TaskMember, error) {
			t.Fatal("ineligible definitions must not be resolved")
			return nil, nil
		},
		save: func(context.Context, pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error) {
			t.Fatal("ineligible definitions must not be saved")
			return nil, nil
		},
	}

	result, err := reconciler.Reconcile(context.Background())

	require.NoError(t, err)
	require.Zero(t, result.Definitions)
}

func builtinTaskForTest() Task {
	return Task{
		ID:            uuid.MustParse("0184dddd-0001-4000-8000-000000000001"),
		Name:          "内置-全网-LTE",
		Mode:          ModeContinuous,
		MetricPaths:   []string{"K1"},
		Granularities: []string{"hourly"},
		Dimension:     DimensionNetwork,
		Technology:    "lte",
		IsBuiltin:     true,
		Creator:       "system",
		Visibility:    VisibilityPublic,
	}
}

func builtinRulesForTest() []pmstream.MetricRule {
	return metricRulesForPaths([]string{"K1"})
}

func metricRulesForPaths(paths []string) []pmstream.MetricRule {
	rules := make([]pmstream.MetricRule, 0, len(paths))
	for _, path := range paths {
		rules = append(rules, pmstream.MetricRule{
			MetricID: path, MetricPath: path, MetricType: "counter",
			Aggregation: pmstream.AggregationSum,
		})
	}
	return rules
}

func builtinCountersForTest() []pmstream.CounterRule {
	return []pmstream.CounterRule{{
		MetricPath: "K1", Aggregation: pmstream.AggregationSum,
	}}
}
