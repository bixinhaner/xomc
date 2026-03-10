package acs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/common/event"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock types (acsH prefix to avoid conflicts with other _test.go files)
// ---------------------------------------------------------------------------

type acsHSessionStore struct {
	createFn  func(ctx context.Context, deviceSN string, session *Session) error
	getFn     func(ctx context.Context, deviceSN string) (*Session, error)
	updateFn  func(ctx context.Context, deviceSN string, session *Session) error
	deleteFn  func(ctx context.Context, deviceSN string) error
	setTTLFn  func(ctx context.Context, deviceSN string, ttl time.Duration) error
}

func (m *acsHSessionStore) Create(ctx context.Context, deviceSN string, session *Session) error {
	if m.createFn != nil {
		return m.createFn(ctx, deviceSN, session)
	}
	return nil
}
func (m *acsHSessionStore) Get(ctx context.Context, deviceSN string) (*Session, error) {
	if m.getFn != nil {
		return m.getFn(ctx, deviceSN)
	}
	return nil, nil
}
func (m *acsHSessionStore) Update(ctx context.Context, deviceSN string, session *Session) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, deviceSN, session)
	}
	return nil
}
func (m *acsHSessionStore) Delete(ctx context.Context, deviceSN string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, deviceSN)
	}
	return nil
}
func (m *acsHSessionStore) SetTTL(ctx context.Context, deviceSN string, ttl time.Duration) error {
	if m.setTTLFn != nil {
		return m.setTTLFn(ctx, deviceSN, ttl)
	}
	return nil
}

type acsHCmdQueue struct {
	popFn func(ctx context.Context, deviceSN string) (*cmdqueue.Command, error)
}

func (m *acsHCmdQueue) Push(_ context.Context, _ string, _ *cmdqueue.Command) error { return nil }
func (m *acsHCmdQueue) Pop(ctx context.Context, deviceSN string) (*cmdqueue.Command, error) {
	if m.popFn != nil {
		return m.popFn(ctx, deviceSN)
	}
	return nil, nil
}
func (m *acsHCmdQueue) Peek(_ context.Context, _ string) (*cmdqueue.Command, error) { return nil, nil }
func (m *acsHCmdQueue) Len(_ context.Context, _ string) (int64, error)              { return 0, nil }
func (m *acsHCmdQueue) Clear(_ context.Context, _ string) error                     { return nil }

type acsHEventBus struct {
	publishFn func(ctx context.Context, subject string, evt event.Event) error
}

func (m *acsHEventBus) Publish(ctx context.Context, subject string, evt event.Event) error {
	if m.publishFn != nil {
		return m.publishFn(ctx, subject, evt)
	}
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
	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)
	return &Handler{
		sessionStore:  &acsHSessionStore{},
		commandQueue:  &acsHCmdQueue{},
		eventBus:      &acsHEventBus{},
		authenticator: &auth.NoopAuthenticator{},
		rpcDispatcher: rpc.NewDispatcher(),
		rateLimiter:   NewDeviceRateLimiter(100, 100),
		admission:     NewAdmissionController(1000),
		metrics:       metrics,
		logger:        zap.NewNop(),
	}
}

// ---------------------------------------------------------------------------
// Tests
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

func TestServeHTTP_EmptyBody_Returns204(t *testing.T) {
	h := newTestACSHandler()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	h.ServeHTTP(w, req)

	// Empty body with no session binding → 204
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestServeHTTP_UnknownSOAP_Returns400(t *testing.T) {
	h := newTestACSHandler()

	w := httptest.NewRecorder()
	body := `<soap:Envelope><soap:Body><UnknownMethod/></soap:Body></soap:Envelope>`
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(body))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCompleteSession(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)
	store := &acsHSessionStore{
		updateFn: func(_ context.Context, _ string, s *Session) error {
			assert.Equal(t, StateComplete, s.State)
			return nil
		},
	}

	h := &Handler{
		sessionStore: store,
		admission:    NewAdmissionController(100),
		metrics:      metrics,
		logger:       zap.NewNop(),
	}

	// Pre-increment so completeSession can decrement
	h.metrics.ActiveSessions.Inc()
	h.admission.Acquire()

	session := &Session{
		DeviceSN:  "SN-COMPLETE",
		State:     StateProcessing,
		StartedAt: time.Now().Add(-5 * time.Second),
		UpdatedAt: time.Now(),
	}

	h.completeSession(context.Background(), "SN-COMPLETE", "192.168.1.1:1234", session)

	assert.Equal(t, StateComplete, session.State)
}

func TestStartSessionReaper(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewACSMetrics(reg)
	admission := NewAdmissionController(100)

	h := &Handler{
		admission: admission,
		metrics:   metrics,
		logger:    zap.NewNop(),
	}

	// Store a stale session
	h.connSessions.Store("192.168.1.1:9999", connSessionEntry{
		DeviceSN:  "SN-STALE",
		CreatedAt: time.Now().Add(-10 * time.Minute),
	})

	h.metrics.ActiveSessions.Inc()
	h.admission.Acquire()

	// Start reaper with very short interval and low maxAge
	h.startSessionReaper(50*time.Millisecond, 1*time.Second)

	// Wait for reaper to run
	time.Sleep(200 * time.Millisecond)

	// Stale entry should be reaped
	_, loaded := h.connSessions.Load("192.168.1.1:9999")
	assert.False(t, loaded, "stale session should be reaped")
}
