package storageprotection

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/omcgo/omcgo/internal/core/components"
)

// CollectorUsageProvider adapts the existing system-info Prometheus collector
// to the storage protection domain. It shares the collector cache, so the
// management page and admission checks observe the same source timestamp.
type CollectorUsageProvider struct {
	collector components.StorageCollector
	resolver  ProtectedPathResolver
}

func NewCollectorUsageProvider(collector components.StorageCollector) *CollectorUsageProvider {
	return &CollectorUsageProvider{collector: collector, resolver: NewEnvProtectedPathResolver()}
}

func NewCollectorUsageProviderWithResolver(collector components.StorageCollector, resolver ProtectedPathResolver) *CollectorUsageProvider {
	if resolver == nil {
		resolver = NewEnvProtectedPathResolver()
	}
	return &CollectorUsageProvider{collector: collector, resolver: resolver}
}

func (p *CollectorUsageProvider) Snapshot(ctx context.Context, targetType TargetType, targetID string) (UsageSnapshot, error) {
	if p == nil || p.collector == nil {
		return UsageSnapshot{TargetType: targetType, TargetID: targetID, Available: false, Reason: "storage collector is not configured"}, nil
	}
	targets, err := p.ListTargets(ctx)
	if err != nil {
		return UsageSnapshot{}, err
	}
	if targetType != TargetFilesystem {
		return UsageSnapshot{TargetType: targetType, TargetID: targetID, Available: false, Reason: fmt.Sprintf("storage target %s/%s was not found", targetType, targetID)}, nil
	}
	if targetID != UnifiedStorageTargetID {
		for _, target := range targets {
			if target.TargetID == targetID {
				return target, nil
			}
		}
		return UsageSnapshot{TargetType: targetType, TargetID: targetID, Available: false, Reason: fmt.Sprintf("storage target %s/%s was not found", targetType, targetID)}, nil
	}
	return worstUsageSnapshot(targets), nil
}

func (p *CollectorUsageProvider) ListTargets(ctx context.Context) ([]UsageSnapshot, error) {
	if p == nil || p.collector == nil {
		return []UsageSnapshot{{TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, Mountpoint: UnifiedStorageMountpoint, Available: false, Reason: "storage collector is not configured"}}, nil
	}
	metrics := p.collector.Collect(ctx)
	paths, err := p.protectedPaths(ctx)
	if err != nil {
		return nil, err
	}
	if unavailable := unavailableHostFilesystemSnapshot(metrics); unavailable != nil {
		if fallback, ok := containerRootFilesystemFallback(paths, unavailable.Reason); ok {
			return []UsageSnapshot{fallback}, nil
		}
		return []UsageSnapshot{*unavailable}, nil
	}
	targets := protectedMountTargets(metrics, paths)
	if len(targets) == 0 {
		if fallback, ok := containerRootFilesystemFallback(paths, "no protected host filesystem mountpoints were found"); ok {
			return []UsageSnapshot{fallback}, nil
		}
		return []UsageSnapshot{{TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, Mountpoint: UnifiedStorageMountpoint, Available: false, Reason: "no protected host filesystem mountpoints were found"}}, nil
	}
	return targets, nil
}

func (p *CollectorUsageProvider) protectedPaths(ctx context.Context) ([]ProtectedPath, error) {
	resolver := p.resolver
	if resolver == nil {
		resolver = NewEnvProtectedPathResolver()
	}
	return resolver.ProtectedPaths(ctx)
}

func unavailableHostFilesystemSnapshot(metrics []components.StorageMetric) *UsageSnapshot {
	for _, metric := range metrics {
		if metric.Kind != "host_filesystem" || metric.Status == "available" {
			continue
		}
		return &UsageSnapshot{
			TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, Mountpoint: UnifiedStorageMountpoint,
			Available: false, Reason: metric.Error, ObservedAt: valueTime(metric.CollectedAt),
		}
	}
	return nil
}

func protectedMountTargets(metrics []components.StorageMetric, paths []ProtectedPath) []UsageSnapshot {
	available := make([]components.StorageMetric, 0, len(metrics))
	for _, metric := range metrics {
		if metric.Kind == "host_filesystem" && metric.Status == "available" && metric.TotalBytes != nil && metric.UsedBytes != nil {
			available = append(available, metric)
		}
	}
	byMountpoint := make(map[string]*UsageSnapshot)
	for _, protectedPath := range paths {
		metric, ok := bestMetricForPath(available, protectedPath.Path)
		if !ok {
			continue
		}
		mountpoint := metricMountpoint(metric)
		if mountpoint == "" {
			continue
		}
		target, ok := byMountpoint[mountpoint]
		if !ok {
			snapshot := usageSnapshotFromMetric(metric)
			byMountpoint[mountpoint] = &snapshot
			target = &snapshot
		}
		target.ProtectedPaths = append(target.ProtectedPaths, protectedPath.Path)
	}
	mountpoints := make([]string, 0, len(byMountpoint))
	for mountpoint := range byMountpoint {
		mountpoints = append(mountpoints, mountpoint)
	}
	sort.Strings(mountpoints)
	targets := make([]UsageSnapshot, 0, len(mountpoints))
	for _, mountpoint := range mountpoints {
		target := *byMountpoint[mountpoint]
		target.ProtectedPaths = uniqueStrings(target.ProtectedPaths)
		targets = append(targets, target)
	}
	return targets
}

func bestMetricForPath(metrics []components.StorageMetric, targetPath string) (components.StorageMetric, bool) {
	var best components.StorageMetric
	bestLen := -1
	for _, metric := range metrics {
		mountpoint := metricMountpoint(metric)
		if !pathContains(mountpoint, targetPath) {
			continue
		}
		if len(mountpoint) > bestLen {
			best = metric
			bestLen = len(mountpoint)
		}
	}
	return best, bestLen >= 0
}

func pathContains(mountpoint, targetPath string) bool {
	mountpoint = strings.TrimRight(mountpoint, "/")
	targetPath = strings.TrimRight(targetPath, "/")
	if mountpoint == "" {
		mountpoint = UnifiedStorageMountpoint
	}
	if targetPath == "" {
		return false
	}
	if mountpoint == UnifiedStorageMountpoint {
		return strings.HasPrefix(targetPath, UnifiedStorageMountpoint)
	}
	return targetPath == mountpoint || strings.HasPrefix(targetPath, mountpoint+"/")
}

func metricMountpoint(metric components.StorageMetric) string {
	if metric.Mountpoint != "" {
		return metric.Mountpoint
	}
	return metric.MountPath
}

func usageSnapshotFromMetric(metric components.StorageMetric) UsageSnapshot {
	ratio := float64(*metric.UsedBytes) / float64(*metric.TotalBytes)
	if metric.UsedPercent != nil {
		ratio = *metric.UsedPercent / 100
	}
	mountpoint := metricMountpoint(metric)
	return UsageSnapshot{
		TargetType: TargetFilesystem, TargetID: filesystemTargetIDForMountpoint(mountpoint), Mountpoint: mountpoint,
		CapacityBytes: *metric.TotalBytes, UsedBytes: *metric.UsedBytes, UsedRatio: ratio,
		ObservedAt: valueTime(metric.CollectedAt), Available: true,
	}
}

func containerRootFilesystemFallback(paths []ProtectedPath, reason string) (UsageSnapshot, bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(UnifiedStorageMountpoint, &stat); err != nil {
		return UsageSnapshot{}, false
	}
	blockSize := uint64(stat.Bsize)
	total := stat.Blocks * blockSize
	available := stat.Bavail * blockSize
	if blockSize == 0 || total == 0 || available > total {
		return UsageSnapshot{}, false
	}
	protectedPaths := make([]string, 0, len(paths))
	for _, protectedPath := range paths {
		if protectedPath.Path != "" {
			protectedPaths = append(protectedPaths, protectedPath.Path)
		}
	}
	return UsageSnapshot{
		TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, Mountpoint: UnifiedStorageMountpoint,
		ProtectedPaths: uniqueStrings(protectedPaths),
		CapacityBytes:  total, UsedBytes: total - available, UsedRatio: float64(total-available) / float64(total),
		ObservedAt: time.Now().UTC(), Available: true,
		Reason: "host filesystem metrics unavailable; using container root filesystem sample: " + reason,
	}, true
}

func filesystemTargetIDForMountpoint(mountpoint string) string {
	if mountpoint == "" || mountpoint == UnifiedStorageMountpoint {
		return UnifiedStorageTargetID
	}
	id := strings.Trim(mountpoint, "/ ")
	replacer := strings.NewReplacer("/", "-", " ", "-", ":", "-")
	id = replacer.Replace(id)
	if id == "" {
		return UnifiedStorageTargetID
	}
	return "mount-" + id
}

func worstUsageSnapshot(targets []UsageSnapshot) UsageSnapshot {
	if len(targets) == 0 {
		return UsageSnapshot{TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, Mountpoint: UnifiedStorageMountpoint, Available: false, Reason: "no protected host filesystem mountpoints were found"}
	}
	worst := targets[0]
	for _, target := range targets[1:] {
		if !target.Available {
			return target
		}
		if target.UsedRatio > worst.UsedRatio {
			worst = target
		}
	}
	worst.TargetType = TargetFilesystem
	worst.TargetID = UnifiedStorageTargetID
	if worst.Mountpoint != "" {
		highestReason := "highest protected mountpoint usage: " + worst.Mountpoint
		if worst.Reason == "" {
			worst.Reason = highestReason
		} else {
			worst.Reason = highestReason + "; " + worst.Reason
		}
	}
	return worst
}

func uniqueStrings(values []string) []string {
	sort.Strings(values)
	result := values[:0]
	var previous string
	for i, value := range values {
		if i == 0 || value != previous {
			result = append(result, value)
			previous = value
		}
	}
	return result
}

func valueTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
