package transfercfg

import (
	"context"
	"testing"
	"time"

	coreevent "github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandleSysConfigSavedEvent_InvalidatesMatchingCategory(t *testing.T) {
	policy := NewPolicy(Snapshot{}, nil)
	policy.cache.Store(&Snapshot{expiresAt: time.Now().Add(time.Minute)})

	handler := HandleSysConfigSavedEvent(policy, zap.NewNop())
	evt, err := coreevent.NewEvent(coreevent.SubjectSysConfigSaved, coreevent.SysConfigSavedPayload{Category: Category})
	require.NoError(t, err)

	require.NoError(t, handler(context.Background(), evt))
	assert.Nil(t, policy.cache.Load())
}

func TestHandleSysConfigSavedEvent_IgnoresOtherCategories(t *testing.T) {
	policy := NewPolicy(Snapshot{}, nil)
	cached := &Snapshot{expiresAt: time.Now().Add(time.Minute)}
	policy.cache.Store(cached)

	handler := HandleSysConfigSavedEvent(policy, zap.NewNop())
	evt, err := coreevent.NewEvent(coreevent.SubjectSysConfigSaved, coreevent.SysConfigSavedPayload{Category: "security"})
	require.NoError(t, err)

	require.NoError(t, handler(context.Background(), evt))
	assert.Same(t, cached, policy.cache.Load())
}

func TestHandleSysConfigSavedEvent_IgnoresDecodeFailure(t *testing.T) {
	policy := NewPolicy(Snapshot{}, nil)
	cached := &Snapshot{expiresAt: time.Now().Add(time.Minute)}
	policy.cache.Store(cached)

	handler := HandleSysConfigSavedEvent(policy, zap.NewNop())
	evt := coreevent.Event{Subject: coreevent.SubjectSysConfigSaved, Payload: []byte(`{"category":`)}

	require.NoError(t, handler(context.Background(), evt))
	assert.Same(t, cached, policy.cache.Load())
}
