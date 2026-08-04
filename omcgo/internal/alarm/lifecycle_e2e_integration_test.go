package alarm

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/northbound/push"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAlarmLifecycleEndToEnd_CommittedFactsReachIndependentDurables(t *testing.T) {
	dsn := os.Getenv("ALARM_LIFECYCLE_E2E_DSN")
	tsdbDSN := os.Getenv("ALARM_LIFECYCLE_E2E_TSDB_DSN")
	natsURL := os.Getenv("ALARM_LIFECYCLE_NATS_TEST_URL")
	if dsn == "" || tsdbDSN == "" || natsURL == "" {
		t.Skip("set lifecycle main DB, TSDB, and NATS integration environment variables")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var databaseName string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName))
	require.Equal(t, "omcgo_lifecycle_a", databaseName,
		"refusing to run lifecycle E2E against a non-dedicated database")
	tsPool, err := pgxpool.New(ctx, tsdbDSN)
	require.NoError(t, err)
	t.Cleanup(tsPool.Close)
	var tsdbName string
	require.NoError(t, tsPool.QueryRow(ctx, "SELECT current_database()").Scan(&tsdbName))
	require.Equal(t, "omcgo_lifecycle_ts_a", tsdbName,
		"refusing to run lifecycle E2E against a non-dedicated TSDB")

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:        "DOMAIN_ALARM",
		Subjects:    []string{"domain.alarm.>"},
		Retention:   nats.LimitsPolicy,
		Storage:     nats.MemoryStorage,
		AllowDirect: true,
		MaxAge:      time.Hour,
		MaxBytes:    16 << 20,
	})
	require.NoError(t, err, "E2E requires an empty dedicated NATS instance")
	t.Cleanup(func() { _ = js.DeleteStream("DOMAIN_ALARM") })

	bus := event.NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	relayNC, err := nats.Connect(natsURL)
	require.NoError(t, err)
	t.Cleanup(relayNC.Close)
	relayJS, err := relayNC.JetStream()
	require.NoError(t, err)
	relayBus := event.NewNATSEventBus(relayNC, relayJS, zap.NewNop())
	t.Cleanup(func() { _ = relayBus.Close() })
	alarmStore := NewPgAlarmStore(pool, tsPool)
	historyProjector := NewHistoryProjector(alarmStore, bus, 1)
	require.NoError(t, historyProjector.Subscribe())
	t.Cleanup(func() { _ = historyProjector.Close() })

	northboundRepo := push.NewPgOutboxRepository(pool, zap.NewNop())
	pushEngine := push.NewEngine(nil, zap.NewNop())
	pushEngine.SetOutboxRepo(northboundRepo)
	pushEngine.AddTarget(&push.Target{
		ID: "oss-e2e", DataTypes: []string{"alarm"}, RetryCount: 3, Enabled: true,
	})
	northboundConsumer := push.NewAlarmLifecycleConsumer(pushEngine, bus, 1)
	require.NoError(t, northboundConsumer.Subscribe())
	t.Cleanup(func() { _ = northboundConsumer.Close() })

	alarm := validLifecycleAlarm()
	alarm.Status = model.AlarmCleared
	clearedAt := time.Now().UTC().Truncate(time.Millisecond)
	alarm.ClearedAt = &clearedAt
	payloads, err := alarmStore.PersistRaisedAndCleared(ctx, alarm)
	require.NoError(t, err)
	require.Len(t, payloads, 2)

	relay := NewAlarmOutboxRelay(
		NewAlarmOutboxRepository(pool), relayBus, zap.NewNop(),
	).WithOptions(AlarmOutboxRelayOptions{WorkerID: "lifecycle-e2e", BatchSize: 10})
	published, err := relay.RelayOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, published)

	raisedEventID := payloads[0].EventID.String()
	clearedEventID := payloads[1].EventID.String()
	require.Eventually(t, func() bool {
		var historyRows, northboundRows, publishedRows int
		if err := tsPool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM alarms_history WHERE alarm_id = $1 AND alarm_version = $2",
			alarm.ID, payloads[1].AlarmVersion,
		).Scan(&historyRows); err != nil {
			return false
		}
		if err := pool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM northbound_outbox WHERE event_id IN ($1, $2)",
			raisedEventID, clearedEventID,
		).Scan(&northboundRows); err != nil {
			return false
		}
		if err := pool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM alarm_event_outbox WHERE aggregate_id = $1 AND status = 'published'",
			alarm.ID,
		).Scan(&publishedRows); err != nil {
			return false
		}
		return historyRows == 1 && northboundRows == 2 && publishedRows == 2
	}, 5*time.Second, 25*time.Millisecond)

	duplicate := event.Event{
		ID: clearedEventID, Subject: event.SubjectDomainAlarmLifecycleCleared,
		Timestamp: payloads[1].OccurredAt,
	}
	duplicate.Payload, err = json.Marshal(payloads[1])
	require.NoError(t, err)
	require.NoError(t, bus.Publish(ctx, duplicate.Subject, duplicate))
	require.Eventually(t, func() bool {
		return historyProjector.Ready(context.Background()) == nil &&
			northboundConsumer.Ready(context.Background()) == nil
	}, 5*time.Second, 25*time.Millisecond)

	var historyRows, northboundRows int
	require.NoError(t, tsPool.QueryRow(ctx,
		"SELECT COUNT(*) FROM alarms_history WHERE alarm_id = $1 AND alarm_version = $2",
		alarm.ID, payloads[1].AlarmVersion,
	).Scan(&historyRows))
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM northbound_outbox WHERE event_id IN ($1, $2)",
		raisedEventID, clearedEventID,
	).Scan(&northboundRows))
	require.Equal(t, 1, historyRows, "TSDB projection must remain idempotent")
	require.Equal(t, 2, northboundRows, "northbound event/target key must remain idempotent")

	// Losing the Relay's NATS connection must not affect the primary database
	// transaction. The failed publication remains retryable and is delivered
	// after a fresh connection is established.
	relayNC.Close()
	outageAlarm := validLifecycleAlarm()
	outagePayload, err := alarmStore.PersistRaised(ctx, outageAlarm)
	require.NoError(t, err)
	failureNow := time.Now().UTC().Add(time.Second)
	outageRelay := NewAlarmOutboxRelay(
		NewAlarmOutboxRepository(pool), relayBus, zap.NewNop(),
	).WithOptions(AlarmOutboxRelayOptions{
		WorkerID: "lifecycle-e2e-outage", BatchSize: 10,
		RetryBase: time.Millisecond, RetryMax: time.Millisecond,
		Now: func() time.Time { return failureNow },
	})
	published, err = outageRelay.RelayOnce(ctx)
	require.Error(t, err)
	require.Zero(t, published)
	var activeRows, failedRows int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM alarms_active WHERE id = $1", outageAlarm.ID,
	).Scan(&activeRows))
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM alarm_event_outbox WHERE event_id = $1 AND status = 'failed'",
		outagePayload.EventID,
	).Scan(&failedRows))
	require.Equal(t, 1, activeRows, "NATS outage must not roll back the alarm transaction")
	require.Equal(t, 1, failedRows, "failed publication must remain retryable")

	recoveredNC, err := nats.Connect(natsURL)
	require.NoError(t, err)
	t.Cleanup(recoveredNC.Close)
	recoveredJS, err := recoveredNC.JetStream()
	require.NoError(t, err)
	recoveredBus := event.NewNATSEventBus(recoveredNC, recoveredJS, zap.NewNop())
	t.Cleanup(func() { _ = recoveredBus.Close() })
	recoveryRelay := NewAlarmOutboxRelay(
		NewAlarmOutboxRepository(pool), recoveredBus, zap.NewNop(),
	).WithOptions(AlarmOutboxRelayOptions{
		WorkerID: "lifecycle-e2e-recovery", BatchSize: 10,
		Now: func() time.Time { return failureNow.Add(2 * time.Millisecond) },
	})
	published, err = recoveryRelay.RelayOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, published)
	require.Eventually(t, func() bool {
		var rows int
		if err := pool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM northbound_outbox WHERE event_id = $1",
			outagePayload.EventID.String(),
		).Scan(&rows); err != nil {
			return false
		}
		return rows == 1
	}, 5*time.Second, 25*time.Millisecond)
}
