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
	"github.com/omcgo/omcgo/internal/pm/calendarfilter"
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
	TaskID               uuid.UUID   `json:"task_id"`
	TaskVersionID        uuid.UUID   `json:"task_version_id"`
	Granularity          Granularity `json:"granularity"`
	WindowStart          time.Time   `json:"window_start"`
	WindowEnd            time.Time   `json:"window_end"`
	EntityKey            string      `json:"entity_key"`
	Revision             int         `json:"revision"`
	VersionEffectiveFrom time.Time   `json:"version_effective_from"`
	VersionEffectiveTo   *time.Time  `json:"version_effective_to"`
	ReceivedSlots        int64       `json:"received_slots"`
	ExpectedSlots        int64       `json:"expected_slots"`
	VersionExpectedSlots int64       `json:"version_expected_slots"`
	CoverageRatio        float64     `json:"coverage_ratio"`
	VersionSliceComplete bool        `json:"version_slice_complete"`
	PeriodComplete       bool        `json:"period_complete"`
	State                string      `json:"state"`
}

type MetricVersionInterval struct {
	MetricPath    string     `json:"metric_path"`
	EffectiveFrom time.Time  `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to"`
}

type ProgressQueryResult struct {
	Rows            []ProgressResult
	Periods         []PeriodProgress
	MetricIntervals []MetricVersionInterval
}

type ProgressService struct {
	pool     *pgxpool.Pool
	store    *RedisWindowStore
	snapshot *SnapshotStore
	timezone calendarfilter.TimezoneProvider
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

type weeklyPreviewCoverage struct {
	received        int64
	naturalExpected int64
	versionExpected int64
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
	return NewProgressServiceWithSnapshot(
		pool, store, NewSnapshotStore(loader, nil),
	)
}

func NewProgressServiceWithSnapshot(
	pool *pgxpool.Pool,
	store *RedisWindowStore,
	snapshot *SnapshotStore,
) *ProgressService {
	return &ProgressService{
		pool: pool, store: store, snapshot: snapshot,
	}
}

func (s *ProgressService) SetTimezoneProvider(
	timezone calendarfilter.TimezoneProvider,
) *ProgressService {
	s.timezone = timezone
	return s
}

func (s *ProgressService) Query(
	ctx context.Context,
	taskID uuid.UUID,
	start, end time.Time,
) (ProgressQueryResult, error) {
	queryCtx, cancel := context.WithTimeout(ctx, progressQueryTimeout)
	defer cancel()
	ctx = queryCtx
	snapshot, err := s.loadSnapshot(ctx)
	if err != nil {
		return ProgressQueryResult{}, err
	}
	metricIntervals := metricVersionIntervalsFromSnapshot(snapshot, taskID)
	builder := storage.Psql.Select(
		"task_id", "task_version_id", "entity_key", "granularity",
		"window_start", "window_end", "revision", "status",
		"version_effective_from", "opened_at", "received_slots", "expected_slots",
	).From("pm_aggregation_windows").
		Where(sq.Eq{
			"task_id":     taskID,
			"status":      []string{"open", "finalizing", "prepared", "failed", "rebuilding"},
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
	activeCandidates := selectActiveProgressCandidates(candidates, snapshot)
	states := make(map[WindowKey]WindowState, len(activeCandidates))
	for _, item := range activeCandidates {
		if progressStatusUnavailable(item.status) {
			return ProgressQueryResult{}, fmt.Errorf(
				"%w: window %s/%s is failed",
				ErrProgressUnavailable, item.key.Granularity, item.key.Start,
			)
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
		states[item.key] = state
	}
	if err := s.revalidateOpenDailyCandidates(ctx, taskID, activeCandidates); err != nil {
		return ProgressQueryResult{}, err
	}
	location := time.UTC
	if s.timezone != nil {
		if configured := s.timezone.Location(ctx); configured != nil {
			location = configured
		}
	}
	result, err := buildProgressQueryResult(
		activeCandidates, states, snapshot, location,
	)
	if err != nil {
		return ProgressQueryResult{}, err
	}
	result.MetricIntervals = metricIntervals
	return result, nil
}

func (s *ProgressService) loadSnapshot(ctx context.Context) (*TaskSnapshot, error) {
	if s.snapshot == nil {
		return nil, fmt.Errorf("PM aggregation progress snapshot is not configured")
	}
	if err := s.snapshot.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("load PM aggregation versions for progress: %w", err)
	}
	return s.snapshot.Current(), nil
}

func metricVersionIntervalsFromSnapshot(
	snapshot *TaskSnapshot,
	taskID uuid.UUID,
) []MetricVersionInterval {
	if snapshot == nil {
		return nil
	}
	versions := make([]*TaskVersionSnapshot, 0, len(snapshot.ByVersion))
	for _, version := range snapshot.ByVersion {
		versions = append(versions, version)
	}
	return metricVersionIntervals(versions, taskID)
}

func metricVersionIntervals(
	versions []*TaskVersionSnapshot,
	taskID uuid.UUID,
) []MetricVersionInterval {
	out := make([]MetricVersionInterval, 0)
	for _, version := range versions {
		if version == nil || version.TaskID != taskID || !version.Enabled {
			continue
		}
		for metricPath := range version.Metrics {
			out = append(out, MetricVersionInterval{
				MetricPath: metricPath, EffectiveFrom: version.EffectiveFrom,
				EffectiveTo: version.EffectiveTo,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MetricPath != out[j].MetricPath {
			return out[i].MetricPath < out[j].MetricPath
		}
		return out[i].EffectiveFrom.Before(out[j].EffectiveFrom)
	})
	return out
}

func (s *ProgressService) revalidateOpenDailyCandidates(
	ctx context.Context,
	taskID uuid.UUID,
	candidates []progressCandidate,
) error {
	versionSet := make(map[uuid.UUID]struct{})
	var minStart, maxEnd time.Time
	for _, item := range candidates {
		if item.key.Granularity != GranularityDaily || item.status != "open" {
			continue
		}
		versionSet[item.key.TaskVersionID] = struct{}{}
		if minStart.IsZero() || item.key.Start.Before(minStart) {
			minStart = item.key.Start
		}
		if maxEnd.IsZero() || item.key.End.After(maxEnd) {
			maxEnd = item.key.End
		}
	}
	if len(versionSet) == 0 {
		return nil
	}
	versionIDs := make([]uuid.UUID, 0, len(versionSet))
	for versionID := range versionSet {
		versionIDs = append(versionIDs, versionID)
	}
	query, args, err := storage.Psql.Select(
		"task_id", "task_version_id", "entity_key", "window_start", "status",
	).From("pm_aggregation_windows").
		Where(sq.Eq{
			"task_id":         taskID,
			"task_version_id": versionIDs,
			"granularity":     string(GranularityDaily),
		}).
		Where(sq.GtOrEq{"window_start": minStart}).
		Where(sq.Lt{"window_start": maxEnd}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build PM progress status recheck: %w", err)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf(
			"%w: recheck PM progress window status: %v",
			ErrProgressUnavailable, err,
		)
	}
	defer rows.Close()
	statuses := make(map[string]string)
	for rows.Next() {
		var key WindowKey
		var status string
		if err := rows.Scan(
			&key.TaskID, &key.TaskVersionID, &key.EntityKey, &key.Start, &status,
		); err != nil {
			return fmt.Errorf(
				"%w: scan PM progress status recheck: %v",
				ErrProgressUnavailable, err,
			)
		}
		key.Granularity = GranularityDaily
		statuses[progressCandidateIdentity(key)] = status
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf(
			"%w: iterate PM progress status recheck: %v",
			ErrProgressUnavailable, err,
		)
	}
	return validateOpenDailyCandidateStatuses(candidates, statuses)
}

func progressCandidateIdentity(key WindowKey) string {
	return fmt.Sprintf(
		"%s|%s|%s|%d",
		key.TaskVersionID, key.EntityKey, key.Granularity,
		key.Start.UTC().UnixNano(),
	)
}

func validateOpenDailyCandidateStatuses(
	candidates []progressCandidate,
	statuses map[string]string,
) error {
	for _, item := range candidates {
		if item.key.Granularity != GranularityDaily || item.status != "open" {
			continue
		}
		status, ok := statuses[progressCandidateIdentity(item.key)]
		if !ok || status != "open" {
			return fmt.Errorf(
				"%w: daily window changed from open to %q during progress read",
				ErrProgressUnavailable, status,
			)
		}
	}
	return nil
}

func selectActiveProgressCandidates(
	candidates []progressCandidate,
	snapshot *TaskSnapshot,
) []progressCandidate {
	activeVersions := make(map[string]progressCandidate)
	for _, item := range candidates {
		group := progressVersionGroup(item.key)
		current, exists := activeVersions[group]
		if !exists || progressVersionAfter(item, current, snapshot) {
			activeVersions[group] = item
		}
	}
	active := make([]progressCandidate, 0, len(candidates))
	for _, item := range candidates {
		selected := activeVersions[progressVersionGroup(item.key)]
		if item.key.TaskVersionID == selected.key.TaskVersionID {
			active = append(active, item)
		}
	}
	return active
}

type weeklyPreviewGroup struct {
	key       WindowKey
	dailies   []progressCandidate
	persisted *progressCandidate
	duplicate []progressCandidate
}

func buildProgressQueryResult(
	candidates []progressCandidate,
	states map[WindowKey]WindowState,
	snapshot *TaskSnapshot,
	location *time.Location,
) (ProgressQueryResult, error) {
	if location == nil {
		location = time.UTC
	}
	candidates = selectActiveProgressCandidates(candidates, snapshot)
	weeklyGroups := make(map[string]*weeklyPreviewGroup)
	for index := range candidates {
		item := candidates[index]
		version := snapshot.ByVersion[item.key.TaskVersionID]
		if version == nil || !containsGranularity(version.Granularities, GranularityWeekly) {
			continue
		}
		switch item.key.Granularity {
		case GranularityDaily:
			// A finalizing daily window may have committed its rollup after this
			// query read the old status but before Redis states are read. Only
			// open windows are guaranteed not to exist in the weekly state yet.
			if item.status != "open" {
				continue
			}
			window, err := WindowFor(item.key.Start, GranularityWeekly, location)
			if err != nil {
				return ProgressQueryResult{}, err
			}
			weeklyKey := item.key
			weeklyKey.Granularity, weeklyKey.Start, weeklyKey.End =
				GranularityWeekly, window.Start, window.End
			groupKey := progressEntityWindowGroup(weeklyKey)
			group := weeklyGroups[groupKey]
			if group == nil {
				group = &weeklyPreviewGroup{key: weeklyKey}
				weeklyGroups[groupKey] = group
			}
			group.dailies = append(group.dailies, item)
		case GranularityWeekly:
			normalized, err := normalizedWeeklyProgressKey(item.key, location)
			if err != nil {
				return ProgressQueryResult{}, err
			}
			groupKey := progressEntityWindowGroup(normalized)
			group := weeklyGroups[groupKey]
			if group == nil {
				group = &weeklyPreviewGroup{key: normalized}
				weeklyGroups[groupKey] = group
			}
			copy := item
			if item.key.Start.Equal(normalized.Start) &&
				item.key.End.Equal(normalized.End) {
				group.persisted = &copy
			} else {
				group.duplicate = append(group.duplicate, copy)
			}
		}
	}

	replacedWeekly := make(map[string]struct{})
	var out []ProgressResult
	periods := make(map[string]PeriodProgress)
	weeklyGroupKeys := make([]string, 0, len(weeklyGroups))
	for key := range weeklyGroups {
		weeklyGroupKeys = append(weeklyGroupKeys, key)
	}
	sort.Strings(weeklyGroupKeys)
	for _, groupKey := range weeklyGroupKeys {
		group := weeklyGroups[groupKey]
		if len(group.dailies) == 0 {
			if group.persisted != nil {
				for _, item := range group.duplicate {
					replacedWeekly[progressEntityWindowGroup(item.key)] = struct{}{}
				}
			}
			continue
		}
		version := snapshot.ByVersion[group.key.TaskVersionID]
		dailyStates := make([]WindowState, 0, len(group.dailies))
		revision := 0
		for _, daily := range group.dailies {
			state, ok := states[daily.key]
			if !ok {
				return ProgressQueryResult{}, fmt.Errorf(
					"%w: state missing for %s/%s",
					ErrProgressUnavailable, daily.key.Granularity, daily.key.Start,
				)
			}
			dailyStates = append(dailyStates, state)
			revision = max(revision, daily.revision)
		}
		var persistedState WindowState
		if group.persisted != nil {
			state, ok := states[group.persisted.key]
			if !ok {
				return ProgressQueryResult{}, fmt.Errorf(
					"%w: state missing for %s/%s",
					ErrProgressUnavailable,
					group.persisted.key.Granularity,
					group.persisted.key.Start,
				)
			}
			persistedState = state
			revision = max(revision, group.persisted.revision)
		}
		key, preview, coverage, err := buildCurrentWeeklyPreviewFromDailies(
			version, group.dailies[0].key, dailyStates, persistedState, location,
		)
		if err != nil {
			return ProgressQueryResult{}, err
		}
		results, err := BuildProgressResults(version, key, revision, preview)
		if err != nil {
			return ProgressQueryResult{}, err
		}
		for index := range results {
			results[index].ReceivedSlots = coverage.received
			results[index].ExpectedSlots = coverage.naturalExpected
			results[index].NaturalExpectedSlots = coverage.naturalExpected
			results[index].VersionExpectedSlots = coverage.versionExpected
		}
		out = appendProgressResults(out, results)
		addProgressPeriod(periods, PeriodProgress{
			TaskID: key.TaskID, TaskVersionID: key.TaskVersionID,
			Granularity: GranularityWeekly,
			WindowStart: key.Start, WindowEnd: key.End,
			EntityKey: key.EntityKey, Revision: revision,
			VersionEffectiveFrom: version.EffectiveFrom,
			VersionEffectiveTo:   version.EffectiveTo,
			ReceivedSlots:        coverage.received,
			ExpectedSlots:        coverage.naturalExpected,
			VersionExpectedSlots: coverage.versionExpected,
			CoverageRatio: progressCoverageRatio(
				coverage.received, coverage.naturalExpected,
			),
			VersionSliceComplete: coverage.versionExpected > 0 &&
				coverage.received >= coverage.versionExpected,
			PeriodComplete: false,
			State:          "partial",
		})
		if group.persisted != nil {
			replacedWeekly[progressEntityWindowGroup(group.persisted.key)] = struct{}{}
		}
		for _, item := range group.duplicate {
			replacedWeekly[progressEntityWindowGroup(item.key)] = struct{}{}
		}
	}

	for _, item := range candidates {
		if item.key.Granularity == GranularityWeekly {
			if _, replaced := replacedWeekly[progressEntityWindowGroup(item.key)]; replaced {
				continue
			}
		}
		state, ok := states[item.key]
		if !ok {
			return ProgressQueryResult{}, fmt.Errorf(
				"%w: state missing for %s/%s",
				ErrProgressUnavailable, item.key.Granularity, item.key.Start,
			)
		}
		version := snapshot.ByVersion[item.key.TaskVersionID]
		results, err := BuildProgressResults(version, item.key, item.revision, state)
		if err != nil {
			return ProgressQueryResult{}, err
		}
		out = appendProgressResults(out, results)
		expected := naturalExpectedSlots(
			version, item.key, WindowState{ExpectedSlots: item.expected},
		)
		period := PeriodProgress{
			TaskID: item.key.TaskID, TaskVersionID: item.key.TaskVersionID,
			Granularity: item.key.Granularity,
			WindowStart: item.key.Start, WindowEnd: item.key.End,
			EntityKey: item.key.EntityKey, Revision: item.revision,
			ReceivedSlots: item.received, ExpectedSlots: expected,
			VersionExpectedSlots: item.expected,
			CoverageRatio:        progressCoverageRatio(item.received, expected),
			VersionSliceComplete: item.expected > 0 && item.received >= item.expected,
			PeriodComplete:       false,
			State:                "partial",
		}
		if version != nil {
			period.VersionEffectiveFrom = version.EffectiveFrom
			period.VersionEffectiveTo = version.EffectiveTo
		}
		addProgressPeriod(periods, period)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Granularity != out[j].Granularity {
			return out[i].Granularity < out[j].Granularity
		}
		if !out[i].WindowStart.Equal(out[j].WindowStart) {
			return out[i].WindowStart.Before(out[j].WindowStart)
		}
		if out[i].DimensionKey != out[j].DimensionKey {
			return out[i].DimensionKey < out[j].DimensionKey
		}
		return out[i].MetricPath < out[j].MetricPath
	})
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

func progressCoverageRatio(received, expected int64) float64 {
	if expected <= 0 {
		return 0
	}
	return min(1, float64(received)/float64(expected))
}

func appendProgressResults(
	current []ProgressResult,
	incoming []ProgressResult,
) []ProgressResult {
	remaining := maxProgressResults - len(current)
	if remaining <= 0 {
		return current
	}
	if len(incoming) > remaining {
		incoming = incoming[:remaining]
	}
	return append(current, incoming...)
}

func addProgressPeriod(
	periods map[string]PeriodProgress,
	period PeriodProgress,
) {
	group := fmt.Sprintf("%s|%d", period.Granularity, period.WindowStart.UTC().UnixNano())
	if current, ok := periods[group]; !ok || lowerProgressCoverage(period, current) {
		periods[group] = period
	}
}

func progressEntityWindowGroup(key WindowKey) string {
	return fmt.Sprintf(
		"%s|%s|%s|%s|%d",
		key.TaskID, key.TaskVersionID, key.EntityKey,
		key.Granularity, key.Start.UTC().UnixNano(),
	)
}

func normalizedWeeklyProgressKey(
	key WindowKey,
	location *time.Location,
) (WindowKey, error) {
	if key.Granularity != GranularityWeekly {
		return key, nil
	}
	window, err := WindowFor(key.Start, GranularityWeekly, location)
	if err != nil {
		return WindowKey{}, err
	}
	key.Start, key.End = window.Start, window.End
	return key, nil
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

func buildCurrentWeeklyPreview(
	version *TaskVersionSnapshot,
	dailyKey WindowKey,
	dailyState WindowState,
	weeklyState WindowState,
	location *time.Location,
) (WindowKey, WindowState, weeklyPreviewCoverage, error) {
	return buildCurrentWeeklyPreviewFromDailies(
		version, dailyKey, []WindowState{dailyState}, weeklyState, location,
	)
}

func buildCurrentWeeklyPreviewFromDailies(
	version *TaskVersionSnapshot,
	dailyKey WindowKey,
	dailyStates []WindowState,
	weeklyState WindowState,
	location *time.Location,
) (WindowKey, WindowState, weeklyPreviewCoverage, error) {
	if version == nil {
		return WindowKey{}, WindowState{}, weeklyPreviewCoverage{},
			fmt.Errorf("PM aggregation task version snapshot missing")
	}
	if location == nil {
		location = time.UTC
	}
	window, err := WindowFor(dailyKey.Start, GranularityWeekly, location)
	if err != nil {
		return WindowKey{}, WindowState{}, weeklyPreviewCoverage{}, err
	}
	key := WindowKey{
		TaskID: dailyKey.TaskID, TaskVersionID: dailyKey.TaskVersionID,
		EntityKey: dailyKey.EntityKey, Granularity: GranularityWeekly,
		Start: window.Start, End: window.End,
	}

	accumulatorCapacity := len(weeklyState.Accumulators)
	for _, dailyState := range dailyStates {
		accumulatorCapacity += len(dailyState.Accumulators)
	}
	accumulators := make(map[string]*Accumulator, accumulatorCapacity)
	merge := func(items []Accumulator) error {
		for _, incoming := range items {
			id, err := accumulatorDefinitionID(incoming.Definition)
			if err != nil {
				return fmt.Errorf("identify PM weekly preview accumulator: %w", err)
			}
			current := accumulators[id]
			if current == nil {
				copy := incoming
				accumulators[id] = &copy
				continue
			}
			if incoming.Count <= 0 {
				continue
			}
			if current.Count <= 0 {
				current.Min, current.Max = incoming.Min, incoming.Max
			} else {
				current.Min = min(current.Min, incoming.Min)
				current.Max = max(current.Max, incoming.Max)
			}
			current.Sum += incoming.Sum
			current.Count += incoming.Count
		}
		return nil
	}
	if err := merge(weeklyState.Accumulators); err != nil {
		return WindowKey{}, WindowState{}, weeklyPreviewCoverage{}, err
	}
	var dailyReceived int64
	for _, dailyState := range dailyStates {
		if err := merge(dailyState.Accumulators); err != nil {
			return WindowKey{}, WindowState{}, weeklyPreviewCoverage{}, err
		}
		dailyReceived += dailyState.ReceivedSlots
	}

	ids := make([]string, 0, len(accumulators))
	for id := range accumulators {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	preview := WindowState{
		Accumulators: make([]Accumulator, 0, len(ids)),
	}
	for _, id := range ids {
		preview.Accumulators = append(preview.Accumulators, *accumulators[id])
	}

	coverage := weeklyPreviewCoverage{
		received:        weeklyState.ReceivedSlots*24 + dailyReceived,
		naturalExpected: 7 * 24,
		versionExpected: expectedVersionChildWindows(
			window, GranularityHourly, version, location,
		),
	}
	preview.ReceivedSlots = coverage.received
	preview.ExpectedSlots = coverage.versionExpected
	return key, preview, coverage, nil
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
		return 7 * 24
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
