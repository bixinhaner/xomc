// Package backup — multi-device orphan file reaper (T-0083).
//
// Closes the long-tail orphan accumulation left by T-0079 first-write-wins
// + T-0076 file_path-based cleanup: when a backup_task targets N devices
// the executor only records the first uploaded file's path; the remaining
// N-1 files have no DB linkage and survive cleanup. The reaper scans the
// bucket weekly and physically removes objects whose embedded taskID8
// prefix is absent from the live backup_tasks set AND whose LastModified
// is older than the retention safety net (RetentionDays + 1 days).
//
// Why "live set + age safety net" rather than a new backup_files table:
//
//   - PRD §2.1 ULTRATHINK: scope control. multi-device backups are the
//     non-mainstream usage; orphan creation rate is low; @weekly bucket
//     scan absorbs it without schema migration.
//   - PRD §2.3 race avoidance: a brand-new upload that lands while the
//     scan is running is necessarily LastModified ≥ now-30d in the
//     opposite direction, so the age gate keeps it out of reach.
//
// Failure modes are fail-open: any error short-circuits the run and
// returns; partial reaps already committed stay reaped.
package backup

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// maxReapPerRun caps RemoveObject invocations per RunOrphanReaperOnce
// invocation so an unexpectedly large orphan population (e.g. the bucket
// just got a backlog of tens of thousands of multi-device backups) cannot
// flood MinIO with API calls in a single tick. The remainder is logged at
// info and picked up next @weekly tick.
//
// 1000/run × 52/year ≈ 52k reap ceiling/year, comfortably above the
// realistic multi-device orphan generation rate.
const maxReapPerRun = 1000

// TaskIDLister returns the set of 8-char hex prefixes of every live
// backup_tasks.id. PgTaskRepository implements this; tests may supply a
// minimal stub. Defined consumer-side so the existing TaskRepository mock
// fleet stays unchanged.
type TaskIDLister interface {
	ListAllTaskIDPrefixes(ctx context.Context) (map[string]struct{}, error)
}

// SetTaskIDLister wires the live-set query for T-0083. Pass nil to
// disable the reaper (preserves T-0073/T-0076/T-0082 behaviour). The
// reaper additionally requires SetMinIO + SetBucketLister to be wired —
// any of the three nil disables the run.
func (m *PolicyMonitor) SetTaskIDLister(l TaskIDLister) {
	m.taskIDLister = l
}

// backupFilenameRe extracts the {taskID8} hex prefix from a backup
// object key. Object keys are full bucket paths laid down by ACS upload
// handler as:
//
//	{category}/YYYY/MM/DD/{taskID8}/{SN}_CFG.{xml|nv}
//
// taskID8 is the directory segment immediately before the canonical
// filename. issue #585 dropped the legacy `backup-{taskID8}-{SN}.{ext}`
// naming entirely (no compression/.enc suffixes either) — historical
// objects under that scheme are not considered.
var backupFilenameRe = regexp.MustCompile(`(?:^|/)([0-9a-f]{8})/[^/]+_CFG\.(?:xml|nv)$`)

// parseBackupFilename returns the (lowercase) taskID8 prefix when the
// object key matches the canonical backup naming convention. ok=false
// signals "not our file, do not touch".
func parseBackupFilename(key string) (taskID8 string, ok bool) {
	mm := backupFilenameRe.FindStringSubmatch(key)
	if mm == nil {
		return "", false
	}
	return mm[1], true
}

// RunOrphanReaperOnce performs a single full-bucket scan + reap pass.
// Exposed for direct test invocation; the @weekly cron in Start() calls
// it on schedule. Returns the count of objects physically deleted.
//
// Disabled (returns 0, nil) when any of MinIO / BucketLister /
// TaskIDLister wiring is absent — preserves T-0073/T-0076/T-0082-only
// deployments without re-running the regex / list pipeline.
//
// Skip categorization (recorded as omc_backup_orphan_skipped_total):
//
//   - pattern_mismatch — object key does not match the canonical naming
//     (operator-uploaded files, unrelated objects, malformed paths).
//   - live_task        — taskID8 hits the live backup_tasks set.
//   - age_recent       — LastModified ≥ cutoff (RetentionDays+1 days).
//   - api_error        — RemoveObject returned an error.
//
// Cap behaviour: reap exits early once maxReapPerRun successful deletes
// accumulate; remaining candidates are picked up next tick.
func (m *PolicyMonitor) RunOrphanReaperOnce(ctx context.Context) (int, error) {
	if m.bucketLister == nil || m.minio == nil || m.taskIDLister == nil {
		return 0, nil
	}

	policy, err := m.policyService.Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("get backup policy for orphan reaper: %w", err)
	}

	liveSet, err := m.taskIDLister.ListAllTaskIDPrefixes(ctx)
	if err != nil {
		return 0, fmt.Errorf("list live backup_tasks prefixes: %w", err)
	}

	// Age safety net: only reap objects older than RetentionDays+1 days.
	// Reading from policy (rather than a hard-coded 30d) keeps the gate
	// aligned with whatever T-0073 cleanup retention the operator
	// configured — a deployment that bumped retention to 60d will see
	// the reaper push its own gate to 61d in lock-step.
	retention := policy.RetentionDays
	if retention < 0 {
		retention = 0
	}
	cutoff := time.Now().AddDate(0, 0, -(retention + 1))

	reaped := 0
	skipped := map[string]int{}

	for obj := range m.bucketLister.ListObjects(ctx, CanonicalRestoreBucket, minio.ListObjectsOptions{
		Recursive: true,
	}) {
		if obj.Err != nil {
			m.recordOrphanSkippedTallies(skipped)
			return reaped, fmt.Errorf("list backup bucket: %w", obj.Err)
		}
		if reaped >= maxReapPerRun {
			m.logger.Info("orphan reaper hit max reap per run; remainder pending next tick",
				zap.Int("max_reap_per_run", maxReapPerRun),
			)
			break
		}

		taskID8, ok := parseBackupFilename(obj.Key)
		if !ok {
			skipped["pattern_mismatch"]++
			continue
		}
		if _, isLive := liveSet[taskID8]; isLive {
			skipped["live_task"]++
			continue
		}
		if obj.LastModified.After(cutoff) {
			skipped["age_recent"]++
			continue
		}

		// Object key from MinIO ListObjects is just the in-bucket path
		// (no bucket prefix), so RemoveObject takes (bucket, obj.Key)
		// directly without splitBucketAndPath round-tripping.
		if err := m.minio.RemoveObject(ctx, CanonicalRestoreBucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			skipped["api_error"]++
			m.logger.Warn("orphan reaper RemoveObject failed; continuing scan",
				zap.String("object_key", obj.Key),
				zap.Error(err),
			)
			continue
		}
		reaped++
		m.metrics.RecordOrphanReaped()
	}

	m.recordOrphanSkippedTallies(skipped)
	m.logger.Info("orphan reaper completed",
		zap.Int("reaped", reaped),
		zap.Any("skipped", skipped),
		zap.Int("retention_days", policy.RetentionDays),
		zap.Time("cutoff", cutoff),
	)
	return reaped, nil
}

// recordOrphanSkippedTallies flushes the per-reason counts into the
// metric backend in one place so the main loop stays compact.
func (m *PolicyMonitor) recordOrphanSkippedTallies(skipped map[string]int) {
	for reason, n := range skipped {
		for i := 0; i < n; i++ {
			m.metrics.RecordOrphanSkipped(reason)
		}
	}
}
