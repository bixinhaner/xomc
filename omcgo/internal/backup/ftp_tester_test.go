package backup

import (
	"bufio"
	stdbytes "bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// =============================================================================
// fakeDialer + fakeFTPConn — drive Dialer interface decisions without real
// network. fakeFTPConn replays a scripted server side of the FTP USER/PASS
// exchange via an in-memory pipe so net/textproto reads/writes
// transparently.
// =============================================================================

type fakeDialer struct {
	dialErr error    // when set, DialContext returns it directly
	conn    net.Conn // when nil + dialErr nil, returns a fresh fakeFTPConn
	script  []string // server-side script lines for fakeFTPConn (CRLF-terminated)
}

func (d *fakeDialer) DialContext(_ context.Context, _, _ string) (net.Conn, error) {
	if d.dialErr != nil {
		return nil, d.dialErr
	}
	if d.conn != nil {
		return d.conn, nil
	}
	return newFakeFTPConn(d.script), nil
}

// fakeFTPConn implements net.Conn using two in-memory pipes:
//   - serverToClient: pre-filled with the scripted FTP server responses
//   - clientToServer: captures USER/PASS lines the SUT writes (for assertions)
//
// Read on the conn returns server lines; Write captures client lines.
type fakeFTPConn struct {
	server *bufio.Reader    // SUT reads from here
	client *stdbytes.Buffer // SUT writes here (assertable)
	mu     sync.Mutex
	closed bool
}

// newFakeFTPConn constructs a fake conn pre-loaded with the supplied
// scripted server-side response lines. Each line should be a complete
// FTP response (e.g. "220 Service ready" or "230 Login successful")
// without trailing CRLF — newFakeFTPConn appends "\r\n" to each.
func newFakeFTPConn(script []string) *fakeFTPConn {
	var serverBuf strings.Builder
	for _, line := range script {
		serverBuf.WriteString(line)
		serverBuf.WriteString("\r\n")
	}
	return &fakeFTPConn{
		server: bufio.NewReader(strings.NewReader(serverBuf.String())),
		client: &stdbytes.Buffer{},
	}
}

func (c *fakeFTPConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return 0, io.EOF
	}
	return c.server.Read(p)
}

func (c *fakeFTPConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, errors.New("write after close")
	}
	return c.client.Write(p)
}

func (c *fakeFTPConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *fakeFTPConn) ClientWritten() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client.String()
}

// Stub net.Addr / deadline methods — net.Conn requires them but we don't
// exercise the values.

type fakeAddr struct{ s string }

func (a fakeAddr) Network() string { return "tcp" }
func (a fakeAddr) String() string  { return a.s }

func (c *fakeFTPConn) LocalAddr() net.Addr                { return fakeAddr{"127.0.0.1:50000"} }
func (c *fakeFTPConn) RemoteAddr() net.Addr               { return fakeAddr{"server:21"} }
func (c *fakeFTPConn) SetDeadline(_ time.Time) error      { return nil }
func (c *fakeFTPConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *fakeFTPConn) SetWriteDeadline(_ time.Time) error { return nil }

// =============================================================================
// V1-V9 tests
// =============================================================================

func makeFTPCfg(t *testing.T, protocol, host string, port int, user, pass string) *FTPConfig {
	t.Helper()
	cfg := &FTPConfig{
		ID:       uuid.New(),
		Host:     host,
		Port:     port,
		Username: user,
		Protocol: protocol,
	}
	if pass != "" {
		p := pass
		cfg.PasswordEncrypted = &p
	}
	return cfg
}

// V1 — FTP protocol + auth success (USER → 331, PASS → 230)
func TestFTPConnectionTester_V1_FTPAuthSuccess(t *testing.T) {
	dialer := &fakeDialer{
		script: []string{
			"220 (vsFTPd 3.0.5)",
			"331 Please specify the password.",
			"230 Login successful.",
		},
	}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	cfg := makeFTPCfg(t, "ftp", "ftp.example.com", 21, "user", "secret")

	result := tester.Test(context.Background(), cfg)
	assert.True(t, result.Success)
	assert.True(t, result.TCPReachable)
	assert.True(t, result.AuthProbeSupported)
	require.NotNil(t, result.AuthProbePassed)
	assert.True(t, *result.AuthProbePassed)
	assert.Contains(t, result.Message, "230")
	assert.Equal(t, "ftp", result.Protocol)
	assert.Equal(t, "ftp.example.com", result.Host)
	assert.Equal(t, 21, result.Port)
}

// V2 — FTP auth rejected (USER → 331, PASS → 530)
func TestFTPConnectionTester_V2_FTPAuthRejected(t *testing.T) {
	dialer := &fakeDialer{
		script: []string{
			"220 (vsFTPd 3.0.5)",
			"331 Please specify the password.",
			"530 Login incorrect.",
		},
	}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	cfg := makeFTPCfg(t, "ftp", "host", 21, "user", "wrong-pass")

	result := tester.Test(context.Background(), cfg)
	assert.False(t, result.Success)
	assert.True(t, result.TCPReachable)
	assert.True(t, result.AuthProbeSupported)
	require.NotNil(t, result.AuthProbePassed)
	assert.False(t, *result.AuthProbePassed)
	assert.Contains(t, result.Message, "530")
}

// V3 — TCP unreachable (dial returns error)
func TestFTPConnectionTester_V3_TCPUnreachable(t *testing.T) {
	dialer := &fakeDialer{
		dialErr: &net.OpError{
			Op:   "dial",
			Net:  "tcp",
			Addr: fakeAddr{"10.0.0.1:21"},
			Err:  errors.New("connection refused"),
		},
	}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	cfg := makeFTPCfg(t, "ftp", "10.0.0.1", 21, "user", "pass")

	result := tester.Test(context.Background(), cfg)
	assert.False(t, result.Success)
	assert.False(t, result.TCPReachable)
	assert.False(t, result.AuthProbeSupported)
	assert.Nil(t, result.AuthProbePassed)
	assert.Contains(t, result.Message, "connection refused")
}

// V4 — DNS resolve failure surfaced through the dialer error
func TestFTPConnectionTester_V4_DNSFail(t *testing.T) {
	dialer := &fakeDialer{
		dialErr: errors.New("dial tcp: lookup nope.invalid: no such host"),
	}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	cfg := makeFTPCfg(t, "ftp", "nope.invalid", 21, "user", "pass")

	result := tester.Test(context.Background(), cfg)
	assert.False(t, result.Success)
	assert.False(t, result.TCPReachable)
	assert.Contains(t, result.Message, "no such host")
}

// V5 — protocol=sftp auth success (SSH handshake + password OK)
func TestFTPConnectionTester_V5_SFTPAuthSuccess(t *testing.T) {
	dialer := &fakeDialer{}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	var capturedUser, capturedAddr string
	tester.sshProbe = func(_ context.Context, addr string, cfg *ssh.ClientConfig) error {
		capturedAddr = addr
		capturedUser = cfg.User
		return nil
	}
	cfg := makeFTPCfg(t, "sftp", "host", 22, "user", "pass")

	result := tester.Test(context.Background(), cfg)
	assert.True(t, result.Success)
	assert.True(t, result.TCPReachable)
	assert.True(t, result.AuthProbeSupported)
	require.NotNil(t, result.AuthProbePassed)
	assert.True(t, *result.AuthProbePassed)
	assert.Contains(t, result.Message, "SFTP authenticated successfully")
	assert.Equal(t, "host:22", capturedAddr)
	assert.Equal(t, "user", capturedUser)
}

// V5b — protocol=sftp auth rejected (wrong password)
func TestFTPConnectionTester_V5b_SFTPAuthRejected(t *testing.T) {
	dialer := &fakeDialer{}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	tester.sshProbe = func(_ context.Context, _ string, _ *ssh.ClientConfig) error {
		return errors.New("ssh: unable to authenticate, attempted methods [none password], no supported methods remain")
	}
	cfg := makeFTPCfg(t, "sftp", "host", 22, "user", "wrong")

	result := tester.Test(context.Background(), cfg)
	assert.False(t, result.Success)
	assert.True(t, result.TCPReachable)
	assert.True(t, result.AuthProbeSupported)
	require.NotNil(t, result.AuthProbePassed)
	assert.False(t, *result.AuthProbePassed)
	assert.Contains(t, result.Message, "SFTP auth failed")
	assert.Contains(t, result.Message, "unable to authenticate")
}

// V5c — SFTP error message must redact the literal password
// even when the SSH server echoes it back (defense-in-depth).
func TestFTPConnectionTester_V5c_SFTPPasswordNotEchoed(t *testing.T) {
	password := "S3cr3tPwd!"
	dialer := &fakeDialer{}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	tester.sshProbe = func(_ context.Context, _ string, _ *ssh.ClientConfig) error {
		return errors.New("ssh: handshake failed: password " + password + " rejected")
	}
	cfg := makeFTPCfg(t, "sftp", "host", 22, "user", password)

	result := tester.Test(context.Background(), cfg)
	assert.False(t, result.Success)
	assert.NotContains(t, result.Message, password,
		"password literal must not appear in SFTP error message")
	assert.Contains(t, result.Message, "[REDACTED]")
}

// V6 — protocol=ftps auth success (TLS handshake + USER/PASS OK)
func TestFTPConnectionTester_V6_FTPSAuthSuccess(t *testing.T) {
	dialer := &fakeDialer{}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	var capturedTLSCfg *tls.Config
	tester.tlsProbe = func(_ context.Context, _, _ string, cfg *tls.Config) (net.Conn, error) {
		capturedTLSCfg = cfg
		// Hand back a fake FTP conn pre-loaded with the scripted server
		// responses — probeFTPAuth runs USER/PASS on this conn just as
		// it does for plain FTP.
		return newFakeFTPConn([]string{
			"220 (ProFTPD 1.3.5)",
			"331 Password required",
			"230 User logged in",
		}), nil
	}
	cfg := makeFTPCfg(t, "ftps", "host", 990, "user", "secret")

	result := tester.Test(context.Background(), cfg)
	assert.True(t, result.Success)
	assert.True(t, result.TCPReachable)
	assert.True(t, result.AuthProbeSupported)
	require.NotNil(t, result.AuthProbePassed)
	assert.True(t, *result.AuthProbePassed)
	assert.Contains(t, result.Message, "230")
	require.NotNil(t, capturedTLSCfg, "tlsProbe must be called for ftps")
	assert.Equal(t, "host", capturedTLSCfg.ServerName, "ServerName drives SNI")
	assert.GreaterOrEqual(t, capturedTLSCfg.MinVersion, uint16(tls.VersionTLS12),
		"TLS 1.2 floor required")
}

// V6b — protocol=ftps TLS handshake failure (e.g. cert / version mismatch)
func TestFTPConnectionTester_V6b_FTPSTLSHandshakeFail(t *testing.T) {
	dialer := &fakeDialer{}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	tester.tlsProbe = func(_ context.Context, _, _ string, _ *tls.Config) (net.Conn, error) {
		return nil, errors.New("tls: server selected unsupported protocol version 301")
	}
	cfg := makeFTPCfg(t, "ftps", "host", 990, "user", "pass")

	result := tester.Test(context.Background(), cfg)
	assert.False(t, result.Success)
	assert.True(t, result.TCPReachable, "TCP probe succeeded before TLS")
	assert.True(t, result.AuthProbeSupported)
	require.NotNil(t, result.AuthProbePassed)
	assert.False(t, *result.AuthProbePassed)
	assert.Contains(t, result.Message, "FTPS TLS handshake failed")
	assert.Contains(t, result.Message, "unsupported protocol version")
}

// V6c — protocol=ftps TLS OK but FTP auth rejected post-TLS
func TestFTPConnectionTester_V6c_FTPSAuthRejected(t *testing.T) {
	dialer := &fakeDialer{}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	tester.tlsProbe = func(_ context.Context, _, _ string, _ *tls.Config) (net.Conn, error) {
		return newFakeFTPConn([]string{
			"220 Service ready",
			"331 Password required",
			"530 Login failed",
		}), nil
	}
	cfg := makeFTPCfg(t, "ftps", "host", 990, "user", "wrong")

	result := tester.Test(context.Background(), cfg)
	assert.False(t, result.Success)
	assert.True(t, result.TCPReachable)
	require.NotNil(t, result.AuthProbePassed)
	assert.False(t, *result.AuthProbePassed)
	assert.Contains(t, result.Message, "530")
}

// V6d — unknown protocol falls through to TCP-only success path
func TestFTPConnectionTester_V6d_UnknownProtocol(t *testing.T) {
	dialer := &fakeDialer{}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	cfg := makeFTPCfg(t, "https", "host", 443, "user", "pass")

	result := tester.Test(context.Background(), cfg)
	assert.True(t, result.Success, "tcp reachable + auth probe unsupported → success")
	assert.True(t, result.TCPReachable)
	assert.False(t, result.AuthProbeSupported)
	assert.Nil(t, result.AuthProbePassed)
	assert.Contains(t, result.Message, "no auth probe")
	assert.Contains(t, result.Message, "https")
}

// V7 — context timeout enforced via dialer's ctx
func TestFTPConnectionTester_V7_ContextTimeout(t *testing.T) {
	dialer := &fakeDialer{
		dialErr: context.DeadlineExceeded,
	}
	tester := NewFTPConnectionTester(dialer, 10*time.Millisecond, zap.NewNop())
	cfg := makeFTPCfg(t, "ftp", "slow.host", 21, "user", "pass")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	result := tester.Test(ctx, cfg)
	assert.False(t, result.Success)
	assert.False(t, result.TCPReachable)
	assert.Contains(t, result.Message, "context deadline exceeded")
}

// V9 — security: configured password never appears in result.Message,
// even when echoed by a misbehaving server in the 530 response.
func TestFTPConnectionTester_V9_PasswordNotEchoed(t *testing.T) {
	password := "S3cr3tPwd!"
	dialer := &fakeDialer{
		script: []string{
			"220 Service ready",
			"331 Please specify the password.",
			"530 Login failed for password " + password, // misbehaving echo
		},
	}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	cfg := makeFTPCfg(t, "ftp", "host", 21, "user", password)

	result := tester.Test(context.Background(), cfg)
	assert.False(t, result.Success)
	// Defense-in-depth: redactSubstring should have removed the literal
	// password from the message. "[REDACTED]" should appear instead.
	assert.NotContains(t, result.Message, password,
		"password literal must not appear in result message")
	assert.Contains(t, result.Message, "[REDACTED]")
}

// V10 — Empty PasswordEncrypted (anonymous-style path) — USER returns 230
// directly, no PASS sent.
func TestFTPConnectionTester_V10_AnonymousLogin(t *testing.T) {
	dialer := &fakeDialer{
		script: []string{
			"220 Service ready",
			"230 Anonymous user logged in",
		},
	}
	tester := NewFTPConnectionTester(dialer, time.Second, zap.NewNop())
	cfg := makeFTPCfg(t, "ftp", "host", 21, "anonymous", "")

	result := tester.Test(context.Background(), cfg)
	assert.True(t, result.Success)
	assert.True(t, result.TCPReachable)
	require.NotNil(t, result.AuthProbePassed)
	assert.True(t, *result.AuthProbePassed)
	assert.Contains(t, result.Message, "without password")
}

// V_RedactSubstring — unit test for the redactSubstring helper used by
// the FTP auth probe to defense-in-depth strip echoed passwords.
func TestRedactSubstring(t *testing.T) {
	cases := []struct {
		haystack string
		needle   string
		want     string
	}{
		{"hello world", "world", "hello [REDACTED]"},
		{"abc abc abc", "abc", "[REDACTED] [REDACTED] [REDACTED]"},
		{"no match", "xyz", "no match"},
		{"empty needle", "", "empty needle"},
		{"", "abc", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.haystack, func(t *testing.T) {
			got := redactSubstring(c.haystack, c.needle)
			assert.Equal(t, c.want, got)
		})
	}
}
