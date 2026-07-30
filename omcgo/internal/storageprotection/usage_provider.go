package storageprotection

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/components"
)

// CollectorUsageProvider adapts the existing system-info Prometheus collector
// to the storage protection domain. It shares the collector cache, so the
// management page and admission checks observe the same source timestamp.
type CollectorUsageProvider struct {
	collector components.StorageCollector
}

func NewCollectorUsageProvider(collector components.StorageCollector) *CollectorUsageProvider {
	return &CollectorUsageProvider{collector: collector}
}

func (p *CollectorUsageProvider) Snapshot(ctx context.Context, targetType TargetType, targetID string) (UsageSnapshot, error) {
	if p == nil || p.collector == nil {
		return UsageSnapshot{TargetType: targetType, TargetID: targetID, Available: false, Reason: "storage collector is not configured"}, nil
	}
	for _, metric := range p.collector.Collect(ctx) {
		if !matchesTarget(metric, targetType, targetID) {
			continue
		}
		if metric.Status != "available" || metric.TotalBytes == nil || metric.UsedBytes == nil {
			return UsageSnapshot{TargetType: targetType, TargetID: targetID, Available: false, Reason: metric.Error}, nil
		}
		ratio := float64(*metric.UsedBytes) / float64(*metric.TotalBytes)
		if metric.UsedPercent != nil {
			ratio = *metric.UsedPercent / 100
		}
		return UsageSnapshot{
			TargetType: targetType, TargetID: targetID,
			CapacityBytes: *metric.TotalBytes, UsedBytes: *metric.UsedBytes,
			UsedRatio: ratio, ObservedAt: valueTime(metric.CollectedAt), Available: true,
		}, nil
	}
	return UsageSnapshot{TargetType: targetType, TargetID: targetID, Available: false, Reason: fmt.Sprintf("storage target %s/%s was not found", targetType, targetID)}, nil
}

func matchesTarget(metric components.StorageMetric, targetType TargetType, targetID string) bool {
	if targetType != TargetFilesystem || targetID != UnifiedStorageTargetID || metric.Kind != "host_filesystem" {
		return false
	}
	// A failed host query still returns a stable unavailable metric without a
	// mountpoint. Match it so the policy enters the configured unknown state;
	// available capacity must come from the physical host root mount, never the
	// duplicate application-visible statfs value or a logical MinIO/DB metric.
	if metric.Status != "available" {
		return metric.ID == "host-filesystem"
	}
	return metric.Mountpoint == UnifiedStorageMountpoint || metric.MountPath == UnifiedStorageMountpoint
}

func valueTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
