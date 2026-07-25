package device

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
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
		infoProjectionSlots: make(chan struct{}, 8),
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
	assert.LessOrEqual(t, syncer.maxActive, 8)
	assert.Greater(t, syncer.maxActive, 1)
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
		infoProjectionSlots: make(chan struct{}, 8),
	}

	processor.asyncSyncDeviceInfo([]*informUpdate{{device: &model.Device{ID: deviceID}}})

	_, pending := processor.infoProjectionRetry.Load(deviceID)
	assert.True(t, pending)
}
