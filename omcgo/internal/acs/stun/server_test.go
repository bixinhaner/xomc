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

// waitForServerAddr polls the synchronized LocalAddr accessor until the
// UDP listener is ready, avoiding unsynchronized reads of srv.conn.
func waitForServerAddr(t *testing.T, srv *Server) *net.UDPAddr {
	t.Helper()
	var addr net.Addr
	require.Eventually(t, func() bool {
		addr = srv.LocalAddr()
		return addr != nil
	}, 2*time.Second, 10*time.Millisecond, "server should start listening")
	udpAddr, ok := addr.(*net.UDPAddr)
	require.True(t, ok, "local addr should be *net.UDPAddr")
	return udpAddr
}

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

	// Get the actual listening address (wait until the listener is ready)
	serverAddr := waitForServerAddr(t, srv)

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

	serverAddr := waitForServerAddr(t, srv)

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

func TestServer_StopWithoutStart(t *testing.T) {
	store := NewStore(nil, zap.NewNop())
	srv := NewServer(Config{ListenAddr: "127.0.0.1:0"}, store, zap.NewNop())

	// conn 尚未建立（nil），Stop 应安全返回不 panic
	assert.Nil(t, srv.LocalAddr())
	assert.NotPanics(t, func() {
		assert.NoError(t, srv.Stop())
	})
}

func TestServer_DefaultConfig(t *testing.T) {
	store := NewStore(nil, zap.NewNop())
	srv := NewServer(Config{}, store, zap.NewNop())

	assert.Equal(t, ":3478", srv.config.ListenAddr)
	assert.Greater(t, srv.config.WorkerSize, 0)
	assert.Equal(t, 2048, srv.config.BufferSize)
}
