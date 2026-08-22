package device

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type concurrencyTrackingInfoSyncer struct {
	mu         sync.Mutex
	active     int
	maxActive  int
	completed  int
	releaseJob <-chan struct{}
}

func (s *concurrencyTrackingInfoSyncer) SyncFromParameters(
	ctx context.Context,
	_ uuid.UUID,
	_ model.CarrierCode,
	_ model.Technology,
	_ string,
) ([]string, error) {
	s.mu.Lock()
	s.active++
	if s.active > s.maxActive {
		s.maxActive = s.active
	}
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.releaseJob:
	}

	s.mu.Lock()
	s.active--
	s.completed++
	s.mu.Unlock()
	return nil, nil
}

func TestAsyncSyncDeviceInfoLimitsConcurrency(t *testing.T) {
	release := make(chan struct{})
	syncer := &concurrencyTrackingInfoSyncer{releaseJob: release}
	processor := &BatchInformProcessor{
		infoSyncer: syncer, logger: zap.NewNop(),
		infoProjectionSlots: make(chan struct{}, maxBatchInformProjectionWorkers),
	}
	updates := make([]*informUpdate, 20)
	for i := range updates {
		updates[i] = &informUpdate{device: &model.Device{
			ID: uuid.New(), SerialNumber: uuid.NewString(),
		}}
	}

	done := make(chan struct{}, 2)
	go func() {
		processor.asyncSyncDeviceInfo(updates[:10])
		done <- struct{}{}
	}()
	go func() {
		processor.asyncSyncDeviceInfo(updates[10:])
		done <- struct{}{}
	}()
	time.Sleep(20 * time.Millisecond)
	close(release)
	<-done
	<-done

	syncer.mu.Lock()
	defer syncer.mu.Unlock()
	assert.Equal(t, 20, syncer.completed)
	assert.LessOrEqual(t, syncer.maxActive, maxBatchInformProjectionWorkers)
	assert.Greater(t, syncer.maxActive, 1)
}

func TestNewBatchInformProcessorClampsHighLoadConfig(t *testing.T) {
	processor := NewBatchInformProcessor(appconfig.BatchProcessorConfig{
		Workers:      8,
		MaxBatchSize: 500,
		InputBuffer:  5000,
	}, nil, nil, nil, nil, nil, nil, zap.NewNop())

	assert.Equal(t, maxBatchInformWorkers, processor.workers)
	assert.Equal(t, maxBatchInformMaxBatchSize, processor.maxBatchSize)
	require.Len(t, processor.workerChans, maxBatchInformWorkers)
	assert.Equal(t, maxBatchInformInputBuffer, cap(processor.workerChans[0]))
	assert.Equal(t, maxBatchInformProjectionWorkers, cap(processor.infoProjectionSlots))
}

func TestNewBatchInformProcessorUsesBackpressureDefaults(t *testing.T) {
	processor := NewBatchInformProcessor(appconfig.BatchProcessorConfig{}, nil, nil, nil, nil, nil, nil, zap.NewNop())

	assert.Equal(t, defaultBatchInformWorkers, processor.workers)
	assert.Equal(t, defaultBatchInformMaxBatchSize, processor.maxBatchSize)
	require.Len(t, processor.workerChans, defaultBatchInformWorkers)
	assert.Equal(t, defaultBatchInformInputBuffer, cap(processor.workerChans[0]))
}

type failingInfoSyncer struct{}

func (failingInfoSyncer) SyncFromParameters(
	context.Context,
	uuid.UUID,
	model.CarrierCode,
	model.Technology,
	string,
) ([]string, error) {
	return nil, errors.New("projection failed")
}

func TestAsyncSyncDeviceInfoRecordsRetryAfterFailure(t *testing.T) {
	deviceID := uuid.New()
	processor := &BatchInformProcessor{
		infoSyncer: failingInfoSyncer{}, logger: zap.NewNop(),
		infoProjectionSlots: make(chan struct{}, maxBatchInformProjectionWorkers),
	}

	processor.asyncSyncDeviceInfo([]*informUpdate{{device: &model.Device{ID: deviceID}}})

	_, pending := processor.infoProjectionRetry.Load(deviceID)
	assert.True(t, pending)
}

type blockingRecordingInfoSyncer struct {
	started chan uuid.UUID
	release chan struct{}
}

func (s *blockingRecordingInfoSyncer) SyncFromParameters(
	ctx context.Context,
	deviceID uuid.UUID,
	_ model.CarrierCode,
	_ model.Technology,
	_ string,
) ([]string, error) {
	s.started <- deviceID
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.release:
		return nil, nil
	}
}

func TestHandleParameterUpsertOutcomeQueuesCommittedSubsetBeforeReturningError(t *testing.T) {
	committedID := uuid.New()
	failedID := uuid.New()
	syncer := &blockingRecordingInfoSyncer{
		started: make(chan uuid.UUID, 2),
		release: make(chan struct{}),
	}
	processor := &BatchInformProcessor{
		infoSyncer: syncer, logger: zap.NewNop(),
		infoProjectionSlots: make(chan struct{}, maxBatchInformProjectionWorkers),
	}
	updates := []*informUpdate{
		{device: &model.Device{ID: committedID, SerialNumber: "committed"}},
		{device: &model.Device{ID: failedID, SerialNumber: "failed"}},
	}
	partialErr := errors.New("contended device write failed")

	err := processor.handleParameterUpsertOutcome(updates, deviceParameterUpsertResult{
		changedDevices: map[uuid.UUID]struct{}{committedID: {}},
	}, partialErr)

	assert.ErrorIs(t, err, partialErr)
	_, retryPending := processor.infoProjectionRetry.Load(committedID)
	assert.True(t, retryPending, "committed parameters need a recoverable projection marker")
	require.Equal(t, committedID, <-syncer.started)
	select {
	case unexpected := <-syncer.started:
		t.Fatalf("uncommitted device %s must not be projected", unexpected)
	default:
	}
	close(syncer.release)
	require.Eventually(t, func() bool {
		_, pending := processor.infoProjectionRetry.Load(committedID)
		return !pending
	}, time.Second, time.Millisecond)
}
