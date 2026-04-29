// Package backup — storage threshold monitor (T-0082).
//
// Hourly poller that sums object sizes in the backup bucket and raises (or
// clears) an alarm when usage crosses BackupPolicy.AlertThresholdPercent of
// BackupPolicy.MaxStorageGB. Both fields are reused from T-0071 schema; no
// migration needed for this task.
//
// Edge-trigger semantics:
//
//	below→above ⇒ publish alarm.raised once
//	above→above ⇒ no alarm (skipped, "already known")
//	above→below ⇒ publish alarm.cleared once
//
// Process restart resets the in-memory edge state, which means the first
// post-restart tick may resend a raised alarm if usage is still above the
// threshold. This is acceptable because the F04 alarm engine deduplicates
// by (source, identifier) — and the alternative (DB-persisted state) would
// add a column for a problem with hourly granularity.
package backup

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// BucketLister is the narrow consumer-side contract PolicyMonitor needs to
// compute bucket usage. *minio.Client satisfies it; defined locally so test
// mocks stay tiny and the production import surface stays unchanged.
type BucketLister interface {
	ListObjects(ctx context.Context, bucket string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
}

// SetBucketLister wires the storage poller for T-0082. Pass nil to disable
// (preserves T-0073/T-0076 behaviour). Disk monitoring requires both this
// and SetEventBus to be wired; either nil disables the poller.
func (m *PolicyMonitor) SetBucketLister(l BucketLister) {
	m.bucketLister = l
}

// SetEventBus wires the alarm publisher for T-0082. Pass nil to disable disk
// alarms. Constructed separately to keep the existing 4-arg NewPolicyMonitor
// signature stable for the 8 pre-existing PolicyMonitor tests.
func (m *PolicyMonitor) SetEventBus(bus event.EventBus) {
	m.bus = bus
}

// RunStorageCheckOnce executes a single bucket usage poll + threshold check.
// Returns the observed used bytes (best-effort; 0 when disabled or skipped).
// Exposed for direct test invocation; the @hourly cron calls it on schedule.
//
// Failure modes are fail-open: list errors record metric and return without
// publishing an alarm. The monitor never panics on storage check errors.
func (m *PolicyMonitor) RunStorageCheckOnce(ctx context.Context) (int64, error) {
	if m.bucketLister == nil || m.bus == nil {
		// Storage monitoring not wired; preserves T-0073/T-0076 behaviour.
		return 0, nil
	}

	policy, err := m.policyService.Get(ctx)
	if err != nil {
		m.metrics.RecordStorageCheck("failure")
		return 0, fmt.Errorf("get backup policy: %w", err)
	}

	// Two opt-out gates:
	//   1. AlertOnFailure=false  — operator silenced all backup alarms.
	//   2. MaxStorageGB<=0        — capacity not configured; threshold math
	//      would divide by zero. Defensive guard against misconfiguration.
	//
	// Review M-1 followup (T-0084): future AlertSeverity column may also
	// introduce a separate DiskAlertEnabled toggle so disk monitoring can
	// stay on while per-task failure spam is suppressed. Until then disk
	// and per-task alarms share the AlertOnFailure kill-switch.
	if !policy.AlertOnFailure {
		m.metrics.RecordStorageCheck("skipped")
		return 0, nil
	}
	if policy.MaxStorageGB <= 0 {
		m.metrics.RecordStorageCheck("skipped")
		m.logger.Warn("MaxStorageGB=0; storage threshold disabled",
			zap.Int("max_storage_gb", policy.MaxStorageGB))
		return 0, nil
	}

	used, err := m.sumBucketUsage(ctx)
	if err != nil {
		// Review fix HIGH-2: record metric + log warn at the function level so
		// direct callers (tests, future ops endpoints) see context without
		// digging into the cron closure's outer warn. The error is still
		// returned for visibility — the @hourly cron closure logs and
		// continues; PRD §2.7 fail-open is achieved via "no false alarm + no
		// panic", not via swallowing the error.
		m.metrics.RecordStorageCheck("failure")
		m.logger.Warn("backup bucket list failed; skipping this storage check tick",
			zap.Error(err))
		return 0, fmt.Errorf("sum backup bucket usage: %w", err)
	}

	// Review fix MED-2: rearrange threshold math to (capacity/100)*pct to
	// avoid int64 overflow when capacity * pct could exceed 2^63 (theoretical
	// at ≈ 92 PB capacity). Precision loss is sub-100 bytes — irrelevant for
	// the threshold use case. capacity itself fits comfortably (int64 max ≈
	// 8 EB; MaxStorageGB validated >=1 has no practical upper bound but
	// real-world usage is GB-to-PB, all safe).
	capacity := int64(policy.MaxStorageGB) * 1024 * 1024 * 1024
	threshold := (capacity / 100) * int64(policy.AlertThresholdPercent)
	// Review fix HIGH-1: use >= rather than strict > so usage exactly at the
	// configured limit counts as "above". Operator mental model is "alert at
	// 80%", and strict > would mis-route a "cleared" event when usage settles
	// exactly on the threshold after a brief excursion above.
	above := used >= threshold

	m.metrics.RecordStorageCheck("success")
	m.metrics.SetStorageUsedBytes(used)
	m.metrics.SetStorageCapacityBytes(capacity)
	if capacity > 0 {
		m.metrics.SetStorageUsageRatio(float64(used) / float64(capacity))
	}

	// Edge-trigger detection (in-memory, sync.Mutex).
	m.storageMu.Lock()
	prev := m.lastAboveThreshold
	m.lastAboveThreshold = above
	m.storageMu.Unlock()

	info := StorageInfo{
		BucketName:       CanonicalRestoreBucket,
		UsedBytes:        used,
		CapacityBytes:    capacity,
		UsagePercent:     usagePercent(used, capacity),
		ThresholdPercent: policy.AlertThresholdPercent,
		AlertEmail:       policy.AlertEmail,
	}

	switch {
	case above && !prev:
		m.logger.Warn("backup storage threshold crossed; publishing alarm.raised",
			zap.Int("usage_percent", info.UsagePercent),
			zap.Int("threshold_percent", info.ThresholdPercent),
			zap.Int64("used_bytes", used),
			zap.Int64("capacity_bytes", capacity),
		)
		return used, PublishStorageThresholdAlarm(ctx, m.bus, m.metrics, TransitionRaised, info)
	case !above && prev:
		m.logger.Info("backup storage threshold relieved; publishing alarm.cleared",
			zap.Int("usage_percent", info.UsagePercent),
			zap.Int("threshold_percent", info.ThresholdPercent),
		)
		return used, PublishStorageThresholdAlarm(ctx, m.bus, m.metrics, TransitionCleared, info)
	default:
		// Already-known state (above→above or below→below): no event, just
		// metric so dashboards can show "alive but quiet".
		m.metrics.RecordStorageThresholdAlarm("skipped")
		return used, nil
	}
}

// sumBucketUsage iterates the canonical backup bucket and returns the total
// of ObjectInfo.Size across all objects. Errors short-circuit to fail-open.
func (m *PolicyMonitor) sumBucketUsage(ctx context.Context) (int64, error) {
	var used int64
	for obj := range m.bucketLister.ListObjects(ctx, CanonicalRestoreBucket, minio.ListObjectsOptions{
		Recursive: true,
	}) {
		if obj.Err != nil {
			return 0, obj.Err
		}
		used += obj.Size
	}
	return used, nil
}

// usagePercent returns int(used/capacity*100). Capacity 0 returns 0 to
// avoid NaN. Values can exceed 100 — over-capacity is a real condition the
// alarm summary needs to surface, so this deliberately does not clamp.
func usagePercent(used, capacity int64) int {
	if capacity <= 0 {
		return 0
	}
	return int(used * 100 / capacity)
}
