package stream

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
	"go.uber.org/zap"
)

const (
	publishedVersionRepairBatchSize         = uint64(100)
	publishedVersionRepairVersionQueryLimit = 8
	publishedVersionRepairLockID            = int64(0x504d565245504149) // "PMVREPAI"
)

type publishedVersionRepairCandidate struct {
	Window               WindowRecord
	VersionEffectiveFrom *time.Time
	VersionEffectiveTo   *time.Time
	AuditFingerprint     string
}

type publishedVersionRepairAction struct {
	Key                  WindowKey
	ExpectedSlots        int64
	VersionEffectiveFrom time.Time
	VersionEffectiveTo   *time.Time
	AuditFingerprint     string
	SourceEventID        string
	EnqueueRebuild       bool
}

type PublishedVersionRepairResult struct {
	Scanned                      int
	Audited                      int
	Enqueued                     int
	VersionQueries               int
	CompletedVersionFingerprints map[uuid.UUID]string
}

func normalizePublishedVersionRepairLimit(limit uint64) uint64 {
	if limit == 0 {
		return 1
	}
	return min(limit, publishedVersionRepairBatchSize)
}

func publishedVersionRepairRemaining(limit uint64, scanned int) uint64 {
	if scanned >= int(limit) {
		return 0
	}
	return limit - uint64(scanned)
}

func publishedVersionRepairBatchCompleted(
	candidateCount int,
	queryLimit uint64,
	allAudited bool,
) bool {
	return allAudited && candidateCount < int(queryLimit)
}

func publishedVersionAuditFingerprint(
	version *TaskVersionSnapshot,
	location *time.Location,
) string {
	if version == nil {
		return ""
	}
	if location == nil {
		location = time.UTC
	}
	effectiveTo := "open"
	if version.EffectiveTo != nil {
		effectiveTo = version.EffectiveTo.UTC().Format(time.RFC3339Nano)
	}
	payload := fmt.Sprintf(
		"v1|%s|%s|%s|%s",
		version.VersionID,
		version.EffectiveFrom.UTC().Format(time.RFC3339Nano),
		effectiveTo,
		location.String(),
	)
	return fmt.Sprintf("v1:%x", sha256.Sum256([]byte(payload)))
}

func planPublishedVersionRepair(
	candidate publishedVersionRepairCandidate,
	version *TaskVersionSnapshot,
	location *time.Location,
) (publishedVersionRepairAction, bool) {
	if version == nil || version.DevicePipeline || version.DeviceRollup ||
		candidate.Window.Status != "published" {
		return publishedVersionRepairAction{}, false
	}
	fingerprint := publishedVersionAuditFingerprint(version, location)
	if candidate.AuditFingerprint == fingerprint {
		return publishedVersionRepairAction{}, false
	}
	expected := expectedSlotsForFinalization(
		candidate.Window.Key,
		version,
		candidate.Window.ExpectedSlots,
		location,
	)
	metadataChanged := candidate.VersionEffectiveFrom == nil ||
		!candidate.VersionEffectiveFrom.Equal(version.EffectiveFrom) ||
		!sameOptionalTime(candidate.VersionEffectiveTo, version.EffectiveTo)
	return publishedVersionRepairAction{
		Key:                  candidate.Window.Key,
		ExpectedSlots:        expected,
		VersionEffectiveFrom: version.EffectiveFrom,
		VersionEffectiveTo:   version.EffectiveTo,
		AuditFingerprint:     fingerprint,
		SourceEventID:        "version-audit:" + fingerprint,
		EnqueueRebuild: candidate.Window.ExpectedSlots != expected ||
			metadataChanged,
	}, true
}

func publishedVersionRepairCandidates(
	version *TaskVersionSnapshot,
	fingerprint string,
	limit uint64,
) sq.SelectBuilder {
	limit = normalizePublishedVersionRepairLimit(limit)
	return storage.Psql.Select(
		"task_id", "task_version_id", "entity_key", "granularity",
		"window_start", "window_end", "status", "expected_slots",
		"received_slots", "finalize_attempts",
		"version_effective_from", "version_effective_to",
		"COALESCE(version_audit_fingerprint, '')",
	).
		From("pm_aggregation_windows").
		Where(sq.Eq{
			"task_version_id": version.VersionID,
			"status":          "published",
			"granularity": []string{
				string(GranularityDaily),
				string(GranularityWeekly),
				string(GranularityMonthly),
			},
		}).
		Where(sq.Or{
			sq.Eq{"version_audit_fingerprint": nil},
			sq.NotEq{"version_audit_fingerprint": fingerprint},
		}).
		OrderBy("window_start", "entity_key", "granularity").
		Limit(limit)
}

func publishedVersionRepairAuditUpdate(
	action publishedVersionRepairAction,
) sq.UpdateBuilder {
	builder := storage.Psql.Update("pm_aggregation_windows").
		Set("expected_slots", action.ExpectedSlots).
		Set("version_effective_from", action.VersionEffectiveFrom).
		Set("version_effective_to", nullablePtrTime(action.VersionEffectiveTo)).
		Set("version_audit_fingerprint", action.AuditFingerprint).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(windowKeyPredicate(action.Key)).
		Where(sq.Eq{"status": "published"}).
		Where(sq.Or{
			sq.Eq{"version_audit_fingerprint": nil},
			sq.NotEq{"version_audit_fingerprint": action.AuditFingerprint},
		})
	if action.EnqueueRebuild {
		builder = builder.Set("rebuild_requested_at", sq.Expr("CURRENT_TIMESTAMP"))
	}
	return builder
}

func publishedVersionRepairEnqueue(
	action publishedVersionRepairAction,
) sq.InsertBuilder {
	return storage.Psql.Insert("pm_aggregation_rebuilds").
		Columns(
			"task_id", "task_version_id", "entity_key", "granularity",
			"window_start", "window_end", "source_event_id",
		).
		Values(
			action.Key.TaskID,
			action.Key.TaskVersionID,
			action.Key.EntityKey,
			string(action.Key.Granularity),
			action.Key.Start,
			action.Key.End,
			action.SourceEventID,
		).
		Suffix(`
ON CONFLICT (task_version_id, entity_key, granularity, window_start) DO UPDATE SET
  source_event_id = EXCLUDED.source_event_id,
  window_end = EXCLUDED.window_end,
  requested_at = now(),
  request_generation = pm_aggregation_rebuilds.request_generation + 1,
  next_attempt_at = now(),
  completed_at = NULL,
  status = CASE
    WHEN pm_aggregation_rebuilds.status = 'completed' THEN 'pending'
    ELSE pm_aggregation_rebuilds.status
  END`)
}

func (r *WindowRepository) RepairPublishedVersionBatch(
	ctx context.Context,
	versions []*TaskVersionSnapshot,
	location *time.Location,
	limit uint64,
) (PublishedVersionRepairResult, error) {
	result := PublishedVersionRepairResult{
		CompletedVersionFingerprints: make(map[uuid.UUID]string),
	}
	versions = normalizePublishedVersionRepairVersions(versions)
	if len(versions) == 0 {
		return result, nil
	}
	limit = normalizePublishedVersionRepairLimit(limit)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin published PM version window repair: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var acquired bool
	if err := tx.QueryRow(
		ctx,
		"SELECT pg_try_advisory_xact_lock($1)",
		publishedVersionRepairLockID,
	).Scan(&acquired); err != nil {
		return result, fmt.Errorf("lock published PM version window repair: %w", err)
	}
	if !acquired {
		return result, nil
	}

	for index, version := range versions {
		remaining := publishedVersionRepairRemaining(limit, result.Scanned)
		if remaining == 0 {
			break
		}
		if version == nil || version.DevicePipeline || version.DeviceRollup ||
			version.EffectiveFrom.IsZero() {
			continue
		}
		versionsRemaining := uint64(len(versions) - index)
		versionLimit := (remaining + versionsRemaining - 1) / versionsRemaining
		fingerprint := publishedVersionAuditFingerprint(version, location)
		query, args, buildErr := publishedVersionRepairCandidates(
			version,
			fingerprint,
			versionLimit,
		).ToSql()
		if buildErr != nil {
			return result, fmt.Errorf("build published PM version repair candidates: %w", buildErr)
		}
		rows, queryErr := tx.Query(ctx, query, args...)
		if queryErr != nil {
			return result, fmt.Errorf("query published PM version repair candidates: %w", queryErr)
		}
		result.VersionQueries++
		var candidates []publishedVersionRepairCandidate
		for rows.Next() {
			var candidate publishedVersionRepairCandidate
			if scanErr := rows.Scan(
				&candidate.Window.Key.TaskID,
				&candidate.Window.Key.TaskVersionID,
				&candidate.Window.Key.EntityKey,
				&candidate.Window.Key.Granularity,
				&candidate.Window.Key.Start,
				&candidate.Window.Key.End,
				&candidate.Window.Status,
				&candidate.Window.ExpectedSlots,
				&candidate.Window.ReceivedSlots,
				&candidate.Window.FinalizeAttempts,
				&candidate.VersionEffectiveFrom,
				&candidate.VersionEffectiveTo,
				&candidate.AuditFingerprint,
			); scanErr != nil {
				rows.Close()
				return result, fmt.Errorf("scan published PM version repair candidate: %w", scanErr)
			}
			candidates = append(candidates, candidate)
		}
		rows.Close()
		if rows.Err() != nil {
			return result, fmt.Errorf("iterate published PM version repair candidates: %w", rows.Err())
		}
		result.Scanned += len(candidates)
		allAudited := true
		for _, candidate := range candidates {
			action, ok := planPublishedVersionRepair(candidate, version, location)
			if !ok {
				allAudited = false
				continue
			}
			updateSQL, updateArgs, buildErr := publishedVersionRepairAuditUpdate(action).ToSql()
			if buildErr != nil {
				return result, fmt.Errorf("build published PM version audit update: %w", buildErr)
			}
			tag, execErr := tx.Exec(ctx, updateSQL, updateArgs...)
			if execErr != nil {
				return result, fmt.Errorf("update published PM version audit: %w", execErr)
			}
			if tag.RowsAffected() != 1 {
				allAudited = false
				continue
			}
			result.Audited++
			if !action.EnqueueRebuild {
				continue
			}
			enqueueSQL, enqueueArgs, buildErr := publishedVersionRepairEnqueue(action).ToSql()
			if buildErr != nil {
				return result, fmt.Errorf("build published PM version repair rebuild: %w", buildErr)
			}
			if _, execErr := tx.Exec(ctx, enqueueSQL, enqueueArgs...); execErr != nil {
				return result, fmt.Errorf("enqueue published PM version repair rebuild: %w", execErr)
			}
			result.Enqueued++
		}
		if publishedVersionRepairBatchCompleted(
			len(candidates),
			versionLimit,
			allAudited,
		) {
			result.CompletedVersionFingerprints[version.VersionID] = fingerprint
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit published PM version window repair: %w", err)
	}
	return result, nil
}

func normalizePublishedVersionRepairVersions(
	versions []*TaskVersionSnapshot,
) []*TaskVersionSnapshot {
	if len(versions) <= publishedVersionRepairVersionQueryLimit {
		return versions
	}
	return versions[:publishedVersionRepairVersionQueryLimit]
}

type publishedVersionRepairBatchStore interface {
	RepairPublishedVersionBatch(
		context.Context,
		[]*TaskVersionSnapshot,
		*time.Location,
		uint64,
	) (PublishedVersionRepairResult, error)
}

type PublishedVersionRepairer struct {
	windows           publishedVersionRepairBatchStore
	snapshot          *SnapshotStore
	location          LocationProvider
	logger            *zap.Logger
	versionQueryLimit int
	cursor            uuid.UUID
	completed         map[uuid.UUID]string
}

func NewPublishedVersionRepairer(
	windows publishedVersionRepairBatchStore,
	snapshot *SnapshotStore,
	location *time.Location,
	logger *zap.Logger,
) *PublishedVersionRepairer {
	return NewPublishedVersionRepairerWithLocationProvider(
		windows, snapshot, fixedLocationProvider(location), logger,
	)
}

func NewPublishedVersionRepairerWithLocationProvider(
	windows publishedVersionRepairBatchStore,
	snapshot *SnapshotStore,
	location LocationProvider,
	logger *zap.Logger,
) *PublishedVersionRepairer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PublishedVersionRepairer{
		windows: windows, snapshot: snapshot, location: location, logger: logger,
		versionQueryLimit: publishedVersionRepairVersionQueryLimit,
		completed:         make(map[uuid.UUID]string),
	}
}

func (r *PublishedVersionRepairer) runOnce(
	ctx context.Context,
) (PublishedVersionRepairResult, error) {
	snapshot := r.snapshot.Current()
	r.pruneCompletedVersions(snapshot)
	location := currentLocation(r.location)
	versions := nextPublishedVersionRepairBatch(
		snapshot,
		location,
		r.completed,
		r.cursor,
		r.versionQueryLimit,
	)
	if len(versions) == 0 {
		return PublishedVersionRepairResult{}, nil
	}
	r.cursor = versions[len(versions)-1].VersionID
	result, err := r.windows.RepairPublishedVersionBatch(
		ctx,
		versions,
		location,
		publishedVersionRepairBatchSize,
	)
	if err != nil {
		return result, err
	}
	for versionID, fingerprint := range result.CompletedVersionFingerprints {
		r.completed[versionID] = fingerprint
	}
	return result, nil
}

func nextPublishedVersionRepairBatch(
	snapshot *TaskSnapshot,
	location *time.Location,
	completed map[uuid.UUID]string,
	after uuid.UUID,
	limit int,
) []*TaskVersionSnapshot {
	if snapshot == nil || limit <= 0 {
		return nil
	}
	var versions []*TaskVersionSnapshot
	for _, versionID := range sortedSnapshotVersionIDs(snapshot) {
		version := snapshot.ByVersion[versionID]
		if version == nil || version.DevicePipeline || version.DeviceRollup ||
			version.EffectiveFrom.IsZero() {
			continue
		}
		if completed[versionID] == publishedVersionAuditFingerprint(version, location) {
			continue
		}
		versions = append(versions, version)
	}
	if len(versions) == 0 {
		return nil
	}
	start := 0
	if after != uuid.Nil {
		start = len(versions)
		for index, version := range versions {
			if version.VersionID.String() > after.String() {
				start = index
				break
			}
		}
		if start == len(versions) {
			start = 0
		}
	}
	count := min(limit, len(versions))
	selected := make([]*TaskVersionSnapshot, 0, count)
	for offset := range count {
		selected = append(selected, versions[(start+offset)%len(versions)])
	}
	return selected
}

func (r *PublishedVersionRepairer) pruneCompletedVersions(snapshot *TaskSnapshot) {
	for versionID := range r.completed {
		version := snapshot.ByVersion[versionID]
		if version == nil || version.DevicePipeline || version.DeviceRollup ||
			version.EffectiveFrom.IsZero() {
			delete(r.completed, versionID)
		}
	}
}

func (r *PublishedVersionRepairer) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	runOnce := func() {
		result, err := r.runOnce(ctx)
		if err != nil {
			if ctx.Err() == nil {
				r.logger.Warn("repair published PM version windows", zap.Error(err))
			}
			return
		}
		if result.Audited > 0 {
			r.logger.Info("audited published PM version windows",
				zap.Int("scanned", result.Scanned),
				zap.Int("audited", result.Audited),
				zap.Int("rebuilds_enqueued", result.Enqueued),
				zap.Int("version_queries", result.VersionQueries))
		}
	}
	runOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
