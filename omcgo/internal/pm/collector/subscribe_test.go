package collector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type queueSubscriptionCall struct {
	subject string
	queue   string
	handler event.EventHandler
}

type publishCall struct {
	subject string
	event   event.Event
}

// countingBus 记录 PM 主队列和注册等待队列的订阅、接力行为。
type countingBus struct {
	queueSubs  []queueSubscriptionCall
	publishes  []publishCall
	publishErr error
}

func (b *countingBus) Publish(_ context.Context, subject string, evt event.Event) error {
	b.publishes = append(b.publishes, publishCall{subject: subject, event: evt})
	return b.publishErr
}
func (b *countingBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return noopSub{}, nil
}
func (b *countingBus) QueueSubscribe(subject string, queue string, handler event.EventHandler) (event.Subscription, error) {
	b.queueSubs = append(b.queueSubs, queueSubscriptionCall{
		subject: subject,
		queue:   queue,
		handler: handler,
	})
	return noopSub{}, nil
}
func (b *countingBus) PullSubscribe(_ string, queue string, _ event.EventHandler) (event.Subscription, error) {
	b.queueSubs = append(b.queueSubs, queueSubscriptionCall{queue: queue})
	return noopSub{}, nil
}
func (b *countingBus) Close() error { return nil }

func (b *countingBus) handler(subject string) event.EventHandler {
	for _, sub := range b.queueSubs {
		if sub.subject == subject {
			return sub.handler
		}
	}
	return nil
}

type noopSub struct{}

func (noopSub) Unsubscribe() error { return nil }

type discardCall struct {
	bucket string
	object string
}

type recordingRawDiscarder struct {
	calls []discardCall
	err   error
}

func (d *recordingRawDiscarder) RemoveObject(
	_ context.Context,
	bucket string,
	object string,
	_ minio.RemoveObjectOptions,
) error {
	d.calls = append(d.calls, discardCall{bucket: bucket, object: object})
	return d.err
}

func newTestCollector() *PMCollector {
	return NewPMCollector(nil, "", nil, nil, nil, nil, zap.NewNop())
}

// 默认（未设并发）退化为单订阅，沿用旧行为。
func TestPMCollector_Subscribe_DefaultSingleSubscription(t *testing.T) {
	bus := &countingBus{}
	require.NoError(t, newTestCollector().Subscribe(bus))
	require.Len(t, bus.queueSubs, 2)
	assert.Equal(t, event.SubjectPMFileDeferred, bus.queueSubs[0].subject)
	assert.Equal(t, "pm-registration-wait", bus.queueSubs[0].queue)
	assert.Equal(t, event.SubjectPMFileReceived, bus.queueSubs[1].subject)
	assert.Equal(t, "pm-workers", bus.queueSubs[1].queue)
}

// SetConcurrency(N) → 主队列和注册等待队列各 N 个订阅，分别使用独立 durable。
func TestPMCollector_Subscribe_NConcurrency(t *testing.T) {
	bus := &countingBus{}
	c := newTestCollector()
	c.SetConcurrency(4)
	require.NoError(t, c.Subscribe(bus))
	require.Len(t, bus.queueSubs, 8)
	for i := 0; i < 4; i++ {
		assert.Equal(t, event.SubjectPMFileDeferred, bus.queueSubs[i].subject)
		assert.Equal(t, "pm-registration-wait", bus.queueSubs[i].queue)
	}
	for i := 4; i < 8; i++ {
		assert.Equal(t, event.SubjectPMFileReceived, bus.queueSubs[i].subject)
		assert.Equal(t, "pm-workers", bus.queueSubs[i].queue)
	}
}

// 并发数 <=0 兜底为 1，避免零订阅静默吞消息。
func TestPMCollector_Subscribe_NonPositiveFallsBackToOne(t *testing.T) {
	for _, n := range []int{0, -3} {
		bus := &countingBus{}
		c := newTestCollector()
		c.SetConcurrency(n)
		require.NoError(t, c.Subscribe(bus))
		assert.Len(t, bus.queueSubs, 2, "n=%d 应让两个 durable 各兜底为单订阅", n)
	}
}

func TestPMCollector_Subscribe_DeferredDeviceDoesNotOccupyMainConsumer(t *testing.T) {
	bus := &countingBus{}
	c := newTestCollector()
	c.SetDeviceLookup(&fakeDeviceLookup{})
	require.NoError(t, c.Subscribe(bus))

	evt, err := event.NewEvent(event.SubjectPMFileReceived, FileReceivedPayload{
		DeviceSN:  "registration-lag",
		MinIOPath: "registration-lag.xml",
	})
	require.NoError(t, err)
	originalTimestamp := evt.Timestamp

	mainHandler := bus.handler(event.SubjectPMFileReceived)
	require.NotNil(t, mainHandler)
	require.NoError(t, mainHandler(context.Background(), evt),
		"main consumer must ACK after durable handoff instead of retaining an ack-pending slot")
	require.Len(t, bus.publishes, 1)
	assert.Equal(t, event.SubjectPMFileDeferred, bus.publishes[0].subject)
	assert.Equal(t, evt.ID, bus.publishes[0].event.ID)
	assert.Equal(t, originalTimestamp, bus.publishes[0].event.Timestamp,
		"registration grace must remain anchored to the original upload event")

	deferredHandler := bus.handler(event.SubjectPMFileDeferred)
	require.NotNil(t, deferredHandler)
	err = deferredHandler(context.Background(), bus.publishes[0].event)
	require.ErrorIs(t, err, reliability.ErrDeferred)
	assert.Len(t, bus.publishes, 1,
		"the wait consumer must use NATS delayed redelivery, not recursively republish")
}

func TestPMCollector_Subscribe_HandoffFailureKeepsOriginalRetryable(t *testing.T) {
	bus := &countingBus{publishErr: errors.New("NATS unavailable")}
	c := newTestCollector()
	c.SetDeviceLookup(&fakeDeviceLookup{})
	require.NoError(t, c.Subscribe(bus))

	evt, err := event.NewEvent(event.SubjectPMFileReceived, FileReceivedPayload{
		DeviceSN:  "registration-lag",
		MinIOPath: "registration-lag.xml",
	})
	require.NoError(t, err)

	err = bus.handler(event.SubjectPMFileReceived)(context.Background(), evt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "handoff deferred PM event")
}

func TestPMCollector_Subscribe_ExpiredRegistrationDeletesRawAndACKs(t *testing.T) {
	bus := &countingBus{}
	c := NewPMCollector(nil, "pm-files", nil, nil, nil, nil, zap.NewNop())
	discarder := &recordingRawDiscarder{}
	c.SetDeviceLookup(&fakeDeviceLookup{})
	c.SetRawObjectDiscarder(discarder)
	require.NoError(t, c.Subscribe(bus))

	evt, err := event.NewEvent(event.SubjectPMFileReceived, FileReceivedPayload{
		DeviceSN:  "never-registered",
		MinIOPath: "never-registered.xml",
	})
	require.NoError(t, err)
	evt.Timestamp = time.Now().Add(-DeviceRegistrationGrace - time.Second)

	err = bus.handler(event.SubjectPMFileReceived)(context.Background(), evt)
	require.NoError(t, err)
	assert.Empty(t, bus.publishes)
	require.Equal(t, []discardCall{{bucket: "pm-files", object: "never-registered.xml"}}, discarder.calls)
}

func TestPMCollector_Subscribe_ExpiredRegistrationMissingRawACKs(t *testing.T) {
	bus := &countingBus{}
	c := NewPMCollector(nil, "pm-files", nil, nil, nil, nil, zap.NewNop())
	discarder := &recordingRawDiscarder{err: minio.ErrorResponse{Code: "NoSuchKey"}}
	c.SetDeviceLookup(&fakeDeviceLookup{})
	c.SetRawObjectDiscarder(discarder)
	require.NoError(t, c.Subscribe(bus))

	evt, err := event.NewEvent(event.SubjectPMFileDeferred, FileReceivedPayload{
		Bucket:    "custom-pm",
		DeviceSN:  "never-registered",
		MinIOPath: "missing.xml",
	})
	require.NoError(t, err)
	evt.Timestamp = time.Now().Add(-DeviceRegistrationGrace - time.Second)

	require.NoError(t, bus.handler(event.SubjectPMFileDeferred)(context.Background(), evt))
	require.Equal(t, []discardCall{{bucket: "custom-pm", object: "missing.xml"}}, discarder.calls)
}

func TestPMCollector_Subscribe_ExpiredRegistrationDeleteFailureDefersWithoutHandoff(t *testing.T) {
	bus := &countingBus{}
	c := NewPMCollector(nil, "pm-files", nil, nil, nil, nil, zap.NewNop())
	discarder := &recordingRawDiscarder{err: errors.New("MinIO unavailable")}
	c.SetDeviceLookup(&fakeDeviceLookup{})
	c.SetRawObjectDiscarder(discarder)
	require.NoError(t, c.Subscribe(bus))

	evt, err := event.NewEvent(event.SubjectPMFileDeferred, FileReceivedPayload{
		DeviceSN:  "never-registered",
		MinIOPath: "retry-delete.xml",
	})
	require.NoError(t, err)
	evt.Timestamp = time.Now().Add(-DeviceRegistrationGrace - time.Second)

	err = bus.handler(event.SubjectPMFileDeferred)(context.Background(), evt)
	require.ErrorIs(t, err, reliability.ErrDeferred)
	assert.Empty(t, bus.publishes, "deferred lane must NAK instead of recursively handing off")
	require.Len(t, discarder.calls, 1)
}
