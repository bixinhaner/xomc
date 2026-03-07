package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/common/event"
	"github.com/omcgo/omcgo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestEngine_AddRemoveListTargets(t *testing.T) {
	engine := NewEngine(nil, testLogger())

	// Initially empty
	assert.Empty(t, engine.ListTargets())

	// Add a target
	engine.AddTarget(&Target{
		ID:        "t1",
		URL:       "http://example.com/push",
		AuthType:  "bearer",
		AuthToken: "secret",
		DataTypes: []string{"alarm", "pm"},
		Format:    "json",
		Enabled:   true,
	})

	targets := engine.ListTargets()
	assert.Len(t, targets, 1)
	assert.Equal(t, "t1", targets[0].ID)
	assert.Equal(t, "http://example.com/push", targets[0].URL)

	// Add another target
	engine.AddTarget(&Target{
		ID:        "t2",
		URL:       "http://oss.example.com/data",
		DataTypes: []string{"config"},
		Enabled:   true,
	})

	assert.Len(t, engine.ListTargets(), 2)

	// Remove first target
	assert.True(t, engine.RemoveTarget("t1"))
	assert.Len(t, engine.ListTargets(), 1)
	assert.Equal(t, "t2", engine.ListTargets()[0].ID)

	// Remove non-existent returns false
	assert.False(t, engine.RemoveTarget("nonexistent"))
}

func TestEngine_GetTarget(t *testing.T) {
	engine := NewEngine(nil, testLogger())

	engine.AddTarget(&Target{
		ID:      "t1",
		URL:     "http://example.com",
		Enabled: true,
	})

	// Found
	target := engine.GetTarget("t1")
	require.NotNil(t, target)
	assert.Equal(t, "t1", target.ID)

	// Not found
	assert.Nil(t, engine.GetTarget("nonexistent"))
}

func TestEngine_FromConfig(t *testing.T) {
	cfgTargets := []config.PushTargetConfig{
		{
			ID:         "cfg1",
			URL:        "http://oss1.example.com",
			AuthType:   "bearer",
			AuthToken:  "token1",
			DataTypes:  []string{"alarm"},
			Format:     "json",
			BatchSize:  100,
			RetryCount: 3,
			Enabled:    true,
		},
		{
			ID:         "cfg2",
			URL:        "http://oss2.example.com",
			DataTypes:  []string{"pm", "config"},
			RetryCount: 2,
			Enabled:    false,
		},
	}

	engine := NewEngine(cfgTargets, testLogger())
	targets := engine.ListTargets()
	assert.Len(t, targets, 2)

	t1 := engine.GetTarget("cfg1")
	require.NotNil(t, t1)
	assert.Equal(t, "http://oss1.example.com", t1.URL)
	assert.Equal(t, "bearer", t1.AuthType)
	assert.True(t, t1.Enabled)

	t2 := engine.GetTarget("cfg2")
	require.NotNil(t, t2)
	assert.False(t, t2.Enabled)
}

func TestEngine_Deliver_Success(t *testing.T) {
	var receivedCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedCount, 1)

		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.NotEmpty(t, body["event_id"])
		assert.Equal(t, "oss.alarm.forward", body["subject"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewEngine(nil, testLogger())
	engine.AddTarget(&Target{
		ID:         "test-target",
		URL:        server.URL,
		AuthType:   "bearer",
		AuthToken:  "test-token",
		DataTypes:  []string{"alarm"},
		RetryCount: 1,
		Enabled:    true,
	})

	evt, err := event.NewEvent(event.SubjectOSSAlarmForward, map[string]string{
		"alarm_id": "123",
		"severity": "critical",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = engine.handleEvent(ctx, evt)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&receivedCount))
}

func TestEngine_Deliver_DisabledTarget(t *testing.T) {
	var called int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewEngine(nil, testLogger())
	engine.AddTarget(&Target{
		ID:        "disabled-target",
		URL:       server.URL,
		DataTypes: []string{"alarm"},
		Enabled:   false, // disabled
	})

	evt, err := event.NewEvent(event.SubjectOSSAlarmForward, map[string]string{"test": "data"})
	require.NoError(t, err)

	err = engine.handleEvent(context.Background(), evt)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestEngine_Deliver_DataTypeMismatch(t *testing.T) {
	var called int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewEngine(nil, testLogger())
	engine.AddTarget(&Target{
		ID:        "pm-only",
		URL:       server.URL,
		DataTypes: []string{"pm"}, // pm only, not alarm
		Enabled:   true,
	})

	// Send alarm event — should not match
	evt, err := event.NewEvent(event.SubjectOSSAlarmForward, map[string]string{"test": "data"})
	require.NoError(t, err)

	err = engine.handleEvent(context.Background(), evt)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestEngine_Deliver_RetryOnFailure(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewEngine(nil, testLogger())
	engine.AddTarget(&Target{
		ID:         "retry-target",
		URL:        server.URL,
		DataTypes:  []string{"alarm"},
		RetryCount: 3,
		Enabled:    true,
	})

	evt, err := event.NewEvent(event.SubjectOSSAlarmForward, map[string]string{"test": "data"})
	require.NoError(t, err)

	err = engine.handleEvent(context.Background(), evt)
	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestEngine_Deliver_ExhaustedRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	engine := NewEngine(nil, testLogger())
	engine.AddTarget(&Target{
		ID:         "failing-target",
		URL:        server.URL,
		DataTypes:  []string{"pm"},
		RetryCount: 2,
		Enabled:    true,
	})

	evt, err := event.NewEvent(event.SubjectOSSPMExport, map[string]string{"test": "data"})
	require.NoError(t, err)

	// handleEvent logs the error but does not return it
	err = engine.handleEvent(context.Background(), evt)
	assert.NoError(t, err) // errors are logged, not propagated
}

func TestEngine_Deliver_CancelledContext(t *testing.T) {
	// Server that blocks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewEngine(nil, testLogger())
	engine.AddTarget(&Target{
		ID:         "slow-target",
		URL:        server.URL,
		DataTypes:  []string{"alarm"},
		RetryCount: 1,
		Enabled:    true,
	})

	evt, err := event.NewEvent(event.SubjectOSSAlarmForward, map[string]string{"test": "data"})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should fail due to context cancellation
	err = engine.handleEvent(ctx, evt)
	assert.NoError(t, err) // errors are logged, not propagated from handleEvent
}

func TestDataTypeFromSubject(t *testing.T) {
	tests := []struct {
		subject  string
		expected string
	}{
		{event.SubjectOSSAlarmForward, "alarm"},
		{event.SubjectOSSPMExport, "pm"},
		{event.SubjectOSSConfigSnapshot, "config"},
		{"unknown.subject", ""},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			assert.Equal(t, tt.expected, dataTypeFromSubject(tt.subject))
		})
	}
}

func TestMatchesDataType(t *testing.T) {
	assert.True(t, matchesDataType([]string{"alarm", "pm"}, "alarm"))
	assert.True(t, matchesDataType([]string{"alarm", "pm"}, "pm"))
	assert.False(t, matchesDataType([]string{"alarm", "pm"}, "config"))
	assert.False(t, matchesDataType(nil, "alarm"))
}
