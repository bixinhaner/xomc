package geofence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type coordinatorRepositoryStub struct {
	payload         event.DeviceLocationObservedPayload
	now             time.Time
	result          CoordinatorResult
	err             error
	calls           int
	reevaluateCalls int
}

func (r *coordinatorRepositoryStub) EvaluateLocation(
	_ context.Context,
	payload event.DeviceLocationObservedPayload,
	now time.Time,
) (CoordinatorResult, error) {
	r.calls++
	r.payload = payload
	r.now = now
	return r.result, r.err
}

func (r *coordinatorRepositoryStub) ReevaluateLatest(
	_ context.Context,
	_ uuid.UUID,
	_ time.Time,
) (CoordinatorResult, error) {
	r.reevaluateCalls++
	return r.result, r.err
}

type coordinatorSubscriptionStub struct {
	unsubscribeCalls int
	err              error
}

func (s *coordinatorSubscriptionStub) Unsubscribe() error {
	s.unsubscribeCalls++
	return s.err
}

type coordinatorEventBusStub struct {
	subject      string
	subjects     []string
	queue        string
	handler      event.EventHandler
	subscription event.Subscription
	err          error
}

func (b *coordinatorEventBusStub) Publish(
	context.Context,
	string,
	event.Event,
) error {
	return nil
}

func (b *coordinatorEventBusStub) Subscribe(
	string,
	event.EventHandler,
) (event.Subscription, error) {
	panic("unexpected broadcast subscription")
}

func (b *coordinatorEventBusStub) QueueSubscribe(
	subject string,
	queue string,
	handler event.EventHandler,
) (event.Subscription, error) {
	b.subject = subject
	b.subjects = append(b.subjects, subject)
	b.queue = queue
	b.handler = handler
	return b.subscription, b.err
}

func (b *coordinatorEventBusStub) PullSubscribe(
	string,
	string,
	event.EventHandler,
) (event.Subscription, error) {
	panic("unexpected pull subscription")
}

func (b *coordinatorEventBusStub) Close() error {
	return nil
}

func TestCoordinatorStartRegistersQueueGroupAndStopUnsubscribes(t *testing.T) {
	subscription := &coordinatorSubscriptionStub{}
	bus := &coordinatorEventBusStub{subscription: subscription}
	coordinator := NewCoordinator(&coordinatorRepositoryStub{})

	require.NoError(t, coordinator.Start(bus))
	assert.Contains(t, bus.subjects, event.SubjectDeviceLocationObserved)
	assert.Contains(t, bus.subjects, event.SubjectGeofenceLifecycleReevaluate)
	assert.Equal(t, CoordinatorQueue, bus.queue)
	require.NotNil(t, bus.handler)

	require.NoError(t, coordinator.Stop())
	assert.Equal(t, 2, subscription.unsubscribeCalls)
	require.NoError(t, coordinator.Stop(), "stop must be idempotent")
	assert.Equal(t, 2, subscription.unsubscribeCalls)
}

func TestCoordinatorHandleDecodesImmutablePayloadAndDelegates(t *testing.T) {
	evaluatedAt := time.Date(2026, 7, 30, 13, 0, 0, 0, time.UTC)
	repository := &coordinatorRepositoryStub{}
	coordinator := NewCoordinator(repository)
	coordinator.now = func() time.Time { return evaluatedAt }
	want := event.DeviceLocationObservedPayload{
		DeviceID:           uuid.New(),
		SerialNumber:       "SN-1",
		Carrier:            "cmcc",
		ObservationVersion: 8,
		Latitude:           30.25,
		Longitude:          120.16,
		ObservedAt:         evaluatedAt.Add(-time.Minute),
		ReceivedAt:         evaluatedAt.Add(-time.Minute),
		SourcePath:         "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common",
	}
	payload, err := json.Marshal(want)
	require.NoError(t, err)

	err = coordinator.Handle(context.Background(), event.Event{
		ID:      uuid.NewString(),
		Subject: event.SubjectDeviceLocationObserved,
		Payload: payload,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, repository.calls)
	assert.Equal(t, want, repository.payload)
	assert.Equal(t, evaluatedAt, repository.now)
}

func TestCoordinatorHandleAcknowledgesRepositoryNoOp(t *testing.T) {
	repository := &coordinatorRepositoryStub{
		result: CoordinatorResult{
			NoOp:   true,
			Reason: "duplicate_observation",
		},
	}
	coordinator := NewCoordinator(repository)
	payload, err := json.Marshal(event.DeviceLocationObservedPayload{
		DeviceID:           uuid.New(),
		SerialNumber:       "SN-1",
		Carrier:            "cmcc",
		ObservationVersion: 8,
		ObservedAt:         time.Now().UTC(),
		ReceivedAt:         time.Now().UTC(),
	})
	require.NoError(t, err)

	err = coordinator.Handle(context.Background(), event.Event{
		ID:      uuid.NewString(),
		Subject: event.SubjectDeviceLocationObserved,
		Payload: payload,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, repository.calls)
}

func TestCoordinatorHandleLifecycleReevaluateDelegatesToLatest(t *testing.T) {
	repository := &coordinatorRepositoryStub{}
	coordinator := NewCoordinator(repository)
	deviceID := uuid.New()
	payload, err := json.Marshal(event.GeofenceLifecycleReevaluatePayload{
		DeviceID:   deviceID,
		GeofenceID: uuid.New(),
		Reason:     "lifecycle:disabled",
	})
	require.NoError(t, err)

	err = coordinator.HandleLifecycleReevaluate(context.Background(), event.Event{
		Subject: event.SubjectGeofenceLifecycleReevaluate,
		Payload: payload,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, repository.reevaluateCalls)
}

func TestCoordinatorHandleReturnsDecodeAndDependencyErrorsForRetry(t *testing.T) {
	dependencyErr := errors.New("postgres unavailable")
	repository := &coordinatorRepositoryStub{err: dependencyErr}
	coordinator := NewCoordinator(repository)

	err := coordinator.Handle(context.Background(), event.Event{
		ID:      uuid.NewString(),
		Subject: event.SubjectDeviceLocationObserved,
		Payload: json.RawMessage(`not-json`),
	})
	require.Error(t, err)
	assert.Zero(t, repository.calls)

	payload, marshalErr := json.Marshal(event.DeviceLocationObservedPayload{
		DeviceID:           uuid.New(),
		SerialNumber:       "SN-1",
		Carrier:            "cmcc",
		ObservationVersion: 8,
		ObservedAt:         time.Now().UTC(),
		ReceivedAt:         time.Now().UTC(),
	})
	require.NoError(t, marshalErr)
	err = coordinator.Handle(context.Background(), event.Event{
		ID:      uuid.NewString(),
		Subject: event.SubjectDeviceLocationObserved,
		Payload: payload,
	})
	require.ErrorIs(t, err, dependencyErr)
	assert.Equal(t, 1, repository.calls)
}
