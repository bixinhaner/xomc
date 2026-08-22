package paramsync

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
)

type recordingResultProcessor struct {
	mu       sync.Mutex
	err      error
	calls    int
	payloads []event.ParamSyncTaskResultPayload
}

func (p *recordingResultProcessor) Process(_ context.Context, payload event.ParamSyncTaskResultPayload) (ResultProcessOutcome, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	p.payloads = append(p.payloads, payload)
	return ResultProcessOutcome{}, p.err
}

func TestResultConsumer_ACKOnlyAfterProcessorCommit(t *testing.T) {
	processor := &recordingResultProcessor{err: errors.New("postgres unavailable")}
	consumer := NewResultConsumer(nil, processor)
	payload := event.ParamSyncTaskResultPayload{EventID: uuid.NewString(), RunID: uuid.New(), RequestID: uuid.New(), TaskID: uuid.NewString()}
	evt, err := event.NewEvent(event.SubjectParamSyncTaskResult, payload)
	require.NoError(t, err)

	err = consumer.Handle(context.Background(), evt)
	assert.ErrorContains(t, err, "postgres unavailable")
	assert.Equal(t, 1, processor.calls)

	processor.err = nil
	assert.NoError(t, consumer.Handle(context.Background(), evt))
	assert.Equal(t, 2, processor.calls)
}

func TestResultConsumer_UsesStableShardForSameDevice(t *testing.T) {
	deviceID := uuid.New()
	first := event.ParamSyncTaskResultPayload{DeviceID: deviceID, DeviceSN: "SN-1", RunID: uuid.New(), TaskID: "task-1"}
	second := event.ParamSyncTaskResultPayload{DeviceID: deviceID, DeviceSN: "SN-1", RunID: uuid.New(), TaskID: "task-2"}

	assert.Equal(t, resultShardIndex(first, 8), resultShardIndex(second, 8))
}

func TestResultConsumer_WithWorkerConfigClampsOversizedConfig(t *testing.T) {
	consumer := NewResultConsumer(nil, &recordingResultProcessor{}).
		WithWorkerConfig(16, 512)

	assert.Equal(t, maxResultConsumerShardCount, consumer.shardCount)
	assert.Equal(t, maxResultConsumerQueueDepth, consumer.queueDepth)
}

func TestResultConsumer_WithWorkerConfigUsesBackpressureDefaults(t *testing.T) {
	consumer := NewResultConsumer(nil, &recordingResultProcessor{}).
		WithWorkerConfig(0, 0)

	assert.Equal(t, defaultResultConsumerShardCount, consumer.shardCount)
	assert.Equal(t, defaultResultConsumerQueueDepth, consumer.queueDepth)
}

func TestResultConsumer_WaitsForShardWorkerBeforeAck(t *testing.T) {
	processor := &blockingResultProcessor{release: make(chan struct{})}
	consumer := NewResultConsumer(nil, processor)
	consumer.startWorkers(1, 8)
	t.Cleanup(func() { consumer.stopWorkers() })
	payload := event.ParamSyncTaskResultPayload{EventID: uuid.NewString(), RunID: uuid.New(), RequestID: uuid.New(), TaskID: uuid.NewString(), DeviceSN: "SN-1"}
	evt, err := event.NewEvent(event.SubjectParamSyncTaskResult, payload)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- consumer.Handle(context.Background(), evt) }()

	require.Eventually(t, func() bool { return processor.started.Load() }, time.Second, 10*time.Millisecond)
	select {
	case err := <-done:
		require.NoError(t, err)
		t.Fatal("handler returned before processor finished")
	default:
	}
	close(processor.release)
	require.NoError(t, <-done)
}

type blockingResultProcessor struct {
	started atomicBool
	release chan struct{}
}

func (p *blockingResultProcessor) Process(context.Context, event.ParamSyncTaskResultPayload) (ResultProcessOutcome, error) {
	p.started.Store(true)
	<-p.release
	return ResultProcessOutcome{}, nil
}

type atomicBool struct {
	mu sync.Mutex
	v  bool
}

func (b *atomicBool) Store(v bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.v = v
}

func (b *atomicBool) Load() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.v
}
