package connreq

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	httpClient    *Client
	udpSender     *UDPSender
	metrics       *DispatcherMetrics
	lanPortLookup LANPortLookup
	logger        *zap.Logger
}

// LANPortLookup resolves the device-side UDP Connection Request port for
// SendLAN, typically by reading Device.ManagementServer.STUNServerPort from
// device_parameters. Returns (port, true) on success; (0, false) when the
// parameter is missing/unparseable so the UDPSender falls back to the
// hard-coded default (3478).
type LANPortLookup func(ctx context.Context, deviceSN string) (port int, ok bool)

// NewDispatcher creates a new Connection Request dispatcher.
// Both httpClient and udpSender are optional (nil means that channel is unavailable).
func NewDispatcher(httpClient *Client, udpSender *UDPSender, logger *zap.Logger) *Dispatcher {
	return &Dispatcher{
		httpClient: httpClient,
		udpSender:  udpSender,
		logger:     logger,
	}
}

// SetMetrics attaches Prometheus metrics to the dispatcher.
func (d *Dispatcher) SetMetrics(m *DispatcherMetrics) {
	d.metrics = m
}

// SetLANPortLookup attaches a per-device LAN UDP CR port resolver.
// nil keeps the hard-coded fallback (3478) for every SendLAN call.
func (d *Dispatcher) SetLANPortLookup(fn LANPortLookup) {
	d.lanPortLookup = fn
}

// Send attempts to wake a device via Connection Request.
//
// Strategy (UDP first because it's cheaper; both UDP paths fire concurrently
// because they target different receiver implementations and don't interfere):
//  1. UDP-STUN: send to NAT-mapped 5-tuple from STUN cache
//     (works for devices following TR-069 Annex G STUN-binding convention)
//  2. UDP-LAN:  send to LAN-IP:3478 parsed from httpURL
//     (works for devices whose receiver listens on a separate fixed-port socket,
//     e.g. BAICELLS BSC7041C243; only viable when ACS can reach device LAN IP)
//  3. HTTP fallback if both UDP paths produce no usable target.
//
// Parameters:
//   - deviceSN: device serial number (used for STUN lookup and dedup)
//   - httpURL: device's HTTP Connection Request URL (may be empty)
//   - serverAddr: this server's address for CPE UDP CR URL field (may be empty)
//   - isENB: whether the device is an eNB (base station) for non-standard UDP format
func (d *Dispatcher) Send(ctx context.Context, deviceSN, httpURL, serverAddr string, isENB bool) error {
	udpSentAny := false

	// UDP path 1: STUN-cache (NAT-mapped 5-tuple)
	if d.udpSender != nil {
		start := time.Now()
		err := d.udpSender.Send(ctx, deviceSN, serverAddr, isENB)
		if err == nil {
			d.logger.Info("udp-stun connection request sent",
				zap.String("device_sn", deviceSN))
			d.recordMetrics("udp", "success", time.Since(start))
			udpSentAny = true
		} else if !errors.Is(err, ErrNoSTUNAddress) {
			d.logger.Warn("udp-stun connection request failed",
				zap.String("device_sn", deviceSN),
				zap.Error(err))
			d.recordMetrics("udp", "failure", time.Since(start))
		} else {
			d.logger.Info("udp-stun skipped: no stun address cached",
				zap.String("device_sn", deviceSN))
		}
	}

	// UDP path 2: LAN-direct (httpURL host + per-device or fallback UDP CR port)
	if d.udpSender != nil && httpURL != "" {
		lanPort, portSource := d.resolveLANPort(ctx, deviceSN)
		start := time.Now()
		err := d.udpSender.SendLAN(ctx, deviceSN, httpURL, lanPort, isENB, serverAddr)
		if err == nil {
			d.logger.Info("udp-lan connection request sent",
				zap.String("device_sn", deviceSN),
				zap.String("http_url", httpURL),
				zap.Int("port", effectiveLANPort(lanPort)),
				zap.String("port_source", portSource))
			d.recordMetrics("udp_lan", "success", time.Since(start))
			udpSentAny = true
		} else if !errors.Is(err, ErrNoLANTarget) {
			d.logger.Warn("udp-lan connection request failed",
				zap.String("device_sn", deviceSN),
				zap.String("http_url", httpURL),
				zap.Int("port", effectiveLANPort(lanPort)),
				zap.String("port_source", portSource),
				zap.Error(err))
			d.recordMetrics("udp_lan", "failure", time.Since(start))
		}
	}

	if udpSentAny {
		return nil
	}

	// Fall back to HTTP Connection Request
	if d.httpClient != nil && httpURL != "" {
		start := time.Now()
		if err := d.httpClient.Send(ctx, deviceSN, httpURL); err != nil {
			d.recordMetrics("http", "failure", time.Since(start))
			return fmt.Errorf("http connection request: %w", err)
		}
		d.logger.Debug("http connection request sent",
			zap.String("device_sn", deviceSN))
		d.recordMetrics("http", "success", time.Since(start))
		return nil
	}

	// No method available — device will connect on next periodic Inform
	d.logger.Debug("no connection request method available, waiting for periodic inform",
		zap.String("device_sn", deviceSN))
	return ErrNoConnectionMethod
}

func (d *Dispatcher) recordMetrics(method, result string, duration time.Duration) {
	if d.metrics == nil {
		return
	}
	d.metrics.SentTotal.WithLabelValues(method, result).Inc()
	d.metrics.DurationSeconds.WithLabelValues(method).Observe(duration.Seconds())
}

// resolveLANPort returns (port, source). source is "device_param" when the
// per-device lookup yielded a positive port, otherwise "default" (the
// UDPSender will substitute lanUDPCRDefaultPort).
func (d *Dispatcher) resolveLANPort(ctx context.Context, deviceSN string) (int, string) {
	if d.lanPortLookup == nil {
		return 0, "default"
	}
	port, ok := d.lanPortLookup(ctx, deviceSN)
	if !ok || port <= 0 {
		return 0, "default"
	}
	return port, "device_param"
}

// effectiveLANPort mirrors the UDPSender fallback for log/metric clarity so
// 0 (caller didn't resolve) becomes the actual wire-level port (3478).
func effectiveLANPort(port int) int {
	if port <= 0 {
		return lanUDPCRDefaultPort
	}
	return port
}
