package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/pm/adhoc"
	"github.com/stretchr/testify/require"
)

func TestBuiltinReconcileIntervalIsFiveMinutes(t *testing.T) {
	require.Equal(t, 5*time.Minute, pmBuiltinReconcileInterval)
}

func TestRunBuiltinReconcileReloadsSnapshotAfterChangedVersion(t *testing.T) {
	reloads := 0
	result, err := runPMBuiltinReconcile(
		context.Background(),
		func(context.Context) (adhoc.BuiltinReconcileResult, error) {
			return adhoc.BuiltinReconcileResult{Definitions: 12, Saved: 8, Changed: 3, Empty: 4}, nil
		},
		func(context.Context) error {
			reloads++
			return nil
		},
	)

	require.NoError(t, err)
	require.Equal(t, 3, result.Changed)
	require.Equal(t, 1, reloads)
}

func TestRunBuiltinReconcileKeepsSnapshotWhenNothingChanged(t *testing.T) {
	reloads := 0
	result, err := runPMBuiltinReconcile(
		context.Background(),
		func(context.Context) (adhoc.BuiltinReconcileResult, error) {
			return adhoc.BuiltinReconcileResult{Definitions: 12, Saved: 8, Empty: 4}, nil
		},
		func(context.Context) error {
			reloads++
			return nil
		},
	)

	require.NoError(t, err)
	require.Zero(t, result.Changed)
	require.Zero(t, reloads)
}

func TestRunBuiltinReconcileDoesNotReloadAfterReconcileError(t *testing.T) {
	reloads := 0
	_, err := runPMBuiltinReconcile(
		context.Background(),
		func(context.Context) (adhoc.BuiltinReconcileResult, error) {
			return adhoc.BuiltinReconcileResult{Definitions: 12, Failed: 1}, errors.New("resolve failed")
		},
		func(context.Context) error {
			reloads++
			return nil
		},
	)

	require.ErrorContains(t, err, "resolve failed")
	require.Zero(t, reloads)
}

func TestInitialBuiltinReconcileRetriesMetadataStartupRace(t *testing.T) {
	attempts := 0
	result, err := reconcilePMBuiltinsUntilReady(
		context.Background(),
		func(context.Context) (adhoc.BuiltinReconcileResult, error) {
			attempts++
			if attempts < 3 {
				return adhoc.BuiltinReconcileResult{Definitions: 12, Failed: 12},
					errors.New("indicator metadata unavailable")
			}
			return adhoc.BuiltinReconcileResult{Definitions: 12, Saved: 12, Changed: 12}, nil
		},
		func(context.Context) error { return nil },
		time.Millisecond,
	)

	require.NoError(t, err)
	require.Equal(t, 3, attempts)
	require.Equal(t, 12, result.Saved)
}

func TestInitialBuiltinReconcileStopsAtContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	attempts := 0

	_, err := reconcilePMBuiltinsUntilReady(
		ctx,
		func(context.Context) (adhoc.BuiltinReconcileResult, error) {
			attempts++
			return adhoc.BuiltinReconcileResult{Definitions: 12, Failed: 12},
				errors.New("indicator metadata unavailable")
		},
		func(context.Context) error { return nil },
		time.Millisecond,
	)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Greater(t, attempts, 1)
}
