package paramsync

import (
	"context"
	"fmt"
	"strings"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

type RequestDeviceResolver interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

type RequestConsumer struct {
	bus     event.EventBus
	service *Service
	devices RequestDeviceResolver
	sub     event.Subscription
}

func NewRequestConsumer(bus event.EventBus, service *Service, devices RequestDeviceResolver) *RequestConsumer {
	return &RequestConsumer{bus: bus, service: service, devices: devices}
}

func (c *RequestConsumer) Start() error {
	if c.bus == nil || c.service == nil || c.devices == nil {
		return fmt.Errorf("parameter sync request consumer dependencies are required")
	}
	sub, err := c.bus.QueueSubscribe(event.SubjectParamSyncRequested, "param-sync-requests", c.Handle)
	if err != nil {
		return fmt.Errorf("subscribe parameter sync requests: %w", err)
	}
	c.sub = sub
	return nil
}

func (c *RequestConsumer) Handle(ctx context.Context, evt event.Event) error {
	var payload event.ParamSyncRequestedPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode parameter sync request event: %w", err)
	}
	dev, err := c.devices.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		return fmt.Errorf("resolve parameter sync request device: %w", err)
	}
	if dev == nil {
		return fmt.Errorf("parameter sync request device %s not found", payload.DeviceSN)
	}
	reason := TriggerReason(strings.TrimSpace(payload.TriggerReason))
	if !reason.Valid() {
		return fmt.Errorf("invalid parameter sync request reason %q", payload.TriggerReason)
	}
	_, err = c.service.Submit(ctx, SubmitCommand{
		DeviceID: dev.ID, DeviceSN: dev.SerialNumber, CallerType: "acs",
		TriggerReason: reason, Scope: SyncScopeReadback,
		RequestedPaths: payload.RequestedPaths, IdempotencyKey: payload.IdempotencyKey,
	})
	return err
}

func (c *RequestConsumer) Stop() error {
	if c.sub == nil {
		return nil
	}
	return c.sub.Unsubscribe()
}
