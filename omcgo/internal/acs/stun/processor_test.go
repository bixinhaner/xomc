package stun

import (
	"context"
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestProcessor(t *testing.T) (*Processor, *Store) {
	t.Helper()
	store := NewStore(nil, zap.NewNop()) // L1-only for tests
	proc := NewProcessor(store, zap.NewNop())
	return proc, store
}

func TestProcessor_HandlePacket_STUNBindingRequest(t *testing.T) {
	proc, store := newTestProcessor(t)
	ctx := context.Background()

	txID := [12]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
	data := buildMinimalSTUN(TypeBindingRequest, txID)
	addr := &net.UDPAddr{IP: net.IPv4(203, 0, 113, 50), Port: 5000}

	resp := proc.HandlePacket(ctx, data, addr)
	require.NotNil(t, resp)

	// Verify response is a valid Binding Response
	msg, err := Decode(resp)
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, TypeBindingResponse, msg.Type)
	assert.Equal(t, txID, msg.TransactionID)

	// Verify MAPPED-ADDRESS contains the source address
	require.Len(t, msg.Attributes, 1)
	mapped := decodeAddressAttr(msg.Attributes[0].Value)
	require.NotNil(t, mapped)
	assert.Equal(t, addr.IP.To4(), mapped.IP.To4())
	assert.Equal(t, addr.Port, mapped.Port)

	// Verify address was stored
	stored, err := store.Get(ctx, addr.String())
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "203.0.113.50", stored.IP)
	assert.Equal(t, 5000, stored.Port)
}

func TestProcessor_HandlePacket_NonStandard_ENB(t *testing.T) {
	proc, store := newTestProcessor(t)
	ctx := context.Background()

	sn := "ENB_SN_12345"
	data := []byte(sn)
	addr := &net.UDPAddr{IP: net.IPv4(10, 20, 30, 40), Port: 6000}

	resp := proc.HandlePacket(ctx, data, addr)
	require.NotNil(t, resp)
	assert.Equal(t, "echoreply", string(resp))

	// Verify address was stored with the SN as key
	stored, err := store.Get(ctx, sn)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "10.20.30.40", stored.IP)
	assert.Equal(t, 6000, stored.Port)
}

func TestProcessor_HandlePacket_NonStandard_Empty(t *testing.T) {
	proc, _ := newTestProcessor(t)
	ctx := context.Background()

	resp := proc.HandlePacket(ctx, []byte(""), &net.UDPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1})
	assert.Nil(t, resp)
}

func TestProcessor_HandlePacket_NonStandard_Whitespace(t *testing.T) {
	proc, store := newTestProcessor(t)
	ctx := context.Background()

	resp := proc.HandlePacket(ctx, []byte("  ENB_SN_99  "), &net.UDPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1})
	assert.Equal(t, "echoreply", string(resp))

	stored, _ := store.Get(ctx, "ENB_SN_99")
	require.NotNil(t, stored)
}

func TestProcessor_HandlePacket_STUNNonBinding(t *testing.T) {
	proc, _ := newTestProcessor(t)
	ctx := context.Background()

	// Send a Binding Response (not a request) - should be ignored
	data := buildMinimalSTUN(TypeBindingResponse, [12]byte{})
	addr := &net.UDPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1}

	resp := proc.HandlePacket(ctx, data, addr)
	assert.Nil(t, resp)
}

func TestProcessor_HandlePacket_STUNInvalid(t *testing.T) {
	proc, _ := newTestProcessor(t)
	ctx := context.Background()

	// Valid STUN header but truncated attribute
	data := buildMinimalSTUN(TypeBindingRequest, [12]byte{})
	binary.BigEndian.PutUint16(data[2:4], 100) // claim 100 bytes of attrs
	addr := &net.UDPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1}

	resp := proc.HandlePacket(ctx, data, addr)
	assert.Nil(t, resp)
}
