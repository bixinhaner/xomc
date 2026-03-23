package connreq

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
)

var (
	// ErrNoConnectionMethod means no HTTP URL or STUN address is available.
	ErrNoConnectionMethod = errors.New("no connection request method available")
	// ErrNoSTUNAddress means the device has no known STUN address.
	ErrNoSTUNAddress = errors.New("no stun address for device")
)

// Dispatcher unifies HTTP and UDP Connection Request sending.
// It tries UDP first (for NAT-traversed devices), then falls back to HTTP.
type Dispatcher struct {
	httpClient *Client
	udpSender  *UDPSender
	logger     *zap.Logger
}

// NewDispatcher creates a new Connection Request dispatcher.
// Both httpClient and udpSender are optional (nil means that channel is unavailable).
func NewDispatcher(httpClient *Client, udpSender *UDPSender, logger *zap.Logger) *Dispatcher {
	return &Dispatcher{
		httpClient: httpClient,
		udpSender:  udpSender,
		logger:     logger,
	}
}

// Send attempts to wake a device via Connection Request.
//
// Strategy:
//  1. If STUN address is known → send UDP CR (works through NAT)
//  2. If HTTP CR URL is known → send HTTP CR (works for directly reachable devices)
//  3. Both unavailable → return ErrNoConnectionMethod (device must wait for periodic Inform)
//
// Parameters:
//   - deviceSN: device serial number (used for STUN lookup and dedup)
//   - httpURL: device's HTTP Connection Request URL (may be empty)
//   - serverAddr: this server's address for CPE UDP CR URL field (may be empty)
//   - isENB: whether the device is an eNB (base station) for non-standard UDP format
func (d *Dispatcher) Send(ctx context.Context, deviceSN, httpURL, serverAddr string, isENB bool) error {
	// Try UDP Connection Request first (works through NAT)
	if d.udpSender != nil {
		err := d.udpSender.Send(ctx, deviceSN, serverAddr, isENB)
		if err == nil {
			d.logger.Debug("udp connection request sent",
				zap.String("device_sn", deviceSN))
			return nil
		}
		if !errors.Is(err, ErrNoSTUNAddress) {
			d.logger.Warn("udp connection request failed, trying http",
				zap.String("device_sn", deviceSN),
				zap.Error(err))
		}
	}

	// Fall back to HTTP Connection Request
	if d.httpClient != nil && httpURL != "" {
		if err := d.httpClient.Send(ctx, deviceSN, httpURL); err != nil {
			return fmt.Errorf("http connection request: %w", err)
		}
		d.logger.Debug("http connection request sent",
			zap.String("device_sn", deviceSN))
		return nil
	}

	// No method available — device will connect on next periodic Inform
	d.logger.Debug("no connection request method available, waiting for periodic inform",
		zap.String("device_sn", deviceSN))
	return ErrNoConnectionMethod
}
