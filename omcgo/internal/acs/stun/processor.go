package stun

import (
	"context"
	"net"
	"strings"

	"go.uber.org/zap"
)

// Processor handles incoming STUN (and non-standard) UDP packets,
// extracts device public addresses, and stores them for Connection Request use.
type Processor struct {
	store  *Store
	logger *zap.Logger
}

// NewProcessor creates a new STUN message processor.
func NewProcessor(store *Store, logger *zap.Logger) *Processor {
	return &Processor{
		store:  store,
		logger: logger,
	}
}

// HandlePacket dispatches an incoming UDP packet to the appropriate handler
// based on whether it is a standard STUN message or a non-standard (eNB) message.
// Returns the response bytes to send back (may be nil if no response needed).
func (p *Processor) HandlePacket(ctx context.Context, data []byte, addr *net.UDPAddr) []byte {
	if IsSTUN(data) {
		return p.handleSTUN(ctx, data, addr)
	}
	return p.handleNonStandard(ctx, data, addr)
}

// handleSTUN processes a standard STUN Binding Request from a CPE device.
//
// Flow:
//  1. Decode the STUN message
//  2. Extract the device's public IP:Port from the UDP source address
//  3. Store the address in the STUN store (keyed by source address for now;
//     the actual device SN mapping happens later when the Inform arrives)
//  4. Construct and return a Binding Response with MAPPED-ADDRESS
func (p *Processor) handleSTUN(ctx context.Context, data []byte, addr *net.UDPAddr) []byte {
	msg, err := Decode(data)
	if err != nil {
		p.logger.Warn("stun decode error",
			zap.String("src", addr.String()),
			zap.Error(err))
		return nil
	}
	if msg == nil {
		// Not a valid STUN message despite passing IsSTUN check
		return nil
	}

	if !msg.IsBindingRequest() {
		p.logger.Debug("stun non-binding-request ignored",
			zap.String("type", TypeString(msg.Type)),
			zap.String("src", addr.String()))
		return nil
	}

	// Store the device's public address. We use the source address string as a
	// temporary key. The proper device SN will be associated when the CPE sends
	// its Inform containing UDPConnectionRequestAddress.
	addrKey := addr.String()
	if err := p.store.SetFromUDPAddr(ctx, addrKey, addr); err != nil {
		p.logger.Warn("store stun address failed",
			zap.String("src", addr.String()),
			zap.Error(err))
	}

	p.logger.Debug("stun binding request processed",
		zap.String("src", addr.String()))

	// Determine response destination (prefer RESPONSE-ADDRESS if present)
	respAddr := addr
	if ra := msg.ResponseAddress(); ra != nil {
		respAddr = ra
	}
	_ = respAddr // response is sent to the same addr by the server; respAddr is noted for future use

	// Build and encode Binding Response
	resp := NewBindingResponse(msg.TransactionID, addr)
	return resp.Encode()
}

// handleNonStandard processes a non-standard UDP packet from an eNB (base station).
//
// Protocol: eNB sends its serial number as UTF-8 text; server replies "echoreply".
//
// Flow:
//  1. Parse the SN from the packet body (UTF-8 text)
//  2. Store the eNB's public address keyed by SN
//  3. Return "echoreply" as confirmation
func (p *Processor) handleNonStandard(ctx context.Context, data []byte, addr *net.UDPAddr) []byte {
	sn := strings.TrimSpace(string(data))
	if sn == "" {
		return nil
	}

	if err := p.store.SetFromUDPAddr(ctx, sn, addr); err != nil {
		p.logger.Warn("store enb stun address failed",
			zap.String("sn", sn),
			zap.String("src", addr.String()),
			zap.Error(err))
	}

	p.logger.Debug("enb stun packet processed",
		zap.String("sn", sn),
		zap.String("src", addr.String()))

	return []byte("echoreply")
}
