package stream

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/redis/go-redis/v9"
)

type ProgressResult struct {
	ID                   uuid.UUID
	TaskID               uuid.UUID
	TaskVersionID        uuid.UUID
	Granularity          Granularity
	WindowStart          time.Time
	WindowEnd            time.Time
	Dimension            Dimension
	DimensionKey         string
	DimensionName        string
	ObjectLDN            string
	DeviceOUI            string
	DeviceSN             string
	MetricPath           string
	MetricType           string
	Operation            AggregationOp
	Value                float64
	Revision             int
	VersionEffectiveFrom time.Time
	VersionEffectiveTo   *time.Time
	ReceivedSlots        int64
	ExpectedSlots        int64
	VersionExpectedSlots int64
	NaturalExpectedSlots int64
	VersionSliceComplete bool
	PeriodComplete       bool
	Partial              bool
}

type PeriodProgress struct {
	Granularity          Granularity `json:"granularity"`
	WindowStart          time.Time   `json:"window_start"`
	WindowEnd            time.Time   `json:"window_end"`
	EntityKey            string      `json:"entity_key"`
	Revision             int         `json:"revision"`
	VersionEffectiveFrom time.Time   `json:"version_effective_from"`
	VersionEffectiveTo   *time.Time  `json:"version_effective_to"`
	ReceivedSlots        int64       `json:"received_slots"`
	ExpectedSlots        int64       `json:"expected_slots"`
}

type ProgressQueryResult struct {
	Rows    []ProgressResult
	Periods []PeriodProgress
}

type ProgressService struct {
	pool   *pgxpool.Pool
	store  *RedisWindowStore
	loader MatchableLoader
}

type progressCandidate struct {
	key       WindowKey
	revision  int
	status    string
	effective *time.Time
	openedAt  time.Time
	received  int64
	expected  int64
}

const (
	progressQueryTimeout = 2500 * time.Millisecond
	maxProgressResults   = 10000
)

var ErrProgressUnavailable = errors.New("PM aggregation progress state unavailable")

func NewProgressService(
	pool *pgxpool.Pool,
	store *RedisWindowStore,
	loader MatchableLoader,
) *ProgressService {
	return &ProgressService{pool: pool, store: store, loader: loader}
}

func (s *ProgressService) Query(
	ctx context.Context,
	taskID uuid.UUID,
	start, end time.Time,
) (ProgressQueryResult, error) {
	queryCtx, cancel := context.WithTimeout(ctx, progressQueryTimeout)
	defer cancel()
	ctx = queryCtx
	versions, err := s.loader.LoadMatchable(ctx, time.Now().UTC())
	if err != nil {
		return ProgressQueryResult{}, fmt.Errorf("load PM aggregation versions for progress: %w", err)
	}
	snapshot := BuildTaskSnapshot(versions)
	builder := storage.Psql.Select(
		"task_id", "task_version_id", "entity_key", "granularity",
		"window_start", "window_end", "revision", "status",
		"version_effective_from", "opened_at", "received_slots", "expected_slots",
	).From("pm_aggregation_windows").
		Where(sq.Eq{
			"task_id":     taskID,
			"status":      []string{"open", "finalizing", "failed", "rebuilding"},
			"granularity": []string{string(GranularityDaily), string(GranularityWeekly)},
		})
	if !start.IsZero() {
		builder = builder.Where(sq.Gt{"window_end": start})
	}
	if !end.IsZero() {
		builder = builder.Where(sq.Lt{"window_start": end})
	}
	query, args, err := builder.OrderBy(
		"granularity", "window_start", "task_version_id", "entity_key",
	).ToSql()
	if err != nil {
		return ProgressQueryResult{}, fmt.Errorf("build PM progress windows query: %w", err)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return ProgressQueryResult{}, fmt.Errorf("query PM progress windows: %w", err)
	}
	defer rows.Close()
	var candidates []progressCandidate
	for rows.Next() {
		var item progressCandidate
		if err := rows.Scan(
			&item.key.TaskID, &item.key.TaskVersionID, &item.key.EntityKey,
			&item.key.Granularity, &item.key.Start, &item.key.End, &item.revision,
			&item.status, &item.effective, &item.openedAt, &item.received, &item.expected,
		); err != nil {
			return ProgressQueryResult{}, fmt.Errorf("scan PM progress window: %w", err)
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		return ProgressQueryResult{}, fmt.Errorf("iterate PM progress windows: %w", err)
	}
	activeVersions := make(map[string]progressCandidate)
	for _, item := range candidates {
		group := progressVersionGroup(item.key)
		current, exists := activeVersions[group]
		if !exists || progressVersionAfter(item, current, snapshot) {
			activeVersions[group] = item
		}
	}
	var out []ProgressResult
	periods := make(map[string]PeriodProgress)
	for _, item := range candidates {
		active := activeVersions[progressVersionGroup(item.key)]
		if item.key.TaskVersionID != active.key.TaskVersionID {
			continue
		}
		if progressStatusUnavailable(item.status) {
			return ProgressQueryResult{}, fmt.Errorf(
				"%w: window %s/%s is failed",
				ErrProgressUnavailable, item.key.Granularity, item.key.Start,
			)
		}
		version := snapshot.ByVersion[item.key.TaskVersionID]
		expected := naturalExpectedSlots(
			version, item.key, WindowState{ExpectedSlots: item.expected},
		)
		period := PeriodProgress{
			Granularity: item.key.Granularity,
			WindowStart: item.key.Start, WindowEnd: item.key.End,
			EntityKey: item.key.EntityKey, Revision: item.revision,
			ReceivedSlots: item.received, ExpectedSlots: expected,
		}
		if version != nil {
			period.VersionEffectiveFrom = version.EffectiveFrom
			period.VersionEffectiveTo = version.EffectiveTo
		}
		group := progressVersionGroup(item.key)
		if current, ok := periods[group]; !ok || lowerProgressCoverage(period, current) {
			periods[group] = period
		}
	}
	for _, item := range candidates {
		active := activeVersions[progressVersionGroup(item.key)]
		if item.key.TaskVersionID != active.key.TaskVersionID {
			continue
		}
		state, err := s.store.Read(ctx, item.key)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return ProgressQueryResult{}, fmt.Errorf(
					"%w: Redis state missing for %s/%s",
					ErrProgressUnavailable, item.key.Granularity, item.key.Start,
				)
			}
			return ProgressQueryResult{}, fmt.Errorf("read PM progress window: %w", err)
		}
		version := snapshot.ByVersion[item.key.TaskVersionID]
		results, err := BuildProgressResults(
			version, item.key, item.revision, state,
		)
		if err != nil {
			return ProgressQueryResult{}, err
		}
		remaining := maxProgressResults - len(out)
		if remaining <= 0 {
			break
		}
		if len(results) > remaining {
			results = results[:remaining]
		}
		out = append(out, results...)
	}
	periodList := make([]PeriodProgress, 0, len(periods))
	for _, period := range periods {
		periodList = append(periodList, period)
	}
	sort.Slice(periodList, func(i, j int) bool {
		if periodList[i].Granularity != periodList[j].Granularity {
			return periodList[i].Granularity < periodList[j].Granularity
		}
		return periodList[i].WindowStart.Before(periodList[j].WindowStart)
	})
	return ProgressQueryResult{Rows: out, Periods: periodList}, nil
}

func lowerProgressCoverage(left, right PeriodProgress) bool {
	leftExpected := max(left.ExpectedSlots, 1)
	rightExpected := max(right.ExpectedSlots, 1)
	leftScaled := left.ReceivedSlots * rightExpected
	rightScaled := right.ReceivedSlots * leftExpected
	if leftScaled != rightScaled {
		return leftScaled < rightScaled
	}
	return left.EntityKey < right.EntityKey
}

func progressStatusUnavailable(status string) bool {
	return status == "failed" || status == "rebuilding"
}

func progressVersionGroup(key WindowKey) string {
	return fmt.Sprintf("%s|%d", key.Granularity, key.Start.UTC().UnixNano())
}

func progressVersionAfter(left, right progressCandidate, snapshot *TaskSnapshot) bool {
	leftEffective := candidateEffectiveFrom(left, snapshot)
	rightEffective := candidateEffectiveFrom(right, snapshot)
	if !leftEffective.Equal(rightEffective) {
		return leftEffective.After(rightEffective)
	}
	if left.openedAt.Equal(right.openedAt) {
		return left.key.TaskVersionID.String() > right.key.TaskVersionID.String()
	}
	return left.openedAt.After(right.openedAt)
}

func candidateEffectiveFrom(item progressCandidate, snapshot *TaskSnapshot) time.Time {
	if item.effective != nil {
		return *item.effective
	}
	if snapshot != nil {
		if version := snapshot.ByVersion[item.key.TaskVersionID]; version != nil {
			return version.EffectiveFrom
		}
	}
	return item.openedAt
}

func BuildProgressResults(
	version *TaskVersionSnapshot,
	key WindowKey,
	revision int,
	state WindowState,
) ([]ProgressResult, error) {
	metrics, _, err := buildFinalizedMetrics(version, state)
	if err != nil {
		return nil, err
	}
	received := state.ReceivedSlots
	versionExpected := state.ExpectedSlots
	naturalExpected := naturalExpectedSlots(version, key, state)
	out := make([]ProgressResult, 0, len(metrics))
	for _, metric := range metrics {
		definition := metric.Definition
		identity := fmt.Sprintf(
			"%s:%s:%s:%s:%s:%s:%d",
			key.TaskVersionID, key.Granularity, key.Start.UTC().Format(time.RFC3339Nano),
			definition.DimensionKey, definition.ObjectLDN, metric.MetricID, revision,
		)
		out = append(out, ProgressResult{
			ID:     uuid.NewSHA1(uuid.NameSpaceOID, []byte(identity)),
			TaskID: key.TaskID, TaskVersionID: key.TaskVersionID,
			Granularity: key.Granularity, WindowStart: key.Start, WindowEnd: key.End,
			Dimension: definition.Dimension, DimensionKey: definition.DimensionKey,
			DimensionName: definition.DimensionName, ObjectLDN: definition.ObjectLDN,
			DeviceOUI: definition.DeviceOUI, DeviceSN: definition.DeviceSN,
			MetricPath: definition.MetricPath, MetricType: metric.MetricType,
			Operation: metric.Operation, Value: metric.Value, Revision: revision,
			VersionEffectiveFrom: version.EffectiveFrom,
			VersionEffectiveTo:   version.EffectiveTo,
			ReceivedSlots:        received, ExpectedSlots: naturalExpected,
			VersionExpectedSlots: versionExpected, NaturalExpectedSlots: naturalExpected,
			VersionSliceComplete: false, PeriodComplete: false, Partial: true,
		})
	}
	return out, nil
}

func naturalExpectedSlots(
	version *TaskVersionSnapshot,
	key WindowKey,
	state WindowState,
) int64 {
	switch key.Granularity {
	case GranularityHourly:
		if version == nil || !version.DevicePipeline {
			return state.ExpectedSlots
		}
		return 4
	case GranularityDaily:
		return 24
	case GranularityWeekly:
		return 7
	case GranularityMonthly:
		location := key.Start.Location()
		start := key.Start.In(location)
		end := key.End.In(location)
		var days int64
		for cursor := start; cursor.Before(end); cursor = cursor.AddDate(0, 0, 1) {
			days++
		}
		return days
	default:
		return stateExpectedFallback(key)
	}
}

func stateExpectedFallback(key WindowKey) int64 {
	if !key.End.After(key.Start) {
		return 0
	}
	return int64(key.End.Sub(key.Start) / time.Hour)
}
