package components

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrometheusStorageCollectorReturnsRealSourceSpecificMetrics(t *testing.T) {
	now := time.Date(2026, time.July, 21, 2, 30, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var result string
		value := func(metricValue string) string {
			if strings.Contains(query, "timestamp(") {
				return fmt.Sprintf("%d", now.Unix())
			}
			return metricValue
		}
		switch {
		case strings.Contains(query, "node_filesystem_size_bytes"):
			result = vectorSample(now, map[string]string{"instance": "node-exporter:9100", "mountpoint": "/", "device": "/dev/vda1", "fstype": "ext4"}, value("1000"))
		case strings.Contains(query, "node_filesystem_avail_bytes"):
			result = vectorSample(now, map[string]string{"instance": "node-exporter:9100", "mountpoint": "/", "device": "/dev/vda1", "fstype": "ext4"}, value("250"))
		case strings.Contains(query, "node_filesystem_files_free"):
			result = vectorSample(now, map[string]string{"instance": "node-exporter:9100", "mountpoint": "/", "device": "/dev/vda1", "fstype": "ext4"}, value("200"))
		case strings.Contains(query, "node_filesystem_files"):
			result = vectorSample(now, map[string]string{"instance": "node-exporter:9100", "mountpoint": "/", "device": "/dev/vda1", "fstype": "ext4"}, value("800"))
		case strings.Contains(query, "minio_cluster_capacity_usable_total_bytes"):
			result = vectorSample(now, map[string]string{}, value("2000"))
		case strings.Contains(query, "minio_cluster_capacity_usable_free_bytes"):
			result = vectorSample(now, map[string]string{}, value("500"))
		case strings.Contains(query, "pg_database_size_bytes"):
			result = vectorSample(now, map[string]string{"instance": "postgres:5432"}, value("321"))
		default:
			http.Error(w, "unexpected query", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"success","data":{"resultType":"vector","result":[%s]}}`, result)
	}))
	defer server.Close()

	collector := NewPrometheusStorageCollector(server.URL, 2*time.Second, 60*time.Second, server.Client())
	collector.now = func() time.Time { return now }
	metrics := collector.Collect(t.Context())

	host := requireStorageMetric(t, metrics, "host_filesystem")
	require.Equal(t, "prometheus/node_exporter", host.Source)
	require.Equal(t, "/", host.MountPath)
	require.Equal(t, uint64(1000), requireValue(t, host.TotalBytes))
	require.Equal(t, uint64(750), requireValue(t, host.UsedBytes))
	require.Equal(t, uint64(250), requireValue(t, host.AvailableBytes))
	require.InDelta(t, 75, requirePercent(t, host.UsedPercent), 0.001)
	requireStorageMetricJSON(t, host, map[string]any{
		"target_type":        "host_filesystem",
		"target_id":          "host-node-exporter-9100-dev-vda1-ext4",
		"mountpoint":         "/",
		"total_inodes":       float64(800),
		"available_inodes":   float64(200),
		"used_inodes":        float64(600),
		"used_inode_percent": float64(75),
	})

	minio := requireStorageMetric(t, metrics, "minio_cluster")
	require.Equal(t, "prometheus/minio", minio.Source)
	require.Equal(t, uint64(2000), requireValue(t, minio.TotalBytes))
	require.Equal(t, uint64(500), requireValue(t, minio.AvailableBytes))
	require.InDelta(t, 75, requirePercent(t, minio.UsedPercent), 0.001)

	database := requireStorageMetric(t, metrics, "database_logical")
	require.Equal(t, "prometheus/otelcol", database.Source)
	require.Equal(t, "postgres:5432", database.Instance)
	require.Equal(t, uint64(321), requireValue(t, database.UsedBytes))
	require.Nil(t, database.TotalBytes)
	require.Nil(t, database.UsedPercent, "logical database size is not a filesystem capacity percentage")
}

func TestPrometheusStorageCollectorRejectsStaleSourceSamples(t *testing.T) {
	now := time.Date(2026, time.July, 21, 2, 30, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		value := "100"
		if strings.Contains(query, "timestamp(") {
			value = fmt.Sprintf("%d", now.Add(-2*time.Minute).Unix())
		}
		labels := map[string]string{}
		if strings.Contains(query, "node_filesystem") {
			labels = map[string]string{"instance": "node:9100", "mountpoint": "/", "device": "/dev/vda1", "fstype": "ext4"}
		} else if strings.Contains(query, "pg_database_size_bytes") {
			labels = map[string]string{"instance": "postgres:5432"}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"success","data":{"resultType":"vector","result":[%s]}}`, vectorSample(now, labels, value))
	}))
	defer server.Close()

	collector := NewPrometheusStorageCollector(server.URL, time.Second, time.Minute, server.Client())
	collector.now = func() time.Time { return now }
	metrics := collector.Collect(t.Context())

	require.NotEmpty(t, metrics)
	for _, metric := range metrics {
		require.Equal(t, "stale", metric.Status)
		require.NotNil(t, metric.CollectedAt)
		require.Equal(t, now.Add(-2*time.Minute), *metric.CollectedAt)
		require.Contains(t, metric.Error, "stale")
	}
}

func TestPrometheusStorageCollectorDoesNotServeFreshCacheWithStaleSamples(t *testing.T) {
	firstNow := time.Date(2026, time.July, 21, 2, 30, 0, 0, time.UTC)
	currentNow := firstNow
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		query := r.URL.Query().Get("query")
		labels := map[string]string{}
		value := "100"
		if strings.Contains(query, "timestamp(") {
			value = fmt.Sprintf("%d", firstNow.Unix())
		}
		if strings.Contains(query, "node_filesystem") {
			labels = map[string]string{"instance": "node:9100", "mountpoint": "/", "device": "/dev/vda1", "fstype": "ext4"}
		} else if strings.Contains(query, "pg_database_size_bytes") {
			labels = map[string]string{"instance": "postgres:5432"}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"success","data":{"resultType":"vector","result":[%s]}}`, vectorSample(currentNow, labels, value))
	}))
	defer server.Close()

	collector := NewPrometheusStorageCollector(server.URL, time.Second, 10*time.Second, server.Client())
	collector.now = func() time.Time { return currentNow }
	require.NotEmpty(t, collector.Collect(t.Context()))
	firstRequests := requests.Load()

	currentNow = currentNow.Add(11 * time.Second) // cache TTL is 15s, source max age is only 10s
	metrics := collector.Collect(t.Context())
	require.Greater(t, requests.Load(), firstRequests, "stale source samples must bypass the response cache")
	for _, metric := range metrics {
		require.Equal(t, "stale", metric.Status)
		require.NotNil(t, metric.CollectedAt)
		require.Equal(t, firstNow, *metric.CollectedAt)
		require.Contains(t, metric.Error, "stale")
	}
}

func TestBuildHostFilesystemMetricsRequiresExactLabelMatch(t *testing.T) {
	now := time.Now().UTC()
	sizes := []prometheusSample{{
		labels: map[string]string{"instance": "node:9100", "device": "/dev/vda1", "mountpoint": "/", "fstype": "ext4", "cluster": "a"},
		value:  1000, at: now,
	}}
	available := []prometheusSample{{
		labels: map[string]string{"instance": "node:9100", "device": "/dev/vda1", "mountpoint": "/", "fstype": "ext4", "cluster": "b"},
		value:  250, at: now,
	}}

	metrics := buildHostFilesystemMetrics(sizes, available, nil, nil, nil)
	require.Len(t, metrics, 2)
	for _, metric := range metrics {
		require.Equal(t, "unavailable", metric.Status)
		require.Nil(t, metric.UsedPercent)
	}
}

func TestBuildHostFilesystemMetricsDeduplicatesBindMountsOfSameDevice(t *testing.T) {
	now := time.Now().UTC()
	base := map[string]string{"instance": "node:9100", "device": "/dev/vda1", "fstype": "ext4"}
	sample := func(mountpoint string, value float64) prometheusSample {
		labels := make(map[string]string, len(base)+1)
		for key, labelValue := range base {
			labels[key] = labelValue
		}
		labels["mountpoint"] = mountpoint
		return prometheusSample{labels: labels, value: value, at: now}
	}

	metrics := buildHostFilesystemMetrics(
		[]prometheusSample{sample("/var/lib", 1000), sample("/var/lib/docker", 1000)},
		[]prometheusSample{sample("/var/lib", 250), sample("/var/lib/docker", 250)}, nil, nil, nil,
	)
	require.Len(t, metrics, 1)
	require.Equal(t, "/var/lib", metrics[0].MountPath)
	require.Equal(t, uint64(1000), requireValue(t, metrics[0].TotalBytes))
}

func TestNodeFilesystemQueriesExcludePseudoFilesystems(t *testing.T) {
	for _, query := range []string{
		nodeSizeQuery, nodeSizeTimeQuery, nodeAvailQuery, nodeAvailTimeQuery,
		nodeFilesQuery, nodeFilesTimeQuery, nodeFilesFreeQuery, nodeFilesFreeTimeQuery,
	} {
		require.Contains(t, query, "fakeowner")
		require.Contains(t, query, "selfowner")
		require.Contains(t, query, "virtiofs")
		require.Contains(t, query, "fuse")
	}
}

func TestCollectAppFilesystemMetricIncludesInodesAndTargetDimensions(t *testing.T) {
	collectedAt := time.Date(2026, time.July, 28, 8, 0, 0, 0, time.UTC)
	metric := collectAppFilesystemMetric("/app/logs", collectedAt, func(string) (filesystemStats, error) {
		return filesystemStats{
			blockSize:       1,
			blocks:          100,
			availableBlocks: 25,
			files:           50,
			availableFiles:  10,
		}, nil
	})

	requireStorageMetricJSON(t, metric, map[string]any{
		"target_type":        "application",
		"target_id":          "app-logs",
		"mountpoint":         "/app/logs",
		"total_inodes":       float64(50),
		"available_inodes":   float64(10),
		"used_inodes":        float64(40),
		"used_inode_percent": float64(80),
	})
}

func TestStorageCompositeTimestampUsesOldestInputSample(t *testing.T) {
	newer := time.Date(2026, time.July, 21, 2, 30, 0, 0, time.UTC)
	older := newer.Add(-5 * time.Second)
	labels := map[string]string{"instance": "node:9100", "device": "/dev/vda1", "mountpoint": "/", "fstype": "ext4"}
	host := buildHostFilesystemMetrics(
		[]prometheusSample{{labels: labels, value: 1000, at: newer}},
		[]prometheusSample{{labels: labels, value: 250, at: older}}, nil, nil, nil,
	)
	require.Equal(t, older, *host[0].CollectedAt)

	minio := buildMinIOMetric(
		[]prometheusSample{{value: 2000, at: newer}},
		[]prometheusSample{{value: 500, at: older}}, nil,
	)
	require.Equal(t, older, *minio.CollectedAt)
}

func TestPrometheusStorageCollectorReturnsUnavailableInsteadOfFabricatingValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "monitoring unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	collector := NewPrometheusStorageCollector(server.URL, time.Second, 60*time.Second, server.Client())
	metrics := collector.Collect(t.Context())

	require.NotEmpty(t, metrics)
	for _, metric := range metrics {
		require.Equal(t, "unavailable", metric.Status)
		require.Nil(t, metric.TotalBytes)
		require.Nil(t, metric.UsedBytes)
		require.Nil(t, metric.AvailableBytes)
		require.Nil(t, metric.UsedPercent)
		require.NotEmpty(t, metric.Error)
	}
}

func vectorSample(at time.Time, labels map[string]string, value string) string {
	labelJSON := ""
	for key, labelValue := range labels {
		if labelJSON != "" {
			labelJSON += ","
		}
		labelJSON += fmt.Sprintf("%q:%q", key, labelValue)
	}
	return fmt.Sprintf(`{"metric":{%s},"value":[%d,%q]}`, labelJSON, at.Unix(), value)
}

func requireStorageMetric(t *testing.T, metrics []StorageMetric, kind string) StorageMetric {
	t.Helper()
	for _, metric := range metrics {
		if metric.Kind == kind && metric.Status == "available" {
			return metric
		}
	}
	t.Fatalf("available storage metric kind %q not found in %#v", kind, metrics)
	return StorageMetric{}
}

func requireValue(t *testing.T, value *uint64) uint64 {
	t.Helper()
	require.NotNil(t, value)
	return *value
}

func requirePercent(t *testing.T, value *float64) float64 {
	t.Helper()
	require.NotNil(t, value)
	return *value
}

func requireStorageMetricJSON(t *testing.T, metric StorageMetric, expected map[string]any) {
	t.Helper()
	encoded, err := json.Marshal(metric)
	require.NoError(t, err)
	var actual map[string]any
	require.NoError(t, json.Unmarshal(encoded, &actual))
	for key, value := range expected {
		require.Equal(t, value, actual[key], key)
	}
}
