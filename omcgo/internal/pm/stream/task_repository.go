package stream

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgTaskRepository struct {
	pool *pgxpool.Pool
	bus  event.EventBus
}

// PgProgressTaskLoader serves read-only progress consumers. Progress results
// need version definitions and metric rules, but never device membership.
type PgProgressTaskLoader struct {
	repository     *PgTaskRepository
	includeMembers bool
}

var obsoleteBuiltinDeviceTaskIDs = []uuid.UUID{
	uuid.MustParse("0184dddd-0005-4000-8000-000000000001"),
	uuid.MustParse("0184dddd-0005-4000-8000-000000000002"),
	uuid.MustParse("0184dddd-0005-4000-8000-000000000003"),
}

func NewPgTaskRepository(pool *pgxpool.Pool, bus event.EventBus) *PgTaskRepository {
	return &PgTaskRepository{pool: pool, bus: bus}
}

func NewPgProgressTaskLoader(pool *pgxpool.Pool) *PgProgressTaskLoader {
	return &PgProgressTaskLoader{
		repository: NewPgTaskRepository(pool, nil), includeMembers: false,
	}
}

// PurgeObsoleteBuiltinDeviceTasks retires the three task rows used by the
// retired hidden-device implementation while preserving immutable history.
func (r *PgTaskRepository) PurgeObsoleteBuiltinDeviceTasks(ctx context.Context) (int, error) {
	effectiveTo := time.Now().UTC().Truncate(slotDuration).Add(slotDuration)
	count, err := r.retireTaskIDs(ctx, obsoleteBuiltinDeviceTaskIDs, effectiveTo)
	if err != nil {
		return 0, fmt.Errorf("retire obsolete PM device tasks: %w", err)
	}
	return count, nil
}

func (r *PgTaskRepository) Save(ctx context.Context, req SaveTaskRequest) (*TaskVersionSnapshot, error) {
	if err := validateSaveTask(req); err != nil {
		return nil, err
	}
	contentHash := taskContentHash(req)
	now := req.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	effectiveFrom := saveEffectiveFrom(req, now)
	plannedEndAt := nullablePtrTime(req.PlannedEndAt)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin save PM aggregation task: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskID := req.TaskID
	if taskID == uuid.Nil {
		taskID = uuid.New()
	}
	insertSQL, insertArgs, buildErr := buildUpsertTaskSQL(taskID, req, plannedEndAt)
	if buildErr != nil {
		return nil, fmt.Errorf("build create PM aggregation task SQL: %w", buildErr)
	}
	if _, err := tx.Exec(ctx, insertSQL, insertArgs...); err != nil {
		return nil, fmt.Errorf("create PM aggregation task: %w", err)
	}
	lockSQL, lockArgs, buildErr := storage.Psql.Select("id", "current_version_id", "planned_end_at").
		From("pm_aggregation_tasks").
		Where(sq.Eq{"id": taskID, "deleted_at": nil}).
		Suffix("FOR UPDATE").
		ToSql()
	if buildErr != nil {
		return nil, fmt.Errorf("build lock PM aggregation task SQL: %w", buildErr)
	}
	var locked uuid.UUID
	var currentVersionID *uuid.UUID
	var currentPlannedEndAt *time.Time
	if err := tx.QueryRow(ctx, lockSQL, lockArgs...).Scan(&locked, &currentVersionID, &currentPlannedEndAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("PM aggregation task %s is deleted", taskID)
		}
		return nil, fmt.Errorf("lock PM aggregation task: %w", err)
	}

	updateSQL, updateArgs, buildErr := buildUpdateTaskMetadataSQL(
		taskID, req, plannedEndAt,
	)
	if buildErr != nil {
		return nil, fmt.Errorf("build update PM aggregation task SQL: %w", buildErr)
	}
	if _, err := tx.Exec(ctx, updateSQL, updateArgs...); err != nil {
		return nil, fmt.Errorf("update PM aggregation task: %w", err)
	}

	if currentVersionID != nil {
		var currentHash []byte
		var currentVersionNo int
		var currentEffectiveFrom time.Time
		hashSQL, hashArgs, buildErr := storage.Psql.Select(
			"version_no", "effective_from", "content_hash",
		).From("pm_aggregation_task_versions").
			Where(sq.Eq{"id": *currentVersionID, "task_id": taskID}).
			ToSql()
		if buildErr != nil {
			return nil, fmt.Errorf("build load current PM aggregation content hash SQL: %w", buildErr)
		}
		if err := tx.QueryRow(ctx, hashSQL, hashArgs...).Scan(
			&currentVersionNo, &currentEffectiveFrom, &currentHash,
		); err != nil {
			return nil, fmt.Errorf("load current PM aggregation content hash: %w", err)
		}
		if len(currentHash) > 0 && bytes.Equal(currentHash, contentHash) {
			if shouldAdjustEffectiveFrom(req, currentVersionNo, currentEffectiveFrom, effectiveFrom) {
				adjustSQL, adjustArgs, buildErr := storage.Psql.Update("pm_aggregation_task_versions").
					Set("effective_from", effectiveFrom).
					Where(sq.Eq{"id": *currentVersionID, "task_id": taskID}).
					ToSql()
				if buildErr != nil {
					return nil, fmt.Errorf("build adjust PM aggregation effective time SQL: %w", buildErr)
				}
				if _, err := tx.Exec(ctx, adjustSQL, adjustArgs...); err != nil {
					return nil, fmt.Errorf("adjust PM aggregation effective time: %w", err)
				}
				currentEffectiveFrom = effectiveFrom
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("commit unchanged PM aggregation task: %w", err)
			}
			snapshot := snapshotFromRequest(
				req, taskID, *currentVersionID, currentVersionNo, currentEffectiveFrom,
			)
			snapshot.NewVersion = false
			if !sameOptionalTime(currentPlannedEndAt, req.PlannedEndAt) {
				r.publishTaskVersionChanged(ctx, taskID, *currentVersionID, currentEffectiveFrom)
			}
			return snapshot, nil
		}
	}

	var versionNo int
	versionSQL, versionArgs, err := storage.Psql.Select("COALESCE(MAX(version_no), 0) + 1").
		From("pm_aggregation_task_versions").
		Where(sq.Eq{"task_id": taskID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build next task version SQL: %w", err)
	}
	if err := tx.QueryRow(ctx, versionSQL, versionArgs...).Scan(&versionNo); err != nil {
		return nil, fmt.Errorf("load next PM aggregation task version: %w", err)
	}
	closeSQL, closeArgs, err := storage.Psql.Update("pm_aggregation_task_versions").
		Set("effective_to", effectiveFrom).
		Where(sq.Eq{"task_id": taskID, "effective_to": nil}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build close task version SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, closeSQL, closeArgs...); err != nil {
		return nil, fmt.Errorf("close current PM aggregation task version: %w", err)
	}

	versionID := uuid.New()
	granularities := make([]string, 0, len(req.Granularities))
	for _, granularity := range req.Granularities {
		granularities = append(granularities, string(granularity))
	}
	objectLDNs := req.ObjectLDNs
	if objectLDNs == nil {
		objectLDNs = []string{}
	}
	versionInsert, versionInsertArgs, err := storage.Psql.Insert("pm_aggregation_task_versions").
		Columns(
			"id", "task_id", "version_no", "enabled", "effective_from", "technology",
			"dimension", "granularities", "object_ldns", "created_by", "content_hash",
		).
		Values(
			versionID, taskID, versionNo, req.Enabled, effectiveFrom, nullableString(req.Technology),
			string(req.Dimension), granularities, objectLDNs, req.Creator, contentHash,
		).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert PM aggregation task version SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, versionInsert, versionInsertArgs...); err != nil {
		return nil, fmt.Errorf("insert PM aggregation task version: %w", err)
	}
	if err := insertMetricRules(ctx, tx, versionID, req.Metrics); err != nil {
		return nil, err
	}
	if err := insertCounterRules(ctx, tx, versionID, req.Counters); err != nil {
		return nil, err
	}
	if err := insertTaskMembers(ctx, tx, versionID, req.Members); err != nil {
		return nil, err
	}
	currentSQL, currentArgs, err := storage.Psql.Update("pm_aggregation_tasks").
		Set("current_version_id", versionID).
		Where(sq.Eq{"id": taskID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build set current PM aggregation version SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, currentSQL, currentArgs...); err != nil {
		return nil, fmt.Errorf("set current PM aggregation task version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit PM aggregation task version: %w", err)
	}

	snapshot := snapshotFromRequest(req, taskID, versionID, versionNo, effectiveFrom)
	snapshot.NewVersion = true
	r.publishTaskVersionChanged(ctx, taskID, versionID, effectiveFrom)
	return snapshot, nil
}

func buildUpsertTaskSQL(
	taskID uuid.UUID,
	req SaveTaskRequest,
	plannedEndAt any,
) (string, []interface{}, error) {
	return storage.Psql.Insert("pm_aggregation_tasks").
		Columns(
			"id", "name", "enabled", "visibility", "creator", "planned_end_at", "source_updated_at",
		).
		Values(
			taskID, req.Name, req.Enabled, req.Visibility, req.Creator,
			plannedEndAt, nullablePtrTime(&req.SourceUpdatedAt),
		).
		Suffix(`ON CONFLICT (id) DO UPDATE SET
deleted_at = NULL,
current_version_id = NULL
WHERE pm_aggregation_tasks.deleted_at IS NOT NULL`).
		ToSql()
}

func buildUpdateTaskMetadataSQL(
	taskID uuid.UUID,
	req SaveTaskRequest,
	plannedEndAt any,
) (string, []interface{}, error) {
	sourceUpdatedAt := nullablePtrTime(&req.SourceUpdatedAt)
	return storage.Psql.Update("pm_aggregation_tasks").
		Set("name", req.Name).
		Set("enabled", req.Enabled).
		Set("visibility", req.Visibility).
		Set("planned_end_at", plannedEndAt).
		Set("source_updated_at", sourceUpdatedAt).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": taskID}).
		Where(
			"(name IS DISTINCT FROM ? OR enabled IS DISTINCT FROM ? OR "+
				"visibility IS DISTINCT FROM ? OR planned_end_at IS DISTINCT FROM ? OR "+
				"source_updated_at IS DISTINCT FROM ?)",
			req.Name, req.Enabled, req.Visibility, plannedEndAt, sourceUpdatedAt,
		).
		ToSql()
}

func (r *PgTaskRepository) publishTaskVersionChanged(ctx context.Context, taskID, versionID uuid.UUID, effectiveFrom time.Time) {
	if r.bus == nil {
		return
	}
	payload := event.PMAggregationTaskVersionChangedPayload{
		TaskID: taskID, TaskVersionID: versionID, EffectiveFrom: effectiveFrom,
	}
	if evt, eventErr := event.NewEvent(event.SubjectPMAggregationTaskVersionChanged, payload); eventErr == nil {
		_ = r.bus.Publish(ctx, event.SubjectPMAggregationTaskVersionChanged, evt)
	}
}

func saveEffectiveFrom(req SaveTaskRequest, now time.Time) time.Time {
	if !req.EffectiveFrom.IsZero() {
		return req.EffectiveFrom.UTC()
	}
	return now.UTC().Truncate(time.Hour).Add(time.Hour)
}

func nullablePtrTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC()
}

func sameOptionalTime(a, b *time.Time) bool {
	if a == nil || a.IsZero() {
		return b == nil || b.IsZero()
	}
	if b == nil || b.IsZero() {
		return false
	}
	return a.UTC().Equal(b.UTC())
}

func shouldAdjustEffectiveFrom(
	req SaveTaskRequest,
	currentVersionNo int,
	currentEffectiveFrom time.Time,
	targetEffectiveFrom time.Time,
) bool {
	return !req.EffectiveFrom.IsZero() &&
		currentVersionNo == 1 &&
		currentEffectiveFrom.After(targetEffectiveFrom)
}

func (r *PgTaskRepository) Delete(ctx context.Context, taskID uuid.UUID) error {
	effectiveTo := time.Now().UTC().Truncate(slotDuration).Add(slotDuration)
	count, err := r.retireTaskIDs(ctx, []uuid.UUID{taskID}, effectiveTo)
	if err != nil {
		return err
	}
	if count == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PgTaskRepository) RetireMissingSourceTasks(
	ctx context.Context,
	taskSubtype string,
	mode string,
	limit uint64,
) (int, error) {
	if limit == 0 {
		limit = 200
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin retire missing-source PM aggregation tasks: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := buildMissingSourceTaskCandidatesSQL(taskSubtype, mode, limit)
	if err != nil {
		return 0, fmt.Errorf("build missing-source PM aggregation task candidates SQL: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("query missing-source PM aggregation task candidates: %w", err)
	}
	missing := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan missing-source PM aggregation task: %w", err)
		}
		missing = append(missing, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate missing-source PM aggregation tasks: %w", err)
	}
	rows.Close()
	effectiveTo := time.Now().UTC().Truncate(slotDuration).Add(slotDuration)
	count, err := retireTaskIDsTx(ctx, tx, missing, effectiveTo)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit retire missing-source PM aggregation tasks: %w", err)
	}
	for _, id := range missing {
		r.publishTaskVersionChanged(ctx, id, uuid.Nil, effectiveTo)
	}
	return count, nil
}

func buildMissingSourceTaskCandidatesSQL(
	taskSubtype string,
	mode string,
	limit uint64,
) (string, []interface{}, error) {
	candidates := storage.Psql.Select("id").
		From("pm_aggregation_tasks").
		Where("deleted_at IS NULL").
		Where(`NOT EXISTS (
SELECT 1 FROM pm_tasks source
WHERE source.id = pm_aggregation_tasks.id
  AND source.task_subtype = ?
  AND source.mode = ?
)`, taskSubtype, mode).
		OrderBy("id").
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED")
	return candidates.ToSql()
}

func (r *PgTaskRepository) retireTaskIDs(
	ctx context.Context,
	taskIDs []uuid.UUID,
	effectiveTo time.Time,
) (int, error) {
	ids := normalizeTaskIDs(taskIDs)
	if len(ids) == 0 {
		return 0, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin retire PM aggregation tasks: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	count, err := retireTaskIDsTx(ctx, tx, ids, effectiveTo)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit retire PM aggregation tasks: %w", err)
	}
	if count > 0 {
		for _, id := range ids {
			r.publishTaskVersionChanged(ctx, id, uuid.Nil, effectiveTo)
		}
	}
	return count, nil
}

func retireTaskIDsTx(
	ctx context.Context,
	tx pgx.Tx,
	taskIDs []uuid.UUID,
	effectiveTo time.Time,
) (int, error) {
	if len(taskIDs) == 0 {
		return 0, nil
	}
	query, args, err := buildRetireTasksSQL(taskIDs)
	if err != nil {
		return 0, fmt.Errorf("build retire PM aggregation tasks SQL: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("retire PM aggregation tasks: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return 0, nil
	}
	closeSQL, closeArgs, err := buildCloseRetiredTaskVersionsSQL(taskIDs, effectiveTo)
	if err != nil {
		return 0, fmt.Errorf("build close retired PM aggregation task versions SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, closeSQL, closeArgs...); err != nil {
		return 0, fmt.Errorf("close retired PM aggregation task versions: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func buildRetireTasksSQL(taskIDs []uuid.UUID) (string, []interface{}, error) {
	return storage.Psql.Update("pm_aggregation_tasks").
		Set("enabled", false).
		Set("deleted_at", sq.Expr("CURRENT_TIMESTAMP")).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": taskIDs}).
		Where("deleted_at IS NULL").
		ToSql()
}

func buildCloseRetiredTaskVersionsSQL(
	taskIDs []uuid.UUID,
	effectiveTo time.Time,
) (string, []interface{}, error) {
	return storage.Psql.Update("pm_aggregation_task_versions").
		Set("effective_to", sq.Expr("GREATEST(effective_from, ?)", effectiveTo)).
		Where(sq.Eq{"task_id": taskIDs, "effective_to": nil}).
		ToSql()
}

func insertMetricRules(ctx context.Context, tx pgx.Tx, versionID uuid.UUID, rules []MetricRule) error {
	builder := storage.Psql.Insert("pm_aggregation_version_metrics").
		Columns(
			"task_version_id", "metric_id", "metric_path", "metric_type",
			"aggregation_op", "formula", "dependencies",
		)
	for _, rule := range rules {
		metricID := rule.MetricID
		if metricID == "" {
			metricID = rule.MetricPath
		}
		builder = builder.Values(
			versionID, metricID, rule.MetricPath, rule.MetricType,
			string(rule.Aggregation), rule.Formula, rule.Dependencies,
		)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert PM aggregation metric rules SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert PM aggregation metric rules: %w", err)
	}
	return nil
}

func insertCounterRules(ctx context.Context, tx pgx.Tx, versionID uuid.UUID, rules []CounterRule) error {
	builder := storage.Psql.Insert("pm_aggregation_version_counters").
		Columns("task_version_id", "metric_path", "aggregation_op")
	for _, rule := range rules {
		builder = builder.Values(versionID, rule.MetricPath, string(rule.Aggregation))
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert PM aggregation counter rules SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert PM aggregation counter rules: %w", err)
	}
	return nil
}

const (
	taskMemberInsertBatchSize = 1000
	taskMemberColumnCount     = 6
)

func taskMemberBatches(members []TaskMember) [][]TaskMember {
	if len(members) == 0 {
		return nil
	}
	batches := make([][]TaskMember, 0, (len(members)+taskMemberInsertBatchSize-1)/taskMemberInsertBatchSize)
	for start := 0; start < len(members); start += taskMemberInsertBatchSize {
		end := min(start+taskMemberInsertBatchSize, len(members))
		batches = append(batches, members[start:end])
	}
	return batches
}

func insertTaskMembers(ctx context.Context, tx pgx.Tx, versionID uuid.UUID, members []TaskMember) error {
	for batchIndex, batch := range taskMemberBatches(members) {
		builder := storage.Psql.Insert("pm_aggregation_version_members").
			Columns(
				"task_version_id", "device_id", "device_sn",
				"dimension_key", "dimension_name", "object_ldn",
			)
		for _, member := range batch {
			builder = builder.Values(
				versionID, member.DeviceID, member.DeviceSN,
				member.DimensionKey, member.DimensionName, member.ObjectLDN,
			)
		}
		query, args, err := builder.ToSql()
		if err != nil {
			return fmt.Errorf("build insert PM aggregation members SQL batch %d: %w", batchIndex+1, err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert PM aggregation members batch %d: %w", batchIndex+1, err)
		}
	}
	return nil
}

func buildLoadMatchableRevisionSQL() (string, []interface{}, error) {
	semanticRow := `jsonb_build_array(
  t.id::text,
  t.name,
  t.enabled::text,
  t.visibility,
  COALESCE(t.current_version_id::text, ''),
  COALESCE(EXTRACT(EPOCH FROM t.planned_end_at)::text, ''),
  COALESCE(EXTRACT(EPOCH FROM t.deleted_at)::text, ''),
  COALESCE(EXTRACT(EPOCH FROM v.effective_from)::text, ''),
  COALESCE(EXTRACT(EPOCH FROM v.effective_to)::text, ''),
  COALESCE(encode(v.content_hash, 'hex'), '')
)::text`
	return storage.Psql.Select(
		"COUNT(*)",
		"COALESCE(md5(string_agg("+semanticRow+", ',' ORDER BY t.id)), md5(''))",
	).From("pm_aggregation_tasks t").
		LeftJoin("pm_aggregation_task_versions v ON v.id = t.current_version_id").
		ToSql()
}

func (r *PgTaskRepository) LoadMatchableRevision(
	ctx context.Context,
) (MatchableRevision, error) {
	query, args, err := buildLoadMatchableRevisionSQL()
	if err != nil {
		return MatchableRevision{}, fmt.Errorf(
			"build load PM aggregation task revision SQL: %w", err,
		)
	}
	var revision MatchableRevision
	if err := r.pool.QueryRow(ctx, query, args...).Scan(
		&revision.TaskCount,
		&revision.Fingerprint,
	); err != nil {
		return MatchableRevision{}, fmt.Errorf(
			"query PM aggregation task revision: %w", err,
		)
	}
	return revision, nil
}

func (r *PgTaskRepository) LoadMatchable(ctx context.Context, at time.Time) ([]*TaskVersionSnapshot, error) {
	return r.loadMatchable(ctx, at, true)
}

func (r *PgProgressTaskLoader) LoadMatchable(
	ctx context.Context,
	at time.Time,
) ([]*TaskVersionSnapshot, error) {
	return r.repository.loadMatchable(ctx, at, r.includeMembers)
}

func (r *PgProgressTaskLoader) LoadMatchableRevision(
	ctx context.Context,
) (MatchableRevision, error) {
	return r.repository.LoadMatchableRevision(ctx)
}

// LoadVersionsByID loads the persisted definitions for the requested task
// versions without device membership. Read-only consumers use this path when
// result rows already identify the exact immutable versions they need.
func (r *PgProgressTaskLoader) LoadVersionsByID(
	ctx context.Context,
	versionIDs []uuid.UUID,
) ([]*TaskVersionSnapshot, error) {
	if len(versionIDs) == 0 {
		return nil, nil
	}
	ids := normalizeTaskVersionIDs(versionIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := buildLoadTaskVersionsByIDSQL(ids)
	if err != nil {
		return nil, fmt.Errorf("build load PM aggregation task versions by ID SQL: %w", err)
	}
	return r.repository.loadTaskVersions(ctx, query, args, false)
}

func (r *PgTaskRepository) LoadRecoveryVersionStates(
	ctx context.Context,
	versionIDs []uuid.UUID,
) (map[uuid.UUID]RecoveryVersionState, error) {
	ids := normalizeTaskVersionIDs(versionIDs)
	states := make(map[uuid.UUID]RecoveryVersionState, len(ids))
	if len(ids) == 0 {
		return states, nil
	}
	query, args, err := storage.Psql.Select(
		"t.id", "t.enabled", "t.deleted_at", "v.id", "v.enabled",
		"v.effective_from", "v.effective_to", "t.planned_end_at",
	).From("pm_aggregation_task_versions v").
		Join("pm_aggregation_tasks t ON t.id = v.task_id").
		Where(sq.Eq{"v.id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build load PM recovery version states SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM recovery version states: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var state RecoveryVersionState
		if err := rows.Scan(
			&state.TaskID, &state.TaskEnabled, &state.TaskDeletedAt,
			&state.VersionID, &state.VersionEnabled, &state.EffectiveFrom,
			&state.EffectiveTo, &state.PlannedEndAt,
		); err != nil {
			return nil, fmt.Errorf("scan PM recovery version state: %w", err)
		}
		states[state.VersionID] = state
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM recovery version states: %w", err)
	}
	return states, nil
}

func buildLoadTaskVersionsByIDSQL(versionIDs []uuid.UUID) (string, []interface{}, error) {
	return taskVersionSelect().
		Where(sq.Eq{"v.id": versionIDs}).
		OrderBy("v.task_id", "v.version_no").
		ToSql()
}

func (r *PgTaskRepository) loadMatchable(
	ctx context.Context,
	at time.Time,
	includeMembers bool,
) ([]*TaskVersionSnapshot, error) {
	query, args, err := taskVersionSelect().
		Where(sq.Or{
			sq.Eq{"v.effective_to": nil},
			sq.GtOrEq{"v.effective_to": at.Add(-45 * 24 * time.Hour)},
		}).
		OrderBy("v.task_id", "v.version_no").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build load matchable task versions SQL: %w", err)
	}
	return r.loadTaskVersions(ctx, query, args, includeMembers)
}

func taskVersionSelect() sq.SelectBuilder {
	return storage.Psql.Select(
		"t.id", "v.id", "v.version_no", "t.name", "t.enabled", "t.deleted_at", "v.enabled",
		"COALESCE(v.technology, '')", "v.dimension", "v.granularities",
		"v.object_ldns", "v.effective_from",
		"(SELECT MIN(lineage.effective_from) FROM pm_aggregation_task_versions lineage WHERE lineage.task_id = v.task_id)",
		"v.effective_to", "t.planned_end_at",
	).From("pm_aggregation_task_versions v").
		Join("pm_aggregation_tasks t ON t.id = v.task_id")
}

func (r *PgTaskRepository) loadTaskVersions(
	ctx context.Context,
	query string,
	args []interface{},
	includeMembers bool,
) ([]*TaskVersionSnapshot, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM aggregation task versions: %w", err)
	}
	var versions []*TaskVersionSnapshot
	for rows.Next() {
		var version TaskVersionSnapshot
		var granularityStrings []string
		var objectLDNs []string
		if err := rows.Scan(
			&version.TaskID, &version.VersionID, &version.VersionNo, &version.Name,
			&version.TaskEnabled, &version.TaskDeletedAt, &version.Enabled,
			&version.Technology, &version.Dimension, &granularityStrings, &objectLDNs,
			&version.EffectiveFrom, &version.LineageEffectiveFrom,
			&version.EffectiveTo, &version.PlannedEndAt,
		); err != nil {
			return nil, fmt.Errorf("scan matchable PM aggregation task version: %w", err)
		}
		version.Metrics = make(map[string]MetricRule)
		version.Counters = make(map[string]CounterRule)
		version.Members = make(map[uuid.UUID][]TaskMember)
		version.ObjectLDNs = make(map[string]struct{}, len(objectLDNs))
		for _, value := range granularityStrings {
			version.Granularities = append(version.Granularities, Granularity(value))
		}
		for _, value := range objectLDNs {
			version.ObjectLDNs[value] = struct{}{}
		}
		versions = append(versions, &version)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate matchable PM aggregation task versions: %w", err)
	}
	rows.Close()
	if err := r.loadDetails(ctx, versions, includeMembers); err != nil {
		return nil, err
	}
	return versions, nil
}

func normalizeTaskVersionIDs(versionIDs []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(versionIDs))
	ids := make([]uuid.UUID, 0, len(versionIDs))
	for _, versionID := range versionIDs {
		if versionID == uuid.Nil {
			continue
		}
		if _, exists := seen[versionID]; exists {
			continue
		}
		seen[versionID] = struct{}{}
		ids = append(ids, versionID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

func normalizeTaskIDs(taskIDs []uuid.UUID) []uuid.UUID {
	return normalizeTaskVersionIDs(taskIDs)
}

func (r *PgTaskRepository) loadDetails(
	ctx context.Context,
	versions []*TaskVersionSnapshot,
	includeMembers bool,
) error {
	if len(versions) == 0 {
		return nil
	}
	byID := make(map[uuid.UUID]*TaskVersionSnapshot, len(versions))
	ids := make([]uuid.UUID, 0, len(versions))
	for _, version := range versions {
		byID[version.VersionID] = version
		ids = append(ids, version.VersionID)
	}
	ruleSQL, ruleArgs, err := storage.Psql.Select(
		"task_version_id", "metric_id", "metric_path", "metric_type",
		"aggregation_op", "formula", "dependencies",
	).From("pm_aggregation_version_metrics").
		Where(sq.Eq{"task_version_id": ids}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build load PM aggregation rules SQL: %w", err)
	}
	ruleRows, err := r.pool.Query(ctx, ruleSQL, ruleArgs...)
	if err != nil {
		return fmt.Errorf("query PM aggregation rules: %w", err)
	}
	for ruleRows.Next() {
		var versionID uuid.UUID
		var rule MetricRule
		if err := ruleRows.Scan(
			&versionID, &rule.MetricID, &rule.MetricPath, &rule.MetricType,
			&rule.Aggregation, &rule.Formula, &rule.Dependencies,
		); err != nil {
			ruleRows.Close()
			return fmt.Errorf("scan PM aggregation rule: %w", err)
		}
		if version := byID[versionID]; version != nil {
			version.Metrics[rule.MetricPath] = rule
		}
	}
	if err := ruleRows.Err(); err != nil {
		ruleRows.Close()
		return fmt.Errorf("iterate PM aggregation rules: %w", err)
	}
	ruleRows.Close()

	counterSQL, counterArgs, err := storage.Psql.Select(
		"task_version_id", "metric_path", "aggregation_op",
	).From("pm_aggregation_version_counters").
		Where(sq.Eq{"task_version_id": ids}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build load PM aggregation counter rules SQL: %w", err)
	}
	counterRows, err := r.pool.Query(ctx, counterSQL, counterArgs...)
	if err != nil {
		return fmt.Errorf("query PM aggregation counter rules: %w", err)
	}
	for counterRows.Next() {
		var versionID uuid.UUID
		var rule CounterRule
		if err := counterRows.Scan(&versionID, &rule.MetricPath, &rule.Aggregation); err != nil {
			counterRows.Close()
			return fmt.Errorf("scan PM aggregation counter rule: %w", err)
		}
		if version := byID[versionID]; version != nil {
			version.Counters[rule.MetricPath] = rule
		}
	}
	if err := counterRows.Err(); err != nil {
		counterRows.Close()
		return fmt.Errorf("iterate PM aggregation counter rules: %w", err)
	}
	counterRows.Close()
	if !includeMembers {
		return nil
	}

	memberSQL, memberArgs, err := storage.Psql.Select(
		"task_version_id", "device_id", "device_sn",
		"dimension_key", "dimension_name", "object_ldn",
	).From("pm_aggregation_version_members").
		Where(sq.Eq{"task_version_id": ids}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build load PM aggregation members SQL: %w", err)
	}
	memberRows, err := r.pool.Query(ctx, memberSQL, memberArgs...)
	if err != nil {
		return fmt.Errorf("query PM aggregation members: %w", err)
	}
	defer memberRows.Close()
	for memberRows.Next() {
		var versionID uuid.UUID
		var member TaskMember
		if err := memberRows.Scan(
			&versionID, &member.DeviceID, &member.DeviceSN, &member.DimensionKey,
			&member.DimensionName, &member.ObjectLDN,
		); err != nil {
			return fmt.Errorf("scan PM aggregation member: %w", err)
		}
		if version := byID[versionID]; version != nil {
			version.Members[member.DeviceID] = append(version.Members[member.DeviceID], member)
		}
	}
	if err := memberRows.Err(); err != nil {
		return fmt.Errorf("iterate PM aggregation members: %w", err)
	}
	return nil
}

func validateSaveTask(req SaveTaskRequest) error {
	if req.Name == "" || req.Creator == "" || len(req.Granularities) == 0 ||
		len(req.Metrics) == 0 || len(req.Counters) == 0 {
		return fmt.Errorf("PM aggregation task requires name, creator, granularities, metrics and counters")
	}
	switch req.Dimension {
	case DimensionDevice, DimensionAggregateGroup, DimensionDeviceGroup,
		DimensionProduct, DimensionBand, DimensionNetwork:
	default:
		return fmt.Errorf("unsupported PM aggregation dimension %q", req.Dimension)
	}
	seenGranularities := make(map[Granularity]struct{}, len(req.Granularities))
	for _, granularity := range req.Granularities {
		switch granularity {
		case GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly:
		default:
			return fmt.Errorf("unsupported PM aggregation granularity %q", granularity)
		}
		if _, exists := seenGranularities[granularity]; exists {
			return fmt.Errorf("duplicate PM aggregation granularity %q", granularity)
		}
		seenGranularities[granularity] = struct{}{}
	}
	for _, required := range []Granularity{
		GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly,
	} {
		if _, exists := seenGranularities[required]; !exists {
			return fmt.Errorf("PM aggregation task requires fixed hourly, daily, weekly and monthly granularities")
		}
	}
	seenMetrics := make(map[string]struct{}, len(req.Metrics))
	for _, rule := range req.Metrics {
		if rule.MetricPath == "" {
			return fmt.Errorf("PM aggregation metric path is empty")
		}
		if _, exists := seenMetrics[rule.MetricPath]; exists {
			return fmt.Errorf("duplicate PM aggregation metric %q", rule.MetricPath)
		}
		seenMetrics[rule.MetricPath] = struct{}{}
		switch rule.Aggregation {
		case AggregationSum, AggregationAvg, AggregationMin, AggregationMax, AggregationFormula:
		default:
			return fmt.Errorf("unsupported PM aggregation operation %q", rule.Aggregation)
		}
		if rule.MetricType != "counter" && rule.MetricType != "kpi" {
			return fmt.Errorf("unsupported PM aggregation metric type %q", rule.MetricType)
		}
		if rule.MetricType == "kpi" && (rule.Formula == "" || len(rule.Dependencies) == 0) {
			return fmt.Errorf("PM aggregation KPI %q requires formula and counter dependencies", rule.MetricPath)
		}
	}
	seenCounters := make(map[string]struct{}, len(req.Counters))
	for _, rule := range req.Counters {
		if rule.MetricPath == "" {
			return fmt.Errorf("PM aggregation counter path is empty")
		}
		if _, exists := seenCounters[rule.MetricPath]; exists {
			return fmt.Errorf("duplicate PM aggregation counter %q", rule.MetricPath)
		}
		seenCounters[rule.MetricPath] = struct{}{}
		switch rule.Aggregation {
		case AggregationSum, AggregationAvg, AggregationMin, AggregationMax:
		default:
			return fmt.Errorf("unsupported PM aggregation counter operation %q", rule.Aggregation)
		}
	}
	for _, member := range req.Members {
		if member.DeviceID == uuid.Nil || member.DeviceSN == "" || member.DimensionKey == "" {
			return fmt.Errorf("PM aggregation member requires device, serial number and dimension key")
		}
	}
	return nil
}

func snapshotFromRequest(
	req SaveTaskRequest,
	taskID, versionID uuid.UUID,
	versionNo int,
	effectiveFrom time.Time,
) *TaskVersionSnapshot {
	snapshot := &TaskVersionSnapshot{
		TaskID: taskID, VersionID: versionID, VersionNo: versionNo,
		Name: req.Name, TaskEnabled: req.Enabled, Enabled: req.Enabled, Technology: req.Technology,
		Dimension: req.Dimension, Granularities: req.Granularities,
		EffectiveFrom: effectiveFrom, PlannedEndAt: req.PlannedEndAt,
		Metrics:  make(map[string]MetricRule),
		Counters: make(map[string]CounterRule),
		Members:  make(map[uuid.UUID][]TaskMember), ObjectLDNs: make(map[string]struct{}),
	}
	for _, rule := range req.Metrics {
		snapshot.Metrics[rule.MetricPath] = rule
	}
	for _, rule := range req.Counters {
		snapshot.Counters[rule.MetricPath] = rule
	}
	for _, member := range req.Members {
		snapshot.Members[member.DeviceID] = append(snapshot.Members[member.DeviceID], member)
	}
	for _, ldn := range req.ObjectLDNs {
		snapshot.ObjectLDNs[ldn] = struct{}{}
	}
	return snapshot
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
