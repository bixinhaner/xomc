// Package syslog — remote syslog forwarder (T-0028 / R-105).
//
// Forwarder sends SystemLog entries to a remote syslog server over UDP
// or TCP using RFC 3164 (BSD format) wire encoding. The format is the
// most widely-supported by collectors (Splunk, Logstash, rsyslog,
// syslog-ng) and avoids RFC 5424's structured-data complexity.
//
// Wire format:
//
//	<priority>timestamp hostname tag: message
//
//	priority = facility * 8 + severity
//	timestamp = "Jan _2 15:04:05" (Mmm dd hh:mm:ss, no year per RFC 3164)
//	hostname = configured local hostname (no FQDN normalization)
//	tag     = configured app name (default "omcgo")
//	message = SystemLog.Message (newlines replaced by spaces)
//
// Operator integration: the producer calls Forwarder.Send(ctx, log) when
// a SystemLog is created. Auto-fanout from Service.LogSystem is *NOT*
// wired in this task — the consumer integration is left to follow-up
// (avoids changing 7-test service.go contract surface). Operators who
// want auto-forward can call Forwarder.Send from their own SystemLog
// producer site.
package syslog

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ProtocolUDP / ProtocolTCP are the two transports T-0028 supports.
// TLS framing (RFC 6587) is out of scope; carve to T-0094 if needed.
const (
	ProtocolUDP = "udp"
	ProtocolTCP = "tcp"

	// DefaultFacility maps to "local0" per RFC 3164. Operators tune
	// via ForwarderConfig if their collector requires a specific
	// facility code.
	DefaultFacility = 16

	// DefaultDialTimeout caps the synchronous dial; persistent TCP
	// reconnect uses the same timeout for each retry.
	DefaultDialTimeout = 5 * time.Second
)

// ForwarderConfig configures Forwarder construction. Endpoint and
// Protocol are required; other fields fall back to documented defaults.
type ForwarderConfig struct {
	Endpoint    string        // "host:port" — required
	Protocol    string        // "udp" or "tcp" — required
	Facility    int           // 0..23; default 16 (local0)
	Hostname    string        // wire hostname; default os.Hostname() at construction
	AppName     string        // wire tag; default "omcgo"
	DialTimeout time.Duration // 0 → DefaultDialTimeout
	Logger      *zap.Logger   // nil → zap.NewNop()
}

// Forwarder sends SystemLog entries over UDP or TCP.
//
// UDP is stateless: each Send dials, writes, closes. Loss-tolerant by
// design — UDP doesn't acknowledge delivery, the wire is fire-and-forget.
//
// TCP is stateful: a single connection is held and reused; reconnect
// happens lazily on Send failure. Mutex protects concurrent Send calls
// against torn writes.
type Forwarder struct {
	cfg ForwarderConfig

	// TCP state
	mu       sync.Mutex
	tcpConn  net.Conn
	tcpAlive bool
}

// NewForwarder validates cfg and returns a Forwarder. Defaults applied:
//   - Facility 0 → DefaultFacility (16 / local0)
//   - Hostname "" → "localhost"
//   - AppName "" → "omcgo"
//   - DialTimeout 0 → DefaultDialTimeout
//   - Logger nil → zap.NewNop()
//
// Returns error when Endpoint is empty or Protocol is unsupported.
func NewForwarder(cfg ForwarderConfig) (*Forwarder, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("syslog forwarder: endpoint is required")
	}
	switch cfg.Protocol {
	case ProtocolUDP, ProtocolTCP:
		// OK
	default:
		return nil, fmt.Errorf("syslog forwarder: unsupported protocol %q (want udp|tcp)", cfg.Protocol)
	}
	// Facility 0 is technically valid (RFC 3164 "kernel"), but operator
	// misconfiguration most commonly looks like "I left Facility unset"
	// → zero value. Treat 0 as "use default local0". Userland apps
	// almost never want facility 0 / kernel; the few who do can patch
	// this check or set Facility=16 explicitly. Range outside 0..23
	// also falls back to local0.
	if cfg.Facility <= 0 || cfg.Facility > 23 {
		cfg.Facility = DefaultFacility
	}
	if cfg.Hostname == "" {
		cfg.Hostname = "localhost"
	}
	if cfg.AppName == "" {
		cfg.AppName = "omcgo"
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = DefaultDialTimeout
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &Forwarder{cfg: cfg}, nil
}

// Send formats the log entry per RFC 3164 and writes it over the
// configured transport. UDP is fire-and-forget; TCP reconnects on send
// failure once before returning the error.
func (f *Forwarder) Send(ctx context.Context, log *SystemLog) error {
	if log == nil {
		return errors.New("syslog forwarder: nil log")
	}
	wire := formatRFC3164(f.cfg, log, time.Now())
	switch f.cfg.Protocol {
	case ProtocolUDP:
		return f.sendUDP(ctx, wire)
	case ProtocolTCP:
		return f.sendTCP(ctx, wire)
	default:
		return fmt.Errorf("syslog forwarder: unsupported protocol %q", f.cfg.Protocol)
	}
}

// Close releases any held TCP connection. Safe to call multiple times;
// no-op for UDP forwarders.
func (f *Forwarder) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tcpConn != nil {
		err := f.tcpConn.Close()
		f.tcpConn = nil
		f.tcpAlive = false
		return err
	}
	return nil
}

// sendUDP dials, writes, and closes. Per-call dial keeps the function
// stateless — UDP has no connection semantics, so holding a "conn" across
// calls would only buy a tiny socket-creation saving while complicating
// error paths.
func (f *Forwarder) sendUDP(ctx context.Context, wire []byte) error {
	dialer := &net.Dialer{Timeout: f.cfg.DialTimeout}
	conn, err := dialer.DialContext(ctx, "udp", f.cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("dial udp %s: %w", f.cfg.Endpoint, err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	} else {
		_ = conn.SetWriteDeadline(time.Now().Add(f.cfg.DialTimeout))
	}
	if _, err := conn.Write(wire); err != nil {
		return fmt.Errorf("write udp: %w", err)
	}
	return nil
}

// sendTCP holds the connection across calls and reconnects lazily on
// send failure. One reconnect attempt per Send to avoid retry storms;
// repeated failures bubble up to the caller for circuit-breaker logic
// at a higher layer.
func (f *Forwarder) sendTCP(ctx context.Context, wire []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Append RFC 6587 octet-stream framing: each TCP message ends with \n.
	// Some collectors (rsyslog, syslog-ng) require the terminator; UDP
	// does not because each datagram is a single message.
	framed := append(wire, '\n')

	if f.tcpConn == nil || !f.tcpAlive {
		if err := f.dialTCPLocked(ctx); err != nil {
			return err
		}
	}

	if err := f.writeTCPLocked(ctx, framed); err != nil {
		// One reconnect attempt then retry.
		f.cfg.Logger.Warn("syslog tcp write failed, reconnecting", zap.Error(err))
		_ = f.tcpConn.Close()
		f.tcpConn = nil
		f.tcpAlive = false
		if err := f.dialTCPLocked(ctx); err != nil {
			return err
		}
		if err := f.writeTCPLocked(ctx, framed); err != nil {
			return fmt.Errorf("syslog tcp retry write: %w", err)
		}
	}
	return nil
}

// dialTCPLocked must be called with f.mu held.
func (f *Forwarder) dialTCPLocked(ctx context.Context) error {
	dialer := &net.Dialer{Timeout: f.cfg.DialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", f.cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("dial tcp %s: %w", f.cfg.Endpoint, err)
	}
	f.tcpConn = conn
	f.tcpAlive = true
	return nil
}

// writeTCPLocked must be called with f.mu held.
func (f *Forwarder) writeTCPLocked(ctx context.Context, framed []byte) error {
	if deadline, ok := ctx.Deadline(); ok {
		_ = f.tcpConn.SetWriteDeadline(deadline)
	} else {
		_ = f.tcpConn.SetWriteDeadline(time.Now().Add(f.cfg.DialTimeout))
	}
	_, err := f.tcpConn.Write(framed)
	return err
}

// formatRFC3164 builds the wire payload. Exported lowercase for direct
// testing without going through net I/O.
//
//	<priority>Mmm dd hh:mm:ss hostname tag: message
//
// SystemLog.Level → severity mapping per RFC 3164:
//
//	emerg/fatal → 0   error/err     → 3
//	alert       → 1   warning/warn  → 4
//	crit        → 2   notice        → 5
//	-           → 3   info          → 6
//	-           → -   debug         → 7
//
// Unknown levels fall back to severity 6 (info) so observability is
// never lost — a typo'd level still gets forwarded.
func formatRFC3164(cfg ForwarderConfig, log *SystemLog, now time.Time) []byte {
	severity := severityFromLevel(log.Level)
	priority := cfg.Facility*8 + severity

	// Replace newlines in the message so receivers don't get confused
	// by mid-payload framing breaks (RFC 3164 messages are line-oriented).
	msg := strings.ReplaceAll(log.Message, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", " ")

	// RFC 3164 timestamp: Mmm dd hh:mm:ss with single-digit days padded
	// by space (NOT zero). Go's reference layout "Jan _2 15:04:05" does
	// exactly that.
	ts := now.Format("Jan _2 15:04:05")

	var b strings.Builder
	b.WriteByte('<')
	fmt.Fprintf(&b, "%d", priority)
	b.WriteByte('>')
	b.WriteString(ts)
	b.WriteByte(' ')
	b.WriteString(cfg.Hostname)
	b.WriteByte(' ')
	b.WriteString(cfg.AppName)
	b.WriteString(": ")
	b.WriteString(msg)
	return []byte(b.String())
}

// severityFromLevel maps a SystemLog.Level string to an RFC 3164 severity
// integer. Case-insensitive; common aliases supported.
func severityFromLevel(level string) int {
	switch strings.ToLower(level) {
	case "emerg", "emergency", "fatal", "panic":
		return 0
	case "alert":
		return 1
	case "crit", "critical":
		return 2
	case "error", "err":
		return 3
	case "warning", "warn":
		return 4
	case "notice":
		return 5
	case "info":
		return 6
	case "debug":
		return 7
	default:
		return 6 // info — never drop unknown level
	}
}
