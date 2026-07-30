package adhoc

import (
	"testing"

	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/stretchr/testify/require"
)

func TestStreamingOutputMetricPathsUnionsTaskAndEnabledByMetricPath(t *testing.T) {
	got := streamingOutputMetricPaths(
		[]string{"K_TASK", "K_SHARED", "C_SELECTED", "K_TASK"},
		[]string{"K_ENABLED", "K_SHARED", "C_ENABLED", "K_ENABLED"},
	)

	require.Equal(t, []string{
		"K_TASK",
		"K_SHARED",
		"C_SELECTED",
		"K_ENABLED",
		"C_ENABLED",
	}, got)
}

func TestStreamingOutputMetricPathsSkipsEmptyPaths(t *testing.T) {
	got := streamingOutputMetricPaths(
		[]string{"", "K_TASK"},
		[]string{"", "K_ENABLED"},
	)

	require.Equal(t, []string{"K_TASK", "K_ENABLED"}, got)
}

func TestStreamingTaskOutputMetricPathsCustomUsesOnlyTaskSelection(t *testing.T) {
	got := streamingTaskOutputMetricPaths(
		&Task{
			IsBuiltin:   false,
			MetricPaths: []string{"C000020013", "C000020014"},
		},
		[]string{"C000010228", "K_ENABLED"},
	)

	require.Equal(t, []string{"C000020013", "C000020014"}, got)
}

func TestStreamingTaskOutputMetricPathsBuiltinIncludesEnabledMetrics(t *testing.T) {
	got := streamingTaskOutputMetricPaths(
		&Task{
			IsBuiltin:   true,
			MetricPaths: []string{"K_TASK"},
		},
		[]string{"K_ENABLED", "K_TASK"},
	)

	require.Equal(t, []string{"K_TASK", "K_ENABLED"}, got)
}

func TestShouldRejectEmptyStreamingMembersCustomOnly(t *testing.T) {
	require.True(t, shouldRejectEmptyStreamingMembers(&Task{IsBuiltin: false}, nil))
	require.False(t, shouldRejectEmptyStreamingMembers(&Task{IsBuiltin: true}, nil))
	require.False(t, shouldRejectEmptyStreamingMembers(nil, nil))
}

func TestStreamingTaskCreatorDefaultsToSystem(t *testing.T) {
	require.Equal(t, "system", streamingTaskCreator(nil))
	require.Equal(t, "system", streamingTaskCreator(&Task{}))
	require.Equal(t, "alice", streamingTaskCreator(&Task{Creator: "alice"}))
}

func TestStreamingEnabledDeviceTypesUsesTaskTechnology(t *testing.T) {
	got, err := streamingEnabledDeviceTypes("lte")

	require.NoError(t, err)
	require.Equal(t, []indicator.DeviceType{indicator.DeviceTypeENB}, got)
}

func TestStreamingEnabledDeviceTypesEmptyTechnologyUsesAllTypes(t *testing.T) {
	got, err := streamingEnabledDeviceTypes("")

	require.NoError(t, err)
	require.Equal(t, []indicator.DeviceType{
		indicator.DeviceTypeENB,
		indicator.DeviceTypeGNB,
		indicator.DeviceTypeGSM,
	}, got)
}

func TestStreamingEnabledDeviceTypesRejectsUnsupportedTechnology(t *testing.T) {
	_, err := streamingEnabledDeviceTypes("wifi")

	require.ErrorContains(t, err, "unsupported technology")
}
