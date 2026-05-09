package device

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"go.uber.org/zap"
)

// InformEventPayload is the structure published by the ACS handler on device.inform.* events.
type InformEventPayload struct {
	DeviceId      tr069.DeviceId               `json:"device_id"`
	Events        []string                     `json:"events"`
	ParameterList []tr069.ParameterValueStruct `json:"parameter_list"`
	CurrentTime   string                       `json:"current_time"`
	RetryCount    int                          `json:"retry_count"`
}

// InformHandler subscribes to device Inform events and routes them to DeviceService.
type InformHandler struct {
	service         *DeviceService
	batchProcessor  *BatchInformProcessor
	carrierRegistry *carrier.CarrierRegistry
	defaultCarrier  model.CarrierCode
	logger          *zap.Logger
}

// NewInformHandler creates a new InformHandler.
func NewInformHandler(
	service *DeviceService,
	carrierRegistry *carrier.CarrierRegistry,
	defaultCarrier model.CarrierCode,
	logger *zap.Logger,
) *InformHandler {
	return &InformHandler{
		service:         service,
		carrierRegistry: carrierRegistry,
		defaultCarrier:  defaultCarrier,
		logger:          logger,
	}
}

// SetBatchProcessor enables batch processing for periodic Inform events.
func (h *InformHandler) SetBatchProcessor(bp *BatchInformProcessor) {
	h.batchProcessor = bp
}

// Subscribe registers event handlers on the event bus for device Inform events.
func (h *InformHandler) Subscribe(bus event.EventBus) error {
	h.logger.Info("subscribing to device inform events...")

	if _, err := bus.QueueSubscribe(event.SubjectDeviceBootstrap, "device-mgr-bootstrap", h.handleBootstrap); err != nil {
		return fmt.Errorf("subscribe bootstrap: %w", err)
	}
	h.logger.Info("subscribed to bootstrap events", zap.String("subject", event.SubjectDeviceBootstrap))

	if _, err := bus.QueueSubscribe(event.SubjectDevicePeriodic, "device-mgr-periodic", h.handlePeriodic); err != nil {
		return fmt.Errorf("subscribe periodic: %w", err)
	}
	h.logger.Info("subscribed to periodic events", zap.String("subject", event.SubjectDevicePeriodic))

	if _, err := bus.QueueSubscribe(event.SubjectDeviceValueChange, "device-mgr-valuechange", h.handlePeriodic); err != nil {
		return fmt.Errorf("subscribe value_change: %w", err)
	}
	h.logger.Info("subscribed to value_change events", zap.String("subject", event.SubjectDeviceValueChange))

	if _, err := bus.QueueSubscribe(event.SubjectDeviceRebootComplete, "device-mgr-reboot", h.handleRebootComplete); err != nil {
		return fmt.Errorf("subscribe reboot_complete: %w", err)
	}
	h.logger.Info("subscribed to reboot_complete events", zap.String("subject", event.SubjectDeviceRebootComplete))

	h.logger.Info("inform handler subscribed to device events successfully")
	return nil
}

func (h *InformHandler) handleBootstrap(ctx context.Context, evt event.Event) error {
	h.logger.Info("handleBootstrap: received bootstrap event",
		zap.String("event_id", evt.ID),
		zap.String("subject", evt.Subject),
		zap.Time("timestamp", evt.Timestamp))

	var payload InformEventPayload
	if err := evt.DecodePayload(&payload); err != nil {
		h.logger.Error("handleBootstrap: decode payload failed", zap.Error(err))
		return fmt.Errorf("decode bootstrap payload: %w", err)
	}

	h.logger.Info("handleBootstrap: payload decoded",
		zap.String("serial_number", payload.DeviceId.SerialNumber),
		zap.String("oui", payload.DeviceId.OUI),
		zap.String("product_class", payload.DeviceId.ProductClass),
		zap.Strings("events", payload.Events),
		zap.Int("param_count", len(payload.ParameterList)))

	inform := payloadToInform(payload)

	// Determine carrier from OUI via registry, falling back to default
	carrierCode := h.resolveCarrier(payload.DeviceId.OUI)
	h.logger.Info("handleBootstrap: carrier resolved",
		zap.String("carrier", string(carrierCode)),
		zap.String("oui", payload.DeviceId.OUI))

	h.logger.Info("handleBootstrap: calling DeviceService.RegisterFromInform",
		zap.String("serial_number", payload.DeviceId.SerialNumber))

	device, err := h.service.RegisterFromInform(ctx, inform, carrierCode)
	if err != nil {
		h.logger.Error("handleBootstrap: RegisterFromInform failed",
			zap.Error(err),
			zap.String("serial_number", payload.DeviceId.SerialNumber),
		)
		return err
	}

	h.logger.Info("handleBootstrap: device registered successfully",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
		zap.String("carrier", string(carrierCode)),
	)

	// Always publish device.registered on BOOTSTRAP events,
	// so that provisioning engine triggers auto-discovery/sync
	// for both new and existing devices.
	h.service.PublishDeviceRegistered(ctx, device)

	return nil
}

// handleRebootComplete handles Inform events that indicate a device has
// rebooted and re-attached (event codes "1 BOOT" and/or "M Reboot") but is
// NOT a fresh bootstrap. Responsibilities:
//   - Ensure the device row exists (auto-register on the rare case a reboot
//     Inform arrives before bootstrap, e.g. ACS cache miss after restart);
//   - Update last_inform_at / status / IP / ConnectionRequestURL / firmware
//     via UpdateFromInform;
//   - Atomically increment boot_count and stamp last_boot_at;
//   - Publish device.reboot.abnormal for watchdog/crash reboots (no M Reboot).
//
// Unlike bootstrap it does NOT re-trigger the provisioning engine: a rebooted
// device keeps its previous configuration identity.
func (h *InformHandler) handleRebootComplete(ctx context.Context, evt event.Event) error {
	h.logger.Info("handleRebootComplete: received reboot event",
		zap.String("event_id", evt.ID),
		zap.String("subject", evt.Subject))

	var payload InformEventPayload
	if err := evt.DecodePayload(&payload); err != nil {
		h.logger.Error("handleRebootComplete: decode payload failed", zap.Error(err))
		return fmt.Errorf("decode reboot_complete payload: %w", err)
	}

	inform := payloadToInform(payload)
	sn := payload.DeviceId.SerialNumber

	device, err := h.service.GetBySerialNumber(ctx, sn)
	if err != nil {
		h.logger.Error("handleRebootComplete: device lookup failed",
			zap.Error(err), zap.String("serial_number", sn))
		return err
	}

	if device == nil {
		h.logger.Info("handleRebootComplete: device not found, auto-registering",
			zap.String("serial_number", sn))
		carrierCode := h.resolveCarrier(payload.DeviceId.OUI)
		registered, regErr := h.service.RegisterFromInform(ctx, inform, carrierCode)
		if regErr != nil {
			h.logger.Error("handleRebootComplete: auto-register failed",
				zap.Error(regErr), zap.String("serial_number", sn))
			return regErr
		}
		device = registered
		// First-time registration via reboot_complete path is equivalent to bootstrap;
		// publish device.registered so ProvisioningEngine can route productClass and
		// kick off Path B/C (auto-sync / model upload).
		h.service.PublishDeviceRegistered(ctx, device)
	} else {
		updated, updErr := h.service.UpdateFromInform(ctx, inform)
		if updErr != nil {
			h.logger.Error("handleRebootComplete: UpdateFromInform failed",
				zap.Error(updErr), zap.String("serial_number", sn))
			return updErr
		}
		if updated != nil {
			device = updated
		}
	}

	if _, err := h.service.RecordBootFromInform(ctx, device, payload.Events); err != nil {
		h.logger.Error("handleRebootComplete: RecordBootFromInform failed",
			zap.Error(err), zap.String("serial_number", sn))
		return err
	}

	return nil
}

func (h *InformHandler) handlePeriodic(ctx context.Context, evt event.Event) error {
	h.logger.Debug("handlePeriodic: received periodic event",
		zap.String("event_id", evt.ID),
		zap.String("subject", evt.Subject))

	var payload InformEventPayload
	if err := evt.DecodePayload(&payload); err != nil {
		h.logger.Error("handlePeriodic: decode payload failed", zap.Error(err))
		return fmt.Errorf("decode periodic payload: %w", err)
	}

	h.logger.Debug("handlePeriodic: payload decoded",
		zap.String("serial_number", payload.DeviceId.SerialNumber),
		zap.Strings("events", payload.Events))

	inform := payloadToInform(payload)
	sn := payload.DeviceId.SerialNumber

	// Step 1: Check if device exists (Redis cache → PostgreSQL fallback)
	device, err := h.service.GetBySerialNumber(ctx, sn)
	if err != nil {
		h.logger.Error("handlePeriodic: device lookup failed",
			zap.Error(err), zap.String("serial_number", sn))
		return err
	}

	// Step 2: Device not found → auto-register
	if device == nil {
		h.logger.Info("handlePeriodic: device not found, auto-registering",
			zap.String("serial_number", sn))

		carrierCode := h.resolveCarrier(payload.DeviceId.OUI)
		registered, regErr := h.service.RegisterFromInform(ctx, inform, carrierCode)
		if regErr != nil {
			h.logger.Error("handlePeriodic: auto-register failed",
				zap.Error(regErr), zap.String("serial_number", sn))
			return regErr
		}
		h.logger.Info("handlePeriodic: device auto-registered",
			zap.String("device_id", registered.ID.String()),
			zap.String("serial_number", registered.SerialNumber),
			zap.String("carrier", string(carrierCode)))
		h.service.PublishDeviceRegistered(ctx, registered)
		return nil
	}

	// Step 3: Device exists → update
	// 如果启用了批量处理器，走异步批量路径
	if h.batchProcessor != nil {
		params, _ := prepareDeviceUpdate(device, inform)
		h.batchProcessor.Submit(device, inform, params)
		h.logger.Debug("handlePeriodic: submitted to batch processor",
			zap.String("device_id", device.ID.String()),
			zap.String("serial_number", device.SerialNumber))
		return nil
	}

	// 回退：原有逐条处理逻辑
	updated, err := h.service.UpdateFromInform(ctx, inform)
	if err != nil {
		h.logger.Error("handlePeriodic: UpdateFromInform failed",
			zap.Error(err), zap.String("serial_number", sn))
		return err
	}

	// Stale-cache fall-through: UpdateFromInform returned (nil, nil) because the
	// cache pointed at a row that has since been deleted from PG. Recover by
	// running auto-register so the device is re-created and downstream
	// provisioning fires (otherwise this device would be stuck forever).
	if updated == nil {
		h.logger.Info("handlePeriodic: stale cache fall-through, auto-registering",
			zap.String("serial_number", sn))
		carrierCode := h.resolveCarrier(payload.DeviceId.OUI)
		registered, regErr := h.service.RegisterFromInform(ctx, inform, carrierCode)
		if regErr != nil {
			h.logger.Error("handlePeriodic: stale-cache fall-through register failed",
				zap.Error(regErr), zap.String("serial_number", sn))
			return regErr
		}
		h.service.PublishDeviceRegistered(ctx, registered)
		return nil
	}

	h.logger.Debug("handlePeriodic: device updated",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber))
	return nil
}

// resolveCarrier resolves the carrier code by looking up the OUI in the
// carrier registry's known OUI-ProductClass mappings. Falls back to defaultCarrier.
func (h *InformHandler) resolveCarrier(oui string) model.CarrierCode {
	if h.carrierRegistry != nil {
		if code := h.carrierRegistry.ResolveByOUI(oui); code != "" {
			return code
		}
	}
	return h.defaultCarrier
}

func payloadToInform(p InformEventPayload) *tr069.InformMessage {
	events := make([]tr069.EventStruct, 0, len(p.Events))
	for _, e := range p.Events {
		events = append(events, tr069.EventStruct{EventCode: e})
	}

	return &tr069.InformMessage{
		DeviceId:      p.DeviceId,
		Event:         events,
		ParameterList: p.ParameterList,
	}
}
