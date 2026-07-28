package storageprotection

import (
	"context"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/stretchr/testify/require"
)

type fakeStorageCollector struct{ metrics []components.StorageMetric }

func (c fakeStorageCollector) Collect(context.Context) []components.StorageMetric { return c.metrics }

func TestCollectorUsageProvider(t *testing.T) {
	total, used, ratio := uint64(1000), uint64(800), float64(80)
	at := time.Now()
	provider := NewCollectorUsageProvider(fakeStorageCollector{metrics: []components.StorageMetric{{
		TargetType: "application", TargetID: "root", Status: "available",
		TotalBytes: &total, UsedBytes: &used, UsedPercent: &ratio, CollectedAt: &at,
	}}})
	snapshot, err := provider.Snapshot(context.Background(), TargetApplication, "root")
	require.NoError(t, err)
	require.True(t, snapshot.Available)
	require.Equal(t, .8, snapshot.UsedRatio)
}
