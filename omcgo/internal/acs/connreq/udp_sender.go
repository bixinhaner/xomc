package connreq

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/omcgo/omcgo/internal/acs/stun"
	"go.uber.org/zap"
)

const (
	// udpRetries is the number of times to send each UDP Connection Request
	// (UDP is unreliable, so we send multiple copies).
	udpRetries = 3
	// enbRequestMessage is the non-standard message sent to eNB devices.
	enbRequestMessage = "infromrequest"
	// defaultUsername for CPE Connection Request HMAC signing.
	defaultUsername = "dps"
	// lanUDPCRDefaultPort is the device-side UDP CR receiver port for LAN-direct
	// devices that do NOT follow TR-069 Annex G STUN-binding convention.
	// Empirically verified on BAICELLS BSC7041C243: the receiver listens on the
	// device's LAN IP at this port (matches Device.ManagementServer.STUNServerPort
	// parameter default of 3478, but on a SEPARATE socket from STUN client).
	lanUDPCRDefaultPort = 3478
)

// ErrNoLANTarget is returned by SendLAN when httpURL cannot yield a usable LAN target.
var ErrNoLANTarget = errors.New("no usable LAN target from connection request url")

// UDPSender sends UDP Connection Request messages to devices whose
// public address was discovered via STUN.
type UDPSender struct {
	store        *stun.Store
	sharedSecret string
	logger       *zap.Logger
}

// NewUDPSender creates a new UDP Connection Request sender.
func NewUDPSender(store *stun.Store, sharedSecret string, logger *zap.Logger) *UDPSender {
	if sharedSecret == "" {
		sharedSecret = defaultUsername
	}
	return &UDPSender{
		store:        store,
		sharedSecret: sharedSecret,
		logger:       logger,
	}
}

// Send sends a UDP Connection Request to the device identified by deviceSN.
// Returns nil if the device has no known STUN address (caller should fall back).
// The isENB flag indicates whether the device is an eNB (base station) or CPE.
func (s *UDPSender) Send(ctx context.Context, deviceSN string, serverAddr string, isENB bool) error {
	info, err := s.store.Get(ctx, deviceSN)
	if err != nil {
		return fmt.Errorf("lookup stun address: %w", err)
	}
	if info == nil {
		return ErrNoSTUNAddress
	}

	addr := info.UDPAddr()

	var msg []byte
	if isENB {
		msg = []byte(enbRequestMessage)
	} else {
		msg = buildCPEConnectionRequest(serverAddr, s.sharedSecret)
	}

	return sendUDP(addr, msg, s.logger)
}

// sendUDP sends a UDP packet to the given address, retrying udpRetries times.
func sendUDP(addr *net.UDPAddr, data []byte, logger *zap.Logger) error {
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("dial udp %s: %w", addr, err)
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	var lastErr error
	for i := 0; i < udpRetries; i++ {
		_, err := conn.Write(data)
		if err != nil {
			lastErr = err
			logger.Warn("udp write failed",
				zap.String("dst", addr.String()),
				zap.Int("attempt", i+1),
				zap.Error(err))
			continue
		}
	}

	if lastErr != nil {
		return fmt.Errorf("udp send to %s failed: %w", addr, lastErr)
	}
	return nil
}

// SendLAN sends a UDP Connection Request directly to the device's LAN address.
//
// Target = host(parsed from httpURL) : port. The port comes from the caller
// (typically resolved from Device.ManagementServer.STUNServerPort device
// parameter); when port <= 0 we fall back to lanUDPCRDefaultPort (3478) — the
// value empirically used by BAICELLS BSC7041C243 / Dengyo BSC7079B243.
//
// This path is required for devices whose UDP CR receiver listens on a SEPARATE
// socket from the STUN client (so the STUN-cache NAT-mapped 5-tuple cannot reach
// the receiver). Only viable when the ACS host can directly route to the device
// LAN IP (same L2 segment or routed network).
func (s *UDPSender) SendLAN(ctx context.Context, deviceSN, httpURL string, port int, isENB bool, serverAddr string) error {
	addr, err := resolveLANUDPTarget(httpURL, port)
	if err != nil {
		return err
	}

	var msg []byte
	if isENB {
		msg = []byte(enbRequestMessage)
	} else {
		msg = buildCPEConnectionRequest(serverAddr, s.sharedSecret)
	}

	s.logger.Info("udp lan cr send",
		zap.String("device_sn", deviceSN),
		zap.String("dst", addr.String()),
		zap.Int("bytes", len(msg)))
	return sendUDP(addr, msg, s.logger)
}

// resolveLANUDPTarget parses httpURL and pairs the host with port (or the
// hard-coded default when port <= 0) to produce a UDP target. Extracted from
// SendLAN to allow address-resolution unit testing without a real socket.
//
// Returns ErrNoLANTarget when httpURL is empty or yields no usable hostname;
// returns a wrapped error when url.Parse or net.ResolveUDPAddr fails.
func resolveLANUDPTarget(httpURL string, port int) (*net.UDPAddr, error) {
	if httpURL == "" {
		return nil, ErrNoLANTarget
	}
	u, err := url.Parse(httpURL)
	if err != nil {
		return nil, fmt.Errorf("parse http url %q: %w", httpURL, err)
	}
	host := u.Hostname()
	if host == "" {
		return nil, ErrNoLANTarget
	}
	if port <= 0 {
		port = lanUDPCRDefaultPort
	}
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("resolve lan udp addr: %w", err)
	}
	return addr, nil
}

// SendRestart sends a UDP restart command to an unresponsive base station.
// Format: "/restart_{md5(deviceSN)}" — this is a last-resort mechanism
// for devices that do not respond to normal Connection Requests.
func (s *UDPSender) SendRestart(ctx context.Context, deviceSN string) error {
	info, err := s.store.Get(ctx, deviceSN)
	if err != nil {
		return fmt.Errorf("lookup stun address: %w", err)
	}
	if info == nil {
		return ErrNoSTUNAddress
	}

	hash := md5.Sum([]byte(deviceSN))
	msg := []byte("/restart_" + hex.EncodeToString(hash[:]))

	s.logger.Info("sending udp restart command",
		zap.String("device_sn", deviceSN),
		zap.String("addr", info.UDPAddr().String()))

	return sendUDP(info.UDPAddr(), msg, s.logger)
}

// buildCPEConnectionRequest constructs the HTTP-like UDP Connection Request
// message for CPE devices (TR-069 Annex G format).
//
// Format:
//
//	GET http://{serverAddr}/?ts={timestamp}&id={random}&un=dps&cn={nonce}&sig={hmac} HTTP/1.1\r\n\r\n
func buildCPEConnectionRequest(serverAddr, secret string) []byte {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	id := strconv.Itoa(rand.Intn(100000))
	un := defaultUsername
	cn := strconv.Itoa(rand.Intn(100000))

	// HMAC-SHA1 signature over ts+id+un+cn
	sigData := ts + id + un + cn
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(sigData))
	sig := hex.EncodeToString(mac.Sum(nil))

	msg := fmt.Sprintf(
		"GET http://%s/?ts=%s&id=%s&un=%s&cn=%s&sig=%s HTTP/1.1\r\n\r\n",
		serverAddr, ts, id, un, cn, sig,
	)
	return []byte(msg)
}
