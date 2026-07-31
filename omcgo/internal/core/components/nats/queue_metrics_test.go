package nats

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type queueMetricsReaderStub struct {
	mu sync.RWMutex

	streams   map[string]*gonats.StreamInfo
	consumers map[string]*gonats.ConsumerInfo
	oldest    map[string]*gonats.RawStreamMsg

	streamErr   map[string]error
	consumerErr map[string]error
	oldestErr   map[string]error

	streamCalls atomic.Int64
}

func (r *queueMetricsReaderStub) StreamInfo(_ context.Context, stream string) (*gonats.StreamInfo, error) {
	r.streamCalls.Add(1)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.streamErr[stream]; err != nil {
		return nil, err
	}
	return r.streams[stream], nil
}

func (r *queueMetricsReaderStub) ConsumerInfo(_ context.Context, stream, durable string) (*gonats.ConsumerInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.consumerErr[queueMetricsTargetKey(stream, durable)]; err != nil {
		return nil, err
	}
	return r.consumers[queueMetricsTargetKey(stream, durable)], nil
}

func (r *queueMetricsReaderStub) GetMsg(_ context.Context, stream string, _ uint64, subject string) (*gonats.RawStreamMsg, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := queueMetricsMessageKey(stream, subject)
	if err := r.oldestErr[key]; err != nil {
		return nil, err
	}
	return r.oldest[key], nil
}

func newQueueMetricsTestReader() *queueMetricsReaderStub {
	return &queueMetricsReaderStub{
		streams:     make(map[string]*gonats.StreamInfo),
		consumers:   make(map[string]*gonats.ConsumerInfo),
		oldest:      make(map[string]*gonats.RawStreamMsg),
		streamErr:   make(map[string]error),
		consumerErr: make(map[string]error),
		oldestErr:   make(map[string]error),
	}
}

func populateQueueMetricsStreams(reader *queueMetricsReaderStub) {
	for _, stream := range DefaultStreams() {
		reader.streams[stream.Name] = &gonats.StreamInfo{
			Config: gonats.StreamConfig{Name: stream.Name},
		}
	}
}

func TestQueueMetricsCollectsKnownStreamAndConsumer(t *testing.T) {
	now := time.Now()
	reader := newQueueMetricsTestReader()
	populateQueueMetricsStreams(reader)
	reader.streams["PM"] = &gonats.StreamInfo{
		Config: gonats.StreamConfig{Name: "PM", MaxBytes: 1000, AllowDirect: true},
		State:  gonats.StreamState{Msgs: 3, Bytes: 64, FirstSeq: 1, LastSeq: 3},
	}
	reader.consumers[queueMetricsTargetKey("PM", "pm-workers")] = &gonats.ConsumerInfo{
		NumPending:     2,
		NumAckPending:  1,
		NumRedelivered: 4,
		Delivered:      gonats.SequenceInfo{Consumer: 20, Stream: 3},
		AckFloor:       gonats.SequenceInfo{Consumer: 18, Stream: 1},
	}
	reader.oldest[queueMetricsMessageKey("PM", "pm.file.received")] = &gonats.RawStreamMsg{
		Subject: "pm.file.received",
		Time:    now.Add(-90 * time.Second),
	}

	reg := prometheus.NewRegistry()
	metrics := newQueueMetricsWithReader(reader, reg, zap.NewNop(), []QueueMetricTarget{
		{Stream: "PM", Durable: "pm-workers", Subject: "pm.file.received"},
	})

	err := metrics.Collect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, float64(3), testutil.ToFloat64(metrics.StreamMessages.WithLabelValues("PM")))
	assert.Equal(t, float64(64), testutil.ToFloat64(metrics.StreamBytes.WithLabelValues("PM")))
	assert.Equal(t, float64(1000), testutil.ToFloat64(metrics.StreamMaxBytes.WithLabelValues("PM")))
	assert.Equal(t, float64(2), testutil.ToFloat64(metrics.ConsumerPending.WithLabelValues("PM", "pm-workers")))
	assert.Equal(t, float64(1), testutil.ToFloat64(metrics.ConsumerAckPending.WithLabelValues("PM", "pm-workers")))
	assert.Equal(t, float64(4), testutil.ToFloat64(metrics.ConsumerRedeliveredTotal.WithLabelValues("PM", "pm-workers")))
	assert.InDelta(t, 90, testutil.ToFloat64(metrics.ConsumerOldestAgeSeconds.WithLabelValues("PM", "pm-workers")), 1)
	assert.Equal(t, float64(1), testutil.ToFloat64(metrics.ConsumerUp.WithLabelValues("PM", "pm-workers")))
	assert.Positive(t, testutil.ToFloat64(metrics.ConsumerSampleTimestampSeconds.WithLabelValues("PM", "pm-workers")))
}

func TestQueueMetricsFailurePreservesLastBusinessValues(t *testing.T) {
	reader := newQueueMetricsTestReader()
	populateQueueMetricsStreams(reader)
	reader.streams["PM"] = &gonats.StreamInfo{
		Config: gonats.StreamConfig{Name: "PM", MaxBytes: 1000, AllowDirect: true},
		State:  gonats.StreamState{Msgs: 7, Bytes: 128, FirstSeq: 1, LastSeq: 7},
	}
	reader.consumers[queueMetricsTargetKey("PM", "pm-workers")] = &gonats.ConsumerInfo{
		NumPending:     7,
		NumAckPending:  2,
		NumRedelivered: 3,
		Delivered:      gonats.SequenceInfo{Consumer: 10, Stream: 7},
		AckFloor:       gonats.SequenceInfo{Consumer: 8, Stream: 5},
	}
	reader.oldest[queueMetricsMessageKey("PM", "pm.file.received")] = &gonats.RawStreamMsg{
		Subject: "pm.file.received",
		Time:    time.Now().Add(-30 * time.Second),
	}

	reg := prometheus.NewRegistry()
	metrics := newQueueMetricsWithReader(reader, reg, zap.NewNop(), []QueueMetricTarget{
		{Stream: "PM", Durable: "pm-workers", Subject: "pm.file.received"},
	})
	require.NoError(t, metrics.Collect(context.Background()))
	previousTimestamp := testutil.ToFloat64(metrics.ConsumerSampleTimestampSeconds.WithLabelValues("PM", "pm-workers"))

	reader.mu.Lock()
	reader.consumerErr[queueMetricsTargetKey("PM", "pm-workers")] = errors.New("consumer unavailable")
	reader.mu.Unlock()
	require.Error(t, metrics.Collect(context.Background()))

	assert.Equal(t, float64(7), testutil.ToFloat64(metrics.ConsumerPending.WithLabelValues("PM", "pm-workers")))
	assert.Equal(t, float64(2), testutil.ToFloat64(metrics.ConsumerAckPending.WithLabelValues("PM", "pm-workers")))
	assert.Equal(t, float64(3), testutil.ToFloat64(metrics.ConsumerRedeliveredTotal.WithLabelValues("PM", "pm-workers")))
	assert.Equal(t, previousTimestamp, testutil.ToFloat64(metrics.ConsumerSampleTimestampSeconds.WithLabelValues("PM", "pm-workers")))
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.ConsumerUp.WithLabelValues("PM", "pm-workers")))
	assert.Equal(t, float64(1), testutil.ToFloat64(metrics.ObserverFailures.WithLabelValues("consumer_info")))
}

func TestQueueMetricsRejectsUnregisteredTargetLabels(t *testing.T) {
	targets := DefaultQueueMetricTargets()
	require.NotEmpty(t, targets)
	assert.Contains(t, targets, QueueMetricTarget{
		Stream: "PM", Durable: "pm-registration-wait", Subject: "pm.file.deferred",
	})
	for _, target := range targets {
		assert.NotContains(t, target.Stream, "device")
		assert.NotContains(t, target.Durable, "SN")
		assert.NotContains(t, target.Subject, "*")
	}
	assert.Equal(t, targets, DefaultQueueMetricTargets(), "target registry must return a stable copy")
}

func TestQueueMetricsStartAndStopAreIdempotent(t *testing.T) {
	reader := newQueueMetricsTestReader()
	reg := prometheus.NewRegistry()
	metrics := newQueueMetricsWithReader(reader, reg, zap.NewNop(), nil,
		WithQueueMetricsInterval(2*time.Millisecond),
		WithQueueMetricsTimeout(20*time.Millisecond),
	)

	metrics.Start()
	metrics.Start()
	time.Sleep(10 * time.Millisecond)
	metrics.Stop()
	metrics.Stop()
	callsAfterStop := reader.streamCalls.Load()
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, callsAfterStop, reader.streamCalls.Load())
}
