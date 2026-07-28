package adhoc

import (
	"context"
	"errors"
	"testing"

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
	require.Equal(t, BuiltinReconcileResult{Definitions: 1, Saved: 1, Empty: 1}, result)
	require.Equal(t, 1, saveCalls)
}

func TestBuiltinReconcilerSavesEditedRuleWithoutHiddenDeviceTask(t *testing.T) {
	task := builtinTaskForTest()
	task.MetricPaths = []string{"K-EDITED"}
	networkMembers := []pmstream.TaskMember{{
		DeviceID: uuid.MustParse("10000000-0000-4000-8000-000000000001"),
		DeviceSN: "SN-1", DimensionKey: "network", DimensionName: "Network",
	}}
	var captured []pmstream.SaveTaskRequest
	reconciler := &BuiltinReconciler{
		list: func(context.Context) ([]Task, error) {
			return []Task{task}, nil
		},
		resolveRules: func(_ context.Context, technology string, paths []string) ([]pmstream.MetricRule, error) {
			require.Equal(t, "lte", technology)
			require.Equal(t, []string{"K-EDITED"}, paths)
			return builtinRulesForTest(), nil
		},
		resolveCounters: func(context.Context, string, []pmstream.MetricRule) ([]pmstream.CounterRule, error) {
			return builtinCountersForTest(), nil
		},
		resolveMembers: func(_ context.Context, task *Task) ([]pmstream.TaskMember, error) {
			return networkMembers, nil
		},
		save: func(_ context.Context, req pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error) {
			captured = append(captured, req)
			return &pmstream.TaskVersionSnapshot{NewVersion: true}, nil
		},
	}

	result, err := reconciler.Reconcile(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, result.Definitions)
	require.Equal(t, 1, result.Saved)
	require.Equal(t, 1, result.Changed)
	require.Len(t, captured, 1)
	require.Equal(t, task.ID, captured[0].TaskID)
	require.Equal(t, task.Name, captured[0].Name)
	require.Equal(t, "lte", captured[0].Technology)
	require.Equal(t, pmstream.DimensionNetwork, captured[0].Dimension)
	require.Equal(t, []pmstream.Granularity{
		pmstream.GranularityHourly, pmstream.GranularityDaily,
		pmstream.GranularityWeekly, pmstream.GranularityMonthly,
	}, captured[0].Granularities)
	require.Equal(t, networkMembers, captured[0].Members)
	require.Equal(t, builtinRulesForTest(), captured[0].Metrics)
	require.Equal(t, builtinCountersForTest(), captured[0].Counters)
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
	require.Equal(t, 2, result.Definitions)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, 1, result.Saved)
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
	return []pmstream.MetricRule{{
		MetricID: "K1", MetricPath: "K1", MetricType: "counter",
		Aggregation: pmstream.AggregationSum,
	}}
}

func builtinCountersForTest() []pmstream.CounterRule {
	return []pmstream.CounterRule{{
		MetricPath: "K1", Aggregation: pmstream.AggregationSum,
	}}
}
