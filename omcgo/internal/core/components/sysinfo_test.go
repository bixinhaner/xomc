package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSystemInfoSerializesOnlyTypedStorageMetrics(t *testing.T) {
	encoded, err := json.Marshal(SystemInfo{})
	require.NoError(t, err)

	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &payload))
	require.Contains(t, payload, "storage", "system information must expose source-specific storage metrics")
	require.NotContains(t, payload, "disks", "the legacy duplicate disk payload must not be exposed")
}

func TestCollectFilesystemInfoReportsActualAppVisibleCapacity(t *testing.T) {
	collectedAt := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)

	info := collectAppFilesystemMetric("/", collectedAt, func(string) (filesystemStats, error) {
		return filesystemStats{blockSize: 4096, blocks: 100, availableBlocks: 20}, nil
	})

	require.Equal(t, "root", info.ID)
	require.Equal(t, "/", info.MountPath)
	require.Equal(t, "app_filesystem", info.Kind)
	require.Equal(t, "statfs/app", info.Source)
	require.Equal(t, uint64(409600), requireValue(t, info.TotalBytes))
	require.Equal(t, uint64(327680), requireValue(t, info.UsedBytes))
	require.Equal(t, uint64(81920), requireValue(t, info.AvailableBytes))
	require.Equal(t, 80.0, requirePercent(t, info.UsedPercent))
	require.Equal(t, collectedAt, *info.CollectedAt)
	require.Equal(t, "available", info.Status)
	require.Empty(t, info.Error)
}

func TestCollectFilesystemInfoReportsUnavailableWhenStatfsFails(t *testing.T) {
	errStatfs := errors.New("permission denied")
	collectedAt := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)

	info := collectAppFilesystemMetric("/app/logs", collectedAt, func(string) (filesystemStats, error) {
		return filesystemStats{}, errStatfs
	})

	require.Equal(t, "app-logs", info.ID)
	require.Equal(t, "/app/logs", info.MountPath)
	require.Equal(t, "unavailable", info.Status)
	require.Equal(t, "permission denied", info.Error)
	require.Nil(t, info.CollectedAt)
	require.Nil(t, info.TotalBytes)
	require.Nil(t, info.UsedBytes)
	require.Nil(t, info.AvailableBytes)
}

func TestAppVisibleFilesystemPathsAddsConfiguredLogDirectoryOnlyWhenPresent(t *testing.T) {
	paths := appVisibleFilesystemPaths("/app/logs", func(path string) bool {
		return path == "/app/logs"
	})
	require.Equal(t, []string{"/", "/app/logs"}, paths)

	paths = appVisibleFilesystemPaths("/app/logs", func(string) bool { return false })
	require.Equal(t, []string{"/"}, paths)
}

type staticStorageCollector []StorageMetric

func (s staticStorageCollector) Collect(context.Context) []StorageMetric {
	return append([]StorageMetric(nil), s...)
}

func TestSystemInfoCombinesAppFilesystemWithExternalStorageMetrics(t *testing.T) {
	now := time.Date(2026, time.July, 21, 3, 0, 0, 0, time.UTC)
	handler := NewSystemInfoHandler(nil, nil, zap.NewNop())
	handler.diskPaths = []string{"/app/logs"}
	handler.now = func() time.Time { return now }
	handler.statfs = func(string) (filesystemStats, error) {
		return filesystemStats{blockSize: 1, blocks: 100, availableBlocks: 15}, nil
	}
	handler.SetStorageCollector(staticStorageCollector{{
		ID: "minio-cluster", Kind: "minio_cluster", Label: "MinIO cluster",
		Source: "prometheus/minio", Status: "unavailable", Error: "metrics missing",
	}})

	metrics := handler.collectStorageMetrics(context.Background())
	require.Len(t, metrics, 2)
	require.Equal(t, "app_filesystem", metrics[0].Kind)
	require.Equal(t, "statfs/app", metrics[0].Source)
	require.Equal(t, uint64(100), requireValue(t, metrics[0].TotalBytes))
	require.Equal(t, "minio_cluster", metrics[1].Kind)
	require.Equal(t, "unavailable", metrics[1].Status)
}
