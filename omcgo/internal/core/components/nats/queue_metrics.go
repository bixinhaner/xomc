package nats

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	DefaultQueueMetricsInterval = 30 * time.Second
	defaultQueueMetricsTimeout  = 5 * time.Second
)

// QueueMetricTarget is a compile-time registered JetStream consumer that is
// safe to expose as a Prometheus label set. Do not construct targets from
// subjects, durable names, device identifiers, or request payloads.
type QueueMetricTarget struct {
	Stream  string
	Durable string
	Subject string
}

var defaultQueueMetricTargets = []QueueMetricTarget{
	{Stream: "PM", Durable: "pm-workers", Subject: "pm.file.received"},
	{Stream: "PM", Durable: "pm-registration-wait", Subject: "pm.file.deferred"},
	{Stream: "MR", Durable: "mr-workers", Subject: "mr.file.received"},
	{Stream: "TRACE_MSG", Durable: "trace-capture", Subject: "trace.message.captured"},
	{Stream: "TRACE_EXPORT", Durable: "trace-export", Subject: "trace.export.requested"},
	{Stream: "BACKUP", Durable: "backup-executors", Subject: "backup.task.created"},
	{Stream: "REPORT", Durable: "report-generators", Subject: "report.generate.requested"},
}

// DefaultQueueMetricTargets returns a copy so callers cannot mutate the
// bounded label registry shared by all observers.
func DefaultQueueMetricTargets() []QueueMetricTarget {
	return append([]QueueMetricTarget(nil), defaultQueueMetricTargets...)
}

type queueMetricsReader interface {
	StreamInfo(ctx context.Context, stream string) (*nats.StreamInfo, error)
	ConsumerInfo(ctx context.Context, stream, durable string) (*nats.ConsumerInfo, error)
	GetMsg(ctx context.Context, stream string, sequence uint64, subject string) (*nats.RawStreamMsg, error)
}

type jetStreamQueueMetricsReader struct {
	js nats.JetStreamContext
}

func (r jetStreamQueueMetricsReader) StreamInfo(ctx context.Context, stream string) (*nats.StreamInfo, error) {
	return r.js.StreamInfo(stream, nats.Context(ctx))
}

func (r jetStreamQueueMetricsReader) ConsumerInfo(ctx context.Context, stream, durable string) (*nats.ConsumerInfo, error) {
	return r.js.ConsumerInfo(stream, durable, nats.Context(ctx))
}

func (r jetStreamQueueMetricsReader) GetMsg(ctx context.Context, stream string, sequence uint64, subject string) (*nats.RawStreamMsg, error) {
	return r.js.GetMsg(stream, sequence, nats.DirectGetNext(subject), nats.Context(ctx))
}

// QueueMetrics observes only registered JetStream streams and consumers. It
// never creates a consumer or consumes a business message.
type QueueMetrics struct {
	StreamMessages *prometheus.GaugeVec
	StreamBytes    *prometheus.GaugeVec
	StreamMaxBytes *prometheus.GaugeVec

	ConsumerPending                *prometheus.GaugeVec
	ConsumerAckPending             *prometheus.GaugeVec
	ConsumerRedeliveredTotal       *prometheus.CounterVec
	ConsumerOldestAgeSeconds       *prometheus.GaugeVec
	ConsumerDeliveredRate          *prometheus.GaugeVec
	ConsumerAckRate                *prometheus.GaugeVec
	ConsumerSampleTimestampSeconds *prometheus.GaugeVec
	ConsumerFreshnessSeconds       *prometheus.GaugeVec
	ConsumerUp                     *prometheus.GaugeVec

	StreamSampleTimestampSeconds *prometheus.GaugeVec
	StreamFreshnessSeconds       *prometheus.GaugeVec
	StreamUp                     *prometheus.GaugeVec
	ObserverFailures             *prometheus.CounterVec

	reader        queueMetricsReader
	targets       []QueueMetricTarget
	interval      time.Duration
	timeout       time.Duration
	logger        *zap.Logger
	collectMu     sync.Mutex
	stateMu       sync.Mutex
	consumerState map[string]queueConsumerState
	streamState   map[string]time.Time
	lifecycleMu   sync.Mutex
	cancel        context.CancelFunc
	started       bool
	stopped       bool
	done          chan struct{}
}

type queueConsumerState struct {
	sampledAt   time.Time
	delivered   uint64
	ack         uint64
	redelivered uint64
}

type QueueMetricsOption func(*QueueMetrics)

func WithQueueMetricsInterval(interval time.Duration) QueueMetricsOption {
	return func(m *QueueMetrics) {
		if interval > 0 {
			m.interval = interval
		}
	}
}

func WithQueueMetricsTimeout(timeout time.Duration) QueueMetricsOption {
	return func(m *QueueMetrics) {
		if timeout > 0 {
			m.timeout = timeout
		}
	}
}

// NewQueueMetrics creates an observer backed by the existing JetStream
// context. The caller owns the Prometheus registry; no NATS connection is
// created here.
func NewQueueMetrics(js nats.JetStreamContext, reg prometheus.Registerer, logger *zap.Logger, opts ...QueueMetricsOption) *QueueMetrics {
	return newQueueMetricsWithReader(jetStreamQueueMetricsReader{js: js}, reg, logger, defaultQueueMetricTargets, opts...)
}

func newQueueMetricsWithReader(reader queueMetricsReader, reg prometheus.Registerer, logger *zap.Logger, targets []QueueMetricTarget, opts ...QueueMetricsOption) *QueueMetrics {
	if reader == nil {
		panic("nats.NewQueueMetrics: reader is nil")
	}
	if reg == nil {
		panic("nats.NewQueueMetrics: registerer is nil")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	m := &QueueMetrics{
		StreamMessages: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_stream_messages",
			Help: "Current messages retained in a registered NATS JetStream stream.",
		}, []string{"stream"}),
		StreamBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_stream_bytes",
			Help: "Current bytes retained in a registered NATS JetStream stream.",
		}, []string{"stream"}),
		StreamMaxBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_stream_max_bytes",
			Help: "Configured maximum bytes for a registered NATS JetStream stream.",
		}, []string{"stream"}),
		ConsumerPending: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_consumer_pending",
			Help: "Messages pending delivery for a registered NATS JetStream consumer.",
		}, []string{"stream", "durable"}),
		ConsumerAckPending: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_consumer_ack_pending",
			Help: "Messages delivered but not yet acknowledged by a registered NATS consumer.",
		}, []string{"stream", "durable"}),
		ConsumerRedeliveredTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_nats_consumer_redelivered_total",
			Help: "Current NATS consumer redelivery count reported by JetStream.",
		}, []string{"stream", "durable"}),
		ConsumerOldestAgeSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_consumer_oldest_age_seconds",
			Help: "Age of the oldest pending message for a registered NATS consumer.",
		}, []string{"stream", "durable"}),
		ConsumerDeliveredRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_consumer_delivered_rate",
			Help: "Rate of NATS consumer deliveries per second between successful samples.",
		}, []string{"stream", "durable"}),
		ConsumerAckRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_consumer_ack_rate",
			Help: "Rate of NATS consumer acknowledgements per second between successful samples.",
		}, []string{"stream", "durable"}),
		ConsumerSampleTimestampSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_consumer_sample_timestamp_seconds",
			Help: "Unix timestamp of the last successful NATS consumer sample.",
		}, []string{"stream", "durable"}),
		ConsumerFreshnessSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_consumer_freshness_seconds",
			Help: "Seconds since the last successful NATS consumer sample.",
		}, []string{"stream", "durable"}),
		ConsumerUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_consumer_up",
			Help: "Whether the last NATS consumer sample succeeded.",
		}, []string{"stream", "durable"}),
		StreamSampleTimestampSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_stream_sample_timestamp_seconds",
			Help: "Unix timestamp of the last successful NATS stream sample.",
		}, []string{"stream"}),
		StreamFreshnessSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_stream_freshness_seconds",
			Help: "Seconds since the last successful NATS stream sample.",
		}, []string{"stream"}),
		StreamUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_nats_stream_up",
			Help: "Whether the last NATS stream sample succeeded.",
		}, []string{"stream"}),
		ObserverFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_nats_observer_failures_total",
			Help: "NATS JetStream observer failures by fixed operation.",
		}, []string{"operation"}),
		reader:        reader,
		targets:       append([]QueueMetricTarget(nil), targets...),
		interval:      DefaultQueueMetricsInterval,
		timeout:       defaultQueueMetricsTimeout,
		logger:        logger,
		consumerState: make(map[string]queueConsumerState),
		streamState:   make(map[string]time.Time),
		done:          make(chan struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}

	reg.MustRegister(
		m.StreamMessages, m.StreamBytes, m.StreamMaxBytes,
		m.ConsumerPending, m.ConsumerAckPending, m.ConsumerRedeliveredTotal,
		m.ConsumerOldestAgeSeconds, m.ConsumerDeliveredRate, m.ConsumerAckRate,
		m.ConsumerSampleTimestampSeconds, m.ConsumerFreshnessSeconds, m.ConsumerUp,
		m.StreamSampleTimestampSeconds, m.StreamFreshnessSeconds, m.StreamUp,
		m.ObserverFailures,
	)
	m.initialize()
	return m
}

func (m *QueueMetrics) initialize() {
	for _, stream := range DefaultStreams() {
		m.StreamMessages.WithLabelValues(stream.Name).Set(0)
		m.StreamBytes.WithLabelValues(stream.Name).Set(0)
		m.StreamMaxBytes.WithLabelValues(stream.Name).Set(0)
		m.StreamSampleTimestampSeconds.WithLabelValues(stream.Name).Set(0)
		m.StreamFreshnessSeconds.WithLabelValues(stream.Name).Set(0)
		m.StreamUp.WithLabelValues(stream.Name).Set(0)
	}
	for _, target := range m.targets {
		labels := []string{target.Stream, target.Durable}
		m.ConsumerPending.WithLabelValues(labels...).Set(0)
		m.ConsumerAckPending.WithLabelValues(labels...).Set(0)
		m.ConsumerRedeliveredTotal.WithLabelValues(labels...).Add(0)
		m.ConsumerOldestAgeSeconds.WithLabelValues(labels...).Set(0)
		m.ConsumerDeliveredRate.WithLabelValues(labels...).Set(0)
		m.ConsumerAckRate.WithLabelValues(labels...).Set(0)
		m.ConsumerSampleTimestampSeconds.WithLabelValues(labels...).Set(0)
		m.ConsumerFreshnessSeconds.WithLabelValues(labels...).Set(0)
		m.ConsumerUp.WithLabelValues(labels...).Set(0)
		m.consumerState[queueMetricsTargetKey(target.Stream, target.Durable)] = queueConsumerState{}
	}
	for _, operation := range []string{"stream_info", "consumer_info", "oldest_message"} {
		m.ObserverFailures.WithLabelValues(operation).Add(0)
	}
}

// Start is idempotent and performs one immediate sample before entering the
// fixed interval loop.
func (m *QueueMetrics) Start() {
	m.lifecycleMu.Lock()
	if m.started || m.stopped {
		m.lifecycleMu.Unlock()
		return
	}
	m.started = true
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.lifecycleMu.Unlock()

	go func() {
		defer close(m.done)
		m.sampleAndLog(ctx)
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.sampleAndLog(ctx)
			}
		}
	}()
}

// Stop is idempotent and waits for an in-flight sample to finish or observe
// the canceled context. The timeout bounds shutdown when NATS is unavailable.
func (m *QueueMetrics) Stop() {
	m.lifecycleMu.Lock()
	if m.stopped {
		m.lifecycleMu.Unlock()
		return
	}
	m.stopped = true
	if !m.started {
		m.lifecycleMu.Unlock()
		return
	}
	cancel := m.cancel
	m.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	<-m.done
}

func (m *QueueMetrics) sampleAndLog(ctx context.Context) {
	if err := m.Collect(ctx); err != nil && ctx.Err() == nil {
		m.logger.Warn("NATS JetStream queue metrics sample failed", zap.Error(err))
	}
}

// Collect performs one bounded sample. Successful business gauges are updated
// atomically per target; failures leave their previous values untouched.
func (m *QueueMetrics) Collect(ctx context.Context) error {
	m.collectMu.Lock()
	defer m.collectMu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}
	sampleCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	now := time.Now()
	var failures []error
	streamInfos := make(map[string]*nats.StreamInfo)
	for _, stream := range DefaultStreams() {
		info, err := m.reader.StreamInfo(sampleCtx, stream.Name)
		if err != nil || info == nil {
			if err == nil {
				err = errors.New("empty stream info")
			}
			m.ObserverFailures.WithLabelValues("stream_info").Inc()
			m.markStreamFailure(stream.Name, now)
			failures = append(failures, fmt.Errorf("stream %s: %w", stream.Name, err))
			continue
		}
		streamInfos[stream.Name] = info
		m.StreamMessages.WithLabelValues(stream.Name).Set(float64(info.State.Msgs))
		m.StreamBytes.WithLabelValues(stream.Name).Set(float64(info.State.Bytes))
		m.StreamMaxBytes.WithLabelValues(stream.Name).Set(float64(info.Config.MaxBytes))
		m.markStreamSuccess(stream.Name, now)
	}

	for _, target := range m.targets {
		info := streamInfos[target.Stream]
		if info == nil {
			m.markConsumerFailure(target, now)
			continue
		}
		consumer, err := m.reader.ConsumerInfo(sampleCtx, target.Stream, target.Durable)
		if err != nil || consumer == nil {
			if err == nil {
				err = errors.New("empty consumer info")
			}
			m.ObserverFailures.WithLabelValues("consumer_info").Inc()
			m.markConsumerFailure(target, now)
			failures = append(failures, fmt.Errorf("consumer %s/%s: %w", target.Stream, target.Durable, err))
			continue
		}

		oldestAge, err := m.oldestAge(sampleCtx, target, consumer, info, now)
		if err != nil {
			m.ObserverFailures.WithLabelValues("oldest_message").Inc()
			m.markConsumerFailure(target, now)
			failures = append(failures, err)
			continue
		}
		m.observeConsumerSuccess(target, consumer, oldestAge, now)
	}

	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	return nil
}

func (m *QueueMetrics) oldestAge(ctx context.Context, target QueueMetricTarget, consumer *nats.ConsumerInfo, stream *nats.StreamInfo, sampledAt time.Time) (time.Duration, error) {
	if consumer.NumPending == 0 && consumer.NumAckPending <= 0 {
		return 0, nil
	}
	sequence := oldestPendingSequence(consumer, stream.State)
	if sequence == 0 {
		return 0, nil
	}
	message, err := m.reader.GetMsg(ctx, target.Stream, sequence, target.Subject)
	if errors.Is(err, nats.ErrMsgNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("oldest message %s/%s: %w", target.Stream, target.Durable, err)
	}
	if message == nil || message.Time.IsZero() {
		return 0, nil
	}
	age := sampledAt.Sub(message.Time)
	if age < 0 {
		return 0, nil
	}
	return age, nil
}

func oldestPendingSequence(consumer *nats.ConsumerInfo, state nats.StreamState) uint64 {
	if consumer == nil || state.FirstSeq == 0 || state.LastSeq == 0 {
		return 0
	}
	sequence := state.FirstSeq
	if consumer.NumAckPending > 0 {
		if consumer.AckFloor.Stream == ^uint64(0) {
			return 0
		}
		if candidate := consumer.AckFloor.Stream + 1; candidate > sequence {
			sequence = candidate
		}
	} else if consumer.Delivered.Stream > 0 {
		if consumer.Delivered.Stream == ^uint64(0) {
			return 0
		}
		if candidate := consumer.Delivered.Stream + 1; candidate > sequence {
			sequence = candidate
		}
	}
	if sequence > state.LastSeq {
		return 0
	}
	return sequence
}

func (m *QueueMetrics) observeConsumerSuccess(target QueueMetricTarget, consumer *nats.ConsumerInfo, oldestAge time.Duration, now time.Time) {
	key := queueMetricsTargetKey(target.Stream, target.Durable)
	labels := []string{target.Stream, target.Durable}
	m.stateMu.Lock()
	previous := m.consumerState[key]
	current := queueConsumerState{
		sampledAt:   now,
		delivered:   consumer.Delivered.Consumer,
		ack:         consumer.AckFloor.Consumer,
		redelivered: uint64(max(0, consumer.NumRedelivered)),
	}
	m.consumerState[key] = current
	m.stateMu.Unlock()

	deliveredRate, ackRate := 0.0, 0.0
	if !previous.sampledAt.IsZero() {
		seconds := now.Sub(previous.sampledAt).Seconds()
		if seconds > 0 {
			if current.delivered >= previous.delivered {
				deliveredRate = float64(current.delivered-previous.delivered) / seconds
			}
			if current.ack >= previous.ack {
				ackRate = float64(current.ack-previous.ack) / seconds
			}
		}
	}
	m.ConsumerPending.WithLabelValues(labels...).Set(float64(consumer.NumPending))
	m.ConsumerAckPending.WithLabelValues(labels...).Set(float64(consumer.NumAckPending))
	redeliveredDelta := current.redelivered
	if !previous.sampledAt.IsZero() && current.redelivered >= previous.redelivered {
		redeliveredDelta = current.redelivered - previous.redelivered
	}
	if redeliveredDelta > 0 {
		m.ConsumerRedeliveredTotal.WithLabelValues(labels...).Add(float64(redeliveredDelta))
	}
	m.ConsumerOldestAgeSeconds.WithLabelValues(labels...).Set(oldestAge.Seconds())
	m.ConsumerDeliveredRate.WithLabelValues(labels...).Set(deliveredRate)
	m.ConsumerAckRate.WithLabelValues(labels...).Set(ackRate)
	m.ConsumerSampleTimestampSeconds.WithLabelValues(labels...).Set(float64(now.UnixNano()) / float64(time.Second))
	m.ConsumerFreshnessSeconds.WithLabelValues(labels...).Set(0)
	m.ConsumerUp.WithLabelValues(labels...).Set(1)
}

func (m *QueueMetrics) markStreamSuccess(stream string, now time.Time) {
	m.stateMu.Lock()
	m.streamState[stream] = now
	m.stateMu.Unlock()
	m.StreamSampleTimestampSeconds.WithLabelValues(stream).Set(float64(now.UnixNano()) / float64(time.Second))
	m.StreamFreshnessSeconds.WithLabelValues(stream).Set(0)
	m.StreamUp.WithLabelValues(stream).Set(1)
}

func (m *QueueMetrics) markStreamFailure(stream string, now time.Time) {
	m.stateMu.Lock()
	last := m.streamState[stream]
	m.stateMu.Unlock()
	m.StreamUp.WithLabelValues(stream).Set(0)
	if last.IsZero() {
		m.StreamFreshnessSeconds.WithLabelValues(stream).Set(0)
		return
	}
	m.StreamFreshnessSeconds.WithLabelValues(stream).Set(now.Sub(last).Seconds())
}

func (m *QueueMetrics) markConsumerFailure(target QueueMetricTarget, now time.Time) {
	labels := []string{target.Stream, target.Durable}
	key := queueMetricsTargetKey(target.Stream, target.Durable)
	m.stateMu.Lock()
	previous := m.consumerState[key].sampledAt
	m.stateMu.Unlock()
	m.ConsumerUp.WithLabelValues(labels...).Set(0)
	if previous.IsZero() {
		m.ConsumerFreshnessSeconds.WithLabelValues(labels...).Set(0)
		return
	}
	m.ConsumerFreshnessSeconds.WithLabelValues(labels...).Set(now.Sub(previous).Seconds())
}

func queueMetricsTargetKey(stream, durable string) string {
	return stream + "\x00" + durable
}

func queueMetricsMessageKey(stream, subject string) string {
	return stream + "\x00" + subject
}
