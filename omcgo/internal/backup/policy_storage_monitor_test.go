package backup

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/core/event"
)

// fakeBucketLister implements BucketLister with deterministic ObjectInfo
// streaming. err: when non-nil, the first ObjectInfo carries Err and the
// channel closes — matching minio-go's stream-error contract.
type fakeBucketLister struct {
	objects []minio.ObjectInfo
	err     error
	buckets []string
}

func (f *fakeBucketLister) ListObjects(_ context.Context, bucket string, _ minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	f.buckets = append(f.buckets, bucket)
	ch := make(chan minio.ObjectInfo, len(f.objects)+1)
	if f.err != nil {
		ch <- minio.ObjectInfo{Err: f.err}
		close(ch)
		return ch
	}
	for _, o := range f.objects {
		ch <- o
	}
	close(ch)
	return ch
}

// fakeEventBus is a minimal event.EventBus capturing every Publish call.
// Concurrent-safe so RunStorageCheckOnce can call it while tests inspect.
type fakeEventBus struct {
	mu        sync.Mutex
	published []capturedEvent
	publishFn func(ctx context.Context, subject string, evt event.Event) error
}

type capturedEvent struct {
	subject string
	evt     event.Event
}

// decodePayload deserialises the captured event into a StorageThresholdAlarmPayload.
func (c capturedEvent) decodePayload(t *testing.T) StorageThresholdAlarmPayload {
	t.Helper()
	var p StorageThresholdAlarmPayload
	if err := c.evt.DecodePayload(&p); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return p
}

func (b *fakeEventBus) Publish(ctx context.Context, subject string, evt event.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.publishFn != nil {
		if err := b.publishFn(ctx, subject, evt); err != nil {
			return err
		}
	}
	b.published = append(b.published, capturedEvent{subject: subject, evt: evt})
	return nil
}

func (b *fakeEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("Subscribe not used in storage monitor tests")
}

func (b *fakeEventBus) QueueSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("QueueSubscribe not used in storage monitor tests")
}

func (b *fakeEventBus) PullSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("PullSubscribe not used in storage monitor tests")
}

func (b *fakeEventBus) Close() error { return nil }

func (b *fakeEventBus) snapshot() []capturedEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]capturedEvent, len(b.published))
	copy(out, b.published)
	return out
}

// makeStoragePolicy returns a fully populated BackupPolicy where the only
// "interesting" knobs for storage tests (MaxStorageGB / AlertThresholdPercent
// / AlertOnFailure) are explicit. Everything else is canonical defaults.
func makeStoragePolicy(maxGB, thresholdPct int, alertOnFailure bool) *BackupPolicy {
	return &BackupPolicy{
		AutoCleanup:           true,
		RetentionDays:         30,
		KeepLastN:             5,
		CleanupTime:           "03:00",
		CompressionLevel:      6,
		CompressionFormat:     "gzip",
		StorageBackend:        "local",
		LocalPath:             "/var/backup/omc",
		EncryptionAlgorithm:   "AES-256-GCM",
		MaxBackupCount:        100,
		MinBackupCount:        3,
		AlertOnFailure:        alertOnFailure,
		MaxStorageGB:          maxGB,
		AlertThresholdPercent: thresholdPct,
	}
}

// gigabyte returns the byte count for n GB. Used to construct test object sizes.
func gigabyte(n int64) int64 { return n * 1024 * 1024 * 1024 }

// newStorageMonitor wires a PolicyMonitor with fakeBucketLister + fakeEventBus.
func newStorageMonitor(t *testing.T, policy *BackupPolicy, lister BucketLister, bus event.EventBus) (*PolicyMonitor, *PolicyMetrics) {
	t.Helper()
	policySvc := NewPolicyService(&monPolicyRepo{current: policy}, zap.NewNop())
	metrics := NewPolicyMetrics(nil)
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, metrics, zap.NewNop())
	mon.SetBucketLister(lister)
	mon.SetEventBus(bus)
	return mon, metrics
}

// V1 — below threshold: no alarm, success metric incremented.
func TestStorageCheck_BelowThreshold_NoAlarm(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(5)}}, // 5 GB / 10 GB = 50%
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	used, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, gigabyte(5), used, "used bytes should sum to 5 GB")
	assert.Empty(t, bus.snapshot(), "no alarm published below threshold")
	mon.storageMu.Lock()
	defer mon.storageMu.Unlock()
	assert.False(t, mon.lastAboveThreshold, "edge state stays below")
}

func TestStorageCheck_UsesValidPhysicalBackupBucket(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	lister := &fakeBucketLister{}
	mon, _ := newStorageMonitor(t, policy, lister, &fakeEventBus{})

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"config-backup"}, lister.buckets)
	require.NoError(t, s3utils.CheckValidBucketNameStrict(lister.buckets[0]))
}

// V2 — first crossing above threshold publishes alarm.raised once.
func TestStorageCheck_FirstAboveThreshold_PublishesRaised(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true) // threshold = 8 GB
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(9)}}, // 90% > 80%
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	events := bus.snapshot()
	require.Len(t, events, 1, "exactly one alarm.raised on below→above transition")
	assert.Equal(t, event.SubjectAlarmRaised, events[0].subject)

	payload := events[0].decodePayload(t)
	assert.Equal(t, "backup", payload.Source)
	assert.Equal(t, "major", payload.Severity)
	assert.Equal(t, storageThresholdIdentifier, payload.Identifier)
	assert.Equal(t, "config-backup", payload.BucketName)
	assert.Equal(t, gigabyte(9), payload.UsedBytes)
	assert.Equal(t, gigabyte(10), payload.CapacityBytes)
	assert.Equal(t, 90, payload.UsagePercent)
	assert.Equal(t, 80, payload.ThresholdPercent)

	mon.storageMu.Lock()
	defer mon.storageMu.Unlock()
	assert.True(t, mon.lastAboveThreshold, "edge state should now be above")
}

// V3 — above-then-still-above does NOT republish (skipped metric only).
func TestStorageCheck_StillAbove_NoRepublish(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(9)}},
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	// First tick: above → publish raised
	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	require.Len(t, bus.snapshot(), 1, "first tick publishes raised")

	// Second tick: still above (now 9.5 GB)
	lister.objects = []minio.ObjectInfo{{Size: gigabyte(9) + gigabyte(1)/2}}
	_, err = mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	assert.Len(t, bus.snapshot(), 1, "second tick must NOT republish; state already above")
}

// V4 — above→below publishes alarm.cleared once.
func TestStorageCheck_DropsBelow_PublishesCleared(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	bus := &fakeEventBus{}

	// Start in above state via direct field set (simulates "process already
	// detected above on a previous tick").
	policySvc := NewPolicyService(&monPolicyRepo{current: policy}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
	mon.SetEventBus(bus)
	mon.storageMu.Lock()
	mon.lastAboveThreshold = true
	mon.storageMu.Unlock()

	// Now drop usage to 60% — below 80% threshold.
	mon.SetBucketLister(&fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(6)}},
	})

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	events := bus.snapshot()
	require.Len(t, events, 1, "exactly one alarm.cleared on above→below transition")
	assert.Equal(t, event.SubjectAlarmCleared, events[0].subject)

	payload := events[0].decodePayload(t)
	assert.Equal(t, storageThresholdIdentifier, payload.Identifier,
		"cleared alarm uses same identifier so F04 engine can pair with raised")
	assert.Equal(t, 60, payload.UsagePercent)

	mon.storageMu.Lock()
	defer mon.storageMu.Unlock()
	assert.False(t, mon.lastAboveThreshold, "edge state should now be below")
}

// V5 — list error fail-open: failure metric, no panic, no alarm.
func TestStorageCheck_ListError_FailOpen(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	lister := &fakeBucketLister{err: errors.New("connection refused")}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.Error(t, err, "list error should propagate to caller for visibility")
	assert.Empty(t, bus.snapshot(), "no alarm on list failure")
	mon.storageMu.Lock()
	defer mon.storageMu.Unlock()
	assert.False(t, mon.lastAboveThreshold, "edge state untouched on list failure")
}

func TestStorageCheck_RequestCancellationDoesNotInflateFailureMetricOrWarning(t *testing.T) {
	for _, queryErr := range []error{
		context.Canceled,
		context.DeadlineExceeded,
		fmt.Errorf("list backup bucket: %w", context.Canceled),
		fmt.Errorf("list backup bucket: %w", context.DeadlineExceeded),
	} {
		core, logs := observer.New(zap.WarnLevel)
		metrics := NewPolicyMetrics(nil)
		policySvc := NewPolicyService(&monPolicyRepo{current: makeStoragePolicy(10, 80, true)}, zap.NewNop())
		mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, metrics, zap.New(core))
		mon.SetBucketLister(&fakeBucketLister{err: queryErr})
		mon.SetEventBus(&fakeEventBus{})

		_, err := mon.RunStorageCheckOnce(context.Background())
		require.ErrorIs(t, err, queryErr)
		assert.Zero(t, testutil.ToFloat64(metrics.storageCheckTotal.WithLabelValues("failure")),
			"request cancellation must not increment storage failure metric")
		assert.Zero(t, logs.Len(), "request cancellation must not emit warning")
	}
}

// V6 — AlertOnFailure=false skips entire check (no list, no alarm).
func TestStorageCheck_AlertOnFailureFalse_Skipped(t *testing.T) {
	policy := makeStoragePolicy(10, 80, false /* alertOnFailure */)
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(99)}}, // even at 990% would alarm if checked
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	assert.Empty(t, bus.snapshot(), "no alarm when AlertOnFailure=false")
}

// V7 — MaxStorageGB=0 skips check (defensive against div-by-zero).
func TestStorageCheck_MaxStorageGBZero_Skipped(t *testing.T) {
	policy := makeStoragePolicy(0, 80, true)
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(100)}},
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	assert.Empty(t, bus.snapshot(), "no alarm when MaxStorageGB=0 — defensive guard")
}

// V8 — process restart cold-start: lastAboveThreshold=false at construction;
// first tick with usage above threshold publishes raised (re-emit acceptable).
func TestStorageCheck_ColdStart_AboveThresholdPublishesOnce(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(9)}},
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	mon.storageMu.Lock()
	require.False(t, mon.lastAboveThreshold, "fresh PolicyMonitor starts in below state")
	mon.storageMu.Unlock()

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	assert.Len(t, bus.snapshot(), 1, "cold start above threshold = re-publish raised once")
}

// V9 — Cron Start does not panic and registers three schedules
// (@daily cleanup + @hourly storage check + @weekly orphan reaper, T-0083).
func TestStorageCheck_CronStartIncludesHourly(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	policySvc := NewPolicyService(&monPolicyRepo{current: policy}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
	mon.SetBucketLister(&fakeBucketLister{})
	mon.SetEventBus(&fakeEventBus{})

	require.NoError(t, mon.Start(context.Background()))
	require.NotNil(t, mon.cron)
	entries := mon.cron.Entries()
	require.Len(t, entries, 3, "Start registers @daily cleanup + @hourly storage check + @weekly orphan reaper")
	mon.Stop()
}

// V10 — backward compat: no SetBucketLister + no SetEventBus =
// RunStorageCheckOnce no-ops without error or panic, no metrics, no alarms.
func TestStorageCheck_NoWiringIsNoOp(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	policySvc := NewPolicyService(&monPolicyRepo{current: policy}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
	// Deliberately skip SetBucketLister + SetEventBus.

	used, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), used)
}

// Boundary — usage exactly at threshold counts as above (HIGH-1 regression).
// Without the >= fix, this test would fail because "exactly 80%" with strict
// > would not raise; with the fix it does raise on first hit.
func TestStorageCheck_ExactlyAtThreshold_PublishesRaised(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(8)}}, // exactly 80%
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	events := bus.snapshot()
	require.Len(t, events, 1, "exactly-at-threshold must raise (HIGH-1 fix)")
	assert.Equal(t, event.SubjectAlarmRaised, events[0].subject)
	assert.Equal(t, 80, events[0].decodePayload(t).UsagePercent)
}

// Boundary — drop clearly below threshold clears the alarm. Note: precision
// loss from the integer-rearranged threshold math (capacity/100*pct truncates
// ~25 MB at 10 GB scale) means "1 byte below" tests would be brittle; the
// rearrangement is intentional (review MED-2 fix). Use a comfortable margin.
func TestStorageCheck_BelowThresholdAfterAbove_Clears(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true) // threshold ≈ 8 GiB
	bus := &fakeEventBus{}
	policySvc := NewPolicyService(&monPolicyRepo{current: policy}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
	mon.SetEventBus(bus)
	mon.storageMu.Lock()
	mon.lastAboveThreshold = true
	mon.storageMu.Unlock()
	mon.SetBucketLister(&fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(7)}}, // 70% — clearly below 80%
	})

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	events := bus.snapshot()
	require.Len(t, events, 1, "70% below 80% threshold must clear")
	assert.Equal(t, event.SubjectAlarmCleared, events[0].subject)
}

// Bonus — over-capacity: usage > capacity yields percent > 100 in payload,
// alarm still fires (severity unchanged at "major"). Covers the %clamp branch.
func TestStorageCheck_OverCapacity_PayloadPercentExceeds100(t *testing.T) {
	policy := makeStoragePolicy(1, 80, true) // threshold = 0.8 GB
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(2) + gigabyte(1)/2}}, // 250%
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	events := bus.snapshot()
	require.Len(t, events, 1)
	payload := events[0].decodePayload(t)
	assert.Equal(t, 250, payload.UsagePercent, "percent must NOT clamp at 100; over-capacity is real")
}

// V6 (T-0084) — storage raised alarm carries policy.AlertSeverity.
func TestStorageCheck_PolicyDrivenSeverity_Raised(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	policy.AlertSeverity = "warning"
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(9)}}, // above
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	events := bus.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, "warning", events[0].decodePayload(t).Severity)
}

// V7 (T-0084) — storage cleared alarm also carries policy.AlertSeverity.
func TestStorageCheck_PolicyDrivenSeverity_Cleared(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	policy.AlertSeverity = "critical"
	bus := &fakeEventBus{}
	policySvc := NewPolicyService(&monPolicyRepo{current: policy}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
	mon.SetEventBus(bus)
	mon.storageMu.Lock()
	mon.lastAboveThreshold = true
	mon.storageMu.Unlock()
	mon.SetBucketLister(&fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(7)}}, // below 80%
	})

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	events := bus.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, event.SubjectAlarmCleared, events[0].subject)
	assert.Equal(t, "critical", events[0].decodePayload(t).Severity,
		"cleared alarm uses same severity as raised so F04 engine pairs correctly")
}

// V8 (T-0084) — empty AlertSeverity falls back to "major" at runtime.
func TestStorageCheck_EmptyAlertSeverity_FallsBackToMajor(t *testing.T) {
	policy := makeStoragePolicy(10, 80, true)
	policy.AlertSeverity = "" // legacy/test edge case
	lister := &fakeBucketLister{
		objects: []minio.ObjectInfo{{Size: gigabyte(9)}},
	}
	bus := &fakeEventBus{}
	mon, _ := newStorageMonitor(t, policy, lister, bus)

	_, err := mon.RunStorageCheckOnce(context.Background())
	require.NoError(t, err)
	events := bus.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, "major", events[0].decodePayload(t).Severity,
		"empty Severity must fall back to alarmSeverityMajor (severityOrDefault)")
}

// Bonus — publisher-level wiring tests (separate from monitor edge logic).
func TestPublishStorageThresholdAlarm_NilBus(t *testing.T) {
	err := PublishStorageThresholdAlarm(context.Background(), nil, NewPolicyMetrics(nil),
		TransitionRaised, StorageInfo{})
	require.Error(t, err)
}

func TestPublishStorageThresholdAlarm_PublishError(t *testing.T) {
	bus := &fakeEventBus{publishFn: func(_ context.Context, _ string, _ event.Event) error {
		return errors.New("nats unavailable")
	}}
	err := PublishStorageThresholdAlarm(context.Background(), bus, NewPolicyMetrics(nil),
		TransitionRaised, StorageInfo{BucketName: "config_backup", CapacityBytes: gigabyte(10)})
	require.Error(t, err)
}

// usagePercent table-driven coverage.
func TestUsagePercent(t *testing.T) {
	cases := []struct {
		name     string
		used     int64
		capacity int64
		want     int
	}{
		{"zero capacity", 100, 0, 0},
		{"negative capacity", 100, -1, 0},
		{"zero usage", 0, gigabyte(10), 0},
		{"half", gigabyte(5), gigabyte(10), 50},
		{"at threshold", gigabyte(8), gigabyte(10), 80},
		{"over capacity", gigabyte(20), gigabyte(10), 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, usagePercent(tc.used, tc.capacity))
		})
	}
}

// storageAlarmRoute table-driven coverage.
func TestStorageAlarmRoute(t *testing.T) {
	info := StorageInfo{
		BucketName: "config_backup", UsedBytes: gigabyte(9), CapacityBytes: gigabyte(10),
		UsagePercent: 90, ThresholdPercent: 80,
	}
	subj, kind, summary := storageAlarmRoute(TransitionRaised, info)
	assert.Equal(t, event.SubjectAlarmRaised, subj)
	assert.Equal(t, "raised", kind)
	assert.Contains(t, summary, "reached 90%")
	assert.Contains(t, summary, "threshold 80%")

	info.UsagePercent = 60
	subj, kind, summary = storageAlarmRoute(TransitionCleared, info)
	assert.Equal(t, event.SubjectAlarmCleared, subj)
	assert.Equal(t, "cleared", kind)
	assert.Contains(t, summary, "dropped to 60%")

	// Unknown transition routes to raised with flagged summary (defensive).
	subj, kind, summary = storageAlarmRoute(StorageTransition(99), info)
	assert.Equal(t, event.SubjectAlarmRaised, subj)
	assert.Equal(t, "raised", kind)
	assert.Contains(t, summary, "unknown transition")
}
