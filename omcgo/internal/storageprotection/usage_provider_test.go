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

type staticProtectedPathResolver struct{ paths []ProtectedPath }

func (r staticProtectedPathResolver) ProtectedPaths(context.Context) ([]ProtectedPath, error) {
	return append([]ProtectedPath(nil), r.paths...), nil
}

func TestCollectorUsageProvider(t *testing.T) {
	total, used, ratio := uint64(1000), uint64(800), float64(80)
	at := time.Now()
	provider := NewCollectorUsageProvider(fakeStorageCollector{metrics: []components.StorageMetric{{
		Kind: "host_filesystem", Mountpoint: "/", TargetType: "host_filesystem", TargetID: "host-node-root", Status: "available",
		TotalBytes: &total, UsedBytes: &used, UsedPercent: &ratio, CollectedAt: &at,
	}}})
	snapshot, err := provider.Snapshot(context.Background(), TargetFilesystem, UnifiedStorageTargetID)
	require.NoError(t, err)
	require.True(t, snapshot.Available)
	require.Equal(t, .8, snapshot.UsedRatio)
}

func TestCollectorUsageProviderFallsBackToContainerRootWhenHostMetricsMissing(t *testing.T) {
	total, used, ratio := uint64(1000), uint64(800), float64(80)
	at := time.Now()
	provider := NewCollectorUsageProviderWithResolver(
		fakeStorageCollector{metrics: []components.StorageMetric{{
			Kind: "minio_cluster", TargetType: "", TargetID: "", Status: "available",
			TotalBytes: &total, UsedBytes: &used, UsedPercent: &ratio, CollectedAt: &at,
		}}},
		staticProtectedPathResolver{paths: []ProtectedPath{
			{ID: "pg", Path: "/var/lib/docker/volumes/omcgo_pgdata/_data"},
			{ID: "logs", Path: "/opt/omc/run/logs"},
		}},
	)
	snapshot, err := provider.Snapshot(context.Background(), TargetFilesystem, UnifiedStorageTargetID)
	require.NoError(t, err)
	require.True(t, snapshot.Available)
	require.Equal(t, UnifiedStorageTargetID, snapshot.TargetID)
	require.Equal(t, UnifiedStorageMountpoint, snapshot.Mountpoint)
	require.Contains(t, snapshot.ProtectedPaths, "/var/lib/docker/volumes/omcgo_pgdata/_data")
	require.Contains(t, snapshot.ProtectedPaths, "/opt/omc/run/logs")
	require.Contains(t, snapshot.Reason, "host filesystem metrics unavailable")
}

func TestEnvProtectedPathResolverUsesNamedVolumesForEmptyDataPaths(t *testing.T) {
	env := map[string]string{
		"POSTGRES_DATA_PATH":         "",
		"TSDB_DATA_PATH":             "/srv/tsdb",
		"OMCGO_DOCKER_ROOT_DIR":      "/mnt/docker",
		"OMCGO_DOCKER_VOLUME_PREFIX": "omcprod",
	}
	lookup := func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}
	resolver := NewEnvProtectedPathResolverWithVolumeResolver(lookup, nil)

	paths, err := resolver.ProtectedPaths(context.Background())
	require.NoError(t, err)

	require.Contains(t, protectedPathValues(paths), "/mnt/docker/volumes/omcprod_pgdata/_data")
	require.Contains(t, protectedPathValues(paths), "/srv/tsdb")
	require.Contains(t, protectedPathValues(paths), "/opt/omc/run/logs")
	require.Contains(t, protectedPathValues(paths), "/opt/omc/data")
	require.Contains(t, protectedPathValues(paths), "/mnt/docker")
}

func TestCollectorUsageProviderDeduplicatesProtectedPathsOnSameMountpoint(t *testing.T) {
	rootTotal, rootUsed, rootRatio := uint64(1000), uint64(500), float64(50)
	dataTotal, dataUsed, dataRatio := uint64(2000), uint64(1700), float64(85)
	at := time.Now()
	provider := NewCollectorUsageProviderWithResolver(
		fakeStorageCollector{metrics: []components.StorageMetric{
			hostMetric("/", rootTotal, rootUsed, rootRatio, at),
			hostMetric("/data", dataTotal, dataUsed, dataRatio, at),
		}},
		staticProtectedPathResolver{paths: []ProtectedPath{
			{ID: "POSTGRES_DATA_PATH", Path: "/data/postgres"},
			{ID: "MINIO_DATA_PATH", Path: "/data/minio"},
		}},
	)

	targets, err := provider.ListTargets(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, "mount-data", targets[0].TargetID)
	require.Equal(t, "/data", targets[0].Mountpoint)
	require.ElementsMatch(t, []string{"/data/postgres", "/data/minio"}, targets[0].ProtectedPaths)
}

func TestCollectorUsageProviderSnapshotUsesHighestProtectedMountpointUsage(t *testing.T) {
	logsTotal, logsUsed, logsRatio := uint64(1000), uint64(700), float64(70)
	dockerTotal, dockerUsed, dockerRatio := uint64(1000), uint64(950), float64(95)
	at := time.Now()
	provider := NewCollectorUsageProviderWithResolver(
		fakeStorageCollector{metrics: []components.StorageMetric{
			hostMetric("/opt", logsTotal, logsUsed, logsRatio, at),
			hostMetric("/var/lib/docker", dockerTotal, dockerUsed, dockerRatio, at),
		}},
		staticProtectedPathResolver{paths: []ProtectedPath{
			{ID: "logs", Path: "/opt/omc/run/logs"},
			{ID: "pg", Path: "/var/lib/docker/volumes/omcgo_pgdata/_data"},
		}},
	)

	snapshot, err := provider.Snapshot(context.Background(), TargetFilesystem, UnifiedStorageTargetID)
	require.NoError(t, err)
	require.True(t, snapshot.Available)
	require.Equal(t, UnifiedStorageTargetID, snapshot.TargetID)
	require.Equal(t, "/var/lib/docker", snapshot.Mountpoint)
	require.Equal(t, .95, snapshot.UsedRatio)
	require.Contains(t, snapshot.Reason, "/var/lib/docker")
}

func protectedPathValues(paths []ProtectedPath) []string {
	values := make([]string, 0, len(paths))
	for _, item := range paths {
		values = append(values, item.Path)
	}
	return values
}

func hostMetric(mountpoint string, total, used uint64, usedPercent float64, at time.Time) components.StorageMetric {
	return components.StorageMetric{
		Kind: "host_filesystem", Mountpoint: mountpoint, MountPath: mountpoint, Status: "available",
		TotalBytes: &total, UsedBytes: &used, UsedPercent: &usedPercent, CollectedAt: &at,
	}
}
