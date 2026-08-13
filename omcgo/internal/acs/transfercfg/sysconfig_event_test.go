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

func TestHandleSysConfigSavedEvent_RefreshesProtocolAndHTTPSAddresses(t *testing.T) {
	values := map[string]string{
		KeyProtocolPolicy:       ProtocolPolicyForceHTTP,
		KeyHTTPSUploadBaseURL:   "https://old-upload.example.com",
		KeyHTTPSDownloadBaseURL: "https://old-download.example.com",
	}
	policy := NewPolicy(Snapshot{}, func(_ context.Context, category, key string) (string, bool) {
		if category != Category {
			return "", false
		}
		value, ok := values[key]
		return value, ok
	})

	initial := policy.Snapshot(context.Background())
	require.Equal(t, ProtocolPolicyForceHTTP, initial.ProtocolPolicy)
	require.Equal(t, "https://old-upload.example.com", initial.Upload.HTTPSBaseURL)

	values[KeyProtocolPolicy] = ProtocolPolicyPreferHTTPS
	values[KeyHTTPSUploadBaseURL] = "https://new-upload.example.com"
	values[KeyHTTPSDownloadBaseURL] = "https://new-download.example.com"
	handler := HandleSysConfigSavedEvent(policy, zap.NewNop())
	evt, err := coreevent.NewEvent(coreevent.SubjectSysConfigSaved, coreevent.SysConfigSavedPayload{Category: Category})
	require.NoError(t, err)
	require.NoError(t, handler(context.Background(), evt))

	refreshed := policy.Snapshot(context.Background())
	assert.Equal(t, ProtocolPolicyPreferHTTPS, refreshed.ProtocolPolicy)
	assert.Equal(t, "https://new-upload.example.com", refreshed.Upload.HTTPSBaseURL)
	assert.Equal(t, "https://new-download.example.com", refreshed.Download.HTTPSBaseURL)
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
