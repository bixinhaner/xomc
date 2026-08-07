package dashboard

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/kpi/expr"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
)

type NetworkRollupPoint struct {
	Technology           model.Technology
	MetricPath           string
	Granularity          metrics.Granularity
	WindowStart          time.Time
	WindowEnd            time.Time
	Value                jsonx.Float
	Complete             bool
	MissingSlots         int64
	CreatedAt            time.Time
	Aggregation          pmstream.AggregationOp
	Formula              string
	SampleCount          int64
	VersionEffectiveFrom time.Time
	StatisType           string
	TaskVersionID        uuid.UUID
	Dependencies         []string
	CounterSignature     []string
}

type NetworkRollupQuery struct {
	Technology  model.Technology
	Granularity metrics.Granularity
	MetricType  metrics.MetricType
	MetricPaths []string
	StartTime   time.Time
	EndTime     time.Time
}

type NetworkRollupReader interface {
	ListSeries(context.Context, NetworkRollupQuery) ([]NetworkRollupPoint, error)
	ListLatestHourly(context.Context, time.Time, time.Time) ([]NetworkRollupPoint, error)
}

type NetworkRollupRepository struct {
	pool             *pgxpool.Pool
	statementTimeout time.Duration
	counterRollups   networkCounterRollupReader
}

type networkCounterRollupReader interface {
	VisitSnapshotsForPeriod(
		context.Context,
		[]uuid.UUID,
		pmstream.Granularity,
		time.Time,
		time.Time,
		func(pmstream.RollupPayload) error,
	) error
}

var _ NetworkRollupReader = (*NetworkRollupRepository)(nil)
var _ CounterSeriesReader = (*NetworkRollupRepository)(nil)

func NewNetworkRollupRepository(pool *pgxpool.Pool, statementTimeout time.Duration) *NetworkRollupRepository {
	return &NetworkRollupRepository{
		pool: pool, statementTimeout: statementTimeout,
		counterRollups: pmstream.NewRollupOutboxRepository(pool),
	}
}

func (r *NetworkRollupRepository) ListSeries(ctx context.Context, query NetworkRollupQuery) ([]NetworkRollupPoint, error) {
	sql, args, err := buildNetworkRollupSeriesSQL(query)
	if err != nil {
		return nil, err
	}
	points, err := r.query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	// Issue #251 guarantees that a task version starts contributing from the
	// next complete hour. An hourly window therefore never needs cross-version
	// composition, so keep this hot dashboard path free of definition joins.
	if query.Granularity == metrics.GranularityHourly {
		return points, nil
	}
	merged := mergeNetworkRollupVersionSlices(points)
	metricType := query.MetricType
	if metricType == "" {
		metricType = metrics.MetricTypeKPI
	}
	if metricType == metrics.MetricTypeKPI && query.Granularity != metrics.GranularityHourly {
		if err := r.recomputePercentVersionSlices(ctx, points, merged); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func (r *NetworkRollupRepository) ListLatestHourly(ctx context.Context, start, end time.Time) ([]NetworkRollupPoint, error) {
	sql, args, err := buildLatestNetworkHourlySQL(start, end)
	if err != nil {
		return nil, err
	}
	return r.query(ctx, sql, args...)
}

func (r *NetworkRollupRepository) query(ctx context.Context, query string, args ...any) ([]NetworkRollupPoint, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("dashboard network rollup repository is not configured")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin dashboard network rollup query: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	if r.statementTimeout > 0 {
		if _, err := tx.Exec(ctx,
			"SELECT set_config('statement_timeout', $1, true)",
			r.statementTimeout.String(),
		); err != nil {
			return nil, fmt.Errorf("set dashboard statement timeout: %w", err)
		}
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query dashboard network rollups: %w", err)
	}
	defer rows.Close()

	points := make([]NetworkRollupPoint, 0)
	for rows.Next() {
		var point NetworkRollupPoint
		var technology string
		var granularity string
		if err := rows.Scan(
			&technology,
			&point.MetricPath,
			&granularity,
			&point.WindowStart,
			&point.WindowEnd,
			&point.Value,
			&point.Complete,
			&point.MissingSlots,
			&point.CreatedAt,
			&point.Aggregation,
			&point.Formula,
			&point.SampleCount,
			&point.VersionEffectiveFrom,
			&point.StatisType,
			&point.TaskVersionID,
			&point.Dependencies,
			&point.CounterSignature,
		); err != nil {
			return nil, fmt.Errorf("scan dashboard network rollup: %w", err)
		}
		point.Technology = model.Technology(technology)
		point.Granularity = metrics.Granularity(granularity)
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dashboard network rollups: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit dashboard network rollup query: %w", err)
	}
	return points, nil
}

func buildNetworkRollupSeriesSQL(query NetworkRollupQuery) (string, []any, error) {
	if err := validateNetworkRollupQuery(query); err != nil {
		return "", nil, err
	}
	metricType := query.MetricType
	if metricType == "" {
		metricType = metrics.MetricTypeKPI
	}
	metricPaths := normalizeMetricPaths(query.MetricPaths)
	formulaColumn := "''"
	statisTypeColumn := "''"
	dependenciesColumn := "ARRAY[]::text[]"
	counterSignatureColumn := "ARRAY[]::text[]"
	loadKPIDefinition := query.Granularity != metrics.GranularityHourly &&
		metricType == metrics.MetricTypeKPI
	if loadKPIDefinition {
		formulaColumn = "COALESCE(metric_rule.formula, '')"
		statisTypeColumn = "COALESCE(dictionary.statis_type, '')"
		dependenciesColumn = "metric_rule.dependencies"
		counterSignatureColumn = `CASE
  WHEN LOWER(COALESCE(dictionary.statis_type, '')) = 'pct' THEN COALESCE((
    SELECT array_agg(counter.metric_path || ':' || counter.aggregation_op ORDER BY counter.metric_path)
      FROM pm_aggregation_version_counters counter
     WHERE counter.task_version_id = r.task_version_id
       AND counter.metric_path = ANY(metric_rule.dependencies)
  ), ARRAY[]::text[])
  ELSE ARRAY[]::text[]
END`
	}
	builder := storage.Psql.Select(
		"r.technology",
		"r.metric_path",
		"r.granularity",
		"r.window_start",
		"r.window_end",
		"r.metric_value",
		"r.complete",
		"r.missing_slots",
		"r.created_at",
		"r.aggregation_op",
		formulaColumn,
		"r.sample_count",
		"r.version_effective_from",
		statisTypeColumn,
		"r.task_version_id",
		dependenciesColumn,
		counterSignatureColumn,
	).
		From("pm_aggregation_results r")
	if loadKPIDefinition {
		builder = builder.Join(`pm_aggregation_version_metrics metric_rule
  ON metric_rule.task_version_id = r.task_version_id
 AND metric_rule.metric_path = r.metric_path`).
			LeftJoin(`pm_metric_dictionary dictionary
  ON dictionary.metric_path = r.metric_path`)
	}
	builder = builder.Join(`pm_aggregation_publications published_revision
  ON published_revision.task_version_id = r.task_version_id
 AND published_revision.granularity = r.granularity
 AND published_revision.window_start = r.window_start
 AND published_revision.status = 'published'
	 AND published_revision.revision = r.revision`).
		Where(sq.Eq{
			"r.dimension":   "network",
			"r.metric_type": string(metricType),
			"r.task_id":     networkTaskIDs(query.Technology),
			"r.granularity": query.Granularity,
			"r.metric_path": metricPaths,
		}).
		Where(sq.GtOrEq{"r.window_start": query.StartTime}).
		Where(sq.Lt{"r.window_start": query.EndTime}).
		OrderBy("r.technology", "r.metric_path", "r.window_start", "r.version_effective_from", "r.created_at")
	if query.Technology != "" {
		builder = builder.Where(sq.Eq{"r.technology": query.Technology})
	}
	return builder.ToSql()
}

func buildLatestNetworkHourlySQL(start, end time.Time) (string, []any, error) {
	if start.IsZero() || !end.After(start) {
		return "", nil, fmt.Errorf("dashboard latest hourly window is invalid")
	}
	return storage.Psql.Select(
		"DISTINCT ON (r.technology, r.metric_path) r.technology",
		"r.metric_path",
		"r.granularity",
		"r.window_start",
		"r.window_end",
		"r.metric_value",
		"r.complete",
		"r.missing_slots",
		"r.created_at",
		"r.aggregation_op",
		"''",
		"r.sample_count",
		"r.version_effective_from",
		"''",
		"r.task_version_id",
		"ARRAY[]::text[]",
		"ARRAY[]::text[]",
	).
		From("pm_aggregation_results r").
		Join(`pm_aggregation_publications published_revision
  ON published_revision.task_version_id = r.task_version_id
 AND published_revision.granularity = r.granularity
 AND published_revision.window_start = r.window_start
 AND published_revision.status = 'published'
 AND published_revision.revision = r.revision`).
		Where(sq.Eq{
			"r.dimension":   "network",
			"r.metric_type": string(metrics.MetricTypeKPI),
			"r.task_id":     networkTaskIDs(""),
			"r.granularity": metrics.GranularityHourly,
		}).
		Where(sq.GtOrEq{"r.window_start": start}).
		Where(sq.Lt{"r.window_start": end}).
		OrderBy("r.technology", "r.metric_path", "r.window_start DESC", "r.created_at DESC").
		ToSql()
}

type networkRollupWindowKey struct {
	technology  model.Technology
	metricPath  string
	granularity metrics.Granularity
	windowStart int64
	windowEnd   int64
}

// mergeNetworkRollupVersionSlices is dashboard-only. PM task versions remain
// immutable; when an unchanged metric spans multiple effective slices of one
// natural window, the homepage combines value-aggregation slices. pct needs
// compact Counter state and is recomputed separately by the repository.
func mergeNetworkRollupVersionSlices(points []NetworkRollupPoint) []NetworkRollupPoint {
	groups := make(map[networkRollupWindowKey][]NetworkRollupPoint)
	keys := make([]networkRollupWindowKey, 0)
	for _, point := range points {
		key := networkRollupWindowKey{
			technology: point.Technology, metricPath: point.MetricPath,
			granularity: point.Granularity,
			windowStart: point.WindowStart.UTC().UnixNano(),
			windowEnd:   point.WindowEnd.UTC().UnixNano(),
		}
		if _, exists := groups[key]; !exists {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], point)
	}
	out := make([]NetworkRollupPoint, 0, len(keys))
	for _, key := range keys {
		slices := compatibleLatestVersionSlices(groups[key])
		latest := slices[len(slices)-1]
		operation := networkRollupOperation(latest)
		if len(slices) == 1 || operation == "pct" {
			out = append(out, latest)
			continue
		}
		merged := mergeNetworkRollupMetadata(latest, slices)
		switch operation {
		case "sum":
			merged.Value = 0
			for _, point := range slices {
				merged.Value = jsonx.Float(float64(merged.Value) + float64(point.Value))
			}
		case "avg":
			var weighted float64
			var samples int64
			for _, point := range slices {
				if point.SampleCount <= 0 {
					continue
				}
				weighted += float64(point.Value) * float64(point.SampleCount)
				samples += point.SampleCount
			}
			if samples == 0 {
				out = append(out, latest)
				continue
			}
			merged.Value = jsonx.Float(weighted / float64(samples))
		case "min":
			for _, point := range slices[:len(slices)-1] {
				if point.Value < merged.Value {
					merged.Value = point.Value
				}
			}
		case "max":
			for _, point := range slices[:len(slices)-1] {
				if point.Value > merged.Value {
					merged.Value = point.Value
				}
			}
		default:
			out = append(out, latest)
			continue
		}
		out = append(out, merged)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Technology != out[j].Technology {
			return out[i].Technology < out[j].Technology
		}
		if out[i].MetricPath != out[j].MetricPath {
			return out[i].MetricPath < out[j].MetricPath
		}
		return out[i].WindowStart.Before(out[j].WindowStart)
	})
	return out
}

func compatibleLatestVersionSlices(points []NetworkRollupPoint) []NetworkRollupPoint {
	slices := append([]NetworkRollupPoint(nil), points...)
	sort.SliceStable(slices, func(i, j int) bool {
		if !slices[i].VersionEffectiveFrom.Equal(slices[j].VersionEffectiveFrom) {
			return slices[i].VersionEffectiveFrom.Before(slices[j].VersionEffectiveFrom)
		}
		return slices[i].CreatedAt.Before(slices[j].CreatedAt)
	})
	latest := slices[len(slices)-1]
	start := len(slices) - 1
	for start > 0 && sameNetworkRollupSemantics(slices[start-1], latest) {
		start--
	}
	return slices[start:]
}

func sameNetworkRollupSemantics(left, right NetworkRollupPoint) bool {
	return networkRollupOperation(left) == networkRollupOperation(right) &&
		left.Aggregation == right.Aggregation &&
		left.Formula == right.Formula &&
		strings.Join(left.Dependencies, "\x1f") == strings.Join(right.Dependencies, "\x1f") &&
		strings.Join(left.CounterSignature, "\x1f") == strings.Join(right.CounterSignature, "\x1f")
}

func networkRollupOperation(point NetworkRollupPoint) string {
	operation := strings.ToLower(strings.TrimSpace(point.StatisType))
	if operation == "" && point.MetricPath != "" &&
		strings.HasPrefix(strings.ToUpper(point.MetricPath), "C") {
		operation = string(point.Aggregation)
	}
	return operation
}

func mergeNetworkRollupMetadata(
	latest NetworkRollupPoint,
	slices []NetworkRollupPoint,
) NetworkRollupPoint {
	merged := latest
	merged.SampleCount = 0
	merged.Complete = true
	merged.MissingSlots = 0
	for _, point := range slices {
		merged.SampleCount += point.SampleCount
		merged.Complete = merged.Complete && point.Complete
		merged.MissingSlots += point.MissingSlots
		if point.CreatedAt.After(merged.CreatedAt) {
			merged.CreatedAt = point.CreatedAt
		}
	}
	return merged
}

type networkPercentTarget struct {
	pointIndex int
	start      time.Time
	end        time.Time
	formula    string
	operations map[string]pmstream.AggregationOp
	states     map[string]*networkCounterState
}

type networkCounterState struct {
	sum   float64
	count int64
	min   float64
	max   float64
}

func (r *NetworkRollupRepository) recomputePercentVersionSlices(
	ctx context.Context,
	points []NetworkRollupPoint,
	merged []NetworkRollupPoint,
) error {
	grouped := make(map[networkRollupWindowKey][]NetworkRollupPoint)
	for _, point := range points {
		grouped[networkRollupKey(point)] = append(grouped[networkRollupKey(point)], point)
	}
	mergedIndexes := make(map[networkRollupWindowKey]int, len(merged))
	for index, point := range merged {
		mergedIndexes[networkRollupKey(point)] = index
	}

	byVersion := make(map[uuid.UUID][]*networkPercentTarget)
	versionSet := make(map[uuid.UUID]struct{})
	var snapshotStart, snapshotEnd time.Time
	for key, group := range grouped {
		slices := compatibleLatestVersionSlices(group)
		latest := slices[len(slices)-1]
		if len(slices) < 2 || networkRollupOperation(latest) != "pct" {
			continue
		}
		operations, ok := parseCounterSignature(latest.Dependencies, latest.CounterSignature)
		if !ok {
			continue
		}
		target := &networkPercentTarget{
			pointIndex: mergedIndexes[key], start: latest.WindowStart, end: latest.WindowEnd,
			formula:    latest.Formula,
			operations: operations, states: make(map[string]*networkCounterState),
		}
		for _, point := range slices {
			versionSet[point.TaskVersionID] = struct{}{}
			byVersion[point.TaskVersionID] = append(byVersion[point.TaskVersionID], target)
		}
		if snapshotStart.IsZero() || target.start.Before(snapshotStart) {
			snapshotStart = target.start
		}
		if target.end.After(snapshotEnd) {
			snapshotEnd = target.end
		}
	}
	if len(versionSet) == 0 {
		return nil
	}
	versionIDs := make([]uuid.UUID, 0, len(versionSet))
	for versionID := range versionSet {
		versionIDs = append(versionIDs, versionID)
	}
	sort.Slice(versionIDs, func(i, j int) bool { return versionIDs[i].String() < versionIDs[j].String() })

	sourceGranularity := pmstream.GranularityHourly
	if len(merged) > 0 && merged[0].Granularity == metrics.GranularityWeekly {
		sourceGranularity = pmstream.GranularityDaily
	}
	if r.counterRollups == nil {
		return fmt.Errorf("dashboard pct Counter rollup reader is not configured")
	}
	if err := r.counterRollups.VisitSnapshotsForPeriod(
		ctx, versionIDs, sourceGranularity, snapshotStart, snapshotEnd,
		func(payload pmstream.RollupPayload) error {
			if payload.EntityKey != "network" {
				return nil
			}
			for _, target := range byVersion[payload.TaskVersionID] {
				if payload.WindowStart.Before(target.start) || !payload.WindowStart.Before(target.end) {
					continue
				}
				mergeNetworkPercentPayload(target, payload)
			}
			return nil
		},
	); err != nil {
		return fmt.Errorf("recompute dashboard pct across PM task versions: %w", err)
	}

	seen := make(map[*networkPercentTarget]struct{})
	for _, targets := range byVersion {
		for _, target := range targets {
			if _, exists := seen[target]; exists {
				continue
			}
			seen[target] = struct{}{}
			counters, sampleCount, ok := target.finalizedCounters()
			if !ok {
				continue
			}
			formula, err := expr.Parse(target.formula)
			if err != nil {
				return fmt.Errorf("parse dashboard pct formula: %w", err)
			}
			value, err := formula.Evaluate(counters)
			if err != nil {
				continue
			}
			key := networkRollupKey(merged[target.pointIndex])
			slices := compatibleLatestVersionSlices(grouped[key])
			point := mergeNetworkRollupMetadata(merged[target.pointIndex], slices)
			point.Value = jsonx.Float(value)
			point.SampleCount = sampleCount
			merged[target.pointIndex] = point
		}
	}
	return nil
}

func networkRollupKey(point NetworkRollupPoint) networkRollupWindowKey {
	return networkRollupWindowKey{
		technology: point.Technology, metricPath: point.MetricPath,
		granularity: point.Granularity,
		windowStart: point.WindowStart.UTC().UnixNano(),
		windowEnd:   point.WindowEnd.UTC().UnixNano(),
	}
}

func parseCounterSignature(
	dependencies []string,
	signature []string,
) (map[string]pmstream.AggregationOp, bool) {
	operations := make(map[string]pmstream.AggregationOp, len(signature))
	for _, item := range signature {
		path, operation, ok := strings.Cut(item, ":")
		if !ok {
			return nil, false
		}
		op := pmstream.AggregationOp(operation)
		switch op {
		case pmstream.AggregationSum, pmstream.AggregationAvg,
			pmstream.AggregationMin, pmstream.AggregationMax:
		default:
			return nil, false
		}
		operations[path] = op
	}
	for _, dependency := range dependencies {
		if _, exists := operations[dependency]; !exists {
			return nil, false
		}
	}
	return operations, len(dependencies) > 0
}

func mergeNetworkPercentPayload(target *networkPercentTarget, payload pmstream.RollupPayload) {
	for _, value := range payload.Values {
		expected, needed := target.operations[value.MetricPath]
		if !needed || value.Operation != expected || value.Count <= 0 {
			continue
		}
		state := target.states[value.MetricPath]
		if state == nil {
			state = &networkCounterState{min: value.Min, max: value.Max}
			target.states[value.MetricPath] = state
		} else {
			if value.Min < state.min {
				state.min = value.Min
			}
			if value.Max > state.max {
				state.max = value.Max
			}
		}
		state.sum += value.Sum
		state.count += value.Count
	}
}

func (target *networkPercentTarget) finalizedCounters() (map[string]float64, int64, bool) {
	counters := make(map[string]float64, len(target.operations))
	var minimumSamples int64
	for path, operation := range target.operations {
		state := target.states[path]
		if state == nil || state.count <= 0 {
			return nil, 0, false
		}
		switch operation {
		case pmstream.AggregationAvg:
			counters[path] = state.sum / float64(state.count)
		case pmstream.AggregationMin:
			counters[path] = state.min
		case pmstream.AggregationMax:
			counters[path] = state.max
		default:
			counters[path] = state.sum
		}
		if minimumSamples == 0 || state.count < minimumSamples {
			minimumSamples = state.count
		}
	}
	return counters, minimumSamples, true
}

func validateNetworkRollupQuery(query NetworkRollupQuery) error {
	if query.Technology != "" && !query.Technology.IsValid() {
		return fmt.Errorf("dashboard network rollup technology %q is invalid", query.Technology)
	}
	switch query.Granularity {
	case metrics.GranularityHourly, metrics.GranularityDaily, metrics.GranularityWeekly:
	default:
		return fmt.Errorf("dashboard network rollup granularity %q is invalid", query.Granularity)
	}
	if query.StartTime.IsZero() || !query.EndTime.After(query.StartTime) {
		return fmt.Errorf("dashboard network rollup time window is invalid")
	}
	if len(normalizeMetricPaths(query.MetricPaths)) == 0 {
		return fmt.Errorf("dashboard network rollup metric paths are required")
	}
	if query.MetricType != "" &&
		query.MetricType != metrics.MetricTypeKPI &&
		query.MetricType != metrics.MetricTypeCounter {
		return fmt.Errorf("dashboard network rollup metric type %q is invalid", query.MetricType)
	}
	return nil
}

func networkTaskIDs(technology model.Technology) []uuid.UUID {
	technologies := []model.Technology{technology}
	if technology == "" {
		technologies = []model.Technology{model.TechLTE, model.TechNR, model.TechGSM}
	}
	taskIDs := make([]uuid.UUID, 0, len(technologies))
	for _, candidate := range technologies {
		taskID, ok := pmstream.BuiltinNetworkTaskID(string(candidate))
		if ok {
			taskIDs = append(taskIDs, taskID)
		}
	}
	return taskIDs
}

func normalizeMetricPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}
