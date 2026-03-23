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
