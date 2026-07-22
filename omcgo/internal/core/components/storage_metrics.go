package components

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const storageMetricsCacheTTL = 15 * time.Second

type StorageMetric struct {
	ID             string     `json:"id"`
	Kind           string     `json:"kind"`
	Label          string     `json:"label"`
	Source         string     `json:"source"`
	MountPath      string     `json:"mount_path,omitempty"`
	Instance       string     `json:"instance,omitempty"`
	TotalBytes     *uint64    `json:"total_bytes,omitempty"`
	UsedBytes      *uint64    `json:"used_bytes,omitempty"`
	AvailableBytes *uint64    `json:"available_bytes,omitempty"`
	UsedPercent    *float64   `json:"used_percent,omitempty"`
	CollectedAt    *time.Time `json:"collected_at,omitempty"`
	Status         string     `json:"status"`
	Error          string     `json:"error,omitempty"`
}

type StorageCollector interface {
	Collect(ctx context.Context) []StorageMetric
}

type PrometheusStorageCollector struct {
	baseURL *url.URL
	initErr error
	timeout time.Duration
	maxAge  time.Duration
	client  *http.Client
	now     func() time.Time

	mu        sync.Mutex
	cachedAt  time.Time
	cachedVal []StorageMetric
}

func NewPrometheusStorageCollector(baseURL string, timeout, maxAge time.Duration, client *http.Client) *PrometheusStorageCollector {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if maxAge <= 0 {
		maxAge = time.Minute
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err == nil && (parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "") {
		err = fmt.Errorf("Prometheus URL must be an absolute HTTP(S) URL")
	}
	return &PrometheusStorageCollector{
		baseURL: parsed,
		initErr: err,
		timeout: timeout,
		maxAge:  maxAge,
		client:  client,
		now:     time.Now,
	}
}

func (c *PrometheusStorageCollector) Collect(ctx context.Context) []StorageMetric {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now().UTC()
	if !c.cachedAt.IsZero() && now.Sub(c.cachedAt) < storageMetricsCacheTTL && c.cachedMetricsFresh(now) {
		return append([]StorageMetric(nil), c.cachedVal...)
	}

	metrics := c.collectUncached(ctx, now)
	c.cachedAt = now
	c.cachedVal = append([]StorageMetric(nil), metrics...)
	return metrics
}

func (c *PrometheusStorageCollector) cachedMetricsFresh(now time.Time) bool {
	for _, metric := range c.cachedVal {
		if metric.Status != "available" {
			continue
		}
		if metric.CollectedAt == nil {
			return false
		}
		age := now.Sub(*metric.CollectedAt)
		if age > c.maxAge || age < -5*time.Second {
			return false
		}
	}
	return true
}

const (
	// Docker Desktop and container hosts expose synthetic bind/image filesystems
	// with plausible-looking but non-physical capacities. Exclude them explicitly;
	// the remaining samples represent writable host/device filesystems.
	nodeFilesystemSelector = `{fstype!~="^(tmpfs|overlay|squashfs|ramfs|erofs|fakeowner|selfowner|virtiofs(\\..*)?|fuse(\\..*)?)$"}`
	nodeSizeQuery          = `node_filesystem_size_bytes` + nodeFilesystemSelector
	nodeSizeTimeQuery      = `timestamp(node_filesystem_size_bytes` + nodeFilesystemSelector + `)`
	nodeAvailQuery         = `node_filesystem_avail_bytes` + nodeFilesystemSelector
	nodeAvailTimeQuery     = `timestamp(node_filesystem_avail_bytes` + nodeFilesystemSelector + `)`
	minioTotalQuery        = `sum(minio_cluster_capacity_usable_total_bytes)`
	minioTotalTimeQuery    = `min(timestamp(minio_cluster_capacity_usable_total_bytes))`
	minioFreeQuery         = `sum(minio_cluster_capacity_usable_free_bytes)`
	minioFreeTimeQuery     = `min(timestamp(minio_cluster_capacity_usable_free_bytes))`
	pgSizeQuery            = `sum by (instance) (pg_database_size_bytes)`
	pgSizeTimeQuery        = `min by (instance) (timestamp(pg_database_size_bytes))`
)

func (c *PrometheusStorageCollector) collectUncached(ctx context.Context, now time.Time) []StorageMetric {
	if c.initErr != nil {
		return unavailableStorageMetrics(c.initErr)
	}

	sizes, sizeErr := c.query(ctx, nodeSizeQuery, nodeSizeTimeQuery, now)
	available, availableErr := c.query(ctx, nodeAvailQuery, nodeAvailTimeQuery, now)
	minioTotal, minioTotalErr := c.query(ctx, minioTotalQuery, minioTotalTimeQuery, now)
	minioFree, minioFreeErr := c.query(ctx, minioFreeQuery, minioFreeTimeQuery, now)
	databaseSizes, databaseErr := c.query(ctx, pgSizeQuery, pgSizeTimeQuery, now)

	metrics := make([]StorageMetric, 0, len(sizes)+3)
	metrics = append(metrics, buildHostFilesystemMetrics(sizes, available, errorsJoin(sizeErr, availableErr))...)
	metrics = append(metrics, buildMinIOMetric(minioTotal, minioFree, errorsJoin(minioTotalErr, minioFreeErr)))
	metrics = append(metrics, buildDatabaseMetrics(databaseSizes, databaseErr)...)
	return metrics
}

type prometheusSample struct {
	labels map[string]string
	value  float64
	at     time.Time
}

func (c *PrometheusStorageCollector) query(ctx context.Context, valueQuery, timestampQuery string, now time.Time) ([]prometheusSample, error) {
	values, err := c.queryRaw(ctx, valueQuery)
	if err != nil {
		return nil, err
	}
	timestamps, err := c.queryRaw(ctx, timestampQuery)
	if err != nil {
		return nil, fmt.Errorf("query source timestamps: %w", err)
	}
	timestampByLabels := make(map[string]prometheusSample, len(timestamps))
	for _, sample := range timestamps {
		timestampByLabels[prometheusLabelsKey(sample.labels)] = sample
	}
	for i := range values {
		timestamp, ok := timestampByLabels[prometheusLabelsKey(values[i].labels)]
		if !ok {
			return nil, fmt.Errorf("Prometheus source timestamp is missing for metric labels")
		}
		at := time.Unix(0, int64(timestamp.value*float64(time.Second))).UTC()
		age := now.Sub(at)
		if age > c.maxAge || age < -5*time.Second {
			return nil, fmt.Errorf("Prometheus source sample is stale or from the future: collected_at=%s", at.Format(time.RFC3339))
		}
		values[i].at = at
	}
	return values, nil
}

func (c *PrometheusStorageCollector) queryRaw(ctx context.Context, query string) ([]prometheusSample, error) {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/v1/query"
	params := endpoint.Query()
	params.Set("query", query)
	endpoint.RawQuery = params.Encode()

	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build Prometheus query: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query Prometheus: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Prometheus returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("read Prometheus response: %w", err)
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []json.RawMessage `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode Prometheus response: %w", err)
	}
	if envelope.Status != "success" || envelope.Data.ResultType != "vector" {
		return nil, fmt.Errorf("unexpected Prometheus response status=%q resultType=%q", envelope.Status, envelope.Data.ResultType)
	}

	samples := make([]prometheusSample, 0, len(envelope.Data.Result))
	for _, result := range envelope.Data.Result {
		if len(result.Value) != 2 {
			return nil, fmt.Errorf("Prometheus sample has %d value fields", len(result.Value))
		}
		var timestamp float64
		var rawValue string
		if err := json.Unmarshal(result.Value[0], &timestamp); err != nil {
			return nil, fmt.Errorf("decode Prometheus sample timestamp: %w", err)
		}
		if err := json.Unmarshal(result.Value[1], &rawValue); err != nil {
			return nil, fmt.Errorf("decode Prometheus sample value: %w", err)
		}
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return nil, fmt.Errorf("invalid Prometheus sample value %q", rawValue)
		}
		at := time.Unix(0, int64(timestamp*float64(time.Second))).UTC()
		samples = append(samples, prometheusSample{labels: result.Metric, value: value, at: at})
	}
	return samples, nil
}

func prometheusLabelsKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		if key != "__name__" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(labels[key])
		builder.WriteByte(0)
	}
	return builder.String()
}

func buildHostFilesystemMetrics(sizes, available []prometheusSample, queryErr error) []StorageMetric {
	if queryErr != nil {
		return []StorageMetric{unavailableStorageMetric("host-filesystem", "host_filesystem", "Host filesystems", "prometheus/node_exporter", queryErr)}
	}
	type pair struct{ size, available *prometheusSample }
	pairs := make(map[string]*pair)
	keyFor := func(sample prometheusSample) string {
		return prometheusLabelsKey(sample.labels)
	}
	for i := range sizes {
		key := keyFor(sizes[i])
		pairs[key] = &pair{size: &sizes[i]}
	}
	for i := range available {
		key := keyFor(available[i])
		if pairs[key] == nil {
			pairs[key] = &pair{}
		}
		pairs[key].available = &available[i]
	}
	keys := make([]string, 0, len(pairs))
	for key := range pairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	metrics := make([]StorageMetric, 0, len(keys))
	bestByDevice := make(map[string]StorageMetric)
	for _, key := range keys {
		pair := pairs[key]
		if pair.size == nil || pair.available == nil || pair.size.value <= 0 || pair.available.value > pair.size.value {
			metrics = append(metrics, unavailableStorageMetric("host-"+safeMetricID(key), "host_filesystem", "Host filesystem", "prometheus/node_exporter", fmt.Errorf("size and available samples do not match")))
			continue
		}
		total := uint64(pair.size.value)
		avail := uint64(pair.available.value)
		used := total - avail
		pct := math.Round(float64(used)*10000/float64(total)) / 100
		collectedAt := oldestTime(pair.size.at, pair.available.at)
		deviceKey := pair.size.labels["instance"] + "\x00" + pair.size.labels["device"] + "\x00" + pair.size.labels["fstype"]
		metric := StorageMetric{
			ID: "host-" + safeMetricID(deviceKey), Kind: "host_filesystem", Label: pair.size.labels["mountpoint"],
			Source: "prometheus/node_exporter", MountPath: pair.size.labels["mountpoint"], Instance: pair.size.labels["instance"],
			TotalBytes: &total, UsedBytes: &used, AvailableBytes: &avail, UsedPercent: &pct,
			CollectedAt: &collectedAt, Status: "available",
		}
		current, exists := bestByDevice[deviceKey]
		if !exists || betterMountPath(metric.MountPath, current.MountPath) {
			bestByDevice[deviceKey] = metric
		}
	}
	deviceKeys := make([]string, 0, len(bestByDevice))
	for key := range bestByDevice {
		deviceKeys = append(deviceKeys, key)
	}
	sort.Strings(deviceKeys)
	for _, key := range deviceKeys {
		metrics = append(metrics, bestByDevice[key])
	}
	if len(metrics) == 0 {
		return []StorageMetric{unavailableStorageMetric("host-filesystem", "host_filesystem", "Host filesystems", "prometheus/node_exporter", fmt.Errorf("no host filesystem samples"))}
	}
	return metrics
}

func betterMountPath(candidate, current string) bool {
	return len(candidate) < len(current) || len(candidate) == len(current) && candidate < current
}

func buildMinIOMetric(totalSamples, freeSamples []prometheusSample, queryErr error) StorageMetric {
	if queryErr != nil || len(totalSamples) != 1 || len(freeSamples) != 1 {
		if queryErr == nil {
			queryErr = fmt.Errorf("expected one MinIO total and free sample")
		}
		return unavailableStorageMetric("minio-cluster", "minio_cluster", "MinIO cluster", "prometheus/minio", queryErr)
	}
	total := uint64(totalSamples[0].value)
	free := uint64(freeSamples[0].value)
	if total == 0 || free > total {
		return unavailableStorageMetric("minio-cluster", "minio_cluster", "MinIO cluster", "prometheus/minio", fmt.Errorf("invalid MinIO total/free capacity"))
	}
	used := total - free
	pct := math.Round(float64(used)*10000/float64(total)) / 100
	collectedAt := oldestTime(totalSamples[0].at, freeSamples[0].at)
	return StorageMetric{ID: "minio-cluster", Kind: "minio_cluster", Label: "MinIO cluster", Source: "prometheus/minio", TotalBytes: &total, UsedBytes: &used, AvailableBytes: &free, UsedPercent: &pct, CollectedAt: &collectedAt, Status: "available"}
}

func buildDatabaseMetrics(samples []prometheusSample, queryErr error) []StorageMetric {
	if queryErr != nil {
		return []StorageMetric{unavailableStorageMetric("database-logical", "database_logical", "Database logical size", "prometheus/otelcol", queryErr)}
	}
	metrics := make([]StorageMetric, 0, len(samples))
	for _, sample := range samples {
		used := uint64(sample.value)
		collectedAt := sample.at
		instance := sample.labels["instance"]
		metrics = append(metrics, StorageMetric{ID: "database-" + safeMetricID(instance), Kind: "database_logical", Label: instance, Source: "prometheus/otelcol", Instance: instance, UsedBytes: &used, CollectedAt: &collectedAt, Status: "available"})
	}
	if len(metrics) == 0 {
		return []StorageMetric{unavailableStorageMetric("database-logical", "database_logical", "Database logical size", "prometheus/otelcol", fmt.Errorf("no database size samples"))}
	}
	return metrics
}

func unavailableStorageMetrics(err error) []StorageMetric {
	return []StorageMetric{
		unavailableStorageMetric("host-filesystem", "host_filesystem", "Host filesystems", "prometheus/node_exporter", err),
		unavailableStorageMetric("minio-cluster", "minio_cluster", "MinIO cluster", "prometheus/minio", err),
		unavailableStorageMetric("database-logical", "database_logical", "Database logical size", "prometheus/otelcol", err),
	}
}

func unavailableStorageMetric(id, kind, label, source string, err error) StorageMetric {
	message := "metric unavailable"
	if err != nil {
		message = err.Error()
	}
	return StorageMetric{ID: id, Kind: kind, Label: label, Source: source, Status: "unavailable", Error: message}
}

func errorsJoin(errs ...error) error {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(parts, "; "))
}

func safeMetricID(value string) string {
	value = strings.Trim(value, "\x00/ ")
	replacer := strings.NewReplacer("\x00", "-", "/", "-", ":", "-", " ", "-")
	return replacer.Replace(value)
}

func oldestTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}
