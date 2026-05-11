package push

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeActiveProvider 简易 ActiveServerProvider 实现。
type fakeActiveProvider struct {
	info *ActiveServerInfo
	err  error
}

func (f *fakeActiveProvider) GetActiveForPush(ctx context.Context) (*ActiveServerInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.info, nil
}

func TestEngine_RefreshActiveTarget_Upsert(t *testing.T) {
	e := NewEngine(nil, zap.NewNop())
	e.SetActiveServerProvider(&fakeActiveProvider{
		info: &ActiveServerInfo{Role: "primary", Host: "10.0.0.1", Port: 8443},
	})

	err := e.RefreshActiveTarget(context.Background())
	require.NoError(t, err)

	target := e.GetTarget(ActiveTargetID)
	require.NotNil(t, target)
	assert.Equal(t, "http://10.0.0.1:8443", target.URL)
	assert.True(t, target.Enabled)
	assert.Equal(t, int64(1), e.RefreshSuccessCount())
	assert.Equal(t, int64(0), e.RefreshFailureCount())
}

func TestEngine_RefreshActiveTarget_NoActive_RemovesExisting(t *testing.T) {
	e := NewEngine(nil, zap.NewNop())

	// 先添加一个 active target 模拟前次 refresh
	e.SetActiveServerProvider(&fakeActiveProvider{
		info: &ActiveServerInfo{Role: "primary", Host: "10.0.0.1", Port: 8443},
	})
	require.NoError(t, e.RefreshActiveTarget(context.Background()))
	require.NotNil(t, e.GetTarget(ActiveTargetID))

	// provider 返 nil → 应移除
	e.SetActiveServerProvider(&fakeActiveProvider{info: nil})
	require.NoError(t, e.RefreshActiveTarget(context.Background()))
	assert.Nil(t, e.GetTarget(ActiveTargetID), "active target should be removed when no active server")
}

func TestEngine_RefreshActiveTarget_ProviderError(t *testing.T) {
	e := NewEngine(nil, zap.NewNop())
	e.SetActiveServerProvider(&fakeActiveProvider{err: errors.New("db down")})

	err := e.RefreshActiveTarget(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get active server")
	assert.Equal(t, int64(0), e.RefreshSuccessCount())
	assert.Equal(t, int64(1), e.RefreshFailureCount())
}

func TestEngine_RefreshActiveTarget_NilProvider_Noop(t *testing.T) {
	e := NewEngine(nil, zap.NewNop())
	// 未注入 provider
	err := e.RefreshActiveTarget(context.Background())
	require.NoError(t, err, "no provider should be silent no-op")
	assert.Equal(t, int64(0), e.RefreshSuccessCount())
}

func TestEngine_RefreshActiveTarget_SwitchUpdatesURL(t *testing.T) {
	e := NewEngine(nil, zap.NewNop())
	provider := &fakeActiveProvider{
		info: &ActiveServerInfo{Role: "primary", Host: "10.0.0.1", Port: 8443},
	}
	e.SetActiveServerProvider(provider)
	require.NoError(t, e.RefreshActiveTarget(context.Background()))
	assert.Equal(t, "http://10.0.0.1:8443", e.GetTarget(ActiveTargetID).URL)

	// 切换 active 到 standby
	provider.info = &ActiveServerInfo{Role: "standby", Host: "10.0.0.2", Port: 8444}
	require.NoError(t, e.RefreshActiveTarget(context.Background()))
	target := e.GetTarget(ActiveTargetID)
	require.NotNil(t, target)
	assert.Equal(t, "http://10.0.0.2:8444", target.URL, "URL should reflect new active server")
}

func TestEngine_HandleServerChanged_TriggersRefresh(t *testing.T) {
	e := NewEngine(nil, zap.NewNop())
	provider := &fakeActiveProvider{
		info: &ActiveServerInfo{Role: "primary", Host: "10.0.0.1", Port: 8443},
	}
	e.SetActiveServerProvider(provider)

	payload, _ := json.Marshal(map[string]interface{}{
		"role": "primary", "action": "active_switch", "host": "10.0.0.1", "port": 8443,
	})
	evt := event.Event{
		Subject: event.SubjectNorthboundServerChanged,
		Payload: payload,
	}
	err := e.handleServerChanged(context.Background(), evt)
	require.NoError(t, err)

	target := e.GetTarget(ActiveTargetID)
	require.NotNil(t, target)
	assert.Equal(t, "http://10.0.0.1:8443", target.URL)
}
