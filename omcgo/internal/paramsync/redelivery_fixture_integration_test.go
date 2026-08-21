package paramsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

func TestResultConsumerRealNATSRedeliveryAfterCommitDoesNotDuplicateBusinessProjection(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	insertReleaseCandidateForTest(t, pool, request.DeviceID, true)
	oldSyncAt := time.Now().UTC().Add(-6 * time.Hour)
	_, err := pool.Exec(ctx, `
UPDATE devices
SET serial_number=$2, last_param_sync_at=$3, last_param_sync_failed_at=$3,
    last_param_sync_error='previous fixture failure'
WHERE id=$1`, request.DeviceID, request.DeviceSN, oldSyncAt)
	require.NoError(t, err)

	runID := uuid.New()
	_, err = pool.Exec(ctx, `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope, status,
  expected_task_count, terminal_task_count, processed_task_count, failed_task_count, started_at
) VALUES ($1,$2,$3,$4,'manual','partial','waiting_device',1,1,0,0,now()-interval '1 minute')`,
		runID, request.ID, request.DeviceID, request.DeviceSN,
	)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`UPDATE parameter_sync_requests SET run_id=$2, active_run_id=$2 WHERE id=$1`,
		request.ID, runID,
	)
	require.NoError(t, err)

	const parameterPath = "Device.X_OMC.Issue372.RedeliverySentinel"
	completed := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: request.DeviceSN, Method: "GetParameterValues",
		Params:     []byte(`{"names":["` + parameterPath + `"]}`),
		CommandKey: "param-sync-" + runID.String() + "-0",
		Source:     task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: request.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(ctx, completed))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM device_tasks WHERE id=$1`, completed.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM device_parameters WHERE device_id=$1`, request.DeviceID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_staging_values WHERE run_id=$1`, runID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_runs WHERE id=$1`, runID)
	})
	taskResult, err := json.Marshal(storedTaskResult{
		StandardParameterValues: []tr069.ParameterValueStruct{{
			Name: parameterPath, Value: "issue-372-redelivery", Type: "xsd:string",
		}},
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
UPDATE device_tasks
SET status='completed', completed_at=now(), result=$2
WHERE id=$1`, completed.ID, taskResult)
	require.NoError(t, err)

	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)
	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("PARAM_SYNC_REDELIVERY_%d", suffix)
	subject := fmt.Sprintf("test.param_sync.task.result.%d", suffix)
	queue := fmt.Sprintf("param-sync-results-%d", suffix)
	durable := queue + "-pull"
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	reg := prometheus.NewRegistry()
	eventMetrics := event.NewEventBusMetrics(reg)
	bus := event.NewNATSEventBus(nc, js, zap.NewNop())
	bus.SetMetrics(eventMetrics)
	bus.SetPullTuning(subject, event.PullTuning{
		BatchSize: 1, Concurrency: 1, AckWait: 300 * time.Millisecond,
		MaxDeliver: 3, MaxAckPending: 1,
	})
	t.Cleanup(func() { _ = bus.Close() })
	paramMetrics := NewMetrics(nil)
	processor := &commitThenFailOnceProcessor{
		inner: NewPGResultProcessor(pool).WithMetrics(paramMetrics),
	}
	consumer := NewResultConsumer(bus, processor).
		WithSubscription(subject, queue).
		WithWorkerConfig(1, 1)
	require.NoError(t, consumer.Start())
	t.Cleanup(func() { _ = consumer.Stop() })

	payload := event.ParamSyncTaskResultPayload{
		EventID: "issue-372-redelivery:" + runID.String(),
		RunID:   runID, RequestID: request.ID, TaskID: completed.ID,
		DeviceID: request.DeviceID, DeviceSN: request.DeviceSN,
		Success: true, ResultRef: "device_tasks:" + completed.ID,
	}
	evt, err := event.NewEvent(subject, payload)
	require.NoError(t, err)
	require.NoError(t, bus.Publish(ctx, subject, evt))

	require.Eventually(t, func() bool {
		return processor.calls.Load() >= 2
	}, 8*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool {
		stats, statsErr := bus.QueueStats(ctx, subject, durable)
		return statsErr == nil &&
			stats.LastSequence == 1 &&
			stats.AckSequence == 1 &&
			stats.Pending == 0 &&
			stats.AckPending == 0 &&
			stats.AckGap == 0
	}, 5*time.Second, 20*time.Millisecond)

	var resultRows, stagingRows, deviceParameterRows, runRows, taskRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM parameter_sync_task_results WHERE run_id=$1 AND task_id=$2`,
		runID, completed.ID,
	).Scan(&resultRows))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM parameter_sync_staging_values WHERE run_id=$1 AND parameter_path=$2`,
		runID, parameterPath,
	).Scan(&stagingRows))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM device_parameters WHERE device_id=$1 AND parameter_path=$2`,
		request.DeviceID, parameterPath,
	).Scan(&deviceParameterRows))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM parameter_sync_runs WHERE request_id=$1`,
		request.ID,
	).Scan(&runRows))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM device_tasks WHERE source='param_sync' AND source_id=$1`,
		runID.String(),
	).Scan(&taskRows))
	assert.Equal(t, 1, resultRows)
	assert.Equal(t, 1, stagingRows)
	assert.Equal(t, 1, deviceParameterRows)
	assert.Equal(t, 1, runRows)
	assert.Equal(t, 1, taskRows)

	stored, err := NewPGRepository(pool).GetRequest(ctx, request.ID)
	require.NoError(t, err)
	assert.Equal(t, RequestStatusSucceeded, stored.Status)
	assert.Nil(t, stored.ActiveRunID)
	var parameterValue string
	var lastParamSyncAt time.Time
	var failedAt *time.Time
	var syncErr *string
	require.NoError(t, pool.QueryRow(ctx, `
SELECT dp.parameter_value, d.last_param_sync_at, d.last_param_sync_failed_at, d.last_param_sync_error
FROM device_parameters dp
JOIN devices d ON d.id=dp.device_id
WHERE dp.device_id=$1 AND dp.parameter_path=$2`, request.DeviceID, parameterPath).
		Scan(&parameterValue, &lastParamSyncAt, &failedAt, &syncErr))
	assert.Equal(t, "issue-372-redelivery", parameterValue)
	assert.True(t, lastParamSyncAt.After(oldSyncAt))
	assert.Nil(t, failedAt)
	assert.Nil(t, syncErr)
	assert.Equal(t, float64(1), testutil.ToFloat64(paramMetrics.ResultRedelivery))
	assert.GreaterOrEqual(t, testutil.ToFloat64(eventMetrics.DeliveryTotal.WithLabelValues(subject, "nak")), float64(1))
	assert.Equal(t, float64(1), testutil.ToFloat64(eventMetrics.DeliveryTotal.WithLabelValues(subject, "ack")))
}

type commitThenFailOnceProcessor struct {
	inner  ResultProcessor
	calls  atomic.Int64
	failed atomic.Bool
}

func (p *commitThenFailOnceProcessor) Process(ctx context.Context, payload event.ParamSyncTaskResultPayload) (ResultProcessOutcome, error) {
	p.calls.Add(1)
	outcome, err := p.inner.Process(ctx, payload)
	if err != nil {
		return outcome, err
	}
	if p.failed.CompareAndSwap(false, true) {
		return outcome, errors.New("simulated ACK link loss after PostgreSQL commit")
	}
	return outcome, nil
}
