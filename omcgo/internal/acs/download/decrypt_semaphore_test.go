package download

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/backup"
)

// V1 — under limit returns immediately, no rejection metric.
func TestDecryptSemaphore_UnderLimit_NoBlock(t *testing.T) {
	sem := NewDecryptSemaphore(4, 1*time.Second, backup.NewPolicyMetrics(nil))
	require.NotNil(t, sem)

	for i := 0; i < 3; i++ {
		waited, err := sem.Acquire(context.Background())
		require.NoError(t, err)
		assert.Less(t, waited, 100*time.Millisecond, "under-limit acquire should be near-instant")
	}
	for i := 0; i < 3; i++ {
		sem.Release()
	}
}

// V2 — at-limit acquirer blocks until timeout, returns sentinel error.
func TestDecryptSemaphore_AtLimit_Timeout(t *testing.T) {
	sem := NewDecryptSemaphore(1, 50*time.Millisecond, backup.NewPolicyMetrics(nil))
	require.NotNil(t, sem)

	// First acquire holds the slot.
	_, err := sem.Acquire(context.Background())
	require.NoError(t, err)

	// Second acquire should time out (no Release called).
	start := time.Now()
	_, err = sem.Acquire(context.Background())
	elapsed := time.Since(start)
	require.ErrorIs(t, err, ErrDecryptSemaphoreTimeout)
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond, "timeout must respect configured duration")
	assert.Less(t, elapsed, 500*time.Millisecond, "timeout must not over-shoot dramatically")

	// Cleanup.
	sem.Release()
}

// V3 — Release unblocks a waiting acquirer.
func TestDecryptSemaphore_ReleaseUnblocksWaiter(t *testing.T) {
	sem := NewDecryptSemaphore(1, 2*time.Second, backup.NewPolicyMetrics(nil))
	_, err := sem.Acquire(context.Background())
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	var waiterErr error
	var waited time.Duration
	go func() {
		defer wg.Done()
		waited, waiterErr = sem.Acquire(context.Background())
	}()

	// Give the goroutine a moment to enter Acquire.
	time.Sleep(20 * time.Millisecond)
	sem.Release()
	wg.Wait()

	require.NoError(t, waiterErr)
	assert.GreaterOrEqual(t, waited, 20*time.Millisecond, "waiter should have observed wait time")
	sem.Release()
}

// V4 — ctx cancel does not leak the slot; subsequent acquirers can still proceed.
func TestDecryptSemaphore_CtxCancel_NoSlotLeak(t *testing.T) {
	sem := NewDecryptSemaphore(1, 5*time.Second, backup.NewPolicyMetrics(nil))
	_, err := sem.Acquire(context.Background())
	require.NoError(t, err)

	// Second acquire with ctx that cancels mid-wait.
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	var cancelErr error
	go func() {
		defer wg.Done()
		_, cancelErr = sem.Acquire(ctx)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	wg.Wait()
	require.ErrorIs(t, cancelErr, context.Canceled,
		"acquire must return ctx.Err on cancel, not slot leak")

	// Release the original slot — third acquirer should now be able to take it.
	sem.Release()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()
	_, err = sem.Acquire(ctx2)
	require.NoError(t, err, "post-cancel slot must be reusable")
	sem.Release()
}

// V5 — limit ≤ 0 returns nil semaphore; Acquire/Release are no-ops.
func TestDecryptSemaphore_DisabledByZeroLimit(t *testing.T) {
	for _, limit := range []int{0, -1, -100} {
		limit := limit
		t.Run("", func(t *testing.T) {
			sem := NewDecryptSemaphore(limit, 1*time.Second, backup.NewPolicyMetrics(nil))
			assert.Nil(t, sem, "limit=%d must return nil (disabled)", limit)
			// Methods on nil receiver are no-ops.
			waited, err := sem.Acquire(context.Background())
			require.NoError(t, err)
			assert.Equal(t, time.Duration(0), waited)
			sem.Release() // must not panic
		})
	}
}

// V6 — concurrent acquirers all eventually succeed when releases happen fast enough.
func TestDecryptSemaphore_ConcurrentFairness(t *testing.T) {
	sem := NewDecryptSemaphore(2, 5*time.Second, backup.NewPolicyMetrics(nil))
	const N = 10
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, err := sem.Acquire(context.Background())
			require.NoError(t, err)
			// Brief work simulation; release promptly so others progress.
			time.Sleep(5 * time.Millisecond)
			sem.Release()
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		// All 10 acquirers completed despite limit=2.
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent acquirers deadlocked — limit=2 should serialise but not starve")
	}
}

// V7 — env var hard cap clamping. NewDecryptSemaphore clamps over-cap to 64.
func TestDecryptSemaphore_HardCapClamp(t *testing.T) {
	sem := NewDecryptSemaphore(999, 1*time.Second, backup.NewPolicyMetrics(nil))
	require.NotNil(t, sem)
	assert.Equal(t, DecryptSemaphoreHardCap, sem.ClampedLimit(),
		"ClampedLimit must report effective slot count after clamp")
	// Review LOW-2 (T-0089): release acquired slots so a future shared-
	// state refactor doesn't introduce flakes.
	t.Cleanup(func() {
		for i := 0; i < DecryptSemaphoreHardCap; i++ {
			sem.Release()
		}
	})
	// Verify capacity by acquiring up to the hard cap without blocking.
	for i := 0; i < DecryptSemaphoreHardCap; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_, err := sem.Acquire(ctx)
		cancel()
		require.NoError(t, err, "acquire %d/%d under clamped limit must not block", i+1, DecryptSemaphoreHardCap)
	}
	// Next acquire must time out — proving cap is exactly DecryptSemaphoreHardCap.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := sem.Acquire(ctx)
	require.Error(t, err, "acquire #%d must hit cap", DecryptSemaphoreHardCap+1)
}

// V8 — nil metrics (production wiring sometimes passes nil for tests) must
// not crash. Verifies all metric Record calls are nil-safe.
func TestDecryptSemaphore_NilMetricsSafe(t *testing.T) {
	sem := NewDecryptSemaphore(1, 50*time.Millisecond, nil)
	require.NotNil(t, sem)

	// Successful acquire+release path.
	_, err := sem.Acquire(context.Background())
	require.NoError(t, err)
	sem.Release()

	// Timeout path (record rejected without panic).
	_, err = sem.Acquire(context.Background())
	require.NoError(t, err)
	_, err = sem.Acquire(context.Background())
	require.ErrorIs(t, err, ErrDecryptSemaphoreTimeout)
	sem.Release()
}

// V9 — exact wait duration is recorded back to caller.
func TestDecryptSemaphore_AcquireReturnsWaitDuration(t *testing.T) {
	sem := NewDecryptSemaphore(1, 500*time.Millisecond, backup.NewPolicyMetrics(nil))
	_, err := sem.Acquire(context.Background())
	require.NoError(t, err)

	// Release after a deterministic delay; second acquirer waits ~delay then succeeds.
	const delay = 80 * time.Millisecond
	go func() {
		time.Sleep(delay)
		sem.Release()
	}()

	waited, err := sem.Acquire(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, waited, delay, "Acquire must report at least the actual wait time")
	assert.Less(t, waited, 5*delay, "Acquire wait must not be wildly inflated")
	sem.Release()
}

// V10 — ErrDecryptSemaphoreTimeout sentinel is exported and errors.Is-comparable.
func TestDecryptSemaphore_ErrSentinelIsComparable(t *testing.T) {
	wrapped := errors.New("wrapper: " + ErrDecryptSemaphoreTimeout.Error())
	assert.False(t, errors.Is(wrapped, ErrDecryptSemaphoreTimeout),
		"non-wrapped error must not match sentinel")

	// Direct match
	assert.True(t, errors.Is(ErrDecryptSemaphoreTimeout, ErrDecryptSemaphoreTimeout))
	// errors.Join wrap
	joinWrapped := errors.Join(errors.New("upstream"), ErrDecryptSemaphoreTimeout)
	assert.True(t, errors.Is(joinWrapped, ErrDecryptSemaphoreTimeout),
		"errors.Join sentinel must match via errors.Is")
	// Review LOW-3 (T-0089): production code wraps with fmt.Errorf %w
	// (handler.go ServeHTTP error switch). Test that pattern explicitly
	// so a future refactor that changes wrap style is caught here.
	fmtWrapped := errorsWrapWithFmt()
	assert.True(t, errors.Is(fmtWrapped, ErrDecryptSemaphoreTimeout),
		"fmt.Errorf %%w wrapped sentinel must match via errors.Is")
}

// errorsWrapWithFmt emulates handler.go's `fmt.Errorf("...: %w", err)`
// production wrap pattern.
func errorsWrapWithFmt() error {
	return fmt.Errorf("decryption failed during ServeHTTP error mapping: %w",
		ErrDecryptSemaphoreTimeout)
}
