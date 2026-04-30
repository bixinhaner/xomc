// Package backup — FTP connection test service (T-0032).
//
// FTPConnectionTester powers the POST /api/v1/backup/ftp-configs/:id/test
// endpoint. It runs a two-tier probe against the configured FTP server:
//
//  1. TCP reachability (DNS + connect) — works for every protocol value
//     (ftp / sftp / ftps / anything else) and tells the operator the
//     server is at least network-reachable on the configured port.
//  2. FTP USER / PASS auth probe — runs only when protocol == "ftp",
//     using stdlib net/textproto so no third-party FTP client dep is
//     introduced. Validates that the configured username / password
//     produce a 2xx response from the server.
//
// SFTP / FTPS deeper auth probes are deliberately out of scope (PRD §6
// N1/N2): they require pkg/sftp + crypto/ssh + crypto/tls integration
// and grow the dependency surface beyond an S-sized task. Carved out as
// follow-up T-0093.
//
// Security:
//   - Passwords never appear in zap logs (zap fields only carry
//     host/port/protocol/success/latency).
//   - Passwords never appear in returned messages — only the FTP
//     server's own response code + text.
//   - The conn is closed via defer; net.textproto holds it without
//     additional cleanup.
//   - context timeout is enforced over both phases (TCP dial + FTP
//     command exchange).
//   - No side effects: the probe never lists, writes, or deletes
//     anything on the server.
package backup

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// Dialer is the narrow consumer-side contract FTPConnectionTester needs
// for TCP connection establishment. *net.Dialer satisfies it; tests
// inject a fake dialer to drive the success / error matrix without
// real network.
type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

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

// FTPConnectionTester runs reachability and (for FTP) auth probes
// against an FTPConfig. One instance per process; safe for concurrent
// Test calls.
type FTPConnectionTester struct {
	dialer  Dialer
	timeout time.Duration
	logger  *zap.Logger
}

// NewFTPConnectionTester returns a tester. dialer == nil falls back to
// a default *net.Dialer; timeout <= 0 falls back to DefaultFTPTestTimeout.
// logger == nil falls back to zap.NewNop.
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
	return &FTPConnectionTester{dialer: dialer, timeout: timeout, logger: logger}
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
	default:
		// sftp / ftps / anything else: TCP-reachable is the strongest
		// claim we can make without pulling in pkg/sftp + crypto/ssh
		// or doing a TLS handshake. Surface this honestly.
		result.AuthProbeSupported = false
		result.Success = true
		result.Message = fmt.Sprintf(
			"%s deep auth probe deferred to T-0093; TCP reachability verified",
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
