package device

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/common/event"
	"github.com/omcgo/omcgo/internal/common/model"
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
	service        *DeviceService
	defaultCarrier model.CarrierCode
	logger         *zap.Logger
}

// NewInformHandler creates a new InformHandler.
func NewInformHandler(service *DeviceService, defaultCarrier model.CarrierCode, logger *zap.Logger) *InformHandler {
	return &InformHandler{
		service:        service,
		defaultCarrier: defaultCarrier,
		logger:         logger,
	}
}

// Subscribe registers event handlers on the event bus for device Inform events.
func (h *InformHandler) Subscribe(bus event.EventBus) error {
	if _, err := bus.QueueSubscribe(event.SubjectDeviceBootstrap, "device-manager", h.handleBootstrap); err != nil {
		return fmt.Errorf("subscribe bootstrap: %w", err)
	}
	if _, err := bus.QueueSubscribe(event.SubjectDevicePeriodic, "device-manager", h.handlePeriodic); err != nil {
		return fmt.Errorf("subscribe periodic: %w", err)
	}
	if _, err := bus.QueueSubscribe(event.SubjectDeviceValueChange, "device-manager", h.handlePeriodic); err != nil {
		return fmt.Errorf("subscribe value_change: %w", err)
	}
	h.logger.Info("inform handler subscribed to device events")
	return nil
}

func (h *InformHandler) handleBootstrap(ctx context.Context, evt event.Event) error {
	var payload InformEventPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode bootstrap payload: %w", err)
	}

	inform := payloadToInform(payload)

	// Determine carrier from OUI or use default
	carrier := h.resolveCarrier(payload.DeviceId.OUI)

	device, err := h.service.RegisterFromInform(ctx, inform, carrier)
	if err != nil {
		h.logger.Error("register device from bootstrap",
			zap.Error(err),
			zap.String("serial_number", payload.DeviceId.SerialNumber),
		)
		return err
	}

	h.logger.Info("device registered from bootstrap event",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
	)
	return nil
}

func (h *InformHandler) handlePeriodic(ctx context.Context, evt event.Event) error {
	var payload InformEventPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode periodic payload: %w", err)
	}

	inform := payloadToInform(payload)

	device, err := h.service.UpdateFromInform(ctx, inform)
	if err != nil {
		h.logger.Error("update device from periodic inform",
			zap.Error(err),
			zap.String("serial_number", payload.DeviceId.SerialNumber),
		)
		return err
	}

	h.logger.Debug("device updated from periodic event",
		zap.String("device_id", device.ID.String()),
		zap.String("serial_number", device.SerialNumber),
	)
	return nil
}

func (h *InformHandler) resolveCarrier(oui string) model.CarrierCode {
	// In a full implementation, this would look up OUI → carrier mapping.
	// For now, use the configured default carrier.
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
