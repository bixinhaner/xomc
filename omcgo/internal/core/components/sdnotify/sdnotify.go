// Package sdnotify implements the minimal subset of the systemd sd_notify(3)
// protocol the OMC services need — with no external dependency.
//
// Under systemd `Type=notify` it lets a process:
//   - announce startup completion (READY=1),
//   - periodically pet the watchdog (WATCHDOG=1) so systemd restarts a hung
//     process even though it has not exited,
//   - announce graceful shutdown start (STOPPING=1).
//
// When not run under systemd (dev `go run`, bare exec, tests) the relevant
// environment variables are absent and every exported call is a cheap no-op,
// so the same binary runs unchanged in every deployment mode.
package sdnotify

import (
	"context"
	"net"
	"os"
	"strconv"
	"time"
)

// notify sends one sd_notify datagram to $NOTIFY_SOCKET.
//
// sd_notify is best-effort by design: a missed notification must never break
// the service, so all errors are swallowed and reported only as a false
// return (useful for tests and callers that want to know whether systemd is
// listening).
func notify(state string) bool {
	socket := os.Getenv("NOTIFY_SOCKET")
	if socket == "" {
		return false
	}
	conn, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		return false
	}
	defer conn.Close()
	_, err = conn.Write([]byte(state))
	return err == nil
}

// Ready tells systemd that startup finished. Required under `Type=notify` —
// without it systemd keeps the unit in "activating" until TimeoutStartSec.
func Ready() bool { return notify("READY=1") }

// Stopping tells systemd that graceful shutdown has begun. systemd then stops
// expecting watchdog pings and applies TimeoutStopSec to the stop phase.
func Stopping() bool { return notify("STOPPING=1") }

// WatchdogInterval reports the cadence at which the watchdog must be pet and
// whether systemd enabled the watchdog for THIS process.
//
// systemd exports WATCHDOG_USEC (the WatchdogSec timeout, in microseconds)
// and — since v229 — WATCHDOG_PID. The recommended pet cadence is half the
// timeout, leaving a full interval of slack before systemd declares a miss.
func WatchdogInterval() (time.Duration, bool) {
	usec := os.Getenv("WATCHDOG_USEC")
	if usec == "" {
		return 0, false
	}
	// WATCHDOG_PID, when set, guards against the variables being inherited by
	// a child process that is not the one systemd is watching.
	if pid := os.Getenv("WATCHDOG_PID"); pid != "" {
		if p, err := strconv.Atoi(pid); err != nil || p != os.Getpid() {
			return 0, false
		}
	}
	us, err := strconv.Atoi(usec)
	if err != nil || us <= 0 {
		return 0, false
	}
	return time.Duration(us) * time.Microsecond / 2, true
}

// StartWatchdog launches a goroutine that pets the systemd watchdog every
// WatchdogInterval until ctx is cancelled. It is a no-op when the watchdog is
// not enabled for this process (dev runs, tests, units without WatchdogSec).
//
// The pet is a plain periodic ping: it proves the Go runtime is still
// scheduling goroutines. A fully wedged process (runtime deadlock, every OS
// thread blocked in cgo/syscalls) can no longer run this goroutine, the pings
// stop, and systemd restarts the unit — a failure mode `Restart=on-failure`
// cannot catch because the process never exits. Dependency-level stalls (a
// hung DB call) are intentionally NOT gated here, to avoid restart storms on
// transient blips; those are surfaced by the /readyz probe and Prometheus
// alerting instead.
func StartWatchdog(ctx context.Context) {
	interval, ok := WatchdogInterval()
	if !ok {
		return
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		notify("WATCHDOG=1") // pet immediately — don't wait a full interval
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				notify("WATCHDOG=1")
			}
		}
	}()
}
