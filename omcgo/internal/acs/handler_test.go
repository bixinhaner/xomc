package acs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPostSessionWakeFailureLogLevel_NoConnectionMethodIsDebug(t *testing.T) {
	assert.Equal(t, zap.DebugLevel, postSessionWakeFailureLogLevel(connreq.ErrNoConnectionMethod))
	assert.Equal(t, zap.WarnLevel, postSessionWakeFailureLogLevel(errors.New("connection refused")))
}

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

type acsHDeviceSessionStore struct {
	current map[string]string
	swapErr error
}

func (s *acsHDeviceSessionStore) Swap(_ context.Context, deviceSN, newSessionID string) (string, error) {
	if s.swapErr != nil {
		return "", s.swapErr
	}
	if s.current == nil {
		s.current = make(map[string]string)
	}
	old := s.current[deviceSN]
	s.current[deviceSN] = newSessionID
	if old == "" || old == newSessionID {
		return "", nil
	}
	return old, nil
}

func (s *acsHDeviceSessionStore) CompareAndDelete(_ context.Context, deviceSN, sessionID string) error {
	if s.current != nil && s.current[deviceSN] == sessionID {
		delete(s.current, deviceSN)
	}
	return nil
}

func (s *acsHDeviceSessionStore) Get(_ context.Context, deviceSN string) (string, error) {
	if s.current == nil {
		return "", nil
	}
	return s.current[deviceSN], nil
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
func (m *acsHEventBus) PullSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return &acsHSubscription{}, nil
}
func (m *acsHEventBus) Close() error { return nil }

type acsHSubscription struct{}

func (s *acsHSubscription) Unsubscribe() error { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestACSHandler() *Handler {
	return newTestACSHandlerWithDeps(newAcsHSessionStore(), &acsHEventBus{})
}

func newTestACSHandlerWithDeps(store SessionStore, bus event.EventBus) *Handler {
	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)
	return &Handler{
		sessionStore:  store,
		taskService:   newAcsHTaskService(),
		eventBus:      bus,
		authenticator: &auth.NoopAuthenticator{},
		rpcDispatcher: rpc.NewDispatcher(),
		rateLimiter:   NewDeviceRateLimiter(100, 100, 10000, zap.NewNop()),
		admission:     NewAdmissionController(1000),
		metrics:       metrics,
		logger:        zap.NewNop(),
	}
}

func TestGetSessionFromCookieRejectsSupersededDeviceSession(t *testing.T) {
	store := newAcsHSessionStore()
	sessionID := "session-old"
	session := &Session{
		ID:        sessionID,
		DeviceSN:  "SN-STALE-SESSION",
		State:     StateRPCPending,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateWithID(context.Background(), sessionID, session))

	h := newTestACSHandlerWithDeps(store, &acsHEventBus{})
	h.deviceSessionStore = &acsHDeviceSessionStore{current: map[string]string{
		"SN-STALE-SESSION": "session-new",
	}}

	req := httptest.NewRequest(http.MethodPost, "/acs", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionID})

	got, gotID := h.getSessionFromCookie(req, zap.NewNop())

	assert.Nil(t, got)
	assert.Equal(t, sessionID, gotID)
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

// UPS VALUE CHANGE Inform carrying InternetGatewayDevice.FaultMgmt.CurrentAlarm.* parameters.
const acsHInformUPSValueChangeCurrentAlarmXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">UPS-ALM-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>Baicells</Manufacturer>
        <OUI>001122</OUI>
        <ProductClass>UPS_M3_BMU</ProductClass>
        <SerialNumber>UPS-SN-ALARM-001</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[1]">
        <EventStruct>
          <EventCode>4 VALUE CHANGE</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2026-08-20T10:00:00Z</CurrentTime>
      <RetryCount>0</RetryCount>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[8]">
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.DeviceInfo.ProductClass</Name>
          <Value xsi:type="xsd:string">UPS_M3_BMU</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.AlarmIdentifier</Name>
          <Value xsi:type="xsd:string">42000</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.PerceivedSeverity</Name>
          <Value xsi:type="xsd:string">Major</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.EventType</Name>
          <Value xsi:type="xsd:string">30003</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.ProbableCause</Name>
          <Value xsi:type="xsd:string">AC Power Off</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.SpecificProblem</Name>
          <Value xsi:type="xsd:string">AC Power Off</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.AlarmRaisedTime</Name>
          <Value xsi:type="xsd:dateTime">2026-08-20T10:00:00Z</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.ManagedObjectInstance</Name>
          <Value xsi:type="xsd:string">InternetGatewayDevice.FaultMgmt.CurrentAlarm.</Value>
        </ParameterValueStruct>
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
// Tests: Request body size limit (#1 — 防止超大 POST 导致 ACS OOM)
// ---------------------------------------------------------------------------

// 失败路径：请求体超过 maxRequestBodySize 上限时，应在读取阶段被 MaxBytesReader
// 拦截并返回 413，而不是把整个 body 读入内存。
func TestServeHTTP_BodyExceedsLimit_Returns413(t *testing.T) {
	h := newTestACSHandler()
	h.maxRequestBodySize = 1024 // 1KB，便于用小 body 触发上限

	oversized := strings.Repeat("A", 4096) // 4KB > 1KB 上限
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(oversized))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code,
		"body exceeding max_request_body_size must be rejected with 413")
}

// 成功路径：请求体在上限内时，不应因大小被拒（内容非法 SOAP 会走到 400，
// 但绝不能是 413）。
func TestServeHTTP_BodyWithinLimit_NotRejectedAsTooLarge(t *testing.T) {
	h := newTestACSHandler()
	h.maxRequestBodySize = 1 << 20 // 1MB

	body := strings.Repeat("A", 4096) // 4KB < 1MB 上限
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(body))
	h.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusRequestEntityTooLarge, w.Code,
		"body within limit must not be rejected as too large")
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
	h := newTestACSHandlerWithDeps(store, bus)

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
	h.admission.Acquire(context.Background(), sessionID)
	h.trackActiveSession(sessionID)

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
	taskSvc := newAcsHTaskService()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, bus)
	h.taskService = taskSvc

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
	h.admission.Acquire(context.Background(), sessionID)
	h.trackActiveSession(sessionID)

	// Queue a GetParameterValues command.
	params, _ := json.Marshal(map[string]interface{}{
		"names": []string{"Device.DeviceInfo.SoftwareVersion"},
	})
	taskSvc.addTask(&task.Task{
		ID:         "cmd-1",
		DeviceSN:   deviceSN,
		Method:     "GetParameterValues",
		Params:     params,
		Status:     task.TaskStatusPending,
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
	h := newTestACSHandlerWithDeps(store, bus)
	taskSvc := h.taskService.(*acsHTaskService)

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
	assert.Equal(t, []string{"TEST-SN-001"}, taskSvc.recoveredDevices)

	// Event should be published with bootstrap subject.
	bus.mu.Lock()
	defer bus.mu.Unlock()
	require.Len(t, bus.published, 1)
	assert.Equal(t, event.SubjectDeviceBootstrap, bus.published[0].Subject)
}

func TestServeHTTP_Inform_DeviceSessionSwapFailureReturns503AndReleasesAdmission(t *testing.T) {
	h := newTestACSHandler()
	h.deviceSessionStore = &acsHDeviceSessionStore{swapErr: errors.New("redis unavailable")}
	taskSvc := h.taskService.(*acsHTaskService)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformBootstrapXML))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, int64(0), h.admission.Current(context.Background()))
	assert.Empty(t, w.Result().Cookies())
	assert.Empty(t, taskSvc.recoveredDevices, "failed session swap must not mutate sent tasks")
}

func TestServeHTTP_Inform_Periodic_PublishesPeriodicEvent(t *testing.T) {
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(newAcsHSessionStore(), bus)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformPeriodicXML))
	req.RemoteAddr = "192.168.1.2:5000"
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	bus.mu.Lock()
	defer bus.mu.Unlock()
	require.Len(t, bus.published, 1)
	assert.Equal(t, event.SubjectDevicePeriodic, bus.published[0].Subject)
	var payload struct {
		RemoteIP      string `json:"remote_ip"`
		Authenticated bool   `json:"authenticated"`
		AuthMethod    string `json:"auth_method"`
	}
	require.NoError(t, bus.published[0].Event.DecodePayload(&payload))
	assert.Equal(t, "192.168.1.2", payload.RemoteIP)
	assert.False(t, payload.Authenticated)
	assert.Equal(t, "none", payload.AuthMethod)
}

func TestServeHTTP_Inform_AuthenticationFailureReturnsChallenge(t *testing.T) {
	h := newTestACSHandler()
	h.authenticator = &auth.BasicAuthenticator{Username: "cpe", Password: "secret"}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformPeriodicXML))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, `Basic realm="ACS"`, w.Header().Get("WWW-Authenticate"))
	assert.Empty(t, w.Result().Cookies())
}

func TestServeHTTP_Inform_PublishesAuthenticatedCredentialIdentity(t *testing.T) {
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(newAcsHSessionStore(), bus)
	h.authenticator = &auth.BasicAuthenticator{Username: "cpe", Password: "secret"}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformPeriodicXML))
	req.SetBasicAuth("cpe", "secret")
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, bus.published, 1)
	var payload struct {
		Authenticated bool   `json:"authenticated"`
		AuthMethod    string `json:"auth_method"`
		CredentialID  string `json:"credential_id"`
	}
	require.NoError(t, bus.published[0].Event.DecodePayload(&payload))
	assert.True(t, payload.Authenticated)
	assert.Equal(t, "basic", payload.AuthMethod)
	assert.Equal(t, "cpe", payload.CredentialID)
}

func TestTransportPeerIPDoesNotTrustForwardedHeaders(t *testing.T) {
	assert.Equal(t, "192.0.2.10", transportPeerIP("192.0.2.10:7547"))
	assert.Equal(t, "2001:db8::10", transportPeerIP("[2001:db8::10]:7547"))
	assert.Empty(t, transportPeerIP("not-an-address"))
}

func TestRequestClientIPTrustsOnlyConfiguredProxy(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("173.18.0.0/16")}

	proxied := httptest.NewRequest(http.MethodPost, "/acs", nil)
	proxied.RemoteAddr = "173.18.0.10:43120"
	proxied.Header.Set("X-Forwarded-For", "172.24.224.35")
	assert.Equal(t, "172.24.224.35", requestClientIP(proxied, trusted))

	direct := httptest.NewRequest(http.MethodPost, "/acs", nil)
	direct.RemoteAddr = "172.24.224.35:43120"
	direct.Header.Set("X-Forwarded-For", "203.0.113.99")
	assert.Equal(t, "172.24.224.35", requestClientIP(direct, trusted))
}

func TestRequestClientIPWalksForwardedChainFromNearestHop(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("173.18.0.0/16")}
	req := httptest.NewRequest(http.MethodPost, "/acs", nil)
	req.RemoteAddr = "173.18.0.10:43120"
	req.Header.Set("X-Forwarded-For", "203.0.113.99, 172.24.224.35")

	assert.Equal(t, "172.24.224.35", requestClientIP(req, trusted))
}

func TestServeHTTP_Inform_Periodic_EnqueuesUECountQuery(t *testing.T) {
	h := newTestACSHandler()
	taskSvc := h.taskService.(*acsHTaskService)
	h.ueCountPolicy = NewUECountPolicy(
		&stubUECountPathResolver{paths: []string{
			"Device.DeviceInfo.UE_Count",
			"Device.DeviceInfo.2.UE_Count",
		}},
		taskSvc,
		stubUECountProbeGate{acquired: true},
		zap.NewNop(),
	)
	runCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go h.ueCountPolicy.Run(runCtx)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformPeriodicXML))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Eventually(t, func() bool {
		taskSvc.mu.Lock()
		defer taskSvc.mu.Unlock()
		return len(taskSvc.tasks) == 1
	}, time.Second, time.Millisecond)
	taskSvc.mu.Lock()
	defer taskSvc.mu.Unlock()
	assert.Equal(t, ueCountGPVDescription, taskSvc.tasks[0].Description)
}

func TestServeHTTP_Inform_UPSBootstrap_SkipsInformPeriodPolicy(t *testing.T) {
	h := newTestACSHandler()
	taskSvc := h.taskService.(*acsHTaskService)
	h.informPeriodPolicy = NewInformPeriodPolicy(
		mockInformPeriodLookup(map[string]string{
			"device:enbInformPeriodAdjustEnable": "true",
			"device:enbInformPeriod":             "60",
		}),
		taskSvc,
		zap.NewNop(),
	)
	body := strings.ReplaceAll(acsHInformBootstrapXML,
		"<ProductClass>SmallCell-LTE</ProductClass>",
		"<ProductClass>UPS_M3_BMU</ProductClass>",
	)
	body = strings.ReplaceAll(body,
		"<SerialNumber>TEST-SN-001</SerialNumber>",
		"<SerialNumber>UPS-SN-BOOTSTRAP-001</SerialNumber>",
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(body))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	taskSvc.mu.Lock()
	defer taskSvc.mu.Unlock()
	assert.Empty(t, taskSvc.tasks)
}

func TestServeHTTP_Inform_UPSPeriodic_SkipsUECountPolicy(t *testing.T) {
	h := newTestACSHandler()
	h.ueCountPolicy = NewUECountPolicy(
		&stubUECountPathResolver{paths: []string{"Device.DeviceInfo.UE_Count"}},
		&stubUECountTaskService{},
		stubUECountProbeGate{acquired: true},
		zap.NewNop(),
	)
	body := strings.ReplaceAll(acsHInformPeriodicXML,
		"<ProductClass>SmallCell-LTE</ProductClass>",
		"<ProductClass>UPS_M3_BMU</ProductClass>",
	)
	body = strings.ReplaceAll(body,
		"<SerialNumber>TEST-SN-002</SerialNumber>",
		"<SerialNumber>UPS-SN-PERIODIC-001</SerialNumber>",
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(body))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, h.ueCountPolicy.queue)
	h.ueCountPolicy.pendingMu.Lock()
	defer h.ueCountPolicy.pendingMu.Unlock()
	assert.Empty(t, h.ueCountPolicy.pending)
}

func TestHasExpeditedEventParams_InternetGatewayDevicePrefix(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		{Name: "InternetGatewayDevice.DeviceInfo.Manufacturer", Value: "Baicells"},
		{Name: "InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.NotificationType", Value: "NewAlarm"},
	}

	assert.True(t, hasExpeditedEventParams(params))
}

func TestFilterExpeditedEventParams_InternetGatewayDevicePrefix(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		{Name: "InternetGatewayDevice.DeviceInfo.Manufacturer", Value: "Baicells"},
		{Name: "InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.NotificationType", Value: "NewAlarm"},
		{Name: "InternetGatewayDevice.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", Value: "50003"},
	}

	filtered := filterExpeditedEventParams(params)
	assert.Len(t, filtered, 2)
	for _, p := range filtered {
		assert.Contains(t, p.Name, "FaultMgmt.ExpeditedEvent.")
	}
}

func TestHasCurrentAlarmParams_InternetGatewayDevicePrefix(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		{Name: "InternetGatewayDevice.DeviceInfo.Manufacturer", Value: "Baicells"},
		{Name: "InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", Value: "42000"},
	}

	assert.True(t, hasCurrentAlarmParams(params))
}

func TestServeHTTP_Inform_UPSValueChangeCurrentAlarmPublishesAlarmEvent(t *testing.T) {
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(newAcsHSessionStore(), bus)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformUPSValueChangeCurrentAlarmXML))
	req.RemoteAddr = "192.168.2.10:5000"
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "InformResponse")

	bus.mu.Lock()
	published := append([]acsHPublishedEvent(nil), bus.published...)
	bus.mu.Unlock()

	require.Len(t, published, 2)
	assert.Equal(t, event.SubjectDeviceValueChange, published[0].Subject)
	assert.Equal(t, event.SubjectDeviceAlarm, published[1].Subject)

	var alarmPayload struct {
		DeviceID      tr069.DeviceId               `json:"device_id"`
		Events        []string                     `json:"events"`
		ParameterList []tr069.ParameterValueStruct `json:"parameter_list"`
	}
	require.NoError(t, published[1].Event.DecodePayload(&alarmPayload))
	assert.Equal(t, "UPS-SN-ALARM-001", alarmPayload.DeviceID.SerialNumber)
	assert.Equal(t, "UPS_M3_BMU", alarmPayload.DeviceID.ProductClass)
	assert.Equal(t, []string{tr069.EventValueChange}, alarmPayload.Events)
	assert.True(t, hasCurrentAlarmParams(alarmPayload.ParameterList))

	for _, publishedEvent := range published {
		assert.NotEqual(t, event.SubjectDeviceExpeditedAlarm, publishedEvent.Subject)
	}
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
	h := newTestACSHandlerWithDeps(store, bus)
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
	taskSvc := h.taskService.(*acsHTaskService)
	// Set max sessions to 1 and fill it.
	h.admission = NewAdmissionController(1)
	h.admission.Acquire(context.Background(), "preexisting-session")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHInformPeriodicXML))
	req.RemoteAddr = "10.0.0.2:2000"
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, float64(1), testutil.ToFloat64(h.metrics.AdmissionRejected))
	assert.Empty(t, taskSvc.recoveredDevices, "rejected Inform must not mutate sent tasks")
}

// ---------------------------------------------------------------------------
// Tests: RPC response handling
// ---------------------------------------------------------------------------

func TestServeHTTP_RPCResponse_CompletesSessionWhenNoMoreCommands(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, bus)

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
	h.admission.Acquire(context.Background(), sessionID)
	h.trackActiveSession(sessionID)

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

func TestServeHTTP_ParamSyncRPCResponseDoesNotPublishDuplicateCanonicalResult(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	taskSvc := newAcsHTaskService()
	h := newTestACSHandlerWithDeps(store, bus)
	h.taskService = taskSvc

	taskItem := &task.Task{
		ID: "param-sync-task-001", DeviceSN: "TEST-SN-001",
		Method: "GetParameterValues", Status: task.TaskStatusSent,
		Source: task.TaskSourceParamSync, SourceID: uuid.NewString(), CreatorID: uuid.NewString(),
		CreatedAt: time.Now(),
	}
	taskSvc.addTask(taskItem)
	sessionID := "param-sync-session-001"
	require.NoError(t, store.CreateWithID(context.Background(), sessionID, &Session{
		ID: sessionID, DeviceSN: taskItem.DeviceSN, State: StateRPCPending,
		LastRPC: taskItem.Method, LastTaskID: taskItem.ID, LastTaskCWMPID: "100001",
		StartedAt: time.Now().Add(-time.Second), UpdatedAt: time.Now(), CWMPId: "100001",
	}))
	h.admission.Acquire(context.Background(), sessionID)
	h.trackActiveSession(sessionID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(acsHGetParamRespXML))
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionID})
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, task.TaskStatusCompleted, taskItem.Status)
	for _, published := range bus.published {
		assert.NotEqual(t, event.SubjectParamSyncTaskResult, published.Subject,
			"TaskTerminalBridge owns the single canonical parameter-sync result")
	}
}

func TestServeHTTP_RPCResponse_ChainsNextCommand(t *testing.T) {
	store := newAcsHSessionStore()
	taskSvc := newAcsHTaskService()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, bus)
	h.taskService = taskSvc

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
	h.admission.Acquire(context.Background(), sessionID)
	h.trackActiveSession(sessionID)
	// Queue another task to be dispatched after the RPC response.
	taskSvc.addTask(&task.Task{
		ID:         "cmd-reboot",
		DeviceSN:   deviceSN,
		Method:     "Reboot",
		Params:     json.RawMessage(`{}`),
		Status:     task.TaskStatusPending,
		CommandKey: "reboot-key",
		CreatedAt:  time.Now(),
	})

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

func TestServeHTTP_PasswordResetResponse_NoCookie_Returns204(t *testing.T) {
	h := newTestACSHandler()

	body := `<?xml version="1.0" encoding="UTF-8"?>
<soap-env:Envelope xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap-env:Header><cwmp:ID soap-env:mustUnderstand="1">reset-id</cwmp:ID></soap-env:Header>
  <soap-env:Body>
    <cwmp:X_BAICELLS_COM_PasswordResetResponse>
      <Status>1</Status>
    </cwmp:X_BAICELLS_COM_PasswordResetResponse>
  </soap-env:Body>
</soap-env:Envelope>`
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRPCResponseMatchesPasswordResetTask(t *testing.T) {
	assert.True(t, rpcResponseMatchesTask(
		soap.MethodBaicellsPasswordResetResp,
		"X_BAICELLS_COM_PasswordReset",
	))
	assert.True(t, rpcResponseMatchesTask(
		soap.MethodCommonPasswordResetResp,
		"X_COMMON_COM_PasswordReset",
	))
	assert.False(t, rpcResponseMatchesTask(
		soap.MethodCommonPasswordResetResp,
		"X_BAICELLS_COM_PasswordReset",
	))
}

// ---------------------------------------------------------------------------
// Tests: TransferComplete
// ---------------------------------------------------------------------------

func TestServeHTTP_TransferComplete_PublishesEvent(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, bus)

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

	h.trackActiveSession("session-complete")
	h.admission.Acquire(context.Background(), "session-complete")

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
	assert.Equal(t, int64(0), h.admission.Current(context.Background()))

	// 验证 Session 已从存储中删除
	sByID, _ := store.GetByID(context.Background(), "session-complete")
	assert.Nil(t, sByID)
}

// issue #65（Option B）契约变更：completeSession(nil) 不再释放准入槽位 —— 没有
// sessionID 无法配对释放，残留槽位由准入 sorted set 的 TTL 过期分自愈回收。这里验证
// nil session 既不能释放 admission，也不能递减无法配对的本地跟踪数。
func TestCompleteSession_NilSession_DoesNotReleaseAdmissionSlot(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)

	h := &Handler{
		sessionStore: newAcsHSessionStore(),
		admission:    NewAdmissionController(100),
		metrics:      metrics,
		logger:       zap.NewNop(),
	}

	h.admission.Acquire(context.Background(), "orphan-no-session")

	h.completeSession(context.Background(), nil)

	// 槽位不被 nil-session 释放（由 TTL 回收）。
	assert.Equal(t, int64(1), h.admission.Current(context.Background()))
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.LocalTrackedSessions))
}

func TestCompleteSessionOnlyDecrementsSessionsTrackedByThisProcess(t *testing.T) {
	metrics := NewACSMetrics(prometheus.NewRegistry())
	h := &Handler{
		sessionStore: newAcsHSessionStore(),
		admission:    NewAdmissionController(100),
		metrics:      metrics,
		logger:       zap.NewNop(),
	}
	metrics.ActiveSessions.Set(41)
	metrics.GlobalActiveSessions.Set(42)

	h.trackActiveSession("local-session")
	assert.Equal(t, float64(1), testutil.ToFloat64(metrics.LocalTrackedSessions))
	assert.Equal(t, float64(41), testutil.ToFloat64(metrics.ActiveSessions))
	assert.Equal(t, float64(42), testutil.ToFloat64(metrics.GlobalActiveSessions))

	// 进程重启前遗留的共享会话不在本地集合中，不得把新进程 gauge 减成负数。
	h.completeSession(context.Background(), &Session{ID: "old-process-session", StartedAt: time.Now()})
	assert.Equal(t, float64(1), testutil.ToFloat64(metrics.LocalTrackedSessions))

	h.completeSession(context.Background(), &Session{ID: "local-session", StartedAt: time.Now()})
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.LocalTrackedSessions))

	// 重复完成也只能递减一次。
	h.completeSession(context.Background(), &Session{ID: "local-session", StartedAt: time.Now()})
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.LocalTrackedSessions))
	assert.Equal(t, float64(41), testutil.ToFloat64(metrics.ActiveSessions))
	assert.Equal(t, float64(42), testutil.ToFloat64(metrics.GlobalActiveSessions))
}

func TestReapLocalActiveSessionsRemovesCrossInstanceOrphans(t *testing.T) {
	metrics := NewACSMetrics(prometheus.NewRegistry())
	h := &Handler{metrics: metrics, logger: zap.NewNop()}
	now := time.Now()
	metrics.ActiveSessions.Set(41)
	metrics.GlobalActiveSessions.Set(42)

	h.trackActiveSessionAt("expired-on-another-instance", now.Add(-6*time.Minute))
	h.trackActiveSessionAt("still-active", now.Add(-time.Minute))
	require.Equal(t, float64(2), testutil.ToFloat64(metrics.LocalTrackedSessions))

	h.reapLocalActiveSessions(now, 5*time.Minute)

	assert.Equal(t, float64(1), testutil.ToFloat64(metrics.LocalTrackedSessions))
	assert.Equal(t, float64(41), testutil.ToFloat64(metrics.ActiveSessions))
	assert.Equal(t, float64(42), testutil.ToFloat64(metrics.GlobalActiveSessions))
	_, expiredStillTracked := h.localActiveSessions.Load("expired-on-another-instance")
	_, activeStillTracked := h.localActiveSessions.Load("still-active")
	assert.False(t, expiredStillTracked)
	assert.True(t, activeStillTracked)
}

func TestRefreshGlobalActiveSessionsUsesAdmissionSourceOfTruth(t *testing.T) {
	metrics := NewACSMetrics(prometheus.NewRegistry())
	admission := NewAdmissionController(100)
	h := &Handler{metrics: metrics, admission: admission, logger: zap.NewNop()}
	require.True(t, admission.Acquire(context.Background(), "one"))
	require.True(t, admission.Acquire(context.Background(), "two"))
	metrics.ActiveSessions.Set(41)
	metrics.GlobalActiveSessions.Set(42)
	metrics.LocalTrackedSessions.Set(43)

	h.refreshGlobalActiveSessions(context.Background())

	assert.Equal(t, float64(2), testutil.ToFloat64(metrics.ActiveSessions))
	assert.Equal(t, float64(2), testutil.ToFloat64(metrics.GlobalActiveSessions))
	assert.Equal(t, float64(43), testutil.ToFloat64(metrics.LocalTrackedSessions))
}

type countingAdmissionController struct {
	currentCalls int
}

func (c *countingAdmissionController) Acquire(context.Context, string) bool { return true }
func (c *countingAdmissionController) Release(context.Context, string)      {}
func (c *countingAdmissionController) Current(context.Context) int64 {
	c.currentCalls++
	return int64(c.currentCalls)
}

func TestRefreshGlobalActiveSessionsSamplesAdmissionOnce(t *testing.T) {
	metrics := NewACSMetrics(prometheus.NewRegistry())
	admission := &countingAdmissionController{}
	h := &Handler{metrics: metrics, admission: admission, logger: zap.NewNop()}

	h.refreshGlobalActiveSessions(context.Background())

	assert.Equal(t, 1, admission.currentCalls)
	assert.Equal(t, float64(1), testutil.ToFloat64(metrics.ActiveSessions))
	assert.Equal(t, float64(1), testutil.ToFloat64(metrics.GlobalActiveSessions))
}

// ---------------------------------------------------------------------------
// Tests: Full Inform → Empty → Complete lifecycle
// ---------------------------------------------------------------------------

func TestFullSessionLifecycle_InformThenEmpty(t *testing.T) {
	store := newAcsHSessionStore()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, bus)

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
	taskSvc := newAcsHTaskService()
	bus := &acsHEventBus{}
	h := newTestACSHandlerWithDeps(store, bus)
	h.taskService = taskSvc

	// Queue a task before the Inform.
	params, _ := json.Marshal(map[string]interface{}{
		"names": []string{"Device.DeviceInfo.SoftwareVersion"},
	})
	taskSvc.addTask(&task.Task{
		ID:         "cmd-gpv",
		DeviceSN:   "TEST-SN-001",
		Method:     "GetParameterValues",
		Params:     params,
		Status:     task.TaskStatusPending,
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
	mu               sync.Mutex
	tasks            []*task.Task
	cwmpIDToTaskMap  map[string]*task.Task
	markSentErrors   map[string]error
	popIndex         int
	recoveredDevices []string
}

func newAcsHTaskService() *acsHTaskService {
	return &acsHTaskService{
		tasks:           make([]*task.Task, 0),
		cwmpIDToTaskMap: make(map[string]*task.Task),
		markSentErrors:  make(map[string]error),
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
	if err := m.markSentErrors[taskID]; err != nil {
		return err
	}
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

func TestPopAndMarkNextTaskSkipsStaleTask(t *testing.T) {
	taskSvc := newAcsHTaskService()
	stale := &task.Task{ID: "stale", DeviceSN: "SN-STALE", Method: "GetParameterValues", Status: task.TaskStatusPending}
	valid := &task.Task{ID: "valid", DeviceSN: "SN-STALE", Method: "GetParameterValues", Status: task.TaskStatusPending}
	taskSvc.addTask(stale)
	taskSvc.addTask(valid)
	taskSvc.markSentErrors[stale.ID] = fmt.Errorf("stale fence: %w", task.ErrTaskNotPending)

	h := &Handler{taskService: taskSvc}
	got, cwmpID, err := h.popAndMarkNextTask(context.Background(), "SN-STALE", zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, valid.ID, got.ID)
	require.NotEmpty(t, cwmpID)
	require.Equal(t, task.TaskStatusSent, valid.Status)
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
	return m.MarkTaskFailedWithResult(ctx, taskID, errorCode, errorMsg, nil)
}

func (m *acsHTaskService) MarkTaskFailedWithResult(ctx context.Context, taskID string, errorCode int, errorMsg string, result json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		if t.ID == taskID {
			t.Status = task.TaskStatusFailed
			t.ErrorCode = errorCode
			t.ErrorMessage = errorMsg
			if len(result) > 0 {
				t.Result = result
			}
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

func (m *acsHTaskService) GetTask(ctx context.Context, taskID string) (*task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		if t.ID == taskID {
			return t, nil
		}
	}
	return nil, nil
}

func (m *acsHTaskService) GetQueueLength(ctx context.Context, deviceSN string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for i := m.popIndex; i < len(m.tasks); i++ {
		if m.tasks[i].DeviceSN == deviceSN && m.tasks[i].Status == task.TaskStatusPending {
			count++
		}
	}
	return count, nil
}

func (m *acsHTaskService) RecoverPendingTasks(ctx context.Context, deviceSN string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recoveredDevices = append(m.recoveredDevices, deviceSN)
	return nil
}

func (m *acsHTaskService) CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	maxRetries := 3 // CreateTaskRequest.MaxRetries 为 *int：nil → 默认 3
	if req.MaxRetries != nil {
		maxRetries = *req.MaxRetries
	}
	t := &task.Task{
		ID:          fmt.Sprintf("task-%d", len(m.tasks)+1),
		DeviceSN:    req.DeviceSN,
		Method:      req.Method,
		Params:      req.Params,
		Priority:    req.Priority,
		Status:      task.TaskStatusPending,
		MaxRetries:  maxRetries,
		CreatedAt:   time.Now(),
		Source:      req.Source,
		CreatorID:   req.CreatorID,
		Description: req.Description,
	}
	m.tasks = append(m.tasks, t)
	return t, nil
}

func TestQueueAutoGPVAfterSPVSkipsCorrelatedDeviceAccessAction(t *testing.T) {
	taskSvc := newAcsHTaskService()
	h := &Handler{taskService: taskSvc}
	params := json.RawMessage(`{"values":[{"name":"Device.Services.FAPService.1.FAPControl.LTE.AdminState","value":"0","type":"xsd:boolean"}]}`)

	h.queueAutoGPVAfterSPV(context.Background(), &task.Task{
		ID: "access-spv", DeviceSN: "SN-ACCESS", Params: params,
		Source: task.TaskSourceDeviceAccess,
	}, zap.NewNop())

	require.Empty(t, taskSvc.tasks)

	h.queueAutoGPVAfterSPV(context.Background(), &task.Task{
		ID: "api-spv-1", DeviceSN: "SN-API", Params: params,
		Source: task.TaskSourceAPI,
	}, zap.NewNop())
	require.Len(t, taskSvc.tasks, 1)
	require.Equal(t, "GetParameterValues", taskSvc.tasks[0].Method)
}

func (m *acsHTaskService) LatestOpenTaskByDeviceAndMethod(
	_ context.Context,
	deviceSN, method, description string,
) (*task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.tasks) - 1; i >= 0; i-- {
		candidate := m.tasks[i]
		if candidate.DeviceSN == deviceSN &&
			candidate.Method == method &&
			candidate.Description == description &&
			(candidate.Status == task.TaskStatusPending || candidate.Status == task.TaskStatusSent) {
			return candidate, nil
		}
	}
	return nil, nil
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
		taskService:   taskSvc, // new task service
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
