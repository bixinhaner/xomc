package stream

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSaveEffectiveFromDefaultsToNextSlot(t *testing.T) {
	now := time.Date(2026, 7, 27, 6, 4, 30, 0, time.UTC)

	got := saveEffectiveFrom(SaveTaskRequest{}, now)

	require.True(t, got.Equal(time.Date(2026, 7, 27, 6, 15, 0, 0, time.UTC)))
}

func TestSaveEffectiveFromUsesExplicitTime(t *testing.T) {
	now := time.Date(2026, 7, 27, 6, 4, 30, 0, time.UTC)
	explicit := time.Date(2026, 7, 27, 6, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	got := saveEffectiveFrom(SaveTaskRequest{EffectiveFrom: explicit}, now)

	require.True(t, got.Equal(time.Date(2026, 7, 26, 22, 0, 0, 0, time.UTC)))
	require.Equal(t, time.UTC, got.Location())
}

func TestShouldAdjustEffectiveFromBackdatesInitialUnchangedVersion(t *testing.T) {
	req := SaveTaskRequest{EffectiveFrom: time.Date(2026, 7, 27, 5, 0, 0, 0, time.UTC)}
	current := time.Date(2026, 7, 27, 5, 15, 0, 0, time.UTC)
	target := time.Date(2026, 7, 27, 5, 0, 0, 0, time.UTC)

	require.True(t, shouldAdjustEffectiveFrom(req, 1, current, target))
	require.False(t, shouldAdjustEffectiveFrom(SaveTaskRequest{}, 1, current, target))
	require.False(t, shouldAdjustEffectiveFrom(req, 2, current, target))
	require.False(t, shouldAdjustEffectiveFrom(req, 1, target, target))
	require.False(t, shouldAdjustEffectiveFrom(req, 1, target.Add(-slotDuration), target))
}
