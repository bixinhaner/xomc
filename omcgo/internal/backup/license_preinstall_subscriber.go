package backup

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
)

// LicensePreinstallSubscriber delivers a license imported before its device
// exists. Both subjects are required: registered covers first onboarding and
// online covers later reconnects. The durable repository claim coalesces them.
type LicensePreinstallSubscriber struct {
	service *LicenseService
	logger  *zap.Logger
}

func NewLicensePreinstallSubscriber(service *LicenseService, logger *zap.Logger) *LicensePreinstallSubscriber {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LicensePreinstallSubscriber{service: service, logger: logger.Named("license-preinstall")}
}

func (s *LicensePreinstallSubscriber) Subscribe(bus event.EventBus) error {
	if s == nil || s.service == nil || bus == nil {
		return nil
	}
	if _, err := bus.QueueSubscribe(
		event.SubjectDeviceRegistered,
		"license-preinstall-registered",
		s.handleRegistered,
	); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceRegistered, err)
	}
	if _, err := bus.QueueSubscribe(
		event.SubjectDeviceOnline,
		"license-preinstall-online",
		s.handleOnline,
	); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceOnline, err)
	}
	// A reconnect that also changes firmware deliberately suppresses
	// device.online. Subscribe to the replacement event so preinstalls are not
	// stranded on exactly that reconnect.
	if _, err := bus.QueueSubscribe(
		event.SubjectDeviceFirmwareChanged,
		"license-preinstall-firmware-online",
		s.handleFirmwareChanged,
	); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceFirmwareChanged, err)
	}
	return nil
}

func (s *LicensePreinstallSubscriber) handleFirmwareChanged(ctx context.Context, evt event.Event) error {
	var payload device.DeviceFirmwareChangedEvent
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("decode device.firmware.changed payload failed", zap.Error(err))
		return nil
	}
	if !payload.BecameOnline {
		return nil
	}
	return s.service.DispatchPendingLicense(ctx, payload.SerialNumber)
}

func (s *LicensePreinstallSubscriber) handleRegistered(ctx context.Context, evt event.Event) error {
	var payload struct {
		SerialNumber string `json:"serial_number"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("decode device.registered payload failed", zap.Error(err))
		return nil
	}
	return s.service.DispatchPendingLicense(ctx, payload.SerialNumber)
}

func (s *LicensePreinstallSubscriber) handleOnline(ctx context.Context, evt event.Event) error {
	var payload device.DeviceOnlineEvent
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("decode device.online payload failed", zap.Error(err))
		return nil
	}
	return s.service.DispatchPendingLicense(ctx, payload.SerialNumber)
}
