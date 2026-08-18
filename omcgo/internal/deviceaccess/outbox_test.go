package deviceaccess

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/require"
)

type outboxExecDB struct {
	storage.DB
	query string
	args  []any
}

func (d *outboxExecDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	d.query = query
	d.args = append([]any(nil), args...)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func TestPgReevaluationQueuePersistsLightweightOutboxRequest(t *testing.T) {
	db := &outboxExecDB{}
	queue := newPgReevaluationQueueWithDB(db)
	queue.now = func() time.Time { return time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC) }

	err := queue.Enqueue(context.Background(), ReevaluationRequest{
		Carrier: " cmcc ", SerialNumber: " SN-1 ", TriggerType: " manual ",
	})

	require.NoError(t, err)
	require.Contains(t, db.query, "INSERT INTO device_access_outbox")
	require.Len(t, db.args, 8)
	require.Equal(t, event.SubjectDeviceAccessReevaluationRequested, db.args[2])
	payload, ok := db.args[4].([]byte)
	require.True(t, ok)
	var request ReevaluationRequest
	require.NoError(t, json.Unmarshal(payload, &request))
	require.Equal(t, "cmcc", request.Carrier)
	require.Equal(t, "SN-1", request.SerialNumber)
	require.Equal(t, "manual", request.TriggerType)
	require.NotEmpty(t, request.TriggerEventID)
	require.NotContains(t, string(payload), "rules")
}

func TestPgReevaluationQueueRejectsIncompleteIdentity(t *testing.T) {
	queue := newPgReevaluationQueueWithDB(&outboxExecDB{})
	require.ErrorIs(t, queue.Enqueue(context.Background(), ReevaluationRequest{
		Carrier: "cmcc", TriggerType: "manual",
	}), ErrSerialNumberRequired)
}

func TestPgGPSProbeQueuePersistsIdempotentOutboxIntent(t *testing.T) {
	db := &outboxExecDB{}
	queue := newPgGPSProbeQueueWithDB(db)
	queue.now = func() time.Time { return time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC) }

	err := queue.EnsureGPSProbe(context.Background(), GPSProbeRequest{
		Carrier: "cmcc", SerialNumber: "SN-1", EvidenceVersion: 3, NeedGPS: true,
	})

	require.NoError(t, err)
	require.Contains(t, db.query, "INSERT INTO device_access_outbox")
	require.Contains(t, db.query, "ON CONFLICT (event_key) DO NOTHING")
	require.Equal(t, event.SubjectDeviceAccessProbeRequested, db.args[2])
	require.Contains(t, db.args[3], "device-access-probe:cmcc:SN-1:v3")
}

type outboxHandlerStub struct {
	request ReevaluationRequest
}

type probeHandlerStub struct{ request GPSProbeRequest }

func (h *probeHandlerStub) EnsureGPSProbe(_ context.Context, request GPSProbeRequest) error {
	h.request = request
	return nil
}

func (h *outboxHandlerStub) Handle(_ context.Context, request ReevaluationRequest) error {
	h.request = request
	return nil
}

func TestReevaluationConsumerDecodesDurableRequest(t *testing.T) {
	handler := &outboxHandlerStub{}
	consumer := NewReevaluationConsumer(nil, handler)
	evt, err := event.NewEvent(event.SubjectDeviceAccessReevaluationRequested, ReevaluationRequest{
		Carrier: "cmcc", SerialNumber: "SN-2", TriggerType: "list_changed",
	})
	require.NoError(t, err)

	require.NoError(t, consumer.Handle(context.Background(), evt))
	require.Equal(t, "SN-2", handler.request.SerialNumber)
	require.Equal(t, "list_changed", handler.request.TriggerType)
}

func TestGPSProbeConsumerDecodesDurableRequest(t *testing.T) {
	handler := &probeHandlerStub{}
	consumer := NewGPSProbeConsumer(nil, handler)
	evt, err := event.NewEvent(event.SubjectDeviceAccessProbeRequested, GPSProbeRequest{
		Carrier: "cmcc", SerialNumber: "SN-2", EvidenceVersion: 4, NeedTAC: true,
	})
	require.NoError(t, err)

	require.NoError(t, consumer.Handle(context.Background(), evt))
	require.Equal(t, "SN-2", handler.request.SerialNumber)
	require.True(t, handler.request.NeedTAC)
}

func TestOutboxFailureMovesToDeadAfterRetryBudget(t *testing.T) {
	db := &outboxExecDB{}
	dispatcher := newOutboxDispatcherWithDB(db, nil)
	dispatcher.now = func() time.Time { return time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC) }
	item := accessOutboxItem{ID: uuid.New(), Attempts: deviceAccessOutboxMaxAttempts - 1}

	require.NoError(t, dispatcher.markFailure(context.Background(), item, context.DeadlineExceeded))
	require.Contains(t, db.query, "UPDATE device_access_outbox")
	require.True(t, strings.Contains(db.query, "status") && strings.Contains(db.query, "attempts"))
	require.Contains(t, db.args, "dead")
	require.Contains(t, db.args, deviceAccessOutboxMaxAttempts)
}
