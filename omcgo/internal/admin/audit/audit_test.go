package audit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ----------------------------------------------------------------------------
// Test fixtures
// ----------------------------------------------------------------------------

// captureSink records the entries it receives.
type captureSink struct {
	mu      sync.Mutex
	entries []Entry
	err     error
}

func (s *captureSink) Write(_ context.Context, e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	return s.err
}

func (s *captureSink) snapshot() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

// resetDefaults returns the package singletons to a clean slate so each
// test starts deterministically.
func resetDefaults(t *testing.T) {
	t.Helper()
	defaultSink.Store(sinkHolder{s: noopSink{}})
	// Replace fallback logger with a no-op so tests do not emit noise.
	SetFallbackLogger(zap.NewNop())
}

// ----------------------------------------------------------------------------
// Tests
// ----------------------------------------------------------------------------

func TestActionConstants_AreCharterMandated(t *testing.T) {
	// W3.G.2 charter mandates exactly these 5 categories.
	want := map[string]string{
		"login":   ActionLogin,
		"config":  ActionConfig,
		"upgrade": ActionUpgrade,
		"reboot":  ActionReboot,
		"delete":  ActionDelete,
	}
	for k, got := range want {
		assert.Equal(t, k, got, "Action%s constant should equal %q", k, k)
	}
}

func TestDefault_NoSinkConfigured_IsNoop(t *testing.T) {
	resetDefaults(t)
	// Should not panic, should not error.
	Log(context.Background(), Entry{
		Action:       ActionLogin,
		ResourceType: ResourceAuth,
		Success:      true,
	})
}

func TestSetDefault_RoutesEntriesToSink(t *testing.T) {
	resetDefaults(t)
	sink := &captureSink{}
	SetDefault(sink)
	t.Cleanup(func() { SetDefault(nil) })

	uid := uuid.New()
	Log(context.Background(), Entry{
		UserID:       &uid,
		Username:     "alice",
		Action:       ActionDelete,
		ResourceType: ResourceDevice,
		ResourceID:   "dev-123",
		Success:      true,
	})

	got := sink.snapshot()
	require.Len(t, got, 1)
	assert.Equal(t, ActionDelete, got[0].Action)
	assert.Equal(t, ResourceDevice, got[0].ResourceType)
	assert.Equal(t, "dev-123", got[0].ResourceID)
	assert.Equal(t, "alice", got[0].Username)
	assert.Equal(t, &uid, got[0].UserID)
	assert.True(t, got[0].Success)
}

func TestSetDefault_NilResetsToNoop(t *testing.T) {
	resetDefaults(t)
	sink := &captureSink{}
	SetDefault(sink)
	SetDefault(nil) // should reset to noop, not crash.

	Log(context.Background(), Entry{Action: ActionLogin, Success: true})
	assert.Empty(t, sink.snapshot(), "after SetDefault(nil), entries should not reach prior sink")
}

func TestLog_EmptyAction_IsDropped(t *testing.T) {
	resetDefaults(t)
	sink := &captureSink{}
	SetDefault(sink)
	t.Cleanup(func() { SetDefault(nil) })

	Log(context.Background(), Entry{Action: "", ResourceType: ResourceDevice})
	assert.Empty(t, sink.snapshot(), "entries with empty action must be dropped")
}

func TestLog_SinkError_IsSwallowed(t *testing.T) {
	resetDefaults(t)
	sink := &captureSink{err: errors.New("boom")}
	SetDefault(sink)
	t.Cleanup(func() { SetDefault(nil) })

	// Must not panic, must not return — Log signature is void.
	Log(context.Background(), Entry{
		Action:       ActionUpgrade,
		ResourceType: ResourceFirmware,
		Success:      true,
	})
	// Entry was still passed to the sink even though Write returned error.
	assert.Len(t, sink.snapshot(), 1)
}

func TestLogAsync_DeliversAfterShortWait(t *testing.T) {
	resetDefaults(t)
	sink := &captureSink{}
	SetDefault(sink)
	t.Cleanup(func() { SetDefault(nil) })

	LogAsync(Entry{
		Action:       ActionReboot,
		ResourceType: ResourceDevice,
		ResourceID:   "dev-async",
		Success:      true,
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(sink.snapshot()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	got := sink.snapshot()
	require.Len(t, got, 1, "LogAsync should deliver within 2s")
	assert.Equal(t, "dev-async", got[0].ResourceID)
}

func TestEntry_FailureCarriesErrorMessage(t *testing.T) {
	resetDefaults(t)
	sink := &captureSink{}
	SetDefault(sink)
	t.Cleanup(func() { SetDefault(nil) })

	Log(context.Background(), Entry{
		Action:       ActionConfig,
		ResourceType: ResourceConfig,
		Success:      false,
		ErrorMessage: "task queue full",
	})
	got := sink.snapshot()
	require.Len(t, got, 1)
	assert.False(t, got[0].Success)
	assert.Equal(t, "task queue full", got[0].ErrorMessage)
}

func TestDefault_AlwaysReturnsNonNilSink(t *testing.T) {
	resetDefaults(t)
	// Default should always return a non-nil Sink (noop fallback).
	require.NotNil(t, Default())
	// Calling Write on the default sink must not error.
	require.NoError(t, Default().Write(context.Background(), Entry{Action: ActionLogin}))
}
