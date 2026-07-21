package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	coreevent "github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/require"
)

type configDeliveryBus struct {
	publishErr error
	subject    string
	event      coreevent.Event
}

func (b *configDeliveryBus) Publish(_ context.Context, subject string, evt coreevent.Event) error {
	b.subject = subject
	b.event = evt
	return b.publishErr
}

func (b *configDeliveryBus) Subscribe(string, coreevent.EventHandler) (coreevent.Subscription, error) {
	return nil, nil
}

func (b *configDeliveryBus) QueueSubscribe(string, string, coreevent.EventHandler) (coreevent.Subscription, error) {
	return nil, nil
}

func (b *configDeliveryBus) PullSubscribe(string, string, coreevent.EventHandler) (coreevent.Subscription, error) {
	return nil, nil
}

func (b *configDeliveryBus) Close() error { return nil }

func TestACSConfigDeliveryHandlerPublishesDurableBatchIdentity(t *testing.T) {
	bus := &configDeliveryBus{}
	handler := newACSConfigDeliveryHandler(bus)
	work := admin.ConfigApplyWork{Batch: admin.ConfigApplyBatch{
		ID:            uuid.New(),
		Category:      "acs_transfer",
		ConfigVersion: 17,
	}}

	actual, err := handler(context.Background(), work)
	require.NoError(t, err)
	require.Equal(t, coreevent.SubjectSysConfigSaved, bus.subject)
	require.Equal(t, "published", actual["delivery"])
	require.Equal(t, bus.event.ID, actual["event_id"])

	var payload coreevent.SysConfigSavedPayload
	require.NoError(t, bus.event.DecodePayload(&payload))
	require.Equal(t, work.Batch.Category, payload.Category)
	require.Equal(t, work.Batch.ID, payload.BatchID)
	require.Equal(t, work.Batch.ConfigVersion, payload.ConfigVersion)
}

func TestACSConfigDeliveryHandlerReturnsPublishFailure(t *testing.T) {
	bus := &configDeliveryBus{publishErr: errors.New("nats unavailable")}
	handler := newACSConfigDeliveryHandler(bus)

	_, err := handler(context.Background(), admin.ConfigApplyWork{Batch: admin.ConfigApplyBatch{
		ID:            uuid.New(),
		Category:      "acs_transfer",
		ConfigVersion: 18,
	}})

	require.ErrorContains(t, err, "publish ACS transfer config event")
	require.ErrorContains(t, err, "nats unavailable")
}

func TestACSConfigDeliveryHandlerFailsWhenEventBusUnavailable(t *testing.T) {
	handler := newACSConfigDeliveryHandler(nil)

	_, err := handler(context.Background(), admin.ConfigApplyWork{Batch: admin.ConfigApplyBatch{
		ID:            uuid.New(),
		Category:      "acs_transfer",
		ConfigVersion: 19,
	}})

	require.ErrorContains(t, err, "event bus is unavailable")
}
