// Package download — backup decrypt concurrency semaphore (T-0089).
//
// T-0075 introduced AES-256-GCM envelope encryption on backup downloads.
// GCM is single-pass authenticated, so maybeDecrypt MUST buffer the entire
// ciphertext (capped at 64 MB + 1 KB) before authenticating. With N
// concurrent download requests, peak memory pressure is 64 MB × N — a
// resource-exhaustion vector flagged in the T-0075 review (M-4) and
// deferred to this task.
//
// DecryptSemaphore caps concurrent decrypt slot allocations so the worst
// case is bounded to (limit × 64 MB). Exceeding requests wait for up to a
// configurable timeout, then receive a sentinel error the handler maps to
// HTTP 503 Service Unavailable + Retry-After.
//
// Design choices (per PRD T-0089 §2):
//   - Buffered channel rather than x/sync/semaphore: KISS, no new dep.
//   - timeout-then-reject rather than pure block: prevents pile-ups.
//   - nil receiver acts as disabled: preserves T-0075 behaviour when the
//     semaphore is not wired (tests, or legacy deployments tuning the env
//     var to 0).
//   - Hard-cap clamp at 64 to defend against operator misconfiguration.
package download

import (
	"context"
	"errors"
	"time"

	"github.com/omcgo/omcgo/internal/backup"
)

// ErrDecryptSemaphoreTimeout is returned by Acquire when the acquire wait
// exceeded the configured timeout. Callers (handler.maybeDecrypt) map this
// to HTTP 503 + Retry-After. Distinct sentinel so the handler's error
// switch can route it differently from ctx.Err().
var ErrDecryptSemaphoreTimeout = errors.New("decrypt semaphore acquire timeout")

// DecryptSemaphoreHardCap is the maximum slot count NewDecryptSemaphore
// accepts. Operator misconfiguration (`OMC_BACKUP_DECRYPT_CONCURRENCY=999`)
// is silently clamped — better than panicking the ACS process at startup.
// 64 × 64 MB = 4 GB peak buffer; beyond this, OOM is more likely than
// throughput improvement.
const DecryptSemaphoreHardCap = 64

// DecryptSemaphore caps concurrent backup-decrypt buffer allocations.
// Constructed by NewDecryptSemaphore and wired via Handler.SetDecryptSemaphore.
//
// Zero-value usage: nil receiver. Acquire/Release on nil are no-ops, so
// disabled deployments preserve T-0075 unbounded behaviour.
type DecryptSemaphore struct {
	slots   chan struct{}
	timeout time.Duration
	metrics *backup.PolicyMetrics
}

// NewDecryptSemaphore returns a semaphore with `limit` slots and per-acquire
// `timeout`. Returns nil when limit ≤ 0 (disabled — preserves T-0075).
// Limit > DecryptSemaphoreHardCap is silently clamped; the caller is
// expected to emit a structured warn log at startup so misconfiguration
// surfaces operationally (cmd/acs/main.go).
//
// Review MED-2 (T-0089): the previous oversize_config metric label
// conflated startup-time clamping with per-acquire rejection — Grafana
// alerts on rejection rate were misled by a one-shot pulse at boot.
// Clamping is now silent at the constructor; visibility moves to the
// caller's startup log.
func NewDecryptSemaphore(limit int, timeout time.Duration, m *backup.PolicyMetrics) *DecryptSemaphore {
	if limit <= 0 {
		return nil
	}
	clamped := limit
	if clamped > DecryptSemaphoreHardCap {
		clamped = DecryptSemaphoreHardCap
	}
	return &DecryptSemaphore{
		slots:   make(chan struct{}, clamped),
		timeout: timeout,
		metrics: m,
	}
}

// ClampedLimit reports the actual slot count after applying
// DecryptSemaphoreHardCap. Useful for caller-side startup logging that
// wants to compare requested vs. effective limit.
func (s *DecryptSemaphore) ClampedLimit() int {
	if s == nil {
		return 0
	}
	return cap(s.slots)
}

// Acquire blocks until a slot is available, ctx is cancelled, or the
// configured timeout elapses (whichever comes first). Returns the wait
// duration plus an error: nil on success; ErrDecryptSemaphoreTimeout when
// the timer fires; ctx.Err() (Canceled / DeadlineExceeded) on ctx cancel.
//
// Caller MUST call Release exactly once when Acquire returns nil. Do NOT
// call Release on failure paths.
//
// Nil receiver returns (0, nil) immediately — disabled mode.
func (s *DecryptSemaphore) Acquire(ctx context.Context) (time.Duration, error) {
	if s == nil {
		return 0, nil
	}
	start := time.Now()
	timer := time.NewTimer(s.timeout)
	defer timer.Stop()
	select {
	case s.slots <- struct{}{}:
		waited := time.Since(start)
		s.metrics.ObserveBackupDecryptWait(waited.Seconds())
		s.metrics.SetBackupDecryptInFlight(len(s.slots))
		return waited, nil
	case <-ctx.Done():
		s.metrics.RecordBackupDecryptRejected("ctx_cancel")
		return time.Since(start), ctx.Err()
	case <-timer.C:
		s.metrics.RecordBackupDecryptRejected("timeout")
		return time.Since(start), ErrDecryptSemaphoreTimeout
	}
}

// Release returns the slot. Must be paired with a successful Acquire.
// Calling on a nil receiver is a no-op (disabled mode).
//
// This method does not return an error: the channel receive cannot fail
// once a slot was successfully sent. Double-release would block (since
// the channel is empty), but the defer pattern in maybeDecrypt makes that
// impossible.
func (s *DecryptSemaphore) Release() {
	if s == nil {
		return
	}
	<-s.slots
	s.metrics.SetBackupDecryptInFlight(len(s.slots))
}
