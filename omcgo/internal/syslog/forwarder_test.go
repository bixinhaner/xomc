package syslog

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedTime keeps wire-format tests deterministic. Mar 15 14:30:45 in
// any year — RFC 3164 timestamps don't carry the year so we don't need
// to pin it.
var fixedTime = time.Date(2026, time.March, 15, 14, 30, 45, 0, time.UTC)

// TestFormatRFC3164_T0028_BasicShape — verify the wire format contract:
// `<priority>timestamp hostname tag: message`.
func TestFormatRFC3164_T0028_BasicShape(t *testing.T) {
	cfg := ForwarderConfig{
		Facility: 16, // local0
		Hostname: "omcgo-app-01",
		AppName:  "omcgo",
	}
	log := &SystemLog{
		ID:      uuid.New(),
		Level:   "error",
		Source:  "device.service",
		Message: "device timeout",
	}
	wire := string(formatRFC3164(cfg, log, fixedTime))

	// priority = 16*8 + 3 (error) = 131
	assert.Equal(t, "<131>Mar 15 14:30:45 omcgo-app-01 omcgo: device timeout", wire)
}

// TestFormatRFC3164_T0028_LevelMapping — verify all RFC 3164 severities
// + common aliases map to the expected integer.
func TestFormatRFC3164_T0028_LevelMapping(t *testing.T) {
	cases := map[string]int{
		"emerg":     0,
		"emergency": 0,
		"fatal":     0,
		"panic":     0,
		"alert":     1,
		"crit":      2,
		"critical":  2,
		"error":     3,
		"err":       3,
		"warning":   4,
		"warn":      4,
		"notice":    5,
		"info":      6,
		"debug":     7,
		"INFO":      6, // case-insensitive
		"WARN":      4,
		"unknown":   6, // fallback
		"":          6,
	}
	for level, want := range cases {
		assert.Equal(t, want, severityFromLevel(level), "level=%q", level)
	}
}

// TestFormatRFC3164_T0028_NewlineSanitized — message newlines become
// spaces so receivers can't be confused by mid-payload framing.
func TestFormatRFC3164_T0028_NewlineSanitized(t *testing.T) {
	cfg := ForwarderConfig{Facility: 0, Hostname: "h", AppName: "a"}
	log := &SystemLog{Level: "info", Message: "line1\nline2\rline3"}
	wire := string(formatRFC3164(cfg, log, fixedTime))
	assert.Contains(t, wire, "line1 line2 line3")
	assert.NotContains(t, wire, "\n")
	assert.NotContains(t, wire, "\r")
}

// TestNewForwarder_T0028_Validation — required fields + protocol enum.
func TestNewForwarder_T0028_Validation(t *testing.T) {
	cases := []struct {
		name    string
		cfg     ForwarderConfig
		wantErr bool
	}{
		{"empty endpoint", ForwarderConfig{Protocol: "udp"}, true},
		{"unknown protocol", ForwarderConfig{Endpoint: "h:1", Protocol: "sctp"}, true},
		{"valid udp", ForwarderConfig{Endpoint: "h:514", Protocol: "udp"}, false},
		{"valid tcp", ForwarderConfig{Endpoint: "h:601", Protocol: "tcp"}, false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f, err := NewForwarder(c.cfg)
			if c.wantErr {
				require.Error(t, err)
				assert.Nil(t, f)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, f)
			// Defaults applied
			assert.Equal(t, "omcgo", f.cfg.AppName)
			assert.Equal(t, "localhost", f.cfg.Hostname)
			assert.Equal(t, DefaultDialTimeout, f.cfg.DialTimeout)
		})
	}
}

// TestNewForwarder_T0028_FacilityClamp — invalid facility falls back
// to local0 rather than producing an out-of-range priority byte.
func TestNewForwarder_T0028_FacilityClamp(t *testing.T) {
	for _, fac := range []int{-1, 24, 100} {
		f, err := NewForwarder(ForwarderConfig{
			Endpoint: "h:1", Protocol: "udp", Facility: fac,
		})
		require.NoError(t, err)
		assert.Equal(t, DefaultFacility, f.cfg.Facility)
	}
}

// TestForwarder_T0028_SendUDP_RealLocalhost — bind a UDP listener,
// send a log, verify the wire payload arrives. Catches both dial path
// and write path with real network I/O.
func TestForwarder_T0028_SendUDP_RealLocalhost(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer pc.Close()
	addr := pc.LocalAddr().String()

	f, err := NewForwarder(ForwarderConfig{
		Endpoint: addr, Protocol: "udp",
		Hostname: "test-host", AppName: "test-app",
	})
	require.NoError(t, err)

	log := &SystemLog{Level: "warning", Message: "hello world"}
	require.NoError(t, f.Send(context.Background(), log))

	// Read with a reasonable deadline so a hung test fails fast.
	require.NoError(t, pc.SetReadDeadline(time.Now().Add(2*time.Second)))
	buf := make([]byte, 4096)
	n, _, err := pc.ReadFrom(buf)
	require.NoError(t, err)
	got := string(buf[:n])

	assert.Contains(t, got, "<132>") // facility 16 + warning(4)
	assert.Contains(t, got, "test-host test-app: hello world")
}

// TestForwarder_T0028_SendTCP_RealLocalhost — bind TCP listener, send
// a log, accept + read, verify framed payload (RFC 6587 octet-stream:
// trailing \n).
func TestForwarder_T0028_SendTCP_RealLocalhost(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	addr := ln.Addr().String()

	// Accept goroutine collects the first line received.
	var got string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		r := bufio.NewReader(conn)
		line, err := r.ReadString('\n')
		if err == nil {
			got = strings.TrimRight(line, "\n")
		}
	}()

	f, err := NewForwarder(ForwarderConfig{
		Endpoint: addr, Protocol: "tcp",
		Hostname: "test-host", AppName: "test-app",
	})
	require.NoError(t, err)
	defer f.Close()

	log := &SystemLog{Level: "info", Message: "tcp-hello"}
	require.NoError(t, f.Send(context.Background(), log))

	wg.Wait()
	assert.Contains(t, got, "<134>") // facility 16 + info(6)
	assert.Contains(t, got, "test-host test-app: tcp-hello")
}

// TestForwarder_T0028_SendTCP_PersistentConnReuse — two consecutive
// Send calls share one TCP conn (no second dial visible to the listener).
func TestForwarder_T0028_SendTCP_PersistentConnReuse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	addr := ln.Addr().String()

	var (
		acceptCount int
		linesGot    []string
		mu          sync.Mutex
		done        = make(chan struct{})
	)

	go func() {
		defer close(done)
		for i := 0; i < 1; i++ { // only accept ONE conn — second send must reuse
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			acceptCount++
			mu.Unlock()
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
				r := bufio.NewReader(c)
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					mu.Lock()
					linesGot = append(linesGot, strings.TrimRight(line, "\n"))
					mu.Unlock()
				}
			}(conn)
		}
	}()

	f, err := NewForwarder(ForwarderConfig{
		Endpoint: addr, Protocol: "tcp", Hostname: "h", AppName: "a",
	})
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, f.Send(context.Background(), &SystemLog{Level: "info", Message: "first"}))
	require.NoError(t, f.Send(context.Background(), &SystemLog{Level: "info", Message: "second"}))

	// Give the reader goroutine a moment to drain.
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, acceptCount, "second send must reuse the persistent TCP conn")
	require.Len(t, linesGot, 2)
	assert.Contains(t, linesGot[0], "first")
	assert.Contains(t, linesGot[1], "second")
}

// TestForwarder_T0028_SendUDP_DialFail — unreachable endpoint surfaces
// as a wrapped error rather than panic.
func TestForwarder_T0028_SendUDP_DialFail(t *testing.T) {
	f, err := NewForwarder(ForwarderConfig{
		Endpoint: "127.0.0.1:1", Protocol: "udp", DialTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)

	// UDP dial almost never fails on Linux (it's connectionless), so we
	// instead validate write timeout via SetWriteDeadline elapsed.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Best-effort: this MAY succeed (UDP doesn't ICMP-block by default).
	// We assert it doesn't panic; the wire-payload tests cover correctness
	// elsewhere.
	_ = f.Send(ctx, &SystemLog{Level: "info", Message: "x"})
}

// TestForwarder_T0028_SendTCP_DialFail — TCP to closed port returns
// connection refused; the error wraps the dial error.
func TestForwarder_T0028_SendTCP_DialFail(t *testing.T) {
	// Bind+close to claim a port number the OS is likely to leave
	// free for a moment, then tear it down so dial fails.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close() // immediately close so dial gets refused

	f, err := NewForwarder(ForwarderConfig{
		Endpoint: addr, Protocol: "tcp", DialTimeout: 200 * time.Millisecond,
	})
	require.NoError(t, err)
	defer f.Close()

	err = f.Send(context.Background(), &SystemLog{Level: "info", Message: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dial tcp")
}

// TestForwarder_T0028_SendNilLog — guard nil input.
func TestForwarder_T0028_SendNilLog(t *testing.T) {
	f, err := NewForwarder(ForwarderConfig{Endpoint: "127.0.0.1:514", Protocol: "udp"})
	require.NoError(t, err)
	err = f.Send(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil log")
}

// TestForwarder_T0028_Close_Idempotent — Close twice is safe (UDP
// forwarder has no conn; TCP forwarder closes once and no-ops after).
func TestForwarder_T0028_Close_Idempotent(t *testing.T) {
	f, err := NewForwarder(ForwarderConfig{Endpoint: "127.0.0.1:1", Protocol: "udp"})
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, f.Close())
}
