package streamtest

import (
	"testing"
	"time"

	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"github.com/stretchr/testify/require"
)

func TestGeneratorIsDeterministicForDeviceAndSlot(t *testing.T) {
	generator := Generator{
		Devices: 10000, Metrics: 4,
		SlotStart: time.Date(2026, 7, 25, 1, 3, 0, 0, time.UTC),
	}
	first, err := generator.Event(9999)
	require.NoError(t, err)
	second, err := generator.Event(9999)
	require.NoError(t, err)
	require.Equal(t, pmstream.SchemaVersion, first.SchemaVersion)
	require.Equal(t, first.DeviceID, second.DeviceID)
	require.Equal(t, first.SourceFileID, second.SourceFileID)
	require.Len(t, first.Measurements[0].Metrics, 4)
	require.Equal(t, "LOAD-00010000", first.DeviceSN)
	require.Equal(t, time.Date(2026, 7, 25, 1, 0, 0, 0, time.UTC), first.WindowStart)
}
