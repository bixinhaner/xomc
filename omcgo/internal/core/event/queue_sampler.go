package event

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const defaultQueueHealthSampleInterval = 30 * time.Second
const defaultQueueHealthSampleTimeout = 5 * time.Second

// QueueStatsProvider is the read-only queue-health contract used by the
// independent PM sampler.
type QueueStatsProvider interface {
	QueueStats(ctx context.Context, subject, durable string) (QueueStats, error)
}

// QueueHealthSampler owns PM queue metric observation. Keeping this separate
// from disk projections makes queue health available even when MinIO, TSDB, or
// upload backpressure configuration is unavailable.
type QueueHealthSampler struct {
	provider      QueueStatsProvider
	metrics       *EventBusMetrics
	interval      time.Duration
	sampleTimeout time.Duration
	logger        *zap.Logger

	lastSuccessfulSample atomic.Int64
	latestMu             sync.RWMutex
	latest               QueueStats
	hasLatest            bool
}

func NewQueueHealthSampler(provider QueueStatsProvider, metrics *EventBusMetrics, interval time.Duration, logger *zap.Logger) *QueueHealthSampler {
	if interval <= 0 {
		interval = defaultQueueHealthSampleInterval
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &QueueHealthSampler{
		provider:      provider,
		metrics:       metrics,
		interval:      interval,
		sampleTimeout: defaultQueueHealthSampleTimeout,
		logger:        logger,
	}
}

// Sample collects and observes one PM queue-health point. Metrics and the
// last-success timestamp change only after a complete successful collection.
func (s *QueueHealthSampler) Sample(ctx context.Context) error {
	if s == nil || s.provider == nil {
		return fmt.Errorf("queue health sampler has no queue stats provider")
	}
	sampleCtx, cancel := context.WithTimeout(ctx, s.sampleTimeout)
	defer cancel()
	stats, err := s.provider.QueueStats(sampleCtx, SubjectPMFileReceived, pmQueueStatsDurable)
	if err != nil {
		return fmt.Errorf("sample PM queue health: %w", err)
	}
	if stats.SampledAt.IsZero() {
		return fmt.Errorf("sample PM queue health: missing sample timestamp")
	}
	s.metrics.observeQueueStats(SubjectPMFileReceived, pmQueueStatsDurable, stats)
	s.latestMu.Lock()
	s.latest = stats
	s.hasLatest = true
	s.latestMu.Unlock()
	s.lastSuccessfulSample.Store(stats.SampledAt.UnixNano())
	return nil
}

// LatestStats returns the last successfully collected queue sample. It lets
// compatibility callers reuse the independent sampler without issuing a
// second JetStream management query.
func (s *QueueHealthSampler) LatestStats() (QueueStats, bool) {
	if s == nil {
		return QueueStats{}, false
	}
	s.latestMu.RLock()
	defer s.latestMu.RUnlock()
	return s.latest, s.hasLatest
}

// LastSuccessfulSample returns zero until the first successful collection.
func (s *QueueHealthSampler) LastSuccessfulSample() time.Time {
	if s == nil {
		return time.Time{}
	}
	if timestamp := s.lastSuccessfulSample.Load(); timestamp != 0 {
		return time.Unix(0, timestamp)
	}
	return time.Time{}
}

// Run samples immediately and then at a fixed interval until ctx is canceled.
func (s *QueueHealthSampler) Run(ctx context.Context) {
	if s == nil || s.provider == nil {
		return
	}
	s.sampleAndLog(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sampleAndLog(ctx)
		}
	}
}

func (s *QueueHealthSampler) sampleAndLog(ctx context.Context) {
	if err := s.Sample(ctx); err != nil && ctx.Err() == nil {
		s.logger.Warn("PM queue health sample failed", zap.Error(err))
	}
}
