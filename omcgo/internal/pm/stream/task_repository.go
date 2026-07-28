package stream

import (
	"bytes"
	"context"
	"fmt"
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

func NewPgTaskRepository(pool *pgxpool.Pool, bus event.EventBus) *PgTaskRepository {
	return &PgTaskRepository{pool: pool, bus: bus}
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

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin save PM aggregation task: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskID := req.TaskID
	if taskID == uuid.Nil {
		taskID = uuid.New()
	}
	insertSQL, insertArgs, buildErr := storage.Psql.Insert("pm_aggregation_tasks").
		Columns("id", "name", "enabled", "visibility", "creator").
		Values(taskID, req.Name, req.Enabled, req.Visibility, req.Creator).
		Suffix("ON CONFLICT (id) DO NOTHING").
		ToSql()
	if buildErr != nil {
		return nil, fmt.Errorf("build create PM aggregation task SQL: %w", buildErr)
	}
	if _, err := tx.Exec(ctx, insertSQL, insertArgs...); err != nil {
		return nil, fmt.Errorf("create PM aggregation task: %w", err)
	}
	lockSQL, lockArgs, buildErr := storage.Psql.Select("id", "current_version_id").
		From("pm_aggregation_tasks").
		Where(sq.Eq{"id": taskID, "deleted_at": nil}).
		Suffix("FOR UPDATE").
		ToSql()
	if buildErr != nil {
		return nil, fmt.Errorf("build lock PM aggregation task SQL: %w", buildErr)
	}
	var locked uuid.UUID
	var currentVersionID *uuid.UUID
	if err := tx.QueryRow(ctx, lockSQL, lockArgs...).Scan(&locked, &currentVersionID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("PM aggregation task %s is deleted", taskID)
		}
		return nil, fmt.Errorf("lock PM aggregation task: %w", err)
	}
	updateSQL, updateArgs, buildErr := storage.Psql.Update("pm_aggregation_tasks").
		Set("name", req.Name).
		Set("enabled", req.Enabled).
		Set("visibility", req.Visibility).
		Where(sq.Eq{"id": taskID}).
		ToSql()
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
	if r.bus != nil {
		payload := event.PMAggregationTaskVersionChangedPayload{
			TaskID: taskID, TaskVersionID: versionID, EffectiveFrom: effectiveFrom,
		}
		if evt, eventErr := event.NewEvent(event.SubjectPMAggregationTaskVersionChanged, payload); eventErr == nil {
			_ = r.bus.Publish(ctx, event.SubjectPMAggregationTaskVersionChanged, evt)
		}
	}
	return snapshot, nil
}

func saveEffectiveFrom(req SaveTaskRequest, now time.Time) time.Time {
	if !req.EffectiveFrom.IsZero() {
		return req.EffectiveFrom.UTC()
	}
	return now.UTC().Truncate(time.Hour).Add(time.Hour)
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete PM aggregation task: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := storage.Psql.Update("pm_aggregation_tasks").
		Set("enabled", false).
		Set("deleted_at", time.Now().UTC()).
		Where(sq.Eq{"id": taskID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete PM aggregation task SQL: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete PM aggregation task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	closeSQL, closeArgs, err := storage.Psql.Update("pm_aggregation_task_versions").
		Set("effective_to", effectiveTo).
		Where(sq.Eq{"task_id": taskID, "effective_to": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build close deleted PM aggregation task version SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, closeSQL, closeArgs...); err != nil {
		return fmt.Errorf("close deleted PM aggregation task version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete PM aggregation task: %w", err)
	}
	if r.bus != nil {
		payload := event.PMAggregationTaskVersionChangedPayload{
			TaskID: taskID, EffectiveFrom: effectiveTo,
		}
		if evt, eventErr := event.NewEvent(event.SubjectPMAggregationTaskVersionChanged, payload); eventErr == nil {
			_ = r.bus.Publish(ctx, event.SubjectPMAggregationTaskVersionChanged, evt)
		}
	}
	return nil
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

func insertTaskMembers(ctx context.Context, tx pgx.Tx, versionID uuid.UUID, members []TaskMember) error {
	if len(members) == 0 {
		return nil
	}
	builder := storage.Psql.Insert("pm_aggregation_version_members").
		Columns(
			"task_version_id", "device_id", "device_sn",
			"dimension_key", "dimension_name", "object_ldn",
		)
	for _, member := range members {
		builder = builder.Values(
			versionID, member.DeviceID, member.DeviceSN,
			member.DimensionKey, member.DimensionName, member.ObjectLDN,
		)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert PM aggregation members SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert PM aggregation members: %w", err)
	}
	return nil
}

func (r *PgTaskRepository) LoadMatchable(ctx context.Context, at time.Time) ([]*TaskVersionSnapshot, error) {
	query, args, err := storage.Psql.Select(
		"t.id", "v.id", "v.version_no", "t.name", "v.enabled",
		"COALESCE(v.technology, '')", "v.dimension", "v.granularities",
		"v.object_ldns", "v.effective_from", "v.effective_to",
	).From("pm_aggregation_task_versions v").
		Join("pm_aggregation_tasks t ON t.id = v.task_id").
		Where(sq.Or{
			sq.Eq{"v.effective_to": nil},
			sq.GtOrEq{"v.effective_to": at.Add(-35 * 24 * time.Hour)},
		}).
		OrderBy("v.task_id", "v.version_no").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build load matchable task versions SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query matchable PM aggregation task versions: %w", err)
	}
	var versions []*TaskVersionSnapshot
	for rows.Next() {
		var version TaskVersionSnapshot
		var granularityStrings []string
		var objectLDNs []string
		if err := rows.Scan(
			&version.TaskID, &version.VersionID, &version.VersionNo, &version.Name, &version.Enabled,
			&version.Technology, &version.Dimension, &granularityStrings, &objectLDNs,
			&version.EffectiveFrom, &version.EffectiveTo,
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
	if err := r.loadDetails(ctx, versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *PgTaskRepository) loadDetails(
	ctx context.Context,
	versions []*TaskVersionSnapshot,
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
		Name: req.Name, Enabled: req.Enabled, Technology: req.Technology,
		Dimension: req.Dimension, Granularities: req.Granularities,
		EffectiveFrom: effectiveFrom, Metrics: make(map[string]MetricRule),
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
