package device

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

type transitionPublisherRecorder struct {
	calls []string
}

func (r *transitionPublisherRecorder) ClearDisconnectedAlarmOnOnline(ctx context.Context, device *model.Device) {
	r.calls = append(r.calls, "clear")
}

func (r *transitionPublisherRecorder) PublishDeviceOnlineEvent(ctx context.Context, device *model.Device) {
	r.calls = append(r.calls, "online")
}

func (r *transitionPublisherRecorder) PublishDeviceFirmwareChangedEvent(ctx context.Context, device *model.Device, oldVersion, newVersion string, becameOnline bool) {
	r.calls = append(r.calls, "firmware")
}

func TestBatchInformProcessor_PublishTransitionEvents_ClearsDisconnectedAlarmBeforeOnline(t *testing.T) {
	recorder := &transitionPublisherRecorder{}
	processor := &BatchInformProcessor{transitionPublisher: recorder}
	update := &informUpdate{
		device: &model.Device{
			ID:           uuid.New(),
			SerialNumber: "SN-001",
			Status:       model.DeviceActive,
			IsOnline:     true,
		},
		oldStatus: model.DeviceOffline,
	}

	processor.publishTransitionEvents(context.Background(), update)

	require.Equal(t, []string{"clear", "online"}, recorder.calls)
}

func TestBatchInformProcessor_PublishTransitionEvents_ClearsDisconnectedAlarmBeforeFirmwareChanged(t *testing.T) {
	recorder := &transitionPublisherRecorder{}
	processor := &BatchInformProcessor{transitionPublisher: recorder}
	update := &informUpdate{
		device: &model.Device{
			ID:              uuid.New(),
			SerialNumber:    "SN-002",
			Status:          model.DeviceActive,
			IsOnline:        true,
			FirmwareVersion: "V2",
		},
		oldStatus:  model.DeviceOffline,
		oldVersion: "V1",
	}

	processor.publishTransitionEvents(context.Background(), update)

	require.Equal(t, []string{"clear", "firmware"}, recorder.calls)
}

func TestBatchInformProcessor_PublishTransitionEvents_DoesNotClearDisconnectedAlarmWhenAlreadyActive(t *testing.T) {
	recorder := &transitionPublisherRecorder{}
	processor := &BatchInformProcessor{transitionPublisher: recorder}
	update := &informUpdate{
		device: &model.Device{
			ID:           uuid.New(),
			SerialNumber: "SN-003",
			Status:       model.DeviceActive,
			IsOnline:     true,
		},
		oldStatus: model.DeviceActive,
	}

	processor.publishTransitionEvents(context.Background(), update)

	require.Empty(t, recorder.calls)
}
