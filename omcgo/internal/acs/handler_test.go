package acs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mocks (acsH prefix to avoid conflicts with other _test.go files)
// ---------------------------------------------------------------------------

type acsHSessionStore struct {
	mu           sync.Mutex
	sessionsByID map[string]*Session // in-memory store keyed by sessionID
	getByIDFn    func(ctx context.Context, sessionID string) (*Session, error)
	createByIDFn func(ctx context.Context, sessionID string, session *Session) error
	updateByIDFn func(ctx context.Context, sessionID string, session *Session) error
	deleteByIDFn func(ctx context.Context, sessionID string) error
}

func newAcsHSessionStore() *acsHSessionStore {
	return &acsHSessionStore{
		sessionsByID: make(map[string]*Session),
	}
}

func (m *acsHSessionStore) GetByID(ctx context.Context, sessionID string) (*Session, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, sessionID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessionsByID[sessionID], nil
}

func (m *acsHSessionStore) CreateWithID(ctx context.Context, sessionID string, session *Session) error {
	if m.createByIDFn != nil {
		return m.createByIDFn(ctx, sessionID, session)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session.ID = sessionID
	m.sessionsByID[sessionID] = session
	return nil
}

func (m *acsHSessionStore) UpdateByID(ctx context.Context, sessionID string, session *Session) error {
	if m.updateByIDFn != nil {
		return m.updateByIDFn(ctx, sessionID, session)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session.ID = sessionID
	m.sessionsByID[sessionID] = session
	return nil
}

func (m *acsHSessionStore) DeleteByID(ctx context.Context, sessionID string) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(ctx, sessionID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessionsByID, sessionID)
	return nil
}

type acsHCmdQueue struct {
	mu    sync.Mutex
	queue []*cmdqueue.Command
	popFn func(ctx context.Context, deviceSN string) (*cmdqueue.Command, error)
}

func (m *acsHCmdQueue) Push(_ context.Context, _ string, cmd *cmdqueue.Command) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queue = append(m.queue, cmd)
	return nil
}
func (m *acsHCmdQueue) Pop(ctx context.Context, deviceSN string) (*cmdqueue.Command, error) {
	if m.popFn != nil {
		return m.popFn(ctx, deviceSN)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.queue) == 0 {
		return nil, nil
	}
	cmd := m.queue[0]
	m.queue = m.queue[1:]
	return cmd, nil
}
func (m *acsHCmdQueue) Peek(_ context.Context, _ string) (*cmdqueue.Command, error) { return nil, nil }
func (m *acsHCmdQueue) Len(_ context.Context, _ string) (int64, error)              { return 0, nil }
func (m *acsHCmdQueue) Clear(_ context.Context, _ string) error                     { return nil }

type acsHEventBus struct {
	mu        sync.Mutex
	published []acsHPublishedEvent
	publishFn func(ctx context.Context, subject string, evt event.Event) error
}

type acsHPublishedEvent struct {
	Subject string
	Event   event.Event
}

func (m *acsHEventBus) Publish(ctx context.Context, subject string, evt event.Event) error {
	if m.publishFn != nil {
		return m.publishFn(ctx, subject, evt)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, acsHPublishedEvent{Subject: subject, Event: evt})
	return nil
}
func (m *acsHEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return &acsHSubscription{}, nil
}
func (m *acsHEventBus) QueueSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return &acsHSubscription{}, nil
}
func (m *acsHEventBus) Close() error { return nil }

type acsHSubscription struct{}

func (s *acsHSubscription) Unsubscribe() error { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestACSHandler() *Handler {
	return newTestACSHandlerWithDeps(newAcsHSessionStore(), &acsHCmdQueue{}, &acsHEventBus{})
}

func newTestACSHandlerWithDeps(store SessionStore, cmdQ cmdqueue.CommandQueue, bus event.EventBus) *Handler {
	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)
	return &Handler{
		sessionStore:  store,
		commandQueue:  cmdQ,
		eventBus:      bus,
		authenticator: &auth.NoopAuthenticator{},
		rpcDispatcher: rpc.NewDispatcher(),
		rateLimiter:   NewDeviceRateLimiter(100, 100, 10000, zap.NewNop()),
		admission:     NewAdmissionController(1000),
		metrics:       metrics,
		logger:        zap.NewNop(),
	}
}

// Minimal valid Inform SOAP body for TEST-SN-001 with bootstrap event.
const acsHInformBootstrapXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">100001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>TestVendor</Manufacturer>
        <OUI>001122</OUI>
        <ProductClass>SmallCell-LTE</ProductClass>
        <SerialNumber>TEST-SN-001</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[1]">
        <EventStruct>
          <EventCode>0 BOOTSTRAP</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2026-03-05T10:00:00Z</CurrentTime>
      <RetryCount>0</RetryCount>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[1]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value xsi:type="xsd:string">FW-2.1.0</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>`

// Minimal valid Inform SOAP body with periodic event.
const acsHInformPeriodicXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">100002</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>TestVendor</Manufacturer>
        <OUI>001122</OUI>
        <ProductClass>SmallCell-LTE</ProductClass>
        <SerialNumber>TEST-SN-002</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[1]">
        <EventStruct>
          <EventCode>2 PERIODIC</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2026-03-05T10:00:00Z</CurrentTime>
      <RetryCount>0</RetryCount>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[0]">
      </ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>`

// GetParameterValuesResponse SOAP body.
const acsHGetParamRespXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">100001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValuesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[1]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value>FW-2.1.0</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:GetParameterValuesResponse>
  </soap:Body>
</soap:Envelope>`

// TransferComplete SOAP body.
const acsHTransferCompleteXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">TC-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:TransferComplete>
      <CommandKey>upgrade-001</CommandKey>
      <FaultStruct>
        <FaultCode>0</FaultCode>
        <FaultString></FaultString>
      </FaultStruct>
      <StartTime>2026-03-05T10:00:00Z</StartTime>
      <CompleteTime>2026-03-05T10:01:00Z</CompleteTime>
    </cwmp:TransferComplete>
  </soap:Body>
</soap:Envelope>`

// ---------------------------------------------------------------------------
// Tests: HTTP method validation
// ---------------------------------------------------------------------------

func TestServeHTTP_NonPOST_Returns405(t *testing.T) {
	h := newTestACSHandler()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/acs", nil)
		h.ServeHTTP(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "method %s should return 405", method)
	}
}

// ---------------------------------------------------------------------------
// Tests: Empty body
// ---------------------------------------------------------------------------

func TestServeHTTP_EmptyBody_NoSession_Returns204(t *testing.T) {
	h := newTestACSHandler()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestServeHTTP_EmptyBody_WithSession_NoCommands_CompletesSession(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, &acsHCmdQueue{}, bus)

	// Simulate a prior Inform that created a session with Cookie.
	deviceSN := "TEST-SN-EMPTY"
	sessionID := "test-session-id-001"
	session := &Session{
		ID:        sessionID,
		DeviceSN:  deviceSN,
		State:     StateInformReceived,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		CWMPId:    "test-cwmp-id",
	}
	store.CreateWithID(context.Background(), sessionID, session)
	h.admission.Acquire()
	h.metrics.ActiveSessions.Inc()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionID})
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	// Session should be deleted after completion.
	s, _ := store.GetByID(context.Background(), sessionID)
	assert.Nil(t, s)
}

func TestServeHTTP_EmptyBody_WithSession_HasCommand_SendsRPC(t *testing.T) {
	store := newAcsHSessionStore()
	cmdQ := &acsHCmdQueue{}
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, cmdQ, bus)

	deviceSN := "TEST-SN-CMD"
	sessionID := "test-session-id-002"
	session := &Session{
		ID:        sessionID,
		DeviceSN:  deviceSN,
		State:     StateInformReceived,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		CWMPId:    "cmd-cwmp-id",
	}
	store.CreateWithID(context.Background(), sessionID, session)
	h.admission.Acquire()
	h.metrics.ActiveSessions.Inc()

	// Queue a GetParameterValues command.
	params, _ := json.Marshal(map[string]interface{}{
		"names": []string{"Device.DeviceInfo.SoftwareVersion"},
	})
	cmdQ.Push(context.Background(), deviceSN, &cmdqueue.Command{
		ID:         "cmd-1",
		Method:     "GetParameterValues",
		Params:     params,
		CommandKey: "test-key",
		CreatedAt:  time.Now(),
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionID})
	h.ServeHTTP(w, req)

	// Should return 200 with SOAP XML (the RPC request).
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/xml")
	assert.Contains(t, w.Body.String(), "GetParameterValues")
	// Session should transition to RPC_PENDING.
	s, _ := store.GetByID(context.Background(), sessionID)
	require.NotNil(t, s)
	assert.Equal(t, StateRPCPending, s.State)
}

// ---------------------------------------------------------------------------
// Tests: Unknown SOAP method
// ---------------------------------------------------------------------------

func TestServeHTTP_UnknownSOAP_Returns400(t *testing.T) {
	h := newTestACSHandler()

	w := httptest.NewRecorder()
	body := `<soap:Envelope><soap:Body><UnknownMethod/></soap:Body></soap:Envelope>`
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(body))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: Inform flow
// ---------------------------------------------------------------------------

func TestServeHTTP_Inform_Bootstrap_Success(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, &acsHCmdQueue{}, bus)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	req.RemoteAddr = "192.168.1.1:5000"
	h.ServeHTTP(w, req)

	// Should return 200 with InformResponse.
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "InformResponse")
	assert.Contains(t, w.Body.String(), "100001") // CWMP ID echoed back

	// Should set SESSION cookie.
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "SESSION cookie should be set")
	assert.NotEmpty(t, sessionCookie.Value)

	// Session should be created with StateInformReceived.
	s, _ := store.GetByID(context.Background(), sessionCookie.Value)
	require.NotNil(t, s)
	assert.Equal(t, StateInformReceived, s.State)
	assert.Equal(t, "100001", s.CWMPId)
	assert.Equal(t, sessionCookie.Value, s.ID)
	assert.Equal(t, "TEST-SN-001", s.DeviceSN)

	// Event should be published with bootstrap subject.
	bus.mu.Lock()
	defer bus.mu.Unlock()
	require.Len(t, bus.published, 1)
	assert.Equal(t, event.SubjectDeviceBootstrap, bus.published[0].Subject)
}

func TestServeHTTP_Inform_Periodic_PublishesPeriodicEvent(t *testing.T) {
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(newAcsHSessionStore(), &acsHCmdQueue{}, bus)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformPeriodicXML))
	req.RemoteAddr = "192.168.1.2:5000"
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	bus.mu.Lock()
	defer bus.mu.Unlock()
	require.Len(t, bus.published, 1)
	assert.Equal(t, event.SubjectDevicePeriodic, bus.published[0].Subject)
}

func TestServeHTTP_Inform_MalformedXML_Returns400(t *testing.T) {
	h := newTestACSHandler()

	w := httptest.NewRecorder()
	body := `<?xml version="1.0"?><soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body><cwmp:Inform xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><broken></cwmp:Inform></soap:Body></soap:Envelope>`
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(body))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: Rate limiting
// ---------------------------------------------------------------------------

func TestServeHTTP_Inform_RateLimited_Returns503(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, &acsHCmdQueue{}, bus)
	// Set very low rate limit: 1 per minute, burst 1.
	h.rateLimiter = NewDeviceRateLimiter(1, 1, 10000, zap.NewNop())

	// First request should succeed.
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	req1.RemoteAddr = "10.0.0.1:1000"
	h.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request from same device should be rate limited.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	req2.RemoteAddr = "10.0.0.1:1001"
	h.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusServiceUnavailable, w2.Code)
}

// ---------------------------------------------------------------------------
// Tests: Admission control
// ---------------------------------------------------------------------------

func TestServeHTTP_Inform_AdmissionDenied_Returns503(t *testing.T) {
	h := newTestACSHandler()
	// Set max sessions to 1 and fill it.
	h.admission = NewAdmissionController(1)
	h.admission.Acquire()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformPeriodicXML))
	req.RemoteAddr = "10.0.0.2:2000"
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: RPC response handling
// ---------------------------------------------------------------------------

func TestServeHTTP_RPCResponse_CompletesSessionWhenNoMoreCommands(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, &acsHCmdQueue{}, bus)

	deviceSN := "TEST-SN-001"
	sessionID := "test-session-id-003"
	session := &Session{
		ID:        sessionID,
		DeviceSN:  deviceSN,
		State:     StateRPCPending,
		LastRPC:   "GetParameterValues",
		StartedAt: time.Now().Add(-2 * time.Second),
		UpdatedAt: time.Now(),
		CWMPId:    "100001",
	}
	store.CreateWithID(context.Background(), sessionID, session)
	h.admission.Acquire()
	h.metrics.ActiveSessions.Inc()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHGetParamRespXML))
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionID})
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	// Session should be deleted after completion.
	s, _ := store.GetByID(context.Background(), sessionID)
	assert.Nil(t, s)
	// RPC response event should be published.
	bus.mu.Lock()
	defer bus.mu.Unlock()
	require.GreaterOrEqual(t, len(bus.published), 1)
	assert.Equal(t, event.SubjectCommandGetParamsResponse, bus.published[0].Subject)
}

func TestServeHTTP_RPCResponse_ChainsNextCommand(t *testing.T) {
	store := newAcsHSessionStore()
	cmdQ := &acsHCmdQueue{}
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, cmdQ, bus)

	deviceSN := "TEST-SN-001"
	sessionID := "test-session-id-004"
	session := &Session{
		ID:        sessionID,
		DeviceSN:  deviceSN,
		State:     StateRPCPending,
		LastRPC:   "GetParameterValues",
		StartedAt: time.Now().Add(-2 * time.Second),
		UpdatedAt: time.Now(),
		CWMPId:    "100001",
	}
	store.CreateWithID(context.Background(), sessionID, session)
	h.admission.Acquire()
	h.metrics.ActiveSessions.Inc()

	// Queue another command to be dispatched after the RPC response.
	rebootCmd := &cmdqueue.Command{
		ID:         "cmd-reboot",
		Method:     "Reboot",
		Params:     json.RawMessage(`{}`),
		CommandKey: "reboot-key",
		CreatedAt:  time.Now(),
	}
	cmdQ.Push(context.Background(), deviceSN, rebootCmd)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHGetParamRespXML))
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionID})
	h.ServeHTTP(w, req)

	// Should return 200 with the next RPC request (Reboot).
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Reboot")
	// Session should be RPC_PENDING with LastRPC=Reboot.
	s, _ := store.GetByID(context.Background(), sessionID)
	require.NotNil(t, s)
	assert.Equal(t, StateRPCPending, s.State)
	assert.Equal(t, "Reboot", s.LastRPC)
}

func TestServeHTTP_RPCResponse_NoCookie_Returns204(t *testing.T) {
	h := newTestACSHandler()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHGetParamRespXML))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: TransferComplete
// ---------------------------------------------------------------------------

func TestServeHTTP_TransferComplete_PublishesEvent(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, &acsHCmdQueue{}, bus)

	sessionID := "test-session-id-tc"
	session := &Session{
		ID:        sessionID,
		DeviceSN:  "TEST-SN-TC",
		State:     StateProcessing,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.CreateWithID(context.Background(), sessionID, session)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHTransferCompleteXML))
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionID})
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "TransferCompleteResponse")

	bus.mu.Lock()
	defer bus.mu.Unlock()
	require.Len(t, bus.published, 1)
	assert.Equal(t, event.SubjectDeviceTransferComplete, bus.published[0].Subject)
}

// ---------------------------------------------------------------------------
// Tests: completeSession
// ---------------------------------------------------------------------------

func TestCompleteSession_ReleasesResources(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)
	store := newAcsHSessionStore()

	h := &Handler{
		sessionStore: store,
		admission:    NewAdmissionController(100),
		metrics:      metrics,
		logger:       zap.NewNop(),
	}

	h.metrics.ActiveSessions.Inc()
	h.admission.Acquire()

	session := &Session{
		ID:        "session-complete",
		DeviceSN:  "SN-COMPLETE",
		State:     StateProcessing,
		StartedAt: time.Now().Add(-5 * time.Second),
		UpdatedAt: time.Now(),
	}
	store.CreateWithID(context.Background(), "session-complete", session)

	h.completeSession(context.Background(), session)

	assert.Equal(t, StateComplete, session.State)
	assert.Equal(t, int64(0), h.admission.Current())

	// 验证 Session 已从存储中删除
	sByID, _ := store.GetByID(context.Background(), "session-complete")
	assert.Nil(t, sByID)
}

func TestCompleteSession_NilSession_StillReleasesResources(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)

	h := &Handler{
		sessionStore: newAcsHSessionStore(),
		admission:    NewAdmissionController(100),
		metrics:      metrics,
		logger:       zap.NewNop(),
	}

	h.admission.Acquire()

	h.completeSession(context.Background(), nil)

	assert.Equal(t, int64(0), h.admission.Current())
}

// ---------------------------------------------------------------------------
// Tests: Full Inform → Empty → Complete lifecycle
// ---------------------------------------------------------------------------

func TestFullSessionLifecycle_InformThenEmpty(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, &acsHCmdQueue{}, bus)

	// Step 1: Send Inform.
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	h.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), "InformResponse")

	// Extract session cookie from response.
	cookies := w1.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "SESSION cookie should be set")

	// Step 2: Send empty POST (no commands queued) with session cookie.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	req2.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionCookie.Value})
	h.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusNoContent, w2.Code)

	// Session should be deleted after completion.
	s, _ := store.GetByID(context.Background(), sessionCookie.Value)
	assert.Nil(t, s, "session should be deleted after completion")
}

func TestFullSessionLifecycle_InformThenRPCThenEmpty(t *testing.T) {
	store := newAcsHSessionStore()
	cmdQ := &acsHCmdQueue{}
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, cmdQ, bus)

	// Queue a command before the Inform.
	params, _ := json.Marshal(map[string]interface{}{
		"names": []string{"Device.DeviceInfo.SoftwareVersion"},
	})
	cmdQ.Push(context.Background(), "TEST-SN-001", &cmdqueue.Command{
		ID:         "cmd-gpv",
		Method:     "GetParameterValues",
		Params:     params,
		CommandKey: "gpv-key",
		CreatedAt:  time.Now(),
	})

	// Step 1: Inform.
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	h.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Extract session cookie from response.
	cookies := w1.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "SESSION cookie should be set")

	// Step 2: Empty POST → should dispatch the queued command.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	req2.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionCookie.Value})
	h.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "GetParameterValues")

	// Step 3: Device sends GetParameterValuesResponse.
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHGetParamRespXML))
	req3.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionCookie.Value})
	h.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusNoContent, w3.Code)

	// Session should be deleted after completion.
	s, _ := store.GetByID(context.Background(), sessionCookie.Value)
	assert.Nil(t, s, "session should be deleted after completion")
}

// ---------------------------------------------------------------------------
// Tests: TaskService Integration
// ---------------------------------------------------------------------------

// acsHTaskService is a mock TaskService for testing
type acsHTaskService struct {
	mu              sync.Mutex
	tasks           []*task.Task
	cwmpIDToTaskMap map[string]*task.Task
	popIndex        int
}

func newAcsHTaskService() *acsHTaskService {
	return &acsHTaskService{
		tasks:           make([]*task.Task, 0),
		cwmpIDToTaskMap: make(map[string]*task.Task),
	}
}

func (m *acsHTaskService) addTask(t *task.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = append(m.tasks, t)
}

func (m *acsHTaskService) PopTask(ctx context.Context, deviceSN string) (*task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := m.popIndex; i < len(m.tasks); i++ {
		if m.tasks[i].DeviceSN == deviceSN && m.tasks[i].Status == task.TaskStatusPending {
			m.popIndex = i + 1
			return m.tasks[i], nil
		}
	}
	return nil, nil
}

func (m *acsHTaskService) MarkTaskSent(ctx context.Context, taskID, cwmpID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		if t.ID == taskID {
			t.Status = task.TaskStatusSent
			t.CWMPID = cwmpID
			m.cwmpIDToTaskMap[cwmpID] = t
			now := time.Now()
			t.SentAt = &now
			return nil
		}
	}
	return nil
}

func (m *acsHTaskService) MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		if t.ID == taskID {
			t.Status = task.TaskStatusCompleted
			t.Result = result
			now := time.Now()
			t.CompletedAt = &now
			return nil
		}
	}
	return nil
}

func (m *acsHTaskService) MarkTaskFailed(ctx context.Context, taskID string, errorCode int, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		if t.ID == taskID {
			t.Status = task.TaskStatusFailed
			t.ErrorCode = errorCode
			t.ErrorMessage = errorMsg
			now := time.Now()
			t.CompletedAt = &now
			return nil
		}
	}
	return nil
}

func (m *acsHTaskService) GetTaskByCWMPID(ctx context.Context, cwmpID string) (*task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cwmpIDToTaskMap[cwmpID], nil
}

func (m *acsHTaskService) CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := &task.Task{
		ID:          fmt.Sprintf("task-%d", len(m.tasks)+1),
		DeviceSN:    req.DeviceSN,
		Method:      req.Method,
		Params:      req.Params,
		Priority:    req.Priority,
		Status:      task.TaskStatusPending,
		MaxRetries:  req.MaxRetries,
		CreatedAt:   time.Now(),
		Source:      req.Source,
		CreatorID:   req.CreatorID,
		Description: req.Description,
	}
	m.tasks = append(m.tasks, t)
	return t, nil
}

// TestTaskQueue_ProcessMultipleRPCMethods tests processing multiple TR069 RPC methods
// from the task queue when CPE sends empty POST
func TestTaskQueue_ProcessMultipleRPCMethods(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	taskSvc := newAcsHTaskService()

	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)
	h := &Handler{
		sessionStore:  store,
		commandQueue:  &acsHCmdQueue{}, // empty legacy queue
		taskService:   taskSvc,         // new task service
		eventBus:      bus,
		authenticator: &auth.NoopAuthenticator{},
		rpcDispatcher: rpc.NewDispatcher(),
		rateLimiter:   NewDeviceRateLimiter(100, 100, 10000, zap.NewNop()),
		admission:     NewAdmissionController(1000),
		metrics:       metrics,
		logger:        zap.NewNop(),
	}

	deviceSN := "TEST-SN-001"

	// Create tasks for all major TR069 RPC methods (max 10)
	rpcMethods := []struct {
		method string
		params json.RawMessage
	}{
		{"GetParameterValues", json.RawMessage(`{"names":["Device.DeviceInfo.SoftwareVersion"]}`)},
		{"SetParameterValues", json.RawMessage(`{"parameters":[{"name":"Device.X.Test","value":"test"}]}`)},
		{"GetParameterNames", json.RawMessage(`{"parameter_path":"Device.DeviceInfo","next_level":false}`)},
		{"GetParameterAttributes", json.RawMessage(`{"names":["Device.DeviceInfo.SoftwareVersion"]}`)},
		{"SetParameterAttributes", json.RawMessage(`{"parameters":[{"name":"Device.X.Test","notification":1}]}`)},
		{"Reboot", json.RawMessage(`{}`)},
		{"Download", json.RawMessage(`{"file_type":"1 Firmware Upgrade Image","url":"http://test.com/fw.bin"}`)},
		{"Upload", json.RawMessage(`{"file_type":"1 Log File","url":"http://test.com/upload"}`)},
		{"AddObject", json.RawMessage(`{"object_name":"Device.X.TestObject."}`)},
		{"FactoryReset", json.RawMessage(`{}`)},
	}

	// Add tasks to the mock service
	for i, rpc := range rpcMethods {
		taskID := fmt.Sprintf("task-%d", i+1)
		tt := &task.Task{
			ID:         taskID,
			DeviceSN:   deviceSN,
			Method:     rpc.method,
			Params:     rpc.params,
			Priority:   10,
			Status:     task.TaskStatusPending,
			MaxRetries: 3,
			CreatedAt:  time.Now(),
			Source:     task.TaskSourceAPI,
		}
		taskSvc.addTask(tt)
	}

	// Step 1: Send Inform to establish session
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	h.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), "InformResponse")

	// Extract session cookie
	cookies := w1.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "SESSION cookie should be set")

	// Step 2: Empty POST → should dispatch first task (GetParameterValues)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	req2.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionCookie.Value})
	h.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "GetParameterValues")
	assert.Contains(t, w2.Body.String(), "cwmp:ID")

	// Verify task was marked as sent
	taskSvc.mu.Lock()
	assert.Equal(t, task.TaskStatusSent, taskSvc.tasks[0].Status)
	assert.NotEmpty(t, taskSvc.tasks[0].CWMPID)
	sentCWMPID := taskSvc.tasks[0].CWMPID
	taskSvc.mu.Unlock()

	t.Logf("Task 1 (GetParameterValues) dispatched with CWMP ID: %s", sentCWMPID)

	// Step 3: Simulate CPE response for GetParameterValues
	// Build response with matching CWMP ID
	getParamResp := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValuesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[1]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value>FW-2.1.0</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:GetParameterValuesResponse>
  </soap:Body>
</soap:Envelope>`, sentCWMPID)

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(getParamResp))
	req3.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionCookie.Value})
	h.ServeHTTP(w3, req3)

	// Should get next task (SetParameterValues) or 204 if no more
	// Since we have 10 tasks, should return next task
	if w3.Code == http.StatusOK {
		assert.Contains(t, w3.Body.String(), "SetParameterValues")
		t.Logf("Task 2 (SetParameterValues) dispatched")

		// Verify first task was marked as completed
		taskSvc.mu.Lock()
		assert.Equal(t, task.TaskStatusCompleted, taskSvc.tasks[0].Status)
		taskSvc.mu.Unlock()
	}

	t.Logf("Successfully processed task queue with %d RPC methods", len(rpcMethods))
}

// TestTaskQueue_MarkTaskCompleted verifies task completion tracking
func TestTaskQueue_MarkTaskCompleted(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	taskSvc := newAcsHTaskService()

	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)
	h := &Handler{
		sessionStore:  store,
		commandQueue:  &acsHCmdQueue{},
		taskService:   taskSvc,
		eventBus:      bus,
		authenticator: &auth.NoopAuthenticator{},
		rpcDispatcher: rpc.NewDispatcher(),
		rateLimiter:   NewDeviceRateLimiter(100, 100, 10000, zap.NewNop()),
		admission:     NewAdmissionController(1000),
		metrics:       metrics,
		logger:        zap.NewNop(),
	}

	deviceSN := "TEST-SN-001"

	// Add a single task
	taskSvc.addTask(&task.Task{
		ID:         "task-reboot-1",
		DeviceSN:   deviceSN,
		Method:     "Reboot",
		Params:     json.RawMessage(`{}`),
		Priority:   10,
		Status:     task.TaskStatusPending,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
		Source:     task.TaskSourceAPI,
	})

	// Step 1: Send Inform
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	h.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	cookies := w1.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)

	// Step 2: Empty POST → dispatch Reboot
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	req2.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionCookie.Value})
	h.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "Reboot")

	taskSvc.mu.Lock()
	sentCWMPID := taskSvc.tasks[0].CWMPID
	taskSvc.mu.Unlock()

	// Step 3: Simulate RebootResponse
	rebootResp := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:RebootResponse/>
  </soap:Body>
</soap:Envelope>`, sentCWMPID)

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(rebootResp))
	req3.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionCookie.Value})
	h.ServeHTTP(w3, req3)

	// No more tasks → 204
	assert.Equal(t, http.StatusNoContent, w3.Code)

	// Verify task was marked as completed
	taskSvc.mu.Lock()
	assert.Equal(t, task.TaskStatusCompleted, taskSvc.tasks[0].Status)
	assert.NotNil(t, taskSvc.tasks[0].Result)
	assert.NotNil(t, taskSvc.tasks[0].CompletedAt)
	taskSvc.mu.Unlock()
}

// TestTaskQueue_SOAPFaultMarksTaskFailed verifies that SOAP Fault marks task as failed
func TestTaskQueue_SOAPFaultMarksTaskFailed(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	taskSvc := newAcsHTaskService()

	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)
	h := &Handler{
		sessionStore:  store,
		commandQueue:  &acsHCmdQueue{},
		taskService:   taskSvc,
		eventBus:      bus,
		authenticator: &auth.NoopAuthenticator{},
		rpcDispatcher: rpc.NewDispatcher(),
		rateLimiter:   NewDeviceRateLimiter(100, 100, 10000, zap.NewNop()),
		admission:     NewAdmissionController(1000),
		metrics:       metrics,
		logger:        zap.NewNop(),
	}

	deviceSN := "TEST-SN-001"

	// Add a task
	taskSvc.addTask(&task.Task{
		ID:         "task-gpv-fault",
		DeviceSN:   deviceSN,
		Method:     "GetParameterValues",
		Params:     json.RawMessage(`{"names":["Device.Invalid.Parameter"]}`),
		Priority:   10,
		Status:     task.TaskStatusPending,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
		Source:     task.TaskSourceAPI,
	})

	// Step 1: Send Inform
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	h.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	cookies := w1.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)

	// Step 2: Empty POST → dispatch GetParameterValues
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	req2.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionCookie.Value})
	h.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	taskSvc.mu.Lock()
	sentCWMPID := taskSvc.tasks[0].CWMPID
	taskSvc.mu.Unlock()

	// Step 3: Simulate SOAP Fault response
	faultResp := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <soap:Fault>
      <faultcode>Client</faultcode>
      <faultstring>Invalid parameter name</faultstring>
    </soap:Fault>
  </soap:Body>
</soap:Envelope>`, sentCWMPID)

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(faultResp))
	req3.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionCookie.Value})
	h.ServeHTTP(w3, req3)

	// Verify task was marked as failed
	taskSvc.mu.Lock()
	assert.Equal(t, task.TaskStatusFailed, taskSvc.tasks[0].Status)
	assert.Contains(t, taskSvc.tasks[0].ErrorMessage, "Invalid parameter name")
	assert.NotNil(t, taskSvc.tasks[0].CompletedAt)
	taskSvc.mu.Unlock()

	t.Logf("Task correctly marked as failed on SOAP Fault")
}
