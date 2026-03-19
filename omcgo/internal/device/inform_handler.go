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

// Subscribe registers event handlers on the event bus for device Inform events.
func (h *InformHandler) Subscribe(bus event.EventBus) error {
	h.logger.Info("subscribing to device inform events...")

	if _, err := bus.QueueSubscribe(event.SubjectDeviceBootstrap, "device-manager", h.handleBootstrap); err != nil {
		return fmt.Errorf("subscribe bootstrap: %w", err)
	}
	h.logger.Info("subscribed to bootstrap events", zap.String("subject", event.SubjectDeviceBootstrap))

	if _, err := bus.QueueSubscribe(event.SubjectDevicePeriodic, "device-manager", h.handlePeriodic); err != nil {
		return fmt.Errorf("subscribe periodic: %w", err)
	}
	h.logger.Info("subscribed to periodic events", zap.String("subject", event.SubjectDevicePeriodic))

	if _, err := bus.QueueSubscribe(event.SubjectDeviceValueChange, "device-manager", h.handlePeriodic); err != nil {
		return fmt.Errorf("subscribe value_change: %w", err)
	}
	h.logger.Info("subscribed to value_change events", zap.String("subject", event.SubjectDeviceValueChange))

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

	h.logger.Debug("handlePeriodic: calling DeviceService.UpdateFromInform",
		zap.String("serial_number", payload.DeviceId.SerialNumber))

	device, err := h.service.UpdateFromInform(ctx, inform)
	if err != nil {
		h.logger.Error("handlePeriodic: UpdateFromInform failed",
			zap.Error(err),
			zap.String("serial_number", payload.DeviceId.SerialNumber),
		)
		return err
	}

	h.logger.Debug("handlePeriodic: device updated successfully",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
	)
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
