package stun

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServer_StartStop(t *testing.T) {
	store := NewStore(nil, zap.NewNop())
	cfg := Config{
		ListenAddr: "127.0.0.1:0", // OS picks a free port
		WorkerSize: 2,
		BufferSize: 2048,
	}
	srv := NewServer(cfg, store, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	cancel()
	srv.Stop()

	err := <-errCh
	assert.NoError(t, err)
}

func TestServer_Integration_STUNBindingRequest(t *testing.T) {
	store := NewStore(nil, zap.NewNop())
	cfg := Config{
		ListenAddr: "127.0.0.1:0",
		WorkerSize: 1,
		BufferSize: 2048,
	}
	srv := NewServer(cfg, store, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()
	time.Sleep(50 * time.Millisecond)

	// Get the actual listening address
	serverAddr := srv.conn.LocalAddr().(*net.UDPAddr)

	// Send a STUN Binding Request from a client
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	require.NoError(t, err)
	defer clientConn.Close()

	txID := [12]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	reqData := buildMinimalSTUN(TypeBindingRequest, txID)

	_, err = clientConn.Write(reqData)
	require.NoError(t, err)

	// Read response
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	respBuf := make([]byte, 2048)
	n, err := clientConn.Read(respBuf)
	require.NoError(t, err)

	// Decode response
	resp, err := Decode(respBuf[:n])
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, TypeBindingResponse, resp.Type)
	assert.Equal(t, txID, resp.TransactionID)

	// Verify MAPPED-ADDRESS attribute exists
	require.NotEmpty(t, resp.Attributes)
	assert.Equal(t, AttrMappedAddress, resp.Attributes[0].Type)

	// Cleanup
	cancel()
	srv.Stop()
}

func TestServer_Integration_NonStandardENB(t *testing.T) {
	store := NewStore(nil, zap.NewNop())
	cfg := Config{
		ListenAddr: "127.0.0.1:0",
		WorkerSize: 1,
		BufferSize: 2048,
	}
	srv := NewServer(cfg, store, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()
	time.Sleep(50 * time.Millisecond)

	serverAddr := srv.conn.LocalAddr().(*net.UDPAddr)

	// Send eNB SN as non-standard packet
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	require.NoError(t, err)
	defer clientConn.Close()

	_, err = clientConn.Write([]byte("ENB_SN_ABC123"))
	require.NoError(t, err)

	// Read echoreply
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	respBuf := make([]byte, 2048)
	n, err := clientConn.Read(respBuf)
	require.NoError(t, err)
	assert.Equal(t, "echoreply", string(respBuf[:n]))

	// Verify address was stored
	info, err := store.Get(context.Background(), "ENB_SN_ABC123")
	require.NoError(t, err)
	require.NotNil(t, info)

	// Cleanup
	cancel()
	srv.Stop()
}

func TestServer_DefaultConfig(t *testing.T) {
	store := NewStore(nil, zap.NewNop())
	srv := NewServer(Config{}, store, zap.NewNop())

	assert.Equal(t, ":3478", srv.config.ListenAddr)
	assert.Greater(t, srv.config.WorkerSize, 0)
	assert.Equal(t, 2048, srv.config.BufferSize)
}
