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
