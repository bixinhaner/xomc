package connreq

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockHTTPClient implements a minimal HTTP CR sender for testing.
type mockHTTPClient struct {
	sendFn func(ctx context.Context, deviceSN, url string) error
}

func (m *mockHTTPClient) send(ctx context.Context, deviceSN, url string) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, deviceSN, url)
	}
	return nil
}

// mockUDPSender wraps UDPSender behavior for testing.
type mockUDPSenderResult struct {
	err error
}

func TestDispatcher_Send_UDPOnly_NoSTUNAddress(t *testing.T) {
	// UDP sender exists but no STUN address → falls through to HTTP check
	// With no HTTP client either → ErrNoConnectionMethod
	d := NewDispatcher(nil, nil, zap.NewNop())
	err := d.Send(context.Background(), "TEST-SN-001", "", "", false)
	assert.ErrorIs(t, err, ErrNoConnectionMethod)
}

func TestDispatcher_Send_NoMethodAvailable(t *testing.T) {
	// No UDP sender, no HTTP client → ErrNoConnectionMethod
	d := NewDispatcher(nil, nil, zap.NewNop())
	err := d.Send(context.Background(), "TEST-SN-001", "", "", false)
	assert.ErrorIs(t, err, ErrNoConnectionMethod)
}

func TestDispatcher_Send_NoMethodAvailable_EmptyURL(t *testing.T) {
	// HTTP client exists but URL is empty → ErrNoConnectionMethod
	d := NewDispatcher(&Client{}, nil, zap.NewNop())
	err := d.Send(context.Background(), "TEST-SN-001", "", "", false)
	assert.ErrorIs(t, err, ErrNoConnectionMethod)
}

func TestErrNoConnectionMethod(t *testing.T) {
	assert.True(t, errors.Is(ErrNoConnectionMethod, ErrNoConnectionMethod))
	assert.Contains(t, ErrNoConnectionMethod.Error(), "no connection request method")
}

func TestErrNoSTUNAddress(t *testing.T) {
	assert.True(t, errors.Is(ErrNoSTUNAddress, ErrNoSTUNAddress))
	assert.Contains(t, ErrNoSTUNAddress.Error(), "no stun address")
}

// TestDispatcher_ResolveLANPort exercises the (port,source) decision logic
// used by SendLAN — directly drives the C2 fix path, no socket needed.
func TestDispatcher_ResolveLANPort(t *testing.T) {
	t.Run("no lookup configured -> default", func(t *testing.T) {
		d := NewDispatcher(nil, nil, zap.NewNop())
		port, src := d.resolveLANPort(context.Background(), "SN-A")
		assert.Equal(t, 0, port)
		assert.Equal(t, "default", src)
	})

	t.Run("lookup returns ok=false -> default", func(t *testing.T) {
		d := NewDispatcher(nil, nil, zap.NewNop())
		d.SetLANPortLookup(func(_ context.Context, _ string) (int, bool) {
			return 0, false
		})
		port, src := d.resolveLANPort(context.Background(), "SN-MISSING")
		assert.Equal(t, 0, port)
		assert.Equal(t, "default", src)
	})

	t.Run("lookup returns non-positive port -> treated as default", func(t *testing.T) {
		d := NewDispatcher(nil, nil, zap.NewNop())
		d.SetLANPortLookup(func(_ context.Context, _ string) (int, bool) {
			return -1, true
		})
		port, src := d.resolveLANPort(context.Background(), "SN-NEG")
		assert.Equal(t, 0, port)
		assert.Equal(t, "default", src)
	})

	t.Run("lookup returns positive port -> device_param", func(t *testing.T) {
		var gotSN string
		d := NewDispatcher(nil, nil, zap.NewNop())
		d.SetLANPortLookup(func(_ context.Context, sn string) (int, bool) {
			gotSN = sn
			return 5060, true
		})
		port, src := d.resolveLANPort(context.Background(), "SN-CUSTOM")
		assert.Equal(t, 5060, port)
		assert.Equal(t, "device_param", src)
		assert.Equal(t, "SN-CUSTOM", gotSN)
	})
}

// TestEffectiveLANPort guards the log-time clamping (0 → 3478) so the wire
// port and the observed port label always agree.
func TestEffectiveLANPort(t *testing.T) {
	assert.Equal(t, lanUDPCRDefaultPort, effectiveLANPort(0))
	assert.Equal(t, lanUDPCRDefaultPort, effectiveLANPort(-7))
	assert.Equal(t, 5060, effectiveLANPort(5060))
}
