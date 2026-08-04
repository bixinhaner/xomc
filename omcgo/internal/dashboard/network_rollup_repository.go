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
	Technology   model.Technology
	MetricPath   string
	Granularity  metrics.Granularity
	WindowStart  time.Time
	WindowEnd    time.Time
	Value        jsonx.Float
	Complete     bool
	MissingSlots int64
	CreatedAt    time.Time
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
	return r.query(ctx, sql, args...)
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
		"DISTINCT ON (r.technology, r.metric_path, r.window_start) r.technology",
		"r.metric_path",
		"r.granularity",
		"r.window_start",
		"r.window_end",
		"r.metric_value",
		"r.complete",
		"r.missing_slots",
		"r.created_at",
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
			"r.metric_type": string(metricType),
			"r.task_id":     networkTaskIDs(query.Technology),
			"r.granularity": query.Granularity,
			"r.metric_path": metricPaths,
		}).
		Where(sq.GtOrEq{"r.window_start": query.StartTime}).
		Where(sq.Lt{"r.window_start": query.EndTime}).
		OrderBy("r.technology", "r.metric_path", "r.window_start", "r.created_at DESC")
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
