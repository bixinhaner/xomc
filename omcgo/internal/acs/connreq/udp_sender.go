package connreq

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
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
)

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
