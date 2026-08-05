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
}

var _ NetworkRollupReader = (*NetworkRollupRepository)(nil)
var _ CounterSeriesReader = (*NetworkRollupRepository)(nil)

func NewNetworkRollupRepository(pool *pgxpool.Pool, statementTimeout time.Duration) *NetworkRollupRepository {
	return &NetworkRollupRepository{pool: pool, statementTimeout: statementTimeout}
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
	mergeSums := query.MetricType == "" || query.MetricType == metrics.MetricTypeKPI
	return mergeNetworkRollupVersionSlices(points, mergeSums), nil
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
		"COALESCE(metric_rule.formula, '')",
		"r.sample_count",
		"r.version_effective_from",
		"COALESCE(dictionary.statis_type, '')",
	).
		From("pm_aggregation_results r").
		Join(`pm_aggregation_version_metrics metric_rule
  ON metric_rule.task_version_id = r.task_version_id
 AND metric_rule.metric_path = r.metric_path`).
		LeftJoin(`pm_metric_dictionary dictionary
  ON dictionary.metric_path = r.metric_path`).
		Join(`pm_aggregation_publications published_revision
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
// immutable; when an unchanged metric spans multiple effective slices of the
// same natural window, the homepage combines sum slices. A changed formula
// keeps the latest slice, preserving the existing semantic-change boundary.
func mergeNetworkRollupVersionSlices(
	points []NetworkRollupPoint,
	mergeSums bool,
) []NetworkRollupPoint {
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
		slices := groups[key]
		latest := slices[0]
		for _, point := range slices[1:] {
			if point.VersionEffectiveFrom.After(latest.VersionEffectiveFrom) ||
				(point.VersionEffectiveFrom.Equal(latest.VersionEffectiveFrom) &&
					point.CreatedAt.After(latest.CreatedAt)) {
				latest = point
			}
		}
		operation := strings.ToLower(strings.TrimSpace(latest.StatisType))
		if operation == "" && latest.MetricPath != "" &&
			strings.HasPrefix(strings.ToUpper(latest.MetricPath), "C") {
			operation = string(latest.Aggregation)
		}
		if !mergeSums || operation != "sum" {
			out = append(out, latest)
			continue
		}
		merged := latest
		merged.Value = 0
		merged.SampleCount = 0
		merged.Complete = true
		merged.MissingSlots = 0
		for _, point := range slices {
			if point.Aggregation != latest.Aggregation || point.Formula != latest.Formula {
				continue
			}
			merged.Value = jsonx.Float(float64(merged.Value) + float64(point.Value))
			merged.SampleCount += point.SampleCount
			merged.Complete = merged.Complete && point.Complete
			merged.MissingSlots += point.MissingSlots
			if point.CreatedAt.After(merged.CreatedAt) {
				merged.CreatedAt = point.CreatedAt
			}
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
