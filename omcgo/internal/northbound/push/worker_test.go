package push

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOutboxRepo is an in-memory implementation of OutboxRepository for tests.
type mockOutboxRepo struct {
	mu        sync.Mutex
	entries   map[uuid.UUID]*OutboxEntry
	insertErr error
}

func newMockOutboxRepo() *mockOutboxRepo {
	return &mockOutboxRepo{entries: make(map[uuid.UUID]*OutboxEntry)}
}

func (m *mockOutboxRepo) Insert(_ context.Context, entry *OutboxEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.insertErr != nil {
		return m.insertErr
	}
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	// Check for duplicate event_id + target_id.
	for _, e := range m.entries {
		if e.EventID == entry.EventID && e.TargetID == entry.TargetID {
			return nil // ON CONFLICT DO NOTHING
		}
	}
	cp := *entry
	cp.Status = OutboxStatusPending
	cp.CreatedAt = time.Now()
	cp.UpdatedAt = time.Now()
	m.entries[cp.ID] = &cp
	return nil
}

func (m *mockOutboxRepo) FetchPending(_ context.Context, limit int) ([]OutboxEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []OutboxEntry
	now := time.Now()
	for _, e := range m.entries {
		if len(result) >= limit {
			break
		}
		if (e.Status == OutboxStatusPending || e.Status == OutboxStatusProcessing) && !e.NextRetryAt.After(now) {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *mockOutboxRepo) MarkProcessing(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[id]; ok {
		e.Status = OutboxStatusProcessing
		e.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("not found")
}

func (m *mockOutboxRepo) MarkDelivered(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[id]; ok {
		e.Status = OutboxStatusDelivered
		e.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("not found")
}

func (m *mockOutboxRepo) MarkFailed(_ context.Context, id uuid.UUID, errMsg string, nextRetry time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[id]; ok {
		e.Status = OutboxStatusPending
		e.Attempts++
		e.LastError = errMsg
		e.NextRetryAt = nextRetry
		e.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("not found")
}

func (m *mockOutboxRepo) MarkDead(_ context.Context, id uuid.UUID, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[id]; ok {
		e.Status = OutboxStatusDead
		e.Attempts++
		e.LastError = errMsg
		e.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("not found")
}

func (m *mockOutboxRepo) ListDead(_ context.Context, limit, offset int) ([]OutboxEntry, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var dead []OutboxEntry
	for _, e := range m.entries {
		if e.Status == OutboxStatusDead {
			dead = append(dead, *e)
		}
	}
	total := len(dead)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return dead[offset:end], total, nil
}

func (m *mockOutboxRepo) Replay(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[id]; ok && e.Status == OutboxStatusDead {
		e.Status = OutboxStatusPending
		e.Attempts = 0
		e.LastError = ""
		e.NextRetryAt = time.Now()
		e.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("not found or not dead")
}

func (m *mockOutboxRepo) getEntry(id uuid.UUID) *OutboxEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[id]
	if !ok {
		return nil
	}
	cp := *e
	return &cp
}

func (m *mockOutboxRepo) allEntries() []*OutboxEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*OutboxEntry
	for _, e := range m.entries {
		cp := *e
		result = append(result, &cp)
	}
	return result
}

// --- Tests ---

func TestEnqueueEvent_WritesToOutbox(t *testing.T) {
	repo := newMockOutboxRepo()
	engine := NewEngine(nil, testLogger())
	engine.SetOutboxRepo(repo)
	engine.AddTarget(&Target{
		ID:         "t1",
		URL:        "http://example.com/push",
		DataTypes:  []string{"alarm"},
		RetryCount: 3,
		Enabled:    true,
	})

	evt, err := event.NewEvent(event.SubjectOSSAlarmForward, map[string]string{"alarm_id": "a1"})
	require.NoError(t, err)

	err = engine.EnqueueEvent(context.Background(), evt)
	require.NoError(t, err)

	entries := repo.allEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "t1", entries[0].TargetID)
	assert.Equal(t, OutboxStatusPending, entries[0].Status)
	assert.Equal(t, 3, entries[0].MaxAttempts)
}

func TestEnqueueEvent_SkipsDisabledTargets(t *testing.T) {
	repo := newMockOutboxRepo()
	engine := NewEngine(nil, testLogger())
	engine.SetOutboxRepo(repo)
	engine.AddTarget(&Target{
		ID:        "disabled",
		URL:       "http://example.com/push",
		DataTypes: []string{"alarm"},
		Enabled:   false,
	})

	evt, err := event.NewEvent(event.SubjectOSSAlarmForward, map[string]string{"test": "data"})
	require.NoError(t, err)

	err = engine.EnqueueEvent(context.Background(), evt)
	require.NoError(t, err)
	assert.Empty(t, repo.allEntries())
}

func TestEnqueueEvent_FallbackWithoutOutbox(t *testing.T) {
	var called int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewEngine(nil, testLogger())
	// No outbox repo set — should fall back to direct delivery.
	engine.AddTarget(&Target{
		ID:         "direct",
		URL:        server.URL,
		DataTypes:  []string{"alarm"},
		RetryCount: 1,
		Enabled:    true,
	})

	evt, err := event.NewEvent(event.SubjectOSSAlarmForward, map[string]string{"test": "data"})
	require.NoError(t, err)

	err = engine.EnqueueEvent(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

func TestOutboxWorker_ProcessEntry_Success(t *testing.T) {
	var deliveryCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&deliveryCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := newMockOutboxRepo()
	engine := NewEngine(nil, testLogger())
	engine.SetOutboxRepo(repo)
	engine.AddTarget(&Target{
		ID:         "t1",
		URL:        server.URL,
		DataTypes:  []string{"alarm"},
		RetryCount: 1,
		Enabled:    true,
	})

	entryID := uuid.New()
	payload, _ := json.Marshal(map[string]interface{}{
		"event_id": "evt-1",
		"subject":  event.SubjectOSSAlarmForward,
		"payload":  map[string]string{"alarm_id": "a1"},
	})

	entry := &OutboxEntry{
		ID:          entryID,
		EventID:     "evt-1",
		Subject:     event.SubjectOSSAlarmForward,
		Payload:     payload,
		TargetID:    "t1",
		Status:      OutboxStatusPending,
		MaxAttempts: 3,
		NextRetryAt: time.Now(),
	}
	require.NoError(t, repo.Insert(context.Background(), entry))

	worker := NewOutboxWorker(engine, repo, testLogger())
	worker.processEntry(context.Background(), *repo.getEntry(entryID))

	assert.Equal(t, int32(1), atomic.LoadInt32(&deliveryCount))
	e := repo.getEntry(entryID)
	require.NotNil(t, e)
	assert.Equal(t, OutboxStatusDelivered, e.Status)
}

func TestOutboxWorker_ProcessEntry_FailThenDead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := newMockOutboxRepo()
	engine := NewEngine(nil, testLogger())
	engine.SetOutboxRepo(repo)
	engine.AddTarget(&Target{
		ID:         "t1",
		URL:        server.URL,
		DataTypes:  []string{"alarm"},
		RetryCount: 1,
		Enabled:    true,
	})

	entryID := uuid.New()
	payload, _ := json.Marshal(map[string]interface{}{"event_id": "evt-1", "subject": "oss.alarm.forward"})

	entry := &OutboxEntry{
		ID:          entryID,
		EventID:     "evt-1",
		Subject:     event.SubjectOSSAlarmForward,
		Payload:     payload,
		TargetID:    "t1",
		Status:      OutboxStatusPending,
		Attempts:    2, // Already 2 attempts, max is 3.
		MaxAttempts: 3,
		NextRetryAt: time.Now(),
	}
	require.NoError(t, repo.Insert(context.Background(), entry))
	// Manually set attempts to 2 (mock always starts at 0).
	repo.mu.Lock()
	repo.entries[entryID].Attempts = 2
	repo.mu.Unlock()

	worker := NewOutboxWorker(engine, repo, testLogger())
	worker.processEntry(context.Background(), OutboxEntry{
		ID:          entryID,
		EventID:     "evt-1",
		Subject:     event.SubjectOSSAlarmForward,
		Payload:     payload,
		TargetID:    "t1",
		Status:      OutboxStatusPending,
		Attempts:    2,
		MaxAttempts: 3,
		NextRetryAt: time.Now(),
	})

	e := repo.getEntry(entryID)
	require.NotNil(t, e)
	assert.Equal(t, OutboxStatusDead, e.Status)
	assert.NotEmpty(t, e.LastError)
}

func TestOutboxWorker_ProcessEntry_TargetNotFound(t *testing.T) {
	repo := newMockOutboxRepo()
	engine := NewEngine(nil, testLogger())
	engine.SetOutboxRepo(repo)
	// No targets added.

	entryID := uuid.New()
	payload, _ := json.Marshal(map[string]interface{}{"event_id": "evt-1"})

	entry := &OutboxEntry{
		ID:          entryID,
		EventID:     "evt-1",
		Subject:     event.SubjectOSSAlarmForward,
		Payload:     payload,
		TargetID:    "nonexistent",
		Status:      OutboxStatusPending,
		MaxAttempts: 3,
		NextRetryAt: time.Now(),
	}
	require.NoError(t, repo.Insert(context.Background(), entry))

	worker := NewOutboxWorker(engine, repo, testLogger())
	worker.processEntry(context.Background(), *repo.getEntry(entryID))

	e := repo.getEntry(entryID)
	require.NotNil(t, e)
	assert.Equal(t, OutboxStatusDead, e.Status)
	assert.Contains(t, e.LastError, "push target not found")
}

func TestOutboxWorker_CircuitBreakerOpen(t *testing.T) {
	repo := newMockOutboxRepo()
	engine := NewEngine(nil, testLogger())
	engine.SetOutboxRepo(repo)
	engine.AddTarget(&Target{
		ID:         "t1",
		URL:        "http://unreachable.example.com",
		DataTypes:  []string{"alarm"},
		RetryCount: 1,
		Enabled:    true,
	})

	// Trip the circuit breaker.
	cb := engine.GetCircuitBreaker("t1")
	require.NotNil(t, cb)
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	assert.Equal(t, reliability.StateOpen, cb.State())

	entryID := uuid.New()
	payload, _ := json.Marshal(map[string]interface{}{"event_id": "evt-1"})

	entry := &OutboxEntry{
		ID:          entryID,
		EventID:     "evt-1",
		Subject:     event.SubjectOSSAlarmForward,
		Payload:     payload,
		TargetID:    "t1",
		Status:      OutboxStatusPending,
		MaxAttempts: 3,
		NextRetryAt: time.Now(),
	}
	require.NoError(t, repo.Insert(context.Background(), entry))

	worker := NewOutboxWorker(engine, repo, testLogger())
	worker.processEntry(context.Background(), *repo.getEntry(entryID))

	e := repo.getEntry(entryID)
	require.NotNil(t, e)
	// Should be marked failed with circuit breaker open message, not dead.
	assert.Equal(t, OutboxStatusPending, e.Status)
	assert.Equal(t, 1, e.Attempts)
	assert.Contains(t, e.LastError, "circuit breaker open")
}

func TestCalcBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
	}

	for _, tt := range tests {
		got := calcBackoff(tt.attempt)
		assert.Equal(t, tt.expected, got, "attempt %d", tt.attempt)
	}
}
