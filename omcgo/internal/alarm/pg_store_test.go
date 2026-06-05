package alarm

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestHistoryUpdatedAtPrefersLastUpdatedAt(t *testing.T) {
	updatedAt := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Second)
	lastUpdatedAt := time.Now().Add(-2 * time.Minute).UTC().Truncate(time.Second)

	got := historyUpdatedAt(&model.Alarm{
		UpdatedAt:     updatedAt,
		LastUpdatedAt: lastUpdatedAt,
	})

	assert.Equal(t, lastUpdatedAt, got)
}

func TestHistoryUpdatedAtFallsBackToUpdatedAt(t *testing.T) {
	updatedAt := time.Now().Add(-3 * time.Minute).UTC().Truncate(time.Second)

	got := historyUpdatedAt(&model.Alarm{UpdatedAt: updatedAt})

	assert.Equal(t, updatedAt, got)
}

func TestHistoryUpdatedAtReturnsZeroWhenNoBusinessTimestampExists(t *testing.T) {
	got := historyUpdatedAt(&model.Alarm{})

	assert.True(t, got.IsZero())
}