package pageconfig

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultScheduleScanInterval  = time.Minute
	defaultScheduleRunLimit      = defaultRunLimit
	defaultScheduleMaxRuns       = 20
	defaultScheduleRetryDelay    = 10 * time.Minute
	defaultResultCleanupInterval = 24 * time.Hour
)

// ScheduleLocker guards one scheduled profile/group/window across app replicas.
type ScheduleLocker interface {
	WithScheduleLock(ctx context.Context, key string, fn func(context.Context) error) (bool, error)
}

type ScheduleRunOptions struct {
	Locker     ScheduleLocker
	Limit      int
	MaxRuns    int
	RetryDelay time.Duration
	Location   *time.Location
}

type ScheduleRunSummary struct {
	Scanned   int
	Due       int
	Ran       int
	Skipped   int
	Failed    int
	LastError string
}

type ScheduleClock interface {
	Now() time.Time
}

type scheduleSystemClock struct{}

func (scheduleSystemClock) Now() time.Time { return time.Now() }

type Scheduler struct {
	svc         *Service
	locker      ScheduleLocker
	logger      *zap.Logger
	clock       ScheduleClock
	interval    time.Duration
	options     ScheduleRunOptions
	lastCleanup time.Time

	stopOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
}

type SchedulerOption func(*Scheduler)

func WithScheduleInterval(interval time.Duration) SchedulerOption {
	return func(s *Scheduler) {
		if interval > 0 {
			s.interval = interval
		}
	}
}

func WithScheduleClock(clock ScheduleClock) SchedulerOption {
	return func(s *Scheduler) {
		if clock != nil {
			s.clock = clock
		}
	}
}

func WithScheduleMaxRuns(maxRuns int) SchedulerOption {
	return func(s *Scheduler) {
		if maxRuns > 0 {
			s.options.MaxRuns = maxRuns
		}
	}
}

func NewScheduler(svc *Service, locker ScheduleLocker, logger *zap.Logger, opts ...SchedulerOption) *Scheduler {
	if logger == nil {
		logger = zap.NewNop()
	}
	s := &Scheduler{
		svc:      svc,
		locker:   locker,
		logger:   logger.Named("page-config-scheduler"),
		clock:    scheduleSystemClock{},
		interval: defaultScheduleScanInterval,
		options: ScheduleRunOptions{
			Locker:     locker,
			Limit:      defaultScheduleRunLimit,
			MaxRuns:    defaultScheduleMaxRuns,
			RetryDelay: defaultScheduleRetryDelay,
			Location:   time.Local,
		},
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

func (s *Scheduler) Start(ctx context.Context) {
	if s == nil || s.svc == nil {
		return
	}
	s.logger.Info("northbound page-config scheduler started",
		zap.Duration("interval", s.interval),
		zap.Int("max_runs", s.options.MaxRuns))
	s.runOnce(ctx)

	go func() {
		defer close(s.doneCh)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.logger.Info("northbound page-config scheduler context cancelled")
				return
			case <-s.stopCh:
				s.logger.Info("northbound page-config scheduler stopped")
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *Scheduler) Wait() {
	if s == nil {
		return
	}
	<-s.doneCh
}

func (s *Scheduler) runOnce(ctx context.Context) {
	now := s.clock.Now()
	summary, err := s.svc.RunDueSchedules(ctx, now, s.options)
	if err != nil {
		s.logger.Warn("northbound page-config schedule scan failed", zap.Error(err))
		return
	}
	if summary.Ran > 0 || summary.Failed > 0 {
		fields := []zap.Field{
			zap.Int("scanned", summary.Scanned),
			zap.Int("due", summary.Due),
			zap.Int("ran", summary.Ran),
			zap.Int("skipped", summary.Skipped),
			zap.Int("failed", summary.Failed),
		}
		if summary.LastError != "" {
			fields = append(fields, zap.String("last_error", summary.LastError))
		}
		s.logger.Info("northbound page-config schedule scan completed", fields...)
	}
	if s.lastCleanup.IsZero() || now.Sub(s.lastCleanup) >= defaultResultCleanupInterval {
		cleanup, err := s.svc.CleanupExpiredResults(ctx, ResultRetentionPolicy{})
		if err != nil {
			s.logger.Warn("northbound page-config result cleanup failed", zap.Error(err))
		} else if cleanup.RunsDeleted > 0 || cleanup.EventsDeleted > 0 {
			s.logger.Info("northbound page-config result cleanup completed",
				zap.Int64("runs_deleted", cleanup.RunsDeleted),
				zap.Int64("events_deleted", cleanup.EventsDeleted),
				zap.Int("run_retention_days", cleanup.RunRetentionDays),
				zap.Int("event_retention_days", cleanup.EventRetentionDays))
		}
		archiveCleanup, err := s.svc.CleanupLocalArchive(ctx, now)
		if err != nil {
			s.logger.Warn("northbound page-config local archive cleanup failed", zap.Error(err))
		} else if archiveCleanup.ObjectsDeleted > 0 {
			s.logger.Info("northbound page-config local archive cleanup completed",
				zap.String("bucket", archiveCleanup.Bucket),
				zap.String("prefix", archiveCleanup.Prefix),
				zap.Int64("objects_deleted", archiveCleanup.ObjectsDeleted),
				zap.Int64("bytes_deleted", archiveCleanup.BytesDeleted),
				zap.Int("retention_days", archiveCleanup.RetentionDays))
		}
		s.lastCleanup = now
	}
}

func (s *Service) RunDueSchedules(ctx context.Context, now time.Time, opts ScheduleRunOptions) (ScheduleRunSummary, error) {
	var summary ScheduleRunSummary
	if s.repo == nil {
		return summary, fmt.Errorf("northbound page-config repository is not configured")
	}
	if now.IsZero() {
		now = time.Now()
	}
	if opts.Limit <= 0 {
		opts.Limit = defaultScheduleRunLimit
	}
	if opts.MaxRuns <= 0 {
		opts.MaxRuns = defaultScheduleMaxRuns
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = defaultScheduleRetryDelay
	}
	if opts.Location == nil {
		opts.Location = time.Local
	}
	if err := s.repo.EnsureDefaults(ctx, s.catalog); err != nil {
		return summary, err
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return summary, err
	}

	units, err := s.scheduleUnits(ctx)
	if err != nil {
		return summary, err
	}
	summary.Scanned = len(units)
	for _, unit := range units {
		if summary.Ran >= opts.MaxRuns {
			break
		}
		candidate, due, err := s.nextScheduleCandidate(ctx, unit, now, opts)
		if err != nil {
			summary.Failed++
			summary.LastError = err.Error()
			continue
		}
		if !due {
			continue
		}
		summary.Due++
		ran, err := s.runScheduleCandidate(ctx, candidate, opts)
		if err != nil {
			summary.Failed++
			summary.LastError = err.Error()
			continue
		}
		if ran {
			summary.Ran++
		} else {
			summary.Skipped++
		}
	}
	return summary, nil
}

type scheduleUnit struct {
	Kind        ProfileKind
	ProfileCode string
	GroupID     string
	Period      Period
	StartMinute int
	ObjectCount int
}

type scheduleCandidate struct {
	scheduleUnit
	WindowStart time.Time
	WindowEnd   time.Time
	Limit       int
	Now         time.Time
}

func (s *Service) scheduleUnits(ctx context.Context) ([]scheduleUnit, error) {
	fileProfiles, err := s.ListFileProfiles(ctx)
	if err != nil {
		return nil, err
	}
	inventoryProfiles, err := s.ListInventoryProfiles(ctx)
	if err != nil {
		return nil, err
	}

	units := make([]scheduleUnit, 0)
	for _, profile := range fileProfiles {
		if !profile.Enabled || profile.Status != StatusNormal {
			continue
		}
		for _, group := range profile.Groups {
			objectCount := len(group.Objects)
			if objectCount == 0 {
				continue
			}
			units = append(units, scheduleUnit{
				Kind:        ProfileKindFile,
				ProfileCode: profile.Code,
				GroupID:     group.ID,
				Period:      group.Period,
				StartMinute: group.StartMinute,
				ObjectCount: objectCount,
			})
		}
	}
	for _, profile := range inventoryProfiles {
		if !profile.Enabled || profile.Status != StatusNormal {
			continue
		}
		units = append(units, scheduleUnit{
			Kind:        ProfileKindInventory,
			ProfileCode: profile.Code,
			GroupID:     profile.Code,
			Period:      profile.Period,
			StartMinute: profile.StartMinute,
			ObjectCount: 1,
		})
	}
	sort.Slice(units, func(i, j int) bool {
		if units[i].Kind != units[j].Kind {
			return units[i].Kind < units[j].Kind
		}
		if units[i].ProfileCode != units[j].ProfileCode {
			return units[i].ProfileCode < units[j].ProfileCode
		}
		return units[i].GroupID < units[j].GroupID
	})
	return units, nil
}

func (s *Service) nextScheduleCandidate(ctx context.Context, unit scheduleUnit, now time.Time, opts ScheduleRunOptions) (scheduleCandidate, bool, error) {
	windowStart, dueEnd, ok := dueWindowForPeriod(unit.Period, unit.StartMinute, now, opts.Location)
	if !ok {
		return scheduleCandidate{}, false, nil
	}
	runResult, err := s.repo.ListFileRuns(ctx, RunFilter{
		ProfileKind: unit.Kind,
		ProfileCode: unit.ProfileCode,
		Limit:       500,
	})
	if err != nil {
		return scheduleCandidate{}, false, err
	}

	statsByEnd := scheduleWindowStatsByEnd(runResult.Items, unit)
	latestCompleteEnd := time.Time{}
	for end, stats := range statsByEnd {
		if end.After(dueEnd) {
			continue
		}
		if stats.successCount >= unit.ObjectCount && end.After(latestCompleteEnd) {
			latestCompleteEnd = end
		}
	}
	candidateEnd := dueEnd
	if !latestCompleteEnd.IsZero() {
		candidateEnd = addPeriod(latestCompleteEnd.In(dueEnd.Location()), unit.Period)
	}
	if candidateEnd.After(dueEnd) {
		return scheduleCandidate{}, false, nil
	}
	if !candidateEnd.Equal(dueEnd) {
		windowStart = periodWindowStart(unit.Period, candidateEnd)
	}

	stats := statsByEnd[scheduleWindowKey(candidateEnd)]
	if stats.running || stats.successCount >= unit.ObjectCount {
		return scheduleCandidate{}, false, nil
	}
	if !stats.lastFailedAt.IsZero() && now.Sub(stats.lastFailedAt) < opts.RetryDelay {
		return scheduleCandidate{}, false, nil
	}
	return scheduleCandidate{
		scheduleUnit: unit,
		WindowStart:  windowStart,
		WindowEnd:    candidateEnd,
		Limit:        opts.Limit,
		Now:          now,
	}, true, nil
}

func (s *Service) runScheduleCandidate(ctx context.Context, candidate scheduleCandidate, opts ScheduleRunOptions) (bool, error) {
	executed := false
	run := func(runCtx context.Context) error {
		recheckNow := candidate.Now
		if recheckNow.IsZero() {
			recheckNow = time.Now()
		}
		rechecked, due, err := s.nextScheduleCandidate(runCtx, candidate.scheduleUnit, recheckNow, opts)
		if err != nil {
			return err
		}
		if !due || !sameMinute(rechecked.WindowEnd, candidate.WindowEnd) {
			return nil
		}
		req := RunProfileRequest{
			GroupID:       candidate.GroupID,
			WindowStart:   &candidate.WindowStart,
			WindowEnd:     &candidate.WindowEnd,
			Limit:         candidate.Limit,
			TriggerReason: runTriggerAuto,
		}
		if candidate.Kind == ProfileKindInventory {
			executed = true
			_, err = s.RunInventoryProfile(runCtx, candidate.ProfileCode, req)
			return err
		}
		executed = true
		_, err = s.RunFileProfile(runCtx, candidate.ProfileCode, req)
		return err
	}
	if opts.Locker == nil {
		err := run(ctx)
		return executed, err
	}
	acquired, err := opts.Locker.WithScheduleLock(ctx, scheduleLockKey(candidate), run)
	return acquired && executed, err
}

type scheduleWindowStats struct {
	successObjects map[string]struct{}
	successCount   int
	running        bool
	lastFailedAt   time.Time
}

func scheduleWindowStatsByEnd(runs []FileRun, unit scheduleUnit) map[time.Time]scheduleWindowStats {
	out := make(map[time.Time]scheduleWindowStats)
	for _, run := range runs {
		if run.GroupID != unit.GroupID || run.WindowEnd == nil {
			continue
		}
		end := scheduleWindowKey(*run.WindowEnd)
		stats := out[end]
		if stats.successObjects == nil {
			stats.successObjects = make(map[string]struct{})
		}
		switch run.Status {
		case RunStatusSuccess:
			markScheduleObjectComplete(stats.successObjects, run)
			stats.successCount = len(stats.successObjects)
		case RunStatusRunning:
			stats.running = true
		case RunStatusFailed:
			if run.CreatedAt.After(stats.lastFailedAt) {
				stats.lastFailedAt = run.CreatedAt
			}
		case RunStatusTerminated:
			if runNoArtifact(run) {
				markScheduleObjectComplete(stats.successObjects, run)
				stats.successCount = len(stats.successObjects)
			}
		}
		out[end] = stats
	}
	return out
}

func markScheduleObjectComplete(objects map[string]struct{}, run FileRun) {
	key := strings.TrimSpace(run.ObjectCode)
	if key == "" {
		key = run.ID
	}
	objects[key] = struct{}{}
}

func runNoArtifact(run FileRun) bool {
	if run.Summary == nil {
		return false
	}
	value, ok := run.Summary["no_artifact"].(bool)
	return ok && value
}

func scheduleWindowKey(t time.Time) time.Time {
	return t.UTC().Truncate(time.Minute)
}

func dueWindowForPeriod(period Period, startMinute int, now time.Time, loc *time.Location) (time.Time, time.Time, bool) {
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc).Truncate(time.Minute)
	delay := time.Duration(normalizeStartMinute(startMinute)) * time.Minute
	end := currentPeriodBoundary(period, now)
	if now.Before(end.Add(delay)) {
		end = previousPeriodBoundary(period, end)
	}
	if end.IsZero() || end.Add(delay).After(now) {
		return time.Time{}, time.Time{}, false
	}
	return periodWindowStart(period, end), end, true
}

func normalizeStartMinute(startMinute int) int {
	if startMinute < 0 {
		return 0
	}
	if startMinute > 59 {
		return 59
	}
	return startMinute
}

func currentPeriodBoundary(period Period, now time.Time) time.Time {
	switch period {
	case Period15M:
		return now.Truncate(15 * time.Minute)
	case Period60M:
		return now.Truncate(time.Hour)
	case Period24H:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case Period7D:
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		offset := (int(dayStart.Weekday()) + 6) % 7
		return dayStart.AddDate(0, 0, -offset)
	case Period1MO:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		return now.Truncate(15 * time.Minute)
	}
}

func previousPeriodBoundary(period Period, end time.Time) time.Time {
	switch period {
	case Period1MO:
		return end.AddDate(0, -1, 0)
	case Period7D:
		return end.AddDate(0, 0, -7)
	case Period24H:
		return end.AddDate(0, 0, -1)
	default:
		return end.Add(-periodDuration(period))
	}
}

func addPeriod(end time.Time, period Period) time.Time {
	switch period {
	case Period1MO:
		return end.AddDate(0, 1, 0)
	case Period7D:
		return end.AddDate(0, 0, 7)
	case Period24H:
		return end.AddDate(0, 0, 1)
	default:
		return end.Add(periodDuration(period))
	}
}

func periodWindowStart(period Period, end time.Time) time.Time {
	return previousPeriodBoundary(period, end)
}

func sameMinute(a, b time.Time) bool {
	return a.Truncate(time.Minute).Equal(b.Truncate(time.Minute))
}

func scheduleLockKey(candidate scheduleCandidate) string {
	return fmt.Sprintf("%s:%s:%s:%s",
		candidate.Kind,
		candidate.ProfileCode,
		candidate.GroupID,
		candidate.WindowEnd.UTC().Format("200601021504"))
}

func (r *PgRepository) WithScheduleLock(ctx context.Context, key string, fn func(context.Context) error) (bool, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("acquire northbound schedule lock connection: %w", err)
	}
	defer conn.Release()

	lockID := scheduleAdvisoryLockID(key)
	var acquired bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockID).Scan(&acquired); err != nil {
		return false, fmt.Errorf("try northbound schedule advisory lock: %w", err)
	}
	if !acquired {
		return false, nil
	}
	defer func() {
		var unlocked bool
		_ = conn.QueryRow(context.Background(), `SELECT pg_advisory_unlock($1)`, lockID).Scan(&unlocked)
	}()
	return true, fn(ctx)
}

func scheduleAdvisoryLockID(key string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("northbound-page-config:"))
	_, _ = h.Write([]byte(key))
	return int64(h.Sum64())
}
