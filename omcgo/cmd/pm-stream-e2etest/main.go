package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/core/event"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"github.com/omcgo/omcgo/internal/pm/streamtest"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type report struct {
	Devices            int           `json:"devices"`
	Slots              int           `json:"slots"`
	MetricsPerDevice   int           `json:"metrics_per_device"`
	EventsPublished    int64         `json:"events_published"`
	PublishFailures    int64         `json:"publish_failures"`
	PublishDurationMS  int64         `json:"publish_duration_ms"`
	TotalDurationMS    int64         `json:"total_duration_ms"`
	ResultRows         int64         `json:"result_rows"`
	SampleCount        int64         `json:"sample_count"`
	Complete           bool          `json:"complete"`
	MissingSlots       int64         `json:"missing_slots"`
	Pending            uint64        `json:"pending"`
	AckPending         int           `json:"ack_pending"`
	DuplicateStable    bool          `json:"duplicate_stable"`
	NATSReplayRestored bool          `json:"nats_replay_restored"`
	TimeoutFinalized   bool          `json:"timeout_finalized"`
	WriteFailureKept   bool          `json:"write_failure_kept_state"`
	ConcurrentSafe     bool          `json:"concurrent_finalizer_safe"`
	OutboxRetrySafe    bool          `json:"outbox_retry_safe"`
	RollupPropagated   bool          `json:"hourly_rollup_propagated"`
	StartedAt          time.Time     `json:"started_at"`
	FinishedAt         time.Time     `json:"finished_at"`
	WindowStart        time.Time     `json:"window_start"`
	WindowEnd          time.Time     `json:"window_end"`
	ResultLatency      time.Duration `json:"-"`
}

func main() {
	var pgDSN, tsDSN, redisAddr, natsURL, output string
	var devices, metrics, concurrency, slots int
	var reset bool
	var replayRecovery bool
	var timeout time.Duration
	flag.StringVar(&pgDSN, "pg", "", "main PostgreSQL DSN")
	flag.StringVar(&tsDSN, "tsdb", "", "TimescaleDB DSN")
	flag.StringVar(&redisAddr, "redis", "127.0.0.1:6380", "Redis address")
	flag.StringVar(&natsURL, "nats", "nats://127.0.0.1:4223", "NATS URL")
	flag.StringVar(&output, "output", "pm-stream-e2etest-result.json", "result JSON path")
	flag.IntVar(&devices, "devices", 10000, "number of devices")
	flag.IntVar(&slots, "slots", 4, "15-minute slots; 4 completes an hourly window")
	flag.IntVar(&metrics, "metrics", 16, "metrics per device event")
	flag.IntVar(&concurrency, "concurrency", 32, "publish and consume concurrency")
	flag.DurationVar(&timeout, "timeout", 4*time.Minute, "end-to-end completion timeout")
	flag.BoolVar(&reset, "reset", false, "clear streaming aggregation state (test environments only)")
	flag.BoolVar(&replayRecovery, "replay-recovery", true, "delete Redis window state and rebuild it from NATS before the final slot")
	flag.Parse()
	if pgDSN == "" || tsDSN == "" || !reset {
		fail(errors.New("-pg, -tsdb and explicit -reset are required"))
	}
	if devices <= 0 || slots != 4 || metrics <= 0 || concurrency <= 0 {
		fail(errors.New("devices/metrics/concurrency must be positive and slots must be 4"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	pgPool := mustPool(ctx, pgDSN)
	defer pgPool.Close()
	tsPool := mustPool(ctx, tsDSN)
	defer tsPool.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()
	nc, err := nats.Connect(natsURL)
	if err != nil {
		fail(fmt.Errorf("connect NATS: %w", err))
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		fail(fmt.Errorf("open JetStream: %w", err))
	}
	if err := resetState(ctx, pgPool, tsPool, redisClient, js); err != nil {
		fail(err)
	}

	logger := zap.NewNop()
	taskRepo := pmstream.NewPgTaskRepository(pgPool, nil)
	windowStart := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour)
	windowEnd := windowStart.Add(time.Hour)
	members := make([]pmstream.TaskMember, 0, devices)
	for index := 0; index < devices; index++ {
		sn := fmt.Sprintf("LOAD-%08d", index+1)
		members = append(members, pmstream.TaskMember{
			DeviceID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(sn)), DeviceSN: sn,
			DimensionKey: "network", DimensionName: "Network",
		})
	}
	rules := make([]pmstream.MetricRule, 0, metrics)
	counters := make([]pmstream.CounterRule, 0, metrics)
	for index := 0; index < metrics; index++ {
		path := fmt.Sprintf("C%04d", index+1)
		rules = append(rules, pmstream.MetricRule{
			MetricID: path, MetricPath: path, MetricType: "counter",
			Aggregation: pmstream.AggregationSum, Dependencies: []string{path},
		})
		counters = append(counters, pmstream.CounterRule{
			MetricPath: path, Aggregation: pmstream.AggregationSum,
		})
	}
	version, err := taskRepo.Save(ctx, pmstream.SaveTaskRequest{
		Name: "PM stream 10000-device verification", Enabled: true,
		Visibility: "private", Creator: "loadtest", Technology: "lte",
		Dimension: pmstream.DimensionNetwork,
		Granularities: []pmstream.Granularity{
			pmstream.GranularityHourly, pmstream.GranularityDaily,
			pmstream.GranularityWeekly, pmstream.GranularityMonthly,
		},
		Metrics: rules, Counters: counters, Members: members,
		Now: windowStart.Truncate(24 * time.Hour).Add(-time.Minute),
	})
	if err != nil {
		fail(fmt.Errorf("create test task version: %w", err))
	}
	snapshot := pmstream.NewSnapshotStore(taskRepo, logger)
	if err := snapshot.Reload(ctx); err != nil {
		fail(fmt.Errorf("load test task snapshot: %w", err))
	}
	store := pmstream.NewRedisWindowStore(redisClient, 45*24*time.Hour)
	if err := store.ValidateConfiguration(ctx); err != nil {
		fail(err)
	}
	windowRepo := pmstream.NewWindowRepository(tsPool)
	finalizer := pmstream.NewFinalizer(windowRepo, store, logger).
		SetConcurrency(concurrency).
		SetSnapshot(snapshot)
	bus := event.NewNATSEventBus(nc, js, logger)
	defer bus.Close()
	bus.SetPullTuning(event.SubjectPMAggregationNormalized, event.PullTuning{
		BatchSize: 100, Concurrency: concurrency, AckWait: 2 * time.Minute,
		MaxAckPending: concurrency * 4,
	})
	for _, subject := range []string{
		event.SubjectPMAggregationHourlyRollup,
		event.SubjectPMAggregationDailyRollup,
	} {
		bus.SetPullTuning(subject, event.PullTuning{
			BatchSize: 100, Concurrency: concurrency, AckWait: 2 * time.Minute,
			MaxAckPending: concurrency * 4,
		})
	}
	consumer := pmstream.NewConsumer(
		bus, snapshot, pmstream.NewMatcher(time.UTC), windowRepo, store, finalizer, logger,
	)
	subscriptions, err := consumer.Subscribe()
	if err != nil {
		fail(err)
	}
	defer func() {
		for _, subscription := range subscriptions {
			_ = subscription.Unsubscribe()
		}
	}()
	rollupRelay := pmstream.NewRollupOutboxRelay(
		pmstream.NewRollupOutboxRepository(tsPool), bus, logger,
	).SetBatch(100)
	go func() { _ = rollupRelay.Run(ctx) }()

	started := time.Now().UTC()
	var published, failed atomic.Int64
	jobs := make(chan struct {
		slot   int
		device int
	})
	var workers sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobs {
				generator := streamtest.Generator{
					Devices: devices, Metrics: metrics,
					SlotStart:  windowStart.Add(time.Duration(job.slot) * 15 * time.Minute),
					Technology: "lte",
				}
				payload, eventErr := generator.Event(job.device)
				if eventErr == nil {
					var envelope event.Event
					envelope, eventErr = event.NewEvent(event.SubjectPMAggregationNormalized, payload)
					if eventErr == nil {
						var data []byte
						data, eventErr = json.Marshal(envelope)
						if eventErr == nil {
							_, eventErr = js.Publish(event.SubjectPMAggregationNormalized, data)
						}
					}
				}
				if eventErr != nil {
					failed.Add(1)
				} else {
					published.Add(1)
				}
			}
		}()
	}
	// Deliberately publish the first three slots out of order.
	for _, slot := range []int{2, 0, 3} {
		for deviceIndex := 0; deviceIndex < devices; deviceIndex++ {
			jobs <- struct {
				slot   int
				device int
			}{slot: slot, device: deviceIndex}
		}
	}
	waitPublished(ctx, &published, int64(devices*3))
	if err := waitConsumerIdle(ctx, js); err != nil {
		fail(err)
	}
	replayRestored := false
	if replayRecovery {
		key := pmstream.WindowKey{
			TaskID: version.TaskID, TaskVersionID: version.VersionID,
			Granularity: pmstream.GranularityHourly, Start: windowStart, End: windowEnd,
		}
		if err := store.Delete(ctx, key); err != nil {
			fail(err)
		}
		recovery := pmstream.NewRecovery(
			js, windowRepo, store, snapshot, pmstream.NewMatcher(time.UTC), logger,
		)
		if err := recovery.ReplayWindow(ctx, key); err != nil {
			fail(fmt.Errorf("rebuild Redis window from NATS: %w", err))
		}
		state, err := store.Read(ctx, key)
		if err != nil {
			fail(err)
		}
		replayRestored = state.ReceivedSlots == int64(devices*3)
		if !replayRestored {
			fail(fmt.Errorf("NATS replay restored %d slots, want %d", state.ReceivedSlots, devices*3))
		}
	}
	for deviceIndex := 0; deviceIndex < devices; deviceIndex++ {
		jobs <- struct {
			slot   int
			device int
		}{slot: 1, device: deviceIndex}
	}
	close(jobs)
	workers.Wait()
	publishFinished := time.Now().UTC()
	if failed.Load() != 0 {
		fail(fmt.Errorf("%d events failed to publish", failed.Load()))
	}

	result, err := waitForResult(
		ctx, tsPool, version.VersionID, windowStart, int64(devices*slots*metrics),
	)
	if err != nil {
		fail(err)
	}
	rollupPropagated, err := waitForHourlyRollup(
		ctx, store, version, windowStart, int64(devices*slots*metrics),
	)
	if err != nil {
		fail(err)
	}
	beforeDuplicate := result
	duplicateGenerator := streamtest.Generator{
		Devices: devices, Metrics: metrics, SlotStart: windowStart, Technology: "lte",
	}
	duplicatePayload, err := duplicateGenerator.Event(0)
	if err != nil {
		fail(err)
	}
	duplicateEnvelope, err := event.NewEvent(event.SubjectPMAggregationNormalized, duplicatePayload)
	if err != nil {
		fail(err)
	}
	duplicateData, _ := json.Marshal(duplicateEnvelope)
	if _, err := js.Publish(event.SubjectPMAggregationNormalized, duplicateData); err != nil {
		fail(err)
	}
	if err := waitConsumerIdle(ctx, js); err != nil {
		fail(err)
	}
	afterDuplicate, err := loadResult(ctx, tsPool, version.VersionID, windowStart)
	if err != nil {
		fail(err)
	}
	faultFinalizer := pmstream.NewFinalizer(windowRepo, store, logger).
		SetConcurrency(concurrency).
		SetSnapshot(snapshot)
	timeoutFinalized, writeFailureKept, concurrentSafe, err := verifyFinalizerFaults(
		ctx, tsPool, windowRepo, store, faultFinalizer, version,
	)
	if err != nil {
		fail(err)
	}
	outboxRetrySafe, err := verifyOutboxRetry(ctx, tsPool, bus, devices, metrics, windowStart)
	if err != nil {
		fail(err)
	}
	info, err := js.ConsumerInfo("PM_AGG_15M", "pm-aggregation-workers-pull")
	if err != nil {
		fail(err)
	}
	finished := time.Now().UTC()
	out := report{
		Devices: devices, Slots: slots, MetricsPerDevice: metrics,
		EventsPublished: published.Load(), PublishFailures: failed.Load(),
		PublishDurationMS: publishFinished.Sub(started).Milliseconds(),
		TotalDurationMS:   finished.Sub(started).Milliseconds(),
		ResultRows:        afterDuplicate.Rows, SampleCount: afterDuplicate.SampleCount,
		Complete: afterDuplicate.Complete, MissingSlots: afterDuplicate.MissingSlots,
		Pending: info.NumPending, AckPending: info.NumAckPending,
		DuplicateStable: beforeDuplicate == afterDuplicate, NATSReplayRestored: replayRestored,
		TimeoutFinalized: timeoutFinalized, WriteFailureKept: writeFailureKept,
		ConcurrentSafe: concurrentSafe, OutboxRetrySafe: outboxRetrySafe,
		RollupPropagated: rollupPropagated,
		StartedAt:        started, FinishedAt: finished, WindowStart: windowStart, WindowEnd: windowEnd,
	}
	writeReport(output, out)
	if out.ResultRows != int64(metrics) || out.SampleCount != int64(devices*slots*metrics) ||
		!out.Complete || out.MissingSlots != 0 || out.Pending != 0 || out.AckPending != 0 ||
		!out.DuplicateStable || (replayRecovery && !out.NATSReplayRestored) ||
		!out.TimeoutFinalized || !out.WriteFailureKept || !out.ConcurrentSafe ||
		!out.OutboxRetrySafe || !out.RollupPropagated {
		fail(fmt.Errorf("end-to-end assertions failed; see %s", output))
	}
}

type failingEventBus struct{}

func (failingEventBus) Publish(context.Context, string, event.Event) error {
	return errors.New("injected NATS outage")
}
func (failingEventBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("unsupported")
}
func (failingEventBus) QueueSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("unsupported")
}
func (failingEventBus) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("unsupported")
}
func (failingEventBus) Close() error { return nil }

func verifyOutboxRetry(
	ctx context.Context,
	pool *pgxpool.Pool,
	bus event.EventBus,
	devices, metrics int,
	windowStart time.Time,
) (bool, error) {
	generator := streamtest.Generator{
		Devices: devices, Metrics: metrics, SlotStart: windowStart, Technology: "lte",
	}
	payload, err := generator.Event(0)
	if err != nil {
		return false, err
	}
	payload.EventID = uuid.New()
	payload.SourceFileID = uuid.New()
	payload.IngestBatchID = uuid.New()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	if err := pmstream.InsertOutbox(ctx, tx, payload, 10*1024*1024); err != nil {
		_ = tx.Rollback(ctx)
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	repo := pmstream.NewOutboxRepository(pool)
	failingRelay := pmstream.NewOutboxRelay(repo, failingEventBus{}, zap.NewNop()).SetBatch(1)
	if _, err := failingRelay.RelayOnce(ctx); err == nil {
		return false, errors.New("expected injected outbox publish failure")
	}
	var publishedAt *time.Time
	var attempts int
	if err := pool.QueryRow(ctx, `
SELECT published_at, publish_attempts
  FROM pm_aggregation_outbox
 WHERE event_id=$1`, payload.EventID).Scan(&publishedAt, &attempts); err != nil {
		return false, err
	}
	if publishedAt != nil || attempts != 1 {
		return false, fmt.Errorf("failed outbox attempt state = published:%v attempts:%d", publishedAt, attempts)
	}
	realRelay := pmstream.NewOutboxRelay(repo, bus, zap.NewNop()).SetBatch(1)
	relayed, err := realRelay.RelayOnce(ctx)
	if err != nil {
		return false, fmt.Errorf("retry outbox publish: %w", err)
	}
	if !relayed {
		return false, errors.New("retry outbox publish did not claim a row")
	}
	if err := pool.QueryRow(ctx, `
SELECT published_at, publish_attempts
  FROM pm_aggregation_outbox
 WHERE event_id=$1`, payload.EventID).Scan(&publishedAt, &attempts); err != nil {
		return false, err
	}
	return publishedAt != nil && attempts == 2, nil
}

func verifyFinalizerFaults(
	ctx context.Context,
	pool *pgxpool.Pool,
	windows *pmstream.WindowRepository,
	store *pmstream.RedisWindowStore,
	finalizer *pmstream.Finalizer,
	version *pmstream.TaskVersionSnapshot,
) (bool, bool, bool, error) {
	base := time.Now().UTC().Truncate(time.Hour).Add(-4 * time.Hour)
	timeoutKey := testWindowKey(base, version)
	timeoutContribution := testContribution(timeoutKey, "timeout", 8, "counter")
	if err := accumulateTestWindow(ctx, windows, store, timeoutContribution); err != nil {
		return false, false, false, err
	}
	if err := finalizer.Finalize(ctx, timeoutKey, pmstream.CloseTimeout); err != nil {
		return false, false, false, fmt.Errorf("timeout finalizer: %w", err)
	}
	var timeoutComplete bool
	var timeoutMissing int64
	if err := pool.QueryRow(ctx, `
SELECT complete, missing_slots
  FROM pm_aggregation_results
 WHERE task_version_id=$1 AND window_start=$2`,
		timeoutKey.TaskVersionID, timeoutKey.Start).
		Scan(&timeoutComplete, &timeoutMissing); err != nil {
		return false, false, false, fmt.Errorf("read timeout result: %w", err)
	}
	timeoutOK := !timeoutComplete && timeoutMissing == 7

	failureKey := testWindowKey(base.Add(time.Hour), version)
	failureContribution := testContribution(
		failureKey, "write-failure", 1, "metric-type-too-long-for-varchar",
	)
	if err := accumulateTestWindow(ctx, windows, store, failureContribution); err != nil {
		return false, false, false, err
	}
	if err := finalizer.Finalize(ctx, failureKey, pmstream.CloseComplete); err == nil {
		return false, false, false, errors.New("expected final write failure")
	}
	var failureStatus string
	if err := pool.QueryRow(ctx, `
SELECT status FROM pm_aggregation_windows
 WHERE task_version_id=$1 AND window_start=$2`,
		failureKey.TaskVersionID, failureKey.Start).Scan(&failureStatus); err != nil {
		return false, false, false, fmt.Errorf("read failed window: %w", err)
	}
	_, stateErr := store.Read(ctx, failureKey)
	failureOK := failureStatus == "failed" && stateErr == nil

	concurrentKey := testWindowKey(base.Add(2*time.Hour), version)
	concurrentContribution := testContribution(concurrentKey, "concurrent", 1, "counter")
	if err := accumulateTestWindow(ctx, windows, store, concurrentContribution); err != nil {
		return false, false, false, err
	}
	var group sync.WaitGroup
	errs := make(chan error, 16)
	for index := 0; index < 16; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			errs <- finalizer.Finalize(ctx, concurrentKey, pmstream.CloseComplete)
		}()
	}
	group.Wait()
	close(errs)
	for finalizeErr := range errs {
		if finalizeErr != nil {
			return false, false, false, fmt.Errorf("concurrent finalizer: %w", finalizeErr)
		}
	}
	var resultRows int64
	var windowStatus string
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM pm_aggregation_results
 WHERE task_version_id=$1 AND window_start=$2`,
		concurrentKey.TaskVersionID, concurrentKey.Start).Scan(&resultRows); err != nil {
		return false, false, false, err
	}
	if err := pool.QueryRow(ctx, `
SELECT status FROM pm_aggregation_windows
 WHERE task_version_id=$1 AND window_start=$2`,
		concurrentKey.TaskVersionID, concurrentKey.Start).Scan(&windowStatus); err != nil {
		return false, false, false, err
	}
	concurrentOK := resultRows == 1 && windowStatus == "published"
	_ = store.Delete(ctx, failureKey)
	return timeoutOK, failureOK, concurrentOK, nil
}

func testWindowKey(start time.Time, version *pmstream.TaskVersionSnapshot) pmstream.WindowKey {
	return pmstream.WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		Granularity: pmstream.GranularityHourly, Start: start, End: start.Add(time.Hour),
	}
}

func testContribution(
	key pmstream.WindowKey,
	source string,
	expected int64,
	metricType string,
) pmstream.Contribution {
	dimension := pmstream.DimensionNetwork
	if metricType == "metric-type-too-long-for-varchar" {
		dimension = pmstream.Dimension("invalid")
	}
	return pmstream.Contribution{
		Key: key, SourceFileID: source + "-" + uuid.NewString(),
		DeviceID: source, SlotStart: key.Start, ExpectedSlots: expected,
		Values: []pmstream.ContributionValue{{
			Dimension: dimension, DimensionKey: "network",
			DimensionName: "Network", Technology: "lte",
			MetricPath: "C0001", MetricType: metricType,
			Operation: pmstream.AggregationSum, Value: 1,
		}},
	}
}

func accumulateTestWindow(
	ctx context.Context,
	windows *pmstream.WindowRepository,
	store *pmstream.RedisWindowStore,
	contribution pmstream.Contribution,
) error {
	if err := windows.EnsureOpen(ctx, contribution); err != nil {
		return err
	}
	result, err := store.Accumulate(ctx, contribution)
	if err != nil {
		return err
	}
	return windows.ObserveReceived(ctx, contribution.Key, result.ReceivedSlots)
}

func waitPublished(ctx context.Context, published *atomic.Int64, expected int64) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for published.Load() < expected {
		select {
		case <-ctx.Done():
			fail(ctx.Err())
		case <-ticker.C:
		}
	}
}

type resultSummary struct {
	Rows         int64
	SampleCount  int64
	Complete     bool
	MissingSlots int64
}

func mustPool(ctx context.Context, dsn string) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fail(err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		fail(err)
	}
	return pool
}

func resetState(
	ctx context.Context,
	pgPool, tsPool *pgxpool.Pool,
	redisClient *redis.Client,
	js nats.JetStreamContext,
) error {
	if _, err := pgPool.Exec(ctx, "TRUNCATE pm_aggregation_tasks CASCADE"); err != nil {
		return fmt.Errorf("reset main aggregation state: %w", err)
	}
	if _, err := tsPool.Exec(ctx, "TRUNCATE pm_aggregation_outbox, pm_aggregation_rollup_outbox, pm_aggregation_counter_rollups, pm_aggregation_results, pm_aggregation_windows"); err != nil {
		return fmt.Errorf("reset TSDB aggregation state: %w", err)
	}
	var cursor uint64
	for {
		keys, next, err := redisClient.Scan(ctx, cursor, "pmagg:*", 1000).Result()
		if err != nil {
			return fmt.Errorf("scan Redis aggregation state: %w", err)
		}
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("reset Redis aggregation state: %w", err)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	streams := []nats.StreamConfig{
		{Name: "PM_AGG_15M", Subjects: []string{"pmaggregation.15m.>"}, Retention: nats.LimitsPolicy, MaxAge: 2 * time.Hour, MaxBytes: 512 << 20, AllowDirect: true, Compression: nats.S2Compression},
		{Name: "PM_AGG_HOURLY", Subjects: []string{"pmaggregation.hourly.>"}, Retention: nats.LimitsPolicy, MaxAge: 48 * time.Hour, MaxBytes: 512 << 20, AllowDirect: true, Compression: nats.S2Compression},
		{Name: "PM_AGG_DAILY", Subjects: []string{"pmaggregation.daily.>"}, Retention: nats.LimitsPolicy, MaxAge: 40 * 24 * time.Hour, MaxBytes: 512 << 20, AllowDirect: true, Compression: nats.S2Compression},
		{Name: "PM_AGG_CONTROL", Subjects: []string{"pmaggregation.control.>"}, Retention: nats.InterestPolicy},
	}
	for _, config := range streams {
		if _, err := js.StreamInfo(config.Name); errors.Is(err, nats.ErrStreamNotFound) {
			if _, err := js.AddStream(&config); err != nil {
				return fmt.Errorf("create %s stream: %w", config.Name, err)
			}
		} else if err != nil {
			return err
		}
		if err := js.PurgeStream(config.Name); err != nil {
			return fmt.Errorf("purge %s test stream: %w", config.Name, err)
		}
	}
	return nil
}

func waitForResult(
	ctx context.Context,
	pool *pgxpool.Pool,
	versionID uuid.UUID,
	windowStart time.Time,
	expectedSamples int64,
) (resultSummary, error) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		result, err := loadResult(ctx, pool, versionID, windowStart)
		if err == nil && result.Complete && result.SampleCount == expectedSamples {
			return result, nil
		}
		if err != nil {
			return resultSummary{}, err
		}
		select {
		case <-ctx.Done():
			return resultSummary{}, fmt.Errorf("wait for complete aggregation result: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForHourlyRollup(
	ctx context.Context,
	store *pmstream.RedisWindowStore,
	version *pmstream.TaskVersionSnapshot,
	hourStart time.Time,
	expectedSamples int64,
) (bool, error) {
	parent, err := pmstream.WindowFor(hourStart, pmstream.GranularityDaily, time.UTC)
	if err != nil {
		return false, err
	}
	key := pmstream.WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		Granularity: pmstream.GranularityDaily, Start: parent.Start, End: parent.End,
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		state, readErr := store.Read(ctx, key)
		if readErr == nil {
			var samples int64
			for _, accumulator := range state.Accumulators {
				samples += accumulator.Count
			}
			if state.ReceivedSlots == 1 && samples == expectedSamples {
				return true, nil
			}
		} else if !errors.Is(readErr, redis.Nil) {
			return false, readErr
		}
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("wait for hourly rollup propagation: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func loadResult(
	ctx context.Context,
	pool *pgxpool.Pool,
	versionID uuid.UUID,
	windowStart time.Time,
) (resultSummary, error) {
	var result resultSummary
	err := pool.QueryRow(ctx, `
SELECT COUNT(*), COALESCE(SUM(sample_count), 0),
       COALESCE(BOOL_AND(complete), false), COALESCE(MAX(missing_slots), 0)
  FROM pm_aggregation_results
 WHERE task_version_id=$1 AND window_start=$2`, versionID, windowStart).
		Scan(&result.Rows, &result.SampleCount, &result.Complete, &result.MissingSlots)
	if err != nil {
		return resultSummary{}, fmt.Errorf("query aggregation result: %w", err)
	}
	return result, nil
}

func waitConsumerIdle(ctx context.Context, js nats.JetStreamContext) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		info, err := js.ConsumerInfo("PM_AGG_15M", "pm-aggregation-workers-pull")
		if err != nil {
			return err
		}
		if info.NumPending == 0 && info.NumAckPending == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func writeReport(path string, value report) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		fail(err)
	}
	fmt.Println(string(data))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
