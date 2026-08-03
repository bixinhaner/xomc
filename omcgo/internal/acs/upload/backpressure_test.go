package upload

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type queueStatsSourceStub struct {
	stats            event.QueueStats
	ok               bool
	lastAttempt      time.Time
	attemptSucceeded bool
}

func (s *queueStatsSourceStub) LatestStats() (event.QueueStats, bool) {
	return s.stats, s.ok
}

func (s *queueStatsSourceStub) LastSampleAttempt() (time.Time, bool) {
	return s.lastAttempt, s.attemptSucceeded
}

func TestParseDiskUsagePct(t *testing.T) {
	tests := []struct {
		name string
		text string
		want float64
		err  bool
	}{
		{
			name: "usable capacity total+free",
			text: `# HELP minio_cluster_capacity_usable_total_bytes x
minio_cluster_capacity_usable_total_bytes{server="m1"} 1000
minio_cluster_capacity_usable_free_bytes{server="m1"} 200`,
			want: 80, // (1000-200)/1000
		},
		{
			name: "node disk free summed across drives",
			text: `minio_node_disk_total_bytes{drive="/d1"} 500
minio_node_disk_total_bytes{drive="/d2"} 500
minio_node_disk_free_bytes{drive="/d1"} 100
minio_node_disk_free_bytes{drive="/d2"} 150`,
			want: 75, // total 1000, free 250 → used 750
		},
		{
			name: "node disk used when no free",
			text: `minio_node_disk_total_bytes 1000
minio_node_disk_used_bytes 900`,
			want: 90,
		},
		{
			name: "no usable metrics",
			text: `some_other_metric 1`,
			err:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDiskUsagePct(tt.text)
			if tt.err {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestProjectedUsagePctIncludesAcceptedPendingBytes(t *testing.T) {
	assert.Equal(t, 70.0, projectedUsagePct(1000, 600, 100))
	assert.Equal(t, 100.0, projectedUsagePct(1000, 950, 100))
}

func TestWatchdog_RecordAcceptedPublishesLastSuccessfulPMUploadTimestamp(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewBackpressureMetrics(reg)
	watchdog := &Watchdog{metrics: metrics}
	acceptedAt := time.Date(2026, 8, 3, 15, 1, 0, 0, time.UTC)

	watchdog.RecordAccepted(acceptedAt)

	require.Equal(t, float64(acceptedAt.Unix()), testutil.ToFloat64(metrics.lastAccepted))
}

func TestDecideBackpressure_Hysteresis(t *testing.T) {
	cfg := BackpressureConfig{
		Enabled: true, DiskHighPct: 85, DiskLowPct: 75,
		IOSomeHighPct: 40, IOSomeLowPct: 20,
	}

	// 未背压：低于高水位不触发。
	assert.False(t, decideBackpressure(false, 80, 30, cfg))
	// 未背压：磁盘越高水位 → 进入。
	assert.True(t, decideBackpressure(false, 90, 30, cfg))
	// 未背压：IO PSI 越高水位 → 进入，即使磁盘空间充足。
	assert.True(t, decideBackpressure(false, 60, 45, cfg))
	// 已背压：介于高低水位之间 → 保持（迟滞，不抖动）。
	assert.True(t, decideBackpressure(true, 80, 30, cfg))
	// 已背压：磁盘回落到低水位以下 → 解除。
	assert.False(t, decideBackpressure(true, 70, 15, cfg))

	// disabled 恒不背压。
	off := cfg
	off.Enabled = false
	assert.False(t, decideBackpressure(true, 99, 99, off))
}

func TestDecideBackpressure_UnknownSignalsFailOpen(t *testing.T) {
	cfg := BackpressureConfig{
		Enabled: true, DiskHighPct: 85, DiskLowPct: 75,
		IOSomeHighPct: 40, IOSomeLowPct: 20,
	}
	// 信号不可用(-1) → 不进入背压（fail-open）。
	assert.False(t, decideBackpressure(false, -1, -1, cfg))
	// 已背压 + 两个信号均不可用 → 解除（fail-open，不长期误堵）。
	assert.False(t, decideBackpressure(true, -1, -1, cfg))
	// 任一可用信号仍处于迟滞区间时保持背压。
	assert.True(t, decideBackpressure(true, -1, 30, cfg))
}

func TestDecideBackpressureQueueSignal_SustainedHighWatermarksActivate(t *testing.T) {
	cfg := BackpressureConfig{
		Enabled: true, DiskHighPct: 85, DiskLowPct: 75,
		IOSomeHighPct: 40, IOSomeLowPct: 20,
		QueuePendingHigh: 2000, QueuePendingLow: 500,
		QueueOldestHigh: 10 * time.Minute, QueueOldestLow: 2 * time.Minute,
	}

	youngBurst := decideBackpressureWithQueue(false, 50, 10, QueueSignal{
		Configured: true,
		Available:  true,
		Stats: event.QueueStats{
			Pending:          2000,
			OldestPendingAge: time.Minute,
		},
	}, cfg)
	assert.False(t, youngBurst.Active,
		"a synchronized PM burst younger than the low age watermark must use the durable queue instead of returning 503")
	youngState, youngDecision := decideBackpressureState(0, 50, 10, QueueSignal{
		Configured: true,
		Available:  true,
		Stats: event.QueueStats{
			Pending:          2000,
			OldestPendingAge: time.Minute,
		},
	}, cfg)
	assert.Zero(t, youngState)
	assert.False(t, youngDecision.Active)

	pending := decideBackpressureWithQueue(false, 50, 10, QueueSignal{
		Configured: true,
		Available:  true,
		Stats: event.QueueStats{
			Pending:          2000,
			OldestPendingAge: 2 * time.Minute,
		},
	}, cfg)
	assert.True(t, pending.Active)
	assert.Equal(t, pressureReasonQueuePending, pending.Reason)

	oldest := decideBackpressureWithQueue(false, 50, 10, QueueSignal{
		Configured: true,
		Available:  true,
		Stats:      event.QueueStats{Pending: 10, OldestPendingAge: 10 * time.Minute},
	}, cfg)
	assert.True(t, oldest.Active)
	assert.Equal(t, pressureReasonQueueOldest, oldest.Reason)
}

func TestBackpressureStateQueueRecoveryIgnoresUnlatchedDiskNeutralBand(t *testing.T) {
	cfg := BackpressureConfig{
		Enabled: true, DiskHighPct: 70, DiskLowPct: 60,
		IOSomeHighPct: 70, IOSomeLowPct: 20,
		QueuePendingHigh: 2000, QueuePendingLow: 500,
		QueueOldestHigh: 10 * time.Minute, QueueOldestLow: 2 * time.Minute,
	}
	engaged, decision := decideBackpressureState(0, 68, 0, QueueSignal{
		Configured:     true,
		Available:      true,
		RatesAvailable: true,
		Stats: event.QueueStats{
			Pending:          2000,
			OldestPendingAge: 2 * time.Minute,
		},
	}, cfg)
	require.True(t, engaged.has(pressureQueue))
	require.False(t, engaged.has(pressureDisk),
		"disk in the neutral band must not latch unless it crossed the high watermark")
	require.True(t, decision.Active)
	require.Equal(t, pressureReasonQueuePending, decision.Reason)

	recovered, decision := decideBackpressureState(engaged, 68, 0, QueueSignal{
		Configured:     true,
		Available:      true,
		RatesAvailable: true,
		Stats:          event.QueueStats{Pending: 0},
		Rates:          QueueRates{PendingPerSecond: 0},
	}, cfg)
	require.Zero(t, recovered)
	require.False(t, decision.Active)
	require.Equal(t, pressureReasonRecovered, decision.Reason)
}

func TestBackpressureStateKeepsOnlySignalsThatActuallyLatched(t *testing.T) {
	cfg := BackpressureConfig{
		Enabled: true, DiskHighPct: 70, DiskLowPct: 60,
		IOSomeHighPct: 70, IOSomeLowPct: 20,
		QueuePendingHigh: 2000, QueuePendingLow: 500,
		QueueOldestHigh: 10 * time.Minute, QueueOldestLow: 2 * time.Minute,
	}
	queueLow := QueueSignal{
		Configured:     true,
		Available:      true,
		RatesAvailable: true,
		Stats:          event.QueueStats{Pending: 0},
	}

	engaged, _ := decideBackpressureState(0, 72, 0, queueLow, cfg)
	require.True(t, engaged.has(pressureDisk))
	require.False(t, engaged.has(pressureQueue))

	stillEngaged, decision := decideBackpressureState(engaged, 68, 0, queueLow, cfg)
	require.Equal(t, pressureDisk, stillEngaged)
	require.True(t, decision.Active)
	require.Equal(t, pressureReasonDisk, decision.Reason)

	recovered, decision := decideBackpressureState(stillEngaged, 59, 0, queueLow, cfg)
	require.Zero(t, recovered)
	require.False(t, decision.Active)
	require.Equal(t, pressureReasonRecovered, decision.Reason)
}

func TestBackpressureStateUnavailableDiskAndIORetainOnlyConfirmedLatches(t *testing.T) {
	cfg := BackpressureConfig{
		Enabled: true, DiskHighPct: 70, DiskLowPct: 60,
		IOSomeHighPct: 70, IOSomeLowPct: 20,
	}

	unlatched, decision := decideBackpressureState(0, -1, -1, QueueSignal{}, cfg)
	require.Zero(t, unlatched, "unavailable probes must remain fail-open before pressure is confirmed")
	require.False(t, decision.Active)

	previous := pressureDisk | pressureIO
	retained, decision := decideBackpressureState(previous, -1, -1, QueueSignal{}, cfg)
	require.Equal(t, previous, retained,
		"unavailable probes must not release previously confirmed disk or I/O pressure")
	require.True(t, decision.Active)
}

func TestDecideBackpressureQueueSignal_ReleaseRequiresEveryLowSignalAndNonPositiveSlope(t *testing.T) {
	cfg := BackpressureConfig{
		Enabled: true, DiskHighPct: 85, DiskLowPct: 75,
		IOSomeHighPct: 40, IOSomeLowPct: 20,
		QueuePendingHigh: 2000, QueuePendingLow: 500,
		QueueOldestHigh: 10 * time.Minute, QueueOldestLow: 2 * time.Minute,
	}
	base := QueueSignal{
		Configured:     true,
		Available:      true,
		RatesAvailable: true,
		Stats:          event.QueueStats{Pending: 400, OldestPendingAge: time.Minute},
		Rates:          QueueRates{PendingPerSecond: 0},
	}

	assert.False(t, decideBackpressureWithQueue(true, 70, 15, base, cfg).Active)

	pendingMiddle := base
	pendingMiddle.Stats.Pending = 600
	assert.True(t, decideBackpressureWithQueue(true, 70, 15, pendingMiddle, cfg).Active)

	oldestMiddle := base
	oldestMiddle.Stats.OldestPendingAge = 3 * time.Minute
	assert.True(t, decideBackpressureWithQueue(true, 70, 15, oldestMiddle, cfg).Active)

	growing := base
	growing.Rates.PendingPerSecond = 0.1
	assert.True(t, decideBackpressureWithQueue(true, 70, 15, growing, cfg).Active)

	unknownSlope := base
	unknownSlope.RatesAvailable = false
	assert.True(t, decideBackpressureWithQueue(true, 70, 15, unknownSlope, cfg).Active,
		"release requires an observed non-positive slope")
}

func TestDecideBackpressureQueueSignal_FailurePreservesState(t *testing.T) {
	cfg := BackpressureConfig{Enabled: true}
	failed := QueueSignal{Configured: true, Available: false}

	assert.False(t, decideBackpressureWithQueue(false, -1, -1, failed, cfg).Active,
		"startup failure must fail open before pressure has been observed")
	assert.True(t, decideBackpressureWithQueue(true, -1, -1, failed, cfg).Active,
		"a queue read failure must not release established pressure")
}

func TestQueueSignalRates_DerivesRatesAndRejectsResetOrStaleSample(t *testing.T) {
	at := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	previous := event.QueueStats{
		Pending: 100, LastSequence: 1000, AckSequence: 900,
		DeliverySequence: 800, AckConsumerSequence: 700, SampledAt: at,
	}
	current := event.QueueStats{
		Pending: 130, LastSequence: 1060, AckSequence: 930,
		DeliverySequence: 860, AckConsumerSequence: 730, SampledAt: at.Add(30 * time.Second),
	}

	rates, ok := deriveQueueRates(previous, current, time.Minute)
	assert.True(t, ok)
	assert.InDelta(t, 1, rates.PendingPerSecond, 0.001)
	assert.InDelta(t, 1, rates.AckAdvancePerSecond, 0.001)
	assert.InDelta(t, 2, rates.DeliveryPerSecond, 0.001)

	jittered := current
	jittered.SampledAt = at.Add(event.QueueHealthSampleInterval + time.Millisecond)
	rates, ok = deriveQueueRates(previous, jittered, bpMinQueueSlopeWindow)
	assert.True(t, ok, "two sampler intervals must absorb query-duration jitter beyond 30s")

	reset := current
	reset.DeliverySequence = 1
	reset.AckConsumerSequence = 1
	rates, ok = deriveQueueRates(previous, reset, time.Minute)
	assert.False(t, ok)
	assert.Equal(t, QueueRates{}, rates)

	unrelatedStreamTraffic := current
	unrelatedStreamTraffic.LastSequence = 100000
	rates, ok = deriveQueueRates(previous, unrelatedStreamTraffic, time.Minute)
	assert.True(t, ok)
	assert.InDelta(t, 2, rates.DeliveryPerSecond, 0.001,
		"shared-stream sequence gaps from unrelated PM subjects must not affect delivery rate")

	stale := current
	stale.SampledAt = at.Add(2 * time.Minute)
	rates, ok = deriveQueueRates(previous, stale, time.Minute)
	assert.False(t, ok)
	assert.Equal(t, QueueRates{}, rates)
}

func TestQueueSignalWatchdog_UsesSamplerCacheAndFailureCannotReleasePressure(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewBackpressureMetrics(reg)
	w := NewWatchdog(nil, nil, metrics, nil)
	w.ioPressure = nil
	w.loadFn = nil
	source := &queueStatsSourceStub{
		ok: true,
		stats: event.QueueStats{
			Pending:          bpDefaultQueuePendingHigh,
			OldestPendingAge: bpDefaultQueueOldestLow,
			SampledAt:        time.Now(),
		},
	}
	w.SetQueueStatsSource(source)

	w.sample(context.Background())
	assert.True(t, w.active.Load())

	source.ok = false
	source.lastAttempt = time.Now().Add(time.Second)
	source.attemptSucceeded = false
	w.sample(context.Background())
	assert.True(t, w.active.Load(), "failed cache read must preserve active pressure")
}

func TestQueueSignalWatchdog_RecentCachedSuccessCannotHideNewerSampleFailure(t *testing.T) {
	now := time.Now()
	source := &queueStatsSourceStub{
		ok: true,
		stats: event.QueueStats{
			Pending:   100,
			SampledAt: now,
		},
		lastAttempt:      now.Add(time.Second),
		attemptSucceeded: false,
	}
	w := NewWatchdog(nil, nil, NewBackpressureMetrics(nil), nil)
	w.ioPressure = nil
	w.loadFn = nil
	w.active.Store(true)
	w.SetQueueStatsSource(source)

	w.sample(context.Background())

	assert.True(t, w.active.Load(),
		"a newer failed sample attempt must preserve pressure despite a fresh cached success")
}

func TestQueueSignalWatchdog_DisableDoesNotForgetObservedPressure(t *testing.T) {
	enabled := true
	lookup := func(_ context.Context, category, key string) (string, bool) {
		if category == BackpressureCategory && key == bpKeyEnabled {
			if enabled {
				return "true", true
			}
			return "false", true
		}
		return "", false
	}
	base := time.Now().Add(-50 * time.Second)
	source := &queueStatsSourceStub{
		ok: true,
		stats: event.QueueStats{
			Pending:             bpDefaultQueuePendingHigh,
			OldestPendingAge:    bpDefaultQueueOldestLow,
			DeliverySequence:    100,
			AckConsumerSequence: 90,
			SampledAt:           base,
		},
		lastAttempt:      base.Add(time.Second),
		attemptSucceeded: true,
	}
	w := NewWatchdog(lookup, nil, NewBackpressureMetrics(nil), nil)
	w.ioPressure = nil
	w.loadFn = nil
	w.SetQueueStatsSource(source)

	w.sample(context.Background())
	assert.True(t, w.active.Load())

	enabled = false
	w.sample(context.Background())
	assert.False(t, w.active.Load(), "disabled gate must admit uploads")

	source.lastAttempt = base.Add(10 * time.Second)
	source.attemptSucceeded = false
	enabled = true
	w.sample(context.Background())
	assert.True(t, w.active.Load(),
		"re-enable after a failed sample must restore remembered pressure")

	source.stats = event.QueueStats{
		Pending:             400,
		DeliverySequence:    140,
		AckConsumerSequence: 130,
		SampledAt:           base.Add(40 * time.Second),
	}
	source.lastAttempt = base.Add(41 * time.Second)
	source.attemptSucceeded = true
	w.sample(context.Background())
	assert.False(t, w.active.Load(),
		"a fresh low sample with a non-positive slope proves recovery")

	startupFailure := NewWatchdog(nil, nil, NewBackpressureMetrics(nil), nil)
	startupFailure.ioPressure = nil
	startupFailure.loadFn = nil
	startupFailure.SetQueueStatsSource(&queueStatsSourceStub{
		lastAttempt: time.Now(), attemptSucceeded: false,
	})
	startupFailure.sample(context.Background())
	assert.False(t, startupFailure.active.Load(), "never-pressured startup failure remains fail-open")
}

func TestQueueSignalWatchdog_InvalidSamplesInvalidateRates(t *testing.T) {
	tests := []struct {
		name       string
		invalidate func(source *queueStatsSourceStub, now time.Time)
	}{
		{
			name: "stale sample",
			invalidate: func(source *queueStatsSourceStub, now time.Time) {
				source.stats.SampledAt = now.Add(-10 * time.Minute)
			},
		},
		{
			name: "newer failed attempt",
			invalidate: func(source *queueStatsSourceStub, _ time.Time) {
				source.lastAttempt = source.stats.SampledAt.Add(time.Second)
				source.attemptSucceeded = false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			source := &queueStatsSourceStub{
				ok: true,
				stats: event.QueueStats{
					Pending:             100,
					DeliverySequence:    100,
					AckConsumerSequence: 80,
					SampledAt:           now.Add(-40 * time.Second),
				},
				lastAttempt:      now.Add(-39 * time.Second),
				attemptSucceeded: true,
			}
			metrics := NewBackpressureMetrics(nil)
			w := NewWatchdog(nil, nil, metrics, nil)
			w.ioPressure = nil
			w.loadFn = nil
			w.SetQueueStatsSource(source)
			w.sample(context.Background())

			source.stats = event.QueueStats{
				Pending:             130,
				DeliverySequence:    160,
				AckConsumerSequence: 110,
				SampledAt:           now.Add(-10 * time.Second),
			}
			source.lastAttempt = now.Add(-9 * time.Second)
			w.sample(context.Background())
			assert.InDelta(t, 2, testutil.ToFloat64(metrics.queueDeliveryRate), 0.001)
			assert.InDelta(t, 1, testutil.ToFloat64(metrics.queueAckRate), 0.001)
			assert.InDelta(t, 1, testutil.ToFloat64(metrics.queueBacklogSlope), 0.001)

			w.sample(context.Background())
			assert.InDelta(t, 2, testutil.ToFloat64(metrics.queueDeliveryRate), 0.001,
				"re-reading the same fresh snapshot must retain its derived rates")
			assert.InDelta(t, 1, testutil.ToFloat64(metrics.queueAckRate), 0.001)
			assert.InDelta(t, 1, testutil.ToFloat64(metrics.queueBacklogSlope), 0.001)

			tt.invalidate(source, now)
			w.sample(context.Background())
			assert.True(t, math.IsNaN(testutil.ToFloat64(metrics.queueDeliveryRate)))
			assert.True(t, math.IsNaN(testutil.ToFloat64(metrics.queueAckRate)))
			assert.True(t, math.IsNaN(testutil.ToFloat64(metrics.queueBacklogSlope)))
		})
	}
}

func TestWatchdogSettersAreSafeDuringSampling(t *testing.T) {
	w := NewWatchdog(nil, nil, NewBackpressureMetrics(nil), nil)
	w.ioPressure = nil
	w.loadFn = nil
	source := &queueStatsSourceStub{
		ok:    true,
		stats: event.QueueStats{Pending: 1, SampledAt: time.Now()},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 100 {
			w.SetQueueStatsSource(source)
			w.SetQueueThresholdDefaults(2000, 500, 10*time.Minute, 2*time.Minute, 5*time.Minute)
		}
	}()
	go func() {
		defer wg.Done()
		for range 100 {
			w.sample(context.Background())
		}
	}()
	wg.Wait()
}

func TestQueueSaturationAlertRequiresFreshSamplerData(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	raw, err := os.ReadFile(filepath.Join(
		filepath.Dir(file), "..", "..", "..", "..",
		"deployments", "monitoring", "alerts", "omc-rules.yml",
	))
	require.NoError(t, err)
	alerts := string(raw)
	require.Contains(t, alerts, "alert: OMCPMQueueAckRateBelowDelivery")
	require.Contains(t, alerts,
		`omc_pm_queue_pending{subject="pm.file.received",durable="pm-workers"} >= 5000`)
	require.Contains(t, alerts, "and max without (subject, durable) (")
	require.Contains(t, alerts,
		`time() - omc_pm_queue_sample_timestamp_seconds{deployment_unit="acs",subject="pm.file.received",durable="pm-workers"}`)
	require.Contains(t, alerts, ") < 60")
}

func TestQueueSignalMetrics_ExportsStateTransitionsRatesAndFreshness(t *testing.T) {
	reg := prometheus.NewRegistry()
	NewBackpressureMetrics(reg)

	names := map[string]bool{}
	families, err := reg.Gather()
	assert.NoError(t, err)
	for _, family := range families {
		names[family.GetName()] = true
	}
	for _, name := range []string{
		"acs_pm_upload_backpressure_active",
		"acs_pm_upload_backpressure_transitions_total",
		"acs_pm_queue_delivery_rate",
		"acs_pm_queue_ack_rate",
		"acs_pm_queue_backlog_slope",
	} {
		assert.Truef(t, names[name], "metric %s must be registered", name)
	}
}

func TestLoadBackpressureConfig_DefaultsAndClamp(t *testing.T) {
	// nil lookup → 全默认。
	def := loadBackpressureConfig(context.Background(), nil)
	assert.True(t, def.Enabled)
	assert.Equal(t, 70.0, def.DiskHighPct)
	assert.Equal(t, 60.0, def.DiskLowPct)
	assert.Equal(t, 70.0, def.IOSomeHighPct)
	assert.Equal(t, 20.0, def.IOSomeLowPct)
	assert.Equal(t, int64(2000), def.MaxInflight)
	assert.Equal(t, 30*time.Second, def.Interval)
	assert.Equal(t, 5000, def.QueuePendingHigh)
	assert.Equal(t, 1000, def.QueuePendingLow)
	assert.Equal(t, 10*time.Minute, def.QueueOldestHigh)
	assert.Equal(t, 2*time.Minute, def.QueueOldestLow)
	assert.Positive(t, def.QueueSlopeWindow)

	// low > high → 夹到 high（防迟滞失效）；interval 过小 → 夹到下限。
	values := map[string]string{
		bpKeyDiskHighPct:      "80",
		bpKeyDiskLowPct:       "90", // > high
		bpKeyInterval:         "1",  // < min 5s
		bpKeyEnabled:          "false",
		bpKeyIOSomeHigh:       "30",
		bpKeyIOSomeLow:        "50", // > high
		bpKeyMaxInflight:      "12",
		bpKeyQueuePendingHigh: "100",
		bpKeyQueuePendingLow:  "200",
		bpKeyQueueOldestHigh:  "60",
		bpKeyQueueOldestLow:   "120",
		bpKeyQueueSlopeWindow: "1",
	}
	lookup := func(_ context.Context, category, key string) (string, bool) {
		if category != BackpressureCategory {
			return "", false
		}
		v, ok := values[key]
		return v, ok
	}
	cfg := loadBackpressureConfig(context.Background(), lookup)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, 80.0, cfg.DiskHighPct)
	assert.Equal(t, 80.0, cfg.DiskLowPct) // clamped to high
	assert.Equal(t, 30.0, cfg.IOSomeHighPct)
	assert.Equal(t, 30.0, cfg.IOSomeLowPct)
	assert.Equal(t, int64(12), cfg.MaxInflight)
	assert.Equal(t, bpMinInterval, cfg.Interval)
	assert.Equal(t, 100, cfg.QueuePendingHigh)
	assert.Equal(t, 100, cfg.QueuePendingLow)
	assert.Equal(t, time.Minute, cfg.QueueOldestHigh)
	assert.Equal(t, time.Minute, cfg.QueueOldestLow)
	assert.Equal(t, 60*time.Second, cfg.QueueSlopeWindow)
	assert.Equal(t, 2*event.QueueHealthSampleInterval, cfg.QueueSlopeWindow)
}

func TestParseIOSomeAvg10(t *testing.T) {
	got, err := parseIOSomeAvg10("some avg10=47.25 avg60=11.00 avg300=2.00 total=1\nfull avg10=3.00 avg60=1.00 avg300=0.50 total=2\n")
	assert.NoError(t, err)
	assert.InDelta(t, 47.25, got, 0.001)
	_, err = parseIOSomeAvg10("full avg10=3.00 total=2\n")
	assert.Error(t, err)
}

func TestWatchdogAcquireReleaseBoundsInflight(t *testing.T) {
	w := NewWatchdog(nil, nil, NewBackpressureMetrics(nil), nil)
	cfg := *w.cfg.Load()
	cfg.MaxInflight = 2
	w.cfg.Store(&cfg)

	ok, reason := w.Acquire()
	assert.True(t, ok, "第一个上传应准入")
	assert.Empty(t, reason)
	ok, _ = w.Acquire()
	assert.True(t, ok, "第二个上传应准入")
	ok, reason = w.Acquire()
	assert.False(t, ok, "达到上限后快速拒绝")
	assert.Equal(t, rejectReasonInflightLimit, reason)
	assert.Equal(t, int64(2), w.inflight.Load())

	w.Release()
	assert.Equal(t, int64(1), w.inflight.Load())
	ok, _ = w.Acquire()
	assert.True(t, ok, "释放后应恢复准入")

	w.active.Store(true)
	ok, reason = w.Acquire()
	assert.False(t, ok, "IO 背压生效时不准入")
	assert.Equal(t, rejectReasonResourcePressure, reason)
	w.Release()
	w.Release()
	w.Release() // 防御重复释放，不得变成负数。
	assert.Equal(t, int64(0), w.inflight.Load())
}
