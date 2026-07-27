package rawcleanup

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Mode                string
	BatchSize           int
	RetentionDays       int
	MinRate             float64
	MaxRate             float64
	Headroom            float64
	RecalculateInterval time.Duration
	ObjectTimeout       time.Duration
	PredictedRate       func(context.Context) float64
	ModeLookup          func(context.Context) string
	RetentionDaysLookup func(context.Context) int
}

type Runner struct {
	repo          Repository
	deleter       ObjectDeleter
	cfg           Config
	probe         PressureProbe
	metrics       *Metrics
	logger        *zap.Logger
	now           func() time.Time
	slowDelete    bool
	pauseUntil    time.Time
	minioBackoff  time.Duration
	currentMode   string
	retentionDays int
}

func NewRunner(repo Repository, deleter ObjectDeleter, cfg Config) *Runner {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 60
	}
	if cfg.MinRate <= 0 {
		cfg.MinRate = 5
	}
	if cfg.MaxRate <= 0 {
		cfg.MaxRate = 100
	}
	if cfg.Headroom <= 0 {
		cfg.Headroom = 1.5
	}
	if cfg.RecalculateInterval <= 0 {
		cfg.RecalculateInterval = 5 * time.Minute
	}
	if cfg.ObjectTimeout <= 0 {
		cfg.ObjectTimeout = 10 * time.Second
	}
	return &Runner{
		repo: repo, deleter: deleter, cfg: cfg, logger: zap.NewNop(), now: time.Now,
		currentMode: cfg.Mode, retentionDays: cfg.RetentionDays,
	}
}

func (r *Runner) SetPressureProbe(probe PressureProbe) { r.probe = probe }
func (r *Runner) SetMetrics(metrics *Metrics)          { r.metrics = metrics }
func (r *Runner) SetLogger(logger *zap.Logger) {
	if logger != nil {
		r.logger = logger.Named("raw-cleanup")
	}
}

func (r *Runner) RunOnce(ctx context.Context) (int, error) {
	lockCtx, cancelLock := context.WithTimeout(ctx, 5*time.Second)
	lease, acquired, err := r.repo.TryLock(lockCtx)
	cancelLock()
	if err != nil || !acquired {
		return 0, err
	}
	defer func() {
		releaseCtx, cancelRelease := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelRelease()
		lease.Release(releaseCtx)
	}()
	now := r.now()
	r.refreshRetentionDays(ctx)
	cutoff := now.AddDate(0, 0, -r.retentionDays)
	listCtx, cancelList := context.WithTimeout(ctx, 5*time.Second)
	candidates, err := r.repo.ListCandidates(listCtx, cutoff, r.cfg.BatchSize)
	cancelList()
	if err != nil {
		return 0, fmt.Errorf("list raw cleanup candidates: %w", err)
	}
	if r.metrics != nil {
		counts := map[Kind]int{}
		for _, candidate := range candidates {
			counts[candidate.Kind]++
		}
		for _, kind := range []Kind{KindPM, KindMR} {
			r.metrics.Candidates.WithLabelValues(string(kind)).Set(float64(counts[kind]))
		}
	}
	mode := r.currentMode
	if r.cfg.ModeLookup != nil {
		modeCtx, cancelMode := context.WithTimeout(ctx, 2*time.Second)
		current := r.cfg.ModeLookup(modeCtx)
		cancelMode()
		if current != "" {
			mode = current
			r.currentMode = current
		}
	}
	if mode != "fallback" && mode != "exclusive" {
		return len(candidates), nil
	}
	if len(candidates) == 0 {
		metadataCtx, cancelMetadata := context.WithTimeout(ctx, 5*time.Second)
		_, err := r.repo.CleanupMetadata(metadataCtx, cutoff, r.cfg.BatchSize)
		cancelMetadata()
		if err != nil {
			return 0, fmt.Errorf("cleanup dependency-safe raw metadata: %w", err)
		}
		return 0, nil
	}
	leaseCtx, cancelLease := context.WithTimeout(ctx, 2*time.Second)
	err = lease.Valid(leaseCtx)
	cancelLease()
	if err != nil {
		return 0, fmt.Errorf("raw cleanup lock lost before object deletion: %w", err)
	}
	requestCtx, cancel := context.WithTimeout(ctx, r.cfg.ObjectTimeout)
	start := time.Now()
	results := r.deleter.Delete(requestCtx, candidates)
	cancel()
	duration := time.Since(start)
	r.slowDelete = duration > 2*time.Second
	throttled := false
	failed := 0
	for _, result := range results {
		if result.Err == nil {
			continue
		}
		failed++
		message := strings.ToLower(result.Err.Error())
		if strings.Contains(message, "503") || strings.Contains(message, "slowdown") ||
			strings.Contains(message, "throttl") || strings.Contains(message, "deadline") {
			throttled = true
		}
	}
	if len(results) > 0 && failed*2 >= len(results) {
		throttled = true
	}
	if throttled {
		if r.minioBackoff <= 0 {
			r.minioBackoff = time.Minute
		} else {
			r.minioBackoff *= 2
		}
		if r.minioBackoff > 5*time.Minute {
			r.minioBackoff = 5 * time.Minute
		}
		r.pauseUntil = now.Add(r.minioBackoff)
	} else if len(results) > 0 {
		r.minioBackoff = 0
	}
	leaseCtx, cancelLease = context.WithTimeout(ctx, 2*time.Second)
	err = lease.Valid(leaseCtx)
	cancelLease()
	if err != nil {
		return 0, fmt.Errorf("raw cleanup lock lost before status update: %w", err)
	}
	markCtx, cancelMark := context.WithTimeout(ctx, 5*time.Second)
	err = r.repo.MarkResults(markCtx, results, now)
	cancelMark()
	if err != nil {
		return 0, fmt.Errorf("mark raw cleanup results: %w", err)
	}
	metadataCtx, cancelMetadata := context.WithTimeout(ctx, 5*time.Second)
	_, err = r.repo.CleanupMetadata(metadataCtx, cutoff, r.cfg.BatchSize)
	cancelMetadata()
	if err != nil {
		return 0, fmt.Errorf("cleanup dependency-safe raw metadata: %w", err)
	}
	if r.metrics != nil {
		r.metrics.Duration.Observe(duration.Seconds())
		for _, result := range results {
			if result.Err == nil {
				r.metrics.Deleted.WithLabelValues(string(result.Candidate.Kind)).Inc()
			} else {
				r.metrics.Failed.WithLabelValues(string(result.Candidate.Kind), "object_store").Inc()
			}
		}
	}
	return len(results), nil
}

func (r *Runner) Run(ctx context.Context) {
	jitter := time.NewTimer(time.Duration(rand.Intn(61)) * time.Second)
	select {
	case <-ctx.Done():
		jitter.Stop()
		return
	case <-jitter.C:
	}
	rate := r.cfg.MinRate
	nextRecalc := time.Time{}
	for ctx.Err() == nil {
		now := r.now()
		if now.After(nextRecalc) {
			rateCtx, cancelRate := context.WithTimeout(ctx, 5*time.Second)
			count, err := r.repo.RecentCreatedCount(rateCtx, now.Add(-time.Hour))
			cancelRate()
			observed := float64(count) / 3600
			predicted := 0.0
			if r.cfg.PredictedRate != nil {
				predictCtx, cancelPredict := context.WithTimeout(ctx, 2*time.Second)
				predicted = r.cfg.PredictedRate(predictCtx)
				cancelPredict()
			}
			if err != nil {
				r.logger.Warn("recent raw file rate unavailable", zap.Error(err))
			} else {
				rate = TargetRate(predicted, observed, r.cfg.Headroom, r.cfg.MinRate, r.cfg.MaxRate)
			}
			r.refreshRetentionDays(ctx)
			cutoff := now.AddDate(0, 0, -r.retentionDays)
			for _, kind := range []Kind{KindPM, KindMR} {
				oldestCtx, cancelOldest := context.WithTimeout(ctx, 5*time.Second)
				oldest, oldestErr := r.repo.OldestExpired(oldestCtx, cutoff, kind)
				cancelOldest()
				if oldestErr != nil {
					r.logger.Warn("oldest raw cleanup candidate unavailable", zap.String("kind", string(kind)), zap.Error(oldestErr))
					continue
				}
				lag := cutoff.Sub(oldest)
				if lag < 0 {
					lag = 0
				}
				if r.metrics != nil {
					r.metrics.OldestAge.WithLabelValues(string(kind)).Set(lag.Seconds())
				}
				if lag > 24*time.Hour {
					r.logger.Warn("raw cleanup expiration lag exceeds SLO", zap.String("kind", string(kind)), zap.Duration("lag", lag))
				}
			}
			nextRecalc = now.Add(r.cfg.RecalculateInterval)
		}
		pressure := Pressure{}
		if r.probe != nil {
			pressure = r.probe.Sample(ctx)
		}
		pressure.SlowDelete = r.slowDelete
		decision := ApplyPressure(rate, pressure, now)
		if now.Before(r.pauseUntil) {
			decision = Decision{Paused: true, Reason: "minio_backoff"}
		}
		if r.metrics != nil {
			r.metrics.Rate.Set(decision.Rate)
			for _, reason := range []string{
				"nats_backlog", "nats_unavailable", "disk_overload", "monitoring_unavailable",
				"cpu", "disk", "minio_slow", "pm_boundary", "minio_backoff",
			} {
				r.metrics.Paused.WithLabelValues(reason).Set(0)
			}
			if decision.Reason != "" {
				v := 0.0
				if decision.Paused {
					v = 1
				}
				r.metrics.Paused.WithLabelValues(decision.Reason).Set(v)
			}
		}
		if !decision.Paused {
			if _, err := r.RunOnce(ctx); err != nil {
				r.logger.Warn("raw object cleanup batch failed", zap.Error(err))
			}
		}
		wait := BatchInterval(r.cfg.BatchSize, decision.Rate)
		if decision.Paused {
			wait = time.Minute
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (r *Runner) refreshRetentionDays(ctx context.Context) {
	if r.cfg.RetentionDaysLookup == nil {
		return
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	days := r.cfg.RetentionDaysLookup(lookupCtx)
	cancel()
	if days > 0 {
		r.retentionDays = days
	}
}
