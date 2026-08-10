package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/core/event"
	coreoutbox "github.com/omcgo/omcgo/internal/core/outbox"
	"github.com/omcgo/omcgo/internal/geofence"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWireUnknownAlarmFallback_InjectsBothReceivers(t *testing.T) {
	alarmReceiver := &alarm.AlarmReceiver{}
	expeditedReceiver := &alarm.ExpeditedEventReceiver{}
	alarmDefRegistry := &definition.Registry{}
	productRegistry := &product.Registry{}

	alarmReceiver, expeditedReceiver, resolver := wireUnknownAlarmFallback(
		alarmReceiver,
		expeditedReceiver,
		alarmDefRegistry,
		productRegistry,
	)

	require.NotNil(t, resolver)
	require.False(t, nilField(t, alarmReceiver, "alarmDefRegistry"))
	require.False(t, nilField(t, alarmReceiver, "productResolver"))
	require.False(t, nilField(t, expeditedReceiver, "alarmDefRegistry"))
	require.False(t, nilField(t, expeditedReceiver, "productResolver"))
}

func nilField(t *testing.T, target any, fieldName string) bool {
	t.Helper()
	value := reflect.ValueOf(target)
	require.Equal(t, reflect.Ptr, value.Kind())
	field := value.Elem().FieldByName(fieldName)
	require.Truef(t, field.IsValid(), "field %s should exist", fieldName)
	return field.IsNil()
}

func TestEnabledIndicatorLookupCachesByDeviceTypeWithinTTL(t *testing.T) {
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	repo := &fakeEnabledIndicatorRepo{
		ids: map[indicator.DeviceType][]string{
			indicator.DeviceTypeENB: {"C0001", "", "K0001"},
			indicator.DeviceTypeGSM: {"CGSM0010001"},
		},
	}
	lookup := &enabledIndicatorLookup{
		repo:  repo,
		ttl:   5 * time.Minute,
		now:   func() time.Time { return now },
		cache: make(map[indicator.DeviceType]enabledIndicatorCacheEntry),
	}

	first, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, first, "C0001")
	require.Contains(t, first, "K0001")
	require.NotContains(t, first, "")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB])

	delete(first, "C0001")
	first["MUTATED"] = struct{}{}
	second, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, second, "C0001", "调用方修改返回 map 不应污染缓存")
	require.NotContains(t, second, "MUTATED")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB], "TTL 内同一制式不应重复查库")

	gsm, err := lookup.LookupEnabledIndicators(context.Background(), "gsm")
	require.NoError(t, err)
	require.Contains(t, gsm, "CGSM0010001")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeGSM], "不同制式使用独立缓存")

	repo.ids[indicator.DeviceTypeENB] = []string{"C0002"}
	now = now.Add(5*time.Minute + time.Nanosecond)
	expired, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, expired, "C0002")
	require.NotContains(t, expired, "C0001")
	require.Equal(t, 2, repo.calls[indicator.DeviceTypeENB], "TTL 过期后应重新查库")
}

func TestEnabledIndicatorLookupFailureIsNotCached(t *testing.T) {
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	repoErr := errors.New("db down")
	repo := &fakeEnabledIndicatorRepo{
		ids: map[indicator.DeviceType][]string{
			indicator.DeviceTypeENB: {"C0001"},
		},
		err: repoErr,
	}
	lookup := &enabledIndicatorLookup{
		repo:  repo,
		ttl:   5 * time.Minute,
		now:   func() time.Time { return now },
		cache: make(map[indicator.DeviceType]enabledIndicatorCacheEntry),
	}

	_, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.ErrorIs(t, err, repoErr)
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB])

	repo.err = nil
	got, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, got, "C0001")
	require.Equal(t, 2, repo.calls[indicator.DeviceTypeENB], "查询失败不应写入缓存")
}

func TestEnabledIndicatorLookupReloadsImmediatelyWhenCacheVersionChanges(t *testing.T) {
	repo := &fakeEnabledIndicatorRepo{ids: map[indicator.DeviceType][]string{
		indicator.DeviceTypeENB: {"K900010040"},
	}}
	version := "1"
	lookup := &enabledIndicatorLookup{
		repo: repo,
		ttl:  5 * time.Minute,
		now:  time.Now,
		readCacheVersion: func(context.Context) (string, error) {
			return version, nil
		},
		cache: make(map[indicator.DeviceType]enabledIndicatorCacheEntry),
	}

	first, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, first, "K900010040")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB])

	repo.ids[indicator.DeviceTypeENB] = []string{
		"K900010040", "C000190005", "C000190007", "C000190009",
	}
	version = "2"
	second, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, second, "C000190005")
	require.Contains(t, second, "C000190007")
	require.Contains(t, second, "C000190009")
	require.Equal(t, 2, repo.calls[indicator.DeviceTypeENB],
		"cross-process cache version bump must bypass the five-minute TTL")
}

func TestEnabledIndicatorLookupDoesNotPublishLoadFromOlderCacheVersion(t *testing.T) {
	repo := &blockingEnabledIndicatorRepo{
		ids:     []string{"K900010040"},
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	var versionMu sync.RWMutex
	version := "1"
	v2Observed := make(chan struct{})
	var v2Once sync.Once
	lookup := &enabledIndicatorLookup{
		repo: repo,
		ttl:  5 * time.Minute,
		now:  time.Now,
		readCacheVersion: func(context.Context) (string, error) {
			versionMu.RLock()
			defer versionMu.RUnlock()
			if version == "2" {
				v2Once.Do(func() { close(v2Observed) })
			}
			return version, nil
		},
		cache: make(map[indicator.DeviceType]enabledIndicatorCacheEntry),
	}

	type result struct {
		ids map[string]struct{}
		err error
	}
	firstResult := make(chan result, 1)
	go func() {
		ids, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
		firstResult <- result{ids: ids, err: err}
	}()
	<-repo.started

	repo.mu.Lock()
	repo.ids = []string{"K900010040", "C000190005", "C000190007", "C000190009"}
	repo.mu.Unlock()
	versionMu.Lock()
	version = "2"
	versionMu.Unlock()

	secondResult := make(chan result, 1)
	go func() {
		ids, err := lookup.LookupEnabledIndicators(context.Background(), "lte")
		secondResult <- result{ids: ids, err: err}
	}()
	<-v2Observed
	close(repo.release)

	for _, got := range []result{<-firstResult, <-secondResult} {
		require.NoError(t, got.err)
		require.Contains(t, got.ids, "C000190005")
		require.Contains(t, got.ids, "C000190007")
		require.Contains(t, got.ids, "C000190009")
	}
}

type blockingEnabledIndicatorRepo struct {
	mu      sync.Mutex
	ids     []string
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *blockingEnabledIndicatorRepo) ListAll(context.Context, indicator.DeviceType) ([]string, error) {
	r.mu.Lock()
	ids := append([]string(nil), r.ids...)
	r.mu.Unlock()
	blocked := false
	r.once.Do(func() {
		blocked = true
		close(r.started)
	})
	if blocked {
		<-r.release
	}
	return ids, nil
}

type fakeEnabledIndicatorRepo struct {
	ids   map[indicator.DeviceType][]string
	err   error
	calls map[indicator.DeviceType]int
}

func (f *fakeEnabledIndicatorRepo) ListAll(_ context.Context, dt indicator.DeviceType) ([]string, error) {
	if f.calls == nil {
		f.calls = make(map[indicator.DeviceType]int)
	}
	f.calls[dt]++
	if f.err != nil {
		return nil, f.err
	}
	return append([]string(nil), f.ids[dt]...), nil
}

func TestKnownReportKeyLookupCachesByDeviceType(t *testing.T) {
	now := time.Date(2026, 7, 31, 5, 0, 0, 0, time.UTC)
	repo := &fakeKnownReportKeyRepo{
		keys: map[indicator.DeviceType][]string{
			indicator.DeviceTypeENB: {"RRC.AttConn", "", "Cqi.00"},
		},
	}
	lookup := &knownReportKeyLookup{
		repo:  repo,
		ttl:   5 * time.Minute,
		now:   func() time.Time { return now },
		cache: make(map[indicator.DeviceType]knownReportKeyCacheEntry),
	}

	first, err := lookup.LookupKnownReportKeys(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, first, "RRC.AttConn")
	require.Contains(t, first, "Cqi.00")
	require.NotContains(t, first, "")

	second, err := lookup.LookupKnownReportKeys(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, second, "RRC.AttConn")
	require.Equal(t, 1, repo.calls[indicator.DeviceTypeENB], "TTL 内不应重复查指标库")

	repo.keys[indicator.DeviceTypeENB] = []string{"RRC.New"}
	now = now.Add(5*time.Minute + time.Nanosecond)
	expired, err := lookup.LookupKnownReportKeys(context.Background(), "lte")
	require.NoError(t, err)
	require.Contains(t, expired, "RRC.New")
	require.NotContains(t, expired, "RRC.AttConn")
	require.Equal(t, 2, repo.calls[indicator.DeviceTypeENB])
}

type fakeKnownReportKeyRepo struct {
	keys  map[indicator.DeviceType][]string
	err   error
	calls map[indicator.DeviceType]int
}

func (f *fakeKnownReportKeyRepo) ListCounterReportKeys(_ context.Context, dt indicator.DeviceType) ([]string, error) {
	if f.calls == nil {
		f.calls = make(map[indicator.DeviceType]int)
	}
	f.calls[dt]++
	if f.err != nil {
		return nil, f.err
	}
	return append([]string(nil), f.keys[dt]...), nil
}

type workerRelayRepository struct {
	claims chan coreoutbox.ClaimOptions
}

func (r *workerRelayRepository) ClaimDue(
	_ context.Context,
	options coreoutbox.ClaimOptions,
) ([]coreoutbox.Entry, error) {
	select {
	case r.claims <- options:
	default:
	}
	return nil, nil
}

func (r *workerRelayRepository) MarkPublished(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	time.Time,
) (bool, error) {
	return true, nil
}

func (r *workerRelayRepository) MarkFailed(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	string,
	time.Time,
) (bool, error) {
	return true, nil
}

func (r *workerRelayRepository) MarkDead(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	string,
	time.Time,
) (bool, error) {
	return true, nil
}

type workerRelayPublisher struct{}

func (workerRelayPublisher) Publish(
	context.Context,
	string,
	event.Event,
) error {
	return nil
}

func TestStartEventOutboxRelayProcessesImmediatelyAndStops(t *testing.T) {
	repo := &workerRelayRepository{claims: make(chan coreoutbox.ClaimOptions, 1)}

	stop, err := startEventOutboxRelay(
		context.Background(),
		repo,
		workerRelayPublisher{},
		zap.NewNop(),
	)
	require.NoError(t, err)

	select {
	case options := <-repo.claims:
		require.Equal(t, 100, options.Limit)
		require.Equal(t, 10, options.MaxAttempts)
		require.Equal(t, 30*time.Second, options.Lease)
	case <-time.After(time.Second):
		t.Fatal("event outbox relay did not process immediately")
	}

	stopped := make(chan struct{})
	go func() {
		stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("event outbox relay did not stop after cancellation")
	}
}

type workerGeofenceSubscription struct {
	unsubscribed bool
}

func (s *workerGeofenceSubscription) Unsubscribe() error {
	s.unsubscribed = true
	return nil
}

type workerGeofenceBus struct {
	subject  string
	subjects []string
	queue    string
	sub      *workerGeofenceSubscription
}

func (b *workerGeofenceBus) Publish(
	context.Context,
	string,
	event.Event,
) error {
	return nil
}

func (b *workerGeofenceBus) Subscribe(
	string,
	event.EventHandler,
) (event.Subscription, error) {
	panic("unexpected Subscribe")
}

func (b *workerGeofenceBus) QueueSubscribe(
	subject string,
	queue string,
	_ event.EventHandler,
) (event.Subscription, error) {
	b.subject = subject
	b.subjects = append(b.subjects, subject)
	b.queue = queue
	b.sub = &workerGeofenceSubscription{}
	return b.sub, nil
}

func (b *workerGeofenceBus) PullSubscribe(
	string,
	string,
	event.EventHandler,
) (event.Subscription, error) {
	panic("unexpected PullSubscribe")
}

func (b *workerGeofenceBus) Close() error {
	return nil
}

func TestStartGeofenceCoordinatorRegistersWorkerConsumerAndStops(t *testing.T) {
	bus := &workerGeofenceBus{}

	stop, err := startGeofenceCoordinator(nil, bus)
	require.NoError(t, err)
	require.Contains(t, bus.subjects, event.SubjectDeviceLocationObserved)
	require.Contains(t, bus.subjects, event.SubjectGeofenceLifecycleReevaluate)
	require.Equal(t, geofence.CoordinatorQueue, bus.queue)

	require.NoError(t, stop())
	require.True(t, bus.sub.unsubscribed)
}

type workerAsyncJobRepository struct {
	lockedJobTypes chan string
}

func (r *workerAsyncJobRepository) Insert(
	context.Context,
	asyncjob.InsertRequest,
) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *workerAsyncJobRepository) GetByID(
	context.Context,
	uuid.UUID,
) (*asyncjob.Job, error) {
	return nil, asyncjob.ErrNoPendingJob
}

func (r *workerAsyncJobRepository) LockNextPending(
	_ context.Context,
	jobType string,
	_ string,
) (*asyncjob.Job, error) {
	r.lockedJobTypes <- jobType
	return nil, asyncjob.ErrNoPendingJob
}

func (r *workerAsyncJobRepository) UpdateHeartbeat(
	context.Context,
	uuid.UUID,
) error {
	return nil
}

func (r *workerAsyncJobRepository) MarkSucceeded(
	context.Context,
	uuid.UUID,
	json.RawMessage,
) error {
	return nil
}

func (r *workerAsyncJobRepository) MarkFailed(
	context.Context,
	uuid.UUID,
	string,
) error {
	return nil
}

func (r *workerAsyncJobRepository) ListZombies(
	context.Context,
	time.Duration,
) ([]asyncjob.Job, error) {
	return nil, nil
}

func (r *workerAsyncJobRepository) ResetZombie(
	context.Context,
	uuid.UUID,
) error {
	return nil
}

type workerManualBindRepository struct{}

func (workerManualBindRepository) ListPendingManualBindItemIDs(
	context.Context,
	uuid.UUID,
	int,
	int,
) ([]uuid.UUID, error) {
	return nil, nil
}

func (workerManualBindRepository) ProcessManualBindItem(
	context.Context,
	geofence.ProcessManualBindItemRequest,
) (geofence.BatchItemResult, error) {
	return geofence.BatchItemResult{}, nil
}

func (workerManualBindRepository) GetBatchProgress(
	context.Context,
	uuid.UUID,
) (geofence.BatchProgress, error) {
	return geofence.BatchProgress{}, nil
}

func TestRegisterGeofenceManualBindRunnerUsesSharedRegistryAndWorkerStopsOnCancel(
	t *testing.T,
) {
	asyncRepository := &workerAsyncJobRepository{
		lockedJobTypes: make(chan string, 1),
	}
	registry := asyncjob.NewRegistry(
		asyncRepository,
		"worker-test",
		zap.NewNop(),
	)

	runner := registerGeofenceManualBindRunner(
		registry,
		workerManualBindRepository{},
		nil,
	)

	require.Equal(t, geofence.ManualBindJobType, runner.JobType())
	ran, err := registry.RunNext(context.Background(), runner.JobType())
	require.NoError(t, err)
	require.False(t, ran)
	require.Equal(t, geofence.ManualBindJobType, <-asyncRepository.lockedJobTypes)

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		runJobTypeWorker(ctx, registry, runner.JobType(), zap.NewNop())
		close(stopped)
	}()
	cancel()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("geofence manual bind worker did not stop after cancellation")
	}
}
