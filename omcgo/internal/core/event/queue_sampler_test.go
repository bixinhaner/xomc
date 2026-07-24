package event

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type queueStatsSourceStub struct {
	stats   QueueStats
	err     error
	calls   atomic.Int64
	called  chan struct{}
	subject string
	durable string
	block   bool
}

func (s *queueStatsSourceStub) QueueStats(ctx context.Context, subject, durable string) (QueueStats, error) {
	s.calls.Add(1)
	s.subject = subject
	s.durable = durable
	select {
	case s.called <- struct{}{}:
	default:
	}
	if err := ctx.Err(); err != nil {
		return QueueStats{}, err
	}
	if s.block {
		<-ctx.Done()
		return QueueStats{}, ctx.Err()
	}
	return s.stats, s.err
}

func TestQueueHealthSamplerRunsWithoutBackpressureOrStorageDependencies(t *testing.T) {
	source := &queueStatsSourceStub{
		stats:  QueueStats{Pending: 5, SampledAt: time.Now()},
		called: make(chan struct{}, 1),
	}
	metrics := NewEventBusMetrics(prometheus.NewRegistry())
	sampler := NewQueueHealthSampler(source, metrics, 10*time.Millisecond, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sampler.Run(ctx)
		close(done)
	}()
	select {
	case <-source.called:
	case <-time.After(time.Second):
		t.Fatal("queue health sampler did not make its startup sample")
	}

	assert.Equal(t, SubjectPMFileReceived, source.subject)
	assert.Equal(t, pmQueueStatsDurable, source.durable)
	assert.Equal(t, float64(5), testutil.ToFloat64(metrics.QueuePending.WithLabelValues(SubjectPMFileReceived, pmQueueStatsDurable)))
	assert.False(t, sampler.LastSuccessfulSample().IsZero())
	latest, ok := sampler.LatestStats()
	require.True(t, ok)
	assert.Equal(t, uint64(5), latest.Pending)

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("queue health sampler did not stop after context cancellation")
	}
	callsAfterStop := source.calls.Load()
	time.Sleep(25 * time.Millisecond)
	assert.Equal(t, callsAfterStop, source.calls.Load(), "canceled sampler must not continue sampling")
}

func TestQueueHealthSamplerOnlyObservesSuccessfulSamples(t *testing.T) {
	source := &queueStatsSourceStub{err: errors.New("nats unavailable"), called: make(chan struct{}, 1)}
	reg := prometheus.NewRegistry()
	metrics := NewEventBusMetrics(reg)
	sampler := NewQueueHealthSampler(source, metrics, time.Hour, zap.NewNop())

	err := sampler.Sample(context.Background())
	require.Error(t, err)
	assert.True(t, sampler.LastSuccessfulSample().IsZero())
	families, gatherErr := reg.Gather()
	require.NoError(t, gatherErr)
	assert.Empty(t, families)
}

func TestQueueHealthSamplerTimeoutDoesNotObserveOrCache(t *testing.T) {
	source := &queueStatsSourceStub{block: true, called: make(chan struct{}, 1)}
	reg := prometheus.NewRegistry()
	metrics := NewEventBusMetrics(reg)
	sampler := NewQueueHealthSampler(source, metrics, time.Hour, zap.NewNop())
	sampler.sampleTimeout = 10 * time.Millisecond

	err := sampler.Sample(context.Background())
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.True(t, sampler.LastSuccessfulSample().IsZero())
	_, ok := sampler.LatestStats()
	assert.False(t, ok)
	families, gatherErr := reg.Gather()
	require.NoError(t, gatherErr)
	assert.Empty(t, families)
}

func TestQueueHealthSamplerProjectionPendingCountRequiresFreshStats(t *testing.T) {
	sampler := NewQueueHealthSampler(nil, nil, time.Hour, zap.NewNop())

	_, err := sampler.ProjectionPendingCount()
	require.Error(t, err, "missing cached stats must preserve projection failure behavior")

	sampler.latestMu.Lock()
	sampler.latest = QueueStats{Pending: 7, AckPending: 2, SampledAt: time.Now()}
	sampler.hasLatest = true
	sampler.latestMu.Unlock()
	count, err := sampler.ProjectionPendingCount()
	require.NoError(t, err)
	assert.Equal(t, uint64(9), count)

	sampler.latestMu.Lock()
	sampler.latest.SampledAt = time.Now().Add(-queueHealthProjectionMaxAge - time.Millisecond)
	sampler.latestMu.Unlock()
	_, err = sampler.ProjectionPendingCount()
	require.Error(t, err, "stale cached stats must not be used for disk projection")
}
