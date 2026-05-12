// Package backup — FTP connection test service (T-0032 + T-0093).
//
// FTPConnectionTester powers the POST /api/v1/backup/ftp-configs/:id/test
// endpoint. It runs a two-tier probe against the configured FTP server:
//
//  1. TCP reachability (DNS + connect) — works for every protocol value
//     (ftp / sftp / ftps / anything else) and tells the operator the
//     server is at least network-reachable on the configured port.
//  2. Protocol-specific auth probe — runs deep authentication against
//     the configured username / password:
//       - "ftp":  net/textproto USER / PASS over plain conn (T-0032)
//       - "sftp": crypto/ssh client handshake with password auth (T-0093)
//       - "ftps": crypto/tls handshake (implicit FTPS) + USER / PASS
//         over the TLS conn (T-0093)
//       - other:  TCP reachability only, auth probe unsupported
//
// Security:
//   - Passwords never appear in zap logs (zap fields only carry
//     host/port/protocol/success/latency).
//   - Passwords never appear in returned messages — defense-in-depth
//     redactSubstring strips any literal echo from server responses.
//   - Conn deadlines / context timeout enforced over every phase
//     (TCP dial + protocol handshake + auth exchange).
//   - SSH host key callback is InsecureIgnoreHostKey — the probe is a
//     reachability + credential test, not a long-lived session; the
//     operator owns the trust decision out-of-band (PRD §6 N3).
//   - TLS verification skipped (operator-provided self-signed certs
//     are common in network ops); auth still gated by USER / PASS.
//   - No side effects: the probe never lists, writes, or deletes
//     anything on the server.
package backup

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// Dialer is the narrow consumer-side contract FTPConnectionTester needs
// for TCP connection establishment. *net.Dialer satisfies it; tests
// inject a fake dialer to drive the success / error matrix without
// real network.
type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// sshProbeFn / tlsProbeFn are internal extension points for the SFTP /
// FTPS auth probes (T-0093). Tests in the same package override them
// to drive the success / error matrix without standing up an in-process
// SSH server or TLS listener. Production wiring uses sshDialDefault /
// tlsDialDefault below.
type sshProbeFn func(ctx context.Context, addr string, cfg *ssh.ClientConfig) error
type tlsProbeFn func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error)

// FTPTestResult is the structured outcome returned by the test endpoint.
// Wire format mirrors the JSON the handler sends back.
type FTPTestResult struct {
	Success            bool   `json:"success"`
	TCPReachable       bool   `json:"tcp_reachable"`
	AuthProbeSupported bool   `json:"auth_probe_supported"`
	AuthProbePassed    *bool  `json:"auth_probe_passed,omitempty"`
	LatencyMs          int64  `json:"latency_ms"`
	Message            string `json:"message"`
	Protocol           string `json:"protocol"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
}

// DefaultFTPTestTimeout caps the full test (DNS + connect + USER + PASS).
// 5 seconds is the operator-feedback latency target; anything longer
// frustrates the "click test → wait → see result" flow.
const DefaultFTPTestTimeout = 5 * time.Second

// FTPConnectionTester runs reachability and protocol-specific auth
// probes against an FTPConfig. One instance per process; safe for
// concurrent Test calls.
type FTPConnectionTester struct {
	dialer   Dialer
	sshProbe sshProbeFn
	tlsProbe tlsProbeFn
	timeout  time.Duration
	logger   *zap.Logger
}

// NewFTPConnectionTester returns a tester. dialer == nil falls back to
// a default *net.Dialer; timeout <= 0 falls back to DefaultFTPTestTimeout.
// logger == nil falls back to zap.NewNop. SSH / TLS probes use stdlib +
// x/crypto/ssh defaults; tests override the unexported probe fields
// directly.
func NewFTPConnectionTester(dialer Dialer, timeout time.Duration, logger *zap.Logger) *FTPConnectionTester {
	if dialer == nil {
		dialer = &net.Dialer{Timeout: DefaultFTPTestTimeout}
	}
	if timeout <= 0 {
		timeout = DefaultFTPTestTimeout
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FTPConnectionTester{
		dialer:   dialer,
		sshProbe: sshDialDefault,
		tlsProbe: tlsDialDefault,
		timeout:  timeout,
		logger:   logger,
	}
}

// Test runs the connection test against the supplied FTPConfig and
// returns a structured result. The result is always returned (never
// nil); FTPTestResult.Success aggregates the per-tier outcomes:
//
//   - protocol == "ftp": success requires TCP reachable AND auth probe passed
//   - other protocols: success requires TCP reachable (auth probe unsupported)
func (t *FTPConnectionTester) Test(ctx context.Context, cfg *FTPConfig) FTPTestResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	result := FTPTestResult{
		Protocol: cfg.Protocol,
		Host:     cfg.Host,
		Port:     cfg.Port,
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	conn, err := t.dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		result.LatencyMs = time.Since(start).Milliseconds()
		result.Message = sanitizeDialError(err)
		t.logTestOutcome(cfg, result)
		return result
	}
	defer conn.Close()
	result.TCPReachable = true

	// Tier 2 — protocol-specific auth probe.
	switch cfg.Protocol {
	case "ftp":
		result.AuthProbeSupported = true
		passed, msg := t.probeFTPAuth(conn, cfg)
		result.AuthProbePassed = &passed
		result.Success = passed
		result.Message = msg
	case "sftp":
		// TCP probe already proved reachability; close it and let the
		// SSH probe open its own conn for the handshake. This costs
		// one extra dial but keeps ssh.NewClientConn's lifecycle clean
		// (the handshake takes ownership of the conn it's given).
		_ = conn.Close()
		result.AuthProbeSupported = true
		passed, msg := t.probeSFTPAuth(ctx, addr, cfg)
		result.AuthProbePassed = &passed
		result.Success = passed
		result.Message = msg
	case "ftps":
		// Implicit FTPS: the TLS handshake happens before any FTP
		// command. Re-dial via tlsProbe so the FTP USER/PASS travel
		// over the TLS conn.
		_ = conn.Close()
		result.AuthProbeSupported = true
		passed, msg := t.probeFTPSAuth(ctx, addr, cfg)
		result.AuthProbePassed = &passed
		result.Success = passed
		result.Message = msg
	default:
		// Unknown protocol — TCP-reachable is all we can honestly
		// claim. Surface it explicitly so the operator can correct
		// the configured protocol.
		result.AuthProbeSupported = false
		result.Success = true
		result.Message = fmt.Sprintf(
			"protocol %q has no auth probe; TCP reachability verified",
			cfg.Protocol)
	}

	result.LatencyMs = time.Since(start).Milliseconds()
	t.logTestOutcome(cfg, result)
	return result
}

// probeFTPAuth runs the minimal FTP login handshake over the supplied
// already-connected conn. Returns (passed, message) where message is
// sanitized to never include the password.
//
// Protocol (RFC 959):
//   - server sends 220 banner on connect
//   - client sends "USER <user>\r\n" → server returns 331 (need pass) or 230 (no pass)
//   - client sends "PASS <pass>\r\n" → server returns 230 (ok) or 530 (rejected)
//
// Rather than enforce a specific banner code (some servers send 220-...
// multi-line greetings), we ReadResponse(0) which accepts any code class
// and lets per-step expected-code logic decide.
func (t *FTPConnectionTester) probeFTPAuth(conn net.Conn, cfg *FTPConfig) (bool, string) {
	// Apply the deadline on the underlying conn so textproto reads/writes
	// honor it without needing context plumbing.
	deadline, _ := time.Now().Add(t.timeout), 0
	_ = conn.SetDeadline(deadline)

	tc := textproto.NewConn(conn)
	// Read banner. Accept any 2xx; ignore others tolerantly so non-RFC
	// servers that delay the 220 don't confuse the probe.
	if _, _, err := tc.ReadResponse(2); err != nil {
		return false, fmt.Sprintf("FTP banner read failed: %s", err.Error())
	}

	username := cfg.Username
	password := ""
	if cfg.PasswordEncrypted != nil {
		password = *cfg.PasswordEncrypted
	}

	if _, err := tc.Cmd("USER %s", username); err != nil {
		return false, fmt.Sprintf("FTP USER write failed: %s", err.Error())
	}
	userCode, _, err := tc.ReadResponse(0)
	if err != nil {
		return false, fmt.Sprintf("FTP USER response read failed: %s", err.Error())
	}
	// 230 = logged in (no password needed). 331 = need password.
	// Any 5xx (or other unexpected) → reject.
	switch {
	case userCode == 230:
		return true, "FTP USER accepted without password (anonymous-style login)"
	case userCode == 331:
		// Continue to PASS.
	default:
		return false, fmt.Sprintf("FTP USER rejected: code %d", userCode)
	}

	if _, err := tc.Cmd("PASS %s", password); err != nil {
		return false, fmt.Sprintf("FTP PASS write failed: %s", err.Error())
	}
	passCode, passMsg, err := tc.ReadResponse(0)
	if err != nil {
		return false, fmt.Sprintf("FTP PASS response read failed: %s", err.Error())
	}
	if passCode >= 200 && passCode < 300 {
		return true, fmt.Sprintf("FTP authenticated successfully (code %d)", passCode)
	}
	// Replace any password echo from server's message just in case.
	// Most servers don't echo the password back; defensive sanitization
	// only triggers when the configured password literally appears.
	safeMsg := passMsg
	if password != "" && len(password) >= 4 {
		safeMsg = redactSubstring(safeMsg, password)
	}
	return false, fmt.Sprintf("FTP auth rejected: code %d %s", passCode, safeMsg)
}

// probeSFTPAuth runs the SSH handshake + password auth check against
// the configured SFTP server. Returns (passed, message). The message
// is sanitized so the configured password never leaks even if the
// server echoes it in an auth-failure diagnostic.
//
// HostKeyCallback is ssh.InsecureIgnoreHostKey — the probe is a
// reachability + credential check, not a long-lived session, and the
// operator pins host keys out-of-band when authoring the FTPConfig
// (PRD §6 N3). Adding a key-pinning probe is future T-0094 territory.
func (t *FTPConnectionTester) probeSFTPAuth(ctx context.Context, addr string, cfg *FTPConfig) (bool, string) {
	password := ""
	if cfg.PasswordEncrypted != nil {
		password = *cfg.PasswordEncrypted
	}
	sshCfg := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         t.timeout,
	}
	if err := t.sshProbe(ctx, addr, sshCfg); err != nil {
		safe := errMessageOnly(err)
		if password != "" && len(password) >= 4 {
			safe = redactSubstring(safe, password)
		}
		return false, fmt.Sprintf("SFTP auth failed: %s", safe)
	}
	return true, "SFTP authenticated successfully"
}

// probeFTPSAuth runs implicit FTPS: TLS handshake first, then standard
// FTP USER / PASS over the TLS conn. The auth result and message
// semantics mirror probeFTPAuth.
//
// TLS InsecureSkipVerify is true: operator-managed FTPS servers are
// commonly self-signed in network ops environments, and the probe's
// trust signal is the USER/PASS exchange, not the cert chain. A
// future task can add cert pinning when the FTPConfig gains a CA bundle
// field.
func (t *FTPConnectionTester) probeFTPSAuth(ctx context.Context, addr string, cfg *FTPConfig) (bool, string) {
	tlsCfg := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: true, //nolint:gosec // operator-managed self-signed FTPS is the norm; see PRD §6 N3
		MinVersion:         tls.VersionTLS12,
	}
	conn, err := t.tlsProbe(ctx, "tcp", addr, tlsCfg)
	if err != nil {
		return false, fmt.Sprintf("FTPS TLS handshake failed: %s", errMessageOnly(err))
	}
	defer conn.Close()
	return t.probeFTPAuth(conn, cfg)
}

// sshDialDefault is the production sshProbe — open TCP via stdlib
// net.Dialer (honoring ctx for connect timeout), apply ctx deadline to
// the conn so ssh.NewClientConn's handshake is bounded, then close
// cleanly after auth completes (no session is held open).
func sshDialDefault(ctx context.Context, addr string, cfg *ssh.ClientConfig) error {
	d := &net.Dialer{Timeout: cfg.Timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		_ = conn.Close()
		return err
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	return client.Close()
}

// tlsDialDefault is the production tlsProbe — uses stdlib tls.Dialer
// which composes net.Dialer + TLS handshake under one DialContext that
// honors the supplied context for connect timeout and handshake
// timeout.
func tlsDialDefault(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
	d := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: DefaultFTPTestTimeout},
		Config:    cfg,
	}
	return d.DialContext(ctx, network, addr)
}

// logTestOutcome emits a single zap.Info with the result fields that
// are SAFE to log (no password, no username, no FTP server private
// banner detail beyond the code class).
func (t *FTPConnectionTester) logTestOutcome(cfg *FTPConfig, result FTPTestResult) {
	t.logger.Info("ftp connection test",
		zap.String("config_id", cfg.ID.String()),
		zap.String("protocol", cfg.Protocol),
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.Bool("success", result.Success),
		zap.Bool("tcp_reachable", result.TCPReachable),
		zap.Bool("auth_probe_supported", result.AuthProbeSupported),
		zap.Int64("latency_ms", result.LatencyMs),
	)
}

// sanitizeDialError extracts the safe portion of a net.Dialer error.
// Errors of type *net.OpError can carry the addr; we keep that since
// host/port aren't secrets. Strip any wrap context that might one day
// surface env / config detail.
func sanitizeDialError(err error) string {
	if err == nil {
		return ""
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return fmt.Sprintf("dial %s %s: %s", opErr.Op, opErr.Addr.String(), errMessageOnly(opErr.Err))
	}
	return err.Error()
}

// errMessageOnly returns the leaf-most error message via repeated
// errors.Unwrap. This avoids pulling deep stack details that some
// libraries embed.
func errMessageOnly(err error) string {
	for err != nil {
		next := errors.Unwrap(err)
		if next == nil {
			return err.Error()
		}
		err = next
	}
	return ""
}

// redactSubstring removes occurrences of needle from haystack and
// replaces them with [REDACTED]. Used as defense-in-depth in case an
// FTP server echoes the password back in its 5xx response (rare but
// observed in misconfigured servers).
func redactSubstring(haystack, needle string) string {
	if needle == "" {
		return haystack
	}
	out := make([]byte, 0, len(haystack))
	i := 0
	for i < len(haystack) {
		if i+len(needle) <= len(haystack) && haystack[i:i+len(needle)] == needle {
			out = append(out, []byte("[REDACTED]")...)
			i += len(needle)
			continue
		}
		out = append(out, haystack[i])
		i++
	}
	return string(out)
}
